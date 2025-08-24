#!/usr/bin/env bash
# N8n Workflow Content Extractor for Qdrant Embeddings
# Extracts embeddable content from n8n workflow JSON files

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

# Temporary directory for extracted content
EXTRACT_TEMP_DIR="/tmp/qdrant-workflow-extract-$$"
trap "rm -rf $EXTRACT_TEMP_DIR" EXIT

#######################################
# Extract content from a single workflow file
# Arguments:
#   $1 - Path to workflow JSON file
# Returns: Extracted content as text
#######################################
qdrant::extract::workflow() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "Workflow file not found: $file"
        return 1
    fi
    
    # Extract key workflow information
    local name=$(jq -r '.name // "Unnamed Workflow"' "$file" 2>/dev/null || echo "Unknown")
    local description=$(jq -r '.description // ""' "$file" 2>/dev/null || echo "")
    local active=$(jq -r '.active // false' "$file" 2>/dev/null || echo "false")
    local tags=$(jq -r '.tags[]?.name // empty' "$file" 2>/dev/null | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Extract node types and count
    local node_types=$(jq -r '[.nodes[].type] | unique | join(", ")' "$file" 2>/dev/null || echo "")
    local node_count=$(jq -r '.nodes | length' "$file" 2>/dev/null || echo "0")
    
    # Extract trigger information
    local triggers=$(jq -r '.nodes[] | select(.type | contains("trigger") or contains("webhook")) | .type' "$file" 2>/dev/null | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Extract integration information (nodes with credentials)
    local integrations=$(jq -r '.nodes[].credentials // empty | keys[]' "$file" 2>/dev/null | sort -u | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Extract webhook paths if any
    local webhook_paths=$(jq -r '.nodes[] | select(.type == "n8n-nodes-base.webhook") | .parameters.path // empty' "$file" 2>/dev/null | tr '\n' ', ' | sed 's/,$//' || echo "")
    
    # Extract comments from sticky notes
    local notes=$(jq -r '.nodes[] | select(.type == "n8n-nodes-base.stickyNote") | .parameters.content // empty' "$file" 2>/dev/null | tr '\n' ' ' || echo "")
    
    # Build semantically-rich embeddable content
    local content="This is an N8n workflow named '$name' that automates business processes."
    
    if [[ -n "$description" ]]; then
        content="$content The workflow's purpose is: $description"
    fi
    
    # Add status in natural language
    if [[ "$active" == "true" ]]; then
        content="$content This workflow is currently active and running in production."
    else
        content="$content This workflow is currently inactive."
    fi
    
    # Describe functionality
    content="$content It contains $node_count processing nodes."
    
    # Describe triggers in natural language
    if [[ -n "$triggers" ]]; then
        local trigger_desc="${triggers//n8n-nodes-base./}"
        trigger_desc="${trigger_desc//,/ and}"
        content="$content The workflow can be triggered by: $trigger_desc."
    fi
    
    # Describe webhook endpoints
    if [[ -n "$webhook_paths" ]]; then
        content="$content It exposes API endpoints at paths: $webhook_paths."
    fi
    
    # Describe integrations
    if [[ -n "$integrations" ]]; then
        content="$content It integrates with these services: $integrations."
    fi
    
    # Describe functionality based on node types
    if [[ "$node_types" == *"httpRequest"* ]]; then
        content="$content Makes HTTP API calls to external services."
    fi
    if [[ "$node_types" == *"postgres"* ]] || [[ "$node_types" == *"mysql"* ]]; then
        content="$content Performs database operations for data storage."
    fi
    if [[ "$node_types" == *"code"* ]] || [[ "$node_types" == *"function"* ]]; then
        content="$content Contains custom JavaScript code for data processing."
    fi
    if [[ "$node_types" == *"if"* ]] || [[ "$node_types" == *"switch"* ]]; then
        content="$content Implements conditional logic and decision trees."
    fi
    if [[ "$node_types" == *"email"* ]]; then
        content="$content Sends automated email notifications."
    fi
    
    # Add tags as categories
    if [[ -n "$tags" ]]; then
        content="$content Categories: $tags."
    fi
    
    # Add notes if present
    if [[ -n "$notes" ]]; then
        content="$content Additional context: $notes"
    fi
    
    # Add file location for reference
    content="$content File location: $file"
    
    # Add workflow purpose analysis based on node types
    local purpose=$(qdrant::extract::analyze_workflow_purpose "$file")
    if [[ -n "$purpose" ]]; then
        content="$content
Purpose: $purpose"
    fi
    
    echo "$content"
}

#######################################
# Analyze workflow purpose based on nodes
# Arguments:
#   $1 - Path to workflow JSON file
# Returns: Purpose description
#######################################
qdrant::extract::analyze_workflow_purpose() {
    local file="$1"
    local purpose=""
    
    # Check for common patterns
    local has_webhook=$(jq -r '.nodes[] | select(.type | contains("webhook"))' "$file" 2>/dev/null | wc -l)
    local has_http=$(jq -r '.nodes[] | select(.type | contains("httpRequest"))' "$file" 2>/dev/null | wc -l)
    local has_database=$(jq -r '.nodes[] | select(.type | contains("postgres") or contains("mysql") or contains("mongodb"))' "$file" 2>/dev/null | wc -l)
    local has_ai=$(jq -r '.nodes[] | select(.type | contains("openai") or contains("ollama") or contains("claude"))' "$file" 2>/dev/null | wc -l)
    local has_email=$(jq -r '.nodes[] | select(.type | contains("email") or contains("gmail"))' "$file" 2>/dev/null | wc -l)
    local has_schedule=$(jq -r '.nodes[] | select(.type | contains("cron") or contains("schedule"))' "$file" 2>/dev/null | wc -l)
    local has_file=$(jq -r '.nodes[] | select(.type | contains("readBinary") or contains("writeBinary"))' "$file" 2>/dev/null | wc -l)
    
    # Build purpose description
    local purposes=()
    
    [[ $has_webhook -gt 0 ]] && purposes+=("API endpoint/webhook handler")
    [[ $has_http -gt 0 ]] && purposes+=("External API integration")
    [[ $has_database -gt 0 ]] && purposes+=("Database operations")
    [[ $has_ai -gt 0 ]] && purposes+=("AI/LLM processing")
    [[ $has_email -gt 0 ]] && purposes+=("Email automation")
    [[ $has_schedule -gt 0 ]] && purposes+=("Scheduled automation")
    [[ $has_file -gt 0 ]] && purposes+=("File processing")
    
    if [[ ${#purposes[@]} -gt 0 ]]; then
        purpose=$(IFS=", "; echo "${purposes[*]}")
    fi
    
    echo "$purpose"
}

#######################################
# Extract content from all workflows in a directory
# Arguments:
#   $1 - Directory path (optional, defaults to current)
# Returns: 0 on success
#######################################
qdrant::extract::workflows_batch() {
    local dir="${1:-.}"
    local output_file="${2:-${EXTRACT_TEMP_DIR}/workflows.txt}"
    
    mkdir -p "$(dirname "$output_file")"
    
    # Find all workflow JSON files
    local workflow_files=()
    while IFS= read -r file; do
        workflow_files+=("$file")
    done < <(find "$dir" -type f -name "*.json" -path "*/initialization/*" -o -path "*/n8n/*" 2>/dev/null)
    
    if [[ ${#workflow_files[@]} -eq 0 ]]; then
        log::warn "No workflow files found in $dir"
        return 0
    fi
    
    log::info "Extracting content from ${#workflow_files[@]} workflow files"
    
    # Extract content from each file
    local count=0
    for file in "${workflow_files[@]}"; do
        # Validate it's actually an n8n workflow
        if jq -e '.nodes' "$file" >/dev/null 2>&1; then
            local content=$(qdrant::extract::workflow "$file")
            if [[ -n "$content" ]]; then
                echo "$content" >> "$output_file"
                echo "---SEPARATOR---" >> "$output_file"
                ((count++))
                
                # Show progress
                if [[ $((count % 10)) -eq 0 ]]; then
                    log::debug "Processed $count/${#workflow_files[@]} workflows"
                fi
            fi
        fi
    done
    
    log::success "Extracted content from $count workflows"
    echo "$output_file"
}

#######################################
# Create metadata for workflow embedding
# Arguments:
#   $1 - Workflow file path
# Returns: JSON metadata
#######################################
qdrant::extract::workflow_metadata() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "{}"
        return
    fi
    
    # Extract metadata
    local name=$(jq -r '.name // "Unknown"' "$file" 2>/dev/null)
    local id=$(jq -r '.id // ""' "$file" 2>/dev/null)
    local active=$(jq -r '.active // false' "$file" 2>/dev/null)
    local created=$(jq -r '.createdAt // ""' "$file" 2>/dev/null)
    local updated=$(jq -r '.updatedAt // ""' "$file" 2>/dev/null)
    local node_count=$(jq -r '.nodes | length' "$file" 2>/dev/null || echo "0")
    local tags=$(jq -c '.tags // []' "$file" 2>/dev/null || echo "[]")
    
    # Get file stats
    local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
    local modified=$(stat -f%Sm -t '%Y-%m-%dT%H:%M:%S' "$file" 2>/dev/null || stat -c %y "$file" 2>/dev/null | cut -d' ' -f1-2 | tr ' ' 'T')
    
    # Build metadata JSON
    jq -n \
        --arg name "$name" \
        --arg id "$id" \
        --arg file "$file" \
        --arg active "$active" \
        --arg created "$created" \
        --arg updated "$updated" \
        --arg node_count "$node_count" \
        --argjson tags "$tags" \
        --arg file_size "$file_size" \
        --arg modified "$modified" \
        --arg type "workflow" \
        --arg extractor "workflows" \
        '{
            name: $name,
            workflow_id: $id,
            source_file: $file,
            active: ($active == "true"),
            created_at: $created,
            updated_at: $updated,
            node_count: ($node_count | tonumber),
            tags: $tags,
            file_size: ($file_size | tonumber),
            file_modified: $modified,
            content_type: $type,
            extractor: $extractor
        }'
}

#######################################
# Validate workflow file
# Arguments:
#   $1 - Workflow file path
# Returns: 0 if valid, 1 if not
#######################################
qdrant::extract::validate_workflow() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Check if it's valid JSON
    if ! jq empty "$file" 2>/dev/null; then
        log::debug "Invalid JSON: $file"
        return 1
    fi
    
    # Check if it has required n8n structure
    if ! jq -e '.nodes' "$file" >/dev/null 2>&1; then
        log::debug "Not an n8n workflow (missing nodes): $file"
        return 1
    fi
    
    return 0
}

#######################################
# Find all workflow files in project
# Arguments:
#   $1 - Root directory to search
# Returns: List of workflow files
#######################################
qdrant::extract::find_workflows() {
    local root="${1:-.}"
    
    # Common workflow locations
    local patterns=(
        "*/initialization/n8n/*.json"
        "*/initialization/automation/*.json"
        "*/initialization/*.json"
        "*/workflows/*.json"
        "*/.n8n/*.json"
    )
    
    local files=()
    for pattern in "${patterns[@]}"; do
        while IFS= read -r file; do
            if qdrant::extract::validate_workflow "$file"; then
                files+=("$file")
            fi
        done < <(find "$root" -path "$pattern" -type f 2>/dev/null)
    done
    
    # Remove duplicates
    printf '%s\n' "${files[@]}" | sort -u
}

#######################################
# Generate summary of all workflows
# Arguments:
#   $1 - Directory to analyze
# Returns: Summary report
#######################################
qdrant::extract::workflows_summary() {
    local dir="${1:-.}"
    
    echo "=== Workflow Analysis Summary ==="
    echo
    
    local workflow_files=()
    while IFS= read -r file; do
        workflow_files+=("$file")
    done < <(qdrant::extract::find_workflows "$dir")
    
    if [[ ${#workflow_files[@]} -eq 0 ]]; then
        echo "No workflows found"
        return 0
    fi
    
    echo "Total Workflows: ${#workflow_files[@]}"
    echo
    
    # Analyze workflow types
    local active_count=0
    local total_nodes=0
    local node_types=()
    
    for file in "${workflow_files[@]}"; do
        local active=$(jq -r '.active // false' "$file" 2>/dev/null)
        [[ "$active" == "true" ]] && ((active_count++))
        
        local nodes=$(jq -r '.nodes | length' "$file" 2>/dev/null || echo "0")
        total_nodes=$((total_nodes + nodes))
        
        while IFS= read -r type; do
            node_types+=("$type")
        done < <(jq -r '.nodes[].type' "$file" 2>/dev/null)
    done
    
    echo "Active Workflows: $active_count / ${#workflow_files[@]}"
    echo "Total Nodes: $total_nodes"
    echo "Average Nodes per Workflow: $((total_nodes / ${#workflow_files[@]}))"
    echo
    
    # Most common node types
    echo "Most Common Node Types:"
    printf '%s\n' "${node_types[@]}" | sort | uniq -c | sort -rn | head -10 | while read count type; do
        echo "  $count: $type"
    done
    echo
    
    # List workflows by size
    echo "Largest Workflows:"
    for file in "${workflow_files[@]}"; do
        local name=$(jq -r '.name // "Unknown"' "$file" 2>/dev/null)
        local nodes=$(jq -r '.nodes | length' "$file" 2>/dev/null || echo "0")
        echo "$nodes|$name|$file"
    done | sort -rn | head -5 | while IFS='|' read nodes name file; do
        echo "  $nodes nodes: $name"
    done
}

#######################################
# Process workflows using unified embedding service
# Arguments:
#   $1 - App ID
# Returns: Number of workflows processed
#######################################
qdrant::embeddings::process_workflows() {
    local app_id="$1"
    local collection="${app_id}-workflows"
    local count=0
    
    # Extract workflows to temp file
    local output_file="$TEMP_DIR/workflows.txt"
    qdrant::extract::workflows_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No workflows found for processing"
        echo "0"
        return 0
    fi
    
    # Process each workflow through unified embedding service
    local content=""
    local processing_workflow=false
    
    while IFS= read -r line; do
        if [[ "$line" == "This is an N8n workflow named"* ]]; then
            # Start of new workflow content
            processing_workflow=true
            content="$line"
        elif [[ "$line" == "---SEPARATOR---" ]] && [[ "$processing_workflow" == true ]]; then
            # End of workflow, process it
            if [[ -n "$content" ]]; then
                # Extract workflow metadata from content
                local metadata
                metadata=$(qdrant::extract::workflow_metadata_from_content "$content")
                
                # Process through unified embedding service
                if qdrant::embedding::process_item "$content" "workflow" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
            processing_workflow=false
            content=""
        elif [[ "$processing_workflow" == true ]]; then
            # Continue accumulating workflow content
            content="${content}"$'\n'"${line}"
        fi
    done < "$output_file"
    
    log::debug "Created $count workflow embeddings"
    echo "$count"
}

#######################################
# Extract metadata from workflow content text
# Arguments:
#   $1 - Workflow content text
# Returns: JSON metadata object
#######################################
qdrant::extract::workflow_metadata_from_content() {
    local content="$1"
    
    # Extract key information from the content text
    local name
    name=$(echo "$content" | grep -o "named '[^']*'" | sed "s/named '//" | sed "s/'//")
    
    local file_path
    file_path=$(echo "$content" | grep -o "File location: .*" | cut -d: -f2- | sed 's/^ *//')
    
    local active_status="false"
    if [[ "$content" == *"currently active and running in production"* ]]; then
        active_status="true"
    fi
    
    # Count nodes mentioned in content
    local node_count
    node_count=$(echo "$content" | grep -o "contains [0-9]* processing nodes" | grep -o "[0-9]*")
    
    # Extract integrations
    local integrations
    integrations=$(echo "$content" | grep -o "integrates with these services: [^.]*" | cut -d: -f2 | sed 's/^ *//')
    
    # Extract purpose
    local purpose
    purpose=$(echo "$content" | grep -o "Purpose: [^$]*" | cut -d: -f2- | sed 's/^ *//')
    
    # Build metadata JSON
    jq -n \
        --arg name "${name:-Unknown}" \
        --arg file_path "${file_path:-}" \
        --arg active "$active_status" \
        --arg nodes "${node_count:-0}" \
        --arg integrations "${integrations:-}" \
        --arg purpose "${purpose:-}" \
        '{
            workflow_name: $name,
            source_file: $file_path,
            active: ($active == "true"),
            node_count: ($nodes | tonumber),
            integrations: $integrations,
            purpose: $purpose,
            content_type: "workflow",
            extractor: "workflows"
        }'
}

# Export processing function for manage.sh
export -f qdrant::embeddings::process_workflows