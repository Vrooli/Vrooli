#!/usr/bin/env bash
# Qdrant Integration Test Phase - v2.0 Compliant
# End-to-end functionality (<120s)

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
QDRANT_LIB_DIR="${RESOURCE_DIR}/lib"

# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/test.sh"

# Run integration tests
qdrant::test_integration