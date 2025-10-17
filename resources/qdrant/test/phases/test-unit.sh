#!/usr/bin/env bash
# Qdrant Unit Test Phase - v2.0 Compliant  
# Library function validation (<60s)

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"

# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/test.sh"

# Run unit tests
qdrant::test_unit