#!/usr/bin/env bash
# Generate Negative Test Fixtures
# Creates corrupted, malformed, and edge-case files for testing error handling

set -euo pipefail

# Get APP_ROOT using cached value or compute once (2 levels up: __test/fixtures/generate-negative-fixtures.sh)
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test/fixtures"
NEGATIVE_DIR="$SCRIPT_DIR/negative-tests"

# Create negative tests directory
mkdir -p "$NEGATIVE_DIR"

echo "Generating negative test fixtures..."
echo "===================================="

#######################################
# CORRUPTED AUDIO FILES
#######################################

echo "Creating corrupted audio files..."
mkdir -p "$NEGATIVE_DIR/audio"

# 1. Empty audio file
touch "$NEGATIVE_DIR/audio/empty.mp3"
echo "  ✓ Created empty.mp3"

# 2. Invalid audio header
echo "NOT_AN_AUDIO_FILE" > "$NEGATIVE_DIR/audio/invalid_header.mp3"
echo "  ✓ Created invalid_header.mp3"

# 3. Truncated audio file (partial MP3 header)
printf "\xFF\xFB" > "$NEGATIVE_DIR/audio/truncated.mp3"
echo "  ✓ Created truncated.mp3"

# 4. Random binary data as audio
dd if=/dev/urandom of="$NEGATIVE_DIR/audio/random_binary.wav" bs=1024 count=10 2>/dev/null
echo "  ✓ Created random_binary.wav"

# 5. Text file with audio extension
cat > "$NEGATIVE_DIR/audio/text_as_audio.mp3" << EOF
This is actually a text file
pretending to be an audio file.
It should be rejected by audio processors.
EOF
echo "  ✓ Created text_as_audio.mp3"

#######################################
# CORRUPTED DOCUMENT FILES
#######################################

echo
echo "Creating corrupted document files..."
mkdir -p "$NEGATIVE_DIR/documents"

# 1. Empty PDF
touch "$NEGATIVE_DIR/documents/empty.pdf"
echo "  ✓ Created empty.pdf"

# 2. Invalid PDF header
echo "NOT_A_PDF_FILE" > "$NEGATIVE_DIR/documents/invalid_pdf.pdf"
echo "  ✓ Created invalid_pdf.pdf"

# 3. Partial PDF header
echo "%PDF-1.4" > "$NEGATIVE_DIR/documents/partial_pdf.pdf"
echo "  ✓ Created partial_pdf.pdf"

# 4. Corrupted Word document
echo "PK" > "$NEGATIVE_DIR/documents/corrupted.docx"  # Partial ZIP header
echo "  ✓ Created corrupted.docx"

# 5. Malformed JSON
cat > "$NEGATIVE_DIR/documents/malformed.json" << 'EOF'
{
  "valid_start": true,
  "but_then": "missing closing brace and quote
EOF
echo "  ✓ Created malformed.json"

# 6. Invalid XML
cat > "$NEGATIVE_DIR/documents/invalid.xml" << 'EOF'
<?xml version="1.0"?>
<root>
  <unclosed_tag>
  <another>value</wrong_closing>
</root>
EOF
echo "  ✓ Created invalid.xml"

# 7. Circular reference JSON (stress test)
cat > "$NEGATIVE_DIR/documents/circular_reference.json" << 'EOF'
{
  "a": {"ref": "#/a"},
  "b": {"ref": "#/b"},
  "recursive": {"nested": {"deeply": {"ref": "#/recursive"}}}
}
EOF
echo "  ✓ Created circular_reference.json"

# 8. Binary file with document extension
dd if=/dev/urandom of="$NEGATIVE_DIR/documents/binary.pdf" bs=1024 count=50 2>/dev/null
echo "  ✓ Created binary.pdf"

# 9. Huge but empty structure
python3 -c "
import json
data = {'level' + str(i): {} for i in range(10000)}
with open('$NEGATIVE_DIR/documents/huge_empty.json', 'w') as f:
    json.dump(data, f)
" 2>/dev/null || echo "{}" > "$NEGATIVE_DIR/documents/huge_empty.json"
echo "  ✓ Created huge_empty.json"

#######################################
# CORRUPTED IMAGE FILES
#######################################

echo
echo "Creating corrupted image files..."
mkdir -p "$NEGATIVE_DIR/images"

# 1. Empty image
touch "$NEGATIVE_DIR/images/empty.jpg"
echo "  ✓ Created empty.jpg"

# 2. Invalid JPEG header
echo "NOT_A_JPEG" > "$NEGATIVE_DIR/images/invalid_jpeg.jpg"
echo "  ✓ Created invalid_jpeg.jpg"

# 3. Partial PNG header
printf "\x89PNG" > "$NEGATIVE_DIR/images/partial_png.png"
echo "  ✓ Created partial_png.png"

# 4. Text file as image
cat > "$NEGATIVE_DIR/images/text_as_image.png" << EOF
This is text content
Not an actual image
Should fail image processing
EOF
echo "  ✓ Created text_as_image.png"

# 5. Truncated GIF
printf "GIF89a" > "$NEGATIVE_DIR/images/truncated.gif"
echo "  ✓ Created truncated.gif"

# 6. Random binary as image
dd if=/dev/urandom of="$NEGATIVE_DIR/images/random.jpg" bs=1024 count=5 2>/dev/null
echo "  ✓ Created random.jpg"

#######################################
# EDGE CASE FILES
#######################################

echo
echo "Creating edge case files..."
mkdir -p "$NEGATIVE_DIR/edge-cases"

# 1. Zero byte file
touch "$NEGATIVE_DIR/edge-cases/zero_bytes.txt"
echo "  ✓ Created zero_bytes.txt"

# 2. File with only whitespace
printf "   \n\t\n   \t" > "$NEGATIVE_DIR/edge-cases/only_whitespace.txt"
echo "  ✓ Created only_whitespace.txt"

# 3. File with null bytes
printf "Hello\x00World\x00Test" > "$NEGATIVE_DIR/edge-cases/null_bytes.txt"
echo "  ✓ Created null_bytes.txt"

# 4. File with special unicode characters
cat > "$NEGATIVE_DIR/edge-cases/unicode_stress.txt" << 'EOF'
零 🚀 𝓣𝓮𝓼𝓽 ñ ® ™ ∞ ⌘ 
‮Right-to-left override test‬
Z̴̡̡͖̖̣̹̰̤̈́̈́̈́a̸̧̛̛̰̣̦̟̤͇̐̓̉l̵̨̧̛̰̖̰̣̉̐̓g̸̨̛̣̰̤̣̐̈́̉̓o̵̧̡̰̤̖̐̈́̓ ̸̨̧̛̰̤̣̐̈́t̵̡̨̛̰̤̖̉̐̓ę̸̧̣̰̤̐̈́x̵̨̧̛̰̤̣̉̐̓t̸̡̨̛̰̤̖̐̈́̓
EOF
echo "  ✓ Created unicode_stress.txt"

# 5. File with very long single line
python3 -c "print('A' * 1000000)" > "$NEGATIVE_DIR/edge-cases/long_single_line.txt" 2>/dev/null || \
    perl -e 'print "A" x 1000000' > "$NEGATIVE_DIR/edge-cases/long_single_line.txt" 2>/dev/null || \
    printf "%1000000s" | tr ' ' 'A' > "$NEGATIVE_DIR/edge-cases/long_single_line.txt"
echo "  ✓ Created long_single_line.txt"

# 6. Filename with special characters (be careful with these)
touch "$NEGATIVE_DIR/edge-cases/file with spaces.txt" 2>/dev/null || true
touch "$NEGATIVE_DIR/edge-cases/file_with_émojis_🎉.txt" 2>/dev/null || true
echo "  ✓ Created files with special names"

# 7. Nested ZIP bomb structure (safe version - not actual zip bomb)
cat > "$NEGATIVE_DIR/edge-cases/nested_structure.json" << 'EOF'
{
  "level1": {
    "level2": {
      "level3": {
        "level4": {
          "level5": {
            "description": "Deeply nested but safe structure for testing depth limits"
          }
        }
      }
    }
  }
}
EOF
echo "  ✓ Created nested_structure.json"

# 8. File with mixed line endings
printf "Line1\r\nLine2\nLine3\rLine4" > "$NEGATIVE_DIR/edge-cases/mixed_line_endings.txt"
echo "  ✓ Created mixed_line_endings.txt"

#######################################
# STRESS TEST FILES
#######################################

echo
echo "Creating stress test files..."
mkdir -p "$NEGATIVE_DIR/stress"

# 1. Many small files in a directory
mkdir -p "$NEGATIVE_DIR/stress/many_files"
for i in {1..100}; do
    echo "File $i" > "$NEGATIVE_DIR/stress/many_files/file_$i.txt"
done
echo "  ✓ Created 100 small files"

# 2. File with many columns (CSV)
python3 -c "
import csv
with open('$NEGATIVE_DIR/stress/many_columns.csv', 'w', newline='') as f:
    writer = csv.writer(f)
    headers = [f'col_{i}' for i in range(1000)]
    writer.writerow(headers)
    writer.writerow(['data'] * 1000)
" 2>/dev/null || echo "col1,col2,col3" > "$NEGATIVE_DIR/stress/many_columns.csv"
echo "  ✓ Created many_columns.csv"

# 3. Rapidly changing file (simulation)
cat > "$NEGATIVE_DIR/stress/changing_content.txt" << EOF
This file simulates content that might change during processing.
Version: 1
Timestamp: $(date)
Note: In real tests, this could be updated by another process.
EOF
echo "  ✓ Created changing_content.txt"

#######################################
# CREATE METADATA FILE
#######################################

cat > "$NEGATIVE_DIR/README.md" << 'EOF'
# Negative Test Fixtures

This directory contains intentionally corrupted, malformed, and edge-case files
designed to test error handling and resilience of resource processors.

## Categories

### Audio
- Empty files
- Invalid headers
- Truncated data
- Non-audio content with audio extensions

### Documents
- Malformed JSON/XML
- Invalid PDF/Word formats
- Circular references
- Huge but empty structures

### Images
- Invalid image formats
- Truncated headers
- Non-image content with image extensions

### Edge Cases
- Zero-byte files
- Unicode stress tests
- Very long lines
- Special characters in filenames
- Mixed line endings
- Null bytes

### Stress Tests
- Many small files
- Large structures
- Rapidly changing content

## Usage

These files should be used to verify that services:
1. Handle errors gracefully without crashing
2. Return appropriate error codes
3. Don't consume excessive resources
4. Properly validate input before processing

## Safety Note

These files are designed to be challenging but safe. They do not contain:
- Actual malware
- Real zip bombs
- Genuinely dangerous payloads
- Files that could harm the system

They are purely for testing error handling capabilities.
EOF
echo "  ✓ Created README.md"

#######################################
# CREATE TEST VALIDATION SCRIPT
#######################################

cat > "$NEGATIVE_DIR/validate-negative-tests.sh" << 'EOF'
#!/usr/bin/env bash
# Validate that negative test files exist and have expected properties

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
NEGATIVE_DIR="${APP_ROOT}/__test/fixtures/negative-tests"

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

echo
echo "Validation complete!"
EOF
chmod +x "$NEGATIVE_DIR/validate-negative-tests.sh"
echo "  ✓ Created validate-negative-tests.sh"

echo
echo "===================================="
echo "Negative test fixtures generated successfully!"
echo "Location: $NEGATIVE_DIR"
echo
echo "These files can be used to test error handling in resource processors."
echo "Run $NEGATIVE_DIR/validate-negative-tests.sh to verify the files."
echo "===================================="