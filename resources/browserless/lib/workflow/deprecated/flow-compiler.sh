#!/usr/bin/env bash

#######################################
# Enhanced Browserless Workflow Compiler with Flow Control
# Generates JavaScript state machines instead of linear execution
#######################################

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../../../.." && builtin pwd)}"
FLOW_COMPILER_DIR="${APP_ROOT}/resources/browserless/lib/workflow/deprecated"
BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"

# Source common utilities for logging
source "${BROWSERLESS_LIB_DIR}/common.sh" 2>/dev/null || {
    # Fallback log functions if common.sh not available
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::header() { echo "=== $* ==="; }
}

# Source dependencies
source "${FLOW_COMPILER_DIR}/flow-parser.sh"
source "${FLOW_COMPILER_DIR}/flow-actions.sh"
source "${FLOW_COMPILER_DIR}/debug.sh"

# Note: Original compiler.sh is sourced in test script to avoid circular dependency

#######################################
# Enhanced workflow compilation with flow control
# Arguments:
#   $1 - Enhanced workflow JSON with flow control graph
# Returns:
#   JavaScript state machine code
#######################################
workflow::compile_with_flow_control() {
    local workflow_json="${1:?Enhanced workflow JSON required}"
    
    # Check if flow control is enabled
    local flow_enabled
    echo "[COMPILER-DEBUG] Testing flow_enabled extraction..." >&2
    flow_enabled=$(echo "$workflow_json" | jq -r '.flow_control.enabled // false' 2>&1)
    local jq_exit=$?
    echo "[COMPILER-DEBUG] flow_enabled result: '$flow_enabled', exit: $jq_exit" >&2
    
    if [[ $jq_exit -ne 0 ]]; then
        echo "[COMPILER-DEBUG] jq failed on flow_enabled, input preview:" >&2
        echo "$workflow_json" | head -c 200 >&2
        echo "" >&2
        return 1
    fi
    
    if [[ "$flow_enabled" != "true" ]]; then
        # Fall back to original linear compilation
        workflow::compile "$workflow_json"
        return $?
    fi
    
    # Extract flow control data
    local flow_graph
    echo "[COMPILER-DEBUG] Testing flow_graph extraction..." >&2
    flow_graph=$(echo "$workflow_json" | jq '.flow_control.graph' 2>&1)
    jq_exit=$?
    echo "[COMPILER-DEBUG] flow_graph exit: $jq_exit" >&2
    if [[ $jq_exit -ne 0 ]]; then
        echo "[COMPILER-DEBUG] flow_graph extraction failed: $flow_graph" >&2
        return 1
    fi
    
    echo "[COMPILER-DEBUG] Testing workflow_name extraction..." >&2
    local workflow_name=$(echo "$workflow_json" | jq -r '.workflow.name' 2>&1)
    jq_exit=$?
    echo "[COMPILER-DEBUG] workflow_name: '$workflow_name', exit: $jq_exit" >&2
    if [[ $jq_exit -ne 0 ]]; then
        echo "[COMPILER-DEBUG] workflow_name extraction failed: $workflow_name" >&2
        return 1
    fi
    
    echo "[COMPILER-DEBUG] Testing debug_level extraction..." >&2
    local debug_level=$(echo "$workflow_json" | jq -r '.workflow.debug_level' 2>&1)
    jq_exit=$?
    echo "[COMPILER-DEBUG] debug_level: '$debug_level', exit: $jq_exit" >&2
    if [[ $jq_exit -ne 0 ]]; then
        echo "[COMPILER-DEBUG] debug_level extraction failed: $debug_level" >&2
        return 1
    fi
    
    echo "[COMPILER-DEBUG] Testing sub_workflows extraction..." >&2
    local sub_workflows=$(echo "$workflow_json" | jq '.flow_control.graph.sub_workflows // {}' 2>&1)
    jq_exit=$?
    echo "[COMPILER-DEBUG] sub_workflows exit: $jq_exit" >&2
    if [[ $jq_exit -ne 0 ]]; then
        echo "[COMPILER-DEBUG] sub_workflows extraction failed: $sub_workflows" >&2
        return 1
    fi
    
    # Generate state machine JavaScript
    workflow::generate_state_machine "$workflow_name" "$debug_level" "$flow_graph" "$sub_workflows"
}

#######################################
# Generate JavaScript state machine
# Arguments:
#   $1 - Workflow name
#   $2 - Debug level
#   $3 - Flow control graph JSON
#   $4 - Sub-workflows JSON
# Returns:
#   Complete JavaScript state machine
#######################################
workflow::generate_state_machine() {
    local workflow_name="$1"
    local debug_level="$2"
    local flow_graph="$3"
    local sub_workflows="$4"
    
    # Extract entry point
    local entry_point
    entry_point=$(echo "$flow_graph" | jq -r '.entry_point')
    
    # Generate function header
    cat <<EOF
export default async ({ page, params, context }) => {
    // Enhanced Workflow State Machine: $workflow_name
    const results = [];
    const outputs = {};
    const debug = new WorkflowDebugger('$workflow_name', '$debug_level', context);
    
    // Flow control state
    let currentStep = '$entry_point';
    const visitedSteps = new Set();
    const stepResults = new Map();
    const maxIterations = 1000; // Prevent infinite loops
    let iterations = 0;
    
    // Sub-workflows registry
    context.subWorkflows = $sub_workflows;
    
    // Flow control graph
    const flowGraph = $flow_graph;
    const labels = flowGraph.labels || {};
    const jumps = flowGraph.jumps || {};
    
    try {
        // Initialize debug
        await debug.init(page);
        
        // State machine execution loop
        while (currentStep && iterations < maxIterations) {
            iterations++;
            
            // Check for infinite loop prevention
            const stepKey = \`\${currentStep}_\${iterations}\`;
            if (visitedSteps.has(currentStep) && iterations > 100) {
                console.warn(\`Step '\${currentStep}' visited multiple times, checking for infinite loop...\`);
            }
            visitedSteps.add(stepKey);
            
            // Get step definition
            const stepDef = flowGraph.steps[currentStep];
            if (!stepDef) {
                throw new Error(\`Step not found in flow graph: \${currentStep}\`);
            }
            
            console.log(\`Executing step: \${currentStep} (iteration \${iterations})\`);
            
            // Execute step
            const stepResult = await executeStep(stepDef, currentStep);
            stepResults.set(currentStep, stepResult);
            results.push(stepResult);
            
            // Determine next step based on result
            currentStep = determineNextStep(stepResult, currentStep, stepDef);
        }
        
        if (iterations >= maxIterations) {
            throw new Error(\`Workflow exceeded maximum iterations (\${maxIterations}) - possible infinite loop\`);
        }
        
        // Workflow completed successfully
        return {
            success: true,
            workflow: '$workflow_name',
            execution_model: 'state_machine',
            steps_executed: results.length,
            iterations: iterations,
            results: results,
            outputs: outputs,
            debug: await debug.getData(),
            timestamp: new Date().toISOString()
        };
        
    } catch (error) {
        // Workflow failed
        const debugData = await debug.getData();
        return {
            success: false,
            workflow: '$workflow_name',
            execution_model: 'state_machine',
            error: error.message,
            failed_at_step: currentStep,
            iterations: iterations,
            results: results,
            outputs: outputs,
            debug: debugData,
            timestamp: new Date().toISOString()
        };
    }
    
    // Step execution function
    async function executeStep(stepDef, stepName) {
        await debug.startStep(stepName, stepDef.action);
        
        try {
$(workflow::generate_step_execution_logic | sed 's/^/            /')
        } catch (stepError) {
            await debug.endStep(stepName, false, stepError);
            
            // Handle error based on step configuration
            const errorHandling = stepDef.on_error || 'fail';
            
            switch (errorHandling) {
                case 'continue':
                    console.warn(\`Step '\${stepName}' failed but continuing: \${stepError.message}\`);
                    return {
                        step: stepName,
                        action: stepDef.action,
                        success: false,
                        error: stepError.message,
                        continued: true,
                        type: 'continue',
                        timestamp: new Date().toISOString()
                    };
                    
                case 'retry':
                    console.log(\`Retrying step: \${stepName}\`);
                    try {
                        // Retry the step once
                        const retryResult = await executeStepAction(stepDef, stepName);
                        await debug.endStep(stepName, true, retryResult);
                        return retryResult;
                    } catch (retryError) {
                        await debug.endStep(stepName, false, retryError);
                        throw new Error(\`Step '\${stepName}' failed after retry: \${retryError.message}\`);
                    }
                    
                default: // 'fail'
                    throw stepError;
            }
        }
    }
    
    // Step action execution
    async function executeStepAction(stepDef, stepName) {
$(workflow::generate_action_dispatcher | sed 's/^/        /')
    }
    
    // Next step determination logic
    function determineNextStep(stepResult, currentStepName, stepDef) {
        // Check if step result contains flow control directive
        if (stepResult.type === 'flow_control' && stepResult.action === 'jump') {
            const target = stepResult.target;
            
            // Resolve label to step name if needed
            if (labels[target]) {
                return labels[target];
            }
            
            // Check if target exists as step
            if (flowGraph.steps[target]) {
                return target;
            }
            
            throw new Error(\`Jump target not found: \${target}\`);
        }
        
        // Check if step result is a sub-workflow call
        if (stepResult.type === 'flow_control' && stepResult.action === 'call_sub_workflow') {
            // TODO: Implement sub-workflow execution
            console.warn('Sub-workflow execution not yet implemented');
            return null;
        }
        
        // Check step-level jump configuration
        const stepFlowControl = stepDef.flow_control || {};
        
        if (stepResult.success && stepFlowControl.jump_to) {
            const target = stepFlowControl.jump_to;
            return labels[target] || target;
        }
        
        // Check jumps defined in flow graph
        const stepJumps = jumps[currentStepName];
        if (stepJumps && stepJumps.length > 0) {
            // For now, take first jump target (can be enhanced with conditions)
            const target = stepJumps[0];
            return labels[target] || target;
        }
        
        // Default: find next step in execution order
        const executionOrder = flowGraph.execution_order || [];
        const currentIndex = executionOrder.indexOf(currentStepName);
        
        if (currentIndex >= 0 && currentIndex < executionOrder.length - 1) {
            return executionOrder[currentIndex + 1];
        }
        
        // No more steps
        return null;
    }
};

// Enhanced Workflow Debugger with State Machine Support
class WorkflowDebugger {
    constructor(name, level, context) {
        this.name = name;
        this.level = level;
        this.context = context;
        this.steps = [];
        this.currentStep = null;
        this.page = null;
        this.stateTransitions = [];
    }
    
    async init(page) {
        this.page = page;
        
        // Set up console log collection if enabled
        if (this.level === 'verbose' || this.level === 'steps') {
            this.consoleLogs = [];
            page.on('console', msg => {
                this.consoleLogs.push({
                    type: msg.type(),
                    text: msg.text(),
                    timestamp: new Date().toISOString()
                });
            });
        }
        
        // Set up network monitoring if verbose
        if (this.level === 'verbose') {
            this.networkRequests = [];
            page.on('request', request => {
                this.networkRequests.push({
                    url: request.url(),
                    method: request.method(),
                    timestamp: new Date().toISOString()
                });
            });
        }
    }
    
    async startStep(name, action) {
        this.currentStep = {
            name: name,
            action: action,
            startTime: Date.now(),
            debug: {}
        };
    }
    
    async endStep(name, success, data) {
        if (this.currentStep && this.currentStep.name === name) {
            this.currentStep.endTime = Date.now();
            this.currentStep.duration = this.currentStep.endTime - this.currentStep.startTime;
            this.currentStep.success = success;
            
            if (!success && data) {
                this.currentStep.error = data.message || data;
            } else if (success && data) {
                this.currentStep.result = data;
            }
            
            // Capture screenshot if configured
            if (this.level === 'steps' || this.level === 'verbose') {
                try {
                    const screenshotPath = \`\${this.context.outputDir}/step-\${this.steps.length + 1}-\${name}.png\`;
                    await this.page.screenshot({ path: screenshotPath });
                    this.currentStep.debug.screenshot = screenshotPath;
                } catch (e) {
                    // Screenshot failed, continue
                }
            }
            
            this.steps.push(this.currentStep);
            this.last_step = name;
            this.currentStep = null;
        }
    }
    
    logStateTransition(from, to, reason) {
        this.stateTransitions.push({
            from: from,
            to: to,
            reason: reason,
            timestamp: new Date().toISOString()
        });
    }
    
    async getData() {
        return {
            workflow: this.name,
            execution_model: 'state_machine',
            level: this.level,
            steps: this.steps,
            state_transitions: this.stateTransitions,
            console: this.consoleLogs || [],
            network: this.networkRequests || [],
            last_step: this.last_step
        };
    }
}
EOF
}

#######################################
# Generate step execution logic for state machine
# Returns:
#   JavaScript code for step execution
#######################################
workflow::generate_step_execution_logic() {
    cat <<'EOF'
// Execute the step action
const actionResult = await executeStepAction(stepDef, stepName);

// Collect debug data if configured
await collectDebugData(stepDef, stepName, actionResult);

await debug.endStep(stepName, true, actionResult);

return {
    step: stepName,
    action: stepDef.action,
    success: true,
    result: actionResult,
    type: actionResult.type || 'continue',
    timestamp: new Date().toISOString()
};
EOF
}

#######################################
# Generate action dispatcher for state machine
# Returns:
#   JavaScript code for action dispatching
#######################################
workflow::generate_action_dispatcher() {
    cat <<'EOF'
const action = stepDef.action;
const params = stepDef.params || {};

// Enhanced action execution with flow control support
switch (action) {
EOF
    
    # Generate cases for each flow control action type
    local flow_actions=(
        "jump_to" "jump_if" "switch" "wait_for_one_of"
        "call_sub_workflow" "include_workflow" "evaluate_with_flow"
        "set_state" "get_state"
    )
    
    for action in "${flow_actions[@]}"; do
        echo "    case '$action':"
        echo "        return await execute_${action}(params, stepName);"
        echo ""
    done
    
    cat <<'EOF'
    default:
        // Fall back to original action implementation
        return await executeOriginalAction(action, params, stepName);
}

// Flow control action implementations
EOF

    # Generate individual action implementations
    workflow::generate_flow_action_implementations
}

#######################################
# Generate flow control action implementations
# Returns:
#   JavaScript implementations of flow control actions
#######################################
workflow::generate_flow_action_implementations() {
    cat <<'EOF'
async function execute_jump_to(params, stepName) {
    const target = params.target || params.jump_to;
    if (!target) {
        throw new Error(`jump_to action requires target parameter`);
    }
    
    return {
        type: 'flow_control',
        action: 'jump',
        target: target,
        timestamp: new Date().toISOString()
    };
}

async function execute_jump_if(params, stepName) {
    const condition = params.condition;
    const successTarget = params.success_target || params.jump_to;
    const failureTarget = params.failure_target || params.else_jump_to;
    
    if (!condition || !successTarget) {
        throw new Error(`jump_if action requires condition and success_target parameters`);
    }
    
    try {
        const conditionResult = await page.evaluate((cond) => {
            return eval(cond);
        }, condition);
        
        if (conditionResult) {
            return {
                type: 'flow_control',
                action: 'jump',
                target: successTarget,
                condition_result: true,
                timestamp: new Date().toISOString()
            };
        } else if (failureTarget) {
            return {
                type: 'flow_control',
                action: 'jump', 
                target: failureTarget,
                condition_result: false,
                timestamp: new Date().toISOString()
            };
        }
        
        return {
            type: 'continue',
            condition_result: false
        };
    } catch (error) {
        console.error('Condition evaluation failed:', error);
        throw error;
    }
}

async function execute_switch(params, stepName) {
    const expression = params.expression;
    const cases = params.cases || {};
    const defaultTarget = params.default;
    
    if (!expression || Object.keys(cases).length === 0) {
        throw new Error(`switch action requires expression and cases parameters`);
    }
    
    try {
        const switchValue = await page.evaluate((expr) => {
            return eval(expr);
        }, expression);
        
        // Check for matching case
        if (cases[switchValue]) {
            return {
                type: 'flow_control',
                action: 'jump',
                target: cases[switchValue],
                switch_value: switchValue,
                timestamp: new Date().toISOString()
            };
        } else if (defaultTarget) {
            return {
                type: 'flow_control',
                action: 'jump',
                target: defaultTarget,
                switch_value: switchValue,
                default_case: true,
                timestamp: new Date().toISOString()
            };
        }
        
        return {
            type: 'continue',
            switch_value: switchValue,
            no_match: true
        };
    } catch (error) {
        console.error('Switch expression evaluation failed:', error);
        throw error;
    }
}

async function execute_wait_for_one_of(params, stepName) {
    const selectors = params.selectors || {};
    const maxWait = params.max_wait || 10000;
    const onMatch = params.on_match || {};
    const defaultTarget = params.default;
    
    const startTime = Date.now();
    const checkInterval = 500;
    
    while (Date.now() - startTime < maxWait) {
        for (const [condition, selector] of Object.entries(selectors)) {
            try {
                const element = await page.$(selector);
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
        
        await page.waitForTimeout(checkInterval);
    }
    
    // Timeout - use default target if specified
    if (defaultTarget) {
        return {
            type: 'flow_control',
            action: 'jump',
            target: defaultTarget,
            timeout: true,
            wait_time: maxWait,
            timestamp: new Date().toISOString()
        };
    }
    
    throw new Error(`wait_for_one_of timed out after ${maxWait}ms`);
}

async function execute_call_sub_workflow(params, stepName) {
    const subWorkflowName = params.sub_workflow || params.workflow;
    const subParams = params.params || {};
    const returnTarget = params.return_to;
    
    if (!subWorkflowName) {
        throw new Error(`call_sub_workflow action requires sub_workflow parameter`);
    }
    
    // Check if sub-workflow exists
    if (!context.subWorkflows || !context.subWorkflows[subWorkflowName]) {
        throw new Error(`Sub-workflow not found: ${subWorkflowName}`);
    }
    
    return {
        type: 'flow_control',
        action: 'call_sub_workflow',
        sub_workflow: subWorkflowName,
        params: subParams,
        return_target: returnTarget,
        timestamp: new Date().toISOString()
    };
}

async function executeOriginalAction(action, params, stepName) {
    // This would integrate with the original action system
    // For now, return a placeholder
    return {
        type: 'continue',
        action: action,
        params: params,
        executed: true,
        timestamp: new Date().toISOString()
    };
}

async function collectDebugData(stepDef, stepName, actionResult) {
    const stepDebug = stepDef.debug || {};
    
    // Collect debug data based on step configuration
    if (stepDebug.screenshot === true) {
        try {
            const screenshotPath = `${context.outputDir}/debug-${stepName}-${Date.now()}.png`;
            await page.screenshot({ path: screenshotPath });
        } catch (e) {
            console.warn(`Screenshot failed for step ${stepName}:`, e.message);
        }
    }
}
EOF
}

#######################################
# Enhanced compile and store with flow control
# Arguments:
#   $1 - Workflow file path
#   $2 - Output directory (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
workflow::compile_and_store_with_flow_control() {
    local workflow_file="${1:?Workflow file required}"
    
    # Set default output directory if BROWSERLESS_DATA_DIR not defined
    local default_output_dir="${BROWSERLESS_DATA_DIR:-/tmp/browserless-data}/workflows"
    local output_dir="${2:-$default_output_dir}"
    
    # Ensure log functions are available
    if ! declare -f log::header >/dev/null 2>&1; then
        log::header() { echo "=== $* ==="; }
        log::info() { echo "[INFO] $*"; }
        log::error() { echo "[ERROR] $*" >&2; }
        log::success() { echo "[SUCCESS] $*"; }
    fi
    
    log::header "🚀 Compiling Enhanced Workflow: $workflow_file"
    
    # Parse with flow control support
    local workflow_json
    workflow_json=$(workflow::parse_with_flow_control "$workflow_file")
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to parse workflow with flow control"
        return 1
    fi
    
    # Extract metadata
    local metadata
    metadata=$(workflow::extract_metadata "$workflow_json")
    
    local workflow_name=$(echo "$metadata" | jq -r '.name')
    log::info "Enhanced workflow name: $workflow_name"
    
    # Create output directory
    local workflow_dir="${output_dir}/${workflow_name}"
    mkdir -p "${workflow_dir}/executions"
    
    # Store original workflow with flow control
    echo "$workflow_json" > "${workflow_dir}/workflow.json"
    
    # Compile to enhanced JavaScript
    log::info "Compiling to enhanced JavaScript state machine..."
    local compiled_js
    compiled_js=$(workflow::compile_with_flow_control "$workflow_json")
    
    # Store compiled function
    echo "$compiled_js" > "${workflow_dir}/compiled.js"
    
    # Enhanced metadata with flow control info
    local flow_enabled
    flow_enabled=$(echo "$workflow_json" | jq -r '.flow_control.enabled // false')
    
    local enhanced_metadata
    enhanced_metadata=$(echo "$metadata" | jq --arg flow "$flow_enabled" '. + {
        flow_control_enabled: ($flow == "true"),
        execution_model: (if ($flow == "true") then "state_machine" else "linear" end)
    }')
    
    echo "$enhanced_metadata" > "${workflow_dir}/metadata.json"
    
    # Create injection manifest
    local injection_manifest
    injection_manifest=$(jq -n \
        --arg name "$workflow_name" \
        --arg desc "$(echo "$enhanced_metadata" | jq -r '.description')" \
        --arg code "$compiled_js" \
        --arg model "$(echo "$enhanced_metadata" | jq -r '.execution_model')" \
        '{
            metadata: {
                name: $name,
                description: $desc,
                version: "2.0.0",
                type: "enhanced_workflow",
                execution_model: $model,
                created: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
            },
            function: {
                code: $code,
                timeout: 120000
            },
            execution: {
                persistent_session: true,
                flow_control: true
            }
        }')
    
    echo "$injection_manifest" > "${workflow_dir}/manifest.json"
    
    log::success "✅ Enhanced workflow compiled successfully"
    log::info "🎯 Execution model: $(echo "$enhanced_metadata" | jq -r '.execution_model')"
    log::info "📁 Stored in: $workflow_dir"
    log::info "🚀 Execute with: resource-browserless workflow run $workflow_name"
    
    return 0
}

# Export functions
export -f workflow::compile_with_flow_control
export -f workflow::generate_state_machine
export -f workflow::generate_step_execution_logic
export -f workflow::generate_action_dispatcher
export -f workflow::generate_flow_action_implementations
export -f workflow::compile_and_store_with_flow_control