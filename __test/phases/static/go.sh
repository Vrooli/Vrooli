#!/usr/bin/env bash
# Go Static Analysis - Type checking and linting for Go files
#
# Performs:
# - Go compilation checks with go build
# - go vet for common mistakes
# - golangci-lint or staticcheck if available
# - gofmt formatting checks

set -euo pipefail

# Get APP_ROOT using cached value or compute once (3 levels up: __test/phases/static/go.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test"
PROJECT_ROOT="${PROJECT_ROOT:-$APP_ROOT}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"

# Find Go modules and files
find_go_modules() {
    # Find directories containing go.mod files
    find "$PROJECT_ROOT" -type f -name "go.mod" -not -path "*/vendor/*" -not -path "*/.git/*" 2>/dev/null | \
        xargs -I{} dirname {} | sort -u
}

find_go_files() {
    local module_dir="${1:-$PROJECT_ROOT}"
    
    # Define exclusion patterns
    local exclude_dirs="vendor .git node_modules dist build cache tmp temp"
    local find_excludes=""
    for dir in $exclude_dirs; do
        find_excludes="$find_excludes -path '*/$dir' -prune -o"
    done
    
    # Find .go files
    eval "find '$module_dir' $find_excludes -type f -name '*.go' -print 2>/dev/null" | sort
}

# Check Go tools availability
check_go_tools() {
    local has_go=false
    local has_gofmt=false
    local has_govet=false
    local has_golangci=false
    local has_staticcheck=false
    
    if command -v go >/dev/null 2>&1; then
        has_go=true
        has_gofmt=true  # gofmt comes with go
        has_govet=true  # go vet comes with go
        local go_version
        go_version=$(go version 2>/dev/null | cut -d' ' -f3 || echo "unknown")
        log_info "Go available (version: $go_version)"
    else
        log_error "Go not found - cannot run Go checks"
    fi
    
    # Check for additional linters
    if command -v golangci-lint >/dev/null 2>&1; then
        has_golangci=true
        local golangci_version
        golangci_version=$(golangci-lint --version 2>/dev/null | head -1 | cut -d' ' -f4 || echo "unknown")
        log_info "golangci-lint available (version: $golangci_version)"
    fi
    
    if command -v staticcheck >/dev/null 2>&1; then
        has_staticcheck=true
        local staticcheck_version
        staticcheck_version=$(staticcheck -version 2>/dev/null | head -1 || echo "unknown")
        log_info "staticcheck available: $staticcheck_version"
    fi
    
    # Suggest installation if advanced linters not found
    if [[ "$has_golangci" != "true" ]] && [[ "$has_staticcheck" != "true" ]]; then
        log_info "For comprehensive Go analysis, install:"
        log_info "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        log_info "  go install honnef.co/go/tools/cmd/staticcheck@latest"
        log_info ""
        log_info "Or install all Go dev tools via runtime script:"
        log_info "  GO_INSTALL_DEV_TOOLS=true scripts/lib/runtimes/go.sh"
    fi
    
    echo "$has_go:$has_gofmt:$has_govet:$has_golangci:$has_staticcheck"
}

# Run go build check
run_go_build() {
    local module_dir="$1"
    local module_name
    module_name=$(basename "$module_dir")
    
    log_test_start "go-build" "$module_name"
    
    # Try to build the module (without producing output)
    local output
    if output=$(cd "$module_dir" && go build -o /dev/null ./... 2>&1); then
        log_test_pass "go-build" "$module_name"
        update_cache "$module_dir" "go-build" "passed" 0 "" 0
        return 0
    else
        log_test_fail "go-build" "$module_name"
        
        # Show first 5 errors
        echo "$output" | head -5 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        local error_count
        error_count=$(echo "$output" | wc -l)
        if [[ "$error_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((error_count - 5)) more errors"
        fi
        
        update_cache "$module_dir" "go-build" "failed" 0 "$output" 1
        return 1
    fi
}

# Run go vet
run_go_vet() {
    local module_dir="$1"
    local module_name
    module_name=$(basename "$module_dir")
    
    log_test_start "go-vet" "$module_name"
    
    local output
    if output=$(cd "$module_dir" && go vet ./... 2>&1); then
        log_test_pass "go-vet" "$module_name"
        update_cache "$module_dir" "go-vet" "passed" 0 "" 0
        return 0
    else
        log_test_fail "go-vet" "$module_name"
        
        # Show first 5 issues
        echo "$output" | head -5 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        local error_count
        error_count=$(echo "$output" | wc -l)
        if [[ "$error_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((error_count - 5)) more issues"
        fi
        
        update_cache "$module_dir" "go-vet" "failed" 0 "$output" 1
        return 1
    fi
}

# Run gofmt check
run_gofmt_check() {
    local module_dir="$1"
    local module_name
    module_name=$(basename "$module_dir")
    
    log_test_start "gofmt" "$module_name"
    
    # Check if any files need formatting
    local unformatted
    unformatted=$(cd "$module_dir" && gofmt -l . 2>/dev/null | grep -v vendor/ || true)
    
    if [[ -z "$unformatted" ]]; then
        log_test_pass "gofmt" "$module_name"
        update_cache "$module_dir" "gofmt" "passed" 0 "" 0
        return 0
    else
        log_test_fail "gofmt" "$module_name"
        
        # Show unformatted files
        echo "$unformatted" | head -5 | while IFS= read -r file; do
            log_warning "    └─ Needs formatting: $file"
        done
        
        local file_count
        file_count=$(echo "$unformatted" | wc -l)
        if [[ "$file_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((file_count - 5)) more files"
        fi
        
        update_cache "$module_dir" "gofmt" "failed" 0 "$unformatted" 1
        return 1
    fi
}

# Run golangci-lint
run_golangci_lint() {
    local module_dir="$1"
    local module_name
    module_name=$(basename "$module_dir")
    
    log_test_start "golangci-lint" "$module_name"
    
    local output
    if output=$(cd "$module_dir" && golangci-lint run ./... 2>&1); then
        log_test_pass "golangci-lint" "$module_name"
        update_cache "$module_dir" "golangci-lint" "passed" 0 "" 0
        return 0
    else
        log_test_fail "golangci-lint" "$module_name"
        
        # Show first 5 issues
        echo "$output" | grep -E "^[^:]+\.go:[0-9]+" | head -5 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        update_cache "$module_dir" "golangci-lint" "failed" 0 "$output" 1
        return 1
    fi
}

# Run Go static analysis
run_go_static() {
    local start_time
    start_time=$(date +%s)
    
    log_section "Go Static Analysis"
    
    # Load cache
    load_cache "static-go"
    
    # Check tools
    local tool_status
    tool_status=$(check_go_tools)
    IFS=':' read -r has_go has_gofmt has_govet has_golangci has_staticcheck <<< "$tool_status"
    
    if [[ "$has_go" != "true" ]]; then
        log_warning "Go not found - skipping Go checks"
        return 0
    fi
    
    # Find Go modules
    log_info "🔍 Discovering Go modules..."
    local modules=()
    while IFS= read -r module; do
        [[ -n "$module" ]] && modules+=("$module")
    done < <(find_go_modules)
    
    # If no modules found, check for individual Go files
    if [[ ${#modules[@]} -eq 0 ]]; then
        log_info "No go.mod files found, checking for individual Go files..."
        local go_files=()
        while IFS= read -r file; do
            [[ -n "$file" ]] && go_files+=("$file")
        done < <(find_go_files)
        
        if [[ ${#go_files[@]} -gt 0 ]]; then
            log_info "📋 Found ${#go_files[@]} Go files (no module structure)"
            modules=("$PROJECT_ROOT")  # Treat root as module
        else
            log_warning "No Go files found"
            return 0
        fi
    else
        log_info "📋 Found ${#modules[@]} Go modules to analyze"
    fi
    
    # Test counters
    local passed=0
    local failed=0
    local skipped=0
    
    # Process each module
    for module in "${modules[@]}"; do
        local module_name
        module_name=$(basename "$module")
        
        # Check what needs testing
        local needs_build=false
        local needs_vet=false
        local needs_fmt=false
        local needs_lint=false
        
        if ! check_cache "$module" "go-build"; then
            needs_build=true
        else
            ((passed++)) || true
        fi
        
        if ! check_cache "$module" "go-vet"; then
            needs_vet=true
        else
            ((passed++)) || true
        fi
        
        if ! check_cache "$module" "gofmt"; then
            needs_fmt=true
        else
            ((passed++)) || true
        fi
        
        if [[ "$has_golangci" == "true" ]]; then
            if ! check_cache "$module" "golangci-lint"; then
                needs_lint=true
            else
                ((passed++)) || true
            fi
        fi
        
        # Run tests if needed
        local build_passed=true
        
        # Test 1: Go build
        if [[ "$needs_build" == "true" ]]; then
            if run_go_build "$module"; then
                ((passed++)) || true
            else
                ((failed++)) || true
                build_passed=false
            fi
        fi
        
        # Only run other tests if build passes
        if [[ "$build_passed" == "true" ]]; then
            # Test 2: Go vet
            if [[ "$needs_vet" == "true" ]]; then
                if run_go_vet "$module"; then
                    ((passed++)) || true
                else
                    ((failed++)) || true
                fi
            fi
            
            # Test 3: gofmt
            if [[ "$needs_fmt" == "true" ]]; then
                if run_gofmt_check "$module"; then
                    ((passed++)) || true
                else
                    ((failed++)) || true
                fi
            fi
            
            # Test 4: golangci-lint (if available)
            if [[ "$needs_lint" == "true" ]] && [[ "$has_golangci" == "true" ]]; then
                if run_golangci_lint "$module"; then
                    ((passed++)) || true
                else
                    ((failed++)) || true
                fi
            fi
        else
            # Skip other tests if build failed
            if [[ "$needs_vet" == "true" ]]; then
                log_test_skip "go-vet" "$module_name" "build failed"
                update_cache "$module" "go-vet" "skipped" 0 "build failed" 0
                ((skipped++)) || true
            fi
            
            if [[ "$needs_fmt" == "true" ]]; then
                log_test_skip "gofmt" "$module_name" "build failed"
                update_cache "$module" "gofmt" "skipped" 0 "build failed" 0
                ((skipped++)) || true
            fi
            
            if [[ "$needs_lint" == "true" ]] && [[ "$has_golangci" == "true" ]]; then
                log_test_skip "golangci-lint" "$module_name" "build failed"
                update_cache "$module" "golangci-lint" "skipped" 0 "build failed" 0
                ((skipped++)) || true
            fi
        fi
    done
    
    # Save cache
    save_cache
    
    # Calculate duration
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Summary
    log_section "Go Static Analysis Summary"
    log_info "📊 Results:"
    log_info "  Total Modules: ${#modules[@]}"
    log_info "  Tests Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log_error "  Tests Failed: $failed"
    fi
    if [[ $skipped -gt 0 ]]; then
        log_warning "  Tests Skipped: $skipped"
    fi
    log_info "  Duration: ${duration}s"
    
    if [[ $failed -eq 0 ]]; then
        log_success "✅ All Go static analysis passed!"
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    if is_dry_run; then
        log_info "[DRY RUN] Would run Go static analysis"
        return 0
    fi
    
    run_go_static
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi