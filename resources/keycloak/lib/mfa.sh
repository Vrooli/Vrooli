#!/usr/bin/env bash
set -euo pipefail

# Multi-Factor Authentication (MFA) Configuration for Keycloak
# Provides MFA/2FA setup and management

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Source dependencies
source "${KEYCLOAK_LIB_DIR}/common.sh"
source "${KEYCLOAK_LIB_DIR}/core.sh"

# Get port from registry
KEYCLOAK_PORT="${KEYCLOAK_PORT:-$(keycloak::get_port)}"

# Enable MFA for a realm
keycloak::mfa::enable() {
    local realm="${1:-master}"
    local mfa_type="${2:-totp}"  # totp, webauthn, sms
    
    log::info "Enabling MFA (${mfa_type}) for realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Get current realm configuration
    local realm_config
    realm_config=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to get realm configuration"
        return 1
    }
    
    # Update realm configuration based on MFA type
    case "${mfa_type}" in
        totp)
            # Enable OTP (One-Time Password) via authenticator apps
            realm_config=$(echo "${realm_config}" | jq '. + {
                "otpPolicyType": "totp",
                "otpPolicyAlgorithm": "HmacSHA1",
                "otpPolicyInitialCounter": 0,
                "otpPolicyDigits": 6,
                "otpPolicyLookAheadWindow": 1,
                "otpPolicyPeriod": 30,
                "otpSupportedApplications": ["totpAppGoogleName", "totpAppMicrosoftAuthenticatorName"],
                "browserFlow": "browser",
                "requiredActions": ["CONFIGURE_TOTP"]
            }')
            ;;
        webauthn)
            # Enable WebAuthn (hardware keys, biometrics)
            realm_config=$(echo "${realm_config}" | jq '. + {
                "webAuthnPolicyRpEntityName": "Keycloak",
                "webAuthnPolicySignatureAlgorithms": ["ES256", "RS256"],
                "webAuthnPolicyRpId": "",
                "webAuthnPolicyAttestationConveyancePreference": "not specified",
                "webAuthnPolicyAuthenticatorAttachment": "not specified",
                "webAuthnPolicyRequireResidentKey": "not specified",
                "webAuthnPolicyUserVerificationRequirement": "not specified",
                "webAuthnPolicyCreateTimeout": 0,
                "webAuthnPolicyAvoidSameAuthenticatorRegister": false,
                "requiredActions": ["webauthn-register"]
            }')
            ;;
        sms)
            log::warning "SMS MFA requires external SMS provider configuration"
            # Basic SMS configuration (requires external provider)
            realm_config=$(echo "${realm_config}" | jq '. + {
                "requiredActions": ["sms_auth"],
                "attributes": {
                    "smsAuthEnabled": "true"
                }
            }')
            ;;
        *)
            log::error "Unknown MFA type: ${mfa_type}"
            return 1
            ;;
    esac
    
    # Update realm with MFA configuration
    local response
    response=$(curl -sf -X PUT \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${realm_config}") || {
        log::error "Failed to enable MFA"
        return 1
    }
    
    log::success "MFA (${mfa_type}) enabled for realm: ${realm}"
}

# Disable MFA for a realm
keycloak::mfa::disable() {
    local realm="${1:-master}"
    
    log::info "Disabling MFA for realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Update realm to remove required actions
    local response
    response=$(curl -sf -X PUT \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d '{
            "requiredActions": []
        }') || {
        log::error "Failed to disable MFA"
        return 1
    }
    
    log::success "MFA disabled for realm: ${realm}"
}

# Configure MFA policy
keycloak::mfa::configure_policy() {
    local realm="${1:-master}"
    local policy_type="${2:-conditional}"  # always, conditional, optional
    
    log::info "Configuring MFA policy (${policy_type}) for realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Get authentication flows
    local flows
    flows=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/authentication/flows" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to get authentication flows"
        return 1
    }
    
    # Find browser flow
    local browser_flow_id
    browser_flow_id=$(echo "${flows}" | jq -r '.[] | select(.alias == "browser") | .id')
    
    if [[ -z "${browser_flow_id}" ]]; then
        log::error "Browser flow not found"
        return 1
    fi
    
    # Create MFA execution based on policy type
    case "${policy_type}" in
        always)
            # MFA required for all users
            local execution_config='{"requirement": "REQUIRED"}'
            ;;
        conditional)
            # MFA based on conditions (risk, IP, etc.)
            local execution_config='{"requirement": "CONDITIONAL"}'
            ;;
        optional)
            # MFA optional for users
            local execution_config='{"requirement": "OPTIONAL"}'
            ;;
        *)
            log::error "Unknown policy type: ${policy_type}"
            return 1
            ;;
    esac
    
    # Update flow execution requirement
    local response
    response=$(curl -sf -X PUT \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/authentication/flows/${browser_flow_id}/executions" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "${execution_config}") || {
        log::error "Failed to configure MFA policy"
        return 1
    }
    
    log::success "MFA policy (${policy_type}) configured for realm: ${realm}"
}

# Enable MFA for specific user
keycloak::mfa::enable_for_user() {
    local realm="${1:-master}"
    local username="${2}"
    
    if [[ -z "${username}" ]]; then
        log::error "Username is required"
        return 1
    fi
    
    log::info "Enabling MFA for user: ${username} in realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Find user
    local users
    users=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users?username=${username}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to find user"
        return 1
    }
    
    local user_id
    user_id=$(echo "${users}" | jq -r '.[0].id')
    
    if [[ -z "${user_id}" || "${user_id}" == "null" ]]; then
        log::error "User not found: ${username}"
        return 1
    fi
    
    # Add required action for MFA setup
    local response
    response=$(curl -sf -X PUT \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users/${user_id}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d '{
            "requiredActions": ["CONFIGURE_TOTP"]
        }') || {
        log::error "Failed to enable MFA for user"
        return 1
    }
    
    log::success "MFA enabled for user: ${username}"
    log::info "User will be prompted to configure MFA on next login"
}

# Check MFA status for a realm
keycloak::mfa::status() {
    local realm="${1:-master}"
    
    log::info "Checking MFA status for realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Get realm configuration
    local realm_config
    realm_config=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to get realm configuration"
        return 1
    }
    
    # Check OTP policy
    local otp_enabled
    otp_enabled=$(echo "${realm_config}" | jq -r '.otpPolicyType // "disabled"')
    
    # Check WebAuthn policy
    local webauthn_enabled
    webauthn_enabled=$(echo "${realm_config}" | jq -r '.webAuthnPolicyRpEntityName // ""')
    
    # Check required actions
    local required_actions
    required_actions=$(echo "${realm_config}" | jq -r '.requiredActions // []')
    
    echo ""
    echo "🔐 MFA Status for realm: ${realm}"
    echo "─────────────────────────────────────"
    echo "📱 TOTP/OTP: ${otp_enabled}"
    if [[ "${otp_enabled}" != "disabled" ]]; then
        echo "   Algorithm: $(echo "${realm_config}" | jq -r '.otpPolicyAlgorithm')"
        echo "   Digits: $(echo "${realm_config}" | jq -r '.otpPolicyDigits')"
        echo "   Period: $(echo "${realm_config}" | jq -r '.otpPolicyPeriod')s"
    fi
    
    if [[ -n "${webauthn_enabled}" ]]; then
        echo "🔑 WebAuthn: Enabled"
        echo "   Entity: ${webauthn_enabled}"
    else
        echo "🔑 WebAuthn: Disabled"
    fi
    
    echo "📋 Required Actions: ${required_actions}"
    echo ""
}

# List users with MFA enabled
keycloak::mfa::list_users() {
    local realm="${1:-master}"
    
    log::info "Listing users with MFA in realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Get all users
    local users
    users=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to get users"
        return 1
    }
    
    echo ""
    echo "👥 Users with MFA Status in realm: ${realm}"
    echo "─────────────────────────────────────────────"
    
    # Check each user's credentials
    echo "${users}" | jq -r '.[] | "\(.username): \(.totp // false)"' | while read -r line; do
        local username="${line%%:*}"
        local has_totp="${line##*: }"
        
        if [[ "${has_totp}" == "true" ]]; then
            echo "✅ ${username} - MFA configured"
        else
            echo "⚠️  ${username} - MFA not configured"
        fi
    done
    echo ""
}

# Generate backup codes for user
keycloak::mfa::generate_backup_codes() {
    local realm="${1:-master}"
    local username="${2}"
    local count="${3:-8}"
    
    if [[ -z "${username}" ]]; then
        log::error "Username is required"
        return 1
    fi
    
    log::info "Generating ${count} backup codes for user: ${username}"
    
    # Note: Keycloak doesn't have built-in backup codes API
    # This would require custom extension or alternative implementation
    log::warning "Backup codes feature requires custom Keycloak extension"
    log::info "Alternative: Use multiple MFA methods (TOTP + WebAuthn) for redundancy"
}

# Main MFA command handler
keycloak::mfa::main() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        enable)
            keycloak::mfa::enable "$@"
            ;;
        disable)
            keycloak::mfa::disable "$@"
            ;;
        configure)
            keycloak::mfa::configure_policy "$@"
            ;;
        enable-user)
            keycloak::mfa::enable_for_user "$@"
            ;;
        status)
            keycloak::mfa::status "$@"
            ;;
        list-users)
            keycloak::mfa::list_users "$@"
            ;;
        backup-codes)
            keycloak::mfa::generate_backup_codes "$@"
            ;;
        *)
            log::error "Unknown MFA subcommand: ${subcommand}"
            echo "Usage: keycloak mfa [enable|disable|configure|enable-user|status|list-users|backup-codes]"
            return 1
            ;;
    esac
}