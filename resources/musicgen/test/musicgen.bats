#!/usr/bin/env bats
# MusicGen Resource Tests

setup() {
    export MUSICGEN_DIR="$(builtin cd "${BATS_TEST_DIRNAME%/*}" && builtin pwd)"
    export VROOLI_DIR="$(builtin cd "${MUSICGEN_DIR}/../../../../" && builtin pwd)"
    source "${MUSICGEN_DIR}/lib/common.sh"
    source "${MUSICGEN_DIR}/lib/core.sh"
}

@test "MusicGen: CLI exists and is executable" {
    [ -x "${MUSICGEN_DIR}/cli.sh" ]
}

@test "MusicGen: Help command works" {
    run "${MUSICGEN_DIR}/cli.sh" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "MusicGen Resource Manager" ]]
}

@test "MusicGen: Status command works" {
    run "${MUSICGEN_DIR}/cli.sh" status
    [ "$status" -eq 0 ]
}

@test "MusicGen: Can check if installed" {
    run musicgen::status
    [ "$status" -eq 0 ]
}

@test "MusicGen: Data directories are created" {
    # These should be created during install
    if musicgen::status | grep -q "Installed: true"; then
        [ -d "${MUSICGEN_DATA_DIR}" ]
        [ -d "${MUSICGEN_MODELS_DIR}" ]
        [ -d "${MUSICGEN_OUTPUT_DIR}" ]
    else
        skip "MusicGen not installed"
    fi
}

@test "MusicGen: Docker image exists when installed" {
    if musicgen::status | grep -q "Installed: true"; then
        run docker images "${MUSICGEN_IMAGE}" --format "{{.Repository}}"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "musicgen" ]]
    else
        skip "MusicGen not installed"
    fi
}

@test "MusicGen: Health endpoint responds when running" {
    if musicgen::status | grep -q "Running: true"; then
        run curl -s "http://localhost:${MUSICGEN_PORT}/health"
        [ "$status" -eq 0 ]
        [[ "$output" =~ "healthy" ]]
    else
        skip "MusicGen not running"
    fi
}

@test "MusicGen: Can list models when running" {
    if musicgen::status | grep -q "Running: true"; then
        run musicgen::list_models
        [ "$status" -eq 0 ]
        [[ "$output" =~ "musicgen" ]]
    else
        skip "MusicGen not running"
    fi
}

@test "MusicGen: Status returns JSON format" {
    run "${MUSICGEN_DIR}/cli.sh" status json
    [ "$status" -eq 0 ]
    # Check if output is valid JSON
    echo "$output" | jq empty
    [ "$?" -eq 0 ]
}