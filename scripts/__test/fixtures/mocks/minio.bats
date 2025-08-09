#!/usr/bin/env bats
# MinIO Mock Test Suite
#
# Comprehensive tests for the MinIO mock implementation
# Tests all mc commands, state management, error injection,
# bucket/object operations, and BATS compatibility features

# Test environment setup
setup() {
    # Set up test directory
    export TEST_DIR="$BATS_TEST_TMPDIR/minio-mock-test"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Set up mock directory
    export MOCK_DIR="${BATS_TEST_DIRNAME}"
    
    # Configure MinIO mock state directory
    export MINIO_MOCK_STATE_DIR="$TEST_DIR/minio-state"
    mkdir -p "$MINIO_MOCK_STATE_DIR"
    
    # Source the MinIO mock
    source "$MOCK_DIR/minio.sh"
    
    # Reset MinIO mock to clean state
    mock::minio::reset
}

teardown() {
    # Clean up test directory
    rm -rf "$TEST_DIR"
}

# Helper functions for assertions
assert_success() {
    if [[ "$status" -ne 0 ]]; then
        echo "Expected success but got status $status" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_failure() {
    if [[ "$status" -eq 0 ]]; then
        echo "Expected failure but got success" >&2
        echo "Output: $output" >&2
        return 1
    fi
}

assert_output() {
    local expected="$1"
    if [[ "$1" == "--partial" ]]; then
        expected="$2"
        if [[ ! "$output" =~ "$expected" ]]; then
            echo "Expected output to contain: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    elif [[ "$1" == "--regexp" ]]; then
        expected="$2"
        if [[ ! "$output" =~ $expected ]]; then
            echo "Expected output to match: $expected" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    else
        if [[ "$output" != "$expected" ]]; then
            echo "Expected: $expected" >&2
            echo "Actual: $output" >&2
            return 1
        fi
    fi
}

refute_output() {
    local pattern="$2"
    if [[ "$1" == "--partial" ]] || [[ "$1" == "--regexp" ]]; then
        if [[ "$output" =~ "$pattern" ]]; then
            echo "Expected output NOT to contain: $pattern" >&2
            echo "Actual output: $output" >&2
            return 1
        fi
    fi
}

assert_line() {
    local index expected
    if [[ "$1" == "--index" ]]; then
        index="$2"
        expected="$3"
        local lines=()
        while IFS= read -r line; do
            lines+=("$line")
        done <<< "$output"
        
        if [[ "${lines[$index]}" != "$expected" ]]; then
            echo "Line $index mismatch" >&2
            echo "Expected: $expected" >&2
            echo "Actual: ${lines[$index]}" >&2
            return 1
        fi
    fi
}

# Basic connectivity and configuration tests
@test "mc: basic version command" {
    run mc version
    assert_success
    assert_output --partial "RELEASE.2024-01-01T00-00-00Z"
}

@test "mc: connection failed when MinIO is stopped" {
    mock::minio::set_state "stopped"
    
    run mc version
    assert_failure
    assert_output --partial "Unable to connect to MinIO server"
}

@test "mc: error injection - connection failed" {
    mock::minio::set_error "connection_failed"
    
    run mc version
    assert_failure
    assert_output --partial "Connection refused"
}

@test "mc: error injection - auth failed" {
    mock::minio::set_error "auth_failed"
    
    run mc version
    assert_failure
    assert_output --partial "Access Denied"
}

@test "mc: error injection - service unavailable" {
    mock::minio::set_error "service_unavailable"
    
    run mc version
    assert_failure
    assert_output --partial "Service Unavailable"
}

@test "mc: command without arguments fails" {
    run mc
    assert_failure
    assert_output --partial "No command specified"
}

@test "mc: unknown command fails" {
    run mc unknowncommand
    assert_failure
    assert_output --partial "Unknown command 'unknowncommand'"
}

# Configuration management tests
@test "mc config host add: basic alias configuration" {
    run mc config host add local http://localhost:9000 minioadmin minioadmin
    assert_success
    assert_output "Added \`local\` successfully."
}

@test "mc config host add: missing arguments fails" {
    run mc config host add local http://localhost:9000
    assert_failure
    assert_output --partial "Missing required arguments"
}

@test "mc config host list: empty list" {
    run mc config host list
    assert_success
    assert_output "No aliases found."
}

@test "mc config host list: shows configured aliases" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc config host add backup http://backup.example.com access secret
    
    run mc config host list
    assert_success
    assert_output --regexp "local -> http://localhost:9000"
    assert_output --regexp "backup -> http://backup.example.com"
}

@test "mc config host rm: remove existing alias" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc config host rm local
    assert_success
    assert_output "Removed \`local\` successfully."
    
    # Verify it's gone
    run mc config host list
    assert_success
    assert_output "No aliases found."
}

@test "mc config host rm: remove non-existent alias fails" {
    run mc config host rm nonexistent
    assert_failure
    assert_output --partial "Alias \`nonexistent\` does not exist"
}

@test "mc config: unknown subcommand fails" {
    run mc config unknown
    assert_failure
    assert_output --partial "Unknown subcommand 'unknown'"
}

# Listing operations tests
@test "mc ls: list aliases when no target specified" {
    run mc ls
    assert_success
    assert_output "No hosts configured."
    
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc config host add backup http://backup.example.com access secret
    
    run mc ls
    assert_success
    assert_output --regexp "local"
    assert_output --regexp "backup"
}

@test "mc ls: list buckets for configured alias" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc ls local
    assert_success
    # Should return empty since no buckets created yet
}

@test "mc ls: list buckets with created buckets" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    mc mb local/backup-bucket
    
    run mc ls local
    assert_success
    assert_output --regexp "test-bucket/"
    assert_output --regexp "backup-bucket/"
}

@test "mc ls: list objects in bucket" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Create a test file and upload it
    echo "test content" > test-file.txt
    mc cp test-file.txt local/test-bucket/
    
    run mc ls local/test-bucket
    assert_success
    assert_output --regexp "test-file.txt"
}

@test "mc ls: list objects with prefix" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Create test files with different prefixes
    echo "data1" > data1.txt
    echo "data2" > data2.txt
    echo "info" > info.txt
    
    mc cp data1.txt local/test-bucket/data/data1.txt
    mc cp data2.txt local/test-bucket/data/data2.txt
    mc cp info.txt local/test-bucket/info.txt
    
    run mc ls local/test-bucket/data/
    assert_success
    assert_output --regexp "data1.txt"
    assert_output --regexp "data2.txt"
    refute_output --regexp "info.txt"
}

@test "mc ls: non-existent alias fails" {
    run mc ls nonexistent
    assert_failure
    assert_output --partial "Alias \`nonexistent\` does not exist"
}

@test "mc ls: non-existent bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc ls local/nonexistent
    assert_failure
    assert_output --partial "specified bucket does not exist"
}

# Bucket management tests
@test "mc mb: create bucket successfully" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc mb local/test-bucket
    assert_success
    assert_output --partial "Bucket created successfully"
    assert_output --partial "local/test-bucket"
}

@test "mc mb: create bucket with non-existent alias fails" {
    run mc mb nonexistent/test-bucket
    assert_failure
    assert_output --partial "Alias \`nonexistent\` does not exist"
}

@test "mc mb: create bucket without bucket name fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc mb local
    assert_failure
    assert_output --partial "Missing bucket name"
}

@test "mc mb: create duplicate bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc mb local/test-bucket
    assert_failure
    assert_output --partial "already exists"
}

@test "mc rb: remove empty bucket successfully" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc rb local/test-bucket
    assert_success
    assert_output --partial "Removed"
    assert_output --partial "local/test-bucket"
}

@test "mc rb: remove non-existent bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc rb local/nonexistent
    assert_failure
    assert_output --partial "specified bucket does not exist"
}

@test "mc rb: remove non-empty bucket without force fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Add an object to the bucket
    echo "test" > test.txt
    mc cp test.txt local/test-bucket/
    
    run mc rb local/test-bucket
    assert_failure
    assert_output --partial "bucket is not empty"
    assert_output --partial "Use --force"
}

@test "mc rb: remove non-empty bucket with force succeeds" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Add an object to the bucket
    echo "test" > test.txt
    mc cp test.txt local/test-bucket/
    
    run mc rb --force local/test-bucket
    assert_success
    assert_output --partial "Removed"
}

# File operations tests
@test "mc cp: upload file to bucket" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    echo "Hello, MinIO!" > test-file.txt
    
    run mc cp test-file.txt local/test-bucket/
    assert_success
    assert_output --partial "test-file.txt"
    assert_output --partial "local/test-bucket"
    assert_output --partial "uploaded"
}

@test "mc cp: upload file with custom object name" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    echo "Hello, MinIO!" > test-file.txt
    
    run mc cp test-file.txt local/test-bucket/custom-name.txt
    assert_success
    assert_output --partial "custom-name.txt"
}

@test "mc cp: upload non-existent file fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc cp non-existent-file.txt local/test-bucket/
    assert_failure
    assert_output --partial "File 'non-existent-file.txt' not found"
}

@test "mc cp: upload to non-existent bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    echo "test" > test.txt
    
    run mc cp test.txt local/nonexistent/
    assert_failure
    assert_output --partial "specified bucket does not exist"
}

@test "mc cp: download file from bucket" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Upload a file first
    echo "Hello, MinIO!" > original-file.txt
    mc cp original-file.txt local/test-bucket/test-file.txt
    
    # Download it back
    run mc cp local/test-bucket/test-file.txt downloaded-file.txt
    assert_success
    assert_output --partial "local/test-bucket/test-file.txt"
    assert_output --partial "downloaded-file.txt"
    assert_output --partial "Downloaded"
    
    # Verify content
    [[ -f "downloaded-file.txt" ]]
    [[ "$(cat downloaded-file.txt)" == "Hello, MinIO!" ]]
}

@test "mc cp: download non-existent object fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc cp local/test-bucket/nonexistent.txt download.txt
    assert_failure
    assert_output --partial "Object does not exist"
}

@test "mc cp: invalid source/target combination fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc cp local/bucket1/file.txt local/bucket2/file.txt
    assert_failure
    assert_output --partial "Invalid source/target combination"
}

@test "mc cp: missing source or target fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc cp test-file.txt
    assert_failure
    assert_output --partial "Source and target are required"
}

# Object removal tests
@test "mc rm: remove object successfully" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    echo "test content" > test-file.txt
    mc cp test-file.txt local/test-bucket/
    
    run mc rm local/test-bucket/test-file.txt
    assert_success
    assert_output --partial "Removed"
    assert_output --partial "local/test-bucket/test-file.txt"
}

@test "mc rm: remove non-existent object fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc rm local/test-bucket/nonexistent.txt
    assert_failure
    assert_output --partial "Object does not exist"
}

@test "mc rm: cannot remove bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc rm local/test-bucket
    assert_failure
    assert_output --partial "Cannot remove bucket, use rb command"
}

@test "mc rm: missing target fails" {
    run mc rm
    assert_failure
    assert_output --partial "Missing target"
}

# Disk usage tests
@test "mc du: show bucket size" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc du local/test-bucket
    assert_success
    assert_output --regexp "[0-9]+B.*test-bucket.*objects"
}

@test "mc du: summarize option shows size only" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Upload some files
    echo "content1" > file1.txt
    echo "content2" > file2.txt
    mc cp file1.txt local/test-bucket/
    mc cp file2.txt local/test-bucket/
    
    run mc du --summarize local/test-bucket
    assert_success
    assert_output --regexp "^[0-9]+B$"
}

@test "mc du: non-existent bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc du local/nonexistent
    assert_failure
    assert_output --partial "specified bucket does not exist"
}

# Bucket policy tests
@test "mc anonymous set: set bucket policy to public" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc anonymous set public local/test-bucket
    assert_success
    assert_output --partial "Access permission for"
    assert_output --partial "is set to"
    assert_output --partial "public"
}

@test "mc anonymous set: set bucket policy to download" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc anonymous set download local/test-bucket
    assert_success
    assert_output --partial "is set to"
    assert_output --partial "download"
}

@test "mc anonymous get: get bucket policy" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    mc anonymous set public local/test-bucket
    
    run mc anonymous get local/test-bucket
    assert_success
    assert_output --partial "Access permission for"
    assert_output --partial "is"
    assert_output --partial "public"
}

@test "mc anonymous get: default policy is none" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc anonymous get local/test-bucket
    assert_success
    assert_output --partial "is"
    assert_output --partial "none"
}

@test "mc anonymous: non-existent bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc anonymous set public local/nonexistent
    assert_failure
    assert_output --partial "specified bucket does not exist"
}

@test "mc anonymous: missing arguments fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc anonymous set
    assert_failure
    assert_output --partial "Policy type and target are required"
}

@test "mc anonymous: unknown action fails" {
    run mc anonymous unknown
    assert_failure
    assert_output --partial "Unknown action 'unknown'"
}

# Object stat tests
@test "mc stat: stat object shows metadata" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    echo "Hello, World!" > test-file.txt
    mc cp test-file.txt local/test-bucket/
    
    run mc stat local/test-bucket/test-file.txt
    assert_success
    assert_output --partial "Name      : test-file.txt"
    assert_output --partial "Date      :"
    assert_output --partial "Size      :"
    assert_output --partial "ETag      :"
    assert_output --partial "Type      : file"
}

@test "mc stat: stat bucket shows folder info" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc stat local/test-bucket
    assert_success
    assert_output --partial "Name      : test-bucket/"
    assert_output --partial "Type      : folder"
}

@test "mc stat: stat non-existent object fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mc stat local/test-bucket/nonexistent.txt
    assert_failure
    assert_output --partial "Object does not exist"
}

@test "mc stat: stat non-existent bucket fails" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc stat local/nonexistent
    assert_failure
    assert_output --partial "specified bucket does not exist"
}

# Admin operations tests
@test "mc admin info: show server information" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc admin info local
    assert_success
    assert_output --partial "localhost:9000"
    assert_output --partial "Uptime:"
    assert_output --partial "Version:"
    assert_output --partial "Drives:"
}

@test "mc admin info: non-existent alias fails" {
    run mc admin info nonexistent
    assert_failure
    assert_output --partial "Alias \`nonexistent\` does not exist"
}

@test "mc admin user add: add user successfully" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc admin user add local testuser testpass
    assert_success
    assert_output --partial "Added user"
    assert_output --partial "testuser"
}

@test "mc admin user list: list users" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mc admin user list local
    assert_success
    assert_output --partial "enabled"
    assert_output --partial "minioadmin"
}

@test "mc admin: unknown subcommand fails" {
    run mc admin unknown
    assert_failure
    assert_output --partial "Unknown subcommand 'unknown'"
}

# State persistence tests
@test "minio state persistence across subshells" {
    # Configure alias and create bucket in parent shell
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    mock::minio::save_state
    
    # Verify in subshell
    output=$(
        source "$MOCK_DIR/minio.sh"
        mc ls local
    )
    [[ "$output" =~ "test-bucket" ]]
}

@test "minio state file creation and loading" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    echo "test content" > test.txt
    mc cp test.txt local/test-bucket/
    mock::minio::save_state
    
    # Check state file exists
    [[ -f "$MINIO_MOCK_STATE_DIR/minio-state.sh" ]]
    
    # Reset without saving state, then reload
    mock::minio::reset false
    mock::minio::load_state
    
    run mc ls local
    assert_success
    assert_output --regexp "test-bucket"
    
    run mc ls local/test-bucket
    assert_success
    assert_output --regexp "test.txt"
}

# Test helper functions
@test "mock::minio::reset clears all data" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    echo "test" > test.txt
    mc cp test.txt local/test-bucket/
    mc anonymous set public local/test-bucket
    mock::minio::set_error "connection_failed"
    
    mock::minio::reset
    
    run mc ls local
    assert_failure
    assert_output --partial "Alias \`local\` does not exist"
    
    # Error mode should be cleared
    run mc version
    assert_success
}

@test "mock::minio::assert_bucket_exists" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    run mock::minio::assert_bucket_exists "test-bucket"
    assert_success
    
    run mock::minio::assert_bucket_exists "nonexistent"
    assert_failure
    assert_output --partial "Bucket 'nonexistent' does not exist"
}

@test "mock::minio::assert_object_exists" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    echo "test" > test.txt
    mc cp test.txt local/test-bucket/
    
    run mock::minio::assert_object_exists "test-bucket" "test.txt"
    assert_success
    
    run mock::minio::assert_object_exists "test-bucket" "nonexistent.txt"
    assert_failure
    assert_output --partial "Object 'nonexistent.txt' does not exist"
}

@test "mock::minio::assert_bucket_policy" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    mc anonymous set public local/test-bucket
    
    run mock::minio::assert_bucket_policy "test-bucket" "public"
    assert_success
    
    run mock::minio::assert_bucket_policy "test-bucket" "private"
    assert_failure
    assert_output --partial "Bucket 'test-bucket' policy mismatch"
    assert_output --partial "Expected: 'private'"
    assert_output --partial "Actual: 'public'"
}

@test "mock::minio::assert_alias_configured" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    run mock::minio::assert_alias_configured "local"
    assert_success
    
    run mock::minio::assert_alias_configured "nonexistent"
    assert_failure
    assert_output --partial "Alias 'nonexistent' is not configured"
}

@test "mock::minio::dump_state shows current state" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    echo "test" > test.txt
    mc cp test.txt local/test-bucket/
    mock::minio::set_config "port" "9100"
    
    run mock::minio::dump_state
    assert_success
    assert_output --partial "MinIO Mock State"
    assert_output --partial "port: 9100"
    assert_output --partial "test-bucket:"
    assert_output --partial "local:"
}

# Complex scenario tests
@test "minio: complete workflow - configure, create, upload, download, cleanup" {
    # Configure MinIO client
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    # Create bucket
    mc mb local/workflow-test
    
    # Set bucket policy
    mc anonymous set download local/workflow-test
    
    # Upload files
    echo "File 1 content" > file1.txt
    echo "File 2 content" > file2.txt
    mc cp file1.txt local/workflow-test/
    mc cp file2.txt local/workflow-test/files/file2.txt
    
    # List bucket contents
    run mc ls local/workflow-test
    assert_success
    assert_output --regexp "file1.txt"
    
    # List with prefix
    run mc ls local/workflow-test/files/
    assert_success
    assert_output --regexp "file2.txt"
    
    # Download files
    mc cp local/workflow-test/file1.txt downloaded1.txt
    mc cp local/workflow-test/files/file2.txt downloaded2.txt
    
    # Verify content
    [[ "$(cat downloaded1.txt)" == "File 1 content" ]]
    [[ "$(cat downloaded2.txt)" == "File 2 content" ]]
    
    # Check bucket policy
    run mc anonymous get local/workflow-test
    assert_success
    assert_output --partial "download"
    
    # Check disk usage
    run mc du local/workflow-test
    assert_success
    assert_output --regexp "[0-9]+B.*2 objects"
    
    # Clean up
    mc rm local/workflow-test/file1.txt
    mc rm local/workflow-test/files/file2.txt
    mc rb local/workflow-test
    
    # Verify cleanup
    run mc ls local
    assert_success
    refute_output --regexp "workflow-test"
}

@test "minio: error handling in complex scenarios" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Try operations on non-existent objects/buckets
    run mc cp local/nonexistent/file.txt download.txt
    assert_failure
    
    run mc rm local/test-bucket/nonexistent.txt
    assert_failure
    
    run mc anonymous set public local/nonexistent
    assert_failure
    
    # Try removing non-empty bucket without force
    echo "test" > test.txt
    mc cp test.txt local/test-bucket/
    
    run mc rb local/test-bucket
    assert_failure
    assert_output --partial "not empty"
    
    # Force removal should work
    run mc rb --force local/test-bucket
    assert_success
}

# Edge cases
@test "minio: empty string and special character handling" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    mc mb local/test-bucket
    
    # Empty file
    touch empty-file.txt
    run mc cp empty-file.txt local/test-bucket/
    assert_success
    
    # File with spaces in name
    echo "content" > "file with spaces.txt"
    run mc cp "file with spaces.txt" local/test-bucket/
    assert_success
    
    # Object with special characters in path
    run mc cp empty-file.txt local/test-bucket/path/to/file.txt
    assert_success
    
    # Verify all files exist
    run mc ls local/test-bucket
    assert_success
    assert_output --regexp "empty-file.txt"
    assert_output --regexp "file with spaces.txt"
    assert_output --regexp "file.txt"
}

@test "minio: bucket and object name validation" {
    mc config host add local http://localhost:9000 minioadmin minioadmin
    
    # Valid bucket names
    run mc mb local/valid-bucket-123
    assert_success
    
    run mc mb local/another.bucket
    assert_success
    
    # Bucket operations with various object names
    echo "test" > test.txt
    run mc cp test.txt local/valid-bucket-123/path/to/my-object_123.txt
    assert_success
    
    run mc ls local/valid-bucket-123/path/
    assert_success
    assert_output --regexp "my-object_123.txt"
}