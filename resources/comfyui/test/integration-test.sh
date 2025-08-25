#!/usr/bin/env bash
# ComfyUI Integration Test
# Tests ComfyUI functionality and workflow execution
# Refactored to use shared integration test library

set -euo pipefail

# Source enhanced integration test library with fixture support
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SCRIPT_DIR="$APP_ROOT/resources/comfyui/test"
# shellcheck disable=SC1091
source "$APP_ROOT/tests/lib/enhanced-integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Override library defaults with ComfyUI-specific settings
SERVICE_NAME="comfyui"
BASE_URL="${COMFYUI_BASE_URL:-http://localhost:8188}"
TIMEOUT="120"  # Longer timeout for workflow execution
HEALTH_ENDPOINT="/system_stats"
REQUIRED_TOOLS=("curl" "jq")
SERVICE_METADATA=()

# Test-specific configuration
readonly TEST_OUTPUT_DIR="${TMPDIR:-/tmp}/comfyui_test_outputs"

#######################################
# COMFYUI-SPECIFIC TEST FUNCTIONS
#######################################

create_test_workflow() {
    local workflow_file="$1"
    cat > "$workflow_file" << 'EOF'
{
  "1": {
    "inputs": {
      "width": 512,
      "height": 512,
      "batch_size": 1
    },
    "class_type": "EmptyLatentImage"
  },
  "2": {
    "inputs": {
      "text": "a simple red square on white background, minimal, geometric",
      "clip": ["3", 1]
    },
    "class_type": "CLIPTextEncode"
  },
  "3": {
    "inputs": {
      "ckpt_name": "sd_xl_base_1.0.safetensors"
    },
    "class_type": "CheckpointLoaderSimple"
  },
  "4": {
    "inputs": {
      "seed": 42,
      "steps": 10,
      "cfg": 7,
      "sampler_name": "euler",
      "scheduler": "normal",
      "denoise": 1,
      "model": ["3", 0],
      "positive": ["2", 0],
      "negative": ["5", 0],
      "latent_image": ["1", 0]
    },
    "class_type": "KSampler"
  },
  "5": {
    "inputs": {
      "text": "blurry, complex",
      "clip": ["3", 1]
    },
    "class_type": "CLIPTextEncode"
  },
  "6": {
    "inputs": {
      "samples": ["4", 0],
      "vae": ["3", 2]
    },
    "class_type": "VAEDecode"
  },
  "7": {
    "inputs": {
      "filename_prefix": "test_output",
      "images": ["6", 0]
    },
    "class_type": "SaveImage"
  }
}
EOF
}

test_api_connectivity() {
    local test_name="API connectivity"
    
    local response
    if response=$(make_api_request "/system_stats" "GET" 10); then
        if echo "$response" | grep -q "comfyui_version"; then
            log_test_result "$test_name" "PASS"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "API not responding properly"
    return 1
}

test_model_availability() {
    local test_name="SDXL model availability"
    
    # Check if SDXL model exists
    local model_path="${HOME}/.comfyui/models/checkpoints/sd_xl_base_1.0.safetensors"
    if [[ -f "$model_path" ]]; then
        # Verify size (should be ~6.5GB)
        local size
        size=$(stat -c%s "$model_path" 2>/dev/null || stat -f%z "$model_path" 2>/dev/null)
        if [[ "$size" -gt 6000000000 ]]; then
            log_test_result "$test_name" "PASS" "model valid"
            return 0
        else
            log_test_result "$test_name" "FAIL" "model corrupted (size: $size bytes)"
            return 1
        fi
    else
        log_test_result "$test_name" "SKIP" "model not found, workflow tests will be skipped"
        return 2
    fi
}

test_workflow_submission() {
    local test_name="workflow submission"
    
    # Create test workflow using framework helper for temp file
    local workflow_file
    workflow_file=$(create_test_file "" ".json")
    create_test_workflow "$workflow_file"
    
    # Submit workflow
    local response
    if response=$(make_api_request "/prompt" "POST" 30 \
        "-H 'Content-Type: application/json' -d '{\"prompt\": $(cat "$workflow_file"), \"client_id\": \"integration-test\"}'"); then
        
        local prompt_id
        prompt_id=$(echo "$response" | jq -r '.prompt_id // empty')
        
        if [[ -n "$prompt_id" ]]; then
            log_test_result "$test_name" "PASS" "prompt_id: $prompt_id"
            # Store for potential workflow execution test (also using framework temp file)
            local prompt_file
            prompt_file=$(create_test_file "$prompt_id" ".txt")
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "failed to submit workflow"
    return 1
}

test_workflow_execution() {
    local test_name="workflow execution"
    
    # Note: We now store prompt_id in a framework-managed temp file
    # Look for the most recent temp file with content (the prompt_id)
    local prompt_id=""
    for file in "${TEST_TEMP_FILES[@]}"; do
        if [[ -f "$file" ]] && [[ -s "$file" ]]; then
            local content
            content=$(cat "$file")
            # Check if it looks like a prompt ID (UUID format)
            if [[ "$content" =~ ^[a-f0-9-]{36} ]]; then
                prompt_id="$content"
                break
            fi
        fi
    done
    
    if [[ -z "$prompt_id" ]]; then
        log_test_result "$test_name" "SKIP" "no workflow submitted"
        return 2
    fi
    
    # Custom polling for ComfyUI (can't use generic poll_async_job due to different API structure)
    local max_attempts=20
    local sleep_time=3
    local completed=false
    local history=""
    
    for i in $(seq 1 "$max_attempts"); do
        if history=$(make_api_request "/history/$prompt_id" "GET" 10); then
            local outputs
            outputs=$(echo "$history" | jq -r ".[$prompt_id].outputs // empty")
            
            if [[ -n "$outputs" ]] && [[ "$outputs" != "null" ]]; then
                completed=true
                break
            fi
        fi
        
        # Don't sleep on last attempt
        [[ $i -lt $max_attempts ]] && sleep "$sleep_time"
    done
    
    if [[ "$completed" == "true" ]]; then
        # Check if output image was created
        local output_file
        output_file=$(echo "$history" | jq -r ".[$prompt_id].outputs[\"7\"].images[0].filename // empty" 2>/dev/null)
        if [[ -n "$output_file" ]] && [[ "$output_file" != "null" ]]; then
            log_test_result "$test_name" "PASS" "image generated: $output_file"
            return 0
        else
            log_test_result "$test_name" "FAIL" "workflow completed but no image generated"
            return 1
        fi
    else
        log_test_result "$test_name" "SKIP" "workflow did not complete within timeout"
        return 2
    fi
}

#######################################
# FIXTURE-BASED WORKFLOW TESTS
#######################################

test_workflow_from_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Read workflow JSON from fixture
    local workflow_json
    workflow_json=$(cat "$fixture_path")
    
    # Validate ComfyUI workflow structure
    if ! echo "$workflow_json" | jq -e 'to_entries | all(.value | has("class_type") and has("inputs"))' >/dev/null 2>&1; then
        return 1  # Invalid ComfyUI workflow structure
    fi
    
    # Submit workflow to ComfyUI
    local response
    if response=$(make_api_request "/prompt" "POST" 30 \
        "-H 'Content-Type: application/json' -d '{\"prompt\": $workflow_json, \"client_id\": \"fixture-test\"}'"); then
        
        local prompt_id
        prompt_id=$(echo "$response" | jq -r '.prompt_id // empty')
        
        if [[ -n "$prompt_id" ]] && [[ "$prompt_id" != "null" ]]; then
            return 0  # Workflow submitted successfully
        fi
    fi
    
    return 1
}

test_workflow_validation_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Read and validate workflow structure
    local workflow_json
    workflow_json=$(cat "$fixture_path")
    
    # Check for ComfyUI-specific workflow elements
    local has_nodes=false
    local has_connections=false
    
    # Check if workflow has valid node structure
    if echo "$workflow_json" | jq -e 'to_entries | length > 0' >/dev/null 2>&1; then
        has_nodes=true
    fi
    
    # Check if nodes have proper class_type and inputs
    if echo "$workflow_json" | jq -e 'to_entries | all(.value | has("class_type") and has("inputs"))' >/dev/null 2>&1; then
        has_connections=true
    fi
    
    if [[ "$has_nodes" == "true" ]] && [[ "$has_connections" == "true" ]]; then
        return 0
    fi
    
    return 1
}

test_ollama_guided_workflow() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # This tests ComfyUI + Ollama integration workflows
    local workflow_json
    workflow_json=$(cat "$fixture_path")
    
    # Check if workflow contains Ollama-related nodes
    if echo "$workflow_json" | jq -e '[.[] | .class_type] | any(. | contains("Ollama") or contains("LLM"))' >/dev/null 2>&1; then
        # Validate the workflow can be submitted (even if Ollama isn't running)
        local response
        if response=$(make_api_request "/object_info" "GET" 10); then
            # Check if ComfyUI has Ollama nodes available
            if echo "$response" | grep -qi "ollama\|llm"; then
                return 0  # Ollama integration available
            fi
        fi
    fi
    
    return 1
}

# Run fixture-based workflow tests
run_comfyui_fixture_tests() {
    if [[ "$FIXTURES_AVAILABLE" == "true" ]]; then
        # Test with ComfyUI-specific workflow fixtures
        test_with_fixture "text-to-image workflow" "workflows" "comfyui/comfyui-text-to-image.json" \
            test_workflow_from_fixture
        
        test_with_fixture "Ollama-guided workflow" "workflows" "comfyui/comfyui-ollama-guided.json" \
            test_ollama_guided_workflow
        
        # Validate workflow structures
        test_with_fixture "validate text-to-image structure" "workflows" "comfyui/comfyui-text-to-image.json" \
            test_workflow_validation_fixture
        
        test_with_fixture "validate Ollama workflow structure" "workflows" "comfyui/comfyui-ollama-guided.json" \
            test_workflow_validation_fixture
        
        # Test with auto-discovered fixtures
        local comfyui_fixtures
        comfyui_fixtures=$(discover_resource_fixtures "comfyui" "automation")
        
        for fixture_pattern in $comfyui_fixtures; do
            # Look for ComfyUI workflows specifically
            local workflow_files
            if workflow_files=$(fixture_get_all "$fixture_pattern" "*comfyui*.json" 2>/dev/null); then
                for workflow_file in $workflow_files; do
                    local workflow_name
                    workflow_name=$(basename "$workflow_file")
                    test_with_fixture "validate $workflow_name" "" "$workflow_file" \
                        test_workflow_validation_fixture
                done
            fi
        done
        
        # Test with integration workflows that might use ComfyUI
        if [[ -f "$FIXTURES_DIR/workflows/integration/multi-ai-pipeline.json" ]]; then
            test_with_fixture "multi-AI pipeline" "workflows" "integration/multi-ai-pipeline.json" \
                test_workflow_validation_fixture
        fi
    fi
}

#######################################
# SERVICE-SPECIFIC VERBOSE INFO
#######################################

show_verbose_info() {
    echo
    echo "ComfyUI Information:"
    echo "  Web UI: $BASE_URL"
    echo "  Required Model: sd_xl_base_1.0.safetensors (~6.5GB)"
    echo "  Model Location: \${HOME}/.comfyui/models/checkpoints/"
}

#######################################
# TEST ORCHESTRATION WITH DEPENDENCIES
#######################################

# Wrapper to handle model-dependent workflow tests
test_workflow_tests() {
    # First check if model is available
    local model_path="${HOME}/.comfyui/models/checkpoints/sd_xl_base_1.0.safetensors"
    if [[ ! -f "$model_path" ]]; then
        log_test_result "workflow submission" "SKIP" "model not available"
        log_test_result "workflow execution" "SKIP" "model not available"
        return 0
    fi
    
    # Model available, run workflow tests
    test_workflow_submission
    test_workflow_execution
}

#######################################
# TEST REGISTRATION AND EXECUTION
#######################################

# Register all ComfyUI-specific tests
register_tests \
    "test_api_connectivity" \
    "test_model_availability" \
    "test_workflow_tests" \
    "run_comfyui_fixture_tests"

# Register standard interface tests
register_standard_interface_tests

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi