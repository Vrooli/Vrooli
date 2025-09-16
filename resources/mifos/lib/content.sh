#!/usr/bin/env bash
################################################################################
# Mifos Content Library
# 
# Content management functions for Mifos X
################################################################################

set -euo pipefail

# ==============================================================================
# CONTENT ADD
# ==============================================================================
mifos::content::add() {
    local content_type="${1:-}"
    local content_data="${2:-}"
    
    if [[ -z "${content_type}" ]]; then
        log::error "Content type required (client|product|loan|savings)"
        return 1
    fi
    
    case "${content_type}" in
        client)
            mifos::content::add_client "${content_data}"
            ;;
        product)
            mifos::content::add_product "${content_data}"
            ;;
        loan)
            mifos::content::add_loan "${content_data}"
            ;;
        savings)
            mifos::content::add_savings "${content_data}"
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            return 1
            ;;
    esac
}

# ==============================================================================
# CONTENT LIST
# ==============================================================================
mifos::content::list() {
    log::header "Available Mifos Content Types"
    
    echo "Content Management:"
    echo "  • clients    - Customer records"
    echo "  • products   - Financial products (loans, savings)"
    echo "  • loans      - Loan accounts"
    echo "  • savings    - Savings accounts"
    echo "  • reports    - Financial reports"
    echo "  • workflows  - Approval workflows"
    echo ""
    echo "Usage Examples:"
    echo "  resource-mifos content add client '{\"firstname\":\"John\",\"lastname\":\"Doe\"}'"
    echo "  resource-mifos content list products"
    echo "  resource-mifos content get loan 12345"
    echo "  resource-mifos content remove client 67890"
    
    return 0
}

# ==============================================================================
# CONTENT GET
# ==============================================================================
mifos::content::get() {
    local content_type="${1:-}"
    local content_id="${2:-}"
    
    if [[ -z "${content_type}" ]]; then
        log::error "Content type required"
        return 1
    fi
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    case "${content_type}" in
        client|clients)
            if [[ -n "${content_id}" ]]; then
                mifos::core::api_request GET "/clients/${content_id}" "" "${auth_token}"
            else
                mifos::core::api_request GET "/clients" "" "${auth_token}"
            fi
            ;;
        product|products)
            if [[ -n "${content_id}" ]]; then
                mifos::core::api_request GET "/loanproducts/${content_id}" "" "${auth_token}"
            else
                echo "Loan Products:"
                mifos::core::api_request GET "/loanproducts" "" "${auth_token}"
                echo ""
                echo "Savings Products:"
                mifos::core::api_request GET "/savingsproducts" "" "${auth_token}"
            fi
            ;;
        loan|loans)
            if [[ -n "${content_id}" ]]; then
                mifos::core::api_request GET "/loans/${content_id}" "" "${auth_token}"
            else
                mifos::core::api_request GET "/loans" "" "${auth_token}"
            fi
            ;;
        savings)
            if [[ -n "${content_id}" ]]; then
                mifos::core::api_request GET "/savingsaccounts/${content_id}" "" "${auth_token}"
            else
                mifos::core::api_request GET "/savingsaccounts" "" "${auth_token}"
            fi
            ;;
        *)
            log::error "Unknown content type: ${content_type}"
            return 1
            ;;
    esac
}

# ==============================================================================
# CONTENT REMOVE
# ==============================================================================
mifos::content::remove() {
    local content_type="${1:-}"
    local content_id="${2:-}"
    
    if [[ -z "${content_type}" ]] || [[ -z "${content_id}" ]]; then
        log::error "Content type and ID required"
        return 1
    fi
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    case "${content_type}" in
        client)
            mifos::core::api_request DELETE "/clients/${content_id}" "" "${auth_token}"
            ;;
        loan)
            mifos::core::api_request DELETE "/loans/${content_id}" "" "${auth_token}"
            ;;
        savings)
            mifos::core::api_request DELETE "/savingsaccounts/${content_id}" "" "${auth_token}"
            ;;
        *)
            log::error "Cannot remove content type: ${content_type}"
            return 1
            ;;
    esac
    
    log::success "Content removed: ${content_type} ${content_id}"
    return 0
}

# ==============================================================================
# CONTENT EXECUTE
# ==============================================================================
mifos::content::execute() {
    local action="${1:-}"
    shift
    
    case "${action}" in
        seed-demo)
            mifos::content::seed_demo_data "$@"
            ;;
        generate-report)
            mifos::content::generate_report "$@"
            ;;
        export-data)
            mifos::content::export_data "$@"
            ;;
        *)
            log::error "Unknown action: ${action}"
            return 1
            ;;
    esac
}

# ==============================================================================
# ADD CLIENT
# ==============================================================================
mifos::content::add_client() {
    local client_data="${1:-}"
    
    if [[ -z "${client_data}" ]]; then
        # Use default client data
        client_data='{
            "officeId": 1,
            "firstname": "Demo",
            "lastname": "Client",
            "dateOfBirth": "01 January 1990",
            "locale": "en",
            "dateFormat": "dd MMMM yyyy",
            "active": true,
            "activationDate": "01 January 2024"
        }'
    fi
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    local response
    response=$(mifos::core::api_request POST "/clients" "${client_data}" "${auth_token}")
    
    if local client_id=$(echo "${response}" | jq -r '.clientId // empty'); then
        log::success "Client created with ID: ${client_id}"
        echo "${response}"
        return 0
    else
        log::error "Failed to create client"
        echo "${response}"
        return 1
    fi
}

# ==============================================================================
# SEED DEMO DATA
# ==============================================================================
mifos::content::seed_demo_data() {
    log::header "Seeding Demo Data"
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    # Create demo clients
    log::info "Creating ${MIFOS_DEMO_CLIENTS_COUNT} demo clients..."
    for ((i=1; i<=MIFOS_DEMO_CLIENTS_COUNT; i++)); do
        local client_data="{
            \"officeId\": 1,
            \"firstname\": \"Demo\",
            \"lastname\": \"Client${i}\",
            \"dateOfBirth\": \"01 January 1990\",
            \"locale\": \"en\",
            \"dateFormat\": \"dd MMMM yyyy\",
            \"active\": true,
            \"activationDate\": \"01 January 2024\"
        }"
        
        if mifos::core::api_request POST "/clients" "${client_data}" "${auth_token}" &>/dev/null; then
            echo "  ✓ Client ${i} created"
        else
            echo "  ✗ Failed to create client ${i}"
        fi
    done
    
    log::success "Demo data seeding complete"
    return 0
}