#!/usr/bin/env bash
# n8n Common Utility Functions
# Shared utilities used across all modules

# Source required utilities
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh" 2>/dev/null || true

#######################################
# Check if Docker is installed
# Returns: 0 if installed, 1 otherwise
#######################################
n8n::check_docker() {
    if ! system::is_command "docker"; then
        log::error "Docker is not installed"
        log::info "Please install Docker first: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        log::info "Start Docker with: sudo systemctl start docker"
        return 1
    fi
    
    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "Current user doesn't have Docker permissions"
        log::info "Add user to docker group: sudo usermod -aG docker $USER"
        log::info "Then log out and back in for changes to take effect"
        return 1
    fi
    
    return 0
}

#######################################
# Check if n8n container exists
# Returns: 0 if exists, 1 otherwise
#######################################
n8n::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${N8N_CONTAINER_NAME}$"
}

#######################################
# Check if n8n is running
# Returns: 0 if running, 1 otherwise
#######################################
n8n::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${N8N_CONTAINER_NAME}$"
}

#######################################
# Check if n8n API is responsive with enhanced diagnostics
# Returns: 0 if responsive, 1 otherwise
#######################################
n8n::is_healthy() {
    # Check if container is running first
    if ! n8n::is_running; then
        return 1
    fi
    
    # Check for database corruption indicators in logs
    local recent_logs
    recent_logs=$(docker logs "$N8N_CONTAINER_NAME" --tail 50 2>&1 || echo "")
    if echo "$recent_logs" | grep -qi "SQLITE_READONLY\|database.*locked\|database.*corrupted"; then
        log::warn "Database corruption detected in logs"
        return 1
    fi
    
    # n8n uses /healthz endpoint for health checks
    if system::is_command "curl"; then
        # Try multiple times as n8n takes time to fully initialize
        local attempts=0
        while [ $attempts -lt 5 ]; do
            if curl -f -s --max-time 5 "$N8N_BASE_URL/healthz" >/dev/null 2>&1; then
                return 0
            fi
            attempts=$((attempts + 1))
            sleep 2
        done
    fi
    return 1
}

#######################################
# Generate secure random password
#######################################
n8n::generate_password() {
    # Use multiple sources for randomness
    if system::is_command "openssl"; then
        openssl rand -base64 32 | tr -d "=+/" | cut -c1-16
    elif [[ -r /dev/urandom ]]; then
        tr -dc 'A-Za-z0-9!@#$%^&*' < /dev/urandom | head -c 16
    else
        # Fallback to timestamp-based password
        echo "n8n$(date +%s)$RANDOM" | sha256sum | cut -c1-16
    fi
}

#######################################
# Detect filesystem corruption in n8n data directory
# Returns: 0 if healthy, 1 if corrupted
#######################################
n8n::detect_filesystem_corruption() {
    if [[ ! -d "$N8N_DATA_DIR" ]]; then
        log::warn "n8n data directory does not exist: $N8N_DATA_DIR"
        return 1
    fi
    
    # Check if directory has zero links (corruption indicator)
    local links
    links=$(stat -c '%h' "$N8N_DATA_DIR" 2>/dev/null || echo "0")
    if [[ "$links" == "0" ]]; then
        log::error "Filesystem corruption detected: directory has 0 links"
        return 1
    fi
    
    # Check for deleted but open database files (if container is running)
    if n8n::is_running; then
        local n8n_pid
        n8n_pid=$(docker exec "$N8N_CONTAINER_NAME" pgrep -f 'node.*n8n' 2>/dev/null | head -1)
        if [[ -n "$n8n_pid" ]]; then
            local deleted_files
            deleted_files=$(docker exec "$N8N_CONTAINER_NAME" lsof -p "$n8n_pid" 2>/dev/null | grep -c '(deleted)' 2>/dev/null || echo "0")
            if [[ "$deleted_files" =~ ^[0-9]+$ ]] && [[ "$deleted_files" -gt 0 ]]; then
                log::error "Database corruption detected: $deleted_files deleted files still open"
                return 1
            fi
        fi
    fi
    
    return 0
}

#######################################
# Check database health and integrity
# Returns: 0 if healthy, 1 if corrupted
#######################################
n8n::check_database_health() {
    local db_file="$N8N_DATA_DIR/database.sqlite"
    
    if [[ ! -f "$db_file" ]]; then
        log::warn "Database file does not exist: $db_file"
        return 1
    fi
    
    # Check file permissions
    if [[ ! -r "$db_file" ]] || [[ ! -w "$db_file" ]]; then
        log::error "Database file has incorrect permissions"
        return 1
    fi
    
    # Check if SQLite can open the database
    if system::is_command "sqlite3"; then
        if ! echo "PRAGMA integrity_check;" | sqlite3 "$db_file" >/dev/null 2>&1; then
            log::error "Database integrity check failed"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Find best available backup
# Returns: path to best backup or empty string
#######################################
n8n::find_best_backup() {
    local backup_pattern="${HOME}/n8n-backup-*/database.sqlite"
    local best_backup=""
    local best_size=0
    
    # Find largest, most recent backup
    for backup in $backup_pattern; do
        if [[ -f "$backup" ]]; then
            local size
            size=$(stat -c '%s' "$backup" 2>/dev/null || echo "0")
            if [[ "$size" -gt "$best_size" ]]; then
                best_backup="$backup"
                best_size="$size"
            fi
        fi
    done
    
    echo "$best_backup"
}

#######################################
# Automatically recover from filesystem corruption
# Returns: 0 if recovered, 1 if recovery failed
#######################################
n8n::auto_recover() {
    log::warn "Attempting automatic recovery..."
    
    # Stop container if running
    if n8n::is_running; then
        log::info "Stopping corrupted n8n container..."
        docker stop "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
        docker rm "$N8N_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Recreate data directory
    log::info "Recreating data directory..."
    trash::safe_remove "$N8N_DATA_DIR" --no-confirm 2>/dev/null || true
    if ! n8n::create_directories; then
        log::error "Failed to recreate data directory"
        return 1
    fi
    
    # Restore from backup if available
    local best_backup
    best_backup=$(n8n::find_best_backup)
    if [[ -n "$best_backup" ]]; then
        log::info "Restoring database from backup: $best_backup"
        cp "$best_backup" "$N8N_DATA_DIR/database.sqlite" || {
            log::error "Failed to restore database from backup"
            return 1
        }
        
        # Fix permissions
        if command -v sudo::restore_owner &>/dev/null; then
            sudo::restore_owner "$N8N_DATA_DIR/database.sqlite"
        else
            chown "$(whoami):$(whoami)" "$N8N_DATA_DIR/database.sqlite" 2>/dev/null || true
        fi
        chmod 644 "$N8N_DATA_DIR/database.sqlite" || true
        
        log::success "Database restored from backup"
    else
        log::warn "No backup found - starting with fresh database"
    fi
    
    return 0
}

#######################################
# Create n8n data directory with enhanced validation
#######################################
n8n::create_directories() {
    log::info "Creating n8n data directory..."
    
    # Check if directory already exists and is corrupted
    if [[ -d "$N8N_DATA_DIR" ]] && ! n8n::detect_filesystem_corruption; then
        log::info "Data directory already exists and appears healthy"
        return 0
    fi
    
    # Remove any corrupted directory
    if [[ -d "$N8N_DATA_DIR" ]]; then
        log::warn "Removing potentially corrupted directory"
        trash::safe_remove "$N8N_DATA_DIR" --no-confirm 2>/dev/null || true
    fi
    
    # Create fresh directory
    mkdir -p "$N8N_DATA_DIR" || {
        log::error "Failed to create n8n data directory"
        return 1
    }
    
    # Set proper ownership and permissions
    if command -v sudo::restore_owner &>/dev/null; then
        sudo::restore_owner "$N8N_DATA_DIR"
    else
        local current_user
        current_user=$(whoami)
        chown "$current_user:$current_user" "$N8N_DATA_DIR" 2>/dev/null || true
    fi
    chmod 755 "$N8N_DATA_DIR" || true
    
    # Verify directory is healthy
    if ! n8n::detect_filesystem_corruption; then
        log::error "Created directory is still corrupted"
        return 1
    fi
    
    # Add rollback action
    resources::add_rollback_action \
        "Remove n8n data directory" \
        "trash::safe_remove '$N8N_DATA_DIR' --no-confirm 2>/dev/null || true" \
        10
    
    log::success "n8n directories created with proper permissions"
    return 0
}

#######################################
# Create Docker network for n8n
#######################################
n8n::create_network() {
    if ! docker network ls | grep -q "$N8N_NETWORK_NAME"; then
        log::info "Creating Docker network for n8n..."
        
        if docker network create "$N8N_NETWORK_NAME" >/dev/null 2>&1; then
            log::success "Docker network created"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove Docker network" \
                "docker network rm $N8N_NETWORK_NAME 2>/dev/null || true" \
                5
        else
            log::warn "Failed to create Docker network (may already exist)"
        fi
    fi
}

#######################################
# Check if port is available
# Args: port number
# Returns: 0 if available, 1 if in use
#######################################
n8n::is_port_available() {
    local port=$1
    
    if system::is_command "lsof"; then
        ! lsof -i :"$port" >/dev/null 2>&1
    elif system::is_command "netstat"; then
        ! netstat -tln | grep -q ":$port "
    else
        # If we can't check, assume it's available
        return 0
    fi
}

#######################################
# Wait for service to be ready
# Args: max_attempts (optional, default 30)
# Returns: 0 if ready, 1 if timeout
#######################################
n8n::wait_for_ready() {
    local max_attempts=${1:-$N8N_HEALTH_CHECK_MAX_ATTEMPTS}
    local attempt=0
    
    log::info "Waiting for n8n to be ready..."
    
    while [ $attempt -lt $max_attempts ]; do
        if n8n::is_healthy; then
            log::success "n8n is ready!"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep $N8N_HEALTH_CHECK_INTERVAL
    done
    
    echo
    log::error "n8n failed to become ready after $((max_attempts * N8N_HEALTH_CHECK_INTERVAL)) seconds"
    return 1
}

#######################################
# Get container environment variable
# Args: variable_name
# Returns: variable value or empty string
#######################################
n8n::get_container_env() {
    local var_name=$1
    
    if n8n::is_running; then
        docker exec "$N8N_CONTAINER_NAME" env | grep "^${var_name}=" | cut -d'=' -f2-
    fi
}

#######################################
# Check if basic auth is enabled
# Returns: 0 if enabled, 1 if disabled
#######################################
n8n::is_basic_auth_enabled() {
    local auth_active=$(n8n::get_container_env "N8N_BASIC_AUTH_ACTIVE")
    [[ "$auth_active" == "true" ]]
}

#######################################
# Validate workflow ID format
# Args: workflow_id
# Returns: 0 if valid, 1 if invalid
#######################################
n8n::validate_workflow_id() {
    local workflow_id=$1
    
    # n8n workflow IDs are typically numeric
    if [[ ! "$workflow_id" =~ ^[0-9]+$ ]]; then
        log::error "Invalid workflow ID format: $workflow_id"
        log::info "Workflow IDs should be numeric (e.g., 1, 42, 123)"
        return 1
    fi
    
    return 0
}

#######################################
# Comprehensive health check with automatic recovery
# Returns: 0 if healthy, 1 if issues detected
#######################################
n8n::comprehensive_health_check() {
    log::info "Running comprehensive n8n health check..."
    
    local issues_found=0
    
    # Check filesystem corruption
    if ! n8n::detect_filesystem_corruption; then
        log::error "Filesystem corruption detected"
        if [[ "${AUTO_RECOVER:-yes}" == "yes" ]]; then
            if n8n::auto_recover; then
                log::success "Automatic recovery completed"
            else
                log::error "Automatic recovery failed"
                issues_found=1
            fi
        else
            issues_found=1
        fi
    fi
    
    # Check database health
    if ! n8n::check_database_health; then
        log::warn "Database health issues detected"
        issues_found=1
    fi
    
    return $issues_found
}

#######################################
# Check for required commands
# Returns: 0 if all present, 1 if any missing
#######################################
n8n::check_requirements() {
    local missing=0
    
    # Required commands
    local required_commands=("docker" "curl")
    
    for cmd in "${required_commands[@]}"; do
        if ! system::is_command "$cmd"; then
            log::error "Required command not found: $cmd"
            missing=1
        fi
    done
    
    # Optional but recommended commands
    local optional_commands=("jq" "openssl" "sqlite3")
    
    for cmd in "${optional_commands[@]}"; do
        if ! system::is_command "$cmd"; then
            log::warn "Optional command not found: $cmd"
            log::info "Some features may be limited without $cmd"
        fi
    done
    
    return $missing
}