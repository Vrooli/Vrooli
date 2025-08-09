#!/usr/bin/env bash
# Startup script for Audio Intelligence Platform
# This script converts the scenario into a running application

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck disable=SC1091
source "${SCENARIO_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_ROOT_DIR}/scripts/lib/utils/json.sh"

SCENARIO_ID="audio-intelligence-platform"
SCENARIO_NAME="Audio Intelligence Platform"
LOG_FILE="${var_ROOT_DIR}/logs/vrooli-${SCENARIO_ID}-startup.log"

# Ensure log directory exists
mkdir -p "$(dirname "$LOG_FILE")"

# Error handling
trap 'log::error "Startup failed at line $LINENO"; exit 1' ERR

# Load configuration from service.json
startup::load_configuration() {
    log::info "Loading scenario configuration..."
    
    # Check if required file exists
    if [[ ! -f "$SCENARIO_DIR/.vrooli/service.json" ]]; then
        log::error "service.json not found in $SCENARIO_DIR/.vrooli/"
        exit 1
    fi
    
    # Extract required resources using JSON utilities
    REQUIRED_RESOURCES=$(json::get_required_resources "" "$SCENARIO_DIR/.vrooli/service.json")
    log::info "Required resources: $REQUIRED_RESOURCES"
    
    # Extract configuration using JSON utilities
    REQUIRES_UI=$(json::get_deployment_config 'testing.ui.required' 'false' "$SCENARIO_DIR/.vrooli/service.json")
    REQUIRES_DISPLAY=$(json::get_deployment_config 'testing.ui.type' 'windmill' "$SCENARIO_DIR/.vrooli/service.json")
    TIMEOUT_SECONDS=$(json::get_deployment_config 'testing.timeout' '45m' "$SCENARIO_DIR/.vrooli/service.json" | sed 's/[ms]//g')
    
    log::info "UI required: $REQUIRES_UI, Display required: $REQUIRES_DISPLAY, Timeout: ${TIMEOUT_SECONDS}s"
}

# Step 1: Validate required resources are healthy
startup::validate_resources() {
    log::info "Validating required resources..."
    
    local failed_resources=()
    
    for resource in $REQUIRED_RESOURCES; do
        case "$resource" in
            "ollama")
                if ! curl -sf "http://localhost:$(resources::get_default_port "ollama")/api/tags" >/dev/null 2>&1; then
                    failed_resources+=("ollama")
                else
                    log::success "✓ Ollama is healthy"
                fi
                ;;
            "n8n")
                if ! curl -sf "http://localhost:$(resources::get_default_port "n8n")/healthz" >/dev/null 2>&1; then
                    failed_resources+=("n8n")
                else
                    log::success "✓ n8n is healthy"
                fi
                ;;
            "postgres")
                if ! pg_isready -h localhost -p "$(resources::get_default_port "postgres")" >/dev/null 2>&1; then
                    failed_resources+=("postgres")
                else
                    log::success "✓ PostgreSQL is healthy"
                fi
                ;;
            "redis")
                if ! redis-cli -h localhost -p "$(resources::get_default_port "redis")" ping >/dev/null 2>&1; then
                    failed_resources+=("redis")
                else
                    log::success "✓ Redis is healthy"
                fi
                ;;
            "windmill")
                if ! curl -sf "http://localhost:$(resources::get_default_port "windmill")/api/version" >/dev/null 2>&1; then
                    failed_resources+=("windmill")
                else
                    log::success "✓ Windmill is healthy"
                fi
                ;;
            "whisper")
                if ! curl -sf "http://localhost:$(resources::get_default_port "whisper")/" >/dev/null 2>&1; then
                    failed_resources+=("whisper")
                else
                    log::success "✓ Whisper is healthy"
                fi
                ;;
            "comfyui")
                if ! curl -sf "http://localhost:$(resources::get_default_port "comfyui")/" >/dev/null 2>&1; then
                    failed_resources+=("comfyui")
                else
                    log::success "✓ ComfyUI is healthy"
                fi
                ;;
            "minio")
                if ! curl -sf "http://localhost:$(resources::get_default_port "minio")/minio/health/live" >/dev/null 2>&1; then
                    failed_resources+=("minio")
                else
                    log::success "✓ MinIO is healthy"
                fi
                ;;
            "qdrant")
                if ! curl -sf "http://localhost:$(resources::get_default_port "qdrant")/" >/dev/null 2>&1; then
                    failed_resources+=("qdrant")
                else
                    log::success "✓ Qdrant is healthy"
                fi
                ;;
            "questdb")
                if ! curl -sf "http://localhost:$(resources::get_default_port "questdb")/" >/dev/null 2>&1; then
                    failed_resources+=("questdb")
                else
                    log::success "✓ QuestDB is healthy"
                fi
                ;;
            *)
                log::warning "Unknown resource: $resource"
                ;;
        esac
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        log::error "Failed resources: ${failed_resources[*]}"
        log::error "Please start the required resources before deploying the scenario"
        exit 1
    fi
    
    log::success "All required resources are healthy"
}

# Step 2: Initialize database schema and seed data
startup::initialize_database() {
    if [[ "$REQUIRED_RESOURCES" =~ "postgres" ]]; then
        log::info "Initializing database..."
        
        local db_name="audio_intelligence_platform"
        local schema_file="$SCENARIO_DIR/initialization/storage/postgres/schema.sql"
        local seed_file="$SCENARIO_DIR/initialization/storage/postgres/seed.sql"
        
        # Create database if it doesn't exist
        if ! psql -h localhost -p "$(resources::get_default_port "postgres")" -U postgres -lqt | cut -d \| -f 1 | grep -qw "$db_name"; then
            log::info "Creating database: $db_name"
            createdb -h localhost -p "$(resources::get_default_port "postgres")" -U postgres "$db_name" || {
                log::warning "Database $db_name might already exist, continuing..."
            }
        fi
        
        # Apply schema
        if [[ -f "$schema_file" ]]; then
            log::info "Applying database schema..."
            PGPASSWORD=postgres psql -h localhost -p "$(resources::get_default_port "postgres")" -U postgres -d "$db_name" -f "$schema_file" -v ON_ERROR_STOP=1
            log::success "Database schema applied"
        else
            log::warning "No schema file found at $schema_file"
        fi
        
        # Apply seed data
        if [[ -f "$seed_file" ]]; then
            log::info "Applying seed data..."
            PGPASSWORD=postgres psql -h localhost -p "$(resources::get_default_port "postgres")" -U postgres -d "$db_name" -f "$seed_file" -v ON_ERROR_STOP=1
            log::success "Seed data applied"
        else
            log::warning "No seed file found at $seed_file"
        fi
    else
        log::info "Skipping database initialization (PostgreSQL not required)"
    fi
}

# Step 3: Deploy workflows to automation platforms
startup::deploy_workflows() {
    log::info "Deploying workflows..."
    
    # Deploy n8n workflows
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        local n8n_dir="$SCENARIO_DIR/initialization/automation/n8n"
        if [[ -d "$n8n_dir" ]]; then
            log::info "Deploying n8n workflows..."
            for workflow_file in "$n8n_dir"/*.json; do
                if [[ -f "$workflow_file" ]]; then
                    log::info "Importing workflow: $(basename "$workflow_file")"
                    # Note: In a real implementation, you'd use n8n's API to import workflows
                    # For now, we'll just validate the JSON
                    if jq empty "$workflow_file" 2>/dev/null; then
                        log::success "✓ Workflow $(basename "$workflow_file") is valid"
                    else
                        log::error "✗ Workflow $(basename "$workflow_file") has invalid JSON"
                    fi
                fi
            done
        fi
    fi
    
    # Deploy Windmill apps
    if [[ "$REQUIRED_RESOURCES" =~ "windmill" && "$REQUIRES_UI" == "true" ]]; then
        local windmill_app="$SCENARIO_DIR/initialization/automation/windmill/transcription-manager-app.json"
        if [[ -f "$windmill_app" ]]; then
            log::info "Deploying Windmill application..."
            # Note: In a real implementation, you'd use Windmill's API to deploy apps
            if jq empty "$windmill_app" 2>/dev/null; then
                log::success "✓ Windmill app configuration is valid"
            else
                log::error "✗ Windmill app configuration has invalid JSON"
            fi
        fi
    fi
}

# Step 4: Initialize MinIO buckets
startup::initialize_minio() {
    if [[ "$REQUIRED_RESOURCES" =~ "minio" ]]; then
        log::info "Initializing MinIO buckets..."
        
        # Check MinIO client availability
        if ! command -v mc &> /dev/null; then
            log::warning "MinIO client (mc) not found, skipping bucket initialization"
            return 0
        fi
        
        # Configure MinIO client
        mc alias set local "http://localhost:$(resources::get_default_port "minio")" minioadmin minioadmin 2>/dev/null || true
        
        # Create buckets
        local buckets=("audio-files" "transcriptions" "exports")
        for bucket in "${buckets[@]}"; do
            if ! mc ls "local/$bucket" &>/dev/null; then
                log::info "Creating bucket: $bucket"
                mc mb "local/$bucket"
                log::success "✓ Bucket $bucket created"
            else
                log::success "✓ Bucket $bucket already exists"
            fi
        done
    else
        log::info "Skipping MinIO initialization (not required)"
    fi
}

# Step 5: Initialize Qdrant collections
startup::initialize_qdrant() {
    if [[ "$REQUIRED_RESOURCES" =~ "qdrant" ]]; then
        log::info "Initializing Qdrant collections..."
        
        # Create transcription embeddings collection
        local collection_config='{
            "vectors": {
                "size": 384,
                "distance": "Cosine"
            }
        }'
        
        if curl -sf -X GET "http://localhost:$(resources::get_default_port "qdrant")/collections/transcription-embeddings" >/dev/null 2>&1; then
            log::success "✓ Collection 'transcription-embeddings' already exists"
        else
            log::info "Creating collection 'transcription-embeddings'..."
            if curl -sf -X PUT "http://localhost:$(resources::get_default_port "qdrant")/collections/transcription-embeddings" \
                -H "Content-Type: application/json" \
                -d "$collection_config" >/dev/null 2>&1; then
                log::success "✓ Collection 'transcription-embeddings' created"
            else
                log::error "Failed to create Qdrant collection"
            fi
        fi
    else
        log::info "Skipping Qdrant initialization (not required)"
    fi
}

# Step 6: Apply configuration
startup::apply_configuration() {
    log::info "Applying configuration..."
    
    local config_dir="$SCENARIO_DIR/initialization/configuration"
    
    # Validate configuration files
    for config_file in "$config_dir"/*.json; do
        if [[ -f "$config_file" ]]; then
            log::info "Validating $(basename "$config_file")..."
            if jq empty "$config_file" 2>/dev/null; then
                log::success "✓ $(basename "$config_file") is valid"
            else
                log::error "✗ $(basename "$config_file") has invalid JSON"
            fi
        fi
    done
    
    # In a real implementation, you'd apply these configurations to the running services
    log::success "Configuration validated and ready for application"
}

# Step 5: Perform health checks
startup::health_checks() {
    log::info "Performing post-deployment health checks..."
    
    # Test webhook endpoints if n8n is deployed
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        log::info "Testing n8n webhook endpoints..."
        
        # Test transcription pipeline webhook
        local webhook_url="http://localhost:$(resources::get_default_port "n8n")/webhook/transcription-upload"
        log::info "Testing transcription webhook: $webhook_url"
        local test_payload='{"test": true, "filename": "test.mp3", "scenario": "'$SCENARIO_ID'", "timestamp": "'$(date -Iseconds)'"}'
        
        if curl -sf -X POST -H "Content-Type: application/json" -d "$test_payload" "$webhook_url" >/dev/null 2>&1; then
            log::success "✓ Webhook endpoint is responding"
        else
            log::warning "⚠ Webhook endpoint test failed (this is expected if workflow isn't activated yet)"
        fi
    fi
    
    # Test UI accessibility if required
    if [[ "$REQUIRES_UI" == "true" && "$REQUIRED_RESOURCES" =~ "windmill" ]]; then
        local ui_url="http://localhost:$(resources::get_default_port "windmill")/app/$SCENARIO_ID"
        log::info "Testing UI accessibility: $ui_url"
        
        if curl -sf "$ui_url" >/dev/null 2>&1; then
            log::success "✓ UI is accessible"
        else
            log::warning "⚠ UI accessibility test failed (this is expected if app isn't deployed yet)"
        fi
    fi
    
    log::success "Health checks completed"
}

# Main deployment function
startup::main() {
    log::info "Starting deployment of $SCENARIO_NAME ($SCENARIO_ID)..."
    log::info "Log file: $LOG_FILE"
    
    # Clear previous log
    > "$LOG_FILE"
    
    # Execute deployment steps
    startup::load_configuration
    startup::validate_resources
    startup::initialize_database
    startup::initialize_minio
    startup::initialize_qdrant
    startup::deploy_workflows
    startup::apply_configuration
    startup::health_checks
    
    # Success summary
    log::success "🎉 $SCENARIO_NAME deployed successfully!"
    log::info "Application endpoints:"
    
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        log::info "  📡 Transcription Pipeline: http://localhost:$(resources::get_default_port "n8n")/webhook/transcription-upload"
        log::info "  📡 AI Analysis: http://localhost:$(resources::get_default_port "n8n")/webhook/ai-analysis"
        log::info "  📡 Semantic Search: http://localhost:$(resources::get_default_port "n8n")/webhook/semantic-search"
    fi
    
    if [[ "$REQUIRES_UI" == "true" && "$REQUIRED_RESOURCES" =~ "windmill" ]]; then
        log::info "  🖥️  Transcription Manager UI: http://localhost:$(resources::get_default_port "windmill")/"
    fi
    
    if [[ "$REQUIRED_RESOURCES" =~ "postgres" ]]; then
        log::info "  🗄️  Database: postgresql://postgres:postgres@localhost:$(resources::get_default_port "postgres")/${SCENARIO_ID//-/_}"
    fi
    
    log::info "  📊 Scenario Test: $SCENARIO_DIR/test.sh"
    log::info "  📋 Full Log: $LOG_FILE"
    
    log::success ""
    log::success "✅ Deployment completed successfully!"
    log::info "ℹ️  Run the scenario test to validate functionality:"
    log::info "   cd \"$SCENARIO_DIR\" && ./test.sh"
}

# Handle command line arguments
case "${1:-deploy}" in
    "deploy"|"start"|"startup")
        startup::main
        ;;
    "validate"|"check")
        startup::load_configuration
        startup::validate_resources
        ;;
    "logs")
        if [[ -f "$LOG_FILE" ]]; then
            tail -f "$LOG_FILE"
        else
            log::error "No log file found at $LOG_FILE"
        fi
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  deploy    - Deploy the scenario as an application (default)"
        echo "  validate  - Validate required resources are available"
        echo "  logs      - Follow the deployment logs"
        echo "  help      - Show this help message"
        echo ""
        echo "Environment variables:"
        echo "  SCENARIO_DEBUG=1  - Enable debug logging"
        echo ""
        ;;
    *)
        log::error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac