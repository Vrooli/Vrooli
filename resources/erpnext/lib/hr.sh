#!/usr/bin/env bash
################################################################################
# ERPNext HR Module Functions
# 
# Manages Human Resources operations
################################################################################

set -euo pipefail

# Determine base path for sourcing
ERPNEXT_LIB_DIR="${ERPNEXT_LIB_DIR:-$(builtin cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
ERPNEXT_BASE_DIR="${ERPNEXT_BASE_DIR:-$(builtin cd "${ERPNEXT_LIB_DIR}/.." && pwd)}"

# Define log functions if not already available
if ! declare -f log::error &>/dev/null; then
    log::error() { echo "[ERROR] $*" >&2; }
    log::info() { echo "[INFO] $*"; }
fi

# Source dependencies if not already loaded
if [[ -z "${ERPNEXT_PORT:-}" ]]; then
    source "${ERPNEXT_BASE_DIR}/config/defaults.sh"
fi

if [[ -f "${ERPNEXT_BASE_DIR}/lib/api.sh" ]]; then
    source "${ERPNEXT_BASE_DIR}/lib/api.sh"
fi

################################################################################
# Employee Management
################################################################################

erpnext::hr::list_employees() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Employee" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
employees = data.get('data', [])
if employees:
    for emp in employees:
        print(f\"Employee: {emp.get('employee_name', emp.get('name', 'N/A'))} - ID: {emp.get('name', 'N/A')} - Department: {emp.get('department', 'N/A')}\")
else:
    print('No employees found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve employees"
        return 1
    fi
}

erpnext::hr::create_employee() {
    local first_name="${1:-}"
    local last_name="${2:-}"
    local email="${3:-}"
    local department="${4:-}"
    local session_id="${5:-}"
    
    if [[ -z "$first_name" ]]; then
        log::error "First name is required"
        echo "Usage: resource-erpnext hr create-employee <first_name> <last_name> [email] [department]"
        return 1
    fi
    
    if [[ -z "$last_name" ]]; then
        log::error "Last name is required"
        echo "Usage: resource-erpnext hr create-employee <first_name> <last_name> [email] [department]"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    # Get current date for joining
    local date_of_joining=$(date +%Y-%m-%d)
    local employee_name="${first_name} ${last_name}"
    
    # Build JSON data
    local data="{
        \"doctype\":\"Employee\",
        \"first_name\":\"${first_name}\",
        \"last_name\":\"${last_name}\",
        \"employee_name\":\"${employee_name}\",
        \"date_of_joining\":\"${date_of_joining}\",
        \"status\":\"Active\""
    
    [[ -n "$email" ]] && data="${data},\"prefered_email\":\"${email}\",\"personal_email\":\"${email}\""
    [[ -n "$department" ]] && data="${data},\"department\":\"${department}\""
    data="${data}}"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Employee" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    emp = data['data']
    print(f\"✅ Employee created: {emp.get('employee_name', 'Unknown')} - ID: {emp.get('name', 'Unknown')}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to create employee')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to create employee"
        return 1
    fi
}

################################################################################
# Department Management
################################################################################

erpnext::hr::list_departments() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Department" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
departments = data.get('data', [])
if departments:
    for dept in departments:
        print(f\"Department: {dept.get('department_name', dept.get('name', 'N/A'))}\")
else:
    print('No departments found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve departments"
        return 1
    fi
}

################################################################################
# Leave Management
################################################################################

erpnext::hr::list_leave_applications() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Leave%20Application" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
applications = data.get('data', [])
if applications:
    for app in applications:
        print(f\"Leave Application: {app.get('name', 'N/A')} - Employee: {app.get('employee_name', 'N/A')} - Status: {app.get('status', 'N/A')}\")
else:
    print('No leave applications found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve leave applications"
        return 1
    fi
}

erpnext::hr::create_leave_application() {
    local employee="${1:-}"
    local from_date="${2:-}"
    local to_date="${3:-}"
    local leave_type="${4:-Casual Leave}"
    local session_id="${5:-}"
    
    if [[ -z "$employee" ]] || [[ -z "$from_date" ]] || [[ -z "$to_date" ]]; then
        log::error "Employee ID, from date, and to date are required"
        echo "Usage: resource-erpnext hr create-leave <employee_id> <from_date> <to_date> [leave_type]"
        echo "Date format: YYYY-MM-DD"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local data="{
        \"doctype\":\"Leave Application\",
        \"employee\":\"${employee}\",
        \"leave_type\":\"${leave_type}\",
        \"from_date\":\"${from_date}\",
        \"to_date\":\"${to_date}\",
        \"status\":\"Open\"
    }"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Leave%20Application" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    leave = data['data']
    print(f\"✅ Leave application created: {leave.get('name', 'Unknown')} - Status: {leave.get('status', 'Unknown')}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to create leave application')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to create leave application"
        return 1
    fi
}

################################################################################
# Attendance Management
################################################################################

erpnext::hr::list_attendance() {
    local date="${1:-$(date +%Y-%m-%d)}"
    local session_id="${2:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Attendance?filters=[[\"attendance_date\",\"=\",\"${date}\"]]" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "Attendance for $date:"
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
records = data.get('data', [])
if records:
    for record in records:
        print(f\"  Employee: {record.get('employee_name', 'N/A')} - Status: {record.get('status', 'N/A')}\")
else:
    print('  No attendance records found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve attendance"
        return 1
    fi
}

erpnext::hr::mark_attendance() {
    local employee="${1:-}"
    local status="${2:-Present}"  # Present or Absent
    local date="${3:-$(date +%Y-%m-%d)}"
    local session_id="${4:-}"
    
    if [[ -z "$employee" ]]; then
        log::error "Employee ID is required"
        echo "Usage: resource-erpnext hr mark-attendance <employee_id> [status] [date]"
        return 1
    fi
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local data="{
        \"doctype\":\"Attendance\",
        \"employee\":\"${employee}\",
        \"attendance_date\":\"${date}\",
        \"status\":\"${status}\"
    }"
    
    local response
    response=$(timeout 5 curl -sf -X POST \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        -H "Content-Type: application/json" \
        -d "$data" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Attendance" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
if 'data' in data:
    att = data['data']
    print(f\"✅ Attendance marked: {att.get('employee_name', 'Unknown')} - {att.get('status', 'Unknown')} on {att.get('attendance_date', 'Unknown')}\")
else:
    print(f\"Error: {data.get('exc', 'Failed to mark attendance')}\")
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to mark attendance"
        return 1
    fi
}

################################################################################
# Salary Structure Management
################################################################################

erpnext::hr::list_salary_structures() {
    local session_id="${1:-}"
    
    # Login if no session provided
    if [[ -z "$session_id" ]]; then
        session_id=$(erpnext::api::login 2>/dev/null) || {
            log::error "Authentication failed"
            return 1
        }
    fi
    
    local response
    response=$(timeout 5 curl -sf \
        -H "Host: ${ERPNEXT_SITE_NAME:-vrooli.local}" \
        -H "Cookie: sid=${session_id}" \
        "http://localhost:${ERPNEXT_PORT}/api/resource/Salary%20Structure" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | python3 -c "
import sys, json
data = json.load(sys.stdin)
structures = data.get('data', [])
if structures:
    for struct in structures:
        print(f\"Salary Structure: {struct.get('name', 'N/A')} - Active: {struct.get('is_active', 'N/A')}\")
else:
    print('No salary structures found')
" 2>/dev/null || echo "$response"
    else
        log::error "Failed to retrieve salary structures"
        return 1
    fi
}

################################################################################
# Export functions for use by other scripts
################################################################################

export -f erpnext::hr::list_employees
export -f erpnext::hr::create_employee
export -f erpnext::hr::list_departments
export -f erpnext::hr::list_leave_applications
export -f erpnext::hr::create_leave_application
export -f erpnext::hr::list_attendance
export -f erpnext::hr::mark_attendance
export -f erpnext::hr::list_salary_structures