#!/usr/bin/env bash
# Documentation Content Extractor for Qdrant Embeddings
# Extracts embeddable content from structured markdown documentation

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"

# Define paths from APP_ROOT
EMBEDDINGS_DIR="${APP_ROOT}/resources/qdrant/embeddings"
EXTRACTOR_DIR="${EMBEDDINGS_DIR}/extractors"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified embedding service
source "${EMBEDDINGS_DIR}/lib/embedding-service.sh"

# Temporary directory for extracted content
EXTRACT_TEMP_DIR="/tmp/qdrant-docs-extract-$$"
trap "rm -rf $EXTRACT_TEMP_DIR" EXIT

# Standard documentation files we look for
STANDARD_DOCS=(
    "ARCHITECTURE.md"
    "SECURITY.md"
    "LESSONS_LEARNED.md"
    "BREAKING_CHANGES.md"
    "PERFORMANCE.md"
    "PATTERNS.md"
)

#######################################
# Extract content between embedding markers
# Arguments:
#   $1 - File path
#   $2 - Start marker pattern (optional)
#   $3 - End marker pattern (optional)
# Returns: Extracted sections
#######################################
qdrant::extract::marked_sections() {
    local file="$1"
    local start_pattern="${2:-<!-- EMBED:.*:START -->}"
    local end_pattern="${3:-<!-- EMBED:.*:END -->}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Extract all marked sections
    awk "
        /$start_pattern/ { capture = 1; section = \"\"; next }
        /$end_pattern/ { 
            if (capture && section != \"\") {
                print section
                print \"---SECTION---\"
            }
            capture = 0
            next
        }
        capture { section = section \$0 \"\\n\" }
    " "$file"
}

#######################################
# Extract content from ARCHITECTURE.md
# Arguments:
#   $1 - Path to ARCHITECTURE.md
# Returns: Extracted content sections
#######################################
qdrant::extract::architecture() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "Architecture doc not found: $file"
        return 1
    fi
    
    local doc_name="Architecture Documentation"
    local file_path="$file"
    
    # Extract marked sections
    local sections=$(qdrant::extract::marked_sections "$file")
    
    if [[ -z "$sections" ]]; then
        # Fallback: Extract major sections by headers
        log::debug "No marked sections found, extracting by headers"
        
        # Extract Design Decisions section
        local design_decisions=$(awk '/^## Design Decisions/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$design_decisions" ]]; then
            echo "Architecture: Design Decisions
File: $file
Content: $design_decisions
---SECTION---"
        fi
        
        # Extract Patterns section
        local patterns=$(awk '/^## Patterns/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$patterns" ]]; then
            echo "Architecture: Patterns
File: $file
Content: $patterns
---SECTION---"
        fi
        
        # Extract Trade-offs section
        local tradeoffs=$(awk '/^## Trade-offs/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$tradeoffs" ]]; then
            echo "Architecture: Trade-offs
File: $file
Content: $tradeoffs
---SECTION---"
        fi
    else
        # Process marked sections
        while IFS= read -r section; do
            if [[ "$section" != "---SECTION---" ]] && [[ -n "$section" ]]; then
                echo "Architecture Documentation
File: $file
Content: $section"
            elif [[ "$section" == "---SECTION---" ]]; then
                echo "---SECTION---"
            fi
        done <<< "$sections"
    fi
}

#######################################
# Extract content from SECURITY.md
# Arguments:
#   $1 - Path to SECURITY.md
# Returns: Extracted content sections
#######################################
qdrant::extract::security() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "Security doc not found: $file"
        return 1
    fi
    
    # Extract marked sections
    local sections=$(qdrant::extract::marked_sections "$file")
    
    if [[ -z "$sections" ]]; then
        # Fallback: Extract key security sections
        local principles=$(awk '/^## Security Principles/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$principles" ]]; then
            echo "Security: Principles
File: $file
Content: $principles
---SECTION---"
        fi
        
        local vulnerabilities=$(awk '/^## Known Vulnerabilities/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$vulnerabilities" ]]; then
            echo "Security: Known Vulnerabilities
File: $file
Content: $vulnerabilities
---SECTION---"
        fi
        
        local mitigations=$(awk '/^## Mitigation Strategies/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$mitigations" ]]; then
            echo "Security: Mitigation Strategies
File: $file
Content: $mitigations
---SECTION---"
        fi
    else
        # Process marked sections
        while IFS= read -r section; do
            if [[ "$section" != "---SECTION---" ]] && [[ -n "$section" ]]; then
                echo "Security Documentation
File: $file
Content: $section"
            elif [[ "$section" == "---SECTION---" ]]; then
                echo "---SECTION---"
            fi
        done <<< "$sections"
    fi
}

#######################################
# Extract content from LESSONS_LEARNED.md
# Arguments:
#   $1 - Path to LESSONS_LEARNED.md
# Returns: Extracted content sections
#######################################
qdrant::extract::lessons() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log::error "Lessons learned doc not found: $file"
        return 1
    fi
    
    # Extract marked sections
    local sections=$(qdrant::extract::marked_sections "$file")
    
    if [[ -z "$sections" ]]; then
        # Fallback: Extract What Worked and What Failed
        local worked=$(awk '/^## What Worked/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$worked" ]]; then
            echo "Lessons Learned: What Worked
File: $file
Content: $worked
---SECTION---"
        fi
        
        local failed=$(awk '/^## What Failed/,/^##[^#]/' "$file" 2>/dev/null | head -n -1)
        if [[ -n "$failed" ]]; then
            echo "Lessons Learned: What Failed
File: $file
Content: $failed
---SECTION---"
        fi
    else
        # Process marked sections
        while IFS= read -r section; do
            if [[ "$section" != "---SECTION---" ]] && [[ -n "$section" ]]; then
                echo "Lessons Learned
File: $file
Content: $section"
            elif [[ "$section" == "---SECTION---" ]]; then
                echo "---SECTION---"
            fi
        done <<< "$sections"
    fi
}

#######################################
# Extract content from any standard doc
# Arguments:
#   $1 - Path to documentation file
# Returns: Extracted content
#######################################
qdrant::extract::standard_doc() {
    local file="$1"
    local filename=$(basename "$file")
    
    case "$filename" in
        "ARCHITECTURE.md")
            qdrant::extract::architecture "$file"
            ;;
        "SECURITY.md")
            qdrant::extract::security "$file"
            ;;
        "LESSONS_LEARNED.md")
            qdrant::extract::lessons "$file"
            ;;
        "BREAKING_CHANGES.md"|"PERFORMANCE.md"|"PATTERNS.md")
            # Generic extraction for other standard docs
            qdrant::extract::generic_doc "$file"
            ;;
        *)
            # Unknown doc type, try generic extraction
            qdrant::extract::generic_doc "$file"
            ;;
    esac
}

#######################################
# Generic doc extraction
# Arguments:
#   $1 - Path to documentation file
# Returns: Extracted content
#######################################
qdrant::extract::generic_doc() {
    local file="$1"
    local filename=$(basename "$file")
    local doc_type="${filename%.md}"
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Try to extract marked sections first
    local sections=$(qdrant::extract::marked_sections "$file")
    
    if [[ -n "$sections" ]]; then
        # Process marked sections
        while IFS= read -r section; do
            if [[ "$section" != "---SECTION---" ]] && [[ -n "$section" ]]; then
                echo "$doc_type Documentation
File: $file
Content: $section"
            elif [[ "$section" == "---SECTION---" ]]; then
                echo "---SECTION---"
            fi
        done <<< "$sections"
    else
        # Fallback: Extract all content
        local content=$(cat "$file")
        echo "$doc_type Documentation
File: $file
Content: $content"
    fi
}

#######################################
# Extract content from all documentation
# Arguments:
#   $1 - Directory path (optional, defaults to current)
# Returns: 0 on success
#######################################
qdrant::extract::docs_batch() {
    local dir="${1:-.}"
    local output_file="${2:-${EXTRACT_TEMP_DIR}/docs.txt}"
    
    mkdir -p "${output_file%/*}"
    
    # Find all standard documentation files
    local doc_files=()
    
    for doc_name in "${STANDARD_DOCS[@]}"; do
        while IFS= read -r file; do
            doc_files+=("$file")
        done < <(find "$dir" -type f -name "$doc_name" 2>/dev/null)
    done
    
    # Also find README files
    while IFS= read -r file; do
        doc_files+=("$file")
    done < <(find "$dir" -type f -name "README.md" -path "*/docs/*" 2>/dev/null)
    
    if [[ ${#doc_files[@]} -eq 0 ]]; then
        log::warn "No documentation files found in $dir"
        return 0
    fi
    
    log::info "Extracting content from ${#doc_files[@]} documentation files"
    
    # Extract content from each file
    local count=0
    for file in "${doc_files[@]}"; do
        local content=$(qdrant::extract::standard_doc "$file")
        if [[ -n "$content" ]]; then
            echo "$content" >> "$output_file"
            echo "---SEPARATOR---" >> "$output_file"
            ((count++))
        fi
    done
    
    log::success "Extracted content from $count documentation files"
    echo "$output_file"
}

#######################################
# Create metadata for documentation embedding
# Arguments:
#   $1 - Documentation file path
# Returns: JSON metadata
#######################################
qdrant::extract::doc_metadata() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "{}"
        return
    fi
    
    local filename=$(basename "$file")
    local doc_type="${filename%.md}"
    local doc_path=${file%/*}
    
    # Count sections
    local section_count=$(grep -c "^##[^#]" "$file" 2>/dev/null || echo "0")
    local marked_sections=$(grep -c "<!-- EMBED:.*:START -->" "$file" 2>/dev/null || echo "0")
    
    # Get file stats
    local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null || echo "0")
    local modified=$(stat -f%Sm -t '%Y-%m-%dT%H:%M:%S' "$file" 2>/dev/null || stat -c %y "$file" 2>/dev/null | cut -d' ' -f1-2 | tr ' ' 'T')
    
    # Check last update date from content (if present)
    local last_entry=$(grep -E "^\[20[0-9]{2}-[0-9]{2}-[0-9]{2}\]" "$file" 2>/dev/null | tail -1 | grep -oE "20[0-9]{2}-[0-9]{2}-[0-9]{2}" || echo "")
    
    # Build metadata JSON
    jq -n \
        --arg filename "$filename" \
        --arg doc_type "$doc_type" \
        --arg file "$file" \
        --arg doc_path "$doc_path" \
        --arg section_count "$section_count" \
        --arg marked_sections "$marked_sections" \
        --arg file_size "$file_size" \
        --arg modified "$modified" \
        --arg last_entry "$last_entry" \
        --arg type "documentation" \
        --arg extractor "docs" \
        '{
            filename: $filename,
            doc_type: $doc_type,
            source_file: $file,
            doc_path: $doc_path,
            section_count: ($section_count | tonumber),
            marked_sections: ($marked_sections | tonumber),
            file_size: ($file_size | tonumber),
            file_modified: $modified,
            last_entry_date: (if $last_entry == "" then null else $last_entry end),
            content_type: $type,
            extractor: $extractor
        }'
}

#######################################
# Validate documentation structure
# Arguments:
#   $1 - Documentation file path
# Returns: 0 if valid, 1 if not
#######################################
qdrant::extract::validate_doc() {
    local file="$1"
    local filename=$(basename "$file")
    
    if [[ ! -f "$file" ]]; then
        return 1
    fi
    
    # Check if it's a known doc type
    local is_standard=false
    for doc_name in "${STANDARD_DOCS[@]}"; do
        if [[ "$filename" == "$doc_name" ]]; then
            is_standard=true
            break
        fi
    done
    
    if [[ "$is_standard" == "true" ]]; then
        # Validate structure based on type
        case "$filename" in
            "ARCHITECTURE.md")
                # Should have Design Decisions section
                if ! grep -q "^## Design Decisions" "$file"; then
                    log::warn "ARCHITECTURE.md missing Design Decisions section"
                fi
                ;;
            "SECURITY.md")
                # Should have Security Principles section
                if ! grep -q "^## Security Principles" "$file"; then
                    log::warn "SECURITY.md missing Security Principles section"
                fi
                ;;
            "LESSONS_LEARNED.md")
                # Should have What Worked/Failed sections
                if ! grep -q "^## What Worked" "$file"; then
                    log::warn "LESSONS_LEARNED.md missing What Worked section"
                fi
                ;;
        esac
    fi
    
    return 0
}

#######################################
# Generate documentation coverage report
# Arguments:
#   $1 - Directory to analyze
# Returns: Coverage report
#######################################
qdrant::extract::docs_coverage() {
    local dir="${1:-.}"
    
    echo "=== Documentation Coverage Report ==="
    echo
    echo "Standard Documentation Files:"
    echo
    
    for doc_name in "${STANDARD_DOCS[@]}"; do
        local found=false
        local path=""
        
        # Check in docs/ directory first
        if [[ -f "$dir/docs/$doc_name" ]]; then
            found=true
            path="$dir/docs/$doc_name"
        elif [[ -f "$dir/$doc_name" ]]; then
            found=true
            path="$dir/$doc_name"
        fi
        
        if [[ "$found" == "true" ]]; then
            local sections=$(grep -c "^##[^#]" "$path" 2>/dev/null || echo "0")
            local marked=$(grep -c "<!-- EMBED:.*:START -->" "$path" 2>/dev/null || echo "0")
            echo "  ✅ $doc_name"
            echo "     Sections: $sections, Marked for embedding: $marked"
        else
            echo "  ❌ $doc_name - MISSING"
        fi
    done
    
    echo
    echo "Recommendations:"
    
    local missing_count=0
    for doc_name in "${STANDARD_DOCS[@]}"; do
        if [[ ! -f "$dir/docs/$doc_name" ]] && [[ ! -f "$dir/$doc_name" ]]; then
            ((missing_count++))
            
            case "$doc_name" in
                "ARCHITECTURE.md")
                    echo "  • Add ARCHITECTURE.md to document design decisions"
                    ;;
                "SECURITY.md")
                    echo "  • Add SECURITY.md to document security considerations"
                    ;;
                "LESSONS_LEARNED.md")
                    echo "  • Add LESSONS_LEARNED.md to capture insights"
                    ;;
                "BREAKING_CHANGES.md")
                    echo "  • Add BREAKING_CHANGES.md for version history"
                    ;;
                "PERFORMANCE.md")
                    echo "  • Add PERFORMANCE.md to track optimizations"
                    ;;
                "PATTERNS.md")
                    echo "  • Add PATTERNS.md to document code patterns"
                    ;;
            esac
        fi
    done
    
    if [[ $missing_count -eq 0 ]]; then
        echo "  All standard documentation files present!"
    fi
}

#######################################
# Process documentation using unified embedding service
# Arguments:
#   $1 - App ID
# Returns: Number of documentation sections processed
#######################################
qdrant::embeddings::process_documentation() {
    local app_id="$1"
    local collection="${app_id}-knowledge"
    local count=0
    
    # Extract documentation to temp file
    local output_file="$TEMP_DIR/docs.txt"
    qdrant::extract::docs_batch "." "$output_file" >&2
    
    if [[ ! -f "$output_file" ]] || [[ ! -s "$output_file" ]]; then
        log::debug "No documentation found for processing"
        echo "0"
        return 0
    fi
    
    # Process each documentation section through unified embedding service
    local content=""
    local processing_doc=false
    
    while IFS= read -r line; do
        if [[ "$line" == *" Documentation" ]]; then
            # Start of new documentation content
            processing_doc=true
            content="$line"
        elif [[ "$line" == "---SEPARATOR---" ]] && [[ "$processing_doc" == true ]]; then
            # End of documentation section, process it
            if [[ -n "$content" ]]; then
                # Extract documentation metadata from content
                local metadata
                metadata=$(qdrant::extract::docs_metadata_from_content "$content")
                
                # Process through unified embedding service
                if qdrant::embedding::process_item "$content" "documentation" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
            processing_doc=false
            content=""
        elif [[ "$line" == "---SECTION---" ]] && [[ "$processing_doc" == true ]]; then
            # Section break within a document - process current section and start new one
            if [[ -n "$content" ]]; then
                local metadata
                metadata=$(qdrant::extract::docs_metadata_from_content "$content")
                
                if qdrant::embedding::process_item "$content" "documentation" "$collection" "$app_id" "$metadata"; then
                    ((count++))
                fi
            fi
            # Start accumulating next section
            content=""
        elif [[ "$processing_doc" == true ]]; then
            # Continue accumulating documentation content
            content="${content}"$'\n'"${line}"
        fi
    done < "$output_file"
    
    log::debug "Created $count documentation embeddings"
    echo "$count"
}

#######################################
# Extract metadata from documentation content text
# Arguments:
#   $1 - Documentation content text
# Returns: JSON metadata object
#######################################
qdrant::extract::docs_metadata_from_content() {
    local content="$1"
    
    # Extract document type from the first line
    local doc_type
    doc_type=$(echo "$content" | head -1 | sed 's/ Documentation$//')
    
    local file_path
    file_path=$(echo "$content" | grep -o "File: .*" | cut -d: -f2- | sed 's/^ *//')
    
    # Extract filename
    local filename=""
    if [[ -n "$file_path" ]]; then
        filename=$(basename "$file_path")
    fi
    
    # Count content sections by looking for markers
    local has_marked_sections="false"
    if [[ "$content" == *"EMBED:"* ]]; then
        has_marked_sections="true"
    fi
    
    # Estimate content complexity
    local content_length
    content_length=$(echo -n "$content" | wc -c)
    
    local line_count
    line_count=$(echo "$content" | wc -l)
    
    # Determine documentation category
    local category="general"
    case "$doc_type" in
        "Architecture") category="technical" ;;
        "Security") category="security" ;;
        "Lessons Learned") category="insights" ;;
        "Performance") category="technical" ;;
        "Breaking Changes") category="versioning" ;;
        "Patterns") category="technical" ;;
    esac
    
    # Build metadata JSON
    jq -n \
        --arg doc_type "$doc_type" \
        --arg filename "${filename:-Unknown}" \
        --arg file_path "${file_path:-}" \
        --arg category "$category" \
        --arg has_marked_sections "$has_marked_sections" \
        --arg content_length "$content_length" \
        --arg line_count "$line_count" \
        '{
            doc_type: $doc_type,
            filename: $filename,
            source_file: $file_path,
            category: $category,
            has_marked_sections: ($has_marked_sections == "true"),
            content_length: ($content_length | tonumber),
            line_count: ($line_count | tonumber),
            content_type: "documentation",
            extractor: "docs"
        }'
}

# Export processing function for manage.sh
export -f qdrant::embeddings::process_documentation

# Export additional functions for testing
export -f qdrant::extract::docs_batch
export -f qdrant::extract::docs_coverage
export -f qdrant::extract::standard_doc
export -f qdrant::extract::architecture
export -f qdrant::extract::security
export -f qdrant::extract::lessons
export -f qdrant::extract::generic_doc