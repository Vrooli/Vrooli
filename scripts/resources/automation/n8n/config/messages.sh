#!/usr/bin/env bash
# n8n User Messages and Help Text
# All user-facing messages, prompts, and documentation

#######################################
# Display usage information
#######################################
n8n::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install n8n with basic auth"
    echo "  $0 --action install --basic-auth no              # Install without authentication"
    echo "  $0 --action install --username admin --password secret  # Custom credentials"
    echo "  $0 --action install --database postgres          # Use PostgreSQL instead of SQLite"
    echo "  $0 --action install --tunnel yes                 # Enable webhook tunnel (dev only)"
    echo "  $0 --action install --build-image yes            # Build custom image with host access"
    echo "  $0 --action status                               # Check n8n status"
    echo "  $0 --action logs                                 # View n8n logs"
    echo "  $0 --action reset-password                       # Reset admin password"
    echo "  $0 --action uninstall                           # Remove n8n"
}

#######################################
# API setup instructions
#######################################
n8n::show_api_setup_instructions() {
    local auth_user="${N8N_BASIC_AUTH_USER:-admin}"
    local auth_pass
    
    # Try to get password from running container
    if docker ps --format "{{.Names}}" | grep -q "^${N8N_CONTAINER_NAME}$"; then
        auth_pass=$(docker exec "$N8N_CONTAINER_NAME" env | grep N8N_BASIC_AUTH_PASSWORD | cut -d'=' -f2)
    fi
    
    cat << EOF
=== n8n API Setup Instructions ===

The n8n CLI execute command has a known bug in versions 1.93.0+.
You need to use the REST API instead. Here's how:

1. Access n8n Web Interface:
   URL: $N8N_BASE_URL
   Username: $auth_user
   Password: ${auth_pass:-[check container logs or env]}

2. Create API Key:
   - Go to Settings → n8n API
   - Click "Create an API key"
   - Set a label (e.g., "CLI Access")
   - Set expiration (optional)
   - Copy the API key (shown only once!)

3. Save API Key (Choose One):
   Option A - Save to configuration file (recommended):
     $0 --action save-api-key --api-key YOUR_API_KEY
   
   Option B - Set environment variable (temporary):
     export N8N_API_KEY="your-api-key-here"

4. Execute Workflows:
   $0 --action execute --workflow-id YOUR_WORKFLOW_ID

Alternative: Direct API Call:
   curl -X POST -H "X-N8N-API-KEY: \$N8N_API_KEY" \\
        -H "Content-Type: application/json" \\
        ${N8N_BASE_URL}/api/v1/workflows/WORKFLOW_ID/execute

Note: This is a workaround for GitHub issue #15567.
The standard CLI command 'n8n execute --id' is broken in current versions.
EOF
}

#######################################
# Success messages
#######################################
n8n::show_success_message() {
    local auth_user="$1"
    local auth_pass="$2"
    local webhook_url="$3"
    
    echo "✓ n8n has been successfully installed!"
    echo
    echo "Access n8n at: $N8N_BASE_URL"
    
    if [[ "$N8N_ENABLE_BASIC_AUTH" == "yes" ]]; then
        echo
        echo "Login credentials:"
        echo "  Username: $auth_user"
        echo "  Password: $auth_pass"
        echo
        echo "IMPORTANT: Save these credentials! The password cannot be recovered."
    fi
    
    if [[ -n "$webhook_url" ]]; then
        echo
        echo "Webhook URL (for testing only): $webhook_url"
        echo "Note: This URL is temporary and only works while the tunnel is active."
    fi
    
    echo
    echo "To check status: $0 --action status"
    echo "To view logs: $0 --action logs"
    echo "To execute workflows: $0 --action api-setup"
}

#######################################
# Error messages
#######################################
n8n::show_docker_error() {
    echo "Error: Docker is not installed or not running."
    echo
    echo "Please install Docker first:"
    echo "  - Ubuntu/Debian: sudo apt-get install docker.io"
    echo "  - MacOS: Install Docker Desktop from docker.com"
    echo "  - Windows: Install Docker Desktop from docker.com"
    echo
    echo "After installation, make sure Docker is running and try again."
}

n8n::show_permission_error() {
    echo "Error: Cannot access Docker daemon. You may need to:"
    echo "  1. Add your user to the docker group: sudo usermod -aG docker \$USER"
    echo "  2. Log out and back in for changes to take effect"
    echo "  3. Or run with sudo (not recommended)"
}

n8n::show_port_error() {
    local port="$1"
    echo "Error: Port $port is already in use."
    echo
    echo "You can either:"
    echo "  1. Stop the service using port $port"
    echo "  2. Use a different port: N8N_CUSTOM_PORT=5679 $0 --action install"
}

n8n::show_health_check_error() {
    echo "Error: n8n failed to start properly."
    echo
    echo "Please check:"
    echo "  1. Container logs: $0 --action logs"
    echo "  2. Port availability: lsof -i :$N8N_PORT"
    echo "  3. Docker status: docker ps -a"
}

#######################################
# Prompts
#######################################
n8n::prompt_basic_auth() {
    echo
    echo "Basic authentication will be enabled by default."
    echo "This adds a username/password requirement to access n8n."
    echo
    read -p "Enable basic authentication? [Y/n]: " -r
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        echo "no"
    else
        echo "yes"
    fi
}

n8n::prompt_tunnel() {
    echo
    echo "Webhook tunnel allows external services to send data to your local n8n."
    echo "This is useful for testing but should NOT be used in production."
    echo
    read -p "Enable webhook tunnel (development only)? [y/N]: " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "yes"
    else
        echo "no"
    fi
}

n8n::prompt_uninstall_confirmation() {
    echo
    echo "Warning: This will stop and remove the n8n container."
    echo "Your workflows and credentials will be backed up to: $N8N_BACKUP_DIR"
    echo
    read -p "Are you sure you want to uninstall n8n? [y/N]: " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        return 1
    fi
    return 0
}

n8n::prompt_password_reset() {
    local current_user="$1"
    echo
    echo "This will reset the password for user: $current_user"
    echo "The n8n container will be restarted with new credentials."
    echo
    read -p "Continue with password reset? [y/N]: " -r
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        return 1
    fi
    return 0
}