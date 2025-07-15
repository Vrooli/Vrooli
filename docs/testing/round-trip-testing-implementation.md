# Round-Trip Testing Implementation

This document describes the implementation of true round-trip testing in Vrooli using Vitest workspaces.

## Overview

Round-trip testing validates the complete data flow through the application:
```
Form Data → Shape Transform → Validation → Endpoint Logic → Database → API Response
```

## Architecture

We use **Vitest workspaces** to run different test types in their appropriate environments:

- **UI Tests** (`ui` project): Run in `happy-dom` environment for React components
- **Round-Trip Tests** (`ui-roundtrip` project): Run in `node` environment to access server code
- **Server Tests** (`server` project): Run in `node` environment
- **Jobs Tests** (`jobs` project): Run in `node` environment
- **Shared Tests** (`shared` project): Run in `node` environment

## File Naming Convention

- Regular UI tests: `*.test.ts` or `*.test.tsx`
- Round-trip tests: `*.roundtrip.test.ts` or `*.roundtrip.test.tsx`

## Available Commands

### Package-Level Commands (in `packages/ui/`)
```bash
# Run all UI tests (regular + round-trip)
pnpm test

# Run only regular UI tests
pnpm test:ui

# Run only round-trip tests
pnpm test:roundtrip

# Watch mode for all tests
pnpm test-watch

# Watch mode for specific test type
pnpm test-watch:ui
pnpm test-watch:roundtrip

# Coverage for all tests
pnpm test-coverage
```

### Root-Level Commands
```bash
# Run all tests across all workspaces
pnpm test:unit

# Run tests for specific workspace
pnpm test:unit:ui           # UI tests only
pnpm test:unit:ui-roundtrip # Round-trip tests only
pnpm test:unit:server       # Server tests
pnpm test:unit:jobs         # Jobs tests
pnpm test:unit:shared       # Shared tests

# Watch mode for all workspaces
pnpm test-watch
```

### Direct Vitest Commands
```bash
# Run all tests
npx vitest

# Run specific project
npx vitest --project ui-roundtrip
npx vitest --project ui
npx vitest --project server

# Run specific file pattern
npx vitest bookmark.roundtrip.test
```

## Writing Round-Trip Tests

Round-trip tests should:

1. **Use Real Functions**: Import and use actual shape, validation, and endpoint functions
2. **Test Complete Flow**: Validate data transformations through all layers
3. **Live in UI Package**: Even though they import server code, they test UI→Server flow
4. **Use `.roundtrip.test.ts` Extension**: This ensures they run in the correct environment

### Example Structure

```typescript
// bookmark.roundtrip.test.ts
import { describe, it, expect } from 'vitest';
import { shapeBookmark, bookmarkValidation } from '@vrooli/shared';
import { bookmarkCreate } from '@vrooli/server/endpoints/logic/bookmark';

describe('Bookmark Round-Trip', () => {
    it('should handle complete flow', async () => {
        // 1. Form data (UI layer)
        const formData = { /* ... */ };
        
        // 2. Shape transform (using real function)
        const apiInput = shapeBookmark.create(formData);
        
        // 3. Validation (using real schema)
        const validated = await bookmarkValidation.create.validate(apiInput);
        
        // 4. Endpoint logic (using real function)
        const result = await bookmarkCreate.logic({
            input: validated,
            userData,
            prisma
        });
        
        // 5. Verify complete flow
        expect(result).toBeDefined();
    });
});
```

## Infrastructure

### Testcontainers
- PostgreSQL and Redis containers start automatically
- Migrations run before tests
- Containers are shared across all test types
- Cleanup happens automatically

### Environment Variables
- Database URL and Redis URL are set automatically
- No manual configuration needed
- Server components initialize on-demand

## Benefits

1. **True Integration Testing**: Tests real code paths, not mocks
2. **Type Safety**: Full TypeScript support across all layers
3. **Environment Isolation**: Each test type runs in its optimal environment
4. **Fast Feedback**: Can run specific test types independently
5. **Easy Debugging**: Clear separation between test types

## Migration Guide

To convert existing tests to round-trip tests:

1. Rename file from `*.test.ts` to `*.roundtrip.test.ts`
2. Remove any mocking of shape/validation functions
3. Import real functions from `@vrooli/shared` and `@vrooli/server`
4. Update test to use real functions instead of mocks

## Troubleshooting

### "Cannot find module" errors
- Ensure server package is built: `cd packages/server && pnpm build`
- Check that the import path is correct

### ES Module errors
- Round-trip tests must use `.roundtrip.test.ts` extension
- Regular UI tests cannot import from `@vrooli/server`

### Test timeout errors
- Round-trip tests may take longer due to real operations
- Default timeout is 60 seconds, can be increased if needed

## Future Improvements

- [ ] Add performance benchmarks for round-trip tests
- [ ] Create more fixture factories for common scenarios
- [ ] Add round-trip test templates/generators
- [ ] Integrate with CI/CD pipeline