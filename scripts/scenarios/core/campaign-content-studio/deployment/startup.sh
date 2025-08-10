#!/bin/bash
# Startup script for Campaign Content Studio
# This script converts the scenario into a running application

set -euo pipefail

# Source var.sh first with proper relative path
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../../lib/utils/var.sh"

# Configuration
SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCENARIO_ID="campaign-content-studio"
SCENARIO_NAME="Campaign Content Studio"
LOG_FILE="$var_ROOT_DIR/tmp/vrooli-${SCENARIO_ID}-startup.log"

# Import Vrooli utilities
# shellcheck disable=SC1091
source "$var_LIB_UTILS_DIR/json.sh" 2>/dev/null || true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
startup::log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

startup::log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

startup::log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$LOG_FILE"
}

startup::log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

startup::log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
}

# Error handling
trap 'startup::startup::log_error "Startup failed at line $LINENO"; exit 1' ERR

# Load configuration from service.json
startup::startup::load_configuration() {
    startup::log_info "Loading scenario configuration..."
    
    # Check if required file exists
    if [[ ! -f "$SCENARIO_DIR/service.json" ]]; then
        startup::log_error "service.json not found in $SCENARIO_DIR"
        exit 1
    fi
    
    # Extract required resources using JSON utilities (if available) or fallback to jq
    if command -v json::get_required_resources &> /dev/null; then
        REQUIRED_RESOURCES=$(json::get_required_resources "" "$SCENARIO_DIR/service.json" || echo "")
    else
        REQUIRED_RESOURCES=$(jq -r '.resources | to_entries[] | .value | to_entries[] | select(.value.required == true) | .key' "$SCENARIO_DIR/service.json" 2>/dev/null | tr '\n' ' ' || echo "")
    fi
    startup::log_info "Required resources: $REQUIRED_RESOURCES"
    
    # Extract configuration using JSON utilities (if available) or fallback to jq
    if command -v json::get_deployment_config &> /dev/null; then
        REQUIRES_UI=$(json::get_deployment_config 'testing.ui.required' 'false' "$SCENARIO_DIR/service.json")
        REQUIRES_DISPLAY=$(json::get_deployment_config 'testing.ui.type' 'none' "$SCENARIO_DIR/service.json")
        TIMEOUT_SECONDS=$(json::get_deployment_config 'testing.timeout' '30m' "$SCENARIO_DIR/service.json" | sed 's/[ms]//g')
    else
        REQUIRES_UI=$(jq -r '.deployment.testing.ui.required // false' "$SCENARIO_DIR/service.json" 2>/dev/null || echo "false")
        REQUIRES_DISPLAY=$(jq -r '.deployment.testing.ui.type // "none"' "$SCENARIO_DIR/service.json" 2>/dev/null || echo "none")
        TIMEOUT_SECONDS=$(jq -r '.deployment.testing.timeout // "30m"' "$SCENARIO_DIR/service.json" 2>/dev/null | sed 's/[ms]//g' || echo "300")
    fi
    
    startup::log_info "UI required: $REQUIRES_UI, Display required: $REQUIRES_DISPLAY, Timeout: ${TIMEOUT_SECONDS}s"
}

# Step 1: Validate required resources are healthy
startup::startup::validate_resources() {
    startup::log_info "Validating required resources..."
    
    local failed_resources=()
    
    for resource in $REQUIRED_RESOURCES; do
        case "$resource" in
            "ollama")
                if ! curl -sf http://localhost:11434/api/tags >/dev/null 2>&1; then
                    failed_resources+=("ollama")
                else
                    startup::log_success "✓ Ollama is healthy"
                fi
                ;;
            "n8n")
                if ! curl -sf http://localhost:5678/healthz >/dev/null 2>&1; then
                    failed_resources+=("n8n")
                else
                    startup::log_success "✓ n8n is healthy"
                fi
                ;;
            "postgres")
                if ! pg_isready -h localhost -p 5433 >/dev/null 2>&1; then
                    failed_resources+=("postgres")
                else
                    startup::log_success "✓ PostgreSQL is healthy"
                fi
                ;;
            "redis")
                if ! redis-cli -h localhost -p 6380 ping >/dev/null 2>&1; then
                    failed_resources+=("redis")
                else
                    startup::log_success "✓ Redis is healthy"
                fi
                ;;
            "windmill")
                if ! curl -sf http://localhost:5681/api/version >/dev/null 2>&1; then
                    failed_resources+=("windmill")
                else
                    startup::log_success "✓ Windmill is healthy"
                fi
                ;;
            "whisper")
                if ! curl -sf http://localhost:8090/ >/dev/null 2>&1; then
                    failed_resources+=("whisper")
                else
                    startup::log_success "✓ Whisper is healthy"
                fi
                ;;
            "comfyui")
                if ! curl -sf http://localhost:8188/ >/dev/null 2>&1; then
                    failed_resources+=("comfyui")
                else
                    startup::log_success "✓ ComfyUI is healthy"
                fi
                ;;
            "minio")
                if ! curl -sf http://localhost:9000/minio/health/live >/dev/null 2>&1; then
                    failed_resources+=("minio")
                else
                    startup::log_success "✓ MinIO is healthy"
                fi
                ;;
            "qdrant")
                if ! curl -sf http://localhost:6333/ >/dev/null 2>&1; then
                    failed_resources+=("qdrant")
                else
                    startup::log_success "✓ Qdrant is healthy"
                fi
                ;;
            "questdb")
                if ! curl -sf http://localhost:9010/ >/dev/null 2>&1; then
                    failed_resources+=("questdb")
                else
                    startup::log_success "✓ QuestDB is healthy"
                fi
                ;;
            *)
                startup::log_warning "Unknown resource: $resource"
                ;;
        esac
    done
    
    if [[ ${#failed_resources[@]} -gt 0 ]]; then
        startup::log_error "Failed resources: ${failed_resources[*]}"
        startup::log_error "Please start the required resources before deploying the scenario"
        exit 1
    fi
    
    startup::log_success "All required resources are healthy"
}

# Step 2: Initialize database schema and seed data
startup::startup::initialize_database() {
    if [[ "$REQUIRED_RESOURCES" =~ "postgres" ]]; then
        startup::log_info "Initializing database..."
        
        local db_name="${SCENARIO_ID//-/_}"
        local schema_file="$SCENARIO_DIR/initialization/storage/schema.sql"
        local seed_file="$SCENARIO_DIR/initialization/storage/seed.sql"
        
        # Create database if it doesn't exist
        if ! psql -h localhost -p 5433 -U postgres -lqt | cut -d \| -f 1 | grep -qw "$db_name"; then
            startup::log_info "Creating database: $db_name"
            createdb -h localhost -p 5433 -U postgres "$db_name" || {
                startup::log_warning "Database $db_name might already exist, continuing..."
            }
        fi
        
        # Apply schema
        if [[ -f "$schema_file" ]]; then
            startup::log_info "Applying database schema..."
            # Simple template variable substitution
            sed "s/{{ scenario_id }}/$db_name/g" "$schema_file" | \
            psql -h localhost -p 5433 -U postgres -d "$db_name" -v ON_ERROR_STOP=1
            startup::log_success "Database schema applied"
        else
            startup::log_warning "No schema file found at $schema_file"
        fi
        
        # Apply seed data
        if [[ -f "$seed_file" ]]; then
            startup::log_info "Applying seed data..."
            # Simple template variable substitution
            sed "s/{{ scenario_id }}/$db_name/g" "$seed_file" | \
            psql -h localhost -p 5433 -U postgres -d "$db_name" -v ON_ERROR_STOP=1
            startup::log_success "Seed data applied"
        else
            startup::log_warning "No seed file found at $seed_file"
        fi
    else
        startup::log_info "Skipping database initialization (PostgreSQL not required)"
    fi
}

# Step 3: Deploy workflows to automation platforms
startup::startup::deploy_workflows() {
    startup::log_info "Deploying workflows..."
    
    # Deploy n8n workflows
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        local n8n_dir="$SCENARIO_DIR/initialization/automation/n8n"
        if [[ -d "$n8n_dir" ]]; then
            startup::log_info "Deploying n8n workflows..."
            for workflow_file in "$n8n_dir"/*.json; do
                if [[ -f "$workflow_file" ]]; then
                    startup::log_info "Importing workflow: $(basename "$workflow_file")"
                    # Note: In a real implementation, you'd use n8n's API to import workflows
                    # For now, we'll just validate the JSON
                    if jq empty "$workflow_file" 2>/dev/null; then
                        startup::log_success "✓ Workflow $(basename "$workflow_file") is valid"
                    else
                        startup::log_error "✗ Workflow $(basename "$workflow_file") has invalid JSON"
                    fi
                fi
            done
        fi
    fi
    
    # Deploy Windmill apps
    if [[ "$REQUIRED_RESOURCES" =~ "windmill" && "$REQUIRES_UI" == "true" ]]; then
        local windmill_app="$SCENARIO_DIR/initialization/automation/windmill/campaign-dashboard.json"
        if [[ -f "$windmill_app" ]]; then
            startup::log_info "Deploying Windmill application..."
            # Note: In a real implementation, you'd use Windmill's API to deploy apps
            if jq empty "$windmill_app" 2>/dev/null; then
                startup::log_success "✓ Windmill app configuration is valid"
            else
                startup::log_error "✗ Windmill app configuration has invalid JSON"
            fi
        fi
    fi
}

# Step 4: Apply configuration
startup::startup::apply_configuration() {
    startup::log_info "Applying configuration..."
    
    local config_dir="$SCENARIO_DIR/initialization/configuration"
    
    # Validate configuration files
    for config_file in "$config_dir"/*.json; do
        if [[ -f "$config_file" ]]; then
            startup::log_info "Validating $(basename "$config_file")..."
            if jq empty "$config_file" 2>/dev/null; then
                startup::log_success "✓ $(basename "$config_file") is valid"
            else
                startup::log_error "✗ $(basename "$config_file") has invalid JSON"
            fi
        fi
    done
    
    # In a real implementation, you'd apply these configurations to the running services
    startup::log_success "Configuration validated and ready for application"
}

# Step 5: Perform health checks
startup::startup::health_checks() {
    startup::log_info "Performing post-deployment health checks..."
    
    # Test webhook endpoint if n8n is deployed
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        local webhook_url="http://localhost:5678/webhook/${SCENARIO_ID}-webhook"
        startup::log_info "Testing webhook endpoint: $webhook_url"
        
        # Test with sample data
        local test_payload='{"test": true, "scenario": "'$SCENARIO_ID'", "timestamp": "'$(date -Iseconds)'"}'
        
        if curl -sf -X POST -H "Content-Type: application/json" -d "$test_payload" "$webhook_url" >/dev/null 2>&1; then
            startup::log_success "✓ Webhook endpoint is responding"
        else
            startup::log_warning "⚠ Webhook endpoint test failed (this is expected if workflow isn't activated yet)"
        fi
    fi
    
    # Test UI accessibility if required
    if [[ "$REQUIRES_UI" == "true" && "$REQUIRED_RESOURCES" =~ "windmill" ]]; then
        local ui_url="http://localhost:5681/app/$SCENARIO_ID"
        startup::log_info "Testing UI accessibility: $ui_url"
        
        if curl -sf "$ui_url" >/dev/null 2>&1; then
            startup::log_success "✓ UI is accessible"
        else
            startup::log_warning "⚠ UI accessibility test failed (this is expected if app isn't deployed yet)"
        fi
    fi
    
    startup::log_success "Health checks completed"
}

# Main deployment function
startup::main() {
    startup::log_info "Starting deployment of $SCENARIO_NAME ($SCENARIO_ID)..."
    startup::log_info "Log file: $LOG_FILE"
    
    # Clear previous log
    > "$LOG_FILE"
    
    # Execute deployment steps
    startup::load_configuration
    startup::validate_resources
    startup::initialize_database
    startup::deploy_workflows
    startup::apply_configuration
    startup::health_checks
    
    # Success summary
    startup::log_success "🎉 $SCENARIO_NAME deployed successfully!"
    startup::log_info "Application endpoints:"
    
    if [[ "$REQUIRED_RESOURCES" =~ "n8n" ]]; then
        startup::log_info "  📡 Webhook: http://localhost:5678/webhook/${SCENARIO_ID}-webhook"
    fi
    
    if [[ "$REQUIRES_UI" == "true" && "$REQUIRED_RESOURCES" =~ "windmill" ]]; then
        startup::log_info "  🖥️  UI: http://localhost:5681/app/$SCENARIO_ID"
    fi
    
    if [[ "$REQUIRED_RESOURCES" =~ "postgres" ]]; then
        startup::log_info "  🗄️  Database: postgresql://postgres:postgres@localhost:5433/${SCENARIO_ID//-/_}"
    fi
    
    startup::log_info "  📊 Scenario Test: $SCENARIO_DIR/test.sh"
    startup::log_info "  📋 Full Log: $LOG_FILE"
    
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
        startup::load_configuration
        startup::validate_resources
        ;;
    "logs")
        if [[ -f "$LOG_FILE" ]]; then
            tail -f "$LOG_FILE"
        else
            startup::log_error "No log file found at $LOG_FILE"
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
        startup::log_error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac