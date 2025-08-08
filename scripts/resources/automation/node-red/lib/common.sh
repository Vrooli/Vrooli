#!/usr/bin/env bash
# Node-RED Common Utility Functions
# Shared utilities used across all modules

# Source shared secrets management library
# Use the same project root detection method as the secrets library
_node_red_detect_project_root() {
    local current_dir
    current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    # Walk up directory tree looking for .vrooli directory
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/.vrooli" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir="$(dirname "$current_dir")"
    done
    
    # Fallback: assume we're in scripts and go up to project root
    echo "/home/matthalloran8/Vrooli"
}

PROJECT_ROOT="$(_node_red_detect_project_root)"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/service/secrets.sh"

#######################################
# Check if Docker is installed
# Returns: 0 if installed, 1 otherwise
#######################################
node_red::check_docker() {
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        log::info "Please install Docker first: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        log::info "Start Docker with: sudo systemctl start docker"
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "Current user doesn't have Docker permissions"
        log::info "Add user to docker group: sudo usermod -aG docker $USER"
        log::info "Then log out and back in for changes to take effect"
        return 1
    fi
    
    return 0
}

#######################################
# Check if Node-RED container exists
# Returns: 0 if exists, 1 otherwise
#######################################
node_red::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"
}

#######################################
# Check if Node-RED is installed
# Returns: 0 if installed, 1 otherwise
#######################################
node_red::is_installed() {
    docker container inspect "$CONTAINER_NAME" >/dev/null 2>&1
}

#######################################
# Check if Node-RED is running
# Returns: 0 if running, 1 otherwise
#######################################
node_red::is_running() {
    local state=$(docker container inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null || echo "false")
    [[ "$state" == "true" ]]
}

#######################################
# Check if Node-RED is healthy and responding
# Returns: 0 if healthy, 1 otherwise
#######################################
node_red::is_healthy() {
    if system::is_command "curl"; then
        # Try multiple times as Node-RED takes time to fully initialize
        local attempts=0
        while [ $attempts -lt 5 ]; do
            if curl -f -s --max-time 5 "http://localhost:$RESOURCE_PORT" >/dev/null 2>&1; then
                return 0
            fi
            attempts=$((attempts + 1))
            sleep 2
        done
    fi
    return 1
}

#######################################
# Wait for Node-RED to be ready
# Returns: 0 if ready, 1 if timeout
#######################################
node_red::wait_for_ready() {
    local max_attempts=${NODE_RED_HEALTH_CHECK_MAX_ATTEMPTS:-30}
    local attempt=0
    
    node_red::show_waiting_message
    
    while [[ $attempt -lt $max_attempts ]]; do
        if curl -s -f "http://localhost:$RESOURCE_PORT" >/dev/null 2>&1; then
            log::success "Node-RED is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 2
    done
    
    node_red::show_startup_timeout_error "$max_attempts"
    return 1
}

#######################################
# Generate secure random secret
#######################################
node_red::generate_secret() {
    # Use multiple sources for randomness
    if system::is_command "openssl"; then
        openssl rand -hex 32 2>/dev/null
    elif [[ -r /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 64
    else
        # Fallback to timestamp-based secret
        echo "node-red-$(date +%s)-$RANDOM" | sha256sum | cut -c1-64
    fi
}

#######################################
# Create Node-RED data directories if needed
#######################################
node_red::create_directories() {
    log::info "Creating Node-RED data directories..."
    
    # Create flows directory if it doesn't exist
    mkdir -p "$SCRIPT_DIR/flows" || {
        log::error "Failed to create flows directory"
        return 1
    }
    
    # Create nodes directory if it doesn't exist
    mkdir -p "$SCRIPT_DIR/nodes" || {
        log::error "Failed to create nodes directory"
        return 1
    }
    
    return 0
}

#######################################
# Update Vrooli resource configuration
# SAFE VERSION - Uses proper ConfigurationManager to prevent data loss
#######################################
node_red::update_resource_config() {
    log::info "Updating Node-RED resource configuration..."
    
    # Source the common resources functions for safe config management
    local resources_common_script="$(cd "$SCRIPT_DIR/../.." && pwd)/common.sh"
    if [[ -f "$resources_common_script" ]]; then
        source "$resources_common_script"
    else
        log::error "Cannot find resources common.sh script for safe configuration management"
        return 1
    fi
    
    # Create Node-RED specific configuration
    local node_red_config=$(cat << EOF
{
    "enabled": true,
    "baseUrl": "http://localhost:$RESOURCE_PORT",
    "adminUrl": "http://localhost:$RESOURCE_PORT/admin",
    "healthCheck": {
        "endpoint": "/flows",
        "intervalMs": 60000,
        "timeoutMs": 5000
    },
    "flows": {
        "directory": "/data/flows",
        "autoBackup": true,
        "backupInterval": "1h"
    }
}
EOF
    )
    
    # Use the safe configuration update function from common.sh
    if resources::update_config "automation" "node-red" "http://localhost:$RESOURCE_PORT" "$node_red_config"; then
        log::success "Node-RED configuration updated safely"
        return 0
    else
        log::error "Failed to update Node-RED configuration using safe method"
        return 1
    fi
}

#######################################
# Remove Node-RED from resource configuration
#######################################
node_red::remove_resource_config() {
    local config_file
    config_file="$(secrets::get_project_config_file)"
    
    if [[ -f "$config_file" ]]; then
        # Remove node-red configuration
        local tmp_file=$(mktemp)
        jq 'del(.services.automation."node-red")' "$config_file" > "$tmp_file" && mv "$tmp_file" "$config_file"
    fi
}

#######################################
# Create default settings.js if it doesn't exist
#######################################
node_red::create_default_settings() {
    if [[ -f "$SCRIPT_DIR/settings.js" ]]; then
        return 0  # Settings already exist
    fi
    
    node_red::show_creating_settings
    
    cat > "$SCRIPT_DIR/settings.js" << 'EOF'
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

#######################################
# Check if port is available
# Arguments: port number
# Returns: 0 if available, 1 if in use
#######################################
node_red::check_port() {
    local port="$1"
    
    if command -v ss >/dev/null 2>&1; then
        ! ss -tlnp | grep -q ":$port "
    elif command -v netstat >/dev/null 2>&1; then
        ! netstat -tlnp | grep -q ":$port "
    else
        # Fallback: try to bind to the port briefly
        ! timeout 1 bash -c "echo >/dev/tcp/localhost/$port" 2>/dev/null
    fi
}

#######################################
# Validate JSON file
# Arguments: file path
# Returns: 0 if valid, 1 if invalid
#######################################
node_red::validate_json() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    jq . "$file" >/dev/null 2>&1
}

#######################################
# Get container logs with error handling
# Arguments: follow (optional, "yes" to follow)
#######################################
node_red::get_logs() {
    local follow="${1:-no}"
    
    if ! node_red::is_installed; then
        node_red::show_not_installed_error
        return 1
    fi
    
    if [[ "$follow" == "yes" ]]; then
        docker logs -f "$CONTAINER_NAME"
    else
        docker logs --tail "$NODE_RED_LOG_LINES" "$CONTAINER_NAME"
    fi
}

#######################################
# Get container resource usage
#######################################
node_red::get_resource_usage() {
    if ! node_red::is_running; then
        return 1
    fi
    
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$CONTAINER_NAME"
}

#######################################
# Check container health status
#######################################
node_red::get_health_status() {
    if ! node_red::is_installed; then
        echo "not-installed"
        return 1
    fi
    
    docker inspect --format='{{.State.Health.Status}}' "$CONTAINER_NAME" 2>/dev/null || echo "unknown"
}