#!/usr/bin/env bash
# exit_codes.sh
# Central definitions of global exit codes for scripts and tests.
# These use default assignments so tests or callers can override them by exporting beforehand.
set -euo pipefail

# Initialize the exit codes array
EXIT_CODES=()

# Define exit codes and add them to the array
: "${EXIT_SUCCESS:=0}"
: "${DESC_EXIT_SUCCESS:=Success}"
EXIT_CODES+=("EXIT_SUCCESS")

: "${ERROR_DEFAULT:=1}"
: "${DESC_ERROR_DEFAULT:=Default error}"
EXIT_CODES+=("ERROR_DEFAULT")

: "${ERROR_USAGE:=64}"
: "${DESC_ERROR_USAGE:=Command line usage error}"
EXIT_CODES+=("ERROR_USAGE")

: "${ERROR_NO_INTERNET:=65}"
: "${DESC_ERROR_NO_INTERNET:=No internet access}"
EXIT_CODES+=("ERROR_NO_INTERNET")

: "${ERROR_ENV_FILE_MISSING:=66}"
: "${DESC_ERROR_ENV_FILE_MISSING:=Environment file missing}"
EXIT_CODES+=("ERROR_ENV_FILE_MISSING")

: "${ERROR_FUNCTION_NOT_FOUND:=67}"
: "${DESC_ERROR_FUNCTION_NOT_FOUND:=Function not found}"
EXIT_CODES+=("ERROR_FUNCTION_NOT_FOUND")

: "${ERROR_DOMAIN_RESOLVE:=68}"
: "${DESC_ERROR_DOMAIN_RESOLVE:=Failed to resolve domain}"
EXIT_CODES+=("ERROR_DOMAIN_RESOLVE")

: "${ERROR_INVALID_SITE_IP:=69}"
: "${DESC_ERROR_INVALID_SITE_IP:=Invalid site IP}"
EXIT_CODES+=("ERROR_INVALID_SITE_IP")

: "${ERROR_CURRENT_IP_FAIL:=70}"
: "${DESC_ERROR_CURRENT_IP_FAIL:=Failed to retrieve current IP}"
EXIT_CODES+=("ERROR_CURRENT_IP_FAIL")

: "${ERROR_SITE_IP_MISMATCH:=71}"
: "${DESC_ERROR_SITE_IP_MISMATCH:=Site IP mismatch}"
EXIT_CODES+=("ERROR_SITE_IP_MISMATCH")

: "${ERROR_BUILD_FAILED:=72}"
: "${DESC_ERROR_BUILD_FAILED:=Build failed}"
EXIT_CODES+=("ERROR_BUILD_FAILED")

: "${ERROR_PROXY_CONTAINER_START_FAILED:=73}"
: "${DESC_ERROR_PROXY_CONTAINER_START_FAILED:=Proxy containers failed to start}"
EXIT_CODES+=("ERROR_PROXY_CONTAINER_START_FAILED")

: "${ERROR_PROXY_LOCATION_NOT_FOUND:=74}"
: "${DESC_ERROR_PROXY_LOCATION_NOT_FOUND:=Proxy location not found or is invalid}"
EXIT_CODES+=("ERROR_PROXY_LOCATION_NOT_FOUND")

: "${ERROR_PROXY_CLONE_FAILED:=75}"
: "${DESC_ERROR_PROXY_CLONE_FAILED:=Proxy clone and setup failed}"
EXIT_CODES+=("ERROR_PROXY_CLONE_FAILED")

: "${ERROR_INSTALLATION_FAILED:=76}"
: "${DESC_ERROR_INSTALLATION_FAILED:=Installation failed}"
EXIT_CODES+=("ERROR_INSTALLATION_FAILED")

: "${ERROR_SUDO_REQUIRED:=77}"
: "${DESC_ERROR_SUDO_REQUIRED:=Sudo required}"
EXIT_CODES+=("ERROR_SUDO_REQUIRED")

: "${ERROR_NETWORK:=78}"
: "${DESC_ERROR_NETWORK:=Network error}"
EXIT_CODES+=("ERROR_NETWORK")

: "${ERROR_INVALID_STATE:=79}"
: "${DESC_ERROR_INVALID_STATE:=Invalid state}"
EXIT_CODES+=("ERROR_INVALID_STATE")

: "${ERROR_DEPENDENCY_MISSING:=80}"
: "${DESC_ERROR_DEPENDENCY_MISSING:=Dependency missing}"
EXIT_CODES+=("ERROR_DEPENDENCY_MISSING")

: "${ERROR_PERMISSION_DENIED:=81}"
: "${DESC_ERROR_PERMISSION_DENIED:=Permission denied}"
EXIT_CODES+=("ERROR_PERMISSION_DENIED")

: "${ERROR_OPERATION_FAILED:=82}"
: "${DESC_ERROR_OPERATION_FAILED:=Operation failed}"
EXIT_CODES+=("ERROR_OPERATION_FAILED")

: "${ERROR_CONFIG_INVALID:=83}"
: "${DESC_ERROR_CONFIG_INVALID:=Configuration invalid}"
EXIT_CODES+=("ERROR_CONFIG_INVALID")

: "${ERROR_NOT_IMPLEMENTED:=84}"
: "${DESC_ERROR_NOT_IMPLEMENTED:=Not implemented}"
EXIT_CODES+=("ERROR_NOT_IMPLEMENTED")

: "${ERROR_VAULT_CONNECTION_FAILED:=85}"
: "${DESC_ERROR_VAULT_CONNECTION_FAILED:=Vault connection failed}"
EXIT_CODES+=("ERROR_VAULT_CONNECTION_FAILED")

: "${ERROR_VAULT_AUTH_FAILED:=86}"
: "${DESC_ERROR_VAULT_AUTH_FAILED:=Vault authentication failed}"
EXIT_CODES+=("ERROR_VAULT_AUTH_FAILED")

: "${ERROR_VAULT_SECRET_FETCH_FAILED:=87}"
: "${DESC_ERROR_VAULT_SECRET_FETCH_FAILED:=Vault secret fetch failed}"
EXIT_CODES+=("ERROR_VAULT_SECRET_FETCH_FAILED")

: "${ERROR_MISSING_DEPENDENCIES:=88}"
: "${DESC_ERROR_MISSING_DEPENDENCIES:=Missing dependencies}"
EXIT_CODES+=("ERROR_MISSING_DEPENDENCIES")

: "${ERROR_JWT_FILE_MISSING:=89}"
: "${DESC_ERROR_JWT_FILE_MISSING:=JWT file missing}"
EXIT_CODES+=("ERROR_JWT_FILE_MISSING")

: "${ERROR_COMMAND_NOT_FOUND:=90}"
: "${DESC_ERROR_COMMAND_NOT_FOUND:=Command not found}"
EXIT_CODES+=("ERROR_COMMAND_NOT_FOUND")

: "${ERROR_SSH_FILE_MISSING:=91}"
: "${DESC_ERROR_SSH_FILE_MISSING:=SSH key file missing}"
EXIT_CODES+=("ERROR_SSH_FILE_MISSING")

: "${ERROR_DOCKER_LOGIN_FAILED:=92}"
: "${DESC_ERROR_DOCKER_LOGIN_FAILED:=Docker login failed}"
EXIT_CODES+=("ERROR_DOCKER_LOGIN_FAILED")

# Helper function to generate exit codes display for usage
exit_codes::print() {
    local var_name code desc_var description
    echo "Exit Codes:"
    for var_name in "${EXIT_CODES[@]}"; do
        code=${!var_name}
        desc_var="DESC_${var_name}"
        # Check if description variable exists, use a default if not
        if [[ -n "${!desc_var+x}" ]]; then
            description="${!desc_var}"
        else
            description="No description available"
        fi
        printf "  %-6s%s\n" "$code" "$description"
    done
}