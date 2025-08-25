#!/bin/bash
# LiteLLM content management functionality

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
LITELLM_CONTENT_DIR="${APP_ROOT}/resources/litellm/lib"

# Source dependencies
source "${LITELLM_CONTENT_DIR}/core.sh"
source "${LITELLM_CONTENT_DIR}/docker.sh"

# Content storage directory
readonly LITELLM_CONTENT_STORAGE="${LITELLM_DATA_DIR}/content"

# Initialize content management
litellm::content::init() {
    local verbose="${1:-false}"
    
    mkdir -p "$LITELLM_CONTENT_STORAGE"/{configs,providers,examples}
    
    [[ "$verbose" == "true" ]] && log::info "Content management initialized"
}

# Add content to LiteLLM
litellm::content::add() {
    local content_type=""
    local content_file=""
    local content_name=""
    local content_data=""
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                content_type="$2"
                shift 2
                ;;
            --file)
                content_file="$2"
                shift 2
                ;;
            --name)
                content_name="$2"
                shift 2
                ;;
            --data)
                content_data="$2"
                shift 2
                ;;
            --verbose)
                verbose="true"
                shift
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$content_type" ]]; then
        echo "Error: --type is required"
        echo "Usage: content add --type <config|provider|example> [--file <file>] [--name <name>] [--data <data>]"
        return 1
    fi
    
    # Initialize content management
    litellm::content::init "$verbose"
    
    case "$content_type" in
        "config"|"configuration")
            litellm::content::add_config "$content_file" "$content_name" "$content_data" "$verbose"
            ;;
        "provider")
            litellm::content::add_provider "$content_file" "$content_name" "$content_data" "$verbose"
            ;;
        "example")
            litellm::content::add_example "$content_file" "$content_name" "$content_data" "$verbose"
            ;;
        *)
            echo "Error: Unknown content type '$content_type'"
            echo "Supported types: config, provider, example"
            return 1
            ;;
    esac
}

# Add configuration content
litellm::content::add_config() {
    local file="$1"
    local name="$2" 
    local data="$3"
    local verbose="$4"
    
    if [[ -n "$file" ]]; then
        if [[ ! -f "$file" ]]; then
            echo "Error: Configuration file '$file' not found"
            return 1
        fi
        
        # Determine name from file if not provided
        if [[ -z "$name" ]]; then
            name=$(basename "$file" .yaml)
            name=$(basename "$name" .yml)
        fi
        
        # Copy configuration file
        cp "$file" "${LITELLM_CONTENT_STORAGE}/configs/${name}.yaml"
        [[ "$verbose" == "true" ]] && log::info "Added configuration: $name"
        
    elif [[ -n "$data" ]]; then
        if [[ -z "$name" ]]; then
            echo "Error: --name is required when using --data"
            return 1
        fi
        
        # Write data to file
        echo "$data" > "${LITELLM_CONTENT_STORAGE}/configs/${name}.yaml"
        [[ "$verbose" == "true" ]] && log::info "Added configuration: $name"
        
    else
        echo "Error: Either --file or --data is required"
        return 1
    fi
}

# Add provider configuration
litellm::content::add_provider() {
    local file="$1"
    local name="$2"
    local data="$3"
    local verbose="$4"
    
    if [[ -n "$file" ]]; then
        if [[ ! -f "$file" ]]; then
            echo "Error: Provider file '$file' not found"
            return 1
        fi
        
        # Determine name from file if not provided
        if [[ -z "$name" ]]; then
            name=$(basename "$file" .json)
            name=$(basename "$name" .yaml)
        fi
        
        # Copy provider configuration
        cp "$file" "${LITELLM_CONTENT_STORAGE}/providers/${name}.json"
        [[ "$verbose" == "true" ]] && log::info "Added provider: $name"
        
    elif [[ -n "$data" ]]; then
        if [[ -z "$name" ]]; then
            echo "Error: --name is required when using --data"
            return 1
        fi
        
        # Write provider data to file
        echo "$data" > "${LITELLM_CONTENT_STORAGE}/providers/${name}.json"
        [[ "$verbose" == "true" ]] && log::info "Added provider: $name"
        
    else
        echo "Error: Either --file or --data is required"
        return 1
    fi
}

# Add example content
litellm::content::add_example() {
    local file="$1"
    local name="$2"
    local data="$3"
    local verbose="$4"
    
    if [[ -n "$file" ]]; then
        if [[ ! -f "$file" ]]; then
            echo "Error: Example file '$file' not found"
            return 1
        fi
        
        # Determine name from file if not provided
        if [[ -z "$name" ]]; then
            name=$(basename "$file")
        fi
        
        # Copy example file
        cp "$file" "${LITELLM_CONTENT_STORAGE}/examples/$name"
        [[ "$verbose" == "true" ]] && log::info "Added example: $name"
        
    elif [[ -n "$data" ]]; then
        if [[ -z "$name" ]]; then
            echo "Error: --name is required when using --data"
            return 1
        fi
        
        # Write example data to file
        echo "$data" > "${LITELLM_CONTENT_STORAGE}/examples/$name"
        [[ "$verbose" == "true" ]] && log::info "Added example: $name"
        
    else
        echo "Error: Either --file or --data is required"
        return 1
    fi
}

# List content
litellm::content::list() {
    local content_type=""
    local format="text"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                content_type="$2"
                shift 2
                ;;
            --format)
                format="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Initialize content management
    litellm::content::init false
    
    if [[ -z "$content_type" ]]; then
        # List all content types
        litellm::content::list_all "$format"
    else
        case "$content_type" in
            "config"|"configuration")
                litellm::content::list_configs "$format"
                ;;
            "provider")
                litellm::content::list_providers "$format"
                ;;
            "example")
                litellm::content::list_examples "$format"
                ;;
            *)
                echo "Error: Unknown content type '$content_type'"
                echo "Supported types: config, provider, example"
                return 1
                ;;
        esac
    fi
}

# List all content
litellm::content::list_all() {
    local format="$1"
    
    if [[ "$format" == "json" ]]; then
        echo "{"
        echo "  \"configs\": ["
        litellm::content::list_configs_json "    "
        echo "  ],"
        echo "  \"providers\": ["
        litellm::content::list_providers_json "    "
        echo "  ],"
        echo "  \"examples\": ["
        litellm::content::list_examples_json "    "
        echo "  ]"
        echo "}"
    else
        echo "=== Configurations ==="
        litellm::content::list_configs text
        echo
        echo "=== Providers ==="
        litellm::content::list_providers text
        echo
        echo "=== Examples ==="
        litellm::content::list_examples text
    fi
}

# List configurations
litellm::content::list_configs() {
    local format="$1"
    
    if [[ "$format" == "json" ]]; then
        litellm::content::list_configs_json ""
    else
        local config_dir="${LITELLM_CONTENT_STORAGE}/configs"
        if [[ -d "$config_dir" ]] && ls "$config_dir"/*.yaml >/dev/null 2>&1; then
            for config in "$config_dir"/*.yaml; do
                local name=$(basename "$config" .yaml)
                local size=$(stat -f%z "$config" 2>/dev/null || stat -c%s "$config" 2>/dev/null || echo "unknown")
                local modified=$(stat -f%Sm "$config" 2>/dev/null || stat -c%y "$config" 2>/dev/null || echo "unknown")
                echo "$name ($size bytes, modified: $modified)"
            done
        else
            echo "No configurations found"
        fi
    fi
}

# List configurations in JSON format
litellm::content::list_configs_json() {
    local indent="$1"
    local config_dir="${LITELLM_CONTENT_STORAGE}/configs"
    local first=true
    
    if [[ -d "$config_dir" ]] && ls "$config_dir"/*.yaml >/dev/null 2>&1; then
        for config in "$config_dir"/*.yaml; do
            if [[ "$first" == "true" ]]; then
                first=false
            else
                echo ","
            fi
            
            local name=$(basename "$config" .yaml)
            local size=$(stat -f%z "$config" 2>/dev/null || stat -c%s "$config" 2>/dev/null || echo 0)
            local modified=$(stat -f%Sm "$config" 2>/dev/null || stat -c%y "$config" 2>/dev/null || echo "unknown")
            
            echo -n "${indent}    {\"name\": \"$name\", \"size\": $size, \"modified\": \"$modified\"}"
        done
        echo
    fi
}

# List providers
litellm::content::list_providers() {
    local format="$1"
    
    if [[ "$format" == "json" ]]; then
        litellm::content::list_providers_json ""
    else
        local provider_dir="${LITELLM_CONTENT_STORAGE}/providers"
        if [[ -d "$provider_dir" ]] && ls "$provider_dir"/*.json >/dev/null 2>&1; then
            for provider in "$provider_dir"/*.json; do
                local name=$(basename "$provider" .json)
                local size=$(stat -f%z "$provider" 2>/dev/null || stat -c%s "$provider" 2>/dev/null || echo "unknown")
                local modified=$(stat -f%Sm "$provider" 2>/dev/null || stat -c%y "$provider" 2>/dev/null || echo "unknown")
                echo "$name ($size bytes, modified: $modified)"
            done
        else
            echo "No providers found"
        fi
    fi
}

# List providers in JSON format
litellm::content::list_providers_json() {
    local indent="$1"
    local provider_dir="${LITELLM_CONTENT_STORAGE}/providers"
    local first=true
    
    if [[ -d "$provider_dir" ]] && ls "$provider_dir"/*.json >/dev/null 2>&1; then
        for provider in "$provider_dir"/*.json; do
            if [[ "$first" == "true" ]]; then
                first=false
            else
                echo ","
            fi
            
            local name=$(basename "$provider" .json)
            local size=$(stat -f%z "$provider" 2>/dev/null || stat -c%s "$provider" 2>/dev/null || echo 0)
            local modified=$(stat -f%Sm "$provider" 2>/dev/null || stat -c%y "$provider" 2>/dev/null || echo "unknown")
            
            echo -n "${indent}    {\"name\": \"$name\", \"size\": $size, \"modified\": \"$modified\"}"
        done
        echo
    fi
}

# List examples
litellm::content::list_examples() {
    local format="$1"
    
    if [[ "$format" == "json" ]]; then
        litellm::content::list_examples_json ""
    else
        local example_dir="${LITELLM_CONTENT_STORAGE}/examples"
        if [[ -d "$example_dir" ]] && ls "$example_dir"/* >/dev/null 2>&1; then
            for example in "$example_dir"/*; do
                local name=$(basename "$example")
                local size=$(stat -f%z "$example" 2>/dev/null || stat -c%s "$example" 2>/dev/null || echo "unknown")
                local modified=$(stat -f%Sm "$example" 2>/dev/null || stat -c%y "$example" 2>/dev/null || echo "unknown")
                echo "$name ($size bytes, modified: $modified)"
            done
        else
            echo "No examples found"
        fi
    fi
}

# List examples in JSON format
litellm::content::list_examples_json() {
    local indent="$1"
    local example_dir="${LITELLM_CONTENT_STORAGE}/examples"
    local first=true
    
    if [[ -d "$example_dir" ]] && ls "$example_dir"/* >/dev/null 2>&1; then
        for example in "$example_dir"/*; do
            if [[ "$first" == "true" ]]; then
                first=false
            else
                echo ","
            fi
            
            local name=$(basename "$example")
            local size=$(stat -f%z "$example" 2>/dev/null || stat -c%s "$example" 2>/dev/null || echo 0)
            local modified=$(stat -f%Sm "$example" 2>/dev/null || stat -c%y "$example" 2>/dev/null || echo "unknown")
            
            echo -n "${indent}    {\"name\": \"$name\", \"size\": $size, \"modified\": \"$modified\"}"
        done
        echo
    fi
}

# Get content
litellm::content::get() {
    local content_type=""
    local content_name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                content_type="$2"
                shift 2
                ;;
            --name)
                content_name="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$content_type" || -z "$content_name" ]]; then
        echo "Error: Both --type and --name are required"
        echo "Usage: content get --type <config|provider|example> --name <name>"
        return 1
    fi
    
    case "$content_type" in
        "config"|"configuration")
            local file="${LITELLM_CONTENT_STORAGE}/configs/${content_name}.yaml"
            ;;
        "provider")
            local file="${LITELLM_CONTENT_STORAGE}/providers/${content_name}.json"
            ;;
        "example")
            local file="${LITELLM_CONTENT_STORAGE}/examples/${content_name}"
            ;;
        *)
            echo "Error: Unknown content type '$content_type'"
            return 1
            ;;
    esac
    
    if [[ -f "$file" ]]; then
        cat "$file"
    else
        echo "Error: Content '$content_name' of type '$content_type' not found"
        return 1
    fi
}

# Remove content
litellm::content::remove() {
    local content_type=""
    local content_name=""
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                content_type="$2"
                shift 2
                ;;
            --name)
                content_name="$2"
                shift 2
                ;;
            --verbose)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$content_type" || -z "$content_name" ]]; then
        echo "Error: Both --type and --name are required"
        echo "Usage: content remove --type <config|provider|example> --name <name>"
        return 1
    fi
    
    case "$content_type" in
        "config"|"configuration")
            local file="${LITELLM_CONTENT_STORAGE}/configs/${content_name}.yaml"
            ;;
        "provider")
            local file="${LITELLM_CONTENT_STORAGE}/providers/${content_name}.json"
            ;;
        "example")
            local file="${LITELLM_CONTENT_STORAGE}/examples/${content_name}"
            ;;
        *)
            echo "Error: Unknown content type '$content_type'"
            return 1
            ;;
    esac
    
    if [[ -f "$file" ]]; then
        rm "$file"
        [[ "$verbose" == "true" ]] && log::info "Removed $content_type: $content_name"
        echo "Removed $content_type: $content_name"
    else
        echo "Error: Content '$content_name' of type '$content_type' not found"
        return 1
    fi
}

# Execute/apply content
litellm::content::execute() {
    local content_type=""
    local content_name=""
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                content_type="$2"
                shift 2
                ;;
            --name)
                content_name="$2"
                shift 2
                ;;
            --verbose)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Validate arguments
    if [[ -z "$content_type" || -z "$content_name" ]]; then
        echo "Error: Both --type and --name are required"
        echo "Usage: content execute --type <config|provider> --name <name>"
        return 1
    fi
    
    case "$content_type" in
        "config"|"configuration")
            litellm::content::apply_config "$content_name" "$verbose"
            ;;
        "provider")
            litellm::content::apply_provider "$content_name" "$verbose"
            ;;
        *)
            echo "Error: Cannot execute content type '$content_type'"
            echo "Supported types for execution: config, provider"
            return 1
            ;;
    esac
}

# Apply configuration
litellm::content::apply_config() {
    local name="$1"
    local verbose="$2"
    
    local source_file="${LITELLM_CONTENT_STORAGE}/configs/${name}.yaml"
    
    if [[ ! -f "$source_file" ]]; then
        echo "Error: Configuration '$name' not found"
        return 1
    fi
    
    # Backup current configuration
    if [[ -f "$LITELLM_CONFIG_FILE" ]]; then
        cp "$LITELLM_CONFIG_FILE" "${LITELLM_CONFIG_FILE}.backup.$(date +%Y%m%d-%H%M%S)"
        [[ "$verbose" == "true" ]] && log::info "Backed up current configuration"
    fi
    
    # Apply new configuration
    cp "$source_file" "$LITELLM_CONFIG_FILE"
    [[ "$verbose" == "true" ]] && log::info "Applied configuration: $name"
    
    # Restart service if running
    if litellm::is_running; then
        [[ "$verbose" == "true" ]] && log::info "Restarting LiteLLM to apply new configuration"
        litellm::restart "$verbose"
    fi
    
    echo "Configuration '$name' applied successfully"
}

# Apply provider configuration  
litellm::content::apply_provider() {
    local name="$1"
    local verbose="$2"
    
    local source_file="${LITELLM_CONTENT_STORAGE}/providers/${name}.json"
    
    if [[ ! -f "$source_file" ]]; then
        echo "Error: Provider configuration '$name' not found"
        return 1
    fi
    
    # This would merge the provider configuration into the main config
    # For now, just show what would be applied
    echo "Provider configuration '$name':"
    cat "$source_file"
    
    [[ "$verbose" == "true" ]] && log::warn "Provider merging not yet implemented - showing configuration only"
    
    return 0
}