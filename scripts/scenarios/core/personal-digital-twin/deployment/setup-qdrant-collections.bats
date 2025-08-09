#!/usr/bin/env bats
# Tests for setup-qdrant-collections.sh

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_service_test "qdrant"
    
    # Mock environment variables
    export QDRANT_HOST="localhost"
    export QDRANT_PORT="6333"
}

teardown() {
    vrooli_cleanup_test
}

# Function to simulate the setup-qdrant-collections.sh script behavior
run_setup_script() {
    # Mock the script path resolution 
    export SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    
    # Source the script and call create_collection function directly for testing
    source "${BATS_TEST_DIRNAME}/setup-qdrant-collections.sh"
}

@test "create_collection function should create new collection" {
    run_setup_script
    
    # Mock that collection doesn't exist
    mock::qdrant::set_collection_exists "memories" "false"
    # Mock successful collection creation
    mock::qdrant::set_collection_creatable "memories" "true"
    
    run create_collection "memories" "768" "Cosine"
    
    assert_success
    assert_output_contains "memories created successfully"
}

@test "create_collection function should skip existing collection" {
    run_setup_script
    
    # Mock that collection already exists
    mock::qdrant::set_collection_exists "memories" "true"
    
    run create_collection "memories" "768" "Cosine"
    
    assert_success
    assert_output_contains "memories already exists, skipping"
}

@test "create_collection function should handle creation failure" {
    run_setup_script
    
    # Mock that collection doesn't exist but creation fails
    mock::qdrant::set_collection_exists "memories" "false"
    mock::qdrant::set_collection_creatable "memories" "false"
    
    run create_collection "memories" "768" "Cosine"
    
    assert_failure
    assert_output_contains "Failed to create collection memories"
}

@test "script should wait for Qdrant to be ready" {
    # Mock that Qdrant is initially not ready, then becomes ready
    mock::qdrant::set_ready "false"
    mock::qdrant::set_ready_after_attempts "3"
    
    run timeout 10s bash "${BATS_TEST_DIRNAME}/setup-qdrant-collections.sh"
    
    assert_success
    assert_output_contains "Qdrant is ready"
}

@test "script should create all required collections" {
    # Mock that Qdrant is ready
    mock::qdrant::set_ready "true"
    
    # Mock that collections don't exist
    mock::qdrant::set_collection_exists "memories" "false"
    mock::qdrant::set_collection_exists "documents" "false"
    mock::qdrant::set_collection_exists "conversations" "false"
    
    # Mock successful creation for all collections
    mock::qdrant::set_collection_creatable "memories" "true"
    mock::qdrant::set_collection_creatable "documents" "true"
    mock::qdrant::set_collection_creatable "conversations" "true"
    
    # Mock payload schema configuration
    mock::qdrant::set_payload_schema_updatable "memories" "true"
    mock::qdrant::set_payload_schema_updatable "documents" "true"
    mock::qdrant::set_payload_schema_updatable "conversations" "true"
    
    run bash "${BATS_TEST_DIRNAME}/setup-qdrant-collections.sh"
    
    assert_success
    assert_output_contains "Creating collection: memories"
    assert_output_contains "Creating collection: documents"
    assert_output_contains "Creating collection: conversations"
    assert_output_contains "Qdrant collections setup complete"
}

@test "script should configure payload schemas" {
    # Mock successful collection setup
    mock::qdrant::set_ready "true"
    mock::qdrant::set_collection_exists "memories" "false"
    mock::qdrant::set_collection_exists "documents" "false"
    mock::qdrant::set_collection_exists "conversations" "false"
    mock::qdrant::set_collection_creatable "memories" "true"
    mock::qdrant::set_collection_creatable "documents" "true"
    mock::qdrant::set_collection_creatable "conversations" "true"
    
    # Mock payload schema configuration
    mock::qdrant::set_payload_schema_updatable "memories" "true"
    mock::qdrant::set_payload_schema_updatable "documents" "true"
    mock::qdrant::set_payload_schema_updatable "conversations" "true"
    
    run bash "${BATS_TEST_DIRNAME}/setup-qdrant-collections.sh"
    
    assert_success
    assert_output_contains "Configuring payload schemas"
}

@test "script should use correct vector dimensions and distance metrics" {
    run_setup_script
    
    # Mock collection creation with specific parameters
    mock::qdrant::set_collection_exists "memories" "false"
    mock::qdrant::expect_collection_parameters "memories" "768" "Cosine"
    
    run create_collection "memories" "768" "Cosine"
    
    assert_success
    # Verify correct parameters were used (this would be validated by the mock)
}

@test "script should handle mixed collection states (some exist, some don't)" {
    # Mock that Qdrant is ready
    mock::qdrant::set_ready "true"
    
    # Mock mixed states: memories exists, others don't
    mock::qdrant::set_collection_exists "memories" "true"
    mock::qdrant::set_collection_exists "documents" "false"
    mock::qdrant::set_collection_exists "conversations" "false"
    
    # Mock successful creation for non-existing collections
    mock::qdrant::set_collection_creatable "documents" "true"
    mock::qdrant::set_collection_creatable "conversations" "true"
    
    # Mock payload schema configuration
    mock::qdrant::set_payload_schema_updatable "memories" "true"
    mock::qdrant::set_payload_schema_updatable "documents" "true"
    mock::qdrant::set_payload_schema_updatable "conversations" "true"
    
    run bash "${BATS_TEST_DIRNAME}/setup-qdrant-collections.sh"
    
    assert_success
    assert_output_contains "memories already exists, skipping"
    assert_output_contains "documents created successfully"
    assert_output_contains "conversations created successfully"
}

@test "script should use proper log functions" {
    # Mock successful scenario
    mock::qdrant::set_ready "true"
    mock::qdrant::set_collection_exists "memories" "false"
    mock::qdrant::set_collection_creatable "memories" "true"
    mock::qdrant::set_payload_schema_updatable "memories" "true"
    
    run bash "${BATS_TEST_DIRNAME}/setup-qdrant-collections.sh"
    
    assert_success
    # Verify log output format
    assert_output_contains "[INFO]"
    assert_output_contains "[SUCCESS]"
}