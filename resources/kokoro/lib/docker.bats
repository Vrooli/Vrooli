#!/usr/bin/env bats
# Tests for Kokoro docker.sh functions

# shellcheck disable=SC1091
source "${BATS_TEST_DIRNAME}/../test/test-helper.bash"

setup_file() {
    vrooli_setup_service_test "kokoro"

    export SETUP_FILE_SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    export SETUP_FILE_KOKORO_DIR="$(dirname "${BATS_TEST_DIRNAME}")"
}

setup() {
    vrooli_auto_setup

    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    KOKORO_DIR="${SETUP_FILE_KOKORO_DIR}"

    export KOKORO_CUSTOM_PORT="8880"
    export KOKORO_CONTAINER_NAME="kokoro-test"
    export KOKORO_DATA_DIR="${BATS_TEST_TMPDIR}/kokoro-data"
    export KOKORO_VOICES_DIR="${KOKORO_DATA_DIR}/voices"
    export KOKORO_IMAGE="gpu-image"
    export KOKORO_CPU_IMAGE="cpu-image"
    export KOKORO_GPU_ENABLED="no"
    export KOKORO_INITIALIZATION_WAIT="0"

    mkdir -p "${KOKORO_VOICES_DIR}"

    source "${KOKORO_DIR}/config/defaults.sh"
    source "${KOKORO_DIR}/config/messages.sh"
    source "${KOKORO_DIR}/lib/common.sh"
    source "${SCRIPT_DIR}/docker.sh"

    defaults::export_config
    messages::export_messages

    docker::check_daemon() { return 0; }
    docker::image_exists() { return 0; }
    docker::create_network() { return 0; }
    docker::container_exists() { return 1; }
    docker::is_running() { return 1; }
    docker::pull_image() { echo "pull:$1"; return 0; }
    docker_resource::create_service_advanced() {
        local volumes="$5"
        local env_name="$6"
        local opts_name="$7"
        eval "local env_array=(\"\${${env_name}[@]}\")"
        eval "local opts_array=(\"\${${opts_name}[@]}\")"
        echo "volumes=${volumes}"
        echo "env=${env_array[*]}"
        echo "opts=${opts_array[*]}"
        return 0
    }
    export -f docker::check_daemon docker::image_exists docker::create_network
    export -f docker::container_exists docker::is_running docker::pull_image
    export -f docker_resource::create_service_advanced
}

teardown() {
    vrooli_cleanup_test
}

@test "kokoro::docker::pull_image uses CPU image when GPU disabled" {
    kokoro::docker::is_gpu_available() { return 1; }
    export -f kokoro::docker::is_gpu_available

    run kokoro::docker::pull_image "no"

    [ "$status" -eq 0 ]
    [[ "$output" == *"pull:cpu-image"* ]]
}

@test "kokoro::docker::pull_image uses GPU image when GPU enabled and available" {
    export KOKORO_GPU_ENABLED="yes"
    kokoro::docker::is_gpu_available() { return 0; }
    export -f kokoro::docker::is_gpu_available

    run kokoro::docker::pull_image "yes"

    [ "$status" -eq 0 ]
    [[ "$output" == *"pull:gpu-image"* ]]
}

@test "kokoro::docker::start_container skips voices mount when no custom voice files exist" {
    kokoro::docker::is_gpu_available() { return 1; }
    export -f kokoro::docker::is_gpu_available

    run kokoro::docker::start_container

    [ "$status" -eq 0 ]
    [[ "$output" == *"No custom Kokoro voices found"* ]]
    [[ "$output" == *"volumes="* ]]
    [[ "$output" != *"/app/api/src/voices"* ]]
}

@test "kokoro::docker::start_container mounts host voices when custom voice files exist" {
    touch "${KOKORO_VOICES_DIR}/af_heart.pt"
    kokoro::docker::is_gpu_available() { return 1; }
    export -f kokoro::docker::is_gpu_available

    run kokoro::docker::start_container

    [ "$status" -eq 0 ]
    [[ "$output" == *"volumes=${KOKORO_VOICES_DIR}:/app/api/src/voices"* ]]
}

@test "kokoro::docker::start_container adds GPU options when enabled and available" {
    export KOKORO_GPU_ENABLED="yes"
    kokoro::docker::is_gpu_available() { return 0; }
    export -f kokoro::docker::is_gpu_available

    run kokoro::docker::start_container

    [ "$status" -eq 0 ]
    [[ "$output" == *"opts=--gpus all"* ]]
    [[ "$output" == *"env=NVIDIA_VISIBLE_DEVICES=all"* ]]
}
