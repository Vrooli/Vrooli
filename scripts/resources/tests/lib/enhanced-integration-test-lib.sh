#!/usr/bin/env bash
# Enhanced Integration Test Library
# Extends the basic integration test library with fixture support and better patterns
set -euo pipefail

_HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${_HERE}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${_HERE}/integration-test-lib.sh"

# Load fixture loader for optional fixture testing
FIXTURE_LOADER_PATH="$var_SCRIPTS_TEST_DIR/fixtures/fixture-loader.bash"
if [[ -f "$FIXTURE_LOADER_PATH" ]]; then
    # shellcheck disable=SC1091
    source "$FIXTURE_LOADER_PATH"
    FIXTURES_AVAILABLE=true
else
    FIXTURES_AVAILABLE=false
fi

#######################################
# Enhanced Configuration
#######################################

# Integration test modes
INTEGRATION_MODE="${INTEGRATION_MODE:-basic}"  # basic, fixtures, comprehensive
FIXTURE_TESTING="${FIXTURE_TESTING:-auto}"     # auto, enabled, disabled

# Test data tracking
declare -a PROCESSED_FIXTURES=()
declare -a FIXTURE_RESULTS=()

#######################################
# Enhanced Test Functions
#######################################

# Test with optional fixture data
# Usage: enhanced_test_lib::test_with_fixture "test_name" "fixture_category" "fixture_name" test_function [expected_response]
enhanced_test_lib::test_with_fixture() {
    local test_name="$1"
    local fixture_category="$2"
    local fixture_name="$3"
    local test_function="$4"
    local expected_response="${5:-}"
    
    # Skip fixture testing if not available or disabled
    if [[ "$FIXTURES_AVAILABLE" != "true" || "$FIXTURE_TESTING" == "disabled" ]]; then
        log_test_result "$test_name" "SKIP" "fixture testing not available/enabled"
        return 2
    fi
    
    # Get fixture path
    local fixture_path
    if ! fixture_path=$(fixture_get_path "$fixture_category" "$fixture_name" 2>/dev/null); then
        log_test_result "$test_name" "SKIP" "fixture not found: $fixture_category/$fixture_name"
        return 2
    fi
    
    # Track fixture usage
    PROCESSED_FIXTURES+=("$fixture_category/$fixture_name")
    
    # Execute the test function with fixture
    if $test_function "$fixture_path" "$expected_response"; then
        FIXTURE_RESULTS+=("PASS:$fixture_category/$fixture_name")
        log_test_result "$test_name" "PASS" "with fixture: $fixture_name"
        return 0
    else
        FIXTURE_RESULTS+=("FAIL:$fixture_category/$fixture_name")
        log_test_result "$test_name" "FAIL" "with fixture: $fixture_name"
        return 1
    fi
}

# Enhanced service health check with deep validation
enhanced_test_lib::enhanced_service_health_check() {
    local test_name="service health check (enhanced)"
    
    # Basic connectivity
    if ! check_service_available "$BASE_URL"; then
        log_test_result "$test_name" "FAIL" "service not responding"
        return 1
    fi
    
    # Check multiple endpoints if available
    local endpoints=("$HEALTH_ENDPOINT")
    if [[ -n "${ADDITIONAL_ENDPOINTS:-}" ]]; then
        IFS=' ' read -ra additional <<< "$ADDITIONAL_ENDPOINTS"
        endpoints+=("${additional[@]}")
    fi
    
    local failed_endpoints=()
    local working_endpoints=()
    
    for endpoint in "${endpoints[@]}"; do
        if make_api_request "$endpoint" "GET" 10 >/dev/null; then
            working_endpoints+=("$endpoint")
        else
            failed_endpoints+=("$endpoint")
        fi
    done
    
    if [[ ${#failed_endpoints[@]} -gt 0 ]]; then
        log_test_result "$test_name" "FAIL" "endpoints failed: ${failed_endpoints[*]}"
        return 1
    else
        log_test_result "$test_name" "PASS" "all endpoints working: ${working_endpoints[*]}"
        return 0
    fi
}

# Test API with different content types
enhanced_test_lib::test_api_content_types() {
    local test_name="API content type support"
    local api_endpoint="$1"
    
    local content_types=(
        "application/json"
        "application/x-www-form-urlencoded"
        "multipart/form-data"
        "text/plain"
    )
    
    local supported_types=()
    local unsupported_types=()
    
    for content_type in "${content_types[@]}"; do
        local response
        if response=$(make_api_request "$api_endpoint" "POST" 10 "-H 'Content-Type: $content_type'"); then
            # Check if response indicates content type is supported (not a 415 error)
            if [[ ! "$response" =~ "Unsupported Media Type"|"415" ]]; then
                supported_types+=("$content_type")
            else
                unsupported_types+=("$content_type")
            fi
        else
            unsupported_types+=("$content_type")  
        fi
    done
    
    if [[ ${#supported_types[@]} -gt 0 ]]; then
        log_test_result "$test_name" "PASS" "supported: ${supported_types[*]}"
        return 0
    else
        log_test_result "$test_name" "FAIL" "no content types supported"
        return 1
    fi
}

# Test service under load (basic)
enhanced_test_lib::test_service_load() {
    local test_name="basic load handling"
    local endpoint="${1:-$HEALTH_ENDPOINT}"
    local concurrent_requests="${2:-5}"
    local request_count="${3:-10}"
    
    log_test_result "$test_name" "INFO" "testing with $concurrent_requests concurrent requests, $request_count total"
    
    # Create temporary directory for results
    local temp_dir="${TMPDIR:-/tmp}/load_test_$$"
    mkdir -p "$temp_dir"
    
    # Function to make single request
    enhanced_test_lib::make_load_request() {
        local request_id="$1"
        local result_file="$temp_dir/result_$request_id"
        
        if make_api_request "$endpoint" "GET" 30 >/dev/null 2>&1; then
            echo "PASS" > "$result_file"
        else
            echo "FAIL" > "$result_file"
        fi
    }
    
    # Launch concurrent requests
    local pids=()
    for ((i=1; i<=request_count; i++)); do
        enhanced_test_lib::make_load_request "$i" &
        pids+=($!)
        
        # Limit concurrency
        if [[ $((i % concurrent_requests)) -eq 0 ]]; then
            wait "${pids[@]}"
            pids=()
        fi
    done
    
    # Wait for remaining requests
    wait "${pids[@]}"
    
    # Count results
    local passed_requests=0
    local failed_requests=0
    
    for ((i=1; i<=request_count; i++)); do
        local result_file="$temp_dir/result_$i"
        if [[ -f "$result_file" ]]; then
            if grep -q "PASS" "$result_file"; then
                ((passed_requests++))
            else
                ((failed_requests++))
            fi
        else
            ((failed_requests++))
        fi
    done
    
    # Cleanup
    rm -rf "$temp_dir"
    
    # Evaluate results (80% success rate considered acceptable)
    local success_rate=$((passed_requests * 100 / request_count))
    
    if [[ $success_rate -ge 80 ]]; then
        log_test_result "$test_name" "PASS" "$success_rate% success rate ($passed_requests/$request_count)"
        return 0
    else
        log_test_result "$test_name" "FAIL" "$success_rate% success rate ($passed_requests/$request_count)"
        return 1
    fi
}

# Validate service configuration
enhanced_test_lib::test_service_configuration() {
    local test_name="service configuration validation"
    
    # Check if service exposes configuration endpoint
    local config_endpoints=("/config" "/api/config" "/v1/config" "/settings")
    local working_config_endpoint=""
    
    for endpoint in "${config_endpoints[@]}"; do
        if make_api_request "$endpoint" "GET" 5 >/dev/null 2>&1; then
            working_config_endpoint="$endpoint"
            break
        fi
    done
    
    if [[ -n "$working_config_endpoint" ]]; then
        local config_response
        config_response=$(make_api_request "$working_config_endpoint" "GET" 10)
        
        # Basic validation that response looks like configuration
        if [[ "$config_response" =~ \{.*\}|\[.*\] ]]; then
            log_test_result "$test_name" "PASS" "config available at $working_config_endpoint"
            return 0
        else
            log_test_result "$test_name" "FAIL" "config endpoint exists but response invalid"
            return 1
        fi
    else
        log_test_result "$test_name" "SKIP" "no configuration endpoint found"
        return 2
    fi
}

#######################################
# Category-specific enhanced test patterns
#######################################

# Enhanced AI service tests
enhanced_test_lib::register_enhanced_ai_tests() {
    local enhanced_ai_tests=(
        "enhanced_service_health_check"
        "test_service_configuration"
        "test_ai_model_availability"
        "test_ai_generation_basic"
    )
    
    REGISTERED_TESTS+=("${enhanced_ai_tests[@]}")
}

enhanced_test_lib::test_ai_model_availability() {
    local test_name="AI model availability check"
    
    # Common model list endpoints
    local model_endpoints=("/api/tags" "/v1/models" "/models" "/api/models")
    local working_endpoint=""
    
    for endpoint in "${model_endpoints[@]}"; do
        if make_api_request "$endpoint" "GET" 10 >/dev/null 2>&1; then
            working_endpoint="$endpoint"
            break
        fi
    done
    
    if [[ -n "$working_endpoint" ]]; then
        local models_response
        models_response=$(make_api_request "$working_endpoint" "GET" 10)
        
        # Check if response contains model information
        if [[ "$models_response" =~ models|data|\[.*\] ]]; then
            log_test_result "$test_name" "PASS" "models available via $working_endpoint"
            return 0
        else
            log_test_result "$test_name" "FAIL" "models endpoint exists but no models found"
            return 1  
        fi
    else
        log_test_result "$test_name" "SKIP" "no model list endpoint found"
        return 2
    fi
}

enhanced_test_lib::test_ai_generation_basic() {
    local test_name="AI generation basic functionality"
    
    # Common generation endpoints
    local gen_endpoints=("/api/generate" "/v1/completions" "/generate" "/api/completions")
    local working_endpoint=""
    
    for endpoint in "${gen_endpoints[@]}"; do
        # Try a simple test request
        local test_data='{"prompt":"hello","max_tokens":5,"temperature":0.1}'
        if make_api_request "$endpoint" "POST" 30 "-H 'Content-Type: application/json' -d '$test_data'" >/dev/null 2>&1; then
            working_endpoint="$endpoint"
            break
        fi
    done
    
    if [[ -n "$working_endpoint" ]]; then
        log_test_result "$test_name" "PASS" "generation working via $working_endpoint"
        return 0
    else
        log_test_result "$test_name" "SKIP" "no working generation endpoint found"
        return 2
    fi
}

# Enhanced automation service tests
enhanced_test_lib::register_enhanced_automation_tests() {
    local enhanced_automation_tests=(
        "enhanced_service_health_check"
        "test_service_configuration"
        "test_automation_workflow_list"
        "test_automation_execution_capability"
    )
    
    REGISTERED_TESTS+=("${enhanced_automation_tests[@]}")
}

enhanced_test_lib::test_automation_workflow_list() {
    local test_name="automation workflow listing"
    
    local workflow_endpoints=("/api/workflows" "/workflows" "/v1/workflows")
    local working_endpoint=""
    
    for endpoint in "${workflow_endpoints[@]}"; do
        if make_api_request "$endpoint" "GET" 10 >/dev/null 2>&1; then
            working_endpoint="$endpoint"
            break
        fi
    done
    
    if [[ -n "$working_endpoint" ]]; then
        log_test_result "$test_name" "PASS" "workflows accessible via $working_endpoint"
        return 0
    else
        log_test_result "$test_name" "SKIP" "no workflow endpoint found"
        return 2
    fi
}

enhanced_test_lib::test_automation_execution_capability() {
    local test_name="automation execution capability"
    
    # Test fixture workflow if available
    if [[ "$FIXTURES_AVAILABLE" == "true" ]]; then
        local workflow_fixture
        if workflow_fixture=$(fixture_get_path "workflows" "n8n/n8n-workflow.json" 2>/dev/null); then
            # This would require complex workflow execution testing
            # For now, just validate that execution endpoints exist
            local exec_endpoints=("/api/executions" "/executions" "/v1/executions")
            
            for endpoint in "${exec_endpoints[@]}"; do
                if make_api_request "$endpoint" "GET" 10 >/dev/null 2>&1; then
                    log_test_result "$test_name" "PASS" "execution endpoint available: $endpoint"
                    return 0
                fi
            done
        fi
    fi
    
    log_test_result "$test_name" "SKIP" "execution testing requires complex setup"
    return 2
}

#######################################
# Enhanced reporting
#######################################

enhanced_test_lib::generate_enhanced_report() {
    echo
    echo "═══════════════════════════════════════════════════════════════"
    echo "  Enhanced Integration Test Report"
    echo "═══════════════════════════════════════════════════════════════"
    echo
    
    # Standard report
    echo "Test Results:"
    echo "  Passed:  $TESTS_PASSED ✅"
    echo "  Failed:  $TESTS_FAILED ❌"
    echo "  Skipped: $TESTS_SKIPPED ⏭️"
    echo
    
    # Generate detailed fixture report if fixtures were used
    if [[ ${#PROCESSED_FIXTURES[@]} -gt 0 ]]; then
        enhanced_test_lib::generate_fixture_report
    fi
    
    # Service-specific insights
    if [[ -n "${SERVICE_NAME:-}" ]]; then
        echo "Service Analysis for: $SERVICE_NAME"
        echo "  Base URL: $BASE_URL"
        echo "  Health endpoint: $HEALTH_ENDPOINT"
        
        if [[ ${#SERVICE_METADATA[@]} -gt 0 ]]; then
            echo "  Metadata:"
            for metadata in "${SERVICE_METADATA[@]}"; do
                echo "    - $metadata"
            done
        fi
        echo
    fi
    
    echo "═══════════════════════════════════════════════════════════════"
}

# Override the main execution to use enhanced reporting
enhanced_test_lib::enhanced_integration_test_main() {
    init_config
    
    # Check prerequisites
    if ! check_required_tools; then
        echo "Prerequisites not met, exiting..."
        exit 1
    fi
    
    # Service availability check
    if ! check_service_available "$BASE_URL"; then
        echo "Service not available at $BASE_URL, exiting..."
        exit 2
    fi
    
    echo "Starting enhanced integration tests for: $SERVICE_NAME"
    echo "Base URL: $BASE_URL"
    echo "Mode: $INTEGRATION_MODE"
    echo
    
    # Run registered tests
    for test_function in "${REGISTERED_TESTS[@]}"; do
        if declare -F "$test_function" >/dev/null; then
            $test_function
        else
            log_test_result "$test_function" "FAIL" "test function not found"
        fi
    done
    
    # Generate enhanced report
    enhanced_test_lib::generate_enhanced_report
    
    # Exit with appropriate code
    if [[ $TESTS_FAILED -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

#######################################
# Fixture Usage Report Generation
#######################################

enhanced_test_lib::generate_fixture_report() {
    echo
    echo "=== FIXTURE USAGE REPORT ==="
    echo "Total fixtures processed: ${#PROCESSED_FIXTURES[@]}"
    
    # Count unique fixtures
    local unique_count=0
    if [[ ${#PROCESSED_FIXTURES[@]} -gt 0 ]]; then
        unique_count=$(printf '%s\n' "${PROCESSED_FIXTURES[@]}" | sort -u | wc -l)
    fi
    echo "Unique fixtures used: $unique_count"
    
    # Show pass/fail by fixture
    local pass_count=0
    local fail_count=0
    for result in "${FIXTURE_RESULTS[@]}"; do
        if [[ "$result" == PASS:* ]]; then
            ((pass_count++))
        else
            ((fail_count++))
        fi
    done
    
    echo "Fixture test results: $pass_count passed, $fail_count failed"
    
    # List fixtures used in verbose mode
    if [[ "$VERBOSE" == "true" ]] && [[ ${#PROCESSED_FIXTURES[@]} -gt 0 ]]; then
        echo
        echo "Detailed fixture usage:"
        printf '%s\n' "${PROCESSED_FIXTURES[@]}" | sort -u | while read -r fixture; do
            echo "  - $fixture"
        done
    fi
}

#######################################
# Fixture Auto-Discovery
#######################################

enhanced_test_lib::discover_resource_fixtures() {
    local resource_name="$1"
    local resource_category="${2:-}"
    
    # If category not provided, try to determine it
    if [[ -z "$resource_category" ]]; then
        case "$resource_name" in
            ollama|whisper|unstructured-io) resource_category="ai" ;;
            n8n|comfyui|node-red|windmill|huginn) resource_category="automation" ;;
            minio|vault|qdrant|questdb|postgres|redis) resource_category="storage" ;;
            judge0) resource_category="execution" ;;
            browserless|claude-code|agent-s2) resource_category="agents" ;;
            searxng) resource_category="search" ;;
        esac
    fi
    
    # Return appropriate fixtures based on resource
    case "$resource_category" in
        "ai")
            case "$resource_name" in
                "whisper")
                    echo "audio/whisper audio/speech_test_short.mp3 audio/speech_mlk_dream.mp3"
                    ;;
                "unstructured-io")
                    echo "documents/pdf documents/office documents/structured"
                    ;;
                "ollama")
                    echo "documents/llm-prompt.json images/ocr"
                    ;;
            esac
            ;;
        "automation")
            # Check for resource-specific workflow fixtures
            if [[ -d "$FIXTURES_DIR/workflows/$resource_name" ]]; then
                echo "workflows/$resource_name"
            else
                echo "workflows"
            fi
            ;;
        "storage")
            # Storage resources should test with various file types
            echo "documents/pdf images/synthetic documents/structured"
            ;;
        "execution")
            echo "documents/code"
            ;;
        "agents")
            case "$resource_name" in
                "browserless")
                    echo "images/real-world documents/web"
                    ;;
                "claude-code")
                    echo "documents/code"
                    ;;
                "agent-s2")
                    echo "images/real-world documents/web"
                    ;;
            esac
            ;;
        "search")
            echo "documents/samples"
            ;;
    esac
}

#######################################
# Fixture Rotation for Testing Variety
#######################################

enhanced_test_lib::rotate_fixtures() {
    local category="$1"
    local max_fixtures="${2:-5}"
    
    if [[ "$FIXTURES_AVAILABLE" != "true" ]]; then
        return 1
    fi
    
    # Get all fixtures and randomly select some
    if command -v shuf >/dev/null 2>&1; then
        fixture_get_all "$category" 2>/dev/null | shuf | head -n "$max_fixtures"
    else
        # Fallback if shuf not available
        fixture_get_all "$category" 2>/dev/null | sort -R | head -n "$max_fixtures"
    fi
}

# Export enhanced functions
export -f enhanced_test_lib::test_with_fixture
export -f enhanced_test_lib::enhanced_service_health_check  
export -f enhanced_test_lib::test_api_content_types
export -f enhanced_test_lib::test_service_load
export -f enhanced_test_lib::test_service_configuration
export -f enhanced_test_lib::register_enhanced_ai_tests
export -f enhanced_test_lib::register_enhanced_automation_tests
export -f enhanced_test_lib::test_ai_model_availability
export -f enhanced_test_lib::test_ai_generation_basic
export -f enhanced_test_lib::test_automation_workflow_list
export -f enhanced_test_lib::test_automation_execution_capability
export -f enhanced_test_lib::generate_enhanced_report
export -f enhanced_test_lib::enhanced_integration_test_main
export -f enhanced_test_lib::generate_fixture_report
export -f enhanced_test_lib::discover_resource_fixtures
export -f enhanced_test_lib::rotate_fixtures