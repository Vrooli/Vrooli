#!/usr/bin/env bash
# Pi-hole Resource Configuration Defaults

# Service configuration
export PIHOLE_VERSION="${PIHOLE_VERSION:-latest}"
export PIHOLE_API_PORT="${PIHOLE_API_PORT:-8087}"
export PIHOLE_DNS_PORT="${PIHOLE_DNS_PORT:-53}"
export PIHOLE_DHCP_PORT="${PIHOLE_DHCP_PORT:-67}"

# Container configuration
export PIHOLE_CONTAINER_NAME="${PIHOLE_CONTAINER_NAME:-vrooli-pihole}"
export PIHOLE_DATA_DIR="${PIHOLE_DATA_DIR:-${HOME}/.vrooli/pihole}"

# DNS configuration
export PIHOLE_UPSTREAM_DNS_1="${PIHOLE_UPSTREAM_DNS_1:-1.1.1.1}"
export PIHOLE_UPSTREAM_DNS_2="${PIHOLE_UPSTREAM_DNS_2:-1.0.0.1}"

# Feature flags
export PIHOLE_ENABLE_DHCP="${PIHOLE_ENABLE_DHCP:-false}"
export PIHOLE_ENABLE_IPV6="${PIHOLE_ENABLE_IPV6:-true}"
export PIHOLE_QUERY_LOGGING="${PIHOLE_QUERY_LOGGING:-true}"

# Performance tuning
export PIHOLE_CACHE_SIZE="${PIHOLE_CACHE_SIZE:-10000}"
export PIHOLE_MAX_QUERIES="${PIHOLE_MAX_QUERIES:-1000000}"

# Security settings
export PIHOLE_WEBPASSWORD_HASH="${PIHOLE_WEBPASSWORD_HASH:-}"
export PIHOLE_TEMP_UNIT="${PIHOLE_TEMP_UNIT:-C}"
export PIHOLE_INTERFACE="${PIHOLE_INTERFACE:-eth0}"

# Blocklist settings
export PIHOLE_UPDATE_GRAVITY_ON_START="${PIHOLE_UPDATE_GRAVITY_ON_START:-true}"
export PIHOLE_SKIPGRAVITYONBOOT="${PIHOLE_SKIPGRAVITYONBOOT:-false}"

# Default blocklists
export PIHOLE_DEFAULT_BLOCKLISTS=(
    "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
    "https://someonewhocares.org/hosts/zero/hosts"
    "https://raw.githubusercontent.com/crazy-max/WindowsSpyBlocker/master/data/hosts/spy.txt"
    "https://hostfiles.frogeye.fr/firstparty-trackers-hosts.txt"
)