#!/usr/bin/env bash
################################################################################
# Mifos Clients Library
# 
# Client management functions for Mifos X
################################################################################

set -euo pipefail

# ==============================================================================
# CREATE CLIENT
# ==============================================================================
mifos::clients::create() {
    local firstname="${1:-Demo}"
    local lastname="${2:-Client}"
    local office_id="${3:-1}"
    
    local client_data="{
        \"officeId\": ${office_id},
        \"firstname\": \"${firstname}\",
        \"lastname\": \"${lastname}\",
        \"dateOfBirth\": \"01 January 1990\",
        \"locale\": \"en\",
        \"dateFormat\": \"dd MMMM yyyy\",
        \"active\": true,
        \"activationDate\": \"$(date '+%d %B %Y')\"
    }"
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    log::info "Creating client: ${firstname} ${lastname}"
    
    local response
    response=$(mifos::core::api_request POST "/clients" "${client_data}" "${auth_token}")
    
    if local client_id=$(echo "${response}" | jq -r '.clientId // empty'); then
        log::success "Client created with ID: ${client_id}"
        echo "${response}" | jq '.'
        return 0
    else
        log::error "Failed to create client"
        echo "${response}"
        return 1
    fi
}

# ==============================================================================
# OPEN ACCOUNT
# ==============================================================================
mifos::clients::open_account() {
    local client_id="${1}"
    local account_type="${2:-savings}"  # savings or loan
    local product_id="${3:-1}"
    
    if [[ -z "${client_id}" ]]; then
        log::error "Client ID required"
        return 1
    fi
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    if [[ "${account_type}" == "savings" ]]; then
        local account_data="{
            \"clientId\": ${client_id},
            \"productId\": ${product_id},
            \"locale\": \"en\",
            \"dateFormat\": \"dd MMMM yyyy\",
            \"submittedOnDate\": \"$(date '+%d %B %Y')\",
            \"nominalAnnualInterestRate\": 2.0
        }"
        
        log::info "Opening savings account for client ${client_id}"
        local response
        response=$(mifos::core::api_request POST "/savingsaccounts" "${account_data}" "${auth_token}")
        
        if local account_id=$(echo "${response}" | jq -r '.savingsId // empty'); then
            log::success "Savings account created with ID: ${account_id}"
            echo "${response}" | jq '.'
            return 0
        else
            log::error "Failed to create savings account"
            echo "${response}"
            return 1
        fi
    else
        log::error "Loan account creation not yet implemented"
        return 1
    fi
}