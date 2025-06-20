import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
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
export const chatMessageDbFixtures: DbTestFixtures<Prisma.ChatMessageCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        versionIndex: 0,
        chat: { connect: { id: chatMessageDbIds.chat1 } },
        user: { connect: { id: chatMessageDbIds.user1 } },
        translations: {
            create: [{
                id: generatePK(),
                language: "en",
                text: "Test message",
            }],
        },
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        versionIndex: 1,
        chat: { connect: { id: chatMessageDbIds.chat1 } },
        user: { connect: { id: chatMessageDbIds.user1 } },
        parent: { connect: { id: chatMessageDbIds.message1 } },
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    text: "This is a complete test message with detailed content and proper formatting.",
                },
                {
                    id: generatePK(),
                    language: "es",
                    text: "Este es un mensaje de prueba completo con contenido detallado y formato adecuado.",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required chat, user, versionIndex, and translations
            publicId: generatePublicId(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            versionIndex: "invalid", // Should be number
            chat: "invalid-chat-reference", // Should be connect object
            user: "invalid-user-reference", // Should be connect object
            parent: "invalid-parent-reference", // Should be connect object
        },
        negativeVersionIndex: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: -1, // Negative version index
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Invalid version index",
                }],
            },
        },
        emptyTranslations: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user1 } },
            translations: {
                create: [], // Empty translations array
            },
        },
    },
    edgeCases: {
        rootMessage: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Root message without parent",
                }],
            },
        },
        replyMessage: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user2 } },
            parent: { connect: { id: chatMessageDbIds.message1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Reply to root message",
                }],
            },
        },
        botMessage: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.bot1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "This is an automated bot response with helpful information.",
                }],
            },
        },
        longMessage: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "This is a very long message that contains a lot of text to test how the system handles messages with substantial content. ".repeat(10),
                }],
            },
        },
        multiLanguageMessage: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user1 } },
            translations: {
                create: [
                    { id: generatePK(), language: "en", text: "Hello world" },
                    { id: generatePK(), language: "es", text: "Hola mundo" },
                    { id: generatePK(), language: "fr", text: "Bonjour le monde" },
                    { id: generatePK(), language: "de", text: "Hallo Welt" },
                    { id: generatePK(), language: "ja", text: "こんにちは世界" },
                ],
            },
        },
        highVersionIndex: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 999,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Message with high version index",
                }],
            },
        },
        specialCharactersMessage: {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatMessageDbIds.chat1 } },
            user: { connect: { id: chatMessageDbIds.user1 } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Message with special chars: !@#$%^&*()_+{}|:<>?[]\\;'\",./ and emojis 😀😁😂",
                }],
            },
        },
    },
};

/**
 * Enhanced factory for creating chat message database fixtures
 */
export class ChatMessageDbFactory extends EnhancedDbFactory<Prisma.ChatMessageCreateInput> {
    
    /**
     * Get the test fixtures for ChatMessage model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ChatMessageCreateInput> {
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
                    publicId: generatePublicId(),
                    versionIndex: 0,
                    chat: { connect: { id: chatMessageDbIds.chat1 } },
                    user: { connect: { id: chatMessageDbIds.user1 } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Duplicate message",
                        }],
                    },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    versionIndex: 0,
                    chat: { connect: { id: "non-existent-chat-id" } },
                    user: { connect: { id: chatMessageDbIds.user1 } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Foreign key violation",
                        }],
                    },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId
                    versionIndex: 0,
                    chat: { connect: { id: chatMessageDbIds.chat1 } },
                    user: { connect: { id: chatMessageDbIds.user1 } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Check constraint violation",
                        }],
                    },
                },
            },
            validation: {
                requiredFieldMissing: chatMessageDbFixtures.invalid.missingRequired,
                invalidDataType: chatMessageDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: "a".repeat(500), // PublicId too long
                    versionIndex: 999999, // Very high version index
                    chat: { connect: { id: chatMessageDbIds.chat1 } },
                    user: { connect: { id: chatMessageDbIds.user1 } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "a".repeat(100000), // Extremely long text
                        }],
                    },
                },
            },
            businessLogic: {
                circularParent: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    versionIndex: 0,
                    chat: { connect: { id: chatMessageDbIds.chat1 } },
                    user: { connect: { id: chatMessageDbIds.user1 } },
                    parent: { connect: { id: chatMessageDbIds.message1 } }, // Would create circular reference
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            text: "Circular parent reference",
                        }],
                    },
                },
                emptyTranslations: chatMessageDbFixtures.invalid.emptyTranslations,
            },
        };
    }

    /**
     * ChatMessage-specific validation
     */
    protected validateSpecific(data: Prisma.ChatMessageCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to ChatMessage
        if (!data.chat) errors.push("ChatMessage chat is required");
        if (!data.user) errors.push("ChatMessage user is required");
        if (!data.publicId) errors.push("ChatMessage publicId is required");
        if (data.versionIndex === undefined) errors.push("ChatMessage versionIndex is required");
        if (!data.translations) errors.push("ChatMessage translations are required");

        // Check version index
        if (typeof data.versionIndex === 'number') {
            if (data.versionIndex < 0) {
                errors.push("Version index cannot be negative");
            }
            if (data.versionIndex > 1000) {
                warnings.push("Very high version index (>1000)");
            }
        }

        // Check translations
        if (data.translations && typeof data.translations === 'object' && 'create' in data.translations) {
            const createTranslations = data.translations.create;
            if (Array.isArray(createTranslations)) {
                if (createTranslations.length === 0) {
                    errors.push("Message must have at least one translation");
                }
                createTranslations.forEach((trans, index) => {
                    if (!trans.text || trans.text.length === 0) {
                        errors.push(`Translation ${index} has empty text`);
                    }
                    if (trans.text && trans.text.length > 10000) {
                        warnings.push(`Translation ${index} text is very long (>10000 chars)`);
                    }
                });
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.ChatMessageCreateInput>
    ): Prisma.ChatMessageCreateInput {
        const factory = new ChatMessageDbFactory();
        return factory.createMinimal({
            chat: { connect: { id: chatId } },
            user: { connect: { id: userId } },
            versionIndex: 0,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Test message",
                }],
            },
            ...overrides,
        });
    }

    static createWithParent(
        chatId: string,
        userId: string,
        parentId: string,
        overrides?: Partial<Prisma.ChatMessageCreateInput>
    ): Prisma.ChatMessageCreateInput {
        return this.createMinimal(chatId, userId, {
            parent: { connect: { id: parentId } },
            ...overrides,
        });
    }

    static createBotMessage(
        chatId: string,
        botId: string,
        overrides?: Partial<Prisma.ChatMessageCreateInput>
    ): Prisma.ChatMessageCreateInput {
        return this.createMinimal(chatId, botId, {
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "This is an automated bot response.",
                }],
            },
            ...overrides,
        });
    }

    static createSequence(
        chatId: string,
        userId: string,
        count: number,
        startIndex: number = 0
    ): Prisma.ChatMessageCreateInput[] {
        const messages = [];
        for (let i = 0; i < count; i++) {
            messages.push(this.createMinimal(chatId, userId, {
                versionIndex: startIndex + i,
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: `Message ${startIndex + i + 1} in sequence`,
                    }],
                },
            }));
        }
        return messages;
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
        const messageData: Prisma.ChatMessageCreateInput = {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: msg.versionIndex ?? 0,
            chat: { connect: { id: options.chatId } },
            user: { connect: { id: msg.userId } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: msg.text || "Test message",
                }],
            },
        };

        if (msg.parentId) {
            messageData.parent = { connect: { id: msg.parentId } };
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
        const messageData: Prisma.ChatMessageCreateInput = {
            id: messageId,
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: options.chatId } },
            user: { connect: { id: node.userId } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: node.text,
                }],
            },
        };

        if (parentId) {
            messageData.parent = { connect: { id: parentId } };
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
            treeStructure: true,
        },
    };
}