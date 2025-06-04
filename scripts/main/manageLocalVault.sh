#!/usr/bin/env bash
# Manages a LOCAL Vault instance for development purposes.
# WARNING: This script is NOT for production Vault management.
set -euo pipefail
DESCRIPTION="Manages a LOCAL Vault instance (start/stop/status), with AppRole setup and optional dev secret seeding for DEVELOPMENT ONLY."

MAIN_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/flow.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/log.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/system.sh"
# shellcheck disable=SC1091
source "${MAIN_DIR}/../helpers/utils/var.sh"

# --- Configuration ---
# Default Vault address for local dev instance
# Ensure VAULT_ADDR is set, potentially from sourced files or exported
: "${VAULT_ADDR:=http://127.0.0.1:8200}"
export VAULT_ADDR
# Use a fixed root token for simplicity in dev mode (matches HashiCorp tutorial)
# Can be overridden by exporting VAULT_DEV_ROOT_TOKEN_ID
: "${VAULT_DEV_ROOT_TOKEN_ID:=root}"
export VAULT_DEV_ROOT_TOKEN_ID

LOCAL_VAULT_LOG=".vault-local.log"
LOCAL_VAULT_PID_FILE=".vault-local.pid"

# --- Helper Functions ---

# Helper function to read a PEM file, escape newlines, and prepare a 'key=value' string
prepare_pem_for_kv_pairs() {
    local key_name="$1"
    local file_path="$2"
    local file_content
    local escaped_content

    if [ ! -f "$file_path" ]; then
        log::warning "PEM file $file_path does not exist. Cannot prepare $key_name for Vault seeding."
        echo "" # Return empty string
        return
    fi

    file_content=$(cat "$file_path")

    if [ -z "$file_content" ]; then
        log::warning "PEM file $file_path is empty. Cannot prepare $key_name for Vault seeding."
        echo "" # Return empty string
        return
    fi

    # Escape newlines to store as a single line in Vault, using the sed method from user example
    escaped_content=$(echo -n "$file_content" | sed ':a;N;$!ba;s/\n/\\n/g')
    
    echo "$key_name=$escaped_content"
}

# Checks if a local Vault process seems to be running (based on PID file).
is_local_vault_running() {
    if [ -f "$LOCAL_VAULT_PID_FILE" ]; then
        local pid
        pid=$(cat "$LOCAL_VAULT_PID_FILE")
        if ps -p "$pid" > /dev/null; then
            return 0 # Process exists
        else
            # PID file exists but process doesn't - stale file?
            rm -f "$LOCAL_VAULT_PID_FILE"
            return 1
        fi
    fi
    return 1 # PID file doesn't exist
}

# Checks Vault status using the CLI.
check_vault_status() {
    log::info "Checking Vault status at $VAULT_ADDR..."
    if vault status > /dev/null 2>&1; then
        log::success "Vault is responding."
        vault status # Print full status
        return 0
    else
        log::error "Vault at $VAULT_ADDR is not responding."
        return 1
    fi
}

# Starts Vault in development mode.
start_local_vault_dev() {
    if is_local_vault_running; then
        log::warning "Local Vault appears to be running already (PID: $(cat "$LOCAL_VAULT_PID_FILE"))."
        check_vault_status
        return 0
    fi

    system::assert_command "vault" "Vault CLI not found. Please run setup first or install manually."

    log::info "Starting local Vault server in DEV mode..."
    log::info "  Address: $VAULT_ADDR"
    log::info "  Root Token: $VAULT_DEV_ROOT_TOKEN_ID"
    log::info "  Log File: $LOCAL_VAULT_LOG"

    # Start in background, redirect output, store PID
    vault server -dev -dev-listen-address="${VAULT_ADDR#*//}" -dev-root-token-id="$VAULT_DEV_ROOT_TOKEN_ID" > "$LOCAL_VAULT_LOG" 2>&1 &
    local vault_pid=$!
    echo "$vault_pid" > "$LOCAL_VAULT_PID_FILE"

    # Wait for Vault to become responsive
    local max_attempts=10
    local attempt=1
    log::info "Waiting for Vault to start (PID: $vault_pid)..."
    while [ $attempt -le $max_attempts ]; do
        # Use curl for health check as vault status might require login
        if curl -k -s "$VAULT_ADDR/v1/sys/health" | jq -e '.initialized == true and .sealed == false and .standby == false' > /dev/null 2>&1; then
            log::success "Vault started successfully (PID: $vault_pid)."
            log::success "Use 'vault login $VAULT_DEV_ROOT_TOKEN_ID' to authenticate."
            return 0
        fi
        log::info "Waiting... (Attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done

    log::error "Vault did not start responding within the expected time."
    log::error "Check logs in $LOCAL_VAULT_LOG"
    # Clean up PID file if startup failed
    rm -f "$LOCAL_VAULT_PID_FILE"
    exit "${ERROR_OPERATION_FAILED:-7}"
}

# Stops the local Vault process.
stop_local_vault() {
    if ! is_local_vault_running; then
        log::info "Local Vault does not appear to be running."
        return 0
    fi

    local pid
    pid=$(cat "$LOCAL_VAULT_PID_FILE")
    log::info "Stopping local Vault process (PID: $pid)..."

    # Try graceful shutdown first, then force
    if kill "$pid" > /dev/null 2>&1; then
        sleep 1 # Give it a moment
        if ps -p "$pid" > /dev/null; then
            log::warning "Graceful shutdown failed, sending SIGKILL..."
            kill -9 "$pid" || true
        fi
        log::success "Vault process (PID: $pid) stopped."
    else
        log::warning "Process with PID $pid not found or could not be signaled. It might have already stopped."
    fi

    rm -f "$LOCAL_VAULT_PID_FILE"
    log::success "Cleaned up PID file."
    return 0
}

# Seeds default development credentials into Vault's KV engine.
# Secrets are read from .env-dev and organized into Vault paths based on policy markers.
seed_local_dev_secrets() {
    log::header "🌱 Seeding .env-dev variables into Local Dev Vault by Policy..."
    local env_file="$var_ENV_DEV_FILE"

    if [ ! -f "$env_file" ]; then
        log::warning "No .env-dev file found at $env_file; skipping Vault seeding."
        return 0
    fi

    log::info "Loading environment variables from $env_file to determine source values..."
    # Source .env-dev to make its variables available for potential computations (like DB_URL)
    set -a
    # shellcheck disable=SC1091
    . "$env_file"
    set +a

    # Log in with dev root token for seeding operations
    log::info "Logging in to Vault with dev root token ('$VAULT_DEV_ROOT_TOKEN_ID') for seeding..."
    if ! VAULT_ADDR="$VAULT_ADDR" vault login -no-print "$VAULT_DEV_ROOT_TOKEN_ID"; then
        log::error "Failed to login with Vault dev root token. Cannot seed secrets."
        return 1
    fi
    log::success "Logged in to Vault for seeding."

    # --- Define Policy to Vault Path Mappings ---
    # These paths should align with what VSO expects (typically KVv2 paths including /data/)
    # Based on k8s/README.md and .env-example structure
    declare -A policy_to_vault_path
    policy_to_vault_path["vrooli-config-shared-all"]="secret/data/vrooli/config/shared-all"
    policy_to_vault_path["vrooli-secrets-postgres"]="secret/data/vrooli/secrets/postgres"
    policy_to_vault_path["vrooli-secrets-redis"]="secret/data/vrooli/secrets/redis"
    policy_to_vault_path["vrooli-secrets-dockerhub"]="secret/data/vrooli/dockerhub/pull-credentials" # Note: path structure
    policy_to_vault_path["vrooli-secrets-shared-server-jobs"]="secret/data/vrooli/secrets/shared-server-jobs"
    # Add other policies here if needed

    local current_policy_name=""
    local current_vault_path=""
    local -a kv_pairs_for_current_policy=()

    # Read the .env-dev file line by line
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Trim leading/trailing whitespace from the line for more robust matching
        line=$(echo "$line" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')

        # Check if the line defines a new policy block, e.g., "# Policy: policy-name-here"
        # This regex captures the policy name directly.
        if [[ "$line" =~ ^#\s*Policy:\s*([a-zA-Z0-9_-]+) ]]; then
            # Process previous policy block if one exists
            if [ -n "$current_policy_name" ] && [ ${#kv_pairs_for_current_policy[@]} -gt 0 ]; then
                log::info "Writing ${#kv_pairs_for_current_policy[@]} KV pairs for policy '$current_policy_name' to Vault path: $current_vault_path"
                if ! VAULT_ADDR="$VAULT_ADDR" VAULT_TOKEN="$VAULT_DEV_ROOT_TOKEN_ID" vault kv put "$current_vault_path" "${kv_pairs_for_current_policy[@]}"; then
                    log::error "Failed to write secrets for policy '$current_policy_name' to $current_vault_path."
                else
                    log::success "Successfully wrote secrets for policy '$current_policy_name'."
                fi
            fi

            # Start new policy block
            current_policy_name="${BASH_REMATCH[1]}" # Extracted directly from regex match
            current_vault_path="${policy_to_vault_path[$current_policy_name]}"
            kv_pairs_for_current_policy=()

            if [ -z "$current_vault_path" ]; then
                log::warning "No Vault path defined for policy '$current_policy_name'. Variables under it will be skipped."
                current_policy_name="" # Reset so we don't try to process further lines for this unknown policy
            else
                log::info "Processing policy: '$current_policy_name' -> Vault Path: '$current_vault_path'"
            fi
            continue
        fi

        # If we are inside a known policy block and the line is a valid KV pair (and not a comment)
        if [ -n "$current_vault_path" ] && [[ "$line" =~ ^[A-Za-z_][A-Za-z0-9_]*=.*$ ]] && [[ ! "$line" =~ ^\s*# ]]; then
            # Exclude Vault-specific and source keys directly here as well
            if [[ "$line" =~ ^(VAULT_|SECRETS_SOURCE) ]]; then
                continue
            fi
            kv_pairs_for_current_policy+=("$line")
        fi
    done < "$env_file"

    # Process the last policy block after loop finishes
    if [ -n "$current_policy_name" ] && [ -n "$current_vault_path" ] && [ ${#kv_pairs_for_current_policy[@]} -gt 0 ]; then
        log::info "Writing ${#kv_pairs_for_current_policy[@]} KV pairs for policy '$current_policy_name' to Vault path: $current_vault_path"
        if ! VAULT_ADDR="$VAULT_ADDR" VAULT_TOKEN="$VAULT_DEV_ROOT_TOKEN_ID" vault kv put "$current_vault_path" "${kv_pairs_for_current_policy[@]}"; then
            log::error "Failed to write secrets for policy '$current_policy_name' to $current_vault_path."
        else
            log::success "Successfully wrote secrets for policy '$current_policy_name'."
        fi
    fi

    # --- Seed JWT Keys and Derived URLs ---
    log::info "Preparing JWT keys and derived secrets for Vault..."
    local shared_secrets_path="${policy_to_vault_path["vrooli-secrets-shared-server-jobs"]}"
    if [ -z "$shared_secrets_path" ]; then
        log::error "Vault path for 'vrooli-secrets-shared-server-jobs' is not defined. Cannot seed JWT keys or derived URLs."
        # return 1 # Decide if this is fatal
    else
        declare -a derived_kv_pairs=()

        # JWT Keys from PEM files
        local jwt_priv_kv_entry
        jwt_priv_kv_entry=$(prepare_pem_for_kv_pairs "JWT_PRIV" "${var_ROOT_DIR}/jwt_priv.pem")
        if [ -n "$jwt_priv_kv_entry" ]; then
            derived_kv_pairs+=("$jwt_priv_kv_entry")
            log::info "JWT_PRIV prepared for Vault seeding (content newline-escaped)."
        fi

        local jwt_pub_kv_entry
        jwt_pub_kv_entry=$(prepare_pem_for_kv_pairs "JWT_PUB" "${var_ROOT_DIR}/jwt_pub.pem")
        if [ -n "$jwt_pub_kv_entry" ]; then
            derived_kv_pairs+=("$jwt_pub_kv_entry")
            log::info "JWT_PUB prepared for Vault seeding (content newline-escaped)."
        fi

        # Compute derived URLs (DB_URL, REDIS_URL) using variables sourced from .env-dev
        # Ensure DB_USER, DB_PASSWORD etc. are available from the sourced .env-dev
        if [ -n "${DB_USER:-}" ] && [ -n "${DB_PASSWORD:-}" ]; then # Check if DB creds are set
          local computed_db_url="DB_URL=postgresql://${DB_USER}:${DB_PASSWORD}@postgres:${PORT_DB:-5432}"
          derived_kv_pairs+=("$computed_db_url")
        else
          log::warning "DB_USER or DB_PASSWORD not found in sourced .env-dev. Cannot compute and seed DB_URL."
        fi

        if [ -n "${REDIS_PASSWORD:-}" ]; then # Check if REDIS_PASSWORD is set
          local computed_redis_url="REDIS_URL=redis://:${REDIS_PASSWORD}@redis:${PORT_REDIS:-6379}"
          derived_kv_pairs+=("$computed_redis_url")
        else
          log::warning "REDIS_PASSWORD not found in sourced .env-dev. Cannot compute and seed REDIS_URL."
        fi
        
        local computed_worker_id="WORKER_ID=${WORKER_ID:-0}" # Default to 0 if not set
        derived_kv_pairs+=("$computed_worker_id")


        if [ ${#derived_kv_pairs[@]} -gt 0 ]; then
            log::info "Writing ${#derived_kv_pairs[@]} derived/JWT entries to Vault path: $shared_secrets_path"
            # Note: This will overwrite existing keys at this path if they have the same name.
            # For a 'merge' behavior, one would need to `vault kv get` then merge and `kv put`.
            # For seeding dev secrets, `kv put` (overwrite) is usually acceptable.
            if ! VAULT_ADDR="$VAULT_ADDR" VAULT_TOKEN="$VAULT_DEV_ROOT_TOKEN_ID" vault kv put "$shared_secrets_path" "${derived_kv_pairs[@]}"; then
                log::error "Failed to write derived/JWT secrets to $shared_secrets_path."
            else
                log::success "Successfully wrote ${#derived_kv_pairs[@]} derived/JWT secrets to $shared_secrets_path."
            fi
        else
            log::warning "No JWT keys found or derived URLs to seed to $shared_secrets_path."
        fi
    fi

    log::success "Vault seeding from .env-dev finished."
    # It's good practice to revoke the token used for seeding if it's not the main dev root token.
    # Since we are using VAULT_DEV_ROOT_TOKEN_ID, we'll leave it.
    # VAULT_ADDR=$VAULT_ADDR vault token revoke -self > /dev/null # Example if using a temporary token
}

# Basic setup for AppRole (useful for local dev)
# Configures the local dev Vault instance with an AppRole auth backend,
# a basic policy, and a role for the application.
setup_local_approle_dev() {
    log::header "🚀 Setting up AppRole for Local Dev Vault..."

    # Check if Vault is running
    if ! is_local_vault_running; then
        log::error "Local Vault is not running. Please start it first using --start-dev."
        return 1
    fi
    # Check if Vault is healthy (unsealed)
    # VAULT_ADDR is implicitly used by check_vault_health via curl
    if ! check_vault_health; then
        log::error "Vault is running but not healthy/unsealed. Cannot configure AppRole."
        return 1
    fi

    # Ensure we are logged in (use the known dev root token)
    log::info "Logging in with dev root token ('$VAULT_DEV_ROOT_TOKEN_ID')..."
    if ! VAULT_ADDR=$VAULT_ADDR vault login -no-print "$VAULT_DEV_ROOT_TOKEN_ID"; then
        log::error "Failed to login with Vault dev root token."
        return 1
    fi
    log::success "Logged in successfully."
    # Re-exporting might not be necessary now, but doesn't hurt
    export VAULT_ADDR

    # 1. Enable AppRole Auth Method (idempotent)
    log::info "Ensuring approle auth method is enabled..."
    if ! VAULT_ADDR=$VAULT_ADDR vault auth list | grep -q 'approle/'; then
        VAULT_ADDR=$VAULT_ADDR vault auth enable approle || {
            log::error "Failed to enable approle auth method."
            return 1
        }
        log::success "AppRole auth method enabled."
    else
        log::info "AppRole auth method already enabled."
    fi

    # 2. Define and Write Policy (idempotent)
    local policy_name="vrooli-app-policy"
    log::info "Ensuring policy '$policy_name' exists..."
    # This policy should grant access to all paths the application (via VSO) might need.
    # It should cover all paths in policy_to_vault_path used by seed_local_dev_secrets.
    # Example: grant read to secret/data/vrooli/config/* and secret/data/vrooli/secrets/*
    # and secret/data/vrooli/dockerhub/*
    local policy_content=$(cat <<-EOF
path "secret/data/vrooli/config/*" {
  capabilities = ["read", "list"]
}
path "secret/data/vrooli/secrets/*" {
  capabilities = ["read", "list"]
}
path "secret/data/vrooli/dockerhub/*" {
  capabilities = ["read", "list"]
}
# Add other paths if necessary based on your Vault structure for Vrooli
EOF
)
    # Check if policy exists
    if ! VAULT_ADDR=$VAULT_ADDR vault policy read "$policy_name" > /dev/null 2>&1; then
        # Write policy if it doesn't exist
        echo "$policy_content" | VAULT_ADDR=$VAULT_ADDR vault policy write "$policy_name" - || {
            log::error "Failed to write policy '$policy_name'."
            return 1
        }
        log::success "Policy '$policy_name' created."
    else
        log::info "Policy '$policy_name' already exists."
    fi

    # 3. Create AppRole Role (idempotent)
    local role_name="vrooli-app-role"
    log::info "Ensuring AppRole role '$role_name' exists..."
    # Check if role exists
    if ! VAULT_ADDR=$VAULT_ADDR vault read "auth/approle/role/$role_name" > /dev/null 2>&1; then
        # Create role if it doesn't exist
        VAULT_ADDR=$VAULT_ADDR vault write "auth/approle/role/$role_name" \
            token_ttl=1h \
            token_max_ttl=4h \
            policies="default,$policy_name" || {
                log::error "Failed to create AppRole role '$role_name'."
                return 1
            }
        log::success "AppRole role '$role_name' created."
    else
        log::info "AppRole role '$role_name' already exists."
    fi

    # 4. Get RoleID
    log::info "Fetching RoleID for role '$role_name'..."
    local role_id
    # Use -format=json and jq for reliable field extraction
    # Read from the specific role-id sub-path
    role_id=$(VAULT_ADDR=$VAULT_ADDR vault read -format=json "auth/approle/role/$role_name/role-id" | jq -r .data.role_id)
    if [ -z "$role_id" ] || [ "$role_id" == "null" ]; then
        log::error "Failed to fetch RoleID for role '$role_name'."
        # Attempt to read again without jq for debugging
        VAULT_ADDR=$VAULT_ADDR vault read "auth/approle/role/$role_name/role-id"
        return 1
    fi
    log::success "Fetched RoleID ($role_id)."

    # 5. Generate SecretID
    log::info "Generating new SecretID for role '$role_name'..."
    local secret_id_data
    secret_id_data=$(VAULT_ADDR=$VAULT_ADDR vault write -f -format=json "auth/approle/role/$role_name/secret-id")
    if [ -z "$secret_id_data" ]; then
        log::error "Failed to generate SecretID for role '$role_name'."
        return 1
    fi
    local secret_id
    secret_id=$(echo "$secret_id_data" | jq -r .data.secret_id)
    local secret_id_accessor
    secret_id_accessor=$(echo "$secret_id_data" | jq -r .data.secret_id_accessor)
    if [ -z "$secret_id" ] || [ "$secret_id" == "null" ]; then
        log::error "Failed to parse SecretID from Vault response."
        return 1
    fi
    log::success "Generated new SecretID."

    # 6. Output instructions for .env-dev file
    log::header "Local Dev AppRole Configuration Complete"
    log::warning "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
    log::warning "!! The following credentials are for LOCAL DEVELOPMENT/TESTING ONLY.       !!"
    log::warning "!! Do NOT use these in staging or production environments.                 !!"
    log::warning "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!"
    log::info    "To use these credentials with your local application, update your '.env-dev' file:"
    log::info    "------------------------------------------------------------------------------"
    log::info    "# --- Vault Settings for Local Dev --- "
    log::info    "SECRETS_SOURCE=vault" 
    log::info    "VAULT_ADDR=$VAULT_ADDR" 
    log::info    "VAULT_SECRET_PATH=secret/data/vrooli/dev # Or adjust as needed"
    log::info    "VAULT_AUTH_METHOD=approle"
    log::info    "VAULT_ROLE_ID=$role_id"
    log::info    "VAULT_SECRET_ID=$secret_id"
    log::info    "------------------------------------------------------------------------------"
    # Optional: Add instructions for other auth methods if needed for testing
    log::info    "(For Token auth, use VAULT_AUTH_METHOD=token and VAULT_TOKEN=$VAULT_DEV_ROOT_TOKEN_ID)"
    log::info    "(SecretID Accessor for reference: $secret_id_accessor)"

    # Logout from root token - REMOVED FOR IDEMPOTENCY ON DEV SERVER
    # VAULT_ADDR=$VAULT_ADDR vault token revoke -self > /dev/null
    # info "Logged out from root token."

    return 0
}

# Initialize Vault for production (persistent storage)
initialize_local_vault_prod() {
    echo "TODO vault admin: initialize_local_vault_prod"
    log::error "Local production Vault initialization not yet implemented."
    return 1
}

# Unseal production Vault
unseal_local_vault_prod() {
    echo "TODO vault admin: unseal_local_vault_prod"
    log::error "Local production Vault unsealing not yet implemented."
    return 1
}

# --- Main Execution ---

parse_arguments() {
    ACTION="status" # Default action
    while [[ $# -gt 0 ]]; do
        key="$1"
        case $key in
        --start-dev)
            ACTION="start-dev"
            shift
            ;;
        --stop)
            ACTION="stop"
            shift
            ;;
        --status)
            ACTION="status"
            shift
            ;;
        -h | --help)
            echo "Usage: $0 [ACTION]"
            echo "Manages a LOCAL Vault server for DEVELOPMENT purposes only."
            echo ""
            echo "Actions:"
            echo "  --start-dev          Start Vault in dev mode, perform AppRole setup, and seed default dev credentials (requires running Vault & root login). (Default if no action)"
            echo "  --stop               Stop the running local Vault instance."
            echo "  --status             Check the status of the local Vault instance."
            echo "  -h, --help           Show this help message."
            echo ""
            echo "Environment Variables:"
            echo "  VAULT_ADDR (Default: http://127.0.0.1:8200)"
            echo "  VAULT_DEV_ROOT_TOKEN_ID (Default: root)"
            exit 0
            ;;
        *)
            log::error "Unknown option: $1"
            exit "${ERROR_USAGE:-1}"
            ;;
        esac
    done
    export ACTION
}

main() {
    parse_arguments "$@"
    log::header "🔧 Local Vault Management Utility (Dev Only) 🔧"

    case "$ACTION" in
        start-dev)
            start_local_vault_dev
            # Only seed if Vault started successfully
            if [ $? -eq 0 ]; then
                setup_local_approle_dev
                # Only seed if AppRole setup was successful (or not strictly necessary if just seeding)
                if [ $? -eq 0 ]; then
                    # Check SECRETS_SOURCE from .env-dev before seeding
                    local env_dev_secrets_source="file" # Default to file if not found or .env-dev doesn't exist
                    if [ -f "$var_ENV_DEV_FILE" ]; then
                        # Grep for SECRETS_SOURCE, strip comments, take last field after '=', convert to lowercase
                        env_dev_secrets_source=$(grep -E '^\s*SECRETS_SOURCE=' "$var_ENV_DEV_FILE" | sed 's/#.*//' | awk -F= '{print $2}' | tr '[:upper:]' '[:lower:]' | xargs || echo "file")
                    fi

                    if [[ "$env_dev_secrets_source" == "file" ]]; then
                        log::info "SECRETS_SOURCE in .env-dev is '$env_dev_secrets_source'. Proceeding with Vault seeding from .env-dev."
                        seed_local_dev_secrets
                    else
                        log::warning "SECRETS_SOURCE in .env-dev is '$env_dev_secrets_source' (not 'file'). Skipping Vault seeding from .env-dev."
                        log::info "Local Vault is running and AppRole is set up. Populate secrets manually or ensure they are already in Vault."
                    fi
                else
                    log::error "AppRole setup failed. Skipping secret seeding."
                fi
            else
                log::error "Local Vault failed to start. Skipping AppRole setup and secret seeding."
            fi
            ;;
        stop)
            stop_local_vault
            ;;
        status)
            check_vault_status
            ;;
        *)
            log::error "Invalid action specified: $ACTION"
            exit "${ERROR_USAGE:-1}"
            ;;
    esac

    log::success "✅ Local Vault management task '$ACTION' completed."
}

main "$@" 