/**
 * Permission & Authorization Fixtures
 * 
 * Centralized permission testing fixtures for the Vrooli platform.
 * These fixtures provide consistent, reusable test data for various
 * permission scenarios across the application.
 */

// User personas - Different types of users with various permission levels
export {
    // Standard personas
    adminUser,
    standardUser,
    premiumUser,
    unverifiedUser,
    bannedUser,
    guestUser,
    botUser,
    customRoleUser,
    suspendedUser,
    expiredPremiumUser,
    
    // Helper functions
    createUserWithPermissions,
    createUserGroup,
} from "./userPersonas.js";

// API key configurations - Various API key permission combinations
export {
    // Predefined API keys
    readOnlyPublicApiKey,
    readPrivateApiKey,
    writeApiKey,
    botApiKey,
    externalApiKey,
    rateLimitedApiKey,
    authReadApiKey,
    authWriteApiKey,
    expiredApiKey,
    revokedApiKey,
    
    // Helper functions
    createApiKeyWithPermissions,
    createApiKeySet,
} from "./apiKeyPermissions.js";

// Team scenarios - Team membership and role configurations
export {
    // Predefined scenarios
    basicTeamScenario,
    privateTeamScenario,
    largeTeamScenario,
    invitationTeamScenario,
    nestedTeamHierarchyScenario,
    crossTeamScenarios,
    
    // Helper functions
    createTeamScenario,
    
    // Types
    type TeamScenario,
} from "./teamScenarios.js";

// Edge cases - Unusual scenarios and stress tests
export {
    // Edge case users
    expiredSession,
    partialUser,
    conflictingPermissionsUser,
    maxPermissionsUser,
    deletingUser,
    rateLimitedUser,
    corsTestUser,
    hijackedSession,
    specialCharsUser,
    malformedSession,
    invalidTypeSession,
    
    // Grouped scenarios
    concurrentSessions,
    permissionInheritance,
    timeBasedPermissions,
    
    // Helper functions
    createEdgeCaseUser,
} from "./edgeCases.js";

// Integration scenarios - Complex multi-actor permission tests
export {
    // Scenarios
    publicResourceScenario,
    teamResourceScenario,
    crossTeamCollaborationScenario,
    botAutomationScenario,
    escalationAttemptScenario,
    
    // Helper functions
    generatePermissionTests,
    
    // Types
    type IntegrationScenario,
} from "./integrationScenarios.js";

// Session helpers - Utilities for creating test sessions
export {
    // Session creators
    createSession,
    createApiKeySession,
    createMultipleSessions,
    quickSession,
    
    // Test helpers
    expectPermissionDenied,
    expectPermissionGranted,
    testPermissionMatrix,
    testPermissionChange,
    testBulkPermissions,
} from "./sessionHelpers.js";

/**
 * Quick reference for common test patterns:
 * 
 * 1. Test basic authentication:
 *    const { req, res } = await quickSession.standard();
 * 
 * 2. Test admin privileges:
 *    const { req, res } = await quickSession.admin();
 * 
 * 3. Test API key access:
 *    const { req, res } = await quickSession.readOnly();
 * 
 * 4. Test permission matrix:
 *    await testPermissionMatrix(
 *      async (session) => endpoint(input, session),
 *      {
 *        admin: true,
 *        standard: false,
 *        guest: false,
 *      }
 *    );
 * 
 * 5. Test team permissions:
 *    const scenario = basicTeamScenario;
 *    const ownerSession = await createSession(scenario.members[0].user);
 */