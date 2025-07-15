# Fixture Reference - Complete API Documentation

This document provides a complete reference for all available fixtures and their APIs in the Vrooli testing ecosystem.

**Purpose**: Quick lookup reference for all fixture types and utilities.

**Prerequisites**: Basic understanding of fixtures from [Fixtures Overview](./fixtures-overview.md).

**Related Documents**:
- [Fixtures Overview](./fixtures-overview.md) - Getting started
- [Fixture Patterns](./fixture-patterns.md) - Usage patterns
- [Round-Trip Testing](./round-trip-testing.md) - Integration testing
- [Fixture Implementation Guide](./fixture-implementation-guide.md) - Adding fixtures

## Quick Navigation

- [Factory Chain Architecture](#factory-chain-architecture) - Unified fixture factory pattern
- [API Fixtures](#api-fixtures) - Request/response fixtures for all object types
- [Config Fixtures](#config-fixtures) - Configuration object fixtures
- [Error Fixtures](#error-fixtures) - Error state fixtures for testing
- [Event Fixtures](#event-fixtures) - Real-time event fixtures
- [Permission Fixtures](#permission-fixtures) - Auth and permission scenarios
- [Database Fixtures](#database-fixtures) - Server-side test data creation
- [UI Fixtures](#ui-fixtures) - Component and MSW fixtures
- [Round-Trip Orchestrators](#round-trip-orchestrators) - Full cycle testing
- [Utility Functions](#utility-functions) - Helper functions
- [Type Definitions](#type-definitions) - TypeScript interfaces

## Factory Chain Architecture

The Unified Fixture Architecture uses a factory chain pattern where each factory connects exactly two layers:

### Core Factory Interfaces

```typescript
// Base factory interface
interface FixtureFactory<TInput, TOutput> {
  transform: (input: TInput) => TOutput | Promise<TOutput>
  validate?: (input: TInput) => ValidationResult
  withError?: (error: Error) => TOutput
}

// Layer-specific factories
interface FormFactory<TFormData, TShape> extends FixtureFactory<TFormData, TShape> {
  simulateFormEvent: (event: FormEvent) => TFormData
  validateForm: (data: TFormData) => FormValidationResult
}

interface ShapeFactory<TShape, TAPIInput> extends FixtureFactory<TShape, TAPIInput> {
  useRealShapeFunction: boolean
  shapeFunction: (shape: TShape) => TAPIInput
}

interface APIFactory<TAPIInput, TValidated> extends FixtureFactory<TAPIInput, TValidated> {
  validationSchema: ValidationSchema
  validateAsync: (input: TAPIInput) => Promise<ValidationResult>
}

interface EndpointFactory<TInput, TDBResult> extends FixtureFactory<TInput, TDBResult> {
  endpoint: APIEndpoint
  withAuth: (session: SessionData) => EndpointFactory<TInput, TDBResult>
  withMockResponse: (response: TDBResult) => EndpointFactory<TInput, TDBResult>
}

interface DatabaseFactory<TInput, TResult> extends FixtureFactory<TInput, TResult> {
  prismaClient: PrismaClient
  withTransaction: boolean
  cleanup: (ids: string[]) => Promise<void>
}

// Round-trip orchestrator
interface RoundTripOrchestrator<TFormData, TUIState> {
  executeFullCycle: (formData: TFormData) => Promise<RoundTripResult<TUIState>>
  testCreateFlow: (formData: TFormData) => Promise<TestResult>
  testUpdateFlow: (id: string, formData: Partial<TFormData>) => Promise<TestResult>
  testDeleteFlow: (id: string) => Promise<TestResult>
  verifyDataIntegrity: (original: TFormData, result: TUIState) => boolean
  injectError: (config: ErrorInjectionConfig) => void
}
```

### Usage Pattern

```typescript
// Import the complete factory chain for an object type
import { BookmarkFactoryChain } from "@/test/fixtures/round-trip/bookmark";

const factoryChain = new BookmarkFactoryChain({
  useRealDatabase: true,
  validateAtEachStep: true
});

// Test individual transformations
const shape = await factoryChain.formFactory.transform(formData);
const apiInput = await factoryChain.shapeFactory.transform(shape);

// Or execute complete round-trip
const result = await factoryChain.orchestrator.executeFullCycle(formData);
```

## API Fixtures

Located in `packages/shared/src/__test/fixtures/api-inputs/`

### Available Object Types (41 total)

| Object Type | Import Path | Variants |
|-------------|-------------|----------|
| User | `userFixtures` | minimal, complete, admin, guest, premium |
| Team | `teamFixtures` | minimal, complete, withMembers |
| Project | `projectFixtures` | minimal, complete, published, draft |
| Routine | `routineFixtures` | minimal, complete, simple, complex |
| Comment | `commentFixtures` | minimal, complete, nested |
| Bookmark | `bookmarkFixtures` | minimal, forResource, forProject |
| Chat | `chatFixtures` | minimal, complete, private, group |
| Meeting | `meetingFixtures` | minimal, scheduled, recurring |
| Note | `noteFixtures` | minimal, complete, withVersions |
| Report | `reportFixtures` | minimal, detailed, resolved |
| Tag | `tagFixtures` | minimal, array, withStats |
| Api | `apiFixtures` | minimal, complete, withVersions |
| Award | `awardFixtures` | minimal, earned, progress |
| Bot | `botFixtures` | minimal, complete, withSettings |
| Code | `codeFixtures` | minimal, complete, withVersions |
| DataStructure | `dataStructureFixtures` | minimal, complete, complex |
| Issue | `issueFixtures` | minimal, open, closed |
| Label | `labelFixtures` | minimal, colored, withEmoji |
| Member | `memberFixtures` | owner, admin, member |
| Notification | `notificationFixtures` | unread, read, grouped |
| Phone | `phoneFixtures` | minimal, verified |
| Premium | `premiumFixtures` | active, expired, trial |
| PullRequest | `pullRequestFixtures` | minimal, open, merged |
| Question | `questionFixtures` | minimal, answered |
| Quiz | `quizFixtures` | minimal, complete, published |
| Reaction | `reactionFixtures` | like, dislike, emoji |
| Reminder | `reminderFixtures` | minimal, recurring |
| Resource | `resourceFixtures` | minimal, link, file |
| Role | `roleFixtures` | admin, moderator, member |
| Run | `runFixtures` | minimal, completed, failed |
| Schedule | `scheduleFixtures` | minimal, recurring, exception |
| SmartContract | `smartContractFixtures` | minimal, deployed |
| Stats | `statsFixtures` | empty, populated |
| Transfer | `transferFixtures` | pending, completed |
| View | `viewFixtures` | single, bulk |
| Wallet | `walletFixtures` | minimal, verified |

### Usage Examples

```typescript
// Import specific fixtures
import { userFixtures, projectFixtures } from "@vrooli/shared";

// Use minimal variant
const user = userFixtures.minimal.create;

// Use complete variant  
const project = projectFixtures.complete.find;

// Import all API fixtures
import { apiFixtures } from "@vrooli/shared/__test/fixtures";
const team = apiFixtures.teamFixtures.withMembers.find;
```

### Fixture Structure

Each fixture type provides:

```typescript
interface FixtureType {
    minimal: {
        create: CreateInput;  // For POST requests
        find: FindResult;     // For GET responses
    };
    complete: {
        create: CreateInput;
        find: FindResult;
    };
    // Additional scenario variants...
}
```

## Config Fixtures

Located in `packages/shared/src/__test/fixtures/config/`

### Available Configuration Types

| Config Type | Import Path | Purpose |
|------------|-------------|---------|
| BotConfig | `botConfigFixtures` | Bot behavior settings |
| ChatConfig | `chatConfigFixtures` | Chat room settings |
| RunConfig | `runConfigFixtures` | Execution settings |
| RoutineConfig | `routineConfigFixtures` | Routine definitions |
| UserPreferences | `userPreferencesFixtures` | User settings |
| SystemConfig | `systemConfigFixtures` | System-wide settings |

### Config Fixture Examples

```typescript
import { botConfigFixtures, chatConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

// Bot configuration
const botSettings = botConfigFixtures.complete;
const limitedBot = botConfigFixtures.restricted;

// Chat configurations
const privateChat = chatConfigFixtures.variants.privateTeamChat;
const publicChat = chatConfigFixtures.variants.publicSupport;
```

## Error Fixtures

Located in `packages/shared/src/__test/fixtures/errors/`

### API Error Fixtures

```typescript
import { apiErrorFixtures } from "@vrooli/shared/__test/fixtures/errors";

// Standard HTTP errors
const badRequest = apiErrorFixtures.badRequest.minimal;       // 400
const unauthorized = apiErrorFixtures.unauthorized.standard;   // 401
const forbidden = apiErrorFixtures.forbidden.standard;         // 403
const notFound = apiErrorFixtures.notFound.standard;          // 404
const rateLimit = apiErrorFixtures.rateLimit.standard;        // 429

// With details
const validationError = apiErrorFixtures.badRequest.withDetails;
// { status: 400, code: "BAD_REQUEST", details: { fields: {...} } }

// Factory functions
const customError = apiErrorFixtures.factories.createValidationError({
    email: "Invalid email format",
    password: "Too short"
});
```

### Network Error Fixtures

```typescript
import { networkErrorFixtures } from "@vrooli/shared/__test/fixtures/errors";

// Connection errors
const timeout = networkErrorFixtures.timeout.client;
const offline = networkErrorFixtures.networkOffline;
const refused = networkErrorFixtures.connectionRefused;

// Display messages
const displayError = networkErrorFixtures.networkOffline.display;
// { title: "You're Offline", message: "...", icon: "wifi_off" }
```

### Validation Error Fixtures

```typescript
import { validationErrorFixtures } from "@vrooli/shared/__test/fixtures/errors";

// Field-specific errors
const formErrors = validationErrorFixtures.formErrors.registration;
// { email: "Required", password: "Min 8 chars", confirmPassword: "Must match" }

// Complex validation
const nestedErrors = validationErrorFixtures.nested.project;
// { name: "Required", team: { id: "Invalid" }, tags: ["Too many"] }
```

## Event Fixtures

Located in `packages/shared/src/__test/fixtures/events/`

### Socket Event Fixtures

```typescript
import { socketEventFixtures } from "@vrooli/shared/__test/fixtures/events";

// Connection events
const connected = socketEventFixtures.connection.connected;
const disconnected = socketEventFixtures.connection.disconnected;
const reconnecting = socketEventFixtures.connection.reconnecting;

// Error events
const authError = socketEventFixtures.errors.unauthorized;
```

### Chat Event Fixtures

```typescript
import { chatEventFixtures } from "@vrooli/shared/__test/fixtures/events";

// Message events
const textMsg = chatEventFixtures.messages.textMessage;
const msgWithFile = chatEventFixtures.messages.messageWithAttachment;

// Typing indicators
const typing = chatEventFixtures.typing.start;
const stopTyping = chatEventFixtures.typing.stop;

// Event sequences
const typingFlow = chatEventFixtures.sequences.typingFlow;
// Array of events with delays for testing flows
```

### Swarm Event Fixtures

```typescript
import { swarmEventFixtures } from "@vrooli/shared/__test/fixtures/events";

// Execution lifecycle
const started = swarmEventFixtures.execution.started;
const progress = swarmEventFixtures.execution.progress;
const completed = swarmEventFixtures.execution.completed;

// Agent events
const agentUpdate = swarmEventFixtures.execution.agentUpdate;
// { agentId, status: "thinking", thought: "..." }

// Collaboration
const handoff = swarmEventFixtures.collaboration.handoff;
// { fromAgent, toAgent, context: {...} }
```

### Event Test Utilities

```typescript
import { MockSocketEmitter } from "@vrooli/ui/__test/fixtures/events";

// Create mock socket
const socket = new MockSocketEmitter();

// Register handlers
socket.on("chat:message", (data) => console.log(data));

// Emit events
socket.emit("chat:message", chatEventFixtures.messages.textMessage.data);

// Emit sequences
await socket.emitSequence(chatEventFixtures.sequences.messageDeliveryFlow);

// Get history
const history = socket.getEmitHistory();
```

## Permission Fixtures

Located in `packages/server/src/__test/fixtures/permissions/`

### User Personas

```typescript
import { 
    adminUser,      // id: "111111111111111111"
    standardUser,   // id: "222222222222222222"  
    premiumUser,    // id: "333333333333333333"
    guestUser,      // isLoggedIn: false
} from "@test/fixtures/permissions";
```

### API Key Fixtures

```typescript
import {
    readOnlyPublicApiKey,   // Read public data only
    readOnlyPrivateApiKey,  // Read private data
    writeApiKey,            // Full CRUD access
    botApiKey,              // Bot operations
} from "@test/fixtures/permissions";
```

### Team Scenarios

```typescript
import { 
    basicTeamScenario,      // Owner, admin, member
    complexTeamScenario,    // Nested teams, multiple roles
    publicTeamScenario,     // Public team with guests
} from "@test/fixtures/permissions";

const { team, members } = basicTeamScenario;
// members[0] = owner
// members[1] = admin  
// members[2] = member
```

### Session Helpers

```typescript
import { quickSession, testPermissionMatrix } from "@test/fixtures/permissions";

// Quick session creation
const adminSession = await quickSession.admin();
const userSession = await quickSession.standard();

// Permission matrix testing
await testPermissionMatrix(
    async (session) => endpoint(input, session),
    {
        admin: true,      // Expected result
        standard: false,
        guest: false,
    }
);
```

### Edge Cases

```typescript
import { 
    bannedUser,
    suspendedUser,
    expiredSession,
    rateLimitedUser,
    malformedSession,
} from "@test/fixtures/permissions/edgeCases";
```

## Database Fixtures

Located in `packages/server/src/__test/fixtures/db/`

### Creation Functions

```typescript
// User creation
async function createTestUser(overrides?: Partial<UserCreateInput>): Promise<User>

// Team creation with members
async function createTestTeam(config: {
    owner?: User;
    members?: User[];
    isPublic?: boolean;
}): Promise<Team>

// Project with full relationships
async function createTestProject(config: {
    owner?: User;
    team?: Team;
    tags?: string[];
    isPublished?: boolean;
}): Promise<Project>
```

### Cleanup Utilities

```typescript
// Clean up test data
async function cleanupUsers(ids: string[]): Promise<void>
async function cleanupProjects(ids: string[]): Promise<void>
async function cleanupTestData(objects: TestObject[]): Promise<void>

// Transaction wrapper for auto-rollback
async function withDbTransaction<T>(
    fn: () => Promise<T>
): Promise<T>
```

## UI Fixtures

Located in `packages/ui/src/__test/fixtures/`

### Directory Structure

- **`api-responses/`** - Mock API response data for UI testing
- **`form-data/`** - Form input test data and initial values
- **`helpers/`** - Transformation and utility functions
- **`round-trip-tests/`** - Examples of end-to-end UI tests
- **`sessions/`** - Session-related fixtures (in development)
- **`ui-states/`** - UI state fixtures (in development)

### API Response Fixtures

```typescript
// packages/ui/src/__test/fixtures/api-responses/bookmarkResponses.ts
export const bookmarkResponses = {
    success: {
        status: 200,
        data: bookmarkFixtures.complete.find,
    },
    error: {
        status: 400,
        error: "Invalid bookmark data",
    },
    notFound: {
        status: 404,
        error: "Bookmark not found",
    },
};
```

### Form Data Fixtures

```typescript
// packages/ui/src/__test/fixtures/form-data/bookmarkFormData.ts
export const bookmarkFormData = {
    valid: {
        bookmarkFor: "Resource",
        forConnect: "resource_123",
    },
    invalid: {
        bookmarkFor: "",  // Missing required field
        forConnect: "123",
    },
};
```

### MSW Integration (Storybook)

MSW handlers are configured within Storybook stories using helpers:

```typescript
// In a Storybook story file
import { storybookMocking } from "@test/helpers/storybookMocking";
import { bookmarkResponses } from "@test/fixtures/api-responses/bookmarkResponses";

export const Default = {
    parameters: {
        msw: {
            handlers: [
                rest.get("/api/bookmark/:id", (req, res, ctx) => {
                    return res(ctx.json(bookmarkResponses.success));
                }),
            ],
        },
    },
};
```

### Test Utilities

```typescript
// packages/ui/src/__test/testUtils.tsx
import { render } from "@testing-library/react";

// Custom render with providers
export function renderWithProviders(ui: React.ReactElement, options = {}) {
    return render(ui, {
        wrapper: AllTheProviders,
        ...options,
    });
}
```

## Round-Trip Orchestrators

Located in `packages/ui/src/__test/fixtures/round-trip-tests/`

Round-trip orchestrators coordinate the complete testing flow from UI form input through database persistence and back to UI display.

### Available Orchestrators

Each object type has its own orchestrator implementing the `RoundTripOrchestrator` interface:

```typescript
// Import specific orchestrator
import { BookmarkRoundTripFactory } from "@/test/fixtures/round-trip-tests/bookmarkRoundTrip";
import { UserRoundTripFactory } from "@/test/fixtures/round-trip-tests/userRoundTrip";
import { TeamRoundTripFactory } from "@/test/fixtures/round-trip-tests/teamRoundTrip";

// Initialize with configuration
const bookmarkOrchestrator = new BookmarkRoundTripFactory({
  useRealDatabase: true,
  validateAtEachStep: true,
  enableRetries: true
});
```

### Orchestrator Methods

```typescript
interface RoundTripOrchestrator<TFormData, TUIState> {
  // Complete lifecycle testing
  executeFullCycle(formData: TFormData): Promise<RoundTripResult>
  
  // CRUD operations
  testCreateFlow(formData: TFormData): Promise<TestResult>
  testUpdateFlow(id: string, updates: Partial<TFormData>): Promise<TestResult>
  testDeleteFlow(id: string): Promise<TestResult>
  
  // Data integrity
  verifyDataIntegrity(original: TFormData, result: TUIState): boolean
  
  // Error testing
  testErrorRecovery(scenario: ErrorScenario): Promise<TestResult>
  injectError(config: ErrorInjectionConfig): void
  
  // Partial flow testing
  testUpToStage(formData: TFormData, stage: TestStage): Promise<PartialResult>
  testFromStage(data: any, stage: TestStage): Promise<PartialResult>
}
```

### Example Usage

```typescript
describe("User Round-Trip Testing", () => {
  const orchestrator = new UserRoundTripFactory();
  
  it("should handle complete user registration flow", async () => {
    const formData = {
      email: "test@example.com",
      password: "SecurePass123!",
      name: "Test User",
      acceptTerms: true
    };
    
    const result = await orchestrator.executeFullCycle(formData);
    
    expect(result.success).toBe(true);
    expect(result.stages).toMatchObject({
      form: "completed",
      validation: "completed",
      api: "completed",
      database: "completed",
      response: "completed",
      ui: "completed"
    });
    
    // Verify user can log in with created account
    const loginResult = await orchestrator.testLoginFlow({
      email: formData.email,
      password: formData.password
    });
    
    expect(loginResult.authenticated).toBe(true);
  });
});
```

## Utility Functions

### ID Generation

```typescript
import { generatePK, DUMMY_ID } from "@vrooli/shared";

// Generate unique ID
const id = generatePK();  // "123456789012345678"

// Temporary ID for creates
const temp = DUMMY_ID;    // "00000000-0000-0000-0000-000000000000"
```

### Shape Functions

```typescript
import { shapeUser, shapeProject } from "@vrooli/shared";

// Transform for create
const createData = shapeUser.create(formData);

// Transform for update  
const updateData = shapeUser.update(formData);

// Transform for display
const displayData = shapeUser.find(apiResponse);
```

### Validation Functions

```typescript
import { userValidation, projectValidation } from "@vrooli/shared";

// Validate create input
const result = await userValidation.create.validate(data);

// Check if valid
if (result.isValid) {
    // Proceed with creation
}
```

## Type Definitions

### Core Fixture Types

```typescript
// Base fixture structure
interface Fixture<TCreate, TFind> {
    minimal: {
        create: TCreate;
        find: TFind;
    };
    complete: {
        create: TCreate;
        find: TFind;
    };
    [scenario: string]: {
        create?: TCreate;
        find?: TFind;
    };
}

// Shape type imports
import type { Shape } from "@vrooli/shared";
type User = Shape.User;
type UserCreateInput = Shape.UserCreateInput;
```

### Test Context Types

```typescript
interface TestContext {
    user?: Shape.User;
    session?: Session;
    req: Request;
    res: Response;
}

interface PermissionTestResult {
    allowed: boolean;
    error?: string;
    code?: string;
}
```

## Compatibility Matrix

| Fixture Type | Server Tests | UI Tests | Integration Tests |
|--------------|--------------|----------|-------------------|
| API Fixtures | ✅ | ✅ | ✅ |
| Config Fixtures | ✅ | ✅ | ✅ |
| Permission Fixtures | ✅ | ⚠️ | ✅ |
| Database Fixtures | ✅ | ❌ | ✅ |
| UI Fixtures | ❌ | ✅ | ⚠️ |

Legend:
- ✅ Fully supported
- ⚠️ Partial support (with adapters)
- ❌ Not applicable

## Best Practices

### Importing Fixtures

```typescript
// ✅ GOOD: Import what you need
import { userFixtures, teamFixtures } from "@vrooli/shared";

// ❌ BAD: Import everything
import * as fixtures from "@vrooli/shared/__test/fixtures";
```

### Using Fixtures

```typescript
// ✅ GOOD: Create a copy for modifications
const user = { ...userFixtures.complete.find, name: "Modified" };

// ❌ BAD: Mutate shared fixture
userFixtures.complete.find.name = "Modified";
```

### Type Safety

```typescript
// ✅ GOOD: Use Shape types
const user: Shape.User = userFixtures.complete.find;

// ❌ BAD: Use 'any' type
const user: any = userFixtures.complete.find;
```

## Troubleshooting

### Import Errors
- Ensure `.js` extension in imports
- Check package.json exports configuration
- Verify TypeScript paths configuration

### Type Mismatches
- Regenerate types: `pnpm prisma generate`
- Update Shape imports
- Check fixture version compatibility

### Missing Fixtures
- Check implementation status in guide
- Create following patterns in this reference
- Submit PR with new fixtures

## Contributing

When adding new fixtures:
1. Follow existing patterns
2. Include all variants (minimal, complete)
3. Add TypeScript types
4. Document in this reference
5. Add usage examples

For questions, see the [Fixture Implementation Guide](./fixture-implementation-guide.md).