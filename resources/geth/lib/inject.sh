#!/usr/bin/env bash
# Geth Injection Functions
# Handles smart contract deployment and interaction

# Get the script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GETH_INJECT_DIR="${APP_ROOT}/resources/geth/lib"

# Source common functions
# shellcheck disable=SC1091
source "${GETH_INJECT_DIR}/common.sh"

# Deploy a smart contract
geth::deploy_contract() {
    local contract_file="$1"
    local from_address="${2:-}"
    
    if [[ ! -f "$contract_file" ]]; then
        echo "[ERROR] Contract file not found: $contract_file"
        return 1
    fi
    
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    echo "[INFO] Deploying contract from $contract_file..."
    
    # Copy contract to container's contracts directory
    local contract_name=$(basename "$contract_file")
    cp "$contract_file" "${GETH_DATA_DIR}/contracts/${contract_name}"
    
    # For development mode, we can use the default account
    if [[ "${GETH_NETWORK}" == "dev" ]] && [[ -z "$from_address" ]]; then
        # Get the coinbase address (default funded account in dev mode)
        local response
        response=$(curl -s -X POST \
            -H "Content-Type: application/json" \
            --data '{"jsonrpc":"2.0","method":"eth_coinbase","params":[],"id":1}' \
            "http://localhost:${GETH_PORT}")
        
        from_address=$(echo "$response" | jq -r '.result' 2>/dev/null)
        
        if [[ -z "$from_address" ]] || [[ "$from_address" == "null" ]]; then
            echo "[ERROR] Could not get default account address"
            return 1
        fi
        
        echo "[INFO] Using dev account: $from_address"
    fi
    
    echo "[SUCCESS] Contract copied to Geth container"
    echo "[INFO] Contract location: /contracts/${contract_name}"
    echo "[INFO] To compile and deploy, use Geth console or web3 tools"
    
    return 0
}

# Execute a script in Geth console
geth::execute_script() {
    local script_file="$1"
    
    if [[ ! -f "$script_file" ]]; then
        echo "[ERROR] Script file not found: $script_file"
        return 1
    fi
    
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    echo "[INFO] Executing script: $script_file"
    
    # Copy script to container
    local script_name=$(basename "$script_file")
    cp "$script_file" "${GETH_DATA_DIR}/scripts/${script_name}"
    
    # Execute script in Geth console
    docker exec "${GETH_CONTAINER_NAME}" geth attach --exec "loadScript('/scripts/${script_name}')" 2>/dev/null
    
    if [[ $? -eq 0 ]]; then
        echo "[SUCCESS] Script executed successfully"
        return 0
    else
        echo "[ERROR] Script execution failed"
        return 1
    fi
}

# Send transaction
geth::send_transaction() {
    local from="$1"
    local to="$2"
    local value="$3"  # in wei
    
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    echo "[INFO] Sending transaction..."
    echo "  From: $from"
    echo "  To: $to"
    echo "  Value: $value wei"
    
    local data='{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{"from":"'$from'","to":"'$to'","value":"'$value'"}],"id":1}'
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data "$data" \
        "http://localhost:${GETH_PORT}")
    
    local tx_hash
    tx_hash=$(echo "$response" | jq -r '.result' 2>/dev/null)
    
    if [[ -n "$tx_hash" ]] && [[ "$tx_hash" != "null" ]]; then
        echo "[SUCCESS] Transaction sent!"
        echo "  TX Hash: $tx_hash"
        return 0
    else
        local error_msg
        error_msg=$(echo "$response" | jq -r '.error.message' 2>/dev/null)
        echo "[ERROR] Transaction failed: ${error_msg:-Unknown error}"
        return 1
    fi
}

# Get account balance
geth::get_balance() {
    local address="$1"
    local block="${2:-latest}"
    
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["'$address'","'$block'"],"id":1}' \
        "http://localhost:${GETH_PORT}")
    
    local balance_hex
    balance_hex=$(echo "$response" | jq -r '.result' 2>/dev/null)
    
    if [[ -n "$balance_hex" ]] && [[ "$balance_hex" != "null" ]]; then
        # Convert hex to decimal (wei)
        local balance_wei
        balance_wei=$(printf "%d\n" "$balance_hex" 2>/dev/null || echo "0")
        
        # Convert wei to ether (1 ether = 10^18 wei)
        local balance_ether
        balance_ether=$(echo "scale=18; $balance_wei / 1000000000000000000" | bc 2>/dev/null || echo "0")
        
        echo "Balance for $address:"
        echo "  Wei: $balance_wei"
        echo "  Ether: $balance_ether"
        return 0
    else
        echo "[ERROR] Could not get balance for $address"
        return 1
    fi
}

# List accounts
geth::list_accounts() {
    if ! geth::is_running; then
        echo "[ERROR] Geth is not running"
        return 1
    fi
    
    local response
    response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        --data '{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}' \
        "http://localhost:${GETH_PORT}")
    
    local accounts
    accounts=$(echo "$response" | jq -r '.result[]' 2>/dev/null)
    
    if [[ -n "$accounts" ]]; then
        echo "Available accounts:"
        echo "$accounts" | while read -r account; do
            echo "  - $account"
        done
        return 0
    else
        echo "[INFO] No accounts found"
        return 0
    fi
}

# Inject data (main function)
geth::inject() {
    local action="${1:-}"
    shift
    
    case "$action" in
        deploy|contract)
            geth::deploy_contract "$@"
            ;;
        script|execute)
            geth::execute_script "$@"
            ;;
        send|transaction)
            geth::send_transaction "$@"
            ;;
        balance)
            geth::get_balance "$@"
            ;;
        accounts|list)
            geth::list_accounts
            ;;
        *)
            echo "Usage: inject {deploy|script|send|balance|accounts} [args...]"
            echo ""
            echo "Commands:"
            echo "  deploy <contract_file> [from_address]  - Deploy a smart contract"
            echo "  script <script_file>                   - Execute a Geth script"
            echo "  send <from> <to> <value_wei>          - Send transaction"
            echo "  balance <address> [block]             - Get account balance"
            echo "  accounts                               - List available accounts"
            return 1
            ;;
    esac
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    geth::inject "$@"
fi