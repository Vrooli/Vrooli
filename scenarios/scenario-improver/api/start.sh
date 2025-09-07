#!/bin/bash

# Scenario Improver API Startup Script

# Configuration
export API_PORT="${API_PORT}"
export QUEUE_DIR="${QUEUE_DIR:-../queue}"
export ENABLE_LOG_MONITORING="${ENABLE_LOG_MONITORING:-false}"
export ENABLE_PERFORMANCE_MONITORING="${ENABLE_PERFORMANCE_MONITORING:-false}"

# Check if API_PORT is set, if not find available port
if [[ -z "$API_PORT" ]]; then
    echo "No API_PORT specified, finding available port..."
    # Find an available port starting from 30150
    API_PORT=30150
    while lsof -i :$API_PORT >/dev/null 2>&1; do
        API_PORT=$((API_PORT + 1))
    done
    export API_PORT
    echo "Selected available port: $API_PORT"
elif lsof -i :$API_PORT >/dev/null 2>&1; then
    echo "Error: Port $API_PORT is already in use"
    echo "You can specify a different port with: API_PORT=30151 ./start.sh"
    exit 1
fi

# Check resource-claude-code is available
if ! command -v resource-claude-code &> /dev/null; then
    echo "Warning: resource-claude-code not found in PATH"
    echo "Claude Code integration may not work properly"
    echo "Install with: vrooli resource claude-code install"
fi

# Create log directory if monitoring is enabled
if [ "$ENABLE_LOG_MONITORING" = "true" ]; then
    mkdir -p /tmp/vrooli/logs
    echo "Log monitoring enabled (writing to /tmp/vrooli/logs)"
fi

# Start the API
echo "Starting Scenario Improver API on port $API_PORT..."
echo "Queue directory: $QUEUE_DIR"
echo "Log monitoring: $ENABLE_LOG_MONITORING"
echo "Performance monitoring: $ENABLE_PERFORMANCE_MONITORING"
echo ""
echo "Access the API at: http://localhost:$API_PORT"
echo "Health check: http://localhost:$API_PORT/health"
echo ""
echo "Press Ctrl+C to stop the server"

# Run the API
exec ./scenario-improver-api