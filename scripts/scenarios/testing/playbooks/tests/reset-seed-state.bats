#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

load_runner() {
    local runner_path="${BATS_TEST_DIRNAME}/../workflow-runner.sh"
    if [ ! -f "$runner_path" ]; then
        fail "workflow-runner.sh not found at ${runner_path}"
    fi
    local previous_opts
    previous_opts=$(set +o)
    set +u +e +o pipefail
    # shellcheck disable=SC1090
    source "$runner_path"
    eval "$previous_opts"
}

setup() {
    load_runner
    TESTING_PLAYBOOKS_LAST_SCENARIO="browser-automation-studio"
}

@test "reset_seed_state skips work when mode is none" {
    apply_calls=0
    _testing_playbooks__cleanup_seeds() { return 0; }
    _testing_playbooks__apply_seeds_if_needed() { apply_calls=$((apply_calls + 1)); return 0; }

    run testing::playbooks::reset_seed_state --mode none --scenario-dir "$BATS_TEST_TMPDIR"
    [ "$status" -eq 0 ]
    [ "$apply_calls" -eq 0 ]
}

@test "reset_seed_state applies seeds for project mode" {
    apply_calls=0
    _testing_playbooks__cleanup_seeds() { return 0; }
    _testing_playbooks__apply_seeds_if_needed() { apply_calls=$((apply_calls + 1)); return 0; }

    run testing::playbooks::reset_seed_state --mode project --scenario-dir "$BATS_TEST_TMPDIR"
    [ "$status" -eq 0 ]
    [ "$apply_calls" -eq 1 ]
}

@test "reset_seed_state restarts scenario for global mode" {
    _testing_playbooks__cleanup_seeds() { return 0; }
    _testing_playbooks__apply_seeds_if_needed() { return 0; }
    testing::core::wait_for_scenario() { return 0; }

    vrooli_calls=()
    vrooli() {
        vrooli_calls+=("$*")
        return 0
    }

    run testing::playbooks::reset_seed_state --mode global --scenario-dir "$BATS_TEST_TMPDIR" --scenario-name bas-cli
    [ "$status" -eq 0 ]
    [[ "${vrooli_calls[*]}" == *"scenario restart bas-cli --clean-stale"* ]]
}
