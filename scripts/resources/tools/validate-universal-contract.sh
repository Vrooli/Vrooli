#!/usr/bin/env bash
# ============================================================================
# Universal Contract Validation Tool (v2.0)
# ============================================================================
#
# Validates resources against the universal contract specification (v2.0).
# Supports three validation layers: syntax, behavioral, and integration.
# Maintains backward compatibility with v1.0 contracts during migration.
#
# Usage:
#   ./validate-universal-contract.sh [OPTIONS]
#
# Options:
#   --resource <name>    Validate specific resource
#   --layer <1|2|3>      Validation layer (default: 1)
#   --contract <ver>     Contract version: v1.0|v2.0 (default: v2.0)
#   --verbose            Show detailed validation output
#   --fix                Attempt to fix issues (where possible)
#   --report <file>      Save validation report
#   --format <type>      Output format: text|json|junit (default: text)
#
# Validation Layers:
#   Layer 1: Syntax - File structure, command registration, basic checks
#   Layer 2: Behavioral - Command execution, exit codes, output format
#   Layer 3: Integration - Full lifecycle, content management, performance
#
# Exit Codes:
#   0 - Validation passed
#   1 - Validation failed
#   2 - Script error
# ============================================================================

set -euo pipefail

# Script setup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tools"
RESOURCES_DIR="${APP_ROOT}/resources"
CONTRACTS_DIR="${APP_ROOT}/scripts/resources/contracts"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}" 2>/dev/null || {
    # Fallback logging
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*"; }
    log::success() { echo "[SUCCESS] $*"; }
}

# Configuration
SPECIFIC_RESOURCE=""
VALIDATION_LAYER=1
CONTRACT_VERSION="v2.0"
VERBOSE=false
FIX_ISSUES=false
REPORT_FILE=""
OUTPUT_FORMAT="text"

# Validation results
declare -A VALIDATION_RESULTS=()
declare -A VALIDATION_ERRORS=()
declare -A VALIDATION_WARNINGS=()
declare -A VALIDATION_FIXES=()
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_COUNT=0

#######################################
# Show help
#######################################
show_help() {
    cat << EOF
Universal Contract Validation Tool (v2.0)

Validates resources against the universal contract specification with
support for three validation layers and backward compatibility.

Usage: $(basename "$0") [OPTIONS]

Options:
    --resource <name>    Validate specific resource
    --layer <1|2|3>      Validation layer (default: 1)
    --contract <ver>     Contract version: v1.0|v2.0 (default: v2.0)
    --verbose            Show detailed validation output
    --fix                Attempt to fix issues
    --report <file>      Save validation report
    --format <type>      Output format: text|json|junit
    --help               Show this help message

Validation Layers:
    Layer 1: Syntax Validation
        - Required files exist (cli.sh, lib/core.sh, etc.)
        - Shell syntax is valid
        - Required commands are registered
        - File permissions are correct
    
    Layer 2: Behavioral Validation
        - Commands execute without error
        - Exit codes match specification
        - Output format is correct
        - Flags are handled properly
    
    Layer 3: Integration Validation
        - Complete lifecycle works
        - Content management flow
        - Resource injection support
        - Performance requirements

Examples:
    # Quick syntax check all resources
    $(basename "$0")
    
    # Full validation of specific resource
    $(basename "$0") --resource ollama --layer 3
    
    # Check v1.0 compatibility
    $(basename "$0") --contract v1.0 --resource node-red
    
    # Generate JSON report
    $(basename "$0") --format json --report validation.json

Exit Codes:
    0 - All validations passed
    1 - One or more validations failed
    2 - Script error

For contract details, see: scripts/resources/contracts/v2.0/universal.yaml
EOF
}

#######################################
# Parse arguments
#######################################
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --resource)
                SPECIFIC_RESOURCE="$2"
                shift 2
                ;;
            --layer)
                VALIDATION_LAYER="$2"
                shift 2
                ;;
            --contract)
                CONTRACT_VERSION="$2"
                shift 2
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --fix)
                FIX_ISSUES=true
                shift
                ;;
            --report)
                REPORT_FILE="$2"
                shift 2
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done
}

#######################################
# Layer 1: Syntax validation
#######################################
validate_layer1_syntax() {
    local resource_dir="$1"
    local resource_name="$(basename "$resource_dir")"
    
    log::info "Layer 1: Syntax validation for $resource_name"
    
    # Check required files based on contract version
    if [[ "$CONTRACT_VERSION" == "v2.0" ]]; then
        validate_v2_file_structure "$resource_dir" "$resource_name"
        validate_v2_command_registration "$resource_dir" "$resource_name"
    else
        validate_v1_file_structure "$resource_dir" "$resource_name"
        validate_v1_command_registration "$resource_dir" "$resource_name"
    fi
    
    # Common validations
    validate_shell_syntax "$resource_dir" "$resource_name"
    validate_file_permissions "$resource_dir" "$resource_name"
    
    # V2.0 specific pattern compliance checks
    if [[ "$CONTRACT_VERSION" == "v2.0" ]]; then
        validate_v2_pattern_compliance "$resource_dir" "$resource_name"
    fi
}

#######################################
# Validate v2.0 file structure
#######################################
validate_v2_file_structure() {
    local resource_dir="$1"
    local resource_name="$2"
    local check_passed=true
    
    # Required files for v2.0
    local required_files=(
        "cli.sh"
        "lib/core.sh"
        "lib/test.sh"
        "config/defaults.sh"
    )
    
    for file in "${required_files[@]}"; do
        ((TOTAL_CHECKS++))
        if [[ -f "$resource_dir/$file" ]]; then
            ((PASSED_CHECKS++))
            [[ "$VERBOSE" == "true" ]] && log::success "  ✓ $file exists"
        else
            ((FAILED_CHECKS++))
            check_passed=false
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Missing: $file; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ $file missing"
            
            # Suggest fix
            if [[ "$FIX_ISSUES" == "true" ]]; then
                VALIDATION_FIXES["$resource_name"]="${VALIDATION_FIXES[$resource_name]:-}Create $file; "
            fi
        fi
    done
    
    # Check for deprecated files
    local deprecated_files=(
        "manage.sh"
        "manage.bats"
        "inject.sh"
    )
    
    for file in "${deprecated_files[@]}"; do
        if [[ -f "$resource_dir/$file" ]]; then
            ((WARNING_COUNT++))
            VALIDATION_WARNINGS["$resource_name"]="${VALIDATION_WARNINGS[$resource_name]:-}Deprecated: $file; "
            [[ "$VERBOSE" == "true" ]] && log::warning "  ⚠ $file is deprecated"
        fi
    done
    
    return $([ "$check_passed" == "true" ] && echo 0 || echo 1)
}

#######################################
# Validate v1.0 file structure (backward compatibility)
#######################################
validate_v1_file_structure() {
    local resource_dir="$1"
    local resource_name="$2"
    local check_passed=true
    
    # Required files for v1.0 (legacy)
    local required_files=(
        "manage.sh"
        "config/defaults.sh"
        "config/messages.sh"
        "lib/common.sh"
    )
    
    for file in "${required_files[@]}"; do
        ((TOTAL_CHECKS++))
        if [[ -f "$resource_dir/$file" ]]; then
            ((PASSED_CHECKS++))
            [[ "$VERBOSE" == "true" ]] && log::success "  ✓ $file exists"
        else
            ((FAILED_CHECKS++))
            check_passed=false
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Missing: $file; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ $file missing"
        fi
    done
    
    return $([ "$check_passed" == "true" ] && echo 0 || echo 1)
}

#######################################
# Validate v2.0 command registration
#######################################
validate_v2_command_registration() {
    local resource_dir="$1"
    local resource_name="$2"
    local cli_file="$resource_dir/cli.sh"
    
    if [[ ! -f "$cli_file" ]]; then
        return 1
    fi
    
    # Required v2.0 commands (base level)
    local required_commands=(
        "help"
        "status"
        "logs"
        "content"
        "test"
        "manage"
    )
    
    for cmd in "${required_commands[@]}"; do
        ((TOTAL_CHECKS++))
        if grep -q "cli::register_command.*[\"']${cmd}[\"']" "$cli_file" 2>/dev/null || \
           grep -q "# Subcommands under" "$cli_file" 2>/dev/null; then
            ((PASSED_CHECKS++))
            [[ "$VERBOSE" == "true" ]] && log::success "  ✓ Command registered: $cmd"
        else
            ((FAILED_CHECKS++))
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Missing command: $cmd; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ Command missing: $cmd"
        fi
    done
}

#######################################
# Validate v2.0 pattern compliance  
# Checks for deprecated patterns that need migration
#######################################
validate_v2_pattern_compliance() {
    local resource_dir="$1"
    local resource_name="$2"
    local check_passed=true
    
    # Check for manage.sh --action patterns (should be resource-* manage *)
    while IFS= read -r -d '' file; do
        if [[ "$file" =~ \.backup\. ]] || [[ "$file" =~ \.bak\. ]]; then
            continue
        fi
        
        local violations_count=0
        while IFS= read -r line_num line_content; do
            ((violations_count++))
            ((TOTAL_CHECKS++))
            ((FAILED_CHECKS++))
            check_passed=false
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Line $line_num: Old manage.sh --action pattern: ${line_content:0:80}...; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ $(basename "$file"):$line_num: manage.sh --action pattern found"
        done < <(grep -n "manage\.sh --action" "$file" 2>/dev/null || true)
        
        if [[ $violations_count -gt 0 ]] && [[ "$FIX_ISSUES" == "true" ]]; then
            VALIDATION_FIXES["$resource_name"]="${VALIDATION_FIXES[$resource_name]:-}Convert manage.sh --action to resource-$resource_name manage; "
        fi
    done < <(find "$resource_dir" \( -name "*.sh" -o -name "*.md" -o -name "*.json" \) -type f -print0 2>/dev/null)
    
    # Check for resource-* inject patterns (should be resource-* content add)
    while IFS= read -r -d '' file; do
        if [[ "$file" =~ \.backup\. ]] || [[ "$file" =~ \.bak\. ]]; then
            continue
        fi
        
        local violations_count=0
        while IFS= read -r line_num line_content; do
            ((violations_count++))
            ((TOTAL_CHECKS++))
            ((FAILED_CHECKS++))
            check_passed=false
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Line $line_num: Old inject pattern: ${line_content:0:80}...; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ $(basename "$file"):$line_num: resource-* inject pattern found"
        done < <(grep -n "resource-[a-z0-9-]* inject" "$file" 2>/dev/null || true)
        
        if [[ $violations_count -gt 0 ]] && [[ "$FIX_ISSUES" == "true" ]]; then
            VALIDATION_FIXES["$resource_name"]="${VALIDATION_FIXES[$resource_name]:-}Convert resource-$resource_name inject to resource-$resource_name content add; "
        fi
    done < <(find "$resource_dir" \( -name "*.sh" -o -name "*.md" -o -name "*.bats" \) -type f -print0 2>/dev/null)
    
    # Check for curl health check patterns (should use CLI)
    while IFS= read -r -d '' file; do
        local violations_count=0
        while IFS= read -r line_num line_content; do
            ((violations_count++))
            ((TOTAL_CHECKS++))
            ((WARNING_COUNT++))
            VALIDATION_WARNINGS["$resource_name"]="${VALIDATION_WARNINGS[$resource_name]:-}Line $line_num: Direct curl health check: ${line_content:0:80}...; "
            [[ "$VERBOSE" == "true" ]] && log::warning "  ⚠ $(basename "$file"):$line_num: curl health check should use CLI"
        done < <(grep -n "curl.*localhost:[0-9]*/health" "$file" 2>/dev/null || true)
        
        if [[ $violations_count -gt 0 ]] && [[ "$FIX_ISSUES" == "true" ]]; then
            VALIDATION_FIXES["$resource_name"]="${VALIDATION_FIXES[$resource_name]:-}Replace curl health checks with resource-$resource_name test smoke; "
        fi
    done < <(find "$resource_dir" -name "*.sh" -type f -print0 2>/dev/null)
    
    # Check for resource-{name} placeholder patterns
    while IFS= read -r -d '' file; do
        local violations_count=0
        while IFS= read -r line_num line_content; do
            ((violations_count++))
            ((TOTAL_CHECKS++))
            ((WARNING_COUNT++))
            VALIDATION_WARNINGS["$resource_name"]="${VALIDATION_WARNINGS[$resource_name]:-}Line $line_num: Generic placeholder pattern: ${line_content:0:80}...; "
            [[ "$VERBOSE" == "true" ]] && log::warning "  ⚠ $(basename "$file"):$line_num: resource-{name} placeholder found"
        done < <(grep -n "resource-{name}" "$file" 2>/dev/null || true)
        
        if [[ $violations_count -gt 0 ]] && [[ "$FIX_ISSUES" == "true" ]]; then
            VALIDATION_FIXES["$resource_name"]="${VALIDATION_FIXES[$resource_name]:-}Replace resource-{name} with resource-$resource_name; "
        fi
    done < <(find "$resource_dir" -name "*.sh" -type f -print0 2>/dev/null)
    
    return $([ "$check_passed" == "true" ] && echo 0 || echo 1)
}

#######################################
# Validate v1.0 command registration (backward compatibility)
#######################################
validate_v1_command_registration() {
    local resource_dir="$1"
    local resource_name="$2"
    local manage_file="$resource_dir/manage.sh"
    
    if [[ ! -f "$manage_file" ]]; then
        return 1
    fi
    
    # Required v1.0 actions
    local required_actions=(
        "install"
        "start"
        "stop"
        "status"
        "logs"
    )
    
    for action in "${required_actions[@]}"; do
        ((TOTAL_CHECKS++))
        if grep -q "\"$action\")" "$manage_file" 2>/dev/null || \
           grep -q "$action)" "$manage_file" 2>/dev/null; then
            ((PASSED_CHECKS++))
            [[ "$VERBOSE" == "true" ]] && log::success "  ✓ Action found: $action"
        else
            ((FAILED_CHECKS++))
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Missing action: $action; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ Action missing: $action"
        fi
    done
}

#######################################
# Validate shell syntax
#######################################
validate_shell_syntax() {
    local resource_dir="$1"
    local resource_name="$2"
    local check_passed=true
    
    # Check all .sh files for syntax errors
    while IFS= read -r -d '' file; do
        ((TOTAL_CHECKS++))
        local filename="$(basename "$file")"
        
        if bash -n "$file" 2>/dev/null; then
            ((PASSED_CHECKS++))
            [[ "$VERBOSE" == "true" ]] && log::success "  ✓ Syntax OK: $filename"
        else
            ((FAILED_CHECKS++))
            check_passed=false
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Syntax error in $filename; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ Syntax error: $filename"
        fi
    done < <(find "$resource_dir" -name "*.sh" -type f -print0 2>/dev/null)
    
    return $([ "$check_passed" == "true" ] && echo 0 || echo 1)
}

#######################################
# Validate file permissions
#######################################
validate_file_permissions() {
    local resource_dir="$1"
    local resource_name="$2"
    
    # Check cli.sh or manage.sh is executable
    local main_script=""
    if [[ -f "$resource_dir/cli.sh" ]]; then
        main_script="$resource_dir/cli.sh"
    elif [[ -f "$resource_dir/manage.sh" ]]; then
        main_script="$resource_dir/manage.sh"
    fi
    
    if [[ -n "$main_script" ]]; then
        ((TOTAL_CHECKS++))
        if [[ -x "$main_script" ]]; then
            ((PASSED_CHECKS++))
            [[ "$VERBOSE" == "true" ]] && log::success "  ✓ $(basename "$main_script") is executable"
        else
            ((FAILED_CHECKS++))
            VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}$(basename "$main_script") not executable; "
            [[ "$VERBOSE" == "true" ]] && log::error "  ✗ $(basename "$main_script") not executable"
            
            if [[ "$FIX_ISSUES" == "true" ]]; then
                chmod +x "$main_script"
                log::success "  Fixed: Made $(basename "$main_script") executable"
            fi
        fi
    fi
}

#######################################
# Layer 2: Behavioral validation
#######################################
validate_layer2_behavioral() {
    local resource_dir="$1"
    local resource_name="$(basename "$resource_dir")"
    
    log::info "Layer 2: Behavioral validation for $resource_name"
    
    # Test help command
    test_command_execution "$resource_dir" "$resource_name" "help" 0
    
    # Test status command
    test_command_execution "$resource_dir" "$resource_name" "status" "0|1|2"
    
    # Test output format
    test_command_output_format "$resource_dir" "$resource_name"
    
    # Test flag handling
    test_flag_handling "$resource_dir" "$resource_name"
}

#######################################
# Test command execution
#######################################
test_command_execution() {
    local resource_dir="$1"
    local resource_name="$2"
    local command="$3"
    local expected_exit_codes="$4"
    
    local cli_path=""
    if [[ -f "$resource_dir/cli.sh" ]]; then
        cli_path="$resource_dir/cli.sh"
    elif [[ -f "$resource_dir/manage.sh" ]]; then
        cli_path="$resource_dir/manage.sh"
        command="--action $command"
    else
        return 1
    fi
    
    ((TOTAL_CHECKS++))
    
    # Execute command and capture exit code
    local exit_code=0
    if timeout 10 bash "$cli_path" $command >/dev/null 2>&1; then
        exit_code=$?
    else
        exit_code=$?
    fi
    
    # Check if exit code matches expected
    if [[ "$expected_exit_codes" =~ $exit_code ]]; then
        ((PASSED_CHECKS++))
        [[ "$VERBOSE" == "true" ]] && log::success "  ✓ Command '$command' returned expected code: $exit_code"
    else
        ((FAILED_CHECKS++))
        VALIDATION_ERRORS["$resource_name"]="${VALIDATION_ERRORS[$resource_name]:-}Command '$command' returned $exit_code (expected: $expected_exit_codes); "
        [[ "$VERBOSE" == "true" ]] && log::error "  ✗ Command '$command' returned $exit_code (expected: $expected_exit_codes)"
    fi
}

#######################################
# Test command output format
#######################################
test_command_output_format() {
    local resource_dir="$1"
    local resource_name="$2"
    
    # Skip for now - would test JSON/text output formats
    [[ "$VERBOSE" == "true" ]] && log::info "  ⏭ Output format testing (not implemented)"
}

#######################################
# Test flag handling
#######################################
test_flag_handling() {
    local resource_dir="$1"
    local resource_name="$2"
    
    # Skip for now - would test various flag combinations
    [[ "$VERBOSE" == "true" ]] && log::info "  ⏭ Flag handling testing (not implemented)"
}

#######################################
# Layer 3: Integration validation
#######################################
validate_layer3_integration() {
    local resource_dir="$1"
    local resource_name="$(basename "$resource_dir")"
    
    log::info "Layer 3: Integration validation for $resource_name"
    log::warning "  Layer 3 validation not yet implemented"
    
    # TODO: Implement full lifecycle testing
    # - Install → Start → Stop → Uninstall
    # - Content management flow
    # - Performance testing
    # - Cross-resource integration
}

#######################################
# Generate text report
#######################################
generate_text_report() {
    local total_resources="${#VALIDATION_RESULTS[@]}"
    local pass_rate=0
    [[ $TOTAL_CHECKS -gt 0 ]] && pass_rate=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
    
    echo "╔══════════════════════════════════════════════════════════════════════╗"
    echo "║              Universal Contract Validation Report                    ║"
    echo "╚══════════════════════════════════════════════════════════════════════╝"
    echo
    echo "Contract Version: $CONTRACT_VERSION"
    echo "Validation Layer: $VALIDATION_LAYER"
    echo "Date: $(date '+%Y-%m-%d %H:%M:%S')"
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "SUMMARY"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Resources Validated: $total_resources"
    echo "Total Checks: $TOTAL_CHECKS"
    echo "  ✅ Passed: $PASSED_CHECKS"
    echo "  ❌ Failed: $FAILED_CHECKS"
    echo "  ⚠️  Warnings: $WARNING_COUNT"
    echo "Pass Rate: ${pass_rate}%"
    echo
    
    if [[ ${#VALIDATION_ERRORS[@]} -gt 0 ]] || [[ ${#VALIDATION_WARNINGS[@]} -gt 0 ]]; then
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "DETAILS"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        
        for resource_name in "${!VALIDATION_RESULTS[@]}"; do
            local errors="${VALIDATION_ERRORS[$resource_name]:-}"
            local warnings="${VALIDATION_WARNINGS[$resource_name]:-}"
            local fixes="${VALIDATION_FIXES[$resource_name]:-}"
            
            if [[ -n "$errors" ]] || [[ -n "$warnings" ]]; then
                echo
                echo "Resource: $resource_name"
                
                if [[ -n "$errors" ]]; then
                    echo "  Errors:"
                    IFS=';' read -ra error_array <<< "$errors"
                    for error in "${error_array[@]}"; do
                        [[ -n "${error// }" ]] && echo "    ❌ ${error// /}"
                    done
                fi
                
                if [[ -n "$warnings" ]]; then
                    echo "  Warnings:"
                    IFS=';' read -ra warning_array <<< "$warnings"
                    for warning in "${warning_array[@]}"; do
                        [[ -n "${warning// }" ]] && echo "    ⚠️  ${warning// /}"
                    done
                fi
                
                if [[ -n "$fixes" ]] && [[ "$FIX_ISSUES" == "true" ]]; then
                    echo "  Suggested Fixes:"
                    IFS=';' read -ra fix_array <<< "$fixes"
                    for fix in "${fix_array[@]}"; do
                        [[ -n "${fix// }" ]] && echo "    → ${fix// /}"
                    done
                fi
            fi
        done
    fi
    
    echo
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $FAILED_CHECKS -eq 0 ]] && [[ $WARNING_COUNT -eq 0 ]]; then
        echo "✅ All validations passed!"
    elif [[ $FAILED_CHECKS -eq 0 ]]; then
        echo "✅ Validation passed with warnings"
    else
        echo "❌ Validation failed"
        echo
        echo "Next steps:"
        echo "  1. Fix the errors listed above"
    fi
}

#######################################
# Generate JSON report
#######################################
generate_json_report() {
    # Implementation would generate JSON format
    log::warning "JSON report format not yet implemented"
    generate_text_report
}

#######################################
# Main execution
#######################################
main() {
    parse_args "$@"
    
    log::info "Starting universal contract validation..."
    log::info "Contract: $CONTRACT_VERSION, Layer: $VALIDATION_LAYER"
    
    # Determine which resources to validate
    local resources_to_validate=()
    
    if [[ -n "$SPECIFIC_RESOURCE" ]]; then
        if [[ -d "$RESOURCES_DIR/$SPECIFIC_RESOURCE" ]]; then
            resources_to_validate+=("$RESOURCES_DIR/$SPECIFIC_RESOURCE")
        else
            log::error "Resource not found: $SPECIFIC_RESOURCE"
            exit 2
        fi
    else
        # Validate all resources
        while IFS= read -r -d '' resource_dir; do
            resources_to_validate+=("$resource_dir")
        done < <(find "$RESOURCES_DIR" -maxdepth 1 -mindepth 1 -type d -print0 2>/dev/null | sort -z)
    fi
    
    # Run validation for each resource
    for resource_dir in "${resources_to_validate[@]}"; do
        local resource_name="$(basename "$resource_dir")"
        
        [[ "$VERBOSE" == "true" ]] && echo
        log::info "Validating: $resource_name"
        
        VALIDATION_RESULTS[$resource_name]="started"
        
        # Run appropriate validation layers
        case $VALIDATION_LAYER in
            1)
                validate_layer1_syntax "$resource_dir"
                ;;
            2)
                validate_layer1_syntax "$resource_dir"
                validate_layer2_behavioral "$resource_dir"
                ;;
            3)
                validate_layer1_syntax "$resource_dir"
                validate_layer2_behavioral "$resource_dir"
                validate_layer3_integration "$resource_dir"
                ;;
            *)
                log::error "Invalid validation layer: $VALIDATION_LAYER"
                exit 2
                ;;
        esac
        
        VALIDATION_RESULTS[$resource_name]="completed"
    done
    
    [[ "$VERBOSE" == "true" ]] && echo
    
    # Generate report
    local report=""
    case "$OUTPUT_FORMAT" in
        json|junit)
            report="$(generate_json_report)"
            ;;
        *)
            report="$(generate_text_report)"
            ;;
    esac
    
    # Output report
    if [[ -n "$REPORT_FILE" ]]; then
        echo "$report" > "$REPORT_FILE"
        log::success "Report saved to: $REPORT_FILE"
    else
        echo "$report"
    fi
    
    # Exit based on validation results
    [[ $FAILED_CHECKS -eq 0 ]] && exit 0 || exit 1
}

# Run main
main "$@"
