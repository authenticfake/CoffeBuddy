// Restituisce il prossimo numero dispari strettamente maggiore di n.
// Se n non è un numero viene lanciato un TypeError.
function nextOdd(n) {
  if (!Number.isFinite(n)) {
    throw new TypeError('Il valore di input deve essere un numero finito');
  }
  // Consideriamo la parte intera (troncamento) per comportarci come int()
  const nInt = Math.trunc(n);
  return (nInt % 2 === 0) ? nInt + 1 : nInt + 2;
}

// Esempi di utilizzo
if (require.main === module) {
  const examples = [3, 4, 0, -1, 2.9];
  examples.forEach(x => console.log(`${x} -> ${nextOdd(x)}`));
}

module.exports = nextOdd;
