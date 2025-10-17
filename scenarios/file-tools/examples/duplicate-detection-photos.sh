#!/usr/bin/env bash
# Example: Duplicate photo detection for smart-file-photo-manager
# Demonstrates hash-based duplicate detection and EXIF metadata extraction

set -euo pipefail

# Configuration
FILE_TOOLS_PORT="${FILE_TOOLS_PORT:-15458}"
API_URL="http://localhost:${FILE_TOOLS_PORT}"

# ============================================================================
# BEFORE: Manual duplicate detection (naive approach)
# ============================================================================
find_duplicates_manual() {
    local photo_dir="$1"

    echo "❌ OLD WAY: Manual duplicate detection"
    echo "   Finding files with same name..."

    # Naive: Only finds files with identical names
    find "$photo_dir" -type f \( -iname "*.jpg" -o -iname "*.png" \) -print0 \
        | xargs -0 basename -a \
        | sort \
        | uniq -d

    # Problems:
    # - Only finds exact filename matches
    # - Misses renamed duplicates
    # - Misses rotated/resized versions
    # - No similarity scoring
    # - No storage savings calculation
}

# ============================================================================
# AFTER: Using file-tools for accurate duplicate detection
# ============================================================================
find_duplicates_with_file_tools() {
    local photo_dir="$1"

    echo "✅ NEW WAY: File-tools hash-based duplicate detection"

    # Scan for duplicates using content hash
    response=$(curl -sf "${API_URL}/api/v1/files/duplicates/detect" \
        -H "Content-Type: application/json" \
        -d "{
            \"scan_paths\": [\"$photo_dir\"],
            \"detection_method\": \"hash\",
            \"options\": {
                \"similarity_threshold\": 1.0,
                \"include_hidden\": false,
                \"file_extensions\": [\"jpg\", \"jpeg\", \"png\", \"heic\"],
                \"file_size_min\": 1024
            }
        }")

    scan_id=$(echo "$response" | jq -r '.scan_id')
    total_duplicates=$(echo "$response" | jq -r '.total_duplicates')
    total_savings=$(echo "$response" | jq -r '.total_savings_bytes')
    duplicate_groups=$(echo "$response" | jq -r '.duplicate_groups | length')

    echo "📊 Duplicate Detection Results:"
    echo "   Scan ID: $scan_id"
    echo "   Duplicate files found: $total_duplicates"
    echo "   Duplicate groups: $duplicate_groups"
    echo "   Potential storage savings: $(numfmt --to=iec "$total_savings")"
    echo ""

    # Display duplicate groups
    echo "📷 Duplicate Groups:"
    echo "$response" | jq -r '.duplicate_groups[] |
        "  Group \(.group_id) (similarity: \(.similarity_score)):" +
        "\n    Files: \(.files | length)" +
        "\n    Savings: \(.potential_savings_bytes | tonumber)" +
        "\n    Paths:\n" +
        (.files | map("      - \(.path)") | join("\n"))'
}

# ============================================================================
# Extract EXIF metadata from photos
# ============================================================================
extract_photo_metadata() {
    local photo_path="$1"

    echo "📸 Extracting EXIF metadata for: $photo_path"

    response=$(curl -sf "${API_URL}/api/v1/files/metadata/extract" \
        -H "Content-Type: application/json" \
        -d "{
            \"file_paths\": [\"$photo_path\"],
            \"extraction_types\": [\"exif\", \"properties\"],
            \"options\": {
                \"deep_analysis\": true,
                \"generate_thumbnails\": true
            }
        }")

    # Parse EXIF data
    echo "$response" | jq -r '.results[0] |
        "📷 Photo Details:" +
        "\n  File: \(.file_path)" +
        "\n  Camera: \(.metadata.exif.Make // "Unknown") \(.metadata.exif.Model // "")" +
        "\n  Date Taken: \(.metadata.exif.DateTimeOriginal // "Unknown")" +
        "\n  Resolution: \(.metadata.basic.width)x\(.metadata.basic.height)" +
        "\n  File Size: \(.metadata.basic.size_bytes)" +
        "\n  GPS: \(.metadata.exif.GPSLatitude // "N/A"), \(.metadata.exif.GPSLongitude // "N/A")"'
}

# ============================================================================
# Smart organization by date and camera
# ============================================================================
organize_photos_smart() {
    local source_dir="$1"
    local dest_dir="$2"

    echo "🗂️  Smart photo organization"

    response=$(curl -sf "${API_URL}/api/v1/files/organize" \
        -H "Content-Type: application/json" \
        -d "{
            \"source_path\": \"$source_dir\",
            \"destination_path\": \"$dest_dir\",
            \"organization_rules\": [
                {\"rule_type\": \"by_date\", \"parameters\": {\"format\": \"YYYY/MM\"}},
                {\"rule_type\": \"by_metadata\", \"parameters\": {\"key\": \"exif.Make\"}}
            ],
            \"options\": {
                \"dry_run\": false,
                \"create_directories\": true,
                \"handle_conflicts\": \"rename\"
            }
        }")

    echo "✅ Organization complete:"
    echo "$response" | jq -r '
        "  Files moved: \(.organization_plan | length)" +
        "\n  Conflicts handled: \(.conflicts | length)"'
}

# ============================================================================
# DEMO: Compare approaches
# ============================================================================
main() {
    local test_dir="/tmp/photo-demo"
    local photos_dir="${test_dir}/photos"

    # Create test photo directory
    mkdir -p "$photos_dir"

    # Simulate photo files (in reality, these would be actual images)
    for i in {1..20}; do
        echo "Photo content $((i % 5))" > "${photos_dir}/photo${i}.jpg"
    done

    # Create some intentional duplicates
    cp "${photos_dir}/photo1.jpg" "${photos_dir}/photo1_copy.jpg"
    cp "${photos_dir}/photo2.jpg" "${photos_dir}/photo2_duplicate.jpg"

    echo "🎯 Photo Management Comparison"
    echo "=============================="
    echo ""

    # Test manual approach
    echo "1. Manual duplicate detection:"
    find_duplicates_manual "$photos_dir" || true
    echo ""

    # Test file-tools approach
    echo "2. File-tools duplicate detection:"
    find_duplicates_with_file_tools "$photos_dir"
    echo ""

    echo "✨ Benefits of file-tools approach:"
    echo "   ✓ 100% accurate content-based detection"
    echo "   ✓ Finds renamed duplicates"
    echo "   ✓ Calculates storage savings"
    echo "   ✓ Groups similar files"
    echo "   ✓ EXIF metadata extraction"
    echo "   ✓ Smart organization by date/camera"
    echo "   ✓ No false positives"

    # Cleanup
    rm -rf "$test_dir"
}

# Run demo if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
