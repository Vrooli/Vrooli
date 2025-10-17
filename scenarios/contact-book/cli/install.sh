#!/bin/bash

# Contact Book CLI Installation Script
# Installs the contact-book CLI globally for use by other scenarios

set -e

# Configuration
CLI_NAME="contact-book"
CLI_SOURCE_PATH="$(dirname "$0")/${CLI_NAME}"
VROOLI_BIN_DIR="${HOME}/.vrooli/bin"
CLI_INSTALL_PATH="${VROOLI_BIN_DIR}/${CLI_NAME}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Create .vrooli/bin directory if it doesn't exist
if [[ ! -d "${VROOLI_BIN_DIR}" ]]; then
    log_info "Creating Vrooli bin directory: ${VROOLI_BIN_DIR}"
    mkdir -p "${VROOLI_BIN_DIR}"
fi

# Check if CLI source exists
if [[ ! -f "${CLI_SOURCE_PATH}" ]]; then
    log_error "CLI source not found at: ${CLI_SOURCE_PATH}"
    exit 1
fi

# Install CLI (create symlink)
log_info "Installing ${CLI_NAME} CLI..."

if [[ -L "${CLI_INSTALL_PATH}" ]]; then
    log_info "Removing existing symlink..."
    rm "${CLI_INSTALL_PATH}"
elif [[ -f "${CLI_INSTALL_PATH}" ]]; then
    log_info "Backing up existing CLI file..."
    mv "${CLI_INSTALL_PATH}" "${CLI_INSTALL_PATH}.backup.$(date +%s)"
fi

# Create symlink
ln -s "$(realpath "${CLI_SOURCE_PATH}")" "${CLI_INSTALL_PATH}"
log_success "CLI installed at: ${CLI_INSTALL_PATH}"

# Make sure it's executable
chmod +x "${CLI_INSTALL_PATH}"

# Check if .vrooli/bin is in PATH
if ! echo "$PATH" | grep -q "${VROOLI_BIN_DIR}"; then
    log_info "Adding ~/.vrooli/bin to PATH..."
    
    # Determine shell and profile file
    SHELL_NAME=$(basename "$SHELL")
    case "$SHELL_NAME" in
        "bash")
            PROFILE_FILE="${HOME}/.bashrc"
            [[ -f "${HOME}/.bash_profile" ]] && PROFILE_FILE="${HOME}/.bash_profile"
            ;;
        "zsh")
            PROFILE_FILE="${HOME}/.zshrc"
            ;;
        "fish")
            PROFILE_FILE="${HOME}/.config/fish/config.fish"
            ;;
        *)
            PROFILE_FILE="${HOME}/.profile"
            ;;
    esac
    
    # Add PATH export to profile if not already there
    PATH_EXPORT="export PATH=\"\$PATH:${VROOLI_BIN_DIR}\""
    
    if [[ -f "$PROFILE_FILE" ]] && grep -q "${VROOLI_BIN_DIR}" "$PROFILE_FILE"; then
        log_info "PATH already includes ~/.vrooli/bin in $PROFILE_FILE"
    else
        log_info "Adding PATH export to $PROFILE_FILE"
        echo "" >> "$PROFILE_FILE"
        echo "# Vrooli CLI tools" >> "$PROFILE_FILE"
        echo "$PATH_EXPORT" >> "$PROFILE_FILE"
        log_success "Added ~/.vrooli/bin to PATH in $PROFILE_FILE"
        log_info "Please restart your shell or run: source $PROFILE_FILE"
    fi
fi

# Test the CLI
log_info "Testing CLI installation..."
if "${CLI_INSTALL_PATH}" --version >/dev/null 2>&1; then
    log_success "CLI installed successfully!"
    
    # Show usage information
    echo ""
    echo "🚀 Contact Book CLI is now available!"
    echo ""
    echo "Usage examples:"
    echo "  contact-book list"
    echo "  contact-book search 'john'"
    echo "  contact-book add --name 'Jane Doe' --email 'jane@example.com'"
    echo "  contact-book help"
    echo ""
    echo "For other scenarios to use:"
    echo "  contact-book search 'sarah' --json | jq '.results[].id'"
    echo ""
else
    log_error "CLI installation failed - command not working"
    exit 1
fi

# Create config directory
CONFIG_DIR="${HOME}/.vrooli/contact-book"
if [[ ! -d "$CONFIG_DIR" ]]; then
    log_info "Creating config directory: $CONFIG_DIR"
    mkdir -p "$CONFIG_DIR"
    
    # Create basic config file
    cat > "${CONFIG_DIR}/config.json" << EOF
{
  "api_base_url": "http://localhost:8080",
  "default_limit": 50,
  "json_output": false,
  "auto_update_check": true
}
EOF
    log_success "Created config file: ${CONFIG_DIR}/config.json"
fi

log_success "Contact Book CLI installation complete!"