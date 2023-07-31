#!/bin/sh
# Gets required secrets from secrets manager, if not already in /run/secrets.
# How to use:
# 1. Create a temporary file (e.g. `TMP_FILE=$(mktemp)`, `TMP_FILE=/tmp/secrets`, `TMP_FILE=/tmp/secrets.$$`, etc.)
# 2. Call this script: `./getSecrets.sh <environment> <tmp_file> <secret1> <secret2> ...`
# 3. Source the temporary file: `. "$TMP_FILE"`
# 4. Remove the temporary file: `rm "$TMP_FILE"`
#
# Arguments:
# - environment: Either 'development' or 'production'
# - tmp_file: The temporary file to store secrets in
# - secret1, secret2, ...: The secrets to fetch from the secret manager
#   Secrets can be renamed with the format `old_name:new_name`. If there's no `:`, the name saved in the file
#   will be the same as the name of the secret in the secret manager.
#
# NOTE: Due to incompatability with some shells and minimal images, we cannot rely on exporting secrets as environment variables.
# This is why it's written to a temporary file, which can be sourced by the parent script.
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

info "Creating or clearing temporary file: $TMP_FILE"
echo "" >>"$TMP_FILE"

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
        # Fetch the secret from the secret manager and store it in /run/secrets/
        # Note: The following is a placeholder and should be replaced with an actual command to fetch the secret
        info "Fetching $secret from the secret manager"
        # TODO Complete
        error "Fetching secrets is not yet implemented"
        exit 1
    fi
    # Store the secret in the temporary file using rename
    echo "Writing $secret to $TMP_FILE as $rename"
    echo "$rename=$(cat "/run/secrets/vrooli/$environment/$secret")" >>"$TMP_FILE"
    shift
done
