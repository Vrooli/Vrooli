#!/usr/bin/env bash
# LDAP/Active Directory Federation for Keycloak

# Add LDAP federation provider
keycloak::add_ldap_provider() {
    local realm="${1:-master}"
    local provider_name="${2:-ldap}"
    local connection_url="${3:-}"
    local users_dn="${4:-}"
    local bind_dn="${5:-}"
    local bind_password="${6:-}"
    
    if [[ -z "$connection_url" || -z "$users_dn" ]]; then
        log::error "LDAP connection URL and users DN are required"
        return 1
    fi
    
    log::info "Adding LDAP provider '$provider_name' to realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Configure LDAP provider
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/components" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
    "name": "${provider_name}",
    "providerId": "ldap",
    "providerType": "org.keycloak.storage.UserStorageProvider",
    "parentId": "${realm}",
    "config": {
        "enabled": ["true"],
        "priority": ["1"],
        "fullSyncPeriod": ["-1"],
        "changedSyncPeriod": ["-1"],
        "cachePolicy": ["DEFAULT"],
        "evictionDay": [],
        "evictionHour": [],
        "evictionMinute": [],
        "maxLifespan": [],
        "batchSizeForSync": ["1000"],
        "editMode": ["WRITABLE"],
        "importEnabled": ["true"],
        "syncRegistrations": ["false"],
        "vendor": ["other"],
        "usernameLDAPAttribute": ["uid"],
        "rdnLDAPAttribute": ["uid"],
        "uuidLDAPAttribute": ["entryUUID"],
        "userObjectClasses": ["inetOrgPerson, organizationalPerson"],
        "connectionUrl": ["${connection_url}"],
        "usersDn": ["${users_dn}"],
        "authType": ["simple"],
        "bindDn": ["${bind_dn}"],
        "bindCredential": ["${bind_password}"],
        "searchScope": ["2"],
        "validatePasswordPolicy": ["false"],
        "trustEmail": ["false"],
        "useTruststoreSpi": ["ldapsOnly"],
        "connectionPooling": ["true"],
        "connectionPoolingAuthentication": [],
        "connectionPoolingDebug": [],
        "connectionPoolingInitSize": [],
        "connectionPoolingMaxSize": [],
        "connectionPoolingPrefSize": [],
        "connectionPoolingProtocol": [],
        "connectionPoolingTimeout": [],
        "connectionTimeout": ["30000"],
        "readTimeout": ["30000"],
        "pagination": ["true"]
    }
}
EOF
    ) || {
        log::error "Failed to add LDAP provider"
        return 1
    }
    
    log::success "LDAP provider '$provider_name' added to realm: $realm"
    return 0
}

# Add Active Directory federation provider
keycloak::add_ad_provider() {
    local realm="${1:-master}"
    local provider_name="${2:-active-directory}"
    local connection_url="${3:-}"
    local users_dn="${4:-}"
    local bind_dn="${5:-}"
    local bind_password="${6:-}"
    
    if [[ -z "$connection_url" || -z "$users_dn" ]]; then
        log::error "AD connection URL and users DN are required"
        return 1
    fi
    
    log::info "Adding Active Directory provider '$provider_name' to realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Configure AD provider with AD-specific settings
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/components" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
    "name": "${provider_name}",
    "providerId": "ldap",
    "providerType": "org.keycloak.storage.UserStorageProvider",
    "parentId": "${realm}",
    "config": {
        "enabled": ["true"],
        "priority": ["1"],
        "fullSyncPeriod": ["-1"],
        "changedSyncPeriod": ["-1"],
        "cachePolicy": ["DEFAULT"],
        "batchSizeForSync": ["1000"],
        "editMode": ["WRITABLE"],
        "importEnabled": ["true"],
        "syncRegistrations": ["false"],
        "vendor": ["ad"],
        "usernameLDAPAttribute": ["sAMAccountName"],
        "rdnLDAPAttribute": ["cn"],
        "uuidLDAPAttribute": ["objectGUID"],
        "userObjectClasses": ["person, organizationalPerson, user"],
        "connectionUrl": ["${connection_url}"],
        "usersDn": ["${users_dn}"],
        "authType": ["simple"],
        "bindDn": ["${bind_dn}"],
        "bindCredential": ["${bind_password}"],
        "searchScope": ["2"],
        "validatePasswordPolicy": ["false"],
        "trustEmail": ["false"],
        "useTruststoreSpi": ["ldapsOnly"],
        "connectionPooling": ["true"],
        "connectionTimeout": ["30000"],
        "readTimeout": ["30000"],
        "pagination": ["true"],
        "allowKerberosAuthentication": ["false"],
        "useKerberosForPasswordAuthentication": ["false"]
    }
}
EOF
    ) || {
        log::error "Failed to add Active Directory provider"
        return 1
    }
    
    log::success "Active Directory provider '$provider_name' added to realm: $realm"
    return 0
}

# List LDAP/AD federation providers
keycloak::list_ldap_providers() {
    local realm="${1:-master}"
    
    log::info "Listing LDAP/AD providers in realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # List user federation providers
    local response
    response=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/components?parent=${realm}&type=org.keycloak.storage.UserStorageProvider" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to list LDAP/AD providers"
        return 1
    }
    
    # Parse and display providers
    echo "$response" | jq -r '.[] | select(.providerId == "ldap") | "\(.name) - \(.config.vendor[0] // "ldap") (\(if .config.enabled[0] == "true" then "enabled" else "disabled" end))"'
    return 0
}

# Remove LDAP/AD provider
keycloak::remove_ldap_provider() {
    local realm="${1:-master}"
    local provider_name="${2:-}"
    
    if [[ -z "$provider_name" ]]; then
        log::error "Provider name is required"
        return 1
    fi
    
    log::info "Removing LDAP/AD provider '$provider_name' from realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Find provider ID by name
    local response
    response=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/components?parent=${realm}&type=org.keycloak.storage.UserStorageProvider" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to find LDAP/AD provider"
        return 1
    }
    
    local provider_id
    provider_id=$(echo "$response" | jq -r --arg name "$provider_name" '.[] | select(.name == $name) | .id')
    
    if [[ -z "$provider_id" ]]; then
        log::error "Provider '$provider_name' not found"
        return 1
    fi
    
    # Remove provider
    curl -sf -X DELETE \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/components/${provider_id}" \
        -H "Authorization: Bearer ${token}" || {
        log::error "Failed to remove LDAP/AD provider"
        return 1
    }
    
    log::success "LDAP/AD provider '$provider_name' removed from realm: $realm"
    return 0
}

# Test LDAP/AD connection
keycloak::test_ldap_connection() {
    local realm="${1:-master}"
    local provider_name="${2:-}"
    
    if [[ -z "$provider_name" ]]; then
        log::error "Provider name is required"
        return 1
    fi
    
    log::info "Testing LDAP/AD connection for provider '$provider_name' in realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Find provider ID by name
    local response
    response=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/components?parent=${realm}&type=org.keycloak.storage.UserStorageProvider" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to find LDAP/AD provider"
        return 1
    }
    
    local provider_id
    provider_id=$(echo "$response" | jq -r --arg name "$provider_name" '.[] | select(.name == $name) | .id')
    
    if [[ -z "$provider_id" ]]; then
        log::error "Provider '$provider_name' not found"
        return 1
    fi
    
    # Test connection
    response=$(curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/testLDAPConnection" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "{\"componentId\":\"${provider_id}\"}" \
        -o /dev/null -w "%{http_code}") || {
        log::error "LDAP/AD connection test failed"
        return 1
    }
    
    if [[ "$response" == "204" ]]; then
        log::success "LDAP/AD connection test successful"
        return 0
    else
        log::error "LDAP/AD connection test failed with status: $response"
        return 1
    fi
}

# Sync users from LDAP/AD
keycloak::sync_ldap_users() {
    local realm="${1:-master}"
    local provider_name="${2:-}"
    local sync_type="${3:-triggerFullSync}"  # triggerFullSync or triggerChangedUsersSync
    
    if [[ -z "$provider_name" ]]; then
        log::error "Provider name is required"
        return 1
    fi
    
    log::info "Syncing users from LDAP/AD provider '$provider_name' in realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Find provider ID by name
    local response
    response=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/components?parent=${realm}&type=org.keycloak.storage.UserStorageProvider" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to find LDAP/AD provider"
        return 1
    }
    
    local provider_id
    provider_id=$(echo "$response" | jq -r --arg name "$provider_name" '.[] | select(.name == $name) | .id')
    
    if [[ -z "$provider_id" ]]; then
        log::error "Provider '$provider_name' not found"
        return 1
    fi
    
    # Trigger sync
    response=$(curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/user-storage/${provider_id}/sync?action=${sync_type}" \
        -H "Authorization: Bearer ${token}") || {
        log::error "Failed to sync LDAP/AD users"
        return 1
    }
    
    # Parse sync results
    local added=$(echo "$response" | jq -r '.added // 0')
    local updated=$(echo "$response" | jq -r '.updated // 0')
    local removed=$(echo "$response" | jq -r '.removed // 0')
    local failed=$(echo "$response" | jq -r '.failed // 0')
    
    log::success "LDAP/AD sync completed: Added=$added, Updated=$updated, Removed=$removed, Failed=$failed"
    return 0
}

# CLI wrapper functions
keycloak::ldap::add() {
    local realm="master"
    local name="ldap"
    local url=""
    local users_dn=""
    local bind_dn=""
    local bind_password=""
    local type="ldap"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --url)
                url="$2"
                shift 2
                ;;
            --users-dn)
                users_dn="$2"
                shift 2
                ;;
            --bind-dn)
                bind_dn="$2"
                shift 2
                ;;
            --bind-password)
                bind_password="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak ldap add --url <ldap://host:port> --users-dn <dn> [--bind-dn <dn>] [--bind-password <pass>] [--name <name>] [--realm <realm>] [--type ldap|ad]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$url" || -z "$users_dn" ]]; then
        log::error "LDAP URL and users DN are required"
        echo "Usage: resource-keycloak ldap add --url <ldap://host:port> --users-dn <dn> [--bind-dn <dn>] [--bind-password <pass>] [--name <name>] [--realm <realm>] [--type ldap|ad]"
        return 1
    fi
    
    if [[ "$type" == "ad" ]]; then
        keycloak::add_ad_provider "$realm" "$name" "$url" "$users_dn" "$bind_dn" "$bind_password"
    else
        keycloak::add_ldap_provider "$realm" "$name" "$url" "$users_dn" "$bind_dn" "$bind_password"
    fi
}

keycloak::ldap::list() {
    local realm="master"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    keycloak::list_ldap_providers "$realm"
}

keycloak::ldap::remove() {
    local realm="master"
    local name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak ldap remove --name <provider-name> [--realm <realm>]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Provider name is required"
        echo "Usage: resource-keycloak ldap remove --name <provider-name> [--realm <realm>]"
        return 1
    fi
    
    keycloak::remove_ldap_provider "$realm" "$name"
}

keycloak::ldap::test() {
    local realm="master"
    local name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak ldap test --name <provider-name> [--realm <realm>]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Provider name is required"
        echo "Usage: resource-keycloak ldap test --name <provider-name> [--realm <realm>]"
        return 1
    fi
    
    keycloak::test_ldap_connection "$realm" "$name"
}

keycloak::ldap::sync() {
    local realm="master"
    local name=""
    local full="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --full)
                full="true"
                shift
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak ldap sync --name <provider-name> [--realm <realm>] [--full]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Provider name is required"
        echo "Usage: resource-keycloak ldap sync --name <provider-name> [--realm <realm>] [--full]"
        return 1
    fi
    
    local sync_type="triggerChangedUsersSync"
    if [[ "$full" == "true" ]]; then
        sync_type="triggerFullSync"
    fi
    
    keycloak::sync_ldap_users "$realm" "$name" "$sync_type"
}