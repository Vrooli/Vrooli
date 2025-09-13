#!/usr/bin/env bash
################################################################################
# MinIO Core Library - v2.0 Contract Compliant
# 
# Core functionality for MinIO S3-compatible object storage
################################################################################

set -euo pipefail

# Ensure dependencies are loaded
if [[ -z "${MINIO_CLI_DIR:-}" ]]; then
    MINIO_CLI_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}/.." && builtin pwd)"
fi

# Source required libraries
for lib in common docker install status api buckets; do
    lib_file="${MINIO_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

################################################################################
# Core Installation Functions
################################################################################

minio::install() {
    log::info "Installing MinIO..."
    
    # Check if already installed
    if minio::is_installed; then
        log::warning "MinIO is already installed"
        return 2
    fi
    
    # Delegate to install library if available
    if command -v minio::install::setup &>/dev/null; then
        minio::install::setup "$@"
    else
        # Fallback installation
        minio::core::basic_install "$@"
    fi
}

minio::uninstall() {
    log::info "Uninstalling MinIO..."
    
    # Check if installed
    if ! minio::is_installed; then
        log::warning "MinIO is not installed"
        return 2
    fi
    
    # Stop if running
    if minio::is_running; then
        minio::docker::stop
    fi
    
    # Remove container
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    docker rm -f "$container_name" 2>/dev/null || true
    
    # Optionally remove data
    if [[ "${1:-}" != "--keep-data" ]]; then
        local data_dir="${HOME}/.minio"
        if [[ -d "$data_dir" ]]; then
            log::warning "Removing MinIO data directory: $data_dir"
            rm -rf "$data_dir"
        fi
    fi
    
    log::success "MinIO uninstalled successfully"
    return 0
}

################################################################################
# Core Status Functions
################################################################################

minio::is_installed() {
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    docker ps -a --filter "name=$container_name" --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"
}

minio::is_running() {
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    docker ps --filter "name=$container_name" --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"
}

minio::is_healthy() {
    local port="${MINIO_PORT:-9000}"
    timeout 5 curl -sf "http://localhost:${port}/minio/health/live" &>/dev/null
}

################################################################################
# Core Health Check
################################################################################

minio::health_check() {
    local timeout="${1:-5}"
    local port="${MINIO_PORT:-9000}"
    
    log::info "Checking MinIO health..."
    
    if ! minio::is_running; then
        log::error "MinIO container is not running"
        return 1
    fi
    
    if timeout "$timeout" curl -sf "http://localhost:${port}/minio/health/live" &>/dev/null; then
        log::success "MinIO is healthy"
        return 0
    else
        log::error "MinIO health check failed"
        return 1
    fi
}

################################################################################
# Core Bucket Operations
################################################################################

minio::create_default_buckets() {
    log::info "Creating default Vrooli buckets..."
    
    local buckets=(
        "vrooli-user-uploads"
        "vrooli-agent-artifacts"
        "vrooli-model-cache"
        "vrooli-temp-storage"
    )
    
    # First try using MC client inside container
    local created_count=0
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    
    # Configure MC alias inside container
    log::info "Configuring MinIO client in container..."
    if docker exec "$container_name" mc alias set local http://localhost:9000 \
        "${MINIO_ROOT_USER:-minioadmin}" "${MINIO_ROOT_PASSWORD:-minioadmin}" &>/dev/null; then
        
        # Create buckets using MC
        for bucket in "${buckets[@]}"; do
            log::info "Creating bucket: $bucket"
            if docker exec "$container_name" mc mb "local/$bucket" &>/dev/null 2>&1 || \
               docker exec "$container_name" mc ls "local/$bucket" &>/dev/null 2>&1; then
                ((created_count++))
                log::success "✓ Bucket ready: $bucket"
                
                # Set appropriate policies
                case "$bucket" in
                    "vrooli-user-uploads")
                        docker exec "$container_name" mc anonymous set download "local/$bucket" &>/dev/null || true
                        ;;
                    "vrooli-temp-storage")
                        # Set lifecycle policy for temp storage (24 hour expiry)
                        docker exec "$container_name" mc ilm add "local/$bucket" \
                            --expiry-days 1 &>/dev/null || true
                        ;;
                esac
            else
                log::warning "Could not create bucket: $bucket"
            fi
        done
    else
        # Fallback to existing methods
        for bucket in "${buckets[@]}"; do
            if command -v minio::buckets::create_custom &>/dev/null; then
                minio::buckets::create_custom "$bucket" "private" || true
            elif command -v minio::api::create_bucket &>/dev/null; then
                minio::api::create_bucket "$bucket" || true
            else
                log::warning "Cannot create bucket $bucket - no bucket creation function available"
            fi
        done
    fi
    
    if [[ $created_count -eq ${#buckets[@]} ]]; then
        log::success "All default buckets created successfully"
    else
        log::warning "Some default buckets may not have been created"
    fi
}

################################################################################
# Core Wait Function
################################################################################

minio::wait_for_ready() {
    local max_wait="${1:-60}"
    local elapsed=0
    local port="${MINIO_PORT:-9000}"
    
    log::info "Waiting for MinIO to be ready (max ${max_wait}s)..."
    
    while [[ $elapsed -lt $max_wait ]]; do
        if timeout 2 curl -sf "http://localhost:${port}/minio/health/live" &>/dev/null; then
            log::success "MinIO is ready"
            return 0
        fi
        
        sleep 2
        elapsed=$((elapsed + 2))
        echo -n "."
    done
    
    echo ""
    log::error "MinIO failed to become ready within ${max_wait}s"
    return 1
}

################################################################################
# Basic Installation (Fallback)
################################################################################

minio::core::basic_install() {
    log::info "Performing basic MinIO installation..."
    
    # Create data directory
    local data_dir="${HOME}/.minio/data"
    mkdir -p "$data_dir"
    
    # Generate credentials if not provided
    local root_user="${MINIO_CUSTOM_ROOT_USER:-minioadmin}"
    local root_password="${MINIO_CUSTOM_ROOT_PASSWORD:-$(openssl rand -base64 32 2>/dev/null || echo "minio123")}"
    
    # Save credentials
    local creds_file="${HOME}/.minio/config/credentials"
    mkdir -p "$(dirname "$creds_file")"
    cat > "$creds_file" <<EOF
MINIO_ROOT_USER=$root_user
MINIO_ROOT_PASSWORD=$root_password
EOF
    chmod 600 "$creds_file"
    
    # Pull Docker image
    docker pull minio/minio:latest
    
    # Create and start container
    local container_name="${MINIO_CONTAINER_NAME:-minio}"
    local api_port="${MINIO_CUSTOM_PORT:-9000}"
    local console_port="${MINIO_CUSTOM_CONSOLE_PORT:-9001}"
    
    docker run -d \
        --name "$container_name" \
        --restart unless-stopped \
        -p "${api_port}:9000" \
        -p "${console_port}:9001" \
        -e "MINIO_ROOT_USER=$root_user" \
        -e "MINIO_ROOT_PASSWORD=$root_password" \
        -v "${data_dir}:/data" \
        minio/minio server /data --console-address ":9001"
    
    # Wait for ready
    if minio::wait_for_ready 30; then
        # Ensure we have the right credentials loaded
        if [[ -f "$creds_file" ]]; then
            source "$creds_file"
            export MINIO_ROOT_USER MINIO_ROOT_PASSWORD
        fi
        
        # Create default buckets
        minio::create_default_buckets
        log::success "MinIO installed successfully"
        return 0
    else
        log::error "MinIO installation completed but service is not ready"
        return 1
    fi
}

################################################################################
# Export Functions
################################################################################

# Export all public functions
export -f minio::install
export -f minio::uninstall
export -f minio::is_installed
export -f minio::is_running
export -f minio::is_healthy
export -f minio::health_check
export -f minio::create_default_buckets
export -f minio::wait_for_ready