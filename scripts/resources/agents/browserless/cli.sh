#!/usr/bin/env bash
################################################################################
# Browserless Resource CLI
# 
# Lightweight CLI interface for Browserless using the CLI Command Framework
#
# Usage:
#   resource-browserless <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    BROWSERLESS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    BROWSERLESS_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
BROWSERLESS_CLI_DIR="$(cd "$(dirname "$BROWSERLESS_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source browserless configuration
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
browserless::export_config 2>/dev/null || true

# Source browserless libraries
for lib in core docker health status api inject usage recovery; do
    lib_file="${BROWSERLESS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Source adapter framework if available
if [[ -f "${BROWSERLESS_CLI_DIR}/adapters/common.sh" ]]; then
    # shellcheck disable=SC1090
    source "${BROWSERLESS_CLI_DIR}/adapters/common.sh"
    # shellcheck disable=SC1090
    source "${BROWSERLESS_CLI_DIR}/adapters/registry.sh"
fi

# Initialize CLI framework
cli::init "browserless" "Browserless Chrome automation management"

# Register additional Browserless-specific commands
cli::register_command "inject" "Inject configuration into Browserless" "browserless_inject" "modifies-system"
cli::register_command "screenshot" "Take screenshot of URL" "browserless_screenshot"
cli::register_command "pdf" "Generate PDF from URL" "browserless_pdf"
cli::register_command "test-apis" "Test all Browserless APIs" "browserless_test_apis"
cli::register_command "metrics" "Show browser pressure/metrics" "browserless_metrics"
cli::register_command "credentials" "Get connection credentials for n8n integration" "browserless_credentials"
cli::register_command "console-capture" "Capture console logs from any URL" "browserless_console_capture"
cli::register_command "uninstall" "Uninstall Browserless (requires --force)" "browserless_uninstall" "modifies-system"

# Function management commands
cli::register_command "list-functions" "List all injected functions" "browserless_list_functions"
cli::register_command "describe" "Show function details" "browserless_describe_function"
cli::register_command "execute" "Execute stored function" "browserless_execute_function"
cli::register_command "remove-function" "Remove stored function" "browserless_remove_function" "modifies-system"
cli::register_command "validate" "Validate function JSON without storing" "browserless_validate_function"

# Workflow management commands
cli::register_command "workflow" "Manage workflows (create, run, list, etc.)" "browserless_workflow"

# Register adapter command for the new "for" pattern
cli::register_command "for" "Use browserless as adapter for other resources" "browserless_adapter"

# Legacy commands for backward compatibility (will delegate to adapter)
cli::register_command "execute-workflow" "[LEGACY] Execute n8n workflow - use 'for n8n execute-workflow' instead" "browserless_execute_workflow_legacy"

################################################################################
# Resource-specific command implementations  
################################################################################

# Inject configuration or test data into browserless
browserless_inject() {
    local file="${1:-}"
    
    # If no file provided, inject test data
    if [[ -z "$file" ]]; then
        browserless::inject
        return $?
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${var_ROOT_DIR}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        echo "Usage: resource-browserless inject [file.json]"
        echo "  Without file: Inject test data for validation"
        echo "  With file: Inject custom function from JSON"
        return 1
    fi
    
    browserless::inject "$file"
}

# Take screenshot
browserless_screenshot() {
    local url="${1:-}"
    local output="${2:-screenshot.png}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for screenshot"
        echo "Usage: resource-browserless screenshot <url> [output-file]"
        return 1
    fi
    
    browserless::safe_screenshot "$url" "$output"
}

# Generate PDF
browserless_pdf() {
    local url="${1:-}"
    local output="${2:-output.pdf}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for PDF generation"
        echo "Usage: resource-browserless pdf <url> [output-file]"
        return 1
    fi
    
    browserless::test_pdf "$url" "$output"
}

# Test all APIs
browserless_test_apis() {
    browserless::test_all_apis
}

# Show browser pressure/metrics
browserless_metrics() {
    browserless::test_pressure
}

# Get credentials for n8n integration  
browserless_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    # Parse arguments
    credentials::parse_args "$@"
    local parse_result=$?
    if [[ $parse_result -eq 2 ]]; then
        credentials::show_help "browserless"
        return 0
    elif [[ $parse_result -ne 0 ]]; then
        return 1
    fi
    
    # Get resource status by checking Docker container
    local status
    status=$(credentials::get_resource_status "$BROWSERLESS_CONTAINER_NAME")
    
    # Build connections array for Browserless
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Build connection JSON for Browserless HTTP API
        local connection_json
        connection_json=$(jq -n \
            --arg id "api" \
            --arg name "Browserless API" \
            --arg n8n_credential_type "httpHeaderAuth" \
            --arg host "localhost" \
            --arg port "${BROWSERLESS_PORT}" \
            --arg path "/json" \
            --arg description "Browserless Chrome automation API" \
            '{
                id: $id,
                name: $name,
                n8n_credential_type: $n8n_credential_type,
                connection: {
                    host: $host,
                    port: ($port | tonumber),
                    path: $path,
                    ssl: false
                },
                auth: {
                    header_name: "Authorization",
                    header_value: "",
                    token: null,
                    api_key: null
                },
                metadata: {
                    description: $description,
                    capabilities: ["browser", "pdf", "screenshot", "scraping", "automation"],
                    version: "latest"
                }
            }')
        
        connections_array=$(echo "$connection_json" | jq -s '.')
    fi
    
    # Build and validate response
    local response
    response=$(credentials::build_response "browserless" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

# Adapter command handler - implements the "for" pattern
browserless_adapter() {
    local target_resource="${1:-}"
    shift || true
    
    if [[ -z "$target_resource" ]]; then
        log::error "Target resource required"
        echo "Usage: resource-browserless for <resource> <command> [options]"
        echo ""
        echo "Available adapters:"
        if declare -f adapter::list >/dev/null; then
            adapter::list
        else
            echo "  n8n     - UI automation for n8n workflows"
            echo "  vault   - UI automation for HashiCorp Vault (coming soon)"
            echo "  grafana - Dashboard automation (coming soon)"
        fi
        return 1
    fi
    
    # Load the appropriate adapter
    local adapter_dir="${BROWSERLESS_CLI_DIR}/adapters/${target_resource}"
    local adapter_api="${adapter_dir}/api.sh"
    
    if [[ ! -f "$adapter_api" ]]; then
        log::error "Adapter not found for resource: $target_resource"
        echo "Available adapters:"
        ls -d "${BROWSERLESS_CLI_DIR}/adapters/"*/ 2>/dev/null | xargs -n1 basename | grep -v common | grep -v registry
        return 1
    fi
    
    # Source the adapter
    # shellcheck disable=SC1090
    source "$adapter_api"
    
    # Dispatch to adapter
    if declare -f "${target_resource}::dispatch" >/dev/null; then
        "${target_resource}::dispatch" "$@"
    else
        log::error "Adapter dispatch function not found: ${target_resource}::dispatch"
        return 1
    fi
}

# Legacy execute-workflow command for backward compatibility
browserless_execute_workflow_legacy() {
    log::warn "⚠️  This command syntax is deprecated. Please use: resource-browserless for n8n execute-workflow"
    echo ""
    
    # Delegate to adapter
    browserless_adapter "n8n" "execute-workflow" "$@"
}

# Execute n8n workflow via browser automation with enhanced session management
browserless_execute_workflow() {
    local workflow_id="${1:-}"
    local n8n_url="${2:-http://localhost:5678}"
    local timeout="${3:-60000}"
    local input_data="${4:-}"
    local use_persistent_session="${5:-true}"
    local session_id="${6:-}"
    
    if [[ -z "$workflow_id" ]]; then
        log::error "Workflow ID required"
        echo "Usage: resource-browserless execute-workflow <workflow-id> [n8n-url] [timeout-ms] [input-json] [persistent-session] [session-id]"
        echo ""
        echo "Enhanced Features:"
        echo "  • Persistent browser sessions (cookies, auth state preserved)"
        echo "  • Extended execution monitoring (waits for actual completion)"
        echo "  • High-quality final state screenshots"
        echo "  • Comprehensive execution state tracking"
        echo ""
        echo "Parameters:"
        echo "  workflow-id        : N8N workflow ID (required)"
        echo "  n8n-url           : N8N instance URL (default: http://localhost:5678)"
        echo "  timeout-ms        : Timeout in milliseconds (default: 60000)"
        echo "  input-json        : Input data as JSON, @file, or \$ENV_VAR (optional)"
        echo "  persistent-session: Use persistent session (true/false, default: true)"
        echo "  session-id        : Custom session ID (auto-generated if not provided)"
        echo ""
        echo "Examples:"
        echo "  # Execute workflow with persistent session (recommended)"
        echo "  resource-browserless execute-workflow my-workflow"
        echo ""
        echo "  # Execute with custom session ID for reuse"
        echo "  resource-browserless execute-workflow my-workflow http://localhost:5678 60000 '{}' true my-session-1"
        echo ""
        echo "  # Execute without session persistence (fresh browser each time)"
        echo "  resource-browserless execute-workflow my-workflow http://localhost:5678 60000 '{}' false"
        echo ""
        echo "  # Execute with JSON input data"
        echo "  resource-browserless execute-workflow embedding-generator http://localhost:5678 60000 '{\"text\":\"Hello world\"}'"
        echo ""
        echo "  # Execute with JSON file input"
        echo "  resource-browserless execute-workflow my-workflow http://localhost:5678 60000 @input.json"
        echo ""
        echo "  # Execute with environment variable input"
        echo "  export WORKFLOW_INPUT='{\"text\":\"test\"}'"
        echo "  resource-browserless execute-workflow embedding-generator"
        return 1
    fi
    
    browserless::execute_n8n_workflow "$workflow_id" "$n8n_url" "$timeout" "$input_data" "$use_persistent_session" "$session_id"
}

# Capture console logs from any URL
browserless_console_capture() {
    local url="${1:-}"
    local output="${2:-console-capture.json}"
    local wait_time="${3:-3000}"
    
    if [[ -z "$url" ]]; then
        log::error "URL required for console capture"
        echo "Usage: resource-browserless console-capture <url> [output-file] [wait-time-ms]"
        echo "Example: resource-browserless console-capture http://localhost:3000 logs.json 5000"
        return 1
    fi
    
    browserless::capture_console_logs "$url" "$output" "$wait_time"
}

################################################################################
# Function Management Commands
################################################################################

# List all injected functions
browserless_list_functions() {
    local functions_dir="${BROWSERLESS_DATA_DIR}/functions"
    local registry_file="${functions_dir}/registry.json"
    
    if [[ ! -f "$registry_file" ]]; then
        log::info "No functions found. Inject functions with: resource-browserless inject <file.json>"
        return 0
    fi
    
    log::header "📁 Browserless Functions"
    echo
    
    local total_functions
    total_functions=$(jq -r '.metadata.total_functions' "$registry_file")
    
    if [[ "$total_functions" == "0" ]]; then
        log::info "No functions stored."
        echo "📝 Inject functions with: resource-browserless inject <file.json>"
        return 0
    fi
    
    echo "Total functions: $total_functions"
    echo
    
    # List functions with details
    jq -r '.functions | to_entries[] | "🔧 \(.key) - Status: \(.value.status) - Created: \(.value.created // "unknown")"' "$registry_file"
    
    echo
    echo "💡 Commands:"
    echo "  resource-browserless describe <function-name>  # Show details"
    echo "  resource-browserless execute <function-name>   # Run function"
    echo "  resource-browserless remove-function <name>    # Delete function"
}

# Show function details
browserless_describe_function() {
    local function_name="${1:-}"
    
    if [[ -z "$function_name" ]]; then
        log::error "Function name required"
        echo "Usage: resource-browserless describe <function-name>"
        return 1
    fi
    
    local function_dir="${BROWSERLESS_DATA_DIR}/functions/${function_name}"
    local manifest_file="${function_dir}/manifest.json"
    
    if [[ ! -f "$manifest_file" ]]; then
        log::error "Function not found: $function_name"
        log::info "List available functions with: resource-browserless list-functions"
        return 1
    fi
    
    log::header "📄 Function Details: $function_name"
    echo
    
    # Show metadata
    echo "📝 Metadata:"
    jq -r '.metadata | to_entries[] | "  \(.key): \(.value)"' "$manifest_file"
    echo
    
    # Show function parameters
    echo "⚙️  Parameters:"
    local param_count
    param_count=$(jq -r '.function.parameters | length' "$manifest_file")
    if [[ "$param_count" == "0" ]]; then
        echo "  No parameters defined"
    else
        jq -r '.function.parameters | to_entries[] | "  \(.key) (\(.value.type)): \(.value.description // "No description")"' "$manifest_file"
    fi
    echo
    
    # Show execution info
    echo "🚀 Execution:"
    echo "  Timeout: $(jq -r '.function.timeout // "60000"' "$manifest_file")ms"
    echo "  Persistent Session: $(jq -r '.execution.persistent_session // true' "$manifest_file")"
    echo
    
    # Show file locations
    echo "📁 Files:"
    echo "  Manifest: $manifest_file"
    echo "  Function: ${function_dir}/function.js"
    echo "  Executions: ${function_dir}/executions/"
    
    # Show recent executions if any
    local executions_dir="${function_dir}/executions"
    if [[ -d "$executions_dir" ]]; then
        local exec_count
        exec_count=$(find "$executions_dir" -name "*.json" -type f | wc -l)
        if [[ "$exec_count" -gt "0" ]]; then
            echo
            echo "📊 Recent Executions: $exec_count total"
            find "$executions_dir" -name "*.json" -type f | sort -r | head -3 | while read -r exec_file; do
                local exec_time
                exec_time=$(basename "$exec_file" .json)
                echo "  $(date -d "@$exec_time" 2>/dev/null || echo "$exec_time")"
            done
        fi
    fi
}

# Execute stored function
browserless_execute_function() {
    local function_name="${1:-}"
    shift || true
    
    if [[ -z "$function_name" ]]; then
        log::error "Function name required"
        echo "Usage: resource-browserless execute <function-name> [params...]"
        echo "Example: resource-browserless execute screenshot-dashboard url=http://localhost:3000 width=1920"
        return 1
    fi
    
    local function_dir="${BROWSERLESS_DATA_DIR}/functions/${function_name}"
    local manifest_file="${function_dir}/manifest.json"
    local function_file="${function_dir}/function.js"
    
    if [[ ! -f "$manifest_file" || ! -f "$function_file" ]]; then
        log::error "Function not found: $function_name"
        log::info "List available functions with: resource-browserless list-functions"
        return 1
    fi
    
    log::header "🚀 Executing Function: $function_name"
    
    # Parse parameters from command line
    local params="{}"
    for arg in "$@"; do
        if [[ "$arg" == *"="* ]]; then
            local key="${arg%%=*}"
            local value="${arg#*=}"
            params=$(echo "$params" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
        fi
    done
    
    log::info "📥 Parameters: $(echo "$params" | jq -c .)"
    
    # Get function code and settings
    local function_code
    function_code=$(<"$function_file")
    
    local timeout
    timeout=$(jq -r '.function.timeout // 60000' "$manifest_file")
    
    local persistent_session
    persistent_session=$(jq -r '.execution.persistent_session // true' "$manifest_file")
    
    # Create execution context
    local execution_id
    execution_id=$(date +%s)
    local execution_dir="${function_dir}/executions"
    mkdir -p "$execution_dir"
    
    # Wrap function with parameter injection and context
    local wrapped_function
    wrapped_function="
    const params = $params;
    const context = {
        mode: 'function_execution',
        functionName: '$function_name',
        executionId: '$execution_id',
        outputDir: '$execution_dir',
        timestamp: new Date().toISOString()
    };
    
    $function_code
    "
    
    log::info "⚡ Executing function via browserless..."
    
    # Execute the function
    local result
    result=$(browserless::run_function "$wrapped_function" "$timeout" "$persistent_session" "function_${function_name}_${execution_id}")
    local exit_code=$?
    
    # Store execution result
    local result_file="${execution_dir}/${execution_id}.json"
    echo "$result" > "$result_file"
    cp "$result_file" "${execution_dir}/latest.json"
    
    # Update registry with execution info
    browserless::update_function_registry "$function_name" "executed"
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "✅ Function executed successfully"
        log::info "📄 Result saved to: $result_file"
        
        # Try to show result summary
        if command -v jq >/dev/null && jq . "$result_file" >/dev/null 2>&1; then
            echo
            echo "📊 Execution Result:"
            jq . "$result_file"
        fi
    else
        log::error "❌ Function execution failed"
        log::info "📄 Error details saved to: $result_file"
    fi
    
    return $exit_code
}

# Remove stored function
browserless_remove_function() {
    local function_name="${1:-}"
    
    if [[ -z "$function_name" ]]; then
        log::error "Function name required"
        echo "Usage: resource-browserless remove-function <function-name>"
        return 1
    fi
    
    local function_dir="${BROWSERLESS_DATA_DIR}/functions/${function_name}"
    
    if [[ ! -d "$function_dir" ]]; then
        log::error "Function not found: $function_name"
        return 1
    fi
    
    log::header "🗑️  Removing Function: $function_name"
    
    # Remove function directory
    rm -rf "$function_dir"
    
    # Update registry
    browserless::update_function_registry "$function_name" "remove"
    
    log::success "✅ Function '$function_name' removed successfully"
}

# Validate function JSON without storing
browserless_validate_function() {
    local file_path="${1:-}"
    
    if [[ -z "$file_path" ]]; then
        log::error "JSON file path required"
        echo "Usage: resource-browserless validate <file.json>"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file_path" == shared:* ]]; then
        file_path="${var_ROOT_DIR}/${file_path#shared:}"
    fi
    
    log::header "✅ Validating Function: $file_path"
    
    # Basic file checks
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if [[ ! -r "$file_path" ]]; then
        log::error "File not readable: $file_path"
        return 1
    fi
    
    # Validate JSON format
    if ! jq . "$file_path" >/dev/null 2>&1; then
        log::error "❌ Invalid JSON format"
        return 1
    fi
    log::success "✅ Valid JSON format"
    
    # Check required fields
    local function_name
    function_name=$(jq -r '.metadata.name // empty' "$file_path")
    if [[ -z "$function_name" ]]; then
        log::error "❌ Missing required field: metadata.name"
        return 1
    fi
    log::success "✅ Function name: $function_name"
    
    # Validate function name format
    if ! [[ "$function_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log::error "❌ Invalid function name format (only alphanumeric, dash, underscore allowed)"
        return 1
    fi
    log::success "✅ Valid function name format"
    
    # Check function code
    local function_code
    function_code=$(jq -r '.function.code // empty' "$file_path")
    if [[ -z "$function_code" ]]; then
        log::error "❌ Missing required field: function.code"
        return 1
    fi
    log::success "✅ Function code present"
    
    # Basic JavaScript syntax validation
    if command -v node >/dev/null; then
        if echo "$function_code" | node -c 2>/dev/null; then
            log::success "✅ JavaScript syntax valid"
        else
            log::warn "⚠️  JavaScript syntax validation failed (may still work in browser context)"
        fi
    else
        log::info "ℹ️  Node.js not available - skipping syntax validation"
    fi
    
    # Show function summary
    echo
    echo "📋 Function Summary:"
    echo "  Name: $function_name"
    echo "  Description: $(jq -r '.metadata.description // "No description"' "$file_path")"
    echo "  Timeout: $(jq -r '.function.timeout // "60000"' "$file_path")ms"
    echo "  Parameters: $(jq -r '.function.parameters | length' "$file_path")"
    
    log::success "✅ Function validation passed"
    log::info "💾 Inject with: resource-browserless inject $file_path"
}

################################################################################
# Workflow Management Commands
################################################################################

# Main workflow command dispatcher
browserless_workflow() {
    local subcommand="${1:-help}"
    shift || true
    
    # Source workflow libraries
    source "${BROWSERLESS_CLI_DIR}/lib/workflow/parser.sh"
    source "${BROWSERLESS_CLI_DIR}/lib/workflow/compiler.sh"
    source "${BROWSERLESS_CLI_DIR}/lib/workflow/debug.sh"
    
    case "$subcommand" in
        create)
            browserless_workflow_create "$@"
            ;;
        run|execute)
            browserless_workflow_run "$@"
            ;;
        list)
            browserless_workflow_list "$@"
            ;;
        describe|show)
            browserless_workflow_describe "$@"
            ;;
        validate)
            browserless_workflow_validate "$@"
            ;;
        results)
            browserless_workflow_results "$@"
            ;;
        delete|remove)
            browserless_workflow_delete "$@"
            ;;
        compile)
            browserless_workflow_compile "$@"
            ;;
        help|--help|-h)
            browserless_workflow_help
            ;;
        *)
            log::error "Unknown workflow subcommand: $subcommand"
            browserless_workflow_help
            return 1
            ;;
    esac
}

# Show workflow help
browserless_workflow_help() {
    cat <<EOF
Browserless Workflow Management

Usage:
  resource-browserless workflow <subcommand> [options]

Subcommands:
  create <workflow.yaml>    Create workflow from YAML/JSON file
  run <name> [params]       Run workflow with parameters
  list                      List all workflows
  describe <name>           Show workflow details
  validate <workflow.yaml>  Validate workflow file
  results <name> [exec-id]  View workflow execution results
  delete <name>             Delete workflow
  compile <workflow.yaml>   Compile workflow to JavaScript
  help                      Show this help message

Examples:
  # Create workflow from YAML file
  resource-browserless workflow create login-flow.yaml
  
  # Run workflow with parameters
  resource-browserless workflow run login-flow username=admin dashboard_url=http://localhost:3000
  
  # View latest execution results
  resource-browserless workflow results login-flow
  
  # List all workflows
  resource-browserless workflow list

Workflow File Format:
  Workflows are defined in YAML or JSON with the following structure:
  
  workflow:
    name: "my-workflow"
    description: "Description of what this workflow does"
    parameters:
      param1:
        type: string
        required: true
    steps:
      - name: "step-name"
        action: "navigate"
        url: "\${params.param1}"
        debug:
          screenshot: true

For more information on workflow syntax and available actions, see:
  /docs/plans/browserless-workflow-system.md
EOF
}

# Create workflow from file
browserless_workflow_create() {
    local workflow_file="${1:-}"
    
    if [[ -z "$workflow_file" ]]; then
        log::error "Workflow file required"
        echo "Usage: resource-browserless workflow create <workflow.yaml>"
        return 1
    fi
    
    if [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow file not found: $workflow_file"
        return 1
    fi
    
    # Compile and store workflow
    workflow::compile_and_store "$workflow_file"
}

# Run workflow
browserless_workflow_run() {
    local workflow_name="${1:-}"
    shift || true
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name required"
        echo "Usage: resource-browserless workflow run <name> [param=value...]"
        return 1
    fi
    
    local workflow_dir="${BROWSERLESS_DATA_DIR}/workflows/${workflow_name}"
    
    if [[ ! -d "$workflow_dir" ]]; then
        log::error "Workflow not found: $workflow_name"
        log::info "List available workflows with: resource-browserless workflow list"
        return 1
    fi
    
    log::header "🚀 Running Workflow: $workflow_name"
    
    # Parse parameters
    local params="{}"
    for arg in "$@"; do
        if [[ "$arg" == *"="* ]]; then
            local key="${arg%%=*}"
            local value="${arg#*=}"
            params=$(echo "$params" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
        fi
    done
    
    log::info "Parameters: $(echo "$params" | jq -c .)"
    
    # Initialize debug execution
    local execution_id=$(date +%s)
    local debug_dir
    debug_dir=$(debug::init_execution "$workflow_name" "$execution_id")
    
    # Create context
    local context
    context=$(jq -n \
        --arg workflow "$workflow_name" \
        --arg execution "$execution_id" \
        --arg output "$debug_dir/outputs" \
        '{
            workflow: $workflow,
            executionId: $execution,
            outputDir: $output,
            timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ"))
        }')
    
    # Get compiled function
    local compiled_js="${workflow_dir}/compiled.js"
    
    if [[ ! -f "$compiled_js" ]]; then
        log::error "Compiled workflow not found. Re-create the workflow."
        return 1
    fi
    
    # Read function code
    local function_code
    function_code=$(<"$compiled_js")
    
    # Execute via browserless
    log::info "Executing workflow..."
    local result
    result=$(browserless::run_function "$function_code" "60000" "true" "workflow_${workflow_name}_${execution_id}")
    local exit_code=$?
    
    # Store result
    echo "$result" > "${debug_dir}/result.json"
    
    # Update metadata
    local status="completed"
    if [[ $exit_code -ne 0 ]]; then
        status="failed"
    fi
    
    jq --arg status "$status" \
       --arg ended "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
       '.status = $status | .ended_at = $ended' \
       "${debug_dir}/metadata.json" > "${debug_dir}/metadata.json.tmp" && \
       mv "${debug_dir}/metadata.json.tmp" "${debug_dir}/metadata.json"
    
    if [[ $exit_code -eq 0 ]]; then
        log::success "✅ Workflow completed successfully"
    else
        log::error "❌ Workflow failed"
    fi
    
    # Show summary
    debug::view_results "$workflow_name" "$execution_id"
}

# List workflows
browserless_workflow_list() {
    local workflows_dir="${BROWSERLESS_DATA_DIR}/workflows"
    
    if [[ ! -d "$workflows_dir" ]]; then
        log::info "No workflows found"
        log::info "Create workflows with: resource-browserless workflow create <workflow.yaml>"
        return 0
    fi
    
    log::header "📋 Browserless Workflows"
    echo
    
    local count=0
    for workflow_dir in "$workflows_dir"/*; do
        if [[ -d "$workflow_dir" ]] && [[ -f "$workflow_dir/metadata.json" ]]; then
            local workflow_name=$(basename "$workflow_dir")
            local metadata
            metadata=$(<"$workflow_dir/metadata.json")
            
            local description=$(echo "$metadata" | jq -r '.description // ""')
            local step_count=$(echo "$metadata" | jq -r '.step_count // 0')
            
            echo "📝 $workflow_name"
            if [[ -n "$description" ]]; then
                echo "   $description"
            fi
            echo "   Steps: $step_count"
            
            # Check for recent executions
            if [[ -d "$workflow_dir/executions" ]]; then
                local exec_count
                exec_count=$(find "$workflow_dir/executions" -maxdepth 1 -type d | wc -l)
                exec_count=$((exec_count - 1))  # Subtract parent directory
                if [[ $exec_count -gt 0 ]]; then
                    echo "   Executions: $exec_count"
                fi
            fi
            echo
            
            count=$((count + 1))
        fi
    done
    
    if [[ $count -eq 0 ]]; then
        log::info "No workflows found"
        log::info "Create workflows with: resource-browserless workflow create <workflow.yaml>"
    else
        echo "Total workflows: $count"
        echo
        echo "Commands:"
        echo "  resource-browserless workflow run <name>      # Run workflow"
        echo "  resource-browserless workflow describe <name>  # Show details"
        echo "  resource-browserless workflow results <name>   # View results"
    fi
}

# Describe workflow
browserless_workflow_describe() {
    local workflow_name="${1:-}"
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name required"
        echo "Usage: resource-browserless workflow describe <name>"
        return 1
    fi
    
    local workflow_dir="${BROWSERLESS_DATA_DIR}/workflows/${workflow_name}"
    
    if [[ ! -d "$workflow_dir" ]]; then
        log::error "Workflow not found: $workflow_name"
        return 1
    fi
    
    log::header "📄 Workflow Details: $workflow_name"
    echo
    
    # Show metadata
    if [[ -f "$workflow_dir/metadata.json" ]]; then
        local metadata
        metadata=$(<"$workflow_dir/metadata.json")
        
        echo "📝 Metadata:"
        echo "  Name: $(echo "$metadata" | jq -r '.name')"
        echo "  Description: $(echo "$metadata" | jq -r '.description // "No description"')"
        echo "  Version: $(echo "$metadata" | jq -r '.version // "1.0.0"')"
        echo "  Steps: $(echo "$metadata" | jq -r '.step_count // 0')"
        echo "  Debug Level: $(echo "$metadata" | jq -r '.debug_level // "none"')"
        echo
        
        echo "🎯 Actions Used:"
        echo "$metadata" | jq -r '.actions[]' | sed 's/^/  - /'
        echo
    fi
    
    # Show parameters
    if [[ -f "$workflow_dir/workflow.json" ]]; then
        local workflow
        workflow=$(<"$workflow_dir/workflow.json")
        
        echo "⚙️  Parameters:"
        local param_count
        param_count=$(echo "$workflow" | jq '.workflow.parameters | length')
        
        if [[ "$param_count" == "0" ]]; then
            echo "  No parameters defined"
        else
            echo "$workflow" | jq -r '.workflow.parameters | to_entries[] | "  \(.key) (\(.value.type // "string")): \(.value.description // "No description")"'
        fi
        echo
    fi
    
    # Show files
    echo "📁 Files:"
    echo "  Workflow: $workflow_dir/workflow.json"
    echo "  Compiled: $workflow_dir/compiled.js"
    echo "  Metadata: $workflow_dir/metadata.json"
    
    # Show recent executions
    if [[ -d "$workflow_dir/executions" ]]; then
        local exec_count
        exec_count=$(find "$workflow_dir/executions" -maxdepth 1 -type d | wc -l)
        exec_count=$((exec_count - 1))  # Subtract parent directory
        
        if [[ $exec_count -gt 0 ]]; then
            echo
            echo "📊 Recent Executions: $exec_count total"
            find "$workflow_dir/executions" -maxdepth 1 -type d | sort -r | head -4 | tail -3 | while read -r exec_dir; do
                if [[ "$exec_dir" != "$workflow_dir/executions" ]]; then
                    local exec_id=$(basename "$exec_dir")
                    echo "  - $exec_id"
                fi
            done
        fi
    fi
}

# Validate workflow file
browserless_workflow_validate() {
    local workflow_file="${1:-}"
    
    if [[ -z "$workflow_file" ]]; then
        log::error "Workflow file required"
        echo "Usage: resource-browserless workflow validate <workflow.yaml>"
        return 1
    fi
    
    if [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow file not found: $workflow_file"
        return 1
    fi
    
    log::header "✅ Validating Workflow: $workflow_file"
    
    # Parse and validate
    local workflow_json
    workflow_json=$(workflow::parse "$workflow_file")
    
    if [[ $? -eq 0 ]]; then
        log::success "✅ Workflow is valid"
        
        # Show summary
        local metadata
        metadata=$(workflow::extract_metadata "$workflow_json")
        
        echo
        echo "📋 Workflow Summary:"
        echo "  Name: $(echo "$metadata" | jq -r '.name')"
        echo "  Description: $(echo "$metadata" | jq -r '.description // "No description"')"
        echo "  Steps: $(echo "$metadata" | jq -r '.step_count')"
        echo "  Actions: $(echo "$metadata" | jq -r '.actions | join(", ")')"
    else
        log::error "❌ Workflow validation failed"
        return 1
    fi
}

# View workflow results
browserless_workflow_results() {
    local workflow_name="${1:-}"
    local execution_id="${2:-latest}"
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name required"
        echo "Usage: resource-browserless workflow results <name> [execution-id]"
        return 1
    fi
    
    debug::view_results "$workflow_name" "$execution_id"
}

# Delete workflow
browserless_workflow_delete() {
    local workflow_name="${1:-}"
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name required"
        echo "Usage: resource-browserless workflow delete <name>"
        return 1
    fi
    
    local workflow_dir="${BROWSERLESS_DATA_DIR}/workflows/${workflow_name}"
    
    if [[ ! -d "$workflow_dir" ]]; then
        log::error "Workflow not found: $workflow_name"
        return 1
    fi
    
    log::header "🗑️  Deleting Workflow: $workflow_name"
    
    # Remove workflow directory
    rm -rf "$workflow_dir"
    
    log::success "✅ Workflow '$workflow_name' deleted successfully"
}

# Compile workflow (for testing/debugging)
browserless_workflow_compile() {
    local workflow_file="${1:-}"
    
    if [[ -z "$workflow_file" ]]; then
        log::error "Workflow file required"
        echo "Usage: resource-browserless workflow compile <workflow.yaml>"
        return 1
    fi
    
    if [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow file not found: $workflow_file"
        return 1
    fi
    
    log::header "🔧 Compiling Workflow: $workflow_file"
    
    # Parse workflow
    local workflow_json
    workflow_json=$(workflow::parse "$workflow_file")
    
    if [[ $? -ne 0 ]]; then
        log::error "Failed to parse workflow"
        return 1
    fi
    
    # Compile to JavaScript
    local compiled_js
    compiled_js=$(workflow::compile "$workflow_json")
    
    echo
    echo "📝 Compiled JavaScript:"
    echo "----------------------------------------"
    echo "$compiled_js"
    echo "----------------------------------------"
}

# Uninstall browserless
browserless_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Browserless and all its data. Use --force to confirm."
        return 1
    fi
    
    browserless::uninstall
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi