#!/usr/bin/env bash
# K6 Content Management Functions
# Replaces inject pattern with clearer add/list/get/remove/execute commands

# Main content command dispatcher
k6::content() {
    local subcommand="$1"
    shift
    
    case "$subcommand" in
        add)
            k6::content::add "$@"
            ;;
        list)
            k6::content::list "$@"
            ;;
        get)
            k6::content::get "$@"
            ;;
        remove)
            k6::content::remove "$@"
            ;;
        execute)
            k6::content::execute "$@"
            ;;
        *)
            echo "Usage: resource-k6 content <add|list|get|remove|execute>"
            echo ""
            echo "Commands:"
            echo "  add     - Add a test script or data file to K6"
            echo "  list    - List all stored content"
            echo "  get     - Retrieve a specific content item"
            echo "  remove  - Remove a content item"
            echo "  execute - Run a stored test script"
            return 1
            ;;
    esac
}

# Add content to K6
k6::content::add() {
    local file=""
    local name=""
    local type="script"  # Default to script
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the file
                if [[ -z "$file" ]]; then
                    file="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$file" ]]; then
        log::error "File required"
        echo "Usage: resource-k6 content add --file <path> [--name <name>] [--type script|data]"
        return 1
    fi
    
    # Initialize if needed
    k6::core::init
    
    # Handle shared resources
    if [[ "$file" =~ ^shared: ]]; then
        local shared_path="${file#shared:}"
        file="${VROOLI_SHARED_DIR:-/opt/vrooli/shared}/${shared_path}"
        
        if [[ ! -f "$file" ]]; then
            log::error "Shared resource not found: $shared_path"
            return 1
        fi
    fi
    
    # Check if file exists
    if [[ ! -f "$file" ]]; then
        log::error "File not found: $file"
        return 1
    fi
    
    # Determine name if not provided
    if [[ -z "$name" ]]; then
        name=$(basename "$file")
    fi
    
    # Determine target directory based on type
    local target_dir=""
    case "$type" in
        script)
            target_dir="$K6_SCRIPTS_DIR"
            # Validate JavaScript file
            if [[ ! "$name" =~ \.js$ ]]; then
                log::warn "Script should have .js extension"
            fi
            ;;
        data)
            target_dir="$K6_DATA_DIR"
            ;;
        *)
            log::error "Invalid type: $type (use 'script' or 'data')"
            return 1
            ;;
    esac
    
    # Copy file to target directory
    local target_path="$target_dir/$name"
    log::info "Adding $type: $name"
    
    cp "$file" "$target_path"
    chmod 644 "$target_path"
    
    log::success "$type added: $name"
    
    # Show info for scripts
    if [[ "$type" == "script" ]]; then
        echo ""
        echo "Script: $name"
        echo "Size: $(stat -c%s "$target_path" 2>/dev/null || stat -f%z "$target_path" 2>/dev/null) bytes"
        
        # Try to extract options from script
        if grep -q "export const options" "$target_path"; then
            echo ""
            echo "Test Options:"
            grep -A 5 "export const options" "$target_path" | head -10
        fi
        
        echo ""
        echo "Execute with: resource-k6 content execute --name $name"
    fi
}

# List all content
k6::content::list() {
    local format="${1:-text}"
    
    if [[ "$format" == "json" ]]; then
        # JSON output
        local scripts=()
        local data_files=()
        local results=()
        
        # List scripts
        if [[ -d "$K6_SCRIPTS_DIR" ]]; then
            while IFS= read -r file; do
                [[ -f "$file" ]] && scripts+=("\"$(basename "$file")\"")
            done < <(find "$K6_SCRIPTS_DIR" -name "*.js" 2>/dev/null)
        fi
        
        # List data files
        if [[ -d "$K6_DATA_DIR" ]]; then
            while IFS= read -r file; do
                [[ -f "$file" ]] && data_files+=("\"$(basename "$file")\"")
            done < <(find "$K6_DATA_DIR" -type f ! -path "*/scripts/*" ! -path "*/results/*" 2>/dev/null)
        fi
        
        # List results
        if [[ -d "$K6_RESULTS_DIR" ]]; then
            while IFS= read -r file; do
                [[ -f "$file" ]] && results+=("\"$(basename "$file")\"")
            done < <(find "$K6_RESULTS_DIR" -name "*.json" 2>/dev/null)
        fi
        
        # Build JSON
        echo "{"
        echo "  \"scripts\": [$(IFS=,; echo "${scripts[*]:-}")],"
        echo "  \"data\": [$(IFS=,; echo "${data_files[*]:-}")],"
        echo "  \"results\": [$(IFS=,; echo "${results[*]:-}")]"
        echo "}"
    else
        # Text output
        echo "K6 Content:"
        echo "==========="
        
        echo ""
        echo "Test Scripts:"
        if [[ -d "$K6_SCRIPTS_DIR" ]]; then
            local script_count=0
            while IFS= read -r file; do
                if [[ -f "$file" ]]; then
                    echo "  - $(basename "$file")"
                    ((script_count++))
                fi
            done < <(find "$K6_SCRIPTS_DIR" -name "*.js" 2>/dev/null | sort)
            [[ $script_count -eq 0 ]] && echo "  No scripts found"
        else
            echo "  Scripts directory not found"
        fi
        
        echo ""
        echo "Data Files:"
        if [[ -d "$K6_DATA_DIR" ]]; then
            local data_count=0
            while IFS= read -r file; do
                if [[ -f "$file" ]]; then
                    echo "  - $(basename "$file")"
                    ((data_count++))
                fi
            done < <(find "$K6_DATA_DIR" -type f ! -path "*/scripts/*" ! -path "*/results/*" 2>/dev/null | sort)
            [[ $data_count -eq 0 ]] && echo "  No data files found"
        else
            echo "  Data directory not found"
        fi
        
        echo ""
        echo "Test Results:"
        if [[ -d "$K6_RESULTS_DIR" ]]; then
            local result_count=$(find "$K6_RESULTS_DIR" -name "*.json" 2>/dev/null | wc -l)
            echo "  $result_count test results stored"
            if [[ $result_count -gt 0 ]]; then
                echo "  Recent results:"
                find "$K6_RESULTS_DIR" -name "*.json" -printf "    - %f (modified: %TY-%Tm-%Td %TH:%TM)\n" 2>/dev/null | sort -r | head -5
            fi
        else
            echo "  Results directory not found"
        fi
    fi
}

# Get specific content
k6::content::get() {
    local name=""
    local type="script"  # Default to script
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the name
                if [[ -z "$name" ]]; then
                    name="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Name required"
        echo "Usage: resource-k6 content get --name <name> [--type script|data|result]"
        return 1
    fi
    
    # Determine source directory based on type
    local source_path=""
    case "$type" in
        script)
            source_path="$K6_SCRIPTS_DIR/$name"
            [[ ! "$name" =~ \.js$ ]] && source_path="${source_path}.js"
            ;;
        data)
            source_path="$K6_DATA_DIR/$name"
            ;;
        result)
            source_path="$K6_RESULTS_DIR/$name"
            [[ ! "$name" =~ \.json$ ]] && source_path="${source_path}.json"
            ;;
        *)
            log::error "Invalid type: $type (use 'script', 'data', or 'result')"
            return 1
            ;;
    esac
    
    # Check if file exists
    if [[ ! -f "$source_path" ]]; then
        log::error "$type not found: $name"
        return 1
    fi
    
    # Output the content
    cat "$source_path"
}

# Remove content
k6::content::remove() {
    local name=""
    local type="script"  # Default to script
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --type)
                type="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the name
                if [[ -z "$name" ]]; then
                    name="$1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Name required"
        echo "Usage: resource-k6 content remove --name <name> [--type script|data|result]"
        return 1
    fi
    
    # Determine target path based on type
    local target_path=""
    case "$type" in
        script)
            target_path="$K6_SCRIPTS_DIR/$name"
            [[ ! "$name" =~ \.js$ ]] && target_path="${target_path}.js"
            ;;
        data)
            target_path="$K6_DATA_DIR/$name"
            ;;
        result)
            target_path="$K6_RESULTS_DIR/$name"
            [[ ! "$name" =~ \.json$ ]] && target_path="${target_path}.json"
            ;;
        *)
            log::error "Invalid type: $type (use 'script', 'data', or 'result')"
            return 1
            ;;
    esac
    
    # Check if file exists
    if [[ ! -f "$target_path" ]]; then
        log::error "$type not found: $name"
        return 1
    fi
    
    # Remove the file
    rm -f "$target_path"
    log::success "$type removed: $name"
}

# Execute a stored test script
k6::content::execute() {
    local name=""
    local options=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                name="$2"
                shift 2
                ;;
            --options)
                options="$2"
                shift 2
                ;;
            *)
                # If no flag, assume it's the name
                if [[ -z "$name" ]]; then
                    name="$1"
                else
                    # Everything else is K6 options
                    options="$options $1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$name" ]]; then
        log::error "Script name required"
        echo "Usage: resource-k6 content execute --name <script> [--options '<k6-options>']"
        return 1
    fi
    
    # Add .js extension if not present
    [[ ! "$name" =~ \.js$ ]] && name="${name}.js"
    
    local script_path="$K6_SCRIPTS_DIR/$name"
    
    # Check if script exists
    if [[ ! -f "$script_path" ]]; then
        log::error "Script not found: $name"
        echo "Available scripts:"
        k6::content::list | grep -A 20 "Test Scripts:" | grep "^  -"
        return 1
    fi
    
    # Run the test using existing K6 test runner
    log::info "Executing K6 test: $name"
    
    # Use the existing run-test function if available
    if declare -f k6::test::run >/dev/null 2>&1; then
        k6::test::run "$name" $options
    else
        # Direct execution via Docker
        local timestamp=$(date +%Y%m%d_%H%M%S)
        local result_file="$K6_RESULTS_DIR/result_${timestamp}_$(basename "$name" .js).json"
        
        docker run --rm \
            -v "$K6_SCRIPTS_DIR:/scripts:ro" \
            -v "$K6_DATA_DIR:/data:ro" \
            -v "$K6_RESULTS_DIR:/results" \
            grafana/k6:latest \
            run $options \
            --out json="/results/$(basename "$result_file")" \
            "/scripts/$name"
        
        if [[ $? -eq 0 ]]; then
            log::success "Test completed. Results saved to: $(basename "$result_file")"
        else
            log::error "Test execution failed"
            return 1
        fi
    fi
}