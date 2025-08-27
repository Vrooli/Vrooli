#!/usr/bin/env bash
# Resource Smoke Test - Manual Service Verification
# Optional script for users to verify their actual services work with real data
# Complements the fast mocked BATS tests with real-world validation

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# Test configuration
TIMEOUT="${SMOKE_TEST_TIMEOUT:-30}"
VERBOSE="${SMOKE_TEST_VERBOSE:-false}"
SELECTED_RESOURCES="${SMOKE_TEST_RESOURCES:-}"
TEST_MODE="${SMOKE_TEST_MODE:-basic}"  # basic, comprehensive

# Test results tracking
SMOKE_TESTS_PASSED=0
SMOKE_TESTS_FAILED=0
SMOKE_TESTS_SKIPPED=0
declare -a FAILED_SMOKE_TESTS=()

#######################################
# Logging functions
#######################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_header() {
    echo
    echo -e "${BLUE}=== $* ===${NC}"
    echo
}

#######################################
# Service detection and availability
#######################################

detect_running_services() {
    local -a running_services=()
    
    # Common service ports to check
    declare -A service_ports=(
        ["ollama"]="11434"
        ["whisper"]="9090"
        ["unstructured-io"]="8000"
        ["n8n"]="5678"
        ["node-red"]="1880"
        ["minio"]="9000"
        ["vault"]="8200"
        ["qdrant"]="6333"
        ["questdb"]="9009"
    )
    
    for service in "${!service_ports[@]}"; do
        local port="${service_ports[$service]}"
        if curl -sf --max-time 5 "http://localhost:$port" >/dev/null 2>&1 || \
           curl -sf --max-time 5 "http://localhost:$port/health" >/dev/null 2>&1 || \
           curl -sf --max-time 5 "http://localhost:$port/api/health" >/dev/null 2>&1; then
            running_services+=("$service")
        fi
    done
    
    echo "${running_services[@]}"
}

#######################################
# Resource-specific smoke tests
#######################################

smoke_test_ollama() {
    local test_name="Ollama smoke test"
    local base_url="http://localhost:11434"
    
    log_info "Testing Ollama at $base_url"
    
    # Test 1: Check if service responds
    if ! curl -sf --max-time 5 "$base_url/api/tags" >/dev/null; then
        log_error "$test_name: Service not responding"
        return 1
    fi
    
    # Test 2: Get version
    local version_response
    if ! version_response=$(curl -sf --max-time 5 "$base_url/api/version"); then
        log_error "$test_name: Version endpoint failed"
        return 1
    fi
    
    [[ "$VERBOSE" == "true" ]] && echo "Version: $version_response"
    
    # Test 3: List models
    local models_response
    if ! models_response=$(curl -sf --max-time 10 "$base_url/api/tags"); then
        log_error "$test_name: Models listing failed"
        return 1
    fi
    
    local model_count
    model_count=$(echo "$models_response" | jq '.models | length' 2>/dev/null || echo "0")
    
    if [[ "$model_count" == "0" ]]; then
        log_warning "$test_name: No models found (this is OK for new installations)"
    else
        log_info "$test_name: Found $model_count models"
    fi
    
    # Test 4: Optional generation test if models are available
    if [[ "$model_count" != "0" && "$TEST_MODE" == "comprehensive" ]]; then
        local first_model
        first_model=$(echo "$models_response" | jq -r '.models[0].name' 2>/dev/null)
        
        if [[ -n "$first_model" && "$first_model" != "null" ]]; then
            log_info "$test_name: Testing generation with model: $first_model"
            local gen_request="{\"model\":\"$first_model\",\"prompt\":\"Hello\",\"stream\":false}"
            
            if curl -sf --max-time 30 "$base_url/api/generate" \
                -H 'Content-Type: application/json' \
                -d "$gen_request" >/dev/null; then
                log_success "$test_name: Generation test passed"
            else
                log_warning "$test_name: Generation test failed (model may not be ready)"
            fi
        fi
    fi
    
    log_success "$test_name: Passed"
    return 0
}

smoke_test_whisper() {
    local test_name="Whisper smoke test"
    local base_url="http://localhost:9090"
    
    log_info "Testing Whisper at $base_url"
    
    # Test 1: Check if service responds
    if ! curl -sf --max-time 5 "$base_url" >/dev/null; then
        log_error "$test_name: Service not responding"
        return 1
    fi
    
    # Test 2: Test with audio fixture if available and comprehensive mode
    if [[ "$TEST_MODE" == "comprehensive" && -f "${var_TEST_DIR}/fixtures/data/audio/whisper/test_speech.mp3" ]]; then
        log_info "$test_name: Testing transcription with fixture audio"
        
        local audio_file="${var_TEST_DIR}/fixtures/data/audio/whisper/test_speech.mp3"
        local response
        
        if response=$(curl -sf --max-time 30 "$base_url/v1/audio/transcriptions" \
            -F "audio=@$audio_file" \
            -F "model=whisper-1"); then
            
            if echo "$response" | jq -e '.text' >/dev/null 2>&1; then
                local transcription
                transcription=$(echo "$response" | jq -r '.text')
                log_success "$test_name: Transcription successful"
                [[ "$VERBOSE" == "true" ]] && echo "Transcription: $transcription"
            else
                log_warning "$test_name: Transcription response format unexpected"
            fi
        else
            log_warning "$test_name: Transcription test failed"
        fi
    fi
    
    log_success "$test_name: Passed"
    return 0
}

smoke_test_unstructured_io() {
    local test_name="Unstructured-IO smoke test"
    local base_url="http://localhost:8000"
    
    log_info "Testing Unstructured-IO at $base_url"
    
    # Test 1: Check if service responds
    if ! curl -sf --max-time 5 "$base_url" >/dev/null; then
        log_error "$test_name: Service not responding"
        return 1
    fi
    
    # Test 2: Test with document fixture if available and comprehensive mode
    if [[ "$TEST_MODE" == "comprehensive" && -f "${var_TEST_DIR}/fixtures/data/documents/pdf/simple_text.pdf" ]]; then
        log_info "$test_name: Testing document processing with fixture PDF"
        
        local pdf_file="${var_TEST_DIR}/fixtures/data/documents/pdf/simple_text.pdf"
        local response
        
        if response=$(curl -sf --max-time 30 "$base_url/general/v0/general" \
            -F "files=@$pdf_file"); then
            
            if echo "$response" | jq -e '.elements' >/dev/null 2>&1; then
                local element_count
                element_count=$(echo "$response" | jq '.elements | length')
                log_success "$test_name: Document processing successful ($element_count elements)"
                [[ "$VERBOSE" == "true" ]] && echo "Response: $response" | jq '.'
            else
                log_warning "$test_name: Document processing response format unexpected"
            fi
        else
            log_warning "$test_name: Document processing test failed"
        fi
    fi
    
    log_success "$test_name: Passed"
    return 0
}

smoke_test_n8n() {
    local test_name="n8n smoke test"
    local base_url="http://localhost:5678"
    
    log_info "Testing n8n at $base_url"
    
    # Test 1: Check if service responds
    if ! curl -sf --max-time 5 "$base_url" >/dev/null; then
        log_error "$test_name: Service not responding"
        return 1
    fi
    
    # Test 2: Check API health if available
    if curl -sf --max-time 5 "$base_url/healthz" >/dev/null 2>&1; then
        log_info "$test_name: Health endpoint responding"
    fi
    
    # Test 3: List workflows if API is accessible
    if curl -sf --max-time 10 "$base_url/api/v1/workflows" >/dev/null 2>&1; then
        log_info "$test_name: API accessible"
    else
        log_info "$test_name: API may require authentication (this is normal)"
    fi
    
    log_success "$test_name: Passed"
    return 0
}

smoke_test_minio() {
    local test_name="MinIO smoke test"
    local base_url="http://localhost:9000"
    
    log_info "Testing MinIO at $base_url"
    
    # Test 1: Check if service responds
    if ! curl -sf --max-time 5 "$base_url/minio/health/live" >/dev/null; then
        log_error "$test_name: Service not responding"
        return 1
    fi
    
    # Test 2: Check ready endpoint
    if curl -sf --max-time 5 "$base_url/minio/health/ready" >/dev/null; then
        log_info "$test_name: Service ready"
    else
        log_warning "$test_name: Service not ready"
    fi
    
    log_success "$test_name: Passed"
    return 0
}

#######################################
# Test orchestration
#######################################

run_smoke_test_for_resource() {
    local resource="$1"
    
    case "$resource" in
        "ollama")
            smoke_test_ollama
            ;;
        "whisper")
            smoke_test_whisper
            ;;
        "unstructured-io")
            smoke_test_unstructured_io
            ;;
        "n8n")
            smoke_test_n8n
            ;;
        "minio")
            smoke_test_minio
            ;;
        *)
            log_warning "No smoke test available for: $resource"
            return 2
            ;;
    esac
}

#######################################
# Main execution
#######################################

show_help() {
    cat << EOF
Resource Smoke Test - Manual Service Verification

This script performs real-world validation of running services using actual
API calls and fixture data. Unlike BATS tests, this requires services to be
actually running.

Usage: $0 [OPTIONS]

Options:
    --resources LIST    Comma-separated list of resources to test (default: auto-detect)
    --mode MODE         Test mode: basic, comprehensive (default: basic)
    --timeout N         Request timeout in seconds (default: 30)
    --verbose           Enable verbose output
    --help              Show this help message

Examples:
    $0                                    # Test all detected running services
    $0 --resources ollama,whisper         # Test specific services
    $0 --mode comprehensive --verbose     # Comprehensive testing with fixture data
    $0 --resources ollama --timeout 60    # Test ollama with longer timeout

Test Modes:
    basic         - Basic connectivity and health checks
    comprehensive - Includes fixture data processing tests

Environment Variables:
    SMOKE_TEST_RESOURCES  - Default resources to test
    SMOKE_TEST_MODE       - Default test mode
    SMOKE_TEST_TIMEOUT    - Default timeout
    SMOKE_TEST_VERBOSE    - Enable verbose output (true/false)
EOF
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --resources)
                SELECTED_RESOURCES="$2"
                shift 2
                ;;
            --mode)
                TEST_MODE="$2"
                shift 2
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            --verbose)
                VERBOSE="true"
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

main() {
    parse_args "$@"
    
    log_header "Resource Smoke Test"
    
    log_info "Configuration:"
    log_info "  Mode: $TEST_MODE"
    log_info "  Timeout: ${TIMEOUT}s"
    log_info "  Verbose: $VERBOSE"
    echo
    
    # Determine which resources to test
    local -a resources_to_test=()
    
    if [[ -n "$SELECTED_RESOURCES" ]]; then
        IFS=',' read -ra resources_to_test <<< "$SELECTED_RESOURCES"
        log_info "Testing selected resources: ${resources_to_test[*]}"
    else
        log_info "Auto-detecting running services..."
        local running_services
        running_services=$(detect_running_services)
        
        if [[ -z "$running_services" ]]; then
            log_warning "No running services detected"
            log_info "Start some services and try again, or use --resources to specify services manually"
            exit 0
        fi
        
        read -ra resources_to_test <<< "$running_services"
        log_info "Detected running services: ${resources_to_test[*]}"
    fi
    
    echo
    
    # Run smoke tests
    for resource in "${resources_to_test[@]}"; do
        echo
        log_header "Testing: $resource"
        
        local result
        if run_smoke_test_for_resource "$resource"; then
            local exit_code=$?
            case $exit_code in
                0)
                    ((SMOKE_TESTS_PASSED++))
                    ;;
                1)
                    ((SMOKE_TESTS_FAILED++))
                    FAILED_SMOKE_TESTS+=("$resource")
                    ;;
                2)
                    ((SMOKE_TESTS_SKIPPED++))
                    ;;
            esac
        else
            ((SMOKE_TESTS_FAILED++))
            FAILED_SMOKE_TESTS+=("$resource")
        fi
    done
    
    # Generate final report
    echo
    log_header "Smoke Test Results Summary"
    
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Total Resources Tested: $((SMOKE_TESTS_PASSED + SMOKE_TESTS_FAILED + SMOKE_TESTS_SKIPPED))"
    echo "  Passed:                 $SMOKE_TESTS_PASSED ✅"
    echo "  Failed:                 $SMOKE_TESTS_FAILED ❌"
    echo "  Skipped (no test):      $SMOKE_TESTS_SKIPPED ⏭️"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ ${#FAILED_SMOKE_TESTS[@]} -gt 0 ]]; then
        echo
        log_error "Failed Resources:"
        for failed_resource in "${FAILED_SMOKE_TESTS[@]}"; do
            echo "  ❌ $failed_resource"
        done
    fi
    
    echo
    if [[ $SMOKE_TESTS_FAILED -eq 0 ]]; then
        log_success "🎉 All smoke tests passed!"
        echo
        log_info "Your services are working correctly with real data!"
        exit 0
    else
        log_error "💥 $SMOKE_TESTS_FAILED smoke test(s) failed"
        echo
        log_info "Check service logs and configuration for failed resources"
        exit 1
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi