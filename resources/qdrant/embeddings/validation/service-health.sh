#!/usr/bin/env bash
# Service Health Checks for Qdrant Embeddings
# Validates that all required services are available and functional

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Configuration
QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
OLLAMA_URL="${OLLAMA_URL:-http://localhost:11434}"
DEFAULT_MODEL="${QDRANT_EMBEDDING_MODEL:-mxbai-embed-large}"
MAX_RETRIES=5
RETRY_DELAY=2

#######################################
# Check if a service is responding with retries
# Arguments:
#   $1 - Service URL
#   $2 - Service name
#   $3 - Timeout in seconds (optional)
# Returns: 0 if healthy, 1 if failed
#######################################
check_service_health() {
    local url="$1"
    local service_name="$2"
    local timeout="${3:-10}"
    
    local attempt=1
    
    while [[ $attempt -le $MAX_RETRIES ]]; do
        log::info "Checking $service_name (attempt $attempt/$MAX_RETRIES)..."
        
        if curl -s --max-time "$timeout" "$url" >/dev/null 2>&1; then
            log::success "$service_name is healthy"
            return 0
        fi
        
        if [[ $attempt -lt $MAX_RETRIES ]]; then
            log::warn "$service_name check failed, retrying in ${RETRY_DELAY}s..."
            sleep "$RETRY_DELAY"
        fi
        
        ((attempt++))
    done
    
    log::error "$service_name failed health check after $MAX_RETRIES attempts"
    return 1
}

#######################################
# Check Qdrant service health and capabilities
# Returns: 0 if healthy, 1 if failed
#######################################
check_qdrant_health() {
    local qdrant_health_url="$QDRANT_URL/health"
    
    if ! check_service_health "$qdrant_health_url" "Qdrant"; then
        return 1
    fi
    
    # Check collections endpoint
    log::info "Checking Qdrant collections endpoint..."
    if curl -s --max-time 10 "$QDRANT_URL/collections" >/dev/null 2>&1; then
        log::success "Qdrant collections endpoint is accessible"
    else
        log::error "Qdrant collections endpoint is not accessible"
        return 1
    fi
    
    # Get cluster info
    local cluster_info=$(curl -s --max-time 10 "$QDRANT_URL/cluster" 2>/dev/null || echo "{}")
    if echo "$cluster_info" | jq -e '.result' >/dev/null 2>&1; then
        log::info "Qdrant cluster status: $(echo "$cluster_info" | jq -r '.result.peer_id // "standalone"')"
    fi
    
    # Check telemetry endpoint
    local telemetry=$(curl -s --max-time 10 "$QDRANT_URL/telemetry" 2>/dev/null || echo "{}")
    if echo "$telemetry" | jq -e '.result.collections' >/dev/null 2>&1; then
        local collection_count=$(echo "$telemetry" | jq -r '.result.collections // 0')
        log::info "Qdrant collections count: $collection_count"
    fi
    
    return 0
}

#######################################
# Check Ollama service health and model availability
# Returns: 0 if healthy, 1 if failed
#######################################
check_ollama_health() {
    local ollama_health_url="$OLLAMA_URL/api/tags"
    
    if ! check_service_health "$ollama_health_url" "Ollama"; then
        return 1
    fi
    
    # Check model availability
    log::info "Checking embedding model: $DEFAULT_MODEL"
    local models=$(curl -s --max-time 10 "$OLLAMA_URL/api/tags" 2>/dev/null || echo "{}")
    
    if echo "$models" | jq -e '.models' >/dev/null 2>&1; then
        local model_count=$(echo "$models" | jq '.models | length')
        log::info "Ollama has $model_count models available"
        
        # Check if our embedding model is available
        local model_available=$(echo "$models" | jq -r --arg model "$DEFAULT_MODEL" '.models[] | select(.name | startswith($model)) | .name' | head -1)
        
        if [[ -n "$model_available" ]]; then
            log::success "Embedding model '$model_available' is available"
            
            # Get model info
            local model_info=$(echo "$models" | jq -r --arg model "$model_available" '.models[] | select(.name == $model)')
            local model_size=$(echo "$model_info" | jq -r '.size // 0')
            local model_modified=$(echo "$model_info" | jq -r '.modified_at // "unknown"')
            
            log::info "  Model size: $(numfmt --to=iec --suffix=B "$model_size" 2>/dev/null || echo "$model_size bytes")"
            log::info "  Modified: $model_modified"
        else
            log::error "Embedding model '$DEFAULT_MODEL' is not available"
            log::info "Available models:"
            echo "$models" | jq -r '.models[]?.name // empty' | sed 's/^/  - /'
            return 1
        fi
    else
        log::error "Could not retrieve model list from Ollama"
        return 1
    fi
    
    return 0
}

#######################################
# Test embedding generation functionality
# Returns: 0 if successful, 1 if failed
#######################################
test_embedding_generation() {
    log::info "Testing embedding generation..."
    
    local test_text="This is a test sentence for embedding generation."
    local embedding_url="$OLLAMA_URL/api/embeddings"
    
    # Create request payload
    local request_payload=$(jq -n \
        --arg model "$DEFAULT_MODEL" \
        --arg prompt "$test_text" \
        '{model: $model, prompt: $prompt}')
    
    log::debug "Sending embedding request..."
    local response=$(curl -s --max-time 30 \
        -X POST "$embedding_url" \
        -H "Content-Type: application/json" \
        -d "$request_payload" 2>/dev/null || echo "{}")
    
    if echo "$response" | jq -e '.embedding' >/dev/null 2>&1; then
        local embedding_dims=$(echo "$response" | jq '.embedding | length')
        log::success "Embedding generation successful (${embedding_dims}D)"
        
        # Validate embedding vector
        local first_values=$(echo "$response" | jq -r '.embedding[0:3] | @csv')
        log::debug "Sample embedding values: $first_values"
        
        # Check for reasonable values
        local zero_count=$(echo "$response" | jq '[.embedding[] | select(. == 0)] | length')
        if [[ "$zero_count" -gt $((embedding_dims / 2)) ]]; then
            log::warn "Embedding has many zero values ($zero_count/$embedding_dims) - model may not be loaded"
        fi
        
        return 0
    else
        log::error "Embedding generation failed"
        log::debug "Response: $response"
        return 1
    fi
}

#######################################
# Check disk space for embedding operations
# Returns: 0 if sufficient, 1 if insufficient
#######################################
check_disk_space() {
    log::info "Checking disk space..."
    
    # Check /tmp space (used for temporary files)
    local tmp_available=$(df /tmp | tail -1 | awk '{print $4}')
    local tmp_available_gb=$((tmp_available / 1024 / 1024))
    
    log::info "Available space in /tmp: ${tmp_available_gb}GB"
    
    if [[ $tmp_available_gb -lt 1 ]]; then
        log::error "Insufficient disk space in /tmp: ${tmp_available_gb}GB (minimum 1GB recommended)"
        return 1
    fi
    
    # Check app root space
    local app_available=$(df "$APP_ROOT" | tail -1 | awk '{print $4}')
    local app_available_gb=$((app_available / 1024 / 1024))
    
    log::info "Available space in app directory: ${app_available_gb}GB"
    
    if [[ $app_available_gb -lt 2 ]]; then
        log::warn "Low disk space in app directory: ${app_available_gb}GB (minimum 2GB recommended)"
    fi
    
    return 0
}

#######################################
# Check memory usage for parallel processing
# Returns: 0 if sufficient, 1 if insufficient
#######################################
check_memory() {
    log::info "Checking memory availability..."
    
    local total_mem=$(free | awk '/Mem:/ {print $2}')
    local available_mem=$(free | awk '/Mem:/ {print $7}')
    local used_percentage=$(free | awk '/Mem:/ {printf "%.0f", $3/$2 * 100}')
    
    local total_gb=$((total_mem / 1024 / 1024))
    local available_gb=$((available_mem / 1024 / 1024))
    
    log::info "Memory usage: ${used_percentage}% (${available_gb}GB available of ${total_gb}GB total)"
    
    if [[ $used_percentage -gt 85 ]]; then
        log::warn "High memory usage: ${used_percentage}% - parallel processing may be affected"
    fi
    
    if [[ $available_gb -lt 2 ]]; then
        log::error "Low available memory: ${available_gb}GB (minimum 2GB recommended for parallel processing)"
        return 1
    fi
    
    return 0
}

#######################################
# Run comprehensive health check
# Returns: 0 if all checks pass, 1 if any fail
#######################################
run_comprehensive_check() {
    log::info "=== Comprehensive Service Health Check ==="
    local failed_checks=0
    
    # Service health checks
    if ! check_qdrant_health; then
        ((failed_checks++))
    fi
    
    if ! check_ollama_health; then
        ((failed_checks++))
    fi
    
    # Functional tests
    if ! test_embedding_generation; then
        ((failed_checks++))
    fi
    
    # System resource checks
    if ! check_disk_space; then
        ((failed_checks++))
    fi
    
    if ! check_memory; then
        ((failed_checks++))
    fi
    
    echo
    if [[ $failed_checks -eq 0 ]]; then
        log::success "All health checks passed! ✅"
        return 0
    else
        log::error "$failed_checks health check(s) failed ❌"
        return 1
    fi
}

#######################################
# Main function
#######################################
main() {
    local check_type="${1:-comprehensive}"
    
    case "$check_type" in
        qdrant)
            check_qdrant_health
            ;;
        ollama)
            check_ollama_health
            ;;
        embedding)
            test_embedding_generation
            ;;
        resources)
            check_disk_space && check_memory
            ;;
        comprehensive|all)
            run_comprehensive_check
            ;;
        *)
            log::error "Unknown check type: $check_type"
            log::info "Usage: $0 <qdrant|ollama|embedding|resources|comprehensive>"
            return 1
            ;;
    esac
}

# Run main if called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi