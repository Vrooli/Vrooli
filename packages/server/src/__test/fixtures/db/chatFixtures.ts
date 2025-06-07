import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Chat model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const chatDbIds = {
    chat1: generatePK(),
    chat2: generatePK(),
    chat3: generatePK(),
    invite1: generatePK(),
    invite2: generatePK(),
    message1: generatePK(),
    message2: generatePK(),
    participant1: generatePK(),
    participant2: generatePK(),
    participant3: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
};

/**
 * Minimal chat data for database creation
 */
export const minimalChatDb: Prisma.ChatCreateInput = {
    id: chatDbIds.chat1,
    publicId: generatePublicId(),
    isPrivate: false,
    openToAnyoneWithInvite: true,
};

/**
 * Private chat with participants
 */
export const privateChatDb: Prisma.ChatCreateInput = {
    id: chatDbIds.chat2,
    publicId: generatePublicId(),
    isPrivate: true,
    openToAnyoneWithInvite: false,
    translations: {
        create: [
            {
                id: chatDbIds.translation1,
                language: "en",
                name: "Private Team Chat",
                description: "Discussion for team members only",
            },
        ],
    },
};

/**
 * Complete chat with all features
 */
export const completeChatDb: Prisma.ChatCreateInput = {
    id: chatDbIds.chat3,
    publicId: generatePublicId(),
    isPrivate: false,
    openToAnyoneWithInvite: true,
    translations: {
        create: [
            {
                id: chatDbIds.translation2,
                language: "en",
                name: "Project Discussion",
                description: "Open discussion about the project",
            },
            {
                id: chatDbIds.translation3,
                language: "es",
                name: "Discusión del Proyecto",
                description: "Discusión abierta sobre el proyecto",
            },
        ],
    },
};

/**
 * Factory for creating chat database fixtures with overrides
 */
export class ChatDbFactory {
    static createMinimal(overrides?: Partial<Prisma.ChatCreateInput>): Prisma.ChatCreateInput {
        return {
            ...minimalChatDb,
            id: generatePK(),
            publicId: generatePublicId(),
            ...overrides,
        };
    }

    static createPrivate(overrides?: Partial<Prisma.ChatCreateInput>): Prisma.ChatCreateInput {
        return {
            ...privateChatDb,
            id: generatePK(),
            publicId: generatePublicId(),
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.ChatCreateInput>): Prisma.ChatCreateInput {
        return {
            ...completeChatDb,
            id: generatePK(),
            publicId: generatePublicId(),
            ...overrides,
        };
    }

    /**
     * Create chat with participants - requires user connections
     */
    static createWithParticipants(
        userIds: string[], 
        overrides?: Partial<Prisma.ChatCreateInput>
    ): Prisma.ChatCreateInput {
        return {
            ...this.createMinimal(overrides),
            participants: {
                create: userIds.map((userId, index) => ({
                    id: generatePK(),
                    user: { connect: { id: userId } },
                })),
            },
        };
    }

    /**
     * Create chat with invites
     */
    static createWithInvites(
        creatorId: string,
        inviteMessages: string[] = ["Join our discussion!"],
        overrides?: Partial<Prisma.ChatCreateInput>
    ): Prisma.ChatCreateInput {
        return {
            ...this.createMinimal(overrides),
            invites: {
                create: inviteMessages.map(message => ({
                    id: generatePK(),
                    message,
                    user: { connect: { id: creatorId } },
                })),
            },
        };
    }

    /**
     * Create chat with initial messages
     */
    static createWithMessages(
        messages: Array<{ userId: string; text: string; language?: string }>,
        overrides?: Partial<Prisma.ChatCreateInput>
    ): Prisma.ChatCreateInput {
        return {
            ...this.createMinimal(overrides),
            messages: {
                create: messages.map(msg => ({
                    id: generatePK(),
                    config: {
                        __version: "1.0.0",
                        resources: [],
                    },
                    user: { connect: { id: msg.userId } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: msg.language || "en",
                            text: msg.text,
                        }],
                    },
                })),
            },
        };
    }
}

/**
 * Helper to create chat with team connection
 */
export function createTeamChat(
    teamId: string,
    overrides?: Partial<Prisma.ChatCreateInput>
): Prisma.ChatCreateInput {
    return {
        ...ChatDbFactory.createMinimal(overrides),
        team: { connect: { id: teamId } },
    };
}

/**
 * Helper to create a full chat setup for testing
 */
export async function seedTestChat(
    prisma: any,
    options: {
        userIds: string[];
        isPrivate?: boolean;
        withMessages?: boolean;
        withInvites?: boolean;
        teamId?: string;
    }
) {
    const chatData = ChatDbFactory.createWithParticipants(options.userIds, {
        isPrivate: options.isPrivate ?? false,
        ...(options.teamId && { team: { connect: { id: options.teamId } } }),
    });

    // Add messages if requested
    if (options.withMessages && options.userIds.length > 0) {
        chatData.messages = {
            create: [{
                id: generatePK(),
                config: { __version: "1.0.0", resources: [] },
                user: { connect: { id: options.userIds[0] } },
                translations: {
                    create: [{
                        id: generatePK(),
                        language: "en",
                        text: "Welcome to the chat!",
                    }],
                },
            }],
        };
    }

    // Add invites if requested
    if (options.withInvites && options.userIds.length > 0) {
        chatData.invites = {
            create: [{
                id: generatePK(),
                message: "You're invited to join our discussion",
                user: { connect: { id: options.userIds[0] } },
            }],
        };
    }

    return await prisma.chat.create({ data: chatData });
}