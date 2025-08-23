#!/usr/bin/env bash
# Simplified System Mock for Resource Testing
# 
# Provides basic system command mocking for testing resource functionality
# Focus: systemctl, docker, network testing, basic system operations
#
# NAMING CONVENTIONS: This mock follows the standard naming pattern expected
# by the convention-based resource testing framework:
#
#   test_system_connection() : Test system network connectivity
#   test_system_health()     : Test system health via systemctl and docker
#   test_system_basic()      : Test basic system functionality
#
# This allows the testing framework to automatically discover and run these
# tests without requiring hardcoded knowledge of system specifics.

# Mock systemctl command
systemctl() {
    case "$1 $2" in
        # Service status checks
        "status docker"|"status docker.service")
            echo "● docker.service - Docker Application Container Engine"
            echo "   Loaded: loaded (/lib/systemd/system/docker.service; enabled; vendor preset: enabled)"
            echo "   Active: active (running) since Mon 2024-01-01 12:00:00 UTC; 1h ago"
            echo "     Docs: https://docs.docker.com"
            echo " Main PID: 1234 (dockerd)"
            echo "    Tasks: 8"
            echo "   Memory: 46.4M"
            echo "   CGroup: /system.slice/docker.service"
            echo "           └─1234 /usr/bin/dockerd -H fd:// --containerd=/run/containerd/containerd.sock"
            return 0
            ;;
        "status postgres"|"status postgresql"|"status postgresql.service")
            echo "● postgresql.service - PostgreSQL RDBMS"
            echo "   Loaded: loaded (/lib/systemd/system/postgresql.service; enabled; vendor preset: enabled)"
            echo "   Active: active (running) since Mon 2024-01-01 12:00:00 UTC; 1h ago"
            echo " Main PID: 5678 (postgres)"
            echo "    Tasks: 7 (limit: 4915)"
            echo "   Memory: 24.1M"
            echo "   CGroup: /system.slice/postgresql.service"
            echo "           └─5678 /usr/lib/postgresql/16/bin/postgres -D /var/lib/postgresql/16/main"
            return 0
            ;;
        "status redis"|"status redis-server"|"status redis.service")
            echo "● redis-server.service - Advanced key-value store"
            echo "   Loaded: loaded (/lib/systemd/system/redis-server.service; enabled; vendor preset: enabled)"
            echo "   Active: active (running) since Mon 2024-01-01 12:00:00 UTC; 1h ago"
            echo "     Docs: http://redis.io/documentation,"
            echo " Main PID: 9101 (redis-server)"
            echo "    Tasks: 4 (limit: 4915)"
            echo "   Memory: 2.3M"
            echo "   CGroup: /system.slice/redis-server.service"
            echo "           └─9101 /usr/bin/redis-server 127.0.0.1:6379"
            return 0
            ;;
        # Service control operations
        "start "*)
            local service_name="${2%.service}"
            echo "Started $service_name.service"
            return 0
            ;;
        "stop "*)
            local service_name="${2%.service}"
            echo "Stopped $service_name.service"
            return 0
            ;;
        "restart "*)
            local service_name="${2%.service}"
            echo "Restarted $service_name.service"
            return 0
            ;;
        "reload "*)
            local service_name="${2%.service}"
            echo "Reloaded $service_name.service"
            return 0
            ;;
        # Service state checks
        "is-active "*)
            echo "active"
            return 0
            ;;
        "is-enabled "*)
            echo "enabled"
            return 0
            ;;
        "is-failed "*)
            echo "active"
            return 1
            ;;
        # Enable/disable services
        "enable "*)
            local service_name="${2%.service}"
            echo "Created symlink /etc/systemd/system/multi-user.target.wants/$service_name.service → /lib/systemd/system/$service_name.service"
            return 0
            ;;
        "disable "*)
            local service_name="${2%.service}"
            echo "Removed /etc/systemd/system/multi-user.target.wants/$service_name.service"
            return 0
            ;;
        # Daemon operations
        "daemon-reload")
            echo "Daemon reload completed"
            return 0
            ;;
        # List operations
        "list-units"|"list-units --all")
            echo "UNIT                     LOAD   ACTIVE SUB     DESCRIPTION"
            echo "docker.service           loaded active running Docker Application Container Engine"
            echo "postgresql.service       loaded active running PostgreSQL RDBMS"
            echo "redis-server.service     loaded active running Advanced key-value store"
            return 0
            ;;
        # Default case
        *)
            echo "systemctl: mock command executed"
            return 0
            ;;
    esac
}

# Mock docker command
docker() {
    case "$1" in
        "ps"|"container ls")
            echo "CONTAINER ID   IMAGE      COMMAND                  CREATED       STATUS       PORTS                    NAMES"
            echo "1234567890ab   postgres   \"docker-entrypoint.s…\"   2 hours ago   Up 2 hours   0.0.0.0:5432->5432/tcp   vrooli-postgres"
            echo "abcdef123456   redis      \"docker-entrypoint.s…\"   2 hours ago   Up 2 hours   0.0.0.0:6379->6379/tcp   vrooli-redis"
            echo "567890abcdef   ollama     \"ollama serve\"           2 hours ago   Up 2 hours   0.0.0.0:11434->11434/tcp vrooli-ollama"
            return 0
            ;;
        "images")
            echo "REPOSITORY   TAG       IMAGE ID       CREATED        SIZE"
            echo "postgres     16        1234567890ab   2 weeks ago    379MB"
            echo "redis        7         abcdef123456   3 weeks ago    138MB"
            echo "ollama       latest    567890abcdef   1 week ago     1.2GB"
            return 0
            ;;
        "version")
            echo "Client: Docker Engine - Community"
            echo " Version:           24.0.0"
            echo " API version:       1.43"
            echo " Go version:        go1.20.10"
            echo " Git commit:        a61e2b4"
            echo " Built:             Mon Apr  1 17:35:09 2024"
            echo " OS/Arch:           linux/amd64"
            echo " Context:           default"
            echo ""
            echo "Server: Docker Engine - Community"
            echo " Engine:"
            echo "  Version:          24.0.0"
            echo "  API version:      1.43 (minimum version 1.12)"
            echo "  Go version:       go1.20.10"
            echo "  Git commit:       60bc101"
            echo "  Built:            Mon Apr  1 17:35:09 2024"
            echo "  OS/Arch:          linux/amd64"
            echo "  Experimental:     false"
            return 0
            ;;
        "info")
            echo "Client: Docker Engine - Community"
            echo " Version:    24.0.0"
            echo " Context:    default"
            echo " Debug Mode: false"
            echo ""
            echo "Server:"
            echo " Containers: 3"
            echo "  Running: 3"
            echo "  Paused: 0"
            echo "  Stopped: 0"
            echo " Images: 10"
            return 0
            ;;
        # Container operations
        "start"|"stop"|"restart"|"kill"|"rm"|"create"|"run")
            echo "docker: mock operation '$1' completed"
            return 0
            ;;
        *)
            echo "docker: mock command executed"
            return 0
            ;;
    esac
}

# Mock nc (netcat) for port testing
nc() {
    case "$*" in
        # Common port tests
        *"localhost 5432"*|*"127.0.0.1 5432"*|*"-z localhost 5432"*|*"-z 127.0.0.1 5432"*)
            return 0  # PostgreSQL port is open
            ;;
        *"localhost 6379"*|*"127.0.0.1 6379"*|*"-z localhost 6379"*|*"-z 127.0.0.1 6379"*)
            return 0  # Redis port is open
            ;;
        *"localhost 11434"*|*"127.0.0.1 11434"*|*"-z localhost 11434"*|*"-z 127.0.0.1 11434"*)
            return 0  # Ollama port is open
            ;;
        *"localhost 8080"*|*"127.0.0.1 8080"*|*"-z localhost 8080"*|*"-z 127.0.0.1 8080"*)
            return 0  # Common web port is open
            ;;
        # Other ports - simulate closed
        *)
            return 1  # Port closed/unreachable
            ;;
    esac
}

# Mock curl for basic web requests
mock_system_curl() {
    # Only handle basic health checks, pass through everything else
    case "$*" in
        *"/health"*|*"/api/health"*)
            echo "OK"
            return 0
            ;;
        *)
            # Pass through to real curl
            command curl "$@"
            return $?
            ;;
    esac
}

# Mock lsof for port checking
lsof() {
    case "$*" in
        *":5432"*)
            echo "COMMAND  PID   USER   FD   TYPE DEVICE SIZE/OFF NODE NAME"
            echo "postgres 1234 postgres  5u  IPv4  12345      0t0  TCP localhost:5432 (LISTEN)"
            return 0
            ;;
        *":6379"*)
            echo "COMMAND  PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME"
            echo "redis-se 5678 redis   6u  IPv4  23456      0t0  TCP localhost:6379 (LISTEN)"
            return 0
            ;;
        *":11434"*)
            echo "COMMAND PID  USER   FD   TYPE DEVICE SIZE/OFF NODE NAME"
            echo "ollama  9101 ollama  7u  IPv4  34567      0t0  TCP localhost:11434 (LISTEN)"
            return 0
            ;;
        *)
            return 1  # Port not found
            ;;
    esac
}

# Mock service command (SysV style)
service() {
    case "$1 $2" in
        "docker status")
            echo "● docker is running"
            return 0
            ;;
        "postgresql status"|"postgres status")
            echo "● postgresql is running"
            return 0
            ;;
        "redis status"|"redis-server status")
            echo "● redis-server is running"
            return 0
            ;;
        *" start")
            echo "$1 started"
            return 0
            ;;
        *" stop")
            echo "$1 stopped"
            return 0
            ;;
        *" restart")
            echo "$1 restarted"
            return 0
            ;;
        *)
            echo "service: mock operation completed"
            return 0
            ;;
    esac
}

# Test functions for system validation
test_systemctl_available() {
    which systemctl >/dev/null 2>&1 || systemctl --version >/dev/null 2>&1
    return $?
}

test_docker_available() {
    docker version >/dev/null 2>&1
    return $?
}

test_network_tools() {
    nc -z localhost 22 >/dev/null 2>&1
    return $?
}

test_automation_health() {
    local service_name="$1"
    
    # Generic health check for automation services
    case "$service_name" in
        "n8n")
            mock_system_curl -s "http://localhost:5678/health" >/dev/null 2>&1
            ;;
        "windmill")
            mock_system_curl -s "http://localhost:8000/api/version" >/dev/null 2>&1
            ;;
        "node-red")
            mock_system_curl -s "http://localhost:1880" >/dev/null 2>&1
            ;;
        *)
            # Generic port check
            nc -z localhost 8080 >/dev/null 2>&1
            ;;
    esac
    
    return $?
}

# Standard naming convention test functions for system mock
test_system_connection() {
    # Test basic system connectivity
    test_network_tools
    return $?
}

test_system_health() {
    # Test system health via systemctl
    test_systemctl_available && test_docker_available
    return $?
}

test_system_basic() {
    # Test basic system functionality
    test_network_tools && test_systemctl_available
    return $?
}

# Convention-based test functions for common automation resources
test_n8n_connection() {
    nc -z localhost 5678 >/dev/null 2>&1
    return $?
}

test_n8n_health() {
    test_automation_health "n8n"
    return $?
}

test_n8n_basic() {
    mock_system_curl -s "http://localhost:5678/health" >/dev/null 2>&1
    return $?
}

test_windmill_connection() {
    nc -z localhost 8000 >/dev/null 2>&1
    return $?
}

test_windmill_health() {
    test_automation_health "windmill"
    return $?
}

test_windmill_basic() {
    mock_system_curl -s "http://localhost:8000/api/version" >/dev/null 2>&1
    return $?
}

test_nodered_connection() {
    nc -z localhost 1880 >/dev/null 2>&1
    return $?
}

test_nodered_health() {
    test_automation_health "node-red"
    return $?
}

test_nodered_basic() {
    mock_system_curl -s "http://localhost:1880" >/dev/null 2>&1
    return $?
}

# Export mock functions
export -f systemctl docker nc lsof service mock_system_curl
export -f test_systemctl_available test_docker_available test_network_tools test_automation_health
export -f test_system_connection test_system_health test_system_basic
export -f test_n8n_connection test_n8n_health test_n8n_basic
export -f test_windmill_connection test_windmill_health test_windmill_basic
export -f test_nodered_connection test_nodered_health test_nodered_basic