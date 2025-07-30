#!/usr/bin/env bash
# Resource-Specific Mock Configurations
# Provides pre-configured mock setups for common resource testing scenarios

# Source mock helpers if not already loaded
if ! declare -f mock::set_response >/dev/null 2>&1; then
    source "$(dirname "${BASH_SOURCE[0]}")/mock_helpers.bash"
fi

#######################################
# AI Resources
#######################################

# Ollama mock setup
mock::ollama::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "ollama" "running"
            mock::http::set_endpoint_state "http://localhost:11434" "healthy"
            mock::set_curl_response "http://localhost:11434/api/tags" \
                '{"models":[{"name":"llama3.1:8b","size":4000000000,"modified_at":"2024-01-01T00:00:00Z"},{"name":"mistral:latest","size":3500000000}]}'
            mock::set_curl_response "http://localhost:11434/api/version" \
                '{"version":"0.1.45"}'
            ;;
        "no_models")
            mock::docker::set_container_state "ollama" "running"
            mock::http::set_endpoint_state "http://localhost:11434" "healthy"
            mock::set_curl_response "http://localhost:11434/api/tags" '{"models":[]}'
            ;;
        "pulling_model")
            mock::docker::set_container_state "ollama" "running"
            mock::http::set_endpoint_state "http://localhost:11434" "healthy"
            mock::docker::set_exec_response "Pulling model llama3.1:8b... 45%" 0
            ;;
    esac
}

# Whisper mock setup
mock::whisper::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "whisper" "running"
            mock::http::set_endpoint_state "http://localhost:8090" "healthy"
            mock::set_curl_response "http://localhost:8090/health" '{"status":"ready","model":"whisper-1"}'
            ;;
        "processing")
            mock::docker::set_container_state "whisper" "running"
            mock::http::set_endpoint_state "http://localhost:8090" "healthy"
            mock::set_curl_response "http://localhost:8090/transcribe" \
                '{"text":"This is a test transcription","language":"en","duration":5.2}'
            ;;
    esac
}

# ComfyUI mock setup
mock::comfyui::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "comfyui" "running"
            mock::http::set_endpoint_state "http://localhost:5679" "healthy"
            mock::set_curl_response "http://localhost:5679/api/system_info" \
                '{"python_version":"3.11","comfyui_version":"1.0.0","models_loaded":2}'
            ;;
    esac
}

#######################################
# Automation Resources
#######################################

# n8n mock setup
mock::n8n::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "n8n" "running"
            mock::http::set_endpoint_state "http://localhost:5678" "healthy"
            mock::set_curl_response "http://localhost:5678/healthz" '{"status":"ok"}'
            mock::set_curl_response "http://localhost:5678/version" \
                '{"version":"1.19.0","versionCli":"1.19.0"}'
            ;;
        "workflows_active")
            mock::n8n::setup "healthy"
            mock::set_curl_response "http://localhost:5678/api/v1/workflows" \
                '[{"id":"1","name":"Test Workflow","active":true,"nodes":5}]'
            ;;
    esac
}

# Node-RED mock setup
mock::node-red::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "node-red" "running"
            mock::http::set_endpoint_state "http://localhost:1880" "healthy"
            mock::set_curl_response "http://localhost:1880/flows" '[]'
            mock::set_curl_response "http://localhost:1880/settings" \
                '{"httpNodeRoot":"/","version":"3.1.0","context":{"default":"memory"}}'
            ;;
        "flows_deployed")
            mock::node-red::setup "healthy"
            mock::set_curl_response "http://localhost:1880/flows" \
                '[{"id":"123","type":"tab","label":"Flow 1"},{"id":"456","type":"http in","url":"/test"}]'
            ;;
    esac
}

# Huginn mock setup
mock::huginn::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "huginn" "running"
            mock::http::set_endpoint_state "http://localhost:4111" "healthy"
            mock::set_curl_response "http://localhost:4111/users/sign_in" "" 0
            ;;
        "agents_running")
            mock::huginn::setup "healthy"
            mock::set_curl_response "http://localhost:4111/agents" \
                '[{"id":1,"name":"Website Monitor","type":"WebsiteAgent","last_check":"2024-01-01T00:00:00Z"}]'
            ;;
    esac
}

# Windmill mock setup
mock::windmill::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "windmill" "running"
            mock::http::set_endpoint_state "http://localhost:5681" "healthy"
            mock::set_curl_response "http://localhost:5681/api/version" '{"version":"1.200.0"}'
            ;;
    esac
}

#######################################
# Agent Resources
#######################################

# Browserless mock setup
mock::browserless::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "browserless" "running"
            mock::http::set_endpoint_state "http://localhost:4110" "healthy"
            mock::set_curl_response "http://localhost:4110/pressure" \
                '{"pressure":{"cpu":0.2,"memory":0.4},"queued":0,"running":0}'
            mock::set_curl_response "http://localhost:4110/config" \
                '{"timeout":30000,"maxConcurrentSessions":10}'
            ;;
        "busy")
            mock::docker::set_container_state "browserless" "running"
            mock::http::set_endpoint_state "http://localhost:4110" "healthy"
            mock::set_curl_response "http://localhost:4110/pressure" \
                '{"pressure":{"cpu":0.9,"memory":0.8},"queued":5,"running":10}'
            ;;
    esac
}

# Agent-S2 mock setup
mock::agent-s2::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "agent-s2" "running"
            mock::http::set_endpoint_state "http://localhost:4113" "healthy"
            mock::set_curl_response "http://localhost:4113/health" '{"status":"ready","version":"0.1.0"}'
            mock::set_curl_response "http://localhost:4113/api/capabilities" \
                '{"screenshot":true,"click":true,"type":true,"scroll":true}'
            ;;
        "task_running")
            mock::agent-s2::setup "healthy"
            mock::set_curl_response "http://localhost:4113/ai/task" \
                '{"task_id":"123","status":"running","progress":45}'
            ;;
    esac
}

#######################################
# Storage Resources
#######################################

# MinIO mock setup
mock::minio::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "minio" "running"
            mock::http::set_endpoint_state "http://localhost:9000" "healthy"
            mock::http::set_endpoint_state "http://localhost:9001" "healthy"
            mock::set_curl_response "http://localhost:9000/minio/health/live" '{"status":"ok"}'
            ;;
        "buckets_created")
            mock::minio::setup "healthy"
            mock::docker::set_exec_response "Bucket 'vrooli-assets' created successfully" 0
            ;;
    esac
}

# Qdrant mock setup
mock::qdrant::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "qdrant" "running"
            mock::http::set_endpoint_state "http://localhost:6333" "healthy"
            mock::set_curl_response "http://localhost:6333/health" '{"status":"ok","version":"1.7.0"}'
            mock::set_curl_response "http://localhost:6333/collections" \
                '{"result":{"collections":[]},"status":"ok"}'
            ;;
        "collections_exist")
            mock::qdrant::setup "healthy"
            mock::set_curl_response "http://localhost:6333/collections" \
                '{"result":{"collections":[{"name":"documents"},{"name":"embeddings"}]},"status":"ok"}'
            ;;
    esac
}

# QuestDB mock setup
mock::questdb::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "questdb" "running"
            mock::http::set_endpoint_state "http://localhost:9009" "healthy"
            mock::set_curl_response "http://localhost:9009/exec?query=SELECT%201" \
                '{"query":"SELECT 1","columns":[{"name":"1","type":"INT"}],"data":[[1]]}'
            ;;
    esac
}

# Vault mock setup
mock::vault::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "vault" "running"
            mock::http::set_endpoint_state "http://localhost:8200" "healthy"
            mock::set_curl_response "http://localhost:8200/v1/sys/health" \
                '{"initialized":true,"sealed":false,"standby":false,"version":"1.15.0"}'
            ;;
        "sealed")
            mock::docker::set_container_state "vault" "running"
            mock::http::set_endpoint_state "http://localhost:8200" "healthy"
            mock::set_curl_response "http://localhost:8200/v1/sys/health" \
                '{"initialized":true,"sealed":true,"standby":false,"version":"1.15.0"}' 503
            ;;
    esac
}

#######################################
# Search Resources
#######################################

# SearXNG mock setup
mock::searxng::setup() {
    local state="${1:-healthy}"
    
    case "$state" in
        "healthy")
            mock::docker::set_container_state "searxng" "running"
            mock::http::set_endpoint_state "http://localhost:8100" "healthy"
            mock::set_curl_response "http://localhost:8100/config" \
                '{"version":"1.0.0","engines_enabled":["google","duckduckgo","bing"]}'
            ;;
        "search_results")
            mock::searxng::setup "healthy"
            mock::set_curl_response "http://localhost:8100/search?q=test&format=json" \
                '{"results":[{"title":"Test Result","url":"https://example.com","content":"Test content"}]}'
            ;;
    esac
}

#######################################
# Utility functions
#######################################

# Setup multiple resources at once
mock::resources::setup_many() {
    local state="${1:-healthy}"
    shift
    
    for resource in "$@"; do
        if declare -f "mock::${resource}::setup" >/dev/null 2>&1; then
            "mock::${resource}::setup" "$state"
        else
            # Fallback to generic setup
            mock::resource::setup "$resource" "installed_running"
        fi
    done
}

# Setup all resources in a category
mock::category::setup() {
    local category="$1"
    local state="${2:-healthy}"
    
    case "$category" in
        "ai")
            mock::resources::setup_many "$state" ollama whisper comfyui
            ;;
        "automation")
            mock::resources::setup_many "$state" n8n node-red huginn windmill
            ;;
        "agents")
            mock::resources::setup_many "$state" browserless agent-s2
            ;;
        "storage")
            mock::resources::setup_many "$state" minio qdrant questdb vault
            ;;
        "search")
            mock::resources::setup_many "$state" searxng
            ;;
    esac
}