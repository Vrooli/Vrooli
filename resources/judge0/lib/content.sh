#!/usr/bin/env bash
#######################################
# Content management for Judge0
# Handles code submissions as content
#######################################

# Store submissions in a local tracking file
JUDGE0_SUBMISSIONS_FILE="${JUDGE0_DATA_DIR:-/tmp/judge0}/submissions.json"

#######################################
# Initialize submissions file
#######################################
judge0::content::init() {
    local dir=$(dirname "$JUDGE0_SUBMISSIONS_FILE")
    [[ -d "$dir" ]] || mkdir -p "$dir"
    [[ -f "$JUDGE0_SUBMISSIONS_FILE" ]] || echo "[]" > "$JUDGE0_SUBMISSIONS_FILE"
}

#######################################
# Add content (submit code)
# Maps to content::add
#######################################
judge0::content::add() {
    local code="${1:-}"
    local language="${2:-javascript}"
    local stdin="${3:-}"
    
    if [[ -z "$code" ]]; then
        log::error "Code required for submission"
        echo "Usage: resource-judge0 content add <code> [language] [stdin]"
        return 1
    fi
    
    # Submit the code and get token
    local response
    response=$(judge0::api::submit "$code" "$language" "$stdin" 2>&1)
    local exit_code=$?
    
    if [[ $exit_code -ne 0 ]]; then
        echo "$response"
        return $exit_code
    fi
    
    # Extract token from response (if available)
    local token=$(echo "$response" | grep -oP 'Token: \K[a-f0-9-]+' || echo "")
    
    if [[ -n "$token" ]]; then
        # Store submission info
        judge0::content::init
        local submission=$(jq -n \
            --arg token "$token" \
            --arg code "$code" \
            --arg language "$language" \
            --arg timestamp "$(date -Iseconds)" \
            '{token: $token, code: $code, language: $language, timestamp: $timestamp}')
        
        jq ". += [$submission]" "$JUDGE0_SUBMISSIONS_FILE" > "${JUDGE0_SUBMISSIONS_FILE}.tmp" && \
            mv "${JUDGE0_SUBMISSIONS_FILE}.tmp" "$JUDGE0_SUBMISSIONS_FILE"
    fi
    
    echo "$response"
}

#######################################
# List content (show submissions)
# Maps to content::list
#######################################
judge0::content::list() {
    judge0::content::init
    
    if ! judge0::is_running; then
        log::warning "Judge0 is not running. Showing cached submissions only."
    fi
    
    local count=$(jq 'length' "$JUDGE0_SUBMISSIONS_FILE")
    
    if [[ "$count" -eq 0 ]]; then
        log::info "No submissions found"
        return 0
    fi
    
    log::info "Recent submissions ($count total):"
    echo ""
    
    # Display submissions in table format
    printf "%-40s %-15s %-25s\n" "Token" "Language" "Timestamp"
    printf "%-40s %-15s %-25s\n" "----------------------------------------" "---------------" "-------------------------"
    
    jq -r '.[] | "\(.token) \(.language) \(.timestamp)"' "$JUDGE0_SUBMISSIONS_FILE" | \
    while IFS=' ' read -r token lang timestamp; do
        printf "%-40s %-15s %-25s\n" "$token" "$lang" "$timestamp"
    done
}

#######################################
# Get content (retrieve submission result)
# Maps to content::get
#######################################
judge0::content::get() {
    local token="${1:-}"
    
    if [[ -z "$token" ]]; then
        log::error "Token required"
        echo "Usage: resource-judge0 content get <token>"
        echo "Get token from 'resource-judge0 content list'"
        return 1
    fi
    
    judge0::api::get_submission "$token"
}

#######################################
# Remove content (delete submission)
# Maps to content::remove
#######################################
judge0::content::remove() {
    local token="${1:-}"
    
    if [[ -z "$token" ]]; then
        log::error "Token required"
        echo "Usage: resource-judge0 content remove <token>"
        return 1
    fi
    
    # Remove from API if running
    if judge0::is_running; then
        judge0::api::delete_submission "$token"
    fi
    
    # Remove from local tracking
    judge0::content::init
    jq "map(select(.token != \"$token\"))" "$JUDGE0_SUBMISSIONS_FILE" > "${JUDGE0_SUBMISSIONS_FILE}.tmp" && \
        mv "${JUDGE0_SUBMISSIONS_FILE}.tmp" "$JUDGE0_SUBMISSIONS_FILE"
    
    log::success "Removed submission: $token"
}

#######################################
# Execute content (submit and wait for result)
# Maps to content::execute
#######################################
judge0::content::execute() {
    # This is the same as add, but emphasizes immediate execution
    judge0::content::add "$@"
}