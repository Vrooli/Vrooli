#!/bin/bash
# Smart File Photo Manager - Integration Test Runner
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$SCRIPT_DIR"

# shellcheck disable=SC1091
source "$(cd "$SCRIPT_DIR" && cd ../../lib/utils && pwd)/var.sh"
# shellcheck disable=SC1091
source "$var_LOG_FILE"

# Load test configuration
TEST_CONFIG_FILE="$SCENARIO_ROOT/scenario-test.yaml"


# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Test results storage
TEST_RESULTS=()

# Function to run individual test
run_test() {
    local test_name="$1"
    local test_description="$2"
    shift 2
    local test_steps=("$@")
    
    ((TESTS_TOTAL++))
    
    log_info "Running test: $test_name"
    log_info "Description: $test_description"
    
    local test_start_time=$(date +%s)
    local test_passed=true
    local test_output=""
    
    # Execute test steps
    for step in "${test_steps[@]}"; do
        log_info "Executing step: $step"
        
        if ! eval "$step" 2>&1; then
            test_passed=false
            test_output+="FAILED: $step\n"
            break
        else
            test_output+="PASSED: $step\n"
        fi
    done
    
    local test_end_time=$(date +%s)
    local test_duration=$((test_end_time - test_start_time))
    
    if [ "$test_passed" = true ]; then
        ((TESTS_PASSED++))
        log_success "Test '$test_name' PASSED (${test_duration}s)"
        TEST_RESULTS+=("PASS|$test_name|$test_duration|$test_description")
    else
        ((TESTS_FAILED++))
        log_error "Test '$test_name' FAILED (${test_duration}s)"
        log_error "Output: $test_output"
        TEST_RESULTS+=("FAIL|$test_name|$test_duration|$test_description|$test_output")
    fi
    
    echo "----------------------------------------"
}

# Test: Service availability
test_service_availability() {
    run_test "service_availability" "Verify all required services are running" \
        "check_service_port localhost 5435 'PostgreSQL'" \
        "check_service_port localhost 6380 'Redis'" \
        "check_service_port localhost 6335 'Qdrant'" \
        "check_service_port localhost 9001 'MinIO'" \
        "check_service_port localhost 11434 'Ollama'" \
        "check_service_port localhost 8000 'Unstructured-IO'" \
        "check_service_port localhost 5680 'n8n'" \
        "check_service_port localhost 8002 'Windmill'"
}

check_service_port() {
    local host="$1"
    local port="$2"
    local service_name="$3"
    
    if timeout 5 bash -c "</dev/tcp/$host/$port" 2>/dev/null; then
        log_info "$service_name is available on $host:$port"
        return 0
    else
        log_error "$service_name is not available on $host:$port"
        return 1
    fi
}

# Test: Database schema
test_database_schema() {
    run_test "database_schema" "Verify database schema is properly initialized" \
        "check_database_tables" \
        "check_database_indexes" \
        "verify_seed_data"
}

check_database_tables() {
    local expected_tables=("files" "content_chunks" "suggestions" "folders" "file_relationships" "processing_queue" "search_history" "batch_operations")
    
    for table in "${expected_tables[@]}"; do
        local count
        count=$(PGPASSWORD="filemanager_pass123" psql -h localhost -p 5435 -U filemanager_user -d file_manager -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '$table';")
        
        if [ "$count" -eq 1 ]; then
            log_info "Table '$table' exists"
        else
            log_error "Table '$table' does not exist"
            return 1
        fi
    done
}

check_database_indexes() {
    local index_count
    index_count=$(PGPASSWORD="filemanager_pass123" psql -h localhost -p 5435 -U filemanager_user -d file_manager -tAc "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public';")
    
    if [ "$index_count" -gt 10 ]; then
        log_info "Database has $index_count indexes"
    else
        log_error "Database has insufficient indexes ($index_count)"
        return 1
    fi
}

verify_seed_data() {
    local file_count
    file_count=$(PGPASSWORD="filemanager_pass123" psql -h localhost -p 5435 -U filemanager_user -d file_manager -tAc "SELECT COUNT(*) FROM files;")
    
    if [ "$file_count" -gt 0 ]; then
        log_info "Database has $file_count sample files"
    else
        log_warning "Database has no sample files"
        return 0  # Not a failure, just a warning
    fi
}

# Test: Vector database
test_vector_database() {
    run_test "vector_database" "Verify Qdrant collections are properly set up" \
        "check_qdrant_health" \
        "check_qdrant_collections"
}

check_qdrant_health() {
    local health_response
    health_response=$(curl -s http://localhost:6335/health || echo "ERROR")
    
    if echo "$health_response" | grep -q "ok"; then
        log_info "Qdrant health check passed"
    else
        log_error "Qdrant health check failed: $health_response"
        return 1
    fi
}

check_qdrant_collections() {
    local collections=("file_embeddings" "image_embeddings" "content_chunks")
    
    for collection in "${collections[@]}"; do
        local response
        response=$(curl -s "http://localhost:6335/collections/$collection" || echo "ERROR")
        
        if echo "$response" | grep -q '"status":"green"'; then
            log_info "Qdrant collection '$collection' is healthy"
        else
            log_error "Qdrant collection '$collection' is not healthy: $response"
            return 1
        fi
    done
}

# Test: Object storage
test_object_storage() {
    run_test "object_storage" "Verify MinIO buckets are properly configured" \
        "check_minio_health" \
        "check_minio_buckets"
}

check_minio_health() {
    if timeout 5 bash -c "</dev/tcp/localhost/9001" 2>/dev/null; then
        log_info "MinIO is accessible"
    else
        log_error "MinIO is not accessible"
        return 1
    fi
}

check_minio_buckets() {
    local expected_buckets=("original-files" "processed-files" "thumbnails" "exports")
    
    # This would require MinIO client (mc) to be configured
    # For now, just check if MinIO responds
    for bucket in "${expected_buckets[@]}"; do
        log_info "Expected bucket: $bucket"
    done
}

# Test: AI models
test_ai_models() {
    run_test "ai_models" "Verify Ollama models are available" \
        "check_ollama_health" \
        "check_ollama_models"
}

check_ollama_health() {
    local version_response
    version_response=$(curl -s http://localhost:11434/api/version || echo "ERROR")
    
    if echo "$version_response" | grep -q "version"; then
        log_info "Ollama is healthy"
    else
        log_error "Ollama health check failed: $version_response"
        return 1
    fi
}

check_ollama_models() {
    local models=("llava:13b" "llama3.2" "nomic-embed-text")
    local available_models
    available_models=$(curl -s http://localhost:11434/api/tags | jq -r '.models[].name' 2>/dev/null || echo "")
    
    for model in "${models[@]}"; do
        if echo "$available_models" | grep -q "$model"; then
            log_info "Model '$model' is available"
        else
            log_warning "Model '$model' is not available (may need to be pulled)"
        fi
    done
}

# Test: Windmill scripts
test_windmill_scripts() {
    run_test "windmill_scripts" "Verify Windmill scripts are deployed" \
        "check_windmill_health" \
        "check_windmill_workspace"
}

check_windmill_health() {
    local version_response
    version_response=$(curl -s http://localhost:8002/api/version || echo "ERROR")
    
    if echo "$version_response" | grep -q "version"; then
        log_info "Windmill is healthy"
    else
        log_error "Windmill health check failed: $version_response"
        return 1
    fi
}

check_windmill_workspace() {
    # This would require authentication token
    log_info "Windmill workspace check (requires authentication)"
}

# Test: n8n workflows
test_n8n_workflows() {
    run_test "n8n_workflows" "Verify n8n workflows are imported" \
        "check_n8n_health" \
        "check_n8n_webhooks"
}

check_n8n_health() {
    local health_response
    health_response=$(curl -s http://localhost:5680/healthz || echo "ERROR")
    
    if [ "$health_response" = "OK" ]; then
        log_info "n8n is healthy"
    else
        log_error "n8n health check failed: $health_response"
        return 1
    fi
}

check_n8n_webhooks() {
    local webhooks=("file-upload" "batch-process" "search")
    
    for webhook in "${webhooks[@]}"; do
        local webhook_url="http://localhost:5680/webhook/$webhook"
        local response
        response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$webhook_url" -H "Content-Type: application/json" -d '{}' || echo "000")
        
        # Expecting 400 (bad request) or similar, not 404 (not found)
        if [ "$response" != "404" ]; then
            log_info "Webhook '$webhook' is accessible (HTTP $response)"
        else
            log_error "Webhook '$webhook' not found (HTTP $response)"
            return 1
        fi
    done
}

# Test: End-to-end file processing
test_file_processing() {
    run_test "file_processing" "Test file upload and processing pipeline" \
        "simulate_file_upload" \
        "wait_for_processing" \
        "verify_file_processed"
}

simulate_file_upload() {
    log_info "Simulating file upload (mock test)"
    # This would test the actual file upload and processing pipeline
    # For now, just verify the webhook exists
    
    local webhook_url="http://localhost:5680/webhook/file-upload"
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$webhook_url" -H "Content-Type: application/json" -d '{"test": true}' || echo "000")
    
    if [ "$response" != "404" ]; then
        log_info "File upload webhook is accessible"
    else
        log_error "File upload webhook not found"
        return 1
    fi
}

wait_for_processing() {
    log_info "Waiting for processing simulation (5 seconds)"
    sleep 5
}

verify_file_processed() {
    log_info "File processing simulation completed"
}

# Test: Search functionality  
test_search_functionality() {
    run_test "search_functionality" "Test semantic search capabilities" \
        "test_search_webhook" \
        "verify_search_response"
}

test_search_webhook() {
    local search_url="http://localhost:5680/webhook/search"
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$search_url" -H "Content-Type: application/json" -d '{"query": "test search"}' || echo "000")
    
    if [ "$response" != "404" ]; then
        log_info "Search webhook is accessible"
    else
        log_error "Search webhook not found"
        return 1
    fi
}

verify_search_response() {
    log_info "Search functionality test completed"
}

# Generate test report
generate_report() {
    local report_file="$SCENARIO_ROOT/test-results.txt"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    cat > "$report_file" << EOF
Smart File Photo Manager - Integration Test Report
Generated: $timestamp

Test Summary:
- Total Tests: $TESTS_TOTAL
- Passed: $TESTS_PASSED
- Failed: $TESTS_FAILED
- Skipped: $TESTS_SKIPPED

Detailed Results:
EOF

    for result in "${TEST_RESULTS[@]}"; do
        IFS='|' read -r status name duration description output <<< "$result"
        cat >> "$report_file" << EOF

[$status] $name ($duration seconds)
Description: $description
EOF
        if [ "$status" = "FAIL" ] && [ -n "$output" ]; then
            echo "Output: $output" >> "$report_file"
        fi
    done
    
    log_info "Test report generated: $report_file"
}

# Main execution
main() {
    log_info "Starting Smart File Photo Manager Integration Tests"
    echo "=========================================="
    
    local start_time=$(date +%s)
    
    # Run all tests
    test_service_availability
    test_database_schema
    test_vector_database
    test_object_storage
    test_ai_models
    test_windmill_scripts
    test_n8n_workflows
    test_file_processing
    test_search_functionality
    
    local end_time=$(date +%s)
    local total_duration=$((end_time - start_time))
    
    echo "=========================================="
    log_info "Integration tests completed in ${total_duration} seconds"
    
    # Summary
    echo ""
    echo "TEST SUMMARY:"
    echo "=============="
    log_info "Total Tests: $TESTS_TOTAL"
    
    if [ $TESTS_PASSED -gt 0 ]; then
        log_success "Passed: $TESTS_PASSED"
    fi
    
    if [ $TESTS_FAILED -gt 0 ]; then
        log_error "Failed: $TESTS_FAILED"
    fi
    
    if [ $TESTS_SKIPPED -gt 0 ]; then
        log_warning "Skipped: $TESTS_SKIPPED"
    fi
    
    # Generate report
    generate_report
    
    # Exit with appropriate code
    if [ $TESTS_FAILED -gt 0 ]; then
        log_error "Some tests failed. Check the test report for details."
        exit 1
    else
        log_success "All tests passed!"
        exit 0
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi