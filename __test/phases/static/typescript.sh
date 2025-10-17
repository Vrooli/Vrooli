#!/usr/bin/env bash
# TypeScript Static Analysis - Type checking and linting for TypeScript files
#
# Performs:
# - TypeScript compilation checks with tsc --noEmit
# - ESLint checks (if available)
# - Checks packages/ui and packages/server

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

# Find TypeScript projects and directories with TypeScript files
find_typescript_projects() {
    local projects=()
    
    # Check scenarios directory (Vrooli's actual TypeScript code)
    if [[ -d "$PROJECT_ROOT/scenarios" ]]; then
        # Look for directories with TypeScript files or package.json, excluding node_modules
        while IFS= read -r -d '' dir; do
            # Skip node_modules and hidden directories
            if [[ "$dir" =~ node_modules ]] || [[ "$(basename "$dir")" =~ ^\. ]]; then
                continue
            fi
            
            local has_ts_files=false
            
            # Check if directory has .ts files (not in node_modules)
            if find "$dir" -maxdepth 1 -name "*.ts" -type f 2>/dev/null | grep -q .; then
                has_ts_files=true
            fi
            
            # Check if directory has package.json with TypeScript
            if [[ -f "$dir/package.json" ]]; then
                # Check if it mentions TypeScript or has .ts files in subdirs (but not node_modules)
                if grep -q "typescript" "$dir/package.json" 2>/dev/null || \
                   find "$dir" -path "*/node_modules" -prune -o -name "*.ts" -type f -print | head -1 | grep -q .; then
                    projects+=("$dir")
                    continue
                fi
            fi
            
            # Add directory if it has TypeScript files
            if [[ "$has_ts_files" == "true" ]]; then
                projects+=("$dir")
            fi
        done < <(find "$PROJECT_ROOT/scenarios" -path "*/node_modules" -prune -o -type d -print0 2>/dev/null)
    fi
    
    # Check for formal TypeScript projects with tsconfig.json
    while IFS= read -r -d '' tsconfig_file; do
        local project_dir
        project_dir=$(dirname "$tsconfig_file")
        # Skip if already added or if it's in node_modules
        if [[ ! "$tsconfig_file" =~ node_modules ]] && [[ ! " ${projects[*]} " =~ " $project_dir " ]]; then
            projects+=("$project_dir")
        fi
    done < <(find "$PROJECT_ROOT" -name "tsconfig.json" -not -path "*/node_modules/*" -print0 2>/dev/null)
    
    # Check for other common TypeScript project locations
    for common_dir in "packages" "apps" "src" "ui" "server" "shared" "client" "frontend" "backend"; do
        if [[ -d "$PROJECT_ROOT/$common_dir" ]]; then
            if [[ -f "$PROJECT_ROOT/$common_dir/tsconfig.json" ]] || \
               find "$PROJECT_ROOT/$common_dir" -maxdepth 2 -name "*.ts" -type f | head -1 | grep -q .; then
                projects+=("$PROJECT_ROOT/$common_dir")
            fi
        fi
    done
    
    # Check root if it has TypeScript files
    if find "$PROJECT_ROOT" -maxdepth 1 -name "*.ts" -type f | head -1 | grep -q . || \
       [[ -f "$PROJECT_ROOT/tsconfig.json" ]]; then
        projects+=("$PROJECT_ROOT")
    fi
    
    # Remove duplicates and sort
    printf '%s\n' "${projects[@]}" | sort -u
}

# Check if TypeScript tools are available
check_typescript_tools() {
    local projects=("$@")  # Accept projects as arguments
    local has_node=false
    local has_tsc=false
    local has_eslint=false
    
    if command -v node >/dev/null 2>&1; then
        has_node=true
        local node_version
        node_version=$(node --version 2>/dev/null || echo "unknown")
        log_info "Node.js available (version: $node_version)"
    else
        log_error "Node.js not found - cannot run TypeScript checks"
    fi
    
    if command -v npx >/dev/null 2>&1; then
        # Check for TypeScript globally first
        if command -v tsc >/dev/null 2>&1; then
            has_tsc=true
            local tsc_version
            tsc_version=$(tsc --version 2>/dev/null | cut -d' ' -f2 || echo "unknown")
            log_info "TypeScript compiler available globally (version: $tsc_version)"
        else
            # Check for TypeScript in project directories
            for project in "${projects[@]}"; do
                if [[ -d "$project" && -f "$project/package.json" ]]; then
                    # Check if this project has TypeScript in its dependencies
                    if grep -q "\"typescript\"" "$project/package.json" 2>/dev/null; then
                        if (cd "$project" && npx tsc --version >/dev/null 2>&1); then
                            has_tsc=true
                            local tsc_version
                            tsc_version=$(cd "$project" && npx tsc --version 2>/dev/null | cut -d' ' -f2 || echo "unknown")
                            log_info "TypeScript compiler available in $project (version: $tsc_version)"
                            break
                        fi
                    fi
                fi
            done
        fi
        
        if [[ "$has_tsc" != "true" ]]; then
            log_warning "TypeScript compiler not found. Install with:"
            log_info "  npm install -g typescript  # global installation"
            log_info "  cd __test && npm install    # or use local test infrastructure"
        fi
        
        # Check for ESLint globally or in projects
        if command -v eslint >/dev/null 2>&1; then
            has_eslint=true
            local eslint_version
            eslint_version=$(eslint --version 2>/dev/null || echo "unknown")
            log_info "ESLint available globally ($eslint_version)"
        else
            # Check in project directories
            for project in "${projects[@]}"; do
                if [[ -d "$project" && -f "$project/package.json" ]]; then
                    if grep -q "\"eslint\"" "$project/package.json" 2>/dev/null; then
                        if (cd "$project" && npx eslint --version >/dev/null 2>&1); then
                            has_eslint=true
                            log_info "ESLint available in $project"
                            break
                        fi
                    fi
                fi
            done
        fi
        
        if [[ "$has_eslint" != "true" ]]; then
            log_info "ESLint not found. Install with:"
            log_info "  npm install -g eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin"
            log_info "  cd __test && npm install    # or use local test infrastructure"
        fi
    fi
    
    echo "$has_node:$has_tsc:$has_eslint"
}

# Run TypeScript compilation check
run_tsc_check() {
    local project_dir="$1"
    local project_name
    project_name=$(basename "$project_dir")
    
    log_test_start "tsc" "$project_name"
    
    # Check if TypeScript is available globally or in project
    local tsc_cmd="tsc"
    if ! command -v tsc >/dev/null 2>&1; then
        # Try npx if global tsc not available
        if [[ -f "$project_dir/package.json" ]] && grep -q "\"typescript\"" "$project_dir/package.json" 2>/dev/null; then
            tsc_cmd="npx tsc"
        else
            log_test_skip "tsc" "$project_name" "TypeScript not available"
            return 0
        fi
    fi
    
    # Find TypeScript files in the directory
    local ts_files=()
    while IFS= read -r -d '' file; do
        ts_files+=("$file")
    done < <(find "$project_dir" -name "*.ts" -type f -print0 2>/dev/null)
    
    if [[ ${#ts_files[@]} -eq 0 ]]; then
        log_test_skip "tsc" "$project_name" "no TypeScript files"
        return 0
    fi
    
    # Create temp tsconfig if none exists
    local use_temp_config=false
    local temp_config=""
    if [[ ! -f "$project_dir/tsconfig.json" ]]; then
        use_temp_config=true
        temp_config="$project_dir/tsconfig.temp.json"
        cat > "$temp_config" << 'EOF'
{
  "compilerOptions": {
    "target": "es2020",
    "module": "esnext",
    "moduleResolution": "node",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "noEmit": true
  },
  "include": ["**/*.ts"],
  "exclude": ["node_modules"]
}
EOF
    fi
    
    # Run tsc with noEmit to check types without building
    local output
    local tsc_config_arg=""
    if [[ "$use_temp_config" == "true" ]]; then
        tsc_config_arg="--project $temp_config"
    fi
    
    if output=$(cd "$project_dir" && $tsc_cmd --noEmit $tsc_config_arg 2>&1); then
        log_test_pass "tsc" "$project_name"
        update_cache "$project_dir" "tsc" "passed" 0 "" 0
        
        # Clean up temp config
        [[ -f "$temp_config" ]] && rm -f "$temp_config"
        return 0
    else
        log_test_fail "tsc" "$project_name"
        
        # Show first 10 errors
        local error_count
        error_count=$(echo "$output" | grep -c "error TS" 2>/dev/null || echo "0")
        # Ensure error_count is a clean integer
        error_count=${error_count//[^0-9]/}
        error_count=${error_count:-0}
        
        echo "$output" | grep "error TS" | head -10 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        if [[ "$error_count" -gt 10 ]]; then
            log_warning "    └─ ... and $((error_count - 10)) more errors"
        fi
        
        update_cache "$project_dir" "tsc" "failed" 0 "$output" 1
        
        # Clean up temp config
        [[ -f "$temp_config" ]] && rm -f "$temp_config"
        return 1
    fi
}

# Run ESLint check
run_eslint_check() {
    local project_dir="$1"
    local project_name
    project_name=$(basename "$project_dir")
    
    # Check if .eslintrc exists
    local has_config=false
    for config in ".eslintrc" ".eslintrc.js" ".eslintrc.json" ".eslintrc.yml" ".eslintrc.yaml" "eslint.config.js" "eslint.config.mjs"; do
        if [[ -f "$project_dir/$config" ]]; then
            has_config=true
            break
        fi
    done
    
    if [[ "$has_config" != "true" ]]; then
        log_test_skip "eslint" "$project_name" "no config found"
        return 0
    fi
    
    log_test_start "eslint" "$project_name"
    
    # Check if ESLint is available globally or in project
    local eslint_cmd="eslint"
    if ! command -v eslint >/dev/null 2>&1; then
        # Try npx if global eslint not available
        if [[ -f "$project_dir/package.json" ]] && grep -q "\"eslint\"" "$project_dir/package.json" 2>/dev/null; then
            eslint_cmd="npx eslint"
        else
            log_test_skip "eslint" "$project_name" "ESLint not available"
            return 0
        fi
    fi
    
    # Run eslint
    local output
    if output=$(cd "$project_dir" && $eslint_cmd . --ext .ts,.tsx --max-warnings 0 2>&1); then
        log_test_pass "eslint" "$project_name"
        update_cache "$project_dir" "eslint" "passed" 0 "" 0
        return 0
    else
        log_test_fail "eslint" "$project_name"
        
        # Show first 10 issues
        echo "$output" | grep -E "^[[:space:]]+[0-9]+:[0-9]+" | head -10 | while IFS= read -r line; do
            log_warning "    └─ $line"
        done
        
        update_cache "$project_dir" "eslint" "failed" 0 "$output" 1
        return 1
    fi
}

# Run TypeScript static analysis
run_typescript_static() {
    local start_time
    start_time=$(date +%s)
    
    log_section "TypeScript Static Analysis"
    
    # Load cache
    load_cache "static-typescript"
    
    # Find TypeScript projects first
    log_info "🔍 Discovering TypeScript projects..."
    local projects=()
    while IFS= read -r project; do
        projects+=("$project")
    done < <(find_typescript_projects)

    local total=${#projects[@]}
    if [[ $total -eq 0 ]]; then
        log_warning "No TypeScript projects found"
        return 0
    fi

    log_info "📋 Found $total TypeScript projects to analyze"
    for project in "${projects[@]}"; do
        local project_name
        project_name=$(basename "$project")
        log_info "  - $project_name: $project"
    done

    # Check tools with the found projects
    local tool_status
    tool_status=$(check_typescript_tools "${projects[@]}")
    IFS=':' read -r has_node has_tsc has_eslint <<< "$tool_status"
    
    if [[ "$has_node" != "true" ]]; then
        log_error "Node.js is required for TypeScript analysis"
        return 1
    fi
    
    if [[ "$has_tsc" != "true" ]]; then
        log_warning "TypeScript compiler not found - skipping TypeScript checks"
        return 0
    fi
    
    # Test counters
    local passed=0
    local failed=0
    local skipped=0
    
    # Process each project
    for project in "${projects[@]}"; do
        local project_name
        project_name=$(basename "$project")
        
        # Check if we need to test this project
        local needs_tsc=false
        local needs_eslint=false
        
        if ! check_cache "$project" "tsc"; then
            needs_tsc=true
        else
            ((passed++)) || true
        fi
        
        if [[ "$has_eslint" == "true" ]]; then
            if ! check_cache "$project" "eslint"; then
                needs_eslint=true
            else
                ((passed++)) || true
            fi
        fi
        
        # Run tests if needed
        if [[ "$needs_tsc" == "true" ]]; then
            if run_tsc_check "$project"; then
                ((passed++)) || true
            else
                ((failed++)) || true
            fi
        fi
        
        if [[ "$needs_eslint" == "true" ]] && [[ "$has_eslint" == "true" ]]; then
            if run_eslint_check "$project"; then
                ((passed++)) || true
            else
                ((failed++)) || true
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
    log_section "TypeScript Static Analysis Summary"
    log_info "📊 Results:"
    log_info "  Total Projects: $total"
    log_info "  Tests Passed: $passed"
    if [[ $failed -gt 0 ]]; then
        log_error "  Tests Failed: $failed"
    fi
    if [[ $skipped -gt 0 ]]; then
        log_warning "  Tests Skipped: $skipped"
    fi
    log_info "  Duration: ${duration}s"
    
    if [[ $failed -eq 0 ]]; then
        log_success "✅ All TypeScript static analysis passed!"
        return 0
    else
        return 1
    fi
}

# Main execution
main() {
    if is_dry_run; then
        log_info "[DRY RUN] Would run TypeScript static analysis"
        return 0
    fi
    
    run_typescript_static
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi