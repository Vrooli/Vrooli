#!/bin/bash
# Comprehensive integration tests for audio-tools scenario
# Tests end-to-end workflows, database integration, and error scenarios

set -euo pipefail

# Configuration
if [ -z "${API_PORT:-}" ]; then
    API_PORT=$(ps aux | grep -E "audio-tools.*-port" | grep -oE "\-port [0-9]+" | awk '{print $2}' | head -1)
    if [ -z "$API_PORT" ]; then
        API_PORT="19607"
    fi
fi
readonly API_BASE="http://localhost:${API_PORT}"
readonly TEST_DIR="/tmp/audio-tools-comprehensive-$$"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly BLUE='\033[0;34m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

# Test tracking
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Setup comprehensive test environment
setup_comprehensive() {
    echo -e "${BLUE}Setting up comprehensive test environment...${NC}"
    mkdir -p "$TEST_DIR"

    # Create multiple test audio files of different sizes and formats
    echo "Creating test audio files..."

    # Small WAV file (1 second, 44.1kHz, stereo)
    ffmpeg -y -f lavfi -i "sine=frequency=440:duration=1" -ar 44100 -ac 2 -f wav "$TEST_DIR/small.wav" 2>/dev/null

    # Medium MP3 file (10 seconds, with noise)
    ffmpeg -y -f lavfi -i "sine=frequency=440:duration=10" -f lavfi -i "anoisesrc=duration=10" -filter_complex "[0:a][1:a]amix=inputs=2:duration=first" -ar 44100 -ac 2 -b:a 128k "$TEST_DIR/medium_noisy.mp3" 2>/dev/null

    # Large WAV file (30 seconds, high quality)
    ffmpeg -y -f lavfi -i "sine=frequency=440:duration=30" -ar 96000 -ac 2 -f wav "$TEST_DIR/large.wav" 2>/dev/null

    # Stereo file with different channels
    ffmpeg -y -f lavfi -i "sine=frequency=440:duration=5" -f lavfi -i "sine=frequency=880:duration=5" -filter_complex "[0:a][1:a]join=inputs=2:channel_layout=stereo" -ar 44100 -ac 2 "$TEST_DIR/stereo_test.wav" 2>/dev/null

    # Wait for API to be ready
    local retries=15
    while [ $retries -gt 0 ]; do
        if curl -sf "$API_BASE/health" > /dev/null 2>&1; then
            echo -e "${GREEN}API is ready${NC}"
            return 0
        fi
        echo "Waiting for API to be ready... ($retries retries left)"
        sleep 2
        retries=$((retries - 1))
    done

    echo -e "${RED}API failed to start${NC}"
    return 1
}

# Cleanup comprehensive test environment
cleanup_comprehensive() {
    echo -e "${BLUE}Cleaning up comprehensive test environment...${NC}"
    rm -rf "$TEST_DIR"
}

# Test function wrapper with timing
test_case() {
    local test_name="$1"
    local test_command="$2"
    local start_time
    local end_time
    local duration

    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    start_time=$(date +%s.%3N)

    echo -e "${BLUE}Running: $test_name${NC}"

    if eval "$test_command"; then
        end_time=$(date +%s.%3N)
        duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0")
        echo -e "${GREEN}✓ $test_name passed (${duration}s)${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        end_time=$(date +%s.%3N)
        duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "0")
        echo -e "${RED}✗ $test_name failed (${duration}s)${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Comprehensive Integration Test 1: Complete Audio Processing Workflow
test_complete_audio_workflow() {
    local job_id
    local asset_id
    local processed_file

    echo "Step 1: Upload and analyze audio file"
    # Upload audio file and get asset ID
    asset_id=$(curl -sf -X POST "$API_BASE/api/v1/audio/analyze" \
        -F "audio=@$TEST_DIR/medium_noisy.mp3" \
        -F "detailed_analysis=true" | jq -r '.asset_id')

    [ -n "$asset_id" ] && [ "$asset_id" != "null" ]

    echo "Step 2: Store audio metadata in database"
    # Verify database storage by checking if we can retrieve the asset
    local stored_asset
    stored_asset=$(curl -sf "$API_BASE/api/v1/audio/assets/$asset_id" | jq -r '.id')
    [ "$stored_asset" = "$asset_id" ]

    echo "Step 3: Apply multiple audio processing operations"
    # Chain operations: noise reduction, normalization, format conversion
    job_id=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/medium_noisy.mp3" \
        -F 'operations=[{"type":"noise_reduction","parameters":{"intensity":0.8}},{"type":"normalize","parameters":{"target_level":-16}}]' \
        -F "output_format=mp3" | jq -r '.job_id')

    [ -n "$job_id" ] && [ "$job_id" != "null" ]

    echo "Step 4: Monitor processing job status"
    # Poll job status until complete
    local status="processing"
    local retries=10
    while [ "$status" = "processing" ] && [ $retries -gt 0 ]; do
        status=$(curl -sf "$API_BASE/api/v1/audio/jobs/$job_id" | jq -r '.status')
        sleep 1
        retries=$((retries - 1))
    done
    [ "$status" = "completed" ]

    echo "Step 5: Retrieve processed audio file"
    processed_file=$(curl -sf "$API_BASE/api/v1/audio/jobs/$job_id/result" | jq -r '.output_files[0].file_path')
    [ -n "$processed_file" ]

    echo "Step 6: Verify audio quality improvement"
    # Compare original and processed audio quality metrics
    local original_quality
    local processed_quality
    original_quality=$(curl -sf -X POST "$API_BASE/api/v1/audio/analyze" \
        -F "audio=@$TEST_DIR/medium_noisy.mp3" | jq -r '.quality_metrics.snr_db')
    processed_quality=$(curl -sf "$API_BASE/api/v1/audio/assets/$asset_id/quality" | jq -r '.snr_improvement_db')

    # Processed audio should have better quality metrics
    [ "$(echo "$processed_quality > 0" | bc -l 2>/dev/null)" = "1" ]

    echo "Step 7: Batch process multiple files"
    # Process multiple files in batch
    local batch_job_id
    batch_job_id=$(curl -sf -X POST "$API_BASE/api/v1/audio/batch/enhance" \
        -F "files[]=@$TEST_DIR/small.wav" \
        -F "files[]=@$TEST_DIR/stereo_test.wav" \
        -F "operation=normalize" | jq -r '.batch_job_id')

    [ -n "$batch_job_id" ]

    echo "Step 8: Verify batch processing results"
    local batch_status="processing"
    retries=15
    while [ "$batch_status" = "processing" ] && [ $retries -gt 0 ]; do
        batch_status=$(curl -sf "$API_BASE/api/v1/audio/batch/$batch_job_id" | jq -r '.status')
        sleep 2
        retries=$((retries - 1))
    done
    [ "$batch_status" = "completed" ]

    echo "Step 9: Generate processing report"
    local report
    report=$(curl -sf "$API_BASE/api/v1/audio/jobs/$job_id/report" | jq -r '.processing_summary.total_files_processed')
    [ "$report" = "1" ]
}

# Comprehensive Integration Test 2: Error Handling and Rollback Scenarios
test_error_handling_and_rollback() {
    local invalid_job_id
    local oversized_file
    local corrupt_file

    echo "Step 1: Test invalid file format handling"
    # Try to process an invalid file format
    local error_response
    error_response=$(curl -sf -w "%{http_code}" -X POST "$API_BASE/api/v1/audio/convert" \
        -F "audio=@$TEST_DIR/small.wav" \
        -F "output_format=invalid_format" 2>/dev/null | tail -1)

    [ "$error_response" = "400" ]

    echo "Step 2: Test oversized file handling"
    # Create an oversized file (simulate)
    dd if=/dev/zero of="$TEST_DIR/oversized.wav" bs=1M count=600 2>/dev/null
    error_response=$(curl -sf -w "%{http_code}" -X POST "$API_BASE/api/v1/audio/analyze" \
        -F "audio=@$TEST_DIR/oversized.wav" 2>/dev/null | tail -1)

    [ "$error_response" = "413" ] || [ "$error_response" = "400" ]

    echo "Step 3: Test corrupt file handling"
    # Create a corrupt file
    echo "corrupt audio data" > "$TEST_DIR/corrupt.wav"
    error_response=$(curl -sf -w "%{http_code}" -X POST "$API_BASE/api/v1/audio/metadata" \
        -F "audio=@$TEST_DIR/corrupt.wav" 2>/dev/null | tail -1)

    [ "$error_response" = "400" ]

    echo "Step 4: Test invalid job ID handling"
    invalid_job_id="invalid-job-uuid-12345"
    error_response=$(curl -sf -w "%{http_code}" "$API_BASE/api/v1/audio/jobs/$invalid_job_id" 2>/dev/null | tail -1)
    [ "$error_response" = "404" ]

    echo "Step 5: Test processing timeout handling"
    # Try to process with extreme parameters that might timeout
    local timeout_job_id
    timeout_job_id=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/large.wav" \
        -F 'operations=[{"type":"speed","parameters":{"speed_factor":0.1}}]' \
        -F "timeout=1" | jq -r '.job_id')

    [ -n "$timeout_job_id" ]

    # Wait for timeout
    sleep 3
    local timeout_status
    timeout_status=$(curl -sf "$API_BASE/api/v1/audio/jobs/$timeout_job_id" | jq -r '.status')
    [ "$timeout_status" = "failed" ] || [ "$timeout_status" = "timeout" ]

    echo "Step 6: Test concurrent processing limits"
    # Launch multiple concurrent requests
    local concurrent_jobs=()
    for i in {1..5}; do
        local concurrent_job
        concurrent_job=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
            -F "audio=@$TEST_DIR/small.wav" \
            -F "operation=normalize" | jq -r '.job_id' &)
        concurrent_jobs+=("$concurrent_job")
    done

    # Wait for all jobs to complete
    wait
    local all_completed=true
    for job in "${concurrent_jobs[@]}"; do
        local job_status
        job_status=$(curl -sf "$API_BASE/api/v1/audio/jobs/$job" | jq -r '.status')
        if [ "$job_status" != "completed" ] && [ "$job_status" != "failed" ]; then
            all_completed=false
            break
        fi
    done
    [ "$all_completed" = true ]

    echo "Step 7: Test database rollback on processing failure"
    # Simulate a processing failure and verify database state
    local failed_job_id
    failed_job_id=$(curl -sf -X POST "$API_BASE/api/v1/audio/edit" \
        -F "audio=@$TEST_DIR/small.wav" \
        -F 'operations=[{"type":"invalid_operation","parameters":{}}]' | jq -r '.job_id')

    [ -n "$failed_job_id" ]

    # Wait for failure
    sleep 2
    local failed_status
    failed_status=$(curl -sf "$API_BASE/api/v1/audio/jobs/$failed_job_id" | jq -r '.status')
    [ "$failed_status" = "failed" ]

    # Verify no orphaned records in database
    local job_exists
    job_exists=$(curl -sf "$API_BASE/api/v1/audio/jobs/$failed_job_id" | jq -r '.id')
    [ -n "$job_exists" ] # Job record should still exist for audit purposes

    echo "Step 8: Test service recovery after errors"
    # Verify service is still healthy after error scenarios
    local health_status
    health_status=$(curl -sf "$API_BASE/health" | jq -r '.status')
    [ "$health_status" = "healthy" ]

    echo "Step 9: Test rate limiting"
    # Rapid fire requests to test rate limiting
    local rate_limit_hit=false
    for i in {1..20}; do
        local rate_limit_response
        rate_limit_response=$(curl -sf -w "%{http_code}" -X POST "$API_BASE/api/v1/audio/metadata" \
            -F "audio=@$TEST_DIR/small.wav" 2>/dev/null | tail -1)
        if [ "$rate_limit_response" = "429" ]; then
            rate_limit_hit=true
            break
        fi
    done
    # Rate limiting may or may not be implemented, so this is informational
    echo "Rate limiting test completed (may not be enforced)"
}

# Performance Test with Realistic Data Volumes
test_realistic_data_volumes() {
    local large_batch_size=10
    local batch_files=()

    echo "Step 1: Create realistic data volume ($large_batch_size files)"
    # Create multiple audio files of varying sizes
    for i in {1..10}; do
        local file_size=$((RANDOM % 5 + 1)) # 1-5 MB files
        local duration=$((RANDOM % 60 + 10)) # 10-70 second files

        ffmpeg -y -f lavfi -i "sine=frequency=$((220 + i * 50)):duration=$duration" \
            -ar 44100 -ac 2 -b:a 128k "$TEST_DIR/batch_$i.mp3" 2>/dev/null

        batch_files+=("$TEST_DIR/batch_$i.mp3")
    done

    echo "Step 2: Test batch processing performance"
    local start_time
    local end_time
    local batch_job_id

    start_time=$(date +%s)

    # Create multipart form data for batch processing
    local form_data=""
    for file in "${batch_files[@]}"; do
        form_data="$form_data -F \"files[]=@$file\""
    done

    # Execute batch processing
    batch_job_id=$(eval "curl -sf -X POST \"$API_BASE/api/v1/audio/batch/convert\" $form_data -F \"output_format=wav\"" | jq -r '.batch_job_id')

    [ -n "$batch_job_id" ] && [ "$batch_job_id" != "null" ]

    echo "Step 3: Monitor batch processing progress"
    local batch_status="processing"
    local progress=0
    local retries=60 # Allow up to 2 minutes for large batch

    while [ "$batch_status" = "processing" ] && [ $retries -gt 0 ]; do
        batch_status=$(curl -sf "$API_BASE/api/v1/audio/batch/$batch_job_id" | jq -r '.status')
        progress=$(curl -sf "$API_BASE/api/v1/audio/batch/$batch_job_id" | jq -r '.progress_percentage // 0')
        echo "Batch progress: $progress% ($batch_status)"
        sleep 2
        retries=$((retries - 1))
    done

    end_time=$(date +%s)
    local processing_time=$((end_time - start_time))

    [ "$batch_status" = "completed" ]
    echo "Batch processing completed in ${processing_time}s"

    echo "Step 4: Verify batch results integrity"
    local results
    results=$(curl -sf "$API_BASE/api/v1/audio/batch/$batch_job_id/results" | jq -r '.files_processed')
    [ "$results" = "$large_batch_size" ]

    echo "Step 5: Test database performance with large dataset"
    # Query database for all processed files
    local db_query_start
    local db_query_end
    db_query_start=$(date +%s.%3N)

    local db_results
    db_results=$(curl -sf "$API_BASE/api/v1/audio/assets?limit=50" | jq -r '.assets | length')

    db_query_end=$(date +%s.%3N)
    local db_query_time
    db_query_time=$(echo "$db_query_end - $db_query_start" | bc 2>/dev/null || echo "0")

    # Database query should be fast (< 1 second)
    [ "$(echo "$db_query_time < 1.0" | bc -l 2>/dev/null)" = "1" ]

    echo "Database query completed in ${db_query_time}s for $db_results records"

    echo "Step 6: Test storage system performance"
    # Verify files are properly stored and retrievable
    local storage_test_file
    storage_test_file=$(curl -sf "$API_BASE/api/v1/audio/batch/$batch_job_id/results" | jq -r '.processed_files[0].file_path')

    [ -n "$storage_test_file" ]

    # Test file retrieval
    local retrieval_status
    retrieval_status=$(curl -sf -w "%{http_code}" "$API_BASE/api/v1/audio/files/$storage_test_file" 2>/dev/null | tail -1)
    [ "$retrieval_status" = "200" ]

    echo "Step 7: Performance metrics validation"
    # Validate that processing meets performance targets
    local avg_processing_time_per_file
    avg_processing_time_per_file=$(echo "scale=2; $processing_time / $large_batch_size" | bc 2>/dev/null || echo "10")

    # Average processing time should be reasonable (< 30 seconds per file)
    [ "$(echo "$avg_processing_time_per_file < 30.0" | bc -l 2>/dev/null)" = "1" ]

    echo "Average processing time: ${avg_processing_time_per_file}s per file"
}

# Main test execution
main() {
    trap cleanup_comprehensive EXIT

    echo -e "${BLUE}================================================================${NC}"
    echo -e "${BLUE}Audio Tools Comprehensive Integration Tests${NC}"
    echo -e "${BLUE}================================================================${NC}"

    setup_comprehensive || exit 1

    echo -e "${YELLOW}Running Test 1: Complete Audio Processing Workflow${NC}"
    test_case "Complete Audio Workflow" test_complete_audio_workflow

    echo -e "${YELLOW}Running Test 2: Error Handling and Rollback${NC}"
    test_case "Error Handling & Rollback" test_error_handling_and_rollback

    echo -e "${YELLOW}Running Test 3: Realistic Data Volumes${NC}"
    test_case "Realistic Data Volumes" test_realistic_data_volumes

    # Summary
    echo -e "${BLUE}================================================================${NC}"
    echo -e "${BLUE}Comprehensive Integration Test Summary${NC}"
    echo -e "${BLUE}================================================================${NC}"
    echo -e "${GREEN}  Total Tests: $TESTS_TOTAL${NC}"
    echo -e "${GREEN}  Passed: $TESTS_PASSED${NC}"
    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${RED}  Failed: $TESTS_FAILED${NC}"
        echo -e "${YELLOW}  Success Rate: $((TESTS_PASSED * 100 / TESTS_TOTAL))%${NC}"
    else
        echo -e "${GREEN}  Failed: $TESTS_FAILED${NC}"
        echo -e "${GREEN}  Success Rate: 100%${NC}"
    fi
    echo -e "${BLUE}================================================================${NC}"

    # Exit with failure if any tests failed
    [ $TESTS_FAILED -eq 0 ]
}

# Run tests
main "$@"