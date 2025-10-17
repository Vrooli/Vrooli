#!/bin/bash

# External Judge0 API fallback for when local execution fails
# Uses the free Judge0 CE API at judge0-ce.p.rapidapi.com

set -euo pipefail

# Source helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Simple logging functions
log() {
    local level="$1"
    shift
    echo "[$(date +'%H:%M:%S')] [$level] $*" >&2
}

# External API configuration
EXTERNAL_API_URL="https://judge0-ce.p.rapidapi.com"
RAPIDAPI_KEY="${JUDGE0_RAPIDAPI_KEY:-}"  # Set this in environment for production
EXTERNAL_ENABLED="${JUDGE0_EXTERNAL_ENABLED:-false}"

# Check if external API is available
is_external_available() {
    [[ "$EXTERNAL_ENABLED" == "true" ]] && [[ -n "$RAPIDAPI_KEY" ]]
}

# Submit to external API
submit_external() {
    local submission="$1"
    
    if ! is_external_available; then
        echo '{"error": "External API not configured"}'
        return 1
    fi
    
    local response=$(curl -sf -X POST \
        "${EXTERNAL_API_URL}/submissions?base64_encoded=false&wait=false" \
        -H "Content-Type: application/json" \
        -H "X-RapidAPI-Key: ${RAPIDAPI_KEY}" \
        -H "X-RapidAPI-Host: judge0-ce.p.rapidapi.com" \
        -d "$submission" 2>/dev/null || echo '{"error": "External API request failed"}')
    
    echo "$response"
}

# Get submission result from external API
get_external() {
    local token="$1"
    
    if ! is_external_available; then
        echo '{"error": "External API not configured"}'
        return 1
    fi
    
    local response=$(curl -sf -X GET \
        "${EXTERNAL_API_URL}/submissions/${token}?base64_encoded=false" \
        -H "X-RapidAPI-Key: ${RAPIDAPI_KEY}" \
        -H "X-RapidAPI-Host: judge0-ce.p.rapidapi.com" 2>/dev/null || echo '{"error": "Failed to get result"}')
    
    echo "$response"
}

# Test external API connectivity
test_external() {
    if ! is_external_available; then
        echo "External API is not configured. Set JUDGE0_EXTERNAL_ENABLED=true and JUDGE0_RAPIDAPI_KEY=<your-key>"
        return 1
    fi
    
    log info "Testing external Judge0 API..."
    
    # Submit test code
    local test_submission='{
        "source_code": "print(\"External API Test\")",
        "language_id": 71
    }'
    
    local submit_response=$(submit_external "$test_submission")
    local token=$(echo "$submit_response" | jq -r '.token // ""')
    
    if [[ -z "$token" ]]; then
        log error "Failed to submit to external API: $submit_response"
        return 1
    fi
    
    log info "Submitted with token: $token"
    
    # Wait for completion
    sleep 2
    
    # Get result
    local result=$(get_external "$token")
    local status=$(echo "$result" | jq -r '.status.description // "unknown"')
    local stdout=$(echo "$result" | jq -r '.stdout // ""')
    
    log info "Status: $status"
    log info "Output: $stdout"
    
    if [[ "$stdout" == "External API Test" ]]; then
        log success "External API is working!"
        return 0
    else
        log error "External API test failed"
        return 1
    fi
}

# Hybrid submission (try local first, fallback to external)
hybrid_submit() {
    local submission="$1"
    
    # Try local Judge0 first
    local local_response=$(curl -sf -X POST \
        "http://localhost:${JUDGE0_PORT:-2358}/submissions?wait=false" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Token: ${JUDGE0_API_KEY:-}" \
        -d "$submission" 2>/dev/null || echo '{}')
    
    local local_token=$(echo "$local_response" | jq -r '.token // ""')
    
    if [[ -n "$local_token" ]]; then
        # Local submission succeeded
        echo "$local_response" | jq '. + {"executor": "local"}'
        return 0
    fi
    
    # Try direct executor
    local language_id=$(echo "$submission" | jq -r '.language_id // 71')
    local source_code=$(echo "$submission" | jq -r '.source_code // ""')
    local stdin=$(echo "$submission" | jq -r '.stdin // ""')
    
    # Map Judge0 language IDs to direct executor languages
    local language="python3"
    case $language_id in
        71|92) language="python3" ;;
        63) language="javascript" ;;
        62) language="java" ;;
        54|50) language="cpp" ;;
        60) language="go" ;;
        73) language="rust" ;;
        72) language="ruby" ;;
        68) language="php" ;;
    esac
    
    local direct_result=$("${SCRIPT_DIR}/direct-executor.sh" execute \
        "$language" \
        "$source_code" \
        "$stdin" \
        5 \
        128 2>/dev/null || echo "FAILED")
    
    if [[ "$direct_result" != "FAILED" ]]; then
        # Direct executor succeeded
        echo "$direct_result" | jq '. + {"executor": "direct", "token": "'$(uuidgen)'"}'
        return 0
    fi
    
    # Try external API as last resort
    if is_external_available; then
        local external_response=$(submit_external "$submission")
        local external_token=$(echo "$external_response" | jq -r '.token // ""')
        
        if [[ -n "$external_token" ]]; then
            echo "$external_response" | jq '. + {"executor": "external"}'
            return 0
        fi
    fi
    
    # All methods failed
    echo '{"error": "All execution methods failed", "executors_tried": ["local", "direct", "external"]}'
    return 1
}

# CLI interface
# Only run CLI if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        submit)
            shift
            submit_external "$@"
            ;;
        get)
            shift
            get_external "$@"
            ;;
        test)
            test_external
            ;;
        hybrid)
            shift
            hybrid_submit "$@"
            ;;
        status)
            if is_external_available; then
                echo '{"available": true, "api_url": "'$EXTERNAL_API_URL'"}'
            else
                echo '{"available": false, "reason": "Not configured"}'
            fi
            ;;
        *)
            echo "Usage: $0 {submit|get|test|hybrid|status} [options]"
            echo ""
            echo "Configure with:"
            echo "  export JUDGE0_EXTERNAL_ENABLED=true"
            echo "  export JUDGE0_RAPIDAPI_KEY=<your-rapidapi-key>"
            exit 1
            ;;
    esac
fi