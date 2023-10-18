#!/bin/bash
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# VAULT_ADDR specifies the address of the Vault server. It can be local or remote.
LOCAL_ADDR="http://127.0.0.1:8200"
export VAULT_ADDR="$LOCAL_ADDR" # Should ideally be a remote address for production

# TODO instead of calling this in deploy.sh, should call in setup.sh before setting secrets

# Check if port 8200 is used by Vault
VAULT_PID=$(lsof -ti :8200)
# Start vault if address is local and it's not already running
if [[ -z "$VAULT_PID" && "$VAULT_ADDR" == "$LOCAL_ADDR" ]]; then
    info "Starting vault in production mode..."
    vault server -config=/path/to/your/config.hcl & # TODO need config file
    sleep 10                                        # Give Vault some time to start up
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

# Ensure Vault is sealed
if vault status | grep -Eq "Sealed\s*false"; then
    warning "Vault is not sealed! This typically indicates a development environment. Since this script is for setting up a production version of the vault, this may be an error. If not, please seal the vault before continuing."
    exit 1
fi

# Now unseal the vault
while vault status | grep -q "Sealed     true"; do
    info "Vault is sealed. Unsealing..."
    # Normally, multiple unseal commands with different unseal keys might be needed. This is a simplification.
    read -p "Enter an unseal key: " UNSEAL_KEY
    vault operator unseal "$UNSEAL_KEY"
done

# Check if we're working with a local Vault address to generate role_id and secret_id
# TODO might need check to prevent generating ids if they already exist
if [[ "$VAULT_ADDR" == "$LOCAL_ADDR" ]]; then
    info "Local Vault detected. Setting up roles and secrets..."
    vault auth enable approle
    echo 'path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }' | vault policy write read-all -
    vault write auth/approle/role/my-role policies=read-all
    ROLE_ID=$(vault read -field=role_id auth/approle/role/my-role/role-id)
    SECRET_ID=$(vault write -f -field=secret_id auth/approle/role/my-role/secret-id)
    echo "$ROLE_ID" >/run/secrets/vrooli/prod/vault_role_id
    echo "$SECRET_ID" >/run/secrets/vrooli/prod/vault_secret_id
else
    info "Remote Vault detected. Checking role and secret IDs..."
    # For a remote Vault, you'd likely have an established AppRole and just need the role_id and secret_id
    if [ ! -f "/run/secrets/vrooli/prod/vault_role_id" ] || [ ! -f "/run/secrets/vrooli/prod/vault_secret_id" ]; then
        error "Role ID and/or Secret ID missing for remote Vault!"
        read -p "Enter the role_id for AppRole 'my-role': " ROLE_ID
        read -p "Enter the secret_id for AppRole 'my-role': " SECRET_ID
        echo "$ROLE_ID" >/run/secrets/vrooli/prod/vault_role_id
        echo "$SECRET_ID" >/run/secrets/vrooli/prod/vault_secret_id
    fi
fi

info "Vault setup complete!"
