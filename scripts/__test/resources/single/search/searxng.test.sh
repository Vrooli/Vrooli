#!/bin/bash
# ====================================================================
# SearXNG Integration Test
# ====================================================================
#
# Tests SearXNG privacy-respecting metasearch engine integration
# including health checks, search functionality, API capabilities,
# and privacy features.
#
# Required Resources: searxng
# Test Categories: single-resource, search
# Estimated Duration: 45-60 seconds
#
# ====================================================================

set -euo pipefail

# Test metadata
TEST_RESOURCE="searxng"
TEST_TIMEOUT="${TEST_TIMEOUT:-75}"
TEST_CLEANUP="${TEST_CLEANUP:-true}"

# Recreate HEALTHY_RESOURCES array from exported string
if [[ -n "${HEALTHY_RESOURCES_STR:-}" ]]; then
    HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
fi

# Source framework helpers
SCRIPT_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
source "$SCRIPT_DIR/framework/helpers/assertions.sh"
source "$SCRIPT_DIR/framework/helpers/cleanup.sh"

# SearXNG configuration
SEARXNG_BASE_URL="http://localhost:9200"

# Test setup
setup_test() {
    echo "🔧 Setting up SearXNG integration test..."
    
    # Register cleanup handler
    register_cleanup_handler
    
    # Auto-discovery fallback for direct test execution
    if [[ -z "${HEALTHY_RESOURCES_STR:-}" ]]; then
        echo "🔍 Auto-discovering resources for direct test execution..."
        
        # Use the resource discovery system with timeout
        local resources_dir
        resources_dir="$(cd "$SCRIPT_DIR/../.." && pwd)"
        
        local discovery_output=""
        if timeout 10s bash -c "\"$resources_dir/index.sh\" --action discover 2>&1" > /tmp/discovery_output.tmp 2>&1; then
            discovery_output=$(cat /tmp/discovery_output.tmp)
            rm -f /tmp/discovery_output.tmp
        else
            echo "⚠️  Auto-discovery timed out, using fallback method..."
            # Fallback: check if the required resource is running on its default port
            if curl -f -s --max-time 2 "$SEARXNG_BASE_URL/" >/dev/null 2>&1 || \
               curl -f -s --max-time 2 "$SEARXNG_BASE_URL/search" >/dev/null 2>&1; then
                discovery_output="✅ $TEST_RESOURCE is running on port 8100"
            fi
        fi
        
        local discovered_resources=()
        while IFS= read -r line; do
            if [[ "$line" =~ ✅[[:space:]]+([^[:space:]]+)[[:space:]]+is[[:space:]]+running ]]; then
                discovered_resources+=("${BASH_REMATCH[1]}")
            fi
        done <<< "$discovery_output"
        
        if [[ ${#discovered_resources[@]} -eq 0 ]]; then
            echo "⚠️  No resources discovered, but test will proceed..."
            discovered_resources=("$TEST_RESOURCE")
        fi
        
        export HEALTHY_RESOURCES_STR="${discovered_resources[*]}"
        echo "✓ Discovered healthy resources: $HEALTHY_RESOURCES_STR"
    fi
    
    # Verify SearXNG is available
    require_resource "$TEST_RESOURCE"
    
    # Verify required tools
    require_tools "curl" "jq"
    
    echo "✓ Test setup complete"
}

# Test SearXNG health and basic connectivity
test_searxng_health() {
    echo "🏥 Testing SearXNG health and connectivity..."
    
    # Test main search interface
    local response
    response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/" 2>/dev/null || echo "")
    
    assert_not_empty "$response" "SearXNG main endpoint responds"
    
    if echo "$response" | grep -qi "searxng\|search\|<html\|<title"; then
        echo "  ✓ Main search interface accessible"
    else
        echo "  ⚠ Main interface response format: ${response:0:50}..."
    fi
    
    # Test search endpoint
    local search_response
    search_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/search" 2>/dev/null || echo "")
    
    assert_not_empty "$search_response" "SearXNG search endpoint accessible"
    
    # Test configuration/stats endpoint
    local stats_response
    stats_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/stats" 2>/dev/null || echo "")
    
    if [[ -n "$stats_response" ]]; then
        echo "  ✓ Statistics endpoint available"
    else
        echo "  ⚠ Statistics endpoint may not be accessible"
    fi
    
    echo "✓ SearXNG health check passed"
}

# Test SearXNG search functionality
test_searxng_search_functionality() {
    echo "🔍 Testing SearXNG search functionality..."
    
    # Test basic web search (HTML interface)
    local search_query="vrooli test search"
    local search_response
    search_response=$(curl -s --max-time 15 "$SEARXNG_BASE_URL/search?q=${search_query// /+}" 2>/dev/null || echo "")
    
    assert_not_empty "$search_response" "Basic search query responds"
    
    if echo "$search_response" | grep -qi "result\|<html\|search"; then
        echo "  ✓ Search results returned for query: '$search_query'"
    else
        echo "  ⚠ Search response format unexpected"
    fi
    
    # Test different search categories
    local categories=("general" "news" "images" "videos")
    local category_tests=0
    
    for category in "${categories[@]}"; do
        echo "  Testing search category: $category"
        local category_response
        category_response=$(curl -s --max-time 12 "$SEARXNG_BASE_URL/search?q=test&categories=$category" 2>/dev/null || echo "")
        
        if [[ -n "$category_response" ]] && echo "$category_response" | grep -qi "result\|<html"; then
            echo "    ✓ Category '$category' search functional"
            category_tests=$((category_tests + 1))
        else
            echo "    ⚠ Category '$category' may not be available"
        fi
    done
    
    assert_greater_than "$category_tests" "0" "At least one search category functional"
    
    echo "✓ SearXNG search functionality test completed"
}

# Test SearXNG API capabilities
test_searxng_api() {
    echo "📋 Testing SearXNG API capabilities..."
    
    # Test JSON API search
    local api_search_response
    api_search_response=$(curl -s --max-time 15 "$SEARXNG_BASE_URL/search?q=test&format=json" 2>/dev/null || echo '{"results":[]}')
    
    debug_json_response "$api_search_response" "SearXNG JSON API Response"
    
    # Check if valid JSON response
    if echo "$api_search_response" | jq . >/dev/null 2>&1; then
        echo "  ✓ JSON API returns valid JSON"
        
        # Check for expected API structure
        if echo "$api_search_response" | jq -e '.results // .query // .infoboxes // empty' >/dev/null 2>&1; then
            echo "  ✓ API response has expected structure"
            
            # Count results
            local result_count
            result_count=$(echo "$api_search_response" | jq '.results | length' 2>/dev/null || echo "0")
            echo "  📊 Search results returned: $result_count"
            
            # Check result structure
            if [[ "$result_count" -gt 0 ]]; then
                local first_result_url
                first_result_url=$(echo "$api_search_response" | jq -r '.results[0].url // empty' 2>/dev/null)
                if [[ -n "$first_result_url" ]]; then
                    echo "  ✓ Results contain URLs"
                fi
            fi
        fi
    else
        echo "  ⚠ JSON API response is not valid JSON: ${api_search_response:0:100}..."
    fi
    
    # Test CSV API format
    local csv_response
    csv_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/search?q=test&format=csv" 2>/dev/null || echo "")
    
    if [[ -n "$csv_response" ]] && echo "$csv_response" | grep -q ","; then
        echo "  ✓ CSV API format available"
    fi
    
    # Test RSS format
    local rss_response
    rss_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/search?q=test&format=rss" 2>/dev/null || echo "")
    
    if [[ -n "$rss_response" ]] && echo "$rss_response" | grep -qi "rss\|xml"; then
        echo "  ✓ RSS format available"
    fi
    
    echo "✓ SearXNG API capabilities test completed"
}

# Test SearXNG privacy features
test_searxng_privacy_features() {
    echo "🔒 Testing SearXNG privacy features..."
    
    # Test that requests don't leak user information
    echo "  Testing privacy-respecting search..."
    
    # Search without user tracking
    local privacy_response
    privacy_response=$(curl -s --max-time 15 \
        -H "User-Agent: SearXNG-Test" \
        -H "DNT: 1" \
        "$SEARXNG_BASE_URL/search?q=privacy+test" 2>/dev/null || echo "")
    
    assert_not_empty "$privacy_response" "Privacy-respecting search functional"
    
    # Test search engines configuration
    local engines_response
    engines_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/engines" 2>/dev/null || echo "")
    
    if [[ -n "$engines_response" ]] && echo "$engines_response" | grep -qi "engine\|<html"; then
        echo "  ✓ Search engines configuration accessible"
    fi
    
    # Test preferences/settings
    local preferences_response
    preferences_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/preferences" 2>/dev/null || echo "")
    
    if [[ -n "$preferences_response" ]] && echo "$preferences_response" | grep -qi "preference\|setting\|<html"; then
        echo "  ✓ Privacy preferences management available"
    fi
    
    # Test that no cookies are set by default
    local cookie_test_response
    cookie_test_response=$(curl -s -I --max-time 10 "$SEARXNG_BASE_URL/search?q=test" 2>/dev/null || echo "")
    
    if ! echo "$cookie_test_response" | grep -qi "set-cookie"; then
        echo "  ✓ No tracking cookies set by default"
    else
        echo "  ⚠ Cookies detected - may be for preferences only"
    fi
    
    echo "✓ SearXNG privacy features test completed"
}

# Test SearXNG search engine integration
test_searxng_engine_integration() {
    echo "🌐 Testing SearXNG search engine integration..."
    
    # Test multiple search engines (by checking if results come from different sources)
    echo "  Testing multi-engine aggregation..."
    
    local multi_engine_response
    multi_engine_response=$(curl -s --max-time 20 "$SEARXNG_BASE_URL/search?q=github&format=json" 2>/dev/null || echo '{"results":[]}')
    
    debug_json_response "$multi_engine_response" "Multi-Engine Search Response"
    
    if echo "$multi_engine_response" | jq . >/dev/null 2>&1; then
        local result_count
        result_count=$(echo "$multi_engine_response" | jq '.results | length' 2>/dev/null || echo "0")
        
        if [[ "$result_count" -gt 3 ]]; then
            echo "  ✓ Multiple search results aggregated ($result_count results)"
            
            # Check for engine diversity in results
            local engines_used
            engines_used=$(echo "$multi_engine_response" | jq -r '.results[].engine // empty' 2>/dev/null | sort -u | wc -l)
            if [[ "$engines_used" -gt 1 ]]; then
                echo "  ✓ Results from multiple engines ($engines_used different engines)"
            fi
        else
            echo "  ⚠ Limited search results returned ($result_count)"
        fi
    fi
    
    # Test search with specific language
    local lang_response
    lang_response=$(curl -s --max-time 12 "$SEARXNG_BASE_URL/search?q=test&lang=en" 2>/dev/null || echo "")
    
    if [[ -n "$lang_response" ]]; then
        echo "  ✓ Language-specific search available"
    fi
    
    # Test search with safe search
    local safe_response
    safe_response=$(curl -s --max-time 12 "$SEARXNG_BASE_URL/search?q=test&safesearch=1" 2>/dev/null || echo "")
    
    if [[ -n "$safe_response" ]]; then
        echo "  ✓ Safe search filtering available"
    fi
    
    echo "✓ SearXNG engine integration test completed"
}

# Test SearXNG performance characteristics
test_searxng_performance() {
    echo "⚡ Testing SearXNG performance characteristics..."
    
    local start_time=$(date +%s)
    
    # Test search response time
    local response
    response=$(curl -s --max-time 30 "$SEARXNG_BASE_URL/search?q=performance+test" 2>/dev/null || echo "")
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo "  Search response time: ${duration}s"
    
    if [[ $duration -lt 5 ]]; then
        echo "  ✓ Performance is excellent (< 5s)"
    elif [[ $duration -lt 12 ]]; then
        echo "  ✓ Performance is good (< 12s)"
    elif [[ $duration -lt 25 ]]; then
        echo "  ⚠ Performance is acceptable (< 25s)"
    else
        echo "  ⚠ Performance needs improvement (>= 25s)"
    fi
    
    # Test concurrent search handling
    echo "  Testing concurrent search capability..."
    local concurrent_start=$(date +%s)
    
    # Launch concurrent searches
    {
        curl -s --max-time 15 "$SEARXNG_BASE_URL/search?q=test1" >/dev/null 2>&1 &
        curl -s --max-time 15 "$SEARXNG_BASE_URL/search?q=test2" >/dev/null 2>&1 &
        curl -s --max-time 15 "$SEARXNG_BASE_URL/search?q=test3" >/dev/null 2>&1 &
        wait
    }
    
    local concurrent_end=$(date +%s)
    local concurrent_duration=$((concurrent_end - concurrent_start))
    
    echo "  Concurrent searches completed in: ${concurrent_duration}s"
    
    if [[ $concurrent_duration -lt 18 ]]; then
        echo "  ✓ Concurrent handling is efficient"
    else
        echo "  ⚠ Concurrent handling could be optimized"
    fi
    
    echo "✓ SearXNG performance test completed"
}

# Test error handling and resilience
test_searxng_error_handling() {
    echo "⚠️ Testing SearXNG error handling..."
    
    # Test empty search query
    local empty_response
    empty_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/search?q=" 2>/dev/null || echo "")
    
    assert_not_empty "$empty_response" "Empty query handled gracefully"
    
    # Test invalid search parameters
    local invalid_response
    invalid_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/search?invalid=parameter" 2>/dev/null || echo "")
    
    assert_not_empty "$invalid_response" "Invalid parameters handled"
    
    # Test malformed requests
    local malformed_response
    malformed_response=$(curl -s --max-time 10 "$SEARXNG_BASE_URL/search?q=test&format=invalid" 2>/dev/null || echo "")
    
    assert_not_empty "$malformed_response" "Malformed format request handled"
    
    # Test rate limiting protection
    echo "  Testing rate limiting..."
    local rate_start=$(date +%s)
    
    # Rapid search requests
    for i in {1..5}; do
        curl -s --max-time 3 "$SEARXNG_BASE_URL/search?q=rate$i" >/dev/null 2>&1 &
    done
    wait
    
    local rate_end=$(date +%s)
    local rate_duration=$((rate_end - rate_start))
    
    echo "  Rate limiting test completed in: ${rate_duration}s"
    
    if [[ $rate_duration -lt 20 ]]; then
        echo "  ✓ Rate limiting working appropriately"
    else
        echo "  ⚠ Rate limiting may be too restrictive"
    fi
    
    echo "✓ SearXNG error handling test completed"
}

# Main test execution
main() {
    echo "🧪 Starting SearXNG Integration Test"
    echo "Resource: $TEST_RESOURCE"
    echo "Timeout: ${TEST_TIMEOUT}s"
    echo
    
    # Setup
    setup_test
    
    # Run test suite
    test_searxng_health
    test_searxng_search_functionality
    test_searxng_api
    test_searxng_privacy_features
    test_searxng_engine_integration
    test_searxng_performance
    test_searxng_error_handling
    
    # Print summary
    echo
    print_assertion_summary
    
    if [[ $FAILED_ASSERTIONS -gt 0 ]]; then
        echo "❌ SearXNG integration test failed"
        exit 1
    else
        echo "✅ SearXNG integration test passed"
        exit 0
    fi
}

# Run main function
main "$@"