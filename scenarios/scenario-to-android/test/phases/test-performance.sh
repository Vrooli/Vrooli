#!/bin/bash
# Performance test phase for scenario-to-android
# Tests build performance, template processing speed, and resource usage

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 5-minute target
testing::phase::init --target-time "300s"

# Change to scenario directory
cd "$TESTING_PHASE_SCENARIO_DIR"

# Test output directory
TEST_OUTPUT_DIR="/tmp/scenario-to-android-perf-test-$$"
mkdir -p "$TEST_OUTPUT_DIR"

# Cleanup function
cleanup_perf_tests() {
    log::info "Cleaning up performance test artifacts"
    rm -rf "$TEST_OUTPUT_DIR"
}
trap cleanup_perf_tests EXIT

# ============================================================================
# Helper Functions
# ============================================================================

# Measure command execution time
measure_time() {
    local description="$1"
    shift
    local start_time end_time duration

    start_time=$(date +%s.%N)
    "$@" &> /dev/null || true
    end_time=$(date +%s.%N)

    # Calculate duration in milliseconds
    duration=$(echo "($end_time - $start_time) * 1000" | bc)
    printf "%.0f" "$duration"
}

# Report timing
report_timing() {
    local description="$1"
    local duration_ms="$2"
    local threshold_ms="$3"

    log::info "$description: ${duration_ms}ms"

    if [ -n "$threshold_ms" ]; then
        if (( $(echo "$duration_ms < $threshold_ms" | bc -l) )); then
            log::success "✓ Performance target met (< ${threshold_ms}ms)"
        else
            testing::phase::add_warning "⚠ Performance target missed (threshold: ${threshold_ms}ms)"
        fi
    fi
}

# ============================================================================
# CLI Command Response Time Tests
# ============================================================================

log::info "Testing CLI command response times"

CLI_PATH="$TESTING_PHASE_SCENARIO_DIR/cli/scenario-to-android"

if [ -x "$CLI_PATH" ]; then
    # Test help command speed
    duration=$(measure_time "Help command" "$CLI_PATH" help)
    report_timing "Help command" "$duration" "100"

    # Test version command speed
    duration=$(measure_time "Version command" "$CLI_PATH" version)
    report_timing "Version command" "$duration" "500"

    # Test status command speed
    duration=$(measure_time "Status command" "$CLI_PATH" status)
    report_timing "Status command" "$duration" "2000"

    # Test templates command speed
    duration=$(measure_time "Templates command" "$CLI_PATH" templates)
    report_timing "Templates command" "$duration" "100"
else
    testing::phase::add_error "CLI binary not available for performance testing"
fi

# ============================================================================
# Template Processing Performance
# ============================================================================

log::info "Testing template processing performance"

TEMPLATES_DIR="$TESTING_PHASE_SCENARIO_DIR/initialization/templates/android"

if [ -d "$TEMPLATES_DIR" ]; then
    # Count template files
    template_count=$(find "$TEMPLATES_DIR" -type f | wc -l)
    log::info "Template file count: $template_count"

    # Test template copying speed
    start_time=$(date +%s.%N)
    cp -r "$TEMPLATES_DIR" "$TEST_OUTPUT_DIR/template-copy" &> /dev/null
    end_time=$(date +%s.%N)
    duration=$(echo "($end_time - $start_time) * 1000" | bc)
    report_timing "Template copy operation" "$(printf "%.0f" "$duration")" "1000"

    # Test template variable processing speed
    if [ -f "$TEMPLATES_DIR/app/build.gradle" ]; then
        start_time=$(date +%s.%N)
        sed -e "s/{{SCENARIO_NAME}}/test-scenario/g" \
            -e "s/{{APP_NAME}}/Test App/g" \
            -e "s/{{VERSION_NAME}}/1.0.0/g" \
            "$TEMPLATES_DIR/app/build.gradle" > "$TEST_OUTPUT_DIR/test.gradle"
        end_time=$(date +%s.%N)
        duration=$(echo "($end_time - $start_time) * 1000" | bc)
        report_timing "Template variable substitution" "$(printf "%.0f" "$duration")" "50"
    fi
else
    testing::phase::add_error "Templates directory not found"
fi

# ============================================================================
# Project Creation Performance
# ============================================================================

log::info "Testing project creation performance"

# Create test scenario
TEST_SCENARIO_DIR="$TEST_OUTPUT_DIR/test-scenario"
mkdir -p "$TEST_SCENARIO_DIR/ui"
cat > "$TEST_SCENARIO_DIR/ui/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Test</h1></body>
</html>
EOF

# Create multiple test files to simulate a real scenario
for i in {1..10}; do
    echo "/* Test file $i */" > "$TEST_SCENARIO_DIR/ui/test-$i.js"
done

log::info "Created test scenario with UI files"

# ============================================================================
# Conversion Script Performance
# ============================================================================

log::info "Testing conversion script performance (without actual build)"

CONVERT_SCRIPT="$TESTING_PHASE_SCENARIO_DIR/cli/convert.sh"

if [ -x "$CONVERT_SCRIPT" ]; then
    # Set up test scenario
    SCENARIOS_DIR="$TEST_OUTPUT_DIR/scenarios"
    mkdir -p "$SCENARIOS_DIR"
    cp -r "$TEST_SCENARIO_DIR" "$SCENARIOS_DIR/perf-test"

    ANDROID_OUTPUT="$TEST_OUTPUT_DIR/android-build"

    # Measure conversion time (template processing only, no Gradle build)
    start_time=$(date +%s.%N)
    "$CONVERT_SCRIPT" \
        --scenario perf-test \
        --output "$ANDROID_OUTPUT" \
        --app-name "Perf Test" \
        --version "1.0.0" &> "$TEST_OUTPUT_DIR/convert.log" || true
    end_time=$(date +%s.%N)
    duration=$(echo "($end_time - $start_time) * 1000" | bc)

    # Target: < 5 seconds for project setup (excluding Gradle build)
    report_timing "Project creation (no build)" "$(printf "%.0f" "$duration")" "5000"

    # Analyze conversion log for timing breakdown
    if [ -f "$TEST_OUTPUT_DIR/convert.log" ]; then
        log_size=$(wc -c < "$TEST_OUTPUT_DIR/convert.log")
        log::info "Conversion log size: ${log_size} bytes"
    fi
else
    testing::phase::add_error "Conversion script not available"
fi

# ============================================================================
# File I/O Performance
# ============================================================================

log::info "Testing file I/O performance"

if [ -d "$ANDROID_OUTPUT/perf-test-android" ]; then
    PROJECT_DIR="$ANDROID_OUTPUT/perf-test-android"

    # Count generated files
    file_count=$(find "$PROJECT_DIR" -type f | wc -l)
    log::info "Generated files: $file_count"

    # Measure directory tree traversal
    start_time=$(date +%s.%N)
    find "$PROJECT_DIR" -type f > /dev/null
    end_time=$(date +%s.%N)
    duration=$(echo "($end_time - $start_time) * 1000" | bc)
    report_timing "Directory tree traversal" "$(printf "%.0f" "$duration")" "500"

    # Measure project size
    project_size=$(du -sh "$PROJECT_DIR" | cut -f1)
    log::info "Project size: $project_size"
fi

# ============================================================================
# Memory Usage Analysis
# ============================================================================

log::info "Analyzing memory usage patterns"

# Test memory usage during template processing
# This is a simple test - real measurement would need process monitoring

if command -v bc &> /dev/null; then
    # Estimate memory usage based on file sizes
    if [ -d "$TEMPLATES_DIR" ]; then
        template_size=$(du -sb "$TEMPLATES_DIR" | cut -f1)
        estimated_memory=$(echo "$template_size * 2" | bc)  # Rough estimate: 2x file size
        log::info "Estimated template memory usage: ~$(echo "scale=2; $estimated_memory / 1024 / 1024" | bc)MB"
    fi
else
    log::warning "bc not available for memory calculations"
fi

# ============================================================================
# Concurrent Operation Tests
# ============================================================================

log::info "Testing concurrent operations"

# Test multiple CLI commands running simultaneously
start_time=$(date +%s.%N)

# Run multiple help commands in parallel
for i in {1..5}; do
    "$CLI_PATH" help &> /dev/null &
done
wait

end_time=$(date +%s.%N)
duration=$(echo "($end_time - $start_time) * 1000" | bc)
report_timing "5 concurrent help commands" "$(printf "%.0f" "$duration")" "1000"

# ============================================================================
# Resource Cleanup Performance
# ============================================================================

log::info "Testing cleanup performance"

# Create large test directory
CLEANUP_TEST_DIR="$TEST_OUTPUT_DIR/cleanup-test"
mkdir -p "$CLEANUP_TEST_DIR"

# Create many files
for i in {1..100}; do
    echo "test" > "$CLEANUP_TEST_DIR/file-$i.txt"
done

# Measure cleanup time
start_time=$(date +%s.%N)
rm -rf "$CLEANUP_TEST_DIR"
end_time=$(date +%s.%N)
duration=$(echo "($end_time - $start_time) * 1000" | bc)
report_timing "Cleanup of 100 files" "$(printf "%.0f" "$duration")" "500"

# ============================================================================
# Performance Summary
# ============================================================================

log::info "Generating performance summary"

cat > "$TEST_OUTPUT_DIR/performance-report.txt" << EOF
Scenario to Android - Performance Test Report
Generated: $(date)

Test Environment:
- OS: $(uname -s)
- Kernel: $(uname -r)
- Shell: $SHELL
- Disk: $(df -h . | tail -1 | awk '{print $4}') available

Performance Targets:
- CLI help command: < 100ms
- CLI version command: < 500ms
- CLI status command: < 2000ms
- Template processing: < 1000ms
- Variable substitution: < 50ms
- Project creation: < 5000ms

Notes:
- Build performance not measured (requires Android SDK)
- Actual build time target: < 5 minutes for simple scenarios
- Memory usage: < 2GB for build process
- Disk usage: < 5GB for project + artifacts

EOF

log::success "Performance report generated: $TEST_OUTPUT_DIR/performance-report.txt"

# End with summary
testing::phase::end_with_summary "Performance tests completed"
