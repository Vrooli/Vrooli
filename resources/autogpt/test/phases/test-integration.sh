#!/usr/bin/env bash
################################################################################
# AutoGPT Integration Tests - End-to-end functionality (<120s)
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
APP_ROOT="$(cd "${RESOURCE_DIR}/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${RESOURCE_DIR}/lib/common.sh"

# Test configuration
AUTOGPT_PORT="${AUTOGPT_PORT_API:-8080}"
TEST_AGENT_NAME="test-agent-$(date +%s)"
TEST_AGENT_CONFIG="/tmp/autogpt-test-agent.yaml"

# Setup test environment
setup_test_env() {
    log::info "Setting up test environment"
    
    # Create test agent configuration
    cat > "${TEST_AGENT_CONFIG}" << EOF
name: ${TEST_AGENT_NAME}
description: Test agent for integration testing
goal: Generate a simple test report
model: gpt-3.5-turbo
max_iterations: 5
memory_backend: local
tools:
  - web_search
  - file_operations
EOF
    
    log::success "Test environment ready"
}

# Cleanup test environment
cleanup_test_env() {
    log::info "Cleaning up test environment"
    
    # Remove test agent if it exists
    vrooli resource autogpt content remove "${TEST_AGENT_NAME}" 2>/dev/null || true
    
    # Remove test config
    rm -f "${TEST_AGENT_CONFIG}"
    
    log::success "Test environment cleaned"
}

# Test agent creation
test_agent_creation() {
    log::test "Agent creation"
    
    # Add agent configuration
    if vrooli resource autogpt content add "${TEST_AGENT_CONFIG}"; then
        log::success "Agent configuration added"
        
        # Verify agent exists
        if vrooli resource autogpt content list | grep -q "${TEST_AGENT_NAME}"; then
            log::success "Agent appears in list"
            return 0
        else
            log::error "Agent not found in list"
            return 1
        fi
    else
        log::error "Failed to add agent configuration"
        return 1
    fi
}

# Test agent retrieval
test_agent_retrieval() {
    log::test "Agent retrieval"
    
    # Get agent details
    if vrooli resource autogpt content get "${TEST_AGENT_NAME}" > /dev/null 2>&1; then
        log::success "Agent details retrieved"
        return 0
    else
        log::error "Failed to retrieve agent details"
        return 1
    fi
}

# Test agent execution
test_agent_execution() {
    log::test "Agent execution"
    
    # Execute agent with simple task
    local output
    if output=$(vrooli resource autogpt content execute "${TEST_AGENT_NAME}" 2>&1); then
        log::success "Agent executed successfully"
        
        # Check if output contains expected elements
        if echo "${output}" | grep -qE "(task|complete|result)"; then
            log::success "Agent produced expected output"
            return 0
        else
            log::warning "Agent output lacks expected elements"
            return 1
        fi
    else
        log::error "Agent execution failed: ${output}"
        return 1
    fi
}

# Test API endpoints
test_api_endpoints() {
    log::test "API endpoints"
    
    local failed=0
    local base_url="http://localhost:${AUTOGPT_PORT}"
    
    # Test agent list endpoint
    if timeout 5 curl -sf "${base_url}/agents" > /dev/null 2>&1; then
        log::success "GET /agents endpoint works"
    else
        log::error "GET /agents endpoint failed"
        ((failed++))
    fi
    
    # Test health endpoint with details
    if timeout 5 curl -sf "${base_url}/health" | jq -e '.status' > /dev/null 2>&1; then
        log::success "Health endpoint returns status"
    else
        log::error "Health endpoint missing status field"
        ((failed++))
    fi
    
    return ${failed}
}

# Test Redis integration
test_redis_integration() {
    log::test "Redis integration"
    
    # Check if Redis is configured
    if [[ -n "${AUTOGPT_REDIS_HOST:-}" ]]; then
        # Test Redis connectivity from container
        if docker exec "${AUTOGPT_CONTAINER_NAME}" redis-cli -h "${AUTOGPT_REDIS_HOST}" ping > /dev/null 2>&1; then
            log::success "Redis connection successful"
            return 0
        else
            log::warning "Redis connection failed"
            return 1
        fi
    else
        log::info "Redis not configured - skipping"
        return 0
    fi
}

# Test LLM provider integration
test_llm_integration() {
    log::test "LLM provider integration"
    
    # Check configured provider
    local provider="${AUTOGPT_AI_PROVIDER:-none}"
    
    case "${provider}" in
        openrouter|ollama|openai)
            log::info "LLM provider configured: ${provider}"
            
            # Verify provider is accessible
            if vrooli resource autogpt status | grep -q "LLM Provider: ${provider}"; then
                log::success "LLM provider is configured"
                return 0
            else
                log::warning "LLM provider not properly configured"
                return 1
            fi
            ;;
        none)
            log::warning "No LLM provider configured"
            return 1
            ;;
        *)
            log::error "Unknown LLM provider: ${provider}"
            return 1
            ;;
    esac
}

# Test lifecycle management
test_lifecycle_management() {
    log::test "Lifecycle management"
    
    local failed=0
    
    # Test restart
    if vrooli resource autogpt manage restart; then
        log::success "Restart successful"
        
        # Wait for service to be ready
        sleep 5
        
        # Verify service is running after restart
        if timeout 5 curl -sf "http://localhost:${AUTOGPT_PORT}/health" > /dev/null 2>&1; then
            log::success "Service healthy after restart"
        else
            log::error "Service unhealthy after restart"
            ((failed++))
        fi
    else
        log::error "Restart failed"
        ((failed++))
    fi
    
    return ${failed}
}

# Main test execution
main() {
    log::header "AutoGPT Integration Tests"
    
    local failed=0
    
    # Setup
    setup_test_env
    
    # Ensure service is running
    if ! autogpt_container_running; then
        log::info "Starting AutoGPT service"
        vrooli resource autogpt manage start --wait || {
            log::error "Failed to start AutoGPT"
            cleanup_test_env
            return 1
        }
    fi
    
    # Run tests
    test_agent_creation || ((failed++))
    test_agent_retrieval || ((failed++))
    test_agent_execution || ((failed++))
    test_api_endpoints || ((failed++))
    test_redis_integration || ((failed++))
    test_llm_integration || ((failed++))
    test_lifecycle_management || ((failed++))
    
    # Cleanup
    cleanup_test_env
    
    # Summary
    if [[ ${failed} -eq 0 ]]; then
        log::success "All integration tests passed"
    else
        log::error "${failed} integration tests failed"
    fi
    
    return ${failed}
}

# Execute
main "$@"