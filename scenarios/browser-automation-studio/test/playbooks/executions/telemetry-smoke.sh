#!/usr/bin/env bash
set -euo pipefail

SCENARIO_NAME="browser-automation-studio"
WORKFLOW_NAME="automation-telemetry-smoke-$(date +%s)"
TMP_DIR=""
API_PORT=""
API_URL=""
BASE_URL=""
WORKFLOW_ID=""
EXECUTION_ID=""
TIMELINE_FILE=""
STOP_SCENARIO="${BAS_AUTOMATION_STOP_SCENARIO:-1}"
EXPORT_FILE=""
PROJECT_ID=""
CREATED_PROJECT_ID=""
PROJECT_DIR=""
HTTP_SERVER_PID=""
SERVER_PORT=""

# Deterministic HTML page (with console logs + fetch) served over localhost for the telemetry run

cleanup() {
  local exit_code=$?

  if [[ -n "$TIMELINE_FILE" && -f "$TIMELINE_FILE" ]]; then
    echo "--- timeline payload ---"
    cat "$TIMELINE_FILE"
    echo "--- end timeline payload ---"
  fi

  if [[ -n "$EXPORT_FILE" && -f "$EXPORT_FILE" ]]; then
    echo "--- export payload ---"
    cat "$EXPORT_FILE"
    echo "--- end export payload ---"
  fi

  if [[ -n "$WORKFLOW_ID" && -n "$API_URL" ]]; then
    curl -sS -X DELETE "${API_URL}/workflows/${WORKFLOW_ID}" >/dev/null 2>&1 || true
  fi

  if [[ -n "$HTTP_SERVER_PID" ]]; then
    kill "$HTTP_SERVER_PID" >/dev/null 2>&1 || true
    wait "$HTTP_SERVER_PID" >/dev/null 2>&1 || true
  fi

  if [[ -n "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
  fi

  if [[ -n "$PROJECT_DIR" && -d "$PROJECT_DIR" ]]; then
    rm -rf "$PROJECT_DIR"
  fi

  if [[ -n "$CREATED_PROJECT_ID" && -n "$API_URL" ]]; then
    curl -sS -X DELETE "${API_URL}/projects/${CREATED_PROJECT_ID}" >/dev/null 2>&1 || true
  fi

  if [[ "$STOP_SCENARIO" == "1" ]]; then
    vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true
  fi

  exit $exit_code
}

trap cleanup EXIT

require_command() {
  local cmd="$1"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "❌ Required command '$cmd' is missing" >&2
    exit 1
  fi
}

require_command vrooli
require_command curl
require_command jq
require_command python3

vrooli scenario stop "$SCENARIO_NAME" >/dev/null 2>&1 || true
vrooli scenario start "$SCENARIO_NAME" --clean-stale >/dev/null

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [[ -z "$API_PORT" ]]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [[ -z "$API_PORT" ]]; then
  echo "❌ Unable to resolve API_PORT for scenario" >&2
  exit 1
fi

BASE_URL="http://localhost:${API_PORT}"
API_URL="${BASE_URL}/api/v1"

if ! timeout 120 bash -c "until curl -sf '${BASE_URL}/health' >/dev/null; do sleep 2; done"; then
  echo "❌ API did not become healthy in time" >&2
  exit 1
fi

echo "✅ API healthy at ${BASE_URL}"

PROJECTS_RESPONSE=$(curl -sS --fail "${API_URL}/projects")
PROJECT_ID=$(echo "$PROJECTS_RESPONSE" | jq -r '.projects[0].id // empty')

if [[ -z "$PROJECT_ID" ]]; then
  PROJECT_DIR=$(mktemp -d)
  PROJECT_NAME="telemetry-project-${WORKFLOW_NAME}"
  PROJECT_PAYLOAD=$(jq -n \
    --arg name "$PROJECT_NAME" \
    --arg folder "$PROJECT_DIR" \
    '{name: $name, folder_path: $folder}')

  PROJECT_RESPONSE=$(curl -sS --fail -X POST \
    -H "Content-Type: application/json" \
    -d "$PROJECT_PAYLOAD" \
    "${API_URL}/projects")

  PROJECT_ID=$(echo "$PROJECT_RESPONSE" | jq -r '.project.id // .project_id // empty')
  if [[ -z "$PROJECT_ID" ]]; then
    echo "❌ Failed to resolve project ID" >&2
    echo "$PROJECT_RESPONSE" >&2
    exit 1
  fi
  CREATED_PROJECT_ID="$PROJECT_ID"
else
  PROJECT_DIR=""
fi

echo "📁 Using project ${PROJECT_ID}"

TMP_DIR=$(mktemp -d)
FLOW_FILE="${TMP_DIR}/flow.json"
TIMELINE_FILE="${TMP_DIR}/timeline.json"
EXPORT_FILE="${TMP_DIR}/export.json"
HTML_FILE="${TMP_DIR}/telemetry.html"

cat <<'HTML' > "$HTML_FILE"
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>BAS Telemetry Smoke</title>
    <style>
      body {
        font-family: sans-serif;
        background: #0f172a;
        color: #e2e8f0;
        text-align: center;
        padding-top: 80px;
      }
      .hero {
        margin: auto;
        max-width: 640px;
        padding: 40px;
        background: #1e293b;
        border-radius: 16px;
        box-shadow: 0 20px 40px rgba(15, 23, 42, 0.4);
      }
      .hero h1 {
        font-size: 2.5rem;
        margin-bottom: 12px;
      }
      .hero p {
        font-size: 1.1rem;
        line-height: 1.6;
      }
      .hero a {
        display: inline-block;
        margin-top: 24px;
        padding: 12px 24px;
        background: #22d3ee;
        color: #0f172a;
        border-radius: 9999px;
        text-decoration: none;
        font-weight: 600;
      }
    </style>
    <script>
      console.log('BAS telemetry smoke console log');
      fetch('https://httpbin.org/get?source=bas-telemetry-smoke')
        .then((r) => r.json())
        .then((data) => console.log('Fetched', data.url))
        .catch((err) => console.error('fetch error', err));
    </script>
  </head>
  <body>
    <div class="hero" id="hero">
      <h1>Browser Automation Studio</h1>
      <p id="subtitle">Telemetry smoke workflow</p>
      <a href="https://example.com" id="cta">Explore more</a>
    </div>
  </body>
</html>
HTML

SERVER_PORT=$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(('127.0.0.1', 0))
port = s.getsockname()[1]
s.close()
print(port)
PY
)

python3 -m http.server "$SERVER_PORT" --bind 127.0.0.1 --directory "$TMP_DIR" >/dev/null 2>&1 &
HTTP_SERVER_PID=$!

if ! timeout 15 bash -c "until curl -sf 'http://127.0.0.1:${SERVER_PORT}/telemetry.html' >/dev/null; do sleep 1; done"; then
  echo "❌ Local telemetry test page did not start" >&2
  exit 1
fi

DATA_URL="http://127.0.0.1:${SERVER_PORT}/telemetry.html"

cat <<JSON > "$FLOW_FILE"
{
  "nodes": [
    {
      "id": "navigate-start",
      "type": "navigate",
      "position": { "x": -240, "y": -60 },
      "data": {
        "label": "Navigate",
        "url": "$DATA_URL",
        "waitUntil": "networkidle2",
        "timeoutMs": 20000
      }
    },
    {
      "id": "wait-ready",
      "type": "wait",
      "position": { "x": 0, "y": -60 },
      "data": {
        "label": "Wait for page",
        "waitType": "time",
        "duration": 1000
      }
    },
    {
      "id": "wait-settle",
      "type": "wait",
      "position": { "x": 200, "y": -60 },
      "data": {
        "label": "Wait for fetch",
        "waitType": "time",
        "duration": 1500
      }
    },
    {
      "id": "screenshot-annotated",
      "type": "screenshot",
      "position": { "x": 360, "y": -60 },
      "data": {
        "label": "Capture annotated",
        "focusSelector": "#hero",
        "highlightSelectors": ["#hero h1", "#subtitle"],
        "maskSelectors": ["#cta"],
        "highlightColor": "#22d3ee",
        "highlightPadding": 16,
        "maskOpacity": 0.5,
        "zoomFactor": 1.12,
        "background": "#0f172a",
        "waitForMs": 400
      }
    }
  ],
  "edges": [
    { "id": "edge-navigate-wait", "source": "navigate-start", "target": "wait-ready" },
    { "id": "edge-wait-chain", "source": "wait-ready", "target": "wait-settle" },
    { "id": "edge-wait-screenshot", "source": "wait-settle", "target": "screenshot-annotated" }
  ]
}
JSON

CREATE_PAYLOAD=$(jq -n \
  --arg name "$WORKFLOW_NAME" \
  --arg project "$PROJECT_ID" \
  --arg folder "/automation" \
  --slurpfile flow "$FLOW_FILE" \
  '{name: $name, project_id: $project, folder_path: $folder, flow_definition: $flow[0]}')

CREATE_RESPONSE=$(curl -sS --fail -X POST \
  -H "Content-Type: application/json" \
  -d "$CREATE_PAYLOAD" \
  "${API_URL}/workflows/create")

WORKFLOW_ID=$(echo "$CREATE_RESPONSE" | jq -r '.workflow_id // .id // empty')
if [[ -z "$WORKFLOW_ID" ]]; then
  echo "❌ Failed to parse workflow ID" >&2
  echo "$CREATE_RESPONSE" >&2
  exit 1
fi

echo "🛠️  Created workflow ${WORKFLOW_ID}"

EXECUTE_RESPONSE=$(curl -sS --fail -X POST \
  -H "Content-Type: application/json" \
  -d '{"parameters": {}, "wait_for_completion": false}' \
  "${API_URL}/workflows/${WORKFLOW_ID}/execute")

EXECUTION_ID=$(echo "$EXECUTE_RESPONSE" | jq -r '.execution_id // empty')
if [[ -z "$EXECUTION_ID" ]]; then
  echo "❌ Failed to start execution" >&2
  echo "$EXECUTE_RESPONSE" >&2
  exit 1
fi

echo "🎬 Execution started ${EXECUTION_ID}"

MAX_POLLS=90
SLEEP_SEC=2
EXECUTION_STATUS="running"
for ((i=1; i<=MAX_POLLS; i++)); do
  STATUS_RESPONSE=$(curl -sS --fail "${API_URL}/executions/${EXECUTION_ID}")
  EXECUTION_STATUS=$(echo "$STATUS_RESPONSE" | jq -r '.status // ""')
  PROGRESS=$(echo "$STATUS_RESPONSE" | jq -r '.progress // 0')
  CURRENT_STEP=$(echo "$STATUS_RESPONSE" | jq -r '.current_step // ""')
  printf "\r⏱️  status=%s progress=%s%% step=%s" "$EXECUTION_STATUS" "$PROGRESS" "$CURRENT_STEP"

  if [[ "$EXECUTION_STATUS" == "completed" ]]; then
    echo
    break
  fi
  if [[ "$EXECUTION_STATUS" == "failed" ]]; then
    echo
    echo "❌ Execution failed" >&2
    echo "$STATUS_RESPONSE" >&2
    exit 1
  fi
  sleep "$SLEEP_SEC"
done

echo
if [[ "$EXECUTION_STATUS" != "completed" ]]; then
  echo "❌ Execution did not complete within timeout" >&2
  exit 1
fi

echo "✅ Execution completed"

curl -sS --fail "${API_URL}/executions/${EXECUTION_ID}/timeline" | tee "$TIMELINE_FILE" >/dev/null

FRAME_COUNT=$(jq '.frames | length' "$TIMELINE_FILE")
if [[ "$FRAME_COUNT" -le 0 ]]; then
  echo "❌ Timeline has no frames" >&2
  exit 1
fi

echo "📸 Timeline includes ${FRAME_COUNT} frames"

SCREENSHOT_COUNT=$(jq '[.frames[] | select(.screenshot.artifact_id != null and .screenshot.artifact_id != "")] | length' "$TIMELINE_FILE")
if [[ "$SCREENSHOT_COUNT" -le 0 ]]; then
  echo "❌ No screenshot artifacts present" >&2
  exit 1
fi

echo "🖼️  Screenshot frames detected: ${SCREENSHOT_COUNT}"

CONSOLE_FRAMES=$(jq '[.frames[] | select((.console_log_count // 0) > 0)] | length' "$TIMELINE_FILE")
if [[ "$CONSOLE_FRAMES" -le 0 ]]; then
  echo "❌ Console telemetry missing from timeline" >&2
  exit 1
fi

echo "🗒️  Console telemetry frames detected: ${CONSOLE_FRAMES}"

NETWORK_FRAMES=$(jq '[.frames[] | select((.network_event_count // 0) > 0)] | length' "$TIMELINE_FILE")
if [[ "$NETWORK_FRAMES" -le 0 ]]; then
  echo "❌ Network telemetry missing from timeline" >&2
  exit 1
fi

echo "🌐 Network telemetry frames detected: ${NETWORK_FRAMES}"

curl -sS --fail -X POST "${API_URL}/executions/${EXECUTION_ID}/export" | tee "$EXPORT_FILE" >/dev/null

EXPORT_FRAME_COUNT=$(jq '.package.frames | length' "$EXPORT_FILE")
if [[ "$EXPORT_FRAME_COUNT" -le 0 ]]; then
  echo "❌ Replay export contains no frames" >&2
  exit 1
fi

EXPORT_SCREENSHOTS=$(jq '.package.summary.screenshot_count // 0' "$EXPORT_FILE")
if [[ "$EXPORT_SCREENSHOTS" -le 0 ]]; then
  echo "❌ Replay export summary missing screenshot count" >&2
  exit 1
fi

EXPORT_ASSETS=$(jq '.package.assets | length' "$EXPORT_FILE")
if [[ "$EXPORT_ASSETS" -le 0 ]]; then
  echo "❌ Replay export lacks asset manifest" >&2
  exit 1
fi

CURSOR_STYLE=$(jq -r '.package.cursor.style // empty' "$EXPORT_FILE")
if [[ -z "$CURSOR_STYLE" ]]; then
  echo "❌ Replay export missing cursor configuration" >&2
  exit 1
fi

echo "📦 Replay export validated (${EXPORT_FRAME_COUNT} frames, ${EXPORT_ASSETS} assets)"

echo "🎯 Telemetry smoke check passed"

TIMELINE_FILE="" # suppress dumping on successful exit
EXPORT_FILE=""
