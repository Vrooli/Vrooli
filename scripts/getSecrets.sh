#!/bin/sh
# Gets required secrets from secrets manager, if not already in /run/secrets.
# Example usage:
#
# TMP_FILE=$(mktemp) && { ./getSecrets.sh <environment> <secret1> <secret2> ... > "$TMP_FILE" 2>/dev/null && . "$TMP_FILE"; } || echo "Failed to get secrets."; rm "$TMP_FILE"
HERE=$(cd "$(dirname "$0")" && pwd)
. "${HERE}/prettify.sh"

# Check if at least two arguments were provided
if [ $# -lt 2 ]; then
    error "Not enough provided. Usage: ./getSecrets.sh <environment> <tmp_file> <secret_1> <secret_2> ..."
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

# Get temporary file
TMP_FILE="$1"
shift

info "Creating temporary file: $TMP_FILE"
echo "" >"$TMP_FILE"

# Function to prompt for a secret
prompt_for_secret() {
    local secret_name="$1"
    local secret_path="/run/secrets/vrooli/$environment/$secret_name"

    # If the secret already exists, skip prompting
    if [ -f "$secret_path" ]; then
        info "Secret $secret_name already exists. Skipping prompt."
        return
    fi

    echo "Please enter the value for $secret_name: "
    read -r secret_value

    echo "$secret_value" >"$secret_path"
}

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

    # Check if the secret doesn't exist in /run/secrets
    if [ ! -f "/run/secrets/vrooli/$environment/$secret" ]; then
        # To fetch secrets, we need the vault's role ID and secret ID
        if [ "$secret" = "vault_role_id" ] || [ "$secret" = "vault_secret_id" ]; then
            # Prompt the user for the secret
            prompt_for_secret "$secret"
        else
            # For any other secret, authenticate with Vault and fetch it
            info "Fetching $secret from Vault"

            # Ensure vault_role_id and vault_secret_id are present in /run/secrets
            if [ ! -f "/run/secrets/vrooli/$environment/vault_role_id" ] || [ ! -f "/run/secrets/vrooli/$environment/vault_secret_id" ]; then
                error "vault_role_id and vault_secret_id must be present in /run/secrets to fetch other secrets."
                exit 1
            fi

            vault_role_id_value=$(cat "/run/secrets/vrooli/$environment/vault_role_id")
            vault_secret_id_value=$(cat "/run/secrets/vrooli/$environment/vault_secret_id")

            # Authenticate with Vault
            vault login -method=approle role_id="$vault_role_id_value" secret_id="$vault_secret_id_value"

            # Fetch the secret
            fetched_secret=$(vault kv get -field="$rename" secret/$environment/$secret)

            if [ -z "$fetched_secret" ]; then
                error "Failed to fetch the secret: $secret"
                exit 1
            else
                # Store the fetched secret in /run/secrets/
                echo "$fetched_secret" >"/run/secrets/vrooli/$environment/$secret"
            fi
        fi
    fi
    echo "Writing $rename to $TMP_FILE"
    # Count the number of lines in the file
    line_count=$(wc -l <"/run/secrets/vrooli/$environment/$secret")
    # Multi-line files should be treated like PEM files
    if [ "$line_count" -gt 1 ]; then
        # For PEM files (multi-line): Remove BEGIN/END lines and convert newlines to \n
        echo "$rename=$(sed -ne '/-BEGIN /,/-END /p' "/run/secrets/vrooli/$environment/$secret" | sed -e '/-BEGIN /d' -e '/-END /d' | awk 'NR > 1 { printf "\\n" } { printf "%s", $0 }')" >>"$TMP_FILE"
    else
        # For single-line files: Directly write the content
        echo "$rename=$(cat "/run/secrets/vrooli/$environment/$secret")" >>"$TMP_FILE"
    fi
    shift
done
