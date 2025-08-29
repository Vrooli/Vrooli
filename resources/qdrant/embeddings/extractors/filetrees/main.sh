#!/usr/bin/env bash
# File Tree Structure Extractor for Qdrant Embeddings
# 
# SCOPE: Processes DIRECTORY ORGANIZATION and file tree structures outside of:
# - scenarios/ folder (handled by scenarios stream)
# - resources/ folder (handled by resources stream) 
# - initialization/ folder (handled by resources/initialization stream)
#
# PROCESSING: Creates semantic directory summaries for structural understanding
# - Analyzes directory purposes and organization patterns
# - Identifies module boundaries and architectural decisions
# - Extracts file type distributions and semantic groupings
# - Output: Module-level summaries for spatial/organizational intelligence
#
# COVERAGE: Processes source code organization, module structure,
# and architectural patterns implied by file system organization

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

# Temporary directory for extracted content
EXTRACT_TEMP_DIR="/tmp/qdrant-filetrees-extract-$$"
trap "rm -rf $EXTRACT_TEMP_DIR" EXIT

# Semantic directory patterns we recognize (ordered by priority)
SEMANTIC_PATTERNS=(
    "resources|resource"   # System resources (high priority)
    "embeddings|embedding" # AI/ML embeddings (high priority)  
    "qdrant"              # Vector database (high priority)
    "extractors|extractor" # Data extractors (high priority)
    "parsers|parser"       # Data parsers (high priority)
    "indexers|indexer"     # Data indexers (high priority)
    "scripts|script"       # Scripts and automation (high priority)
    "lib|libs|library"     # Core libraries (high priority)
    "components|component"  # UI components
    "services|service"      # Business logic services
    "utils|util"           # Utility functions
    "types|typing"         # Type definitions
    "hooks|hook"           # React/framework hooks
    "api|apis"             # API endpoints/clients
    "models|model"         # Data models
    "views|view"           # UI views/pages
    "controllers|controller" # MVC controllers
    "middleware"           # Middleware functions
    "config|configs|configuration" # Configuration files
    "test|tests|__test__|__tests__|spec|specs" # Test files
    "docs|doc|documentation" # Documentation
    "assets|static"        # Static assets
    "styles|css|scss"      # Styling files
)

#######################################
# Analyze directory purpose based on name and contents
# Arguments:
#   $1 - Directory path
# Returns: Purpose string and category
#######################################
qdrant::extract::analyze_directory_purpose() {
    local dir="$1"
    local dir_name=$(basename "$dir")
    local purpose="general"
    local category="source"
    
    # Check against semantic patterns (ordered by priority)
    for pattern in "${SEMANTIC_PATTERNS[@]}"; do
        if [[ "$dir_name" =~ ^($pattern)s?$ ]]; then
            case "$pattern" in
                "resources|resource") purpose="system resources and infrastructure"; category="system" ;;
                "embeddings|embedding") purpose="AI embeddings and vector processing"; category="system" ;;
                "qdrant") purpose="vector database and search infrastructure"; category="system" ;;
                "extractors|extractor") purpose="data extraction and processing pipelines"; category="system" ;;
                "parsers|parser") purpose="data parsing and transformation logic"; category="system" ;;
                "indexers|indexer") purpose="data indexing and search optimization"; category="system" ;;
                "scripts|script") purpose="automation scripts and system utilities"; category="system" ;;
                "lib|libs|library") purpose="core libraries and foundational code"; category="core" ;;
                "components|component") purpose="UI components and presentation layer"; category="frontend" ;;
                "services|service") purpose="business logic and application services"; category="backend" ;;
                "utils|util") purpose="shared utility functions and helpers"; category="shared" ;;
                "types|typing") purpose="type definitions and interfaces"; category="types" ;;
                "hooks|hook") purpose="custom hooks and reactive logic"; category="frontend" ;;
                "api|apis") purpose="API endpoints, routes, and external interfaces"; category="api" ;;
                "models|model") purpose="data models and entity definitions"; category="data" ;;
                "views|view") purpose="UI views, pages, and screens"; category="frontend" ;;
                "controllers|controller") purpose="request handlers and application controllers"; category="backend" ;;
                "middleware") purpose="middleware functions and request processing"; category="backend" ;;
                "config|configs|configuration") purpose="configuration files and settings"; category="config" ;;
                "test|tests|__test__|__tests__|spec|specs") purpose="test files and testing utilities"; category="test" ;;
                "docs|doc|documentation") purpose="documentation and guides"; category="docs" ;;
                "assets|static") purpose="static assets and resources"; category="assets" ;;
                "styles|css|scss") purpose="styling and visual design files"; category="styles" ;;
            esac
            break
        fi
    done
    
    # Path-aware classification for unmatched directories
    if [[ "$purpose" == "general" ]]; then
        local dir_path="$dir"
        
        # Check if we're within known system paths  
        if [[ "$dir_path" == *"/resources/"* ]] || [[ "$dir_path" == *"/scripts/"* ]]; then
            purpose="system module component"
            category="system"
        elif [[ "$dir_path" == *"/qdrant/"* ]] || [[ "$dir_path" == *"/embeddings/"* ]]; then
            purpose="AI/vector processing component"
            category="system"
        elif [[ "$dir_path" == *"/lib/"* ]] || [[ "$dir_path" == *"/libraries/"* ]]; then
            purpose="library and utility functions"
            category="core"
        elif [[ "$dir_path" == *"/src/"* ]] && [[ "$dir_path" == *"/components/"* ]]; then
            purpose="application components"
            category="frontend"
        elif [[ "$dir_path" == *"/src/"* ]] && [[ "$dir_path" == *"/services/"* ]]; then
            purpose="application business logic"
            category="backend"
        elif [[ "$dir_path" == *"/docs/"* ]] || [[ "$dir_path" == *"/documentation/"* ]]; then
            purpose="documentation and guides"
            category="docs"
        elif [[ "$dir_path" == *"/test/"* ]] || [[ "$dir_path" == *"/tests/"* ]]; then
            purpose="testing infrastructure"
            category="test"
        fi
    fi
    
    # Additional heuristics based on file contents (only if not already classified)
    if [[ "$purpose" == "general" ]]; then
        local has_components=$(find "$dir" -maxdepth 1 -name "*.tsx" -o -name "*.jsx" 2>/dev/null | wc -l)
        local has_tests=$(find "$dir" -maxdepth 1 -name "*test*" -o -name "*spec*" 2>/dev/null | wc -l)
        local has_configs=$(find "$dir" -maxdepth 1 -name "*.json" -o -name "*.yaml" -o -name "*.yml" -o -name "*.toml" 2>/dev/null | wc -l)
        local has_shell_scripts=$(find "$dir" -maxdepth 1 -name "*.sh" 2>/dev/null | wc -l)
        
        # Prioritize by significance and file counts
        if [[ $has_components -gt 0 ]]; then
            purpose="React/UI components"
            category="frontend"
        elif [[ $has_shell_scripts -gt 2 ]]; then
            purpose="automation scripts and utilities"
            category="scripts"
        elif [[ $has_configs -gt 2 ]]; then
            purpose="configuration and settings"
            category="config"
        elif [[ $has_tests -gt 0 ]] && [[ $has_tests -gt $(($(find "$dir" -maxdepth 1 -type f 2>/dev/null | wc -l) / 2)) ]]; then
            # Only classify as test if tests make up more than half the files
            purpose="test files and testing logic"
            category="test"
        else
            purpose="application logic and implementation"
            category="source"
        fi
    fi
    
    echo "$purpose|$category"
}

#######################################
# Get file type distribution for directory
# Arguments:
#   $1 - Directory path
# Returns: JSON object with file type counts
#######################################
qdrant::extract::get_file_distribution() {
    local dir="$1"
    
    # Count files by extension
    local js_count=$(find "$dir" -maxdepth 1 -name "*.js" -o -name "*.jsx" 2>/dev/null | wc -l)
    local ts_count=$(find "$dir" -maxdepth 1 -name "*.ts" -o -name "*.tsx" 2>/dev/null | wc -l)
    local py_count=$(find "$dir" -maxdepth 1 -name "*.py" 2>/dev/null | wc -l)
    local sh_count=$(find "$dir" -maxdepth 1 -name "*.sh" -o -name "*.bash" 2>/dev/null | wc -l)
    local css_count=$(find "$dir" -maxdepth 1 -name "*.css" -o -name "*.scss" -o -name "*.sass" -o -name "*.less" 2>/dev/null | wc -l)
    local config_count=$(find "$dir" -maxdepth 1 -name "*.json" -o -name "*.yaml" -o -name "*.yml" -o -name "*.toml" 2>/dev/null | wc -l)
    local md_count=$(find "$dir" -maxdepth 1 -name "*.md" 2>/dev/null | wc -l)
    
    jq -n \
        --arg js "$js_count" \
        --arg ts "$ts_count" \
        --arg py "$py_count" \
        --arg sh "$sh_count" \
        --arg css "$css_count" \
        --arg config "$config_count" \
        --arg md "$md_count" \
        '{
            javascript: ($js | tonumber),
            typescript: ($ts | tonumber),
            python: ($py | tonumber),
            shell: ($sh | tonumber),
            css: ($css | tonumber),
            config: ($config | tonumber),
            markdown: ($md | tonumber)
        }'
}

#######################################
# Extract key files from directory (entry points, main files)
# Arguments:
#   $1 - Directory path
# Returns: Comma-separated list of key files
#######################################
qdrant::extract::get_key_files() {
    local dir="$1"
    local key_files=()
    
    # Look for common entry point patterns
    local entry_points=("index" "main" "app" "server" "client" "init")
    for pattern in "${entry_points[@]}"; do
        while IFS= read -r file; do
            if [[ -n "$file" ]]; then
                key_files+=("$(basename "$file")")
            fi
        done < <(find "$dir" -maxdepth 1 -name "${pattern}.*" 2>/dev/null)
    done
    
    # If no entry points, get first few files alphabetically
    if [[ ${#key_files[@]} -eq 0 ]]; then
        while IFS= read -r file; do
            if [[ -n "$file" ]]; then
                key_files+=("$(basename "$file")")
            fi
        done < <(find "$dir" -maxdepth 1 -type f ! -name ".*" 2>/dev/null | sort | head -3)
    fi
    
    # Join with commas, limit to 5 files
    printf "%s" "$(printf '%s,' "${key_files[@]:0:5}" | sed 's/,$//')"
}

#######################################
# Extract directory tree summary
# Arguments:
#   $1 - Directory path
#   $2 - Relative path from app root
# Returns: JSON line with directory summary
#######################################
qdrant::extract::directory_summary() {
    local dir="$1"
    local rel_path="${2:-.}"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    local dir_name=$(basename "$dir")
    
    # Get basic directory stats
    local file_count=$(find "$dir" -maxdepth 1 -type f ! -name ".*" 2>/dev/null | wc -l)
    local subdir_count=$(find "$dir" -maxdepth 1 -type d ! -name ".*" ! -path "$dir" 2>/dev/null | wc -l)
    
    # Skip if too few files (noise reduction)
    if [[ $file_count -lt 3 ]]; then
        return 0
    fi
    
    # Get purpose and category
    local purpose_info=$(qdrant::extract::analyze_directory_purpose "$dir")
    local purpose=${purpose_info%|*}
    local category=${purpose_info#*|}
    
    # Get file type distribution
    local file_distribution=$(qdrant::extract::get_file_distribution "$dir")
    
    # Get key files
    local key_files=$(qdrant::extract::get_key_files "$dir")
    
    # Get subdirectories
    local subdirs=""
    if [[ $subdir_count -gt 0 ]]; then
        subdirs=$(find "$dir" -maxdepth 1 -type d ! -name ".*" ! -path "$dir" -exec basename {} \; 2>/dev/null | sort | head -5 | tr '\n' ',' | sed 's/,$//')
    fi
    
    # Determine primary language
    local primary_lang="mixed"
    local js_count=$(echo "$file_distribution" | jq -r '.javascript')
    local ts_count=$(echo "$file_distribution" | jq -r '.typescript')
    local py_count=$(echo "$file_distribution" | jq -r '.python')
    local sh_count=$(echo "$file_distribution" | jq -r '.shell')
    
    if [[ $ts_count -gt $js_count && $ts_count -gt $py_count && $ts_count -gt $sh_count ]]; then
        primary_lang="TypeScript"
    elif [[ $js_count -gt $py_count && $js_count -gt $sh_count ]]; then
        primary_lang="JavaScript"
    elif [[ $py_count -gt $sh_count ]]; then
        primary_lang="Python"
    elif [[ $sh_count -gt 0 ]]; then
        primary_lang="Shell"
    fi
    
    # Build comprehensive content summary
    local content_summary="The $rel_path directory is a $purpose module containing $file_count files"
    
    if [[ "$primary_lang" != "mixed" ]]; then
        content_summary="$content_summary (primarily $primary_lang)"
    fi
    
    if [[ $subdir_count -gt 0 ]]; then
        content_summary="$content_summary organized into $subdir_count subdirectories"
        if [[ -n "$subdirs" ]]; then
            content_summary="$content_summary: $subdirs"
        fi
    fi
    
    if [[ -n "$key_files" ]]; then
        content_summary="$content_summary. Key files include: $key_files"
    fi
    
    content_summary="$content_summary. This $category module handles $purpose."
    
    # Output JSON with comprehensive metadata
    jq -n \
        --arg content "$content_summary" \
        --arg dir_path "$dir" \
        --arg rel_path "$rel_path" \
        --arg dir_name "$dir_name" \
        --arg purpose "$purpose" \
        --arg category "$category" \
        --arg primary_lang "$primary_lang" \
        --arg file_count "$file_count" \
        --arg subdir_count "$subdir_count" \
        --arg key_files "$key_files" \
        --arg subdirs "$subdirs" \
        --argjson file_distribution "$file_distribution" \
        '{
            content: $content,
            metadata: {
                source_path: $dir_path,
                relative_path: $rel_path,
                directory_name: $dir_name,
                purpose: $purpose,
                category: $category,
                primary_language: $primary_lang,
                file_count: ($file_count | tonumber),
                subdirectory_count: ($subdir_count | tonumber),
                key_files: $key_files,
                subdirectories: $subdirs,
                file_distribution: $file_distribution,
                content_type: "file_tree",
                extractor: "file_trees"
            }
        }' | jq -c
}

#######################################
# Find semantic directories to analyze
# Arguments:
#   $1 - Root directory to search
# Returns: List of directories to analyze
#######################################
qdrant::extract::find_semantic_directories() {
    local root_dir="$1"
    
    # Find directories with meaningful content, excluding noise
    find "$root_dir" -type d \
        ! -path "*/.git/*" \
        ! -path "*/node_modules/*" \
        ! -path "*/dist/*" \
        ! -path "*/build/*" \
        ! -path "*/vendor/*" \
        ! -path "*/.venv/*" \
        ! -path "*/venv/*" \
        ! -path "*/target/*" \
        ! -path "*/__pycache__/*" \
        ! -path "*/.next/*" \
        ! -path "*/.cache/*" \
        ! -path "*/tmp/*" \
        ! -path "*/temp/*" \
        ! -name ".*" \
        2>/dev/null | \
    while IFS= read -r dir; do
        # Only include directories with at least 3 files
        local file_count=$(find "$dir" -maxdepth 1 -type f ! -name ".*" 2>/dev/null | wc -l)
        if [[ $file_count -ge 3 ]]; then
            echo "$dir"
        fi
    done
}

#######################################
# Extract file tree summaries in batch
# Arguments:
#   $1 - Directory path (optional, defaults to current)
#   $2 - Output file path
# Returns: 0 on success
#######################################
qdrant::extract::file_trees_batch() {
    local dir="${1:-.}"
    local output_file="${2:-${EXTRACT_TEMP_DIR}/file_trees.jsonl}"
    
    mkdir -p "${output_file%/*}"
    > "$output_file"
    
    # Get absolute path for relative path calculations
    local abs_root=$(cd "$dir" && pwd)
    
    # Find semantic directories to analyze
    local directories
    directories=$(qdrant::extract::find_semantic_directories "$dir")
    
    if [[ -z "$directories" ]]; then
        log::warn "No semantic directories found in $dir"
        return 0
    fi
    
    local dir_count=$(echo "$directories" | wc -l)
    log::info "Extracting file tree summaries from $dir_count semantic directories"
    
    local count=0
    while IFS= read -r directory; do
        if [[ -n "$directory" ]]; then
            # Calculate relative path
            local rel_path=${directory#$abs_root}
            rel_path=${rel_path#/}  # Remove leading slash
            [[ -z "$rel_path" ]] && rel_path="."
            
            # Extract directory summary
            if qdrant::extract::directory_summary "$directory" "$rel_path" >> "$output_file"; then
                ((count++))
            fi
        fi
    done <<< "$directories"
    
    log::success "Extracted file tree summaries from $count directories"
    echo "$output_file"
}

#######################################
# Extract metadata from file tree content text
# Arguments:
#   $1 - File tree content text
# Returns: JSON metadata object
#######################################
qdrant::extract::filetrees_metadata_from_content() {
    local content="$1"
    
    # Extract directory path from content
    local dir_path
    dir_path=$(echo "$content" | sed -n 's/^The \(.*\) directory is a.*/\1/p' | head -1)
    
    local dir_name=""
    if [[ -n "$dir_path" ]]; then
        dir_name=$(basename "$dir_path")
    fi
    
    # Extract file count
    local file_count
    file_count=$(echo "$content" | sed -n 's/.*containing \([0-9]\+\) files.*/\1/p' | head -1)
    
    # Extract primary language
    local primary_lang
    primary_lang=$(echo "$content" | sed -n 's/.*(primarily \([^)]\+\)).*/\1/p' | head -1)
    
    # Extract subdirectory count
    local subdir_count
    subdir_count=$(echo "$content" | sed -n 's/.*into \([0-9]\+\) subdirectories.*/\1/p' | head -1)
    
    # Determine category from content
    local category="source"
    if [[ "$content" == *"test"* ]]; then
        category="test"
    elif [[ "$content" == *"component"* ]] || [[ "$content" == *"UI"* ]]; then
        category="frontend"
    elif [[ "$content" == *"service"* ]] || [[ "$content" == *"API"* ]]; then
        category="backend"
    elif [[ "$content" == *"config"* ]]; then
        category="config"
    elif [[ "$content" == *"documentation"* ]]; then
        category="docs"
    fi
    
    # Estimate content complexity
    local content_length
    content_length=$(echo -n "$content" | wc -c)
    
    # Build metadata JSON - ensure numeric fields have valid defaults
    local safe_file_count="${file_count:-0}"
    local safe_subdir_count="${subdir_count:-0}"
    local safe_content_length="${content_length:-0}"
    
    # Ensure numeric fields are not empty
    [[ -z "$safe_file_count" || "$safe_file_count" == " " ]] && safe_file_count="0"
    [[ -z "$safe_subdir_count" || "$safe_subdir_count" == " " ]] && safe_subdir_count="0"
    [[ -z "$safe_content_length" || "$safe_content_length" == " " ]] && safe_content_length="0"
    
    jq -n \
        --arg dir_name "${dir_name:-Unknown}" \
        --arg dir_path "${dir_path:-}" \
        --arg primary_lang "${primary_lang:-mixed}" \
        --arg category "$category" \
        --arg file_count "$safe_file_count" \
        --arg subdir_count "$safe_subdir_count" \
        --arg content_length "$safe_content_length" \
        '{
            directory_name: $dir_name,
            source_path: $dir_path,
            primary_language: $primary_lang,
            category: $category,
            file_count: ($file_count | tonumber),
            subdirectory_count: ($subdir_count | tonumber),
            content_length: ($content_length | tonumber),
            content_type: "file_tree",
            extractor: "file_trees"
        }'
}

#######################################
# Process file tree embeddings
# Arguments:
#   $1 - App ID
# Returns: Number of file trees processed
#######################################
qdrant::embeddings::process_file_trees() {
    local app_id="$1"
    local collection="${app_id}-filetrees"
    local count=0
    
    # Extract file trees to temp file
    local output_file="$TEMP_DIR/file_trees.jsonl"
    qdrant::extract::file_trees_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No file trees found for processing"
        echo "0"
        return 0
    fi
    
    # Use the new batch processing function for massive speedup!
    count=$(qdrant::embedding::process_jsonl_file "$output_file" "file_tree" "$collection" "$app_id")
    
    log::debug "Created $count file tree embeddings using real batch processing"
    echo "$count"
}

# Export processing function for cli.sh
export -f qdrant::embeddings::process_file_trees

# Export additional functions for testing
export -f qdrant::extract::file_trees_batch
export -f qdrant::extract::directory_summary
export -f qdrant::extract::analyze_directory_purpose
export -f qdrant::extract::find_semantic_directories
export -f qdrant::extract::get_file_distribution
export -f qdrant::extract::get_key_files
export -f qdrant::extract::filetrees_metadata_from_content