#!/usr/bin/env bash
################################################################################
# Cline Integration Hub - Connect with other Vrooli resources
# 
# Enables integration with Judge0, other AI resources, and Vrooli services
#
################################################################################

set -euo pipefail

# Integration registry
declare -A INTEGRATIONS
INTEGRATIONS=(
    ["judge0"]="Auto-generate tests with Judge0"
    ["ollama"]="Connect to local Ollama models"
    ["litellm"]="Use LiteLLM proxy for unified LLM access"
    ["n8n"]="Integrate with n8n workflows"
    ["postgres"]="Store conversation history in PostgreSQL"
    ["redis"]="Cache responses with Redis"
    ["qdrant"]="Vector search for code embeddings"
)

#######################################
# Show available integrations
#######################################
cline::integrate::list() {
    log::info "Available Cline integrations:"
    echo ""
    
    for integration in "${!INTEGRATIONS[@]}"; do
        local desc="${INTEGRATIONS[$integration]}"
        local status="❌ Not connected"
        
        # Check if integration is active
        if cline::integrate::is_active "$integration"; then
            status="✅ Connected"
        fi
        
        printf "  %-12s - %-40s [%s]\n" "$integration" "$desc" "$status"
    done
}

#######################################
# Check if integration is active
#######################################
cline::integrate::is_active() {
    local integration="${1:-}"
    local config_file="${CLINE_CONFIG_DIR}/integrations/${integration}.conf"
    
    [[ -f "$config_file" ]] && grep -q "enabled=true" "$config_file"
}

#######################################
# Enable an integration
#######################################
cline::integrate::enable() {
    local integration="${1:-}"
    
    if [[ -z "$integration" ]]; then
        log::error "Please specify an integration to enable"
        cline::integrate::list
        return 1
    fi
    
    if [[ -z "${INTEGRATIONS[$integration]:-}" ]]; then
        log::error "Unknown integration: $integration"
        cline::integrate::list
        return 1
    fi
    
    log::info "Enabling $integration integration..."
    
    # Create integration config directory
    mkdir -p "${CLINE_CONFIG_DIR}/integrations"
    
    case "$integration" in
        judge0)
            cline::integrate::setup_judge0
            ;;
        ollama)
            cline::integrate::setup_ollama
            ;;
        litellm)
            cline::integrate::setup_litellm
            ;;
        n8n)
            cline::integrate::setup_n8n
            ;;
        postgres)
            cline::integrate::setup_postgres
            ;;
        redis)
            cline::integrate::setup_redis
            ;;
        qdrant)
            cline::integrate::setup_qdrant
            ;;
        *)
            log::error "Integration not yet implemented: $integration"
            return 1
            ;;
    esac
}

#######################################
# Disable an integration
#######################################
cline::integrate::disable() {
    local integration="${1:-}"
    
    if [[ -z "$integration" ]]; then
        log::error "Please specify an integration to disable"
        cline::integrate::list
        return 1
    fi
    
    local config_file="${CLINE_CONFIG_DIR}/integrations/${integration}.conf"
    
    if [[ -f "$config_file" ]]; then
        log::info "Disabling $integration integration..."
        echo "enabled=false" > "$config_file"
        log::success "✓ Integration disabled: $integration"
    else
        log::warn "Integration not configured: $integration"
    fi
}

#######################################
# Setup Judge0 integration
#######################################
cline::integrate::setup_judge0() {
    log::info "Setting up Judge0 integration for automatic test generation..."
    
    # Check if Judge0 is running
    if vrooli resource status judge0 2>/dev/null | grep -q "Running"; then
        log::success "✓ Judge0 is running"
    else
        log::info "Starting Judge0..."
        vrooli resource judge0 manage start --wait || {
            log::error "Failed to start Judge0"
            return 1
        }
    fi
    
    # Create integration config
    cat > "${CLINE_CONFIG_DIR}/integrations/judge0.conf" << EOF
enabled=true
endpoint=http://localhost:2358
auto_test=false
test_on_save=false
languages=python,javascript,typescript,go,rust
EOF
    
    log::success "✓ Judge0 integration enabled"
    log::info "You can now use: cline integrate execute judge0 test <file>"
}

#######################################
# Setup Ollama integration
#######################################
cline::integrate::setup_ollama() {
    log::info "Setting up Ollama integration..."
    
    # Check if Ollama is running
    if timeout 2 curl -sf http://localhost:11434/api/version &>/dev/null; then
        log::success "✓ Ollama is running"
        
        # Save integration config
        cat > "${CLINE_CONFIG_DIR}/integrations/ollama.conf" << EOF
enabled=true
endpoint=http://localhost:11434
models=$(ollama list 2>/dev/null | tail -n +2 | awk '{print $1}' | tr '\n' ',' | sed 's/,$//')
default_model=llama3.2:3b
EOF
        
        log::success "✓ Ollama integration enabled"
    else
        log::error "Ollama is not running. Please start it first."
        return 1
    fi
}

#######################################
# Setup LiteLLM integration
#######################################
cline::integrate::setup_litellm() {
    log::info "Setting up LiteLLM integration..."
    
    # Check if LiteLLM is running
    if vrooli resource status litellm 2>/dev/null | grep -q "Running"; then
        log::success "✓ LiteLLM is running"
        
        cat > "${CLINE_CONFIG_DIR}/integrations/litellm.conf" << EOF
enabled=true
endpoint=http://localhost:11435
api_base=/v1
model_fallback=true
EOF
        
        log::success "✓ LiteLLM integration enabled"
    else
        log::warn "LiteLLM is not running. Start it with: vrooli resource litellm manage start"
        return 1
    fi
}

#######################################
# Setup n8n integration
#######################################
cline::integrate::setup_n8n() {
    log::info "Setting up n8n workflow integration..."
    
    # Check if n8n is running
    if vrooli resource status n8n 2>/dev/null | grep -q "Running"; then
        log::success "✓ n8n is running"
        
        cat > "${CLINE_CONFIG_DIR}/integrations/n8n.conf" << EOF
enabled=true
endpoint=http://localhost:5678
webhook_url=http://localhost:5678/webhook/cline
auto_trigger=false
EOF
        
        log::success "✓ n8n integration enabled"
        log::info "Configure webhooks in n8n to trigger on code changes"
    else
        log::warn "n8n is not running. Start it with: vrooli resource n8n manage start"
        return 1
    fi
}

#######################################
# Setup PostgreSQL integration
#######################################
cline::integrate::setup_postgres() {
    log::info "Setting up PostgreSQL for conversation history..."
    
    # Check if PostgreSQL is running
    if vrooli resource status postgres 2>/dev/null | grep -q "Running"; then
        log::success "✓ PostgreSQL is running"
        
        cat > "${CLINE_CONFIG_DIR}/integrations/postgres.conf" << EOF
enabled=true
host=localhost
port=5433
database=cline_history
user=postgres
table=conversations
max_history=1000
EOF
        
        log::success "✓ PostgreSQL integration enabled"
        log::info "Conversation history will be stored in PostgreSQL"
    else
        log::warn "PostgreSQL is not running. Start it with: vrooli resource postgres manage start"
        return 1
    fi
}

#######################################
# Setup Redis integration
#######################################
cline::integrate::setup_redis() {
    log::info "Setting up Redis for response caching..."
    
    # Check if Redis is running
    if vrooli resource status redis 2>/dev/null | grep -q "Running"; then
        log::success "✓ Redis is running"
        
        cat > "${CLINE_CONFIG_DIR}/integrations/redis.conf" << EOF
enabled=true
host=localhost
port=6380
db=0
ttl=3600
max_cache_size=100MB
EOF
        
        log::success "✓ Redis integration enabled"
        log::info "Responses will be cached for faster access"
    else
        log::warn "Redis is not running. Start it with: vrooli resource redis manage start"
        return 1
    fi
}

#######################################
# Setup Qdrant integration
#######################################
cline::integrate::setup_qdrant() {
    log::info "Setting up Qdrant for vector search..."
    
    # Check if Qdrant is running
    if vrooli resource status qdrant 2>/dev/null | grep -q "Running"; then
        log::success "✓ Qdrant is running"
        
        cat > "${CLINE_CONFIG_DIR}/integrations/qdrant.conf" << EOF
enabled=true
endpoint=http://localhost:6333
collection=cline_code_embeddings
vector_size=768
distance=cosine
EOF
        
        log::success "✓ Qdrant integration enabled"
        log::info "Code embeddings will be stored for semantic search"
    else
        log::warn "Qdrant is not running. Start it with: vrooli resource qdrant manage start"
        return 1
    fi
}

#######################################
# Execute integration-specific commands
#######################################
cline::integrate::execute() {
    local integration="${1:-}"
    local action="${2:-}"
    shift 2 || true
    
    if [[ -z "$integration" ]] || [[ -z "$action" ]]; then
        log::error "Usage: cline integrate execute <integration> <action> [args...]"
        return 1
    fi
    
    if ! cline::integrate::is_active "$integration"; then
        log::error "Integration not enabled: $integration"
        log::info "Enable it with: cline integrate enable $integration"
        return 1
    fi
    
    case "$integration" in
        judge0)
            cline::integrate::judge0_execute "$action" "$@"
            ;;
        *)
            log::error "Execute not implemented for: $integration"
            return 1
            ;;
    esac
}

#######################################
# Execute Judge0 commands
#######################################
cline::integrate::judge0_execute() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        test)
            local file="${1:-}"
            if [[ -z "$file" ]] || [[ ! -f "$file" ]]; then
                log::error "Please specify a valid file to test"
                return 1
            fi
            
            log::info "Generating tests for $file with Judge0..."
            
            # Extract code content
            local code=$(cat "$file" | base64 -w 0)
            local language_id=71  # Python default
            
            # Detect language from extension
            case "${file##*.}" in
                js) language_id=63 ;;
                ts) language_id=74 ;;
                py) language_id=71 ;;
                go) language_id=60 ;;
                rs) language_id=73 ;;
            esac
            
            # Submit to Judge0
            local response=$(curl -sf -X POST http://localhost:2358/submissions \
                -H "Content-Type: application/json" \
                -d "{\"source_code\":\"$code\",\"language_id\":$language_id}")
            
            if [[ -n "$response" ]]; then
                log::success "✓ Test generation submitted to Judge0"
                echo "$response" | jq -r '.token' 2>/dev/null || echo "$response"
            else
                log::error "Failed to submit to Judge0"
                return 1
            fi
            ;;
        *)
            log::error "Unknown Judge0 action: $action"
            return 1
            ;;
    esac
}

# Export functions
export -f cline::integrate::list
export -f cline::integrate::enable
export -f cline::integrate::disable
export -f cline::integrate::is_active
export -f cline::integrate::execute
export -f cline::integrate::setup_judge0
export -f cline::integrate::setup_ollama
export -f cline::integrate::setup_litellm
export -f cline::integrate::setup_n8n
export -f cline::integrate::setup_postgres
export -f cline::integrate::setup_redis
export -f cline::integrate::setup_qdrant
export -f cline::integrate::judge0_execute