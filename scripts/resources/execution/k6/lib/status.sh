#!/usr/bin/env bash
# K6 Status Functions

# Source format utility for JSON support
K6_STATUS_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${K6_STATUS_SCRIPT_DIR}/../../../../lib/utils/format.sh" 2>/dev/null || true

# Check K6 status
k6::status::check() {
    local format="${1:-plain}"
    
    # Initialize if needed
    k6::core::init 2>/dev/null || true
    
    # Check installation
    local installed="false"
    local running="false"
    local health="unhealthy"
    local message="K6 not installed"
    local version="unknown"
    
    # Check native installation
    if k6::core::is_installed_native; then
        installed="true"
        version=$(k6 version 2>/dev/null | grep -oP 'k6 v\K[0-9.]+' || echo "unknown")
        running="true"  # CLI tool is always "running" when installed
        health="healthy"
        message="K6 CLI installed and ready"
    fi
    
    # Check Docker installation
    if k6::core::is_running; then
        installed="true"
        running="true"
        health="healthy"
        message="K6 container running"
        if [[ "$version" == "unknown" ]]; then
            version=$(docker exec "$K6_CONTAINER_NAME" k6 version 2>/dev/null | grep -oP 'k6 v\K[0-9.]+' || echo "unknown")
        fi
    fi
    
    # Count test scripts
    local test_count=0
    if [[ -d "$K6_SCRIPTS_DIR" ]]; then
        test_count=$(find "$K6_SCRIPTS_DIR" -name "*.js" 2>/dev/null | wc -l)
    fi
    
    # Count results
    local results_count=0
    if [[ -d "$K6_RESULTS_DIR" ]]; then
        results_count=$(find "$K6_RESULTS_DIR" -name "*.json" -o -name "*.csv" 2>/dev/null | wc -l)
    fi
    
    # Format output
    format::output "$format" "kv" \
        "name" "k6" \
        "category" "$K6_CATEGORY" \
        "description" "$K6_DESCRIPTION" \
        "installed" "$installed" \
        "running" "$running" \
        "healthy" "$health" \
        "health_message" "$message" \
        "version" "$version" \
        "port" "$K6_PORT" \
        "test_scripts" "$test_count" \
        "results" "$results_count" \
        "scripts_dir" "$K6_SCRIPTS_DIR" \
        "results_dir" "$K6_RESULTS_DIR"
}