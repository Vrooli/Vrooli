# Resource Integration Tests - NPM Scripts

The resource integration tests can now be run directly from the project root using npm/pnpm scripts.

## Available Commands

From the project root (`/home/matthalloran8/Vrooli`), you can run:

### Basic Test Commands

```bash
# Run all resource integration tests
pnpm test:resources

# Run only single-resource tests
pnpm test:resources:single

# Run only business scenario tests
pnpm test:resources:scenarios

# Quick test run with 30s timeout and fail-fast
pnpm test:resources:quick

# Debug mode with verbose output and HTTP logging
pnpm test:resources:debug

# List available business scenarios
pnpm test:resources:list

# Clean test artifacts and logs
pnpm test:resources:clean
```

### Testing Specific Resources

To test a specific resource, you'll need to pass additional arguments:

```bash
# Test a specific resource
pnpm test:resources -- --resource qdrant

# Debug a specific resource
pnpm test:resources:debug -- --resource ollama

# Test multiple specific resources
pnpm test:resources -- --resource qdrant --resource minio
```

### Advanced Usage

```bash
# Run scenarios with specific filters
pnpm test:resources:scenarios -- --scenarios "category=ai"

# Run with custom timeout
pnpm test:resources -- --timeout 120

# Run with business report
pnpm test:resources -- --business-report

# Run in verbose mode
pnpm test:resources -- --verbose
```

## Integration with Existing Test Suite

The resource tests complement the existing test structure:

- `pnpm test` - Runs shell and unit tests
- `pnpm test:unit` - Runs unit tests for all packages
- `pnpm test:integration` - Runs package integration tests
- `pnpm test:resources` - Runs resource integration tests (NEW)

## CI/CD Integration

For CI/CD pipelines, you might want to use:

```bash
# Quick validation
pnpm test:resources:quick

# Full test with JSON output
pnpm test:resources -- --output-format json

# Test only if resources are available
pnpm test:resources -- --fail-fast
```

## Examples

```bash
# From project root
cd /home/matthalloran8/Vrooli

# Run all resource tests
pnpm test:resources

# Debug Qdrant integration
pnpm test:resources:debug -- --resource qdrant

# Test AI resources only
pnpm test:resources:single -- --resource ollama --resource whisper

# Clean up after testing
pnpm test:resources:clean
```

## Notes

- These commands automatically change to the correct directory
- All test runner options are available by adding `--` followed by the options
- The tests respect the resource configuration in `~/.vrooli/resources.local.json`
- Only enabled and healthy resources will be tested