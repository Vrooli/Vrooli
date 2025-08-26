#!/usr/bin/env bash
# BTCPay Core Operations

btcpay::core::credentials() {
    log::info "BTCPay Server Credentials"
    log::info "========================="
    
    if btcpay::is_installed; then
        echo "Web Interface: ${BTCPAY_BASE_URL}"
        echo "API Endpoint: ${BTCPAY_BASE_URL}/api/v1"
        echo ""
        echo "Database:"
        echo "  Host: ${BTCPAY_POSTGRES_CONTAINER}"
        echo "  Port: 5432"
        echo "  Database: ${BTCPAY_POSTGRES_DB}"
        echo "  Username: ${BTCPAY_POSTGRES_USER}"
        echo "  Password: [configured in container]"
        echo ""
        
        # Check for API keys
        if [[ -f "${BTCPAY_CONFIG_DIR}/api-keys.json" ]]; then
            echo "API Keys: Configured (see ${BTCPAY_CONFIG_DIR}/api-keys.json)"
        else
            echo "API Keys: Not configured"
            echo "  To generate: Use BTCPay web interface -> Account -> API Keys"
        fi
        
        echo ""
        echo "Configuration Directory: ${BTCPAY_CONFIG_DIR}"
        echo "Data Directory: ${BTCPAY_DATA_DIR}"
    else
        log::warning "BTCPay Server is not installed"
    fi
}

btcpay::core::validate_config() {
    local config_file="${1:-${BTCPAY_CONFIG_DIR}/settings.json}"
    
    if [[ ! -f "$config_file" ]]; then
        log::error "Configuration file not found: $config_file"
        return 1
    fi
    
    log::info "Validating BTCPay configuration..."
    
    # Basic JSON validation
    if jq empty "$config_file" 2>/dev/null; then
        log::success "Configuration is valid JSON"
    else
        log::error "Configuration is not valid JSON"
        return 1
    fi
    
    # Check required fields
    local network=$(jq -r '.Network // empty' "$config_file")
    if [[ -z "$network" ]]; then
        log::warning "Network not specified, will use default (Mainnet)"
    else
        log::info "Network: $network"
    fi
    
    log::success "Configuration validation complete"
    return 0
}

export -f btcpay::core::credentials
export -f btcpay::core::validate_config