#!/bin/bash

# OpenFOAM Smoke Test Phase
# Quick health validation (<30s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "$SCRIPT_DIR")"
RESOURCE_DIR="$(dirname "$TEST_DIR")"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Run smoke tests
openfoam::test::smoke