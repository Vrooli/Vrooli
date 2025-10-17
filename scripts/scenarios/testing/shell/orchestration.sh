#!/usr/bin/env bash
# Test orchestration utilities - comprehensive test suite execution
set -euo pipefail

# Source all shell modules
SHELL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHELL_DIR/core.sh"
source "$SHELL_DIR/connectivity.sh"
source "$SHELL_DIR/resources.sh"
source "$SHELL_DIR/cli.sh"

# Get testing library directory
TESTING_LIB_DIR="$(cd "$SHELL_DIR/.." && pwd)"

# Run unit tests for detected languages
testing::orchestration::run_unit_tests() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local coverage_warn_threshold="${2:-80}"
    local coverage_error_threshold="${3:-70}"
    
    echo "🧪 Running unit tests for scenario: $scenario_name"
    
    # Source the universal unit test runner
    source "$TESTING_LIB_DIR/unit/run-all.sh"
    
    # Detect languages and run appropriate tests
    local languages
    mapfile -t languages < <(testing::core::detect_languages)
    
    if [ ${#languages[@]} -eq 0 ]; then
        echo "ℹ️  No supported languages detected for unit testing"
        return 0
    fi
    
    echo "📋 Detected languages: ${languages[*]}"
    
    # Build arguments for the universal runner
    local runner_args=(
        "--coverage-warn" "$coverage_warn_threshold"
        "--coverage-error" "$coverage_error_threshold"
    )
    
    # Skip languages not present
    local all_langs=("go" "node" "python")
    for lang in "${all_langs[@]}"; do
        if ! printf '%s\n' "${languages[@]}" | grep -q "^$lang$"; then
            runner_args+=("--skip-$lang")
        fi
    done
    
    # Run the universal test runner
    testing::unit::run_all_tests "${runner_args[@]}"
}

# Run a comprehensive test suite for a scenario
testing::orchestration::run_comprehensive_tests() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    local coverage_warn="${2:-80}"
    local coverage_error="${3:-70}"
    
    echo "🚀 Running comprehensive tests for scenario: $scenario_name"
    echo ""
    
    local total_test_suites=0
    local failed_test_suites=0
    
    # Unit tests
    echo "=== Unit Tests ==="
    ((total_test_suites++))
    if testing::orchestration::run_unit_tests "$scenario_name" "$coverage_warn" "$coverage_error"; then
        echo "✅ Unit tests passed"
    else
        echo "❌ Unit tests failed"
        ((failed_test_suites++))
    fi
    echo ""
    
    # API connectivity
    echo "=== API Integration ==="
    ((total_test_suites++))
    if testing::connectivity::test_api "$scenario_name"; then
        echo "✅ API integration tests passed"
    else
        echo "❌ API integration tests failed"
        ((failed_test_suites++))
    fi
    echo ""
    
    # UI connectivity
    echo "=== UI Integration ==="
    ((total_test_suites++))
    if testing::connectivity::test_ui "$scenario_name"; then
        echo "✅ UI integration tests passed"
    else
        echo "❌ UI integration tests failed"
        ((failed_test_suites++))
    fi
    echo ""
    
    # Resource integrations
    echo "=== Resource Integration ==="
    ((total_test_suites++))
    if testing::resources::test_all "$scenario_name"; then
        echo "✅ Resource integration tests passed"
    else
        echo "❌ Resource integration tests failed"
        ((failed_test_suites++))
    fi
    echo ""
    
    # CLI integration
    echo "=== CLI Integration ==="
    ((total_test_suites++))
    if testing::cli::test_integration "$scenario_name"; then
        echo "✅ CLI integration tests passed"
    else
        echo "❌ CLI integration tests failed"
        ((failed_test_suites++))
    fi
    echo ""
    
    # Summary
    echo "📊 Comprehensive Test Summary:"
    echo "   Test suites run: $total_test_suites"
    echo "   Test suites passed: $((total_test_suites - failed_test_suites))"
    echo "   Test suites failed: $failed_test_suites"
    
    if [ $failed_test_suites -eq 0 ]; then
        echo ""
        echo "🎉 All comprehensive tests passed!"
        return 0
    else
        echo ""
        echo "💥 Some comprehensive tests failed"
        return 1
    fi
}

# Run integration tests only
testing::orchestration::run_integration_tests() {
    local scenario_name="${1:-$(testing::core::detect_scenario)}"
    
    echo "🔗 Running integration tests for scenario: $scenario_name"
    echo ""
    
    local total_tests=0
    local failed_tests=0
    
    # Connectivity tests
    ((total_tests++))
    if testing::connectivity::test_all "$scenario_name"; then
        echo "✅ Connectivity tests passed"
    else
        echo "❌ Connectivity tests failed"
        ((failed_tests++))
    fi
    
    # Resource tests
    ((total_tests++))
    if testing::resources::test_all "$scenario_name"; then
        echo "✅ Resource tests passed"
    else
        echo "❌ Resource tests failed"
        ((failed_tests++))
    fi
    
    # CLI tests
    ((total_tests++))
    if testing::cli::test_integration "$scenario_name"; then
        echo "✅ CLI tests passed"
    else
        echo "❌ CLI tests failed"
        ((failed_tests++))
    fi
    
    echo ""
    echo "📊 Integration Test Summary:"
    echo "   Tests run: $total_tests"
    echo "   Tests passed: $((total_tests - failed_tests))"
    echo "   Tests failed: $failed_tests"
    
    if [ $failed_tests -eq 0 ]; then
        return 0
    else
        return 1
    fi
}

# Export functions
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    export -f testing::orchestration::run_unit_tests
    export -f testing::orchestration::run_comprehensive_tests
    export -f testing::orchestration::run_integration_tests
fi