#!/usr/bin/env bash
set -euo pipefail

# SearXNG Data Injection Adapter
# This script handles injection of engines, settings, and configurations into SearXNG
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject search engines, settings, and configurations into SearXNG meta search engine"

SEARXNG_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SEARXNG_SCRIPT_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Source SearXNG configuration if available
if [[ -f "${SEARXNG_SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${SEARXNG_SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default SearXNG settings
readonly DEFAULT_SEARXNG_HOST="http://localhost:8080"
readonly DEFAULT_SEARXNG_DATA_DIR="${HOME}/.searxng"
readonly DEFAULT_SEARXNG_SETTINGS_DIR="${DEFAULT_SEARXNG_DATA_DIR}/settings"
readonly DEFAULT_SEARXNG_ENGINES_DIR="${DEFAULT_SEARXNG_DATA_DIR}/engines"

# SearXNG settings (can be overridden by environment)
SEARXNG_HOST="${SEARXNG_HOST:-$DEFAULT_SEARXNG_HOST}"
SEARXNG_DATA_DIR="${SEARXNG_DATA_DIR:-$DEFAULT_SEARXNG_DATA_DIR}"
SEARXNG_SETTINGS_DIR="${SEARXNG_SETTINGS_DIR:-$DEFAULT_SEARXNG_SETTINGS_DIR}"
SEARXNG_ENGINES_DIR="${SEARXNG_ENGINES_DIR:-$DEFAULT_SEARXNG_ENGINES_DIR}"

# Operation tracking
declare -a SEARXNG_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
searxng_inject::usage() {
    cat << EOF
SearXNG Data Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects search engines, preferences, and configurations into SearXNG based on 
    scenario configuration. Supports validation, injection, status checks, 
    and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the data injection
    --status      Check status of injected data
    --rollback    Rollback injected data
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "engines": [
        {
          "name": "google",
          "enabled": true,
          "categories": ["general"],
          "weight": 1.0,
          "timeout": 5.0
        },
        {
          "name": "duckduckgo",
          "enabled": true,
          "categories": ["general"],
          "safe_search": 1
        }
      ],
      "preferences": {
        "autocomplete": "google",
        "language": "en",
        "locale": "en",
        "safe_search": 1,
        "results_per_page": 10,
        "theme": "simple"
      },
      "outgoing": {
        "request_timeout": 5.0,
        "max_request_timeout": 15.0,
        "pool_connections": 100,
        "pool_maxsize": 20,
        "enable_http2": true
      },
      "server": {
        "secret_key": "change-this-secret",
        "limiter": true,
        "public_instance": false,
        "image_proxy": true
      },
      "configurations": [
        {
          "key": "search.formats",
          "value": ["html", "json", "rss"]
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"engines": [{"name": "google", "enabled": true}]}'
    
    # Enable specific search engines
    $0 --inject '{"engines": [{"name": "duckduckgo", "enabled": true, "weight": 1.5}]}'
    
    # Configure preferences and server settings
    $0 --inject '{"preferences": {"language": "en", "theme": "simple"}}'

EOF
}

#######################################
# Check if SearXNG is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
searxng_inject::check_accessibility() {
    # Check if SearXNG is running
    if curl -s --max-time 5 "${SEARXNG_HOST}/healthz" 2>/dev/null | grep -q "OK"; then
        log::debug "SearXNG is accessible at $SEARXNG_HOST"
        return 0
    elif curl -s --max-time 5 "${SEARXNG_HOST}" 2>/dev/null | grep -q "SearXNG"; then
        log::debug "SearXNG is accessible at $SEARXNG_HOST"
        return 0
    else
        log::error "SearXNG is not accessible at $SEARXNG_HOST"
        log::info "Ensure SearXNG is running: ./scripts/resources/search/searxng/manage.sh --action start"
        return 1
    fi
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
searxng_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    SEARXNG_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added SearXNG rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
searxng_inject::execute_rollback() {
    if [[ ${#SEARXNG_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No SearXNG rollback actions to execute"
        return 0
    fi
    
    log::info "Executing SearXNG rollback actions..."
    
    local success_count=0
    local total_count=${#SEARXNG_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#SEARXNG_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${SEARXNG_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "SearXNG rollback completed: $success_count/$total_count actions successful"
    SEARXNG_ROLLBACK_ACTIONS=()
}

#######################################
# Validate engine configuration
# Arguments:
#   $1 - engines configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
searxng_inject::validate_engines() {
    local engines_config="$1"
    
    log::debug "Validating engine configurations..."
    
    # Check if engines is an array
    local engines_type
    engines_type=$(echo "$engines_config" | jq -r 'type')
    
    if [[ "$engines_type" != "array" ]]; then
        log::error "Engines configuration must be an array, got: $engines_type"
        return 1
    fi
    
    # Validate each engine
    local engine_count
    engine_count=$(echo "$engines_config" | jq 'length')
    
    # Common search engines
    local valid_engines=("google" "bing" "duckduckgo" "startpage" "qwant" "yahoo" "yandex" "wikipedia" "github" "gitlab" "stackoverflow" "arxiv" "pubmed" "youtube" "reddit" "twitter")
    
    for ((i=0; i<engine_count; i++)); do
        local engine
        engine=$(echo "$engines_config" | jq -c ".[$i]")
        
        # Check required fields
        local name enabled
        name=$(echo "$engine" | jq -r '.name // empty')
        enabled=$(echo "$engine" | jq -r '.enabled // true')
        
        if [[ -z "$name" ]]; then
            log::error "Engine at index $i missing required 'name' field"
            return 1
        fi
        
        # Warn if engine name is not in common list (but don't fail)
        local is_known=false
        for valid_name in "${valid_engines[@]}"; do
            if [[ "$name" == "$valid_name" ]]; then
                is_known=true
                break
            fi
        done
        
        if [[ "$is_known" == false ]]; then
            log::warn "Engine '$name' is not in the list of common engines"
        fi
        
        log::debug "Engine '$name' configuration is valid"
    done
    
    log::success "All engine configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
searxng_inject::validate_config() {
    local config="$1"
    
    log::info "Validating SearXNG injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in SearXNG injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_engines has_preferences has_outgoing has_server has_configurations
    has_engines=$(echo "$config" | jq -e '.engines' >/dev/null 2>&1 && echo "true" || echo "false")
    has_preferences=$(echo "$config" | jq -e '.preferences' >/dev/null 2>&1 && echo "true" || echo "false")
    has_outgoing=$(echo "$config" | jq -e '.outgoing' >/dev/null 2>&1 && echo "true" || echo "false")
    has_server=$(echo "$config" | jq -e '.server' >/dev/null 2>&1 && echo "true" || echo "false")
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_engines" == "false" && "$has_preferences" == "false" && "$has_outgoing" == "false" && "$has_server" == "false" && "$has_configurations" == "false" ]]; then
        log::error "SearXNG injection configuration must have 'engines', 'preferences', 'outgoing', 'server', or 'configurations'"
        return 1
    fi
    
    # Validate engines if present
    if [[ "$has_engines" == "true" ]]; then
        local engines
        engines=$(echo "$config" | jq -c '.engines')
        
        if ! searxng_inject::validate_engines "$engines"; then
            return 1
        fi
    fi
    
    # Validate preferences if present
    if [[ "$has_preferences" == "true" ]]; then
        local preferences
        preferences=$(echo "$config" | jq -c '.preferences')
        
        # Check preferences is an object
        local pref_type
        pref_type=$(echo "$preferences" | jq -r 'type')
        
        if [[ "$pref_type" != "object" ]]; then
            log::error "Preferences must be an object, got: $pref_type"
            return 1
        fi
    fi
    
    log::success "SearXNG injection configuration is valid"
    return 0
}

#######################################
# Configure engine
# Arguments:
#   $1 - engine configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::configure_engine() {
    local engine_config="$1"
    
    local name enabled categories weight timeout safe_search
    name=$(echo "$engine_config" | jq -r '.name')
    enabled=$(echo "$engine_config" | jq -r '.enabled // true')
    categories=$(echo "$engine_config" | jq -r '.categories // ["general"]')
    weight=$(echo "$engine_config" | jq -r '.weight // 1.0')
    timeout=$(echo "$engine_config" | jq -r '.timeout // 5.0')
    safe_search=$(echo "$engine_config" | jq -r '.safe_search // 0')
    
    log::info "Configuring engine: $name"
    
    # Create engines directory
    mkdir -p "$SEARXNG_ENGINES_DIR"
    
    # Save engine configuration
    local engine_file="${SEARXNG_ENGINES_DIR}/${name}.yml"
    cat > "$engine_file" << EOF
# SearXNG Engine Configuration: $name
name: $name
engine: $name
enabled: $enabled
categories: $(echo "$categories" | jq -r '.[]' | sed 's/^/  - /')
weight: $weight
timeout: $timeout
safe_search: $safe_search
configured: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF
    
    log::success "Configured engine: $name"
    
    # Add rollback action
    searxng_inject::add_rollback_action \
        "Remove engine configuration: $name" \
        "trash::safe_remove '${engine_file}' --temp"
    
    return 0
}

#######################################
# Apply preferences
# Arguments:
#   $1 - preferences JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::apply_preferences() {
    local preferences="$1"
    
    log::info "Applying SearXNG preferences..."
    
    # Create settings directory
    mkdir -p "$SEARXNG_SETTINGS_DIR"
    
    # Save preferences
    local pref_file="${SEARXNG_SETTINGS_DIR}/preferences.json"
    echo "$preferences" | jq '.' > "$pref_file"
    
    # Create YAML version for SearXNG
    local yaml_file="${SEARXNG_SETTINGS_DIR}/preferences.yml"
    cat > "$yaml_file" << EOF
# SearXNG Preferences
preferences:
  autocomplete: $(echo "$preferences" | jq -r '.autocomplete // "google"')
  language: $(echo "$preferences" | jq -r '.language // "en"')
  locale: $(echo "$preferences" | jq -r '.locale // "en"')
  safe_search: $(echo "$preferences" | jq -r '.safe_search // 0')
  results_per_page: $(echo "$preferences" | jq -r '.results_per_page // 10')
  theme: $(echo "$preferences" | jq -r '.theme // "simple"')
EOF
    
    log::success "Applied preferences"
    
    # Add rollback actions
    searxng_inject::add_rollback_action \
        "Remove preferences" \
        "trash::safe_remove '${pref_file}' --temp; trash::safe_remove '${yaml_file}' --temp"
    
    return 0
}

#######################################
# Apply outgoing settings
# Arguments:
#   $1 - outgoing settings JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::apply_outgoing() {
    local outgoing="$1"
    
    log::info "Applying SearXNG outgoing settings..."
    
    # Create settings directory
    mkdir -p "$SEARXNG_SETTINGS_DIR"
    
    # Save outgoing settings
    local outgoing_file="${SEARXNG_SETTINGS_DIR}/outgoing.yml"
    cat > "$outgoing_file" << EOF
# SearXNG Outgoing Settings
outgoing:
  request_timeout: $(echo "$outgoing" | jq -r '.request_timeout // 5.0')
  max_request_timeout: $(echo "$outgoing" | jq -r '.max_request_timeout // 15.0')
  pool_connections: $(echo "$outgoing" | jq -r '.pool_connections // 100')
  pool_maxsize: $(echo "$outgoing" | jq -r '.pool_maxsize // 20')
  enable_http2: $(echo "$outgoing" | jq -r '.enable_http2 // true')
  proxies:
    all://: []
EOF
    
    log::success "Applied outgoing settings"
    
    # Add rollback action
    searxng_inject::add_rollback_action \
        "Remove outgoing settings" \
        "trash::safe_remove '${outgoing_file}' --temp"
    
    return 0
}

#######################################
# Apply server settings
# Arguments:
#   $1 - server settings JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::apply_server() {
    local server="$1"
    
    log::info "Applying SearXNG server settings..."
    
    # Create settings directory
    mkdir -p "$SEARXNG_SETTINGS_DIR"
    
    # Save server settings
    local server_file="${SEARXNG_SETTINGS_DIR}/server.yml"
    cat > "$server_file" << EOF
# SearXNG Server Settings
server:
  secret_key: "$(echo "$server" | jq -r '.secret_key // "change-this-secret"')"
  limiter: $(echo "$server" | jq -r '.limiter // true')
  public_instance: $(echo "$server" | jq -r '.public_instance // false')
  image_proxy: $(echo "$server" | jq -r '.image_proxy // true')
  http_protocol_version: "1.1"
  method: "GET"
EOF
    
    log::success "Applied server settings"
    log::warn "Note: Server settings require SearXNG restart to take effect"
    
    # Add rollback action
    searxng_inject::add_rollback_action \
        "Remove server settings" \
        "trash::safe_remove '${server_file}' --temp"
    
    return 0
}

#######################################
# Inject engines
# Arguments:
#   $1 - engines configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::inject_engines() {
    local engines_config="$1"
    
    log::info "Injecting SearXNG engines..."
    
    local engine_count
    engine_count=$(echo "$engines_config" | jq 'length')
    
    if [[ "$engine_count" -eq 0 ]]; then
        log::info "No engines to inject"
        return 0
    fi
    
    local failed_engines=()
    
    for ((i=0; i<engine_count; i++)); do
        local engine
        engine=$(echo "$engines_config" | jq -c ".[$i]")
        
        local name
        name=$(echo "$engine" | jq -r '.name')
        
        if ! searxng_inject::configure_engine "$engine"; then
            failed_engines+=("$name")
        fi
    done
    
    # Create combined engines file
    if [[ -d "$SEARXNG_ENGINES_DIR" ]]; then
        local combined_file="${SEARXNG_SETTINGS_DIR}/engines.yml"
        
        {
            echo "# Combined SearXNG Engines Configuration"
            echo "engines:"
            
            for engine_file in "$SEARXNG_ENGINES_DIR"/*.yml; do
                if [[ -f "$engine_file" ]]; then
                    echo ""
                    sed 's/^/  /' "$engine_file"
                fi
            done
        } > "$combined_file"
        
        log::info "Created combined engines configuration"
    fi
    
    if [[ ${#failed_engines[@]} -eq 0 ]]; then
        log::success "All engines injected successfully"
        return 0
    else
        log::error "Failed to inject engines: ${failed_engines[*]}"
        return 1
    fi
}

#######################################
# Apply configurations
# Arguments:
#   $1 - configurations JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::apply_configurations() {
    local configurations="$1"
    
    log::info "Applying SearXNG configurations..."
    
    local config_count
    config_count=$(echo "$configurations" | jq 'length')
    
    if [[ "$config_count" -eq 0 ]]; then
        log::info "No configurations to apply"
        return 0
    fi
    
    # Save configuration settings
    local config_file="${SEARXNG_DATA_DIR}/config.json"
    echo "$configurations" > "$config_file"
    
    # Create settings.yml snippet
    local settings_snippet="${SEARXNG_SETTINGS_DIR}/custom.yml"
    cat > "$settings_snippet" << EOF
# Custom SearXNG Configuration
EOF
    
    for ((i=0; i<config_count; i++)); do
        local config_item
        config_item=$(echo "$configurations" | jq -c ".[$i]")
        
        local key value
        key=$(echo "$config_item" | jq -r '.key')
        value=$(echo "$config_item" | jq -r '.value')
        
        echo "$key: $value" >> "$settings_snippet"
    done
    
    log::success "Applied configuration settings"
    
    # Add rollback actions
    searxng_inject::add_rollback_action \
        "Remove configuration files" \
        "trash::safe_remove '${config_file}' --temp; trash::safe_remove '${settings_snippet}' --temp"
    
    return 0
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into SearXNG"
    
    # Check SearXNG accessibility (optional for file-based operations)
    searxng_inject::check_accessibility || true
    
    # Clear previous rollback actions
    SEARXNG_ROLLBACK_ACTIONS=()
    
    # Inject engines if present
    local has_engines
    has_engines=$(echo "$config" | jq -e '.engines' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_engines" == "true" ]]; then
        local engines
        engines=$(echo "$config" | jq -c '.engines')
        
        if ! searxng_inject::inject_engines "$engines"; then
            log::error "Failed to inject engines"
            searxng_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply preferences if present
    local has_preferences
    has_preferences=$(echo "$config" | jq -e '.preferences' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_preferences" == "true" ]]; then
        local preferences
        preferences=$(echo "$config" | jq -c '.preferences')
        
        if ! searxng_inject::apply_preferences "$preferences"; then
            log::error "Failed to apply preferences"
            searxng_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply outgoing settings if present
    local has_outgoing
    has_outgoing=$(echo "$config" | jq -e '.outgoing' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_outgoing" == "true" ]]; then
        local outgoing
        outgoing=$(echo "$config" | jq -c '.outgoing')
        
        if ! searxng_inject::apply_outgoing "$outgoing"; then
            log::error "Failed to apply outgoing settings"
            searxng_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply server settings if present
    local has_server
    has_server=$(echo "$config" | jq -e '.server' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_server" == "true" ]]; then
        local server
        server=$(echo "$config" | jq -c '.server')
        
        if ! searxng_inject::apply_server "$server"; then
            log::error "Failed to apply server settings"
            searxng_inject::execute_rollback
            return 1
        fi
    fi
    
    # Apply configurations if present
    local has_configurations
    has_configurations=$(echo "$config" | jq -e '.configurations' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_configurations" == "true" ]]; then
        local configurations
        configurations=$(echo "$config" | jq -c '.configurations')
        
        if ! searxng_inject::apply_configurations "$configurations"; then
            log::error "Failed to apply configurations"
            searxng_inject::execute_rollback
            return 1
        fi
    fi
    
    log::success "✅ SearXNG data injection completed"
    
    # Note about applying settings
    if [[ "$has_engines" == "true" || "$has_server" == "true" || "$has_outgoing" == "true" ]]; then
        log::warn "Note: Some settings require SearXNG restart to take effect"
        log::info "Restart SearXNG: ./scripts/resources/search/searxng/manage.sh --action restart"
    fi
    
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
searxng_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking SearXNG injection status"
    
    # Check SearXNG accessibility
    local is_running=false
    if searxng_inject::check_accessibility; then
        is_running=true
        log::success "✅ SearXNG is running"
    else
        log::warn "⚠️  SearXNG is not running"
    fi
    
    # Check engine configurations
    if [[ -d "$SEARXNG_ENGINES_DIR" ]]; then
        local engine_count
        engine_count=$(find "$SEARXNG_ENGINES_DIR" -name "*.yml" 2>/dev/null | wc -l)
        
        if [[ "$engine_count" -gt 0 ]]; then
            log::info "Found $engine_count engine configurations"
            
            # List enabled engines
            local enabled_count
            enabled_count=$(grep -l "enabled: true" "$SEARXNG_ENGINES_DIR"/*.yml 2>/dev/null | wc -l)
            
            if [[ "$enabled_count" -gt 0 ]]; then
                log::info "  - $enabled_count engines enabled"
            fi
        else
            log::info "No engine configurations found"
        fi
    else
        log::info "Engines directory does not exist"
    fi
    
    # Check preferences
    if [[ -f "${SEARXNG_SETTINGS_DIR}/preferences.json" ]]; then
        log::info "Preferences configured"
        
        local theme language
        theme=$(jq -r '.theme // "unknown"' "${SEARXNG_SETTINGS_DIR}/preferences.json")
        language=$(jq -r '.language // "unknown"' "${SEARXNG_SETTINGS_DIR}/preferences.json")
        
        log::info "  - Theme: $theme"
        log::info "  - Language: $language"
    else
        log::info "No preferences configured"
    fi
    
    # Check outgoing settings
    if [[ -f "${SEARXNG_SETTINGS_DIR}/outgoing.yml" ]]; then
        log::info "Outgoing settings configured"
    fi
    
    # Check server settings
    if [[ -f "${SEARXNG_SETTINGS_DIR}/server.yml" ]]; then
        log::info "Server settings configured"
    fi
    
    # Test API if running
    if [[ "$is_running" == true ]]; then
        log::info "Testing SearXNG API..."
        
        # Test search endpoint
        local search_test
        search_test=$(curl -s "${SEARXNG_HOST}/search?q=test&format=json" 2>/dev/null || echo "{}")
        
        if echo "$search_test" | jq -e '.results' >/dev/null 2>&1; then
            log::success "✅ SearXNG search API is responding"
            
            # Get stats if available
            local stats
            stats=$(curl -s "${SEARXNG_HOST}/stats" 2>/dev/null || echo "{}")
            
            if echo "$stats" | jq -e '.engines' >/dev/null 2>&1; then
                local active_engines
                active_engines=$(echo "$stats" | jq '.engines | length // 0')
                log::info "  - Active engines: $active_engines"
            fi
        else
            log::error "❌ SearXNG search API not responding properly"
        fi
        
        # Test preferences endpoint
        if curl -s "${SEARXNG_HOST}/preferences" 2>/dev/null | grep -q "Preferences"; then
            log::info "✅ Preferences endpoint accessible"
        fi
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
searxng_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        searxng_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            searxng_inject::validate_config "$config"
            ;;
        "--inject")
            searxng_inject::inject_data "$config"
            ;;
        "--status")
            searxng_inject::check_status "$config"
            ;;
        "--rollback")
            searxng_inject::execute_rollback
            ;;
        "--help")
            searxng_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            searxng_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        searxng_inject::usage
        exit 1
    fi
    
    searxng_inject::main "$@"
fi