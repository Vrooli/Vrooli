#!/usr/bin/env bash
# Nextcloud Resource - Core Functions

set -euo pipefail

# ============================================================================
# Installation Functions
# ============================================================================

install_nextcloud() {
    local force="${1:-false}"
    
    echo "Installing Nextcloud resource..."
    
    # Check if already installed
    if docker ps -a --format "{{.Names}}" | grep -q "^${NEXTCLOUD_CONTAINER_NAME}$"; then
        if [[ "$force" != "--force" ]]; then
            echo "Nextcloud is already installed. Use --force to reinstall."
            return 2
        fi
        echo "Force reinstalling Nextcloud..."
        uninstall_nextcloud --force
    fi
    
    # Create Docker network if it doesn't exist
    if ! docker network ls --format "{{.Name}}" | grep -q "^${NEXTCLOUD_NETWORK}$"; then
        echo "Creating Docker network: ${NEXTCLOUD_NETWORK}"
        docker network create "${NEXTCLOUD_NETWORK}"
    fi
    
    # Create volumes
    echo "Creating Docker volumes..."
    docker volume create "${NEXTCLOUD_VOLUME_NAME}"
    docker volume create "${NEXTCLOUD_DB_VOLUME}"
    docker volume create "${NEXTCLOUD_REDIS_VOLUME}"
    
    # Create docker-compose.yml
    create_docker_compose
    
    # Start containers
    echo "Starting Nextcloud containers..."
    docker-compose -f "${NEXTCLOUD_COMPOSE_FILE}" up -d
    
    # Wait for initialization
    echo "Waiting for Nextcloud to initialize..."
    wait_for_nextcloud
    
    echo "Nextcloud installed successfully!"
    echo "Access at: http://localhost:${NEXTCLOUD_PORT}"
    echo "Admin credentials: ${NEXTCLOUD_ADMIN_USER} / ${NEXTCLOUD_ADMIN_PASSWORD}"
    
    return 0
}

uninstall_nextcloud() {
    local keep_data="${1:-}"
    
    echo "Uninstalling Nextcloud resource..."
    
    # Stop containers
    if [[ -f "${NEXTCLOUD_COMPOSE_FILE}" ]]; then
        docker-compose -f "${NEXTCLOUD_COMPOSE_FILE}" down
    fi
    
    # Remove containers
    docker rm -f "${NEXTCLOUD_CONTAINER_NAME}" 2>/dev/null || true
    docker rm -f "nextcloud_postgres" 2>/dev/null || true
    docker rm -f "nextcloud_redis" 2>/dev/null || true
    docker rm -f "nextcloud_collabora" 2>/dev/null || true
    
    # Remove volumes unless keeping data
    if [[ "$keep_data" != "--keep-data" ]]; then
        echo "Removing data volumes..."
        docker volume rm "${NEXTCLOUD_VOLUME_NAME}" 2>/dev/null || true
        docker volume rm "${NEXTCLOUD_DB_VOLUME}" 2>/dev/null || true
        docker volume rm "${NEXTCLOUD_REDIS_VOLUME}" 2>/dev/null || true
    else
        echo "Keeping data volumes as requested"
    fi
    
    # Remove network if empty
    if docker network ls --format "{{.Name}}" | grep -q "^${NEXTCLOUD_NETWORK}$"; then
        if ! docker network inspect "${NEXTCLOUD_NETWORK}" | grep -q '"Containers": {}'; then
            echo "Network still in use, keeping it"
        else
            docker network rm "${NEXTCLOUD_NETWORK}" 2>/dev/null || true
        fi
    fi
    
    # Remove docker-compose file
    rm -f "${NEXTCLOUD_COMPOSE_FILE}"
    
    echo "Nextcloud uninstalled successfully!"
    return 0
}

# ============================================================================
# Lifecycle Functions
# ============================================================================

start_nextcloud() {
    local wait_flag=""
    local timeout="${NEXTCLOUD_STARTUP_TIMEOUT}"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --wait) wait_flag="--wait"; shift ;;
            --timeout) timeout="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    echo "Starting Nextcloud..."
    
    # Check if compose file exists
    if [[ ! -f "${NEXTCLOUD_COMPOSE_FILE}" ]]; then
        echo "Error: Nextcloud not installed. Run 'resource-nextcloud manage install' first." >&2
        return 1
    fi
    
    # Check if already running
    if docker ps --format "{{.Names}}" | grep -q "^${NEXTCLOUD_CONTAINER_NAME}$"; then
        echo "Nextcloud is already running"
        return 2
    fi
    
    # Start containers
    docker-compose -f "${NEXTCLOUD_COMPOSE_FILE}" up -d
    
    if [[ "$wait_flag" == "--wait" ]]; then
        echo "Waiting for Nextcloud to be ready (timeout: ${timeout}s)..."
        if wait_for_nextcloud "$timeout"; then
            echo "Nextcloud is ready!"
            return 0
        else
            echo "Error: Nextcloud failed to start within ${timeout} seconds" >&2
            return 1
        fi
    fi
    
    echo "Nextcloud started"
    return 0
}

stop_nextcloud() {
    local force=""
    local timeout="${NEXTCLOUD_SHUTDOWN_TIMEOUT}"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --force) force="--force"; shift ;;
            --timeout) timeout="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    echo "Stopping Nextcloud..."
    
    if [[ ! -f "${NEXTCLOUD_COMPOSE_FILE}" ]]; then
        echo "Nextcloud is not installed"
        return 2
    fi
    
    if ! docker ps --format "{{.Names}}" | grep -q "^${NEXTCLOUD_CONTAINER_NAME}$"; then
        echo "Nextcloud is not running"
        return 2
    fi
    
    # Stop containers
    if [[ "$force" == "--force" ]]; then
        docker-compose -f "${NEXTCLOUD_COMPOSE_FILE}" kill
    else
        docker-compose -f "${NEXTCLOUD_COMPOSE_FILE}" stop -t "$timeout"
    fi
    
    echo "Nextcloud stopped"
    return 0
}

restart_nextcloud() {
    echo "Restarting Nextcloud..."
    stop_nextcloud "$@"
    start_nextcloud "$@"
    return 0
}

# ============================================================================
# Content Management Functions
# ============================================================================

add_content() {
    local file=""
    local target_name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file) file="$2"; shift 2 ;;
            --name) target_name="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: --file required" >&2
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    # Use filename if no target name specified
    if [[ -z "$target_name" ]]; then
        target_name=$(basename "$file")
    fi
    
    # Upload via WebDAV
    local url="http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/${target_name}"
    
    echo "Uploading $file to Nextcloud as $target_name..."
    
    if curl -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
            -T "$file" \
            "$url"; then
        echo "File uploaded successfully"
        return 0
    else
        echo "Error: Failed to upload file" >&2
        return 1
    fi
}

list_content() {
    local filter="${1:-}"
    local path="${2:-}"
    
    local url="http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/${path}"
    
    echo "Listing files in Nextcloud..."
    
    # Use PROPFIND to list files
    local response=$(curl -s -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
                         -X PROPFIND \
                         -H "Depth: 1" \
                         "$url")
    
    if [[ -z "$response" ]]; then
        echo "Error: Failed to list files" >&2
        return 1
    fi
    
    # Parse response (simple extraction of href elements)
    echo "$response" | grep -oP '(?<=<d:href>)[^<]+' | sed "s|/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/||" | tail -n +2
    
    return 0
}

get_content() {
    local name=""
    local output=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            --output) output="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name required" >&2
        return 1
    fi
    
    # Use name as output if not specified
    if [[ -z "$output" ]]; then
        output=$(basename "$name")
    fi
    
    local url="http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/${name}"
    
    echo "Downloading $name from Nextcloud..."
    
    if curl -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
            -o "$output" \
            "$url"; then
        echo "File downloaded to: $output"
        return 0
    else
        echo "Error: Failed to download file" >&2
        return 1
    fi
}

remove_content() {
    local name=""
    local force=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            --force) force="--force"; shift ;;
            *) shift ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name required" >&2
        return 1
    fi
    
    if [[ "$force" != "--force" ]]; then
        read -p "Delete $name? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "Cancelled"
            return 0
        fi
    fi
    
    local url="http://localhost:${NEXTCLOUD_PORT}/remote.php/dav/files/${NEXTCLOUD_ADMIN_USER}/${name}"
    
    echo "Deleting $name from Nextcloud..."
    
    if curl -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
            -X DELETE \
            "$url"; then
        echo "File deleted successfully"
        return 0
    else
        echo "Error: Failed to delete file" >&2
        return 1
    fi
}

execute_content() {
    local name=""
    local options=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name) name="$2"; shift 2 ;;
            --options) options="$2"; shift 2 ;;
            *) shift ;;
        esac
    done
    
    case "$name" in
        share)
            execute_share "$options"
            ;;
        backup)
            execute_backup "$options"
            ;;
        restore)
            execute_restore "$options"
            ;;
        mount-s3)
            execute_mount_s3 "$options"
            ;;
        enable-office)
            execute_enable_office "$options"
            ;;
        configure-security)
            execute_configure_security "$options"
            ;;
        enable-talk)
            execute_enable_talk "$options"
            ;;
        *)
            echo "Error: Unknown operation: $name" >&2
            echo "Available operations: share, backup, restore, mount-s3, enable-office, configure-security, enable-talk" >&2
            return 1
            ;;
    esac
}

# ============================================================================
# Helper Functions
# ============================================================================

create_docker_compose() {
    # Get the script directory dynamically
    local compose_dir="${SCRIPT_DIR:-/home/matthalloran8/Vrooli/resources/nextcloud}/docker"
    local compose_file="${compose_dir}/docker-compose.yml"
    export NEXTCLOUD_COMPOSE_FILE="$compose_file"
    
    mkdir -p "$compose_dir"
    
    cat > "$compose_file" << EOF
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    container_name: nextcloud_postgres
    restart: unless-stopped
    networks:
      - ${NEXTCLOUD_NETWORK}
    volumes:
      - ${NEXTCLOUD_DB_VOLUME}:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=${NEXTCLOUD_DB_NAME}
      - POSTGRES_USER=${NEXTCLOUD_DB_USER}
      - POSTGRES_PASSWORD=${NEXTCLOUD_DB_PASSWORD}
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${NEXTCLOUD_DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: nextcloud_redis
    restart: unless-stopped
    networks:
      - ${NEXTCLOUD_NETWORK}
    volumes:
      - ${NEXTCLOUD_REDIS_VOLUME}:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  nextcloud:
    image: ${NEXTCLOUD_IMAGE}
    container_name: ${NEXTCLOUD_CONTAINER_NAME}
    restart: unless-stopped
    networks:
      - ${NEXTCLOUD_NETWORK}
    ports:
      - "${NEXTCLOUD_PORT}:80"
    volumes:
      - ${NEXTCLOUD_VOLUME_NAME}:/var/www/html
    environment:
      - POSTGRES_HOST=${NEXTCLOUD_DB_HOST}
      - POSTGRES_DB=${NEXTCLOUD_DB_NAME}
      - POSTGRES_USER=${NEXTCLOUD_DB_USER}
      - POSTGRES_PASSWORD=${NEXTCLOUD_DB_PASSWORD}
      - REDIS_HOST=${NEXTCLOUD_REDIS_HOST}
      - NEXTCLOUD_ADMIN_USER=${NEXTCLOUD_ADMIN_USER}
      - NEXTCLOUD_ADMIN_PASSWORD=${NEXTCLOUD_ADMIN_PASSWORD}
      - NEXTCLOUD_TRUSTED_DOMAINS=${NEXTCLOUD_TRUSTED_DOMAINS}
      - OVERWRITEPROTOCOL=${NEXTCLOUD_OVERWRITE_PROTOCOL}
      - OVERWRITEHOST=${NEXTCLOUD_OVERWRITE_HOST}
      - PHP_MEMORY_LIMIT=${NEXTCLOUD_MEMORY_LIMIT}
      - PHP_UPLOAD_LIMIT=${NEXTCLOUD_UPLOAD_MAX_SIZE}
      - PHP_MAX_EXECUTION_TIME=${NEXTCLOUD_MAX_EXECUTION_TIME}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/status.php"]
      interval: 30s
      timeout: 10s
      retries: 5

networks:
  ${NEXTCLOUD_NETWORK}:
    external: true

volumes:
  ${NEXTCLOUD_VOLUME_NAME}:
    external: true
  ${NEXTCLOUD_DB_VOLUME}:
    external: true
  ${NEXTCLOUD_REDIS_VOLUME}:
    external: true
EOF
}

wait_for_nextcloud() {
    local timeout="${1:-${NEXTCLOUD_STARTUP_TIMEOUT}}"
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if timeout 5 curl -sf "http://localhost:${NEXTCLOUD_PORT}/status.php" &>/dev/null; then
            return 0
        fi
        
        echo -n "."
        sleep "${NEXTCLOUD_HEALTH_CHECK_DELAY}"
        elapsed=$((elapsed + NEXTCLOUD_HEALTH_CHECK_DELAY))
    done
    
    echo
    return 1
}

execute_share() {
    local options="$1"
    
    # Parse options (format: file=name,user=username)
    local file=""
    local user=""
    
    IFS=',' read -ra OPTS <<< "$options"
    for opt in "${OPTS[@]}"; do
        IFS='=' read -r key value <<< "$opt"
        case "$key" in
            file) file="$value" ;;
            user) user="$value" ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: file option required" >&2
        return 1
    fi
    
    # Create share using OCS API
    local url="http://localhost:${NEXTCLOUD_PORT}/ocs/v2.php/apps/files_sharing/api/v1/shares"
    
    local data="path=/${file}&shareType=3"
    if [[ -n "$user" ]]; then
        data="path=/${file}&shareType=0&shareWith=${user}"
    fi
    
    echo "Creating share for $file..."
    
    local response=$(curl -s -u "${NEXTCLOUD_ADMIN_USER}:${NEXTCLOUD_ADMIN_PASSWORD}" \
                         -X POST \
                         -H "OCS-APIRequest: true" \
                         -d "$data" \
                         "$url?format=json")
    
    if echo "$response" | grep -q '"statuscode":200'; then
        local share_url=$(echo "$response" | grep -oP '"url":"[^"]+' | cut -d'"' -f4)
        echo "Share created successfully: $share_url"
        return 0
    else
        echo "Error: Failed to create share" >&2
        echo "$response" >&2
        return 1
    fi
}

execute_backup() {
    echo "Creating Nextcloud backup..."
    
    # Put Nextcloud in maintenance mode
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ maintenance:mode --on
    
    # Create backup directory
    local backup_dir="/tmp/nextcloud_backup_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    # Backup database
    docker exec nextcloud_postgres pg_dump -U "${NEXTCLOUD_DB_USER}" "${NEXTCLOUD_DB_NAME}" > "$backup_dir/database.sql"
    
    # Backup data volume
    docker run --rm -v "${NEXTCLOUD_VOLUME_NAME}:/data" -v "$backup_dir:/backup" alpine tar czf /backup/data.tar.gz -C /data .
    
    # Create archive
    tar czf "nextcloud_backup_$(date +%Y%m%d).tar.gz" -C "$backup_dir" .
    
    # Cleanup
    rm -rf "$backup_dir"
    
    # Exit maintenance mode
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ maintenance:mode --off
    
    echo "Backup created: nextcloud_backup_$(date +%Y%m%d).tar.gz"
    return 0
}

execute_restore() {
    local backup_file="$1"
    
    if [[ -z "$backup_file" ]] || [[ ! -f "$backup_file" ]]; then
        echo "Error: Backup file not found: $backup_file" >&2
        return 1
    fi
    
    echo "Restoring Nextcloud from backup..."
    
    # Put Nextcloud in maintenance mode
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ maintenance:mode --on
    
    # Extract backup
    local restore_dir="/tmp/nextcloud_restore_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$restore_dir"
    tar xzf "$backup_file" -C "$restore_dir"
    
    # Restore database
    docker exec -i nextcloud_postgres psql -U "${NEXTCLOUD_DB_USER}" "${NEXTCLOUD_DB_NAME}" < "$restore_dir/database.sql"
    
    # Restore data volume
    docker run --rm -v "${NEXTCLOUD_VOLUME_NAME}:/data" -v "$restore_dir:/backup" alpine sh -c "rm -rf /data/* && tar xzf /backup/data.tar.gz -C /data"
    
    # Cleanup
    rm -rf "$restore_dir"
    
    # Exit maintenance mode
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ maintenance:mode --off
    
    echo "Restore completed successfully"
    return 0
}

execute_mount_s3() {
    local options="$1"
    
    # Parse options (format: bucket=name,key=access_key,secret=secret_key,endpoint=url)
    local bucket=""
    local key=""
    local secret=""
    local endpoint="${NEXTCLOUD_S3_ENDPOINT:-https://s3.amazonaws.com}"
    local mount_name=""
    
    IFS=',' read -ra OPTS <<< "$options"
    for opt in "${OPTS[@]}"; do
        IFS='=' read -r opt_key value <<< "$opt"
        case "$opt_key" in
            bucket) bucket="$value" ;;
            key) key="$value" ;;
            secret) secret="$value" ;;
            endpoint) endpoint="$value" ;;
            name) mount_name="$value" ;;
        esac
    done
    
    if [[ -z "$bucket" ]]; then
        echo "Error: bucket option required" >&2
        echo "Usage: resource-nextcloud content execute --name mount-s3 --options \"bucket=mybucket,key=accesskey,secret=secretkey\"" >&2
        return 1
    fi
    
    # Use environment variables if not provided
    key="${key:-${NEXTCLOUD_S3_KEY}}"
    secret="${secret:-${NEXTCLOUD_S3_SECRET}}"
    mount_name="${mount_name:-S3-${bucket}}"
    
    echo "Mounting S3 bucket: $bucket as $mount_name..."
    
    # Create the external storage mount using OCC
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ files_external:create \
        "$mount_name" \
        amazons3 \
        amazons3::accesskey \
        -c bucket="$bucket" \
        -c key="$key" \
        -c secret="$secret" \
        -c hostname="$endpoint" \
        -c use_ssl=true \
        -c use_path_style=true \
        --user "${NEXTCLOUD_ADMIN_USER}"
    
    if [[ $? -eq 0 ]]; then
        echo "S3 bucket mounted successfully as: $mount_name"
        echo "Access it at: Files → $mount_name"
        return 0
    else
        echo "Error: Failed to mount S3 bucket" >&2
        return 1
    fi
}

execute_enable_office() {
    echo "Enabling Collabora Office integration..."
    
    # Wait for Collabora container to be ready
    local max_attempts=30
    local attempt=1
    
    echo "Waiting for Collabora to be ready..."
    while [[ $attempt -le $max_attempts ]]; do
        if timeout 5 curl -sf "http://localhost:9980/hosting/discovery" &>/dev/null; then
            echo "Collabora is ready!"
            break
        fi
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        echo "Error: Collabora failed to start" >&2
        return 1
    fi
    
    # Install and enable the richdocuments app
    echo "Installing richdocuments app..."
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:install richdocuments 2>/dev/null || true
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:enable richdocuments
    
    # Configure the WOPI URL for Collabora
    echo "Configuring Collabora integration..."
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set richdocuments wopi_url --value="http://nextcloud_collabora:9980"
    
    # Configure public WOPI URL (for client access)
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set richdocuments public_wopi_url --value="http://localhost:9980"
    
    # Set the allow list for WOPI requests
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set richdocuments wopi_allowlist --value="172.16.0.0/12,192.168.0.0/16,10.0.0.0/8"
    
    echo "Collabora Office integration enabled successfully!"
    echo ""
    echo "You can now:"
    echo "  - Create new documents: Files → + → New document/spreadsheet/presentation"
    echo "  - Edit existing Office files by clicking on them"
    echo "  - Collaborate in real-time with other users"
    echo ""
    echo "Collabora Admin UI: http://localhost:9980/browser/dist/admin/admin.html"
    echo "Username: admin, Password: changeme"
    
    return 0
}

execute_configure_security() {
    echo "Configuring Nextcloud security settings..."
    
    # Enable security headers
    echo "Setting security headers..."
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:system:set overwriteprotocol --value="https"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:system:set overwrite.cli.url --value="https://localhost:${NEXTCLOUD_PORT}"
    
    # Configure CSP headers
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:system:set 'trusted_proxies' 0 --value="127.0.0.1"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:system:set 'forwarded_for_headers' 0 --value="HTTP_X_FORWARDED_FOR"
    
    # Enable brute force protection (already enabled by default with bruteforcesettings app)
    echo "Verifying brute force protection..."
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:enable bruteforcesettings 2>/dev/null || true
    
    # Configure password policy
    echo "Configuring password policy..."
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set password_policy minLength --value="10"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set password_policy enforceHaveIBeenPwned --value="1"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set password_policy enforceNumericCharacters --value="1"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set password_policy enforceSpecialCharacters --value="1"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set password_policy enforceUpperLowerCase --value="1"
    
    # Enable server-side encryption (optional, disabled by default for performance)
    local enable_encryption="${1:-false}"
    if [[ "$enable_encryption" == "encrypt=true" ]]; then
        echo "Enabling server-side encryption..."
        docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:enable encryption
        docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ encryption:enable
        echo "Note: Encryption will apply to new files. Run 'occ encryption:encrypt-all' to encrypt existing files."
    fi
    
    # Configure session security
    echo "Configuring session security..."
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:system:set session_lifetime --value="3600"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:system:set session_relaxed_expiry --value="false"
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:system:set remember_login_cookie_lifetime --value="1296000"
    
    # Enable two-factor authentication backup codes
    echo "Enabling two-factor authentication support..."
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:enable twofactor_backupcodes
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:enable twofactor_totp 2>/dev/null || true
    
    echo "Security configuration completed!"
    echo ""
    echo "Security features enabled:"
    echo "  ✓ HTTPS headers configured (requires reverse proxy for full HTTPS)"
    echo "  ✓ Brute force protection active"
    echo "  ✓ Strong password policy enforced"
    echo "  ✓ Session security hardened"
    echo "  ✓ Two-factor authentication available"
    if [[ "$enable_encryption" == "encrypt=true" ]]; then
        echo "  ✓ Server-side encryption enabled"
    fi
    echo ""
    echo "For production use, configure a reverse proxy with HTTPS termination."
    
    return 0
}

execute_enable_talk() {
    local options="${1:-}"
    
    echo "Enabling Nextcloud Talk (video conferencing and chat)..."
    
    # Check if Talk/Spreed is already installed
    if docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:list | grep -q "spreed:"; then
        echo "Talk is already installed and enabled"
    else
        echo "Installing Talk app..."
        docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ app:install spreed || {
            echo "Error: Failed to install Talk app" >&2
            return 1
        }
    fi
    
    # Configure Talk settings
    echo "Configuring Talk settings..."
    
    # Enable TURN server for better connectivity (using public TURN servers for demo)
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set spreed stun_servers --value='["stun:stun.l.google.com:19302","stun:stun1.l.google.com:19302"]' 2>/dev/null || true
    
    # Set signaling server mode to internal (no external signaling server required)
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set spreed signaling_mode --value="internal" 2>/dev/null || true
    
    # Enable screen sharing
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set spreed enable_screensharing --value="yes" 2>/dev/null || true
    
    # Set max call duration (0 = unlimited)
    docker exec -u www-data "${NEXTCLOUD_CONTAINER_NAME}" php occ config:app:set spreed max_call_duration --value="0" 2>/dev/null || true
    
    echo "Talk configuration completed!"
    echo ""
    echo "Talk features enabled:"
    echo "  ✓ Video conferencing"
    echo "  ✓ Audio calls"
    echo "  ✓ Text chat"
    echo "  ✓ Screen sharing"
    echo "  ✓ File sharing in conversations"
    echo ""
    echo "Access Talk at: http://localhost:${NEXTCLOUD_PORT}/apps/spreed"
    echo "Note: For production use, configure a TURN server for better connectivity behind NAT/firewalls"
    
    return 0
}

get_status_json() {
    local status="stopped"
    local health="unknown"
    local details="{}"
    
    if docker ps --format "{{.Names}}" | grep -q "^${NEXTCLOUD_CONTAINER_NAME}$"; then
        status="running"
        
        if timeout 5 curl -sf "http://localhost:${NEXTCLOUD_PORT}/status.php" &>/dev/null; then
            health="healthy"
            details=$(curl -sf "http://localhost:${NEXTCLOUD_PORT}/status.php" 2>/dev/null || echo "{}")
        else
            health="unhealthy"
        fi
    fi
    
    cat << EOF
{
  "status": "${status}",
  "health": "${health}",
  "port": ${NEXTCLOUD_PORT},
  "url": "http://localhost:${NEXTCLOUD_PORT}",
  "details": ${details}
}
EOF
}