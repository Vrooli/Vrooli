#!/bin/bash
# Business rule coverage for tech-tree-designer
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"
cd "${TESTING_PHASE_SCENARIO_DIR}"

testing::phase::check "Go business rule tests" bash -c 'cd api && go test -tags testing -run "^TestBusiness_" -v -timeout 90s'

testing::phase::end_with_summary "Business logic validation completed"
