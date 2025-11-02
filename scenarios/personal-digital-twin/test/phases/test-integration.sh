#!/bin/bash
# Runs API-centric integration checks covering persona, data, and chat surfaces.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/connectivity.sh"

testing::phase::init --target-time "300s" --require-runtime

missing_tools=()
for tool in curl jq; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    missing_tools+=("$tool")
  fi
done

if [ ${#missing_tools[@]} -gt 0 ]; then
  testing::phase::add_error "Required tooling missing: ${missing_tools[*]}"
  testing::phase::end_with_summary "Integration checks blocked"
fi

SCENARIO_NAME="$TESTING_PHASE_SCENARIO_NAME"
API_BASE_URL=$(testing::connectivity::get_api_url "$SCENARIO_NAME" || true)
if [ -z "$API_BASE_URL" ]; then
  testing::phase::add_error "Unable to resolve API URL for $SCENARIO_NAME"
  testing::phase::end_with_summary "Integration checks blocked"
fi

CHAT_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" CHAT_PORT 2>/dev/null || true)
CHAT_PORT=$(echo "$CHAT_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$CHAT_PORT" ]; then
  CHAT_BASE_URL="$API_BASE_URL"
else
  CHAT_BASE_URL="http://localhost:${CHAT_PORT}"
fi

PERSONA_ID=""
DATA_SOURCE_ID=""
SESSION_ID=""

check_api_health() {
  local response
  response=$(curl -s "$API_BASE_URL/health") || return 1
  if echo "$response" | grep -qi "healthy"; then
    return 0
  fi
  log::error "Health response: $response"
  return 1
}

create_persona() {
  local response
  response=$(curl -s -X POST "$API_BASE_URL/api/persona/create" \
    -H "Content-Type: application/json" \
    -d '{"name": "Integration Test Persona", "description": "Scenario integration validation"}') || return 1
  PERSONA_ID=$(echo "$response" | jq -r '.id // empty')
  if [ -z "$PERSONA_ID" ]; then
    log::error "Persona creation response: $response"
    return 1
  fi
  log::info "Created persona $PERSONA_ID"
}

fetch_persona() {
  local response
  response=$(curl -s "$API_BASE_URL/api/persona/${PERSONA_ID}") || return 1
  echo "$response" | jq -e '.id == env.PERSONA_ID' >/dev/null 2>&1
}

list_personas() {
  curl -s "$API_BASE_URL/api/personas" | jq -e '.personas | length >= 1' >/dev/null 2>&1
}

connect_data_source() {
  local response
  local payload
  payload=$(jq -cn --arg persona "$PERSONA_ID" '{persona_id: $persona, source_type: "file", source_config: {path: "/test/data"}}')
  response=$(curl -s -X POST "$API_BASE_URL/api/datasource/connect" \
    -H "Content-Type: application/json" \
    -d "$payload") || return 1
  DATA_SOURCE_ID=$(echo "$response" | jq -r '.source_id // empty')
  if [ -z "$DATA_SOURCE_ID" ]; then
    log::error "Data source response: $response"
    return 1
  fi
  curl -s "$API_BASE_URL/api/datasources/${PERSONA_ID}" | jq -e '.data_sources | length >= 1' >/dev/null 2>&1
}

start_training_job() {
  local response
  local payload
  payload=$(jq -cn --arg persona "$PERSONA_ID" '{persona_id: $persona, model: "llama3", technique: "fine-tune"}')
  response=$(curl -s -X POST "$API_BASE_URL/api/train/start" \
    -H "Content-Type: application/json" \
    -d "$payload") || return 1
  echo "$response" | jq -e '.job_id' >/dev/null 2>&1
}

create_api_token() {
  local response
  local payload
  payload=$(jq -cn --arg persona "$PERSONA_ID" '{persona_id: $persona, name: "integration-token", permissions: ["read", "write"]}')
  response=$(curl -s -X POST "$API_BASE_URL/api/tokens/create" \
    -H "Content-Type: application/json" \
    -d "$payload") || return 1
  echo "$response" | jq -e '.token' >/dev/null 2>&1
}

search_documents() {
  local response
  local payload
  payload=$(jq -cn --arg persona "$PERSONA_ID" '{persona_id: $persona, query: "integration test query", limit: 5}')
  response=$(curl -s -X POST "$API_BASE_URL/api/search" \
    -H "Content-Type: application/json" \
    -d "$payload") || return 1
  echo "$response" | jq -e '.results' >/dev/null 2>&1
}

chat_interaction() {
  local response
  local payload
  payload=$(jq -cn --arg persona "$PERSONA_ID" '{persona_id: $persona, message: "Hello from integration tests"}')
  response=$(curl -s -X POST "$CHAT_BASE_URL/api/chat" \
    -H "Content-Type: application/json" \
    -d "$payload") || return 1
  SESSION_ID=$(echo "$response" | jq -r '.session_id // empty')
  if [ -z "$SESSION_ID" ]; then
    log::error "Chat response: $response"
    return 1
  fi
  followup_chat
}

followup_chat() {
  local response
  local payload
  payload=$(jq -cn --arg persona "$PERSONA_ID" --arg session "$SESSION_ID" '{persona_id: $persona, message: "Thanks!", session_id: $session}')
  response=$(curl -s -X POST "$CHAT_BASE_URL/api/chat" \
    -H "Content-Type: application/json" \
    -d "$payload") || return 1
  local history
  history=$(curl -s "$CHAT_BASE_URL/api/chat/history/${SESSION_ID}?persona_id=${PERSONA_ID}") || return 1
  echo "$history" | jq -e '.messages | length >= 2' >/dev/null 2>&1
}

orchestrate_checks() {
  testing::phase::check "API health endpoint" check_api_health
  testing::phase::check "Create persona" create_persona
  testing::phase::check "Retrieve persona" fetch_persona
  testing::phase::check "List personas" list_personas
  testing::phase::check "Connect data source" connect_data_source
  testing::phase::check "Launch training job" start_training_job
  testing::phase::check "Create API token" create_api_token
  testing::phase::check "Search documents" search_documents
  testing::phase::check "Handle chat interaction" chat_interaction
}

orchestrate_checks

testing::phase::end_with_summary "Integration workflows validated"
