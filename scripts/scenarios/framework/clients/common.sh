#!/bin/bash
# Common utilities for scenario testing framework
# Provides shared functionality across all handlers and clients

set -euo pipefail

# Colors for output (only define if not already defined)
if [[ -z "${RED:-}" ]]; then
    readonly RED='\033[0;31m'
    readonly GREEN='\033[0;32m'
    readonly YELLOW='\033[1;33m'
    readonly BLUE='\033[0;34m'
    readonly NC='\033[0m'
fi

# Common configuration
export FRAMEWORK_TIMEOUT=${FRAMEWORK_TIMEOUT:-30}
export FRAMEWORK_RETRY_COUNT=${FRAMEWORK_RETRY_COUNT:-3}
export FRAMEWORK_RETRY_DELAY=${FRAMEWORK_RETRY_DELAY:-2}

# Print functions with consistent formatting
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1" >&2
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_debug() {
    if [[ "${FRAMEWORK_DEBUG:-false}" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1" >&2
    fi
}

# Get resource URL with multiple fallback methods
get_resource_url() {
    local resource_name="$1"
    local resource_url=""
    
    print_debug "Looking up URL for resource: $resource_name"
    
    # Method 1: Check .vrooli/resources.local.json
    local resources_config=""
    
    # Try current directory first
    if [[ -f ".vrooli/resources.local.json" ]]; then
        resources_config=".vrooli/resources.local.json"
    # Try user home directory
    elif [[ -f "$HOME/.vrooli/resources.local.json" ]]; then
        resources_config="$HOME/.vrooli/resources.local.json"
    # Try git root directory
    elif [[ -n "${PWD:-}" ]]; then
        local git_root=$(git rev-parse --show-toplevel 2>/dev/null || echo "")
        if [[ -n "$git_root" && -f "$git_root/.vrooli/resources.local.json" ]]; then
            resources_config="$git_root/.vrooli/resources.local.json"
        fi
    fi
    
    if [[ -n "$resources_config" ]] && [[ -f "$resources_config" ]]; then
        print_debug "Found resources config: $resources_config"
        
        if command -v jq >/dev/null 2>&1; then
            # Try different path structures in the JSON
            resource_url=$(jq -r ".\"$resource_name\".url // .resources.\"$resource_name\".url // empty" "$resources_config" 2>/dev/null || echo "")
            
            # If empty, try with enabled resources
            if [[ -z "$resource_url" ]]; then
                resource_url=$(jq -r "to_entries[] | select(.value.enabled == true and .key == \"$resource_name\") | .value.url // empty" "$resources_config" 2>/dev/null || echo "")
            fi
        else
            print_debug "jq not available, skipping JSON parsing"
        fi
    fi
    
    # Method 2: Check environment variables
    if [[ -z "$resource_url" ]]; then
        local env_var_name="${resource_name^^}_URL"
        resource_url="${!env_var_name:-}"
        if [[ -n "$resource_url" ]]; then
            print_debug "Found URL in environment variable: $env_var_name"
        fi
    fi
    
    # Method 3: Default ports for common resources
    if [[ -z "$resource_url" ]]; then
        print_debug "Using default URL for resource: $resource_name"
        
        case "$resource_name" in
            # AI Services
            ollama)
                resource_url="http://localhost:11434"
                ;;
            whisper)
                resource_url="http://localhost:9000"
                ;;
            comfyui)
                resource_url="http://localhost:8188"
                ;;
            unstructured-io)
                resource_url="http://localhost:8001"
                ;;
            
            # Automation Services
            windmill)
                resource_url="http://localhost:8000"
                ;;
            n8n)
                resource_url="http://localhost:5678"
                ;;
            node-red)
                resource_url="http://localhost:1880"
                ;;
            huginn)
                resource_url="http://localhost:3000"
                ;;
            
            # Agent Services
            agent-s2)
                resource_url="http://localhost:8080"
                ;;
            claude-code)
                # Claude-code is a CLI tool, check if binary exists
                if command -v claude >/dev/null 2>&1; then
                    resource_url="local://claude"
                else
                    resource_url=""
                fi
                ;;
            browserless)
                resource_url="http://localhost:3001"
                ;;
            
            # Search Services
            searxng)
                resource_url="http://localhost:8888"
                ;;
            
            # Execution Services
            judge0)
                resource_url="http://localhost:2358"
                ;;
            
            # Storage Services
            postgres|postgresql)
                resource_url="postgresql://postgres@localhost:5432/postgres"
                ;;
            redis)
                resource_url="redis://localhost:6379"
                ;;
            minio)
                resource_url="http://localhost:9001"
                ;;
            qdrant)
                resource_url="http://localhost:6333"
                ;;
            questdb)
                resource_url="http://localhost:9000"
                ;;
            vault)
                resource_url="http://localhost:8200"
                ;;
            
            *)
                resource_url=""
                ;;
        esac
    fi
    
    print_debug "Resolved URL for $resource_name: $resource_url"
    echo "$resource_url"
}

# Check if a resource is available
is_resource_available() {
    local resource_name="$1"
    local resource_url
    
    resource_url=$(get_resource_url "$resource_name")
    
    if [[ -z "$resource_url" ]]; then
        return 1
    fi
    
    check_url_health "$resource_url" 5
}

# Check URL health with retries
check_url_health() {
    local url="$1"
    local timeout="${2:-$FRAMEWORK_TIMEOUT}"
    local retries="${3:-1}"
    
    print_debug "Checking health: $url (timeout: ${timeout}s, retries: $retries)"
    
    for ((i=1; i<=retries; i++)); do
        if [[ $i -gt 1 ]]; then
            print_debug "Retry $i/$retries for: $url"
            sleep "$FRAMEWORK_RETRY_DELAY"
        fi
        
        # Handle different URL types
        case "$url" in
            local://*)
                # Local command check
                local command_name="${url#local://}"
                if command -v "$command_name" >/dev/null 2>&1; then
                    return 0
                fi
                ;;
            http://*|https://*)
                # HTTP health check
                if check_http_health "$url" "$timeout"; then
                    return 0
                fi
                ;;
            postgresql://*|postgres://*)
                # PostgreSQL check
                if check_postgres_health "$url"; then
                    return 0
                fi
                ;;
            redis://*)
                # Redis check
                if check_redis_health "$url"; then
                    return 0
                fi
                ;;
            *)
                print_debug "Unknown URL scheme: $url"
                return 1
                ;;
        esac
    done
    
    return 1
}

# Check HTTP health with multiple endpoints
check_http_health() {
    local url="$1"
    local timeout="${2:-10}"
    
    # Common health endpoints to try
    local endpoints=(
        "/health"
        "/healthz"
        "/status"
        "/api/health"
        "/api/v1/health"
        "/ping"
        "/"
    )
    
    for endpoint in "${endpoints[@]}"; do
        local full_url="${url}${endpoint}"
        
        if curl -s --max-time "$timeout" --fail "$full_url" >/dev/null 2>&1; then
            print_debug "Health check passed: $full_url"
            return 0
        fi
    done
    
    return 1
}

# Check PostgreSQL health
check_postgres_health() {
    local url="$1"
    
    if command -v pg_isready >/dev/null 2>&1; then
        # Extract host and port from URL
        local host=$(echo "$url" | sed 's|postgresql://[^@]*@\([^:]*\).*|\1|')
        local port=$(echo "$url" | sed 's|.*:\([0-9]*\)/.*|\1|')
        
        pg_isready -h "$host" -p "${port:-5432}" >/dev/null 2>&1
    else
        # Fallback to nc if available
        local host=$(echo "$url" | sed 's|postgresql://[^@]*@\([^:]*\).*|\1|')
        local port=$(echo "$url" | sed 's|.*:\([0-9]*\)/.*|\1|')
        
        if command -v nc >/dev/null 2>&1; then
            nc -z "$host" "${port:-5432}" 2>/dev/null
        else
            return 1
        fi
    fi
}

# Check Redis health
check_redis_health() {
    local url="$1"
    
    if command -v redis-cli >/dev/null 2>&1; then
        local host=$(echo "$url" | sed 's|redis://\([^:]*\).*|\1|')
        local port=$(echo "$url" | sed 's|.*:\([0-9]*\).*|\1|')
        
        redis-cli -h "$host" -p "${port:-6379}" ping >/dev/null 2>&1
    else
        # Fallback to nc
        local host=$(echo "$url" | sed 's|redis://\([^:]*\).*|\1|')
        local port=$(echo "$url" | sed 's|.*:\([0-9]*\).*|\1|')
        
        if command -v nc >/dev/null 2>&1; then
            nc -z "$host" "${port:-6379}" 2>/dev/null
        else
            return 1
        fi
    fi
}

# Make HTTP request with retries and error handling
make_http_request() {
    local method="$1"
    local url="$2"
    local headers="$3"
    local body="$4"
    local timeout="${5:-$FRAMEWORK_TIMEOUT}"
    local retries="${6:-$FRAMEWORK_RETRY_COUNT}"
    
    print_debug "HTTP $method $url (timeout: ${timeout}s, retries: $retries)"
    
    for ((i=1; i<=retries; i++)); do
        if [[ $i -gt 1 ]]; then
            print_debug "HTTP retry $i/$retries"
            sleep "$FRAMEWORK_RETRY_DELAY"
        fi
        
        local curl_args=(
            -X "$method"
            -s
            --max-time "$timeout"
            --show-error
            --write-out "HTTPSTATUS:%{http_code}|SIZE:%{size_download}|TIME:%{time_total}"
        )
        
        # Add headers if provided
        if [[ -n "$headers" ]]; then
            while IFS= read -r header; do
                if [[ -n "$header" ]]; then
                    curl_args+=(-H "$header")
                fi
            done <<< "$headers"
        fi
        
        # Add body if provided
        if [[ -n "$body" ]]; then
            if [[ -f "$body" ]]; then
                # Body is a file path
                curl_args+=(-T "$body")
            else
                # Body is inline data
                curl_args+=(-d "$body")
            fi
        fi
        
        # Execute request
        local response
        if response=$(curl "${curl_args[@]}" "$url" 2>/dev/null); then
            echo "$response"
            return 0
        fi
    done
    
    print_error "HTTP request failed after $retries attempts: $method $url"
    return 1
}

# Wait for service to become available
wait_for_service() {
    local resource_name="$1"
    local timeout="${2:-60}"
    local interval="${3:-2}"
    
    print_info "Waiting for $resource_name to become available..."
    
    local elapsed=0
    while [[ $elapsed -lt $timeout ]]; do
        if is_resource_available "$resource_name"; then
            print_success "$resource_name is available"
            return 0
        fi
        
        echo -n "."
        sleep "$interval"
        elapsed=$((elapsed + interval))
    done
    
    echo
    print_error "$resource_name not available after ${timeout}s"
    return 1
}

# Create temporary file with cleanup
create_temp_file() {
    local prefix="${1:-framework}"
    local suffix="${2:-}"
    
    local temp_file
    temp_file=$(mktemp "/tmp/${prefix}_XXXXXX${suffix}")
    
    # Register for cleanup
    if [[ -z "${TEMP_FILES:-}" ]]; then
        export TEMP_FILES=()
    fi
    TEMP_FILES+=("$temp_file")
    
    echo "$temp_file"
}

# Cleanup temporary files
cleanup_temp_files() {
    if [[ -n "${TEMP_FILES:-}" ]]; then
        for file in "${TEMP_FILES[@]}"; do
            if [[ -f "$file" ]]; then
                rm -f "$file"
                print_debug "Cleaned up temp file: $file"
            fi
        done
        unset TEMP_FILES
    fi
}

# Register cleanup trap
trap cleanup_temp_files EXIT

# JSON utilities (fallback if jq not available)
json_extract() {
    local json="$1"
    local path="$2"
    
    if command -v jq >/dev/null 2>&1; then
        echo "$json" | jq -r "$path // empty"
    else
        # Simple fallback for basic extraction
        case "$path" in
            ".status")
                echo "$json" | sed -n 's/.*"status":"\([^"]*\)".*/\1/p'
                ;;
            ".message")
                echo "$json" | sed -n 's/.*"message":"\([^"]*\)".*/\1/p'
                ;;
            ".error")
                echo "$json" | sed -n 's/.*"error":"\([^"]*\)".*/\1/p'
                ;;
            *)
                echo ""
                ;;
        esac
    fi
}

# Validate JSON
is_valid_json() {
    local data="$1"
    
    if command -v jq >/dev/null 2>&1; then
        echo "$data" | jq empty 2>/dev/null
    else
        # Basic validation - starts with { or [
        [[ "$data" =~ ^[[:space:]]*[{\[] ]]
    fi
}

# URL encode
url_encode() {
    local string="$1"
    
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import urllib.parse; print(urllib.parse.quote('$string'))"
    else
        # Basic URL encoding for common characters
        echo "$string" | sed 's/ /%20/g; s/!/%21/g; s/"/%22/g; s/#/%23/g; s/&/%26/g'
    fi
}

# Export all functions
export -f get_resource_url
export -f is_resource_available
export -f check_url_health
export -f check_http_health
export -f check_postgres_health
export -f check_redis_health
export -f make_http_request
export -f wait_for_service
export -f create_temp_file
export -f cleanup_temp_files
export -f json_extract
export -f is_valid_json
export -f url_encode
export -f print_info
export -f print_success
export -f print_error
export -f print_warning
export -f print_debug