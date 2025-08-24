#!/bin/bash
# ====================================================================
# Workflow Test Helper Functions
# ====================================================================
# Provides functions to work with workflow metadata and execution in tests

# Get APP_ROOT using cached value or compute once (3 levels up: __test/fixtures/workflows/test_helpers.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
WORKFLOW_DIR="${APP_ROOT}/__test/fixtures/workflows"
METADATA_FILE="$WORKFLOW_DIR/workflows.yaml"

# Load metadata for a specific workflow
get_workflow_metadata() {
    local workflow_path="$1"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        echo "ERROR: Metadata file not found: $METADATA_FILE" >&2
        return 1
    fi
    
    # Use yq to query the metadata (assumes yq is installed)
    if command -v yq >/dev/null 2>&1; then
        yq eval ".workflows[][] | select(.path == \"$workflow_path\")" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get workflows by platform
get_workflows_by_platform() {
    local platform="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".workflows.${platform}[].path" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get all workflows with a specific tag
get_workflows_by_tag() {
    local tag="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".workflows[][] | select(.tags[] == \"$tag\") | .path" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get workflows for a specific test suite
get_test_suite_workflows() {
    local suite_name="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".testSuites.${suite_name}[]" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get workflow platform
get_workflow_platform() {
    local workflow_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".workflows[][] | select(.path == \"$workflow_path\") | .platform" "$METADATA_FILE"
    else
        return 1
    fi
}

# Get workflow integration resources
get_workflow_integrations() {
    local workflow_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".workflows[][] | select(.path == \"$workflow_path\") | .integration[]" "$METADATA_FILE"
    else
        return 1
    fi
}

# Get expected duration for workflow
get_workflow_duration() {
    local workflow_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".workflows[][] | select(.path == \"$workflow_path\") | .expectedDuration" "$METADATA_FILE"
    else
        return 1
    fi
}

# Get workflow complexity level
get_workflow_complexity() {
    local workflow_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".workflows[][] | select(.path == \"$workflow_path\") | .complexity" "$METADATA_FILE"
    else
        return 1
    fi
}

# Get platform configuration
get_platform_config() {
    local platform="$1"
    local config_key="$2"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".platforms.$platform.$config_key" "$METADATA_FILE"
    else
        return 1
    fi
}

# Get integration endpoint
get_integration_endpoint() {
    local integration="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".integrationExpectations.$integration.endpoint" "$METADATA_FILE"
    else
        return 1
    fi
}

# Check if platform is available
is_platform_available() {
    local platform="$1"
    local port=$(get_platform_config "$platform" "port")
    local health_endpoint=$(get_platform_config "$platform" "health_check")
    
    if [[ -z "$port" ]]; then
        echo "Unknown platform: $platform" >&2
        return 1
    fi
    
    # Check if port is open
    if ! nc -z localhost "$port" 2>/dev/null; then
        echo "Platform $platform not available on port $port" >&2
        return 1
    fi
    
    # If health check endpoint is defined, test it
    if [[ -n "$health_endpoint" && "$health_endpoint" != "null" ]]; then
        if ! curl -s -f "http://localhost:$port$health_endpoint" >/dev/null 2>&1; then
            echo "Platform $platform health check failed" >&2
            return 1
        fi
    fi
    
    return 0
}

# Execute workflow on specified platform
execute_workflow() {
    local platform="$1"
    local workflow_path="$2"
    local timeout="${3:-60}"  # Default 60 second timeout
    
    case "$platform" in
        "n8n")
            execute_n8n_workflow "$workflow_path" "$timeout"
            ;;
        "node-red")
            execute_node_red_workflow "$workflow_path" "$timeout"
            ;;
        "windmill")
            execute_windmill_workflow "$workflow_path" "$timeout"
            ;;
        "huginn")
            execute_huginn_workflow "$workflow_path" "$timeout"
            ;;
        "comfyui")
            execute_comfyui_workflow "$workflow_path" "$timeout"
            ;;
        *)
            echo "ERROR: Unsupported platform: $platform" >&2
            return 1
            ;;
    esac
}

# N8N workflow execution
execute_n8n_workflow() {
    local workflow_path="$1"
    local timeout="${2:-60}"
    local port=$(get_platform_config "n8n" "port")
    
    # Import workflow
    local import_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d @"$WORKFLOW_DIR/$workflow_path" \
        "http://localhost:$port/api/v1/workflows/import")
    
    if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to import N8N workflow" >&2
        return 1
    fi
    
    # Extract workflow ID from response
    local workflow_id=$(echo "$import_response" | jq -r '.id // empty')
    if [[ -z "$workflow_id" ]]; then
        echo "ERROR: Could not extract workflow ID from import response" >&2
        return 1
    fi
    
    # Execute workflow
    timeout "$timeout" curl -s -X POST \
        "http://localhost:$port/api/v1/workflows/$workflow_id/execute"
}

# Node-RED flow execution
execute_node_red_workflow() {
    local workflow_path="$1"
    local timeout="${2:-60}"
    local port=$(get_platform_config "node-red" "port")
    
    # Import flow
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d @"$WORKFLOW_DIR/$workflow_path" \
        "http://localhost:$port/flows"
    
    if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to import Node-RED flow" >&2
        return 1
    fi
    
    # Find inject node and trigger it
    local inject_node=$(jq -r '.[] | select(.type == "inject") | .id' "$WORKFLOW_DIR/$workflow_path")
    if [[ -n "$inject_node" ]]; then
        timeout "$timeout" curl -s -X POST \
            "http://localhost:$port/inject/$inject_node"
    else
        echo "ERROR: No inject node found in Node-RED flow" >&2
        return 1
    fi
}

# Windmill workflow execution
execute_windmill_workflow() {
    local workflow_path="$1"
    local timeout="${2:-60}"
    local port=$(get_platform_config "windmill" "port")
    
    echo "WARNING: Windmill execution not yet implemented" >&2
    return 1
}

# Huginn agent execution
execute_huginn_workflow() {
    local workflow_path="$1"
    local timeout="${2:-60}"
    local port=$(get_platform_config "huginn" "port")
    
    echo "WARNING: Huginn execution not yet implemented" >&2
    return 1
}

# ComfyUI workflow execution
execute_comfyui_workflow() {
    local workflow_path="$1"
    local timeout="${2:-60}"
    local port=$(get_platform_config "comfyui" "port")
    
    echo "WARNING: ComfyUI execution not yet implemented" >&2
    return 1
}

# Import workflow to platform
import_workflow() {
    local platform="$1"
    local workflow_path="$2"
    
    case "$platform" in
        "n8n")
            import_n8n_workflow "$workflow_path"
            ;;
        "node-red")
            import_node_red_workflow "$workflow_path"
            ;;
        *)
            echo "ERROR: Import not implemented for platform: $platform" >&2
            return 1
            ;;
    esac
}

# Import N8N workflow
import_n8n_workflow() {
    local workflow_path="$1"
    local port=$(get_platform_config "n8n" "port")
    
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d @"$WORKFLOW_DIR/$workflow_path" \
        "http://localhost:$port/api/v1/workflows/import"
}

# Import Node-RED flow
import_node_red_workflow() {
    local workflow_path="$1"
    local port=$(get_platform_config "node-red" "port")
    
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d @"$WORKFLOW_DIR/$workflow_path" \
        "http://localhost:$port/flows"
}

# Test workflow with expected behavior
test_workflow_execution() {
    local workflow_path="$1"
    local platform=$(get_workflow_platform "$workflow_path")
    local expected_duration=$(get_workflow_duration "$workflow_path")
    
    if [[ -z "$platform" ]]; then
        echo "ERROR: Could not determine platform for workflow: $workflow_path" >&2
        return 1
    fi
    
    # Check if platform is available
    if ! is_platform_available "$platform"; then
        echo "SKIP: Platform $platform is not available" >&2
        return 2  # Skip code
    fi
    
    # Execute workflow with timeout based on expected duration
    local timeout=$((expected_duration + 30))  # Add 30 second buffer
    local start_time=$(date +%s)
    
    if execute_workflow "$platform" "$workflow_path" "$timeout"; then
        local end_time=$(date +%s)
        local actual_duration=$((end_time - start_time))
        
        echo "SUCCESS: Workflow executed in ${actual_duration}s (expected: ${expected_duration}s)"
        return 0
    else
        echo "ERROR: Workflow execution failed" >&2
        return 1
    fi
}

# Validate all workflows in test suite
validate_test_suite() {
    local suite_name="$1"
    local failed_count=0
    local skipped_count=0
    local success_count=0
    
    echo "Validating test suite: $suite_name"
    
    while IFS= read -r workflow_path; do
        echo "Testing workflow: $workflow_path"
        
        if test_workflow_execution "$workflow_path"; then
            ((success_count++))
        elif [[ $? -eq 2 ]]; then
            ((skipped_count++))
        else
            ((failed_count++))
        fi
    done < <(get_test_suite_workflows "$suite_name")
    
    echo "Results: $success_count passed, $failed_count failed, $skipped_count skipped"
    return $failed_count
}

# Example usage in tests:
#
# # In a BATS test:
# @test "basic workflow integration" {
#     source "$FIXTURES_DIR/workflows/test_helpers.sh"
#     
#     # Test all basic integration workflows
#     while IFS= read -r workflow_path; do
#         run test_workflow_execution "$workflow_path"
#         if [[ $status -eq 2 ]]; then
#             skip "Platform not available"
#         fi
#         assert_success
#     done < <(get_workflows_by_tag "basic-integration")
# }
#
# @test "n8n platform workflows" {
#     source "$FIXTURES_DIR/workflows/test_helpers.sh"
#     
#     if ! is_platform_available "n8n"; then
#         skip "N8N not available"
#     fi
#     
#     while IFS= read -r workflow_path; do
#         run execute_workflow "n8n" "$workflow_path"
#         assert_success
#     done < <(get_workflows_by_platform "n8n")
# }
#
# @test "smoke test suite" {
#     source "$FIXTURES_DIR/workflows/test_helpers.sh"
#     
#     run validate_test_suite "smokeTests"
#     assert_success
# }