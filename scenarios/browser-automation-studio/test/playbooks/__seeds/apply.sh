#!/usr/bin/env bash
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCENARIO_NAME="${SCENARIO_NAME:-browser-automation-studio}"
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
if [ -z "$API_PORT" ]; then
  echo "❌ Unable to resolve API_PORT for ${SCENARIO_NAME}. Ensure the scenario is running before applying seeds." >&2
  exit 1
fi
API_URL="http://localhost:${API_PORT}/api/v1"
DATA_DIR="${SCENARIO_DIR}/data/projects/demo"
mkdir -p "$DATA_DIR"

FIXTURES_DIR="${SCENARIO_DIR}/test/artifacts/runtime"
SEED_STATE_FILE="${FIXTURES_DIR}/seed-state.json"
mkdir -p "$FIXTURES_DIR"

SEED_PROJECT_NAME=""
SEED_WORKFLOW_NAME=""
SEED_PROJECT_FOLDER="/demo"

if [ -f "$SEED_STATE_FILE" ] && command -v jq >/dev/null 2>&1; then
  existing_seed=$(cat "$SEED_STATE_FILE")
  SEED_PROJECT_NAME=$(printf '%s' "$existing_seed" | jq -r '.projectName // empty')
  SEED_WORKFLOW_NAME=$(printf '%s' "$existing_seed" | jq -r '.workflowName // empty')
  SEED_PROJECT_FOLDER=$(printf '%s' "$existing_seed" | jq -r '.projectFolder // "/demo"')
fi

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "❌ Required tool '$1' is missing. Install it to apply BAS seeds." >&2
    exit 1
  fi
}

require_tool jq
require_tool curl

curl_json() {
  local method="$1"
  local url="$2"
  local body="${3:-}"
  if [ -n "$body" ]; then
    curl -s -X "$method" -H 'Content-Type: application/json' -d "$body" "$url"
  else
    curl -s -X "$method" "$url"
  fi
}

# Clean up disk-based workflow storage to prevent workflows from being restored
echo "🧹 Cleaning up disk-based workflow storage..."
if [ -d "${SCENARIO_DIR}/data/projects" ]; then
  rm -rf "${SCENARIO_DIR}/data/projects"/*
  echo "✓ Cleared data/projects directory"
fi

# Remove orphaned workflows (test workflows with NULL project_id)
echo "🧹 Cleaning up orphaned test workflows..."
curl_json DELETE "$API_URL/workflows/cleanup-orphaned" >/dev/null || true

# Remove every project so the test run starts from a known state.
list_resp=$(curl_json GET "$API_URL/projects") || list_resp='{}'
project_ids=$(printf '%s' "$list_resp" | jq -r '.projects[]? | (.project.id // .Project.id // .id // empty)')
if [ -n "$project_ids" ]; then
  while IFS= read -r pid; do
    [ -z "$pid" ] && continue
    curl_json DELETE "$API_URL/projects/${pid}" >/dev/null || true
  done <<< "$project_ids"
fi

if [ -z "$SEED_PROJECT_NAME" ] || [ -z "$SEED_WORKFLOW_NAME" ]; then
  run_id=$(date +%s)
  if command -v openssl >/dev/null 2>&1; then
    suffix=$(openssl rand -hex 3 | tr -d '\n')
  else
    suffix=$(dd if=/dev/urandom bs=4 count=1 2>/dev/null | od -An -tx1 | tr -d ' \n' | cut -c1-6)
  fi
  SEED_PROJECT_NAME="Demo Browser Automations ${run_id}-${suffix}"
  SEED_WORKFLOW_NAME="Demo Smoke Workflow ${run_id}-${suffix}"
fi

# Create demo project that all workflows expect.
create_payload=$(jq -n \
  --arg name "$SEED_PROJECT_NAME" \
  --arg desc "Seeded project for BAS integration testing" \
  --arg folder "$DATA_DIR" \
  '{name: $name, description: $desc, folder_path: $folder}')
create_resp=$(curl_json POST "$API_URL/projects" "$create_payload") || {
  echo "❌ Failed to create demo project" >&2
  printf '%s\n' "$create_resp" >&2
  exit 1
}
project_id=$(printf '%s' "$create_resp" | jq -r '.project_id // .project.id // empty')
if [ -z "$project_id" ]; then
  echo "❌ Unable to determine project_id from API response" >&2
  printf '%s\n' "$create_resp" >&2
  exit 1
fi

# Seed a simple workflow so workflow-listing tests have something to select.
IFS= read -r -d '' workflow_definition <<'JSON' || true
{
  "nodes": [
    {
      "id": "seed-navigate",
      "type": "navigate",
      "position": {"x": 0, "y": 0},
      "data": {
        "label": "Navigate to example.com",
        "destinationType": "url",
        "url": "https://example.com",
        "waitUntil": "domcontentloaded",
        "timeoutMs": 15000
      }
    },
    {
      "id": "seed-wait",
      "type": "wait",
      "position": {"x": 240, "y": 0},
      "data": {
        "label": "Settle page",
        "durationMs": 1500,
        "waitType": "duration"
      }
    }
  ],
  "edges": [
    {"id": "seed-edge", "source": "seed-navigate", "target": "seed-wait", "type": "smoothstep"}
  ]
}
JSON

workflow_payload=$(jq -n \
  --arg pid "$project_id" \
  --arg name "$SEED_WORKFLOW_NAME" \
  --arg folder "$SEED_PROJECT_FOLDER" \
  --argjson flow "$workflow_definition" \
  '{project_id: $pid, name: $name, folder_path: $folder, flow_definition: $flow}')
workflow_resp=$(curl_json POST "$API_URL/workflows/create" "$workflow_payload") || {
  echo "❌ Failed to create seed workflow" >&2
  printf '%s\n' "$workflow_resp" >&2
  exit 1
}

workflow_id=$(printf '%s' "$workflow_resp" | jq -r '.workflow.id // .workflow_id // .id // empty')
if [ -z "$workflow_id" ]; then
  echo "❌ Unable to determine workflow_id from API response" >&2
  printf '%s\n' "$workflow_resp" >&2
  exit 1
fi

seed_payload=$(jq -n \
  --arg projectId "$project_id" \
  --arg projectName "$SEED_PROJECT_NAME" \
  --arg projectFolder "$SEED_PROJECT_FOLDER" \
  --arg workflowId "$workflow_id" \
  --arg workflowName "$SEED_WORKFLOW_NAME" \
  '{projectId: $projectId, projectName: $projectName, projectFolder: $projectFolder, workflowId: $workflowId, workflowName: $workflowName}')

# Validate seed payload has all required keys before writing
required_keys=("projectId" "projectName" "projectFolder" "workflowId" "workflowName")
for key in "${required_keys[@]}"; do
  value=$(printf '%s' "$seed_payload" | jq -r ".$key // empty")
  if [ -z "$value" ] || [ "$value" = "null" ]; then
    echo "❌ Seed state validation failed: missing or empty '$key'" >&2
    printf '%s\n' "$seed_payload" >&2
    exit 1
  fi
done

printf '%s\n' "$seed_payload" > "$SEED_STATE_FILE"

echo "📝 Wrote BAS seed state to ${SEED_STATE_FILE}"

echo "✅ BAS seed data applied (project ${project_id})"
