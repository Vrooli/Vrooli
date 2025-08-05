#!/usr/bin/env bash
# Contract Parser - YAML Contract Reading and Merging
# Part of Layer 1 Syntax Validation System

set -euo pipefail

# Global variables for contract parsing
declare -g VROOLI_CONTRACTS_DIR=""
declare -g VROOLI_CONTRACT_CACHE=""

# Initialize associative array safely
if ! declare -p VROOLI_PARSED_CONTRACTS &>/dev/null; then
    declare -gA VROOLI_PARSED_CONTRACTS=()
fi

#######################################
# Initialize contract parser
# Arguments:
#   $1 - contracts directory (optional, auto-detect if not provided)
# Returns:
#   0 on success, 1 on error
#######################################
contract_parser_init() {
    local contracts_dir="${1:-}"
    
    # Auto-detect contracts directory if not provided
    if [[ -z "$contracts_dir" ]]; then
        # Look for project root by searching for scripts/resources/contracts
        local current_dir="$(pwd)"
        while [[ "$current_dir" != "/" ]] && [[ "$current_dir" != "$HOME" ]]; do
            if [[ -d "$current_dir/scripts/resources/contracts" ]]; then
                contracts_dir="$current_dir/scripts/resources/contracts"
                break
            fi
            current_dir="$(dirname "$current_dir")"
        done
        
        # Fallback to relative path from script location
        if [[ -z "$contracts_dir" ]]; then
            local script_dir
            script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)
            contracts_dir="$script_dir/scripts/resources/contracts"
        fi
    fi
    
    if [[ ! -d "$contracts_dir" ]]; then
        echo "Contract directory not found: $contracts_dir" >&2
        return 1
    fi
    
    VROOLI_CONTRACTS_DIR="$contracts_dir"
    VROOLI_CONTRACT_CACHE="/tmp/vrooli_contract_cache_$$"
    mkdir -p "$VROOLI_CONTRACT_CACHE"
    
    # Ensure associative array is properly initialized
    if ! declare -p VROOLI_PARSED_CONTRACTS &>/dev/null; then
        declare -gA VROOLI_PARSED_CONTRACTS=()
    fi
    
    echo "Contract parser initialized: $contracts_dir"
    return 0
}

#######################################
# Parse YAML value from file (simple YAML parser for bash)
# Arguments:
#   $1 - YAML file path
#   $2 - key path (e.g., "version" or "required_actions.install.description")
# Returns:
#   0 on success, 1 on error
# Outputs:
#   The parsed value
#######################################
parse_yaml_value() {
    local yaml_file="$1"
    local key_path="$2"
    
    if [[ ! -f "$yaml_file" ]]; then
        echo "YAML file not found: $yaml_file" >&2
        return 1
    fi
    
    # Handle simple key paths first
    if [[ "$key_path" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]]; then
        # Simple key like "version" or "category"
        local result
        result=$(grep "^${key_path}:" "$yaml_file" | sed 's/^[^:]*: *"*\([^"]*\)"*/\1/' | head -1)
        if [[ -n "$result" ]]; then
            echo "$result"
            return 0
        else
            echo "Key not found: $key_path" >&2
            return 1
        fi
    fi
    
    # Handle nested paths like "required_actions.install.description"
    IFS='.' read -ra KEYS <<< "$key_path"
    local current_level=0
    local target_level=${#KEYS[@]}
    local in_target_section=false
    local result=""
    
    while IFS= read -r line; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ "$line" =~ ^[[:space:]]*$ ]] && continue
        
        # Count indentation level
        local indent_level=0
        if [[ "$line" =~ ^([[:space:]]*) ]]; then
            indent_level=$((${#BASH_REMATCH[1]} / 2))
        fi
        
        # Extract key and value
        local key=""
        local value=""
        if [[ "$line" =~ ^[[:space:]]*([^:]+):[[:space:]]*(.*)$ ]]; then
            key="${BASH_REMATCH[1]// /}"
            value="${BASH_REMATCH[2]}"
        fi
        
        # Check if we're entering the target section
        if [[ $indent_level -eq $current_level && "$key" == "${KEYS[$current_level]}" ]]; then
            if [[ $((current_level + 1)) -eq $target_level ]]; then
                # We found the target key
                result="$value"
                result="${result#\"}"  # Remove leading quote
                result="${result%\"}"  # Remove trailing quote
                echo "$result"
                return 0
            else
                # Move to next level
                current_level=$((current_level + 1))
                in_target_section=true
            fi
        elif [[ $in_target_section && $indent_level -le $((current_level - 1)) ]]; then
            # We've left the target section
            break
        fi
    done < "$yaml_file"
    
    return 1
}

#######################################
# Get all keys under a YAML section
# Arguments:
#   $1 - YAML file path
#   $2 - section path (e.g., "required_actions")
# Returns:
#   0 on success, 1 on error
# Outputs:
#   List of keys, one per line
#######################################
get_yaml_section_keys() {
    local yaml_file="$1"
    local section_path="$2"
    
    if [[ ! -f "$yaml_file" ]]; then
        echo "YAML file not found: $yaml_file" >&2
        return 1
    fi
    
    local in_section=false
    local section_indent_level=-1
    local found_section=false
    local found_keys=false
    
    while IFS= read -r line; do
        # Skip comments and empty lines
        [[ "$line" =~ ^[[:space:]]*# ]] && continue
        [[ "$line" =~ ^[[:space:]]*$ ]] && continue
        
        # Count indentation level
        local indent_level=0
        if [[ "$line" =~ ^([[:space:]]*) ]]; then
            indent_level=$((${#BASH_REMATCH[1]} / 2))
        fi
        
        # Extract key
        local key=""
        if [[ "$line" =~ ^[[:space:]]*([^:]+): ]]; then
            key="${BASH_REMATCH[1]// /}"
        fi
        
        if [[ "$in_section" == "false" ]]; then
            # Looking for the section start
            if [[ "$key" == "$section_path" ]]; then
                in_section=true
                found_section=true
                section_indent_level=$indent_level
            fi
        else
            # Inside the section
            if [[ $indent_level -le $section_indent_level ]]; then
                # We've left the section
                break
            elif [[ $indent_level -eq $((section_indent_level + 1)) && -n "$key" ]]; then
                # This is a direct child key
                echo "$key"
                found_keys=true
            fi
        fi
    done < "$yaml_file"
    
    # Return failure if section not found or no keys found
    if [[ "$found_section" == "false" ]]; then
        echo "Section not found: $section_path" >&2
        return 1
    elif [[ "$found_keys" == "false" ]]; then
        echo "Section exists but has no keys: $section_path" >&2
        return 1
    fi
    
    return 0
}

#######################################
# Load contract file and handle inheritance
# Arguments:
#   $1 - contract file name (e.g., "core.yaml" or "ai.yaml")
# Returns:
#   0 on success, 1 on error
# Outputs:
#   Path to merged contract file
#######################################
load_contract() {
    local contract_name="$1"
    local loading_stack="${2:-}"  # Internal parameter for circular detection
    
    local contract_path="$VROOLI_CONTRACTS_DIR/v1.0/$contract_name"
    
    if [[ ! -f "$contract_path" ]]; then
        echo "Contract not found: $contract_path" >&2
        return 1
    fi
    
    # Check for circular inheritance
    if [[ "$loading_stack" == *"|$contract_name|"* ]]; then
        echo "Circular inheritance detected: $loading_stack -> $contract_name" >&2
        return 1
    fi
    
    # Check cache first
    local cache_key="${contract_name/.yaml/}"
    if [[ -v "VROOLI_PARSED_CONTRACTS[$cache_key]" ]]; then
        echo "${VROOLI_PARSED_CONTRACTS[$cache_key]}"
        return 0
    fi
    
    # Add current contract to loading stack
    local current_stack="${loading_stack}|${contract_name}|"
    
    # Check if this contract extends another
    local extends_contract=""
    extends_contract=$(parse_yaml_value "$contract_path" "extends" 2>/dev/null || echo "")
    
    local merged_contract_path="$VROOLI_CONTRACT_CACHE/${cache_key}_merged.yaml"
    
    if [[ -n "$extends_contract" ]]; then
        # Load parent contract first with inheritance stack
        local parent_contract_path
        parent_contract_path=$(load_contract "$extends_contract" "$current_stack")
        
        if [[ $? -ne 0 ]]; then
            echo "Failed to load parent contract: $extends_contract" >&2
            return 1
        fi
        
        # Merge parent and child contracts
        merge_contracts "$parent_contract_path" "$contract_path" "$merged_contract_path"
    else
        # No inheritance, just copy the contract
        cp "$contract_path" "$merged_contract_path"
    fi
    
    # Cache the result
    VROOLI_PARSED_CONTRACTS["$cache_key"]="$merged_contract_path"
    
    echo "$merged_contract_path"
    return 0
}

#######################################
# Merge two contract files (parent and child)
# Arguments:
#   $1 - parent contract file path
#   $2 - child contract file path
#   $3 - output merged contract file path
# Returns:
#   0 on success, 1 on error
#######################################
merge_contracts() {
    local parent_contract="$1"
    local child_contract="$2"
    local output_contract="$3"
    
    # Check if parent contract exists
    if [[ ! -f "$parent_contract" ]]; then
        echo "Parent contract not found: $parent_contract" >&2
        return 1
    fi
    
    # Check if child contract exists
    if [[ ! -f "$child_contract" ]]; then
        echo "Child contract not found: $child_contract" >&2
        return 1
    fi
    
    # Start with parent contract as base
    cp "$parent_contract" "$output_contract"
    
    # Add a merge marker
    echo "" >> "$output_contract"
    echo "# === Merged from child contract ===" >> "$output_contract"
    
    # Append child-specific sections
    local in_additional_section=false
    local in_category_section=false
    
    while IFS= read -r line; do
        # Skip metadata from child
        if [[ "$line" =~ ^(version|contract_type|extends|description): ]]; then
            continue
        fi
        
        # Add additional_actions section
        if [[ "$line" =~ ^additional_actions: ]]; then
            in_additional_section=true
            echo "" >> "$output_contract"
            echo "$line" >> "$output_contract"
            continue
        fi
        
        # Add category-specific sections (sections with category prefix)
        if [[ "$line" =~ ^[a-z]+_[a-z]+: ]]; then
            in_category_section=true
            echo "" >> "$output_contract"
            echo "$line" >> "$output_contract"
            continue
        fi
        
        # Add lines that are part of additional or category sections
        if [[ "$in_additional_section" == "true" || "$in_category_section" == "true" ]]; then
            # Check if we're still in the section (indented or empty line)
            if [[ "$line" =~ ^[[:space:]] || "$line" =~ ^[[:space:]]*$ ]]; then
                echo "$line" >> "$output_contract"
            else
                # We've left the section
                in_additional_section=false
                in_category_section=false
                # Check if this line starts a new section
                if [[ "$line" =~ ^[a-z]+_[a-z]+: ]]; then
                    in_category_section=true
                    echo "" >> "$output_contract"
                    echo "$line" >> "$output_contract"
                fi
            fi
        fi
    done < "$child_contract"
    
    return 0
}

#######################################
# Get required actions for a resource category
# Arguments:
#   $1 - resource category (e.g., "ai", "automation", "storage")
# Returns:
#   0 on success, 1 on error
# Outputs:
#   List of required actions, one per line
#######################################
get_required_actions() {
    local category="$1"
    local contract_file="${category}.yaml"
    
    # Load the merged contract
    local merged_contract
    merged_contract=$(load_contract "$contract_file")
    
    if [[ $? -ne 0 ]]; then
        # Fallback to core contract if category contract not found
        merged_contract=$(load_contract "core.yaml")
        if [[ $? -ne 0 ]]; then
            # Both contracts failed to load
            echo "Failed to load contracts" >&2
            return 1
        fi
    fi
    
    # Get required actions from core contract only (additional actions are optional)
    get_yaml_section_keys "$merged_contract" "required_actions"
    
    return $?
}

#######################################
# Get help patterns for a resource category
# Arguments:
#   $1 - resource category (e.g., "ai", "automation", "storage")
# Returns:
#   0 on success, 1 on error
# Outputs:
#   List of help patterns, one per line
#######################################
get_help_patterns() {
    local category="$1"
    local contract_file="${category}.yaml"
    
    # Load the merged contract
    local merged_contract
    merged_contract=$(load_contract "$contract_file")
    
    if [[ $? -ne 0 ]]; then
        # Fallback to core contract
        merged_contract=$(load_contract "core.yaml")
    fi
    
    # Extract help patterns from YAML array
    awk '/^help_patterns:/ {p=1; next} /^[^ ]/ {p=0} p && /^  - / {gsub(/^  - "|"$/, ""); print}' "$merged_contract"
    
    return 0
}

#######################################
# Get error handling patterns for a resource category
# Arguments:
#   $1 - resource category (e.g., "ai", "automation", "storage")
# Returns:
#   0 on success, 1 on error
# Outputs:
#   List of error handling requirements, one per line
#######################################
get_error_handling_patterns() {
    local category="$1"
    local contract_file="${category}.yaml"
    
    # Load the merged contract
    local merged_contract
    merged_contract=$(load_contract "$contract_file")
    
    if [[ $? -ne 0 ]]; then
        # Fallback to core contract
        merged_contract=$(load_contract "core.yaml")
    fi
    
    # Extract error handling patterns from YAML array
    awk '/^error_handling:/ {p=1; next} /^[^ ]/ {p=0} p && /^  - / {gsub(/^  - "|"$/, ""); print}' "$merged_contract"
    
    return 0
}

#######################################
# Get required files for a resource category
# Arguments:
#   $1 - resource category (e.g., "ai", "automation", "storage")
# Returns:
#   0 on success, 1 on error
# Outputs:
#   List of required files, one per line
#######################################
get_required_files() {
    local category="$1"
    local contract_file="${category}.yaml"
    
    # Load the merged contract
    local merged_contract
    merged_contract=$(load_contract "$contract_file")
    
    if [[ $? -ne 0 ]]; then
        # Fallback to core contract
        merged_contract=$(load_contract "core.yaml")
    fi
    
    # Extract required files from YAML array
    awk '/^required_files:/ {p=1; next} /^[^ ]/ {p=0} p && /^  - / {gsub(/^  - "|"$/, ""); print}' "$merged_contract"
    
    return 0
}

#######################################
# Cleanup contract parser resources
# Arguments: None
# Returns: 0
#######################################
contract_parser_cleanup() {
    if [[ -d "$VROOLI_CONTRACT_CACHE" ]]; then
        rm -rf "$VROOLI_CONTRACT_CACHE"
    fi
    
    # Clear associative array safely
    if declare -p VROOLI_PARSED_CONTRACTS &>/dev/null; then
        unset VROOLI_PARSED_CONTRACTS
    fi
    declare -gA VROOLI_PARSED_CONTRACTS=()
    
    echo "Contract parser cleanup completed"
    return 0
}

#######################################
# Validate contract syntax
# Arguments:
#   $1 - contract file path
# Returns:
#   0 if valid, 1 if invalid
#######################################
validate_contract_syntax() {
    local contract_path="$1"
    
    if [[ ! -f "$contract_path" ]]; then
        echo "Contract file not found: $contract_path" >&2
        return 1
    fi
    
    # Basic YAML syntax validation
    local required_fields=("version" "contract_type" "description")
    
    for field in "${required_fields[@]}"; do
        if ! grep -q "^${field}:" "$contract_path"; then
            echo "Missing required field: $field" >&2
            return 1
        fi
    done
    
    # Check for required_actions or additional_actions section
    if ! grep -q "^required_actions:" "$contract_path" && ! grep -q "^additional_actions:" "$contract_path"; then
        echo "Missing required_actions or additional_actions section" >&2
        return 1
    fi
    
    echo "Contract syntax validation passed"
    return 0
}

# Export functions for use in other scripts
export -f contract_parser_init
export -f parse_yaml_value
export -f get_yaml_section_keys
export -f load_contract
export -f merge_contracts
export -f get_required_actions
export -f get_help_patterns
export -f get_error_handling_patterns
export -f get_required_files
export -f contract_parser_cleanup
export -f validate_contract_syntax