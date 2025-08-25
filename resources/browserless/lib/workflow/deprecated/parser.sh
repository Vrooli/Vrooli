#!/usr/bin/env bash

#######################################
# Browserless Workflow Parser
# Parses and validates workflow definitions in YAML/JSON format
#######################################

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"
WORKFLOW_PARSER_DIR="${APP_ROOT}/resources/browserless/lib/workflow/deprecated"
WORKFLOW_LIB_DIR="${APP_ROOT}/resources/browserless/lib/workflow"

#######################################
# Parse workflow from YAML or JSON file
# Arguments:
#   $1 - Workflow file path
# Returns:
#   0 on success, 1 on failure
# Outputs:
#   Parsed workflow as JSON
#######################################
workflow::parse() {
    local file_path="${1:?Workflow file required}"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "Workflow file not found: $file_path"
        return 1
    fi
    
    local workflow_json
    local file_ext="${file_path##*.}"
    
    case "$file_ext" in
        yaml|yml)
            # Convert YAML to JSON
            if command -v yq >/dev/null 2>&1; then
                workflow_json=$(yq eval -o=json '.' "$file_path" 2>/dev/null)
            elif command -v python3 >/dev/null 2>&1; then
                # Fallback to Python if yq not available
                workflow_json=$(python3 -c "
import sys, json, yaml
with open('$file_path', 'r') as f:
    data = yaml.safe_load(f)
    print(json.dumps(data))
" 2>/dev/null)
            else
                log::error "YAML parsing requires 'yq' or Python with PyYAML"
                return 1
            fi
            ;;
        json)
            # Already JSON, just validate
            workflow_json=$(jq '.' "$file_path" 2>/dev/null)
            ;;
        *)
            log::error "Unsupported file format: $file_ext (use .yaml, .yml, or .json)"
            return 1
            ;;
    esac
    
    if [[ -z "$workflow_json" ]]; then
        log::error "Failed to parse workflow file"
        return 1
    fi
    
    # Validate and output
    if workflow::validate_structure "$workflow_json"; then
        echo "$workflow_json"
        return 0
    else
        return 1
    fi
}

#######################################
# Validate workflow structure
# Arguments:
#   $1 - Workflow JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
workflow::validate_structure() {
    local workflow_json="${1:?Workflow JSON required}"
    
    # Check required fields
    local workflow_name
    workflow_name=$(echo "$workflow_json" | jq -r '.workflow.name // empty')
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow missing required field: workflow.name"
        return 1
    fi
    
    # Check for steps
    local steps_count
    steps_count=$(echo "$workflow_json" | jq -r '.workflow.steps | length' 2>/dev/null)
    
    if [[ -z "$steps_count" ]] || [[ "$steps_count" == "0" ]]; then
        log::error "Workflow must have at least one step"
        return 1
    fi
    
    # Validate each step
    local step_index=0
    while [[ $step_index -lt $steps_count ]]; do
        local step
        step=$(echo "$workflow_json" | jq -r ".workflow.steps[$step_index]")
        
        local step_name
        step_name=$(echo "$step" | jq -r '.name // empty')
        
        local step_action
        step_action=$(echo "$step" | jq -r '.action // empty')
        
        if [[ -z "$step_name" ]]; then
            log::error "Step $((step_index + 1)) missing required field: name"
            return 1
        fi
        
        if [[ -z "$step_action" ]]; then
            log::error "Step '$step_name' missing required field: action"
            return 1
        fi
        
        # Validate action is supported
        if ! workflow::is_valid_action "$step_action"; then
            log::error "Step '$step_name' has unknown action: $step_action"
            return 1
        fi
        
        step_index=$((step_index + 1))
    done
    
    return 0
}

#######################################
# Check if action is valid
# Arguments:
#   $1 - Action name
# Returns:
#   0 if valid, 1 if invalid
#######################################
workflow::is_valid_action() {
    local action="${1:?Action required}"
    
    # List of supported actions
    local valid_actions=(
        # Navigation
        "navigate" "reload" "go_back" "go_forward"
        
        # Interaction
        "click" "fill_form" "type" "select" "upload_file"
        "hover" "focus" "clear"
        
        # Wait
        "wait" "wait_for_element" "wait_for_text"
        "wait_for_navigation" "wait_for_redirect" "wait_for_network_idle"
        
        # Data
        "screenshot" "extract_text" "extract_data" "extract_table"
        "get_attribute" "get_cookies" "set_cookies"
        
        # Validation
        "assert_text" "assert_url" "assert_element"
        "assert_visible" "assert_value"
        
        # Advanced
        "execute_script" "evaluate" "scroll" "dialog_respond"
        "new_tab" "switch_tab" "close_tab"
        "set_viewport" "emulate_device"
        
        # Utility
        "log" "debug" "comment"
    )
    
    for valid_action in "${valid_actions[@]}"; do
        if [[ "$action" == "$valid_action" ]]; then
            return 0
        fi
    done
    
    return 1
}

#######################################
# Extract workflow metadata
# Arguments:
#   $1 - Workflow JSON
# Returns:
#   Metadata as JSON
#######################################
workflow::extract_metadata() {
    local workflow_json="${1:?Workflow JSON required}"
    
    echo "$workflow_json" | jq '{
        name: .workflow.name,
        description: (.workflow.description // ""),
        version: (.workflow.version // "1.0.0"),
        debug_level: (.workflow.debug_level // "none"),
        parameters: (.workflow.parameters // {}),
        step_count: (.workflow.steps | length),
        actions: ([.workflow.steps[].action] | unique)
    }'
}

#######################################
# Resolve workflow variables
# Arguments:
#   $1 - Workflow JSON
#   $2 - Parameters JSON
#   $3 - Context JSON
# Returns:
#   Workflow with resolved variables
#######################################
workflow::resolve_variables() {
    local workflow_json="${1:?Workflow JSON required}"
    local params_json="${2:-{}}"
    local context_json="${3:-{}}"
    
    # Combine all variables
    local variables
    variables=$(jq -s '.[0] * .[1] * .[2]' \
        <(echo "$params_json" | jq '{params: .}') \
        <(echo "$context_json" | jq '{context: .}') \
        <(echo '{"secrets": {}}'))
    
    # Replace ${var} patterns in workflow
    local resolved_workflow="$workflow_json"
    
    # Extract all variable references
    local var_refs
    var_refs=$(echo "$workflow_json" | grep -o '\${[^}]*}' | sort -u || true)
    
    # Replace each variable
    while IFS= read -r var_ref; do
        if [[ -n "$var_ref" ]]; then
            # Extract variable path (e.g., params.username)
            local var_path="${var_ref:2:-1}"
            
            # Get variable value
            local var_value
            var_value=$(echo "$variables" | jq -r ".$var_path // \"$var_ref\"")
            
            # Replace in workflow
            resolved_workflow=$(echo "$resolved_workflow" | sed "s|${var_ref}|${var_value}|g")
        fi
    done <<< "$var_refs"
    
    echo "$resolved_workflow"
}

#######################################
# Convert workflow to simplified format for compilation
# Arguments:
#   $1 - Workflow JSON
# Returns:
#   Simplified workflow structure
#######################################
workflow::simplify() {
    local workflow_json="${1:?Workflow JSON required}"
    
    # Extract essential structure for compilation
    echo "$workflow_json" | jq '{
        name: .workflow.name,
        debug_level: (.workflow.debug_level // "none"),
        steps: [.workflow.steps[] | {
            name: .name,
            action: .action,
            params: (. | del(.name, .action, .debug, .on_error)),
            debug: (.debug // {}),
            on_error: (.on_error // "fail")
        }]
    }'
}

# Export functions
export -f workflow::parse
export -f workflow::validate_structure
export -f workflow::is_valid_action
export -f workflow::extract_metadata
export -f workflow::resolve_variables
export -f workflow::simplify