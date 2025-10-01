def next_odd_number(n):
    """
    Return the next odd number after the given integer n.
    If n is odd, return n + 2.
    If n is even, return n + 1.
    """
    if n % 2 == 0:
        return n + 1
    else:
        return n + 2

# Example usage:
# print(next_odd_number(4))  # Output: 5
# print(next_odd_number(5))  # Output: 7
