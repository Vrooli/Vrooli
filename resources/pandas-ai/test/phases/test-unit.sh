#!/usr/bin/env bash
set -euo pipefail

# Unit tests for Pandas-AI library functions

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Source the test library and run unit tests
source "${APP_ROOT}/resources/pandas-ai/lib/test.sh"

# Run unit tests
pandas_ai::test::unit
exit $?