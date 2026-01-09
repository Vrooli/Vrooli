#!/usr/bin/env bash
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

usage() {
    cat <<'USAGE'
Usage: refresh-shared-package.sh <package-dir> <scenario|all> [options]

Rebuilds a shared package in packages/ and re-runs scenario setup so that
dependents pull in the updated bundle. Scenarios that were running get
restarted (unless --no-restart is provided); stopped scenarios remain stopped
after setup completes. When "all" is specified, only scenarios that actually
declare the package as a dependency are touched.

Arguments:
  <package-dir>   Folder name under packages/ (e.g., api-base)
  <scenario|all>  Scenario name to refresh or "all" for every scenario

Options:
  --no-restart    Do not restart scenarios that are currently running.
                  They stay up and you'll need to run `vrooli scenario restart`
                  manually to pick up the new build.
  -h, --help      Show this help.
USAGE
}

PACKAGE_INPUT=""
SCENARIO_TARGET=""
AUTO_RESTART=true

POSITIONAL=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-restart)
            AUTO_RESTART=false
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --)
            shift
            while [[ $# -gt 0 ]]; do
                POSITIONAL+=("$1")
                shift
            done
            ;;
        -*)
            log::error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            POSITIONAL+=("$1")
            shift
            ;;
    esac
done

if [[ ${#POSITIONAL[@]} -lt 2 ]]; then
    usage
    exit 1
fi

PACKAGE_INPUT="${POSITIONAL[0]}"
SCENARIO_TARGET="${POSITIONAL[1]}"

PACKAGE_DIR="${var_PACKAGES_DIR}/${PACKAGE_INPUT}"
if [[ ! -d "$PACKAGE_DIR" ]]; then
    log::error "Package directory not found: $PACKAGE_DIR"
    exit 1
fi

PACKAGE_JSON="${PACKAGE_DIR}/package.json"
if [[ ! -f "$PACKAGE_JSON" ]]; then
    log::error "package.json not found in ${PACKAGE_DIR}"
    exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
    log::error "jq is required to parse package metadata"
    exit 1
fi

PACKAGE_NAME=$(jq -r '.name // ""' "$PACKAGE_JSON")
if [[ -z "$PACKAGE_NAME" || "$PACKAGE_NAME" == "null" ]]; then
    log::error "Unable to read package name from ${PACKAGE_JSON}"
    exit 1
fi

log::info "Rebuilding shared package ${PACKAGE_NAME}"
(
    cd "$APP_ROOT"
    pnpm --filter "$PACKAGE_NAME" install
    pnpm --filter "$PACKAGE_NAME" build
)

ensure_lifecycle_loaded() {
    if declare -F lifecycle::is_scenario_running >/dev/null 2>&1; then
        return
    fi
    lifecycle::main() { return 0; }
    # shellcheck disable=SC1091
    source "${var_LIB_UTILS_DIR}/lifecycle.sh"
    unset -f lifecycle::main
}

ensure_lifecycle_loaded

scenario_uses_package() {
    local scenario="$1"
    local scenario_path="${var_SCENARIOS_DIR}/${scenario}"
    [[ -d "$scenario_path" ]] || return 1

    local found=false
    while IFS= read -r -d '' pkg; do
        if jq -e --arg name "$PACKAGE_NAME" '
            (.dependencies // {} | has($name)) or
            (.devDependencies // {} | has($name)) or
            (.peerDependencies // {} | has($name)) or
            (.optionalDependencies // {} | has($name))
        ' "$pkg" >/dev/null 2>&1; then
            found=true
            break
        fi
    done < <(find "$scenario_path" -type f -name package.json \
        -not -path '*/node_modules/*' -print0 2>/dev/null)

    [[ "$found" == true ]]
}

collect_scenarios() {
    local target="$1"
    local -n _result=$2

    if [[ "$target" == "all" ]]; then
        if [[ ! -d "$var_SCENARIOS_DIR" ]]; then
            log::error "Scenarios directory not found: $var_SCENARIOS_DIR"
            exit 1
        fi
        while IFS= read -r -d '' dir; do
            local scenarios_name
            scenarios_name="$(basename "$dir")"
            if scenario_uses_package "$scenarios_name"; then
                _result+=("$scenarios_name")
            else
                log::info "Skipping ${scenarios_name} (does not depend on ${PACKAGE_NAME})"
            fi
        done < <(find "$var_SCENARIOS_DIR" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)
    else
        local scenario_path="${var_SCENARIOS_DIR}/${target}"
        if [[ ! -d "$scenario_path" ]]; then
            log::error "Scenario not found: $target"
            exit 1
        fi
        if scenario_uses_package "$target"; then
            _result+=("$target")
        else
            log::warning "Scenario ${target} does not depend on ${PACKAGE_NAME}; nothing to refresh."
        fi
    fi
}

run_scenario_setup() {
    local scenario="$1"
    log::info "Running setup for ${scenario}"
    vrooli scenario setup "$scenario"
}

stop_scenario_if_running() {
    local scenario="$1"
    if lifecycle::is_scenario_running "$scenario"; then
        log::info "Stopping running scenario ${scenario}"
        if ! vrooli scenario stop "$scenario"; then
            log::warning "Unable to stop ${scenario} via lifecycle; continuing"
        fi
    fi
}

start_scenario() {
    local scenario="$1"
    log::info "Starting scenario ${scenario}"
    vrooli scenario run "$scenario"
}

SCENARIOS=()
collect_scenarios "$SCENARIO_TARGET" SCENARIOS

if [[ ${#SCENARIOS[@]} -eq 0 ]]; then
    log::warning "No scenarios found to refresh"
    exit 0
fi

log::info "Processing ${#SCENARIOS[@]} scenario(s): ${SCENARIOS[*]}"

RESTARTED=()
SETUP_ONLY=()
SKIPPED_RESTART=()

FAILED=()

for scenario in "${SCENARIOS[@]}"; do
    log::info "────────────────────────────────────────"
    log::info "Scenario: ${scenario}"

    local_running=false
    if lifecycle::is_scenario_running "$scenario"; then
        local_running=true
        log::info "Current state: running"
    else
        log::info "Current state: stopped"
    fi

    if [[ "$local_running" == true ]]; then
        if [[ "$AUTO_RESTART" == true ]]; then
            if stop_scenario_if_running "$scenario" && run_scenario_setup "$scenario" && start_scenario "$scenario"; then
                RESTARTED+=("$scenario")
            else
                FAILED+=("$scenario")
                log::error "Scenario ${scenario} failed during refresh; continuing"
                continue
            fi
        else
            log::warning "--no-restart set; scenario '${scenario}' remains running. Restart manually after validating."
            if run_scenario_setup "$scenario"; then
                SKIPPED_RESTART+=("$scenario")
            else
                FAILED+=("$scenario")
                log::error "Scenario ${scenario} failed during setup; continuing"
                continue
            fi
        fi
    else
        if run_scenario_setup "$scenario"; then
            SETUP_ONLY+=("$scenario")
        else
            FAILED+=("$scenario")
            log::error "Scenario ${scenario} failed during setup; continuing"
            continue
        fi
    fi

    log::success "Finished ${scenario}"
    log::info ""
done

log::info "========================================"
log::success "Package refresh complete"

if [[ ${#RESTARTED[@]} -gt 0 ]]; then
    log::info "Restarted scenarios: ${RESTARTED[*]}"
fi
if [[ ${#SETUP_ONLY[@]} -gt 0 ]]; then
    log::info "Setup-only (remained stopped): ${SETUP_ONLY[*]}"
fi
if [[ ${#SKIPPED_RESTART[@]} -gt 0 ]]; then
    log::warning "Manual restart required for: ${SKIPPED_RESTART[*]}"
fi
if [[ ${#FAILED[@]} -gt 0 ]]; then
    log::error "Failed scenarios (check logs): ${FAILED[*]}"
fi
