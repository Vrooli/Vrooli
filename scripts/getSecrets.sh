#!/bin/sh
# Gets required secrets from secrets manager.
# Example usage:
#
# TMP_FILE=$(mktemp) && { ./getSecrets.sh <environment> <secret1> <secret2> ... > "$TMP_FILE" 2>/dev/null && . "$TMP_FILE"; } || echo "Failed to get secrets."; rm "$TMP_FILE"
HERE=$(cd "$(dirname "$0")" && pwd)
. "${HERE}/utils.sh"
. "${HERE}/vaultTools.sh"

# Check if at least two arguments were provided
if [ $# -lt 2 ]; then
    error "Not enough arguments provided. Usage: ./getSecrets.sh <environment> <tmp_file> <secret_1> <secret_2> ..."
    exit 1
fi

# Get environment
environment="$1"
shift
case $environment in
[Dd]ev*)
    environment="dev"
    ;;
[Pp]rod*)
    environment="prod"
    ;;
*)
    error "Invalid environment: $environment. Expected 'dev' or 'prod'."
    exit 1
    ;;
esac

# Set env file based on the environment
env_file="${HERE}/../.env"
if [ "$environment" == "prod" ]; then
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

# Check if Vault is initialized and unsealed
assert_vault_initialized
STARTED_SEALED="false"
if is_vault_sealed_status "true"; then
    STARTED_SEALED="true"
    INIT_OUTPUT_FILE="${HERE}/../.vault-init-output.txt"
    unseal_vault "$INIT_OUTPUT_FILE" 3
fi

reseal() {
    if [ "$STARTED_SEALED" = "true" ]; then
        info "Resealing Vault"
        vault operator seal
    fi
}

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

# Authenticate with Vault using AppRole
get_role_keys "$environment"
vault write auth/approle/login role_id="$vault_role_id_value" secret_id="$vault_secret_id_value"
login_status=$?
if [ $login_status -ne 0 ]; then
    error "Failed to authenticate with Vault using AppRole: $login_status"
    reseal
    exit 1
fi

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
    fetch_status=$?
    if [ $fetch_status -ne 0 ]; then
        error "Failed to fetch the secret $secret. Got status $fetch_status"
        shift
        continue
    elif [ -z "$fetched_secret" ]; then
        error "Secret $secret is empty."
        shift
        continue
    fi

    echo "Writing $rename to $TMP_FILE"
    as_single_line=$(echo -n "$fetched_secret" | sed ':a;N;$!ba;s/\n/\\n/g')
    echo "$rename=\"$as_single_line\"" >>"$TMP_FILE"
    shift
done

reseal
