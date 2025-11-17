#!/usr/bin/env bash
set -euo pipefail

SCENARIO_NAME="${SCENARIO_NAME:-browser-automation-studio}"
API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
if [ -z "$API_PORT" ]; then
  # Nothing to clean if scenario unavailable.
  exit 0
fi
API_URL="http://localhost:${API_PORT}/api/v1"

if ! command -v curl >/dev/null 2>&1; then
  exit 0
fi

list_resp=$(curl -s "$API_URL/projects" || echo '{}')
if ! command -v jq >/dev/null 2>&1; then
  exit 0
fi
project_ids=$(printf '%s' "$list_resp" | jq -r '.projects[]?.project.id // .projects[]?.Project.id // empty')
if [ -z "$project_ids" ]; then
  exit 0
fi
while IFS= read -r pid; do
  [ -z "$pid" ] && continue
  curl -s -X DELETE "$API_URL/projects/${pid}" >/dev/null || true
done <<< "$project_ids"

echo "🧹 BAS seed data removed"
