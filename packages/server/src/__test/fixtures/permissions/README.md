# Permission & Authorization Fixtures

This directory contains standardized fixtures for testing different permission scenarios across the Vrooli platform.

## Overview

Instead of recreating permission test scenarios in each test file, these fixtures provide:
- Consistent user personas with predefined permissions
- Common authorization scenarios
- API key configurations with various permission levels
- Team membership scenarios with different roles

## The Permission Testing Challenge

Before this fixture system, permission tests were scattered across endpoint files, making it difficult to:
- Test the same user across different endpoints
- Ensure consistent permission behavior
- Test complex scenarios (team permissions, API keys, etc.)
- Avoid duplicating test user creation logic

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