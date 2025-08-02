# Test Fixtures Directory

This directory contains sample documents and test data used by the unstructured-io test suite.

## Purpose

Test fixtures provide consistent, reliable test data for:
- BATS shell script tests
- Python API tests
- Integration testing
- Performance benchmarking

## Recommended Fixtures

### Sample Documents
- `sample.pdf` - Multi-page PDF with text, tables, and images
- `simple.txt` - Plain text document for basic testing
- `complex.docx` - Word document with formatting, tables, lists
- `table-heavy.xlsx` - Excel file with multiple sheets and complex tables
- `scanned.pdf` - Scanned document requiring OCR
- `multilingual.pdf` - Document with multiple languages for OCR testing

### HTML Examples
- `simple.html` - Basic HTML with headers, paragraphs, lists
- `table.html` - HTML document focused on table extraction
- `complex.html` - Complex HTML with nested structures

### Images for OCR
- `text-image.png` - Clear text image for OCR testing
- `poor-quality.jpg` - Low-quality scan to test OCR limits
- `handwritten.jpg` - Handwritten text sample

## Usage in Tests

```bash
# Example BATS test usage
@test "process sample PDF" {
    run unstructured_io::process_document "tests/fixtures/sample.pdf"
    [ "$status" -eq 0 ]
}
```

```python
# Example Python test usage
def test_pdf_processing():
    result = client.process_document("tests/fixtures/sample.pdf")
    assert len(result) > 0
```

## Adding New Fixtures

When adding new test fixtures:
1. Use descriptive filenames that indicate the test purpose
2. Keep file sizes reasonable (< 5MB preferred)
3. Include a variety of formats and complexity levels
4. Document any special characteristics or expected behavior
5. Update relevant test files to use the new fixtures

## Security Note

- Do not include sensitive or proprietary content in fixtures
- Use only publicly available or generated test content
- Ensure all fixtures are safe for version control and distribution