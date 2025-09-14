#!/usr/bin/env bash
# BTCPay Content Operations

btcpay::content::add() {
    local file="${1:-}"
    local type="${2:-store}"
    
    if [[ -z "$file" ]] || [[ ! -f "$file" ]]; then
        log::error "Valid file required for content add"
        return 1
    fi
    
    case "$type" in
        store)
            btcpay::inject_store "$file"
            ;;
        webhook)
            btcpay::inject_webhook "$file"
            ;;
        invoice)
            btcpay::inject_invoice_template "$file"
            ;;
        api_key)
            btcpay::inject_api_key "$file"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
}

btcpay::content::list() {
    local type="${1:-all}"
    
    log::info "Listing BTCPay content (type: $type)"
    
    case "$type" in
        stores|all)
            if [[ -d "${BTCPAY_CONFIG_DIR}/stores" ]]; then
                log::info "Stores:"
                ls -la "${BTCPAY_CONFIG_DIR}/stores/" 2>/dev/null || echo "  No stores configured"
            fi
            ;;&
        webhooks|all)
            if [[ -d "${BTCPAY_CONFIG_DIR}/webhooks" ]]; then
                log::info "Webhooks:"
                ls -la "${BTCPAY_CONFIG_DIR}/webhooks/" 2>/dev/null || echo "  No webhooks configured"
            fi
            ;;&
        invoices|all)
            if [[ -d "${BTCPAY_CONFIG_DIR}/invoice-templates" ]]; then
                log::info "Invoice Templates:"
                ls -la "${BTCPAY_CONFIG_DIR}/invoice-templates/" 2>/dev/null || echo "  No invoice templates configured"
            fi
            ;;
    esac
}

btcpay::content::get() {
    local name="${1:-}"
    local type="${2:-store}"
    
    if [[ -z "$name" ]]; then
        log::error "Name required for content get"
        return 1
    fi
    
    local file_path
    case "$type" in
        store)
            file_path="${BTCPAY_CONFIG_DIR}/stores/${name}.json"
            ;;
        webhook)
            file_path="${BTCPAY_CONFIG_DIR}/webhooks/${name}.json"
            ;;
        invoice)
            file_path="${BTCPAY_CONFIG_DIR}/invoice-templates/${name}.json"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    if [[ -f "$file_path" ]]; then
        cat "$file_path"
    else
        log::error "Content not found: $name ($type)"
        return 1
    fi
}

btcpay::content::remove() {
    local name="${1:-}"
    local type="${2:-store}"
    
    if [[ -z "$name" ]]; then
        log::error "Name required for content remove"
        return 1
    fi
    
    local file_path
    case "$type" in
        store)
            file_path="${BTCPAY_CONFIG_DIR}/stores/${name}.json"
            ;;
        webhook)
            file_path="${BTCPAY_CONFIG_DIR}/webhooks/${name}.json"
            ;;
        invoice)
            file_path="${BTCPAY_CONFIG_DIR}/invoice-templates/${name}.json"
            ;;
        *)
            log::error "Unknown content type: $type"
            return 1
            ;;
    esac
    
    if [[ -f "$file_path" ]]; then
        rm -f "$file_path"
        log::success "Removed $type: $name"
    else
        log::warning "Content not found: $name ($type)"
        return 1
    fi
}

btcpay::content::execute() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        create-invoice)
            btcpay::create_invoice "$@"
            ;;
        check-payment)
            btcpay::check_payment "$@"
            ;;
        generate-address)
            btcpay::generate_address "$@"
            ;;
        *)
            log::error "Unknown execute action: $action"
            log::info "Available actions: create-invoice, check-payment, generate-address"
            return 1
            ;;
    esac
}

# Business functionality - delegated to core.sh
# The actual implementations are in core.sh with proper API integration

export -f btcpay::content::add
export -f btcpay::content::list
export -f btcpay::content::get
export -f btcpay::content::remove
export -f btcpay::content::execute
export -f btcpay::create_invoice
export -f btcpay::check_payment
export -f btcpay::generate_address