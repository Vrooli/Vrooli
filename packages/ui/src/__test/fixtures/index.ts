/**
 * Production-Grade UI Fixtures - Central Export File
 * 
 * This file provides centralized exports for the complete fixtures-updated architecture.
 * It eliminates the issues found in the legacy fixtures and provides a robust testing foundation.
 */

// ============================================
// CORE TYPES
// ============================================
export * from "./types.js";

// ============================================
// FACTORIES
// ============================================
export {
    BookmarkFixtureFactory,
    bookmarkFixtures,
    bookmarkTestScenarios,
} from "./factories/BookmarkFixtureFactory.js";

export {
    UserFixtureFactory,
    userFixtures,
    userTestScenarios,
} from "./factories/UserFixtureFactory.js";

export {
    TeamFixtureFactory,
    teamFixtures,
    teamTestScenarios,
} from "./factories/TeamFixtureFactory.js";

export {
    ProjectFixtureFactory,
    projectFixtures,
    projectTestScenarios,
} from "./factories/ProjectFixtureFactory.js";

export {
    RoutineFixtureFactory,
    routineFixtures,
    routineTestScenarios,
} from "./factories/RoutineFixtureFactory.js";

export {
    ChatFixtureFactory,
    chatFixtures,
    chatTestScenarios,
} from "./factories/ChatFixtureFactory.js";

// ============================================
// UI STATES
// ============================================
export {
    BookmarkStateFactory, bookmarkStates, bookmarkStateScenarios, BookmarkStateTransitionValidator, bookmarkStateValidator,
} from "./states/bookmarkStates.js";

// ============================================
// API RESPONSES
// ============================================

// Bookmark responses (primary example)
export {
    BookmarkMSWHandlers, bookmarkMSWHandlers, BookmarkResponseFactory, bookmarkResponseFactory, bookmarkResponseScenarios,
} from "./api-responses/bookmarkResponses.js";

// ApiKey responses
export {
    ApiKeyMSWHandlers, apiKeyMSWHandlers, ApiKeyResponseFactory, apiKeyResponseFactory, apiKeyResponseScenarios,
} from "./api-responses/ApiKeyResponses.js";

// Award responses
export {
    AwardMSWHandlers, awardMSWHandlers, AwardResponseFactory, awardResponseFactory, awardResponseScenarios,
} from "./api-responses/AwardResponses.js";

// ChatInvite responses
export {
    ChatInviteMSWHandlers, chatInviteMSWHandlers, ChatInviteResponseFactory, chatInviteResponseFactory, chatInviteResponseScenarios,
} from "./api-responses/ChatInviteResponses.js";

// ChatMessage responses
export {
    ChatMessageMSWHandlers, chatMessageMSWHandlers, ChatMessageResponseFactory, chatMessageResponseFactory, chatMessageResponseScenarios,
} from "./api-responses/ChatMessageResponses.js";

// ChatParticipant responses
export {
    ChatParticipantMSWHandlers, chatParticipantMSWHandlers, ChatParticipantResponseFactory, chatParticipantResponseFactory, chatParticipantResponseScenarios,
} from "./api-responses/ChatParticipantResponses.js";

// Issue responses
export {
    IssueMSWHandlers, issueMSWHandlers, IssueResponseFactory, issueResponseFactory, issueResponseScenarios,
} from "./api-responses/IssueResponses.js";

// MeetingInvite responses
export {
    MeetingInviteMSWHandlers, meetingInviteMSWHandlers, MeetingInviteResponseFactory, meetingInviteResponseFactory, meetingInviteResponseScenarios,
} from "./api-responses/MeetingInviteResponses.js";

// MemberInvite responses
export {
    MemberInviteMSWHandlers, memberInviteMSWHandlers, MemberInviteResponseFactory, memberInviteResponseFactory, memberInviteResponseScenarios,
} from "./api-responses/MemberInviteResponses.js";

// Notification responses
export {
    NotificationMSWHandlers, notificationMSWHandlers, NotificationResponseFactory, notificationResponseFactory, notificationResponseScenarios,
} from "./api-responses/NotificationResponses.js";

// Payment responses
export {
    PaymentMSWHandlers, paymentMSWHandlers, PaymentResponseFactory, paymentResponseFactory, paymentResponseScenarios,
} from "./api-responses/PaymentResponses.js";

// Premium responses
export {
    PremiumMSWHandlers, premiumMSWHandlers, PremiumResponseFactory, premiumResponseFactory, premiumResponseScenarios,
} from "./api-responses/PremiumResponses.js";

// PullRequest responses
export {
    PullRequestMSWHandlers, pullRequestMSWHandlers, PullRequestResponseFactory, pullRequestResponseFactory, pullRequestResponseScenarios,
} from "./api-responses/PullRequestResponses.js";

// PushDevice responses
export {
    PushDeviceMSWHandlers, pushDeviceMSWHandlers, PushDeviceResponseFactory, pushDeviceResponseFactory, pushDeviceResponseScenarios,
} from "./api-responses/PushDeviceResponses.js";

// Reaction responses
export {
    ReactionMSWHandlers, reactionMSWHandlers, ReactionResponseFactory, reactionResponseFactory, reactionResponseScenarios,
} from "./api-responses/ReactionResponses.js";

// ReminderItem responses
export {
    ReminderItemMSWHandlers, reminderItemMSWHandlers, ReminderItemResponseFactory, reminderItemResponseFactory, reminderItemResponseScenarios,
} from "./api-responses/ReminderItemResponses.js";

// ReminderList responses
export {
    ReminderListMSWHandlers, reminderListMSWHandlers, ReminderListResponseFactory, reminderListResponseFactory, reminderListResponseScenarios,
} from "./api-responses/ReminderListResponses.js";

// ReportResponse responses
export {
    ReportResponseMSWHandlers, reportResponseMSWHandlers, ReportResponseResponseFactory, reportResponseResponseFactory, reportResponseScenarios,
} from "./api-responses/ReportResponseResponses.js";

// Resource responses
export {
    ResourceMSWHandlers, resourceMSWHandlers, ResourceResponseFactory, resourceResponseFactory, resourceResponseScenarios,
} from "./api-responses/ResourceResponses.js";

// ResourceVersion responses
export {
    ResourceVersionMSWHandlers, resourceVersionMSWHandlers, ResourceVersionResponseFactory, resourceVersionResponseFactory, resourceVersionResponseScenarios,
} from "./api-responses/ResourceVersionResponses.js";

// ProjectVersion responses (extends ResourceVersion)
export {
    ProjectVersionMSWHandlers, projectVersionMSWHandlers, ProjectVersionResponseFactory, projectVersionResponseFactory,
} from "./api-responses/ProjectVersionResponses.js";

// ResourceVersionRelation responses
export {
    ResourceVersionRelationMSWHandlers, resourceVersionRelationMSWHandlers, ResourceVersionRelationResponseFactory, resourceVersionRelationResponseFactory, resourceVersionRelationResponseScenarios,
} from "./api-responses/ResourceVersionRelationResponses.js";

// Run responses
export {
    RunMSWHandlers, runMSWHandlers, RunResponseFactory, runResponseFactory, runResponseScenarios,
} from "./api-responses/RunResponses.js";

// RunIO responses
export {
    RunIOMSWHandlers, runIOMSWHandlers, RunIOResponseFactory, runIOResponseFactory, runIOResponseScenarios,
} from "./api-responses/RunIOResponses.js";

// RunStep responses
export {
    RunStepMSWHandlers, runStepMSWHandlers, RunStepResponseFactory, runStepResponseFactory, runStepResponseScenarios,
} from "./api-responses/RunStepResponses.js";

// Schedule responses
export {
    ScheduleMSWHandlers, scheduleMSWHandlers, ScheduleResponseFactory, scheduleResponseFactory, scheduleResponseScenarios,
} from "./api-responses/ScheduleResponses.js";

// ScheduleException responses
export {
    ScheduleExceptionMSWHandlers, scheduleExceptionMSWHandlers, ScheduleExceptionResponseFactory, scheduleExceptionResponseFactory, scheduleExceptionResponseScenarios,
} from "./api-responses/ScheduleExceptionResponses.js";

// ScheduleRecurrence responses
export {
    ScheduleRecurrenceMSWHandlers, scheduleRecurrenceMSWHandlers, ScheduleRecurrenceResponseFactory, scheduleRecurrenceResponseFactory, scheduleRecurrenceResponseScenarios,
} from "./api-responses/ScheduleRecurrenceResponses.js";

// StatsResource responses
export {
    StatsResourceMSWHandlers, statsResourceMSWHandlers, StatsResourceResponseFactory, statsResourceResponseFactory, statsResourceResponseScenarios,
} from "./api-responses/StatsResourceResponses.js";

// StatsSite responses
export {
    StatsSiteMSWHandlers, statsSiteMSWHandlers, StatsSiteResponseFactory, statsSiteResponseFactory, statsSiteResponseScenarios,
} from "./api-responses/StatsSiteResponses.js";

// StatsTeam responses
export {
    StatsTeamMSWHandlers, statsTeamMSWHandlers, StatsTeamResponseFactory, statsTeamResponseFactory, statsTeamResponseScenarios,
} from "./api-responses/StatsTeamResponses.js";

// StatsUser responses
export {
    StatsUserMSWHandlers, statsUserMSWHandlers, StatsUserResponseFactory, statsUserResponseFactory, statsUserResponseScenarios,
} from "./api-responses/StatsUserResponses.js";

// Tag responses
export {
    TagMSWHandlers, tagMSWHandlers, TagResponseFactory, tagResponseFactory, tagResponseScenarios,
} from "./api-responses/TagResponses.js";

// Transfer responses
export {
    TransferMSWHandlers, transferMSWHandlers, TransferResponseFactory, transferResponseFactory, transferResponseScenarios,
} from "./api-responses/TransferResponses.js";

// View responses
export {
    ViewMSWHandlers, viewMSWHandlers, ViewResponseFactory, viewResponseFactory, viewResponseScenarios,
} from "./api-responses/ViewResponses.js";

// Wallet responses
export {
    WalletMSWHandlers, walletMSWHandlers, WalletResponseFactory, walletResponseFactory, walletResponseScenarios,
} from "./api-responses/WalletResponses.js";

// ============================================
// FORM DATA (remaining legacy fixtures)
// ============================================
// Note: All form-data fixtures have been migrated to form-testing/

// ============================================
// FORM TESTING (preferred approach)
// ============================================
export * from "./form-testing/index.js";

/**
 * Type-safe testing patterns
 */
export const typeSafePatterns = {
    /**
     * Validate that a form data object is properly typed
     */
    validateFormDataType: (data: any): data is BookmarkFormData => {
        return (
            typeof data === "object" &&
            data !== null &&
            typeof data.bookmarkFor === "string" &&
            typeof data.forConnect === "string"
        );
    },

    /**
     * Validate that a UI state object is properly typed
     */
    validateUIStateType: (state: any): state is ExtendedBookmarkUIState => {
        return (
            typeof state === "object" &&
            state !== null &&
            typeof state.isLoading === "boolean" &&
            Array.isArray(state.availableLists) &&
            typeof state.showListSelection === "boolean"
        );
    },

    /**
     * Type guard for API responses
     */
    isAPIResponse: (response: any): response is APIResponse<any> => {
        return (
            typeof response === "object" &&
            response !== null &&
            response.data !== undefined &&
            response.meta !== undefined &&
            typeof response.meta.timestamp === "string"
        );
    },
};

// Import type definitions for external use
import type { APIResponse } from "./api-responses/bookmarkResponses.js";
import type { ExtendedBookmarkUIState } from "./states/bookmarkStates.js";
import type { BookmarkFormData } from "./types.js";

