#!/bin/bash
# Run All Scenarios Test Runner
# Orchestrates testing of all scenarios in the scenarios/ directory

set -euo pipefail

# Colors are available from log.sh via var.sh

# Configuration
QUICK_MODE=false
VERBOSE=false
PARALLEL=false
SCENARIOS_DIR=""
FILTER=""
EXIT_ON_FIRST_FAILURE=false

# Counters
SCENARIOS_PASSED=0
SCENARIOS_FAILED=0
SCENARIOS_DEGRADED=0
SCENARIOS_SKIPPED=0

# Results tracking
declare -a FAILED_SCENARIOS=()
declare -a DEGRADED_SCENARIOS=()
declare -a PASSED_SCENARIOS=()

# Resolve script location
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/scenarios/validation"
SCENARIOS_ROOT="${APP_ROOT}/scripts/scenarios"

# Source required utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

# Print functions
print_info() {
    echo -e "${BLUE}[INFO]${RESET} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${RESET} $1"
}

print_error() {
    echo -e "${RED}[✗]${RESET} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}[!]${RESET} $1"
}

print_header() {
    echo -e "\n${BLUE}╔════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${BLUE}║$1${RESET}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════╝${RESET}"
}

# Show usage
show_help() {
    cat << EOF
Run All Scenarios Test Runner

Usage: $(basename "$0") [OPTIONS]

Options:
    --quick             Skip slow tests and use shorter timeouts
    --verbose, -v       Verbose output from individual scenario tests
    --parallel, -p      Run scenarios in parallel (experimental)
    --scenarios <dir>   Override scenarios directory (default: ../core)
    --filter <pattern>  Only run scenarios matching pattern (e.g., "multi-modal*")
    --fail-fast         Exit on first scenario failure
    --help, -h          Show this help message

Examples:
    # Run all scenarios
    $(basename "$0")
    
    # Quick run with shorter timeouts
    $(basename "$0") --quick
    
    # Run only multi-modal scenarios
    $(basename "$0") --filter "multi-modal*"
    
    # Verbose output with fail-fast
    $(basename "$0") --verbose --fail-fast

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --quick)
                QUICK_MODE=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --parallel|-p)
                PARALLEL=true
                shift
                ;;
            --scenarios)
                SCENARIOS_DIR="$2"
                shift 2
                ;;
            --filter)
                FILTER="$2"
                shift 2
                ;;
            --fail-fast)
                EXIT_ON_FIRST_FAILURE=true
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
    
    # Default scenarios directory
    SCENARIOS_DIR="${SCENARIOS_DIR:-$SCENARIOS_ROOT/core}"
    
    if [[ ! -d "$SCENARIOS_DIR" ]]; then
        print_error "Scenarios directory not found: $SCENARIOS_DIR"
        exit 1
    fi
}

# Get list of scenarios to test
get_scenarios() {
    local scenarios=()
    
    if [[ -n "$FILTER" ]]; then
        # Use shell globbing for pattern matching
        while IFS= read -r -d '' scenario_dir; do
            local scenario_name="$(basename "$scenario_dir")"
            if [[ "$scenario_name" == $FILTER ]]; then
                scenarios+=("$scenario_dir")
            fi
        done < <(find "$SCENARIOS_DIR" -mindepth 1 -maxdepth 1 -type d -print0)
    else
        # Get all scenario directories
        while IFS= read -r -d '' scenario_dir; do
            scenarios+=("$scenario_dir")
        done < <(find "$SCENARIOS_DIR" -mindepth 1 -maxdepth 1 -type d -print0)
    fi
    
    # Sort scenarios for consistent ordering
    IFS=$'\n' scenarios=($(sort <<<"${scenarios[*]}"))
    
    printf '%s\n' "${scenarios[@]}"
}

# Check if scenario has test files
has_test_files() {
    local scenario_dir="$1"
    
    # Check for test.sh or scenario-test.yaml
    [[ -f "$scenario_dir/test.sh" ]] || [[ -f "$scenario_dir/scenario-test.yaml" ]]
}

# Run a single scenario test
run_scenario_test() {
    local scenario_dir="$1"
    local scenario_name="$(basename "$scenario_dir")"
    
    print_info "Testing scenario: $scenario_name"
    
    # Check if scenario has test files
    if ! has_test_files "$scenario_dir"; then
        print_warning "No test files found in $scenario_name, skipping"
        SCENARIOS_SKIPPED=$((SCENARIOS_SKIPPED + 1))
        return 0
    fi
    
    # Prepare test command
    local test_cmd=""
    local test_args=()
    
    if [[ -f "$scenario_dir/test.sh" ]]; then
        test_cmd="$scenario_dir/test.sh"
    else
        # Fallback to direct framework runner
        test_cmd="$SCRIPT_DIR/scenario-test-runner.sh"
        test_args+=("--scenario" "$scenario_dir")
        
        if [[ -f "$scenario_dir/scenario-test.yaml" ]]; then
            test_args+=("--config" "scenario-test.yaml")
        fi
    fi
    
    # Add common arguments
    if [[ "$VERBOSE" == "true" ]]; then
        test_args+=("--verbose")
    fi
    
    if [[ "$QUICK_MODE" == "true" ]]; then
        test_args+=("--quick")  # Add quick mode flag
    fi
    
    # Run the test
    local start_time=$(date +%s)
    local test_output=""
    local exit_code=0
    
    if test_output=$("$test_cmd" "${test_args[@]}" 2>&1); then
        exit_code=0
    else
        exit_code=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Process results
    case $exit_code in
        0)
            print_success "$scenario_name (${duration}s)"
            SCENARIOS_PASSED=$((SCENARIOS_PASSED + 1))
            PASSED_SCENARIOS+=("$scenario_name")
            ;;
        2)
            print_warning "$scenario_name - degraded functionality (${duration}s)"
            SCENARIOS_DEGRADED=$((SCENARIOS_DEGRADED + 1))
            DEGRADED_SCENARIOS+=("$scenario_name")
            ;;
        *)
            print_error "$scenario_name - failed (${duration}s)"
            SCENARIOS_FAILED=$((SCENARIOS_FAILED + 1))
            FAILED_SCENARIOS+=("$scenario_name")
            
            # Show test output on failure if not verbose
            if [[ "$VERBOSE" != "true" ]]; then
                echo "--- Test Output ---"
                echo "$test_output"
                echo "--- End Output ---"
            fi
            ;;
    esac
    
    # Exit early if fail-fast is enabled
    if [[ "$EXIT_ON_FIRST_FAILURE" == "true" && $exit_code -ne 0 && $exit_code -ne 2 ]]; then
        print_error "Exiting due to --fail-fast and test failure"
        exit 1
    fi
    
    return $exit_code
}

# Run scenarios in parallel
run_scenarios_parallel() {
    local scenarios=("$@")
    local pids=()
    local results_dir=$(mktemp -d)
    
    print_info "Running ${#scenarios[@]} scenarios in parallel..."
    
    # Start all scenario tests in background
    for scenario_dir in "${scenarios[@]}"; do
        local scenario_name="$(basename "$scenario_dir")"
        local result_file="$results_dir/$scenario_name"
        
        (
            run_scenario_test "$scenario_dir" > "$result_file.log" 2>&1
            echo $? > "$result_file.exit"
        ) &
        
        pids+=($!)
    done
    
    # Wait for all tests to complete
    for pid in "${pids[@]}"; do
        wait "$pid"
    done
    
    # Process results
    for scenario_dir in "${scenarios[@]}"; do
        local scenario_name="$(basename "$scenario_dir")"
        local result_file="$results_dir/$scenario_name"
        local exit_code=0
        
        if [[ -f "$result_file.exit" ]]; then
            exit_code=$(cat "$result_file.exit")
        fi
        
        if [[ -f "$result_file.log" ]]; then
            cat "$result_file.log"
        fi
        
        # Update counters based on exit code
        case $exit_code in
            0)
                SCENARIOS_PASSED=$((SCENARIOS_PASSED + 1))
                PASSED_SCENARIOS+=("$scenario_name")
                ;;
            2)
                SCENARIOS_DEGRADED=$((SCENARIOS_DEGRADED + 1))
                DEGRADED_SCENARIOS+=("$scenario_name")
                ;;
            *)
                SCENARIOS_FAILED=$((SCENARIOS_FAILED + 1))
                FAILED_SCENARIOS+=("$scenario_name")
                ;;
        esac
    done
    
    # Cleanup
    trash::safe_remove "$results_dir" --no-confirm
}

# Run scenarios sequentially
run_scenarios_sequential() {
    local scenarios=("$@")
    
    print_info "Running ${#scenarios[@]} scenarios sequentially..."
    
    for scenario_dir in "${scenarios[@]}"; do
        run_scenario_test "$scenario_dir"
        echo  # Add spacing between scenarios
    done
}

# Generate final report
generate_report() {
    local total=$((SCENARIOS_PASSED + SCENARIOS_FAILED + SCENARIOS_DEGRADED + SCENARIOS_SKIPPED))
    
    print_header "                    Final Test Report                     "
    
    echo -e "  ${GREEN}Passed:${RESET}    $SCENARIOS_PASSED"
    if [[ $SCENARIOS_DEGRADED -gt 0 ]]; then
        echo -e "  ${YELLOW}Degraded:${RESET}  $SCENARIOS_DEGRADED"
    fi
    echo -e "  ${RED}Failed:${RESET}    $SCENARIOS_FAILED"
    echo -e "  ${YELLOW}Skipped:${RESET}   $SCENARIOS_SKIPPED"
    echo "  ──────────────────────────────────────────────────────"
    echo "  Total:     $total"
    
    if [[ $total -gt 0 ]]; then
        local success_rate=$(( (SCENARIOS_PASSED + SCENARIOS_DEGRADED) * 100 / total ))
        echo "  Success Rate: ${success_rate}%"
    fi
    
    # Show failed scenarios
    if [[ ${#FAILED_SCENARIOS[@]} -gt 0 ]]; then
        echo
        echo -e "${RED}Failed Scenarios:${RESET}"
        for scenario in "${FAILED_SCENARIOS[@]}"; do
            echo "  - $scenario"
        done
    fi
    
    # Show degraded scenarios
    if [[ ${#DEGRADED_SCENARIOS[@]} -gt 0 ]]; then
        echo
        echo -e "${YELLOW}Degraded Scenarios:${RESET}"
        for scenario in "${DEGRADED_SCENARIOS[@]}"; do
            echo "  - $scenario"
        done
    fi
    
    # Show passed scenarios if verbose
    if [[ "$VERBOSE" == "true" && ${#PASSED_SCENARIOS[@]} -gt 0 ]]; then
        echo
        echo -e "${GREEN}Passed Scenarios:${RESET}"
        for scenario in "${PASSED_SCENARIOS[@]}"; do
            echo "  - $scenario"
        done
    fi
    
    echo
    
    # Determine exit code
    if [[ $SCENARIOS_FAILED -gt 0 ]]; then
        print_error "Some scenarios failed"
        return 1
    elif [[ $SCENARIOS_DEGRADED -gt 0 ]]; then
        print_warning "All scenarios passed but some have degraded functionality"
        return 0
    else
        print_success "All scenarios passed successfully"
        return 0
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    print_header "           Vrooli Scenarios Test Runner                   "
    echo
    print_info "Scenarios directory: $SCENARIOS_DIR"
    if [[ -n "$FILTER" ]]; then
        print_info "Filter pattern: $FILTER"
    fi
    print_info "Mode: $(if [[ "$QUICK_MODE" == "true" ]]; then echo "Quick"; else echo "Full"; fi)"
    print_info "Execution: $(if [[ "$PARALLEL" == "true" ]]; then echo "Parallel"; else echo "Sequential"; fi)"
    echo
    
    # Get scenarios to test
    local scenarios
    readarray -t scenarios < <(get_scenarios)
    
    if [[ ${#scenarios[@]} -eq 0 ]]; then
        print_warning "No scenarios found to test"
        exit 0
    fi
    
    print_info "Found ${#scenarios[@]} scenario(s) to test"
    echo
    
    # Run scenarios
    local start_time=$(date +%s)
    
    if [[ "$PARALLEL" == "true" ]]; then
        run_scenarios_parallel "${scenarios[@]}"
    else
        run_scenarios_sequential "${scenarios[@]}"
    fi
    
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    
    echo
    print_info "Total execution time: ${total_duration}s"
    echo
    
    # Generate and return with report exit code
    generate_report
}

# Execute main function
main "$@"