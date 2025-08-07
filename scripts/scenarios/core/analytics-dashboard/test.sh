#!/bin/bash
# Resource Monitoring Platform - Integration Tests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"; }
test_count=0
passed_count=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    ((test_count++))
    log "TEST ${test_count}: ${test_name}"
    
    if eval "$test_command" >/dev/null 2>&1; then
        log "✓ PASS: ${test_name}"
        ((passed_count++))
        return 0
    else
        log "✗ FAIL: ${test_name}"
        return 1
    fi
}

echo "=========================================="
echo "Resource Monitoring Platform - Tests"
echo "$(date)"
echo "=========================================="

# Test 1: Core services availability
run_test "PostgreSQL connectivity" \
    'PGPASSWORD="${POSTGRES_PASSWORD:-vrooli}" psql -h localhost -p 5433 -U "${POSTGRES_USER:-vrooli}" -d "${POSTGRES_DB:-vrooli}" -c "SELECT 1" >/dev/null'

run_test "Redis connectivity" \
    'redis-cli -p 6380 ping | grep -q PONG'

run_test "QuestDB connectivity" \
    'curl -s "http://localhost:9009/status" | grep -q "QuestDB"'

run_test "Vault connectivity" \
    'curl -s "http://localhost:8200/v1/sys/health" | grep -q "initialized"'

run_test "n8n connectivity" \
    'curl -s "http://localhost:5678/healthz" | grep -q "ok"'

run_test "Windmill connectivity" \
    'curl -s "http://localhost:5681/api/version"'

# Test 2: Database schema
run_test "Resource monitoring schema exists" \
    'PGPASSWORD="${POSTGRES_PASSWORD:-vrooli}" psql -h localhost -p 5433 -U "${POSTGRES_USER:-vrooli}" -d "${POSTGRES_DB:-vrooli}" -c "SELECT 1 FROM resource_monitoring.resources LIMIT 1"'

run_test "QuestDB tables exist" \
    'curl -s "http://localhost:9009/exec?query=SELECT%20COUNT(*)%20FROM%20resource_metrics" | grep -q dataset'

# Test 3: API endpoints
run_test "Resource discovery API" \
    'curl -s -X POST "http://localhost:5678/webhook/discover-resources"'

run_test "Alert handler API" \
    'curl -s -X POST -H "Content-Type: application/json" -d "{\"resource_name\":\"test\",\"severity\":\"info\",\"message\":\"test\"}" "http://localhost:5678/webhook/alert-handler"'

# Test 4: Configuration files
run_test "Monitor config exists" \
    'test -f "${SCRIPT_DIR}/initialization/configuration/monitor-config.json"'

run_test "Resource registry config exists" \
    'test -f "${SCRIPT_DIR}/initialization/configuration/resource-registry.json"'

run_test "Alert rules config exists" \
    'test -f "${SCRIPT_DIR}/initialization/configuration/alert-rules.json"'

# Test 5: Workflows
run_test "n8n resource monitor workflow exists" \
    'test -f "${SCRIPT_DIR}/initialization/automation/n8n/resource-monitor.json"'

run_test "Node-RED metrics collector exists" \
    'test -f "${SCRIPT_DIR}/initialization/automation/node-red/metrics-collector.json"'

# Test 6: Deployment scripts
run_test "Startup script exists and executable" \
    'test -x "${SCRIPT_DIR}/deployment/startup.sh"'

run_test "Monitor script exists and executable" \
    'test -x "${SCRIPT_DIR}/deployment/monitor.sh"'

# Test 7: Functional tests
run_test "Resource discovery functionality" \
    "${SCRIPT_DIR}/deployment/discover.sh"

run_test "System health check" \
    "${SCRIPT_DIR}/deployment/monitor.sh"

echo ""
echo "=========================================="
echo "Test Results: ${passed_count}/${test_count} passed"
echo "=========================================="

if [ $passed_count -eq $test_count ]; then
    echo "🎉 All tests passed\!"
    exit 0
else
    echo "❌ $((test_count - passed_count)) test(s) failed"
    exit 1
fi
