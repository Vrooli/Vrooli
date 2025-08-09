#!/usr/bin/env bats
# Tests for deploy-windmill-apps.sh

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_service_test "windmill"
    
    # Set up test scenario directory structure
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/personal-digital-twin"
    export TEST_APPS_DIR="${TEST_SCENARIO_DIR}/initialization/automation/windmill"
    mkdir -p "${TEST_APPS_DIR}"
    
    # Create sample app files
    cat > "${TEST_APPS_DIR}/persona-manager.json" <<'EOF'
{
  "summary": "Persona Management App",
  "value": {
    "type": "app"
  }
}
EOF
    
    cat > "${TEST_APPS_DIR}/data-source-config.json" <<'EOF'
{
  "summary": "Data Source Configuration",
  "value": {
    "type": "config"
  }
}
EOF
    
    # Mock environment variables
    export WINDMILL_HOST="localhost"
    export WINDMILL_PORT="5681"
    export WINDMILL_WORKSPACE="demo"
}

teardown() {
    vrooli_cleanup_test
}

# Function to simulate the deploy-windmill-apps.sh script behavior
run_deploy_script() {
    # Mock the script path resolution 
    export SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SCENARIO_DIR="${TEST_SCENARIO_DIR}"
    
    # Source the script and call deploy_app function directly for testing
    source "${BATS_TEST_DIRNAME}/deploy-windmill-apps.sh"
}

@test "deploy_app function should deploy valid app file" {
    run_deploy_script
    
    # Mock successful API response
    mock::windmill::set_app_deployable "persona-manager" "true"
    
    run deploy_app "${TEST_APPS_DIR}/persona-manager.json"
    
    assert_success
    assert_output_contains "persona-manager deployed successfully"
}

@test "deploy_app function should handle missing app file" {
    run_deploy_script
    
    run deploy_app "/nonexistent/app.json"
    
    assert_failure
    assert_output_contains "App file not found"
}

@test "deploy_app function should update existing app on conflict" {
    run_deploy_script
    
    # Mock initial deployment failure (conflict)
    mock::windmill::inject_error "create" "conflict"
    # Mock successful update
    mock::windmill::set_app_updatable "persona-manager" "true"
    
    run deploy_app "${TEST_APPS_DIR}/persona-manager.json"
    
    assert_success
    assert_output_contains "persona-manager updated successfully"
}

@test "script should process all JSON files in windmill directory" {
    # Mock that windmill is ready
    mock::windmill::set_ready "true"
    
    # Mock successful deployments
    mock::windmill::set_app_deployable "persona-manager" "true"
    mock::windmill::set_app_deployable "data-source-config" "true"
    
    # Run the actual script
    run bash "${BATS_TEST_DIRNAME}/deploy-windmill-apps.sh"
    
    assert_success
    assert_output_contains "Windmill app deployment complete"
}

@test "script should wait for Windmill to be ready" {
    # Mock that windmill is initially not ready, then becomes ready
    mock::windmill::set_ready "false"
    mock::windmill::set_ready_after_attempts "3"
    
    run timeout 10s bash "${BATS_TEST_DIRNAME}/deploy-windmill-apps.sh"
    
    assert_success
    assert_output_contains "Windmill is ready"
}

@test "script should handle missing apps directory gracefully" {
    # Remove the apps directory
    rm -rf "${TEST_APPS_DIR}"
    
    run bash "${BATS_TEST_DIRNAME}/deploy-windmill-apps.sh"
    
    assert_failure
    assert_output_contains "Apps directory not found"
}

@test "script should use proper log functions" {
    # Mock successful scenario
    mock::windmill::set_ready "true"
    mock::windmill::set_app_deployable "persona-manager" "true"
    mock::windmill::set_app_deployable "data-source-config" "true"
    
    run bash "${BATS_TEST_DIRNAME}/deploy-windmill-apps.sh"
    
    assert_success
    # Verify log output format
    assert_output_contains "[INFO]"
    assert_output_contains "[SUCCESS]"
}