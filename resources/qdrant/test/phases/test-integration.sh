#!/usr/bin/env bash
# Qdrant Integration Test Phase - v2.0 Compliant
# End-to-end functionality (<120s)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"

# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/test.sh"

# Run integration tests
qdrant::test_integration