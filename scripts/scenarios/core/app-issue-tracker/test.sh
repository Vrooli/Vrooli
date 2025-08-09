#!/usr/bin/env bash
# Issue Tracker Integration Test
# Validates all components of the centralized issue tracking system

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${SCENARIO_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper
test::run_test() {
    local test_name=$1
    local test_command=$2
    
    log::info "Testing: $test_name... "
    
    if eval "$test_command" > /dev/null 2>&1; then
        log::success "✓ $test_name"
        ((TESTS_PASSED++))
        return 0
    else
        log::error "✗ $test_name"
        ((TESTS_FAILED++))
        return 1
    fi
}

log::info "🧪 Running Issue Tracker Integration Tests"
log::info "==========================================="

# Get resource ports
POSTGRES_PORT=$(resources::get_default_port "postgres")
QDRANT_PORT=$(resources::get_default_port "qdrant")
REDIS_PORT=$(resources::get_default_port "redis")
MINIO_PORT=$(resources::get_default_port "minio")
WINDMILL_PORT=$(resources::get_default_port "windmill")
N8N_PORT=$(resources::get_default_port "n8n")
OLLAMA_PORT=$(resources::get_default_port "ollama")

# Test 1: Service Connectivity
echo ""
echo "1. Service Connectivity Tests"
echo "------------------------------"
test::run_test "PostgreSQL connection" "nc -z localhost $POSTGRES_PORT"
test::run_test "Qdrant connection" "nc -z localhost $QDRANT_PORT"
test::run_test "Redis connection" "nc -z localhost $REDIS_PORT"
test::run_test "MinIO connection" "nc -z localhost $MINIO_PORT"
test::run_test "Windmill connection" "nc -z localhost $WINDMILL_PORT"
test::run_test "n8n connection" "nc -z localhost $N8N_PORT"
test::run_test "Ollama connection" "nc -z localhost $OLLAMA_PORT"

# Test 2: Database Schema
echo ""
echo "2. Database Schema Tests"
echo "------------------------"
test::run_test "Database exists" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -lqt | grep -q issue_tracker"
test::run_test "Apps table exists" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -c 'SELECT 1 FROM apps LIMIT 1;'"
test::run_test "Issues table exists" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -c 'SELECT 1 FROM issues LIMIT 1;'"
test::run_test "Agents table exists" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -c 'SELECT 1 FROM agents LIMIT 1;'"
test::run_test "Pattern groups table exists" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -c 'SELECT 1 FROM pattern_groups LIMIT 1;'"

# Test 3: Seed Data
echo ""
echo "3. Seed Data Tests"
echo "------------------"
test::run_test "Sample apps loaded" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -t -c 'SELECT COUNT(*) FROM apps;' | grep -q '[1-9]'"
test::run_test "Agent templates loaded" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -t -c 'SELECT COUNT(*) FROM agents;' | grep -q '[1-9]'"
test::run_test "Pattern groups loaded" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -t -c 'SELECT COUNT(*) FROM pattern_groups;' | grep -q '[1-9]'"

# Test 4: Vector Database
echo ""
echo "4. Vector Database Tests"
echo "------------------------"
test::run_test "Qdrant health check" "curl -s http://localhost:$QDRANT_PORT/health | grep -q '\"status\":\"ok\"'"
test::run_test "Issues collection exists" "curl -s http://localhost:$QDRANT_PORT/collections/issues | grep -q '\"status\":\"ok\"'"

# Test 5: Windmill Components
echo ""
echo "5. Windmill Component Tests"
echo "---------------------------"
test::run_test "Windmill API accessible" "curl -s -o /dev/null -w '%{http_code}' http://localhost:$WINDMILL_PORT/api/version | grep -q '200'"
test::run_test "Issue tracker workspace" "curl -s http://localhost:$WINDMILL_PORT/api/workspaces | grep -q 'issue_tracker'"

# Test 6: n8n Workflows
echo ""
echo "6. n8n Workflow Tests"
echo "--------------------"
test::run_test "n8n health check" "curl -s http://localhost:$N8N_PORT/healthz | grep -q '\"status\":\"ok\"'"

# Test 7: Ollama Models
echo ""
echo "7. AI Model Tests"
echo "-----------------"
test::run_test "Ollama API accessible" "curl -s http://localhost:$OLLAMA_PORT/api/tags | grep -q 'models'"
test::run_test "Embedding model available" "curl -s http://localhost:$OLLAMA_PORT/api/tags | grep -q 'nomic-embed-text'"

# Test 8: CLI Tool
echo ""
echo "8. CLI Tool Tests"
echo "-----------------"
CLI_PATH="${SCENARIO_DIR}/cli/vrooli-tracker.sh"
test::run_test "CLI script exists" "[ -f $CLI_PATH ]"
test::run_test "CLI is executable" "[ -x $CLI_PATH ]"
test::run_test "CLI help command" "$CLI_PATH --help | grep -q 'Vrooli Issue Tracker CLI'"

# Test 9: API Token Generation
echo ""
echo "9. API Token Tests"
echo "------------------"
API_TOKEN=$(PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -t -c "SELECT api_token FROM apps WHERE name = 'vrooli-core' LIMIT 1;" 2>/dev/null | tr -d ' ')
test::run_test "API token generated" "[ -n \"$API_TOKEN\" ]"
test::run_test "API token format" "echo \"$API_TOKEN\" | grep -q '^tk_'"

# Test 10: Create Test Issue via SQL
echo ""
echo "10. Issue Creation Tests"
echo "------------------------"
TEST_ISSUE_ID=$(PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -t -c "INSERT INTO issues (app_id, title, description, type, priority) VALUES ((SELECT id FROM apps WHERE name = 'vrooli-core'), 'Test Issue', 'Integration test issue', 'bug', 'low') RETURNING id;" 2>/dev/null | tr -d ' ')
test::run_test "Create test issue" "[ -n \"$TEST_ISSUE_ID\" ]"
test::run_test "Verify issue created" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -t -c \"SELECT COUNT(*) FROM issues WHERE id = '$TEST_ISSUE_ID';\" | grep -q '1'"

# Test 11: Pattern Detection
echo ""
echo "11. Pattern Detection Tests"
echo "---------------------------"
test::run_test "Issue metrics view" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -c 'SELECT * FROM issue_metrics LIMIT 1;'"
test::run_test "Agent performance view" "PGPASSWORD=postgres psql -h localhost -p $POSTGRES_PORT -U postgres -d issue_tracker -c 'SELECT * FROM agent_performance LIMIT 1;'"

# Test 12: Redis Event Bus
echo ""
echo "12. Event Bus Tests"
echo "-------------------"
test::run_test "Redis ping" "redis-cli -p $REDIS_PORT ping | grep -q PONG"
test::run_test "Set test key" "redis-cli -p $REDIS_PORT SET issue_tracker:test:key 'test_value' | grep -q OK"
test::run_test "Get test key" "redis-cli -p $REDIS_PORT GET issue_tracker:test:key | grep -q 'test_value'"

# Summary
log::success ""
log::success "=========================================="
log::info "Test Summary"
log::success "=========================================="
log::success "Tests Passed: $TESTS_PASSED"
log::error "Tests Failed: $TESTS_FAILED"

if [ $TESTS_FAILED -eq 0 ]; then
    log::success ""
    log::success "✅ All tests passed!"
    log::success ""
    log::info "The Issue Tracker scenario is fully operational."
    log::info "You can now:"
    log::info "  1. Access the dashboard at http://localhost:$WINDMILL_PORT/apps/issue_tracker_dashboard"
    log::info "  2. Use the CLI: $CLI_PATH create --title 'Your issue'"
    log::info "  3. View metrics at http://localhost:$WINDMILL_PORT/apps/issue_tracker_metrics"
    exit 0
else
    log::success ""
    log::error "❌ Some tests failed"
    log::info "Please check the failed components and run the startup script:"
    log::info "  ${SCENARIO_DIR}/deployment/startup.sh"
    exit 1
fi