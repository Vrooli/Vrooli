#!/bin/bash
# NOTE: Run outside of Docker container
# Prepares project for deployment to VPS:
# 1. Asks for version number, and updates all package.json files accordingly.
# 2. Builds the React app, making sure to include environment variables and post-build commands.
# 3. Builds all Docker containers, making sure to include environment variables and post-build commands.
# 3. Copies the tarballs for the React app and Docker containers to the VPS.
#
# Arguments (all optional):
# -v: Version number to use (e.g. "1.0.0")
# -d: Deploy to VPS (y/N)
# -h: Show this help message
# -a: Generate computed API information (GraphQL query/mutation selectors and OpenAPI schema)
# -e: .env file location (e.g. "/root/my-folder/.env"). Defaults to .env-prod
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Read arguments
ENV_FILE="${HERE}/../.env-prod"
while getopts "v:d:ha:e:" opt; do
    case $opt in
    v)
        VERSION=$OPTARG
        ;;
    d)
        DEPLOY=$OPTARG
        ;;
    a)
        API_GENERATE=$OPTARG
        ;;
    e)
        ENV_FILE=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-v VERSION] [-d DEPLOY] [-h] [-a API_GENERATE] [-e ENV_FILE]"
        echo "  -v --version: Version number to use (e.g. \"1.0.0\")"
        echo "  -d --deploy: Deploy to VPS (y/N)"
        echo "  -h --help: Show this help message"
        echo "  -a --api-generate: Generate computed API information (GraphQL query/mutation selectors and OpenAPI schema)"
        echo "  -e --env-file: .env file location (e.g. \"/root/my-folder/.env\")"
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

# Load variables from .env file
if [ -f "${ENV_FILE}" ]; then
    info "Loading variables from ${ENV_FILE}..."
    . "${ENV_FILE}"
else
    error "Could not find .env file at ${ENV_FILE}. Exiting..."
    exit 1
fi

# Check for required variables
check_var() {
    if [ -z "${!1}" ]; then
        error "Variable ${1} is not set. Exiting..."
        exit 1
    else
        info "Variable ${1} is set to ${!1}"
    fi
}
check_var SERVER_LOCATION
check_var PORT_SERVER
check_var SERVER_URL
check_var SITE_IP
check_var VAPID_PUBLIC_KEY
check_var GOOGLE_TRACKING_ID

# Extract the current version number from the package.json file
CURRENT_VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
# Ask for version number, if not supplied in arguments
SHOULD_UPDATE_VERSION=false
if [ -z "$VERSION" ]; then
    prompt "What version number do you want to deploy? (current is ${CURRENT_VERSION}). Leave blank if keeping the same version number."
    warning "WARNING: Keeping the same version number will overwrite the previous build."
    read -r ENTERED_VERSION
    # If version entered, set version
    if [ ! -z "$ENTERED_VERSION" ]; then
        VERSION=$ENTERED_VERSION
        SHOULD_UPDATE_VERSION=true
    else
        info "Keeping the same version number."
        VERSION=$CURRENT_VERSION
    fi
else
    SHOULD_UPDATE_VERSION=true
fi

# Update package.json files for every package, if necessary
if [ "${SHOULD_UPDATE_VERSION}" = true ]; then
    cd ${HERE}/../packages
    # Find every directory containing a package.json file, up to 3 levels deep
    for dir in $(find . -maxdepth 3 -name package.json -printf '%h '); do
        info "Updating package.json for ${dir}"
        # Go to directory
        cd ${dir}
        # Patch with yarn
        yarn version patch --new-version ${VERSION} --no-git-tag-version
        # Go back to packages directory
        cd ${HERE}/../packages
    done
fi

# Navigate to server directory
cd ${HERE}/../packages/server

# Build shared
"${HERE}/shared.sh"

# Build server
info "Building server..."
yarn build
if [ $? -ne 0 ]; then
    error "Failed to build server"
    exit 1
fi

# Navigate to UI directory
cd ${HERE}/../packages/ui

# Create local .env file
touch .env
# Set environment variables
echo "VITE_SERVER_LOCATION=${SERVER_LOCATION}" >>.env
echo "VITE_PORT_SERVER=${PORT_SERVER}" >>.env
echo "VITE_SERVER_URL=${SERVER_URL}" >>.env
echo "VITE_SITE_IP=${SITE_IP}" >>.env
echo "VITE_VAPID_PUBLIC_KEY=${VAPID_PUBLIC_KEY}" >>.env
echo "VITE_GOOGLE_ADSENSE_PUBLISHER_ID=${GOOGLE_ADSENSE_PUBLISHER_ID}" >>.env
echo "VITE_GOOGLE_TRACKING_ID=${GOOGLE_TRACKING_ID}" >>.env
# Set trap to remove .env file on exit
trap "rm ${HERE}/../packages/ui/.env" EXIT

# Generate query/mutation selectors
if [ -z "$API_GENERATE" ]; then
    prompt "Do you want to regenerate computed API information (GraphQL query/mutation selectors and OpenAPI schema)? (y/N)"
    read -n1 -r API_GENERATE
    echo
fi
if [ "${API_GENERATE}" = "y" ] || [ "${API_GENERATE}" = "Y" ] || [ "${API_GENERATE}" = "yes" ] || [ "${API_GENERATE}" = "Yes" ]; then
    info "Generating GraphQL query/mutation selectors... (this may take a minute)"
    NODE_OPTIONS="--max-old-space-size=4096" && ts-node --esm --experimental-specifier-resolution node ./src/tools/api/gqlSelects.ts
    if [ $? -ne 0 ]; then
        error "Failed to generate query/mutation selectors"
        echo "${HERE}/../packages/ui/src/tools/api/gqlSelects.ts"
        # This IS a critical error, so we'll exit
        exit 1
    fi
    info "Generating OpenAPI schema..."
    cd ${HERE}/../packages/shared
    NODE_OPTIONS="--max-old-space-size=4096" && ts-node --esm --experimental-specifier-resolution node ./src/tools/gqlToJson.ts
    if [ $? -ne 0 ]; then
        error "Failed to generate OpenAPI schema"
        echo "${HERE}/../packages/shared/src/tools/gqlToJson.ts"
        # This is NOT a critical error, so we'll continue
    fi
    cd ${HERE}/../packages/ui
fi

# Build React app
info "Building React app..."
yarn build
if [ $? -ne 0 ]; then
    error "Failed to build React app"
    exit 1
fi

# Generate sitemap.xml
ts-node --esm --experimental-specifier-resolution node ./src/tools/sitemap.ts
if [ $? -ne 0 ]; then
    error "Failed to generate sitemap.xml"
    echo "${HERE}/../packages/ui/src/tools/sitemap.ts"
    # This is not a critical error, so we don't exit
fi

# Create brave-rewards-verification.txt file
if [ -z "${BRAVE_REWARDS_TOKEN}" ]; then
    error "BRAVE_REWARDS_TOKEN is not set. Not creating dist/.well-known/brave-rewards-verification.txt file."
else
    info "Creating dist/.well-known/brave-rewards-verification.txt file..."
    mkdir dist/.well-known
    cd ${HERE}/../packages/ui/dist/.well-known
    echo "This is a Brave Rewards publisher verification file.\n" >brave-rewards-verification.txt
    echo "Domain: vrooli.com" >>brave-rewards-verification.txt
    echo "Token: ${BRAVE_REWARDS_TOKEN}" >>brave-rewards-verification.txt
    cd ../..
fi

# Create assetlinks.json file for Google Play Store
if [ -z "${GOOGLE_PLAY_FINGERPRINT}" ]; then
    error "GOOGLE_PLAY_FINGERPRINT is not set. Not creating dist/.well-known/assetlinks.json file for Google Play Trusted Web Activity (TWA)."
else
    info "Creating dist/.well-known/assetlinks.json file for Google Play Trusted Web Activity (TWA)..."
    mkdir dist/.well-known
    cd ${HERE}/../packages/ui/dist/.well-known
    echo "[{\"relation\": [\"delegate_permission/common.handle_all_urls\"],\"target\": {\"namespace\": \"android_app\",\"package_name\": \"com.vrooli.twa\",\"sha256_cert_fingerprints\": [\"${GOOGLE_PLAY_FINGERPRINT}\"]}}]" >assetlinks.json
    cd ../..
fi

# Create ads.txt file for Google AdSense
if [ -z "${GOOGLE_ADSENSE_PUBLISHER_ID}" ]; then
    error "GOOGLE_ADSENSE_PUBLISHER_ID is not set. Not creating dist/ads.txt file for Google AdSense."
else
    info "Creating dist/ads.txt file for Google AdSense..."
    cd ${HERE}/../packages/ui/dist
    echo "google.com, ${GOOGLE_ADSENSE_PUBLISHER_ID}, DIRECT, f08c47fec0942fa0" >ads.txt
    cd ../..
fi

# Remove sitemap data for user-generated content,
# since this is generated dynamically by the production server.
info "Removing sitemap information from dist folder..."
cd ${HERE}/../packages/ui/dist
rm -f sitemap.xml sitemaps/*.xml.gz
rmdir sitemaps
cd ../..

# Compress build
info "Compressing build..."
tar -czf ${HERE}/../build.tar.gz -C ${HERE}/../packages/ui/dist .
trap "rm build.tar.gz" EXIT
if [ $? -ne 0 ]; then
    error "Failed to compress build"
    exit 1
fi

# Build Docker images
cd ${HERE}/..
info "Building (and Pulling) Docker images..."
docker-compose --env-file ${ENV_FILE} -f docker-compose-prod.yml build
docker pull ankane/pgvector:v0.4.1
docker pull redis:7-alpine

# Save and compress Docker images
info "Saving Docker images..."
docker save -o production-docker-images.tar ui:prod server:prod ankane/pgvector:v0.4.1 redis:7-alpine
if [ $? -ne 0 ]; then
    error "Failed to save Docker images"
    exit 1
fi
trap "rm production-docker-images.tar*" EXIT
info "Compressing Docker images..."
gzip -f production-docker-images.tar
if [ $? -ne 0 ]; then
    error "Failed to compress Docker images"
    exit 1
fi

# Send docker images to Docker Hub
prompt "Would you like to send the Docker images to Docker Hub? (y/N)"
read -n1 -r SEND_TO_DOCKER_HUB
echo
if [ "${SEND_TO_DOCKER_HUB}" = "y" ] || [ "${SEND_TO_DOCKER_HUB}" = "Y" ]; then
    "${HERE}/dockerToRegistry.sh -b n -v ${VERSION}"
    if [ $? -ne 0 ]; then
        error "Failed to send Docker images to Docker Hub"
        exit 1
    fi
fi

# Copy build to VPS
if [ -z "$DEPLOY" ]; then
    prompt "Build successful! Would you like to send the build to the production server? (y/N)"
    read -n1 -r DEPLOY
    echo
fi

if [ "${DEPLOY}" = "y" ] || [ "${DEPLOY}" = "Y" ] || [ "${DEPLOY}" = "yes" ] || [ "${DEPLOY}" = "Yes" ]; then
    "${HERE}/keylessSsh.sh" -e ${ENV_FILE}
    BUILD_DIR="${SITE_IP}:/var/tmp/${VERSION}/"
    prompt "Going to copy build and .env-prod to ${BUILD_DIR}. Press any key to continue..."
    read -n1 -r -s
    rsync -ri --info=progress2 -e "ssh -i ~/.ssh/id_rsa_${SITE_IP}" build.tar.gz production-docker-images.tar.gz ${ENV_FILE} root@${BUILD_DIR}
    if [ $? -ne 0 ]; then
        error "Failed to copy files to ${BUILD_DIR}"
        exit 1
    fi
    success "Files copied to ${BUILD_DIR}! To finish deployment, run deploy.sh on the VPS."
else
    BUILD_DIR="/var/tmp/${VERSION}"
    info "Copying build locally to ${BUILD_DIR}."
    # Make sure to create missing directories
    mkdir -p "${BUILD_DIR}"
    cp -p build.tar.gz production-docker-images.tar.gz ${BUILD_DIR}
    if [ $? -ne 0 ]; then
        error "Failed to copy build.tar.gz and production-docker-images.tar.gz to ${BUILD_DIR}"
        exit 1
    fi
    # If building locally, use .env and rename it to .env-prod
    cp -p ${ENV_FILE} ${BUILD_DIR}/.env-prod
    if [ $? -ne 0 ]; then
        error "Failed to copy ${ENV_FILE} to ${BUILD_DIR}/.env-prod"
        exit 1
    fi
fi

success "Build process completed successfully! Now run deploy.sh on the VPS to finish deployment, or locally to test deployment."
