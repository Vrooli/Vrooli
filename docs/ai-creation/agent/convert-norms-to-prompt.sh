#!/bin/bash

# Script to convert agent norms to prompt structure
# This converts the old norms array to the new prompt object

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
    cat << EOF
Convert Agent Norms to Prompt Structure

USAGE:
    $0 [OPTIONS] [FILES...]

OPTIONS:
    --directory DIR     Convert all agents in directory
    --dry-run          Show what would be changed without modifying files
    --help             Show this help message

EXAMPLES:
    # Convert specific agent file
    $0 staged/coordinator/workflow-coordinator.json

    # Convert all agents in directory
    $0 --directory staged/

    # Dry run to see changes
    $0 --dry-run --directory staged/
EOF
}

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $*" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

convert_norms_to_prompt() {
    local file="$1"
    local dry_run="$2"
    
    # Check if file has norms
    if ! jq -e '.norms' "$file" >/dev/null 2>&1; then
        log "No norms found in $file - skipping"
        return 0
    fi
    
    # Extract norms and convert to natural language
    local prompt_content=""
    local obligations=""
    local permissions=""
    local prohibitions=""
    
    # Extract norms by modality
    obligations=$(jq -r '.norms[]? | select(.modality == "obligation") | .target' "$file" 2>/dev/null | tr '\n' '|' | sed 's/|$//')
    permissions=$(jq -r '.norms[]? | select(.modality == "permission") | .target' "$file" 2>/dev/null | tr '\n' '|' | sed 's/|$//')
    prohibitions=$(jq -r '.norms[]? | select(.modality == "prohibition") | .target' "$file" 2>/dev/null | tr '\n' '|' | sed 's/|$//')
    
    # Build prompt content
    local parts=()
    
    if [[ -n "$obligations" ]]; then
        # Convert hyphenated phrases to natural language
        local obligation_text=$(echo "$obligations" | sed 's/|/, /g' | sed 's/-/ /g')
        parts+=("You must ${obligation_text}")
    fi
    
    if [[ -n "$permissions" ]]; then
        local permission_text=$(echo "$permissions" | sed 's/|/, /g' | sed 's/-/ /g')
        parts+=("You are permitted to ${permission_text}")
    fi
    
    if [[ -n "$prohibitions" ]]; then
        local prohibition_text=$(echo "$prohibitions" | sed 's/|/, /g' | sed 's/-/ /g')
        parts+=("Never ${prohibition_text}")
    fi
    
    # Join parts with periods
    prompt_content=$(printf "%s. " "${parts[@]}" | sed 's/\. $//')
    
    if [[ -z "$prompt_content" ]]; then
        log "No norms content found in $file - skipping"
        return 0
    fi
    
    log "Converting norms in $file"
    log "Generated prompt: $prompt_content"
    
    if [[ "$dry_run" == "true" ]]; then
        echo "Would convert $file:"
        echo "  From norms: $(jq -c '.norms' "$file")"
        echo "  To prompt: {\"mode\": \"supplement\", \"source\": \"direct\", \"content\": \"$prompt_content\"}"
        return 0
    fi
    
    # Create the new structure with prompt instead of norms
    local temp_file="${file}.tmp"
    
    # Remove norms and add prompt
    jq --arg content "$prompt_content" '
        del(.norms) | 
        .prompt = {
            "mode": "supplement",
            "source": "direct",
            "content": $content
        }
    ' "$file" > "$temp_file"
    
    # Replace original file
    mv "$temp_file" "$file"
    success "Converted $file"
}

# Parse command line arguments
DRY_RUN=false
DIRECTORY=""
FILES=()

while [[ $# -gt 0 ]]; do
    case $1 in
        --directory)
            DIRECTORY="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        -*)
            error "Unknown option: $1"
            usage
            exit 1
            ;;
        *)
            FILES+=("$1")
            shift
            ;;
    esac
done

# Collect files to convert
if [[ -n "$DIRECTORY" ]]; then
    if [[ ! -d "$DIRECTORY" ]]; then
        error "Directory not found: $DIRECTORY"
        exit 1
    fi
    mapfile -t FILES < <(find "$DIRECTORY" -name "*.json" -type f)
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
    error "No files specified for conversion"
    usage
    exit 1
fi

# Main conversion loop
total_files=0
converted_files=0

for file in "${FILES[@]}"; do
    ((total_files++))
    if convert_norms_to_prompt "$file" "$DRY_RUN"; then
        ((converted_files++))
    fi
done

echo ""
echo "======================================"
echo "Conversion Summary:"
echo "  Total files: $total_files"
echo "  Converted files: $converted_files"
echo "======================================"

if [[ "$DRY_RUN" == "true" ]]; then
    echo ""
    echo "This was a dry run. No files were modified."
    echo "Run without --dry-run to apply changes."
fi

exit 0