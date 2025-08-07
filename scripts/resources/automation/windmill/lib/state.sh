#!/usr/bin/env bash
# Windmill State Management System
# Handles password persistence, installation state, and auto-recovery
# Ensures no sensitive data is stored in git

#######################################
# Initialize state management system
# Creates necessary directories and files
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::state_init() {
    local project_name="${1:-$WINDMILL_PROJECT_NAME}"
    local state_root="${WINDMILL_STATE_DIR:-$HOME/.windmill-state}"
    local state_dir="$state_root/installations/$project_name"
    
    # Create state directory structure
    mkdir -p "$state_dir/backups" "$state_root/global" 2>/dev/null
    
    # Create .gitignore to ensure state is never committed
    if [[ ! -f "$state_root/.gitignore" ]]; then
        cat > "$state_root/.gitignore" <<EOF
# Windmill State Management - Never commit these files
*
!.gitignore
EOF
        chmod 600 "$state_root/.gitignore"
    fi
    
    # Initialize config file if not exists
    if [[ ! -f "$state_dir/config.json" ]]; then
        cat > "$state_dir/config.json" <<EOF
{
    "project_name": "$project_name",
    "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
    "version": "1.0",
    "installation_id": "$(uuidgen 2>/dev/null || echo "$project_name-$(date +%s)")"
}
EOF
        chmod 600 "$state_dir/config.json"
    fi
    
    export WINDMILL_STATE_DIR="$state_root"
    export WINDMILL_INSTALLATION_STATE_DIR="$state_dir"
    
    return 0
}

#######################################
# Get machine-specific encryption key
# Uses machine ID or hostname as base
# Returns: Encryption key string
#######################################
windmill::get_machine_key() {
    local machine_id=""
    
    # Try multiple sources for machine identity
    if [[ -f /etc/machine-id ]]; then
        machine_id=$(cat /etc/machine-id)
    elif [[ -f /var/lib/dbus/machine-id ]]; then
        machine_id=$(cat /var/lib/dbus/machine-id)
    else
        # Fallback to hostname + MAC address
        machine_id="$(hostname)-$(ip link show | grep -m1 'link/ether' | awk '{print $2}' 2>/dev/null || echo "fallback")"
    fi
    
    # Generate consistent key from machine ID
    echo "${machine_id}${WINDMILL_PROJECT_NAME}" | sha256sum | cut -d' ' -f1
}

#######################################
# Encrypt a secret value
# Arguments:
#   $1 - Key name
#   $2 - Secret value
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::encrypt_secret() {
    local key="$1"
    local value="$2"
    local state_dir="${WINDMILL_INSTALLATION_STATE_DIR}"
    
    if [[ -z "$state_dir" ]]; then
        windmill::state_init
        state_dir="${WINDMILL_INSTALLATION_STATE_DIR}"
    fi
    
    local machine_key=$(windmill::get_machine_key)
    local secret_file="$state_dir/secrets.enc"
    local temp_file="$state_dir/.secrets.tmp"
    
    # Read existing secrets or create new
    local secrets_json="{}"
    if [[ -f "$secret_file" ]]; then
        local decrypted=$(windmill::decrypt_file "$secret_file" 2>/dev/null)
        if [[ -n "$decrypted" ]] && echo "$decrypted" | jq . >/dev/null 2>&1; then
            secrets_json="$decrypted"
        fi
    fi
    
    # Add/update the secret
    local encrypted_value=$(echo -n "$value" | openssl enc -aes-256-cbc -salt -pass "pass:$machine_key" -pbkdf2 -base64 2>/dev/null)
    
    # Update JSON with new secret
    echo "$secrets_json" | jq --arg key "$key" --arg val "$encrypted_value" '. + {($key): $val}' > "$temp_file"
    
    # Encrypt the entire secrets file
    windmill::encrypt_file "$temp_file" "$secret_file"
    rm -f "$temp_file"
    
    chmod 600 "$secret_file"
    return 0
}

#######################################
# Decrypt a secret value
# Arguments:
#   $1 - Key name
# Returns: Decrypted value or empty string
#######################################
windmill::decrypt_secret() {
    local key="$1"
    local state_dir="${WINDMILL_INSTALLATION_STATE_DIR}"
    
    if [[ -z "$state_dir" ]]; then
        windmill::state_init
        state_dir="${WINDMILL_INSTALLATION_STATE_DIR}"
    fi
    
    local secret_file="$state_dir/secrets.enc"
    
    if [[ ! -f "$secret_file" ]]; then
        return 1
    fi
    
    local machine_key=$(windmill::get_machine_key)
    local secrets_json=$(windmill::decrypt_file "$secret_file" 2>/dev/null)
    
    # Check if decryption was successful
    if [[ -z "$secrets_json" ]] || ! echo "$secrets_json" | jq . >/dev/null 2>&1; then
        return 1
    fi
    
    # Extract the specific secret
    local encrypted_value=$(echo "$secrets_json" | jq -r --arg key "$key" '.[$key] // empty' 2>/dev/null)
    
    if [[ -z "$encrypted_value" ]]; then
        return 1
    fi
    
    # Decrypt the value
    echo "$encrypted_value" | openssl enc -aes-256-cbc -d -salt -pass "pass:$machine_key" -pbkdf2 -base64 2>/dev/null
}

#######################################
# Encrypt a file
# Arguments:
#   $1 - Input file
#   $2 - Output file
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::encrypt_file() {
    local input_file="$1"
    local output_file="$2"
    local machine_key=$(windmill::get_machine_key)
    
    openssl enc -aes-256-cbc -salt -in "$input_file" -out "$output_file" \
        -pass "pass:$machine_key" -pbkdf2 2>/dev/null
    
    return $?
}

#######################################
# Decrypt a file
# Arguments:
#   $1 - Input file
# Returns: Decrypted content
#######################################
windmill::decrypt_file() {
    local input_file="$1"
    local machine_key=$(windmill::get_machine_key)
    
    openssl enc -aes-256-cbc -d -salt -in "$input_file" \
        -pass "pass:$machine_key" -pbkdf2 2>/dev/null
}

#######################################
# Detect installation state
# Checks for existing volumes and state files
# Returns: State type (new|existing|orphaned|corrupted)
#######################################
windmill::detect_installation_state() {
    local project_name="${1:-$WINDMILL_PROJECT_NAME}"
    local state_dir="${WINDMILL_INSTALLATION_STATE_DIR:-$HOME/.windmill-state/installations/$project_name}"
    
    local volume_exists=$(docker volume ls -q | grep -c "^${project_name}_db_data$")
    local state_exists=0
    local container_exists=$(docker ps -aq --filter "name=${project_name}-db" | wc -l)
    
    if [[ -f "$state_dir/config.json" ]]; then
        state_exists=1
    fi
    
    # Determine state
    if [[ $volume_exists -eq 0 && $state_exists -eq 0 ]]; then
        echo "new"
    elif [[ $volume_exists -gt 0 && $state_exists -gt 0 ]]; then
        echo "existing"
    elif [[ $volume_exists -gt 0 && $state_exists -eq 0 ]]; then
        echo "orphaned"
    elif [[ $volume_exists -eq 0 && $state_exists -gt 0 ]]; then
        echo "corrupted"
    else
        echo "unknown"
    fi
}

#######################################
# Get or generate database password
# Handles all password lifecycle scenarios
# Returns: Database password
#######################################
windmill::get_database_password() {
    local project_name="${1:-$WINDMILL_PROJECT_NAME}"
    local force_new="${2:-false}"
    
    # Initialize state if needed
    windmill::state_init "$project_name"
    
    local state=$(windmill::detect_installation_state "$project_name")
    local password=""
    
    case "$state" in
        "new")
            echo "INFO: New Windmill installation detected" >&2
            password=$(windmill::generate_secure_password)
            windmill::encrypt_secret "db_password" "$password"
            windmill::save_password_history "$password"
            ;;
            
        "existing")
            echo "INFO: Existing Windmill installation detected" >&2
            password=$(windmill::decrypt_secret "db_password")
            
            if [[ -z "$password" ]]; then
                echo "WARN: No password found in state, attempting recovery" >&2
                password=$(windmill::recover_orphaned_password)
            fi
            ;;
            
        "orphaned")
            echo "WARN: Found Windmill volumes without state file" >&2
            password=$(windmill::recover_orphaned_password)
            ;;
            
        "corrupted")
            echo "WARN: State file exists but Docker volumes missing" >&2
            if [[ "$force_new" == "true" ]]; then
                echo "INFO: Force flag set, generating new password" >&2
                password=$(windmill::generate_secure_password)
                windmill::encrypt_secret "db_password" "$password"
            else
                password=$(windmill::decrypt_secret "db_password")
                echo "INFO: Using password from state file" >&2
            fi
            ;;
            
        *)
            echo "ERROR: Unknown installation state: $state" >&2
            return 1
            ;;
    esac
    
    if [[ -z "$password" ]]; then
        echo "ERROR: Failed to determine database password" >&2
        return 1
    fi
    
    echo "$password"
    return 0
}

#######################################
# Generate a secure random password
# Returns: Secure password string
#######################################
windmill::generate_secure_password() {
    local length="${1:-32}"
    
    # Try different methods to generate secure password
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -base64 "$length" | tr -d "=+/" | cut -c1-"$length"
    elif command -v pwgen >/dev/null 2>&1; then
        pwgen -s "$length" 1
    else
        # Fallback to /dev/urandom
        tr -dc 'A-Za-z0-9!@#$%^&*()_+=' < /dev/urandom | head -c "$length"
    fi
}

#######################################
# Save password to history for recovery
# Arguments:
#   $1 - Password to save
# Returns: 0 if successful
#######################################
windmill::save_password_history() {
    local password="$1"
    local state_dir="${WINDMILL_INSTALLATION_STATE_DIR}"
    local history_file="$state_dir/backups/password_history.enc"
    local max_history="${WINDMILL_MAX_PASSWORD_HISTORY:-5}"
    
    # Get current history
    local history="[]"
    if [[ -f "$history_file" ]]; then
        history=$(windmill::decrypt_file "$history_file" 2>/dev/null || echo "[]")
    fi
    
    # Add new password with timestamp
    local entry="{\"password\": \"$password\", \"timestamp\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\"}"
    history=$(echo "$history" | jq --argjson entry "$entry" '. + [$entry] | .[-'"$max_history"':]')
    
    # Save encrypted history
    local temp_file="$state_dir/.history.tmp"
    echo "$history" > "$temp_file"
    windmill::encrypt_file "$temp_file" "$history_file"
    rm -f "$temp_file"
    
    chmod 600 "$history_file"
    return 0
}

#######################################
# Test database connection with password
# Arguments:
#   $1 - Password to test
#   $2 - Database container name (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
windmill::test_database_connection() {
    local password="$1"
    local container_name="${2:-${WINDMILL_PROJECT_NAME}-db}"
    
    # Check if container is running
    if ! docker ps --format '{{.Names}}' | grep -q "^${container_name}$"; then
        return 1
    fi
    
    # Test connection with password from external network (like Windmill app would)
    docker run --rm --network "${WINDMILL_PROJECT_NAME}_network" postgres:16 \
        psql "postgresql://postgres:${password}@${container_name}:5432/windmill" \
        -c "SELECT 1" >/dev/null 2>&1
}

#######################################
# Recover password for orphaned installation
# Tries various recovery methods
# Returns: Recovered password or empty
#######################################
windmill::recover_orphaned_password() {
    echo "INFO: Attempting to recover password for orphaned installation"
    
    # Check if .env file exists with password
    local env_file="${SCRIPT_DIR}/docker/.env"
    if [[ -f "$env_file" ]]; then
        local env_password=$(grep "^WINDMILL_DB_PASSWORD=" "$env_file" | cut -d'=' -f2)
        if [[ -n "$env_password" ]] && windmill::test_database_connection "$env_password"; then
            echo "SUCCESS: Recovered password from .env file"
            windmill::encrypt_secret "db_password" "$env_password"
            windmill::save_password_history "$env_password"
            echo "$env_password"
            return 0
        fi
    fi
    
    # Try default passwords
    local default_passwords=("changeme-secure-password-here" "changeme" "windmill")
    for pwd in "${default_passwords[@]}"; do
        if windmill::test_database_connection "$pwd"; then
            echo "SUCCESS: Recovered using default password"
            windmill::encrypt_secret "db_password" "$pwd"
            windmill::save_password_history "$pwd"
            echo "$pwd"
            return 0
        fi
    done
    
    # Generate new password and force reset
    echo "INFO: Could not recover password, generating new one"
    local new_password=$(windmill::generate_secure_password)
    
    if windmill::reset_database_password "$new_password"; then
        windmill::encrypt_secret "db_password" "$new_password"
        windmill::save_password_history "$new_password"
        echo "$new_password"
        return 0
    fi
    
    return 1
}

#######################################
# Reset database password
# Arguments:
#   $1 - New password
#   $2 - Container name (optional)
# Returns: 0 if successful
#######################################
windmill::reset_database_password() {
    local new_password="$1"
    local container_name="${2:-${WINDMILL_PROJECT_NAME}-db}"
    
    echo "INFO: Resetting database password"
    
    # Use trust authentication temporarily to reset password
    docker exec "$container_name" bash -c "
        cp /var/lib/postgresql/data/pg_hba.conf /var/lib/postgresql/data/pg_hba.conf.bak
        echo 'local all all trust' > /var/lib/postgresql/data/pg_hba.conf
        echo 'host all all all trust' >> /var/lib/postgresql/data/pg_hba.conf
        pg_ctl reload -D /var/lib/postgresql/data
        sleep 2
        psql -U postgres -c \"ALTER USER postgres PASSWORD '${new_password}';\"
        mv /var/lib/postgresql/data/pg_hba.conf.bak /var/lib/postgresql/data/pg_hba.conf
        pg_ctl reload -D /var/lib/postgresql/data
    " >/dev/null 2>&1
    
    sleep 3
    
    # Test new password
    windmill::test_database_connection "$new_password"
}

#######################################
# Perform preflight checks before starting
# Returns: 0 if all checks pass
#######################################
windmill::preflight_checks() {
    local all_passed=true
    
    echo "INFO: Running preflight checks..."
    
    # Check 1: Database password validity
    if [[ -n "${WINDMILL_DB_PASSWORD:-}" ]]; then
        if windmill::test_database_connection "$WINDMILL_DB_PASSWORD"; then
            echo "SUCCESS: ✓ Database password valid"
        else
            echo "WARN: ✗ Database password invalid or database not accessible"
            
            if [[ "${WINDMILL_AUTO_RECOVER:-true}" == "true" ]]; then
                echo "INFO: Attempting auto-recovery..."
                if windmill::auto_recover_password; then
                    echo "SUCCESS: ✓ Password recovery successful"
                else
                    echo "ERROR: ✗ Password recovery failed"
                    all_passed=false
                fi
            else
                all_passed=false
            fi
        fi
    fi
    
    # Check 2: Volume consistency
    local volumes_ok=true
    for volume in db_data server_data worker_data logs; do
        local volume_name="${WINDMILL_PROJECT_NAME}_${volume}"
        if docker volume ls -q | grep -q "^${volume_name}$"; then
            echo "SUCCESS: ✓ Volume exists: $volume_name"
        else
            echo "INFO: ○ Volume will be created: $volume_name"
        fi
    done
    
    # Check 3: Network availability
    local network_name="${WINDMILL_PROJECT_NAME}_network"
    if docker network ls --format '{{.Name}}' | grep -q "^${network_name}$"; then
        echo "SUCCESS: ✓ Network exists: $network_name"
    else
        echo "INFO: ○ Network will be created: $network_name"
    fi
    
    # Check 4: Port availability
    if ! netstat -tuln 2>/dev/null | grep -q ":${WINDMILL_SERVER_PORT} "; then
        echo "SUCCESS: ✓ Port available: $WINDMILL_SERVER_PORT"
    else
        echo "ERROR: ✗ Port in use: $WINDMILL_SERVER_PORT"
        all_passed=false
    fi
    
    if [[ "$all_passed" == "true" ]]; then
        echo "SUCCESS: All preflight checks passed"
        return 0
    else
        echo "ERROR: Some preflight checks failed"
        return 1
    fi
}

#######################################
# Auto-recover password mismatch
# Returns: 0 if successful
#######################################
windmill::auto_recover_password() {
    local state_password=$(windmill::decrypt_secret "db_password")
    
    if [[ -z "$state_password" ]]; then
        echo "ERROR: No password found in state"
        return 1
    fi
    
    echo "INFO: Synchronizing password with database"
    
    if windmill::reset_database_password "$state_password"; then
        # Update .env file
        local env_file="${SCRIPT_DIR}/docker/.env"
        if [[ -f "$env_file" ]]; then
            sed -i "s|^WINDMILL_DB_PASSWORD=.*|WINDMILL_DB_PASSWORD=${state_password}|" "$env_file"
        fi
        
        export WINDMILL_DB_PASSWORD="$state_password"
        echo "SUCCESS: Password synchronized successfully"
        return 0
    fi
    
    return 1
}

#######################################
# Lock state to prevent concurrent access
# Arguments:
#   $1 - Lock timeout in seconds (default: 30)
# Returns: 0 if lock acquired
#######################################
windmill::acquire_lock() {
    local timeout="${1:-30}"
    local lock_file="${WINDMILL_INSTALLATION_STATE_DIR}/.lock"
    local start_time=$(date +%s)
    
    while [[ -f "$lock_file" ]]; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [[ $elapsed -gt $timeout ]]; then
            echo "WARN: Lock timeout exceeded, forcing lock"
            rm -f "$lock_file"
            break
        fi
        
        sleep 1
    done
    
    echo "$$" > "$lock_file"
    return 0
}

#######################################
# Release state lock
# Returns: 0
#######################################
windmill::release_lock() {
    local lock_file="${WINDMILL_INSTALLATION_STATE_DIR}/.lock"
    rm -f "$lock_file"
    return 0
}

#######################################
# Update state configuration
# Arguments:
#   $1 - Key
#   $2 - Value
# Returns: 0 if successful
#######################################
windmill::update_state_config() {
    local key="$1"
    local value="$2"
    local config_file="${WINDMILL_INSTALLATION_STATE_DIR}/config.json"
    
    if [[ ! -f "$config_file" ]]; then
        windmill::state_init
    fi
    
    local temp_file="${config_file}.tmp"
    jq --arg key "$key" --arg val "$value" '. + {($key): $val}' "$config_file" > "$temp_file"
    mv "$temp_file" "$config_file"
    chmod 600 "$config_file"
    
    return 0
}

#######################################
# Get state configuration value
# Arguments:
#   $1 - Key
# Returns: Value or empty
#######################################
windmill::get_state_config() {
    local key="$1"
    local config_file="${WINDMILL_INSTALLATION_STATE_DIR}/config.json"
    
    if [[ ! -f "$config_file" ]]; then
        return 1
    fi
    
    jq -r --arg key "$key" '.[$key] // empty' "$config_file"
}