# Testing Templates

This directory contains **template files** that scenarios can copy and customize for their specific testing needs. These are NOT libraries to be imported, but rather starting points for your own test helpers.

## Important: Templates vs Libraries

- **Templates** (this directory): Copy and customize for your scenario
- **Libraries** (../shell/, ../unit/): Source/import directly in your tests

## Go Templates

### test_helpers.go.template

A comprehensive template for Go HTTP testing utilities. Copy this to your scenario's `api/` directory and customize.

**Features:**
- Test environment setup with isolated directories
- HTTP request/response testing utilities
- JSON validation helpers
- Performance and concurrency testing
- Test data generators

**Usage:**
```bash
# Copy to your scenario
cp scripts/lib/testing/templates/go/test_helpers.go.template \
   scenarios/my-scenario/api/test_helpers.go

# Customize the package name and adapt to your needs
sed -i 's/package httptest/package main/' scenarios/my-scenario/api/test_helpers.go
```

**Key Functions to Customize:**
- `SetupTestLogger()` - Replace with your scenario's logger
- `SetupTestDirectory()` - Add your scenario-specific initialization
- Test data generators - Create scenario-specific test data

### error_patterns.go.template

Sophisticated error testing patterns for comprehensive test coverage.

**Features:**
- Reusable error test patterns
- REST resource testing patterns
- Test scenario builder with fluent interface
- Performance and concurrency test patterns

**Usage:**
```bash
# Copy to your scenario
cp scripts/lib/testing/templates/go/error_patterns.go.template \
   scenarios/my-scenario/api/test_patterns.go

# Customize the package name
sed -i 's/package patterns/package main/' scenarios/my-scenario/api/test_patterns.go
```

**Customization Points:**
- Add scenario-specific error patterns
- Create custom validation functions
- Define business logic test patterns

## Using Templates Effectively

### 1. Copy, Don't Import

These templates are designed to be copied into your scenario and customized:

```bash
# Wrong - trying to import
import "path/to/templates/go/test_helpers.go.template"  # ❌ Won't work

# Right - copy and customize
cp templates/go/test_helpers.go.template my-scenario/api/test_helpers.go  # ✅
```

### 2. Adapt to Your Scenario

After copying, customize the templates:

1. **Change package names** to match your scenario
2. **Update imports** to use your scenario's dependencies
3. **Customize helper functions** for your specific needs
4. **Remove unused functions** to keep code clean
5. **Add scenario-specific utilities**

### 3. Example: visited-tracker

The visited-tracker scenario shows excellent template usage:

```go
// scenarios/visited-tracker/api/test_helpers.go
package main  // Changed from 'package httptest'

// Added scenario-specific types
type TestCampaign struct {
    Campaign     *Campaign
    TrackedFiles []TrackedFile
    Cleanup      func()
}

// Customized for file storage initialization
func setupTestDirectory(t *testing.T) *TestEnvironment {
    // ... 
    if err := initFileStorage(); err != nil {  // Scenario-specific
        // ...
    }
    // ...
}
```

## Best Practices

### 1. Start Simple

Don't copy everything at once. Start with the functions you need:

```go
// Start with just what you need
func makeHTTPRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
    // Your simplified version
}
```

### 2. Evolve Over Time

As your tests grow, add more sophisticated patterns:

```go
// Later, add error patterns
func testCommonErrors(t *testing.T, handler http.HandlerFunc) {
    // Test invalid UUID, missing resources, etc.
}
```

### 3. Share Common Patterns

If you develop useful patterns, consider:
1. Contributing them back to the templates
2. Creating scenario-specific documentation
3. Sharing with other scenario developers

## Template Structure

```
templates/
├── go/
│   ├── test_helpers.go.template      # HTTP testing utilities
│   └── error_patterns.go.template    # Error testing patterns
└── README.md                          # This file
```

## Future Templates

We plan to add templates for:
- Node.js/TypeScript testing utilities
- Python testing helpers
- Shell script testing patterns
- Integration test templates
- Performance test templates

## Contributing

To contribute a new template:

1. Create a well-documented, generic template
2. Place it in the appropriate language directory
3. Update this README with usage instructions
4. Provide an example of how it's used in a real scenario

## Remember

**Templates are starting points, not constraints.** Feel free to:
- Modify extensively
- Delete what you don't need
- Add what you do need
- Create your own patterns

The goal is to give you a head start, not to enforce a rigid structure.