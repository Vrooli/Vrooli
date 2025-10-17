#!/usr/bin/env bash
# BTCPay Lightning Network Operations

# Lightning Network configuration and management functions
btcpay::lightning::setup() {
    log::header "Setting up Lightning Network for BTCPay"
    
    # Check if BTCPay is running
    if ! btcpay::is_running; then
        log::error "BTCPay Server must be running to configure Lightning"
        return 1
    fi
    
    # Check if Lightning is already configured
    if btcpay::lightning::is_configured; then
        log::info "Lightning Network is already configured"
        return 0
    fi
    
    log::info "Starting LND (Lightning Network Daemon) container..."
    
    # Ensure data directories exist
    mkdir -p "${BTCPAY_DATA_DIR}/lnd"
    mkdir -p "${BTCPAY_DATA_DIR}/lnd/data"
    mkdir -p "${BTCPAY_DATA_DIR}/lnd/logs"
    
    # Start LND container
    docker run -d \
        --name "${BTCPAY_CONTAINER_NAME}-lnd" \
        --network "${BTCPAY_NETWORK}" \
        -v "${BTCPAY_DATA_DIR}/lnd/data:/root/.lnd" \
        -v "${BTCPAY_DATA_DIR}/lnd/logs:/logs" \
        -e LND_CHAIN="bitcoin" \
        -e LND_ENVIRONMENT="mainnet" \
        -e LND_RESTLISTEN="0.0.0.0:8080" \
        -e LND_RPCLISTEN="0.0.0.0:10009" \
        -e LND_BITCOIN_NODE="neutrino" \
        -e LND_EXTERNALIP="" \
        -e LND_ALIAS="BTCPay-Lightning" \
        --restart unless-stopped \
        lightninglabs/lnd:v0.17.0-beta
    
    # Wait for LND to start
    log::info "Waiting for Lightning Network Daemon to initialize..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if docker exec "${BTCPAY_CONTAINER_NAME}-lnd" lncli --network=mainnet getinfo &>/dev/null; then
            log::success "Lightning Network Daemon started successfully"
            break
        fi
        sleep 2
        ((attempt++))
    done
    
    if [ $attempt -ge $max_attempts ]; then
        log::warning "LND is taking longer than expected to start. It may still be syncing."
        log::info "You can check status with: resource-btcpay lightning status"
    fi
    
    # Configure BTCPay to use LND
    btcpay::lightning::configure_btcpay
    
    log::success "Lightning Network setup complete"
    return 0
}

btcpay::lightning::configure_btcpay() {
    log::info "Configuring BTCPay to use Lightning Network..."
    
    # Create Lightning configuration
    cat > "${BTCPAY_CONFIG_DIR}/lightning.json" <<EOF
{
    "enabled": true,
    "implementation": "lnd",
    "connection": {
        "type": "grpc",
        "host": "${BTCPAY_CONTAINER_NAME}-lnd",
        "port": 10009,
        "macaroonPath": "/root/.lnd/data/chain/bitcoin/mainnet/admin.macaroon",
        "tlsCertPath": "/root/.lnd/tls.cert"
    }
}
EOF
    
    log::success "Lightning configuration saved"
    
    # Restart BTCPay to apply configuration
    log::info "Restarting BTCPay Server to apply Lightning configuration..."
    docker restart "${BTCPAY_CONTAINER_NAME}"
    
    # Wait for BTCPay to come back online
    sleep 10
    
    if btcpay::is_running; then
        log::success "BTCPay Server restarted with Lightning support"
    else
        log::error "BTCPay Server failed to restart"
        return 1
    fi
}

btcpay::lightning::is_configured() {
    # Check if LND container exists and is running
    if docker ps --filter "name=${BTCPAY_CONTAINER_NAME}-lnd" --format "{{.Names}}" | grep -q "${BTCPAY_CONTAINER_NAME}-lnd"; then
        return 0
    fi
    return 1
}

btcpay::lightning::status() {
    log::header "Lightning Network Status"
    
    if ! btcpay::lightning::is_configured; then
        log::warning "Lightning Network is not configured"
        log::info "Run 'resource-btcpay lightning setup' to configure"
        return 1
    fi
    
    # Check LND container status
    local lnd_status=$(docker ps --filter "name=${BTCPAY_CONTAINER_NAME}-lnd" --format "{{.Status}}" | head -1)
    if [[ -n "$lnd_status" ]]; then
        log::info "LND Container: Running ($lnd_status)"
    else
        log::error "LND Container: Not running"
        return 1
    fi
    
    # Try to get LND info
    if docker exec "${BTCPAY_CONTAINER_NAME}-lnd" lncli --network=mainnet getinfo &>/dev/null; then
        local node_info=$(docker exec "${BTCPAY_CONTAINER_NAME}-lnd" lncli --network=mainnet getinfo 2>/dev/null)
        if [[ -n "$node_info" ]]; then
            log::success "Lightning Node is operational"
            echo "$node_info" | jq -r '. | "Node ID: \(.identity_pubkey)\nSynced: \(.synced_to_chain)\nActive Channels: \(.num_active_channels)\nPeers: \(.num_peers)"' 2>/dev/null || echo "$node_info"
        fi
    else
        log::warning "Lightning Node is initializing or requires wallet setup"
        log::info "This is normal for first-time setup. The node needs to sync with the blockchain."
    fi
    
    # Check BTCPay Lightning configuration
    if [[ -f "${BTCPAY_CONFIG_DIR}/lightning.json" ]]; then
        log::info "Lightning configuration found in BTCPay"
        jq -r '.enabled' "${BTCPAY_CONFIG_DIR}/lightning.json" | xargs -I {} echo "Lightning Enabled: {}"
    fi
    
    return 0
}

btcpay::lightning::create_invoice() {
    local amount="${1:-}"
    local memo="${2:-Lightning Payment}"
    
    if [[ -z "$amount" ]]; then
        log::error "Usage: btcpay::lightning::create_invoice <amount_sats> [memo]"
        return 1
    fi
    
    if ! btcpay::lightning::is_configured; then
        log::error "Lightning Network is not configured"
        log::info "Run 'resource-btcpay lightning setup' first"
        return 1
    fi
    
    log::info "Creating Lightning invoice for ${amount} satoshis..."
    
    # Create invoice using LND
    local invoice_response=$(docker exec "${BTCPAY_CONTAINER_NAME}-lnd" \
        lncli --network=mainnet addinvoice \
        --amt="${amount}" \
        --memo="${memo}" 2>/dev/null)
    
    if [[ -z "$invoice_response" ]]; then
        log::error "Failed to create Lightning invoice"
        log::info "Make sure the Lightning wallet is initialized and unlocked"
        return 1
    fi
    
    # Extract payment request (bolt11 invoice)
    local payment_request=$(echo "$invoice_response" | jq -r '.payment_request // empty')
    local r_hash=$(echo "$invoice_response" | jq -r '.r_hash // empty')
    
    if [[ -n "$payment_request" ]]; then
        log::success "Lightning invoice created successfully"
        echo "Payment Request (BOLT11): ${payment_request}"
        echo "Invoice Hash: ${r_hash}"
        echo ""
        echo "Share the payment request with the payer."
        echo "Check status with: resource-btcpay lightning check-invoice --hash ${r_hash}"
    else
        log::error "Could not extract payment request from response"
        echo "$invoice_response"
        return 1
    fi
}

btcpay::lightning::check_invoice() {
    local r_hash="${1:-}"
    
    if [[ -z "$r_hash" ]]; then
        log::error "Usage: btcpay::lightning::check_invoice <invoice_hash>"
        return 1
    fi
    
    if ! btcpay::lightning::is_configured; then
        log::error "Lightning Network is not configured"
        return 1
    fi
    
    log::info "Checking Lightning invoice status..."
    
    # Lookup invoice using LND
    local invoice_info=$(docker exec "${BTCPAY_CONTAINER_NAME}-lnd" \
        lncli --network=mainnet lookupinvoice "${r_hash}" 2>/dev/null)
    
    if [[ -z "$invoice_info" ]]; then
        log::error "Failed to lookup invoice"
        return 1
    fi
    
    # Parse invoice status
    local settled=$(echo "$invoice_info" | jq -r '.settled // false')
    local state=$(echo "$invoice_info" | jq -r '.state // "UNKNOWN"')
    local value=$(echo "$invoice_info" | jq -r '.value // 0')
    local amt_paid=$(echo "$invoice_info" | jq -r '.amt_paid_sat // 0')
    
    echo "Invoice Status:"
    echo "  State: ${state}"
    echo "  Settled: ${settled}"
    echo "  Amount: ${value} satoshis"
    if [[ "$settled" == "true" ]]; then
        echo "  Amount Paid: ${amt_paid} satoshis"
        log::success "Invoice has been paid!"
    else
        log::info "Invoice is still pending payment"
    fi
}

btcpay::lightning::open_channel() {
    local peer_pubkey="${1:-}"
    local amount="${2:-}"
    
    if [[ -z "$peer_pubkey" || -z "$amount" ]]; then
        log::error "Usage: btcpay::lightning::open_channel <peer_pubkey> <amount_sats>"
        echo ""
        echo "Example: btcpay::lightning::open_channel 03abc...def 100000"
        echo ""
        echo "Popular Lightning nodes to connect to:"
        echo "  ACINQ: 03864ef025fde8fb587d989186ce6a4a186895ee44a926bfc370e2c366597a3f8f"
        echo "  Bitrefill: 030c3f19d742ca294a55c00376b3b355c3c90d61c6b6b39554dbc7ac19b141c14f"
        return 1
    fi
    
    if ! btcpay::lightning::is_configured; then
        log::error "Lightning Network is not configured"
        return 1
    fi
    
    log::info "Opening Lightning channel with ${peer_pubkey}..."
    log::info "Channel size: ${amount} satoshis"
    
    # First, try to connect to the peer
    log::info "Connecting to peer..."
    docker exec "${BTCPAY_CONTAINER_NAME}-lnd" \
        lncli --network=mainnet connect "${peer_pubkey}" 2>/dev/null || true
    
    # Open channel
    local channel_response=$(docker exec "${BTCPAY_CONTAINER_NAME}-lnd" \
        lncli --network=mainnet openchannel \
        --node_key="${peer_pubkey}" \
        --local_amt="${amount}" 2>/dev/null)
    
    if [[ -n "$channel_response" ]]; then
        log::success "Channel opening initiated"
        echo "$channel_response"
        log::info "Channel will be active after blockchain confirmation"
    else
        log::error "Failed to open channel"
        log::info "Make sure you have sufficient balance and the peer is online"
        return 1
    fi
}

btcpay::lightning::list_channels() {
    if ! btcpay::lightning::is_configured; then
        log::error "Lightning Network is not configured"
        return 1
    fi
    
    log::header "Lightning Channels"
    
    # List channels
    local channels=$(docker exec "${BTCPAY_CONTAINER_NAME}-lnd" \
        lncli --network=mainnet listchannels 2>/dev/null)
    
    if [[ -z "$channels" ]]; then
        log::warning "No channels found or LND not ready"
        return 1
    fi
    
    # Parse and display channel info
    echo "$channels" | jq -r '.channels[] | "Channel ID: \(.chan_id)\nPeer: \(.remote_pubkey[0:16])...\nCapacity: \(.capacity) sats\nLocal Balance: \(.local_balance) sats\nRemote Balance: \(.remote_balance) sats\nActive: \(.active)\n---"' 2>/dev/null || echo "$channels"
}

btcpay::lightning::balance() {
    if ! btcpay::lightning::is_configured; then
        log::error "Lightning Network is not configured"
        return 1
    fi
    
    log::header "Lightning Wallet Balance"
    
    # Get wallet balance
    local wallet_balance=$(docker exec "${BTCPAY_CONTAINER_NAME}-lnd" \
        lncli --network=mainnet walletbalance 2>/dev/null)
    
    if [[ -n "$wallet_balance" ]]; then
        echo "$wallet_balance" | jq -r '"On-chain Balance:\n  Confirmed: \(.confirmed_balance) sats\n  Unconfirmed: \(.unconfirmed_balance) sats\n  Total: \(.total_balance) sats"' 2>/dev/null || echo "$wallet_balance"
    fi
    
    # Get channel balance
    local channel_balance=$(docker exec "${BTCPAY_CONTAINER_NAME}-lnd" \
        lncli --network=mainnet channelbalance 2>/dev/null)
    
    if [[ -n "$channel_balance" ]]; then
        echo ""
        echo "$channel_balance" | jq -r '"Lightning Balance:\n  Local: \(.local_balance.sat) sats\n  Remote: \(.remote_balance.sat) sats\n  Pending Open: \(.pending_open_local_balance.sat) sats"' 2>/dev/null || echo "$channel_balance"
    fi
}

# Export all functions
export -f btcpay::lightning::setup
export -f btcpay::lightning::configure_btcpay
export -f btcpay::lightning::is_configured
export -f btcpay::lightning::status
export -f btcpay::lightning::create_invoice
export -f btcpay::lightning::check_invoice
export -f btcpay::lightning::open_channel
export -f btcpay::lightning::list_channels
export -f btcpay::lightning::balance