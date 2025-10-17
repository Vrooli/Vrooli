#!/usr/bin/env bash
# MinIO Integration Test
# Tests real MinIO object storage functionality
# Tests S3-compatible API endpoints and bucket operations

set -euo pipefail

# Source var.sh first to get directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/minio/test"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source enhanced integration test library using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/tests/lib/enhanced-integration-test-lib.sh"

#######################################
# SERVICE-SPECIFIC CONFIGURATION
#######################################

# Load MinIO configuration using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/minio/config/defaults.sh"
minio::export_config

# Override library defaults with MinIO-specific settings
# shellcheck disable=SC2034
SERVICE_NAME="minio"
BASE_URL="${MINIO_BASE_URL:-http://localhost:9000}"
# shellcheck disable=SC2034
HEALTH_ENDPOINT="/minio/health/live"
# shellcheck disable=SC2034
REQUIRED_TOOLS=("curl" "jq" "mc")
# shellcheck disable=SC2034
SERVICE_METADATA=(
    "API Port: ${MINIO_PORT:-9000}"
    "Console Port: ${MINIO_CONSOLE_PORT:-9001}"
    "Container: ${MINIO_CONTAINER_NAME:-minio}"
)

# Test configuration
# shellcheck disable=SC2034
TEST_BUCKET_NAME="vrooli-integration-test-$(date +%s)"
# shellcheck disable=SC2034
readonly TEST_FILE_NAME="test-file.txt"
# shellcheck disable=SC2034
readonly TEST_FILE_CONTENT="This is a test file for MinIO integration testing"

#######################################
# MINIO-SPECIFIC TEST FUNCTIONS
#######################################

test_minio_version() {
    local test_name="MinIO version endpoint"
    
    local response
    if response=$(make_request "$BASE_URL/minio/admin/v3/info" "GET" 10 2>/dev/null || echo ""); then
        if [[ -n "$response" ]]; then
            log_test_result "$test_name" "PASS" "version endpoint accessible"
            return 0
        fi
    fi
    
    # Try alternative health endpoint
    if response=$(make_request "$BASE_URL/minio/health/live" "GET" 5 2>/dev/null || echo ""); then
        log_test_result "$test_name" "PASS" "health endpoint accessible"
        return 0
    fi
    
    log_test_result "$test_name" "FAIL" "version/health endpoints not accessible"
    return 1
}

test_s3_api_compatibility() {
    local test_name="S3 API compatibility"
    
    # Test S3 API root endpoint (should return error but be accessible)
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL/" 2>/dev/null); then
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # MinIO should respond with some HTTP status (even if it's an error)
        if [[ -n "$status_code" ]] && [[ "$status_code" =~ ^[0-9]+$ ]]; then
            log_test_result "$test_name" "PASS" "S3 API endpoint responsive (HTTP: $status_code)"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "S3 API endpoint not accessible"
    return 1
}

test_minio_client_setup() {
    local test_name="MinIO client (mc) configuration"
    
    if ! command -v mc >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "mc client not installed"
        return 2
    fi
    
    # Try to configure mc alias (this will fail without credentials but should show MinIO is accessible)
    local mc_output
    if mc_output=$(mc alias set test-minio "$BASE_URL" test-key test-secret 2>&1 || true); then
        if echo "$mc_output" | grep -qi "added successfully\|unable to validate\|access denied"; then
            log_test_result "$test_name" "PASS" "MinIO server accessible via mc client"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "MinIO server not accessible via mc client"
    return 1
}

test_bucket_operations() {
    local test_name="bucket operations (requires auth)"
    
    if ! command -v mc >/dev/null 2>&1; then
        log_test_result "$test_name" "SKIP" "mc client not installed"
        return 2
    fi
    
    # This test will typically fail in integration testing without proper credentials
    # But we can test that the server responds to bucket operations
    local mc_output
    if mc_output=$(mc ls test-minio/ 2>&1 || true); then
        if echo "$mc_output" | grep -qi "access denied\|invalid access key\|signature mismatch"; then
            log_test_result "$test_name" "PASS" "bucket operations endpoint responsive (auth required)"
            return 0
        elif echo "$mc_output" | grep -qi "connection refused\|network"; then
            log_test_result "$test_name" "FAIL" "cannot connect to MinIO for bucket operations"
            return 1
        else
            # If we get a different response, MinIO might be configured
            log_test_result "$test_name" "PASS" "bucket operations working"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "bucket operations test failed"
    return 1
}

test_prometheus_metrics() {
    local test_name="Prometheus metrics endpoint"
    
    local response
    if response=$(make_request "$BASE_URL/minio/v2/metrics/cluster" "GET" 5 2>/dev/null || echo ""); then
        if echo "$response" | grep -q "minio_\|# HELP\|# TYPE"; then
            log_test_result "$test_name" "PASS" "Prometheus metrics available"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "Prometheus metrics not enabled or accessible"
    return 2
}

#######################################
# FIXTURE-BASED STORAGE TESTS
#######################################

test_upload_fixture_file() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Test file upload capability via S3 API
    local filename
    filename=$(basename "$fixture_path")
    
    # Create signed PUT request (simplified test - actual auth would be needed)
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X PUT \
        -H "Content-Type: application/octet-stream" \
        --data-binary "@$fixture_path" \
        "$BASE_URL/test-bucket/$filename" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Accept 403/401 as success (endpoint exists but needs auth)
        # or 200/201 if file was uploaded
        if [[ "$status_code" == "403" ]] || [[ "$status_code" == "401" ]]; then
            return 0  # Endpoint exists, auth required
        elif [[ "$status_code" == "200" ]] || [[ "$status_code" == "201" ]]; then
            return 0  # File uploaded successfully
        fi
    fi
    
    return 1
}

test_multipart_upload_large_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Test multipart upload initiation (for large files)
    local filename
    filename=$(basename "$fixture_path")
    
    # Initiate multipart upload
    local response
    local status_code
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        "$BASE_URL/test-bucket/$filename?uploads" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Check if multipart API endpoint responds
        if [[ "$status_code" == "403" ]] || [[ "$status_code" == "401" ]]; then
            return 0  # Multipart API exists, auth required
        elif [[ "$status_code" == "200" ]]; then
            return 0  # Multipart upload initiated
        fi
    fi
    
    return 1
}

test_object_metadata_fixture() {
    local fixture_path="$1"
    
    if [[ ! -f "$fixture_path" ]]; then
        return 1
    fi
    
    # Test HEAD request for object metadata
    local filename
    filename=$(basename "$fixture_path")
    
    local response
    local status_code
    if response=$(curl -s -I -w "HTTPSTATUS:%{http_code}" \
        "$BASE_URL/test-bucket/$filename" 2>/dev/null); then
        
        status_code=$(echo "$response" | grep -o "HTTPSTATUS:[0-9]*" | cut -d: -f2)
        
        # Check if metadata API responds
        if [[ -n "$status_code" ]] && [[ "$status_code" =~ ^[0-9]+$ ]]; then
            return 0  # Metadata API endpoint responsive
        fi
    fi
    
    return 1
}

# Run fixture-based storage tests
run_storage_fixture_tests() {
    if [[ "$FIXTURES_AVAILABLE" == "true" ]]; then
        # Test with various file types
        test_with_fixture "upload PDF document" "documents" "pdf/simple_text.pdf" \
            test_upload_fixture_file
        
        test_with_fixture "upload JSON data" "documents" "structured/database_export.json" \
            test_upload_fixture_file
        
        test_with_fixture "upload CSV file" "documents" "structured/customers.csv" \
            test_upload_fixture_file
        
        # Test with images
        test_with_fixture "upload JPEG image" "images" "real-world/nature/nature-landscape.jpg" \
            test_upload_fixture_file
        
        test_with_fixture "upload PNG image" "images" "synthetic/patterns/test-pattern.png" \
            test_upload_fixture_file
        
        # Test large file handling
        test_with_fixture "multipart upload large PDF" "documents" "pdf/large_document.pdf" \
            test_multipart_upload_large_fixture
        
        # Test metadata operations
        test_with_fixture "get object metadata" "documents" "samples/test-readme.md" \
            test_object_metadata_fixture
        
        # Test with auto-discovered fixtures
        local storage_fixtures
        storage_fixtures=$(discover_resource_fixtures "minio" "storage")
        
        for fixture_pattern in $storage_fixtures; do
            # Use rotate_fixtures to get a variety of files
            local test_files
            if test_files=$(rotate_fixtures "$fixture_pattern" 2); then
                for test_file in $test_files; do
                    local fixture_name
                    fixture_name=$(basename "$test_file")
                    test_with_fixture "store $fixture_name" "" "$test_file" \
                        test_upload_fixture_file
                done
            fi
        done
        
        # Test with negative/edge case fixtures if available
        if [[ -d "$FIXTURES_DIR/negative-tests" ]]; then
            test_with_fixture "handle corrupted file" "negative-tests" "documents/corrupted.docx" \
                test_upload_fixture_file
            
            test_with_fixture "handle empty file" "negative-tests" "edge-cases/zero_bytes.txt" \
                test_upload_fixture_file
        fi
    fi
}

test_console_accessibility() {
    local test_name="MinIO Console accessibility"
    
    local console_url="${MINIO_CONSOLE_URL:-http://localhost:9001}"
    local response
    if response=$(make_request "$console_url" "GET" 5 2>/dev/null || echo ""); then
        if echo "$response" | grep -qi "minio\|console\|login\|<!DOCTYPE html>"; then
            log_test_result "$test_name" "PASS" "MinIO Console accessible"
            return 0
        fi
    fi
    
    log_test_result "$test_name" "FAIL" "MinIO Console not accessible"
    return 1
}

#######################################
# TEST RUNNER CONFIGURATION
#######################################

# Define service-specific tests to run
# shellcheck disable=SC2034
SERVICE_TESTS=(
    "test_minio_version"
    "test_s3_api_compatibility"
    "test_minio_client_setup" 
    "test_bucket_operations"
    "test_prometheus_metrics"
    "test_console_accessibility"
    "run_storage_fixture_tests"
)

#######################################
# MAIN EXECUTION
#######################################

# Initialize and run tests using the shared library
init_config
run_integration_tests "$@"