#!/usr/bin/env bash
# Health Check Validator
# Validates health check responses against JSON schemas

set -euo pipefail

# Get the directory where this script is located
HEALTH_VALIDATOR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_DIR="${HEALTH_VALIDATOR_DIR}/../schemas"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Validate a health response against a schema
# Usage: validate_health_response <response_json> <service_type>
# Service types: ui, api, service
validate_health_response() {
    local response="$1"
    local service_type="${2:-service}"
    local schema_file="${SCHEMA_DIR}/health-${service_type}.schema.json"
    
    # Check if schema file exists
    if [[ ! -f "$schema_file" ]]; then
        echo -e "${YELLOW}⚠️  Schema file not found: $schema_file${NC}" >&2
        return 1
    fi
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        echo -e "${RED}❌ jq is required for JSON validation${NC}" >&2
        return 1
    fi
    
    # Basic validation using jq
    # This checks required fields and basic structure
    local validation_errors=""
    
    # Check required fields based on service type
    case "$service_type" in
        ui)
            # UI must have status, service, timestamp, readiness, and api_connectivity
            if ! echo "$response" | jq -e '.status' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: status\n"
            fi
            if ! echo "$response" | jq -e '.service' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: service\n"
            fi
            if ! echo "$response" | jq -e '.timestamp' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: timestamp\n"
            fi
            if ! echo "$response" | jq -e '.readiness' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: readiness\n"
            fi
            if ! echo "$response" | jq -e '.api_connectivity' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: api_connectivity\n"
            else
                # Check api_connectivity required fields
                if ! echo "$response" | jq '.api_connectivity | has("connected")' | grep -q true 2>/dev/null; then
                    validation_errors="${validation_errors}Missing required field: api_connectivity.connected\n"
                fi
                if ! echo "$response" | jq '.api_connectivity | has("api_url")' | grep -q true 2>/dev/null; then
                    validation_errors="${validation_errors}Missing required field: api_connectivity.api_url\n"
                fi
                if ! echo "$response" | jq '.api_connectivity | has("last_check")' | grep -q true 2>/dev/null; then
                    validation_errors="${validation_errors}Missing required field: api_connectivity.last_check\n"
                fi
            fi
            
            # Check status enum
            local status=$(echo "$response" | jq -r '.status // ""')
            if [[ ! "$status" =~ ^(healthy|degraded|unhealthy)$ ]]; then
                validation_errors="${validation_errors}Invalid status value: $status (must be healthy, degraded, or unhealthy)\n"
            fi
            
            # Check readiness boolean
            local readiness=$(echo "$response" | jq -r '.readiness // null')
            if [[ "$readiness" != "true" ]] && [[ "$readiness" != "false" ]]; then
                validation_errors="${validation_errors}Invalid readiness value: $readiness (must be boolean true or false)\n"
            fi
            
            # Check timestamp format (basic ISO 8601 check)
            local timestamp=$(echo "$response" | jq -r '.timestamp // ""')
            if [[ -n "$timestamp" ]] && ! echo "$timestamp" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}'; then
                validation_errors="${validation_errors}Invalid timestamp format: $timestamp (must be ISO 8601 format)\n"
            fi
            ;;
            
        api)
            # API must have status, service, timestamp, and readiness
            if ! echo "$response" | jq -e '.status' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: status\n"
            fi
            if ! echo "$response" | jq -e '.service' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: service\n"
            fi
            if ! echo "$response" | jq -e '.timestamp' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: timestamp\n"
            fi
            if ! echo "$response" | jq -e '.readiness' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: readiness\n"
            fi
            
            # Check status enum
            local status=$(echo "$response" | jq -r '.status // ""')
            if [[ ! "$status" =~ ^(healthy|degraded|unhealthy)$ ]]; then
                validation_errors="${validation_errors}Invalid status value: $status (must be healthy, degraded, or unhealthy)\n"
            fi
            
            # Check readiness boolean
            local readiness=$(echo "$response" | jq -r '.readiness // null')
            if [[ "$readiness" != "true" ]] && [[ "$readiness" != "false" ]]; then
                validation_errors="${validation_errors}Invalid readiness value: $readiness (must be boolean true or false)\n"
            fi
            
            # Check timestamp format (basic ISO 8601 check)
            local timestamp=$(echo "$response" | jq -r '.timestamp // ""')
            if [[ -n "$timestamp" ]] && ! echo "$timestamp" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}'; then
                validation_errors="${validation_errors}Invalid timestamp format: $timestamp (must be ISO 8601 format)\n"
            fi
            ;;
            
        service)
            # Generic service must have status, service, timestamp, and readiness
            if ! echo "$response" | jq -e '.status' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: status\n"
            fi
            if ! echo "$response" | jq -e '.service' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: service\n"
            fi
            if ! echo "$response" | jq -e '.timestamp' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: timestamp\n"
            fi
            if ! echo "$response" | jq -e '.readiness' > /dev/null 2>&1; then
                validation_errors="${validation_errors}Missing required field: readiness\n"
            fi
            
            # Check status enum
            local status=$(echo "$response" | jq -r '.status // ""')
            if [[ ! "$status" =~ ^(healthy|degraded|unhealthy)$ ]]; then
                validation_errors="${validation_errors}Invalid status value: $status (must be healthy, degraded, or unhealthy)\n"
            fi
            
            # Check readiness boolean
            local readiness=$(echo "$response" | jq -r '.readiness // null')
            if [[ "$readiness" != "true" ]] && [[ "$readiness" != "false" ]]; then
                validation_errors="${validation_errors}Invalid readiness value: $readiness (must be boolean true or false)\n"
            fi
            
            # Check timestamp format (basic ISO 8601 check)
            local timestamp=$(echo "$response" | jq -r '.timestamp // ""')
            if [[ -n "$timestamp" ]] && ! echo "$timestamp" | grep -qE '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}'; then
                validation_errors="${validation_errors}Invalid timestamp format: $timestamp (must be ISO 8601 format)\n"
            fi
            ;;
    esac
    
    # Return validation result
    if [[ -n "$validation_errors" ]]; then
        echo -e "${RED}❌ Validation failed for $service_type health check:${NC}" >&2
        echo -e "$validation_errors" >&2
        return 1
    else
        echo -e "${GREEN}✅ Valid $service_type health response${NC}" >&2
        return 0
    fi
}

# Get health status from response
# Returns: healthy, degraded, unhealthy, or unknown
get_health_status() {
    local response="$1"
    echo "$response" | jq -r '.status // "unknown"' 2>/dev/null || echo "unknown"
}

# Check if UI has API connectivity
# Returns: true, false, or unknown
check_ui_api_connectivity() {
    local response="$1"
    local connected=$(echo "$response" | jq -r '.api_connectivity.connected // "unknown"' 2>/dev/null)
    echo "$connected"
}

# Get API connectivity error if any
get_api_connectivity_error() {
    local response="$1"
    echo "$response" | jq -r '.api_connectivity.error // null' 2>/dev/null
}

# Format health response for display
format_health_response() {
    local response="$1"
    local service_type="${2:-service}"
    
    local status=$(get_health_status "$response")
    local service=$(echo "$response" | jq -r '.service // "unknown"' 2>/dev/null)
    
    echo "Service: $service"
    echo -n "Status: "
    case "$status" in
        healthy)
            echo -e "${GREEN}✅ $status${NC}"
            ;;
        degraded)
            echo -e "${YELLOW}⚠️  $status${NC}"
            ;;
        unhealthy)
            echo -e "${RED}❌ $status${NC}"
            ;;
        *)
            echo "❓ $status"
            ;;
    esac
    
    # Show API connectivity for UI services
    if [[ "$service_type" == "ui" ]]; then
        local api_connected=$(check_ui_api_connectivity "$response")
        local api_url=$(echo "$response" | jq -r '.api_connectivity.api_url // "unknown"' 2>/dev/null)
        
        echo -n "API Connectivity: "
        if [[ "$api_connected" == "true" ]]; then
            local latency=$(echo "$response" | jq -r '.api_connectivity.latency_ms // null' 2>/dev/null)
            echo -e "${GREEN}✅ Connected to $api_url${NC}"
            [[ "$latency" != "null" ]] && echo "  Latency: ${latency}ms"
        elif [[ "$api_connected" == "false" ]]; then
            echo -e "${RED}❌ Disconnected from $api_url${NC}"
            local error=$(get_api_connectivity_error "$response")
            [[ "$error" != "null" ]] && echo "  Error: $error"
        else
            echo "❓ Unknown"
        fi
    fi
}

# Main function for CLI usage
main() {
    local response="${1:-}"
    local service_type="${2:-service}"
    
    if [[ -z "$response" ]]; then
        echo "Usage: $0 <json_response> [service_type]"
        echo "Service types: ui, api, service (default)"
        echo ""
        echo "Example:"
        echo "  curl -s http://localhost:3000/health | $0 - ui"
        exit 1
    fi
    
    # Read from stdin if response is "-"
    if [[ "$response" == "-" ]]; then
        response=$(cat)
    fi
    
    # Validate the response
    if validate_health_response "$response" "$service_type"; then
        echo ""
        format_health_response "$response" "$service_type"
        exit 0
    else
        exit 1
    fi
}

# Run main if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi