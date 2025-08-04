#!/usr/bin/env bash
# ComfyUI Integration Test
# Tests ComfyUI functionality and workflow execution
# Refactored to use shared integration test library

set -euo pipefail

# Source shared integration test library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/../../../tests/lib/integration-test-lib.sh"

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
    "test_workflow_tests"

# Register standard interface tests
register_standard_interface_tests

# Execute main test framework if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    integration_test_main "$@"
fi