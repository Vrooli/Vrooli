#!/bin/bash
# Apache Kafka Resource - Main Test Runner
# v2.0 Contract Compliant

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source test library
source "$SCRIPT_DIR/lib/test.sh"

# Main test runner
main() {
    local test_type="${1:-all}"
    
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
            echo "Usage: $0 [smoke|integration|unit|all]"
            exit 1
            ;;
    esac
}

# Run tests
main "$@"