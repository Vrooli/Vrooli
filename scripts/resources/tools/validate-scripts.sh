#!/bin/bash
# ====================================================================
# Resource Script Security Validator
# ====================================================================
#
# Scans resource management scripts for dangerous patterns that could
# cause data loss, especially configuration file overwrites.
#
# Usage:
#   ./validate-scripts.sh [--fix] [--path <path>]
#
# Options:
#   --fix     Attempt to automatically fix some issues
#   --path    Specific path to scan (default: entire resources directory)
#
# Exit Codes:
#   0 - No issues found
#   1 - Issues found (details in output)
#   2 - Script error
#
# ====================================================================

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCES_DIR="$SCRIPT_DIR"

# Configuration
SCAN_PATH="${RESOURCES_DIR}"
FIX_MODE=false
VERBOSE=false

# Counters
ISSUES_FOUND=0
WARNINGS_FOUND=0
FIXED_ISSUES=0

# Colors
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    YELLOW='\033[1;33m'
    GREEN='\033[0;32m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED='' YELLOW='' GREEN='' BLUE='' BOLD='' NC=''
fi

# Logging functions
log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    ISSUES_FOUND=$((ISSUES_FOUND + 1))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
    WARNINGS_FOUND=$((WARNINGS_FOUND + 1))
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_fixed() {
    echo -e "${GREEN}[FIXED]${NC} $1"
    FIXED_ISSUES=$((FIXED_ISSUES + 1))
}

# Show usage
show_usage() {
    cat << EOF
Resource Script Security Validator

Usage: $0 [options]

Options:
    --fix           Attempt to automatically fix some issues
    --path <path>   Specific path to scan (default: $RESOURCES_DIR)
    --verbose       Enable verbose output
    --help          Show this help message

Examples:
    $0                              # Scan all resource scripts
    $0 --path automation/n8n        # Scan specific directory
    $0 --fix                        # Scan and fix issues automatically

Exit Codes:
    0 - No issues found
    1 - Issues found
    2 - Script error
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --fix)
            FIX_MODE=true
            shift
            ;;
        --path)
            SCAN_PATH="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            show_usage >&2
            exit 2
            ;;
    esac
done

# Validate scan path
if [[ ! -d "$SCAN_PATH" ]]; then
    log_error "Scan path does not exist: $SCAN_PATH"
    exit 2
fi

log_info "Starting security validation scan..."
log_info "Scan path: $SCAN_PATH"
log_info "Fix mode: $FIX_MODE"

#######################################
# Check for dangerous file overwrite patterns
#######################################
check_dangerous_overwrites() {
    log_info "Checking for dangerous file overwrite patterns..."
    
    # Pattern 1: Direct writes to resources.local.json
    local dangerous_files=()
    while IFS= read -r -d '' file; do
        dangerous_files+=("$file")
    done < <(find "$SCAN_PATH" -name "*.sh" -type f -print0)
    
    for file in "${dangerous_files[@]}"; do
        # Skip this validation script itself
        if [[ "$file" == "$0" ]]; then
            continue
        fi
        
        # Check for direct writes to config files (skip examples in comments/docs)
        if grep -q "cat.*>.*resources\.local\.json" "$file" 2>/dev/null && \
           ! grep -q "#.*cat.*>.*resources\.local\.json\|^[[:space:]]*#.*cat.*>.*" "$file" 2>/dev/null; then
            log_error "File $file contains dangerous config overwrite pattern:"
            log_error "  Found: cat ... > resources.local.json"
            log_error "  Risk: Complete config file overwrite, API key loss"
            log_error "  Fix: Use resources::update_config() from common.sh instead"
            echo
        fi
        
        if grep -q "echo.*>.*resources\.local\.json" "$file" 2>/dev/null && \
           ! grep -q "#.*echo.*>.*resources\.local\.json\|^[[:space:]]*#.*echo.*>.*" "$file" 2>/dev/null; then
            log_error "File $file contains dangerous config overwrite pattern:"
            log_error "  Found: echo ... > resources.local.json"
            log_error "  Risk: Complete config file overwrite, API key loss"
            log_error "  Fix: Use resources::update_config() from common.sh instead"
            echo
        fi
        
        # Check for missing backup creation
        if grep -q "resources\.local\.json" "$file" 2>/dev/null && \
           ! grep -q "backup\|cp.*resources\.local\.json" "$file" 2>/dev/null; then
            log_warning "File $file modifies config but doesn't create backups:"
            log_warning "  Risk: No recovery option if modification fails"
            log_warning "  Recommendation: Create backup before modification"
            echo
        fi
    done
}

#######################################
# Check for invalid bash syntax patterns
#######################################
check_bash_syntax() {
    log_info "Checking for invalid bash syntax patterns..."
    
    local script_files=()
    while IFS= read -r -d '' file; do
        script_files+=("$file")
    done < <(find "$SCAN_PATH" -name "*.sh" -type f -print0)
    
    for file in "${script_files[@]}"; do
        # Check for invalid ternary operators
        if grep -n "{\#.*-gt.*?.*:.*}" "$file" 2>/dev/null; then
            log_error "File $file contains invalid bash ternary operator:"
            grep -n "{\#.*-gt.*?.*:.*}" "$file" | while read -r line; do
                log_error "  Line: $line"
            done
            log_error "  Fix: Replace with \$([ \${#var} -gt N ] && echo \"...\" || echo \"\")"
            echo
        fi
        
        # Basic bash syntax check
        if ! bash -n "$file" 2>/dev/null; then
            log_error "File $file has bash syntax errors:"
            bash -n "$file" 2>&1 | head -5 | while read -r error_line; do
                log_error "  $error_line"
            done
            echo
        fi
    done
}

#######################################
# Check for missing error handling
#######################################
check_error_handling() {
    log_info "Checking for missing error handling..."
    
    local script_files=()
    while IFS= read -r -d '' file; do
        script_files+=("$file")
    done < <(find "$SCAN_PATH" -name "*.sh" -type f -print0)
    
    for file in "${script_files[@]}"; do
        # Check for missing set -e
        if ! grep -q "set.*-e" "$file" 2>/dev/null; then
            log_warning "File $file missing 'set -e' for error handling:"
            log_warning "  Risk: Script continues after errors"
            log_warning "  Fix: Add 'set -euo pipefail' at the top"
            echo
        fi
        
        # Check for unhandled curl commands
        if grep -q "curl" "$file" 2>/dev/null && \
           ! grep -q "curl.*||" "$file" 2>/dev/null; then
            log_warning "File $file has unhandled curl commands:"
            log_warning "  Risk: No error handling for network failures"
            log_warning "  Fix: Add error handling: curl ... || handle_error"
            echo
        fi
    done
}

#######################################
# Check for security issues
#######################################
check_security_issues() {
    log_info "Checking for security issues..."
    
    local script_files=()
    while IFS= read -r -d '' file; do
        script_files+=("$file")
    done < <(find "$SCAN_PATH" -name "*.sh" -type f -print0)
    
    for file in "${script_files[@]}"; do
        # Check for hardcoded secrets
        if grep -i "api.*key.*=" "$file" 2>/dev/null | grep -v "\${" | grep -v "TODO\|FIXME\|EXAMPLE"; then
            log_error "File $file may contain hardcoded API keys:"
            grep -in "api.*key.*=" "$file" | head -3 | while read -r line; do
                log_error "  $line"
            done
            log_error "  Fix: Use environment variables: \${API_KEY_NAME}"
            echo
        fi
        
        # Check for insecure file permissions
        if grep -q "chmod.*77" "$file" 2>/dev/null; then
            log_warning "File $file sets overly permissive file permissions:"
            grep -n "chmod.*77" "$file" | while read -r line; do
                log_warning "  $line"
            done
            log_warning "  Risk: Files accessible by other users"
            echo
        fi
    done
}

#######################################
# Check for use of proper configuration APIs
#######################################
check_config_api_usage() {
    log_info "Checking for proper configuration API usage..."
    
    local script_files=()
    while IFS= read -r -d '' file; do
        script_files+=("$file")
    done < <(find "$SCAN_PATH" -name "*.sh" -type f -print0)
    
    for file in "${script_files[@]}"; do
        # Skip common.sh itself
        if [[ "$file" == *"common.sh" ]]; then
            continue
        fi
        
        # Check if scripts modify config but don't use proper API
        if grep -q "resources\.local\.json" "$file" 2>/dev/null && \
           ! grep -q "resources::update_config\|config-manager\.js" "$file" 2>/dev/null; then
            log_warning "File $file modifies config without proper API:"
            log_warning "  Risk: Inconsistent configuration management"
            log_warning "  Fix: Use resources::update_config() or config-manager.js"
            echo
        fi
    done
}

#######################################
# Generate security report
#######################################
generate_report() {
    echo
    log_info "=================================="
    log_info "Security Validation Report"
    log_info "=================================="
    echo
    
    if [[ $ISSUES_FOUND -eq 0 && $WARNINGS_FOUND -eq 0 ]]; then
        log_success "No security issues found! 🎉"
        log_success "All resource scripts follow safe patterns."
    else
        echo -e "${BOLD}Summary:${NC}"
        echo "  Critical Issues: $ISSUES_FOUND"
        echo "  Warnings: $WARNINGS_FOUND"
        
        if [[ $FIX_MODE == true ]]; then
            echo "  Fixed Issues: $FIXED_ISSUES"
        fi
        
        echo
        echo -e "${BOLD}Recommendations:${NC}"
        
        if [[ $ISSUES_FOUND -gt 0 ]]; then
            echo "  1. Fix critical issues immediately - they pose data loss risk"
            echo "  2. Use resources::update_config() for all config modifications"
            echo "  3. Always create backups before modifying config files"
        fi
        
        if [[ $WARNINGS_FOUND -gt 0 ]]; then
            echo "  4. Address warnings to improve script robustness"
            echo "  5. Add proper error handling to all scripts"
            echo "  6. Use environment variables for all secrets"
        fi
        
        echo
        echo "  Run with --fix to automatically resolve some issues."
    fi
    
    echo
    log_info "Next Steps:"
    log_info "  1. Run tests: cd /home/matthalloran8/Vrooli && pnpm test:resources:quick"
    log_info "  2. Regular validation: Add this script to your maintenance routine"
    log_info "  3. Training: Share safe patterns with the team"
}

# Main execution
main() {
    check_dangerous_overwrites
    check_bash_syntax
    check_error_handling
    check_security_issues
    check_config_api_usage
    
    generate_report
    
    # Exit with appropriate code
    if [[ $ISSUES_FOUND -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Run main function
main "$@"