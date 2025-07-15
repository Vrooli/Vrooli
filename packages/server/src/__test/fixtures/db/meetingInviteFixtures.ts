import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for MeetingInvite model - used for seeding test data
 */

export class MeetingInviteDbFactory {
    static createMinimal(
        meetingId: string,
        userId: string,
        overrides?: Partial<Prisma.meeting_inviteCreateInput>,
    ): Prisma.meeting_inviteCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            status: "Pending",
            meeting: { connect: { id: meetingId } },
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createWithMessage(
        meetingId: string,
        userId: string,
        message: string,
        overrides?: Partial<Prisma.meeting_inviteCreateInput>,
    ): Prisma.meeting_inviteCreateInput {
        return this.createMinimal(meetingId, userId, {
            message,
            ...overrides,
        });
    }

    static createAccepted(
        meetingId: string,
        userId: string,
        overrides?: Partial<Prisma.meeting_inviteCreateInput>,
    ): Prisma.meeting_inviteCreateInput {
        return this.createMinimal(meetingId, userId, {
            status: "Accepted",
            ...overrides,
        });
    }

    static createDeclined(
        meetingId: string,
        userId: string,
        overrides?: Partial<Prisma.meeting_inviteCreateInput>,
    ): Prisma.meeting_inviteCreateInput {
        return this.createMinimal(meetingId, userId, {
            status: "Declined",
            ...overrides,
        });
    }
}

/**
 * Helper to seed meeting invites for testing
 */
export async function seedMeetingInvites(
    prisma: any,
    options: {
        meetingId: string;
        userIds: string[];
        status?: "Pending" | "Accepted" | "Declined";
        withCustomMessages?: boolean;
        messagePrefix?: string;
    },
) {
    const invites = [];

    for (let i = 0; i < options.userIds.length; i++) {
        const userId = options.userIds[i];
        let inviteData: Prisma.meeting_inviteCreateInput;

        if (options.status === "Accepted") {
            inviteData = MeetingInviteDbFactory.createAccepted(options.meetingId, userId);
        } else if (options.status === "Declined") {
            inviteData = MeetingInviteDbFactory.createDeclined(options.meetingId, userId);
        } else if (options.withCustomMessages) {
            const messagePrefix = options.messagePrefix || "Custom meeting invite message";
            inviteData = MeetingInviteDbFactory.createWithMessage(
                options.meetingId,
                userId,
                `${messagePrefix} for user ${i + 1}`,
            );
        } else {
            inviteData = MeetingInviteDbFactory.createMinimal(options.meetingId, userId);
        }

        const invite = await prisma.meetingInvite.create({ 
            data: inviteData,
            include: {
                meeting: true,
                user: true,
            },
        });
        invites.push(invite);
    }

    return invites;
}

/**
 * Helper to create a meeting with invites
 */
export async function createMeetingWithInvites(
    prisma: any,
    options: {
        meetingData: Prisma.MeetingCreateInput;
        invitedUserIds: string[];
        inviteStatus?: "Pending" | "Accepted" | "Declined";
        withCustomMessages?: boolean;
    },
) {
    // Create the meeting first
    const meeting = await prisma.meeting.create({
        data: options.meetingData,
        include: {
            attendees: true,
            invites: true,
        },
    });

    // Create invites for the meeting
    const invites = await seedMeetingInvites(prisma, {
        meetingId: meeting.id,
        userIds: options.invitedUserIds,
        status: options.inviteStatus,
        withCustomMessages: options.withCustomMessages,
    });

    return {
        meeting,
        invites,
    };
}
