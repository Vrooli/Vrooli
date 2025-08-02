# Vrooli Test Fixtures

This directory contains test fixtures (sample files and data) used throughout Vrooli's resource testing ecosystem. These fixtures enable comprehensive testing of resource integrations, AI capabilities, and data processing pipelines.

## 📁 Directory Structure

```
fixtures/
├── README.md                    # This file
├── FIXTURE_METADATA_PATTERN.md  # Metadata system documentation
├── images/                      # Image test fixtures
├── audio/                       # Audio test fixtures
├── video/                       # Video test fixtures
├── documents/                   # Document test fixtures
└── code/                        # Code snippet fixtures
```

## 🎯 Purpose

Test fixtures serve multiple critical functions:

1. **Resource Integration Testing**: Validate that resources (Ollama, Whisper, etc.) correctly process various file types
2. **AI Model Testing**: Test AI capabilities with known inputs and expected outputs
3. **Edge Case Handling**: Ensure robust error handling with malformed or edge-case files
4. **Performance Benchmarking**: Measure processing times across different file sizes and complexities
5. **Regression Testing**: Detect changes in resource behavior across updates

## 📂 Fixture Categories

### [Images](./images/README.md)
Test images for OCR, image analysis, and vision model testing.
- **Formats**: PNG, JPEG, WebP, GIF, BMP, TIFF, HEIC
- **Categories**: Synthetic, real-world photos, OCR test images, edge cases
- **Metadata**: [metadata.yaml](./images/metadata.yaml) with dimensions, tags, and test data
- **Use Cases**: OCR accuracy, image classification, format support validation

### [Audio](./audio/README.md)
Audio files for transcription, speech recognition, and audio processing tests.
- **Formats**: WAV, MP3, OGG, FLAC, M4A, AAC
- **Categories**: Speech, music, silence, sound effects, edge cases
- **Metadata**: [metadata.yaml](./audio/metadata.yaml) with duration, sample rates, expected transcriptions
- **Use Cases**: Whisper testing, format support, transcription accuracy

### [Video](./video/README.md)
Video files for multimedia processing and analysis.
- **Formats**: MP4, AVI, MOV, WEBM, FLV, MPEG
- **Categories**: Test patterns, real content, various resolutions
- **Use Cases**: Video analysis, frame extraction, format support

### [Documents](./documents/README.md)
Document files for text extraction and content analysis.
- **Formats**: PDF, DOCX, TXT, RTF, Markdown
- **Categories**: Office documents, code files, structured data
- **Metadata**: Document properties and expected extraction results
- **Use Cases**: Unstructured.io testing, content extraction, parsing accuracy

### [Code](./code/README.md)
Code snippets and scripts for code analysis and execution testing.
- **Languages**: JavaScript, Python, Go, Java, Shell scripts
- **Categories**: Algorithms, syntax examples, error cases
- **Use Cases**: Code analysis, syntax highlighting, execution testing

## 🔧 Using Fixtures in Tests

### Basic Usage
```bash
# In BATS tests
@test "process image file" {
    local test_image="$FIXTURES_DIR/images/real-world/nature/mountain_landscape.jpg"
    run process_image "$test_image"
    assert_success
}
```

### With Metadata
```bash
# Using metadata helpers
source "$FIXTURES_DIR/images/test_helpers.sh"

@test "OCR accuracy test" {
    # Get all OCR test images
    while IFS= read -r image_path; do
        local expected=$(get_expected_text "$image_path")
        run ocr_process "$FIXTURES_DIR/images/$image_path"
        assert_output --partial "$expected"
    done < <(get_images_by_tag "ocr")
}
```

### Test Suites
Each fixture type defines test suites in its metadata file:
```bash
# Get files for specific test suite
get_test_suite_images "formatSupport"  # Returns paths for format testing
get_test_suite_audio "transcriptionAccuracy"  # Returns audio files for accuracy testing
```

## 📊 Metadata System

Fixtures use a standardized YAML metadata system documented in [FIXTURE_METADATA_PATTERN.md](./FIXTURE_METADATA_PATTERN.md).

Key benefits:
- **Data-driven testing**: Tests can iterate over fixtures with specific properties
- **Validation**: Ensure fixtures match their metadata
- **Discoverability**: Find appropriate test files by tags or properties
- **Maintenance**: Track expected outputs and test configurations

### Metadata Files
- `images/metadata.yaml` - Image metadata and test expectations
- `audio/metadata.yaml` - Audio properties and transcription expectations
- `video/metadata.yaml` - Video metadata and processing expectations
- `documents/metadata.yaml` - Document properties and extraction expectations
- `workflows/metadata.yaml` - Workflow definitions and test configurations

### Helper Scripts
Each fixture type provides helper scripts:
- `test_helpers.sh` - Bash functions for accessing metadata in tests
- `validate_metadata.py` - Python script to validate metadata accuracy

## 🚀 Adding New Fixtures

### 1. Choose Appropriate Directory
Place fixtures in the correct category subdirectory. Create new subdirectories for logical grouping.

### 2. Follow Naming Conventions
- Use descriptive names: `speech_interview_30sec.mp3` not `audio1.mp3`
- Include key properties: `image_ocr_invoice_300dpi.png`
- Use underscores for spaces, lowercase for consistency

### 3. Update Metadata
Add entry to the appropriate YAML file:
```yaml
- path: category/new_fixture.ext
  format: ext
  fileSize: 12345
  tags: [tag1, tag2]
  testData:
    expectedOutput: "..."
```

### 4. Validate
Run validation to ensure metadata matches:
```bash
cd fixtures/images && python validate_metadata.py
```

### 5. Document Special Cases
If the fixture has special requirements or known issues, document them in the metadata or README.

## 🔍 Best Practices

1. **Diversity**: Include fixtures that represent real-world variety
2. **Edge Cases**: Always include malformed/corrupt files for error testing
3. **Size Range**: Include small, medium, and large files for performance testing
4. **Licensing**: Only use content that's freely licensed (Creative Commons, public domain)
5. **Documentation**: Document any special properties or expected behaviors
6. **Organization**: Group related fixtures in subdirectories
7. **Validation**: Regularly validate that fixtures match their metadata

## 🛠️ Maintenance

### Regular Tasks
- Run metadata validation scripts monthly
- Update metadata when adding new fixtures
- Remove outdated or redundant fixtures
- Ensure all fixtures have appropriate test coverage

### Validation Commands
```bash
# Validate all fixture metadata
for dir in images audio video documents; do
    if [ -f "$dir/validate_metadata.py" ]; then
        echo "Validating $dir..."
        (cd "$dir" && python validate_metadata.py)
    fi
done
```

## 📚 Related Documentation

- [Test Framework Documentation](../framework/README.md)
- [Resource Testing Guide](../README.md)
- [Fixture Metadata Pattern](./FIXTURE_METADATA_PATTERN.md)

## 🤝 Contributing

When contributing new fixtures:
1. Ensure files are appropriately licensed
2. Add comprehensive metadata
3. Include both typical and edge cases
4. Update relevant documentation
5. Run validation scripts

For questions or suggestions, please refer to the main [Vrooli Testing Documentation](../README.md).