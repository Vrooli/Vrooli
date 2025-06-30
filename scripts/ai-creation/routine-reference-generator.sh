#!/bin/bash
#---------------------------------------------------------------------------------------------------
# Routine Reference Generator
# 
# This script extracts ID, name, and type from all staged routine JSON files to create a searchable
# JSON reference for use when building routines that reference existing ones as subroutines.
#
# Features:
# - Excludes tag files (_tags.json, tags-reference.json, etc.)
# - Outputs structured JSON with routines grouped by category and type
# - Includes metadata and usage examples for jq queries
# - Should be run after adding new routine files to staged/
#
# Usage: ./scripts/ai-creation/routine-reference-generator.sh [output_file]
#
# Examples:
#   ./scripts/ai-creation/routine-reference-generator.sh
#   ./scripts/ai-creation/routine-reference-generator.sh docs/ai-creation/routine/my-reference.json
#---------------------------------------------------------------------------------------------------

# Get the script's directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Default output file
OUTPUT_FILE="${1:-$PROJECT_ROOT/docs/ai-creation/routine/routine-reference.json}"

# Base directory for staged routines
STAGED_DIR="$PROJECT_ROOT/docs/ai-creation/routine/staged"

# Ensure jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is not installed. Please install it first."
    exit 1
fi

# Initialize JSON structure
cat > "$OUTPUT_FILE" << EOF
{
  "metadata": {
    "generatedOn": "$(date -Iseconds)",
    "totalRoutines": 0,
    "version": "1.0"
  },
  "routines": [],
  "byCategory": {},
  "byType": {}
}
EOF

# Collect all routine data
routines_json="[]"
total_count=0

# Process each category directory
for category_dir in "$STAGED_DIR"/*; do
    if [ -d "$category_dir" ]; then
        category_name=$(basename "$category_dir")
        
        # Process each JSON file in the category (exclude tag files and non-routine files)
        for json_file in "$category_dir"/*.json; do
            if [ -f "$json_file" ] && [[ ! "$(basename "$json_file")" =~ ^(_tags|tags-|all-tags) ]]; then
                # Extract routine data
                id=$(jq -r '.id // "null"' "$json_file" 2>/dev/null)
                name=$(jq -r '.versions[0].translations[0].name // "null"' "$json_file" 2>/dev/null)
                description=$(jq -r '.versions[0].translations[0].description // ""' "$json_file" 2>/dev/null)
                publicId=$(jq -r '.publicId // ""' "$json_file" 2>/dev/null)
                
                # Determine routine type
                if jq -e '.versions[0].resourceSubType' "$json_file" &>/dev/null; then
                    type=$(jq -r '.versions[0].resourceSubType' "$json_file")
                else
                    # For multi-step routines, check if it has subroutines
                    if jq -e '.versions[0].routineVersionSupbroutines' "$json_file" &>/dev/null; then
                        type="RoutineMultiStep"
                    else
                        type="Unknown"
                    fi
                fi
                
                # Skip entries with null ID (shouldn't happen after filtering, but safety check)
                if [ "$id" != "null" ] && [ "$name" != "null" ]; then
                    # Create routine object and add to array
                    routine_obj=$(jq -n \
                        --arg id "$id" \
                        --arg name "$name" \
                        --arg type "$type" \
                        --arg category "$category_name" \
                        --arg description "$description" \
                        --arg publicId "$publicId" \
                        --arg file "$(basename "$json_file")" \
                        '{
                            id: $id,
                            name: $name,
                            type: $type,
                            category: $category,
                            description: $description,
                            publicId: $publicId,
                            file: $file
                        }')
                    
                    routines_json=$(echo "$routines_json" | jq ". += [$routine_obj]")
                    ((total_count++))
                fi
            fi
        done
    fi
done

# Create grouped data structures
by_category_json=$(echo "$routines_json" | jq 'group_by(.category) | map({key: .[0].category, value: .}) | from_entries')
by_type_json=$(echo "$routines_json" | jq 'group_by(.type) | map({key: .[0].type, value: .}) | from_entries')

# Build final JSON structure
final_json=$(jq -n \
    --arg generated_on "$(date -Iseconds)" \
    --arg total_count "$total_count" \
    --argjson routines "$routines_json" \
    --argjson by_category "$by_category_json" \
    --argjson by_type "$by_type_json" \
    '{
        metadata: {
            generatedOn: $generated_on,
            totalRoutines: ($total_count | tonumber),
            version: "1.0",
            usage: {
                searchById: "jq \".routines[] | select(.id == \\\"YOUR_ID\\\")\"",
                searchByName: "jq \".routines[] | select(.name | test(\\\"PATTERN\\\"; \\\"i\\\"))\"",
                getByCategory: "jq \".byCategory.CATEGORY_NAME.routines\"",
                getByType: "jq \".byType.TYPE_NAME.routines\"",
                listCategories: "jq \".byCategory | keys\"",
                listTypes: "jq \".byType | keys\""
            }
        },
        routines: $routines,
        byCategory: $by_category,
        byType: $by_type
    }')

# Write final JSON to file
echo "$final_json" > "$OUTPUT_FILE"

echo "Routine reference JSON generated at: $OUTPUT_FILE"
echo "Total routines processed: $total_count"
echo ""
echo "Usage examples:"
echo "  # List all routine types"
echo "  jq '.byType | keys' $OUTPUT_FILE"
echo ""
echo "  # Find routine by ID"
echo "  jq '.routines[] | select(.id == \"7829564732190847634\")' $OUTPUT_FILE"
echo ""
echo "  # Get all multi-step routines"
echo "  jq '.byType.RoutineMultiStep.routines' $OUTPUT_FILE"