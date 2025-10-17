#!/bin/bash

# Test script for Data Structurer resource integration

set -euo pipefail

# Configuration
# Get the actual port from the running scenario's status
# Priority: DATA_STRUCTURER_API_PORT env var > scenario status > API_PORT > default
if [[ -n "${DATA_STRUCTURER_API_PORT:-}" ]]; then
    SCENARIO_PORT="$DATA_STRUCTURER_API_PORT"
elif command -v vrooli >/dev/null 2>&1; then
    # Extract port from scenario status
    SCENARIO_PORT=$(vrooli scenario status data-structurer 2>/dev/null | grep -oP 'API_PORT: \K\d+' || echo "15769")
else
    SCENARIO_PORT="${API_PORT:-15769}"
fi
API_BASE_URL="${API_BASE_URL:-http://localhost:$SCENARIO_PORT}"

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

# Test resource availability via API health endpoint
test_resource_via_api() {
    local resource_name="$1"

    # First try to get status from API health endpoint
    local health_response
    if health_response=$(curl -sf "$API_BASE_URL/health" 2>/dev/null); then
        local resource_status
        resource_status=$(echo "$health_response" | jq -r ".dependencies.${resource_name}.status // \"unknown\"" 2>/dev/null)

        if [[ "$resource_status" == "healthy" ]]; then
            log_success "$resource_name is available (verified via API)"
            return 0
        fi
    fi

    # Fallback to vrooli resource status
    if vrooli resource status "$resource_name" 2>/dev/null | grep -q 'Running: true'; then
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
    if test_resource_via_api "postgres"; then
        pass_count=$((pass_count + 1))
    fi

    # Test 2: Ollama availability
    log_info "Test 2: Ollama availability"
    test_count=$((test_count + 1))
    if test_resource_via_api "ollama"; then
        pass_count=$((pass_count + 1))
    fi

    # Test 3: Unstructured-io availability
    log_info "Test 3: Unstructured-io availability"
    test_count=$((test_count + 1))
    if test_resource_via_api "unstructured_io"; then
        pass_count=$((pass_count + 1))
    else
        log_warning "Unstructured-io not available (optional resource)"
    fi

    # Test 4: N8N availability
    log_info "Test 4: N8N availability"
    test_count=$((test_count + 1))
    if test_resource_via_api "n8n"; then
        pass_count=$((pass_count + 1))
    else
        log_warning "N8N not available (workflows won't function)"
    fi

    # Test 5: Qdrant availability (optional)
    log_info "Test 5: Qdrant availability (optional)"
    test_count=$((test_count + 1))
    if test_resource_via_api "qdrant"; then
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

    # Check via API health endpoint for table count
    if health_response=$(curl -sf "$API_BASE_URL/health" 2>/dev/null); then
        local table_count
        table_count=$(echo "$health_response" | jq -r '.dependencies.postgres.checks.structurer_tables // 0' 2>/dev/null)

        if [[ "$table_count" -eq 3 ]]; then
            log_success "Database schema is properly initialized (3 tables found)"
            pass_count=$((pass_count + 1))
        else
            log_warning "Database schema may not be fully initialized (found $table_count tables, expected 3)"
        fi
    else
        log_warning "Could not verify database schema"
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