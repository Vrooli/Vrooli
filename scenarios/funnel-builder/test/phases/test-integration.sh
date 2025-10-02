#!/bin/bash
# Integration tests for funnel-builder
# Tests component interactions and dependencies

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
SCENARIO_NAME="funnel-builder"

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

echo "Running integration tests..."

# Check scenario is running
run_test "Scenario is running" "vrooli scenario status '$SCENARIO_NAME'"

# Check PostgreSQL resource is accessible
run_test "PostgreSQL resource accessible" "resource-postgres status"

# Check database schema exists
run_test "Funnel builder schema exists" "docker exec vrooli-postgres-main psql -U vrooli -d vrooli -tAc \"SELECT 1 FROM information_schema.schemata WHERE schema_name = 'funnel_builder'\""

# Check required tables exist
run_test "Funnels table exists" "docker exec vrooli-postgres-main psql -U vrooli -d vrooli -tAc \"SELECT 1 FROM information_schema.tables WHERE table_schema = 'funnel_builder' AND table_name = 'funnels'\""

run_test "Funnel steps table exists" "docker exec vrooli-postgres-main psql -U vrooli -d vrooli -tAc \"SELECT 1 FROM information_schema.tables WHERE table_schema = 'funnel_builder' AND table_name = 'funnel_steps'\""

run_test "Leads table exists" "docker exec vrooli-postgres-main psql -U vrooli -d vrooli -tAc \"SELECT 1 FROM information_schema.tables WHERE table_schema = 'funnel_builder' AND table_name = 'leads'\""

# Summary
echo ""
echo "Integration Tests: $pass_count/$test_count passed"

if [ $fail_count -gt 0 ]; then
    exit 1
fi

exit 0
