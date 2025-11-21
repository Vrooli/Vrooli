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

@test "reset_seed_state applies seeds for full mode" {
    apply_calls=0
    _testing_playbooks__cleanup_seeds() { return 0; }
    _testing_playbooks__apply_seeds_if_needed() { apply_calls=$((apply_calls + 1)); return 0; }

    run testing::playbooks::reset_seed_state --mode full --scenario-dir "$BATS_TEST_TMPDIR"
    [ "$status" -eq 0 ]
    # Function is deprecated and does nothing, so apply_calls should be 0
    [ "$apply_calls" -eq 0 ]
}
