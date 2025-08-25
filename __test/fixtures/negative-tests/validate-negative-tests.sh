#!/usr/bin/env bash
# Validate that negative test files exist and have expected properties
# Now generates files on-demand using the generator script

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NEGATIVE_DIR="${APP_ROOT}/__test/fixtures/negative-tests"
GENERATOR_SCRIPT="${APP_ROOT}/__test/fixtures/generators/generate-stress-files.sh"

# Generate negative test files if they don't exist
if [[ ! -d "$NEGATIVE_DIR/images" ]] || [[ ! -d "$NEGATIVE_DIR/audio" ]] || \
   [[ ! -d "$NEGATIVE_DIR/documents" ]] || [[ ! -d "$NEGATIVE_DIR/edge-cases" ]] || \
   [[ ! -d "$NEGATIVE_DIR/stress" ]]; then
    echo "Generating negative test files on-demand..."
    "$GENERATOR_SCRIPT" all-negative
    echo
fi

echo "Validating negative test fixtures..."
echo "===================================="

# Check audio files
echo "Checking audio files..."
for file in empty.mp3 invalid_header.mp3 truncated.mp3 random_binary.wav text_as_audio.mp3; do
    if [[ -f "$NEGATIVE_DIR/audio/$file" ]]; then
        echo "  ✓ $file exists"
    else
        echo "  ✗ $file missing"
    fi
done

# Check document files
echo
echo "Checking document files..."
for file in empty.pdf invalid_pdf.pdf partial_pdf.pdf corrupted.docx malformed.json invalid.xml; do
    if [[ -f "$NEGATIVE_DIR/documents/$file" ]]; then
        echo "  ✓ $file exists"
    else
        echo "  ✗ $file missing"
    fi
done

# Check image files
echo
echo "Checking image files..."
for file in empty.jpg invalid_jpeg.jpg partial_png.png text_as_image.png truncated.gif random.jpg; do
    if [[ -f "$NEGATIVE_DIR/images/$file" ]]; then
        echo "  ✓ $file exists"
    else
        echo "  ✗ $file missing"
    fi
done

# Check edge cases
echo
echo "Checking edge case files..."
if [[ -f "$NEGATIVE_DIR/edge-cases/zero_bytes.txt" ]]; then
    size=$(stat -f%z "$NEGATIVE_DIR/edge-cases/zero_bytes.txt" 2>/dev/null || stat -c%s "$NEGATIVE_DIR/edge-cases/zero_bytes.txt" 2>/dev/null || echo "unknown")
    if [[ "$size" == "0" ]]; then
        echo "  ✓ zero_bytes.txt is 0 bytes"
    else
        echo "  ⚠ zero_bytes.txt is $size bytes (expected 0)"
    fi
fi

# Check stress test files
echo
echo "Checking stress test files..."
for file in changing_content.txt many_columns.csv; do
    if [[ -f "$NEGATIVE_DIR/stress/$file" ]]; then
        echo "  ✓ $file exists"
    else
        echo "  ✗ $file missing"
    fi
done

echo
echo "Validation complete!"
echo
echo "Note: Files are now generated on-demand by the generator script."
echo "To regenerate: $GENERATOR_SCRIPT all-negative"
echo "To clean up: $GENERATOR_SCRIPT clean"