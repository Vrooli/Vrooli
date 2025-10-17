#!/usr/bin/env bash
# Python Static Analysis - Type checking and linting for Python files
#
# Performs:
# - Type checking with mypy or pyright
# - Linting with ruff, flake8, or pylint
# - Focuses on orchestrators and AI tools

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test"
PROJECT_ROOT="${PROJECT_ROOT:-$APP_ROOT}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

# Find Python files
find_python_files() {
    local scripts_dir="$PROJECT_ROOT"
    
    # Define exclusion patterns
    local exclude_dirs="__test __test-revised .git node_modules vendor dist build cache tmp temp .next .cache .turbo coverage out target debug release .vscode .idea __pycache__ .pytest_cache .tox venv env .env .venv .virtualenv virtualenv"
    local find_excludes=""
    for dir in $exclude_dirs; do
        find_excludes="$find_excludes -path '*/$dir' -prune -o"
    done
    
    # Find .py files
    eval "find '$scripts_dir' $find_excludes -type f -name '*.py' -print 2>/dev/null" | sort
}

# Check Python tools availability
check_python_tools() {
    local has_python=false
    local has_mypy=false
    local has_pyright=false
    local has_ruff=false
    local has_flake8=false
    local has_pylint=false
    
    if command -v python3 >/dev/null 2>&1; then
        has_python=true
        local python_version
        python_version=$(python3 --version 2>/dev/null | cut -d' ' -f2 || echo "unknown")
        log_info "Python available (version: $python_version)"
    elif command -v python >/dev/null 2>&1; then
        has_python=true
        local python_version
        python_version=$(python --version 2>/dev/null | cut -d' ' -f2 || echo "unknown")
        log_info "Python available (version: $python_version)"
    else
        log_error "Python not found - cannot run Python checks"
    fi
    
    # Check for type checkers
    if command -v mypy >/dev/null 2>&1; then
        has_mypy=true
        local mypy_version
        mypy_version=$(mypy --version 2>/dev/null | cut -d' ' -f2 || echo "unknown")
        log_info "mypy available (version: $mypy_version)"
    fi
    
    if command -v pyright >/dev/null 2>&1; then
        has_pyright=true
        log_info "pyright available"
    fi
    
    # Check for linters
    if command -v ruff >/dev/null 2>&1; then
        has_ruff=true
        local ruff_version
        ruff_version=$(ruff --version 2>/dev/null | cut -d' ' -f2 || echo "unknown")
        log_info "ruff available (version: $ruff_version)"
    fi
    
    if command -v flake8 >/dev/null 2>&1; then
        has_flake8=true
        local flake8_version
        flake8_version=$(flake8 --version 2>/dev/null | head -1 || echo "unknown")
        log_info "flake8 available: $flake8_version"
    fi
    
    if command -v pylint >/dev/null 2>&1; then
        has_pylint=true
        local pylint_version
        pylint_version=$(pylint --version 2>/dev/null | head -1 | cut -d' ' -f2 || echo "unknown")
        log_info "pylint available (version: $pylint_version)"
    fi
    
    # If no tools found, suggest installation
    if [[ "$has_mypy" != "true" ]] && [[ "$has_pyright" != "true" ]]; then
        log_warning "No Python type checker found. Install with:"
        log_info "  pip install mypy  # or"
        log_info "  npm install -g pyright"
        log_info ""
        log_info "Or install all Python dev tools via runtime script:"
        log_info "  PYTHON_INSTALL_DEV_TOOLS=true scripts/lib/runtimes/python.sh"
    fi
    
    if [[ "$has_ruff" != "true" ]] && [[ "$has_flake8" != "true" ]] && [[ "$has_pylint" != "true" ]]; then
        log_warning "No Python linter found. Install with:"
        log_info "  pip install ruff  # (recommended, fast)"
        log_info "  pip install flake8  # or"
        log_info "  pip install pylint"
        log_info ""
        log_info "Or install all Python dev tools via runtime script:"
        log_info "  PYTHON_INSTALL_DEV_TOOLS=true scripts/lib/runtimes/python.sh"
    fi
    
    echo "$has_python:$has_mypy:$has_pyright:$has_ruff:$has_flake8:$has_pylint"
}

# Run Python syntax check
run_python_syntax() {
    local file="$1"
    local relative_path
    relative_path=$(relative_path "$file")
    
    log_test_start "py-syntax" "$relative_path"
    
    local python_cmd="python3"
    if ! command -v python3 >/dev/null 2>&1; then
        python_cmd="python"
    fi
    
    # Compile the file to check syntax
    if $python_cmd -m py_compile "$file" 2>/dev/null; then
        log_test_pass "py-syntax" "$relative_path"
        # Clean up compiled file
        rm -f "${file}c" 2>/dev/null
        rm -rf "$(dirname "$file")/__pycache__" 2>/dev/null
        update_cache "$file" "py-syntax" "passed" 0 "" 0
        return 0
    else
        log_test_fail "py-syntax" "$relative_path"
        update_cache "$file" "py-syntax" "failed" 0 "syntax error" 1
        return 1
    fi
}

# Run mypy type check
run_mypy_check() {
    local file="$1"
    local relative_path
    relative_path=$(relative_path "$file")
    
    log_test_start "mypy" "$relative_path"
    
    local output
    if output=$(mypy "$file" --ignore-missing-imports --no-error-summary 2>&1); then
        log_test_pass "mypy" "$relative_path"
        update_cache "$file" "mypy" "passed" 0 "" 0
        return 0
    else
        log_test_fail "mypy" "$relative_path"
        
        # Show first 5 errors
        echo "$output" | head -5 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        local error_count
        error_count=$(echo "$output" | wc -l)
        if [[ "$error_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((error_count - 5)) more issues"
        fi
        
        update_cache "$file" "mypy" "failed" 0 "$output" 1
        return 1
    fi
}

# Run ruff linter
run_ruff_check() {
    local file="$1"
    local relative_path
    relative_path=$(relative_path "$file")
    
    log_test_start "ruff" "$relative_path"
    
    local output
    if output=$(ruff check "$file" 2>&1); then
        log_test_pass "ruff" "$relative_path"
        update_cache "$file" "ruff" "passed" 0 "" 0
        return 0
    else
        log_test_fail "ruff" "$relative_path"
        
        # Show first 5 issues
        echo "$output" | head -5 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        local error_count
        error_count=$(echo "$output" | wc -l)
        if [[ "$error_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((error_count - 5)) more issues"
        fi
        
        update_cache "$file" "ruff" "failed" 0 "$output" 1
        return 1
    fi
}

# Run flake8 linter
run_flake8_check() {
    local file="$1"
    local relative_path
    relative_path=$(relative_path "$file")
    
    log_test_start "flake8" "$relative_path"
    
    local output
    if output=$(flake8 "$file" 2>&1); then
        log_test_pass "flake8" "$relative_path"
        update_cache "$file" "flake8" "passed" 0 "" 0
        return 0
    else
        log_test_fail "flake8" "$relative_path"
        
        # Show first 5 issues
        echo "$output" | head -5 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        local error_count
        error_count=$(echo "$output" | wc -l)
        if [[ "$error_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((error_count - 5)) more issues"
        fi
        
        update_cache "$file" "flake8" "failed" 0 "$output" 1
        return 1
    fi
}

# Run Python static analysis
run_python_static() {
    local start_time
    start_time=$(date +%s)
    
    log_section "Python Static Analysis"
    
    # Load cache
    load_cache "static-python"
    
    # Check tools
    local tool_status
    tool_status=$(check_python_tools)
    IFS=':' read -r has_python has_mypy has_pyright has_ruff has_flake8 has_pylint <<< "$tool_status"
    
    if [[ "$has_python" != "true" ]]; then
        log_warning "Python not found - skipping Python checks"
        return 0
    fi
    
    # Find Python files
    log_info "🔍 Discovering Python files..."
    local files=()
    while IFS= read -r file; do
        [[ -n "$file" ]] && files+=("$file")
    done < <(find_python_files)
    
    local total=${#files[@]}
    if [[ $total -eq 0 ]]; then
        log_warning "No Python files found"
        return 0
    fi
    
    log_info "📋 Found $total Python files to analyze"
    
    # Determine which linter to use (prefer ruff for speed)
    local linter=""
    if [[ "$has_ruff" == "true" ]]; then
        linter="ruff"
    elif [[ "$has_flake8" == "true" ]]; then
        linter="flake8"
    fi
    
    # Filter out cached results
    local files_to_test=()
    local syntax_cached=0
    local type_cached=0
    local lint_cached=0
    
    for file in "${files[@]}"; do
        local needs_test=false
        
        # Check syntax cache
        if ! check_cache "$file" "py-syntax"; then
            needs_test=true
        else
            ((syntax_cached++)) || true
        fi
        
        # Check type checker cache
        if [[ "$has_mypy" == "true" ]]; then
            if ! check_cache "$file" "mypy"; then
                needs_test=true
            else
                ((type_cached++)) || true
            fi
        fi
        
        # Check linter cache
        if [[ -n "$linter" ]]; then
            if ! check_cache "$file" "$linter"; then
                needs_test=true
            else
                ((lint_cached++)) || true
            fi
        fi
        
        if [[ "$needs_test" == "true" ]]; then
            files_to_test+=("$file")
        fi
    done
    
    local num_to_test=${#files_to_test[@]}
    
    if [[ $num_to_test -eq 0 ]]; then
        log_success "All $total files validated from cache!"
        save_cache
        return 0
    fi
    
    log_info "📊 Testing $num_to_test files (cached: syntax=$syntax_cached, type=$type_cached, lint=$lint_cached)"
    
    # Test counters
    local passed=0
    local failed=0
    local skipped=0
    
    # Process files
    for file in "${files_to_test[@]}"; do
        # Test 1: Python syntax
        if run_python_syntax "$file"; then
            ((passed++)) || true
            
            # Test 2: Type checking (only if syntax passes)
            if [[ "$has_mypy" == "true" ]]; then
                if run_mypy_check "$file"; then
                    ((passed++)) || true
                else
                    ((failed++)) || true
                fi
            fi
            
            # Test 3: Linting (only if syntax passes)
            if [[ "$linter" == "ruff" ]]; then
                if run_ruff_check "$file"; then
                    ((passed++)) || true
                else
                    ((failed++)) || true
                fi
            elif [[ "$linter" == "flake8" ]]; then
                if run_flake8_check "$file"; then
                    ((passed++)) || true
                else
                    ((failed++)) || true
                fi
            fi
        else
            ((failed++)) || true
            
            # Skip other checks if syntax fails
            if [[ "$has_mypy" == "true" ]]; then
                local relative_path
                relative_path=$(relative_path "$file")
                log_test_skip "mypy" "$relative_path" "syntax error"
                update_cache "$file" "mypy" "skipped" 0 "syntax error" 0
                ((skipped++)) || true
            fi
            
            if [[ -n "$linter" ]]; then
                local relative_path
                relative_path=$(relative_path "$file")
                log_test_skip "$linter" "$relative_path" "syntax error"
                update_cache "$file" "$linter" "skipped" 0 "syntax error" 0
                ((skipped++)) || true
            fi
        fi
    done
    
    # Add cached results to counters
    passed=$((passed + syntax_cached + type_cached + lint_cached))
    
    # Save cache
    save_cache
    
    # Calculate duration
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Summary
    log_section "Python Static Analysis Summary"
    log_info "📊 Results:"
    log_info "  Total Files: $total"
    log_info "  Tests Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log_error "  Tests Failed: $failed"
    fi
    if [[ $skipped -gt 0 ]]; then
        log_warning "  Tests Skipped: $skipped"
    fi
    log_info "  Cached Results: $((syntax_cached + type_cached + lint_cached))"
    log_info "  Duration: ${duration}s"
    
    if [[ $failed -eq 0 ]]; then
        log_success "✅ All Python static analysis passed!"
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    if is_dry_run; then
        log_info "[DRY RUN] Would run Python static analysis"
        return 0
    fi
    
    run_python_static
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi