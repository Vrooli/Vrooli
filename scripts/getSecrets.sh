#!/bin/sh
# Gets required secrets from secrets manager.
# Example usage:
#
# TMP_FILE=$(mktemp) && { ./getSecrets.sh <environment> <secret1> <secret2> ... > "$TMP_FILE" 2>/dev/null && . "$TMP_FILE"; } || echo "Failed to get secrets."; rm "$TMP_FILE"
HERE=$(cd "$(dirname "$0")" && pwd)
. "${HERE}/prettify.sh"

# Check if at least two arguments were provided
if [ $# -lt 2 ]; then
    error "Not enough arguments provided. Usage: ./getSecrets.sh <environment> <tmp_file> <secret_1> <secret_2> ..."
    exit 1
fi

# Get environment
environment="$1"
shift
# Check if valid environment
if [ "$environment" = "development" ]; then
    environment="dev"
elif [ "$environment" = "production" ]; then
    environment="prod"
else
    error "Invalid environment: $environment. Expected 'development' or 'production'."
    exit 1
fi

# Set env file based on the environment
env_file="${HERE}/../.env"
if [ "$environment" == "production" ]; then
    env_file="${HERE}/../.env-prod"
fi
# Check if env file exists
if [ ! -f "$env_file" ]; then
    error "Environment file $env_file does not exist."
    exit 1
fi
# Source the env file
. "$env_file"
# Export vault address, so vault commands can be run
export VAULT_ADDR=$VAULT_ADDR

# Get temporary file
TMP_FILE="$1"
shift

info "Creating temporary file: $TMP_FILE"
echo "" >"$TMP_FILE"

# Function to prompt for a secret (that can't be fetched from Vault)
prompt_for_secret() {
    local secret_name="$1"
    echo "Please enter the value for $secret_name: "
    read -r secret_value
}

# Fetch the vault's role ID and secret ID
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

# Authenticate with Vault using AppRole
vault login -method=approle role_id="$vault_role_id_value" secret_id="$vault_secret_id_value"

# Loop over the rest of the arguments to get secrets
while [ $# -gt 0 ]; do
    # Check if secret contains ":" character for renaming
    if echo "$1" | grep -q ":"; then
        secret=$(echo "$1" | cut -d: -f1)
        rename=$(echo "$1" | cut -d: -f2)
    else
        secret="$1"
        rename="$1"
    fi

    secret_path="secret/vrooli/$environment/$secret"
    info "Fetching $secret from $secret_path in Vault"
    fetched_secret=$(vault kv get -field=value $secret_path)
    if [ -z "$fetched_secret" ]; then
        error "Failed to fetch the secret: $secret $?"
        shift
        continue
    fi

    echo "Writing $rename to $TMP_FILE"
    as_single_line=$(echo -n "$fetched_secret" | sed ':a;N;$!ba;s/\n/\\n/g')
    echo "$rename=\"$as_single_line\"" >>"$TMP_FILE"
    shift
done
