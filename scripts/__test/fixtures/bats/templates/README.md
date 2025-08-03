# BATS Test Templates

This directory contains template files to help you quickly create new BATS tests for different scenarios.

## Template Organization

Templates are organized by test complexity and purpose:

### 📁 basic/
**Simple, straightforward tests**
- `standard-test.bats` - Basic test with standard mocks (Docker, HTTP, system commands)
- Use for: Unit tests, simple script validation, basic command testing

### 📁 resource/
**Single resource integration tests**
- `resource-test.bats` - Test a single Vrooli resource (Ollama, Whisper, etc.)
- `comprehensive-resource-test.bats` - Exhaustive testing of a resource's functionality
- Use for: Testing resource installation, configuration, health checks

### 📁 integration/
**Multi-resource coordination tests**
- `integration-test.bats` - Basic multi-resource test
- `data-pipeline.bats` - Testing data flow between resources
- `multi-resource-workflow.bats` - Complex workflow across multiple resources
- Use for: End-to-end testing, workflow validation, resource interaction

### 📁 performance/
**Performance and benchmark tests**
- `performance-test.bats` - Basic performance testing
- `load-testing.bats` - Load and stress testing
- `resource-benchmarks.bats` - Resource-specific performance benchmarks
- Use for: Performance regression testing, optimization validation

### 📁 advanced/
**Complex testing scenarios**
- `advanced-test.bats` - Advanced testing patterns and edge cases
- Use for: Complex state management, error recovery testing, edge cases

## Quick Start

1. **Choose the right template:**
   ```bash
   # For a simple test
   cp basic/standard-test.bats my-test.bats
   
   # For testing Ollama
   cp resource/resource-test.bats test-ollama.bats
   
   # For testing Ollama + Whisper integration
   cp integration/integration-test.bats test-ollama-whisper.bats
   ```

2. **Customize the template:**
   - Update the test description at the top
   - Modify the setup() function for your needs
   - Replace example tests with your actual test cases
   - Remove any unused assertions or examples

3. **Run your test:**
   ```bash
   bats my-test.bats
   ```

## Template Selection Guide

| If you need to... | Use this template |
|-------------------|-------------------|
| Test a simple bash script | `basic/standard-test.bats` |
| Test Docker commands | `basic/standard-test.bats` |
| Test HTTP endpoints | `basic/standard-test.bats` |
| Test Ollama installation | `resource/resource-test.bats` |
| Test all Ollama features | `resource/comprehensive-resource-test.bats` |
| Test Ollama + Whisper | `integration/integration-test.bats` |
| Test data processing pipeline | `integration/data-pipeline.bats` |
| Test performance regression | `performance/performance-test.bats` |
| Test under load | `performance/load-testing.bats` |
| Test complex error handling | `advanced/advanced-test.bats` |

## Template Features

All templates include:
- ✅ Proper BATS version requirement (`bats_require_minimum_version 1.5.0`)
- ✅ Automatic path resolution using the new path resolver
- ✅ Appropriate setup/teardown functions
- ✅ Example test cases relevant to the template type
- ✅ Common assertions pre-configured
- ✅ Helpful comments explaining customization points

## Creating Custom Templates

If none of the existing templates fit your needs:

1. Start with the closest matching template
2. Save it with a descriptive name in the appropriate directory
3. Document any special requirements in comments
4. Consider contributing it back if it's generally useful

## Best Practices

1. **Always use the path resolver** - Don't hardcode paths
2. **Clean up in teardown()** - Always call `cleanup_mocks`
3. **Use appropriate assertions** - Check the assertions reference
4. **Test isolation** - Each test should be independent
5. **Meaningful test names** - Use descriptive @test descriptions
6. **Mock appropriately** - Only mock what you need for performance

## See Also

- [Assertions Reference](../docs/assertions.md)
- [Mock Registry](../docs/mock-registry.md)
- [Setup Guide](../docs/setup-guide.md)
- [Troubleshooting](../docs/troubleshooting.md)