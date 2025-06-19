# Fixture Implementation Guide

This document provides step-by-step guidance for implementing new fixtures and maintaining existing ones in the Vrooli testing ecosystem.

**Purpose**: Learn how to add new fixtures and maintain the fixture system.

**Prerequisites**: 
- Read [Fixtures Overview](./fixtures-overview.md) first
- Understand [Fixture Patterns](./fixture-patterns.md)

**Related Documents**:
- [Fixtures Overview](./fixtures-overview.md) - Quick start guide
- [Fixture Patterns](./fixture-patterns.md) - Pattern catalog
- [Round-Trip Testing](./round-trip-testing.md) - Integration testing
- [Fixture Reference](./fixture-reference.md) - Complete API reference

## Implementation Status

### ✅ Complete
- **API Fixtures**: 40/41 object types have fixtures
- **Config Fixtures**: All configuration objects covered
- **Database Fixtures**: Full coverage in server package

### 🚧 In Progress
- **Permission Fixtures**: New permission/auth fixtures being added
- **UI Component Fixtures**: MSW handlers for all endpoints
- **Test Helpers**: Utility functions for common operations

### 📋 High Priority (Most Used Features)
1. Comment system fixtures (used everywhere)
2. Project/Routine fixtures (core functionality)
3. User/Team fixtures (authentication/authorization)
4. Chat fixtures (real-time features)

### 🔧 Missing Infrastructure
- `useCommenter` hook (follow `useBookmarker` pattern)
- `useReporter` hook (for report functionality)
- `useSharer` hook (for sharing features)
- Error state fixtures (network, validation, API errors)
- Real-time event fixtures (socket, chat, swarm events)

## Directory Organization

```
packages/
├── server/src/__test/fixtures/          # 🗄️  DATABASE FIXTURES
│   ├── db/                              # Already well-organized
│   │   ├── userFixtures.ts              # ✅ Already exists
│   │   ├── projectFixtures.ts           # ✅ Already exists  
│   │   └── [39 more object types]       # ✅ Comprehensive coverage
│   │
│   ├── permissions/                     # 🔐 PERMISSION & AUTH FIXTURES (NEW!)
│   │   ├── userPersonas.ts              # ✅ Standard user types
│   │   ├── apiKeyPermissions.ts         # ✅ API key configurations
│   │   ├── teamScenarios.ts             # ✅ Team membership scenarios
│   │   ├── edgeCases.ts                 # ✅ Edge cases & stress tests
│   │   ├── integrationScenarios.ts      # ✅ Complex multi-actor tests
│   │   ├── sessionHelpers.ts            # ✅ Test session utilities
│   │   ├── example.test.ts              # ✅ Usage examples
│   │   └── index.ts                     # ✅ Central exports
│   │
│   └── execution/                       # 🤖 AI EXECUTION FIXTURES
│       ├── tier1-coordination/          # Swarms, MOISE+ orgs
│       ├── tier2-process/               # Routines, navigators
│       ├── tier3-execution/             # Strategies, executors
│       ├── emergent-capabilities/       # Agent types, evolution
│       └── integration-scenarios/       # Complete examples
│
├── shared/src/__test/fixtures/          # 🔗 SHARED FIXTURES
│   ├── api/                             # API request/response
│   │   ├── userFixtures.ts              # ✅ Already exists
│   │   ├── projectFixtures.ts           # ✅ Already exists
│   │   └── [38 more object types]       # ✅ Near-complete
│   │
│   ├── config/                          # Configuration objects
│   │   ├── botConfigFixtures.ts         # ✅ Bot settings
│   │   ├── chatConfigFixtures.ts        # ✅ Chat settings
│   │   └── [other config types]         # ✅ Complete coverage
│   │
│   ├── errors/                          # 🚨 ERROR STATE FIXTURES (NEW!)
│   │   ├── index.ts                     # Central exports
│   │   ├── apiErrors.ts                 # HTTP error responses
│   │   ├── validationErrors.ts          # Field validation errors
│   │   ├── networkErrors.ts             # Connection/timeout errors
│   │   ├── authErrors.ts                # Auth/permission errors
│   │   ├── businessErrors.ts            # Business logic errors
│   │   └── systemErrors.ts              # Infrastructure errors
│   │
│   └── events/                          # 📡 REAL-TIME EVENT FIXTURES (NEW!)
│       ├── index.ts                     # Central exports
│       ├── socketEvents.ts              # Base socket events
│       ├── chatEvents.ts                # Chat messaging events
│       ├── swarmEvents.ts               # AI execution events
│       ├── notificationEvents.ts        # Push notifications
│       ├── collaborationEvents.ts       # Multi-user events
│       └── systemEvents.ts              # System status events
│
└── ui/src/__test/                      # 🎨 UI TESTING INFRASTRUCTURE
    ├── fixtures/                        # UI-specific test fixtures
    │   ├── api-responses/               # Mock API response data
    │   │   └── bookmarkResponses.ts     # Example response fixtures
    │   ├── form-data/                   # Form input test data
    │   │   └── bookmarkFormData.ts      # Example form fixtures
    │   ├── helpers/                     # Transformation utilities
    │   │   └── bookmarkTransformations.ts
    │   ├── round-trip-tests/            # End-to-end test examples
    │   │   └── bookmarkRoundTrip.test.ts
    │   ├── sessions/                    # Session fixtures (placeholder)
    │   ├── ui-states/                   # UI state fixtures (placeholder)
    │   └── index.ts                     # Central exports
    │
    └── helpers/                         # Test utilities
        ├── storybookDecorators.ts       # Storybook decorators
        ├── storybookMocking.ts          # MSW integration for Storybook
        └── testUtils.tsx                # Common test utilities
```

## Adding New Fixtures - Step by Step

### Step 1: Analyze the Object Type

First, understand the object's structure and relationships:

```typescript
// 1. Check the Shape definition
import { Shape } from "@vrooli/shared";
type MyObject = Shape.MyObject;

// 2. Check validation schema
import { myObjectValidation } from "@vrooli/shared";

// 3. Check endpoints
import { endpointsMyObject } from "@vrooli/shared";

// 4. Identify relationships
// Does it belong to a user? Team? Project?
// What other objects reference it?
```

### Step 2: Create Basic Fixtures

Start with minimal and complete variants:

```typescript
// packages/shared/src/__test/fixtures/api/myObjectFixtures.ts

import { generatePK } from "@vrooli/shared";
import type { Shape } from "@vrooli/shared";

export const myObjectFixtures = {
    // Minimal valid object
    minimal: {
        create: {
            name: "Test MyObject",
            description: "Minimal test object",
        } satisfies Partial<Shape.MyObjectCreateInput>,
        
        find: {
            id: generatePK(),
            name: "Test MyObject", 
            description: "Minimal test object",
            created_at: "2024-01-01T00:00:00Z",
            updated_at: "2024-01-01T00:00:00Z",
        } satisfies Shape.MyObject,
    },
    
    // Complete object with all fields
    complete: {
        create: {
            name: "Complete MyObject",
            description: "Full-featured test object",
            isPrivate: false,
            tags: ["test", "fixture"],
            customField: "custom value",
            // Add all optional fields
        } satisfies Shape.MyObjectCreateInput,
        
        find: {
            id: generatePK(),
            name: "Complete MyObject",
            description: "Full-featured test object", 
            isPrivate: false,
            tags: [
                { id: "tag_1", name: "test" },
                { id: "tag_2", name: "fixture" },
            ],
            customField: "custom value",
            stats: {
                views: 100,
                likes: 10,
            },
            owner: {
                id: "user_123",
                name: "Test User",
            },
            created_at: "2024-01-01T00:00:00Z",
            updated_at: "2024-01-01T00:00:00Z",
        } satisfies Shape.MyObject,
    },
};
```

### Step 3: Add Scenario-Based Fixtures

Create fixtures for common use cases:

```typescript
export const myObjectScenarios = {
    // Permission scenarios
    privateOwned: {
        ...myObjectFixtures.complete.find,
        isPrivate: true,
        owner: { id: "owner_123", name: "Owner" },
    },
    
    publicShared: {
        ...myObjectFixtures.complete.find,
        isPrivate: false,
        team: { id: "team_123", name: "Public Team" },
    },
    
    // State scenarios  
    draft: {
        ...myObjectFixtures.minimal.find,
        status: "draft",
        isPublished: false,
    },
    
    published: {
        ...myObjectFixtures.complete.find,
        status: "published",
        isPublished: true,
        publishedAt: "2024-01-01T00:00:00Z",
    },
    
    // Relationship scenarios
    withComments: {
        ...myObjectFixtures.complete.find,
        comments: [
            { id: "comment_1", text: "Great work!" },
            { id: "comment_2", text: "Needs improvement" },
        ],
        commentsCount: 2,
    },
};
```

### Step 4: Create Factory Functions

For dynamic or unique data:

```typescript
// packages/shared/src/__test/fixtures/api/myObjectFactories.ts

let counter = 0;

export function createMyObjectFixture(
    overrides: Partial<Shape.MyObject> = {}
): Shape.MyObject {
    counter++;
    return {
        ...myObjectFixtures.minimal.find,
        id: `myobject_${counter}`,
        name: `MyObject ${counter}`,
        ...overrides,
    };
}

export function createMyObjectWithRelations(config: {
    owner?: Shape.User;
    team?: Shape.Team;
    tags?: string[];
}): Shape.MyObject {
    return {
        ...myObjectFixtures.complete.find,
        owner: config.owner || createUserFixture(),
        team: config.team,
        tags: config.tags?.map(tag => ({
            id: generatePK(),
            name: tag,
        })),
    };
}
```

### Step 5: Add to Index Exports

Make fixtures easily importable:

```typescript
// packages/shared/src/__test/fixtures/api/index.ts

export * from './myObjectFixtures.js';
export * from './myObjectFactories.js';

// Add to namespace export
export const apiFixtures = {
    // ... existing fixtures
    myObjectFixtures,
    myObjectScenarios,
};
```

### Step 6: Create Database Fixtures (Server)

For server-side testing with real database:

```typescript
// packages/server/src/__test/fixtures/db/myObjectFixtures.ts

import { DbProvider } from "../../../services/index.js";
import { myObjectFixtures } from "@vrooli/shared/__test/fixtures";

export async function createTestMyObject(
    data: Partial<Shape.MyObjectCreateInput> = {},
    owner?: { id: string }
) {
    const db = DbProvider.get();
    
    return db.myObject.create({
        data: {
            ...myObjectFixtures.minimal.create,
            ...data,
            owner: owner ? { connect: { id: owner.id } } : undefined,
        },
        include: {
            owner: true,
            tags: true,
            stats: true,
        },
    });
}

export async function createMyObjectWithRelations() {
    const owner = await createTestUser();
    const team = await createTestTeam({ owner });
    
    return createTestMyObject({
        name: "Object with relations",
        team: { connect: { id: team.id } },
    }, owner);
}
```

### Step 7: Create Test Helpers

Common operations for testing:

```typescript
// packages/server/src/__test/helpers/myObjectHelpers.ts

export async function cleanupMyObjects(ids: string[]) {
    const db = DbProvider.get();
    
    // Clean up in correct order for foreign keys
    await db.comment.deleteMany({
        where: { parent: { id: { in: ids } } }
    });
    
    await db.myObject.deleteMany({
        where: { id: { in: ids } }
    });
}

export async function verifyMyObjectState(
    id: string,
    expected: Partial<Shape.MyObject>
) {
    const db = DbProvider.get();
    const actual = await db.myObject.findUnique({
        where: { id },
        include: { owner: true, tags: true },
    });
    
    expect(actual).toMatchObject(expected);
    return actual;
}
```

### Step 8: Write Example Tests

Show how to use the fixtures:

```typescript
// packages/server/src/__test/fixtures/example.test.ts

describe("MyObject Fixture Examples", () => {
    it("should use minimal fixture", async () => {
        const data = myObjectFixtures.minimal.create;
        const result = await endpointsMyObject.create({
            input: data,
            context: createTestContext(),
        });
        expect(result.success).toBe(true);
    });
    
    it("should use factory for unique data", async () => {
        const objects = Array.from({ length: 3 }, () => 
            createMyObjectFixture()
        );
        
        // Each has unique ID
        const ids = objects.map(o => o.id);
        expect(new Set(ids).size).toBe(3);
    });
    
    it("should test with relationships", async () => {
        const withRelations = await createMyObjectWithRelations();
        expect(withRelations.owner).toBeDefined();
        expect(withRelations.team).toBeDefined();
    });
});
```

## Fixture Maintenance

### Versioning Strategy

When schemas change, maintain backward compatibility:

```typescript
export const myObjectFixtures = {
    v1: {
        // Original fixture structure
    },
    v2: {
        // Updated structure with new fields
    },
    // Current version (default)
    minimal: { /* latest */ },
    complete: { /* latest */ },
};
```

### Update Process

1. **Schema Changes**: Update Shape types first
2. **Validation Updates**: Ensure validation matches
3. **Fixture Updates**: Update fixtures to match
4. **Test Updates**: Fix broken tests
5. **Migration Guide**: Document changes

### Automated Maintenance

Create a fixture validation script:

```typescript
// scripts/validateFixtures.ts

import { Shape, Validation } from "@vrooli/shared";
import { apiFixtures } from "@vrooli/shared/__test/fixtures";

async function validateAllFixtures() {
    const errors = [];
    
    // Test each fixture type
    for (const [name, fixtures] of Object.entries(apiFixtures)) {
        try {
            // Validate create inputs
            if (fixtures.minimal?.create) {
                await Validation[name].create.validate(
                    fixtures.minimal.create
                );
            }
            
            // Type check find outputs
            const _typeCheck: Shape[name] = fixtures.minimal?.find;
            
        } catch (error) {
            errors.push({ fixture: name, error });
        }
    }
    
    return errors;
}
```

## Implementation Roadmap

### Phase 1: Core Infrastructure (⚡ Immediate)
1. **Complete missing action hooks**
   - Create `useCommenter` hook
   - Create `useReporter` hook  
   - Create `useSharer` hook

2. **Implement error state fixtures**
   - Create error fixture directories
   - Add API error fixtures (400, 401, 403, 404, 429, 500)
   - Add network error fixtures (timeout, offline, connection)
   - Add validation error fixtures with field details

3. **Implement real-time event fixtures**
   - Create event fixture directories
   - Add socket connection events
   - Add chat messaging events
   - Add swarm execution events
   - Create MockSocketEmitter utility

4. **Validate existing fixtures**
   - Run validation script
   - Fix any schema mismatches
   - Update outdated fixtures

### Phase 2: Permission Fixtures (🕐 Next Week)
1. **Create permission personas**
   - Standard user types (admin, member, guest)
   - API key configurations
   - Team role scenarios

2. **Add edge cases**
   - Suspended users
   - Expired sessions
   - Rate-limited requests

### Phase 3: UI Testing Fixtures (🚀 Next Sprint)
1. **MSW handler generation**
   - Create handlers for all endpoints
   - Add error scenarios
   - Include loading states

2. **Component fixtures**
   - Form initial values
   - List display data
   - Navigation states

### Phase 4: Automation (📈 Future)
1. **Fixture generation CLI**
   ```bash
   pnpm generate:fixture --type=api --object=MyObject
   ```

2. **Automatic validation**
   - Pre-commit hooks
   - CI validation
   - Schema sync

## Best Practices

### DO's ✅
- Keep fixtures minimal but valid
- Use TypeScript for type safety
- Document fixture purposes
- Test fixtures themselves
- Version fixtures when schemas change

### DON'Ts ❌
- Don't hardcode IDs (use generatePK)
- Don't include timestamps (unless testing time)
- Don't create circular dependencies
- Don't modify shared fixtures
- Don't skip cleanup in tests

## Troubleshooting

### Common Issues

**Fixture validation fails**
- Check recent schema changes
- Ensure all required fields present
- Validate against correct version

**Type errors with fixtures**
- Update Shape type imports
- Regenerate types with `pnpm prisma generate`
- Check for version mismatches

**Tests using fixtures fail**
- Verify fixture data is valid
- Check relationship IDs exist
- Ensure proper cleanup

## Next Steps

1. **Audit current fixtures** for completeness
2. **Add missing fixtures** for your feature area
3. **Convert tests** to use shared fixtures
4. **Document** any special fixture needs

For questions or improvements, see the [main testing documentation](./README.md).