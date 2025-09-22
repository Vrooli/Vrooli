#!/bin/bash

set -e

echo "Running all tests for QR Code Generator"

cd test/phases

./test-structure.sh
./test-dependencies.sh
./test-unit.sh
./test-integration.sh
./test-performance.sh
./test-business.sh

echo "All tests completed."
