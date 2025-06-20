import { generatePK, generatePublicId } from "../../../../../shared/src/id/index.js";
import { type Prisma } from "@prisma/client";
import { messageConfigFixtures } from "../../../../../shared/src/__test/fixtures/config/messageConfigFixtures.js";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for ChatMessage model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const chatMessageDbIds = {
    message1: generatePK(),
    message2: generatePK(),
    message3: generatePK(),
    chat1: generatePK(),
    chat2: generatePK(),
    user1: generatePK(),
    user2: generatePK(),
    bot1: generatePK(),
};

/**
 * Enhanced test fixtures for ChatMessage model following standard structure
 */
export const chatMessageDbFixtures: DbTestFixtures<Prisma.chat_messageCreateInput> = {
    minimal: {
        id: generatePK(),
        versionIndex: 0,
        language: "en",
        text: "Test message",
        chatId: BigInt(chatMessageDbIds.chat1),
        userId: BigInt(chatMessageDbIds.user1),
        config: messageConfigFixtures.minimal as any,
    },
    complete: {
        id: generatePK(),
        versionIndex: 1,
        language: "en",
        text: "This is a complete test message with detailed content and proper formatting.",
        chatId: BigInt(chatMessageDbIds.chat1),
        userId: BigInt(chatMessageDbIds.user1),
        parentId: BigInt(chatMessageDbIds.message1),
        config: messageConfigFixtures.complete as any,
        score: 5,
    },
    invalid: {
        missingRequired: {
            // Missing required chat, user, versionIndex, language, and text
            id: generatePK(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            language: 123, // Should be string
            text: null, // Should be string
            versionIndex: "invalid", // Should be number
            chatId: "invalid-chat-reference", // Should be BigInt
            userId: "invalid-user-reference", // Should be BigInt
            parentId: "invalid-parent-reference", // Should be BigInt
        },
        negativeVersionIndex: {
            id: generatePK(),
            versionIndex: -1, // Negative version index
            language: "en",
            text: "Invalid version index",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
        },
        emptyText: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "", // Empty text
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
        },
    },
    edgeCases: {
        rootMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Root message without parent",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.variants.userMessage as any,
        },
        replyMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Reply to root message",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user2),
            parentId: BigInt(chatMessageDbIds.message1),
            config: messageConfigFixtures.minimal as any,
        },
        botMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "This is an automated bot response with helpful information.",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.bot1),
            config: messageConfigFixtures.variants.assistantWithTools as any,
        },
        longMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "This is a very long message that contains a lot of text to test how the system handles messages with substantial content. ".repeat(10),
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
        },
        multiLanguageMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Hello world",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
        },
        highVersionIndex: {
            id: generatePK(),
            versionIndex: 999,
            language: "en",
            text: "Message with high version index",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
        },
        specialCharactersMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Message with special chars: !@#$%^&*()_+{}|:<>?[]\\;'\",./ and emojis 😀😁😂",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
        },
        pendingMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Message still pending delivery",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
            score: -1, // Could indicate pending state
        },
        failedMessage: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Message failed to send",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            config: messageConfigFixtures.minimal as any,
            score: -2, // Could indicate failed state
        },
        threadedConversation: {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Part of a threaded conversation",
            chatId: BigInt(chatMessageDbIds.chat1),
            userId: BigInt(chatMessageDbIds.user1),
            parentId: BigInt(chatMessageDbIds.message2),
            config: messageConfigFixtures.minimal as any,
        },
    },
};

/**
 * Enhanced factory for creating chat message database fixtures
 */
export class ChatMessageDbFactory extends EnhancedDbFactory<Prisma.chat_messageCreateInput> {
    
    /**
     * Get the test fixtures for ChatMessage model
     */
    protected getFixtures(): DbTestFixtures<Prisma.chat_messageCreateInput> {
        return chatMessageDbFixtures;
    }

    /**
     * Get ChatMessage-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: chatMessageDbIds.message1, // Duplicate ID
                    versionIndex: 0,
                    language: "en",
                    text: "Duplicate message",
                    chatId: BigInt(chatMessageDbIds.chat1),
                    userId: BigInt(chatMessageDbIds.user1),
                    config: messageConfigFixtures.minimal as any,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    versionIndex: 0,
                    language: "en",
                    text: "Foreign key violation",
                    chat: { connect: { id: "non-existent-chat-id" } },
                    user: { connect: { id: chatMessageDbIds.user1 } },
                    config: messageConfigFixtures.minimal as any,
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    versionIndex: 0,
                    language: "", // Empty language
                    text: "Check constraint violation",
                    chatId: BigInt(chatMessageDbIds.chat1),
                    userId: BigInt(chatMessageDbIds.user1),
                    config: messageConfigFixtures.minimal as any,
                },
            },
            validation: {
                requiredFieldMissing: chatMessageDbFixtures.invalid.missingRequired,
                invalidDataType: chatMessageDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    versionIndex: 999999, // Very high version index
                    language: "en",
                    text: "a".repeat(50000), // Text too long (max 32768)
                    chatId: BigInt(chatMessageDbIds.chat1),
                    userId: BigInt(chatMessageDbIds.user1),
                    config: messageConfigFixtures.minimal as any,
                },
            },
            businessLogic: {
                circularParent: {
                    id: generatePK(),
                    versionIndex: 0,
                    language: "en",
                    text: "Circular parent reference",
                    chatId: BigInt(chatMessageDbIds.chat1),
                    userId: BigInt(chatMessageDbIds.user1),
                    parentId: BigInt(chatMessageDbIds.message1), // Would create circular reference
                    config: messageConfigFixtures.minimal as any,
                },
                emptyText: chatMessageDbFixtures.invalid.emptyText,
            },
        };
    }

    /**
     * ChatMessage-specific validation
     */
    protected validateSpecific(data: Prisma.chat_messageCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to ChatMessage
        if (!data.chat) errors.push("ChatMessage chat is required");
        if (!data.user) errors.push("ChatMessage user is required");
        if (!data.language) errors.push("ChatMessage language is required");
        if (!data.text) errors.push("ChatMessage text is required");
        if (data.versionIndex === undefined) errors.push("ChatMessage versionIndex is required");

        // Check version index
        if (typeof data.versionIndex === 'number') {
            if (data.versionIndex < 0) {
                errors.push("Version index cannot be negative");
            }
            if (data.versionIndex > 1000) {
                warnings.push("Very high version index (>1000)");
            }
        }

        // Check text content
        if (data.text && typeof data.text === 'string') {
            if (data.text.length === 0) {
                errors.push("Message text cannot be empty");
            }
            if (data.text.length > 32768) {
                errors.push("Message text exceeds maximum length of 32768 characters");
            }
            if (data.text.length > 10000) {
                warnings.push("Message text is very long (>10000 chars)");
            }
        }

        // Check language
        if (data.language && (data.language.length < 2 || data.language.length > 3)) {
            errors.push("Language code must be 2-3 characters");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>
    ): Prisma.chat_messageCreateInput {
        const factory = new ChatMessageDbFactory();
        return factory.createMinimal({
            chatId: BigInt(chatId),
            userId: BigInt(userId),
            versionIndex: 0,
            language: "en",
            text: "Test message",
            config: messageConfigFixtures.minimal as any,
            ...overrides,
        });
    }

    static createWithParent(
        chatId: string,
        userId: string,
        parentId: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>
    ): Prisma.chat_messageCreateInput {
        return this.createMinimal(chatId, userId, {
            parentId: BigInt(parentId),
            ...overrides,
        });
    }

    static createBotMessage(
        chatId: string,
        botId: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>
    ): Prisma.chat_messageCreateInput {
        return this.createMinimal(chatId, botId, {
            config: messageConfigFixtures.variants.assistantWithTools as any,
            text: "This is an automated bot response.",
            ...overrides,
        });
    }

    /**
     * Create user message with specific text
     */
    static createUserMessage(
        chatId: string,
        userId: string,
        text: string,
        score: number = 0,
        overrides?: Partial<Prisma.chat_messageCreateInput>
    ): Prisma.chat_messageCreateInput {
        return this.createMinimal(chatId, userId, {
            config: messageConfigFixtures.variants.userMessage as any,
            text,
            score,
            ...overrides,
        });
    }

    /**
     * Create bot response with tool calls
     */
    static createBotResponseWithTools(
        chatId: string,
        botId: string,
        text: string,
        toolCalls?: any[],
        overrides?: Partial<Prisma.chat_messageCreateInput>
    ): Prisma.chat_messageCreateInput {
        const config = toolCalls 
            ? {
                ...messageConfigFixtures.variants.assistantWithTools,
                toolCalls,
            }
            : messageConfigFixtures.variants.assistantWithTools;

        return this.createMinimal(chatId, botId, {
            config,
            text,
            ...overrides,
        });
    }

    /**
     * Create threaded conversation message
     */
    static createThreadedMessage(
        chatId: string,
        userId: string,
        parentId: string,
        text: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>
    ): Prisma.chat_messageCreateInput {
        return this.createWithParent(chatId, userId, parentId, {
            text,
            ...overrides,
        });
    }

    static createSequence(
        chatId: string,
        userId: string,
        count: number,
        startIndex: number = 0
    ): Prisma.chat_messageCreateInput[] {
        const messages = [];
        for (let i = 0; i < count; i++) {
            messages.push(this.createMinimal(chatId, userId, {
                versionIndex: startIndex + i,
                text: `Message ${startIndex + i + 1} in sequence`,
            }));
        }
        return messages;
    }

    /**
     * Create a realistic conversation between users and bot
     */
    static createConversationSequence(
        chatId: string,
        participants: {
            userId: string;
            botId: string;
        },
        messages: Array<{
            type: "user" | "bot";
            text: string;
            score?: number;
            parentId?: string;
        }>
    ): Prisma.chat_messageCreateInput[] {
        const messageData: Prisma.chat_messageCreateInput[] = [];
        const factory = new ChatMessageDbFactory();

        messages.forEach((msg, index) => {
            const isBot = msg.type === "bot";
            const senderId = isBot ? participants.botId : participants.userId;
            const config = isBot 
                ? messageConfigFixtures.variants.assistantWithTools 
                : messageConfigFixtures.variants.userMessage;

            const message: Prisma.chat_messageCreateInput = {
                id: generatePK(),
                versionIndex: index,
                language: "en",
                text: msg.text,
                chatId: BigInt(chatId),
                userId: BigInt(senderId),
                config,
                score: msg.score || 0,
            };

            if (msg.parentId) {
                message.parentId = BigInt(msg.parentId);
            }

            messageData.push(message);
        });

        return messageData;
    }

    /**
     * Create typing indicator message
     */
    static createTypingIndicator(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>
    ): Prisma.chat_messageCreateInput {
        return this.createMinimal(chatId, userId, {
            config: {
                ...messageConfigFixtures.minimal,
                role: "system",
                eventTopic: "typing-indicator",
            },
            text: "...",
            score: -1,
            ...overrides,
        });
    }
}

/**
 * Enhanced helper to seed multiple test chat messages with comprehensive options
 */
export async function seedChatMessages(
    prisma: any,
    options: {
        chatId: string;
        messages: Array<{
            userId: string;
            text?: string;
            parentId?: string;
            versionIndex?: number;
        }>;
    }
): Promise<BulkSeedResult<any>> {
    const createdMessages = [];
    let rootCount = 0;
    let replyCount = 0;
    let botCount = 0;

    for (const msg of options.messages) {
        const messageData: Prisma.chat_messageCreateInput = {
            id: generatePK(),
            versionIndex: msg.versionIndex ?? 0,
            language: "en",
            text: msg.text || "Test message",
            chatId: BigInt(options.chatId),
            userId: BigInt(msg.userId),
            config: messageConfigFixtures.minimal as any,
        };

        if (msg.parentId) {
            messageData.parentId = BigInt(msg.parentId);
            replyCount++;
        } else {
            rootCount++;
        }

        // Check if this is a bot message (simplified check)
        if (msg.userId.includes('bot') || (msg.text && msg.text.includes('bot'))) {
            botCount++;
        }

        const message = await prisma.chatMessage.create({
            data: messageData,
            include: {
                translations: true,
                user: true,
            },
        });
        createdMessages.push(message);
    }

    return {
        records: createdMessages,
        summary: {
            total: createdMessages.length,
            withAuth: 0, // ChatMessages don't have auth
            bots: botCount,
            teams: 0, // ChatMessages don't have teams
            rootMessages: rootCount,
            replies: replyCount,
        },
    };
}

/**
 * Helper to seed a conversation tree
 */
export async function seedConversationTree(
    prisma: any,
    options: {
        chatId: string;
        structure: Array<{
            id?: string;
            userId: string;
            text: string;
            parentId?: string;
            children?: any[];
        }>;
    }
): Promise<BulkSeedResult<any>> {
    const createdMessages: Record<string, any> = {};
    let totalCreated = 0;
    let botCount = 0;

    async function createNode(node: any, parentId?: string) {
        const messageId = node.id || generatePK();
        const messageData: Prisma.chat_messageCreateInput = {
            id: messageId,
            versionIndex: 0,
            language: "en",
            text: node.text,
            chatId: BigInt(options.chatId),
            userId: BigInt(node.userId),
            config: messageConfigFixtures.minimal as any,
        };

        if (parentId) {
            messageData.parentId = BigInt(parentId);
        }

        // Check if this is a bot message
        if (node.userId.includes('bot') || node.text.includes('bot')) {
            botCount++;
        }

        const message = await prisma.chatMessage.create({
            data: messageData,
            include: {
                translations: true,
                user: true,
            },
        });
        createdMessages[messageId] = message;
        totalCreated++;

        // Create children recursively
        if (node.children) {
            for (const child of node.children) {
                await createNode(child, messageId);
            }
        }

        return message;
    }

    // Create all root nodes
    for (const node of options.structure) {
        await createNode(node);
    }

    return {
        records: Object.values(createdMessages),
        summary: {
            total: totalCreated,
            withAuth: 0,
            bots: botCount,
            teams: 0,
            treeNodes: totalCreated,
        },
    };
}