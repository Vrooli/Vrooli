# Scenario Testing Implementation

This document describes the implementation of multi-step scenario testing in Vrooli using Vitest workspaces.

## Overview

Scenario testing validates complex user workflows that involve multiple operations:
```
User Registration → Project Creation → Team Invitation → Resource Bookmarking → etc.
```

## Architecture

We use **Vitest workspaces** to run scenario tests in the appropriate environment:

- **UI Tests** (`ui` project): Run in `happy-dom` environment for React components
- **Round-Trip Tests** (`ui-roundtrip` project): Run in `node` environment for single operations
- **Scenario Tests** (`ui-scenarios` project): Run in `node` environment for multi-step workflows
- **Server Tests** (`server` project): Run in `node` environment
- **Jobs Tests** (`jobs` project): Run in `node` environment
- **Shared Tests** (`shared` project): Run in `node` environment

## Key Differences

### Round-Trip Tests vs Scenario Tests

| Aspect | Round-Trip Tests | Scenario Tests |
|--------|------------------|----------------|
| **Purpose** | Test single operation through all layers | Test multi-step user workflows |
| **Scope** | One atomic operation | Multiple coordinated operations |
| **Example** | Create bookmark | User signs up → creates project → invites team |
| **File Pattern** | `*.roundtrip.test.ts` | `*.scenario.test.ts` |
| **Orchestrators** | Not needed | Uses scenario orchestrator classes |

## File Organization

```
packages/ui/src/
├── __test/
│   ├── fixtures/
│   │   ├── scenarios/          # Scenario orchestrators
│   │   │   ├── README.md
│   │   │   ├── userBookmarksProject.scenario.ts      # Orchestrator class
│   │   │   └── userBookmarksProject.scenario.test.ts # Actual test
│   │   └── examples/
│   │       └── true-round-trip.roundtrip.test.ts
│   └── roundtrip/             # Round-trip tests (suggested)
│       └── bookmark.roundtrip.test.ts
└── views/
    └── objects/
        └── bookmark/
            ├── bookmark.roundtrip.test.ts    # Co-located round-trip test
            └── bookmark.scenario.test.ts     # Co-located scenario test
```

## File Naming Requirements

- **Scenario Tests**: MUST use `*.scenario.test.ts` or `*.scenario.test.tsx`
- **Scenario Orchestrators**: Use `*.scenario.ts` (without `.test`)
- **Round-Trip Tests**: MUST use `*.roundtrip.test.ts` or `*.roundtrip.test.tsx`

## Available Commands

### Package-Level Commands (in `packages/ui/`)
```bash
# Run all UI tests (regular + round-trip + scenarios)
pnpm test

# Run only regular UI tests
pnpm test:ui

# Run only round-trip tests
pnpm test:roundtrip

# Run only scenario tests
pnpm test:scenarios

# Watch mode for all tests
pnpm test-watch

# Watch mode for specific test type
pnpm test-watch:ui
pnpm test-watch:roundtrip
pnpm test-watch:scenarios

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
pnpm test:unit:ui-scenarios # Scenario tests only
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
npx vitest --project ui-scenarios
npx vitest --project ui-roundtrip
npx vitest --project ui

# Run specific file pattern
npx vitest userBookmarksProject.scenario.test
```

## Writing Scenario Tests

### 1. Create Scenario Orchestrator

```typescript
// userWorkflow.scenario.ts
export class UserWorkflowScenario {
    constructor(
        private integrationTests: Record<string, IntegrationTest>,
        private fixtureFactories: Record<string, FixtureFactory>
    ) {}
    
    async execute(config: ScenarioConfig): Promise<ScenarioResult> {
        const steps = [
            { name: "create_user", action: "create", objectType: "user" },
            { name: "create_project", action: "create", objectType: "project" },
            { name: "invite_members", action: "create", objectType: "invite" }
        ];
        
        // Execute steps with dependency management
        return await this.orchestrator.execute({ steps });
    }
}
```

### 2. Create Scenario Test

```typescript
// userWorkflow.scenario.test.ts
import { describe, it, expect } from 'vitest';
import { UserWorkflowScenario } from './userWorkflow.scenario.js';

describe('User Workflow Scenario', () => {
    it('should complete full user journey', async () => {
        const scenario = new UserWorkflowScenario(
            integrationTests,
            fixtureFactories
        );
        
        const result = await scenario.execute({
            user: { name: "Test User", email: "test@example.com" },
            project: { name: "Test Project" },
            invites: ["member1@example.com", "member2@example.com"]
        });
        
        expect(result.success).toBe(true);
        expect(result.stepResults).toHaveLength(3);
    });
});
```

## Scenario Test Features

### Step Dependencies
```typescript
{
    name: "create_project",
    dependencies: ["create_user"], // Won't run until user is created
    data: { ownerId: "{{create_user.id}}" } // Template resolution
}
```

### Assertions
```typescript
{
    name: "verify_bookmarks",
    assertions: [
        { type: "exists", target: "bookmark.id", expected: true },
        { type: "count", target: "bookmarks", expected: 3 },
        { type: "equals", target: "user.name", expected: "Test User" }
    ]
}
```

### Error Handling
```typescript
{
    name: "risky_operation",
    optional: true, // Won't fail scenario if this step fails
    retries: 3,     // Retry up to 3 times
    timeout: 5000   // Custom timeout
}
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
- Server components can be imported directly

## Common Scenario Types

1. **Authentication Flows**
   - Signup → Email Verification → Login → Password Reset

2. **Team Collaboration**
   - Create Team → Invite Members → Assign Roles → Share Resources

3. **Content Management**
   - Create Content → Add Tags → Publish → Share → Bookmark

4. **Routine Execution**
   - Create Routine → Configure Steps → Start Run → Monitor Progress → Complete

5. **Payment Flows**
   - Trial User → Hit Limits → View Plans → Subscribe → Access Features

## Best Practices

### DO's ✅
- Use scenario tests for multi-step workflows
- Use round-trip tests for single operations
- Include both happy path and error scenarios
- Test real user journeys
- Use meaningful step names
- Implement proper cleanup

### DON'Ts ❌
- Don't use scenarios for simple CRUD operations
- Don't mock the functions being tested
- Don't create overly complex scenarios
- Don't skip error handling tests
- Don't hardcode test data

## Troubleshooting

### "No projects matched filter"
- Run from package directory: `cd packages/ui`
- Use correct project name: `ui-scenarios` not `scenarios`

### Import Errors
- Ensure server is built: `cd packages/server && pnpm build`
- Check import paths use `.js` extension

### Test Timeouts
- Scenario tests have 120s timeout by default
- Can be increased in specific tests if needed

### Environment Issues
- Scenarios must use `*.scenario.test.ts` naming
- Regular UI tests cannot import from `@vrooli/server`

## Migration Guide

To convert integration tests to scenarios:

1. Create orchestrator class in `*.scenario.ts`
2. Move test logic to `*.scenario.test.ts`
3. Update imports to use `.js` extensions
4. Ensure proper naming convention
5. Run with `pnpm test:scenarios`

## Future Improvements

- [ ] Visual scenario builder/editor
- [ ] Scenario recording from user sessions
- [ ] Parallel scenario execution
- [ ] Scenario performance benchmarks
- [ ] Integration with CI/CD pipelines