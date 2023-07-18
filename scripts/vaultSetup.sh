#!/bin/sh
# Sets up Hashicorp Vault for development
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

DEPLOYMENT="vault"
IMAGE="hashicorp/vault"
PORT=8200

header "Checking if the Vault deployment exists"
if kubectl get deployments | grep -q $DEPLOYMENT; then
    info "Vault deployment already exists"
else
    info "Creating Vault deployment"
    kubectl create deployment $DEPLOYMENT --image=$IMAGE
fi

header "Setting environment variables"
kubectl set env deployment/$DEPLOYMENT VAULT_DEV_ROOT_TOKEN_ID=myroot VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200

header "Checking if the Vault service exists"
if kubectl get service | grep -q $DEPLOYMENT; then
    info "Vault service already exists"
else
    info "Exposing Vault deployment"
    kubectl expose deployment $DEPLOYMENT --type=LoadBalancer --port=8200
fi

info "Stopping existing port forwarding, if any"
kill $(pgrep -f 'kubectl port-forward')

info "Starting port forwarding"
nohup kubectl port-forward service/$DEPLOYMENT $PORT:$PORT >/dev/null 2>&1 &

success "Vault is ready to use. Visit http://localhost:$PORT to access the UI"
