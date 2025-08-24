#!/usr/bin/env bash
# Static Analysis Phase - Orchestrator for all language-specific static analyzers
# 
# Orchestrates static analysis across multiple languages:
# - Bash/Shell scripts (shellcheck, syntax validation)
# - TypeScript (tsc, ESLint)
# - Python (mypy, ruff/flake8)
# - Go (go vet, gofmt, golangci-lint)
#
# Each language has its own analyzer in static/ subdirectory

set -euo pipefail

# Get APP_ROOT using cached value or compute once (2 levels up: __test/phases/test-static.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test"
PROJECT_ROOT="${PROJECT_ROOT:-$APP_ROOT}"
STATIC_DIR="$SCRIPT_DIR/phases/static"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"

show_usage() {
    cat << 'EOF'
Static Analysis Phase - Multi-language static analysis orchestrator

Usage: ./test-static.sh [OPTIONS] [LANGUAGES]

LANGUAGES:
  all         Run all language analyzers (default)
  bash        Run Bash/Shell script analysis only
  typescript  Run TypeScript analysis only
  python      Run Python analysis only
  go          Run Go analysis only

OPTIONS:
  --verbose         Show detailed output for each file tested
  --parallel        Run language analyzers in parallel
  --no-cache        Skip caching optimizations
  --dry-run         Show what would be tested without running
  --clear-cache     Clear static analysis cache before running
  --resource=NAME   Analyze only specific resource (e.g., --resource=ollama)
  --scenario=NAME   Analyze only specific scenario (e.g., --scenario=app-generator)
  --path=PATH       Analyze only specific directory/file path
  --help            Show this help

WHAT THIS PHASE TESTS:
  Bash:       shellcheck, syntax validation (bash -n)
  TypeScript: tsc --noEmit, ESLint (if configured)
  Python:     syntax check, mypy/pyright, ruff/flake8
  Go:         go build, go vet, gofmt, golangci-lint

SCOPING EXAMPLES:
  ./test-static.sh --resource=ollama           # Analyze only ollama resource files
  ./test-static.sh --scenario=app-generator    # Analyze only app-generator scenario
  ./test-static.sh --path=scenarios/core       # Analyze only specific directory
  ./test-static.sh typescript --resource=n8n   # TypeScript analysis for n8n only

COMBINED EXAMPLES:
  ./test-static.sh                             # Run all static analysis
  ./test-static.sh bash python                 # Run only bash and python
  ./test-static.sh --parallel                  # Run all analyzers in parallel
  ./test-static.sh --clear-cache all           # Clear cache and run all
EOF
}

# Parse command line arguments
CLEAR_CACHE=""
LANGUAGES=()
RUN_PARALLEL=false
SCOPE_RESOURCE=""
SCOPE_SCENARIO=""
SCOPE_PATH=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clear-cache)
            CLEAR_CACHE="true"
            shift
            ;;
        --parallel)
            RUN_PARALLEL=true
            export PARALLEL=1
            shift
            ;;
        --resource=*)
            SCOPE_RESOURCE="${1#*=}"
            shift
            ;;
        --scenario=*)
            SCOPE_SCENARIO="${1#*=}"
            shift
            ;;
        --path=*)
            SCOPE_PATH="${1#*=}"
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        --verbose|--no-cache|--dry-run)
            # These are handled by the environment
            shift
            ;;
        all|bash|typescript|python|go)
            LANGUAGES+=("$1")
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Default to all languages if none specified
if [[ ${#LANGUAGES[@]} -eq 0 ]]; then
    LANGUAGES=("all")
fi

# Expand "all" to individual languages
if [[ " ${LANGUAGES[*]} " =~ " all " ]]; then
    LANGUAGES=("bash" "typescript" "python" "go")
fi

# Clear cache if requested
if [[ -n "$CLEAR_CACHE" ]]; then
    log_info "🗑️  Clearing static analysis cache..."
    for lang in "${LANGUAGES[@]}"; do
        case "$lang" in
            bash)       rm -f "$SCRIPT_DIR/cache/static-bash.json" ;;
            typescript) rm -f "$SCRIPT_DIR/cache/static-typescript.json" ;;
            python)     rm -f "$SCRIPT_DIR/cache/static-python.json" ;;
            go)         rm -f "$SCRIPT_DIR/cache/static-go.json" ;;
        esac
    done
fi

# Check which analyzers are available
check_available_analyzers() {
    local available=()
    
    for lang in "${LANGUAGES[@]}"; do
        local analyzer="$STATIC_DIR/${lang}.sh"
        if [[ -f "$analyzer" && -x "$analyzer" ]]; then
            available+=("$lang")
        elif [[ -f "$analyzer" ]]; then
            # Make it executable if it exists but isn't executable
            chmod +x "$analyzer"
            available+=("$lang")
        else
            log_warning "Analyzer not found: $analyzer"
        fi
    done
    
    printf '%s\n' "${available[@]}"
}

# Run a single language analyzer
run_language_analyzer() {
    local language="$1"
    local analyzer="$STATIC_DIR/${language}.sh"
    
    if [[ ! -f "$analyzer" ]]; then
        log_error "Analyzer not found: $analyzer"
        return 1
    fi
    
    log_info "🔍 Running $language static analysis..."
    
    # Build command with scoping arguments
    local cmd=("$analyzer")
    [[ -n "$SCOPE_RESOURCE" ]] && cmd+=("--resource=$SCOPE_RESOURCE")
    [[ -n "$SCOPE_SCENARIO" ]] && cmd+=("--scenario=$SCOPE_SCENARIO")
    [[ -n "$SCOPE_PATH" ]] && cmd+=("--path=$SCOPE_PATH")
    
    if "${cmd[@]}"; then
        log_success "✅ $language analysis completed successfully"
        return 0
    else
        log_error "❌ $language analysis failed"
        return 1
    fi
}

# Run analyzers in parallel
run_parallel_analysis() {
    local languages=("$@")
    local pids=()
    local results_dir="/tmp/static_results_$$"
    mkdir -p "$results_dir"
    
    log_info "🚀 Running ${#languages[@]} analyzers in parallel..."
    
    # Start all analyzers in background
    for lang in "${languages[@]}"; do
        (
            if run_language_analyzer "$lang"; then
                echo "PASS" > "$results_dir/$lang"
            else
                echo "FAIL" > "$results_dir/$lang"
            fi
        ) &
        pids+=($!)
        log_info "  Started $lang analyzer (PID: ${pids[-1]})"
    done
    
    # Wait for all to complete
    log_info "⏳ Waiting for analyzers to complete..."
    for pid in "${pids[@]}"; do
        wait "$pid" 2>/dev/null || true
    done
    
    # Collect results
    local all_passed=true
    local passed_count=0
    local failed_count=0
    
    for lang in "${languages[@]}"; do
        if [[ -f "$results_dir/$lang" ]]; then
            local result
            result=$(cat "$results_dir/$lang")
            if [[ "$result" == "PASS" ]]; then
                ((passed_count++))
                log_success "  ✅ $lang: PASSED"
            else
                ((failed_count++))
                log_error "  ❌ $lang: FAILED"
                all_passed=false
            fi
        else
            log_error "  ⚠️  $lang: NO RESULT"
            all_passed=false
            ((failed_count++))
        fi
    done
    
    # Clean up
    rm -rf "$results_dir"
    
    log_info ""
    log_info "📊 Parallel Analysis Summary:"
    log_info "  Total: ${#languages[@]} languages"
    log_info "  Passed: $passed_count"
    log_info "  Failed: $failed_count"
    
    if [[ "$all_passed" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# Run analyzers sequentially
run_sequential_analysis() {
    local languages=("$@")
    local all_passed=true
    local passed_count=0
    local failed_count=0
    
    for lang in "${languages[@]}"; do
        if run_language_analyzer "$lang"; then
            ((passed_count++))
        else
            ((failed_count++))
            all_passed=false
        fi
    done
    
    log_info ""
    log_info "📊 Sequential Analysis Summary:"
    log_info "  Total: ${#languages[@]} languages"
    log_info "  Passed: $passed_count"
    log_info "  Failed: $failed_count"
    
    if [[ "$all_passed" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    local start_time
    start_time=$(date +%s)
    
    log_section "Static Analysis Phase (Multi-Language)"
    
    # Show scope information
    if [[ -n "$SCOPE_RESOURCE" || -n "$SCOPE_SCENARIO" || -n "$SCOPE_PATH" ]]; then
        log_info "🎯 Scoped analysis enabled:"
        [[ -n "$SCOPE_RESOURCE" ]] && log_info "  📦 Resource: $SCOPE_RESOURCE"
        [[ -n "$SCOPE_SCENARIO" ]] && log_info "  🎬 Scenario: $SCOPE_SCENARIO"
        [[ -n "$SCOPE_PATH" ]] && log_info "  📁 Path: $SCOPE_PATH"
    fi
    
    if is_dry_run; then
        log_info "🔍 [DRY RUN] Would run static analysis for: ${LANGUAGES[*]}"
        for lang in "${LANGUAGES[@]}"; do
            log_info "  - $lang: $STATIC_DIR/${lang}.sh"
        done
        
        if [[ -n "$SCOPE_RESOURCE" || -n "$SCOPE_SCENARIO" || -n "$SCOPE_PATH" ]]; then
            log_info "🎯 Scope filters would be applied:"
            [[ -n "$SCOPE_RESOURCE" ]] && log_info "  📦 Resource: $SCOPE_RESOURCE"
            [[ -n "$SCOPE_SCENARIO" ]] && log_info "  🎬 Scenario: $SCOPE_SCENARIO"
            [[ -n "$SCOPE_PATH" ]] && log_info "  📁 Path: $SCOPE_PATH"
        fi
        return 0
    fi
    
    # Check which analyzers are actually available
    local available_analyzers=()
    while IFS= read -r analyzer; do
        [[ -n "$analyzer" ]] && available_analyzers+=("$analyzer")
    done < <(check_available_analyzers)
    
    if [[ ${#available_analyzers[@]} -eq 0 ]]; then
        log_error "No analyzers available for requested languages: ${LANGUAGES[*]}"
        return 1
    fi
    
    log_info "📋 Running analyzers for: ${available_analyzers[*]}"
    
    # Run the analyzers
    local analysis_result
    if [[ "$RUN_PARALLEL" == "true" ]] && [[ ${#available_analyzers[@]} -gt 1 ]]; then
        if run_parallel_analysis "${available_analyzers[@]}"; then
            analysis_result=0
        else
            analysis_result=1
        fi
    else
        if run_sequential_analysis "${available_analyzers[@]}"; then
            analysis_result=0
        else
            analysis_result=1
        fi
    fi
    
    # Calculate total duration
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_info ""
    log_section "Static Analysis Complete"
    log_info "⏱️  Total duration: ${duration}s"
    
    if [[ $analysis_result -eq 0 ]]; then
        log_success "🎉 All static analysis passed!"
        return 0
    else
        log_error "💥 Some static analysis checks failed"
        return 1
    fi
}

# Export functions for testing
export -f check_available_analyzers run_language_analyzer
export -f run_parallel_analysis run_sequential_analysis

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi