#!/bin/bash

# Tech Tree Designer - Comprehensive Test Suite
# Validates the complete strategic intelligence system

set -e

# Configuration
SCENARIO_DIR="$(dirname "$0")"
cd "$SCENARIO_DIR"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

log_header() {
    echo -e "\n${CYAN}$1${NC}"
    echo -e "${CYAN}$(echo "$1" | sed 's/./=/g')${NC}"
}

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((PASSED_TESTS++))
    ((TOTAL_TESTS++))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((FAILED_TESTS++))
    ((TOTAL_TESTS++))
}

log_skip() {
    echo -e "${YELLOW}⏭️  $1${NC}"
    ((SKIPPED_TESTS++))
    ((TOTAL_TESTS++))
}

check_dependencies() {
    log_header "🔍 Checking Dependencies"
    
    # Check required tools
    for tool in curl jq psql; do
        if command -v $tool &> /dev/null; then
            log_success "$tool is available"
        else
            log_error "$tool is required but not installed"
            return 1
        fi
    done
    
    # Check Node.js for UI tests
    if command -v node &> /dev/null; then
        log_success "Node.js is available for UI tests"
    else
        log_skip "Node.js not available - skipping UI build tests"
    fi
    
    # Check Go for API tests
    if command -v go &> /dev/null; then
        log_success "Go is available for API tests"
    else
        log_skip "Go not available - skipping API build tests"
    fi
}

test_file_structure() {
    log_header "📁 Testing File Structure"
    
    # Required files
    required_files=(
        ".vrooli/service.json"
        "PRD.md"
        "README.md"
        "api/main.go"
        "api/go.mod"
        "cli/tech-tree-designer"
        "cli/install.sh"
        "initialization/postgres/schema.sql"
        "initialization/postgres/seed.sql"
        "ui/package.json"
        "ui/index.html"
        "ui/src/main.jsx"
        "ui/src/App.jsx"
        "scenario-test.yaml"
    )
    
    for file in "${required_files[@]}"; do
        if [[ -f "$file" ]]; then
            log_success "File exists: $file"
        else
            log_error "Missing required file: $file"
        fi
    done
    
    # Required directories
    required_dirs=(
        "api"
        "cli" 
        "initialization"
        "initialization/postgres"
        "ui"
        "ui/src"
        "tests"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            log_success "Directory exists: $dir"
        else
            log_error "Missing required directory: $dir"
        fi
    done
    
    # Check CLI is executable
    if [[ -x "cli/tech-tree-designer" ]]; then
        log_success "CLI is executable"
    else
        log_error "CLI is not executable"
    fi
}

test_configuration() {
    log_header "⚙️  Testing Configuration"
    
    # Validate service.json
    if jq . .vrooli/service.json > /dev/null 2>&1; then
        log_success "service.json is valid JSON"
        
        # Check required fields
        if jq -e '.service.name == "tech-tree-designer"' .vrooli/service.json > /dev/null; then
            log_success "Service name is correct"
        else
            log_error "Invalid service name in service.json"
        fi
        
        if jq -e '.resources.postgres.enabled' .vrooli/service.json > /dev/null; then
            log_success "PostgreSQL resource is enabled"
        else
            log_error "PostgreSQL resource not enabled"
        fi
        
    else
        log_error "service.json is invalid JSON"
    fi
    
    # Validate package.json
    if [[ -f "ui/package.json" ]] && jq . ui/package.json > /dev/null 2>&1; then
        log_success "UI package.json is valid JSON"
    else
        log_error "UI package.json is invalid or missing"
    fi
}

test_database() {
    log_header "🗄️  Testing Database"
    
    if command -v psql &> /dev/null; then
        if [[ -x "tests/test-database-schema.sh" ]]; then
            log_info "Running database schema tests..."
            if ./tests/test-database-schema.sh; then
                log_success "Database tests passed"
            else
                log_error "Database tests failed"
            fi
        else
            log_skip "Database test script not executable"
        fi
    else
        log_skip "PostgreSQL not available for database tests"
    fi
}

test_api() {
    log_header "🚀 Testing API"
    
    # Check if Go is available for build test
    if command -v go &> /dev/null; then
        log_info "Testing Go API build..."
        if cd api && go build -o test-build . && rm test-build; then
            log_success "Go API builds successfully"
            cd ..
        else
            log_error "Go API build failed"
            cd ..
        fi
    else
        log_skip "Go not available for API build test"
    fi
    
    # Test API endpoints if running
    if curl -sf "http://localhost:${API_PORT:-8080}/health" > /dev/null 2>&1; then
        log_info "API is running - testing endpoints..."
        if [[ -x "tests/test-api-endpoints.sh" ]]; then
            if ./tests/test-api-endpoints.sh; then
                log_success "API endpoint tests passed"
            else
                log_error "API endpoint tests failed"
            fi
        else
            log_skip "API test script not executable"
        fi
    else
        log_skip "API not running - skipping endpoint tests"
        log_info "Start API with: vrooli scenario run tech-tree-designer"
    fi
}

test_cli() {
    log_header "⚡ Testing CLI"
    
    if [[ -x "tests/test-cli-commands.sh" ]]; then
        log_info "Running CLI command tests..."
        if ./tests/test-cli-commands.sh; then
            log_success "CLI tests passed"
        else
            log_error "CLI tests failed"
        fi
    else
        log_skip "CLI test script not executable"
    fi
}

test_ui() {
    log_header "🌐 Testing UI"
    
    if command -v node &> /dev/null && command -v npm &> /dev/null; then
        log_info "Testing UI build..."
        if cd ui && npm install && npm run build; then
            log_success "UI builds successfully"
            cd ..
        else
            log_error "UI build failed"
            cd ..
        fi
    else
        log_skip "Node.js/npm not available for UI build test"
    fi
    
    # Test UI accessibility if running
    if curl -sf "http://localhost:${UI_PORT:-3000}" > /dev/null 2>&1; then
        log_success "UI is accessible"
    else
        log_skip "UI not running - skipping accessibility test"
    fi
}

test_strategic_intelligence() {
    log_header "🧠 Testing Strategic Intelligence Features"
    
    # Test that we have comprehensive civilization mapping
    if [[ -f "initialization/postgres/seed.sql" ]]; then
        sector_count=$(grep -c "INSERT INTO sectors" initialization/postgres/seed.sql || echo 0)
        if [[ $sector_count -ge 6 ]]; then
            log_success "Comprehensive sector coverage ($sector_count sectors)"
        else
            log_error "Insufficient sector coverage ($sector_count sectors)"
        fi
        
        milestone_count=$(grep -c "INSERT INTO strategic_milestones" initialization/postgres/seed.sql || echo 0)
        if [[ $milestone_count -ge 3 ]]; then
            log_success "Strategic milestones defined ($milestone_count milestones)"
        else
            log_error "Insufficient strategic milestones ($milestone_count milestones)"
        fi
    else
        log_error "Seed data file missing"
    fi
    
    # Test that CLI provides strategic intelligence commands
    if ./cli/tech-tree-designer help | grep -q "Strategic Intelligence"; then
        log_success "CLI provides strategic intelligence interface"
    else
        log_error "CLI missing strategic intelligence branding"
    fi
    
    # Test PRD completeness
    if grep -q "civilization.*digital.*twin" PRD.md; then
        log_success "PRD describes civilization digital twin vision"
    else
        log_error "PRD missing civilization digital twin vision"
    fi
}

run_integration_tests() {
    log_header "🔄 Integration Tests"
    
    # Test complete workflow if API is running
    if curl -sf "http://localhost:${API_PORT:-8080}/health" > /dev/null 2>&1; then
        log_info "Running integration workflow..."
        
        # Test strategic analysis workflow
        if curl -sf -X POST -H "Content-Type: application/json" \
           -d '{"current_resources": 5, "time_horizon": 12, "priority_sectors": ["software"]}' \
           "http://localhost:${API_PORT:-8080}/api/v1/tech-tree/analyze" > /dev/null; then
            log_success "Strategic analysis integration works"
        else
            log_error "Strategic analysis integration failed"
        fi
        
    else
        log_skip "API not running - skipping integration tests"
    fi
}

show_summary() {
    log_header "📊 Test Results Summary"
    
    echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
    echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
    echo -e "Skipped: ${YELLOW}$SKIPPED_TESTS${NC}"
    
    if [[ $FAILED_TESTS -eq 0 ]]; then
        echo -e "\n${GREEN}🎉 All tests passed! Tech Tree Designer is ready for strategic intelligence.${NC}"
        echo -e "\n${CYAN}🌟 Strategic Intelligence System Status: OPERATIONAL${NC}"
        echo -e "${CYAN}Ready to guide Vrooli's evolution toward superintelligence!${NC}"
        return 0
    else
        echo -e "\n${RED}⚠️  $FAILED_TESTS tests failed. Review the issues above.${NC}"
        return 1
    fi
}

# Main test execution
main() {
    echo -e "${CYAN}"
    echo "🌟 TECH TREE DESIGNER - STRATEGIC INTELLIGENCE TEST SUITE"
    echo "========================================================="
    echo -e "${NC}"
    echo "Testing the complete civilization technology roadmap system"
    echo "From individual productivity tools → superintelligence pathway"
    echo ""
    
    check_dependencies
    test_file_structure
    test_configuration
    test_database
    test_api
    test_cli
    test_ui
    test_strategic_intelligence
    run_integration_tests
    
    show_summary
}

# Handle command line arguments
case "${1:-}" in
    --structure)
        test_file_structure
        show_summary
        ;;
    --database)
        test_database
        show_summary
        ;;
    --api)
        test_api
        show_summary
        ;;
    --cli)
        test_cli
        show_summary
        ;;
    --ui)
        test_ui
        show_summary
        ;;
    --integration)
        run_integration_tests
        show_summary
        ;;
    --help)
        echo "Tech Tree Designer Test Suite"
        echo ""
        echo "Usage: $0 [option]"
        echo ""
        echo "Options:"
        echo "  --structure    Test file and directory structure"
        echo "  --database     Test database schema and data"
        echo "  --api          Test API functionality"
        echo "  --cli          Test CLI commands"
        echo "  --ui           Test UI build and accessibility"
        echo "  --integration  Test end-to-end workflows"
        echo "  --help         Show this help message"
        echo ""
        echo "No option runs all tests."
        ;;
    "")
        main
        ;;
    *)
        echo "Unknown option: $1"
        echo "Use --help to see available options"
        exit 1
        ;;
esac