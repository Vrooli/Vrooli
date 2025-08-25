#!/usr/bin/env bash
# Geth Resource CLI

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    GETH_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${GETH_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
GETH_CLI_DIR="${APP_ROOT}/resources/geth"

# Source libraries
source "${GETH_CLI_DIR}/lib/common.sh"
source "${GETH_CLI_DIR}/lib/install.sh"
source "${GETH_CLI_DIR}/lib/start.sh"
source "${GETH_CLI_DIR}/lib/stop.sh"
source "${GETH_CLI_DIR}/lib/status.sh"
source "${GETH_CLI_DIR}/lib/inject.sh"

# Show help
show_help() {
    cat << EOF
Geth Resource CLI

Usage: $(basename "$0") <command> [options]

Commands:
    install [network]      Install Geth (networks: dev, mainnet, goerli, sepolia)
    uninstall             Uninstall Geth
    start                 Start Geth node
    stop                  Stop Geth node
    status [format]       Show Geth status (formats: text, json)
    
    inject <subcommand>   Inject data into Geth:
      deploy <file>       Deploy smart contract
      script <file>       Execute Geth script
      send <from> <to> <wei>  Send transaction
      balance <address>   Get account balance
      accounts           List accounts
    
    accounts             List available accounts
    balance <address>    Get balance for address
    peers                Show connected peers
    sync                 Show sync status
    block                Show current block number
    
    console              Attach to Geth console
    logs [lines]         Show Geth logs
    help                 Show this help message

Examples:
    $(basename "$0") install dev           # Install dev network
    $(basename "$0") start                 # Start Geth
    $(basename "$0") inject deploy contract.sol  # Deploy contract
    $(basename "$0") balance 0x123...      # Check balance
    $(basename "$0") console               # Open Geth console

Environment Variables:
    GETH_NETWORK         Network to use (default: dev)
    GETH_PORT           JSON-RPC port (default: 8545)
    GETH_WS_PORT        WebSocket port (default: 8546)
    GETH_DATA_DIR       Data directory (default: ~/.vrooli/geth)

EOF
}

# Attach to Geth console
geth_console() {
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    echo "[INFO] Attaching to Geth console..."
    echo "[INFO] Type 'exit' to leave the console"
    docker exec -it "${GETH_CONTAINER_NAME}" geth attach
}

# Show logs
geth_logs() {
    local lines="${1:-100}"
    
    if ! docker::container_exists "${GETH_CONTAINER_NAME}"; then
        echo "[ERROR] Geth container does not exist"
        return 1
    fi
    
    docker logs --tail "$lines" -f "${GETH_CONTAINER_NAME}"
}

# Show peers
geth_peers() {
    local count
    count=$(geth::get_peer_count)
    echo "Connected peers: $count"
    
    if [[ "$count" -gt 0 ]]; then
        # Get peer details
        local response
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            --data '{"jsonrpc":"2.0","method":"admin_peers","params":[],"id":1}' \
            "http://localhost:${GETH_PORT}" 2>/dev/null)
        
        if [[ -n "$response" ]]; then
            echo "$response" | jq '.result[]' 2>/dev/null | head -20
        fi
    fi
}

# Show sync status
geth_sync() {
    local status
    status=$(geth::get_sync_status)
    echo "Sync status: $status"
    
    if [[ "$status" == "syncing" ]]; then
        # Get detailed sync info
        local response
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' \
            "http://localhost:${GETH_PORT}" 2>/dev/null)
        
        if [[ -n "$response" ]]; then
            echo "$response" | jq '.result' 2>/dev/null
        fi
    fi
}

# Show block number
geth_block() {
    local block
    block=$(geth::get_block_number)
    echo "Current block: $block"
}

# Main command handler
main() {
    local command="${1:-}"
    shift
    
    case "$command" in
        install)
            geth::install "$@"
            ;;
        uninstall)
            geth::uninstall
            ;;
        start)
            geth::start
            ;;
        stop)
            geth::stop
            ;;
        status)
            geth::status "$@"
            ;;
        inject)
            geth::inject "$@"
            ;;
        accounts)
            geth::list_accounts
            ;;
        balance)
            geth::get_balance "$@"
            ;;
        console)
            geth_console
            ;;
        logs)
            geth_logs "$@"
            ;;
        peers)
            geth_peers
            ;;
        sync)
            geth_sync
            ;;
        block)
            geth_block
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            echo "Error: Unknown command '$command'"
            echo "Run '$(basename "$0") help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
