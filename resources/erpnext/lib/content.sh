#!/bin/bash
# ERPNext Content Management Functions

# Source API functions if available
if [[ -f "${APP_ROOT}/resources/erpnext/lib/api.sh" ]]; then
    source "${APP_ROOT}/resources/erpnext/lib/api.sh"
fi

# Add content (inject items)
erpnext::content::add() {
    local type=""
    local name=""
    local file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            *)
                file="$1"
                shift
                ;;
        esac
    done
    
    # If no type specified but file provided, infer type from extension
    if [[ -z "$type" ]] && [[ -n "$file" ]]; then
        case "$file" in
            *.json)
                type="doctype"
                ;;
            *.py)
                type="script"
                ;;
            *)
                type="app"
                ;;
        esac
    fi
    
    # Validate arguments
    if [[ -z "$type" ]]; then
        log::error "Usage: content add --type <app|doctype|script> [--name <name>] [file]"
        return 1
    fi
    
    # Handle different content types
    case "$type" in
        doctype)
            if [[ -n "$file" ]] && [[ -f "$file" ]]; then
                # Read doctype definition from file
                local doctype_json=$(cat "$file")
                name="${name:-$(echo "$doctype_json" | jq -r '.name // .doctype' 2>/dev/null)}"
            else
                # Create simple doctype
                name="${name:-CustomDocType}"
                local doctype_json="{\"doctype\":\"DocType\",\"name\":\"$name\",\"module\":\"Custom\",\"fields\":[{\"fieldname\":\"name1\",\"label\":\"Name\",\"fieldtype\":\"Data\"}]}"
            fi
            
            # Login and create doctype
            local session_id
            session_id=$(erpnext::api::login 2>/dev/null) || {
                log::error "Failed to authenticate"
                return 1
            }
            
            erpnext::api::create_doctype "DocType" "$doctype_json" "$session_id" || {
                log::error "Failed to create DocType"
                erpnext::api::logout "$session_id" 2>/dev/null || true
                return 1
            }
            
            erpnext::api::logout "$session_id" 2>/dev/null || true
            log::success "DocType '$name' created successfully"
            ;;
            
        app)
            name="${name:-$file}"
            if [[ -z "$name" ]]; then
                log::error "App name required"
                return 1
            fi
            
            log::info "Installing app: $name"
            docker exec erpnext-app bench get-app "$name" 2>&1 || {
                log::error "Failed to get app"
                return 1
            }
            docker exec erpnext-app bench --site "${ERPNEXT_SITE_NAME:-vrooli.local}" install-app "$name" 2>&1 || {
                log::error "Failed to install app"
                return 1
            }
            log::success "App '$name' installed successfully"
            ;;
            
        script)
            if [[ -z "$file" ]] || [[ ! -f "$file" ]]; then
                log::error "Script file required"
                return 1
            fi
            
            name="${name:-$(basename "$file" .py)}"
            local data_dir="${HOME}/.erpnext/scripts"
            mkdir -p "$data_dir"
            cp "$file" "$data_dir/${name}.py"
            log::success "Script '$name' added successfully"
            ;;
            
        *)
            log::error "Unknown type: $type"
            log::info "Supported types: doctype, app, script"
            return 1
            ;;
    esac
}

# List content (injected items)
erpnext::content::list() {
    local content_type="${1:-all}"
    
    if ! erpnext::is_running; then
        log::error "ERPNext must be running to list content"
        return 1
    fi
    
    case "$content_type" in
        all)
            log::info "=== ERPNext Content ==="
            echo ""
            
            # List apps
            log::info "Installed Apps:"
            docker exec erpnext-app bench list-apps 2>/dev/null || echo "  Unable to list apps"
            echo ""
            
            # List modules (which contain doctypes)
            log::info "Available Modules:"
            local session_id
            session_id=$(erpnext::api::login 2>/dev/null) || {
                echo "  Unable to authenticate for module listing"
                return 0
            }
            
            local modules_response
            modules_response=$(erpnext::api::get_modules "$session_id" 2>/dev/null)
            if [[ -n "$modules_response" ]]; then
                echo "$modules_response" | jq -r '.message[].module_name' 2>/dev/null | sed 's/^/  - /'
            else
                echo "  Unable to retrieve modules"
            fi
            
            erpnext::api::logout "$session_id" 2>/dev/null || true
            echo ""
            
            # List local scripts
            log::info "Local Scripts:"
            if [[ -d "${HOME}/.erpnext/scripts" ]]; then
                ls -1 "${HOME}/.erpnext/scripts" 2>/dev/null | sed 's/^/  - /' || echo "  None"
            else
                echo "  None"
            fi
            ;;
            
        apps|app)
            log::info "Installed Apps:"
            docker exec erpnext-app bench list-apps 2>/dev/null || {
                log::error "Failed to list apps"
                return 1
            }
            ;;
            
        modules|module|doctypes|doctype)
            log::info "Available Modules/DocTypes:"
            local session_id
            session_id=$(erpnext::api::login 2>/dev/null) || {
                log::error "Failed to authenticate"
                return 1
            }
            
            local modules_response
            modules_response=$(erpnext::api::get_modules "$session_id" 2>/dev/null)
            if [[ -n "$modules_response" ]]; then
                echo "$modules_response" | jq -r '.message[].module_name' 2>/dev/null
            else
                log::warn "No modules found or unable to retrieve"
            fi
            
            erpnext::api::logout "$session_id" 2>/dev/null || true
            ;;
            
        scripts|script)
            log::info "Local Scripts:"
            if [[ -d "${HOME}/.erpnext/scripts" ]]; then
                ls -la "${HOME}/.erpnext/scripts" 2>/dev/null || echo "None"
            else
                echo "None"
            fi
            ;;
            
        *)
            log::error "Unknown content type: $content_type"
            log::info "Valid types: all, apps, modules, doctypes, scripts"
            return 1
            ;;
    esac
}

# Get specific content item
erpnext::content::get() {
    local name="$1"
    local type="${2:-}"
    
    if [ -z "$name" ]; then
        log::error "Usage: content get <name> [type]"
        return 1
    fi
    
    local data_dir="${HOME}/.erpnext"
    local found=false
    
    # Search in different directories
    for search_type in apps doctypes scripts; do
        if [ -n "$type" ] && [ "$type" != "$search_type" ]; then
            continue
        fi
        
        case "$search_type" in
            apps)
                if [ -f "$data_dir/apps/$name" ]; then
                    echo "Found app: $name"
                    ls -la "$data_dir/apps/$name"
                    found=true
                fi
                ;;
            doctypes)
                if [ -f "$data_dir/doctypes/${name}.json" ]; then
                    echo "Found doctype: $name"
                    cat "$data_dir/doctypes/${name}.json"
                    found=true
                fi
                ;;
            scripts)
                if [ -f "$data_dir/scripts/${name}.py" ]; then
                    echo "Found script: $name"
                    cat "$data_dir/scripts/${name}.py"
                    found=true
                fi
                ;;
        esac
    done
    
    if [ "$found" = false ]; then
        log::error "Content not found: $name"
        return 1
    fi
    
    return 0
}

# Remove content item
erpnext::content::remove() {
    local name="$1"
    local type="${2:-}"
    
    if [ -z "$name" ]; then
        log::error "Usage: content remove <name> [type]"
        return 1
    fi
    
    local data_dir="${HOME}/.erpnext"
    local removed=false
    
    # Remove from different directories
    for search_type in apps doctypes scripts; do
        if [ -n "$type" ] && [ "$type" != "$search_type" ]; then
            continue
        fi
        
        case "$search_type" in
            apps)
                if [ -f "$data_dir/apps/$name" ]; then
                    rm "$data_dir/apps/$name" && log::success "Removed app: $name"
                    removed=true
                fi
                ;;
            doctypes)
                if [ -f "$data_dir/doctypes/${name}.json" ]; then
                    rm "$data_dir/doctypes/${name}.json" && log::success "Removed doctype: $name"
                    removed=true
                fi
                ;;
            scripts)
                if [ -f "$data_dir/scripts/${name}.py" ]; then
                    rm "$data_dir/scripts/${name}.py" && log::success "Removed script: $name"
                    removed=true
                fi
                ;;
        esac
    done
    
    if [ "$removed" = false ]; then
        log::error "Content not found: $name"
        return 1
    fi
    
    return 0
}

# Execute ERPNext business operations (optional)
erpnext::content::execute() {
    local operation="$1"
    shift
    
    case "$operation" in
        bench)
            if ! erpnext::is_running; then
                log::error "ERPNext is not running"
                return 1
            fi
            
            log::info "Executing bench command: $*"
            docker exec erpnext-app bench "$@"
            ;;
        sql)
            if ! erpnext::is_running; then
                log::error "ERPNext is not running"
                return 1
            fi
            
            local query="$1"
            log::info "Executing SQL query"
            docker exec erpnext-app bench --site "${ERPNEXT_SITE_NAME}" mariadb --execute "$query"
            ;;
        *)
            log::error "Unknown operation: $operation"
            log::info "Available operations: bench, sql"
            return 1
            ;;
    esac
}