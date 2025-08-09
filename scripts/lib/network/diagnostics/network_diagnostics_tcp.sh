#!/usr/bin/env bash
# TCP optimization functions for network diagnostics
# Handles TCP-specific fixes like TSO, MTU, ECN, etc.
set -eo pipefail

# Source var.sh with relative path first
LIB_NETWORK_DIAGNOSTICS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${LIB_NETWORK_DIAGNOSTICS_DIR}/../../utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"

# Function to make TCP offload fix permanent
network_diagnostics_tcp::make_tso_permanent() {
    local interface="$1"
    log::info "Making TCP offload fix permanent for $interface..."
    
    # Method 1: systemd-networkd (most reliable)
    if systemctl is-enabled systemd-networkd >/dev/null 2>&1; then
        local networkd_file="/etc/systemd/network/99-tso-fix.network"
        log::info "  → Using systemd-networkd configuration"
        
        if sudo tee "$networkd_file" >/dev/null 2>&1 <<EOF
[Match]
Name=$interface

[Link]
GenericSegmentationOffload=false
TCPSegmentationOffload=false
EOF
        then
            log::success "  ✓ Created $networkd_file"
            sudo systemctl restart systemd-networkd 2>/dev/null || true
            log::success "  ✓ TCP offload fix is now permanent via systemd-networkd"
            return 0
        fi
    fi
    
    # Method 2: NetworkManager dispatcher (if NetworkManager is active)
    if systemctl is-active NetworkManager >/dev/null 2>&1; then
        local dispatcher_file="/etc/NetworkManager/dispatcher.d/99-disable-tso"
        log::info "  → Using NetworkManager dispatcher"
        
        if sudo tee "$dispatcher_file" >/dev/null 2>&1 <<EOF
#!/bin/bash
# Disable TCP/GSO offload to fix TLS issues
if [ "\$1" = "$interface" ] && [ "\$2" = "up" ]; then
    ethtool -K $interface tso off gso off 2>/dev/null || true
fi
EOF
        then
            sudo chmod +x "$dispatcher_file" 2>/dev/null
            log::success "  ✓ Created NetworkManager dispatcher script"
            log::success "  ✓ TCP offload fix is now permanent via NetworkManager"
            return 0
        fi
    fi
    
    # Method 3: Create a simple systemd service
    local service_file="/etc/systemd/system/fix-tso.service"
    log::info "  → Creating systemd service as backup"
    
    if sudo tee "$service_file" >/dev/null 2>&1 <<EOF
[Unit]
Description=Disable TCP Segmentation Offload to fix TLS issues
After=network.target

[Service]
Type=oneshot
ExecStart=/sbin/ethtool -K $interface tso off gso off
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
    then
        sudo systemctl daemon-reload 2>/dev/null
        sudo systemctl enable fix-tso.service 2>/dev/null
        log::success "  ✓ Created and enabled systemd service"
        return 0
    fi
    
    log::warning "  ⚠️  Could not make fix permanent (requires manual setup)"
    return 1
}

# Check TCP settings
network_diagnostics_tcp::check_tcp_settings() {
    log::info "Checking TCP settings..."
    
    # Check TCP congestion control
    local congestion_control
    congestion_control=$(sysctl net.ipv4.tcp_congestion_control 2>/dev/null | awk '{print $3}')
    log::info "  TCP congestion control: ${congestion_control:-unknown}"
    
    # Check TCP keepalive settings
    local keepalive_time
    keepalive_time=$(sysctl net.ipv4.tcp_keepalive_time 2>/dev/null | awk '{print $3}')
    log::info "  TCP keepalive time: ${keepalive_time:-unknown} seconds"
    
    # Check TCP window scaling
    local window_scaling
    window_scaling=$(sysctl net.ipv4.tcp_window_scaling 2>/dev/null | awk '{print $3}')
    log::info "  TCP window scaling: ${window_scaling:-unknown}"
    
    # Check TCP timestamps
    local timestamps
    timestamps=$(sysctl net.ipv4.tcp_timestamps 2>/dev/null | awk '{print $3}')
    log::info "  TCP timestamps: ${timestamps:-unknown}"
    
    # Check ECN
    local ecn
    ecn=$(sysctl net.ipv4.tcp_ecn 2>/dev/null | awk '{print $3}')
    log::info "  TCP ECN: ${ecn:-unknown}"
    
    return 0
}

# Fix ECN issues
network_diagnostics_tcp::fix_ecn() {
    log::info "Attempting to fix ECN issues..."
    
    if flow::can_run_sudo "disable ECN"; then
        # Disable ECN
        if sudo sysctl -w net.ipv4.tcp_ecn=0 >/dev/null 2>&1; then
            log::success "  ✓ Disabled TCP ECN"
            
            # Make permanent
            if sudo tee -a /etc/sysctl.conf >/dev/null 2>&1 <<< "net.ipv4.tcp_ecn=0"; then
                log::success "  ✓ Made ECN change permanent"
            fi
            return 0
        else
            log::warning "  ⚠️  Could not disable ECN"
            return 1
        fi
    else
        log::info "  → Skipping ECN fix (requires sudo)"
        return 1
    fi
}

# Fix MTU size
network_diagnostics_tcp::fix_mtu_size() {
    local interface="$1"
    local new_mtu="${2:-1400}"
    
    log::info "Attempting to reduce MTU to $new_mtu on $interface..."
    
    if flow::can_run_sudo "change MTU"; then
        if sudo ip link set dev "$interface" mtu "$new_mtu" 2>/dev/null; then
            log::success "  ✓ Set MTU to $new_mtu on $interface"
            
            # Make permanent (varies by distro)
            if [[ -f /etc/network/interfaces ]]; then
                # Debian/Ubuntu
                if ! grep -q "mtu $new_mtu" /etc/network/interfaces 2>/dev/null; then
                    echo "  mtu $new_mtu" | sudo tee -a /etc/network/interfaces >/dev/null
                    log::success "  ✓ Made MTU change permanent in /etc/network/interfaces"
                fi
            elif command -v nmcli >/dev/null 2>&1; then
                # NetworkManager
                local connection
                connection=$(nmcli -t -f NAME,DEVICE con show --active | grep ":$interface" | cut -d: -f1)
                if [[ -n "$connection" ]]; then
                    sudo nmcli con mod "$connection" ethernet.mtu "$new_mtu" 2>/dev/null
                    log::success "  ✓ Made MTU change permanent in NetworkManager"
                fi
            fi
            return 0
        else
            log::warning "  ⚠️  Could not change MTU"
            return 1
        fi
    else
        log::info "  → Skipping MTU fix (requires sudo)"
        return 1
    fi
}

# Fix PMTU probing
network_diagnostics_tcp::fix_pmtu_probing() {
    log::info "Attempting to fix Path MTU discovery issues..."
    
    if flow::can_run_sudo "fix PMTU"; then
        # Enable PMTU discovery
        if sudo sysctl -w net.ipv4.tcp_mtu_probing=1 >/dev/null 2>&1; then
            log::success "  ✓ Enabled TCP MTU probing"
        fi
        
        # Set base MSS to lower value for better compatibility
        if sudo sysctl -w net.ipv4.tcp_base_mss=1024 >/dev/null 2>&1; then
            log::success "  ✓ Set TCP base MSS to 1024"
        fi
        
        # Make changes permanent
        if sudo tee -a /etc/sysctl.conf >/dev/null 2>&1 <<EOF
net.ipv4.tcp_mtu_probing=1
net.ipv4.tcp_base_mss=1024
EOF
        then
            log::success "  ✓ Made PMTU changes permanent"
        fi
        return 0
    else
        log::info "  → Skipping PMTU fix (requires sudo)"
        return 1
    fi
}

# Test MTU discovery
network_diagnostics_tcp::test_mtu_discovery() {
    local target="${1:-google.com}"
    log::info "Testing Path MTU discovery to $target..."
    
    if command -v ping >/dev/null 2>&1; then
        # Test with Don't Fragment flag
        local mtu_sizes=(1500 1472 1400 1280 1024)
        local working_mtu=""
        
        for mtu in "${mtu_sizes[@]}"; do
            if ping -M "do" -s $((mtu - 28)) -c 1 -W 2 "$target" >/dev/null 2>&1; then
                working_mtu=$mtu
                break
            fi
        done
        
        if [[ -n "$working_mtu" ]]; then
            log::info "  → Maximum working MTU to $target: $working_mtu bytes"
            echo "$working_mtu"
            return 0
        else
            log::warning "  ⚠️  Could not determine working MTU (all sizes failed)"
            return 1
        fi
    else
        log::warning "  ⚠️  ping command not available"
        return 1
    fi
}

# Check PMTU status
network_diagnostics_tcp::check_pmtu_status() {
    log::info "Checking Path MTU discovery status..."
    
    local mtu_probing
    mtu_probing=$(sysctl net.ipv4.tcp_mtu_probing 2>/dev/null | awk '{print $3}')
    case "$mtu_probing" in
        0) log::info "  PMTU probing: Disabled" ;;
        1) log::info "  PMTU probing: Enabled (disabled by default)" ;;
        2) log::info "  PMTU probing: Always enabled" ;;
        *) log::info "  PMTU probing: Unknown status" ;;
    esac
    
    local base_mss
    base_mss=$(sysctl net.ipv4.tcp_base_mss 2>/dev/null | awk '{print $3}')
    log::info "  TCP base MSS: ${base_mss:-unknown}"
    
    return 0
}

# Export functions for external use
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # Script was sourced, export functions
    export -f network_diagnostics_tcp::make_tso_permanent
    export -f network_diagnostics_tcp::check_tcp_settings
    export -f network_diagnostics_tcp::fix_ecn
    export -f network_diagnostics_tcp::fix_mtu_size
    # Commenting out due to bash export/import issues with heredocs
    # export -f network_diagnostics_tcp::fix_pmtu_probing
    export -f network_diagnostics_tcp::test_mtu_discovery
    export -f network_diagnostics_tcp::check_pmtu_status
fi