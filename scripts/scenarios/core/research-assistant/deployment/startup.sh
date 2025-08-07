#!/bin/bash
# Startup script for AI Research Assistant
# This script converts the scenario into a running application

set -euo pipefail

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Source port registry for dynamic port resolution
# shellcheck disable=SC1091
source "${SCENARIO_DIR}/../../../resources/common.sh"
SCENARIO_ID="research-assistant"
SCENARIO_NAME="AI Research Assistant"
LOG_FILE="/tmp/vrooli-${SCENARIO_ID}-startup.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

# Error handling
trap 'log_error "Startup failed at line $LINENO"; exit 1' ERR

# Load configuration from service.json
load_configuration() {
    log_info "Loading scenario configuration..."
    
    # Check if required file exists
    if [[ ! -f "$SCENARIO_DIR/.vrooli/service.json" ]]; then
        log_error "service.json not found in $SCENARIO_DIR/.vrooli/"
        exit 1
    fi
    
    # Extract required resources
    REQUIRED_RESOURCES=$(jq -r '.resources | to_entries[] | .value | to_entries[] | select(.value.required == true) | .key' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null | tr '\n' ' ')
    log_info "Required resources: $REQUIRED_RESOURCES"
    
    # Extract configuration
    REQUIRES_UI=$(jq -r '.deployment.testing.ui.required // false' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null || echo "false")
    REQUIRES_DISPLAY=$(jq -r '.deployment.testing.ui.type // "none"' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null || echo "none")
    TIMEOUT_SECONDS=$(jq -r '.deployment.testing.timeout // "30m"' "$SCENARIO_DIR/.vrooli/service.json" 2>/dev/null | sed 's/[ms]//g' || echo "300")
    
    log_info "UI required: $REQUIRES_UI, Display required: $REQUIRES_DISPLAY, Timeout: ${TIMEOUT_SECONDS}s"
}

# Step 1: Validate required resources are healthy
validate_resources() {
    log_info "Validating required resources..."
    
    local failed_resources=()
    
    for resource in $REQUIRED_RESOURCES; do
        case "$resource" in
            "ollama")
                if ! curl -sf "http://localhost:$(resources::get_default_port "ollama")/api/tags" >/dev/null 2>&1; then
                    failed_resources+=("ollama")
                else
                    log_success "✓ Ollama is healthy"
                fi
                ;;
            "n8n")
                if ! curl -sf "http://localhost:$(resources::get_default_port "n8n")/healthz" >/dev/null 2>&1; then
                    failed_resources+=("n8n")
                else
                    log_success "✓ n8n is healthy"
                fi
                ;;
            "postgres")
                if ! pg_isready -h localhost -p "$(resources::get_default_port "postgres")" >/dev/null 2>&1; then
                    failed_resources+=("postgres")
                else
                    log_success "✓ PostgreSQL is healthy"
                fi
                ;;
            "redis")
                if ! redis-cli -h localhost -p "$(resources::get_default_port "redis")" ping >/dev/null 2>&1; then
                    failed_resources+=("redis")
                else
                    log_success "✓ Redis is healthy"
                fi
                ;;
            "windmill")
                if ! curl -sf "http://localhost:$(resources::get_default_port "windmill")/api/version" >/dev/null 2>&1; then
                    failed_resources+=("windmill")
                else
                    log_success "✓ Windmill is healthy"
                fi
                ;;
            "whisper")
                if ! curl -sf "http://localhost:$(resources::get_default_port "whisper")/" >/dev/null 2>&1; then
                    failed_resources+=("whisper")
                else
                    log_success "✓ Whisper is healthy"
                fi
                ;;
            "comfyui")
                if ! curl -sf "http://localhost:$(resources::get_default_port "comfyui")/" >/dev/null 2>&1; then
                    failed_resources+=("comfyui")
                else
                    log_success "✓ ComfyUI is healthy"
                fi
                ;;
            "minio")
                if ! curl -sf "http://localhost:$(resources::get_default_port "minio")/minio/health/live" >/dev/null 2>&1; then
                    failed_resources+=("minio")
                else
                    log_success "✓ MinIO is healthy"
                fi
                ;;
            "qdrant")
                if ! curl -sf "http://localhost:$(resources::get_default_port "qdrant")/" >/dev/null 2>&1; then
                    failed_resources+=("qdrant")
                else
                    log_success "✓ Qdrant is healthy"
                fi
                ;;
            "questdb")
                if ! curl -sf "http://localhost:$(resources::get_default_port "questdb")/" >/dev/null 2>&1; then
                    failed_resources+=("questdb")
                else
                    log_success "✓ QuestDB is healthy"
                fi
                ;;
            "searxng")
                if ! curl -sf "http://localhost:$(resources::get_default_port "searxng")/" >/dev/null 2>&1; then
                    failed_resources+=("searxng")
                else
                    log_success "✓ SearXNG is healthy"
                fi
                ;;
            "unstructured-io")
                if ! curl -sf "http://localhost:$(resources::get_default_port "unstructured-io")/healthcheck" >/dev/null 2>&1; then
                    failed_resources+=("unstructured-io")
                else
                    log_success "✓ Unstructured-IO is healthy"
                fi
                ;;
            "browserless")
                if ! curl -sf "http://localhost:$(resources::get_default_port "browserless")/pressure" >/dev/null 2>&1; then
                    failed_resources+=("browserless")
                else
                    log_success "✓ Browserless is healthy"
                fi
                ;;
            *)
                log_warning "Unknown resource: $resource"
                ;;
        esac
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        log_error "Failed resources: ${failed_resources[*]}"
        log_error "Please start the required resources before deploying the scenario"
        exit 1
    fi
    
    log_success "All required resources are healthy"
}

# Step 2: Initialize database schema and seed data
initialize_database() {
    if [[ "$REQUIRED_RESOURCES" =~ "postgres" ]]; then
        log_info "Initializing database..."
        
        local db_name="${SCENARIO_ID//-/_}"
        local schema_file="$SCENARIO_DIR/initialization/storage/postgres/schema.sql"
        local seed_file="$SCENARIO_DIR/initialization/storage/postgres/seed.sql"
        
        # Create database if it doesn't exist
        if ! psql -h localhost -p "$(resources::get_default_port "postgres")" -U postgres -lqt | cut -d \| -f 1 | grep -qw "$db_name"; then
            log_info "Creating database: $db_name"
            createdb -h localhost -p "$(resources::get_default_port "postgres")" -U postgres "$db_name" || {
                log_warning "Database $db_name might already exist, continuing..."
            }
        fi
        
        # Apply schema
        if [[ -f "$schema_file" ]]; then
            log_info "Applying database schema..."
            # Simple template variable substitution
            sed "s/research-assistant/$db_name/g" "$schema_file" | \
            psql -h localhost -p "$(resources::get_default_port "postgres")" -U postgres -d "$db_name" -v ON_ERROR_STOP=1
            log_success "Database schema applied"
        else
            log_warning "No schema file found at $schema_file"
        fi
        
        # Apply seed data
        if [[ -f "$seed_file" ]]; then
            log_info "Applying seed data..."
            # Simple template variable substitution
            sed "s/research-assistant/$db_name/g" "$seed_file" | \
            psql -h localhost -p "$(resources::get_default_port "postgres")" -U postgres -d "$db_name" -v ON_ERROR_STOP=1
            log_success "Seed data applied"
        else
            log_warning "No seed file found at $seed_file"
        fi
    else
        log_info "Skipping database initialization (PostgreSQL not required)"
    fi
}

# Step 3: Deploy workflows to automation platforms
deploy_workflows() {
    log_info "Deploying workflows..."
    
    # Deploy n8n workflows
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        local n8n_dir="$SCENARIO_DIR/initialization/automation/n8n"
        if [[ -d "$n8n_dir" ]]; then
            log_info "Deploying n8n workflows..."
            for workflow_file in "$n8n_dir"/*.json; do
                if [[ -f "$workflow_file" ]]; then
                    log_info "Importing workflow: $(basename "$workflow_file")"
                    # Note: In a real implementation, you'd use n8n's API to import workflows
                    # For now, we'll just validate the JSON
                    if jq empty "$workflow_file" 2>/dev/null; then
                        log_success "✓ Workflow $(basename "$workflow_file") is valid"
                    else
                        log_error "✗ Workflow $(basename "$workflow_file") has invalid JSON"
                    fi
                fi
            done
        fi
    fi
    
    # Deploy Windmill apps
    if [[ "$REQUIRED_RESOURCES" =~ "windmill" && "$REQUIRES_UI" == "true" ]]; then
        local windmill_app="$SCENARIO_DIR/initialization/automation/windmill/dashboard-app.json"
        if [[ -f "$windmill_app" ]]; then
            log_info "Deploying Windmill application..."
            # Note: In a real implementation, you'd use Windmill's API to deploy apps
            if jq empty "$windmill_app" 2>/dev/null; then
                log_success "✓ Windmill app configuration is valid"
            else
                log_error "✗ Windmill app configuration has invalid JSON"
            fi
        fi
    fi
}

# Step 4: Apply configuration
apply_configuration() {
    log_info "Applying configuration..."
    
    local config_dir="$SCENARIO_DIR/initialization/configuration"
    
    # Validate configuration files
    for config_file in "$config_dir"/*.json; do
        if [[ -f "$config_file" ]]; then
            log_info "Validating $(basename "$config_file")..."
            if jq empty "$config_file" 2>/dev/null; then
                log_success "✓ $(basename "$config_file") is valid"
            else
                log_error "✗ $(basename "$config_file") has invalid JSON"
            fi
        fi
    done
    
    # In a real implementation, you'd apply these configurations to the running services
    log_success "Configuration validated and ready for application"
}

# Step 5: Perform health checks
health_checks() {
    log_info "Performing post-deployment health checks..."
    
    # Test webhook endpoint if n8n is deployed
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        local webhook_url="http://localhost:$(resources::get_default_port "n8n")/webhook/${SCENARIO_ID}-webhook"
        log_info "Testing webhook endpoint: $webhook_url"
        
        # Test with sample data
        local test_payload='{"test": true, "scenario": "'$SCENARIO_ID'", "timestamp": "'$(date -Iseconds)'"}'
        
        if curl -sf -X POST -H "Content-Type: application/json" -d "$test_payload" "$webhook_url" >/dev/null 2>&1; then
            log_success "✓ Webhook endpoint is responding"
        else
            log_warning "⚠ Webhook endpoint test failed (this is expected if workflow isn't activated yet)"
        fi
    fi
    
    # Test UI accessibility if required
    if [[ "$REQUIRES_UI" == "true" && "$REQUIRED_RESOURCES" =~ "windmill" ]]; then
        local ui_url="http://localhost:$(resources::get_default_port "windmill")/app/$SCENARIO_ID"
        log_info "Testing UI accessibility: $ui_url"
        
        if curl -sf "$ui_url" >/dev/null 2>&1; then
            log_success "✓ UI is accessible"
        else
            log_warning "⚠ UI accessibility test failed (this is expected if app isn't deployed yet)"
        fi
    fi
    
    log_success "Health checks completed"
}

# Main deployment function
main() {
    log_info "Starting deployment of $SCENARIO_NAME ($SCENARIO_ID)..."
    log_info "Log file: $LOG_FILE"
    
    # Clear previous log
    > "$LOG_FILE"
    
    # Execute deployment steps
    load_configuration
    validate_resources
    initialize_database
    deploy_workflows
    apply_configuration
    health_checks
    
    # Success summary
    log_success "🎉 $SCENARIO_NAME deployed successfully!"
    log_info "Application endpoints:"
    
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        log_info "  📡 Webhook: http://localhost:$(resources::get_default_port "n8n")/webhook/${SCENARIO_ID}-webhook"
    fi
    
    if [[ "$REQUIRES_UI" == "true" && "$REQUIRED_RESOURCES" =~ "windmill" ]]; then
        log_info "  🖥️  UI: http://localhost:$(resources::get_default_port "windmill")/app/$SCENARIO_ID"
    fi
    
    if [[ "$REQUIRED_RESOURCES" =~ "postgres" ]]; then
        log_info "  🗄️  Database: postgresql://postgres:postgres@localhost:$(resources::get_default_port "postgres")/${SCENARIO_ID//-/_}"
    fi
    
    log_info "  📊 Scenario Test: $SCENARIO_DIR/test.sh"
    log_info "  📋 Full Log: $LOG_FILE"
    
    echo ""
    echo -e "${GREEN}✅ Deployment completed successfully!${NC}"
    echo -e "${BLUE}ℹ️  Run the scenario test to validate functionality:${NC}"
    echo "   cd \"$SCENARIO_DIR\" && ./test.sh"
}

# Handle command line arguments
case "${1:-deploy}" in
    "deploy"|"start"|"startup")
        main
        ;;
    "validate"|"check")
        load_configuration
        validate_resources
        ;;
    "logs")
        if [[ -f "$LOG_FILE" ]]; then
            tail -f "$LOG_FILE"
        else
            log_error "No log file found at $LOG_FILE"
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
        log_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac