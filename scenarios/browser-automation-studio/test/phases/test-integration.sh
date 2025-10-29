#!/bin/bash
set -euo pipefail

echo "=== Test Integration ==="

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SCENARIO_DIR="${ROOT_DIR}/scenarios/browser-automation-studio"

cd "$SCENARIO_DIR"

AUTOMATION_SCRIPT="automation/executions/telemetry-smoke.sh"

if [[ ! -x "$AUTOMATION_SCRIPT" ]]; then
  echo "❌ Missing telemetry smoke automation script ($AUTOMATION_SCRIPT)" >&2
  exit 1
fi

# Allow outer harness to decide whether to stop the scenario afterwards
: "${BAS_AUTOMATION_STOP_SCENARIO:=1}"

echo "→ Running telemetry smoke automation"

if BAS_AUTOMATION_STOP_SCENARIO="${BAS_AUTOMATION_STOP_SCENARIO}" "$AUTOMATION_SCRIPT"; then
  echo "✅ Telemetry smoke automation passed"
else
  echo "❌ Telemetry smoke automation failed" >&2
  exit 1
fi

echo "✅ Integration tests completed"
