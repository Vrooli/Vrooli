#!/usr/bin/env bats

log::info() { echo "INFO: $*"; }
log::warn() { echo "WARN: $*"; }
log::error() { echo "ERROR: $*" >&2; }
log::debug() { echo "DEBUG: $*" >&2; }

declare -ag scenario_run_calls=()

SCRIPT_DIR="$(cd "$(dirname "${BATS_TEST_FILENAME:-${BASH_SOURCE[0]}}")" && pwd)"
source "${SCRIPT_DIR}/dependencies.sh"

setup() {
    export var_ROOT_DIR="$BATS_TEST_TMPDIR/app"
    mkdir -p "$var_ROOT_DIR/scenarios"
    scenario_run_calls=()
    SCENARIO_DEPENDENCY_STACK=()
}

# shellcheck disable=SC2317 # referenced indirectly during tests
scenario::run() {
    local scenario_name="$1"
    local phase="${2:-develop}"
    scenario_run_calls+=("${scenario_name}:${phase}")
    scenario::dependencies::stack_push "$scenario_name"
    scenario::dependencies::ensure_started "$scenario_name" "$phase"
    local result=$?
    scenario::dependencies::stack_pop "$scenario_name"
    return $result
}

create_scenario() {
    local name="$1"
    local json="$2"
    local dir="$var_ROOT_DIR/scenarios/$name/.vrooli"
    mkdir -p "$dir"
    printf '%s\n' "$json" > "$dir/service.json"
}

@test "starts required scenario dependencies" {
    create_scenario "child" '{"service":{"name":"child"},"dependencies":{"resources":{}}}'
    create_scenario "parent" '{"service":{"name":"parent"},"dependencies":{"resources":{},"scenarios":{"child":{"required":true}}}}'

    scenario_run_calls=()
    scenario::dependencies::stack_push "parent"
    scenario::dependencies::ensure_started "parent" "develop"
    local result=$?
    scenario::dependencies::stack_pop "parent"

    [ "$result" -eq 0 ]
    [ "${#scenario_run_calls[@]}" -gt 0 ]
    [ "${scenario_run_calls[0]-}" = "child:develop" ]
}

@test "fails when dependency is missing" {
    create_scenario "lonely" '{"service":{"name":"lonely"},"dependencies":{"resources":{},"scenarios":{"ghost":{"required":true}}}}'

    scenario::dependencies::stack_push "lonely"
    run scenario::dependencies::ensure_started "lonely" "develop"
    scenario::dependencies::stack_pop "lonely"

    [ "$status" -ne 0 ]
    [[ "$output" == *"missing scenario 'ghost'"* ]]
}

@test "detects circular dependencies" {
    create_scenario "alpha" '{"service":{"name":"alpha"},"dependencies":{"resources":{},"scenarios":{"beta":{"required":true}}}}'
    create_scenario "beta" '{"service":{"name":"beta"},"dependencies":{"resources":{},"scenarios":{"alpha":{"required":true}}}}'

    scenario::dependencies::stack_push "alpha"
    run scenario::dependencies::ensure_started "alpha" "develop"
    scenario::dependencies::stack_pop "alpha"

    [ "$status" -ne 0 ]
    [[ "$output" == *"Circular scenario dependency"* ]]
}
