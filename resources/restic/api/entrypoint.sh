#!/bin/sh
# Entrypoint script to run both API and scheduler

set -e

# Start backup scheduler in background if enabled
if [[ "${ENABLE_SCHEDULER:-true}" == "true" ]]; then
    echo "Starting backup scheduler..."
    /usr/local/bin/backup-scheduler.sh daemon &
fi

# Start API server
echo "Starting Restic API server..."
exec /usr/local/bin/restic-api