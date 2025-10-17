#!/usr/bin/env bash
# Pi-hole Core Library - Lifecycle and DNS management functions
set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${RESOURCE_DIR}/config"

# Load configuration
source "${CONFIG_DIR}/defaults.sh"

# Container and service management
CONTAINER_NAME="vrooli-pihole"
PIHOLE_VERSION="${PIHOLE_VERSION:-latest}"
PIHOLE_API_PORT="${PIHOLE_API_PORT:-8087}"

# Check if port 53 is available, use alternative if not
if timeout 1 bash -c "echo > /dev/tcp/127.0.0.1/53" 2>/dev/null; then
    # Port 53 is in use, use alternative port
    PIHOLE_DNS_PORT="${PIHOLE_DNS_PORT:-5353}"
    echo "Note: Port 53 is in use, using alternative DNS port ${PIHOLE_DNS_PORT}"
else
    PIHOLE_DNS_PORT="${PIHOLE_DNS_PORT:-53}"
fi

PIHOLE_DATA_DIR="${HOME}/.vrooli/pihole"

# Install Pi-hole
install_pihole() {
    local force=false
    local skip_validation=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true; shift ;;
            --skip-validation) skip_validation=true; shift ;;
            *) shift ;;
        esac
    done
    
    echo "Installing Pi-hole resource..."
    
    # Check if already installed
    if docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Pi-hole is already installed"
        return 2
    fi
    
    # Create data directory
    mkdir -p "${PIHOLE_DATA_DIR}/etc-pihole"
    mkdir -p "${PIHOLE_DATA_DIR}/etc-dnsmasq.d"
    
    # Generate secure password
    local webpassword
    webpassword=$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-16)
    echo "$webpassword" > "${PIHOLE_DATA_DIR}/.webpassword"
    chmod 600 "${PIHOLE_DATA_DIR}/.webpassword"
    
    # Detect available DNS port
    local dns_port=53
    # Check if port 53 is in use using ss
    if ss -tuln | grep -q ":53 "; then
        # Port 53 is in use, use alternative
        dns_port=5353
        echo "Note: Port 53 is in use by system (likely systemd-resolved)"
        echo "Using alternative DNS port ${dns_port}"
        echo "To test DNS: dig @localhost -p ${dns_port} google.com"
    fi
    
    # Save port configuration
    echo "DNS_PORT=${dns_port}" > "${PIHOLE_DATA_DIR}/.port_config"
    
    # Pull Docker image
    echo "Pulling Pi-hole Docker image..."
    docker pull "pihole/pihole:${PIHOLE_VERSION}"
    
    # Create container
    echo "Creating Pi-hole container..."
    docker create \
        --name "${CONTAINER_NAME}" \
        -p "${dns_port}:53/tcp" \
        -p "${dns_port}:53/udp" \
        -p "${PIHOLE_API_PORT}:80/tcp" \
        -e TZ="$(cat /etc/timezone 2>/dev/null || echo 'UTC')" \
        -e WEBPASSWORD="$webpassword" \
        -e DNSMASQ_LISTENING="all" \
        -e PIHOLE_DNS_="1.1.1.1;1.0.0.1" \
        -e QUERY_LOGGING="true" \
        -v "${PIHOLE_DATA_DIR}/etc-pihole:/etc/pihole" \
        -v "${PIHOLE_DATA_DIR}/etc-dnsmasq.d:/etc/dnsmasq.d" \
        --restart unless-stopped \
        --cap-add NET_ADMIN \
        "pihole/pihole:${PIHOLE_VERSION}"
    
    # Post-installation validation
    if [[ "$skip_validation" != "true" ]]; then
        echo "Validating installation..."
        if docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
            echo "Pi-hole installed successfully"
        else
            echo "Error: Installation validation failed" >&2
            return 1
        fi
    fi
    
    echo "Pi-hole installation complete"
    echo "Web password saved to: ${PIHOLE_DATA_DIR}/.webpassword"
    return 0
}

# Start Pi-hole service
start_pihole() {
    local wait_ready=false
    local timeout=60
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait) wait_ready=true; shift ;;
            --timeout) timeout="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    echo "Starting Pi-hole..."
    
    # Check if container exists
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Error: Pi-hole is not installed. Run 'vrooli resource pihole manage install' first" >&2
        return 1
    fi
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Pi-hole is already running"
        return 2
    fi
    
    # Start container
    docker start "${CONTAINER_NAME}"
    
    # Wait for service to be ready
    if [[ "$wait_ready" == "true" ]]; then
        echo "Waiting for Pi-hole to be ready (timeout: ${timeout}s)..."
        local elapsed=0
        while [[ $elapsed -lt $timeout ]]; do
            if check_health; then
                echo "Pi-hole is ready"
                return 0
            fi
            sleep 2
            elapsed=$((elapsed + 2))
        done
        echo "Error: Pi-hole failed to become ready within ${timeout}s" >&2
        return 1
    fi
    
    echo "Pi-hole started"
    return 0
}

# Stop Pi-hole service
stop_pihole() {
    local force=false
    local timeout=30
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true; shift ;;
            --timeout) timeout="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    echo "Stopping Pi-hole..."
    
    # Check if running
    if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Pi-hole is not running"
        return 2
    fi
    
    # Stop container
    if [[ "$force" == "true" ]]; then
        docker kill "${CONTAINER_NAME}"
    else
        docker stop --time "$timeout" "${CONTAINER_NAME}"
    fi
    
    echo "Pi-hole stopped"
    return 0
}

# Restart Pi-hole service
restart_pihole() {
    echo "Restarting Pi-hole..."
    stop_pihole "$@" || true
    sleep 2
    start_pihole "$@"
}

# Uninstall Pi-hole
uninstall_pihole() {
    local force=false
    local keep_data=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force=true; shift ;;
            --keep-data) keep_data=true; shift ;;
            *) shift ;;
        esac
    done
    
    echo "Uninstalling Pi-hole..."
    
    # Stop if running
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Stopping Pi-hole..."
        docker stop "${CONTAINER_NAME}"
    fi
    
    # Remove container
    if docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Removing container..."
        docker rm "${CONTAINER_NAME}"
    fi
    
    # Remove data if requested
    if [[ "$keep_data" != "true" ]]; then
        echo "Removing data directory..."
        rm -rf "${PIHOLE_DATA_DIR}"
    else
        echo "Keeping data directory: ${PIHOLE_DATA_DIR}"
    fi
    
    echo "Pi-hole uninstalled"
    return 0
}

# Check service health
check_health() {
    # Check if container is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        return 1
    fi
    
    # Load port configuration if exists
    local dns_port="${PIHOLE_DNS_PORT}"
    if [[ -f "${PIHOLE_DATA_DIR}/.port_config" ]]; then
        source "${PIHOLE_DATA_DIR}/.port_config"
        dns_port="${DNS_PORT:-${PIHOLE_DNS_PORT}}"
    fi
    
    # Check DNS service on configured port
    if ! timeout 5 nc -z localhost "${dns_port}" 2>/dev/null; then
        return 1
    fi
    
    # Check API health - just verify web server responds
    if ! timeout 5 curl -If "http://localhost:${PIHOLE_API_PORT}/admin/" 2>/dev/null | grep -q "HTTP/1.1"; then
        return 1
    fi
    
    return 0
}

# Show service status
show_status() {
    local verbose=false
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --verbose) verbose=true; shift ;;
            --json) json_output=true; shift ;;
            *) shift ;;
        esac
    done
    
    # Check container status
    local status="stopped"
    local health="unhealthy"
    local blocked_domains=0
    local queries_today=0
    
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        status="running"
        if check_health; then
            health="healthy"
            
            # Get statistics if available
            local stats
            if stats=$(timeout 5 curl -sf "http://localhost:${PIHOLE_API_PORT}/admin/api.php?summary" 2>/dev/null); then
                blocked_domains=$(echo "$stats" | jq -r '.domains_being_blocked // 0')
                queries_today=$(echo "$stats" | jq -r '.dns_queries_today // 0')
            fi
        fi
    fi
    
    if [[ "$json_output" == "true" ]]; then
        cat <<EOF
{
  "status": "$status",
  "health": "$health",
  "container": "${CONTAINER_NAME}",
  "dns_port": ${PIHOLE_DNS_PORT},
  "api_port": ${PIHOLE_API_PORT},
  "blocked_domains": $blocked_domains,
  "queries_today": $queries_today
}
EOF
    else
        echo "Pi-hole Status"
        echo "=============="
        echo "Status: $status"
        echo "Health: $health"
        echo "Container: ${CONTAINER_NAME}"
        echo "DNS Port: ${PIHOLE_DNS_PORT}"
        echo "API Port: ${PIHOLE_API_PORT}"
        
        if [[ "$status" == "running" && "$health" == "healthy" ]]; then
            echo ""
            echo "Statistics:"
            echo "  Blocked Domains: $blocked_domains"
            echo "  Queries Today: $queries_today"
        fi
        
        if [[ "$verbose" == "true" && "$status" == "running" ]]; then
            echo ""
            echo "Container Details:"
            docker ps --filter "name=${CONTAINER_NAME}" --format "table {{.Status}}\t{{.Ports}}"
        fi
    fi
}

# Show service logs
show_logs() {
    local lines=50
    local follow=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --lines) lines="$2"; shift 2 ;;
            --follow) follow=true; shift ;;
            *) shift ;;
        esac
    done
    
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        echo "Error: Pi-hole container not found" >&2
        return 1
    fi
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f "${CONTAINER_NAME}"
    else
        docker logs --tail "$lines" "${CONTAINER_NAME}"
    fi
}

# Show credentials
show_credentials() {
    local json_output=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json) json_output=true; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ ! -f "${PIHOLE_DATA_DIR}/.webpassword" ]]; then
        echo "Error: Credentials not found. Is Pi-hole installed?" >&2
        return 1
    fi
    
    local password
    password=$(cat "${PIHOLE_DATA_DIR}/.webpassword")
    
    # Get API token from container if running
    local api_token=""
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        api_token=$(docker exec "${CONTAINER_NAME}" cat /etc/pihole/setupVars.conf 2>/dev/null | grep WEBPASSWORD | cut -d= -f2 | sha256sum | cut -d' ' -f1 || echo "")
    fi
    
    if [[ "$json_output" == "true" ]]; then
        cat <<EOF
{
  "web_url": "http://localhost:${PIHOLE_API_PORT}/admin",
  "web_password": "$password",
  "api_token": "$api_token"
}
EOF
    else
        echo "Pi-hole Credentials"
        echo "==================="
        echo "Web URL: http://localhost:${PIHOLE_API_PORT}/admin"
        echo "Web Password: $password"
        if [[ -n "$api_token" ]]; then
            echo "API Token: $api_token"
        fi
    fi
}

# Content management functions

# Show blocking statistics
show_stats() {
    if ! check_health; then
        echo "Error: Pi-hole is not running or unhealthy" >&2
        return 1
    fi
    
    local stats
    if ! stats=$(timeout 5 curl -sf "http://localhost:${PIHOLE_API_PORT}/admin/api.php?summary" 2>/dev/null); then
        echo "Error: Failed to retrieve statistics" >&2
        return 1
    fi
    
    echo "Pi-hole Statistics"
    echo "=================="
    echo "$stats" | jq -r '
        "Domains Blocked: \(.domains_being_blocked)
Queries Today: \(.dns_queries_today)
Queries Blocked: \(.ads_blocked_today)
Percent Blocked: \(.ads_percentage_today)%
Unique Clients: \(.unique_clients)
Status: \(.status)"
    '
}

# Update blocklists
update_blocklists() {
    if ! check_health; then
        echo "Error: Pi-hole is not running or unhealthy" >&2
        return 1
    fi
    
    echo "Updating Pi-hole blocklists..."
    docker exec "${CONTAINER_NAME}" pihole -g
    echo "Blocklists updated successfully"
}

# Manage blacklist
manage_blacklist() {
    local action="${1:-list}"
    local domain="${2:-}"
    
    if ! check_health; then
        echo "Error: Pi-hole is not running or unhealthy" >&2
        return 1
    fi
    
    case "$action" in
        add)
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required for add action" >&2
                return 1
            fi
            docker exec "${CONTAINER_NAME}" pihole -b "$domain"
            echo "Added $domain to blacklist"
            ;;
        remove)
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required for remove action" >&2
                return 1
            fi
            docker exec "${CONTAINER_NAME}" pihole -b -d "$domain"
            echo "Removed $domain from blacklist"
            ;;
        list)
            docker exec "${CONTAINER_NAME}" pihole -b -l
            ;;
        *)
            echo "Error: Unknown blacklist action: $action" >&2
            echo "Valid actions: add, remove, list" >&2
            return 1
            ;;
    esac
}

# Manage whitelist
manage_whitelist() {
    local action="${1:-list}"
    local domain="${2:-}"
    
    if ! check_health; then
        echo "Error: Pi-hole is not running or unhealthy" >&2
        return 1
    fi
    
    case "$action" in
        add)
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required for add action" >&2
                return 1
            fi
            docker exec "${CONTAINER_NAME}" pihole -w "$domain"
            echo "Added $domain to whitelist"
            ;;
        remove)
            if [[ -z "$domain" ]]; then
                echo "Error: Domain required for remove action" >&2
                return 1
            fi
            docker exec "${CONTAINER_NAME}" pihole -w -d "$domain"
            echo "Removed $domain from whitelist"
            ;;
        list)
            docker exec "${CONTAINER_NAME}" pihole -w -l
            ;;
        *)
            echo "Error: Unknown whitelist action: $action" >&2
            echo "Valid actions: add, remove, list" >&2
            return 1
            ;;
    esac
}

# Query DNS logs
query_logs() {
    local tail_lines=100
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --tail) tail_lines="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if ! check_health; then
        echo "Error: Pi-hole is not running or unhealthy" >&2
        return 1
    fi
    
    echo "Recent DNS queries (last $tail_lines):"
    docker exec "${CONTAINER_NAME}" tail -n "$tail_lines" /var/log/pihole/pihole.log
}

# Disable blocking temporarily
disable_blocking() {
    local duration=300  # Default 5 minutes
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --duration) duration="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if ! check_health; then
        echo "Error: Pi-hole is not running or unhealthy" >&2
        return 1
    fi
    
    local api_token
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        api_token=$(docker exec "${CONTAINER_NAME}" cat /etc/pihole/setupVars.conf 2>/dev/null | grep WEBPASSWORD | cut -d= -f2 | sha256sum | cut -d' ' -f1 || echo "")
    fi
    
    if [[ -z "$api_token" ]]; then
        echo "Error: Could not retrieve API token" >&2
        return 1
    fi
    
    echo "Disabling Pi-hole blocking for ${duration} seconds..."
    curl -sf "http://localhost:${PIHOLE_API_PORT}/admin/api.php?disable=${duration}&auth=${api_token}"
    echo "Blocking disabled for ${duration} seconds"
}

# Enable blocking
enable_blocking() {
    if ! check_health; then
        echo "Error: Pi-hole is not running or unhealthy" >&2
        return 1
    fi
    
    local api_token
    if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
        api_token=$(docker exec "${CONTAINER_NAME}" cat /etc/pihole/setupVars.conf 2>/dev/null | grep WEBPASSWORD | cut -d= -f2 | sha256sum | cut -d' ' -f1 || echo "")
    fi
    
    if [[ -z "$api_token" ]]; then
        echo "Error: Could not retrieve API token" >&2
        return 1
    fi
    
    echo "Enabling Pi-hole blocking..."
    curl -sf "http://localhost:${PIHOLE_API_PORT}/admin/api.php?enable&auth=${api_token}"
    echo "Blocking enabled"
}