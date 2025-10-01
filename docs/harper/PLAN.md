# PLAN — CoffeeBuddy (On-Prem)

Status: implementation plan for the next phase. Scope and constraints strictly follow IDEA.md and SPEC.md (Java17, Kubernetes, NGINX ingress + WSO2 internal gateway, Postgres HA, Kafka (3 partitions baseline), Keycloak OIDC, Vault, Prometheus/Grafana, Jenkins CI). Acceptance Criteria from SPEC are included (mandatory).

## Overview (goal)
Deliver a production-ready on‑prem CoffeeBuddy system that:
- Accepts Slack events via internal ingress and buffers them in Kafka.
- Implements event-driven Run lifecycle: create run, place orders, assign runner, schedule reminders.
- Persists domain state in Postgres and stores secrets in Vault.
- Exposes Admin REST protected by Keycloak and emits Prometheus metrics.
- Deploys via Helm and Jenkins to enterprise Kubernetes.

Planned cadence: 6 sprints, 2 weeks each (12 weeks). Adjust duration by team velocity. Sprint numbering and work items below are minimal-testable units.

---

## Sprint plan (work items, owners, acceptance for each)

Sprint 0 — Platform & Security Baseline (platform, devops) — 2w
- Tasks
  - Provision dev namespaces on K8s; install NGINX ingress + WSO2 APIM test instance (internal-only).
  - Deploy test Keycloak realm & client for OIDC; create test users/roles.
  - Deploy Vault dev cluster or dev instance, enable secret engine for application secrets.
  - Provision Postgres (dev HA shim or single-node dev) and Kafka (3-partition dev cluster).
  - Jenkins baseline job and credentials store integration with Vault.
- Acceptance
  - Keycloak reachable internally; test OIDC token issued.
  - Vault reachable; sample secret written and read by CI.
  - Postgres and Kafka reachable from cluster pods.
- Owner: Platform / DevOps

Sprint 1 — Slack Ingress Adapter + Kafka pipeline (backend, infra) — 2w
- Tasks
  - Implement Slack Ingress Adapter service (HTTP) that:
    - Verifies Slack request signature.
    - Validates payloads and produces messages to kafka topic slack.inbound.events.
    - Responds per Slack protocol (ack).
  - Containerize and deploy adapter with Helm chart entries.
  - Integrate adapter with WSO2 internal gateway and NGINX ingress (internal routing).
- Acceptance
  - Simulated Slack POST to ingress results in a message on kafka topic slack.inbound.events.
  - Adapter validates signatures and responds within Slack protocol timing.
- Owner: Backend, Platform

Sprint 2 — Event Processor / Run Service core + DB schema (backend, qa) — 2w
- Tasks
  - Implement Run Service that consumes slack.inbound.events and:
    - Normalizes events to runs.commands topic (create_run, place_order).
    - Persists core tables: users, runs, orders, assignments_history, audit_events (schema migration scripts).
    - Emits runs.events domain events.
  - Add lightweight scheduler capability for deadlines (in-process cron for dev).
  - Unit + integration tests for DB operations.
- Acceptance
  - POST /coffee simulated event -> run row is persisted in Postgres and run_created event appears on runs.events.
  - DB schema migrations runnable via CI.
- Owner: Backend

Sprint 3 — Orders, Preferences, Notification Service (backend, qa) — 2w
- Tasks
  - Implement orders flow: create/update orders linked to run and users; preference defaults applied and persisted.
  - Implement Notification Service that consumes runs.events/notifications.outbound and posts to Slack Web API via enterprise proxy.
  - Update Slack message summaries via block updates (use batching where possible).
  - Implement orders update summary message behaviour.
- Acceptance
  - Placing an order results in an orders row; if preferences exist, defaults applied.
  - Notification service posts/updates messages to Slack channel via proxy; end-to-end order confirmation ≤ 2 minutes under normal dev load.
- Owner: Backend, QA

Sprint 4 — Runner Assignment Service + Scheduler reminders (backend, qa) — 2w
- Tasks
  - Implement deterministic Runner Assignment Service (library/module) that:
    - Excludes opt-outs, prefers least-recently-assigned, avoids consecutive assignment where possible.
    - Records decisions in assignments_history with rationale.
  - Integrate assignment triggering at ordering_deadline.
  - Implement scheduled reminders (configurable cadence) and Notification calls.
- Acceptance
  - Assignment decision stored in DB and published; for consecutive runs with ≥3 eligible users, algorithm avoids assigning same user twice in a row.
  - Scheduled reminders emitted and posted (test via short deadline).
- Owner: Backend

Sprint 5 — Admin API, Observability, Security hardening (backend, platform, qa) — 2w
- Tasks
  - Implement Admin REST API protected by Keycloak OIDC (scoped roles), with endpoints listed in SPEC (/api/v1/runs, /api/v1/users/{id}/preferences).
  - Emit Prometheus metrics per SPEC and expose metrics endpoint.
  - Add structured logging with correlation_id per run; configure Grafana dashboards skeleton.
  - Harden service-to-service auth (mTLS or signed tokens) and Vault integration (secrets mount via CSI or runtime fetch).
- Acceptance
  - Admin endpoints require valid Keycloak tokens; metrics visible in Prometheus; sample Grafana dashboards populated.
  - Secrets not baked into images; retrieved from Vault at runtime.
- Owner: Backend, Platform

Sprint 6 — CI/CD, Helm charts, integration tests, resilience & deploy (devops, qa) — 2w
- Tasks
  - Finalize Helm chart skeleton for all components (see Helm section).
  - Jenkins pipelines: build, unit tests, image push, helm deploy to dev, run integration acceptance test suite, promote artifacts.
  - Implement load/burst resilience tests (Kafka buffering verification).
  - Prepare runbooks and rollback plan.
- Acceptance
  - Jenkins pipeline deploys to dev namespace; all pods reach Ready; integration acceptance tests pass.
  - Under simulated burst, all valid inbound events get persisted to Kafka and processed (N in -> N processed).
- Owner: DevOps, QA

---

## Definition of Done (per component)
- Code in main branch with passing unit tests and code review.
- Container image produced and scanned; no secrets in image.
- Helm chart templates present with values documented.
- Deployment to dev namespace completes with readiness probes passing.
- Acceptance tests (automated) pass.
- Docs: runbook + API spec + DB migration scripts checked in.

---

## Acceptance Criteria (mandatory — copied verbatim from SPEC)
1. Slack integration
   - When a simulated Slack /coffee command is POSTed to the ingress, the system creates a run, persists it in Postgres, and posts a confirmation message to the Slack channel. End-to-end time from first request to confirmation message ≤ 2 minutes under normal load.
2. Order & preferences
   - A placed order (via Slack simulated event) results in an orders row linked to the run and the user. If the user has stored preferences, the order uses those defaults; preference updates persist and apply to subsequent orders.
3. Runner assignment fairness
   - For consecutive runs with at least three eligible users, the assignment algorithm does not assign the same user twice in a row (unless only one eligible), and stores assignment reasoning in assignments_history. Assignment decision verifiable from DB and domain events.
4. Reminders & scheduling
   - Scheduled reminders are emitted and posted to Slack per configured schedule (test by setting a short deadline and verifying reminder messages).
5. Resilience to bursts
   - Under a simulated burst of inbound Slack events, no events are lost: all valid inbound events are written to Kafka and eventually processed (test by sending N events and asserting N runs/orders processed).
6. Security & network
   - Admin REST endpoints require valid Keycloak OIDC tokens; secrets are not present in code or container images and are retrieved from Vault at runtime. All external traffic to Slack is routed via enterprise proxy/gateway.
7. Observability
   - Prometheus metrics listed above are emitted and visible in Grafana. Alerts trigger when consumer lag exceeds configured threshold or when slack_api_errors exceed threshold in a given period.
8. Deployability
   - Helm chart and Jenkins pipeline produce a repeatable deployment to on-prem K8s; after deployment, all required pods reach ready state and services are reachable internally.

---

## Acceptance-test automation plan (QA)
Primary goal: automate the SPEC acceptance criteria.

Test harness components
- Slack Event Simulator
  - Small HTTP client service or test script that posts signed Slack-like payloads to the Ingress. Configurable to simulate slash commands and interactive events.
- Kafka injector/validator
  - Use kafkacat or a lightweight Java test harness to assert messages on topics (slack.inbound.events, runs.events).
- DB validator
  - SQL scripts or test code that queries Postgres to assert rows in runs, orders, assignments_history, audit_events.
- Notification verifier
  - Mock Slack Web API (dev proxy) that records outgoing webhook calls for assertions (to avoid external Slack).
- Prometheus assertions
  - Integration tests query Prometheus for metric presence and thresholds (e.g., consumer lag, slack_api_errors).
- Keycloak token test
  - Automated test obtains OIDC token from dev Keycloak and asserts Admin REST rejects/accepts appropriately.
- Vault secret test
  - Test ensures service fails when secret missing and succeeds when secret available; asserts no secrets in container images.

Test scenarios (automated)
- End-to-end /coffee flow: Simulator -> Ingress -> Kafka -> Run Service -> Postgres -> Notification recorded. Assert time ≤ 2 minutes.
- Orders with preferences: Create user pref, post order, validate orders row uses defaults.
- Assignment fairness: Create 3 eligible users, create 3 runs, assert no same user twice (per acceptance).
- Reminders: Configure short deadline, assert reminder messages posted.
- Burst test: Send N events concurrently, assert N kafka writes and N orders/runs processed eventually; measure consumer lag.
- Security tests: Call Admin API without token (fail), with token (pass).
- Observability tests: Assert key metrics emitted and alerts fire on simulated faults.

CI integration
- Run fast unit tests and static analysis on PRs.
- On successful merge, Jenkins deploys to dev K8s and runs the automated suite.
- Failure blocks promotion.

---

## Helm chart skeleton (artifact list & values)
Charts:
- coffee-buddy/
  - Chart.yaml
  - values.yaml (global options)
  - templates/
    - slack-ingress-adapter-deployment.yaml
    - slack-ingress-adapter-service.yaml
    - run-service-deployment.yaml
    - notification-service-deployment.yaml
    - admin-api-deployment.yaml
    - configmap-templates.yaml (logging, prometheus annotations)
    - secret-empty.yaml (placeholder – secrets injected via Vault CSI)
    - serviceaccount.yaml
    - ingress.yaml (internal-only annotations for WSO2)
    - hpa.yaml (optional)
    - _helpers.tpl
  - templates/grafana-dashboards-configmap.yaml

Recommended values.yaml keys (minimum)
- image.repository, image.tag, image.pullPolicy
- replicaCount
- resources (requests/limits)
- ingress.enabled, ingress.hosts, ingress.annotations (internal-only)
- keycloak:
  - issuer, clientId, realm, caCert
- kafka:
  - bootstrapServers, topicNames
- postgres:
  - jdbcUrl, userSecretRef
- vault:
  - enabled, path, mountMethod (CSI | runtime)
- slack:
  - outboundProxy, tokenSecretRef
- prometheus:
  - annotations: scrape: true

Secrets handling
- Do not store secrets in values.yaml. Use Vault CSI or Kubernetes External Secrets pattern. Chart must support both mount methods.

---

## Jenkins CI pipeline (stages)
Pipeline (multibranch or declarative) — stages:
1. Checkout
2. Build
   - mvn -DskipTests package (Java17)
   - static code analysis (SpotBugs/Checkstyle)
3. Unit tests (mvn test) — fail on regression
4. Container build & scan
   - Build image, scan for vulnerabilities
   - Push to internal registry
5. Helm lint
6. Deploy to dev
   - helm upgrade --install --namespace dev
7. Integration acceptance tests
   - Run automated harness (Slack simulator, Kafka assertions, DB validator, Prometheus checks)
8. Promote artifact
   - On success, tag image and optionally deploy to staging (manual approval)
9. Notify (Slack internal channel) with results

Credentials & secrets
- Jenkins retrieves OIDC client secrets, docker creds, and Vault token from enterprise secret store; does not store secrets in build logs.

Rollback & promotion
- Use helm rollback on failed post-deploy checks.
- Manual approval required for prod deployment.

---

## Rollout, monitoring & runbook highlights
- Deploy to dev -> run integration tests -> staging for canary -> prod with manual approval.
- Smoke tests: health endpoints, DB connectivity, Kafka consumer lag < threshold, example /coffee flow test.
- Rollback: helm rollback <release> <revision> and restart affected pods; escalate if Postgres/Kafka impacted.
- Alerting: Prometheus alerts for consumer lag, slack_api_errors rate, pod crashloops.
- Ops playbook: recovery steps for Kafka (consumer group offsets), Postgres failover, Vault unavailability.

---

## Risks (actionable) & mitigations
- Slack rate limits: buffer inbound events in Kafka (implemented); Notification Service uses exponential backoff and batching.
- Assignment bug: store reasoning in assignments_history and provide Admin API override.
- On‑prem outages: CI-run recovery drills; document recovery runbooks; use Postgres HA and Kafka replication.
- Secrets leakage: enforce Vault-only secrets and image scanning in CI.

---

## Deliverables for KIT phase (next)
- Working Helm chart (skeleton above) and values examples.
- Jenkins pipeline scripts (Jenkinsfile).
- Automated acceptance-test harness (Slack simulator, Kafka validator, DB scripts).
- SQL migration files and DB schema docs.
- Runner Assignment module (Java library) with unit tests and audit logging.
- Deployment runbook and rollback procedure.

---

If you want, I will:
- Convert this plan into a prioritized Jira-style backlog with ticket templates and acceptance-test checklists, or
- Produce the Helm values.yaml + minimal templates and a Jenkinsfile skeleton to jump straight to KIT. Which do you prefer?