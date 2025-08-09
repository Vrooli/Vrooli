#!/usr/bin/env bash
# Resource Interface Validation Library
# Provides standardized validation of resource manage.sh interfaces and common functions

set -euo pipefail

_HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${_HERE}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

#######################################
# Category-specific function requirements
#######################################

# AI Resources - Expected functions and their signatures
declare -A AI_REQUIRED_FUNCTIONS=(
    ["status"]="No parameters - Check service status"
    ["install"]="[--force] - Install the AI service"
    ["start"]="[--wait] - Start the AI service"
    ["stop"]="[--force] - Stop the AI service"
    ["restart"]="[--wait] - Restart the AI service"
    ["test"]="[--verbose] - Test AI service functionality"
    ["logs"]="[--tail N] - Show service logs"
)

declare -A AI_OPTIONAL_FUNCTIONS=(
    ["models"]="[--list|--pull|--remove] - Manage AI models"
    ["generate"]="--text TEXT [--model MODEL] - Generate content"
    ["health"]="[--deep] - Deep health check with model"
    ["config"]="[--show|--set KEY=VALUE] - Manage configuration"
)

# Automation Resources - Expected functions
declare -A AUTOMATION_REQUIRED_FUNCTIONS=(
    ["status"]="No parameters - Check automation service status"
    ["install"]="[--force] - Install automation platform"
    ["start"]="[--wait] - Start automation service"
    ["stop"]="[--force] - Stop automation service"
    ["restart"]="[--wait] - Restart automation service"
    ["test"]="[--verbose] - Test automation functionality"
    ["logs"]="[--tail N] - Show service logs"
)

declare -A AUTOMATION_OPTIONAL_FUNCTIONS=(
    ["workflows"]="[--list|--create|--execute] - Manage workflows"
    ["import"]="--file FILE - Import workflow/configuration"
    ["export"]="--output FILE - Export workflows"
    ["backup"]="--destination DIR - Backup automation data"
    ["restore"]="--source FILE - Restore from backup"
)

# Storage Resources - Expected functions
declare -A STORAGE_REQUIRED_FUNCTIONS=(
    ["status"]="No parameters - Check storage service status"
    ["install"]="[--force] - Install storage service"
    ["start"]="[--wait] - Start storage service"
    ["stop"]="[--force] - Stop storage service"
    ["restart"]="[--wait] - Restart storage service"
    ["test"]="[--verbose] - Test storage functionality"
    ["logs"]="[--tail N] - Show service logs"
)

declare -A STORAGE_OPTIONAL_FUNCTIONS=(
    ["backup"]="--destination DIR - Create data backup"
    ["restore"]="--source FILE - Restore from backup" 
    ["migrate"]="--from VERSION --to VERSION - Migrate data"
    ["cleanup"]="[--dry-run] - Clean up old data"
    ["connect"]="--test - Test database connection"
)

# Execution Resources - Expected functions
declare -A EXECUTION_REQUIRED_FUNCTIONS=(
    ["status"]="No parameters - Check execution service status"
    ["install"]="[--force] - Install execution environment"
    ["start"]="[--wait] - Start execution service"
    ["stop"]="[--force] - Stop execution service"
    ["restart"]="[--wait] - Restart execution service"
    ["test"]="[--verbose] - Test execution functionality"
    ["logs"]="[--tail N] - Show service logs"
)

declare -A EXECUTION_OPTIONAL_FUNCTIONS=(
    ["execute"]="--code CODE [--language LANG] - Execute code"
    ["languages"]="[--list] - List supported languages"
    ["limits"]="[--show|--set] - Manage execution limits"
)

# Agent Resources - Expected functions  
declare -A AGENT_REQUIRED_FUNCTIONS=(
    ["status"]="No parameters - Check agent service status"
    ["install"]="[--force] - Install agent service"
    ["start"]="[--wait] - Start agent service"
    ["stop"]="[--force] - Stop agent service"
    ["restart"]="[--wait] - Restart agent service"
    ["test"]="[--verbose] - Test agent functionality"
    ["logs"]="[--tail N] - Show service logs"
)

declare -A AGENT_OPTIONAL_FUNCTIONS=(
    ["screenshot"]="[--output FILE] - Take screenshot"
    ["automate"]="--task TASK - Execute automation task"
    ["interact"]="--element SELECTOR - Interact with UI element"
)

#######################################
# Resource category detection
#######################################

interface_validation::detect_resource_category() {
    local resource_path="$1"
    
    # Extract category from path (e.g., scripts/resources/ai/ollama -> ai)
    if [[ "$resource_path" =~ scripts/resources/([^/]+)/([^/]+) ]]; then
        echo "${BASH_REMATCH[1]}"
    else
        echo "unknown"
    fi
}

#######################################
# Validate manage.sh exists and is executable
#######################################

interface_validation::validate_manage_script() {
    local resource_path="$1"
    local manage_script="$resource_path/manage.sh"
    
    # Check if manage.sh exists
    if [[ ! -f "$manage_script" ]]; then
        echo "FAIL: manage.sh not found at $manage_script"
        return 1
    fi
    
    # Check if executable
    if [[ ! -x "$manage_script" ]]; then
        echo "FAIL: manage.sh not executable at $manage_script"
        return 1
    fi
    
    echo "PASS: manage.sh exists and is executable"
    return 0
}

#######################################
# Extract available actions from manage.sh
#######################################

interface_validation::extract_available_actions() {
    local resource_path="$1"
    local manage_script="$resource_path/manage.sh"
    
    # Look for case statements that define actions
    grep -E 'case.*action.*in|"[a-z]+"\)' "$manage_script" | \
        sed -n 's/.*"\([a-z][a-z-]*\)").*/\1/p' | \
        sort -u
}

#######################################
# Validate required functions exist
#######################################

interface_validation::validate_required_functions() {
    local resource_path="$1"
    local category="$2"
    
    local -n required_functions_ref
    case "$category" in
        "ai") required_functions_ref=AI_REQUIRED_FUNCTIONS ;;
        "automation") required_functions_ref=AUTOMATION_REQUIRED_FUNCTIONS ;;
        "storage") required_functions_ref=STORAGE_REQUIRED_FUNCTIONS ;;
        "execution") required_functions_ref=EXECUTION_REQUIRED_FUNCTIONS ;;
        "agents") required_functions_ref=AGENT_REQUIRED_FUNCTIONS ;;
        *) 
            echo "SKIP: Unknown category '$category'"
            return 2
            ;;
    esac
    
    local available_actions
    available_actions=$(interface_validation::extract_available_actions "$resource_path")
    
    local missing_functions=()
    local found_functions=()
    
    for required_function in "${!required_functions_ref[@]}"; do
        if echo "$available_actions" | grep -q "^${required_function}$"; then
            found_functions+=("$required_function")
        else
            missing_functions+=("$required_function")
        fi
    done
    
    # Report results
    echo "Found functions: ${found_functions[*]}"
    
    if [[ ${#missing_functions[@]} -gt 0 ]]; then
        echo "FAIL: Missing required functions: ${missing_functions[*]}"
        return 1
    else
        echo "PASS: All required functions present"
        return 0
    fi
}

#######################################
# Validate function help/usage patterns
#######################################

interface_validation::validate_function_help() {
    local resource_path="$1"
    local manage_script="$resource_path/manage.sh"
    
    # Check that --help is supported
    local help_output
    if help_output=$("$manage_script" --help 2>&1); then
        if [[ "$help_output" =~ Usage:|Actions:|OPTIONS: ]]; then
            echo "PASS: Help output is well-formatted"
            return 0
        else
            echo "FAIL: Help output exists but poorly formatted"
            return 1
        fi
    else
        echo "FAIL: --help flag not supported or returns error"
        return 1
    fi
}

#######################################
# Validate configuration file structure
#######################################

interface_validation::validate_config_structure() {
    local resource_path="$1"
    local config_dir="$resource_path/config"
    
    local missing_files=()
    local found_files=()
    
    # Check for required config files
    local required_config_files=("defaults.sh" "messages.sh")
    
    for config_file in "${required_config_files[@]}"; do
        if [[ -f "$config_dir/$config_file" ]]; then
            found_files+=("$config_file")
        else
            missing_files+=("$config_file")
        fi
    done
    
    echo "Found config files: ${found_files[*]}"
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        echo "FAIL: Missing config files: ${missing_files[*]}"
        return 1
    else
        echo "PASS: All required config files present"
        return 0
    fi
}

#######################################
# Validate error handling patterns
#######################################

interface_validation::validate_error_handling() {
    local resource_path="$1"
    local manage_script="$resource_path/manage.sh"
    
    # Check for common error handling patterns
    local error_patterns=(
        "set -euo pipefail"
        "trap.*EXIT"
        "log_error\|echo.*ERROR"
        "return [1-9]"
        "exit [1-9]"
    )
    
    local found_patterns=()
    local missing_patterns=()
    
    for pattern in "${error_patterns[@]}"; do
        if grep -q "$pattern" "$manage_script"; then
            found_patterns+=("$pattern")
        else
            missing_patterns+=("$pattern")
        fi
    done
    
    # At least half of error handling patterns should be present
    local required_count=$((${#error_patterns[@]} / 2))
    
    if [[ ${#found_patterns[@]} -ge $required_count ]]; then
        echo "PASS: Adequate error handling patterns found"
        return 0
    else
        echo "FAIL: Insufficient error handling patterns"
        echo "Found: ${found_patterns[*]}"
        echo "Missing: ${missing_patterns[*]}"
        return 1
    fi
}

#######################################
# Validate port registry integration
#######################################

interface_validation::validate_port_registry() {
    local resource_path="$1"
    local resource_name
    resource_name=$(basename "$resource_path")
    
    # Check if resource is registered in port registry
    local port_registry="$var_PORT_REGISTRY_FILE"
    
    if [[ ! -f "$port_registry" ]]; then
        echo "SKIP: Port registry not found"
        return 2
    fi
    
    if grep -q "\\[\"$resource_name\"\\]" "$port_registry"; then
        echo "PASS: Resource registered in port registry"
        return 0
    else
        echo "WARNING: Resource not found in port registry"
        return 1
    fi
}

#######################################
# Main interface validation function
#######################################

interface_validation::validate_resource_interface() {
    local resource_path="$1"
    local resource_name
    resource_name=$(basename "$resource_path")
    local category
    category=$(interface_validation::detect_resource_category "$resource_path")
    
    echo "Validating resource interface: $resource_name (category: $category)"
    echo "========================================"
    
    local tests_passed=0
    local tests_failed=0
    local tests_skipped=0
    
    # Array of validation functions to run
    local validations=(
        "interface_validation::validate_manage_script"
        "interface_validation::validate_required_functions $resource_path $category"
        "interface_validation::validate_function_help"
        "interface_validation::validate_config_structure"
        "interface_validation::validate_error_handling"
        "interface_validation::validate_port_registry"
    )
    
    for validation in "${validations[@]}"; do
        echo
        echo "Running: $validation"
        echo "----------------------------------------"
        
        local result
        if result=$($validation "$resource_path" 2>&1); then
            local exit_code=$?
            echo "$result"
            
            case $exit_code in
                0) ((tests_passed++)) ;;
                1) ((tests_failed++)) ;;
                2) ((tests_skipped++)) ;;
            esac
        else
            echo "FAIL: Validation function error"
            ((tests_failed++))
        fi
    done
    
    echo
    echo "========================================"
    echo "Interface Validation Summary:"
    echo "  Passed: $tests_passed"
    echo "  Failed: $tests_failed"  
    echo "  Skipped: $tests_skipped"
    echo "========================================"
    
    if [[ $tests_failed -eq 0 ]]; then
        echo "✅ Resource interface validation PASSED"
        return 0
    else
        echo "❌ Resource interface validation FAILED"
        return 1
    fi
}

# Export functions for use in BATS tests
export -f interface_validation::detect_resource_category
export -f interface_validation::validate_manage_script
export -f interface_validation::extract_available_actions
export -f interface_validation::validate_required_functions
export -f interface_validation::validate_function_help
export -f interface_validation::validate_config_structure
export -f interface_validation::validate_error_handling
export -f interface_validation::validate_port_registry
export -f interface_validation::validate_resource_interface