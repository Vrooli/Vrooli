#!/bin/sh
# Gets required secrets from secrets manager, if
# not already in /run/secrets
HERE=$(cd "$(dirname "$0")" && pwd)
. "${HERE}/prettify.sh"

# Variable to hold secret names
secrets=""
# Variable to hold environment
environment=""

# Parse options
while getopts ":s:e:" opt; do
    case $opt in
    s)
        secrets="$secrets $OPTARG"
        ;;
    e)
        # Check if valid environment
        if [ "$OPTARG" == "development" ]; then
            environment="dev"
        elif [ "$OPTARG" == "production" ]; then
            environment="prod"
        else
            error "Invalid environment: $OPTARG. Expected 'development' or 'production'."
            exit 1
        fi
        ;;
    \?)
        error "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    :)
        error "Option -$OPTARG requires an argument." >&2
        exit 1
        ;;
    esac
done

# Exit if no environment set
if [ -z "$environment" ]; then
    error "No environment set. Please use -e option with 'development' or 'production'."
    exit 1
fi

# Loop over the list of secrets
for secret in $secrets; do
    # Check if the secret already exists
    if [ ! -f "/run/secrets/vrooli/$environment/$secret" ]; then
        # Fetch the secret from the secret manager and store it in /run/secrets/
        # Note: The following is a placeholder and should be replaced with an actual command to fetch the secret
        info "Fetching $secret from the secret manager"
        # TODO Replace the following line with a command to fetch the secret
        echo "Placeholder secret value" >"/run/secrets/vrooli/$environment/$secret"
    fi
    # Export the secret as an environment variable
    export $secret=$(cat "/run/secrets/vrooli/$environment/$secret")
done
