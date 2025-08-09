#!/usr/bin/env bash
set -euo pipefail

APP_UTILS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${APP_UTILS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/flow.sh"
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/exit_codes.sh"

#######################################
# SSH Key Authorization Utility
# 
# This utility handles SSH public key authorization for CI/CD deployment:
# - Interactive key addition
# - File-based key addition  
# - SSH key validation
# - authorized_keys management
#######################################

authorize_key::validate_ssh_key() {
    local key_content="$1"
    local key_name="${2:-provided key}"
    
    log::info "Validating SSH key: $key_name"
    
    # Check if key content is provided
    if [[ -z "$key_content" ]]; then
        log::error "No SSH key content provided"
        return 1
    fi
    
    # Basic SSH key format validation
    if [[ ! "$key_content" =~ ^(ssh-rsa|ssh-ed25519|ecdsa-sha2-nistp256|ecdsa-sha2-nistp384|ecdsa-sha2-nistp521) ]]; then
        log::error "Invalid SSH key format. Key must start with ssh-rsa, ssh-ed25519, or ecdsa-*"
        return 1
    fi
    
    # Check key length (basic sanity check)
    if [[ ${#key_content} -lt 50 ]]; then
        log::error "SSH key appears too short to be valid"
        return 1
    fi
    
    log::success "✅ SSH key validation passed: $key_name"
    return 0
}

authorize_key::setup_authorized_keys() {
    local ssh_dir="$HOME/.ssh"
    local authorized_keys_file="$ssh_dir/authorized_keys"
    
    log::info "Setting up SSH authorized_keys file..."
    
    # Create .ssh directory if it doesn't exist
    if [[ ! -d "$ssh_dir" ]]; then
        log::info "Creating SSH directory: $ssh_dir"
        mkdir -p "$ssh_dir"
        chmod 700 "$ssh_dir"
    fi
    
    # Create authorized_keys file if it doesn't exist
    if [[ ! -f "$authorized_keys_file" ]]; then
        log::info "Creating authorized_keys file: $authorized_keys_file"
        touch "$authorized_keys_file"
        chmod 600 "$authorized_keys_file"
    fi
    
    # Verify permissions
    if [[ $(stat -c %a "$ssh_dir") != "700" ]]; then
        log::info "Fixing SSH directory permissions"
        chmod 700 "$ssh_dir"
    fi
    
    if [[ $(stat -c %a "$authorized_keys_file") != "600" ]]; then
        log::info "Fixing authorized_keys permissions"
        chmod 600 "$authorized_keys_file"
    fi
    
    log::success "✅ SSH authorized_keys setup complete"
    echo "$authorized_keys_file"
}

authorize_key::add_key_to_authorized_keys() {
    local key_content="$1"
    local key_name="${2:-unnamed-key}"
    local authorized_keys_file="$3"
    
    log::info "Adding SSH key to authorized_keys: $key_name"
    
    # Check if key already exists
    if grep -Fq "$key_content" "$authorized_keys_file" 2>/dev/null; then
        log::info "SSH key already exists in authorized_keys: $key_name"
        return 0
    fi
    
    # Add key with comment
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    {
        echo ""
        echo "# Added by Vrooli authorize-key utility on $timestamp"
        echo "# Key name: $key_name"
        echo "$key_content"
    } >> "$authorized_keys_file"
    
    log::success "✅ SSH key added to authorized_keys: $key_name"
}

authorize_key::read_key_from_stdin() {
    log::info "Reading SSH public key from stdin..."
    log::info "Paste your SSH public key and press Ctrl-D when done:"
    
    local key_content
    key_content=$(cat)
    
    if [[ -z "$key_content" ]]; then
        log::error "No key content received"
        return 1
    fi
    
    # Remove extra whitespace and newlines
    key_content=$(echo "$key_content" | tr -d '\n\r' | sed 's/[[:space:]]\+/ /g' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    
    echo "$key_content"
}

authorize_key::read_key_from_file() {
    local key_file="$1"
    
    log::info "Reading SSH public key from file: $key_file"
    
    if [[ ! -f "$key_file" ]]; then
        log::error "SSH key file not found: $key_file"
        return 1
    fi
    
    if [[ ! -r "$key_file" ]]; then
        log::error "Cannot read SSH key file: $key_file"
        return 1
    fi
    
    local key_content
    key_content=$(cat "$key_file")
    
    # Remove extra whitespace and newlines
    key_content=$(echo "$key_content" | tr -d '\n\r' | sed 's/[[:space:]]\+/ /g' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    
    echo "$key_content"
}

authorize_key::interactive_mode() {
    log::header "🔐 SSH Key Authorization Utility"
    
    # Setup authorized_keys file
    local authorized_keys_file
    authorized_keys_file=$(authorize_key::setup_authorized_keys)
    
    while true; do
        echo ""
        log::info "SSH Key Authorization Options:"
        echo "  1. Add key from file"
        echo "  2. Add key from stdin (paste)"
        echo "  3. Exit"
        echo ""
        
        local choice
        read -rp "Select option (1-3): " choice
        
        case $choice in
            1)
                local key_file
                read -rp "Enter path to SSH public key file: " key_file
                
                if [[ -n "$key_file" ]]; then
                    local key_content
                    if key_content=$(authorize_key::read_key_from_file "$key_file"); then
                        local key_name=$(basename "$key_file")
                        if authorize_key::validate_ssh_key "$key_content" "$key_name"; then
                            authorize_key::add_key_to_authorized_keys "$key_content" "$key_name" "$authorized_keys_file"
                        fi
                    fi
                fi
                ;;
            2)
                local key_content
                if key_content=$(authorize_key::read_key_from_stdin); then
                    if authorize_key::validate_ssh_key "$key_content" "pasted key"; then
                        local key_name
                        read -rp "Enter a name for this key (optional): " key_name
                        key_name=${key_name:-"pasted-key-$(date +%s)"}
                        authorize_key::add_key_to_authorized_keys "$key_content" "$key_name" "$authorized_keys_file"
                    fi
                fi
                ;;
            3)
                log::info "Exiting SSH key authorization"
                break
                ;;
            *)
                log::error "Invalid option: $choice"
                ;;
        esac
    done
}

authorize_key::file_mode() {
    local key_file="$1"
    local key_name="${2:-$(basename "$key_file")}"
    
    log::header "🔐 SSH Key Authorization from File"
    
    # Setup authorized_keys file
    local authorized_keys_file
    authorized_keys_file=$(authorize_key::setup_authorized_keys)
    
    # Read and validate key
    local key_content
    if key_content=$(authorize_key::read_key_from_file "$key_file"); then
        if authorize_key::validate_ssh_key "$key_content" "$key_name"; then
            authorize_key::add_key_to_authorized_keys "$key_content" "$key_name" "$authorized_keys_file"
            log::success "✅ SSH key authorization complete"
        else
            exit $ERROR_INVALID_SSH_KEY
        fi
    else
        exit $ERROR_SSH_KEY_FILE_READ
    fi
}

authorize_key::print_usage() {
    cat << EOF
SSH Key Authorization Utility for Vrooli CI/CD

USAGE:
    $0 [OPTIONS]

OPTIONS:
    --file PATH         Authorize key from file
    --name NAME         Optional name for the key
    --help              Show this help message

EXAMPLES:
    # Interactive mode
    $0

    # Authorize key from file
    $0 --file /path/to/key.pub --name "GitHub Actions"

    # Read from stdin
    cat key.pub | $0 --file -

EOF
}

authorize_key::main() {
    local key_file=""
    local key_name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --file)
                key_file="$2"
                shift 2
                ;;
            --name)
                key_name="$2"
                shift 2
                ;;
            --help)
                authorize_key::print_usage
                exit 0
                ;;
            *)
                log::error "Unknown option: $1"
                authorize_key::print_usage
                exit 1
                ;;
        esac
    done
    
    if [[ -n "$key_file" ]]; then
        if [[ "$key_file" == "-" ]]; then
            # Read from stdin
            local key_content
            if key_content=$(authorize_key::read_key_from_stdin); then
                local authorized_keys_file
                authorized_keys_file=$(authorize_key::setup_authorized_keys)
                
                if authorize_key::validate_ssh_key "$key_content" "${key_name:-stdin key}"; then
                    authorize_key::add_key_to_authorized_keys "$key_content" "${key_name:-stdin-key-$(date +%s)}" "$authorized_keys_file"
                    log::success "✅ SSH key authorization complete"
                fi
            fi
        else
            # File mode
            authorize_key::file_mode "$key_file" "$key_name"
        fi
    else
        # Interactive mode
        authorize_key::interactive_mode
    fi
}

# Execute if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    authorize_key::main "$@"
fi