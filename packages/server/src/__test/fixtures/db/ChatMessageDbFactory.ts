import { type Prisma, type PrismaClient } from "@prisma/client";
import { type chat_message } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";
import { messageConfigFixtures } from "../../../../../shared/src/__test/fixtures/config/messageConfigFixtures.js";

interface ChatMessageRelationConfig extends RelationConfig {
    chat: { chatId: string };
    user?: { userId: string };
    parent?: { parentId: string };
    children?: number;
    reactions?: Array<{ userId: string; emoji: string }>;
}

/**
 * Enhanced database fixture factory for ChatMessage model
 * Provides comprehensive testing capabilities for chat messaging systems
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for message threading (parent/child relationships)
 * - Version control for edited messages
 * - Reaction and score tracking
 * - Language support
 * - Tool call and run configuration
 * - Predefined test scenarios
 */
export class ChatMessageDbFactory extends EnhancedDatabaseFactory<
    chat_message,
    Prisma.chat_messageCreateInput,
    Prisma.chat_messageInclude,
    Prisma.chat_messageUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("ChatMessage", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.chat_message;
    }

    /**
     * Get complete test fixtures for ChatMessage model
     */
    protected getFixtures(): DbTestFixtures<Prisma.chat_messageCreateInput, Prisma.chat_messageUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                language: "en",
                text: "Hello, world!",
                score: 0,
                versionIndex: 0,
                config: messageConfigFixtures.minimal as unknown as Prisma.InputJsonValue,
                chat: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                language: "en",
                text: "This is a complete message with all features enabled. It includes formatting, mentions, and more!",
                score: 42,
                versionIndex: 2,
                config: messageConfigFixtures.complete as unknown as Prisma.InputJsonValue,
                chat: { connect: { id: this.generateId() } },
                user: { connect: { id: this.generateId() } },
            },
            invalid: {
                missingRequired: {
                    // Missing id, language, text, chat
                    score: 0,
                    versionIndex: 0,
                },
                invalidTypes: {
                    id: "not-a-snowflake",
                    language: 123, // Should be string
                    text: null, // Should be string
                    score: "not-a-number", // Should be number
                    versionIndex: "zero", // Should be number
                    config: "not-an-object", // Should be object
                    chat: null, // Should be object
                },
                textTooLong: {
                    id: this.generateId(),
                    language: "en",
                    text: "a".repeat(32769), // Exceeds max length of 32768
                    score: 0,
                    versionIndex: 0,
                    chat: { connect: { id: this.generateId() } },
                },
                invalidLanguage: {
                    id: this.generateId(),
                    language: "invalid", // Should be valid language code
                    text: "Invalid language code",
                    score: 0,
                    versionIndex: 0,
                    chat: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                maxLengthMessage: {
                    id: this.generateId(),
                    language: "en",
                    text: "a".repeat(32768), // Max length
                    score: 0,
                    versionIndex: 0,
                    chat: { connect: { id: this.generateId() } },
                },
                unicodeMessage: {
                    id: this.generateId(),
                    language: "en",
                    text: "Hello 👋 World 🌍! Here's some Unicode: 你好世界 مرحبا بالعالم",
                    score: 0,
                    versionIndex: 0,
                    chat: { connect: { id: this.generateId() } },
                },
                multiLanguageMessage: {
                    id: this.generateId(),
                    language: "ja",
                    text: "こんにちは世界！これは日本語のメッセージです。",
                    score: 0,
                    versionIndex: 0,
                    chat: { connect: { id: this.generateId() } },
                },
                editedMessage: {
                    id: this.generateId(),
                    language: "en",
                    text: "This message has been edited multiple times",
                    score: 5,
                    versionIndex: 5, // High version index indicates multiple edits
                    chat: { connect: { id: this.generateId() } },
                },
                highScoreMessage: {
                    id: this.generateId(),
                    language: "en",
                    text: "This is a highly upvoted message",
                    score: 999,
                    versionIndex: 0,
                    chat: { connect: { id: this.generateId() } },
                },
                threadedMessage: {
                    id: this.generateId(),
                    language: "en",
                    text: "This is a reply in a thread",
                    score: 0,
                    versionIndex: 0,
                    chat: { connect: { id: this.generateId() } },
                    parent: { connect: { id: this.generateId() } },
                },
                botMessage: {
                    id: this.generateId(),
                    language: "en",
                    text: "I'm an AI assistant. Here's the information you requested:",
                    score: 0,
                    versionIndex: 0,
                    config: messageConfigFixtures.variants.assistantWithTools,
                    chat: { connect: { id: this.generateId() } },
                    user: { connect: { id: this.generateId() } },
                },
            },
            updates: {
                minimal: {
                    text: "Updated message text",
                    versionIndex: { increment: 1 },
                },
                complete: {
                    text: "Completely updated message with new content and configuration",
                    score: { increment: 10 },
                    versionIndex: { increment: 1 },
                    config: messageConfigFixtures.complete as unknown as Prisma.InputJsonValue,
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.chat_messageCreateInput>): Prisma.chat_messageCreateInput {
        return {
            id: this.generateId(),
            language: "en",
            text: "Hello!",
            score: 0,
            versionIndex: 0,
            config: messageConfigFixtures.minimal as unknown as Prisma.InputJsonValue,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.chat_messageCreateInput>): Prisma.chat_messageCreateInput {
        return {
            id: this.generateId(),
            language: "en",
            text: "This is a complete message with rich content, formatting, and metadata.",
            score: 0,
            versionIndex: 0,
            config: messageConfigFixtures.complete as unknown as Prisma.InputJsonValue,
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            simpleUserMessage: {
                name: "simpleUserMessage",
                description: "Basic user message",
                config: {
                    overrides: {
                        text: "Hey everyone!",
                        config: messageConfigFixtures.variants.userMessage,
                    },
                    chat: { chat: { connect: { id: this.generateId() } } },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            botResponse: {
                name: "botResponse",
                description: "AI bot response with tool usage",
                config: {
                    overrides: {
                        text: "I've searched for that information. Here's what I found:",
                        config: messageConfigFixtures.variants.assistantWithTools,
                    },
                    chat: { chat: { connect: { id: this.generateId() } } },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            threadConversation: {
                name: "threadConversation",
                description: "Threaded conversation with replies",
                config: {
                    overrides: {
                        text: "This is a reply to your question",
                    },
                    chat: { chat: { connect: { id: this.generateId() } } },
                    user: { user: { connect: { id: this.generateId() } } },
                    parent: { parent: { connect: { id: this.generateId() } } },
                },
            },
            editedMessage: {
                name: "editedMessage",
                description: "Message that has been edited",
                config: {
                    overrides: {
                        text: "Updated: This message has been corrected",
                        versionIndex: 3,
                    },
                    chat: { chat: { connect: { id: this.generateId() } } },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            popularMessage: {
                name: "popularMessage",
                description: "Highly upvoted message",
                config: {
                    overrides: {
                        text: "This is a really helpful answer!",
                        score: 150,
                    },
                    chat: { chat: { connect: { id: this.generateId() } } },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            systemMessage: {
                name: "systemMessage",
                description: "System notification message",
                config: {
                    overrides: {
                        text: "User joined the chat",
                        config: messageConfigFixtures.variants.systemMessage,
                    },
                    chat: { chat: { connect: { id: this.generateId() } } },
                    // No user for system messages
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.chat_messageInclude {
        return {
            chat: {
                select: {
                    id: true,
                    publicId: true,
                    isPrivate: true,
                },
            },
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                    isBot: true,
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
                        },
                    },
                },
            },
            children: {
                select: {
                    id: true,
                    text: true,
                    score: true,
                },
                orderBy: {
                    createdAt: "asc",
                },
            },
            _count: {
                select: {
                    children: true,
                    reactions: true,
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

        // Handle chat connection (required)
        if (config.chat) {
            data.chat = {
                connect: { id: BigInt(config.chat.chatId) },
            };
        } else {
            throw new Error("ChatMessage requires a chat connection");
        }

        // Handle user connection (optional - system messages may not have user)
        if (config.user) {
            data.user = {
                connect: { id: BigInt(config.user.userId) },
            };
        }

        // Handle parent message for threading
        if (config.parent) {
            data.parent = {
                connect: { id: BigInt(config.parent.parentId) },
            };
        }

        // Note: Child messages and reactions would be created separately

        return data;
    }

    /**
     * Create a user message
     */
    async createUserMessage(
        chatId: string,
        userId: string,
        text: string,
        language = "en",
    ): Promise<chat_message> {
        return await this.createWithRelations({
            overrides: {
                text,
                language,
                config: messageConfigFixtures.variants.userMessage,
            },
            chat: { chatId },
            user: { userId },
        });
    }

    /**
     * Create a bot message with tool usage
     */
    async createBotMessage(
        chatId: string,
        botId: string,
        text: string,
        toolCalls?: any[],
    ): Promise<chat_message> {
        const config = toolCalls 
            ? { ...messageConfigFixtures.variants.assistantWithTools, toolCalls }
            : messageConfigFixtures.variants.assistantWithTools;

        return await this.createWithRelations({
            overrides: {
                text,
                config,
            },
            chat: { chatId },
            user: { userId: botId },
        });
    }

    /**
     * Create a threaded reply
     */
    async createReply(
        parentMessageId: string,
        userId: string,
        text: string,
    ): Promise<chat_message> {
        // Get parent message to inherit chat
        const parent = await this.prisma.chat_message.findUnique({
            where: { id: parentMessageId },
            select: { chatId: true },
        });

        if (!parent) {
            throw new Error("Parent message not found");
        }

        return await this.createWithRelations({
            overrides: {
                text,
            },
            chat: { chatId: parent.chatId },
            user: { userId },
            parent: { parentId: parentMessageId },
        });
    }

    /**
     * Edit a message
     */
    async editMessage(messageId: string, newText: string): Promise<chat_message> {
        return await this.prisma.chat_message.update({
            where: { id: messageId },
            data: {
                text: newText,
                versionIndex: { increment: 1 },
            },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Update message score
     */
    async updateScore(messageId: string, delta: number): Promise<chat_message> {
        return await this.prisma.chat_message.update({
            where: { id: messageId },
            data: {
                score: { increment: delta },
            },
            include: this.getDefaultInclude(),
        });
    }

    protected async checkModelConstraints(record: ChatMessage): Promise<string[]> {
        const violations: string[] = [];
        
        // Check text length
        if (record.text.length > 32768) {
            violations.push("Message text exceeds maximum length of 32768 characters");
        }

        // Check language code format
        if (!/^[a-z]{2,3}$/.test(record.language)) {
            violations.push("Invalid language code format");
        }

        // Check version index
        if (record.versionIndex < 0) {
            violations.push("Version index cannot be negative");
        }

        // Check parent relationship
        if (record.parentId) {
            const parent = await this.prisma.chat_message.findUnique({
                where: { id: record.parentId },
                select: { chatId: true },
            });
            
            if (!parent) {
                violations.push("Parent message does not exist");
            } else if (parent.chatId !== record.chatId) {
                violations.push("Parent message must be in the same chat");
            }
        }

        // Check circular references
        if (record.parentId === record.id) {
            violations.push("Message cannot be its own parent");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            children: true,
            reactions: true,
            reactionSummaries: true,
            reports: true,
        };
    }

    protected async deleteRelatedRecords(
        record: chat_message,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete in order of dependencies
        
        // Delete child messages recursively
        if (shouldDelete("children") && record.children?.length) {
            await tx.chat_message.deleteMany({
                where: { parentId: record.id },
            });
        }

        // Delete reactions
        if (shouldDelete("reactions") && record.reactions?.length) {
            await tx.reaction.deleteMany({
                where: { chatMessageId: record.id },
            });
        }

        // Delete reaction summaries
        if (shouldDelete("reactionSummaries") && record.reactionSummaries?.length) {
            await tx.reactionSummary.deleteMany({
                where: { chatMessageId: record.id },
            });
        }

        // Delete reports
        if (shouldDelete("reports") && record.reports?.length) {
            await tx.report.deleteMany({
                where: { chatMessageId: record.id },
            });
        }
    }

    /**
     * Create a message thread
     */
    async createMessageThread(
        chatId: string,
        messages: Array<{ userId: string; text: string }>,
    ): Promise<chat_message[]> {
        const thread: chat_message[] = [];
        let parentId: string | undefined;

        for (const msg of messages) {
            const message = await this.createWithRelations({
                overrides: {
                    text: msg.text,
                },
                chat: { chatId },
                user: { userId: msg.userId },
                parent: parentId ? { parentId } : undefined,
            });

            thread.push(message);
            parentId = message.id; // Next message will reply to this one
        }

        return thread;
    }

    /**
     * Create a conversation between users
     */
    async createConversation(
        chatId: string,
        exchanges: Array<{ userId: string; text: string; isBot?: boolean }>,
    ): Promise<chat_message[]> {
        const messages: chat_message[] = [];

        for (const exchange of exchanges) {
            const config = exchange.isBot 
                ? messageConfigFixtures.variants.assistantWithTools
                : messageConfigFixtures.variants.userMessage;

            const message = await this.createWithRelations({
                overrides: {
                    text: exchange.text,
                    config,
                },
                chat: { chatId },
                user: { userId: exchange.userId },
            });

            messages.push(message);
        }

        return messages;
    }

    /**
     * Create test message scenarios
     */
    async createTestingScenarios(chatId: string): Promise<{
        userMessage: chat_message;
        botMessage: chat_message;
        threadedMessage: chat_message;
        editedMessage: chat_message;
        popularMessage: chat_message;
    }> {
        const userMessage = await this.createUserMessage(
            chatId,
            "user-123",
            "Hello, I have a question",
        );

        const botMessage = await this.createBotMessage(
            chatId,
            "bot-123",
            "I'd be happy to help! What would you like to know?",
        );

        const threadParent = await this.createUserMessage(
            chatId,
            "user-456",
            "This is the start of a thread",
        );

        const threadedMessage = await this.createReply(
            threadParent.id,
            "user-789",
            "This is a reply in the thread",
        );

        const toEdit = await this.createUserMessage(
            chatId,
            "user-edit",
            "This message will be edited",
        );

        const editedMessage = await this.editMessage(
            toEdit.id,
            "This message has been edited",
        );

        const popular = await this.createUserMessage(
            chatId,
            "user-popular",
            "This is a really helpful answer!",
        );

        const popularMessage = await this.updateScore(popular.id, 100);

        return {
            userMessage,
            botMessage,
            threadedMessage,
            editedMessage,
            popularMessage,
        };
    }
}

// Export factory creator function
export const createChatMessageDbFactory = (prisma: PrismaClient) => 
    new ChatMessageDbFactory(prisma);

// Export the class for type usage
export { ChatMessageDbFactory as ChatMessageDbFactoryClass };
