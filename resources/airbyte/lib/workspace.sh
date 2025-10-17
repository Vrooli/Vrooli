#!/bin/bash
# Airbyte Multi-Workspace Support

set -euo pipefail

# Resource metadata
RESOURCE_NAME="airbyte"
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${RESOURCE_DIR}/data"
WORKSPACE_DIR="${DATA_DIR}/workspaces"

# Logging functions
log_info() {
    echo "[INFO] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

# Create new workspace
workspace_create() {
    local name="${1:-}"
    local description="${2:-Default workspace}"
    
    if [[ -z "$name" ]]; then
        log_error "Workspace name required"
        return 1
    fi
    
    log_info "Creating workspace: ${name}..."
    
    # Create workspace directory
    mkdir -p "${WORKSPACE_DIR}/${name}"
    
    # Create workspace configuration
    cat > "${WORKSPACE_DIR}/${name}/config.json" <<EOF
{
    "name": "${name}",
    "description": "${description}",
    "created_at": "$(date -Iseconds)",
    "connections": [],
    "sources": [],
    "destinations": []
}
EOF
    
    # Create workspace-specific environment
    cat > "${WORKSPACE_DIR}/${name}/env.sh" <<EOF
#!/bin/bash
# Workspace-specific environment variables
export AIRBYTE_WORKSPACE="${name}"
export AIRBYTE_WORKSPACE_DIR="${WORKSPACE_DIR}/${name}"
export AIRBYTE_CONFIG_ROOT="${WORKSPACE_DIR}/${name}/config"
export AIRBYTE_LOCAL_ROOT="${WORKSPACE_DIR}/${name}/local"
EOF
    
    chmod +x "${WORKSPACE_DIR}/${name}/env.sh"
    
    # Initialize workspace directories
    mkdir -p "${WORKSPACE_DIR}/${name}/config"
    mkdir -p "${WORKSPACE_DIR}/${name}/local"
    mkdir -p "${WORKSPACE_DIR}/${name}/logs"
    
    log_info "✅ Workspace created: ${name}"
    echo "Location: ${WORKSPACE_DIR}/${name}"
    
    return 0
}

# List workspaces
workspace_list() {
    log_info "Available workspaces:"
    
    if [[ ! -d "$WORKSPACE_DIR" ]] || [[ -z "$(ls -A "$WORKSPACE_DIR" 2>/dev/null)" ]]; then
        echo "  (none)"
        echo ""
        echo "Create a workspace with: vrooli resource airbyte workspace create <name>"
        return 0
    fi
    
    local current_workspace="${AIRBYTE_WORKSPACE:-default}"
    
    for dir in "$WORKSPACE_DIR"/*; do
        if [[ -d "$dir" ]]; then
            local name
            name=$(basename "$dir")
            local marker=""
            if [[ "$name" == "$current_workspace" ]]; then
                marker=" (current)"
            fi
            
            # Get workspace info
            if [[ -f "${dir}/config.json" ]]; then
                local description
                description=$(jq -r '.description' "${dir}/config.json" 2>/dev/null || echo "")
                local connections
                connections=$(jq -r '.connections | length' "${dir}/config.json" 2>/dev/null || echo "0")
                echo "  - ${name}${marker}: ${description} (${connections} connections)"
            else
                echo "  - ${name}${marker}"
            fi
        fi
    done
    
    return 0
}

# Switch workspace
workspace_switch() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log_error "Workspace name required"
        return 1
    fi
    
    if [[ ! -d "${WORKSPACE_DIR}/${name}" ]]; then
        log_error "Workspace not found: ${name}"
        echo "Available workspaces:"
        workspace_list
        return 1
    fi
    
    log_info "Switching to workspace: ${name}..."
    
    # Update active workspace
    echo "${name}" > "${DATA_DIR}/active_workspace"
    
    # Source workspace environment
    source "${WORKSPACE_DIR}/${name}/env.sh"
    
    log_info "✅ Switched to workspace: ${name}"
    echo ""
    echo "Workspace environment variables set:"
    echo "  AIRBYTE_WORKSPACE=${AIRBYTE_WORKSPACE}"
    echo "  AIRBYTE_CONFIG_ROOT=${AIRBYTE_CONFIG_ROOT}"
    
    return 0
}

# Delete workspace
workspace_delete() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        log_error "Workspace name required"
        return 1
    fi
    
    if [[ "$name" == "default" ]]; then
        log_error "Cannot delete default workspace"
        return 1
    fi
    
    if [[ ! -d "${WORKSPACE_DIR}/${name}" ]]; then
        log_error "Workspace not found: ${name}"
        return 1
    fi
    
    # Check if workspace is active
    local current_workspace
    current_workspace=$(cat "${DATA_DIR}/active_workspace" 2>/dev/null || echo "default")
    if [[ "$name" == "$current_workspace" ]]; then
        log_error "Cannot delete active workspace. Switch to another workspace first."
        return 1
    fi
    
    log_info "Deleting workspace: ${name}..."
    echo "This will permanently delete all workspace data."
    echo -n "Are you sure? (y/N): "
    read -r confirm
    
    if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
        log_info "Deletion cancelled"
        return 0
    fi
    
    # Delete workspace directory
    rm -rf "${WORKSPACE_DIR}/${name}"
    
    log_info "✅ Workspace deleted: ${name}"
    
    return 0
}

# Export workspace configuration
workspace_export() {
    local name="${1:-}"
    local output="${2:-}"
    
    if [[ -z "$name" ]]; then
        # Use current workspace
        name=$(cat "${DATA_DIR}/active_workspace" 2>/dev/null || echo "default")
    fi
    
    if [[ ! -d "${WORKSPACE_DIR}/${name}" ]]; then
        log_error "Workspace not found: ${name}"
        return 1
    fi
    
    if [[ -z "$output" ]]; then
        output="${name}_export_$(date +%Y%m%d_%H%M%S).tar.gz"
    fi
    
    log_info "Exporting workspace: ${name}..."
    
    # Create export archive
    tar czf "$output" -C "$WORKSPACE_DIR" "$name"
    
    log_info "✅ Workspace exported to: ${output}"
    
    return 0
}

# Import workspace configuration
workspace_import() {
    local archive="${1:-}"
    local name="${2:-}"
    
    if [[ -z "$archive" ]]; then
        log_error "Archive file required"
        return 1
    fi
    
    if [[ ! -f "$archive" ]]; then
        log_error "Archive not found: ${archive}"
        return 1
    fi
    
    # Extract workspace name if not provided
    if [[ -z "$name" ]]; then
        name=$(tar tzf "$archive" | head -1 | cut -d'/' -f1)
    fi
    
    if [[ -d "${WORKSPACE_DIR}/${name}" ]]; then
        log_error "Workspace already exists: ${name}"
        echo "Use a different name or delete existing workspace first"
        return 1
    fi
    
    log_info "Importing workspace: ${name}..."
    
    # Extract archive
    mkdir -p "$WORKSPACE_DIR"
    tar xzf "$archive" -C "$WORKSPACE_DIR"
    
    # Rename if different name specified
    if [[ "$name" != "$(tar tzf "$archive" | head -1 | cut -d'/' -f1)" ]]; then
        mv "${WORKSPACE_DIR}/$(tar tzf "$archive" | head -1 | cut -d'/' -f1)" "${WORKSPACE_DIR}/${name}"
    fi
    
    log_info "✅ Workspace imported: ${name}"
    
    return 0
}

# Clone workspace
workspace_clone() {
    local source="${1:-}"
    local target="${2:-}"
    
    if [[ -z "$source" ]] || [[ -z "$target" ]]; then
        log_error "Usage: workspace clone <source> <target>"
        return 1
    fi
    
    if [[ ! -d "${WORKSPACE_DIR}/${source}" ]]; then
        log_error "Source workspace not found: ${source}"
        return 1
    fi
    
    if [[ -d "${WORKSPACE_DIR}/${target}" ]]; then
        log_error "Target workspace already exists: ${target}"
        return 1
    fi
    
    log_info "Cloning workspace: ${source} -> ${target}..."
    
    # Copy workspace directory
    cp -r "${WORKSPACE_DIR}/${source}" "${WORKSPACE_DIR}/${target}"
    
    # Update workspace name in config
    if [[ -f "${WORKSPACE_DIR}/${target}/config.json" ]]; then
        jq ".name = \"${target}\" | .created_at = \"$(date -Iseconds)\"" \
            "${WORKSPACE_DIR}/${target}/config.json" > "${WORKSPACE_DIR}/${target}/config.json.tmp"
        mv "${WORKSPACE_DIR}/${target}/config.json.tmp" "${WORKSPACE_DIR}/${target}/config.json"
    fi
    
    log_info "✅ Workspace cloned: ${target}"
    
    return 0
}

# Show workspace info
workspace_info() {
    local name="${1:-}"
    
    if [[ -z "$name" ]]; then
        # Use current workspace
        name=$(cat "${DATA_DIR}/active_workspace" 2>/dev/null || echo "default")
    fi
    
    if [[ ! -d "${WORKSPACE_DIR}/${name}" ]]; then
        log_error "Workspace not found: ${name}"
        return 1
    fi
    
    echo "Workspace: ${name}"
    echo "Location: ${WORKSPACE_DIR}/${name}"
    
    if [[ -f "${WORKSPACE_DIR}/${name}/config.json" ]]; then
        echo ""
        jq '.' "${WORKSPACE_DIR}/${name}/config.json"
    fi
    
    # Show disk usage
    echo ""
    echo "Disk usage:"
    du -sh "${WORKSPACE_DIR}/${name}"/* 2>/dev/null | sed 's/^/  /'
    
    return 0
}

# Initialize default workspace
workspace_init() {
    if [[ ! -d "${WORKSPACE_DIR}/default" ]]; then
        log_info "Initializing default workspace..."
        workspace_create "default" "Default Airbyte workspace"
    fi
    
    # Set active workspace if not set
    if [[ ! -f "${DATA_DIR}/active_workspace" ]]; then
        echo "default" > "${DATA_DIR}/active_workspace"
    fi
    
    return 0
}

# Main workspace command handler
cmd_workspace() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        create)
            workspace_create "$@"
            ;;
        list)
            workspace_list "$@"
            ;;
        switch|use)
            workspace_switch "$@"
            ;;
        delete|remove)
            workspace_delete "$@"
            ;;
        export)
            workspace_export "$@"
            ;;
        import)
            workspace_import "$@"
            ;;
        clone)
            workspace_clone "$@"
            ;;
        info)
            workspace_info "$@"
            ;;
        init)
            workspace_init "$@"
            ;;
        *)
            echo "Usage: vrooli resource airbyte workspace <subcommand> [options]"
            echo ""
            echo "Subcommands:"
            echo "  create <name> [description]  Create new workspace"
            echo "  list                        List all workspaces"
            echo "  switch <name>               Switch to workspace"
            echo "  delete <name>               Delete workspace"
            echo "  export [name] [file]        Export workspace configuration"
            echo "  import <file> [name]        Import workspace configuration"
            echo "  clone <source> <target>     Clone workspace"
            echo "  info [name]                 Show workspace details"
            echo "  init                        Initialize default workspace"
            return 1
            ;;
    esac
}