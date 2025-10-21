#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${API_PORT:-}" ]]; then
  echo "[agent-metareasoning-manager][tests] API_PORT environment variable is required" >&2
  exit 1
fi

payload='{"input":"Should we adopt an async decision queue?","context":"Metareasoning manager smoke test","metadata":{"source":"scenario-tests","interface":"script"}}'

response=$(curl -sSf -X POST "http://127.0.0.1:${API_PORT}/reasoning/pros-cons" \
  -H 'Content-Type: application/json' \
  -d "$payload")

echo "$response" | grep -q 'analysis'

echo "[agent-metareasoning-manager][tests] Pros/cons reasoning endpoint responded successfully"
