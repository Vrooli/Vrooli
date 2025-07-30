import { type member, type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { DbTestFixtures, RelationConfig } from "./types.js";
// Removed generatePK import - using this.generateId() instead

interface MemberRelationConfig extends RelationConfig {
    withUser?: boolean;
    withTeam?: boolean;
    userId?: bigint;
    teamId?: bigint;
    isAdmin?: boolean;
    permissions?: object;
}

/**
 * Enhanced database fixture factory for Member model
 * Provides comprehensive testing capabilities for team members
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for admin status and permissions-based authorization
 * - Team membership scenarios
 * - Permission testing configurations
 * - Predefined test scenarios
 */
export class MemberDbFactory extends EnhancedDatabaseFactory<
    member,
    Prisma.memberCreateInput,
    Prisma.memberInclude,
    Prisma.memberUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("member", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.member;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.memberCreateInput>): Prisma.memberCreateInput {
        return {
            id: this.generateId(),
            publicId: this.generateId().toString(),
            user: { connect: { id: this.generateId() } },
            team: { connect: { id: this.generateId() } },
            isAdmin: false,
            permissions: {},
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.memberCreateInput>): Prisma.memberCreateInput {
        return {
            ...this.generateMinimalData(),
            ...overrides,
        };
    }

    /**
     * Get complete test fixtures for Member model
     */
    protected getFixtures(): DbTestFixtures<Prisma.memberCreateInput, Prisma.memberUpdateInput> {
        const userId = this.generateId();
        const teamId = this.generateId();

        return {
            minimal: this.generateMinimalData(),

            complete: this.generateCompleteData(),

            edgeCases: {
                owner: {
                    id: this.generateId(),
                    publicId: this.generateId().toString(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    isAdmin: true,
                    permissions: { manageTeam: true, manageMembers: true },
                },

                admin: {
                    id: this.generateId(),
                    publicId: this.generateId().toString(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    isAdmin: true,
                    permissions: { manageMembers: true },
                },

                regularMember: {
                    id: this.generateId(),
                    publicId: this.generateId().toString(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    isAdmin: false,
                    permissions: {},
                },
            },

            invalid: {
                missingRequired: {
                    id: this.generateId(),
                    publicId: this.generateId().toString(),
                    team: { connect: { id: teamId } },
                    isAdmin: false,
                    permissions: {},
                    // Missing user connection
                } as any,

                invalidTypes: {
                    id: "not-a-bigint" as any,
                    publicId: this.generateId().toString(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    isAdmin: "not-boolean" as any,
                    permissions: "not-object" as any,
                } as any,
            },


            updates: {
                minimal: {
                    isAdmin: true,
                    permissions: { manageMembers: true },
                },

                complete: {
                    isAdmin: true,
                    permissions: { manageTeam: true, manageMembers: true },
                },

                promoteToAdmin: {
                    isAdmin: true,
                    permissions: { manageMembers: true },
                },

                promoteToOwner: {
                    isAdmin: true,
                    permissions: { manageTeam: true, manageMembers: true },
                },

                demoteToMember: {
                    isAdmin: false,
                    permissions: {},
                },
            },
        };
    }

    /**
     * Create member with specific admin status and permissions
     */
    async createWithAdminStatus(
        userId: bigint,
        teamId: bigint,
        isAdmin: boolean,
        permissions: object = {},
        overrides?: Partial<Prisma.memberCreateInput>,
    ) {
        const data = {
            id: this.generateId(),
            publicId: this.generateId().toString(),
            user: { connect: { id: userId } },
            team: { connect: { id: teamId } },
            isAdmin,
            permissions,
            ...overrides,
        };

        return this.createMinimal(data);
    }

    /**
     * Create team owner
     */
    async createOwner(userId: bigint, teamId: bigint, overrides?: Partial<Prisma.memberCreateInput>) {
        return this.createWithAdminStatus(userId, teamId, true, { manageTeam: true, manageMembers: true }, overrides);
    }

    /**
     * Create team admin
     */
    async createAdmin(userId: bigint, teamId: bigint, overrides?: Partial<Prisma.memberCreateInput>) {
        return this.createWithAdminStatus(userId, teamId, true, { manageMembers: true }, overrides);
    }

    /**
     * Create regular team member
     */
    async createMember(userId: bigint, teamId: bigint, overrides?: Partial<Prisma.memberCreateInput>) {
        return this.createWithAdminStatus(userId, teamId, false, {}, overrides);
    }

    /**
     * Create member with relationships
     */
    async createWithRelations(config: MemberRelationConfig) {
        return this.prisma.$transaction(async (tx) => {
            let userId = config.userId;
            let teamId = config.teamId;

            // Create user if needed
            if (!userId && config.withUser) {
                const user = await tx.user.create({
                    data: {
                        id: this.generateId(),
                        publicId: this.generateId().toString(),
                        name: "Test User",
                        handle: `test_user_${this.generateId().toString().slice(-6)}`,
                        status: "Unlocked",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                    },
                });
                userId = user.id;
            }

            // Create team if needed
            if (!teamId && config.withTeam) {
                const team = await tx.team.create({
                    data: {
                        id: this.generateId(),
                        publicId: this.generateId().toString(),
                        name: "Test Team",
                        handle: `test_team_${this.generateId().toString().slice(-6)}`,
                        isPrivate: false,
                        createdBy: { connect: { id: userId || this.generateId() } },
                    },
                });
                teamId = team.id;
            }

            if (!userId || !teamId) {
                throw new Error("Both userId and teamId must be provided or generated");
            }

            return tx.member.create({
                data: {
                    id: this.generateId(),
                    publicId: this.generateId().toString(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    isAdmin: config.isAdmin || false,
                    permissions: config.permissions || {},
                },
                include: {
                    user: true,
                    team: true,
                },
            });
        });
    }

    /**
     * Create multiple members for a team
     */
    async createTeamMembers(
        teamId: bigint,
        members: Array<{ userId: bigint; isAdmin?: boolean; permissions?: object }>,
    ) {
        return this.prisma.$transaction(async (tx) => {
            const createdMembers = [];

            for (const member of members) {
                const created = await tx.member.create({
                    data: {
                        id: this.generateId(),
                        publicId: this.generateId().toString(),
                        user: { connect: { id: member.userId } },
                        team: { connect: { id: teamId } },
                        isAdmin: member.isAdmin || false,
                        permissions: member.permissions || {},
                    },
                    include: {
                        user: true,
                        team: true,
                    },
                });
                createdMembers.push(created);
            }

            return createdMembers;
        });
    }

    /**
     * Verify member admin status
     */
    async verifyAdminStatus(memberId: bigint, expectedIsAdmin: boolean) {
        const member = await this.prisma.member.findUnique({
            where: { id: memberId },
        });

        if (!member) {
            throw new Error(`Member ${memberId} not found`);
        }

        if (member.isAdmin !== expectedIsAdmin) {
            throw new Error(
                `Admin status mismatch: expected ${expectedIsAdmin}, got ${member.isAdmin}`,
            );
        }

        return member;
    }

    /**
     * Verify team membership count
     */
    async verifyTeamMemberCount(teamId: bigint, expectedCount: number) {
        const count = await this.prisma.member.count({
            where: { teamId },
        });

        if (count !== expectedCount) {
            throw new Error(
                `Member count mismatch for team ${teamId}: expected ${expectedCount}, got ${count}`,
            );
        }

        return count;
    }
}

/**
 * Factory function to create MemberDbFactory instance
 */
export function createMemberDbFactory(prisma: PrismaClient): MemberDbFactory {
    return new MemberDbFactory(prisma);
}
