#!/usr/bin/env bash
################################################################################
# Qdrant Monitoring Dashboard
# 
# Comprehensive monitoring dashboard showing system health, performance,
# error rates, and operational metrics in a user-friendly format
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"
source "${APP_ROOT}/resources/qdrant/lib/error-handler.sh"

# Configuration
QDRANT_URL="http://localhost:6333"
REFRESH_INTERVAL="${DASHBOARD_REFRESH:-10}"
CONTAINER_NAME="${QDRANT_CONTAINER:-qdrant}"

#######################################
# Get system uptime and basic stats
#######################################
get_system_stats() {
    local container_stats
    container_stats=$(docker stats "$CONTAINER_NAME" --no-stream --format "table {{.MemUsage}}\t{{.MemPerc}}\t{{.CPUPerc}}" 2>/dev/null | tail -n1 || echo "N/A N/A N/A")
    
    local mem_usage=$(echo "$container_stats" | awk '{print $1}')
    local mem_percent=$(echo "$container_stats" | awk '{print $2}')
    local cpu_percent=$(echo "$container_stats" | awk '{print $3}')
    
    # Get container uptime
    local uptime
    uptime=$(docker inspect "$CONTAINER_NAME" --format '{{.State.StartedAt}}' 2>/dev/null | xargs -I {} date -d {} +%s 2>/dev/null || echo "0")
    local current_time=$(date +%s)
    local uptime_seconds=$((current_time - uptime))
    local uptime_formatted=$(printf "%dd %02dh %02dm" $((uptime_seconds/86400)) $((uptime_seconds%86400/3600)) $((uptime_seconds%3600/60)))
    
    echo "MEMORY_USAGE=$mem_usage"
    echo "MEMORY_PERCENT=$mem_percent"  
    echo "CPU_PERCENT=$cpu_percent"
    echo "UPTIME=$uptime_formatted"
}

#######################################
# Get Qdrant API performance stats
#######################################
get_api_stats() {
    local start_time=$(date +%s%3N)
    
    # Test basic API response time
    local api_response
    if api_response=$(timeout 5 curl -s "$QDRANT_URL/" 2>/dev/null); then
        local end_time=$(date +%s%3N)
        local response_time=$((end_time - start_time))
        
        local version
        version=$(echo "$api_response" | jq -r '.version // "unknown"' 2>/dev/null || echo "unknown")
        
        echo "API_STATUS=healthy"
        echo "API_RESPONSE_TIME=${response_time}ms"
        echo "QDRANT_VERSION=$version"
    else
        echo "API_STATUS=unhealthy"
        echo "API_RESPONSE_TIME=timeout"
        echo "QDRANT_VERSION=unknown"
    fi
}

#######################################
# Get collections statistics
#######################################
get_collections_stats() {
    local collections_response
    if collections_response=$(timeout 10 curl -s "$QDRANT_URL/collections" 2>/dev/null); then
        local collection_count
        collection_count=$(echo "$collections_response" | jq '.result.collections | length' 2>/dev/null || echo "0")
        
        local total_points=0
        local total_vectors=0
        local collections_status="healthy"
        
        # Get detailed stats for each collection
        while IFS= read -r collection_name; do
            [[ -n "$collection_name" && "$collection_name" != "null" ]] || continue
            
            local collection_info
            if collection_info=$(timeout 5 curl -s "$QDRANT_URL/collections/$collection_name" 2>/dev/null); then
                local points_count
                points_count=$(echo "$collection_info" | jq -r '.result.points_count // 0' 2>/dev/null || echo "0")
                local vectors_count  
                vectors_count=$(echo "$collection_info" | jq -r '.result.indexed_vectors_count // 0' 2>/dev/null || echo "0")
                local status
                status=$(echo "$collection_info" | jq -r '.result.status // "unknown"' 2>/dev/null || echo "unknown")
                
                total_points=$((total_points + points_count))
                total_vectors=$((total_vectors + vectors_count))
                
                if [[ "$status" != "green" ]]; then
                    collections_status="degraded"
                fi
            else
                collections_status="degraded"
            fi
        done <<< "$(echo "$collections_response" | jq -r '.result.collections[].name' 2>/dev/null)"
        
        echo "COLLECTION_COUNT=$collection_count"
        echo "TOTAL_POINTS=$total_points"
        echo "TOTAL_VECTORS=$total_vectors"
        echo "COLLECTIONS_STATUS=$collections_status"
    else
        echo "COLLECTION_COUNT=0"
        echo "TOTAL_POINTS=0"
        echo "TOTAL_VECTORS=0"
        echo "COLLECTIONS_STATUS=unreachable"
    fi
}

#######################################
# Get error statistics
#######################################
get_error_stats() {
    local error_metrics_file="$HOME/.qdrant/error-metrics.json"
    
    if [[ -f "$error_metrics_file" ]]; then
        local metrics
        metrics=$(cat "$error_metrics_file" 2>/dev/null || echo '{}')
        
        local embedding_errors
        embedding_errors=$(echo "$metrics" | jq -r '.embedding_errors // 0')
        local search_errors
        search_errors=$(echo "$metrics" | jq -r '.search_errors // 0')
        local api_errors
        api_errors=$(echo "$metrics" | jq -r '.api_errors // 0')
        local total_errors=$((embedding_errors + search_errors + api_errors))
        
        echo "EMBEDDING_ERRORS=$embedding_errors"
        echo "SEARCH_ERRORS=$search_errors" 
        echo "API_ERRORS=$api_errors"
        echo "TOTAL_ERRORS=$total_errors"
        
        # Error rate calculation (approximate)
        local operations_estimate=$((total_points / 10))  # Rough estimate
        if [[ $operations_estimate -gt 0 ]]; then
            local error_rate=$(( (total_errors * 100) / operations_estimate ))
            echo "ERROR_RATE=${error_rate}%"
        else
            echo "ERROR_RATE=0%"
        fi
    else
        echo "EMBEDDING_ERRORS=0"
        echo "SEARCH_ERRORS=0"
        echo "API_ERRORS=0"
        echo "TOTAL_ERRORS=0"
        echo "ERROR_RATE=0%"
    fi
    
    # Count failed operations in queue
    local failed_ops_count
    failed_ops_count=$(find "$HOME/.qdrant/failed-operations" -name "*.json" -type f 2>/dev/null | wc -l || echo "0")
    echo "FAILED_OPERATIONS=$failed_ops_count"
}

#######################################
# Get embeddings system stats
#######################################
get_embeddings_stats() {
    local embeddings_identity_file="$HOME/.qdrant/identity/vrooli-main.json"
    
    if [[ -f "$embeddings_identity_file" ]]; then
        local identity
        identity=$(cat "$embeddings_identity_file" 2>/dev/null || echo '{}')
        
        local total_embeddings
        total_embeddings=$(echo "$identity" | jq -r '.stats.total_embeddings // 0')
        local last_refresh
        last_refresh=$(echo "$identity" | jq -r '.stats.last_refresh_duration // "Never"')
        
        echo "EMBEDDINGS_TOTAL=$total_embeddings"
        echo "LAST_REFRESH=$last_refresh"
    else
        echo "EMBEDDINGS_TOTAL=0"
        echo "LAST_REFRESH=Never"
    fi
    
    # Check if Ollama is available
    if curl -s http://localhost:11434/api/tags >/dev/null 2>&1; then
        echo "OLLAMA_STATUS=available"
        # Get embedding models count
        local models_count
        models_count=$(curl -s http://localhost:11434/api/tags 2>/dev/null | jq '[.models[] | select(.name | contains("embed"))] | length' 2>/dev/null || echo "0")
        echo "EMBEDDING_MODELS=$models_count"
    else
        echo "OLLAMA_STATUS=unavailable"
        echo "EMBEDDING_MODELS=0"
    fi
}

#######################################
# Display status indicator
# Arguments:
#   $1 - Status value
#   $2 - Good status (optional, default: "healthy")
#######################################
status_indicator() {
    local status="$1"
    local good_status="${2:-healthy}"
    
    case "$status" in
        "$good_status"|"green"|"ok"|"available")
            echo "✅"
            ;;
        "degraded"|"yellow"|"warning")
            echo "⚠️"
            ;;
        "unhealthy"|"red"|"error"|"unavailable"|"unreachable")
            echo "❌"
            ;;
        *)
            echo "❓"
            ;;
    esac
}

#######################################
# Display the dashboard
#######################################
display_dashboard() {
    # Clear screen and show header
    clear
    echo "🚀 Qdrant Embeddings System - Live Dashboard"
    echo "=============================================="
    echo "Last updated: $(date '+%Y-%m-%d %H:%M:%S')"
    echo

    # Collect all stats
    local stats
    stats=$(mktemp)
    {
        get_system_stats
        get_api_stats
        get_collections_stats
        get_error_stats
        get_embeddings_stats
    } > "$stats"
    
    # Source the stats
    source "$stats"
    rm -f "$stats"
    
    # Display System Health
    echo "📊 System Health"
    echo "├─ Container: $(status_indicator "running") Running (${UPTIME})"
    echo "├─ API: $(status_indicator "${API_STATUS}") ${API_STATUS^} (${API_RESPONSE_TIME})"
    echo "├─ Collections: $(status_indicator "${COLLECTIONS_STATUS}") ${COLLECTIONS_STATUS^} (${COLLECTION_COUNT} total)"
    echo "└─ Ollama: $(status_indicator "${OLLAMA_STATUS}" "available") ${OLLAMA_STATUS^} (${EMBEDDING_MODELS} models)"
    echo
    
    # Display Performance Metrics
    echo "⚡ Performance Metrics"
    echo "├─ Memory: ${MEMORY_USAGE} (${MEMORY_PERCENT})"
    echo "├─ CPU: ${CPU_PERCENT}"
    echo "├─ API Response: ${API_RESPONSE_TIME}"
    echo "└─ Collections: ${COLLECTION_COUNT} collections, ${TOTAL_POINTS} points"
    echo
    
    # Display Data Statistics
    echo "📈 Data Statistics"
    echo "├─ Total Vectors: ${TOTAL_VECTORS}"
    echo "├─ Total Points: ${TOTAL_POINTS}"
    echo "├─ Total Embeddings: ${EMBEDDINGS_TOTAL}"
    echo "└─ Last Refresh: ${LAST_REFRESH}"
    echo
    
    # Display Error Statistics
    local error_status="healthy"
    if [[ ${TOTAL_ERRORS} -gt 0 ]]; then
        error_status="warning"
    fi
    if [[ ${TOTAL_ERRORS} -gt 100 ]]; then
        error_status="unhealthy"
    fi
    
    echo "🚨 Error Statistics"
    echo "├─ Overall Status: $(status_indicator "$error_status") Error Rate: ${ERROR_RATE}"
    echo "├─ Embedding Errors: ${EMBEDDING_ERRORS}"
    echo "├─ Search Errors: ${SEARCH_ERRORS}"
    echo "├─ API Errors: ${API_ERRORS}"
    echo "└─ Failed Operations: ${FAILED_OPERATIONS}"
    echo
    
    # Display Recent Activity
    echo "📋 Recent Activity"
    if [[ -f "$HOME/.qdrant/health.log" ]]; then
        echo "├─ Last Health Check:"
        tail -n 3 "$HOME/.qdrant/health.log" 2>/dev/null | sed 's/^/│  /' || echo "│  No recent health checks"
    else
        echo "├─ No health check logs available"
    fi
    echo
    
    # Display Quick Actions
    echo "🔧 Quick Actions"
    echo "├─ [r] Refresh now"
    echo "├─ [h] Health check"
    echo "├─ [s] Show statistics"  
    echo "├─ [e] Error details"
    echo "├─ [c] Clear screen"
    echo "└─ [q] Quit"
    echo
    echo "Refreshing in ${REFRESH_INTERVAL}s... (Press any key for action menu)"
}

#######################################
# Show detailed statistics
#######################################
show_detailed_stats() {
    clear
    echo "📊 Detailed Qdrant Statistics"
    echo "============================="
    echo
    
    # HTTP client stats
    if command -v http_client::stats >/dev/null 2>&1; then
        http_client::stats
        echo
    fi
    
    # Error handler stats
    error_handler::get_stats
    echo
    
    echo "Press any key to return to dashboard..."
    read -n 1 -s
}

#######################################
# Show error details
#######################################
show_error_details() {
    clear
    echo "🚨 Error Analysis"
    echo "================="
    echo
    
    # Show recent errors
    if [[ -f "$HOME/.qdrant/errors.log" ]]; then
        echo "Recent Errors (last 20):"
        tail -n 20 "$HOME/.qdrant/errors.log" | jq -r '"\(.timestamp) [\(.type | ascii_upcase)] \(.message)"' 2>/dev/null || \
            tail -n 20 "$HOME/.qdrant/errors.log"
    else
        echo "No error log found"
    fi
    
    echo
    echo "Failed Operations:"
    local failed_count
    failed_count=$(find "$HOME/.qdrant/failed-operations" -name "*.json" -type f 2>/dev/null | wc -l)
    
    if [[ $failed_count -gt 0 ]]; then
        echo "Found $failed_count failed operations in queue"
        find "$HOME/.qdrant/failed-operations" -name "*.json" -type f -exec basename {} \; | head -10
        if [[ $failed_count -gt 10 ]]; then
            echo "... and $((failed_count - 10)) more"
        fi
    else
        echo "No failed operations in queue"
    fi
    
    echo
    echo "Actions:"
    echo "[r] Retry failed operations"
    echo "[c] Clean old errors"
    echo "[b] Back to dashboard"
    echo
    echo -n "Choose action: "
    read -n 1 -s action
    
    case "$action" in
        r) error_handler::retry_failed_operations; echo; echo "Press any key to continue..."; read -n 1 -s ;;
        c) error_handler::cleanup; echo; echo "Press any key to continue..."; read -n 1 -s ;;
    esac
}

#######################################
# Interactive dashboard loop
#######################################
run_interactive_dashboard() {
    echo "🚀 Starting Qdrant Monitoring Dashboard..."
    echo "Press Ctrl+C to exit"
    sleep 2
    
    while true; do
        display_dashboard
        
        # Read user input with timeout
        if read -n 1 -s -t "$REFRESH_INTERVAL" action; then
            case "$action" in
                r|R) continue ;;
                h|H) 
                    /home/matthalloran8/Vrooli/resources/qdrant/monitoring/health-check.sh
                    echo "Press any key to continue..."
                    read -n 1 -s
                    ;;
                s|S) show_detailed_stats ;;
                e|E) show_error_details ;;
                c|C) clear ;;
                q|Q) echo; echo "👋 Goodbye!"; exit 0 ;;
            esac
        fi
    done
}

#######################################
# Main function
#######################################
main() {
    local mode="${1:-interactive}"
    
    # Initialize error handler
    error_handler::init
    
    case "$mode" in
        interactive)
            run_interactive_dashboard
            ;;
        once)
            display_dashboard
            ;;
        stats)
            show_detailed_stats
            ;;
        errors)
            show_error_details
            ;;
        *)
            echo "Usage: $0 [interactive|once|stats|errors]"
            exit 1
            ;;
    esac
}

# Handle Ctrl+C gracefully
trap 'echo; echo "👋 Goodbye!"; exit 0' INT

main "$@"