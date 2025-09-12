#!/usr/bin/env bash
################################################################################
# Keycloak Content Management - v2.0 Contract Compliant
# Manages realm configurations, users, clients, and other Keycloak content
################################################################################

set -euo pipefail

# Define paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Source dependencies if not already loaded
[[ -z "${KEYCLOAK_CONTAINER_NAME:-}" ]] && source "${APP_ROOT}/resources/keycloak/config/defaults.sh"

# Content directory for storing realm configurations
KEYCLOAK_CONTENT_DIR="${KEYCLOAK_CONTENT_DIR:-/data/keycloak/content}"

# Add content to Keycloak (realm, user, client configurations)
keycloak::content::add() {
    local file_path=""
    local content_type=""
    local content_name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file_path="$2"
                shift 2
                ;;
            --type)
                content_type="$2"
                shift 2
                ;;
            --name)
                content_name="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    # Validate required arguments
    if [[ -z "$file_path" ]]; then
        log::error "Missing required --file argument"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    # Auto-detect content type if not specified
    if [[ -z "$content_type" ]]; then
        content_type=$(keycloak::content::detect_type "$file_path")
    fi
    
    # Auto-generate name if not specified
    if [[ -z "$content_name" ]]; then
        content_name=$(basename "$file_path" .json)
    fi
    
    # Process based on content type
    case "$content_type" in
        realm)
            keycloak::content::add_realm "$file_path" "$content_name"
            ;;
        user|users)
            keycloak::content::add_users "$file_path" "$content_name"
            ;;
        client|clients)
            keycloak::content::add_clients "$file_path" "$content_name"
            ;;
        *)
            log::error "Unknown content type: $content_type"
            log::info "Supported types: realm, user, client"
            return 1
            ;;
    esac
}

# List available content
keycloak::content::list() {
    local format="${1:-text}"
    
    # Check if Keycloak is running
    if ! docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
        log::warning "Keycloak is not running"
        return 1
    fi
    
    # Get admin token
    local token=$(keycloak::content::get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to authenticate with Keycloak"
        return 1
    fi
    
    local port="${KEYCLOAK_PORT:-8070}"
    
    # Get realms
    local realms=$(timeout 5 curl -sf \
        -H "Authorization: Bearer $token" \
        "http://localhost:${port}/admin/realms" 2>/dev/null || echo "[]")
    
    case "$format" in
        json)
            echo "$realms"
            ;;
        yaml)
            echo "$realms" | python3 -c "import sys, json, yaml; yaml.dump(json.load(sys.stdin), sys.stdout)"
            ;;
        text|*)
            echo "=== Keycloak Content ==="
            echo
            echo "Realms:"
            echo "$realms" | jq -r '.[].realm' 2>/dev/null | while read -r realm; do
                if [[ -n "$realm" ]]; then
                    echo "  - $realm"
                    
                    # Get user count
                    local user_count=$(timeout 5 curl -sf \
                        -H "Authorization: Bearer $token" \
                        "http://localhost:${port}/admin/realms/${realm}/users/count" 2>/dev/null || echo "0")
                    echo "    Users: $user_count"
                    
                    # Get client count
                    local clients=$(timeout 5 curl -sf \
                        -H "Authorization: Bearer $token" \
                        "http://localhost:${port}/admin/realms/${realm}/clients" 2>/dev/null || echo "[]")
                    local client_count=$(echo "$clients" | jq '. | length' 2>/dev/null || echo "0")
                    echo "    Clients: $client_count"
                fi
            done
            ;;
    esac
}

# Get specific content
keycloak::content::get() {
    local content_name=""
    local output_file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                content_name="$2"
                shift 2
                ;;
            --output)
                output_file="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$content_name" ]]; then
        log::error "Missing required --name argument"
        return 1
    fi
    
    # Check if Keycloak is running
    if ! docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    # Get admin token
    local token=$(keycloak::content::get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to authenticate with Keycloak"
        return 1
    fi
    
    local port="${KEYCLOAK_PORT:-8070}"
    
    # Try to get realm configuration
    local realm_config=$(timeout 5 curl -sf \
        -H "Authorization: Bearer $token" \
        "http://localhost:${port}/admin/realms/${content_name}" 2>/dev/null)
    
    if [[ -n "$realm_config" ]]; then
        if [[ -n "$output_file" ]]; then
            echo "$realm_config" | jq '.' > "$output_file"
            log::success "Realm configuration saved to $output_file"
        else
            echo "$realm_config" | jq '.'
        fi
        return 0
    else
        log::error "Content not found: $content_name"
        return 1
    fi
}

# Remove content
keycloak::content::remove() {
    local content_name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                content_name="$2"
                shift 2
                ;;
            *)
                log::error "Unknown argument: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$content_name" ]]; then
        log::error "Missing required --name argument"
        return 1
    fi
    
    # Check if Keycloak is running
    if ! docker ps --format "table {{.Names}}" | grep -q "${KEYCLOAK_CONTAINER_NAME}"; then
        log::error "Keycloak is not running"
        return 1
    fi
    
    # Get admin token
    local token=$(keycloak::content::get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to authenticate with Keycloak"
        return 1
    fi
    
    local port="${KEYCLOAK_PORT:-8070}"
    
    # Don't allow deletion of master realm
    if [[ "$content_name" == "master" ]]; then
        log::error "Cannot delete master realm"
        return 1
    fi
    
    # Try to delete realm
    if timeout 5 curl -sf -X DELETE \
        -H "Authorization: Bearer $token" \
        "http://localhost:${port}/admin/realms/${content_name}" 2>/dev/null; then
        log::success "Removed realm: $content_name"
        return 0
    else
        log::error "Failed to remove content: $content_name"
        return 1
    fi
}

# Execute content operations (e.g., run authentication flow)
keycloak::content::execute() {
    log::info "Execute operation not applicable for Keycloak"
    log::info "Use 'content add' to import configurations"
    log::info "Use 'content list' to view existing content"
    return 2
}

# Helper: Detect content type from JSON file
keycloak::content::detect_type() {
    local file_path="$1"
    
    # Check if file has realm field
    if jq -e '.realm' "$file_path" >/dev/null 2>&1; then
        echo "realm"
    elif jq -e '.users' "$file_path" >/dev/null 2>&1; then
        echo "users"
    elif jq -e '.clients' "$file_path" >/dev/null 2>&1; then
        echo "clients"
    elif jq -e '.clientId' "$file_path" >/dev/null 2>&1; then
        echo "client"
    elif jq -e '.username' "$file_path" >/dev/null 2>&1; then
        echo "user"
    else
        echo "unknown"
    fi
}

# Helper: Add realm configuration
keycloak::content::add_realm() {
    local file_path="$1"
    local realm_name="$2"
    
    # Get admin token
    local token=$(keycloak::content::get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to authenticate with Keycloak"
        return 1
    fi
    
    local port="${KEYCLOAK_PORT:-8070}"
    
    # Check if realm already exists
    if timeout 5 curl -sf \
        -H "Authorization: Bearer $token" \
        "http://localhost:${port}/admin/realms/${realm_name}" >/dev/null 2>&1; then
        log::warning "Realm already exists: $realm_name"
        return 2
    fi
    
    # Create realm
    if timeout 10 curl -sf -X POST \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d @"$file_path" \
        "http://localhost:${port}/admin/realms" 2>&1; then
        log::success "Added realm: $realm_name"
        return 0
    else
        log::error "Failed to add realm: $realm_name"
        return 1
    fi
}

# Helper: Add users
keycloak::content::add_users() {
    local file_path="$1"
    local realm_name="${2:-master}"
    
    # Get admin token
    local token=$(keycloak::content::get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to authenticate with Keycloak"
        return 1
    fi
    
    local port="${KEYCLOAK_PORT:-8070}"
    
    # Read users from file
    local users=$(cat "$file_path")
    
    # Check if it's an array or single user
    if echo "$users" | jq -e 'type == "array"' >/dev/null 2>&1; then
        # Multiple users
        local count=$(echo "$users" | jq '. | length')
        local i=0
        echo "$users" | jq -c '.[]' | while read -r user; do
            local username=$(echo "$user" | jq -r '.username // empty')
            if [[ -n "$username" ]]; then
                if timeout 10 curl -sf -X POST \
                    -H "Authorization: Bearer $token" \
                    -H "Content-Type: application/json" \
                    -d "$user" \
                    "http://localhost:${port}/admin/realms/${realm_name}/users" 2>&1; then
                    log::success "Added user: $username to realm: $realm_name"
                else
                    log::warning "Failed to add user: $username"
                fi
            fi
            ((i++))
        done
    else
        # Single user
        local username=$(echo "$users" | jq -r '.username // empty')
        if [[ -n "$username" ]]; then
            if timeout 10 curl -sf -X POST \
                -H "Authorization: Bearer $token" \
                -H "Content-Type: application/json" \
                -d "$users" \
                "http://localhost:${port}/admin/realms/${realm_name}/users" 2>&1; then
                log::success "Added user: $username to realm: $realm_name"
                return 0
            else
                log::error "Failed to add user: $username"
                return 1
            fi
        else
            log::error "Invalid user data in file"
            return 1
        fi
    fi
}

# Helper: Add clients
keycloak::content::add_clients() {
    local file_path="$1"
    local realm_name="${2:-master}"
    
    # Get admin token
    local token=$(keycloak::content::get_admin_token)
    if [[ -z "$token" ]]; then
        log::error "Failed to authenticate with Keycloak"
        return 1
    fi
    
    local port="${KEYCLOAK_PORT:-8070}"
    
    # Read clients from file
    local clients=$(cat "$file_path")
    
    # Check if it's an array or single client
    if echo "$clients" | jq -e 'type == "array"' >/dev/null 2>&1; then
        # Multiple clients
        echo "$clients" | jq -c '.[]' | while read -r client; do
            local client_id=$(echo "$client" | jq -r '.clientId // empty')
            if [[ -n "$client_id" ]]; then
                if timeout 10 curl -sf -X POST \
                    -H "Authorization: Bearer $token" \
                    -H "Content-Type: application/json" \
                    -d "$client" \
                    "http://localhost:${port}/admin/realms/${realm_name}/clients" 2>&1; then
                    log::success "Added client: $client_id to realm: $realm_name"
                else
                    log::warning "Failed to add client: $client_id"
                fi
            fi
        done
    else
        # Single client
        local client_id=$(echo "$clients" | jq -r '.clientId // empty')
        if [[ -n "$client_id" ]]; then
            if timeout 10 curl -sf -X POST \
                -H "Authorization: Bearer $token" \
                -H "Content-Type: application/json" \
                -d "$clients" \
                "http://localhost:${port}/admin/realms/${realm_name}/clients" 2>&1; then
                log::success "Added client: $client_id to realm: $realm_name"
                return 0
            else
                log::error "Failed to add client: $client_id"
                return 1
            fi
        else
            log::error "Invalid client data in file"
            return 1
        fi
    fi
}

# Helper: Get admin token
keycloak::content::get_admin_token() {
    local port="${KEYCLOAK_PORT:-8070}"
    local admin_user="${KEYCLOAK_ADMIN_USER:-admin}"
    local admin_pass="${KEYCLOAK_ADMIN_PASSWORD:-admin}"
    
    local response=$(timeout 5 curl -sf -X POST \
        "http://localhost:${port}/realms/master/protocol/openid-connect/token" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "username=${admin_user}" \
        -d "password=${admin_pass}" \
        -d "grant_type=password" \
        -d "client_id=admin-cli" 2>/dev/null || echo "{}")
    
    echo "$response" | jq -r '.access_token // empty' 2>/dev/null
}