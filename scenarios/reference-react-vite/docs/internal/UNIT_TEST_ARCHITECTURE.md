# Unit Test Architecture

This document describes the unit testing infrastructure for the reference-react-vite scenario.

## Overview

The scenario follows professional test architecture patterns:
- Co-located test files (tests live next to the code they test)
- Centralized mock packages (reusable, consistent mocking)
- Table-driven tests with systematic edge case coverage
- Dependency injection for testability

## Go API Testing

### Test File Organization

```
api/
├── domain/
│   ├── tasks/
│   │   ├── task.go
│   │   └── task_test.go      # Co-located domain tests
│   ├── projects/
│   │   ├── project.go
│   │   └── project_test.go
│   └── notes/
│       ├── note.go
│       └── note_test.go
├── handlers/
│   ├── tasks.go
│   └── tasks_test.go         # Co-located handler tests
├── internal/
│   ├── mocks/
│   │   └── repository.go     # Centralized mock implementations
│   └── testutil/
│       ├── helpers.go        # Test utilities
│       └── fixtures.go       # Test data factories
└── repository/
    └── repository.go         # Interfaces (no tests here)
```

### Running Tests

```bash
# Run all tests
cd api && go test ./...

# Run tests with verbose output
go test ./... -v

# Run domain tests only
go test ./domain/...

# Run handler tests only
go test ./handlers/...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Patterns

#### Table-Driven Tests

All tests use table-driven patterns for systematic coverage:

```go
func TestStatus_Validate(t *testing.T) {
    tests := []struct {
        name     string
        status   Status
        wantErr  bool
        category string  // happy_path, boundary, error, edge_case
    }{
        {"valid_pending", StatusPending, false, "happy_path"},
        {"invalid_empty", Status(""), true, "error"},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            err := tc.status.Validate()
            // assertions...
        })
    }
}
```

#### Mock Repository Pattern

Mocks use a builder pattern for configuration:

```go
// Create a mock with pre-populated data
repo := mocks.NewMockTaskRepository().
    WithTask(testutil.NewTaskFactory().
        WithID("task-123").
        WithTitle("Test Task").
        Build())

// Create a mock that returns errors
repo := mocks.NewMockTaskRepository().
    WithCreateError(errors.New("database error"))

// Use in tests
handler := handlers.NewTaskHandler(repo)
```

#### Test Factory Pattern

Factory functions create valid test data with optional overrides:

```go
// Default valid task
task := testutil.NewTaskFactory().Build()

// Task with custom values
task := testutil.NewTaskFactory().
    WithID("custom-id").
    WithTitle("Custom Title").
    WithStatus(tasks.StatusCompleted).
    Build()
```

### Mock Repository Interface

Each repository mock provides:

| Method | Purpose |
|--------|---------|
| `WithTask(task)` | Pre-populate with test data |
| `WithCreateError(err)` | Configure Create to fail |
| `WithFindError(err)` | Configure FindByID to fail |
| `WithListError(err)` | Configure List to fail |
| `WithUpdateError(err)` | Configure Update to fail |
| `WithDeleteError(err)` | Configure Delete to fail |
| `CreateCallCount()` | Count Create invocations |
| `DeleteCallCount()` | Count Delete invocations |
| `Reset()` | Clear all state |

### Test Helpers

```go
// HTTP response assertions
testutil.AssertStatus(t, rec, http.StatusOK)
testutil.AssertJSON(t, rec, &response)
testutil.AssertContentType(t, rec, "application/json")
testutil.AssertError(t, rec, "expected error message")

// Request creation
req := testutil.MakeJSONRequest(t, http.MethodPost, "/api/v1/tasks", body)

// Pointer helpers for UpdateInput fields
testutil.StringPtr("value")
testutil.IntPtr(42)
```

## TypeScript UI Testing

### Test File Organization

```
ui/src/
├── App.tsx
├── App.test.tsx              # Co-located component test
├── test-utils/
│   ├── setup.ts              # Global test setup
│   ├── index.ts              # Re-exports
│   ├── renderWithProviders.tsx
│   └── factories.ts          # Test data factories
└── lib/
    └── api.ts                # API module (mocked in tests)
```

### Running Tests

```bash
# Run all tests
cd ui && pnpm test

# Run tests in watch mode
pnpm test -- --watch

# Run with coverage
pnpm test -- --coverage
```

### Test Configuration

Tests are configured in `vite.config.ts`:

```typescript
export default defineConfig({
  test: {
    globals: true,           // Use global describe/it/expect
    environment: 'jsdom',    // Browser-like environment
    setupFiles: ['./src/test-utils/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['json-summary', 'json', 'text'],
    }
  }
});
```

### Test Patterns

#### Component Testing with Providers

```tsx
import { renderWithProviders, createMockHealthResponse } from './test-utils';

describe('App', () => {
  it('displays health status', async () => {
    vi.mocked(fetchHealth).mockResolvedValue(
      createMockHealthResponse({ status: 'healthy' })
    );

    renderWithProviders(<App />);

    await waitFor(() => {
      expect(screen.getByText('Status: healthy')).toBeInTheDocument();
    });
  });
});
```

#### Mock API Calls

```tsx
// Mock the API module
vi.mock('./lib/api', () => ({
  fetchHealth: vi.fn(),
}));

import { fetchHealth } from './lib/api';

beforeEach(() => {
  vi.clearAllMocks();
});

it('handles API errors', async () => {
  vi.mocked(fetchHealth).mockRejectedValue(new Error('Network error'));
  // ...
});
```

#### Test Data Factories

```typescript
// Create mock data with defaults
const health = createMockHealthResponse();

// Create with overrides
const health = createMockHealthResponse({
  status: 'unhealthy',
  service: 'test-service',
});

// Create mock list responses
const tasks = createMockListResponse([
  createMockTask({ id: '1' }),
  createMockTask({ id: '2' }),
]);
```

### Global Test Setup

The `setup.ts` file configures:
- `@testing-library/jest-dom` matchers
- Automatic cleanup after each test
- Mock for `ResizeObserver` (not in jsdom)
- Mock for `IntersectionObserver` (not in jsdom)
- Mock for `matchMedia` (not in jsdom)

## Systematic Edge Case Coverage

### Equivalence Partitioning

Tests cover input partitions:
- Valid inputs (happy path)
- Invalid inputs (error cases)
- Boundary values (min, max, min-1, max+1)
- Edge cases (unicode, whitespace, null/empty)

### Example Categories

```go
tests := []struct {
    // ...
    category string  // For documentation
}{
    // Happy path - normal successful operations
    {"valid_input", input, false, "happy_path"},

    // Boundary values - at limits
    {"at_min_length", minInput, false, "boundary"},
    {"at_max_length", maxInput, false, "boundary"},
    {"above_max", tooLongInput, true, "boundary"},

    // Error cases - invalid inputs
    {"empty_value", emptyInput, true, "error"},
    {"invalid_format", badInput, true, "error"},

    // Edge cases - unusual but valid
    {"unicode_content", unicodeInput, false, "edge_case"},
    {"whitespace_trimmed", spacedInput, false, "edge_case"},
}
```

## Related Documentation

- [DOC: docs/internal/SEAMS.md] - Architectural seams and test strategy
- [DOC: docs/concepts/ARCHITECTURE.md] - Domain architecture

## Last Updated

2026-03-11 - Initial documentation with Phase 3 unit testing infrastructure
