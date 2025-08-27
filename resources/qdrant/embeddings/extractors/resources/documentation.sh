#!/usr/bin/env bash
# Resource Documentation Extractor
# Extracts and processes markdown documentation from resource directories
#
# This simplified extractor uses the markdown parser for reliable extraction
# and outputs structured JSON lines with rich metadata.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source the markdown parser
source "${APP_ROOT}/resources/qdrant/embeddings/lib/markdown-parser.sh"

#######################################
# Extract documentation from a resource directory
# 
# Automatically discovers and processes all markdown files using:
# 1. Marked sections (<!-- EMBED:NAME:START/END -->) when present
# 2. Hierarchical sections (H1 headers with subsections) otherwise
# 3. Document overview as fallback
#
# Arguments:
#   $1 - Path to resource directory
# Returns: JSON lines of extracted content with metadata
#######################################
qdrant::extract::resource_documentation() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        log::error "Directory not found: $dir" >&2
        return 1
    fi
    
    local resource_name=$(basename "$dir")
    log::info "Extracting documentation for resource: $resource_name" >&2
    
    # Find all markdown files in the resource directory and docs subdirectory
    local markdown_files=()
    
    # First check for README.md in root (highest priority)
    [[ -f "$dir/README.md" ]] && markdown_files+=("$dir/README.md")
    
    # Then find all other markdown files
    while IFS= read -r file; do
        # Skip if already added (avoid duplicate README)
        [[ "$file" == "$dir/README.md" ]] && continue
        markdown_files+=("$file")
    done < <(find "$dir" -type f -name "*.md" 2>/dev/null | sort)
    
    if [[ ${#markdown_files[@]} -eq 0 ]]; then
        log::warn "No markdown files found in $dir" >&2
        return 1
    fi
    
    log::debug "Found ${#markdown_files[@]} markdown files to process" >&2
    
    local total_sections=0
    
    # Process each markdown file
    for file in "${markdown_files[@]}"; do
        local filename=$(basename "$file")
        local relative_path="${file#$dir/}"
        local doc_type="${filename%.md}"
        
        log::debug "Processing: $relative_path" >&2
        
        # Try marked sections first (highest fidelity)
        local sections=$(markdown::extract_marked_sections "$file")
        local section_count=$(echo "$sections" | jq 'length')
        
        if [[ $section_count -gt 0 ]]; then
            log::debug "  Found $section_count marked sections" >&2
            
            # Output each marked section as a JSON line
            echo "$sections" | jq -c '.[]' | while read -r section; do
                local title=$(echo "$section" | jq -r '.title // empty')
                local content=$(echo "$section" | jq -r '.content // empty')
                
                # Skip empty content
                [[ -z "$content" || "$content" == "null" ]] && continue
                
                # Build enriched content with context
                local enriched_content="Resource: $resource_name"
                enriched_content="$enriched_content | Document: $doc_type"
                [[ -n "$title" ]] && enriched_content="$enriched_content | Section: $title"
                enriched_content="$enriched_content | Content: $content"
                
                # Output as JSON line with full metadata
                jq -n \
                    --arg content "$enriched_content" \
                    --arg resource "$resource_name" \
                    --arg source_file "$file" \
                    --arg relative_path "$relative_path" \
                    --arg filename "$filename" \
                    --arg doc_type "$doc_type" \
                    --arg section "$title" \
                    --arg extraction_method "marked" \
                    '{
                        content: $content,
                        metadata: {
                            resource: $resource,
                            source_file: $source_file,
                            relative_path: $relative_path,
                            filename: $filename,
                            doc_type: $doc_type,
                            section: $section,
                            extraction_method: $extraction_method,
                            content_type: "resource_documentation"
                        }
                    }' | jq -c
                
                ((total_sections++))
            done
        else
            # No marked sections, try hierarchical extraction
            sections=$(markdown::extract_hierarchical_sections "$file")
            section_count=$(echo "$sections" | jq 'length')
            
            if [[ $section_count -gt 0 ]]; then
                log::debug "  Found $section_count hierarchical sections" >&2
                
                # Output each H1 section with its subsections
                echo "$sections" | jq -c '.[]' | while read -r section; do
                    local title=$(echo "$section" | jq -r '.title // empty')
                    local content=$(echo "$section" | jq -r '.content // empty')
                    
                    # Skip very short content (less than 100 chars is probably not useful)
                    [[ -z "$content" || "$content" == "null" || ${#content} -lt 100 ]] && continue
                    
                    # Build enriched content
                    local enriched_content="Resource: $resource_name"
                    enriched_content="$enriched_content | Document: $doc_type"
                    [[ -n "$title" ]] && enriched_content="$enriched_content | Section: $title"
                    enriched_content="$enriched_content | Content: $content"
                    
                    # Output as JSON line
                    jq -n \
                        --arg content "$enriched_content" \
                        --arg resource "$resource_name" \
                        --arg source_file "$file" \
                        --arg relative_path "$relative_path" \
                        --arg filename "$filename" \
                        --arg doc_type "$doc_type" \
                        --arg section "$title" \
                        --arg extraction_method "hierarchical" \
                        '{
                            content: $content,
                            metadata: {
                                resource: $resource,
                                source_file: $source_file,
                                relative_path: $relative_path,
                                filename: $filename,
                                doc_type: $doc_type,
                                section: $section,
                                extraction_method: $extraction_method,
                                content_type: "resource_documentation"
                            }
                        }' | jq -c
                    
                    ((total_sections++))
                done
            else
                # Fallback to document overview
                log::debug "  Using document overview extraction" >&2
                
                sections=$(markdown::parse_sections "$file" 2000)
                
                echo "$sections" | jq -c '.[]' | while read -r section; do
                    local content=$(echo "$section" | jq -r '.content // empty')
                    
                    # Skip empty content
                    [[ -z "$content" || "$content" == "null" || ${#content} -lt 50 ]] && continue
                    
                    # Build enriched content for overview
                    local enriched_content="Resource: $resource_name"
                    enriched_content="$enriched_content | Document: $doc_type (Overview)"
                    enriched_content="$enriched_content | Content: $content"
                    
                    # Output as JSON line
                    jq -n \
                        --arg content "$enriched_content" \
                        --arg resource "$resource_name" \
                        --arg source_file "$file" \
                        --arg relative_path "$relative_path" \
                        --arg filename "$filename" \
                        --arg doc_type "$doc_type" \
                        --arg extraction_method "overview" \
                        '{
                            content: $content,
                            metadata: {
                                resource: $resource,
                                source_file: $source_file,
                                relative_path: $relative_path,
                                filename: $filename,
                                doc_type: $doc_type,
                                extraction_method: $extraction_method,
                                content_type: "resource_documentation"
                            }
                        }' | jq -c
                    
                    ((total_sections++))
                done
            fi
        fi
    done
    
    log::success "Extracted $total_sections sections from ${#markdown_files[@]} files" >&2
}

# Export for use by resources.sh
export -f qdrant::extract::resource_documentation