#!/usr/bin/env bash
# MinIO Status Monitoring - Standardized Format
# Functions for checking MinIO health and status

# Source format utilities and required libraries
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MINIO_STATUS_DIR="${APP_ROOT}/resources/minio/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/minio/config/defaults.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/minio/config/messages.sh"

# Ensure configuration is exported
if command -v minio::export_config &>/dev/null; then
    minio::export_config 2>/dev/null || true
fi

# Source common functions after config is initialized
# shellcheck disable=SC1091
source "${MINIO_STATUS_DIR}/common.sh"

#######################################
# Collect MinIO status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
minio::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    local docker_available="false"
    
    # Check Docker
    if docker::check_daemon >/dev/null 2>&1; then
        docker_available="true"
    fi
    
    if minio::common::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "$MINIO_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        
        if minio::common::is_running; then
            running="true"
            
            if minio::common::health_check; then
                healthy="true"
                health_message="Healthy - All systems operational"
            else
                health_message="Unhealthy - API not responding"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container not found"
    fi
    
    # Basic resource information
    status_data+=("name" "minio")
    status_data+=("category" "storage")
    status_data+=("description" "S3-compatible object storage server")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$MINIO_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("docker_available" "$docker_available")
    
    # Service endpoints and ports
    status_data+=("api_port" "$MINIO_PORT")
    status_data+=("console_port" "$MINIO_CONSOLE_PORT")
    status_data+=("api_url" "$MINIO_BASE_URL")
    status_data+=("console_url" "$MINIO_CONSOLE_URL")
    status_data+=("health_endpoint" "http://localhost:$MINIO_PORT/minio/health/live")
    
    # Configuration details
    status_data+=("image" "$MINIO_IMAGE")
    status_data+=("version" "$MINIO_VERSION")
    status_data+=("data_dir" "$MINIO_DATA_DIR")
    
    # Check disk space
    local disk_space="unknown"
    local disk_available="false"
    if minio::common::check_disk_space >/dev/null 2>&1; then
        disk_available="true"
        local available_gb=$(df -BG "${MINIO_DATA_DIR}%/*" 2>/dev/null | awk 'NR==2 {print $4}' | sed 's/G//' || echo "unknown")
        disk_space="${available_gb}GB"
    fi
    status_data+=("disk_space" "$disk_space")
    status_data+=("disk_available" "$disk_available")
    
    # Check Vrooli configuration
    local vrooli_registered="false"
    if minio::status::check_vrooli_config; then
        vrooli_registered="true"
    fi
    status_data+=("vrooli_registered" "$vrooli_registered")
    
    # Port status (only if running, skip if fast mode)
    if [[ "$running" == "true" ]]; then
        local api_port_status="unknown"
        local console_port_status="unknown"
        
        if [[ "$fast_mode" == "true" ]]; then
            api_port_status="N/A"
            console_port_status="N/A"
        elif [[ "$healthy" == "true" ]]; then
            # When healthy, ports are definitely in use by MinIO
            api_port_status="in_use"
            console_port_status="in_use"
        else
            # Otherwise check if ports are available
            if ! minio::common::is_port_available "${MINIO_PORT}"; then
                api_port_status="in_use"
            else
                api_port_status="available"
            fi
            
            if ! minio::common::is_port_available "${MINIO_CONSOLE_PORT}"; then
                console_port_status="in_use"
            else
                console_port_status="available"
            fi
        fi
        
        status_data+=("api_port_status" "$api_port_status")
        status_data+=("console_port_status" "$console_port_status")
        
        # Runtime information (only if healthy, skip if fast mode)
        if [[ "$healthy" == "true" ]]; then
            # Get storage usage
            local storage_used="unknown"
            if [[ "$fast_mode" == "false" && -d "$MINIO_DATA_DIR" ]]; then
                storage_used=$(timeout 3s du -sh "$MINIO_DATA_DIR" 2>/dev/null | cut -f1 || echo "unknown")
            elif [[ "$fast_mode" == "true" ]]; then
                storage_used="N/A"
            fi
            status_data+=("storage_used" "$storage_used")
            
            # Check credentials
            local has_credentials="false"
            if [[ -n "${MINIO_ROOT_USER:-}" && -n "${MINIO_ROOT_PASSWORD:-}" ]]; then
                has_credentials="true"
            fi
            status_data+=("has_credentials" "$has_credentials")
            status_data+=("root_user" "${MINIO_ROOT_USER:-unknown}")
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}


#######################################
# Display status in text format
#######################################
minio::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 MinIO Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[docker_available]:-false}" == "true" ]]; then
        log::success "   ✅ Docker: Available"
    else
        log::error "   ❌ Docker: Not available"
    fi
    
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install MinIO, run: resource-minio manage install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    log::info "🐳 Container Info:"
    log::info "   📦 Name: ${data[container_name]:-unknown}"
    log::info "   📊 Status: ${data[container_status]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    log::info "   🔖 Version: ${data[version]:-unknown}"
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔌 API URL: ${data[api_url]:-unknown}"
    log::info "   🎨 Console URL: ${data[console_url]:-unknown}"
    log::info "   🏥 Health Check: ${data[health_endpoint]:-unknown}"
    echo
    
    # Port status
    log::info "📶 Port Status:"
    log::info "   🔌 API Port: ${data[api_port]:-unknown}"
    if [[ "${data[api_port_status]:-unknown}" == "in_use" ]]; then
        log::success "      Status: In use by MinIO"
    elif [[ "${data[api_port_status]:-unknown}" == "available" ]]; then
        log::warn "      Status: Available (not listening)"
    else
        log::info "      Status: ${data[api_port_status]:-unknown}"
    fi
    
    log::info "   🎨 Console Port: ${data[console_port]:-unknown}"
    if [[ "${data[console_port_status]:-unknown}" == "in_use" ]]; then
        log::success "      Status: In use by MinIO"
    elif [[ "${data[console_port_status]:-unknown}" == "available" ]]; then
        log::warn "      Status: Available (not listening)"
    else
        log::info "      Status: ${data[console_port_status]:-unknown}"
    fi
    echo
    
    # Storage information
    log::info "💾 Storage Info:"
    log::info "   📁 Data Directory: ${data[data_dir]:-unknown}"
    if [[ "${data[disk_available]:-false}" == "true" ]]; then
        log::success "   💿 Disk Space: ${data[disk_space]:-unknown} available"
    else
        log::error "   💿 Disk Space: Insufficient"
    fi
    
    if [[ "${data[running]:-false}" == "true" && "${data[healthy]:-false}" == "true" ]]; then
        log::info "   📊 Used Space: ${data[storage_used]:-unknown}"
    fi
    echo
    
    # Configuration status
    log::info "⚙️  Configuration:"
    if [[ "${data[vrooli_registered]:-false}" == "true" ]]; then
        log::success "   ✅ Vrooli Config: Registered"
    else
        log::warn "   ⚠️  Vrooli Config: Not registered"
    fi
    
    if [[ "${data[has_credentials]:-false}" == "true" ]]; then
        log::success "   ✅ Credentials: Configured"
        log::info "   👤 Root User: ${data[root_user]:-unknown}"
    else
        log::warn "   ⚠️  Credentials: Not found"
    fi
}

#######################################
# Legacy function for backwards compatibility
# Returns: 0 if healthy, 1 if not
#######################################
minio::status::check() {
    local verbose=${1:-false}
    
    if [[ "$verbose" == "true" ]]; then
        minio::status::show --format text
    else
        minio::status::show --format text >/dev/null 2>&1
    fi
    
    return $?
}

#######################################
# Check if MinIO is registered in Vrooli config
# Returns: 0 if registered, 1 if not
#######################################
minio::status::check_vrooli_config() {
    local config_file
    config_file="$(secrets::get_project_config_file)"
    
    if [[ ! -f "$config_file" ]]; then
        return 1
    fi
    
    # Check if MinIO is in the config using jq for proper JSON parsing
    # Try both possible locations in the config structure
    if jq -e '.dependencies.resources.minio.enabled == true' "$config_file" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

#######################################
# Show container resource usage
#######################################
minio::status::show_resources() {
    if ! minio::common::is_running; then
        log::warn "MinIO is not running"
        return 1
    fi
    
    log::info "MinIO Resource Usage:"
    log::info "===================="
    
    # Get container stats
    local stats=$(minio::docker::stats)
    
    if [[ -n "$stats" && "$stats" != "{}" ]]; then
        # Parse JSON stats (this is simplified, real implementation would use jq)
        echo "$stats" | grep -E "(MemUsage|CPUPerc|NetIO|BlockIO)" || {
            # Fallback to docker stats command
            docker stats "$MINIO_CONTAINER_NAME" --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
        }
    fi
    
    # Show disk usage
    log::info ""
    log::info "Storage Usage:"
    local data_size=$(du -sh "${MINIO_DATA_DIR}" 2>/dev/null | cut -f1 || echo "N/A")
    log::info "  Data Directory: $data_size"
    
    # Show bucket stats
    if minio::common::is_running; then
        log::info ""
        minio::buckets::show_stats
    fi
}

#######################################
# Display credentials
# Returns: 0 on success, 1 on failure
#######################################
minio::status::show_credentials() {
    local creds_file="${MINIO_CONFIG_DIR}/credentials"
    
    log::info "MinIO Access Credentials:"
    log::info "========================"
    
    # Try to load from file first
    if [[ -f "$creds_file" ]]; then
        source "$creds_file"
    fi
    
    if [[ -n "${MINIO_ROOT_USER}" && -n "${MINIO_ROOT_PASSWORD}" ]]; then
        log::info "Username: ${MINIO_ROOT_USER}"
        log::info "Password: ${MINIO_ROOT_PASSWORD}"
        log::info ""
        log::info "Console URL: ${MINIO_CONSOLE_URL}"
        log::info "API Endpoint: ${MINIO_BASE_URL}"
        
        if [[ "${MINIO_ROOT_USER}" == "minioadmin" && "${MINIO_ROOT_PASSWORD}" == "minioadmin" ]]; then
            log::warn ""
            log::warn "⚠️  Using default credentials - not secure for production!"
            log::info "Run '$0 --action reset-credentials' to generate secure credentials"
        fi
        
        return 0
    else
        log::error "Credentials not found"
        log::info "MinIO may not be installed or credentials were not saved"
        return 1
    fi
}

#######################################
# Monitor MinIO health continuously
# Arguments:
#   $1 - Check interval in seconds (default: 5)
#######################################
minio::status::monitor() {
    local interval=${1:-5}
    
    log::info "Monitoring MinIO health (Press Ctrl+C to stop)..."
    log::info ""
    
    local iteration=0
    while true; do
        # Clear screen every 10 iterations
        if [[ $((iteration % 10)) -eq 0 ]]; then
            clear
            log::info "MinIO Health Monitor - $(date)"
            log::info "========================================"
        fi
        
        # Quick health check
        if minio::common::health_check; then
            printf "\r[$(date +%H:%M:%S)] Status: ✅ Healthy    "
        else
            printf "\r[$(date +%H:%M:%S)] Status: ❌ Unhealthy  "
        fi
        
        sleep "$interval"
        iteration=$((iteration + 1))
    done
}

#######################################
# Run diagnostic checks
# Returns: 0 if all pass, 1 if any fail
#######################################
minio::status::diagnose() {
    log::info "Running MinIO Diagnostics..."
    log::info "==========================="
    
    local issues=0
    
    # Check Docker
    log::info "1. Checking Docker..."
    if docker::check_daemon; then
        log::success "   Docker is available"
    else
        log::error "   Docker is not available"
        ((issues++))
    fi
    
    # Check ports
    log::info "2. Checking port availability..."
    if minio::common::is_port_available "${MINIO_PORT}"; then
        if minio::common::is_running; then
            log::error "   Port ${MINIO_PORT} appears free but MinIO is running"
            ((issues++))
        else
            log::success "   Port ${MINIO_PORT} is available"
        fi
    else
        if minio::common::is_running; then
            log::success "   Port ${MINIO_PORT} is in use by MinIO"
        else
            log::error "   Port ${MINIO_PORT} is in use by another process"
            ((issues++))
        fi
    fi
    
    # Check disk space
    log::info "3. Checking disk space..."
    if minio::common::check_disk_space; then
        log::success "   Sufficient disk space available"
    else
        log::error "   Insufficient disk space"
        ((issues++))
    fi
    
    # Check container
    log::info "4. Checking container..."
    if minio::common::container_exists; then
        log::success "   Container exists"
        
        if minio::common::is_running; then
            log::success "   Container is running"
            
            # Check logs for errors
            log::info "5. Checking container logs..."
            local error_count=$(docker logs "$MINIO_CONTAINER_NAME" 2>&1 | grep -c -i "error" || echo "0")
            if [[ $error_count -gt 0 ]]; then
                log::warn "   Found $error_count error(s) in logs"
                log::info "   Run '$0 --action logs' to view full logs"
            else
                log::success "   No errors in recent logs"
            fi
        else
            log::warn "   Container is stopped"
            ((issues++))
        fi
    else
        log::warn "   Container not found"
        ((issues++))
    fi
    
    # Summary
    log::info ""
    if [[ $issues -eq 0 ]]; then
        log::success "Diagnostics passed! No issues found."
        return 0
    else
        log::error "Diagnostics failed: $issues issue(s) found"
        return 1
    fi
}

#######################################
# Test MinIO functionality
# Returns: 0 if all tests pass, 1 if any fail, 2 if service not ready
#######################################
minio::test() {
    log::info "Testing MinIO functionality..."
    
    # Test 1: Check if MinIO is installed (container exists)
    if ! minio::common::container_exists; then
        log::error "❌ MinIO container is not installed"
        return 1
    fi
    log::success "✅ MinIO container is installed"
    
    # Test 2: Check if service is running
    if ! minio::common::is_running; then
        log::error "❌ MinIO service is not running"
        return 2
    fi
    log::success "✅ MinIO service is running"
    
    # Test 3: Check API health
    if ! minio::common::health_check; then
        log::error "❌ MinIO API is not responding"
        return 1
    fi
    log::success "✅ MinIO API is healthy"
    
    # Test 4: Check authentication and basic API operations
    log::info "Testing API authentication..."
    if minio::api::configure_mc >/dev/null 2>&1; then
        log::success "✅ Authentication successful"
        
        # Test 5: Test bucket operations
        log::info "Testing bucket operations..."
        local test_bucket="vrooli-test-bucket-$(date +%s)"
        
        if minio::api::create_bucket "$test_bucket" >/dev/null 2>&1; then
            log::success "✅ Bucket creation successful"
            
            # Clean up test bucket
            minio::api::mc rm --recursive --force "local/$test_bucket" >/dev/null 2>&1 || true
            log::success "✅ Bucket cleanup successful"
        else
            log::warn "⚠️  Bucket operations test failed - may be permission issue"
        fi
    else
        log::error "❌ Authentication failed"
        return 1
    fi
    
    # Test 6: Check storage metrics
    log::info "Testing storage metrics..."
    local used_space
    used_space=$(docker exec "$MINIO_CONTAINER_NAME" df -h /data 2>/dev/null | tail -1 | awk '{print $3}' || echo "unknown")
    if [[ "$used_space" != "unknown" ]]; then
        log::success "✅ Storage metrics available (used: $used_space)"
    else
        log::warn "⚠️  Storage metrics unavailable"
    fi
    
    log::success "🎉 All MinIO tests passed"
    return 0
}

#######################################
# Show comprehensive MinIO information
#######################################
minio::info() {
    cat << EOF
=== MinIO Resource Information ===

ID: minio
Category: storage
Display Name: MinIO Object Storage
Description: S3-compatible object storage server

Service Details:
- Container Name: $MINIO_CONTAINER_NAME
- API Port: $MINIO_PORT
- Console Port: $MINIO_CONSOLE_PORT
- API URL: http://localhost:$MINIO_PORT
- Console URL: http://localhost:$MINIO_CONSOLE_PORT
- Data Directory: $MINIO_DATA_DIR

Endpoints:
- Health Check: http://localhost:$MINIO_PORT/minio/health/live
- Server Info: http://localhost:$MINIO_PORT/minio/admin/v3/info
- Bucket List: http://localhost:$MINIO_PORT/minio/admin/v3/list-buckets
- Upload: PUT http://localhost:$MINIO_PORT/{bucket}/{object}
- Download: GET http://localhost:$MINIO_PORT/{bucket}/{object}

Authentication:
- Root User: $MINIO_ROOT_USER
- Root Password: [HIDDEN - use --action show-credentials]
- Access Key: $MINIO_ROOT_USER
- Secret Key: [HIDDEN - use --action show-credentials]

Configuration:
- Docker Image: $MINIO_IMAGE
- Version: $MINIO_VERSION
- Data Persistence: $MINIO_DATA_DIR
- Web Console: Enabled
- API Compatibility: S3 v4 signature

MinIO Features:
- S3-compatible API
- Multi-tenant buckets
- Versioning support
- Server-side encryption
- Erasure coding
- Cross-region replication
- Event notifications
- Lambda triggers
- Web-based management console

Example Usage:
# Create a bucket
$0 --action create-bucket --bucket my-data --policy download

# Upload test data
$0 --action test-upload

# List all buckets
$0 --action list-buckets

# Monitor health
$0 --action monitor --interval 10

Documentation: https://min.io/docs/minio/linux/reference/minio-server.html
EOF
}

# Export functions for subshell availability
export -f minio::test
export -f minio::info

#######################################
# Show MinIO status using standardized format
# Args: [--format json|text] [--verbose] [--fast]
#######################################
minio::status::show() {
    status::run_standard "minio" "minio::status::collect_data" "minio::status::display_text" "$@"
}

#######################################
# Main status function for CLI registration
#######################################
minio::status() {
    minio::status::show "$@"
}

#######################################
# Display status in text format
#######################################
minio::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📦 MinIO Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install MinIO, run: resource-minio manage install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🔌 API: ${data[api_url]:-unknown}"
    log::info "   🖥️  Console: ${data[console_url]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 API Port: ${data[api_port]:-unknown}"
    log::info "   🖥️  Console Port: ${data[console_port]:-unknown}"
    log::info "   📁 Data Directory: ${data[data_dir]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    if [[ -n "${data[storage_used]:-}" ]]; then
        log::info "   💾 Storage Used: ${data[storage_used]}"
    fi
}

# Export additional status functions
export -f minio::status::collect_data
export -f minio::status::show
export -f minio::status::display_text
export -f minio::status
