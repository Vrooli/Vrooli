/* c8 ignore start */
/**
 * API Fixtures - Validation and Transformation Layer
 * 
 * This module provides type-safe fixtures for all API operations, with integration
 * for validation schemas and shape transformation functions.
 * 
 * @example Basic usage:
 * ```typescript
 * import { userAPIFixtures } from "@vrooli/shared";
 * 
 * const user = userAPIFixtures.createFactory({ name: "Test User" });
 * const validation = await userAPIFixtures.validateCreate(user);
 * ```
 * 
 * @example With shape transformation:
 * ```typescript
 * const formData = { userName: "Test", private: "yes" };
 * const apiInput = userAPIFixtures.transformToAPI(formData);
 * ```
 */

// ========================================
// Core Infrastructure
// ========================================

export * from "./types.js";
export * from "./BaseAPIFixtureFactory.js";
export * from "./integrationUtils.js";

// ========================================
// Legacy Fixture Exports (Individual)
// ========================================

export * from "./apiKeyExternalFixtures.js";
export * from "./apiKeyFixtures.js";
export * from "./bookmarkFixtures.js";
export * from "./bookmarkListFixtures.js";
export * from "./botFixtures.js";
export * from "./chatFixtures.js";
export * from "./chatInviteFixtures.js";
export * from "./chatMessageFixtures.js";
export * from "./chatParticipantFixtures.js";
export * from "./commentFixtures.js";
export * from "./emailFixtures.js";
export * from "./issueFixtures.js";
export * from "./meetingFixtures.js";
export * from "./meetingInviteFixtures.js";
export * from "./memberFixtures.js";
export * from "./memberInviteFixtures.js";
export * from "./notificationSubscriptionFixtures.js";
export * from "./phoneFixtures.js";
export * from "./pullRequestFixtures.js";
export * from "./pushDeviceFixtures.js";
export * from "./reminderFixtures.js";
export * from "./reminderItemFixtures.js";
export * from "./reminderListFixtures.js";
export * from "./reportFixtures.js";
export * from "./reportResponseFixtures.js";
export * from "./resourceFixtures.js";
export * from "./resourceVersionFixtures.js";
export * from "./resourceVersionRelationFixtures.js";
export * from "./runFixtures.js";
export * from "./runIOFixtures.js";
export * from "./runStepFixtures.js";
export * from "./scheduleExceptionFixtures.js";
export * from "./scheduleFixtures.js";
export * from "./scheduleRecurrenceFixtures.js";
export * from "./tagFixtures.js";
export * from "./teamFixtures.js";
export * from "./transferFixtures.js";
export * from "./userFixtures.js";
export * from "./walletFixtures.js";

// ========================================
// Type-Safe API Fixture Factories
// ========================================

// Modern type-safe factories with zero `any` types and validation integration
export * from "./factories/userAPIFixtures.js";
export * from "./factories/teamAPIFixtures.js";
export * from "./factories/bookmarkAPIFixtures.js";
export * from "./factories/projectAPIFixtures.js";

// TODO: Implement type-safe factories for remaining objects
// ... (all other objects)

// ========================================
// Namespace Export for Organized Access
// ========================================

// Import legacy fixtures for namespace
import { userFixtures, userTranslationFixtures } from "./userFixtures.js";
import { teamFixtures } from "./teamFixtures.js";
import { bookmarkFixtures } from "./bookmarkFixtures.js";
import { bookmarkListFixtures } from "./bookmarkListFixtures.js";
import { botFixtures } from "./botFixtures.js";
import { chatFixtures } from "./chatFixtures.js";
import { commentFixtures } from "./commentFixtures.js";
// Import modern type-safe factories
import { teamAPIFixtures } from "./factories/teamAPIFixtures.js";
import { userAPIFixtures } from "./factories/userAPIFixtures.js";
import { bookmarkAPIFixtures } from "./factories/bookmarkAPIFixtures.js";
import { projectAPIFixtures } from "./factories/projectAPIFixtures.js";
// ... add more as needed

/**
 * Namespace export for organized access to all API fixtures
 * 
 * @example Legacy fixtures:
 * ```typescript
 * import { apiFixtures } from "@vrooli/shared";
 * 
 * const user = apiFixtures.userFixtures.minimal.create;
 * const team = apiFixtures.teamFixtures.complete.create;
 * ```
 * 
 * @example Modern type-safe factories:
 * ```typescript
 * import { apiFixtures } from "@vrooli/shared";
 * 
 * const user = apiFixtures.userAPIFixtures.createFactory({ name: "Test" });
 * const team = apiFixtures.teamAPIFixtures.createPublicTeam("My Team");
 * const validation = await apiFixtures.teamAPIFixtures.validateCreate(team);
 * ```
 */
export const apiFixtures = {
    // Legacy User-related fixtures
    userFixtures,
    userTranslationFixtures,
    botFixtures,

    // Legacy Team-related fixtures  
    teamFixtures,

    // Legacy Bookmark-related fixtures
    bookmarkFixtures,
    bookmarkListFixtures,

    // Legacy Chat-related fixtures
    chatFixtures,

    // Legacy Content fixtures
    commentFixtures,

    // Modern Type-Safe API Factories
    userAPIFixtures,
    teamAPIFixtures,
    bookmarkAPIFixtures,
    projectAPIFixtures,

    // TODO: Add all other fixtures to namespace
    // This provides organized access while maintaining backward compatibility
};
