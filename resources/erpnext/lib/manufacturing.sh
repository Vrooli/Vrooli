#!/usr/bin/env bash
################################################################################
# ERPNext Manufacturing Module Functions
# 
# Provides production planning and control functionality
################################################################################

set -euo pipefail

# Source dependencies if not already loaded
if [[ -z "${ERPNEXT_PORT:-}" ]]; then
    source "${APP_ROOT}/resources/erpnext/config/defaults.sh"
fi

if [[ -f "${APP_ROOT}/resources/erpnext/lib/api.sh" ]]; then
    source "${APP_ROOT}/resources/erpnext/lib/api.sh"
fi

################################################################################
# Bill of Materials (BOM) Management
################################################################################

erpnext::manufacturing::list_boms() {
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get all BOMs
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/BOM?fields=[\"name\",\"item\",\"quantity\",\"is_active\"]" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq -r '.data[] | "\(.name) - \(.item) (Qty: \(.quantity), Active: \(.is_active))"' 2>/dev/null || echo "No BOMs found"
    else
        echo "No BOMs found or unable to retrieve"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::manufacturing::create_bom() {
    local item_code="${1}"
    local quantity="${2:-1}"
    
    if [[ -z "$item_code" ]]; then
        log::error "Usage: manufacturing create-bom <item_code> [quantity]"
        return 1
    }
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Create a basic BOM
    local bom_data=$(jq -n \
        --arg item "$item_code" \
        --arg qty "$quantity" \
        '{
            "doctype": "BOM",
            "item": $item,
            "quantity": ($qty | tonumber),
            "is_active": 1,
            "is_default": 1,
            "items": []
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${bom_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/BOM" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "BOM for '$item_code' created successfully"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to create BOM"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::manufacturing::add_bom_item() {
    local bom_name="${1}"
    local item_code="${2}"
    local qty="${3:-1}"
    
    if [[ -z "$bom_name" ]] || [[ -z "$item_code" ]]; then
        log::error "Usage: manufacturing add-bom-item <bom_name> <item_code> [quantity]"
        return 1
    }
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get existing BOM
    local bom_response
    bom_response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/BOM/${bom_name}" 2>/dev/null)
    
    if [[ -z "$bom_response" ]]; then
        log::error "BOM not found: $bom_name"
        erpnext::api::logout "$session_id" 2>/dev/null || true
        return 1
    }
    
    # Add item to BOM items array
    local updated_bom=$(echo "$bom_response" | jq \
        --arg item "$item_code" \
        --arg qty "$qty" \
        '.data.items += [{"item_code": $item, "qty": ($qty | tonumber)}]')
    
    # Update BOM
    local response
    response=$(timeout 5 curl -sf -X PUT \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${updated_bom}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/BOM/${bom_name}" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Added ${qty} x ${item_code} to BOM"
    else
        log::error "Failed to update BOM"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Work Order Management
################################################################################

erpnext::manufacturing::list_work_orders() {
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get all work orders
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Work Order?fields=[\"name\",\"production_item\",\"qty\",\"produced_qty\",\"status\"]" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq -r '.data[] | "\(.name) - \(.production_item) (\(.produced_qty)/\(.qty), Status: \(.status))"' 2>/dev/null || echo "No work orders found"
    else
        echo "No work orders found or unable to retrieve"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::manufacturing::create_work_order() {
    local item_code="${1}"
    local qty="${2:-1}"
    local bom="${3:-}"
    
    if [[ -z "$item_code" ]]; then
        log::error "Usage: manufacturing create-work-order <item_code> [quantity] [bom_name]"
        return 1
    }
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Create work order
    local work_order_data=$(jq -n \
        --arg item "$item_code" \
        --arg qty "$qty" \
        --arg bom "$bom" \
        '{
            "doctype": "Work Order",
            "production_item": $item,
            "qty": ($qty | tonumber),
            "bom_no": (if $bom != "" then $bom else null end),
            "wip_warehouse": "Work In Progress - TC",
            "fg_warehouse": "Finished Goods - TC"
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${work_order_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Work Order" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Work order for ${qty} x ${item_code} created successfully"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to create work order"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Production Planning
################################################################################

erpnext::manufacturing::get_production_plan() {
    local from_date="${1:-$(date +%Y-%m-01)}"
    local to_date="${2:-$(date +%Y-%m-%d)}"
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get production plans
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Production Plan?filters=[[\"posting_date\",\">=\",\"${from_date}\"],[\"posting_date\",\"<=\",\"${to_date}\"]]" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "No production plans found"
    else
        echo "No production plans found for the period"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::manufacturing::get_stock_entry() {
    local work_order="${1:-}"
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get stock entries related to manufacturing
    local filters=""
    if [[ -n "$work_order" ]]; then
        filters="?filters=[[\"work_order\",\"=\",\"${work_order}\"]]"
    else
        filters="?filters=[[\"stock_entry_type\",\"in\",[\"Manufacture\",\"Material Transfer for Manufacture\"]]]"
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Stock Entry${filters}" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "No stock entries found"
    else
        echo "No stock entries found"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::manufacturing::list_boms
export -f erpnext::manufacturing::create_bom
export -f erpnext::manufacturing::add_bom_item
export -f erpnext::manufacturing::list_work_orders
export -f erpnext::manufacturing::create_work_order
export -f erpnext::manufacturing::get_production_plan
export -f erpnext::manufacturing::get_stock_entry