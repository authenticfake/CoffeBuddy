#include <stdio.h>
#include <stdlib.h>
#include <time.h>

void generate_random_numbers(int count) {
    // Seed the random number generator
    srand(time(NULL));
    
    for (int i = 0; i < count; i++) {
        int random_number = rand(); // Generate a random number
        printf("Random Number %d: %d\n", i + 1, random_number);
    }
}

int main() {
    int count;
    printf("Enter the number of random numbers to generate: ");
    scanf("%d", &count);
    generate_random_numbers(count);
    return 0;
}