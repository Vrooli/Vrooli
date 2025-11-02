#!/bin/bash
# Exercises user-facing workflows via CLI and UI surfaces.
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s" --require-runtime

scenario_name="$TESTING_PHASE_SCENARIO_NAME"
api_base=""
ui_base=""

if ! api_base=$(testing::connectivity::get_api_url "$scenario_name"); then
  testing::phase::add_error "Unable to discover API URL; business validations cannot continue"
  testing::phase::end_with_summary "Business validations incomplete"
fi

ui_available=true
if ! ui_base=$(testing::connectivity::get_ui_url "$scenario_name"); then
  testing::phase::add_warning "UI port unavailable; skipping UI smoke checks"
  ui_available=false
fi

has_jq=false
if command -v jq >/dev/null 2>&1; then
  has_jq=true
fi

if [ -x "./cli/scalable-app-cookbook" ]; then
  if $has_jq; then
    testing::phase::check "CLI status reports healthy" bash -c \
      "./cli/scalable-app-cookbook status --json | jq -e '.status == \"healthy\"' >/dev/null"
  else
    testing::phase::check "CLI status command" bash -c \
      "./cli/scalable-app-cookbook status --json | grep -qi healthy"
  fi

  if $has_jq; then
    testing::phase::check "CLI search returns patterns" bash -c \
      "./cli/scalable-app-cookbook search jwt --json | jq -e '.patterns | length > 0' >/dev/null"
  else
    testing::phase::check "CLI search command" bash -c \
      "./cli/scalable-app-cookbook search jwt --json | grep -qi patterns"
  fi

  testing::phase::check "CLI chapters list" bash -c \
    "./cli/scalable-app-cookbook chapters --json | grep -qi 'Part A'"

  testing::phase::check "CLI help lists commands" bash -c \
    "./cli/scalable-app-cookbook help | grep -q 'Scalable App Cookbook CLI'"
else
  testing::phase::add_warning "CLI executable missing; skipping CLI validations"
  testing::phase::add_test skipped
fi

if $ui_available; then
  testing::phase::check "UI serves main page" bash -c \
    "curl -sf --retry 3 --retry-delay 2 '${ui_base}/' | grep -q 'Scalable App Cookbook'"
fi

if $has_jq; then
  testing::phase::check "API stats match CLI expectations" bash -c \
    "curl -sf '${api_base}/api/v1/patterns/stats' | jq -e '.statistics.total_patterns >= 8 and .statistics.chapters >= 1' >/dev/null"
else
  testing::phase::check "API stats endpoint" curl -sf "${api_base}/api/v1/patterns/stats"
fi

testing::phase::end_with_summary "Business workflow validation completed"
