#!/usr/bin/env bash
# BTCPay Multi-Currency Support

# Supported cryptocurrencies configuration
btcpay::multicurrency::get_supported_chains() {
    # Return supported chains based on configuration
    echo "btc,ltc,eth"
}

btcpay::multicurrency::configure() {
    log::info "Configuring multi-currency support..."
    
    # Update configuration to support multiple chains
    local chains="${1:-btc,ltc}"
    
    # Store configuration
    cat > "${BTCPAY_CONFIG_DIR}/multicurrency.json" <<EOF
{
    "enabled": true,
    "chains": "${chains}",
    "defaultCurrency": "BTC",
    "supportedCurrencies": [
        {
            "code": "BTC",
            "name": "Bitcoin",
            "chain": "btc",
            "enabled": true,
            "confirmations": 1
        },
        {
            "code": "LTC", 
            "name": "Litecoin",
            "chain": "ltc",
            "enabled": true,
            "confirmations": 6
        },
        {
            "code": "ETH",
            "name": "Ethereum",
            "chain": "eth",
            "enabled": false,
            "confirmations": 12
        }
    ]
}
EOF
    
    log::success "Multi-currency configuration saved"
    log::info "Enabled chains: ${chains}"
    log::warning "Container restart required for changes to take effect"
    
    return 0
}

btcpay::multicurrency::enable_currency() {
    local currency="${1:-}"
    
    if [[ -z "$currency" ]]; then
        log::error "Usage: btcpay::multicurrency::enable_currency <currency_code>"
        return 1
    fi
    
    # Check if configuration exists
    if [[ ! -f "${BTCPAY_CONFIG_DIR}/multicurrency.json" ]]; then
        log::warning "Multi-currency not configured, initializing with defaults..."
        btcpay::multicurrency::configure
    fi
    
    # Update configuration to enable the currency
    local config_file="${BTCPAY_CONFIG_DIR}/multicurrency.json"
    
    # Check if currency is supported
    if ! jq -e ".supportedCurrencies[] | select(.code == \"${currency}\")" "$config_file" &>/dev/null; then
        log::error "Currency ${currency} is not supported"
        log::info "Supported currencies: BTC, LTC, ETH"
        return 1
    fi
    
    # Enable the currency
    local tmp_file=$(mktemp)
    jq ".supportedCurrencies |= map(if .code == \"${currency}\" then .enabled = true else . end)" "$config_file" > "$tmp_file"
    mv "$tmp_file" "$config_file"
    
    log::success "Currency ${currency} enabled"
    
    # Update chains list
    local enabled_chains=$(jq -r '.supportedCurrencies[] | select(.enabled == true) | .chain' "$config_file" | paste -sd,)
    jq ".chains = \"${enabled_chains}\"" "$config_file" > "$tmp_file"
    mv "$tmp_file" "$config_file"
    
    log::info "Active chains: ${enabled_chains}"
    log::warning "Restart BTCPay Server to apply changes"
    
    return 0
}

btcpay::multicurrency::disable_currency() {
    local currency="${1:-}"
    
    if [[ -z "$currency" ]]; then
        log::error "Usage: btcpay::multicurrency::disable_currency <currency_code>"
        return 1
    fi
    
    # Don't allow disabling BTC
    if [[ "$currency" == "BTC" ]]; then
        log::error "Cannot disable Bitcoin (BTC) - it's the primary currency"
        return 1
    fi
    
    local config_file="${BTCPAY_CONFIG_DIR}/multicurrency.json"
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Multi-currency configuration not found"
        return 1
    fi
    
    # Disable the currency
    local tmp_file=$(mktemp)
    jq ".supportedCurrencies |= map(if .code == \"${currency}\" then .enabled = false else . end)" "$config_file" > "$tmp_file"
    mv "$tmp_file" "$config_file"
    
    log::success "Currency ${currency} disabled"
    
    # Update chains list
    local enabled_chains=$(jq -r '.supportedCurrencies[] | select(.enabled == true) | .chain' "$config_file" | paste -sd,)
    jq ".chains = \"${enabled_chains}\"" "$config_file" > "$tmp_file"
    mv "$tmp_file" "$config_file"
    
    log::info "Active chains: ${enabled_chains}"
    log::warning "Restart BTCPay Server to apply changes"
    
    return 0
}

btcpay::multicurrency::list() {
    log::info "Multi-Currency Configuration"
    log::info "============================"
    
    local config_file="${BTCPAY_CONFIG_DIR}/multicurrency.json"
    
    if [[ ! -f "$config_file" ]]; then
        log::warning "Multi-currency not configured"
        log::info "Run 'resource-btcpay multicurrency configure' to set up"
        return 0
    fi
    
    # Display enabled status
    local enabled=$(jq -r '.enabled' "$config_file")
    echo "Multi-currency enabled: ${enabled}"
    echo ""
    
    # List currencies
    echo "Supported Currencies:"
    echo "-------------------"
    jq -r '.supportedCurrencies[] | "\(.code) - \(.name): \(if .enabled then "✓ Enabled" else "✗ Disabled" end) (confirmations: \(.confirmations))"' "$config_file"
    
    echo ""
    echo "Active chains: $(jq -r '.chains' "$config_file")"
    echo "Default currency: $(jq -r '.defaultCurrency' "$config_file")"
    
    return 0
}

btcpay::multicurrency::status() {
    log::info "Multi-Currency Status"
    log::info "===================="
    
    # Check if multi-currency is configured
    if [[ ! -f "${BTCPAY_CONFIG_DIR}/multicurrency.json" ]]; then
        log::warning "Multi-currency not configured"
        return 0
    fi
    
    # Check current container configuration
    if btcpay::is_running; then
        local current_chains=$(docker inspect "${BTCPAY_CONTAINER_NAME}" 2>/dev/null | jq -r '.[0].Config.Env[] | select(startswith("BTCPAY_CHAINS="))' | cut -d= -f2)
        echo "Container configured chains: ${current_chains}"
        
        local config_chains=$(jq -r '.chains' "${BTCPAY_CONFIG_DIR}/multicurrency.json")
        echo "Configuration chains: ${config_chains}"
        
        if [[ "$current_chains" != "$config_chains" ]]; then
            log::warning "Configuration mismatch - restart required"
        else
            log::success "Configuration is synchronized"
        fi
    else
        log::info "BTCPay Server is not running"
    fi
    
    return 0
}

btcpay::multicurrency::apply() {
    log::info "Applying multi-currency configuration..."
    
    if [[ ! -f "${BTCPAY_CONFIG_DIR}/multicurrency.json" ]]; then
        log::error "Multi-currency configuration not found"
        return 1
    fi
    
    # Get enabled chains
    local chains=$(jq -r '.chains' "${BTCPAY_CONFIG_DIR}/multicurrency.json")
    
    if [[ -z "$chains" ]]; then
        log::error "No chains enabled"
        return 1
    fi
    
    log::info "Restarting BTCPay Server with chains: ${chains}"
    
    # Stop existing containers
    btcpay::docker::stop
    
    # Remove old containers to force recreation with new config
    docker rm "${BTCPAY_CONTAINER_NAME}" 2>/dev/null || true
    docker rm "${BTCPAY_NBXPLORER_CONTAINER}" 2>/dev/null || true
    
    # Update environment for multi-chain support
    export BTCPAY_CHAINS="${chains}"
    export NBXPLORER_CHAINS="${chains}"
    
    # Restart with new configuration
    btcpay::docker::start
    
    if btcpay::is_running; then
        log::success "Multi-currency configuration applied successfully"
        btcpay::multicurrency::status
        return 0
    else
        log::error "Failed to apply multi-currency configuration"
        return 1
    fi
}

# CLI command handlers
btcpay::multicurrency() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        configure)
            btcpay::multicurrency::configure "$@"
            ;;
        enable)
            btcpay::multicurrency::enable_currency "$@"
            ;;
        disable)
            btcpay::multicurrency::disable_currency "$@"
            ;;
        list)
            btcpay::multicurrency::list
            ;;
        status)
            btcpay::multicurrency::status
            ;;
        apply)
            btcpay::multicurrency::apply
            ;;
        --help|-h|help)
            cat <<EOF
Multi-Currency Management Commands:

  configure [chains]     Configure multi-currency support
  enable <currency>      Enable a specific currency (BTC, LTC, ETH)
  disable <currency>     Disable a specific currency
  list                   List all supported currencies and their status
  status                 Show multi-currency configuration status
  apply                  Apply configuration and restart with new settings

Examples:
  resource-btcpay multicurrency configure btc,ltc
  resource-btcpay multicurrency enable LTC
  resource-btcpay multicurrency list
  resource-btcpay multicurrency apply
EOF
            ;;
        *)
            log::error "Unknown multicurrency subcommand: $subcommand"
            log::info "Use 'resource-btcpay multicurrency --help' for usage"
            return 1
            ;;
    esac
}