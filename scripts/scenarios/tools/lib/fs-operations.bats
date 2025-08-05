#!/usr/bin/env bats

################################################################################
# BATS Test Suite for fs-operations.sh
#
# Tests the file system operations module using BATS framework
################################################################################

# Setup and teardown
setup() {
    # Load the module under test
    source "${BATS_TEST_DIRNAME}/fs-operations.sh"
    
    # Create temporary test directory
    TEST_TEMP_DIR="${BATS_TMPDIR}/fs-operations-test-${BATS_TEST_NUMBER}"
    mkdir -p "$TEST_TEMP_DIR"
}

teardown() {
    # Clean up test directory
    rm -rf "$TEST_TEMP_DIR" 2>/dev/null || true
}

################################################################################
# Path Resolution Tests
################################################################################

@test "resolve_project_paths sets global variables correctly" {
    resolve_project_paths
    
    [ -n "$FS_PROJECT_ROOT" ]
    [ -n "$FS_SCENARIOS_DIR" ]
    [ -d "$FS_PROJECT_ROOT" ]
    [ -d "$FS_SCENARIOS_DIR" ]
}

@test "resolve_project_paths works with custom offsets" {
    # Test with different project root offset (4 levels up from lib directory)
    resolve_project_paths "../../../.." "scripts/scenarios/core"
    
    [[ "$FS_PROJECT_ROOT" =~ Vrooli$ ]]
    [[ "$FS_SCENARIOS_DIR" =~ scenarios/core$ ]]
}

@test "resolve_scenario_path returns valid path for existing scenario" {
    # First resolve project paths
    resolve_project_paths
    
    # Get first available scenario
    first_scenario=$(list_available_scenarios | head -1)
    skip_if_empty "$first_scenario" "No scenarios available for testing"
    
    run resolve_scenario_path "$first_scenario"
    
    [ "$status" -eq 0 ]
    [ -d "$output" ]
    [[ "$output" =~ $first_scenario$ ]]
}

@test "resolve_scenario_path fails for non-existent scenario" {
    resolve_project_paths
    
    run resolve_scenario_path "non-existent-scenario-12345"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

@test "normalize_path handles absolute paths correctly" {
    run normalize_path "/absolute/path/test"
    
    [ "$status" -eq 0 ]
    [ "$output" = "/absolute/path/test" ]
}

@test "normalize_path converts relative paths" {
    # Create test file for relative path testing
    touch "$TEST_TEMP_DIR/testfile"
    cd "$TEST_TEMP_DIR"
    
    run normalize_path "testfile"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ testfile$ ]]
    [[ "$output" = /* ]]  # Should be absolute
}

################################################################################
# Directory Operations Tests
################################################################################

@test "list_available_scenarios returns scenario names" {
    resolve_project_paths
    
    run list_available_scenarios
    
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    # Output should contain at least one scenario name
    [ "$(echo "$output" | wc -l)" -gt 0 ]
}

@test "create_app_directory_structure creates standard directories" {
    local target_dir="$TEST_TEMP_DIR/app-test"
    
    run create_app_directory_structure "$target_dir"
    
    [ "$status" -eq 0 ]
    
    # Check that standard directories were created
    [ -d "$target_dir/bin" ]
    [ -d "$target_dir/config" ]
    [ -d "$target_dir/data" ]
    [ -d "$target_dir/scripts" ]
    [ -d "$target_dir/manifests" ]
}

@test "create_app_directory_structure creates additional directories" {
    local target_dir="$TEST_TEMP_DIR/app-custom-test"
    
    run create_app_directory_structure "$target_dir" "custom1" "custom2"
    
    [ "$status" -eq 0 ]
    
    # Check standard directories
    [ -d "$target_dir/bin" ]
    [ -d "$target_dir/config" ]
    
    # Check additional directories
    [ -d "$target_dir/custom1" ]
    [ -d "$target_dir/custom2" ]
}

@test "check_directory_writable succeeds for writable directory" {
    run check_directory_writable "$TEST_TEMP_DIR"
    
    [ "$status" -eq 0 ]
}

@test "check_directory_writable fails for non-existent parent" {
    run check_directory_writable "/non/existent/parent/dir"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Parent directory does not exist" ]]
}

################################################################################
# File Operations Tests
################################################################################

@test "check_file_exists succeeds for existing file" {
    local test_file="$TEST_TEMP_DIR/existing-file.txt"
    echo "test content" > "$test_file"
    
    run check_file_exists "$test_file"
    
    [ "$status" -eq 0 ]
}

@test "check_file_exists fails for non-existent file" {
    run check_file_exists "$TEST_TEMP_DIR/non-existent-file.txt"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "not found" ]]
}

@test "load_service_json loads valid JSON" {
    local json_file="$TEST_TEMP_DIR/valid.json"
    echo '{"test": "value", "number": 42}' > "$json_file"
    
    run load_service_json "$json_file"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"test": "value"' ]]
    [[ "$output" =~ '"number": 42' ]]
}

@test "load_service_json rejects invalid JSON" {
    local json_file="$TEST_TEMP_DIR/invalid.json"
    echo '{"test": invalid json syntax}' > "$json_file"
    
    run load_service_json "$json_file"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid JSON syntax" ]]
}

@test "create_safe_backup creates backup and metadata" {
    local source_file="$TEST_TEMP_DIR/source.txt"
    echo "original content" > "$source_file"
    
    run create_safe_backup "$source_file" "test-reason"
    
    [ "$status" -eq 0 ]
    
    # Check backup file exists
    local backup_file="$output"
    [ -f "$backup_file" ]
    
    # Check metadata file exists
    [ -f "${backup_file}.meta" ]
    
    # Check backup content matches original
    [ "$(cat "$backup_file")" = "original content" ]
    
    # Check metadata contains expected fields
    [[ "$(cat "${backup_file}.meta")" =~ "test-reason" ]]
    [[ "$(cat "${backup_file}.meta")" =~ "source_file" ]]
}

@test "copy_scenario_files copies files successfully" {
    local source_dir="$TEST_TEMP_DIR/source"
    local target_dir="$TEST_TEMP_DIR/target"
    
    # Create source directory with files
    mkdir -p "$source_dir"
    echo "file1 content" > "$source_dir/file1.txt"
    echo "file2 content" > "$source_dir/file2.txt"
    mkdir -p "$source_dir/subdir"
    echo "subfile content" > "$source_dir/subdir/subfile.txt"
    
    run copy_scenario_files "$source_dir" "$target_dir"
    
    [ "$status" -eq 0 ]
    
    # Check files were copied
    [ -f "$target_dir/file1.txt" ]
    [ -f "$target_dir/file2.txt" ]
    [ -f "$target_dir/subdir/subfile.txt" ]
    
    # Check content was preserved
    [ "$(cat "$target_dir/file1.txt")" = "file1 content" ]
    [ "$(cat "$target_dir/subdir/subfile.txt")" = "subfile content" ]
}

@test "copy_file_safely copies single file with directory creation" {
    local source_file="$TEST_TEMP_DIR/source.txt"
    local target_file="$TEST_TEMP_DIR/new/deep/dir/target.txt"
    
    echo "test content" > "$source_file"
    
    run copy_file_safely "$source_file" "$target_file"
    
    [ "$status" -eq 0 ]
    [ -f "$target_file" ]
    [ "$(cat "$target_file")" = "test content" ]
}

################################################################################
# Safety and Validation Tests
################################################################################

@test "check_required_tools succeeds when all tools are available" {
    run check_required_tools
    
    [ "$status" -eq 0 ]
}

@test "check_required_tools fails when tool is missing" {
    run check_required_tools "non-existent-tool-12345"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Required tools not found" ]]
    [[ "$output" =~ "non-existent-tool-12345" ]]
}

@test "check_required_tools accepts additional tools" {
    run check_required_tools "bash" "echo"
    
    [ "$status" -eq 0 ]
}

@test "validate_scenario_structure succeeds for valid scenario" {
    resolve_project_paths
    
    # Get first available scenario
    local first_scenario=$(list_available_scenarios | head -1)
    skip_if_empty "$first_scenario" "No scenarios available for testing"
    
    local scenario_path=$(resolve_scenario_path "$first_scenario")
    
    run validate_scenario_structure "$scenario_path"
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "validation passed" ]]
}

@test "validate_scenario_structure fails for invalid scenario" {
    local fake_scenario="$TEST_TEMP_DIR/fake-scenario"
    mkdir -p "$fake_scenario"
    # No service.json file
    
    run validate_scenario_structure "$fake_scenario"
    
    [ "$status" -eq 1 ]
    [[ "$output" =~ "service.json" ]]
    [[ "$output" =~ "not found" ]]
}

@test "run_preflight_checks succeeds with valid setup" {
    run run_preflight_checks "$TEST_TEMP_DIR"
    
    [ "$status" -eq 0 ]
}

@test "run_preflight_checks fails with invalid directory" {
    run run_preflight_checks "/root/no-permission-dir"
    
    [ "$status" -eq 1 ]
}

################################################################################
# Utility Tests
################################################################################

@test "make_executable sets executable permissions" {
    local test_dir="$TEST_TEMP_DIR/exec-test"
    mkdir -p "$test_dir"
    
    # Create test script
    echo '#!/bin/bash' > "$test_dir/test-script.sh"
    echo 'echo "test"' >> "$test_dir/test-script.sh"
    
    # Should not be executable initially
    [ ! -x "$test_dir/test-script.sh" ]
    
    run make_executable "$test_dir" "*.sh"
    
    [ "$status" -eq 0 ]
    [ -x "$test_dir/test-script.sh" ]
}

@test "get_scenario_name extracts name from path" {
    run get_scenario_name "/path/to/scenarios/my-scenario-name"
    
    [ "$status" -eq 0 ]
    [ "$output" = "my-scenario-name" ]
}

@test "fs_operations_info displays module information" {
    resolve_project_paths
    
    run fs_operations_info
    
    [ "$status" -eq 0 ]
    [[ "$output" =~ "File System Operations Module" ]]
    [[ "$output" =~ "Available Functions" ]]
    [[ "$output" =~ "$FS_PROJECT_ROOT" ]]
}

################################################################################
# Integration Tests
################################################################################

@test "complete workflow: resolve paths, validate scenario, create app structure" {
    # Resolve project paths
    resolve_project_paths
    
    # Get a real scenario
    local scenario_name=$(list_available_scenarios | head -1)
    skip_if_empty "$scenario_name" "No scenarios available for integration test"
    
    local scenario_path=$(resolve_scenario_path "$scenario_name")
    
    # Validate the scenario
    run validate_scenario_structure "$scenario_path"
    [ "$status" -eq 0 ]
    
    # Create app structure
    local app_dir="$TEST_TEMP_DIR/integration-app"
    run create_app_directory_structure "$app_dir"
    [ "$status" -eq 0 ]
    
    # Copy scenario files
    run copy_scenario_files "$scenario_path" "$app_dir/data"
    [ "$status" -eq 0 ]
    
    # Verify service.json was copied
    [ -f "$app_dir/data/service.json" ]
    
    # Load and validate the copied service.json
    run load_service_json "$app_dir/data/service.json"
    [ "$status" -eq 0 ]
}

################################################################################
# Helper Functions
################################################################################

# Skip test if value is empty
skip_if_empty() {
    local value="$1"
    local message="$2"
    
    if [[ -z "$value" ]]; then
        skip "$message"
    fi
}