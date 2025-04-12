#!/bin/bash
# Sets up a development or production version of Hashicorp Vault,
# either directly on the host machine or in a Kubernetes cluster.
HERE=$(dirname $0)
. "${HERE}/utils.sh"
. "${HERE}/vaultTools.sh"

VAULT_PORT=8200
# Get the PID of the process running at the port, which is hopefully Vault
VAULT_PID=$(lsof -ti :$VAULT_PORT)
# Base path for secrets taken from the vault
SECRETS_PATH="/run/secrets/vrooli"

# Default values
ENVIRONMENT="dev"
USE_KUBERNETES=false
SHUTDOWN_VAULT=false

# Read arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
    -h | --help)
        echo "Usage: $0 [-h HELP] [-e ENV_SETUP] [-k KUBERNETES] [-m MODULES_REINSTALL] [-p PROD] [-r REMOTE] [-x SHUTDOWN]"
        echo "  -h --help: Show this help message"
        echo "  -k --kubernetes: If set, will use Kubernetes instead of Docker Compose"
        echo "  -p --prod: If set, will skip steps that are only required for development"
        echo "  -x --shutdown: If set, will shut down Vault instead of starting it"
        exit 0
        ;;
    -k | --kubernetes)
        USE_KUBERNETES=true
        shift # past argument
        ;;
    -p | --prod)
        ENVIRONMENT="prod"
        shift # past argument
        ;;
    -x | --shutdown)
        SHUTDOWN_VAULT=true
        shift # past argument
        ;;
    esac
done

# Set env file based on the environment
env_file="${HERE}/../.env-prod"
if [ "$ENVIRONMENT" = "dev" ]; then
    env_file="${HERE}/../.env-dev"
fi
# Check if env file exists
if [ ! -f "$env_file" ]; then
    error "Environment file $env_file does not exist."
    exit 1
fi
# Source the env file
. "$env_file"

# When running locally, make sure the port specified for Vault is actually Vault
assert_port_is_vault() {
    if [[ ! -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        if ! ps -p "$VAULT_PID" -o comm= | grep -q "vault"; then
            error "Port ${VAULT_PORT} is in use by another process. Please ensure Vault's port isn't conflicting with other services."
            exit 1
        fi
    fi
}

setup_docker_dev() {
    # Define a log file for Vault output
    local vault_log=".vault-local.log"

    # Start vault if address is local and it's not already running
    if [[ -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Starting vault..."
        vault server -dev >"$vault_log" 2>&1 &
        VAULT_PID=$!

        # Wait for Vault to start
        info "Waiting for Vault to start..."
        local max_attempts=10
        local attempt=1
        local vault_ready=0
        while [ $attempt -le $max_attempts ]; do
            if vault status >/dev/null 2>&1; then
                vault_ready=1
                break
            fi
            info "Waiting for Vault to start (attempt $attempt)..."
            sleep 1
            ((attempt++))
        done

        if [ $vault_ready -ne 1 ]; then
            error "Vault did not start within expected time."
            return 1
        fi

        success "Vault started. Logs (including root token) can be found in $vault_log"
    fi

    # Perform checks
    assert_port_is_vault # Correct port if local
    assert_vault_initialized

    # Check if Vault is in development mode and switch if necessary
    if is_vault_sealed_status "true"; then
        if prompt_confirm "Vault is sealed (production mode). This needs to be shut down to set up a development vault. Continue? (y/N): "; then
            info "Switching to development mode..."
            # Shut down Vault and restart in development mode
            shutdown_docker_prod
            success "Vault shut down."
            setup_docker_dev
            assert_vault_sealed_status "false"
        else
            warning "Vault is in production mode. This is not recommended for development environments."
            assert_vault_sealed_status "true"
            return
        fi
    fi

    # Setup roles, policies, etc. when using a local Vault address.
    # We'll assume that a remote Vault is already configured.
    if [[ ! -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        if ! vault auth list | grep -q "approle/"; then
            info "Configuring Vault policies and AppRole..."
            # Enable AppRole authentication
            vault auth enable approle
            # Create a role named 'my-role' with policy 'read-all' (assuming you'd create this policy,
            # for now it just allows access to all paths in Vault) TODO create better policies later
            echo 'path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }' | vault policy write read-all -
            vault write auth/approle/role/my-role policies=read-all
            # Fetch the role_id
            ROLE_ID=$(vault read -field=role_id auth/approle/role/my-role/role-id)
            # Generate a new secret_id
            SECRET_ID=$(vault write -f -field=secret_id auth/approle/role/my-role/secret-id)
            # Store the role_id and secret_id in the specified locations
            mkdir -p "$SECRETS_PATH/$ENVIRONMENT"
            echo "$ROLE_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_role_id"
            echo "$SECRET_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_secret_id"
        else
            info "AppRole already configured in Vault."
        fi
    fi
}

shutdown_vault_confirm() {
    warning "Shutting down Vault will delete all data stored in Vault."
    if prompt_confirm "Are you sure you want to continue? (y/n): "; then
        info "Shutting down Vault..."
    else
        info "Cancelling shutdown."
        exit 0
    fi
}

shutdown_docker_dev() {
    shutdown_vault_confirm
    # Stop Vault if address is local and it's running
    if [[ ! -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Shutting down Vault..."

        if kill -9 "$VAULT_PID"; then
            echo "Vault (PID: $VAULT_PID) shut down successfully."
        else
            echo "Failed to shut down Vault (PID: $VAULT_PID)."
            return 1
        fi

        VAULT_PID=""
        return 0
    fi
}

setup_docker_prod() {
    INIT_OUTPUT_FILE="${HERE}/../.vault-init-output.txt"
    KV_PATH="secret"

    # Start vault if address is local and it's not already running
    if [[ -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Starting vault in production mode..."
        # TODO improve config later - can add storage backend for high availability, enable tls, setting up certificate, access control, etc.
        vault server -config="${HERE}/../vault-config.hcl" &
        sleep 10 # Give Vault some time to start up
    fi

    assert_port_is_vault # Correct port if local

    # Check if Vault is initialized, if not, initialize it
    if ! is_vault_initialized; then
        info "Initializing Vault..."
        vault operator init >"$INIT_OUTPUT_FILE"
        info "Vault initialized. Unseal keys and root token are stored in vault-init-output.txt"
    fi
    assert_vault_initialized

    # Check if Vault is in development mode and switch if necessary
    if is_vault_sealed_status "false"; then
        if prompt_confirm "Vault is unsealed (development mode). This needs to be shut down to set up a production vault. Continue? (y/N): "; then
            info "Switching to production mode..."
            # Shut down Vault and restart in production mode
            shutdown_docker_dev
            success "Vault shut down."
            setup_docker_prod
            assert_vault_sealed_status "true"
        else
            warning "Vault is in development mode. This is not recommended for production environments."
            assert_vault_sealed_status "false"
            return
        fi
    fi

    # Unseal the vault
    MAX_UNSEAL_ATTEMPTS=3
    unseal_vault "$INIT_OUTPUT_FILE" "$MAX_UNSEAL_ATTEMPTS"

    # Authenticate with Vault using the root token
    login_root "$INIT_OUTPUT_FILE"

    # Check if we're working with a local Vault address to generate role_id and secret_id
    if [[ "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Local Vault detected. Setting up roles and secrets..."

        # Enable the KV secrets engine if not already
        if ! vault secrets list | grep -q "^$KV_PATH/"; then
            info "Enabling KV secrets engine at path '$KV_PATH'..."
            vault secrets enable -path="$KV_PATH" kv || {
                error "Failed to enable KV secrets engine at path '$KV_PATH'."
                return 1
            }
        else
            info "KV secrets engine is already enabled at path '$KV_PATH'."
        fi

        # Check if approle auth is already enabled
        if ! vault auth list | grep -q '^approle/'; then
            info "Enabling approle auth method..."
            vault auth enable approle
        else
            info "Approle auth method is already enabled."
        fi

        # Write policy if it doesn't exist
        if ! vault policy list | grep -q 'read-all'; then
            echo 'path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }' | vault policy write read-all -
        fi

        # Check if role exists
        if ! vault read auth/approle/role/my-role >/dev/null 2>&1; then
            vault write auth/approle/role/my-role policies=read-all
        fi

        # Fetch or generate role_id and secret_id
        ROLE_ID=$(vault read -field=role_id auth/approle/role/my-role/role-id)
        SECRET_ID=$(vault write -f -field=secret_id auth/approle/role/my-role/secret-id)

        mkdir -p "$SECRETS_PATH/$ENVIRONMENT"
        echo "$ROLE_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_role_id"
        echo "$SECRET_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_secret_id"
    else
        info "Remote Vault detected. Checking role and secret IDs..."
        # For a remote Vault, you'd likely have an established AppRole and just need the role_id and secret_id
        if [ ! -f "$SECRETS_PATH/$ENVIRONMENT/vault_role_id" ] || [ ! -f "$SECRETS_PATH/$ENVIRONMENT/vault_secret_id" ]; then
            error "Role ID and/or Secret ID missing for remote Vault!"
            read -p "Enter the role_id for AppRole 'my-role': " ROLE_ID
            read -p "Enter the secret_id for AppRole 'my-role': " SECRET_ID
            mkdir -p "$SECRETS_PATH/$ENVIRONMENT"
            echo "$ROLE_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_role_id"
            echo "$SECRET_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_secret_id"
        fi
    fi

    # Reseal the vault
    info "Resealing Vault..."
    vault operator seal

    # Confirm that the Vault is sealed
    if is_vault_sealed_status "true"; then
        success "Vault has been successfully resealed."
    else
        error "Failed to reseal Vault. Please investigate."
    fi
}

shutdown_docker_prod() {
    shutdown_vault_confirm
    # Stop Vault if address is local and it's running
    if [[ ! -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Shutting down Vault..."

        if kill -9 "$VAULT_PID"; then
            echo "Vault (PID: $VAULT_PID) shut down successfully."
        else
            echo "Failed to shut down Vault (PID: $VAULT_PID)."
            return 1
        fi

        VAULT_PID=""
        return 0
    fi
}

setup_kubernetes_dev() {
    # Define some constants
    NAMESPACE="vrooli-vault-dev"
    RELEASE_NAME="vault"

    # Check if Vault Helm chart is added, if not, add it
    VAULT_HELM="https://helm.releases.hashicorp.com"
    if ! helm repo list | grep -q "$VAULT_HELM"; then
        helm repo add hashicorp "$VAULT_HELM"
    fi

    # Update Helm repo to get the latest charts
    helm repo update

    # Check if Vault is already deployed within Kubernetes
    if ! helm list -n $NAMESPACE | grep -q $RELEASE_NAME; then
        info "Deploying Vault with Helm..."
        helm install $RELEASE_NAME hashicorp/vault --values some-values-file.yaml -n $NAMESPACE

        info "Waiting for Vault to be ready..."
        # This loop waits for the Vault pod to be ready.
        #
        # The `kubectl` command retrieves all pods in the specified namespace ($NAMESPACE) with the labels for the Helm release.
        #
        # The `-o 'jsonpath=...'` option extracts the status of the "Ready" condition from the pod's status conditions.
        # This "Ready" condition can have a status of "True", "False", or "Unknown".
        # The loop continues as long as the status is not "True".
        #
        # Thus, this loop effectively makes the script wait until the Vault pod is fully up and ready to accept traffic.
        while [[ $(kubectl get pods -n $NAMESPACE -l "app.kubernetes.io/name=vault,app.kubernetes.io/instance=$RELEASE_NAME" -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}') != "True" ]]; do
            echo "Waiting for Vault pod to be ready..."
            sleep 5
        done
    else
        info "Vault is already deployed in the namespace $NAMESPACE."
    fi

    # Initialize and unseal Vault
    if ! is_vault_initialized; then
        info "Initializing Vault..."
        vault operator init -key-shares=1 -key-threshold=1
    fi

    if is_vault_sealed_status "true"; then
        info "Unsealing Vault..."
        UNSEAL_KEY=$(vault operator init -format=json | jq -r ".unseal_keys_b64[0]") # Using jq to parse the JSON output and get the unseal key
        vault operator unseal "$UNSEAL_KEY"
    fi

    # Enable and configure AppRole
    if ! vault auth list | grep -q "approle/"; then
        info "Configuring Vault policies and AppRole..."
        vault auth enable approle
        echo 'path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }' | vault policy write read-all -
        vault write auth/approle/role/my-role policies=read-all
    else
        info "AppRole already configured in Vault."
    fi
}

shutdown_kubernetes_dev() {
    shutdown_vault_confirm
    # Define some constants
    NAMESPACE="vrooli-vault-dev"
    RELEASE_NAME="vault"

    # Check if Vault is already deployed within Kubernetes
    if helm list -n $NAMESPACE | grep -q $RELEASE_NAME; then
        info "Deleting Vault deployment..."
        helm delete $RELEASE_NAME -n $NAMESPACE
    else
        info "Vault is not deployed in the namespace $NAMESPACE."
    fi
}

setup_kubernetes_prod() {
    echo "TODO"
    # # Check if the DOKS cluster is running using doctl
    # CLUSTER_NAME="my-doks-cluster"

    # # DigitalOcean Kubernetes (DOKS) is DigitalOcean's managed Kubernetes service.
    # # With DOKS, you don't have to manage the Kubernetes control plane. Instead, you manage
    # # your workloads and worker nodes. Remember to delete your cluster when you're done
    # # (using `doctl kubernetes cluster delete <cluster-name>`) to avoid incurring unnecessary costs.

    # # Check if the DOKS cluster exists
    # if ! doctl kubernetes cluster get "$CLUSTER_NAME" >/dev/null 2>&1; then
    #     info "Creating DOKS cluster..."
    #     doctl kubernetes cluster create "$CLUSTER_NAME" --region nyc1 --version 1.21.3-do.0 --size s-2vcpu-4gb

    #     if [ $? -ne 0 ]; then
    #         error "Failed to create DOKS cluster"
    #         exit 1
    #     else
    #         success "DOKS cluster created successfully"

    #         # Configure kubectl to communicate with the DOKS cluster
    #         info "Configuring kubectl for DOKS..."
    #         doctl kubernetes cluster kubeconfig save "$CLUSTER_NAME"
    #         success "kubectl configured for DOKS"
    #     fi
    # else
    #     success "DOKS cluster is already running"
    # fi

    # # Note: When you're done with the DOKS cluster, remember to delete it to avoid extra charges.
    # # Use the command: `doctl kubernetes cluster delete <cluster-name>`
}

shutdown_kubernetes_prod() {
    shutdown_vault_confirm
    echo "TODO"
}

# Call the appropriate setup function
header "Vault setup starting..."
header "Environment: $ENVIRONMENT"
header "Kubernetes: $USE_KUBERNETES"
export VAULT_ADDR=$VAULT_ADDR
if $USE_KUBERNETES; then
    if [ "$ENVIRONMENT" = "dev" ]; then
        if $SHUTDOWN_VAULT; then
            shutdown_kubernetes_dev
        else
            setup_kubernetes_dev
        fi
    else
        if $SHUTDOWN_VAULT; then
            shutdown_kubernetes_prod
        else
            setup_kubernetes_prod
        fi
    fi
else
    if [ "$ENVIRONMENT" = "dev" ]; then
        if $SHUTDOWN_VAULT; then
            shutdown_docker_dev
        else
            setup_docker_dev
        fi
    else
        if $SHUTDOWN_VAULT; then
            shutdown_docker_prod
        else
            setup_docker_prod
        fi
    fi
fi
success "Vault setup complete!"
