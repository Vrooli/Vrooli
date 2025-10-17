#!/bin/bash

# Injection functions for SageMath

sagemath_inject() {
    local file="${1:-}"
    
    if [ -z "$file" ]; then
        echo "Error: No file specified for injection" >&2
        echo "Usage: sagemath inject <file.sage|file.ipynb>" >&2
        return 1
    fi
    
    if [ ! -f "$file" ]; then
        echo "Error: File not found: $file" >&2
        return 1
    fi
    
    local filename=$(basename "$file")
    local extension="${filename##*.}"
    
    case "$extension" in
        sage|py)
            # Copy to scripts directory
            echo "Injecting SageMath script: $filename"
            cp "$file" "$SAGEMATH_SCRIPTS_DIR/"
            echo "Script injected to: $SAGEMATH_SCRIPTS_DIR/$filename"
            
            # If container is running, execute it
            if sagemath_container_running; then
                echo "Executing script..."
                sagemath_run_script "$SAGEMATH_SCRIPTS_DIR/$filename"
            fi
            ;;
            
        ipynb)
            # Copy to notebooks directory
            echo "Injecting Jupyter notebook: $filename"
            cp "$file" "$SAGEMATH_NOTEBOOKS_DIR/"
            echo "Notebook injected to: $SAGEMATH_NOTEBOOKS_DIR/$filename"
            ;;
            
        *)
            echo "Error: Unsupported file type: $extension" >&2
            echo "Supported types: .sage, .py, .ipynb" >&2
            return 1
            ;;
    esac
    
    return 0
}

# Run a SageMath script
sagemath_run_script() {
    local script="${1:-}"
    
    if [ -z "$script" ]; then
        echo "Error: No script specified" >&2
        return 1
    fi
    
    if ! sagemath_container_running; then
        echo "Error: SageMath is not running" >&2
        return 1
    fi
    
    local script_name=$(basename "$script")
    local internal_path="/home/sage/scripts/$script_name"
    
    # Check if script exists in container
    if ! docker exec "$SAGEMATH_CONTAINER_NAME" test -f "$internal_path"; then
        # Copy script to container if it's not there
        if [ -f "$script" ]; then
            cp "$script" "$SAGEMATH_SCRIPTS_DIR/"
        else
            echo "Error: Script not found: $script" >&2
            return 1
        fi
    fi
    
    # Execute script
    echo "Executing SageMath script: $script_name"
    local output_file="$SAGEMATH_OUTPUTS_DIR/${script_name%.*}_$(date +%Y%m%d_%H%M%S).out"
    
    if docker exec "$SAGEMATH_CONTAINER_NAME" sage "$internal_path" > "$output_file" 2>&1; then
        echo "Script executed successfully"
        echo "Output saved to: $output_file"
        cat "$output_file"
        return 0
    else
        echo "Error: Script execution failed" >&2
        cat "$output_file" >&2
        return 1
    fi
}