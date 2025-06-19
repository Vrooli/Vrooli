# UI Test Fixtures

This directory contains comprehensive test fixtures for UI testing in the Vrooli application. These fixtures provide consistent, realistic test data for component testing, integration testing, and development.

## Directory Structure

```
fixtures/
├── api-responses/      # Mock API response data
├── form-data/         # Form input test data
├── helpers/           # Transformation utilities
├── round-trip-tests/  # End-to-end test examples
├── sessions/          # Session and auth fixtures
├── ui-states/         # Loading, error, and success states
└── index.ts           # Central exports
```

## Usage

### Basic Import

```typescript
import { 
  minimalUserResponse, 
  completeProjectResponse,
  loadingStates,
  authenticatedUserSession 
} from "@/test/fixtures";
```

### API Response Fixtures

Mock API responses for testing components that consume API data:

```typescript
import { userResponseVariants } from "@/test/fixtures";

// In your test
const mockUser = userResponseVariants.complete;
render(<UserProfile user={mockUser} />);
```

### Form Data Fixtures

Pre-filled form data for testing form components:

```typescript
import { minimalProjectCreateFormInput } from "@/test/fixtures";

// In your test
const { getByLabelText } = render(<ProjectCreateForm />);
fireEvent.change(getByLabelText("Project Name"), {
  target: { value: minimalProjectCreateFormInput.name }
});
```

### UI State Fixtures

Test various UI states like loading, error, and success:

```typescript
import { loadingStates, apiErrors } from "@/test/fixtures";

// Test loading state
render(<DataList loading={true} data={null} />);

// Test error state
render(<DataList loading={false} error={apiErrors.serverError} />);
```

### Session Fixtures

Test different authentication states:

```typescript
import { guestSession, adminUserSession } from "@/test/fixtures";

// Test as guest
render(<App session={guestSession} />);

// Test as admin
render(<App session={adminUserSession} />);
```

## Available Fixtures

### Objects

- **User**: Basic users, bots, admins, suspended users
- **Team**: Public teams, private teams, with various member roles
- **Project**: Simple projects, multi-version projects, private projects
- **Chat**: Direct messages, group chats, team chats
- **Routine**: API routines, multi-step routines, smart contract routines
- **Bookmark**: Various bookmark types and lists

### States

- **Loading**: Initial load, refresh, pagination, upload progress
- **Error**: API errors (400-500), network errors, validation errors
- **Success**: Create, update, delete, upload success states

### Sessions

- **Guest**: Unauthenticated user
- **Authenticated**: Regular logged-in user
- **Premium**: User with premium features
- **Admin**: User with admin privileges
- **Team Member**: User acting as team
- **Suspended**: Restricted user account

## Best Practices

1. **Use Minimal Fixtures First**: Start with `minimal*` fixtures and only use `complete*` when testing complex scenarios

2. **Combine Fixtures**: Mix and match fixtures to create realistic test scenarios
   ```typescript
   const teamWithProjects = {
     ...completeTeamResponse,
     projects: [minimalProjectResponse, completeProjectResponse]
   };
   ```

3. **Override as Needed**: Fixtures are meant to be customized
   ```typescript
   const customUser = {
     ...minimalUserResponse,
     name: "Custom Name",
     isPrivate: true
   };
   ```

4. **Use Variants**: Many fixtures provide variants for common scenarios
   ```typescript
   const botUser = userResponseVariants.bot;
   const privateTeam = teamResponseVariants.private;
   ```

5. **Test Edge Cases**: Use error and edge case fixtures
   ```typescript
   const rateLimitError = apiErrors.rateLimited;
   const expiredSession = sessionStates.expired;
   ```

## Helper Functions

Many fixture files include helper functions:

- `createLoadingState(message)`: Create custom loading states
- `validateTeamHandle(handle)`: Validate form inputs
- `transformFormToApiInput(data)`: Convert form data to API format
- `isSessionValid(session)`: Check session validity

## Contributing

When adding new fixtures:

1. Follow the existing naming conventions
2. Provide both minimal and complete variants
3. Include common error/edge cases
4. Add TypeScript types from `@vrooli/shared`
5. Export from the main index.ts file
6. Document any special usage requirements

## Examples

See the `round-trip-tests/` directory for complete examples of how to use these fixtures in end-to-end testing scenarios.