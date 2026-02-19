#!/usr/bin/env bash
# Seed script for web-console BAS (Browser Automation Suite)
# Creates default session data for BAS tests and workflows.
set -euo pipefail

API_PORT="${API_PORT:-$(vrooli scenario port web-console API_PORT 2>/dev/null || echo 17086)}"
API_BASE="http://127.0.0.1:${API_PORT}/api/v1"

echo "[seed] Creating default terminal session..."
curl -sf -X POST "${API_BASE}/sessions" \
  -H "Content-Type: application/json" \
  -d '{"shell":"/bin/bash","cols":120,"rows":40}' \
  -o /dev/null

echo "[seed] Seeding complete."
