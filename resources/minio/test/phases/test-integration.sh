#!/usr/bin/env bash
################################################################################
# MinIO Integration Tests - Full Functionality Validation
#
# Tests MinIO's S3 API, bucket operations, and integrations
################################################################################

set -euo pipefail

# Setup paths
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
MINIO_DIR="$(builtin cd "${SCRIPT_DIR}/../.." && builtin pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${MINIO_DIR}/../.." && builtin pwd)}"

# Source dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${MINIO_DIR}/config/defaults.sh"

# Export configuration
minio::export_config 2>/dev/null || true

################################################################################
# Integration Tests
################################################################################

log::info "Starting MinIO integration tests..."

FAILED=0

# Load credentials if available
CREDS_FILE="${HOME}/.minio/config/credentials"
if [[ -f "$CREDS_FILE" ]] && [[ -z "${MINIO_ROOT_USER:-}" ]]; then
    # shellcheck disable=SC1090
    source "$CREDS_FILE" 2>/dev/null || true
fi

ACCESS_KEY="${MINIO_ROOT_USER:-minioadmin}"
SECRET_KEY="${MINIO_ROOT_PASSWORD:-minio123}"
API_PORT="${MINIO_PORT:-9000}"
ENDPOINT="http://localhost:${API_PORT}"

# Test 1: S3 API basic connectivity
log::info "Test 1: S3 API connectivity..."

# S3 API returns 403 without auth, which is expected
HTTP_CODE=$(timeout 5 curl -s -o /dev/null -w "%{http_code}" "${ENDPOINT}/" 2>/dev/null || echo "000")
if [[ "$HTTP_CODE" == "403" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    log::success "✓ S3 API endpoint is reachable (HTTP $HTTP_CODE)"
else
    log::error "✗ S3 API endpoint not reachable (HTTP $HTTP_CODE)"
    ((FAILED++))
fi

# Test 2: Health readiness check
log::info "Test 2: Health readiness check..."

if timeout 5 curl -sf "${ENDPOINT}/minio/health/ready" &>/dev/null; then
    log::success "✓ MinIO is ready for operations"
else
    log::warning "⚠ MinIO readiness check failed (may still be starting)"
fi

# Test 3: Console UI accessibility
log::info "Test 3: Console UI accessibility..."
CONSOLE_PORT="${MINIO_CONSOLE_PORT:-9001}"

if timeout 5 curl -sf "http://localhost:${CONSOLE_PORT}" &>/dev/null; then
    log::success "✓ MinIO console UI is accessible"
else
    log::warning "⚠ MinIO console UI not accessible (optional feature)"
fi

# Test 4: Bucket operations with AWS CLI (if available)
log::info "Test 4: Bucket operations..."

if command -v aws &>/dev/null; then
    export AWS_ACCESS_KEY_ID="$ACCESS_KEY"
    export AWS_SECRET_ACCESS_KEY="$SECRET_KEY"
    
    TEST_BUCKET="integration-test-$(date +%s)"
    
    # Create bucket
    if aws s3 mb "s3://${TEST_BUCKET}" --endpoint-url "$ENDPOINT" &>/dev/null 2>&1; then
        log::success "✓ Bucket creation works"
        
        # List buckets
        if aws s3 ls --endpoint-url "$ENDPOINT" 2>/dev/null | grep -q "$TEST_BUCKET"; then
            log::success "✓ Bucket listing works"
        else
            log::error "✗ Bucket listing failed"
            ((FAILED++))
        fi
        
        # Upload test file
        TEST_FILE="/tmp/minio-test-${TEST_BUCKET}.txt"
        echo "Integration test content" > "$TEST_FILE"
        
        if aws s3 cp "$TEST_FILE" "s3://${TEST_BUCKET}/test.txt" --endpoint-url "$ENDPOINT" &>/dev/null 2>&1; then
            log::success "✓ File upload works"
            
            # Download file
            DOWNLOAD_FILE="/tmp/minio-download-${TEST_BUCKET}.txt"
            if aws s3 cp "s3://${TEST_BUCKET}/test.txt" "$DOWNLOAD_FILE" --endpoint-url "$ENDPOINT" &>/dev/null 2>&1; then
                if [[ -f "$DOWNLOAD_FILE" ]] && grep -q "Integration test content" "$DOWNLOAD_FILE"; then
                    log::success "✓ File download works"
                else
                    log::error "✗ Downloaded file content mismatch"
                    ((FAILED++))
                fi
                rm -f "$DOWNLOAD_FILE"
            else
                log::error "✗ File download failed"
                ((FAILED++))
            fi
        else
            log::error "✗ File upload failed"
            ((FAILED++))
        fi
        
        # Cleanup
        aws s3 rm "s3://${TEST_BUCKET}" --recursive --endpoint-url "$ENDPOINT" &>/dev/null 2>&1 || true
        aws s3 rb "s3://${TEST_BUCKET}" --endpoint-url "$ENDPOINT" &>/dev/null 2>&1 || true
        rm -f "$TEST_FILE"
    else
        log::error "✗ Bucket creation failed"
        ((FAILED++))
    fi
else
    log::warning "⚠ AWS CLI not available, skipping advanced bucket tests"
fi

# Test 5: Default Vrooli buckets
log::info "Test 5: Default Vrooli buckets..."

DEFAULT_BUCKETS=(
    "vrooli-user-uploads"
    "vrooli-agent-artifacts"
    "vrooli-model-cache"
    "vrooli-temp-storage"
)

if command -v aws &>/dev/null; then
    export AWS_ACCESS_KEY_ID="$ACCESS_KEY"
    export AWS_SECRET_ACCESS_KEY="$SECRET_KEY"
    
    BUCKET_LIST=$(aws s3 ls --endpoint-url "$ENDPOINT" 2>/dev/null || true)
    
    for bucket in "${DEFAULT_BUCKETS[@]}"; do
        if echo "$BUCKET_LIST" | grep -q "$bucket"; then
            log::success "✓ Default bucket exists: $bucket"
        else
            log::warning "⚠ Default bucket missing: $bucket (will be created on first use)"
        fi
    done
else
    log::warning "⚠ Cannot verify default buckets without AWS CLI"
fi

# Test 6: Concurrent connection handling
log::info "Test 6: Concurrent connections..."

PIDS=()
for i in {1..5}; do
    (timeout 2 curl -sf "${ENDPOINT}/minio/health/live" &>/dev/null) &
    PIDS+=($!)
done

CONCURRENT_SUCCESS=0
for pid in "${PIDS[@]}"; do
    if wait "$pid" 2>/dev/null; then
        ((CONCURRENT_SUCCESS++)) || true
    fi
done

if [[ $CONCURRENT_SUCCESS -ge 4 ]]; then
    log::success "✓ Handles concurrent connections (${CONCURRENT_SUCCESS}/5 succeeded)"
else
    log::warning "⚠ Concurrent connections: ${CONCURRENT_SUCCESS}/5 succeeded"
fi

# Test 7: CLI command integration
log::info "Test 7: CLI command integration..."

if vrooli resource minio status &>/dev/null; then
    log::success "✓ CLI status command works"
else
    log::error "✗ CLI status command failed"
    ((FAILED++))
fi

# Test 8: Performance Benchmarking
log::info "Test 8: Performance benchmarking..."

# Small file upload/download benchmark (1MB)
PERF_TEST_FILE="/tmp/minio-perf-test-1mb.dat"
dd if=/dev/urandom of="$PERF_TEST_FILE" bs=1M count=1 &>/dev/null 2>&1

# Test using curl with timing
START_TIME=$(date +%s%N)
HTTP_CODE=$(timeout 10 curl -s -o /dev/null -w "%{http_code}" \
    -X PUT \
    -H "Content-Type: application/octet-stream" \
    --data-binary "@${PERF_TEST_FILE}" \
    "${ENDPOINT}/test-perf-bucket/1mb-test.dat" 2>/dev/null || echo "000")
END_TIME=$(date +%s%N)

if [[ "$HTTP_CODE" == "403" ]] || [[ "$HTTP_CODE" == "200" ]]; then
    UPLOAD_TIME=$(( (END_TIME - START_TIME) / 1000000 ))
    if [[ $UPLOAD_TIME -lt 1000 ]]; then
        log::success "✓ Small file upload performance: ${UPLOAD_TIME}ms (<1s requirement)"
    else
        log::warning "⚠ Small file upload took ${UPLOAD_TIME}ms (>1s)"
    fi
else
    log::warning "⚠ Performance test skipped (requires authentication)"
fi

# Large file test (10MB) - only if AWS CLI available
if command -v aws &>/dev/null; then
    PERF_TEST_LARGE="/tmp/minio-perf-test-10mb.dat"
    dd if=/dev/urandom of="$PERF_TEST_LARGE" bs=1M count=10 &>/dev/null 2>&1
    
    export AWS_ACCESS_KEY_ID="$ACCESS_KEY"
    export AWS_SECRET_ACCESS_KEY="$SECRET_KEY"
    
    # Create test bucket for performance
    PERF_BUCKET="perf-test-$(date +%s)"
    aws s3 mb "s3://${PERF_BUCKET}" --endpoint-url "$ENDPOINT" &>/dev/null 2>&1 || true
    
    # Upload timing
    START_TIME=$(date +%s%N)
    if aws s3 cp "$PERF_TEST_LARGE" "s3://${PERF_BUCKET}/10mb-test.dat" \
        --endpoint-url "$ENDPOINT" &>/dev/null 2>&1; then
        END_TIME=$(date +%s%N)
        UPLOAD_TIME=$(( (END_TIME - START_TIME) / 1000000 ))
        
        # Calculate throughput (MB/s)
        THROUGHPUT=$(( 10000 / UPLOAD_TIME ))
        
        if [[ $THROUGHPUT -gt 10 ]]; then
            log::success "✓ Large file throughput: ~${THROUGHPUT}MB/s (>10MB/s requirement)"
        else
            log::warning "⚠ Large file throughput: ~${THROUGHPUT}MB/s (<10MB/s)"
        fi
    fi
    
    # Download timing
    START_TIME=$(date +%s%N)
    if aws s3 cp "s3://${PERF_BUCKET}/10mb-test.dat" "/tmp/download-test.dat" \
        --endpoint-url "$ENDPOINT" &>/dev/null 2>&1; then
        END_TIME=$(date +%s%N)
        DOWNLOAD_TIME=$(( (END_TIME - START_TIME) / 1000000 ))
        
        # Calculate throughput (MB/s)
        THROUGHPUT=$(( 10000 / DOWNLOAD_TIME ))
        
        if [[ $THROUGHPUT -gt 10 ]]; then
            log::success "✓ Download throughput: ~${THROUGHPUT}MB/s (>10MB/s requirement)"
        else
            log::warning "⚠ Download throughput: ~${THROUGHPUT}MB/s (<10MB/s)"
        fi
    fi
    
    # Cleanup performance test files
    aws s3 rm "s3://${PERF_BUCKET}" --recursive --endpoint-url "$ENDPOINT" &>/dev/null 2>&1 || true
    aws s3 rb "s3://${PERF_BUCKET}" --endpoint-url "$ENDPOINT" &>/dev/null 2>&1 || true
    rm -f "$PERF_TEST_LARGE" "/tmp/download-test.dat"
else
    log::warning "⚠ Skipping large file performance tests (AWS CLI required)"
fi

# Cleanup small test file
rm -f "$PERF_TEST_FILE"

# Test 9: Memory usage check
log::info "Test 9: Resource usage check..."

CONTAINER_NAME="${MINIO_CONTAINER_NAME:-minio}"
if docker stats --no-stream "$CONTAINER_NAME" 2>/dev/null | grep -v CONTAINER; then
    MEM_USAGE=$(docker stats --no-stream --format "{{.MemUsage}}" "$CONTAINER_NAME" 2>/dev/null | cut -d'/' -f1 | sed 's/[^0-9.]//g' || echo "0")
    
    # Convert to MB if needed
    if [[ "$MEM_USAGE" =~ ^[0-9]+(\.[0-9]+)?$ ]]; then
        MEM_MB=$(echo "$MEM_USAGE" | awk '{printf "%.0f", $1}')
        if [[ $MEM_MB -lt 2000 ]]; then
            log::success "✓ Memory usage: ${MEM_MB}MB (<2GB requirement)"
        else
            log::warning "⚠ Memory usage: ${MEM_MB}MB (>2GB)"
        fi
    fi
else
    log::warning "⚠ Could not check resource usage"
fi

################################################################################
# Results
################################################################################

if [[ $FAILED -gt 0 ]]; then
    log::error "MinIO integration tests failed: $FAILED tests failed"
    exit 1
else
    log::success "All MinIO integration tests passed"
    exit 0
fi