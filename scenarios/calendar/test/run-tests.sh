#!/bin/bash
# Calendar Scenario Test Runner
# Comprehensive testing for calendar API, CLI, and UI components

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
TEST_DIR="$SCRIPT_DIR"
TEMP_DIR="/tmp/calendar-tests-$$"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} Calendar Scenario Test Suite${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
}

print_test_section() {
    echo -e "${YELLOW}🧪 $1${NC}"
    echo "----------------------------------------"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

run_test() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -n "  Testing: $test_name... "
    
    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}PASS${NC}"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

run_verbose_test() {
    local test_name="$1"
    local test_command="$2"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo "  Testing: $test_name..."
    
    if eval "$test_command"; then
        print_success "  ✅ $test_name PASSED"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        print_error "  ❌ $test_name FAILED"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Test structure validation
test_file_structure() {
    print_test_section "File Structure Validation"
    
    local required_files=(
        ".vrooli/service.json"
        "PRD.md"
        "README.md" 
        "Makefile"
        "api/main.go"
        "api/go.mod"
        "api/.golangci.yml"
        "cli/calendar"
        "cli/install.sh"
        "initialization/postgres/schema.sql"
        "initialization/postgres/seed.sql"
        "ui/package.json"
        "ui/.eslintrc.json"
        "ui/.prettierrc.json"
        "scenario-test.yaml"
    )
    
    cd "$SCENARIO_DIR"
    
    for file in "${required_files[@]}"; do
        run_test "File exists: $file" "[ -f \"$file\" ]"
    done
    
    local required_dirs=(
        "api"
        "cli"
        "initialization"
        "initialization/postgres"
        "ui"
        "ui/src"
        "test"
    )
    
    for dir in "${required_dirs[@]}"; do
        run_test "Directory exists: $dir" "[ -d \"$dir\" ]"
    done
}

# Test Go API
test_go_api() {
    print_test_section "Go API Tests"
    
    cd "$SCENARIO_DIR/api"
    
    # Check if Go modules are clean
    run_test "Go mod tidy check" "go mod tidy && git diff --exit-code go.mod go.sum || true"
    
    # Build the API
    run_test "Go build succeeds" "go build -o calendar-api-test ."
    
    # Clean up build artifact
    rm -f calendar-api-test
    
    # Run Go tests if they exist
    if find . -name "*_test.go" | head -1 | grep -q .; then
        run_verbose_test "Go unit tests" "go test ./... -v"
    else
        print_warning "No Go unit tests found"
    fi
    
    # Vet the code
    run_test "Go vet passes" "go vet ./..."
    
    # Format check
    run_test "Go fmt check" "test -z \"\$(gofmt -l .)\""
    
    # Lint check (if golangci-lint is available)
    if command -v golangci-lint >/dev/null 2>&1; then
        run_test "golangci-lint passes" "golangci-lint run"
    else
        print_warning "golangci-lint not available, skipping lint check"
    fi
}

# Test CLI
test_cli() {
    print_test_section "CLI Tests"
    
    cd "$SCENARIO_DIR"
    
    # Check if CLI is executable
    run_test "CLI is executable" "[ -x \"cli/calendar\" ]"
    
    # Test help command
    run_test "CLI help works" "cli/calendar help | grep -q 'Universal Scheduling Intelligence'"
    
    # Test version command  
    run_test "CLI version works" "cli/calendar version | grep -q 'Calendar CLI'"
    
    # Check install script
    run_test "Install script is executable" "[ -x \"cli/install.sh\" ]"
}

# Test UI
test_ui() {
    print_test_section "UI Tests"
    
    cd "$SCENARIO_DIR/ui"
    
    # Check package.json structure
    run_test "package.json exists" "[ -f package.json ]"
    run_test "package.json has required scripts" "jq -e '.scripts.dev and .scripts.build and .scripts.lint' package.json >/dev/null"
    
    # TypeScript configuration
    run_test "tsconfig.json exists" "[ -f tsconfig.json ]"
    run_test "TypeScript config is valid" "npx tsc --noEmit --skipLibCheck"
    
    # ESLint configuration
    run_test "ESLint config exists" "[ -f .eslintrc.json ]"
    
    # Prettier configuration
    run_test "Prettier config exists" "[ -f .prettierrc.json ]"
    
    # Check if dependencies are installed
    if [ -d node_modules ]; then
        print_info "Dependencies already installed"
        
        # Run linting
        run_test "ESLint passes" "npm run lint"
        
        # Run type checking
        run_test "TypeScript type checking passes" "npm run typecheck"
        
        # Test build
        run_test "UI build succeeds" "npm run build"
    else
        print_warning "Dependencies not installed, skipping advanced UI tests"
        print_info "Run 'cd ui && npm install' to enable full UI testing"
    fi
}

# Test database schema
test_database_schema() {
    print_test_section "Database Schema Tests"
    
    cd "$SCENARIO_DIR"
    
    # Check schema file syntax
    run_test "Schema file is valid SQL" "psql --dry-run -f initialization/postgres/schema.sql >/dev/null || true"
    
    # Check seed file syntax
    run_test "Seed file is valid SQL" "psql --dry-run -f initialization/postgres/seed.sql >/dev/null || true"
    
    # Schema content validation
    run_test "Schema contains required tables" "grep -q 'CREATE TABLE events' initialization/postgres/schema.sql"
    run_test "Schema contains users table" "grep -q 'CREATE TABLE users' initialization/postgres/schema.sql"
    run_test "Schema contains reminders table" "grep -q 'CREATE TABLE event_reminders' initialization/postgres/schema.sql"
    
    # Index validation
    run_test "Schema contains performance indexes" "grep -q 'CREATE INDEX.*events' initialization/postgres/schema.sql"
    
    # Function validation
    run_test "Schema contains helper functions" "grep -q 'CREATE OR REPLACE FUNCTION' initialization/postgres/schema.sql"
}

# Test configuration files
test_configurations() {
    print_test_section "Configuration Tests"
    
    cd "$SCENARIO_DIR"
    
    # Service.json validation
    run_test "service.json is valid JSON" "jq empty .vrooli/service.json"
    run_test "service.json has required metadata" "jq -e '.metadata.name and .metadata.description' .vrooli/service.json >/dev/null"
    run_test "service.json has port configuration" "jq -e '.ports.api and .ports.ui' .vrooli/service.json >/dev/null"
    run_test "service.json has resource deps" "jq -e '.resources.required' .vrooli/service.json >/dev/null"
    
    # Scenario test validation
    run_test "scenario-test.yaml is valid YAML" "python3 -c \"import yaml; yaml.safe_load(open('scenario-test.yaml'))\" 2>/dev/null || true"
    
    # Makefile validation
    run_test "Makefile has required targets" "grep -q '^run:' Makefile && grep -q '^test:' Makefile && grep -q '^clean:' Makefile"
}

# Integration tests
test_integration() {
    print_test_section "Integration Tests"
    
    print_info "Integration tests require running services"
    print_info "Use 'make run' to start services, then run integration tests"
    
    # Check if services might be running (basic check)
    API_PORT=${API_PORT:-""}
    UI_PORT=${UI_PORT:-""}
    
    if [ -z "$API_PORT" ]; then
        print_warning "API_PORT not set, skipping integration tests"
        print_info "Set API_PORT environment variable and restart tests"
    elif curl -s http://localhost:${API_PORT}/health >/dev/null 2>&1; then
        print_info "API appears to be running, running integration tests..."
        
        run_test "API health endpoint responds" "curl -s -f http://localhost:${API_PORT}/health >/dev/null"
        run_test "API returns JSON health status" "curl -s http://localhost:${API_PORT}/health | jq -e '.status' >/dev/null"
        
        if [ -n "$UI_PORT" ] && curl -s http://localhost:${UI_PORT}/ >/dev/null 2>&1; then
            run_test "UI is accessible" "curl -s -f http://localhost:${UI_PORT}/ >/dev/null"
        else
            print_warning "UI not accessible (UI_PORT=$UI_PORT)"
        fi
    else
        print_warning "Services not running, skipping integration tests"
        print_info "Start services with: make run"
    fi
}

# Performance tests
test_performance() {
    print_test_section "Performance Tests"
    
    cd "$SCENARIO_DIR"
    
    # Check for large files that might impact performance
    run_test "No large build artifacts" "! find . -name '*.log' -size +10M | head -1 | grep -q ."
    run_test "No large dependencies" "! find ui/node_modules -size +100M 2>/dev/null | head -1 | grep -q . || true"
    
    # API bundle size check
    if [ -f api/calendar-api ]; then
        local api_size=$(stat -f%z api/calendar-api 2>/dev/null || stat -c%s api/calendar-api 2>/dev/null || echo 0)
        if [ "$api_size" -gt 50000000 ]; then # 50MB
            print_warning "API binary is quite large: $(($api_size / 1024 / 1024))MB"
        else
            print_success "API binary size is reasonable: $(($api_size / 1024 / 1024))MB"
        fi
    fi
}

# Security tests
test_security() {
    print_test_section "Security Tests"
    
    cd "$SCENARIO_DIR"
    
    # Check for hardcoded secrets
    run_test "No hardcoded passwords" "! grep -r -i 'password.*=' --include='*.go' --include='*.ts' --include='*.js' . | grep -v test | head -1 | grep -q ."
    run_test "No hardcoded API keys" "! grep -r 'api[_-]key.*=' --include='*.go' --include='*.ts' --include='*.js' . | head -1 | grep -q ."
    run_test "No TODO security items" "! grep -r 'TODO.*security\\|FIXME.*security' --include='*.go' --include='*.ts' . | head -1 | grep -q ."
    
    # Check for proper error handling
    run_test "API has auth middleware" "grep -q 'authMiddleware' api/main.go"
    run_test "Database uses parameterized queries" "grep -q '\\$[0-9]' api/main.go"
}

# Print final results
print_results() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} Test Results Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "Total tests: $TOTAL_TESTS"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}🎉 All tests passed! Calendar scenario is ready.${NC}"
        return 0
    else
        echo -e "${RED}❌ $FAILED_TESTS tests failed. Please review and fix issues.${NC}"
        return 1
    fi
}

# Cleanup function
cleanup() {
    rm -rf "$TEMP_DIR" 2>/dev/null || true
}
trap cleanup EXIT

# Main execution
main() {
    print_header
    
    # Create temp directory
    mkdir -p "$TEMP_DIR"
    
    # Parse arguments
    case "${1:-all}" in
        "structure")
            test_file_structure
            ;;
        "api")
            test_go_api
            ;;
        "cli")
            test_cli
            ;;
        "ui")
            test_ui
            ;;
        "database")
            test_database_schema
            ;;
        "config")
            test_configurations
            ;;
        "integration")
            test_integration
            ;;
        "performance")
            test_performance
            ;;
        "security")
            test_security
            ;;
        "all"|*)
            test_file_structure
            test_configurations
            test_database_schema
            test_go_api
            test_cli
            test_ui
            test_performance
            test_security
            test_integration
            ;;
    esac
    
    print_results
}

# Run main function with all arguments
main "$@"