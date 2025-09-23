#!/bin/bash
set -e

echo "Running tests for local-info-scout"

cd phases
./test-structure.sh
./test-dependencies.sh
./test-business.sh
./test-integration.sh
./test-performance.sh

echo "All tests passed!"