#!/usr/bin/env bash
set -euo pipefail

SCENARIO_NAME="browser-automation-studio"
STOP_SCENARIO="${BAS_AUTOMATION_STOP_SCENARIO:-1}"
PROJECT_NAME="Demo Browser Automations"
WORKFLOW_NAME="Demo: Capture Example.com Hero"
API_PORT=""
BASE_URL=""
API_URL=""
PROJECT_ID=""

cleanup() {
  local exit_code=$?

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

API_PORT=$(vrooli scenario port "$SCENARIO_NAME" | awk -F= '/API_PORT/ {print $2}' | tr -d ' ')
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
PROJECT_ID=$(echo "$PROJECTS_RESPONSE" | jq -r --arg name "$PROJECT_NAME" '.projects[] | select(.project.name == $name or .name == $name) | (.project.id // .id)')
if [[ -z "$PROJECT_ID" || "$PROJECT_ID" == "null" ]]; then
  echo "❌ Demo project '${PROJECT_NAME}' not found" >&2
  echo "$PROJECTS_RESPONSE" >&2
  exit 1
fi

PROJECT_DETAILS=$(echo "$PROJECTS_RESPONSE" | jq -r --arg id "$PROJECT_ID" '.projects[] | select((.project.id // .id) == $id)')
PROJECT_PATH=$(echo "$PROJECT_DETAILS" | jq -r '.project.folder_path // .folder_path // empty')
if [[ -z "$PROJECT_PATH" || "$PROJECT_PATH" == "null" ]]; then
  echo "⚠️  Demo project located but folder path missing"
else
  echo "📁 Demo project folder: ${PROJECT_PATH}"
fi

WORKFLOWS_RESPONSE=$(curl -sS --fail "${API_URL}/projects/${PROJECT_ID}/workflows")
WORKFLOW_ID=$(echo "$WORKFLOWS_RESPONSE" | jq -r --arg name "$WORKFLOW_NAME" '.workflows[] | select(.name == $name) | .id')
if [[ -z "$WORKFLOW_ID" || "$WORKFLOW_ID" == "null" ]]; then
  echo "❌ Demo workflow '${WORKFLOW_NAME}' not found" >&2
  echo "$WORKFLOWS_RESPONSE" >&2
  exit 1
fi

echo "🎯 Demo project and workflow available (project_id=${PROJECT_ID}, workflow_id=${WORKFLOW_ID})"
echo "You can open the dashboard and run '${WORKFLOW_NAME}' immediately."
