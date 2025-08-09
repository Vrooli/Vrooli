#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Personal Digital Twin - n8n Workflow Import
# Imports n8n workflows for data ingestion and processing
################################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

SCENARIO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
N8N_HOST="${N8N_HOST:-localhost}"
N8N_PORT="${N8N_PORT:-5678}"
N8N_API_URL="http://${N8N_HOST}:${N8N_PORT}/api/v1"

# Wait for n8n to be ready
log::info "Waiting for n8n to be ready..."
until curl -sf "${N8N_API_URL}/workflows" >/dev/null 2>&1; do
    echo "Waiting for n8n..."
    sleep 2
done

log::success "n8n is ready!"

# Import workflow function
import_workflow() {
    local workflow_file="$1"
    local workflow_name="$(basename "$workflow_file" .json)"
    
    log::info "Importing workflow: $workflow_name"
    
    if [[ ! -f "$workflow_file" ]]; then
        log::error "Workflow file not found: $workflow_file"
        return 1
    fi
    
    # Check if workflow already exists
    existing_workflow=$(curl -s "${N8N_API_URL}/workflows" | jq -r ".data[] | select(.name==\"$workflow_name\") | .id" 2>/dev/null || echo "")
    
    if [[ -n "$existing_workflow" ]]; then
        log::info "Workflow $workflow_name already exists (ID: $existing_workflow), updating..."
        
        # Update existing workflow
        curl -X PATCH "${N8N_API_URL}/workflows/${existing_workflow}" \
            -H "Content-Type: application/json" \
            -d @"$workflow_file" >/dev/null 2>&1
            
        if [[ $? -eq 0 ]]; then
            log::success "Workflow $workflow_name updated successfully"
        else
            log::error "Failed to update workflow $workflow_name"
            return 1
        fi
    else
        # Create new workflow
        curl -X POST "${N8N_API_URL}/workflows" \
            -H "Content-Type: application/json" \
            -d @"$workflow_file" >/dev/null 2>&1
            
        if [[ $? -eq 0 ]]; then
            log::success "Workflow $workflow_name imported successfully"
        else
            log::error "Failed to import workflow $workflow_name"
            return 1
        fi
    fi
}

# Import all workflows from the automation/n8n directory
log::info "Importing n8n workflows..."

WORKFLOWS_DIR="${SCENARIO_DIR}/initialization/automation/n8n"
if [[ ! -d "$WORKFLOWS_DIR" ]]; then
    log::error "Workflows directory not found: $WORKFLOWS_DIR"
    exit 1
fi

# Import each workflow file
for workflow_file in "${WORKFLOWS_DIR}"/*.json; do
    if [[ -f "$workflow_file" ]]; then
        import_workflow "$workflow_file"
    fi
done

# Activate workflows after import
log::info "Activating imported workflows..."
workflow_ids=$(curl -s "${N8N_API_URL}/workflows" | jq -r '.data[].id' 2>/dev/null || echo "")

if [[ -n "$workflow_ids" ]]; then
    for workflow_id in $workflow_ids; do
        curl -X PATCH "${N8N_API_URL}/workflows/${workflow_id}/activate" >/dev/null 2>&1
        if [[ $? -eq 0 ]]; then
            log::success "Activated workflow ID: $workflow_id"
        else
            log::error "Failed to activate workflow ID: $workflow_id"
        fi
    done
fi

log::success "n8n workflow import complete!"