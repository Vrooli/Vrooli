#!/bin/bash
# NOTE 1: Run outside of Docker container, on production server
# NOTE 2: First run build.sh on development server
# NOTE 3: If docker-compose file was changed since the last build, you should prune the containers and images before running this script.
# NOTE 4: If environment variables were added since the last build, you should make sure that the vault has been updated with the new secrets.
# Finishes up the deployment process, which was started by build.sh:
# 1. Checks if Nginx containers are running
# 2. Copies current database and build to a safe location, under a temporary directory.
# 3. Runs git fetch and git pull to get the latest changes.
# 4. Runs setup.sh
# 5. Moves build created by build.sh to the correct location.
# 6. Restarts docker containers
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Read arguments
SETUP_ARGS=()
# Clean up old builds
CLEAN_BUILDS=false
# Number of recent builds to keep
KEEP_BUILDS=3
# Skip confirmation when cleaning old builds
SKIP_CLEAN_CONFIRM=false

# Read arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
    -v | --version)
        if [ -z "$2" ] || [[ "$2" == -* ]]; then
            echo "Error: Option $key requires an argument."
            exit 1
        fi
        VERSION="${2}"
        shift # past argument
        shift # past value
        ;;
    -c | --clean)
        CLEAN_BUILDS=true
        shift # past argument
        ;;
    -k | --keep)
        if [ -z "$2" ] || [[ "$2" == -* ]]; then
            echo "Error: Option $key requires an argument."
            exit 1
        fi
        KEEP_BUILDS="${2}"
        shift # past argument
        shift # past value
        ;;
    -y | --yes)
        SKIP_CLEAN_CONFIRM=true
        shift # past argument
        ;;
    -h | --help)
        echo "Usage: $0 [-v VERSION] [-c] [-k KEEP_BUILDS] [-y] [-h]"
        echo "  -v --version: Version number to use (e.g. \"1.0.0\")"
        echo "  -c --clean: Clean up old build directories"
        echo "  -k --keep: Number of recent builds to keep when cleaning (default: 3)"
        echo "  -y --yes: Skip confirmation when cleaning build directories"
        echo "  -h --help: Show this help message"
        exit 0
        ;;
    *)
        SETUP_ARGS+=("$key")
        shift # past argument
        ;;
    esac
done

# Extract the current version number from the package.json file
CURRENT_VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
# Ask for version number, if not supplied in arguments
if [ -z "$VERSION" ]; then
    prompt "What version number do you want to deploy? (current is ${CURRENT_VERSION}). Leave blank if keeping the same version number."
    warning "WARNING: Keeping the same version number will overwrite the previous build AND database backup."
    read -r ENTERED_VERSION
    # If version entered, set version
    if [ ! -z "$ENTERED_VERSION" ]; then
        VERSION=$ENTERED_VERSION
    else
        VERSION=$CURRENT_VERSION
    fi
fi

# Copy current database and build to a safe location, under a temporary directory.
cd ${HERE}/..
DB_TMP="/var/tmp/${VERSION}/postgres"
DB_CURR="${HERE}/../data/postgres-prod"
BUILD_TMP="/var/tmp/${VERSION}/old-build"
BUILD_CURR="${HERE}/../packages/ui/dist"
# Don't copy database if it already exists in /var/tmp, or it doesn't exist in DB_CURR
if [ -d "${DB_TMP}" ]; then
    info "Old database already exists at ${DB_TMP}, so not copying it"
elif [ ! -d "${DB_CURR}" ]; then
    warning "Current database does not exist at ${DB_CURR}, so not copying it"
else
    info "Copying old database to ${DB_TMP}"
    cp -rp "${DB_CURR}" "${DB_TMP}"
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

# Pull the latest changes from the repository.
info "Pulling latest changes from repository..."
git fetch
git pull
if [ $? -ne 0 ]; then
    warning "Could not pull latest changes from repository. You likely have uncommitted changes. This may cause issues."
fi

# Running setup.sh
info "Running setup.sh..."
"${HERE}/setup.sh" "${SETUP_ARGS[@]}" -p
if [ $? -ne 0 ]; then
    error "setup.sh failed"
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

# Copy the .env-prod file to the correct location
if [ -f "${BUILD_ZIP}/.env-prod" ]; then
    info "Moving .env-prod file to ${HERE}/../.env-prod"
    cp "${BUILD_ZIP}/.env-prod" "${HERE}/../.env-prod"
    if [ $? -ne 0 ]; then
        error "Failed to move .env-prod file to ${HERE}/../.env-prod"
        exit 1
    fi
else
    error "Could not find .env-prod file at ${BUILD_ZIP}/.env-prod"
    exit 1
fi

# Stop docker containers
info "Stopping docker containers..."
docker-compose --env-file .env-prod down

info "Getting production secrets..."
readarray -t secrets <"${HERE}/secrets_list.txt"
TMP_FILE=$(mktemp) && { "${HERE}/getSecrets.sh" production ${TMP_FILE} "${secrets[@]}" 2>/dev/null && . "$TMP_FILE"; } || echo "Failed to get secrets."
rm "$TMP_FILE"
export DB_URL="postgresql://${DB_USER}:${DB_PASSWORD}@db:${PORT_DB:-5432}"
export REDIS_URL="redis://:${REDIS_PASSWORD}@redis:${PORT_REDIS:-6379}"
# Not sure why, but these need to be exported for the server to read them.
# This is not the case for the other secrets.
export JWT_PRIV
export JWT_PUB
# Determine where this script is running (local or remote)
export SERVER_LOCATION=$("${HERE}/domainCheck.sh" $SITE_IP $API_URL | tail -n 1)
if [ $? -ne 0 ]; then
    echo $SERVER_LOCATION
    error "Failed to determine server location"
    exit 1
fi

# Set up reverse proxy
"${HERE}/proxySetup.sh"
if [ $? -ne 0 ]; then
    error "Failed to set up proxy"
    exit 1
fi

# Function to clean up old build directories
cleanup_old_builds() {
    # Create a list of all build directories sorted by modification time, newest first
    local ALL_BUILDS=($(ls -td /var/tmp/[0-9]* 2>/dev/null))
    
    # If there are fewer builds than we want to keep, do nothing
    local TOTAL_BUILDS=${#ALL_BUILDS[@]}
    if [ $TOTAL_BUILDS -le $KEEP_BUILDS ]; then
        info "Found $TOTAL_BUILDS build directories, which is fewer than or equal to the limit ($KEEP_BUILDS). Nothing to clean up."
        return 0
    fi
    
    # Calculate how many builds to remove
    local REMOVE_COUNT=$((TOTAL_BUILDS - KEEP_BUILDS))
    info "Found $TOTAL_BUILDS build directories. Will keep $KEEP_BUILDS recent ones and remove $REMOVE_COUNT old ones."
    
    # Get the builds to remove (skip the first $KEEP_BUILDS)
    local BUILDS_TO_REMOVE=("${ALL_BUILDS[@]:$KEEP_BUILDS}")
    
    # Display builds that will be removed
    echo "The following build directories will be removed:"
    for dir in "${BUILDS_TO_REMOVE[@]}"; do
        echo "  - $dir ($(du -sh "$dir" 2>/dev/null | cut -f1) - last modified $(date -r "$dir" '+%Y-%m-%d %H:%M:%S'))"
    done
    
    # Ask for confirmation unless skipped
    if [ "$SKIP_CLEAN_CONFIRM" != true ]; then
        prompt "Do you want to proceed with removing these directories? (y/N)"
        read -n 1 -r REPLY
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            info "Cleanup aborted."
            return 0
        fi
    fi
    
    # Remove the old build directories
    local SPACE_BEFORE=$(df -h /var/tmp | awk 'NR==2 {print $4}')
    for dir in "${BUILDS_TO_REMOVE[@]}"; do
        info "Removing $dir..."
        rm -rf "$dir"
    done
    
    local SPACE_AFTER=$(df -h /var/tmp | awk 'NR==2 {print $4}')
    success "Cleanup complete. Free space in /var/tmp changed from $SPACE_BEFORE to $SPACE_AFTER."
}

# Clean up old builds if requested
if [ "$CLEAN_BUILDS" = true ]; then
    header "Cleaning up old build directories"
    cleanup_old_builds
fi

# Restart docker containers.
info "Restarting docker containers..."
docker-compose --env-file .env-prod -f docker-compose-prod.yml up -d

success "Done! You may need to wait a few minutes for the Docker containers to finish starting up."
info "Now that you've deployed, here are some next steps:"
info "- Manually check that the site is working correctly"
info "- Make sure that all environment variables are correct (e.g. Stripe keys), if you haven't already"
info "- Upload the sitemap index file from packages/ui/public/sitemap.xml to Google Search Console, Bing Webmaster Tools, and Yandex Webmaster Tools"
info "- Let everyone on social media know that you've deployed a new version of Vrooli!"
info "- Consider cleaning up old build directories with: $0 -c -k 3"
