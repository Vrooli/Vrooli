#!/usr/bin/env sage
# Basic SageMath calculations example

print("SageMath Basic Calculations")
print("=" * 40)

# Symbolic algebra
print("\n1. Symbolic Algebra:")
x, y = var('x y')
expr = (x + y)^3
expanded = expr.expand()
print(f"   (x + y)^3 = {expanded}")

# Solving equations
print("\n2. Solving Equations:")
eq = x^2 + 5*x + 6 == 0
solutions = solve(eq, x)
print(f"   Equation: {eq}")
print(f"   Solutions: {solutions}")

# Calculus
print("\n3. Calculus:")
f = sin(x) * cos(x)
df = diff(f, x)
integral = integrate(f, x)
print(f"   f(x) = {f}")
print(f"   f'(x) = {df}")
print(f"   ∫f(x)dx = {integral}")

# Linear algebra
print("\n4. Linear Algebra:")
A = matrix([[1, 2, 3], [4, 5, 6], [7, 8, 10]])
det_A = A.det()
inv_A = A.inverse()
print(f"   Matrix A:")
print(A)
print(f"   det(A) = {det_A}")
print(f"   A^(-1) =")
print(inv_A)

# Number theory
print("\n5. Number Theory:")
n = 100
factors = factor(n)
primes = prime_range(1, 50)
print(f"   Factorization of {n}: {factors}")
print(f"   Primes less than 50: {primes}")

# Plotting (save to file)
print("\n6. Plotting:")
plot_file = "/home/sage/outputs/sine_cosine_plot.png"
p = plot([sin(x), cos(x)], (x, -2*pi, 2*pi), 
         legend_label=['sin(x)', 'cos(x)'],
         color=['blue', 'red'])
p.save(plot_file)
print(f"   Plot saved to: {plot_file}")

print("\n" + "=" * 40)
print("All calculations completed successfully!")