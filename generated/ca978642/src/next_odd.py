def next_odd(n):
    """
    Restituisce il prossimo numero dispari strettamente maggiore di n.

    Parametri:
    - n: valore intero (o convertibile in intero). Se non è convertibile viene sollevato TypeError.

    Esempi:
    next_odd(4) -> 5
    next_odd(5) -> 7
    next_odd(-2) -> -1
    next_odd(2.9) -> 3  # viene considerata la parte intera (2)
    """
    try:
        n_int = int(n)
    except Exception:
        raise TypeError("Il valore di input deve essere un intero o convertibile in intero")

    if n_int % 2 == 0:
        return n_int + 1
    else:
        return n_int + 2


# Esempi di utilizzo
if __name__ == "__main__":
    test_values = [3, 4, 0, -1, 2.9, "5"]
    for v in test_values:
        print(f"{v} -> {next_odd(v)}")
