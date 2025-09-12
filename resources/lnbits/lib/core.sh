#!/usr/bin/env bash
# LNbits Core Functionality Library

set -euo pipefail

# Source configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/config/defaults.sh"

# Container and service management
LNBITS_CONTAINER="lnbits-server"
LNBITS_POSTGRES_CONTAINER="lnbits-postgres"
LNBITS_NETWORK="lnbits-network"

# Check if LNbits is installed
is_installed() {
    docker image inspect "${LNBITS_IMAGE}" &>/dev/null
}

# Check if LNbits is running
is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${LNBITS_CONTAINER}$"
}

# Check health endpoint
check_health() {
    timeout 5 curl -sf "http://localhost:${LNBITS_PORT}/api/v1/health" &>/dev/null
}

# Get container uptime
get_uptime() {
    if is_running; then
        docker ps --format "table {{.Status}}" --filter "name=${LNBITS_CONTAINER}" | tail -1
    else
        echo "Not running"
    fi
}

# Get or generate PostgreSQL password
get_postgres_password() {
    local password_file="${LNBITS_CONFIG_DIR}/.postgres_password"
    
    if [[ -f "$password_file" ]]; then
        cat "$password_file"
    else
        # Generate new password if file doesn't exist
        local new_password="lnbits_secure_$(openssl rand -hex 16)"
        echo "$new_password" > "$password_file"
        chmod 600 "$password_file"
        echo "$new_password"
    fi
}

# Install LNbits
manage_install() {
    local force=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if is_installed && [[ "$force" != "true" ]]; then
        echo "LNbits is already installed. Use --force to reinstall."
        exit 2
    fi
    
    echo "Installing LNbits..."
    
    # Create necessary directories
    mkdir -p "${LNBITS_DATA_DIR}"
    mkdir -p "${LNBITS_CONFIG_DIR}"
    mkdir -p "${LNBITS_LOGS_DIR}"
    
    # Create Docker network
    docker network create "${LNBITS_NETWORK}" 2>/dev/null || true
    
    # Pull Docker images
    echo "Pulling Docker images..."
    docker pull "${LNBITS_IMAGE}"
    docker pull "${POSTGRES_IMAGE}"
    
    # Start PostgreSQL for LNbits with a fixed password
    if ! docker ps -a --format "{{.Names}}" | grep -q "^${LNBITS_POSTGRES_CONTAINER}$"; then
        echo "Starting PostgreSQL for LNbits..."
        docker run -d \
            --name "${LNBITS_POSTGRES_CONTAINER}" \
            --network "${LNBITS_NETWORK}" \
            -e POSTGRES_USER="${LNBITS_POSTGRES_USER}" \
            -e POSTGRES_PASSWORD="lnbitssecret123" \
            -e POSTGRES_DB="${LNBITS_POSTGRES_DB}" \
            -v "${LNBITS_POSTGRES_DATA}:/var/lib/postgresql/data" \
            "${POSTGRES_IMAGE}"
        
        # Wait for PostgreSQL to be ready
        echo "Waiting for PostgreSQL to be ready..."
        sleep 10
    fi
    
    echo "LNbits installation complete!"
    echo "Run '${RESOURCE_NAME} manage start' to start the service."
}

# Start LNbits service
manage_start() {
    local wait_for_ready=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait)
                wait_for_ready=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if ! is_installed; then
        echo "Error: LNbits is not installed. Run '${RESOURCE_NAME} manage install' first." >&2
        exit 1
    fi
    
    if is_running; then
        echo "LNbits is already running."
        exit 0
    fi
    
    echo "Starting LNbits..."
    
    # Ensure PostgreSQL is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${LNBITS_POSTGRES_CONTAINER}$"; then
        docker start "${LNBITS_POSTGRES_CONTAINER}"
        echo "Waiting for PostgreSQL to be ready..."
        sleep 5
    fi
    
    # Start LNbits container
    docker run -d \
        --name "${LNBITS_CONTAINER}" \
        --network "${LNBITS_NETWORK}" \
        -p "${LNBITS_PORT}:5000" \
        -e LNBITS_ADMIN_UI="${LNBITS_ADMIN_UI}" \
        -e LNBITS_SITE_TITLE="${LNBITS_SITE_TITLE}" \
        -e LNBITS_DATABASE_URL="postgres://lnbits:lnbitssecret123@${LNBITS_POSTGRES_CONTAINER}:5432/lnbits" \
        -e LNBITS_BACKEND_WALLET_CLASS="${LNBITS_BACKEND_WALLET}" \
        -e FAKE_WALLET_SECRET="${FAKE_WALLET_SECRET}" \
        -e LNBITS_DATA_FOLDER="/app/data" \
        -v "${LNBITS_DATA_DIR}:/app/data" \
        -v "${LNBITS_EXTENSIONS_DIR}:/app/lnbits/extensions" \
        "${LNBITS_IMAGE}"
    
    if [[ "$wait_for_ready" == "true" ]]; then
        echo "Waiting for LNbits to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if check_health; then
                echo "LNbits is ready!"
                exit 0
            fi
            sleep 2
            ((attempt++))
        done
        
        echo "Error: LNbits failed to become ready within 60 seconds" >&2
        exit 1
    fi
    
    echo "LNbits started successfully."
}

# Stop LNbits service
manage_stop() {
    if ! is_running; then
        echo "LNbits is not running."
        exit 0
    fi
    
    echo "Stopping LNbits..."
    docker stop "${LNBITS_CONTAINER}" &>/dev/null
    docker rm "${LNBITS_CONTAINER}" &>/dev/null
    echo "LNbits stopped."
}

# Restart LNbits service
manage_restart() {
    echo "Restarting LNbits..."
    manage_stop
    sleep 2
    manage_start "$@"
}

# Uninstall LNbits
manage_uninstall() {
    local force=false
    local keep_data=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                shift
                ;;
            --keep-data)
                keep_data=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if ! is_installed; then
        echo "LNbits is not installed."
        exit 2
    fi
    
    if [[ "$force" != "true" ]]; then
        read -p "Are you sure you want to uninstall LNbits? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Uninstall cancelled."
            exit 0
        fi
    fi
    
    echo "Uninstalling LNbits..."
    
    # Stop and remove containers
    docker stop "${LNBITS_CONTAINER}" &>/dev/null || true
    docker rm "${LNBITS_CONTAINER}" &>/dev/null || true
    docker stop "${LNBITS_POSTGRES_CONTAINER}" &>/dev/null || true
    docker rm "${LNBITS_POSTGRES_CONTAINER}" &>/dev/null || true
    
    # Remove images
    docker rmi "${LNBITS_IMAGE}" &>/dev/null || true
    docker rmi "${POSTGRES_IMAGE}" &>/dev/null || true
    
    # Remove network
    docker network rm "${LNBITS_NETWORK}" &>/dev/null || true
    
    # Remove data if not keeping
    if [[ "$keep_data" != "true" ]]; then
        echo "Removing data directories..."
        rm -rf "${LNBITS_DATA_DIR}"
    else
        echo "Data preserved in ${LNBITS_DATA_DIR}"
    fi
    
    echo "LNbits uninstalled successfully."
}

# Content management functions

# Add wallet or payment
content_add() {
    local type=""
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$type" ]]; then
        echo "Error: --type is required (wallet or payment)" >&2
        exit 1
    fi
    
    if ! is_running; then
        echo "Error: LNbits is not running" >&2
        exit 1
    fi
    
    case "$type" in
        wallet)
            create_wallet "$name"
            ;;
        payment)
            echo "Use 'content execute --action invoice' to create payments" >&2
            exit 1
            ;;
        *)
            echo "Error: Unknown type: $type" >&2
            exit 1
            ;;
    esac
}

# Create a new wallet
create_wallet() {
    local name="${1:-New Wallet}"
    
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${LNBITS_PORT}/api/v1/wallet" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$name\"}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        echo "Error: Failed to create wallet" >&2
        exit 1
    fi
    
    echo "Wallet created successfully:"
    echo "$response" | jq '.'
}

# List wallets or payments
content_list() {
    local type="${1:-wallet}"
    
    if ! is_running; then
        echo "Error: LNbits is not running" >&2
        exit 1
    fi
    
    case "$type" in
        wallet)
            list_wallets
            ;;
        payment)
            echo "Payment listing requires wallet API key" >&2
            exit 1
            ;;
        *)
            echo "Error: Unknown type: $type" >&2
            exit 1
            ;;
    esac
}

# List all wallets
list_wallets() {
    echo "Note: Wallet listing requires admin access."
    echo "Access the web UI at http://localhost:${LNBITS_PORT}/wallet"
}

# Get wallet or payment details
content_get() {
    local id="$1"
    
    if [[ -z "$id" ]]; then
        echo "Error: ID is required" >&2
        exit 1
    fi
    
    if ! is_running; then
        echo "Error: LNbits is not running" >&2
        exit 1
    fi
    
    echo "Getting details requires wallet API key."
    echo "Use the web UI or provide API key."
}

# Remove wallet
content_remove() {
    local id="$1"
    
    if [[ -z "$id" ]]; then
        echo "Error: ID is required" >&2
        exit 1
    fi
    
    echo "Wallet removal requires admin access."
    echo "Use the web UI for wallet management."
}

# Execute payment operations
content_execute() {
    local action=""
    local amount=""
    local memo=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --action)
                action="$2"
                shift 2
                ;;
            --amount)
                amount="$2"
                shift 2
                ;;
            --memo)
                memo="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$action" ]]; then
        echo "Error: --action is required (invoice, pay, balance)" >&2
        exit 1
    fi
    
    if ! is_running; then
        echo "Error: LNbits is not running" >&2
        exit 1
    fi
    
    case "$action" in
        invoice)
            create_invoice "$amount" "$memo"
            ;;
        pay)
            echo "Payment execution requires wallet API key" >&2
            exit 1
            ;;
        balance)
            echo "Balance check requires wallet API key" >&2
            exit 1
            ;;
        *)
            echo "Error: Unknown action: $action" >&2
            exit 1
            ;;
    esac
}

# Create an invoice
create_invoice() {
    local amount="${1:-1000}"
    local memo="${2:-Payment}"
    
    echo "Creating invoice requires wallet API key."
    echo "Example with API key:"
    echo "curl -X POST http://localhost:${LNBITS_PORT}/api/v1/payments \\"
    echo "  -H 'X-Api-Key: YOUR_WALLET_API_KEY' \\"
    echo "  -H 'Content-Type: application/json' \\"
    echo "  -d '{\"out\": false, \"amount\": $amount, \"memo\": \"$memo\"}'"
}