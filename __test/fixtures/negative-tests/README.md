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
