/**
 * Central export for all database fixtures
 * These fixtures follow Prisma's shape for database operations
 * Enhanced with organized namespaces and utility functions
 */

// Core fixture types and utilities
export * from "./types.js";
export { EnhancedDbFactory } from "./EnhancedDbFactory.js";

export * from "./apiKeyFixtures.js";
export * from "./apiKeyExternalFixtures.js";
export * from "./awardFixtures.js";
export * from "./creditAccountFixtures.js";
export * from "./creditLedgerEntryFixtures.js";
export * from "./bookmarkFixtures.js";
export * from "./bookmarkListFixtures.js";
export * from "./chatFixtures.js";
export * from "./chatInviteFixtures.js";
export * from "./chatMessageFixtures.js";
export * from "./chatParticipantFixtures.js";
export * from "./commentFixtures.js";
export * from "./issueFixtures.js";
export * from "./meetingFixtures.js";
export * from "./meetingInviteFixtures.js";
export * from "./memberFixtures.js";
export * from "./memberInviteFixtures.js";
export * from "./notificationFixtures.js";
export * from "./paymentFixtures.js";
export * from "./planFixtures.js";
export * from "./premiumFixtures.js";
export * from "./pullRequestFixtures.js";
export * from "./pushDeviceFixtures.js";
export * from "./reactionFixtures.js";
export * from "./reminderFixtures.js";
export * from "./reminderItemFixtures.js";
export * from "./reminderListFixtures.js";
export * from "./reportFixtures.js";
export * from "./reportResponseFixtures.js";
export * from "./reputationHistoryFixtures.js";
export * from "./resourceFixtures.js";
export * from "./resourceVersionFixtures.js";
export * from "./resourceVersionRelationFixtures.js";
export * from "./runFixtures.js";
export * from "./runIOFixtures.js";
export * from "./runStepFixtures.js";
export * from "./scheduleFixtures.js";
export * from "./scheduleExceptionFixtures.js";
export * from "./scheduleRecurrenceFixtures.js";
export * from "./sessionFixtures.js";
export * from "./statsFixtures.js";
export * from "./tagFixtures.js";
export * from "./teamFixtures.js";
export * from "./transferFixtures.js";
export * from "./userFixtures.js";
export * from "./viewFixtures.js";
export * from "./walletFixtures.js";

// Namespace exports for organized access
import { 
    userDbFixtures, 
    UserDbFactory, 
    seedTestUsers,
    createSessionUser,
    userDbIds
} from "./userFixtures.js";

/**
 * Organized namespace exports for better developer experience
 * Usage: import { userDb } from "../fixtures/db";
 */
export const userDb = {
    fixtures: userDbFixtures,
    factory: UserDbFactory,
    seed: seedTestUsers,
    createSession: createSessionUser,
    ids: userDbIds,
};

/**
 * Utility functions for cross-fixture operations
 */
export const dbFixtureUtils = {
    /**
     * Get all available fixture categories across all models
     */
    getAllCategories(): string[] {
        return ['minimal', 'complete', 'invalid', 'edgeCase', 'error'];
    },

    /**
     * Get all enhanced factories
     */
    getEnhancedFactories(): Record<string, any> {
        return {
            user: UserDbFactory,
            // Add more as they're enhanced
        };
    },

    /**
     * Validate fixture data across all models
     */
    validateFixture(modelName: string, data: any): boolean {
        const factories = this.getEnhancedFactories();
        const factory = factories[modelName];
        
        if (!factory) {
            console.warn(`No enhanced factory found for model: ${modelName}`);
            return false;
        }

        const instance = new factory();
        const validation = instance.validateFixture(data);
        
        if (!validation.isValid) {
            console.error(`Fixture validation failed for ${modelName}:`, validation.errors);
        }

        return validation.isValid;
    },
};

/**
 * Common test data patterns
 */
export const commonTestPatterns = {
    /**
     * Create a basic test environment with user authentication
     */
    createBasicEnvironment() {
        const userFactory = new UserDbFactory();
        return {
            user: userFactory.createWithRelationships({ withAuth: true }).data,
        };
    },

    /**
     * Create test data for permission testing
     */
    createPermissionTestData() {
        const userFactory = new UserDbFactory();
        
        return {
            admin: userFactory.createWithRelationships({ 
                withAuth: true,
                withRoles: [{ id: "admin-role", name: "Admin" }]
            }).data,
            member: userFactory.createWithRelationships({ withAuth: true }).data,
            guest: userFactory.createMinimal(),
        };
    },

    /**
     * Create edge case test data
     */
    createEdgeCaseData() {
        const userFactory = new UserDbFactory();
        
        return {
            maxLengthUser: userFactory.createEdgeCase("maxLengthHandle"),
            lockedUser: userFactory.createEdgeCase("lockedAccount"),
            deletedUser: userFactory.createEdgeCase("deletedAccount"),
            complexBot: userFactory.createEdgeCase("botWithComplexSettings"),
            multilingualUser: userFactory.createEdgeCase("multiLanguageTranslations"),
        };
    },
};

/**
 * Error scenario helpers
 */
export const errorScenarios = {
    /**
     * Get constraint violation scenarios for all models
     */
    getConstraintViolations() {
        return {
            user: {
                uniqueViolation: new UserDbFactory().createInvalid("uniqueViolation"),
                foreignKeyViolation: new UserDbFactory().createInvalid("foreignKeyViolation"),
            },
        };
    },

    /**
     * Get validation error scenarios for all models
     */
    getValidationErrors() {
        return {
            user: {
                missingRequired: new UserDbFactory().createInvalid("missingRequired"),
                invalidTypes: new UserDbFactory().createInvalid("invalidTypes"),
            },
        };
    },
};