#!/bin/bash
# Script to validate that all routine tags exist in the tag reference files

echo "=== Vrooli Routine Tag Validation ==="
echo ""

validation_errors=0
validation_warnings=0
total_routines=0
total_tags_checked=0

# Function to check if a tag exists in the global reference
check_tag_in_global() {
    local tag_id="$1"
    local tag_string="$2"
    
    # Check if tag exists in global reference by ID and string
    local found=$(jq -r --arg id "$tag_id" --arg tag "$tag_string" '
        (.categories[].tags[], .["cross-cutting-tags"][]) | 
        select(.id == $id and .tag == $tag) | 
        .id
    ' tags-reference.json 2>/dev/null)
    
    if [[ -n "$found" ]]; then
        return 0  # Found
    else
        return 1  # Not found
    fi
}

# Function to check if a tag exists in category reference
check_tag_in_category() {
    local category="$1"
    local tag_id="$2"
    local tag_string="$3"
    
    if [[ ! -f "${category}/_tags.json" ]]; then
        return 1  # Category file doesn't exist
    fi
    
    # Check if tag exists in category reference
    local found=$(jq -r --arg id "$tag_id" --arg tag "$tag_string" '
        (.availableTags[], (.crossCuttingTags[] as $ct | 
            if $ct == $tag then 
                {id: "cross-cutting", tag: $ct} 
            else 
                empty 
            end)) | 
        select(.id == $id and .tag == $tag) | 
        .id
    ' "${category}/_tags.json" 2>/dev/null)
    
    if [[ -n "$found" ]]; then
        return 0  # Found
    else
        return 1  # Not found
    fi
}

# Function to validate a single routine file
validate_routine() {
    local routine_file="$1"
    local category="$2"
    local routine_name=$(basename "$routine_file")
    
    echo "🔍 Validating: $category/$routine_name"
    
    # Check if file is valid JSON
    if ! jq empty "$routine_file" 2>/dev/null; then
        echo "  ❌ INVALID JSON: $routine_file"
        ((validation_errors++))
        return
    fi
    
    # Check if tags field exists
    local has_tags=$(jq 'has("tags")' "$routine_file" 2>/dev/null)
    if [[ "$has_tags" != "true" ]]; then
        echo "  ❌ MISSING TAGS FIELD: $routine_file"
        ((validation_errors++))
        return
    fi
    
    # Get tags array
    local tags_json=$(jq '.tags' "$routine_file" 2>/dev/null)
    if [[ "$tags_json" == "null" || "$tags_json" == "[]" ]]; then
        echo "  ⚠️  EMPTY TAGS: $routine_file"
        ((validation_warnings++))
        return
    fi
    
    # Validate each tag
    local tag_count=$(jq '.tags | length' "$routine_file" 2>/dev/null)
    local valid_tags=0
    local invalid_tags=0
    
    for ((i=0; i<tag_count; i++)); do
        local tag_id=$(jq -r ".tags[$i].id" "$routine_file" 2>/dev/null)
        local tag_string=$(jq -r ".tags[$i].tag" "$routine_file" 2>/dev/null)
        local tag_has_translations=$(jq ".tags[$i] | has(\"translations\")" "$routine_file" 2>/dev/null)
        
        ((total_tags_checked++))
        
        # Check tag structure
        if [[ -z "$tag_id" || "$tag_id" == "null" ]]; then
            echo "  ❌ TAG MISSING ID: $tag_string in $routine_file"
            ((invalid_tags++))
            continue
        fi
        
        if [[ -z "$tag_string" || "$tag_string" == "null" ]]; then
            echo "  ❌ TAG MISSING STRING: ID $tag_id in $routine_file"
            ((invalid_tags++))
            continue
        fi
        
        if [[ "$tag_has_translations" != "true" ]]; then
            echo "  ❌ TAG MISSING TRANSLATIONS: $tag_string in $routine_file"
            ((invalid_tags++))
            continue
        fi
        
        # Check if tag exists in global reference
        if check_tag_in_global "$tag_id" "$tag_string"; then
            ((valid_tags++))
            echo "  ✅ VALID TAG: $tag_string (ID: $tag_id)"
        else
            echo "  ❌ INVALID TAG: $tag_string (ID: $tag_id) - NOT FOUND in global reference"
            ((invalid_tags++))
        fi
        
        # Also check if tag is appropriate for this category
        if ! check_tag_in_category "$category" "$tag_id" "$tag_string"; then
            echo "  ⚠️  TAG NOT IN CATEGORY: $tag_string - not listed in $category/_tags.json (may still be valid if cross-cutting)"
            ((validation_warnings++))
        fi
    done
    
    if [[ $invalid_tags -gt 0 ]]; then
        ((validation_errors++))
        echo "  📊 Summary: $valid_tags valid, $invalid_tags invalid tags"
    else
        echo "  📊 Summary: $valid_tags valid tags"
    fi
    
    echo ""
}

# Main validation loop
echo "📋 Starting validation of all routine files..."
echo ""

for category in */; do
    # Skip non-category directories
    if [[ "$category" == "main-routines/" ]]; then
        continue
    fi
    
    category_name="${category%/}"
    
    # Check if category has _tags.json
    if [[ ! -f "${category}_tags.json" ]]; then
        echo "⚠️  WARNING: Category $category_name missing _tags.json file"
        ((validation_warnings++))
        continue
    fi
    
    # Validate all JSON files in category (except _tags.json)
    for routine_file in "${category}"*.json; do
        if [[ -f "$routine_file" && "$(basename "$routine_file")" != "_tags.json" ]]; then
            validate_routine "$routine_file" "$category_name"
            ((total_routines++))
        fi
    done
done

# Final summary
echo "=========================================="
echo "🎯 VALIDATION SUMMARY"
echo "=========================================="
echo "📊 Total routines validated: $total_routines"
echo "🏷️  Total tags checked: $total_tags_checked"
echo "❌ Validation errors: $validation_errors"
echo "⚠️  Validation warnings: $validation_warnings"
echo ""

if [[ $validation_errors -eq 0 ]]; then
    echo "✅ SUCCESS: All routine tags are valid!"
    echo "🚀 Ready for database import - all tags exist in reference files"
else
    echo "💥 FAILURE: $validation_errors routine(s) have invalid tags"
    echo "🔧 Fix these issues before importing to database"
fi

if [[ $validation_warnings -gt 0 ]]; then
    echo "📝 NOTE: $validation_warnings warning(s) found - review for potential improvements"
fi

echo ""
echo "💡 Next steps:"
if [[ $validation_errors -eq 0 ]]; then
    echo "1. Import tags from tags-reference.json to database first"
    echo "2. Then import routine files - all tag references will be valid"
else
    echo "1. Fix the validation errors above"
    echo "2. Re-run this script to verify fixes"
    echo "3. Then proceed with database import"
fi

# Exit with error code if validation failed
exit $validation_errors