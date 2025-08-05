#!/usr/bin/env bash
# Script Analyzer - Static Analysis of manage.sh Scripts
# Part of Layer 1 Syntax Validation System

set -euo pipefail

#######################################
# Extract all actions from a manage.sh script
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 on success, 1 on error
# Outputs:
#   List of actions, one per line
#######################################
extract_script_actions() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        echo "Script not found: $script_path" >&2
        return 1
    fi
    
    # Look for case statements that define actions
    # Pattern matches: "action") or action) or "action1"|"action2")
    {
        # First, find quoted patterns: "action")
        grep -E '^\s*"[a-zA-Z][a-zA-Z0-9_-]*"(\|"[a-zA-Z][a-zA-Z0-9_-]*")*\)' "$script_path" | \
            sed -E 's/^\s*"([^"]+)".*\)$/\1/' || true
        
        # Then, find unquoted patterns: action)
        grep -E '^\s*[a-zA-Z][a-zA-Z0-9_-]*\)' "$script_path" | \
            sed -E 's/^\s*([a-zA-Z0-9_-]+)\)$/\1/' || true
    } | sort -u
    
    return 0
}

#######################################
# Check if script has proper error handling patterns
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 if has good error handling, 1 if missing patterns
# Outputs:
#   List of found/missing patterns
#######################################
check_error_handling_patterns() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        echo "MISSING: Script not found: $script_path"
        return 1
    fi
    
    local found_patterns=()
    local missing_patterns=()
    
    # Check for strict mode
    if grep -q "set -euo pipefail" "$script_path"; then
        found_patterns+=("strict_mode")
    else
        missing_patterns+=("strict_mode")
    fi
    
    # Check for cleanup trap
    if grep -q "trap.*EXIT" "$script_path"; then
        found_patterns+=("cleanup_trap")
    else
        missing_patterns+=("cleanup_trap")
    fi
    
    # Check for error logging
    if grep -qE "(log_error|echo.*ERROR|echo.*\\[ERROR\\])" "$script_path"; then
        found_patterns+=("error_logging")
    else
        missing_patterns+=("error_logging")
    fi
    
    # Check for proper exit codes
    if grep -qE "(exit [1-9]|return [1-9])" "$script_path"; then
        found_patterns+=("exit_codes")
    else
        missing_patterns+=("exit_codes")
    fi
    
    # Output results
    for pattern in "${found_patterns[@]}"; do
        echo "FOUND: $pattern"
    done
    
    for pattern in "${missing_patterns[@]}"; do
        echo "MISSING: $pattern"
    done
    
    # Return success if we found at least half the patterns
    local required_count=2
    if [[ ${#found_patterns[@]} -ge $required_count ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Validate help patterns in script
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 if help patterns are implemented, 1 if missing
# Outputs:
#   List of found/missing help patterns
#######################################
validate_help_patterns() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        echo "MISSING: Script not found: $script_path"
        return 1
    fi
    
    local help_patterns=("--help" "-h" "--version")
    local found_patterns=()
    local missing_patterns=()
    
    for pattern in "${help_patterns[@]}"; do
        if grep -qF "$pattern" "$script_path"; then
            found_patterns+=("$pattern")
        else
            missing_patterns+=("$pattern")
        fi
    done
    
    # Output results
    for pattern in "${found_patterns[@]}"; do
        echo "FOUND: $pattern"
    done
    
    for pattern in "${missing_patterns[@]}"; do
        echo "MISSING: $pattern"
    done
    
    # Must have at least --help
    if [[ " ${found_patterns[*]} " =~ " --help " ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Check required files structure around the script
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 if structure is correct, 1 if missing files
# Outputs:
#   List of found/missing files
#######################################
check_required_files() {
    local script_path="$1"
    local script_dir
    script_dir=$(dirname "$script_path")
    
    if [[ ! -f "$script_path" ]]; then
        echo "MISSING: Script not found: $script_path"
        return 1
    fi
    
    local required_files=("config/defaults.sh" "config/messages.sh" "lib/common.sh")
    local found_files=()
    local missing_files=()
    
    for file in "${required_files[@]}"; do
        if [[ -f "$script_dir/$file" ]]; then
            found_files+=("$file")
        else
            missing_files+=("$file")
        fi
    done
    
    # Check for required directories
    local required_dirs=("config" "lib")
    for dir in "${required_dirs[@]}"; do
        if [[ -d "$script_dir/$dir" ]]; then
            found_files+=("$dir/")
        else
            missing_files+=("$dir/")
        fi
    done
    
    # Output results
    for file in "${found_files[@]}"; do
        echo "FOUND: $file"
    done
    
    for file in "${missing_files[@]}"; do
        echo "MISSING: $file"
    done
    
    # Must have config and lib directories
    if [[ -d "$script_dir/config" && -d "$script_dir/lib" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Analyze argument parsing patterns
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 if patterns are consistent, 1 if issues found
# Outputs:
#   Analysis of argument patterns
#######################################
analyze_argument_patterns() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        echo "MISSING: Script not found: $script_path"
        return 1
    fi
    
    local issues=()
    local good_patterns=()
    
    # Check for --action pattern
    if grep -q "\\--action" "$script_path"; then
        good_patterns+=("action_flag")
    else
        issues+=("missing_action_flag")
    fi
    
    # Check for --yes pattern
    if grep -q "\\--yes" "$script_path"; then
        good_patterns+=("yes_flag")
    else
        issues+=("missing_yes_flag")
    fi
    
    # Check for case statement handling arguments
    if grep -qE "case.*\\$[a-zA-Z_].*in" "$script_path"; then
        good_patterns+=("case_statement")
    else
        issues+=("missing_case_statement")
    fi
    
    # Check for argument validation
    if grep -qE "(\\[.*-z.*\\]|\\[.*-n.*\\])" "$script_path"; then
        good_patterns+=("argument_validation")
    else
        issues+=("missing_argument_validation")
    fi
    
    # Output results
    for pattern in "${good_patterns[@]}"; do
        echo "GOOD: $pattern"
    done
    
    for issue in "${issues[@]}"; do
        echo "ISSUE: $issue"
    done
    
    # Must have basic argument patterns
    if [[ " ${good_patterns[*]} " =~ " action_flag " && " ${good_patterns[*]} " =~ " case_statement " ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Check configuration loading patterns
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 if configuration loading is proper, 1 if issues
# Outputs:
#   Configuration loading analysis
#######################################
check_configuration_loading() {
    local script_path="$1"
    local script_dir
    script_dir=$(dirname "$script_path")
    
    if [[ ! -f "$script_path" ]]; then
        echo "MISSING: Script not found: $script_path"
        return 1
    fi
    
    local found_patterns=()
    local missing_patterns=()
    
    # Check for sourcing config files
    if grep -q "source.*config/defaults\\.sh" "$script_path" || grep -q "\\. .*config/defaults\\.sh" "$script_path"; then
        found_patterns+=("defaults_sourced")
    else
        missing_patterns+=("defaults_sourced")
    fi
    
    if grep -q "source.*config/messages\\.sh" "$script_path" || grep -q "\\. .*config/messages\\.sh" "$script_path"; then
        found_patterns+=("messages_sourced")
    else
        missing_patterns+=("messages_sourced")
    fi
    
    if grep -q "source.*lib/common\\.sh" "$script_path" || grep -q "\\. .*lib/common\\.sh" "$script_path"; then
        found_patterns+=("common_sourced")
    else
        missing_patterns+=("common_sourced")
    fi
    
    # Check if config files exist
    if [[ -f "$script_dir/config/defaults.sh" ]]; then
        found_patterns+=("defaults_exists")
    else
        missing_patterns+=("defaults_exists")
    fi
    
    # Output results
    for pattern in "${found_patterns[@]}"; do
        echo "FOUND: $pattern"
    done
    
    for pattern in "${missing_patterns[@]}"; do
        echo "MISSING: $pattern"
    done
    
    # Must source at least common.sh
    if [[ " ${found_patterns[*]} " =~ " common_sourced " ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Extract function definitions from script
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 on success, 1 on error
# Outputs:
#   List of function names, one per line
#######################################
extract_function_definitions() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        echo "Script not found: $script_path" >&2
        return 1
    fi
    
    # Find function definitions
    grep -E "^[a-zA-Z_][a-zA-Z0-9_]*\\(\\)" "$script_path" | \
        sed 's/().*$//' | \
        sort -u
    
    return 0
}

#######################################
# Check script shebang and permissions
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 if correct, 1 if issues
# Outputs:
#   Analysis of shebang and permissions
#######################################
check_script_basics() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        echo "MISSING: Script not found: $script_path"
        return 1
    fi
    
    local issues=()
    local good_patterns=()
    
    # Check shebang
    local first_line
    first_line=$(head -1 "$script_path")
    if [[ "$first_line" =~ ^#!/.*bash ]]; then
        good_patterns+=("bash_shebang")
    else
        issues+=("missing_bash_shebang")
    fi
    
    # Check if executable
    if [[ -x "$script_path" ]]; then
        good_patterns+=("executable")
    else
        issues+=("not_executable")
    fi
    
    # Check file size (should not be empty)
    if [[ -s "$script_path" ]]; then
        good_patterns+=("not_empty")
    else
        issues+=("empty_file")
    fi
    
    # Output results
    for pattern in "${good_patterns[@]}"; do
        echo "GOOD: $pattern"
    done
    
    for issue in "${issues[@]}"; do
        echo "ISSUE: $issue"
    done
    
    # Must be executable with bash shebang
    if [[ " ${good_patterns[*]} " =~ " bash_shebang " && " ${good_patterns[*]} " =~ " executable " ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Comprehensive script analysis
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 if script passes all checks, 1 if issues found
# Outputs:
#   Complete analysis report
#######################################
analyze_script_comprehensive() {
    local script_path="$1"
    
    echo "=== Comprehensive Script Analysis ==="
    echo "Script: $script_path"
    echo
    
    local checks_passed=0
    local checks_failed=0
    
    # Array of analysis functions
    local analyses=(
        "check_script_basics"
        "extract_script_actions"
        "check_error_handling_patterns"
        "validate_help_patterns"
        "check_required_files"
        "analyze_argument_patterns"
        "check_configuration_loading"
    )
    
    for analysis in "${analyses[@]}"; do
        echo "--- $analysis ---"
        if $analysis "$script_path"; then
            echo "✅ PASSED"
            ((checks_passed++))
        else
            echo "❌ FAILED"
            ((checks_failed++))
        fi
        echo
    done
    
    echo "=== Analysis Summary ==="
    echo "Checks passed: $checks_passed"
    echo "Checks failed: $checks_failed"
    echo
    
    if [[ $checks_failed -eq 0 ]]; then
        echo "🎉 Script analysis PASSED"
        return 0
    else
        echo "⚠️  Script analysis FAILED"
        return 1
    fi
}

#######################################
# Get script complexity metrics
# Arguments:
#   $1 - path to manage.sh script
# Returns:
#   0 on success, 1 on error
# Outputs:
#   Script complexity metrics
#######################################
get_script_metrics() {
    local script_path="$1"
    
    if [[ ! -f "$script_path" ]]; then
        echo "Script not found: $script_path" >&2
        return 1
    fi
    
    echo "=== Script Metrics ==="
    echo "File: $script_path"
    echo "Lines of code: $(wc -l < "$script_path")"
    echo "Functions defined: $(extract_function_definitions "$script_path" | wc -l)"
    echo "Actions implemented: $(extract_script_actions "$script_path" | wc -l)"
    echo "File size: $(stat -f%z "$script_path" 2>/dev/null || stat -c%s "$script_path" 2>/dev/null || echo "unknown") bytes"
    echo
    
    return 0
}

# Export functions for use in other scripts
export -f extract_script_actions
export -f check_error_handling_patterns
export -f validate_help_patterns
export -f check_required_files
export -f analyze_argument_patterns
export -f check_configuration_loading
export -f extract_function_definitions
export -f check_script_basics
export -f analyze_script_comprehensive
export -f get_script_metrics