#!/bin/bash
# Restic backup scheduler script

set -euo pipefail

# Default backup paths (can be overridden by environment variables)
BACKUP_PATHS="${BACKUP_PATHS:-/backup/home/matthalloran8/Vrooli,/backup/volumes}"
BACKUP_SCHEDULE="${BACKUP_SCHEDULE:-0 2 * * *}"  # Default: 2 AM daily

# Function to run scheduled backup
run_backup() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Starting scheduled backup..."
    
    # Create backup using API
    curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "{\"paths\":[\"${BACKUP_PATHS//,/\",\"}\"],\"tags\":[\"scheduled\",\"daily\"]}" \
        "http://localhost:8000/backup"
    
    if [[ $? -eq 0 ]]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Backup completed successfully"
    else
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Backup failed"
    fi
}

# Check if we should run in daemon mode
if [[ "${1:-}" == "daemon" ]]; then
    echo "Starting backup scheduler daemon..."
    echo "Schedule: $BACKUP_SCHEDULE"
    echo "Paths: $BACKUP_PATHS"
    
    # Create cron job
    echo "$BACKUP_SCHEDULE /usr/local/bin/backup-scheduler.sh run" > /etc/crontabs/root
    
    # Start cron daemon
    crond -f -l 2
elif [[ "${1:-}" == "run" ]]; then
    run_backup
else
    echo "Usage: $0 [daemon|run]"
    exit 1
fi