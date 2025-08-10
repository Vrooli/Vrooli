#!/usr/bin/env bats
# Tests for startup.sh

# Source test setup infrastructure
# shellcheck disable=SC1091
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Mock external dependencies
    mock::docker::reset
    
    # Create test scenario structure
    export TEST_SCENARIO_DIR="$VROOLI_TEST_TMPDIR/campaign-content-studio"
    mkdir -p "$TEST_SCENARIO_DIR/.vrooli"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/storage/postgres"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/automation/n8n"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/automation/windmill"
    mkdir -p "$TEST_SCENARIO_DIR/initialization/configuration"
    
    # Create test service.json
    cat > "$TEST_SCENARIO_DIR/.vrooli/service.json" <<EOF
{
  "resources": {
    "ai": {
      "ollama": {"required": true}
    },
    "automation": {
      "n8n": {"required": true},
      "windmill": {"required": true}
    },
    "storage": {
      "postgres": {"required": true},
      "qdrant": {"required": true},
      "minio": {"required": true}
    }
  },
  "deployment": {
    "testing": {
      "ui": {"required": true, "type": "windmill"},
      "timeout": "30m"
    }
  }
}
EOF
    
    # Mock commands that startup script uses
    curl() { echo "Mock curl response"; return 0; }
    jq() { command jq "$@"; }
    pg_isready() { echo "Mock pg_isready"; return 0; }
    createdb() { echo "Mock createdb"; return 0; }
    psql() { echo "Mock psql"; return 0; }
    export -f curl pg_isready createdb psql
}

teardown() {
    vrooli_cleanup_test
}

@test "startup::load_configuration parses service.json correctly" {
    # Source the script functions  
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Override SCENARIO_DIR for test
    SCENARIO_DIR="$TEST_SCENARIO_DIR"
    
    # Run the function
    run startup::load_configuration
    
    # Verify execution
    assert_success
    assert_output_contains "Loading scenario configuration"
}

@test "startup::load_configuration handles missing service.json" {
    # Source the script functions
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Set invalid scenario dir
    SCENARIO_DIR="$VROOLI_TEST_TMPDIR/nonexistent"
    
    # Run the function
    run startup::load_configuration
    
    # Should fail with error
    assert_failure
    assert_output_contains "service.json not found"
}

@test "startup::validate_resources checks service health" {
    # Source the script functions with mocks
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Mock resources::get_default_port if it exists
    resources::get_default_port() { echo "8080"; }
    export -f resources::get_default_port || true
    
    # Set up test data
    REQUIRED_RESOURCES="ollama n8n postgres qdrant minio"
    SCENARIO_DIR="$TEST_SCENARIO_DIR"
    LOG_FILE="$VROOLI_TEST_TMPDIR/startup.log"
    
    # Run the function
    run startup::validate_resources
    
    # Should succeed with mocked services
    assert_success
    assert_output_contains "Validating required resources"
}

@test "startup::initialize_database creates database and applies schema" {
    # Source the script functions
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Mock resources::get_default_port if it exists
    resources::get_default_port() { echo "5432"; }
    export -f resources::get_default_port || true
    
    # Create test schema file
    cat > "$TEST_SCENARIO_DIR/initialization/storage/postgres/schema.sql" <<EOF
CREATE TABLE IF NOT EXISTS campaigns (id SERIAL PRIMARY KEY);
EOF
    
    # Set up test data
    REQUIRED_RESOURCES="postgres"
    SCENARIO_DIR="$TEST_SCENARIO_DIR"
    SCENARIO_ID="campaign-content-studio"
    LOG_FILE="$VROOLI_TEST_TMPDIR/startup.log"
    
    # Run the function
    run startup::initialize_database
    
    # Should succeed 
    assert_success
    assert_output_contains "Initializing database"
}

@test "startup::deploy_workflows validates JSON files" {
    # Source the script functions
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Create test workflow files
    cat > "$TEST_SCENARIO_DIR/initialization/automation/n8n/campaign-workflow.json" <<EOF
{"name": "campaign-workflow", "nodes": []}
EOF
    
    cat > "$TEST_SCENARIO_DIR/initialization/automation/windmill/dashboard-app.json" <<EOF
{"name": "campaign-dashboard", "version": "1.0"}
EOF
    
    # Set up test data
    REQUIRED_RESOURCES="n8n windmill"
    REQUIRES_UI="true"
    SCENARIO_DIR="$TEST_SCENARIO_DIR"
    LOG_FILE="$VROOLI_TEST_TMPDIR/startup.log"
    
    # Run the function
    run startup::deploy_workflows
    
    # Should succeed and validate JSON
    assert_success
    assert_output_contains "Deploying workflows"
    assert_output_contains "campaign-workflow.json is valid"
}

@test "startup script sources var.sh correctly" {
    # Test that var.sh is sourced and variables are available
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Check that var_ variables are available
    [[ -n "$var_ROOT_DIR" ]]
    [[ -n "$var_LIB_UTILS_DIR" ]]
}

@test "startup script handles command line arguments" {
    # Test help command
    run bash "$BATS_TEST_DIRNAME/startup.sh" help
    assert_success
    assert_output_contains "Usage:"
    assert_output_contains "Commands:"
    
    # Test unknown command
    run bash "$BATS_TEST_DIRNAME/startup.sh" unknown
    assert_failure
    assert_output_contains "Unknown command: unknown"
}