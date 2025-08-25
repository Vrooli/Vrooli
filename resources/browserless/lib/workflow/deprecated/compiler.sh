#!/usr/bin/env bash

#######################################
# Browserless Workflow Compiler
# Compiles workflow definitions into executable JavaScript functions
#######################################

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"
WORKFLOW_COMPILER_DIR="${APP_ROOT}/resources/browserless/lib/workflow/deprecated"

# Source dependencies
source "${WORKFLOW_COMPILER_DIR}/parser.sh"
source "${WORKFLOW_COMPILER_DIR}/actions.sh"
source "${WORKFLOW_COMPILER_DIR}/debug.sh"

# Source enhanced flow control components if available
if [[ -f "${WORKFLOW_COMPILER_DIR}/flow-compiler.sh" ]]; then
    source "${WORKFLOW_COMPILER_DIR}/flow-compiler.sh"
    FLOW_CONTROL_AVAILABLE=true
else
    FLOW_CONTROL_AVAILABLE=false
fi

#######################################
# Compile workflow to JavaScript function
# Arguments:
#   $1 - Workflow JSON
# Returns:
#   JavaScript function code
#######################################
workflow::compile() {
    local workflow_json="${1:?Workflow JSON required}"
    
    # Check if workflow uses flow control
    local flow_enabled
    flow_enabled=$(echo "$workflow_json" | jq -r '.workflow.flow_control.enabled // false')
    
    # Use enhanced compiler if flow control is enabled and available
    if [[ "$flow_enabled" == "true" ]] && [[ "$FLOW_CONTROL_AVAILABLE" == "true" ]]; then
        log::info "🔄 Using enhanced flow control compiler"
        workflow::compile_with_flow_control "$workflow_json"
        return $?
    fi
    
    # Fall back to original linear compilation
    log::info "📝 Using linear workflow compiler"
    
    # Simplify workflow for compilation
    local simplified
    simplified=$(workflow::simplify "$workflow_json")
    
    local workflow_name=$(echo "$simplified" | jq -r '.name')
    local debug_level=$(echo "$simplified" | jq -r '.debug_level')
    local steps=$(echo "$simplified" | jq -r '.steps')
    
    # Generate function header
    cat <<EOF
export default async ({ page, params, context }) => {
    // Workflow: $workflow_name
    const results = [];
    const outputs = {};
    const debug = new WorkflowDebugger('$workflow_name', '$debug_level', context);
    
    try {
        // Initialize debug
        await debug.init(page);
        
EOF
    
    # Compile each step
    local step_count
    step_count=$(echo "$steps" | jq 'length')
    
    local step_index=0
    while [[ $step_index -lt $step_count ]]; do
        local step
        step=$(echo "$steps" | jq ".[$step_index]")
        
        local step_name=$(echo "$step" | jq -r '.name')
        local step_action=$(echo "$step" | jq -r '.action')
        local step_params=$(echo "$step" | jq -r '.params')
        local step_debug=$(echo "$step" | jq -r '.debug')
        local step_on_error=$(echo "$step" | jq -r '.on_error')
        
        # Generate step code
        cat <<EOF
        // Step $((step_index + 1)): $step_name
        await debug.startStep('$step_name', '$step_action');
        try {
$(action::get_implementation "$step_action" "$step_params" | sed 's/^/            /')
            await debug.endStep('$step_name', true, results);
        } catch (stepError) {
            await debug.endStep('$step_name', false, stepError);
EOF
        
        # Handle error strategy
        case "$step_on_error" in
            "retry")
                cat <<EOF
            // Retry once on error
            console.log('Retrying step: $step_name');
            try {
$(action::get_implementation "$step_action" "$step_params" | sed 's/^/                /')
                await debug.endStep('$step_name', true, results);
            } catch (retryError) {
                throw new Error(\`Step '$step_name' failed after retry: \${retryError.message}\`);
            }
EOF
                ;;
            "continue")
                cat <<EOF
            // Continue on error
            console.warn(\`Step '$step_name' failed: \${stepError.message}\`);
            results.push({
                step: '$step_name',
                error: stepError.message,
                continued: true
            });
EOF
                ;;
            *)  # "fail" or default
                cat <<EOF
            throw new Error(\`Step '$step_name' failed: \${stepError.message}\`);
EOF
                ;;
        esac
        
        cat <<EOF
        }
        
        // Collect debug data if configured
$(workflow::compile_debug_collection "$step_debug" "$step_name" | sed 's/^/        /')
        
EOF
        
        step_index=$((step_index + 1))
    done
    
    # Generate function footer
    cat <<EOF
        // Workflow completed successfully
        return {
            success: true,
            workflow: '$workflow_name',
            steps_completed: $step_count,
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
            error: error.message,
            failed_at_step: debugData.last_step,
            results: results,
            outputs: outputs,
            debug: debugData,
            timestamp: new Date().toISOString()
        };
    }
};

// Workflow Debug Helper Class
class WorkflowDebugger {
    constructor(name, level, context) {
        this.name = name;
        this.level = level;
        this.context = context;
        this.steps = [];
        this.currentStep = null;
        this.page = null;
    }
    
    async init(page) {
        this.page = page;
        
        // Set up console log collection if verbose
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
    
    async getData() {
        return {
            workflow: this.name,
            level: this.level,
            steps: this.steps,
            console: this.consoleLogs || [],
            network: this.networkRequests || [],
            last_step: this.last_step
        };
    }
}
EOF
}

#######################################
# Compile debug collection code for a step
# Arguments:
#   $1 - Debug configuration JSON
#   $2 - Step name
# Returns:
#   JavaScript code for debug collection
#######################################
workflow::compile_debug_collection() {
    local debug_config="${1:-{}}"
    local step_name="$2"
    
    local code=""
    
    # Check if screenshot is requested
    local screenshot=$(echo "$debug_config" | jq -r '.screenshot // false')
    if [[ "$screenshot" == "true" ]]; then
        code="${code}
await debug.captureScreenshot('$step_name');"
    fi
    
    # Check if console logs are requested
    local console=$(echo "$debug_config" | jq -r '.console // false')
    if [[ "$console" == "true" ]]; then
        code="${code}
await debug.captureConsole('$step_name');"
    fi
    
    # Check if network is requested
    local network=$(echo "$debug_config" | jq -r '.network // false')
    if [[ "$network" == "true" ]]; then
        code="${code}
await debug.captureNetwork('$step_name');"
    fi
    
    # Check if HTML is requested
    local html=$(echo "$debug_config" | jq -r '.html // false')
    if [[ "$html" == "true" ]]; then
        code="${code}
const html_$step_name = await page.content();
outputs['${step_name}_html'] = html_$step_name;"
    fi
    
    if [[ -z "$code" ]]; then
        echo "// No debug collection for this step"
    else
        echo "$code"
    fi
}

#######################################
# Compile and store workflow
# Arguments:
#   $1 - Workflow file path
#   $2 - Output directory (optional)
# Returns:
#   0 on success, 1 on failure
#######################################
workflow::compile_and_store() {
    local workflow_file="${1:?Workflow file required}"
    local output_dir="${2:-${BROWSERLESS_DATA_DIR}/workflows}"
    
    log::header "📦 Compiling Workflow: $workflow_file"
    
    # Parse workflow - check for flow control first
    local workflow_json
    
    # Try enhanced parsing if flow control is available
    if [[ "$FLOW_CONTROL_AVAILABLE" == "true" ]]; then
        # Check if this workflow might use flow control
        if grep -q "flow_control:\|jump_to:\|label:\|switch:\|call_workflow:" "$workflow_file" 2>/dev/null; then
            log::info "🔍 Detected flow control features, using enhanced parser"
            workflow_json=$(workflow::parse_with_flow_control "$workflow_file")
        else
            workflow_json=$(workflow::parse "$workflow_file")
        fi
    else
        workflow_json=$(workflow::parse "$workflow_file")
    fi
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to parse workflow"
        return 1
    fi
    
    # Extract metadata
    local metadata
    metadata=$(workflow::extract_metadata "$workflow_json")
    
    local workflow_name=$(echo "$metadata" | jq -r '.name')
    log::info "Workflow name: $workflow_name"
    
    # Create output directory
    local workflow_dir="${output_dir}/${workflow_name}"
    mkdir -p "${workflow_dir}/executions"
    
    # Store original workflow
    echo "$workflow_json" > "${workflow_dir}/workflow.json"
    
    # Compile to JavaScript
    log::info "Compiling to JavaScript..."
    local compiled_js
    compiled_js=$(workflow::compile "$workflow_json")
    
    # Store compiled function
    echo "$compiled_js" > "${workflow_dir}/compiled.js"
    
    # Store metadata
    echo "$metadata" > "${workflow_dir}/metadata.json"
    
    # Create injection manifest for browserless
    local injection_manifest
    injection_manifest=$(jq -n \
        --arg name "$workflow_name" \
        --arg desc "$(echo "$metadata" | jq -r '.description')" \
        --arg code "$compiled_js" \
        '{
            metadata: {
                name: $name,
                description: $desc,
                version: "1.0.0",
                type: "workflow",
                created: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
            },
            function: {
                code: $code,
                timeout: 60000
            },
            execution: {
                persistent_session: true
            }
        }')
    
    echo "$injection_manifest" > "${workflow_dir}/manifest.json"
    
    log::success "✅ Workflow compiled successfully"
    log::info "📁 Stored in: $workflow_dir"
    log::info "🚀 Execute with: resource-browserless workflow run $workflow_name"
    
    return 0
}

# Export functions
export -f workflow::compile
export -f workflow::compile_debug_collection
export -f workflow::compile_and_store