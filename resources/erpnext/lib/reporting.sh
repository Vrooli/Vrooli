#!/usr/bin/env bash
################################################################################
# ERPNext Reporting Module Helper Functions
# 
# Provides reporting and analytics capabilities for ERPNext
################################################################################

set -euo pipefail

# Source dependencies
source "${APP_ROOT}/resources/erpnext/lib/api.sh"
source "${APP_ROOT}/resources/erpnext/config/defaults.sh"

################################################################################
# Report Management Functions
################################################################################

erpnext::report::list() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for report access"
        return 1
    fi
    
    # List all reports
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.query_builder.run" \
        -d "doctype=Report&fields=[\"name\",\"ref_doctype\",\"report_type\",\"is_standard\"]&filters=[]&order_by=name&limit=100" 2>/dev/null
}

erpnext::report::get() {
    local report_name="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for report access"
        return 1
    fi
    
    # Get report details
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.form.load.getdoc" \
        -d "doctype=Report&name=${report_name}" 2>/dev/null
}

erpnext::report::execute() {
    local report_name="${1}"
    local filters="${2:-{}}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for report execution"
        return 1
    fi
    
    # Execute report with filters
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "{\"report_name\":\"${report_name}\",\"filters\":${filters}}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.query_report.run" 2>/dev/null
}

erpnext::report::create() {
    local report_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for report creation"
        return 1
    fi
    
    # Create new report
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${report_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.form.save.savedocs" 2>/dev/null
}

################################################################################
# Report Builder Functions
################################################################################

erpnext::report::build_query() {
    local doctype="${1}"
    local fields="${2}"
    local filters="${3:-[]}"
    local session_id="${4}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi
    
    # Build and execute query report
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -d "doctype=${doctype}&fields=${fields}&filters=${filters}&order_by=creation%20desc&limit=100" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.query_builder.run" 2>/dev/null
}

erpnext::report::export() {
    local report_name="${1}"
    local format="${2:-csv}" # csv, xlsx, pdf
    local filters="${3:-{}}"
    local session_id="${4}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for report export"
        return 1
    fi
    
    # Export report in specified format
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.query_report.export_query" \
        -d "report_name=${report_name}&file_format_type=${format}&filters=${filters}" 2>/dev/null
}

################################################################################
# Analytics Functions
################################################################################

erpnext::report::get_dashboard_data() {
    local module="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi
    
    # Get dashboard data for module
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.moduleview.get_dashboard_data" \
        -d "module=${module}" 2>/dev/null
}

erpnext::report::get_chart_data() {
    local chart_name="${1}"
    local timespan="${2:-Last Month}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi
    
    # Get chart data
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.doctype.dashboard_chart.dashboard_chart.get" \
        -d "chart_name=${chart_name}&timespan=${timespan}" 2>/dev/null
}

################################################################################
# CLI Wrapper Functions
################################################################################

erpnext::report::cli::list() {
    log::info "Listing ERPNext reports..."
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # List reports
    local result
    result=$(erpnext::report::list "$session_id")
    
    if [[ -n "$result" ]]; then
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::info "No reports found or unable to retrieve"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::report::cli::get() {
    local report_name="${1:-}"
    
    if [[ -z "$report_name" ]]; then
        log::error "Report name required. Usage: report get <name>"
        return 1
    fi
    
    log::info "Getting report: $report_name"
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # Get report
    local result
    result=$(erpnext::report::get "$report_name" "$session_id")
    
    if [[ -n "$result" ]]; then
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Report not found: $report_name"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::report::cli::execute() {
    local report_name="${1:-}"
    local filters="${2:-{}}"
    
    if [[ -z "$report_name" ]]; then
        log::error "Report name required. Usage: report execute <name> [filters_json]"
        return 1
    fi
    
    log::info "Executing report: $report_name"
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # Execute report
    local result
    result=$(erpnext::report::execute "$report_name" "$filters" "$session_id")
    
    if [[ -n "$result" ]]; then
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to execute report"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::report::cli::create() {
    local report_file="${1:-}"
    
    if [[ -z "$report_file" ]] || [[ ! -f "$report_file" ]]; then
        log::error "Report JSON file required. Usage: report create <file.json>"
        return 1
    fi
    
    log::info "Creating report from: $report_file"
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # Create report
    local report_json
    report_json=$(cat "$report_file")
    
    local result
    result=$(erpnext::report::create "$report_json" "$session_id")
    
    if [[ -n "$result" ]]; then
        log::success "Report created successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to create report"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::report::cli::export() {
    local report_name="${1:-}"
    local format="${2:-csv}"
    local filters="${3:-{}}"
    
    if [[ -z "$report_name" ]]; then
        log::error "Report name required. Usage: report export <name> [format] [filters_json]"
        return 1
    fi
    
    log::info "Exporting report: $report_name as $format"
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # Export report
    local result
    result=$(erpnext::report::export "$report_name" "$format" "$filters" "$session_id")

    if [[ -n "$result" ]]; then
        local output_file
        output_file="/tmp/erpnext_report_${report_name}_$(date +%Y%m%d_%H%M%S).${format}"
        echo "$result" > "$output_file"
        log::success "Report exported to: $output_file"
    else
        log::error "Failed to export report"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::report::list
export -f erpnext::report::get
export -f erpnext::report::execute
export -f erpnext::report::create
export -f erpnext::report::build_query
export -f erpnext::report::export
export -f erpnext::report::get_dashboard_data
export -f erpnext::report::get_chart_data
export -f erpnext::report::cli::list
export -f erpnext::report::cli::get
export -f erpnext::report::cli::execute
export -f erpnext::report::cli::create
export -f erpnext::report::cli::export