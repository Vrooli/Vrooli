#!/usr/bin/env bash
#######################################
# Git Utilities Library
# Provides git-related functions for the Vrooli platform
#######################################

set -euo pipefail

#######################################
# Get current git commit hash
# Returns: 
#   Git commit hash or "no-git" if not in a git repository
#######################################
git::get_commit() {
    git rev-parse HEAD 2>/dev/null || echo "no-git"
}

#######################################
# Check if there are uncommitted changes in the repository
# Returns:
#   0 if there are uncommitted changes
#   1 if the working tree is clean
#######################################
git::has_uncommitted_changes() {
    ! git diff-index --quiet HEAD -- 2>/dev/null
}

#######################################
# Get short commit hash (first 8 characters)
# Returns: 
#   Short git commit hash or "no-git" if not in a git repository
#######################################
git::get_short_commit() {
    local commit=$(git::get_commit)
    if [[ "$commit" != "no-git" ]]; then
        echo "${commit:0:8}"
    else
        echo "no-git"
    fi
}

#######################################
# Check if current directory is a git repository
# Returns:
#   0 if in a git repository
#   1 if not in a git repository
#######################################
git::is_repository() {
    git rev-parse --is-inside-work-tree >/dev/null 2>&1
}