#!/bin/sh
# Vault functions shared by multiple scripts.
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

# Checks if Vault is initialized.
is_vault_initialized() {
    output=$(vault status 2>&1)
    exit_status=$?

    # Check if Vault is not initialized
    if echo "$output" | grep -E "Initialized\s*false" >/dev/null; then
        return 1
    elif echo "$output" | grep -E "Initialized\s*true" >/dev/null; then
        return 0
    else
        error "Failed to get Vault status. Received exit number $exit_status"
        error "Output: $output"
        return 1
    fi
}

assert_vault_initialized() {
    if ! is_vault_initialized; then
        error "Vault at $VAULT_ADDR is not initialized!"
        exit 1
    fi
    success "Vault is initialized."
}

# Checks if the Vault is either sealed or unsealed, depending on the expected status.
is_vault_sealed_status() {
    expected_status="$1" # Either "true" or "false"

    output=$(vault status 2>&1)
    exit_status=$?

    # Check if Vault is sealed or unsealed
    if echo "$output" | grep -E "Sealed\s*true" >/dev/null; then
        CURRENT_STATUS="true"
    elif echo "$output" | grep -E "Sealed\s*false" >/dev/null; then
        CURRENT_STATUS="false"
    else
        error "Failed to get Vault status. Received exit number $exit_status"
        error "Output: $output"
        return 1
    fi

    # Compare the current status with the expected status
    if [ "$CURRENT_STATUS" = "$expected_status" ]; then
        return 0
    else
        return 1
    fi
}

assert_vault_sealed_status() {
    expected_status="$1"
    if ! is_vault_sealed_status "$expected_status"; then
        error "Expected vault sealed status to be $expected_status, but it was not."
        exit 1
    fi
    success "Is vault sealed? $expected_status"
}

unseal_vault() {
    init_output_file="$1"
    max_unseal_attempts="$2"

    if is_vault_sealed_status "true"; then
        info "Vault is sealed. Starting unsealing process..."

        if [ -f "$init_output_file" ]; then
            info "Reading unseal keys from $init_output_file"
            unseal_keys=$(grep 'Unseal Key' "$init_output_file" | awk '{print $4}')
            for key in $unseal_keys; do
                vault operator unseal "$key"
                if is_vault_sealed_status "false"; then
                    success "Vault is unsealed."
                    break
                fi
            done
        else
            warning "Unseal keys file not found at $init_output_file. Please enter the unseal keys manually."
            i=1
            while [ "$i" -le "$max_unseal_attempts" ]; do
                printf "Enter Unseal Key %d: " "$i"
                read -r unseal_key
                vault operator unseal "$unseal_key"
                if is_vault_sealed_status "false"; then
                    success "Vault is unsealed."
                    break
                fi
                i=$((i + 1))
            done
        fi
    else
        info "Vault is already unsealed."
    fi
}

login_root() {
    init_output_file="$1"
    root_token=""

    # Attempt to read the root token from the init output file
    if [ -f "$init_output_file" ]; then
        root_token=$(grep 'Initial Root Token' "$init_output_file" | awk '{print $4}')
    fi

    # Prompt for manual entry if the token was not found in the file
    if [ -z "$root_token" ]; then
        warning "Root token not found. Please enter the root token manually."
        read -p "Enter the root token: " root_token
    fi

    # Authenticate with Vault using the root token
    info "Authenticating with Vault using the root token..."
    vault login "$root_token" || {
        error "Failed to authenticate with Vault using the root token."
        return 1
    }

    return 0
}

# Fetch the vault's role ID and secret ID
get_role_keys() {
    environment="$1"
    vault_role_id_path="/run/secrets/vrooli/$environment/vault_role_id"
    vault_secret_id_path="/run/secrets/vrooli/$environment/vault_secret_id"

    if [ ! -f "$vault_role_id_path" ]; then
        prompt_for_secret "vault_role_id"
    fi
    if [ ! -f "$vault_secret_id_path" ]; then
        prompt_for_secret "vault_secret_id"
    fi

    vault_role_id_value=$(cat "${vault_role_id_path}")
    vault_secret_id_value=$(cat "${vault_secret_id_path}")
}
