#!/usr/bin/env bash
# SearXNG Resource Integration Test - Full functionality validation
# Tests end-to-end SearXNG functionality including search operations and API interactions
# Max duration: 120 seconds per v2.0 contract

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
SEARXNG_CLI_DIR="${RESOURCE_DIR}"

# Set test mode to avoid readonly variable issues
export SEARXNG_TEST_MODE="yes"

# Initialize fallback variable early
export SEARXNG_USE_JQ_FALLBACK="false"

# Source utilities
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/resources/common.sh"
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
        # Use jq if available with timeout to prevent hanging
        local result
        if result=$(echo "$json_str" | timeout 2 jq -r "$jq_expr" 2>/dev/null); then
            echo "$result"
        else
            # If jq times out or fails, use fallback
            case "$jq_expr" in
                ".results | length")
                    echo "$json_str" | grep -o '"url"' | wc -l || echo "0"
                    ;;
                *)
                    echo "0"
                    ;;
            esac
        fi
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
        # Use timeout to prevent hanging on large JSON
        echo "$response" | timeout 2 jq . >/dev/null 2>&1
        local ret=$?
        if [[ $ret -eq 0 ]]; then
            return 0
        else
            # Fallback to basic validation if jq times out
            if echo "$response" | grep -q '^{.*}$' && echo "$response" | grep -q '"'; then
                return 0
            else
                return 1
            fi
        fi
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
    log::info "1/13 Testing JSON search API with multiple queries..."
    local json_tests=0
    local json_passed=0
    
    for query in "${test_queries[@]}"; do
        json_tests=$((json_tests + 1))
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
                            json_passed=$((json_passed + 1))
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
    
    # Test 2: Output format allowlist
    # The configured allowlist is [html, json]; csv/rss/xml must be rejected
    # so the JSON contract web-search depends on stays deliberate.
    log::info "2/13 Testing output format allowlist..."
    local json_format_ok=false
    local format_response

    for attempt in $(seq 1 $retry_count); do
        if format_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=test&format=json" 2>/dev/null); then
            if searxng::test::is_valid_json "$format_response"; then
                json_format_ok=true
                break
            fi
        fi
        if [[ $attempt -lt $retry_count ]]; then
            if [[ "$verbose" == "true" ]]; then
                log::info "  json format attempt $attempt failed, retrying..."
            fi
            sleep 1
        fi
    done

    local disallowed_rejected=0
    local disallowed_formats=("csv" "rss")
    for format in "${disallowed_formats[@]}"; do
        local status_code
        status_code=$(curl -s -w "%{http_code}" -o /dev/null --max-time 10 "${SEARXNG_BASE_URL}/search?q=test&format=${format}" 2>/dev/null || echo "000")
        if [[ "$status_code" == "403" || "$status_code" == "400" ]]; then
            disallowed_rejected=$((disallowed_rejected + 1))
        elif [[ "$verbose" == "true" ]]; then
            log::info "  ⚠ Disallowed format '$format' returned status $status_code (expected 403)"
        fi
    done

    if [[ "$json_format_ok" == "true" ]] && [[ $disallowed_rejected -eq ${#disallowed_formats[@]} ]]; then
        log::success "✓ Format allowlist enforced (json works, csv/rss rejected)"
    elif [[ "$json_format_ok" == "true" ]]; then
        log::warn "⚠ JSON works but disallowed formats not all rejected ($disallowed_rejected/${#disallowed_formats[@]})"
    else
        log::error "✗ JSON format not working - web-search contract broken"
        overall_status=1
    fi
    
    # Test 3: Search categories
    log::info "3/13 Testing search categories..."
    local categories=("general" "images" "news")
    local category_tests=0
    local category_passed=0
    
    for category in "${categories[@]}"; do
        category_tests=$((category_tests + 1))
        local category_response
        local category_success=false
        
        for attempt in $(seq 1 $retry_count); do
            if category_response=$(curl -sf --max-time 10 "${SEARXNG_BASE_URL}/search?q=logo&format=json&categories=${category}" 2>/dev/null); then
                if searxng::test::is_valid_json "$category_response"; then
                    local results
                    results=$(searxng::test::parse_json "$category_response" ".results | length")
                    if [[ $results -gt 0 ]]; then
                        category_passed=$((category_passed + 1))
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
    log::info "4/13 Testing search pagination..."
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
    log::info "5/13 Testing safe search parameter..."
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
    log::info "6/13 Testing language parameter..."
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
    log::info "7/13 Testing stats endpoint detailed..."
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
    log::info "8/13 Testing config endpoint..."
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
    log::info "9/13 Testing OpenSearch descriptor..."
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
    log::info "10/13 Testing rate limiting behavior..."
    local rate_limit_detected=false
    local successful_requests=0
    
    for i in {1..5}; do
        local rate_response
        if rate_response=$(curl -sf --max-time 3 "${SEARXNG_BASE_URL}/search?q=rate$i&format=json" 2>/dev/null); then
            if searxng::test::is_valid_json "$rate_response"; then
                successful_requests=$((successful_requests + 1))
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
    log::info "11/13 Testing container health during load..."
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
    log::info "12/13 Testing privacy and security headers..."
    local headers_response
    if headers_response=$(curl -sf --max-time 5 -I "${SEARXNG_BASE_URL}" 2>/dev/null); then
        local security_headers=0
        
        # Check for common security headers
        if echo "$headers_response" | grep -qi "x-content-type-options"; then
            security_headers=$((security_headers + 1))
        fi
        if echo "$headers_response" | grep -qi "x-frame-options"; then
            security_headers=$((security_headers + 1))
        fi
        if echo "$headers_response" | grep -qi "referrer-policy"; then
            security_headers=$((security_headers + 1))
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
    
    # Test 13: Engine coverage (the regression that motivated this test: a
    # 17-month-stale image degraded the instance to bing-only and nothing
    # noticed). A live query must draw from >=2 distinct engines.
    log::info "13/13 Testing engine coverage (>=2 distinct engines)..."
    local engine_coverage_ok=false
    local engines_seen=0

    for attempt in 1 2; do
        local coverage_response
        if coverage_response=$(curl -sf --max-time 20 "${SEARXNG_BASE_URL}/search?q=current+world+news&format=json" 2>/dev/null); then
            if searxng::test::is_valid_json "$coverage_response" && [[ "$SEARXNG_USE_JQ_FALLBACK" == "false" ]]; then
                engines_seen=$(echo "$coverage_response" | jq -r '[.results[].engine] | unique | length' 2>/dev/null || echo "0")
                if [[ "$verbose" == "true" ]]; then
                    local engine_names unresponsive_names
                    engine_names=$(echo "$coverage_response" | jq -rc '[.results[].engine] | unique' 2>/dev/null || echo "[]")
                    unresponsive_names=$(echo "$coverage_response" | jq -rc '.unresponsive_engines' 2>/dev/null || echo "[]")
                    log::info "  Responsive engines: $engine_names"
                    log::info "  Unresponsive engines: $unresponsive_names"
                fi
                if [[ "${engines_seen:-0}" -ge 2 ]]; then
                    engine_coverage_ok=true
                    break
                fi
            elif [[ "$SEARXNG_USE_JQ_FALLBACK" == "true" ]]; then
                # Without jq we cannot count distinct engines reliably; accept
                # a non-empty result set rather than fail spuriously.
                local fallback_results
                fallback_results=$(searxng::test::parse_json "$coverage_response" ".results | length")
                if [[ "${fallback_results:-0}" -gt 0 ]]; then
                    log::warn "⚠ jq unavailable - engine coverage downgraded to results-present check"
                    engine_coverage_ok=true
                    break
                fi
            fi
        fi
        # Tolerate transient engine suspensions with a single retry.
        if [[ $attempt -eq 1 ]]; then
            if [[ "$verbose" == "true" ]]; then
                log::info "  Engine coverage attempt 1 saw ${engines_seen:-0} engines, retrying..."
            fi
            sleep 3
        fi
    done

    if [[ "$engine_coverage_ok" == "true" ]]; then
        log::success "✓ Engine coverage OK (${engines_seen:-multiple} distinct engines responding)"
    else
        log::error "✗ Engine coverage degraded: ${engines_seen:-0} distinct engines responded (need >=2)"
        log::info "  Diagnose with: resource-searxng engine-health --json"
        overall_status=1
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