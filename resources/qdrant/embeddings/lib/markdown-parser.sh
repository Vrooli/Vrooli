#!/usr/bin/env bash
# Markdown Section Parser - Reliable extraction of markdown sections into structured JSON
# 
# This parser extracts content from markdown files in multiple ways:
# 1. Marked sections: Content between <!-- EMBED:NAME:START --> and <!-- EMBED:NAME:END -->
# 2. Hierarchical sections: Each H1 header (#) with ALL its subsections as one unit
# 3. Document overview: First 50 lines as a summary
#
# Why this exists:
# - Previous extractors processed sections line-by-line, creating fragmented embeddings
# - Proper multi-line section handling requires careful boundary detection
# - JSON escaping must handle control characters and special formatting
#
# Output format:
# [
#   {
#     "title": "Section Name",      // null if no title
#     "content": "Full content...",  // Complete multi-line content
#     "type": "marked|h1_hierarchical|document"  // Section type
#   }
# ]

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

#######################################
# Extract marked sections with EMBED comments
# 
# Looks for sections marked with HTML comments like:
# <!-- EMBED:SECTION_NAME:START -->
# ... content to embed ...
# <!-- EMBED:SECTION_NAME:END -->
#
# These sections are explicitly marked for embedding by documentation authors.
# Each marked section becomes a single JSON object with the full content between markers.
#
# Arguments:
#   $1 - Path to markdown file
# Returns: 
#   JSON array of sections with fields:
#   - title: The SECTION_NAME from the marker
#   - content: Complete content between START and END markers
#   - type: "marked"
# Example:
#   <!-- EMBED:API_USAGE:START -->
#   Here's how to use the API...
#   Multiple lines of content
#   <!-- EMBED:API_USAGE:END -->
#   Returns: [{"title": "API_USAGE", "content": "Here's how to use the API...\nMultiple lines of content", "type": "marked"}]
#######################################
markdown::extract_marked_sections() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "[]"
        return 1
    fi
    
    # Use a temporary file to collect sections
    local temp_sections="/tmp/marked_sections_$$.txt"
    > "$temp_sections"
    
    local in_section=false
    local section_name=""
    local section_content=""
    
    while IFS= read -r line; do
        # Check for section start
        if [[ "$line" =~ \<\!--[[:space:]]*EMBED:([^:]+):START[[:space:]]*--\> ]]; then
            in_section=true
            section_name="${BASH_REMATCH[1]}"
            section_content=""
            continue
        fi
        
        # Check for section end
        if [[ "$line" =~ \<\!--[[:space:]]*EMBED:[^:]+:END[[:space:]]*--\> ]]; then
            if [[ "$in_section" == "true" && -n "$section_content" ]]; then
                # Write section data to temp file with delimiter
                echo "===SECTION_START===" >> "$temp_sections"
                echo "$section_name" >> "$temp_sections"
                echo "===CONTENT_START===" >> "$temp_sections"
                echo "$section_content" >> "$temp_sections"
                echo "===SECTION_END===" >> "$temp_sections"
            fi
            in_section=false
            section_name=""
            section_content=""
            continue
        fi
        
        # Accumulate section content
        if [[ "$in_section" == "true" ]]; then
            if [[ -n "$section_content" ]]; then
                section_content="${section_content}
${line}"
            else
                section_content="$line"
            fi
        fi
    done < "$file"
    
    # Convert collected sections to JSON using jq
    local json_array="[]"
    local reading_section=false
    local current_title=""
    local current_content=""
    
    while IFS= read -r line; do
        if [[ "$line" == "===SECTION_START===" ]]; then
            reading_section=true
            current_title=""
            current_content=""
        elif [[ "$line" == "===CONTENT_START===" ]]; then
            # Title was the previous line
            continue
        elif [[ "$line" == "===SECTION_END===" ]]; then
            if [[ -n "$current_content" ]]; then
                # Use jq to properly escape and add to array
                json_array=$(echo "$json_array" | jq \
                    --arg title "$current_title" \
                    --arg content "$current_content" \
                    --arg type "marked" \
                    '. += [{title: $title, content: $content, type: $type}]')
            fi
            reading_section=false
        elif [[ "$reading_section" == "true" ]]; then
            if [[ -z "$current_title" && "$line" != "===CONTENT_START===" ]]; then
                current_title="$line"
            elif [[ "$line" != "===CONTENT_START===" ]]; then
                if [[ -n "$current_content" ]]; then
                    current_content="${current_content}
${line}"
                else
                    current_content="$line"
                fi
            fi
        fi
    done < "$temp_sections"
    
    # Cleanup
    rm -f "$temp_sections"
    
    echo "$json_array"
}

#######################################
# Extract hierarchical sections (H1 with all subsections)
#
# Extracts each top-level header (# Header) along with ALL its content until the next H1.
# This includes all subsections (##, ###, etc.), paragraphs, code blocks, lists, etc.
# Useful for documents organized by major topics where each H1 is a complete unit.
#
# Arguments:
#   $1 - Path to markdown file
# Returns:
#   JSON array of sections with fields:
#   - title: The H1 header text (without the # prefix)
#   - content: ALL content under this H1 until the next H1 or end of file
#   - type: "h1_hierarchical"
# Example:
#   # Installation
#   ## Prerequisites
#   - Item 1
#   - Item 2
#   ## Steps
#   1. Do this
#   2. Do that
#   # Usage
#   ...
#   Returns: [
#     {"title": "Installation", "content": "## Prerequisites\n- Item 1\n- Item 2\n## Steps\n1. Do this\n2. Do that", "type": "h1_hierarchical"},
#     {"title": "Usage", "content": "...", "type": "h1_hierarchical"}
#   ]
#######################################
markdown::extract_hierarchical_sections() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "[]"
        return 1
    fi
    
    # Use a temporary file to collect sections
    local temp_sections="/tmp/hierarchical_sections_$$.txt"
    > "$temp_sections"
    
    local current_h1_title=""
    local current_h1_content=""
    local in_h1_section=false
    
    while IFS= read -r line; do
        # Check for H1 header
        if [[ "$line" =~ ^#[[:space:]]+ ]]; then
            # Save previous H1 section if exists
            if [[ "$in_h1_section" == "true" && -n "$current_h1_content" ]]; then
                echo "===SECTION_START===" >> "$temp_sections"
                echo "$current_h1_title" >> "$temp_sections"
                echo "===CONTENT_START===" >> "$temp_sections"
                echo "$current_h1_content" >> "$temp_sections"
                echo "===SECTION_END===" >> "$temp_sections"
            fi
            
            # Start new H1 section
            current_h1_title=$(echo "$line" | sed 's/^#[[:space:]]*//' | sed 's/[[:space:]]*$//')
            current_h1_content=""
            in_h1_section=true
            continue
        fi
        
        # Accumulate all content under current H1
        if [[ "$in_h1_section" == "true" ]]; then
            if [[ -n "$current_h1_content" ]]; then
                current_h1_content="${current_h1_content}
${line}"
            else
                current_h1_content="$line"
            fi
        fi
    done < "$file"
    
    # Save last H1 section
    if [[ "$in_h1_section" == "true" && -n "$current_h1_content" ]]; then
        echo "===SECTION_START===" >> "$temp_sections"
        echo "$current_h1_title" >> "$temp_sections"
        echo "===CONTENT_START===" >> "$temp_sections"
        echo "$current_h1_content" >> "$temp_sections"
        echo "===SECTION_END===" >> "$temp_sections"
    fi
    
    # Convert to JSON
    local json_array="[]"
    local reading_section=false
    local current_title=""
    local current_content=""
    
    while IFS= read -r line; do
        if [[ "$line" == "===SECTION_START===" ]]; then
            reading_section=true
            current_title=""
            current_content=""
        elif [[ "$line" == "===CONTENT_START===" ]]; then
            continue
        elif [[ "$line" == "===SECTION_END===" ]]; then
            if [[ -n "$current_content" ]]; then
                # Use jq to properly escape and add to array
                json_array=$(echo "$json_array" | jq \
                    --arg title "$current_title" \
                    --arg content "$current_content" \
                    --arg type "h1_hierarchical" \
                    '. += [{title: $title, content: $content, type: $type}]')
            fi
            reading_section=false
        elif [[ "$reading_section" == "true" ]]; then
            if [[ -z "$current_title" && "$line" != "===CONTENT_START===" ]]; then
                current_title="$line"
            elif [[ "$line" != "===CONTENT_START===" ]]; then
                if [[ -n "$current_content" ]]; then
                    current_content="${current_content}
${line}"
                else
                    current_content="$line"
                fi
            fi
        fi
    done < "$temp_sections"
    
    # Cleanup
    rm -f "$temp_sections"
    
    echo "$json_array"
}

#######################################
# Parse markdown file into document overview
#
# Creates a high-level summary of the document by extracting the first 50 lines.
# Useful for giving context about what the entire document contains.
# This is typically used as a fallback when other extraction methods don't apply.
#
# Arguments:
#   $1 - Path to markdown file
#   $2 - Optional: Max content length (default: unlimited)
# Returns:
#   JSON array with a single document overview:
#   - title: The filename without .md extension
#   - content: First 50 lines of the document, space-collapsed
#   - type: "document"
# Example:
#   For a file "API.md" with lots of content
#   Returns: [{"title": "API", "content": "First 50 lines joined...", "type": "document"}]
#
# Note: This function intentionally returns minimal content to provide a quick overview
# rather than the complete document. For full content extraction, use the hierarchical
# or marked section extractors.
#######################################
markdown::parse_sections() {
    local file="$1"
    local max_length="${2:-0}"
    
    if [[ ! -f "$file" ]]; then
        echo "[]"
        return 1
    fi
    
    local filename=$(basename "$file" .md)
    
    # Get document overview (first 50 lines)
    local overview_content=$(head -50 "$file" 2>/dev/null | tr '\n' ' ' | sed 's/  */ /g')
    
    if [[ $max_length -gt 0 && ${#overview_content} -gt $max_length ]]; then
        overview_content="${overview_content:0:$max_length}..."
    fi
    
    # Create JSON with overview
    local json_array
    json_array=$(echo "[]" | jq \
        --arg title "$filename" \
        --arg content "$overview_content" \
        --arg type "document" \
        '. += [{title: $title, content: $content, type: $type}]')
    
    echo "$json_array"
}

# Export functions for use by other scripts
export -f markdown::extract_marked_sections  # Extract EMBED-marked sections
export -f markdown::extract_hierarchical_sections  # Extract H1 sections with all subsections
export -f markdown::parse_sections  # Extract document overview (first 50 lines)