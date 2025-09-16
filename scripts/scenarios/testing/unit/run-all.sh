#!/bin/bash
# Universal unit test runner that runs all detected language tests
# Sources individual language runners and executes them
set -euo pipefail

# Get the directory where this script is located
UNIT_TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source all language-specific runners
source "$UNIT_TEST_DIR/go.sh"
source "$UNIT_TEST_DIR/node.sh"
source "$UNIT_TEST_DIR/python.sh"

# Run all unit tests for a scenario
# Usage: testing::unit::run_all_tests [options]
# Options:
#   --go-dir PATH               Directory for Go tests (default: api)
#   --node-dir PATH             Directory for Node.js tests (default: ui)
#   --python-dir PATH           Directory for Python tests (default: .)
#   --skip-go                   Skip Go tests
#   --skip-node                 Skip Node.js tests
#   --skip-python               Skip Python tests
#   --verbose                   Verbose output for all tests
#   --fail-fast                 Stop on first test failure
#   --coverage-warn PERCENT     Coverage warning threshold (default: 80)
#   --coverage-error PERCENT    Coverage error threshold (default: 50)
testing::unit::run_all_tests() {
    local go_dir="api"
    local node_dir="ui"
    local python_dir="."
    local skip_go=false
    local skip_node=false
    local skip_python=false
    local verbose=false
    local fail_fast=false
    local coverage_warn_threshold=80
    local coverage_error_threshold=50
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --go-dir)
                go_dir="$2"
                shift 2
                ;;
            --node-dir)
                node_dir="$2"
                shift 2
                ;;
            --python-dir)
                python_dir="$2"
                shift 2
                ;;
            --skip-go)
                skip_go=true
                shift
                ;;
            --skip-node)
                skip_node=true
                shift
                ;;
            --skip-python)
                skip_python=true
                shift
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            --fail-fast)
                fail_fast=true
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
    
    local error_count=0
    local test_count=0
    local skipped_count=0
    
    echo "🧪 Running all unit tests..."
    echo ""
    
    # Run Go tests
    if [ "$skip_go" = false ]; then
        local go_args="--dir $go_dir --coverage-warn $coverage_warn_threshold --coverage-error $coverage_error_threshold"
        if [ "$verbose" = true ]; then
            go_args="$go_args --verbose"
        fi
        
        if testing::unit::run_go_tests $go_args; then
            ((test_count++)) || true
        else
            ((error_count++)) || true
            if [ "$fail_fast" = true ]; then
                echo "❌ Stopping due to Go test failure (fail-fast enabled)"
                return 1
            fi
        fi
        echo ""
    else
        ((skipped_count++)) || true
    fi
    
    # Run Node.js tests
    if [ "$skip_node" = false ]; then
        local node_args="--dir $node_dir --coverage-warn $coverage_warn_threshold --coverage-error $coverage_error_threshold"
        if [ "$verbose" = true ]; then
            node_args="$node_args --verbose"
        fi
        
        if testing::unit::run_node_tests $node_args; then
            ((test_count++)) || true
        else
            ((error_count++)) || true
            if [ "$fail_fast" = true ]; then
                echo "❌ Stopping due to Node.js test failure (fail-fast enabled)"
                return 1
            fi
        fi
        echo ""
    else
        ((skipped_count++)) || true
    fi
    
    # Run Python tests
    if [ "$skip_python" = false ]; then
        local python_args="--dir $python_dir"
        if [ "$verbose" = true ]; then
            python_args="$python_args --verbose"
        fi
        
        # Note: Python test runner doesn't have coverage thresholds implemented yet
        if testing::unit::run_python_tests $python_args; then
            ((test_count++)) || true
        else
            ((error_count++)) || true
            if [ "$fail_fast" = true ]; then
                echo "❌ Stopping due to Python test failure (fail-fast enabled)"
                return 1
            fi
        fi
        echo ""
    else
        ((skipped_count++)) || true
    fi
    
    # Summary
    echo "📊 Unit Test Summary:"
    echo "   Tests passed: $test_count"
    echo "   Tests failed: $error_count"
    echo "   Tests skipped: $skipped_count"
    
    if [ $error_count -eq 0 ]; then
        echo ""
        echo "✅ All unit tests passed!"
        return 0
    else
        echo ""
        echo "❌ Some unit tests failed"
        return 1
    fi
}

# Export main function
export -f testing::unit::run_all_tests

# If script is executed directly (not sourced), run all tests
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    testing::unit::run_all_tests "$@"
fi
