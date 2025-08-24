#!/bin/bash
# ====================================================================
# Image Test Helper Functions
# ====================================================================
# Provides functions to work with image metadata in tests
#

# Get APP_ROOT using cached value or compute once (3 levels up: __test/fixtures/images/test_helpers.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
IMAGES_DIR="${APP_ROOT}/__test/fixtures/images"
METADATA_FILE="$IMAGES_DIR/images.yaml"

# Load metadata for a specific image
get_image_metadata() {
    local image_path="$1"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        echo "ERROR: Metadata file not found: $METADATA_FILE" >&2
        return 1
    fi
    
    # Use yq to query the metadata (assumes yq is installed)
    if command -v yq >/dev/null 2>&1; then
        yq eval ".images[][] | select(.path == \"$image_path\")" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get all images with a specific tag
get_images_by_tag() {
    local tag="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".images[][] | select(.tags[] == \"$tag\") | .path" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Get test suite images
get_test_suite_images() {
    local suite_name="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".testSuites.${suite_name}[]" "$METADATA_FILE"
    else
        echo "WARNING: yq not installed. Install with: pip install yq" >&2
        return 1
    fi
}

# Validate image dimensions
validate_image_dimensions() {
    local image_path="$1"
    local expected_width="$2"
    local expected_height="$3"
    
    local full_path="$IMAGES_DIR/$image_path"
    
    if [[ ! -f "$full_path" ]]; then
        echo "ERROR: Image not found: $full_path" >&2
        return 1
    fi
    
    # Get actual dimensions using identify (ImageMagick)
    if command -v identify >/dev/null 2>&1; then
        local dims
        dims=$(identify -format "%wx%h" "$full_path" 2>/dev/null)
        local actual_width="${dims%x*}"
        local actual_height="${dims#*x}"
        
        if [[ "$actual_width" == "$expected_width" && "$actual_height" == "$expected_height" ]]; then
            return 0
        else
            echo "Dimension mismatch: expected ${expected_width}x${expected_height}, got ${actual_width}x${actual_height}" >&2
            return 1
        fi
    else
        echo "WARNING: ImageMagick not installed. Cannot validate dimensions." >&2
        return 2
    fi
}

# Get expected OCR text for an image
get_expected_ocr_text() {
    local image_path="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval ".images[][] | select(.path == \"$image_path\") | .testData.expectedText" "$METADATA_FILE"
    else
        return 1
    fi
}

# Example usage in tests:
# 
# # In a BATS test:
# @test "thumbnail generation for tagged images" {
#     source "$FIXTURES_DIR/images/test_helpers.sh"
#     
#     # Get all images tagged for thumbnail testing
#     while IFS= read -r image_path; do
#         # Test thumbnail generation for each image
#         run generate_thumbnail "$IMAGES_DIR/$image_path"
#         assert_success
#     done < <(get_images_by_tag "thumbnail")
# }
#
# @test "OCR validation" {
#     source "$FIXTURES_DIR/images/test_helpers.sh"
#     
#     image_path="ocr/images/1_simple_text.png"
#     expected_text=$(get_expected_ocr_text "$image_path")
#     
#     run perform_ocr "$IMAGES_DIR/$image_path"
#     assert_success
#     assert_output --partial "$expected_text"
# }