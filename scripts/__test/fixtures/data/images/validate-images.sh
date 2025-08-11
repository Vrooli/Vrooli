#!/bin/bash
# Vrooli Image Fixtures Type-Specific Validator
# Validates image-specific requirements and tests with available resources

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source var.sh first to get proper directory variables
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true

# Source trash system for safe removal using var_ variables
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
FIXTURES_DIR="$SCRIPT_DIR"
METADATA_FILE="$FIXTURES_DIR/metadata.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_IMAGES=0
VALID_IMAGES=0
FAILED_IMAGES=0
TESTED_IMAGES=0

print_section() {
    echo -e "${YELLOW}--- $1 ---${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Check if required tools are available
check_image_tools() {
    local missing_tools=()
    
    if ! command -v identify >/dev/null 2>&1; then
        missing_tools+=("identify (ImageMagick)")
    fi
    
    if ! command -v file >/dev/null 2>&1; then
        missing_tools+=("file")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        print_warning "Some image validation tools are missing: ${missing_tools[*]}"
        print_info "Install ImageMagick for complete validation: sudo apt-get install imagemagick"
        return 1
    fi
    
    return 0
}

# Validate individual image file
validate_image_file() {
    local image_path="$1"
    local expected_format="$2"
    local expected_dimensions="$3"  # Array as string: "[width, height]"
    local expected_size="$4"
    
    TOTAL_IMAGES=$((TOTAL_IMAGES + 1))
    
    if [[ ! -f "$image_path" ]]; then
        print_error "Image not found: $image_path"
        FAILED_IMAGES=$((FAILED_IMAGES + 1))
        return 1
    fi
    
    local errors=0
    local filename=$(basename "$image_path")
    
    # Check file exists and is readable
    if [[ ! -r "$image_path" ]]; then
        print_error "$filename: Not readable"
        errors=$((errors + 1))
    fi
    
    # Check file size
    local actual_size=$(stat -c%s "$image_path" 2>/dev/null || echo "0")
    if [[ $actual_size -eq 0 ]]; then
        print_error "$filename: Empty file"
        errors=$((errors + 1))
    fi
    
    # Basic file type validation
    if command -v file >/dev/null 2>&1; then
        local file_type=$(file -b --mime-type "$image_path" 2>/dev/null)
        
        case "$expected_format" in
            "png")
                if [[ "$file_type" != "image/png" ]]; then
                    print_error "$filename: Expected PNG, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
            "jpg"|"jpeg")
                if [[ "$file_type" != "image/jpeg" ]]; then
                    print_error "$filename: Expected JPEG, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
            "gif")
                if [[ "$file_type" != "image/gif" ]]; then
                    print_error "$filename: Expected GIF, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
            "webp")
                if [[ "$file_type" != "image/webp" ]]; then
                    print_error "$filename: Expected WebP, got $file_type"
                    errors=$((errors + 1))
                fi
                ;;
            "heic")
                # HEIC detection can be tricky, just check it's not a common web format
                if [[ "$file_type" =~ ^image/(png|jpeg|gif|webp)$ ]]; then
                    print_error "$filename: Expected HEIC, got common web format $file_type"
                    errors=$((errors + 1))
                fi
                ;;
        esac
    fi
    
    # Advanced validation with ImageMagick
    if command -v identify >/dev/null 2>&1; then
        local identify_output=$(identify "$image_path" 2>/dev/null || echo "")
        
        if [[ -n "$identify_output" ]]; then
            # Extract dimensions
            local actual_dimensions=$(echo "$identify_output" | grep -o '[0-9]\+x[0-9]\+' | head -1)
            
            if [[ -n "$actual_dimensions" && "$expected_dimensions" != "[0, 0]" ]]; then
                # Convert "[width, height]" to "widthxheight"
                local expected_dim_clean=$(echo "$expected_dimensions" | sed 's/\[//g; s/\]//g; s/, /x/g')
                
                if [[ "$actual_dimensions" != "$expected_dim_clean" ]]; then
                    print_error "$filename: Dimension mismatch. Expected $expected_dim_clean, got $actual_dimensions"
                    errors=$((errors + 1))
                fi
            fi
            
            # Check if image is corrupted
            if ! identify -ping "$image_path" >/dev/null 2>&1; then
                print_error "$filename: Image appears corrupted"
                errors=$((errors + 1))
            fi
        else
            print_warning "$filename: Could not read image with ImageMagick"
        fi
    fi
    
    if [[ $errors -gt 0 ]]; then
        print_error "$filename: $errors validation errors"
        FAILED_IMAGES=$((FAILED_IMAGES + 1))
        return 1
    fi
    
    print_success "$filename: Format validation passed"
    VALID_IMAGES=$((VALID_IMAGES + 1))
    return 0
}

# Test OCR images with available OCR tools
test_ocr_capabilities() {
    print_section "Testing OCR Capabilities"
    
    # Check if tesseract is available
    if ! command -v tesseract >/dev/null 2>&1; then
        print_info "Tesseract not available - OCR testing skipped"
        return 0
    fi
    
    # Find OCR test images
    local ocr_images=($(find "$FIXTURES_DIR/ocr" -name "*.png" -o -name "*.jpg" 2>/dev/null | head -3))
    
    if [[ ${#ocr_images[@]} -eq 0 ]]; then
        print_info "No OCR test images found"
        return 0
    fi
    
    for image in "${ocr_images[@]}"; do
        local filename=$(basename "$image")
        print_info "Testing OCR on $filename"
        
        # Quick OCR test
        local ocr_output=$(tesseract "$image" stdout 2>/dev/null | head -c 100)
        
        if [[ -n "$ocr_output" ]]; then
            print_success "$filename: OCR extraction successful"
            TESTED_IMAGES=$((TESTED_IMAGES + 1))
        else
            print_warning "$filename: OCR extraction returned no text"
        fi
    done
}

# Test image processing with available tools
test_image_processing() {
    print_section "Testing Image Processing"
    
    if ! command -v convert >/dev/null 2>&1; then
        print_info "ImageMagick convert not available - processing tests skipped"
        return 0
    fi
    
    # Find a small test image
    local test_image=$(find "$FIXTURES_DIR" -name "*.png" -o -name "*.jpg" | head -1)
    
    if [[ -z "$test_image" ]]; then
        print_info "No test images found for processing test"
        return 0
    fi
    
    local filename=$(basename "$test_image")
    print_info "Testing image processing with $filename"
    
    # Test basic operations
    local temp_dir=$(mktemp -d)
    
    # Test resize
    if convert "$test_image" -resize 50x50 "$temp_dir/resized.png" 2>/dev/null; then
        print_success "Image resize test passed"
    else
        print_warning "Image resize test failed"
    fi
    
    # Test format conversion
    if convert "$test_image" "$temp_dir/converted.jpg" 2>/dev/null; then
        print_success "Format conversion test passed"
    else
        print_warning "Format conversion test failed"
    fi
    
    # Cleanup
    trash::safe_remove "$temp_dir" --test-cleanup
}

# Validate all images from metadata
validate_all_images() {
    print_section "Validating Image Files"
    
    if [[ ! -f "$METADATA_FILE" ]]; then
        print_error "metadata.yaml not found"
        return 1
    fi
    
    # Check if yq is available
    if ! command -v yq >/dev/null 2>&1; then
        print_error "yq required for metadata parsing"
        return 1
    fi
    
    # Extract all image paths using a simpler approach
    local paths=$(yq eval '.images[][] | select(has("path")) | .path' "$METADATA_FILE" 2>/dev/null)
    
    if [[ -z "$paths" ]]; then
        print_error "No image data found in metadata.yaml"
        return 1
    fi
    
    while IFS= read -r path; do
        if [[ -n "$path" && "$path" != "null" ]]; then
            # Get properties for this specific path
            local format=$(yq eval ".images[][] | select(.path == \"$path\") | .format" "$METADATA_FILE" 2>/dev/null)
            local dimensions=$(yq eval ".images[][] | select(.path == \"$path\") | .dimensions" "$METADATA_FILE" 2>/dev/null)
            local filesize=$(yq eval ".images[][] | select(.path == \"$path\") | .fileSize" "$METADATA_FILE" 2>/dev/null)
            
            local full_path="$FIXTURES_DIR/$path"
            validate_image_file "$full_path" "$format" "$dimensions" "$filesize"
        fi
    done <<< "$paths"
}

# Generate validation report
generate_report() {
    print_section "Image Validation Summary"
    
    echo "Total images validated: $TOTAL_IMAGES"
    echo "Valid images: $VALID_IMAGES"
    echo "Failed validations: $FAILED_IMAGES"
    echo "Images tested with tools: $TESTED_IMAGES"
    echo
    
    if [[ $FAILED_IMAGES -eq 0 ]]; then
        print_success "All image fixtures validated successfully!"
        echo -e "${GREEN}✅ Image formats verified${NC}"
        echo -e "${GREEN}✅ Dimensions validated${NC}"
        echo -e "${GREEN}✅ Files accessible and readable${NC}"
    else
        print_error "$FAILED_IMAGES image validation failures"
        echo -e "${RED}❌ Fix image issues before running tests${NC}"
        return 1
    fi
    
    # Tool availability report
    echo
    print_info "Image Processing Tool Availability:"
    if command -v identify >/dev/null 2>&1; then
        echo "  - ImageMagick identify: ✅ Available"
    else
        echo "  - ImageMagick identify: ❌ Missing"
    fi
    
    if command -v convert >/dev/null 2>&1; then
        echo "  - ImageMagick convert: ✅ Available"
    else
        echo "  - ImageMagick convert: ❌ Missing"
    fi
    
    if command -v tesseract >/dev/null 2>&1; then
        echo "  - Tesseract OCR: ✅ Available"
    else
        echo "  - Tesseract OCR: ❌ Missing"
    fi
}

main() {
    print_section "Image Fixtures Type-Specific Validation"
    
    check_image_tools || print_info "Continuing with limited validation capabilities"
    validate_all_images
    
    # Run optional tests if tools are available
    test_ocr_capabilities
    test_image_processing
    
    echo
    generate_report
}

# Show usage if requested
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
    echo "Vrooli Image Fixtures Type-Specific Validator"
    echo ""
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help    Show this help message"
    echo ""
    echo "This validator performs image-specific validation:"
    echo "  - File format verification (PNG, JPEG, GIF, WebP, HEIC)"
    echo "  - Dimension validation using ImageMagick"
    echo "  - Image corruption detection"
    echo "  - OCR capability testing (if Tesseract available)"
    echo "  - Basic image processing tests"
    echo ""
    echo "Required tools for full validation:"
    echo "  - ImageMagick (identify, convert)"
    echo "  - Tesseract (for OCR testing)"
    echo "  - file command"
    exit 0
fi

# Run validation if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi