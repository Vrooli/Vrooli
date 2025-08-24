#!/usr/bin/env bash
# Bash Static Analysis - Language-specific checker for shell scripts
#
# Performs:
# - Shellcheck static analysis
# - Bash syntax validation with 'bash -n'
# - Uses caching to avoid unnecessary reruns
# - Supports scoping to specific resources, scenarios, or paths

set -euo pipefail

# Get APP_ROOT using cached value or compute once (3 levels up: __test/phases/static/bash.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test"
PROJECT_ROOT="${PROJECT_ROOT:-$APP_ROOT}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

# Parse command line arguments for scoping
SCOPE_RESOURCE=""
SCOPE_SCENARIO=""
SCOPE_PATH=""

while [[ $# -gt 0 ]]; do
    case $1 in
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
        *)
            shift  # Ignore other arguments
            ;;
    esac
done

# Check if script should be tested based on scoping
should_test_script() {
    local script_path="$1"
    
    # If no scoping, test everything
    if [[ -z "$SCOPE_RESOURCE" && -z "$SCOPE_SCENARIO" && -z "$SCOPE_PATH" ]]; then
        return 0
    fi
    
    # Check path scope
    if [[ -n "$SCOPE_PATH" ]]; then
        if [[ "$script_path" == "$SCOPE_PATH"* ]]; then
            return 0
        fi
    fi
    
    # Check resource scope
    if [[ -n "$SCOPE_RESOURCE" ]]; then
        if [[ "$script_path" == *"/resources/"* ]] && [[ "$script_path" == *"$SCOPE_RESOURCE"* ]]; then
            return 0
        fi
    fi
    
    # Check scenario scope
    if [[ -n "$SCOPE_SCENARIO" ]]; then
        if [[ "$script_path" == *"/scenarios/"* ]] && [[ "$script_path" == *"$SCOPE_SCENARIO"* ]]; then
            return 0
        fi
    fi
    
    return 1
}

# Find all shell scripts in project
find_shell_scripts() {
    local scripts_dir="$PROJECT_ROOT"
    
    # If we have a specific path scope, start from there
    if [[ -n "$SCOPE_PATH" ]]; then
        if [[ "$SCOPE_PATH" = /* ]]; then
            scripts_dir="$SCOPE_PATH"  # Absolute path
        else
            scripts_dir="$PROJECT_ROOT/$SCOPE_PATH"  # Relative path
        fi
        
        if [[ ! -d "$scripts_dir" ]]; then
            log_warning "Scoped path not found: $scripts_dir"
            return 0
        fi
    fi
    
    # Define exclusion patterns for directories we should skip
    local exclude_dirs="__test __test-revised .git node_modules vendor dist build cache tmp temp .next .cache .turbo coverage out target debug release .vscode .idea __pycache__ .pytest_cache .tox venv env .env .venv"
    local find_excludes=""
    for dir in $exclude_dirs; do
        find_excludes="$find_excludes -path '*/$dir' -prune -o"
    done
    
    local shell_scripts=()
    
    # Find .sh files
    while IFS= read -r -d '' script; do
        if should_test_script "$script"; then
            shell_scripts+=("$script")
        fi
    done < <(eval "find '$scripts_dir' $find_excludes -type f -name '*.sh' -print0 2>/dev/null")
    
    # Find files with shell shebangs (but no .sh extension)
    while IFS= read -r -d '' script; do
        if [[ -f "$script" && -s "$script" && ! "$script" =~ \.sh$ ]]; then
            if head -n1 "$script" 2>/dev/null | grep -q '^#!/.*\(bash\|sh\|zsh\|ksh\)'; then
                if should_test_script "$script"; then
                    shell_scripts+=("$script")
                fi
            fi
        fi
    done < <(eval "find '$scripts_dir' $find_excludes -type f -executable -print0 2>/dev/null")
    
    # Remove duplicates and validate
    local unique_scripts
    local validated_scripts=()
    mapfile -t unique_scripts < <(printf '%s\n' "${shell_scripts[@]}" | sort -u)
    
    for script in "${unique_scripts[@]}"; do
        if [[ -f "$script" && -r "$script" ]]; then
            validated_scripts+=("$script")
        fi
    done
    
    printf '%s\n' "${validated_scripts[@]}"
}

# Check if tools are available
check_bash_tools() {
    local has_bash=true
    local has_shellcheck=false
    
    if ! command -v bash >/dev/null 2>&1; then
        log_error "bash not found - cannot perform syntax validation"
        has_bash=false
    fi
    
    if command -v shellcheck >/dev/null 2>&1; then
        has_shellcheck=true
        local shellcheck_version
        shellcheck_version=$(shellcheck --version | grep '^version:' | cut -d' ' -f2 2>/dev/null || echo "unknown")
        log_info "shellcheck available (version: $shellcheck_version)"
    else
        log_warning "shellcheck not found - static analysis will be limited"
        log_info "Install: apt-get install shellcheck (Ubuntu) or brew install shellcheck (macOS)"
    fi
    
    echo "$has_bash:$has_shellcheck"
}

# Run bash static analysis
run_bash_static() {
    local start_time
    start_time=$(date +%s)
    
    log_section "Bash Static Analysis"
    
    # Show scope information
    if [[ -n "$SCOPE_RESOURCE" || -n "$SCOPE_SCENARIO" || -n "$SCOPE_PATH" ]]; then
        log_info "🎯 Scoped analysis enabled:"
        [[ -n "$SCOPE_RESOURCE" ]] && log_info "  📦 Resource: $SCOPE_RESOURCE"
        [[ -n "$SCOPE_SCENARIO" ]] && log_info "  🎬 Scenario: $SCOPE_SCENARIO"
        [[ -n "$SCOPE_PATH" ]] && log_info "  📁 Path: $SCOPE_PATH"
    fi
    
    # Load cache
    load_cache "static-bash"
    
    # Check tools
    local tool_status
    tool_status=$(check_bash_tools)
    IFS=':' read -r has_bash has_shellcheck <<< "$tool_status"
    
    if [[ "$has_bash" != "true" ]]; then
        log_error "Required tool 'bash' is not available"
        return 1
    fi
    
    # Find scripts
    log_info "🔍 Discovering shell scripts..."
    local scripts=()
    while IFS= read -r script; do
        scripts+=("$script")
    done < <(find_shell_scripts)
    
    local total=${#scripts[@]}
    if [[ $total -eq 0 ]]; then
        log_warning "No shell scripts found"
        return 0
    fi
    
    log_info "📋 Found $total shell scripts to analyze"
    
    # Filter out cached results
    local files_to_test=()
    local syntax_cached=0
    local shellcheck_cached=0
    
    for script in "${scripts[@]}"; do
        local needs_test=false
        
        # Check syntax cache
        if ! check_cache "$script" "bash-syntax"; then
            needs_test=true
        else
            ((syntax_cached++)) || true
        fi
        
        # Check shellcheck cache if available
        if [[ "$has_shellcheck" == "true" ]]; then
            if ! check_cache "$script" "shellcheck"; then
                needs_test=true
            else
                ((shellcheck_cached++)) || true
            fi
        fi
        
        if [[ "$needs_test" == "true" ]]; then
            files_to_test+=("$script")
        fi
    done
    
    local num_to_test=${#files_to_test[@]}
    
    if [[ $num_to_test -eq 0 ]]; then
        log_success "All $total files validated from cache!"
        save_cache
        return 0
    fi
    
    log_info "📊 Testing $num_to_test files (cached: syntax=$syntax_cached, shellcheck=$shellcheck_cached)"
    
    # Test counters
    local passed=0
    local failed=0
    
    # Process files
    for script in "${files_to_test[@]}"; do
        local relative_path
        relative_path=$(relative_path "$script")
        
        # Test 1: Bash syntax
        log_test_start "bash-syntax" "$relative_path"
        if validate_shell_syntax "$script" "shell script"; then
            log_test_pass "bash-syntax" "$relative_path"
            update_cache "$script" "bash-syntax" "passed" 0 "" 0
            ((passed++)) || true
            
            # Test 2: Shellcheck (only if syntax passes and shellcheck available)
            if [[ "$has_shellcheck" == "true" ]]; then
                log_test_start "shellcheck" "$relative_path"
                if run_shellcheck "$script" "shell script"; then
                    log_test_pass "shellcheck" "$relative_path"
                    update_cache "$script" "shellcheck" "passed" 0 "" 0
                    ((passed++)) || true
                else
                    log_test_fail "shellcheck" "$relative_path"
                    update_cache "$script" "shellcheck" "failed" 0 "shellcheck issues" 1
                    ((failed++)) || true
                fi
            fi
        else
            log_test_fail "bash-syntax" "$relative_path"
            update_cache "$script" "bash-syntax" "failed" 0 "syntax error" 1
            ((failed++)) || true
            
            # Skip shellcheck if syntax fails
            if [[ "$has_shellcheck" == "true" ]]; then
                log_test_skip "shellcheck" "$relative_path" "syntax error"
                update_cache "$script" "shellcheck" "skipped" 0 "syntax error" 0
            fi
        fi
    done
    
    # Add cached results to counters
    passed=$((passed + syntax_cached))
    if [[ "$has_shellcheck" == "true" ]]; then
        passed=$((passed + shellcheck_cached))
    fi
    
    # Save cache
    save_cache
    
    # Calculate duration
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Summary
    log_section "Bash Static Analysis Summary"
    log_info "📊 Results:"
    log_info "  Total Scripts: $total"
    log_info "  Tests Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log_error "  Tests Failed: $failed"
    fi
    log_info "  Cached Results: $((syntax_cached + shellcheck_cached))"
    log_info "  Duration: ${duration}s"
    
    if [[ $failed -eq 0 ]]; then
        log_success "✅ All bash static analysis passed!"
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    if is_dry_run; then
        log_info "[DRY RUN] Would run bash static analysis"
        return 0
    fi
    
    run_bash_static
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi