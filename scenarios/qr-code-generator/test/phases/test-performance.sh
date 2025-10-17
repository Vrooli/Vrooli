#!/bin/bash
# QR Code Generator - Performance Test Phase

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "30s"

cd "$TESTING_PHASE_SCENARIO_DIR/api"

echo "Running Go benchmarks..."
go test -bench=. -benchmem -benchtime=3s 2>&1 | tee /tmp/qr-bench.txt

# Check if benchmarks completed
if [ $? -eq 0 ]; then
    testing::phase::record_success "Benchmark tests completed"
else
    testing::phase::record_failure "Benchmark tests failed"
fi

testing::phase::end_with_summary "Performance tests completed"
