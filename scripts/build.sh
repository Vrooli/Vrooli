#!/bin/bash
# NOTE: Run outside of Docker container
# Prepares project for deployment via Docker Compose or Kubernetes
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"

# Read arguments
ENV_FILE="${HERE}/../.env-prod"
TEST="y"
while getopts "v:d:u:ha:e:t:" opt; do
    case $opt in
    h)
        echo "Usage: $0 [-v VERSION] [-d DEPLOY_VPS_VPS] [-u USE_KUBERNETES] [-h] [-a API_GENERATE] [-e ENV_FILE]"
        echo "  -v --version: Version number to use (e.g. \"1.0.0\")"
        echo "  -d --deploy-vps: Deploy to VPS? (y/N)"
        echo "  -u --use-kubernetes: Deploy to Kubernetes? This overrides '--deploy-vps'. (y/N)"
        echo "  -h --help: Show this help message"
        echo "  -a --api-generate: Generate computed API information (GraphQL query/mutation selectors and OpenAPI schema)"
        echo "  -e --env-file: .env file location (e.g. \"/root/my-folder/.env\")"
        echo "  -t --test: Runs all tests to ensure code is working before building. Defaults to true. (y/N)"
        exit 0
        ;;
    a)
        API_GENERATE=$OPTARG
        ;;
    d)
        DEPLOY_VPS=$OPTARG
        ;;
    e)
        ENV_FILE=$OPTARG
        ;;
    t)
        TEST=$OPTARG
        ;;
    u)
        USE_KUBERNETES=$OPTARG
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
check_var PORT_SERVER
check_var SERVER_URL
check_var SITE_IP
check_var VAPID_PUBLIC_KEY
check_var GOOGLE_TRACKING_ID

# Extract the current version number from the package.json file
CURRENT_VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
# If version was entered (and it's different from the current version), set version
if [ ! -z "$VERSION" ] && [ "$VERSION" != "$CURRENT_VERSION" ]; then
    # Update package.json files for every package
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
else
    info "No version supplied, or version supplied is the current version. Sticking with version ${CURRENT_VERSION}."
    VERSION=$CURRENT_VERSION
fi

# Navigate to shared directory
cd ${HERE}/../packages/shared

# Run tests
if [[ "$TEST" =~ ^[Yy]([Ee][Ss])?$ ]]; then
    header "Running unit tests for shared..."
    yarn test
    header "Type checking shared..."
    yarn type-check
    if [ $? -ne 0 ]; then
        error "Failed to run unit tests for shared"
        exit 1
    fi
else
    warning "Skipping unit tests for shared..."
    warning "Skipping type checking for shared..."
fi

# Navigate to server directory
cd ${HERE}/../packages/server

# Run tests
if [[ "$TEST" =~ ^[Yy]([Ee][Ss])?$ ]]; then
    header "Running unit tests for server..."
    yarn test
    header "Type checking server..."
    yarn type-check
    if [ $? -ne 0 ]; then
        error "Failed to run unit tests for server"
        exit 1
    fi
else
    warning "Skipping unit tests for server..."
    warning "Skipping type checking for server..."
fi

# Build shared
info "Building shared..."
"${HERE}/shared.sh"
if [ $? -ne 0 ]; then
    error "Failed to build shared"
    exit 1
fi

# Build server
info "Building server..."
yarn build && yarn post-build
if [ $? -ne 0 ]; then
    error "Failed to build server"
    exit 1
fi

# Navigate to UI directory
cd ${HERE}/../packages/ui

# Run tests
if [[ "$TEST" =~ ^[Yy]([Ee][Ss])?$ ]]; then
    header "Running unit tests for UI..."
    yarn test
    header "Type checking UI..."
    yarn type-check
    if [ $? -ne 0 ]; then
        error "Failed to run unit tests for UI"
        exit 1
    fi
else
    warning "Skipping unit tests for UI..."
    warning "Skipping type checking for UI..."
fi

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
if [[ "$API_GENERATE" =~ ^[Yy]([Ee][Ss])?$ ]]; then
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

# Compress build TODO should probably compress builds for server and shared too, as well as all node_modules folders
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
docker pull ankane/pgvector:v0.4.4
docker pull redis:7-alpine

# Save and compress Docker images
info "Saving Docker images..."
docker save -o production-docker-images.tar ui:prod server:prod docs:prod ankane/pgvector:v0.4.4 redis:7-alpine
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
if [ -z "$USE_KUBERNETES" ]; then
    prompt "Would you like to use Kubernetes? (y/N)"
    read -n1 -r USE_KUBERNETES
    echo
fi

if [ "${USE_KUBERNETES}" = "n" ] || [ "${USE_KUBERNETES}" = "N" ]; then
    if [ -z "$DEPLOY_VPS" ]; then
        prompt "Would you like to send the build to the production server? (y/N)"
        read -n1 -r DEPLOY_VPS
        echo
    fi
fi

if [[ "$USE_KUBERNETES" =~ ^[Yy]([Ee][Ss])?$ ]]; then
    # Update build version in Kubernetes config
    if [ "${SHOULD_UPDATE_VERSION}" = true ]; then
        K8S_CONFIG_FILE="${HERE}/../k8s-docker-compose-prod-env.yml"
        if [ -f "${K8S_CONFIG_FILE}" ]; then
            info "Updating Kubernetes configuration file with version ${VERSION}"
            sed -i "s/v[0-9]*\.[0-9]*\.[0-9]*/v${VERSION}/g" "${K8S_CONFIG_FILE}"
            if [ $? -ne 0 ]; then
                error "Failed to update Kubernetes configuration file"
                # Consider whether to exit or not in case of failure
                # exit 1
            else
                success "Kubernetes configuration file updated successfully."
            fi
        else
            warning "Kubernetes configuration file not found. Continuing without updating."
        fi
    fi
    # Upload images to Docker Hub
    "${HERE}/dockerToRegistry.sh" -b n -v ${VERSION}
    if [ $? -ne 0 ]; then
        error "Failed to send Docker images to Docker Hub"
        exit 1
    fi
    # Store secrets used by Kubernetes
    "${HERE}/setKubernetesSecrets.sh" -e "production" "${secrets[@]}"
    if [ $? -ne 0 ]; then
        error "Failed to set Kubernetes secrets"
        exit 1
    fi
    # Add ui build to S3
    S3_BUCKET="vrooli-bucket"
    S3_PATH="builds/v${VERSION}/"
    prompt "Going to upload build.tar.gz to s3://${S3_BUCKET}/${S3_PATH}. Press any key to continue..."
    read -n1 -r -s
    AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} aws s3 cp build.tar.gz s3://${S3_BUCKET}/${S3_PATH}build.tar.gz
    if [ $? -ne 0 ]; then
        error "Failed to upload build.tar.gz to s3://${S3_BUCKET}/${S3_PATH}"
        exit 1
    fi
    success "build.tar.gz uploaded to s3://${S3_BUCKET}/${S3_PATH}!"
elif [[ "$DEPLOY_VPS" =~ ^[Yy]([Ee][Ss])?$ ]]; then
    # Copy build to VPS
    "${HERE}/keylessSsh.sh" -e ${ENV_FILE}
    BUILD_DIR="${SITE_IP}:/var/tmp/${VERSION}/"
    prompt "Going to copy build and .env-prod to ${BUILD_DIR}. Press any key to continue..."
    read -n1 -r -s
    rsync -ri --info=progress2 -e "ssh -i ~/.ssh/id_rsa_${SITE_IP}" build.tar.gz production-docker-images.tar.gz root@${BUILD_DIR}
    # ENV_FILE must be copied as .env-prod since that's what deploy.sh expects
    rsync -ri --info=progress2 -e "ssh -i ~/.ssh/id_rsa_${SITE_IP}" ${ENV_FILE} root@${BUILD_DIR}/.env-prod
    if [ $? -ne 0 ]; then
        error "Failed to copy files to ${BUILD_DIR}"
        exit 1
    fi
    success "Files copied to ${BUILD_DIR}! To finish deployment, run deploy.sh on the VPS."
else
    # Copy build locally
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
