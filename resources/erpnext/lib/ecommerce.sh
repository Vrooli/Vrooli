#!/usr/bin/env bash
################################################################################
# ERPNext E-commerce Module Functions
# 
# Provides shopping cart and online store functionality
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
# E-commerce Product Management
################################################################################

erpnext::ecommerce::list_products() {
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get items marked as show_in_website
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Item?filters=[[\"show_in_website\",\"=\",1]]&fields=[\"item_code\",\"item_name\",\"standard_rate\",\"stock_uom\"]" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq -r '.data[] | "\(.item_code) - \(.item_name) (\(.standard_rate) \(.stock_uom))"' 2>/dev/null || echo "No products found"
    else
        echo "No products found or unable to retrieve"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::ecommerce::add_product() {
    local item_code="${1}"
    local item_name="${2}"
    local price="${3:-0}"
    local description="${4:-}"
    
    if [[ -z "$item_code" ]] || [[ -z "$item_name" ]]; then
        log::error "Usage: ecommerce add-product <item_code> <item_name> [price] [description]"
        return 1
    }
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Create item with e-commerce settings
    local item_data=$(jq -n \
        --arg code "$item_code" \
        --arg name "$item_name" \
        --arg price "$price" \
        --arg desc "$description" \
        '{
            "doctype": "Item",
            "item_code": $code,
            "item_name": $name,
            "item_group": "Products",
            "stock_uom": "Unit",
            "is_stock_item": 1,
            "show_in_website": 1,
            "website_item_groups": [{"item_group": "Products"}],
            "standard_rate": ($price | tonumber),
            "description": $desc
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${item_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Item" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Product '$item_name' added successfully"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to add product"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Shopping Cart Operations
################################################################################

erpnext::ecommerce::get_cart() {
    local customer="${1:-Guest}"
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get active shopping cart for customer
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/erpnext.shopping_cart.cart.get_cart_quotation" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        echo "Cart is empty or unable to retrieve"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::ecommerce::add_to_cart() {
    local item_code="${1}"
    local qty="${2:-1}"
    
    if [[ -z "$item_code" ]]; then
        log::error "Usage: ecommerce add-to-cart <item_code> [quantity]"
        return 1
    }
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Add item to cart
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "{\"item_code\":\"${item_code}\",\"qty\":${qty}}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/erpnext.shopping_cart.cart.update_cart" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Added ${qty} x ${item_code} to cart"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to add to cart"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Store Configuration
################################################################################

erpnext::ecommerce::configure_store() {
    local store_name="${1:-My Store}"
    local currency="${2:-USD}"
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Configure shopping cart settings
    local settings_data=$(jq -n \
        --arg name "$store_name" \
        --arg curr "$currency" \
        '{
            "doctype": "Shopping Cart Settings",
            "enabled": 1,
            "company": $name,
            "default_currency": $curr,
            "enable_checkout": 1,
            "show_price": 1,
            "show_stock_availability": 1
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${settings_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Shopping Cart Settings" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Store configured successfully"
    else
        log::warn "Store configuration may already exist or failed"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::ecommerce::list_products
export -f erpnext::ecommerce::add_product
export -f erpnext::ecommerce::get_cart
export -f erpnext::ecommerce::add_to_cart
export -f erpnext::ecommerce::configure_store