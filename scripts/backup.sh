#!/bin/bash
# This script periodically backs up the database and essential files from a remote server

# Exit codes
export ERROR_USAGE=64
export ERROR_INVALID_INPUT=65

HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

# Default values
BACKUP_COUNT="5"
INTERVAL=86400 # 24 hours
WILL_LOOP=false

# Validation constants
MIN_BACKUP_COUNT=1
MAX_BACKUP_COUNT=100
MIN_INTERVAL=300     # 5 minutes
MAX_INTERVAL=2592000 # 30 days

usage() {
    cat <<EOF
Usage: $(basename "$0") [-c COUNT] [-h HELP] [-i INTERVAL] [-l LOOP]
  -c --count: The number of most recent backup files to keep
  -h --help: Show this help message
  -i --interval: The interval in seconds for fetching the logs, if running on a loop
  -l --loop: Whether to run this script on a loop, or to exit after one run

Exit Codes:
    0                                  Success
    $ERROR_USAGE                       Command line usage error
    $ERROR_INVALID_INPUT               Invalid input parameters

EOF
}

validate_number() {
    local value=$1
    local min=$2
    local max=$3
    local param_name=$4

    if ! [[ "$value" =~ ^[0-9]+$ ]]; then
        echo "Error: $param_name must be a number"
        exit $ERROR_INVALID_INPUT
    fi

    if [ "$value" -lt "$min" ] || [ "$value" -gt "$max" ]; then
        echo "Error: $param_name must be between $min and $max"
        exit $ERROR_INVALID_INPUT
    fi
}

validate_boolean() {
    local value=$1
    local param_name=$2

    if ! [[ "$value" =~ ^(true|false)$ ]]; then
        echo "Error: $param_name must be 'true' or 'false'"
        exit $ERROR_INVALID_INPUT
    fi
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        key="$1"
        case $key in
        -h | --help)
            usage
            exit 0
            ;;
        -c | --count)
            if [ -z "$2" ] || [[ "$2" == -* ]]; then
                echo "Error: Option $key requires an argument."
                exit $ERROR_USAGE
            fi
            BACKUP_COUNT="${2}"
            validate_number "$BACKUP_COUNT" $MIN_BACKUP_COUNT $MAX_BACKUP_COUNT "backup count"
            shift # past argument
            shift # past value
            ;;
        -i | --interval)
            if [ -z "$2" ] || [[ "$2" == -* ]]; then
                echo "Error: Option $key requires an argument."
                exit $ERROR_USAGE
            fi
            INTERVAL="${2}"
            validate_number "$INTERVAL" $MIN_INTERVAL $MAX_INTERVAL "interval"
            shift # past argument
            shift # past value
            ;;
        -l | --loop)
            if [ -z "$2" ] || [[ "$2" == -* ]]; then
                echo "Error: Option $key requires an argument."
                exit $ERROR_USAGE
            fi
            WILL_LOOP="${2}"
            validate_boolean "$WILL_LOOP" "loop"
            shift # past argument
            shift # past value
            ;;
        *)
            # Unknown option
            echo "Unknown option: $1"
            shift # past argument
            ;;
        esac
    done
}

start_backup_loop() {
    if [ -z "$SITE_IP" ]; then
        echo "Error: SITE_IP not set in environment"
        exit $ERROR_INVALID_INPUT
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
}

main() {
    parse_arguments "$@"

    load_env_file "production"

    "${HERE}/keylessSsh.sh"

    start_backup_loop
}

run_if_executed main "$@"
