#!/usr/bin/env bash

# Network Utilities Library - TCP connection monitoring and process counting
# Part of the modular loop system
#
# This module provides simplified network monitoring functions with a focus
# on counting worker processes and TCP connections for rate limiting.

set -euo pipefail

# Prevent multiple sourcing
if [[ -n "${_AUTO_NETWORK_UTILS_SOURCED:-}" ]]; then
    return 0
fi
readonly _AUTO_NETWORK_UTILS_SOURCED=1

# Source constants
LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$LIB_DIR/constants.sh"

# Count processes matching a pattern
# Args: $1 - pattern (optional, defaults to claude-related processes)
# Returns: count of matching processes
# Outputs: process count to stdout
network_utils::count_worker_processes() {
    local pattern="${1:-claude|anthropic|resource-claude-code}"
    
    # Return 0 if pattern is empty (disabled)
    if [[ -z "$pattern" ]]; then
        echo "0"
        return 0
    fi
    
    local count=0
    
    # Primary method: Use pgrep for accurate process counting
    if command -v pgrep >/dev/null 2>&1; then
        # Split pattern by | and count each separately to avoid shell interpretation issues
        IFS='|' read -ra patterns <<< "$pattern"
        for p in "${patterns[@]}"; do
            local partial_count
            partial_count=$(pgrep -fc "$p" 2>/dev/null || echo 0)
            count=$((count + partial_count))
        done
        echo "$count"
        return 0
    fi
    
    # Fallback: Use ps with grep
    if command -v ps >/dev/null 2>&1; then
        count=$(ps aux 2>/dev/null | grep -E "$pattern" | grep -v grep | wc -l)
        echo "$count"
        return 0
    fi
    
    # No process counting available
    echo "0"
    return 1
}

# Count TCP connections for specific processes
# Args: $1 - process filter pattern (optional)
# Returns: count of TCP connections
# Note: This is a more expensive operation than process counting
network_utils::count_tcp_connections() {
    local filter="${1:-}"
    
    # Return 0 if filter is empty (disabled)
    if [[ -z "$filter" ]]; then
        echo "0"
        return 0
    fi
    
    local count=0
    
    # Try ss first (modern, faster)
    if command -v ss >/dev/null 2>&1; then
        # Count established TCP connections with process info
        count=$(ss -ptn state established 2>/dev/null | grep -E "$filter" | wc -l)
        echo "$count"
        return 0
    fi
    
    # Fallback to netstat
    if command -v netstat >/dev/null 2>&1; then
        count=$(netstat -tnp 2>/dev/null | grep ESTABLISHED | grep -E "$filter" | wc -l)
        echo "$count"
        return 0
    fi
    
    # No network monitoring available
    echo "0"
    return 1
}

# Check if worker connections are within limits
# Args: $1 - max connections (optional, defaults to MAX_TCP_CONNECTIONS)
#       $2 - filter pattern (optional, defaults to LOOP_TCP_FILTER)
# Returns: 0 if within limits, 1 if exceeded
network_utils::check_connection_limit() {
    local max_connections="${1:-${MAX_TCP_CONNECTIONS:-15}}"
    local filter="${2:-${LOOP_TCP_FILTER:-}}"
    
    # If filter is empty, gating is disabled
    if [[ -z "$filter" ]]; then
        return 0
    fi
    
    # Count worker processes (preferred method - faster and more reliable)
    local count
    count=$(network_utils::count_worker_processes "$filter")
    
    if [[ $count -gt $max_connections ]]; then
        if declare -F log_with_timestamp >/dev/null 2>&1; then
            log_with_timestamp "Connection limit exceeded: $count processes > $max_connections limit"
        fi
        return 1
    fi
    
    return 0
}

# Get detailed connection stats for debugging
# Args: $1 - filter pattern (optional)
# Returns: JSON object with connection stats
network_utils::get_connection_stats() {
    local filter="${1:-${LOOP_TCP_FILTER:-}}"
    
    local process_count
    process_count=$(network_utils::count_worker_processes "$filter")
    
    local tcp_count
    tcp_count=$(network_utils::count_tcp_connections "$filter")
    
    # Output as JSON
    printf '{"process_count":%d,"tcp_connections":%d,"filter":"%s","timestamp":"%s"}\n' \
        "$process_count" \
        "$tcp_count" \
        "$filter" \
        "$(date -Is)"
}

# Wait for connections to drop below threshold
# Args: $1 - max connections, $2 - filter pattern, $3 - max wait seconds (optional, default 30)
# Returns: 0 if connections dropped, 1 if timeout
network_utils::wait_for_connection_drop() {
    local max_connections="${1:-${MAX_TCP_CONNECTIONS:-15}}"
    local filter="${2:-${LOOP_TCP_FILTER:-}}"
    local max_wait="${3:-30}"
    
    local waited=0
    while [[ $waited -lt $max_wait ]]; do
        if network_utils::check_connection_limit "$max_connections" "$filter"; then
            return 0
        fi
        sleep 1
        ((waited++))
    done
    
    if declare -F log_with_timestamp >/dev/null 2>&1; then
        log_with_timestamp "Timeout waiting for connections to drop below $max_connections (waited ${max_wait}s)"
    fi
    return 1
}

# Check if specific port is in use
# Args: $1 - port number
# Returns: 0 if in use, 1 if free
network_utils::is_port_in_use() {
    local port="$1"
    
    # Try ss first
    if command -v ss >/dev/null 2>&1; then
        if ss -ln 2>/dev/null | grep -q ":$port "; then
            return 0
        fi
    # Fallback to netstat
    elif command -v netstat >/dev/null 2>&1; then
        if netstat -ln 2>/dev/null | grep -q ":$port "; then
            return 0
        fi
    # Last resort: try to bind to the port
    elif command -v nc >/dev/null 2>&1; then
        if ! nc -z localhost "$port" 2>/dev/null; then
            return 1
        fi
        return 0
    fi
    
    return 1
}

# Export functions for external use
export -f network_utils::count_worker_processes
export -f network_utils::count_tcp_connections
export -f network_utils::check_connection_limit
export -f network_utils::get_connection_stats
export -f network_utils::wait_for_connection_drop
export -f network_utils::is_port_in_use