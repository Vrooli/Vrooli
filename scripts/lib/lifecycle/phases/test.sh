#!/usr/bin/env bash
################################################################################
# Universal Test Phase Handler
# 
# Handles generic test execution:
# - Unit tests
# - Integration tests
# - Shell script tests
# - Coverage reporting
#
# App-specific logic should be in app/lifecycle/test.sh
################################################################################

set -euo pipefail

# Get script directory
LIB_LIFECYCLE_PHASES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${LIB_LIFECYCLE_PHASES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/args.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

################################################################################
# Test Functions
################################################################################

#######################################
# Run shell script tests
# Returns:
#   0 if tests pass, 1 on failure
#######################################
test::run_shell() {
    log::info "Running shell script tests..."
    
    # Check for bats
    if ! command -v bats &> /dev/null; then
        log::warning "bats is not installed, skipping shell tests"
        return 0
    fi
    
    # Find test files
    local test_files=()
    
    # Look for .bats files in scripts/__test
    if [[ -d "${var_SCRIPTS_TEST_DIR}" ]]; then
        while IFS= read -r -d '' file; do
            test_files+=("$file")
        done < <(find "${var_SCRIPTS_TEST_DIR}" -name "*.bats" -print0 2>/dev/null)
    fi
    
    # Look for .bats files in scripts/
    while IFS= read -r -d '' file; do
        test_files+=("$file")
    done < <(find "${var_SCRIPTS_DIR}" -maxdepth 2 -name "*.bats" -print0 2>/dev/null)
    
    if [[ ${#test_files[@]} -eq 0 ]]; then
        log::info "No shell tests found"
        return 0
    fi
    
    log::info "Found ${#test_files[@]} test file(s)"
    
    # Run tests
    local failed=0
    for test_file in "${test_files[@]}"; do
        log::info "Running: $test_file"
        if bats "$test_file"; then
            log::success "✅ Passed: $(basename "$test_file")"
        else
            log::error "❌ Failed: $(basename "$test_file")"
            failed=$((failed + 1))
        fi
    done
    
    if [[ $failed -gt 0 ]]; then
        log::error "$failed test file(s) failed"
        return 1
    fi
    
    log::success "✅ All shell tests passed"
    return 0
}

#######################################
# Run unit tests
# Arguments:
#   $1 - Test pattern (optional)
# Returns:
#   0 if tests pass, 1 on failure
#######################################
test::run_unit() {
    local pattern="${1:-}"
    
    log::info "Running unit tests..."
    
    # Check for package.json
    if [[ ! -f "${var_ROOT_DIR}/package.json" ]]; then
        log::info "No package.json found, skipping unit tests"
        return 0
    fi
    
    # Determine test command
    local test_cmd=""
    
    if command -v pnpm &> /dev/null; then
        test_cmd="pnpm test:unit"
    elif command -v npm &> /dev/null; then
        test_cmd="npm run test:unit"
    else
        log::warning "No package manager found, skipping unit tests"
        return 0
    fi
    
    # Add pattern if provided
    if [[ -n "$pattern" ]]; then
        test_cmd="$test_cmd -- $pattern"
    fi
    
    # Run tests
    log::info "Running: $test_cmd"
    if (cd "${var_ROOT_DIR}" && eval "$test_cmd"); then
        log::success "✅ Unit tests passed"
        return 0
    else
        log::error "Unit tests failed"
        return 1
    fi
}

#######################################
# Run integration tests
# Returns:
#   0 if tests pass, 1 on failure
#######################################
test::run_integration() {
    log::info "Running integration tests..."
    
    # Check if integration tests exist
    local has_integration=false
    
    # Check package.json for integration test script
    if [[ -f "${var_ROOT_DIR}/package.json" ]] && command -v jq &> /dev/null; then
        if jq -e '.scripts["test:integration"]' "${var_ROOT_DIR}/package.json" &> /dev/null; then
            has_integration=true
        fi
    fi
    
    if [[ "$has_integration" == "false" ]]; then
        log::info "No integration tests configured"
        return 0
    fi
    
    # Determine test command
    local test_cmd=""
    
    if command -v pnpm &> /dev/null; then
        test_cmd="pnpm test:integration"
    elif command -v npm &> /dev/null; then
        test_cmd="npm run test:integration"
    fi
    
    # Run tests
    log::info "Running: $test_cmd"
    if (cd "${var_ROOT_DIR}" && eval "$test_cmd"); then
        log::success "✅ Integration tests passed"
        return 0
    else
        log::error "Integration tests failed"
        return 1
    fi
}

#######################################
# Run tests with coverage
# Returns:
#   0 if tests pass, 1 on failure
#######################################
test::run_coverage() {
    log::info "Running tests with coverage..."
    
    # Check if coverage script exists
    local has_coverage=false
    
    if [[ -f "${var_ROOT_DIR}/package.json" ]] && command -v jq &> /dev/null; then
        if jq -e '.scripts["test:coverage"]' "${var_ROOT_DIR}/package.json" &> /dev/null; then
            has_coverage=true
        fi
    fi
    
    if [[ "$has_coverage" == "false" ]]; then
        log::info "No coverage configuration found"
        return 0
    fi
    
    # Determine test command
    local test_cmd=""
    
    if command -v pnpm &> /dev/null; then
        test_cmd="pnpm test:coverage"
    elif command -v npm &> /dev/null; then
        test_cmd="npm run test:coverage"
    fi
    
    # Run tests
    log::info "Running: $test_cmd"
    if (cd "${var_ROOT_DIR}" && eval "$test_cmd"); then
        log::success "✅ Coverage tests passed"
        
        # Look for coverage report
        if [[ -d "${var_ROOT_DIR}/coverage" ]]; then
            log::info "Coverage report available at: ${var_ROOT_DIR}/coverage"
        fi
        
        return 0
    else
        log::error "Coverage tests failed"
        return 1
    fi
}

################################################################################
# Main Test Logic
################################################################################

#######################################
# Run universal test tasks
# Handles generic test operations
# Globals:
#   TEST_TYPE
#   TEST_PATTERN
#   COVERAGE
#   BAIL_ON_FAIL
# Returns:
#   0 on success, 1 on failure
#######################################
test::universal::main() {
    # Initialize phase
    phase::init "Test"
    
    # Get parameters from environment or defaults
    local test_type="${TEST_TYPE:-all}"
    local test_pattern="${TEST_PATTERN:-}"
    local coverage="${COVERAGE:-no}"
    local bail_on_fail="${BAIL_ON_FAIL:-yes}"
    
    log::info "Universal test starting..."
    log::debug "Parameters:"
    log::debug "  Type: $test_type"
    log::debug "  Pattern: $test_pattern"
    log::debug "  Coverage: $coverage"
    log::debug "  Bail on fail: $bail_on_fail"
    
    # Track failures
    local failed_tests=()
    
    # Step 1: Run pre-test hook
    phase::run_hook "preTest"
    
    # Step 2: Run tests based on type
    log::header "🧪 Running Tests"
    
    case "$test_type" in
        all)
            # Run all test types
            if ! test::run_shell; then
                failed_tests+=("shell")
                if flow::is_yes "$bail_on_fail"; then
                    log::error "Shell tests failed, stopping"
                    return 1
                fi
            fi
            
            if ! test::run_unit "$test_pattern"; then
                failed_tests+=("unit")
                if flow::is_yes "$bail_on_fail"; then
                    log::error "Unit tests failed, stopping"
                    return 1
                fi
            fi
            
            if ! test::run_integration; then
                failed_tests+=("integration")
                if flow::is_yes "$bail_on_fail"; then
                    log::error "Integration tests failed, stopping"
                    return 1
                fi
            fi
            ;;
            
        shell)
            if ! test::run_shell; then
                failed_tests+=("shell")
            fi
            ;;
            
        unit)
            if ! test::run_unit "$test_pattern"; then
                failed_tests+=("unit")
            fi
            ;;
            
        integration)
            if ! test::run_integration; then
                failed_tests+=("integration")
            fi
            ;;
            
        *)
            log::error "Unknown test type: $test_type"
            return 1
            ;;
    esac
    
    # Step 3: Run coverage if requested
    if flow::is_yes "$coverage"; then
        log::header "📊 Coverage Report"
        test::run_coverage || {
            log::warning "Coverage generation failed"
        }
    fi
    
    # Step 4: Run app-specific tests
    local app_test="${var_APP_LIFECYCLE_DIR:-}/test.sh"
    if [[ -f "$app_test" ]]; then
        log::header "🧪 App-Specific Tests"
        if ! bash "$app_test"; then
            failed_tests+=("app-specific")
            if flow::is_yes "$bail_on_fail"; then
                log::error "App-specific tests failed"
                return 1
            fi
        fi
    fi
    
    # Step 5: Run post-test hook
    phase::run_hook "postTest"
    
    # Step 6: Report results
    if [[ ${#failed_tests[@]} -gt 0 ]]; then
        log::error "The following test suites failed:"
        for suite in "${failed_tests[@]}"; do
            log::error "  - $suite"
        done
        return 1
    fi
    
    # Complete phase
    phase::complete
    
    log::success "✅ All tests passed"
    
    return 0
}

#######################################
# Entry point for direct execution
#######################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Check if being called by lifecycle engine
    if [[ "${LIFECYCLE_PHASE:-}" == "test" ]]; then
        test::universal::main "$@"
    else
        log::error "This script should be called through the lifecycle engine"
        log::info "Use: ./scripts/manage.sh test [options]"
        exit 1
    fi
fi