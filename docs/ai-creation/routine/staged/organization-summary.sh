#!/bin/bash
# Script to show the current organization status

echo "=== Vrooli Routine Organization Summary ==="
echo ""

echo "📁 Category Distribution:"
echo "========================"

total_routines=0
for category in */; do
    if [[ "$category" == "main-routines/" ]]; then
        continue
    fi
    
    category_name="${category%/}"
    json_count=$(find "$category" -name "*.json" -not -name "_tags.json" | wc -l)
    total_routines=$((total_routines + json_count))
    
    if [[ $json_count -gt 0 ]]; then
        echo "  $category_name: $json_count routines"
    fi
done

echo ""
echo "📊 Total organized routines: $total_routines"
echo ""

echo "🏷️  Tag Coverage Analysis:"
echo "========================="

routines_with_tags=0
routines_without_tags=0

for category in */; do
    if [[ "$category" == "main-routines/" ]]; then
        continue
    fi
    
    for routine in "$category"*.json; do
        if [[ -f "$routine" && "$routine" != *"_tags.json" ]]; then
            tag_count=$(jq '.tags | length' "$routine" 2>/dev/null || echo "0")
            if [[ $tag_count -gt 0 ]]; then
                routines_with_tags=$((routines_with_tags + 1))
            else
                routines_without_tags=$((routines_without_tags + 1))
                echo "  ⚠️  Missing tags: $(basename "$routine")"
            fi
        fi
    done
done

echo "  ✅ Routines with tags: $routines_with_tags"
echo "  ❌ Routines without tags: $routines_without_tags"
echo ""

echo "📋 Category Setup Status:"
echo "========================"

for category in */; do
    if [[ "$category" == "main-routines/" ]]; then
        continue
    fi
    
    category_name="${category%/}"
    
    if [[ -f "${category}_tags.json" ]]; then
        tag_count=$(jq '.availableTags | length' "${category}_tags.json" 2>/dev/null || echo "0")
        echo "  ✅ $category_name: _tags.json ($tag_count tags)"
    else
        echo "  ❌ $category_name: missing _tags.json"
    fi
done

echo ""
echo "🔧 Next Steps:"
echo "============="
echo "1. Review routines without tags and add appropriate ones"
echo "2. Test import process with a few sample routines"
echo "3. Update generation scripts to use new category structure"
echo "4. Consider creating more routines for empty categories"
echo ""

echo "💡 Usage:"
echo "========"
echo "  Import all: vrooli routine import-dir ."
echo "  Import category: vrooli routine import-dir ./productivity/"
echo "  Validate: vrooli routine validate ./category/routine.json"