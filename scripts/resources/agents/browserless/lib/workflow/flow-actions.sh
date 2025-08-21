#!/usr/bin/env bash

#######################################
# Enhanced Browserless Workflow Actions with Flow Control
# Implements new action types for jumping, branching, and sub-workflows
#######################################

# Get script directory
FLOW_ACTIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source original actions for base functionality
source "${FLOW_ACTIONS_DIR}/actions.sh"

#######################################
# Jump Control Actions
#######################################

# Unconditional jump to label or step
action::impl_jump_to() {
    local params="$1"
    local target=$(echo "$params" | jq -r '.target // .jump_to // ""')
    
    if [[ -z "$target" ]]; then
        log::error "jump_to action requires target parameter"
        return 1
    fi
    
    cat <<EOF
// Unconditional jump to: $target
return {
    type: 'flow_control',
    action: 'jump',
    target: '$target',
    timestamp: new Date().toISOString()
};
EOF
}

# Conditional jump based on page state
action::impl_jump_if() {
    local params="$1"
    local condition=$(echo "$params" | jq -r '.condition // ""')
    local success_target=$(echo "$params" | jq -r '.success_target // .jump_to // ""')
    local failure_target=$(echo "$params" | jq -r '.failure_target // .else_jump_to // ""')
    
    if [[ -z "$condition" ]] || [[ -z "$success_target" ]]; then
        log::error "jump_if action requires condition and success_target parameters"
        return 1
    fi
    
    cat <<EOF
// Conditional jump based on: $condition
try {
    const conditionResult = await page.evaluate(() => {
        return $condition;
    });
    
    if (conditionResult) {
        return {
            type: 'flow_control',
            action: 'jump',
            target: '$success_target',
            condition_result: true,
            timestamp: new Date().toISOString()
        };
    } else if ('$failure_target' !== '') {
        return {
            type: 'flow_control', 
            action: 'jump',
            target: '$failure_target',
            condition_result: false,
            timestamp: new Date().toISOString()
        };
    }
    
    // Continue to next step if no failure target
    return {
        type: 'continue',
        condition_result: false
    };
} catch (error) {
    console.error('Condition evaluation failed:', error);
    throw error;
}
EOF
}

# Multi-way branching based on expression
action::impl_switch() {
    local params="$1"
    local expression=$(echo "$params" | jq -r '.expression // ""')
    local cases=$(echo "$params" | jq -r '.cases // {}')
    local default_target=$(echo "$params" | jq -r '.default // ""')
    
    if [[ -z "$expression" ]] || [[ "$cases" == "{}" ]]; then
        log::error "switch action requires expression and cases parameters"
        return 1
    fi
    
    cat <<EOF
// Multi-way branch on expression: $expression
try {
    const switchValue = await page.evaluate(() => {
        return $expression;
    });
    
    const cases = $cases;
    const defaultTarget = '$default_target';
    
    // Check if we have a matching case
    if (cases[switchValue]) {
        return {
            type: 'flow_control',
            action: 'jump', 
            target: cases[switchValue],
            switch_value: switchValue,
            timestamp: new Date().toISOString()
        };
    } else if (defaultTarget !== '') {
        return {
            type: 'flow_control',
            action: 'jump',
            target: defaultTarget,
            switch_value: switchValue,
            default_case: true,
            timestamp: new Date().toISOString()
        };
    }
    
    // No match and no default - continue
    return {
        type: 'continue',
        switch_value: switchValue,
        no_match: true
    };
} catch (error) {
    console.error('Switch expression evaluation failed:', error);
    throw error;
}
EOF
}

#######################################
# Advanced Waiting with Flow Control
#######################################

# Wait for one of multiple conditions and branch accordingly
action::impl_wait_for_one_of() {
    local params="$1"
    local selectors=$(echo "$params" | jq -r '.selectors // {}')
    local max_wait=$(echo "$params" | jq -r '.max_wait // 10000')
    local on_match=$(echo "$params" | jq -r '.on_match // {}')
    local default_target=$(echo "$params" | jq -r '.default // ""')
    
    cat <<EOF
// Wait for one of multiple conditions
try {
    const selectors = $selectors;
    const onMatch = $on_match;
    const maxWait = $max_wait;
    
    const startTime = Date.now();
    const checkInterval = 500;
    
    while (Date.now() - startTime < maxWait) {
        for (const [condition, selector] of Object.entries(selectors)) {
            try {
                const element = await page.\$(selector);
                if (element) {
                    const target = onMatch[condition];
                    if (target) {
                        return {
                            type: 'flow_control',
                            action: 'jump',
                            target: target,
                            matched_condition: condition,
                            matched_selector: selector,
                            wait_time: Date.now() - startTime,
                            timestamp: new Date().toISOString()
                        };
                    }
                }
            } catch (e) {
                // Continue checking other conditions
            }
        }
        
        // Wait before next check
        await page.waitForTimeout(checkInterval);
    }
    
    // Timeout - use default target if specified
    const defaultTarget = '$default_target';
    if (defaultTarget !== '') {
        return {
            type: 'flow_control',
            action: 'jump',
            target: defaultTarget,
            timeout: true,
            wait_time: maxWait,
            timestamp: new Date().toISOString()
        };
    }
    
    throw new Error(\`wait_for_one_of timed out after \${maxWait}ms\`);
} catch (error) {
    console.error('wait_for_one_of failed:', error);
    throw error;
}
EOF
}

#######################################
# Sub-workflow Management
#######################################

# Call a sub-workflow and return to specified point
action::impl_call_sub_workflow() {
    local params="$1"
    local sub_workflow=$(echo "$params" | jq -r '.sub_workflow // .workflow // ""')
    local workflow_params=$(echo "$params" | jq -r '.params // {}')
    local return_target=$(echo "$params" | jq -r '.return_to // ""')
    
    if [[ -z "$sub_workflow" ]]; then
        log::error "call_sub_workflow action requires sub_workflow parameter"
        return 1
    fi
    
    cat <<EOF
// Call sub-workflow: $sub_workflow
try {
    const subWorkflowName = '$sub_workflow';
    const subParams = $workflow_params;
    const returnTarget = '$return_target';
    
    // Check if sub-workflow exists
    if (!context.subWorkflows || !context.subWorkflows[subWorkflowName]) {
        throw new Error(\`Sub-workflow not found: \${subWorkflowName}\`);
    }
    
    // Execute sub-workflow (this will be handled by the flow engine)
    return {
        type: 'flow_control',
        action: 'call_sub_workflow',
        sub_workflow: subWorkflowName,
        params: subParams,
        return_target: returnTarget,
        timestamp: new Date().toISOString()
    };
} catch (error) {
    console.error('Sub-workflow call failed:', error);
    throw error;
}
EOF
}

# Include another workflow inline (similar to call but no return)
action::impl_include_workflow() {
    local params="$1"
    local workflow_name=$(echo "$params" | jq -r '.workflow // ""')
    local workflow_params=$(echo "$params" | jq -r '.params // {}')
    
    if [[ -z "$workflow_name" ]]; then
        log::error "include_workflow action requires workflow parameter"
        return 1
    fi
    
    cat <<EOF
// Include workflow: $workflow_name
try {
    const workflowName = '$workflow_name';
    const workflowParams = $workflow_params;
    
    return {
        type: 'flow_control',
        action: 'include_workflow',
        workflow: workflowName,
        params: workflowParams,
        timestamp: new Date().toISOString()
    };
} catch (error) {
    console.error('Workflow inclusion failed:', error);
    throw error;
}
EOF
}

#######################################
# Enhanced Page Analysis for Flow Control
#######################################

# Enhanced evaluate action that can trigger flow control
action::impl_evaluate_with_flow() {
    local params="$1"
    local script=$(echo "$params" | jq -r '.script // ""')
    local output=$(echo "$params" | jq -r '.output // ""')
    local flow_control=$(echo "$params" | jq -r '.flow_control // {}')
    
    if [[ -z "$script" ]]; then
        log::error "evaluate_with_flow action requires script parameter"
        return 1
    fi
    
    cat <<EOF
// Enhanced evaluate with flow control
try {
    const result = await page.evaluate(() => {
        $script
    });
    
    // Store result if output specified
    const output = '$output';
    if (output !== '') {
        outputs[output] = result;
    }
    
    // Check flow control conditions
    const flowControl = $flow_control;
    if (flowControl && Array.isArray(flowControl)) {
        for (const rule of flowControl) {
            if (rule.condition) {
                const conditionMet = await page.evaluate((r, res) => {
                    // Replace \${result.xxx} patterns in condition
                    const condition = r.condition.replace(/\\\${result\.([^}]+)}/g, (match, prop) => {
                        return res[prop];
                    });
                    return eval(condition);
                }, rule, result);
                
                if (conditionMet && rule.jump_to) {
                    return {
                        type: 'flow_control',
                        action: 'jump',
                        target: rule.jump_to,
                        evaluation_result: result,
                        condition_matched: rule.condition,
                        timestamp: new Date().toISOString()
                    };
                }
            }
        }
        
        // Check for default jump
        const defaultRule = flowControl.find(r => r.default);
        if (defaultRule && defaultRule.jump_to) {
            return {
                type: 'flow_control',
                action: 'jump',
                target: defaultRule.jump_to,
                evaluation_result: result,
                default_case: true,
                timestamp: new Date().toISOString()
            };
        }
    }
    
    return {
        type: 'continue',
        evaluation_result: result
    };
} catch (error) {
    console.error('Enhanced evaluation failed:', error);
    throw error;
}
EOF
}

#######################################
# State Management Actions
#######################################

# Set workflow state for complex flows
action::impl_set_state() {
    local params="$1"
    local key=$(echo "$params" | jq -r '.key // ""')
    local value=$(echo "$params" | jq -r '.value // ""')
    
    if [[ -z "$key" ]]; then
        log::error "set_state action requires key parameter"
        return 1
    fi
    
    cat <<EOF
// Set workflow state: $key
try {
    const stateKey = '$key';
    const stateValue = '$value';
    
    if (!context.state) {
        context.state = {};
    }
    
    context.state[stateKey] = stateValue;
    
    return {
        type: 'continue',
        state_updated: { [stateKey]: stateValue }
    };
} catch (error) {
    console.error('State setting failed:', error);
    throw error;
}
EOF
}

# Get workflow state
action::impl_get_state() {
    local params="$1"
    local key=$(echo "$params" | jq -r '.key // ""')
    local output=$(echo "$params" | jq -r '.output // ""')
    
    cat <<EOF
// Get workflow state: $key
try {
    const stateKey = '$key';
    const outputKey = '$output';
    
    const stateValue = context.state ? context.state[stateKey] : null;
    
    if (outputKey !== '') {
        outputs[outputKey] = stateValue;
    }
    
    return {
        type: 'continue',
        state_value: stateValue
    };
} catch (error) {
    console.error('State retrieval failed:', error);
    throw error;
}
EOF
}

#######################################
# Enhanced action implementation dispatcher
#######################################
action::get_flow_control_implementation() {
    local action_name="${1:?Action name required}"
    local params="${2:-{}}"
    
    case "$action_name" in
        # Jump control
        "jump_to"|"goto")
            action::impl_jump_to "$params"
            ;;
        "jump_if"|"conditional_jump")
            action::impl_jump_if "$params"
            ;;
        "switch"|"branch"|"route")
            action::impl_switch "$params"
            ;;
            
        # Advanced waiting
        "wait_for_one_of")
            action::impl_wait_for_one_of "$params"
            ;;
            
        # Sub-workflows
        "call_sub_workflow"|"call_workflow")
            action::impl_call_sub_workflow "$params"
            ;;
        "include_workflow")
            action::impl_include_workflow "$params"
            ;;
            
        # Enhanced evaluation
        "evaluate_with_flow")
            action::impl_evaluate_with_flow "$params"
            ;;
            
        # State management
        "set_state")
            action::impl_set_state "$params"
            ;;
        "get_state")
            action::impl_get_state "$params"
            ;;
            
        *)
            # Fall back to original action implementation
            action::get_implementation "$action_name" "$params"
            ;;
    esac
}

# Export functions
export -f action::impl_jump_to
export -f action::impl_jump_if  
export -f action::impl_switch
export -f action::impl_wait_for_one_of
export -f action::impl_call_sub_workflow
export -f action::impl_include_workflow
export -f action::impl_evaluate_with_flow
export -f action::impl_set_state
export -f action::impl_get_state
export -f action::get_flow_control_implementation