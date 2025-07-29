# Python example demonstrating input/output handling
# This shows how to work with stdin in Judge0

# Read user input
name = input("Enter your name: ")
age = int(input("Enter your age: "))

# Process and output
print(f"Hello, {name}!")
print(f"In 10 years, you'll be {age + 10} years old.")

# Test with multiple lines of input
print("\nEnter numbers (one per line, enter 0 to stop):")
numbers = []
while True:
    num = int(input())
    if num == 0:
        break
    numbers.append(num)

if numbers:
    print(f"Sum: {sum(numbers)}")
    print(f"Average: {sum(numbers) / len(numbers):.2f}")
    print(f"Max: {max(numbers)}")
    print(f"Min: {min(numbers)}")