#!/bin/bash
# Scenario Test Runner - Main orchestrator for declarative scenario tests
# This replaces 1000+ lines of boilerplate with a simple, maintainable runner

set -euo pipefail

# Source var.sh first with proper relative path
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../lib/utils/var.sh"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Global variables
SCENARIO_DIR=""
CONFIG_FILE="scenario-test.yaml"
VERBOSE=false
DRY_RUN=false
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0
TESTS_DEGRADED=0

# Resolve script location using var_ variables
FRAMEWORK_DIR="$var_SCRIPTS_SCENARIOS_DIR/framework"

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --scenario)
                SCENARIO_DIR="$2"
                shift 2
                ;;
            --config)
                CONFIG_FILE="$2"
                shift 2
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    # Validate required arguments
    if [[ -z "$SCENARIO_DIR" ]]; then
        print_error "Scenario directory is required"
        show_help
        exit 1
    fi

    if [[ ! -d "$SCENARIO_DIR" ]]; then
        print_error "Scenario directory does not exist: $SCENARIO_DIR"
        exit 1
    fi
}

# Show help message
show_help() {
    cat << EOF
Scenario Test Runner - Declarative testing for Vrooli scenarios

Usage: $(basename "$0") --scenario <dir> [options]

Options:
    --scenario <dir>    Path to scenario directory (required)
    --config <file>     Test configuration file (default: scenario-test.yaml)
    --verbose, -v       Verbose output
    --dry-run          Show what would be executed without running
    --help, -h         Show this help message

Examples:
    # Run tests for a scenario
    $(basename "$0") --scenario ./core/research-assistant

    # Run with custom config file
    $(basename "$0") --scenario ./core/my-scenario --config custom-test.yaml

    # Dry run to see what would be executed
    $(basename "$0") --scenario ./core/my-scenario --dry-run
EOF
}

# Load and parse YAML configuration
load_config() {
    local config_path="$SCENARIO_DIR/$CONFIG_FILE"
    
    if [[ ! -f "$config_path" ]]; then
        print_error "Configuration file not found: $config_path"
        print_info "Creating minimal test configuration..."
        create_minimal_config "$config_path"
        return 0
    fi

    print_info "Loading configuration from: $CONFIG_FILE"
    
    # For now, we'll use basic parsing. In production, use yq or python
    # This is a simplified parser for demonstration
    export TEST_CONFIG="$config_path"
}

# Create minimal configuration if none exists
create_minimal_config() {
    local config_path="$1"
    
    cat > "$config_path" << 'EOF'
version: 1.0
scenario: unnamed-scenario

structure:
  required_files:
    - metadata.yaml
    - manifest.yaml
    - README.md

resources:
  required: []
  optional: []

tests:
  - name: "Basic Structure Check"
    type: structure
    validate:
      - metadata.yaml
      - manifest.yaml

validation:
  success_rate: 100
EOF
    
    print_success "Created minimal configuration: $config_path"
}

# Validate scenario structure
validate_structure() {
    print_info "Validating scenario structure..."
    
    if [[ -f "$FRAMEWORK_DIR/validators/structure.sh" ]]; then
        source "$FRAMEWORK_DIR/validators/structure.sh"
        validate_scenario_structure "$SCENARIO_DIR" "$TEST_CONFIG"
        local result=$?
        [[ "$VERBOSE" == "true" ]] && print_info "Structure validation returned: $result"
        return $result
    else
        print_warning "Structure validator not found, skipping..."
        return 0
    fi
}

# Check resource health
check_resources() {
    print_info "Checking resource health..."
    
    if [[ -f "$FRAMEWORK_DIR/validators/resources.sh" ]]; then
        source "$FRAMEWORK_DIR/validators/resources.sh"
        check_resource_health "$SCENARIO_DIR" "$TEST_CONFIG"
        return $?
    else
        print_warning "Resource validator not found, skipping..."
        return 0
    fi
}

# Execute a single test based on type
execute_test() {
    local test_name="$1"
    local test_type="$2"
    local test_data="$3"
    
    [[ "$VERBOSE" == "true" ]] && print_info "Executing test: $test_name (type: $test_type)"
    [[ "$VERBOSE" == "true" ]] && print_info "Before test execution: TESTS_PASSED=$TESTS_PASSED, TESTS_FAILED=$TESTS_FAILED"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        print_info "[DRY RUN] Would execute: $test_name"
        return 0
    fi
    
    local handler="$FRAMEWORK_DIR/handlers/${test_type}.sh"
    
    if [[ -f "$handler" ]]; then
        source "$handler"
        # Call the handler function (each handler implements execute_<type>_test)
        "execute_${test_type}_test" "$test_name" "$test_data"
        local result=$?
        [[ "$VERBOSE" == "true" ]] && print_info "Handler returned result: $result"
        
        if [[ $result -eq 0 ]]; then
            print_success "$test_name"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            [[ "$VERBOSE" == "true" ]] && print_info "Test passed. TESTS_PASSED=$TESTS_PASSED, TESTS_FAILED=$TESTS_FAILED"
        elif [[ $result -eq 2 ]]; then
            print_warning "$test_name (degraded functionality)"
            TESTS_DEGRADED=$((TESTS_DEGRADED + 1))
            [[ "$VERBOSE" == "true" ]] && print_info "Test degraded. TESTS_PASSED=$TESTS_PASSED, TESTS_FAILED=$TESTS_FAILED, TESTS_DEGRADED=$TESTS_DEGRADED"
        else
            print_error "$test_name"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            [[ "$VERBOSE" == "true" ]] && print_info "Test failed. TESTS_PASSED=$TESTS_PASSED, TESTS_FAILED=$TESTS_FAILED"
        fi
        
        return $result
    else
        print_warning "Handler not found for type: $test_type (skipping test)"
        TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
        return 0
    fi
}

# Parse tests from YAML configuration
parse_yaml_tests() {
    local config_file="$1"
    
    # Extract tests section from YAML (simplified parser)
    # This extracts test blocks separated by "- name:"
    awk '
        /^tests:/ { in_tests=1; next }
        in_tests && /^[a-z_]+:/ && !/^  / { in_tests=0 }
        in_tests && /^  - name:/ { 
            if (test_block) print "---TEST_SEPARATOR---"
            test_block=1
        }
        in_tests && test_block { print }
    ' "$config_file"
}

# Extract test field value
get_test_field() {
    local test_block="$1"
    local field="$2"
    echo "$test_block" | grep "[[:space:]]*${field}:" | head -1 | sed "s/.*${field}:[[:space:]]*//" | sed 's/"//g' | xargs
}

# Run all tests from configuration
run_tests() {
    print_info "Running scenario tests..."
    
    if [[ ! -f "$TEST_CONFIG" ]]; then
        print_error "Test configuration not found: $TEST_CONFIG"
        return 1
    fi
    
    # Parse tests from YAML
    local tests_raw=$(parse_yaml_tests "$TEST_CONFIG")
    [[ "$VERBOSE" == "true" ]] && print_info "Parsed tests raw: '$tests_raw'"
    
    if [[ -z "$tests_raw" ]]; then
        print_warning "No tests defined in configuration"
        # Still run custom tests if they exist
        if [[ -f "$SCENARIO_DIR/custom-tests.sh" ]]; then
            print_info "Running custom tests..."
            # Export helper functions for custom tests
            export -f print_info
            export -f print_success
            export -f print_error
            export -f print_warning
            
            # Source custom test helpers
            if [[ -f "$FRAMEWORK_DIR/handlers/custom.sh" ]]; then
                source "$FRAMEWORK_DIR/handlers/custom.sh"
                # Export custom print functions
                export -f print_custom_info
                export -f print_custom_success
                export -f print_custom_error
                export -f print_custom_warning
            fi
            
            source "$SCENARIO_DIR/custom-tests.sh"
            if declare -f run_custom_tests >/dev/null; then
                run_custom_tests
            fi
        fi
        return 0
    fi
    
    # Split tests and execute each one
    local IFS=$'\n'
    local test_blocks=()
    local current_block=""
    
    while read -r line; do
        if [[ "$line" == "---TEST_SEPARATOR---" ]]; then
            if [[ -n "$current_block" ]]; then
                test_blocks+=("$current_block")
            fi
            current_block=""
        else
            current_block="${current_block}${line}"$'\n'
        fi
    done <<< "$tests_raw"
    
    # Add the last block
    if [[ -n "$current_block" ]]; then
        test_blocks+=("$current_block")
    fi
    
    [[ "$VERBOSE" == "true" ]] && print_info "Found ${#test_blocks[@]} test blocks"
    
    # Execute each test
    for test_block in "${test_blocks[@]}"; do
        local test_name=$(get_test_field "$test_block" "name")
        local test_type=$(get_test_field "$test_block" "type")
        [[ "$VERBOSE" == "true" ]] && print_info "Processing test: name='$test_name', type='$test_type'"
        
        if [[ -z "$test_name" || -z "$test_type" ]]; then
            print_warning "Skipping test with missing name or type"
            continue
        fi
        
        # Create temporary file with test configuration
        local test_config_file=$(mktemp)
        echo "$test_block" > "$test_config_file"
        
        # Execute the test based on type
        execute_test "$test_name" "$test_type" "$test_config_file"
        
        # Clean up temp file
        rm -f "$test_config_file"
    done
}

# Generate test report
generate_report() {
    local total=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED + TESTS_DEGRADED))
    local success_rate=0
    local healthy_rate=0
    
    if [[ $total -gt 0 ]]; then
        success_rate=$(((TESTS_PASSED + TESTS_DEGRADED) * 100 / total))
        healthy_rate=$((TESTS_PASSED * 100 / total))
    fi
    
    echo
    echo "════════════════════════════════════════════════════════"
    echo "                    Test Summary                        "
    echo "════════════════════════════════════════════════════════"
    echo -e "  ${GREEN}Passed:${NC}  $TESTS_PASSED"
    if [[ $TESTS_DEGRADED -gt 0 ]]; then
        echo -e "  ${YELLOW}Degraded:${NC} $TESTS_DEGRADED"
    fi
    echo -e "  ${RED}Failed:${NC}  $TESTS_FAILED"
    echo -e "  ${YELLOW}Skipped:${NC} $TESTS_SKIPPED"
    echo "  ──────────────────────────────────────────────────────"
    echo "  Total:   $total"
    if [[ $TESTS_DEGRADED -gt 0 ]]; then
        echo "  Overall Success Rate: ${success_rate}% (includes degraded)"
        echo "  Healthy Success Rate: ${healthy_rate}%"
    else
        echo "  Success Rate: ${success_rate}%"
    fi
    echo "════════════════════════════════════════════════════════"
    echo
    
    # Return non-zero if any tests failed (degraded tests are still considered successful)
    [[ $TESTS_FAILED -eq 0 ]]
}

# Cleanup function
cleanup() {
    [[ "$VERBOSE" == "true" ]] && print_info "Cleaning up..."
    # Add cleanup logic here if needed
}

# Main execution
main() {
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Parse arguments
    parse_args "$@"
    
    # Print header
    echo
    echo "╔════════════════════════════════════════════════════════╗"
    echo "║           Scenario Test Runner v1.0                    ║"
    echo "╚════════════════════════════════════════════════════════╝"
    echo
    print_info "Testing scenario: $(basename "$SCENARIO_DIR")"
    [[ "$DRY_RUN" == "true" ]] && print_warning "DRY RUN MODE - No tests will be executed"
    echo
    
    # Load configuration
    load_config
    
    # Validate structure
    validate_structure
    local structure_result=$?
    [[ "$VERBOSE" == "true" ]] && print_info "Structure validation result: $structure_result"
    
    # Check resources
    check_resources
    local resources_result=$?
    [[ "$VERBOSE" == "true" ]] && print_info "Resource check result: $resources_result"
    
    # Run tests
    run_tests
    local run_tests_result=$?
    [[ "$VERBOSE" == "true" ]] && print_info "Tests completed with result: $run_tests_result. TESTS_PASSED=$TESTS_PASSED, TESTS_FAILED=$TESTS_FAILED"
    
    # Generate report and capture exit code
    generate_report
    local exit_code=$?
    [[ "$VERBOSE" == "true" ]] && print_info "Report generated with exit code: $exit_code"
    
    # Explicitly exit with the correct code
    exit $exit_code
}

# Execute main function
main "$@"