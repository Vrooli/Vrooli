#!/bin/bash
set -euo pipefail

# Integration Testing Phase for personal-digital-twin
# This script runs integration tests that verify end-to-end functionality

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "🔗 Running integration tests for personal-digital-twin..."

# Ensure scenario is running
SCENARIO_NAME="personal-digital-twin"

# Check if scenario is running via lifecycle
if ! pgrep -f "personal-digital-twin-api" > /dev/null; then
    echo "⚠️  Scenario not running, attempting to start..."
    vrooli scenario start "$SCENARIO_NAME" || {
        echo "❌ Failed to start scenario for integration tests"
        exit 1
    }
    sleep 5
fi

# Get API port from environment or default
API_PORT="${API_PORT:-8080}"
CHAT_PORT="${CHAT_PORT:-8081}"

# Wait for API to be ready
echo "⏳ Waiting for API to be ready..."
for i in {1..30}; do
    if curl -s "http://localhost:${API_PORT}/health" > /dev/null 2>&1; then
        echo "✅ API is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ API failed to become ready"
        exit 1
    fi
    sleep 1
done

# Run integration tests
echo "🧪 Running API integration tests..."

# Test 1: Health check
echo "  → Testing health endpoint..."
HEALTH_RESPONSE=$(curl -s "http://localhost:${API_PORT}/health")
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo "  ✅ Health check passed"
else
    echo "  ❌ Health check failed"
    exit 1
fi

# Test 2: Create persona
echo "  → Testing persona creation..."
PERSONA_RESPONSE=$(curl -s -X POST "http://localhost:${API_PORT}/api/persona/create" \
    -H "Content-Type: application/json" \
    -d '{"name": "Integration Test Persona", "description": "Test persona for integration testing"}')

if echo "$PERSONA_RESPONSE" | grep -q "id"; then
    PERSONA_ID=$(echo "$PERSONA_RESPONSE" | jq -r '.id')
    echo "  ✅ Persona created: $PERSONA_ID"
else
    echo "  ❌ Persona creation failed"
    echo "  Response: $PERSONA_RESPONSE"
    exit 1
fi

# Test 3: Get persona
echo "  → Testing persona retrieval..."
GET_PERSONA_RESPONSE=$(curl -s "http://localhost:${API_PORT}/api/persona/${PERSONA_ID}")
if echo "$GET_PERSONA_RESPONSE" | grep -q "Integration Test Persona"; then
    echo "  ✅ Persona retrieval passed"
else
    echo "  ❌ Persona retrieval failed"
    exit 1
fi

# Test 4: List personas
echo "  → Testing persona list..."
LIST_RESPONSE=$(curl -s "http://localhost:${API_PORT}/api/personas")
if echo "$LIST_RESPONSE" | grep -q "personas"; then
    echo "  ✅ Persona list passed"
else
    echo "  ❌ Persona list failed"
    exit 1
fi

# Test 5: Connect data source
echo "  → Testing data source connection..."
DS_RESPONSE=$(curl -s -X POST "http://localhost:${API_PORT}/api/datasource/connect" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"source_type\": \"file\", \"source_config\": {\"path\": \"/test/data\"}}")

if echo "$DS_RESPONSE" | grep -q "source_id"; then
    echo "  ✅ Data source connection passed"
else
    echo "  ❌ Data source connection failed"
    exit 1
fi

# Test 6: Start training
echo "  → Testing training job creation..."
TRAIN_RESPONSE=$(curl -s -X POST "http://localhost:${API_PORT}/api/train/start" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"model\": \"llama2\", \"technique\": \"fine-tuning\"}")

if echo "$TRAIN_RESPONSE" | grep -q "job_id"; then
    echo "  ✅ Training job creation passed"
else
    echo "  ❌ Training job creation failed"
    exit 1
fi

# Test 7: Create API token
echo "  → Testing API token creation..."
TOKEN_RESPONSE=$(curl -s -X POST "http://localhost:${API_PORT}/api/tokens/create" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"name\": \"integration-test-token\", \"permissions\": [\"read\", \"write\"]}")

if echo "$TOKEN_RESPONSE" | grep -q "token"; then
    echo "  ✅ API token creation passed"
else
    echo "  ❌ API token creation failed"
    exit 1
fi

# Test 8: Search documents
echo "  → Testing document search..."
SEARCH_RESPONSE=$(curl -s -X POST "http://localhost:${API_PORT}/api/search" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"query\": \"test query\", \"limit\": 5}")

if echo "$SEARCH_RESPONSE" | grep -q "results"; then
    echo "  ✅ Document search passed"
else
    echo "  ❌ Document search failed"
    exit 1
fi

# Test 9: Chat endpoint
echo "  → Testing chat functionality..."
CHAT_RESPONSE=$(curl -s -X POST "http://localhost:${CHAT_PORT}/api/chat" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"message\": \"Hello, how are you?\"}")

if echo "$CHAT_RESPONSE" | grep -q "response"; then
    SESSION_ID=$(echo "$CHAT_RESPONSE" | jq -r '.session_id')
    echo "  ✅ Chat functionality passed"
else
    echo "  ❌ Chat functionality failed"
    exit 1
fi

# Test 10: Chat history
echo "  → Testing chat history retrieval..."
HISTORY_RESPONSE=$(curl -s "http://localhost:${CHAT_PORT}/api/chat/history/${SESSION_ID}?persona_id=${PERSONA_ID}")

if echo "$HISTORY_RESPONSE" | grep -q "messages"; then
    echo "  ✅ Chat history retrieval passed"
else
    echo "  ❌ Chat history retrieval failed"
    exit 1
fi

echo "🎉 All integration tests passed!"

testing::phase::end_with_summary "Integration tests completed successfully"
