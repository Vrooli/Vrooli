import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type chat_message, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

// TODO: Update this when messageConfigFixtures is available
const messageConfigFixtures = {
    minimal: { __version: "1.0.0", role: "user" },
    complete: { __version: "1.0.0", role: "user", turnId: 1 },
    variants: {
        assistantMessage: { __version: "1.0.0", role: "assistant" },
        systemMessage: { __version: "1.0.0", role: "system" },
        toolMessage: { __version: "1.0.0", role: "tool" },
    },
};

interface ChatMessageRelationConfig extends RelationConfig {
    chatId?: string;
    userId?: string;
    parentId?: string;
    withReplies?: boolean | number;
    isBot?: boolean;
    withToolCalls?: boolean;
    messageType?: "user" | "assistant" | "system" | "tool";
}

/**
 * Database fixture factory for ChatMessage model
 * Handles threaded conversations, AI responses, file attachments, and tool calls
 */
export class ChatMessageDbFactory extends DatabaseFixtureFactory<
    chat_message,
    Prisma.chat_messageCreateInput,
    Prisma.chat_messageInclude,
    Prisma.chat_messageUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("chat_message", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_message;
    }

    protected getMinimalData(overrides?: Partial<Prisma.chat_messageCreateInput>): Prisma.chat_messageCreateInput {
        return {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Test message",
            chat: { connect: { id: generatePK() } },
            user: { connect: { id: generatePK() } },
            config: messageConfigFixtures.minimal as any,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.chat_messageCreateInput>): Prisma.chat_messageCreateInput {
        return {
            id: generatePK(),
            versionIndex: 1,
            language: "en",
            text: "This is a complete test message with detailed content and proper formatting.",
            chat: { connect: { id: generatePK() } },
            user: { connect: { id: generatePK() } },
            config: messageConfigFixtures.complete as any,
            score: 5,
            ...overrides,
        };
    }

    /**
     * Create message with specific chat and user
     */
    async createMessage(
        chatId: string | bigint, 
        userId: string | bigint, 
        text: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>,
    ): Promise<chat_message> {
        const data: Prisma.chat_messageCreateInput = {
            ...this.getMinimalData(),
            chat: { connect: { id: BigInt(chatId) } },
            user: { connect: { id: BigInt(userId) } },
            text,
            ...overrides,
        };
        
        const result = await this.prisma.chat_message.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create user message
     */
    async createUserMessage(
        chatId: string | bigint, 
        userId: string | bigint, 
        text: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>,
    ): Promise<chat_message> {
        return this.createMessage(chatId, userId, text, {
            config: messageConfigFixtures.minimal as any,
            ...overrides,
        });
    }

    /**
     * Create bot response message
     */
    async createBotMessage(
        chatId: string | bigint, 
        botId: string | bigint, 
        text: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>,
    ): Promise<chat_message> {
        return this.createMessage(chatId, botId, text, {
            config: messageConfigFixtures.variants.assistantMessage as any,
            ...overrides,
        });
    }

    /**
     * Create reply message
     */
    async createReply(
        chatId: string | bigint, 
        userId: string | bigint, 
        parentId: string | bigint,
        text: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>,
    ): Promise<chat_message> {
        return this.createMessage(chatId, userId, text, {
            parent: { connect: { id: BigInt(parentId) } },
            ...overrides,
        });
    }

    /**
     * Create bot message with tool calls
     */
    async createBotWithTools(
        chatId: string | bigint, 
        botId: string | bigint, 
        text: string,
        toolCalls?: any[],
    ): Promise<chat_message> {
        const config = toolCalls 
            ? {
                ...messageConfigFixtures.variants.toolMessage,
                toolCalls,
            }
            : messageConfigFixtures.variants.toolMessage;

        return this.createMessage(chatId, botId, text, {
            config: config as any,
        });
    }

    /**
     * Create system message
     */
    async createSystemMessage(
        chatId: string | bigint, 
        text: string,
        overrides?: Partial<Prisma.chat_messageCreateInput>,
    ): Promise<chat_message> {
        // System messages use a special user ID (could be null or system user)
        const systemUserId = BigInt(generatePK()); // In real system, this would be a system user
        
        return this.createMessage(chatId, systemUserId.toString(), text, {
            config: messageConfigFixtures.variants.systemMessage as any,
            ...overrides,
        });
    }

    protected getDefaultInclude(): Prisma.chat_messageInclude {
        return {
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                    isBot: true,
                    profileImage: true,
                },
            },
            chat: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                    translations: {
                        where: { language: "en" },
                        select: {
                            name: true,
                        },
                        take: 1,
                    },
                },
            },
            parent: {
                select: {
                    id: true,
                    text: true,
                    user: {
                        select: {
                            id: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.chat_messageCreateInput,
        config: ChatMessageRelationConfig,
        tx: any,
    ): Promise<Prisma.chat_messageCreateInput> {
        const data = { ...baseData };

        // Apply chat and user connections
        if (config.chatId) {
            data.chat = { connect: { id: BigInt(config.chatId) } };
        }
        
        if (config.userId) {
            data.user = { connect: { id: BigInt(config.userId) } };
        }

        // Apply parent relationship
        if (config.parentId) {
            data.parent = { connect: { id: BigInt(config.parentId) } };
        }

        // Apply message type configuration
        if (config.messageType) {
            switch (config.messageType) {
                case "user":
                    data.config = messageConfigFixtures.minimal as any;
                    break;
                case "assistant":
                    data.config = messageConfigFixtures.variants.assistantMessage as any;
                    break;
                case "system":
                    data.config = messageConfigFixtures.variants.systemMessage as any;
                    break;
                case "tool":
                    data.config = messageConfigFixtures.variants.toolMessage as any;
                    break;
            }
        }

        // Handle tool calls
        if (config.withToolCalls) {
            data.config = {
                ...messageConfigFixtures.variants.toolMessage,
                toolCalls: [
                    {
                        id: `call_${nanoid(8)}`,
                        function: {
                            name: "search_documents",
                            arguments: JSON.stringify({ query: "test", limit: 5 }),
                        },
                        result: {
                            success: true,
                            output: "Search completed successfully",
                        },
                    },
                ],
            } as any;
        }

        return data;
    }

    /**
     * Create a conversation sequence
     */
    async createConversation(
        chatId: string | bigint,
        participants: { userId: string | bigint; botId: string | bigint },
        messages: Array<{
            type: "user" | "bot";
            text: string;
            score?: number;
            parentId?: string | bigint;
        }>,
    ): Promise<chat_message[]> {
        const createdMessages: chat_message[] = [];

        for (let i = 0; i < messages.length; i++) {
            const msg = messages[i];
            const isBot = msg.type === "bot";
            const senderId = isBot ? participants.botId : participants.userId;
            
            const message = await this.createMessage(chatId, senderId, msg.text, {
                versionIndex: i,
                parent: msg.parentId ? { connect: { id: BigInt(msg.parentId) } } : undefined,
                score: msg.score || 0,
                config: isBot 
                    ? messageConfigFixtures.variants.assistantMessage as any
                    : messageConfigFixtures.minimal as any,
            });
            
            createdMessages.push(message);
        }

        return createdMessages;
    }

    /**
     * Create a threaded conversation
     */
    async createThread(
        chatId: string | bigint,
        userId: string | bigint,
        rootText: string,
        replies: Array<{ userId: string | bigint; text: string; isBot?: boolean }>,
    ): Promise<{ root: chat_message; replies: chat_message[] }> {
        // Create root message
        const root = await this.createUserMessage(chatId, userId, rootText);

        // Create replies
        const replyMessages: chat_message[] = [];
        for (const reply of replies) {
            const message = reply.isBot 
                ? await this.createBotMessage(chatId, reply.userId, reply.text, {
                    parent: { connect: { id: BigInt(root.id) } },
                })
                : await this.createUserMessage(chatId, reply.userId, reply.text, {
                    parent: { connect: { id: BigInt(root.id) } },
                });
            replyMessages.push(message);
        }

        return { root, replies: replyMessages };
    }

    protected async checkModelConstraints(record: chat_message): Promise<string[]> {
        const violations: string[] = [];
        
        // Check text is not empty
        if (!record.text || record.text.trim().length === 0) {
            violations.push("Message text cannot be empty");
        }

        // Check text length
        if (record.text && record.text.length > 32768) {
            violations.push("Message text exceeds maximum length of 32768 characters");
        }

        // Check version index is non-negative
        if (record.versionIndex < 0) {
            violations.push("Version index cannot be negative");
        }

        // Check language code format
        if (!record.language || record.language.length < 2 || record.language.length > 3) {
            violations.push("Language code must be 2-3 characters");
        }

        // Check chat exists
        const chat = await this.prisma.chat.findUnique({
            where: { id: record.chatId },
        });
        if (!chat) {
            violations.push("Referenced chat does not exist");
        }

        // Check user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.userId },
        });
        if (!user) {
            violations.push("Referenced user does not exist");
        }

        // Check parent exists if specified
        if (record.parentId) {
            const parent = await this.prisma.chat_message.findUnique({
                where: { id: record.parentId },
            });
            if (!parent) {
                violations.push("Referenced parent message does not exist");
            } else if (parent.chatId !== record.chatId) {
                violations.push("Parent message must be in the same chat");
            }
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, chatId, userId, versionIndex, language, text
                id: generatePK(),
            },
            invalidTypes: {
                id: "not-a-snowflake",
                language: 123, // Should be string
                text: null, // Should be string
                versionIndex: "invalid", // Should be number
                chatId: "invalid-chat-reference", // Should be BigInt
                userId: "invalid-user-reference", // Should be BigInt
                parentId: "invalid-parent-reference", // Should be BigInt
            },
            emptyText: {
                id: generatePK(),
                versionIndex: 0,
                language: "en",
                text: "", // Empty text
                chatId: BigInt(generatePK()),
                userId: BigInt(generatePK()),
                config: messageConfigFixtures.minimal as any,
            },
            negativeVersionIndex: {
                id: generatePK(),
                versionIndex: -1, // Negative version index
                language: "en",
                text: "Invalid version index",
                chatId: BigInt(generatePK()),
                userId: BigInt(generatePK()),
                config: messageConfigFixtures.minimal as any,
            },
            invalidLanguage: {
                id: generatePK(),
                versionIndex: 0,
                language: "x", // Too short
                text: "Invalid language",
                chatId: BigInt(generatePK()),
                userId: BigInt(generatePK()),
                config: messageConfigFixtures.minimal as any,
            },
            textTooLong: {
                id: generatePK(),
                versionIndex: 0,
                language: "en",
                text: "a".repeat(50000), // Too long
                chatId: BigInt(generatePK()),
                userId: BigInt(generatePK()),
                config: messageConfigFixtures.minimal as any,
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.chat_messageCreateInput> {
        const baseChatId = BigInt(generatePK());
        const baseUserId = BigInt(generatePK());
        const botUserId = BigInt(generatePK());
        
        return {
            rootMessage: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: baseUserId } },
                text: "Root message without parent",
                config: messageConfigFixtures.minimal as any,
            },
            replyMessage: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: baseUserId } },
                text: "Reply to root message",
                parent: { connect: { id: BigInt(generatePK()) } }, // Would be set to actual parent ID
                config: messageConfigFixtures.minimal as any,
            },
            botMessage: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: botUserId } },
                text: "This is an automated bot response with helpful information.",
                config: messageConfigFixtures.variants.assistantMessage as any,
            },
            longMessage: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: baseUserId } },
                text: "This is a very long message that contains a lot of text to test how the system handles messages with substantial content. ".repeat(10),
            },
            specialCharactersMessage: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: baseUserId } },
                text: "Message with special chars: !@#$%^&*()_+{}|:<>?[]\\;'\",./ and emojis 😀😁😂🚀💬",
            },
            multiLanguageMessage: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: baseUserId } },
                language: "es",
                text: "Mensaje en español con caracteres especiales: ñáéíóú çñü",
            },
            highVersionIndex: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: baseUserId } },
                versionIndex: 999,
                text: "Message with high version index",
            },
            messageWithToolCalls: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: botUserId } },
                text: "I'll search for that information.",
                config: {
                    ...messageConfigFixtures.variants.toolMessage,
                    toolCalls: [
                        {
                            id: "call_search_1",
                            function: {
                                name: "web_search",
                                arguments: JSON.stringify({ query: "test query", limit: 10 }),
                            },
                            result: {
                                success: true,
                                output: { results: ["result1", "result2"] },
                            },
                        },
                    ],
                } as any,
            },
            messageWithRuns: {
                ...this.getMinimalData(),
                chat: { connect: { id: baseChatId } },
                user: { connect: { id: botUserId } },
                text: "Processing your request...",
                config: messageConfigFixtures.variants.systemMessage as any,
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            replies: true,
        };
    }

    protected async deleteRelatedRecords(
        record: chat_message & {
            replies?: any[];
        },
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete reply messages first (children)
        if (record.replies?.length && remainingDepth > 0) {
            for (const reply of record.replies) {
                await this.deleteRelatedRecords(reply, remainingDepth - 1, tx);
                await tx.chat_message.delete({ where: { id: reply.id } });
            }
        }
    }

    /**
     * Test scenario helpers
     */
    async createTypingIndicator(chatId: string | bigint, userId: string | bigint): Promise<chat_message> {
        return this.createMessage(chatId, userId, "...", {
            config: {
                ...messageConfigFixtures.minimal,
                role: "system",
                eventTopic: "typing-indicator",
            } as any,
            score: -1,
        });
    }

    async createErrorMessage(chatId: string | bigint, botId: string | bigint, errorText: string): Promise<chat_message> {
        return this.createMessage(chatId, botId, errorText, {
            config: messageConfigFixtures.variants.systemMessage as any,
            score: -2,
        });
    }

    async createBroadcastMessage(chatId: string | bigint, userId: string | bigint, text: string): Promise<chat_message> {
        return this.createMessage(chatId, userId, text, {
            config: messageConfigFixtures.variants.systemMessage as any,
        });
    }
}

// Export factory creator function
export const createChatMessageDbFactory = (prisma: PrismaClient) => new ChatMessageDbFactory(prisma);
