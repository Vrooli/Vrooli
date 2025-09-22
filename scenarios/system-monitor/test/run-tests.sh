#!/bin/bash
set -e

echo "Running comprehensive tests for System Monitor"

# Run all phases
echo "=== Running test phases ==="
cd phases && ./test-unit.sh
./test-business.sh
./test-dependencies.sh
./test-integration.sh
./test-performance.sh
./test-structure.sh
cd ..

echo "All tests completed successfully!"
