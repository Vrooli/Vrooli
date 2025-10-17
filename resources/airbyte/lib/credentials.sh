#!/bin/bash
# Airbyte Credential Management Library
# Provides secure storage and retrieval of connector credentials

set -euo pipefail

# Credential storage directory
CRED_DIR="${DATA_DIR}/credentials"
CRED_MASTER_KEY="${CRED_DIR}/.master_key"

# Initialize credential storage
init_credentials() {
    if [[ ! -d "$CRED_DIR" ]]; then
        mkdir -p "$CRED_DIR"
        chmod 700 "$CRED_DIR"
    fi
    
    # Generate master key if not exists
    if [[ ! -f "$CRED_MASTER_KEY" ]]; then
        openssl rand -hex 32 > "$CRED_MASTER_KEY"
        chmod 600 "$CRED_MASTER_KEY"
    fi
}

# Encrypt credential data
encrypt_credential() {
    local data="$1"
    local key
    key=$(cat "$CRED_MASTER_KEY")
    
    echo "$data" | openssl enc -aes-256-cbc -a -salt -pass pass:"$key" -pbkdf2
}

# Decrypt credential data
decrypt_credential() {
    local encrypted="$1"
    local key
    key=$(cat "$CRED_MASTER_KEY")
    
    echo "$encrypted" | openssl enc -aes-256-cbc -d -a -pass pass:"$key" -pbkdf2
}

# Store credential
store_credential() {
    local name="$1"
    local type="$2"  # source, destination, api_key
    local data="$3"
    
    init_credentials
    
    # Create credential metadata
    local metadata
    metadata=$(jq -n \
        --arg name "$name" \
        --arg type "$type" \
        --arg created "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        '{name: $name, type: $type, created: $created}')
    
    # Encrypt sensitive data
    local encrypted
    encrypted=$(encrypt_credential "$data")
    
    # Store credential
    local cred_file="${CRED_DIR}/${name}.cred"
    jq -n \
        --argjson metadata "$metadata" \
        --arg encrypted "$encrypted" \
        '{metadata: $metadata, encrypted: $encrypted}' > "$cred_file"
    
    chmod 600 "$cred_file"
    
    log_info "Credential stored: $name"
}

# Retrieve credential
get_credential() {
    local name="$1"
    
    init_credentials
    
    local cred_file="${CRED_DIR}/${name}.cred"
    if [[ ! -f "$cred_file" ]]; then
        log_error "Credential not found: $name"
        return 1
    fi
    
    # Get encrypted data
    local encrypted
    encrypted=$(jq -r '.encrypted' "$cred_file")
    
    # Decrypt and return
    decrypt_credential "$encrypted"
}

# List credentials
list_credentials() {
    init_credentials
    
    if [[ ! -d "$CRED_DIR" ]]; then
        echo "[]"
        return
    fi
    
    local credentials=()
    shopt -s nullglob
    for cred_file in "$CRED_DIR"/*.cred; do
        if [[ -f "$cred_file" ]]; then
            local metadata
            metadata=$(jq '.metadata' "$cred_file")
            credentials+=("$metadata")
        fi
    done
    shopt -u nullglob
    
    if [[ ${#credentials[@]} -eq 0 ]]; then
        echo "[]"
    else
        printf '%s\n' "${credentials[@]}" | jq -s '.'
    fi
}

# Remove credential
remove_credential() {
    local name="$1"
    
    local cred_file="${CRED_DIR}/${name}.cred"
    if [[ ! -f "$cred_file" ]]; then
        log_error "Credential not found: $name"
        return 1
    fi
    
    rm -f "$cred_file"
    log_info "Credential removed: $name"
}

# Rotate master key
rotate_master_key() {
    init_credentials
    
    log_info "Rotating master key..."
    
    # Create backup of old key
    local old_key
    old_key=$(cat "$CRED_MASTER_KEY")
    cp "$CRED_MASTER_KEY" "${CRED_MASTER_KEY}.old"
    
    # Generate new key
    openssl rand -hex 32 > "$CRED_MASTER_KEY"
    chmod 600 "$CRED_MASTER_KEY"
    
    # Re-encrypt all credentials with new key
    shopt -s nullglob
    for cred_file in "$CRED_DIR"/*.cred; do
        if [[ -f "$cred_file" ]]; then
            # Get encrypted data with old key
            local encrypted
    encrypted=$(jq -r '.encrypted' "$cred_file")
            local decrypted
            decrypted=$(echo "$encrypted" | openssl enc -aes-256-cbc -d -a -pass pass:"$old_key" -pbkdf2)
            
            # Re-encrypt with new key
            local new_encrypted
            new_encrypted=$(encrypt_credential "$decrypted")
            
            # Update credential file
            local metadata
            metadata=$(jq '.metadata' "$cred_file")
            jq -n \
                --argjson metadata "$metadata" \
                --arg encrypted "$new_encrypted" \
                '{metadata: $metadata, encrypted: $encrypted}' > "${cred_file}.new"
            
            mv "${cred_file}.new" "$cred_file"
        fi
    done
    shopt -u nullglob
    
    # Remove old key backup
    rm -f "${CRED_MASTER_KEY}.old"
    
    log_info "Master key rotated successfully"
}

# Apply credential to connector configuration
apply_credential() {
    local config="$1"
    local cred_name="$2"
    
    # Get credential
    local cred_data
    if ! cred_data=$(get_credential "$cred_name"); then
        return 1
    fi
    
    # Merge credential with config
    echo "$config" | jq --argjson cred "$cred_data" '. + $cred'
}

# Validate credential format
validate_credential() {
    local data="$1"
    local type="$2"
    
    case "$type" in
        api_key)
            # Check for required fields
            echo "$data" | jq -e '.api_key' > /dev/null || return 1
            ;;
        source|destination)
            # Check for connection configuration
            echo "$data" | jq -e '.connectionConfiguration' > /dev/null || return 1
            ;;
        *)
            log_error "Unknown credential type: $type"
            return 1
            ;;
    esac
    
    return 0
}