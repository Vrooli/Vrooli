#!/usr/bin/env bash

#######################################
# N8N UI Selectors Library
#
# Centralized collection of CSS selectors for N8N UI automation
# This file provides reliable selectors for various N8N interface elements
#######################################

#######################################
# N8N Page Selectors
#######################################

# Main navigation
declare -r N8N_NAV_WORKFLOWS="[data-test-id=\"main-nav-workflows\"]"
declare -r N8N_NAV_CREDENTIALS="[data-test-id=\"main-nav-credentials\"]"
declare -r N8N_NAV_EXECUTIONS="[data-test-id=\"main-nav-executions\"]"
declare -r N8N_NAV_SETTINGS="[data-test-id=\"main-nav-settings\"]"

# Workflow page selectors
declare -r N8N_WORKFLOW_LIST="[data-test-id=\"workflows-list\"]"
declare -r N8N_WORKFLOW_ITEM="[data-test-id=\"workflow-card\"]"
declare -r N8N_WORKFLOW_NAME="[data-test-id=\"workflow-name\"]"
declare -r N8N_WORKFLOW_ACTIVE_TOGGLE="[data-test-id=\"workflow-activate-switch\"]"
declare -r N8N_WORKFLOW_EXECUTE_BUTTON="[data-test-id=\"workflow-execute-btn\"]"
declare -r N8N_WORKFLOW_SAVE_BUTTON="[data-test-id=\"workflow-save-btn\"]"

# Node selectors
declare -r N8N_NODE_ADD_BUTTON="[data-test-id=\"canvas-add-button\"]"
declare -r N8N_NODE_SEARCH="[data-test-id=\"node-creator-search\"]"
declare -r N8N_NODE_ITEM="[data-test-id=\"node-creator-item\"]"

# Credentials page selectors  
declare -r N8N_CREDENTIALS_LIST="[data-test-id=\"credentials-list\"]"
declare -r N8N_CREDENTIALS_ADD_BUTTON="[data-test-id=\"resources-list-add\"]"
declare -r N8N_CREDENTIALS_SEARCH="[data-test-id=\"credential-search\"]"
declare -r N8N_CREDENTIALS_ITEM="[data-test-id=\"credential-card\"]"
declare -r N8N_CREDENTIALS_NAME="[data-test-id=\"credential-name\"]"
declare -r N8N_CREDENTIALS_SAVE_BUTTON="[data-test-id=\"credential-save-button\"]"
declare -r N8N_CREDENTIALS_TEST_BUTTON="[data-test-id=\"credential-test-button\"]"

# Execution page selectors
declare -r N8N_EXECUTION_LIST="[data-test-id=\"executions-list\"]"
declare -r N8N_EXECUTION_ITEM="[data-test-id=\"execution-item\"]"
declare -r N8N_EXECUTION_STATUS="[data-test-id=\"execution-status\"]"
declare -r N8N_EXECUTION_TIME="[data-test-id=\"execution-time\"]"

# Form and input selectors
declare -r N8N_INPUT_TEXT="input[type=\"text\"], .el-input__inner"
declare -r N8N_INPUT_PASSWORD="input[type=\"password\"]"
declare -r N8N_BUTTON_PRIMARY=".btn-primary, .el-button--primary"
declare -r N8N_BUTTON_SECONDARY=".btn-secondary, .el-button--default"
declare -r N8N_DROPDOWN=".el-select, .dropdown-menu"
declare -r N8N_MODAL=".el-dialog, .modal"
declare -r N8N_NOTIFICATION=".el-notification, .notification"

# Canvas and editor selectors
declare -r N8N_CANVAS="[data-test-id=\"workflow-canvas\"]"
declare -r N8N_NODE_CANVAS_ITEM=".node-default, .node"
declare -r N8N_CONNECTION_LINE=".connection-line, .edge"

# Loading and status selectors
declare -r N8N_LOADING_SPINNER=".el-loading-spinner, .spinner"
declare -r N8N_ERROR_MESSAGE=".el-message--error, .error"
declare -r N8N_SUCCESS_MESSAGE=".el-message--success, .success"

#######################################
# Selector Helper Functions
#######################################

#######################################
# Get workflow selector by name
# Arguments:
#   $1 - Workflow name
# Returns:
#   CSS selector for specific workflow
#######################################
n8n::selector::workflow_by_name() {
    local workflow_name="${1:?Workflow name required}"
    echo "${N8N_WORKFLOW_ITEM}:has([title=\"${workflow_name}\"])"
}

#######################################
# Get credential selector by name
# Arguments:
#   $1 - Credential name
# Returns:
#   CSS selector for specific credential
#######################################
n8n::selector::credential_by_name() {
    local credential_name="${1:?Credential name required}"
    echo "${N8N_CREDENTIALS_ITEM}:has([title=\"${credential_name}\"])"
}

#######################################
# Get node selector by type
# Arguments:
#   $1 - Node type (e.g., "HTTP Request", "Set", etc.)
# Returns:
#   CSS selector for specific node type
#######################################
n8n::selector::node_by_type() {
    local node_type="${1:?Node type required}"
    echo "${N8N_NODE_ITEM}[data-node-type=\"${node_type}\"]"
}

#######################################
# Wait for element with timeout
# Arguments:
#   $1 - CSS selector
#   $2 - Timeout in milliseconds (default: 10000)
# Returns:
#   0 if element found, 1 if timeout
#######################################
n8n::selector::wait_for_element() {
    local selector="${1:?CSS selector required}"
    local timeout="${2:-10000}"
    
    # This would be used with browserless page automation
    # For now, just return the selector for use in automation scripts
    echo "await page.waitForSelector('${selector}', { timeout: ${timeout} })"
}

#######################################
# Build compound selector for complex UI elements
# Arguments:
#   $1 - Base selector
#   $2 - Child selector
# Returns:
#   Combined CSS selector
#######################################
n8n::selector::combine() {
    local base="${1:?Base selector required}"
    local child="${2:?Child selector required}"
    echo "${base} ${child}"
}

# Export all selector constants
export N8N_NAV_WORKFLOWS N8N_NAV_CREDENTIALS N8N_NAV_EXECUTIONS N8N_NAV_SETTINGS
export N8N_WORKFLOW_LIST N8N_WORKFLOW_ITEM N8N_WORKFLOW_NAME N8N_WORKFLOW_ACTIVE_TOGGLE
export N8N_WORKFLOW_EXECUTE_BUTTON N8N_WORKFLOW_SAVE_BUTTON
export N8N_NODE_ADD_BUTTON N8N_NODE_SEARCH N8N_NODE_ITEM
export N8N_CREDENTIALS_LIST N8N_CREDENTIALS_ADD_BUTTON N8N_CREDENTIALS_SEARCH
export N8N_CREDENTIALS_ITEM N8N_CREDENTIALS_NAME N8N_CREDENTIALS_SAVE_BUTTON N8N_CREDENTIALS_TEST_BUTTON
export N8N_EXECUTION_LIST N8N_EXECUTION_ITEM N8N_EXECUTION_STATUS N8N_EXECUTION_TIME
export N8N_INPUT_TEXT N8N_INPUT_PASSWORD N8N_BUTTON_PRIMARY N8N_BUTTON_SECONDARY
export N8N_DROPDOWN N8N_MODAL N8N_NOTIFICATION
export N8N_CANVAS N8N_NODE_CANVAS_ITEM N8N_CONNECTION_LINE
export N8N_LOADING_SPINNER N8N_ERROR_MESSAGE N8N_SUCCESS_MESSAGE

# Export helper functions
export -f n8n::selector::workflow_by_name
export -f n8n::selector::credential_by_name
export -f n8n::selector::node_by_type
export -f n8n::selector::wait_for_element
export -f n8n::selector::combine