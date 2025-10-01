Funzione "next odd" (prossimo dispari)

Questo progetto contiene due semplici implementazioni della funzione che, dato un valore in input,
restituisce il prossimo numero dispari strettamente maggiore.

File:
- next_odd.py: implementazione Python con esempi d'uso
- nextOdd.js: implementazione JavaScript con esempi d'uso

Regole/Comportamento:
- L'input viene convertito in intero (tramite int() in Python o Math.trunc() in JavaScript).
- Se l'intero risultante è pari, la funzione restituisce intero + 1.
- Se l'intero risultante è dispari, la funzione restituisce intero + 2 (cioè il prossimo dispari successivo).
- Se l'input non è convertibile in numero/intero, viene sollevato un errore.

Esempi:
- next_odd(4) -> 5
- next_odd(5) -> 7
- next_odd(-2) -> -1

Uso rapido:
- Python: python3 next_odd.py
- JavaScript: node nextOdd.js
