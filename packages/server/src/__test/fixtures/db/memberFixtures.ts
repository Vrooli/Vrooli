import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK, generatePublicId } from "@vrooli/shared";

/**
 * Database fixtures for Member model - used for seeding test data
 */

// Cached IDs for consistent testing - lazy initialization pattern
let _memberDbIds: Record<string, bigint> | null = null;
export function getMemberDbIds() {
    if (!_memberDbIds) {
        _memberDbIds = {
            member1: generatePK(),
            member2: generatePK(),
            member3: generatePK(),
            member4: generatePK(),
            invite1: generatePK(),
            invite2: generatePK(),
            invite3: generatePK(),
            invite4: generatePK(),
        };
    }
    return _memberDbIds;
}

export class MemberDbFactory {
    static createMinimal(
        userId: string | bigint,
        teamId: string | bigint,
        overrides?: Partial<Prisma.memberCreateInput>,
    ): Prisma.memberCreateInput {
        // Generate ID only when method is called, not at module load
        const id = overrides?.id || generatePK();
        const userIdBigint = typeof userId === "string" ? BigInt(userId) : userId;
        const teamIdBigint = typeof teamId === "string" ? BigInt(teamId) : teamId;
        return {
            id,
            publicId: generatePublicId(),
            user: { connect: { id: userIdBigint } },
            team: { connect: { id: teamIdBigint } },
            isAdmin: false,
            permissions: {},
            ...overrides,
        };
    }

    static createOwner(
        userId: string | bigint,
        teamId: string | bigint,
        overrides?: Partial<Prisma.memberCreateInput>,
    ): Prisma.memberCreateInput {
        return this.createMinimal(userId, teamId, {
            isAdmin: true,
            permissions: { manageTeam: true, manageMembers: true },
            ...overrides,
        });
    }

    static createAdmin(
        userId: string | bigint,
        teamId: string | bigint,
        overrides?: Partial<Prisma.memberCreateInput>,
    ): Prisma.memberCreateInput {
        return this.createMinimal(userId, teamId, {
            isAdmin: true,
            permissions: { manageMembers: true },
            ...overrides,
        });
    }

    static createWithPermissions(
        userId: string | bigint,
        teamId: string | bigint,
        permissions: string[],
        overrides?: Partial<Prisma.memberCreateInput>,
    ): Prisma.memberCreateInput {
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
        userId: string | bigint,
        teamId: string | bigint,
        overrides?: Partial<Prisma.member_inviteCreateInput>,
    ): Prisma.member_inviteCreateInput {
        // Generate ID only when method is called, not at module load
        const id = overrides?.id || generatePK();
        const userIdBigint = typeof userId === "string" ? BigInt(userId) : userId;
        const teamIdBigint = typeof teamId === "string" ? BigInt(teamId) : teamId;
        return {
            id,
            user: { connect: { id: userIdBigint } },
            team: { connect: { id: teamIdBigint } },
            message: "You're invited to join our team!",
            willBeAdmin: false,
            willHavePermissions: [],
            ...overrides,
        };
    }

    static createAdminInvite(
        userId: string | bigint,
        teamId: string | bigint,
        overrides?: Partial<Prisma.member_inviteCreateInput>,
    ): Prisma.member_inviteCreateInput {
        return this.createMinimal(userId, teamId, {
            willBeAdmin: true,
            message: "You're invited to join as an admin!",
            ...overrides,
        });
    }

    static createWithPermissions(
        userId: string | bigint,
        teamId: string | bigint,
        permissions: string[],
        overrides?: Partial<Prisma.member_inviteCreateInput>,
    ): Prisma.member_inviteCreateInput {
        return this.createMinimal(userId, teamId, {
            willHavePermissions: permissions,
            ...overrides,
        });
    }
}

/**
 * Predefined member fixtures for testing - lazy initialization
 */
const _minimalMemberDb: Prisma.memberCreateInput | null = null;
export function getMinimalMemberDb(userId: string | bigint, teamId: string | bigint): Prisma.memberCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberDbFactory.createMinimal(userId, teamId);
}

const _ownerMemberDb: Prisma.memberCreateInput | null = null;
export function getOwnerMemberDb(userId: string | bigint, teamId: string | bigint): Prisma.memberCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberDbFactory.createOwner(userId, teamId);
}

const _adminMemberDb: Prisma.memberCreateInput | null = null;
export function getAdminMemberDb(userId: string | bigint, teamId: string | bigint): Prisma.memberCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberDbFactory.createAdmin(userId, teamId);
}

/**
 * Predefined member invite fixtures for testing - lazy initialization
 */
const _minimalMemberInviteDb: Prisma.member_inviteCreateInput | null = null;
export function getMinimalMemberInviteDb(userId: string | bigint, teamId: string | bigint): Prisma.member_inviteCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberInviteDbFactory.createMinimal(userId, teamId);
}

const _adminMemberInviteDb: Prisma.member_inviteCreateInput | null = null;
export function getAdminMemberInviteDb(userId: string | bigint, teamId: string | bigint): Prisma.member_inviteCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberInviteDbFactory.createAdminInvite(userId, teamId);
}

/**
 * Helper to seed a team with members
 */
export async function seedTeamWithMembers(
    prisma: PrismaClient,
    options: {
        teamId: string | bigint;
        ownerId: string | bigint;
        memberIds?: (string | bigint)[];
        adminIds?: (string | bigint)[];
        withInvites?: Array<{ userId: string | bigint; willBeAdmin?: boolean }>;
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
                )
                : MemberInviteDbFactory.createMinimal(
                    invite.userId,
                    options.teamId,
                );

            const createdInvite = await prisma.member_invite.create({ data: inviteData });
            invites.push(createdInvite);
        }
    }

    return { members, invites };
}

// Note: Unlike teamFixtures.ts, we don't export memberDbIds directly to avoid calling generatePK() at module load
// If backward compatibility is needed, use getMemberDbIds() function instead
