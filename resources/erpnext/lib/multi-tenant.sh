#!/usr/bin/env bash
################################################################################
# ERPNext Multi-tenant Support Module
# 
# Provides multi-company/organization management functionality
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
# Company Management
################################################################################

erpnext::multi_tenant::list_companies() {
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get all companies/organizations
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Company?fields=[\"name\",\"company_name\",\"abbr\",\"default_currency\",\"enabled\"]" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq -r '.data[] | "\(.name) (\(.abbr)) - Currency: \(.default_currency), Enabled: \(.enabled)"' 2>/dev/null || echo "No companies found"
    else
        echo "No companies found or unable to retrieve"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

erpnext::multi_tenant::create_company() {
    local company_name="${1}"
    local abbr="${2}"
    local currency="${3:-USD}"
    local country="${4:-United States}"
    
    if [[ -z "$company_name" ]] || [[ -z "$abbr" ]]; then
        log::error "Usage: multi-tenant create-company <company_name> <abbreviation> [currency] [country]"
        return 1
    fi
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Create new company/organization
    local company_data
    company_data=$(jq -n \
        --arg name "$company_name" \
        --arg abbr "$abbr" \
        --arg curr "$currency" \
        --arg country "$country" \
        '{
            "doctype": "Company",
            "company_name": $name,
            "abbr": $abbr,
            "default_currency": $curr,
            "country": $country,
            "create_chart_of_accounts_based_on": "Standard Template",
            "chart_of_accounts": "Standard",
            "enabled": 1
        }')
    
    local response
    response=$(timeout 10 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${company_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Company" 2>/dev/null)

    if [[ -n "$response" ]]; then
        log::success "Company '$company_name' created successfully"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to create company"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# User-Company Assignment
################################################################################

erpnext::multi_tenant::assign_user_to_company() {
    local user_email="${1}"
    local company_name="${2}"

    if [[ -z "$user_email" ]] || [[ -z "$company_name" ]]; then
        log::error "Usage: multi-tenant assign-user <user_email> <company_name> [role]"
        return 1
    fi
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Create user permission for company
    local permission_data
    permission_data=$(jq -n \
        --arg user "$user_email" \
        --arg company "$company_name" \
        '{
            "doctype": "User Permission",
            "user": $user,
            "allow": "Company",
            "for_value": $company,
            "apply_to_all_doctypes": 1
        }')
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${permission_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/User Permission" 2>/dev/null)

    if [[ -n "$response" ]]; then
        log::success "User '$user_email' assigned to company '$company_name'"
    else
        log::warn "User assignment may already exist or failed"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Company Switching
################################################################################

erpnext::multi_tenant::switch_company() {
    local company_name="${1}"
    
    if [[ -z "$company_name" ]]; then
        log::error "Usage: multi-tenant switch-company <company_name>"
        return 1
    fi
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Set default company in session
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "{\"company\":\"${company_name}\"}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.doctype.desktop_icon.desktop_icon.set_default_company" 2>/dev/null)

    if [[ -n "$response" ]]; then
        log::success "Switched to company '$company_name'"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        log::error "Failed to switch company"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Company-specific Data
################################################################################

erpnext::multi_tenant::get_company_data() {
    local company_name="${1}"
    local doctype="${2:-Customer}"
    
    if [[ -z "$company_name" ]]; then
        log::error "Usage: multi-tenant get-data <company_name> [doctype]"
        return 1
    fi
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Get data filtered by company
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/${doctype}?filters=[[\"company\",\"=\",\"${company_name}\"]]" 2>/dev/null)
    
    if [[ -n "$response" ]]; then
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
    else
        echo "No data found for company '$company_name'"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Company Settings
################################################################################

erpnext::multi_tenant::configure_company() {
    local company_name="${1}"
    local setting="${2}"
    local value="${3}"
    
    if [[ -z "$company_name" ]] || [[ -z "$setting" ]] || [[ -z "$value" ]]; then
        log::error "Usage: multi-tenant configure <company_name> <setting> <value>"
        return 1
    fi
    
    local session_id
    session_id=$(erpnext::api::login 2>/dev/null) || {
        log::error "Failed to authenticate"
        return 1
    }
    
    # Update company setting
    local update_data
    update_data=$(jq -n \
        --arg setting "$setting" \
        --arg value "$value" \
        '{($setting): $value}')

    local response
    response=$(timeout 5 curl -sf -X PUT \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${update_data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Company/${company_name}" 2>/dev/null)

    if [[ -n "$response" ]]; then
        log::success "Company '$company_name' setting '$setting' updated"
    else
        log::error "Failed to update company setting"
    fi
    
    erpnext::api::logout "$session_id" 2>/dev/null || true
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::multi_tenant::list_companies
export -f erpnext::multi_tenant::create_company
export -f erpnext::multi_tenant::assign_user_to_company
export -f erpnext::multi_tenant::switch_company
export -f erpnext::multi_tenant::get_company_data
export -f erpnext::multi_tenant::configure_company