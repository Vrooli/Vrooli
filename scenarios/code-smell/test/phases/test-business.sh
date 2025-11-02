#!/bin/bash
# Exercise core business workflows: rule management, CLI analysis, and learning APIs.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "240s" --require-runtime

SCENARIO_NAME="${TESTING_PHASE_SCENARIO_NAME}"

if ! command -v curl >/dev/null 2>&1; then
  testing::phase::add_error "curl CLI required for business workflow checks"
  testing::phase::end_with_summary "Business validation failed"
fi

API_PORT_OUTPUT=$(vrooli scenario port "$SCENARIO_NAME" API_PORT 2>/dev/null || true)
API_PORT=$(echo "$API_PORT_OUTPUT" | awk -F= '/=/{print $2}' | tr -d ' ')
if [ -z "$API_PORT" ]; then
  API_PORT=$(echo "$API_PORT_OUTPUT" | tr -d '[:space:]')
fi

if [ -z "$API_PORT" ]; then
  testing::phase::add_error "Unable to resolve API_PORT for $SCENARIO_NAME"
  testing::phase::end_with_summary "Business validation incomplete"
fi

API_URL="http://localhost:${API_PORT}/api/v1"
SCENARIO_DIR="$TESTING_PHASE_SCENARIO_DIR"
RULE_FILE="$SCENARIO_DIR/initialization/rules/_auto-test-rule.yaml"

cleanup_rule_file() {
  rm -f "$RULE_FILE" 2>/dev/null || true
}

testing::phase::register_cleanup cleanup_rule_file

# CLI analysis check (ensures CLI can talk to running API)
if [ -x "$SCENARIO_DIR/cli/code-smell" ]; then
  testing::phase::check "CLI analyze command succeeds" bash -c 'cd "$0" && ./cli/code-smell analyze ./lib >/dev/null' "$SCENARIO_DIR"
else
  testing::phase::add_warning "cli/code-smell binary not executable; skipping CLI analyze"
  testing::phase::add_test skipped
fi

# API analyze endpoint returns violations payload
ANALYZE_PAYLOAD='{"paths":["./lib/rules-engine.js"],"auto_fix":false}'
if command -v jq >/dev/null 2>&1; then
  testing::phase::check "API analyze endpoint returns JSON" bash -c 'curl -fsS -X POST -H "Content-Type: application/json" --data "$1" "$0" | jq -e ".violations" >/dev/null' "$API_URL/code-smell/analyze" "$ANALYZE_PAYLOAD"
else
  testing::phase::check "API analyze endpoint responds" bash -c 'curl -fsS -X POST -H "Content-Type: application/json" --data "$1" "$0" | grep -q "violations"' "$API_URL/code-smell/analyze" "$ANALYZE_PAYLOAD"
fi

# Hot-reload validation: append a temporary rule and confirm it becomes visible
if [ -x "$SCENARIO_DIR/cli/code-smell" ]; then
  cat <<YAML >"$RULE_FILE"
name: Auto Test Rule
pattern: AUTOTEST_TOKEN
risk_level: safe
fix:
  description: Remove AUTOTEST_TOKEN marker
YAML
  sleep 2
  testing::phase::check "Hot reload exposes new rule" bash -c 'cd "$0" && ./cli/code-smell rules list | grep -q "Auto Test Rule"' "$SCENARIO_DIR"
else
  testing::phase::add_warning "cli/code-smell binary not executable; skipping hot-reload validation"
  testing::phase::add_test skipped
fi

# Learning endpoint accepts feedback
LEARN_PAYLOAD='{"pattern":"console.log","is_positive":false,"context":{"description":"debug logging"}}'
if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Pattern learning endpoint" bash -c 'curl -fsS -X POST -H "Content-Type: application/json" --data "$1" "$0" | jq -e ".success == true" >/dev/null' "$API_URL/code-smell/learn" "$LEARN_PAYLOAD"
else
  testing::phase::check "Pattern learning endpoint" bash -c 'curl -fsS -X POST -H "Content-Type: application/json" --data "$1" "$0" | grep -q "\"success\":true"' "$API_URL/code-smell/learn" "$LEARN_PAYLOAD"
fi

# Statistics endpoint basic sanity
if command -v jq >/dev/null 2>&1; then
  testing::phase::check "Statistics endpoint" bash -c 'curl -fsS "$0" | jq -e ".files_analyzed" >/dev/null' "$API_URL/code-smell/stats?period=all"
else
  testing::phase::check "Statistics endpoint" bash -c 'curl -fsS "$0" | grep -q "files_analyzed"' "$API_URL/code-smell/stats?period=all"
fi

testing::phase::end_with_summary "Business workflow validation completed"
