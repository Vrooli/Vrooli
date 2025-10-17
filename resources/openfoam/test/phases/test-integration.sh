#!/bin/bash

# OpenFOAM Integration Test Phase
# End-to-end functionality testing

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Run integration tests
openfoam::test::integration