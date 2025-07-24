#!/usr/bin/env bash
set -euo pipefail

# Puppeteer Browser Automation Service Setup and Management
# This script handles installation, configuration, and management of Puppeteer using Docker

DESCRIPTION="Install and manage Puppeteer browser automation service using Docker"

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${SCRIPT_DIR}/.."

# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/../helpers/utils/args.sh"

# Puppeteer configuration
readonly PUPPETEER_PORT="${PUPPETEER_CUSTOM_PORT:-$(resources::get_default_port "puppeteer")}"
readonly PUPPETEER_BASE_URL="http://localhost:${PUPPETEER_PORT}"
readonly PUPPETEER_CONTAINER_NAME="puppeteer"
readonly PUPPETEER_DATA_DIR="${HOME}/.puppeteer"
readonly PUPPETEER_IMAGE="ghcr.io/browserless/chromium:latest"

#######################################
# Parse command line arguments
#######################################
puppeteer::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|logs|info" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if Puppeteer appears to be already installed/running" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "max-browsers" \
        --desc "Maximum concurrent browser instances" \
        --type "value" \
        --default "5"
    
    args::register \
        --name "headless" \
        --desc "Run browsers in headless mode" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    args::register \
        --name "timeout" \
        --desc "Browser timeout in milliseconds" \
        --type "value" \
        --default "30000"
    
    if args::is_asking_for_help "$@"; then
        puppeteer::usage
        exit 0
    fi
    
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export MAX_BROWSERS=$(args::get "max-browsers")
    export HEADLESS=$(args::get "headless")
    export TIMEOUT=$(args::get "timeout")
}

#######################################
# Display usage information
#######################################
puppeteer::usage() {
    args::usage "$DESCRIPTION"
    echo
    echo "Examples:"
    echo "  $0 --action install                              # Install Puppeteer with default settings"
    echo "  $0 --action install --max-browsers 10           # Install with 10 max browsers"
    echo "  $0 --action install --headless no                # Install with headed browsers"
    echo "  $0 --action status                               # Check Puppeteer status"
    echo "  $0 --action logs                                 # View Puppeteer logs"
    echo "  $0 --action uninstall                           # Remove Puppeteer"
}

#######################################
# Check if Docker is installed
# Returns: 0 if installed, 1 otherwise
#######################################
puppeteer::check_docker() {
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
# Check if Puppeteer container exists
# Returns: 0 if exists, 1 otherwise
#######################################
puppeteer::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${PUPPETEER_CONTAINER_NAME}$"
}

#######################################
# Check if Puppeteer is running
# Returns: 0 if running, 1 otherwise
#######################################
puppeteer::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${PUPPETEER_CONTAINER_NAME}$"
}

#######################################
# Check if Puppeteer API is responsive
# Returns: 0 if responsive, 1 otherwise
#######################################
puppeteer::is_healthy() {
    if system::is_command "curl"; then
        # Try multiple times as the service takes time to initialize
        local attempts=0
        while [ $attempts -lt 5 ]; do
            # Browserless uses /pressure endpoint instead of /health
            if curl -f -s --max-time 5 "$PUPPETEER_BASE_URL/pressure" >/dev/null 2>&1; then
                return 0
            fi
            attempts=$((attempts + 1))
            sleep 2
        done
    fi
    return 1
}

#######################################
# Create Puppeteer data directory
#######################################
puppeteer::create_directories() {
    log::info "Creating Puppeteer data directory..."
    
    mkdir -p "$PUPPETEER_DATA_DIR" || {
        log::error "Failed to create Puppeteer data directory"
        return 1
    }
    
    # Add rollback action
    resources::add_rollback_action \
        "Remove Puppeteer data directory" \
        "rm -rf $PUPPETEER_DATA_DIR 2>/dev/null || true" \
        10
    
    log::success "Puppeteer directories created"
    return 0
}

#######################################
# Create Docker network for Puppeteer
#######################################
puppeteer::create_network() {
    local network_name="puppeteer-network"
    
    if ! docker network ls | grep -q "$network_name"; then
        log::info "Creating Docker network for Puppeteer..."
        
        if docker network create "$network_name" >/dev/null 2>&1; then
            log::success "Docker network created"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove Docker network" \
                "docker network rm $network_name 2>/dev/null || true" \
                5
        else
            log::warn "Failed to create Docker network (may already exist)"
        fi
    fi
}

#######################################
# Build Puppeteer Docker command
#######################################
puppeteer::build_docker_command() {
    local docker_cmd="docker run -d"
    docker_cmd+=" --name $PUPPETEER_CONTAINER_NAME"
    docker_cmd+=" --network puppeteer-network"
    docker_cmd+=" -p ${PUPPETEER_PORT}:3000"
    docker_cmd+=" -v ${PUPPETEER_DATA_DIR}:/workspace"
    docker_cmd+=" --restart unless-stopped"
    docker_cmd+=" --shm-size=2gb"
    
    # Environment variables (using new variable names)
    docker_cmd+=" -e CONCURRENT=$MAX_BROWSERS"
    docker_cmd+=" -e TIMEOUT=$TIMEOUT"
    docker_cmd+=" -e ENABLE_DEBUGGER=false"
    
    # Note: DEFAULT_BLOCK_ADS, DEFAULT_STEALTH, KEEP_ALIVE, and DEFAULT_HEADLESS 
    # are deprecated. Headless mode is controlled via launch args in the API calls.
    
    # Security settings
    docker_cmd+=" --cap-add=SYS_ADMIN"
    docker_cmd+=" --security-opt seccomp=unconfined"
    
    # Image
    docker_cmd+=" $PUPPETEER_IMAGE"
    
    echo "$docker_cmd"
}

#######################################
# Start Puppeteer container
#######################################
puppeteer::start_container() {
    log::info "Starting Puppeteer container..."
    
    # Build and execute Docker command
    local docker_cmd
    docker_cmd=$(puppeteer::build_docker_command)
    
    if eval "$docker_cmd" >/dev/null 2>&1; then
        log::success "Puppeteer container started"
        
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove Puppeteer container" \
            "docker stop $PUPPETEER_CONTAINER_NAME 2>/dev/null; docker rm $PUPPETEER_CONTAINER_NAME 2>/dev/null || true" \
            25
        
        return 0
    else
        log::error "Failed to start Puppeteer container"
        return 1
    fi
}

#######################################
# Update Vrooli configuration
#######################################
puppeteer::update_config() {
    # Create JSON with proper escaping
    local additional_config
    additional_config=$(cat <<EOF
{
    "features": {
        "screenshots": true,
        "pdf": true,
        "scraping": true,
        "automation": true
    },
    "browser": {
        "maxConcurrency": "$MAX_BROWSERS",
        "headless": $([[ "$HEADLESS" == "yes" ]] && echo "true" || echo "false"),
        "timeout": "$TIMEOUT"
    },
    "api": {
        "version": "v1",
        "statusEndpoint": "/pressure",
        "screenshotEndpoint": "/screenshot",
        "pdfEndpoint": "/pdf",
        "contentEndpoint": "/content",
        "functionEndpoint": "/function",
        "scrapeEndpoint": "/scrape"
    },
    "container": {
        "name": "$PUPPETEER_CONTAINER_NAME",
        "image": "$PUPPETEER_IMAGE"
    }
}
EOF
)
    
    resources::update_config "agents" "puppeteer" "$PUPPETEER_BASE_URL" "$additional_config"
}

#######################################
# Complete Puppeteer installation
#######################################
puppeteer::install() {
    log::header "🎭 Installing Puppeteer Browser Automation (Docker)"
    
    # Start rollback context
    resources::start_rollback_context "install_puppeteer_docker"
    
    # Check if already installed
    if puppeteer::container_exists && puppeteer::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Puppeteer is already installed and running"
        log::info "Use --force yes to reinstall"
        return 0
    fi
    
    # Check Docker
    if ! puppeteer::check_docker; then
        return 1
    fi
    
    # Validate port assignment
    if ! resources::validate_port "puppeteer" "$PUPPETEER_PORT"; then
        log::error "Port validation failed for Puppeteer"
        log::info "You can set a custom port with: export PUPPETEER_CUSTOM_PORT=<port>"
        return 1
    fi
    
    # Create directories
    if ! puppeteer::create_directories; then
        resources::handle_error \
            "Failed to create Puppeteer directories" \
            "system" \
            "Check directory permissions"
        return 1
    fi
    
    # Create Docker network
    puppeteer::create_network
    
    # Start Puppeteer container
    if ! puppeteer::start_container; then
        resources::handle_error \
            "Failed to start Puppeteer container" \
            "system" \
            "Check Docker logs: docker logs $PUPPETEER_CONTAINER_NAME"
        return 1
    fi
    
    # Wait for service to be ready
    log::info "Waiting for Puppeteer to start..."
    
    # Wait for container to be running and port to be available
    local wait_time=0
    local max_wait=60
    while [ $wait_time -lt $max_wait ]; do
        if puppeteer::is_running && ss -tlnp 2>/dev/null | grep -q ":$PUPPETEER_PORT"; then
            log::info "Container is running and port is bound"
            break
        fi
        sleep 2
        wait_time=$((wait_time + 2))
        echo -n "."
    done
    echo
    
    if [ $wait_time -ge $max_wait ]; then
        resources::handle_error \
            "Puppeteer failed to start within timeout" \
            "system" \
            "Check container logs for errors"
        return 1
    fi
    
    # Give Puppeteer time to initialize
    log::info "Waiting for Puppeteer to complete initialization..."
    sleep 10
    
    if puppeteer::is_healthy; then
        log::success "✅ Puppeteer is running and healthy on port $PUPPETEER_PORT"
        
        # Display access information
        echo
        log::header "🌐 Puppeteer Access Information"
        log::info "URL: $PUPPETEER_BASE_URL"
        log::info "Status Check: $PUPPETEER_BASE_URL/pressure"
        log::info "Max Browsers: $MAX_BROWSERS"
        log::info "Headless Mode: $HEADLESS"
        log::info "Timeout: ${TIMEOUT}ms"
        
        # Update Vrooli configuration
        if ! puppeteer::update_config; then
            log::warn "Failed to update Vrooli configuration"
            log::info "Puppeteer is installed but may need manual configuration in Vrooli"
        fi
        
        # Clear rollback context on success
        ROLLBACK_ACTIONS=()
        OPERATION_ID=""
        
        echo
        log::header "🎯 Next Steps"
        log::info "1. Access Puppeteer at: $PUPPETEER_BASE_URL"
        log::info "2. Use the API endpoints for browser automation"
        log::info "3. Check the docs: https://www.browserless.io/docs/"
        
        return 0
    else
        log::warn "Puppeteer started but health check failed"
        log::info "Check logs: docker logs $PUPPETEER_CONTAINER_NAME"
        return 0
    fi
}

#######################################
# Stop Puppeteer
#######################################
puppeteer::stop() {
    if ! puppeteer::is_running; then
        log::info "Puppeteer is not running"
        return 0
    fi
    
    log::info "Stopping Puppeteer..."
    
    if docker stop "$PUPPETEER_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "Puppeteer stopped"
    else
        log::error "Failed to stop Puppeteer"
        return 1
    fi
}

#######################################
# Start Puppeteer
#######################################
puppeteer::start() {
    if puppeteer::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "Puppeteer is already running on port $PUPPETEER_PORT"
        return 0
    fi
    
    log::info "Starting Puppeteer..."
    
    # Check if container exists
    if ! puppeteer::container_exists; then
        log::error "Puppeteer container does not exist. Run install first."
        return 1
    fi
    
    # Start Puppeteer container
    if docker start "$PUPPETEER_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "Puppeteer started"
        
        # Wait for service to be ready
        if resources::wait_for_service "Puppeteer" "$PUPPETEER_PORT" 30; then
            log::success "✅ Puppeteer is running on port $PUPPETEER_PORT"
            log::info "Access Puppeteer at: $PUPPETEER_BASE_URL"
        else
            log::warn "Puppeteer started but may not be fully ready yet"
        fi
    else
        log::error "Failed to start Puppeteer"
        return 1
    fi
}

#######################################
# Restart Puppeteer
#######################################
puppeteer::restart() {
    log::info "Restarting Puppeteer..."
    puppeteer::stop
    sleep 2
    puppeteer::start
}

#######################################
# Show Puppeteer logs
#######################################
puppeteer::logs() {
    if ! puppeteer::container_exists; then
        log::error "Puppeteer container does not exist"
        return 1
    fi
    
    log::info "Showing Puppeteer logs (Ctrl+C to exit)..."
    docker logs -f "$PUPPETEER_CONTAINER_NAME"
}

#######################################
# Uninstall Puppeteer
#######################################
puppeteer::uninstall() {
    log::header "🗑️  Uninstalling Puppeteer"
    
    if ! flow::is_yes "$YES"; then
        log::warn "This will remove Puppeteer and all browser data"
        read -p "Are you sure you want to continue? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    # Stop and remove Puppeteer container
    if puppeteer::container_exists; then
        log::info "Removing Puppeteer container..."
        docker stop "$PUPPETEER_CONTAINER_NAME" 2>/dev/null || true
        docker rm "$PUPPETEER_CONTAINER_NAME" 2>/dev/null || true
        log::success "Puppeteer container removed"
    fi
    
    # Remove Docker network
    if docker network ls | grep -q "puppeteer-network"; then
        log::info "Removing Docker network..."
        docker network rm "puppeteer-network" 2>/dev/null || true
    fi
    
    # Backup data before removal
    if [[ -d "$PUPPETEER_DATA_DIR" ]]; then
        local backup_dir="$HOME/puppeteer-backup-$(date +%Y%m%d-%H%M%S)"
        log::info "Backing up Puppeteer data to: $backup_dir"
        cp -r "$PUPPETEER_DATA_DIR" "$backup_dir" 2>/dev/null || true
    fi
    
    # Remove data directory
    read -p "Remove Puppeteer data directory? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$PUPPETEER_DATA_DIR" 2>/dev/null || true
        log::info "Data directory removed"
    fi
    
    # Remove from Vrooli config
    resources::remove_config "agents" "puppeteer"
    
    log::success "✅ Puppeteer uninstalled successfully"
}

#######################################
# Show Puppeteer status
#######################################
puppeteer::status() {
    log::header "📊 Puppeteer Status"
    
    # Check Docker
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    # Check container status
    if puppeteer::container_exists; then
        if puppeteer::is_running; then
            log::success "✅ Puppeteer container is running"
            
            # Get container stats
            local stats
            stats=$(docker stats "$PUPPETEER_CONTAINER_NAME" --no-stream --format "CPU: {{.CPUPerc}} | Memory: {{.MemUsage}}" 2>/dev/null || echo "")
            if [[ -n "$stats" ]]; then
                log::info "Resource usage: $stats"
            fi
            
            # Check health
            if puppeteer::is_healthy; then
                log::success "✅ Puppeteer API is healthy"
            else
                log::warn "⚠️  Puppeteer API health check failed"
            fi
            
            # Additional details
            echo
            log::info "Puppeteer Details:"
            log::info "  Web UI: $PUPPETEER_BASE_URL"
            log::info "  Container: $PUPPETEER_CONTAINER_NAME"
            log::info "  Max Browsers: $MAX_BROWSERS"
            log::info "  Headless Mode: $HEADLESS"
            
            # Show logs command
            echo
            log::info "View logs: $0 --action logs"
        else
            log::warn "⚠️  Puppeteer container exists but is not running"
            log::info "Start with: $0 --action start"
        fi
    else
        log::error "❌ Puppeteer is not installed"
        log::info "Install with: $0 --action install"
    fi
}

#######################################
# Show Puppeteer information
#######################################
puppeteer::info() {
    cat << EOF
=== Puppeteer Resource Information ===

ID: puppeteer
Category: agents
Display Name: Puppeteer
Description: Browser automation powered by Chrome/Chromium

Service Details:
- Container Name: $PUPPETEER_CONTAINER_NAME
- Service Port: $PUPPETEER_PORT
- Service URL: $PUPPETEER_BASE_URL
- Docker Image: $PUPPETEER_IMAGE
- Data Directory: $PUPPETEER_DATA_DIR

Endpoints:
- Status Check: $PUPPETEER_BASE_URL/pressure
- Screenshot: POST $PUPPETEER_BASE_URL/screenshot
- PDF: POST $PUPPETEER_BASE_URL/pdf
- Content: POST $PUPPETEER_BASE_URL/content
- Function: POST $PUPPETEER_BASE_URL/function
- Scrape: POST $PUPPETEER_BASE_URL/scrape

Configuration:
- Max Browsers: ${MAX_BROWSERS:-5}
- Headless Mode: ${HEADLESS:-yes}
- Timeout: ${TIMEOUT:-30000}ms

Puppeteer Features:
- High-performance browser automation
- Screenshot generation
- PDF generation
- Web scraping
- JavaScript execution
- Form automation
- Performance monitoring
- Network interception

Example Usage:
# Take a screenshot
curl -X POST $PUPPETEER_BASE_URL/screenshot \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}' \\
  --output screenshot.png

# Generate PDF
curl -X POST $PUPPETEER_BASE_URL/pdf \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}' \\
  --output document.pdf

# Scrape webpage
curl -X POST $PUPPETEER_BASE_URL/content \\
  -H "Content-Type: application/json" \\
  -d '{"url": "https://example.com"}'

For more information, visit: https://www.browserless.io/docs/
EOF
}

#######################################
# Main execution function
#######################################
puppeteer::main() {
    puppeteer::parse_arguments "$@"
    
    case "$ACTION" in
        "install")
            puppeteer::install
            ;;
        "uninstall")
            puppeteer::uninstall
            ;;
        "start")
            puppeteer::start
            ;;
        "stop")
            puppeteer::stop
            ;;
        "restart")
            puppeteer::restart
            ;;
        "status")
            puppeteer::status
            ;;
        "logs")
            puppeteer::logs
            ;;
        "info")
            puppeteer::info
            ;;
        *)
            log::error "Unknown action: $ACTION"
            puppeteer::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    puppeteer::main "$@"
fi