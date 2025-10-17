#!/usr/bin/env bash
set -euo pipefail

# Password Policy Management for Keycloak
# Provides password policy configuration and enforcement

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KEYCLOAK_LIB_DIR="${APP_ROOT}/resources/keycloak/lib"

# Source dependencies
source "${KEYCLOAK_LIB_DIR}/common.sh"
source "${KEYCLOAK_LIB_DIR}/core.sh"

# Get port from registry
KEYCLOAK_PORT="${KEYCLOAK_PORT:-$(keycloak::get_port)}"

# Set password policy for a realm
keycloak::password_policy::set() {
    local realm="${1:-master}"
    shift || true
    
    log::info "Setting password policy for realm: ${realm}"
    
    # Build policy string from arguments
    local policy_items=()
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --length)
                policy_items+=("length($2)")
                shift 2
                ;;
            --digits)
                policy_items+=("digits($2)")
                shift 2
                ;;
            --lowercase)
                policy_items+=("lowerCase($2)")
                shift 2
                ;;
            --uppercase)
                policy_items+=("upperCase($2)")
                shift 2
                ;;
            --special)
                policy_items+=("specialChars($2)")
                shift 2
                ;;
            --not-username)
                policy_items+=("notUsername")
                shift
                ;;
            --not-email)
                policy_items+=("notEmail")
                shift
                ;;
            --expire-days)
                policy_items+=("forceExpiredPasswordChange($2)")
                shift 2
                ;;
            --history)
                policy_items+=("passwordHistory($2)")
                shift 2
                ;;
            --blacklist)
                policy_items+=("passwordBlacklist($2)")
                shift 2
                ;;
            --regex)
                policy_items+=("regexPattern($2)")
                shift 2
                ;;
            *)
                log::error "Unknown policy option: $1"
                return 1
                ;;
        esac
    done
    
    # Join policy items with " and "
    local policy_string
    policy_string=$(IFS=" and "; echo "${policy_items[*]}")
    
    if [[ -z "${policy_string}" ]]; then
        log::error "No policy options provided"
        echo "Usage: keycloak password-policy set <realm> [options]"
        echo "Options:"
        echo "  --length <min>         Minimum password length"
        echo "  --digits <min>         Minimum number of digits"
        echo "  --lowercase <min>      Minimum lowercase letters"
        echo "  --uppercase <min>      Minimum uppercase letters"
        echo "  --special <min>        Minimum special characters"
        echo "  --not-username         Password cannot be username"
        echo "  --not-email           Password cannot be email"
        echo "  --expire-days <days>   Password expiration in days"
        echo "  --history <count>      Password history (prevent reuse)"
        echo "  --blacklist <file>     Password blacklist file"
        echo "  --regex <pattern>      Regex pattern requirement"
        return 1
    fi
    
    log::info "Policy string: ${policy_string}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Update realm with password policy
    local response
    response=$(curl -sf -X PUT \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d "{\"passwordPolicy\": \"${policy_string}\"}") || {
        log::error "Failed to set password policy"
        return 1
    }
    
    log::success "Password policy set for realm: ${realm}"
}

# Get current password policy
keycloak::password_policy::get() {
    local realm="${1:-master}"
    
    log::info "Getting password policy for realm: ${realm}"
    
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
    
    local policy
    policy=$(echo "${realm_config}" | jq -r '.passwordPolicy // "No policy set"')
    
    echo ""
    echo "🔐 Password Policy for realm: ${realm}"
    echo "─────────────────────────────────────────"
    
    if [[ "${policy}" == "No policy set" ]]; then
        echo "⚠️  No password policy configured"
    else
        # Parse and display policy in readable format
        echo "Current policy: ${policy}"
        echo ""
        echo "Policy breakdown:"
        
        # Parse each policy component
        IFS=' and ' read -ra POLICIES <<< "${policy}"
        for p in "${POLICIES[@]}"; do
            if [[ "${p}" == "length("* ]]; then
                local len="${p#length(}"
                len="${len%)}"
                echo "  📏 Minimum length: ${len}"
            elif [[ "${p}" == "digits("* ]]; then
                local dig="${p#digits(}"
                dig="${dig%)}"
                echo "  🔢 Minimum digits: ${dig}"
            elif [[ "${p}" == "lowerCase("* ]]; then
                local low="${p#lowerCase(}"
                low="${low%)}"
                echo "  🔡 Minimum lowercase: ${low}"
            elif [[ "${p}" == "upperCase("* ]]; then
                local up="${p#upperCase(}"
                up="${up%)}"
                echo "  🔠 Minimum uppercase: ${up}"
            elif [[ "${p}" == "specialChars("* ]]; then
                local spec="${p#specialChars(}"
                spec="${spec%)}"
                echo "  ✨ Minimum special chars: ${spec}"
            elif [[ "${p}" == "notUsername" ]]; then
                echo "  ❌ Cannot be username"
            elif [[ "${p}" == "notEmail" ]]; then
                echo "  ❌ Cannot be email"
            elif [[ "${p}" == "passwordHistory("* ]]; then
                local hist="${p#passwordHistory(}"
                hist="${hist%)}"
                echo "  📜 Password history: ${hist}"
            elif [[ "${p}" == "forceExpiredPasswordChange("* ]]; then
                local exp="${p#forceExpiredPasswordChange(}"
                exp="${exp%)}"
                echo "  ⏰ Expires after: ${exp} days"
            elif [[ "${p}" == "passwordBlacklist("* ]]; then
                local bl="${p#passwordBlacklist(}"
                bl="${bl%)}"
                echo "  🚫 Blacklist: ${bl}"
            elif [[ "${p}" == "regexPattern("* ]]; then
                local regex="${p#regexPattern(}"
                regex="${regex%)}"
                echo "  🔍 Regex pattern: ${regex}"
            else
                echo "  ❓ ${p}"
            fi
        done
    fi
    echo ""
}

# Clear password policy
keycloak::password_policy::clear() {
    local realm="${1:-master}"
    
    log::info "Clearing password policy for realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Clear password policy
    local response
    response=$(curl -sf -X PUT \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json" \
        -d '{"passwordPolicy": ""}') || {
        log::error "Failed to clear password policy"
        return 1
    }
    
    log::success "Password policy cleared for realm: ${realm}"
}

# Apply preset password policy
keycloak::password_policy::apply_preset() {
    local realm="${1:-master}"
    local preset="${2:-moderate}"
    
    log::info "Applying ${preset} password policy preset to realm: ${realm}"
    
    case "${preset}" in
        basic)
            # Basic security
            keycloak::password_policy::set "${realm}" \
                --length 8 \
                --not-username
            ;;
        moderate)
            # Moderate security (recommended)
            keycloak::password_policy::set "${realm}" \
                --length 10 \
                --digits 1 \
                --lowercase 1 \
                --uppercase 1 \
                --not-username \
                --not-email \
                --history 3
            ;;
        strong)
            # Strong security
            keycloak::password_policy::set "${realm}" \
                --length 12 \
                --digits 2 \
                --lowercase 2 \
                --uppercase 2 \
                --special 1 \
                --not-username \
                --not-email \
                --history 5 \
                --expire-days 90
            ;;
        paranoid)
            # Maximum security
            keycloak::password_policy::set "${realm}" \
                --length 16 \
                --digits 3 \
                --lowercase 3 \
                --uppercase 3 \
                --special 2 \
                --not-username \
                --not-email \
                --history 10 \
                --expire-days 30
            ;;
        *)
            log::error "Unknown preset: ${preset}"
            echo "Available presets: basic, moderate, strong, paranoid"
            return 1
            ;;
    esac
    
    log::success "Applied ${preset} password policy preset"
}

# Validate password against policy
keycloak::password_policy::validate() {
    local realm="${1:-master}"
    local password="${2}"
    
    if [[ -z "${password}" ]]; then
        log::error "Password is required"
        return 1
    fi
    
    log::info "Validating password against policy for realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    # Note: Keycloak doesn't have a direct password validation endpoint
    # This would typically be done client-side or through custom extension
    
    # Get current policy
    local realm_config
    realm_config=$(curl -sf -X GET \
        "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}" \
        -H "Authorization: Bearer ${token}" \
        -H "Content-Type: application/json") || {
        log::error "Failed to get realm configuration"
        return 1
    }
    
    local policy
    policy=$(echo "${realm_config}" | jq -r '.passwordPolicy // ""')
    
    if [[ -z "${policy}" ]]; then
        log::success "No policy set - password is valid"
        return 0
    fi
    
    # Basic client-side validation
    local valid=true
    local issues=()
    
    # Check length
    if [[ "${policy}" =~ length\(([0-9]+)\) ]]; then
        local min_length="${BASH_REMATCH[1]}"
        if [[ ${#password} -lt ${min_length} ]]; then
            valid=false
            issues+=("Password must be at least ${min_length} characters")
        fi
    fi
    
    # Check digits
    if [[ "${policy}" =~ digits\(([0-9]+)\) ]]; then
        local min_digits="${BASH_REMATCH[1]}"
        local digit_count=$(echo "${password}" | grep -o '[0-9]' | wc -l)
        if [[ ${digit_count} -lt ${min_digits} ]]; then
            valid=false
            issues+=("Password must contain at least ${min_digits} digits")
        fi
    fi
    
    # Check lowercase
    if [[ "${policy}" =~ lowerCase\(([0-9]+)\) ]]; then
        local min_lower="${BASH_REMATCH[1]}"
        local lower_count=$(echo "${password}" | grep -o '[a-z]' | wc -l)
        if [[ ${lower_count} -lt ${min_lower} ]]; then
            valid=false
            issues+=("Password must contain at least ${min_lower} lowercase letters")
        fi
    fi
    
    # Check uppercase
    if [[ "${policy}" =~ upperCase\(([0-9]+)\) ]]; then
        local min_upper="${BASH_REMATCH[1]}"
        local upper_count=$(echo "${password}" | grep -o '[A-Z]' | wc -l)
        if [[ ${upper_count} -lt ${min_upper} ]]; then
            valid=false
            issues+=("Password must contain at least ${min_upper} uppercase letters")
        fi
    fi
    
    if [[ "${valid}" == "true" ]]; then
        log::success "Password meets policy requirements"
    else
        log::error "Password does not meet policy requirements:"
        for issue in "${issues[@]}"; do
            echo "  ❌ ${issue}"
        done
        return 1
    fi
}

# Force password reset for users
keycloak::password_policy::force_reset() {
    local realm="${1:-master}"
    local username="${2:-}"
    
    log::info "Forcing password reset in realm: ${realm}"
    
    # Get access token
    local token
    token=$(keycloak::get_admin_token) || return 1
    
    if [[ -n "${username}" ]]; then
        # Force reset for specific user
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
        
        # Add UPDATE_PASSWORD required action
        local response
        response=$(curl -sf -X PUT \
            "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users/${user_id}" \
            -H "Authorization: Bearer ${token}" \
            -H "Content-Type: application/json" \
            -d '{"requiredActions": ["UPDATE_PASSWORD"]}') || {
            log::error "Failed to force password reset"
            return 1
        }
        
        log::success "Password reset required for user: ${username}"
    else
        # Force reset for all users
        log::warning "Forcing password reset for ALL users in realm: ${realm}"
        
        # Get all users
        local users
        users=$(curl -sf -X GET \
            "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users" \
            -H "Authorization: Bearer ${token}" \
            -H "Content-Type: application/json") || {
            log::error "Failed to get users"
            return 1
        }
        
        # Force reset for each user
        echo "${users}" | jq -r '.[].id' | while read -r user_id; do
            curl -sf -X PUT \
                "http://localhost:${KEYCLOAK_PORT}/admin/realms/${realm}/users/${user_id}" \
                -H "Authorization: Bearer ${token}" \
                -H "Content-Type: application/json" \
                -d '{"requiredActions": ["UPDATE_PASSWORD"]}' || continue
        done
        
        log::success "Password reset required for all users"
    fi
}

# Main password policy command handler
keycloak::password_policy::main() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        set)
            keycloak::password_policy::set "$@"
            ;;
        get)
            keycloak::password_policy::get "$@"
            ;;
        clear)
            keycloak::password_policy::clear "$@"
            ;;
        preset)
            keycloak::password_policy::apply_preset "$@"
            ;;
        validate)
            keycloak::password_policy::validate "$@"
            ;;
        force-reset)
            keycloak::password_policy::force_reset "$@"
            ;;
        *)
            log::error "Unknown password-policy subcommand: ${subcommand}"
            echo "Usage: keycloak password-policy [set|get|clear|preset|validate|force-reset]"
            return 1
            ;;
    esac
}