#!/bin/bash
set -euo pipefail

echo "[INFO] Huginn integration smoke check (placeholder)"

if ! command -v vrooli >/dev/null 2>&1; then
  echo "[WARN] vrooli CLI not available; skipping Huginn integration check"
  exit 0
fi

API_PORT="${API_PORT:-$(vrooli scenario port bookmark-intelligence-hub API_PORT 2>/dev/null)}"
if [[ -z "${API_PORT}" ]]; then
  echo "[WARN] Scenario port unavailable; skipping Huginn integration check"
  exit 0
fi

if curl -sf "http://localhost:${API_PORT}/health" >/dev/null; then
  echo "[OK] API health endpoint reachable; Huginn integration placeholder passed"
else
  echo "[ERROR] API health check failed; downstream Huginn validation skipped"
  exit 1
fi
