import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ChatMessage model - used for seeding test data
 */

export class ChatMessageDbFactory {
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.ChatMessageCreateInput>
    ): Prisma.ChatMessageCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            versionIndex: 0,
            chat: { connect: { id: chatId } },
            user: { connect: { id: userId } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    text: "Test message",
                }],
            },
            ...overrides,
        };
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
 * Helper to seed chat messages for testing
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
) {
    const createdMessages = [];

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

    return createdMessages;
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
) {
    const createdMessages: Record<string, any> = {};

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

        const message = await prisma.chatMessage.create({
            data: messageData,
            include: {
                translations: true,
                user: true,
            },
        });
        createdMessages[messageId] = message;

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

    return createdMessages;
}