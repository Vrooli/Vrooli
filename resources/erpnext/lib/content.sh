#!/bin/bash
# ERPNext Content Management Functions

# Add content (inject items)
erpnext::content::add() {
    local type=""
    local file=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            *)
                file="$1"
                shift
                ;;
        esac
    done
    
    # Validate arguments
    if [ -z "$type" ] || [ -z "$file" ]; then
        log::error "Usage: content add --type <app|doctype|script> <file>"
        return 1
    fi
    
    # Delegate to injection function
    erpnext::inject --type "$type" "$file"
}

# List content (injected items)
erpnext::content::list() {
    local format="${1:-text}"
    erpnext::list "$format"
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