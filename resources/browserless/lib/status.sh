#!/usr/bin/env bash
#
# Browserless status functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_STATUS_DIR="${APP_ROOT}/resources/browserless/lib"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "$BROWSERLESS_STATUS_DIR/common.sh"

#######################################
# Get memory usage information from Docker stats
# Returns: JSON object with memory stats or empty object if unavailable
#######################################
function get_memory_stats() {
    if ! is_running; then
        echo "{}"
        return
    fi

    local stats
    stats=$(docker stats "$BROWSERLESS_CONTAINER_NAME" --no-stream --format "{{.MemUsage}}" 2>/dev/null || echo "")

    if [[ -z "$stats" ]]; then
        echo "{}"
        return
    fi

    # Parse "used / limit" format (e.g., "3.888GiB / 4GiB" or "93.68MiB / 6GiB")
    local used limit usage_pct
    used=$(echo "$stats" | awk '{print $1}')
    limit=$(echo "$stats" | awk '{print $3}')

    # Convert to bytes for accurate percentage calculation
    local used_bytes limit_bytes

    # Convert used to bytes
    if [[ "$used" =~ GiB ]]; then
        used_bytes=$(echo "$used" | sed 's/GiB//' | awk '{printf "%.0f", $1 * 1024 * 1024 * 1024}')
    elif [[ "$used" =~ MiB ]]; then
        used_bytes=$(echo "$used" | sed 's/MiB//' | awk '{printf "%.0f", $1 * 1024 * 1024}')
    elif [[ "$used" =~ KiB ]]; then
        used_bytes=$(echo "$used" | sed 's/KiB//' | awk '{printf "%.0f", $1 * 1024}')
    else
        used_bytes=$(echo "$used" | sed 's/B//' | awk '{printf "%.0f", $1}')
    fi

    # Convert limit to bytes
    if [[ "$limit" =~ GiB ]]; then
        limit_bytes=$(echo "$limit" | sed 's/GiB//' | awk '{printf "%.0f", $1 * 1024 * 1024 * 1024}')
    elif [[ "$limit" =~ MiB ]]; then
        limit_bytes=$(echo "$limit" | sed 's/MiB//' | awk '{printf "%.0f", $1 * 1024 * 1024}')
    elif [[ "$limit" =~ KiB ]]; then
        limit_bytes=$(echo "$limit" | sed 's/KiB//' | awk '{printf "%.0f", $1 * 1024}')
    else
        limit_bytes=$(echo "$limit" | sed 's/B//' | awk '{printf "%.0f", $1}')
    fi

    if [[ "$limit_bytes" -gt 0 ]]; then
        usage_pct=$(awk -v used="$used_bytes" -v limit="$limit_bytes" 'BEGIN {printf "%.2f", (used/limit)*100}')
    else
        usage_pct="0"
    fi

    cat <<EOF
{
  "used": "$used",
  "limit": "$limit",
  "usage_percent": $usage_pct
}
EOF
}

#######################################
# Check for Chrome crashes in recent logs
# Returns: Number of crashes found in last 100 lines
#######################################
function check_chrome_crashes() {
    if ! is_running; then
        echo "0"
        return
    fi

    local count
    count=$(docker logs "$BROWSERLESS_CONTAINER_NAME" --tail 100 2>&1 | grep -c "Page crashed!" 2>/dev/null || echo "0")
    # Ensure it's a single numeric value
    count=$(echo "$count" | tr -d '\n' | tr -d ' ')
    if [[ "$count" =~ ^[0-9]+$ ]]; then
        echo "$count"
    else
        echo "0"
    fi
}

#######################################
# Analyze overall health and provide diagnostics
# Returns: JSON object with health details
#######################################
function get_health_diagnostics() {
    local memory_stats chrome_crashes pressure_data
    local warnings=()
    local status="healthy"

    memory_stats=$(get_memory_stats)
    chrome_crashes=$(check_chrome_crashes)
    pressure_data=$(get_metrics)

    # Ensure chrome_crashes is a valid number
    if ! [[ "$chrome_crashes" =~ ^[0-9]+$ ]]; then
        chrome_crashes="0"
    fi

    # Extract memory usage percentage
    local mem_pct
    mem_pct=$(echo "$memory_stats" | jq -r '.usage_percent // 0' 2>/dev/null || echo "0")

    # Check for warning conditions
    if (( $(echo "$mem_pct > 80" | bc -l 2>/dev/null || echo "0") )); then
        warnings+=("High memory usage: ${mem_pct}%")
        status="warning"
    fi

    if [[ "$chrome_crashes" -gt 0 ]]; then
        warnings+=("$chrome_crashes Chrome crashes detected in recent logs")
        status="degraded"
    fi

    # Check queue pressure if available
    local queue_length
    queue_length=$(echo "$pressure_data" | jq -r '.queued // 0' 2>/dev/null || echo "0")
    if [[ "$queue_length" -gt 5 ]]; then
        warnings+=("High queue pressure: $queue_length requests queued")
        [[ "$status" == "healthy" ]] && status="warning"
    fi

    # Build warnings JSON array
    local warnings_json="[]"
    if [[ ${#warnings[@]} -gt 0 ]]; then
        warnings_json=$(printf '%s\n' "${warnings[@]}" | jq -R . | jq -s .)
    fi

    cat <<EOF
{
  "status": "$status",
  "memory": $memory_stats,
  "chrome_crashes": $chrome_crashes,
  "pressure": $pressure_data,
  "warnings": $warnings_json
}
EOF
}

function status() {
    local format_type="text"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format_type="${2:-text}"
                shift 2
                ;;
            json)
                format_type="json"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done

    # Collect status information
    local installed="No"
    local running="No"
    local health="Unknown"
    local container_status="not_found"
    local metrics="{}"
    local diagnostics="{}"

    # Check if Docker image exists
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "$BROWSERLESS_IMAGE"; then
        installed="Yes"
    fi

    # Check container status
    if is_running; then
        running="Yes"
        container_status=$(get_container_status)

        # Check health
        health=$(check_health)
        if [[ "$health" == "healthy" ]]; then
            health="Healthy"
            metrics=$(get_metrics)
            diagnostics=$(get_health_diagnostics)

            # Update health based on diagnostics
            local diag_status
            diag_status=$(echo "$diagnostics" | jq -r '.status // "healthy"')
            if [[ "$diag_status" != "healthy" ]]; then
                health="Degraded"
            fi
        else
            health="Unhealthy"
        fi
    fi
    
    # Format output based on requested format
    if [[ "$format_type" == "json" ]]; then
        cat <<EOF
{
  "name": "browserless",
  "description": "Headless Chrome automation service",
  "category": "agents",
  "installed": $([ "$installed" = "Yes" ] && echo "true" || echo "false"),
  "running": $([ "$running" = "Yes" ] && echo "true" || echo "false"),
  "healthy": $([ "$health" = "Healthy" ] && echo "true" || echo "false"),
  "status": "$health",
  "container": {
    "name": "$BROWSERLESS_CONTAINER_NAME",
    "status": "$container_status",
    "image": "$BROWSERLESS_IMAGE"
  },
  "endpoints": {
    "base": "http://localhost:${BROWSERLESS_PORT}",
    "docs": "http://localhost:${BROWSERLESS_PORT}/docs",
    "pressure": "http://localhost:${BROWSERLESS_PORT}/pressure",
    "screenshot": "http://localhost:${BROWSERLESS_PORT}/chrome/screenshot",
    "pdf": "http://localhost:${BROWSERLESS_PORT}/chrome/pdf",
    "content": "http://localhost:${BROWSERLESS_PORT}/chrome/content",
    "function": "http://localhost:${BROWSERLESS_PORT}/chrome/function"
  },
  "configuration": {
    "port": ${BROWSERLESS_PORT},
    "max_concurrent_sessions": ${BROWSERLESS_MAX_CONCURRENT_SESSIONS},
    "workspace_dir": "$BROWSERLESS_WORKSPACE_DIR"
  },
  "metrics": $metrics,
  "diagnostics": $diagnostics
}
EOF
    else
        echo "🌐 Browserless Status"
        echo "─────────────────────"
        echo "Installed: $installed"
        echo "Running: $running"
        echo "Health: $health"
        echo "Port: $BROWSERLESS_PORT"
        echo "Container: $BROWSERLESS_CONTAINER_NAME"
        echo "Status: $container_status"

        # Show diagnostics if available
        if [[ "$diagnostics" != "{}" ]]; then
            echo ""
            echo "📊 Health Diagnostics"
            echo "─────────────────────"

            # Memory stats
            local mem_used mem_limit mem_pct
            mem_used=$(echo "$diagnostics" | jq -r '.memory.used // "N/A"')
            mem_limit=$(echo "$diagnostics" | jq -r '.memory.limit // "N/A"')
            mem_pct=$(echo "$diagnostics" | jq -r '.memory.usage_percent // 0')

            if [[ "$mem_used" != "N/A" ]]; then
                echo "Memory: $mem_used / $mem_limit (${mem_pct}%)"

                # Color-code memory warning
                if (( $(echo "$mem_pct > 80" | bc -l 2>/dev/null || echo "0") )); then
                    echo "  ⚠️  High memory usage detected"
                fi
            fi

            # Chrome crashes
            local crashes
            crashes=$(echo "$diagnostics" | jq -r '.chrome_crashes // 0')
            if [[ "$crashes" -gt 0 ]]; then
                echo "Chrome Crashes: $crashes detected in recent logs"
                echo "  ⚠️  Browser instability - consider restarting browserless"
            fi

            # Queue pressure
            local queued running
            queued=$(echo "$diagnostics" | jq -r '.pressure.queued // 0')
            running=$(echo "$diagnostics" | jq -r '.pressure.running // 0')

            if [[ "$queued" != "0" || "$running" != "0" ]]; then
                echo "Sessions: $running running, $queued queued"

                if [[ "$queued" -gt 5 ]]; then
                    echo "  ⚠️  High queue pressure"
                fi
            fi

            # Overall warnings
            local warning_count
            warning_count=$(echo "$diagnostics" | jq -r '.warnings | length')
            if [[ "$warning_count" -gt 0 ]]; then
                echo ""
                echo "⚠️  Warnings:"
                echo "$diagnostics" | jq -r '.warnings[]' | sed 's/^/  • /'
            fi
        fi
    fi
}
