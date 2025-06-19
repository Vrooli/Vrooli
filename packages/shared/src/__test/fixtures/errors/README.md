# Error State Fixtures

Error fixtures provide consistent error scenarios for testing error handling across the Vrooli application. These fixtures help ensure that error states are handled gracefully and consistently throughout the codebase.

## Overview

The error fixtures are organized into six categories:

1. **API Errors** - HTTP error responses (400, 401, 403, 404, 429, 500, etc.)
2. **Validation Errors** - Field-level validation failures
3. **Network Errors** - Connection issues, timeouts, offline states
4. **Auth Errors** - Authentication and authorization failures
5. **Business Errors** - Business logic violations, limits, conflicts
6. **System Errors** - Infrastructure and service failures

## Usage

### Basic Import

```typescript
import { 
    apiErrorFixtures,
    validationErrorFixtures,
    networkErrorFixtures 
} from "@vrooli/shared/__test/fixtures/errors";
```

### API Error Examples

```typescript
// Test 400 Bad Request handling
const badRequest = apiErrorFixtures.badRequest.withDetails;
if (response.status === badRequest.status) {
    showFieldErrors(badRequest.details.fields);
}

// Test rate limiting
const rateLimitError = apiErrorFixtures.rateLimit.standard;
if (response.status === 429) {
    const retryAfter = rateLimitError.retryAfter; // 60 seconds
    showRetryMessage(retryAfter);
}

// Create custom validation error
const customError = apiErrorFixtures.factories.createValidationError({
    email: "Invalid format",
    password: "Too short"
});
```

### Network Error Examples

```typescript
// Handle timeout
try {
    await fetchData();
} catch (error) {
    const timeoutError = networkErrorFixtures.timeout.client;
    showError(timeoutError.display.title, timeoutError.display.message);
    if (timeoutError.display.retry) {
        retryOperation();
    }
}

// Check offline state
const offlineError = networkErrorFixtures.networkOffline;
if (!navigator.onLine) {
    showOfflineMessage(offlineError.display);
}
```

### Validation Error Examples

```typescript
// Form validation
const registrationErrors = validationErrorFixtures.formErrors.registration;
form.setErrors(registrationErrors.fields);

// Nested validation
const projectErrors = validationErrorFixtures.nested.project;
if (projectErrors.team) {
    showTeamErrors(projectErrors.team);
}

// Array field errors
const emailErrors = validationErrorFixtures.arrayErrors.emails;
emailErrors.fields.emails.forEach((error, index) => {
    if (error) {
        markEmailInvalid(index, error);
    }
});
```

### Auth Error Examples

```typescript
// Handle login failures
const loginError = authErrorFixtures.login.accountLocked;
if (error.code === loginError.code) {
    showLockoutMessage(loginError.details.lockoutDuration);
    redirectTo(loginError.action.url);
}

// Check permissions
const permissionError = authErrorFixtures.permissions.insufficientRole;
if (!hasRole(permissionError.details.requiredRole)) {
    showPermissionDenied(permissionError.message);
}
```

### Business Error Examples

```typescript
// Resource limits
const creditError = businessErrorFixtures.limits.creditLimit;
if (credits < creditError.details.required) {
    showUpgradePrompt(creditError.userAction);
}

// Workflow violations
const workflowError = businessErrorFixtures.workflow.prerequisiteNotMet;
if (workflowError.details.missingSteps.length > 0) {
    redirectToOnboarding(workflowError.details.missingSteps);
}
```

### System Error Examples

```typescript
// Database errors
const dbError = systemErrorFixtures.database.connectionLost;
if (dbError.recovery.automatic) {
    waitForRecovery(dbError.recovery.estimatedRecovery);
} else {
    showMaintenanceMode();
}

// Service failures
const serviceError = systemErrorFixtures.externalService.aiServiceDown;
if (serviceError.recovery.fallback) {
    useAlternativeService(serviceError.recovery.fallback);
}
```

## Factory Functions

Each fixture category provides factory functions for creating custom errors:

```typescript
// API errors
const customApiError = apiErrorFixtures.factories.createApiError(
    418, "TEAPOT", "I'm a teapot", { brewing: true }
);

// Validation errors
const fieldError = validationErrorFixtures.factories.createFieldError(
    "username", "Username already taken"
);

// Network errors
const timeoutError = networkErrorFixtures.factories.createTimeoutError(5000);

// Auth errors
const permissionError = authErrorFixtures.factories.createPermissionError(
    "projects", "write", "read"
);

// Business errors
const limitError = businessErrorFixtures.factories.createLimitError(
    "storage", 5.2, 5.0, "/billing/storage"
);

// System errors
const criticalError = systemErrorFixtures.factories.createCriticalError(
    "PaymentService", "Payment processing unavailable", "use_backup_gateway"
);
```

## Testing Patterns

### Component Testing

```typescript
import { render, screen } from "@testing-library/react";
import { apiErrorFixtures } from "@vrooli/shared/__test/fixtures/errors";

it("should display validation errors", () => {
    const error = apiErrorFixtures.badRequest.withDetails;
    render(<Form error={error} />);
    
    expect(screen.getByText("Invalid email format")).toBeInTheDocument();
    expect(screen.getByText("Password must be at least 8 characters")).toBeInTheDocument();
});
```

### API Mocking with MSW

```typescript
import { rest } from "msw";
import { apiErrorFixtures } from "@vrooli/shared/__test/fixtures/errors";

const handlers = [
    rest.post("/api/login", (req, res, ctx) => {
        return res(
            ctx.status(401),
            ctx.json(apiErrorFixtures.unauthorized.standard)
        );
    }),
];
```

### Error Boundary Testing

```typescript
import { networkErrorFixtures } from "@vrooli/shared/__test/fixtures/errors";

it("should handle network errors gracefully", () => {
    const error = networkErrorFixtures.connectionRefused;
    
    render(
        <ErrorBoundary fallback={<ErrorDisplay />}>
            <ComponentThatThrows error={error.error} />
        </ErrorBoundary>
    );
    
    expect(screen.getByText(error.display.title)).toBeInTheDocument();
});
```

## Best Practices

1. **Use appropriate variants**: Choose minimal fixtures for simple tests, detailed fixtures for comprehensive testing
2. **Leverage factory functions**: Create custom errors for specific test scenarios
3. **Test error recovery**: Use recovery information to test retry logic and fallbacks
4. **Mock consistently**: Use the same error fixtures in both unit and integration tests
5. **Handle all error types**: Test network, validation, auth, business, and system errors

## Adding New Error Fixtures

To add new error scenarios:

1. Identify the appropriate category (api, validation, network, auth, business, system)
2. Add the new fixture following existing patterns
3. Include both minimal and detailed variants where applicable
4. Add factory functions for dynamic error creation
5. Update this documentation with usage examples

## Error Properties Reference

### API Errors
- `status`: HTTP status code
- `code`: Error code constant
- `message`: User-friendly message
- `details`: Additional error details
- `retryAfter`: Seconds until retry (rate limiting)

### Network Errors
- `error`: JavaScript Error object
- `display`: UI display properties (title, message, icon, retry)
- `metadata`: Additional context (url, duration, attempts)

### Validation Errors
- `fields`: Field-specific error messages
- `message`: Overall validation message

### Auth Errors
- `code`: Auth error code
- `message`: Error message
- `details`: Context-specific details
- `action`: Suggested user action

### Business Errors
- `code`: Business error code
- `message`: Error message
- `type`: Error category (limit, conflict, state, workflow, constraint, policy)
- `details`: Business-specific context
- `userAction`: Suggested resolution

### System Errors
- `code`: System error code
- `message`: Error message
- `severity`: Error severity (critical, error, warning)
- `component`: Affected system component
- `details`: Technical details
- `recovery`: Recovery options (automatic, retryable, fallback)