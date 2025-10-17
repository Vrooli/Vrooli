#!/usr/bin/env bash
# Redis Resource Messages
# User-facing messages for Redis management operations

# Installation Messages
MSG_INSTALL_STARTING="🚀 Installing Redis resource..."
MSG_INSTALL_SUCCESS="✅ Redis resource installed successfully"
MSG_INSTALL_FAILED="❌ Redis installation failed"
MSG_ALREADY_INSTALLED="ℹ️  Redis is already installed"
MSG_PULLING_IMAGE="📥 Pulling Redis Docker image..."
MSG_CREATING_DIRECTORIES="📁 Creating Redis directories..."
MSG_GENERATING_CONFIG="⚙️  Generating Redis configuration..."

# Start/Stop Messages
MSG_STARTING_CONTAINER="🚀 Starting Redis container..."
MSG_START_SUCCESS="✅ Redis started successfully"
MSG_START_FAILED="❌ Failed to start Redis"
MSG_STOPPING_CONTAINER="🛑 Stopping Redis container..."
MSG_STOP_SUCCESS="✅ Redis stopped successfully"
MSG_STOP_FAILED="❌ Failed to stop Redis"
MSG_RESTART_SUCCESS="✅ Redis restarted successfully"

# Status Messages
MSG_STATUS_RUNNING="🟢 Redis is running"
MSG_STATUS_STOPPED="🔴 Redis is stopped"
MSG_STATUS_NOT_INSTALLED="❌ Redis is not installed"
MSG_STATUS_UNHEALTHY="⚠️  Redis is unhealthy"
MSG_CHECKING_STATUS="🔍 Checking Redis status..."

# Connection Messages
MSG_CONNECTION_INFO="📡 Redis Connection Information:"
MSG_CONNECTION_HOST="   Host: localhost"
MSG_CONNECTION_PORT="   Port: ${REDIS_PORT}"
MSG_CONNECTION_CLI="   CLI: redis-cli -p ${REDIS_PORT}"
MSG_CONNECTION_URL="   URL: redis://localhost:${REDIS_PORT}"

# Client Instance Messages
MSG_CLIENT_CREATE_START="🚀 Creating Redis instance for client: "
MSG_CLIENT_CREATE_SUCCESS="✅ Client Redis instance created successfully"
MSG_CLIENT_CREATE_FAILED="❌ Failed to create client Redis instance"
MSG_CLIENT_DESTROY_START="🗑️  Destroying Redis instance for client: "
MSG_CLIENT_DESTROY_SUCCESS="✅ Client Redis instance destroyed"
MSG_CLIENT_DESTROY_FAILED="❌ Failed to destroy client Redis instance"
MSG_CLIENT_PORT_ALLOCATED="📍 Allocated port for client: "

# Database Messages
MSG_DATABASE_COUNT="📊 Number of databases: ${REDIS_DATABASES}"
MSG_DATABASE_SELECT="🔄 Switching to database: "
MSG_DATABASE_FLUSH="⚠️  Flushing database: "
MSG_DATABASE_FLUSH_ALL="⚠️  Flushing ALL databases"
MSG_DATABASE_FLUSH_CONFIRM="Are you sure you want to flush? This cannot be undone! (yes/no): "

# Backup/Restore Messages
MSG_BACKUP_START="💾 Starting Redis backup..."
MSG_BACKUP_SUCCESS="✅ Redis backup completed successfully"
MSG_BACKUP_FAILED="❌ Redis backup failed"
MSG_BACKUP_LOCATION="📁 Backup saved to: "
MSG_RESTORE_START="🔄 Starting Redis restore..."
MSG_RESTORE_SUCCESS="✅ Redis restore completed successfully"
MSG_RESTORE_FAILED="❌ Redis restore failed"
MSG_RESTORE_FILE_NOT_FOUND="❌ Backup file not found: "

# Configuration Messages
MSG_CONFIG_UPDATE="⚙️  Updating Redis configuration..."
MSG_CONFIG_RELOAD="🔄 Reloading Redis configuration..."
MSG_CONFIG_SUCCESS="✅ Configuration updated successfully"
MSG_CONFIG_FAILED="❌ Configuration update failed"
MSG_CONFIG_MEMORY="💾 Max memory set to: ${REDIS_MAX_MEMORY}"
MSG_CONFIG_PERSISTENCE="📝 Persistence mode: ${REDIS_PERSISTENCE}"

# Performance Messages
MSG_BENCHMARK_START="⚡ Running Redis benchmark..."
MSG_BENCHMARK_COMPLETE="✅ Benchmark completed"
MSG_STATS_HEADER="📊 Redis Statistics:"
MSG_MEMORY_USAGE="💾 Memory Usage: "
MSG_CONNECTED_CLIENTS="👥 Connected Clients: "
MSG_TOTAL_COMMANDS="🔢 Total Commands Processed: "
MSG_OPS_PER_SECOND="⚡ Operations/Second: "

# Error Messages
MSG_ERROR_DOCKER="❌ Docker is not running or not installed"
MSG_ERROR_PORT_IN_USE="❌ Port ${REDIS_PORT} is already in use"
MSG_ERROR_CONNECTION="❌ Cannot connect to Redis"
MSG_ERROR_PERMISSION="❌ Permission denied. Try running with appropriate permissions"
MSG_ERROR_CLIENT_EXISTS="❌ Client instance already exists: "
MSG_ERROR_CLIENT_NOT_FOUND="❌ Client instance not found: "
MSG_ERROR_INVALID_ACTION="❌ Invalid action: "
MSG_ERROR_MISSING_PARAM="❌ Missing required parameter: "

# Warning Messages
MSG_WARN_DATA_LOSS="⚠️  WARNING: This will permanently delete all Redis data!"
MSG_WARN_PRODUCTION="⚠️  WARNING: Not recommended for production use without proper security configuration"
MSG_WARN_NO_PASSWORD="⚠️  WARNING: Redis is running without password protection"

# Help Messages
MSG_HELP_HEADER="Redis Resource Management"
MSG_HELP_USAGE="Usage: $0 --action <action> [options]"
MSG_HELP_ACTIONS="Available actions:"
MSG_HELP_CLI_EXAMPLE="Example CLI usage: redis-cli -p ${REDIS_PORT}"
MSG_HELP_CONNECT_EXAMPLE="Example connection: redis://localhost:${REDIS_PORT}/0"

#######################################
# Initialize messages with current configuration
#######################################
redis::messages::init() {
    # Update messages that include variables
    MSG_CONNECTION_PORT="   Port: ${REDIS_PORT}"
    MSG_CONNECTION_CLI="   CLI: redis-cli -p ${REDIS_PORT}"
    MSG_CONNECTION_URL="   URL: redis://localhost:${REDIS_PORT}"
    MSG_DATABASE_COUNT="📊 Number of databases: ${REDIS_DATABASES}"
    MSG_CONFIG_MEMORY="💾 Max memory set to: ${REDIS_MAX_MEMORY}"
    MSG_CONFIG_PERSISTENCE="📝 Persistence mode: ${REDIS_PERSISTENCE}"
    MSG_ERROR_PORT_IN_USE="❌ Port ${REDIS_PORT} is already in use"
    MSG_HELP_CLI_EXAMPLE="Example CLI usage: redis-cli -p ${REDIS_PORT}"
    MSG_HELP_CONNECT_EXAMPLE="Example connection: redis://localhost:${REDIS_PORT}/0"
}