/**
 * Permission & Authorization Fixtures
 * 
 * Enhanced permission testing fixtures using the factory pattern
 * for the unified fixture architecture.
 * 
 * This implementation provides:
 * - Type-safe factories for all permission scenarios
 * - Zero `any` types throughout
 * - Integration with the unified fixture architecture
 * - Comprehensive coverage of all 47+ objects needing permission testing
 */

// Export all types
export * from "./types.js";

// Export factories
export { UserSessionFactory } from "./factories/UserSessionFactory.js";
export { ApiKeyFactory } from "./factories/ApiKeyFactory.js";
export { BasePermissionFactory } from "./factories/BasePermissionFactory.js";

// Export validators
export { PermissionValidator } from "./validators/PermissionValidator.js";

// Export enhanced session helpers
export {
    createSession,
    createApiKeySession,
    createMultipleSessions,
    quickSession,
    expectPermissionDenied,
    expectPermissionGranted,
    testPermissionMatrix,
    testPermissionChange,
    testBulkPermissions,
    checkPermission,
    checkAccess,
    createPermissionContext,
    createMockRequest,
    createMockResponse,
} from "./sessionHelpers.js";

// Create singleton instances for common use
import { UserSessionFactory } from "./factories/UserSessionFactory.js";
import { ApiKeyFactory } from "./factories/ApiKeyFactory.js";
import { PermissionValidator } from "./validators/PermissionValidator.js";

export const userSessionFactory = new UserSessionFactory();
export const apiKeyFactory = new ApiKeyFactory();
export const permissionValidator = new PermissionValidator();

// Export pre-configured personas for backward compatibility
export const adminUser = userSessionFactory.createAdmin();
export const standardUser = userSessionFactory.createStandard();
export const premiumUser = userSessionFactory.createPremium();
export const unverifiedUser = userSessionFactory.createUnverified();
export const bannedUser = userSessionFactory.createBanned();
export const guestUser = userSessionFactory.createGuest();
export const botUser = userSessionFactory.createBot();
export const customRoleUser = userSessionFactory.createWithCustomRole("Custom", ["custom.permission"]);
export const suspendedUser = userSessionFactory.createSuspended();
export const expiredPremiumUser = userSessionFactory.createExpiredPremium();

// Export pre-configured API keys for backward compatibility
export const readOnlyPublicApiKey = apiKeyFactory.createReadOnlyPublic();
export const readPrivateApiKey = apiKeyFactory.createReadPrivate();
export const writeApiKey = apiKeyFactory.createWrite();
export const botApiKey = apiKeyFactory.createBot();
export const externalApiKey = apiKeyFactory.createExternal();
export const rateLimitedApiKey = apiKeyFactory.createRateLimited();
export const authReadApiKey = apiKeyFactory.createAuthRead();
export const authWriteApiKey = apiKeyFactory.createAuthWrite();
export const expiredApiKey = apiKeyFactory.createExpired();
export const revokedApiKey = apiKeyFactory.createRevoked();

// Export helper functions for backward compatibility
export function createUserWithPermissions(permissions: string[]) {
    return userSessionFactory.createWithCustomRole("Dynamic", permissions);
}

export function createUserGroup(size: number) {
    return userSessionFactory.createTeam(size);
}

export function createApiKeyWithPermissions(
    read: "None" | "Public" | "Private" | "Auth",
    write: "None" | "Public" | "Private" | "Auth",
    bot = false,
) {
    return apiKeyFactory.createCustom(read, write, bot);
}

export function createApiKeySet(userId: string) {
    return apiKeyFactory.createSet(userId);
}

// Re-export existing modules for full backward compatibility
export * from "./userPersonas.js";
export * from "./apiKeyPermissions.js";
export * from "./teamScenarios.js";
export * from "./edgeCases.js";
export * from "./integrationScenarios.js";
export * from "./sessionHelpers.js";

/**
 * Quick Start Guide:
 * 
 * 1. Using factories for custom scenarios:
 * ```typescript
 * const customUser = userSessionFactory.createSession({
 *     handle: "custom",
 *     hasPremium: true,
 *     roles: [{ role: { name: "Editor", permissions: JSON.stringify(["content.*"]) }}],
 * });
 * ```
 * 
 * 2. Testing permissions:
 * ```typescript
 * const canEdit = permissionValidator.hasPermission(customUser, "content.edit");
 * const canAccess = permissionValidator.canAccess(customUser, "read", resource);
 * ```
 * 
 * 3. Quick session creation:
 * ```typescript
 * const { req, res } = await quickSession.admin();
 * const { req: apiReq, res: apiRes } = await quickSession.readOnly();
 * ```
 * 
 * 4. Permission matrix testing:
 * ```typescript
 * await testPermissionMatrix(
 *     async (session) => endpoint(input, session),
 *     {
 *         admin: true,
 *         standard: false,
 *         guest: false,
 *         readOnly: false,
 *     }
 * );
 * ```
 * 
 * 5. Creating complex scenarios:
 * ```typescript
 * const teamOwner = userSessionFactory.withTeam(
 *     userSessionFactory.createPremium(),
 *     "team_123",
 *     "Owner"
 * );
 * ```
 */
