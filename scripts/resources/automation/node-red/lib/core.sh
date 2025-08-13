#!/usr/bin/env bash
# Node-RED Core Functions - Consolidated Node-RED-specific logic
# All generic operations delegated to shared libraries

# Source shared libraries
NODE_RED_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${NODE_RED_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/http-utils.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/status-engine.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/health-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/backup-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/init-framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/inject_framework.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/wait-utils.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

# Node-RED Configuration Constants
# These are set in config/defaults.sh as readonly
# Only set non-readonly variables here
# Variables that aren't set as readonly in defaults.sh
: "${NODE_RED_FLOWS_FILE:=flows.json}"
: "${NODE_RED_CREDENTIALS_FILE:=flows_cred.json}"
: "${NODE_RED_SETTINGS_FILE:=settings.js}"
: "${BASIC_AUTH:=yes}"
: "${AUTH_USERNAME:=admin}"
: "${BUILD_IMAGE:=no}"

node_red::get_init_config() {
    local auth_password="${1:-}"
    
    # Select image (custom if available)
    local image_to_use="$NODE_RED_IMAGE"
    if [[ "$BUILD_IMAGE" == "yes" ]] && docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${NODE_RED_CUSTOM_IMAGE}$"; then
        image_to_use="$NODE_RED_CUSTOM_IMAGE"
    fi
    
    local timezone
    timezone=$(timedatectl show -p Timezone --value 2>/dev/null || echo 'UTC')
    
    local volumes_array="[\"${NODE_RED_DATA_DIR}:/data\""
    if [[ "$BUILD_IMAGE" == "yes" ]]; then
        volumes_array+=",\"/var/run/docker.sock:/var/run/docker.sock:rw\""
        volumes_array+=",\"${PWD}:/workspace:rw\""
        volumes_array+=",\"$HOME:/host/home:rw\""
    fi
    volumes_array+="]"
    
    local config='{
        "resource_name": "node-red",
        "container_name": "'$NODE_RED_CONTAINER_NAME'",
        "data_dir": "'$NODE_RED_DATA_DIR'",
        "port": '$NODE_RED_PORT',
        "image": "'$image_to_use'",
        "env_vars": {
            "TZ": "'$timezone'",
            "NODE_RED_ENABLE_PROJECTS": "false",
            "NODE_RED_ENABLE_SAFE_MODE": "false"
        },
        "volumes": '$volumes_array',
        "networks": ["'$NODE_RED_NETWORK_NAME'"],
        "first_run_check": "node_red::is_first_run",
        "setup_func": "node_red::first_time_setup",
        "wait_for_ready": "node_red::wait_for_ready"
    }'
    
    if [[ "$BASIC_AUTH" == "yes" && -n "$auth_password" ]]; then
        config=$(echo "$config" | jq --arg username "$AUTH_USERNAME" --arg password "$auth_password" '
            .env_vars += {
                "NODE_RED_CREDENTIAL_SECRET": $password
            }
        ')
    fi
    
    echo "$config"
}

node_red::is_first_run() {
    [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]
}

# First-time setup for Node-RED
# Sets up initial flows and configuration
node_red::first_time_setup() {
    log::info "Setting up Node-RED for first run..."
    
    mkdir -p "$NODE_RED_DATA_DIR"
    
    # Copy default settings if it doesn't exist
    if [[ ! -f "$NODE_RED_DATA_DIR/$NODE_RED_SETTINGS_FILE" ]]; then
        if [[ -f "${NODE_RED_SCRIPT_DIR}/config/settings.js" ]]; then
            cp "${NODE_RED_SCRIPT_DIR}/config/settings.js" "$NODE_RED_DATA_DIR/$NODE_RED_SETTINGS_FILE"
        fi
    fi
    
    # Copy default flows if available
    local default_flows="${NODE_RED_SCRIPT_DIR}/examples/default-flows.json"
    if [[ -f "$default_flows" && ! -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        cp "$default_flows" "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE"
        log::info "Installed default flows"
    fi
    
    chown -R 1000:1000 "$NODE_RED_DATA_DIR" 2>/dev/null || true
}

# Wait for Node-RED to be ready
node_red::wait_for_ready() {
    local max_attempts=30
    local attempt=1
    
    log::info "Waiting for Node-RED to be ready..."
    
    while [[ $attempt -le $max_attempts ]]; do
        if node_red::is_responding; then
            log::success "Node-RED is ready!"
            return 0
        fi
        
        log::info "Attempt $attempt/$max_attempts: Node-RED not ready yet..."
        sleep 2
        ((attempt++))
    done
    
    log::error "Node-RED failed to become ready within timeout"
    return 1
}

node_red::is_responding() {
    local url="http://localhost:$NODE_RED_PORT"
    
    # Try to get the Node-RED editor page
    if curl -s -f "$url" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

# Install Node-RED using init framework
node_red::install() {
    log::info "Installing Node-RED using init framework..."
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    local auth_password=""
    if [[ "$BASIC_AUTH" == "yes" ]]; then
        auth_password=$(openssl rand -hex 8)
        log::info "Generated authentication password for admin user"
    fi
    
    local init_config
    init_config=$(node_red::get_init_config "$auth_password")
    
    if init::setup_resource "$init_config"; then
        if [[ "$BASIC_AUTH" == "yes" ]]; then
            log::success "Node-RED installed successfully!"
            log::info "Access URL: http://localhost:$NODE_RED_PORT"
            log::info "Username: $AUTH_USERNAME"
            log::info "Password: $auth_password"
            log::warn "Save these credentials - they won't be shown again"
        else
            log::success "Node-RED installed successfully!"
            log::info "Access URL: http://localhost:$NODE_RED_PORT"
            log::warn "No authentication configured - consider enabling it for security"
        fi
        return 0
    else
        log::error "Node-RED installation failed"
        return 1
    fi
}

# Uninstall Node-RED
node_red::uninstall() {
    log::info "Uninstalling Node-RED..."
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    # Stop and remove container
    if docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        docker::stop_container "$NODE_RED_CONTAINER_NAME"
        docker::remove_container "$NODE_RED_CONTAINER_NAME"
    fi
    
    # Remove network if it exists and is empty
    docker::remove_network_if_empty "$NODE_RED_NETWORK_NAME"
    
    log::success "Node-RED uninstalled successfully"
    return 0
}

# Start Node-RED service
node_red::start() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not installed. Run: $0 --action install"
        return 1
    fi
    
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::info "Node-RED is already running"
        return 0
    fi
    
    log::info "Starting Node-RED..."
    docker::start_container "$NODE_RED_CONTAINER_NAME"
    
    if node_red::wait_for_ready; then
        log::success "Node-RED started successfully"
        log::info "Access URL: http://localhost:$NODE_RED_PORT"
        return 0
    else
        log::error "Node-RED failed to start properly"
        return 1
    fi
}

# Stop Node-RED service
node_red::stop() {
    if ! docker::check_daemon; then
        return 1
    fi
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::info "Node-RED is not installed"
        return 0
    fi
    
    if ! docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        log::info "Node-RED is already stopped"
        return 0
    fi
    
    log::info "Stopping Node-RED..."
    docker::stop_container "$NODE_RED_CONTAINER_NAME"
    log::success "Node-RED stopped successfully"
    return 0
}

# Restart Node-RED service
node_red::restart() {
    log::info "Restarting Node-RED..."
    node_red::stop
    sleep 2
    node_red::start
}

# Show Node-RED logs
node_red::view_logs() {
    local lines="${1:-100}"
    local follow="${2:-no}"
    
    if ! docker::check_daemon; then
        return 1
    fi
    
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not installed"
        return 1
    fi
    
    local args="--tail $lines"
    if [[ "$follow" == "yes" ]]; then
        args+=" --follow"
    fi
    
    # shellcheck disable=SC2086
    docker logs $args "$NODE_RED_CONTAINER_NAME"
}

node_red::get_status_config() {
    echo '{
        "resource": {
            "name": "node-red",
            "category": "automation",
            "description": "Flow-based programming for event-driven applications",
            "port": '$NODE_RED_PORT',
            "container_name": "'$NODE_RED_CONTAINER_NAME'",
            "data_dir": "'$NODE_RED_DATA_DIR'"
        },
        "endpoints": {
            "ui": "'$NODE_RED_BASE_URL'",
            "api": "'$NODE_RED_BASE_URL'/flows",
            "health": "'$NODE_RED_BASE_URL'"
        },
        "health_tiers": {
            "healthy": "All systems operational",
            "degraded": "Node-RED responding but flows may need configuration",
            "unhealthy": "Service not responding - Try: ./manage.sh --action restart"
        }
    }'
}

# Node-RED Utility Functions
# Consolidated from lib/common.sh

# Generate secure random secret
node_red::generate_secret() {
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 32 2>/dev/null
    elif [[ -r /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64
    else
        # Fallback to timestamp-based secret
        echo "node-red-$(date +%s)-$RANDOM" | sha256sum | cut -c1-64
    fi
}

node_red::create_default_settings() {
    local settings_file="${NODE_RED_SCRIPT_DIR}/config/settings.js"
    
    if [[ -f "$settings_file" ]]; then
        return 0  # Settings already exist
    fi
    
    log::info "Creating default Node-RED settings..."
    
    cat > "$settings_file" << 'EOF'
module.exports = {
    // Flow file settings
    flowFile: 'flows.json',
    flowFilePretty: true,
    
    // User directory
    userDir: '/data/',
    
    // Node-RED settings
    uiPort: process.env.PORT || 1880,
    
    // Logging
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    },
    
    // Editor theme
    editorTheme: {
        theme: "dark",
        projects: {
            enabled: false
        }
    },
    
    // Function node settings
    functionGlobalContext: {
        // Add global libraries here
        // os: require('os'),
    },
    
    // Allow external npm modules in function nodes
    functionExternalModules: true,
    
    // Debug settings
    debugMaxLength: 1000,
    
    // Exec node settings
    execMaxBufferSize: 10000000, // 10MB
    
    // HTTP request timeout
    httpRequestTimeout: 120000, // 2 minutes
}
EOF
}

node_red::update() {
    if ! docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        log::error "Node-RED is not installed. Run: $0 --action install"
        return 1
    fi
    
    log::info "Updating Node-RED to latest version..."
    
    # Pull latest official image if not using custom
    if [[ "$BUILD_IMAGE" != "yes" ]]; then
        log::info "Pulling latest Node-RED image..."
        docker::pull_image "$NODE_RED_IMAGE"
    fi
    
    # Restart with updated image
    log::info "Restarting Node-RED with updated image..."
    node_red::restart
    
    log::success "Node-RED updated successfully"
    return 0
}

# Register Node-RED with injection framework
node_red::register_injection_framework() {
    inject_framework::register "node-red" \
        --service-host "http://localhost:$NODE_RED_PORT" \
        --health-endpoint "/" \
        --validate-func "node_red::validate_injection_config" \
        --inject-func "node_red::perform_injection" \
        --health-func "node_red::is_responding"
}

# Inject flows into Node-RED
node_red::inject() {
    local injection_config="${INJECTION_CONFIG:-}"
    
    if [[ -z "$injection_config" ]]; then
        log::error "Missing required --injection-config parameter"
        log::info "Usage: manage.sh --action inject --injection-config '{\"flows\":[...]}'"
        return 1
    fi
    
    node_red::register_injection_framework
    inject_framework::main --inject --config "$injection_config"
}

# Validate injection configuration
node_red::validate_injection() {
    local injection_config="${INJECTION_CONFIG:-}"
    
    if [[ -z "$injection_config" ]]; then
        log::error "Missing required --injection-config parameter"
        log::info "Usage: manage.sh --action validate-injection --injection-config '{\"flows\":[...]}'"
        return 1
    fi
    
    node_red::register_injection_framework
    inject_framework::main --validate --config "$injection_config"
}

node_red::validate_injection_config() {
    local config="$1"
    
    if ! echo "$config" | jq empty 2>/dev/null; then
        log::error "Invalid JSON configuration"
        return 1
    fi
    
    if ! echo "$config" | jq -e '.flows' >/dev/null 2>&1; then
        log::error "Configuration missing 'flows' array"
        return 1
    fi
    
    local flow_count
    flow_count=$(echo "$config" | jq '.flows | length')
    
    local i=0
    while [[ $i -lt $flow_count ]]; do
        local flow
        flow=$(echo "$config" | jq -r ".flows[$i]")
        
        local name file
        name=$(echo "$flow" | jq -r '.name // empty')
        file=$(echo "$flow" | jq -r '.file // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Flow at index $i missing 'name' field"
            return 1
        fi
        
        if [[ -z "$file" ]]; then
            log::error "Flow '$name' missing 'file' field"
            return 1
        fi
        
        # Resolve and validate file path
        local resolved_file
        if command -v inject_framework::resolve_file_path >/dev/null 2>&1; then
            resolved_file=$(inject_framework::resolve_file_path "$file")
        else
            resolved_file="$file"
        fi
        
        if [[ ! -f "$resolved_file" ]]; then
            log::error "Flow file not found for '$name': $resolved_file"
            return 1
        fi
        
        if ! jq empty "$resolved_file" 2>/dev/null; then
            log::error "Invalid JSON in flow file '$name': $resolved_file"
            return 1
        fi
        
        ((i++))
    done
    
    log::success "Configuration validation passed"
    return 0
}

# Perform actual flow injection
node_red::perform_injection() {
    local config="$1"
    
    if ! node_red::is_responding; then
        log::error "Node-RED is not accessible at http://localhost:$NODE_RED_PORT"
        return 1
    fi
    
    local flow_count
    flow_count=$(echo "$config" | jq '.flows | length')
    
    log::info "Injecting $flow_count flow(s) into Node-RED..."
    
    # Backup current flows if they exist
    if [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        local backup_file="$NODE_RED_DATA_DIR/${NODE_RED_FLOWS_FILE}.backup.$(date +%Y%m%d-%H%M%S)"
        cp "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" "$backup_file"
        log::info "Backed up current flows to: $(basename "$backup_file")"
    fi
    
    # Process each flow
    local i=0
    local success=true
    while [[ $i -lt $flow_count ]]; do
        local flow
        flow=$(echo "$config" | jq -r ".flows[$i]")
        
        local name file
        name=$(echo "$flow" | jq -r '.name')
        file=$(echo "$flow" | jq -r '.file')
        
        # Resolve file path
        local resolved_file
        if command -v inject_framework::resolve_file_path >/dev/null 2>&1; then
            resolved_file=$(inject_framework::resolve_file_path "$file")
        else
            resolved_file="$file"
        fi
        
        log::info "Injecting flow '$name'..."
        
        # Try API injection first
        if curl -s --max-time 10 \
            -X POST \
            -H "Content-Type: application/json" \
            -d "@$resolved_file" \
            "http://localhost:$NODE_RED_PORT/flows" >/dev/null 2>&1; then
            log::success "Flow '$name' injected via API"
        else
            # Fallback to file copy method
            if [[ -d "$NODE_RED_DATA_DIR" ]]; then
                cp "$resolved_file" "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE"
                log::success "Flow '$name' injected via file copy"
            else
                log::error "Failed to inject flow '$name'"
                success=false
                break
            fi
        fi
        
        ((i++))
    done
    
    if [[ "$success" == true ]]; then
        log::success "All flows injected successfully"
        log::info "Node-RED may need to be restarted to load new flows"
        return 0
    else
        log::error "Flow injection failed"
        return 1
    fi
}

# Optimized Flow Management API - Data-driven approach
node_red::flow_operation() {
    local operation="$1"
    shift
    
    # Pre-check: Node-RED must be running
    docker::container_running "$NODE_RED_CONTAINER_NAME" || {
        log::error "Node-RED is not running"
        return 1
    }
    
    local base_url="http://localhost:$NODE_RED_PORT"
    local response http_code
    
    # Define operation configurations
    case "$operation" in
        list)
            response=$(http::request "GET" "$base_url/flows")
            http_code=$?
            if [[ $http_code -ne 200 ]]; then
                log::error "Failed to fetch flows (HTTP $http_code)"
                return 1
            fi
            if [[ -n "$response" ]]; then
                echo "$response" | jq -r '.[] | select(.type == "tab") | "ID: \(.id)\nName: \(.label)\nDisabled: \(.disabled // false)\n"' 2>/dev/null || log::info "No flows found or empty response"
            else
                log::info "Empty response from API"
            fi
            ;;
            
        export)
            local output_file="${1:-node-red-flows-$(date +%Y%m%d-%H%M%S).json}"
            response=$(http::request "GET" "$base_url/flows")
            http_code=$?
            [[ $http_code -eq 200 ]] || { log::error "Export failed (HTTP $http_code)"; return 1; }
            echo "$response" | jq . > "$output_file"
            log::success "Flows exported to: $output_file"
            ;;
            
        import)
            local flow_file="$1"
            [[ -f "$flow_file" ]] || { log::error "Flow file not found: $flow_file"; return 1; }
            jq empty "$flow_file" 2>/dev/null || { log::error "Invalid JSON in flow file"; return 1; }
            
            # Backup current flows
            local backup="flows-backup-$(date +%Y%m%d-%H%M%S).json"
            http::request "GET" "$base_url/flows" > "$backup" 2>/dev/null
            
            # Import flows
            response=$(http::request "POST" "$base_url/flows" "$(cat "$flow_file")" "Content-Type: application/json")
            http_code=$?
            [[ $http_code -eq 200 || $http_code -eq 204 ]] && log::success "Flows imported from: $flow_file" || {
                log::error "Import failed (HTTP $http_code)"
                return 1
            }
            ;;
            
        execute)
            local endpoint="$1" data="${2:-}"
            [[ -n "$endpoint" ]] || { log::error "Endpoint required"; return 1; }
            
            if [[ -n "$data" ]]; then
                response=$(http::request "POST" "$base_url$endpoint" "$data" "Content-Type: application/json")
            else
                response=$(http::request "GET" "$base_url$endpoint")
            fi
            http_code=$?
            [[ $http_code -eq 200 || $http_code -eq 201 || $http_code -eq 204 ]] && {
                log::success "Flow executed"
                [[ -n "$response" ]] && echo "$response"
            } || { log::error "Execution failed (HTTP $http_code)"; return 1; }
            ;;
            
        enable|disable)
            local flow_id="$1"
            [[ -n "$flow_id" ]] || { log::error "Flow ID required"; return 1; }
            
            # Get current flow
            response=$(http::request "GET" "$base_url/flow/$flow_id")
            [[ $? -eq 200 ]] || { log::error "Flow not found: $flow_id"; return 1; }
            
            # Update disabled flag
            local disabled="$([[ "$operation" == "disable" ]] && echo "true" || echo "false")"
            response=$(echo "$response" | jq ".disabled = $disabled" | http::request "PUT" "$base_url/flow/$flow_id" "-" "Content-Type: application/json")
            [[ $? -eq 200 || $? -eq 204 ]] && log::success "Flow ${operation}d" || {
                log::error "Failed to $operation flow"
                return 1
            }
            ;;
            
        *)
            log::error "Unknown operation: $operation (valid: list|export|import|execute|enable|disable)"
            return 1
            ;;
    esac
}

# Status Functions (merged from status.sh)

node_red::status() {
    docker::check_daemon || return 1
    status::display_unified_status "$(node_red::get_status_config)" "node_red::display_additional_info"
}

node_red::display_additional_info() {
    echo -e "\n🔧 Node-RED Specific Information:"
    
    # Flows info
    if [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]]; then
        local flow_count=$(jq length "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" 2>/dev/null || echo "0")
        echo "   📋 Flows: $flow_count loaded"
    else
        echo "   📋 Flows: No flows file found"
    fi
    
    # Credentials & Auth
    [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_CREDENTIALS_FILE" ]] && 
        echo "   🔐 Credentials: File exists" || echo "   🔐 Credentials: No credentials file"
    
    [[ "$BASIC_AUTH" == "yes" ]] && 
        echo "   🛡️  Authentication: Enabled (username: $AUTH_USERNAME)" || echo "   🛡️  Authentication: Disabled"
    
    # Version (if running)
    if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
        local version=$(docker exec "$NODE_RED_CONTAINER_NAME" node -e "console.log(require('/usr/src/node-red/package.json').version)" 2>/dev/null || echo "unknown")
        echo "   📦 Version: $version"
        
        # Resource usage
        echo -e "\n💾 Resource Usage:"
        local stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}" "$NODE_RED_CONTAINER_NAME" 2>/dev/null | tail -n +2)
        if [[ -n "$stats" ]]; then
            echo "   🔥 CPU: $(echo "$stats" | awk '{print $1}')"
            echo "   🧠 Memory: $(echo "$stats" | awk '{print $2}')"
        fi
    fi
}

node_red::get_status_json() {
    local container_status="not_installed" service_status="stopped" port_accessible=false version="unknown"
    local flows_count=0 has_credentials=false
    
    if docker::container_exists "$NODE_RED_CONTAINER_NAME"; then
        if docker::container_running "$NODE_RED_CONTAINER_NAME"; then
            container_status="running"
            service_status=$(http::check_endpoint "http://localhost:$NODE_RED_PORT" && echo "running" || echo "starting")
            port_accessible=$([[ "$service_status" == "running" ]] && echo "true" || echo "false")
            version=$(docker exec "$NODE_RED_CONTAINER_NAME" node -e "console.log(require('/usr/src/node-red/package.json').version)" 2>/dev/null || echo "unknown")
        else
            container_status="stopped"
        fi
    fi
    
    [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" ]] && flows_count=$(jq length "$NODE_RED_DATA_DIR/$NODE_RED_FLOWS_FILE" 2>/dev/null || echo 0)
    [[ -f "$NODE_RED_DATA_DIR/$NODE_RED_CREDENTIALS_FILE" ]] && has_credentials=true
    
    cat <<EOF
{
    "service": "Node-RED",
    "container_status": "$container_status",
    "service_status": "$service_status",
    "port_accessible": $port_accessible,
    "version": "$version",
    "configuration": {
        "port": $NODE_RED_PORT,
        "data_dir": "$NODE_RED_DATA_DIR",
        "authentication_enabled": $([[ "$BASIC_AUTH" == "yes" ]] && echo "true" || echo "false"),
        "flows_count": $flows_count,
        "has_credentials": $has_credentials
    },
    "urls": {
        "editor": "http://localhost:$NODE_RED_PORT",
        "settings": "http://localhost:$NODE_RED_PORT/settings",
        "flows": "http://localhost:$NODE_RED_PORT/flows"
    },
    "timestamp": "$(date -Iseconds)"
}
EOF
}