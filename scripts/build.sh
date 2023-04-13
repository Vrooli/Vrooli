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
# -g: Generate GraphQL tags for queries/mutations
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# Read arguments
while getopts "v:d:hg:" opt; do
    case $opt in
    v)
        VERSION=$OPTARG
        ;;
    d)
        DEPLOY=$OPTARG
        ;;
    g)
        GRAPHQL_GENERATE=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-v VERSION] [-d DEPLOY] [-h]"
        echo "  -v --version: Version number to use (e.g. \"1.0.0\")"
        echo "  -d --deploy: Deploy to VPS (y/N)"
        echo "  -h --help: Show this help message"
        echo "  -g --graphql-generate: Generate GraphQL tags for queries/mutations"
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
if [ -f "${HERE}/../.env" ]; then
    source "${HERE}/../.env"
else
    error "Could not find .env file. Exiting..."
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

# Ask for version number, if not supplied in arguments
AUTO_DETECT_VERSION=false
if [ -z "$VERSION" ]; then
    prompt "What version number do you want to deploy? (e.g. 1.0.0). Leave blank if keeping the same version number."
    warning "WARNING: Keeping the same version number will overwrite the previous build."
    read -r VERSION
    # If no version number was entered, use the version number found in the package.json files
    if [ -z "$VERSION" ]; then
        info "No version number entered. Using version number found in package.json files."
        VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
        info "Version number found in package.json files: ${VERSION}"
        AUTO_DETECT_VERSION=true
    fi
fi

# Update package.json files for every package, if version number was not auto-detected
if [ "${AUTO_DETECT_VERSION}" = false ]; then
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

# Build server
info "Building server..."
yarn build

# Navigate to UI directory
cd ${HERE}/../packages/ui

# Create local .env file
touch .env
# Set environment variables
echo "VITE_SERVER_LOCATION=${SERVER_LOCATION}" >>.env
echo "VITE_PORT_SERVER=${PORT_SERVER}" >>.env
echo "VITE_SERVER_URL=${SERVER_URL}" >>.env
echo "VITE_SITE_IP=${SITE_IP}" >>.env
# Set trap to remove .env file on exit
trap "rm .env" EXIT

# Generate query/mutation selectors
if [ -z "$GRAPHQL_GENERATE" ]; then
    prompt "Do you want to generate GraphQL query/mutation selectors? (y/N)"
    read -r GRAPHQL_GENERATE
fi
if [ "${GRAPHQL_GENERATE}" = "y" ] || [ "${GRAPHQL_GENERATE}" = "Y" ] || [ "${GRAPHQL_GENERATE}" = "yes" ] || [ "${GRAPHQL_GENERATE}" = "Yes" ]; then
    info "Generating GraphQL query/mutation selectors... (this may take a minute)"
    ts-node --esm --experimental-specifier-resolution node ./src/tools/api/gqlSelects.ts
    if [ $? -ne 0 ]; then
        error "Failed to generate query/mutation selectors"
        echo "${HERE}/../packages/ui/src/tools/api/gqlSelects.ts"
        # This IS a critical error, so we'll exit
        exit 1
    fi
fi

# Build React app
info "Building React app..."
yarn build
if [ $? -ne 0 ]; then
    error "Failed to build React app"
    exit 1
fi

# Use production icons
mv dist/prod/* dist/
if [ $? -ne 0 ]; then
    error "Failed to move production icons"
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

# Compress build
info "Compressing build..."
tar -czf ${HERE}/../build.tar.gz dist
trap "rm build.tar.gz" EXIT
if [ $? -ne 0 ]; then
    error "Failed to compress build"
    exit 1
fi

# Build Docker images
cd ${HERE}/..
info "Building (and Pulling) Docker images..."
docker-compose --env-file .env -f docker-compose-prod.yml build
docker pull postgres:13-alpine
docker pull redis:7-alpine

# Save and compress Docker images
info "Saving Docker images..."
docker save -o production-docker-images.tar ui:prod server:prod postgres:13-alpine redis:7-alpine
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

# Copy build to VPS
if [ -z "$DEPLOY" ]; then
    prompt "Build successful! Would you like to send the build to the production server? (y/N)"
    read -r DEPLOY
fi

if [ "${DEPLOY}" = "y" ] || [ "${DEPLOY}" = "Y" ] || [ "${DEPLOY}" = "yes" ] || [ "${DEPLOY}" = "Yes" ]; then
    source "${HERE}/keylessSsh.sh"
    BUILD_DIR="${SITE_IP}:/var/tmp/${VERSION}/"
    prompt "Going to copy build and .env-prod to ${BUILD_DIR}. Press any key to continue..."
    read -r
    rsync -ri build.tar.gz production-docker-images.tar.gz .env-prod root@${BUILD_DIR}
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
    cp -p .env ${BUILD_DIR}/.env-prod
    if [ $? -ne 0 ]; then
        error "Failed to copy .env to ${BUILD_DIR}/.env-prod"
        exit 1
    fi
fi

success "Build process completed successfully! Now run deploy.sh on the VPS to finish deployment, or locally to test deployment."
