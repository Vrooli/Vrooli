#!/bin/bash
# Docker Mock Helpers
# Provides helpers for mocking Docker operations in tests

# Mock docker state storage
declare -A MOCK_DOCKER_CONTAINER_STATE
declare -A MOCK_DOCKER_CONTAINER_HEALTH

#######################################
# Set mock container state
# Globals: MOCK_DOCKER_CONTAINER_STATE
# Arguments:
#   $1 - container name
#   $2 - state (running, stopped, etc)
# Returns: 0
#######################################
mock::docker::set_container_state() {
    local container="$1"
    local state="$2"
    
    MOCK_DOCKER_CONTAINER_STATE["$container"]="$state"
    return 0
}

#######################################
# Set mock container health
# Globals: MOCK_DOCKER_CONTAINER_HEALTH
# Arguments:
#   $1 - container name
#   $2 - health status
# Returns: 0
#######################################
mock::docker::set_container_health() {
    local container="$1"
    local health="$2"
    
    MOCK_DOCKER_CONTAINER_HEALTH["$container"]="$health"
    return 0
}

#######################################
# Get mock container state
# Globals: MOCK_DOCKER_CONTAINER_STATE
# Arguments:
#   $1 - container name
# Returns: 0 if running, 1 otherwise
#######################################
mock::docker::get_container_state() {
    local container="$1"
    local state="${MOCK_DOCKER_CONTAINER_STATE[$container]:-stopped}"
    
    echo "$state"
    [[ "$state" == "running" ]] && return 0 || return 1
}

# Export functions
export -f mock::docker::set_container_state
export -f mock::docker::set_container_health
export -f mock::docker::get_container_state