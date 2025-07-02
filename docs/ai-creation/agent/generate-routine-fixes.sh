#!/bin/bash

# Routine Name Fix Generator
# This script generates suggestions for fixing routine name mismatches

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

# Read available and missing routines
AVAILABLE_FILE="$AGENT_DIR/available_routines.txt"
MISSING_FILE="$AGENT_DIR/missing_routines.txt"

[[ -f "$AVAILABLE_FILE" ]] || { echo "Available routines file not found: $AVAILABLE_FILE"; exit 1; }
[[ -f "$MISSING_FILE" ]] || { echo "Missing routines file not found: $MISSING_FILE"; exit 1; }

log "Generating routine name fix suggestions..."

# Create output files
ALIAS_SUGGESTIONS="$AGENT_DIR/routine_alias_suggestions.txt"
RENAME_SUGGESTIONS="$AGENT_DIR/routine_rename_suggestions.txt"
HIGH_CONFIDENCE_FIXES="$AGENT_DIR/high_confidence_fixes.json"

# Clear output files
> "$ALIAS_SUGGESTIONS"
> "$RENAME_SUGGESTIONS"
> "$HIGH_CONFIDENCE_FIXES"

echo "# High-Confidence Routine Name Fixes" >> "$ALIAS_SUGGESTIONS"
echo "# Generated on $(date)" >> "$ALIAS_SUGGESTIONS"
echo "" >> "$ALIAS_SUGGESTIONS"

echo "[" >> "$HIGH_CONFIDENCE_FIXES"

FIXES_FOUND=0

# Function to calculate simple similarity score
similarity_score() {
    local missing="$1"
    local available="$2"
    
    # Normalize both names (lowercase, remove special chars, trim spaces)
    local norm_missing=$(echo "$missing" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9 ]//g' | sed 's/  */ /g' | xargs)
    local norm_available=$(echo "$available" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9 ]//g' | sed 's/  */ /g' | xargs)
    
    # Count matching words
    local missing_words=($norm_missing)
    local available_words=($norm_available)
    local matches=0
    local total_missing=${#missing_words[@]}
    
    for word in "${missing_words[@]}"; do
        if [[ " ${available_words[*]} " =~ " ${word} " ]]; then
            ((matches++))
        fi
    done
    
    # Calculate percentage match
    if [[ $total_missing -gt 0 ]]; then
        echo $((matches * 100 / total_missing))
    else
        echo 0
    fi
}

# Check for high-confidence abbreviation matches
log "Checking for abbreviation patterns..."

while IFS= read -r missing_routine; do
    [[ -z "$missing_routine" ]] && continue
    
    best_match=""
    best_score=0
    
    while IFS= read -r available_routine; do
        [[ -z "$available_routine" ]] && continue
        
        score=$(similarity_score "$missing_routine" "$available_routine")
        
        # Check for common abbreviation patterns
        if [[ $score -ge 70 ]]; then
            # Additional checks for abbreviations
            normalized_missing=$(echo "$missing_routine" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9 ]//g')
            normalized_available=$(echo "$available_routine" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9 ]//g')
            
            # Check for common abbreviations
            if [[ "$normalized_missing" =~ manager && "$normalized_available" =~ mgr ]] || \
               [[ "$normalized_missing" =~ coordinator && "$normalized_available" =~ coord ]] || \
               [[ "$normalized_missing" =~ analyzer && "$normalized_available" =~ an ]] || \
               [[ "$normalized_missing" =~ optimizer && "$normalized_available" =~ opt ]] || \
               [[ "$normalized_missing" =~ designer && "$normalized_available" =~ des ]] || \
               [[ "$normalized_missing" =~ validator && "$normalized_available" =~ valid ]] || \
               [[ "$normalized_missing" =~ planner && "$normalized_available" =~ plan ]] || \
               [[ "$normalized_missing" =~ extractor && "$normalized_available" =~ extract ]]; then
                score=$((score + 20))  # Boost score for known abbreviation patterns
            fi
            
            if [[ $score -gt $best_score ]]; then
                best_score=$score
                best_match="$available_routine"
            fi
        fi
    done < "$AVAILABLE_FILE"
    
    # If we found a good match, record it
    if [[ $best_score -ge 85 && -n "$best_match" ]]; then
        echo "# Score: $best_score%" >> "$ALIAS_SUGGESTIONS"
        echo "\"$missing_routine\" -> \"$best_match\"" >> "$ALIAS_SUGGESTIONS"
        echo "" >> "$ALIAS_SUGGESTIONS"
        
        # Generate JSON entry
        if [[ $FIXES_FOUND -gt 0 ]]; then
            echo "," >> "$HIGH_CONFIDENCE_FIXES"
        fi
        cat >> "$HIGH_CONFIDENCE_FIXES" << EOF
  {
    "missing": "$missing_routine",
    "available": "$best_match",
    "confidence": $best_score,
    "action": "create_alias"
  }EOF
        
        ((FIXES_FOUND++))
    fi
    
done < "$MISSING_FILE"

echo "" >> "$HIGH_CONFIDENCE_FIXES"
echo "]" >> "$HIGH_CONFIDENCE_FIXES"

# Generate summary
log "Generating summary report..."

cat > "$RENAME_SUGGESTIONS" << EOF
# Routine Name Standardization Suggestions
# Generated on $(date)

## Summary
- Total missing routines analyzed: $(wc -l < "$MISSING_FILE")
- High-confidence fixes found: $FIXES_FOUND
- Potential reduction in missing routines: $FIXES_FOUND ($(( FIXES_FOUND * 100 / $(wc -l < "$MISSING_FILE") ))%)

## Recommended Actions

### 1. Create Aliases for Existing Routines
See: $ALIAS_SUGGESTIONS

### 2. Standardize Naming Conventions
Common issues found:
- Heavy abbreviation usage (Manager -> Mgr, Coordinator -> Coord)
- Inconsistent word forms (Analyzer vs Analysis, Planner vs Planning)
- Missing word endings (Extract vs Extractor)

### 3. Implementation Options

#### Option A: Create Aliases
Add alias support to routine resolution system to map alternative names to existing routines.

#### Option B: Rename Routines  
Update routine names to match agent expectations (more breaking changes).

#### Option C: Update Agent References
Modify agent definitions to use existing routine names (least breaking).

## High-Confidence Fixes (>85% match)
EOF

if [[ $FIXES_FOUND -gt 0 ]]; then
    echo "" >> "$RENAME_SUGGESTIONS"
    echo "The following $FIXES_FOUND routine name mismatches can be resolved immediately:" >> "$RENAME_SUGGESTIONS"
    echo "" >> "$RENAME_SUGGESTIONS"
    
    jq -r '.[] | "- \(.missing) -> \(.available) (\(.confidence)% confidence)"' "$HIGH_CONFIDENCE_FIXES" >> "$RENAME_SUGGESTIONS"
fi

echo ""
echo "============================================="
echo "ROUTINE NAME FIX ANALYSIS SUMMARY"
echo "============================================="
echo ""
echo "Total missing routines: $(wc -l < "$MISSING_FILE")"
echo "High-confidence fixes found: $FIXES_FOUND"
echo "Potential reduction: $(( FIXES_FOUND * 100 / $(wc -l < "$MISSING_FILE") ))%"
echo ""
echo "Files generated:"
echo "  - Alias suggestions: $ALIAS_SUGGESTIONS"
echo "  - Rename suggestions: $RENAME_SUGGESTIONS"  
echo "  - JSON fixes: $HIGH_CONFIDENCE_FIXES"
echo ""

if [[ $FIXES_FOUND -gt 0 ]]; then
    success "Found $FIXES_FOUND high-confidence fixes that could immediately reduce missing routines!"
else
    echo "No high-confidence automatic fixes found. Manual review may be needed."
fi

success "Analysis complete!"