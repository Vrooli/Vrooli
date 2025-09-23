#!/bin/bash

set -e

echo "Running Brand Manager test suite..."

cd "$(dirname "$0")/phases"

echo "Executing test phases..."

./test-business.sh
./test-dependencies.sh
./test-integration.sh
./test-performance.sh
./test-structure.sh
./test-unit.sh

echo "All test phases completed successfully!"