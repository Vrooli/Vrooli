#!/usr/bin/env bash
# Scenario dependency helpers: resolve and start dependent scenarios before lifecycle phases.
set -euo pipefail

# Guard against multiple sourcing
if [[ -n "${_SCENARIO_DEPENDENCIES_SH:-}" ]]; then
    return 0
fi
_SCENARIO_DEPENDENCIES_SH=1

# These arrays are shared across all dependency operations
# shellcheck disable=SC2034 # referenced via functions in this file
declare -ag SCENARIO_DEPENDENCY_STACK=()

################################################################################
# Internal stack helpers
################################################################################
scenario::dependencies::stack_push() {
    local name="$1"
    SCENARIO_DEPENDENCY_STACK+=("$name")
}

scenario::dependencies::stack_pop() {
    local expected="${1:-}"
    local length=${#SCENARIO_DEPENDENCY_STACK[@]}
    (( length == 0 )) && return 0

    local idx=$((length - 1))
    local actual="${SCENARIO_DEPENDENCY_STACK[$idx]}"
    unset 'SCENARIO_DEPENDENCY_STACK[$idx]'
    if (( length == 1 )); then
        SCENARIO_DEPENDENCY_STACK=()
    else
        # Rebuild array to avoid sparse indexes
        SCENARIO_DEPENDENCY_STACK=(${SCENARIO_DEPENDENCY_STACK[@]})
    fi

    if [[ -n "$expected" && "$actual" != "$expected" ]]; then
        log::warn "Dependency stack imbalance: expected '$expected' but popped '$actual'"
    fi
}

scenario::dependencies::stack_contains() {
    local target="$1"
    for item in "${SCENARIO_DEPENDENCY_STACK[@]}"; do
        [[ "$item" == "$target" ]] && return 0
    done
    return 1
}

scenario::dependencies::report_cycle() {
    local dependency="$1"
    local path="${SCENARIO_DEPENDENCY_STACK[*]} -> $dependency"
    log::error "🔁 Circular scenario dependency detected: $path"
}

################################################################################
# Phase helpers
################################################################################
scenario::dependencies::phase_requires_bootstrap() {
    local phase="${1:-}"
    case "$phase" in
        develop|test)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

scenario::dependencies::dependency_phase() {
    # For now dependencies always need to be running, so boot them in develop mode
    echo "develop"
}

################################################################################
# Entry point for ensuring dependencies are running
################################################################################
scenario::dependencies::ensure_started() {
    local scenario_name="$1"
    local caller_phase="$2"

    local scenario_dir="${var_ROOT_DIR}/scenarios/${scenario_name}"
    local service_json="${scenario_dir}/.vrooli/service.json"

    if [[ ! -f "$service_json" ]]; then
        log::debug "No service.json for scenario '$scenario_name' - skipping dependency bootstrap"
        return 0
    fi

    local dependencies_json
    dependencies_json=$(jq -c '.dependencies.scenarios // {}' "$service_json" 2>/dev/null || echo '{}')

    if [[ "$dependencies_json" == "{}" ]]; then
        return 0
    fi

    local dependency_phase
    dependency_phase=$(scenario::dependencies::dependency_phase "$caller_phase")

    local dependency
    while IFS= read -r dependency; do
        [[ -z "$dependency" ]] && continue
        local required
        required=$(echo "$dependencies_json" | jq -r --arg dep "$dependency" '.[$dep].required // true' 2>/dev/null || echo "true")
        if [[ "${required,,}" != "true" ]]; then
            continue
        fi

        if scenario::dependencies::stack_contains "$dependency"; then
            scenario::dependencies::report_cycle "$dependency"
            return 1
        fi

        if ! scenario::dependencies::start_dependency "$scenario_name" "$dependency" "$dependency_phase"; then
            return 1
        fi
    done < <(echo "$dependencies_json" | jq -r 'keys[]?' 2>/dev/null || true)

    return 0
}

scenario::dependencies::start_dependency() {
    local parent="$1"
    local dependency="$2"
    local dependency_phase="$3"

    if [[ "$parent" == "$dependency" ]]; then
        log::error "Scenario '$parent' cannot depend on itself"
        return 1
    fi

    local dependency_dir="${var_ROOT_DIR}/scenarios/${dependency}"
    if [[ ! -d "$dependency_dir" ]]; then
        log::error "Scenario '$parent' depends on missing scenario '$dependency'"
        return 1
    fi

    log::info "🔗 Ensuring dependency '$dependency' for scenario '$parent'"
    if ! scenario::run "$dependency" "$dependency_phase"; then
        log::error "Failed to start dependency '$dependency' required by '$parent'"
        return 1
    fi

    return 0
}
