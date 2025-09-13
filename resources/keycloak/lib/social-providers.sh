#!/usr/bin/env bash
# Social Login Provider Management for Keycloak

# CLI wrapper functions that parse arguments properly
keycloak::social::add_github() {
    local realm="master"
    local client_id=""
    local client_secret=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --client-id)
                client_id="$2"
                shift 2
                ;;
            --client-secret)
                client_secret="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak social add-github --client-id <id> --client-secret <secret> [--realm <realm>]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$client_id" || -z "$client_secret" ]]; then
        log::error "GitHub Client ID and Secret are required"
        echo "Usage: resource-keycloak social add-github --client-id <id> --client-secret <secret> [--realm <realm>]"
        return 1
    fi
    
    keycloak::add_github_provider "$realm" "$client_id" "$client_secret"
}

keycloak::social::add_google() {
    local realm="master"
    local client_id=""
    local client_secret=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --client-id)
                client_id="$2"
                shift 2
                ;;
            --client-secret)
                client_secret="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak social add-google --client-id <id> --client-secret <secret> [--realm <realm>]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$client_id" || -z "$client_secret" ]]; then
        log::error "Google Client ID and Secret are required"
        echo "Usage: resource-keycloak social add-google --client-id <id> --client-secret <secret> [--realm <realm>]"
        return 1
    fi
    
    keycloak::add_google_provider "$realm" "$client_id" "$client_secret"
}

keycloak::social::add_facebook() {
    local realm="master"
    local app_id=""
    local app_secret=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --app-id)
                app_id="$2"
                shift 2
                ;;
            --app-secret)
                app_secret="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak social add-facebook --app-id <id> --app-secret <secret> [--realm <realm>]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$app_id" || -z "$app_secret" ]]; then
        log::error "Facebook App ID and Secret are required"
        echo "Usage: resource-keycloak social add-facebook --app-id <id> --app-secret <secret> [--realm <realm>]"
        return 1
    fi
    
    keycloak::add_facebook_provider "$realm" "$app_id" "$app_secret"
}

keycloak::social::list() {
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
    
    keycloak::list_social_providers "$realm"
}

keycloak::social::remove() {
    local realm="master"
    local alias=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --alias)
                alias="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak social remove --alias <provider-alias> [--realm <realm>]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$alias" ]]; then
        log::error "Provider alias is required"
        echo "Usage: resource-keycloak social remove --alias <provider-alias> [--realm <realm>]"
        return 1
    fi
    
    keycloak::remove_social_provider "$realm" "$alias"
}

keycloak::social::test() {
    local realm="master"
    local alias=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --realm)
                realm="$2"
                shift 2
                ;;
            --alias)
                alias="$2"
                shift 2
                ;;
            *)
                log::error "Unknown option: $1"
                echo "Usage: resource-keycloak social test --alias <provider-alias> [--realm <realm>]"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$alias" ]]; then
        log::error "Provider alias is required"
        echo "Usage: resource-keycloak social test --alias <provider-alias> [--realm <realm>]"
        return 1
    fi
    
    keycloak::test_social_provider "$realm" "$alias"
}

# Add GitHub provider to a realm
keycloak::add_github_provider() {
    local realm="${1:-master}"
    local client_id="${2:-}"
    local client_secret="${3:-}"
    
    if [[ -z "$client_id" || -z "$client_secret" ]]; then
        log::error "GitHub Client ID and Secret are required"
        return 1
    fi
    
    log::info "Adding GitHub provider to realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Configure GitHub provider
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/identity-provider/instances" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
    "alias": "github",
    "displayName": "GitHub",
    "providerId": "github",
    "enabled": true,
    "trustEmail": true,
    "storeToken": true,
    "addReadTokenRoleOnCreate": false,
    "config": {
        "clientId": "${client_id}",
        "clientSecret": "${client_secret}",
        "useJwksUrl": "true",
        "syncMode": "IMPORT"
    }
}
EOF
    ) || {
        log::error "Failed to add GitHub provider"
        return 1
    }
    
    log::success "GitHub provider added to realm: $realm"
    return 0
}

# Add Google provider to a realm
keycloak::add_google_provider() {
    local realm="${1:-master}"
    local client_id="${2:-}"
    local client_secret="${3:-}"
    
    if [[ -z "$client_id" || -z "$client_secret" ]]; then
        log::error "Google Client ID and Secret are required"
        return 1
    fi
    
    log::info "Adding Google provider to realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Configure Google provider
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/identity-provider/instances" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
    "alias": "google",
    "displayName": "Google",
    "providerId": "google",
    "enabled": true,
    "trustEmail": true,
    "storeToken": true,
    "config": {
        "clientId": "${client_id}",
        "clientSecret": "${client_secret}",
        "useJwksUrl": "true",
        "syncMode": "IMPORT"
    }
}
EOF
    ) || {
        log::error "Failed to add Google provider"
        return 1
    }
    
    log::success "Google provider added to realm: $realm"
    return 0
}

# Add Facebook provider to a realm
keycloak::add_facebook_provider() {
    local realm="${1:-master}"
    local app_id="${2:-}"
    local app_secret="${3:-}"
    
    if [[ -z "$app_id" || -z "$app_secret" ]]; then
        log::error "Facebook App ID and Secret are required"
        return 1
    fi
    
    log::info "Adding Facebook provider to realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Configure Facebook provider
    local response
    response=$(curl -sf -X POST \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/identity-provider/instances" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d @- <<EOF
{
    "alias": "facebook",
    "displayName": "Facebook",
    "providerId": "facebook",
    "enabled": true,
    "trustEmail": true,
    "storeToken": true,
    "config": {
        "clientId": "${app_id}",
        "clientSecret": "${app_secret}",
        "syncMode": "IMPORT"
    }
}
EOF
    ) || {
        log::error "Failed to add Facebook provider"
        return 1
    }
    
    log::success "Facebook provider added to realm: $realm"
    return 0
}

# List social providers in a realm
keycloak::list_social_providers() {
    local realm="${1:-master}"
    
    log::info "Listing social providers in realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # List providers
    local response
    response=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/identity-provider/instances" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to list social providers"
        return 1
    }
    
    echo "$response" | jq -r '.[] | "\(.alias) - \(.displayName) (\(if .enabled then "enabled" else "disabled" end))"'
    return 0
}

# Remove a social provider from a realm
keycloak::remove_social_provider() {
    local realm="${1:-master}"
    local alias="${2:-}"
    
    if [[ -z "$alias" ]]; then
        log::error "Provider alias is required"
        return 1
    fi
    
    log::info "Removing social provider '$alias' from realm: $realm"
    
    # Get admin token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Remove provider
    local response
    response=$(curl -sf -X DELETE \
        "http://localhost:${KEYCLOAK_PORT:-8070}/admin/realms/${realm}/identity-provider/instances/${alias}" \
        -H "Authorization: Bearer ${token}") || {
        log::error "Failed to remove social provider"
        return 1
    }
    
    log::success "Social provider '$alias' removed from realm: $realm"
    return 0
}

# Test social provider configuration
keycloak::test_social_provider() {
    local realm="${1:-master}"
    local alias="${2:-}"
    
    if [[ -z "$alias" ]]; then
        log::error "Provider alias is required"
        return 1
    fi
    
    log::info "Testing social provider '$alias' in realm: $realm"
    
    # Get the provider URL
    local provider_url="http://localhost:${KEYCLOAK_PORT:-8070}/realms/${realm}/broker/${alias}/login"
    
    # Check if the login URL is accessible
    if timeout 5 curl -sf -I "$provider_url" >/dev/null 2>&1; then
        log::success "Social provider '$alias' is configured and accessible"
        log::info "Login URL: $provider_url"
        return 0
    else
        log::error "Social provider '$alias' is not accessible"
        return 1
    fi
}