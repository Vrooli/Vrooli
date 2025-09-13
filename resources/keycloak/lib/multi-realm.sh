#!/usr/bin/env bash
################################################################################
# Keycloak Multi-Realm Tenant Isolation Support
# Provides functions for managing isolated tenant realms in Keycloak
################################################################################

set -euo pipefail

# Source dependencies
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
source "${APP_ROOT}/resources/keycloak/lib/common.sh" 2>/dev/null || true

# Create an isolated tenant realm with default settings
keycloak::realm::create_tenant() {
    local tenant_name="${1:-}"
    local tenant_admin_email="${2:-}"
    local tenant_admin_password="${3:-}"
    
    if [[ -z "$tenant_name" ]]; then
        log::error "Tenant name is required"
        echo "Usage: keycloak realm create-tenant <tenant-name> [admin-email] [admin-password]"
        return 1
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Normalize tenant name for realm ID
    local realm_id
    realm_id=$(echo "$tenant_name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')
    
    log::info "Creating isolated tenant realm: $realm_id"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    if [[ -z "${token}" ]]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    # Create realm configuration with tenant isolation settings
    local realm_config
    realm_config=$(cat <<EOF
{
    "id": "${realm_id}",
    "realm": "${realm_id}",
    "displayName": "${tenant_name}",
    "enabled": true,
    "sslRequired": "external",
    "registrationAllowed": false,
    "loginWithEmailAllowed": true,
    "duplicateEmailsAllowed": false,
    "resetPasswordAllowed": true,
    "editUsernameAllowed": false,
    "bruteForceProtected": true,
    "permanentLockout": false,
    "maxFailureWaitSeconds": 900,
    "minimumQuickLoginWaitSeconds": 60,
    "waitIncrementSeconds": 60,
    "quickLoginCheckMilliSeconds": 1000,
    "maxDeltaTimeSeconds": 43200,
    "failureFactor": 30,
    "defaultRoles": ["${realm_id}-user"],
    "roles": {
        "realm": [
            {
                "name": "${realm_id}-admin",
                "description": "Administrator role for ${tenant_name} tenant",
                "composite": true,
                "composites": {
                    "realm": ["${realm_id}-user", "${realm_id}-manager"]
                }
            },
            {
                "name": "${realm_id}-manager",
                "description": "Manager role for ${tenant_name} tenant",
                "composite": false
            },
            {
                "name": "${realm_id}-user",
                "description": "Basic user role for ${tenant_name} tenant",
                "composite": false
            }
        ]
    },
    "groups": [
        {
            "name": "${realm_id}-users",
            "path": "/${realm_id}-users",
            "attributes": {
                "tenant": ["${tenant_name}"]
            }
        }
    ],
    "eventsEnabled": true,
    "eventsExpiration": 7200,
    "enabledEventTypes": ["LOGIN", "LOGIN_ERROR", "LOGOUT", "LOGOUT_ERROR"],
    "adminEventsEnabled": true,
    "adminEventsDetailsEnabled": true,
    "internationalizationEnabled": false,
    "supportedLocales": ["en"],
    "browserFlow": "browser",
    "registrationFlow": "registration",
    "directGrantFlow": "direct grant",
    "resetCredentialsFlow": "reset credentials",
    "clientAuthenticationFlow": "clients",
    "dockerAuthenticationFlow": "docker auth",
    "attributes": {
        "tenantId": "${realm_id}",
        "tenantName": "${tenant_name}",
        "tenantType": "isolated",
        "createdAt": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    }
}
EOF
)
    
    # Create the realm
    local response
    response=$(timeout 10 curl -sf -X POST "http://localhost:${port}/admin/realms" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${realm_config}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Successfully created tenant realm: ${realm_id}"
        
        # Create admin user if email and password provided
        if [[ -n "$tenant_admin_email" ]] && [[ -n "$tenant_admin_password" ]]; then
            keycloak::realm::create_tenant_admin "$realm_id" "$tenant_admin_email" "$tenant_admin_password"
        fi
        
        # Create default client for the tenant
        keycloak::realm::create_default_client "$realm_id"
        
        return 0
    else
        log::error "Failed to create tenant realm: ${response}"
        return 1
    fi
}

# Create tenant admin user
keycloak::realm::create_tenant_admin() {
    local realm_id="${1}"
    local admin_email="${2}"
    local admin_password="${3}"
    
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    
    local user_config
    user_config=$(cat <<EOF
{
    "username": "${admin_email}",
    "email": "${admin_email}",
    "emailVerified": true,
    "enabled": true,
    "credentials": [{
        "type": "password",
        "value": "${admin_password}",
        "temporary": false
    }],
    "realmRoles": ["${realm_id}-admin"],
    "groups": ["/${realm_id}-users"],
    "attributes": {
        "role": ["tenant-admin"],
        "tenant": ["${realm_id}"]
    }
}
EOF
)
    
    local response
    response=$(timeout 10 curl -sf -X POST "http://localhost:${port}/admin/realms/${realm_id}/users" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${user_config}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Created tenant admin user: ${admin_email}"
    else
        log::warning "Failed to create tenant admin: ${response}"
    fi
}

# Create default OAuth2 client for tenant
keycloak::realm::create_default_client() {
    local realm_id="${1}"
    
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    
    local client_config
    client_config=$(cat <<EOF
{
    "clientId": "${realm_id}-app",
    "name": "${realm_id} Application",
    "description": "Default application client for ${realm_id} tenant",
    "enabled": true,
    "clientAuthenticatorType": "client-secret",
    "secret": "$(openssl rand -hex 32)",
    "redirectUris": ["http://localhost:*", "https://*.${realm_id}.local/*"],
    "webOrigins": ["+"],
    "standardFlowEnabled": true,
    "implicitFlowEnabled": false,
    "directAccessGrantsEnabled": true,
    "serviceAccountsEnabled": true,
    "authorizationServicesEnabled": false,
    "publicClient": false,
    "protocol": "openid-connect",
    "fullScopeAllowed": false,
    "defaultClientScopes": ["openid", "profile", "email"],
    "attributes": {
        "tenant": "${realm_id}",
        "type": "default"
    }
}
EOF
)
    
    local response
    response=$(timeout 10 curl -sf -X POST "http://localhost:${port}/admin/realms/${realm_id}/clients" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${client_config}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Created default client for tenant: ${realm_id}-app"
    else
        log::warning "Failed to create default client: ${response}"
    fi
}

# List all tenant realms
keycloak::realm::list_tenants() {
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    if [[ -z "${token}" ]]; then
        log::error "Failed to get admin token"
        return 1
    fi
    
    log::info "Listing tenant realms..."
    
    # Get all realms
    local realms_json
    realms_json=$(timeout 10 curl -sf -X GET "http://localhost:${port}/admin/realms" \
        -H "Authorization: Bearer ${token}" 2>/dev/null)
    
    if [[ -z "$realms_json" ]]; then
        log::error "Failed to fetch realms"
        return 1
    fi
    
    # Filter and display tenant realms (exclude master realm)
    echo "$realms_json" | jq -r '.[] | select(.realm != "master") | 
        "\n📦 Tenant: \(.displayName // .realm)" +
        "\n   ID: \(.realm)" +
        "\n   Enabled: \(.enabled)" +
        "\n   Type: \(.attributes.tenantType // "standard")" +
        "\n   Created: \(.attributes.createdAt // "N/A")"'
    
    # Count tenants
    local tenant_count
    tenant_count=$(echo "$realms_json" | jq '[.[] | select(.realm != "master")] | length')
    
    log::info "Total tenant realms: ${tenant_count}"
}

# Get tenant realm details
keycloak::realm::get_tenant() {
    local realm_id="${1:-}"
    
    if [[ -z "$realm_id" ]]; then
        log::error "Realm ID is required"
        echo "Usage: keycloak realm get-tenant <realm-id>"
        return 1
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    
    log::info "Getting details for tenant realm: ${realm_id}"
    
    # Get realm details
    local realm_json
    realm_json=$(timeout 10 curl -sf -X GET "http://localhost:${port}/admin/realms/${realm_id}" \
        -H "Authorization: Bearer ${token}" 2>/dev/null)
    
    if [[ -z "$realm_json" ]]; then
        log::error "Failed to fetch realm details or realm not found"
        return 1
    fi
    
    # Display realm details
    echo "$realm_json" | jq '.'
    
    # Get user count
    local user_count
    user_count=$(timeout 10 curl -sf -X GET "http://localhost:${port}/admin/realms/${realm_id}/users/count" \
        -H "Authorization: Bearer ${token}" 2>/dev/null || echo "0")
    
    # Get client count
    local clients_json
    clients_json=$(timeout 10 curl -sf -X GET "http://localhost:${port}/admin/realms/${realm_id}/clients" \
        -H "Authorization: Bearer ${token}" 2>/dev/null || echo "[]")
    local client_count
    client_count=$(echo "$clients_json" | jq 'length')
    
    log::info "Statistics:"
    log::info "  Users: ${user_count}"
    log::info "  Clients: ${client_count}"
}

# Delete tenant realm
keycloak::realm::delete_tenant() {
    local realm_id="${1:-}"
    
    if [[ -z "$realm_id" ]]; then
        log::error "Realm ID is required"
        echo "Usage: keycloak realm delete-tenant <realm-id>"
        return 1
    fi
    
    if [[ "$realm_id" == "master" ]]; then
        log::error "Cannot delete master realm"
        return 1
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    
    log::warning "Deleting tenant realm: ${realm_id}"
    
    # Delete the realm
    local response
    response=$(timeout 10 curl -sf -X DELETE "http://localhost:${port}/admin/realms/${realm_id}" \
        -H "Authorization: Bearer ${token}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        log::success "Successfully deleted tenant realm: ${realm_id}"
        return 0
    else
        log::error "Failed to delete tenant realm: ${response}"
        return 1
    fi
}

# Export tenant realm configuration
keycloak::realm::export_tenant() {
    local realm_id="${1:-}"
    local export_file="${2:-}"
    
    if [[ -z "$realm_id" ]]; then
        log::error "Realm ID is required"
        echo "Usage: keycloak realm export-tenant <realm-id> [export-file]"
        return 1
    fi
    
    if [[ -z "$export_file" ]]; then
        export_file="/tmp/keycloak-${realm_id}-export-$(date +%Y%m%d-%H%M%S).json"
    fi
    
    local port
    port=$(keycloak::get_port)
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token)
    
    log::info "Exporting tenant realm: ${realm_id}"
    
    # Export realm with users and groups
    local realm_json
    realm_json=$(timeout 10 curl -sf -X GET "http://localhost:${port}/admin/realms/${realm_id}" \
        -H "Authorization: Bearer ${token}" 2>/dev/null)
    
    if [[ -z "$realm_json" ]]; then
        log::error "Failed to fetch realm configuration"
        return 1
    fi
    
    # Save to file
    echo "$realm_json" | jq '.' > "$export_file"
    
    log::success "Exported realm configuration to: ${export_file}"
    log::info "File size: $(du -h "$export_file" | cut -f1)"
}