#!/usr/bin/env bash
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCENARIO_NAME="${SCENARIO_NAME:-browser-automation-studio}"
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
if [ -z "$API_PORT" ]; then
  echo "❌ Unable to resolve API_PORT for ${SCENARIO_NAME}. Ensure the scenario is running before applying seeds." >&2
  exit 1
fi
API_URL="http://localhost:${API_PORT}/api/v1"
DATA_DIR="${SCENARIO_DIR}/data/projects/demo"
mkdir -p "$DATA_DIR"

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

# Remove every project so the test run starts from a known state.
list_resp=$(curl_json GET "$API_URL/projects") || list_resp='{}'
project_ids=$(printf '%s' "$list_resp" | jq -r '.projects[]? | (.project.id // .Project.id // .id // empty)')
if [ -n "$project_ids" ]; then
  while IFS= read -r pid; do
    [ -z "$pid" ] && continue
    curl_json DELETE "$API_URL/projects/${pid}" >/dev/null || true
  done <<< "$project_ids"
fi

# Create demo project that all workflows expect.
create_payload=$(jq -n \
  --arg name "Demo Browser Automations" \
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
  --arg name "Demo Smoke Workflow" \
  --arg folder "/demo" \
  --argjson flow "$workflow_definition" \
  '{project_id: $pid, name: $name, folder_path: $folder, flow_definition: $flow}')
workflow_resp=$(curl_json POST "$API_URL/workflows/create" "$workflow_payload") || {
  echo "❌ Failed to create seed workflow" >&2
  printf '%s\n' "$workflow_resp" >&2
  exit 1
}

echo "✅ BAS seed data applied (project ${project_id})"
