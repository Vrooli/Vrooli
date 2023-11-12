#!/bin/bash
# Sets secrets from an environment variable and .pem files into the secrets location.
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

# Set env file and vault address based on the environment
env_file="${HERE}/../.env"
VAULT_ADDR="http://127.0.0.1:8200" # Assuming development Vault runs locally
if [ "$environment" == "production" ]; then
    env_file="${HERE}/../.env-prod"
    # VAULT_ADDR="https://vault.myproductiondomain.com"  # Replace with your production Vault address TODO
fi

# Check if env file exists
if [ ! -f "$env_file" ]; then
    error "Environment file $env_file does not exist."
    exit 1
fi

# Create directories if they don't exist
mkdir -p /run/secrets/vrooli/$environment

# Detect if a vault is running, and is in the desired state (i.e. sealed for prod)
VAULT_RUNNING=false
if [ $(curl -s -o /dev/null -w "%{http_code}" $VAULT_ADDR/v1/sys/health) == "200" ]; then
    if [ "$environment" == "dev" ]; then
        VAULT_RUNNING=true
    elif [ "$environment" == "prod" ] && [ $(curl -s $VAULT_ADDR/v1/sys/seal-status | jq '.sealed') == "true" ]; then
        VAULT_RUNNING=true
    fi
fi

# Function to store file in vault
store_in_vault() {
    local key=$1
    local file_path=$2
    if [ ! -f "$file_path" ]; then
        warning "File $file_path does not exist."
        return
    fi
    local value=$(cat "$file_path")
    echo "$value" >"/run/secrets/vrooli/$environment/$key"
    if $VAULT_RUNNING; then
        vault kv put secret/vrooli/$environment/$key value="$value"
    fi
}
# Store JWT keys in vault
store_in_vault "jwt_priv" "${HERE}/../jwt_priv.pem"
store_in_vault "jwt_pub" "${HERE}/../jwt_pub.pem"

# Read lines in env file
while IFS= read -r line || [ -n "$line" ]; do
    # Ignore lines that start with '#' or are blank
    if echo "$line" | grep -q -v '^#' && [ -n "$line" ]; then
        key=$(echo "$line" | cut -d '=' -f 1)
        value=$(echo "$line" | cut -d '=' -f 2-)
        echo "$value" >"/run/secrets/vrooli/$environment/$key"

        # Set the secret in the vault if it's running TODO probably won't work for prod, since it's sealed
        if $VAULT_RUNNING; then
            vault kv put secret/vrooli/$environment/$key value="$value"
        fi
    fi
done <"$env_file"
