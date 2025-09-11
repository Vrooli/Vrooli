# SageMath Mathematical Operations Guide

## Overview
SageMath provides comprehensive mathematical computing capabilities through a unified interface. This guide covers common operations and practical examples.

## Quick Start

### Basic Calculations
```bash
# Simple arithmetic
resource-sagemath content calculate "2 + 2"
resource-sagemath content calculate "factorial(10)"
resource-sagemath content calculate "sqrt(2).n(100)"  # 100 digits precision
```

### Symbolic Mathematics
```bash
# Differentiation
resource-sagemath content calculate "diff(sin(x), x)"
resource-sagemath content calculate "diff(x^3 + 2*x^2 - x + 1, x, 2)"  # Second derivative

# Integration
resource-sagemath content calculate "integrate(x^2, x)"
resource-sagemath content calculate "integrate(sin(x), x, 0, pi)"  # Definite integral

# Limits
resource-sagemath content calculate "limit((sin(x)/x), x=0)"
resource-sagemath content calculate "limit(1/x, x=0, dir='+')"  # One-sided limit
```

## Advanced Operations

### Linear Algebra
```bash
# Matrix operations
resource-sagemath content calculate "matrix([[1,2],[3,4]]).det()"
resource-sagemath content calculate "matrix([[1,2],[3,4]]).inverse()"
resource-sagemath content calculate "matrix([[1,2],[3,4]]).eigenvalues()"

# Vector operations
resource-sagemath content calculate "vector([1,2,3]).dot_product(vector([4,5,6]))"
resource-sagemath content calculate "vector([1,2,3]).cross_product(vector([4,5,6]))"
```

### Number Theory
```bash
# Prime numbers
resource-sagemath content calculate "is_prime(1234567891)"
resource-sagemath content calculate "next_prime(1000)"
resource-sagemath content calculate "prime_pi(1000)"  # Count of primes ≤ 1000

# Factorization
resource-sagemath content calculate "factor(2025)"
resource-sagemath content calculate "gcd(48, 18)"
resource-sagemath content calculate "lcm(12, 18)"

# Modular arithmetic
resource-sagemath content calculate "power_mod(2, 100, 13)"
resource-sagemath content calculate "inverse_mod(7, 13)"
```

### Calculus & Analysis
```bash
# Series expansion
resource-sagemath content calculate "taylor(sin(x), x, 0, 5)"
resource-sagemath content calculate "series(exp(x), x, 0, 5)"

# Differential equations
resource-sagemath content calculate "desolve(diff(y,x) + y == x, y)"
resource-sagemath content calculate "desolve(diff(y,x,2) + y == 0, y)"

# Partial fractions
resource-sagemath content calculate "partial_fraction(1/(x^2-1), x)"
```

### Statistics & Probability
```bash
# Combinations and permutations
resource-sagemath content calculate "binomial(10, 3)"
resource-sagemath content calculate "factorial(5)"
resource-sagemath content calculate "falling_factorial(10, 3)"

# Statistical functions
resource-sagemath content calculate "mean([1,2,3,4,5])"
resource-sagemath content calculate "std([1,2,3,4,5])"
resource-sagemath content calculate "variance([1,2,3,4,5])"
```

### Polynomial Operations
```bash
# Polynomial manipulation
resource-sagemath content calculate "expand((x + 1)^5)"
resource-sagemath content calculate "factor(x^4 - 1)"
resource-sagemath content calculate "(x^2 + 2*x + 1).roots()"

# Polynomial division
resource-sagemath content calculate "(x^3 + 2*x^2 - x - 2) // (x - 1)"
resource-sagemath content calculate "(x^3 + 2*x^2 - x - 2) % (x - 1)"
```

### Graph Theory
```bash
# Create and analyze graphs
resource-sagemath content calculate "graphs.PetersenGraph().chromatic_number()"
resource-sagemath content calculate "graphs.CompleteGraph(5).diameter()"
resource-sagemath content calculate "graphs.CycleGraph(6).is_bipartite()"
```

## Working with Scripts

### Creating Mathematical Scripts
```sage
# example_analysis.sage
# Comprehensive mathematical analysis example

# Define symbolic variables
x, y, z = var('x y z')

# Solve system of equations
equations = [
    x + y + z == 6,
    x - y + z == 2,
    2*x + y - z == 1
]
solution = solve(equations, [x, y, z])
print(f"Solution: {solution}")

# Analyze function properties
f = x^3 - 3*x^2 + 2*x
critical_points = solve(diff(f, x) == 0, x)
print(f"Critical points: {critical_points}")

# Compute definite integral
integral_result = integrate(f, x, 0, 2)
print(f"Integral from 0 to 2: {integral_result}")
```

### Running Scripts
```bash
# Add script to SageMath
resource-sagemath content add --file example_analysis.sage

# Execute script
resource-sagemath content execute example_analysis.sage

# View results
resource-sagemath content get example_analysis.sage
```

## Integration with Jupyter Notebooks

### Creating Notebooks
```python
# Create a Jupyter notebook with SageMath kernel
# example_notebook.ipynb
{
 "cells": [
  {
   "cell_type": "code",
   "source": [
    "# Complex number operations\n",
    "z = 3 + 4*I\n",
    "print(f'Magnitude: {abs(z)}')\n",
    "print(f'Argument: {arg(z)}')\n",
    "print(f'Conjugate: {conjugate(z)}')"
   ]
  }
 ]
}
```

### Managing Notebooks
```bash
# Add notebook
resource-sagemath content add --file example_notebook.ipynb

# Open notebook interface
resource-sagemath content notebook

# List available notebooks
resource-sagemath content list
```

## Performance Optimization

### Large-Scale Computations
```sage
# Efficient prime generation
primes_list = prime_range(1, 1000000)

# Fast matrix operations
A = random_matrix(ZZ, 100, 100)
det_A = A.determinant()  # Uses optimized algorithms

# Parallel computation (when available)
@parallel
def compute_factorial(n):
    return factorial(n)

results = list(compute_factorial([100, 200, 300, 400]))
```

### Memory Management
```sage
# Clear variables to free memory
reset()

# Use generators for large sequences
prime_gen = (p for p in prime_range(1, 10^9))

# Numerical approximations instead of exact symbolic
N(pi, 50)  # 50-digit approximation instead of symbolic pi
```

## Common Use Cases

### Financial Mathematics
```sage
# Compound interest
P = 1000  # Principal
r = 0.05  # Rate
n = 12    # Compounds per year
t = 10    # Time in years
A = P * (1 + r/n)^(n*t)
print(f"Amount after {t} years: ${A.n():.2f}")
```

### Engineering Calculations
```sage
# Signal processing - Fourier transform
f(t) = sin(2*pi*t) + 0.5*sin(4*pi*t)
fourier_transform = f.fourier_transform()
```

### Scientific Computing
```sage
# Differential equations for physics
# Harmonic oscillator
t = var('t')
x = function('x')(t)
DE = diff(x, t, 2) + x == 0
solution = desolve(DE, x, ics=[0, 1, 0])
```

## Troubleshooting

### Common Issues
- **Syntax Errors**: SageMath uses Python syntax with mathematical enhancements
- **Variable Declaration**: Use `var('x')` to declare symbolic variables
- **Precision Issues**: Use `.n()` for numerical approximations
- **Performance**: For large computations, consider using compiled functions

### Getting Help
```bash
# View function documentation
resource-sagemath content calculate "help(solve)"
resource-sagemath content calculate "factor?"  # Quick help
resource-sagemath content calculate "factor??"  # Source code
```

## Best Practices
1. **Use symbolic computation when exact answers are needed**
2. **Switch to numerical methods for performance with large datasets**
3. **Save frequently-used calculations as scripts**
4. **Leverage SageMath's built-in mathematical constants and functions**
5. **Use appropriate data structures (matrices, vectors, polynomials)**

## Additional Resources
- SageMath Documentation: https://doc.sagemath.org/
- Mathematical Functions Reference: https://doc.sagemath.org/html/en/reference/
- Tutorial: https://doc.sagemath.org/html/en/tutorial/