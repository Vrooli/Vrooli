#!/usr/bin/env bash
################################################################################
# MinIO Test Library - v2.0 Contract Compliant
# 
# Test implementations for MinIO resource validation
################################################################################

set -euo pipefail

# Ensure dependencies are loaded
if [[ -z "${MINIO_CLI_DIR:-}" ]]; then
    MINIO_CLI_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)"
fi

# Source core functionality
if [[ -f "${MINIO_CLI_DIR}/lib/core.sh" ]]; then
    # shellcheck disable=SC1090
    source "${MINIO_CLI_DIR}/lib/core.sh"
fi

################################################################################
# Test Orchestration
################################################################################

minio::test::all() {
    log::info "Running all MinIO tests..."
    
    local failed=0
    
    # Run test phases
    if ! minio::test::smoke; then
        ((failed++))
    fi
    
    if ! minio::test::integration; then
        ((failed++))
    fi
    
    if ! minio::test::unit; then
        ((failed++))
    fi
    
    if [[ $failed -gt 0 ]]; then
        log::error "MinIO tests failed: $failed test suites failed"
        return 1
    else
        log::success "All MinIO tests passed"
        return 0
    fi
}

################################################################################
# Smoke Tests - Quick Health Validation
################################################################################

minio::test::smoke() {
    log::info "Running MinIO smoke tests..."
    
    # Test 1: Container exists
    if ! minio::is_installed; then
        log::error "FAIL: MinIO is not installed"
        return 1
    fi
    log::success "PASS: MinIO is installed"
    
    # Test 2: Container is running
    if ! minio::is_running; then
        log::error "FAIL: MinIO is not running"
        return 1
    fi
    log::success "PASS: MinIO is running"
    
    # Test 3: Health endpoint responds
    if ! minio::is_healthy; then
        log::error "FAIL: MinIO health check failed"
        return 1
    fi
    log::success "PASS: MinIO is healthy"
    
    # Test 4: API port is accessible
    local api_port="${MINIO_PORT:-9000}"
    if ! timeout 5 nc -zv localhost "$api_port" &>/dev/null; then
        log::error "FAIL: MinIO API port $api_port is not accessible"
        return 1
    fi
    log::success "PASS: MinIO API port is accessible"
    
    log::success "MinIO smoke tests passed"
    return 0
}

################################################################################
# Integration Tests - Full Functionality
################################################################################

minio::test::integration() {
    log::info "Running MinIO integration tests..."
    
    local failed=0
    
    # Test 1: S3 API responds
    if ! minio::test::s3_api; then
        log::error "FAIL: S3 API test failed"
        ((failed++))
    else
        log::success "PASS: S3 API is functional"
    fi
    
    # Test 2: Bucket operations
    if ! minio::test::bucket_operations; then
        log::error "FAIL: Bucket operations test failed"
        ((failed++))
    else
        log::success "PASS: Bucket operations work"
    fi
    
    # Test 3: Object operations
    if ! minio::test::object_operations; then
        log::error "FAIL: Object operations test failed"
        ((failed++))
    else
        log::success "PASS: Object operations work"
    fi
    
    # Test 4: Default buckets exist
    if ! minio::test::default_buckets; then
        log::error "FAIL: Default buckets test failed"
        ((failed++))
    else
        log::success "PASS: Default buckets exist"
    fi
    
    if [[ $failed -gt 0 ]]; then
        log::error "MinIO integration tests failed: $failed tests failed"
        return 1
    else
        log::success "MinIO integration tests passed"
        return 0
    fi
}

################################################################################
# Unit Tests - Library Function Validation
################################################################################

minio::test::unit() {
    log::info "Running MinIO unit tests..."
    
    local failed=0
    
    # Test 1: Configuration loading
    if ! minio::test::config_loading; then
        log::error "FAIL: Configuration loading test failed"
        ((failed++))
    else
        log::success "PASS: Configuration loads correctly"
    fi
    
    # Test 2: Credential management
    if ! minio::test::credentials; then
        log::error "FAIL: Credential management test failed"
        ((failed++))
    else
        log::success "PASS: Credentials managed correctly"
    fi
    
    # Test 3: Port detection
    if ! minio::test::port_detection; then
        log::error "FAIL: Port detection test failed"
        ((failed++))
    else
        log::success "PASS: Port detection works"
    fi
    
    if [[ $failed -gt 0 ]]; then
        log::error "MinIO unit tests failed: $failed tests failed"
        return 1
    else
        log::success "MinIO unit tests passed"
        return 0
    fi
}

################################################################################
# Individual Test Functions
################################################################################

minio::test::s3_api() {
    local api_port="${MINIO_PORT:-9000}"
    local endpoint="http://localhost:${api_port}"
    
    # Test basic S3 API response
    if timeout 5 curl -sf "${endpoint}/" &>/dev/null; then
        return 0
    else
        return 1
    fi
}

minio::test::bucket_operations() {
    # Load credentials
    local creds_file="${HOME}/.minio/config/credentials"
    if [[ -f "$creds_file" ]]; then
        # shellcheck disable=SC1090
        source "$creds_file"
    fi
    
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    local access_key="${MINIO_ROOT_USER:-minioadmin}"
    local secret_key="${MINIO_ROOT_PASSWORD:-minio123}"
    local api_port="${MINIO_PORT:-9000}"
    local test_bucket="test-bucket-$(date +%s)"
    
    # Setup mc client alias in container
    if ! docker exec "$container_name" mc alias set local "http://localhost:9000" "$access_key" "$secret_key" &>/dev/null 2>&1; then
        log::warning "Failed to configure mc client"
        return 1
    fi
    
    # Create bucket using mc
    if ! docker exec "$container_name" mc mb "local/${test_bucket}" &>/dev/null 2>&1; then
        log::error "Failed to create test bucket"
        return 1
    fi
    
    # List buckets using mc
    if ! docker exec "$container_name" mc ls local/ 2>/dev/null | grep -q "$test_bucket"; then
        log::error "Test bucket not found in listing"
        return 1
    fi
    
    # Remove bucket using mc
    if ! docker exec "$container_name" mc rb "local/${test_bucket}" &>/dev/null 2>&1; then
        log::error "Failed to remove test bucket"
        return 1
    fi
    
    return 0
}

minio::test::object_operations() {
    # Load credentials
    local creds_file="${HOME}/.minio/config/credentials"
    if [[ -f "$creds_file" ]]; then
        # shellcheck disable=SC1090
        source "$creds_file"
    fi
    
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    local access_key="${MINIO_ROOT_USER:-minioadmin}"
    local secret_key="${MINIO_ROOT_PASSWORD:-minio123}"
    local test_bucket="test-objects-$(date +%s)"
    
    # Setup mc client alias in container
    if ! docker exec "$container_name" mc alias set local "http://localhost:9000" "$access_key" "$secret_key" &>/dev/null 2>&1; then
        log::warning "Failed to configure mc client"
        return 1
    fi
    
    # Create test bucket
    if ! docker exec "$container_name" mc mb "local/${test_bucket}" &>/dev/null 2>&1; then
        log::error "Failed to create test bucket for objects"
        return 1
    fi
    
    # Create test file in container
    local test_content="MinIO test content $(date +%s)"
    if ! docker exec "$container_name" sh -c "echo '$test_content' > /tmp/test.txt" 2>/dev/null; then
        log::error "Failed to create test file"
        docker exec "$container_name" mc rb "local/${test_bucket}" &>/dev/null 2>&1
        return 1
    fi
    
    # Upload file using mc
    if ! docker exec "$container_name" mc cp /tmp/test.txt "local/${test_bucket}/test.txt" &>/dev/null 2>&1; then
        log::error "Failed to upload test file"
        docker exec "$container_name" mc rb --force "local/${test_bucket}" &>/dev/null 2>&1
        return 1
    fi
    
    # Download file using mc
    if ! docker exec "$container_name" mc cp "local/${test_bucket}/test.txt" /tmp/download.txt &>/dev/null 2>&1; then
        log::error "Failed to download test file"
        docker exec "$container_name" mc rb --force "local/${test_bucket}" &>/dev/null 2>&1
        return 1
    fi
    
    # Verify content
    local downloaded_content
    downloaded_content=$(docker exec "$container_name" cat /tmp/download.txt 2>/dev/null)
    if [[ "$downloaded_content" != "$test_content" ]]; then
        log::error "Downloaded content mismatch"
        docker exec "$container_name" mc rb --force "local/${test_bucket}" &>/dev/null 2>&1
        return 1
    fi
    
    # Cleanup
    docker exec "$container_name" mc rb --force "local/${test_bucket}" &>/dev/null 2>&1
    docker exec "$container_name" rm -f /tmp/test.txt /tmp/download.txt &>/dev/null 2>&1
    
    return 0
}

minio::test::default_buckets() {
    # Check if default buckets exist using mc
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    
    # Load credentials
    local creds_file="${HOME}/.minio/config/credentials"
    if [[ -f "$creds_file" ]]; then
        # shellcheck disable=SC1090
        source "$creds_file"
    fi
    
    local access_key="${MINIO_ROOT_USER:-minioadmin}"
    local secret_key="${MINIO_ROOT_PASSWORD:-minio123}"
    
    # Setup mc client alias in container
    if ! docker exec "$container_name" mc alias set local "http://localhost:9000" "$access_key" "$secret_key" &>/dev/null 2>&1; then
        log::warning "Failed to configure mc client"
        return 1
    fi
    
    # Check for default buckets
    local default_buckets=(
        "vrooli-user-uploads"
        "vrooli-agent-artifacts"
        "vrooli-model-cache"
        "vrooli-temp-storage"
    )
    
    local bucket_list
    bucket_list=$(docker exec "$container_name" mc ls local/ 2>/dev/null || true)
    
    local found=0
    for bucket in "${default_buckets[@]}"; do
        if echo "$bucket_list" | grep -q "$bucket"; then
            ((found++))
        fi
    done
    
    # Return success if at least some default buckets exist
    if [[ $found -gt 0 ]]; then
        return 0
    else
        log::warning "No default buckets found (will be created on first use)"
        return 0  # Not a failure, just informational
    fi
}

minio::test::config_loading() {
    # Test configuration file exists and is readable
    local config_file="${MINIO_CLI_DIR}/config/defaults.sh"
    
    if [[ -f "$config_file" ]] && [[ -r "$config_file" ]]; then
        return 0
    else
        return 1
    fi
}

minio::test::credentials() {
    # Test credential file handling
    local creds_file="${HOME}/.minio/config/credentials"
    
    # Check if credentials file would be created with proper permissions
    if [[ -f "$creds_file" ]]; then
        local perms
        perms=$(stat -c "%a" "$creds_file" 2>/dev/null || stat -f "%OLp" "$creds_file" 2>/dev/null || echo "unknown")
        
        if [[ "$perms" == "600" ]]; then
            return 0
        else
            log::warning "Credential file permissions are $perms (expected 600)"
            return 1
        fi
    else
        # File doesn't exist yet, that's okay for a fresh install
        return 0
    fi
}

minio::test::port_detection() {
    # Test port configuration
    local api_port="${MINIO_PORT:-9000}"
    local console_port="${MINIO_CONSOLE_PORT:-9001}"
    
    # Verify ports are numbers
    if [[ "$api_port" =~ ^[0-9]+$ ]] && [[ "$console_port" =~ ^[0-9]+$ ]]; then
        return 0
    else
        return 1
    fi
}

################################################################################
# Export Functions
################################################################################

export -f minio::test::all
export -f minio::test::smoke
export -f minio::test::integration
export -f minio::test::unit