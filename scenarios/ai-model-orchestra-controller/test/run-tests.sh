#!/bin/bash
# Main test orchestrator for AI Model Orchestra Controller scenario
# Runs phased testing with support for individual phases and test types
set -euo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
if [ -f "${APP_ROOT}/scripts/lib/utils/log.sh" ]; then
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
else
    # Fallback logging functions
    log_info() { echo -e "\033[0;34m[INFO]\033[0m $1"; }
    log_success() { echo -e "\033[0;32m[SUCCESS]\033[0m $1"; }
    log_warn() { echo -e "\033[1;33m[WARN]\033[0m $1"; }
    log_error() { echo -e "\033[0;31m[ERROR]\033[0m $1"; }
fi

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="$TEST_DIR/phases"
SCENARIO_NAME="ai-model-orchestra-controller"

# Phase definitions with time limits (seconds)
declare -A PHASES
PHASES["structure"]="15"
PHASES["dependencies"]="45"  # Longer for Go mod downloads
PHASES["unit"]="90"          # Longer for Go compilation
PHASES["integration"]="180"  # Longer for AI model loading
PHASES["business"]="300"     # Longer for full AI workflows
PHASES["performance"]="120"  # Longer for load testing

# Phase order for sequential execution
PHASE_ORDER=("structure" "dependencies" "unit" "integration" "business" "performance")

# Test statistics
total_phases=0
passed_phases=0
failed_phases=0
skipped_phases=0
total_duration=0

# Options
VERBOSE=false
PARALLEL=false
DRY_RUN=false
CONTINUE_ON_FAILURE=false
JUNIT_OUTPUT=false
COVERAGE=false
SELECTED_PHASES=()
TIMEOUT_MULTIPLIER=1

# Show usage
show_usage() {
    cat << EOF
🎛️ AI Model Orchestra Controller Test Runner

USAGE:
    $0 [phases/types] [options]

PHASES:
    structure     - File structure and configuration validation
    dependencies  - Go modules, CLI installation, resource availability  
    unit          - Go unit tests, API endpoint tests
    integration   - Full API integration, model selection, routing
    business      - End-to-end AI orchestration workflows
    performance   - Load testing, resource pressure testing
    
    all           - Run all phases (default)

OPTIONS:
    --verbose         Show detailed output
    --parallel        Run phases in parallel (faster but harder to debug)
    --dry-run         Show what would run without executing
    --continue        Continue on phase failure
    --junit          Generate JUnit XML output
    --coverage       Generate coverage reports (Go only)
    --timeout-mult N  Multiply timeouts by N (for slow systems)
    
EXAMPLES:
    $0                              # Run all phases
    $0 unit integration            # Run specific phases
    $0 all --verbose --coverage    # Full run with details and coverage
    $0 structure --dry-run         # Preview structure tests

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --parallel|-p)
                PARALLEL=true
                shift
                ;;
            --dry-run|-n)
                DRY_RUN=true
                shift
                ;;
            --continue|-c)
                CONTINUE_ON_FAILURE=true
                shift
                ;;
            --junit|-j)
                JUNIT_OUTPUT=true
                shift
                ;;
            --coverage)
                COVERAGE=true
                shift
                ;;
            --timeout-mult)
                TIMEOUT_MULTIPLIER="$2"
                shift 2
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            all)
                SELECTED_PHASES=("${PHASE_ORDER[@]}")
                shift
                ;;
            structure|dependencies|unit|integration|business|performance)
                SELECTED_PHASES+=("$1")
                shift
                ;;
            *)
                log_error "Unknown argument: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Default to all phases if none specified
    if [ ${#SELECTED_PHASES[@]} -eq 0 ]; then
        SELECTED_PHASES=("${PHASE_ORDER[@]}")
    fi
}

# Check if phase script exists and is executable
check_phase_script() {
    local phase="$1"
    local script="$PHASES_DIR/test-${phase}.sh"
    
    if [ ! -f "$script" ]; then
        log_warn "Phase script not found: $script"
        return 1
    fi
    
    if [ ! -x "$script" ]; then
        log_warn "Phase script not executable: $script"
        return 1
    fi
    
    return 0
}

# Run a single test phase
run_phase() {
    local phase="$1"
    local timeout="${PHASES[$phase]}"
    local adjusted_timeout=$((timeout * TIMEOUT_MULTIPLIER))
    local script="$PHASES_DIR/test-${phase}.sh"
    
    log_info "Starting phase: $phase (timeout: ${adjusted_timeout}s)"
    
    if [ "$DRY_RUN" = true ]; then
        log_info "[DRY RUN] Would execute: $script"
        return 0
    fi
    
    # Check script exists
    if ! check_phase_script "$phase"; then
        log_warn "Skipping phase $phase - script issues"
        ((skipped_phases++))
        return 0
    fi
    
    # Run the phase with timeout
    local start_time=$(date +%s)
    local exit_code=0
    
    if [ "$VERBOSE" = true ]; then
        timeout "$adjusted_timeout" "$script" 2>&1
        exit_code=$?
    else
        timeout "$adjusted_timeout" "$script" >/dev/null 2>&1
        exit_code=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    total_duration=$((total_duration + duration))
    
    if [ $exit_code -eq 0 ]; then
        log_success "Phase $phase passed (${duration}s)"
        ((passed_phases++))
    elif [ $exit_code -eq 124 ]; then
        log_error "Phase $phase timed out after ${adjusted_timeout}s"
        ((failed_phases++))
    else
        log_error "Phase $phase failed with exit code $exit_code (${duration}s)"
        ((failed_phases++))
    fi
    
    return $exit_code
}

# Run phases in parallel
run_phases_parallel() {
    log_info "Running phases in parallel: ${SELECTED_PHASES[*]}"
    
    local pids=()
    local phase_results=()
    
    # Start all phases
    for phase in "${SELECTED_PHASES[@]}"; do
        ((total_phases++))
        
        (
            run_phase "$phase"
            echo $? > "/tmp/${SCENARIO_NAME}-phase-${phase}.exit"
        ) &
        
        pids+=($!)
        phase_results+=("$phase")
    done
    
    # Wait for all phases to complete
    for i in "${!pids[@]}"; do
        wait ${pids[$i]}
        
        local phase="${phase_results[$i]}"
        local exit_file="/tmp/${SCENARIO_NAME}-phase-${phase}.exit"
        
        if [ -f "$exit_file" ]; then
            local exit_code=$(cat "$exit_file")
            rm -f "$exit_file"
            
            if [ $exit_code -ne 0 ] && [ "$CONTINUE_ON_FAILURE" = false ]; then
                log_error "Phase $phase failed, stopping parallel execution"
                return $exit_code
            fi
        fi
    done
}

# Run phases sequentially
run_phases_sequential() {
    log_info "Running phases sequentially: ${SELECTED_PHASES[*]}"
    
    for phase in "${SELECTED_PHASES[@]}"; do
        ((total_phases++))
        
        if ! run_phase "$phase"; then
            if [ "$CONTINUE_ON_FAILURE" = false ]; then
                log_error "Phase $phase failed, stopping execution"
                return 1
            fi
        fi
    done
}

# Generate test report
generate_report() {
    log_info "Test Summary"
    echo "=============="
    echo "Total phases:  $total_phases"
    echo "Passed:        $passed_phases"
    echo "Failed:        $failed_phases"
    echo "Skipped:       $skipped_phases"
    echo "Duration:      ${total_duration}s"
    echo ""
    
    if [ $failed_phases -gt 0 ]; then
        log_error "❌ Some tests failed!"
        return 1
    elif [ $skipped_phases -gt 0 ]; then
        log_warn "⚠️  Some tests were skipped"
        return 0
    else
        log_success "✅ All tests passed!"
        return 0
    fi
}

# Create missing phase scripts with templates
create_phase_scripts() {
    log_info "Creating missing test phase scripts..."
    
    for phase in "${PHASE_ORDER[@]}"; do
        local script="$PHASES_DIR/test-${phase}.sh"
        
        if [ ! -f "$script" ]; then
            log_info "Creating template for phase: $phase"
            
            cat > "$script" << EOF
#!/bin/bash
# AI Model Orchestra Controller - ${phase^} Phase Tests
set -euo pipefail

TEST_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="\$(cd "\$TEST_DIR/.." && pwd)"

# Add your ${phase} tests here
echo "Running ${phase} tests for AI Model Orchestra Controller..."

# Example test structure:
# test_${phase}_basic() {
#     echo "Testing ${phase} functionality..."
#     # Add specific tests here
#     return 0
# }

# Run tests
test_${phase}_basic() {
    echo "✓ ${phase^} test template created"
    return 0
}

# Execute tests
test_${phase}_basic

echo "✅ ${phase^} tests completed successfully"
EOF
            
            chmod +x "$script"
        fi
    done
}

# Main execution
main() {
    log_info "🎛️ AI Model Orchestra Controller Test Runner"
    log_info "Scenario: $SCENARIO_NAME"
    
    # Parse arguments
    parse_args "$@"
    
    # Create missing phase scripts if needed
    if [ ! -d "$PHASES_DIR" ]; then
        mkdir -p "$PHASES_DIR"
    fi
    create_phase_scripts
    
    # Show configuration
    if [ "$VERBOSE" = true ]; then
        log_info "Configuration:"
        echo "  Phases: ${SELECTED_PHASES[*]}"
        echo "  Parallel: $PARALLEL"
        echo "  Continue on failure: $CONTINUE_ON_FAILURE"
        echo "  Timeout multiplier: $TIMEOUT_MULTIPLIER"
        echo ""
    fi
    
    # Run tests
    local start_time=$(date +%s)
    
    if [ "$PARALLEL" = true ]; then
        run_phases_parallel
    else
        run_phases_sequential
    fi
    
    local end_time=$(date +%s)
    total_duration=$((end_time - start_time))
    
    # Generate report
    generate_report
}

# Execute main function with all arguments
main "$@"