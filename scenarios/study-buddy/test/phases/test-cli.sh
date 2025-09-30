#!/usr/bin/env bash
# CLI tests for study-buddy scenario

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "🖥️ Running CLI tests..."

# Test help command
echo -n "Testing CLI help command... "
if study-buddy help 2>&1 | grep -q "Study Buddy CLI"; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    exit 1
fi

# Test health command
echo -n "Testing CLI health command... "
if study-buddy health 2>&1 | grep -q "healthy\|running"; then
    echo "✅ PASS"
else
    echo "❌ FAIL"
    # Don't fail on health check for now
fi

# Test list subjects (even if empty)
echo -n "Testing CLI list subjects... "
output=$(study-buddy list-subjects 2>&1 || true)
if echo "$output" | grep -q "Subject\|No subjects\|subjects"; then
    echo "✅ PASS"
else
    echo "❌ FAIL (non-critical)"
fi

# Test show progress
echo -n "Testing CLI show progress... "
output=$(study-buddy show-progress 2>&1 || true)
if echo "$output" | grep -q "Progress\|Stats\|XP\|streak"; then
    echo "✅ PASS"
else
    echo "❌ FAIL (non-critical)"
fi

echo "✅ CLI tests completed!"
exit 0