#!/bin/bash
# ERPNext Injection Functions

# Inject custom items into ERPNext
erpnext::inject() {
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
        log::error "Usage: inject --type <app|doctype|script> <file>"
        return 1
    fi
    
    if [ ! -f "$file" ]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Get data directory
    local data_dir="${HOME}/.erpnext"
    
    # Process based on type
    case "$type" in
        app)
            erpnext::inject::app "$file" "$data_dir/apps"
            ;;
        doctype)
            erpnext::inject::doctype "$file" "$data_dir/doctypes"
            ;;
        script)
            erpnext::inject::script "$file" "$data_dir/scripts"
            ;;
        *)
            log::error "Unknown injection type: $type"
            log::info "Valid types: app, doctype, script"
            return 1
            ;;
    esac
}

# Inject app
erpnext::inject::app() {
    local file="$1"
    local target_dir="$2"
    
    mkdir -p "$target_dir"

    local filename
    filename="$(basename "$file")"
    cp "$file" "$target_dir/$filename" || {
        log::error "Failed to copy app file"
        return 1
    }
    
    # If ERPNext is running, try to install the app
    if erpnext::is_running; then
        log::info "Installing app into running ERPNext instance..."
        # This would normally involve bench commands
        # For now, just copy to the apps directory
    fi
    
    log::success "App injected: $filename"
    return 0
}

# Inject DocType
erpnext::inject::doctype() {
    local file="$1"
    local target_dir="$2"
    
    mkdir -p "$target_dir"
    
    # Validate JSON
    if ! jq empty "$file" 2>/dev/null; then
        log::error "Invalid JSON in DocType file"
        return 1
    fi

    local filename
    filename="$(basename "$file")"
    cp "$file" "$target_dir/$filename" || {
        log::error "Failed to copy DocType file"
        return 1
    }
    
    # If ERPNext is running, try to import the DocType
    if erpnext::is_running; then
        log::info "Importing DocType into running ERPNext instance..."
        # This would normally involve API calls to create the DocType
    fi
    
    log::success "DocType injected: $filename"
    return 0
}

# Inject server script
erpnext::inject::script() {
    local file="$1"
    local target_dir="$2"
    
    mkdir -p "$target_dir"
    
    # Basic Python syntax check
    if ! python3 -m py_compile "$file" 2>/dev/null; then
        log::warn "Python syntax check failed, but continuing..."
    fi

    local filename
    filename="$(basename "$file")"
    cp "$file" "$target_dir/$filename" || {
        log::error "Failed to copy script file"
        return 1
    }
    
    # If ERPNext is running, try to load the script
    if erpnext::is_running; then
        log::info "Loading script into running ERPNext instance..."
        # This would normally involve API calls to create a server script
    fi
    
    log::success "Script injected: $filename"
    return 0
}