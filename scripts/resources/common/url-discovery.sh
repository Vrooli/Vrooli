#!/usr/bin/env bash
set -euo pipefail

# Resource URL Discovery Infrastructure
# This module provides dynamic resource URL discovery, validation, and health checking
# Part of the Vrooli resource management system

# Prevent multiple sourcing
[[ -n "${VROOLI_URL_DISCOVERY_SOURCED:-}" ]] && return 0
readonly VROOLI_URL_DISCOVERY_SOURCED=1

# Source dependencies if not already loaded
if [[ -z "${VROOLI_COMMON_SOURCED:-}" ]]; then
    _HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    # shellcheck disable=SC1091
    source "${_HERE}/../../../lib/utils/var.sh"
    # shellcheck disable=SC1091
    source "${var_RESOURCES_COMMON_FILE}"
fi

# Source trash system for safe removal
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Cache settings
readonly URL_DISCOVERY_CACHE_DIR="/tmp/vrooli-url-discovery"
readonly URL_DISCOVERY_CACHE_TTL=30  # seconds

# Resource URL configuration registry
# Format: resource_name="type=http|tcp|cli,default_port=XXXX,health_path=/path,name=Display Name"
declare -A RESOURCE_URL_CONFIG=(
    # Storage Resources
    ["postgres"]="type=tcp,default_port=5432,health_check=tcp,name=PostgreSQL Database"
    ["redis"]="type=tcp,default_port=6379,health_check=tcp,name=Redis Cache"
    ["minio"]="type=http,default_port=9001,health_path=/minio/health/ready,name=MinIO Console,api_port=9000"
    ["qdrant"]="type=http,default_port=6333,health_path=/,name=Qdrant Vector Database"
    ["questdb"]="type=http,default_port=9009,health_path=/,name=QuestDB"
    ["vault"]="type=http,default_port=8200,health_path=/v1/sys/health,name=HashiCorp Vault"
    
    # AI Resources
    ["ollama"]="type=http,default_port=11434,health_path=/api/tags,name=Ollama AI"
    ["whisper"]="type=http,default_port=9000,health_path=/health,name=Whisper Speech-to-Text"
    ["unstructured-io"]="type=http,default_port=8000,health_path=/healthcheck,name=Unstructured.io API"
    
    # Automation Resources
    ["n8n"]="type=http,default_port=5678,health_path=/api/v1/info,name=n8n Workflow Automation"
    ["windmill"]="type=http,default_port=8000,health_path=/api/version,name=Windmill Automation"
    ["node-red"]="type=http,default_port=1880,health_path=/,name=Node-RED Flow-based Programming"
    ["comfyui"]="type=http,default_port=8188,health_path=/system_stats,name=ComfyUI"
    ["huginn"]="type=http,default_port=3000,health_path=/,name=Huginn Agent System"
    
    # Agent Resources
    ["agent-s2"]="type=http,default_port=4444,health_path=/api/health,name=Agent-S2 Browser Automation"
    ["browserless"]="type=http,default_port=3000,health_path=/,name=Browserless Chrome"
    ["claude-code"]="type=cli,name=Claude Code AI Assistant"
    
    # Execution Resources
    ["judge0"]="type=http,default_port=2358,health_path=/system_info,name=Judge0 Code Execution"
    
    # Search Resources
    ["searxng"]="type=http,default_port=8080,health_path=/config,name=SearXNG Privacy-Respecting Search"
)

################################################################################
# URL Discovery Cache Management
################################################################################

# Initialize cache directory
url_discovery::init_cache() {
    mkdir -p "$URL_DISCOVERY_CACHE_DIR"
}

# Get cached URL data if valid
url_discovery::get_cache() {
    local resource="$1"
    local cache_file="${URL_DISCOVERY_CACHE_DIR}/${resource}.json"
    
    if [[ -f "$cache_file" ]]; then
        local cache_age
        cache_age=$(($(date +%s) - $(stat -c %Y "$cache_file")))
        
        if (( cache_age < URL_DISCOVERY_CACHE_TTL )); then
            cat "$cache_file"
            return 0
        fi
    fi
    
    return 1
}

# Store URL data in cache
url_discovery::set_cache() {
    local resource="$1"
    local json_data="$2"
    local cache_file="${URL_DISCOVERY_CACHE_DIR}/${resource}.json"
    
    echo "$json_data" > "$cache_file"
}

# Clear cache for a resource or all resources
url_discovery::clear_cache() {
    local resource="${1:-}"
    
    if [[ -n "$resource" ]]; then
        rm -f "${URL_DISCOVERY_CACHE_DIR}/${resource}.json"
    else
        if command -v trash::safe_remove >/dev/null 2>&1; then
            trash::safe_remove "$URL_DISCOVERY_CACHE_DIR" --no-confirm
        else
            rm -rf "$URL_DISCOVERY_CACHE_DIR"
        fi
        url_discovery::init_cache
    fi
}

################################################################################
# URL Discovery Methods
################################################################################

# Parse resource configuration string
url_discovery::parse_config() {
    local resource="$1"
    local config="${RESOURCE_URL_CONFIG[$resource]:-}"
    
    if [[ -z "$config" ]]; then
        return 1
    fi
    
    # Parse config string into associative array
    declare -A parsed_config
    IFS=',' read -ra config_parts <<< "$config"
    
    for part in "${config_parts[@]}"; do
        if [[ "$part" == *"="* ]]; then
            local key="${part%%=*}"
            local value="${part#*=}"
            parsed_config["$key"]="$value"
        fi
    done
    
    # Output as JSON for easy consumption
    {
        echo "{"
        local first=true
        for key in "${!parsed_config[@]}"; do
            [[ "$first" == "true" ]] && first=false || echo ","
            printf '  "%s": "%s"' "$key" "${parsed_config[$key]}"
        done
        echo ""
        echo "}"
    }
}

# Discover URLs via resource manage.sh script
url_discovery::query_manage_script() {
    local resource="$1"
    local manage_script
    
    # Find the manage.sh script for this resource
    manage_script=$(find "${var_SCRIPTS_RESOURCES_DIR}" -path "*/*/manage.sh" -exec grep -l "DESCRIPTION.*$resource" {} \; 2>/dev/null | head -1)
    
    if [[ -z "$manage_script" ]] || [[ ! -x "$manage_script" ]]; then
        return 1
    fi
    
    # Try the new --action url command (will fail gracefully if not implemented)
    if "$manage_script" --action url 2>/dev/null; then
        return 0
    fi
    
    return 1
}

# Discover URLs via Docker inspection
url_discovery::query_docker() {
    local resource="$1"
    
    if ! command -v docker >/dev/null 2>&1; then
        return 1
    fi
    
    # Look for running containers with resource name
    local containers
    containers=$(docker ps --filter "name=$resource" --format "table {{.Names}}" 2>/dev/null | tail -n +2)
    
    if [[ -z "$containers" ]]; then
        return 1
    fi
    
    # Get port mappings for the first matching container
    local container_name
    container_name=$(echo "$containers" | head -1)
    
    local ports
    ports=$(docker port "$container_name" 2>/dev/null)
    
    if [[ -n "$ports" ]]; then
        # Parse ports and create URL JSON
        local primary_port
        primary_port=$(echo "$ports" | head -1 | cut -d: -f2)
        
        if [[ -n "$primary_port" ]]; then
            cat << EOF
{
  "primary": "http://localhost:${primary_port}",
  "type": "docker-discovered",
  "status": "unknown"
}
EOF
            return 0
        fi
    fi
    
    return 1
}

# Discover URLs via configuration fallback
url_discovery::query_config_fallback() {
    local resource="$1"
    local config_json
    
    config_json=$(url_discovery::parse_config "$resource")
    
    if [[ -z "$config_json" ]]; then
        return 1
    fi
    
    local type default_port health_path name api_port
    type=$(echo "$config_json" | jq -r '.type // empty')
    default_port=$(echo "$config_json" | jq -r '.default_port // empty')
    health_path=$(echo "$config_json" | jq -r '.health_path // empty')
    name=$(echo "$config_json" | jq -r '.name // empty')
    api_port=$(echo "$config_json" | jq -r '.api_port // empty')
    
    if [[ "$type" == "cli" ]]; then
        # CLI tools don't have URLs
        cat << EOF
{
  "name": "${name}",
  "type": "cli",
  "status": "available"
}
EOF
        return 0
    fi
    
    if [[ "$type" == "tcp" ]]; then
        # TCP services
        cat << EOF
{
  "primary": "localhost:${default_port}",
  "name": "${name}",
  "type": "tcp",
  "status": "unknown"
}
EOF
        return 0
    fi
    
    if [[ "$type" == "http" && -n "$default_port" ]]; then
        # HTTP services
        local urls="\"primary\": \"http://localhost:${default_port}\""
        
        if [[ -n "$health_path" ]]; then
            urls="${urls}, \"health\": \"http://localhost:${default_port}${health_path}\""
        fi
        
        if [[ -n "$api_port" ]]; then
            urls="${urls}, \"api\": \"http://localhost:${api_port}\""
        fi
        
        cat << EOF
{
  ${urls},
  "name": "${name}",
  "type": "http",
  "status": "unknown"
}
EOF
        return 0
    fi
    
    return 1
}

################################################################################
# URL Validation and Health Checking
################################################################################

# Check if a URL is reachable
url_discovery::validate_url() {
    local url="$1"
    local timeout="${2:-5}"
    
    if [[ "$url" == localhost:* ]] || [[ "$url" == *:* ]] && [[ "$url" != http* ]]; then
        # TCP connection check
        local host port
        host="${url%%:*}"
        port="${url##*:}"
        timeout "$timeout" bash -c "</dev/tcp/$host/$port" 2>/dev/null
    else
        # HTTP/HTTPS check
        curl --connect-timeout "$timeout" --max-time "$timeout" -s -o /dev/null -w "%{http_code}" "$url" >/dev/null 2>&1
    fi
}

# Get health status of a service
url_discovery::get_health_status() {
    local url="$1"
    local timeout="${2:-5}"
    
    if url_discovery::validate_url "$url" "$timeout"; then
        if [[ "$url" == http* ]]; then
            local http_code
            http_code=$(curl --connect-timeout "$timeout" --max-time "$timeout" -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
            
            case "$http_code" in
                200|201|204|301|302) echo "healthy" ;;
                000) echo "unreachable" ;;
                *) echo "degraded" ;;
            esac
        else
            echo "healthy"
        fi
    else
        echo "unhealthy"
    fi
}

################################################################################
# Main Discovery API
################################################################################

# Discover all URLs for a resource
url_discovery::discover() {
    local resource="$1"
    local force_refresh="${2:-false}"
    
    url_discovery::init_cache
    
    # Check cache first unless force refresh
    if [[ "$force_refresh" != "true" ]]; then
        if url_discovery::get_cache "$resource"; then
            return 0
        fi
    fi
    
    local discovery_result=""
    
    # Try discovery methods in order of preference
    if [[ -z "$discovery_result" ]]; then
        discovery_result=$(url_discovery::query_manage_script "$resource" 2>/dev/null || echo "")
    fi
    
    if [[ -z "$discovery_result" ]]; then
        discovery_result=$(url_discovery::query_docker "$resource" 2>/dev/null || echo "")
    fi
    
    if [[ -z "$discovery_result" ]]; then
        discovery_result=$(url_discovery::query_config_fallback "$resource" 2>/dev/null || echo "")
    fi
    
    if [[ -z "$discovery_result" ]]; then
        # No discovery method worked
        cat << EOF
{
  "error": "Unable to discover URLs for resource: $resource",
  "status": "unknown"
}
EOF
        return 1
    fi
    
    # Add health status to discovered URLs
    local enhanced_result primary_url health_status
    primary_url=$(echo "$discovery_result" | jq -r '.primary // empty')
    
    if [[ -n "$primary_url" ]]; then
        health_status=$(url_discovery::get_health_status "$primary_url")
    else
        health_status="unknown"
    fi
    
    enhanced_result=$(echo "$discovery_result" | jq \
        --arg resource "$resource" \
        --arg status "$health_status" \
        --arg discovered_at "$(date -Iseconds)" \
        '. + {"status": $status, "discovered_at": $discovered_at, "resource": $resource}')
    
    # Cache the result
    url_discovery::set_cache "$resource" "$enhanced_result"
    
    echo "$enhanced_result"
}

# Discover URLs for multiple resources
url_discovery::discover_multiple() {
    local resources=("$@")
    local results="{"
    local first=true
    
    for resource in "${resources[@]}"; do
        [[ "$first" == "true" ]] && first=false || results+=","
        results+='"'"$resource"'": '
        
        local resource_result
        resource_result=$(url_discovery::discover "$resource" 2>/dev/null || echo '{"error": "Discovery failed"}')
        results+="$resource_result"
    done
    
    results+="}"
    echo "$results"
}

# Format URLs for display
url_discovery::format_display() {
    local resource="$1"
    local json_data="$2"
    local show_status="${3:-true}"
    
    local name type status primary health api
    name=$(echo "$json_data" | jq -r '.name // .resource // "'"$resource"'"')
    type=$(echo "$json_data" | jq -r '.type // "unknown"')
    status=$(echo "$json_data" | jq -r '.status // "unknown"')
    primary=$(echo "$json_data" | jq -r '.primary // empty')
    health=$(echo "$json_data" | jq -r '.health // empty')
    api=$(echo "$json_data" | jq -r '.api // empty')
    
    # Status indicator
    local status_icon=""
    if [[ "$show_status" == "true" ]]; then
        case "$status" in
            healthy) status_icon="✅ " ;;
            degraded) status_icon="⚠️ " ;;
            unhealthy|unreachable) status_icon="❌ " ;;
            unknown) status_icon="⏳ " ;;
            *) status_icon="❓ " ;;
        esac
    fi
    
    if [[ "$type" == "cli" ]]; then
        echo "${status_icon}${name}: CLI Tool Available"
        return
    fi
    
    if [[ -n "$primary" ]]; then
        echo "${status_icon}${name}: ${primary}"
        
        if [[ -n "$health" && "$health" != "$primary" ]]; then
            echo "  └─ Health: ${health}"
        fi
        
        if [[ -n "$api" && "$api" != "$primary" ]]; then
            echo "  └─ API: ${api}"
        fi
    else
        echo "${status_icon}${name}: No URLs available"
    fi
}

################################################################################
# Service.json Integration
################################################################################

# Get URL overrides from service.json
url_discovery::get_service_overrides() {
    local service_json_path="$1"
    local resource="$2"
    
    if [[ ! -f "$service_json_path" ]]; then
        return 1
    fi
    
    jq -r --arg resource "$resource" '
        .deployment.urls.resources[$resource] // empty
    ' "$service_json_path" 2>/dev/null
}

# Apply service.json URL overrides
url_discovery::apply_overrides() {
    local resource="$1"
    local discovered_json="$2"
    local service_json_path="${3:-}"
    
    if [[ -z "$service_json_path" ]] || [[ ! -f "$service_json_path" ]]; then
        echo "$discovered_json"
        return
    fi
    
    local overrides
    overrides=$(url_discovery::get_service_overrides "$service_json_path" "$resource")
    
    if [[ -n "$overrides" && "$overrides" != "null" ]]; then
        # Merge overrides with discovered URLs
        echo "$discovered_json" | jq --argjson overrides "$overrides" '. + $overrides'
    else
        echo "$discovered_json"
    fi
}

################################################################################
# Utility Functions
################################################################################

# List all configured resources
url_discovery::list_resources() {
    printf '%s\n' "${!RESOURCE_URL_CONFIG[@]}" | sort
}

# Get resource categories
url_discovery::get_resource_categories() {
    cat << 'EOF'
{
  "storage": ["postgres", "redis", "minio", "qdrant", "questdb", "vault"],
  "ai": ["ollama", "whisper", "unstructured-io"],
  "automation": ["n8n", "windmill", "node-red", "comfyui", "huginn"],
  "agents": ["agent-s2", "browserless", "claude-code"],
  "execution": ["judge0"],
  "search": ["searxng"]
}
EOF
}

# Health check all known resources
url_discovery::health_check_all() {
    local resources
    mapfile -t resources < <(url_discovery::list_resources)
    
    echo "🔍 Checking health of all configured resources..."
    echo ""
    
    local healthy=0 unhealthy=0 unknown=0
    
    for resource in "${resources[@]}"; do
        local result
        result=$(url_discovery::discover "$resource" 2>/dev/null || echo '{"status": "error"}')
        url_discovery::format_display "$resource" "$result" true
        
        local status
        status=$(echo "$result" | jq -r '.status // "unknown"')
        
        case "$status" in
            healthy) ((healthy++)) ;;
            unhealthy|unreachable|error) ((unhealthy++)) ;;
            *) ((unknown++)) ;;
        esac
    done
    
    echo ""
    echo "📊 Summary: ${healthy} healthy, ${unhealthy} unhealthy, ${unknown} unknown"
}

# Export functions for external use
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Being sourced - export functions
    export -f url_discovery::discover
    export -f url_discovery::discover_multiple  
    export -f url_discovery::format_display
    export -f url_discovery::validate_url
    export -f url_discovery::get_health_status
    export -f url_discovery::clear_cache
    export -f url_discovery::list_resources
    export -f url_discovery::health_check_all
fi