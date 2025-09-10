#!/usr/bin/env bash
# Matrix Synapse Resource - Smoke Tests
# Quick validation that must complete in <30 seconds

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"

# Run smoke tests
test_smoke