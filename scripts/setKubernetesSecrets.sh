#!/bin/bash
# Script to create Kubernetes secrets based on passed secret names.
# Secrets are retrieved either from /run/secrets or via getSecrets.sh.

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Default values for command line options
ENVIRONMENT="production"

# Read arguments for environment, others are considered as secret names
while getopts "e:h" opt; do
    case $opt in
    e)
        ENVIRONMENT=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-e ENVIRONMENT] secret1 secret2 ..."
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
shift $((OPTIND - 1))

# Check for Kubernetes cluster presence
CLUSTER_NAME="vrooli-${ENVIRONMENT}-cluster"
if ! kubectl config get-contexts | grep -q "$CLUSTER_NAME"; then
    error "Kubernetes cluster '$CLUSTER_NAME' not found."
    exit 1
fi
kubectl config use-context "$CLUSTER_NAME"

# Function to create or update a Kubernetes secret
create_or_update_k8s_secret() {
    local secret_name="$1"
    local secret_value="$2"
    kubectl create secret generic "${secret_name}" \
        --from-literal="${secret_name}=${secret_value}" \
        --dry-run=client -o yaml | kubectl apply -f -
}

# Process each secret passed as argument
for secret in "$@"; do
    TMP_FILE=$(mktemp)
    if ./getSecrets.sh "$ENVIRONMENT" "$TMP_FILE" "$secret" 2>/dev/null; then
        . "$TMP_FILE"
        secret_value=${!secret}
        create_or_update_k8s_secret "$secret" "$secret_value"
    else
        error "Failed to retrieve secret: $secret"
        rm "$TMP_FILE"
        exit 1
    fi
    rm "$TMP_FILE"
done
