# Permission & Authorization Fixtures

This directory contains standardized fixtures for testing authentication and authorization scenarios across the Vrooli platform. These fixtures form the **security testing layer** of our unified fixture architecture.

## Overview

Permission fixtures provide:
- **Consistent user personas** with predefined permissions and authentication states
- **API key configurations** with various permission scopes and limitations
- **Team membership scenarios** with role-based access control
- **Edge case handling** for security boundaries and error conditions
- **Integration scenarios** for multi-user workflows and permission cascading
- **Session management utilities** for creating realistic authentication contexts

## Current Architecture Assessment

### What's Working Well ✅

1. **Comprehensive User Personas** - Covers 10+ user types from admin to suspended accounts
2. **API Key Coverage** - 10+ API key configurations with different permission levels
3. **Team Scenarios** - Multiple team configurations including nested hierarchies
4. **Edge Case Testing** - Security-focused edge cases like CSRF attacks and rate limiting
5. **Session Helpers** - Convenient utilities for creating test sessions
6. **Type Safety** - Full TypeScript support with proper typing
7. **Integration Scenarios** - Complex multi-actor permission tests

### Areas for Improvement 🔧

1. **Missing Round-Trip Integration** - No clear integration with the unified fixture architecture
2. **Limited Permission Validation** - Could use more granular permission checking utilities
3. **No Permission Inheritance Testing** - Limited support for cascading permissions
4. **Sparse Documentation** - Example test file but limited inline documentation
5. **No Performance Testing** - Missing fixtures for testing permission check performance
6. **Limited OAuth/SSO Support** - No fixtures for external authentication providers
7. **No Rate Limiting Simulation** - Basic rate limiting user but no comprehensive testing

## The Permission Testing Challenge

Before this fixture system, permission tests were scattered across endpoint files, making it difficult to:
- Test the same user across different endpoints
- Ensure consistent permission behavior
- Test complex scenarios (team permissions, API keys, etc.)
- Avoid duplicating test user creation logic
- Validate security boundaries across layers

## Quick Start

```typescript
import { 
    adminUser,
    standardUser,
    quickSession,
    testPermissionMatrix 
} from "@test/fixtures/permissions";

// One-liner session creation
const { req, res } = await quickSession.admin();

// Test permission matrix across multiple scenarios
await testPermissionMatrix(
    async (session) => endpoint.findOne({ input }, session),
    {
        admin: true,      // Admin can access
        standard: false,  // Regular user cannot
        guest: false,     // Guest cannot
    }
);
```

## Fixture Categories

### 1. **User Personas** (`userPersonas.ts`)
Consistent user types with predictable permissions:

```typescript
import { adminUser, standardUser, guestUser } from "@test/fixtures/permissions";

// Each persona has consistent IDs and properties
adminUser.id === "111111111111111111" // Always the same
standardUser.id === "222222222222222222"
guestUser.isLoggedIn === false
```

Available personas:
- `adminUser` - System administrator with full access
- `standardUser` - Regular user with standard permissions
- `premiumUser` - User with premium features
- `unverifiedUser` - New user with unverified email
- `bannedUser` - Soft-locked account
- `guestUser` - Not logged in
- `botUser` - Bot account with special permissions
- `customRoleUser` - User with custom role permissions
- `suspendedUser` - Temporarily suspended account (HardLocked)
- `expiredPremiumUser` - User whose premium subscription has expired

### 2. **API Key Configurations** (`apiKeyPermissions.ts`)
Various API key permission combinations:

```typescript
import { readOnlyPublicApiKey, writeApiKey, botApiKey } from "@test/fixtures/permissions";

// Predefined API keys with different scopes
readOnlyPublicApiKey.permissions.read === "Public"
writeApiKey.permissions.write === "Private"
botApiKey.permissions.bot === true
```

Available API keys:
- `readOnlyPublicApiKey` - Can only read public data
- `readPrivateApiKey` - Can read public and private user data
- `writeApiKey` - Full CRUD permissions on user data
- `botApiKey` - Special permissions for automated operations
- `externalApiKey` - Third-party integration with limited scope
- `rateLimitedApiKey` - For testing quota enforcement
- `authReadApiKey` - Can read auth-specific data
- `authWriteApiKey` - Full access including auth operations
- `expiredApiKey` - API key that has expired (isExpired: true)
- `revokedApiKey` - API key that was manually revoked (isRevoked: true)

### 3. **Team Scenarios** (`teamScenarios.ts`)
Complex team membership testing:

```typescript
import { basicTeamScenario } from "@test/fixtures/permissions";

// Complete team with owner, admin, and member
const { team, members } = basicTeamScenario;
members[0].role === "Owner"
members[1].role === "Admin"
members[2].role === "Member"
```

Available scenarios:
- `basicTeamScenario` - Standard team hierarchy
- `privateTeamScenario` - Private team with restricted access
- `largeTeamScenario` - Large team with multiple admins
- `invitationTeamScenario` - Team with pending invitations
- `nestedTeamHierarchyScenario` - Parent organization with sub-teams
- `crossTeamScenarios` - User with roles in multiple teams

### 4. **Session Helpers** (`sessionHelpers.ts`)
Quick test session creation utilities:

```typescript
// Quick session creators
const { req, res } = await quickSession.admin();
const { req, res } = await quickSession.standard();
const { req, res } = await quickSession.readOnly();

// Custom sessions
const { req, res } = await quickSession.withUser(customUser);
const { req, res } = await quickSession.withApiKey(customApiKey);

// Test helpers
await expectPermissionDenied(() => endpoint(input, session));
await expectPermissionGranted(() => endpoint(input, session));

// Test permission changes (before/after)
await testPermissionChange(
    (session) => endpoint(input, session),
    standardUser,                    // Before: standard user
    expiredPremiumUser,             // After: expired premium
    { beforeShouldPass: false, afterShouldPass: false }
);

// Bulk permission testing
await testBulkPermissions(
    [
        { name: "create", fn: (s) => createEndpoint(input, s) },
        { name: "update", fn: (s) => updateEndpoint(input, s) },
    ],
    [
        { name: "admin", session: adminUser },
        { name: "apiKey", session: writeApiKey, isApiKey: true },
    ],
    {
        create: { admin: true, apiKey: true },
        update: { admin: true, apiKey: false },
    }
);
```

### 5. **Edge Cases** (`edgeCases.ts`)
Security and edge case testing:

```typescript
import { 
    bannedUser,
    expiredSession,
    hijackedSession,
    rateLimitedUser 
} from "@test/fixtures/permissions/edgeCases";
```

Available edge cases:
- `expiredSession` - Expired authentication
- `partialUser` - User with missing data
- `conflictingPermissionsUser` - Multiple conflicting roles
- `maxPermissionsUser` - User with all possible permissions
- `deletingUser` - Account in deletion process
- `rateLimitedUser` - Rate limit exceeded
- `hijackedSession` - CSRF mismatch
- `specialCharsUser` - XSS attempt in user data
- `malformedSession` - Session missing required fields
- `invalidTypeSession` - Session with wrong data types

### 6. **Integration Scenarios** (`integrationScenarios.ts`)
Multi-actor permission tests:

```typescript
import { teamResourceScenario } from "@test/fixtures/permissions/integrationScenarios";

// Test shows how team owner, admin, member, and non-member
// interact with a private team resource
const tests = generatePermissionTests(teamResourceScenario);
// Automatically generates 12 test cases (3 actions × 4 actors)
```

## Real-World Examples

### Testing Bookmark List Permissions

```typescript
import { 
    standardUser,
    quickSession,
    testPermissionMatrix 
} from "@test/fixtures/permissions";

describe("BookmarkList Permissions", () => {
    it("enforces owner-only access to private lists", async () => {
        // Create a private bookmark list owned by standard user
        const list = await createBookmarkList({
            ownerId: standardUser.id,
            isPrivate: true
        });

        // Test access matrix automatically
        await testPermissionMatrix(
            async (session) => {
                return bookmarkList.findOne(
                    { input: { id: list.id } },
                    session,
                    bookmarkList_findOne
                );
            },
            {
                admin: true,      // Admins can access anything
                standard: true,   // Owner can access their own
                premium: false,   // Other users cannot
                guest: false,     // Guests cannot
                readOnly: false,  // API keys cannot
            }
        );
    });
});
```

### Testing Team Permissions

```typescript
describe("Team Resource Access", () => {
    it("respects team role hierarchy", async () => {
        const scenario = basicTeamScenario;
        
        for (const member of scenario.members) {
            const session = await createSession(member.user);
            
            if (member.role === "Owner") {
                await expectPermissionGranted(() => deleteResource(session));
            } else {
                await expectPermissionDenied(() => deleteResource(session));
            }
        }
    });
});
```

### Security Testing

```typescript
describe("Security Edge Cases", () => {
    it("prevents access from banned users", async () => {
        const { req, res } = await createSession(bannedUser);
        await expectPermissionDenied(
            () => endpoint({ input }, { req, res }),
            /Account is locked/
        );
    });

    it("validates CSRF tokens", async () => {
        const hijacked = await createSession(hijackedSession);
        expect(() => validateCSRF(hijacked.req)).toThrow();
    });
});
```

## Best Practices

1. **Use Consistent Personas**: Always use the predefined personas instead of creating custom users
2. **Test Permission Matrices**: Use `testPermissionMatrix` for comprehensive coverage
3. **Include Edge Cases**: Don't forget to test banned users, expired sessions, etc.
4. **Test Both Positive and Negative**: Verify both allowed and denied access
5. **Use Type-Safe Helpers**: Leverage TypeScript for catching permission errors early

## See Also

- `example.test.ts` - Complete working examples
- Individual fixture files for detailed documentation
- Server endpoint tests for real-world usage patterns

## Ideal Permission Fixture Architecture

### Vision: Comprehensive Security Testing Layer

Permission fixtures should evolve to become the **definitive security testing foundation** for Vrooli, providing:

1. **Type-Safe Permission Factory Pattern**
```typescript
interface PermissionFixtureFactory<TSession, TContext> {
  // User persona management
  personas: Record<string, TSession>;
  scenarios: Record<string, TContext>;
  
  // Factory methods
  createSession: (persona: string, overrides?: Partial<TSession>) => TSession;
  createContext: (session: TSession, permissions?: string[]) => TContext;
  
  // Permission validation
  hasPermission: (context: TContext, permission: string) => boolean;
  canAccess: (context: TContext, resource: any) => boolean;
  
  // Complex scenarios
  createMultiUserScenario: (config: MultiUserConfig) => MultiUserContext;
  simulateTeamInteraction: (scenario: TeamScenario) => TestResult;
  
  // Performance testing
  measurePermissionCheck: (context: TContext, operation: () => any) => PerformanceMetrics;
}
```

2. **Enhanced Permission Categories**

### Authentication Providers
```typescript
// OAuth/SSO providers
export const oauthProviders = {
  google: createOAuthSession("google", { scope: ["email", "profile"] }),
  github: createOAuthSession("github", { scope: ["user", "repo"] }),
  microsoft: createOAuthSession("microsoft", { scope: ["User.Read"] }),
};

// Multi-factor authentication
export const mfaScenarios = {
  totp: createMFASession("totp", { verified: true }),
  sms: createMFASession("sms", { phoneVerified: true }),
  biometric: createMFASession("biometric", { deviceTrusted: true }),
};
```

### Permission Inheritance
```typescript
// Cascading permissions
export const inheritanceScenarios = {
  // Organization → Team → Project hierarchy
  organizationHierarchy: {
    org: { permissions: ["manage_all"] },
    team: { inherits: "org", adds: ["team_specific"] },
    project: { inherits: "team", restricts: ["delete"] },
  },
  
  // Role-based inheritance
  roleHierarchy: {
    superAdmin: { permissions: ["*"] },
    admin: { inherits: "superAdmin", excludes: ["system_config"] },
    moderator: { inherits: "admin", excludes: ["user_management"] },
  },
};
```

### Rate Limiting & Quotas
```typescript
// Comprehensive rate limiting
export const rateLimitingFixtures = {
  // API rate limits
  apiLimits: {
    free: { requests: 100, window: "1h", burst: 10 },
    premium: { requests: 10000, window: "1h", burst: 100 },
    enterprise: { requests: -1, window: null, burst: -1 }, // Unlimited
  },
  
  // Resource quotas
  resourceQuotas: {
    storage: { free: "1GB", premium: "100GB", enterprise: "unlimited" },
    aiCredits: { free: 100, premium: 10000, enterprise: 100000 },
    teamMembers: { free: 5, premium: 50, enterprise: -1 },
  },
  
  // Testing utilities
  simulateRateLimit: async (user: AuthenticatedSessionData, requests: number) => {
    // Simulate hitting rate limits
  },
};
```

### Security Boundaries
```typescript
// Security edge cases and attack vectors
export const securityBoundaries = {
  // Input validation
  injectionAttempts: {
    sql: "'; DROP TABLE users; --",
    xss: "<script>alert('xss')</script>",
    pathTraversal: "../../../etc/passwd",
    commandInjection: "; rm -rf /",
  },
  
  // Session attacks
  sessionAttacks: {
    hijacking: createHijackedSession(),
    fixation: createFixatedSession(),
    replay: createReplayAttack(),
  },
  
  // Permission escalation
  escalationAttempts: {
    roleManipulation: attemptRoleChange("user", "admin"),
    scopeExpansion: attemptScopeExpansion(["read"], ["write"]),
    resourceOwnership: attemptOwnershipTakeover(resource),
  },
};
```

### Performance Testing
```typescript
// Permission check performance
export const performanceFixtures = {
  // Large permission sets
  complexPermissions: {
    manyRoles: createUserWithRoles(Array(100).fill("role")),
    deepHierarchy: createNestedTeamHierarchy(10), // 10 levels deep
    manyResources: createUserWithResources(1000),
  },
  
  // Caching scenarios
  cacheScenarios: {
    cold: { cached: false, expectedTime: ">50ms" },
    warm: { cached: true, expectedTime: "<5ms" },
    invalidated: { cached: "stale", expectedTime: ">50ms" },
  },
};
```

### Round-Trip Integration
```typescript
// Integration with unified fixture architecture
export const roundTripPermissions = {
  // API → DB → UI flow
  fullStackFlow: async (persona: string) => {
    const session = await createSession(personas[persona]);
    const apiResponse = await makeApiCall(session);
    const dbState = await verifyDatabaseState(apiResponse);
    const uiState = await renderWithPermissions(dbState, session);
    return { session, apiResponse, dbState, uiState };
  },
  
  // Permission change propagation
  propagationTest: async (user: AuthenticatedSessionData, newRole: string) => {
    await updateUserRole(user.id, newRole);
    await verifyApiPermissions(user.id, newRole);
    await verifyCacheInvalidation(user.id);
    await verifyUIUpdates(user.id, newRole);
  },
};
```

## Best Practices for Permission Testing

### 1. Use Realistic Scenarios
```typescript
// ✅ Good: Test real-world permission combinations
const projectOwnerInTeam = {
  user: standardUser,
  personalProject: { ownerId: standardUser.id },
  teamProject: { teamId: "team_123", createdById: standardUser.id },
};

// ❌ Bad: Unrealistic permission combinations
const impossibleUser = {
  isGuest: true,
  isAdmin: true, // Guests can't be admins
};
```

### 2. Test Permission Boundaries
```typescript
// Always test both sides of permission boundaries
describe("Resource Access", () => {
  it("allows owner access", async () => {
    const session = await createSession(resourceOwner);
    await expectPermissionGranted(() => accessResource(session));
  });
  
  it("denies non-owner access", async () => {
    const session = await createSession(otherUser);
    await expectPermissionDenied(() => accessResource(session));
  });
});
```

### 3. Verify Permission Cascading
```typescript
// Test that permissions cascade correctly
it("inherits team permissions", async () => {
  const teamMember = await addUserToTeam(user, team, "Member");
  const session = await createSession(teamMember);
  
  // Should have team resource access
  await expectPermissionGranted(() => accessTeamResource(session));
  
  // Should not have team admin access
  await expectPermissionDenied(() => adminTeamAction(session));
});
```

### 4. Test Time-Based Permissions
```typescript
// Test permissions that change over time
it("expires temporary permissions", async () => {
  const tempAdmin = grantTemporaryPermission(user, "admin", "1h");
  
  // Should have permission immediately
  await expectPermissionGranted(() => adminAction(tempAdmin));
  
  // Fast-forward time
  jest.advanceTimersByTime(3600001); // 1 hour + 1ms
  
  // Should no longer have permission
  await expectPermissionDenied(() => adminAction(tempAdmin));
});
```

## Integration with Other Fixtures

### With API Fixtures
```typescript
import { apiFixtures } from "@vrooli/shared/__test/fixtures";
import { adminUser, quickSession } from "./permissions";

// Combine permission and API fixtures
const adminSession = await quickSession.admin();
const apiData = apiFixtures.projectFixtures.complete.create;
const response = await createProject(apiData, adminSession);
```

### With Database Fixtures
```typescript
import { UserDbFactory } from "../db";
import { premiumUser } from "./permissions";

// Create database user matching permission fixture
const dbUser = await UserDbFactory.createComplete({
  id: premiumUser.id,
  hasPremium: true,
});
```

### With Event Fixtures
```typescript
import { eventFixtures } from "@vrooli/shared/__test/fixtures";
import { botUser } from "./permissions";

// Test bot permissions with events
const botSession = await createSession(botUser);
await socket.emit(eventFixtures.chat.bot.executeCommand, botSession);
```

## Future Enhancements

1. **OAuth Provider Integration** - Support for external authentication
2. **Biometric Authentication** - Device-based authentication fixtures
3. **Blockchain Permissions** - Smart contract based permissions
4. **AI-Driven Permissions** - Dynamic permission assignment based on behavior
5. **Compliance Testing** - GDPR, CCPA, HIPAA compliance scenarios
6. **Cross-Platform Sessions** - Mobile, desktop, API unified sessions

## See Also

- `example.test.ts` - Complete working examples
- Individual fixture files for detailed documentation
- Server endpoint tests for real-world usage patterns
- [Fixtures Overview](/docs/testing/fixtures-overview.md) - Unified fixture architecture
- [Round-Trip Testing](/docs/testing/round-trip-testing.md) - Full-stack permission flows