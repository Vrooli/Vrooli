#!/bin/bash

# Execution proxy that intercepts Judge0 API calls and uses direct executor when isolate fails
# This allows Judge0 to function in Docker-in-Docker environments

set -euo pipefail

# Source helpers
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../config/defaults.sh"

# Simple logging functions
log() {
    local level="$1"
    shift
    echo "[$(date +'%H:%M:%S')] [$level] $*" >&2
}

# Map Judge0 language IDs to our executor languages
declare -A LANGUAGE_MAP=(
    ["71"]="python3"    # Python 3.8+
    ["92"]="python3"    # Python 3.11
    ["63"]="javascript" # Node.js
    ["62"]="java"       # Java
    ["54"]="cpp"        # C++
    ["50"]="cpp"        # C
    ["60"]="go"         # Go
    ["73"]="rust"       # Rust
    ["72"]="ruby"       # Ruby
    ["68"]="php"        # PHP
)

# Storage for submissions (simple file-based)
SUBMISSIONS_DIR="/tmp/judge0-submissions"
mkdir -p "$SUBMISSIONS_DIR"

# Generate unique token
generate_token() {
    echo "$(date +%s)-$(uuidgen | tr -d '-' | head -c 16)"
}

# Store submission
store_submission() {
    local token="$1"
    local data="$2"
    echo "$data" > "$SUBMISSIONS_DIR/$token.json"
}

# Retrieve submission
get_submission() {
    local token="$1"
    local file="$SUBMISSIONS_DIR/$token.json"
    
    if [[ -f "$file" ]]; then
        cat "$file"
    else
        echo '{"error": "Submission not found"}'
    fi
}

# Process submission using direct executor
process_submission() {
    local submission="$1"
    local token=$(generate_token)
    
    # Extract fields from submission
    local language_id=$(echo "$submission" | jq -r '.language_id // 71')
    local source_code=$(echo "$submission" | jq -r '.source_code // ""')
    local stdin=$(echo "$submission" | jq -r '.stdin // ""')
    local cpu_time_limit=$(echo "$submission" | jq -r '.cpu_time_limit // 2')
    local memory_limit=$(echo "$submission" | jq -r '.memory_limit // 128000' | awk '{print int($1/1024)}')
    
    # Map language ID to our executor language
    local language="${LANGUAGE_MAP[$language_id]:-python3}"
    
    # Try Judge0 API first
    local judge0_result=$(curl -sf -X POST \
        "http://localhost:${JUDGE0_PORT}/submissions?wait=false" \
        -H "Content-Type: application/json" \
        -H "X-Auth-Token: ${JUDGE0_API_KEY}" \
        -d "$submission" 2>/dev/null || echo '{}')
    
    # Check if Judge0 succeeded
    if [[ $(echo "$judge0_result" | jq -r '.token // ""') != "" ]]; then
        # Judge0 worked, return its token
        echo "$judge0_result"
        return 0
    fi
    
    # Judge0 failed, use direct executor
    log info "Judge0 submission failed, using direct executor for language: $language"
    
    # Execute using direct executor
    local result=$("${SCRIPT_DIR}/direct-executor.sh" execute \
        "$language" \
        "$source_code" \
        "$stdin" \
        "$cpu_time_limit" \
        "$memory_limit")
    
    # Add Judge0-compatible fields
    local enhanced_result=$(echo "$result" | jq ". + {
        \"token\": \"$token\",
        \"language_id\": $language_id,
        \"source_code\": $(echo "$source_code" | jq -Rs .),
        \"stdin\": $(echo "$stdin" | jq -Rs .),
        \"created_at\": \"$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")\",
        \"finished_at\": \"$(date -u +"%Y-%m-%dT%H:%M:%S.%3NZ")\"
    }")
    
    # Store result
    store_submission "$token" "$enhanced_result"
    
    # Return token response
    echo "{\"token\": \"$token\"}"
}

# Process batch submissions
process_batch() {
    local submissions="$1"
    local tokens=()
    
    echo "$submissions" | jq -c '.submissions[]' | while read -r submission; do
        local result=$(process_submission "$submission")
        local token=$(echo "$result" | jq -r '.token')
        tokens+=("\"$token\"")
    done
    
    # Return array of tokens
    echo "{\"tokens\": [$(IFS=,; echo "${tokens[*]}")]}"
}

# Get submission result
get_result() {
    local token="$1"
    
    # First check if it's our token (timestamp-uuid format)
    if [[ "$token" =~ ^[0-9]+-[a-f0-9]{16}$ ]]; then
        # Our token, get from storage
        get_submission "$token"
    else
        # Judge0 token, query Judge0 API
        curl -sf -X GET \
            "http://localhost:${JUDGE0_PORT}/submissions/$token" \
            -H "X-Auth-Token: ${JUDGE0_API_KEY}" || \
            echo '{"error": "Failed to retrieve submission"}'
    fi
}

# Health check that tests actual execution
health_check() {
    # Test direct executor
    local test_result=$("${SCRIPT_DIR}/direct-executor.sh" execute \
        "python3" \
        'print("health_check_ok")' \
        "" \
        2 \
        128)
    
    if echo "$test_result" | grep -q "health_check_ok"; then
        echo '{"status": "healthy", "executor": "direct", "message": "Direct executor working"}'
        return 0
    fi
    
    # Test Judge0 API
    local judge0_health=$(curl -sf "http://localhost:${JUDGE0_PORT}/system_info" \
        -H "X-Auth-Token: ${JUDGE0_API_KEY}" 2>/dev/null || echo '{}')
    
    if [[ $(echo "$judge0_health" | jq -r '.version // ""') != "" ]]; then
        echo '{"status": "healthy", "executor": "judge0", "message": "Judge0 API working"}'
        return 0
    fi
    
    echo '{"status": "unhealthy", "message": "No working executor available"}'
    return 1
}

# CLI interface
case "${1:-}" in
    submit)
        shift
        process_submission "$@"
        ;;
    batch)
        shift
        process_batch "$@"
        ;;
    get)
        shift
        get_result "$@"
        ;;
    health)
        health_check
        ;;
    test)
        # Test submission
        local test_submission='{
            "language_id": 71,
            "source_code": "print(\"Test from proxy\")",
            "stdin": ""
        }'
        local result=$(process_submission "$test_submission")
        echo "Submission result: $result"
        local token=$(echo "$result" | jq -r '.token')
        echo "Getting result for token: $token"
        get_result "$token"
        ;;
    *)
        echo "Usage: $0 {submit|batch|get|health|test} [options]"
        exit 1
        ;;
esac