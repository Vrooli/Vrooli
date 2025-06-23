/**
 * Central export for all database fixtures
 * These fixtures follow Prisma's shape for database operations
 * Enhanced with organized namespaces and utility functions
 */

// Core fixture types and utilities
export * from "./types.js";
export { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
export { DatabaseFactoryRegistry, getFactoryRegistry, resetFactoryRegistry } from "./DatabaseFactoryRegistry.js";

// Enhanced database factories - Core Objects
export { UserDbFactory, createUserDbFactory } from "./UserDbFactory.js";
export { TeamDbFactory, createTeamDbFactory } from "./TeamDbFactory.js";

// Content Objects
export { BookmarkDbFactory, createBookmarkDbFactory } from "./BookmarkDbFactory.js";
export { BookmarkListDbFactory, createBookmarkListDbFactory } from "./BookmarkListDbFactory.js";
export { CommentDbFactory, createCommentDbFactory } from "./CommentDbFactory.js";
export { IssueDbFactory, createIssueDbFactory } from "./IssueDbFactory.js";
export { TagDbFactory, createTagDbFactory } from "./TagDbFactory.js";

// User Extension Objects
export { AuthDbFactory, createAuthDbFactory } from "./AuthDbFactory.js";
export { EmailDbFactory, createEmailDbFactory } from "./EmailDbFactory.js";
export { PhoneDbFactory, createPhoneDbFactory } from "./PhoneDbFactory.js";
export { PaymentDbFactory, createPaymentDbFactory } from "./PaymentDbFactory.js";
export { SessionDbFactory, createSessionDbFactory } from "./SessionDbFactory.js";

// Communication Objects
export { ChatDbFactory, createChatDbFactory } from "./ChatDbFactory.js";
export { ChatInviteDbFactory, createChatInviteDbFactory } from "./ChatInviteDbFactory.js";
export { ChatMessageDbFactory, createChatMessageDbFactory } from "./ChatMessageDbFactory.js";
export { ChatParticipantDbFactory, createChatParticipantDbFactory } from "./ChatParticipantDbFactory.js";
export { MeetingDbFactory, createMeetingDbFactory } from "./MeetingDbFactory.js";
export { MeetingInviteDbFactory, createMeetingInviteDbFactory } from "./MeetingInviteDbFactory.js";
export { NotificationDbFactory, createNotificationDbFactory } from "./NotificationDbFactory.js";

// Interaction Objects
export { ReactionDbFactory, createReactionDbFactory } from "./ReactionDbFactory.js";
export { ReactionSummaryDbFactory, createReactionSummaryDbFactory } from "./ReactionSummaryDbFactory.js";
export { ViewDbFactory, createViewDbFactory } from "./ViewDbFactory.js";
export { ReportDbFactory, createReportDbFactory } from "./ReportDbFactory.js";
export { ReportResponseDbFactory, createReportResponseDbFactory } from "./ReportResponseDbFactory.js";

// Billing Objects
export { CreditAccountDbFactory, createCreditAccountDbFactory } from "./CreditAccountDbFactory.js";

// Team Management Objects
export { MemberDbFactory, createMemberDbFactory } from "./MemberDbFactory.js";

// Resource management factories
export { ResourceDbFactory, createResourceDbFactory } from "./ResourceDbFactory.js";
export { ResourceVersionDbFactory, createResourceVersionDbFactory } from "./ResourceVersionDbFactory.js";
export { ResourceVersionRelationDbFactory, createResourceVersionRelationDbFactory } from "./ResourceVersionRelationDbFactory.js";
export { ProjectDbFactory, createProjectDbFactory } from "./ProjectDbFactory.js";
export { ProjectVersionDbFactory, createProjectVersionDbFactory } from "./ProjectVersionDbFactory.js";
export { RoutineDbFactory, createRoutineDbFactory } from "./RoutineDbFactory.js";
export { RoutineVersionDbFactory, createRoutineVersionDbFactory } from "./RoutineVersionDbFactory.js";

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
export * from "./ScheduleExceptionDbFactory.js";
export * from "./ScheduleRecurrenceDbFactory.js";

// Schedule and Execution Objects
export { ScheduleDbFactory, createScheduleDbFactory } from "./ScheduleDbFactory.js";
export { RunDbFactory, createRunDbFactory } from "./RunDbFactory.js";
export { RunIODbFactory, createRunIODbFactory } from "./RunIODbFactory.js";
export { RunStepDbFactory, createRunStepDbFactory } from "./RunStepDbFactory.js";
export { ReminderDbFactory, createReminderDbFactory } from "./ReminderDbFactory.js";
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
    userDbIds,
} from "./userFixtures.js";
import { BookmarkDbFactory, createBookmarkDbFactory } from "./BookmarkDbFactory.js";
import { BookmarkListDbFactory, createBookmarkListDbFactory } from "./BookmarkListDbFactory.js";
import { CommentDbFactory, createCommentDbFactory } from "./CommentDbFactory.js";
import { IssueDbFactory, createIssueDbFactory } from "./IssueDbFactory.js";
import { TagDbFactory, createTagDbFactory } from "./TagDbFactory.js";
import { AuthDbFactory, createAuthDbFactory } from "./AuthDbFactory.js";
import { EmailDbFactory, createEmailDbFactory } from "./EmailDbFactory.js";
import { PhoneDbFactory, createPhoneDbFactory } from "./PhoneDbFactory.js";
import { PaymentDbFactory, createPaymentDbFactory } from "./PaymentDbFactory.js";
import { SessionDbFactory, createSessionDbFactory } from "./SessionDbFactory.js";
import { ResourceDbFactory, createResourceDbFactory } from "./ResourceDbFactory.js";
import { ResourceVersionDbFactory, createResourceVersionDbFactory } from "./ResourceVersionDbFactory.js";
import { ResourceVersionRelationDbFactory, createResourceVersionRelationDbFactory } from "./ResourceVersionRelationDbFactory.js";
import { ProjectDbFactory, createProjectDbFactory } from "./ProjectDbFactory.js";
import { ProjectVersionDbFactory, createProjectVersionDbFactory } from "./ProjectVersionDbFactory.js";
import { RoutineDbFactory, createRoutineDbFactory } from "./RoutineDbFactory.js";
import { RoutineVersionDbFactory, createRoutineVersionDbFactory } from "./RoutineVersionDbFactory.js";
// Schedule and execution object imports
import { ScheduleDbFactory, createScheduleDbFactory } from "./ScheduleDbFactory.js";
import { ScheduleExceptionDbFactory, createScheduleExceptionDbFactory } from "./ScheduleExceptionDbFactory.js";
import { ScheduleRecurrenceEnhancedDbFactory, createScheduleRecurrenceEnhancedDbFactory } from "./ScheduleRecurrenceEnhancedDbFactory.js";
import { RunDbFactory, createRunDbFactory } from "./RunDbFactory.js";
import { RunIODbFactory, createRunIODbFactory } from "./RunIODbFactory.js";
import { RunStepDbFactory, createRunStepDbFactory } from "./RunStepDbFactory.js";
import { ReminderDbFactory, createReminderDbFactory, ReminderListDbFactory, createReminderListDbFactory } from "./ReminderDbFactory.js";
import { CreditAccountDbFactory, createCreditAccountDbFactory } from "./CreditAccountDbFactory.js";
import { MemberDbFactory, createMemberDbFactory } from "./MemberDbFactory.js";

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

export const bookmarkDb = {
    factory: BookmarkDbFactory,
    create: createBookmarkDbFactory,
};

export const bookmarkListDb = {
    factory: BookmarkListDbFactory,
    create: createBookmarkListDbFactory,
};

export const commentDb = {
    factory: CommentDbFactory,
    create: createCommentDbFactory,
};

export const issueDb = {
    factory: IssueDbFactory,
    create: createIssueDbFactory,
};

export const tagDb = {
    factory: TagDbFactory,
    create: createTagDbFactory,
};

export const authDb = {
    factory: AuthDbFactory,
    create: createAuthDbFactory,
};

export const emailDb = {
    factory: EmailDbFactory,
    create: createEmailDbFactory,
};

export const phoneDb = {
    factory: PhoneDbFactory,
    create: createPhoneDbFactory,
};

export const paymentDb = {
    factory: PaymentDbFactory,
    create: createPaymentDbFactory,
};

export const sessionDb = {
    factory: SessionDbFactory,
    create: createSessionDbFactory,
};

export const resourceDb = {
    factory: ResourceDbFactory,
    create: createResourceDbFactory,
};

export const resourceVersionDb = {
    factory: ResourceVersionDbFactory,
    create: createResourceVersionDbFactory,
};

export const resourceVersionRelationDb = {
    factory: ResourceVersionRelationDbFactory,
    create: createResourceVersionRelationDbFactory,
};

export const projectDb = {
    factory: ProjectDbFactory,
    create: createProjectDbFactory,
};

export const projectVersionDb = {
    factory: ProjectVersionDbFactory,
    create: createProjectVersionDbFactory,
};

export const routineDb = {
    factory: RoutineDbFactory,
    create: createRoutineDbFactory,
};

export const routineVersionDb = {
    factory: RoutineVersionDbFactory,
    create: createRoutineVersionDbFactory,
};

// Schedule and execution object namespaces
export const scheduleDb = {
    factory: ScheduleDbFactory,
    create: createScheduleDbFactory,
};

export const scheduleExceptionDb = {
    factory: ScheduleExceptionDbFactory,
    create: createScheduleExceptionDbFactory,
};

export const scheduleRecurrenceDb = {
    factory: ScheduleRecurrenceEnhancedDbFactory,
    create: createScheduleRecurrenceEnhancedDbFactory,
};

export const runDb = {
    factory: RunDbFactory,
    create: createRunDbFactory,
};

export const runIODb = {
    factory: RunIODbFactory,
    create: createRunIODbFactory,
};

export const runStepDb = {
    factory: RunStepDbFactory,
    create: createRunStepDbFactory,
};

export const reminderDb = {
    factory: ReminderDbFactory,
    create: createReminderDbFactory,
};

export const reminderListDb = {
    factory: ReminderListDbFactory,
    create: createReminderListDbFactory,
};

export const creditAccountDb = {
    factory: CreditAccountDbFactory,
    create: createCreditAccountDbFactory,
};

export const memberDb = {
    factory: MemberDbFactory,
    create: createMemberDbFactory,
};

/**
 * Utility functions for cross-fixture operations
 */
export const dbFixtureUtils = {
    /**
     * Get all available fixture categories across all models
     */
    getAllCategories(): string[] {
        return ["minimal", "complete", "invalid", "edgeCase", "error"];
    },

    /**
     * Get all enhanced factories
     */
    getEnhancedFactories(): Record<string, any> {
        return {
            user: UserDbFactory,
            team: TeamDbFactory,
            bookmark: BookmarkDbFactory,
            bookmarkList: BookmarkListDbFactory,
            comment: CommentDbFactory,
            issue: IssueDbFactory,
            tag: TagDbFactory,
            auth: AuthDbFactory,
            email: EmailDbFactory,
            phone: PhoneDbFactory,
            payment: PaymentDbFactory,
            session: SessionDbFactory,
            resource: ResourceDbFactory,
            resourceVersion: ResourceVersionDbFactory,
            resourceVersionRelation: ResourceVersionRelationDbFactory,
            project: ProjectDbFactory,
            projectVersion: ProjectVersionDbFactory,
            routine: RoutineDbFactory,
            routineVersion: RoutineVersionDbFactory,
            // Schedule and execution objects
            schedule: ScheduleDbFactory,
            scheduleException: ScheduleExceptionDbFactory,
            scheduleRecurrence: ScheduleRecurrenceEnhancedDbFactory,
            run: RunDbFactory,
            runIO: RunIODbFactory,
            runStep: RunStepDbFactory,
            reminder: ReminderDbFactory,
            reminderList: ReminderListDbFactory,
            creditAccount: CreditAccountDbFactory,
            member: MemberDbFactory,
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
                withRoles: [{ id: "admin-role", name: "Admin" }],
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
