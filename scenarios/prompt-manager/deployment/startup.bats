#!/usr/bin/env bats
# Tests for startup.sh

bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/prompt-manager/deployment"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_integration_test "postgres" "qdrant" "redis" "ollama"
    
    # Set up test scenario directory structure
    export TEST_SCENARIO_DIR="${BATS_TEST_TMPDIR}/prompt-manager"
    export TEST_POSTGRES_DIR="${TEST_SCENARIO_DIR}/initialization/storage/postgres"
    mkdir -p "${TEST_POSTGRES_DIR}"
    
    # Create sample schema file
    cat > "${TEST_POSTGRES_DIR}/schema.sql" <<'EOF'
CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS prompts (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
EOF
}

teardown() {
    vrooli_cleanup_test
}

@test "script should initialize PostgreSQL database schema" {
    # Mock that schema file exists and is executable
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Database schema initialized"
}

@test "script should handle missing schema file gracefully" {
    # Remove schema file
    trash::safe_remove "${TEST_POSTGRES_DIR}/schema.sql" --test-cleanup
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Schema file not found"
}

@test "script should check Qdrant vector database availability" {
    # Mock Qdrant as available
    mock::qdrant::set_available "true"
    mock::qdrant::set_collection_creatable "prompts" "true"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Qdrant vector database is available"
    assert_output_contains "Qdrant collection created"
}

@test "script should handle Qdrant unavailability gracefully" {
    # Mock Qdrant as unavailable
    mock::qdrant::set_available "false"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Qdrant not available - vector operations will be limited"
}

@test "script should check Ollama availability" {
    # Mock Ollama as available
    mock::ollama::set_available "true"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Ollama is available for prompt testing"
}

@test "script should handle Ollama unavailability gracefully" {
    # Mock Ollama as unavailable
    mock::ollama::set_available "false"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Ollama not available - prompt testing will be limited"
}

@test "script should initialize Redis session store" {
    # Mock Redis as available
    mock::redis::set_available "true"
    mock::redis::set_key_settable "prompt_manager:initialized" "true"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Redis session store initialized"
}

@test "script should handle Redis initialization failure" {
    # Mock Redis as unavailable
    mock::redis::set_available "false"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Redis initialization failed - sessions may not work properly"
}

@test "script should perform API health check" {
    # Mock API as healthy
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "API health check passed"
}

@test "script should handle API unavailability" {
    # Mock API as unavailable
    mock::http::set_endpoint_response "http://localhost:8085/health" "000" ""
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "API not yet available"
}

@test "script should complete startup process successfully" {
    # Mock all services as available
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    mock::qdrant::set_available "true"
    mock::qdrant::set_collection_creatable "prompts" "true"
    mock::ollama::set_available "true"
    mock::redis::set_available "true"
    mock::redis::set_key_settable "prompt_manager:initialized" "true"
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Prompt Manager startup process completed"
    assert_output_contains "Dashboard should be available at: http://localhost:3005"
    assert_output_contains "Default credentials"
}

@test "script should use proper log functions" {
    # Mock basic successful scenario
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "true"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    # Verify log output format
    assert_output_contains "[INFO]"
    assert_output_contains "[SUCCESS]"
}

@test "script should provide default credentials information" {
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Email: admin@promptmanager.local"
    assert_output_contains "Password: ChangeMeNow123!"
}

@test "script should handle database initialization warnings appropriately" {
    # Mock database initialization as failing (already exists scenario)
    mock::postgres::set_schema_executable "${TEST_POSTGRES_DIR}/schema.sql" "false"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    assert_output_contains "Database initialization failed or already exists"
}

@test "script should create Qdrant collection with correct parameters" {
    # Mock Qdrant availability and expect specific collection parameters
    mock::qdrant::set_available "true"
    mock::qdrant::expect_collection_parameters "prompts" "1536" "Cosine"
    
    run bash "${BATS_TEST_DIRNAME}/startup.sh"
    
    assert_success
    # Verify correct parameters were used (this would be validated by the mock)
}