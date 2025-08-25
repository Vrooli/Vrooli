#!/usr/bin/env bash
# ComfyUI Workflow Test Suite
# Tests basic ComfyUI functionality and workflow execution

set -euo pipefail

# Source test utilities using unique directory variable
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
COMFYUI_LIB_DIR="${APP_ROOT}/resources/comfyui/lib"
MANAGE_SCRIPT="${COMFYUI_LIB_DIR}/../manage.sh"
# Source trash module for safe cleanup
# shellcheck disable=SC1091
source "${COMFYUI_LIB_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test configuration
TEST_WORKFLOW_FILE="/tmp/test_workflow_simple.json"
TEST_OUTPUT_DIR="/tmp/comfyui_test_outputs"
TEST_TIMEOUT=120  # 2 minutes for workflow execution

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

#######################################
# Print test result
#######################################
print_result() {
    local test_name="$1"
    local result="$2"
    
    if [[ "$result" == "PASS" ]]; then
        echo -e "${GREEN}✓${NC} $test_name"
    else
        echo -e "${RED}✗${NC} $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_RUN=$((TESTS_RUN + 1))
}

#######################################
# Create a simple test workflow
#######################################
create_test_workflow() {
    cat > "$TEST_WORKFLOW_FILE" << 'EOF'
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

#######################################
# Test ComfyUI status
#######################################
test_status() {
    echo "Testing ComfyUI status..."
    
    if "$MANAGE_SCRIPT" --action status >/dev/null 2>&1; then
        print_result "ComfyUI status check" "PASS"
    else
        print_result "ComfyUI status check" "FAIL"
        echo "  ComfyUI is not running. Please start it first."
        return 1
    fi
}

#######################################
# Test model availability
#######################################
test_models() {
    echo "Testing model availability..."
    
    # Check if SDXL model exists
    local model_path="${HOME}/.comfyui/models/checkpoints/sd_xl_base_1.0.safetensors"
    if [[ -f "$model_path" ]]; then
        # Verify size (should be ~6.5GB)
        local size=$(stat -c%s "$model_path" 2>/dev/null || stat -f%z "$model_path" 2>/dev/null)
        if [[ "$size" -gt 6000000000 ]]; then
            print_result "SDXL model available and valid" "PASS"
        else
            print_result "SDXL model available and valid" "FAIL"
            echo "  Model exists but appears corrupted (size: $size bytes)"
        fi
    else
        print_result "SDXL model available and valid" "FAIL"
        echo "  Model not found. Run: $MANAGE_SCRIPT --action download-models"
    fi
}

#######################################
# Test API connectivity
#######################################
test_api() {
    echo "Testing API connectivity..."
    
    # Test system stats endpoint
    if curl -s http://localhost:8188/system_stats | grep -q "comfyui_version"; then
        print_result "API connectivity" "PASS"
    else
        print_result "API connectivity" "FAIL"
        echo "  Could not connect to ComfyUI API"
    fi
}

#######################################
# Test workflow execution
#######################################
test_workflow_execution() {
    echo "Testing workflow execution..."
    
    # Create test workflow
    create_test_workflow
    
    # Submit workflow
    local response=$(curl -s -X POST http://localhost:8188/prompt \
        -H "Content-Type: application/json" \
        -d "{\"prompt\": $(cat "$TEST_WORKFLOW_FILE"), \"client_id\": \"test-suite\"}")
    
    local prompt_id=$(echo "$response" | jq -r '.prompt_id // empty')
    
    if [[ -z "$prompt_id" ]]; then
        print_result "Workflow submission" "FAIL"
        echo "  Failed to submit workflow: $response"
        return
    fi
    
    print_result "Workflow submission" "PASS"
    
    # Wait for completion (with timeout)
    echo "  Waiting for workflow completion (timeout: ${TEST_TIMEOUT}s)..."
    local start_time=$(date +%s)
    local completed=false
    
    while [[ $(($(date +%s) - start_time)) -lt $TEST_TIMEOUT ]]; do
        local history=$(curl -s "http://localhost:8188/history/$prompt_id")
        local outputs=$(echo "$history" | jq -r ".[\"$prompt_id\"].outputs // empty")
        
        if [[ -n "$outputs" ]] && [[ "$outputs" != "null" ]]; then
            completed=true
            break
        fi
        
        sleep 2
    done
    
    if [[ "$completed" == "true" ]]; then
        print_result "Workflow execution" "PASS"
        
        # Check if output image was created
        local output_file=$(echo "$history" | jq -r ".[\"$prompt_id\"].outputs[\"7\"].images[0].filename // empty")
        if [[ -n "$output_file" ]]; then
            print_result "Output image generation" "PASS"
            echo "  Generated: $output_file"
        else
            print_result "Output image generation" "FAIL"
        fi
    else
        print_result "Workflow execution" "FAIL"
        echo "  Workflow did not complete within timeout"
    fi
}

#######################################
# Test import/export workflow
#######################################
test_workflow_import() {
    echo "Testing workflow import..."
    
    if "$MANAGE_SCRIPT" --action import-workflow --workflow "$TEST_WORKFLOW_FILE" >/dev/null 2>&1; then
        print_result "Workflow import" "PASS"
    else
        print_result "Workflow import" "FAIL"
    fi
}

#######################################
# Main test runner
#######################################
main() {
    echo "========================================="
    echo "ComfyUI Integration Test Suite"
    echo "========================================="
    echo
    
    TESTS_RUN=0
    TESTS_FAILED=0
    
    # Run tests
    test_status || exit 1
    test_models
    test_api
    test_workflow_import
    test_workflow_execution
    
    # Cleanup
    trash::safe_remove "$TEST_WORKFLOW_FILE" --test-cleanup
    
    # Summary
    echo
    echo "========================================="
    echo "Test Summary"
    echo "========================================="
    echo "Total tests: $TESTS_RUN"
    echo "Passed: $((TESTS_RUN - TESTS_FAILED))"
    echo "Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed.${NC}"
        exit 1
    fi
}

# Run tests
main "$@"