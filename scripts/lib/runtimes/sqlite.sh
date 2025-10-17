#!/usr/bin/env bash
# Install SQLite3 via platform-appropriate method
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
RUNTIME_DIR="${APP_ROOT}/scripts/lib/runtimes"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "$var_LIB_SYSTEM_DIR/system_commands.sh"
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/retry.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh"

# Clean interface for setup.sh
sqlite::ensure_installed() {
    sqlite::check_and_install "$@"
}

# Default SQLite version if building from source
DEFAULT_SQLITE_VERSION="3450000"  # SQLite 3.45.0 (format: XXYYZZ00)

#######################################
# Get platform name with SQLite-specific conventions
# Returns: Platform name (linux, mac, windows, unknown)
#######################################
sqlite::get_platform() {
    system::detect_platform
}

#######################################
# Verify SQLite installation
# Returns: 0 if installed, 1 if not
#######################################
sqlite::verify_installation() {
    if system::is_command sqlite3; then
        local version
        version=$(sqlite3 --version 2>/dev/null | awk '{print $1}' || echo "unknown")
        log::success "SQLite installed: $version"
        return 0
    else
        log::error "SQLite installation verification failed"
        return 1
    fi
}

#######################################
# Install SQLite from source
# Arguments:
#   version - SQLite version code (default: latest)
# Returns: 0 on success, 1 on failure
#######################################
sqlite::install_from_source() {
    local version="${1:-$DEFAULT_SQLITE_VERSION}"
    
    log::info "Installing SQLite from source (version: $version)..."
    
    # Check if we can run privileged operations
    if ! flow::can_run_sudo "SQLite source installation"; then
        log::error "Source installation requires root privileges"
        return 1
    fi
    
    # Install build dependencies
    local pm
    pm=$(system::detect_pm)
    
    case "$pm" in
        apt-get)
            system::install_pkg build-essential
            system::install_pkg libreadline-dev
            ;;
        yum|dnf)
            system::install_pkg gcc
            system::install_pkg make
            system::install_pkg readline-devel
            ;;
        brew)
            # Homebrew usually has these already
            :
            ;;
        *)
            log::warning "Cannot install build dependencies for package manager: $pm"
            ;;
    esac
    
    # Download SQLite source with retry
    local sqlite_url="https://www.sqlite.org/2024/sqlite-autoconf-${version}.tar.gz"
    local sqlite_tarball="/tmp/sqlite-${version}.tar.gz"
    local sqlite_dir="/tmp/sqlite-autoconf-${version}"
    
    log::info "Downloading SQLite source from: $sqlite_url"
    if ! retry::download "$sqlite_url" > "$sqlite_tarball"; then
        log::error "Failed to download SQLite source"
        return 1
    fi
    
    # Extract and build
    cd /tmp || return 1
    tar -xzf "$sqlite_tarball" || {
        log::error "Failed to extract SQLite source"
        rm -f "$sqlite_tarball"
        return 1
    }
    
    cd "$sqlite_dir" || {
        log::error "Failed to enter SQLite source directory"
        rm -rf "$sqlite_dir" "$sqlite_tarball"
        return 1
    }
    
    # Configure, compile, and install
    log::info "Configuring SQLite build..."
    ./configure --prefix=/usr/local || {
        log::error "Failed to configure SQLite"
        cd /tmp && rm -rf "$sqlite_dir" "$sqlite_tarball"
        return 1
    }
    
    log::info "Compiling SQLite (this may take a few minutes)..."
    make -j"$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 1)" || {
        log::error "Failed to compile SQLite"
        cd /tmp && rm -rf "$sqlite_dir" "$sqlite_tarball"
        return 1
    }
    
    log::info "Installing SQLite..."
    flow::maybe_run_sudo make install || {
        log::error "Failed to install SQLite"
        cd /tmp && rm -rf "$sqlite_dir" "$sqlite_tarball"
        return 1
    }
    
    # Clean up
    cd /tmp
    rm -rf "$sqlite_dir" "$sqlite_tarball"
    
    # Update library cache on Linux
    if [[ "$(sqlite::get_platform)" == "linux" ]]; then
        flow::maybe_run_sudo ldconfig
    fi
    
    # Create symlink in /usr/bin for compatibility
    if [[ -f /usr/local/bin/sqlite3 ]]; then
        flow::maybe_run_sudo ln -sf /usr/local/bin/sqlite3 /usr/bin/sqlite3 || true
    fi
    
    log::success "SQLite installed from source successfully"
    return 0
}

#######################################
# Install SQLite via package manager
# Returns: 0 on success, 1 on failure
#######################################
sqlite::install_via_package_manager() {
    local platform
    platform=$(sqlite::get_platform)
    
    log::info "Installing SQLite via package manager..."
    
    case "$platform" in
        linux)
            local pm
            pm=$(system::detect_pm)
            
            case "$pm" in
                apt-get)
                    system::update
                    system::install_pkg sqlite3
                    system::install_pkg libsqlite3-dev  # Development headers
                    ;;
                yum)
                    system::install_pkg sqlite
                    system::install_pkg sqlite-devel  # Development headers
                    ;;
                dnf)
                    system::install_pkg sqlite
                    system::install_pkg sqlite-devel  # Development headers
                    ;;
                pacman)
                    system::install_pkg sqlite
                    ;;
                apk)
                    system::install_pkg sqlite
                    system::install_pkg sqlite-dev  # Development headers
                    ;;
                *)
                    log::warning "Package manager $pm not supported, trying source installation"
                    sqlite::install_from_source
                    return $?
                    ;;
            esac
            ;;
        darwin|mac)
            if system::is_command brew; then
                brew install sqlite3
                
                # macOS often needs path adjustment for sqlite3
                local brew_prefix
                brew_prefix=$(brew --prefix)
                if [[ -f "${brew_prefix}/opt/sqlite/bin/sqlite3" ]]; then
                    log::info "SQLite installed via Homebrew"
                    log::info "You may need to add to PATH: ${brew_prefix}/opt/sqlite/bin"
                    
                    # Try to create symlink for immediate availability
                    flow::maybe_run_sudo ln -sf "${brew_prefix}/opt/sqlite/bin/sqlite3" /usr/local/bin/sqlite3 || true
                fi
            else
                log::error "Homebrew not found. Please install Homebrew first."
                return 1
            fi
            ;;
        windows)
            log::error "Windows installation not supported by this script"
            log::info "Please install SQLite manually from https://www.sqlite.org/download.html"
            return 1
            ;;
        *)
            log::error "Unsupported platform: $platform"
            return 1
            ;;
    esac
    
    # Verify installation
    sqlite::verify_installation
}

#######################################
# Check and install SQLite if needed
# Arguments:
#   --force - Force reinstallation even if already installed
#   --source - Install from source instead of package manager
#   --version - Version to install (for source installation)
# Returns: 0 on success, 1 on failure
#######################################
sqlite::check_and_install() {
    local force=false
    local from_source=false
    local version="$DEFAULT_SQLITE_VERSION"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force)
                force=true
                shift
                ;;
            --source)
                from_source=true
                shift
                ;;
            --version)
                version="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Check if already installed
    if [[ "$force" != "true" ]] && system::is_command sqlite3; then
        local current_version
        current_version=$(sqlite3 --version 2>/dev/null | awk '{print $1}' || echo "unknown")
        log::info "SQLite is already installed: $current_version"
        
        # If running with sudo on Linux, check if SQLite is available to the actual user
        if sudo::is_running_as_sudo && [[ "$(sqlite::get_platform)" == "linux" ]]; then
            local actual_user
            actual_user=$(sudo::get_actual_user)
            
            if sudo::exec_as_actual_user 'command -v sqlite3 >/dev/null 2>&1'; then
                log::info "SQLite is available to user: ${actual_user}"
                return 0
            else
                log::info "SQLite is NOT available to ${actual_user}, installing system-wide..."
                force=true
            fi
        else
            return 0
        fi
    fi
    
    # Install SQLite
    if [[ "$from_source" == "true" ]]; then
        sqlite::install_from_source "$version"
    else
        sqlite::install_via_package_manager
    fi
    
    local result=$?
    
    # Final verification
    if [[ $result -eq 0 ]]; then
        sqlite::verify_installation
        result=$?
    fi
    
    return $result
}

#######################################
# Get SQLite version
# Returns: Version string or "not installed"
#######################################
sqlite::get_version() {
    if system::is_command sqlite3; then
        sqlite3 --version 2>/dev/null | awk '{print $1}' || echo "unknown"
    else
        echo "not installed"
    fi
}

#######################################
# Test SQLite database integrity
# Arguments:
#   db_path - Path to SQLite database file
# Returns: 0 if OK, 1 if corrupted
#######################################
sqlite::check_integrity() {
    local db_path="$1"
    
    if [[ ! -f "$db_path" ]]; then
        log::error "Database file not found: $db_path"
        return 1
    fi
    
    if ! system::is_command sqlite3; then
        log::error "sqlite3 command not found"
        return 1
    fi
    
    local result
    result=$(sqlite3 "$db_path" "PRAGMA integrity_check;" 2>/dev/null || echo "error")
    
    if [[ "$result" == "ok" ]]; then
        log::success "Database integrity check passed"
        return 0
    else
        log::warn "Database integrity check failed: $result"
        return 1
    fi
}

# If this script is run directly, invoke its main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    sqlite::check_and_install "$@"
fi