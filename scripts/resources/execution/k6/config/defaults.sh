#!/usr/bin/env bash
# K6 Configuration Defaults

# Service configuration
export K6_NAME="k6"
export K6_DISPLAY_NAME="K6 Load Testing"
export K6_CATEGORY="execution"
export K6_DESCRIPTION="Modern load testing tool with JavaScript scripting"

# Container configuration
export K6_CONTAINER_NAME="${K6_CONTAINER_NAME:-vrooli-k6}"
export K6_IMAGE="${K6_IMAGE:-grafana/k6:latest}"
export K6_PORT="${K6_PORT:-6565}"

# Runtime configuration
export K6_DATA_DIR="${K6_DATA_DIR:-/home/matthalloran8/.k6}"
export K6_SCRIPTS_DIR="${K6_DATA_DIR}/scripts"
export K6_RESULTS_DIR="${K6_DATA_DIR}/results"
export K6_CONFIG_DIR="${K6_DATA_DIR}/config"

# Test defaults
export K6_DEFAULT_VUS="${K6_DEFAULT_VUS:-10}"
export K6_DEFAULT_DURATION="${K6_DEFAULT_DURATION:-30s}"
export K6_DEFAULT_ITERATIONS="${K6_DEFAULT_ITERATIONS:-100}"

# Grafana Cloud configuration (optional)
export K6_GRAFANA_CLOUD_TOKEN="${K6_GRAFANA_CLOUD_TOKEN:-}"
export K6_GRAFANA_CLOUD_PROJECT_ID="${K6_GRAFANA_CLOUD_PROJECT_ID:-}"

# Export config function
k6::export_config() {
    export K6_NAME K6_DISPLAY_NAME K6_CATEGORY K6_DESCRIPTION
    export K6_CONTAINER_NAME K6_IMAGE K6_PORT
    export K6_DATA_DIR K6_SCRIPTS_DIR K6_RESULTS_DIR K6_CONFIG_DIR
    export K6_DEFAULT_VUS K6_DEFAULT_DURATION K6_DEFAULT_ITERATIONS
    export K6_GRAFANA_CLOUD_TOKEN K6_GRAFANA_CLOUD_PROJECT_ID
}