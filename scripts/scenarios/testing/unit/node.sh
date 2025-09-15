#!/bin/bash
# Generic Node.js unit test runner for scenarios
# Can be sourced and used by any scenario's test suite
set -euo pipefail

# Run Node.js unit tests for a scenario
# Usage: testing::unit::run_node_tests [options]
# Options:
#   --dir PATH              Directory containing Node.js code (default: ui)
#   --timeout MS            Test timeout in milliseconds (default: 30000)
#   --test-cmd CMD          Custom test command (default: reads from package.json)
#   --verbose               Verbose test output (default: false)
#   --coverage-warn PERCENT Coverage warning threshold (default: 80)
#   --coverage-error PERCENT Coverage error threshold (default: 50)
testing::unit::run_node_tests() {
    local node_dir="ui"
    local timeout="30000"
    local test_cmd=""
    local verbose=false
    local coverage_warn_threshold=80
    local coverage_error_threshold=50
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                node_dir="$2"
                shift 2
                ;;
            --timeout)
                timeout="$2"
                shift 2
                ;;
            --test-cmd)
                test_cmd="$2"
                shift 2
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
    
    echo "📦 Running Node.js unit tests..."
    
    # Check if Node.js is available
    if ! command -v node >/dev/null 2>&1; then
        echo "❌ Node.js is not installed"
        return 1
    fi
    
    # Check if we have Node.js code
    if [ ! -d "$node_dir" ]; then
        echo "ℹ️  No $node_dir directory found, skipping Node.js tests"
        return 0
    fi
    
    if [ ! -f "$node_dir/package.json" ]; then
        echo "ℹ️  No package.json found in $node_dir, skipping Node.js tests"
        return 0
    fi
    
    # Save current directory and change to Node directory
    local original_dir=$(pwd)
    cd "$node_dir"
    
    # Check if test script is defined in package.json
    if [ -z "$test_cmd" ]; then
        test_cmd=$(node -e "const pkg=require('./package.json'); console.log(pkg.scripts?.test || '')" 2>/dev/null || echo "")
    fi
    
    if [ -z "$test_cmd" ] || [ "$test_cmd" == "echo \"Error: no test specified\" && exit 1" ]; then
        echo "ℹ️  No test script defined in package.json"
        
        # Check if there are test files anyway
        local test_files=$(find . \( -name "*.test.js" -o -name "*.spec.js" -o -name "*.test.ts" -o -name "*.spec.ts" \) -type f | wc -l)
        if [ "$test_files" -gt 0 ]; then
            echo "   Found $test_files test file(s) but no test script configured"
            echo "   💡 Add a test script to package.json"
        fi
        
        cd "$original_dir"
        return 0
    fi
    
    # Install dependencies if node_modules doesn't exist
    if [ ! -d "node_modules" ]; then
        echo "📦 Installing Node.js dependencies..."
        if command -v npm >/dev/null 2>&1; then
            if ! npm install --silent; then
                echo "❌ Failed to install Node.js dependencies"
                cd "$original_dir"
                return 1
            fi
        else
            echo "❌ npm is not installed"
            cd "$original_dir"
            return 1
        fi
    fi
    
    echo "🧪 Running Node.js tests..."
    
    # Set test timeout environment variable if supported
    export JEST_TIMEOUT="$timeout"
    export MOCHA_TIMEOUT="$timeout"
    export VITEST_TIMEOUT="$timeout"
    
    # Run tests with coverage
    local test_output
    local test_success=false
    
    if [ "$verbose" = true ]; then
        # Run tests with coverage enabled and capture output
        if test_output=$(npm test -- --coverage 2>&1); then
            test_success=true
            echo "$test_output"
        else
            echo "$test_output"
            echo "❌ Node.js unit tests failed"
            cd "$original_dir"
            return 1
        fi
    else
        # Run tests with coverage enabled and capture output
        if test_output=$(npm test -- --coverage --silent 2>&1); then
            test_success=true
            echo "$test_output"
        else
            echo "$test_output"
            echo "❌ Node.js unit tests failed"
            cd "$original_dir"
            return 1
        fi
    fi
    
    if [ "$test_success" = true ]; then
        echo "✅ Node.js unit tests completed successfully"
        
        # Parse coverage results - try multiple patterns
        local coverage_line=$(echo "$test_output" | grep -E "(All files|Statements|Lines|Functions|Branches).*[0-9]+(\.[0-9]+)?%" | head -1)
        local coverage_percent=""
        
        if [ -n "$coverage_line" ]; then
            # Extract first percentage from coverage line
            coverage_percent=$(echo "$coverage_line" | grep -o '[0-9]\+\(\.[0-9]\+\)\?%' | head -1 | sed 's/%//')
        fi
        
        # If no coverage line found, try looking for coverage table
        if [ -z "$coverage_percent" ]; then
            coverage_percent=$(echo "$test_output" | grep -o '[0-9]\+\(\.[0-9]\+\)\?%' | head -1 | sed 's/%//')
        fi
            
        if [ -n "$coverage_percent" ]; then
            local coverage_num=$(echo "$coverage_percent" | cut -d. -f1)
            
            # Check coverage thresholds
            echo ""
            if [ "$coverage_num" -lt "$coverage_error_threshold" ]; then
                echo "❌ ERROR: Node.js test coverage ($coverage_percent%) is below error threshold ($coverage_error_threshold%)"
                echo "   This indicates insufficient test coverage. Please add more comprehensive tests."
                cd "$original_dir"
                return 1
            elif [ "$coverage_num" -lt "$coverage_warn_threshold" ]; then
                echo "⚠️  WARNING: Node.js test coverage ($coverage_percent%) is below warning threshold ($coverage_warn_threshold%)"
                echo "   Consider adding more tests to improve code coverage."
            else
                echo "✅ Node.js test coverage ($coverage_percent%) meets quality thresholds"
            fi
        else
            echo "ℹ️  No coverage information found in test output. Coverage may not be configured."
        fi
        
        cd "$original_dir"
        return 0
    fi
}

# Export function for use by sourcing scripts
export -f testing::unit::run_node_tests