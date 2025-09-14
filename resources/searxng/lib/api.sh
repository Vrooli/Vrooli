#!/usr/bin/env bash
# SearXNG API Operations
# Direct interaction with SearXNG search and management APIs

#######################################
# Execute search command with named argument parsing
# Supports both positional and named arguments
# Named arguments: --query, --format, --category, --language, etc.
# Positional arguments: query format category language ...
#######################################
searxng::execute_search() {
    local query=""
    local format="json"
    local category="general"
    local language="$SEARXNG_DEFAULT_LANG"
    local pageno="1"
    local safesearch="1"
    local time_range=""
    local output_format="json"
    local limit=""
    local save_file=""
    local append_file=""
    
    # Parse named arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --name)
                # Skip --name argument (used by CLI framework)
                shift 2
                ;;
            --query)
                query="$2"
                shift 2
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            --category)
                category="$2"
                shift 2
                ;;
            --language)
                language="$2"
                shift 2
                ;;
            --page)
                pageno="$2"
                shift 2
                ;;
            --safesearch)
                safesearch="$2"
                shift 2
                ;;
            --time-range)
                time_range="$2"
                shift 2
                ;;
            --output)
                output_format="$2"
                shift 2
                ;;
            --limit)
                limit="$2"
                shift 2
                ;;
            --save)
                save_file="$2"
                shift 2
                ;;
            --append)
                append_file="$2"
                shift 2
                ;;
            -*)
                log::warn "Unknown argument: $1"
                shift
                ;;
            *)
                # First positional argument is the query if not set
                if [[ -z "$query" ]]; then
                    query="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Call the actual search function with positional arguments
    searxng::search "$query" "$format" "$category" "$language" "$pageno" "$safesearch" "$time_range" "$output_format" "$limit" "$save_file" "$append_file"
}

#######################################
# Execute lucky search with named argument parsing
# Named arguments: --query
#######################################
searxng::execute_lucky() {
    local query=""
    
    # Parse named arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --name)
                # Skip --name argument (used by CLI framework)
                shift 2
                ;;
            --query)
                query="$2"
                shift 2
                ;;
            -*)
                log::warn "Unknown argument: $1"
                shift
                ;;
            *)
                # First positional argument is the query if not set
                if [[ -z "$query" ]]; then
                    query="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Call the actual lucky function
    searxng::lucky "$query"
}

#######################################
# Execute headlines search with named argument parsing
# Named arguments: --topic
#######################################
searxng::execute_headlines() {
    local topic=""
    
    # Parse named arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --name)
                # Skip --name argument (used by CLI framework)
                shift 2
                ;;
            --topic)
                topic="$2"
                shift 2
                ;;
            -*)
                log::warn "Unknown argument: $1"
                shift
                ;;
            *)
                # First positional argument is the topic if not set
                if [[ -z "$topic" ]]; then
                    topic="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Call the actual headlines function
    searxng::headlines "$topic"
}

#######################################
# Execute batch search with named argument parsing
# Named arguments: --file
#######################################
searxng::execute_batch() {
    local file=""
    
    # Parse named arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --name)
                # Skip --name argument (used by CLI framework)
                shift 2
                ;;
            --file)
                file="$2"
                shift 2
                ;;
            -*)
                log::warn "Unknown argument: $1"
                shift
                ;;
            *)
                # First positional argument is the file if not set
                if [[ -z "$file" ]]; then
                    file="$1"
                fi
                shift
                ;;
        esac
    done
    
    # Call the actual batch search function
    searxng::batch_search_file "$file"
}

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
    local pageno="${5:-1}"
    local safesearch="${6:-1}"
    local time_range="${7:-}"
    local output_format="${8:-json}"
    local limit="${9:-}"
    local save_file="${10:-}"
    local append_file="${11:-}"
    
    if [[ -z "$query" ]]; then
        log::error "Search query is required"
        return 1
    fi
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    # Validate parameters
    if ! [[ "$pageno" =~ ^[1-9][0-9]*$ ]]; then
        log::error "Page number must be a positive integer"
        return 1
    fi
    
    if [[ -n "$time_range" ]] && [[ ! "$time_range" =~ ^(hour|day|week|month|year)$ ]]; then
        log::error "Invalid time range. Use: hour, day, week, month, year"
        return 1
    fi
    
    if [[ ! "$safesearch" =~ ^[0-2]$ ]]; then
        log::error "Safe search must be 0, 1, or 2"
        return 1
    fi
    
    # URL encode the query using a simpler approach
    local encoded_query
    encoded_query=$(printf '%s' "$query" | sed 's/ /%20/g; s/&/%26/g; s/#/%23/g; s/+/%2B/g')
    
    # Build search URL
    local search_url="${SEARXNG_BASE_URL}/search"
    search_url+="?q=${encoded_query}"
    search_url+="&format=${format}"
    search_url+="&categories=${category}"
    search_url+="&language=${language}"
    search_url+="&pageno=${pageno}"
    search_url+="&safesearch=${safesearch}"
    
    # Add optional parameters
    [[ -n "$time_range" ]] && search_url+="&time_range=${time_range}"
    
    log::info "Searching for: $query"
    [[ "$pageno" != "1" ]] && log::info "Page: $pageno"
    [[ -n "$time_range" ]] && log::info "Time range: $time_range"
    log::debug "Search URL: $search_url"
    
    # Perform search with timeout
    local search_result
    if search_result=$(curl -sf --max-time "$SEARXNG_API_TIMEOUT" "$search_url"); then
        # Check for suspended engines and provide helpful messaging
        if command -v jq >/dev/null 2>&1 && echo "$search_result" | jq . >/dev/null 2>&1; then
            local suspended_engines
            suspended_engines=$(echo "$search_result" | jq -r '.unresponsive_engines[]? | select(.[1] | contains("Suspended")) | .[0]' 2>/dev/null | tr '\n' ',' | sed 's/,$//')
            
            if [[ -n "$suspended_engines" ]]; then
                log::warn "Some search engines are temporarily suspended: $suspended_engines"
                log::info "This may be due to rate limiting. Results may be limited."
            fi
            
            # Check if we have any results
            local result_count
            result_count=$(echo "$search_result" | jq '.results | length' 2>/dev/null || echo "0")
            if [[ "$result_count" == "0" ]]; then
                log::warn "No search results found. This could be due to:"
                log::info "  • Query too specific or no matching content"
                log::info "  • Search engines being rate-limited or suspended"
                log::info "  • Network connectivity issues"
                log::info "Try a simpler query or wait a moment before searching again"
            fi
        fi
        
        # Process results based on output format and options
        local processed_result
        processed_result=$(searxng::format_output "$search_result" "$output_format" "$limit")
        
        # Handle file operations
        if [[ -n "$save_file" ]]; then
            if echo "$processed_result" > "$save_file"; then
                log::success "Results saved to: $save_file"
            else
                log::error "Failed to save results to: $save_file"
                return 1
            fi
        elif [[ -n "$append_file" ]]; then
            local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
            local jsonl_entry
            if command -v jq >/dev/null 2>&1; then
                jsonl_entry=$(echo "$search_result" | jq -c --arg ts "$timestamp" --arg q "$query" '{timestamp: $ts, query: $q, results: .}')
            else
                # Fallback without jq
                jsonl_entry="{\"timestamp\":\"$timestamp\",\"query\":\"$query\",\"results\":$search_result}"
            fi
            
            if echo "$jsonl_entry" >> "$append_file"; then
                log::success "Results appended to: $append_file"
            else
                log::error "Failed to append results to: $append_file"
                return 1
            fi
        fi
        
        # Output results (unless saving to file)
        if [[ -z "$save_file" ]]; then
            echo "$processed_result"
        fi
        
        return 0
    else
        # Enhanced error handling for failed requests
        local http_code
        http_code=$(curl -sf --max-time "$SEARXNG_API_TIMEOUT" -w "%{http_code}" -o /dev/null "$search_url" 2>/dev/null || echo "000")
        
        case "$http_code" in
            "429")
                log::error "Search request failed: Rate limited (too many requests)"
                log::info "Please wait a moment before trying again"
                ;;
            "500"|"502"|"503"|"504")
                log::error "Search request failed: SearXNG server error (HTTP $http_code)"
                log::info "The search service may be temporarily unavailable"
                ;;
            "000")
                log::error "Search request failed: Network timeout or connection error"
                log::info "Check network connectivity and SearXNG service status"
                ;;
            *)
                log::error "Search request failed: HTTP $http_code"
                ;;
        esac
        return 1
    fi
}

#######################################
# Format search output based on requested format
# Arguments:
#   $1 - raw JSON search result
#   $2 - output format
#   $3 - result limit (optional)
#######################################
searxng::format_output() {
    local raw_result="$1"
    local output_format="${2:-json}"
    local limit="${3:-}"
    
    # Apply limit if specified
    local jq_filter='.results'
    if [[ -n "$limit" ]] && [[ "$limit" =~ ^[1-9][0-9]*$ ]]; then
        jq_filter=".results[:$limit]"
    fi
    
    case "$output_format" in
        "json")
            if [[ -n "$limit" ]]; then
                echo "$raw_result" | jq "{query: .query, number_of_results: .number_of_results, results: $jq_filter}"
            else
                echo "$raw_result"
            fi
            ;;
        "title-only")
            echo "$raw_result" | jq -r "$jq_filter | .[] | .title"
            ;;
        "title-url")
            echo "$raw_result" | jq -r "$jq_filter | .[] | \"\(.title)\n\(.url)\n\""
            ;;
        "csv")
            # CSV header
            echo "title,url,content,engine"
            # CSV data
            echo "$raw_result" | jq -r "$jq_filter | .[] | [.title, .url, .content, .engine] | @csv"
            ;;
        "markdown")
            echo "# Search Results"
            echo ""
            local query
            query=$(echo "$raw_result" | jq -r '.query // "Unknown"')
            echo "**Query:** $query"
            echo ""
            echo "$raw_result" | jq -r "$jq_filter | .[] | \"## \(.title)\n\n**URL:** \(.url)\n\n\(.content)\n\n**Source:** \(.engine)\n\n---\n\""
            ;;
        "compact")
            echo "$raw_result" | jq -r "$jq_filter | .[] | \"• \(.title)\n  \(.url)\n  \(.content[:100])...\n\""
            ;;
        *)
            log::error "Unknown output format: $output_format"
            echo "$raw_result"
            ;;
    esac
}

#######################################
# Advanced search with comprehensive filtering
# Arguments:
#   Named arguments for all filter options
#######################################
searxng::advanced_search() {
    local query=""
    local category="general"
    local language="$SEARXNG_DEFAULT_LANG"
    local time_range=""
    local engines=""
    local exclude_engines=""
    local pageno="1"
    local safesearch="1"
    local output_format="json"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --query|-q)
                query="$2"
                shift 2
                ;;
            --category|-c)
                category="$2"
                shift 2
                ;;
            --language|-l)
                language="$2"
                shift 2
                ;;
            --time|-t)
                time_range="$2"
                shift 2
                ;;
            --engines|-e)
                engines="$2"
                shift 2
                ;;
            --exclude|-x)
                exclude_engines="$2"
                shift 2
                ;;
            --page|-p)
                pageno="$2"
                shift 2
                ;;
            --safe|-s)
                safesearch="$2"
                shift 2
                ;;
            --format|-f)
                output_format="$2"
                shift 2
                ;;
            --help|-h)
                echo "Advanced SearXNG Search"
                echo
                echo "Usage: resource-searxng content advanced-search [options]"
                echo
                echo "Options:"
                echo "  --query, -q <text>       Search query (required)"
                echo "  --category, -c <cat>     Category: general, images, videos, news, music, files, science, social, map"
                echo "  --language, -l <lang>    Language code: en, de, fr, es, it, pt, ru, zh, ja, etc."
                echo "  --time, -t <range>       Time range: hour, day, week, month, year"
                echo "  --engines, -e <list>     Comma-separated list of engines to use"
                echo "  --exclude, -x <list>     Comma-separated list of engines to exclude"
                echo "  --page, -p <num>         Page number (default: 1)"
                echo "  --safe, -s <0|1|2>       Safe search: 0=off, 1=moderate, 2=strict"
                echo "  --format, -f <fmt>       Output format: json, csv, html, rss"
                echo
                echo "Examples:"
                echo "  # Search for recent AI news in English"
                echo "  resource-searxng content advanced-search -q 'artificial intelligence' -c news -t week -l en"
                echo
                echo "  # Search for images using specific engines"
                echo "  resource-searxng content advanced-search -q 'cats' -c images -e 'google,bing'"
                echo
                echo "  # Search excluding certain engines"
                echo "  resource-searxng content advanced-search -q 'privacy' -x 'google,bing' -s 2"
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$query" ]]; then
        log::error "Search query required. Use --help for usage information."
        return 1
    fi
    
    # Build the search URL with all filters
    local search_url="${SEARXNG_BASE_URL}/search"
    local encoded_query
    encoded_query=$(printf '%s' "$query" | sed 's/ /%20/g; s/&/%26/g; s/#/%23/g; s/+/%2B/g')
    
    search_url+="?q=${encoded_query}"
    search_url+="&format=${output_format}"
    search_url+="&categories=${category}"
    search_url+="&language=${language}"
    search_url+="&pageno=${pageno}"
    search_url+="&safesearch=${safesearch}"
    
    [[ -n "$time_range" ]] && search_url+="&time_range=${time_range}"
    [[ -n "$engines" ]] && search_url+="&engines=${engines}"
    [[ -n "$exclude_engines" ]] && search_url+="&disabled_engines=${exclude_engines}"
    
    log::info "Advanced search for: $query"
    log::info "Filters:"
    echo "  Category: $category"
    echo "  Language: $language"
    [[ -n "$time_range" ]] && echo "  Time range: $time_range"
    [[ -n "$engines" ]] && echo "  Using engines: $engines"
    [[ -n "$exclude_engines" ]] && echo "  Excluding: $exclude_engines"
    echo "  Safe search: $safesearch"
    echo "  Page: $pageno"
    
    # Execute search
    local result
    if result=$(curl -sf --max-time "$SEARXNG_API_TIMEOUT" "$search_url"); then
        if [[ "$output_format" == "json" ]] && command -v jq >/dev/null 2>&1; then
            echo "$result" | jq '.'
        else
            echo "$result"
        fi
        return 0
    else
        log::error "Search failed"
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
    
    local i=0
    while [[ $i -lt $num_queries ]]; do
        local query="${test_queries[$((i % ${#test_queries[@]}))]}"
        local start_time
        start_time=$(date +%s)
        
        # Use direct curl instead of searxng::search to avoid output processing issues
        local encoded_query
        encoded_query=$(echo "$query" | sed 's/ /%20/g')
        
        if timeout 5 curl -sf "${SEARXNG_BASE_URL}/search?q=${encoded_query}&format=json" >/dev/null 2>&1; then
            local end_time
            end_time=$(date +%s)
            local duration=$((end_time - start_time))
            
            total_time=$((total_time + duration))
            ((successful_queries++)) || true
            
            echo "  Query $((i+1)): ✅ ${duration}s - $query"
        else
            ((failed_queries++)) || true
            echo "  Query $((i+1)): ❌ Failed - $query"
        fi
        ((i++)) || true
    done
    
    echo
    echo "Benchmark Results:"
    echo "  Total queries: $num_queries"
    echo "  Successful: $successful_queries"
    echo "  Failed: $failed_queries"
    
    if [[ $successful_queries -gt 0 ]]; then
        local avg_time=$((total_time / successful_queries))
        echo "  Average response time: ${avg_time}s"
        echo "  Total time: ${total_time}s"
        
        # Performance assessment
        if [[ $avg_time -lt 1 ]]; then
            echo "  Performance: ✅ Excellent (< 1s average)"
        elif [[ $avg_time -lt 3 ]]; then
            echo "  Performance: ✅ Good (< 3s average)"
        elif [[ $avg_time -lt 5 ]]; then
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
    echo "  resource-searxng content execute --name search --query 'your search term'"
    echo "  resource-searxng test integration"
    echo "  resource-searxng content execute --name benchmark"
}

#######################################
# Quick action: Get latest headlines
# Arguments:
#   $1 - topic filter (optional)
#######################################
searxng::headlines() {
    local topic="${1:-}"
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    local query
    if [[ -n "$topic" ]]; then
        query="$topic news headlines"
        log::info "Getting headlines for topic: $topic"
    else
        query="news headlines"
        log::info "Getting latest headlines"
    fi
    
    # Search for headlines with news category and recent time range
    searxng::search "$query" "json" "news" "en" "1" "1" "day" "compact" "10"
}

#######################################
# Quick action: Lucky search (first result URL)
# Arguments:
#   $1 - search query
#######################################
searxng::lucky() {
    local query="$1"
    
    if [[ -z "$query" ]]; then
        log::error "Search query is required for lucky search"
        return 1
    fi
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    log::info "Feeling lucky with: $query"
    
    # Get first result URL by calling API directly to avoid log output
    local encoded_query
    encoded_query=$(printf '%s' "$query" | sed 's/ /%20/g; s/&/%26/g; s/#/%23/g; s/+/%2B/g')
    
    local search_url="${SEARXNG_BASE_URL}/search?q=${encoded_query}&format=json&categories=general&language=en&pageno=1&safesearch=1"
    
    local first_url
    first_url=$(curl -sf --max-time "$SEARXNG_API_TIMEOUT" "$search_url" | jq -r '.results[0].url // empty' 2>/dev/null)
    
    if [[ -n "$first_url" ]]; then
        echo "$first_url"
        return 0
    else
        log::error "No results found for query: $query"
        return 1
    fi
}

#######################################
# Batch search from file
# Arguments:
#   $1 - file path containing queries (one per line)
#######################################
searxng::batch_search_file() {
    local file_path="$1"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    if [[ ! -r "$file_path" ]]; then
        log::error "Cannot read file: $file_path"
        return 1
    fi
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    log::info "Starting batch search from file: $file_path"
    
    local line_count total_queries current_query=0
    total_queries=$(wc -l < "$file_path")
    
    log::info "Processing $total_queries queries..."
    
    while IFS= read -r query; do
        # Skip empty lines and comments
        [[ -z "$query" ]] && continue
        [[ "$query" =~ ^[[:space:]]*# ]] && continue
        
        ((current_query++))
        
        log::info "[$current_query/$total_queries] Searching: $query"
        
        # Create output filename
        local safe_query
        safe_query=$(echo "$query" | tr ' /' '_-' | tr -cd '[:alnum:]_-')
        local output_file="results_${current_query}_${safe_query}.json"
        
        # Perform search and save results
        if searxng::search "$query" "json" "general" "en" "1" "1" "" "json" "" "$output_file"; then
            log::success "Results saved to: $output_file"
        else
            log::error "Failed to search: $query"
        fi
        
        # Rate limiting - small delay between requests
        sleep 1
    done < "$file_path"
    
    log::success "Batch search completed: $current_query queries processed"
}

#######################################
# Batch search from comma-separated queries
# Arguments:
#   $1 - comma-separated list of queries
#######################################
searxng::batch_search_queries() {
    local queries_string="$1"
    
    if [[ -z "$queries_string" ]]; then
        log::error "Query list cannot be empty"
        return 1
    fi
    
    if ! searxng::is_healthy; then
        log::error "SearXNG is not running or healthy"
        return 1
    fi
    
    # Split queries by comma
    IFS=',' read -ra queries <<< "$queries_string"
    local total_queries=${#queries[@]}
    
    log::info "Starting batch search with $total_queries queries"
    
    local current_query=0
    for query in "${queries[@]}"; do
        # Trim whitespace
        query=$(echo "$query" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        # Skip empty queries
        [[ -z "$query" ]] && continue
        
        ((current_query++))
        
        log::info "[$current_query/$total_queries] Searching: $query"
        
        # Create output filename
        local safe_query
        safe_query=$(echo "$query" | tr ' /' '_-' | tr -cd '[:alnum:]_-')
        local output_file="batch_${current_query}_${safe_query}.json"
        
        # Perform search and save results
        if searxng::search "$query" "json" "general" "en" "1" "1" "" "json" "" "$output_file"; then
            log::success "Results saved to: $output_file"
        else
            log::error "Failed to search: $query"
        fi
        
        # Rate limiting - small delay between requests
        sleep 1
    done
    
    log::success "Batch search completed: $current_query queries processed"
}