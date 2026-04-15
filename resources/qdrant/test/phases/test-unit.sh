#!/usr/bin/env bash
# Qdrant Unit Test Phase - v2.0 Compliant  
# Library function validation (<60s)

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
QDRANT_LIB_DIR="${RESOURCE_DIR}/lib"

# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/test.sh"

# Run unit tests
qdrant::test_unit