#!/usr/bin/env bash
# Agent S2 Security Installation Script
# Installs AppArmor profiles and sets up security monitoring

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
AGENT_S2_SECURITY_DIR="${APP_ROOT}/resources/agent-s2/security"

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source common utilities using var_ variables
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_DIR}/common.sh"

# Source trash system for safe removal
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

SECURITY_DIR="${AGENT_S2_SECURITY_DIR}"
APPARMOR_PROFILE_DIR="/etc/apparmor.d"
LOG_DIR="/var/log/agent-s2-audit"

# Use log functions from common.sh instead of custom print functions
print_status() { log::info "$@"; }
print_success() { log::success "$@"; }
print_warning() { log::warning "$@"; }
print_error() { log::error "$@"; }

#######################################
# Check if running as root
#######################################
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root for security component installation"
        print_status "Run: sudo $0"
        exit 1
    fi
}

#######################################
# Check system compatibility
#######################################
check_system() {
    print_status "Checking system compatibility..."
    
    # Check if AppArmor is available
    if ! command -v aa-status >/dev/null 2>&1; then
        print_warning "AppArmor not found. Installing AppArmor utilities..."
        
        if command -v apt-get >/dev/null 2>&1; then
            sudo::exec_with_fallback "apt-get update -qq"
            sudo::exec_with_fallback "apt-get install -y apparmor-utils apparmor-profiles"
        elif command -v yum >/dev/null 2>&1; then
            sudo::exec_with_fallback "yum install -y apparmor-utils apparmor-profiles"
        elif command -v dnf >/dev/null 2>&1; then
            sudo::exec_with_fallback "dnf install -y apparmor-utils apparmor-profiles"
        else
            print_error "Could not install AppArmor. Please install manually."
            return 1
        fi
    fi
    
    # Check if AppArmor kernel module is loaded
    if ! lsmod | grep -q apparmor; then
        print_warning "AppArmor kernel module not loaded"
        print_status "Attempting to load AppArmor module..."
        
        if sudo::exec_with_fallback "modprobe apparmor" 2>/dev/null; then
            print_success "AppArmor module loaded"
        else
            print_warning "Could not load AppArmor module. Security profiles may not work."
        fi
    fi
    
    # Check AppArmor status
    if aa-status >/dev/null 2>&1; then
        print_success "AppArmor is active and functional"
    else
        print_warning "AppArmor may not be fully functional"
    fi
}

#######################################
# Install AppArmor profile
#######################################
install_apparmor_profile() {
    print_status "Installing AppArmor profile for Agent S2 host mode..."
    
    local profile_name="docker-agent-s2-host"
    local profile_source="${SECURITY_DIR}/apparmor/${profile_name}"
    local profile_target="${APPARMOR_PROFILE_DIR}/${profile_name}"
    
    if [[ ! -f "$profile_source" ]]; then
        print_error "AppArmor profile not found: $profile_source"
        return 1
    fi
    
    # Copy profile to system directory
    cp "$profile_source" "$profile_target"
    chmod 644 "$profile_target"
    
    print_success "AppArmor profile copied to $profile_target"
    
    # Load the profile
    print_status "Loading AppArmor profile..."
    if aa-complain "$profile_target" 2>/dev/null; then
        print_success "AppArmor profile loaded in complain mode"
        print_status "Profile will log violations but allow them (recommended for testing)"
    else
        print_warning "Failed to load AppArmor profile in complain mode"
    fi
    
    # Optionally enforce the profile (commented out for safety)
    # print_status "Enforcing AppArmor profile..."
    # if aa-enforce "$profile_target" 2>/dev/null; then
    #     print_success "AppArmor profile is now enforced"
    # else
    #     print_warning "Failed to enforce AppArmor profile"
    # fi
    
    # Verify profile is loaded
    if aa-status | grep -q "$profile_name"; then
        print_success "AppArmor profile verification successful"
    else
        print_warning "AppArmor profile may not be properly loaded"
    fi
}

#######################################
# Create audit log directory
#######################################
setup_audit_logging() {
    print_status "Setting up audit logging..."
    
    # Create audit log directory
    mkdir -p "$LOG_DIR"
    chmod 750 "$LOG_DIR"
    
    # Set ownership (if running in Docker context, this may not work as expected)
    if id agents2 >/dev/null 2>&1; then
        chown agents2:agents2 "$LOG_DIR"
        print_success "Audit log directory ownership set to agents2"
    else
        print_warning "User 'agents2' not found. Audit log directory owned by root."
    fi
    
    print_success "Audit logging directory created: $LOG_DIR"
    
    # Create logrotate configuration
    setup_log_rotation
}

#######################################
# Setup log rotation for audit logs
#######################################
setup_log_rotation() {
    print_status "Setting up log rotation for audit logs..."
    
    local logrotate_config="/etc/logrotate.d/agent-s2-audit"
    
    cat > "$logrotate_config" << 'EOF'
/var/log/agent-s2-audit/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 agents2 agents2
    postrotate
        # Signal any processes that might need to reopen log files
        pkill -USR1 -f "agent-s2" || true
    endscript
}
EOF
    
    chmod 644 "$logrotate_config"
    print_success "Log rotation configured for audit logs"
}

#######################################
# Setup security monitoring service (systemd)
#######################################
setup_monitoring_service() {
    print_status "Setting up security monitoring service..."
    
    local service_file="/etc/systemd/system/agent-s2-security-monitor.service"
    
    cat > "$service_file" << 'EOF'
[Unit]
Description=Agent S2 Security Monitor
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=root
ExecStart=/usr/bin/docker logs -f agent-s2 | grep -E "(SECURITY|AUDIT)" >> /var/log/agent-s2-audit/security-monitor.log
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
    
    chmod 644 "$service_file"
    
    # Reload systemd and enable service
    systemctl daemon-reload
    systemctl enable agent-s2-security-monitor.service
    
    print_success "Security monitoring service configured"
    print_status "Start with: systemctl start agent-s2-security-monitor.service"
}

#######################################
# Create security configuration file
#######################################
create_security_config() {
    print_status "Creating security configuration file..."
    
    local config_file="/etc/agent-s2-security.conf"
    
    cat > "$config_file" << 'EOF'
# Agent S2 Security Configuration
# This file contains security settings for Agent S2 dual-mode operation

# AppArmor Configuration
APPARMOR_PROFILE_HOST="docker-agent-s2-host"
APPARMOR_PROFILE_SANDBOX="docker-default"

# Audit Logging
AUDIT_LOG_DIR="/var/log/agent-s2-audit"
AUDIT_LOG_RETENTION_DAYS="30"
AUDIT_LOG_MAX_SIZE_MB="100"

# Security Monitoring
SECURITY_MONITOR_ENABLED="true"
THREAT_DETECTION_LEVEL="medium"  # low, medium, high
ALERT_THRESHOLD_SCORE="40"       # 0-100

# Host Mode Security Constraints
HOST_MODE_MAX_MOUNT_SIZE_GB="10"
HOST_MODE_FORBIDDEN_PATHS="/etc:/var:/usr:/boot:/sys:/proc"
HOST_MODE_ALLOWED_NETWORKS="public"  # public, private, all

# Network Security
NETWORK_ACCESS_LOGGING="true"
SUSPICIOUS_NETWORK_DETECTION="true"

# File System Security
FILE_ACCESS_LOGGING="true"
SUSPICIOUS_FILE_PATTERNS="*.key:*.pem:password:secret:token"

# Process Security
PROCESS_MONITORING="true"
SUSPICIOUS_COMMAND_DETECTION="true"
EOF
    
    chmod 600 "$config_file"
    print_success "Security configuration created: $config_file"
}

#######################################
# Verify security installation
#######################################
verify_installation() {
    print_status "Verifying security installation..."
    
    local errors=0
    
    # Check AppArmor profile
    if aa-status | grep -q "docker-agent-s2-host"; then
        print_success "✓ AppArmor profile loaded"
    else
        print_error "✗ AppArmor profile not loaded"
        ((errors++))
    fi
    
    # Check audit directory
    if [[ -d "$LOG_DIR" ]] && [[ -w "$LOG_DIR" ]]; then
        print_success "✓ Audit log directory accessible"
    else
        print_error "✗ Audit log directory not accessible"
        ((errors++))
    fi
    
    # Check logrotate configuration
    if [[ -f "/etc/logrotate.d/agent-s2-audit" ]]; then
        print_success "✓ Log rotation configured"
    else
        print_warning "⚠ Log rotation not configured"
    fi
    
    # Check security configuration
    if [[ -f "/etc/agent-s2-security.conf" ]]; then
        print_success "✓ Security configuration file present"
    else
        print_error "✗ Security configuration file missing"
        ((errors++))
    fi
    
    if [[ $errors -eq 0 ]]; then
        print_success "Security installation verification passed"
        return 0
    else
        print_error "Security installation verification failed with $errors errors"
        return 1
    fi
}

#######################################
# Display security information
#######################################
show_security_info() {
    echo
    print_status "Agent S2 Security Installation Complete"
    echo
    echo "Security Features Installed:"
    echo "  • AppArmor profile for host mode security"
    echo "  • Audit logging system with rotation"
    echo "  • Security monitoring and threat detection"
    echo "  • File system and network access controls"
    echo
    echo "Configuration Files:"
    echo "  • AppArmor Profile: ${APPARMOR_PROFILE_DIR}/docker-agent-s2-host"
    echo "  • Audit Logs: $LOG_DIR"
    echo "  • Security Config: /etc/agent-s2-security.conf"
    echo "  • Log Rotation: /etc/logrotate.d/agent-s2-audit"
    echo
    echo "To use host mode with security:"
    echo "  1. Start Agent S2 with host mode enabled"
    echo "  2. Docker will automatically use the AppArmor profile"
    echo "  3. All actions will be audited to $LOG_DIR"
    echo "  4. Security violations will be logged and alerted"
    echo
    echo "AppArmor Profile Status:"
    aa-status | grep -E "(docker-agent-s2|agent-s2)" || echo "  No Agent S2 profiles currently active"
    echo
    print_warning "Note: AppArmor profile is in 'complain' mode for initial testing"
    print_status "Switch to 'enforce' mode with: sudo aa-enforce ${APPARMOR_PROFILE_DIR}/docker-agent-s2-host"
}

#######################################
# Main installation function
#######################################
main() {
    print_status "Starting Agent S2 Security Installation..."
    
    # Perform installation steps
    check_root
    check_system
    install_apparmor_profile
    setup_audit_logging
    setup_monitoring_service
    create_security_config
    
    # Verify installation
    if verify_installation; then
        show_security_info
        print_success "Agent S2 security installation completed successfully!"
    else
        print_error "Agent S2 security installation completed with errors!"
        exit 1
    fi
}

# Parse command line arguments
case "${1:-install}" in
    "install")
        main
        ;;
    "verify")
        check_root
        verify_installation
        ;;
    "uninstall")
        print_status "Uninstalling Agent S2 security components..."
        trash::safe_remove "${APPARMOR_PROFILE_DIR}/docker-agent-s2-host" --production
        trash::safe_remove "$LOG_DIR" --production
        trash::safe_remove "/etc/logrotate.d/agent-s2-audit" --production
        trash::safe_remove "/etc/systemd/system/agent-s2-security-monitor.service" --production
        trash::safe_remove "/etc/agent-s2-security.conf" --production
        systemctl daemon-reload
        print_success "Agent S2 security components uninstalled"
        ;;
    "status")
        print_status "Agent S2 Security Status:"
        verify_installation
        ;;
    *)
        echo "Usage: $0 {install|verify|uninstall|status}"
        echo
        echo "Commands:"
        echo "  install   - Install security components (default)"
        echo "  verify    - Verify security installation"
        echo "  uninstall - Remove security components"
        echo "  status    - Show security status"
        exit 1
        ;;
esac