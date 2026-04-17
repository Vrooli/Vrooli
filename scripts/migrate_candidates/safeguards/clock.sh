#!/usr/bin/env bash
# Script to verify and fix system clock accuracy.
# An accurate system clock is critical for SSL/TLS certificate validation.
# This script first checks if the clock is accurate (within 5 minutes) by comparing
# to network time sources. Only if the clock is inaccurate does it require sudo
# permissions to synchronize.
set -euo pipefail

# Source var.sh with cached APP_ROOT pattern
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/flow.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_SYSTEM_DIR/system_commands.sh"

# Check if system clock is reasonably accurate (within 5 minutes)
clock::is_accurate() {
    # Get current system time
    local system_time
    system_time=$(date +%s)
    
    # Try to get accurate time from an HTTPS endpoint
    # Using major CDN endpoints that are fast and reliable
    local remote_time=""
    if system::is_command "curl"; then
        # Try multiple endpoints in case one is blocked/slow
        local endpoints=("https://www.google.com" "https://www.cloudflare.com" "https://httpbin.org/get")
        for endpoint in "${endpoints[@]}"; do
            # Try to get time from HTTPS response header (timeout after 3 seconds)
            local date_header
            date_header=$(curl -sI --max-time 3 "$endpoint" 2>/dev/null | grep -i "^date:" | cut -d' ' -f2-)
            if [[ -n "$date_header" ]]; then
                # Convert to epoch time
                remote_time=$(date -d "$date_header" +%s 2>/dev/null || echo "")
                if [[ -n "$remote_time" ]]; then
                    break  # Successfully got time
                fi
            fi
        done
    fi
    
    # If we couldn't get remote time, do basic sanity checks
    if [[ -z "$remote_time" ]]; then
        # Check if time is reasonable (between 2020 and 2035)
        local year_2020=1577836800  # Jan 1, 2020
        local year_2035=2051222400  # Jan 1, 2035
        
        if [[ $system_time -lt $year_2020 ]] || [[ $system_time -gt $year_2035 ]]; then
            log::warning "System clock appears to be significantly wrong (year is not 2020-2035)"
            return 1
        fi
        
        # If we can't check against network time, assume it's OK if in reasonable range
        log::info "Cannot verify exact time accuracy (no network time source), but year is reasonable"
        return 0
    fi
    
    # Calculate time difference
    local time_diff=$((system_time - remote_time))
    # Get absolute value
    if [[ $time_diff -lt 0 ]]; then
        time_diff=$((0 - time_diff))
    fi
    
    # Allow up to 5 minutes difference (300 seconds)
    if [[ $time_diff -gt 300 ]]; then
        log::warning "System clock is off by $time_diff seconds"
        return 1
    fi
    
    log::info "System clock is accurate (within $time_diff seconds)"
    return 0
}

# Fix the system clock
clock::fix() {
    log::header "Making sure the system clock is accurate"
    
    # First check if clock is already accurate
    if clock::is_accurate; then
        log::success "System clock is accurate: $(date)"
        return 0
    fi
    
    # Clock is not accurate, try to fix it
    log::warning "System clock needs adjustment"
    
    # Check for sudo capability
    if ! flow::can_run_sudo "system clock synchronization (hwclock -s)"; then
        log::error "System clock is inaccurate but cannot sync without sudo access"
        log::warning "This may cause SSL/TLS certificate validation errors"
        log::info "Options:"
        log::info "  1. Run setup with sudo: sudo ./scripts/manage.sh setup"
        log::info "  2. Fix clock manually: sudo ntpdate -s time.nist.gov"
        log::info "  3. Continue anyway: ./scripts/manage.sh setup --sudo-mode skip"
        
        # Exit with error since clock sync is important for certificates
        exit "${ERROR_DEFAULT}"
    fi
    
    # Try to sync the clock
    local synced=false
    
    # Try hwclock first
    if system::is_command "hwclock"; then
        log::info "Syncing hardware clock..."
        if sudo hwclock -s 2>/dev/null; then
            synced=true
        fi
    fi
    
    # Try ntpdate if hwclock didn't work
    if [[ "$synced" == "false" ]] && system::is_command "ntpdate"; then
        log::info "Syncing with NTP server..."
        if sudo ntpdate -s time.nist.gov 2>/dev/null; then
            synced=true
        fi
    fi
    
    # Try timedatectl if available
    if [[ "$synced" == "false" ]] && system::is_command "timedatectl"; then
        log::info "Syncing with timedatectl..."
        if sudo timedatectl set-ntp true 2>/dev/null; then
            synced=true
        fi
    fi
    
    if [[ "$synced" == "true" ]]; then
        log::success "System clock synchronized: $(date)"
    else
        log::warning "Could not automatically sync clock, but will continue"
        log::info "Current time: $(date)"
    fi
}

# If this script is run directly, invoke its main function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    clock::fix "$@"
fi