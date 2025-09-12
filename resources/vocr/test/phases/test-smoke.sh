#!/usr/bin/env bash
################################################################################
# VOCR Smoke Test Phase - Quick health validation (<30s)
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "${PHASE_DIR}/.." && pwd)"
VOCR_DIR="$(cd "${TEST_DIR}/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "${VOCR_DIR}/../.." && pwd)}"

# Source test library
source "${VOCR_DIR}/lib/test.sh"

# Run smoke tests
vocr::test::smoke