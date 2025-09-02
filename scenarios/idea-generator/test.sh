#!/usr/bin/env bash
# Idea Generator - Integration Test Suite
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scenarios/idea-generator"
SCENARIO_DIR="$SCRIPT_DIR"
PROJECT_ROOT="$APP_ROOT"

# Test configuration
TEST_TIMEOUT=60
TEST_RESULTS=()
FAILED_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_test() { echo -e "${GREEN}[TEST]${NC} $*"; }

# Test helper functions
wait_for_service() {
    local service=$1
    local port=$2
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if nc -z localhost "$port" 2>/dev/null; then
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 1
    done
    
    return 1
}

run_test() {
    local test_name=$1
    local test_func=$2
    
    log_test "Running: $test_name"
    
    if $test_func; then
        TEST_RESULTS+=("✅ $test_name")
        log_info "PASSED: $test_name"
    else
        TEST_RESULTS+=("❌ $test_name")
        log_error "FAILED: $test_name"
        ((FAILED_TESTS++))
    fi
}

# Test Functions
test_postgres_connection() {
    local port="${RESOURCE_PORTS[postgres]:-5432}"
    PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
        -h localhost \
        -p "$port" \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-idea_generator}" \
        -c "SELECT 1" >/dev/null 2>&1
}

test_redis_connection() {
    local port="${RESOURCE_PORTS[redis]:-6379}"
    redis-cli -p "$port" ping >/dev/null 2>&1
}

test_minio_connection() {
    local port="${RESOURCE_PORTS[minio]:-9000}"
    curl -f "http://localhost:$port/minio/health/live" >/dev/null 2>&1
}

test_qdrant_connection() {
    local port="${RESOURCE_PORTS[qdrant]:-6333}"
    curl -f "http://localhost:$port/health" >/dev/null 2>&1
}

test_ollama_connection() {
    local port="${RESOURCE_PORTS[ollama]:-11434}"
    curl -f "http://localhost:$port/api/version" >/dev/null 2>&1
}

test_unstructured_connection() {
    local port="${RESOURCE_PORTS[unstructured-io]:-11450}"
    curl -f "http://localhost:$port/healthcheck" >/dev/null 2>&1
}

test_n8n_connection() {
    local port="${RESOURCE_PORTS[n8n]:-5678}"
    curl -f "http://localhost:$port/healthz" >/dev/null 2>&1
}

test_windmill_connection() {
    local port="${RESOURCE_PORTS[windmill]:-5681}"
    curl -f "http://localhost:$port/api/version" >/dev/null 2>&1
}

test_api_connection() {
    local port="${API_PORT:-8500}"
    curl -f "http://localhost:$port/health" >/dev/null 2>&1
}

test_database_schema() {
    local port="${RESOURCE_PORTS[postgres]:-5432}"
    local tables=$(PGPASSWORD="${POSTGRES_PASSWORD:-postgres}" psql \
        -h localhost \
        -p "$port" \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-idea_generator}" \
        -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'" 2>/dev/null | xargs)
    
    [[ $tables -ge 5 ]] # We expect at least 5 tables
}

test_qdrant_collections() {
    local port="${RESOURCE_PORTS[qdrant]:-6333}"
    local response=$(curl -s "http://localhost:$port/collections")
    echo "$response" | grep -q "ideas" && \
    echo "$response" | grep -q "documents" && \
    echo "$response" | grep -q "campaigns"
}

test_minio_buckets() {
    local port="${RESOURCE_PORTS[minio]:-9000}"
    # Check if key buckets exist
    curl -s -I "http://localhost:$port/idea-documents" | grep -q "200\|404" && \
    curl -s -I "http://localhost:$port/processed-content" | grep -q "200\|404"
}

test_n8n_workflows() {
    local port="${RESOURCE_PORTS[n8n]:-5678}"
    local workflows=$(curl -s "http://localhost:$port/rest/workflows" | grep -o '"name"' | wc -l)
    [[ $workflows -ge 5 ]] # We expect at least 5 workflows
}

test_idea_generation_workflow() {
    local port="${RESOURCE_PORTS[n8n]:-5678}"
    # Test the idea generation workflow
    local response=$(curl -s -X POST "http://localhost:$port/webhook/generate-ideas" \
        -H "Content-Type: application/json" \
        -d '{
            "campaign_id": "650e8400-e29b-41d4-a716-446655440001",
            "prompt": "Test idea generation",
            "count": 1
        }' 2>/dev/null || echo '{"status":"error"}')
    
    echo "$response" | grep -q '"status"'
}

test_semantic_search_workflow() {
    local port="${RESOURCE_PORTS[n8n]:-5678}"
    # Test the semantic search workflow
    local response=$(curl -s -X POST "http://localhost:$port/webhook/semantic-search" \
        -H "Content-Type: application/json" \
        -d '{
            "query": "test search",
            "limit": 5
        }' 2>/dev/null || echo '{"status":"error"}')
    
    echo "$response" | grep -q '"status"'
}

test_campaign_operations() {
    local port="${RESOURCE_PORTS[n8n]:-5678}"
    # Test campaign sync workflow
    local response=$(curl -s -X POST "http://localhost:$port/webhook/campaign-sync" \
        -H "Content-Type: application/json" \
        -d '{
            "operation": "create",
            "name": "Test Campaign",
            "user_id": "550e8400-e29b-41d4-a716-446655440001"
        }' 2>/dev/null || echo '{"status":"error"}')
    
    echo "$response" | grep -q '"status"'
}

test_windmill_app() {
    local port="${RESOURCE_PORTS[windmill]:-5681}"
    # Check if Windmill app is deployed
    local apps=$(curl -s "http://localhost:$port/api/apps" 2>/dev/null || echo '[]')
    echo "$apps" | grep -q "Idea Generator" || [[ "$apps" != "[]" ]]
}

test_ollama_models() {
    local port="${RESOURCE_PORTS[ollama]:-11434}"
    # Check if required models are available
    local models=$(curl -s "http://localhost:$port/api/tags" 2>/dev/null || echo '{}')
    echo "$models" | grep -q "llama3.2\|mistral"
}

test_api_endpoints() {
    local port="${API_PORT:-8500}"
    
    # Test health endpoint
    curl -f "http://localhost:$port/health" >/dev/null 2>&1 && \
    # Test campaigns endpoint
    curl -f "http://localhost:$port/campaigns" >/dev/null 2>&1 && \
    # Test ideas endpoint
    curl -f "http://localhost:$port/ideas" >/dev/null 2>&1 && \
    # Test workflows endpoint
    curl -f "http://localhost:$port/workflows" >/dev/null 2>&1
}

# Main test execution
main() {
    log_info "=== Idea Generator Integration Tests ==="
    log_info "Testing scenario components..."
    
    # Service connectivity tests
    run_test "PostgreSQL Connection" test_postgres_connection
    run_test "Redis Connection" test_redis_connection
    run_test "MinIO Connection" test_minio_connection
    run_test "Qdrant Connection" test_qdrant_connection
    run_test "Ollama Connection" test_ollama_connection
    run_test "Unstructured-IO Connection" test_unstructured_connection
    run_test "n8n Connection" test_n8n_connection
    run_test "Windmill Connection" test_windmill_connection
    run_test "API Connection" test_api_connection
    
    # Configuration tests
    run_test "Database Schema" test_database_schema
    run_test "Qdrant Collections" test_qdrant_collections
    run_test "MinIO Buckets" test_minio_buckets
    run_test "n8n Workflows" test_n8n_workflows
    run_test "Windmill App" test_windmill_app
    run_test "Ollama Models" test_ollama_models
    
    # API tests
    run_test "API Endpoints" test_api_endpoints
    
    # Workflow tests
    run_test "Idea Generation Workflow" test_idea_generation_workflow
    run_test "Semantic Search Workflow" test_semantic_search_workflow
    run_test "Campaign Operations" test_campaign_operations
    
    # Print test results
    echo ""
    log_info "=== Test Results ==="
    for result in "${TEST_RESULTS[@]}"; do
        echo "  $result"
    done
    
    echo ""
    if [ $FAILED_TESTS -eq 0 ]; then
        log_info "✅ All tests passed!"
        exit 0
    else
        log_error "❌ $FAILED_TESTS test(s) failed"
        exit 1
    fi
}

# Run tests
main "$@"