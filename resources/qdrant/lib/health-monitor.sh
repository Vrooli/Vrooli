#!/usr/bin/env bash
# Enhanced Health Monitoring for Qdrant Vector Operations
# Provides detailed health metrics and performance monitoring

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
QDRANT_LIB_DIR="${APP_ROOT}/resources/qdrant/lib"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/api-client.sh"
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/search-optimizer.sh"

# Health status levels
readonly HEALTH_EXCELLENT="excellent"
readonly HEALTH_GOOD="good"
readonly HEALTH_DEGRADED="degraded"
readonly HEALTH_CRITICAL="critical"
readonly HEALTH_DOWN="down"

#######################################
# Get comprehensive health status
# Returns: JSON with detailed health metrics
#######################################
qdrant::health::get_status() {
    local health_data='{
        "timestamp": "'$(date -Iseconds)'",
        "service": "qdrant",
        "status": "unknown",
        "components": {}
    }'
    
    # Check basic connectivity
    local api_status="down"
    local api_latency_ms=0
    local api_start api_end
    
    api_start=$(date +%s%N)
    if timeout 5 curl -sf "http://localhost:${QDRANT_PORT}/" &>/dev/null; then
        api_status="up"
        api_end=$(date +%s%N)
        api_latency_ms=$(((api_end - api_start) / 1000000))
    fi
    
    health_data=$(echo "$health_data" | jq \
        --arg status "$api_status" \
        --argjson latency "$api_latency_ms" \
        '.components.api = {
            status: $status,
            latency_ms: $latency
        }')
    
    if [[ "$api_status" == "down" ]]; then
        health_data=$(echo "$health_data" | jq '.status = "down"')
        echo "$health_data"
        return 1
    fi
    
    # Get telemetry data
    local telemetry
    if telemetry=$(qdrant::api::get "/telemetry" 2>/dev/null); then
        local app_version commit
        app_version=$(echo "$telemetry" | jq -r '.result.app.version // "unknown"')
        commit=$(echo "$telemetry" | jq -r '.result.app.commit // "unknown"')
        
        health_data=$(echo "$health_data" | jq \
            --arg version "$app_version" \
            --arg commit "$commit" \
            '.components.version = {
                version: $version,
                commit: $commit
            }')
    fi
    
    # Check collections health
    local collections_health
    collections_health=$(qdrant::health::check_collections)
    health_data=$(echo "$health_data" | jq \
        --argjson collections "$collections_health" \
        '.components.collections = $collections')
    
    # Check memory usage
    local memory_health
    memory_health=$(qdrant::health::check_memory)
    health_data=$(echo "$health_data" | jq \
        --argjson memory "$memory_health" \
        '.components.memory = $memory')
    
    # Check search performance
    local search_health
    search_health=$(qdrant::health::check_search_performance)
    health_data=$(echo "$health_data" | jq \
        --argjson search "$search_health" \
        '.components.search = $search')
    
    # Calculate overall health status
    local overall_status
    overall_status=$(qdrant::health::calculate_overall_status "$health_data")
    health_data=$(echo "$health_data" | jq \
        --arg status "$overall_status" \
        '.status = $status')
    
    echo "$health_data"
    return 0
}

#######################################
# Check collections health
# Returns: JSON with collection metrics
#######################################
qdrant::health::check_collections() {
    local collections_health='{
        "status": "unknown",
        "total_collections": 0,
        "total_vectors": 0,
        "largest_collection": null,
        "issues": []
    }'
    
    local collections_json
    if ! collections_json=$(qdrant::api::get "/collections" 2>/dev/null); then
        collections_health=$(echo "$collections_health" | jq '.status = "error"')
        echo "$collections_health"
        return 1
    fi
    
    local collections
    collections=$(echo "$collections_json" | jq -r '.result.collections[].name // empty')
    
    local total_collections=0
    local total_vectors=0
    local largest_size=0
    local largest_name=""
    local issues=()
    
    while IFS= read -r collection; do
        [[ -z "$collection" ]] && continue
        ((total_collections++))
        
        local collection_info
        if collection_info=$(qdrant::api::get "/collections/${collection}" 2>/dev/null); then
            local vector_count indexed_count segments_count
            vector_count=$(echo "$collection_info" | jq -r '.result.vectors_count // 0')
            indexed_count=$(echo "$collection_info" | jq -r '.result.indexed_vectors_count // 0')
            segments_count=$(echo "$collection_info" | jq -r '.result.segments_count // 0')
            
            total_vectors=$((total_vectors + vector_count))
            
            if [[ $vector_count -gt $largest_size ]]; then
                largest_size=$vector_count
                largest_name=$collection
            fi
            
            # Check for indexing issues
            if [[ $vector_count -gt 0 ]] && [[ $indexed_count -eq 0 ]]; then
                issues+=("Collection '$collection' has no indexed vectors")
            elif [[ $vector_count -gt 0 ]]; then
                local index_percentage=$((indexed_count * 100 / vector_count))
                if [[ $index_percentage -lt 90 ]]; then
                    issues+=("Collection '$collection' only ${index_percentage}% indexed")
                fi
            fi
            
            # Check for excessive segments
            if [[ $segments_count -gt 10 ]]; then
                issues+=("Collection '$collection' has $segments_count segments (consider optimization)")
            fi
        fi
    done <<< "$collections"
    
    # Determine status based on findings
    local status="good"
    if [[ ${#issues[@]} -gt 0 ]]; then
        status="degraded"
    fi
    if [[ $total_collections -eq 0 ]]; then
        status="empty"
    fi
    
    collections_health=$(echo "$collections_health" | jq \
        --arg status "$status" \
        --argjson total_collections "$total_collections" \
        --argjson total_vectors "$total_vectors" \
        --arg largest_name "$largest_name" \
        --argjson largest_size "$largest_size" \
        --argjson issues "$(printf '%s\n' "${issues[@]}" | jq -Rs 'split("\n") | map(select(. != ""))')" \
        '.status = $status |
         .total_collections = $total_collections |
         .total_vectors = $total_vectors |
         .largest_collection = (if $largest_name != "" then {name: $largest_name, size: $largest_size} else null end) |
         .issues = $issues')
    
    echo "$collections_health"
    return 0
}

#######################################
# Check memory usage
# Returns: JSON with memory metrics
#######################################
qdrant::health::check_memory() {
    local memory_health='{
        "status": "unknown",
        "container_memory_mb": 0,
        "container_limit_mb": 0,
        "usage_percentage": 0
    }'
    
    # Get container memory stats
    local container_stats
    if container_stats=$(docker stats "${QDRANT_CONTAINER_NAME}" --no-stream --format "{{json .}}" 2>/dev/null); then
        local mem_usage mem_limit
        mem_usage=$(echo "$container_stats" | jq -r '.MemUsage' | cut -d'/' -f1 | sed 's/[^0-9.]//g')
        mem_limit=$(echo "$container_stats" | jq -r '.MemUsage' | cut -d'/' -f2 | sed 's/[^0-9.]//g')
        
        # Convert to MB (assuming they're in GiB if > 1000)
        if (( $(echo "$mem_usage < 1000" | bc -l) )); then
            mem_usage_mb=$(echo "$mem_usage" | bc)
        else
            mem_usage_mb=$(echo "$mem_usage / 1024" | bc)
        fi
        
        if (( $(echo "$mem_limit < 1000" | bc -l) )); then
            mem_limit_mb=$(echo "$mem_limit" | bc)
        else
            mem_limit_mb=$(echo "$mem_limit / 1024" | bc)
        fi
        
        local usage_pct=0
        if [[ "$mem_limit_mb" != "0" ]]; then
            usage_pct=$(echo "scale=2; $mem_usage_mb * 100 / $mem_limit_mb" | bc)
        fi
        
        # Determine status based on usage
        local status="good"
        if (( $(echo "$usage_pct > 90" | bc -l) )); then
            status="critical"
        elif (( $(echo "$usage_pct > 75" | bc -l) )); then
            status="degraded"
        fi
        
        memory_health=$(echo "$memory_health" | jq \
            --arg status "$status" \
            --argjson usage "$mem_usage_mb" \
            --argjson limit "$mem_limit_mb" \
            --argjson pct "$usage_pct" \
            '.status = $status |
             .container_memory_mb = $usage |
             .container_limit_mb = $limit |
             .usage_percentage = $pct')
    fi
    
    echo "$memory_health"
    return 0
}

#######################################
# Check search performance
# Returns: JSON with performance metrics
#######################################
qdrant::health::check_search_performance() {
    local search_health='{
        "status": "unknown",
        "average_latency_ms": 0,
        "collections_tested": 0,
        "performance_grade": "unknown"
    }'
    
    # Get performance metrics from optimizer
    local perf_metrics
    if perf_metrics=$(qdrant::search::performance_metrics 2>/dev/null); then
        local avg_latency collections_tested perf_status
        avg_latency=$(echo "$perf_metrics" | jq -r '.summary.average_search_latency_ms // 0')
        collections_tested=$(echo "$perf_metrics" | jq -r '.summary.total_collections // 0')
        perf_status=$(echo "$perf_metrics" | jq -r '.summary.performance_status // "unknown"')
        
        # Map performance status to health status
        local status="good"
        case "$perf_status" in
            excellent) status="excellent" ;;
            good) status="good" ;;
            degraded) status="degraded" ;;
            critical) status="critical" ;;
            *) status="unknown" ;;
        esac
        
        search_health=$(echo "$search_health" | jq \
            --arg status "$status" \
            --argjson latency "$avg_latency" \
            --argjson tested "$collections_tested" \
            --arg grade "$perf_status" \
            '.status = $status |
             .average_latency_ms = $latency |
             .collections_tested = $tested |
             .performance_grade = $grade')
    fi
    
    echo "$search_health"
    return 0
}

#######################################
# Calculate overall health status
# Arguments:
#   $1 - Health data JSON
# Returns: Overall status string
#######################################
qdrant::health::calculate_overall_status() {
    local health_data="${1:-}"
    
    local api_status collections_status memory_status search_status
    api_status=$(echo "$health_data" | jq -r '.components.api.status // "unknown"')
    collections_status=$(echo "$health_data" | jq -r '.components.collections.status // "unknown"')
    memory_status=$(echo "$health_data" | jq -r '.components.memory.status // "unknown"')
    search_status=$(echo "$health_data" | jq -r '.components.search.status // "unknown"')
    
    # If API is down, overall is down
    if [[ "$api_status" == "down" ]]; then
        echo "$HEALTH_DOWN"
        return 0
    fi
    
    # Check for critical issues
    if [[ "$memory_status" == "critical" ]] || [[ "$search_status" == "critical" ]]; then
        echo "$HEALTH_CRITICAL"
        return 0
    fi
    
    # Check for degraded components
    local degraded_count=0
    for status in "$collections_status" "$memory_status" "$search_status"; do
        if [[ "$status" == "degraded" ]]; then
            ((degraded_count++))
        fi
    done
    
    if [[ $degraded_count -ge 2 ]]; then
        echo "$HEALTH_DEGRADED"
    elif [[ $degraded_count -eq 1 ]]; then
        echo "$HEALTH_GOOD"
    else
        echo "$HEALTH_EXCELLENT"
    fi
    
    return 0
}

#######################################
# Display health status in human-readable format
#######################################
qdrant::health::display() {
    local health_data
    if ! health_data=$(qdrant::health::get_status); then
        echo "❌ Qdrant health check failed"
        return 1
    fi
    
    local overall_status
    overall_status=$(echo "$health_data" | jq -r '.status')
    
    # Status emoji
    local status_emoji="❓"
    case "$overall_status" in
        excellent) status_emoji="✨" ;;
        good) status_emoji="✅" ;;
        degraded) status_emoji="⚠️" ;;
        critical) status_emoji="🔴" ;;
        down) status_emoji="❌" ;;
    esac
    
    echo "=== Qdrant Health Status ==="
    echo
    echo "$status_emoji Overall Status: ${overall_status^^}"
    echo
    
    # API status
    local api_status api_latency
    api_status=$(echo "$health_data" | jq -r '.components.api.status // "unknown"')
    api_latency=$(echo "$health_data" | jq -r '.components.api.latency_ms // 0')
    echo "🌐 API: $api_status (${api_latency}ms response time)"
    
    # Version info
    local version
    version=$(echo "$health_data" | jq -r '.components.version.version // "unknown"')
    echo "📦 Version: $version"
    
    # Collections health
    local total_collections total_vectors
    total_collections=$(echo "$health_data" | jq -r '.components.collections.total_collections // 0')
    total_vectors=$(echo "$health_data" | jq -r '.components.collections.total_vectors // 0')
    echo "📁 Collections: $total_collections (${total_vectors} total vectors)"
    
    # Memory status
    local mem_usage mem_limit mem_pct
    mem_usage=$(echo "$health_data" | jq -r '.components.memory.container_memory_mb // 0')
    mem_limit=$(echo "$health_data" | jq -r '.components.memory.container_limit_mb // 0')
    mem_pct=$(echo "$health_data" | jq -r '.components.memory.usage_percentage // 0')
    echo "💾 Memory: ${mem_usage}MB / ${mem_limit}MB (${mem_pct}%)"
    
    # Search performance
    local avg_latency perf_grade
    avg_latency=$(echo "$health_data" | jq -r '.components.search.average_latency_ms // 0')
    perf_grade=$(echo "$health_data" | jq -r '.components.search.performance_grade // "unknown"')
    echo "🔍 Search Performance: $perf_grade (${avg_latency}ms avg latency)"
    
    # Issues if any
    local issues
    issues=$(echo "$health_data" | jq -r '.components.collections.issues[]? // empty')
    if [[ -n "$issues" ]]; then
        echo
        echo "⚠️  Issues Detected:"
        echo "$issues" | while IFS= read -r issue; do
            echo "  - $issue"
        done
    fi
    
    echo
    return 0
}