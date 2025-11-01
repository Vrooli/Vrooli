#!/bin/bash
# Runs Go benchmarks to establish baseline performance characteristics.

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

if testing::phase::check "Go benchmark suite" bash -c 'cd api && go test -bench=. -benchmem ./...'; then
  :
fi

testing::phase::end_with_summary "Performance benchmarks completed"
