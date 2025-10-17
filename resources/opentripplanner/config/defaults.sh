#!/usr/bin/env bash
# OpenTripPlanner Default Configuration

# Determine APP_ROOT if not set
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source port registry to get the allocated port (no fallback to avoid conflicts)
source "${APP_ROOT}/scripts/resources/port_registry.sh" || {
    echo "Error: Failed to source port_registry.sh" >&2
    exit 1
}

# Service configuration - port from registry
export OTP_PORT="${RESOURCE_PORTS["opentripplanner"]}"
export OTP_HEAP_SIZE="${OTP_HEAP_SIZE:-2G}"
export OTP_BUILD_TIMEOUT="${OTP_BUILD_TIMEOUT:-300}"

# Data directories
export OTP_DATA_DIR="${OTP_DATA_DIR:-${HOME}/.vrooli/opentripplanner/data}"
export OTP_CACHE_DIR="${OTP_CACHE_DIR:-${HOME}/.vrooli/opentripplanner/cache}"

# Docker configuration
export OPENTRIPPLANNER_IMAGE="${OPENTRIPPLANNER_IMAGE:-opentripplanner/opentripplanner:latest-jvm17}"
export OPENTRIPPLANNER_CONTAINER="${OPENTRIPPLANNER_CONTAINER:-vrooli-opentripplanner}"
export OPENTRIPPLANNER_NETWORK="${OPENTRIPPLANNER_NETWORK:-vrooli-network}"

# Graph building options
export OTP_BUILD_STREET_GRAPH="${OTP_BUILD_STREET_GRAPH:-true}"
export OTP_BUILD_TRANSIT_GRAPH="${OTP_BUILD_TRANSIT_GRAPH:-true}"
export OTP_CACHE_GRAPHS="${OTP_CACHE_GRAPHS:-true}"

# Routing defaults
export OTP_MAX_WALK_DISTANCE="${OTP_MAX_WALK_DISTANCE:-800}"
export OTP_MAX_TRANSFERS="${OTP_MAX_TRANSFERS:-3}"
export OTP_WALK_SPEED="${OTP_WALK_SPEED:-1.34}"
export OTP_BIKE_SPEED="${OTP_BIKE_SPEED:-5.0}"