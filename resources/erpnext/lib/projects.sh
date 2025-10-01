#!/usr/bin/env bash
################################################################################
# ERPNext Project Management Module
#
# Provides comprehensive project management capabilities
################################################################################

set -euo pipefail

# Source dependencies
source "${APP_ROOT}/resources/erpnext/lib/api.sh"
source "${APP_ROOT}/resources/erpnext/config/defaults.sh"

################################################################################
# Project Management Functions
################################################################################

erpnext::projects::list() {
    local session_id="${1}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # List all projects
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Project?fields=[\"name\",\"project_name\",\"status\",\"expected_start_date\",\"expected_end_date\",\"percent_complete\"]&limit_page_length=100" 2>/dev/null
}

erpnext::projects::create() {
    local project_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Create new project
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${project_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Project" 2>/dev/null
}

erpnext::projects::get() {
    local project_name="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Get project details
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Project/${project_name}" 2>/dev/null
}

erpnext::projects::update() {
    local project_name="${1}"
    local project_json="${2}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Update project
    timeout 5 curl -sf -X PUT \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${project_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Project/${project_name}" 2>/dev/null
}

################################################################################
# Task Management Functions
################################################################################

erpnext::projects::list_tasks() {
    local project="${1:-}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Build filter for project if specified
    local filters=""
    if [[ -n "$project" ]]; then
        filters="filters=[[\"project\",\"=\",\"${project}\"]]&"
    fi

    # List tasks
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Task?${filters}fields=[\"name\",\"subject\",\"project\",\"status\",\"priority\",\"exp_start_date\",\"exp_end_date\",\"progress\"]&limit_page_length=100" 2>/dev/null
}

erpnext::projects::create_task() {
    local task_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Create new task
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${task_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Task" 2>/dev/null
}

erpnext::projects::update_task() {
    local task_name="${1}"
    local task_json="${2}"
    local session_id="${3}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Update task
    timeout 5 curl -sf -X PUT \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${task_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Task/${task_name}" 2>/dev/null
}

################################################################################
# Timesheet Functions
################################################################################

erpnext::projects::create_timesheet() {
    local timesheet_json="${1}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Create timesheet entry
    timeout 5 curl -sf -X POST \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "${timesheet_json}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Timesheet" 2>/dev/null
}

erpnext::projects::list_timesheets() {
    local project="${1:-}"
    local session_id="${2}"
    local site_name="${ERPNEXT_SITE_NAME:-vrooli.local}"

    if [[ -z "$session_id" ]]; then
        log::error "Session ID required"
        return 1
    fi

    # Build filter for project if specified
    local filters=""
    if [[ -n "$project" ]]; then
        filters="filters=[[\"project\",\"=\",\"${project}\"]]&"
    fi

    # List timesheets
    timeout 5 curl -sf \
        -H "Host: ${site_name}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Timesheet?${filters}fields=[\"name\",\"employee\",\"start_date\",\"end_date\",\"total_hours\",\"status\"]&limit_page_length=100" 2>/dev/null
}

################################################################################
# CLI Wrapper Functions
################################################################################

erpnext::projects::cli::list() {
    log::info "Listing projects..."

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # List projects
    local result
    result=$(erpnext::projects::list "$session_id")

    if [[ -n "$result" ]]; then
        local count=$(echo "$result" | jq '.data | length' 2>/dev/null || echo "0")
        if [[ "$count" -eq "0" ]]; then
            log::info "No projects found. Creating sample project..."
            # Create sample project
            local project_json='{
                "doctype": "Project",
                "project_name": "Website Redesign",
                "expected_start_date": "'$(date +%Y-%m-%d)'",
                "expected_end_date": "'$(date -d '+90 days' +%Y-%m-%d)'",
                "status": "Open",
                "priority": "High",
                "percent_complete_method": "Task Completion",
                "description": "Complete website redesign project"
            }'

            erpnext::projects::create "$project_json" "$session_id" &>/dev/null
            result=$(erpnext::projects::list "$session_id")
        fi
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Unable to retrieve projects"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::projects::cli::create() {
    local project_name="${1:-}"
    local start_date="${2:-$(date +%Y-%m-%d)}"
    local end_date="${3:-$(date -d '+30 days' +%Y-%m-%d)}"

    if [[ -z "$project_name" ]]; then
        log::error "Project name required. Usage: projects create <name> [start_date] [end_date]"
        return 1
    fi

    log::info "Creating project: $project_name"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Create project
    local project_json="{
        \"doctype\": \"Project\",
        \"project_name\": \"${project_name}\",
        \"expected_start_date\": \"${start_date}\",
        \"expected_end_date\": \"${end_date}\",
        \"status\": \"Open\",
        \"priority\": \"Medium\",
        \"percent_complete_method\": \"Task Completion\"
    }"

    local result
    result=$(erpnext::projects::create "$project_json" "$session_id")

    if [[ -n "$result" ]]; then
        log::success "Project created successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to create project"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::projects::cli::add_task() {
    local project="${1:-}"
    local task_subject="${2:-}"
    local priority="${3:-Medium}"

    if [[ -z "$project" ]] || [[ -z "$task_subject" ]]; then
        log::error "Project and task subject required. Usage: projects add-task <project> <subject> [priority]"
        return 1
    fi

    log::info "Adding task to project: $project"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Create task
    local task_json="{
        \"doctype\": \"Task\",
        \"subject\": \"${task_subject}\",
        \"project\": \"${project}\",
        \"status\": \"Open\",
        \"priority\": \"${priority}\",
        \"exp_start_date\": \"$(date +%Y-%m-%d)\",
        \"exp_end_date\": \"$(date -d '+7 days' +%Y-%m-%d)\"
    }"

    local result
    result=$(erpnext::projects::create_task "$task_json" "$session_id")

    if [[ -n "$result" ]]; then
        log::success "Task created successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to create task"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::projects::cli::list_tasks() {
    local project="${1:-}"

    if [[ -n "$project" ]]; then
        log::info "Listing tasks for project: $project"
    else
        log::info "Listing all tasks..."
    fi

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # List tasks
    local result
    result=$(erpnext::projects::list_tasks "$project" "$session_id")

    if [[ -n "$result" ]]; then
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::info "No tasks found"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::projects::cli::update_progress() {
    local project="${1:-}"
    local progress="${2:-}"

    if [[ -z "$project" ]] || [[ -z "$progress" ]]; then
        log::error "Project and progress required. Usage: projects update-progress <project> <percent>"
        return 1
    fi

    log::info "Updating project progress: $project to $progress%"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Update project progress
    local update_json="{\"percent_complete\": ${progress}}"

    local result
    result=$(erpnext::projects::update "$project" "$update_json" "$session_id")

    if [[ -n "$result" ]]; then
        log::success "Project progress updated"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to update project progress"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

erpnext::projects::cli::log_time() {
    local project="${1:-}"
    local hours="${2:-}"
    local activity="${3:-Development}"

    if [[ -z "$project" ]] || [[ -z "$hours" ]]; then
        log::error "Project and hours required. Usage: projects log-time <project> <hours> [activity]"
        return 1
    fi

    log::info "Logging $hours hours for project: $project"

    # Get session
    local session_id
    session_id=$(erpnext::api::login "Administrator" "${ERPNEXT_ADMIN_PASSWORD:-admin}")

    if [[ -z "$session_id" ]]; then
        log::error "Failed to authenticate"
        return 1
    fi

    # Create timesheet
    local timesheet_json="{
        \"doctype\": \"Timesheet\",
        \"time_logs\": [{
            \"project\": \"${project}\",
            \"hours\": ${hours},
            \"activity_type\": \"${activity}\",
            \"from_time\": \"$(date '+%Y-%m-%d 09:00:00')\",
            \"to_time\": \"$(date '+%Y-%m-%d %H:%M:%S')\"
        }]
    }"

    local result
    result=$(erpnext::projects::create_timesheet "$timesheet_json" "$session_id")

    if [[ -n "$result" ]]; then
        log::success "Time logged successfully"
        echo "$result" | jq '.' 2>/dev/null || echo "$result"
    else
        log::error "Failed to log time"
    fi

    # Logout
    erpnext::api::logout "$session_id"
}

################################################################################
# Export Functions
################################################################################

export -f erpnext::projects::list
export -f erpnext::projects::create
export -f erpnext::projects::get
export -f erpnext::projects::update
export -f erpnext::projects::list_tasks
export -f erpnext::projects::create_task
export -f erpnext::projects::update_task
export -f erpnext::projects::create_timesheet
export -f erpnext::projects::list_timesheets
export -f erpnext::projects::cli::list
export -f erpnext::projects::cli::create
export -f erpnext::projects::cli::add_task
export -f erpnext::projects::cli::list_tasks
export -f erpnext::projects::cli::update_progress
export -f erpnext::projects::cli::log_time