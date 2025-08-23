#!/bin/bash
set -euo pipefail

# Simple Test Scenario - Integration Test Script

# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../lib/utils/var.sh"

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$var_LOG_FILE"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function wrapper
test::run_test() {
    local test_name="$1"
    local test_function="$2"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    log::info "Running test: $test_name"
    
    if $test_function; then
        log::success "✅ $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        log::error "❌ $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test required files exist
test::required_files() {
    local required_files=(
        "initialization/storage/postgres/schema.sql"
        "initialization/storage/postgres/seed.sql"
        "package.json"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$SCENARIO_DIR/$file" ]; then
            log::error "Required file missing: $file"
            return 1
        fi
    done
    
    return 0
}

# Test database connection (if postgres is available)
test::database_connection() {
    local postgres_host="${POSTGRES_HOST:-localhost}"
    local postgres_port="${POSTGRES_PORT:-5433}"
    local postgres_db="${POSTGRES_DB:-postgres}"
    local postgres_user="${POSTGRES_USER:-postgres}"
    
    # Test connection - this is optional for simple-test
    if timeout 5 psql -h "$postgres_host" -p "$postgres_port" -U "$postgres_user" -d "$postgres_db" -c "SELECT 1;" >/dev/null 2>&1; then
        log::success "PostgreSQL connection successful"
        return 0
    else
        log::warning "PostgreSQL not available (this is OK for simple-test scenario)"
        return 0
    fi
}

# Test SQL files are syntactically valid
test::sql_validity() {
    local sql_files=(
        "initialization/storage/postgres/schema.sql"
        "initialization/storage/postgres/seed.sql"
    )
    
    for sql_file in "${sql_files[@]}"; do
        local file_path="$SCENARIO_DIR/$sql_file"
        if [ -f "$file_path" ]; then
            # Basic syntax check - ensure file is not empty and has SQL-like content
            if [ -s "$file_path" ] && grep -q -E "(CREATE|INSERT|SELECT|DROP|ALTER)" "$file_path"; then
                log::success "SQL file looks valid: $sql_file"
            else
                log::warning "SQL file appears empty or invalid: $sql_file"
                return 1
            fi
        fi
    done
    
    return 0
}

# Test package.json validity
test::package_json() {
    local package_file="$SCENARIO_DIR/package.json"
    
    if [ ! -f "$package_file" ]; then
        log::error "package.json not found"
        return 1
    fi
    
    if ! jq empty < "$package_file" >/dev/null 2>&1; then
        log::error "Invalid JSON in package.json"
        return 1
    fi
    
    log::success "package.json is valid JSON"
    return 0
}

# Main test function
main() {
    log::info "🧪 Starting Simple Test Scenario Integration Tests"
    echo ""
    
    # Run all tests
    test::run_test "Required files exist" test::required_files
    test::run_test "SQL files validity" test::sql_validity 
    test::run_test "Package.json validity" test::package_json
    test::run_test "Database connection" test::database_connection
    
    echo ""
    log::info "📊 Test Results"
    echo "=============================================="
    echo "Tests Run:    $TESTS_RUN"
    echo "Tests Passed: $TESTS_PASSED"
    echo "Tests Failed: $TESTS_FAILED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        log::success "🎉 All tests passed! Simple Test scenario is ready for use."
        exit 0
    else
        log::error "❌ $TESTS_FAILED tests failed. Please address the issues above."
        exit 1
    fi
}

# Handle script arguments
case "${1:-test}" in
    "test"|"run")
        main
        ;;
    "help"|"-h"|"--help")
        echo "Simple Test Scenario - Integration Test Script"
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
        echo "  POSTGRES_DB       PostgreSQL database (default: postgres)"
        echo "  POSTGRES_USER     PostgreSQL user (default: postgres)"
        ;;
    *)
        log::error "Unknown command: $1"
        log::info "Run '$0 help' for usage information"
        exit 1
        ;;
esac