#!/usr/bin/env bash
# Test that qdrant exports.sh produces expected environment variables.
# Regression test for: Qdrant defaults.sh wrapped vars in a function that
# the lifecycle system never called, so QDRANT_URL was never set.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../../../.." && pwd)}"

EXPORTS_FILE="${APP_ROOT}/resources/qdrant/config/exports.sh"

if [[ ! -f "$EXPORTS_FILE" ]]; then
    echo "FAIL: exports.sh not found at $EXPORTS_FILE"
    exit 1
fi

# Source exports.sh in a subshell and capture environment
EXPORTED_ENV=$(
    export APP_ROOT="$APP_ROOT"
    # shellcheck disable=SC1090
    source "$EXPORTS_FILE" 2>/dev/null
    env | grep "^QDRANT_" | sort
)

PASS=0
FAIL=0

check_var() {
    local var_name="$1"
    local expected_pattern="${2:-}"

    local value
    value=$(echo "$EXPORTED_ENV" | grep "^${var_name}=" | head -1 | cut -d= -f2-)

    if [[ -z "$value" ]]; then
        echo "FAIL: $var_name not exported"
        FAIL=$((FAIL + 1))
        return
    fi

    if [[ -n "$expected_pattern" ]] && [[ ! "$value" =~ $expected_pattern ]]; then
        echo "FAIL: $var_name=$value does not match pattern '$expected_pattern'"
        FAIL=$((FAIL + 1))
        return
    fi

    echo "PASS: $var_name=$value"
    PASS=$((PASS + 1))
}

echo "=== Qdrant exports.sh tests ==="

# Critical: QDRANT_URL must be exported (this was the root cause bug)
check_var "QDRANT_URL" "^http://.*:[0-9]+"
check_var "QDRANT_BASE_URL" "^http://.*:[0-9]+"
check_var "QDRANT_PORT" "^[0-9]+"
check_var "QDRANT_GRPC_PORT" "^[0-9]+"
check_var "QDRANT_HOST"
check_var "QDRANT_HEALTH_CHECK" "healthz"
check_var "QDRANT_CONTAINER_NAME"
check_var "QDRANT_RESOURCE_VERSION"

# Verify URL and BASE_URL are identical
URL_VAL=$(echo "$EXPORTED_ENV" | grep "^QDRANT_URL=" | cut -d= -f2-)
BASE_VAL=$(echo "$EXPORTED_ENV" | grep "^QDRANT_BASE_URL=" | cut -d= -f2-)
if [[ "$URL_VAL" == "$BASE_VAL" ]]; then
    echo "PASS: QDRANT_URL == QDRANT_BASE_URL ($URL_VAL)"
    PASS=$((PASS + 1))
else
    echo "FAIL: QDRANT_URL ($URL_VAL) != QDRANT_BASE_URL ($BASE_VAL)"
    FAIL=$((FAIL + 1))
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"

if [[ "$FAIL" -gt 0 ]]; then
    exit 1
fi
