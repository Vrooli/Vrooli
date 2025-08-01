#!/bin/bash
# ====================================================================
# Test File Validation Script
# ====================================================================
#
# Validates test files for required patterns and compliance with
# the testing framework standards.
#
# Usage: ./test-validator.sh [OPTIONS]
#
# Options:
#   --fix      - Automatically fix common issues
#   --verbose  - Show detailed output
#   --path     - Specific path to validate (default: all tests)
#
# Exit codes:
#   0 - All tests are compliant
#   1 - Validation failures found
#   2 - Critical validation errors
#
# ====================================================================

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VERBOSE=false
FIX_ISSUES=false
SPECIFIC_PATH=""

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validation results
TOTAL_FILES=0
VALID_FILES=0
ISSUES_FOUND=0
declare -a VALIDATION_ERRORS=()
declare -a WARNINGS=()

# Helper functions
log_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" >&2
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}🐛 $1${NC}"
    fi
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --fix)
                FIX_ISSUES=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --path)
                SPECIFIC_PATH="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [--fix] [--verbose] [--path PATH]"
                echo "Validate test files for framework compliance"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 2
                ;;
        esac
    done
}

# Validate a single test file
validate_test_file() {
    local test_file="$1"
    local file_issues=0
    
    log_verbose "Validating: $test_file"
    
    # Check 1: Resource initialization pattern
    if ! grep -q "HEALTHY_RESOURCES_STR" "$test_file"; then
        VALIDATION_ERRORS+=("$test_file: Missing HEALTHY_RESOURCES_STR initialization pattern")
        ((file_issues++))
        
        if [[ "$FIX_ISSUES" == "true" ]]; then
            fix_resource_initialization "$test_file"
        fi
    fi
    
    # Check 2: Required imports
    if ! grep -q "source.*assertions.sh" "$test_file"; then
        VALIDATION_ERRORS+=("$test_file: Missing assertions.sh import")
        ((file_issues++))
    fi
    
    if ! grep -q "source.*cleanup.sh" "$test_file"; then
        VALIDATION_ERRORS+=("$test_file: Missing cleanup.sh import")
        ((file_issues++))
    fi
    
    # Check 3: Test configuration variables
    if ! grep -q "TEST_RESOURCE=" "$test_file"; then
        WARNINGS+=("$test_file: Missing TEST_RESOURCE variable")
    fi
    
    # Check 4: Main function structure
    if ! grep -q "main()" "$test_file"; then
        VALIDATION_ERRORS+=("$test_file: Missing main() function")
        ((file_issues++))
    fi
    
    # Check 5: Set proper shell options
    if ! grep -q "set -euo pipefail" "$test_file"; then
        WARNINGS+=("$test_file: Missing 'set -euo pipefail' for strict error handling")
    fi
    
    # Check 6: Shebang line
    if ! head -1 "$test_file" | grep -q "#!/bin/bash"; then
        WARNINGS+=("$test_file: Missing or incorrect shebang line")
    fi
    
    ISSUES_FOUND=$((ISSUES_FOUND + file_issues))
    
    if [[ $file_issues -eq 0 ]]; then
        VALID_FILES=$((VALID_FILES + 1))
        log_verbose "✓ $test_file is compliant"
    else
        log_verbose "✗ $test_file has $file_issues issues"
    fi
    
    return $file_issues
}

# Fix resource initialization pattern
fix_resource_initialization() {
    local test_file="$1"
    
    log_info "Fixing resource initialization in $test_file"
    
    # Find the right place to insert the pattern
    # Look for TEST_RESOURCE or API_BASE variable definitions
    local insert_line
    if grep -n "API_BASE=" "$test_file" >/dev/null; then
        insert_line=$(grep -n "API_BASE=" "$test_file" | head -1 | cut -d: -f1)
    elif grep -n "TEST_RESOURCE=" "$test_file" >/dev/null; then
        insert_line=$(grep -n "TEST_RESOURCE=" "$test_file" | head -1 | cut -d: -f1)
    else
        log_warning "Could not find suitable insertion point in $test_file"
        return 1
    fi
    
    # Create backup
    cp "$test_file" "$test_file.backup"
    
    # Insert the pattern after the found line
    {
        head -n "$insert_line" "$test_file"
        echo ""
        echo "# Recreate HEALTHY_RESOURCES array from exported string"
        echo "if [[ -n \"\${HEALTHY_RESOURCES_STR:-}\" ]]; then"
        echo "    HEALTHY_RESOURCES=(\$HEALTHY_RESOURCES_STR)"
        echo "fi"
        echo ""
        tail -n +$((insert_line + 1)) "$test_file"
    } > "$test_file.tmp"
    
    mv "$test_file.tmp" "$test_file"
    log_success "Fixed resource initialization in $test_file"
}

# Find all test files
find_test_files() {
    local search_path="${SPECIFIC_PATH:-$TESTS_DIR}"
    
    if [[ -f "$search_path" ]]; then
        echo "$search_path"
    else
        find "$search_path" -name "*.test.sh" -type f | sort
    fi
}

# Main validation function
run_validation() {
    local test_files
    test_files=$(find_test_files)
    
    if [[ -z "$test_files" ]]; then
        log_error "No test files found in ${SPECIFIC_PATH:-$TESTS_DIR}"
        exit 2
    fi
    
    log_info "Starting test file validation..."
    log_info "Search path: ${SPECIFIC_PATH:-$TESTS_DIR}"
    log_info "Fix mode: $FIX_ISSUES"
    echo
    
    while IFS= read -r test_file; do
        TOTAL_FILES=$((TOTAL_FILES + 1))
        validate_test_file "$test_file"
    done <<< "$test_files"
}

# Generate validation report
generate_report() {
    echo
    echo "=========================================="
    echo "            VALIDATION REPORT            "
    echo "=========================================="
    echo
    echo "📊 Summary:"
    echo "   Total files validated: $TOTAL_FILES"
    echo "   Compliant files: $VALID_FILES"
    echo "   Files with issues: $((TOTAL_FILES - VALID_FILES))"
    echo "   Total issues found: $ISSUES_FOUND"
    echo
    
    if [[ ${#VALIDATION_ERRORS[@]} -gt 0 ]]; then
        echo "🚨 Critical Issues:"
        for error in "${VALIDATION_ERRORS[@]}"; do
            log_error "$error"
        done
        echo
    fi
    
    if [[ ${#WARNINGS[@]} -gt 0 ]]; then
        echo "⚠️  Warnings:"
        for warning in "${WARNINGS[@]}"; do
            log_warning "$warning"
        done
        echo
    fi
    
    if [[ $ISSUES_FOUND -eq 0 && ${#WARNINGS[@]} -eq 0 ]]; then
        log_success "All test files are compliant! 🎉"
        return 0
    elif [[ $ISSUES_FOUND -eq 0 ]]; then
        log_info "All critical issues resolved. Only warnings remain."
        return 0
    else
        log_error "$ISSUES_FOUND critical issues found."
        if [[ "$FIX_ISSUES" != "true" ]]; then
            echo
            log_info "Run with --fix to automatically resolve common issues."
        fi
        return 1
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    # Validate script directory exists
    if [[ ! -d "$TESTS_DIR" ]]; then
        log_error "Tests directory not found: $TESTS_DIR"
        exit 2
    fi
    
    run_validation
    generate_report
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi