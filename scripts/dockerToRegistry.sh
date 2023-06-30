#!/bin/bash
# Adds production docker images to the Docker Hub registry
# TODO NOTE: DO NOT RUN THIS YET. We need to implement a secret manager to store sensitive environment variables
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"
. "${HERE}/../.env"

# Default values for command line options
BUILD="y"

# Read arguments
while getopts "b:hv:" opt; do
    case $opt in
    b)
        BUILD=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-b BUILD] [-h] [-v VERSION]"
        echo "  -b --build: Build Docker images (y/N)"
        echo "  -h --help: Show this help message"
        echo "  -v --version: Version number to use (e.g. \"1.0.0\")"
        exit 0
        ;;
    v)
        VERSION=$OPTARG
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

# Extract the current version number from the ui package.json file
CURRENT_VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
# Ask for version number, if not supplied in arguments
if [ -z "$VERSION" ]; then
    prompt "What version number do you want to use? (current is ${CURRENT_VERSION}). Leave blank if keeping the same version number."
    warning "WARNING: Keeping the same version number will overwrite the previous image."
    read -r ENTERED_VERSION
    echo
    # If version entered, set version
    if [ ! -z "$ENTERED_VERSION" ]; then
        VERSION=$ENTERED_VERSION
    else
        info "Keeping the same version number."
        VERSION=$CURRENT_VERSION
    fi
fi

# Login to Docker Hub
info "Logging into Docker Hub"
docker login -u $DOCKER_USERNAME -p $DOCKER_PASSWORD

# Build the Docker images
if [ "$BUILD" = "y" ] || [ "$BUILD" = "Y" ] || [ "$BUILD" = "yes" ] || [ "$BUILD" = "Yes" ] || [ "$BUILD" = "YES" ]; then
    cd ${HERE}/..
    info "Building Docker images"
    docker-compose --env-file .env-prod -f docker-compose-prod.yml build
    if [ $? -ne 0 ]; then
        error "Failed to build Docker images"
        exit 1
    fi
    cd ${HERE}
fi

# Tag the Docker images
info "Tagging Docker images"
docker tag ui:prod $DOCKER_USERNAME/vrooli_ui:$VERSION
docker tag server:prod $DOCKER_USERNAME/vrooli_server:$VERSION
docker tag docs:prod $DOCKER_USERNAME/vrooli_docs:$VERSION

# Push the Docker images to Docker Hub
info "Pushing Docker images to Docker Hub"
docker push $DOCKER_USERNAME/vrooli_ui:$VERSION
docker push $DOCKER_USERNAME/vrooli_server:$VERSION
docker push $DOCKER_USERNAME/vrooli_docs:$VERSION

success "Docker images pushed successfully"
