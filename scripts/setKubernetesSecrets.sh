#!/bin/bash
# Script to extract AWS credentials from .env-prod and create a Kubernetes secret.
# This is the only secret that Kubernetes needs directly. All other secrets are
# handled by Hashicorp Vault.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Default values for command line options
ENVIRONMENT="production"

# Read arguments
while getopts "e:h" opt; do
    case $opt in
    e)
        ENVIRONMENT=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-e ENVIRONMENT]"
        echo "  -e --environment: Environment to use (development/production)"
        exit 0
        ;;
    \?)
        echo "Invalid option: -$OPTARG" >&2
        exit 1
        ;;
    :)
        echo "Option -$OPTARG requires an argument." >&2
        exit 1
        ;;
    esac
done

# Set environment-specific variables
if [ "$ENVIRONMENT" = "development" ]; then
    ENV_FILE="${HERE}/../.env"
    CLUSTER_NAME="vrooli-dev-cluster"
elif [ "$ENVIRONMENT" = "production" ]; then
    ENV_FILE="${HERE}/../.env-prod"
    CLUSTER_NAME="vrooli-prod-cluster"
else
    error "Invalid environment specified."
    exit 1
fi

# Source the environment variables
if [ -f "$ENV_FILE" ]; then
    . "$ENV_FILE"
else
    error "The $ENV_FILE file does not exist."
    exit 1
fi

# Check for the existence of the Kubernetes cluster
if kubectl config get-contexts | grep -q "$CLUSTER_NAME"; then
    info "Switching to cluster $CLUSTER_NAME"
    kubectl config use-context "$CLUSTER_NAME"
else
    error "Kubernetes cluster '$CLUSTER_NAME' not found."
    exit 1
fi

# Check for AWS credentials in the environment file
if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    error "AWS credentials not found in $ENV_FILE."
    exit 1
fi

# Create Kubernetes secret
info "Creating Kubernetes secret for AWS credentials"
kubectl create secret generic aws-s3-credentials \
    --from-literal=AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY_ID" \
    --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY" \
    --dry-run=client -o yaml | kubectl apply -f -

if [ $? -eq 0 ]; then
    success "Kubernetes secret created successfully."
else
    error "Failed to create Kubernetes secret."
    exit 1
fi
