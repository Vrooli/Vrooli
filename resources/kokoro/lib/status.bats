#!/usr/bin/env bats
# Tests for Kokoro status.sh functions

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../test/test-helper.bash"

setup_file() {
    vrooli_setup_service_test "kokoro"

    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    KOKORO_DIR="$(dirname "$SCRIPT_DIR")"

    source "${KOKORO_DIR}/config/defaults.sh"
    source "${KOKORO_DIR}/config/messages.sh"
    source "${KOKORO_DIR}/lib/common.sh"
    source "${SCRIPT_DIR}/status.sh"

    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_KOKORO_DIR="$KOKORO_DIR"
}

setup() {
    vrooli_auto_setup

    export KOKORO_PORT="8880"
    export KOKORO_BASE_URL="http://localhost:8880"
    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_DATA_DIR="${BATS_TEST_TMPDIR}/kokoro-status"
    export KOKORO_VOICES_DIR="${KOKORO_DATA_DIR}/voices"
    export KOKORO_DEFAULT_VOICE="af_heart"
    export KOKORO_GPU_ENABLED="no"
    export KOKORO_API_TIMEOUT="1"

    mkdir -p "${KOKORO_DATA_DIR}" "${KOKORO_VOICES_DIR}"

    common::container_exists() { return 0; }
    common::is_running() { return 0; }
    kokoro::is_healthy() { return 0; }
    kokoro::get_docker_image() { echo "cpu-image"; }
    curl() { echo '["af_heart","af_bella"]'; }
    timeout() { shift; "$@"; }
    docker() {
        case "$1" in
            inspect) echo "running" ;;
            stats) echo "1.5%;512MiB / 2GiB" ;;
            *) return 0 ;;
        esac
    }
    export -f common::container_exists common::is_running kokoro::is_healthy kokoro::get_docker_image curl timeout docker
}

teardown() {
    vrooli_cleanup_test
}

@test "kokoro::status::collect_data reports healthy running resource" {
    run kokoro::status::collect_data

    [ "$status" -eq 0 ]
    [[ "$output" == *$'installed\ntrue'* ]]
    [[ "$output" == *$'running\ntrue'* ]]
    [[ "$output" == *$'healthy\ntrue'* ]]
    [[ "$output" == *$'health_message\nHealthy - TTS synthesis service ready'* ]]
}

@test "status::quick_status returns running when container is healthy" {
    run status::quick_status

    [ "$status" -eq 0 ]
    [ "$output" = "running" ]
}

@test "status::quick_status returns stopped when container is not running" {
    common::is_running() { return 1; }
    export -f common::is_running

    run status::quick_status

    [ "$status" -eq 1 ]
    [ "$output" = "stopped" ]
}

@test "status::is_ready succeeds when voices endpoint responds" {
    curl() { echo "200"; }
    export -f curl

    run status::is_ready

    [ "$status" -eq 0 ]
}
