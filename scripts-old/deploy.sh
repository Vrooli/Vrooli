# Read arguments
SETUP_ARGS=()
ENV_FILE_PROD=".env-prod"
ENV_FILE_DEV=".env-dev"
ENV_FILE=${ENV_FILE_PROD}
# Clean up old builds
CLEAN_BUILDS=false
# Number of recent builds to keep
KEEP_BUILDS=3
# Skip confirmation prompts
SKIP_CONFIRMATIONS=false


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
    -p | --prod)
        if is_yes "$2"; then
            ENV_FILE=${ENV_FILE_PROD}
        else
            ENV_FILE=${ENV_FILE_DEV}
        fi
        SETUP_ARGS+=("$key" "$2") # Also pass this to setup.sh
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
        SKIP_CONFIRMATIONS=true
        SETUP_ARGS+=("$key") # Also pass this to setup.sh
        shift # past argument
        ;;
    -h | --help)
        echo "Usage: $0 [-v VERSION] [-c] [-k KEEP_BUILDS] [-y] [-h]"
        echo "  -v --version:   Version number to use (e.g. \"1.0.0\")"
        echo "  -p --prod:      (y/N) If true, will use production environment variables and docker-compose-prod.yml file"
        echo "  -c --clean:     Clean up old build directories"
        echo "  -k --keep:      Number of recent builds to keep when cleaning (default: 3)"
        echo "  -y --yes:       Automatically answer yes to all confirmation prompts"
        echo "  -h --help:      Show this help message"
        exit 0
        ;;
    *)
        SETUP_ARGS+=("$key")
        shift # past argument
        ;;
    esac
done

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

# Pull the latest changes from the repository.
info "Pulling latest changes from repository..."
git fetch
git pull
if [ $? -ne 0 ]; then
    warning "Could not pull latest changes from repository. You likely have uncommitted changes. This may cause issues."
fi

# Copy the environment file to the correct location
ENV_FILE_DEST="${HERE}/../${ENV_FILE}"
if [ -f "${BUILD_ZIP}/${ENV_FILE}" ]; then
    info "Copying ${ENV_FILE} file from build to ${ENV_FILE_DEST}"
    cp "${BUILD_ZIP}/${ENV_FILE}" "${ENV_FILE_DEST}"
    if [ $? -ne 0 ]; then
        error "Failed to copy ${ENV_FILE} file to ${ENV_FILE_DEST}"
        exit 1
    fi
else
    if [ -f "${ENV_FILE_DEST}" ]; then
        warning "Could not find ${ENV_FILE} file at ${BUILD_ZIP}/${ENV_FILE}, but found existing file at ${ENV_FILE_DEST}. Using existing file."
    else
        error "Could not find ${ENV_FILE} file at ${BUILD_ZIP}/${ENV_FILE} and no existing file at ${ENV_FILE_DEST}. Deployment cannot continue without environment configuration."
        exit 1
    fi
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
    if [ "$SKIP_CONFIRMATIONS" != true ]; then
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
