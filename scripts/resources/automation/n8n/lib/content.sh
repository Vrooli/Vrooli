#!/bin/bash
# n8n content management library
# Manages workflows, credentials, and configurations

# Source dependencies
source "${N8N_CLI_DIR}/lib/core.sh"
source "${N8N_CLI_DIR}/lib/api.sh"

# Content directory for storing workflows and configurations
N8N_CONTENT_DIR="${var_ROOT_DIR}/data/resources/n8n/content"

# Initialize content directory
n8n::content::init() {
    mkdir -p "${N8N_CONTENT_DIR}/workflows"
    mkdir -p "${N8N_CONTENT_DIR}/credentials"
    mkdir -p "${N8N_CONTENT_DIR}/configs"
}

# Main content management function
n8n::content() {
    local action="${1:-}"
    shift
    
    case "$action" in
        add)
            n8n::content::add "$@"
            ;;
        list)
            n8n::content::list "$@"
            ;;
        get)
            n8n::content::get "$@"
            ;;
        remove)
            n8n::content::remove "$@"
            ;;
        execute)
            n8n::content::execute "$@"
            ;;
        *)
            echo "Usage: content <add|list|get|remove|execute> [options]"
            echo ""
            echo "Commands:"
            echo "  add --file <file> [--type <type>] [--name <name>]  Add content from file"
            echo "  list [--type <type>]                                List stored content"
            echo "  get --name <name>                                   Get specific content"
            echo "  remove --name <name>                                Remove content"
            echo "  execute --name <name>                               Execute/activate workflow"
            echo ""
            echo "Content types: workflow, credential, config"
            return 1
            ;;
    esac
}

# Add content from file
n8n::content::add() {
    local file=""
    local type="workflow"
    local name=""
    local activate=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --activate)
                activate=true
                shift
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        echo "Error: --file is required"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "Error: File not found: $file"
        return 1
    fi
    
    # Auto-generate name if not provided
    if [[ -z "$name" ]]; then
        name="$(basename "$file" | sed 's/\.[^.]*$//')"
    fi
    
    # Initialize content directory
    n8n::content::init
    
    # Determine target directory based on type
    local target_dir="${N8N_CONTENT_DIR}/${type}s"
    mkdir -p "$target_dir"
    
    # Copy file to content directory
    local target_file="${target_dir}/${name}.json"
    cp "$file" "$target_file"
    
    # For workflows, also import into n8n
    if [[ "$type" == "workflow" ]]; then
        # Import workflow using existing inject functionality
        n8n::inject::workflow "$target_file"
        local result=$?
        
        if [[ $result -eq 0 ]]; then
            echo "[SUCCESS] Added workflow: $name"
            
            # Activate if requested
            if [[ "$activate" == true ]]; then
                n8n::content::execute --name "$name"
            fi
        else
            echo "[ERROR] Failed to import workflow: $name"
            rm -f "$target_file"
            return 1
        fi
    elif [[ "$type" == "credential" ]]; then
        # Import credential using existing inject functionality
        n8n::inject::credential "$target_file"
        local result=$?
        
        if [[ $result -eq 0 ]]; then
            echo "[SUCCESS] Added credential: $name"
        else
            echo "[ERROR] Failed to import credential: $name"
            rm -f "$target_file"
            return 1
        fi
    else
        echo "[SUCCESS] Added $type: $name"
    fi
    
    return 0
}

# List stored content
n8n::content::list() {
    local type=""
    local format="text"
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type)
                type="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    # Initialize content directory
    n8n::content::init
    
    if [[ "$format" == "json" ]]; then
        echo "{"
        echo '  "content": ['
        local first=true
    else
        echo "[HEADER]  n8n Content"
    fi
    
    # List content based on type
    local dirs=()
    if [[ -n "$type" ]]; then
        dirs=("${N8N_CONTENT_DIR}/${type}s")
    else
        dirs=("${N8N_CONTENT_DIR}/workflows" "${N8N_CONTENT_DIR}/credentials" "${N8N_CONTENT_DIR}/configs")
    fi
    
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            local content_type="$(basename "$dir" | sed 's/s$//')"
            
            for file in "$dir"/*.json 2>/dev/null; do
                if [[ -f "$file" ]]; then
                    local content_name="$(basename "$file" .json)"
                    
                    if [[ "$format" == "json" ]]; then
                        if [[ "$first" != true ]]; then
                            echo ","
                        fi
                        echo -n "    {\"type\": \"$content_type\", \"name\": \"$content_name\"}"
                        first=false
                    else
                        echo "[INFO]    $content_type: $content_name"
                    fi
                fi
            done
        fi
    done
    
    if [[ "$format" == "json" ]]; then
        echo ""
        echo "  ]"
        echo "}"
    fi
    
    return 0
}

# Get specific content
n8n::content::get() {
    local name=""
    local output=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --output)
                output="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required"
        return 1
    fi
    
    # Initialize content directory
    n8n::content::init
    
    # Search for content file
    local found_file=""
    for dir in "${N8N_CONTENT_DIR}/workflows" "${N8N_CONTENT_DIR}/credentials" "${N8N_CONTENT_DIR}/configs"; do
        if [[ -f "$dir/${name}.json" ]]; then
            found_file="$dir/${name}.json"
            break
        fi
    done
    
    if [[ -z "$found_file" ]]; then
        echo "Error: Content not found: $name"
        return 1
    fi
    
    # Output content
    if [[ -n "$output" ]]; then
        cp "$found_file" "$output"
        echo "[SUCCESS] Content saved to: $output"
    else
        cat "$found_file"
    fi
    
    return 0
}

# Remove content
n8n::content::remove() {
    local name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required"
        return 1
    fi
    
    # Initialize content directory
    n8n::content::init
    
    # Search for content file
    local found_file=""
    local content_type=""
    for dir in "${N8N_CONTENT_DIR}/workflows" "${N8N_CONTENT_DIR}/credentials" "${N8N_CONTENT_DIR}/configs"; do
        if [[ -f "$dir/${name}.json" ]]; then
            found_file="$dir/${name}.json"
            content_type="$(basename "$dir" | sed 's/s$//')"
            break
        fi
    done
    
    if [[ -z "$found_file" ]]; then
        echo "Error: Content not found: $name"
        return 1
    fi
    
    # For workflows, also delete from n8n
    if [[ "$content_type" == "workflow" ]]; then
        # Get workflow ID from n8n
        local workflow_id=$(n8n::api::get_workflow_id_by_name "$name" 2>/dev/null)
        if [[ -n "$workflow_id" ]]; then
            n8n::api::delete_workflow "$workflow_id"
        fi
    fi
    
    # Remove file
    rm -f "$found_file"
    echo "[SUCCESS] Removed $content_type: $name"
    
    return 0
}

# Execute/activate workflow
n8n::content::execute() {
    local name=""
    local test_mode=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --test)
                test_mode=true
                shift
                ;;
            *)
                echo "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        echo "Error: --name is required"
        return 1
    fi
    
    # Initialize content directory
    n8n::content::init
    
    # Check if workflow exists in content
    if [[ ! -f "${N8N_CONTENT_DIR}/workflows/${name}.json" ]]; then
        echo "Error: Workflow not found: $name"
        return 1
    fi
    
    # Get workflow ID from n8n
    local workflow_id=$(n8n::api::get_workflow_id_by_name "$name" 2>/dev/null)
    
    if [[ -z "$workflow_id" ]]; then
        echo "Error: Workflow not found in n8n: $name"
        echo "Try adding it first with: resource-n8n content add --file ${N8N_CONTENT_DIR}/workflows/${name}.json"
        return 1
    fi
    
    if [[ "$test_mode" == true ]]; then
        # Test execution
        echo "[INFO] Testing workflow: $name"
        n8n::api::test_workflow "$workflow_id"
    else
        # Activate workflow
        echo "[INFO] Activating workflow: $name"
        n8n::api::activate_workflow "$workflow_id"
    fi
    
    return $?
}

# Export the main function
export -f n8n::content
export -f n8n::content::init
export -f n8n::content::add
export -f n8n::content::list
export -f n8n::content::get
export -f n8n::content::remove
export -f n8n::content::execute