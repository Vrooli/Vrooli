#!/usr/bin/env bats
# Tests for import-n8n-workflows.sh

bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_service_test "n8n"
    
    # Set up test scenario directory structure
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/personal-digital-twin"
    export TEST_WORKFLOWS_DIR="${TEST_SCENARIO_DIR}/initialization/automation/n8n"
    mkdir -p "${TEST_WORKFLOWS_DIR}"
    
    # Create sample workflow files
    cat > "${TEST_WORKFLOWS_DIR}/data-ingestion.json" <<'EOF'
{
  "name": "Data Ingestion Pipeline",
  "nodes": [
    {
      "name": "Start",
      "type": "n8n-nodes-base.start"
    }
  ],
  "connections": {}
}
EOF
    
    cat > "${TEST_WORKFLOWS_DIR}/model-training.json" <<'EOF'
{
  "name": "Model Training Workflow",
  "nodes": [
    {
      "name": "Trigger",
      "type": "n8n-nodes-base.cron"
    }
  ],
  "connections": {}
}
EOF
    
    # Mock environment variables
    export N8N_HOST="localhost"
    export N8N_PORT="5678"
}

teardown() {
    vrooli_cleanup_test
}

# Function to simulate the import-n8n-workflows.sh script behavior
run_import_script() {
    # Mock the script path resolution 
    export SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SCENARIO_DIR="${TEST_SCENARIO_DIR}"
    
    # Source the script and call import_workflow function directly for testing
    source "${BATS_TEST_DIRNAME}/import-n8n-workflows.sh"
}

@test "import_workflow function should import new workflow" {
    run_import_script
    
    # Mock that workflow doesn't exist
    mock::n8n::set_workflow_exists "data-ingestion" "false"
    # Mock successful creation
    mock::n8n::set_workflow_creatable "data-ingestion" "true"
    
    run import_workflow "${TEST_WORKFLOWS_DIR}/data-ingestion.json"
    
    assert_success
    assert_output_contains "data-ingestion imported successfully"
}

@test "import_workflow function should update existing workflow" {
    run_import_script
    
    # Mock that workflow exists
    mock::n8n::set_workflow_exists "data-ingestion" "true"
    mock::n8n::set_workflow_id "data-ingestion" "123"
    # Mock successful update
    mock::n8n::set_workflow_updatable "data-ingestion" "true"
    
    run import_workflow "${TEST_WORKFLOWS_DIR}/data-ingestion.json"
    
    assert_success
    assert_output_contains "data-ingestion updated successfully"
}

@test "import_workflow function should handle missing workflow file" {
    run_import_script
    
    run import_workflow "/nonexistent/workflow.json"
    
    assert_failure
    assert_output_contains "Workflow file not found"
}

@test "script should process all JSON files in n8n directory" {
    # Mock that n8n is ready
    mock::n8n::set_ready "true"
    
    # Mock successful workflow operations
    mock::n8n::set_workflow_exists "data-ingestion" "false"
    mock::n8n::set_workflow_creatable "data-ingestion" "true"
    mock::n8n::set_workflow_exists "model-training" "false" 
    mock::n8n::set_workflow_creatable "model-training" "true"
    
    # Mock workflow activation
    mock::n8n::set_workflow_activatable "123" "true"
    mock::n8n::set_workflow_activatable "456" "true"
    
    run bash "${BATS_TEST_DIRNAME}/import-n8n-workflows.sh"
    
    assert_success
    assert_output_contains "n8n workflow import complete"
}

@test "script should wait for n8n to be ready" {
    # Mock that n8n is initially not ready, then becomes ready
    mock::n8n::set_ready "false"
    mock::n8n::set_ready_after_attempts "3"
    
    run timeout 10s bash "${BATS_TEST_DIRNAME}/import-n8n-workflows.sh"
    
    assert_success
    assert_output_contains "n8n is ready"
}

@test "script should activate workflows after import" {
    # Mock successful import scenario
    mock::n8n::set_ready "true"
    mock::n8n::set_workflow_exists "data-ingestion" "false"
    mock::n8n::set_workflow_creatable "data-ingestion" "true"
    mock::n8n::set_workflow_list_ids "123 456"
    mock::n8n::set_workflow_activatable "123" "true"
    mock::n8n::set_workflow_activatable "456" "true"
    
    run bash "${BATS_TEST_DIRNAME}/import-n8n-workflows.sh"
    
    assert_success
    assert_output_contains "Activating imported workflows"
    assert_output_contains "Activated workflow ID: 123"
}

@test "script should handle missing workflows directory gracefully" {
    # Remove the workflows directory
    trash::safe_remove "${TEST_WORKFLOWS_DIR}" --test-cleanup
    
    run bash "${BATS_TEST_DIRNAME}/import-n8n-workflows.sh"
    
    assert_failure
    assert_output_contains "Workflows directory not found"
}

@test "script should use proper log functions" {
    # Mock successful scenario
    mock::n8n::set_ready "true"
    mock::n8n::set_workflow_exists "data-ingestion" "false"
    mock::n8n::set_workflow_creatable "data-ingestion" "true"
    
    run bash "${BATS_TEST_DIRNAME}/import-n8n-workflows.sh"
    
    assert_success
    # Verify log output format
    assert_output_contains "[INFO]"
    assert_output_contains "[SUCCESS]"
}