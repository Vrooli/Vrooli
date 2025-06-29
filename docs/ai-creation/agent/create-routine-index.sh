#!/bin/bash

# Create routine index for agent-routine connection validation
# This script builds a routine label index from the routine directory

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROUTINE_DIR="$SCRIPT_DIR/../routine"
OUTPUT_FILE="$SCRIPT_DIR/cache/routine-index.json"

echo "Creating routine index from $ROUTINE_DIR"

# Create output directory if it doesn't exist
mkdir -p "$(dirname "$OUTPUT_FILE")"

# Start building the index
cat > "$OUTPUT_FILE" << 'EOF'
{
  "lastUpdated": "",
  "totalRoutines": 0,
  "availableNames": [],
  "routinesByName": {},
  "notes": "This file maps routine names to routine metadata for agent validation"
}
EOF

# Update timestamp
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
jq --arg timestamp "$TIMESTAMP" '.lastUpdated = $timestamp' "$OUTPUT_FILE" > "${OUTPUT_FILE}.tmp" && mv "${OUTPUT_FILE}.tmp" "$OUTPUT_FILE"

# Scan for routine files and extract metadata
TOTAL_ROUTINES=0
NAMES_ARRAY="[]"
ROUTINES_BY_NAME="{}"

# Check if routine reference exists
ROUTINE_REF="$ROUTINE_DIR/routine-reference.json"
if [[ -f "$ROUTINE_REF" ]]; then
    echo "Using existing routine reference: $ROUTINE_REF"
    
    # Extract routine labels and metadata
    ROUTINE_DATA=$(jq -r '.routines[] | {name: .name, id: .id, type: .type, description: .translations[0].description}' "$ROUTINE_REF" 2>/dev/null || echo "{}")
    
    if [[ "$ROUTINE_DATA" != "{}" ]]; then
        # Build names array
        NAMES_ARRAY=$(jq -r '[.routines[].name]' "$ROUTINE_REF")
        
        # Build routines by name mapping
        ROUTINES_BY_NAME=$(jq -r '.routines | map({(.name): {id: .id, type: .type, description: .translations[0].description}}) | add' "$ROUTINE_REF")
        
        # Get total count
        TOTAL_ROUTINES=$(jq -r '.routines | length' "$ROUTINE_REF")
        
        echo "Found $TOTAL_ROUTINES routines in reference file"
    else
        echo "Warning: Routine reference exists but contains no routines"
    fi
else
    echo "Warning: Routine reference not found at $ROUTINE_REF"
    echo "Scanning staged routine files directly..."
    
    # Fallback: scan staged files directly
    STAGED_DIR="$ROUTINE_DIR/staged"
    if [[ -d "$STAGED_DIR" ]]; then
        ROUTINE_FILES=($(find "$STAGED_DIR" -name "*.json" -type f 2>/dev/null || true))
        
        if [[ ${#ROUTINE_FILES[@]} -gt 0 ]]; then
            TOTAL_ROUTINES=${#ROUTINE_FILES[@]}
            
            # Extract names from files (use filename as name)
            NAMES=()
            for file in "${ROUTINE_FILES[@]}"; do
                BASENAME=$(basename "$file" .json)
                NAMES+=("\"$BASENAME\"")
            done
            
            NAMES_ARRAY="[$(IFS=,; echo "${NAMES[*]}")]"
            
            echo "Found $TOTAL_ROUTINES routine files in staged directory"
        fi
    fi
fi

# Update the index file
jq --argjson total "$TOTAL_ROUTINES" \
   --argjson names "$NAMES_ARRAY" \
   --argjson routines "$ROUTINES_BY_NAME" \
   '.totalRoutines = $total | .availableNames = $names | .routinesByName = $routines' \
   "$OUTPUT_FILE" > "${OUTPUT_FILE}.tmp" && mv "${OUTPUT_FILE}.tmp" "$OUTPUT_FILE"

echo "Routine index created: $OUTPUT_FILE"
echo "Total routines indexed: $TOTAL_ROUTINES"

if [[ $TOTAL_ROUTINES -gt 0 ]]; then
    echo "Available routine names:"
    jq -r '.availableNames[]' "$OUTPUT_FILE" | head -10
    if [[ $TOTAL_ROUTINES -gt 10 ]]; then
        echo "... and $(( TOTAL_ROUTINES - 10 )) more"
    fi
else
    echo "Warning: No routines found. Agents will not be able to reference any routines."
    echo "Please ensure routine system is set up and routine-reference.json exists."
fi