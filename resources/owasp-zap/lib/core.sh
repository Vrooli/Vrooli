#!/usr/bin/env bash
################################################################################
# OWASP ZAP Resource - Core Library Functions
################################################################################

# Source configuration
source "${ZAP_CLI_DIR}/config/defaults.sh"

################################################################################
# Installation Functions
################################################################################

zap::install() {
    local force="${1:-false}"
    
    log_info "Installing OWASP ZAP Security Scanner..."
    
    # Check if Docker is available
    if ! command -v docker &> /dev/null; then
        log_error "Docker is required but not installed"
        return 1
    fi
    
    # Pull ZAP Docker image
    log_info "Pulling ZAP Docker image: ${ZAP_IMAGE}"
    if ! docker pull "${ZAP_IMAGE}"; then
        log_error "Failed to pull ZAP Docker image"
        return 1
    fi
    
    # Create data directories
    mkdir -p "${ZAP_DATA_DIR}" "${ZAP_SESSION_DIR}" "${ZAP_REPORT_DIR}" "${ZAP_POLICY_DIR}"
    
    # Generate API key if not set
    if [[ -z "${ZAP_API_KEY}" ]] && [[ "${ZAP_DISABLE_API_KEY}" != "true" ]]; then
        ZAP_API_KEY=$(openssl rand -hex 32 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
        echo "export ZAP_API_KEY='${ZAP_API_KEY}'" > "${ZAP_DATA_DIR}/api_key.sh"
        chmod 600 "${ZAP_DATA_DIR}/api_key.sh"
        log_info "Generated API key and saved to ${ZAP_DATA_DIR}/api_key.sh"
    fi
    
    log_success "OWASP ZAP installed successfully"
    return 0
}

zap::uninstall() {
    local keep_data="${1:-false}"
    
    log_info "Uninstalling OWASP ZAP..."
    
    # Stop if running
    zap::stop
    
    # Remove container if exists
    if docker ps -a --format '{{.Names}}' | grep -q "^${ZAP_CONTAINER_NAME}$"; then
        docker rm -f "${ZAP_CONTAINER_NAME}" &>/dev/null
    fi
    
    # Optionally remove data
    if [[ "${keep_data}" != "true" ]]; then
        log_info "Removing ZAP data directory..."
        rm -rf "${ZAP_DATA_DIR}"
    else
        log_info "Keeping ZAP data directory: ${ZAP_DATA_DIR}"
    fi
    
    log_success "OWASP ZAP uninstalled"
    return 0
}

################################################################################
# Lifecycle Functions
################################################################################

zap::start() {
    local wait="${1:-false}"
    
    log_info "Starting OWASP ZAP daemon..."
    
    # Load API key if exists
    if [[ -f "${ZAP_DATA_DIR}/api_key.sh" ]]; then
        source "${ZAP_DATA_DIR}/api_key.sh"
    fi
    
    # Check if already running
    if zap::is_running; then
        log_warn "OWASP ZAP is already running"
        return 0
    fi
    
    # Prepare Docker run command
    local docker_cmd="docker run -d --name ${ZAP_CONTAINER_NAME}"
    docker_cmd+=" -u zap"
    docker_cmd+=" -p ${ZAP_API_PORT}:${ZAP_API_PORT}"
    docker_cmd+=" -p ${ZAP_PROXY_PORT}:${ZAP_PROXY_PORT}"
    docker_cmd+=" -v ${ZAP_DATA_DIR}:/zap/data:rw"
    docker_cmd+=" -e ZAP_PORT=${ZAP_API_PORT}"
    docker_cmd+=" --memory=${ZAP_MEMORY}"
    docker_cmd+=" ${ZAP_IMAGE}"
    docker_cmd+=" zap.sh -daemon"
    docker_cmd+=" -port ${ZAP_API_PORT}"
    docker_cmd+=" -host ${ZAP_HOST}"
    
    # Configure API key
    if [[ "${ZAP_DISABLE_API_KEY}" == "true" ]]; then
        docker_cmd+=" -config api.disablekey=true"
    elif [[ -n "${ZAP_API_KEY}" ]]; then
        docker_cmd+=" -config api.key=${ZAP_API_KEY}"
    fi
    
    # Add JVM memory settings
    docker_cmd+=" -config jvm.heap=${ZAP_JVM_HEAP}"
    
    # Execute Docker run
    if ! eval "${docker_cmd}"; then
        log_error "Failed to start OWASP ZAP container"
        return 1
    fi
    
    # Wait for startup if requested
    if [[ "${wait}" == "true" ]]; then
        log_info "Waiting for ZAP to be ready..."
        local count=0
        while [[ ${count} -lt ${ZAP_STARTUP_TIMEOUT} ]]; do
            if zap::health_check; then
                log_success "OWASP ZAP is ready"
                return 0
            fi
            sleep 1
            ((count++))
        done
        log_error "ZAP failed to start within ${ZAP_STARTUP_TIMEOUT} seconds"
        return 1
    fi
    
    log_success "OWASP ZAP daemon started"
    return 0
}

zap::stop() {
    log_info "Stopping OWASP ZAP..."
    
    if ! zap::is_running; then
        log_info "OWASP ZAP is not running"
        return 0
    fi
    
    # Stop container
    if docker stop -t "${ZAP_SHUTDOWN_TIMEOUT}" "${ZAP_CONTAINER_NAME}" &>/dev/null; then
        docker rm "${ZAP_CONTAINER_NAME}" &>/dev/null
        log_success "OWASP ZAP stopped"
        return 0
    else
        log_error "Failed to stop OWASP ZAP"
        return 1
    fi
}

zap::restart() {
    log_info "Restarting OWASP ZAP..."
    zap::stop
    sleep 2
    zap::start true
}

################################################################################
# Status Functions
################################################################################

zap::status() {
    local format="${1:-text}"
    
    if [[ "${format}" == "json" ]]; then
        zap::status_json
    else
        zap::status_text
    fi
}

zap::status_text() {
    echo "OWASP ZAP Security Scanner Status"
    echo "================================="
    
    if zap::is_running; then
        echo "Status: Running"
        echo "API Port: ${ZAP_API_PORT}"
        echo "Proxy Port: ${ZAP_PROXY_PORT}"
        
        # Get version if available
        local version=$(zap::get_version 2>/dev/null)
        if [[ -n "${version}" ]]; then
            echo "Version: ${version}"
        fi
        
        # Check for active scans
        local scan_status=$(zap::get_scan_status 2>/dev/null)
        if [[ -n "${scan_status}" ]]; then
            echo "Active Scans: ${scan_status}"
        fi
    else
        echo "Status: Not Running"
    fi
}

zap::status_json() {
    local running="false"
    local version=""
    local scan_status=""
    
    if zap::is_running; then
        running="true"
        version=$(zap::get_version 2>/dev/null || echo "unknown")
        scan_status=$(zap::get_scan_status 2>/dev/null || echo "0")
    fi
    
    cat <<EOF
{
  "running": ${running},
  "api_port": ${ZAP_API_PORT},
  "proxy_port": ${ZAP_PROXY_PORT},
  "version": "${version}",
  "active_scans": "${scan_status}"
}
EOF
}

################################################################################
# Health Check Functions
################################################################################

zap::health_check() {
    # Check if container is running
    if ! zap::is_running; then
        return 1
    fi
    
    # Check API endpoint
    local api_url="http://localhost:${ZAP_API_PORT}/JSON/core/view/version"
    if [[ -n "${ZAP_API_KEY}" ]] && [[ "${ZAP_DISABLE_API_KEY}" != "true" ]]; then
        api_url+="?apikey=${ZAP_API_KEY}"
    fi
    
    if timeout "${ZAP_HEALTH_CHECK_TIMEOUT}" curl -sf "${api_url}" &>/dev/null; then
        return 0
    else
        return 1
    fi
}

zap::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${ZAP_CONTAINER_NAME}$"
}

################################################################################
# API Helper Functions
################################################################################

zap::api_request() {
    local endpoint="$1"
    local method="${2:-GET}"
    local data="${3:-}"
    
    local api_url="http://localhost:${ZAP_API_PORT}${endpoint}"
    
    # Add API key if needed
    if [[ -n "${ZAP_API_KEY}" ]] && [[ "${ZAP_DISABLE_API_KEY}" != "true" ]]; then
        if [[ "${api_url}" == *"?"* ]]; then
            api_url+="&apikey=${ZAP_API_KEY}"
        else
            api_url+="?apikey=${ZAP_API_KEY}"
        fi
    fi
    
    local curl_cmd="curl -sf"
    if [[ "${method}" == "POST" ]]; then
        curl_cmd+=" -X POST"
        if [[ -n "${data}" ]]; then
            curl_cmd+=" -d '${data}'"
        fi
    fi
    
    eval "${curl_cmd} '${api_url}'"
}

zap::get_version() {
    local response=$(zap::api_request "/JSON/core/view/version")
    echo "${response}" | jq -r '.version' 2>/dev/null
}

zap::get_scan_status() {
    local response=$(zap::api_request "/JSON/ascan/view/scans")
    echo "${response}" | jq -r '.scans | length' 2>/dev/null
}

################################################################################
# Content Management Functions
################################################################################

zap::content_add() {
    local target="$1"
    local type="${2:-url}"
    
    log_info "Adding scan target: ${target}"
    
    # Add target to ZAP context
    local response=$(zap::api_request "/JSON/core/action/accessUrl?url=${target}" "POST")
    
    if [[ $? -eq 0 ]]; then
        log_success "Target added successfully"
        return 0
    else
        log_error "Failed to add target"
        return 1
    fi
}

zap::content_execute() {
    local target="$1"
    local scan_type="${2:-spider}"
    
    log_info "Starting ${scan_type} scan on ${target}..."
    
    case "${scan_type}" in
        spider)
            zap::api_request "/JSON/spider/action/scan?url=${target}" "POST"
            ;;
        active)
            zap::api_request "/JSON/ascan/action/scan?url=${target}" "POST"
            ;;
        api)
            zap::api_request "/JSON/openapi/action/importUrl?url=${target}" "POST"
            ;;
        *)
            log_error "Unknown scan type: ${scan_type}"
            return 1
            ;;
    esac
    
    if [[ $? -eq 0 ]]; then
        log_success "Scan started successfully"
        return 0
    else
        log_error "Failed to start scan"
        return 1
    fi
}

zap::content_list() {
    local format="${1:-json}"
    
    log_info "Retrieving scan results..."
    
    local alerts=$(zap::api_request "/JSON/core/view/alerts")
    
    if [[ "${format}" == "json" ]]; then
        echo "${alerts}"
    else
        echo "${alerts}" | jq -r '.alerts[] | "\(.risk) - \(.alert) at \(.url)"' 2>/dev/null
    fi
}

zap::content_get() {
    local format="${1:-json}"
    local output_file="${2:-}"
    
    log_info "Generating report in ${format} format..."
    
    local report_data
    case "${format}" in
        json)
            report_data=$(zap::api_request "/JSON/core/view/alerts")
            ;;
        html)
            report_data=$(zap::api_request "/OTHER/core/other/htmlreport")
            ;;
        xml)
            report_data=$(zap::api_request "/OTHER/core/other/xmlreport")
            ;;
        *)
            log_error "Unsupported format: ${format}"
            return 1
            ;;
    esac
    
    if [[ -n "${output_file}" ]]; then
        echo "${report_data}" > "${output_file}"
        log_success "Report saved to ${output_file}"
    else
        echo "${report_data}"
    fi
    
    return 0
}

################################################################################
# Export Functions
################################################################################

export -f zap::install
export -f zap::uninstall
export -f zap::start
export -f zap::stop
export -f zap::restart
export -f zap::status
export -f zap::health_check
export -f zap::is_running
export -f zap::content_add
export -f zap::content_execute
export -f zap::content_list
export -f zap::content_get