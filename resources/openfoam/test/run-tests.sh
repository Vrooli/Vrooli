#!/bin/bash

# OpenFOAM Test Runner
# Main test orchestration script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source test library
source "$RESOURCE_DIR/lib/test.sh"

# Run specified test phase or all tests
main() {
    local phase="${1:-all}"
    
    echo "OpenFOAM Test Runner"
    echo "===================="
    echo "Phase: $phase"
    echo ""
    
    case "$phase" in
        smoke)
            openfoam::test::smoke
            ;;
        integration)
            openfoam::test::integration
            ;;
        unit)
            openfoam::test::unit
            ;;
        all)
            openfoam::test::all
            ;;
        *)
            echo "Error: Unknown test phase: $phase"
            echo "Valid phases: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi