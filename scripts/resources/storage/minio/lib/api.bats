#!/usr/bin/env bats
# Tests for MinIO API functions (lib/api.sh)

# Load Vrooli test infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"

# Expensive setup operations run once per file
setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "minio"
    
    # Load MinIO specific configuration once per file
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    MINIO_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and API functions once
    source "${MINIO_DIR}/config/defaults.sh"
    source "${MINIO_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/api.sh"
}

# Lightweight per-test setup
setup() {
    # Setup standard Vrooli mocks
    vrooli_auto_setup
    
    # Set test environment variables (lightweight per-test)
    export MINIO_CUSTOM_PORT="9001"
    export MINIO_CONTAINER_NAME="minio-test"
    export MINIO_BASE_URL="http://localhost:9001"
    export MINIO_ACCESS_KEY="testkey"
    export MINIO_SECRET_KEY="testsecret"
    export MINIO_API_TIMEOUT="30"
    
    # Export config functions
    minio::export_config
    minio::export_messages
    
    # Mock health check function for API tests
    minio::is_healthy() {
        return 0  # Always healthy for API tests
    }
    export -f minio::is_healthy
    
    # Mock log functions
    log::header() { echo "=== $* ==="; }
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::warning() { echo "[WARNING] $*" >&2; }
    export -f log::header log::info log::error log::success log::warning
}

# BATS teardown function - runs after each test
teardown() {
    vrooli_cleanup_test
}

# ============================================================================
# API Health Check Tests
# ============================================================================

@test "minio::api::test validates MinIO API connectivity" {
    # Mock successful mc command
    mc() {
        if [[ "$*" =~ "admin info" ]]; then
            echo "●  localhost:9001"
            echo "   Uptime: 1 day"
            echo "   Version: 2023-12-07T04:16:00Z"
            return 0
        fi
        return 1
    }
    export -f mc
    
    run minio::api::test
    [ "$status" -eq 0 ]
    [[ "$output" =~ "API connectivity test passed" ]]
}

@test "minio::api::test fails when service is unhealthy" {
    # Override health check to fail
    minio::is_healthy() {
        return 1
    }
    
    run minio::api::test
    [ "$status" -eq 1 ]
    [[ "$output" =~ "MinIO is not running or healthy" ]]
}

# ============================================================================
# Bucket Management Tests
# ============================================================================

@test "minio::api::list_buckets displays bucket information" {
    # Mock successful mc ls command
    mc() {
        if [[ "$*" =~ "ls" ]]; then
            cat << 'EOF'
[2023-12-01 10:00:00 UTC]     0B test-bucket/
[2023-12-01 11:00:00 UTC]     0B backup-bucket/
[2023-12-01 12:00:00 UTC]     0B data-bucket/
EOF
            return 0
        fi
        return 1
    }
    export -f mc
    
    run minio::api::list_buckets
    [ "$status" -eq 0 ]
    [[ "$output" =~ "test-bucket" ]]
    [[ "$output" =~ "backup-bucket" ]]
    [[ "$output" =~ "data-bucket" ]]
}

@test "minio::api::list_buckets handles no buckets" {
    # Mock empty mc ls response
    mc() {
        if [[ "$*" =~ "ls" ]]; then
            echo ""
            return 0
        fi
        return 1
    }
    export -f mc
    
    run minio::api::list_buckets
    [ "$status" -eq 0 ]
    [[ "$output" =~ "No buckets found" ]]
}

@test "minio::api::create_bucket creates new bucket successfully" {
    # Mock successful mc mb command
    mc() {
        if [[ "$*" =~ "mb.*test-bucket" ]]; then
            echo "Bucket created successfully 'minio/test-bucket'."
            return 0
        fi
        return 1
    }
    export -f mc
    
    run minio::api::create_bucket "test-bucket"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Bucket created successfully" ]]
}

@test "minio::api::create_bucket fails with missing bucket name" {
    run minio::api::create_bucket ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Bucket name is required" ]]
}

@test "minio::api::create_bucket handles existing bucket" {
    # Mock mc mb command failure for existing bucket
    mc() {
        if [[ "$*" =~ "mb.*existing-bucket" ]]; then
            echo "mc: <ERROR> Unable to make bucket 'minio/existing-bucket'. Your previous request to create the named bucket succeeded and you already own it."
            return 1
        fi
        return 1
    }
    export -f mc
    
    run minio::api::create_bucket "existing-bucket"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "already own it" ]]
}

# ============================================================================
# File Operations Tests
# ============================================================================

@test "minio::api::upload_file uploads file successfully" {
    # Create test file
    local test_file="/tmp/test-upload.txt"
    echo "test content" > "$test_file"
    
    # Mock successful mc cp command
    mc() {
        if [[ "$*" =~ "cp.*test-upload.txt.*test-bucket" ]]; then
            echo "/tmp/test-upload.txt -> minio/test-bucket/test-upload.txt"
            return 0
        fi
        return 1
    }
    export -f mc
    
    run minio::api::upload_file "$test_file" "test-bucket"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Upload completed successfully" ]]
}

@test "minio::api::upload_file validates parameters" {
    run minio::api::upload_file "" "test-bucket"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "File path is required" ]]
    
    run minio::api::upload_file "/tmp/test.txt" ""
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Bucket name is required" ]]
}

@test "minio::api::download_file downloads file successfully" {
    # Mock successful mc cp command
    mc() {
        if [[ "$*" =~ "cp.*test-bucket/test-file.txt" ]]; then
            echo "minio/test-bucket/test-file.txt -> /tmp/downloaded-file.txt"
            return 0
        fi
        return 1
    }
    export -f mc
    
    run minio::api::download_file "test-bucket" "test-file.txt" "/tmp/downloaded-file.txt"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Download completed successfully" ]]
}
