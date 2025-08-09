#!/bin/bash
# Automation import for Smart File Photo Manager
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VROOLI_ROOT="$(cd "$SCENARIO_ROOT/../../.." && pwd)"

# Load environment variables
source "$VROOLI_ROOT/scripts/resources/lib/resource-helper.sh"

# Configuration
N8N_HOST="localhost"
N8N_PORT="5680"
WINDMILL_HOST="localhost"
WINDMILL_PORT="8002"

# Get default ports if available
if command -v "resources::get_default_port" &> /dev/null; then
    N8N_PORT=$(resources::get_default_port "n8n" || echo "5680")
    WINDMILL_PORT=$(resources::get_default_port "windmill" || echo "8002")
fi

N8N_URL="http://$N8N_HOST:$N8N_PORT"
WINDMILL_URL="http://$WINDMILL_HOST:$WINDMILL_PORT"

# Paths
N8N_WORKFLOWS_DIR="$SCENARIO_ROOT/initialization/automation/n8n"
WINDMILL_SCRIPTS_DIR="$SCENARIO_ROOT/initialization/automation/windmill/scripts"
WINDMILL_APPS_DIR="$SCENARIO_ROOT/initialization/automation/windmill/apps"

log_info() {
    echo "[INFO] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo "[ERROR] $(date '+%Y-%m-%d %H:%M:%S') - $1" >&2
}

log_success() {
    echo "[SUCCESS] $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Wait for services to be ready
wait_for_n8n() {
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for n8n on port $N8N_PORT..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$N8N_URL/healthz" >/dev/null 2>&1; then
            log_success "n8n is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    log_error "n8n failed to become ready"
    return 1
}

wait_for_windmill() {
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for Windmill on port $WINDMILL_PORT..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$WINDMILL_URL/api/version" >/dev/null 2>&1; then
            log_success "Windmill is ready"
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    log_error "Windmill failed to become ready"
    return 1
}

# Import n8n workflow
import_n8n_workflow() {
    local workflow_file="$1"
    local workflow_name
    workflow_name=$(basename "$workflow_file" .json)
    
    if [ ! -f "$workflow_file" ]; then
        log_error "Workflow file not found: $workflow_file"
        return 1
    fi
    
    log_info "Importing n8n workflow: $workflow_name"
    
    # Check if workflow already exists
    local existing_workflows
    existing_workflows=$(curl -s "$N8N_URL/api/v1/workflows" 2>/dev/null || echo "[]")
    
    if echo "$existing_workflows" | grep -q "\"name\":\"$workflow_name\""; then
        log_info "Workflow '$workflow_name' already exists, updating..."
        
        # Get existing workflow ID
        local workflow_id
        workflow_id=$(echo "$existing_workflows" | jq -r ".data[] | select(.name==\"$workflow_name\") | .id" 2>/dev/null || echo "")
        
        if [ -n "$workflow_id" ]; then
            # Update existing workflow
            if curl -s -X PUT "$N8N_URL/api/v1/workflows/$workflow_id" \
                -H "Content-Type: application/json" \
                -d @"$workflow_file" >/dev/null; then
                log_success "Updated workflow: $workflow_name"
            else
                log_error "Failed to update workflow: $workflow_name"
                return 1
            fi
        else
            log_error "Could not find workflow ID for: $workflow_name"
            return 1
        fi
    else
        # Import new workflow
        if curl -s -X POST "$N8N_URL/api/v1/workflows" \
            -H "Content-Type: application/json" \
            -d @"$workflow_file" >/dev/null; then
            log_success "Imported workflow: $workflow_name"
        else
            log_error "Failed to import workflow: $workflow_name"
            return 1
        fi
    fi
}

# Import all n8n workflows
import_n8n_workflows() {
    log_info "Importing n8n workflows..."
    
    if [ ! -d "$N8N_WORKFLOWS_DIR" ]; then
        log_error "n8n workflows directory not found: $N8N_WORKFLOWS_DIR"
        return 1
    fi
    
    local workflow_count=0
    
    for workflow_file in "$N8N_WORKFLOWS_DIR"/*.json; do
        [ -f "$workflow_file" ] || continue
        
        if import_n8n_workflow "$workflow_file"; then
            ((workflow_count++))
        fi
    done
    
    log_success "Imported $workflow_count n8n workflows"
}

# Import Windmill script
import_windmill_script() {
    local script_file="$1"
    local script_name
    script_name=$(basename "$script_file" .ts)
    
    if [ ! -f "$script_file" ]; then
        log_error "Script file not found: $script_file"
        return 1
    fi
    
    log_info "Importing Windmill script: $script_name"
    
    # Create script payload
    local script_content
    script_content=$(cat "$script_file")
    
    local payload
    payload=$(jq -n \
        --arg path "f/file_manager/$script_name" \
        --arg summary "File manager automation script" \
        --arg content "$script_content" \
        --arg language "typescript" \
        '{
            path: $path,
            summary: $summary,
            content: $content,
            language: $language,
            schema: {
                "$schema": "https://json-schema.org/draft/2020-12/schema",
                "type": "object",
                "properties": {}
            }
        }')
    
    # Import script
    if curl -s -X POST "$WINDMILL_URL/api/w/file_manager/scripts" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $WINDMILL_TOKEN" \
        -d "$payload" >/dev/null; then
        log_success "Imported script: $script_name"
    else
        log_error "Failed to import script: $script_name"
        return 1
    fi
}

# Import Windmill app
import_windmill_app() {
    local app_file="$1"
    local app_name
    app_name=$(basename "$app_file" .json)
    
    if [ ! -f "$app_file" ]; then
        log_error "App file not found: $app_file"
        return 1
    fi
    
    log_info "Importing Windmill app: $app_name"
    
    # Import app
    if curl -s -X POST "$WINDMILL_URL/api/w/file_manager/apps" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $WINDMILL_TOKEN" \
        -d @"$app_file" >/dev/null; then
        log_success "Imported app: $app_name"
    else
        log_error "Failed to import app: $app_name"
        return 1
    fi
}

# Setup Windmill workspace
setup_windmill_workspace() {
    log_info "Setting up Windmill workspace: file_manager"
    
    # Check if workspace exists
    local workspaces
    workspaces=$(curl -s "$WINDMILL_URL/api/workspaces" 2>/dev/null || echo "[]")
    
    if echo "$workspaces" | grep -q '"id":"file_manager"'; then
        log_info "Workspace 'file_manager' already exists"
    else
        # Create workspace
        local workspace_payload='{
            "id": "file_manager",
            "name": "File Manager",
            "domain": "file-manager"
        }'
        
        if curl -s -X POST "$WINDMILL_URL/api/workspaces" \
            -H "Content-Type: application/json" \
            -d "$workspace_payload" >/dev/null; then
            log_success "Created Windmill workspace: file_manager"
        else
            log_error "Failed to create Windmill workspace"
            return 1
        fi
    fi
}

# Import all Windmill scripts
import_windmill_scripts() {
    log_info "Importing Windmill scripts..."
    
    if [ ! -d "$WINDMILL_SCRIPTS_DIR" ]; then
        log_error "Windmill scripts directory not found: $WINDMILL_SCRIPTS_DIR"
        return 1
    fi
    
    local script_count=0
    
    for script_file in "$WINDMILL_SCRIPTS_DIR"/*.ts; do
        [ -f "$script_file" ] || continue
        
        if import_windmill_script "$script_file"; then
            ((script_count++))
        fi
    done
    
    log_success "Imported $script_count Windmill scripts"
}

# Import all Windmill apps
import_windmill_apps() {
    log_info "Importing Windmill apps..."
    
    if [ ! -d "$WINDMILL_APPS_DIR" ]; then
        log_error "Windmill apps directory not found: $WINDMILL_APPS_DIR"
        return 1
    fi
    
    local app_count=0
    
    for app_file in "$WINDMILL_APPS_DIR"/*.json; do
        [ -f "$app_file" ] || continue
        
        if import_windmill_app "$app_file"; then
            ((app_count++))
        fi
    done
    
    log_success "Imported $app_count Windmill apps"
}

# Verify imports
verify_n8n_imports() {
    log_info "Verifying n8n workflow imports..."
    
    local workflows
    workflows=$(curl -s "$N8N_URL/api/v1/workflows" 2>/dev/null || echo '{"data": []}')
    
    local workflow_count
    workflow_count=$(echo "$workflows" | jq '.data | length' 2>/dev/null || echo "0")
    
    if [ "$workflow_count" -gt 0 ]; then
        log_success "n8n verification passed: $workflow_count workflows active"
    else
        log_error "n8n verification failed: no workflows found"
        return 1
    fi
}

verify_windmill_imports() {
    log_info "Verifying Windmill imports..."
    
    local scripts
    scripts=$(curl -s "$WINDMILL_URL/api/w/file_manager/scripts/list" 2>/dev/null || echo "[]")
    
    local script_count
    script_count=$(echo "$scripts" | jq 'length' 2>/dev/null || echo "0")
    
    if [ "$script_count" -gt 0 ]; then
        log_success "Windmill verification passed: $script_count scripts imported"
    else
        log_info "Windmill verification: no scripts found (may be normal if none exist)"
    fi
}

# Main execution
main() {
    log_info "Importing automations for File Manager..."
    
    # Wait for services to be ready
    wait_for_n8n
    wait_for_windmill
    
    # Setup Windmill workspace
    setup_windmill_workspace
    
    # Import n8n workflows
    if [ -d "$N8N_WORKFLOWS_DIR" ]; then
        import_n8n_workflows
        verify_n8n_imports
    else
        log_info "No n8n workflows directory found, skipping n8n import"
    fi
    
    # Import Windmill components
    if [ -d "$WINDMILL_SCRIPTS_DIR" ]; then
        import_windmill_scripts
    else
        log_info "No Windmill scripts directory found, skipping script import"
    fi
    
    if [ -d "$WINDMILL_APPS_DIR" ]; then
        import_windmill_apps
    else
        log_info "No Windmill apps directory found, skipping app import"
    fi
    
    # Verify Windmill imports
    verify_windmill_imports
    
    log_success "Automation import completed successfully"
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi