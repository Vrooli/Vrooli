#!/usr/bin/env bash
# remote_session_protect.sh - Protect remote desktop sessions from memory pressure
# Prevents GDM3/desktop crashes during high memory usage and ensures easy recovery
set -euo pipefail

# Source var.sh with relative path first
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/flow.sh"
# shellcheck disable=SC1091
source "$var_LIB_SYSTEM_DIR/system_commands.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

# Configuration constants
readonly DESKTOP_MIN_PERCENT=15    # Minimum % of RAM for desktop
readonly DESKTOP_MIN_MB=4096       # Absolute minimum for desktop (MB)
readonly DESKTOP_BUFFER_MB=2048    # Additional soft buffer (MB)
readonly WORKLOAD_HIGH_PERCENT=85  # Start throttling workloads at this %
readonly WORKLOAD_MAX_PERCENT=95   # Hard cap for workloads
readonly MAX_SWAP_GB=64            # Maximum swap size (GB)
readonly SWAP_FILE="/swapfile"     # Swap file location
readonly DESKTOP_UID=1000          # Default desktop user UID

#######################################
# Check if remote desktop is installed/configured or if GUI is present
# Returns:
#   0 if remote desktop or GUI desktop detected, 1 otherwise
#######################################
remote_session::is_remote_desktop_installed() {
    # Check for display managers (indicates GUI system that could use remote access)
    if (systemctl list-unit-files | grep -qE "(gdm3?|lightdm|sddm|xdm)\.service") 2>/dev/null; then
        log::info "Display manager detected - system has GUI that may benefit from protections"
        return 0
    fi
    
    # Check for xrdp service
    if (systemctl list-unit-files | grep -qE "xrdp\.service") 2>/dev/null; then
        log::info "xrdp service detected"
        return 0
    fi
    
    # Check if RDP port is configured (check firewall.sh config)
    local firewall_script="${LIB_SYSTEM_DIR}/firewall.sh"
    if [[ -f "$firewall_script" ]] && grep -q "3389" "$firewall_script" 2>/dev/null; then
        log::info "RDP port configured in firewall setup"
        return 0
    fi
    
    # Check for VNC services
    if (systemctl list-unit-files | grep -qE "(vncserver|tigervnc)") 2>/dev/null; then
        log::info "VNC service detected"
        return 0
    fi
    
    # Check if GUI processes are running (X11 or Wayland)
    if pgrep -x "Xorg|Xwayland|gnome-shell|kde|xfce" >/dev/null 2>&1; then
        log::info "GUI desktop processes detected"
        return 0
    fi
    
    return 1
}

#######################################
# Check GDM3 service status and restart if needed
# Returns:
#   0 on success (healthy or recovered), 1 on failure
#######################################
remote_session::check_gdm3_status() {
    log::info "🖥️  Checking GDM3 display manager status..."
    
    # First, try to find which GDM service name is in use
    # Prefer gdm3 if it exists (even as alias) since that's what users typically use
    local gdm_service=""
    
    # Check if gdm3.service exists (either as main service or alias)
    # Note: grep -q with pipefail can cause SIGPIPE (141), so we temporarily disable set -e
    local check_result
    set +e  # Temporarily disable exit on error
    systemctl list-unit-files | grep -qE "gdm3\.service"
    check_result=$?
    set -e  # Re-enable exit on error
    
    if [[ $check_result -eq 0 ]] || [[ $check_result -eq 141 ]]; then
        gdm_service="gdm3"
        log::info "Found gdm3.service - using gdm3 for commands"
    else
        # Fall back to gdm.service if gdm3 doesn't exist
        set +e
        systemctl list-unit-files | grep -qE "gdm\.service"
        check_result=$?
        set -e
        if [[ $check_result -eq 0 ]] || [[ $check_result -eq 141 ]]; then
            gdm_service="gdm"
            log::info "Found gdm.service - using gdm for commands"
        fi
    fi
    
    # If we found GDM in any form, check its status
    if [[ -n "$gdm_service" ]]; then
        if systemctl is-active --quiet "${gdm_service}" 2>/dev/null; then
            log::success "✅ GDM service (${gdm_service}) is running"
            
            # Additional check: verify Xorg/Wayland processes
            if pgrep -x "Xorg|Xwayland|gnome-shell" >/dev/null 2>&1; then
                log::success "✅ Display server processes detected"
            else
                log::warning "GDM running but no display server processes found"
            fi
        else
            log::warning "⚠️  GDM service (${gdm_service}) is not running"
            
            if sudo::can_use_sudo; then
                log::info "Attempting to restart ${gdm_service}..."
                if sudo::exec_with_fallback "systemctl restart '${gdm_service}'" 2>/dev/null; then
                    log::success "✅ GDM restarted successfully"
                    sleep 3  # Give it time to start
                    
                    # Verify it's actually running now
                    if systemctl is-active --quiet "${gdm_service}" 2>/dev/null; then
                        log::success "✅ GDM is now active"
                    else
                        log::error "GDM failed to stay running after restart"
                        return 1
                    fi
                else
                    log::error "Failed to restart GDM"
                    log::info "You may need to run manually: sudo systemctl restart ${gdm_service}"
                    return 1
                fi
            else
                log::warning "Sudo access required to restart GDM"
                log::info "To fix manually, run: sudo systemctl restart ${gdm_service}"
                return 1
            fi
        fi
    else
        # No GDM found, check for other display managers
        log::info "GDM not found, checking for alternative display managers..."
        
        for dm in lightdm sddm xdm; do
            if (systemctl list-unit-files | grep -qE "${dm}\.service") 2>/dev/null; then
                log::info "Found ${dm} display manager"
                
                if ! systemctl is-active --quiet "${dm}" 2>/dev/null; then
                    log::warning "Display manager ${dm} is not running"
                    if sudo::can_use_sudo; then
                        log::info "Attempting to restart ${dm}..."
                        if sudo::exec_with_fallback "systemctl restart '${dm}'" 2>/dev/null; then
                            log::success "✅ Display manager ${dm} restarted successfully"
                            return 0
                        else
                            log::error "Failed to restart ${dm}"
                            return 1
                        fi
                    fi
                else
                    log::success "✅ Display manager ${dm} is running"
                fi
                return 0
            fi
        done
        
        log::info "No display manager found (headless system?)"
        return 0
    fi
    
    # Check xrdp if installed
    if (systemctl list-unit-files | grep -qE "xrdp\.service") 2>/dev/null; then
        if ! systemctl is-active --quiet xrdp 2>/dev/null; then
            log::warning "xrdp service is not running"
            if sudo::can_use_sudo; then
                sudo::exec_with_fallback "systemctl restart xrdp" 2>/dev/null || true
                log::info "Attempted to restart xrdp service"
            fi
        else
            log::success "✅ xrdp service is running"
        fi
    fi
    
    return 0
}

#######################################
# Calculate memory allocations based on system RAM
# Outputs:
#   Sets global variables for memory calculations
#######################################
remote_session::calculate_memory_allocation() {
    # Get total memory in KB and convert to MB/GB
    local mem_kb
    mem_kb=$(awk '/MemTotal:/ {print $2}' /proc/meminfo)
    local mem_mb=$((mem_kb / 1024))
    local mem_gb=$((mem_mb / 1024))
    
    # Calculate desktop reservations
    local desktop_min_mb=$((mem_mb * DESKTOP_MIN_PERCENT / 100))
    if [[ $desktop_min_mb -lt $DESKTOP_MIN_MB ]]; then
        desktop_min_mb=$DESKTOP_MIN_MB
    fi
    local desktop_low_mb=$((desktop_min_mb + DESKTOP_BUFFER_MB))
    
    # Calculate workload limits
    local workload_high_mb=$((mem_mb * WORKLOAD_HIGH_PERCENT / 100))
    local workload_max_mb=$((mem_mb * WORKLOAD_MAX_PERCENT / 100))
    
    # Calculate target swap
    local target_swap_gb=$mem_gb
    if [[ $target_swap_gb -gt $MAX_SWAP_GB ]]; then
        target_swap_gb=$MAX_SWAP_GB
    fi
    
    # Export for use by other functions
    export SYSTEM_MEM_MB=$mem_mb
    export SYSTEM_MEM_GB=$mem_gb
    export DESKTOP_MIN_CALC_MB=$desktop_min_mb
    export DESKTOP_LOW_CALC_MB=$desktop_low_mb
    export WORKLOAD_HIGH_MB=$workload_high_mb
    export WORKLOAD_MAX_MB=$workload_max_mb
    export TARGET_SWAP_GB=$target_swap_gb
    
    log::info "📊 Memory allocation calculated:"
    log::info "  System RAM: ${mem_gb}GB"
    log::info "  Desktop MemoryMin: ${desktop_min_mb}MB"
    log::info "  Desktop MemoryLow: ${desktop_low_mb}MB"
    log::info "  Workload MemoryHigh: ${WORKLOAD_HIGH_PERCENT}% (${workload_high_mb}MB)"
    log::info "  Workload MemoryMax: ${WORKLOAD_MAX_PERCENT}% (${workload_max_mb}MB)"
    log::info "  Target swap: ${target_swap_gb}GB"
}

#######################################
# Configure swap space
# Returns:
#   0 on success, 1 on failure
#######################################
remote_session::configure_swap() {
    log::info "💾 Configuring swap space..."
    
    # Calculate memory first if not already done
    if [[ -z "${TARGET_SWAP_GB:-}" ]]; then
        remote_session::calculate_memory_allocation
    fi
    
    # Check current swap
    local total_swap_kb
    total_swap_kb=$(awk '/SwapTotal:/ {print $2}' /proc/meminfo)
    local total_swap_gb=$((total_swap_kb / 1024 / 1024))
    
    log::info "Current swap: ${total_swap_gb}GB, Target: ${TARGET_SWAP_GB}GB"
    
    if [[ $total_swap_gb -ge $TARGET_SWAP_GB ]]; then
        log::success "✅ Swap space is sufficient (${total_swap_gb}GB)"
        return 0
    fi
    
    if ! sudo::can_use_sudo; then
        log::warning "Sudo access required to configure swap"
        log::info "Current swap (${total_swap_gb}GB) is less than recommended (${TARGET_SWAP_GB}GB)"
        return 0  # Don't fail, just warn
    fi
    
    log::info "Creating/resizing swapfile to ${TARGET_SWAP_GB}GB..."
    
    # Disable existing swap file if it exists
    if [[ -f "$SWAP_FILE" ]]; then
        sudo::exec_with_fallback "swapoff '$SWAP_FILE'" 2>/dev/null || true
    fi
    
    # Create new swap file
    if sudo::exec_with_fallback "fallocate -l '${TARGET_SWAP_GB}G' '$SWAP_FILE'" 2>/dev/null; then
        log::info "Swap file created with fallocate"
    else
        # Fallback to dd if fallocate fails
        log::info "Using dd to create swap file (this may take a while)..."
        if ! sudo::exec_with_fallback "dd if=/dev/zero of='$SWAP_FILE' bs=1G count='$TARGET_SWAP_GB' status=progress" 2>/dev/null; then
            log::error "Failed to create swap file"
            return 1
        fi
    fi
    
    # Set permissions and make swap
    sudo::exec_with_fallback "chmod 600 '$SWAP_FILE'"
    sudo::exec_with_fallback "mkswap '$SWAP_FILE'" >/dev/null 2>&1
    
    # Add to fstab if not already there
    if ! grep -qE "^\\s*${SWAP_FILE}\\s" /etc/fstab 2>/dev/null; then
        echo "${SWAP_FILE} none swap sw 0 0" | sudo::exec_with_fallback "tee -a /etc/fstab" >/dev/null
        log::info "Added swap file to /etc/fstab"
    fi
    
    # Enable swap
    if sudo::exec_with_fallback "swapon '$SWAP_FILE'" 2>/dev/null; then
        log::success "✅ Swap file activated"
    else
        # Try enabling all swap from fstab
        sudo::exec_with_fallback "swapon -a" 2>/dev/null || true
    fi
    
    # Verify new swap size
    total_swap_kb=$(awk '/SwapTotal:/ {print $2}' /proc/meminfo)
    total_swap_gb=$((total_swap_kb / 1024 / 1024))
    log::success "✅ Total swap now: ${total_swap_gb}GB"
    
    return 0
}

#######################################
# Protect desktop user slice with memory reservations
# Returns:
#   0 on success, 1 on failure
#######################################
remote_session::protect_desktop_slice() {
    log::info "🛡️  Protecting desktop session memory..."
    
    # Calculate memory if not already done
    if [[ -z "${DESKTOP_MIN_CALC_MB:-}" ]]; then
        remote_session::calculate_memory_allocation
    fi
    
    if ! sudo::can_use_sudo; then
        log::warning "Sudo access required to protect desktop memory"
        log::info "Desktop session may be vulnerable to OOM killer"
        return 0  # Don't fail
    fi
    
    # Create systemd slice configuration directory
    local slice_dir="/etc/systemd/system/user-${DESKTOP_UID}.slice.d"
    sudo::exec_with_fallback "mkdir -p '$slice_dir'"
    
    # Create memory protection configuration
    local config_file="${slice_dir}/50-memory-protect.conf"
    sudo::exec_with_fallback "tee '$config_file'" >/dev/null <<EOF
[Slice]
# Vrooli Remote Session Protection
# Hard reservation: cannot be reclaimed under memory pressure
MemoryMin=${DESKTOP_MIN_CALC_MB}M
# Soft preference: try to keep at least this much
MemoryLow=${DESKTOP_LOW_CALC_MB}M
# Don't kill desktop processes first
ManagedOOMPreference=omit
EOF
    
    log::success "✅ Created desktop memory protection config"
    
    # Apply to running cgroup if possible (cgroup v2)
    local cgroup_path="/sys/fs/cgroup/user.slice/user-${DESKTOP_UID}.slice"
    if [[ -d "$cgroup_path" ]]; then
        # Check if we have cgroup v2
        if [[ -f "${cgroup_path}/memory.min" ]]; then
            echo $((DESKTOP_MIN_CALC_MB * 1024 * 1024)) | sudo::exec_with_fallback "tee '${cgroup_path}/memory.min'" >/dev/null 2>&1 || true
            echo $((DESKTOP_LOW_CALC_MB * 1024 * 1024)) | sudo::exec_with_fallback "tee '${cgroup_path}/memory.low'" >/dev/null 2>&1 || true
            log::success "✅ Applied memory protection to running session"
        else
            log::info "cgroup v1 detected, changes will apply on next login"
        fi
    fi
    
    return 0
}

#######################################
# Create workload slice for containers and batch jobs
# Returns:
#   0 on success, 1 on failure
#######################################
remote_session::configure_workload_slice() {
    log::info "📦 Creating workload containment slice..."
    
    if ! sudo::can_use_sudo; then
        log::warning "Sudo access required to create workload slice"
        return 0  # Don't fail
    fi
    
    # Create workload slice configuration
    sudo::exec_with_fallback "tee /etc/systemd/system/workload.slice" >/dev/null <<EOF
[Unit]
Description=Slice for batch jobs and AI workloads
Documentation=https://github.com/Vrooli/Vrooli

[Slice]
# Start throttling at high watermark
MemoryHigh=${WORKLOAD_HIGH_PERCENT}%
# Hard cap at maximum
MemoryMax=${WORKLOAD_MAX_PERCENT}%
# Prefer killing workload processes under pressure
ManagedOOMMemoryPressure=kill
ManagedOOMMemoryPressureLimit=60%
# Lower OOM score preference (kill these first)
ManagedOOMPreference=avoid
EOF
    
    log::success "✅ Created workload.slice configuration"
    
    # Reload systemd to pick up new slice
    sudo::exec_with_fallback "systemctl daemon-reload"
    
    return 0
}

#######################################
# Update Docker configuration to use workload slice
# Returns:
#   0 on success, 1 on failure
#######################################
remote_session::update_docker_config() {
    log::info "🐳 Configuring Docker to use workload slice..."
    
    # Check if Docker is installed and running
    if ! (systemctl list-unit-files | grep -qE "docker\.service") 2>/dev/null; then
        log::info "Docker not installed, skipping configuration"
        return 0
    fi
    
    if ! sudo::can_use_sudo; then
        log::warning "Sudo access required to configure Docker"
        return 0  # Don't fail
    fi
    
    # Ensure Docker config directory exists
    sudo::exec_with_fallback "mkdir -p /etc/docker"
    
    local docker_config="/etc/docker/daemon.json"
    local temp_config="/tmp/docker-daemon-$$.json"
    
    # Create or update Docker configuration
    if [[ -f "$docker_config" ]]; then
        # Backup existing config
        sudo::exec_with_fallback "cp '$docker_config' '${docker_config}.backup'"
        
        # Try to merge with existing config
        if command -v jq >/dev/null 2>&1; then
            # Use jq for proper JSON merging
            sudo::exec_with_fallback "cat '$docker_config'" | jq '. + {
                "exec-opts": ((.["exec-opts"] // []) + ["native.cgroupdriver=systemd"] | unique),
                "default-cgroup-parent": "workload.slice"
            }' > "$temp_config" 2>/dev/null || {
                log::warning "Failed to merge Docker config with jq, using simple approach"
                # Fallback to simple replacement
                echo '{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "default-cgroup-parent": "workload.slice"
}' > "$temp_config"
            }
        else
            # Simple approach without jq
            log::info "jq not found, using simple Docker config"
            echo '{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "default-cgroup-parent": "workload.slice"
}' > "$temp_config"
        fi
    else
        # Create new config
        echo '{
  "exec-opts": ["native.cgroupdriver=systemd"],
  "default-cgroup-parent": "workload.slice"
}' > "$temp_config"
    fi
    
    # Validate JSON before applying
    if command -v python3 >/dev/null 2>&1; then
        if python3 -m json.tool < "$temp_config" >/dev/null 2>&1; then
            sudo::exec_with_fallback "mv '$temp_config' '$docker_config'"
            log::success "✅ Docker configuration updated"
        else
            log::error "Invalid JSON in Docker config, not applying"
            rm -f "$temp_config"
            return 1
        fi
    else
        # No validation available, just apply
        sudo::exec_with_fallback "mv '$temp_config' '$docker_config'"
        log::success "✅ Docker configuration updated"
    fi
    
    # Reload and restart Docker
    sudo::exec_with_fallback "systemctl daemon-reload"
    
    # Only restart if Docker is running
    if systemctl is-active --quiet docker 2>/dev/null; then
        log::info "Restarting Docker to apply changes..."
        if sudo::exec_with_fallback "systemctl restart docker" 2>/dev/null; then
            log::success "✅ Docker restarted with workload slice"
        else
            log::warning "Failed to restart Docker, changes will apply on next restart"
        fi
    else
        log::info "Docker not running, changes will apply when started"
    fi
    
    return 0
}

#######################################
# Main configuration function
# Arguments:
#   $1 - Operation (configure|cleanup|check)
# Returns:
#   0 on success, 1 on failure
#######################################
remote_session::main() {
    local operation="${1:-configure}"
    
    case "$operation" in
        configure)
            log::header "🖥️  Configuring remote session protections"
            
            # Check if this is relevant for the system
            if ! remote_session::is_remote_desktop_installed; then
                log::info "No GUI desktop or remote access detected, skipping protections"
                log::info "This appears to be a headless server without desktop environment"
                return 0
            fi
            
            log::success "Desktop environment detected, applying memory protections..."
            
            # Calculate memory allocations
            remote_session::calculate_memory_allocation
            
            # Apply all protections
            local all_success=true
            
            # Check/fix GDM3
            remote_session::check_gdm3_status || all_success=false
            
            # Configure swap
            remote_session::configure_swap || all_success=false
            
            # Protect desktop memory
            remote_session::protect_desktop_slice || all_success=false
            
            # Create workload slice
            remote_session::configure_workload_slice || all_success=false
            
            # Configure Docker if present
            remote_session::update_docker_config || true  # Don't fail if Docker config fails
            
            # Reload systemd to apply all changes
            if sudo::can_use_sudo; then
                sudo::exec_with_fallback "systemctl daemon-reload"
            fi
            
            if [[ "$all_success" == "true" ]]; then
                log::success "✅ Remote session protections configured successfully"
                log::info "Changes are active immediately for new processes"
                log::info "Full protection applies after next login"
            else
                log::warning "⚠️  Some protections could not be applied"
                log::info "System will still function but may be vulnerable to memory pressure"
            fi
            ;;
            
        check)
            log::header "🔍 Checking remote session status"
            
            if ! remote_session::is_remote_desktop_installed; then
                log::info "No remote desktop installed"
                return 0
            fi
            
            # Just check GDM3 status without fixing
            if systemctl is-active --quiet gdm3 2>/dev/null || systemctl is-active --quiet gdm 2>/dev/null; then
                log::success "✅ Display manager is running"
            else
                log::warning "⚠️  Display manager is not running"
                log::info "Run with 'configure' to fix: $0 configure"
                return 1
            fi
            ;;
            
        cleanup)
            log::header "🧹 Removing remote session protections"
            
            if ! sudo::can_use_sudo; then
                log::error "Sudo access required for cleanup"
                return 1
            fi
            
            # Remove desktop slice protection
            local slice_dir="/etc/systemd/system/user-${DESKTOP_UID}.slice.d"
            if [[ -d "$slice_dir" ]]; then
                sudo::exec_with_fallback "bash -c 'source ${var_LIB_SYSTEM_DIR}/trash.sh && trash::safe_remove \"$slice_dir\" --no-confirm'"
                log::info "Removed desktop memory protection"
            fi
            
            # Remove workload slice
            if [[ -f "/etc/systemd/system/workload.slice" ]]; then
                sudo::exec_with_fallback "rm -f '/etc/systemd/system/workload.slice'"
                log::info "Removed workload slice"
            fi
            
            # Note: Not removing swap or Docker config as these may be wanted for other reasons
            
            sudo::exec_with_fallback "systemctl daemon-reload"
            log::success "✅ Remote session protections removed"
            ;;
            
        *)
            log::error "Unknown operation: $operation"
            log::info "Usage: $0 {configure|check|cleanup}"
            return 1
            ;;
    esac
    
    return 0
}

# Run main function if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    remote_session::main "$@"
fi