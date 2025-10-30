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

# Deterministic HTML page (with console logs + fetch) encoded as data URL
DATA_URL="data:text/html;base64,PCFkb2N0eXBlIGh0bWw+PGh0bWw+PGhlYWQ+PG1ldGEgY2hhcnNldD0ndXRmLTgnPjx0aXRsZT5CQVMgVGVsZW1ldHJ5IFNtb2tlPC90aXRsZT48c3R5bGU+Ym9keXtmb250LWZhbWlseTpzYW5zLXNlcmlmO2JhY2tncm91bmQ6IzBmMTcyYTtjb2xvcjojZTJlOGYwO3RleHQtYWxpZ246Y2VudGVyO3BhZGRpbmctdG9wOjgwcHg7fSAuaGVyb3ttYXJnaW46YXV0bzttYXgtd2lkdGg6NjQwcHg7cGFkZGluZzo0MHB4O2JhY2tncm91bmQ6IzFlMjkzYjtib3JkZXItcmFkaXVzOjE2cHg7Ym94LXNoYWRvdzowIDIwcHggNDBweCByZ2JhKDE1LDIzLDQyLC40KTt9IC5oZXJvIGgxe2ZvbnQtc2l6ZToyLjVyZW07bWFyZ2luLWJvdHRvbToxMnB4O30gLmhlcm8gcHtmb250LXNpemU6MS4xcmVtO2xpbmUtaGVpZ2h0OjEuNjt9IC5oZXJvIGF7ZGlzcGxheTppbmxpbmUtYmxvY2s7bWFyZ2luLXRvcDoyNHB4O3BhZGRpbmc6MTJweCAyNHB4O2JhY2tncm91bmQ6IzIyZDNlZTtjb2xvcjojMGYxNzJhO2JvcmRlci1yYWRpdXM6OTk5OXB4O3RleHQtZGVjb3JhdGlvbjpub25lO2ZvbnQtd2VpZ2h0OjYwMDt9PC9zdHlsZT48c2NyaXB0PmNvbnNvbGUubG9nKCdCQVMgdGVsZW1ldHJ5IHNtb2tlIGNvbnNvbGUgbG9nJyk7ZmV0Y2goJ2h0dHBzOi8vaHR0cGJpbi5vcmcvZ2V0P3NvdXJjZT1iYXMtdGVsZW1ldHJ5LXNtb2tlJykudGhlbihyPT5yLmpzb24oKSkudGhlbihkYXRhPT5jb25zb2xlLmxvZygnRmV0Y2hlZCcsZGF0YS51cmwpKS5jYXRjaChlcnI9PmNvbnNvbGUuZXJyb3IoJ2ZldGNoIGVycm9yJyxlcnIpKTs8L3NjcmlwdD48L2hlYWQ+PGJvZHk+PGRpdiBjbGFzcz0naGVybycgaWQ9J2hlcm8nPjxoMT5Ccm93c2VyIEF1dG9tYXRpb24gU3R1ZGlvPC9oMT48cCBpZD0nc3VidGl0bGUnPlRlbGVtZXRyeSBzbW9rZSB3b3JrZmxvdzwvcD48YSBocmVmPSdodHRwczovL2V4YW1wbGUuY29tJyBpZD0nY3RhJz5FeHBsb3JlIG1vcmU8L2E+PC9kaXY+PC9ib2R5PjwvaHRtbD4="

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

  if [[ -n "$TMP_DIR" ]]; then
    rm -rf "$TMP_DIR"
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

TMP_DIR=$(mktemp -d)
FLOW_FILE="${TMP_DIR}/flow.json"
TIMELINE_FILE="${TMP_DIR}/timeline.json"
EXPORT_FILE="${TMP_DIR}/export.json"

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
      "id": "wait-hero",
      "type": "wait",
      "position": { "x": 0, "y": -60 },
      "data": {
        "label": "Wait for hero",
        "waitType": "element",
        "selector": "#hero",
        "timeoutMs": 15000
      }
    },
    {
      "id": "assert-hero",
      "type": "assert",
      "position": { "x": 200, "y": -60 },
      "data": {
        "label": "Hero text",
        "selector": "#hero h1",
        "assertMode": "text_contains",
        "expectedValue": "Browser Automation Studio",
        "timeoutMs": 15000,
        "failureMessage": "Hero heading did not match"
      }
    },
    {
      "id": "wait-settle",
      "type": "wait",
      "position": { "x": 320, "y": -60 },
      "data": {
        "label": "Wait for fetch",
        "waitType": "time",
        "duration": 1500
      }
    },
    {
      "id": "screenshot-annotated",
      "type": "screenshot",
      "position": { "x": 480, "y": -60 },
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
    { "id": "edge-navigate-wait", "source": "navigate-start", "target": "wait-hero" },
    { "id": "edge-wait-assert", "source": "wait-hero", "target": "assert-hero" },
    { "id": "edge-assert-wait", "source": "assert-hero", "target": "wait-settle" },
    { "id": "edge-wait-screenshot", "source": "wait-settle", "target": "screenshot-annotated" }
  ]
}
JSON

CREATE_PAYLOAD=$(jq -n \
  --arg name "$WORKFLOW_NAME" \
  --arg folder "/automation" \
  --slurpfile flow "$FLOW_FILE" \
  '{name: $name, folder_path: $folder, flow_definition: $flow[0]}')

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

HIGHLIGHT_COUNT=$(jq '[.frames[] | select(((.highlight_regions // []) | length) > 0)] | length' "$TIMELINE_FILE")
if [[ "$HIGHLIGHT_COUNT" -le 0 ]]; then
  echo "❌ No frame contains highlight metadata" >&2
  exit 1
fi

echo "✨ Highlight frames detected: ${HIGHLIGHT_COUNT}"

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

ASSERTION_PASSES=$(jq '[.frames[] | select(.step_type == "assert" and (.assertion.success == true))] | length' "$TIMELINE_FILE")
if [[ "$ASSERTION_PASSES" -le 0 ]]; then
  echo "❌ Assertion step missing or did not pass" >&2
  exit 1
fi

echo "✅ Assertion metadata captured"

curl -sS --fail -X POST "${API_URL}/executions/${EXECUTION_ID}/export" | tee "$EXPORT_FILE" >/dev/null

EXPORT_FRAME_COUNT=$(jq '.frames | length' "$EXPORT_FILE")
if [[ "$EXPORT_FRAME_COUNT" -le 0 ]]; then
  echo "❌ Replay export contains no frames" >&2
  exit 1
fi

EXPORT_SCREENSHOTS=$(jq '.summary.screenshot_count // 0' "$EXPORT_FILE")
if [[ "$EXPORT_SCREENSHOTS" -le 0 ]]; then
  echo "❌ Replay export summary missing screenshot count" >&2
  exit 1
fi

EXPORT_ASSETS=$(jq '.assets | length' "$EXPORT_FILE")
if [[ "$EXPORT_ASSETS" -le 0 ]]; then
  echo "❌ Replay export lacks asset manifest" >&2
  exit 1
fi

HIGHLIGHT_EXPORT=$(jq '[.frames[] | select(((.highlight_regions // []) | length) > 0)] | length' "$EXPORT_FILE")
if [[ "$HIGHLIGHT_EXPORT" -le 0 ]]; then
  echo "❌ Replay export frames missing highlight metadata" >&2
  exit 1
fi

CURSOR_STYLE=$(jq -r '.cursor.style // empty' "$EXPORT_FILE")
if [[ -z "$CURSOR_STYLE" ]]; then
  echo "❌ Replay export missing cursor configuration" >&2
  exit 1
fi

echo "📦 Replay export validated (${EXPORT_FRAME_COUNT} frames, ${EXPORT_ASSETS} assets)"

echo "🎯 Telemetry smoke check passed"

TIMELINE_FILE="" # suppress dumping on successful exit
EXPORT_FILE=""
