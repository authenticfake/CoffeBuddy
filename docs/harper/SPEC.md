# SPEC — CoffeeBuddy (On-Prem)

Summary
- CoffeeBuddy coordinates office coffee orders entirely on-prem: Slack-driven order creation, fair runner assignment, reminders, and persisted user preferences. Designed to run inside an enterprise Kubernetes cluster using the provided tech constraints (Java17 services, Postgres, Kafka, Keycloak OIDC, WSO2 API gateway internal-only, Vault, Prometheus/Grafana, Jenkins CI).

Goals
- Fast, low-friction coffee runs initiated from Slack.
- Fair, auditable runner assignment.
- Remembered user preferences (drink, size, extras).
- Operate fully inside corporate network; no public cloud dependencies.

Non-goals
- Payment processing, vendor integrations, delivery logistics.

Architecture (components)
- Slack Ingress Adapter (stateless service)
  - Receives Slack events (slash commands, interactions) via Kubernetes Ingress + WSO2 internal gateway.
  - Validates Slack signatures; forwards events to Kafka.
- Event Processor / Run Service (Java microservice)
  - Consumes Kafka events, applies business logic (create run, join/leave, assign runner, schedule reminders).
  - Persists domain state to Postgres.
  - Publishes domain events to Kafka for other consumers.
- Runner Assignment Service (library or microservice)
  - Deterministic fair-selection algorithm using user run history and opt-outs.
- Notification Service (Slack Outbound)
  - Posts messages/blocks back to Slack; sends reminders.
  - Uses Slack Web API via on-prem proxy.
- Admin API (REST, authenticated via Keycloak OIDC)
  - Manage settings, view runs, export audit logs.
- Scheduler (lightweight cron inside the Run Service)
  - Triggers reminders and time-based transitions (e.g., close ordering window).
- Observability stack
  - Prometheus metrics, Grafana dashboards, structured logs.
- Secrets & Config
  - Vault for Slack tokens, DB credentials, OIDC client secrets.
- CI/CD
  - Jenkins pipeline builds images, runs tests, and deploys Helm charts to on-prem K8s.

Data model (core tables)
- users
  - id (UUID), slack_user_id, display_name, email, preferences JSON, opt_out boolean, last_assigned_at timestamp
- runs
  - id (UUID), channel_id, created_by (user id), status (open/closed/completed), created_at, ordering_deadline, assigned_runner_id
- orders
  - id (UUID), run_id, user_id, item JSON (drink,size,extras), notes, created_at
- assignments_history
  - id, run_id, user_id, assigned_at, reason
- audit_events
  - id, event_type, payload JSON, created_at

Kafka topics (examples)
- slack.inbound.events — raw validated Slack events
- runs.commands — normalized commands (create_run, join_run, place_order, close_run)
- runs.events — domain events (run_created, order_placed, runner_assigned)
- notifications.outbound — messages to be sent to Slack
Retention and partitioning follow ops policy (use given 3 partitions baseline).

APIs & Contracts
- Slack -> Ingress (HTTP POST)
  - Accepts Slack event payloads; verifies signature; responds per Slack protocol.
- Internal: Runs REST (admin)
  - /api/v1/runs [GET] — list runs (OIDC required)
  - /api/v1/runs/{id} [GET] — run detail
  - /api/v1/users/{id}/preferences [PUT] — set preferences
  - All admin endpoints require Keycloak OIDC token and role-based scopes.
- Events: publish to Kafka topics listed above.

Core flows (sequence highlights)
1. Create run via Slack (/coffee)
  - Slack Adapter validates -> posts slack.inbound.events -> Event Processor consumes -> creates run row and default ordering_deadline -> publishes run_created -> Notification service posts summary to channel (includes join instructions).
2. Join & place order
  - User interacts or sends a message -> Slack event -> order command -> Event Processor writes orders record and publishes order_placed -> Notification updates run summary message.
3. Assign runner
  - At ordering_deadline (or when triggered), Runner Assignment Service selects eligible user using deterministic rotation honoring opt-outs and recent assignments -> assignment recorded and published -> Notification posts assigned runner and reminders.
4. Reminders
  - Scheduler triggers reminder events; notifications posted to Slack.
5. Completion
  - When run marked completed, persist final state and assignment history for audit.

Runner assignment algorithm (requirements)
- Deterministic, auditable, and fair:
  - Exclude users who opt out.
  - Prefer users with fewer recent assignments (round-robin / least-recently-assigned).
  - Avoid assigning same runner twice in a row unless only eligible runner remains.
  - Store assignment decisions and rationale in assignments_history for audit.

Security & Compliance
- No public outbound dependencies; Slack API calls routed through enterprise-approved proxy.
- All external endpoints internal-only via WSO2 APIM; ingress NGINX restricted to corporate network.
- Admin REST protected by Keycloak OIDC; service-to-service auth uses mutual TLS or signed tokens (ops to choose one consistent pattern).
- Secrets (Slack tokens, DB creds, OIDC client secrets) stored in Vault and mounted via CSI or fetched at startup.
- Database encryption at rest and in transit per enterprise policy.

Observability & SLOs
- Metrics to expose:
  - run_count_open, run_creation_latency, order_placement_latency, runner_assignment_latency, slack_api_errors, kafka_consumer_lag, processed_event_count, reminders_sent.
- Logs: structured JSON with correlation_id per run.
- Alerts:
  - High consumer lag, slack outbound error rate spike, failed assignment errors, pod crashlooping.
- Dashboards: run activity, fairness metrics (assignments per user), SLA for order confirmation times.

Operational constraints & mapping
- Runtime: Java17 services packaged as containers.
- Platform: Kubernetes (deploy via Helm charts).
- Messaging: Kafka with at least 3 partitions.
- DB: Postgres HA for relational state.
- API ingress: NGINX + WSO2 (internal-only).
- Auth: Keycloak OIDC.
- Secrets: HashiCorp Vault.
- CI: Jenkins pipelines to build, test, and deploy.

Risks & mitigations
- Slack rate limits — buffer inbound events in Kafka, apply exponential backoff for Slack Web API calls, batch updates to channel where possible.
- Single-point runner assignment bug — record assignment decisions and provide manual override via Admin API.
- On-prem outages (Kafka/Postgres) — define retention and retry semantics; document recovery runbooks.

Acceptance Criteria (required — measurable & testable)
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

Deliverables for the next phase (PLAN)
- Sequence of implementation sprints mapping components to work items.
- Acceptance-test automation plan (Slack event simulator, Kafka injectors, DB validators).
- Helm chart skeleton and CI pipeline tasks.

Notes / Constraints explicitly carried from IDEA.md
- Must run fully on-prem in enterprise network.
- Use Java17, Kubernetes, NGINX ingress, Postgres, Kafka (3 partitions baseline), Keycloak OIDC, Vault, WSO2 API gateway internal-only, Prometheus & Grafana, Jenkins CI.
- Slack events flow through on-prem gateway/proxy; rate limits mitigated via Kafka buffering and backoff.

Acceptance Criteria are mandatory and above.