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
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# Read arguments
SETUP_ARGS=()
for arg in "$@"; do
    case $arg in
    -v | --version)
        VERSION="${2}"
        shift
        shift
        ;;
    -n | --nginx-location)
        NGINX_LOCATION="${2}"
        shift
        shift
        ;;
    -h | --help)
        echo "Usage: $0 [-v VERSION] [-n NGINX_LOCATION] [-h]"
        echo "  -v --version: Version number to use (e.g. \"1.0.0\")"
        echo "  -n --nginx-location: Nginx proxy location (e.g. \"/root/NginxSSLReverseProxy\")"
        echo "  -h --help: Show this help message"
        exit 0
        ;;
    *)
        SETUP_ARGS+=("${arg}")
        shift
        ;;
    esac
done

# Ask for version number, if not supplied in arguments
if [ -z "$VERSION" ]; then
    prompt "What version number do you want to deploy? (e.g. 1.0.0). Leave blank if keeping the same version number."
    warning "WARNING: Keeping the same version number will overwrite the previous build AND database backup."
    read -r VERSION
    # If no version number was entered, use the version number found in the package.json files
    if [ -z "$VERSION" ]; then
        info "No version number entered. Using version number found in package.json files."
        VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
        info "Version number found in package.json files: ${VERSION}"
    fi
fi

# Check if nginx-proxy and nginx-proxy-le are running
if [ ! "$(docker ps -q -f name=nginx-proxy)" ] || [ ! "$(docker ps -q -f name=nginx-proxy-le)" ]; then
    error "Proxy containers are not running!"
    if [ -z "$NGINX_LOCATION" ]; then
        prompt "Enter path to proxy container directory (defaults to /root/NginxSSLReverseProxy):"
        read -r NGINX_LOCATION
        if [ -z "$NGINX_LOCATION" ]; then
            NGINX_LOCATION="/root/NginxSSLReverseProxy"
        fi
    fi
    # Check if ${NGINX_LOCATION}/docker-compose.yml or ${NGINX_LOCATION}/docker-compose.yaml exists
    if [ -f "${NGINX_LOCATION}/docker-compose.yml" ] || [ -f "${NGINX_LOCATION}/docker-compose.yaml" ]; then
        info "Starting proxy containers..."
        cd "${NGINX_LOCATION}" && docker-compose up -d
    else
        error "Could not find docker-compose.yml file in ${NGINX_LOCATION}"
        exit 1
    fi
fi

# Copy current database and build to a safe location, under a temporary directory.
cd ${HERE}/..
DB_TMP="/var/tmp/${VERSION}/postgres"
DB_CURR="${HERE}/../data/postgres"
BUILD_TMP="/var/tmp/${VERSION}/old-build"
BUILD_CURR="${HERE}/../packages/ui/dist"
# Don't copy database if it already exists in /var/tmp, or it doesn't exist in DB_CURR
if [ -d "${DB_TMP}" ]; then
    info "Old database already exists at ${DB_TMP}, so not copying it"
elif [ ! -d "${DB_CURR}" ]; then
    warning "Current database does not exist at ${DB_CURR}, so not copying it"
else
    info "Copying old database to ${DB_TMP}"
    cp -rp ${HERE}/../data/postgres "${DB_TMP}"
    if [ $? -ne 0 ]; then
        error "Could not copy database to ${DB_TMP}"
        exit 1
    fi
fi

# Stash old build if it doesn't already exists in /var/tmp.
if [ -d "${BUILD_TMP}" ]; then
    info "Old build already exists at ${BUILD_TMP}, so not moving it"
elif [ -d "${BUILD_CURR}" ]; then
    info "Moving old build to ${BUILD_TMP}"
    mv -f "${BUILD_CURR}" "${BUILD_TMP}"
    if [ $? -ne 0 ]; then
        error "Could not move build to ${BUILD_TMP}"
        exit 1
    fi
fi

# Extract the zipped build created by build.sh
BUILD_ZIP="/var/tmp/${VERSION}"
if [ -f "${BUILD_ZIP}/build.tar.gz" ]; then
    info "Extracting build at ${BUILD_ZIP}/build.tar.gz"
    mkdir -p "${BUILD_CURR}"
    tar -xzf "${BUILD_ZIP}/build.tar.gz" -C "${BUILD_CURR}" --strip-components=1
    if [ $? -ne 0 ]; then
        error "Failed to extract build at ${BUILD_ZIP}/build.tar.gz"
        exit 1
    fi
else
    error "Could not find build at ${BUILD_ZIP}/build.tar.gz"
    exit 1
fi

# Transfer and load Docker images
if [ -f "${BUILD_ZIP}/production-docker-images.tar.gz" ]; then
    info "Loading Docker images from ${BUILD_ZIP}/production-docker-images.tar.gz"
    docker load -i "${BUILD_ZIP}/production-docker-images.tar.gz"
    if [ $? -ne 0 ]; then
        error "Failed to load Docker images from ${BUILD_ZIP}/production-docker-images.tar.gz"
        exit 1
    fi
else
    error "Could not find Docker images archive at ${BUILD_ZIP}/production-docker-images.tar.gz"
    exit 1
fi

# Stop docker containers
info "Stopping docker containers..."
docker-compose --env-file ${BUILD_ZIP}/.env-prod down

# Pull the latest changes from the repository.
info "Pulling latest changes from repository..."
git fetch
git pull
if [ $? -ne 0 ]; then
    error "Could not pull latest changes from repository. You likely have uncommitted changes."
    exit 1
fi

# Running setup.sh
info "Running setup.sh..."
"${HERE}/setup.sh" "${SETUP_ARGS[@]}" -p
if [ $? -ne 0 ]; then
    error "setup.sh failed"
    exit 1
fi

# Move and decompress build created by build.sh to the correct location.
info "Moving and decompressing new build to correct location..."
rm -rf ${HERE}/../packages/ui/dist
tar -xzf /var/tmp/${VERSION}/build.tar.gz -C ${HERE}/../packages/ui
if [ $? -ne 0 ]; then
    error "Could not move and decompress build to correct location"
    exit 1
fi

# Restart docker containers.
info "Restarting docker containers..."
docker-compose --env-file ${BUILD_ZIP}/.env-prod -f ${HERE}/../docker-compose-prod.yml up -d

success "Done! You may need to wait a few minutes for the Docker containers to finish starting up."
info "Now that you've deployed, here are some next steps:"
info "Manually check that the site is working correctly"
info "Upload the sitemap index file from packages/ui/public/sitemap.xml to Google Search Console, Bing Webmaster Tools, and Yandex Webmaster Tools"
info "Let everyone on social media know that you've deployed a new version of Vrooli!"
