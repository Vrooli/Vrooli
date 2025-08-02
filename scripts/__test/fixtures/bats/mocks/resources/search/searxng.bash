#!/usr/bin/env bash
# SearXNG Resource Mock Implementation
# Provides realistic mock responses for SearXNG search engine service

# Prevent duplicate loading
if [[ "${SEARXNG_MOCK_LOADED:-}" == "true" ]]; then
    return 0
fi
export SEARXNG_MOCK_LOADED="true"

#######################################
# Setup SearXNG mock environment
# Arguments: $1 - state (healthy, unhealthy, installing, stopped)
#######################################
mock::searxng::setup() {
    local state="${1:-healthy}"
    
    # Configure SearXNG-specific environment
    export SEARXNG_PORT="${SEARXNG_PORT:-8080}"
    export SEARXNG_BASE_URL="http://localhost:${SEARXNG_PORT}"
    export SEARXNG_CONTAINER_NAME="${TEST_NAMESPACE}_searxng"
    
    # Set up Docker mock state
    mock::docker::set_container_state "$SEARXNG_CONTAINER_NAME" "$state"
    
    # Configure HTTP endpoints based on state
    case "$state" in
        "healthy")
            mock::searxng::setup_healthy_endpoints
            ;;
        "unhealthy")
            mock::searxng::setup_unhealthy_endpoints
            ;;
        "installing")
            mock::searxng::setup_installing_endpoints
            ;;
        "stopped")
            mock::searxng::setup_stopped_endpoints
            ;;
        *)
            echo "[SEARXNG_MOCK] Unknown state: $state" >&2
            return 1
            ;;
    esac
    
    echo "[SEARXNG_MOCK] SearXNG mock configured with state: $state"
}

#######################################
# Setup healthy SearXNG endpoints
#######################################
mock::searxng::setup_healthy_endpoints() {
    # Health endpoint
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/healthz" \
        '{"status":"ok","version":"2023.3.6"}'
    
    # Search API endpoint
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/search" \
        '{
            "query": "test query",
            "number_of_results": 3,
            "results": [
                {
                    "url": "https://example.com/result1",
                    "title": "Test Result 1",
                    "content": "This is the first test result content",
                    "engine": "google",
                    "parsed_url": ["https", "example.com", "/result1", "", "", ""],
                    "template": "default.html",
                    "engines": ["google"],
                    "positions": [1],
                    "score": 1.0,
                    "category": "general"
                },
                {
                    "url": "https://example.org/result2",
                    "title": "Test Result 2",
                    "content": "This is the second test result content",
                    "engine": "bing",
                    "parsed_url": ["https", "example.org", "/result2", "", "", ""],
                    "template": "default.html",
                    "engines": ["bing"],
                    "positions": [2],
                    "score": 0.9,
                    "category": "general"
                }
            ],
            "answers": [],
            "corrections": [],
            "infoboxes": [],
            "suggestions": ["test queries", "test search"],
            "unresponsive_engines": []
        }'
    
    # Config endpoint
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/config" \
        '{
            "categories": ["general", "images", "videos", "news", "map", "music", "it", "science"],
            "engines": [
                {"name": "google", "categories": ["general"], "shortcut": "go"},
                {"name": "bing", "categories": ["general"], "shortcut": "bi"},
                {"name": "duckduckgo", "categories": ["general"], "shortcut": "ddg"}
            ],
            "plugins": ["Hash plugin", "Self Informations", "Tracker URL remover"],
            "instance_name": "SearXNG Test Instance",
            "locales": ["en", "en-US", "de", "fr"],
            "safesearch": 0,
            "autocomplete": ["google", "duckduckgo"]
        }'
    
    # Stats endpoint
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/stats" \
        '{
            "engines": {
                "google": {"total": 1000, "successful": 950, "errors": 50},
                "bing": {"total": 800, "successful": 760, "errors": 40},
                "duckduckgo": {"total": 1200, "successful": 1150, "errors": 50}
            }
        }'
}

#######################################
# Setup unhealthy SearXNG endpoints
#######################################
mock::searxng::setup_unhealthy_endpoints() {
    # Health endpoint returns error
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/healthz" \
        '{"status":"error","error":"Search engines unavailable"}' \
        "GET" \
        "503"
    
    # Search endpoint returns error
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/search" \
        '{"error":"Service temporarily unavailable"}' \
        "GET" \
        "503"
}

#######################################
# Setup installing SearXNG endpoints
#######################################
mock::searxng::setup_installing_endpoints() {
    # Health endpoint returns installing status
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/healthz" \
        '{"status":"installing","progress":85,"current_step":"Configuring search engines"}'
    
    # Other endpoints return not ready
    mock::http::set_endpoint_response "$SEARXNG_BASE_URL/search" \
        '{"error":"SearXNG is still initializing"}' \
        "GET" \
        "503"
}

#######################################
# Setup stopped SearXNG endpoints
#######################################
mock::searxng::setup_stopped_endpoints() {
    # All endpoints fail to connect
    mock::http::set_endpoint_unreachable "$SEARXNG_BASE_URL"
}

#######################################
# Mock SearXNG-specific operations
#######################################

# Mock search with different categories
mock::searxng::search_category() {
    local query="$1"
    local category="${2:-general}"
    
    echo '{
        "query": "'$query'",
        "category": "'$category'",
        "number_of_results": 5,
        "results": [
            {
                "url": "https://example.com/'$category'/1",
                "title": "'$category' result for '$query'",
                "content": "Content related to '$query' in '$category' category"
            }
        ]
    }'
}

#######################################
# Export mock functions
#######################################
export -f mock::searxng::setup
export -f mock::searxng::setup_healthy_endpoints
export -f mock::searxng::setup_unhealthy_endpoints
export -f mock::searxng::setup_installing_endpoints
export -f mock::searxng::setup_stopped_endpoints
export -f mock::searxng::search_category

echo "[SEARXNG_MOCK] SearXNG mock implementation loaded"