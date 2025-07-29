#!/usr/bin/env bats
# Tests for ComfyUI workflows.sh functions

# Setup for each test
setup() {
    # Set test environment
    export COMFYUI_CUSTOM_PORT="8188"
    export COMFYUI_CONTAINER_NAME="comfyui-test"
    export COMFYUI_BASE_URL="http://localhost:8188"
    export COMFYUI_WORKFLOWS_DIR="/tmp/comfyui-test/workflows"
    export WORKFLOW_FILE="/tmp/test-workflow.json"
    export WORKFLOW_ID="test-workflow-123"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    COMFYUI_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories and files
    mkdir -p "$COMFYUI_WORKFLOWS_DIR"
    
    # Create a test workflow file
    cat > "$WORKFLOW_FILE" <<EOF
{
  "1": {
    "inputs": {
      "ckpt_name": "model.safetensors"
    },
    "class_type": "CheckpointLoaderSimple"
  },
  "2": {
    "inputs": {
      "text": "a beautiful landscape",
      "clip": ["1", 1]
    },
    "class_type": "CLIPTextEncode"
  },
  "3": {
    "inputs": {
      "seed": 12345,
      "steps": 20,
      "cfg": 8.0,
      "sampler_name": "euler",
      "scheduler": "normal",
      "positive": ["2", 0],
      "negative": ["2", 0],
      "latent_image": ["4", 0]
    },
    "class_type": "KSampler"
  }
}
EOF
    
    # Mock system functions
    system::is_command() {
        case "$1" in
            "curl"|"jq"|"python3"|"find") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    # Mock curl for API interactions
    curl() {
        case "$*" in
            *"/api/v1/queue"*)
                if [[ "$*" =~ "POST" ]]; then
                    echo '{"prompt_id":"test-prompt-456","number":1,"node_errors":{}}'
                else
                    echo '{"queue_running":[{"prompt_id":"test-prompt-456","number":1}],"queue_pending":[]}'
                fi
                ;;
            *"/api/v1/history"*)
                echo '{"test-prompt-456":{"prompt":[{"1":{"inputs":{"ckpt_name":"model.safetensors"}}}],"outputs":{"3":{"images":[{"filename":"ComfyUI_00001_.png","type":"output"}]}},"status":{"status_str":"success","completed":true}}}'
                ;;
            *"/api/v1/interrupt"*)
                echo '{"interrupted":true}'
                ;;
            *"/api/v1/object_info"*)
                echo '{"CheckpointLoaderSimple":{"input":{"required":{"ckpt_name":["CHECKPOINT"]}}},"CLIPTextEncode":{"input":{"required":{"text":"STRING","clip":"CLIP"}}}}'
                ;;
            *"/api/v1/prompt"*)
                echo '{"prompt_id":"test-prompt-789"}'
                ;;
            *"/view"*)
                echo "Binary image data"
                ;;
            *) 
                echo "CURL: $*"
                return 0
                ;;
        esac
    }
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".prompt_id"*) echo "test-prompt-456" ;;
            *".queue_running | length"*) echo "1" ;;
            *".status.completed"*) echo "true" ;;
            *".status.status_str"*) echo "success" ;;
            *".outputs"*) echo '{"3":{"images":[{"filename":"ComfyUI_00001_.png"}]}}' ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock python3 for workflow validation
    python3() {
        case "$*" in
            *"validate"*)
                echo '{"valid": true, "errors": []}'
                ;;
            *"convert"*)
                echo "Workflow converted successfully"
                ;;
            *) echo "PYTHON: $*" ;;
        esac
        return 0
    }
    
    # Mock find for workflow discovery
    find() {
        case "$*" in
            *"-name *.json"*)
                echo "$COMFYUI_WORKFLOWS_DIR/text2img.json"
                echo "$COMFYUI_WORKFLOWS_DIR/img2img.json"
                echo "$COMFYUI_WORKFLOWS_DIR/inpainting.json"
                ;;
            *) echo "FIND: $*" ;;
        esac
    }
    
    # Mock file operations
    cat() {
        if [[ "$1" == "$WORKFLOW_FILE" ]]; then
            echo '{"1":{"inputs":{"ckpt_name":"model.safetensors"},"class_type":"CheckpointLoaderSimple"}}'
        else
            command cat "$@"
        fi
    }
    
    # Mock log functions
    log::info() { echo "INFO: $1"; }
    log::error() { echo "ERROR: $1"; }
    log::warn() { echo "WARN: $1"; }
    log::success() { echo "SUCCESS: $1"; }
    log::debug() { echo "DEBUG: $1"; }
    log::header() { echo "=== $1 ==="; }
    
    # Mock ComfyUI utility functions
    comfyui::container_exists() { return 0; }
    comfyui::is_running() { return 0; }
    comfyui::is_healthy() { return 0; }
    
    # Load configuration and messages
    source "${COMFYUI_DIR}/config/defaults.sh"
    source "${COMFYUI_DIR}/config/messages.sh"
    comfyui::export_config
    comfyui::export_messages
    
    # Load the functions to test
    source "${COMFYUI_DIR}/lib/workflows.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$COMFYUI_WORKFLOWS_DIR"
    rm -f "$WORKFLOW_FILE"
}

# Test workflow execution
@test "comfyui::execute_workflow runs workflow successfully" {
    result=$(comfyui::execute_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "Executing workflow" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "completed" ]]
    [[ "$result" =~ "test-prompt-456" ]]
}

# Test workflow execution with missing file
@test "comfyui::execute_workflow handles missing workflow file" {
    run comfyui::execute_workflow "/nonexistent/workflow.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
    [[ "$output" =~ "not found" ]]
}

# Test workflow validation
@test "comfyui::validate_workflow validates workflow structure" {
    result=$(comfyui::validate_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "validation" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "passed" ]]
}

# Test workflow validation failure
@test "comfyui::validate_workflow detects invalid workflow" {
    # Create invalid workflow
    echo '{"invalid": "json"}' > "/tmp/invalid-workflow.json"
    
    run comfyui::validate_workflow "/tmp/invalid-workflow.json"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "invalid" ]]
    
    rm -f "/tmp/invalid-workflow.json"
}

# Test workflow listing
@test "comfyui::list_workflows shows available workflows" {
    result=$(comfyui::list_workflows)
    
    [[ "$result" =~ "text2img.json" ]]
    [[ "$result" =~ "img2img.json" ]]
    [[ "$result" =~ "inpainting.json" ]]
}

# Test workflow queue status
@test "comfyui::get_queue_status shows current queue" {
    result=$(comfyui::get_queue_status)
    
    [[ "$result" =~ "queue" ]]
    [[ "$result" =~ "running" ]] || [[ "$result" =~ "pending" ]]
}

# Test workflow interruption
@test "comfyui::interrupt_workflow stops running workflow" {
    result=$(comfyui::interrupt_workflow)
    
    [[ "$result" =~ "interrupt" ]] || [[ "$result" =~ "stopped" ]]
    [[ "$result" =~ "SUCCESS:" ]] || [[ "$result" =~ "interrupted" ]]
}

# Test workflow history
@test "comfyui::get_workflow_history shows execution history" {
    result=$(comfyui::get_workflow_history)
    
    [[ "$result" =~ "history" ]]
    [[ "$result" =~ "test-prompt-456" ]]
    [[ "$result" =~ "success" ]]
}

# Test specific workflow history
@test "comfyui::get_workflow_history shows specific workflow results" {
    result=$(comfyui::get_workflow_history "test-prompt-456")
    
    [[ "$result" =~ "test-prompt-456" ]]
    [[ "$result" =~ "success" ]] || [[ "$result" =~ "completed" ]]
}

# Test workflow output retrieval
@test "comfyui::get_workflow_outputs retrieves workflow results" {
    result=$(comfyui::get_workflow_outputs "test-prompt-456")
    
    [[ "$result" =~ "output" ]]
    [[ "$result" =~ "ComfyUI_00001_.png" ]]
}

# Test workflow progress monitoring
@test "comfyui::monitor_workflow_progress tracks execution progress" {
    result=$(comfyui::monitor_workflow_progress "test-prompt-456")
    
    [[ "$result" =~ "progress" ]] || [[ "$result" =~ "monitoring" ]]
}

# Test workflow template creation
@test "comfyui::create_workflow_template generates workflow template" {
    result=$(comfyui::create_workflow_template "text2img")
    
    [[ "$result" =~ "template" ]]
    [[ "$result" =~ "text2img" ]]
}

# Test workflow parameter update
@test "comfyui::update_workflow_parameters modifies workflow parameters" {
    result=$(comfyui::update_workflow_parameters "$WORKFLOW_FILE" "steps" "30")
    
    [[ "$result" =~ "updated" ]] || [[ "$result" =~ "parameter" ]]
    [[ "$result" =~ "steps" ]]
}

# Test workflow conversion
@test "comfyui::convert_workflow_format converts workflow format" {
    result=$(comfyui::convert_workflow_format "$WORKFLOW_FILE" "api")
    
    [[ "$result" =~ "convert" ]] || [[ "$result" =~ "format" ]]
}

# Test workflow backup
@test "comfyui::backup_workflow creates workflow backup" {
    result=$(comfyui::backup_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "backup" ]] || [[ "$result" =~ "backed up" ]]
}

# Test workflow restore
@test "comfyui::restore_workflow restores workflow from backup" {
    result=$(comfyui::restore_workflow "$WORKFLOW_FILE.backup")
    
    [[ "$result" =~ "restore" ]] || [[ "$result" =~ "restored" ]]
}

# Test workflow optimization
@test "comfyui::optimize_workflow improves workflow performance" {
    result=$(comfyui::optimize_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "optimize" ]] || [[ "$result" =~ "optimized" ]]
}

# Test workflow dependencies check
@test "comfyui::check_workflow_dependencies validates workflow requirements" {
    result=$(comfyui::check_workflow_dependencies "$WORKFLOW_FILE")
    
    [[ "$result" =~ "dependencies" ]] || [[ "$result" =~ "requirements" ]]
}

# Test workflow sharing
@test "comfyui::share_workflow exports workflow for sharing" {
    result=$(comfyui::share_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "share" ]] || [[ "$result" =~ "export" ]]
}

# Test workflow import
@test "comfyui::import_workflow imports external workflow" {
    result=$(comfyui::import_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "import" ]] || [[ "$result" =~ "imported" ]]
}

# Test workflow scheduling
@test "comfyui::schedule_workflow schedules workflow execution" {
    result=$(comfyui::schedule_workflow "$WORKFLOW_FILE" "daily")
    
    [[ "$result" =~ "schedule" ]] || [[ "$result" =~ "scheduled" ]]
}

# Test batch workflow execution
@test "comfyui::execute_batch_workflows runs multiple workflows" {
    result=$(comfyui::execute_batch_workflows "$COMFYUI_WORKFLOWS_DIR")
    
    [[ "$result" =~ "batch" ]] || [[ "$result" =~ "multiple" ]]
}

# Test workflow performance analysis
@test "comfyui::analyze_workflow_performance analyzes execution metrics" {
    result=$(comfyui::analyze_workflow_performance "test-prompt-456")
    
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "analysis" ]]
}

# Test workflow error analysis
@test "comfyui::analyze_workflow_errors examines workflow failures" {
    result=$(comfyui::analyze_workflow_errors "test-prompt-456")
    
    [[ "$result" =~ "error" ]] || [[ "$result" =~ "analysis" ]]
}

# Test workflow caching
@test "comfyui::cache_workflow_results caches execution results" {
    result=$(comfyui::cache_workflow_results "test-prompt-456")
    
    [[ "$result" =~ "cache" ]] || [[ "$result" =~ "cached" ]]
}

# Test workflow version control
@test "comfyui::version_workflow manages workflow versions" {
    result=$(comfyui::version_workflow "$WORKFLOW_FILE" "v1.1")
    
    [[ "$result" =~ "version" ]] || [[ "$result" =~ "versioned" ]]
}

# Test workflow documentation
@test "comfyui::document_workflow generates workflow documentation" {
    result=$(comfyui::document_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "document" ]] || [[ "$result" =~ "documentation" ]]
}

# Test workflow testing
@test "comfyui::test_workflow performs workflow testing" {
    result=$(comfyui::test_workflow "$WORKFLOW_FILE")
    
    [[ "$result" =~ "test" ]] || [[ "$result" =~ "testing" ]]
}

# Test workflow comparison
@test "comfyui::compare_workflows compares workflow differences" {
    result=$(comfyui::compare_workflows "$WORKFLOW_FILE" "$WORKFLOW_FILE")
    
    [[ "$result" =~ "compare" ]] || [[ "$result" =~ "comparison" ]]
}

# Test workflow statistics
@test "comfyui::get_workflow_statistics provides workflow usage stats" {
    result=$(comfyui::get_workflow_statistics)
    
    [[ "$result" =~ "statistics" ]] || [[ "$result" =~ "stats" ]]
}

# Test workflow cleanup
@test "comfyui::cleanup_workflows removes old workflow files" {
    result=$(comfyui::cleanup_workflows)
    
    [[ "$result" =~ "cleanup" ]] || [[ "$result" =~ "cleaned" ]]
}

# Test workflow search
@test "comfyui::search_workflows finds workflows by criteria" {
    result=$(comfyui::search_workflows "text2img")
    
    [[ "$result" =~ "text2img" ]]
    [[ "$result" =~ "found" ]] || [[ "$result" =~ "search" ]]
}

# Test workflow categories
@test "comfyui::categorize_workflows organizes workflows by type" {
    result=$(comfyui::categorize_workflows)
    
    [[ "$result" =~ "category" ]] || [[ "$result" =~ "organize" ]]
}

# Test workflow recommendations
@test "comfyui::recommend_workflows suggests workflows" {
    result=$(comfyui::recommend_workflows "image generation")
    
    [[ "$result" =~ "recommend" ]] || [[ "$result" =~ "suggest" ]]
}