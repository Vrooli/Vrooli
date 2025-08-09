#!/bin/bash
set -euo pipefail

# Brand Manager Monitoring Script
# Continuously monitors service health and integration activities

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$(dirname "$(dirname "$SCENARIO_DIR")")")"

# Source common utilities
source "$PROJECT_ROOT/scripts/lib/utils/var.sh" 
source "$PROJECT_ROOT/scripts/lib/utils/logging.sh"

# Configuration
MONITOR_INTERVAL="${MONITOR_INTERVAL:-30}"
POSTGRES_HOST="${POSTGRES_HOST:-postgres}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-brand_manager}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-password}"

N8N_ENDPOINT="${N8N_ENDPOINT:-http://n8n:5678}"
WINDMILL_ENDPOINT="${WINDMILL_ENDPOINT:-http://windmill:8000}"
OLLAMA_ENDPOINT="${OLLAMA_ENDPOINT:-http://ollama:11434}"
COMFYUI_ENDPOINT="${COMFYUI_ENDPOINT:-http://comfyui:8188}"
MINIO_ENDPOINT="${MINIO_ENDPOINT:-minio:9000}"

# Health check functions
check_service_health() {
    local service_name="$1"
    local endpoint="$2"
    local timeout="${3:-10}"
    
    if timeout "$timeout" curl -sf "$endpoint" >/dev/null 2>&1; then
        echo "✓ $service_name: healthy"
        return 0
    else
        echo "✗ $service_name: unhealthy"
        return 1
    fi
}

check_tcp_health() {
    local service_name="$1"
    local host="$2"
    local port="$3"
    local timeout="${4:-5}"
    
    if timeout "$timeout" bash -c "</dev/tcp/$host/$port" >/dev/null 2>&1; then
        echo "✓ $service_name: healthy"
        return 0
    else
        echo "✗ $service_name: unhealthy"
        return 1
    fi
}

# Database metrics
get_database_metrics() {
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    local total_brands
    local active_integrations
    local completed_integrations
    local failed_integrations
    
    total_brands=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM brands;" 2>/dev/null || echo "0")
    active_integrations=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM integration_requests WHERE status IN ('pending', 'processing');" 2>/dev/null || echo "0")
    completed_integrations=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM integration_requests WHERE status = 'completed';" 2>/dev/null || echo "0")
    failed_integrations=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "SELECT COUNT(*) FROM integration_requests WHERE status = 'failed';" 2>/dev/null || echo "0")
    
    echo "Database Metrics:"
    echo "  Total Brands: $(echo $total_brands | xargs)"
    echo "  Active Integrations: $(echo $active_integrations | xargs)" 
    echo "  Completed Integrations: $(echo $completed_integrations | xargs)"
    echo "  Failed Integrations: $(echo $failed_integrations | xargs)"
}

# AI service metrics
get_ai_metrics() {
    echo "AI Service Metrics:"
    
    # Ollama models
    local ollama_models
    ollama_models=$(curl -sf "$OLLAMA_ENDPOINT/api/tags" 2>/dev/null | jq -r '.models[].name' 2>/dev/null | wc -l || echo "0")
    echo "  Ollama Models Loaded: $ollama_models"
    
    # ComfyUI status
    local comfyui_status
    if curl -sf "$COMFYUI_ENDPOINT/system_stats" >/dev/null 2>&1; then
        comfyui_status="online"
    else
        comfyui_status="offline"
    fi
    echo "  ComfyUI Status: $comfyui_status"
}

# Integration monitoring
monitor_integrations() {
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    echo "Active Integrations:"
    
    local active_integrations
    active_integrations=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "
        SELECT ir.claude_session_id, b.name, ir.integration_type, ir.status, 
               EXTRACT(EPOCH FROM (NOW() - ir.created_at))/60 as minutes_running
        FROM integration_requests ir 
        JOIN brands b ON ir.brand_id = b.id 
        WHERE ir.status IN ('pending', 'processing')
        ORDER BY ir.created_at;
    " 2>/dev/null || echo "")
    
    if [ -z "$active_integrations" ] || [ "$active_integrations" = " " ]; then
        echo "  No active integrations"
    else
        echo "$active_integrations" | while read -r line; do
            if [ -n "$line" ] && [ "$line" != " " ]; then
                echo "  $line"
            fi
        done
    fi
}

# Resource usage monitoring
monitor_resources() {
    echo "Resource Usage:"
    
    # Disk usage
    local disk_usage
    disk_usage=$(df -h / | awk 'NR==2 {print $5}' | sed 's/%//')
    echo "  Disk Usage: ${disk_usage}%"
    
    # Memory usage
    if command -v free >/dev/null 2>&1; then
        local memory_usage
        memory_usage=$(free | awk 'NR==2{printf "%.1f", $3*100/$2}')
        echo "  Memory Usage: ${memory_usage}%"
    fi
    
    # Load average
    if [ -f /proc/loadavg ]; then
        local load_avg
        load_avg=$(cat /proc/loadavg | awk '{print $1}')
        echo "  Load Average: $load_avg"
    fi
}

# Alert on issues
check_alerts() {
    export PGPASSWORD="$POSTGRES_PASSWORD"
    
    # Check for stuck integrations (running > 10 minutes)
    local stuck_integrations
    stuck_integrations=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "
        SELECT COUNT(*) FROM integration_requests 
        WHERE status = 'processing' AND created_at < NOW() - INTERVAL '10 minutes';
    " 2>/dev/null | xargs || echo "0")
    
    if [ "$stuck_integrations" -gt 0 ]; then
        log_warn "Alert: $stuck_integrations integration(s) appear to be stuck (running > 10 minutes)"
    fi
    
    # Check for recent failures
    local recent_failures
    recent_failures=$(psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c "
        SELECT COUNT(*) FROM integration_requests 
        WHERE status = 'failed' AND created_at > NOW() - INTERVAL '1 hour';
    " 2>/dev/null | xargs || echo "0")
    
    if [ "$recent_failures" -gt 0 ]; then
        log_warn "Alert: $recent_failures integration(s) failed in the last hour"
    fi
}

# Main monitoring loop
run_monitoring() {
    log_info "Brand Manager monitoring started (interval: ${MONITOR_INTERVAL}s)"
    
    while true; do
        echo "=================================="
        echo "Brand Manager Health Check - $(date)"
        echo "=================================="
        
        # Service health checks
        echo "Service Health:"
        check_tcp_health "PostgreSQL" "$POSTGRES_HOST" "$POSTGRES_PORT"
        check_service_health "MinIO" "http://$MINIO_ENDPOINT/minio/health/ready"
        check_service_health "n8n" "$N8N_ENDPOINT/healthz"
        check_service_health "Windmill" "$WINDMILL_ENDPOINT/api/version"
        check_service_health "Ollama" "$OLLAMA_ENDPOINT/api/tags"
        check_service_health "ComfyUI" "$COMFYUI_ENDPOINT/system_stats"
        echo ""
        
        # Metrics
        get_database_metrics
        echo ""
        
        get_ai_metrics
        echo ""
        
        monitor_integrations
        echo ""
        
        monitor_resources
        echo ""
        
        # Check for alerts
        check_alerts
        
        echo "Next check in ${MONITOR_INTERVAL} seconds..."
        echo ""
        
        sleep "$MONITOR_INTERVAL"
    done
}

# Signal handling
cleanup() {
    log_info "Monitoring stopped by signal"
    exit 0
}

trap cleanup SIGTERM SIGINT

# Help function
show_help() {
    echo "Brand Manager Monitor"
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -i, --interval SECONDS    Set monitoring interval (default: 30)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Environment Variables:"
    echo "  MONITOR_INTERVAL         Monitoring interval in seconds"
    echo "  POSTGRES_HOST           PostgreSQL host"
    echo "  POSTGRES_PORT           PostgreSQL port"
    echo "  N8N_ENDPOINT            n8n endpoint URL"
    echo "  WINDMILL_ENDPOINT       Windmill endpoint URL"
    echo ""
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -i|--interval)
            MONITOR_INTERVAL="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Validate interval
if ! [[ "$MONITOR_INTERVAL" =~ ^[0-9]+$ ]] || [ "$MONITOR_INTERVAL" -lt 5 ]; then
    log_error "Invalid monitoring interval. Must be a number >= 5"
    exit 1
fi

# Run monitoring
run_monitoring