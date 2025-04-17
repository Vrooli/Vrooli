#!/bin/bash
# Add debug flags and print environment and docker tool versions
set -o errexit
set -o nounset
set -o pipefail
set -x

# Debug: print PATH and tool locations
echo "--- Debug: Environment PATH ---"
echo "$PATH"
echo "--- Debug: Docker version ---"
docker --version || true
echo "--- Debug: Which docker-compose ---"
which docker-compose || true
echo "--- Debug: docker-compose version ---"
docker-compose --version 2>&1 || true
echo "--- Debug: docker compose plugin version ---"
docker compose version 2>&1 || true

# NOTE: Run outside of Docker container
# Prepares project for deployment via Docker Compose or Kubernetes
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Read arguments
ENV_FILE_PROD=".env-prod"
ENV_FILE_DEV=".env-dev"
ENV_FILE=${ENV_FILE_PROD}
DOCKER_COMPOSE_FILE_PROD="docker-compose-prod.yml"
DOCKER_COMPOSE_FILE_DEV="docker-compose.yml"
DOCKER_COMPOSE_FILE=${DOCKER_COMPOSE_FILE_PROD}
KUBERNETES_FILE_PROD="k8s-docker-compose-prod-env.yml"
KUBERNETES_FILE_DEV="k8s-docker-compose-dev-env.yml"
KUBERNETES_FILE=${KUBERNETES_FILE_PROD}
ENVIRONMENT=${NODE_ENV:-development}
TEST="y"
API_GENERATE="n"
GOOGLE_PLAY_KEYS_GENERATE="y"
SKIP_CONFIRMATIONS="n"

## Parse both short and long options
PARSED_OPTS=$(getopt -o v:d:u:ha:t:p:g:y -l version:,deploy-vps:,use-kubernetes:,help,api-generate:,test:,prod:,yes,google-play-keys-generate -- "$@")
if [ $? -ne 0 ]; then
    echo "Error: Failed to parse options" >&2
    exit 1
fi
eval set -- "$PARSED_OPTS"
while true; do
    case "$1" in
        -v|--version)
            VERSION="$2"; shift 2;;
        -d|--deploy-vps)
            DEPLOY_VPS="$2"; shift 2;;
        -u|--use-kubernetes)
            USE_KUBERNETES="$2"; shift 2;;
        -a|--api-generate)
            API_GENERATE="$2"; shift 2;;
        -g|--google-play-keys-generate)
            GOOGLE_PLAY_KEYS_GENERATE="$2"; shift 2;;
        -t|--test)
            TEST="$2"; shift 2;;
        -p|--prod)
            if is_yes "$2"; then
                ENV_FILE=${ENV_FILE_PROD}; DOCKER_COMPOSE_FILE=${DOCKER_COMPOSE_FILE_PROD}; KUBERNETES_FILE=${KUBERNETES_FILE_PROD}; ENVIRONMENT="production"
            else
                ENV_FILE=${ENV_FILE_DEV}; DOCKER_COMPOSE_FILE=${DOCKER_COMPOSE_FILE_DEV}; KUBERNETES_FILE=${KUBERNETES_FILE_DEV}; ENVIRONMENT="development"
            fi
            shift 2;;
        -y|--yes)
            SKIP_CONFIRMATIONS="y"; shift;;
        -h|--help)
            # Display help
            echo "Usage: $0 [-v VERSION] [-d DEPLOY_VPS_VPS] [-u USE_KUBERNETES] [-h] [-a API_GENERATE] [-e ENV_FILE]"
            echo "  -v --version:        Version number to use (e.g. \"1.0.0\")"
            echo "  -d --deploy-vps:     (y/N) Deploy to VPS?"
            echo "  -u --use-kubernetes: (y/N) Deploy to Kubernetes? This overrides '--deploy-vps'"
            echo "  -h --help:           Show this help message"
            echo "  -a --api-generate:   Generate computed API information (GraphQL query/mutation selectors and OpenAPI schema)"
            echo "  -g --google-play-keys-generate: (y/N) Generate Google Play Store keystore and assetlinks. Defaults to true."
            echo "  -p --prod:           (y/N) If true, will use production environment variables and docker-compose-prod.yml file"
            echo "  -t --test:           (y/N) Runs all tests to ensure code is working before building. Defaults to true."
            echo "  -y --yes:            (y/N) Skip all confirmations. Defaults to false."
            exit 0
            ;;
        --)
            shift; break;;
        *)
            echo "Error: Unknown option $1" >&2; exit 1;;
    esac
done

# Load variables from ${ENV_FILE} file
if [ -f "${ENV_FILE}" ]; then
    info "Loading variables from ${HERE}/../${ENV_FILE}..."
    . "${HERE}/../${ENV_FILE}"
else
    error "Could not find ${ENV_FILE} file at ${HERE}/../${ENV_FILE}. Exiting..."
    exit 1
fi

# Check for required variables
check_var() {
    if [ -z "${!1}" ]; then
        error "Variable ${1} is not set. Exiting..."
        exit 1
    else
        info "Variable ${1} is set to ${!1:0:5}..."
    fi
}
check_var PORT_API
check_var API_URL
check_var SITE_IP
check_var VAPID_PUBLIC_KEY
check_var STRIPE_PUBLISHABLE_KEY
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
    # Update root package.json file too
    cd ${HERE}/..
    info "Updating root package.json"
    yarn version patch --new-version ${VERSION} --no-git-tag-version
else
    info "No version supplied, or version supplied is the current version. Sticking with version ${CURRENT_VERSION}."
    VERSION=$CURRENT_VERSION
fi

# Run bash script tests
if is_yes "$TEST"; then
    run_step "Running bash script tests (bats)" "${HERE}/tests/__runTests.sh"
else
    warning "Skipping bash script tests..."
fi

# Navigate to shared directory
cd ${HERE}/../packages/shared

# Run tests
if is_yes "$TEST"; then
    run_step "Running unit tests for shared..." "yarn test"
    run_step "Type checking shared..." "yarn type-check"
else
    warning "Skipping unit tests for shared..."
    warning "Skipping type checking for shared..."
fi

# Build shared
run_step "Building shared" "${HERE}/shared.sh"

# Navigate to server directory
cd ${HERE}/../packages/server

# Run tests
if is_yes "$TEST"; then
    run_step "Running unit tests for server..." "yarn test"
    run_step "Type checking server..." "yarn type-check"
else
    warning "Skipping unit tests for server..."
    warning "Skipping type checking for server..."
fi

# Build server
run_step "Building server" "yarn build && yarn post-build"

# Navigate to UI directory
cd ${HERE}/../packages/ui

# Run tests
if is_yes "$TEST"; then
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

# Create local environment file
TMP_ENV_FILE=".env-tmp"
touch ${TMP_ENV_FILE}
# Set environment variables
echo "VITE_SERVER_LOCATION=${SERVER_LOCATION:-remote}" >>${TMP_ENV_FILE}
echo "VITE_PORT_API=${PORT_API}" >>${TMP_ENV_FILE}
echo "VITE_API_URL=${API_URL}" >>${TMP_ENV_FILE}
echo "VITE_SITE_IP=${SITE_IP}" >>${TMP_ENV_FILE}
echo "VITE_VAPID_PUBLIC_KEY=${VAPID_PUBLIC_KEY}" >>${TMP_ENV_FILE}
echo "VITE_STRIPE_PUBLISHABLE_KEY=${STRIPE_PUBLISHABLE_KEY}" >>${TMP_ENV_FILE}
echo "VITE_GOOGLE_ADSENSE_PUBLISHER_ID=${GOOGLE_ADSENSE_PUBLISHER_ID}" >>${TMP_ENV_FILE}
echo "VITE_GOOGLE_TRACKING_ID=${GOOGLE_TRACKING_ID}" >>${TMP_ENV_FILE}
# Set trap to remove ${TMP_ENV_FILE} file on exit
trap "rm ${HERE}/../packages/ui/${TMP_ENV_FILE}" EXIT

# Generate API information
if is_yes "$API_GENERATE"; then
    script_location="${HERE}/../packages/server/src/__tools/api/buildAPIPrismaSelects.ts"
    run_step "Generating Prisma 'select' objects for each endpoint..." "NODE_OPTIONS=\"--max-old-space-size=4096\" npx vite-node ${script_location}"

    script_location="${HERE}/../packages/shared/src/__tools/api/buildOpenAPISchema.ts"
    run_step "Generating OpenAPI schema" "NODE_OPTIONS=\"--max-old-space-size=4096\" npx vite-node ${script_location}"
fi

# Build React app
run_step "Building React app" "yarn build"

# Normalize UI_URL to ensure it ends with exactly one "/"
export UI_URL="${UI_URL%/}/"
DOMAIN=$(echo "${UI_URL%/}" | sed -E 's|https?://([^/]+)|\1|')
echo "Got domain ${DOMAIN} from UI_URL ${UI_URL}"

# Generate sitemap.xml
script_location="${HERE}/../packages/ui/src/__tools/sitemap/sitemap.ts"
run_step "Generating sitemap.xml" "npx tsx ${script_location}"

# Replace placeholder url in public files
sed -i'' "s|<UI_URL>|${UI_URL}|g" "${HERE}/../packages/ui/dist/manifest.dark.manifest"
sed -i'' "s|\*.<DOMAIN>|*.${DOMAIN}|g" "${HERE}/../packages/ui/dist/manifest.dark.manifest"
sed -i'' "s|<UI_URL>|${UI_URL}|g" "${HERE}/../packages/ui/dist/manifest.light.manifest"
sed -i'' "s|\*.<DOMAIN>|*.${DOMAIN}|g" "${HERE}/../packages/ui/dist/manifest.light.manifest"
sed -i'' "s|<UI_URL>|${UI_URL}|g" "${HERE}/../packages/ui/dist/robots.txt"
sed -i'' "s|<UI_URL>|${UI_URL}|g" "${HERE}/../packages/ui/dist/search.xml"

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

# Create Twilio domain verification file
# WARNING: This probably won't work, as it's an html file, so our app will get confused and think we're
# trying to serve a page that doesn't exist. If this happens, you can add a new DNS record instead.
# Set the host to "_twilio", the type to "TXT", and the value to what you would have put in the file.
if [ -z "${TWILIO_DOMAIN_VERIFICATION_CODE}" ]; then
    error "TWILIO_DOMAIN_VERIFICATION_CODE is not set. Not creating Twilio domain verification file, which is needed to send SMS messages."
else
    # File name is the same as the value
    info "Creating dist/${TWILIO_DOMAIN_VERIFICATION_CODE}.html file..."
    cd ${HERE}/../packages/ui/dist
    echo "twilio-domain-verification=${TWILIO_DOMAIN_VERIFICATION_CODE}" >${TWILIO_DOMAIN_VERIFICATION_CODE}.html
    cd ../..
fi

# Setup keys for Google Play Store
KEYSTORE_PATH="${HERE}/../upload-keystore.jks"
KEYSTORE_ALIAS="upload"
KEYSTORE_PASSWORD="${GOOGLE_PLAY_KEYSTORE_PASSWORD}"

if is_yes "$GOOGLE_PLAY_KEYS_GENERATE"; then
    # Check if keystore file exists
    if [ ! -f "${KEYSTORE_PATH}" ]; then
        if is_yes "$SKIP_CONFIRMATIONS"; then
            info "Keystore file not found. This is needed to upload the app the Google Play store. Creating keystore file..."
            REPLY="y"
        else
            prompt "Keystore file not found. This is needed to upload the app the Google Play store. Would you like to create the file? (Y/n)"
            read -n1 -r REPLY
            echo
        fi
        if is_yes "$REPLY"; then
            # Generate the keystore file
            header "Generating keystore file..."
            info "Before we begin, you'll need to provide some information for the keystore certificate:"
            info "This information should be accurate, but it does not have to match exactly with the information you used to register your Google Play account."
            info "1. First and Last Name: Your full legal name. Example: John Doe"
            info "2. Organizational Unit: The department within your organization managing the key. Example: IT"
            info "3. Organization: The legal name of your company or organization. Example: Vrooli"
            info "4. City or Locality: The city where your organization is based. Example: New York City"
            info "5. State or Province: The state or province where your organization is located. Example: New York"
            info "6. Country Code: The two-letter ISO code for the country of your organization. Example: US"
            info "This information helps to identify the holder of the key."
            if is_yes "$SKIP_CONFIRMATIONS"; then
                info "Skipping confirmation..."
                REPLY="y"
            else
                prompt "Press any key to continue..."
                read -n1 -r -s
            fi

            run_step "Generating keystore file for Google Play Store" "keytool -genkey -v -keystore \"${KEYSTORE_PATH}\" -alias \"${KEYSTORE_ALIAS}\" -keyalg RSA -keysize 2048 -validity 10000 -storepass \"${KEYSTORE_PASSWORD}\""
        fi
    fi
fi

create_assetlinks_file() {
    # Create assetlinks.json file for Google Play Store
    if [ -f "${KEYSTORE_PATH}" ]; then
        header "Creating dist/.well-known/assetlinks.json file so Google can verify the app with the website..."
        # Export the PEM file from keystore
        PEM_PATH="${HERE}/../upload_certificate.pem"
        keytool -export -rfc -keystore "${KEYSTORE_PATH}" -alias "${KEYSTORE_ALIAS}" -file "${PEM_PATH}" -storepass "${KEYSTORE_PASSWORD}"
        if [ $? -ne 0 ]; then
            warning "PEM file could not be generated. The app cannot be uploaded to the Google Play store without it."
            return 1
        fi

        # Extract SHA-256 fingerprint
        GOOGLE_PLAY_FINGERPRINT=$(keytool -list -keystore "${KEYSTORE_PATH}" -alias "${KEYSTORE_ALIAS}" -storepass "${KEYSTORE_PASSWORD}" -v | grep "SHA256:" | awk '{ print $2 }')
        if [ $? -ne 0 ]; then
            warning "SHA-256 fingerprint could not be extracted. The app cannot be uploaded to the Google Play store without it."
            return 1
        else
            success "SHA-256 fingerprint extracted successfully: $GOOGLE_PLAY_FINGERPRINT"
        fi

        # Create assetlinks.json file for Google Play Store
        if [ -z "${GOOGLE_PLAY_FINGERPRINT}" ]; then
            warning "GOOGLE_PLAY_FINGERPRINT is not set. Not creating dist/.well-known/assetlinks.json file for Google Play Trusted Web Activity (TWA)."
        else
            info "Creating dist/.well-known/assetlinks.json file for Google Play Trusted Web Activity (TWA)..."
            mkdir -p ${HERE}/../packages/ui/dist/.well-known
            cd ${HERE}/../packages/ui/dist/.well-known
            echo "[{" >assetlinks.json
            echo "    \"relation\": [\"delegate_permission/common.handle_all_urls\"]," >>assetlinks.json
            echo "    \"target\": {" >>assetlinks.json
            echo "      \"namespace\": \"android_app\"," >>assetlinks.json
            echo "      \"package_name\": \"com.vrooli.twa\"," >>assetlinks.json
            # Check if the GOOGLE_PLAY_DOMAIN_FINGERPRINT variable is set and append it to the array
            if [ ! -z "${GOOGLE_PLAY_DOMAIN_FINGERPRINT}" ]; then
                echo "      \"sha256_cert_fingerprints\": [" >>assetlinks.json
                echo "          \"${GOOGLE_PLAY_FINGERPRINT}\"," >>assetlinks.json
                echo "          \"${GOOGLE_PLAY_DOMAIN_FINGERPRINT}\"" >>assetlinks.json
                echo "      ]" >>assetlinks.json
            else
                echo "      \"sha256_cert_fingerprints\": [\"${GOOGLE_PLAY_FINGERPRINT}\"]" >>assetlinks.json
            fi
            echo "    }" >>assetlinks.json
            echo "}]" >>assetlinks.json
            cd ${HERE}/..
        fi
    fi
}
if is_yes "$GOOGLE_PLAY_KEYS_GENERATE"; then
    create_assetlinks_file
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
rm -f sitemap.xml sitemaps/*.xml.gz sitemaps/*.xml
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

# Prefer Docker Compose plugin if available, else fallback to docker-compose
if command -v docker compose >/dev/null 2>&1; then
    echo "Using Docker Compose plugin to build images"
    docker compose --env-file "${HERE}/../${ENV_FILE}" -f "${DOCKER_COMPOSE_FILE}" build --no-cache --progress=plain
elif command -v docker-compose >/dev/null 2>&1; then
    echo "Using legacy docker-compose to build images"
    docker-compose --env-file "${HERE}/../${ENV_FILE}" -f "${DOCKER_COMPOSE_FILE}" build --no-cache
else
    error "Neither docker compose plugin nor docker-compose binary is available. Cannot build Docker images."
fi

# Pull necessary images regardless of docker-compose success
docker pull ankane/pgvector:v0.4.4
docker pull redis:7.4.0-alpine

# Save and compress Docker images
info "Saving Docker images..."

# Initialize an array to store available images
AVAILABLE_IMAGES=()

# Check each image individually and add to array if available
IMAGE_SUFFIX="prod"
if [ "${ENVIRONMENT}" = "development" ]; then
    IMAGE_SUFFIX="dev"
fi
for IMAGE in "ui:${IMAGE_SUFFIX}" "server:${IMAGE_SUFFIX}" "jobs:${IMAGE_SUFFIX}" "ankane/pgvector:v0.4.4" "redis:7.4.0-alpine"; do
    if docker image inspect "$IMAGE" >/dev/null 2>&1; then
        AVAILABLE_IMAGES+=("$IMAGE")
        info "Image $IMAGE found and will be saved"
    else
        warning "Image $IMAGE not found and will be skipped"
    fi
done

# If no images are available, exit with error
if [ ${#AVAILABLE_IMAGES[@]} -eq 0 ]; then
    error "No Docker images available to save"
    exit 1
fi

# Save available images
docker save -o production-docker-images.tar "${AVAILABLE_IMAGES[@]}"
if [ $? -ne 0 ]; then
    error "Failed to save Docker images"
    exit 1
fi

trap "rm production-docker-images.tar*" EXIT
run_step "Compressing Docker images..." "gzip -f production-docker-images.tar"

# Send docker images to Docker Hub
if [ -z "$USE_KUBERNETES" ]; then
    prompt "Would you like to use Kubernetes? (y/N)"
    read -n1 -r USE_KUBERNETES
    echo
fi

if ! is_yes "$USE_KUBERNETES"; then
    if [ -z "$DEPLOY_VPS" ]; then
        prompt "Would you like to send the build to the production server? (y/N)"
        read -n1 -r DEPLOY_VPS
        echo
    fi
fi

if is_yes "$USE_KUBERNETES"; then
    # Update build version in Kubernetes config
    if [ "${SHOULD_UPDATE_VERSION}" = true ]; then
        K8S_CONFIG_FILE="${HERE}/../${KUBERNETES_FILE}"
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
    if is_yes "$SKIP_CONFIRMATIONS"; then
        info "Skipping confirmation..."
        REPLY="y"
    else
        prompt "Going to upload build.tar.gz to s3://${S3_BUCKET}/${S3_PATH}. Press any key to continue..."
        read -n1 -r -s
    fi
    AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} aws s3 cp build.tar.gz s3://${S3_BUCKET}/${S3_PATH}build.tar.gz
    if [ $? -ne 0 ]; then
        error "Failed to upload build.tar.gz to s3://${S3_BUCKET}/${S3_PATH}"
        exit 1
    fi
    success "build.tar.gz uploaded to s3://${S3_BUCKET}/${S3_PATH}!"
elif is_yes "$DEPLOY_VPS"; then
    # Copy build to VPS
    "${HERE}/keylessSsh.sh" -e ${ENV_FILE}
    BUILD_DIR="/var/tmp/${VERSION}/"
    if is_yes "$SKIP_CONFIRMATIONS"; then
        info "Skipping confirmation..."
        REPLY="y"
    else
        prompt "Going to copy build to ${SITE_IP}:${BUILD_DIR}. Press any key to continue..."
        read -n1 -r -s
    fi
    # Ensure that target directory exists
    ssh -i ~/.ssh/id_rsa_${SITE_IP} root@${SITE_IP} "mkdir -p ${BUILD_DIR}"
    # Copy everything
    rsync -ri --info=progress2 -e "ssh -i ~/.ssh/id_rsa_${SITE_IP}" build.tar.gz production-docker-images.tar.gz ${ENV_FILE} root@${SITE_IP}:${BUILD_DIR}
    if [ $? -ne 0 ]; then
        error "Failed to copy files to ${SITE_IP}:${BUILD_DIR}"
        exit 1
    fi
    success "Files copied to ${SITE_IP}:${BUILD_DIR}! To finish deployment, run deploy.sh on the VPS."
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
fi

success "Build process completed successfully! Now run deploy.sh on the VPS to finish deployment, or locally to test deployment."
