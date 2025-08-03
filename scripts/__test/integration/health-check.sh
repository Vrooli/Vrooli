#!/usr/bin/env bash
# Health Check System - Validates service availability before integration tests
# Checks if required services are running and accessible

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR

# Load configuration
CONFIG_FILE="$SCRIPT_DIR/../config/test-config.yaml"

# Default settings
VERBOSE=false
REQUIRED_ONLY=false
RETRY_COUNT=3
RETRY_DELAY=2

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -r|--required-only)
            REQUIRED_ONLY=true
            shift
            ;;
        --retry)
            RETRY_COUNT="$2"
            shift 2
            ;;
        *)
            shift
            ;;
    esac
done

# Logging functions
log_info() {
    [[ "$VERBOSE" == "true" ]] && echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $*"
}

log_error() {
    echo -e "${RED}[✗]${NC} $*"
}

# Service health check functions
check_http_health() {
    local service="$1"
    local url="$2"
    local retries="$RETRY_COUNT"
    
    while [[ $retries -gt 0 ]]; do
        if curl -sf --max-time 5 "$url" > /dev/null 2>&1; then
            return 0
        fi
        retries=$((retries - 1))
        [[ $retries -gt 0 ]] && sleep "$RETRY_DELAY"
    done
    
    return 1
}

check_tcp_port() {
    local service="$1"
    local host="$2"
    local port="$3"
    local retries="$RETRY_COUNT"
    
    while [[ $retries -gt 0 ]]; do
        if timeout 5 bash -c "echo > /dev/tcp/$host/$port" 2>/dev/null; then
            return 0
        fi
        retries=$((retries - 1))
        [[ $retries -gt 0 ]] && sleep "$RETRY_DELAY"
    done
    
    return 1
}

check_docker_container() {
    local service="$1"
    local container="$2"
    
    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${container}$"; then
        local status=$(docker inspect -f '{{.State.Status}}' "$container" 2>/dev/null)
        [[ "$status" == "running" ]]
    else
        return 1
    fi
}

check_process() {
    local service="$1"
    local process="$2"
    
    pgrep -f "$process" > /dev/null 2>&1
}

# Service definitions with health check methods
declare -A SERVICE_CHECKS=(
    # AI Services
    ["ollama"]="http|http://localhost:11434/api/tags"
    ["whisper"]="http|http://localhost:9000/health"
    ["unstructured-io"]="http|http://localhost:8000/general/v0/general"
    
    # Agents
    ["agent-s2"]="http|http://localhost:8111/"
    ["browserless"]="http|http://localhost:3000/"
    ["claude-code"]="tcp|localhost|5173"
    
    # Automation
    ["comfyui"]="http|http://localhost:8188/"
    ["huginn"]="http|http://localhost:3001/"
    ["n8n"]="http|http://localhost:5678/"
    ["node-red"]="http|http://localhost:1880/"
    ["windmill"]="http|http://localhost:8000/"
    
    # Storage
    ["postgres"]="tcp|localhost|5432"
    ["redis"]="tcp|localhost|6379"
    ["minio"]="http|http://localhost:9001/"
    ["qdrant"]="http|http://localhost:6333/health"
    ["questdb"]="http|http://localhost:9000/"
    ["vault"]="http|http://localhost:8200/v1/sys/health"
    
    # Search
    ["searxng"]="http|http://localhost:8080/"
    
    # Execution
    ["judge0"]="http|http://localhost:2358/health"
)

# Required services (must be running)
REQUIRED_SERVICES=("postgres" "redis")

# Check individual service
check_service() {
    local service="$1"
    local check_spec="${SERVICE_CHECKS[$service]:-}"
    
    if [[ -z "$check_spec" ]]; then
        log_warning "$service: No health check defined"
        return 2  # Unknown
    fi
    
    IFS='|' read -r check_type check_param1 check_param2 <<< "$check_spec"
    
    log_info "Checking $service ($check_type)..."
    
    case "$check_type" in
        http)
            if check_http_health "$service" "$check_param1"; then
                log_success "$service is healthy"
                return 0
            else
                log_error "$service is not responding"
                return 1
            fi
            ;;
        tcp)
            if check_tcp_port "$service" "$check_param1" "$check_param2"; then
                log_success "$service is accessible"
                return 0
            else
                log_error "$service port is not open"
                return 1
            fi
            ;;
        docker)
            if check_docker_container "$service" "$check_param1"; then
                log_success "$service container is running"
                return 0
            else
                log_error "$service container is not running"
                return 1
            fi
            ;;
        process)
            if check_process "$service" "$check_param1"; then
                log_success "$service process is running"
                return 0
            else
                log_error "$service process is not found"
                return 1
            fi
            ;;
        *)
            log_warning "$service: Unknown check type: $check_type"
            return 2
            ;;
    esac
}

# Main health check execution
main() {
    local exit_code=0
    local healthy_count=0
    local unhealthy_count=0
    local unknown_count=0
    
    echo "======================================="
    echo "      Service Health Check Report      "
    echo "======================================="
    echo
    
    # Check required services first
    echo "Required Services:"
    echo "------------------"
    for service in "${REQUIRED_SERVICES[@]}"; do
        if check_service "$service"; then
            healthy_count=$((healthy_count + 1))
        else
            unhealthy_count=$((unhealthy_count + 1))
            exit_code=1
        fi
    done
    echo
    
    # Check optional services (if not required-only mode)
    if [[ "$REQUIRED_ONLY" != "true" ]]; then
        echo "Optional Services:"
        echo "------------------"
        for service in "${!SERVICE_CHECKS[@]}"; do
            # Skip if already checked as required
            if [[ " ${REQUIRED_SERVICES[@]} " =~ " ${service} " ]]; then
                continue
            fi
            
            if check_service "$service"; then
                healthy_count=$((healthy_count + 1))
            else
                status=$?
                if [[ $status -eq 1 ]]; then
                    unhealthy_count=$((unhealthy_count + 1))
                else
                    unknown_count=$((unknown_count + 1))
                fi
            fi
        done
        echo
    fi
    
    # Summary
    echo "======================================="
    echo "Summary:"
    echo "  Healthy:   $healthy_count"
    echo "  Unhealthy: $unhealthy_count"
    [[ $unknown_count -gt 0 ]] && echo "  Unknown:   $unknown_count"
    echo "======================================="
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "All required services are healthy"
    else
        log_error "Some required services are not healthy"
    fi
    
    exit $exit_code
}

# Run main function
main