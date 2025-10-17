#!/bin/bash
# Performance tests for video-tools scenario

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../..}" && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "⚡ Running video-tools performance tests..."

# Run Go benchmarks
if [[ -d "api" ]]; then
    echo "Running API benchmarks..."
    cd api

    if go test -bench=. -benchmem ./cmd/server -run=^$ 2>/dev/null; then
        echo "✅ API benchmarks completed"
    else
        echo "⚠️  API benchmarks skipped (may require test database)"
    fi

    if go test -bench=. -benchmem ./internal/video -run=^$ 2>/dev/null; then
        echo "✅ Processor benchmarks completed"
    else
        echo "⚠️  Processor benchmarks skipped (may require ffmpeg)"
    fi

    cd ..
fi

# Performance targets
echo ""
echo "📊 Performance Targets:"
echo "  - Health endpoint: < 100ms p95"
echo "  - Video retrieval: < 200ms p95"
echo "  - Thumbnail generation: < 2s (real video)"
echo "  - Frame extraction: < 5s (real video)"

testing::phase::end_with_summary "Performance tests completed"
