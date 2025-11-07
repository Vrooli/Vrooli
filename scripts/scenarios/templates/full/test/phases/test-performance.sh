#!/bin/bash
# Performance validation including Lighthouse, bundle size, benchmarks, and load tests
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/performance.sh"

testing::phase::init --target-time "180s" --require-runtime

testing::performance::validate_all \
  --scenario "$TESTING_PHASE_SCENARIO_NAME"

testing::phase::end_with_summary "Performance validation completed"
