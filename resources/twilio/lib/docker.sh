#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_LIB_DIR="${APP_ROOT}/resources/twilio/lib"

# Source common functions
source "$TWILIO_LIB_DIR/common.sh"

# Twilio doesn't use Docker containers - it's a cloud service
# These handlers provide appropriate responses for a cloud-based service

twilio::docker::start() {
    # For Twilio (cloud service), "start" means ensure auth is set up and monitor is running
    twilio::start "$@"
}

twilio::docker::stop() {
    # For Twilio (cloud service), "stop" means stop the local monitor
    twilio::stop "$@"
}

twilio::docker::restart() {
    # For Twilio (cloud service), "restart" means restart the monitor
    log::header "🔄 Restarting Twilio Monitor"
    twilio::stop
    twilio::start "$@"
}

twilio::docker::logs() {
    # Show Twilio logs
    twilio::logs "$@"
}