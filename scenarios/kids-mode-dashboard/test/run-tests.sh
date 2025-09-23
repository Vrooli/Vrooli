#!/bin/bash
set -e

echo "=== Running tests for kids-mode-dashboard ==="

# Run dependency tests
echo "Running dependency tests..."
./phases/test-dependencies.sh

# Run business logic tests
echo "Running business logic tests..."
./phases/test-business.sh

echo "✅ All tests completed successfully!"