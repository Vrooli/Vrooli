#!/bin/bash
# HTTP Mock Helpers
# Provides helpers for mocking HTTP endpoints in tests

# Mock response storage
declare -A MOCK_HTTP_RESPONSES
declare -A MOCK_HTTP_SEQUENCES
declare -A MOCK_HTTP_SEQUENCE_INDEX

#######################################
# Set a mock HTTP endpoint response
# Globals: MOCK_HTTP_RESPONSES
# Arguments:
#   $1 - endpoint URL
#   $2 - status code
#   $3 - response body
# Returns: 0
#######################################
mock::http::set_endpoint_response() {
    local endpoint="$1"
    local status="$2"
    local body="$3"
    
    MOCK_HTTP_RESPONSES["$endpoint"]="$status|$body"
    return 0
}

#######################################
# Set a sequence of mock HTTP responses
# Globals: MOCK_HTTP_SEQUENCES, MOCK_HTTP_SEQUENCE_INDEX
# Arguments:
#   $1 - endpoint URL
#   $2 - comma-separated status codes
#   $3 - comma-separated response bodies
# Returns: 0
#######################################
mock::http::set_endpoint_sequence() {
    local endpoint="$1"
    local statuses="$2"
    local bodies="$3"
    
    MOCK_HTTP_SEQUENCES["$endpoint"]="$statuses|$bodies"
    MOCK_HTTP_SEQUENCE_INDEX["$endpoint"]=0
    return 0
}

#######################################
# Get mock response for endpoint
# Globals: MOCK_HTTP_RESPONSES, MOCK_HTTP_SEQUENCES
# Arguments:
#   $1 - endpoint URL
# Returns: status code and prints body
#######################################
mock::http::get_response() {
    local endpoint="$1"
    
    # Check if we have a sequence
    if [[ -n "${MOCK_HTTP_SEQUENCES[$endpoint]}" ]]; then
        local sequence="${MOCK_HTTP_SEQUENCES[$endpoint]}"
        local index="${MOCK_HTTP_SEQUENCE_INDEX[$endpoint]:-0}"
        
        # Parse sequence
        local statuses="${sequence%%|*}"
        local bodies="${sequence##*|}"
        
        # Convert to arrays
        IFS=',' read -ra status_array <<< "$statuses"
        IFS=',' read -ra body_array <<< "$bodies"
        
        # Get current response
        local status="${status_array[$index]:-200}"
        local body="${body_array[$index]:-ok}"
        
        # Increment index for next call
        ((MOCK_HTTP_SEQUENCE_INDEX[$endpoint]++))
        
        echo "$body"
        [[ "$status" == "200" ]] && return 0 || return 1
    fi
    
    # Check static response
    if [[ -n "${MOCK_HTTP_RESPONSES[$endpoint]}" ]]; then
        local response="${MOCK_HTTP_RESPONSES[$endpoint]}"
        local status="${response%%|*}"
        local body="${response##*|}"
        
        echo "$body"
        [[ "$status" == "200" ]] && return 0 || return 1
    fi
    
    # Default response
    echo '{"status": "ok"}'
    return 0
}

# Override curl to use mock responses
curl() {
    # Extract URL from arguments
    local url=""
    for arg in "$@"; do
        if [[ "$arg" =~ ^https?:// ]]; then
            url="$arg"
            break
        fi
    done
    
    # Get mock response
    if [[ -n "$url" ]]; then
        mock::http::get_response "$url"
        return $?
    fi
    
    # Default curl behavior from standard mock
    builtin command curl "$@" 2>/dev/null || echo '{"status": "ok"}'
    return 0
}

# Export functions
export -f mock::http::set_endpoint_response
export -f mock::http::set_endpoint_sequence
export -f mock::http::get_response
export -f curl