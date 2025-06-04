
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
# Set trap to remove ${TMP_ENV_FILE} file on exit (ignore missing)
trap "rm -f ${HERE}/../packages/ui/${TMP_ENV_FILE}" EXIT

# Generate API information
if is_yes "$API_GENERATE"; then
    script_location="${HERE}/../packages/server/src/__tools/api/buildAPIPrismaSelects.ts"
    run_step "Generating Prisma 'select' objects for each endpoint..." "NODE_OPTIONS=\"--max-old-space-size=4096\" npx vite-node ${script_location}"

    script_location="${HERE}/../packages/shared/src/__tools/api/buildOpenAPISchema.ts"
    run_step "Generating OpenAPI schema" "NODE_OPTIONS=\"--max-old-space-size=4096\" npx vite-node ${script_location}"

    script_location="${HERE}/../packages/server/src/__tools/ai/generateToolSchemas.ts"
    run_step "Generating MCP schemas" "NODE_OPTIONS=\"--max-old-space-size=4096\" npx vite-node ${script_location}"
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

# Used to be brave rewards, ..., google play signing

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
# Remove sitemaps dir if it exists, but don't fail if it's missing
rmdir sitemaps || true
cd ../..

# Compress build TODO should probably compress builds for server and shared too, as well as all node_modules folders
info "Compressing build..."
tar -czf ${HERE}/../build.tar.gz -C ${HERE}/../packages/ui/dist .
trap "rm build.tar.gz" EXIT
if [ $? -ne 0 ]; then
    error "Failed to compress build"
    exit 1
fi

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
   # Already replaced in updated script
fi