#!/usr/bin/env bash
################################################################################
# Hash Utilities
# 
# Generic hash computation and file change detection functions
################################################################################

set -euo pipefail

# Source dependencies
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/system_commands.sh"

# Compute hash of a file for change detection
hash::compute_file_hash() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo ""
        return 1
    fi
    
    if system::is_command shasum; then
        shasum -a 256 "$file" | awk '{print $1}'
    elif system::is_command sha256sum; then
        sha256sum "$file" | awk '{print $1}'
    else
        log::error "No hash utility found (shasum or sha256sum required)"
        return 1
    fi
}

# Check if a file has changed by comparing hashes
hash::has_file_changed() {
    local file="$1"
    local hash_file="${2:-${file}.hash}"
    
    # Compute current hash
    local current_hash
    current_hash=$(hash::compute_file_hash "$file") || return 1
    
    # Read previous hash if exists
    local prev_hash=""
    if [[ -f "$hash_file" ]]; then
        prev_hash=$(cat "$hash_file")
    fi
    
    # Compare hashes
    if [[ "$current_hash" == "$prev_hash" ]]; then
        return 1  # File has not changed
    else
        # Save new hash
        echo "$current_hash" > "$hash_file"
        return 0  # File has changed
    fi
}