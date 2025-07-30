import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";

/**
 * Database fixtures for Member model - used for seeding test data
 */

// Cached IDs for consistent testing - lazy initialization pattern
let _memberDbIds: Record<string, string> | null = null;
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
        userId: string,
        teamId: string,
        overrides?: Partial<Prisma.memberCreateInput>,
    ): Prisma.memberCreateInput {
        // Generate ID only when method is called, not at module load
        const id = overrides?.id || generatePK();
        return {
            id,
            publicId: generatePK(),
            user: { connect: { id: userId } },
            team: { connect: { id: teamId } },
            isAdmin: false,
            permissions: {},
            ...overrides,
        };
    }

    static createOwner(
        userId: string,
        teamId: string,
        overrides?: Partial<Prisma.memberCreateInput>,
    ): Prisma.memberCreateInput {
        return this.createMinimal(userId, teamId, {
            isAdmin: true,
            permissions: { manageTeam: true, manageMembers: true },
            ...overrides,
        });
    }

    static createAdmin(
        userId: string,
        teamId: string,
        overrides?: Partial<Prisma.memberCreateInput>,
    ): Prisma.memberCreateInput {
        return this.createMinimal(userId, teamId, {
            isAdmin: true,
            permissions: { manageMembers: true },
            ...overrides,
        });
    }

    static createWithPermissions(
        userId: string,
        teamId: string,
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
        userId: string,
        teamId: string,
        createdById: string,
        overrides?: Partial<Prisma.MemberInviteCreateInput>,
    ): Prisma.MemberInviteCreateInput {
        // Generate ID only when method is called, not at module load
        const id = overrides?.id || generatePK();
        return {
            id,
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
 * Predefined member fixtures for testing - lazy initialization
 */
const _minimalMemberDb: Prisma.memberCreateInput | null = null;
export function getMinimalMemberDb(userId: string, teamId: string): Prisma.memberCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberDbFactory.createMinimal(userId, teamId);
}

const _ownerMemberDb: Prisma.memberCreateInput | null = null;
export function getOwnerMemberDb(userId: string, teamId: string): Prisma.memberCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberDbFactory.createOwner(userId, teamId);
}

const _adminMemberDb: Prisma.memberCreateInput | null = null;
export function getAdminMemberDb(userId: string, teamId: string): Prisma.memberCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberDbFactory.createAdmin(userId, teamId);
}

/**
 * Predefined member invite fixtures for testing - lazy initialization
 */
const _minimalMemberInviteDb: Prisma.MemberInviteCreateInput | null = null;
export function getMinimalMemberInviteDb(userId: string, teamId: string, createdById: string): Prisma.MemberInviteCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberInviteDbFactory.createMinimal(userId, teamId, createdById);
}

const _adminMemberInviteDb: Prisma.MemberInviteCreateInput | null = null;
export function getAdminMemberInviteDb(userId: string, teamId: string, createdById: string): Prisma.MemberInviteCreateInput {
    // Create new instance each time to avoid shared state issues
    return MemberInviteDbFactory.createAdminInvite(userId, teamId, createdById);
}

/**
 * Helper to seed a team with members
 */
export async function seedTeamWithMembers(
    prisma: PrismaClient,
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

// Note: Unlike teamFixtures.ts, we don't export memberDbIds directly to avoid calling generatePK() at module load
// If backward compatibility is needed, use getMemberDbIds() function instead
