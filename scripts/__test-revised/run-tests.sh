#!/usr/bin/env bash
# Vrooli Testing Infrastructure - Main Orchestrator
# 
# Phase-based testing system that can run each testing phase independently
# or all phases sequentially. Designed for Vrooli's scenario-first architecture.
#
# Usage: ./run-tests.sh [PHASE] [OPTIONS]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"

show_usage() {
    cat << 'EOF'
Usage: ./run-tests.sh [PHASE] [OPTIONS]

PHASES:
  static      Run static analysis (shellcheck, bash -n) on all scripts/
  resources   Validate scripts/resources/ construction & test with mocks  
  scenarios   Validate scenarios & run integration tests for converted ones
  bats        Execute all BATS tests with caching mechanism
  all         Run all phases sequentially (default)

OPTIONS:
  --verbose       Show detailed output from all phases
  --parallel      Run tests in parallel where possible
  --no-cache      Skip caching optimizations
  --dry-run       Show what would be tested without running
  --help         Show this help

EXAMPLES:
  ./run-tests.sh                    # Run all phases
  ./run-tests.sh static             # Only static analysis
  ./run-tests.sh resources --verbose # Resource testing with details
  ./run-tests.sh scenarios --dry-run # Show what scenarios would be tested
  ./run-tests.sh bats --no-cache    # Run BATS without caching

Each phase can be run independently and will report its own results.
EOF
}

# Parse command line arguments
PHASE="all"
VERBOSE=""
PARALLEL=""
NO_CACHE=""
DRY_RUN=""

while [[ $# -gt 0 ]]; do
    case $1 in
        static|resources|scenarios|bats|all)
            PHASE="$1"
            shift
            ;;
        --verbose)
            VERBOSE="--verbose"
            shift
            ;;
        --parallel)
            PARALLEL="--parallel"
            shift
            ;;
        --no-cache)
            NO_CACHE="--no-cache"
            shift
            ;;
        --dry-run)
            DRY_RUN="--dry-run"
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Set environment variables for all phases
export PROJECT_ROOT
export SCRIPT_DIR
export VERBOSE
export PARALLEL
export NO_CACHE
export DRY_RUN

# Phase execution tracking
declare -A PHASE_RESULTS=()
TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0

run_phase() {
    local phase_name="$1"
    local phase_script="$SCRIPT_DIR/phases/test-$phase_name.sh"
    
    if [[ ! -x "$phase_script" ]]; then
        log_error "Phase script not found or not executable: $phase_script"
        return 1
    fi
    
    log_info "🚀 Starting $phase_name phase..."
    
    local start_time
    start_time=$(date +%s)
    
    # Build command with only non-empty arguments
    local cmd=("$phase_script")
    [[ -n "$VERBOSE" ]] && cmd+=("$VERBOSE")
    [[ -n "$PARALLEL" ]] && cmd+=("$PARALLEL")
    [[ -n "$NO_CACHE" ]] && cmd+=("$NO_CACHE")
    [[ -n "$DRY_RUN" ]] && cmd+=("$DRY_RUN")
    
    if "${cmd[@]}"; then
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        PHASE_RESULTS["$phase_name"]="PASSED (${duration}s)"
        log_success "✅ $phase_name phase completed in ${duration}s"
        return 0
    else
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        PHASE_RESULTS["$phase_name"]="FAILED (${duration}s)"
        log_error "❌ $phase_name phase failed after ${duration}s"
        return 1
    fi
}

# Main execution logic
main() {
    log_info "🔬 Vrooli Testing Infrastructure v2.0"
    log_info "📁 Project root: $PROJECT_ROOT"
    log_info "🎯 Phase: $PHASE"
    [[ -n "$VERBOSE" ]] && log_info "📝 Verbose mode enabled"
    [[ -n "$PARALLEL" ]] && log_info "🚀 Parallel execution enabled"
    [[ -n "$NO_CACHE" ]] && log_info "🚫 Cache disabled"
    [[ -n "$DRY_RUN" ]] && log_info "👀 Dry run mode - no actual execution"
    
    local overall_success=true
    
    case $PHASE in
        static)
            run_phase "static" || overall_success=false
            ;;
        resources)
            run_phase "resources" || overall_success=false
            ;;
        scenarios)
            run_phase "scenarios" || overall_success=false
            ;;
        bats)
            run_phase "bats" || overall_success=false
            ;;
        all)
            # Run all phases in sequence
            local phases=("static" "resources" "scenarios" "bats")
            
            for phase in "${phases[@]}"; do
                if ! run_phase "$phase"; then
                    overall_success=false
                    # Continue with other phases even if one fails
                fi
            done
            ;;
        *)
            log_error "Invalid phase: $PHASE"
            show_usage
            exit 1
            ;;
    esac
    
    # Report final results
    log_info ""
    log_info "📊 Test Results Summary:"
    log_info "========================"
    
    for phase in "${!PHASE_RESULTS[@]}"; do
        local result="${PHASE_RESULTS[$phase]}"
        if [[ "$result" =~ ^PASSED ]]; then
            log_success "  ✅ $phase: $result"
        else
            log_error "  ❌ $phase: $result"
        fi
    done
    
    if $overall_success; then
        log_success ""
        log_success "🎉 All requested phases completed successfully!"
        exit 0
    else
        log_error ""
        log_error "💥 One or more phases failed. Check the logs above for details."
        exit 1
    fi
}

# Trap to ensure cleanup on exit
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 ]] && [[ ${#PHASE_RESULTS[@]} -gt 0 ]]; then
        log_info ""
        log_info "🛑 Execution interrupted. Partial results:"
        for phase in "${!PHASE_RESULTS[@]}"; do
            local result="${PHASE_RESULTS[$phase]}"
            log_info "  $phase: $result"
        done
    fi
}

trap cleanup EXIT

# Run main function
main "$@"