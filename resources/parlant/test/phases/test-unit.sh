#!/usr/bin/env bash
set -euo pipefail

# Parlant Unit Test
# Library function validation (<60s)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(dirname "${SCRIPT_DIR}")"
RESOURCE_DIR="$(dirname "${TEST_DIR}")"
LIB_DIR="${RESOURCE_DIR}/lib"

# Source test library
source "${LIB_DIR}/test.sh"

# Run unit test
parlant_test_unit