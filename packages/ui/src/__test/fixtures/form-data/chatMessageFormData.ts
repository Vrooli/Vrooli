import { DUMMY_ID, type ChatMessageShape } from "@vrooli/shared";

/**
 * Form data fixtures for chat message creation and editing
 * These represent data as it appears in form state before submission
 */

/**
 * Minimal chat message form input - just text
 */
export const minimalChatMessageFormInput: Partial<ChatMessageShape> = {
    text: "Hello, this is a test message",
};

/**
 * Complete chat message form input with all fields
 */
export const completeChatMessageFormInput: Partial<ChatMessageShape> = {
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

/**
 * Reply message form input
 */
export const replyMessageFormInput: Partial<ChatMessageShape> = {
    text: "This is a reply to the previous message",
    parentId: "123456789012345678", // Parent message ID (18 digits)
    parent: {
        __typename: "ChatMessage",
        __connect: true,
        id: "123456789012345678",
    },
};

/**
 * Message with mentions form input
 */
export const messageWithMentionsFormInput: Partial<ChatMessageShape> = {
    text: "Hey @user1 and @user2, check this out!",
    config: {
        __version: "1.0.0",
        resources: [],
        mentions: ["user_123456789012345678", "user_987654321098765432"],
    },
};

/**
 * System message form input
 */
export const systemMessageFormInput: Partial<ChatMessageShape> = {
    text: "User joined the chat",
    config: {
        __version: "1.0.0",
        resources: [],
        role: "system",
    },
};

/**
 * Assistant message form input
 */
export const assistantMessageFormInput: Partial<ChatMessageShape> = {
    text: "I can help you with that. Here's what I found...",
    config: {
        __version: "1.0.0",
        resources: [],
        role: "assistant",
        respondingBots: ["@assistant"],
    },
};

/**
 * Tool response message form input
 */
export const toolMessageFormInput: Partial<ChatMessageShape> = {
    text: "Executed search function",
    config: {
        __version: "1.0.0",
        resources: [],
        role: "tool",
        toolCalls: [
            {
                id: "tool_123456789012345678",
                function: {
                    name: "search",
                    arguments: JSON.stringify({ query: "AI assistant" }),
                },
                result: {
                    success: true,
                    output: "Found 5 results",
                },
            },
        ],
    },
};

/**
 * Message with attachments/resources form input
 */
export const messageWithResourcesFormInput: Partial<ChatMessageShape> = {
    text: "Check out these resources",
    config: {
        __version: "1.0.0",
        resources: [
            { id: "resource_123", type: "document", name: "Project Plan.pdf" },
            { id: "resource_456", type: "image", name: "Screenshot.png" },
        ],
    },
};

/**
 * Edited message form input
 */
export const editedMessageFormInput: Partial<ChatMessageShape> = {
    id: "123456789012345678",
    text: "This message has been edited",
    versionIndex: 1, // Incremented for edits
};

/**
 * Long message form input (testing text limits)
 */
export const longMessageFormInput: Partial<ChatMessageShape> = {
    text: "Lorem ipsum dolor sit amet, ".repeat(100).trim(), // ~3000 characters
};

/**
 * Message with special formatting
 */
export const formattedMessageFormInput: Partial<ChatMessageShape> = {
    text: `# Heading
This message has **bold** and *italic* text.
- List item 1
- List item 2

\`\`\`javascript
const code = "example";
\`\`\``,
};

/**
 * Multi-language message form input
 */
export const multiLanguageMessageFormInput: Partial<ChatMessageShape> = {
    text: "Hello world",
    language: "en",
    // Note: In real usage, translations would be handled separately
};

/**
 * Invalid form inputs for testing validation
 */
export const invalidChatMessageFormInputs = {
    emptyText: {
        text: "",
    },
    whitespaceOnly: {
        text: "   \n\t   ",
    },
    textTooLong: {
        text: "x".repeat(32769), // Over the 32768 character limit
    },
    invalidRole: {
        text: "Valid message",
        config: {
            __version: "1.0.0",
            resources: [],
            // @ts-expect-error - Testing invalid role
            role: "invalid_role",
        },
    },
    missingConfigVersion: {
        text: "Valid message",
        config: {
            // @ts-expect-error - Missing required __version
            resources: [],
            role: "user",
        },
    },
    invalidParentId: {
        text: "Reply message",
        parentId: "invalid-id", // Not a valid snowflake ID
    },
    negativeVersionIndex: {
        text: "Valid message",
        versionIndex: -1,
    },
};

/**
 * Helper function to transform form data to API input
 */
export const transformChatMessageFormToApiInput = (formData: Partial<ChatMessageShape>): ChatMessageShape => {
    return {
        __typename: "ChatMessage",
        id: formData.id || DUMMY_ID,
        text: formData.text || "",
        language: formData.language || "en",
        versionIndex: formData.versionIndex || 0,
        config: formData.config || {
            __version: "1.0.0",
            resources: [],
        },
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        reactionSummaries: [],
        you: {
            canDelete: true,
            canUpdate: true,
            canReply: true,
            canReport: true,
            canReact: true,
        },
        ...formData,
    };
};

/**
 * Helper function to validate message content
 */
export const validateChatMessageContent = (message: string): string | null => {
    if (!message || !message.trim()) return "Message cannot be empty";
    if (message.length > 32768) return "Message is too long (max 32,768 characters)";
    return null;
};

/**
 * Helper function to extract mentions from message text
 */
export const extractMentionsFromMessage = (text: string): string[] => {
    const mentionRegex = /@(\w+)/g;
    const mentions: string[] = [];
    let match;
    
    while ((match = mentionRegex.exec(text)) !== null) {
        mentions.push(match[1]);
    }
    
    return [...new Set(mentions)]; // Remove duplicates
};

/**
 * Helper to check if message has been edited
 */
export const isEditedMessage = (message: Partial<ChatMessageShape>): boolean => {
    return (message.versionIndex ?? 0) > 0;
};

/**
 * Mock form states for testing
 */
export const chatMessageFormStates = {
    pristine: {
        values: { text: "" },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    typing: {
        values: { text: "Hello wo" },
        errors: {},
        touched: { text: true },
        isValid: true,
        isSubmitting: false,
    },
    withErrors: {
        values: { text: "" },
        errors: { text: "Message cannot be empty" },
        touched: { text: true },
        isValid: false,
        isSubmitting: false,
    },
    submitting: {
        values: { text: "Sending this message..." },
        errors: {},
        touched: { text: true },
        isValid: true,
        isSubmitting: true,
    },
    submitted: {
        values: { text: "" }, // Reset after successful submit
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
};

/**
 * Chat message status transitions
 */
export const chatMessageStatusTransitions = {
    new: { status: "unsent" as const },
    sending: { status: "sending" as const },
    sent: { status: "sent" as const },
    failed: { status: "failed" as const },
    editing: { status: "editing" as const },
};