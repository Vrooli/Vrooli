#!/bin/bash
set -e
echo "Running all tests..."
./test/phases/test-structure.sh
./test/phases/test-dependencies.sh
./test/phases/test-unit.sh
./test/phases/test-integration.sh
./test/phases/test-business.sh
./test/phases/test-performance.sh
echo "All tests completed."