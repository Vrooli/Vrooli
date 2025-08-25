#!/usr/bin/env bash
# Geth Installation Functions

# Get the script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
GETH_INSTALL_DIR="${APP_ROOT}/resources/geth/lib"

# Source common functions
# shellcheck disable=SC1091
source "${GETH_INSTALL_DIR}/common.sh"

# Install Geth
geth::install() {
    local network="${1:-${GETH_NETWORK}}"
    
    echo "[INFO] Installing Geth (Ethereum client) ${GETH_VERSION}..."
    
    # Initialize directories
    geth::init_directories
    
    # Pull Docker image
    echo "[INFO] Pulling Geth Docker image..."
    if ! docker pull "${GETH_IMAGE}"; then
        echo "[ERROR] Failed to pull Geth image"
        return 1
    fi
    
    # Create container
    echo "[INFO] Creating Geth container..."
    local docker_cmd="docker create --name ${GETH_CONTAINER_NAME}"
    docker_cmd="$docker_cmd -p ${GETH_PORT}:8545"
    docker_cmd="$docker_cmd -p ${GETH_WS_PORT}:8546"
    docker_cmd="$docker_cmd -p ${GETH_P2P_PORT}:30303"
    docker_cmd="$docker_cmd -p ${GETH_P2P_PORT}:30303/udp"
    docker_cmd="$docker_cmd -v ${GETH_DATA_DIR}/data:/root/.ethereum"
    docker_cmd="$docker_cmd -v ${GETH_DATA_DIR}/contracts:/contracts"
    docker_cmd="$docker_cmd -v ${GETH_DATA_DIR}/scripts:/scripts"
    docker_cmd="$docker_cmd --restart unless-stopped"
    docker_cmd="$docker_cmd ${GETH_IMAGE}"
    
    # Add Geth command based on network
    if [[ "$network" == "dev" ]]; then
        # Development mode with instant mining and prefunded accounts
        docker_cmd="$docker_cmd --dev"
        docker_cmd="$docker_cmd --dev.period 1"
        docker_cmd="$docker_cmd --http"
        docker_cmd="$docker_cmd --http.addr 0.0.0.0"
        docker_cmd="$docker_cmd --http.port 8545"
        docker_cmd="$docker_cmd --http.api eth,net,web3,personal,admin,debug,miner,txpool"
        docker_cmd="$docker_cmd --http.corsdomain '*'"
        docker_cmd="$docker_cmd --http.vhosts '*'"
        docker_cmd="$docker_cmd --ws"
        docker_cmd="$docker_cmd --ws.addr 0.0.0.0"
        docker_cmd="$docker_cmd --ws.port 8546"
        docker_cmd="$docker_cmd --ws.api eth,net,web3,personal,admin,debug,miner,txpool"
        docker_cmd="$docker_cmd --ws.origins '*'"
        docker_cmd="$docker_cmd --networkid ${GETH_CHAIN_ID}"
        docker_cmd="$docker_cmd --allow-insecure-unlock"
    elif [[ "$network" == "mainnet" ]]; then
        docker_cmd="$docker_cmd --syncmode snap"
        docker_cmd="$docker_cmd --http"
        docker_cmd="$docker_cmd --http.addr 0.0.0.0"
        docker_cmd="$docker_cmd --http.port 8545"
        docker_cmd="$docker_cmd --http.api eth,net,web3"
        docker_cmd="$docker_cmd --http.corsdomain 'localhost'"
        docker_cmd="$docker_cmd --ws"
        docker_cmd="$docker_cmd --ws.addr 0.0.0.0"
        docker_cmd="$docker_cmd --ws.port 8546"
        docker_cmd="$docker_cmd --ws.api eth,net,web3"
        docker_cmd="$docker_cmd --ws.origins 'localhost'"
    elif [[ "$network" == "goerli" ]] || [[ "$network" == "sepolia" ]]; then
        docker_cmd="$docker_cmd --${network}"
        docker_cmd="$docker_cmd --syncmode snap"
        docker_cmd="$docker_cmd --http"
        docker_cmd="$docker_cmd --http.addr 0.0.0.0"
        docker_cmd="$docker_cmd --http.port 8545"
        docker_cmd="$docker_cmd --http.api eth,net,web3"
        docker_cmd="$docker_cmd --ws"
        docker_cmd="$docker_cmd --ws.addr 0.0.0.0"
        docker_cmd="$docker_cmd --ws.port 8546"
        docker_cmd="$docker_cmd --ws.api eth,net,web3"
    fi
    
    if eval "$docker_cmd"; then
        echo "[SUCCESS] Geth container created successfully"
        
        # Register CLI
        echo "[INFO] Registering Geth CLI..."
        local cli_path="${GETH_INSTALL_DIR}/../cli.sh"
        local install_cli_script="${GETH_INSTALL_DIR}/../../../lib/resources/install-resource-cli.sh"
        if [[ -f "$cli_path" ]] && [[ -f "$install_cli_script" ]]; then
            bash "$install_cli_script" geth "$cli_path" || true
        fi
        
        # Create example genesis file for private networks
        cat > "${GETH_DATA_DIR}/scripts/genesis.json" << 'EOF'
{
  "config": {
    "chainId": 1337,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0,
    "parisBlock": 0,
    "shanghaiBlock": 0,
    "cancunBlock": 0
  },
  "difficulty": "0x20000",
  "gasLimit": "0x8000000",
  "alloc": {
    "0x0000000000000000000000000000000000000001": {
      "balance": "1000000000000000000000"
    }
  }
}
EOF
        
        return 0
    else
        echo "[ERROR] Failed to create Geth container"
        return 1
    fi
}

# Uninstall Geth
geth::uninstall() {
    echo "[INFO] Uninstalling Geth..."
    
    # Stop container if running
    if geth::is_running; then
        docker stop "${GETH_CONTAINER_NAME}" >/dev/null 2>&1
    fi
    
    # Remove container
    if docker::container_exists "${GETH_CONTAINER_NAME}"; then
        docker rm -f "${GETH_CONTAINER_NAME}" >/dev/null 2>&1
    fi
    
    # Optionally remove data (prompt user)
    if [[ -d "${GETH_DATA_DIR}" ]]; then
        echo "[INFO] Geth data directory exists at ${GETH_DATA_DIR}"
        echo "[INFO] Data directory preserved. Remove manually if needed."
    fi
    
    echo "[SUCCESS] Geth uninstalled"
}

# Main execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-}" in
        install)
            geth::install "${2:-}"
            ;;
        uninstall)
            geth::uninstall
            ;;
        *)
            echo "Usage: $0 {install|uninstall} [network]"
            echo "Networks: dev (default), mainnet, goerli, sepolia"
            exit 1
            ;;
    esac
fi