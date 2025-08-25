#!/bin/bash

# BTCPay Server Status Functions

set -euo pipefail

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BTCPAY_STATUS_DIR="${APP_ROOT}/resources/btcpay/lib"

# Source common functions
source "${BTCPAY_STATUS_DIR}/common.sh"

# Main status function
btcpay::status() {
    local format="${1:-text}"
    
    # Gather status information
    local installed="false"
    local running="false"
    local health="unknown"
    local version="unknown"
    local container_status="not_found"
    local api_accessible="false"
    local postgres_running="false"
    
    if btcpay::is_installed; then
        installed="true"
        version=$(docker image inspect "${BTCPAY_IMAGE}" --format '{{index .RepoDigests 0}}' 2>/dev/null | cut -d@ -f2 | cut -c1-12 || echo "unknown")
    fi
    
    if btcpay::is_running; then
        running="true"
        container_status="running"
        health=$(btcpay::get_health)
        
        # Check API accessibility
        if curl -sf "${BTCPAY_BASE_URL}/api/v1/health" &>/dev/null; then
            api_accessible="true"
        fi
        
        # Check PostgreSQL
        if docker ps --filter "name=${BTCPAY_POSTGRES_CONTAINER}" --format "{{.Names}}" | grep -q "${BTCPAY_POSTGRES_CONTAINER}"; then
            postgres_running="true"
        fi
    fi
    
    # Build status data (ensure booleans are valid JSON)
    local status_data=$(jq -n \
        --arg name "btcpay" \
        --arg description "Self-hosted Bitcoin payment processor" \
        --arg category "execution" \
        --argjson installed "${installed}" \
        --argjson running "${running}" \
        --arg health "${health}" \
        --arg version "${version}" \
        --arg container_name "${BTCPAY_CONTAINER_NAME}" \
        --arg container_status "${container_status}" \
        --arg postgres_container "${BTCPAY_POSTGRES_CONTAINER}" \
        --argjson postgres_running "${postgres_running}" \
        --arg api_url "${BTCPAY_BASE_URL}" \
        --argjson api_accessible "${api_accessible}" \
        --arg port "${BTCPAY_PORT}" \
        --arg protocol "${BTCPAY_PROTOCOL}" \
        --arg data_dir "${BTCPAY_DATA_DIR}" \
        '{
            name: $name,
            description: $description,
            category: $category,
            installed: $installed,
            running: $running,
            health: $health,
            version: $version,
            container: {
                name: $container_name,
                status: $container_status,
                postgres: $postgres_container,
                postgres_running: $postgres_running
            },
            api: {
                url: $api_url,
                accessible: $api_accessible
            },
            config: {
                port: ($port | tonumber),
                protocol: $protocol,
                data_dir: $data_dir
            },
            status: $health
        }')
    
    # Format output
    if [[ "${format}" == "json" ]]; then
        echo "${status_data}"
    else
        btcpay_status_template "${status_data}"
    fi
}

# Status template for text output
btcpay_status_template() {
    local data="$1"
    
    # Check if jq is available
    if ! command -v jq &>/dev/null; then
        echo "[ERROR] jq is required for formatting"
        return 1
    fi
    
    cat <<EOF
[HEADER]  BTCPay Server Status Report

Description: Self-hosted Bitcoin payment processor
Category: execution

Basic Status:
  $(format::status_icon "$(echo "$data" | jq -r '.installed')") Installed: $(echo "$data" | jq -r '.installed')
  $(format::status_icon "$(echo "$data" | jq -r '.running')") Running: $(echo "$data" | jq -r '.running')
  $(format::status_icon "$(echo "$data" | jq -r 'if .health == "healthy" then "true" else "false" end')") Health: $(echo "$data" | jq -r '.health')

Container Info:
  📦 Name: $(echo "$data" | jq -r '.container.name')
  📊 Status: $(echo "$data" | jq -r '.container.status')
  🗄️ PostgreSQL: $(echo "$data" | jq -r 'if .container.postgres_running then "Running" else "Stopped" end')
  🖼️ Version: $(echo "$data" | jq -r '.version')

API Endpoint:
  🌐 URL: $(echo "$data" | jq -r '.api.url')
  $(format::status_icon "$(echo "$data" | jq -r '.api.accessible')") Accessible: $(echo "$data" | jq -r '.api.accessible')

Configuration:
  🔌 Port: $(echo "$data" | jq -r '.config.port')
  🔐 Protocol: $(echo "$data" | jq -r '.config.protocol')
  📁 Data Directory: $(echo "$data" | jq -r '.config.data_dir')

Status Message:
  $(format::status_icon "$(echo "$data" | jq -r 'if .health == "healthy" then "true" else "false" end')") $(echo "$data" | jq -r 'if .health == "healthy" then "BTCPay Server is healthy and ready" else "BTCPay Server needs attention" end')
EOF
}

# Export function
export -f btcpay::status