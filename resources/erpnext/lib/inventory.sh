#!/usr/bin/env bash
################################################################################
# ERPNext Inventory Management Module
#
# Provides comprehensive inventory management capabilities
################################################################################

set -euo pipefail

# Source dependencies
source "${APP_ROOT}/resources/erpnext/lib/api.sh"
source "${APP_ROOT}/resources/erpnext/config/defaults.sh"

################################################################################
# Item Management Functions
################################################################################

erpnext::inventory::list_items() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # List all items (without field filter to avoid timeout)
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Item?limit_page_length=100" 2>/dev/null
}

erpnext::inventory::create_item() {
    local item_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Create new item
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${item_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Item" 2>/dev/null
}

erpnext::inventory::get_item() {
    local item_code="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Get item details
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Item/${item_code}" 2>/dev/null
}

erpnext::inventory::update_item() {
    local item_code="${1}"
    local item_json="${2}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Update item
    timeout 5 curl -sf -X PUT \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${item_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Item/${item_code}" 2>/dev/null
}

################################################################################
# Stock Management Functions
################################################################################

erpnext::inventory::get_stock_balance() {
    local item_code="${1}"
    local warehouse="${2:-}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Get stock balance
    local filters="{\"item_code\":\"${item_code}\""
    if [[ -n "$warehouse" ]]; then
        filters="${filters},\"warehouse\":\"${warehouse}\""
    fi
    filters="${filters}}"

    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Bin?filters=${filters}" 2>/dev/null
}

erpnext::inventory::create_stock_entry() {
    local entry_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Create stock entry
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${entry_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Stock Entry" 2>/dev/null
}

################################################################################
# Warehouse Management Functions
################################################################################

erpnext::inventory::list_warehouses() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # List all warehouses (without field filter to avoid timeout)
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Warehouse?limit_page_length=100" 2>/dev/null
}

erpnext::inventory::create_warehouse() {
    local warehouse_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Create new warehouse
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${warehouse_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Warehouse" 2>/dev/null
}

################################################################################
# Purchase Order Functions
################################################################################

erpnext::inventory::create_purchase_order() {
    local po_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Create purchase order
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${po_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Purchase Order" 2>/dev/null
}

erpnext::inventory::list_purchase_orders() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # List purchase orders (without field filter to avoid timeout)
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Purchase Order?limit_page_length=100" 2>/dev/null
}

################################################################################
# CLI Wrapper Functions
################################################################################

erpnext::inventory::cli::list_items() {
    log::info "Listing inventory items..."

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # List items
    local result
    result=$(erpnext::inventory::list_items "$session_id")

    if [[ -n "$result" ]]; then
        local count=$(echo "$result" | jq '.data | length' 2>/dev/null || echo "0")
        if [[ "$count" -eq "0" ]]; then
            log::info "No items found. Creating sample items..."
            # Create sample items
            local item1='{"doctype":"Item","item_code":"LAPTOP-001","item_name":"Laptop Computer","item_group":"Products","stock_uom":"Unit","is_stock_item":1,"valuation_rate":1000}'
            local item2='{"doctype":"Item","item_code":"MOUSE-001","item_name":"Wireless Mouse","item_group":"Products","stock_uom":"Unit","is_stock_item":1,"valuation_rate":25}'

            erpnext::inventory::create_item "$item1" "$session_id" &>/dev/null
            erpnext::inventory::create_item "$item2" "$session_id" &>/dev/null

            result=$(erpnext::inventory::list_items "$session_id")
        fi
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Unable to retrieve items"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::inventory::cli::add_item() {
    local item_code="${1:-}"
    local item_name="${2:-}"
    local price="${3:-0}"

    if [[ -z "$item_code" ]] || [[ -z "$item_name" ]]; then
        log::error "Item code and name required. Usage: inventory add-item <code> <name> [price]"
        return 1
    fi

    log::info "Adding item: $item_code - $item_name"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Create item
    local item_json="{\"doctype\":\"Item\",\"item_code\":\"${item_code}\",\"item_name\":\"${item_name}\",\"item_group\":\"Products\",\"stock_uom\":\"Unit\",\"is_stock_item\":1,\"valuation_rate\":${price}}"

    local result
    result=$(erpnext::inventory::create_item "$item_json" "$session_id")

    if [[ -n "$result" ]]; then
        log::success "Item created successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to create item"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::inventory::cli::check_stock() {
    local item_code="${1:-}"

    if [[ -z "$item_code" ]]; then
        log::error "Item code required. Usage: inventory check-stock <item_code>"
        return 1
    fi

    log::info "Checking stock for: $item_code"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Check stock balance
    local result
    result=$(erpnext::inventory::get_stock_balance "$item_code" "" "$session_id")

    if [[ -n "$result" ]]; then
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::info "No stock found for item: $item_code"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::inventory::cli::list_warehouses() {
    log::info "Listing warehouses..."

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # List warehouses
    local result
    result=$(erpnext::inventory::list_warehouses "$session_id")

    if [[ -n "$result" ]]; then
        local count=$(echo "$result" | jq '.data | length' 2>/dev/null || echo "0")
        if [[ "$count" -eq "0" ]]; then
            log::info "No warehouses found. Creating default warehouse..."
            local warehouse='{"doctype":"Warehouse","warehouse_name":"Main Warehouse","warehouse_type":"Transit"}'
            erpnext::inventory::create_warehouse "$warehouse" "$session_id" &>/dev/null
            result=$(erpnext::inventory::list_warehouses "$session_id")
        fi
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Unable to retrieve warehouses"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::inventory::cli::create_po() {
    local supplier="${1:-}"
    local item_code="${2:-}"
    local qty="${3:-1}"
    local rate="${4:-100}"

    if [[ -z "$supplier" ]] || [[ -z "$item_code" ]]; then
        log::error "Supplier and item code required. Usage: inventory create-po <supplier> <item_code> [qty] [rate]"
        return 1
    fi

    log::info "Creating purchase order for $qty x $item_code from $supplier"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Create purchase order
    local po_json="{
        \"doctype\":\"Purchase Order\",
        \"supplier\":\"${supplier}\",
        \"transaction_date\":\"$(date +%Y-%m-%d)\",
        \"items\":[{
            \"item_code\":\"${item_code}\",
            \"qty\":${qty},
            \"rate\":${rate},
            \"schedule_date\":\"$(date -d '+7 days' +%Y-%m-%d)\"
        }]
    }"

    local result
    result=$(erpnext::inventory::create_purchase_order "$po_json" "$session_id")

    if [[ -n "$result" ]]; then
        log::success "Purchase order created successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to create purchase order"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::inventory::list_items
export -f erpnext::inventory::create_item
export -f erpnext::inventory::get_item
export -f erpnext::inventory::update_item
export -f erpnext::inventory::get_stock_balance
export -f erpnext::inventory::create_stock_entry
export -f erpnext::inventory::list_warehouses
export -f erpnext::inventory::create_warehouse
export -f erpnext::inventory::create_purchase_order
export -f erpnext::inventory::list_purchase_orders
export -f erpnext::inventory::cli::list_items
export -f erpnext::inventory::cli::add_item
export -f erpnext::inventory::cli::check_stock
export -f erpnext::inventory::cli::list_warehouses
export -f erpnext::inventory::cli::create_po