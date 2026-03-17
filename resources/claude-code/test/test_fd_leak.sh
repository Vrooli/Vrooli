#!/usr/bin/env bash
# Regression test: fd 3 must NOT be inherited by the claude process.
#
# Root cause: execute.sh uses `exec 3>&1 1>&2` to save stdout as fd 3 before
# redirecting log output to stderr. The `{...} | tee ... >&3` pipeline uses
# fd 3 to route Claude's JSON stream back to the calling process. Without
# `3>&-` on the timeout/claude command, claude inherits fd 3 — a write end
# of the pipe back to the calling process. This leaked fd prevents claude's
# Node.js event loop from exiting after completing its work, causing runs
# to appear stuck in "Running" state indefinitely.
#
# Fix: Add `3>&-` to the timeout/claude invocation to close the inherited fd.

set -euo pipefail

TESTS_PASSED=0
TESTS_FAILED=0

pass() { echo "  PASS: $1"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail() { echo "  FAIL: $1"; TESTS_FAILED=$((TESTS_FAILED + 1)); }

echo "=== Regression test: fd 3 leak in execute.sh pipeline ==="

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXECUTE_SH="${SCRIPT_DIR}/lib/execute.sh"

# Test 1: Verify that execute.sh contains `3>&-` on the claude invocation
echo "Test 1: execute.sh closes fd 3 for claude invocations"
if [[ ! -f "$EXECUTE_SH" ]]; then
    fail "execute.sh not found at $EXECUTE_SH"
else
    if grep -q 'timeout.*claude.*3>&-' "$EXECUTE_SH"; then
        pass "claude invocations close fd 3"
    else
        fail "claude invocations missing 3>&- (fd 3 leak)"
    fi
fi

# Test 2: Verify both stream-json and text mode paths have the fix
echo "Test 2: both output mode paths close fd 3"
MATCH_COUNT=$(grep -c 'timeout.*claude.*3>&-' "$EXECUTE_SH" 2>/dev/null || echo 0)
if [[ "$MATCH_COUNT" -ge 2 ]]; then
    pass "both output paths close fd 3 ($MATCH_COUNT occurrences)"
elif [[ "$MATCH_COUNT" -eq 1 ]]; then
    fail "only one output path closes fd 3 (need both stream-json and text)"
else
    fail "no output paths close fd 3"
fi

# Test 3: Verify `3>&-` actually prevents fd inheritance (functional test)
echo "Test 3: 3>&- prevents fd inheritance in pipeline pattern"
RESULT_FILE=$(mktemp)
(
    # Simulate execute.sh pattern: save stdout as fd 3
    exec 3>&1 1>&2
    # Child with 3>&- should NOT have fd 3
    {
        if [[ -e /proc/self/fd/3 ]]; then
            echo "LEAKED" > "$RESULT_FILE"
        else
            echo "CLOSED" > "$RESULT_FILE"
        fi
    } 3>&-
) 2>/dev/null
RESULT=$(cat "$RESULT_FILE" 2>/dev/null || echo "ERROR")
rm -f "$RESULT_FILE"
if [[ "$RESULT" == "CLOSED" ]]; then
    pass "3>&- prevents fd 3 inheritance"
elif [[ "$RESULT" == "LEAKED" ]]; then
    fail "fd 3 still inherited despite 3>&-"
else
    fail "test error: could not determine fd 3 status"
fi

# Test 4: Verify fd 3 IS inherited without the fix (control test)
echo "Test 4: control - fd 3 IS inherited without 3>&-"
RESULT_FILE=$(mktemp)
(
    exec 3>&1 1>&2
    # Child without 3>&- SHOULD have fd 3
    {
        if [[ -e /proc/self/fd/3 ]]; then
            echo "INHERITED" > "$RESULT_FILE"
        else
            echo "NOT_INHERITED" > "$RESULT_FILE"
        fi
    }
) 2>/dev/null
RESULT=$(cat "$RESULT_FILE" 2>/dev/null || echo "ERROR")
rm -f "$RESULT_FILE"
if [[ "$RESULT" == "INHERITED" ]]; then
    pass "control: fd 3 is inherited without 3>&- (confirms bug mechanism)"
else
    fail "control: fd 3 was not inherited even without 3>&- (test may be broken)"
fi

echo ""
echo "=== Results: $TESTS_PASSED passed, $TESTS_FAILED failed ==="
[[ $TESTS_FAILED -eq 0 ]] && exit 0 || exit 1
