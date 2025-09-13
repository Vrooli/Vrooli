#!/usr/bin/env bash
################################################################################
# ERPNext Proxy Helper Functions
# 
# Provides proper request routing for ERPNext multi-tenant setup
################################################################################

set -euo pipefail

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/resources/erpnext/config/defaults.sh"

################################################################################
# Proxy Request Functions
################################################################################

erpnext::proxy::request() {
    local method="${1:-GET}"
    local path="${2:-/}"
    local data="${3:-}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    # Build curl command with proper headers
    local curl_cmd="curl -sf -X $method"
    curl_cmd="$curl_cmd -H 'Host: $site_name'"
    curl_cmd="$curl_cmd -H 'X-Forwarded-Host: localhost:${ERPNEXT_PORT}'"
    curl_cmd="$curl_cmd -H 'X-Forwarded-Proto: http'"
    
    if [[ -n "$data" ]]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json'"
        curl_cmd="$curl_cmd -d '$data'"
    fi
    
    curl_cmd="$curl_cmd 'http://localhost:${ERPNEXT_PORT}${path}'"
    
    eval $curl_cmd
}

erpnext::proxy::get() {
    erpnext::proxy::request "GET" "$1"
}

erpnext::proxy::post() {
    erpnext::proxy::request "POST" "$1" "$2"
}

################################################################################
# Browser Access Helper
################################################################################

erpnext::proxy::get_access_url() {
    echo "http://localhost:${ERPNEXT_PORT}"
}

erpnext::proxy::get_login_info() {
    cat <<EOF
=== ERPNext Access Information ===
URL: http://localhost:${ERPNEXT_PORT}
Username: Administrator
Password: ${ERPNEXT_ADMIN_PASSWORD:-admin}

Note: If you see "localhost does not exist", add this to /etc/hosts:
127.0.0.1 vrooli.local

Then access at: http://vrooli.local:${ERPNEXT_PORT}
EOF
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::proxy::request
export -f erpnext::proxy::get
export -f erpnext::proxy::post
export -f erpnext::proxy::get_access_url
export -f erpnext::proxy::get_login_info