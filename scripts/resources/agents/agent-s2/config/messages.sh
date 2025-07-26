#!/usr/bin/env bash
# Agent S2 Messages Configuration
# All user-facing messages for consistent communication

#######################################
# Export message constants
# Idempotent - safe to call multiple times
#######################################
agents2::export_messages() {
    # Installation messages
    if [[ -z "${MSG_INSTALLING:-}" ]]; then
        readonly MSG_INSTALLING="🤖 Installing Agent S2 Autonomous Computer Interaction Service"
    fi
    if [[ -z "${MSG_ALREADY_INSTALLED:-}" ]]; then
        readonly MSG_ALREADY_INSTALLED="Agent S2 is already installed and running"
    fi
    if [[ -z "${MSG_USE_FORCE:-}" ]]; then
        readonly MSG_USE_FORCE="Use --force yes to reinstall"
    fi
    if [[ -z "${MSG_INSTALL_SUCCESS:-}" ]]; then
        readonly MSG_INSTALL_SUCCESS="✅ Agent S2 installation completed successfully"
    fi
    if [[ -z "${MSG_INSTALL_FAILED:-}" ]]; then
        readonly MSG_INSTALL_FAILED="❌ Agent S2 installation failed"
    fi

    # Docker messages
    if [[ -z "${MSG_DOCKER_REQUIRED:-}" ]]; then
        readonly MSG_DOCKER_REQUIRED="Docker is required but not available"
    fi
    if [[ -z "${MSG_BUILDING_IMAGE:-}" ]]; then
        readonly MSG_BUILDING_IMAGE="Building Agent S2 Docker image (this may take several minutes)..."
    fi
    if [[ -z "${MSG_BUILD_SUCCESS:-}" ]]; then
        readonly MSG_BUILD_SUCCESS="Docker image built successfully"
    fi
    if [[ -z "${MSG_BUILD_FAILED:-}" ]]; then
        readonly MSG_BUILD_FAILED="Failed to build Docker image"
    fi
    if [[ -z "${MSG_PULLING_BASE:-}" ]]; then
        readonly MSG_PULLING_BASE="Pulling base image..."
    fi

    # Directory messages
    if [[ -z "${MSG_CREATING_DIRS:-}" ]]; then
        readonly MSG_CREATING_DIRS="Creating Agent S2 directories..."
    fi
    if [[ -z "${MSG_DIRECTORIES_CREATED:-}" ]]; then
        readonly MSG_DIRECTORIES_CREATED="Directories created successfully"
    fi
    if [[ -z "${MSG_CREATE_DIRS_FAILED:-}" ]]; then
        readonly MSG_CREATE_DIRS_FAILED="Failed to create directories"
    fi

    # Container messages
    if [[ -z "${MSG_STARTING_CONTAINER:-}" ]]; then
        readonly MSG_STARTING_CONTAINER="Starting Agent S2 container..."
    fi
    if [[ -z "${MSG_CONTAINER_STARTED:-}" ]]; then
        readonly MSG_CONTAINER_STARTED="Agent S2 container started"
    fi
    if [[ -z "${MSG_CONTAINER_RUNNING:-}" ]]; then
        readonly MSG_CONTAINER_RUNNING="Agent S2 is already running"
    fi
    if [[ -z "${MSG_CONTAINER_NOT_RUNNING:-}" ]]; then
        readonly MSG_CONTAINER_NOT_RUNNING="Agent S2 is not running"
    fi
    if [[ -z "${MSG_STOPPING_CONTAINER:-}" ]]; then
        readonly MSG_STOPPING_CONTAINER="Stopping Agent S2 container..."
    fi
    if [[ -z "${MSG_CONTAINER_STOPPED:-}" ]]; then
        readonly MSG_CONTAINER_STOPPED="Agent S2 container stopped"
    fi

    # Network messages
    if [[ -z "${MSG_CREATING_NETWORK:-}" ]]; then
        readonly MSG_CREATING_NETWORK="Creating Docker network..."
    fi
    if [[ -z "${MSG_NETWORK_EXISTS:-}" ]]; then
        readonly MSG_NETWORK_EXISTS="Docker network already exists"
    fi
    if [[ -z "${MSG_NETWORK_CREATED:-}" ]]; then
        readonly MSG_NETWORK_CREATED="Docker network created"
    fi
    if [[ -z "${MSG_NETWORK_FAILED:-}" ]]; then
        readonly MSG_NETWORK_FAILED="Failed to create Docker network"
    fi

    # Health check messages
    if [[ -z "${MSG_WAITING_READY:-}" ]]; then
        readonly MSG_WAITING_READY="Waiting for Agent S2 to be ready..."
    fi
    if [[ -z "${MSG_SERVICE_HEALTHY:-}" ]]; then
        readonly MSG_SERVICE_HEALTHY="✅ Agent S2 is running and healthy"
    fi
    if [[ -z "${MSG_HEALTH_CHECK_FAILED:-}" ]]; then
        readonly MSG_HEALTH_CHECK_FAILED="⚠️  Agent S2 started but health check failed"
    fi
    if [[ -z "${MSG_STARTUP_TIMEOUT:-}" ]]; then
        readonly MSG_STARTUP_TIMEOUT="Agent S2 failed to start within timeout period"
    fi

    # Configuration messages
    if [[ -z "${MSG_CONFIG_UPDATE_SUCCESS:-}" ]]; then
        readonly MSG_CONFIG_UPDATE_SUCCESS="Vrooli resource configuration updated successfully"
    fi
    if [[ -z "${MSG_CONFIG_UPDATE_FAILED:-}" ]]; then
        readonly MSG_CONFIG_UPDATE_FAILED="Failed to update Vrooli resource configuration"
    fi

    # Status messages
    if [[ -z "${MSG_STATUS_HEADER:-}" ]]; then
        readonly MSG_STATUS_HEADER="📊 Agent S2 Status"
    fi
    if [[ -z "${MSG_SERVICE_RUNNING:-}" ]]; then
        readonly MSG_SERVICE_RUNNING="✅ Agent S2 is running"
    fi
    if [[ -z "${MSG_SERVICE_NOT_INSTALLED:-}" ]]; then
        readonly MSG_SERVICE_NOT_INSTALLED="Agent S2 is not installed"
    fi
    if [[ -z "${MSG_PORT_LISTENING:-}" ]]; then
        readonly MSG_PORT_LISTENING="✅ Service is listening on port"
    fi
    if [[ -z "${MSG_PORT_NOT_ACCESSIBLE:-}" ]]; then
        readonly MSG_PORT_NOT_ACCESSIBLE="⚠️  Port is not accessible"
    fi

    # Uninstall messages
    if [[ -z "${MSG_UNINSTALLING:-}" ]]; then
        readonly MSG_UNINSTALLING="🗑️  Uninstalling Agent S2"
    fi
    if [[ -z "${MSG_UNINSTALL_WARNING:-}" ]]; then
        readonly MSG_UNINSTALL_WARNING="This will remove Agent S2 containers and optionally all data"
    fi
    if [[ -z "${MSG_UNINSTALL_SUCCESS:-}" ]]; then
        readonly MSG_UNINSTALL_SUCCESS="✅ Agent S2 uninstalled successfully"
    fi
    if [[ -z "${MSG_UNINSTALL_CANCELLED:-}" ]]; then
        readonly MSG_UNINSTALL_CANCELLED="Uninstall cancelled"
    fi

    # Export all messages
    export MSG_INSTALLING MSG_ALREADY_INSTALLED MSG_USE_FORCE
    export MSG_INSTALL_SUCCESS MSG_INSTALL_FAILED
    export MSG_DOCKER_REQUIRED MSG_BUILDING_IMAGE MSG_BUILD_SUCCESS
    export MSG_BUILD_FAILED MSG_PULLING_BASE
    export MSG_CREATING_DIRS MSG_DIRECTORIES_CREATED MSG_CREATE_DIRS_FAILED
    export MSG_STARTING_CONTAINER MSG_CONTAINER_STARTED MSG_CONTAINER_RUNNING
    export MSG_CONTAINER_NOT_RUNNING MSG_STOPPING_CONTAINER MSG_CONTAINER_STOPPED
    export MSG_CREATING_NETWORK MSG_NETWORK_EXISTS MSG_NETWORK_CREATED
    export MSG_NETWORK_FAILED
    export MSG_WAITING_READY MSG_SERVICE_HEALTHY MSG_HEALTH_CHECK_FAILED
    export MSG_STARTUP_TIMEOUT
    export MSG_CONFIG_UPDATE_SUCCESS MSG_CONFIG_UPDATE_FAILED
    export MSG_STATUS_HEADER MSG_SERVICE_RUNNING MSG_SERVICE_NOT_INSTALLED
    export MSG_PORT_LISTENING MSG_PORT_NOT_ACCESSIBLE
    export MSG_UNINSTALLING MSG_UNINSTALL_WARNING MSG_UNINSTALL_SUCCESS
    export MSG_UNINSTALL_CANCELLED
}