#!/bin/bash
# NOTE 1: Run outside of Docker container, on production server
# NOTE 2: First run build.sh on development server
# NOTE 3: If docker-compose file was changed since the last build, you should prune the containers and images before running this script.
# Finishes up the deployment process, which was started by build.sh:
# 1. Checks if Nginx containers are running
# 2. Copies current database and build to a safe location, under a temporary directory.
# 3. Runs git fetch and git pull to get the latest changes.
# 4. Runs setup.sh
# 5. Moves build created by build.sh to the correct location.
# 6. Restarts docker containers
#
# Arguments (all optional):
# -v: Version number to use (e.g. "1.0.0")
# -n: Nginx proxy location (e.g. "/root/NginxSSLReverseProxy")
# -l: Project location (e.g. "/root/Vrooli")
# -h: Show this help message
HERE=`dirname $0`
source "${HERE}/prettify.sh"

# Read arguments
while getopts ":v:d:h" opt; do
  case $opt in
    v)
      VERSION=$OPTARG
      ;;
    n)
      NGINX_LOCATION=$OPTARG
      ;;
    l)
      PROJECT_LOCATION=$OPTARG
      ;;
    h)
      echo "Usage: $0 [-v VERSION] [-d DEPLOY] [-h]"
      echo "  -v --version: Version number to use (e.g. \"1.0.0\")"
      echo "  -h --help: Show this help message"
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

# Ask for version number, if not supplied in arguments
if [ -z "$VERSION" ]; then
    echo "What version number do you want to deploy? (e.g. 1.0.0)"
    read -r VERSION
fi

# Ask for project location, if not supplied in arguments
if [ -z "$PROJECT_LOCATION" ]; then
    echo "Where is the project located? (defaults to /root/Vrooli)"
    read -r PROJECT_LOCATION
    if [ -z "$PROJECT_LOCATION" ]; then
        PROJECT_LOCATION="/root/Vrooli"
    fi
fi

# Check if nginx-proxy and nginx-proxy-le are running
if [ ! "$(docker ps -q -f name=nginx-proxy)" ] || [ ! "$(docker ps -q -f name=nginx-proxy-le)" ]; then
    error "Proxy containers are not running!"
    if [ -z "$NGINX_LOCATION" ]; then
        echo "Enter path to proxy container directory (defaults to /root/NginxSSLReverseProxy):"
        read -r NGINX_LOCATION
        if [ -z "$NGINX_LOCATION" ]; then
            NGINX_LOCATION="/root/NginxSSLReverseProxy"
        fi
    fi
    # Check if ${NGINX_LOCATION}/docker-compose.yml or ${NGINX_LOCATION}/docker-compose.yaml exists
    if [ -f "${NGINX_LOCATION}/docker-compose.yml" ] || [ -f "${NGINX_LOCATION}/docker-compose.yaml" ]; then
        # Start proxy containers
        cd "${NGINX_LOCATION}" && docker-compose up -d
    else
        error "Could not find docker-compose.yml file in ${NGINX_LOCATION}"
        exit 1
    fi
fi

# Copy current database and build to a safe location, under a temporary directory.
cd "${PROJECT_LOCATION}"
DB_TMP="/var/tmp/${VERSION}/postgres"
BUILD_TMP="/var/tmp/${VERSION}/old-build"
# Don't copy database if it already exists in /var/tmp
if [ ! -d "${DB_TMP}" ]; then
    info "Copying old database to ${DB_TMP}"
    cp -rp ./data/postgres "${DB_TMP}"
    if [ $? -ne 0 ]; then
        error "Could not copy database to ${DB_TMP}"
        exit 1
    fi
else 
    info "Old database already exists at ${DB_TMP}, so not copying it"
fi
if [ ! -d "${BUILD_TMP}" ]; then
    info "Moving old build to ${BUILD_TMP}"
    mv -f ./packages/ui/build "${BUILD_TMP}"
    if [ $? -ne 0 ]; then
        error "Could not move build to ${BUILD_TMP}"
        exit 1
    fi
else
    info "Old build already exists at ${BUILD_TMP}, so not moving it"
fi

# Stop docker containers
info "Stopping docker containers..."
docker-compose down

# Pull the latest changes from the repository.
info "Pulling latest changes from repository..."
git fetch
git pull

# Running setup.sh
info "Running setup.sh..."
./scripts/setup.sh
if [ $? -ne 0 ]; then
    error "setup.sh failed"
    exit 1
fi

# Move and decompress build created by build.sh to the correct location.
info "Moving and decompressing new build to correct location..."
rm -rf ./packages/ui/build
tar -xzf /var/tmp/${VERSION}/build.tar.gz -C ./packages/ui
if [ $? -ne 0 ]; then
    error "Could not move and decompress build to correct location"
    exit 1
fi

# Restart docker containers.
info "Restarting docker containers..."
docker-compose -f docker-compose-prod.yml up --build -d

success "Done! You may need to wait a few minutes for the Docker containers to finish starting up."