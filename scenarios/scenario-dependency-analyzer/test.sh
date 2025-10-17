#!/bin/bash

# Scenario Dependency Analyzer Test Script
# Validates core functionality and integration points

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test configuration
API_PORT="${DEPENDENCY_ANALYZER_API_PORT:-20400}"
API_BASE_URL="http://localhost:${API_PORT}"
CLI_COMMAND="./cli/scenario-dependency-analyzer"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_exit_code="${3:-0}"
    
    ((TESTS_RUN++))
    
    echo
    log_info "Running test: $test_name"
    
    if eval "$test_command" >/dev/null 2>&1; then
        local exit_code=0
    else
        local exit_code=$?
    fi
    
    if [[ $exit_code -eq $expected_exit_code ]]; then
        log_success "$test_name"
        return 0
    else
        log_error "$test_name (expected exit code $expected_exit_code, got $exit_code)"
        return 1
    fi
}

run_api_test() {
    local test_name="$1"
    local endpoint="$2"
    local method="${3:-GET}"
    local expected_status="${4:-200}"
    
    ((TESTS_RUN++))
    
    echo
    log_info "Running API test: $test_name"
    
    local status_code
    if [[ "$method" == "GET" ]]; then
        status_code=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL$endpoint")
    elif [[ "$method" == "POST" ]]; then
        status_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d '{"name":"test","description":"test scenario","requirements":["postgres"]}')
    fi
    
    if [[ "$status_code" == "$expected_status" ]]; then
        log_success "$test_name (HTTP $status_code)"
        return 0
    else
        log_error "$test_name (expected HTTP $expected_status, got $status_code)"
        return 1
    fi
}

# Main test execution
main() {
    echo "======================================"
    echo "Scenario Dependency Analyzer Test Suite"
    echo "======================================"
    
    # Check prerequisites
    log_info "Checking prerequisites..."
    
    if [[ ! -f "$CLI_COMMAND" ]]; then
        log_error "CLI not found at $CLI_COMMAND"
        exit 1
    fi
    
    if ! command -v curl >/dev/null; then
        log_error "curl not found - required for API tests"
        exit 1
    fi
    
    # Wait for API to be ready
    log_info "Waiting for API to be ready at $API_BASE_URL..."
    for i in {1..30}; do
        if curl -sf "$API_BASE_URL/health" >/dev/null 2>&1; then
            log_success "API is ready"
            break
        fi
        if [[ $i -eq 30 ]]; then
            log_error "API failed to start within 30 seconds"
            exit 1
        fi
        sleep 1
    done
    
    # Basic functionality tests
    echo
    echo "=== BASIC FUNCTIONALITY TESTS ==="
    
    run_test "CLI help command" "$CLI_COMMAND help"
    run_test "CLI version command" "$CLI_COMMAND version"
    run_test "CLI status command" "$CLI_COMMAND status --json"
    
    # API health tests
    echo
    echo "=== API HEALTH TESTS ==="
    
    run_api_test "API health endpoint" "/health" "GET" "200"
    run_api_test "API analysis health check" "/api/v1/health/analysis" "GET" "200"
    
    # Core analysis tests
    echo
    echo "=== CORE ANALYSIS TESTS ==="
    
    # Test analyzing this scenario itself (meta-analysis)
    run_test "CLI self-analysis" "$CLI_COMMAND analyze scenario-dependency-analyzer --output json"
    
    # Test graph generation
    run_test "CLI graph generation" "$CLI_COMMAND graph resources --format json"
    run_test "CLI combined graph" "$CLI_COMMAND graph combined --format json"
    
    # API endpoint tests
    echo
    echo "=== API ENDPOINT TESTS ==="
    
    run_api_test "Scenario dependencies endpoint" "/api/v1/scenarios/scenario-dependency-analyzer/dependencies" "GET" "200"
    run_api_test "Graph generation endpoint" "/api/v1/graph/combined" "GET" "200"
    run_api_test "Resource graph endpoint" "/api/v1/graph/resources" "GET" "200"
    run_api_test "Proposed scenario analysis" "/api/v1/analyze/proposed" "POST" "200"
    
    # Database connectivity tests
    echo
    echo "=== DATABASE TESTS ==="
    
    if command -v resource-postgres >/dev/null; then
        run_test "Database schema check" "resource-postgres execute 'SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN (\"scenario_dependencies\", \"dependency_graphs\");' | grep -q '2'"
        run_test "Database seed data check" "resource-postgres execute 'SELECT COUNT(*) FROM scenario_dependencies WHERE scenario_name = \"scenario-dependency-analyzer\";' | grep -q '[1-9]'"
    else
        log_warning "Skipping database tests - resource-postgres not available"
    fi
    
    # Integration tests
    echo
    echo "=== INTEGRATION TESTS ==="
    
    # Test analysis of a common scenario if it exists
    if [[ -d "../chart-generator" ]]; then
        run_test "Chart generator analysis" "$CLI_COMMAND analyze chart-generator --output json"
    else
        log_warning "Skipping chart-generator analysis - scenario not found"
    fi
    
    # Test proposed scenario analysis
    run_test "Proposed scenario analysis via CLI" "$CLI_COMMAND propose 'AI-powered data analysis tool with PostgreSQL storage and Redis caching'"
    
    # Performance tests
    echo
    echo "=== PERFORMANCE TESTS ==="
    
    # Test that analysis completes within reasonable time (30 seconds)
    run_test "Analysis performance test" "timeout 30 $CLI_COMMAND analyze scenario-dependency-analyzer"
    
    # Test that graph generation is reasonably fast (10 seconds)
    run_test "Graph generation performance" "timeout 10 $CLI_COMMAND graph combined --format json"
    
    # UI accessibility tests
    echo
    echo "=== UI TESTS ==="
    
    if curl -sf "http://localhost:${DEPENDENCY_ANALYZER_UI_PORT:-20401}" >/dev/null 2>&1; then
        run_api_test "UI health check" "/" "GET" "200" 
    else
        log_warning "Skipping UI tests - UI server not accessible"
    fi
    
    # Resource integration tests
    echo
    echo "=== RESOURCE INTEGRATION TESTS ==="
    
    if command -v resource-claude-code >/dev/null; then
        log_info "Claude Code resource available - integration should work"
    else
        log_warning "Claude Code resource not available - proposed scenario analysis will use fallbacks"
    fi
    
    if command -v resource-qdrant >/dev/null; then
        log_info "Qdrant resource available - similarity matching should work"
    else
        log_warning "Qdrant resource not available - similarity matching will use fallbacks"
    fi
    
    # Summary
    echo
    echo "======================================"
    echo "TEST SUMMARY"
    echo "======================================"
    echo "Tests Run:    $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}ALL TESTS PASSED!${NC}"
        echo
        echo "✅ Scenario Dependency Analyzer is working correctly"
        echo
        echo "Quick verification commands:"
        echo "  $CLI_COMMAND status"
        echo "  $CLI_COMMAND analyze scenario-dependency-analyzer"
        echo "  curl $API_BASE_URL/health"
        echo "  Open http://localhost:${DEPENDENCY_ANALYZER_UI_PORT:-20401} in browser"
        
        return 0
    else
        echo -e "${RED}SOME TESTS FAILED!${NC}"
        echo
        echo "❌ Please check the failed tests and ensure:"
        echo "  - All required resources are running (postgres, claude-code, qdrant)"
        echo "  - API server started successfully"
        echo "  - Database schema is initialized"
        echo "  - CLI is installed correctly"
        
        return 1
    fi
}

# Trap to ensure we always show summary
trap 'echo; echo "Test execution interrupted"; exit 1' INT TERM

# Run main test function
main "$@"