#!/usr/bin/env bash
# OpenMRS Test Runner

set -euo pipefail

# Get the absolute path to this script's directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source common libraries
source "${RESOURCE_DIR}/../../scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Parse arguments
PHASE="${1:-all}"

# Run tests
log::info "Running OpenMRS tests - Phase: $PHASE"

case "$PHASE" in
    smoke)
        "${SCRIPT_DIR}/phases/test-smoke.sh"
        ;;
    integration)
        "${SCRIPT_DIR}/phases/test-integration.sh"
        ;;
    unit)
        "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    all)
        "${SCRIPT_DIR}/phases/test-smoke.sh" && \
        "${SCRIPT_DIR}/phases/test-integration.sh" && \
        "${SCRIPT_DIR}/phases/test-unit.sh"
        ;;
    *)
        log::error "Unknown test phase: $PHASE"
        echo "Available phases: smoke, integration, unit, all"
        exit 1
        ;;
esac