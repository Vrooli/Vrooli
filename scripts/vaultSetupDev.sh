#!/bin/bash
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

#TODO instead of calling this in develop.sh, should call in setup before setting secrets

# VAULT_ADDR specifies the address of the Vault server. It can be local or remote.
LOCAL_ADDR="http://127.0.0.1:8200"
export VAULT_ADDR="$LOCAL_ADDR" # This is a development setup, so we're assuming a local address

# Check if port 8200 is used by Vault
VAULT_PID=$(lsof -ti :8200)
# Start vault if address is local and it's not already running
if [[ -z "$VAULT_PID" && "$VAULT_ADDR" == "$LOCAL_ADDR" ]]; then
    info "Starting vault in development mode..."
    vault server -dev & # NOTE: Can stop vault using `pkill -9 vault`
    sleep 5             # Give Vault some time to start up
# If something is running at port 8200 and we're using a local address, make sure it's Vault
elif [[ ! -z "$VAULT_PID" && "$VAULT_ADDR" == "$LOCAL_ADDR" ]]; then
    if ! ps -p "$VAULT_PID" -o comm= | grep -q "vault"; then
        error "Port 8200 is in use by another process. Please ensure Vault's port isn't conflicting with other services."
        exit 1
    fi
fi

# Ensure Vault is initialized
if ! vault status | grep -Eq "Initialized\s*true"; then
    error "Vault at $VAULT_ADDR is not initialized!"
    exit 1
fi

# Ensure Vault is unsealed
if vault status | grep -Eq "Sealed\s*true"; then
    error "Vault is sealed. This typically indicates a production environment. Since this script is for setting up a development version of the vault, this may be an error. If not, please unseal the vault before continuing."
    exit 1
fi

# Setup roles, policies, etc.
if ! vault auth list | grep -q "approle/"; then
    info "Configuring Vault policies and AppRole..."
    # Enable AppRole authentication
    vault auth enable approle
    # Create a role named 'my-role' with policy 'read-all' (assuming you'd create this policy,
    # for now it just allows access to all paths in Vault) TODO create better policies later
    echo 'path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }' | vault policy write read-all -
    vault write auth/approle/role/my-role policies=read-all
    # Fetch the role_id
    ROLE_ID=$(vault read -field=role_id auth/approle/role/my-role/role-id)
    # Generate a new secret_id
    SECRET_ID=$(vault write -f -field=secret_id auth/approle/role/my-role/secret-id)
    # Store the role_id and secret_id in the specified locations
    echo "$ROLE_ID" >/run/secrets/vrooli/dev/vault_role_id
    echo "$SECRET_ID" >/run/secrets/vrooli/dev/vault_secret_id
else
    info "AppRole already configured in Vault."
fi

success "Vault setup complete!"
