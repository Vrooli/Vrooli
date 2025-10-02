#!/bin/bash
# Unit tests for funnel-builder
# Tests individual components in isolation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_count=0
pass_count=0
fail_count=0

run_test() {
    local test_name="$1"
    local test_command="$2"

    test_count=$((test_count + 1))
    echo -n "  Testing $test_name... "

    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        pass_count=$((pass_count + 1))
        return 0
    else
        echo -e "${RED}✗${NC}"
        fail_count=$((fail_count + 1))
        return 1
    fi
}

echo "Running unit tests..."

# Go API compilation test
if [ -f "$SCENARIO_DIR/api/main.go" ]; then
    run_test "Go API compilation" "cd '$SCENARIO_DIR/api' && go build -o /tmp/funnel-builder-test-build . && rm /tmp/funnel-builder-test-build"
fi

# UI TypeScript compilation test (if applicable)
if [ -f "$SCENARIO_DIR/ui/package.json" ]; then
    run_test "UI TypeScript check" "cd '$SCENARIO_DIR/ui' && npm run build &>/dev/null"
fi

# Database schema validation
if [ -f "$SCENARIO_DIR/initialization/storage/postgres/schema.sql" ]; then
    run_test "Database schema syntax" "grep -q 'CREATE TABLE' '$SCENARIO_DIR/initialization/storage/postgres/schema.sql'"
fi

# CLI script syntax
if [ -f "$SCENARIO_DIR/cli/funnel-builder" ]; then
    run_test "CLI script syntax" "bash -n '$SCENARIO_DIR/cli/funnel-builder'"
fi

# Summary
echo ""
echo "Unit Tests: $pass_count/$test_count passed"

if [ $fail_count -gt 0 ]; then
    exit 1
fi

exit 0
