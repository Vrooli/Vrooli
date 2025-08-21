#!/usr/bin/env bash
########################################
# Simple Workflow Interpreter
# 
# Replaces the complex YAML→JSON→JavaScript compiler
# with a simple interpreter that executes atomic
# browser operations directly from YAML.
#
# Key differences from compiler:
#   - No JSON compilation
#   - No JavaScript generation
#   - Direct execution of browser operations
#   - Step-by-step visibility
#   - Simple error handling
########################################
set -euo pipefail

# Get script directory
INTERPRETER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BROWSERLESS_LIB_DIR="$(dirname "$INTERPRETER_DIR")"

# Source required libraries
source "/home/matthalloran8/Vrooli/scripts/lib/utils/log.sh" 2>/dev/null || true
source "${BROWSERLESS_LIB_DIR}/browser-ops.sh"
source "${BROWSERLESS_LIB_DIR}/session-manager.sh"

# Workflow execution state
declare -A WORKFLOW_LABELS=()  # Map of label names to step indexes
declare -a WORKFLOW_STEPS=()   # Array of workflow steps
declare CURRENT_STEP=0
declare SESSION_ID=""
declare -A VARIABLES=()        # Store variables from evaluate actions

########################################
# Parse YAML workflow into steps array
########################################
workflow::parse_yaml() {
    local yaml_file="$1"
    
    if [[ ! -f "$yaml_file" ]]; then
        log::error "Workflow file not found: $yaml_file"
        return 1
    fi
    
    log::info "Parsing workflow: $yaml_file"
    
    # Use yq to parse YAML and extract steps
    local steps_json
    if ! steps_json=$(yq eval -o=json '.steps' "$yaml_file" 2>/dev/null); then
        log::error "Failed to parse YAML file"
        return 1
    fi
    
    # Load steps into array
    local step_count
    step_count=$(echo "$steps_json" | jq 'length')
    
    for ((i=0; i<step_count; i++)); do
        local step
        step=$(echo "$steps_json" | jq -r ".[$i]")
        WORKFLOW_STEPS+=("$step")
        
        # Check if step has a label and store it
        local label
        label=$(echo "$step" | jq -r '.label // empty')
        if [[ -n "$label" ]]; then
            WORKFLOW_LABELS["$label"]=$i
            log::debug "Registered label '$label' at step $i"
        fi
    done
    
    log::info "Loaded ${#WORKFLOW_STEPS[@]} steps"
    return 0
}

########################################
# Execute a single workflow step
########################################
workflow::execute_step() {
    local step="$1"
    local step_index="$2"
    
    local name=$(echo "$step" | jq -r '.name // "unnamed"')
    local action=$(echo "$step" | jq -r '.action // empty')
    
    log::info "[Step $step_index] Executing: $name ($action)"
    
    case "$action" in
        navigate)
            local url=$(echo "$step" | jq -r '.url // empty')
            url=$(workflow::substitute_variables "$url")
            browser::navigate "$url" "$SESSION_ID"
            ;;
            
        click)
            local selector=$(echo "$step" | jq -r '.selector // empty')
            selector=$(workflow::substitute_variables "$selector")
            browser::click "$selector" "$SESSION_ID"
            ;;
            
        type|fill)
            local selector=$(echo "$step" | jq -r '.selector // empty')
            local text=$(echo "$step" | jq -r '.text // empty')
            selector=$(workflow::substitute_variables "$selector")
            text=$(workflow::substitute_variables "$text")
            browser::fill "$selector" "$text" "$SESSION_ID"
            ;;
            
        wait)
            local duration=$(echo "$step" | jq -r '.duration // 1')
            log::debug "Waiting ${duration} seconds"
            sleep "$duration"
            ;;
            
        wait_for_element)
            local selector=$(echo "$step" | jq -r '.selector // empty')
            local timeout=$(echo "$step" | jq -r '.timeout // 30')
            selector=$(workflow::substitute_variables "$selector")
            browser::wait_for_element "$selector" "$timeout" "$SESSION_ID"
            ;;
            
        screenshot)
            local path=$(echo "$step" | jq -r '.path // "screenshot.png"')
            path=$(workflow::substitute_variables "$path")
            browser::screenshot "$path" "$SESSION_ID"
            log::info "Screenshot saved to: $path"
            ;;
            
        evaluate)
            local script=$(echo "$step" | jq -r '.script // empty')
            local var_name=$(echo "$step" | jq -r '.variable // empty')
            script=$(workflow::substitute_variables "$script")
            
            local result
            result=$(browser::evaluate_script "$script" "$SESSION_ID")
            
            if [[ -n "$var_name" ]]; then
                VARIABLES["$var_name"]="$result"
                log::debug "Stored variable '$var_name' = '$result'"
            fi
            ;;
            
        if|condition)
            local condition=$(echo "$step" | jq -r '.condition // empty')
            local then_action=$(echo "$step" | jq -r '.then // empty')
            local else_action=$(echo "$step" | jq -r '.else // empty')
            
            # Evaluate condition
            local result
            if [[ "$condition" =~ ^selector: ]]; then
                # Check if element exists
                local selector="${condition#selector:}"
                selector=$(workflow::substitute_variables "$selector")
                if browser::element_exists "$selector" "$SESSION_ID"; then
                    result="true"
                else
                    result="false"
                fi
            else
                # Evaluate as JavaScript
                condition=$(workflow::substitute_variables "$condition")
                result=$(browser::evaluate_script "$condition" "$SESSION_ID")
            fi
            
            # Execute appropriate branch
            if [[ "$result" == "true" ]]; then
                if [[ -n "$then_action" ]]; then
                    workflow::goto_label "$then_action"
                fi
            else
                if [[ -n "$else_action" ]]; then
                    workflow::goto_label "$else_action"
                fi
            fi
            ;;
            
        goto|jump)
            local label=$(echo "$step" | jq -r '.label // empty')
            workflow::goto_label "$label"
            ;;
            
        loop)
            local count=$(echo "$step" | jq -r '.count // 1')
            local start_label=$(echo "$step" | jq -r '.start // empty')
            local end_label=$(echo "$step" | jq -r '.end // empty')
            
            for ((i=0; i<count; i++)); do
                log::debug "Loop iteration $((i+1)) of $count"
                if [[ -n "$start_label" ]]; then
                    workflow::goto_label "$start_label"
                    workflow::execute_until_label "$end_label"
                fi
            done
            ;;
            
        log)
            local message=$(echo "$step" | jq -r '.message // empty')
            message=$(workflow::substitute_variables "$message")
            log::info "[Workflow Log] $message"
            ;;
            
        *)
            log::warn "Unknown action: $action"
            ;;
    esac
    
    return 0
}

########################################
# Substitute variables in text
########################################
workflow::substitute_variables() {
    local text="$1"
    
    # Replace {{variable}} with actual values
    for var in "${!VARIABLES[@]}"; do
        text="${text//\{\{$var\}\}/${VARIABLES[$var]}}"
    done
    
    # Replace environment variables
    text=$(envsubst <<< "$text")
    
    echo "$text"
}

########################################
# Jump to a labeled step
########################################
workflow::goto_label() {
    local label="$1"
    
    if [[ -z "$label" ]]; then
        return 0
    fi
    
    if [[ -n "${WORKFLOW_LABELS[$label]:-}" ]]; then
        CURRENT_STEP="${WORKFLOW_LABELS[$label]}"
        log::debug "Jumping to label '$label' (step $CURRENT_STEP)"
    else
        log::error "Label not found: $label"
        return 1
    fi
}

########################################
# Execute steps until a specific label
########################################
workflow::execute_until_label() {
    local end_label="$1"
    local end_step="${WORKFLOW_LABELS[$end_label]:-${#WORKFLOW_STEPS[@]}}"
    
    while [[ $CURRENT_STEP -lt $end_step ]] && [[ $CURRENT_STEP -lt ${#WORKFLOW_STEPS[@]} ]]; do
        local step="${WORKFLOW_STEPS[$CURRENT_STEP]}"
        workflow::execute_step "$step" "$CURRENT_STEP"
        CURRENT_STEP=$((CURRENT_STEP + 1))
    done
}

########################################
# Main workflow execution
########################################
workflow::run() {
    local yaml_file="$1"
    local session_name="${2:-workflow_$$}"
    
    # Parse the workflow
    if ! workflow::parse_yaml "$yaml_file"; then
        return 1
    fi
    
    # Create browser session
    SESSION_ID="$session_name"
    log::info "Creating browser session: $SESSION_ID"
    if ! session::create "$SESSION_ID"; then
        log::error "Failed to create browser session"
        return 1
    fi
    
    # Execute workflow steps
    local success=0
    CURRENT_STEP=0
    
    while [[ $CURRENT_STEP -lt ${#WORKFLOW_STEPS[@]} ]]; do
        local step="${WORKFLOW_STEPS[$CURRENT_STEP]}"
        
        if ! workflow::execute_step "$step" "$CURRENT_STEP"; then
            log::error "Step $CURRENT_STEP failed"
            success=1
            break
        fi
        
        CURRENT_STEP=$((CURRENT_STEP + 1))
    done
    
    # Cleanup
    log::info "Destroying browser session: $SESSION_ID"
    session::destroy "$SESSION_ID"
    
    if [[ $success -eq 0 ]]; then
        log::success "Workflow completed successfully"
    else
        log::error "Workflow failed at step $CURRENT_STEP"
    fi
    
    return $success
}

########################################
# Entry point for direct execution
########################################
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -lt 1 ]]; then
        echo "Usage: $0 <workflow.yaml> [session_name]"
        exit 1
    fi
    
    workflow::run "$@"
fi