#!/bin/bash
set -euo pipefail

# Scenario Generator V1 - Integration Test Script

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    log_info "Running test: $test_name"
    
    if $test_function; then
        log_success "✅ $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log_error "❌ $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test database connection and schema
test_database_schema() {
    local postgres_host="${POSTGRES_HOST:-localhost}"
    local postgres_port="${POSTGRES_PORT:-5433}"
    local postgres_db="${POSTGRES_DB:-vrooli}"
    local postgres_user="${POSTGRES_USER:-postgres}"
    
    # Test connection
    if \! timeout 10 psql -h "$postgres_host" -p "$postgres_port" -U "$postgres_user" -d "$postgres_db" -c "SELECT 1;" >/dev/null 2>&1; then
        log_error "Cannot connect to PostgreSQL database"
        return 1
    fi
    
    # Test required tables exist
    local required_tables=("campaigns" "scenarios" "claude_interactions" "validation_results" "improvement_patterns")
    
    for table in "${required_tables[@]}"; do
        if \! psql -h "$postgres_host" -p "$postgres_port" -U "$postgres_user" -d "$postgres_db" -c "SELECT 1 FROM $table LIMIT 1;" >/dev/null 2>&1; then
            log_error "Required table '$table' does not exist or is not accessible"
            return 1
        fi
    done
    
    return 0
}

# Test required files exist
test_scenario_files() {
    local required_files=(
        "initialization/storage/schema.sql"
        "initialization/storage/seed.sql" 
        "initialization/automation/n8n/main-workflow.json"
        "initialization/automation/n8n/planning-workflow.json"
        "initialization/automation/n8n/building-workflow.json"
        "initialization/automation/n8n/validation-workflow.json"
        "initialization/automation/windmill/scenario-dashboard-app.json"
        "prompts/initial-planning-prompt.md"
        "prompts/plan-refinement-prompt.md"
        "prompts/implementation-prompt.md"
        "prompts/validation-prompt.md"
        "scenario-test.yaml"
        "README.md"
    )
    
    for file in "${required_files[@]}"; do
        if [ \! -f "$SCRIPT_DIR/$file" ]; then
            log_error "Required file missing: $file"
            return 1
        fi
    done
    
    return 0
}

# Test JSON files are valid
test_json_validity() {
    local json_files=(
        "initialization/automation/n8n/main-workflow.json"
        "initialization/automation/n8n/planning-workflow.json"
        "initialization/automation/n8n/building-workflow.json"
        "initialization/automation/n8n/validation-workflow.json"
        "initialization/automation/windmill/scenario-dashboard-app.json"
    )
    
    for json_file in "${json_files[@]}"; do
        local file_path="$SCRIPT_DIR/$json_file"
        if [ -f "$file_path" ]; then
            if \! jq empty < "$file_path" >/dev/null 2>&1; then
                log_error "Invalid JSON in file: $json_file"
                return 1
            fi
        fi
    done
    
    return 0
}

# Test SQL files are syntactically valid
test_sql_validity() {
    local postgres_host="${POSTGRES_HOST:-localhost}"
    local postgres_port="${POSTGRES_PORT:-5433}"
    local postgres_db="${POSTGRES_DB:-vrooli}"
    local postgres_user="${POSTGRES_USER:-postgres}"
    
    local sql_files=(
        "initialization/storage/schema.sql"
        "initialization/storage/seed.sql"
    )
    
    for sql_file in "${sql_files[@]}"; do
        local file_path="$SCRIPT_DIR/$sql_file"
        if [ -f "$file_path" ]; then
            # Test SQL syntax by doing a dry run
            if \! timeout 30 psql -h "$postgres_host" -p "$postgres_port" -U "$postgres_user" -d "$postgres_db" --set ON_ERROR_STOP=1 --set AUTOCOMMIT=off -f "$file_path" -v ON_ERROR_STOP=1 >/dev/null 2>&1; then
                log_error "SQL syntax error or execution error in: $sql_file"
                return 1
            fi
        fi
    done
    
    return 0
}

# Test Claude Code integration
test_claude_code_integration() {
    local claude_script="${VROOLI_ROOT:-/home/matthalloran8/Vrooli}/scripts/resources/agents/claude-code/manage.sh"
    
    if [ \! -f "$claude_script" ]; then
        log_error "Claude Code management script not found"
        return 1
    fi
    
    # Test basic health check
    if \! bash "$claude_script" --action health-check --check-type basic --format json >/dev/null 2>&1; then
        log_warn "Claude Code health check failed (may need authentication)"
        return 1
    fi
    
    return 0
}

# Test scenario-test.yaml configuration
test_scenario_configuration() {
    local config_file="$SCRIPT_DIR/scenario-test.yaml"
    
    if [ \! -f "$config_file" ]; then
        log_error "scenario-test.yaml not found"
        return 1
    fi
    
    # Check required resources are declared
    local required_resources=("windmill" "n8n" "postgres" "minio" "redis" "claude-code")
    
    for resource in "${required_resources[@]}"; do
        if \! grep -q "$resource" "$config_file"; then
            log_error "Required resource '$resource' not found in scenario-test.yaml"
            return 1
        fi
    done
    
    return 0
}

# Test deployment script
test_deployment_script() {
    local deploy_script="$SCRIPT_DIR/deployment/startup.sh"
    
    if [ \! -f "$deploy_script" ]; then
        log_error "Deployment script not found"
        return 1
    fi
    
    if [ \! -x "$deploy_script" ]; then
        log_error "Deployment script is not executable"
        return 1
    fi
    
    # Test help command works
    if \! "$deploy_script" help >/dev/null 2>&1; then
        log_error "Deployment script help command failed"
        return 1
    fi
    
    return 0
}

# Test sample data integrity
test_sample_data() {
    local postgres_host="${POSTGRES_HOST:-localhost}"
    local postgres_port="${POSTGRES_PORT:-5433}"
    local postgres_db="${POSTGRES_DB:-vrooli}"
    local postgres_user="${POSTGRES_USER:-postgres}"
    
    # Check if sample campaigns exist
    local campaign_count
    campaign_count=$(psql -h "$postgres_host" -p "$postgres_port" -U "$postgres_user" -d "$postgres_db" -t -c "SELECT COUNT(*) FROM campaigns;" 2>/dev/null | xargs)
    
    if [ "$campaign_count" -lt 1 ]; then
        log_error "No sample campaigns found in database"
        return 1
    fi
    
    return 0
}

# Main test function
main() {
    log_info "🧪 Starting Scenario Generator V1 Integration Tests"
    echo ""
    
    # Run all tests
    run_test "Required files exist" test_scenario_files
    run_test "JSON files are valid" test_json_validity
    run_test "Scenario configuration" test_scenario_configuration
    run_test "Deployment script" test_deployment_script
    run_test "Database schema" test_database_schema
    run_test "SQL files validity" test_sql_validity
    run_test "Sample data integrity" test_sample_data
    run_test "Claude Code integration" test_claude_code_integration
    
    echo ""
    log_info "📊 Test Results"
    echo "=============================================="
    echo "Tests Run:    $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log_success "🎉 All tests passed\! Scenario Generator V1 is ready for use."
        exit 0
    else
        log_error "❌ $TESTS_FAILED tests failed. Please address the issues above."
        exit 1
    fi
}

# Handle script arguments
case "${1:-test}" in
    "test"|"run")
        main
        ;;
    "help"|"-h"|"--help")
        echo "Scenario Generator V1 - Integration Test Script"
        echo ""
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  test, run     Run all integration tests (default)"
        echo "  help          Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  POSTGRES_HOST     PostgreSQL host (default: localhost)"
        echo "  POSTGRES_PORT     PostgreSQL port (default: 5433)"
        echo "  POSTGRES_DB       PostgreSQL database (default: vrooli)"
        echo "  POSTGRES_USER     PostgreSQL user (default: postgres)"
        ;;
    *)
        log_error "Unknown command: $1"
        log_info "Run '$0 help' for usage information"
        exit 1
        ;;
esac
TEST_EOF < /dev/null
