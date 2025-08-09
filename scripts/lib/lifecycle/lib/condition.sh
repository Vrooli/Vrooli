#!/usr/bin/env bash
# Lifecycle Engine - Condition Evaluation Module
# Handles evaluation of conditional expressions for steps and phases

set -euo pipefail

# Determine script directory
_HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${_HERE}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Guard against re-sourcing
[[ -n "${_CONDITION_MODULE_LOADED:-}" ]] && return 0
declare -gr _CONDITION_MODULE_LOADED=1

# Supported condition types (exported for potential external use)
readonly CONDITION_TYPES=(
    "equals"      # VAR == value
    "not_equals"  # VAR != value
    "exists"      # File/directory exists
    "changed"     # File changed (git)
    "command"     # Command succeeds
    "expression"  # Shell expression
    "always"      # Always true
    "never"       # Always false
)

# Export for external validation
export CONDITION_TYPES

#######################################
# Evaluate a condition expression
# Arguments:
#   $1 - Condition expression or JSON object
# Returns:
#   0 if condition is true, 1 if false
#######################################
condition::evaluate() {
    local condition="${1:-}"
    
    # Empty condition is always true
    [[ -z "$condition" ]] && return 0
    
    # Handle JSON condition object (must start with { or [)
    if [[ "$condition" =~ ^\{.*\}$ ]] || [[ "$condition" =~ ^\[.*\]$ ]]; then
        if echo "$condition" | jq empty 2>/dev/null; then
            condition::evaluate_json "$condition"
            return $?
        fi
    fi
    
    # Handle string conditions
    condition::evaluate_string "$condition"
}

#######################################
# Evaluate JSON condition object
# Arguments:
#   $1 - JSON condition object
# Returns:
#   0 if true, 1 if false
#######################################
condition::evaluate_json() {
    local json="$1"
    
    local type
    type=$(echo "$json" | jq -r '.type // "expression"')
    
    case "$type" in
        equals)
            condition::evaluate_equals "$json"
            ;;
        not_equals)
            condition::evaluate_not_equals "$json"
            ;;
        exists)
            condition::evaluate_exists "$json"
            ;;
        changed)
            condition::evaluate_changed "$json"
            ;;
        command)
            condition::evaluate_command "$json"
            ;;
        expression)
            local expr
            expr=$(echo "$json" | jq -r '.expression // ""')
            condition::evaluate_string "$expr"
            ;;
        and)
            condition::evaluate_and "$json"
            ;;
        or)
            condition::evaluate_or "$json"
            ;;
        not)
            condition::evaluate_not "$json"
            ;;
        *)
            echo "Warning: Unknown condition type: $type" >&2
            return 1
            ;;
    esac
}

#######################################
# Evaluate string condition expression
# Arguments:
#   $1 - String expression
# Returns:
#   0 if true, 1 if false
#######################################
condition::evaluate_string() {
    local expr="${1:-}"
    
    # Substitute environment variables
    local evaluated
    evaluated=$(eval "echo \"$expr\"" 2>/dev/null || echo "$expr")
    
    # Handle special string patterns
    case "$evaluated" in
        true|TRUE|yes|YES|1)
            return 0
            ;;
        false|FALSE|no|NO|0)
            return 1
            ;;
        *"=="*)
            condition::evaluate_string_equals "$evaluated"
            ;;
        *"!="*)
            condition::evaluate_string_not_equals "$evaluated"
            ;;
        *"changed:"*)
            local file="${evaluated#*changed:}"
            file="${file// /}"  # Trim spaces
            condition::check_file_changed "$file"
            ;;
        *"exists:"*)
            local path="${evaluated#*exists:}"
            path="${path// /}"  # Trim spaces
            [[ -e "$path" ]]
            ;;
        *)
            # Try to evaluate as shell expression
            condition::evaluate_shell_expression "$expr"
            ;;
    esac
}

#######################################
# Evaluate equality condition
# Arguments:
#   $1 - JSON object with 'left' and 'right' fields
# Returns:
#   0 if equal, 1 if not
#######################################
condition::evaluate_equals() {
    local json="$1"
    
    local left right
    left=$(echo "$json" | jq -r '.left // ""')
    right=$(echo "$json" | jq -r '.right // ""')
    
    # Expand variables
    left=$(eval "echo \"$left\"" 2>/dev/null || echo "$left")
    right=$(eval "echo \"$right\"" 2>/dev/null || echo "$right")
    
    [[ "$left" == "$right" ]]
}

#######################################
# Evaluate inequality condition
# Arguments:
#   $1 - JSON object with 'left' and 'right' fields
# Returns:
#   0 if not equal, 1 if equal
#######################################
condition::evaluate_not_equals() {
    local json="$1"
    
    local left right
    left=$(echo "$json" | jq -r '.left // ""')
    right=$(echo "$json" | jq -r '.right // ""')
    
    # Expand variables
    left=$(eval "echo \"$left\"" 2>/dev/null || echo "$left")
    right=$(eval "echo \"$right\"" 2>/dev/null || echo "$right")
    
    [[ "$left" != "$right" ]]
}

#######################################
# Evaluate file/directory existence
# Arguments:
#   $1 - JSON object with 'path' field
# Returns:
#   0 if exists, 1 if not
#######################################
condition::evaluate_exists() {
    local json="$1"
    
    local path
    path=$(echo "$json" | jq -r '.path // ""')
    
    # Expand path
    path=$(eval "echo \"$path\"" 2>/dev/null || echo "$path")
    
    [[ -e "$path" ]]
}

#######################################
# Evaluate if file changed (using git)
# Arguments:
#   $1 - JSON object with 'path' field
# Returns:
#   0 if changed, 1 if not
#######################################
condition::evaluate_changed() {
    local json="$1"
    
    local path
    path=$(echo "$json" | jq -r '.path // ""')
    
    condition::check_file_changed "$path"
}

#######################################
# Check if file has changed using git
# Arguments:
#   $1 - File path
# Returns:
#   0 if changed, 1 if not
#######################################
condition::check_file_changed() {
    local path="$1"
    
    # Check if we're in a git repo
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        # Not in git repo, assume changed
        return 0
    fi
    
    # Check if file has changes
    if git diff --quiet HEAD -- "$path" 2>/dev/null; then
        # No changes
        return 1
    else
        # Has changes
        return 0
    fi
}

#######################################
# Evaluate command condition
# Arguments:
#   $1 - JSON object with 'command' field
# Returns:
#   0 if command succeeds, 1 if fails
#######################################
condition::evaluate_command() {
    local json="$1"
    
    local cmd
    cmd=$(echo "$json" | jq -r '.command // ""')
    
    # Execute command and check exit code
    if eval "$cmd" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

#######################################
# Evaluate AND condition (all must be true)
# Arguments:
#   $1 - JSON object with 'conditions' array
# Returns:
#   0 if all true, 1 if any false
#######################################
condition::evaluate_and() {
    local json="$1"
    
    local conditions
    conditions=$(echo "$json" | jq -r '.conditions // []')
    
    local count
    count=$(echo "$conditions" | jq 'length')
    
    local i=0
    while [[ $i -lt $count ]]; do
        local cond
        cond=$(echo "$conditions" | jq ".[$i]")
        
        if ! condition::evaluate "$cond"; then
            return 1
        fi
        
        ((i++))
    done
    
    return 0
}

#######################################
# Evaluate OR condition (any must be true)
# Arguments:
#   $1 - JSON object with 'conditions' array
# Returns:
#   0 if any true, 1 if all false
#######################################
condition::evaluate_or() {
    local json="$1"
    
    local conditions
    conditions=$(echo "$json" | jq -r '.conditions // []')
    
    local count
    count=$(echo "$conditions" | jq 'length')
    
    local i=0
    while [[ $i -lt $count ]]; do
        local cond
        cond=$(echo "$conditions" | jq ".[$i]")
        
        if condition::evaluate "$cond"; then
            return 0
        fi
        
        ((i++))
    done
    
    return 1
}

#######################################
# Evaluate NOT condition (invert result)
# Arguments:
#   $1 - JSON object with 'condition' field
# Returns:
#   0 if condition is false, 1 if true
#######################################
condition::evaluate_not() {
    local json="$1"
    
    local cond
    cond=$(echo "$json" | jq -r '.condition // ""')
    
    if condition::evaluate "$cond"; then
        return 1
    else
        return 0
    fi
}

#######################################
# Evaluate string equality (VAR == value)
# Arguments:
#   $1 - String containing "=="
# Returns:
#   0 if equal, 1 if not
#######################################
condition::evaluate_string_equals() {
    local expr="$1"
    
    local left="${expr%% ==*}"
    local right="${expr##*== }"
    
    # Remove quotes
    left="${left//\"/}"
    left="${left//\'/}"
    right="${right//\"/}"
    right="${right//\'/}"
    
    # Trim whitespace
    left="${left// /}"
    right="${right// /}"
    
    [[ "$left" == "$right" ]]
}

#######################################
# Evaluate string inequality (VAR != value)
# Arguments:
#   $1 - String containing "!="
# Returns:
#   0 if not equal, 1 if equal
#######################################
condition::evaluate_string_not_equals() {
    local expr="$1"
    
    local left="${expr%% !=*}"
    local right="${expr##*!= }"
    
    # Remove quotes
    left="${left//\"/}"
    left="${left//\'/}"
    right="${right//\"/}"
    right="${right//\'/}"
    
    # Trim whitespace
    left="${left// /}"
    right="${right// /}"
    
    [[ "$left" != "$right" ]]
}

#######################################
# Evaluate shell expression using [[]]
# Arguments:
#   $1 - Shell expression
# Returns:
#   0 if true, 1 if false
#######################################
condition::evaluate_shell_expression() {
    local expr="$1"
    
    # Try to evaluate as shell test expression
    if eval "[[ $expr ]]" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

#######################################
# Check if step should be skipped
# Arguments:
#   $1 - Step name
# Returns:
#   0 if should skip, 1 if should run
#######################################
condition::should_skip_step() {
    local step_name="${1:-}"
    
    # Check --only option
    if [[ -n "${ONLY_STEP:-}" ]] && [[ "$step_name" != "$ONLY_STEP" ]]; then
        return 0
    fi
    
    # Check --skip option
    if [[ -n "${SKIP_STEPS_LIST:-}" ]]; then
        if echo ":${SKIP_STEPS_LIST}:" | grep -q ":${step_name}:"; then
            return 0
        fi
    fi
    
    # Check environment variable (SKIP_STEP_NAME=true)
    local skip_var="SKIP_${step_name^^}"
    skip_var="${skip_var//-/_}"
    
    if [[ "${!skip_var:-false}" == "true" ]]; then
        return 0
    fi
    
    return 1
}

# If sourced for testing, don't auto-execute
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script should be sourced, not executed directly" >&2
    exit 1
fi