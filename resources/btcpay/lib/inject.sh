#!/bin/bash

# BTCPay Server Injection Functions

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
BTCPAY_INJECT_DIR="${APP_ROOT}/resources/btcpay/lib"

# Source common functions
source "${BTCPAY_INJECT_DIR}/common.sh"

# Main inject function
btcpay::inject() {
    local file="${1:-}"
    local type="${2:-store}"  # store, webhook, invoice, api_key
    
    if [[ -z "${file}" ]]; then
        log::error "No file specified for injection"
        return 1
    fi
    
    if [[ ! -f "${file}" ]]; then
        log::error "File not found: ${file}"
        return 1
    fi
    
    # Check if BTCPay is running
    if ! btcpay::is_running; then
        log::error "BTCPay Server is not running"
        return 1
    fi
    
    case "${type}" in
        store)
            btcpay::inject_store "${file}"
            ;;
        webhook)
            btcpay::inject_webhook "${file}"
            ;;
        invoice)
            btcpay::inject_invoice_template "${file}"
            ;;
        api_key)
            btcpay::inject_api_key "${file}"
            ;;
        *)
            log::error "Unknown injection type: ${type}"
            return 1
            ;;
    esac
}

# Inject store configuration
btcpay::inject_store() {
    local config_file="$1"
    local store_name=$(basename "${config_file}" .json)
    
    log::info "Injecting store configuration: ${store_name}"
    
    # Copy configuration to BTCPay data directory
    cp "${config_file}" "${BTCPAY_CONFIG_DIR}/stores/${store_name}.json"
    
    # TODO: Use BTCPay API to create store when API key is available
    log::warning "Manual configuration required in BTCPay UI"
    log::info "Store configuration saved to: ${BTCPAY_CONFIG_DIR}/stores/${store_name}.json"
    
    return 0
}

# Inject webhook configuration
btcpay::inject_webhook() {
    local webhook_file="$1"
    local webhook_name=$(basename "${webhook_file}" .json)
    
    log::info "Injecting webhook: ${webhook_name}"
    
    # Copy webhook configuration
    cp "${webhook_file}" "${BTCPAY_CONFIG_DIR}/webhooks/${webhook_name}.json"
    
    log::success "Webhook configuration saved"
    return 0
}

# Inject invoice template
btcpay::inject_invoice_template() {
    local template_file="$1"
    local template_name=$(basename "${template_file}" .json)
    
    log::info "Injecting invoice template: ${template_name}"
    
    # Copy template to BTCPay
    cp "${template_file}" "${BTCPAY_CONFIG_DIR}/templates/${template_name}.json"
    
    log::success "Invoice template saved"
    return 0
}

# Inject API key configuration
btcpay::inject_api_key() {
    local key_file="$1"
    
    log::info "Injecting API key configuration"
    
    # Store API key securely
    if command -v resource-vault &>/dev/null; then
        local api_key=$(cat "${key_file}")
        # TODO: Store in Vault when write capability is available
        echo "${api_key}" > "${BTCPAY_CONFIG_DIR}/.api_key"
        chmod 600 "${BTCPAY_CONFIG_DIR}/.api_key"
        log::success "API key stored securely"
    else
        cp "${key_file}" "${BTCPAY_CONFIG_DIR}/.api_key"
        chmod 600 "${BTCPAY_CONFIG_DIR}/.api_key"
        log::warning "API key stored locally (Vault not available)"
    fi
    
    return 0
}

# Export functions
export -f btcpay::inject
export -f btcpay::inject_store
export -f btcpay::inject_webhook
export -f btcpay::inject_invoice_template
export -f btcpay::inject_api_key