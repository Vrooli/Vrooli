# Test File Generators

This directory contains scripts that generate test files on-demand rather than storing them statically.

## Why Generators?

- **Disk Space**: Saves ~60MB by not storing large test files
- **Flexibility**: Easy to adjust file sizes and counts for different test scenarios  
- **Performance**: Generate only what's needed for specific tests
- **Maintenance**: No need to track large binary files in git

## Available Generators

### `generate-stress-files.sh`

Main generator script for creating test files on-demand.

#### Commands:

```bash
# Generate 100 small files (default stress test)
./generate-stress-files.sh many-files

# Generate custom number of files
./generate-stress-files.sh many-files /tmp/test 500

# Generate large text file (10MB default)
./generate-stress-files.sh huge-text

# Generate custom size text file
./generate-stress-files.sh huge-text /tmp/big.txt 50

# Generate all standard test files
./generate-stress-files.sh all

# Clean up generated files
./generate-stress-files.sh clean
```

### `ensure-test-files.sh`

Helper script for tests to automatically generate required files.

```bash
# In your test script:
source /path/to/ensure-test-files.sh

# Ensure stress test files exist
ensure_stress_files

# Ensure edge case files exist  
ensure_edge_files

# Ensure all test files exist
ensure_all_test_files

# Clean up when done (optional)
cleanup_generated_files
```

## Files Generated On-Demand

### Stress Test Files
- `many_files/` - Directory with 100+ small files
- `many_columns.csv` - CSV with 1000 columns
- `changing_content.txt` - Simulated changing file

### Edge Case Files  
- `huge_text.txt` - 10MB text file
- `binary_disguised.pdf` - 100KB binary data
- `corrupted.json` - Malformed JSON
- `invalid_xml.xml` - Invalid XML structure
- `malformed.csv` - Broken CSV format
- `empty.pdf` - Zero-byte PDF
- `unicode_stress_test.txt` - Unicode edge cases
- `long_single_line.txt` - 1M character single line
- `network_timeout.url` - Network timeout simulation

## Usage in Tests

### Example 1: BATS Test
```bash
@test "handle many files" {
    source "$BATS_TEST_DIRNAME/../fixtures/generators/ensure-test-files.sh"
    ensure_stress_files
    
    # Your test using the files
    run process_directory "$STRESS_DIR/many_files"
    assert_success
}
```

### Example 2: Shell Script Test
```bash
#!/usr/bin/env bash
source "$(dirname "$0")/../fixtures/generators/ensure-test-files.sh"

# Generate files if needed
ensure_all_test_files

# Run your tests
test_large_file_handling() {
    local result=$(process_file "$EDGE_DIR/huge_text.txt")
    [[ "$result" == "success" ]]
}
```

### Example 3: Direct Generation
```bash
# Generate specific files for a test
./generators/generate-stress-files.sh many-files /tmp/test-dir 1000
./generators/generate-stress-files.sh huge-text /tmp/massive.txt 100

# Run tests...

# Clean up
rm -rf /tmp/test-dir /tmp/massive.txt
```

## Disk Space Savings

By using generators instead of static files:
- **Removed**: ~60MB of static test files
- **Added**: ~10KB of generator scripts
- **Net Savings**: ~60MB

## Best Practices

1. **Always ensure files exist** before using them in tests
2. **Clean up** generated files after tests if they're large
3. **Use custom paths** for parallel test execution
4. **Document** which files your test expects to be generated

## Maintenance

To regenerate all standard test files:
```bash
./generate-stress-files.sh all
```

To clean all generated files:
```bash
./generate-stress-files.sh clean
```