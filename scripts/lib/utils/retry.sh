#!/usr/bin/env bash
################################################################################
# Retry Utilities
# 
# Generic retry logic for commands that may fail temporarily
################################################################################

set -euo pipefail

# Source dependencies
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

#######################################
# Execute a command with retry logic
# Arguments:
#   $1 - Maximum number of attempts (default: 3)
#   $2 - Delay between attempts in seconds (default: 2)
#   $3+ - Command to execute
# Returns:
#   0 if command succeeds, 1 if all attempts fail
#######################################
retry::execute() {
    local max_attempts="${1:-3}"
    local delay="${2:-2}"
    shift 2
    local command=("$@")
    
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log::debug "Attempt $attempt of $max_attempts: ${command[*]}"
        
        if "${command[@]}"; then
            return 0
        fi
        
        local exit_code=$?
        
        if [[ $attempt -lt $max_attempts ]]; then
            log::warning "Command failed with exit code $exit_code, retrying in ${delay} seconds..."
            sleep "$delay"
        else
            log::error "Command failed after $max_attempts attempts"
        fi
        
        ((attempt++))
    done
    
    return 1
}

#######################################
# Download a URL with retry logic
# Specialized function for curl downloads
# Arguments:
#   $1 - URL to download
#   $2 - Maximum attempts (optional, default: 3)
#   $3 - Delay between attempts (optional, default: 2)
# Returns:
#   0 if download succeeds, 1 if all attempts fail
#######################################
retry::download() {
    local url="$1"
    local max_attempts="${2:-3}"
    local delay="${3:-2}"
    
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log::info "Download attempt $attempt of $max_attempts..."
        
        if curl -fsSL "$url"; then
            return 0
        fi
        
        if [[ $attempt -lt $max_attempts ]]; then
            log::warning "Download failed, retrying in ${delay} seconds..."
            sleep "$delay"
        fi
        
        ((attempt++))
    done
    
    log::error "Download failed after $max_attempts attempts"
    return 1
}

#######################################
# Execute with exponential backoff
# Each retry doubles the delay time
# Arguments:
#   $1 - Maximum number of attempts (default: 3)
#   $2 - Initial delay in seconds (default: 1)
#   $3+ - Command to execute
# Returns:
#   0 if command succeeds, 1 if all attempts fail
#######################################
retry::exponential_backoff() {
    local max_attempts="${1:-3}"
    local initial_delay="${2:-1}"
    shift 2
    local command=("$@")
    
    local attempt=1
    local delay=$initial_delay
    
    while [[ $attempt -le $max_attempts ]]; do
        log::debug "Attempt $attempt of $max_attempts (delay: ${delay}s): ${command[*]}"
        
        if "${command[@]}"; then
            return 0
        fi
        
        if [[ $attempt -lt $max_attempts ]]; then
            log::warning "Command failed, retrying in ${delay} seconds..."
            sleep "$delay"
            delay=$((delay * 2))  # Double the delay for next attempt
        fi
        
        ((attempt++))
    done
    
    log::error "Command failed after $max_attempts attempts"
    return 1
}