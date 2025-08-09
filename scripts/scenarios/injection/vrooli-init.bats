#!/usr/bin/env bats
# Tests for Vrooli Self-Initialization Script (vrooli-init.sh)

# Load test setup
# shellcheck disable=SC1091
source "$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)/../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Set up test environment for vrooli-init
    export TEST_SCENARIOS_DIR="${VROOLI_TEST_TMPDIR}/scenarios"
    export TEST_INIT_STATE_FILE="${VROOLI_TEST_TMPDIR}/.vrooli/.initialization-state.json"
    
    # Create test scenarios directory structure
    mkdir -p "${TEST_SCENARIOS_DIR}/test-scenario"
    mkdir -p "${TEST_SCENARIOS_DIR}/another-scenario"
    
    # Create service.json for test scenario
    cat > "${TEST_SCENARIOS_DIR}/test-scenario/service.json" << 'EOF'
{
  "spec": {
    "dependencies": {
      "resources": [
        {
          "name": "n8n",
          "version": "latest"
        },
        {
          "name": "postgres",
          "version": "15"
        }
      ]
    }
  }
}
EOF

    # Create service.json for another scenario (doesn't use target resources)
    cat > "${TEST_SCENARIOS_DIR}/another-scenario/service.json" << 'EOF'
{
  "spec": {
    "dependencies": {
      "resources": [
        {
          "name": "redis",
          "version": "latest"
        }
      ]
    }
  }
}
EOF

    # Create test workflow files
    mkdir -p "${TEST_SCENARIOS_DIR}/test-scenario/initialization/workflows/n8n"
    cat > "${TEST_SCENARIOS_DIR}/test-scenario/initialization/workflows/n8n/test-workflow.json" << 'EOF'
{
  "name": "test-workflow",
  "active": true,
  "nodes": []
}
EOF

    # Create test database files
    mkdir -p "${TEST_SCENARIOS_DIR}/test-scenario/initialization/database"
    cat > "${TEST_SCENARIOS_DIR}/test-scenario/initialization/database/schema.sql" << 'EOF'
CREATE TABLE test_table (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255)
);
EOF

    # Create test configuration files
    mkdir -p "${TEST_SCENARIOS_DIR}/test-scenario/initialization/configuration"
    cat > "${TEST_SCENARIOS_DIR}/test-scenario/initialization/configuration/storage-config.json" << 'EOF'
{
  "minio": {
    "buckets": [
      {
        "name": "test-bucket",
        "policy": "private"
      }
    ]
  }
}
EOF

    cat > "${TEST_SCENARIOS_DIR}/test-scenario/initialization/configuration/ai-config.json" << 'EOF'
{
  "ollama": {
    "models": [
      "llama2",
      "codellama"
    ]
  }
}
EOF

    # Override SCENARIOS_DIR to point to our test directory
    SCENARIOS_DIR="${TEST_SCENARIOS_DIR}"
    
    # Create initialization state directory
    mkdir -p "$(dirname "${TEST_INIT_STATE_FILE}")"
}

teardown() {
    vrooli_cleanup_test
}

@test "vrooli_init::parse_args handles all options correctly" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    parse_args --scenario test-scenario --resources n8n,postgres --dry-run --verbose --force
    
    [ "${SCENARIO_NAME}" = "test-scenario" ]
    [ "${RESOURCES}" = "n8n,postgres" ]
    [ "${DRY_RUN}" = true ]
    [ "${VERBOSE}" = true ]
    [ "${FORCE}" = true ]
}

@test "vrooli_init::parse_args sets default values" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    parse_args
    
    [ "${DRY_RUN}" = false ]
    [ "${VERBOSE}" = false ]
    [ "${FORCE}" = false ]
    [ -z "${SCENARIO_NAME}" ]
    [ -z "${RESOURCES}" ]
}

@test "vrooli_init::get_scenarios returns specific scenario when requested" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    SCENARIO_NAME="test-scenario"
    
    run get_scenarios
    assert_success
    assert_output "test-scenario"
}

@test "vrooli_init::get_scenarios returns all scenarios when none specified" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    SCENARIO_NAME=""
    
    run get_scenarios
    assert_success
    assert_output --partial "test-scenario"
    assert_output --partial "another-scenario"
}

@test "vrooli_init::get_scenarios fails for non-existent scenario" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    SCENARIO_NAME="non-existent-scenario"
    
    run get_scenarios
    assert_failure
    assert_output --partial "Scenario not found"
}

@test "vrooli_init::get_resources returns specific resources when requested" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    RESOURCES="n8n,postgres"
    
    run get_resources
    assert_success
    assert_output "n8n postgres"
}

@test "vrooli_init::get_resources returns default resources when none specified" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    RESOURCES=""
    
    run get_resources
    assert_success
    assert_output --partial "n8n"
    assert_output --partial "postgres"
}

@test "vrooli_init::check_init_state returns false when already initialized" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    # Create init state file showing resource is already initialized
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    mkdir -p "$(dirname "${INIT_STATE_FILE}")"
    cat > "${INIT_STATE_FILE}" << 'EOF'
{
  "scenarios": {
    "test-scenario": {
      "resources": {
        "n8n": true
      }
    }
  }
}
EOF
    
    run check_init_state "test-scenario" "n8n"
    assert_failure  # Function returns 1 when already initialized
}

@test "vrooli_init::check_init_state returns true when not initialized" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    
    run check_init_state "test-scenario" "n8n"
    assert_success  # Function returns 0 when not initialized
}

@test "vrooli_init::check_init_state respects force flag" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    FORCE=true
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    
    run check_init_state "test-scenario" "n8n"
    assert_success  # Force flag bypasses init state check
}

@test "vrooli_init::update_init_state creates state file and updates it" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    DRY_RUN=false
    
    update_init_state "test-scenario" "n8n"
    
    # Verify state file was created and updated
    [ -f "${INIT_STATE_FILE}" ]
    local state=$(jq -r '.scenarios["test-scenario"].resources.n8n' "${INIT_STATE_FILE}")
    [ "$state" = "true" ]
}

@test "vrooli_init::update_init_state skips in dry run mode" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    DRY_RUN=true
    
    run update_init_state "test-scenario" "n8n"
    assert_success
    assert_output --partial "[DRY RUN]"
    
    # Verify no state file was created
    [ ! -f "${INIT_STATE_FILE}" ]
}

@test "vrooli_init::init_n8n processes workflows correctly" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    DRY_RUN=true  # Use dry run to avoid actual file operations
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    
    run init_n8n "test-scenario"
    assert_success
    assert_output --partial "Initializing n8n workflows from: test-scenario"
    assert_output --partial "Importing workflow: test-workflow"
    assert_output --partial "[DRY RUN]"
}

@test "vrooli_init::init_n8n skips when no workflow directory exists" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    VERBOSE=true
    
    run init_n8n "another-scenario"  # This scenario has no n8n workflows
    assert_success
    assert_output --partial "No n8n workflows in scenario: another-scenario"
}

@test "vrooli_init::init_postgres processes database files correctly" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    DRY_RUN=true
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    
    run init_postgres "test-scenario"
    assert_success
    assert_output --partial "Initializing PostgreSQL data from: test-scenario"
    assert_output --partial "[DRY RUN] Would create schema"
    assert_output --partial "[DRY RUN] Would run: schema.sql"
}

@test "vrooli_init::init_minio processes storage configuration correctly" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    DRY_RUN=true
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    
    run init_minio "test-scenario"
    assert_success
    assert_output --partial "Initializing MinIO from: test-scenario"
    assert_output --partial "Creating bucket: test-bucket"
    assert_output --partial "[DRY RUN] Would create bucket"
}

@test "vrooli_init::init_ollama processes AI configuration correctly" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    DRY_RUN=true
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    
    run init_ollama "test-scenario"
    assert_success
    assert_output --partial "Initializing Ollama models from: test-scenario"
    assert_output --partial "Pulling model: llama2"
    assert_output --partial "Pulling model: codellama"
    assert_output --partial "[DRY RUN] Would pull model"
}

@test "vrooli_init::init_resource delegates to correct resource function" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    DRY_RUN=true
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    FORCE=true  # Force to bypass init state check
    
    run init_resource "test-scenario" "n8n"
    assert_success
    assert_output --partial "Initializing n8n workflows"
}

@test "vrooli_init::init_resource skips already initialized resources" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    # Create init state showing resource is already initialized
    INIT_STATE_FILE="${TEST_INIT_STATE_FILE}"
    mkdir -p "$(dirname "${INIT_STATE_FILE}")"
    cat > "${INIT_STATE_FILE}" << 'EOF'
{
  "scenarios": {
    "test-scenario": {
      "resources": {
        "n8n": true
      }
    }
  }
}
EOF
    
    run init_resource "test-scenario" "n8n"
    assert_success
    assert_output --partial "Skipping already initialized"
}

@test "vrooli_init::init_resource handles unsupported resources" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    FORCE=true
    
    run init_resource "test-scenario" "unsupported-resource"
    assert_success
    assert_output --partial "Unsupported resource for self-init"
}

@test "vrooli_init::main processes scenarios and resources correctly" {
    source "${BATS_TEST_DIRNAME}/vrooli-init.sh"
    
    # Override directory variables to use test directories
    SCENARIOS_DIR="${TEST_SCENARIOS_DIR}"
    DRY_RUN=true
    FORCE=true
    
    run main --dry-run --force --scenario test-scenario --resources n8n,postgres
    assert_success
    assert_output --partial "Vrooli Self-Initialization"
    assert_output --partial "DRY RUN MODE"
    assert_output --partial "Processing Scenario: test-scenario"
    assert_output --partial "Initializing n8n from test-scenario"
    assert_output --partial "Initializing postgres from test-scenario"
}