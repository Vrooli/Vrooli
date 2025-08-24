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

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
INTERPRETER_DIR="${APP_ROOT}/resources/browserless/lib/workflow"
BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"

# Source required libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "${BROWSERLESS_LIB_DIR}/browser-ops.sh"
source "${BROWSERLESS_LIB_DIR}/session-manager.sh"

# Source stateful browser operations if available
# TEMPORARILY DISABLED for testing combined workflow
USE_STATEFUL_SESSIONS=false
log::debug "Stateful sessions disabled for testing"

# Workflow execution state
declare -A WORKFLOW_LABELS=()     # Map of label names to step indexes
declare -a WORKFLOW_STEPS=()      # Array of workflow steps
declare -A SUB_WORKFLOWS=()       # Map of sub-workflow names to their steps
declare -A WORKFLOW_PARAMS=()     # Parameters passed to workflow
declare -A WORKFLOW_CONTEXT=()    # Context variables like outputDir
declare CURRENT_STEP=0
declare SESSION_ID=""
declare -A VARIABLES=()           # Store variables from evaluate actions
declare LAST_EVAL_RESULT=""       # Store last evaluation result
declare DEBUG_OUTPUT_DIR=""       # Directory for debug outputs

########################################
# Parse YAML workflow into steps and sub-workflows
########################################
workflow::parse_yaml() {
    local yaml_file="$1"
    
    if [[ ! -f "$yaml_file" ]]; then
        log::error "Workflow file not found: $yaml_file"
        return 1
    fi
    
    log::info "Parsing workflow: $yaml_file"
    
    # Parse main workflow steps
    local steps_json
    if ! steps_json=$(yq eval -o=json '.workflow.steps // .steps' "$yaml_file" 2>/dev/null); then
        log::error "Failed to parse workflow steps"
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
    
    # Parse sub-workflows
    local sub_workflows_json
    if sub_workflows_json=$(yq eval -o=json '.workflow.sub_workflows // .sub_workflows // {}' "$yaml_file" 2>/dev/null); then
        local sub_names
        sub_names=$(echo "$sub_workflows_json" | jq -r 'keys[]' 2>/dev/null || true)
        
        while IFS= read -r sub_name; do
            if [[ -n "$sub_name" ]]; then
                local sub_steps
                sub_steps=$(echo "$sub_workflows_json" | jq -r ".\"$sub_name\".steps")
                SUB_WORKFLOWS["$sub_name"]="$sub_steps"
                log::debug "Registered sub-workflow: $sub_name"
            fi
        done <<< "$sub_names"
    fi
    
    log::info "Loaded ${#WORKFLOW_STEPS[@]} main steps and ${#SUB_WORKFLOWS[@]} sub-workflows"
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
    local debug_json=$(echo "$step" | jq -r '.debug // {}')
    local output_path=$(echo "$step" | jq -r '.output // empty')
    
    log::info "[Step $step_index] Executing: $name ($action)"
    
    # Handle pre-step debugging
    # Skip pre-debug for browser_script since it handles its own screenshots
    if [[ "$action" != "browser_script" ]]; then
        workflow::handle_debug "$debug_json" "pre" "$step_index" "$name"
    fi
    
    case "$action" in
        navigate)
            local url=$(echo "$step" | jq -r '.url // empty')
            url=$(workflow::substitute_variables "$url")
            
            if [[ "$USE_STATEFUL_SESSIONS" == "true" ]]; then
                browser::navigate_stateful "$url" "$SESSION_ID"
            else
                browser::navigate "$url" "$SESSION_ID"
            fi
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
            
        browser_script)
            # Execute JavaScript with full access to page object (browser context)
            local script=$(echo "$step" | jq -r '.script // empty')
            local var_name=$(echo "$step" | jq -r '.variable // empty')
            local output_path=$(echo "$step" | jq -r '.output // empty')
            script=$(workflow::substitute_variables "$script")
            
            log::debug "Executing browser script"
            
            local result
            result=$(browser::execute_js "$script" "$SESSION_ID")
            
            # Always store as last eval result for conditions
            LAST_EVAL_RESULT="$result"
            
            if [[ -n "$var_name" ]]; then
                VARIABLES["$var_name"]="$result"
                log::debug "Stored variable '$var_name' = '$result'"
            fi
            
            log::debug "Browser script result: $result"
            
            # Save screenshot if present in result and remove base64 data
            if echo "$result" | jq -e '.screenshotData' > /dev/null 2>&1; then
                local screenshot_data screenshot_path
                screenshot_data=$(echo "$result" | jq -r '.screenshotData')
                screenshot_path=$(echo "$result" | jq -r '.screenshotPath')
                
                if [[ -n "$screenshot_data" ]] && [[ "$screenshot_data" != "null" ]]; then
                    # Decode base64 screenshot to actual PNG file
                    echo "$screenshot_data" | base64 -d > "$screenshot_path" 2>/dev/null || true
                    log::info "📸 Screenshot saved to: $screenshot_path"
                fi
            fi
            
            # ALWAYS remove base64 data from result - humans don't want to see this garbage
            result=$(echo "$result" | jq 'del(.screenshotData) | del(.base64) | del(.data) | walk(if type == "object" then del(.screenshotData) | del(.base64) | del(.data) else . end)')
            
            # Save output if specified
            if [[ -n "$output_path" ]]; then
                local resolved_path
                resolved_path=$(workflow::substitute_variables "$output_path")
                mkdir -p "$(dirname "$resolved_path")"
                
                # ALWAYS remove any base64 data from final output - humans don't want to see this
                local clean_result
                if echo "$result" | jq -e '.' > /dev/null 2>&1; then
                    # JSON result - remove any base64 data fields
                    clean_result=$(echo "$result" | jq 'del(.screenshotData) | del(.base64) | del(.data) | walk(if type == "object" then del(.screenshotData) | del(.base64) | del(.data) else . end)')
                else
                    # Non-JSON result - use as is
                    clean_result="$result"
                fi
                
                echo "$clean_result" | jq '.' > "$resolved_path" 2>/dev/null || echo "$clean_result" > "$resolved_path"
                log::info "Result saved to: $resolved_path"
            fi
            ;;
            
        wait)
            local duration=$(echo "$step" | jq -r '.duration // 1000')
            # Convert milliseconds to seconds for sleep command
            local duration_seconds=$(echo "scale=3; $duration / 1000" | bc 2>/dev/null || echo "1")
            log::debug "Waiting ${duration}ms (${duration_seconds}s)"
            sleep "$duration_seconds"
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
            
            if [[ "$USE_STATEFUL_SESSIONS" == "true" ]]; then
                browser::screenshot_stateful "$path" "$SESSION_ID"
            else
                browser::screenshot "$path" "$SESSION_ID"
            fi
            log::info "Screenshot saved to: $path"
            ;;
            
        evaluate)
            local script=$(echo "$step" | jq -r '.script // empty')
            local var_name=$(echo "$step" | jq -r '.variable // empty')
            script=$(workflow::substitute_variables "$script")
            
            local result
            if [[ "$USE_STATEFUL_SESSIONS" == "true" ]]; then
                result=$(browser::evaluate_stateful "$script" "$SESSION_ID")
            else
                result=$(browser::evaluate "$script" "$SESSION_ID")
            fi
            
            # Always store as last eval result for conditions
            LAST_EVAL_RESULT="$result"
            
            if [[ -n "$var_name" ]]; then
                VARIABLES["$var_name"]="$result"
                log::debug "Stored variable '$var_name' = '$result'"
            fi
            
            log::debug "Evaluation result: $result"
            
            # Save output if specified
            if [[ -n "$output_path" ]]; then
                local resolved_path
                resolved_path=$(workflow::substitute_variables "$output_path")
                mkdir -p "$(dirname "$resolved_path")"
                echo "$result" > "$resolved_path"
                log::debug "Saved result to: $resolved_path"
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
                result=$(browser::evaluate "$condition" "$SESSION_ID")
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
            
        jump_if)
            local condition=$(echo "$step" | jq -r '.condition // empty')
            local success_target=$(echo "$step" | jq -r '.success_target // empty')
            local failure_target=$(echo "$step" | jq -r '.failure_target // empty')
            
            # Substitute variables in condition
            condition=$(workflow::substitute_variables "$condition")
            
            local result="false"
            if [[ "$condition" =~ ^selector: ]]; then
                # Check if element exists
                local selector="${condition#selector:}"
                if browser::element_exists "$selector" "$SESSION_ID"; then
                    result="true"
                fi
            else
                # Evaluate as JavaScript expression
                result=$(browser::evaluate "return ($condition) ? 'true' : 'false';" "$SESSION_ID")
            fi
            
            log::debug "Condition '$condition' evaluated to: $result"
            
            if [[ "$result" == "true" ]] && [[ -n "$success_target" ]]; then
                workflow::goto_label "$success_target"
                return 0  # Skip incrementing step counter
            elif [[ "$result" == "false" ]] && [[ -n "$failure_target" ]]; then
                workflow::goto_label "$failure_target"
                return 0  # Skip incrementing step counter
            fi
            ;;
            
        call_sub_workflow)
            local sub_workflow_name=$(echo "$step" | jq -r '.sub_workflow // empty')
            local jump_to=$(echo "$step" | jq -r '.jump_to // empty')
            
            if [[ -z "$sub_workflow_name" ]]; then
                log::error "Sub-workflow name required for call_sub_workflow"
                return 1
            fi
            
            log::info "Calling sub-workflow: $sub_workflow_name"
            
            if workflow::execute_sub_workflow "$sub_workflow_name"; then
                log::success "Sub-workflow '$sub_workflow_name' completed successfully"
                if [[ -n "$jump_to" ]]; then
                    workflow::goto_label "$jump_to"
                    return 0  # Skip incrementing step counter
                fi
            else
                log::error "Sub-workflow '$sub_workflow_name' failed"
                return 1
            fi
            ;;
            
        switch)
            local expression=$(echo "$step" | jq -r '.expression // empty')
            local cases_json=$(echo "$step" | jq -r '.cases // {}')
            local default_target=$(echo "$step" | jq -r '.default // empty')
            
            # Substitute variables and evaluate expression
            expression=$(workflow::substitute_variables "$expression")
            local switch_result
            switch_result=$(browser::evaluate "return ($expression);" "$SESSION_ID")
            
            log::debug "Switch expression '$expression' evaluated to: '$switch_result'"
            
            # Check if result matches any case
            local target_label
            target_label=$(echo "$cases_json" | jq -r ".\"$switch_result\" // empty")
            
            if [[ -n "$target_label" ]]; then
                log::debug "Switch matched case '$switch_result' -> '$target_label'"
                workflow::goto_label "$target_label"
                return 0  # Skip incrementing step counter
            elif [[ -n "$default_target" ]]; then
                log::debug "Switch using default target: '$default_target'"
                workflow::goto_label "$default_target"
                return 0  # Skip incrementing step counter
            else
                log::warn "Switch expression '$switch_result' did not match any case and no default provided"
            fi
            ;;
            
        navigate_and_evaluate)
            local url=$(echo "$step" | jq -r '.url // empty')
            local script=$(echo "$step" | jq -r '.script // empty')
            local var_name=$(echo "$step" | jq -r '.variable // empty')
            
            url=$(workflow::substitute_variables "$url")
            script=$(workflow::substitute_variables "$script")
            
            log::info "Navigating to $url and executing combined script"
            
            # Create combined JavaScript that navigates and then evaluates
            local combined_js="
                // Navigate to the URL
                console.log('Navigating to: $url');
                await page.goto('$url', { 
                    waitUntil: 'networkidle2',
                    timeout: 30000 
                });
                
                // Wait for page to be ready
                await page.waitForFunction(() => document.readyState === 'complete', { timeout: 10000 }).catch(() => {});
                await new Promise(resolve => setTimeout(resolve, 2000));
                
                console.log('Navigation completed to:', page.url());
                console.log('Page title:', await page.title());
                
                // Execute the user script and return result
                const scriptResult = await (async function() {
                    $script
                })();
                
                return scriptResult;
            "
            
            local result
            result=$(browser::execute_js "$combined_js" "$SESSION_ID")
            
            # Always store as last eval result for conditions
            LAST_EVAL_RESULT="$result"
            
            if [[ -n "$var_name" ]]; then
                VARIABLES["$var_name"]="$result"
                log::debug "Stored variable '$var_name' = '$result'"
            fi
            
            # Save output if specified
            if [[ -n "$output_path" ]]; then
                local resolved_path
                resolved_path=$(workflow::substitute_variables "$output_path")
                mkdir -p "$(dirname "$resolved_path")"
                echo "$result" > "$resolved_path"
                log::debug "Saved result to: $resolved_path"
            fi
            
            log::debug "Navigate and evaluate result: $result"
            ;;
            
        *)
            log::warn "Unknown action: $action"
            ;;
    esac
    
    # Handle post-step debugging
    # Skip post-debug for browser_script since it handles its own screenshots
    if [[ "$action" != "browser_script" ]]; then
        workflow::handle_debug "$debug_json" "post" "$step_index" "$name"
    fi
    
    return 0
}

########################################
# Handle debugging for a step
########################################
workflow::handle_debug() {
    local debug_json="$1"
    local phase="$2"        # "pre" or "post"
    local step_index="$3"
    local step_name="$4"
    
    if [[ "$debug_json" == "{}" ]] || [[ "$debug_json" == "null" ]]; then
        return 0
    fi
    
    # Check if this phase should be debugged
    local debug_screenshot=$(echo "$debug_json" | jq -r '.screenshot // false')
    local debug_html=$(echo "$debug_json" | jq -r '.html // false')
    local debug_console=$(echo "$debug_json" | jq -r '.console // false')
    
    local debug_prefix="${DEBUG_OUTPUT_DIR}/step${step_index}_${step_name}_${phase}"
    
    if [[ "$debug_screenshot" == "true" ]]; then
        local screenshot_path="${debug_prefix}.png"
        if browser::screenshot "$screenshot_path" "$SESSION_ID"; then
            log::info "🖼️  DEBUG: Screenshot saved to $screenshot_path"
        else
            log::warn "Failed to capture debug screenshot"
        fi
    fi
    
    if [[ "$debug_html" == "true" ]]; then
        local html_path="${debug_prefix}.html"
        local html_content
        if html_content=$(browser::evaluate "return document.documentElement.outerHTML;" "$SESSION_ID"); then
            echo "$html_content" > "$html_path"
            log::info "📄 DEBUG: HTML saved to $html_path"
        else
            log::warn "Failed to capture debug HTML"
        fi
    fi
    
    if [[ "$debug_console" == "true" ]]; then
        local console_path="${debug_prefix}_console.json"
        local console_logs
        if console_logs=$(browser::get_console_logs "$SESSION_ID"); then
            echo "$console_logs" > "$console_path"
            log::info "🖥️  DEBUG: Console logs saved to $console_path"
        else
            log::warn "Failed to capture debug console logs"
        fi
    fi
}

########################################
# Substitute variables in text
########################################
workflow::substitute_variables() {
    local text="$1"
    
    # Replace ${params.variable} with workflow parameters
    for param in "${!WORKFLOW_PARAMS[@]}"; do
        text="${text//\$\{params.$param\}/${WORKFLOW_PARAMS[$param]}}"
    done
    
    # Replace ${context.variable} with context variables
    for context_var in "${!WORKFLOW_CONTEXT[@]}"; do
        text="${text//\$\{context.$context_var\}/${WORKFLOW_CONTEXT[$context_var]}}"
    done
    
    # Replace {{variable}} with runtime variables
    for var in "${!VARIABLES[@]}"; do
        text="${text//\{\{$var\}\}/${VARIABLES[$var]}}"
    done
    
    # Replace window.__lastEvalResult with stored result
    if [[ -n "$LAST_EVAL_RESULT" ]]; then
        text="${text//window.__lastEvalResult/$LAST_EVAL_RESULT}"
    fi
    
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
# Execute a sub-workflow
########################################
workflow::execute_sub_workflow() {
    local sub_workflow_name="$1"
    
    if [[ -z "${SUB_WORKFLOWS[$sub_workflow_name]:-}" ]]; then
        log::error "Sub-workflow not found: $sub_workflow_name"
        return 1
    fi
    
    log::info "Executing sub-workflow: $sub_workflow_name"
    
    # Get sub-workflow steps
    local sub_steps_json="${SUB_WORKFLOWS[$sub_workflow_name]}"
    local sub_step_count
    sub_step_count=$(echo "$sub_steps_json" | jq 'length')
    
    # Execute each step in the sub-workflow
    for ((i=0; i<sub_step_count; i++)); do
        local sub_step
        sub_step=$(echo "$sub_steps_json" | jq -r ".[$i]")
        
        local step_name
        step_name=$(echo "$sub_step" | jq -r '.name // "unnamed"')
        
        log::debug "[Sub-workflow $sub_workflow_name] Step $i: $step_name"
        
        if ! workflow::execute_step "$sub_step" "sub_$i"; then
            log::error "Sub-workflow '$sub_workflow_name' failed at step $i"
            return 1
        fi
    done
    
    log::success "Sub-workflow '$sub_workflow_name' completed"
    return 0
}

########################################
# Main workflow execution
########################################
workflow::run() {
    local yaml_file="$1"
    shift
    
    local session_name="workflow_$$"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --param)
                if [[ "$2" =~ ^([^=]+)=(.*)$ ]]; then
                    local param_name="${BASH_REMATCH[1]}"
                    local param_value="${BASH_REMATCH[2]}"
                    WORKFLOW_PARAMS["$param_name"]="$param_value"
                    log::debug "Set parameter: $param_name = $param_value"
                    shift 2
                else
                    log::error "Invalid parameter format. Use --param name=value"
                    return 1
                fi
                ;;
            --session)
                session_name="$2"
                shift 2
                ;;
            *)
                # Treat as session name for backward compatibility
                session_name="$1"
                shift
                ;;
        esac
    done
    
    log::info "Starting workflow with ${#WORKFLOW_PARAMS[@]} parameters"
    
    # Set up debug output directory
    DEBUG_OUTPUT_DIR="${HOME}/.vrooli/browserless/debug/$(basename "$yaml_file" .yaml)/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$DEBUG_OUTPUT_DIR"
    WORKFLOW_CONTEXT["outputDir"]="$DEBUG_OUTPUT_DIR"
    
    log::info "Debug output directory: $DEBUG_OUTPUT_DIR"
    
    # Parse the workflow
    if ! workflow::parse_yaml "$yaml_file"; then
        return 1
    fi
    
    # Create browser session
    SESSION_ID="$session_name"
    
    if [[ "$USE_STATEFUL_SESSIONS" == "true" ]]; then
        log::info "Initializing stateful browser session: $SESSION_ID"
        if browser::init_stateful_session "$SESSION_ID"; then
            log::success "Stateful session initialized"
        else
            log::warn "Failed to initialize stateful session, falling back to stateless mode"
            USE_STATEFUL_SESSIONS=false
            if ! session::create "$SESSION_ID"; then
                log::error "Failed to create browser session"
                return 1
            fi
        fi
    else
        log::info "Creating browser session: $SESSION_ID"
        if ! session::create "$SESSION_ID"; then
            log::error "Failed to create browser session"
            return 1
        fi
    fi
    
    # Execute workflow steps
    local success=0
    CURRENT_STEP=0
    
    while [[ $CURRENT_STEP -lt ${#WORKFLOW_STEPS[@]} ]]; do
        local step="${WORKFLOW_STEPS[$CURRENT_STEP]}"
        local old_step=$CURRENT_STEP
        
        if ! workflow::execute_step "$step" "$CURRENT_STEP"; then
            log::error "Step $CURRENT_STEP failed"
            success=1
            break
        fi
        
        # Only increment if step counter wasn't changed by the action (e.g., jump_if)
        if [[ $CURRENT_STEP -eq $old_step ]]; then
            CURRENT_STEP=$((CURRENT_STEP + 1))
        fi
    done
    
    # Cleanup
    if [[ "$USE_STATEFUL_SESSIONS" == "true" ]]; then
        log::info "Stateful browser session cleanup complete: $SESSION_ID"
    else
        log::info "Destroying browser session: $SESSION_ID"
        session::destroy "$SESSION_ID"
    fi
    
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
        echo "Usage: $0 <workflow.yaml> [options]"
        echo "Options:"
        echo "  --param name=value    Set workflow parameter"
        echo "  --session name        Set session name"
        echo "  session_name          Set session name (backward compatibility)"
        echo ""
        echo "Examples:"
        echo "  $0 workflow.yaml --param n8n_url=http://localhost:5678 --param include_inactive=true"
        echo "  $0 workflow.yaml --session my_session"
        exit 1
    fi
    
    workflow::run "$@"
fi