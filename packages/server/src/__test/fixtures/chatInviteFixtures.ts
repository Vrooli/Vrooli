import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ChatInvite model - used for seeding test data
 */

export class ChatInviteDbFactory {
    static createMinimal(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.ChatInviteCreateInput>
    ): Prisma.ChatInviteCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            message: "You're invited to join this chat!",
            chat: { connect: { id: chatId } },
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createWithCustomMessage(
        chatId: string,
        userId: string,
        message: string,
        overrides?: Partial<Prisma.ChatInviteCreateInput>
    ): Prisma.ChatInviteCreateInput {
        return this.createMinimal(chatId, userId, {
            message,
            ...overrides,
        });
    }

    static createAccepted(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.ChatInviteCreateInput>
    ): Prisma.ChatInviteCreateInput {
        return this.createMinimal(chatId, userId, {
            status: "Accepted",
            ...overrides,
        });
    }

    static createDeclined(
        chatId: string,
        userId: string,
        overrides?: Partial<Prisma.ChatInviteCreateInput>
    ): Prisma.ChatInviteCreateInput {
        return this.createMinimal(chatId, userId, {
            status: "Declined",
            ...overrides,
        });
    }
}

/**
 * Helper to seed chat invites for testing
 */
export async function seedChatInvites(
    prisma: any,
    options: {
        chatId: string;
        userIds: string[];
        status?: "Pending" | "Accepted" | "Declined";
        withCustomMessages?: boolean;
    }
) {
    const invites = [];

    for (let i = 0; i < options.userIds.length; i++) {
        const userId = options.userIds[i];
        let inviteData: Prisma.ChatInviteCreateInput;

        if (options.status === "Accepted") {
            inviteData = ChatInviteDbFactory.createAccepted(options.chatId, userId);
        } else if (options.status === "Declined") {
            inviteData = ChatInviteDbFactory.createDeclined(options.chatId, userId);
        } else if (options.withCustomMessages) {
            inviteData = ChatInviteDbFactory.createWithCustomMessage(
                options.chatId,
                userId,
                `Custom invite message for user ${i + 1}`
            );
        } else {
            inviteData = ChatInviteDbFactory.createMinimal(options.chatId, userId);
        }

        const invite = await prisma.chatInvite.create({ data: inviteData });
        invites.push(invite);
    }

    return invites;
}