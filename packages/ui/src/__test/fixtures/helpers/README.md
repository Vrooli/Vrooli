# UI Test Helpers

This directory contains transformation utilities and mock services that bridge the gap between UI form data and API interactions in the Vrooli testing ecosystem. These helpers are essential for round-trip testing and ensuring data integrity across the application.

## Purpose

UI Test Helpers serve as the **transformation and mock service layer** for UI testing:

- **Transform form data to API requests** - Convert UI form inputs to properly shaped API request objects
- **Transform API responses to form data** - Convert API responses back to UI-editable form structures
- **Provide mock services** - Simulate API operations with in-memory storage for testing
- **Validate data integrity** - Ensure data consistency throughout transformation cycles
- **Support round-trip testing** - Enable end-to-end data flow testing from UI to API and back

## Current Architecture

### Helper Files Overview

| File | Purpose | Key Functions |
|------|---------|---------------|
| `apiKeyTransformations.js` | API key mock service | `create()`, `findById()`, `update()`, `delete()` |
| `bookmarkTransformations.ts` | Bookmark data transformations and mock service | `transformFormToCreateRequest()`, `transformApiResponseToForm()`, `validateBookmarkFormData()` |
| `reportResponseTransformations.ts` | Report response transformations and mock service | Form-to-API transformations, validation, mock CRUD operations |
| `resourceTransformations.ts` | Resource mock service | CRUD operations with in-memory storage |
| `teamTransformations.ts` | Team mock service | Team creation and management operations |

### Key Patterns

1. **Transformation Functions**
   ```typescript
   // Form to API request
   export function transformFormToCreateRequest(formData: FormData): CreateInput {
       // Convert UI form structure to API request structure
   }
   
   // API response to Form
   export function transformApiResponseToForm(response: ApiResponse): FormData {
       // Convert API response back to editable form structure
   }
   ```

2. **Mock Services**
   ```typescript
   export const mockService = {
       async create(data: CreateInput): Promise<Response> {
           // Simulate API delay and create response
       },
       async findById(id: string): Promise<Response> {
           // Retrieve from in-memory storage
       },
       async update(id: string, data: UpdateInput): Promise<Response> {
           // Update in storage and return updated object
       },
       async delete(id: string): Promise<{ success: boolean }> {
           // Remove from storage
       }
   };
   ```

3. **In-Memory Storage**
   ```typescript
   // Global test storage for persistence across test operations
   (globalThis as any).__testObjectStorage = {};
   ```

## Ideal Architecture

### Goals

The ideal UI Test Helpers architecture should provide:

1. **Consistent Testing Infrastructure** - Unified patterns for all UI component testing
2. **MSW Integration** - Seamless API mocking for component and integration tests
3. **Storybook Support** - Decorators and utilities for visual testing
4. **Form Testing Utilities** - Helpers for testing complex form interactions
5. **Async Testing Patterns** - Support for loading states, error handling, and real-time updates
6. **Round-Trip Testing Support** - Complete data flow validation from UI to API and back

### Proposed Structure

```
packages/ui/src/__test/fixtures/helpers/
├── README.md                          # This file
├── transformations/                   # Data transformation utilities
│   ├── index.ts                      # Barrel export
│   ├── base.ts                       # Base transformation interfaces
│   ├── apiKey.ts                     # API key transformations
│   ├── bookmark.ts                   # Bookmark transformations
│   ├── reportResponse.ts             # Report response transformations
│   ├── resource.ts                   # Resource transformations
│   └── team.ts                       # Team transformations
├── services/                         # Mock service implementations
│   ├── index.ts                      # Barrel export
│   ├── base.ts                       # Base service interface
│   ├── apiKeyService.ts              # API key mock service
│   ├── bookmarkService.ts            # Bookmark mock service
│   ├── reportResponseService.ts      # Report response mock service
│   ├── resourceService.ts            # Resource mock service
│   └── teamService.ts                # Team mock service
├── testing/                          # Testing utilities
│   ├── index.ts                      # Barrel export
│   ├── render.tsx                    # Enhanced render with providers
│   ├── userEvents.ts                 # User event utilities
│   ├── formHelpers.ts                # Form testing utilities
│   ├── asyncHelpers.ts               # Async testing utilities
│   └── accessibility.ts              # A11y testing utilities
├── msw/                              # MSW integration
│   ├── index.ts                      # Barrel export
│   ├── handlers.ts                   # Common MSW handlers
│   ├── server.ts                     # MSW server setup
│   └── utils.ts                      # MSW utilities
└── storybook/                        # Storybook utilities
    ├── index.ts                      # Barrel export
    ├── decorators.ts                 # Story decorators
    ├── mocking.ts                    # Mocking utilities
    └── parameters.ts                 # Common story parameters
```

### Enhanced Interfaces

```typescript
// Base transformation interface
interface Transformation<TForm, TCreate, TUpdate, TResponse> {
    // Core transformations
    formToCreate: (form: TForm) => TCreate;
    formToUpdate: (id: string, form: Partial<TForm>) => TUpdate;
    responseToForm: (response: TResponse) => TForm;
    
    // Validation
    validate: (form: TForm) => string[];
    
    // Comparison
    areEqual: (form1: TForm, form2: TForm) => boolean;
}

// Base service interface
interface MockService<TCreate, TUpdate, TResponse> {
    // CRUD operations
    create: (data: TCreate) => Promise<TResponse>;
    findById: (id: string) => Promise<TResponse>;
    findMany: (filter?: any) => Promise<TResponse[]>;
    update: (id: string, data: TUpdate) => Promise<TResponse>;
    delete: (id: string) => Promise<{ success: boolean }>;
    
    // Utility operations
    clearAll: () => void;
    seed: (data: TResponse[]) => void;
    verify: (id: string, expected: Partial<TResponse>) => Promise<boolean>;
}

// Testing utilities interface
interface UITestHelpers {
    // Rendering
    render: (component: ReactElement, options?: RenderOptions) => RenderResult;
    renderWithRouter: (component: ReactElement, options?: RouterOptions) => RenderResult;
    renderWithProviders: (component: ReactElement, options?: ProviderOptions) => RenderResult;
    
    // User interactions
    user: UserEvent;
    fillForm: (container: HTMLElement, data: Record<string, any>) => Promise<void>;
    submitForm: (form: HTMLFormElement) => Promise<void>;
    
    // Async utilities
    waitForLoadingToFinish: () => Promise<void>;
    waitForErrorToAppear: () => Promise<HTMLElement>;
    waitForSuccessMessage: () => Promise<HTMLElement>;
    
    // MSW utilities
    setupMSW: (handlers?: RequestHandler[]) => void;
    mockApiResponse: <T>(endpoint: string, response: T) => void;
    mockApiError: (endpoint: string, error: any) => void;
    
    // Accessibility
    checkA11y: (container: HTMLElement) => Promise<void>;
    getByLabelText: (text: string) => HTMLElement;
    
    // Storybook
    withDecorators: (...decorators: Decorator[]) => Decorator;
    mockData: <T>(template: T, overrides?: Partial<T>) => T;
}
```

## Integration with Unified Fixture Architecture

### 1. Use Shared Fixtures

```typescript
import { apiFixtures } from "@vrooli/shared/__test/fixtures";
import { transformFormToCreateRequest } from "./transformations/bookmark";

// Use shared fixtures as base data
const bookmarkData = apiFixtures.bookmarkFixtures.complete.create;
const formData = transformApiResponseToForm(bookmarkData);
```

### 2. Support Round-Trip Testing

```typescript
// Test complete data flow
it("should maintain data integrity through transformations", async () => {
    // 1. Start with form data
    const formData = createBookmarkFormData();
    
    // 2. Transform to API request
    const request = transformFormToCreateRequest(formData);
    
    // 3. Send through mock service
    const response = await mockBookmarkService.create(request);
    
    // 4. Transform back to form
    const resultForm = transformApiResponseToForm(response);
    
    // 5. Verify data integrity
    expect(areBookmarkFormsEqual(formData, resultForm)).toBe(true);
});
```

### 3. MSW Integration for Component Tests

```typescript
import { setupMSW } from "./msw";
import { mockBookmarkService } from "./services/bookmarkService";

// Setup MSW with mock service
setupMSW([
    http.post("/api/v2/bookmark", async ({ request }) => {
        const data = await request.json();
        const response = await mockBookmarkService.create(data);
        return HttpResponse.json(response);
    })
]);
```

## Best Practices

### 1. Transformation Consistency

- Always validate data before transformation
- Use TypeScript types from `@vrooli/shared`
- Handle optional fields explicitly
- Preserve data integrity through round trips

### 2. Mock Service Reliability

- Simulate realistic API delays
- Store data in global test storage for persistence
- Provide cleanup methods for test isolation
- Return deep copies to prevent mutations

### 3. Testing Patterns

```typescript
// Use descriptive test names
describe("BookmarkForm", () => {
    it("should create bookmark with new list when user selects 'Create New List'", async () => {
        // Test implementation
    });
});

// Test error scenarios
it("should display validation errors for invalid form data", async () => {
    const { user, getByRole, findByText } = renderWithProviders(<BookmarkForm />);
    
    await user.click(getByRole("button", { name: /submit/i }));
    
    expect(await findByText("Bookmark type is required")).toBeInTheDocument();
});

// Test loading states
it("should show loading spinner while creating bookmark", async () => {
    const { user, getByRole, findByRole } = renderWithProviders(<BookmarkForm />);
    
    await fillForm({ bookmarkFor: "Project", forConnect: "123" });
    await user.click(getByRole("button", { name: /submit/i }));
    
    expect(await findByRole("progressbar")).toBeInTheDocument();
});
```

### 4. Component Testing Guidelines

- Render with all necessary providers
- Use user event utilities for interactions
- Test accessibility with every component
- Verify error boundaries work correctly
- Test responsive behavior when needed

## Usage Examples

### Basic Component Test

```typescript
import { renderWithProviders, user, fillForm } from "../__test/fixtures/helpers/testing";
import { mockTeamService } from "../__test/fixtures/helpers/services/teamService";

describe("TeamForm", () => {
    it("should create team with form data", async () => {
        const { getByRole, findByText } = renderWithProviders(<TeamForm />);
        
        await fillForm({
            name: "Test Team",
            handle: "test-team",
            bio: "A test team description"
        });
        
        await user.click(getByRole("button", { name: /create team/i }));
        
        expect(await findByText("Team created successfully")).toBeInTheDocument();
    });
});
```

### Storybook Story with MSW

```typescript
import type { Meta, StoryObj } from "@storybook/react";
import { http, HttpResponse } from "msw";
import { TeamView } from "./TeamView";
import { mockTeamService } from "../__test/fixtures/helpers/services/teamService";
import { centeredDecorator } from "../__test/fixtures/helpers/storybook/decorators";

const meta: Meta<typeof TeamView> = {
    title: "Objects/Team/View",
    component: TeamView,
    decorators: [centeredDecorator],
    parameters: {
        msw: {
            handlers: [
                http.get("/api/v2/team/:id", async ({ params }) => {
                    const team = await mockTeamService.findById(params.id as string);
                    return HttpResponse.json({ team });
                })
            ]
        }
    }
};

export default meta;
```

### Form Validation Test

```typescript
import { validateBookmarkFormData } from "../__test/fixtures/helpers/transformations/bookmark";

describe("Bookmark Form Validation", () => {
    it("should validate required fields", () => {
        const errors = validateBookmarkFormData({
            bookmarkFor: "",
            forConnect: "",
            createNewList: true,
            newListLabel: ""
        });
        
        expect(errors).toContain("Bookmark type is required");
        expect(errors).toContain("Target object ID is required");
        expect(errors).toContain("List label is required when creating a new list");
    });
});
```

## Migration Path

### Phase 1: Organize Existing Helpers
1. Move transformation functions to `transformations/` directory
2. Extract mock services to `services/` directory
3. Create barrel exports for easy imports

### Phase 2: Add Testing Utilities
1. Create enhanced render functions with providers
2. Add form testing utilities
3. Implement async testing helpers

### Phase 3: MSW Integration
1. Create common MSW handlers using mock services
2. Add MSW server setup for tests
3. Document MSW patterns for component tests

### Phase 4: Storybook Enhancement
1. Consolidate decorators into single location
2. Create story parameter presets
3. Add visual regression testing utilities

## Future Enhancements

1. **Performance Testing Utilities**
   - React performance measurement helpers
   - Bundle size impact testing
   - Render performance benchmarks

2. **Visual Regression Testing**
   - Screenshot comparison utilities
   - Visual diff reporting
   - Responsive design testing

3. **E2E Testing Bridge**
   - Utilities that work in both unit and E2E tests
   - Shared data fixtures for Playwright
   - Cross-test-type assertions

4. **AI-Assisted Testing**
   - Smart test generation based on component props
   - Automatic edge case discovery
   - Test coverage suggestions

## Related Documentation

- [Fixtures Overview](/docs/testing/fixtures-overview.md) - Overview of the fixture system
- [Round-Trip Testing](/docs/testing/round-trip-testing.md) - End-to-end data flow testing
- [UI Test Utils](/packages/ui/src/__test/testUtils.tsx) - Current test utilities
- [Storybook Helpers](/packages/ui/src/__test/helpers/) - Storybook decorators and mocking