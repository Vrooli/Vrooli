#!/bin/bash

# ElectionGuard Test Runner
# Main entry point for all test phases

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test phases
source "${RESOURCE_DIR}/lib/test.sh"

# Main test runner
main() {
    local phase="${1:-all}"
    
    echo "ElectionGuard Test Suite"
    echo "========================"
    echo "Running phase: $phase"
    echo ""
    
    # Run the requested test phase
    handle_test "$phase"
    
    exit $?
}

# Run main with all arguments
main "$@"