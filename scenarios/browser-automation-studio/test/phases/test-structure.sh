#!/bin/bash
# Validates scenario structure
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/structure.sh"

testing::phase::init --target-time "30s"

testing::structure::validate_all --scenario "$TESTING_PHASE_SCENARIO_NAME"

testing::phase::end_with_summary "Structure validation completed"
