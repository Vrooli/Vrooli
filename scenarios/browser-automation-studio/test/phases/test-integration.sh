#!/bin/bash
set -euo pipefail

echo "=== Test Integration ==="

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCENARIO_DIR="${ROOT_DIR}/scenarios/browser-automation-studio"

if [[ ! -d "$SCENARIO_DIR" ]]; then
  SCENARIO_DIR="$ROOT_DIR"
fi

cd "$SCENARIO_DIR"

AUTOMATION_SCRIPT="automation/executions/telemetry-smoke.sh"

if [[ ! -x "$AUTOMATION_SCRIPT" ]]; then
  echo "❌ Missing telemetry smoke automation script ($AUTOMATION_SCRIPT)" >&2
  exit 1
fi

SCENARIO_NAME="browser-automation-studio"
SCENARIO_RUNNING=0

cleanup() {
  if [[ $SCENARIO_RUNNING -eq 1 ]]; then
    vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

echo "→ Running telemetry smoke automation"

if BAS_AUTOMATION_STOP_SCENARIO=0 "$AUTOMATION_SCRIPT"; then
  echo "✅ Telemetry smoke automation passed"
else
  echo "❌ Telemetry smoke automation failed" >&2
  exit 1
fi

SCENARIO_RUNNING=1

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [[ -z "$API_PORT" ]]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [[ -z "$API_PORT" ]]; then
  echo "❌ Unable to resolve API_PORT for scenario" >&2
  exit 1
fi

export BAS_INTEGRATION_API_URL="http://localhost:${API_PORT}/api/v1"
echo "→ Running workflow version history integration against ${BAS_INTEGRATION_API_URL}"

node tests/workflow-version-history.integration.mjs

vrooli scenario stop "$SCENARIO_NAME" >/dev/null
SCENARIO_RUNNING=0

echo "✅ Integration tests completed"
