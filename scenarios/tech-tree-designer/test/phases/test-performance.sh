#!/bin/bash
# Performance and benchmarking for tech-tree-designer
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "150s"
cd "${TESTING_PHASE_SCENARIO_DIR}"

testing::phase::check "Go performance tests" bash -c 'cd api && go test -tags testing -run "^TestPerformance_" -v -timeout 120s'

testing::phase::check "Go benchmarks" bash -c 'cd api && go test -tags testing -bench=. -benchtime=1s -run=^$ -timeout 120s'

testing::phase::end_with_summary "Performance validation completed"
