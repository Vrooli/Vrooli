#!/usr/bin/env bash
################################################################################
# ERPNext CRM Module Functions
# 
# Manages Customer Relationship Management operations
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
# Customer Management
################################################################################

erpnext::crm::list_customers() {
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
        "http://localhost:${ERPNEXT_PORT}/api/resource/Customer" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
customers = data.get('data', [])
if customers:
    for customer in customers:
        print(f\"Customer: {customer.get('name', 'N/A')} - Type: {customer.get('customer_type', 'N/A')}\")
else:
    print('No customers found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve customers"
        return 1
    fi
}

erpnext::crm::create_customer() {
    local name="${1:-}"
    local type="${2:-Company}"
    local session_id="${3:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Customer name is required"
        echo "Usage: resource-erpnext crm create-customer <name> [type]"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local data="{\"customer_name\":\"${name}\",\"customer_type\":\"${type}\",\"customer_group\":\"All Customer Groups\",\"territory\":\"All Territories\"}"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Customer" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    customer = data['data']
    print(f\"✅ Customer created: {customer.get('name', 'Unknown')}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to create customer')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to create customer"
        return 1
    fi
}

################################################################################
# Lead Management
################################################################################

erpnext::crm::list_leads() {
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
        "http://localhost:${ERPNEXT_PORT}/api/resource/Lead" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
leads = data.get('data', [])
if leads:
    for lead in leads:
        print(f\"Lead: {lead.get('lead_name', 'N/A')} - Status: {lead.get('status', 'N/A')}\")
else:
    print('No leads found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve leads"
        return 1
    fi
}

erpnext::crm::create_lead() {
    local name="${1:-}"
    local email="${2:-}"
    local company="${3:-}"
    local session_id="${4:-}"
    
    if [[ -z "$name" ]]; then
        log::error "Lead name is required"
        echo "Usage: resource-erpnext crm create-lead <name> [email] [company]"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    # Build JSON data
    local data="{\"lead_name\":\"${name}\",\"source\":\"Direct\",\"status\":\"Open\""
    [[ -n "$email" ]] && data="${data%\}},\"email_id\":\"${email}\"}"
    [[ -n "$company" ]] && data="${data%\}},\"company_name\":\"${company}\"}"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Lead" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    lead = data['data']
    print(f\"✅ Lead created: {lead.get('name', 'Unknown')} - Status: {lead.get('status', 'Unknown')}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to create lead')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to create lead"
        return 1
    fi
}

################################################################################
# Contact Management
################################################################################

erpnext::crm::list_contacts() {
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
        "http://localhost:${ERPNEXT_PORT}/api/resource/Contact" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
contacts = data.get('data', [])
if contacts:
    for contact in contacts:
        print(f\"Contact: {contact.get('full_name', contact.get('name', 'N/A'))} - Email: {contact.get('email_id', 'N/A')}\")
else:
    print('No contacts found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve contacts"
        return 1
    fi
}

erpnext::crm::create_contact() {
    local first="${1:-}"
    local last="${2:-}"
    local email="${3:-}"
    local session_id="${4:-}"
    
    if [[ -z "$first" ]]; then
        log::error "First name is required"
        echo "Usage: resource-erpnext crm create-contact <first_name> [last_name] [email]"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    # Build JSON data
    local data="{\"first_name\":\"${first}\""
    [[ -n "$last" ]] && data="${data},\"last_name\":\"${last}\""
    
    # Add email as nested array for ERPNext structure
    if [[ -n "$email" ]]; then
        data="${data},\"email_ids\":[{\"email_id\":\"${email}\",\"is_primary\":1}]"
    fi
    data="${data}}"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Contact" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    contact = data['data']
    print(f\"✅ Contact created: {contact.get('name', 'Unknown')}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to create contact')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to create contact"
        return 1
    fi
}

################################################################################
# Opportunity Management
################################################################################

erpnext::crm::list_opportunities() {
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
        "http://localhost:${ERPNEXT_PORT}/api/resource/Opportunity" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
opportunities = data.get('data', [])
if opportunities:
    for opp in opportunities:
        print(f\"Opportunity: {opp.get('title', opp.get('name', 'N/A'))} - Status: {opp.get('status', 'N/A')}\")
else:
    print('No opportunities found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve opportunities"
        return 1
    fi
}

################################################################################
# Export functions for use by other scripts
################################################################################

export -f erpnext::crm::list_customers
export -f erpnext::crm::create_customer
export -f erpnext::crm::list_leads
export -f erpnext::crm::create_lead
export -f erpnext::crm::list_contacts
export -f erpnext::crm::create_contact
export -f erpnext::crm::list_opportunities