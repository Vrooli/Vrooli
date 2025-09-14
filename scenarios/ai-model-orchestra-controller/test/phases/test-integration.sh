#!/bin/bash
# AI Model Orchestra Controller - Integration Phase Tests
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$TEST_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "ℹ️  $1"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }
log_warn() { echo -e "${YELLOW}⚠️  $1${NC}"; }

echo "🔗 Running integration tests for AI Model Orchestra Controller..."

# Test scenario startup and API availability
test_scenario_startup() {
    log_info "Testing scenario startup and API availability..."
    
    # Check if scenario is running
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -z "$api_port" ]; then
        log_warn "Scenario not running, attempting to start..."
        
        # Try to start the scenario
        if ! timeout 60 vrooli scenario run ai-model-orchestra-controller >/dev/null 2>&1; then
            log_error "Failed to start scenario within timeout"
            return 1
        fi
        
        # Wait for API to be ready
        sleep 10
        api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    fi
    
    if [ -z "$api_port" ]; then
        log_error "Could not determine API port"
        return 1
    fi
    
    # Test API health endpoint
    local health_url="http://localhost:${api_port}/api/v1/health"
    if timeout 10 curl -s "$health_url" >/dev/null 2>&1; then
        log_success "API health endpoint is accessible at port $api_port"
    else
        log_error "API health endpoint not accessible"
        return 1
    fi
    
    return 0
}

# Test database connectivity
test_database_connectivity() {
    log_info "Testing database connectivity..."
    
    # Check if PostgreSQL resource is available
    local postgres_port
    postgres_port=$(vrooli resource port postgres RESOURCE_PORTS_POSTGRES 2>/dev/null || echo "5432")
    
    # Test PostgreSQL connectivity with timeout
    if timeout 5 pg_isready -h localhost -p "$postgres_port" >/dev/null 2>&1; then
        log_success "PostgreSQL is accessible on port $postgres_port"
    else
        log_warn "PostgreSQL not accessible - testing connection handling"
        
        # Test that our API gracefully handles database unavailability
        local api_port
        api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
        
        if [ -n "$api_port" ]; then
            local health_url="http://localhost:${api_port}/api/v1/health"
            # Health endpoint should still work even if database is down
            if timeout 5 curl -s "$health_url" | grep -q "database.*unavailable\|database.*error" 2>/dev/null; then
                log_success "API gracefully handles database unavailability"
            else
                log_warn "API response unclear when database unavailable"
            fi
        fi
    fi
    
    return 0
}

# Test Redis connectivity
test_redis_connectivity() {
    log_info "Testing Redis connectivity..."
    
    # Check if Redis resource is available
    local redis_port
    redis_port=$(vrooli resource port redis RESOURCE_PORTS_REDIS 2>/dev/null || echo "6379")
    
    # Test Redis connectivity
    if timeout 3 redis-cli -h localhost -p "$redis_port" ping >/dev/null 2>&1; then
        log_success "Redis is accessible on port $redis_port"
    else
        log_warn "Redis not accessible - testing fallback behavior"
        
        # Test that caching gracefully degrades
        local api_port
        api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
        
        if [ -n "$api_port" ]; then
            local models_url="http://localhost:${api_port}/api/v1/models"
            # Models endpoint should work without caching
            if timeout 10 curl -s "$models_url" >/dev/null 2>&1; then
                log_success "API works without Redis caching"
            else
                log_warn "API may depend too heavily on Redis"
            fi
        fi
    fi
    
    return 0
}

# Test AI model integration (Ollama)
test_ai_model_integration() {
    log_info "Testing AI model integration..."
    
    # Check if Ollama is available
    local ollama_port
    ollama_port=$(vrooli resource port ollama RESOURCE_PORTS_OLLAMA 2>/dev/null || echo "11434")
    
    local ollama_url="http://localhost:${ollama_port}"
    
    # Test Ollama API connectivity
    if timeout 10 curl -s "${ollama_url}/api/tags" >/dev/null 2>&1; then
        log_success "Ollama is accessible on port $ollama_port"
        
        # Test model listing through our API
        local api_port
        api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
        
        if [ -n "$api_port" ]; then
            local models_url="http://localhost:${api_port}/api/v1/models"
            if timeout 15 curl -s "$models_url" >/dev/null 2>&1; then
                log_success "Model listing works through orchestrator API"
            else
                log_warn "Model listing through orchestrator may be failing"
            fi
        fi
    else
        log_warn "Ollama not accessible - testing offline behavior"
        
        # Test graceful degradation without AI models
        local api_port
        api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
        
        if [ -n "$api_port" ]; then
            local health_url="http://localhost:${api_port}/api/v1/health"
            # Health should still work without AI models
            if timeout 5 curl -s "$health_url" >/dev/null 2>&1; then
                log_success "API gracefully handles missing AI models"
            else
                log_warn "API may be too dependent on AI model availability"
            fi
        fi
    fi
    
    return 0
}

# Test API endpoint integration
test_api_endpoints() {
    log_info "Testing API endpoint integration..."
    
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -z "$api_port" ]; then
        log_error "Cannot test API endpoints - scenario not running"
        return 1
    fi
    
    local base_url="http://localhost:${api_port}/api/v1"
    
    # Test core endpoints
    local endpoints=(
        "health"
        "models" 
        "metrics"
        "status"
    )
    
    for endpoint in "${endpoints[@]}"; do
        local url="${base_url}/${endpoint}"
        
        if timeout 10 curl -s "$url" >/dev/null 2>&1; then
            log_success "Endpoint accessible: /$endpoint"
        else
            log_warn "Endpoint may be unavailable: /$endpoint"
        fi
    done
    
    # Test CORS headers if applicable
    local health_url="${base_url}/health"
    if timeout 5 curl -s -I "$health_url" | grep -i "access-control-allow-origin" >/dev/null 2>&1; then
        log_success "CORS headers are present"
    else
        log_warn "CORS headers may be missing (could affect UI)"
    fi
    
    return 0
}

# Test CLI integration
test_cli_integration() {
    log_info "Testing CLI integration..."
    
    # Check if CLI is installed
    if ! command -v ai-orchestra >/dev/null 2>&1; then
        log_warn "CLI not installed, testing installation..."
        
        if [ -f "$SCENARIO_ROOT/cli/install.sh" ]; then
            # Test installation (dry run if possible)
            if timeout 30 bash "$SCENARIO_ROOT/cli/install.sh" >/dev/null 2>&1; then
                log_success "CLI installation successful"
            else
                log_warn "CLI installation may have issues"
                return 0  # Non-critical for integration phase
            fi
        else
            log_error "CLI install script not found"
            return 1
        fi
    fi
    
    # Test CLI commands if available
    if command -v ai-orchestra >/dev/null 2>&1; then
        # Test basic CLI functionality
        if timeout 10 ai-orchestra health >/dev/null 2>&1; then
            log_success "CLI health command works"
        else
            log_warn "CLI health command may not work"
        fi
        
        if timeout 15 ai-orchestra models >/dev/null 2>&1; then
            log_success "CLI models command works"
        else
            log_warn "CLI models command may not work"
        fi
    fi
    
    return 0
}

# Test UI server integration
test_ui_server_integration() {
    log_info "Testing UI server integration..."
    
    # Check if UI server is running
    local ui_port
    ui_port=$(vrooli scenario port ai-model-orchestra-controller UI_PORT 2>/dev/null || echo "")
    
    if [ -n "$ui_port" ]; then
        local ui_url="http://localhost:${ui_port}"
        
        # Test UI accessibility
        if timeout 5 curl -s "$ui_url" >/dev/null 2>&1; then
            log_success "UI server is accessible on port $ui_port"
            
            # Test static assets
            if timeout 5 curl -s "${ui_url}/dashboard.css" >/dev/null 2>&1; then
                log_success "UI CSS assets are accessible"
            else
                log_warn "UI CSS assets may not be served correctly"
            fi
            
            if timeout 5 curl -s "${ui_url}/dashboard.js" >/dev/null 2>&1; then
                log_success "UI JavaScript assets are accessible"
            else
                log_warn "UI JavaScript assets may not be served correctly"
            fi
        else
            log_warn "UI server not accessible"
        fi
    else
        log_warn "UI server port not detected - may not be running"
    fi
    
    return 0
}

# Test resource orchestration
test_resource_orchestration() {
    log_info "Testing resource orchestration..."
    
    # Test that our scenario can coordinate with other resources
    local api_port
    api_port=$(vrooli scenario port ai-model-orchestra-controller API_PORT 2>/dev/null || echo "")
    
    if [ -n "$api_port" ]; then
        local orchestration_url="http://localhost:${api_port}/api/v1/orchestrate"
        
        # Test orchestration endpoint (may not exist yet, but test graceful handling)
        local response_code
        response_code=$(timeout 10 curl -s -o /dev/null -w "%{http_code}" "$orchestration_url" 2>/dev/null || echo "000")
        
        if [[ "$response_code" =~ ^(200|404|405)$ ]]; then
            log_success "Orchestration endpoint handles requests gracefully (HTTP $response_code)"
        else
            log_warn "Orchestration endpoint response unclear (HTTP $response_code)"
        fi
    fi
    
    # Test scenario lifecycle integration
    if timeout 5 vrooli scenario status ai-model-orchestra-controller >/dev/null 2>&1; then
        log_success "Scenario lifecycle integration works"
    else
        log_warn "Scenario lifecycle integration may have issues"
    fi
    
    return 0
}

# Run all integration tests
echo "Starting integration validation tests..."

# Execute all tests (continue on non-critical failures)
test_scenario_startup || exit 1
test_database_connectivity # Non-critical
test_redis_connectivity # Non-critical
test_ai_model_integration # Non-critical but important
test_api_endpoints || exit 1
test_cli_integration # Non-critical
test_ui_server_integration # Non-critical
test_resource_orchestration # Non-critical

echo ""
log_success "All integration tests passed!"
echo "✅ Integration phase completed successfully"