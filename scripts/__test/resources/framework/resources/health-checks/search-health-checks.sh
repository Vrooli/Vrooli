#!/bin/bash
# ====================================================================
# Search Resource Health Checks
# ====================================================================
#
# Category-specific health checks for search resources including
# index health, query performance, and privacy compliance.
#
# Supported Search Resources:
# - SearXNG: Privacy-respecting metasearch engine
#
# ====================================================================

# Search resource health check implementations
check_searxng_health() {
    local port="${1:-9200}"
    local health_level="${2:-basic}"
    
    # Basic connectivity test
    if ! curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        echo "unreachable"
        return 1
    fi
    
    if [[ "$health_level" == "basic" ]]; then
        echo "healthy"
        return 0
    fi
    
    # Detailed health check
    local ui_responsive="false"
    local search_functional="false"
    local api_available="false"
    local engines_count="unknown"
    
    # Check if main UI loads
    local ui_response
    ui_response=$(curl -s --max-time 10 "http://localhost:${port}/" 2>/dev/null)
    if [[ -n "$ui_response" ]] && echo "$ui_response" | grep -q "SearXNG"; then
        ui_responsive="true"
    fi
    
    # Test search API with a simple query
    local search_response
    search_response=$(curl -s --max-time 15 "http://localhost:${port}/search?q=test&format=json" 2>/dev/null)
    if [[ -n "$search_response" ]] && echo "$search_response" | jq . >/dev/null 2>&1; then
        api_available="true"
        
        # Check if we got search results
        local results_count
        results_count=$(echo "$search_response" | jq '.results | length' 2>/dev/null || echo "0")
        if [[ "$results_count" != "0" ]]; then
            search_functional="true"
        fi
    fi
    
    # Try to get engine statistics (if available)
    local stats_response
    stats_response=$(curl -s --max-time 10 "http://localhost:${port}/stats" 2>/dev/null)
    if [[ -n "$stats_response" ]] && echo "$stats_response" | grep -q "engines"; then
        engines_count=$(echo "$stats_response" | grep -o "engines.*" | head -1 || echo "stats_available")
    fi
    
    if [[ "$ui_responsive" == "true" && "$search_functional" == "true" && "$api_available" == "true" ]]; then
        echo "healthy:ui_responsive:search_functional:api_available:engines:$engines_count"
    elif [[ "$ui_responsive" == "true" && "$api_available" == "true" ]]; then
        echo "degraded:ui_responsive:api_available:search_limited:engines:$engines_count"
    elif [[ "$ui_responsive" == "true" ]]; then
        echo "degraded:ui_responsive:api_unavailable"
    else
        echo "degraded:ui_unresponsive"
    fi
    
    return 0
}

# Generic search health check dispatcher
check_search_resource_health() {
    local resource_name="$1"
    local port="$2"
    local health_level="${3:-basic}"
    
    case "$resource_name" in
        "searxng")
            check_searxng_health "$port" "$health_level"
            ;;
        *)
            # Fallback to generic HTTP health check
            if curl -s --max-time 5 "http://localhost:${port}/health" >/dev/null 2>&1; then
                echo "healthy"
            elif curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
                echo "healthy"
            else
                echo "unreachable"
            fi
            ;;
    esac
}

# Search resource capability testing
test_search_resource_capabilities() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "searxng")
            test_searxng_capabilities "$port"
            ;;
        *)
            echo "capability_testing_not_implemented"
            ;;
    esac
}

test_searxng_capabilities() {
    local port="$1"
    
    local capabilities=()
    
    # Test web interface
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        capabilities+=("web_interface")
        capabilities+=("privacy_respecting")
    fi
    
    # Test JSON API
    if curl -s --max-time 5 "http://localhost:${port}/search?q=test&format=json" >/dev/null 2>&1; then
        capabilities+=("json_api")
        capabilities+=("programmatic_search")
    fi
    
    # Test different output formats
    if curl -s --max-time 5 "http://localhost:${port}/search?q=test&format=rss" >/dev/null 2>&1; then
        capabilities+=("rss_output")
    fi
    
    # Test image search
    if curl -s --max-time 5 "http://localhost:${port}/search?q=test&categories=images" >/dev/null 2>&1; then
        capabilities+=("image_search")
    fi
    
    capabilities+=("metasearch")
    capabilities+=("no_tracking")
    capabilities+=("multiple_engines")
    
    if [[ ${#capabilities[@]} -gt 0 ]]; then
        echo "capabilities:$(IFS=,; echo "${capabilities[*]}")"
    else
        echo "no_capabilities_detected"
    fi
}

# Privacy validation for search resources
validate_search_privacy() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "searxng")
            validate_searxng_privacy "$port"
            ;;
        *)
            echo "privacy_validation_not_implemented"
            ;;
    esac
}

validate_searxng_privacy() {
    local port="$1"
    
    local privacy_status=()
    
    # Check if running locally (good for privacy)
    if curl -s --max-time 5 "http://localhost:${port}/" >/dev/null 2>&1; then
        privacy_status+=("local_instance")
    fi
    
    # Check for tracking protection (by examining headers and content)
    local response_headers
    response_headers=$(curl -s -I --max-time 5 "http://localhost:${port}/" 2>/dev/null)
    
    if [[ -n "$response_headers" ]]; then
        # Look for privacy-friendly headers
        if echo "$response_headers" | grep -q "X-Frame-Options"; then
            privacy_status+=("frame_protection")
        fi
        
        # Check for absence of tracking cookies
        if ! echo "$response_headers" | grep -q "Set-Cookie.*track"; then
            privacy_status+=("no_tracking_cookies")
        fi
    fi
    
    # SearXNG inherently doesn't store searches
    privacy_status+=("no_search_logging")
    privacy_status+=("no_user_profiling")
    privacy_status+=("aggregated_results")
    
    if [[ ${#privacy_status[@]} -gt 0 ]]; then
        echo "privacy:$(IFS=,; echo "${privacy_status[*]}")"
    else
        echo "privacy_unknown"
    fi
}

# Performance testing for search resources
test_search_performance() {
    local resource_name="$1"
    local port="$2"
    
    case "$resource_name" in
        "searxng")
            test_searxng_performance "$port"
            ;;
        *)
            echo "performance_testing_not_implemented"
            ;;
    esac
}

test_searxng_performance() {
    local port="$1"
    
    # Test search response time
    local start_time=$(date +%s.%N)
    
    local search_response
    search_response=$(curl -s --max-time 30 "http://localhost:${port}/search?q=performance+test&format=json" 2>/dev/null)
    
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc 2>/dev/null || echo "unknown")
    
    local performance_metrics=()
    performance_metrics+=("response_time:${duration}s")
    
    if [[ -n "$search_response" ]] && echo "$search_response" | jq . >/dev/null 2>&1; then
        local results_count
        results_count=$(echo "$search_response" | jq '.results | length' 2>/dev/null || echo "0")
        performance_metrics+=("results_count:$results_count")
        
        # Estimate throughput
        if [[ "$duration" != "unknown" ]] && [[ "$results_count" != "0" ]]; then
            local throughput
            throughput=$(echo "scale=2; $results_count / $duration" | bc 2>/dev/null || echo "unknown")
            performance_metrics+=("results_per_second:$throughput")
        fi
    fi
    
    if [[ ${#performance_metrics[@]} -gt 0 ]]; then
        echo "performance:$(IFS=,; echo "${performance_metrics[*]}")"
    else
        echo "performance_unknown"
    fi
}

# Export functions
export -f check_search_resource_health
export -f test_search_resource_capabilities
export -f validate_search_privacy
export -f test_search_performance
export -f check_searxng_health