# Fixture Patterns - Best Practices and Examples

This document provides detailed patterns and best practices for creating and using fixtures in the Vrooli testing ecosystem.

**Purpose**: Learn advanced fixture patterns and when to use each approach.

**Prerequisites**: Read [Fixtures Overview](./fixtures-overview.md) first.

**Related Documents**:
- [Fixtures Overview](./fixtures-overview.md) - Quick start guide
- [Round-Trip Testing](./round-trip-testing.md) - End-to-end validation
- [Fixture Implementation Guide](./fixture-implementation-guide.md) - Adding new fixtures
- [Fixture Reference](./fixture-reference.md) - Complete API reference

## Architecture Overview

Vrooli's architecture provides exceptional consistency across **41 object types** with well-organized patterns at every layer:

### Shape Objects (41 objects)
Located in `/packages/shared/src/shape/models/models.ts`

**Purpose**: Transform data between different representations (API ↔ UI ↔ DB)

```typescript
// Every object has consistent shape functions
export const shapeBookmark = {
    create: (data) => transformForCreate(data),
    update: (data) => transformForUpdate(data),
    find: (data) => transformForFind(data),
};
```

### API Endpoints (47 endpoint groups)
Located in `/packages/shared/src/api/pairs.ts`

**Purpose**: Define available operations for each object type

```typescript
// Consistent endpoint structure
export const endpointsBookmark = {
    create: { path: "/bookmark", method: "POST" },
    update: { path: "/bookmark", method: "PUT" },
    delete: { path: "/bookmark", method: "DELETE" },
    find: { path: "/bookmark", method: "GET" },
};
```

### Validation Schemas (40 models)
Located in `/packages/shared/src/validation/models/`

**Purpose**: Ensure data integrity at every layer

```typescript
// Consistent validation patterns
export const bookmarkValidation = {
    create: yup.object().shape({
        bookmarkFor: yup.string().required(),
        forConnect: yup.string().required(),
    }),
};
```

## Pattern Catalog

### Pattern 1: Minimal Fixtures
**When to use**: Unit tests, quick validation, performance-critical tests

```typescript
export const userMinimal = {
    id: generatePK(),
    email: "test@example.com",
    name: "Test User",
};

// Usage
it("should validate user email", () => {
    const user = userFixtures.minimal.create;
    expect(isValidEmail(user.email)).toBe(true);
});
```

**Pros**: Fast, focused, easy to understand
**Cons**: May not catch integration issues

### Pattern 2: Complete Fixtures
**When to use**: Integration tests, UI testing, complex workflows

```typescript
export const userComplete = {
    ...userMinimal,
    bio: "Test bio",
    profile: profileFixtures.complete,
    settings: userSettingsFixtures.default,
    teams: [teamFixtures.minimal],
    created_at: new Date().toISOString(),
};

// Usage
it("should display complete user profile", () => {
    const user = userFixtures.complete.create;
    render(<UserProfile user={user} />);
    expect(screen.getByText(user.bio)).toBeInTheDocument();
});
```

**Pros**: Realistic data, catches edge cases
**Cons**: More setup time, larger memory footprint

### Pattern 3: Scenario-Based Fixtures
**When to use**: Testing specific user flows, permission testing, state transitions

```typescript
export const userScenarios = {
    admin: { ...userComplete, roles: ["admin"] },
    guest: { ...userMinimal, isGuest: true },
    premium: { ...userComplete, premium: premiumFixtures.active },
    suspended: { ...userComplete, status: "suspended" },
};

// Usage
describe("Permission Tests", () => {
    it("admin can delete any comment", () => {
        const admin = userFixtures.admin.create;
        const result = checkPermission(admin, "comment.delete");
        expect(result).toBe(true);
    });
});
```

**Pros**: Clear intent, reusable scenarios
**Cons**: Can proliferate quickly

### Pattern 4: Builder Pattern
**When to use**: Need flexible fixture creation with overrides

```typescript
class UserFixtureBuilder {
    private user = { ...userMinimal };
    
    withTeams(teams: Team[]) {
        this.user.teams = teams;
        return this;
    }
    
    withPremium() {
        this.user.premium = premiumFixtures.active;
        return this;
    }
    
    build() {
        return this.user;
    }
}

// Usage
const premiumTeamOwner = new UserFixtureBuilder()
    .withTeams([teamFixtures.complete])
    .withPremium()
    .build();
```

**Pros**: Flexible, readable, composable
**Cons**: More code, potential over-engineering

### Pattern 5: Factory Functions
**When to use**: Need dynamic data, unique values, or sequences

```typescript
let userCounter = 0;
export function createUser(overrides = {}) {
    userCounter++;
    return {
        id: `user_${userCounter}`,
        email: `user${userCounter}@example.com`,
        name: `Test User ${userCounter}`,
        ...overrides,
    };
}

// Usage
it("should handle multiple users", () => {
    const users = Array.from({ length: 5 }, () => createUser());
    expect(users[0].email).not.toBe(users[1].email);
});
```

**Pros**: Unique data, avoids conflicts
**Cons**: Less predictable, harder to debug

## Leveraging Existing Architecture

### Don't Duplicate - Integrate!

Instead of creating mock implementations, use the real functions:

```typescript
// ❌ WRONG: Creating duplicate logic
function mockTransformUser(data) {
    return {
        id: data.id,
        fullName: `${data.firstName} ${data.lastName}`,
        // Duplicating logic that exists in shapeUser
    };
}

// ✅ CORRECT: Use existing shape functions
import { shapeUser } from "@vrooli/shared";

const transformed = shapeUser.create(userData);
```

### Real Function Integration

```typescript
import { shapeComment, commentValidation, endpointsComment } from "@vrooli/shared";

describe("Comment Creation Flow", () => {
    it("should validate and transform comment data", async () => {
        // Start with fixture
        const commentData = commentFixtures.minimal.create;
        
        // Use real validation
        const validationResult = await commentValidation.create.validate(commentData);
        expect(validationResult.isValid).toBe(true);
        
        // Use real transformation
        const transformed = shapeComment.create(commentData);
        expect(transformed).toHaveProperty("__typename", "Comment");
        
        // Would use real endpoint in integration test
        // const result = await endpointsComment.create({ input: transformed });
    });
});
```

## Complete Example: Project Fixture Pattern

Here's a complete example showing all layers working together:

```typescript
// 1. Define fixtures following existing patterns
export const projectFixtures = {
    minimal: {
        create: {
            name: "Test Project",
            description: "Minimal test project",
            isPrivate: false,
        },
        find: {
            id: "proj_123",
            name: "Test Project",
            description: "Minimal test project",
            isPrivate: false,
            created_at: new Date().toISOString(),
        },
    },
    complete: {
        create: {
            name: "Complete Project",
            description: "Full-featured test project",
            isPrivate: false,
            tags: ["test", "fixture"],
            team: { id: "team_123" },
        },
        find: {
            id: "proj_456",
            name: "Complete Project",
            description: "Full-featured test project",
            isPrivate: false,
            tags: tagFixtures.array(2),
            team: teamFixtures.minimal.find,
            versions: [projectVersionFixtures.published],
            stats: statsProjectFixtures.default,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
        },
    },
};

// 2. Test using real functions
describe("Project Management", () => {
    it("should create project with proper validation", async () => {
        // Use fixture as base
        const projectData = projectFixtures.complete.create;
        
        // Validate with real schema
        const validation = await projectValidation.create.validate(projectData);
        expect(validation.isValid).toBe(true);
        
        // Transform with real shape function
        const shaped = shapeProject.create({
            ...projectData,
            __typename: "Project",
            id: DUMMY_ID,
        });
        
        // In real test, would call actual endpoint
        expect(shaped).toMatchObject({
            name: projectData.name,
            description: projectData.description,
        });
    });
    
    it("should handle project with relationships", () => {
        // Use complete fixture with relationships
        const project = projectFixtures.complete.find;
        
        // Test relationship handling
        expect(project.team).toBeDefined();
        expect(project.versions).toHaveLength(1);
        expect(project.stats).toBeDefined();
    });
});

// 3. Use in UI tests with MSW
import { server } from "../mocks/server.js";
import { rest } from "msw";

describe("Project UI", () => {
    it("should display project details", async () => {
        // Mock API response with fixture
        server.use(
            rest.get("/api/project/:id", (req, res, ctx) => {
                return res(ctx.json({
                    success: true,
                    data: projectFixtures.complete.find,
                }));
            })
        );
        
        // Render and test
        render(<ProjectView projectId="proj_456" />);
        await waitFor(() => {
            expect(screen.getByText("Complete Project")).toBeInTheDocument();
        });
    });
});
```

## Performance Considerations

### Fixture Loading Strategies

```typescript
// Lazy loading for large fixtures
const getLargeFixture = () => {
    if (!_cachedFixture) {
        _cachedFixture = require("./largeFixture.json");
    }
    return _cachedFixture;
};

// Fixture pooling for unique data
class FixturePool {
    private used = new Set();
    private available = [];
    
    get() {
        if (this.available.length === 0) {
            this.available.push(createUser());
        }
        const fixture = this.available.pop();
        this.used.add(fixture);
        return fixture;
    }
    
    release(fixture) {
        this.used.delete(fixture);
        this.available.push(fixture);
    }
}
```

### Memory Management

```typescript
describe("Large Dataset Tests", () => {
    let fixtures;
    
    beforeAll(() => {
        // Load fixtures once for suite
        fixtures = loadLargeFixtures();
    });
    
    afterAll(() => {
        // Clean up to free memory
        fixtures = null;
    });
    
    // Reuse fixtures across tests
    it("test 1", () => {
        const data = fixtures.subset1;
        // Test logic
    });
});
```

## Anti-Patterns to Avoid

### 1. Fixture Mutation
```typescript
// ❌ WRONG: Mutating shared fixtures
const user = userFixtures.complete;
user.email = "new@example.com"; // Affects all tests!

// ✅ CORRECT: Create a copy
const user = { ...userFixtures.complete, email: "new@example.com" };
```

### 2. Overly Complex Fixtures
```typescript
// ❌ WRONG: Kitchen sink fixture
const everything = {
    user: userFixtures.complete,
    team: teamFixtures.complete,
    projects: Array(10).fill(projectFixtures.complete),
    // ... 20 more nested objects
};

// ✅ CORRECT: Minimal required data
const testData = {
    user: userFixtures.minimal,
    project: projectFixtures.minimal,
};
```

### 3. Random Data in Fixtures
```typescript
// ❌ WRONG: Non-deterministic fixtures
const badFixture = {
    id: Math.random().toString(),
    createdAt: new Date(), // Changes every run
};

// ✅ CORRECT: Predictable data
const goodFixture = {
    id: "test_123",
    createdAt: "2024-01-01T00:00:00Z",
};
```

### Pattern 6: Error State Fixtures
**When to use**: Testing error handling, user feedback, recovery flows

```typescript
// packages/shared/src/__test/fixtures/errors/apiErrors.ts
export const apiErrorFixtures = {
  // Standard HTTP errors
  badRequest: {
    minimal: {
      status: 400,
      code: "BAD_REQUEST",
      message: "Invalid request parameters"
    },
    withDetails: {
      status: 400,
      code: "BAD_REQUEST",
      message: "Invalid request parameters",
      details: {
        fields: {
          email: "Invalid email format",
          password: "Password too short"
        }
      }
    }
  },

  // Rate limiting
  rateLimit: {
    standard: {
      status: 429,
      code: "RATE_LIMIT_EXCEEDED",
      message: "Too many requests",
      retryAfter: 60,
      limit: 100,
      remaining: 0,
      reset: new Date(Date.now() + 60000).toISOString()
    }
  },

  // Factory functions
  factories: {
    createValidationError: (fields: Record<string, string>) => ({
      status: 400,
      code: "VALIDATION_ERROR",
      message: "Validation failed",
      details: { fields }
    })
  }
};

// Usage
it("should handle validation errors", () => {
  const error = apiErrorFixtures.badRequest.withDetails;
  render(<Form error={error} />);
  expect(screen.getByText("Invalid email format")).toBeInTheDocument();
});
```

**Pros**: Consistent error handling, better UX testing
**Cons**: Need to maintain error catalog

### Pattern 7: Real-time Event Fixtures
**When to use**: Testing WebSocket events, live updates, collaborative features

```typescript
// packages/shared/src/__test/fixtures/events/chatEvents.ts
export const chatEventFixtures = {
  messages: {
    textMessage: {
      event: "chat:message",
      data: {
        id: "msg_123",
        chatId: "chat_456",
        userId: "user_789",
        content: "Hello world",
        timestamp: new Date().toISOString(),
        status: "sent"
      }
    }
  },

  // Event sequences for testing flows
  sequences: {
    typingFlow: [
      { event: "chat:typing", data: { isTyping: true } },
      { delay: 2000 },
      { event: "chat:message", data: { content: "..." } },
      { event: "chat:typing", data: { isTyping: false } }
    ]
  }
};

// Usage with mock emitter
const mockSocket = new MockSocketEmitter();
await mockSocket.emitSequence(chatEventFixtures.sequences.typingFlow);
```

**Pros**: Test complex real-time flows, predictable event timing
**Cons**: Requires mock infrastructure

## Error State Fixtures Organization

Error fixtures provide consistent error scenarios for testing error handling across the application.

### Directory Structure
```
packages/shared/src/__test/fixtures/errors/
├── index.ts                    # Central export point
├── apiErrors.ts               # API-specific errors
├── validationErrors.ts        # Field validation errors
├── networkErrors.ts           # Network/connection errors
├── authErrors.ts              # Authentication/authorization
├── businessErrors.ts          # Business logic errors
└── systemErrors.ts            # System/infrastructure errors
```

### Implementation Example

```typescript
// packages/shared/src/__test/fixtures/errors/networkErrors.ts
export const networkErrorFixtures = {
  timeout: {
    client: new Error("Request timeout after 30000ms"),
    server: {
      status: 504,
      code: "GATEWAY_TIMEOUT",
      message: "Request timed out"
    }
  },

  connectionRefused: {
    error: new Error("ECONNREFUSED"),
    display: {
      title: "Connection Failed",
      message: "Unable to connect to server",
      retry: true
    }
  },

  networkOffline: {
    error: new Error("Network request failed"),
    display: {
      title: "You're Offline",
      message: "Check your internet connection",
      icon: "wifi_off"
    }
  }
};
```

## Real-time Event Fixtures Organization

Event fixtures simulate WebSocket and real-time events for testing live features.

### Directory Structure
```
packages/shared/src/__test/fixtures/events/
├── index.ts
├── socketEvents.ts            # Base socket event types
├── chatEvents.ts             # Chat-specific events
├── swarmEvents.ts            # AI swarm execution events
├── notificationEvents.ts     # Push notifications
├── collaborationEvents.ts    # Multi-user collaboration
└── systemEvents.ts           # System status updates
```

### Swarm Event Example

```typescript
// packages/shared/src/__test/fixtures/events/swarmEvents.ts
export const swarmEventFixtures = {
  execution: {
    started: {
      event: "swarm:execution:started",
      data: {
        executionId: "exec_123",
        routineId: "routine_456",
        agents: ["agent_1", "agent_2"],
        estimatedDuration: 30000
      }
    },
    
    progress: {
      event: "swarm:execution:progress",
      data: {
        executionId: "exec_123",
        progress: 0.45,
        currentStep: "data_processing",
        message: "Processing input data...",
        metrics: {
          tokensUsed: 1234,
          computeTime: 5678
        }
      }
    }
  }
};
```

### Test Utilities

```typescript
// packages/ui/src/__test/fixtures/events/mockEventEmitters.ts
export class MockSocketEmitter {
  private handlers: Map<string, Set<Function>> = new Map();
  private emitHistory: Array<{event: string, data: any, timestamp: number}> = [];

  on(event: string, handler: Function) {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
  }

  emit(event: string, data: any) {
    this.emitHistory.push({ event, data, timestamp: Date.now() });
    const handlers = this.handlers.get(event);
    if (handlers) {
      handlers.forEach(handler => handler(data));
    }
  }

  // Test helpers
  async emitSequence(sequence: EventSequence[]) {
    for (const item of sequence) {
      if ('delay' in item) {
        await new Promise(resolve => setTimeout(resolve, item.delay));
      } else {
        this.emit(item.event, item.data);
      }
    }
  }
}
```

## Next Steps

1. **Review existing fixtures** in your test suite
2. **Identify duplication** that could use shared fixtures
3. **Apply patterns** that match your testing needs
4. **Contribute new fixtures** following these patterns
5. **Implement error and event fixtures** for comprehensive coverage

For implementation details, see the [Fixture Implementation Guide](./fixture-implementation-guide.md).