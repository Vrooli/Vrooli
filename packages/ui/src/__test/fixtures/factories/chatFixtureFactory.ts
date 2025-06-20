/* c8 ignore start */
/**
 * Chat fixture factory implementation
 * 
 * This demonstrates the complete pattern for implementing fixtures for a Chat object type
 * with full type safety and integration with @vrooli/shared.
 */

// Import base types from the main @vrooli/shared package
import type { Chat, ChatCreateInput, ChatUpdateInput } from "@vrooli/shared";

// Import utility functions
import { generatePK } from "@vrooli/shared";

// Import base fixture factory classes
import { BaseFormFixtureFactory } from "../BaseFormFixtureFactory.js";
import { BaseRoundTripOrchestrator } from "../BaseRoundTripOrchestrator.js";
import { BaseMSWHandlerFactory } from "../BaseMSWHandlerFactory.js";
import { createValidationAdapter } from "../utils/integration.js";
import type {
    UIFixtureFactory,
    FormFixtureFactory,
    RoundTripOrchestrator,
    MSWHandlerFactory,
    UIStateFixtureFactory,
    ComponentTestUtils,
    TestAPIClient,
    DatabaseVerifier
} from "../types.js";
import { registerFixture } from "./index.js";

// Create minimal test fixtures since shared fixtures may not be exportable
const basicChatConfig = {
    __version: "1.0.0" as const,
    stats: {
        totalToolCalls: 0,
        totalCredits: "0",
        startedAt: Date.now(),
        lastProcessingCycleEndedAt: null,
    },
};

/**
 * Chat form data type
 * 
 * This includes UI-specific fields that don't exist in the API input type.
 */
export interface ChatFormData {
    name?: string;
    description?: string;
    openToAnyoneWithInvite?: boolean;
    team?: { id: string } | null;
    language?: string;
    chatSettings?: object;
    isDirectMessage?: boolean; // UI-specific
    initialParticipants?: string[]; // UI-specific
    [key: string]: unknown; // Index signature for Record<string, unknown> compatibility
}

/**
 * Chat UI state type
 */
export interface ChatUIState {
    isLoading: boolean;
    chat: Chat | null;
    error: string | null;
    isEditing: boolean;
    hasUnsavedChanges: boolean;
    participants: Array<{ id: string; name: string; handle: string }>;
    messageCount: number;
    isDirectMessage: boolean;
}

/**
 * Chat form fixture factory
 */
class ChatFormFixtureFactory extends BaseFormFixtureFactory<ChatFormData, ChatCreateInput> {
    constructor() {
        super({
            scenarios: {
                minimal: {
                    name: "Test Chat",
                    openToAnyoneWithInvite: false
                },
                complete: {
                    name: "Project Discussion",
                    description: "Chat for discussing project updates and team coordination",
                    openToAnyoneWithInvite: true,
                    team: { id: "423456789012345678" },
                    language: "en",
                    chatSettings: basicChatConfig,
                    isDirectMessage: false,
                    initialParticipants: ["623456789012345678", "623456789012345679"]
                },
                invalid: {
                    name: "", // Empty name
                    description: "x".repeat(3000), // Too long description
                    language: "invalid-lang",
                    team: { id: "invalid-id" }
                },
                directMessage: {
                    name: "Direct Message",
                    description: "Private conversation between two users",
                    openToAnyoneWithInvite: false,
                    language: "en",
                    chatSettings: basicChatConfig,
                    isDirectMessage: true,
                    initialParticipants: ["623456789012345678"]
                },
                teamChat: {
                    name: "Team Collaboration",
                    description: "Team chat for project collaboration",
                    openToAnyoneWithInvite: false,
                    team: { id: "423456789012345678" },
                    language: "en",
                    chatSettings: basicChatConfig,
                    isDirectMessage: false,
                    initialParticipants: ["623456789012345678", "623456789012345679", "623456789012345680"]
                },
                publicChat: {
                    name: "Public Discussion",
                    description: "Open chat for public discussions",
                    openToAnyoneWithInvite: true,
                    language: "en",
                    chatSettings: basicChatConfig,
                    isDirectMessage: false,
                    initialParticipants: []
                }
            },
            
            validate: createValidationAdapter<ChatFormData>(
                async (data: ChatFormData) => {
                    // Additional UI validation
                    const errors: string[] = [];
                    
                    if (!data.name || data.name.trim().length === 0) {
                        errors.push("name: Chat name is required");
                    }
                    
                    if (data.name && data.name.length > 50) {
                        errors.push("name: Chat name must be 50 characters or less");
                    }
                    
                    if (data.description && data.description.length > 2048) {
                        errors.push("description: Description must be 2048 characters or less");
                    }
                    
                    if (data.isDirectMessage && data.initialParticipants && data.initialParticipants.length > 1) {
                        errors.push("initialParticipants: Direct messages can only have one other participant");
                    }
                    
                    if (errors.length > 0) {
                        return { isValid: false, errors };
                    }
                    
                    // Simple validation that always passes for basic implementation
                    return { isValid: true };
                }
            ),
            
            shapeToAPI: (formData) => {
                // Transform to ChatCreateInput
                const apiInput: ChatCreateInput = {
                    id: generatePK().toString(),
                    ...(formData.openToAnyoneWithInvite !== undefined && { 
                        openToAnyoneWithInvite: formData.openToAnyoneWithInvite 
                    }),
                    ...(formData.team && { teamConnect: formData.team.id })
                };

                // Add translations if name or description provided
                if (formData.name || formData.description) {
                    apiInput.translationsCreate = [{
                        id: generatePK().toString(),
                        language: formData.language || "en",
                        ...(formData.name && { name: formData.name }),
                        ...(formData.description && { description: formData.description })
                    }];
                }

                return apiInput;
            }
        });
    }
    
    /**
     * Create direct message form data
     */
    createDirectMessageFormData(otherUserId: string): ChatFormData {
        return {
            name: "Direct Message",
            openToAnyoneWithInvite: false,
            language: "en",
            chatSettings: basicChatConfig,
            isDirectMessage: true,
            initialParticipants: [otherUserId]
        };
    }
    
    /**
     * Create team chat form data
     */
    createTeamChatFormData(teamId: string, chatName: string): ChatFormData {
        return {
            name: chatName,
            description: `Team chat for ${chatName}`,
            openToAnyoneWithInvite: false,
            team: { id: teamId },
            language: "en",
            chatSettings: basicChatConfig,
            isDirectMessage: false,
            initialParticipants: []
        };
    }
}

/**
 * Chat MSW handler factory
 */
class ChatMSWHandlerFactory extends BaseMSWHandlerFactory<ChatCreateInput, ChatUpdateInput, Chat> {
    constructor() {
        super({
            baseUrl: "/api",
            endpoints: {
                create: "/chat",
                update: "/chat",
                delete: "/chat",
                find: "/chat",
                list: "/chats"
            },
            successResponses: {
                create: (input) => ({
                    __typename: "Chat" as const,
                    id: input.id,
                    openToAnyoneWithInvite: input.openToAnyoneWithInvite || false,
                    team: input.teamConnect ? { __typename: "Team" as const, id: input.teamConnect } : null,
                    translations: input.translationsCreate || [],
                    invites: [],
                    messages: [],
                    participants: [],
                    participantsCount: 0,
                    invitesCount: 0,
                    createdAt: new Date().toISOString(),
                    publicId: generatePK().toString()
                } as Chat),
                update: (input) => ({
                    __typename: "Chat" as const,
                    id: input.id,
                    openToAnyoneWithInvite: false,
                    team: null,
                    translations: [],
                    invites: [],
                    messages: [],
                    participants: [],
                    participantsCount: 0,
                    invitesCount: 0,
                    createdAt: new Date().toISOString(),
                    publicId: generatePK().toString(),
                    ...input
                } as Chat),
                find: (id) => ({
                    __typename: "Chat" as const,
                    id,
                    openToAnyoneWithInvite: false,
                    team: null,
                    translations: [],
                    invites: [],
                    messages: [],
                    participants: [],
                    participantsCount: 0,
                    invitesCount: 0,
                    createdAt: new Date().toISOString(),
                    publicId: generatePK().toString()
                } as Chat)
            },
            validate: {
                create: (input) => {
                    const errors: string[] = [];
                    
                    if (!input.id) {
                        errors.push("ID is required");
                    }
                    
                    if (input.translationsCreate) {
                        for (const translation of input.translationsCreate) {
                            if (!translation.language) {
                                errors.push("Translation language is required");
                            }
                            if (translation.name && translation.name.length > 50) {
                                errors.push("Translation name too long");
                            }
                            if (translation.description && translation.description.length > 2048) {
                                errors.push("Translation description too long");
                            }
                        }
                    }
                    
                    return {
                        isValid: errors.length === 0,
                        errors
                    };
                }
            }
        });
    }
    
    /**
     * Create handlers for direct message creation
     */
    createDirectMessageHandlers() {
        return this.createCustomHandler({
            method: "POST",
            path: "/chat/direct",
            response: {
                __typename: "Chat" as const,
                id: generatePK().toString(),
                openToAnyoneWithInvite: false,
                team: null,
                translations: [{
                    __typename: "ChatTranslation" as const,
                    id: generatePK().toString(),
                    language: "en",
                    name: "Direct Message"
                }],
                invites: [],
                messages: [],
                participants: [],
                participantsCount: 0,
                invitesCount: 0,
                createdAt: new Date().toISOString(),
                publicId: generatePK().toString()
            }
        });
    }
    
    /**
     * Create handlers for team chat creation
     */
    createTeamChatHandlers(teamId: string) {
        return this.createCustomHandler({
            method: "POST",
            path: "/chat/team",
            response: {
                __typename: "Chat" as const,
                id: generatePK().toString(),
                openToAnyoneWithInvite: false,
                team: { __typename: "Team" as const, id: teamId },
                translations: [],
                invites: [],
                messages: [],
                participants: [],
                participantsCount: 0,
                invitesCount: 0,
                createdAt: new Date().toISOString(),
                publicId: generatePK().toString()
            }
        });
    }
}

/**
 * Chat UI state fixture factory
 */
class ChatUIStateFixtureFactory implements UIStateFixtureFactory<ChatUIState> {
    createLoadingState(context?: { type: string }): ChatUIState {
        return {
            isLoading: true,
            chat: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            participants: [],
            messageCount: 0,
            isDirectMessage: false
        };
    }
    
    createErrorState(error: { message: string }): ChatUIState {
        return {
            isLoading: false,
            chat: null,
            error: error.message,
            isEditing: false,
            hasUnsavedChanges: false,
            participants: [],
            messageCount: 0,
            isDirectMessage: false
        };
    }
    
    createSuccessState(data: Chat): ChatUIState {
        return {
            isLoading: false,
            chat: data,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            participants: data.participants?.map(p => ({
                id: p.id,
                name: p.user.name || p.user.handle,
                handle: p.user.handle
            })) || [],
            messageCount: data.messages?.length || 0,
            isDirectMessage: !data.team && data.participantsCount === 2
        };
    }
    
    createEmptyState(): ChatUIState {
        return {
            isLoading: false,
            chat: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            participants: [],
            messageCount: 0,
            isDirectMessage: false
        };
    }
    
    transitionToLoading(currentState: ChatUIState): ChatUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null
        };
    }
    
    transitionToSuccess(currentState: ChatUIState, data: Chat): ChatUIState {
        return {
            ...currentState,
            isLoading: false,
            chat: data,
            error: null,
            hasUnsavedChanges: false,
            participants: data.participants?.map(p => ({
                id: p.id,
                name: p.user.name || p.user.handle,
                handle: p.user.handle
            })) || [],
            messageCount: data.messages?.length || 0,
            isDirectMessage: !data.team && data.participantsCount === 2
        };
    }
    
    transitionToError(currentState: ChatUIState, error: { message: string }): ChatUIState {
        return {
            ...currentState,
            isLoading: false,
            error: error.message
        };
    }
    
    /**
     * Transition to editing state
     */
    transitionToEditing(currentState: ChatUIState): ChatUIState {
        return {
            ...currentState,
            isEditing: true,
            hasUnsavedChanges: false
        };
    }
    
    /**
     * Mark as having unsaved changes
     */
    markUnsavedChanges(currentState: ChatUIState): ChatUIState {
        return {
            ...currentState,
            hasUnsavedChanges: true
        };
    }
}

/**
 * Complete Chat fixture factory
 */
export class ChatFixtureFactory implements UIFixtureFactory<
    ChatFormData,
    ChatCreateInput,
    ChatUpdateInput,
    Chat,
    ChatUIState
> {
    readonly objectType = "chat";
    
    form: ChatFormFixtureFactory;
    roundTrip: RoundTripOrchestrator<ChatFormData, Chat>;
    handlers: ChatMSWHandlerFactory;
    states: ChatUIStateFixtureFactory;
    componentUtils: ComponentTestUtils<any>;
    
    constructor(apiClient: TestAPIClient, dbVerifier: DatabaseVerifier) {
        this.form = new ChatFormFixtureFactory();
        this.handlers = new ChatMSWHandlerFactory();
        this.states = new ChatUIStateFixtureFactory();
        
        // Initialize round-trip orchestrator
        this.roundTrip = new BaseRoundTripOrchestrator({
            apiClient,
            dbVerifier,
            formFixture: this.form,
            endpoints: {
                create: "/api/chat",
                update: "/api/chat",
                delete: "/api/chat",
                find: "/api/chat"
            },
            tableName: "chat",
            fieldMappings: {
                id: "id",
                openToAnyoneWithInvite: "openToAnyoneWithInvite",
                teamConnect: "teamId"
            }
        });
        
        // Component utils would be initialized here
        this.componentUtils = {} as any; // Placeholder
    }
    
    createFormData(scenario: "minimal" | "complete" | "directMessage" | "teamChat" | "publicChat" | string = "minimal"): ChatFormData {
        return this.form.createFormData(scenario);
    }
    
    createAPIInput(formData: ChatFormData): ChatCreateInput {
        // Transform form data to API input
        const apiInput: ChatCreateInput = {
            id: generatePK().toString(),
            ...(formData.openToAnyoneWithInvite !== undefined && { 
                openToAnyoneWithInvite: formData.openToAnyoneWithInvite 
            }),
            ...(formData.team && { teamConnect: formData.team.id })
        };

        // Add translations if name or description provided
        if (formData.name || formData.description) {
            apiInput.translationsCreate = [{
                id: generatePK().toString(),
                language: formData.language || "en",
                ...(formData.name && { name: formData.name }),
                ...(formData.description && { description: formData.description })
            }];
        }

        return apiInput;
    }
    
    createMockResponse(overrides?: Partial<Chat>): Chat {
        return {
            __typename: "Chat" as const,
            id: generatePK().toString(),
            openToAnyoneWithInvite: false,
            team: null,
            invites: [],
            messages: [],
            participants: [],
            participantsCount: 0,
            invitesCount: 0,
            createdAt: new Date().toISOString(),
            publicId: generatePK().toString(),
            translations: [],
            ...overrides
        } as Chat;
    }
    
    setupMSW(scenario: "success" | "error" | "loading" = "success"): void {
        // This would integrate with the MSW server
        // For now, it's a placeholder
        console.log(`Setting up MSW handlers for scenario: ${scenario}`);
    }
    
    async testCreateFlow(formData?: ChatFormData): Promise<Chat> {
        const data = formData || this.createFormData("minimal");
        const result = await this.roundTrip.testCreateFlow(data);
        
        if (!result.success) {
            throw new Error(`Create flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.id as unknown as Chat;
    }
    
    async testUpdateFlow(id: string, updates: Partial<ChatFormData>): Promise<Chat> {
        const result = await this.roundTrip.testUpdateFlow(id, updates);
        
        if (!result.success) {
            throw new Error(`Update flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.updatedData as Chat;
    }
    
    async testDeleteFlow(id: string): Promise<boolean> {
        const result = await this.roundTrip.testDeleteFlow(id);
        return result.success;
    }
    
    async testRoundTrip(formData?: ChatFormData) {
        const data = formData || this.createFormData("complete");
        const result = await this.roundTrip.executeFullCycle({
            formData: data,
            validateEachStep: true
        });
        
        if (!result.success) {
            throw new Error(`Round trip failed: ${result.errors?.join(", ")}`);
        }
        
        return {
            success: result.success,
            formData: data,
            apiResponse: result.data!.apiResponse,
            uiState: this.states.createSuccessState(result.data!.apiResponse)
        };
    }
    
    /**
     * Test direct message creation flow
     */
    async testDirectMessageFlow(otherUserId: string): Promise<Chat> {
        const formData = this.form.createDirectMessageFormData(otherUserId);
        return this.testCreateFlow(formData);
    }
    
    /**
     * Test team chat creation flow
     */
    async testTeamChatFlow(teamId: string, chatName: string): Promise<Chat> {
        const formData = this.form.createTeamChatFormData(teamId, chatName);
        return this.testCreateFlow(formData);
    }
}

// Register in the global registry
// This would normally be done after creating the API client and DB verifier
// registerFixture("chat", new ChatFixtureFactory(apiClient, dbVerifier));
/* c8 ignore stop */