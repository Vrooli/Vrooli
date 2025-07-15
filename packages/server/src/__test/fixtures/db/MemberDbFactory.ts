import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { DbTestFixtures, RelationConfig } from "./types.js";
// Removed generatePK import - using this.generateId() instead

interface MemberRelationConfig extends RelationConfig {
    withUser?: boolean;
    withTeam?: boolean;
    userId?: bigint;
    teamId?: bigint;
    role?: "Owner" | "Admin" | "Member";
}

/**
 * Enhanced database fixture factory for Member model
 * Provides comprehensive testing capabilities for team members
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different member roles
 * - Team membership scenarios
 * - Permission testing configurations
 * - Predefined test scenarios
 */
export class MemberDbFactory extends EnhancedDatabaseFactory<
    member, // Using any to avoid type issues
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
            user: { connect: { id: this.generateId() } },
            team: { connect: { id: this.generateId() } },
            role: "Member",
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.memberCreateInput>): Prisma.memberCreateInput {
        return {
            ...this.generateMinimalData(),
            role: "Member",
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
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    role: "Owner",
                },
                
                admin: {
                    id: this.generateId(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    role: "Admin",
                },
                
                regularMember: {
                    id: this.generateId(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    role: "Member",
                },
            },
            
            invalid: {
                missingRequired: {
                    id: this.generateId(),
                    team: { connect: { id: teamId } },
                    role: "Member",
                    // Missing user connection
                } as any,
                
                invalidTypes: {
                    id: "not-a-bigint" as any,
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    role: "InvalidRole" as any,
                } as any,
            },
            
            edgeCase: {
                duplicateMembership: {
                    id: this.generateId(),
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    role: "Member",
                    // Should be handled by unique constraint in DB
                },
            },
            
            updates: {
                minimal: {
                    role: "Admin",
                },
                
                complete: {
                    role: "Owner",
                },
                
                promoteToAdmin: {
                    role: "Admin",
                },
                
                promoteToOwner: {
                    role: "Owner",
                },
                
                demoteToMember: {
                    role: "Member",
                },
            },
        };
    }

    /**
     * Create member with specific role
     */
    async createWithRole(
        userId: bigint, 
        teamId: bigint, 
        role: "Owner" | "Admin" | "Member",
        overrides?: Partial<Prisma.memberCreateInput>,
    ) {
        const data = {
            id: this.generateId(),
            user: { connect: { id: userId } },
            team: { connect: { id: teamId } },
            role,
            ...overrides,
        };
        
        return this.createMinimal(data);
    }

    /**
     * Create team owner
     */
    async createOwner(userId: bigint, teamId: bigint, overrides?: Partial<Prisma.memberCreateInput>) {
        return this.createWithRole(userId, teamId, "Owner", overrides);
    }

    /**
     * Create team admin
     */
    async createAdmin(userId: bigint, teamId: bigint, overrides?: Partial<Prisma.memberCreateInput>) {
        return this.createWithRole(userId, teamId, "Admin", overrides);
    }

    /**
     * Create regular team member
     */
    async createMember(userId: bigint, teamId: bigint, overrides?: Partial<Prisma.memberCreateInput>) {
        return this.createWithRole(userId, teamId, "Member", overrides);
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
                    user: { connect: { id: userId } },
                    team: { connect: { id: teamId } },
                    role: config.role || "Member",
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
        members: Array<{ userId: bigint; role?: "Owner" | "Admin" | "Member" }>,
    ) {
        return this.prisma.$transaction(async (tx) => {
            const createdMembers = [];
            
            for (const member of members) {
                const created = await tx.member.create({
                    data: {
                        id: this.generateId(),
                        user: { connect: { id: member.userId } },
                        team: { connect: { id: teamId } },
                        role: member.role || "Member",
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
     * Verify member role
     */
    async verifyRole(memberId: bigint, expectedRole: "Owner" | "Admin" | "Member") {
        const member = await this.prisma.member.findUnique({
            where: { id: memberId },
        });
        
        if (!member) {
            throw new Error(`Member ${memberId} not found`);
        }
        
        if (member.role !== expectedRole) {
            throw new Error(
                `Role mismatch: expected ${expectedRole}, got ${member.role}`,
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
