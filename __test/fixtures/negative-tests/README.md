# Negative Test Fixtures (Generated On-Demand)

This directory contains scripts for generating negative test cases - files that are intentionally corrupted, malformed, or edge cases designed to test error handling and resilience.

## 🚀 Files Are Now Generated On-Demand!

As of August 2025, all negative test files are **generated dynamically** rather than stored statically. This saves approximately **1.4MB** of disk space and makes the test suite more maintainable.

## 📋 How It Works

### Automatic Generation
When tests need negative test files, they are automatically generated using:
```bash
/home/matthalloran8/Vrooli/__test/fixtures/generators/generate-stress-files.sh all-negative
```

### Manual Generation
To manually generate all negative test files:
```bash
../generators/generate-stress-files.sh all-negative
```

### Cleanup
To remove generated files:
```bash
../generators/generate-stress-files.sh clean
```

## 📁 Categories

### Audio
- Empty files
- Invalid headers
- Truncated data
- Non-audio content with audio extensions
- Random binary data as WAV

### Documents
- Malformed JSON/XML
- Invalid PDF/Word formats
- Circular references
- Huge but empty structures
- Corrupted DOCX
- Partial PDFs

### Images
- Invalid image formats
- Truncated headers
- Non-image content with image extensions
- Empty JPEGs
- Random binary as images

### Edge Cases
- Zero-byte files
- Unicode stress tests
- Very long lines (1M characters)
- Special characters in filenames
- Mixed line endings
- Null bytes in text
- Files with spaces and emojis in names

### Stress Tests
- Many columns CSV (1000 columns)
- Changing content simulation
- Directory with 100+ small files (via many-files command)

## 🧪 Usage

These files should be used to verify that services:
1. Handle errors gracefully without crashing
2. Return appropriate error codes
3. Don't consume excessive resources
4. Properly validate input before processing

### Example Test Usage
```bash
# The validate script auto-generates files if needed
./validate-negative-tests.sh

# Or in your test code
source /path/to/generators/ensure-test-files.sh
ensure_all_test_files  # Generates if missing
```

## 📊 Benefits of On-Demand Generation

- **Disk Space**: Saves ~1.4MB by not storing files statically
- **Git Hygiene**: No binary files in repository
- **Flexibility**: Easy to adjust file sizes and properties
- **Maintenance**: Single source of truth in generator script
- **CI/CD Friendly**: Generate only what's needed per test

## 🔒 Safety Note

These files are designed to be challenging but safe. They do not contain:
- Actual malware
- Real zip bombs
- Genuinely dangerous payloads
- Files that could harm the system

They are purely for testing error handling capabilities.

## 📚 See Also

- [Generator Script](../generators/generate-stress-files.sh)
- [Generator Documentation](../generators/README.md)
- [Validation Script](./validate-negative-tests.sh)