#!/usr/bin/env bash
################################################################################
# Mifos Common Library
# 
# Common utility functions for Mifos resource
################################################################################

set -euo pipefail

# ==============================================================================
# COMMON UTILITIES
# ==============================================================================

# Format currency amounts
mifos::common::format_currency() {
    local amount="${1}"
    local currency="${2:-${MIFOS_BASE_CURRENCY}}"
    
    printf "%s %'.2f" "${currency}" "${amount}"
}

# Parse date for API
mifos::common::parse_date() {
    local date_str="${1}"
    local format="${2:-dd MMMM yyyy}"
    
    echo "${date_str}"
}

# Get current timestamp
mifos::common::timestamp() {
    date '+%Y-%m-%d %H:%M:%S'
}

# Validate JSON
mifos::common::validate_json() {
    local json="${1}"
    
    if echo "${json}" | jq -e . &>/dev/null; then
        return 0
    else
        return 1
    fi
}