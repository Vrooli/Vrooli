#!/usr/bin/env bash
set -euo pipefail

# Parlant Smoke Test
# Quick validation that service is healthy (<30s)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "${SCRIPT_DIR}")"
RESOURCE_DIR="$(dirname "${TEST_DIR}")"
LIB_DIR="${RESOURCE_DIR}/lib"

# Source test library
source "${LIB_DIR}/test.sh"

# Run smoke test
parlant_test_smoke