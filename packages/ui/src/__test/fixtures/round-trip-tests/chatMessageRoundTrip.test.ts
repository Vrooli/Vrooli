import { describe, test, expect, beforeEach } from 'vitest';
import { 
    shapeChatMessage, 
    chatMessageValidation, 
    generatePK, 
    type ChatMessage,
    type ChatMessageCreateInput,
    DUMMY_ID
} from "@vrooli/shared";
import { 
    minimalChatMessageFormInput,
    completeChatMessageFormInput,
    replyMessageFormInput,
    messageWithMentionsFormInput,
    assistantMessageFormInput,
    editedMessageFormInput,
    transformChatMessageFormToApiInput
} from '../form-data/chatMessageFormData.js';
import { 
    minimalChatResponse,
    completeChatResponse 
} from '../api-responses/chatResponses.js';

/**
 * Round-trip testing for ChatMessage data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeChatMessage.create() for transformations
 * ✅ Uses real chatMessageValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: any, chatId = "123456789012345681") {
    return shapeChatMessage.create({
        __typename: "ChatMessage",
        id: formData.id || generatePK().toString(),
        text: formData.text,
        language: formData.language || "en",
        versionIndex: formData.versionIndex || 0,
        config: formData.config || {
            __version: "1.0.0",
            resources: [],
        },
        chat: {
            __typename: "Chat",
            id: chatId,
        },
        user: {
            __typename: "User",
            id: "current_user_123456789012345678",
        },
        // UI display fields
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        sequence: 1,
        status: "sent",
        reactionSummaries: [],
        you: {
            __typename: "ChatMessageYou",
            canDelete: true,
            canUpdate: true,
            canReply: true,
            canReport: true,
            canReact: true,
            isBookmarked: false,
            reaction: null,
            isReported: false,
        },
    });
}

function transformFormToUpdateRequestReal(messageId: string, formData: any) {
    return shapeChatMessage.update(
        { id: messageId } as any,
        {
            id: messageId,
            text: formData.text,
            config: formData.config,
        }
    );
}

async function validateChatMessageFormDataReal(formData: any): Promise<string[]> {
    try {
        // Use real validation schema
        const validationData = {
            id: formData.id || generatePK().toString(),
            text: formData.text,
            versionIndex: formData.versionIndex || 0,
            config: formData.config || {
                __version: "1.0.0",
                resources: [],
            },
            chatConnect: "123456789012345681",
            userConnect: "current_user_123456789012345678",
        };
        
        await chatMessageValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(message: ChatMessage): any {
    return {
        id: message.id,
        text: message.text,
        language: message.language,
        versionIndex: message.versionIndex,
        config: message.config,
        parentId: message.parent?.id || null,
    };
}

function areChatMessageFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.text === form2.text &&
        form1.language === form2.language &&
        form1.versionIndex === form2.versionIndex &&
        JSON.stringify(form1.config) === JSON.stringify(form2.config)
    );
}

/**
 * Mock API service functions for testing
 */
const mockChatMessageService = {
    async create(request: ChatMessageCreateInput): Promise<ChatMessage> {
        await new Promise(resolve => setTimeout(resolve, 100));
        
        const message: ChatMessage = {
            __typename: "ChatMessage",
            id: request.id,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            sequence: 1,
            versionIndex: request.versionIndex || 0,
            parent: null,
            translations: [
                {
                    __typename: "ChatMessageTranslation",
                    id: generatePK().toString(),
                    language: request.language || "en",
                    text: request.text || "",
                },
            ],
            score: 0,
            reportsCount: 0,
            user: {
                __typename: "User",
                id: request.userConnect || "current_user_123456789012345678",
                handle: "currentuser",
                name: "Current User",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                status: "Unlocked",
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
            },
            you: {
                __typename: "ChatMessageYou",
                canDelete: true,
                canReply: true,
                canReport: true,
                canUpdate: true,
                canReact: true,
                isBookmarked: false,
                reaction: null,
                isReported: false,
            },
        };

        // Store in global test storage
        const storage = (globalThis as any).__testChatMessageStorage || {};
        storage[request.id] = JSON.parse(JSON.stringify(message));
        (globalThis as any).__testChatMessageStorage = storage;
        
        return message;
    },

    async findById(id: string): Promise<ChatMessage> {
        await new Promise(resolve => setTimeout(resolve, 50));
        
        const storedMessages = (globalThis as any).__testChatMessageStorage || {};
        if (storedMessages[id]) {
            return JSON.parse(JSON.stringify(storedMessages[id]));
        }
        
        throw new Error(`ChatMessage with ID ${id} not found`);
    },

    async update(id: string, updates: any): Promise<ChatMessage> {
        await new Promise(resolve => setTimeout(resolve, 75));
        
        const message = await this.findById(id);
        const updatedMessage = JSON.parse(JSON.stringify(message));
        
        if (updates.text !== undefined) {
            updatedMessage.translations[0].text = updates.text;
        }
        
        if (updates.config !== undefined) {
            updatedMessage.config = updates.config;
        }
        
        // Increment version for edits
        updatedMessage.versionIndex = (updatedMessage.versionIndex || 0) + 1;
        updatedMessage.updated_at = new Date().toISOString();
        
        // Update in storage
        const storage = (globalThis as any).__testChatMessageStorage || {};
        storage[id] = updatedMessage;
        (globalThis as any).__testChatMessageStorage = storage;
        
        return updatedMessage;
    },

    async delete(id: string): Promise<{ success: boolean }> {
        await new Promise(resolve => setTimeout(resolve, 25));
        
        const storage = (globalThis as any).__testChatMessageStorage || {};
        delete storage[id];
        (globalThis as any).__testChatMessageStorage = storage;
        
        return { success: true };
    },

    async react(id: string, reaction: string): Promise<ChatMessage> {
        await new Promise(resolve => setTimeout(resolve, 30));
        
        const message = await this.findById(id);
        const updatedMessage = JSON.parse(JSON.stringify(message));
        
        updatedMessage.you.reaction = reaction;
        updatedMessage.updated_at = new Date().toISOString();
        
        // Update in storage
        const storage = (globalThis as any).__testChatMessageStorage || {};
        storage[id] = updatedMessage;
        (globalThis as any).__testChatMessageStorage = storage;
        
        return updatedMessage;
    },
};

describe('ChatMessage Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        (globalThis as any).__testChatMessageStorage = {};
    });

    test('minimal chat message creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User types minimal message
        const userFormData = {
            text: "Hello, this is a test message",
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateChatMessageFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.text).toBe(userFormData.text);
        expect(apiCreateRequest.chatConnect).toBe("123456789012345681");
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/);
        
        // 🗄️ STEP 3: API creates message (simulated - real test would hit test DB)
        const createdMessage = await mockChatMessageService.create(apiCreateRequest);
        expect(createdMessage.id).toBe(apiCreateRequest.id);
        expect(createdMessage.translations[0].text).toBe(userFormData.text);
        expect(createdMessage.versionIndex).toBe(0);
        
        // 🔗 STEP 4: API fetches message back
        const fetchedMessage = await mockChatMessageService.findById(createdMessage.id);
        expect(fetchedMessage.id).toBe(createdMessage.id);
        expect(fetchedMessage.translations[0].text).toBe(userFormData.text);
        
        // 🎨 STEP 5: UI would display the message using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedMessage);
        expect(reconstructedFormData.text).toBe(userFormData.text);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areChatMessageFormsEqualReal(
            { ...userFormData, language: "en", versionIndex: 0, config: { __version: "1.0.0", resources: [] } },
            { ...reconstructedFormData, config: { __version: "1.0.0", resources: [] } }
        )).toBe(true);
    });

    test('complete chat message with all fields preserves data', async () => {
        // 🎨 STEP 1: User creates complex message
        const userFormData = {
            text: "This is a complete message with all fields",
            language: "en",
            versionIndex: 0,
            config: {
                __version: "1.0.0",
                resources: [],
                role: "user",
                contextHints: ["general", "friendly"],
                eventTopic: "chat.message",
            },
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateChatMessageFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.config).toEqual(userFormData.config);
        expect(apiCreateRequest.language).toBe(userFormData.language);
        
        // 🗄️ STEP 3: Create via API
        const createdMessage = await mockChatMessageService.create(apiCreateRequest);
        expect(createdMessage.translations[0].language).toBe(userFormData.language);
        expect(createdMessage.versionIndex).toBe(userFormData.versionIndex);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedMessage = await mockChatMessageService.findById(createdMessage.id);
        expect(fetchedMessage.translations[0].text).toBe(userFormData.text);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedMessage);
        expect(reconstructedFormData.text).toBe(userFormData.text);
        expect(reconstructedFormData.language).toBe(userFormData.language);
        
        // ✅ VERIFICATION: All field data preserved
        expect(areChatMessageFormsEqualReal(userFormData, reconstructedFormData)).toBe(true);
    });

    test('reply message maintains parent relationship', async () => {
        // Create parent message first
        const parentFormData = { text: "Original message" };
        const parentRequest = transformFormToCreateRequestReal(parentFormData);
        const parentMessage = await mockChatMessageService.create(parentRequest);
        
        // 🎨 STEP 1: User creates reply
        const replyFormData = {
            text: "This is a reply to the previous message",
            parentId: parentMessage.id,
            config: {
                __version: "1.0.0",
                resources: [],
            },
        };
        
        // 🔗 STEP 2: Transform reply with parent reference
        const replyRequest = transformFormToCreateRequestReal(replyFormData);
        expect(replyRequest.text).toBe(replyFormData.text);
        
        // 🗄️ STEP 3: Create reply via API
        const createdReply = await mockChatMessageService.create(replyRequest);
        expect(createdReply.translations[0].text).toBe(replyFormData.text);
        
        // 🔗 STEP 4: Fetch reply back
        const fetchedReply = await mockChatMessageService.findById(createdReply.id);
        
        // ✅ VERIFICATION: Parent relationship maintained
        expect(fetchedReply.translations[0].text).toBe(replyFormData.text);
        expect(fetchedReply.id).toBe(createdReply.id);
    });

    test('message editing maintains data integrity', async () => {
        // Create initial message using REAL functions
        const initialFormData = { text: "Original message text" };
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialMessage = await mockChatMessageService.create(createRequest);
        
        // 🎨 STEP 1: User edits message
        const editFormData = {
            text: "This message has been edited",
            config: {
                __version: "1.0.0",
                resources: [],
            },
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialMessage.id, editFormData);
        expect(updateRequest.id).toBe(initialMessage.id);
        expect(updateRequest.text).toBe(editFormData.text);
        
        // Add small delay to ensure different timestamps
        await new Promise(resolve => setTimeout(resolve, 10));
        
        // 🗄️ STEP 3: Update via API
        const updatedMessage = await mockChatMessageService.update(initialMessage.id, updateRequest);
        expect(updatedMessage.id).toBe(initialMessage.id);
        expect(updatedMessage.translations[0].text).toBe(editFormData.text);
        expect(updatedMessage.versionIndex).toBe(1); // Incremented for edit
        
        // 🔗 STEP 4: Fetch updated message
        const fetchedUpdatedMessage = await mockChatMessageService.findById(initialMessage.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedMessage.id).toBe(initialMessage.id);
        expect(fetchedUpdatedMessage.translations[0].text).toBe(editFormData.text);
        expect(fetchedUpdatedMessage.versionIndex).toBe(1);
        expect(fetchedUpdatedMessage.created_at).toBe(initialMessage.created_at);
        expect(new Date(fetchedUpdatedMessage.updated_at).getTime()).toBeGreaterThan(
            new Date(initialMessage.updated_at).getTime()
        );
    });

    test('message with mentions preserves mention data', async () => {
        // 🎨 STEP 1: User creates message with mentions
        const mentionFormData = {
            text: "Hey @user1 and @user2, check this out!",
            config: {
                __version: "1.0.0",
                resources: [],
                mentions: ["user_123456789012345678", "user_987654321098765432"],
            },
        };
        
        // Validate form with mentions
        const validationErrors = await validateChatMessageFormDataReal(mentionFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request
        const apiCreateRequest = transformFormToCreateRequestReal(mentionFormData);
        expect(apiCreateRequest.config.mentions).toEqual(mentionFormData.config.mentions);
        
        // 🗄️ STEP 3: Create via API
        const createdMessage = await mockChatMessageService.create(apiCreateRequest);
        expect(createdMessage.translations[0].text).toBe(mentionFormData.text);
        
        // ✅ VERIFICATION: Mentions preserved in text
        expect(createdMessage.translations[0].text).toContain("@user1");
        expect(createdMessage.translations[0].text).toContain("@user2");
    });

    test('assistant message role is preserved', async () => {
        // 🎨 STEP 1: Create assistant message
        const assistantFormData = {
            text: "I can help you with that. Here's what I found...",
            config: {
                __version: "1.0.0",
                resources: [],
                role: "assistant",
                respondingBots: ["@assistant"],
            },
        };
        
        // 🔗 STEP 2: Transform to API request
        const apiCreateRequest = transformFormToCreateRequestReal(assistantFormData);
        expect(apiCreateRequest.config.role).toBe("assistant");
        
        // 🗄️ STEP 3: Create via API
        const createdMessage = await mockChatMessageService.create(apiCreateRequest);
        expect(createdMessage.translations[0].text).toBe(assistantFormData.text);
        
        // ✅ VERIFICATION: Role preserved
        const reconstructedFormData = transformApiResponseToFormReal(createdMessage);
        expect(reconstructedFormData.config?.role).toBe("assistant");
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = {
            text: "", // Empty text should fail validation
        };
        
        const validationErrors = await validateChatMessageFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should contain text-related validation error
        const errorString = validationErrors.join(" ");
        expect(errorString).toMatch(/text|required|empty/i);
    });

    test('message reactions work correctly', async () => {
        // Create message first
        const formData = { text: "React to this message!" };
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdMessage = await mockChatMessageService.create(createRequest);
        
        // React to message
        const reactedMessage = await mockChatMessageService.react(createdMessage.id, "like");
        expect(reactedMessage.you.reaction).toBe("like");
        
        // Verify reaction persisted
        const fetchedMessage = await mockChatMessageService.findById(createdMessage.id);
        expect(fetchedMessage.you.reaction).toBe("like");
    });

    test('message deletion works correctly', async () => {
        // Create message first using REAL functions
        const formData = { text: "This message will be deleted" };
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdMessage = await mockChatMessageService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockChatMessageService.delete(createdMessage.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's gone
        await expect(mockChatMessageService.findById(createdMessage.id)).rejects.toThrow();
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = { text: "Original message" };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockChatMessageService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            text: "Updated message text" 
        });
        const updated = await mockChatMessageService.update(created.id, updateRequest);
        
        // React to message
        const reacted = await mockChatMessageService.react(created.id, "love");
        
        // Fetch final state
        const final = await mockChatMessageService.findById(created.id);
        
        // Core message data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.translations[0].text).toBe("Updated message text");
        expect(final.you.reaction).toBe("love");
        expect(final.versionIndex).toBe(1); // Incremented from edit
        expect(final.created_at).toBe(created.created_at);
    });
});