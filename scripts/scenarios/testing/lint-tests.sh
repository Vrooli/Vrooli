#!/bin/bash
# Test Script Safety Linter
# Scans test scripts for potentially dangerous patterns that could cause data loss
set -euo pipefail

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_FILES=0
ISSUES_FOUND=0
WARNINGS_FOUND=0

# Function to print colored output
print_error() { echo -e "${RED}❌ ERROR: $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  WARNING: $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ️  INFO: $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }

# Function to scan a single file
scan_file() {
    local file="$1"
    local file_issues=0
    local file_warnings=0
    
    echo ""
    print_info "Scanning: $file"
    
    # Check for dangerous rm patterns in BATS teardown functions specifically
    if [[ "$file" == *.bats ]]; then
        # Extract teardown function and check for dangerous patterns
        local teardown_section=$(sed -n '/^teardown()/,/^}/p' "$file")
        if echo "$teardown_section" | grep -q 'rm.*\$.*\*'; then
            if ! echo "$teardown_section" | grep -q '\[ -n.*\$'; then
                print_error "BATS teardown has unguarded rm with variables"
                print_info "Found in teardown() function"
                print_info "Pattern: rm \$VAR* without [ -n \"\$VAR\" ] check"
                print_info "Fix: Add variable validation in teardown()"
                ((file_issues++))
            fi
        fi
    fi
    
    # Check for unguarded rm with wildcards (skip if it's in a protected block)
    local unguarded_rm=$(grep -n 'rm.*\*.*2>/dev/null' "$file" 2>/dev/null || true)
    if [ -n "$unguarded_rm" ]; then
        # Check if any are truly unguarded (not in if blocks checking variables)
        local truly_unguarded=""
        while IFS= read -r line; do
            local line_num=$(echo "$line" | cut -d: -f1)
            local start_check=$((line_num - 3))
            [ $start_check -lt 1 ] && start_check=1
            
            local context=$(sed -n "${start_check},${line_num}p" "$file")
            if ! echo "$context" | grep -q 'if \[ -n.*\$\|if \[\[ .*-n.*\$'; then
                truly_unguarded="${truly_unguarded}${line}\n"
            fi
        done <<< "$unguarded_rm"
        
        if [ -n "$truly_unguarded" ]; then
            print_error "Unguarded rm with wildcards found"
            echo -e "$truly_unguarded"
            print_info "Pattern: rm * without variable validation"
            print_info "Fix: Add path validation before rm command"
            ((file_issues++))
        fi
    fi
    
    # Check BATS teardown patterns
    if [[ "$file" == *.bats ]] && grep -A 10 '^teardown()' "$file" 2>/dev/null | grep -q 'rm.*\$'; then
        if ! grep -A 10 '^teardown()' "$file" 2>/dev/null | grep -q '\[ -n.*\$'; then
            print_error "BATS teardown without variable validation"
            print_info "BATS teardown functions run even when tests are skipped"
            print_info "Fix: Add [ -n \"\${VAR:-}\" ] check before rm commands"
            ((file_issues++))
        fi
    fi
    
    # Check for rm with root or dangerous paths
    if grep -n 'rm.*["\x27]/["\x27]' "$file" 2>/dev/null; then
        print_error "Potential rm of root directory found"
        print_info "Pattern: rm involving root path \"/\""
        print_info "Fix: Add path validation to prevent root deletion"
        ((file_issues++))
    fi
    
    # Check for TEST_FILE_PREFIX in setup vs teardown
    if [[ "$file" == *.bats ]]; then
        if grep -q '^setup()' "$file" 2>/dev/null; then
            local setup_line=$(grep -n 'TEST_FILE_PREFIX' "$file" | grep -v '^#' | head -1 | cut -d: -f1)
            local skip_line=$(grep -n 'skip' "$file" | head -1 | cut -d: -f1)
            
            if [ -n "$setup_line" ] && [ -n "$skip_line" ] && [ "$skip_line" -lt "$setup_line" ]; then
                print_error "TEST_FILE_PREFIX set after skip condition in setup()"
                print_info "This causes teardown() to run with empty variables"
                print_info "Fix: Move TEST_FILE_PREFIX assignment before skip conditions"
                ((file_issues++))
            fi
        fi
    fi
    
    # Warnings for potentially unsafe patterns
    
    # Check for rm without explicit path validation
    if grep -n '^[[:space:]]*rm ' "$file" 2>/dev/null | grep -v '/tmp/' | grep -v 'rm.*-.*tmp'; then
        print_warning "rm command without explicit /tmp path restriction"
        print_info "Consider restricting cleanup to /tmp directories only"
        ((file_warnings++))
    fi
    
    # Check for missing error handling on rm
    if grep -n 'rm.*\$' "$file" 2>/dev/null | grep -v '2>/dev/null\||| true'; then
        print_warning "rm command without error handling"  
        print_info "Consider adding '2>/dev/null || true' for graceful failure"
        ((file_warnings++))
    fi
    
    # Check for hardcoded paths that might be dangerous
    if grep -n '/home/\|/usr/\|/opt/\|/var/' "$file" 2>/dev/null | grep 'rm\|cp\|mv'; then
        print_warning "File operations on system directories detected"
        print_info "Ensure these operations are safe and intentional"
        ((file_warnings++))
    fi
    
    if [ $file_issues -eq 0 ] && [ $file_warnings -eq 0 ]; then
        print_success "No issues found"
    else
        echo ""
        print_info "File summary: $file_issues errors, $file_warnings warnings"
    fi
    
    ((ISSUES_FOUND += file_issues))
    ((WARNINGS_FOUND += file_warnings))
    ((TOTAL_FILES++))
}

# Main execution
main() {
    local search_path="${1:-.}"
    
    echo ""
    print_info "🔍 Test Script Safety Linter"
    print_info "Scanning for dangerous patterns in test scripts..."
    print_info "Search path: $search_path"
    
    # Find test scripts
    local test_files=()
    while IFS= read -r -d '' file; do
        test_files+=("$file")
    done < <(find "$search_path" \( -name "*.bats" -o -name "*test*.sh" -o -name "test-*.sh" \) -type f -print0)
    
    if [ ${#test_files[@]} -eq 0 ]; then
        print_warning "No test files found in $search_path"
        exit 0
    fi
    
    print_info "Found ${#test_files[@]} test files to scan"
    
    # Scan each file
    for file in "${test_files[@]}"; do
        scan_file "$file"
    done
    
    # Summary
    echo ""
    print_info "📊 Scan Summary"
    echo "Files scanned: $TOTAL_FILES"
    echo "Errors found: $ISSUES_FOUND" 
    echo "Warnings found: $WARNINGS_FOUND"
    
    if [ $ISSUES_FOUND -eq 0 ] && [ $WARNINGS_FOUND -eq 0 ]; then
        print_success "All test scripts passed safety checks!"
        exit 0
    elif [ $ISSUES_FOUND -eq 0 ]; then
        print_warning "No errors, but $WARNINGS_FOUND warnings found"
        print_info "Consider reviewing warnings for potential improvements"
        exit 0
    else
        print_error "$ISSUES_FOUND critical safety issues found"
        print_info "Please fix errors before committing test scripts"
        exit 1
    fi
}

# Show usage if no arguments
if [ $# -eq 0 ]; then
    cat << 'EOF'
🔍 Test Script Safety Linter

Usage: lint-tests.sh [path]

Scans test scripts for potentially dangerous patterns that could cause data loss.

Examples:
  lint-tests.sh                           # Scan current directory
  lint-tests.sh scenarios/my-scenario/    # Scan specific scenario
  lint-tests.sh test/                     # Scan test directory

Detected Issues:
  ❌ Dangerous wildcard patterns (rm $VAR*)
  ❌ Unguarded cleanup operations  
  ❌ BATS teardown without variable validation
  ❌ Operations on root/system directories
  ❌ Variables set after skip conditions

Warnings:
  ⚠️  Missing error handling on rm commands
  ⚠️  Operations outside /tmp directories
  ⚠️  File operations on system paths

Exit codes:
  0 - No issues found or warnings only
  1 - Critical safety issues found

See: scripts/scenarios/testing/SAFETY-GUIDELINES.md
EOF
    exit 0
fi

main "$@"