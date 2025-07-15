#!/bin/bash

# Agent Routine Dependency Analysis Generator
# This script generates analysis files comparing agent routine needs vs available routines

set -euo pipefail

# Default settings
UPDATE_REFERENCE=true

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-update)
            UPDATE_REFERENCE=false
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "OPTIONS:"
            echo "  --no-update    Skip updating routine reference (use existing file)"
            echo "  --help         Show this help message"
            echo ""
            echo "This script automatically updates the routine reference from staged routines"
            echo "and then generates dependency analysis files."
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_DIR="$SCRIPT_DIR"
ROUTINE_DIR="$SCRIPT_DIR/../routine"
ROUTINE_REFERENCE="$AGENT_DIR/routine-reference.json"

# Output files
AGENT_REFS_FILE="$AGENT_DIR/agent_routine_references.txt"
AVAILABLE_ROUTINES_FILE="$AGENT_DIR/available_routines.txt"
MISSING_ROUTINES_FILE="$AGENT_DIR/missing_routines.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $*" >&2
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

# Update routine reference first (if requested)
if [[ "$UPDATE_REFERENCE" == "true" ]]; then
    log "Updating routine reference from latest staged routines..."

    if [[ -f "$ROUTINE_DIR/generate-routine-reference.sh" ]]; then
        cd "$ROUTINE_DIR"
        # Capture the output to see if there are actual changes
        UPDATE_OUTPUT=$(./generate-routine-reference.sh 2>&1)
        UPDATE_STATUS=$?
        
        if [[ $UPDATE_STATUS -eq 0 ]]; then
            # Check if routine count increased
            if echo "$UPDATE_OUTPUT" | grep -q "Generated routine reference with"; then
                ROUTINE_COUNT=$(echo "$UPDATE_OUTPUT" | grep "Generated routine reference with" | sed 's/.*with \([0-9]*\) routines.*/\1/')
                success "Routine reference updated successfully ($ROUTINE_COUNT routines)"
            else
                success "Routine reference update completed"
            fi
        else
            warn "Failed to update routine reference, using existing file"
            log "Update error: $UPDATE_OUTPUT"
        fi
        cd "$AGENT_DIR"
    else
        warn "Routine reference generator not found, using existing file"
    fi
else
    log "Skipping routine reference update (using existing file)"
fi

# Check if routine reference exists
[[ -f "$ROUTINE_REFERENCE" ]] || error "Routine reference not found: $ROUTINE_REFERENCE"

log "Starting agent routine dependency analysis..."

# 1. Extract routine references from all agent JSON files
log "Extracting routine references from agent files..."

# Create temp file for agent routine references
TEMP_AGENT_REFS=$(mktemp)

# Find all agent JSON files and extract routine labels from behaviors
find "$AGENT_DIR/staged" -name "*.json" -type f | while read -r agent_file; do
    # Extract routine labels from the "label" field in behaviors
    grep -o '"label"[[:space:]]*:[[:space:]]*"[^"]*"' "$agent_file" 2>/dev/null | \
    sed 's/.*"label"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' | \
    while read -r label; do
        # Convert kebab-case to Title Case for consistency
        echo "$label" | sed 's/-/ /g' | sed 's/\b\w/\U&/g'
    done
done | sort -u > "$TEMP_AGENT_REFS"

# Count and save agent routine references
AGENT_REF_COUNT=$(wc -l < "$TEMP_AGENT_REFS")
cp "$TEMP_AGENT_REFS" "$AGENT_REFS_FILE"

log "Found $AGENT_REF_COUNT unique routine references in agent files"

# 2. Extract available routine names from routine reference
log "Extracting available routine names from routine reference..."

# Create temp file for available routines
TEMP_AVAILABLE=$(mktemp)

# Extract routine names from the routine reference JSON
grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' "$ROUTINE_REFERENCE" | \
sed 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' | \
sort -u > "$TEMP_AVAILABLE"

# Count and save available routines
AVAILABLE_COUNT=$(wc -l < "$TEMP_AVAILABLE")
cp "$TEMP_AVAILABLE" "$AVAILABLE_ROUTINES_FILE"

log "Found $AVAILABLE_COUNT unique routine names in routine reference"

# 3. Find missing routines (needed by agents but not available)
log "Computing missing routines..."

# Create temp file for missing routines
TEMP_MISSING=$(mktemp)

# Find routines that are in agent references but not in available routines
comm -23 "$TEMP_AGENT_REFS" "$TEMP_AVAILABLE" > "$TEMP_MISSING"

# Count and save missing routines
MISSING_COUNT=$(wc -l < "$TEMP_MISSING")
cp "$TEMP_MISSING" "$MISSING_ROUTINES_FILE"

# 4. Generate summary report
log "Generating summary report..."

echo ""
echo "============================================="
echo "AGENT ROUTINE DEPENDENCY ANALYSIS SUMMARY"
echo "============================================="
echo ""
echo "Total routine references needed by agents: $AGENT_REF_COUNT"
echo "Total routines available in reference:     $AVAILABLE_COUNT"  
echo "Missing routines (needed but not available): $MISSING_COUNT"
echo ""

if [[ $MISSING_COUNT -eq 0 ]]; then
    success "All agent routine dependencies are satisfied! ✅"
else
    SATISFACTION_RATE=$(( (AGENT_REF_COUNT - MISSING_COUNT) * 100 / AGENT_REF_COUNT ))
    echo "Dependency satisfaction rate: ${SATISFACTION_RATE}%"
    echo ""
    echo "Missing routines (first 10):"
    head -10 "$MISSING_ROUTINES_FILE" | sed 's/^/  - /'
    
    if [[ $MISSING_COUNT -gt 10 ]]; then
        echo "  ... and $(( MISSING_COUNT - 10 )) more"
    fi
fi

echo ""
echo "Files generated:"
echo "  - Agent routine references: $AGENT_REFS_FILE"
echo "  - Available routines:       $AVAILABLE_ROUTINES_FILE"
echo "  - Missing routines:         $MISSING_ROUTINES_FILE"

# Clean up temp files
rm -f "$TEMP_AGENT_REFS" "$TEMP_AVAILABLE" "$TEMP_MISSING"

success "Analysis complete!"