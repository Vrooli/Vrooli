#!/usr/bin/env bash
# Quick verification script for new Tier 2 mocks

# Use cached APP_ROOT pattern for path robustness
APP_ROOT="${APP_ROOT:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")/.." && builtin pwd)}"
export APP_ROOT

echo "=== Testing New Tier 2 Mocks ==="
echo ""

# Test logs mock
echo -n "Testing logs: "
if bash -c '
    source "${APP_ROOT}/__test/mocks/tier2/logs.sh" >/dev/null 2>&1
    mock::log_call "test" "message" >/dev/null 2>&1
    echo "success"
' 2>/dev/null; then
    echo "✓ Works"
else
    echo "✗ Failed"
fi

# Test jq mock
echo -n "Testing jq: "
if bash -c '
    source "${APP_ROOT}/__test/mocks/tier2/jq.sh" >/dev/null 2>&1
    result=$(jq --version)
    [[ "$result" == "jq-1.6" ]] && echo "success"
' 2>/dev/null; then
    echo "✓ Works"
else
    echo "✗ Failed"
fi

# Test verification mock
echo -n "Testing verification: "
if bash -c '
    source "${APP_ROOT}/__test/mocks/tier2/verification.sh" >/dev/null 2>&1
    mock::verify_call "test" "arg" >/dev/null 2>&1
    echo "success"
' 2>/dev/null; then
    echo "✓ Works"
else
    echo "✗ Failed"
fi

# Test dig mock
echo -n "Testing dig: "
if bash -c '
    source "${APP_ROOT}/__test/mocks/tier2/dig.sh" >/dev/null 2>&1
    result=$(dig +short example.com)
    [[ -n "$result" ]] && echo "success"
' 2>/dev/null; then
    echo "✓ Works"
else
    echo "✗ Failed"
fi

echo ""
echo "=== Mock Count Verification ==="
echo "Total Tier 2 mocks: $(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" | wc -l)"
echo "All executable: $(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" -executable | wc -l)"
echo "Average lines: $(wc -l "${APP_ROOT}"/__test/mocks/tier2/*.sh 2>/dev/null | tail -1 | awk '{print int($1/'$(find "${APP_ROOT}/__test/mocks/tier2" -name "*.sh" | wc -l)')}')"