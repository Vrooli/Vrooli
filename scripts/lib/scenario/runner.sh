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
    
    # Set context for scenario execution
    export SCENARIO_NAME="$scenario_name"
    export SCENARIO_PATH="$scenario_path"
    # APP_ROOT remains as-is in scenarios (already correct)
    
    # Change to scenario directory
    cd "$scenario_path" || return 1
    
    # Execute manage.sh which will handle the scenario context
    "${var_ROOT_DIR}/scripts/manage.sh" "$@"
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