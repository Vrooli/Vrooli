#!/usr/bin/env bash
################################################################################
# ERPNext API Helper Functions
# 
# Provides authenticated API access to ERPNext
################################################################################

set -euo pipefail

# Define log functions if not already available
if ! declare -f log::error &>/dev/null; then
    log::error() { echo "[ERROR] $*" >&2; }
    log::info() { echo "[INFO] $*"; }
fi

# Source dependencies if not already loaded
if [[ -z "${ERPNEXT_PORT:-}" ]]; then
    source "${APP_ROOT}/resources/erpnext/config/defaults.sh"
fi

################################################################################
# Utility Functions
################################################################################

# URL encode a string (for query parameters)
erpnext::api::urlencode() {
    local string="${1}"
    local strlen=${#string}
    local encoded=""
    local pos c o

    for (( pos=0 ; pos<strlen ; pos++ )); do
        c=${string:$pos:1}
        case "$c" in
            [-_.~a-zA-Z0-9] ) o="${c}" ;;
            * ) printf -v o '%%%02x' "'$c"
        esac
        encoded+="${o}"
    done
    echo "${encoded}"
}

################################################################################
# Authentication Functions
################################################################################

erpnext::api::login() {
    local username="${1:-Administrator}"
    local password="${2:-${ERPNEXT_ADMIN_PASSWORD:-admin}}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    # Use cookie jar to capture session
    local cookie_file="/tmp/erpnext_cookies_$$"

    # shellcheck disable=SC2034  # response is used for exit code check
    if timeout 5 curl -sf -c "$cookie_file" -X POST \
        -H "Host: ${site_name}" \
        -H "Content-Type: application/json" \
        -d "{\"usr\":\"${username}\",\"pwd\":\"${password}\"}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/login" &>/dev/null; then

        # Extract session ID from cookie file (last field on sid line)
        local sid
        sid=$(grep -E "sid\s+" "$cookie_file" 2>/dev/null | awk '{print $NF}')

        # Clean up cookie file
        rm -f "$cookie_file"

        if [[ -n "$sid" ]]; then
            echo "$sid"
        else
            return 1
        fi
    else
        # Clean up cookie file on failure
        rm -f "$cookie_file"
        return 1
    fi
}

erpnext::api::logout() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for logout"
        return 1
    fi
    
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/logout" &>/dev/null
}

################################################################################
# API Request Functions
################################################################################

erpnext::api::get() {
    local endpoint="${1}"
    local session_id="${2:-}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -n "$session_id" ]]; then
        timeout 5 curl -sf \
            -H "Host: ${site_name}" \
            -H "Cookie: sid=${session_id}" \
            "http://localhost:${ERPNEXT_PORT}${endpoint}"
    else
        timeout 5 curl -sf \
            -H "Host: ${site_name}" \
            "http://localhost:${ERPNEXT_PORT}${endpoint}"
    fi
}

erpnext::api::post() {
    local endpoint="${1}"
    local data="${2}"
    local session_id="${3:-}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -n "$session_id" ]]; then
        timeout 5 curl -sf -X POST \
            -H "Host: ${site_name}" \
            -H "Content-Type: application/json" \
            -H "Cookie: sid=${session_id}" \
            -d "${data}" \
            "http://localhost:${ERPNEXT_PORT}${endpoint}"
    else
        timeout 5 curl -sf -X POST \
            -H "Host: ${site_name}" \
            -H "Content-Type: application/json" \
            -d "${data}" \
            "http://localhost:${ERPNEXT_PORT}${endpoint}"
    fi
}

################################################################################
# DocType Operations
################################################################################

erpnext::api::list_doctypes() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for API access"
        return 1
    fi
    
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.desktop.get_desktop_page" 2>/dev/null
}

erpnext::api::get_doctype() {
    local doctype="${1}"
    local name="${2:-}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for API access"
        return 1
    fi
    
    local endpoint="/api/resource/${doctype}"
    if [[ -n "$name" ]]; then
        endpoint="${endpoint}/${name}"
    fi
    
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}${endpoint}"
}

erpnext::api::create_doctype() {
    local doctype="${1}"
    local data="${2}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for API access"
        return 1
    fi
    
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Content-Type: application/json" \
        -H "Cookie: sid=${session_id}" \
        -d "${data}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/${doctype}"
}

################################################################################
# Module Operations
################################################################################

erpnext::api::get_modules() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for API access"
        return 1
    fi
    
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.moduleview.get_modules" 2>/dev/null
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::api::urlencode
export -f erpnext::api::login
export -f erpnext::api::logout
export -f erpnext::api::get
export -f erpnext::api::post
export -f erpnext::api::list_doctypes
export -f erpnext::api::get_doctype
export -f erpnext::api::create_doctype
export -f erpnext::api::get_modules