#!/bin/bash
# Resource Health Validator - Checks if required resources are available and healthy

set -euo pipefail

# Colors for output (use RES_ prefix to avoid conflicts)
if [[ -z "${RES_RED:-}" ]]; then
    readonly RES_RED='\033[0;31m'
    readonly RES_GREEN='\033[0;32m'
    readonly RES_YELLOW='\033[1;33m'
    readonly RES_BLUE='\033[0;34m'
    readonly RES_NC='\033[0m'
fi

# Health check results
RESOURCE_ERRORS=0
RESOURCE_WARNINGS=0
RESOURCE_TIMEOUT=30

# Print functions
print_check() {
    echo -n "  Checking $1... "
}

print_pass() {
    echo -e "${RES_GREEN}✓${RES_NC}"
}

print_fail() {
    echo -e "${RES_YELLOW}⚠${RES_NC} $1 (will continue with degraded functionality)"
    ((RESOURCE_ERRORS++))
}

print_warn() {
    echo -e "${RES_YELLOW}⚠${RES_NC} $1"
    ((RESOURCE_WARNINGS++))
}

# Map resource names to service.json paths (flattened structure)
get_service_path() {
    local resource_name="$1"
    # In the new flattened structure, all resources are directly under resources
    echo "resources.$resource_name"
}

# Check if resource is enabled in service.json
is_resource_enabled() {
    local resource_name="$1"
    local service_config="$2"
    
    if [[ ! -f "$service_config" ]] || ! command -v jq >/dev/null 2>&1; then
        return 0  # Assume enabled if we can't check
    fi
    
    local service_path=$(get_service_path "$resource_name")
    local enabled=$(jq -r ".$service_path.enabled // false" "$service_config" 2>/dev/null || echo "false")
    
    [[ "$enabled" == "true" ]]
}

# Get resource URL from service.json format
get_service_url() {
    local resource_name="$1"
    local service_config="$2"
    
    if [[ ! -f "$service_config" ]] || ! command -v jq >/dev/null 2>&1; then
        return 1
    fi
    
    local service_path=$(get_service_path "$resource_name")
    
    # Check if resource is enabled
    if ! is_resource_enabled "$resource_name" "$service_config"; then
        return 1  # Resource disabled
    fi
    
    # Try different URL patterns
    local resource_url=""
    
    # Pattern 1: baseUrl (most HTTP services)
    resource_url=$(jq -r ".$service_path.baseUrl // empty" "$service_config" 2>/dev/null || echo "")
    
    # Pattern 2: host + port (databases, some services)
    if [[ -z "$resource_url" ]]; then
        local host=$(jq -r ".$service_path.host // empty" "$service_config" 2>/dev/null || echo "")
        local port=$(jq -r ".$service_path.port // empty" "$service_config" 2>/dev/null || echo "")
        
        if [[ -n "$host" && -n "$port" ]]; then
            # Determine protocol based on resource type
            case "$resource_name" in
                postgres|postgresql)
                    resource_url="postgresql://$host:$port"
                    ;;
                postgis)
                    # PostGIS uses PostgreSQL protocol with spatial database
                    resource_url="postgresql://$host:$port/spatial"
                    ;;
                redis)
                    resource_url="redis://$host:$port"
                    ;;
                *)
                    resource_url="http://$host:$port"
                    ;;
            esac
        fi
    fi
    
    if [[ -n "$resource_url" ]]; then
        echo "$resource_url"
        return 0
    fi
    
    return 1
}

# Get resource URL from various sources (NEW VERSION)
get_resource_url() {
    local resource_name="$1"
    local resource_url=""
    
    # Method 1: Check .vrooli/service.json (NEW FORMAT)
    local service_config="$HOME/.vrooli/service.json"
    if [[ ! -f "$service_config" ]]; then
        service_config="$(git rev-parse --show-toplevel 2>/dev/null)/.vrooli/service.json" || true
    fi
    
    if [[ -f "$service_config" ]]; then
        resource_url=$(get_service_url "$resource_name" "$service_config" 2>/dev/null || echo "")
        if [[ -n "$resource_url" ]]; then
            echo "$resource_url"
            return 0
        fi
    fi
    
    # Method 2: Check .vrooli/service.json (OLD FORMAT - backward compatibility)
    local resources_config="$HOME/.vrooli/service.json"
    if [[ ! -f "$resources_config" ]]; then
        resources_config="$(git rev-parse --show-toplevel 2>/dev/null)/.vrooli/service.json" || true
    fi
    
    if [[ -f "$resources_config" ]] && command -v jq >/dev/null 2>&1; then
        resource_url=$(jq -r ".\"$resource_name\".url // empty" "$resources_config" 2>/dev/null || echo "")
        if [[ -n "$resource_url" ]]; then
            echo "$resource_url"
            return 0
        fi
    fi
    
    # Method 3: Check environment variables
    if [[ -z "$resource_url" ]]; then
        # Convert hyphens to underscores for valid variable names
        local env_var_name="${resource_name^^}"
        env_var_name="${env_var_name//-/_}_URL"
        resource_url="${!env_var_name:-}"
        if [[ -n "$resource_url" ]]; then
            echo "$resource_url"
            return 0
        fi
    fi
    
    # Method 4: Use port_registry.sh as source of truth (FALLBACK)
    # Source port registry if available
    local port_registry="${APP_ROOT}/scripts/resources/port_registry.sh"
    if [[ -f "$port_registry" ]]; then
        # Source in subshell to avoid polluting current environment
        local port=$(bash -c "source '$port_registry' && echo \"\${RESOURCE_PORTS[$resource_name]:-}\"")
        
        if [[ -n "$port" ]]; then
            # Determine protocol based on resource type
            case "$resource_name" in
                postgres|postgresql)
                    resource_url="postgresql://localhost:$port"
                    ;;
                postgis)
                    resource_url="postgresql://localhost:$port/spatial"
                    ;;
                redis)
                    resource_url="redis://localhost:$port"
                    ;;
                claude-code)
                    # Claude-code doesn't have a web interface, check if binary exists
                    if command -v claude >/dev/null 2>&1; then
                        resource_url="local://claude"
                    fi
                    ;;
                *)
                    resource_url="http://localhost:$port"
                    ;;
            esac
        fi
    fi
    
    echo "$resource_url"
}

# Check HTTP health endpoint
check_http_health() {
    local url="$1"
    local timeout="${2:-$RESOURCE_TIMEOUT}"
    
    # Try different common health endpoints
    local endpoints=(
        "/health"
        "/healthz"
        "/status"
        "/api/health"
        "/api/v1/health"
        "/"
    )
    
    for endpoint in "${endpoints[@]}"; do
        if curl -s --max-time "$timeout" --fail "${url}${endpoint}" >/dev/null 2>&1; then
            return 0
        fi
    done
    
    # Try just connecting to the base URL
    if curl -s --max-time "$timeout" --fail "$url" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

# Check database connectivity
check_database_health() {
    local url="$1"
    
    # Extract protocol
    local protocol="${url%%://*}"
    
    case "$protocol" in
        postgresql|postgres)
            if command -v pg_isready >/dev/null 2>&1; then
                local host=$(echo "$url" | sed 's/.*:\/\/\([^:]*\).*/\1/')
                local port=$(echo "$url" | sed 's/.*:\([0-9]*\).*/\1/')
                pg_isready -h "$host" -p "${port:-5432}" >/dev/null 2>&1
            else
                # Fallback to nc if available
                local host=$(echo "$url" | sed 's/.*:\/\/\([^:]*\).*/\1/')
                local port=$(echo "$url" | sed 's/.*:\([0-9]*\).*/\1/')
                nc -z "$host" "${port:-5432}" 2>/dev/null
            fi
            ;;
        redis)
            if command -v redis-cli >/dev/null 2>&1; then
                local host=$(echo "$url" | sed 's/.*:\/\/\([^:]*\).*/\1/')
                local port=$(echo "$url" | sed 's/.*:\([0-9]*\).*/\1/')
                redis-cli -h "$host" -p "${port:-6379}" ping >/dev/null 2>&1
            else
                # Fallback to nc
                local host=$(echo "$url" | sed 's/.*:\/\/\([^:]*\).*/\1/')
                local port=$(echo "$url" | sed 's/.*:\([0-9]*\).*/\1/')
                nc -z "$host" "${port:-6379}" 2>/dev/null
            fi
            ;;
        *)
            return 1
            ;;
    esac
}

# Check local command availability
check_local_health() {
    local resource_name="$1"
    
    case "$resource_name" in
        claude-code)
            command -v claude >/dev/null 2>&1
            ;;
        *)
            command -v "$resource_name" >/dev/null 2>&1
            ;;
    esac
}

# Check individual resource health
check_resource() {
    local resource_name="$1"
    local required="${2:-true}"
    
    print_check "$resource_name"
    
    # First check if resource is explicitly disabled in service.json
    local service_config="$HOME/.vrooli/service.json"
    if [[ ! -f "$service_config" ]]; then
        service_config="$(git rev-parse --show-toplevel 2>/dev/null)/.vrooli/service.json" || true
    fi
    
    if [[ -f "$service_config" ]] && command -v jq >/dev/null 2>&1; then
        if ! is_resource_enabled "$resource_name" "$service_config"; then
            if [[ "$required" == "true" ]]; then
                print_fail "Disabled in service configuration"
                return 1
            else
                print_warn "Disabled in service configuration (optional)"
                return 0
            fi
        fi
    fi
    
    local resource_url=$(get_resource_url "$resource_name")
    
    if [[ -z "$resource_url" ]]; then
        if [[ "$required" == "true" ]]; then
            print_fail "URL not configured"
            return 1
        else
            print_warn "URL not configured (optional)"
            return 0
        fi
    fi
    
    # Check health based on URL type
    local health_ok=false
    
    if [[ "$resource_url" =~ ^local:// ]]; then
        if check_local_health "$resource_name"; then
            health_ok=true
        fi
    elif [[ "$resource_url" =~ ^(postgresql|postgres|redis):// ]]; then
        if check_database_health "$resource_url"; then
            health_ok=true
        fi
    elif [[ "$resource_url" =~ ^https?:// ]]; then
        if check_http_health "$resource_url" 10; then
            health_ok=true
        fi
    fi
    
    if [[ "$health_ok" == "true" ]]; then
        print_pass
        return 0
    else
        if [[ "$required" == "true" ]]; then
            print_fail "Not available at $resource_url"
            return 1
        else
            print_warn "Not available at $resource_url (optional)"
            return 0
        fi
    fi
}

# Parse required resources from config
get_required_resources() {
    local config_file="$1"
    
    if [[ -f "$config_file" ]]; then
        # Extract required resources from YAML using precise parsing (supports both inline arrays and line-by-line format)
        awk '
            /^resources:/ { in_resources=1; next }
            in_resources && /^[[:space:]]*required:[[:space:]]*\[.*\]/ { 
                # Handle inline array: required: [item1, item2, item3]
                if (match($0, /\[([^\]]+)\]/)) {
                    bracket_content = substr($0, RSTART+1, RLENGTH-2)
                    gsub(/[[:space:]]/, "", bracket_content)  # Remove spaces
                    n = split(bracket_content, items, ",")
                    for (i = 1; i <= n; i++) {
                        print items[i]
                    }
                }
                next
            }
            in_resources && /^[[:space:]]*required:/ { in_required=1; next }
            in_resources && /^[[:space:]]*optional:/ { in_required=0; next }
            in_resources && /^[a-z_]+:/ && !/^[[:space:]]/ { in_resources=0; in_required=0; next }
            in_resources && in_required && /^[[:space:]]*-/ { 
                gsub(/^[[:space:]]*-[[:space:]]*/, ""); 
                gsub(/[\[\]",]/, "");
                print 
            }
        ' "$config_file"
    fi
}

# Parse optional resources from config
get_optional_resources() {
    local config_file="$1"
    
    if [[ -f "$config_file" ]]; then
        # Extract optional resources from YAML using precise parsing (supports both inline arrays and line-by-line format)
        awk '
            /^resources:/ { in_resources=1; next }
            in_resources && /^[[:space:]]*optional:[[:space:]]*\[.*\]/ { 
                # Handle inline array: optional: [item1, item2, item3]
                if (match($0, /\[([^\]]+)\]/)) {
                    bracket_content = substr($0, RSTART+1, RLENGTH-2)
                    gsub(/[[:space:]]/, "", bracket_content)  # Remove spaces
                    n = split(bracket_content, items, ",")
                    for (i = 1; i <= n; i++) {
                        print items[i]
                    }
                }
                next
            }
            in_resources && /^[[:space:]]*optional:/ { in_optional=1; next }
            in_resources && /^[[:space:]]*required:/ { in_optional=0; next }
            in_resources && /^[a-z_]+:/ && !/^[[:space:]]/ { in_resources=0; in_optional=0; next }
            in_resources && in_optional && /^[[:space:]]*-/ { 
                gsub(/^[[:space:]]*-[[:space:]]*/, ""); 
                gsub(/[\[\]",]/, "");
                print 
            }
        ' "$config_file"
    fi
}

# Check resource compatibility
check_resource_compatibility() {
    local resources=("$@")
    
    # Check for known incompatibilities
    local has_ollama=false
    local has_comfyui=false
    
    for resource in "${resources[@]}"; do
        case "$resource" in
            ollama) has_ollama=true ;;
            comfyui) has_comfyui=true ;;
        esac
    done
    
    # Add compatibility checks here
    # For now, all resources are considered compatible
    return 0
}

# Main resource health check function
check_resource_health() {
    local scenario_dir="$1"
    local config_file="${2:-$scenario_dir/scenario-test.yaml}"
    
    echo "  ┌─ Resource Health Check ────────────────────────────┐"
    
    local all_resources=()
    
    # Check required resources
    local required_resources
    readarray -t required_resources < <(get_required_resources "$config_file")
    
    if [[ ${#required_resources[@]} -gt 0 ]]; then
        echo "    Required Resources:"
        for resource in "${required_resources[@]}"; do
            if [[ -n "$resource" ]]; then
                # Capture return code to prevent set -e from exiting
                check_resource "$resource" true || true
                all_resources+=("$resource")
            fi
        done
    fi
    
    # Check optional resources
    local optional_resources
    readarray -t optional_resources < <(get_optional_resources "$config_file")
    
    if [[ ${#optional_resources[@]} -gt 0 ]]; then
        echo "    Optional Resources:"
        for resource in "${optional_resources[@]}"; do
            if [[ -n "$resource" ]]; then
                # Capture return code to prevent set -e from exiting
                check_resource "$resource" false || true
                all_resources+=("$resource")
            fi
        done
    fi
    
    # Check resource compatibility
    if [[ ${#all_resources[@]} -gt 0 ]]; then
        check_resource_compatibility "${all_resources[@]}"
    fi
    
    echo "  └────────────────────────────────────────────────────┘"
    
    # Report results with graceful degradation
    if [[ $RESOURCE_ERRORS -gt 0 ]]; then
        echo -e "  ${RES_YELLOW}Resource health check degraded with $RESOURCE_ERRORS unavailable resource(s)${RES_NC}"
        echo -e "  ${RES_BLUE}Note: Tests will run with available resources only${RES_NC}"
        return 0  # Allow tests to continue with warnings
    elif [[ $RESOURCE_WARNINGS -gt 0 ]]; then
        echo -e "  ${RES_YELLOW}Resource health check passed with $RESOURCE_WARNINGS warning(s)${RES_NC}"
        return 0
    else
        echo -e "  ${RES_GREEN}All resources are healthy${RES_NC}"
        return 0
    fi
}

# Export functions for use by test runner
export -f check_resource_health
export -f get_resource_url
export -f check_resource