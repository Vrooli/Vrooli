# UI Testing Setup

This directory contains the testing infrastructure for the UI package, providing unified testing with optional database integration via testcontainers.

## Overview

The UI package uses a unified testing approach where all tests run with the same configuration and scripts. Tests that need database access automatically get testcontainers PostgreSQL and Redis instances, while simpler unit tests run without them.

### Key Features
- **Unified Configuration**: Single `vitest.config.ts` for all tests
- **Automatic Database Setup**: Testcontainers start when needed for integration tests
- **Type-Safe Fixtures**: Production-grade fixtures using real `@vrooli/shared` functions
- **Seamless Testing**: Run all tests with `pnpm test`

## Directory Structure

```
__test/
├── fixtures/              # Legacy fixtures (being migrated)
├── fixtures-updated/       # New production-grade fixtures
│   ├── factories/         # Type-safe fixture factories
│   ├── integrations/      # Real database integration tests
│   ├── scenarios/         # Multi-step workflow tests
│   ├── states/           # UI state fixtures
│   └── types.ts          # Core type definitions
├── mocks/                 # MSW server setup
├── helpers/              # Storybook utilities
├── setup.vitest.ts       # Unified test setup with database support
├── testUtils.tsx         # React testing utilities
└── README.md             # This file
```

## Quick Start

### Running Tests
```bash
# Run all tests (unit and integration)
pnpm test

# Run tests in watch mode
pnpm test-watch

# Run with coverage
pnpm test-coverage

# Run with pretest build
pnpm test-all
```

All tests run with the same configuration. Tests that need database access will automatically have testcontainers available.

### Writing Unit Tests

```typescript
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MyComponent } from '../MyComponent.js';

describe('MyComponent', () => {
    it('should render correctly', () => {
        render(<MyComponent />);
        // Vitest has built-in DOM assertions - no jest-dom needed!
        expect(screen.getByText('Hello')).toBeInTheDocument();
    });
});
```

### Writing Integration Tests

```typescript
import { describe, it, expect } from 'vitest';
import { BookmarkFixtureFactory } from './fixtures/factories/BookmarkFixtureFactory.js';
import { renderWithProviders, waitForDatabaseRecord } from './testUtils.js';

describe('Bookmark Integration', () => {
    it('should save form data to database', async () => {
        const factory = new BookmarkFixtureFactory();
        const formData = factory.createFormData('complete');
        
        const { user, getByRole } = renderWithProviders(<BookmarkForm />);
        
        // Fill and submit form
        await user.type(getByRole('textbox', { name: /list name/i }), formData.newListLabel);
        await user.click(getByRole('button', { name: /save/i }));
        
        // Verify data was saved to real database
        const saved = await waitForDatabaseRecord('bookmark', {
            list: { label: formData.newListLabel }
        });
        expect(saved).toBeDefined();
    });
});
```

## Fixture Architecture

The UI fixtures follow a production-grade architecture with:

### Fixture Factories
Type-safe data generation using real functions from `@vrooli/shared`:

```typescript
const factory = new BookmarkFixtureFactory();

// Generate different scenarios
const minimal = factory.createFormData('minimal');
const complete = factory.createFormData('complete');
const withNewList = factory.createFormData('withNewList');

// Transform using real shape functions
const apiInput = factory.transformToAPIInput(formData);

// Validate using real validation schemas
const validation = await factory.validateFormData(formData);
```

### Integration Testing
Real database testing with automatic cleanup:

```typescript
const integration = new BookmarkIntegrationTest();

// Test complete flow with database verification
const result = await integration.testCompleteFlow({
    formData,
    shouldSucceed: true
});

expect(result.data.databaseRecord).toBeDefined();
```

### Scenario Orchestration
Multi-step workflow testing:

```typescript
const scenario = new UserBookmarksProjectScenario();

await scenario.execute({
    user: userConfig,
    project: projectConfig,
    bookmarks: bookmarkConfigs
});
```

## Database Integration

Tests that need database access automatically get testcontainers PostgreSQL and Redis instances:

- **Automatic Setup**: Database containers start automatically when `DB_URL` and `REDIS_URL` are set
- **Isolation**: Each test suite gets a fresh database
- **Transactions**: Tests run in transactions that are rolled back
- **Real Constraints**: Tests validate actual foreign key constraints
- **Performance**: Measure actual database query performance

The setup is handled by `vitest.global-setup.ts` which runs once before all tests.

## MSW Integration

For component testing without full database integration:

```typescript
import { server } from './mocks/server.js';
import { bookmarkFixtures } from './fixtures/factories/BookmarkFixtureFactory.js';

// Setup MSW handlers
const handlers = bookmarkFixtures.createMSWHandlers();
server.use(...handlers.success);

// Test component with mocked API
render(<BookmarkComponent />);
```

## Built-in Vitest Features

### DOM Assertions
Vitest includes built-in DOM assertions forked from `@testing-library/jest-dom`:
- `toBeInTheDocument()` - Check if element exists in DOM
- `toHaveTextContent()` - Verify element text content
- `toBeChecked()` - Check checkbox/radio state
- `toBeDisabled()` - Check if element is disabled
- `toContainHTML()` - Verify HTML content
- And many more!

**No additional setup required** - these work out of the box with Vitest.

## Best Practices

### DO's ✅
- Use TypeScript types from `@vrooli/shared`
- Test with real validation schemas and shape functions
- Use testcontainers for integration tests
- Clean up test data between tests
- Use semantic queries (getByRole, getByLabelText)
- Test user interactions, not implementation details
- Leverage Vitest's built-in DOM assertions

### DON'Ts ❌
- Don't use `any` types in fixtures
- Don't mock internal business logic
- Don't skip database verification in integration tests
- Don't create global test state
- Don't test implementation details
- Don't hardcode test data in components

## Performance Considerations

- **Simple tests**: Fast (~1-5 seconds per test) when not using database
- **Database tests**: Slower (~5-30 seconds per test) due to database operations
- **Memory**: Tests with database use more memory due to testcontainers
- **Execution**: All tests run sequentially for database consistency
- **Startup**: Initial container startup adds ~10-20 seconds to first test run

## Troubleshooting

### Common Issues

1. **"Test containers not available"**
   - This is normal for simple unit tests
   - Database is only available when containers have started
   - Check that global setup completed successfully

2. **"Database client not available"**
   - This message appears for tests that don't need database access
   - Integration utilities automatically check if database is available
   - Normal behavior for component-only tests

3. **Tests timing out**
   - Database tests can take longer due to container startup
   - Default timeout is 60 seconds which should be sufficient
   - First test run takes longer due to container initialization

4. **Memory issues**
   - Testcontainers require more memory
   - The scripts already include `NODE_OPTIONS='--max-old-space-size=8192'`

### Debugging

```bash
# Run single test file
pnpm test src/path/to/test.test.ts

# Enable debug logging
DEBUG=testcontainers* pnpm test

# Check container status
docker ps

# View test output with more detail
pnpm test -- --reporter=verbose
```

## Migration from Legacy Fixtures

If you encounter legacy fixture patterns:

1. **Identify the object type** (e.g., Bookmark, User, Team)
2. **Check if new fixture exists** in `fixtures-updated/factories/`
3. **Use the new factory pattern** instead of legacy helpers
4. **Follow the integration test pattern** for database testing

Example migration:
```typescript
// Legacy ❌
import { mockBookmarkService } from '../helpers/bookmarkTransformations.js';
const bookmark = await mockBookmarkService.create(data);

// New ✅
import { BookmarkFixtureFactory } from '../fixtures/factories/BookmarkFixtureFactory.js';
const factory = new BookmarkFixtureFactory();
const integration = new BookmarkIntegrationTest();
const result = await integration.testCreateFlow(factory.createFormData('complete'));
```

## Contributing

When adding new test fixtures:

1. **Follow the factory pattern** - Create classes that implement standardized interfaces
2. **Use real functions** - Import shape and validation functions from `@vrooli/shared`
3. **Add integration tests** - Create corresponding integration test files
4. **Document scenarios** - Provide clear scenario names and examples
5. **Include error cases** - Test both success and failure paths
6. **Update exports** - Add new fixtures to the central index files