#!/usr/bin/env bash
# This script periodically backs up the database and essential files from a remote server
set -euo pipefail

MAIN_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/env.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/keyless_ssh.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/var.sh"

# Default values
BACKUP_COUNT="5"

backup::do() {
    if [ -z "$SITE_IP" ]; then
        log::error "SITE_IP not set in environment"
        exit $ERROR_USAGE
    fi

    # Get SSH key path using helper
    local ssh_key
    ssh_key=$(keyless_ssh::get_key_path)

    # Set the remote server location, using SITE_IP from .env
    remote_server="root@${SITE_IP}"
    log::info "Remote server: ${remote_server}"

    # Fetch the version number from the package.json on the remote server
    VERSION=$(ssh -i "$ssh_key" "$remote_server" "jq -r .version < \"${var_REMOTE_ROOT_DIR}/package.json\"")
    log::info "Version number retrieved from remote package.json: ${VERSION}"
    # Sanitize VERSION to allow only safe characters in filename
    if [[ ! "${VERSION}" =~ ^[A-Za-z0-9._-]+$ ]]; then
        log::warning "VERSION contains unsafe characters; defaulting to 'unknown'"
        VERSION="unknown"
    fi

    # Set the local directory to save the backup files to
    backup_root_dir="${var_BACKUPS_DIR}/${SITE_IP}"
    local_dir="${backup_root_dir}/$(date +"%Y%m%d%H%M%S")"

    # Create the backup directory
    mkdir -p "${local_dir}"

    # Backup the database, data directory, JWT files, and .env* files
    ssh -i "$ssh_key" "$remote_server" "cd ${var_REMOTE_ROOT_DIR} && tar --ignore-failed-read -czf - data/postgres-prod jwt_* .env*" >"${local_dir}/backup-$VERSION.tar.gz"

    # Remove old backup directories to keep only the most recent BACKUP_COUNT backups
    shopt -s nullglob
    mapfile -t dirs < <(printf '%s\n' "$backup_root_dir"/*/ | sort -r)
    shopt -u nullglob
    if [ "${#dirs[@]}" -gt "$BACKUP_COUNT" ]; then
        for dir in "${dirs[@]:$BACKUP_COUNT}"; do
            rm -rf -- "$dir"
        done
    fi

    # Log the backup operation
    log::info "Backup created: ${local_dir}/backup-$VERSION.tar.gz"
}

backup::init() {
    export NODE_ENV="${NODE_ENV:-production}"
    env::load_env_file
    keyless_ssh::connect
}

backup::schedule() {
    # Load environment only (skip SSH connect for scheduling)
    export NODE_ENV="${NODE_ENV:-production}"
    env::load_env_file

    LOG_DIR="${var_ROOT_DIR}/data"
    mkdir -p "${LOG_DIR}"

    # Define cron schedule and command
    CRON_SCHEDULE="@daily"
    CRON_CMD="${MAIN_DIR}/backup.sh run_backup"
    # Cron entry: append stdout to backup.log and stderr to backup.err
    CRON_ENTRY="${CRON_SCHEDULE} ${CRON_CMD} >> ${LOG_DIR}/backup.log 2>> ${LOG_DIR}/backup.err"

    # Install cron entry if not already present
    if crontab -l 2>/dev/null | grep -F "${CRON_CMD}" >/dev/null; then
        log::info "Backup cron already installed"
    else
        (crontab -l 2>/dev/null || true; echo "${CRON_ENTRY}") | crontab -
        log::success "✅ Scheduled backup cron job: ${CRON_ENTRY}"
    fi

    # Start a backup immediately
    keyless_ssh::connect
    backup::do
}

backup::main() {
    if [[ "${1:-}" == "run_backup" ]]; then
        backup::init
        backup::do
    else
        backup::schedule
    fi
}

backup::main "$@"
