#!/usr/bin/env bash
# Matrix Synapse Resource - Unit Tests
# Library function validation

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run unit tests
test_unit