#!/bin/bash
# Starts the development environment, using sensible defaults.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Default values
BUILD=""
FORCE_RECREATE=""
DOCKER_COMPOSE_FILE="-f docker-compose.yml"
USE_KUBERNETES=false

# Read arguments
SETUP_ARGS=()
PROD_FLAG_FOUND=false
for arg in "$@"; do
    case $arg in
    -b | --build)
        BUILD="--build"
        shift
        ;;
    -f | --force-recreate)
        FORCE_RECREATE="--force-recreate"
        shift
        ;;
    -k | --kubernetes)
        USE_KUBERNETES=true
        shift
        ;;
    -p | --prod)
        PROD_FLAG_FOUND=true
        SETUP_ARGS+=("$1")
        shift
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
        SETUP_ARGS+=("${arg}")
        shift
        ;;
    esac
done

NODE_ENV="development"
if $PROD_FLAG_FOUND; then
    NODE_ENV="production"
fi

# Running setup.sh
info "Running setup.sh..."
. "${HERE}/setup.sh" -e y -r n "${SETUP_ARGS[@]}"
if [ $? -ne 0 ]; then
    error "setup.sh failed"
    exit 1
fi

info "Getting ${NODE_ENV} secrets..."
readarray -t secrets <"${HERE}/secrets_list.txt"
TMP_FILE=$(mktemp) && { "${HERE}/getSecrets.sh" ${NODE_ENV} ${TMP_FILE} "${secrets[@]}" 2>/dev/null && . "$TMP_FILE"; } || echo "Failed to get secrets."
rm "$TMP_FILE"
export DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@db:${PORT_DB:-5432}"
export REDIS_URL="redis://:${REDIS_PASSWORD}@redis:${PORT_REDIS:-6379}"
# Not sure why, but these need to be exported for the server to read them.
# This is not the case for the other secrets.
export JWT_PRIV
export JWT_PUB
export SERVER_LOCATION=$("${HERE}/domainCheck.sh" $SITE_IP $SERVER_URL | tail -n 1)
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
    if $PROD_FLAG_FOUND; then
        DOCKER_COMPOSE_FILE="-f docker-compose-prod.yml"
    fi
    info "Docker compose file: ${DOCKER_COMPOSE_FILE}"

    docker-compose down

    # If server is not local, set up reverse proxy
    if [[ "$SERVER_LOCATION" != "local" ]]; then
        . "${HERE}/proxySetup.sh" -n "${NGINX_LOCATION}"
    fi

    # Start the development environment
    info "Starting development environment using Docker Compose..."
    docker-compose $DOCKER_COMPOSE_FILE up $BUILD $FORCE_RECREATE -d
fi
