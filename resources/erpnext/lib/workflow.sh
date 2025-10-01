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

    # List all workflows using the correct REST API endpoint
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Workflow?fields=[\"name\",\"document_type\",\"is_active\"]&limit_page_length=100" 2>/dev/null
}

erpnext::workflow::get() {
    local workflow_name="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow access"
        return 1
    fi

    # Get workflow details using REST API
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Workflow/${workflow_name}" 2>/dev/null
}

erpnext::workflow::create() {
    local workflow_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow creation"
        return 1
    fi

    # Create new workflow using REST API
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${workflow_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Workflow" 2>/dev/null
}

erpnext::workflow::update() {
    local workflow_name="${1}"
    local workflow_json="${2}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow update"
        return 1
    fi

    # Update workflow using REST API
    timeout 5 curl -sf -X PUT \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${workflow_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Workflow/${workflow_name}" 2>/dev/null
}

erpnext::workflow::delete() {
    local workflow_name="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required for workflow deletion"
        return 1
    fi

    # Delete workflow using REST API
    timeout 5 curl -sf -X DELETE \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Workflow/${workflow_name}" 2>/dev/null
}

################################################################################
# Workflow Creation Helpers
################################################################################

erpnext::workflow::create_sample() {
    local session_id="${1}"

    # Create a sample purchase order approval workflow
    local workflow_json='
    {
        "doctype": "Workflow",
        "workflow_name": "Purchase Order Approval",
        "document_type": "Purchase Order",
        "is_active": 1,
        "override_status": 0,
        "send_email_alert": 0,
        "conditions": "",
        "workflow_state_field": "workflow_state",
        "states": [
            {
                "state": "Draft",
                "doc_status": "0",
                "allow_edit": "Purchase User",
                "update_field": "status",
                "update_value": "Draft"
            },
            {
                "state": "Pending Approval",
                "doc_status": "0",
                "allow_edit": "Purchase Manager",
                "update_field": "status",
                "update_value": "To Receive and Bill"
            },
            {
                "state": "Approved",
                "doc_status": "1",
                "allow_edit": "Purchase Manager",
                "update_field": "status",
                "update_value": "To Receive and Bill"
            },
            {
                "state": "Rejected",
                "doc_status": "2",
                "allow_edit": "Purchase Manager",
                "update_field": "status",
                "update_value": "Cancelled"
            }
        ],
        "transitions": [
            {
                "state": "Draft",
                "action": "Submit for Approval",
                "next_state": "Pending Approval",
                "allowed": "Purchase User",
                "allow_self_approval": 1,
                "condition": ""
            },
            {
                "state": "Pending Approval",
                "action": "Approve",
                "next_state": "Approved",
                "allowed": "Purchase Manager",
                "allow_self_approval": 0,
                "condition": ""
            },
            {
                "state": "Pending Approval",
                "action": "Reject",
                "next_state": "Rejected",
                "allowed": "Purchase Manager",
                "allow_self_approval": 0,
                "condition": ""
            }
        ]
    }
    '

    erpnext::workflow::create "$workflow_json" "$session_id"
}

################################################################################
# Workflow State Functions
################################################################################

erpnext::workflow::get_states() {
    local workflow_name="${1}"
    local session_id="${2}"

    # Get workflow details and extract states
    local workflow_data
    workflow_data=$(erpnext::workflow::get "$workflow_name" "$session_id")

    if [[ -n "$workflow_data" ]]; then
        echo "$workflow_data" | jq '.data.states[] | {state: .state, doc_status: .doc_status}' 2>/dev/null
    fi
}

erpnext::workflow::get_transitions() {
    local workflow_name="${1}"
    local session_id="${2}"

    # Get workflow details and extract transitions
    local workflow_data
    workflow_data=$(erpnext::workflow::get "$workflow_name" "$session_id")

    if [[ -n "$workflow_data" ]]; then
        echo "$workflow_data" | jq '.data.transitions[] | {state: .state, action: .action, next_state: .next_state, allowed: .allowed}' 2>/dev/null
    fi
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
        local count=$(echo "$result" | jq '.data | length' 2>/dev/null || echo "0")
        if [[ "$count" -eq "0" ]]; then
            log::info "No workflows found. Creating sample workflow..."
            local sample_result
            sample_result=$(erpnext::workflow::create_sample "$session_id")
            if [[ -n "$sample_result" ]]; then
                log::success "Sample workflow created. Listing workflows..."
                result=$(erpnext::workflow::list "$session_id")
            fi
        fi
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Unable to retrieve workflows"
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

    if [[ -z "$workflow_file" ]]; then
        log::info "No file provided. Creating sample Purchase Order Approval workflow..."

        # Get session
        local session_id
        session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

        if [[ -z "$session_id" ]]; then
            log::error "Failed to authenticate"
            return 1
        fi

        # Create sample workflow
        local result
        result=$(erpnext::workflow::create_sample "$session_id")

        if [[ -n "$result" ]]; then
            log::success "Sample workflow created successfully"
            echo "$result" | jq '.' 2>/dev/null || echo "$result"
        else
            log::error "Failed to create sample workflow"
        fi

        # Logout
        erpnext::api::logout "$session_id"
        return 0
    fi

    if [[ ! -f "$workflow_file" ]]; then
        log::error "File not found: $workflow_file"
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

erpnext::workflow::cli::delete() {
    local workflow_name="${1:-}"

    if [[ -z "$workflow_name" ]]; then
        log::error "Workflow name required. Usage: workflow delete <name>"
        return 1
    fi

    log::info "Deleting workflow: $workflow_name"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Delete workflow
    local result
    result=$(erpnext::workflow::delete "$workflow_name" "$session_id")

    if [[ -n "$result" ]]; then
        log::success "Workflow deleted successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to delete workflow"
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
export -f erpnext::workflow::update
export -f erpnext::workflow::delete
export -f erpnext::workflow::create_sample
export -f erpnext::workflow::get_states
export -f erpnext::workflow::get_transitions
export -f erpnext::workflow::cli::list
export -f erpnext::workflow::cli::get
export -f erpnext::workflow::cli::create
export -f erpnext::workflow::cli::delete