#!/usr/bin/env bats
# Tests for validate.sh

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_integration_test "postgres" "qdrant" "redis" "ollama"
}

teardown() {
    vrooli_cleanup_test
}

@test "script should validate PostgreSQL database with tags table" {
    # Mock successful PostgreSQL validation
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::postgres::set_query_result "SELECT COUNT(*) FROM tags" "0"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "PostgreSQL database with tags"
}

@test "script should fail when PostgreSQL database is not configured" {
    # Mock PostgreSQL database validation failure
    mock::postgres::set_table_exists "prompt_manager" "tags" "false"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_failure
    assert_output_contains "PostgreSQL database not configured"
}

@test "script should validate Qdrant vector store configuration" {
    # Mock all previous checks as successful
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    # Mock Qdrant collection validation
    mock::qdrant::set_collection_exists "prompts" "true"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "Qdrant vector store configured"
}

@test "script should fail when Qdrant collection is not created" {
    # Mock PostgreSQL as successful but Qdrant as failed
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "false"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_failure
    assert_output_contains "Qdrant collection not created"
}

@test "script should validate Redis session store" {
    # Mock previous checks as successful
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    # Mock Redis validation
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "Redis session store"
}

@test "script should fail when Redis session store is not initialized" {
    # Mock previous checks as successful but Redis as failed
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "false"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_failure
    assert_output_contains "Redis session store not initialized"
}

@test "script should check Ollama availability (optional)" {
    # Mock successful scenario with Ollama available
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    mock::http::set_endpoint_response "http://localhost:8085/api/auth/login" "400" ""
    mock::http::set_endpoint_response "http://localhost:8085/api/prompts/semantic" "200" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "Ollama LLM available for testing"
}

@test "script should handle Ollama unavailability gracefully" {
    # Mock successful scenario with Ollama unavailable
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "false"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    mock::http::set_endpoint_response "http://localhost:8085/api/auth/login" "400" ""
    mock::http::set_endpoint_response "http://localhost:8085/api/prompts/semantic" "200" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "Ollama not available (prompt testing limited)"
}

@test "script should validate UI dashboard accessibility" {
    # Mock successful scenario up to UI check
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "UI dashboard accessible"
}

@test "script should fail when UI dashboard is not accessible" {
    # Mock successful scenario except UI
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "000" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_failure
    assert_output_contains "UI dashboard not accessible"
}

@test "script should validate API health" {
    # Mock successful scenario up to API health check
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "API healthy"
}

@test "script should fail when API is not healthy" {
    # Mock successful scenario except API health
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "500" "unhealthy"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_failure
    assert_output_contains "API not healthy"
}

@test "script should test authentication endpoint responsiveness" {
    # Mock full successful scenario
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    mock::http::set_endpoint_response "http://localhost:8085/api/auth/login" "401" ""
    mock::http::set_endpoint_response "http://localhost:8085/api/prompts/semantic" "200" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "Authentication endpoint responsive"
}

@test "script should test semantic search endpoint readiness" {
    # Mock full successful scenario
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    mock::http::set_endpoint_response "http://localhost:8085/api/auth/login" "401" ""
    mock::http::set_endpoint_response "http://localhost:8085/api/prompts/semantic" "200" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "Semantic search endpoint ready"
}

@test "script should complete full validation successfully" {
    # Mock all components as successful
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    mock::http::set_endpoint_response "http://localhost:8085/api/auth/login" "401" ""
    mock::http::set_endpoint_response "http://localhost:8085/api/prompts/semantic" "200" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    assert_output_contains "Prompt Manager validation successful"
    assert_output_contains "Campaign-based prompt management ready"
    assert_output_contains "Access the dashboard at: http://localhost:3005"
}

@test "script should use proper log functions" {
    # Mock basic successful scenario
    mock::postgres::set_table_exists "prompt_manager" "tags" "true"
    mock::qdrant::set_collection_exists "prompts" "true"
    mock::redis::set_key_exists "prompt_manager:initialized" "true"
    mock::ollama::set_available "true"
    mock::http::set_endpoint_response "http://localhost:3005" "200" ""
    mock::http::set_endpoint_response "http://localhost:8085/health" "200" "healthy"
    mock::http::set_endpoint_response "http://localhost:8085/api/auth/login" "401" ""
    mock::http::set_endpoint_response "http://localhost:8085/api/prompts/semantic" "200" ""
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_success
    # Verify log output format
    assert_output_contains "[INFO]"
    assert_output_contains "[SUCCESS]"
}

@test "script should exit early on first critical failure" {
    # Mock PostgreSQL failure (first check)
    mock::postgres::set_table_exists "prompt_manager" "tags" "false"
    
    run bash "${BATS_TEST_DIRNAME}/validate.sh"
    
    assert_failure
    assert_output_contains "PostgreSQL database not configured"
    # Should not continue to other checks
    refute_output_contains "Qdrant"
}