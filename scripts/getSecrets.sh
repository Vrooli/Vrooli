#!/bin/sh
# Gets required secrets from secrets manager, if not already in /run/secrets.
# Usage: ./getSecrets.sh <environment> <secret1> <secret2> ...
# - environment: Either 'development' or 'production'
HERE=$(cd "$(dirname "$0")" && pwd)
. "${HERE}/prettify.sh"

# Variable to hold environment
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

# Loop over the rest of the arguments to get secrets
while [ $# -gt 0 ]; do
    secret="$1"

    # Check if the secret doesn't exist in /run/secrets
    if [ ! -f "/run/secrets/vrooli/$environment/$secret" ]; then
        # Fetch the secret from the secret manager and store it in /run/secrets/
        # Note: The following is a placeholder and should be replaced with an actual command to fetch the secret
        info "Fetching $secret from the secret manager"
        # TODO Replace the following line with a command to fetch the secret
        echo "Placeholder secret value" >"/run/secrets/vrooli/$environment/$secret"
    fi
    # Export the secret as an environment variable
    export $secret=$(cat "/run/secrets/vrooli/$environment/$secret")
    shift
done
