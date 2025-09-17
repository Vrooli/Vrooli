#!/usr/bin/env bash
################################################################################
# ERPNext Accounting Module Functions
# 
# Manages financial accounting operations
################################################################################

set -euo pipefail

# Determine base path for sourcing
ERPNEXT_LIB_DIR="${ERPNEXT_LIB_DIR:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
ERPNEXT_BASE_DIR="${ERPNEXT_BASE_DIR:-$(builtin cd "${ERPNEXT_LIB_DIR}/.." && pwd)}"

# Define log functions if not already available
if ! declare -f log::error &>/dev/null; then
    log::error() { echo "[ERROR] $*" >&2; }
    log::info() { echo "[INFO] $*"; }
fi

# Source dependencies if not already loaded
if [[ -z "${ERPNEXT_PORT:-}" ]]; then
    source "${ERPNEXT_BASE_DIR}/config/defaults.sh"
fi

if [[ -f "${ERPNEXT_BASE_DIR}/lib/api.sh" ]]; then
    source "${ERPNEXT_BASE_DIR}/lib/api.sh"
fi

################################################################################
# Sales Invoice Management
################################################################################

erpnext::accounting::list_invoices() {
    local type="${1:-Sales Invoice}"  # Sales Invoice or Purchase Invoice
    local session_id="${2:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/${type// /%20}" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
invoices = data.get('data', [])
if invoices:
    for inv in invoices:
        print(f\"Invoice: {inv.get('name', 'N/A')} - Customer: {inv.get('customer', inv.get('supplier', 'N/A'))} - Total: {inv.get('grand_total', 0)}\")
else:
    print('No invoices found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve invoices"
        return 1
    fi
}

erpnext::accounting::create_invoice() {
    local customer="${1:-}"
    local amount="${2:-100}"
    local item="${3:-Service}"
    local session_id="${4:-}"
    
    if [[ -z "$customer" ]]; then
        log::error "Customer name is required"
        echo "Usage: resource-erpnext accounting create-invoice <customer> [amount] [item_description]"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    # Get current date in YYYY-MM-DD format
    local posting_date=$(date +%Y-%m-%d)
    
    local data="{
        \"doctype\":\"Sales Invoice\",
        \"customer\":\"${customer}\",
        \"posting_date\":\"${posting_date}\",
        \"due_date\":\"${posting_date}\",
        \"currency\":\"USD\",
        \"items\":[{
            \"item_code\":\"${item}\",
            \"description\":\"${item}\",
            \"qty\":1,
            \"rate\":${amount},
            \"amount\":${amount}
        }]
    }"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Sales%20Invoice" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    invoice = data['data']
    print(f\"✅ Invoice created: {invoice.get('name', 'Unknown')} - Total: {invoice.get('grand_total', 0)}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to create invoice')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to create invoice"
        return 1
    fi
}

################################################################################
# Journal Entry Management
################################################################################

erpnext::accounting::list_journal_entries() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Journal%20Entry" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
entries = data.get('data', [])
if entries:
    for entry in entries:
        print(f\"Journal Entry: {entry.get('name', 'N/A')} - Date: {entry.get('posting_date', 'N/A')} - Total Debit: {entry.get('total_debit', 0)}\")
else:
    print('No journal entries found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve journal entries"
        return 1
    fi
}

################################################################################
# Chart of Accounts Management
################################################################################

erpnext::accounting::list_accounts() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Account?limit=50" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
accounts = data.get('data', [])
if accounts:
    for acc in accounts:
        print(f\"Account: {acc.get('name', 'N/A')} - Type: {acc.get('account_type', 'N/A')} - Root: {acc.get('root_type', 'N/A')}\")
else:
    print('No accounts found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve accounts"
        return 1
    fi
}

################################################################################
# Payment Management
################################################################################

erpnext::accounting::list_payments() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Payment%20Entry" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
payments = data.get('data', [])
if payments:
    for payment in payments:
        print(f\"Payment: {payment.get('name', 'N/A')} - Party: {payment.get('party_name', 'N/A')} - Amount: {payment.get('paid_amount', 0)}\")
else:
    print('No payments found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve payments"
        return 1
    fi
}

erpnext::accounting::create_payment() {
    local party="${1:-}"
    local amount="${2:-100}"
    local type="${3:-Receive}"  # Receive or Pay
    local session_id="${4:-}"
    
    if [[ -z "$party" ]]; then
        log::error "Party name is required"
        echo "Usage: resource-erpnext accounting create-payment <party> [amount] [type]"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    # Get current date in YYYY-MM-DD format
    local posting_date=$(date +%Y-%m-%d)
    
    # Determine party type based on payment type
    local party_type="Customer"
    [[ "$type" == "Pay" ]] && party_type="Supplier"
    
    local data="{
        \"doctype\":\"Payment Entry\",
        \"payment_type\":\"${type}\",
        \"party_type\":\"${party_type}\",
        \"party\":\"${party}\",
        \"posting_date\":\"${posting_date}\",
        \"paid_amount\":${amount},
        \"received_amount\":${amount}
    }"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Payment%20Entry" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    payment = data['data']
    print(f\"✅ Payment created: {payment.get('name', 'Unknown')} - Amount: {payment.get('paid_amount', 0)}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to create payment')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to create payment"
        return 1
    fi
}

################################################################################
# Financial Reports
################################################################################

erpnext::accounting::get_balance_sheet() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    echo "Balance Sheet Report:"
    echo "===================="
    
    # Get accounts by type
    for account_type in "Asset" "Liability" "Equity"; do
        echo ""
        echo "$account_type Accounts:"
        
        local response
        response=$(timeout 5 curl -sf \
            -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
            -H "Cookie: sid=${session_id}" \
            "http://localhost:${ERPNEXT_PORT}/api/resource/Account?filters=[[\"root_type\",\"=\",\"${account_type}\"]]&limit=20" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
accounts = data.get('data', [])
if accounts:
    for acc in accounts:
        print(f\"  - {acc.get('name', 'N/A')}: {acc.get('account_type', 'N/A')}\")
else:
    print('  No accounts found')
" 2>/dev/null || echo "  Error parsing response"
        else
            echo "  Failed to retrieve $account_type accounts"
        fi
    done
}

################################################################################
# Export functions for use by other scripts
################################################################################

export -f erpnext::accounting::list_invoices
export -f erpnext::accounting::create_invoice
export -f erpnext::accounting::list_journal_entries
export -f erpnext::accounting::list_accounts
export -f erpnext::accounting::list_payments
export -f erpnext::accounting::create_payment
export -f erpnext::accounting::get_balance_sheet