#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Personal Digital Twin - Windmill App Deployment
# Deploys Windmill apps for persona management UI
################################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
WINDMILL_HOST="${WINDMILL_HOST:-localhost}"
WINDMILL_PORT="${WINDMILL_PORT:-5681}"
WINDMILL_API_URL="http://${WINDMILL_HOST}:${WINDMILL_PORT}/api"
WORKSPACE="${WINDMILL_WORKSPACE:-demo}"

log::info() { echo "[INFO] $*"; }
log::error() { echo "[ERROR] $*" >&2; }
log::success() { echo "[SUCCESS] $*"; }

# Wait for Windmill to be ready
log::info "Waiting for Windmill to be ready..."
until curl -sf "${WINDMILL_API_URL}/version" >/dev/null 2>&1; do
    echo "Waiting for Windmill..."
    sleep 2
done

log::success "Windmill is ready!"

# Deploy app function
deploy_app() {
    local app_file="$1"
    local app_name="$(basename "$app_file" .json)"
    
    log::info "Deploying Windmill app: $app_name"
    
    if [[ ! -f "$app_file" ]]; then
        log::error "App file not found: $app_file"
        return 1
    fi
    
    # Create or update app
    curl -X POST "${WINDMILL_API_URL}/w/${WORKSPACE}/apps" \
        -H "Content-Type: application/json" \
        -d @"$app_file" >/dev/null 2>&1
        
    if [[ $? -eq 0 ]]; then
        log::success "App $app_name deployed successfully"
    else
        # Try updating if it already exists
        app_id=$(echo "$app_name" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g')
        curl -X PUT "${WINDMILL_API_URL}/w/${WORKSPACE}/apps/u/admin/${app_id}" \
            -H "Content-Type: application/json" \
            -d @"$app_file" >/dev/null 2>&1
            
        if [[ $? -eq 0 ]]; then
            log::success "App $app_name updated successfully"
        else
            log::error "Failed to deploy app $app_name"
            return 1
        fi
    fi
}

# Deploy all apps from the automation/windmill directory
log::info "Deploying Windmill apps..."

APPS_DIR="${SCENARIO_DIR}/initialization/automation/windmill"
if [[ ! -d "$APPS_DIR" ]]; then
    log::error "Apps directory not found: $APPS_DIR"
    exit 1
fi

# Deploy each app file
for app_file in "${APPS_DIR}"/*.json; do
    if [[ -f "$app_file" ]]; then
        deploy_app "$app_file"
    fi
done

log::success "Windmill app deployment complete!"
log::info "Apps accessible at: http://${WINDMILL_HOST}:${WINDMILL_PORT}/apps"