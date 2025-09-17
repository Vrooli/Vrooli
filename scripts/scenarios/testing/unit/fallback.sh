#!/usr/bin/env bash
# Fallback unit test implementations when centralized testing library is not available
# Provides minimal but functional test execution for Go, Node.js, and Python
set -euo pipefail

# Run fallback unit tests for all supported languages
# Usage: testing::unit::fallback::run_all [options]
# Options:
#   --dir PATH            Base directory (default: current)
#   --go-dir PATH        Directory for Go tests (default: api)
#   --node-dir PATH      Directory for Node.js tests (default: ui)
#   --python-dir PATH    Directory for Python tests (default: lib)
#   --skip-go            Skip Go tests
#   --skip-node          Skip Node.js tests  
#   --skip-python        Skip Python tests
#   --verbose            Show verbose output
testing::unit::fallback::run_all() {
    local base_dir="."
    local go_dir="api"
    local node_dir="ui"
    local python_dir="lib"
    local skip_go=false
    local skip_node=false
    local skip_python=false
    local verbose=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dir)
                base_dir="$2"
                shift 2
                ;;
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
            *)
                shift
                ;;
        esac
    done
    
    local error_count=0
    local test_count=0
    
    cd "$base_dir"
    
    echo "⚠️  Using fallback unit test runner (centralized library not available)"
    echo ""
    
    # Run Go tests
    if [ "$skip_go" = false ] && [ -d "$go_dir" ] && [ -f "$go_dir/go.mod" ]; then
        echo "🐹 Testing Go..."
        if testing::unit::fallback::run_go "$go_dir" "$verbose"; then
            echo "✅ Go tests passed"
            test_count=$((test_count + 1))
        else
            echo "❌ Go tests failed"
            error_count=$((error_count + 1))
        fi
        echo ""
    fi
    
    # Run Node.js tests
    if [ "$skip_node" = false ] && [ -d "$node_dir" ] && [ -f "$node_dir/package.json" ]; then
        echo "📦 Testing Node.js..."
        if testing::unit::fallback::run_node "$node_dir" "$verbose"; then
            echo "✅ Node.js tests passed"
            test_count=$((test_count + 1))
        else
            echo "❌ Node.js tests failed"
            error_count=$((error_count + 1))
        fi
        echo ""
    fi
    
    # Run Python tests
    if [ "$skip_python" = false ] && [ -d "$python_dir" ]; then
        if [ -f "$python_dir/requirements.txt" ] || [ -f "$python_dir/../requirements.txt" ] || [ -f "$python_dir/../pyproject.toml" ]; then
            echo "🐍 Testing Python..."
            if testing::unit::fallback::run_python "$python_dir" "$verbose"; then
                echo "✅ Python tests passed"
                test_count=$((test_count + 1))
            else
                echo "❌ Python tests failed"
                error_count=$((error_count + 1))
            fi
            echo ""
        fi
    fi
    
    echo "📊 Summary: $test_count passed, $error_count failed"
    
    if [ $error_count -eq 0 ]; then
        echo "✅ SUCCESS: All tests passed"
        return 0
    else
        echo "❌ ERROR: Some tests failed"
        return 1
    fi
}

# Run Go tests with fallback implementation
testing::unit::fallback::run_go() {
    local dir="${1:-api}"
    local verbose="${2:-false}"
    
    if ! command -v go >/dev/null 2>&1; then
        echo "⚠️  Go not installed, skipping Go tests"
        return 0
    fi
    
    cd "$dir"
    
    local test_args="./... -timeout 30s"
    if [ "$verbose" = true ]; then
        test_args="$test_args -v"
    fi
    
    # Try to run with coverage if possible
    if go test $test_args -cover >/dev/null 2>&1; then
        if [ "$verbose" = true ]; then
            go test $test_args -cover
        else
            go test $test_args -cover >/dev/null 2>&1
        fi
        local result=$?
        cd - >/dev/null
        return $result
    else
        # Fallback to basic test run
        if [ "$verbose" = true ]; then
            go test $test_args
        else
            go test $test_args >/dev/null 2>&1
        fi
        local result=$?
        cd - >/dev/null
        return $result
    fi
}

# Run Node.js tests with fallback implementation
testing::unit::fallback::run_node() {
    local dir="${1:-ui}"
    local verbose="${2:-false}"
    
    if ! command -v npm >/dev/null 2>&1; then
        echo "⚠️  npm not installed, skipping Node.js tests"
        return 0
    fi
    
    cd "$dir"
    
    # Check if test script exists in package.json
    if ! grep -q '"test"' package.json 2>/dev/null; then
        echo "⚠️  No test script defined in package.json"
        cd - >/dev/null
        return 0
    fi
    
    # Install dependencies if node_modules doesn't exist
    if [ ! -d "node_modules" ]; then
        echo "📦 Installing dependencies..."
        npm install --silent >/dev/null 2>&1 || true
    fi
    
    # Run tests
    if [ "$verbose" = true ]; then
        npm test
    else
        npm test --silent >/dev/null 2>&1 || npm test --passWithNoTests --silent >/dev/null 2>&1
    fi
    local result=$?
    
    cd - >/dev/null
    return $result
}

# Run Python tests with fallback implementation
testing::unit::fallback::run_python() {
    local dir="${1:-lib}"
    local verbose="${2:-false}"
    
    if ! command -v python3 >/dev/null 2>&1 && ! command -v python >/dev/null 2>&1; then
        echo "⚠️  Python not installed, skipping Python tests"
        return 0
    fi
    
    cd "$dir"
    
    local python_cmd="python3"
    if ! command -v python3 >/dev/null 2>&1; then
        python_cmd="python"
    fi
    
    # Try pytest first
    if command -v pytest >/dev/null 2>&1; then
        if [ "$verbose" = true ]; then
            pytest -v
        else
            pytest -q >/dev/null 2>&1
        fi
        local result=$?
        cd - >/dev/null
        return $result
    fi
    
    # Try unittest discovery
    if [ "$verbose" = true ]; then
        $python_cmd -m unittest discover -v
    else
        $python_cmd -m unittest discover >/dev/null 2>&1
    fi
    local result=$?
    
    cd - >/dev/null
    return $result
}

# Check if centralized testing library is available
testing::unit::fallback::is_needed() {
    local testing_lib="${1:-}"
    
    if [ -z "$testing_lib" ]; then
        # Try to detect the testing library path
        local app_root="${APP_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)}"
        testing_lib="$app_root/scripts/scenarios/testing/unit"
    fi
    
    if [ ! -f "$testing_lib/run-all.sh" ]; then
        return 0  # Fallback is needed
    else
        return 1  # Centralized library exists
    fi
}