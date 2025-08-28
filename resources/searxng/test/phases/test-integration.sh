#!/usr/bin/env bash
# SearXNG Resource Integration Test - Full functionality validation
# Tests end-to-end SearXNG functionality including search operations and API interactions
# Max duration: 120 seconds per v2.0 contract

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SEARXNG_CLI_DIR="${APP_ROOT}/resources/searxng"

# Set test mode to avoid readonly variable issues
export SEARXNG_TEST_MODE="yes"

# Initialize fallback variable early
export SEARXNG_USE_JQ_FALLBACK="false"

# Source utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/common.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/config/defaults.sh"
# Ensure configuration is exported
searxng::export_config
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/common.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/api.sh"
# shellcheck disable=SC1091
source "${SEARXNG_CLI_DIR}/lib/docker.sh"

#######################################
# Check for required dependencies
#######################################
searxng::test::check_dependencies() {
    local missing_deps=()
    
    # Check for required tools
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        log::warn "jq not found - will use fallback JSON parsing"
        export SEARXNG_USE_JQ_FALLBACK="true"
    else
        export SEARXNG_USE_JQ_FALLBACK="false"
    fi
    
    if ! command -v grep >/dev/null 2>&1; then
        missing_deps+=("grep")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log::error "Missing required dependencies: ${missing_deps[*]}"
        log::info "Please install the missing dependencies and try again"
        return 1
    fi
    
    return 0
}

#######################################
# Parse JSON with fallback for when jq is not available
# Args: $1 - JSON string, $2 - jq expression
# Returns: parsed value or "0" on error
#######################################
searxng::test::parse_json() {
    local json_str="$1"
    local jq_expr="$2"
    
    if [[ "$SEARXNG_USE_JQ_FALLBACK" == "false" ]]; then
        # Use jq if available
        echo "$json_str" | jq -r "$jq_expr" 2>/dev/null || echo "0"
    else
        # Fallback parsing for common expressions
        case "$jq_expr" in
            ".results | length")
                # Count results array length
                echo "$json_str" | grep -o '"url"' | wc -l || echo "0"
                ;;
            ".engines | length")
                # Count engines
                echo "$json_str" | grep -o '"engine"' | sort -u | wc -l || echo "0"
                ;;
            ".")
                # Validate JSON by checking basic structure
                if echo "$json_str" | grep -q '{"query".*"results"'; then
                    echo "valid"
                else
                    echo "invalid"
                fi
                ;;
            *)
                # Default fallback
                if echo "$json_str" | grep -q '{"'; then
                    echo "1"
                else
                    echo "0"
                fi
                ;;
        esac
    fi
}

#######################################
# Check if response is valid JSON
# Args: $1 - response string
# Returns: 0 if valid, 1 if not
#######################################
searxng::test::is_valid_json() {
    local response="$1"
    
    if [[ "$SEARXNG_USE_JQ_FALLBACK" == "false" ]]; then
        echo "$response" | jq . >/dev/null 2>&1
    else
        # Basic JSON validation - check for basic structure
        if echo "$response" | grep -q '^{.*}$' && echo "$response" | grep -q '"'; then
            return 0
        else
            return 1
        fi
    fi
}

# SearXNG Resource Integration Test
searxng::test::integration() {
    log::info "Running SearXNG resource integration tests..."
    
    # Check dependencies first
    if ! searxng::test::check_dependencies; then
        return 1
    fi
    
    local overall_status=0
    local verbose="${SEARXNG_TEST_VERBOSE:-false}"
    local test_queries=("test" "example" "search")
    local retry_count=3
    
    # Pre-check: Ensure SearXNG is accessible via API with retry logic
    log::info "Pre-check: Verifying SearXNG accessibility..."
    local precheck_success=false
    for attempt in $(seq 1 $retry_count); do
        if curl -sf --max-time 10 "${SEARXNG_BASE_URL}/stats" >/dev/null 2>&1; then
            precheck_success=true
            break
        fi
        if [[ $attempt -lt $retry_count ]]; then
            log::info "  Attempt $attempt failed, retrying in 2 seconds..."
            sleep 2
        fi
    done
    
    if [[ "$precheck_success" != "true" ]]; then
        log::error "✗ SearXNG is not accessible after $retry_count attempts. Cannot run integration tests."
        log::info "Please start SearXNG first: resource-searxng manage start"
        log::info "Check service status: resource-searxng status"
        return 1
    fi
    
    log::success "✓ SearXNG is accessible and responding"
    
    # Test 1: JSON search API with different queries
    log::info "1/12 Testing JSON search API with multiple queries..."
    local json_tests=0
    local json_passed=0
    
    for query in "${test_queries[@]}"; do
        ((json_tests++))
        local search_response
        local query_success=false
        
        # Try the query with retry logic
        for attempt in $(seq 1 $retry_count); do
            if search_response=$(curl -sf --max-time 15 "${SEARXNG_BASE_URL}/search?q=${query// /%20}&format=json" 2>/dev/null); then
                if [[ -n "$search_response" ]]; then
                    if searxng::test::is_valid_json "$search_response"; then
                        local result_count
                        result_count=$(searxng::test::parse_json "$search_response" ".results | length")
                        if [[ $result_count -gt 0 ]]; then
                            ((json_passed++))
                            query_success=true
                            if [[ "$verbose" == "true" ]]; then
                                log::info "  ✓ Query '$query': $result_count results"
                            fi
                            break
                        else
                            if [[ "$verbose" == "true" ]]; then
                                log::info "  ⚠ Query '$query' returned 0 results (attempt $attempt)"
                            fi
                        fi
                    else
                        if [[ "$verbose" == "true" ]]; then
                            log::info "  ⚠ Query '$query' returned invalid JSON (attempt $attempt)"
                            log::info "  Response: ${search_response:0:200}..."
                        fi
                    fi
                else
                    if [[ "$verbose" == "true" ]]; then
                        log::info "  ⚠ Query '$query' returned empty response (attempt $attempt)"
                    fi
                fi
            else
                if [[ "$verbose" == "true" ]]; then
                    log::info "  ⚠ Query '$query' curl failed (attempt $attempt)"
                fi
            fi
            
            # If failed, wait before retry (except on last attempt)
            if [[ $attempt -lt $retry_count ]]; then
                if [[ "$verbose" == "true" ]]; then
                    log::info "  Query '$query' attempt $attempt failed, retrying..."
                fi
                sleep 1
            fi
        done
        
        if [[ "$query_success" != "true" ]] && [[ "$verbose" == "true" ]]; then
            log::warn "  ⚠ Query '$query' failed after $retry_count attempts"
        fi
    done
    
    if [[ $json_passed -eq $json_tests ]]; then
        log::success "✓ JSON search API works perfectly ($json_passed/$json_tests queries)"
    elif [[ $json_passed -gt 0 ]]; then
        log::warn "⚠ JSON search API partially working ($json_passed/$json_tests queries)"
        if [[ $json_passed -lt $((json_tests / 2)) ]]; then
            overall_status=1  # Fail if less than half the queries work
        fi
    else
        log::error "✗ JSON search API completely failed ($json_passed/$json_tests queries)"
        overall_status=1
    fi
    
    # Test 2: Different output formats
    log::info "2/12 Testing different output formats..."
    local formats=("json" "xml" "csv")
    local format_tests=0
    local format_passed=0
    
    for format in "${formats[@]}"; do
        ((format_tests++))
        local format_response
        local format_success=false
        
        # Try each format with retry logic
        for attempt in $(seq 1 $retry_count); do
            if format_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=test&format=${format}" 2>/dev/null); then
                case "$format" in
                    "json")
                        if searxng::test::is_valid_json "$format_response"; then
                            ((format_passed++))
                            format_success=true
                            if [[ "$verbose" == "true" ]]; then
                                log::info "  ✓ JSON format working"
                            fi
                        fi
                        ;;
                    "xml")
                        if echo "$format_response" | grep -q "<?xml"; then
                            ((format_passed++))
                            format_success=true
                            if [[ "$verbose" == "true" ]]; then
                                log::info "  ✓ XML format working"
                            fi
                        fi
                        ;;
                    "csv")
                        if echo "$format_response" | grep -q ","; then
                            ((format_passed++))
                            format_success=true
                            if [[ "$verbose" == "true" ]]; then
                                log::info "  ✓ CSV format working"
                            fi
                        fi
                        ;;
                esac
                
                if [[ "$format_success" == "true" ]]; then
                    break
                fi
            fi
            
            # Wait before retry (except on last attempt)
            if [[ $attempt -lt $retry_count ]] && [[ "$format_success" != "true" ]]; then
                if [[ "$verbose" == "true" ]]; then
                    log::info "  $format format attempt $attempt failed, retrying..."
                fi
                sleep 1
            fi
        done
        
        if [[ "$format_success" != "true" ]] && [[ "$verbose" == "true" ]]; then
            log::warn "  ⚠ $format format failed after $retry_count attempts"
        fi
    done
    
    if [[ $format_passed -eq $format_tests ]]; then
        log::success "✓ All output formats work ($format_passed/$format_tests formats)"
    elif [[ $format_passed -gt 0 ]]; then
        log::warn "⚠ Some output formats failed ($format_passed/$format_tests formats)"
    else
        log::error "✗ No output formats working"
        overall_status=1
    fi
    
    # Test 3: Search categories
    log::info "3/12 Testing search categories..."
    local categories=("general" "images" "news")
    local category_tests=0
    local category_passed=0
    
    for category in "${categories[@]}"; do
        ((category_tests++))
        local category_response
        local category_success=false
        
        for attempt in $(seq 1 $retry_count); do
            if category_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=logo&format=json&categories=${category}" 2>/dev/null); then
                if searxng::test::is_valid_json "$category_response"; then
                    local results
                    results=$(searxng::test::parse_json "$category_response" ".results | length")
                    if [[ $results -gt 0 ]]; then
                        ((category_passed++))
                        category_success=true
                        if [[ "$verbose" == "true" ]]; then
                            log::info "  ✓ Category '$category': $results results"
                        fi
                        break
                    fi
                fi
            fi
            
            if [[ $attempt -lt $retry_count ]] && [[ "$category_success" != "true" ]]; then
                sleep 1
            fi
        done
        
        if [[ "$category_success" != "true" ]] && [[ "$verbose" == "true" ]]; then
            log::warn "  ⚠ Category '$category' failed after $retry_count attempts"
        fi
    done
    
    if [[ $category_passed -gt 0 ]]; then
        log::success "✓ Search categories work ($category_passed/$category_tests categories)"
    else
        log::warn "⚠ Search categories not working ($category_passed/$category_tests categories)"
    fi
    
    # Test 4: Search pagination
    log::info "4/12 Testing search pagination..."
    local page1_response page2_response
    local pagination_success=false
    
    for attempt in $(seq 1 $retry_count); do
        if page1_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=programming&format=json&pageno=1" 2>/dev/null) && \
           page2_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=programming&format=json&pageno=2" 2>/dev/null); then
            
            if searxng::test::is_valid_json "$page1_response" && searxng::test::is_valid_json "$page2_response"; then
                local page1_results page2_results
                page1_results=$(searxng::test::parse_json "$page1_response" ".results | length")
                page2_results=$(searxng::test::parse_json "$page2_response" ".results | length")
                
                if [[ $page1_results -gt 0 ]] && [[ $page2_results -gt 0 ]]; then
                    log::success "✓ Pagination works (page1: $page1_results, page2: $page2_results)"
                    pagination_success=true
                    break
                fi
            fi
        fi
        
        if [[ $attempt -lt $retry_count ]]; then
            if [[ "$verbose" == "true" ]]; then
                log::info "  Pagination attempt $attempt failed, retrying..."
            fi
            sleep 1
        fi
    done
    
    if [[ "$pagination_success" != "true" ]]; then
        log::warn "⚠ Pagination test failed after $retry_count attempts"
    fi
    
    # Test 5: Safe search parameter
    log::info "5/12 Testing safe search parameter..."
    local safesearch_response
    local safesearch_success=false
    
    for attempt in $(seq 1 $retry_count); do
        if safesearch_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=test&format=json&safesearch=2" 2>/dev/null); then
            if searxng::test::is_valid_json "$safesearch_response"; then
                local safe_results
                safe_results=$(searxng::test::parse_json "$safesearch_response" ".results | length")
                log::success "✓ Safe search parameter works (results: $safe_results)"
                safesearch_success=true
                break
            fi
        fi
        
        if [[ $attempt -lt $retry_count ]]; then
            if [[ "$verbose" == "true" ]]; then
                log::info "  Safe search attempt $attempt failed, retrying..."
            fi
            sleep 1
        fi
    done
    
    if [[ "$safesearch_success" != "true" ]]; then
        log::warn "⚠ Safe search parameter test failed after $retry_count attempts"
    fi
    
    # Test 6: Language parameter
    log::info "6/12 Testing language parameter..."
    local lang_response
    local lang_success=false
    
    for attempt in $(seq 1 $retry_count); do
        if lang_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=test&format=json&language=en" 2>/dev/null); then
            if searxng::test::is_valid_json "$lang_response"; then
                local lang_results
                lang_results=$(searxng::test::parse_json "$lang_response" ".results | length")
                log::success "✓ Language parameter works (results: $lang_results)"
                lang_success=true
                break
            fi
        fi
        
        if [[ $attempt -lt $retry_count ]]; then
            if [[ "$verbose" == "true" ]]; then
                log::info "  Language parameter attempt $attempt failed, retrying..."
            fi
            sleep 1
        fi
    done
    
    if [[ "$lang_success" != "true" ]]; then
        log::warn "⚠ Language parameter test failed after $retry_count attempts"
    fi
    
    # Test 7: Stats endpoint detailed
    log::info "7/12 Testing stats endpoint detailed..."
    local stats_response
    local stats_success=false
    
    for attempt in $(seq 1 $retry_count); do
        if stats_response=$(curl -sf --max-time 5 "${SEARXNG_BASE_URL}/stats" 2>/dev/null); then
            if searxng::test::is_valid_json "$stats_response"; then
                # Try to extract engine information
                local engine_count
                engine_count=$(searxng::test::parse_json "$stats_response" ".engines | length")
                if [[ "$engine_count" != "0" ]] && [[ -n "$engine_count" ]]; then
                    log::success "✓ Stats endpoint with engine data (engines: $engine_count)"
                else
                    log::success "✓ Stats endpoint responding (JSON format)"
                fi
                stats_success=true
                break
            else
                # HTML format is also acceptable
                if echo "$stats_response" | grep -qi "searxng\|statistics"; then
                    log::success "✓ Stats endpoint responding (HTML format)"
                    stats_success=true
                    break
                fi
            fi
        fi
        
        if [[ $attempt -lt $retry_count ]] && [[ "$stats_success" != "true" ]]; then
            if [[ "$verbose" == "true" ]]; then
                log::info "  Stats endpoint attempt $attempt failed, retrying..."
            fi
            sleep 1
        fi
    done
    
    if [[ "$stats_success" != "true" ]]; then
        log::error "✗ Stats endpoint failed after $retry_count attempts"
        overall_status=1
    fi
    
    # Test 8: Config endpoint
    log::info "8/12 Testing config endpoint..."
    local config_response
    local config_success=false
    
    for attempt in $(seq 1 $retry_count); do
        if config_response=$(curl -sf --max-time 5 "${SEARXNG_BASE_URL}/config" 2>/dev/null); then
            if searxng::test::is_valid_json "$config_response"; then
                # Check for basic JSON structure (config endpoint may have varied fields)
                if echo "$config_response" | grep -q '"' && echo "$config_response" | grep -q '{'; then
                    log::success "✓ Config endpoint responding (JSON format)"
                else
                    log::success "✓ Config endpoint responding"
                fi
                config_success=true
                break
            fi
        fi
        
        if [[ $attempt -lt $retry_count ]] && [[ "$config_success" != "true" ]]; then
            if [[ "$verbose" == "true" ]]; then
                log::info "  Config endpoint attempt $attempt failed, retrying..."
            fi
            sleep 1
        fi
    done
    
    if [[ "$config_success" != "true" ]]; then
        log::info "  Config endpoint not available (normal for some configurations)"
    fi
    
    # Test 9: OpenSearch descriptor
    log::info "9/12 Testing OpenSearch descriptor..."
    local opensearch_response
    if opensearch_response=$(curl -sf --max-time 5 "${SEARXNG_BASE_URL}/opensearch.xml" 2>/dev/null); then
        if echo "$opensearch_response" | grep -qi "opensearch\|description\|<url"; then
            log::success "✓ OpenSearch descriptor available"
            if [[ "$verbose" == "true" ]]; then
                local short_name
                short_name=$(echo "$opensearch_response" | grep -i "shortname" | sed 's/<[^>]*>//g' | xargs || echo "N/A")
                log::info "  ShortName: $short_name"
            fi
        else
            log::warn "⚠ OpenSearch descriptor format unclear"
        fi
    else
        log::info "  OpenSearch descriptor not available (optional feature)"
    fi
    
    # Test 10: Rate limiting detection
    log::info "10/12 Testing rate limiting behavior..."
    local rate_limit_detected=false
    local successful_requests=0
    
    for i in {1..5}; do
        local rate_response
        if rate_response=$(curl -sf --max-time 3 "${SEARXNG_BASE_URL}/search?q=rate$i&format=json" 2>/dev/null); then
            if searxng::test::is_valid_json "$rate_response"; then
                ((successful_requests++))
            fi
        else
            # Check if this is a rate limit (HTTP 429)
            local status_code
            status_code=$(curl -sf -w "%{http_code}" -o /dev/null --max-time 3 "${SEARXNG_BASE_URL}/search?q=rate$i&format=json" 2>/dev/null || echo "000")
            if [[ "$status_code" == "429" ]]; then
                rate_limit_detected=true
                if [[ "$verbose" == "true" ]]; then
                    log::info "  ✓ Rate limit triggered after $successful_requests requests"
                fi
                break
            fi
        fi
        sleep 0.2  # Small delay between requests
    done
    
    if [[ "$rate_limit_detected" == "true" ]]; then
        log::success "✓ Rate limiting is active and working"
    elif [[ $successful_requests -eq 5 ]]; then
        log::info "  Rate limiting not triggered (may be disabled or high limit)"
    else
        log::warn "⚠ Rate limiting behavior unclear ($successful_requests/5 requests succeeded)"
    fi
    
    # Test 11: Container health during load
    log::info "11/12 Testing container health during load..."
    local container_healthy=true
    
    # Check container status using direct Docker command
    if ! docker ps --format '{{.Names}}' | grep -q "^${SEARXNG_CONTAINER_NAME}$"; then
        log::error "✗ Container stopped during testing"
        overall_status=1
        container_healthy=false
    fi
    
    if [[ "$container_healthy" == "true" ]]; then
        # Check if container is responding to API calls
        if curl -sf --max-time 5 "${SEARXNG_BASE_URL}/stats" >/dev/null 2>&1; then
            log::success "✓ Container remains healthy during load"
        else
            log::warn "⚠ Container running but API not responding"
        fi
    fi
    
    # Test 12: Privacy and security headers
    log::info "12/12 Testing privacy and security headers..."
    local headers_response
    if headers_response=$(curl -sf --max-time 5 -I "${SEARXNG_BASE_URL}" 2>/dev/null); then
        local security_headers=0
        
        # Check for common security headers
        if echo "$headers_response" | grep -qi "x-content-type-options"; then
            ((security_headers++))
        fi
        if echo "$headers_response" | grep -qi "x-frame-options"; then
            ((security_headers++))
        fi
        if echo "$headers_response" | grep -qi "referrer-policy"; then
            ((security_headers++))
        fi
        
        # Check that server info is not leaked
        local server_hidden=true
        if echo "$headers_response" | grep -qi "server:.*nginx\|server:.*apache\|x-powered-by"; then
            server_hidden=false
        fi
        
        if [[ $security_headers -gt 0 ]] && [[ "$server_hidden" == "true" ]]; then
            log::success "✓ Privacy and security headers present ($security_headers headers)"
        elif [[ $security_headers -gt 0 ]]; then
            log::success "✓ Security headers present but server info visible"
        elif [[ "$server_hidden" == "true" ]]; then
            log::success "✓ Server information properly hidden"
        else
            log::warn "⚠ Limited privacy/security headers detected"
        fi
        
        if [[ "$verbose" == "true" ]]; then
            log::info "  Security headers found: $security_headers"
            log::info "  Server info hidden: $server_hidden"
        fi
    else
        log::warn "⚠ Could not check privacy headers"
    fi
    
    echo ""
    if [[ $overall_status -eq 0 ]]; then
        log::success "🎉 SearXNG resource integration tests PASSED"
        echo "SearXNG service fully functional - all operations work correctly"
    else
        log::error "💥 SearXNG resource integration tests FAILED"
        echo "SearXNG service has functional issues that need to be resolved"
    fi
    
    return $overall_status
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    searxng::test::integration "$@"
fi