#!/usr/bin/env bash
set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
SCENARIO_NAME="${SCENARIO_NAME:-browser-automation-studio}"
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)

SEED_STATE_FILE="${SCENARIO_DIR}/test/artifacts/runtime/seed-state.json"

# Check prerequisites before attempting cleanup
if [ -z "$API_PORT" ]; then
  # Nothing to clean if scenario unavailable - leave seed file intact for next run
  exit 0
fi
API_URL="http://localhost:${API_PORT}/api/v1"

if ! command -v curl >/dev/null 2>&1; then
  # Can't verify cleanup without curl - leave seed file intact
  exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
  # Can't parse response without jq - leave seed file intact
  exit 0
fi

# Fetch list of projects
list_resp=$(curl -s "$API_URL/projects" || echo '{}')

# Extract project IDs using correct JSON path
project_ids=$(printf '%s' "$list_resp" | jq -r '.projects[].id // empty')

# Delete all projects if any exist
if [ -n "$project_ids" ]; then
  while IFS= read -r pid; do
    [ -z "$pid" ] && continue
    curl -s -X DELETE "$API_URL/projects/${pid}" >/dev/null || true
  done <<< "$project_ids"
fi

# Only remove seed state file after successful cleanup
# (or if there were no projects to clean)
rm -f "$SEED_STATE_FILE" 2>/dev/null || true

echo "🧹 BAS seed data removed"
