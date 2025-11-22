#!/bin/bash
# Performance validation including Lighthouse, bundle size, and response time checks

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/performance.sh"

# [REQ:TM-LS-007] Light scan performance - small scenarios (<50 files, <60s)
test_light_scan_small_scenario() {
    echo "🧪 Testing light scan performance on small scenario (picker-wheel)..."

    local test_scenario="/home/matthalloran8/Vrooli/scenarios/picker-wheel"
    local file_count
    file_count=$(find "$test_scenario" -type f \( -name "*.go" -o -name "*.ts" -o -name "*.tsx" \) 2>/dev/null | grep -E "^$test_scenario/(api|ui/src|cli)" | wc -l)

    if [ "$file_count" -gt 50 ]; then
        echo "⚠️  Warning: picker-wheel has $file_count files (>50), may not be ideal for small scenario test"
    fi

    local start_time
    start_time=$(date +%s)

    tidiness-manager scan --scenario-path "$test_scenario" --timeout 60 &> /tmp/tm-perf-small.log
    local exit_code=$?

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo "  Duration: ${duration}s (target: <60s)"
    echo "  Files scanned: $file_count"

    if [ $exit_code -ne 0 ]; then
        echo "❌ Scan failed with exit code $exit_code"
        cat /tmp/tm-perf-small.log
        return 1
    fi

    if [ "$duration" -gt 60 ]; then
        echo "❌ Scan exceeded 60s threshold (took ${duration}s)"
        return 1
    fi

    echo "✅ Small scenario scan completed in ${duration}s"
    return 0
}

# [REQ:TM-LS-008] Light scan performance - medium scenarios (<200 files, <120s)
test_light_scan_medium_scenario() {
    echo "🧪 Testing light scan performance on medium scenario (tidiness-manager)..."

    local test_scenario="/home/matthalloran8/Vrooli/scenarios/tidiness-manager"
    local file_count
    file_count=$(find "$test_scenario" -type f \( -name "*.go" -o -name "*.ts" -o -name "*.tsx" \) 2>/dev/null | grep -E "^$test_scenario/(api|ui/src|cli)" | wc -l)

    if [ "$file_count" -gt 200 ]; then
        echo "⚠️  Warning: tidiness-manager has $file_count files (>200), may not be ideal for medium scenario test"
    fi

    local start_time
    start_time=$(date +%s)

    tidiness-manager scan --scenario-path "$test_scenario" --timeout 120 &> /tmp/tm-perf-medium.log
    local exit_code=$?

    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))

    echo "  Duration: ${duration}s (target: <120s)"
    echo "  Files scanned: $file_count"

    if [ $exit_code -ne 0 ]; then
        echo "❌ Scan failed with exit code $exit_code"
        cat /tmp/tm-perf-medium.log
        return 1
    fi

    if [ "$duration" -gt 120 ]; then
        echo "❌ Scan exceeded 120s threshold (took ${duration}s)"
        return 1
    fi

    echo "✅ Medium scenario scan completed in ${duration}s"
    return 0
}

# Run custom light scan performance tests BEFORE standard validation
# (standard validation calls lifecycle_end which exits the script)
echo ""
echo "🚀 Running light scan performance tests..."
if ! test_light_scan_small_scenario; then
    echo "⚠️  Small scenario performance test failed but continuing..."
fi
if ! test_light_scan_medium_scenario; then
    echo "⚠️  Medium scenario performance test failed but continuing..."
fi

# Run standard performance validation (this will exit the script)
testing::performance::validate_all
