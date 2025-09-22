#!/bin/bash
set -e

echo \"Running tests for scenario-to-ios\"

cd test/phases

./test-unit.sh
./test-integration.sh
./test-structure.sh
./test-dependencies.sh
./test-performance.sh
./test-business.sh

echo \"All tests completed successfully\"