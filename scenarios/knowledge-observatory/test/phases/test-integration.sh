#!/bin/bash
#
# Integration Test Phase for knowledge-observatory
# Integrates with centralized Vrooli testing infrastructure
#

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running integration tests..."

# Check if required resources are available
POSTGRES_AVAILABLE=false
QDRANT_AVAILABLE=false

if command -v psql &>/dev/null; then
    if psql -h "${POSTGRES_HOST:-localhost}" -p "${POSTGRES_PORT:-5432}" -U "${POSTGRES_USER:-postgres}" -c "SELECT 1" &>/dev/null 2>&1; then
        POSTGRES_AVAILABLE=true
        testing::phase::success "PostgreSQL available"
    fi
fi

if command -v curl &>/dev/null; then
    qdrant_port="${QDRANT_PORT:-6333}"
    if curl -sf "http://localhost:${qdrant_port}/health" &>/dev/null; then
        QDRANT_AVAILABLE=true
        testing::phase::success "Qdrant available"
    fi
fi

if [ "$POSTGRES_AVAILABLE" = false ]; then
    testing::phase::warn "PostgreSQL not available, skipping database integration tests"
fi

if [ "$QDRANT_AVAILABLE" = false ]; then
    testing::phase::warn "Qdrant not available, skipping vector database integration tests"
fi

# Test API health endpoint integration
testing::phase::log "Testing API health check integration..."

API_PORT="${API_PORT:-17822}"
if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    testing::phase::success "API health check successful"

    # Test health response structure
    health_response=$(curl -s "http://localhost:$API_PORT/health")
    if echo "$health_response" | grep -q '"status"'; then
        testing::phase::success "Health response has status field"
    else
        testing::phase::error "Health response missing status field"
    fi

    if echo "$health_response" | grep -q '"dependencies"'; then
        testing::phase::success "Health response includes dependency checks"
    else
        testing::phase::warn "Health response may be missing dependency information"
    fi
else
    testing::phase::warn "API not running on port $API_PORT, skipping live API tests"
fi

# Test search endpoint integration
if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    testing::phase::log "Testing search endpoint integration..."

    search_response=$(curl -s -X POST "http://localhost:$API_PORT/api/v1/knowledge/search" \
        -H "Content-Type: application/json" \
        -d '{"query":"test","limit":5}')

    if [ $? -eq 0 ]; then
        testing::phase::success "Search endpoint responds"

        if echo "$search_response" | grep -q '"results"'; then
            testing::phase::success "Search response has results field"
        else
            testing::phase::error "Search response missing results field"
        fi
    else
        testing::phase::error "Search endpoint failed"
    fi
fi

# Test graph endpoint integration
if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    testing::phase::log "Testing graph endpoint integration..."

    graph_response=$(curl -s "http://localhost:$API_PORT/api/v1/knowledge/graph")

    if [ $? -eq 0 ]; then
        testing::phase::success "Graph endpoint responds"

        if echo "$graph_response" | grep -q '"nodes"' && echo "$graph_response" | grep -q '"edges"'; then
            testing::phase::success "Graph response has nodes and edges"
        else
            testing::phase::error "Graph response missing required fields"
        fi
    else
        testing::phase::error "Graph endpoint failed"
    fi
fi

# Test CLI integration with API
testing::phase::log "Testing CLI integration..."

if command -v knowledge-observatory &>/dev/null || [ -f "cli/knowledge-observatory" ]; then
    CLI_CMD="knowledge-observatory"
    if [ ! -f "/home/matthalloran8/.vrooli/bin/knowledge-observatory" ]; then
        CLI_CMD="./cli/knowledge-observatory"
    fi

    # Test CLI status command
    if API_PORT=$API_PORT $CLI_CMD status >/dev/null 2>&1; then
        testing::phase::success "CLI status command works"
    else
        testing::phase::warn "CLI status command failed (API may not be running)"
    fi

    # Test CLI help
    if $CLI_CMD --help >/dev/null 2>&1; then
        testing::phase::success "CLI help command works"
    else
        testing::phase::error "CLI help command failed"
    fi
else
    testing::phase::warn "CLI not found, skipping CLI integration tests"
fi

# Test resource integration with Qdrant
if [ "$QDRANT_AVAILABLE" = true ]; then
    testing::phase::log "Testing Qdrant integration..."

    if command -v resource-qdrant &>/dev/null; then
        # Test collection listing
        if timeout 5 resource-qdrant collections list &>/dev/null; then
            testing::phase::success "Can list Qdrant collections"
        else
            testing::phase::warn "Qdrant collections list failed or timed out"
        fi
    else
        testing::phase::warn "resource-qdrant CLI not available"
    fi
fi

# Test UI accessibility
testing::phase::log "Testing UI integration..."

UI_PORT="${UI_PORT:-35785}"
if curl -sf "http://localhost:$UI_PORT/" &>/dev/null; then
    testing::phase::success "UI is accessible"

    # Check if UI can reach API
    ui_content=$(curl -s "http://localhost:$UI_PORT/")
    if echo "$ui_content" | grep -qi "knowledge"; then
        testing::phase::success "UI content appears valid"
    else
        testing::phase::warn "UI content may not be loading correctly"
    fi
else
    testing::phase::warn "UI not running on port $UI_PORT"
fi

testing::phase::end_with_summary "Integration tests completed"
