#!/usr/bin/env bash

# Unstructured.io Status Functions
# This file contains functions for checking service status and health

#######################################
# Check and display service status
#######################################
unstructured_io::status() {
    local verbose="${1:-yes}"
    
    if [ "$verbose" = "yes" ]; then
        echo "🔍 Checking Unstructured.io status..."
        echo
    fi
    
    # Check if container exists
    if ! unstructured_io::container_exists; then
        [ "$verbose" = "yes" ] && echo "$MSG_UNSTRUCTURED_IO_NOT_FOUND"
        return 1
    fi
    [ "$verbose" = "yes" ] && echo "$MSG_STATUS_CONTAINER_OK"
    
    # Check if container is running
    if ! unstructured_io::container_running; then
        [ "$verbose" = "yes" ] && echo "$MSG_UNSTRUCTURED_IO_NOT_RUNNING"
        return 1
    fi
    [ "$verbose" = "yes" ] && echo "$MSG_STATUS_CONTAINER_RUNNING"
    
    # Check port binding by testing API response
    if curl -s -f "http://localhost:${UNSTRUCTURED_IO_PORT}/healthcheck" > /dev/null 2>&1; then
        [ "$verbose" = "yes" ] && echo "$MSG_STATUS_PORT_OK"
    else
        [ "$verbose" = "yes" ] && echo "⚠️  Port $UNSTRUCTURED_IO_PORT is not responding"
        return 1
    fi
    
    # Check API health
    if unstructured_io::check_health; then
        [ "$verbose" = "yes" ] && echo "$MSG_STATUS_API_OK"
    else
        [ "$verbose" = "yes" ] && echo "$MSG_UNSTRUCTURED_IO_UNHEALTHY"
        return 1
    fi
    
    if [ "$verbose" = "yes" ]; then
        echo
        echo "$MSG_STATUS_FORMATS_SUPPORTED"
        echo
        echo "$MSG_UNSTRUCTURED_IO_RUNNING"
    fi
    
    return 0
}

#######################################
# Check API health
#######################################
unstructured_io::check_health() {
    local response
    response=$(curl -s -w "\n%{http_code}" "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_HEALTH_ENDPOINT}" 2>/dev/null)
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Display service information
#######################################
unstructured_io::info() {
    echo "📄 Unstructured.io Service Information"
    echo "======================================"
    echo
    echo "Service URL: $UNSTRUCTURED_IO_BASE_URL"
    echo "Container: $UNSTRUCTURED_IO_CONTAINER_NAME"
    echo "Docker Image: $UNSTRUCTURED_IO_IMAGE"
    echo
    echo "$MSG_CONFIG_STRATEGY"
    echo "$MSG_CONFIG_LANGUAGES"
    echo "$MSG_CONFIG_MEMORY"
    echo "$MSG_CONFIG_CPU"
    echo "$MSG_CONFIG_TIMEOUT"
    echo
    echo "$MSG_API_ENDPOINT_INFO"
    echo "$MSG_API_PROCESS"
    echo "$MSG_API_BATCH"
    echo "$MSG_API_HEALTH"
    echo "$MSG_API_METRICS"
    echo
    echo "Supported Document Formats:"
    echo "=========================="
    local count=0
    for format in "${UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]}"; do
        printf "%-8s" ".$format"
        ((count++))
        if [ $((count % 8)) -eq 0 ]; then
            echo
        fi
    done
    [ $((count % 8)) -ne 0 ] && echo
    echo
    
    # Show current status
    unstructured_io::status
}

#######################################
# View container logs
#######################################
unstructured_io::logs() {
    local follow="${1:-no}"
    
    if ! unstructured_io::container_exists; then
        echo "$MSG_UNSTRUCTURED_IO_NOT_FOUND"
        return 1
    fi
    
    echo "📋 Unstructured.io Container Logs"
    echo "================================="
    
    if [ "$follow" = "yes" ]; then
        docker logs -f "$UNSTRUCTURED_IO_CONTAINER_NAME"
    else
        docker logs --tail 50 "$UNSTRUCTURED_IO_CONTAINER_NAME"
    fi
}

#######################################
# Get processing metrics
#######################################
unstructured_io::metrics() {
    if ! unstructured_io::status "no"; then
        echo "❌ Unstructured.io is not running"
        return 1
    fi
    
    echo "📊 Processing Metrics"
    echo "===================="
    
    local response
    response=$(curl -s "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_METRICS_ENDPOINT}" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        echo "$response" | grep -E "^(# HELP|# TYPE|unstructured_)" | head -20
    else
        echo "❌ Failed to retrieve metrics"
        return 1
    fi
}

#######################################
# Get container resource usage
#######################################
unstructured_io::resource_usage() {
    if ! unstructured_io::container_running; then
        echo "Container is not running"
        return 1
    fi
    
    echo "💻 Resource Usage"
    echo "================"
    
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}" \
        "$UNSTRUCTURED_IO_CONTAINER_NAME"
}

#######################################
# Export functions for subshell availability
#######################################
export -f unstructured_io::status
export -f unstructured_io::check_health
export -f unstructured_io::resource_usage