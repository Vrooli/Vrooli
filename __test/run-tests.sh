#!/usr/bin/env bash
# Vrooli Testing Infrastructure - Main Orchestrator
# 
# Phase-based testing system that can run each testing phase independently
# or all phases sequentially. Designed for Vrooli's scenario-first architecture.
#
# Usage: ./run-tests.sh [PHASE] [OPTIONS]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"

show_usage() {
    cat << 'EOF'
Usage: ./run-tests.sh [PHASE] [OPTIONS]

PHASES:
  static        Run static analysis (shellcheck, TypeScript, Python, Go) on all files
  structure     Validate file/directory structures and configurations  
  integration   Run integration tests (resource mocks, app testing, etc.)
  unit          Execute all unit tests (BATS) with caching mechanism
  docs          Validate markdown documentation (syntax and file references)
  all           Run all phases sequentially (default)

OPTIONS:
  --verbose       Show detailed output from all phases
  --parallel      Run tests in parallel where possible
  --no-cache      Skip caching optimizations
  --dry-run       Show what would be tested without running
  --clear-cache   Clear test cache before running
  --help         Show this help

EXAMPLES:
  ./run-tests.sh                      # Run all phases
  ./run-tests.sh static               # Only static analysis
  ./run-tests.sh structure --verbose  # Structure validation with details
  ./run-tests.sh integration --dry-run # Show what integration tests would run
  ./run-tests.sh unit --no-cache      # Run unit tests without caching
  ./run-tests.sh docs --verbose       # Documentation validation with details

Each phase can be run independently and will report its own results.
EOF
}

# Parse command line arguments
PHASE="all"
VERBOSE=""
PARALLEL=""
NO_CACHE=""
DRY_RUN=""
CLEAR_CACHE=""
PHASE_ARGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        static|structure|integration|unit|docs|all)
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
        --clear-cache)
            CLEAR_CACHE="--clear-cache"
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            # If we have a phase selected, treat remaining args as phase-specific
            if [[ "$PHASE" != "all" ]]; then
                PHASE_ARGS+=("$1")
                shift
            else
                log_error "Unknown option: $1"
                show_usage
                exit 1
            fi
            ;;
    esac
done

# Set environment variables for all phases - ensure they have values
# This prevents issues with empty variable exports
VERBOSE="${VERBOSE:-}"
PARALLEL="${PARALLEL:-}"
NO_CACHE="${NO_CACHE:-}"
DRY_RUN="${DRY_RUN:-}"
CLEAR_CACHE="${CLEAR_CACHE:-}"

export PROJECT_ROOT
export SCRIPT_DIR
export VERBOSE
export PARALLEL
export NO_CACHE
export DRY_RUN
export CLEAR_CACHE

# Also export these for backward compatibility
export LOG_LEVEL="${LOG_LEVEL:-INFO}"
export TEST_CACHE_TTL="${TEST_CACHE_TTL:-3600}"
export TEST_CACHE_MAX_AGE="${TEST_CACHE_MAX_AGE:-86400}"

# Ensure all test framework scripts have proper permissions
ensure_script_permissions() {
    local scripts_to_check=(
        "$SCRIPT_DIR/phases/test-static.sh"
        "$SCRIPT_DIR/phases/test-structure.sh"
        "$SCRIPT_DIR/phases/test-integration.sh"
        "$SCRIPT_DIR/phases/test-unit.sh"
        "$SCRIPT_DIR/phases/test-docs.sh"
        "$SCRIPT_DIR/shared/logging.bash"
        "$SCRIPT_DIR/shared/test-helpers.bash"
        "$SCRIPT_DIR/shared/cache.bash"
    )
    
    local permission_issues=false
    
    for script in "${scripts_to_check[@]}"; do
        if [[ ! -f "$script" ]]; then
            log_error "Required script missing: $script"
            permission_issues=true
            continue
        fi
        
        # Check if file is readable
        if [[ ! -r "$script" ]]; then
            log_error "Script not readable: $script"
            permission_issues=true
            continue
        fi
        
        # For phase scripts, ensure they're executable
        if [[ "$script" =~ /phases/test-.*\.sh$ ]]; then
            if [[ ! -x "$script" ]]; then
                log_warning "Phase script not executable, attempting to fix: $script"
                if chmod +x "$script" 2>/dev/null; then
                    log_success "Fixed permissions for: $script"
                else
                    log_error "Cannot make script executable: $script"
                    permission_issues=true
                fi
            fi
        fi
        
        # For shared scripts, ensure they're readable (they're sourced, not executed)
        if [[ "$script" =~ /shared/.*\.bash$ ]]; then
            if [[ ! -r "$script" ]]; then
                log_error "Shared script not readable: $script"
                permission_issues=true
            fi
        fi
    done
    
    # Check cache directory permissions
    local cache_dir="$SCRIPT_DIR/cache"
    if [[ -d "$cache_dir" ]]; then
        if [[ ! -w "$cache_dir" ]]; then
            log_error "Cache directory not writable: $cache_dir"
            permission_issues=true
        fi
    else
        # Try to create cache directory
        if ! mkdir -p "$cache_dir" 2>/dev/null; then
            log_error "Cannot create cache directory: $cache_dir"
            permission_issues=true
        fi
    fi
    
    if [[ "$permission_issues" == "true" ]]; then
        log_error "Permission issues detected. Fix these issues before running tests."
        return 1
    fi
    
    return 0
}

# Phase execution tracking
declare -A PHASE_RESULTS=()
TOTAL_TESTS=0
TOTAL_PASSED=0
TOTAL_FAILED=0

run_phase() {
    local phase_name="$1"
    shift  # Remove phase_name, remaining args are phase-specific
    local phase_args=("$@")
    local phase_script="$SCRIPT_DIR/phases/test-$phase_name.sh"
    
    # Double-check permissions before running (in case they changed)
    if [[ ! -f "$phase_script" ]]; then
        log_error "Phase script not found: $phase_script"
        return 1
    fi
    
    if [[ ! -x "$phase_script" ]]; then
        log_warning "Phase script not executable, attempting to fix: $phase_script"
        if chmod +x "$phase_script" 2>/dev/null; then
            log_success "Fixed permissions for: $phase_script"
        else
            log_error "Cannot make phase script executable: $phase_script"
            return 1
        fi
    fi
    
    log_info "🚀 Starting $phase_name phase..."
    if [[ ${#phase_args[@]} -gt 0 ]]; then
        log_info "📋 Phase arguments: ${phase_args[*]}"
    fi
    
    local start_time
    start_time=$(date +%s)
    
    # Build command with only non-empty arguments
    local cmd=("$phase_script")
    [[ -n "$VERBOSE" ]] && cmd+=("$VERBOSE")
    [[ -n "$PARALLEL" ]] && cmd+=("$PARALLEL")
    [[ -n "$NO_CACHE" ]] && cmd+=("$NO_CACHE")
    [[ -n "$DRY_RUN" ]] && cmd+=("$DRY_RUN")
    [[ -n "$CLEAR_CACHE" ]] && cmd+=("$CLEAR_CACHE")
    
    # Add phase-specific arguments
    if [[ ${#phase_args[@]} -gt 0 ]]; then
        cmd+=("${phase_args[@]}")
    fi
    
    # Execute with explicit environment variable propagation
    # This ensures all variables are available to the child process
    if env \
        PROJECT_ROOT="$PROJECT_ROOT" \
        SCRIPT_DIR="$SCRIPT_DIR" \
        VERBOSE="$VERBOSE" \
        PARALLEL="$PARALLEL" \
        NO_CACHE="$NO_CACHE" \
        DRY_RUN="$DRY_RUN" \
        CLEAR_CACHE="$CLEAR_CACHE" \
        LOG_LEVEL="$LOG_LEVEL" \
        TEST_CACHE_TTL="$TEST_CACHE_TTL" \
        TEST_CACHE_MAX_AGE="$TEST_CACHE_MAX_AGE" \
        "${cmd[@]}"; then
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
    
    # Check and fix script permissions before proceeding
    if ! ensure_script_permissions; then
        log_error "Script permission validation failed. Cannot continue."
        exit 1
    fi
    
    local overall_success=true
    
    case $PHASE in
        static)
            run_phase "static" "${PHASE_ARGS[@]}" || overall_success=false
            ;;
        structure)
            run_phase "structure" "${PHASE_ARGS[@]}" || overall_success=false
            ;;
        integration)
            run_phase "integration" "${PHASE_ARGS[@]}" || overall_success=false
            ;;
        unit)
            run_phase "unit" "${PHASE_ARGS[@]}" || overall_success=false
            ;;
        docs)
            run_phase "docs" "${PHASE_ARGS[@]}" || overall_success=false
            ;;
        all)
            # Run all phases in sequence (no phase-specific args for all)
            local phases=("static" "structure" "integration" "unit" "docs")
            
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