#!/usr/bin/env bash
# Default configuration for Prometheus + Grafana resource

# Resource identification
export RESOURCE_NAME="prometheus-grafana"
export RESOURCE_TYPE="monitoring"
export RESOURCE_VERSION="1.0.0"

# Service ports (using non-conflicting ports)
export PROMETHEUS_PORT="${PROMETHEUS_PORT:-9090}"
export GRAFANA_PORT="${GRAFANA_PORT:-3030}"  # Non-standard to avoid conflicts
export ALERTMANAGER_PORT="${ALERTMANAGER_PORT:-9093}"
export NODE_EXPORTER_PORT="${NODE_EXPORTER_PORT:-9100}"

# Service configuration
export PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-2.45.0}"
export GRAFANA_VERSION="${GRAFANA_VERSION:-10.0.3}"
export ALERTMANAGER_VERSION="${ALERTMANAGER_VERSION:-0.26.0}"
export NODE_EXPORTER_VERSION="${NODE_EXPORTER_VERSION:-1.6.1}"

# Storage configuration
export PROMETHEUS_RETENTION="${PROMETHEUS_RETENTION:-15d}"
export PROMETHEUS_STORAGE_PATH="${PROMETHEUS_STORAGE_PATH:-./data/prometheus}"
export GRAFANA_STORAGE_PATH="${GRAFANA_STORAGE_PATH:-./data/grafana}"
export ALERTMANAGER_STORAGE_PATH="${ALERTMANAGER_STORAGE_PATH:-./data/alertmanager}"

# Scrape configuration
export PROMETHEUS_SCRAPE_INTERVAL="${PROMETHEUS_SCRAPE_INTERVAL:-15s}"
export PROMETHEUS_SCRAPE_TIMEOUT="${PROMETHEUS_SCRAPE_TIMEOUT:-10s}"
export PROMETHEUS_EVALUATION_INTERVAL="${PROMETHEUS_EVALUATION_INTERVAL:-15s}"

# Grafana configuration
export GRAFANA_ADMIN_USER="${GRAFANA_ADMIN_USER:-admin}"
export GRAFANA_ADMIN_PASSWORD="${GRAFANA_ADMIN_PASSWORD:-}"  # Generated if empty
export GRAFANA_ANONYMOUS_ENABLED="${GRAFANA_ANONYMOUS_ENABLED:-false}"
export GRAFANA_AUTH_BASIC_ENABLED="${GRAFANA_AUTH_BASIC_ENABLED:-true}"

# Docker configuration
export DOCKER_NETWORK="${DOCKER_NETWORK:-vrooli-network}"
export DOCKER_COMPOSE_FILE="${DOCKER_COMPOSE_FILE:-docker-compose.yml}"

# Resource limits
export PROMETHEUS_MEMORY_LIMIT="${PROMETHEUS_MEMORY_LIMIT:-1G}"
export GRAFANA_MEMORY_LIMIT="${GRAFANA_MEMORY_LIMIT:-512M}"
export ALERTMANAGER_MEMORY_LIMIT="${ALERTMANAGER_MEMORY_LIMIT:-256M}"

# Timeouts
export STARTUP_TIMEOUT="${STARTUP_TIMEOUT:-60}"
export SHUTDOWN_TIMEOUT="${SHUTDOWN_TIMEOUT:-30}"
export HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-5}"

# Logging
export LOG_LEVEL="${LOG_LEVEL:-info}"
export LOG_FILE="${LOG_FILE:-./logs/prometheus-grafana.log}"

# Feature flags
export ENABLE_ALERTMANAGER="${ENABLE_ALERTMANAGER:-true}"
export ENABLE_NODE_EXPORTER="${ENABLE_NODE_EXPORTER:-true}"
export ENABLE_SERVICE_DISCOVERY="${ENABLE_SERVICE_DISCOVERY:-true}"
export ENABLE_RECORDING_RULES="${ENABLE_RECORDING_RULES:-false}"

# Integration settings
export VROOLI_RESOURCES_PATH="${VROOLI_RESOURCES_PATH:-/home/matthalloran8/Vrooli/resources}"
export AUTO_DISCOVER_RESOURCES="${AUTO_DISCOVER_RESOURCES:-true}"

# Security settings
export ENABLE_TLS="${ENABLE_TLS:-false}"
export TLS_CERT_PATH="${TLS_CERT_PATH:-}"
export TLS_KEY_PATH="${TLS_KEY_PATH:-}"

# Alert configuration
export ALERT_SMTP_HOST="${ALERT_SMTP_HOST:-}"
export ALERT_SMTP_PORT="${ALERT_SMTP_PORT:-587}"
export ALERT_EMAIL_FROM="${ALERT_EMAIL_FROM:-prometheus@vrooli.local}"
export ALERT_EMAIL_TO="${ALERT_EMAIL_TO:-}"

# Backup configuration
export BACKUP_ENABLED="${BACKUP_ENABLED:-false}"
export BACKUP_PATH="${BACKUP_PATH:-./backups}"
export BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"

#######################################
# Generate secure password if not set
#######################################
if [[ -z "$GRAFANA_ADMIN_PASSWORD" ]]; then
    if command -v openssl &> /dev/null; then
        GRAFANA_ADMIN_PASSWORD=$(openssl rand -base64 12)
    else
        GRAFANA_ADMIN_PASSWORD="changeme$(date +%s)"
    fi
    export GRAFANA_ADMIN_PASSWORD
fi

#######################################
# Validate configuration
#######################################
validate_config() {
    local errors=0

    # Check ports
    for port in "$PROMETHEUS_PORT" "$GRAFANA_PORT" "$ALERTMANAGER_PORT" "$NODE_EXPORTER_PORT"; do
        if ! [[ "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 || "$port" -gt 65535 ]]; then
            echo "Error: Invalid port number: $port" >&2
            ((errors++))
        fi
    done

    # Check paths
    for path in "$PROMETHEUS_STORAGE_PATH" "$GRAFANA_STORAGE_PATH" "$ALERTMANAGER_STORAGE_PATH"; do
        if [[ ! -d "$(dirname "$path")" ]]; then
            echo "Warning: Parent directory for $path does not exist" >&2
        fi
    done

    return $errors
}