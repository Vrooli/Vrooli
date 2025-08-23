#!/usr/bin/env bats

# OBS Studio Injection Tests

setup() {
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-support/load"
    load "${BATS_TEST_DIRNAME}/../../../../__test/helpers/bats-assert/load"
    
    # Source the injection library
    source "${BATS_TEST_DIRNAME}/../lib/inject.sh"
    
    # Test data directory
    export TEST_DATA_DIR="${BATS_TEST_DIRNAME}/../../../../__test/fixtures/data"
    export OBS_TEST_DIR="${TEST_DATA_DIR}/obs-studio"
    
    # Create test directory if needed
    mkdir -p "$OBS_TEST_DIR"
}

@test "obs::check_health returns success when OBS is running" {
    # Mock obs::is_running to return success
    obs::is_running() { return 0; }
    
    run obs::check_health
    assert_success
}

@test "obs::check_health returns failure when OBS is not running" {
    # Mock obs::is_running to return failure
    obs::is_running() { return 1; }
    
    run obs::check_health
    assert_failure
    assert_output --partial "not accessible"
}

@test "obs::validate_scene validates correct scene config" {
    # Create a valid test scene
    cat > "$OBS_TEST_DIR/valid-scene.json" <<EOF
{
    "name": "Test Scene",
    "sources": [
        {
            "name": "Test Source",
            "type": "color_source"
        }
    ]
}
EOF
    
    local scene_obj
    scene_obj=$(jq -n --arg file "$OBS_TEST_DIR/valid-scene.json" '{file: $file}')
    
    run obs::validate_scene "$scene_obj" 0 "Test Scene"
    assert_success
}

@test "obs::validate_scene fails on missing file" {
    local scene_obj
    scene_obj=$(jq -n --arg file "/nonexistent/file.json" '{file: $file}')
    
    run obs::validate_scene "$scene_obj" 0 "Test Scene"
    assert_failure
    assert_output --partial "file not found"
}

@test "obs::validate_scene fails on invalid JSON" {
    # Create an invalid JSON file
    echo "not valid json" > "$OBS_TEST_DIR/invalid.json"
    
    local scene_obj
    scene_obj=$(jq -n --arg file "$OBS_TEST_DIR/invalid.json" '{file: $file}')
    
    run obs::validate_scene "$scene_obj" 0 "Test Scene"
    assert_failure
    assert_output --partial "not valid JSON"
}

@test "obs::process_scenes handles empty scenes array" {
    local config='{"scenes": []}'
    
    run obs::process_scenes "$config"
    assert_success
    assert_output --partial "No scenes to inject"
}

@test "obs::inject validates JSON configuration" {
    # Mock health check
    obs::check_health() { return 0; }
    
    run obs::inject "invalid json"
    assert_failure
    assert_output --partial "Invalid JSON"
}

teardown() {
    # Clean up test files
    rm -rf "$OBS_TEST_DIR"
}