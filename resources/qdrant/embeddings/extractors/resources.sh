#!/usr/bin/env bash
# Resource Content Extractor for Qdrant Embeddings
# Extracts embeddable content from resource definitions and configurations

set -euo pipefail

# Get directory of this script
EXTRACTOR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EMBEDDINGS_DIR="$(dirname "$EXTRACTOR_DIR")"

# Source required utilities
source "${EMBEDDINGS_DIR}/../../../scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Temporary directory for extracted content
EXTRACT_TEMP_DIR="/tmp/qdrant-resource-extract-$$"
trap "rm -rf $EXTRACT_TEMP_DIR" EXIT

#######################################
# Extract resource information from CLI files
# Arguments:
#   $1 - Path to resource CLI file
# Returns: Extracted resource capabilities
#######################################
qdrant::extract::resource_cli() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    local resource_dir=$(dirname "$file")
    local resource_name=$(basename "$resource_dir")
    
    # Extract resource name and description from CLI header
    local description=""
    if grep -q "^# Description:" "$file"; then
        description=$(grep "^# Description:" "$file" | cut -d: -f2- | sed 's/^ *//')
    elif grep -q "^# $resource_name" "$file"; then
        description=$(grep -A 1 "^# $resource_name" "$file" | tail -1 | sed 's/^# *//')
    fi
    
    # Extract available commands
    local commands=$(grep -E "^[[:space:]]*(\"[^\"]+\"|'[^']+'|[a-z_-]+)\)" "$file" | \
        sed -E "s/^[[:space:]]*(\"([^\"]+)\"|'([^']+)'|([a-z_-]+))\).*/\2\3\4/" | \
        grep -v -E "^(true|false|yes|no|[0-9]+|[\$*])$" | \
        tr '\n' ', ' | sed 's/,$//')
    
    # Build content
    local content="Resource: $resource_name"
    [[ -n "$description" ]] && content="$content
Description: $description"
    
    content="$content
File: $file
Type: CLI Resource
Commands: $commands"
    
    # Check for capabilities in lib directory
    if [[ -d "$resource_dir/lib" ]]; then
        local lib_files=$(find "$resource_dir/lib" -type f -name "*.sh" 2>/dev/null | wc -l)
        content="$content
Library Files: $lib_files"
        
        # Extract key capabilities from lib files
        local capabilities=""
        
        # Look for specific patterns
        [[ -f "$resource_dir/lib/api.sh" ]] && capabilities="${capabilities}API "
        [[ -f "$resource_dir/lib/auth.sh" ]] && capabilities="${capabilities}Authentication "
        [[ -f "$resource_dir/lib/collections.sh" ]] && capabilities="${capabilities}Collections "
        [[ -f "$resource_dir/lib/models.sh" ]] && capabilities="${capabilities}Models "
        [[ -f "$resource_dir/lib/search.sh" ]] && capabilities="${capabilities}Search "
        [[ -f "$resource_dir/lib/backup.sh" ]] && capabilities="${capabilities}Backup "
        [[ -f "$resource_dir/lib/workflows.sh" ]] && capabilities="${capabilities}Workflows "
        [[ -f "$resource_dir/lib/database.sh" ]] && capabilities="${capabilities}Database "
        
        [[ -n "$capabilities" ]] && content="$content
Capabilities: $capabilities"
    fi
    
    echo "$content"
}

#######################################
# Extract resource configuration
# Arguments:
#   $1 - Path to resource directory
# Returns: Configuration details
#######################################
qdrant::extract::resource_config() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    local resource_name=$(basename "$dir")
    local content=""
    
    # Check for common config files
    if [[ -f "$dir/config.yaml" ]] || [[ -f "$dir/config.yml" ]]; then
        local config_file=$(ls "$dir"/config.y*ml 2>/dev/null | head -1)
        content="Resource Config: $resource_name
File: $config_file
Type: YAML Configuration"
        
        # Extract key settings
        if grep -q "^port:" "$config_file" 2>/dev/null; then
            local port=$(grep "^port:" "$config_file" | cut -d: -f2 | tr -d ' ')
            content="$content
Port: $port"
        fi
        
        if grep -q "^host:" "$config_file" 2>/dev/null; then
            local host=$(grep "^host:" "$config_file" | cut -d: -f2 | tr -d ' ')
            content="$content
Host: $host"
        fi
    fi
    
    # Check for .env or environment files
    if [[ -f "$dir/.env" ]] || [[ -f "$dir/.env.example" ]]; then
        local env_file=$(ls "$dir"/.env* 2>/dev/null | head -1)
        [[ -n "$content" ]] && content="$content
---SEPARATOR---
"
        content="${content}Resource Environment: $resource_name
File: $env_file
Type: Environment Configuration"
        
        # Extract environment variables (without values for security)
        local env_vars=$(grep -E "^[A-Z_]+=.*$" "$env_file" 2>/dev/null | cut -d= -f1 | tr '\n' ', ' | sed 's/,$//')
        [[ -n "$env_vars" ]] && content="$content
Environment Variables: $env_vars"
    fi
    
    # Check for Docker configuration
    if [[ -f "$dir/Dockerfile" ]]; then
        [[ -n "$content" ]] && content="$content
---SEPARATOR---
"
        content="${content}Resource Container: $resource_name
File: $dir/Dockerfile
Type: Docker Container"
        
        # Extract base image
        local base_image=$(grep "^FROM " "$dir/Dockerfile" | head -1 | cut -d' ' -f2)
        [[ -n "$base_image" ]] && content="$content
Base Image: $base_image"
        
        # Extract exposed ports
        local ports=$(grep "^EXPOSE " "$dir/Dockerfile" | cut -d' ' -f2- | tr '\n' ', ' | sed 's/,$//')
        [[ -n "$ports" ]] && content="$content
Exposed Ports: $ports"
    fi
    
    echo "$content"
}

#######################################
# Extract resource documentation
# Arguments:
#   $1 - Path to resource directory
# Returns: Documentation content
#######################################
qdrant::extract::resource_docs() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    local resource_name=$(basename "$dir")
    local content=""
    
    # Check for README
    if [[ -f "$dir/README.md" ]]; then
        content="Resource Documentation: $resource_name
File: $dir/README.md"
        
        # Extract key sections
        local overview=$(awk '/^## Overview/,/^##[^#]/' "$dir/README.md" 2>/dev/null | grep -v "^##" | head -5 | tr '\n' ' ')
        [[ -n "$overview" ]] && content="$content
Overview: $overview"
        
        local features=$(awk '/^## Features/,/^##[^#]/' "$dir/README.md" 2>/dev/null | grep "^- " | head -5 | sed 's/^- //' | tr '\n' ', ' | sed 's/,$//')
        [[ -n "$features" ]] && content="$content
Features: $features"
        
        local usage=$(awk '/^## Usage/,/^##[^#]/' "$dir/README.md" 2>/dev/null | grep -v "^##" | head -3 | tr '\n' ' ')
        [[ -n "$usage" ]] && content="$content
Usage: $usage"
    fi
    
    # Check for API documentation
    if [[ -f "$dir/API.md" ]] || [[ -f "$dir/docs/API.md" ]]; then
        local api_file=$(find "$dir" -name "API.md" -type f 2>/dev/null | head -1)
        [[ -n "$content" ]] && content="$content
---SEPARATOR---
"
        content="${content}Resource API: $resource_name
File: $api_file"
        
        # Count endpoints
        local endpoint_count=$(grep -cE "^###.*\`(GET|POST|PUT|DELETE|PATCH)" "$api_file" 2>/dev/null || echo "0")
        content="$content
API Endpoints: $endpoint_count"
    fi
    
    echo "$content"
}

#######################################
# Detect resource category from resource name
# Arguments:
#   $1 - Resource name
# Returns: Category string
#######################################
qdrant::extract::detect_resource_category() {
    local resource_name="$1"
    
    # Map resources to categories based on known patterns
    case "$resource_name" in
        ollama|litellm|gemini|openai|claude-code|openrouter|llamaindex|langchain)
            echo "ai"
            ;;
        postgres|redis|qdrant|minio|neo4j|questdb|vault|postgis)
            echo "storage"
            ;;
        n8n|windmill|node-red|huginn)
            echo "automation"
            ;;
        browserless|agent-s2|autogpt|crewai)
            echo "agents"
            ;;
        judge0|ffmpeg|blender|obs-studio|kicad|openscad)
            echo "execution"
            ;;
        btcpay|erpnext|odoo|keycloak|mail-in-a-box|home-assistant)
            echo "services"
            ;;
        *)
            echo "utility"
            ;;
    esac
}

#######################################
# Extract resource dependencies
# Arguments:
#   $1 - Path to resource directory
# Returns: Dependencies information
#######################################
qdrant::extract::resource_dependencies() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    local resource_name=$(basename "$dir")
    local content=""
    local deps=""
    
    # Check package.json for Node.js resources
    if [[ -f "$dir/package.json" ]]; then
        local dep_count=$(jq -r '.dependencies | length' "$dir/package.json" 2>/dev/null || echo "0")
        local dev_dep_count=$(jq -r '.devDependencies | length' "$dir/package.json" 2>/dev/null || echo "0")
        
        content="Resource Dependencies: $resource_name
Type: Node.js/npm
Dependencies: $dep_count
Dev Dependencies: $dev_dep_count"
        
        # Extract key dependencies
        local key_deps=$(jq -r '.dependencies | keys | .[]' "$dir/package.json" 2>/dev/null | head -10 | tr '\n' ', ' | sed 's/,$//')
        [[ -n "$key_deps" ]] && content="$content
Key Packages: $key_deps"
    fi
    
    # Check requirements.txt for Python resources
    if [[ -f "$dir/requirements.txt" ]]; then
        [[ -n "$content" ]] && content="$content
---SEPARATOR---
"
        local req_count=$(grep -cE "^[a-zA-Z]" "$dir/requirements.txt" 2>/dev/null || echo "0")
        
        content="${content}Resource Dependencies: $resource_name
Type: Python/pip
Dependencies: $req_count"
        
        # Extract key dependencies
        local key_deps=$(head -10 "$dir/requirements.txt" | grep -E "^[a-zA-Z]" | cut -d= -f1 | cut -d'<' -f1 | cut -d'>' -f1 | tr '\n' ', ' | sed 's/,$//')
        [[ -n "$key_deps" ]] && content="$content
Key Packages: $key_deps"
    fi
    
    # Check go.mod for Go resources
    if [[ -f "$dir/go.mod" ]]; then
        [[ -n "$content" ]] && content="$content
---SEPARATOR---
"
        local dep_count=$(grep -c "^\s*require " "$dir/go.mod" 2>/dev/null || echo "0")
        
        content="${content}Resource Dependencies: $resource_name
Type: Go modules
Dependencies: $dep_count"
    fi
    
    # Check for Docker Compose dependencies
    if [[ -f "$dir/docker-compose.yml" ]] || [[ -f "$dir/docker-compose.yaml" ]]; then
        local compose_file=$(ls "$dir"/docker-compose.y*ml 2>/dev/null | head -1)
        [[ -n "$content" ]] && content="$content
---SEPARATOR---
"
        
        # Extract services
        local services=$(grep "^  [a-z]" "$compose_file" 2>/dev/null | sed 's/:$//' | tr -d ' ' | tr '\n' ', ' | sed 's/,$//')
        
        content="${content}Resource Services: $resource_name
Type: Docker Compose
File: $compose_file
Services: $services"
    fi
    
    echo "$content"
}

#######################################
# Extract integration information
# Arguments:
#   $1 - Path to resource directory
# Returns: Integration details
#######################################
qdrant::extract::resource_integrations() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    local resource_name=$(basename "$dir")
    local integrations=""
    
    # Check for adapter directories
    if [[ -d "$dir/adapters" ]]; then
        for adapter_dir in "$dir/adapters"/*; do
            if [[ -d "$adapter_dir" ]]; then
                local adapter_name=$(basename "$adapter_dir")
                integrations="${integrations}${adapter_name} "
            fi
        done
    fi
    
    # Check for integration patterns in code
    local cli_file="$dir/cli.sh"
    if [[ -f "$cli_file" ]]; then
        # Check for other resource references
        grep -h "resource-" "$cli_file" 2>/dev/null | grep -oE "resource-[a-z-]+" | sort -u | while read -r ref; do
            local ref_name=${ref#resource-}
            if [[ "$ref_name" != "$resource_name" ]]; then
                integrations="${integrations}${ref_name} "
            fi
        done
    fi
    
    if [[ -n "$integrations" ]]; then
        echo "Resource Integrations: $resource_name
Integrates With: $integrations
Type: Cross-Resource Integration"
    fi
}

#######################################
# Extract all resource information
# Arguments:
#   $1 - Directory path (optional, defaults to current)
# Returns: 0 on success
#######################################
qdrant::extract::resources_batch() {
    local dir="${1:-.}"
    local output_file="${2:-${EXTRACT_TEMP_DIR}/resources.txt}"
    
    mkdir -p "$(dirname "$output_file")"
    
    # Find all resource directories
    local resource_dirs=()
    
    # Check if caching is available for performance optimization
    if [[ -f "${EMBEDDINGS_DIR}/lib/cache.sh" ]]; then
        source "${EMBEDDINGS_DIR}/lib/cache.sh"
        # Use cached resource locations if available
        while IFS= read -r resource_dir; do
            resource_dirs+=("$resource_dir")
        done < <(qdrant::cache::get_resources 2>/dev/null)
    fi
    
    # If no cached results, use direct discovery
    if [[ ${#resource_dirs[@]} -eq 0 ]]; then
        # Look for resources in NEW structure first (flat under resources/)
        if [[ -d "$dir/resources" ]]; then
            # Optimized single-level find for flatter structure
            while IFS= read -r resource_dir; do
                resource_dirs+=("$resource_dir")
            done < <(find "$dir/resources" -mindepth 1 -maxdepth 1 -type d -exec test -f {}/cli.sh \; -print 2>/dev/null)
        fi
        
        # Fallback to OLD structure if no resources found in new location
        if [[ ${#resource_dirs[@]} -eq 0 ]] && [[ -d "$dir/scripts/resources" ]]; then
            while IFS= read -r resource_dir; do
                if [[ -f "$resource_dir/cli.sh" ]]; then
                    resource_dirs+=("$resource_dir")
                fi
            done < <(find "$dir/scripts/resources" -mindepth 2 -maxdepth 3 -type d 2>/dev/null)
        fi
    fi
    
    # Also check for .vrooli/service.json resources
    if [[ -f "$dir/.vrooli/service.json" ]]; then
        local resource_names=$(jq -r '.resources | keys[]' "$dir/.vrooli/service.json" 2>/dev/null)
        for resource_name in $resource_names; do
            # Try new structure first
            local resource_path="$dir/resources/$resource_name"
            if [[ -d "$resource_path" ]] && [[ ! " ${resource_dirs[@]} " =~ " $resource_path " ]]; then
                resource_dirs+=("$resource_path")
            else
                # Fallback to old structure
                resource_path="$dir/scripts/resources/*/$resource_name"
                local found_dir=$(ls -d $resource_path 2>/dev/null | head -1)
                if [[ -d "$found_dir" ]] && [[ ! " ${resource_dirs[@]} " =~ " $found_dir " ]]; then
                    resource_dirs+=("$found_dir")
                fi
            fi
        done
    fi
    
    if [[ ${#resource_dirs[@]} -eq 0 ]]; then
        log::warn "No resource directories found in $dir"
        return 0
    fi
    
    log::info "Extracting content from ${#resource_dirs[@]} resources"
    
    # Extract content from each resource
    local count=0
    for resource_dir in "${resource_dirs[@]}"; do
        local content=""
        
        # Extract CLI information
        if [[ -f "$resource_dir/cli.sh" ]]; then
            content=$(qdrant::extract::resource_cli "$resource_dir/cli.sh")
        fi
        
        # Extract configuration
        local config=$(qdrant::extract::resource_config "$resource_dir")
        [[ -n "$config" ]] && content="${content}
---SEPARATOR---
$config"
        
        # Extract documentation
        local docs=$(qdrant::extract::resource_docs "$resource_dir")
        [[ -n "$docs" ]] && content="${content}
---SEPARATOR---
$docs"
        
        # Extract dependencies
        local deps=$(qdrant::extract::resource_dependencies "$resource_dir")
        [[ -n "$deps" ]] && content="${content}
---SEPARATOR---
$deps"
        
        # Extract integrations
        local integrations=$(qdrant::extract::resource_integrations "$resource_dir")
        [[ -n "$integrations" ]] && content="${content}
---SEPARATOR---
$integrations"
        
        if [[ -n "$content" ]]; then
            echo "$content" >> "$output_file"
            echo "---SEPARATOR---" >> "$output_file"
            ((count++))
        fi
    done
    
    log::success "Extracted content from $count resources"
    echo "$output_file"
}

#######################################
# Create metadata for resource embedding
# Arguments:
#   $1 - Resource directory path
# Returns: JSON metadata
#######################################
qdrant::extract::resource_metadata() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        echo "{}"
        return
    fi
    
    local resource_name=$(basename "$dir")
    # Detect resource type - handle both old and new structures
    local parent_dir=$(basename "$(dirname "$dir")")
    local resource_type=""
    if [[ "$parent_dir" == "resources" ]]; then
        # New structure - detect type from resource name
        resource_type=$(qdrant::extract::detect_resource_category "$resource_name")
    else
        # Old structure - use parent directory as type
        resource_type="$parent_dir"
    fi
    
    # Count components
    local lib_count=$(find "$dir/lib" -type f -name "*.sh" 2>/dev/null | wc -l)
    local adapter_count=$(find "$dir/adapters" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | wc -l)
    local doc_count=$(find "$dir" -maxdepth 2 -name "*.md" -type f 2>/dev/null | wc -l)
    
    # Check for key files
    local has_cli="false"
    local has_config="false"
    local has_docker="false"
    local has_tests="false"
    
    [[ -f "$dir/cli.sh" ]] && has_cli="true"
    [[ -f "$dir/config.yaml" ]] || [[ -f "$dir/config.yml" ]] && has_config="true"
    [[ -f "$dir/Dockerfile" ]] || [[ -f "$dir/docker-compose.yml" ]] && has_docker="true"
    [[ -d "$dir/__test__" ]] || [[ -d "$dir/tests" ]] && has_tests="true"
    
    # Check if enabled in service.json
    local is_enabled="false"
    if [[ -f ".vrooli/service.json" ]]; then
        is_enabled=$(jq -r ".resources.$resource_name.enabled // false" ".vrooli/service.json" 2>/dev/null)
    fi
    
    # Get directory stats
    local dir_size=$(du -sk "$dir" 2>/dev/null | cut -f1)
    local file_count=$(find "$dir" -type f 2>/dev/null | wc -l)
    local modified=$(stat -f%Sm -t '%Y-%m-%dT%H:%M:%S' "$dir" 2>/dev/null || stat -c %y "$dir" 2>/dev/null | cut -d' ' -f1-2 | tr ' ' 'T')
    
    # Build metadata JSON
    jq -n \
        --arg name "$resource_name" \
        --arg resource_type "$resource_type" \
        --arg dir "$dir" \
        --arg lib_count "$lib_count" \
        --arg adapter_count "$adapter_count" \
        --arg doc_count "$doc_count" \
        --arg has_cli "$has_cli" \
        --arg has_config "$has_config" \
        --arg has_docker "$has_docker" \
        --arg has_tests "$has_tests" \
        --arg is_enabled "$is_enabled" \
        --arg dir_size "$dir_size" \
        --arg file_count "$file_count" \
        --arg modified "$modified" \
        --arg type "resource" \
        --arg extractor "resources" \
        '{
            resource_name: $name,
            resource_type: $resource_type,
            source_directory: $dir,
            library_files: ($lib_count | tonumber),
            adapter_count: ($adapter_count | tonumber),
            documentation_files: ($doc_count | tonumber),
            has_cli: ($has_cli == "true"),
            has_config: ($has_config == "true"),
            has_docker: ($has_docker == "true"),
            has_tests: ($has_tests == "true"),
            is_enabled: ($is_enabled == "true"),
            directory_size_kb: ($dir_size | tonumber),
            total_files: ($file_count | tonumber),
            directory_modified: $modified,
            content_type: $type,
            extractor: $extractor
        }'
}

#######################################
# Generate resource summary report
# Arguments:
#   $1 - Directory to analyze
# Returns: Summary report
#######################################
qdrant::extract::resources_summary() {
    local dir="${1:-.}"
    
    echo "=== Resource Analysis Summary ==="
    echo
    
    # Find all resources
    local resource_dirs=()
    # Check new structure first
    if [[ -d "$dir/resources" ]]; then
        while IFS= read -r resource_dir; do
            if [[ -f "$resource_dir/cli.sh" ]]; then
                resource_dirs+=("$resource_dir")
            fi
        done < <(find "$dir/resources" -mindepth 1 -maxdepth 1 -type d 2>/dev/null)
    fi
    # Fallback to old structure
    if [[ ${#resource_dirs[@]} -eq 0 ]] && [[ -d "$dir/scripts/resources" ]]; then
        while IFS= read -r resource_dir; do
            if [[ -f "$resource_dir/cli.sh" ]]; then
                resource_dirs+=("$resource_dir")
            fi
        done < <(find "$dir/scripts/resources" -mindepth 2 -maxdepth 3 -type d 2>/dev/null)
    fi
    
    if [[ ${#resource_dirs[@]} -eq 0 ]]; then
        echo "No resources found"
        return 0
    fi
    
    echo "Total Resources: ${#resource_dirs[@]}"
    echo
    
    # Categorize resources
    local ai_resources=()
    local storage_resources=()
    local automation_resources=()
    local agent_resources=()
    local other_resources=()
    
    for resource_dir in "${resource_dirs[@]}"; do
        local resource_name=$(basename "$resource_dir")
        # Detect category - new structure doesn't have category dirs
        local parent_dir=$(basename "$(dirname "$resource_dir")")
        local resource_category=""
        if [[ "$parent_dir" == "resources" ]]; then
            # New structure - need to detect category from resource name
            resource_category=$(qdrant::extract::detect_resource_category "$resource_name")
        else
            # Old structure - use parent directory as category
            resource_category="$parent_dir"
        fi
        
        case "$resource_category" in
            ai|llm|inference)
                ai_resources+=("$resource_name")
                ;;
            storage|database|cache)
                storage_resources+=("$resource_name")
                ;;
            automation|workflow)
                automation_resources+=("$resource_name")
                ;;
            agents|agent)
                agent_resources+=("$resource_name")
                ;;
            *)
                other_resources+=("$resource_name")
                ;;
        esac
    done
    
    # Display by category
    if [[ ${#ai_resources[@]} -gt 0 ]]; then
        echo "🤖 AI/Inference Resources: ${#ai_resources[@]}"
        for resource in "${ai_resources[@]}"; do
            echo "  • $resource"
        done
        echo
    fi
    
    if [[ ${#storage_resources[@]} -gt 0 ]]; then
        echo "💾 Storage Resources: ${#storage_resources[@]}"
        for resource in "${storage_resources[@]}"; do
            echo "  • $resource"
        done
        echo
    fi
    
    if [[ ${#automation_resources[@]} -gt 0 ]]; then
        echo "⚙️ Automation Resources: ${#automation_resources[@]}"
        for resource in "${automation_resources[@]}"; do
            echo "  • $resource"
        done
        echo
    fi
    
    if [[ ${#agent_resources[@]} -gt 0 ]]; then
        echo "🤖 Agent Resources: ${#agent_resources[@]}"
        for resource in "${agent_resources[@]}"; do
            echo "  • $resource"
        done
        echo
    fi
    
    if [[ ${#other_resources[@]} -gt 0 ]]; then
        echo "📦 Other Resources: ${#other_resources[@]}"
        for resource in "${other_resources[@]}"; do
            echo "  • $resource"
        done
        echo
    fi
    
    # Check enabled resources
    if [[ -f "$dir/.vrooli/service.json" ]]; then
        echo "Enabled Resources:"
        local enabled_count=0
        for resource_dir in "${resource_dirs[@]}"; do
            local resource_name=$(basename "$resource_dir")
            local is_enabled=$(jq -r ".resources.$resource_name.enabled // false" "$dir/.vrooli/service.json" 2>/dev/null)
            if [[ "$is_enabled" == "true" ]]; then
                echo "  ✅ $resource_name"
                ((enabled_count++))
            fi
        done
        
        if [[ $enabled_count -eq 0 ]]; then
            echo "  None currently enabled"
        fi
        echo
    fi
    
    # Integration analysis
    echo "Resource Integrations:"
    local integration_count=0
    for resource_dir in "${resource_dirs[@]}"; do
        if [[ -d "$resource_dir/adapters" ]]; then
            local adapters=$(ls -d "$resource_dir/adapters"/* 2>/dev/null | wc -l)
            if [[ $adapters -gt 0 ]]; then
                local resource_name=$(basename "$resource_dir")
                echo "  • $resource_name has $adapters adapter(s)"
                ((integration_count++))
            fi
        fi
    done
    
    if [[ $integration_count -eq 0 ]]; then
        echo "  No inter-resource adapters found"
    fi
}