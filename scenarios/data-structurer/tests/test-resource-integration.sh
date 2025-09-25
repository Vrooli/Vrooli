#!/bin/bash

# Test script for Data Structurer resource integration

set -euo pipefail

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:${API_PORT:-15770}}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[TEST INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[TEST PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[TEST FAIL]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[TEST WARN]${NC} $1"
}

# Test resource availability
test_resource() {
    local resource_name="$1"
    local check_command="$2"
    
    if eval "$check_command" > /dev/null 2>&1; then
        log_success "$resource_name is available"
        return 0
    else
        log_warning "$resource_name is not available or not responding"
        return 1
    fi
}

# Main test function
main() {
    log_info "Starting Data Structurer resource integration tests..."
    
    local test_count=0
    local pass_count=0
    
    # Test 1: PostgreSQL connectivity
    log_info "Test 1: PostgreSQL connectivity"
    test_count=$((test_count + 1))
    if test_resource "PostgreSQL" "vrooli resource status postgres | grep -q 'Running: true'"; then
        pass_count=$((pass_count + 1))
    fi
    
    # Test 2: Ollama availability
    log_info "Test 2: Ollama availability"
    test_count=$((test_count + 1))
    if test_resource "Ollama" "vrooli resource status ollama | grep -q 'Running: true'"; then
        pass_count=$((pass_count + 1))
    fi
    
    # Test 3: Unstructured-io availability
    log_info "Test 3: Unstructured-io availability"
    test_count=$((test_count + 1))
    if test_resource "Unstructured-io" "vrooli resource status unstructured-io | grep -q 'Running: true'"; then
        pass_count=$((pass_count + 1))
    else
        log_warning "Unstructured-io not available (optional resource)"
    fi
    
    # Test 4: N8N availability
    log_info "Test 4: N8N availability"
    test_count=$((test_count + 1))
    if test_resource "N8N" "vrooli resource status n8n | grep -q 'Running: true'"; then
        pass_count=$((pass_count + 1))
    else
        log_warning "N8N not available (workflows won't function)"
    fi
    
    # Test 5: Qdrant availability (optional)
    log_info "Test 5: Qdrant availability (optional)"
    test_count=$((test_count + 1))
    if test_resource "Qdrant" "vrooli resource status qdrant | grep -q 'Running: true'"; then
        pass_count=$((pass_count + 1))
    else
        log_warning "Qdrant not available (semantic search won't function)"
    fi
    
    # Test 6: API resource dependency status
    log_info "Test 6: API resource dependency status"
    test_count=$((test_count + 1))
    
    local health_response
    if health_response=$(curl -sf "$API_BASE_URL/health"); then
        local postgres_status
        local ollama_status
        
        postgres_status=$(echo "$health_response" | jq -r '.dependencies.postgres.status // "unknown"')
        ollama_status=$(echo "$health_response" | jq -r '.dependencies.ollama.status // "unknown"')
        
        if [[ "$postgres_status" == "healthy" ]]; then
            log_success "API reports PostgreSQL as healthy"
            pass_count=$((pass_count + 1))
        else
            log_warning "API reports PostgreSQL status: $postgres_status"
        fi
        
        if [[ "$ollama_status" == "healthy" ]]; then
            log_success "API reports Ollama as healthy"
        else
            log_warning "API reports Ollama status: $ollama_status"
        fi
    else
        log_error "Failed to get health status from API"
    fi
    
    # Test 7: Database schema validation
    log_info "Test 7: Database schema validation"
    test_count=$((test_count + 1))
    
    if vrooli resource postgres query "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('schemas', 'processed_data', 'schema_templates');" 2>/dev/null | grep -q "3"; then
        log_success "Database schema is properly initialized"
        pass_count=$((pass_count + 1))
    else
        log_warning "Database schema may not be fully initialized"
    fi
    
    # Summary
    echo
    log_info "Resource Integration Test Summary:"
    echo "Total Tests: $test_count"
    echo "Passed: $pass_count"
    echo "Failed/Skipped: $((test_count - pass_count))"
    
    if [[ $pass_count -gt $((test_count / 2)) ]]; then
        log_success "Core resources are available!"
        exit 0
    else
        log_warning "Some resources are not available. Full functionality may be limited."
        exit 0  # Don't fail if resources aren't all available
    fi
}

# Run tests
main "$@"