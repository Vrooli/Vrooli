# Fibonacci Sequence - Multi-Language Example

This example demonstrates the same Fibonacci algorithm implemented in multiple languages, showing Judge0's versatility.

## JavaScript
```javascript
// fibonacci.js
function fibonacci(n) {
    if (n <= 1) return n;
    let a = 0, b = 1;
    for (let i = 2; i <= n; i++) {
        [a, b] = [b, a + b];
    }
    return b;
}

// Calculate first 10 Fibonacci numbers
for (let i = 0; i < 10; i++) {
    console.log(`F(${i}) = ${fibonacci(i)}`);
}
```

## Python
```python
# fibonacci.py
def fibonacci(n):
    if n <= 1:
        return n
    a, b = 0, 1
    for _ in range(2, n + 1):
        a, b = b, a + b
    return b

# Calculate first 10 Fibonacci numbers
for i in range(10):
    print(f"F({i}) = {fibonacci(i)}")
```

## Go
```go
// fibonacci.go
package main

import "fmt"

func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    a, b := 0, 1
    for i := 2; i <= n; i++ {
        a, b = b, a+b
    }
    return b
}

func main() {
    // Calculate first 10 Fibonacci numbers
    for i := 0; i < 10; i++ {
        fmt.Printf("F(%d) = %d\n", i, fibonacci(i))
    }
}
```

## Rust
```rust
// fibonacci.rs
fn fibonacci(n: u32) -> u32 {
    if n <= 1 {
        return n;
    }
    let mut a = 0;
    let mut b = 1;
    for _ in 2..=n {
        let temp = a + b;
        a = b;
        b = temp;
    }
    b
}

fn main() {
    // Calculate first 10 Fibonacci numbers
    for i in 0..10 {
        println!("F({}) = {}", i, fibonacci(i));
    }
}
```

## Java
```java
// Fibonacci.java
public class Fibonacci {
    public static int fibonacci(int n) {
        if (n <= 1) return n;
        int a = 0, b = 1;
        for (int i = 2; i <= n; i++) {
            int temp = a + b;
            a = b;
            b = temp;
        }
        return b;
    }
    
    public static void main(String[] args) {
        // Calculate first 10 Fibonacci numbers
        for (int i = 0; i < 10; i++) {
            System.out.println("F(" + i + ") = " + fibonacci(i));
        }
    }
}
```

## C++
```cpp
// fibonacci.cpp
#include <iostream>
using namespace std;

int fibonacci(int n) {
    if (n <= 1) return n;
    int a = 0, b = 1;
    for (int i = 2; i <= n; i++) {
        int temp = a + b;
        a = b;
        b = temp;
    }
    return b;
}

int main() {
    // Calculate first 10 Fibonacci numbers
    for (int i = 0; i < 10; i++) {
        cout << "F(" << i << ") = " << fibonacci(i) << endl;
    }
    return 0;
}
```

## Testing with Judge0

Run all implementations and compare:
```bash
# Test each language
for lang in javascript python go rust java cpp; do
    echo "Testing $lang..."
    ./manage.sh --action submit \
        --code "@fibonacci.$lang" \
        --language "$lang"
done
```

Expected output for all languages:
```
F(0) = 0
F(1) = 1
F(2) = 1
F(3) = 2
F(4) = 3
F(5) = 5
F(6) = 8
F(7) = 13
F(8) = 21
F(9) = 34
```