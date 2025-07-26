#!/usr/bin/env bash
# n8n Password Management Functions
# Authentication and password reset functionality

#######################################
# Reset admin password
#######################################
n8n::reset_password() {
    log::header "🔐 Reset n8n Password"
    
    if ! n8n::container_exists; then
        log::error "n8n is not installed"
        return 1
    fi
    
    # Generate new password
    local new_password
    new_password=$(n8n::generate_password)
    
    log::info "Generating new password..."
    
    # Get current environment variables
    local current_env
    current_env=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null)
    
    # Update password in environment
    local new_env=()
    local username="admin"
    while IFS= read -r env_var; do
        if [[ "$env_var" =~ ^N8N_BASIC_AUTH_PASSWORD= ]]; then
            new_env+=("-e" "N8N_BASIC_AUTH_PASSWORD=$new_password")
        elif [[ "$env_var" =~ ^N8N_BASIC_AUTH_USER= ]]; then
            username=$(echo "$env_var" | cut -d'=' -f2)
            new_env+=("-e" "$env_var")
        elif [[ -n "$env_var" ]]; then
            new_env+=("-e" "$env_var")
        fi
    done <<< "$current_env"
    
    # Ensure basic auth is enabled
    if ! echo "$current_env" | grep -q "N8N_BASIC_AUTH_ACTIVE=true"; then
        new_env+=("-e" "N8N_BASIC_AUTH_ACTIVE=true")
    fi
    
    # Get container configuration
    local image
    image=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{.Config.Image}}')
    
    local volumes
    volumes=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range .Mounts}}{{if eq .Type "bind"}}-v {{.Source}}:{{.Destination}} {{end}}{{end}}')
    
    local ports
    ports=$(docker inspect "$N8N_CONTAINER_NAME" --format='{{range $p, $conf := .NetworkSettings.Ports}}{{if $conf}}-p {{(index $conf 0).HostPort}}:{{$p}} {{end}}{{end}}' | sed 's|/tcp||g')
    
    # Stop and remove old container
    log::info "Stopping n8n container..."
    docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1
    docker rm "$N8N_CONTAINER_NAME" >/dev/null 2>&1
    
    # Start new container with updated password
    log::info "Starting n8n with new password..."
    if docker run -d \
        --name "$N8N_CONTAINER_NAME" \
        --network "$N8N_NETWORK_NAME" \
        $ports \
        $volumes \
        --restart unless-stopped \
        "${new_env[@]}" \
        "$image" >/dev/null 2>&1; then
        
        log::success "Password updated successfully"
        
        # Wait for service to start
        if resources::wait_for_service "n8n" "$N8N_PORT" 30; then
            echo
            log::header "🔑 New n8n Credentials"
            log::info "Username: $username"
            log::warn "Password: $new_password (save this password!)"
            log::info "URL: $N8N_BASE_URL"
        fi
        
        return 0
    else
        log::error "Failed to restart n8n with new password"
        return 1
    fi
}

#######################################
# Get current auth credentials from container
# Returns: Username and password if found
#######################################
n8n::get_current_credentials() {
    if ! n8n::is_running; then
        return 1
    fi
    
    local username=$(n8n::get_container_env "N8N_BASIC_AUTH_USER")
    local password=$(n8n::get_container_env "N8N_BASIC_AUTH_PASSWORD")
    
    if [[ -n "$username" ]] && [[ -n "$password" ]]; then
        echo "Username: $username"
        echo "Password: $password"
        return 0
    else
        return 1
    fi
}

#######################################
# Check if authentication is required
# Returns: 0 if auth required, 1 if not
#######################################
n8n::is_auth_required() {
    if n8n::is_running; then
        local auth_active=$(n8n::get_container_env "N8N_BASIC_AUTH_ACTIVE")
        [[ "$auth_active" == "true" ]]
    else
        # Check if we're planning to enable auth
        [[ "$N8N_ENABLE_BASIC_AUTH" == "yes" ]]
    fi
}

#######################################
# Generate auth configuration for Docker
# Args: username, password
# Returns: Environment variables for auth
#######################################
n8n::generate_auth_config() {
    local username="${1:-$N8N_DEFAULT_USERNAME}"
    local password="$2"
    
    if [[ "$N8N_ENABLE_BASIC_AUTH" == "yes" ]]; then
        echo "-e N8N_BASIC_AUTH_ACTIVE=true"
        echo "-e N8N_BASIC_AUTH_USER=$username"
        echo "-e N8N_BASIC_AUTH_PASSWORD=$password"
    else
        echo "-e N8N_BASIC_AUTH_ACTIVE=false"
    fi
}

#######################################
# Validate password strength
# Args: password
# Returns: 0 if valid, 1 if too weak
#######################################
n8n::validate_password() {
    local password="$1"
    
    # Check minimum length
    if [[ ${#password} -lt 8 ]]; then
        log::warn "Password should be at least 8 characters long"
        return 1
    fi
    
    # Check for at least one uppercase, lowercase, and number
    if [[ ! "$password" =~ [A-Z] ]] || [[ ! "$password" =~ [a-z] ]] || [[ ! "$password" =~ [0-9] ]]; then
        log::warn "Password should contain uppercase, lowercase, and numbers"
        return 1
    fi
    
    return 0
}

#######################################
# Save credentials to secure location
# Args: username, password
# Returns: 0 on success, 1 on failure
#######################################
n8n::save_credentials() {
    local username="$1"
    local password="$2"
    local creds_file="${N8N_DATA_DIR}/.credentials"
    
    # Create credentials file with restricted permissions
    if echo -e "username=$username\npassword=$password" > "$creds_file"; then
        chmod 600 "$creds_file"
        log::success "Credentials saved securely"
        return 0
    else
        log::error "Failed to save credentials"
        return 1
    fi
}

#######################################
# Load saved credentials
# Returns: Username and password if found
#######################################
n8n::load_credentials() {
    local creds_file="${N8N_DATA_DIR}/.credentials"
    
    if [[ -f "$creds_file" ]]; then
        source "$creds_file"
        if [[ -n "$username" ]] && [[ -n "$password" ]]; then
            echo "username=$username"
            echo "password=$password"
            return 0
        fi
    fi
    
    return 1
}