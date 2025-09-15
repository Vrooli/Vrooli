#!/bin/bash
# Generic Go unit test runner for scenarios
# Can be sourced and used by any scenario's test suite
set -euo pipefail

# Run Go unit tests for a scenario
# Usage: testing::unit::run_go_tests [options]
# Options:
#   --dir PATH              Directory containing Go code (default: api)
#   --timeout SEC           Test timeout in seconds (default: 30)
#   --coverage              Generate coverage report (default: true)
#   --verbose               Verbose test output (default: false)
#   --coverage-warn PERCENT Coverage warning threshold (default: 80)
#   --coverage-error PERCENT Coverage error threshold (default: 50)
testing::unit::run_go_tests() {
    local api_dir="api"
    local timeout="30s"
    local coverage=true
    local verbose=false
    local coverage_warn_threshold=80
    local coverage_error_threshold=50
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                api_dir="$2"
                shift 2
                ;;
            --timeout)
                timeout="${2}s"
                shift 2
                ;;
            --no-coverage)
                coverage=false
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            --coverage-warn)
                coverage_warn_threshold="$2"
                shift 2
                ;;
            --coverage-error)
                coverage_error_threshold="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    echo "🐹 Running Go unit tests..."
    
    # Check if Go is available
    if ! command -v go >/dev/null 2>&1; then
        echo "❌ Go is not installed"
        return 1
    fi
    
    # Check if we have Go code
    if [ ! -d "$api_dir" ]; then
        echo "ℹ️  No $api_dir directory found, skipping Go tests"
        return 0
    fi
    
    if [ ! -f "$api_dir/go.mod" ]; then
        echo "ℹ️  No go.mod found in $api_dir, skipping Go tests"
        return 0
    fi
    
    # Save current directory and change to API directory
    local original_dir=$(pwd)
    cd "$api_dir"
    
    # Check if there are any test files
    local test_files=$(find . -name "*_test.go" -type f | wc -l)
    if [ "$test_files" -eq 0 ]; then
        echo "ℹ️  No Go test files (*_test.go) found"
        cd "$original_dir"
        return 0
    fi
    
    echo "📦 Downloading Go module dependencies..."
    if ! go mod download; then
        echo "❌ Failed to download Go dependencies"
        cd "$original_dir"
        return 1
    fi
    
    # Build test command
    local test_cmd="go test"
    if [ "$verbose" = true ]; then
        test_cmd="$test_cmd -v"
    fi
    test_cmd="$test_cmd ./... -timeout $timeout"
    if [ "$coverage" = true ]; then
        test_cmd="$test_cmd -cover -coverprofile=coverage.out"
    fi
    
    echo "🧪 Running Go tests..."
    
    # Run tests
    if eval "$test_cmd"; then
        echo "✅ Go unit tests completed successfully"
        
        # Display coverage summary if coverage file exists
        if [ "$coverage" = true ] && [ -f "coverage.out" ]; then
            echo ""
            echo "📊 Go Test Coverage Summary:"
            local coverage_line=$(go tool cover -func=coverage.out | tail -1)
            echo "$coverage_line"
            
            # Extract coverage percentage
            local coverage_percent=$(echo "$coverage_line" | grep -o '[0-9]*\.[0-9]*%' | sed 's/%//' | head -1)
            if [ -n "$coverage_percent" ]; then
                local coverage_num=$(echo "$coverage_percent" | cut -d. -f1)
                
                # Check coverage thresholds
                echo ""
                if [ "$coverage_num" -lt "$coverage_error_threshold" ]; then
                    echo "❌ ERROR: Go test coverage ($coverage_percent%) is below error threshold ($coverage_error_threshold%)"
                    echo "   This indicates insufficient test coverage. Please add more comprehensive tests."
                    cd "$original_dir"
                    return 1
                elif [ "$coverage_num" -lt "$coverage_warn_threshold" ]; then
                    echo "⚠️  WARNING: Go test coverage ($coverage_percent%) is below warning threshold ($coverage_warn_threshold%)"
                    echo "   Consider adding more tests to improve code coverage."
                else
                    echo "✅ Go test coverage ($coverage_percent%) meets quality thresholds"
                fi
            else
                echo "⚠️  WARNING: Could not parse coverage percentage from: $coverage_line"
            fi
            
            # Generate HTML coverage report for manual inspection
            go tool cover -html=coverage.out -o coverage.html 2>/dev/null && \
                echo "ℹ️  HTML coverage report generated: $api_dir/coverage.html"
        fi
        
        cd "$original_dir"
        return 0
    else
        echo "❌ Go unit tests failed"
        cd "$original_dir"
        return 1
    fi
}

# Export function for use by sourcing scripts
export -f testing::unit::run_go_tests