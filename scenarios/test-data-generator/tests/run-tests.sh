#!/bin/bash

# Test Data Generator - Comprehensive Test Runner
# This script runs all tests for the test-data-generator scenario

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="$SCENARIO_DIR/api"
UI_DIR="$SCENARIO_DIR/ui"
TESTS_DIR="$SCENARIO_DIR/tests"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_color() {
    local color=$1
    shift
    echo -e "${color}$*${NC}"
}

print_error() {
    print_color $RED "❌ $*"
}

print_success() {
    print_color $GREEN "✅ $*"
}

print_info() {
    print_color $BLUE "ℹ️  $*"
}

print_warning() {
    print_color $YELLOW "⚠️  $*"
}

print_header() {
    echo
    echo "=================================================="
    print_color $BLUE "$1"
    echo "=================================================="
}

# Global variables for tracking
TESTS_PASSED=0
TESTS_FAILED=0
START_TIME=$(date +%s)

# Function to run a test and track results
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    print_info "Running: $test_name"
    
    if eval "$test_command" > /dev/null 2>&1; then
        print_success "PASSED: $test_name"
        ((TESTS_PASSED++))
        return 0
    else
        print_error "FAILED: $test_name"
        print_info "Command: $test_command"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Check if required tools are installed
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    run_test "Node.js availability" "command -v node"
    run_test "npm availability" "command -v npm"
    run_test "curl availability" "command -v curl"
    
    if command -v jq &> /dev/null; then
        run_test "jq availability (optional)" "command -v jq"
    else
        print_warning "jq not found - some tests may be limited"
    fi
}

# Test file structure
test_file_structure() {
    print_header "Testing File Structure"
    
    local required_files=(
        "$SCENARIO_DIR/README.md"
        "$SCENARIO_DIR/scenario-test.yaml"
        "$API_DIR/package.json"
        "$API_DIR/server.js"
        "$UI_DIR/package.json" 
        "$UI_DIR/server.js"
        "$UI_DIR/index.html"
        "$UI_DIR/app.js"
        "$UI_DIR/styles.css"
        "$SCENARIO_DIR/cli/test-data-generator"
        "$SCENARIO_DIR/cli/install.sh"
        "$SCENARIO_DIR/initialization/startup.sh"
    )
    
    for file in "${required_files[@]}"; do
        run_test "File exists: $(basename $file)" "test -f '$file'"
    done
    
    run_test "CLI script is executable" "test -x '$SCENARIO_DIR/cli/test-data-generator'"
    run_test "Install script is executable" "test -x '$SCENARIO_DIR/cli/install.sh'"
    run_test "Startup script is executable" "test -x '$SCENARIO_DIR/initialization/startup.sh'"
}

# Install dependencies
install_dependencies() {
    print_header "Installing Dependencies"
    
    # API dependencies
    print_info "Installing API dependencies..."
    cd "$API_DIR"
    if npm install > /dev/null 2>&1; then
        print_success "API dependencies installed"
    else
        print_error "Failed to install API dependencies"
        return 1
    fi
    
    # UI dependencies  
    print_info "Installing UI dependencies..."
    cd "$UI_DIR"
    if npm install > /dev/null 2>&1; then
        print_success "UI dependencies installed"
    else
        print_error "Failed to install UI dependencies"
        return 1
    fi
}

# Start services for testing
start_services() {
    print_header "Starting Services"
    
    # Start API server in background
    print_info "Starting API server..."
    cd "$API_DIR"
    npm start > /dev/null 2>&1 &
    API_PID=$!
    sleep 3
    
    if kill -0 $API_PID 2>/dev/null; then
        print_success "API server started (PID: $API_PID)"
    else
        print_error "Failed to start API server"
        return 1
    fi
    
    # Start UI server in background
    print_info "Starting UI server..."
    cd "$UI_DIR"
    npm start > /dev/null 2>&1 &
    UI_PID=$!
    sleep 3
    
    if kill -0 $UI_PID 2>/dev/null; then
        print_success "UI server started (PID: $UI_PID)"
    else
        print_error "Failed to start UI server"
        kill $API_PID 2>/dev/null || true
        return 1
    fi
}

# Stop services
stop_services() {
    print_info "Stopping services..."
    
    if [ ! -z "$API_PID" ]; then
        kill $API_PID 2>/dev/null || true
        print_info "API server stopped"
    fi
    
    if [ ! -z "$UI_PID" ]; then
        kill $UI_PID 2>/dev/null || true  
        print_info "UI server stopped"
    fi
}

# Test API endpoints
test_api() {
    print_header "Testing API Endpoints"
    
    local api_url="http://localhost:3001"
    
    run_test "API health check" "curl -s '$api_url/health' | grep -q 'healthy'"
    run_test "Get data types" "curl -s '$api_url/api/types' | grep -q 'types'"
    
    # Test user generation
    run_test "Generate users (JSON)" "curl -s -X POST -H 'Content-Type: application/json' -d '{\"count\":3}' '$api_url/api/generate/users' | grep -q 'success'"
    
    # Test company generation
    run_test "Generate companies" "curl -s -X POST -H 'Content-Type: application/json' -d '{\"count\":2}' '$api_url/api/generate/companies' | grep -q 'success'"
    
    # Test custom schema
    run_test "Generate custom data" "curl -s -X POST -H 'Content-Type: application/json' -d '{\"count\":2,\"schema\":{\"id\":\"uuid\",\"name\":\"string\"}}' '$api_url/api/generate/custom' | grep -q 'success'"
    
    # Test error handling
    run_test "API error handling" "curl -s '$api_url/nonexistent' | grep -q 'error'"
}

# Test UI
test_ui() {
    print_header "Testing UI"
    
    local ui_url="http://localhost:3002"
    
    run_test "UI health check" "curl -s '$ui_url/health' | grep -q 'healthy'"
    run_test "UI main page loads" "curl -s '$ui_url/' | grep -q 'Test Data Generator'"
    run_test "UI includes JavaScript" "curl -s '$ui_url/' | grep -q 'app.js'"
    run_test "UI includes CSS" "curl -s '$ui_url/' | grep -q 'styles.css'"
}

# Test CLI
test_cli() {
    print_header "Testing CLI"
    
    local cli="$SCENARIO_DIR/cli/test-data-generator"
    
    run_test "CLI help command" "$cli help | grep -q 'USAGE'"
    run_test "CLI types command" "$cli types | grep -q 'Available data types'"
    run_test "CLI health command" "$cli health | grep -q 'healthy'"
    
    # Test data generation (requires API to be running)
    run_test "CLI generate users" "$cli generate users --count 2 | grep -q '\"id\"'"
    run_test "CLI generate with seed" "$cli generate users --count 1 --seed 123 --pretty | grep -q '\"name\"'"
}

# Run unit tests if they exist
run_unit_tests() {
    print_header "Running Unit Tests"
    
    cd "$API_DIR"
    if [ -f "package.json" ] && npm list jest > /dev/null 2>&1; then
        if npm test > /dev/null 2>&1; then
            print_success "Unit tests passed"
        else
            print_error "Unit tests failed"
        fi
    else
        print_info "No unit tests configured - skipping"
    fi
}

# Performance tests
test_performance() {
    print_header "Performance Tests"
    
    local api_url="http://localhost:3001"
    
    print_info "Testing generation of 1000 records..."
    local start_time=$(date +%s%N)
    
    if curl -s -X POST -H 'Content-Type: application/json' -d '{"count":1000}' "$api_url/api/generate/users" | grep -q 'success'; then
        local end_time=$(date +%s%N)
        local duration=$(( (end_time - start_time) / 1000000 ))
        print_success "Generated 1000 records in ${duration}ms"
        
        if [ $duration -lt 5000 ]; then
            print_success "Performance test: PASSED (< 5 seconds)"
        else
            print_warning "Performance test: SLOW (> 5 seconds)"
        fi
    else
        print_error "Performance test: FAILED"
    fi
}

# Generate test report
generate_report() {
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))
    local total_tests=$((TESTS_PASSED + TESTS_FAILED))
    
    print_header "Test Report"
    
    echo "Test Summary:"
    echo "  Total Tests: $total_tests"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    echo "  Duration: ${duration}s"
    echo
    
    if [ $TESTS_FAILED -eq 0 ]; then
        print_success "All tests passed! 🎉"
        echo
        print_info "The Test Data Generator scenario is ready to use:"
        print_info "  API: http://localhost:3001"
        print_info "  UI:  http://localhost:3002"
        print_info "  CLI: test-data-generator --help"
        return 0
    else
        print_error "$TESTS_FAILED test(s) failed"
        return 1
    fi
}

# Cleanup function
cleanup() {
    print_info "Cleaning up..."
    stop_services
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM

# Main test execution
main() {
    print_header "Test Data Generator - Test Suite"
    print_info "Starting comprehensive test suite..."
    
    check_prerequisites
    test_file_structure
    install_dependencies
    start_services
    
    # Give services time to fully start
    sleep 2
    
    test_api
    test_ui
    test_cli
    run_unit_tests
    test_performance
    
    stop_services
    
    # Wait a moment for services to stop
    sleep 1
    
    generate_report
}

# Run with error handling
if main "$@"; then
    exit 0
else
    cleanup
    exit 1
fi