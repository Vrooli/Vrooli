#!/bin/bash
# Sets up a development or production version of Hashicorp Vault,
# either directly on the host machine or in a Kubernetes cluster.
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

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
for arg in "$@"; do
    case $arg in
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
        shift
        ;;
    -p | --prod)
        ENVIRONMENT="prod"
        shift
        ;;
    -x | --shutdown)
        SHUTDOWN_VAULT=true
        shift
        ;;
    esac
done

# Source .env or .env-prod file
if [ "$ENVIRONMENT" = "dev" ]; then
    if [ ! -f "${HERE}/../.env" ]; then
        error "Environment file ${HERE}/../.env does not exist."
        exit 1
    fi
    . "${HERE}/../.env"
else
    if [ ! -f "${HERE}/../.env-prod" ]; then
        error "Environment file ${HERE}/../.env-prod does not exist."
        exit 1
    fi
    . "${HERE}/../.env-prod"
fi

# When running locally, make sure the port specified for Vault is actually Vault
assert_port_is_vault() {
    if [[ ! -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        if ! ps -p "$VAULT_PID" -o comm= | grep -q "vault"; then
            error "Port ${VAULT_PORT} is in use by another process. Please ensure Vault's port isn't conflicting with other services."
            exit 1
        fi
    fi
}

# Checks if Vault is initialized.
is_vault_initialized() {
    OUTPUT=$(vault status 2>&1)
    EXIT_STATUS=$?

    if [ $EXIT_STATUS -ne 0 ]; then
        error "Failed to get Vault status. Error: $OUTPUT"
        return 1
    elif ! echo "$OUTPUT" | grep -Eq "Initialized\s*true"; then
        error "Vault at $VAULT_ADDR is not initialized!"
        return 1
    else
        return 0
    fi
}
assert_vault_initialized() {
    if ! is_vault_initialized; then
        error "Vault at $VAULT_ADDR is not initialized!"
        exit 1
    fi
    success "Vault is initialized."
}

# Checks if the Vault is either sealed or unsealed, depending on the expected status.
is_vault_sealed_status() {
    EXPECTED_STATUS="$1" # Either "true" or "false"

    OUTPUT=$(vault status 2>&1)
    EXIT_STATUS=$?

    if [ $EXIT_STATUS -ne 0 ]; then
        error "Failed to get Vault status. Error: $OUTPUT"
        return 1
    elif [ "$EXPECTED_STATUS" = "true" ] && echo "$OUTPUT" | grep -Eq "Sealed\s*false"; then
        error "Vault is not sealed. This typically indicates a development environment. Please ensure it's sealed before continuing."
        return 1
    elif [ "$EXPECTED_STATUS" = "false" ] && echo "$OUTPUT" | grep -Eq "Sealed\s*true"; then
        error "Vault is sealed. This typically indicates a production environment. Please ensure it's unsealed before continuing."
        return 1
    else
        return 0
    fi
}
assert_vault_sealed_status() {
    EXPECTED_STATUS="$1"
    if ! is_vault_sealed_status "$EXPECTED_STATUS"; then
        error "Vault is not $EXPECTED_STATUS!"
        exit 1
    fi
    success "Is vault sealed? $EXPECTED_STATUS"
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
    assert_vault_sealed_status "false" # Unsealed

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
            echo "$ROLE_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_role_id"
            echo "$SECRET_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_secret_id"
        else
            info "AppRole already configured in Vault."
        fi
    fi
}

shutdown_vault_confirm() {
    warning "Shutting down Vault will delete all data stored in Vault."
    prompt "Are you sure you want to continue? (y/n)"
    read -n1 -r CONFIRM
    echo

    if [[ "$CONFIRM" =~ ^[Yy]([Ee][Ss])?$ ]]; then
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
        pkill -9 vault
    fi
}

setup_docker_prod() {
    # Start vault if address is local and it's not already running
    if [[ -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Starting vault in production mode..."
        vault server -config=/path/to/your/config.hcl & # TODO need config file
        sleep 10                                        # Give Vault some time to start up
    fi

    # Perform checks
    assert_port_is_vault # Correct port if local
    assert_vault_initialized
    assert_vault_sealed_status "true" # Sealed

    # Now unseal the vault
    while vault status | grep -Eq "Sealed\s*true"; do
        info "Vault is sealed. Unsealing..."
        # Normally, multiple unseal commands with different unseal keys might be needed. This is a simplification.
        read -p "Enter an unseal key: " UNSEAL_KEY
        vault operator unseal "$UNSEAL_KEY"
    done

    # Check if we're working with a local Vault address to generate role_id and secret_id
    # TODO might need check to prevent generating ids if they already exist
    if [[ "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Local Vault detected. Setting up roles and secrets..."
        vault auth enable approle
        echo 'path "*" { capabilities = ["create", "read", "update", "delete", "list", "sudo"] }' | vault policy write read-all -
        vault write auth/approle/role/my-role policies=read-all
        ROLE_ID=$(vault read -field=role_id auth/approle/role/my-role/role-id)
        SECRET_ID=$(vault write -f -field=secret_id auth/approle/role/my-role/secret-id)
        echo "$ROLE_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_role_id"
        echo "$SECRET_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_secret_id"
    else
        info "Remote Vault detected. Checking role and secret IDs..."
        # For a remote Vault, you'd likely have an established AppRole and just need the role_id and secret_id
        if [ ! -f "$SECRETS_PATH/$ENVIRONMENT/vault_role_id" ] || [ ! -f "$SECRETS_PATH/$ENVIRONMENT/vault_secret_id" ]; then
            error "Role ID and/or Secret ID missing for remote Vault!"
            read -p "Enter the role_id for AppRole 'my-role': " ROLE_ID
            read -p "Enter the secret_id for AppRole 'my-role': " SECRET_ID
            echo "$ROLE_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_role_id"
            echo "$SECRET_ID" >"$SECRETS_PATH/$ENVIRONMENT/vault_secret_id"
        fi
    fi
}

shutdown_docker_prod() {
    shutdown_vault_confirm
    # Stop Vault if address is local and it's running
    if [[ ! -z "$VAULT_PID" && "$VAULT_ADDR" == "$VAULT_ADDR_LOCAL" ]]; then
        info "Shutting down Vault..."
        pkill -9 vault
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
info "Vault setup starting..."
info "Environment: $ENVIRONMENT"
info "Kubernetes: $USE_KUBERNETES"
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

# #!/bin/sh
# # Sets up Hashicorp Vault for development
# HERE=$(dirname $0)
# . "${HERE}/prettify.sh"

# DEPLOYMENT="vault"
# IMAGE="hashicorp/vault"
# PORT=8200

# header "Checking if the Vault deployment exists"
# if kubectl get deployments | grep -q $DEPLOYMENT; then
#     info "Vault deployment already exists"
# else
#     info "Creating Vault deployment"
#     kubectl create deployment $DEPLOYMENT --image=$IMAGE
# fi

# header "Setting environment variables"
# kubectl set env deployment/$DEPLOYMENT VAULT_DEV_ROOT_TOKEN_ID=myroot VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200

# header "Checking if the Vault service exists"
# if kubectl get service | grep -q $DEPLOYMENT; then
#     info "Vault service already exists"
# else
#     info "Exposing Vault deployment"
#     kubectl expose deployment $DEPLOYMENT --type=LoadBalancer --port=8200
# fi

# info "Stopping existing port forwarding, if any"
# kill $(pgrep -f 'kubectl port-forward')

# info "Starting port forwarding"
# nohup kubectl port-forward service/$DEPLOYMENT $PORT:$PORT >/dev/null 2>&1 &

# success "Vault is ready to use. Visit http://localhost:$PORT to access the UI"
