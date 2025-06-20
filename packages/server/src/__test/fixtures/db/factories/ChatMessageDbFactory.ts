import { type ChatMessageCreateInput, type ChatMessageUpdateInput, generatePK } from "@vrooli/shared";
import { nanoid } from "nanoid";
import { type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";
import { messageConfigFixtures } from "../../../../../shared/src/__test/fixtures/config/messageConfigFixtures.js";

interface ChatMessageRelationConfig extends RelationConfig {
    chatId?: string;
    userId?: string;
    parentId?: string;
    withReplies?: boolean | number;
    isBot?: boolean;
    withToolCalls?: boolean;
    messageType?: 'user' | 'assistant' | 'system' | 'tool';
}

/**
 * Database fixture factory for ChatMessage model
 * Handles threaded conversations, AI responses, file attachments, and tool calls
 */
export class ChatMessageDbFactory extends DatabaseFixtureFactory<
    any,
    ChatMessageCreateInput,
    any,
    ChatMessageUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('chat_message', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_message;
    }

    protected getMinimalData(overrides?: Partial<ChatMessageCreateInput>): ChatMessageCreateInput {
        // Generate placeholder IDs - these should be overridden with actual chat/user IDs
        const chatId = BigInt(generatePK());
        const userId = BigInt(generatePK());
        
        return {
            id: generatePK(),
            versionIndex: 0,
            language: "en",
            text: "Test message",
            chatId,
            userId,
            config: messageConfigFixtures.minimal as any,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<ChatMessageCreateInput>): ChatMessageCreateInput {
        // Generate placeholder IDs - these should be overridden with actual chat/user IDs
        const chatId = BigInt(generatePK());
        const userId = BigInt(generatePK());
        
        return {
            id: generatePK(),
            versionIndex: 1,
            language: "en",
            text: "This is a complete test message with detailed content and proper formatting.",
            chatId,
            userId,
            config: messageConfigFixtures.complete as any,
            score: 5,
            ...overrides,
        };
    }

    /**
     * Create message with specific chat and user
     */
    async createMessage(
        chatId: string, 
        userId: string, 
        text: string,
        overrides?: Partial<ChatMessageCreateInput>
    ): Promise<any> {
        const data: ChatMessageCreateInput = {
            ...this.getMinimalData(),
            chatId: BigInt(chatId),
            userId: BigInt(userId),
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
        chatId: string, 
        userId: string, 
        text: string,
        overrides?: Partial<ChatMessageCreateInput>
    ): Promise<any> {
        return this.createMessage(chatId, userId, text, {
            config: messageConfigFixtures.variants.userMessage as any,
            ...overrides,
        });
    }

    /**
     * Create bot response message
     */
    async createBotMessage(
        chatId: string, 
        botId: string, 
        text: string,
        overrides?: Partial<ChatMessageCreateInput>
    ): Promise<any> {
        return this.createMessage(chatId, botId, text, {
            config: messageConfigFixtures.variants.assistantWithTools as any,
            ...overrides,
        });
    }

    /**
     * Create reply message
     */
    async createReply(
        chatId: string, 
        userId: string, 
        parentId: string,
        text: string,
        overrides?: Partial<ChatMessageCreateInput>
    ): Promise<any> {
        return this.createMessage(chatId, userId, text, {
            parentId: BigInt(parentId),
            ...overrides,
        });
    }

    /**
     * Create bot message with tool calls
     */
    async createBotWithTools(
        chatId: string, 
        botId: string, 
        text: string,
        toolCalls?: any[]
    ): Promise<any> {
        const config = toolCalls 
            ? {
                ...messageConfigFixtures.variants.assistantWithTools,
                toolCalls,
            }
            : messageConfigFixtures.variants.assistantWithTools;

        return this.createMessage(chatId, botId, text, {
            config: config as any,
        });
    }

    /**
     * Create system message
     */
    async createSystemMessage(
        chatId: string, 
        text: string,
        overrides?: Partial<ChatMessageCreateInput>
    ): Promise<any> {
        // System messages use a special user ID (could be null or system user)
        const systemUserId = BigInt(generatePK()); // In real system, this would be a system user
        
        return this.createMessage(chatId, systemUserId.toString(), text, {
            config: messageConfigFixtures.variants.systemMessage as any,
            ...overrides,
        });
    }

    protected getDefaultInclude(): any {
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
                        where: { language: 'en' },
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
            replies: {
                take: 3,
                select: {
                    id: true,
                    text: true,
                    createdAt: true,
                    user: {
                        select: {
                            id: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
                orderBy: { createdAt: 'asc' },
            },
        };
    }

    protected async applyRelationships(
        baseData: ChatMessageCreateInput,
        config: ChatMessageRelationConfig,
        tx: any
    ): Promise<ChatMessageCreateInput> {
        let data = { ...baseData };

        // Apply chat and user connections
        if (config.chatId) {
            data.chatId = BigInt(config.chatId);
        }
        
        if (config.userId) {
            data.userId = BigInt(config.userId);
        }

        // Apply parent relationship
        if (config.parentId) {
            data.parentId = BigInt(config.parentId);
        }

        // Apply message type configuration
        if (config.messageType) {
            switch (config.messageType) {
                case 'user':
                    data.config = messageConfigFixtures.variants.userMessage as any;
                    break;
                case 'assistant':
                    data.config = messageConfigFixtures.variants.assistantWithTools as any;
                    break;
                case 'system':
                    data.config = messageConfigFixtures.variants.systemMessage as any;
                    break;
                case 'tool':
                    data.config = messageConfigFixtures.variants.toolErrorMessage as any;
                    break;
            }
        }

        // Handle tool calls
        if (config.withToolCalls) {
            data.config = {
                ...messageConfigFixtures.variants.assistantWithTools,
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
        chatId: string,
        participants: { userId: string; botId: string },
        messages: Array<{
            type: "user" | "bot";
            text: string;
            score?: number;
            parentId?: string;
        }>
    ): Promise<any[]> {
        const createdMessages: any[] = [];

        for (let i = 0; i < messages.length; i++) {
            const msg = messages[i];
            const isBot = msg.type === "bot";
            const senderId = isBot ? participants.botId : participants.userId;
            
            const message = await this.createMessage(chatId, senderId, msg.text, {
                versionIndex: i,
                parentId: msg.parentId ? BigInt(msg.parentId) : undefined,
                score: msg.score || 0,
                config: isBot 
                    ? messageConfigFixtures.variants.assistantWithTools as any
                    : messageConfigFixtures.variants.userMessage as any,
            });
            
            createdMessages.push(message);
        }

        return createdMessages;
    }

    /**
     * Create a threaded conversation
     */
    async createThread(
        chatId: string,
        userId: string,
        rootText: string,
        replies: Array<{ userId: string; text: string; isBot?: boolean }>
    ): Promise<{ root: any; replies: any[] }> {
        // Create root message
        const root = await this.createUserMessage(chatId, userId, rootText);

        // Create replies
        const replyMessages: any[] = [];
        for (const reply of replies) {
            const message = reply.isBot 
                ? await this.createBotMessage(chatId, reply.userId, reply.text, {
                    parentId: BigInt(root.id),
                })
                : await this.createUserMessage(chatId, reply.userId, reply.text, {
                    parentId: BigInt(root.id),
                });
            replyMessages.push(message);
        }

        return { root, replies: replyMessages };
    }

    protected async checkModelConstraints(record: any): Promise<string[]> {
        const violations: string[] = [];
        
        // Check text is not empty
        if (!record.text || record.text.trim().length === 0) {
            violations.push('Message text cannot be empty');
        }

        // Check text length
        if (record.text && record.text.length > 32768) {
            violations.push('Message text exceeds maximum length of 32768 characters');
        }

        // Check version index is non-negative
        if (record.versionIndex < 0) {
            violations.push('Version index cannot be negative');
        }

        // Check language code format
        if (!record.language || record.language.length < 2 || record.language.length > 3) {
            violations.push('Language code must be 2-3 characters');
        }

        // Check chat exists
        const chat = await this.prisma.chat.findUnique({
            where: { id: record.chatId },
        });
        if (!chat) {
            violations.push('Referenced chat does not exist');
        }

        // Check user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.userId },
        });
        if (!user) {
            violations.push('Referenced user does not exist');
        }

        // Check parent exists if specified
        if (record.parentId) {
            const parent = await this.prisma.chat_message.findUnique({
                where: { id: record.parentId },
            });
            if (!parent) {
                violations.push('Referenced parent message does not exist');
            } else if (parent.chatId !== record.chatId) {
                violations.push('Parent message must be in the same chat');
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
    getEdgeCaseScenarios(): Record<string, ChatMessageCreateInput> {
        const baseChatId = BigInt(generatePK());
        const baseUserId = BigInt(generatePK());
        const botUserId = BigInt(generatePK());
        
        return {
            rootMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                text: "Root message without parent",
                config: messageConfigFixtures.variants.userMessage as any,
            },
            replyMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                text: "Reply to root message",
                parentId: BigInt(generatePK()), // Would be set to actual parent ID
                config: messageConfigFixtures.minimal as any,
            },
            botMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: botUserId,
                text: "This is an automated bot response with helpful information.",
                config: messageConfigFixtures.variants.assistantWithTools as any,
            },
            longMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                text: "This is a very long message that contains a lot of text to test how the system handles messages with substantial content. ".repeat(10),
            },
            specialCharactersMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                text: "Message with special chars: !@#$%^&*()_+{}|:<>?[]\\;'\",./ and emojis 😀😁😂🚀💬",
            },
            multiLanguageMessage: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                language: "es",
                text: "Mensaje en español con caracteres especiales: ñáéíóú çñü",
            },
            highVersionIndex: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: baseUserId,
                versionIndex: 999,
                text: "Message with high version index",
            },
            messageWithToolCalls: {
                ...this.getMinimalData(),
                chatId: baseChatId,
                userId: botUserId,
                text: "I'll search for that information.",
                config: {
                    ...messageConfigFixtures.variants.assistantWithTools,
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
                chatId: baseChatId,
                userId: botUserId,
                text: "Processing your request...",
                config: messageConfigFixtures.variants.messageWithMultipleRuns as any,
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            replies: true,
        };
    }

    protected async deleteRelatedRecords(
        record: any,
        remainingDepth: number,
        tx: any
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
    async createTypingIndicator(chatId: string, userId: string): Promise<any> {
        return this.createMessage(chatId, userId, "...", {
            config: {
                ...messageConfigFixtures.minimal,
                role: "system",
                eventTopic: "typing-indicator",
            } as any,
            score: -1,
        });
    }

    async createErrorMessage(chatId: string, botId: string, errorText: string): Promise<any> {
        return this.createMessage(chatId, botId, errorText, {
            config: messageConfigFixtures.variants.toolErrorMessage as any,
            score: -2,
        });
    }

    async createBroadcastMessage(chatId: string, userId: string, text: string): Promise<any> {
        return this.createMessage(chatId, userId, text, {
            config: messageConfigFixtures.variants.broadcastMessage as any,
        });
    }
}

// Export factory creator function
export const createChatMessageDbFactory = (prisma: PrismaClient) => new ChatMessageDbFactory(prisma);