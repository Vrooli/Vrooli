#!/bin/bash
# Step-CA Test Runner

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test library
source "${RESOURCE_DIR}/lib/test.sh"
source "${RESOURCE_DIR}/config/defaults.sh"

# Main test runner
main() {
    local test_type="${1:-all}"
    
    echo "🧪 Step-CA Test Runner"
    echo "  Test Type: $test_type"
    echo ""
    
    case "$test_type" in
        smoke)
            test_smoke
            ;;
        integration)
            test_integration
            ;;
        unit)
            test_unit
            ;;
        all)
            test_all
            ;;
        *)
            echo "❌ Unknown test type: $test_type"
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
}

# Run tests
main "$@"