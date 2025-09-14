#!/bin/bash
# Main test orchestrator for Visited Tracker scenario
# Runs phased testing with support for individual phases and test types
set -euo pipefail

# Setup paths and utilities
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh"

# Test configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PHASES_DIR="$TEST_DIR/phases"
SCENARIO_NAME="visited-tracker"

# Phase definitions with time limits (seconds)
declare -A PHASES
PHASES["structure"]="15"
PHASES["dependencies"]="30" 
PHASES["unit"]="60"
PHASES["integration"]="120"
PHASES["business"]="180"
PHASES["performance"]="60"

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
🧪 Visited Tracker Test Runner

USAGE:
    $0 [phases/types] [options]

PHASES:
    structure      Structure validation (<15s)
    dependencies   Resource and dependency checks (<30s)  
    unit          Unit tests for all languages (<60s)
    integration   API, database, CLI integration (<120s)
    business      End-to-end business logic (<180s)
    performance   Performance and load testing (<60s)
    all           All phases in sequence

TEST TYPES:
    go            Go unit tests only
    node          Node.js unit tests only
    python        Python unit tests only
    bats          CLI BATS tests only
    ui            UI automation tests only

QUICK MODES:
    quick         structure + dependencies + unit
    smoke         structure + dependencies only
    core          structure + dependencies + unit + integration

OPTIONS:
    --verbose, -v          Show detailed output for all phases
    --parallel, -p         Run tests in parallel where possible
    --timeout N            Multiply phase timeouts by N (default: 1)
    --dry-run, -n          Show what tests would run without executing
    --continue, -c         Continue testing even if a phase fails
    --junit                Output results in JUnit XML format
    --coverage             Generate coverage reports where applicable
    --help, -h             Show this help message

EXAMPLES:
    $0                     # Run all phases sequentially
    $0 structure           # Run only structure validation
    $0 unit integration    # Run unit and integration tests
    $0 go bats            # Run Go tests and CLI BATS tests
    $0 quick              # Quick feedback: structure + dependencies + unit
    $0 all --verbose      # All phases with detailed output
    $0 --parallel core    # Core phases in parallel
    $0 --dry-run all      # See what would be executed

PHASE TIME LIMITS:
$(for phase in "${PHASE_ORDER[@]}"; do
    printf "    %-12s %3ds\n" "$phase" "${PHASES[$phase]}"
done)

EOF
}

# Parse command line arguments
parse_args() {
    local args=()
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                show_usage
                exit 0
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --parallel|-p) 
                PARALLEL=true
                shift
                ;;
            --timeout)
                TIMEOUT_MULTIPLIER="$2"
                shift 2
                ;;
            --dry-run|-n)
                DRY_RUN=true
                shift
                ;;
            --continue|-c)
                CONTINUE_ON_FAILURE=true
                shift
                ;;
            --junit)
                JUNIT_OUTPUT=true
                shift
                ;;
            --coverage)
                COVERAGE=true
                shift
                ;;
            all)
                SELECTED_PHASES=("${PHASE_ORDER[@]}")
                shift
                ;;
            quick)
                SELECTED_PHASES=("structure" "dependencies" "unit")
                shift
                ;;
            smoke)
                SELECTED_PHASES=("structure" "dependencies")
                shift
                ;;
            core)
                SELECTED_PHASES=("structure" "dependencies" "unit" "integration")
                shift
                ;;
            structure|dependencies|unit|integration|business|performance)
                args+=("$1")
                shift
                ;;
            go|node|python|bats|ui)
                args+=("$1")
                shift
                ;;
            *)
                log::error "❌ Unknown argument: $1"
                echo ""
                show_usage
                exit 1
                ;;
        esac
    done
    
    # If no phases specified, run all
    if [ ${#args[@]} -eq 0 ] && [ ${#SELECTED_PHASES[@]} -eq 0 ]; then
        SELECTED_PHASES=("${PHASE_ORDER[@]}")
    elif [ ${#args[@]} -gt 0 ]; then
        SELECTED_PHASES=("${args[@]}")
    fi
}

# Run a single phase
run_phase() {
    local phase="$1"
    local phase_script="$PHASES_DIR/test-$phase.sh"
    local timeout_seconds="${PHASES[$phase]}"
    local actual_timeout=$((timeout_seconds * TIMEOUT_MULTIPLIER))
    
    log::info "🚀 Running Phase: $phase"
    log::info "   Script: $phase_script"
    log::info "   Timeout: ${actual_timeout}s"
    
    if [ "$DRY_RUN" = "true" ]; then
        log::warning "   [DRY RUN] Would execute: $phase_script"
        return 0
    fi
    
    if [ ! -f "$phase_script" ]; then
        log::error "❌ Phase script not found: $phase_script"
        return 1
    fi
    
    if [ ! -x "$phase_script" ]; then
        log::error "❌ Phase script not executable: $phase_script"
        return 1
    fi
    
    local start_time phase_start_time
    phase_start_time=$(date +%s)
    
    # Run phase with timeout
    local phase_result=0
    if [ "$VERBOSE" = "true" ]; then
        timeout "${actual_timeout}s" "$phase_script" || phase_result=$?
    else
        timeout "${actual_timeout}s" "$phase_script" 2>/dev/null || phase_result=$?
    fi
    
    local phase_end_time phase_duration
    phase_end_time=$(date +%s)
    phase_duration=$((phase_end_time - phase_start_time))
    
    # Analyze results
    case $phase_result in
        0)
            log::success "✅ Phase '$phase' passed in ${phase_duration}s"
            ((passed_phases++))
            ;;
        124)
            log::error "❌ Phase '$phase' timed out after ${actual_timeout}s"
            ((failed_phases++))
            ;;
        *)
            log::error "❌ Phase '$phase' failed with exit code $phase_result in ${phase_duration}s"
            ((failed_phases++))
            ;;
    esac
    
    total_duration=$((total_duration + phase_duration))
    ((total_phases++))
    
    echo ""
    return $phase_result
}

# Run a specific test type
run_test_type() {
    local test_type="$1"
    
    log::info "🎯 Running Test Type: $test_type"
    
    case $test_type in
        go)
            if [ -x "$TEST_DIR/unit/go.sh" ]; then
                log::info "   Executing: $TEST_DIR/unit/go.sh"
                [ "$DRY_RUN" = "true" ] || "$TEST_DIR/unit/go.sh"
            else
                log::warning "⚠️  Go test runner not found or not executable"
                return 1
            fi
            ;;
        node)
            if [ -x "$TEST_DIR/unit/node.sh" ]; then
                log::info "   Executing: $TEST_DIR/unit/node.sh"
                [ "$DRY_RUN" = "true" ] || "$TEST_DIR/unit/node.sh"
            else
                log::warning "⚠️  Node.js test runner not found or not executable"
                return 1
            fi
            ;;
        python)
            if [ -x "$TEST_DIR/unit/python.sh" ]; then
                log::info "   Executing: $TEST_DIR/unit/python.sh"
                [ "$DRY_RUN" = "true" ] || "$TEST_DIR/unit/python.sh"
            else
                log::warning "⚠️  Python test runner not found or not executable"
                return 1
            fi
            ;;
        bats)
            if [ -x "$TEST_DIR/cli/run-cli-tests.sh" ]; then
                log::info "   Executing: $TEST_DIR/cli/run-cli-tests.sh"
                [ "$DRY_RUN" = "true" ] || "$TEST_DIR/cli/run-cli-tests.sh"
            else
                log::warning "⚠️  BATS test runner not found or not executable"
                return 1
            fi
            ;;
        ui)
            if [ -x "$TEST_DIR/ui/run-ui-tests.sh" ]; then
                log::info "   Executing: $TEST_DIR/ui/run-ui-tests.sh"
                [ "$DRY_RUN" = "true" ] || "$TEST_DIR/ui/run-ui-tests.sh"
            else
                log::warning "⚠️  UI test runner not found"
                log::info "💡 Consider adding UI automation tests using browser-automation-studio"
                return 1
            fi
            ;;
        *)
            log::error "❌ Unknown test type: $test_type"
            return 1
            ;;
    esac
    
    echo ""
    return 0
}

# Run tests in parallel (basic implementation)
run_parallel() {
    local items=("$@")
    local pids=()
    local results=()
    
    log::info "🚀 Running ${#items[@]} items in parallel..."
    
    for item in "${items[@]}"; do
        (
            if [[ " ${PHASE_ORDER[*]} " =~ " $item " ]]; then
                run_phase "$item"
            else
                run_test_type "$item"
            fi
        ) &
        pids+=($!)
    done
    
    # Wait for all background processes
    local overall_result=0
    for i in "${!pids[@]}"; do
        local pid="${pids[$i]}"
        local item="${items[$i]}"
        
        if wait "$pid"; then
            log::success "✅ Parallel item '$item' completed"
        else
            log::error "❌ Parallel item '$item' failed"
            overall_result=1
        fi
    done
    
    return $overall_result
}

# Generate JUnit XML output
generate_junit() {
    local junit_file="$TEST_DIR/junit-results.xml"
    local timestamp
    timestamp=$(date -Iseconds)
    
    cat > "$junit_file" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="visited-tracker-tests" tests="$total_phases" failures="$failed_phases" skipped="$skipped_phases" time="$total_duration" timestamp="$timestamp">
EOF
    
    for phase in "${SELECTED_PHASES[@]}"; do
        if [[ " ${PHASE_ORDER[*]} " =~ " $phase " ]]; then
            cat >> "$junit_file" << EOF
    <testcase classname="visited-tracker" name="phase-$phase" time="0">
    </testcase>
EOF
        fi
    done
    
    cat >> "$junit_file" << EOF
</testsuite>
EOF
    
    log::info "📄 JUnit results written to: $junit_file"
}

# Main execution function
main() {
    local overall_start_time
    overall_start_time=$(date +%s)
    
    log::header "🧪 Visited Tracker Comprehensive Test Suite"
    log::info "   Test directory: $TEST_DIR"
    log::info "   Selected phases/types: ${SELECTED_PHASES[*]}"
    log::info "   Options: verbose=$VERBOSE, parallel=$PARALLEL, dry-run=$DRY_RUN"
    echo ""
    
    # Validate test infrastructure
    if [ ! -d "$PHASES_DIR" ]; then
        log::error "❌ Phases directory not found: $PHASES_DIR"
        exit 1
    fi
    
    # Execute tests
    local execution_result=0
    
    if [ "$PARALLEL" = "true" ]; then
        # Run in parallel (experimental)
        if ! run_parallel "${SELECTED_PHASES[@]}"; then
            execution_result=1
        fi
    else
        # Run sequentially
        for item in "${SELECTED_PHASES[@]}"; do
            local item_result=0
            
            if [[ " ${PHASE_ORDER[*]} " =~ " $item " ]]; then
                if ! run_phase "$item"; then
                    item_result=1
                fi
            else
                if ! run_test_type "$item"; then
                    item_result=1
                fi
            fi
            
            # Handle failures
            if [ $item_result -ne 0 ]; then
                execution_result=1
                if [ "$CONTINUE_ON_FAILURE" = "false" ]; then
                    log::error "❌ Stopping execution due to failure in: $item"
                    break
                else
                    log::warning "⚠️  Continuing despite failure in: $item"
                fi
            fi
        done
    fi
    
    # Final summary
    local overall_end_time overall_duration
    overall_end_time=$(date +%s)
    overall_duration=$((overall_end_time - overall_start_time))
    
    echo ""
    echo -e "📊 Test Execution Summary"
    echo "═══════════════════════════════════════════"
    echo -e "   Total phases run: $total_phases"
    echo -e "   Phases passed: $passed_phases"
    echo -e "   Phases failed: $failed_phases"
    echo -e "   Phases skipped: $skipped_phases"
    echo -e "   Total duration: ${overall_duration}s"
    echo ""
    
    # Generate reports if requested
    if [ "$JUNIT_OUTPUT" = "true" ]; then
        generate_junit
    fi
    
    if [ "$COVERAGE" = "true" ]; then
        log::info "📈 Coverage reports may be available in:"
        echo "   • Go: api/coverage.html"
        echo "   • Node.js: ui/coverage/lcov-report/index.html"
        echo "   • Python: htmlcov/index.html"
    fi
    
    # Final status
    if [ $execution_result -eq 0 ] && [ $failed_phases -eq 0 ]; then
        log::success "🎉 All tests passed successfully!"
        log::success "✅ Visited Tracker testing infrastructure is working correctly"
    elif [ $failed_phases -eq 0 ]; then
        log::success "✅ Test execution completed (with some skipped)"
    else
        log::error "❌ Test execution completed with failures"
        log::error "   $failed_phases out of $total_phases phases failed"
        echo ""
        log::info "💡 Troubleshooting tips:"
        echo "   • Ensure visited-tracker scenario is running: vrooli scenario start visited-tracker"
        echo "   • Check dependencies: vrooli resource start postgres"
        echo "   • Install CLI: cd cli && ./install.sh"
        echo "   • Run individual phases for detailed debugging"
    fi
    
    echo ""
    log::info "📚 For more information, see:"
    echo "   • docs/scenarios/PHASED_TESTING_ARCHITECTURE.md"
    echo "   • Test files in: $TEST_DIR"
    
    exit $execution_result
}

# Parse arguments and run
parse_args "$@"
main