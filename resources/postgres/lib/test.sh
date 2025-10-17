#!/usr/bin/env bash
# PostgreSQL Resource Test Library
# Delegates to test/run-tests.sh for v2.0 contract compliance

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGRES_TEST_DIR="${APP_ROOT}/resources/postgres/test"

# Delegate to test runner
postgres::test::run() {
    local test_type="${1:-all}"
    
    if [[ -f "${POSTGRES_TEST_DIR}/run-tests.sh" ]]; then
        bash "${POSTGRES_TEST_DIR}/run-tests.sh" "$test_type"
        return $?
    else
        echo "Error: Test runner not found at ${POSTGRES_TEST_DIR}/run-tests.sh" >&2
        return 1
    fi
}

# Export for CLI usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    postgres::test::run "$@"
fi