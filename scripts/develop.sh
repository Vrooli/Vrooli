#!/bin/bash
# Starts the development environment, using sensible defaults.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Default values
BUILD=""
FORCE_RECREATE=""
DOCKER_COMPOSE_FILE="docker-compose.yml"
USE_KUBERNETES=false
ENV_FILE=".env-dev"

# Read arguments
SETUP_ARGS=()
PROD_FLAG_FOUND=false
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
    -b | --build)
        BUILD="--build"
        shift # past argument
        ;;
    -f | --force-recreate)
        FORCE_RECREATE="--force-recreate"
        shift # past argument
        ;;
    -k | --kubernetes)
        USE_KUBERNETES=true
        shift # past argument
        ;;
    -p | --prod)
        PROD_FLAG_FOUND=true
        SETUP_ARGS+=("$1")
        shift # past argument
        ;;
    -h | --help)
        echo "Usage: $0 [-b BUILD] [-f FORCE_RECREATE] [-h]"
        echo "  -b --build: Build images - If set to \"y\", will build the Docker images"
        echo "  -f --force-recreate: Force recreate - If set to \"y\", will force recreate the Docker images"
        echo "  -k --kubernetes: If set, will use Kubernetes instead of Docker Compose"
        echo "  -p --prod: If set, will use the production docker-compose file (docker-compose-prod.yml)"
        echo "  -h --help: Show this help message"
        exit 0
        ;;
    *)
        SETUP_ARGS+=("$key")
        shift # past argument
        ;;
    esac
done

NODE_ENV="development"
if $PROD_FLAG_FOUND; then
    NODE_ENV="production"
    ENV_FILE=".env-prod"
    DOCKER_COMPOSE_FILE="docker-compose-prod.yml"
fi

# Running setup.sh
info "Running setup.sh..."
"${HERE}/setup.sh" -e y -r n "${SETUP_ARGS[@]}"
if [ $? -ne 0 ]; then
    error "setup.sh failed"
    exit 1
fi

info "Getting ${NODE_ENV} secrets..."
readarray -t secrets <"${HERE}/secrets_list.txt"
TMP_FILE=$(mktemp) && { "${HERE}/getSecrets.sh" ${NODE_ENV} ${TMP_FILE} "${secrets[@]}" 2>/dev/null && . "$TMP_FILE"; } || echo "Failed to get secrets."
rm "$TMP_FILE"
export DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@postgres:${PORT_DB:-5432}"
export REDIS_URL="redis://:${REDIS_PASSWORD}@redis:${PORT_REDIS:-6379}"
export WORKER_ID=0 # This is fine for single-node deployments, but should be set to the pod ordinal for multi-node deployments.
# Not sure why, but these need to be exported for the server to read them.
# This is not the case for the other secrets.
export JWT_PRIV
export JWT_PUB
export SERVER_LOCATION=$("${HERE}/domainCheck.sh" $SITE_IP $API_URL | tail -n 1)
if [ $? -ne 0 ]; then
    echo $SERVER_LOCATION
    error "Failed to determine server location"
    exit 1
fi

# If using Kubernetes, start Minikube. Otherwise, start Docker Compose
if $USE_KUBERNETES; then
    info "Starting development environment using Kubernetes..."
    if ! minikube status >/dev/null 2>&1; then
        info "Starting Minikube..."
        # NOTE: If this is failing, try running `minikube delete` and then running this script again.
        minikube start --driver=docker --force # TODO get rid of --force by running Docker in rootless mode
        if [ $? -ne 0 ]; then
            error "Failed to start Minikube"
            exit 1
        else
            success "Minikube started successfully"
        fi
        # Enable ingress
        info "Enabling ingress..."
        minikube addons enable ingress
        if [ $? -ne 0 ]; then
            error "Failed to enable ingress"
            exit 1
        else
            success "Ingress enabled successfully"
        fi
        info "Enabling ingress-dns..."
        minikube addons enable ingress-dns
        if [ $? -ne 0 ]; then
            error "Failed to enable ingress-dns"
            exit 1
        else
            success "Ingress-nginx enabled successfully"
        fi
        # Store secrets used by Kubernetes
        "${HERE}/setKubernetesSecrets.sh" -e "production" "${secrets[@]}"
        if [ $? -ne 0 ]; then
            error "Failed to set Kubernetes secrets"
            exit 1
        fi
        # TODO start the rest of the Kubernetes environment. Not sure exactly what's needed yet
    else
        success "Minikube is already running"
    fi
else
    info "Docker compose file: ${DOCKER_COMPOSE_FILE}"

    docker-compose down

    # Start the reverse proxy
    "${HERE}/proxySetup.sh"
    if [ $? -ne 0 ]; then
        error "Failed to set up proxy"
        exit 1
    fi

    # Start the development environment
    info "Starting development environment using Docker Compose and ${ENV_FILE}..."
    docker-compose --env-file "${ENV_FILE}" -f "${DOCKER_COMPOSE_FILE}" up $BUILD $FORCE_RECREATE -d
fi
