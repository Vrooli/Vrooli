#!/bin/bash
# Generic Python unit test runner for scenarios
# Can be sourced and used by any scenario's test suite
set -euo pipefail

# Run Python unit tests for a scenario
# Usage: testing::unit::run_python_tests [options]
# Options:
#   --dir PATH       Directory containing Python code (default: .)
#   --timeout SEC    Test timeout in seconds (default: 30)
#   --framework FW   Test framework: pytest, unittest, nose (default: auto-detect)
#   --verbose        Verbose test output (default: false)
testing::unit::run_python_tests() {
    local python_dir="."
    local timeout="30"
    local framework=""
    local verbose=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                python_dir="$2"
                shift 2
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            --framework)
                framework="$2"
                shift 2
                ;;
            --verbose)
                verbose=true
                shift
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    echo "🐍 Running Python unit tests..."
    
    # Check if Python is available
    if ! command -v python3 >/dev/null 2>&1 && ! command -v python >/dev/null 2>&1; then
        echo "❌ Python is not installed"
        return 1
    fi
    
    # Determine Python command
    local python_cmd="python"
    if command -v python3 >/dev/null 2>&1; then
        python_cmd="python3"
    fi
    
    # Check if we have Python code
    if [ "$python_dir" != "." ] && [ ! -d "$python_dir" ]; then
        echo "ℹ️  No $python_dir directory found, skipping Python tests"
        return 0
    fi
    
    # Check for Python project indicators
    local has_python_project=false
    if [ -f "$python_dir/requirements.txt" ] || [ -f "$python_dir/setup.py" ] || [ -f "$python_dir/pyproject.toml" ] || [ -f "$python_dir/Pipfile" ]; then
        has_python_project=true
    fi
    
    # Check for Python files
    local python_files=$(find "$python_dir" -name "*.py" -type f | wc -l)
    if [ "$python_files" -eq 0 ]; then
        echo "ℹ️  No Python files found, skipping Python tests"
        return 0
    fi
    
    # Save current directory and change to Python directory
    local original_dir=$(pwd)
    if [ "$python_dir" != "." ]; then
        cd "$python_dir"
    fi
    
    # Check for test files
    local test_files=$(find . \( -name "test_*.py" -o -name "*_test.py" -o -name "test*.py" \) -type f | wc -l)
    if [ "$test_files" -eq 0 ]; then
        echo "ℹ️  No Python test files found (test_*.py, *_test.py)"
        cd "$original_dir"
        return 0
    fi
    
    # Auto-detect test framework if not specified
    if [ -z "$framework" ]; then
        if [ -f "pytest.ini" ] || [ -f "setup.cfg" ] && grep -q "pytest" "setup.cfg" 2>/dev/null; then
            framework="pytest"
        elif $python_cmd -c "import pytest" 2>/dev/null; then
            framework="pytest"
        elif [ -d "tests" ] && [ -f "tests/__init__.py" ]; then
            framework="unittest"
        else
            framework="unittest"  # Default fallback
        fi
    fi
    
    echo "🧪 Running Python tests with $framework..."
    
    # Build test command based on framework
    local test_cmd=""
    case "$framework" in
        pytest)
            test_cmd="$python_cmd -m pytest"
            if [ "$verbose" = true ]; then
                test_cmd="$test_cmd -v"
            fi
            test_cmd="$test_cmd --timeout=$timeout --tb=short"
            # Add coverage if pytest-cov is available
            if $python_cmd -c "import pytest_cov" 2>/dev/null; then
                test_cmd="$test_cmd --cov=. --cov-report=term-missing"
            fi
            ;;
        unittest)
            test_cmd="$python_cmd -m unittest discover"
            if [ "$verbose" = true ]; then
                test_cmd="$test_cmd -v"
            fi
            test_cmd="timeout ${timeout}s $test_cmd"
            ;;
        nose|nose2)
            test_cmd="$python_cmd -m $framework"
            if [ "$verbose" = true ]; then
                test_cmd="$test_cmd -v"
            fi
            test_cmd="timeout ${timeout}s $test_cmd"
            ;;
        *)
            echo "❌ Unknown test framework: $framework"
            cd "$original_dir"
            return 1
            ;;
    esac
    
    # Run tests
    if eval "$test_cmd"; then
        echo "✅ Python unit tests completed successfully"
        cd "$original_dir"
        return 0
    else
        echo "❌ Python unit tests failed"
        cd "$original_dir"
        return 1
    fi
}

# Export function for use by sourcing scripts
export -f testing::unit::run_python_tests