#!/bin/bash

# OpenFOAM Unit Test Phase
# Library function validation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Run unit tests
openfoam::test::unit