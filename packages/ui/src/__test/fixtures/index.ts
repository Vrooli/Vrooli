/**
 * Production-Grade UI Fixtures - Central Export File
 * 
 * This file provides centralized exports for the complete fixtures-updated architecture.
 * It eliminates the issues found in the legacy fixtures and provides a robust testing foundation.
 */

// ============================================
// CORE TYPES
// ============================================
export * from './types.js';

// ============================================
// FACTORIES
// ============================================
export { 
    BookmarkFixtureFactory,
    bookmarkFixtures,
    bookmarkTestScenarios 
} from './factories/BookmarkFixtureFactory.js';

export {
    UserFixtureFactory,
    userFixtures,
    userTestScenarios
} from './factories/UserFixtureFactory.js';

export {
    TeamFixtureFactory,
    teamFixtures,
    teamTestScenarios
} from './factories/TeamFixtureFactory.js';

export {
    ProjectFixtureFactory,
    projectFixtures,
    projectTestScenarios
} from './factories/ProjectFixtureFactory.js';

export {
    RoutineFixtureFactory,
    routineFixtures,
    routineTestScenarios
} from './factories/RoutineFixtureFactory.js';

export {
    ChatFixtureFactory,
    chatFixtures,
    chatTestScenarios
} from './factories/ChatFixtureFactory.js';

// ============================================
// INTEGRATIONS
// ============================================
export {
    BookmarkIntegrationTest,
    TestAPIClient,
    DatabaseVerifier,
    TestTransactionManager
} from './integrations/bookmarkIntegration.test.js';

// ============================================
// SCENARIOS
// ============================================
export {
    UserBookmarksProjectScenario,
    BookmarkScenarioFactory,
    ScenarioContext,
    AssertionEngine
} from './scenarios/userBookmarksProject.scenario.js';

// ============================================
// UI STATES
// ============================================
export {
    BookmarkStateFactory,
    BookmarkStateTransitionValidator,
    bookmarkStates,
    bookmarkStateValidator,
    bookmarkStateScenarios
} from './states/bookmarkStates.js';

// ============================================
// API RESPONSES
// ============================================

// Bookmark responses (primary example)
export {
    BookmarkResponseFactory,
    BookmarkMSWHandlers,
    bookmarkResponseFactory,
    bookmarkMSWHandlers,
    bookmarkResponseScenarios
} from './api-responses/bookmarkResponses.js';

// ApiKey responses
export {
    ApiKeyResponseFactory,
    ApiKeyMSWHandlers,
    apiKeyResponseFactory,
    apiKeyMSWHandlers,
    apiKeyResponseScenarios
} from './api-responses/ApiKeyResponses.js';

// Award responses
export {
    AwardResponseFactory,
    AwardMSWHandlers,
    awardResponseFactory,
    awardMSWHandlers,
    awardResponseScenarios
} from './api-responses/AwardResponses.js';

// ChatInvite responses
export {
    ChatInviteResponseFactory,
    ChatInviteMSWHandlers,
    chatInviteResponseFactory,
    chatInviteMSWHandlers,
    chatInviteResponseScenarios
} from './api-responses/ChatInviteResponses.js';

// ChatMessage responses
export {
    ChatMessageResponseFactory,
    ChatMessageMSWHandlers,
    chatMessageResponseFactory,
    chatMessageMSWHandlers,
    chatMessageResponseScenarios
} from './api-responses/ChatMessageResponses.js';

// ChatParticipant responses
export {
    ChatParticipantResponseFactory,
    ChatParticipantMSWHandlers,
    chatParticipantResponseFactory,
    chatParticipantMSWHandlers,
    chatParticipantResponseScenarios
} from './api-responses/ChatParticipantResponses.js';

// Issue responses
export {
    IssueResponseFactory,
    IssueMSWHandlers,
    issueResponseFactory,
    issueMSWHandlers,
    issueResponseScenarios
} from './api-responses/IssueResponses.js';

// MeetingInvite responses
export {
    MeetingInviteResponseFactory,
    MeetingInviteMSWHandlers,
    meetingInviteResponseFactory,
    meetingInviteMSWHandlers,
    meetingInviteResponseScenarios
} from './api-responses/MeetingInviteResponses.js';

// MemberInvite responses
export {
    MemberInviteResponseFactory,
    MemberInviteMSWHandlers,
    memberInviteResponseFactory,
    memberInviteMSWHandlers,
    memberInviteResponseScenarios
} from './api-responses/MemberInviteResponses.js';

// Notification responses
export {
    NotificationResponseFactory,
    NotificationMSWHandlers,
    notificationResponseFactory,
    notificationMSWHandlers,
    notificationResponseScenarios
} from './api-responses/NotificationResponses.js';

// Payment responses
export {
    PaymentResponseFactory,
    PaymentMSWHandlers,
    paymentResponseFactory,
    paymentMSWHandlers,
    paymentResponseScenarios
} from './api-responses/PaymentResponses.js';

// Premium responses
export {
    PremiumResponseFactory,
    PremiumMSWHandlers,
    premiumResponseFactory,
    premiumMSWHandlers,
    premiumResponseScenarios
} from './api-responses/PremiumResponses.js';

// PullRequest responses
export {
    PullRequestResponseFactory,
    PullRequestMSWHandlers,
    pullRequestResponseFactory,
    pullRequestMSWHandlers,
    pullRequestResponseScenarios
} from './api-responses/PullRequestResponses.js';

// PushDevice responses
export {
    PushDeviceResponseFactory,
    PushDeviceMSWHandlers,
    pushDeviceResponseFactory,
    pushDeviceMSWHandlers,
    pushDeviceResponseScenarios
} from './api-responses/PushDeviceResponses.js';

// Reaction responses
export {
    ReactionResponseFactory,
    ReactionMSWHandlers,
    reactionResponseFactory,
    reactionMSWHandlers,
    reactionResponseScenarios
} from './api-responses/ReactionResponses.js';

// ReminderItem responses
export {
    ReminderItemResponseFactory,
    ReminderItemMSWHandlers,
    reminderItemResponseFactory,
    reminderItemMSWHandlers,
    reminderItemResponseScenarios
} from './api-responses/ReminderItemResponses.js';

// ReminderList responses
export {
    ReminderListResponseFactory,
    ReminderListMSWHandlers,
    reminderListResponseFactory,
    reminderListMSWHandlers,
    reminderListResponseScenarios
} from './api-responses/ReminderListResponses.js';

// ReportResponse responses
export {
    ReportResponseResponseFactory,
    ReportResponseMSWHandlers,
    reportResponseResponseFactory,
    reportResponseMSWHandlers,
    reportResponseScenarios
} from './api-responses/ReportResponseResponses.js';

// Resource responses
export {
    ResourceResponseFactory,
    ResourceMSWHandlers,
    resourceResponseFactory,
    resourceMSWHandlers,
    resourceResponseScenarios
} from './api-responses/ResourceResponses.js';

// ResourceVersion responses
export {
    ResourceVersionResponseFactory,
    ResourceVersionMSWHandlers,
    resourceVersionResponseFactory,
    resourceVersionMSWHandlers,
    resourceVersionResponseScenarios
} from './api-responses/ResourceVersionResponses.js';

// ResourceVersionRelation responses
export {
    ResourceVersionRelationResponseFactory,
    ResourceVersionRelationMSWHandlers,
    resourceVersionRelationResponseFactory,
    resourceVersionRelationMSWHandlers,
    resourceVersionRelationResponseScenarios
} from './api-responses/ResourceVersionRelationResponses.js';

// Run responses
export {
    RunResponseFactory,
    RunMSWHandlers,
    runResponseFactory,
    runMSWHandlers,
    runResponseScenarios
} from './api-responses/RunResponses.js';

// RunIO responses
export {
    RunIOResponseFactory,
    RunIOMSWHandlers,
    runIOResponseFactory,
    runIOMSWHandlers,
    runIOResponseScenarios
} from './api-responses/RunIOResponses.js';

// RunStep responses
export {
    RunStepResponseFactory,
    RunStepMSWHandlers,
    runStepResponseFactory,
    runStepMSWHandlers,
    runStepResponseScenarios
} from './api-responses/RunStepResponses.js';

// Schedule responses
export {
    ScheduleResponseFactory,
    ScheduleMSWHandlers,
    scheduleResponseFactory,
    scheduleMSWHandlers,
    scheduleResponseScenarios
} from './api-responses/ScheduleResponses.js';

// ScheduleException responses
export {
    ScheduleExceptionResponseFactory,
    ScheduleExceptionMSWHandlers,
    scheduleExceptionResponseFactory,
    scheduleExceptionMSWHandlers,
    scheduleExceptionResponseScenarios
} from './api-responses/ScheduleExceptionResponses.js';

// ScheduleRecurrence responses
export {
    ScheduleRecurrenceResponseFactory,
    ScheduleRecurrenceMSWHandlers,
    scheduleRecurrenceResponseFactory,
    scheduleRecurrenceMSWHandlers,
    scheduleRecurrenceResponseScenarios
} from './api-responses/ScheduleRecurrenceResponses.js';

// StatsResource responses
export {
    StatsResourceResponseFactory,
    StatsResourceMSWHandlers,
    statsResourceResponseFactory,
    statsResourceMSWHandlers,
    statsResourceResponseScenarios
} from './api-responses/StatsResourceResponses.js';

// StatsSite responses
export {
    StatsSiteResponseFactory,
    StatsSiteMSWHandlers,
    statsSiteResponseFactory,
    statsSiteMSWHandlers,
    statsSiteResponseScenarios
} from './api-responses/StatsSiteResponses.js';

// StatsTeam responses
export {
    StatsTeamResponseFactory,
    StatsTeamMSWHandlers,
    statsTeamResponseFactory,
    statsTeamMSWHandlers,
    statsTeamResponseScenarios
} from './api-responses/StatsTeamResponses.js';

// StatsUser responses
export {
    StatsUserResponseFactory,
    StatsUserMSWHandlers,
    statsUserResponseFactory,
    statsUserMSWHandlers,
    statsUserResponseScenarios
} from './api-responses/StatsUserResponses.js';

// Tag responses
export {
    TagResponseFactory,
    TagMSWHandlers,
    tagResponseFactory,
    tagMSWHandlers,
    tagResponseScenarios
} from './api-responses/TagResponses.js';

// Transfer responses
export {
    TransferResponseFactory,
    TransferMSWHandlers,
    transferResponseFactory,
    transferMSWHandlers,
    transferResponseScenarios
} from './api-responses/TransferResponses.js';

// View responses
export {
    ViewResponseFactory,
    ViewMSWHandlers,
    viewResponseFactory,
    viewMSWHandlers,
    viewResponseScenarios
} from './api-responses/ViewResponses.js';

// Wallet responses
export {
    WalletResponseFactory,
    WalletMSWHandlers,
    walletResponseFactory,
    walletMSWHandlers,
    walletResponseScenarios
} from './api-responses/WalletResponses.js';

// ============================================
// FORM DATA
// ============================================
export {
    BookmarkFormDataFactory,
    BookmarkFormInteractionSimulator,
    bookmarkFormFactory,
    bookmarkFormSimulator,
    bookmarkFormScenarios
} from './form-data/bookmarkFormData.js';

// ============================================
// CONVENIENCE EXPORTS
// ============================================

/**
 * Complete fixture testing toolkits
 * Provides all the tools needed for comprehensive testing of each object type
 */
export const bookmarkTestingToolkit = {
    // Data generation
    fixtures: bookmarkFixtures,
    formFactory: bookmarkFormFactory,
    responseFactory: bookmarkResponseFactory,
    stateFactory: bookmarkStates,
    
    // Testing utilities
    integration: {
        createTest: (apiClient?: any, dbVerifier?: any, transactionManager?: any) => 
            new BookmarkIntegrationTest(apiClient, dbVerifier, transactionManager, bookmarkFixtures)
    },
    
    // Form testing
    form: {
        simulator: bookmarkFormSimulator,
        scenarios: bookmarkFormScenarios
    },
    
    // MSW setup
    msw: {
        handlers: bookmarkMSWHandlers,
        scenarios: bookmarkResponseScenarios
    },
    
    // UI state testing
    state: {
        factory: bookmarkStates,
        validator: bookmarkStateValidator,
        scenarios: bookmarkStateScenarios
    },
    
    // Scenario testing
    scenarios: {
        factory: BookmarkScenarioFactory,
        userBookmarksProject: UserBookmarksProjectScenario
    }
};

/**
 * Quick setup functions for common testing patterns
 */
export const testingUtils = {
    /**
     * Setup MSW with bookmark success handlers
     */
    setupMSWSuccess: () => bookmarkMSWHandlers.createSuccessHandlers(),
    
    /**
     * Setup MSW with bookmark error handlers
     */
    setupMSWErrors: () => bookmarkMSWHandlers.createErrorHandlers(),
    
    /**
     * Setup MSW with loading simulation
     */
    setupMSWLoading: (delay: number = 2000) => bookmarkMSWHandlers.createLoadingHandlers(delay),
    
    /**
     * Create a complete bookmark test scenario
     */
    createBookmarkTestScenario: (scenario: 'minimal' | 'complete' | 'withNewList' | 'withExistingList' = 'complete') => ({
        formData: bookmarkFixtures.createFormData(scenario),
        apiInput: bookmarkFixtures.transformToAPIInput(bookmarkFixtures.createFormData(scenario)),
        mockResponse: bookmarkFixtures.createMockResponse(),
        uiState: bookmarkStates.createUIState('success'),
        formState: bookmarkFormScenarios.validForm(),
        mswHandlers: bookmarkMSWHandlers.createSuccessHandlers()
    }),
    
    /**
     * Create error testing scenario
     */
    createErrorTestScenario: (errorType: 'validation' | 'network' | 'permission' | 'server' = 'validation') => ({
        formData: bookmarkFixtures.createFormData('invalid'),
        uiState: bookmarkStates.createUIState('error'),
        formState: bookmarkFormScenarios.formWithErrors(),
        mswHandlers: bookmarkMSWHandlers.createErrorHandlers(),
        errorResponse: (() => {
            switch (errorType) {
                case 'validation':
                    return bookmarkResponseScenarios.validationError();
                case 'network':
                    return bookmarkResponseFactory.createNetworkErrorResponse();
                case 'permission':
                    return bookmarkResponseScenarios.permissionError();
                case 'server':
                    return bookmarkResponseScenarios.serverError();
                default:
                    return bookmarkResponseScenarios.validationError();
            }
        })()
    }),
    
    /**
     * Create form testing utilities for a specific scenario
     */
    createFormTestUtils: (scenario: 'empty' | 'minimal' | 'complete' | 'invalid' = 'complete') => ({
        formData: bookmarkFormFactory.createFormData(scenario),
        formState: bookmarkFormScenarios.validForm(),
        simulator: bookmarkFormSimulator,
        validation: bookmarkFormFactory.validateFormData(bookmarkFormFactory.createFormData(scenario))
    }),
    
    /**
     * Create integration test setup
     */
    createIntegrationTestSetup: () => {
        // In a real implementation, these would be properly initialized
        // For now, we return a structure that shows the intended pattern
        return {
            apiClient: new TestAPIClient(),
            dbVerifier: new DatabaseVerifier(),
            transactionManager: new TestTransactionManager(),
            integration: new BookmarkIntegrationTest(
                new TestAPIClient(),
                new DatabaseVerifier(),
                new TestTransactionManager(),
                bookmarkFixtures
            )
        };
    }
};

/**
 * Type-safe testing patterns
 */
export const typeSafePatterns = {
    /**
     * Validate that a form data object is properly typed
     */
    validateFormDataType: (data: any): data is BookmarkFormData => {
        return (
            typeof data === 'object' &&
            data !== null &&
            typeof data.bookmarkFor === 'string' &&
            typeof data.forConnect === 'string'
        );
    },
    
    /**
     * Validate that a UI state object is properly typed
     */
    validateUIStateType: (state: any): state is ExtendedBookmarkUIState => {
        return (
            typeof state === 'object' &&
            state !== null &&
            typeof state.isLoading === 'boolean' &&
            Array.isArray(state.availableLists) &&
            typeof state.showListSelection === 'boolean'
        );
    },
    
    /**
     * Type guard for API responses
     */
    isAPIResponse: (response: any): response is APIResponse<any> => {
        return (
            typeof response === 'object' &&
            response !== null &&
            response.data !== undefined &&
            response.meta !== undefined &&
            typeof response.meta.timestamp === 'string'
        );
    }
};

/**
 * Best practices documentation accessible from code
 */
export const bestPractices = {
    /**
     * Guidelines for using the fixtures system
     */
    guidelines: {
        doUseRealValidation: "Always use real validation functions from @vrooli/shared",
        doTestCompleteFlow: "Test the complete flow from form to database and back",
        doIsolateTests: "Use transaction isolation to prevent test interference",
        doTestErrorScenarios: "Include comprehensive error scenario testing",
        doUseTypeScript: "Maintain strict TypeScript typing throughout",
        
        dontUseFakeRoundTrips: "Never use fake services that don't hit the database",
        dontUseAnyTypes: "Eliminate all 'any' types in favor of proper interfaces",
        dontCreateGlobalState: "Avoid global state that can pollute tests",
        dontSkipValidation: "Always validate data with real validation schemas",
        dontMockCoreLogic: "Don't mock internal business logic"
    },
    
    /**
     * Example usage patterns
     */
    examples: {
        basicTest: `
// ✅ Good: Using real fixtures with type safety
const formData = bookmarkFixtures.createFormData('complete');
const validation = await bookmarkFixtures.validateFormData(formData);
expect(validation.isValid).toBe(true);
        `,
        
        integrationTest: `
// ✅ Good: Real integration test
const integration = testingUtils.createIntegrationTestSetup();
const result = await integration.integration.testCompleteFlow({
    formData: bookmarkFixtures.createFormData('complete'),
    shouldSucceed: true
});
expect(result.success).toBe(true);
        `,
        
        componentTest: `
// ✅ Good: Component test with fixtures
const { formData, mswHandlers } = testingUtils.createBookmarkTestScenario();
server.use(...mswHandlers);
render(<BookmarkForm initialData={formData} />);
        `
    }
};

/**
 * User testing toolkit
 */
export const userTestingToolkit = {
    fixtures: userFixtures,
    scenarios: userTestScenarios,
    msw: {
        handlers: userFixtures.createMSWHandlers(),
        success: userTestScenarios.successHandlers,
        error: userTestScenarios.errorHandlers,
        loading: userTestScenarios.loadingHandlers
    },
    states: {
        loading: userTestScenarios.loadingState,
        authenticated: userTestScenarios.authenticatedState,
        unauthenticated: userTestScenarios.unauthenticatedState,
        error: userTestScenarios.errorState
    }
};

/**
 * Team testing toolkit
 */
export const teamTestingToolkit = {
    fixtures: teamFixtures,
    scenarios: teamTestScenarios,
    msw: {
        handlers: teamFixtures.createMSWHandlers(),
        success: teamTestScenarios.successHandlers,
        error: teamTestScenarios.errorHandlers,
        loading: teamTestScenarios.loadingHandlers
    },
    states: {
        loading: teamTestScenarios.loadingState,
        memberView: teamTestScenarios.memberViewState,
        ownerView: teamTestScenarios.ownerViewState,
        error: teamTestScenarios.errorState
    }
};

/**
 * Project testing toolkit
 */
export const projectTestingToolkit = {
    fixtures: projectFixtures,
    scenarios: projectTestScenarios,
    msw: {
        handlers: projectFixtures.createMSWHandlers(),
        success: projectTestScenarios.successHandlers,
        error: projectTestScenarios.errorHandlers,
        loading: projectTestScenarios.loadingHandlers
    },
    states: {
        loading: projectTestScenarios.loadingState,
        viewerView: projectTestScenarios.viewerState,
        ownerView: projectTestScenarios.ownerState,
        error: projectTestScenarios.errorState
    }
};

/**
 * Routine testing toolkit
 */
export const routineTestingToolkit = {
    fixtures: routineFixtures,
    scenarios: routineTestScenarios,
    msw: {
        handlers: routineFixtures.createMSWHandlers(),
        success: routineTestScenarios.successHandlers,
        error: routineTestScenarios.errorHandlers,
        loading: routineTestScenarios.loadingHandlers
    },
    states: {
        loading: routineTestScenarios.loadingState,
        success: routineTestScenarios.successState,
        running: routineTestScenarios.runningState,
        error: routineTestScenarios.errorState
    }
};

/**
 * Chat testing toolkit
 */
export const chatTestingToolkit = {
    fixtures: chatFixtures,
    scenarios: chatTestScenarios,
    msw: {
        handlers: chatFixtures.createMSWHandlers(),
        success: chatTestScenarios.successHandlers,
        error: chatTestScenarios.errorHandlers,
        loading: chatTestScenarios.loadingHandlers
    },
    states: {
        loading: chatTestScenarios.loadingState,
        withMessages: chatTestScenarios.withMessagesState,
        typing: chatTestScenarios.typingState,
        error: chatTestScenarios.errorState
    }
};

// Import type definitions for external use
import type { ExtendedBookmarkUIState } from './states/bookmarkStates.js';
import type { APIResponse } from './api-responses/bookmarkResponses.js';
import type { BookmarkFormData, UserFormData, TeamFormData, ProjectFormData, RoutineFormData, ChatFormData } from './types.js';