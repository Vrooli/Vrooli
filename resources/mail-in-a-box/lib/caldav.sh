#!/bin/bash

source "$(dirname "${BASH_SOURCE[0]}")/core.sh"

# Source log functions
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || {
    # Fallback log functions if log.sh not available
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
}

CALDAV_PORT="${CALDAV_PORT:-5232}"
CALDAV_CONTAINER="mailinabox-caldav"
CALDAV_DATA_DIR="${MAILINABOX_DATA_DIR:-/var/lib/mailinabox}/radicale/data"

caldav_check_service() {
    if ! docker ps --filter "name=${CALDAV_CONTAINER}" --filter "status=running" -q 2>/dev/null | grep -q .; then
        log::error "CalDAV/CardDAV service not running"
        return 1
    fi
    return 0
}

caldav_create_user() {
    local username="$1"
    local password="$2"
    
    if [[ -z "$username" ]] || [[ -z "$password" ]]; then
        log::error "Username and password required"
        return 1
    fi
    
    if ! caldav_check_service; then
        return 1
    fi
    
    log::info "Creating CalDAV/CardDAV user: $username"
    
    # Hash password using bcrypt
    local hashed_password
    hashed_password=$(docker exec "${CALDAV_CONTAINER}" python3 -c "
import bcrypt
password = '${password}'.encode('utf-8')
hashed = bcrypt.hashpw(password, bcrypt.gensalt())
print(hashed.decode('utf-8'))
" 2>/dev/null) || {
        # Fallback to htpasswd if bcrypt not available
        hashed_password=$(docker exec "${CALDAV_CONTAINER}" htpasswd -nbB "$username" "$password" 2>/dev/null | cut -d: -f2)
    }
    
    if [[ -z "$hashed_password" ]]; then
        log::error "Failed to hash password"
        return 1
    fi
    
    # Add user to htpasswd file
    docker exec "${CALDAV_CONTAINER}" sh -c "echo '${username}:${hashed_password}' >> /data/users" || {
        log::error "Failed to add user"
        return 1
    }
    
    # Create user collections directory
    docker exec "${CALDAV_CONTAINER}" mkdir -p "/data/collections/collection-root/${username}" || {
        log::error "Failed to create user directory"
        return 1
    }
    
    log::success "CalDAV/CardDAV user created: $username"
    echo "Access URL: http://localhost:${CALDAV_PORT}/${username}"
    return 0
}

caldav_list_users() {
    if ! caldav_check_service; then
        return 1
    fi
    
    log::info "CalDAV/CardDAV users:"
    docker exec "${CALDAV_CONTAINER}" sh -c "[ -f /data/users ] && cut -d: -f1 /data/users || echo 'No users configured'" 2>/dev/null || {
        log::error "Failed to list users"
        return 1
    }
    return 0
}

caldav_delete_user() {
    local username="$1"
    
    if [[ -z "$username" ]]; then
        log::error "Username required"
        return 1
    fi
    
    if ! caldav_check_service; then
        return 1
    fi
    
    log::info "Deleting CalDAV/CardDAV user: $username"
    
    # Remove user from htpasswd file
    docker exec "${CALDAV_CONTAINER}" sh -c "sed -i '/^${username}:/d' /data/users" || {
        log::error "Failed to remove user from auth file"
        return 1
    }
    
    # Remove user collections
    docker exec "${CALDAV_CONTAINER}" rm -rf "/data/collections/collection-root/${username}" || {
        log::error "Failed to remove user data"
        return 1
    }
    
    log::success "CalDAV/CardDAV user deleted: $username"
    return 0
}

caldav_test_connection() {
    local username="${1:-admin}"
    local password="${2:-password}"
    
    if ! caldav_check_service; then
        return 1
    fi
    
    log::info "Testing CalDAV/CardDAV connection..."
    
    # Test basic connection
    if timeout 5 curl -sf "http://localhost:${CALDAV_PORT}/.web/" &>/dev/null; then
        log::success "Web interface accessible"
    else
        log::error "Web interface not accessible"
        return 1
    fi
    
    # Test authenticated access
    if timeout 5 curl -sf -u "${username}:${password}" "http://localhost:${CALDAV_PORT}/${username}/" &>/dev/null; then
        log::success "Authenticated access working"
    else
        log_warning "Authenticated access failed (user may not exist)"
    fi
    
    return 0
}

caldav_health() {
    if ! caldav_check_service; then
        echo "unhealthy"
        return 1
    fi
    
    if timeout 5 curl -sf "http://localhost:${CALDAV_PORT}/.web/" &>/dev/null; then
        echo "healthy"
        return 0
    else
        echo "unhealthy"
        return 1
    fi
}

caldav_info() {
    log::info "CalDAV/CardDAV Service Information"
    echo "===================="
    echo "Container: ${CALDAV_CONTAINER}"
    echo "Port: ${CALDAV_PORT}"
    echo "Web UI: http://localhost:${CALDAV_PORT}/.web/"
    echo "CalDAV URL: http://localhost:${CALDAV_PORT}/[username]/calendar"
    echo "CardDAV URL: http://localhost:${CALDAV_PORT}/[username]/contacts"
    echo "Data Directory: ${CALDAV_DATA_DIR}"
    echo "===================="
    
    if caldav_check_service; then
        log::success "Service is running"
        caldav_list_users
    else
        log_warning "Service is not running"
    fi
}