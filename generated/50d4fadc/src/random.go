package main

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	mrand "math/rand"
	"time"
)

// Inizializza il seed per math/rand (usato come fallback e per i float)
func init() {
	mrand.Seed(time.Now().UnixNano())
}

// RandomInt restituisce un intero casuale nell'intervallo [min, max] (estremi inclusi).
// Usa crypto/rand per una distribuzione uniforme e sicura; se non disponibile,
// effettua il fallback su math/rand.
func RandomInt(min, max int) (int, error) {
	if min > max {
		min, max = max, min
	}
	// Usa int64 per evitare overflow quando calcoliamo l'ampiezza
	rangeSize := int64(max) - int64(min) + 1
	if rangeSize <= 0 {
		return 0, errors.New("intervallo non valido")
	}

	// Prova con crypto/rand
	nBig, err := crand.Int(crand.Reader, big.NewInt(rangeSize))
	if err == nil {
		return min + int(nBig.Int64()), nil
	}

	// Fallback su math/rand
	return min + int(mrand.Int63n(rangeSize)), nil
}

// RandomFloat restituisce un float64 casuale nell'intervallo [min, max).
func RandomFloat(min, max float64) (float64, error) {
	if min > max {
		min, max = max, min
	}
	if min == max {
		return min, nil
	}
	return min + mrand.Float64()*(max-min), nil
}

func main() {
	// Esempio d'uso
	v, err := RandomInt(1, 100)
	if err != nil {
		panic(err)
	}
	fmt.Println("Intero casuale [1,100]:", v)

	x, _ := RandomFloat(0, 1)
	fmt.Println("Float casuale [0,1):", x)
}
