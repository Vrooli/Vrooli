#!/usr/bin/env bash
set -euo pipefail

SCENARIO_NAME="browser-automation-studio"
WORKFLOW_NAME="automation-heartbeat-stall"
TMP_DIR=""
API_PORT=""
WS_PORT=""
API_URL=""
WATCH_LOG=""
EXECUTION_ID=""
WORKFLOW_ID=""
HEARTBEAT_ENV_VALUE="0"

cleanup() {
  local exit_code=$?
  if [[ -n "$WATCH_LOG" && -f "$WATCH_LOG" ]]; then
    echo "--- execution watch output ---"
    cat "$WATCH_LOG"
    echo "--- end execution watch output ---"
  fi

  if [[ -n "$WORKFLOW_ID" ]]; then
    API_PORT="${API_PORT:-}"
    if [[ -n "$API_PORT" ]]; then
      API_PORT="$API_PORT" browser-automation-studio workflow delete "$WORKFLOW_ID" >/dev/null 2>&1 || true
    fi
  fi

  if [[ -n "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
  fi

  # Stop scenario only if we started it in this script
  if [[ "${BAS_AUTOMATION_STOP_SCENARIO:-1}" == "1" ]]; then
    vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true
  fi

  exit $exit_code
}

trap cleanup EXIT

require_command() {
  local cmd=$1
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "❌ Required command '$cmd' is not installed" >&2
    exit 1
  fi
}

require_command vrooli
require_command browser-automation-studio
require_command jq
require_command curl

# Ensure scenario is not running so we can launch with the heartbeat override
vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true

export BROWSERLESS_HEARTBEAT_INTERVAL="$HEARTBEAT_ENV_VALUE"
BAS_AUTOMATION_STOP_SCENARIO=1 vrooli scenario start "$SCENARIO_NAME" --clean-stale >/dev/null

echo "⏳ Waiting for API to become healthy..."

API_PORT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT | awk -F= '/API_PORT/ {print $2}' | tr -d ' ')
WS_PORT=$(vrooli scenario port "$SCENARIO_NAME" WS_PORT | awk -F= '/WS_PORT/ {print $2}' | tr -d ' ')

if [[ -z "$API_PORT" ]]; then
  echo "❌ Unable to determine API_PORT" >&2
  exit 1
fi

API_URL="http://localhost:${API_PORT}"

if ! timeout 120 bash -c "until curl -sf '${API_URL}/health' >/dev/null; do sleep 2; done"; then
  echo "❌ API did not become healthy within timeout" >&2
  exit 1
fi

echo "✅ API healthy on ${API_URL}"

# Prepare temporary working area
TMP_DIR=$(mktemp -d)
FLOW_FILE="${TMP_DIR}/stall-flow.json"
WATCH_LOG="${TMP_DIR}/execution-watch.log"

cat <<'JSON' > "$FLOW_FILE"
{
  "nodes": [
    {
      "id": "navigate-start",
      "type": "navigate",
      "data": {
        "url": "https://example.com",
        "waitUntil": "domcontentloaded",
        "timeoutMs": 20000
      }
    },
    {
      "id": "wait-stall",
      "type": "wait",
      "data": {
        "type": "time",
        "duration": 20000
      }
    }
  ],
  "edges": [
    {
      "id": "edge-1",
      "source": "navigate-start",
      "target": "wait-stall"
    }
  ]
}
JSON

export API_PORT
if [[ -n "$WS_PORT" ]]; then
  export WS_URL="ws://localhost:${WS_PORT}"
fi

CREATE_OUTPUT=$(browser-automation-studio workflow create "$WORKFLOW_NAME" --folder "/automation" --from-file "$FLOW_FILE")
if ! echo "$CREATE_OUTPUT" | grep -q "Workflow ID"; then
  echo "❌ Failed to create workflow" >&2
  echo "$CREATE_OUTPUT" >&2
  exit 1
fi

WORKFLOW_ID=$(echo "$CREATE_OUTPUT" | awk -F'Workflow ID: ' '/Workflow ID/ {print $2}' | head -n1)
WORKFLOW_ID=$(echo "$WORKFLOW_ID" | tr -d '\r')
if [[ -z "$WORKFLOW_ID" ]]; then
  echo "❌ Could not parse workflow ID" >&2
  exit 1
fi

echo "🛠️  Created workflow ${WORKFLOW_ID}"

EXEC_OUTPUT=$(browser-automation-studio workflow execute "$WORKFLOW_ID")
if ! echo "$EXEC_OUTPUT" | grep -q "Execution ID"; then
  echo "❌ Failed to start execution" >&2
  echo "$EXEC_OUTPUT" >&2
  exit 1
fi

EXECUTION_ID=$(echo "$EXEC_OUTPUT" | awk -F'Execution ID: ' '/Execution ID/ {print $2}' | head -n1)
EXECUTION_ID=$(echo "$EXECUTION_ID" | tr -d '\r')
if [[ -z "$EXECUTION_ID" ]]; then
  echo "❌ Could not parse execution ID" >&2
  exit 1
fi

echo "🎬 Started execution ${EXECUTION_ID}"

# Start watch in background and capture output
( API_PORT="$API_PORT" browser-automation-studio execution watch "$EXECUTION_ID" >"$WATCH_LOG" 2>&1 ) &
WATCH_PID=$!

# Poll execution status until completion or timeout
MAX_POLLS=40
SLEEP_BETWEEN=5
SUCCESS=false

for ((i=1; i<=MAX_POLLS; i++)); do
  STATUS_RESPONSE=$(curl -sf "${API_URL}/api/v1/executions/${EXECUTION_ID}") || true
  STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status // empty')
  if [[ "$STATUS" == "completed" ]]; then
    SUCCESS=true
    break
  fi
  if [[ "$STATUS" == "failed" ]]; then
    echo "❌ Execution failed"
    echo "$STATUS_RESPONSE" | jq
    break
  fi
  sleep "$SLEEP_BETWEEN"
 done

wait "$WATCH_PID" || true

if [[ "$SUCCESS" != "true" ]]; then
  echo "❌ Execution did not complete successfully" >&2
  exit 1
fi

echo "🔍 Inspecting execution watch output for stall alerts"

if ! grep -q "heartbeat stalled" "$WATCH_LOG"; then
  echo "❌ Did not detect heartbeat stall messaging in watch output" >&2
  exit 1
fi

echo "✅ Heartbeat stall alert detected"

echo "Automation completed successfully"
