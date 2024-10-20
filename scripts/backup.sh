#!/bin/bash
# This script periodically backs up the database and essential files from a remote server
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"
. "${HERE}/keylessSsh.sh"

# Read arguments
while getopts "c:hi:l:v:" opt; do
    case $opt in
    h)
        echo "Usage: $0 [-c COUNT] [-h HELP] [-i INTERVAL] [-l LOOP]"
        echo "  -c --count: The number of most recent backup files to keep"
        echo "  -h --help: Show this help message"
        echo "  -i --interval: The interval in seconds for fetching the logs, if running on a loop"
        echo "  -l --loop: Whether to run this script on a loop, or to exit after one run"
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
    WILL_LOOP=false
fi

# Set the remote server location, using SITE_IP from .env
remote_server="root@${SITE_IP}"
info "Remote server: ${remote_server}"

# Fetch the version number from the package.json on the remote server
VERSION=$(ssh -i ~/.ssh/id_rsa_${SITE_IP} $remote_server "cat /root/Vrooli/package.json | grep '\"version\":' | head -1 | awk -F: '{ print \$2 }' | sed 's/[\", ]//g'")
info "Version number retrieved from remote package.json: ${VERSION}"

# Set the local directory to save the backup files to
backup_root_dir="${HERE}/../backups/${SITE_IP}"
local_dir="${backup_root_dir}/$(date +"%Y%m%d%H%M%S")"

# Set the interval in seconds for fetching the files
if [ -z "$INTERVAL" ]; then
    INTERVAL=86400 # default is 24 hours
fi

while true; do
    # Create the backup directory
    mkdir -p "${local_dir}"

    # Backup the database, data directory, JWT files, and .env* files
    ssh -i ~/.ssh/id_rsa_${SITE_IP} $remote_server "cd /root/Vrooli && tar -czf - data/postgres-prod jwt_* .env*" >"${local_dir}/backup-$VERSION.tar.gz"

    # Remove old backup directories to keep only the most recent k backups
    ls -t "$backup_root_dir" | tail -n +$((BACKUP_COUNT + 1)) | xargs -I {} rm -r "$backup_root_dir"/{}

    # Log the backup operation
    info "Backup created: ${local_dir}/backup-$VERSION.tar.gz"

    # If not running on a loop, exit the script
    if [ "$WILL_LOOP" = false ]; then
        exit 0
    fi
    # Otherwise, wait for the specified interval before creating the next backup
    info "Waiting $INTERVAL seconds before creating the next backup..."
    sleep $INTERVAL
done
