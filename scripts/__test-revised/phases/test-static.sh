#!/usr/bin/env bash
# Static Analysis Phase - Vrooli Testing Infrastructure
# 
# Performs static analysis on all shell scripts in scripts/ directory:
# - Discovers all shell scripts
# - Runs shellcheck static analysis
# - Validates bash syntax with 'bash -n'
# - Uses caching to avoid unnecessary reruns
# - Provides detailed reporting

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="${PROJECT_ROOT:-$(cd "$SCRIPT_DIR/../.." && pwd)}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

show_usage() {
    cat << 'EOF'
Static Analysis Phase - Test all shell scripts with shellcheck and bash syntax validation

Usage: ./test-static.sh [OPTIONS]

OPTIONS:
  --verbose      Show detailed output for each file tested
  --parallel     Run tests in parallel (when possible)  
  --no-cache     Skip caching optimizations
  --dry-run      Show what would be tested without running
  --clear-cache  Clear static analysis cache before running
  --help         Show this help

WHAT THIS PHASE TESTS:
  1. Discovers all shell scripts in scripts/ directory
  2. Validates bash syntax with 'bash -n'
  3. Runs shellcheck static analysis (if available)
  4. Uses intelligent caching based on file modification time
  5. Reports summary of all findings

EXAMPLES:
  ./test-static.sh                    # Run all static analysis
  ./test-static.sh --verbose          # Show detailed output
  ./test-static.sh --no-cache         # Force re-analysis of all files
  ./test-static.sh --clear-cache      # Clear cache and re-run
EOF
}

# Parse command line arguments
CLEAR_CACHE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clear-cache)
            CLEAR_CACHE="true"
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        --verbose|--parallel|--no-cache|--dry-run)
            # These are handled by the main orchestrator
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Clear cache if requested
if [[ -n "$CLEAR_CACHE" ]]; then
    clear_cache "static"
fi

# Main static analysis function
run_static_analysis() {
    log_section "Static Analysis Phase"
    
    reset_test_counters
    reset_cache_stats
    
    # Load cache for this phase
    load_cache "static"
    
    # Find all shell scripts in scripts/ directory
    log_info "🔍 Discovering shell scripts in scripts/ directory..."
    
    local scripts_dir="$PROJECT_ROOT/scripts"
    if [[ ! -d "$scripts_dir" ]]; then
        log_error "Scripts directory not found: $scripts_dir"
        return 1
    fi
    
    # Define exclusion patterns for directories we should skip
    local exclude_dirs="__test __test-revised .git node_modules vendor dist build cache tmp temp"
    local find_excludes=""
    for dir in $exclude_dirs; do
        find_excludes="$find_excludes -path '*/$dir' -prune -o"
    done
    
    # Find shell scripts with various patterns
    local shell_scripts=()
    
    # Find .sh files (excluding test directories)
    while IFS= read -r -d '' script; do
        shell_scripts+=("$script")
    done < <(eval "find '$scripts_dir' $find_excludes -type f -name '*.sh' -print0 2>/dev/null")
    
    # Find files with shell shebangs (but no .sh extension, excluding test directories)
    while IFS= read -r -d '' script; do
        if [[ ! "$script" =~ \.sh$ ]] && head -n1 "$script" 2>/dev/null | grep -q '^#!/.*sh'; then
            shell_scripts+=("$script")
        fi
    done < <(eval "find '$scripts_dir' $find_excludes -type f -executable -print0 2>/dev/null")
    
    # Remove duplicates and sort
    local unique_scripts
    mapfile -t unique_scripts < <(printf '%s\n' "${shell_scripts[@]}" | sort -u)
    
    local total_scripts=${#unique_scripts[@]}
    
    if [[ $total_scripts -eq 0 ]]; then
        log_warning "No shell scripts found in $scripts_dir"
        print_test_summary
        return 0
    fi
    
    log_info "📋 Found $total_scripts shell scripts to analyze"
    
    # Process each script
    local current=0
    for script in "${unique_scripts[@]}"; do
        ((current++))
        local relative_script
        relative_script=$(relative_path "$script")
        
        log_progress "$current" "$total_scripts" "Analyzing scripts"
        
        # Test 1: Bash syntax validation
        local syntax_passed=false
        if run_cached_test "$script" "bash-syntax" "validate_shell_syntax '$script' 'shell script'" "bash-syntax: $relative_script"; then
            syntax_passed=true
            increment_test_counter "passed"
        else
            increment_test_counter "failed"
        fi
        
        # Test 2: Shellcheck static analysis (only if syntax is valid)
        if [[ "$syntax_passed" == "true" ]]; then
            if run_cached_test "$script" "shellcheck" "run_shellcheck '$script' 'shell script'" "shellcheck: $relative_script"; then
                increment_test_counter "passed"
            else
                increment_test_counter "failed"
            fi
        else
            log_test_skip "shellcheck: $relative_script" "Skipped due to syntax errors"
            cache_test_result "$script" "shellcheck" "skipped" "syntax errors"
            increment_test_counter "skipped"
        fi
    done
    
    # Save cache before printing results
    save_cache
    
    # Print final results
    print_test_summary
    
    # Show cache statistics if verbose
    if is_verbose; then
        log_info ""
        show_cache_stats
    fi
    
    # Return success if no failures
    local counters
    read -r total passed failed skipped <<< "$(get_test_counters)"
    
    if [[ $failed -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Additional helper functions for static analysis

check_static_analysis_tools() {
    log_info "🔧 Checking available static analysis tools..."
    
    local tools_available=true
    
    if ! command -v bash >/dev/null 2>&1; then
        log_error "bash not found - cannot perform syntax validation"
        tools_available=false
    else
        is_verbose && log_success "bash available for syntax validation"
    fi
    
    if ! command -v shellcheck >/dev/null 2>&1; then
        log_warning "shellcheck not found - static analysis will be limited"
        log_info "Install shellcheck for comprehensive static analysis:"
        log_info "  apt-get install shellcheck    # Ubuntu/Debian"
        log_info "  brew install shellcheck       # macOS"
        log_info "  dnf install ShellCheck        # Fedora"
    else
        local shellcheck_version
        shellcheck_version=$(shellcheck --version | grep '^version:' | cut -d' ' -f2 2>/dev/null || echo "unknown")
        is_verbose && log_success "shellcheck available (version: $shellcheck_version)"
    fi
    
    if ! command -v find >/dev/null 2>&1; then
        log_error "find not found - cannot discover shell scripts"
        tools_available=false
    else
        is_verbose && log_success "find available for script discovery"
    fi
    
    # Return based on boolean value
    if [[ "$tools_available" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# Show discovered scripts summary
show_discovery_summary() {
    local scripts_dir="$PROJECT_ROOT/scripts"
    
    log_info ""
    log_info "🔍 Script Discovery Summary:"
    log_info "================================"
    
    # Define exclusion patterns (same as in run_static_analysis)
    local exclude_dirs="__test __test-revised .git node_modules vendor dist build cache tmp temp"
    local find_excludes=""
    for dir in $exclude_dirs; do
        find_excludes="$find_excludes -path '*/$dir' -prune -o"
    done
    
    # Count by file extension (with exclusions)
    local sh_count=0
    local executable_count=0
    
    # Count .sh files with exclusions
    if command -v find >/dev/null 2>&1; then
        sh_count=$(eval "find '$scripts_dir' $find_excludes -type f -name '*.sh' -print 2>/dev/null" | wc -l || echo "0")
    fi
    
    # Count executable files with shell shebangs (with exclusions)
    if command -v find >/dev/null 2>&1 && command -v grep >/dev/null 2>&1; then
        local exec_files
        exec_files=$(eval "find '$scripts_dir' $find_excludes -type f -executable ! -name '*.sh' -print 2>/dev/null")
        if [[ -n "$exec_files" ]]; then
            # Count shell scripts among executable files
            executable_count=$(echo "$exec_files" | xargs -I{} head -n1 {} 2>/dev/null | grep -c '^#!/.*sh' 2>/dev/null || echo "0")
        else
            executable_count=0
        fi
    fi
    
    # Ensure counts are clean integers (avoid any formatting issues)
    sh_count="${sh_count:-0}"
    executable_count="${executable_count:-0}"
    
    # Clean up any whitespace or formatting issues - but avoid string manipulation that could cause duplication
    if [[ "$sh_count" =~ ^[0-9]+$ ]]; then
        sh_count="$sh_count"
    else
        sh_count="0"
    fi
    
    if [[ "$executable_count" =~ ^[0-9]+$ ]]; then
        executable_count="$executable_count"
    else
        executable_count="0"
    fi
    
    log_info "  .sh files: $sh_count"
    log_info "  Executable shell scripts: ${executable_count}"
    log_info "  Total shell scripts: $((sh_count + executable_count))"
    
    # Show directory breakdown if verbose (simplified)
    if is_verbose; then
        log_info ""
        log_info "📁 Top-level directories containing scripts:"
        find "$scripts_dir" -maxdepth 2 -name "*.sh" 2>/dev/null | \
            sed "s|$scripts_dir/||" | \
            cut -d'/' -f1 | \
            sort | uniq -c | \
            head -10 | \
            while read -r count dir; do
                log_info "  $dir/: $count .sh files"
            done
    fi
}

# Main execution
main() {
    if is_dry_run; then
        log_info "🔍 [DRY RUN] Static Analysis Phase"
        log_info "Would analyze all shell scripts in: $PROJECT_ROOT/scripts"
        show_discovery_summary
        return 0
    fi
    
    # Check that required tools are available
    if ! check_static_analysis_tools; then
        log_error "Required tools are missing for static analysis"
        return 1
    fi
    
    # Show discovery summary if verbose
    if is_verbose; then
        show_discovery_summary
    fi
    
    # Run the static analysis
    if run_static_analysis; then
        log_success "✅ Static analysis phase completed successfully"
        return 0
    else
        log_error "❌ Static analysis phase completed with failures"
        return 1
    fi
}

# Export functions for testing
export -f run_static_analysis check_static_analysis_tools show_discovery_summary

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi