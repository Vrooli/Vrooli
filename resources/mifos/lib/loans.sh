#!/usr/bin/env bash
################################################################################
# Mifos Loans Library
# 
# Loan management functions for Mifos X
################################################################################

set -euo pipefail

# ==============================================================================
# DISBURSE LOAN
# ==============================================================================
mifos::loans::disburse() {
    local loan_id="${1}"
    local amount="${2:-}"
    
    if [[ -z "${loan_id}" ]]; then
        log::error "Loan ID required"
        return 1
    fi
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    local disburse_data="{
        \"locale\": \"en\",
        \"dateFormat\": \"dd MMMM yyyy\",
        \"actualDisbursementDate\": \"$(date '+%d %B %Y')\",
        \"transactionAmount\": ${amount:-0}
    }"
    
    log::info "Disbursing loan ${loan_id}"
    
    local response
    response=$(mifos::core::api_request POST "/loans/${loan_id}?command=disburse" "${disburse_data}" "${auth_token}")
    
    if echo "${response}" | jq -e '.loanId' &>/dev/null; then
        log::success "Loan ${loan_id} disbursed successfully"
        echo "${response}" | jq '.'
        return 0
    else
        log::error "Failed to disburse loan"
        echo "${response}"
        return 1
    fi
}

# ==============================================================================
# CHECK PAYMENT STATUS
# ==============================================================================
mifos::loans::payment_status() {
    local loan_id="${1}"
    
    if [[ -z "${loan_id}" ]]; then
        log::error "Loan ID required"
        return 1
    fi
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    log::info "Checking payment status for loan ${loan_id}"
    
    local response
    response=$(mifos::core::api_request GET "/loans/${loan_id}" "" "${auth_token}")
    
    if echo "${response}" | jq -e '.id' &>/dev/null; then
        echo "Loan ID: $(echo "${response}" | jq -r '.id')"
        echo "Status: $(echo "${response}" | jq -r '.status.value // "Unknown"')"
        echo "Principal: $(echo "${response}" | jq -r '.principal // 0')"
        echo "Outstanding: $(echo "${response}" | jq -r '.summary.totalOutstanding // 0')"
        echo "Paid: $(echo "${response}" | jq -r '.summary.totalRepayment // 0')"
        return 0
    else
        log::error "Failed to get loan status"
        echo "${response}"
        return 1
    fi
}

# ==============================================================================
# CREATE LOAN PRODUCT
# ==============================================================================
mifos::products::create_loan() {
    local product_name="${1:-Demo Loan Product}"
    local interest_rate="${2:-10}"
    
    local auth_token
    auth_token=$(mifos::core::authenticate) || return 1
    
    local product_data="{
        \"name\": \"${product_name}\",
        \"shortName\": \"DLP\",
        \"currencyCode\": \"${MIFOS_BASE_CURRENCY}\",
        \"digitsAfterDecimal\": 2,
        \"principal\": 10000,
        \"numberOfRepayments\": 12,
        \"repaymentEvery\": 1,
        \"repaymentFrequencyType\": 2,
        \"interestRatePerPeriod\": ${interest_rate},
        \"interestRateFrequencyType\": 2,
        \"amortizationType\": 1,
        \"interestType\": 0,
        \"interestCalculationPeriodType\": 1,
        \"transactionProcessingStrategyId\": 1,
        \"locale\": \"en\",
        \"dateFormat\": \"dd MMMM yyyy\"
    }"
    
    log::info "Creating loan product: ${product_name}"
    
    local response
    response=$(mifos::core::api_request POST "/loanproducts" "${product_data}" "${auth_token}")
    
    if local product_id=$(echo "${response}" | jq -r '.resourceId // empty'); then
        log::success "Loan product created with ID: ${product_id}"
        echo "${response}" | jq '.'
        return 0
    else
        log::error "Failed to create loan product"
        echo "${response}"
        return 1
    fi
}