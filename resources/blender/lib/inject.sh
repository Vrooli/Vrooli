#!/bin/bash
# Blender injection functionality

# Get script directory
BLENDER_INJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Only source core.sh if functions aren't already defined
if ! type blender::init &>/dev/null; then
    source "${BLENDER_INJECT_DIR}/core.sh"
fi

# Inject a Python script for Blender
blender::inject() {
    local file="$1"
    
    if [[ -z "$file" ]]; then
        echo "[ERROR] No file specified for injection"
        return 1
    fi
    
    if [[ ! -f "$file" ]]; then
        echo "[ERROR] File not found: $file"
        return 1
    fi
    
    # Check if it's a Python file
    if [[ ! "$file" =~ \.py$ ]]; then
        echo "[WARNING] File does not have .py extension: $file"
        echo "[INFO] Blender expects Python scripts"
    fi
    
    # Initialize if needed
    blender::init >/dev/null 2>&1
    
    # Create injected directory if not exists
    local injected_dir="${BLENDER_SCRIPTS_DIR}/injected"
    mkdir -p "$injected_dir"
    
    # Copy file to scripts directory
    local filename=$(basename "$file")
    local target="${injected_dir}/${filename}"
    
    echo "[INFO] Injecting script: $filename"
    cp "$file" "$target" || {
        echo "[ERROR] Failed to inject script"
        return 1
    }
    
    # Make it readable
    chmod 644 "$target"
    
    # Validate Python syntax
    if python3 -m py_compile "$target" 2>/dev/null; then
        echo "[SUCCESS] Script injected successfully: $filename"
        
        # Store metadata
        local metadata_file="${injected_dir}/.${filename}.meta"
        cat > "$metadata_file" << EOF
{
    "filename": "$filename",
    "injected_at": "$(date -Iseconds)",
    "source": "$file",
    "size": $(stat -c%s "$target"),
    "hash": "$(sha256sum "$target" | cut -d' ' -f1)"
}
EOF
    else
        echo "[WARNING] Script has Python syntax errors but was injected anyway"
    fi
    
    # If Blender is running, validate it can be loaded
    if blender::is_running; then
        echo "[INFO] Validating script in Blender..."
        
        # Create a validation wrapper script
        local validator="${BLENDER_DATA_DIR}/temp/validate_${filename}"
        cat > "$validator" << EOF
import sys
import traceback

try:
    # Try to compile the script
    with open('/scripts/injected/${filename}', 'r') as f:
        code = f.read()
    compile(code, '${filename}', 'exec')
    print("[SUCCESS] Script is valid Blender Python")
    sys.exit(0)
except Exception as e:
    print(f"[ERROR] Script validation failed: {e}")
    traceback.print_exc()
    sys.exit(1)
EOF
        
        if docker exec "$BLENDER_CONTAINER_NAME" blender --background --python "/config/temp/validate_$(basename "$validator")" &>/dev/null; then
            echo "[SUCCESS] Script validated in Blender environment"
        else
            echo "[WARNING] Script may have Blender-specific issues"
        fi
        
        rm -f "$validator"
    fi
    
    return 0
}

# List injected scripts with metadata
blender::list_injected() {
    local format="${1:-text}"
    
    # Initialize if needed
    blender::init >/dev/null 2>&1
    
    local injected_dir="${BLENDER_SCRIPTS_DIR}/injected"
    
    if [[ ! -d "$injected_dir" ]]; then
        echo "[INFO] No injected scripts found"
        return 0
    fi
    
    local count=$(find "$injected_dir" -name "*.py" -type f | wc -l)
    
    if [[ "$format" == "json" ]]; then
        echo "{"
        echo "  \"count\": $count,"
        echo "  \"scripts\": ["
        
        local first=true
        for script in "$injected_dir"/*.py; do
            [[ -f "$script" ]] || continue
            
            if [[ "$first" != "true" ]]; then
                echo ","
            fi
            first=false
            
            local name=$(basename "$script")
            local size=$(stat -c%s "$script" 2>/dev/null || echo 0)
            local modified=$(stat -c%Y "$script" 2>/dev/null || echo 0)
            
            echo -n "    {\"name\": \"$name\", \"size\": $size, \"modified\": $modified}"
        done
        
        echo ""
        echo "  ]"
        echo "}"
    else
        echo "[INFO] Injected scripts: $count"
        
        if [[ $count -gt 0 ]]; then
            for script in "$injected_dir"/*.py; do
                [[ -f "$script" ]] || continue
                
                local name=$(basename "$script")
                local size=$(stat -c%s "$script" 2>/dev/null || echo 0)
                local modified=$(stat -c%y "$script" 2>/dev/null | cut -d. -f1)
                
                echo "  - $name (${size} bytes, modified: ${modified})"
                
                # Show metadata if available
                local meta="${injected_dir}/.${name}.meta"
                if [[ -f "$meta" ]]; then
                    local injected_at=$(jq -r '.injected_at' "$meta" 2>/dev/null)
                    if [[ -n "$injected_at" && "$injected_at" != "null" ]]; then
                        echo "    Injected: $injected_at"
                    fi
                fi
            done
        fi
    fi
    
    return 0
}

# Remove an injected script
blender::remove_injected() {
    local script="$1"
    
    if [[ -z "$script" ]]; then
        echo "[ERROR] No script specified"
        return 1
    fi
    
    # Initialize if needed
    blender::init >/dev/null 2>&1
    
    local injected_dir="${BLENDER_SCRIPTS_DIR}/injected"
    local target="${injected_dir}/${script}"
    
    if [[ ! -f "$target" ]]; then
        echo "[ERROR] Script not found: $script"
        return 1
    fi
    
    echo "[INFO] Removing script: $script"
    rm -f "$target" "${injected_dir}/.${script}.meta"
    
    echo "[SUCCESS] Script removed"
    return 0
}

# Export output files from Blender container
blender::export() {
    local source_file="$1"
    local dest_path="$2"
    
    if [[ -z "$source_file" || -z "$dest_path" ]]; then
        echo "[ERROR] Usage: export <source_file> <destination_path>"
        return 1
    fi
    
    # Check if container is running
    if ! blender::is_running; then
        echo "[ERROR] Blender container is not running"
        return 1
    fi
    
    # Check if source exists in container
    if ! docker exec "$BLENDER_CONTAINER_NAME" test -f "/output/$source_file"; then
        echo "[ERROR] File not found in container: $source_file"
        echo "[INFO] Available files:"
        docker exec "$BLENDER_CONTAINER_NAME" ls -la /output/ 2>/dev/null || true
        return 1
    fi
    
    # Create destination directory if needed
    local dest_dir
    dest_dir=$(dirname "$dest_path")
    if [[ ! -d "$dest_dir" ]]; then
        mkdir -p "$dest_dir"
    fi
    
    # Copy file from container
    echo "[INFO] Exporting: $source_file -> $dest_path"
    if docker cp "${BLENDER_CONTAINER_NAME}:/output/$source_file" "$dest_path"; then
        echo "[SUCCESS] File exported successfully"
        
        # Show file info
        if [[ -f "$dest_path" ]]; then
            local size
            size=$(du -h "$dest_path" | cut -f1)
            echo "[INFO] File size: $size"
        fi
        return 0
    else
        echo "[ERROR] Failed to export file"
        return 1
    fi
}

# Export all output files from Blender container
blender::export_all() {
    local dest_dir="$1"
    
    if [[ -z "$dest_dir" ]]; then
        echo "[ERROR] Usage: export-all <destination_directory>"
        return 1
    fi
    
    # Check if container is running
    if ! blender::is_running; then
        echo "[ERROR] Blender container is not running"
        return 1
    fi
    
    # Create destination directory
    mkdir -p "$dest_dir"
    
    # Get list of files in output directory
    local files
    files=$(docker exec "$BLENDER_CONTAINER_NAME" ls -1 /output/ 2>/dev/null)
    
    if [[ -z "$files" ]]; then
        echo "[INFO] No files to export"
        return 0
    fi
    
    echo "[INFO] Exporting all output files to: $dest_dir"
    local count=0
    
    while IFS= read -r file; do
        if [[ -n "$file" ]]; then
            if docker cp "${BLENDER_CONTAINER_NAME}:/output/$file" "$dest_dir/$file" 2>/dev/null; then
                echo "  ✓ Exported: $file"
                ((count++))
            else
                echo "  ✗ Failed to export: $file"
            fi
        fi
    done <<< "$files"
    
    echo "[SUCCESS] Exported $count files"
    return 0
}

# Export functions
export -f blender::inject
export -f blender::list_injected
export -f blender::remove_injected
export -f blender::export
export -f blender::export_all