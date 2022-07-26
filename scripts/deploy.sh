#!/bin/bash
# NOTE 1: Run outside of Docker container, on production server
# NOTE 2: First run build.sh on development server
# NOTE 3: If docker-compose was changed since the last build, you should prune the containers and images before running this script.
# Finishes up the deployment process, which was started by build.sh:
# 1. Checks if Nginx containers are running
# 2. Copies current database and build to a safe location, under a temporary directory.
# 3. Runs git fetch and git pull to get the latest changes.
# 4. Runs yarn cache clean and yarn to make sure that node_modules are up to date.
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
info "Copying current database and build to a safe location, under the same temporary directory that the new build is using..."
cd "${PROJECT_LOCATION}"
cp -rp ./data/postgres /tmp/${VERSION}
cp -rp ./packages/ui/build/* /tmp/${VERSION}/old-build

# Stop docker containers
info "Stopping docker containers..."
docker-compose down

# Pull the latest changes from the repository.
info "Pulling latest changes from repository..."
git fetch
git pull

# Run yarn cache clean and yarn to make sure that node_modules are up to date.
info "Cleaning yarn cache..."
yarn cache clean
info "Running yarn..."
yarn

# Move build created by build.sh to the correct location.
info "Moving new build to correct location..."
rm -rf ./packages/ui/build
cp -rp /tmp/${VERSION}/build ./packages/ui/build

# Restart docker containers.
info "Restarting docker containers..."
docker-compose up --build -d

success "Done! You may need to wait a few minutes for the Docker containers to finish starting up."