/* c8 ignore start */
/**
 * Comment fixture factory implementation
 * 
 * This demonstrates the complete pattern for implementing fixtures for a Vrooli object type
 * with full type safety and integration with @vrooli/shared.
 */

import {
    type Comment,
    type CommentCreateInput,
    type CommentUpdateInput,
    type CommentFor,
    shapeComment,
    commentValidation,
    commentFixtures as sharedCommentFixtures
} from "@vrooli/shared";
import { generatePK } from "@vrooli/shared";
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

/**
 * Comment form data type
 * 
 * This includes UI-specific fields like parentId and mentions
 * that don't exist in the API input type.
 */
export interface CommentFormData extends Record<string, unknown> {
    text: string;
    commentedOn: { __typename: string; id: string };
    threadId?: string;
    parentId?: string;
    language?: string;
    isEditing?: boolean;
    mentions?: string[];
}

/**
 * Comment UI state type
 */
export interface CommentUIState {
    isLoading: boolean;
    comment: Comment | null;
    error: string | null;
    isEditing: boolean;
    hasUnsavedChanges: boolean;
    showReplies: boolean;
    replyCount: number;
}

/**
 * Comment form fixture factory
 */
class CommentFormFixtureFactory extends BaseFormFixtureFactory<CommentFormData, CommentCreateInput> {
    constructor() {
        super({
            scenarios: {
                minimal: {
                    text: "This is a test comment",
                    commentedOn: { __typename: "Issue", id: "123456789012345678" },
                    language: "en"
                },
                complete: {
                    text: "This is a comprehensive comment with detailed feedback about the issue. It provides constructive criticism and suggestions for improvement.",
                    commentedOn: { __typename: "ResourceVersion", id: "123456789012345679" },
                    threadId: "thread123",
                    language: "en",
                    mentions: ["@user1", "@user2"]
                },
                invalid: {
                    text: "", // Empty text
                    commentedOn: { __typename: "InvalidType", id: "invalid-id" },
                    language: "invalid-lang"
                },
                reply: {
                    text: "This is a reply to another comment",
                    commentedOn: { __typename: "Issue", id: "123456789012345680" },
                    parentId: "123456789012345681",
                    language: "en"
                },
                withMentions: {
                    text: "Great work @user1! This addresses the concerns raised by @user2 and @user3.",
                    commentedOn: { __typename: "PullRequest", id: "123456789012345682" },
                    language: "en",
                    mentions: ["@user1", "@user2", "@user3"]
                },
                multiLanguage: {
                    text: "Este es un comentario en español",
                    commentedOn: { __typename: "ResourceVersion", id: "123456789012345683" },
                    language: "es"
                }
            },
            
            validate: createValidationAdapter<CommentFormData>(
                async (data) => {
                    // Additional UI validation
                    const errors: string[] = [];
                    
                    if (!data.text || (typeof data.text === 'string' && data.text.trim().length === 0)) {
                        errors.push("text: Comment text is required");
                    }
                    
                    if (typeof data.text === 'string' && data.text.length > 32768) {
                        errors.push("text: Comment text exceeds maximum length");
                    }
                    
                    if (!data.commentedOn || 
                        typeof data.commentedOn !== 'object' || 
                        !('id' in data.commentedOn) || 
                        !('__typename' in data.commentedOn)) {
                        errors.push("commentedOn: Valid commented object is required");
                    }
                    
                    if (errors.length > 0) {
                        return { isValid: false, errors };
                    }
                    
                    // Use shared validation for the rest
                    const apiInput = this.shapeToAPI!(data);
                    return commentValidation.create.validate(apiInput);
                }
            ),
            
            shapeToAPI: (formData) => {
                // Transform to CommentCreateInput
                return {
                    id: generatePK().toString(),
                    createdFor: formData.commentedOn.__typename as CommentFor,
                    forConnect: formData.commentedOn.id,
                    ...(formData.parentId && { parentConnect: formData.parentId }),
                    translationsCreate: [{
                        id: generatePK().toString(),
                        language: formData.language || "en",
                        text: formData.text
                    }]
                };
            }
        });
    }
    
    /**
     * Create reply form data
     */
    createReplyFormData(parentId: string, scenario: "minimal" | "complete" = "minimal"): CommentFormData {
        if (scenario === "minimal") {
            return {
                text: "This is a reply",
                commentedOn: { __typename: "Issue", id: "123456789012345678" },
                parentId,
                language: "en"
            };
        }
        
        return {
            text: "This is a detailed reply with additional context and references.",
            commentedOn: { __typename: "Issue", id: "123456789012345678" },
            parentId,
            language: "en",
            mentions: ["@originalAuthor"]
        };
    }
    
    /**
     * Create edit form data from existing comment
     */
    createEditFormData(existingComment: Comment, newText?: string): CommentFormData {
        const translation = existingComment.translations[0];
        return {
            text: newText || translation?.text || "",
            commentedOn: { 
                __typename: existingComment.commentedOn.__typename, 
                id: existingComment.commentedOn.id 
            },
            language: translation?.language || "en",
            isEditing: true
        };
    }
}

/**
 * Comment MSW handler factory
 */
class CommentMSWHandlerFactory extends BaseMSWHandlerFactory<CommentCreateInput, CommentUpdateInput, Comment> {
    constructor() {
        super({
            baseUrl: "/api",
            endpoints: {
                create: "/comment",
                update: "/comment",
                delete: "/comment",
                find: "/comment",
                list: "/comments"
            },
            successResponses: {
                create: (input) => this.createMockComment({
                    id: input.id,
                    commentedOn: {
                        __typename: input.createdFor,
                        id: input.forConnect
                    } as any,
                    translations: input.translationsCreate || []
                }),
                update: (input) => this.createMockComment({
                    id: input.id,
                    translations: input.translationsUpdate || input.translationsCreate || []
                }),
                find: (id) => this.createMockComment({
                    id
                })
            },
            validate: {
                create: (input) => {
                    const errors: string[] = [];
                    
                    if (!input.id) {
                        errors.push("ID is required");
                    }
                    
                    if (!input.createdFor) {
                        errors.push("createdFor is required");
                    }
                    
                    if (!input.forConnect) {
                        errors.push("forConnect is required");
                    }
                    
                    if (!input.translationsCreate || input.translationsCreate.length === 0) {
                        errors.push("At least one translation is required");
                    }
                    
                    if (input.translationsCreate) {
                        input.translationsCreate.forEach((translation, index) => {
                            if (!translation.text || translation.text.trim().length === 0) {
                                errors.push(`Translation ${index}: text is required`);
                            }
                            if (translation.text && translation.text.length > 32768) {
                                errors.push(`Translation ${index}: text exceeds maximum length`);
                            }
                        });
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
     * Create a mock Comment object
     */
    createMockComment(overrides: Partial<Comment> = {}): Comment {
        return {
            __typename: "Comment",
            id: overrides.id || generatePK().toString(),
            bookmarkedBy: [],
            bookmarks: 0,
            commentedOn: overrides.commentedOn || {
                __typename: "Issue",
                id: "123456789012345678"
            } as any,
            createdAt: new Date().toISOString(),
            owner: null,
            reports: [],
            reportsCount: 0,
            score: 0,
            translations: overrides.translations || [{
                __typename: "CommentTranslation",
                id: generatePK().toString(),
                language: "en",
                text: "This is a test comment"
            }],
            translationsCount: overrides.translations?.length || 1,
            updatedAt: new Date().toISOString(),
            you: {
                __typename: "CommentYou",
                canBookmark: true,
                canDelete: true,
                canReact: true,
                canReply: true,
                canReport: true,
                canUpdate: true,
                isBookmarked: false,
                reaction: null
            },
            ...overrides
        };
    }
    
    /**
     * Create thread-specific handlers
     */
    createThreadHandlers() {
        return this.createCustomHandler({
            method: "GET",
            path: "/comment/thread/:threadId",
            response: {
                threads: [
                    {
                        comment: this.createMockComment(),
                        childThreads: [],
                        totalInThread: 1
                    }
                ],
                totalThreads: 1
            }
        });
    }
    
    /**
     * Create reply handlers
     */
    createReplyHandlers() {
        return this.createCustomHandler({
            method: "POST",
            path: "/comment/:parentId/reply",
            response: this.createMockComment({
                id: generatePK().toString(),
                // parentConnect would be set in the actual response
            })
        });
    }
}

/**
 * Comment UI state fixture factory
 */
class CommentUIStateFixtureFactory implements UIStateFixtureFactory<CommentUIState> {
    createLoadingState(context?: { type: string }): CommentUIState {
        return {
            isLoading: true,
            comment: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            showReplies: false,
            replyCount: 0
        };
    }
    
    createErrorState(error: { message: string }): CommentUIState {
        return {
            isLoading: false,
            comment: null,
            error: error.message,
            isEditing: false,
            hasUnsavedChanges: false,
            showReplies: false,
            replyCount: 0
        };
    }
    
    createSuccessState(data: Comment): CommentUIState {
        return {
            isLoading: false,
            comment: data,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            showReplies: false,
            replyCount: 0
        };
    }
    
    createEmptyState(): CommentUIState {
        return {
            isLoading: false,
            comment: null,
            error: null,
            isEditing: false,
            hasUnsavedChanges: false,
            showReplies: false,
            replyCount: 0
        };
    }
    
    transitionToLoading(currentState: CommentUIState): CommentUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null
        };
    }
    
    transitionToSuccess(currentState: CommentUIState, data: Comment): CommentUIState {
        return {
            ...currentState,
            isLoading: false,
            comment: data,
            error: null,
            hasUnsavedChanges: false
        };
    }
    
    transitionToError(currentState: CommentUIState, error: { message: string }): CommentUIState {
        return {
            ...currentState,
            isLoading: false,
            error: error.message
        };
    }
    
    /**
     * Toggle editing state
     */
    transitionToEditing(currentState: CommentUIState): CommentUIState {
        return {
            ...currentState,
            isEditing: true,
            hasUnsavedChanges: false
        };
    }
    
    /**
     * Toggle replies visibility
     */
    transitionToShowReplies(currentState: CommentUIState, replyCount: number): CommentUIState {
        return {
            ...currentState,
            showReplies: !currentState.showReplies,
            replyCount
        };
    }
}

/**
 * Complete Comment fixture factory
 */
export class CommentFixtureFactory implements UIFixtureFactory<
    CommentFormData,
    CommentCreateInput,
    CommentUpdateInput,
    Comment,
    CommentUIState
> {
    readonly objectType = "comment";
    
    form: CommentFormFixtureFactory;
    roundTrip: RoundTripOrchestrator<CommentFormData, Comment>;
    handlers: CommentMSWHandlerFactory;
    states: CommentUIStateFixtureFactory;
    componentUtils: ComponentTestUtils<any>;
    
    constructor(apiClient: TestAPIClient, dbVerifier: DatabaseVerifier) {
        this.form = new CommentFormFixtureFactory();
        this.handlers = new CommentMSWHandlerFactory();
        this.states = new CommentUIStateFixtureFactory();
        
        // Initialize round-trip orchestrator
        this.roundTrip = new BaseRoundTripOrchestrator({
            apiClient,
            dbVerifier,
            formFixture: this.form,
            endpoints: {
                create: "/api/comment",
                update: "/api/comment",
                delete: "/api/comment",
                find: "/api/comment"
            },
            tableName: "comment",
            fieldMappings: {
                text: "translations[0].text",
                language: "translations[0].language",
                commentedOn: "commentedOn.id"
            }
        });
        
        // Component utils would be initialized here
        this.componentUtils = {} as any; // Placeholder
    }
    
    createFormData(scenario: "minimal" | "complete" | "invalid" | "reply" | "withMentions" | "multiLanguage" | string = "minimal"): CommentFormData {
        return this.form.createFormData(scenario);
    }
    
    createAPIInput(formData: CommentFormData): CommentCreateInput {
        return this.form.transformToAPIInput(formData);
    }
    
    createMockResponse(overrides?: Partial<Comment>): Comment {
        return this.handlers.createMockComment(overrides);
    }
    
    setupMSW(scenario: "success" | "error" | "loading" = "success"): void {
        // This would integrate with the MSW server
        // For now, it's a placeholder
        console.log(`Setting up MSW handlers for scenario: ${scenario}`);
    }
    
    async testCreateFlow(formData?: CommentFormData): Promise<Comment> {
        const data = formData || this.createFormData("minimal");
        const result = await this.roundTrip.testCreateFlow(data);
        
        if (!result.success) {
            throw new Error(`Create flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.id as unknown as Comment;
    }
    
    async testUpdateFlow(id: string, updates: Partial<CommentFormData>): Promise<Comment> {
        const result = await this.roundTrip.testUpdateFlow(id, updates);
        
        if (!result.success) {
            throw new Error(`Update flow failed: ${result.errors?.join(", ")}`);
        }
        
        return result.metadata?.updatedData as Comment;
    }
    
    async testDeleteFlow(id: string): Promise<boolean> {
        const result = await this.roundTrip.testDeleteFlow(id);
        return result.success;
    }
    
    async testRoundTrip(formData?: CommentFormData) {
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
     * Test comment threading functionality
     */
    async testThreadFlow(parentComment: Comment, replyData?: CommentFormData) {
        const reply = replyData || this.form.createReplyFormData(parentComment.id);
        const result = await this.testCreateFlow(reply);
        
        return {
            parentComment,
            reply: result,
            uiState: this.states.transitionToShowReplies(
                this.states.createSuccessState(parentComment),
                1
            )
        };
    }
    
    /**
     * Test comment editing functionality
     */
    async testEditFlow(existingComment: Comment, newText: string) {
        const editData = this.form.createEditFormData(existingComment, newText);
        const editingState = this.states.transitionToEditing(
            this.states.createSuccessState(existingComment)
        );
        
        const updateInput: CommentUpdateInput = {
            id: existingComment.id,
            translationsUpdate: [{
                id: existingComment.translations[0].id,
                text: newText
            }]
        };
        
        // Simulate the update flow
        const updatedComment = {
            ...existingComment,
            translations: [{
                ...existingComment.translations[0],
                text: newText
            }]
        };
        
        return {
            editData,
            editingState,
            updatedComment,
            finalState: this.states.transitionToSuccess(editingState, updatedComment)
        };
    }
}

// Register in the global registry
// This would normally be done after creating the API client and DB verifier
// registerFixture("comment", new CommentFixtureFactory(apiClient, dbVerifier));
/* c8 ignore stop */