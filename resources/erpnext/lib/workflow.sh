#!/usr/bin/env bash
################################################################################
# ERPNext Workflow Engine Helper Functions
# 
# Provides workflow management capabilities for ERPNext
################################################################################

set -euo pipefail

# Source dependencies
source "${APP_ROOT}/resources/erpnext/lib/api.sh"
source "${APP_ROOT}/resources/erpnext/config/defaults.sh"

################################################################################
# Workflow Management Functions
################################################################################

erpnext::workflow::list() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow access"
        return 1
    fi
    
    # List all workflows
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.query_builder.run" \
        -d "doctype=Workflow&fields=[\"name\",\"document_type\",\"is_active\"]&filters=[]&order_by=name&limit=100" 2>/dev/null
}

erpnext::workflow::get() {
    local workflow_name="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow access"
        return 1
    fi
    
    # Get workflow details
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.form.load.getdoc" \
        -d "doctype=Workflow&name=${workflow_name}" 2>/dev/null
}

erpnext::workflow::create() {
    local workflow_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow creation"
        return 1
    fi
    
    # Create new workflow
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${workflow_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.form.save.savedocs" 2>/dev/null
}

erpnext::workflow::execute_transition() {
    local doctype="${1}"
    local doc_name="${2}"
    local action="${3}"
    local session_id="${4}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow transition"
        return 1
    fi
    
    # Execute workflow transition
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "{\"doc\":{\"doctype\":\"${doctype}\",\"name\":\"${doc_name}\"},\"action\":\"${action}\"}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.workflow.doctype.workflow.workflow.apply_workflow" 2>/dev/null
}

################################################################################
# Workflow State Functions
################################################################################

erpnext::workflow::get_states() {
    local workflow_name="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi
    
    # Get workflow states
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.query_builder.run" \
        -d "doctype=Workflow%20Document%20State&fields=[\"state\",\"doc_status\"]&filters=[[\"parent\",\"=\",\"${workflow_name}\"]]" 2>/dev/null
}

erpnext::workflow::get_transitions() {
    local workflow_name="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"
    
    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi
    
    # Get workflow transitions
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/method/frappe.desk.query_builder.run" \
        -d "doctype=Workflow%20Transition&fields=[\"state\",\"next_state\",\"action\",\"allowed\"]&filters=[[\"parent\",\"=\",\"${workflow_name}\"]]" 2>/dev/null
}

################################################################################
# CLI Wrapper Functions
################################################################################

erpnext::workflow::cli::list() {
    log::info "Listing ERPNext workflows..."
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # List workflows
    local result
    result=$(erpnext::workflow::list "$session_id")
    
    if [[ -n "$result" ]]; then
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::info "No workflows found or unable to retrieve"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::workflow::cli::get() {
    local workflow_name="${1:-}"
    
    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name required. Usage: workflow get <name>"
        return 1
    fi
    
    log::info "Getting workflow: $workflow_name"
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # Get workflow
    local result
    result=$(erpnext::workflow::get "$workflow_name" "$session_id")
    
    if [[ -n "$result" ]]; then
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Workflow not found: $workflow_name"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::workflow::cli::create() {
    local workflow_file="${1:-}"
    
    if [[ -z "$workflow_file" ]] || [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow JSON file required. Usage: workflow create <file.json>"
        return 1
    fi
    
    log::info "Creating workflow from: $workflow_file"
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # Create workflow
    local workflow_json
    workflow_json=$(cat "$workflow_file")
    
    local result
    result=$(erpnext::workflow::create "$workflow_json" "$session_id")
    
    if [[ -n "$result" ]]; then
        log::success "Workflow created successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to create workflow"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::workflow::cli::transition() {
    local doctype="${1:-}"
    local doc_name="${2:-}"
    local action="${3:-}"
    
    if [[ -z "$doctype" ]] || [[ -z "$doc_name" ]] || [[ -z "$action" ]]; then
        log::error "All parameters required. Usage: workflow transition <doctype> <doc_name> <action>"
        return 1
    fi
    
    log::info "Executing workflow transition: $action on $doctype/$doc_name"
    
    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")
    
    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi
    
    # Execute transition
    local result
    result=$(erpnext::workflow::execute_transition "$doctype" "$doc_name" "$action" "$session_id")
    
    if [[ -n "$result" ]]; then
        log::success "Transition executed successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to execute transition"
    fi
    
    # Logout
    erpnext::api::logout "$session_id"
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::workflow::list
export -f erpnext::workflow::get
export -f erpnext::workflow::create
export -f erpnext::workflow::execute_transition
export -f erpnext::workflow::get_states
export -f erpnext::workflow::get_transitions
export -f erpnext::workflow::cli::list
export -f erpnext::workflow::cli::get
export -f erpnext::workflow::cli::create
export -f erpnext::workflow::cli::transition