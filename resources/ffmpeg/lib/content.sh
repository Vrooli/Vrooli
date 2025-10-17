#!/bin/bash
# FFmpeg Content Management Functions - v2.0 Universal Contract Compliant

# Source configuration
ffmpeg::content::init() {
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
    source "${APP_ROOT}/scripts/lib/utils/log.sh"
    source "${APP_ROOT}/resources/ffmpeg/config/defaults.sh"
    ffmpeg::export_config
    
    # Ensure output directories exist
    mkdir -p "${FFMPEG_DATA_DIR}" "${FFMPEG_OUTPUT_DIR}" "${FFMPEG_TEMP_DIR}"
}

# Content add: Add media files to the processing queue or workspace
ffmpeg::content::add() {
    ffmpeg::content::init
    
    local file_path="$1"
    local name="${2:-}"
    local type="${3:-auto}"
    
    if [[ -z "$file_path" ]]; then
        log::error "Usage: content add <file_path> [name] [type]"
        return 1
    fi
    
    if [[ ! -f "$file_path" ]]; then
        log::error "File not found: $file_path"
        return 1
    fi
    
    local basename_file=$(basename "$file_path")
    local dest_name="${name:-$basename_file}"
    local dest_path="${FFMPEG_DATA_DIR}/${dest_name}"
    
    # Copy or symlink file to data directory
    if [[ "$file_path" != "$dest_path" ]]; then
        cp "$file_path" "$dest_path" || {
            log::error "Failed to add file to workspace"
            return 1
        }
    fi
    
    # Create metadata file
    local metadata_file="${FFMPEG_DATA_DIR}/.${dest_name}.meta"
    {
        echo "original_path=$file_path"
        echo "added_date=$(date -Iseconds)"
        echo "type=$type"
        echo "size=$(stat -c%s "$dest_path" 2>/dev/null || stat -f%z "$dest_path" 2>/dev/null)"
    } > "$metadata_file"
    
    log::success "Added media file: $dest_name"
    return 0
}

# Content list: List all media files in workspace
ffmpeg::content::list() {
    ffmpeg::content::init
    
    local format="${1:-text}"
    local filter="${2:-*}"
    
    local files=()
    while IFS= read -r -d '' file; do
        [[ "$(basename "$file")" != .* ]] && files+=("$file")
    done < <(find "${FFMPEG_DATA_DIR}" -maxdepth 1 -name "$filter" -type f -print0 2>/dev/null)
    
    if [[ ${#files[@]} -eq 0 ]]; then
        if [[ "$format" == "json" ]]; then
            echo "[]"
        else
            log::info "No media files found in workspace"
        fi
        return 0
    fi
    
    case "$format" in
        json)
            echo "["
            local first=true
            for file in "${files[@]}"; do
                [[ "$first" != "true" ]] && echo ","
                first=false
                local name=$(basename "$file")
                local size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null)
                local modified=$(stat -c%y "$file" 2>/dev/null | cut -d. -f1 || stat -f%Sm "$file" 2>/dev/null)
                echo "  {"
                echo "    \"name\": \"$name\","
                echo "    \"path\": \"$file\","
                echo "    \"size\": $size,"
                echo "    \"modified\": \"$modified\""
                echo -n "  }"
            done
            echo
            echo "]"
            ;;
        *)
            log::header "📁 Media Files in Workspace"
            printf "%-30s %-12s %-20s\n" "Name" "Size" "Modified"
            printf "%-30s %-12s %-20s\n" "----" "----" "--------"
            for file in "${files[@]}"; do
                local name=$(basename "$file")
                local size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null)
                local size_human=$(numfmt --to=iec "$size" 2>/dev/null || echo "$size")
                local modified=$(stat -c%y "$file" 2>/dev/null | cut -d' ' -f1 || stat -f%Sm "$file" 2>/dev/null | cut -d' ' -f1)
                printf "%-30s %-12s %-20s\n" "$name" "$size_human" "$modified"
            done
            ;;
    esac
    
    return 0
}

# Content get: Retrieve/copy specific media file
ffmpeg::content::get() {
    ffmpeg::content::init
    
    local name="$1"
    local output_path="$2"
    
    if [[ -z "$name" ]]; then
        log::error "Usage: content get <name> [output_path]"
        return 1
    fi
    
    local file_path="${FFMPEG_DATA_DIR}/${name}"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "Media file not found: $name"
        return 1
    fi
    
    if [[ -n "$output_path" ]]; then
        cp "$file_path" "$output_path" || {
            log::error "Failed to copy file to: $output_path"
            return 1
        }
        log::success "Media file copied to: $output_path"
    else
        # Just show the path if no output specified
        echo "$file_path"
    fi
    
    return 0
}

# Content remove: Remove media file from workspace
ffmpeg::content::remove() {
    ffmpeg::content::init
    
    local name="$1"
    local force="${2:-false}"
    
    if [[ -z "$name" ]]; then
        log::error "Usage: content remove <name> [--force]"
        return 1
    fi
    
    local file_path="${FFMPEG_DATA_DIR}/${name}"
    local metadata_file="${FFMPEG_DATA_DIR}/.${name}.meta"
    
    if [[ ! -f "$file_path" ]]; then
        log::error "Media file not found: $name"
        return 1
    fi
    
    if [[ "$force" != "--force" && "$force" != "true" ]]; then
        log::error "Remove requires --force flag to prevent accidental deletion"
        log::info "Usage: content remove $name --force"
        return 1
    fi
    
    rm -f "$file_path" "$metadata_file" || {
        log::error "Failed to remove file: $name"
        return 1
    }
    
    log::success "Removed media file: $name"
    return 0
}