#!/usr/bin/env bash
################################################################################
# AudioCraft Test Runner
# Main test orchestration script
################################################################################
set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Load configuration
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"
source "${RESOURCE_DIR}/lib/test.sh"

# Test phase to run
PHASE="${1:-all}"

################################################################################
# Run Test Phase
################################################################################
run_phase() {
    local phase="$1"
    
    case "$phase" in
        all)
            test::run_all
            ;;
        smoke)
            echo "🔥 Running smoke tests..."
            if [[ -f "${SCRIPT_DIR}/phases/test-smoke.sh" ]]; then
                bash "${SCRIPT_DIR}/phases/test-smoke.sh"
            else
                test::smoke::health_check
                test::smoke::models_endpoint
            fi
            ;;
        integration)
            echo "🔗 Running integration tests..."
            if [[ -f "${SCRIPT_DIR}/phases/test-integration.sh" ]]; then
                bash "${SCRIPT_DIR}/phases/test-integration.sh"
            else
                test::integration::generate_music
                test::integration::generate_sound
                test::integration::concurrent_requests
            fi
            ;;
        unit)
            echo "📦 Running unit tests..."
            if [[ -f "${SCRIPT_DIR}/phases/test-unit.sh" ]]; then
                bash "${SCRIPT_DIR}/phases/test-unit.sh"
            else
                test::unit::configuration
                test::unit::directories
            fi
            ;;
        *)
            echo "Unknown test phase: $phase"
            echo "Available phases: all, smoke, integration, unit"
            exit 1
            ;;
    esac
}

# Run requested phase
run_phase "$PHASE"