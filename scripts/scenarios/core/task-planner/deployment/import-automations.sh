#!/bin/bash

# Task Planner - Automation Import Script
# Imports n8n workflows and Windmill apps/scripts

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"; }
success() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $*${NC}"; }
warn() { echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $*${NC}"; }
error() { echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $*${NC}" >&2; }

# Service configuration
N8N_HOST="${N8N_HOST:-localhost}"
N8N_PORT="${N8N_PORT:-5679}"
N8N_BASE_URL="http://${N8N_HOST}:${N8N_PORT}"

WINDMILL_HOST="${WINDMILL_HOST:-localhost}"
WINDMILL_PORT="${WINDMILL_PORT:-8001}"
WINDMILL_BASE_URL="http://${WINDMILL_HOST}:${WINDMILL_PORT}"
WINDMILL_WORKSPACE="task_planner"

# File paths
N8N_WORKFLOWS_DIR="${SCENARIO_DIR}/initialization/automation/n8n"
WINDMILL_SCRIPTS_DIR="${SCENARIO_DIR}/initialization/automation/windmill/scripts"
WINDMILL_APPS_DIR="${SCENARIO_DIR}/initialization/automation/windmill"

# Wait for services to be ready
wait_for_n8n() {
    log "Waiting for n8n to be ready..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "${N8N_BASE_URL}/healthz" > /dev/null 2>&1; then
            success "n8n is ready"
            return 0
        fi
        
        warn "Waiting for n8n... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    error "n8n did not become ready within expected time"
    return 1
}

wait_for_windmill() {
    log "Waiting for Windmill to be ready..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "${WINDMILL_BASE_URL}/api/version" > /dev/null 2>&1; then
            success "Windmill is ready"
            return 0
        fi
        
        warn "Waiting for Windmill... (attempt $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    error "Windmill did not become ready within expected time"
    return 1
}

# Import n8n workflows
import_n8n_workflows() {
    log "Importing n8n workflows..."
    
    if [[ ! -d "$N8N_WORKFLOWS_DIR" ]]; then
        error "n8n workflows directory not found: $N8N_WORKFLOWS_DIR"
        return 1
    fi
    
    local workflows_imported=0
    local workflows_skipped=0
    local workflows_failed=0
    
    # Find all workflow files
    while IFS= read -r -d '' workflow_file; do
        local workflow_name
        workflow_name=$(basename "$workflow_file" .json)
        
        log "Processing workflow: $workflow_name"
        
        # Check if workflow already exists
        local existing_workflows
        existing_workflows=$(curl -s "${N8N_BASE_URL}/api/v1/workflows" 2>/dev/null || echo "[]")
        
        if echo "$existing_workflows" | jq -e --arg name "$workflow_name" '.data[]? | select(.name == $name)' > /dev/null 2>&1; then
            warn "Workflow '$workflow_name' already exists, skipping"
            ((workflows_skipped++))
            continue
        fi
        
        # Import workflow
        local import_response
        import_response=$(curl -s -X POST "${N8N_BASE_URL}/api/v1/workflows/import" \
            -H "Content-Type: application/json" \
            -d @"$workflow_file" 2>/dev/null) || {
            error "Failed to import workflow: $workflow_name"
            ((workflows_failed++))
            continue
        }
        
        # Check import result
        if echo "$import_response" | jq -e '.data.id' > /dev/null 2>&1; then
            local workflow_id
            workflow_id=$(echo "$import_response" | jq -r '.data.id')
            success "Imported workflow: $workflow_name (ID: $workflow_id)"
            ((workflows_imported++))
        else
            error "Failed to import workflow: $workflow_name"
            echo "$import_response" | jq '.' 2>/dev/null || echo "$import_response"
            ((workflows_failed++))
        fi
        
    done < <(find "$N8N_WORKFLOWS_DIR" -name "*.json" -print0)
    
    log "n8n workflow import summary:"
    log "  ✅ Imported: $workflows_imported"
    log "  ⏭️  Skipped: $workflows_skipped" 
    log "  ❌ Failed: $workflows_failed"
    
    if [[ $workflows_failed -gt 0 ]]; then
        return 1
    fi
    
    success "n8n workflows imported successfully"
}

# Create Windmill workspace if it doesn't exist
setup_windmill_workspace() {
    log "Setting up Windmill workspace: $WINDMILL_WORKSPACE"
    
    # Check if workspace exists
    local workspaces
    workspaces=$(curl -s "${WINDMILL_BASE_URL}/api/workspaces" 2>/dev/null || echo "[]")
    
    if echo "$workspaces" | jq -e --arg ws "$WINDMILL_WORKSPACE" '.[] | select(.id == $ws)' > /dev/null 2>&1; then
        log "Workspace '$WINDMILL_WORKSPACE' already exists"
        return 0
    fi
    
    # Create workspace
    local workspace_data='{
        "id": "'$WINDMILL_WORKSPACE'",
        "name": "Task Planner",
        "owner": "admin"
    }'
    
    local create_response
    create_response=$(curl -s -X POST "${WINDMILL_BASE_URL}/api/workspaces" \
        -H "Content-Type: application/json" \
        -d "$workspace_data" 2>/dev/null) || {
        error "Failed to create Windmill workspace"
        return 1
    }
    
    success "Created Windmill workspace: $WINDMILL_WORKSPACE"
}

# Import Windmill scripts
import_windmill_scripts() {
    log "Importing Windmill TypeScript scripts..."
    
    if [[ ! -d "$WINDMILL_SCRIPTS_DIR" ]]; then
        error "Windmill scripts directory not found: $WINDMILL_SCRIPTS_DIR"
        return 1
    fi
    
    local scripts_imported=0
    local scripts_skipped=0
    local scripts_failed=0
    
    # Find all TypeScript script files
    while IFS= read -r -d '' script_file; do
        local script_name
        script_name=$(basename "$script_file" .ts)
        
        log "Processing script: $script_name"
        
        # Check if script already exists
        local existing_scripts
        existing_scripts=$(curl -s "${WINDMILL_BASE_URL}/api/w/${WINDMILL_WORKSPACE}/scripts" 2>/dev/null || echo "[]")
        
        if echo "$existing_scripts" | jq -e --arg path "f/${WINDMILL_WORKSPACE}/${script_name}" '.[] | select(.path == $path)' > /dev/null 2>&1; then
            warn "Script '$script_name' already exists, updating"
        fi
        
        # Read script content
        local script_content
        script_content=$(cat "$script_file")
        
        # Create script data
        local script_data
        script_data=$(jq -n \
            --arg path "f/${WINDMILL_WORKSPACE}/${script_name}" \
            --arg summary "Task Planner - $script_name" \
            --arg content "$script_content" \
            --arg language "typescript" \
            '{
                path: $path,
                summary: $summary,
                description: ("Generated script for Task Planner: " + $path),
                content: $content,
                language: $language,
                schema: {
                    "$schema": "https://json-schema.org/draft/2020-12/schema",
                    "type": "object",
                    "properties": {},
                    "required": []
                }
            }')
        
        # Import script
        local import_response
        import_response=$(curl -s -X POST "${WINDMILL_BASE_URL}/api/w/${WINDMILL_WORKSPACE}/scripts" \
            -H "Content-Type: application/json" \
            -d "$script_data" 2>/dev/null) || {
            error "Failed to import script: $script_name"
            ((scripts_failed++))
            continue
        }
        
        # Check import result  
        if echo "$import_response" | jq -e '.path' > /dev/null 2>&1; then
            local script_path
            script_path=$(echo "$import_response" | jq -r '.path')
            success "Imported script: $script_name -> $script_path"
            ((scripts_imported++))
        else
            error "Failed to import script: $script_name"
            echo "$import_response" | jq '.' 2>/dev/null || echo "$import_response"
            ((scripts_failed++))
        fi
        
    done < <(find "$WINDMILL_SCRIPTS_DIR" -name "*.ts" -print0)
    
    log "Windmill script import summary:"
    log "  ✅ Imported: $scripts_imported"
    log "  ⏭️  Skipped: $scripts_skipped"
    log "  ❌ Failed: $scripts_failed"
    
    if [[ $scripts_failed -gt 0 ]]; then
        return 1
    fi
    
    success "Windmill scripts imported successfully"
}

# Import Windmill apps
import_windmill_apps() {
    log "Importing Windmill applications..."
    
    local apps_imported=0
    local apps_skipped=0
    local apps_failed=0
    
    # Find all app definition files
    while IFS= read -r -d '' app_file; do
        local app_name
        app_name=$(basename "$app_file" .json)
        
        # Skip files in subdirectories (like scripts/)
        if [[ "$app_file" != "$WINDMILL_APPS_DIR"/*.json ]]; then
            continue
        fi
        
        log "Processing app: $app_name"
        
        # Check if app already exists
        local existing_apps
        existing_apps=$(curl -s "${WINDMILL_BASE_URL}/api/w/${WINDMILL_WORKSPACE}/apps" 2>/dev/null || echo "[]")
        
        if echo "$existing_apps" | jq -e --arg path "$app_name" '.[] | select(.path == $path)' > /dev/null 2>&1; then
            warn "App '$app_name' already exists, updating"
        fi
        
        # Read app definition
        local app_content
        app_content=$(cat "$app_file")
        
        # Create app data with proper structure
        local app_data
        app_data=$(echo "$app_content" | jq --arg path "$app_name" '{
            path: $path,
            summary: (.summary // "Task Planner Dashboard"),
            value: .value,
            policy: {
                on_behalf_of: null,
                on_behalf_of_email: null
            }
        }')
        
        # Import app
        local import_response
        import_response=$(curl -s -X POST "${WINDMILL_BASE_URL}/api/w/${WINDMILL_WORKSPACE}/apps" \
            -H "Content-Type: application/json" \
            -d "$app_data" 2>/dev/null) || {
            error "Failed to import app: $app_name"
            ((apps_failed++))
            continue
        }
        
        # Check import result
        if echo "$import_response" | jq -e '.path' > /dev/null 2>&1; then
            local app_path
            app_path=$(echo "$import_response" | jq -r '.path')
            success "Imported app: $app_name -> $app_path"
            ((apps_imported++))
        else
            error "Failed to import app: $app_name"
            echo "$import_response" | jq '.' 2>/dev/null || echo "$import_response"
            ((apps_failed++))
        fi
        
    done < <(find "$WINDMILL_APPS_DIR" -maxdepth 1 -name "*.json" -print0)
    
    log "Windmill app import summary:"
    log "  ✅ Imported: $apps_imported"
    log "  ⏭️  Skipped: $apps_skipped"
    log "  ❌ Failed: $apps_failed"
    
    if [[ $apps_failed -gt 0 ]]; then
        return 1
    fi
    
    success "Windmill apps imported successfully"
}

# Verify automation setup
verify_automation_setup() {
    log "Verifying automation setup..."
    
    # Check n8n workflows
    local n8n_workflows
    n8n_workflows=$(curl -s "${N8N_BASE_URL}/api/v1/workflows" 2>/dev/null || echo '{"data": []}')
    local n8n_count
    n8n_count=$(echo "$n8n_workflows" | jq '.data | length')
    
    success "n8n: $n8n_count workflows available"
    
    # List n8n workflows
    echo "$n8n_workflows" | jq -r '.data[]? | "  📋 " + .name + " (ID: " + (.id | tostring) + ")"'
    
    # Check Windmill scripts
    local windmill_scripts
    windmill_scripts=$(curl -s "${WINDMILL_BASE_URL}/api/w/${WINDMILL_WORKSPACE}/scripts" 2>/dev/null || echo "[]")
    local windmill_script_count
    windmill_script_count=$(echo "$windmill_scripts" | jq 'length')
    
    success "Windmill: $windmill_script_count scripts available"
    
    # List Windmill scripts
    echo "$windmill_scripts" | jq -r '.[]? | "  🔧 " + .path + " (" + .language + ")"'
    
    # Check Windmill apps
    local windmill_apps
    windmill_apps=$(curl -s "${WINDMILL_BASE_URL}/api/w/${WINDMILL_WORKSPACE}/apps" 2>/dev/null || echo "[]")
    local windmill_app_count
    windmill_app_count=$(echo "$windmill_apps" | jq 'length')
    
    success "Windmill: $windmill_app_count apps available"
    
    # List Windmill apps
    echo "$windmill_apps" | jq -r '.[]? | "  📱 " + .path + " - " + (.summary // "No description")'
}

# Display automation information
show_automation_info() {
    log "Automation Services Information:"
    echo ""
    echo "🔄 n8n Workflow Engine:"
    echo "  📍 URL: $N8N_BASE_URL"
    echo "  🎛️  Admin UI: $N8N_BASE_URL"
    echo "  🔗 Webhooks: $N8N_BASE_URL/webhook/"
    echo ""
    echo "⚡ Windmill Automation Platform:"
    echo "  📍 URL: $WINDMILL_BASE_URL"
    echo "  🏢 Workspace: $WINDMILL_WORKSPACE"
    echo "  🎯 Dashboard: $WINDMILL_BASE_URL/apps/get/task-dashboard"
    echo "  🔧 Scripts: $WINDMILL_BASE_URL/w/$WINDMILL_WORKSPACE/scripts"
    echo ""
    echo "🔗 Integration Endpoints:"
    echo "  📝 Parse Text: POST $N8N_BASE_URL/webhook/parse-text"
    echo "  🔬 Research Task: POST $N8N_BASE_URL/webhook/research-task"  
    echo "  🚀 Implement Task: POST $N8N_BASE_URL/webhook/implement-task"
    echo "  📊 Status Update: POST $N8N_BASE_URL/webhook/status-update"
    echo ""
}

# Main execution
main() {
    log "Importing automation workflows and apps..."
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        error "jq is required for automation import. Please install it first."
        return 1
    fi
    
    # Setup steps
    wait_for_n8n
    wait_for_windmill
    
    # Import n8n workflows
    import_n8n_workflows
    
    # Setup Windmill
    setup_windmill_workspace
    import_windmill_scripts
    import_windmill_apps
    
    # Verify setup
    verify_automation_setup
    show_automation_info
    
    success "Automation import completed successfully!"
    
    # Export connection details for other scripts
    cat > "${SCRIPT_DIR}/.automation_connection" << EOF
export N8N_BASE_URL="$N8N_BASE_URL"
export WINDMILL_BASE_URL="$WINDMILL_BASE_URL"
export WINDMILL_WORKSPACE="$WINDMILL_WORKSPACE"
EOF
    
    success "Connection details saved to ${SCRIPT_DIR}/.automation_connection"
}

# Execute main function
main "$@"