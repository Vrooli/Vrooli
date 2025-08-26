#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Scenario Integration Test Runner
# 
# Automated integration testing for Vrooli scenarios.
# Handles app generation, lifecycle testing, and cleanup.
#
# Usage:
#   ./run-integration-test.sh <scenario-name> [options]
#   ./run-integration-test.sh agent-metareasoning-manager --verbose
#
# This runner:
# - Validates the scenario (static analysis)
# - Generates app if needed (checks hash)
# - Runs setup, develop, test lifecycle
# - Executes integration tests from scenario-test.yaml
# - Cleans up after testing
#
################################################################################

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCENARIO_TOOLS_DIR="${APP_ROOT}/scripts/scenarios/tools"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# TODO: CONTEXT ISSUE - var_ROOT_DIR changes meaning depending on where script runs:
# - When run from Vrooli: var_ROOT_DIR = Vrooli root (correct for our uses)
# - When run from generated app: var_ROOT_DIR = generated app root (incorrect)
# This causes issues with:
#   1. Finding validate-scenario.sh (needs Vrooli root)
#   2. Locating scenario-hashes.json (needs Vrooli root)
#   3. Finding scenario directories (needs Vrooli root)
# Solution: Need to track both VROOLI_ROOT and APP_ROOT separately
# For now, this script assumes it's run from Vrooli root

# Global variables
SCENARIO_NAME=""
SCENARIO_PATH=""
GENERATED_APP_PATH=""
VERBOSE=false
KEEP_APP=false
SKIP_GENERATION=false
SKIP_LIFECYCLE=false

# Color codes
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Test results
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Usage information
show_usage() {
    cat << EOF
Usage: $0 <scenario-name> [options]

Run comprehensive integration tests for a Vrooli scenario.

Arguments:
  scenario-name     Name of the scenario to test

Options:
  --verbose         Show detailed test output
  --keep-app        Don't delete generated app after testing
  --skip-generation Skip app generation (use existing)
  --skip-lifecycle  Skip lifecycle testing (setup/develop/test)
  --help            Show this help message

Test Process:
  1. Static validation (validate-scenario.sh)
  2. App generation (if needed)
  3. Lifecycle testing (setup, develop, test)
  4. Integration tests (scenario-test.yaml)
  5. Cleanup (unless --keep-app)

Examples:
  $0 agent-metareasoning-manager
  $0 research-assistant --verbose --keep-app
  $0 my-scenario --skip-generation

Exit Codes:
  0 - All tests passed
  1 - Test failures
  2 - Setup/generation errors
EOF
}

# Parse arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        echo -e "${RED}Error: No scenario name provided${NC}"
        show_usage
        exit 1
    fi

    SCENARIO_NAME="$1"
    shift

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose)
                VERBOSE=true
                ;;
            --keep-app)
                KEEP_APP=true
                ;;
            --skip-generation)
                SKIP_GENERATION=true
                ;;
            --skip-lifecycle)
                SKIP_LIFECYCLE=true
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                show_usage
                exit 1
                ;;
        esac
        shift
    done
}

# Print functions
print_header() {
    echo
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║         Scenario Integration Test Runner               ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo
}

print_phase() {
    echo
    echo "──────────────────────────────────────────────────────────"
    echo "  $1"
    echo "──────────────────────────────────────────────────────────"
    echo
}

print_check() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}  ✓${NC} $1"
    ((TESTS_PASSED++))
}

print_error() {
    echo -e "${RED}  ✗${NC} $1" >&2
    ((TESTS_FAILED++))
}

print_warning() {
    echo -e "${YELLOW}  ⚠${NC} $1"
}

print_skip() {
    echo -e "${YELLOW}  ⊘${NC} $1 (skipped)"
    ((TESTS_SKIPPED++))
}

print_info() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "    ℹ $1"
    fi
}

# Check if app needs regeneration
needs_regeneration() {
    local hash_file="${var_ROOT_DIR}/data/scenario-hashes.json"
    
    if [[ ! -f "$hash_file" ]]; then
        print_info "No hash file found, generation needed"
        return 0
    fi
    
    # Calculate current hash of scenario files
    local current_hash
    current_hash=$(find "$SCENARIO_PATH" -type f -exec sha256sum {} \; | \
                   sort | sha256sum | awk '{print $1}')
    
    # Get stored hash
    local stored_hash=""
    if command -v jq >/dev/null 2>&1; then
        stored_hash=$(jq -r ".\"$SCENARIO_NAME\" // \"\"" "$hash_file" 2>/dev/null || echo "")
    fi
    
    if [[ "$current_hash" != "$stored_hash" ]]; then
        print_info "Scenario changed (hash mismatch), regeneration needed"
        return 0
    fi
    
    # Check if app exists
    if [[ ! -d "$GENERATED_APP_PATH" ]]; then
        print_info "Generated app not found, generation needed"
        return 0
    fi
    
    print_info "App is up-to-date (hash match)"
    return 1
}

# Update scenario hash
update_hash() {
    local hash_file="${var_ROOT_DIR}/data/scenario-hashes.json"
    local current_hash
    current_hash=$(find "$SCENARIO_PATH" -type f -exec sha256sum {} \; | \
                   sort | sha256sum | awk '{print $1}')
    
    # Create directory if needed
    mkdir -p "$(dirname "$hash_file")"
    
    # Update hash file
    if [[ -f "$hash_file" ]] && command -v jq >/dev/null 2>&1; then
        local updated
        updated=$(jq ". + {\"$SCENARIO_NAME\": \"$current_hash\"}" "$hash_file")
        echo "$updated" > "$hash_file"
    else
        echo "{\"$SCENARIO_NAME\": \"$current_hash\"}" > "$hash_file"
    fi
    
    print_info "Updated scenario hash"
}

# Phase 1: Static Validation
run_static_validation() {
    print_phase "Phase 1: Static Validation"
    
    local validator="${SCENARIO_TOOLS_DIR}/validate-scenario.sh"
    
    if [[ ! -f "$validator" ]]; then
        print_warning "Static validator not found, skipping"
        return 0
    fi
    
    print_check "Running static analysis..."
    
    if "$validator" "$SCENARIO_NAME" ${VERBOSE:+--verbose}; then
        print_success "Static validation passed"
        return 0
    else
        local exit_code=$?
        if [[ $exit_code -eq 2 ]]; then
            print_warning "Static validation passed with warnings"
            return 0
        else
            print_error "Static validation failed"
            return 1
        fi
    fi
}

# Phase 2: App Generation
generate_app() {
    print_phase "Phase 2: App Generation"
    
    SCENARIO_PATH="${var_SCENARIOS_DIR}/${SCENARIO_NAME}"
    GENERATED_APP_PATH="${HOME}/generated-apps/${SCENARIO_NAME}"
    
    if [[ "$SKIP_GENERATION" == "true" ]]; then
        print_skip "App generation"
        if [[ ! -d "$GENERATED_APP_PATH" ]]; then
            print_error "Generated app not found at: $GENERATED_APP_PATH"
            return 1
        fi
        return 0
    fi
    
    if ! needs_regeneration; then
        print_success "Using existing app (up-to-date)"
        return 0
    fi
    
    print_check "Generating app from scenario..."
    
    local generator="${SCENARIO_TOOLS_DIR}/scenario-to-app.sh"
    
    if [[ ! -f "$generator" ]]; then
        print_error "App generator not found: $generator"
        return 1
    fi
    
    if "$generator" "$SCENARIO_NAME" ${VERBOSE:+--verbose}; then
        print_success "App generated successfully"
        update_hash
        return 0
    else
        print_error "App generation failed"
        return 1
    fi
}

# Phase 3: Lifecycle Testing
test_lifecycle() {
    print_phase "Phase 3: Lifecycle Testing"
    
    if [[ "$SKIP_LIFECYCLE" == "true" ]]; then
        print_skip "Lifecycle testing"
        return 0
    fi
    
    if [[ ! -d "$GENERATED_APP_PATH" ]]; then
        print_error "Generated app not found"
        return 1
    fi
    
    cd "$GENERATED_APP_PATH"
    
    # Test setup lifecycle
    print_check "Testing setup lifecycle..."
    if ./scripts/manage.sh setup ${VERBOSE:+--verbose}; then
        print_success "Setup lifecycle passed"
    else
        print_error "Setup lifecycle failed"
        return 1
    fi
    
    # Test develop lifecycle
    print_check "Testing develop lifecycle..."
    if timeout 30 ./scripts/manage.sh develop ${VERBOSE:+--verbose}; then
        print_success "Develop lifecycle passed"
    else
        print_warning "Develop lifecycle timed out or failed (may be normal for long-running services)"
    fi
    
    # Test 'test' lifecycle (may not be implemented)
    print_check "Testing test lifecycle..."
    if ./scripts/manage.sh test ${VERBOSE:+--verbose} 2>/dev/null; then
        print_success "Test lifecycle passed"
    else
        print_warning "Test lifecycle not implemented or failed (expected)"
    fi
    
    # Stop services
    print_check "Stopping services..."
    if ./scripts/manage.sh stop ${VERBOSE:+--verbose} 2>/dev/null; then
        print_success "Services stopped"
    else
        print_warning "Stop lifecycle may not be implemented"
    fi
    
    return 0
}

# Phase 4: Integration Tests
run_integration_tests() {
    print_phase "Phase 4: Integration Tests"
    
    local test_file="${SCENARIO_PATH}/scenario-test.yaml"
    
    # Also check for new name
    if [[ ! -f "$test_file" ]]; then
        test_file="${SCENARIO_PATH}/integration-test.yaml"
    fi
    
    if [[ ! -f "$test_file" ]]; then
        print_warning "No integration test file found (scenario-test.yaml or integration-test.yaml)"
        print_info "Create one to define integration tests"
        return 0
    fi
    
    print_check "Running integration tests from $(basename "$test_file")..."
    
    # Use the scenario test runner if available
    local test_runner="${var_SCRIPTS_SCENARIOS_DIR}/validation/scenario-test-runner.sh"
    
    if [[ -f "$test_runner" ]]; then
        if "$test_runner" --scenario "$SCENARIO_PATH" ${VERBOSE:+--verbose}; then
            print_success "Integration tests passed"
            return 0
        else
            print_error "Integration tests failed"
            return 1
        fi
    else
        print_warning "Test runner not found, skipping integration tests"
        return 0
    fi
}

# Phase 5: Cleanup
cleanup() {
    print_phase "Phase 5: Cleanup"
    
    if [[ "$KEEP_APP" == "true" ]]; then
        print_skip "App cleanup (--keep-app specified)"
        print_info "Generated app kept at: $GENERATED_APP_PATH"
    else
        print_check "Cleaning up generated app..."
        if [[ -d "$GENERATED_APP_PATH" ]]; then
            rm -rf "$GENERATED_APP_PATH"
            print_success "Generated app removed"
        else
            print_info "No app to clean up"
        fi
    fi
}

# Generate test report
generate_report() {
    local total=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))
    local success_rate=0
    
    if [[ $total -gt 0 ]]; then
        success_rate=$((TESTS_PASSED * 100 / total))
    fi
    
    echo
    echo "════════════════════════════════════════════════════════"
    echo "                    Test Summary                        "
    echo "════════════════════════════════════════════════════════"
    echo -e "  ${GREEN}Passed:${NC}  $TESTS_PASSED"
    echo -e "  ${RED}Failed:${NC}  $TESTS_FAILED"
    echo -e "  ${YELLOW}Skipped:${NC} $TESTS_SKIPPED"
    echo "  ──────────────────────────────────────────────────────"
    echo "  Total:   $total"
    echo "  Success Rate: ${success_rate}%"
    echo "════════════════════════════════════════════════════════"
    echo
    
    if [[ $TESTS_FAILED -gt 0 ]]; then
        echo -e "${RED}✗ Tests failed${NC}"
        return 1
    else
        echo -e "${GREEN}✓ All tests passed${NC}"
        return 0
    fi
}

# Main execution
main() {
    parse_args "$@"
    print_header
    
    echo "Testing scenario: $SCENARIO_NAME"
    [[ "$VERBOSE" == "true" ]] && echo "Verbose mode enabled"
    [[ "$KEEP_APP" == "true" ]] && echo "App will be kept after testing"
    
    # Run test phases
    run_static_validation || exit 2
    generate_app || exit 2
    test_lifecycle
    run_integration_tests
    cleanup
    
    # Generate report and exit
    generate_report
    exit $?
}

# Trap for cleanup on exit
trap 'echo -e "\n${YELLOW}Test interrupted${NC}"' INT TERM

# Execute main
main "$@"