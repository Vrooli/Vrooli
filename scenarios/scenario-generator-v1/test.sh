#!/usr/bin/env bash
# Scenario Generator V1 - Integration Test Script

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/scenario-generator-v1"
API_PORT="${SERVICE_PORT:-8080}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"

# Colors
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
CYAN='\033[1;36m'
NC='\033[0m'

info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

test_api_health() {
    info "Testing API health endpoint..."
    
    if curl -sf "http://localhost:$API_PORT/health" > /dev/null; then
        success "API health check passed"
        return 0
    else
        error "API health check failed"
        return 1
    fi
}

test_database_connection() {
    info "Testing database connection..."
    
    if pg_isready -h localhost -p "$POSTGRES_PORT" -U postgres &> /dev/null; then
        success "Database connection test passed"
        return 0
    else
        error "Database connection test failed"
        return 1
    fi
}

test_api_endpoints() {
    info "Testing API endpoints..."
    
    local base_url="http://localhost:$API_PORT/api"
    local failures=0
    
    # Test scenarios endpoint
    if curl -sf "$base_url/scenarios" > /dev/null; then
        success "GET /api/scenarios works"
    else
        error "GET /api/scenarios failed"
        failures=$((failures + 1))
    fi
    
    # Test templates endpoint
    if curl -sf "$base_url/templates" > /dev/null; then
        success "GET /api/templates works"
    else
        error "GET /api/templates failed"
        failures=$((failures + 1))
    fi
    
    # Test search endpoint
    if curl -sf "$base_url/search/scenarios?q=test" > /dev/null; then
        success "GET /api/search/scenarios works"
    else
        error "GET /api/search/scenarios failed"
        failures=$((failures + 1))
    fi
    
    if [ $failures -eq 0 ]; then
        success "All API endpoint tests passed"
        return 0
    else
        error "$failures API endpoint tests failed"
        return 1
    fi
}

test_cli_availability() {
    info "Testing CLI availability..."
    
    if command -v scenario-generator &> /dev/null; then
        success "CLI is installed and available"
        
        # Test CLI help command
        if scenario-generator help > /dev/null 2>&1; then
            success "CLI help command works"
        else
            warn "CLI help command failed"
        fi
        
        return 0
    else
        warn "CLI is not installed globally"
        return 1
    fi
}

test_claude_code() {
    info "Testing Claude Code availability..."
    
    if command -v claude &> /dev/null; then
        success "Claude Code CLI is available"
        return 0
    else
        warn "Claude Code CLI is not available - generation features will be limited"
        return 1
    fi
}

test_database_schema() {
    info "Testing database schema..."
    
    local postgres_url="postgres://postgres:postgres@localhost:$POSTGRES_PORT/postgres?sslmode=disable"
    
    # Test if main tables exist
    local tables=("scenarios" "scenario_templates" "generation_logs")
    local failures=0
    
    for table in "${tables[@]}"; do
        if psql "$postgres_url" -c "SELECT 1 FROM $table LIMIT 1;" > /dev/null 2>&1; then
            success "Table '$table' exists and is accessible"
        else
            error "Table '$table' is missing or inaccessible"
            failures=$((failures + 1))
        fi
    done
    
    if [ $failures -eq 0 ]; then
        success "Database schema test passed"
        return 0
    else
        error "$failures database schema tests failed"
        return 1
    fi
}

test_scenario_creation() {
    info "Testing scenario creation via API..."
    
    local api_url="http://localhost:$API_PORT/api/scenarios"
    
    # Create test scenario
    local test_data='{
        "name": "Test Scenario",
        "description": "A test scenario for validation",
        "prompt": "Create a simple test scenario",
        "complexity": "simple",
        "category": "test"
    }'
    
    local response
    if response=$(curl -sf -X POST -H "Content-Type: application/json" -d "$test_data" "$api_url"); then
        success "Scenario creation test passed"
        
        # Extract scenario ID for cleanup
        local scenario_id
        scenario_id=$(echo "$response" | jq -r '.id' 2>/dev/null || echo "")
        
        if [[ -n "$scenario_id" && "$scenario_id" != "null" ]]; then
            info "Created test scenario with ID: $scenario_id"
            success "Scenario creation response is valid"
        else
            warn "Scenario created but response format is unexpected"
        fi
        
        return 0
    else
        error "Scenario creation test failed"
        return 1
    fi
}

run_all_tests() {
    info "🧪 Running Scenario Generator V1 Integration Tests..."
    echo
    
    local total_tests=0
    local passed_tests=0
    
    # API Health Test
    total_tests=$((total_tests + 1))
    if test_api_health; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # Database Connection Test
    total_tests=$((total_tests + 1))
    if test_database_connection; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # Database Schema Test
    total_tests=$((total_tests + 1))
    if test_database_schema; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # API Endpoints Test
    total_tests=$((total_tests + 1))
    if test_api_endpoints; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # Scenario Creation Test
    total_tests=$((total_tests + 1))
    if test_scenario_creation; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # CLI Availability Test (optional)
    total_tests=$((total_tests + 1))
    if test_cli_availability; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # Claude Code Test (optional)
    total_tests=$((total_tests + 1))
    if test_claude_code; then
        passed_tests=$((passed_tests + 1))
    fi
    echo
    
    # Summary
    echo "=========================================="
    info "Test Summary: $passed_tests/$total_tests tests passed"
    
    if [ $passed_tests -eq $total_tests ]; then
        success "🎉 All tests passed! Scenario Generator V1 is working correctly."
        return 0
    elif [ $passed_tests -ge $((total_tests - 2)) ]; then
        warn "⚠️  Most tests passed. Some optional features may not be available."
        return 0
    else
        error "❌ Multiple tests failed. Please check your setup."
        return 1
    fi
}

# Check if jq is available for JSON parsing
if ! command -v jq &> /dev/null; then
    warn "jq is not installed. Some tests may not work properly."
fi

# Main test execution
case "${1:-all}" in
    "all")
        run_all_tests
        ;;
    "api")
        test_api_health && test_api_endpoints
        ;;
    "db"|"database")
        test_database_connection && test_database_schema
        ;;
    "cli")
        test_cli_availability
        ;;
    "claude")
        test_claude_code
        ;;
    "create")
        test_scenario_creation
        ;;
    "help")
        echo "Usage: $0 [test-type]"
        echo
        echo "Test types:"
        echo "  all       - Run all tests (default)"
        echo "  api       - Test API health and endpoints"
        echo "  database  - Test database connection and schema"
        echo "  cli       - Test CLI availability"
        echo "  claude    - Test Claude Code availability"
        echo "  create    - Test scenario creation"
        echo "  help      - Show this help message"
        ;;
    *)
        error "Unknown test type: $1. Use 'help' for available options."
        exit 1
        ;;
esac
