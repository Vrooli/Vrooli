#!/bin/bash
# This script periodically backs up the database from a remote server
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/prettify.sh"
. "${HERE}/keylessSsh.sh"

# Read arguments
while getopts "c:hi:l:v:" opt; do
    case $opt in
    h)
        echo "Usage: $0 [-c COUNT] [-h HELP] [-i INTERVAL] [-l LOOP] [-v VERSION]"
        echo "  -c --count: The number of most recent backup files to keep"
        echo "  -h --help: Show this help message"
        echo "  -i --interval: The interval in seconds for fetching the logs, if running on a loop"
        echo "  -l --loop: Whether to run this script on a loop, or to exit after one run"
        echo "  -v --version: Version number of site on server (e.g. \"1.0.0\"). Used to locate the database directory"
        exit 0
        ;;
    c)
        BACKUP_COUNT=$OPTARG
        ;;
    i)
        INTERVAL=$OPTARG
        ;;
    l)
        WILL_LOOP=$OPTARG
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
if [ -f "${HERE}/../.env" ]; then
    . "${HERE}/../.env"
else
    error "Could not find .env file. Exiting..."
    exit 1
fi

# The number of most recent backup files to keep
if [ -z "$BACKUP_COUNT" ]; then
    BACKUP_COUNT=5
fi

if [ -z "$WILL_LOOP" ]; then
    WILL_LOOP=true
fi

# Set the remote server location, using SITE_IP from .env
remote_server="root@${SITE_IP}"
info "Remote server: ${remote_server}"

# If no version number is defined, use the version number found in the package.json files
if [ -z "$VERSION" ]; then
    info "No version number defined. Using version number found in package.json files."
    VERSION=$(cat ${HERE}/../packages/ui/package.json | grep version | head -1 | awk -F: '{ print $2 }' | sed 's/[",]//g' | tr -d '[[:space:]]')
    info "Version number found in package.json files: ${VERSION}"
fi

# Set the local directory to save the database files to
local_dir="${HERE}/../data/db-remote-${SITE_IP}-${VERSION}"

# Set the interval in seconds for fetching the files
if [ -z "$INTERVAL" ]; then
    INTERVAL=86400 # default is 24 hours
fi

while true; do
    # Get the current date and time
    now=$(date +"%Y-%m-%d-%H-%M-%S")

    # Set the filename for the backup file
    backup_filename="db-backup-$now.sql"

    # Create the backup file using the mysqldump command
    # ssh $remote_server "pg_dump -Fc -U ${DB_USER}" >"$local_dir/$backup_filename"
    ssh -i ~/.ssh/id_rsa_${SITE_IP} $remote_server "cd /var/tmp/${VERSION}/data && tar -czf - postgres" >"$local_dir/$backup_filename"

    # Compress the backup file to save disk space
    gzip "$local_dir/$backup_filename"

    # Remove old backup files to keep only the most recent k backups
    ls -t "$local_dir"/*.sql.gz | tail -n +$((BACKUP_COUNT + 1)) | xargs rm --

    # Log the backup operation
    info "Backup created: ${local_dir}/${backup_filename}.gz"

    # If not running on a loop, exit the script
    if [ "$WILL_LOOP" = false ]; then
        exit 0
    fi
    # Otherwise, wait for the specified interval before creating the next backup
    info "Waiting $INTERVAL seconds before creating the next backup..."
    sleep $INTERVAL
done
