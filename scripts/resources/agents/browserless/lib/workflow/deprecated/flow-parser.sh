#!/usr/bin/env bash

#######################################
# Enhanced Browserless Workflow Parser with Flow Control
# Parses workflows with labels, jumps, sub-workflows, and conditional branching
#######################################

# Get script directory
FLOW_PARSER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BROWSERLESS_LIB_DIR="$(dirname "$FLOW_PARSER_DIR")"

# Source common utilities for logging
source "${BROWSERLESS_LIB_DIR}/common.sh" 2>/dev/null || {
    # Fallback log functions if common.sh not available
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::header() { echo "=== $* ==="; }
}

# Source original parser for base functionality
source "${FLOW_PARSER_DIR}/parser.sh"

#######################################
# Enhanced workflow parsing with flow control support
# Arguments:
#   $1 - Workflow file path
# Returns:
#   Enhanced workflow JSON with flow control graph
#######################################
workflow::parse_with_flow_control() {
    local file_path="${1:?Workflow file required}"
    
    # Parse workflow file manually to avoid override issues
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
    
    # Validate with flow control support
    if ! workflow::validate_structure_with_flow_control "$workflow_json"; then
        return 1
    fi
    
    # Enhance with flow control analysis
    local enhanced_workflow
    enhanced_workflow=$(workflow::analyze_flow_control "$workflow_json")
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    echo "$enhanced_workflow"
}

#######################################
# Analyze workflow for flow control structures
# Arguments:
#   $1 - Base workflow JSON
# Returns:
#   Enhanced workflow with flow control graph
#######################################
workflow::analyze_flow_control() {
    local workflow_json="${1:?Workflow JSON required}"
    
    log::info "🔄 Analyzing flow control structures..."
    
    # Build flow control graph
    local flow_graph
    flow_graph=$(workflow::build_flow_graph "$workflow_json")
    
    # Validate flow control structure
    if ! workflow::validate_flow_control "$flow_graph"; then
        log::error "Flow control validation failed"
        return 1
    fi
    
    # Add flow control metadata to workflow
    echo "$workflow_json" | jq \
        --argjson flow_graph "$flow_graph" \
        '. + {
            flow_control: {
                enabled: true,
                graph: $flow_graph,
                execution_model: "state_machine"
            }
        }'
}

#######################################
# Build flow control graph from workflow steps
# Arguments:
#   $1 - Workflow JSON
# Returns:
#   Flow control graph JSON
#######################################
workflow::build_flow_graph() {
    local workflow_json="${1:?Workflow JSON required}"
    
    local steps
    steps=$(echo "$workflow_json" | jq '.workflow.steps')
    
    local step_count
    step_count=$(echo "$steps" | jq 'length')
    
    # Initialize graph structure
    local graph='{
        "steps": {},
        "labels": {},
        "entry_point": null,
        "jumps": {},
        "sub_workflows": {},
        "execution_order": []
    }'
    
    # Process each step to build graph
    local step_index=0
    local entry_point_set=false
    
    while [[ $step_index -lt $step_count ]]; do
        local step
        step=$(echo "$steps" | jq ".[$step_index]")
        
        local step_name
        step_name=$(echo "$step" | jq -r '.name')
        
        local step_label
        step_label=$(echo "$step" | jq -r '.label // empty')
        
        local step_action
        step_action=$(echo "$step" | jq -r '.action')
        
        # Set entry point (first step without explicit entry point)
        if [[ "$entry_point_set" == "false" ]]; then
            graph=$(echo "$graph" | jq --arg step "$step_name" '.entry_point = $step')
            entry_point_set=true
        fi
        
        # Add step to graph
        graph=$(echo "$graph" | jq --argjson step "$step" --arg name "$step_name" '.steps[$name] = $step')
        
        # Add label mapping if present
        if [[ -n "$step_label" ]]; then
            graph=$(echo "$graph" | jq --arg label "$step_label" --arg step "$step_name" '.labels[$label] = $step')
        fi
        
        # Analyze jump targets
        local jump_targets
        jump_targets=$(workflow::extract_jump_targets "$step")
        
        if [[ "$jump_targets" != "[]" ]]; then
            graph=$(echo "$graph" | jq --argjson targets "$jump_targets" --arg step "$step_name" '.jumps[$step] = $targets')
        fi
        
        # Add to execution order for linear fallback
        graph=$(echo "$graph" | jq --arg step "$step_name" '.execution_order += [$step]')
        
        step_index=$((step_index + 1))
    done
    
    # Process sub-workflows if present
    local sub_workflows
    sub_workflows=$(echo "$workflow_json" | jq '.workflow.sub_workflows // {}')
    
    if [[ "$sub_workflows" != "{}" ]]; then
        graph=$(echo "$graph" | jq --argjson subs "$sub_workflows" '.sub_workflows = $subs')
    fi
    
    echo "$graph"
}

#######################################
# Extract jump targets from a step
# Arguments:
#   $1 - Step JSON
# Returns:
#   Array of jump targets
#######################################
workflow::extract_jump_targets() {
    local step_json="${1:?Step JSON required}"
    
    local targets='[]'
    local action
    action=$(echo "$step_json" | jq -r '.action')
    
    # Check different jump patterns
    case "$action" in
        "jump_to"|"goto")
            local target
            target=$(echo "$step_json" | jq -r '.target // .jump_to // empty')
            if [[ -n "$target" ]]; then
                targets=$(echo "$targets" | jq --arg t "$target" '. + [$t]')
            fi
            ;;
        "jump_if"|"conditional_jump")
            local success_target
            success_target=$(echo "$step_json" | jq -r '.on_success.jump_to // .success_target // empty')
            local failure_target
            failure_target=$(echo "$step_json" | jq -r '.on_failure.jump_to // .failure_target // .else_jump_to // empty')
            
            if [[ -n "$success_target" ]]; then
                targets=$(echo "$targets" | jq --arg t "$success_target" '. + [$t]')
            fi
            if [[ -n "$failure_target" ]]; then
                targets=$(echo "$targets" | jq --arg t "$failure_target" '. + [$t]')
            fi
            ;;
        "switch"|"branch")
            local cases
            cases=$(echo "$step_json" | jq -r '.cases // {} | to_entries[] | .value')
            while IFS= read -r target; do
                if [[ -n "$target" ]]; then
                    targets=$(echo "$targets" | jq --arg t "$target" '. + [$t]')
                fi
            done <<< "$cases"
            
            local default_target
            default_target=$(echo "$step_json" | jq -r '.default // empty')
            if [[ -n "$default_target" ]]; then
                targets=$(echo "$targets" | jq --arg t "$default_target" '. + [$t]')
            fi
            ;;
        "call_workflow"|"include_workflow"|"call_sub_workflow")
            local return_target
            return_target=$(echo "$step_json" | jq -r '.return_to // .on_success.jump_to // empty')
            if [[ -n "$return_target" ]]; then
                targets=$(echo "$targets" | jq --arg t "$return_target" '. + [$t]')
            fi
            ;;
        *)
            # Check for general flow control attributes
            local jump_to
            jump_to=$(echo "$step_json" | jq -r '.jump_to // empty')
            if [[ -n "$jump_to" ]]; then
                targets=$(echo "$targets" | jq --arg t "$jump_to" '. + [$t]')
            fi
            
            # Check on_success/on_error flow control
            local on_success_jump
            on_success_jump=$(echo "$step_json" | jq -r '.on_success.jump_to // empty')
            if [[ -n "$on_success_jump" ]]; then
                targets=$(echo "$targets" | jq --arg t "$on_success_jump" '. + [$t]')
            fi
            
            local on_error_jump
            on_error_jump=$(echo "$step_json" | jq -r '.on_error.jump_to // empty')
            if [[ -n "$on_error_jump" ]]; then
                targets=$(echo "$targets" | jq --arg t "$on_error_jump" '. + [$t]')
            fi
            ;;
    esac
    
    echo "$targets"
}

#######################################
# Validate flow control structure
# Arguments:
#   $1 - Flow control graph JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
workflow::validate_flow_control() {
    local flow_graph="${1:?Flow graph required}"
    
    log::info "✅ Validating flow control structure..."
    
    # Check entry point exists
    local entry_point
    entry_point=$(echo "$flow_graph" | jq -r '.entry_point // empty')
    
    if [[ -z "$entry_point" ]]; then
        log::error "Flow control graph missing entry point"
        return 1
    fi
    
    # Check all jump targets exist
    local jumps
    jumps=$(echo "$flow_graph" | jq -r '.jumps // {}')
    
    if [[ "$jumps" != "{}" ]]; then
        echo "$jumps" | jq -r 'to_entries[] | .value[]' | while read -r target; do
            if [[ -n "$target" ]]; then
                # Check if target exists as step name or label
                local step_exists
                step_exists=$(echo "$flow_graph" | jq -r --arg target "$target" '
                    (.steps | has($target)) or (.labels | has($target))
                ')
                
                if [[ "$step_exists" != "true" ]]; then
                    log::error "Jump target not found: $target"
                    return 1
                fi
            fi
        done
    fi
    
    # Check for circular dependencies (basic)
    if ! workflow::check_circular_dependencies "$flow_graph"; then
        log::error "Circular dependency detected in flow control"
        return 1
    fi
    
    log::success "✅ Flow control structure is valid"
    return 0
}

#######################################
# Check for circular dependencies in flow graph
# Arguments:
#   $1 - Flow control graph JSON
# Returns:
#   0 if no cycles, 1 if cycles detected
#######################################
workflow::check_circular_dependencies() {
    local flow_graph="${1:?Flow graph required}"
    
    # Simple cycle detection - could be enhanced with proper DFS
    local steps
    steps=$(echo "$flow_graph" | jq -r '.steps | keys[]')
    
    local visited_count=0
    local max_visits=1000  # Prevent infinite loops during validation
    
    while IFS= read -r step; do
        if [[ -n "$step" ]]; then
            visited_count=$((visited_count + 1))
            if [[ $visited_count -gt $max_visits ]]; then
                log::warn "Potential circular dependency detected (visited $visited_count steps)"
                return 1
            fi
        fi
    done <<< "$steps"
    
    return 0
}

#######################################
# Enhanced action validation with flow control actions
# Arguments:
#   $1 - Action name
# Returns:
#   0 if valid, 1 if invalid
#######################################
workflow::is_valid_flow_control_action() {
    local action="${1:?Action required}"
    
    # Check base actions first
    if workflow::is_valid_action "$action"; then
        return 0
    fi
    
    # Check flow control specific actions
    local flow_control_actions=(
        # Jump control
        "jump_to" "goto" "jump_if" "conditional_jump"
        
        # Branching
        "switch" "branch" "route"
        
        # Sub-workflow
        "call_workflow" "include_workflow" "call_sub_workflow"
        
        # Loop control
        "repeat" "for_each" "while" "until"
        
        # Advanced flow
        "wait_for_one_of" "parallel" "sequence"
        
        # State management
        "set_state" "get_state" "checkpoint"
    )
    
    for valid_action in "${flow_control_actions[@]}"; do
        if [[ "$action" == "$valid_action" ]]; then
            return 0
        fi
    done
    
    return 1
}

#######################################
# Enhanced workflow simplification with flow control
# Arguments:
#   $1 - Enhanced workflow JSON
# Returns:
#   Simplified workflow for compilation
#######################################
workflow::simplify_with_flow_control() {
    local workflow_json="${1:?Workflow JSON required}"
    
    # Extract flow control graph
    local flow_graph
    flow_graph=$(echo "$workflow_json" | jq '.flow_control.graph // {}')
    
    # Create enhanced simplification
    echo "$workflow_json" | jq --argjson flow_graph "$flow_graph" '{
        name: .workflow.name,
        debug_level: (.workflow.debug_level // "none"),
        flow_control: {
            enabled: true,
            graph: $flow_graph
        },
        steps: [.workflow.steps[] | {
            name: .name,
            label: (.label // null),
            action: .action,
            params: (. | del(.name, .action, .debug, .on_error, .label, .jump_to, .on_success, .on_failure)),
            debug: (.debug // {}),
            on_error: (.on_error // "fail"),
            flow_control: {
                jump_to: (.jump_to // null),
                on_success: (.on_success // null),
                on_failure: (.on_failure // null),
                cases: (.cases // null),
                default: (.default // null)
            }
        }],
        sub_workflows: (.workflow.sub_workflows // {})
    }'
}

# Enhanced validation (don't override original)
workflow::validate_structure_with_flow_control() {
    local workflow_json="${1:?Workflow JSON required}"
    
    # Basic structure validation (inline to avoid function override issues)
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
    
    # Enhanced validation for steps with flow control actions
    local steps
    steps=$(echo "$workflow_json" | jq '.workflow.steps // []')
    
    local step_index=0
    while [[ $step_index -lt $steps_count ]]; do
        local step
        step=$(echo "$steps" | jq ".[$step_index]")
        
        local step_action
        step_action=$(echo "$step" | jq -r '.action // empty')
        
        local step_name
        step_name=$(echo "$step" | jq -r '.name // empty')
        
        # Validate action using enhanced validation
        if ! workflow::is_valid_flow_control_action "$step_action"; then
            log::error "Step '$step_name' has unknown action: $step_action"
            return 1
        fi
        
        step_index=$((step_index + 1))
    done
    
    return 0
}

# Export enhanced functions
export -f workflow::parse_with_flow_control
export -f workflow::analyze_flow_control
export -f workflow::build_flow_graph
export -f workflow::extract_jump_targets
export -f workflow::validate_flow_control
export -f workflow::check_circular_dependencies
export -f workflow::is_valid_flow_control_action
export -f workflow::simplify_with_flow_control