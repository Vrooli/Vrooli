#!/usr/bin/env bash
# Test that qdrant resource.json declares the native environment export contract.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="${APP_ROOT:-$(cd "$SCRIPT_DIR/../../../.." && pwd)}"

MANIFEST_FILE="${APP_ROOT}/resources/qdrant/resource.json"

if [[ ! -f "$MANIFEST_FILE" ]]; then
    echo "FAIL: resource.json not found at $MANIFEST_FILE"
    exit 1
fi

PASS=0
FAIL=0

check_declared_export() {
    local jq_expr="$1"
    local label="$2"
    local expected_pattern="${3:-}"
    local value
    value=$(jq -r "$jq_expr // empty" "$MANIFEST_FILE")

    if [[ -z "$value" ]]; then
        echo "FAIL: $label not declared"
        FAIL=$((FAIL + 1))
        return
    fi

    if [[ -n "$expected_pattern" ]] && [[ ! "$value" =~ $expected_pattern ]]; then
        echo "FAIL: $label=$value does not match pattern '$expected_pattern'"
        FAIL=$((FAIL + 1))
        return
    fi

    echo "PASS: $label=$value"
    PASS=$((PASS + 1))
}

echo "=== Qdrant native environment export tests ==="

check_declared_export '.environment_exports.static.QDRANT_HOST' "QDRANT_HOST"
check_declared_export '.environment_exports.from_ports.QDRANT_PORT' "QDRANT_PORT mapping" '^http$'
check_declared_export '.environment_exports.from_ports.QDRANT_GRPC_PORT' "QDRANT_GRPC_PORT mapping" '^grpc$'
check_declared_export '.environment_exports.derived.QDRANT_URL.template' "QDRANT_URL template" '^http://'
check_declared_export '.environment_exports.derived.QDRANT_BASE_URL.template' "QDRANT_BASE_URL template" 'QDRANT_URL'
check_declared_export '.environment_exports.derived.QDRANT_GRPC_URL.template' "QDRANT_GRPC_URL template" '^grpc://'

BASE_TEMPLATE=$(jq -r '.environment_exports.derived.QDRANT_BASE_URL.template // empty' "$MANIFEST_FILE")
if [[ "$BASE_TEMPLATE" == '${QDRANT_URL}' ]]; then
    echo "PASS: QDRANT_BASE_URL derives from QDRANT_URL ($BASE_TEMPLATE)"
    PASS=$((PASS + 1))
else
    echo "FAIL: QDRANT_BASE_URL template ($BASE_TEMPLATE) does not derive from QDRANT_URL"
    FAIL=$((FAIL + 1))
fi

echo ""
echo "Results: $PASS passed, $FAIL failed"

if [[ "$FAIL" -gt 0 ]]; then
    exit 1
fi
