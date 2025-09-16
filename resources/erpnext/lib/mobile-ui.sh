#!/usr/bin/env bash
################################################################################
# ERPNext Mobile UI Enhancement Module
# 
# Provides mobile-responsive UI configuration and customization
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
# Mobile UI Configuration
################################################################################

erpnext::mobile_ui::enable_responsive() {
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Enable mobile-responsive settings
    local settings_data=$(jq -n '{
        "doctype": "Website Settings",
        "enable_view_tracking": 1,
        "disable_signup": 0,
        "hide_footer_signup": 0,
        "show_language_selector": 0,
        "banner_html": "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">",
        "navbar_template": "Standard Navbar",
        "footer_template": "Standard Footer"
    }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${settings_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Website Settings" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Mobile responsive UI enabled"
    else
        log::warn "Mobile UI settings may already exist"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::mobile_ui::configure_theme() {
    local theme_name="${1:-ERPNext Mobile}"
    local primary_color="${2:-#5e64ff}"
    local background_color="${3:-#ffffff}"
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Create or update mobile-optimized theme
    local theme_data=$(jq -n \
        --arg name "$theme_name" \
        --arg primary "$primary_color" \
        --arg bg "$background_color" \
        '{
            "doctype": "Website Theme",
            "theme": $name,
            "module": "Website",
            "custom": 1,
            "theme_scss": "// Mobile-first responsive design\n@media (max-width: 768px) {\n  .navbar { padding: 0.5rem; }\n  .container { padding: 0 15px; }\n  .page-content { padding: 1rem; }\n  .form-control { font-size: 16px; }\n  .btn { padding: 0.5rem 1rem; }\n  table { font-size: 14px; }\n  .desktop-only { display: none !important; }\n}\n\n@media (min-width: 769px) {\n  .mobile-only { display: none !important; }\n}",
            "primary_color": $primary,
            "background_color": $bg,
            "font_size": "14px",
            "button_rounded_corners": 1,
            "button_shadows": 0
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${theme_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Website Theme" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Mobile theme '$theme_name' configured"
    else
        log::warn "Theme configuration may already exist"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Mobile App Configuration
################################################################################

erpnext::mobile_ui::configure_pwa() {
    local app_name="${1:-ERPNext}"
    local app_short_name="${2:-ERP}"
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Configure Progressive Web App settings
    local pwa_data=$(jq -n \
        --arg name "$app_name" \
        --arg short "$app_short_name" \
        '{
            "doctype": "Website Settings",
            "app_name": $name,
            "app_short_name": $short,
            "app_logo": "/assets/erpnext/images/erp-icon.svg",
            "splash_image": "/assets/erpnext/images/splash.png",
            "enable_splash_screen": 1,
            "theme_color": "#5e64ff",
            "background_color": "#ffffff",
            "start_url": "/app",
            "display": "standalone",
            "orientation": "portrait"
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${pwa_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.website.doctype.website_settings.website_settings.set_pwa_settings" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "PWA configuration set for '$app_name'"
    else
        log::warn "PWA settings may already be configured"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Mobile Menu Configuration
################################################################################

erpnext::mobile_ui::configure_mobile_menu() {
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Configure mobile-optimized menu items
    local menu_items=(
        '{"label": "Dashboard", "url": "/app", "icon": "dashboard"}'
        '{"label": "Sales", "url": "/app/sales", "icon": "shopping-cart"}'
        '{"label": "Inventory", "url": "/app/stock", "icon": "box"}'
        '{"label": "Reports", "url": "/app/reports", "icon": "bar-chart"}'
        '{"label": "Settings", "url": "/app/settings", "icon": "settings"}'
    )
    
    for item in "${menu_items[@]}"; do
        local response
        response=$(timeout 5 curl -sf -X POST \
            -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
            -H "Cookie: sid=${session_id}" \
            -H "Content-Type: application/json" \
            -d "{\"doctype\":\"Top Bar Item\",\"parent_label\":\"Mobile Menu\",\"item\":$item}" \
            "http://localhost:${ERPNEXT_PORT}/api/resource/Top Bar Item" 2>/dev/null) || true
    done
    
    log::success "Mobile menu configured"
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Touch Optimization
################################################################################

erpnext::mobile_ui::optimize_touch() {
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Add custom CSS for touch optimization
    local css_data=$(jq -n '{
        "doctype": "Custom CSS",
        "css": "/* Touch-friendly targets */\n.clickable, .btn, a, input, select, textarea {\n  min-height: 44px;\n  min-width: 44px;\n}\n\n/* Improved touch scrolling */\n.scrollable {\n  -webkit-overflow-scrolling: touch;\n  overflow-y: auto;\n}\n\n/* Disable hover effects on touch devices */\n@media (hover: none) {\n  .hover-effect:hover {\n    background: inherit;\n  }\n}\n\n/* Larger form inputs on mobile */\n@media (max-width: 768px) {\n  input, select, textarea {\n    font-size: 16px !important;\n  }\n}"
    }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${css_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.website.doctype.website_settings.website_settings.add_custom_css" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Touch optimization applied"
    else
        log::warn "Touch optimization may already be configured"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Mobile Dashboard
################################################################################

erpnext::mobile_ui::create_mobile_dashboard() {
    local dashboard_name="${1:-Mobile Dashboard}"
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Create mobile-optimized dashboard
    local dashboard_data=$(jq -n \
        --arg name "$dashboard_name" \
        '{
            "doctype": "Dashboard",
            "dashboard_name": $name,
            "is_default": 0,
            "is_standard": 0,
            "module": "Core",
            "charts": [
                {
                    "chart": "Sales Overview",
                    "width": "Full"
                },
                {
                    "chart": "Inventory Status",
                    "width": "Half"
                },
                {
                    "chart": "Cash Flow",
                    "width": "Half"
                }
            ],
            "cards": [
                {
                    "card": "Total Sales",
                    "width": "Half"
                },
                {
                    "card": "Pending Orders",
                    "width": "Half"
                }
            ]
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${dashboard_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Dashboard" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        log::success "Mobile dashboard '$dashboard_name' created"
    else
        log::warn "Dashboard may already exist"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::mobile_ui::enable_responsive
export -f erpnext::mobile_ui::configure_theme
export -f erpnext::mobile_ui::configure_pwa
export -f erpnext::mobile_ui::configure_mobile_menu
export -f erpnext::mobile_ui::optimize_touch
export -f erpnext::mobile_ui::create_mobile_dashboard