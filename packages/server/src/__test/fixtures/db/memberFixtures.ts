import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Member model - used for seeding test data
 */

export class MemberDbFactory {
    static createMinimal(
        userId: string,
        teamId: string,
        overrides?: Partial<Prisma.MemberCreateInput>,
    ): Prisma.MemberCreateInput {
        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            team: { connect: { id: teamId } },
            role: "Member",
            ...overrides,
        };
    }

    static createOwner(
        userId: string,
        teamId: string,
        overrides?: Partial<Prisma.MemberCreateInput>,
    ): Prisma.MemberCreateInput {
        return this.createMinimal(userId, teamId, {
            role: "Owner",
            ...overrides,
        });
    }

    static createAdmin(
        userId: string,
        teamId: string,
        overrides?: Partial<Prisma.MemberCreateInput>,
    ): Prisma.MemberCreateInput {
        return this.createMinimal(userId, teamId, {
            role: "Admin",
            ...overrides,
        });
    }

    static createWithPermissions(
        userId: string,
        teamId: string,
        permissions: string[],
        overrides?: Partial<Prisma.MemberCreateInput>,
    ): Prisma.MemberCreateInput {
        return this.createMinimal(userId, teamId, {
            permissions,
            ...overrides,
        });
    }
}

/**
 * Database fixtures for MemberInvite model
 */
export class MemberInviteDbFactory {
    static createMinimal(
        userId: string,
        teamId: string,
        createdById: string,
        overrides?: Partial<Prisma.MemberInviteCreateInput>,
    ): Prisma.MemberInviteCreateInput {
        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            team: { connect: { id: teamId } },
            createdBy: { connect: { id: createdById } },
            message: "You're invited to join our team!",
            willBeAdmin: false,
            willHavePermissions: [],
            ...overrides,
        };
    }

    static createAdminInvite(
        userId: string,
        teamId: string,
        createdById: string,
        overrides?: Partial<Prisma.MemberInviteCreateInput>,
    ): Prisma.MemberInviteCreateInput {
        return this.createMinimal(userId, teamId, createdById, {
            willBeAdmin: true,
            message: "You're invited to join as an admin!",
            ...overrides,
        });
    }

    static createWithPermissions(
        userId: string,
        teamId: string,
        createdById: string,
        permissions: string[],
        overrides?: Partial<Prisma.MemberInviteCreateInput>,
    ): Prisma.MemberInviteCreateInput {
        return this.createMinimal(userId, teamId, createdById, {
            willHavePermissions: permissions,
            ...overrides,
        });
    }
}

/**
 * Helper to seed a team with members
 */
export async function seedTeamWithMembers(
    prisma: any,
    options: {
        teamId: string;
        ownerId: string;
        memberIds?: string[];
        adminIds?: string[];
        withInvites?: Array<{ userId: string; willBeAdmin?: boolean }>;
    },
) {
    const members = [];

    // Create owner
    const owner = await prisma.member.create({
        data: MemberDbFactory.createOwner(options.ownerId, options.teamId),
    });
    members.push(owner);

    // Create regular members
    if (options.memberIds) {
        for (const memberId of options.memberIds) {
            const member = await prisma.member.create({
                data: MemberDbFactory.createMinimal(memberId, options.teamId),
            });
            members.push(member);
        }
    }

    // Create admins
    if (options.adminIds) {
        for (const adminId of options.adminIds) {
            const admin = await prisma.member.create({
                data: MemberDbFactory.createAdmin(adminId, options.teamId),
            });
            members.push(admin);
        }
    }

    // Create invites
    const invites = [];
    if (options.withInvites) {
        for (const invite of options.withInvites) {
            const inviteData = invite.willBeAdmin
                ? MemberInviteDbFactory.createAdminInvite(
                    invite.userId,
                    options.teamId,
                    options.ownerId,
                )
                : MemberInviteDbFactory.createMinimal(
                    invite.userId,
                    options.teamId,
                    options.ownerId,
                );
            
            const createdInvite = await prisma.memberInvite.create({ data: inviteData });
            invites.push(createdInvite);
        }
    }

    return { members, invites };
}
