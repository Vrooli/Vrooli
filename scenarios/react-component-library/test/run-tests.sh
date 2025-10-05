#!/bin/bash
# Main test runner for react-component-library scenario

set -e

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCENARIO_DIR"

echo "🧪 React Component Library Test Suite"
echo "======================================"
echo ""

# Parse arguments
RUN_UNIT=true
RUN_INTEGRATION=false
RUN_PERFORMANCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --all)
            RUN_UNIT=true
            RUN_INTEGRATION=true
            RUN_PERFORMANCE=true
            shift
            ;;
        --unit)
            RUN_UNIT=true
            shift
            ;;
        --integration)
            RUN_INTEGRATION=true
            shift
            ;;
        --performance)
            RUN_PERFORMANCE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--all|--unit|--integration|--performance]"
            exit 1
            ;;
    esac
done

# Run unit tests
if [ "$RUN_UNIT" = true ]; then
    echo "📦 Running Unit Tests..."
    if [ -f "test/phases/test-unit.sh" ]; then
        bash test/phases/test-unit.sh
    else
        echo "⚠️  Unit test script not found"
    fi
    echo ""
fi

# Run integration tests
if [ "$RUN_INTEGRATION" = true ]; then
    echo "🔗 Running Integration Tests..."
    if [ -f "test/phases/test-integration.sh" ]; then
        bash test/phases/test-integration.sh
    else
        echo "⚠️  Integration test script not found"
    fi
    echo ""
fi

# Run performance tests
if [ "$RUN_PERFORMANCE" = true ]; then
    echo "⚡ Running Performance Tests..."
    if [ -f "test/phases/test-performance.sh" ]; then
        bash test/phases/test-performance.sh
    else
        echo "⚠️  Performance test script not found"
    fi
    echo ""
fi

echo ""
echo "✅ Test suite completed!"
