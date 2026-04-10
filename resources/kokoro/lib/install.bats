#!/usr/bin/env bats
# Tests for Kokoro installation functions

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../test/test-helper.bash"

setup_file() {
    vrooli_setup_service_test "kokoro"

    export SETUP_FILE_KOKORO_LIB_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_KOKORO_DIR="$(dirname "${BATS_TEST_DIRNAME}")"
}

setup() {
    vrooli_auto_setup

    KOKORO_LIB_DIR="${SETUP_FILE_KOKORO_LIB_DIR}"
    KOKORO_DIR="${SETUP_FILE_KOKORO_DIR}"

    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_DATA_DIR="${BATS_TEST_TMPDIR}/kokoro-install"
    export KOKORO_VOICES_DIR="${KOKORO_DATA_DIR}/voices"
    export FORCE="false"

    source "${KOKORO_DIR}/config/defaults.sh"
    source "${KOKORO_DIR}/config/messages.sh"
    source "${KOKORO_DIR}/lib/common.sh"
    source "${KOKORO_DIR}/lib/docker.sh"
    source "${KOKORO_LIB_DIR}/install.sh"

    defaults::export_config
    messages::export_messages

    common::check_docker() { return 0; }
    common::container_exists() { return 1; }
    common::is_running() { return 1; }
    kokoro::create_directories() { mkdir -p "${KOKORO_VOICES_DIR}"; return 0; }
    kokoro::docker::pull_image() { echo "pull ok"; return 0; }
    kokoro::docker::start_container() { echo "start ok"; return 0; }
    docker::stop_container() { return 0; }
    docker::remove_container() { return 0; }
    kokoro::wait_for_health() { return 0; }
    export -f common::check_docker common::container_exists common::is_running
    export -f kokoro::create_directories kokoro::docker::pull_image kokoro::docker::start_container
    export -f docker::stop_container docker::remove_container kokoro::wait_for_health
}

teardown() {
    vrooli_cleanup_test
}

@test "kokoro::install is idempotent when container is already running" {
    common::container_exists() { return 0; }
    common::is_running() { return 0; }
    export -f common::container_exists common::is_running

    run kokoro::install

    [ "$status" -eq 0 ]
    [[ "$output" == *"already installed and running"* ]]
}

@test "kokoro::install starts container when prerequisites succeed" {
    run kokoro::install

    [ "$status" -eq 0 ]
    [[ "$output" == *"installed and running successfully"* ]]
}

@test "kokoro::install fails when image pull fails" {
    kokoro::docker::pull_image() { return 1; }
    export -f kokoro::docker::pull_image

    run kokoro::install

    [ "$status" -eq 1 ]
    [[ "$output" == *"Failed to pull Kokoro Docker image"* ]] || [[ "$output" == *"installation failed"* ]]
}

@test "kokoro::start tells callers to use manage install when container is missing" {
    common::is_running() { return 1; }
    common::container_exists() { return 1; }
    export -f common::is_running common::container_exists

    run kokoro::start

    [ "$status" -eq 1 ]
    [[ "$output" == *"resource-kokoro manage install"* ]]
}
