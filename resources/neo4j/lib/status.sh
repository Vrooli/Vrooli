#!/usr/bin/env bash
# Neo4j Resource - Status Functions

# Get script directory and source common
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
NEO4J_LIB_DIR="${APP_ROOT}/resources/neo4j/lib"
source "$NEO4J_LIB_DIR/common.sh"
source "$NEO4J_LIB_DIR/start.sh"

# Source format utility for consistent output
FORMAT_UTIL="${NEO4J_LIB_DIR}/../../../lib/utils/format.sh"
[[ -f "$FORMAT_UTIL" ]] && source "$FORMAT_UTIL"
# shellcheck disable=SC1091
source "${NEO4J_LIB_DIR}/../../../lib/status-args.sh"

#######################################
# Collect Neo4j status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
neo4j::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Check if installed
    local installed="false"
    if neo4j_is_installed; then
        installed="true"
    fi
    
    # Check if running
    local running="false"
    if neo4j_is_running; then
        running="true"
    fi
    
    # Check health
    local healthy="false"
    local health_message="Not running"
    local version=""
    local nodes="0"
    local relationships="0"
    
    if [[ "$running" == "true" ]]; then
        # Try to get basic status (skip if fast mode)
        if [[ "$fast_mode" == "false" ]] && timeout 2s curl -s "http://localhost:$NEO4J_HTTP_PORT/" 2>/dev/null | grep -q '"neo4j_version"'; then
            healthy="true"
            health_message="Neo4j is healthy and accepting connections"
            
            # Try to get version
            version=$(timeout 3s docker exec "$NEO4J_CONTAINER_NAME" neo4j --version 2>/dev/null | grep -oP 'neo4j \K[0-9.]+' || echo "unknown")
            
            # Try to get node/relationship count (requires auth)
            local cypher_result=$(echo 'MATCH (n) RETURN count(n) as nodes' | \
                timeout 3s docker exec -i "$NEO4J_CONTAINER_NAME" cypher-shell \
                -u neo4j -p "VrooliNeo4j2024!" \
                --format plain 2>/dev/null | tail -1)
            if [[ -n "$cypher_result" ]]; then
                nodes="$cypher_result"
            fi
        elif [[ "$fast_mode" == "true" ]]; then
            # Fast mode - assume healthy if running
            healthy="true"
            health_message="Fast mode - skipped health check"
            version="N/A"
            nodes="N/A"
        else
            health_message="Neo4j is running but not responding"
        fi
    elif [[ "$installed" == "true" ]]; then
        health_message="Neo4j is installed but not running"
    else
        health_message="Neo4j is not installed"
    fi
    
    # Container status
    local container_status="not_found"
    if [[ "$installed" == "true" ]]; then
        container_status=$(docker inspect --format='{{.State.Status}}' "$NEO4J_CONTAINER_NAME" 2>/dev/null || echo "unknown")
    fi
    
    # Basic resource information
    status_data+=("name" "neo4j")
    status_data+=("category" "storage")
    status_data+=("description" "Native property graph database")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$NEO4J_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("version" "$version")
    status_data+=("http_port" "$NEO4J_HTTP_PORT")
    status_data+=("bolt_port" "$NEO4J_BOLT_PORT")
    status_data+=("nodes" "$nodes")
    status_data+=("relationships" "$relationships")
    status_data+=("http_url" "http://localhost:$NEO4J_HTTP_PORT")
    status_data+=("bolt_url" "bolt://localhost:$NEO4J_BOLT_PORT")
    
    # Docker image if available
    if [[ "$installed" == "true" ]]; then
        local image
        image=$(docker inspect --format='{{.Config.Image}}' "$NEO4J_CONTAINER_NAME" 2>/dev/null || echo "unknown")
        status_data+=("image" "$image")
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display Neo4j status in text format
# Args: data_array (key-value pairs)
#######################################
neo4j::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    echo "Neo4j Status Report"
    echo "==================="
    echo
    
    # Basic status
    echo "Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        echo "  ✅ Installed: Yes"
    else
        echo "  ❌ Installed: No"
        echo
        echo "Installation Required:"
        echo "  To install Neo4j, run the installation command"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        echo "  ✅ Running: Yes"
    else
        echo "  ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "  ✅ Health: Healthy"
    else
        echo "  ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    echo "Container Info:"
    echo "  📦 Name: ${data[container_name]:-unknown}"
    echo "  📊 Status: ${data[container_status]:-unknown}"
    echo "  🖼️  Image: ${data[image]:-unknown}"
    echo
    
    # Service endpoints
    echo "Service Endpoints:"
    echo "  🌐 HTTP: ${data[http_url]:-unknown}"
    echo "  ⚡ Bolt: ${data[bolt_url]:-unknown}"
    echo "  📶 HTTP Port: ${data[http_port]:-unknown}"
    echo "  📶 Bolt Port: ${data[bolt_port]:-unknown}"
    echo
    
    # Database information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "Database Info:"
        echo "  📊 Version: ${data[version]:-unknown}"
        echo "  🔸 Nodes: ${data[nodes]:-0}"
        echo "  🔗 Relationships: ${data[relationships]:-0}"
    fi
}

neo4j_status() {
    status::run_standard "neo4j" "neo4j::status::collect_data" "neo4j::status::display_text" "$@"
}