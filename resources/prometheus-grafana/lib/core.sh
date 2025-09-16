#!/usr/bin/env bash
# Core functionality for Prometheus + Grafana resource

set -euo pipefail

# Get script directory - use LIB_DIR to avoid conflicts
if [[ -z "${LIB_DIR:-}" ]]; then
    LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    readonly LIB_DIR
fi

# Get resource directory
if [[ -z "${RESOURCE_DIR:-}" ]]; then
    RESOURCE_DIR="$(dirname "$LIB_DIR")"
    readonly RESOURCE_DIR
fi

# Source configuration if not already sourced
if [[ -z "${RESOURCE_NAME:-}" ]]; then
    source "${RESOURCE_DIR}/config/defaults.sh"
fi

#######################################
# Install the monitoring stack
#######################################
install_resource() {
    echo "Installing Prometheus + Grafana monitoring stack..."
    
    # Create required directories
    mkdir -p "${RESOURCE_DIR}/data/prometheus"
    mkdir -p "${RESOURCE_DIR}/data/grafana"
    mkdir -p "${RESOURCE_DIR}/data/alertmanager"
    mkdir -p "${RESOURCE_DIR}/config/prometheus"
    mkdir -p "${RESOURCE_DIR}/config/grafana"
    mkdir -p "${RESOURCE_DIR}/config/alertmanager"
    mkdir -p "${RESOURCE_DIR}/logs"
    
    # Generate Prometheus configuration
    generate_prometheus_config
    
    # Generate Grafana configuration
    generate_grafana_config
    
    # Generate Alertmanager configuration
    generate_alertmanager_config
    
    # Create docker-compose file
    generate_docker_compose
    
    # Pull Docker images
    echo "Pulling Docker images..."
    docker pull "prom/prometheus:v${PROMETHEUS_VERSION}"
    docker pull "grafana/grafana:${GRAFANA_VERSION}"
    docker pull "prom/alertmanager:v${ALERTMANAGER_VERSION}"
    docker pull "prom/node-exporter:v${NODE_EXPORTER_VERSION}"
    
    echo "Installation complete!"
    echo "Grafana admin password: ${GRAFANA_ADMIN_PASSWORD}"
    echo "Save this password securely!"
}

#######################################
# Start the monitoring stack
#######################################
start_resource() {
    echo "Starting Prometheus + Grafana monitoring stack..."
    
    # Check if already running
    if is_running; then
        echo "Monitoring stack is already running"
        return 0
    fi
    
    # Validate configuration
    if ! validate_config; then
        echo "Configuration validation failed" >&2
        return 1
    fi
    
    # Start services
    cd "${RESOURCE_DIR}"
    docker-compose -f docker-compose.yml up -d
    
    # Wait for services to be healthy
    echo "Waiting for services to be healthy..."
    local timeout=60
    local elapsed=0
    
    while [[ $elapsed -lt $timeout ]]; do
        if check_health; then
            echo "All services are healthy!"
            return 0
        fi
        sleep 2
        ((elapsed += 2))
    done
    
    echo "Warning: Services did not become healthy within ${timeout} seconds"
    return 1
}

#######################################
# Stop the monitoring stack
#######################################
stop_resource() {
    echo "Stopping Prometheus + Grafana monitoring stack..."
    
    if ! is_running; then
        echo "Monitoring stack is not running"
        return 0
    fi
    
    cd "${RESOURCE_DIR}"
    docker-compose -f docker-compose.yml down
    
    echo "Monitoring stack stopped"
}

#######################################
# Restart the monitoring stack
#######################################
restart_resource() {
    echo "Restarting Prometheus + Grafana monitoring stack..."
    stop_resource
    sleep 2
    start_resource
}

#######################################
# Uninstall the monitoring stack
#######################################
uninstall_resource() {
    local keep_data="${1:-false}"
    
    echo "Uninstalling Prometheus + Grafana monitoring stack..."
    
    # Stop services if running
    stop_resource
    
    # Remove containers and networks
    cd "${RESOURCE_DIR}"
    docker-compose -f docker-compose.yml down -v
    
    # Remove data if requested
    if [[ "$keep_data" != "true" ]]; then
        echo "Removing data directories..."
        rm -rf "${RESOURCE_DIR}/data"
        rm -rf "${RESOURCE_DIR}/logs"
    else
        echo "Keeping data directories"
    fi
    
    echo "Uninstall complete"
}

#######################################
# Check if services are running
#######################################
is_running() {
    docker ps --format "{{.Names}}" | grep -q "prometheus-grafana" || return 1
}

#######################################
# Check health of all services
#######################################
check_health() {
    # Check Prometheus
    if ! timeout 5 curl -sf "http://localhost:${PROMETHEUS_PORT}/-/healthy" > /dev/null 2>&1; then
        return 1
    fi
    
    # Check Grafana
    if ! timeout 5 curl -sf "http://localhost:${GRAFANA_PORT}/api/health" > /dev/null 2>&1; then
        return 1
    fi
    
    # Check Alertmanager if enabled
    if [[ "$ENABLE_ALERTMANAGER" == "true" ]]; then
        if ! timeout 5 curl -sf "http://localhost:${ALERTMANAGER_PORT}/-/healthy" > /dev/null 2>&1; then
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Get service status
#######################################
get_status() {
    if is_running; then
        if check_health; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "stopped"
    fi
}

#######################################
# Show detailed status
#######################################
show_status() {
    local json_output="${1:-false}"
    
    local prometheus_status="stopped"
    local grafana_status="stopped"
    local alertmanager_status="stopped"
    
    if timeout 5 curl -sf "http://localhost:${PROMETHEUS_PORT}/-/healthy" > /dev/null 2>&1; then
        prometheus_status="healthy"
    elif docker ps | grep -q "prometheus-grafana-prometheus"; then
        prometheus_status="unhealthy"
    fi
    
    if timeout 5 curl -sf "http://localhost:${GRAFANA_PORT}/api/health" > /dev/null 2>&1; then
        grafana_status="healthy"
    elif docker ps | grep -q "prometheus-grafana-grafana"; then
        grafana_status="unhealthy"
    fi
    
    if [[ "$ENABLE_ALERTMANAGER" == "true" ]]; then
        if timeout 5 curl -sf "http://localhost:${ALERTMANAGER_PORT}/-/healthy" > /dev/null 2>&1; then
            alertmanager_status="healthy"
        elif docker ps | grep -q "prometheus-grafana-alertmanager"; then
            alertmanager_status="unhealthy"
        fi
    else
        alertmanager_status="disabled"
    fi
    
    if [[ "$json_output" == "true" ]]; then
        cat << EOF
{
  "overall": "$(get_status)",
  "services": {
    "prometheus": "$prometheus_status",
    "grafana": "$grafana_status",
    "alertmanager": "$alertmanager_status"
  },
  "urls": {
    "prometheus": "http://localhost:${PROMETHEUS_PORT}",
    "grafana": "http://localhost:${GRAFANA_PORT}",
    "alertmanager": "http://localhost:${ALERTMANAGER_PORT}"
  }
}
EOF
    else
        echo "Prometheus + Grafana Monitoring Stack Status"
        echo "============================================"
        echo "Overall: $(get_status)"
        echo ""
        echo "Services:"
        echo "  Prometheus:    $prometheus_status (http://localhost:${PROMETHEUS_PORT})"
        echo "  Grafana:       $grafana_status (http://localhost:${GRAFANA_PORT})"
        echo "  Alertmanager:  $alertmanager_status (http://localhost:${ALERTMANAGER_PORT})"
    fi
}

#######################################
# Show service logs
#######################################
show_logs() {
    local service="${1:-all}"
    local follow="${2:-false}"
    
    local follow_flag=""
    [[ "$follow" == "true" ]] && follow_flag="-f"
    
    cd "${RESOURCE_DIR}"
    
    case "$service" in
        prometheus)
            docker-compose logs $follow_flag prometheus
            ;;
        grafana)
            docker-compose logs $follow_flag grafana
            ;;
        alertmanager)
            docker-compose logs $follow_flag alertmanager
            ;;
        all)
            docker-compose logs $follow_flag
            ;;
        *)
            echo "Unknown service: $service"
            echo "Valid services: prometheus, grafana, alertmanager, all"
            return 1
            ;;
    esac
}

#######################################
# Show access credentials
#######################################
show_credentials() {
    echo "Grafana Access Credentials"
    echo "=========================="
    echo "URL: http://localhost:${GRAFANA_PORT}"
    echo "Username: ${GRAFANA_ADMIN_USER}"
    
    # Try to retrieve password from environment or config
    if [[ -n "${GRAFANA_ADMIN_PASSWORD:-}" ]]; then
        echo "Password: ${GRAFANA_ADMIN_PASSWORD}"
    elif [[ -f "${RESOURCE_DIR}/.grafana_password" ]]; then
        echo "Password: $(cat "${RESOURCE_DIR}/.grafana_password")"
    else
        echo "Password: Check installation output or docker-compose.yml"
    fi
    
    echo ""
    echo "Prometheus URL: http://localhost:${PROMETHEUS_PORT}"
    echo "Alertmanager URL: http://localhost:${ALERTMANAGER_PORT}"
}

#######################################
# Generate Prometheus configuration
#######################################
generate_prometheus_config() {
    cat > "${RESOURCE_DIR}/config/prometheus/prometheus.yml" << EOF
global:
  scrape_interval: ${PROMETHEUS_SCRAPE_INTERVAL}
  scrape_timeout: ${PROMETHEUS_SCRAPE_TIMEOUT}
  evaluation_interval: ${PROMETHEUS_EVALUATION_INTERVAL}

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - alertmanager:${ALERTMANAGER_PORT}

rule_files:
  - /etc/prometheus/rules/*.yml

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['node-exporter:9100']

  - job_name: 'grafana'
    static_configs:
      - targets: ['grafana:3000']

  # Docker service discovery
  - job_name: 'docker-containers'
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
    relabel_configs:
      - source_labels: [__meta_docker_container_label_prometheus_job]
        target_label: job
      - source_labels: [__meta_docker_container_label_prometheus_port]
        target_label: __address__
        replacement: \${1}:9090
EOF
}

#######################################
# Generate Grafana configuration
#######################################
generate_grafana_config() {
    # Save password for later reference
    echo "${GRAFANA_ADMIN_PASSWORD}" > "${RESOURCE_DIR}/.grafana_password"
    chmod 600 "${RESOURCE_DIR}/.grafana_password"
    
    # Create datasource provisioning
    mkdir -p "${RESOURCE_DIR}/config/grafana/provisioning/datasources"
    cat > "${RESOURCE_DIR}/config/grafana/provisioning/datasources/prometheus.yml" << EOF
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
EOF

    # Create dashboard provisioning
    mkdir -p "${RESOURCE_DIR}/config/grafana/provisioning/dashboards"
    cat > "${RESOURCE_DIR}/config/grafana/provisioning/dashboards/default.yml" << EOF
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
EOF
}

#######################################
# Generate Alertmanager configuration
#######################################
generate_alertmanager_config() {
    cat > "${RESOURCE_DIR}/config/alertmanager/alertmanager.yml" << EOF
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:5001/webhook'
        send_resolved: true
EOF
}

#######################################
# Generate docker-compose.yml
#######################################
generate_docker_compose() {
    cat > "${RESOURCE_DIR}/docker-compose.yml" << EOF
services:
  prometheus:
    image: prom/prometheus:v${PROMETHEUS_VERSION}
    container_name: prometheus-grafana-prometheus
    user: "nobody"
    ports:
      - "${PROMETHEUS_PORT}:9090"
    volumes:
      - ./config/prometheus:/etc/prometheus:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=${PROMETHEUS_RETENTION}'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--web.enable-lifecycle'
    restart: unless-stopped
    networks:
      - monitoring
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:9090/-/healthy"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s

  grafana:
    image: grafana/grafana:${GRAFANA_VERSION}
    container_name: prometheus-grafana-grafana
    user: "472"
    ports:
      - "${GRAFANA_PORT}:3000"
    volumes:
      - grafana-data:/var/lib/grafana
      - ./config/grafana/provisioning:/etc/grafana/provisioning:ro
    environment:
      - GF_SECURITY_ADMIN_USER=${GRAFANA_ADMIN_USER}
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
      - GF_SERVER_HTTP_PORT=3000
      - GF_AUTH_ANONYMOUS_ENABLED=${GRAFANA_ANONYMOUS_ENABLED}
      - GF_AUTH_BASIC_ENABLED=${GRAFANA_AUTH_BASIC_ENABLED}
      - GF_PATHS_DATA=/var/lib/grafana
      - GF_PATHS_LOGS=/var/log/grafana
      - GF_PATHS_PLUGINS=/var/lib/grafana/plugins
      - GF_PATHS_PROVISIONING=/etc/grafana/provisioning
    restart: unless-stopped
    networks:
      - monitoring
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3000/api/health"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s

  alertmanager:
    image: prom/alertmanager:v${ALERTMANAGER_VERSION}
    container_name: prometheus-grafana-alertmanager
    user: "nobody"
    ports:
      - "${ALERTMANAGER_PORT}:9093"
    volumes:
      - ./config/alertmanager:/etc/alertmanager:ro
      - alertmanager-data:/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
      - '--storage.path=/alertmanager'
    restart: unless-stopped
    networks:
      - monitoring
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:9093/-/healthy"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s

  node-exporter:
    image: prom/node-exporter:v${NODE_EXPORTER_VERSION}
    container_name: prometheus-grafana-node-exporter
    ports:
      - "${NODE_EXPORTER_PORT}:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.rootfs=/rootfs'
      - '--path.sysfs=/host/sys'
      - '--collector.filesystem.mount-points-exclude=^/(sys|proc|dev|host|etc)(\$\$|/)'
    restart: unless-stopped
    networks:
      - monitoring
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:9100/metrics"]
      interval: 10s
      timeout: 5s
      retries: 3

networks:
  monitoring:
    driver: bridge

volumes:
  prometheus-data:
    driver: local
  grafana-data:
    driver: local
  alertmanager-data:
    driver: local
EOF
}

#######################################
# Content management functions
#######################################
list_content() {
    local content_type="${1:-metrics}"
    
    case "$content_type" in
        metrics)
            curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/label/__name__/values" | jq -r '.data[]' | head -20
            ;;
        dashboards)
            curl -s "http://localhost:${GRAFANA_PORT}/api/search" | jq -r '.[] | .title'
            ;;
        alerts)
            curl -s "http://localhost:${ALERTMANAGER_PORT}/api/v1/alerts" | jq -r '.[] | .labels.alertname'
            ;;
        *)
            echo "Unknown content type: $content_type"
            echo "Valid types: metrics, dashboards, alerts"
            return 1
            ;;
    esac
}

#######################################
# Add content
#######################################
add_content() {
    local content_type="${1:-}"
    local file="${2:-}"
    
    if [[ -z "$content_type" || -z "$file" ]]; then
        echo "Usage: add_content <type> <file>"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "File not found: $file"
        return 1
    fi
    
    case "$content_type" in
        dashboard)
            # Import dashboard to Grafana
            local dashboard_json=$(cat "$file")
            curl -X POST \
                -H "Content-Type: application/json" \
                -d "{\"dashboard\": $dashboard_json, \"overwrite\": true}" \
                "http://admin:${GRAFANA_ADMIN_PASSWORD}@localhost:${GRAFANA_PORT}/api/dashboards/db"
            ;;
        alert)
            # Add alert rule to Prometheus
            cp "$file" "${RESOURCE_DIR}/config/prometheus/rules/"
            # Reload Prometheus configuration
            curl -X POST "http://localhost:${PROMETHEUS_PORT}/-/reload"
            ;;
        *)
            echo "Unknown content type: $content_type"
            echo "Valid types: dashboard, alert"
            return 1
            ;;
    esac
}

#######################################
# Get content
#######################################
get_content() {
    local item="${1:-}"
    
    if [[ -z "$item" ]]; then
        echo "Usage: get_content <metric_name>"
        return 1
    fi
    
    # Query Prometheus for the metric
    curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/query?query=${item}" | jq '.'
}

#######################################
# Remove content
#######################################
remove_content() {
    local item="${1:-}"
    
    if [[ -z "$item" ]]; then
        echo "Usage: remove_content <dashboard_name>"
        return 1
    fi
    
    # This would remove a dashboard from Grafana
    # Implementation depends on specific requirements
    echo "Remove functionality not yet implemented"
}

#######################################
# Execute PromQL query
#######################################
execute_query() {
    local query="${1:-}"
    
    if [[ -z "$query" ]]; then
        echo "Usage: execute_query <promql_query>"
        return 1
    fi
    
    curl -s "http://localhost:${PROMETHEUS_PORT}/api/v1/query?query=${query}" | jq '.'
}