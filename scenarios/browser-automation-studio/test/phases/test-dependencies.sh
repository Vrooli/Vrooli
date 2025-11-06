#!/bin/bash
# Dependencies validation using unified helper
# Validates runtimes, package managers, resources, and connectivity
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/dependencies.sh"

testing::phase::init --target-time "60s"

testing::dependencies::validate_all \
  --scenario "$TESTING_PHASE_SCENARIO_NAME"

testing::phase::end_with_summary "Dependency validation completed"
