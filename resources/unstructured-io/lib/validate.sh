#!/usr/bin/env bash

# Unstructured.io Installation Validation
# This file contains functions to validate the installation and configuration

# Source var.sh for directory variables
# shellcheck disable=SC1091
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"

#######################################
# Validate Unstructured.io installation
#######################################
unstructured_io::validate_installation() {
    local verbose="${1:-yes}"
    local errors=0
    local warnings=0
    
    if [[ "$verbose" == "yes" ]]; then
        echo "🔍 Validating Unstructured.io Installation"
        echo "=========================================="
    fi
    
    # Check Docker
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking Docker installation... "
    fi
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker not found"
        ((errors++))
    elif ! docker info &> /dev/null; then
        echo "❌ Docker daemon not running"
        ((errors++))
    else
        if [[ "$verbose" == "yes" ]]; then
            echo "✅ OK"
        fi
    fi
    
    # Check container existence
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking container existence... "
    fi
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${UNSTRUCTURED_IO_CONTAINER_NAME}$"; then
        echo "❌ Container not found"
        ((errors++))
    else
        if [[ "$verbose" == "yes" ]]; then
            echo "✅ OK"
        fi
    fi
    
    # Check container status
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking container status... "
    fi
    local container_status=$(docker inspect -f '{{.State.Status}}' "$UNSTRUCTURED_IO_CONTAINER_NAME" 2>/dev/null)
    if [[ "$container_status" != "running" ]]; then
        echo "⚠️  Container not running (status: $container_status)"
        ((warnings++))
    else
        if [[ "$verbose" == "yes" ]]; then
            echo "✅ Running"
        fi
    fi
    
    # Check port availability
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking port $UNSTRUCTURED_IO_PORT... "
    fi
    if nc -z localhost "$UNSTRUCTURED_IO_PORT" 2>/dev/null; then
        if [[ "$verbose" == "yes" ]]; then
            echo "✅ Open"
        fi
    else
        echo "❌ Port not accessible"
        ((errors++))
    fi
    
    # Check API health
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking API health... "
    fi
    local health_response=$(curl -s -o /dev/null -w "%{http_code}" \
        "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_HEALTH_ENDPOINT}" 2>/dev/null)
    if [[ "$health_response" == "200" ]]; then
        if [[ "$verbose" == "yes" ]]; then
            echo "✅ Healthy"
        fi
    else
        echo "❌ API not responding (HTTP $health_response)"
        ((errors++))
    fi
    
    # Check cache directory
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking cache directory... "
    fi
    if [[ -d "$UNSTRUCTURED_IO_CACHE_DIR" ]]; then
        if [[ -w "$UNSTRUCTURED_IO_CACHE_DIR" ]]; then
            if [[ "$verbose" == "yes" ]]; then
                echo "✅ Writable"
            fi
        else
            echo "⚠️  Not writable"
            ((warnings++))
        fi
    else
        echo "⚠️  Not found (will be created on first use)"
        ((warnings++))
    fi
    
    # Check memory limits
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking container resources... "
    fi
    local mem_limit=$(docker inspect "$UNSTRUCTURED_IO_CONTAINER_NAME" --format='{{.HostConfig.Memory}}' 2>/dev/null)
    if [[ -n "$mem_limit" ]] && [[ "$mem_limit" -gt 0 ]]; then
        local mem_gb=$((mem_limit / 1073741824))
        if [[ "$verbose" == "yes" ]]; then
            echo "✅ Memory limit: ${mem_gb}GB"
        fi
    else
        echo "⚠️  No memory limit set"
        ((warnings++))
    fi
    
    # Test document processing
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Testing document processing... "
    fi
    local test_file="/tmp/unstructured_validate_$$.txt"
    echo "Validation test" > "$test_file"
    
    if unstructured_io::process_document "$test_file" "fast" "text" > /dev/null 2>&1; then
        if [[ "$verbose" == "yes" ]]; then
            echo "✅ Working"
        fi
    else
        echo "❌ Processing failed"
        ((errors++))
    fi
    trash::safe_remove "$test_file" --temp
    
    # Check supported formats
    if [[ "$verbose" == "yes" ]]; then
        echo -n "Checking format support... "
        echo "✅ ${#UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]} formats supported"
    fi
    
    # Validate configuration
    if [[ "$verbose" == "yes" ]]; then
        echo
        echo "Configuration:"
        echo "  Base URL: $UNSTRUCTURED_IO_BASE_URL"
        echo "  Default strategy: $UNSTRUCTURED_IO_DEFAULT_STRATEGY"
        echo "  Default languages: $UNSTRUCTURED_IO_DEFAULT_LANGUAGES"
        echo "  Timeout: ${UNSTRUCTURED_IO_TIMEOUT_SECONDS}s"
        echo "  Max file size: $UNSTRUCTURED_IO_MAX_FILE_SIZE"
        echo "  Cache enabled: ${UNSTRUCTURED_IO_CACHE_ENABLED:-yes}"
    fi
    
    # Summary
    if [[ "$verbose" == "yes" ]]; then
        echo
        echo "=========================================="
        echo "Validation Summary:"
        if [[ $errors -eq 0 ]]; then
            echo "✅ All checks passed!"
        else
            echo "❌ Found $errors error(s)"
        fi
        
        if [[ $warnings -gt 0 ]]; then
            echo "⚠️  Found $warnings warning(s)"
        fi
    fi
    
    # Return error code if any errors found
    if [[ $errors -gt 0 ]]; then
        return 1
    else
        return 0
    fi
}

# Export function
export -f unstructured_io::validate_installation