/**
 * Central export for all UI fixtures
 * 
 * This module provides the main entry point for the new type-safe fixture system.
 * Legacy exports are maintained for backward compatibility but should be migrated
 * to use the new factory-based system.
 */

// Form data fixtures
export * from "./form-data/bookmarkFormData.js";
export * from "./form-data/userFormData.js";
export * from "./form-data/teamFormData.js";
export * from "./form-data/projectFormData.js";
export * from "./form-data/chatFormData.js";

// API response fixtures  
export * from "./api-responses/bookmarkResponses.js";
export * from "./api-responses/userResponses.js";
export * from "./api-responses/teamResponses.js";
export * from "./api-responses/projectResponses.js";
export * from "./api-responses/chatResponses.js";
export * from "./api-responses/routineResponses.js";

// Transformation utilities
export * from "./helpers/bookmarkTransformations.js";

// Event testing utilities
export * from "./events/index.js";

// UI state fixtures
export * from "./ui-states/loadingStates.js";
export * from "./ui-states/errorStates.js";
export * from "./ui-states/successStates.js";

// Session fixtures
export * from "./sessions/sessionFixtures.js";

// New type-safe fixture system
export * from "./types.js";
export * from "./BaseFormFixtureFactory.js";
export * from "./BaseRoundTripOrchestrator.js";
export * from "./BaseMSWHandlerFactory.js";
export * from "./utils/integration.js";
export * from "./factories/index.js";

// Export specific factory implementations as they are created
export { UserFixtureFactory } from "./factories/userFixtureFactory.js";
export { TagFixtureFactory } from "./factories/tagFixtureFactory.js";
export { TeamFixtureFactory } from "./factories/teamFixtureFactory.js";
export { ChatFixtureFactory } from "./factories/chatFixtureFactory.js";
export { BookmarkFixtureFactory } from "./factories/bookmarkFixtureFactory.js";
export { CommentFixtureFactory } from "./factories/commentFixtureFactory.js";
export { BotFixtureFactory } from "./factories/botFixtureFactory.js";
export { ProjectFixtureFactory } from "./factories/projectFixtureFactory.js";
export { ResourceFixtureFactory } from "./factories/resourceFixtureFactory.js";
export { ScheduleFixtureFactory } from "./factories/scheduleFixtureFactory.js";

/**
 * Quick access to commonly used fixtures
 */
// Bookmark fixtures
export {
    minimalBookmarkFormInput,
    completeBookmarkFormInput,
    bookmarkWithExistingListFormInput,
} from "./form-data/bookmarkFormData.js";

export {
    minimalBookmarkResponse,
    completeBookmarkResponse,
    bookmarkResponseVariants,
} from "./api-responses/bookmarkResponses.js";

export {
    transformFormToCreateRequest,
    transformApiResponseToForm,
    mockBookmarkService,
} from "./helpers/bookmarkTransformations.js";

// User fixtures
export {
    minimalUserResponse,
    completeUserResponse,
    currentUserResponse,
    botUserResponse,
    userResponseVariants,
} from "./api-responses/userResponses.js";

export {
    minimalRegistrationFormInput,
    loginFormInput,
    minimalProfileUpdateFormInput,
} from "./form-data/userFormData.js";

// Team fixtures
export {
    minimalTeamResponse,
    completeTeamResponse,
    teamResponseVariants,
} from "./api-responses/teamResponses.js";

export {
    minimalTeamCreateFormInput,
    completeTeamCreateFormInput,
} from "./form-data/teamFormData.js";

// Project fixtures
export {
    minimalProjectResponse,
    completeProjectResponse,
    projectResponseVariants,
} from "./api-responses/projectResponses.js";

export {
    minimalProjectCreateFormInput,
    completeProjectCreateFormInput,
} from "./form-data/projectFormData.js";

// Chat fixtures
export {
    minimalChatResponse,
    completeChatResponse,
    chatResponseVariants,
} from "./api-responses/chatResponses.js";

export {
    textMessageFormInput,
    directMessageFormInput,
} from "./form-data/chatFormData.js";

// Routine fixtures
export {
    minimalRoutineResponse,
    completeRoutineResponse,
    routineResponseVariants,
} from "./api-responses/routineResponses.js";

// State fixtures
export { loadingStates, componentLoadingStates } from "./ui-states/loadingStates.js";
export { apiErrors, networkErrors, validationErrors } from "./ui-states/errorStates.js";
export { successStates, operationSuccessStates } from "./ui-states/successStates.js";

// Session fixtures
export {
    guestSession,
    authenticatedUserSession,
    premiumUserSession,
    adminUserSession,
} from "./sessions/sessionFixtures.js";