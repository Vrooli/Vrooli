#!/usr/bin/env bash
set -euo pipefail

# Get script directory for sourcing utils
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
source "${SCRIPT_DIR}/../utils/var.sh"
source "${SCRIPT_DIR}/../utils/log.sh"

scenario::run() {
    local scenario_name="$1"
    shift
    
    local scenario_path="${var_ROOT_DIR}/scenarios/${scenario_name}"
    
    if [[ ! -d "$scenario_path" ]]; then
        log::error "Scenario not found: $scenario_name"
        return 1
    fi
    
    # Get the phase (default to 'develop' if not specified)
    local phase="${1:-develop}"
    shift || true
    
    # Set up logging for the scenario lifecycle execution
    local lifecycle_log="${HOME}/.vrooli/logs/${scenario_name}.log"
    mkdir -p "$(dirname "$lifecycle_log")"
    
    # Clear the log file for fresh execution (no confusion from old runs)
    > "$lifecycle_log"
    
    # Call lifecycle.sh directly, capturing output to both console and log file
    log::info "Running scenario '$scenario_name' with direct lifecycle execution"
    
    # Use tee to show output on console AND write to log file
    # This preserves real-time output while capturing for later review
    "${SCRIPT_DIR}/../utils/lifecycle.sh" "$scenario_name" "$phase" "$@" 2>&1 | tee -a "$lifecycle_log"
    
    # Preserve the exit code from the pipeline
    return "${PIPESTATUS[0]}"
}

scenario::list() {
    log::info "Available scenarios:"
    for scenario in "${var_ROOT_DIR}"/scenarios/*/; do
        if [[ -d "$scenario" ]]; then
            local name="${scenario%/}"
            name="${name##*/}"
            local service_json="${scenario}/.vrooli/service.json"
            if [[ -f "$service_json" ]]; then
                local description=$(jq -r '.service.description // ""' "$service_json" 2>/dev/null || echo "")
                if [[ -n "$description" ]]; then
                    echo "  • $name - $description"
                else
                    echo "  • $name"
                fi
            else
                echo "  • $name"
            fi
        fi
    done
}

scenario::test() {
    local scenario_name="$1"
    shift
    scenario::run "$scenario_name" test "$@"
}