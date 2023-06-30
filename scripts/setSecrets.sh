#!/bin/bash
# Sets secrets from an environment variable into the secrets location.
# Useful when developing locally with Docker Compose, instead of Kubernetes.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Variable to hold environment
environment=""

# Parse options
while getopts ":e:" opt; do
    case $opt in
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

# Create directories if they don't exist
mkdir -p /run/secrets/vrooli/$environment

# Read lines in env file
while IFS= read -r line || [ -n "$line" ]; do
    # Ignore lines that start with '#' or are blank
    if echo "$line" | grep -q -v '^#' && [ -n "$line" ]; then
        key=$(echo "$line" | cut -d '=' -f 1)
        value=$(echo "$line" | cut -d '=' -f 2-)
        echo "$value" >"/run/secrets/vrooli/$environment/$key"
    fi
done <"$env_file"
