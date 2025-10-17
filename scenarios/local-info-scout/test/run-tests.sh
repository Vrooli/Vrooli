#!/bin/bash
set -e

echo "Running tests for local-info-scout"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$SCRIPT_DIR/phases"
./test-structure.sh
./test-dependencies.sh
./test-business.sh
./test-integration.sh
./test-performance.sh

echo "All tests passed!"