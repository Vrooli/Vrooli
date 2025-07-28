#!/usr/bin/env bash
# Vault User-Facing Messages
# All user interface messages and help text

#######################################
# Initialize message configuration
#######################################
vault::messages::init() {
    # Define message catalog with multi-language support
    declare -gA VAULT_MESSAGES=(
        # Installation messages
        ["MSG_VAULT_INSTALL_START"]="🔧 Installing Vault secret management service..."
        ["MSG_VAULT_INSTALL_SUCCESS"]="✅ Vault installed successfully"
        ["MSG_VAULT_INSTALL_FAILED"]="❌ Vault installation failed"
        ["MSG_VAULT_ALREADY_INSTALLED"]="ℹ️  Vault is already installed"
        
        # Configuration messages
        ["MSG_VAULT_CONFIG_CREATING"]="📋 Creating Vault configuration..."
        ["MSG_VAULT_CONFIG_SUCCESS"]="✅ Vault configuration created successfully"
        ["MSG_VAULT_CONFIG_FAILED"]="❌ Failed to create Vault configuration"
        ["MSG_VAULT_CONFIG_EXISTS"]="ℹ️  Vault configuration already exists"
        
        # Startup messages
        ["MSG_VAULT_START_STARTING"]="🚀 Starting Vault service..."
        ["MSG_VAULT_START_SUCCESS"]="✅ Vault started successfully"
        ["MSG_VAULT_START_FAILED"]="❌ Failed to start Vault"
        ["MSG_VAULT_ALREADY_RUNNING"]="ℹ️  Vault is already running"
        
        # Initialization messages
        ["MSG_VAULT_INIT_STARTING"]="🔐 Initializing Vault..."
        ["MSG_VAULT_INIT_SUCCESS"]="✅ Vault initialized successfully"
        ["MSG_VAULT_INIT_FAILED"]="❌ Vault initialization failed"
        ["MSG_VAULT_ALREADY_INITIALIZED"]="ℹ️  Vault is already initialized"
        
        # Unsealing messages
        ["MSG_VAULT_UNSEAL_STARTING"]="🔓 Unsealing Vault..."
        ["MSG_VAULT_UNSEAL_SUCCESS"]="✅ Vault unsealed successfully"
        ["MSG_VAULT_UNSEAL_FAILED"]="❌ Failed to unseal Vault"
        ["MSG_VAULT_ALREADY_UNSEALED"]="ℹ️  Vault is already unsealed"
        
        # Secret engine messages
        ["MSG_VAULT_SECRET_ENGINE_ENABLE"]="⚙️  Enabling secret engines..."
        ["MSG_VAULT_SECRET_ENGINE_SUCCESS"]="✅ Secret engines enabled successfully"
        ["MSG_VAULT_SECRET_ENGINE_FAILED"]="❌ Failed to enable secret engines"
        
        # Secret management messages
        ["MSG_VAULT_SECRET_PUT_SUCCESS"]="✅ Secret stored successfully"
        ["MSG_VAULT_SECRET_PUT_FAILED"]="❌ Failed to store secret"
        ["MSG_VAULT_SECRET_GET_SUCCESS"]="✅ Secret retrieved successfully"
        ["MSG_VAULT_SECRET_GET_FAILED"]="❌ Failed to retrieve secret"
        ["MSG_VAULT_SECRET_DELETE_SUCCESS"]="✅ Secret deleted successfully"
        ["MSG_VAULT_SECRET_DELETE_FAILED"]="❌ Failed to delete secret"
        
        # Status messages
        ["MSG_VAULT_STATUS_HEALTHY"]="✅ Vault is running and healthy"
        ["MSG_VAULT_STATUS_UNHEALTHY"]="⚠️  Vault is running but unhealthy"
        ["MSG_VAULT_STATUS_STOPPED"]="🛑 Vault is not running"
        ["MSG_VAULT_STATUS_NOT_INSTALLED"]="❌ Vault is not installed"
        
        # Stop messages
        ["MSG_VAULT_STOP_STOPPING"]="🛑 Stopping Vault service..."
        ["MSG_VAULT_STOP_SUCCESS"]="✅ Vault stopped successfully"
        ["MSG_VAULT_STOP_FAILED"]="❌ Failed to stop Vault"
        ["MSG_VAULT_NOT_RUNNING"]="ℹ️  Vault is not running"
        
        # Restart messages
        ["MSG_VAULT_RESTART_RESTARTING"]="🔄 Restarting Vault service..."
        ["MSG_VAULT_RESTART_SUCCESS"]="✅ Vault restarted successfully"
        ["MSG_VAULT_RESTART_FAILED"]="❌ Failed to restart Vault"
        
        # Uninstall messages
        ["MSG_VAULT_UNINSTALL_UNINSTALLING"]="🗑️  Uninstalling Vault..."
        ["MSG_VAULT_UNINSTALL_SUCCESS"]="✅ Vault uninstalled successfully"
        ["MSG_VAULT_UNINSTALL_FAILED"]="❌ Failed to uninstall Vault"
        ["MSG_VAULT_NOT_INSTALLED"]="ℹ️  Vault is not installed"
        
        # Development mode messages
        ["MSG_VAULT_DEV_MODE_WARNING"]="⚠️  Development mode enabled - DO NOT use in production!"
        ["MSG_VAULT_DEV_TOKEN_INFO"]="🔑 Development root token: ${VAULT_DEV_ROOT_TOKEN_ID}"
        ["MSG_VAULT_DEV_UNSEAL_INFO"]="🔓 Development mode: auto-unsealed"
        
        # Security warnings
        ["MSG_VAULT_SECURITY_WARNING"]="🔐 SECURITY: Store unseal keys and root token securely"
        ["MSG_VAULT_TOKEN_LOCATION"]="📄 Root token saved to: ${VAULT_TOKEN_FILE}"
        ["MSG_VAULT_UNSEAL_KEYS_LOCATION"]="🔑 Unseal keys saved to: ${VAULT_UNSEAL_KEYS_FILE}"
        
        # Migration messages
        ["MSG_VAULT_MIGRATION_START"]="📦 Starting secret migration..."
        ["MSG_VAULT_MIGRATION_SUCCESS"]="✅ Secret migration completed successfully"
        ["MSG_VAULT_MIGRATION_FAILED"]="❌ Secret migration failed"
        
        # Troubleshooting messages
        ["MSG_VAULT_TROUBLESHOOT_LOGS"]="📋 Check logs: ./manage.sh --action logs"
        ["MSG_VAULT_TROUBLESHOOT_CONFIG"]="⚙️  Check configuration: ./manage.sh --action status"
        ["MSG_VAULT_TROUBLESHOOT_PORT"]="🌐 Check port conflicts: netstat -tlnp | grep ${VAULT_PORT}"
        ["MSG_VAULT_TROUBLESHOOT_RESTART"]="🔄 Try restarting: ./manage.sh --action restart"
        ["MSG_VAULT_TROUBLESHOOT_REINIT"]="🔄 Try reinitializing: ./manage.sh --action init-dev"
        
        # Integration messages
        ["MSG_VAULT_INTEGRATION_N8N"]="🔗 n8n integration ready"
        ["MSG_VAULT_INTEGRATION_NODERED"]="🔗 Node-RED integration ready"
        ["MSG_VAULT_INTEGRATION_AGENT_S2"]="🔗 Agent-S2 integration ready"
        
        # Backup messages
        ["MSG_VAULT_BACKUP_START"]="💾 Starting Vault backup..."
        ["MSG_VAULT_BACKUP_SUCCESS"]="✅ Vault backup completed successfully"
        ["MSG_VAULT_BACKUP_FAILED"]="❌ Vault backup failed"
        
        # Restore messages
        ["MSG_VAULT_RESTORE_START"]="📥 Starting Vault restore..."
        ["MSG_VAULT_RESTORE_SUCCESS"]="✅ Vault restore completed successfully"
        ["MSG_VAULT_RESTORE_FAILED"]="❌ Vault restore failed"
    )
}

#######################################
# Display a message by key
# Arguments:
#   $1 - message type (info, warn, error, success)
#   $2 - message key
#   $3+ - optional arguments for message formatting
#######################################
vault::message() {
    local msg_type="$1"
    local msg_key="$2"
    shift 2
    
    local message="${VAULT_MESSAGES[$msg_key]:-$msg_key}"
    
    # Format message with arguments if provided
    if [[ $# -gt 0 ]]; then
        # shellcheck disable=SC2059
        message=$(printf "$message" "$@")
    fi
    
    case "$msg_type" in
        "info")
            log::info "$message"
            ;;
        "warn")
            log::warn "$message"
            ;;
        "error")
            log::error "$message"
            ;;
        "success")
            log::success "$message"
            ;;
        *)
            echo "$message"
            ;;
    esac
}

#######################################
# Show help for Vault commands
#######################################
vault::show_help() {
    cat << 'EOF'
HashiCorp Vault Secret Management

DESCRIPTION:
    Vault provides secure, auditable access to tokens, passwords, certificates,
    encryption keys for protecting secrets and other sensitive data using UI,
    CLI, or HTTP API.

USAGE:
    ./manage.sh --action <action> [options]

ACTIONS:
    install           Install Vault container
    uninstall         Remove Vault container and data
    start            Start Vault service
    stop             Stop Vault service
    restart          Restart Vault service
    status           Show Vault status and health
    logs             Show Vault logs
    init-dev         Initialize Vault in development mode
    init-prod        Initialize Vault in production mode
    unseal           Unseal Vault (production mode)
    put-secret       Store a secret
    get-secret       Retrieve a secret
    list-secrets     List secrets at path
    delete-secret    Delete a secret
    migrate-env      Migrate .env file to Vault
    backup           Backup Vault data
    restore          Restore Vault data

EXAMPLES:
    # Development setup
    ./manage.sh --action init-dev

    # Store a secret
    ./manage.sh --action put-secret --path "environments/dev/api-key" --value "secret123"

    # Get a secret
    ./manage.sh --action get-secret --path "environments/dev/api-key"

    # List secrets
    ./manage.sh --action list-secrets --path "environments/"

    # Migrate .env file
    ./manage.sh --action migrate-env --env-file .env --vault-prefix "environments/dev"

CONFIGURATION:
    Port: ${VAULT_PORT}
    Mode: ${VAULT_MODE}
    Data Directory: ${VAULT_DATA_DIR}
    Config Directory: ${VAULT_CONFIG_DIR}

SECURITY NOTES:
    - Development mode stores data in memory and auto-unseals
    - Production mode requires manual unsealing after restarts
    - Always secure your root token and unseal keys
    - Enable audit logging for production environments

For more information, see: ${SCRIPT_DIR}/README.md
EOF
}