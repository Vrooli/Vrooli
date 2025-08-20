#!/usr/bin/env sage
# SageMath cryptography example

print("SageMath Cryptography Examples")
print("=" * 40)

# RSA key generation example
print("\n1. RSA Key Generation:")
p = random_prime(2^512)
q = random_prime(2^512)
n = p * q
phi_n = (p-1) * (q-1)
e = 65537  # Common public exponent

# Find private exponent d
d = inverse_mod(e, phi_n)

print(f"   Key size: {n.nbits()} bits")
print(f"   Public exponent (e): {e}")
print(f"   Private exponent (d): {d}")

# Elliptic curves
print("\n2. Elliptic Curves:")
E = EllipticCurve(GF(97), [2, 3])
print(f"   Curve: {E}")
print(f"   Order: {E.order()}")

# Find some points
points = E.rational_points()[:5]
print(f"   First 5 points: {points}")

# Discrete logarithm
print("\n3. Discrete Logarithm:")
p = 1019
g = primitive_root(p)
h = power_mod(g, 234, p)
print(f"   Prime p: {p}")
print(f"   Generator g: {g}")
print(f"   h = g^234 mod p = {h}")

# Try to solve discrete log (small example)
for x in range(100):
    if power_mod(g, x, p) == h:
        print(f"   Found x = {x} such that g^x = h (mod p)")
        break

# Hash functions (using Python's hashlib via sage)
print("\n4. Hash Functions:")
import hashlib
message = "Hello, SageMath!"
sha256_hash = hashlib.sha256(message.encode()).hexdigest()
print(f"   Message: {message}")
print(f"   SHA-256: {sha256_hash}")

# Primality testing
print("\n5. Primality Testing:")
test_numbers = [561, 1105, 1729, 2047, 2465]
for n in test_numbers:
    is_prime = is_prime(n)
    is_pseudoprime = is_pseudoprime(n)
    print(f"   {n}: Prime={is_prime}, Pseudoprime={is_pseudoprime}")

print("\n" + "=" * 40)
print("Cryptography examples completed!")