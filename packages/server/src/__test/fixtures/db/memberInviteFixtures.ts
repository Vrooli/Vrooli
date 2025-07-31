import { type Prisma } from "@prisma/client";
import { generatePK, generatePublicId } from "@vrooli/shared";

/**
 * Database fixtures for MemberInvite model - used for seeding test data
 */

export class MemberInviteDbFactory {
    static createMinimal(
        teamId: bigint,
        userId: bigint,
        overrides?: Partial<Prisma.member_inviteCreateInput>,
    ): Prisma.member_inviteCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            status: "Pending",
            team: { connect: { id: teamId } },
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createWithMessage(
        teamId: bigint,
        userId: bigint,
        message: string,
        overrides?: Partial<Prisma.member_inviteCreateInput>,
    ): Prisma.member_inviteCreateInput {
        return this.createMinimal(teamId, userId, {
            message,
            ...overrides,
        });
    }

    static createAccepted(
        teamId: bigint,
        userId: bigint,
        overrides?: Partial<Prisma.member_inviteCreateInput>,
    ): Prisma.member_inviteCreateInput {
        return this.createMinimal(teamId, userId, {
            status: "Accepted",
            ...overrides,
        });
    }

    static createDeclined(
        teamId: bigint,
        userId: bigint,
        overrides?: Partial<Prisma.member_inviteCreateInput>,
    ): Prisma.member_inviteCreateInput {
        return this.createMinimal(teamId, userId, {
            status: "Declined",
            ...overrides,
        });
    }
}

/**
 * Helper to seed member invites for testing
 */
export async function seedMemberInvites(
    prisma: any,
    options: {
        teamId: bigint;
        userIds: bigint[];
        status?: "Pending" | "Accepted" | "Declined";
        withCustomMessages?: boolean;
        messagePrefix?: string;
    },
) {
    const invites = [];

    for (let i = 0; i < options.userIds.length; i++) {
        const userId = options.userIds[i];
        let inviteData: Prisma.member_inviteCreateInput;

        if (options.status === "Accepted") {
            inviteData = MemberInviteDbFactory.createAccepted(options.teamId, userId);
        } else if (options.status === "Declined") {
            inviteData = MemberInviteDbFactory.createDeclined(options.teamId, userId);
        } else if (options.withCustomMessages) {
            const messagePrefix = options.messagePrefix || "Custom team invite message";
            inviteData = MemberInviteDbFactory.createWithMessage(
                options.teamId,
                userId,
                `${messagePrefix} for user ${i + 1}`,
            );
        } else {
            inviteData = MemberInviteDbFactory.createMinimal(options.teamId, userId);
        }

        const invite = await prisma.memberInvite.create({
            data: inviteData,
            include: {
                team: true,
                user: true,
            },
        });
        invites.push(invite);
    }

    return invites;
}

/**
 * Helper to create a team with member invites
 */
export async function createTeamWithInvites(
    prisma: any,
    options: {
        teamData: Prisma.TeamCreateInput;
        invitedUserIds: bigint[];
        inviteStatus?: "Pending" | "Accepted" | "Declined";
        withCustomMessages?: boolean;
    },
) {
    // Create the team first
    const team = await prisma.team.create({
        data: options.teamData,
        include: {
            members: true,
            invites: true,
        },
    });

    // Create invites for the team
    const invites = await seedMemberInvites(prisma, {
        teamId: team.id,
        userIds: options.invitedUserIds,
        status: options.inviteStatus,
        withCustomMessages: options.withCustomMessages,
    });

    return {
        team,
        invites,
    };
}
