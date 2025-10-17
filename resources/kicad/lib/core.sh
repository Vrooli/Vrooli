#!/bin/bash
# KiCad Core Functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_CORE_DIR="${APP_ROOT}/resources/kicad/lib"
KICAD_CLI_DIR="${APP_ROOT}/resources/kicad"

# Source shared utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${KICAD_CLI_DIR}/config/defaults.sh"
source "${KICAD_CORE_DIR}/common.sh"

# Test functions delegation
kicad::test::smoke() {
    "${KICAD_CLI_DIR}/test/phases/test-smoke.sh"
    return $?
}

kicad::test::integration() {
    "${KICAD_CLI_DIR}/test/phases/test-integration.sh"
    return $?
}

kicad::test::unit() {
    "${KICAD_CLI_DIR}/test/phases/test-unit.sh"
    return $?
}

kicad::test::all() {
    "${KICAD_CLI_DIR}/test/run-tests.sh" --all
    return $?
}

# Show resource runtime information (v2.0 contract requirement)
kicad::info() {
    local json_flag="${1:-}"
    local runtime_file="${KICAD_CLI_DIR}/config/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        log::error "Runtime configuration not found at $runtime_file"
        return 1
    fi
    
    if [[ "$json_flag" == "--json" ]]; then
        cat "$runtime_file"
    else
        log::header "KiCad Resource Information"
        log::info "Runtime Configuration:"
        jq -r '
            "  Startup Order: \(.startup_order)",
            "  Startup Timeout: \(.startup_timeout)s",
            "  Startup Time Estimate: \(.startup_time_estimate)",
            "  Dependencies: \(.dependencies | join(", "))",
            "  Recovery Attempts: \(.recovery_attempts)",
            "  Priority: \(.priority)"
        ' "$runtime_file"
    fi
    return 0
}

# Export functions
export -f kicad::test::smoke
export -f kicad::test::integration
export -f kicad::test::unit
export -f kicad::test::all
export -f kicad::info