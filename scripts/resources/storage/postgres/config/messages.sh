#!/usr/bin/env bash
# PostgreSQL User-Facing Messages
# All user messages in one place for consistency

#######################################
# Initialize message constants
# Idempotent - safe to call multiple times
#######################################
postgres::messages::init() {
    # Success messages
    [[ -z "${MSG_INSTALL_SUCCESS:-}" ]] && readonly MSG_INSTALL_SUCCESS="✅ PostgreSQL resource installed successfully"
    [[ -z "${MSG_CREATE_SUCCESS:-}" ]] && readonly MSG_CREATE_SUCCESS="✅ PostgreSQL instance created successfully"
    [[ -z "${MSG_DESTROY_SUCCESS:-}" ]] && readonly MSG_DESTROY_SUCCESS="✅ PostgreSQL instance destroyed successfully"
    [[ -z "${MSG_START_SUCCESS:-}" ]] && readonly MSG_START_SUCCESS="✅ PostgreSQL instance started successfully"
    [[ -z "${MSG_STOP_SUCCESS:-}" ]] && readonly MSG_STOP_SUCCESS="✅ PostgreSQL instance stopped successfully"
    [[ -z "${MSG_RESTART_SUCCESS:-}" ]] && readonly MSG_RESTART_SUCCESS="✅ PostgreSQL instance restarted successfully"
    [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]] && readonly MSG_UNINSTALL_SUCCESS="✅ PostgreSQL resource uninstalled successfully"
    [[ -z "${MSG_DATABASE_CREATED:-}" ]] && readonly MSG_DATABASE_CREATED="✅ Database created successfully"
    [[ -z "${MSG_BACKUP_SUCCESS:-}" ]] && readonly MSG_BACKUP_SUCCESS="✅ Backup completed successfully"
    [[ -z "${MSG_RESTORE_SUCCESS:-}" ]] && readonly MSG_RESTORE_SUCCESS="✅ Restore completed successfully"
    [[ -z "${MSG_MIGRATION_SUCCESS:-}" ]] && readonly MSG_MIGRATION_SUCCESS="✅ Migrations applied successfully"
    
    # Status messages
    [[ -z "${MSG_HEALTHY:-}" ]] && readonly MSG_HEALTHY="✅ PostgreSQL instance is healthy"
    [[ -z "${MSG_RUNNING:-}" ]] && readonly MSG_RUNNING="✅ PostgreSQL instance is running"
    [[ -z "${MSG_INSTANCE_AVAILABLE:-}" ]] && readonly MSG_INSTANCE_AVAILABLE="PostgreSQL instance available at"
    
    # Info messages
    [[ -z "${MSG_CHECKING_DOCKER:-}" ]] && readonly MSG_CHECKING_DOCKER="Checking Docker availability..."
    [[ -z "${MSG_CHECKING_PORTS:-}" ]] && readonly MSG_CHECKING_PORTS="Checking port availability..."
    [[ -z "${MSG_FINDING_PORT:-}" ]] && readonly MSG_FINDING_PORT="Finding available port for instance..."
    [[ -z "${MSG_PULLING_IMAGE:-}" ]] && readonly MSG_PULLING_IMAGE="Pulling PostgreSQL Docker image..."
    [[ -z "${MSG_CREATING_DIRECTORIES:-}" ]] && readonly MSG_CREATING_DIRECTORIES="Creating instance directories..."
    [[ -z "${MSG_STARTING_CONTAINER:-}" ]] && readonly MSG_STARTING_CONTAINER="Starting PostgreSQL container..."
    [[ -z "${MSG_WAITING_STARTUP:-}" ]] && readonly MSG_WAITING_STARTUP="Waiting for PostgreSQL to start..."
    [[ -z "${MSG_CREATING_DATABASE:-}" ]] && readonly MSG_CREATING_DATABASE="Creating database..."
    [[ -z "${MSG_APPLYING_TEMPLATE:-}" ]] && readonly MSG_APPLYING_TEMPLATE="Applying configuration template..."
    [[ -z "${MSG_LISTING_INSTANCES:-}" ]] && readonly MSG_LISTING_INSTANCES="Listing PostgreSQL instances..."
    
    # Warning messages
    [[ -z "${MSG_PORT_IN_USE:-}" ]] && readonly MSG_PORT_IN_USE="Port is already in use"
    [[ -z "${MSG_INSTANCE_EXISTS:-}" ]] && readonly MSG_INSTANCE_EXISTS="PostgreSQL instance already exists"
    [[ -z "${MSG_LOW_DISK_SPACE:-}" ]] && readonly MSG_LOW_DISK_SPACE="Low disk space detected"
    [[ -z "${MSG_MAX_INSTANCES:-}" ]] && readonly MSG_MAX_INSTANCES="Maximum number of instances reached"
    [[ -z "${MSG_INSTANCE_STOPPED:-}" ]] && readonly MSG_INSTANCE_STOPPED="Instance is stopped"
    
    # Error messages
    [[ -z "${MSG_DOCKER_NOT_FOUND:-}" ]] && readonly MSG_DOCKER_NOT_FOUND="Docker is not installed or not running"
    [[ -z "${MSG_INSTALL_FAILED:-}" ]] && readonly MSG_INSTALL_FAILED="PostgreSQL resource installation failed"
    [[ -z "${MSG_CREATE_FAILED:-}" ]] && readonly MSG_CREATE_FAILED="Failed to create PostgreSQL instance"
    [[ -z "${MSG_START_FAILED:-}" ]] && readonly MSG_START_FAILED="Failed to start PostgreSQL instance"
    [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]] && readonly MSG_HEALTH_CHECK_FAILED="PostgreSQL health check failed"
    [[ -z "${MSG_DATABASE_CREATE_FAILED:-}" ]] && readonly MSG_DATABASE_CREATE_FAILED="Failed to create database"
    [[ -z "${MSG_INSUFFICIENT_DISK:-}" ]] && readonly MSG_INSUFFICIENT_DISK="Insufficient disk space"
    [[ -z "${MSG_INSTANCE_NOT_FOUND:-}" ]] && readonly MSG_INSTANCE_NOT_FOUND="PostgreSQL instance not found"
    [[ -z "${MSG_NO_AVAILABLE_PORT:-}" ]] && readonly MSG_NO_AVAILABLE_PORT="No available ports in range"
    [[ -z "${MSG_BACKUP_FAILED:-}" ]] && readonly MSG_BACKUP_FAILED="Backup operation failed"
    [[ -z "${MSG_RESTORE_FAILED:-}" ]] && readonly MSG_RESTORE_FAILED="Restore operation failed"
    
    # Help messages
    [[ -z "${MSG_HELP_CONNECTION:-}" ]] && readonly MSG_HELP_CONNECTION="Use 'manage.sh --action connect --instance <name>' to get connection details"
    [[ -z "${MSG_HELP_LIST:-}" ]] && readonly MSG_HELP_LIST="Use 'manage.sh --action list' to see all instances"
    [[ -z "${MSG_HELP_CREATE:-}" ]] && readonly MSG_HELP_CREATE="Use 'manage.sh --action create --instance <name>' to create a new instance"
    [[ -z "${MSG_HELP_CREDENTIALS:-}" ]] && readonly MSG_HELP_CREDENTIALS="Credentials saved to instance configuration"
}

# Initialize messages when sourced
postgres::messages::init