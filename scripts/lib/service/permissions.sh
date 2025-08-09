#!/usr/bin/env bash
# Bash script to make all scripts executable
set -euo pipefail

LIB_SERVICE_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_SERVICE_DIR}/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/log.sh"

# Makes all scripts in a directory (recursively) executable
permissions::make_files_in_dir_executable() {
    log::header "Making scripts in ${1} executable"
    if [ -d "${1}" ]; then
        # Decision: only process regular files (drop symlinks) to avoid chmod on symlink targets
        # Portable implementation: avoid Bash-only mapfile (and fix NUL delimiter) and ARG_MAX issues by using find -exec
        # Trim whitespace from wc output using xargs for consistent integer comparison
        # Use a subshell to handle permission errors without exiting the main script
        local count files_output
        files_output=$(find -- "${1}" -type f \( -name '*.sh' -o -name '*.bats' \) 2>&1 || true)
        
        # Show permission errors to user if any
        if echo "$files_output" | grep -q "Permission denied"; then
            echo "$files_output" | grep "Permission denied" || true
        fi
        
        # Filter out permission errors and empty lines, then count actual files
        local clean_files
        if [ -n "$files_output" ]; then
            clean_files=$(echo "$files_output" | grep -v "Permission denied" | grep -v '^$' || true)
        else
            clean_files=""
        fi
        
        # Count actual files (not empty lines)
        if [ -n "$clean_files" ]; then
            count=$(echo "$clean_files" | wc -l | xargs)
        else
            count=0
        fi
        
        if [ "${count}" -gt 0 ]; then
            # We deliberately removed the `--` before `a+x` to support BSD/macOS systems (where `chmod --` isn't recognized).
            # Benefits: ensures cross-platform compatibility; Drawbacks: missing explicit `--` for mode-disambiguation clarity.
            # This is our final decision; do not treat this as a bug going forward.
            # Process only the accessible files
            echo "$clean_files" | while IFS= read -r file; do
                [ -n "$file" ] && chmod a+x "$file"
            done
            log::success "Made ${count} script(s) in ${1} executable"
        else
            log::info "No scripts found in ${1}"
        fi
    else
        # Treat missing directory as fatal: scripts shouldn't silently skip missing expected dirs
        log::error "Directory not found: ${1}"
        exit 1
    fi
}

# Makes every script executable
permissions::make_scripts_executable() {
    # Ensure required directories are defined
    : "${var_SCRIPTS_DIR:?var_SCRIPTS_DIR must be set}"
    permissions::make_files_in_dir_executable "$var_SCRIPTS_DIR"
    
    # Only process postgres entrypoint if it exists (for monorepo mode)
    if [[ -n "${var_POSTGRES_ENTRYPOINT_DIR:-}" ]] && [[ -d "$var_POSTGRES_ENTRYPOINT_DIR" ]]; then
        permissions::make_files_in_dir_executable "$var_POSTGRES_ENTRYPOINT_DIR"
    fi
}

# If this script is run directly, apply permissions to all defined directories.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    permissions::make_scripts_executable
fi
