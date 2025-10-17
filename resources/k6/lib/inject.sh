#!/usr/bin/env bash
# K6 Injection Functions

# Inject test scripts into K6
k6::inject::execute() {
    local source="$1"
    
    if [[ -z "$source" ]]; then
        log::error "Source file or shared reference required"
        echo "Usage: resource-k6 inject <file.js|shared:path/to/script.js>"
        return 1
    fi
    
    # Initialize if needed
    k6::core::init
    
    # Handle shared resources
    if [[ "$source" =~ ^shared: ]]; then
        local shared_path="${source#shared:}"
        source="${VROOLI_SHARED_DIR:-/opt/vrooli/shared}/${shared_path}"
        
        if [[ ! -f "$source" ]]; then
            log::error "Shared resource not found: $shared_path"
            return 1
        fi
    fi
    
    # Check if source is a file or JSON config
    if [[ -f "$source" ]] && [[ "$source" == *.json ]]; then
        k6::inject::from_json "$source"
    elif [[ -f "$source" ]] && [[ "$source" == *.js ]]; then
        k6::inject::script "$source"
    else
        log::error "Invalid source: $source"
        log::info "Source must be a .js test script or .json configuration file"
        return 1
    fi
}

# Inject a single script
k6::inject::script() {
    local script_file="$1"
    local script_name=$(basename "$script_file")
    local target_path="$K6_SCRIPTS_DIR/$script_name"
    
    log::info "Injecting K6 test script: $script_name"
    
    # Validate JavaScript syntax (basic check)
    if ! node -c "$script_file" 2>/dev/null && ! command -v node >/dev/null 2>&1; then
        log::warn "Cannot validate JavaScript syntax (Node.js not installed)"
    fi
    
    # Copy script to K6 directory
    cp "$script_file" "$target_path"
    chmod 644 "$target_path"
    
    log::success "Test script injected: $target_path"
    
    # Show basic info about the script
    echo ""
    echo "Script Information:"
    echo "=================="
    echo "Name: $script_name"
    echo "Size: $(stat -c%s "$target_path" 2>/dev/null || stat -f%z "$target_path" 2>/dev/null) bytes"
    
    # Try to extract options from script
    if grep -q "export const options" "$target_path"; then
        echo ""
        echo "Test Options:"
        grep -A 10 "export const options" "$target_path" | head -15
    fi
    
    echo ""
    echo "Run with: resource-k6 run-test $script_name"
}

# Inject from JSON configuration
k6::inject::from_json() {
    local config_file="$1"
    
    log::info "Processing K6 injection configuration..."
    
    # Parse JSON configuration
    if ! command -v jq >/dev/null 2>&1; then
        log::error "jq is required for JSON configuration"
        return 1
    fi
    
    # Validate JSON
    if ! jq empty "$config_file" 2>/dev/null; then
        log::error "Invalid JSON configuration"
        return 1
    fi
    
    # Process test scripts
    local script_count=$(jq -r '.scripts | length' "$config_file" 2>/dev/null || echo 0)
    
    if [[ $script_count -gt 0 ]]; then
        log::info "Injecting $script_count test scripts..."
        
        for i in $(seq 0 $((script_count - 1))); do
            local script_name=$(jq -r ".scripts[$i].name" "$config_file")
            local script_content=$(jq -r ".scripts[$i].content" "$config_file")
            local script_file="$K6_SCRIPTS_DIR/$script_name"
            
            if [[ -n "$script_name" ]] && [[ -n "$script_content" ]]; then
                echo "$script_content" > "$script_file"
                chmod 644 "$script_file"
                log::success "Injected: $script_name"
            fi
        done
    fi
    
    # Process test data
    local data_count=$(jq -r '.data | length' "$config_file" 2>/dev/null || echo 0)
    
    if [[ $data_count -gt 0 ]]; then
        log::info "Injecting $data_count data files..."
        
        for i in $(seq 0 $((data_count - 1))); do
            local data_name=$(jq -r ".data[$i].name" "$config_file")
            local data_content=$(jq -r ".data[$i].content" "$config_file")
            local data_file="$K6_DATA_DIR/$data_name"
            
            if [[ -n "$data_name" ]] && [[ -n "$data_content" ]]; then
                echo "$data_content" > "$data_file"
                chmod 644 "$data_file"
                log::success "Injected data: $data_name"
            fi
        done
    fi
    
    # Process environment variables
    local env_vars=$(jq -r '.environment | to_entries[] | "\(.key)=\(.value)"' "$config_file" 2>/dev/null || true)
    
    if [[ -n "$env_vars" ]]; then
        log::info "Setting environment variables..."
        echo "$env_vars" > "$K6_CONFIG_DIR/environment"
        log::success "Environment variables configured"
    fi
    
    log::success "K6 injection completed"
}

# List injected content
k6::inject::list() {
    echo "Injected K6 Content:"
    echo "==================="
    
    echo ""
    echo "Test Scripts:"
    if [[ -d "$K6_SCRIPTS_DIR" ]]; then
        ls -la "$K6_SCRIPTS_DIR"/*.js 2>/dev/null || echo "  No scripts found"
    else
        echo "  Scripts directory not found"
    fi
    
    echo ""
    echo "Data Files:"
    if [[ -d "$K6_DATA_DIR" ]]; then
        ls -la "$K6_DATA_DIR"/* 2>/dev/null | grep -v "scripts\|results\|config" || echo "  No data files found"
    else
        echo "  Data directory not found"
    fi
    
    echo ""
    echo "Results:"
    if [[ -d "$K6_RESULTS_DIR" ]]; then
        local result_count=$(ls "$K6_RESULTS_DIR"/*.json 2>/dev/null | wc -l)
        echo "  $result_count test results stored"
    else
        echo "  Results directory not found"
    fi
}