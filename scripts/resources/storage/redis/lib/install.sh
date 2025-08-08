#!/usr/bin/env bash
# Redis Installation Functions
# Functions for installing and uninstalling Redis resource

# Source shared secrets management library
# Use the same project root detection method as the secrets library
_redis_install_detect_project_root() {
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

PROJECT_ROOT="$(_redis_install_detect_project_root)"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/service/secrets.sh"

#######################################
# Install Redis resource
# Returns: 0 on success, 1 on failure
#######################################
redis::install::main() {
    log::info "${MSG_INSTALL_STARTING}"
    
    # Check if already installed
    if redis::common::container_exists; then
        if redis::common::is_running; then
            log::info "${MSG_ALREADY_INSTALLED}"
            redis::status::show
            return 0
        else
            log::info "Redis container exists but is not running. Starting..."
            return redis::docker::start
        fi
    fi
    
    # Pre-installation checks
    if ! redis::install::check_prerequisites; then
        return 1
    fi
    
    # Check port availability
    if ! redis::common::is_port_available "$REDIS_PORT"; then
        log::error "${MSG_ERROR_PORT_IN_USE}"
        log::info "Current port usage for ${REDIS_PORT}:"
        netstat -tuln | grep ":${REDIS_PORT} " || ss -tuln | grep ":${REDIS_PORT} "
        return 1
    fi
    
    # Pull image
    if ! redis::docker::pull_image; then
        return 1
    fi
    
    # Create and start container
    if ! redis::docker::create_container; then
        return 1
    fi
    
    # Wait for Redis to be ready
    if ! redis::common::wait_for_ready; then
        log::error "${MSG_INSTALL_FAILED}"
        redis::install::cleanup_failed_install
        return 1
    fi
    
    # Post-installation setup
    redis::install::post_install_setup
    
    log::success "${MSG_INSTALL_SUCCESS}"
    redis::status::show
    redis::install::show_next_steps
    
    return 0
}

#######################################
# Check installation prerequisites
# Returns: 0 if all checks pass, 1 if any fail
#######################################
redis::install::check_prerequisites() {
    # Check Docker
    if ! command -v docker >/dev/null 2>&1; then
        log::error "${MSG_ERROR_DOCKER}"
        return 1
    fi
    
    # Check Docker daemon
    if ! docker info >/dev/null 2>&1; then
        log::error "${MSG_ERROR_DOCKER}"
        return 1
    fi
    
    # Check disk space (require at least 1GB free)
    local available_space
    available_space=$(df "${HOME}" | awk 'NR==2 {print $4}')
    local required_space=1048576  # 1GB in KB
    
    if [[ "$available_space" -lt "$required_space" ]]; then
        log::error "Insufficient disk space. Required: 1GB, Available: $(redis::common::format_bytes $((available_space * 1024)))"
        return 1
    fi
    
    # Check memory (warn if less than 2GB available)
    local available_memory
    available_memory=$(free -m | awk 'NR==2{print $7}')
    if [[ "$available_memory" -lt 2048 ]]; then
        log::warn "Low available memory: ${available_memory}MB. Redis may perform poorly."
    fi
    
    return 0
}

#######################################
# Post-installation setup
#######################################
redis::install::post_install_setup() {
    # Create backup directory in project root
    local redis_config_dir
    redis_config_dir="$(secrets::get_project_config_dir)/redis"
    mkdir -p "${redis_config_dir}/backups"
    
    # Set up log rotation (basic)
    redis::install::setup_log_rotation
    
    # Create basic CLI helper script
    redis::install::create_cli_helper
    
    # Update resource configuration
    redis::install::update_resource_config
}

#######################################
# Setup log rotation for Redis logs
#######################################
redis::install::setup_log_rotation() {
    # Log rotation is handled by Docker's logging driver
    # No need for custom logrotate configuration with volumes
    log::debug "Log rotation handled by Docker"
    return 0
}

#######################################
# Create CLI helper script
#######################################
redis::install::create_cli_helper() {
    local redis_config_dir
    redis_config_dir="$(secrets::get_project_config_dir)/redis"
    local cli_script="${redis_config_dir}/redis-cli"
    
    cat > "$cli_script" << 'EOF'
#!/bin/bash
# Redis CLI Helper for Vrooli Resource
# This script connects to the Redis resource instance

# Use the same project root detection method as the secrets library
_detect_project_root() {
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
    
    # Fallback
    echo "/home/matthalloran8/Vrooli"
}

PROJECT_ROOT="$(_detect_project_root)"
source "$PROJECT_ROOT/scripts/lib/service/secrets.sh" 2>/dev/null || true

REDIS_PORT="${REDIS_PORT:-6380}"
REDIS_PASSWORD="$(secrets::resolve "REDIS_PASSWORD" 2>/dev/null || echo "")"

if [[ -n "$REDIS_PASSWORD" ]]; then
    docker exec -it vrooli-redis-resource redis-cli -p 6379 -a "$REDIS_PASSWORD" "$@"
else
    docker exec -it vrooli-redis-resource redis-cli -p 6379 "$@"
fi
EOF
    
    chmod +x "$cli_script"
    log::debug "CLI helper created: $cli_script"
}

#######################################
# Update resource configuration
#######################################
redis::install::update_resource_config() {
    local config_file
    config_file="$(secrets::get_project_config_file)"
    
    if [[ -f "$config_file" ]]; then
        # Use jq to update configuration if available
        if command -v jq >/dev/null 2>&1; then
            local temp_config
            temp_config=$(mktemp)
            
            jq --arg port "$REDIS_PORT" \
               --arg url "redis://localhost:$REDIS_PORT" \
               '.services.storage.redis = {
                   "enabled": true,
                   "baseUrl": $url,
                   "port": ($port | tonumber),
                   "instances": {
                       "default": {
                           "port": ($port | tonumber),
                           "baseUrl": $url,
                           "maxMemory": "'"$REDIS_MAX_MEMORY"'",
                           "persistence": "'"$REDIS_PERSISTENCE"'",
                           "password": null
                       }
                   }
               }' "$config_file" > "$temp_config" && mv "$temp_config" "$config_file"
            
            log::debug "Updated resource configuration"
        else
            log::debug "jq not available, skipping configuration update"
        fi
    fi
}

#######################################
# Show next steps after installation
#######################################
redis::install::show_next_steps() {
    echo
    log::info "🎉 Redis resource is ready! Next steps:"
    echo
    echo "   📋 Basic usage:"
    echo "      redis-cli -p ${REDIS_PORT}                    # Connect to Redis"
    echo "      redis-cli -p ${REDIS_PORT} SET key value      # Set a key"
    echo "      redis-cli -p ${REDIS_PORT} GET key            # Get a key"
    echo
    echo "   🔧 Management:"
    echo "      $0 --action status                           # Check status"
    echo "      $0 --action monitor                          # Real-time monitoring"
    echo "      $0 --action backup                           # Create backup"
    echo
    echo "   📚 Integration:"
    echo "      Connection URL: redis://localhost:${REDIS_PORT}"
    echo "      Use database 0-$((REDIS_DATABASES - 1)) for different applications"
    echo
    if [[ -z "$REDIS_PASSWORD" ]]; then
        echo "   ⚠️  Security: No password is set. Consider setting REDIS_PASSWORD for production use."
        echo
    fi
}

#######################################
# Uninstall Redis resource
# Arguments:
#   $1 - remove data (yes/no)
# Returns: 0 on success, 1 on failure
#######################################
redis::install::uninstall() {
    local remove_data="${1:-no}"
    
    log::info "🗑️  Uninstalling Redis resource..."
    
    # Remove container
    if ! redis::docker::remove_container "$remove_data"; then
        return 1
    fi
    
    # Remove CLI helper from project config directory
    local redis_config_dir
    redis_config_dir="$(secrets::get_project_config_dir)/redis"
    rm -f "${redis_config_dir}/redis-cli"
    
    # Remove configuration if data is being removed
    if [[ "$remove_data" == "yes" ]]; then
        rm -f "${redis_config_dir}/logrotate.conf"
        
        # Update resource configuration
        local config_file
        config_file="$(secrets::get_project_config_file)"
        if [[ -f "$config_file" ]] && command -v jq >/dev/null 2>&1; then
            local temp_config
            temp_config=$(mktemp)
            jq 'del(.services.storage.redis)' "$config_file" > "$temp_config" && mv "$temp_config" "$config_file"
        fi
    fi
    
    log::success "✅ Redis resource uninstalled"
    return 0
}

#######################################
# Upgrade Redis resource
# Returns: 0 on success, 1 on failure
#######################################
redis::install::upgrade() {
    log::info "⬆️  Upgrading Redis resource..."
    
    if ! redis::common::container_exists; then
        log::error "Redis is not installed"
        return 1
    fi
    
    # Create backup before upgrade
    log::info "Creating backup before upgrade..."
    local backup_name="pre-upgrade-$(date +%Y%m%d-%H%M%S)"
    if ! redis::backup::create "$backup_name"; then
        log::warn "Backup failed, but continuing with upgrade..."
    fi
    
    # Pull latest image
    if ! redis::docker::pull_image; then
        return 1
    fi
    
    # Stop current container
    redis::docker::stop
    
    # Remove old container (keep data)
    docker rm "${REDIS_CONTAINER_NAME}" >/dev/null 2>&1
    
    # Create new container with same configuration
    if ! redis::docker::create_container; then
        log::error "Failed to create upgraded container"
        return 1
    fi
    
    # Wait for Redis to be ready
    if ! redis::common::wait_for_ready; then
        log::error "Upgraded Redis failed to start properly"
        return 1
    fi
    
    log::success "✅ Redis resource upgraded successfully"
    redis::status::show
    
    return 0
}

#######################################
# Cleanup after failed installation
#######################################
redis::install::cleanup_failed_install() {
    log::debug "Cleaning up failed installation..."
    
    # Remove container if it exists
    if redis::common::container_exists; then
        docker rm -f "${REDIS_CONTAINER_NAME}" >/dev/null 2>&1
    fi
    
    # Remove partial directories
    if [[ -d "${REDIS_DATA_DIR}" ]]; then
        local file_count
        file_count=$(find "${REDIS_DATA_DIR}" -type f 2>/dev/null | wc -l)
        if [[ "$file_count" -eq 0 ]]; then
            rmdir "${REDIS_DATA_DIR}" 2>/dev/null
        fi
    fi
}