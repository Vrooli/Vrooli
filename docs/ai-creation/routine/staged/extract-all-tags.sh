#!/bin/bash
# Script to extract all unique tags from routine files for database seeding

echo "=== Extracting All Tags from Routine Files ==="
echo ""

output_file="all-tags-for-import.json"
temp_file="/tmp/all_tags_temp.json"

# Initialize empty array
echo "[]" > "$temp_file"

echo "🔍 Scanning all routine files for tags..."

tag_count=0
file_count=0

for category in */; do
    # Skip non-category directories  
    if [[ "$category" == "main-routines/" ]]; then
        continue
    fi
    
    category_name="${category%/}"
    
    for routine_file in "${category}"*.json; do
        if [[ -f "$routine_file" && "$(basename "$routine_file")" != "_tags.json" ]]; then
            ((file_count++))
            
            # Extract tags from this file and add to collection
            if jq -e '.tags' "$routine_file" >/dev/null 2>&1; then
                # Get tags from this file
                file_tags=$(jq '.tags' "$routine_file" 2>/dev/null)
                
                if [[ "$file_tags" != "null" && "$file_tags" != "[]" ]]; then
                    # Add these tags to our collection, avoiding duplicates by ID
                    temp_file_new=$(mktemp)
                    jq --argjson new_tags "$file_tags" '
                        . as $existing |
                        $new_tags as $new |
                        ($existing + $new) | 
                        unique_by(.id)
                    ' "$temp_file" > "$temp_file_new"
                    mv "$temp_file_new" "$temp_file"
                    
                    local_count=$(echo "$file_tags" | jq 'length')
                    tag_count=$((tag_count + local_count))
                fi
            fi
        fi
    done
done

# Save final result
mv "$temp_file" "$output_file"

# Get final unique count
unique_count=$(jq 'length' "$output_file")

echo ""
echo "📊 Extraction Results:"
echo "======================"
echo "📁 Files scanned: $file_count"
echo "🏷️  Total tag instances: $tag_count"
echo "🔑 Unique tags: $unique_count"
echo "💾 Output file: $output_file"
echo ""

echo "🏷️  Unique tags found:"
jq -r '.[] | "  - \(.tag) (ID: \(.id))"' "$output_file" | sort

echo ""
echo "💡 Usage:"
echo "========"
echo "1. Import these tags to your database first:"
echo "   cat $output_file | jq '.[]' | while read tag; do"
echo "     # Insert tag into database using your import method"
echo "   done"
echo ""
echo "2. Then import routine files - all tag references will be valid"
echo ""

# Also create a simpler format for easier database import
simple_output="tags-for-db-import.json"
jq 'map({
    id: .id,
    tag: .tag,
    translations: .translations
})' "$output_file" > "$simple_output"

echo "📝 Also created simplified format: $simple_output"

# Create SQL-like format as well
sql_output="tags-for-db-import.sql"
echo "-- Tags for database import" > "$sql_output"
echo "-- Generated from routine files on $(date)" >> "$sql_output"
echo "" >> "$sql_output"

jq -r '.[] | 
    "INSERT INTO tags (id, tag) VALUES (\"\(.id)\", \"\(.tag)\");
     INSERT INTO tag_translations (id, tag_id, language, description) VALUES (\"\(.translations[0].id)\", \"\(.id)\", \"\(.translations[0].language)\", \"\(.translations[0].description)\");"
' "$output_file" >> "$sql_output"

echo "🗄️  Also created SQL format: $sql_output"

echo ""
echo "✅ Tag extraction complete!"