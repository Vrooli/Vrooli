#!/usr/bin/env bash
# K6 Core Functions

# Initialize K6 environment
k6::core::init() {
    # Create necessary directories
    mkdir -p "$K6_DATA_DIR"
    mkdir -p "$K6_SCRIPTS_DIR"
    mkdir -p "$K6_RESULTS_DIR"
    mkdir -p "$K6_CONFIG_DIR"
}

# Get K6 credentials for integration
k6::core::credentials() {
    local format="${1:-plain}"
    
    case "$format" in
        json)
            cat << EOF
{
  "type": "k6",
  "name": "K6 Load Testing",
  "port": $K6_PORT,
  "scripts_dir": "$K6_SCRIPTS_DIR",
  "results_dir": "$K6_RESULTS_DIR",
  "grafana_cloud_configured": $([ -n "$K6_GRAFANA_CLOUD_TOKEN" ] && echo "true" || echo "false")
}
EOF
            ;;
        pretty)
            echo "🔧 K6 Load Testing Credentials"
            echo "================================"
            echo "Port: $K6_PORT"
            echo "Scripts Directory: $K6_SCRIPTS_DIR"
            echo "Results Directory: $K6_RESULTS_DIR"
            if [[ -n "$K6_GRAFANA_CLOUD_TOKEN" ]]; then
                echo "Grafana Cloud: Configured"
            else
                echo "Grafana Cloud: Not configured"
            fi
            ;;
        *)
            echo "K6_PORT=$K6_PORT"
            echo "K6_SCRIPTS_DIR=$K6_SCRIPTS_DIR"
            echo "K6_RESULTS_DIR=$K6_RESULTS_DIR"
            ;;
    esac
}

# Check if K6 is installed (native)
k6::core::is_installed_native() {
    command -v k6 >/dev/null 2>&1
}

# Check if K6 is running (container)
k6::core::is_running() {
    docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${K6_CONTAINER_NAME}$"
}