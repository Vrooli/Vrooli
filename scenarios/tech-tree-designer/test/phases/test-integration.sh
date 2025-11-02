#!/bin/bash
# Runs integration suite for tech-tree-designer
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"
cd "${TESTING_PHASE_SCENARIO_DIR}"

testing::phase::check "Go integration tests" bash -c 'cd api && go test -tags testing -run "^TestIntegration_" -v -timeout 90s'

testing::phase::end_with_summary "Integration validation completed"
