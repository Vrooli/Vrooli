import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ChatParticipant model - used for seeding test data
 */

export class ChatParticipantDbFactory {
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            chat: { connect: { id: chatId } },
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createAdmin(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        return this.createMinimal(chatId, userId, {
            permissions: JSON.stringify({
                canDelete: true,
                canInvite: true,
                canKick: true,
                canUpdate: true,
            }),
            ...overrides,
        });
    }

    static createMember(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        return this.createMinimal(chatId, userId, {
            permissions: JSON.stringify({
                canDelete: false,
                canInvite: false,
                canKick: false,
                canUpdate: false,
            }),
            ...overrides,
        });
    }

    static createWithCustomPermissions(
        chatId: string,
        userId: string,
        permissions: Record<string, boolean>,
        overrides?: Partial<Prisma.Chat_participantsCreateInput>
    ): Prisma.Chat_participantsCreateInput {
        return this.createMinimal(chatId, userId, {
            permissions: JSON.stringify(permissions),
            ...overrides,
        });
    }
}

/**
 * Helper to seed chat participants for testing
 */
export async function seedChatParticipants(
    prisma: any,
    options: {
        chatId: string;
        participants: Array<{
            userId: string;
            role?: "admin" | "member";
            permissions?: Record<string, boolean>;
        }>;
    }
) {
    const createdParticipants = [];

    for (const participant of options.participants) {
        let participantData: Prisma.Chat_participantsCreateInput;

        if (participant.role === "admin") {
            participantData = ChatParticipantDbFactory.createAdmin(options.chatId, participant.userId);
        } else if (participant.permissions) {
            participantData = ChatParticipantDbFactory.createWithCustomPermissions(
                options.chatId,
                participant.userId,
                participant.permissions
            );
        } else {
            participantData = ChatParticipantDbFactory.createMember(options.chatId, participant.userId);
        }

        const chatParticipant = await prisma.chat_participants.create({
            data: participantData,
            include: {
                chat: true,
                user: true,
            },
        });
        createdParticipants.push(chatParticipant);
    }

    return createdParticipants;
}

/**
 * Helper to add participants to an existing chat
 */
export async function addParticipantsToChat(
    prisma: any,
    chatId: string,
    userIds: string[],
    role: "admin" | "member" = "member"
) {
    const participants = userIds.map(userId => ({
        userId,
        role,
    }));

    return seedChatParticipants(prisma, {
        chatId,
        participants,
    });
}

/**
 * Helper to get all participants for a chat
 */
export async function getChatParticipants(
    prisma: any,
    chatId: string
) {
    return prisma.chat_participants.findMany({
        where: { chatId },
        include: {
            user: true,
            chat: true,
        },
    });
}