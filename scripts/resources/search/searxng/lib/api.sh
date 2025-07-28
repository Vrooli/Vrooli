#!/usr/bin/env bash
# SearXNG API Operations
# Direct interaction with SearXNG search and management APIs

#######################################
# Perform a search via SearXNG API
# Arguments:
#   $1 - search query
#   $2 - format (optional: json, xml, csv, rss - default: json)
#   $3 - category (optional: general, images, videos, news, music, files, science)
#   $4 - language (optional: en, de, fr, etc.)
#######################################
searxng::search() {
    local query="$1"
    local format="${2:-json}"
    local category="${3:-general}"
    local language="${4:-$SEARXNG_DEFAULT_LANG}"
    
    if [[ -z "$query" ]]; then
        log::error "Search query is required"
        return 1
    fi
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    # URL encode the query
    local encoded_query
    encoded_query=$(printf '%s' "$query" | curl -Gso /dev/null -w %{url_effective} --data-urlencode @- "" | cut -c 3-)
    
    # Build search URL
    local search_url="${SEARXNG_BASE_URL}/search"
    search_url+="?q=${encoded_query}"
    search_url+="&format=${format}"
    search_url+="&categories=${category}"
    search_url+="&language=${language}"
    
    log::info "Searching for: $query"
    log::debug "Search URL: $search_url"
    
    # Perform search with timeout
    if curl -sf --max-time "$SEARXNG_API_TIMEOUT" "$search_url"; then
        return 0
    else
        log::error "Search request failed"
        return 1
    fi
}

#######################################
# Get SearXNG statistics
#######################################
searxng::get_stats() {
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    local stats_url="${SEARXNG_BASE_URL}/stats"
    
    if curl -sf --max-time "$SEARXNG_API_TIMEOUT" "$stats_url"; then
        return 0
    else
        log::error "Failed to retrieve SearXNG statistics"
        return 1
    fi
}

#######################################
# Get SearXNG configuration
#######################################
searxng::get_api_config() {
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    local config_url="${SEARXNG_BASE_URL}/config"
    
    if curl -sf --max-time "$SEARXNG_API_TIMEOUT" "$config_url"; then
        return 0
    else
        log::error "Failed to retrieve SearXNG configuration"
        return 1
    fi
}

#######################################
# Test API endpoints
#######################################
searxng::test_api() {
    log::header "SearXNG API Test"
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    local test_results=0
    
    # Test stats endpoint
    echo "Testing /stats endpoint..."
    if searxng::get_stats >/dev/null 2>&1; then
        echo "  ✅ /stats endpoint responding"
    else
        echo "  ❌ /stats endpoint failed"
        ((test_results++))
    fi
    
    # Test config endpoint
    echo "Testing /config endpoint..."
    if searxng::get_api_config >/dev/null 2>&1; then
        echo "  ✅ /config endpoint responding"
    else
        echo "  ❌ /config endpoint failed"
        ((test_results++))
    fi
    
    # Test search endpoint
    echo "Testing /search endpoint..."
    if searxng::search "test query" "json" >/dev/null 2>&1; then
        echo "  ✅ /search endpoint responding"
    else
        echo "  ❌ /search endpoint failed"
        ((test_results++))
    fi
    
    # Test different formats
    echo "Testing response formats..."
    local formats=("json" "xml" "csv")
    for format in "${formats[@]}"; do
        if searxng::search "test" "$format" >/dev/null 2>&1; then
            echo "  ✅ $format format working"
        else
            echo "  ❌ $format format failed"
            ((test_results++))
        fi
    done
    
    # Test different categories
    echo "Testing search categories..."
    local categories=("general" "images" "news")
    for category in "${categories[@]}"; do
        if searxng::search "test" "json" "$category" >/dev/null 2>&1; then
            echo "  ✅ $category category working"
        else
            echo "  ❌ $category category failed"
            ((test_results++))
        fi
    done
    
    echo
    if [[ $test_results -eq 0 ]]; then
        log::success "All API tests passed"
        return 0
    else
        log::error "API tests failed: $test_results errors"
        return 1
    fi
}

#######################################
# Perform interactive search
#######################################
searxng::interactive_search() {
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    echo "SearXNG Interactive Search"
    echo "Type 'quit' to exit"
    echo
    
    while true; do
        echo -n "Search query: "
        read -r query
        
        if [[ "$query" == "quit" ]] || [[ "$query" == "exit" ]]; then
            break
        fi
        
        if [[ -z "$query" ]]; then
            continue
        fi
        
        echo "Searching for: $query"
        echo "----------------------------------------"
        
        # Perform search and format output
        local search_result
        search_result=$(searxng::search "$query" "json" 2>/dev/null)
        
        if [[ $? -eq 0 ]] && [[ -n "$search_result" ]]; then
            # Try to format JSON nicely if jq is available
            if command -v jq >/dev/null 2>&1; then
                echo "$search_result" | jq '.results[] | {title: .title, url: .url, content: .content}' 2>/dev/null || echo "$search_result"
            else
                echo "$search_result"
            fi
        else
            echo "Search failed or no results"
        fi
        
        echo "----------------------------------------"
        echo
    done
    
    echo "Search session ended"
}

#######################################
# Benchmark search performance
# Arguments:
#   $1 - number of test queries (default: 10)
#######################################
searxng::benchmark() {
    local num_queries="${1:-10}"
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    log::info "Running SearXNG performance benchmark..."
    log::info "Number of test queries: $num_queries"
    
    # Test queries
    local test_queries=(
        "artificial intelligence"
        "machine learning"
        "docker containers"
        "open source software"
        "web development"
        "javascript frameworks"
        "database design"
        "network security"
        "cloud computing"
        "data science"
    )
    
    local total_time=0
    local successful_queries=0
    local failed_queries=0
    
    echo
    echo "Running benchmark..."
    
    for ((i=0; i<num_queries; i++)); do
        local query="${test_queries[$((i % ${#test_queries[@]}))]}"
        local start_time
        start_time=$(date +%s.%N)
        
        if searxng::search "$query" "json" >/dev/null 2>&1; then
            local end_time
            end_time=$(date +%s.%N)
            local duration
            duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "0")
            
            total_time=$(echo "$total_time + $duration" | bc -l 2>/dev/null || echo "$total_time")
            ((successful_queries++))
            
            printf "  Query %2d: ✅ %.3fs - %s\n" $((i+1)) "$duration" "$query"
        else
            ((failed_queries++))
            printf "  Query %2d: ❌ Failed - %s\n" $((i+1)) "$query"
        fi
    done
    
    echo
    echo "Benchmark Results:"
    echo "  Total queries: $num_queries"
    echo "  Successful: $successful_queries"
    echo "  Failed: $failed_queries"
    
    if [[ $successful_queries -gt 0 ]]; then
        local avg_time
        avg_time=$(echo "scale=3; $total_time / $successful_queries" | bc -l 2>/dev/null || echo "0")
        echo "  Average response time: ${avg_time}s"
        echo "  Total time: ${total_time}s"
        
        # Performance assessment
        if (( $(echo "$avg_time < 1.0" | bc -l 2>/dev/null || echo 0) )); then
            echo "  Performance: ✅ Excellent (< 1s average)"
        elif (( $(echo "$avg_time < 3.0" | bc -l 2>/dev/null || echo 0) )); then
            echo "  Performance: ✅ Good (< 3s average)"
        elif (( $(echo "$avg_time < 5.0" | bc -l 2>/dev/null || echo 0) )); then
            echo "  Performance: ⚠️  Acceptable (< 5s average)"
        else
            echo "  Performance: ❌ Poor (> 5s average)"
        fi
    fi
}

#######################################
# Show API usage examples
#######################################
searxng::show_api_examples() {
    log::header "SearXNG API Usage Examples"
    
    echo "Basic Search:"
    echo "  curl '${SEARXNG_BASE_URL}/search?q=artificial+intelligence&format=json'"
    echo
    
    echo "Image Search:"
    echo "  curl '${SEARXNG_BASE_URL}/search?q=nature&categories=images&format=json'"
    echo
    
    echo "News Search:"
    echo "  curl '${SEARXNG_BASE_URL}/search?q=technology&categories=news&format=json'"
    echo
    
    echo "Get Statistics:"
    echo "  curl '${SEARXNG_BASE_URL}/stats'"
    echo
    
    echo "Get Configuration:"
    echo "  curl '${SEARXNG_BASE_URL}/config'"
    echo
    
    echo "Available Formats:"
    echo "  - json (default)"
    echo "  - xml"
    echo "  - csv"
    echo "  - rss"
    echo
    
    echo "Available Categories:"
    echo "  - general (default)"
    echo "  - images"
    echo "  - videos"
    echo "  - news"
    echo "  - music"
    echo "  - files"
    echo "  - science"
    echo
    
    echo "Script Usage:"
    echo "  ./manage.sh --action search --query 'your search term'"
    echo "  ./manage.sh --action api-test"
    echo "  ./manage.sh --action benchmark"
}