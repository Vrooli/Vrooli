#!/usr/bin/env bash
# Vault Storage Strategies - Handles different storage backends for Vault
# Extracted from docker.sh to simplify the main implementation

#######################################
# Determine best storage strategy
# Returns: volumes, bind, or inmem
#######################################
vault::storage::determine_strategy() {
    if [[ -n "${VAULT_STORAGE_STRATEGY:-}" ]] && [[ "${VAULT_STORAGE_STRATEGY}" != "auto" ]]; then
        echo "${VAULT_STORAGE_STRATEGY}"
        return 0
    fi
    
    # Auto-detect best strategy
    if [[ -f /.dockerenv ]]; then
        # Running inside container/CI
        echo "inmem"
    elif command -v docker &>/dev/null && docker volume ls &>/dev/null 2>&1; then
        # Docker available with volume support
        echo "volumes"
    elif [[ -w "${HOME}" ]]; then
        # Can write to home directory
        echo "bind"
    else
        # Restricted environment
        echo "inmem"
    fi
}

#######################################
# Create Docker volumes for Vault
# Returns: 0 on success, 1 on failure
#######################################
vault::storage::create_volumes() {
    local volumes=("${VAULT_VOLUME_DATA}" "${VAULT_VOLUME_CONFIG}" "${VAULT_VOLUME_LOGS}")
    
    for volume in "${volumes[@]}"; do
        if ! docker volume inspect "${volume}" >/dev/null 2>&1; then
            log::info "Creating Docker volume: ${volume}"
            if ! docker volume create "${volume}" >/dev/null 2>&1; then
                log::error "Failed to create volume: ${volume}"
                return 1
            fi
        else
            log::info "Volume already exists: ${volume}"
        fi
    done
    
    return 0
}

#######################################
# Remove Docker volumes for Vault
# Arguments:
#   $1 - force removal (yes/no)
#######################################
vault::storage::remove_volumes() {
    local force="${1:-no}"
    local volumes=("${VAULT_VOLUME_DATA}" "${VAULT_VOLUME_CONFIG}" "${VAULT_VOLUME_LOGS}")
    
    for volume in "${volumes[@]}"; do
        if docker volume inspect "${volume}" >/dev/null 2>&1; then
            if [[ "${force}" == "yes" ]]; then
                log::info "Removing Docker volume: ${volume}"
                docker volume rm "${volume}" >/dev/null 2>&1 || true
            else
                log::warn "Volume exists but not removing (use --remove-data yes): ${volume}"
            fi
        fi
    done
}

#######################################
# Copy configuration to Docker volume
# Returns: 0 on success, 1 on failure
#######################################
vault::storage::copy_config_to_volume() {
    local temp_config="/tmp/vault-config-$$"
    
    # Create temporary config directory
    mkdir -p "${temp_config}"
    
    # Generate configuration
    vault::create_config "${VAULT_MODE}"
    
    # Copy local config to temp
    if [[ -f "${VAULT_CONFIG_DIR}/vault.hcl" ]]; then
        cp "${VAULT_CONFIG_DIR}/vault.hcl" "${temp_config}/vault.hcl"
    fi
    
    # Copy TLS certificates if they exist
    if [[ -d "${VAULT_CONFIG_DIR}/tls" ]]; then
        cp -r "${VAULT_CONFIG_DIR}/tls" "${temp_config}/tls"
    fi
    
    # Use Alpine container to copy files to volume
    log::info "Copying configuration to Docker volume..."
    docker run --rm \
        -v "${VAULT_VOLUME_CONFIG}:/target" \
        -v "${temp_config}:/source:ro" \
        alpine sh -c "cp -r /source/* /target/ && chmod -R 755 /target"
    
    local result=$?
    
    # Clean up temp directory
    trash::safe_remove "${temp_config}" --temp
    
    return ${result}
}

#######################################
# Prepare bind mount directories with correct permissions
# Returns: 0 on success, 1 on failure
#######################################
vault::storage::prepare_bind_directories() {
    local dirs=("${VAULT_DATA_DIR}" "${VAULT_CONFIG_DIR}" "${VAULT_LOGS_DIR}")
    
    for dir in "${dirs[@]}"; do
        if [[ ! -d "${dir}" ]]; then
            if ! mkdir -p "${dir}" 2>/dev/null; then
                log::error "Cannot create directory: ${dir}"
                log::info "Try: sudo mkdir -p ${dir} && sudo chown $(sudo::get_actual_user 2>/dev/null || echo '$USER'):$(sudo::get_actual_group 2>/dev/null || echo '$USER') ${dir}"
                return 1
            fi
        fi
        
        # Check if directory is writable
        if [[ ! -w "${dir}" ]]; then
            log::error "Directory not writable: ${dir}"
            log::info "Try: sudo chown $(sudo::get_actual_user 2>/dev/null || echo '$USER'):$(sudo::get_actual_group 2>/dev/null || echo '$USER') ${dir}"
            return 1
        fi
        
        # Set permissions
        chmod 755 "${dir}" 2>/dev/null || true
    done
    
    # Special handling for TLS files
    if [[ -d "${VAULT_CONFIG_DIR}/tls" ]]; then
        chmod 755 "${VAULT_CONFIG_DIR}/tls" 2>/dev/null || true
        chmod 644 "${VAULT_CONFIG_DIR}/tls"/*.crt 2>/dev/null || true
        chmod 640 "${VAULT_CONFIG_DIR}/tls"/*.key 2>/dev/null || true
    fi
    
    return 0
}

#######################################
# Prepare storage based on strategy
# Arguments:
#   $1 - storage strategy (volumes/bind/inmem)
# Returns: 0 on success, 1 on failure
#######################################
vault::storage::prepare() {
    local strategy="${1}"
    
    case "${strategy}" in
        volumes)
            # Create Docker volumes
            if ! vault::storage::create_volumes; then
                log::error "Failed to create Docker volumes"
                return 1
            fi
            # Copy configuration to volume
            if ! vault::storage::copy_config_to_volume; then
                log::error "Failed to copy configuration to volume"
                return 1
            fi
            ;;
        bind)
            # Prepare bind mount directories
            if ! vault::storage::prepare_bind_directories; then
                log::error "Failed to prepare bind mount directories"
                return 1
            fi
            # Ensure directories and config exist
            vault::ensure_directories
            vault::create_config "$VAULT_MODE"
            ;;
        inmem)
            # In-memory storage, minimal setup needed
            vault::ensure_directories
            vault::create_config "$VAULT_MODE"
            ;;
        *)
            log::error "Unknown storage strategy: ${strategy}"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Get volume mounts based on storage strategy
# Arguments:
#   $1 - storage strategy (volumes/bind/inmem)
# Returns: Space-separated volume mount strings
#######################################
vault::storage::get_volumes() {
    local strategy="${1}"
    
    case "${strategy}" in
        volumes)
            echo "${VAULT_VOLUME_DATA}:/vault/data ${VAULT_VOLUME_CONFIG}:/vault/config ${VAULT_VOLUME_LOGS}:/vault/logs"
            ;;
        bind)
            echo "${VAULT_DATA_DIR}:/vault/data ${VAULT_CONFIG_DIR}:/vault/config ${VAULT_LOGS_DIR}:/vault/logs"
            ;;
        inmem)
            # Only mount config if in production mode
            if [[ "$VAULT_MODE" == "prod" ]]; then
                echo "${VAULT_CONFIG_DIR}:/vault/config:ro"
            else
                echo ""
            fi
            ;;
        *)
            echo ""
            ;;
    esac
}

#######################################
# Repair permissions for Vault directories
# Returns: 0 on success, 1 on failure
#######################################
vault::storage::repair_permissions() {
    log::info "Attempting to repair Vault directory permissions..."
    
    local dirs=("${VAULT_DATA_DIR}" "${VAULT_CONFIG_DIR}" "${VAULT_LOGS_DIR}")
    local repaired=0
    local failed=0
    
    for dir in "${dirs[@]}"; do
        if [[ -d "${dir}" ]]; then
            # Try to fix ownership
            if sudo::restore_owner "${dir}" -R; then
                log::info "Fixed ownership for: ${dir}"
                repaired=$((repaired + 1))
            else
                log::warn "Cannot fix ownership for ${dir} without elevated permissions"
                failed=$((failed + 1))
            fi
            
            # Try to fix permissions
            if chmod -R 755 "${dir}" 2>/dev/null; then
                log::info "Fixed permissions for: ${dir}"
            else
                log::warn "Cannot fix permissions for ${dir}"
            fi
        fi
    done
    
    if [[ ${failed} -gt 0 ]]; then
        log::error "Some directories need elevated permissions to repair"
        log::info "Run the following command with sudo:"
        log::info "  sudo chown -R $(sudo::get_actual_user 2>/dev/null || echo '$USER'):$(sudo::get_actual_group 2>/dev/null || echo '$USER') ${VAULT_DATA_DIR%/*"
        log::info "  sudo chmod -R 755 ${VAULT_DATA_DIR%/*"
        return 1
    fi
    
    if [[ ${repaired} -gt 0 ]]; then
        log::info "Successfully repaired ${repaired} directories"
    fi
    
    return 0
}

#######################################
# Cleanup storage resources
# Arguments:
#   $1 - storage strategy (volumes/bind/inmem)
#   $2 - remove data (yes/no)
#######################################
vault::storage::cleanup() {
    local strategy="${1}"
    local remove_data="${2:-no}"
    
    if [[ "${strategy}" == "volumes" ]]; then
        # Optionally remove volumes based on remove_data flag
        if [[ "${remove_data}" == "yes" ]]; then
            vault::storage::remove_volumes "yes"
        else
            log::info "Docker volumes preserved (use --remove-data yes to remove)"
        fi
    fi
}