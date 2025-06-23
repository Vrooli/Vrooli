/**
 * Chat Form Data Fixtures
 * 
 * Provides comprehensive form data fixtures for chat creation, message composition,
 * and chat management forms with React Hook Form integration.
 */

import { useForm, type UseFormReturn, type FieldErrors } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import * as yup from "yup";
import { act, waitFor } from "@testing-library/react";
import type { 
    ChatCreateInput,
    ChatUpdateInput,
    ChatMessageCreateInput,
    Chat,
    ChatMessage,
    ChatParticipantCreateInput,
    LlmTask,
} from "@vrooli/shared";
import { 
    chatValidation,
    chatMessageValidation,
    LlmTask as LlmTaskEnum,
} from "@vrooli/shared";

/**
 * UI-specific form data for chat creation
 */
export interface ChatCreateFormData {
    // Basic info
    name?: string;
    description?: string;
    
    // Privacy settings
    isPrivate: boolean;
    
    // Initial participants (UI-specific)
    participants?: Array<{
        userId?: string;
        handle?: string;
        canEdit?: boolean;
        canDelete?: boolean;
    }>;
    
    // Chat settings
    allowBots?: boolean;
    maxParticipants?: number;
    
    // AI settings
    llmTask?: LlmTask;
    aiInstructions?: string;
}

/**
 * UI-specific form data for chat message composition
 */
export interface ChatMessageFormData {
    content: string;
    
    // Message type indicators (UI-specific)
    isCommand?: boolean;
    isReply?: boolean;
    replyToId?: string;
    
    // Attachments
    attachments?: Array<{
        name: string;
        type: "image" | "file" | "link";
        url: string;
        description?: string;
    }>;
    
    // AI interaction
    requestAIResponse?: boolean;
    aiContext?: string;
    
    // Formatting options
    useMarkdown?: boolean;
    mentionedUsers?: string[];
}

/**
 * UI-specific form data for chat settings
 */
export interface ChatSettingsFormData {
    name?: string;
    description?: string;
    isPrivate: boolean;
    allowBots?: boolean;
    maxParticipants?: number;
    
    // AI configuration
    llmTask?: LlmTask;
    aiInstructions?: string;
    
    // Participant management
    participantPermissions?: {
        canInvite: boolean;
        canEdit: boolean;
        canDelete: boolean;
    };
}

/**
 * Extended form state with chat-specific properties
 */
export interface ChatFormState<T = ChatCreateFormData> {
    values: T;
    errors: Record<string, string>;
    touched: Record<string, boolean>;
    isDirty: boolean;
    isSubmitting: boolean;
    isValid: boolean;
    isValidating?: boolean;
    
    // Chat-specific state
    chatState?: {
        currentChat?: Partial<Chat>;
        participants?: Array<{
            id: string;
            handle: string;
            name: string;
            isOnline: boolean;
        }>;
        messageCount?: number;
        hasUnreadMessages?: boolean;
    };
    
    // Message composition state
    messageState?: {
        isTyping?: boolean;
        typingUsers?: string[];
        lastMessage?: Partial<ChatMessage>;
        characterCount?: number;
        hasUnsavedDraft?: boolean;
    };
    
    // AI state
    aiState?: {
        isAIResponding?: boolean;
        lastAIResponse?: string;
        aiAvailable?: boolean;
        remainingTokens?: number;
    };
}

/**
 * Chat form data factory with React Hook Form integration
 */
export class ChatFormDataFactory {
    /**
     * Create validation schema for chat creation
     */
    private createChatCreationSchema(): yup.ObjectSchema<ChatCreateFormData> {
        return yup.object({
            name: yup
                .string()
                .max(100, "Chat name cannot exceed 100 characters")
                .optional(),
                
            description: yup
                .string()
                .max(500, "Description cannot exceed 500 characters")
                .optional(),
                
            isPrivate: yup
                .boolean()
                .required("Privacy setting is required"),
                
            participants: yup.array(
                yup.object({
                    userId: yup.string().optional(),
                    handle: yup.string().optional(),
                    canEdit: yup.boolean().optional(),
                    canDelete: yup.boolean().optional(),
                }).test("user-identification", "Either userId or handle is required", function(value) {
                    return !!(value?.userId || value?.handle);
                }),
            ).optional(),
            
            allowBots: yup.boolean().optional(),
            
            maxParticipants: yup
                .number()
                .min(2, "Chat must allow at least 2 participants")
                .max(100, "Maximum 100 participants allowed")
                .optional(),
                
            llmTask: yup
                .mixed<LlmTask>()
                .oneOf(Object.values(LlmTaskEnum))
                .optional(),
                
            aiInstructions: yup
                .string()
                .max(2000, "AI instructions cannot exceed 2000 characters")
                .when("llmTask", {
                    is: (val: LlmTask | undefined) => !!val,
                    then: (schema) => schema.optional(),
                    otherwise: (schema) => schema.optional(),
                }),
        }).defined();
    }
    
    /**
     * Create validation schema for chat messages
     */
    private createMessageSchema(): yup.ObjectSchema<ChatMessageFormData> {
        return yup.object({
            content: yup
                .string()
                .required("Message content is required")
                .min(1, "Message cannot be empty")
                .max(10000, "Message is too long"),
                
            isCommand: yup.boolean().optional(),
            isReply: yup.boolean().optional(),
            replyToId: yup.string().optional(),
            
            attachments: yup.array(
                yup.object({
                    name: yup.string().required("Attachment name is required"),
                    type: yup.string().oneOf(["image", "file", "link"]).required(),
                    url: yup.string().url("Invalid URL").required("Attachment URL is required"),
                    description: yup.string().max(200, "Description too long").optional(),
                }),
            ).max(10, "Maximum 10 attachments per message").optional(),
            
            requestAIResponse: yup.boolean().optional(),
            aiContext: yup.string().max(1000, "AI context too long").optional(),
            useMarkdown: yup.boolean().optional(),
            
            mentionedUsers: yup.array(yup.string()).optional(),
        }).defined();
    }
    
    /**
     * Create validation schema for chat settings
     */
    private createSettingsSchema(): yup.ObjectSchema<ChatSettingsFormData> {
        return yup.object({
            name: yup
                .string()
                .max(100, "Chat name cannot exceed 100 characters")
                .optional(),
                
            description: yup
                .string()
                .max(500, "Description cannot exceed 500 characters")
                .optional(),
                
            isPrivate: yup
                .boolean()
                .required("Privacy setting is required"),
                
            allowBots: yup.boolean().optional(),
            
            maxParticipants: yup
                .number()
                .min(2, "Chat must allow at least 2 participants")
                .max(100, "Maximum 100 participants allowed")
                .optional(),
                
            llmTask: yup
                .mixed<LlmTask>()
                .oneOf(Object.values(LlmTaskEnum))
                .optional(),
                
            aiInstructions: yup
                .string()
                .max(2000, "AI instructions cannot exceed 2000 characters")
                .optional(),
                
            participantPermissions: yup.object({
                canInvite: yup.boolean().required(),
                canEdit: yup.boolean().required(),
                canDelete: yup.boolean().required(),
            }).optional(),
        }).defined();
    }
    
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `chat_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create chat creation form data for different scenarios
     */
    createChatFormData(
        scenario: "empty" | "minimal" | "complete" | "invalid" | "privateChat" | "groupChat" | "aiChat" | "withParticipants",
    ): ChatCreateFormData {
        switch (scenario) {
            case "empty":
                return {
                    isPrivate: false,
                };
                
            case "minimal":
                return {
                    isPrivate: false,
                };
                
            case "complete":
                return {
                    name: "Project Discussion",
                    description: "Chat for discussing project updates and coordination",
                    isPrivate: false,
                    allowBots: true,
                    maxParticipants: 20,
                    llmTask: LlmTaskEnum.Start,
                    aiInstructions: "Help with project management and technical questions",
                    participants: [
                        {
                            handle: "alice",
                            canEdit: true,
                            canDelete: false,
                        },
                        {
                            handle: "bob",
                            canEdit: false,
                            canDelete: false,
                        },
                    ],
                };
                
            case "invalid":
                return {
                    name: "A".repeat(150), // Too long
                    isPrivate: false,
                    maxParticipants: 1, // Too few
                    participants: [
                        {
                            // Missing both userId and handle
                            canEdit: true,
                        },
                    ],
                };
                
            case "privateChat":
                return {
                    name: "Private Discussion",
                    description: "Confidential team discussion",
                    isPrivate: true,
                    allowBots: false,
                    maxParticipants: 5,
                };
                
            case "groupChat":
                return {
                    name: "Team Standup",
                    description: "Daily standup meeting chat",
                    isPrivate: false,
                    allowBots: true,
                    maxParticipants: 15,
                    participants: [
                        { handle: "alice", canEdit: true, canDelete: true },
                        { handle: "bob", canEdit: true, canDelete: false },
                        { handle: "charlie", canEdit: false, canDelete: false },
                    ],
                };
                
            case "aiChat":
                return {
                    name: "AI Assistant",
                    description: "Chat with AI assistant for code help",
                    isPrivate: false,
                    allowBots: true,
                    llmTask: LlmTaskEnum.Start,
                    aiInstructions: "You are a helpful coding assistant. Provide clear, concise answers to programming questions.",
                    maxParticipants: 10,
                };
                
            case "withParticipants":
                return {
                    name: "Development Team",
                    isPrivate: false,
                    participants: [
                        { userId: "user_1", canEdit: true, canDelete: true },
                        { userId: "user_2", canEdit: true, canDelete: false },
                        { handle: "designer1", canEdit: false, canDelete: false },
                        { handle: "pm1", canEdit: true, canDelete: false },
                    ],
                };
                
            default:
                throw new Error(`Unknown chat creation scenario: ${scenario}`);
        }
    }
    
    /**
     * Create chat message form data for different scenarios
     */
    createMessageFormData(
        scenario: "empty" | "simple" | "complete" | "invalid" | "withAttachments" | "aiRequest" | "reply" | "command",
    ): ChatMessageFormData {
        switch (scenario) {
            case "empty":
                return {
                    content: "",
                };
                
            case "simple":
                return {
                    content: "Hello everyone!",
                };
                
            case "complete":
                return {
                    content: "Here are the latest updates on the project. @alice please review the new features.",
                    useMarkdown: true,
                    mentionedUsers: ["alice"],
                    attachments: [
                        {
                            name: "Project Update",
                            type: "file",
                            url: "https://example.com/project-update.pdf",
                            description: "Latest project status report",
                        },
                    ],
                };
                
            case "invalid":
                return {
                    content: "", // Empty content
                    attachments: [
                        {
                            name: "", // Empty name
                            type: "file",
                            url: "not-a-url", // Invalid URL
                        },
                    ],
                };
                
            case "withAttachments":
                return {
                    content: "Check out these resources",
                    attachments: [
                        {
                            name: "Screenshot",
                            type: "image",
                            url: "https://example.com/screenshot.png",
                            description: "UI mockup",
                        },
                        {
                            name: "Documentation",
                            type: "link",
                            url: "https://docs.example.com",
                            description: "Updated documentation",
                        },
                    ],
                };
                
            case "aiRequest":
                return {
                    content: "Can you help me debug this React component?",
                    requestAIResponse: true,
                    aiContext: "Working on a form component with validation issues",
                    useMarkdown: true,
                };
                
            case "reply":
                return {
                    content: "That's a great point!",
                    isReply: true,
                    replyToId: "msg_123",
                };
                
            case "command":
                return {
                    content: "/help commands",
                    isCommand: true,
                };
                
            default:
                throw new Error(`Unknown message scenario: ${scenario}`);
        }
    }
    
    /**
     * Create chat settings form data
     */
    createSettingsFormData(
        scenario: "basic" | "complete" | "restrictive" | "aiEnabled",
    ): ChatSettingsFormData {
        switch (scenario) {
            case "basic":
                return {
                    name: "General Chat",
                    isPrivate: false,
                };
                
            case "complete":
                return {
                    name: "Project Coordination",
                    description: "Main channel for project updates and discussions",
                    isPrivate: false,
                    allowBots: true,
                    maxParticipants: 25,
                    participantPermissions: {
                        canInvite: true,
                        canEdit: true,
                        canDelete: false,
                    },
                };
                
            case "restrictive":
                return {
                    name: "Admin Only",
                    description: "Restricted channel for administrators",
                    isPrivate: true,
                    allowBots: false,
                    maxParticipants: 5,
                    participantPermissions: {
                        canInvite: false,
                        canEdit: false,
                        canDelete: false,
                    },
                };
                
            case "aiEnabled":
                return {
                    name: "AI Assistance",
                    description: "Chat with AI helper",
                    isPrivate: false,
                    allowBots: true,
                    llmTask: LlmTaskEnum.Start,
                    aiInstructions: "Provide helpful responses and assist with questions",
                };
                
            default:
                throw new Error(`Unknown settings scenario: ${scenario}`);
        }
    }
    
    /**
     * Create form state for different scenarios
     */
    createFormState<T extends ChatCreateFormData | ChatMessageFormData | ChatSettingsFormData>(
        scenario: "pristine" | "dirty" | "submitting" | "withErrors" | "valid" | "typing" | "aiResponding",
        formType: "create" | "message" | "settings" = "create",
    ): ChatFormState<T> {
        let baseFormData: any;
        
        switch (formType) {
            case "create":
                baseFormData = this.createChatFormData("complete");
                break;
            case "message":
                baseFormData = this.createMessageFormData("complete");
                break;
            case "settings":
                baseFormData = this.createSettingsFormData("complete");
                break;
        }
        
        switch (scenario) {
            case "pristine":
                return {
                    values: (formType === "create" 
                        ? this.createChatFormData("empty")
                        : formType === "message"
                        ? this.createMessageFormData("empty")
                        : this.createSettingsFormData("basic")) as T,
                    errors: {},
                    touched: {},
                    isDirty: false,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "dirty":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: formType === "message" ? { content: true } : { name: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "submitting":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: true,
                    isValid: true,
                };
                
            case "withErrors":
                const errors = formType === "create" 
                    ? {
                        name: "Chat name is too long",
                        maxParticipants: "Must allow at least 2 participants",
                    }
                    : formType === "message"
                    ? {
                        content: "Message content is required",
                    }
                    : {
                        description: "Description is too long",
                    };
                    
                return {
                    values: (formType === "create"
                        ? this.createChatFormData("invalid")
                        : formType === "message"
                        ? this.createMessageFormData("invalid")
                        : this.createSettingsFormData("basic")) as T,
                    errors,
                    touched: Object.keys(errors).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: false,
                    isValid: false,
                };
                
            case "valid":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: Object.keys(baseFormData).reduce((acc, key) => ({
                        ...acc,
                        [key]: true,
                    }), {}),
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                };
                
            case "typing":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: { content: true },
                    isDirty: true,
                    isSubmitting: false,
                    isValid: true,
                    messageState: {
                        isTyping: true,
                        characterCount: (baseFormData as ChatMessageFormData).content?.length || 0,
                        hasUnsavedDraft: true,
                    },
                };
                
            case "aiResponding":
                return {
                    values: baseFormData as T,
                    errors: {},
                    touched: { content: true },
                    isDirty: false,
                    isSubmitting: false,
                    isValid: true,
                    aiState: {
                        isAIResponding: true,
                        aiAvailable: true,
                        remainingTokens: 1500,
                    },
                };
                
            default:
                throw new Error(`Unknown form state scenario: ${scenario}`);
        }
    }
    
    /**
     * Create React Hook Form instance
     */
    createFormInstance<T extends ChatCreateFormData | ChatMessageFormData | ChatSettingsFormData>(
        formType: "create" | "message" | "settings",
        initialData?: Partial<T>,
    ): UseFormReturn<T> {
        let defaultValues: any;
        let resolver: any;
        
        switch (formType) {
            case "create":
                defaultValues = {
                    ...this.createChatFormData("empty"),
                    ...initialData,
                };
                resolver = yupResolver(this.createChatCreationSchema());
                break;
                
            case "message":
                defaultValues = {
                    ...this.createMessageFormData("empty"),
                    ...initialData,
                };
                resolver = yupResolver(this.createMessageSchema());
                break;
                
            case "settings":
                defaultValues = {
                    ...this.createSettingsFormData("basic"),
                    ...initialData,
                };
                resolver = yupResolver(this.createSettingsSchema());
                break;
        }
        
        return useForm<T>({
            mode: "onChange",
            reValidateMode: "onChange",
            shouldFocusError: true,
            defaultValues,
            resolver,
        });
    }
    
    /**
     * Validate form data using real validation
     */
    async validateFormData<T extends ChatCreateFormData | ChatMessageFormData>(
        formData: T,
        formType: "create" | "message",
    ): Promise<{
        isValid: boolean;
        errors?: Record<string, string>;
        apiInput?: ChatCreateInput | ChatMessageCreateInput;
    }> {
        try {
            const schema = formType === "create" 
                ? this.createChatCreationSchema()
                : this.createMessageSchema();
                
            await schema.validate(formData, { abortEarly: false });
            
            const apiInput = formType === "create"
                ? this.transformChatToAPIInput(formData as ChatCreateFormData)
                : this.transformMessageToAPIInput(formData as ChatMessageFormData);
                
            if (formType === "create") {
                await chatValidation.create.validate(apiInput);
            } else {
                await chatMessageValidation.create.validate(apiInput);
            }
            
            return {
                isValid: true,
                apiInput,
            };
        } catch (error: any) {
            const errors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        errors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                errors.general = error.message;
            }
            
            return {
                isValid: false,
                errors,
            };
        }
    }
    
    /**
     * Transform chat form data to API input
     */
    private transformChatToAPIInput(formData: ChatCreateFormData): ChatCreateInput {
        const input: ChatCreateInput = {
            id: this.generateId(),
            isPrivate: formData.isPrivate,
        };
        
        if (formData.name || formData.description) {
            input.translationsCreate = [{
                id: this.generateId(),
                language: "en",
                name: formData.name,
                description: formData.description,
            }];
        }
        
        if (formData.participants) {
            input.participantsCreate = formData.participants.map(p => ({
                id: this.generateId(),
                userConnect: p.userId || p.handle || "",
                canEdit: p.canEdit,
                canDelete: p.canDelete,
            }));
        }
        
        return input;
    }
    
    /**
     * Transform message form data to API input
     */
    private transformMessageToAPIInput(formData: ChatMessageFormData): ChatMessageCreateInput {
        return {
            id: this.generateId(),
            chatConnect: "chat_123", // Would be provided by context
            parent: formData.replyToId ? { connect: formData.replyToId } : undefined,
            translationsCreate: [{
                id: this.generateId(),
                language: "en",
                text: formData.content,
            }],
        };
    }
}

/**
 * Form interaction simulator for chat forms
 */
export class ChatFormInteractionSimulator {
    private interactionDelay = 100;
    
    constructor(delay?: number) {
        this.interactionDelay = delay || 100;
    }
    
    /**
     * Simulate chat message typing with realistic delays
     */
    async simulateMessageTyping(
        formInstance: UseFormReturn<ChatMessageFormData>,
        message: string,
        options?: { withPauses?: boolean; typingSpeed?: number },
    ): Promise<void> {
        const { withPauses = true, typingSpeed = 50 } = options || {};
        
        // Clear field first
        await this.fillField(formInstance, "content", "");
        
        for (let i = 1; i <= message.length; i++) {
            const partialMessage = message.substring(0, i);
            await this.fillField(formInstance, "content", partialMessage);
            
            // Add realistic pauses at punctuation and spaces
            if (withPauses && /[.!?,:;]\s/.test(message.charAt(i - 1))) {
                await new Promise(resolve => setTimeout(resolve, typingSpeed * 3));
            } else {
                await new Promise(resolve => setTimeout(resolve, typingSpeed));
            }
        }
    }
    
    /**
     * Simulate adding participants to chat
     */
    async simulateAddingParticipants(
        formInstance: UseFormReturn<ChatCreateFormData>,
        participants: ChatCreateFormData["participants"],
    ): Promise<void> {
        if (!participants) return;
        
        for (let i = 0; i < participants.length; i++) {
            const participant = participants[i];
            
            if (participant.userId) {
                await this.fillField(formInstance, `participants.${i}.userId` as any, participant.userId);
            } else if (participant.handle) {
                await this.fillField(formInstance, `participants.${i}.handle` as any, participant.handle);
            }
            
            if (participant.canEdit !== undefined) {
                await this.fillField(formInstance, `participants.${i}.canEdit` as any, participant.canEdit);
            }
            
            if (participant.canDelete !== undefined) {
                await this.fillField(formInstance, `participants.${i}.canDelete` as any, participant.canDelete);
            }
            
            await new Promise(resolve => setTimeout(resolve, 200));
        }
    }
    
    /**
     * Fill field helper
     */
    private async fillField(
        formInstance: UseFormReturn<any>,
        fieldName: string,
        value: any,
    ): Promise<void> {
        act(() => {
            formInstance.setValue(fieldName, value, {
                shouldDirty: true,
                shouldTouch: true,
                shouldValidate: true,
            });
        });
        
        await waitFor(() => {
            expect(formInstance.formState.isValidating).toBe(false);
        });
    }
}

// Export factory instances
export const chatFormFactory = new ChatFormDataFactory();
export const chatFormSimulator = new ChatFormInteractionSimulator();

// Export pre-configured scenarios
export const chatFormScenarios = {
    // Chat creation scenarios
    emptyChatCreation: () => chatFormFactory.createFormState("pristine", "create"),
    validChatCreation: () => chatFormFactory.createFormState("valid", "create"),
    chatCreationWithErrors: () => chatFormFactory.createFormState("withErrors", "create"),
    
    // Message scenarios
    emptyMessage: () => chatFormFactory.createFormState("pristine", "message"),
    typingMessage: () => chatFormFactory.createFormState("typing", "message"),
    aiRespondingMessage: () => chatFormFactory.createFormState("aiResponding", "message"),
    
    // Chat types
    privateChat: () => chatFormFactory.createChatFormData("privateChat"),
    groupChat: () => chatFormFactory.createChatFormData("groupChat"),
    aiChat: () => chatFormFactory.createChatFormData("aiChat"),
    
    // Message types
    simpleMessage: () => chatFormFactory.createMessageFormData("simple"),
    messageWithAttachments: () => chatFormFactory.createMessageFormData("withAttachments"),
    aiRequestMessage: () => chatFormFactory.createMessageFormData("aiRequest"),
    replyMessage: () => chatFormFactory.createMessageFormData("reply"),
    commandMessage: () => chatFormFactory.createMessageFormData("command"),
    
    // Interactive workflows
    async quickMessageWorkflow(formInstance: UseFormReturn<ChatMessageFormData>) {
        const simulator = new ChatFormInteractionSimulator();
        const message = "Hello team! How is everyone doing?";
        await simulator.simulateMessageTyping(formInstance, message, { typingSpeed: 30 });
        return message;
    },
    
    async aiRequestWorkflow(formInstance: UseFormReturn<ChatMessageFormData>) {
        const simulator = new ChatFormInteractionSimulator();
        const message = "Can you help me understand this error?";
        await simulator.simulateMessageTyping(formInstance, message);
        await simulator.fillField(formInstance, "requestAIResponse", true);
        return message;
    },
};
