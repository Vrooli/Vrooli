#!/usr/bin/env bash
set -euo pipefail

# Define Keycloak lib directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Dependencies are expected to be sourced by caller

# Inject realm configuration into Keycloak
keycloak::inject() {
    local inject_file="${1:-}"
    
    if [[ -z "${inject_file}" ]]; then
        log::error "No injection file specified"
        return 1
    fi
    
    if [[ ! -f "${inject_file}" ]]; then
        log::error "Injection file not found: ${inject_file}"
        return 1
    fi
    
    # Check if Keycloak is running
    if ! keycloak::is_running; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Determine file type and handle accordingly
    local filename
    filename=$(basename "${inject_file}")
    local extension="${filename##*.}"
    
    case "${extension}" in
        json)
            # Validate JSON first
            if ! jq empty "${inject_file}" >/dev/null 2>&1; then
                log::error "Invalid JSON in file: ${inject_file}"
                return 1
            fi
            
            # Check if it's a realm configuration
            local realm_name
            realm_name=$(jq -r '.realm // .id // empty' "${inject_file}" 2>/dev/null)
            
            if [[ -n "${realm_name}" ]]; then
                # Import realm configuration
                keycloak::inject_realm "${inject_file}" "${realm_name}"
            else
                # Try to import as user or client configuration
                keycloak::inject_generic "${inject_file}"
            fi
            ;;
            
        *)
            log::error "Unsupported file type: ${extension}"
            log::info "Supported types: .json (realm configurations)"
            return 1
            ;;
    esac
    
    return 0
}

# Inject realm configuration
keycloak::inject_realm() {
    local realm_file="$1"
    local realm_name="$2"
    local port
    port=$(keycloak::get_port)
    
    log::info "Injecting realm configuration: ${realm_name}"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    if [[ -z "${token}" ]]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    # Check if realm already exists
    local realm_exists
    realm_exists=$(timeout 10 curl -sf -H "Authorization: Bearer ${token}" \
        "http://localhost:${port}/admin/realms/${realm_name}" >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "${realm_exists}" == "true" ]]; then
        log::warning "Realm '${realm_name}' already exists, updating..."
        # Update existing realm
        local response
        response=$(timeout 10 curl -sf -X PUT "http://localhost:${port}/admin/realms/${realm_name}" \
            -H "Authorization: Bearer ${token}" \
            -H "Content-Type: application/json" \
            -d @"${realm_file}" 2>&1)
        
        if [[ $? -eq 0 ]]; then
            log::success "Successfully updated realm '${realm_name}'"
        else
            log::error "Failed to update realm: ${response}"
            return 1
        fi
    else
        log::info "Creating new realm '${realm_name}'"
        # Create new realm
        local response
        response=$(timeout 10 curl -sf -X POST "http://localhost:${port}/admin/realms" \
            -H "Authorization: Bearer ${token}" \
            -H "Content-Type: application/json" \
            -d @"${realm_file}" 2>&1)
        
        if [[ $? -eq 0 ]]; then
            log::success "Successfully created realm '${realm_name}'"
        else
            log::error "Failed to create realm: ${response}"
            return 1
        fi
    fi
    
    return 0
}

# Inject generic configuration (users, clients, etc.)
keycloak::inject_generic() {
    local config_file="$1"
    local port
    port=$(keycloak::get_port)
    
    log::info "Injecting configuration from ${config_file}"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    if [[ -z "${token}" ]]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    # Try to determine what kind of configuration this is
    local config_type
    config_type=$(jq -r 'keys[0]' "${config_file}" 2>/dev/null || echo "unknown")
    
    case "${config_type}" in
        users|user)
            keycloak::inject_users "${config_file}" "${token}"
            ;;
        clients|client)
            keycloak::inject_clients "${config_file}" "${token}"
            ;;
        *)
            log::warning "Unknown configuration type, attempting generic import"
            # Copy to import directory for realm import on restart
            cp "${config_file}" "${KEYCLOAK_REALMS_DIR}/"
            log::info "Configuration copied to import directory. Restart Keycloak to import."
            ;;
    esac
    
    return 0
}

# Inject users
keycloak::inject_users() {
    local users_file="$1"
    local token="$2"
    local port
    port=$(keycloak::get_port)
    
    log::info "Injecting users"
    # Implementation for user injection would go here
    # This is a placeholder for now
    log::warning "User injection not yet implemented"
}

# Inject clients
keycloak::inject_clients() {
    local clients_file="$1"
    local token="$2"
    local port
    port=$(keycloak::get_port)
    
    log::info "Injecting clients"
    # Implementation for client injection would go here
    # This is a placeholder for now
    log::warning "Client injection not yet implemented"
}

# List injected data (realms, users, clients)
keycloak::list_injected() {
    if ! keycloak::is_running; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    if [[ -z "${token}" ]]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    log::info "Keycloak Statistics:"
    
    # Get realms
    log::info "Realms:"
    local realms
    realms=$(timeout 10 curl -sf -H "Authorization: Bearer ${token}" \
        "http://localhost:${port}/admin/realms" 2>/dev/null | jq -r '.[].realm' 2>/dev/null || echo "Error getting realms")
    
    if [[ "${realms}" == "Error getting realms" ]]; then
        echo "  Error retrieving realm list"
    else
        echo "${realms}" | while IFS= read -r realm; do
            if [[ -n "${realm}" ]]; then
                echo "  - ${realm}"
                
                # Get user count for each realm
                local user_count
                user_count=$(timeout 10 curl -sf -H "Authorization: Bearer ${token}" \
                    "http://localhost:${port}/admin/realms/${realm}/users/count" 2>/dev/null || echo "unknown")
                echo "    Users: ${user_count}"
                
                # Get client count for each realm
                local client_count
                client_count=$(timeout 10 curl -sf -H "Authorization: Bearer ${token}" \
                    "http://localhost:${port}/admin/realms/${realm}/clients" 2>/dev/null | \
                    jq '. | length' 2>/dev/null || echo "unknown")
                echo "    Clients: ${client_count}"
            fi
        done
    fi
}

# Clear all data from Keycloak
keycloak::clear_data() {
    if ! keycloak::is_running; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    log::warning "This will remove all custom realms, users, and clients (master realm will remain)"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log::info "Operation cancelled"
        return 0
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    if [[ -z "${token}" ]]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    # Get all realms except master
    local realms
    realms=$(timeout 10 curl -sf -H "Authorization: Bearer ${token}" \
        "http://localhost:${port}/admin/realms" 2>/dev/null | \
        jq -r '.[] | select(.realm != "master") | .realm' 2>/dev/null || true)
    
    if [[ -n "${realms}" ]]; then
        echo "${realms}" | while IFS= read -r realm; do
            if [[ -n "${realm}" ]]; then
                log::info "Deleting realm: ${realm}"
                timeout 10 curl -sf -X DELETE -H "Authorization: Bearer ${token}" \
                    "http://localhost:${port}/admin/realms/${realm}" >/dev/null 2>&1 || {
                    log::warning "Failed to delete realm: ${realm}"
                }
            fi
        done
    fi
    
    # Clear import directory
    if [[ -d "${KEYCLOAK_REALMS_DIR}" ]]; then
        rm -f "${KEYCLOAK_REALMS_DIR}"/*
    fi
    
    log::success "Keycloak data cleared"
}