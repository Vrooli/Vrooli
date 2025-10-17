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

btcpay::core::create_invoice() {
    local store_id="${1:-}"
    local amount="${2:-}"
    local currency="${3:-BTC}"
    local description="${4:-Payment}"
    
    if [[ -z "$store_id" || -z "$amount" ]]; then
        log::error "Usage: btcpay::core::create_invoice <store_id> <amount> [currency] [description]"
        return 1
    fi
    
    # Check if BTCPay is running
    if ! btcpay::is_running; then
        log::error "BTCPay Server is not running"
        return 1
    fi
    
    # Check for API key
    local api_key=""
    if [[ -f "${BTCPAY_CONFIG_DIR}/api-keys.json" ]]; then
        api_key=$(jq -r '.default // empty' "${BTCPAY_CONFIG_DIR}/api-keys.json")
    fi
    
    if [[ -z "$api_key" ]]; then
        log::warning "No API key configured. Invoice creation may fail."
        log::info "Generate an API key through BTCPay web interface -> Account -> API Keys"
    fi
    
    # Create invoice request
    local invoice_data=$(jq -n \
        --arg amount "$amount" \
        --arg currency "$currency" \
        --arg desc "$description" \
        '{
            amount: ($amount | tonumber),
            currency: $currency,
            metadata: {
                orderId: (now | tostring),
                itemDesc: $desc
            }
        }')
    
    log::info "Creating invoice for $amount $currency..."
    
    # Make API request
    local response
    if [[ -n "$api_key" ]]; then
        response=$(timeout 5 curl -sf -X POST \
            -H "Authorization: token $api_key" \
            -H "Content-Type: application/json" \
            -d "$invoice_data" \
            "${BTCPAY_BASE_URL}/api/v1/stores/${store_id}/invoices" 2>&1)
    else
        response=$(timeout 5 curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "$invoice_data" \
            "${BTCPAY_BASE_URL}/api/v1/stores/${store_id}/invoices" 2>&1)
    fi
    
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        echo "$response" | jq '.'
        log::success "Invoice created successfully"
        return 0
    else
        log::error "Failed to create invoice: $response"
        return 1
    fi
}

btcpay::core::get_payment_status() {
    local store_id="${1:-}"
    local invoice_id="${2:-}"
    
    if [[ -z "$store_id" || -z "$invoice_id" ]]; then
        log::error "Usage: btcpay::core::get_payment_status <store_id> <invoice_id>"
        return 1
    fi
    
    # Check if BTCPay is running
    if ! btcpay::is_running; then
        log::error "BTCPay Server is not running"
        return 1
    fi
    
    # Check for API key
    local api_key=""
    if [[ -f "${BTCPAY_CONFIG_DIR}/api-keys.json" ]]; then
        api_key=$(jq -r '.default // empty' "${BTCPAY_CONFIG_DIR}/api-keys.json")
    fi
    
    log::info "Checking payment status for invoice $invoice_id..."
    
    # Make API request
    local response
    if [[ -n "$api_key" ]]; then
        response=$(timeout 5 curl -sf \
            -H "Authorization: token $api_key" \
            "${BTCPAY_BASE_URL}/api/v1/stores/${store_id}/invoices/${invoice_id}" 2>&1)
    else
        response=$(timeout 5 curl -sf \
            "${BTCPAY_BASE_URL}/api/v1/stores/${store_id}/invoices/${invoice_id}" 2>&1)
    fi
    
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        echo "$response" | jq '.'
        local status=$(echo "$response" | jq -r '.status // "unknown"')
        log::success "Invoice status: $status"
        return 0
    else
        log::error "Failed to get payment status: $response"
        return 1
    fi
}

# CLI wrapper functions
btcpay::create_invoice() {
    # Parse CLI arguments for invoice creation
    local store_id=""
    local amount=""
    local currency="BTC"
    local description="Payment"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --store-id|--store)
                store_id="$2"
                shift 2
                ;;
            --amount)
                amount="$2"
                shift 2
                ;;
            --currency)
                currency="$2"
                shift 2
                ;;
            --description|--desc)
                description="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: resource-btcpay content create-invoice --store-id <id> --amount <amount> [--currency <BTC>] [--description <text>]"
                echo ""
                echo "Options:"
                echo "  --store-id, --store <id>     Store ID (required)"
                echo "  --amount <amount>            Amount to charge (required)"
                echo "  --currency <code>            Currency code (default: BTC)"
                echo "  --description <text>         Invoice description (default: Payment)"
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$store_id" || -z "$amount" ]]; then
        log::error "Missing required parameters"
        echo "Usage: resource-btcpay content create-invoice --store-id <id> --amount <amount>"
        return 1
    fi
    
    btcpay::core::create_invoice "$store_id" "$amount" "$currency" "$description"
}

btcpay::check_payment() {
    # Parse CLI arguments for payment status check
    local store_id=""
    local invoice_id=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --store-id|--store)
                store_id="$2"
                shift 2
                ;;
            --invoice-id|--invoice)
                invoice_id="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: resource-btcpay content check-payment --store-id <id> --invoice-id <id>"
                echo ""
                echo "Options:"
                echo "  --store-id, --store <id>       Store ID (required)"
                echo "  --invoice-id, --invoice <id>   Invoice ID (required)"
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$store_id" || -z "$invoice_id" ]]; then
        log::error "Missing required parameters"
        echo "Usage: resource-btcpay content check-payment --store-id <id> --invoice-id <id>"
        return 1
    fi
    
    btcpay::core::get_payment_status "$store_id" "$invoice_id"
}

btcpay::generate_address() {
    log::info "Generate address functionality is available through the BTCPay web interface"
    log::info "Access at: ${BTCPAY_BASE_URL}"
    log::info "This feature requires wallet configuration in BTCPay Server"
}

btcpay::core::create_store() {
    local store_name="${1:-}"
    local default_currency="${2:-BTC}"
    
    if [[ -z "$store_name" ]]; then
        log::error "Usage: btcpay::core::create_store <store_name> [default_currency]"
        return 1
    fi
    
    # Check if BTCPay is running
    if ! btcpay::is_running; then
        log::error "BTCPay Server is not running"
        return 1
    fi
    
    # Check for API key
    local api_key=""
    if [[ -f "${BTCPAY_CONFIG_DIR}/api-keys.json" ]]; then
        api_key=$(jq -r '.default // empty' "${BTCPAY_CONFIG_DIR}/api-keys.json")
    fi
    
    if [[ -z "$api_key" ]]; then
        log::error "API key required for store creation"
        log::info "Generate an API key through BTCPay web interface -> Account -> API Keys"
        return 1
    fi
    
    # Create store request
    local store_data=$(jq -n \
        --arg name "$store_name" \
        --arg currency "$default_currency" \
        '{
            name: $name,
            defaultCurrency: $currency,
            networkFeeMode: "Always",
            defaultPaymentMethod: "BTC",
            derivationScheme: null
        }')
    
    log::info "Creating store: $store_name..."
    
    # Make API request
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Authorization: token $api_key" \
        -H "Content-Type: application/json" \
        -d "$store_data" \
        "${BTCPAY_BASE_URL}/api/v1/stores" 2>&1)
    
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        echo "$response" | jq '.'
        local store_id=$(echo "$response" | jq -r '.id // empty')
        if [[ -n "$store_id" ]]; then
            log::success "Store created successfully with ID: $store_id"
            # Save store ID for future reference
            echo "$store_id" >> "${BTCPAY_CONFIG_DIR}/stores.txt"
        fi
        return 0
    else
        log::error "Failed to create store: $response"
        return 1
    fi
}

btcpay::core::list_stores() {
    # Check if BTCPay is running
    if ! btcpay::is_running; then
        log::error "BTCPay Server is not running"
        return 1
    fi
    
    # Check for API key
    local api_key=""
    if [[ -f "${BTCPAY_CONFIG_DIR}/api-keys.json" ]]; then
        api_key=$(jq -r '.default // empty' "${BTCPAY_CONFIG_DIR}/api-keys.json")
    fi
    
    if [[ -z "$api_key" ]]; then
        log::error "API key required to list stores"
        log::info "Generate an API key through BTCPay web interface -> Account -> API Keys"
        return 1
    fi
    
    log::info "Fetching stores..."
    
    # Make API request
    local response
    response=$(timeout 5 curl -sf \
        -H "Authorization: token $api_key" \
        "${BTCPAY_BASE_URL}/api/v1/stores" 2>&1)
    
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        echo "$response" | jq '.'
        local store_count=$(echo "$response" | jq '. | length')
        log::success "Found $store_count store(s)"
        return 0
    else
        log::error "Failed to list stores: $response"
        return 1
    fi
}

btcpay::core::configure_webhook() {
    local store_id="${1:-}"
    local webhook_url="${2:-}"
    local events="${3:-InvoiceCreated,InvoiceReceivedPayment,InvoiceProcessing,InvoiceExpired,InvoiceSettled,InvoiceInvalid}"
    
    if [[ -z "$store_id" || -z "$webhook_url" ]]; then
        log::error "Usage: btcpay::core::configure_webhook <store_id> <webhook_url> [events]"
        return 1
    fi
    
    # Check if BTCPay is running
    if ! btcpay::is_running; then
        log::error "BTCPay Server is not running"
        return 1
    fi
    
    # Check for API key
    local api_key=""
    if [[ -f "${BTCPAY_CONFIG_DIR}/api-keys.json" ]]; then
        api_key=$(jq -r '.default // empty' "${BTCPAY_CONFIG_DIR}/api-keys.json")
    fi
    
    if [[ -z "$api_key" ]]; then
        log::error "API key required for webhook configuration"
        log::info "Generate an API key through BTCPay web interface -> Account -> API Keys"
        return 1
    fi
    
    # Create webhook request
    local webhook_data=$(jq -n \
        --arg url "$webhook_url" \
        --arg events "$events" \
        '{
            url: $url,
            secret: null,
            enabled: true,
            automaticRedelivery: true,
            authorizedEvents: {
                everything: false,
                specificEvents: ($events | split(","))
            }
        }')
    
    log::info "Configuring webhook for store $store_id..."
    
    # Make API request
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Authorization: token $api_key" \
        -H "Content-Type: application/json" \
        -d "$webhook_data" \
        "${BTCPAY_BASE_URL}/api/v1/stores/${store_id}/webhooks" 2>&1)
    
    local exit_code=$?
    
    if [[ $exit_code -eq 0 ]]; then
        echo "$response" | jq '.'
        local webhook_id=$(echo "$response" | jq -r '.id // empty')
        if [[ -n "$webhook_id" ]]; then
            log::success "Webhook configured successfully with ID: $webhook_id"
        fi
        return 0
    else
        log::error "Failed to configure webhook: $response"
        return 1
    fi
}

# CLI wrapper functions for store management
btcpay::create_store() {
    local store_name=""
    local currency="BTC"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                store_name="$2"
                shift 2
                ;;
            --currency)
                currency="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: resource-btcpay content create-store --name <name> [--currency <BTC>]"
                echo ""
                echo "Options:"
                echo "  --name <name>        Store name (required)"
                echo "  --currency <code>    Default currency (default: BTC)"
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$store_name" ]]; then
        log::error "Missing required parameter: --name"
        echo "Usage: resource-btcpay content create-store --name <name>"
        return 1
    fi
    
    btcpay::core::create_store "$store_name" "$currency"
}

btcpay::list_stores() {
    btcpay::core::list_stores
}

btcpay::configure_webhook() {
    local store_id=""
    local webhook_url=""
    local events=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --store-id|--store)
                store_id="$2"
                shift 2
                ;;
            --url)
                webhook_url="$2"
                shift 2
                ;;
            --events)
                events="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: resource-btcpay content configure-webhook --store-id <id> --url <url> [--events <events>]"
                echo ""
                echo "Options:"
                echo "  --store-id <id>     Store ID (required)"
                echo "  --url <url>         Webhook URL (required)"
                echo "  --events <events>   Comma-separated event names (optional)"
                echo ""
                echo "Default events: InvoiceCreated,InvoiceReceivedPayment,InvoiceProcessing,InvoiceExpired,InvoiceSettled,InvoiceInvalid"
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$store_id" || -z "$webhook_url" ]]; then
        log::error "Missing required parameters"
        echo "Usage: resource-btcpay content configure-webhook --store-id <id> --url <url>"
        return 1
    fi
    
    if [[ -z "$events" ]]; then
        btcpay::core::configure_webhook "$store_id" "$webhook_url"
    else
        btcpay::core::configure_webhook "$store_id" "$webhook_url" "$events"
    fi
}

btcpay::generate_address() {
    log::info "Generate address functionality is available through the BTCPay web interface"
    log::info "Access at: ${BTCPAY_BASE_URL}"
    log::info "This feature requires wallet configuration in BTCPay Server"
}

export -f btcpay::core::credentials
export -f btcpay::core::validate_config
export -f btcpay::core::create_invoice
export -f btcpay::core::get_payment_status
export -f btcpay::core::create_store
export -f btcpay::core::list_stores
export -f btcpay::core::configure_webhook
export -f btcpay::create_invoice
export -f btcpay::check_payment
export -f btcpay::create_store
export -f btcpay::list_stores
export -f btcpay::configure_webhook
export -f btcpay::generate_address