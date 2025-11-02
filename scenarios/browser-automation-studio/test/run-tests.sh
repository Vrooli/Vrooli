#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/runner.sh"

# Enforce requirement coverage at phase boundaries so missing evidence fails
# the suite. Individual phases can opt-out by exporting
# TESTING_REQUIREMENTS_ENFORCE=0 before ending.
export TESTING_REQUIREMENTS_ENFORCE=1

SCENARIO_NAME="browser-automation-studio"

# Initialize shared runner with runtime management enabled so integration phases
# can assume the scenario is available.
testing::runner::init \
  --scenario-name "$SCENARIO_NAME" \
  --scenario-dir "$SCENARIO_DIR" \
  --test-dir "$TEST_DIR" \
  --log-dir "$TEST_DIR/artifacts" \
  --default-manage-runtime true

# Register standard phase lineup. Timeouts mirror historical expectations while
# enabling runtime-aware phases to opt in via --requires-runtime.
testing::runner::register_phase --name structure --script "$TEST_DIR/phases/test-structure.sh" --timeout 30

testing::runner::register_phase --name dependencies --script "$TEST_DIR/phases/test-dependencies.sh" --timeout 60

testing::runner::register_phase --name unit --script "$TEST_DIR/phases/test-unit.sh" --timeout 120

testing::runner::register_phase --name integration --script "$TEST_DIR/phases/test-integration.sh" --timeout 240 --requires-runtime true

testing::runner::register_phase --name business --script "$TEST_DIR/phases/test-business.sh" --timeout 120

testing::runner::register_phase --name performance --script "$TEST_DIR/phases/test-performance.sh" --timeout 60 --requires-runtime true

# Presets for quick iteration.
testing::runner::define_preset quick "structure unit"
testing::runner::define_preset smoke "structure integration"
testing::runner::define_preset comprehensive "structure dependencies unit integration business performance"

RESULT=0
testing::runner::execute "$@" || RESULT=$?

if { [ -f "$SCENARIO_DIR/docs/requirements.yaml" ] || [ -d "$SCENARIO_DIR/requirements" ]; } \
  && command -v node >/dev/null 2>&1; then
  if node "$APP_ROOT/scripts/requirements/report.js" --scenario "$SCENARIO_NAME" --mode sync >/dev/null; then
    log::info "📋 Requirements registry synced after test run"
  else
    log::warning "⚠️  Failed to sync requirements registry; leaving requirement files unchanged"
  fi
else
  log::warning "⚠️  Skipping requirements sync (missing requirements registry or Node runtime)"
fi

exit $RESULT
