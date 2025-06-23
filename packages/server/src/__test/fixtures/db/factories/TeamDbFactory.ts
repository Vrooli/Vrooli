import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type team, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface TeamRelationConfig extends RelationConfig {
    owner?: { userId: string | bigint };
    members?: Array<{ userId: string | bigint; isAdmin?: boolean }>;
    translations?: Array<{ language: string; name: string; bio?: string }>;
    resources?: number;
}

/**
 * Database fixture factory for Team model
 * Handles team creation with members, projects, and other relationships
 */
export class TeamDbFactory extends DatabaseFixtureFactory<
    team,
    Prisma.teamCreateInput,
    Prisma.teamInclude,
    Prisma.teamUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("team", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.team;
    }

    protected getMinimalData(overrides?: Partial<Prisma.teamCreateInput>): Prisma.teamCreateInput {
        const uniqueHandle = `team_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isOpenToNewMembers: true,
            isPrivate: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.teamCreateInput>): Prisma.teamCreateInput {
        const uniqueHandle = `complete_team_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isOpenToNewMembers: true,
            isPrivate: false,
            bannerImage: "https://example.com/team-banner.jpg",
            profileImage: "https://example.com/team-logo.jpg",
            translations: {
                create: [
                    {
                        id: generatePK(),
                        language: "en",
                        name: "Complete Test Team",
                        bio: "A comprehensive test team with all features",
                    },
                    {
                        id: generatePK(),
                        language: "es",
                        name: "Equipo de Prueba Completo",
                        bio: "Un equipo de prueba integral con todas las funcionalidades",
                    },
                ],
            },
            ...overrides,
        };
    }

    protected getDefaultInclude(): Prisma.teamInclude {
        return {
            translations: true,
            members: {
                select: {
                    id: true,
                    isAdmin: true,
                    user: {
                        select: {
                            id: true,
                            publicId: true,
                            handle: true,
                        },
                    },
                },
            },
            resources: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            _count: {
                select: {
                    members: true,
                    resources: true,
                    chats: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.teamCreateInput,
        config: TeamRelationConfig,
        tx: any,
    ): Promise<Prisma.teamCreateInput> {
        const data = { ...baseData };

        // Handle owner (first member with Owner role)
        if (config.owner) {
            data.members = {
                create: [{
                    id: generatePK(),
                    publicId: generatePublicId(),
                    userId: BigInt(config.owner.userId),
                    isAdmin: true,
                }],
            };
        }

        // Handle additional members
        if (config.members && Array.isArray(config.members)) {
            const memberCreates = config.members.map(member => ({
                id: generatePK(),
                publicId: generatePublicId(),
                userId: BigInt(member.userId),
                isAdmin: member.isAdmin || false,
            }));

            if (data.members?.create) {
                // Append to existing members (owner)
                const existingCreates = Array.isArray(data.members.create) ? data.members.create : [data.members.create];
                data.members = {
                    create: [...existingCreates, ...memberCreates],
                };
            } else {
                data.members = { create: memberCreates };
            }
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: generatePK(),
                    ...trans,
                })),
            };
        }

        return data;
    }

    /**
     * Create a team with owner and members
     */
    async createWithOwnerAndMembers(
        ownerId: string | bigint,
        memberIds: (string | bigint)[] = [],
    ): Promise<team> {
        const members = [
            { userId: ownerId, isAdmin: true },
            ...memberIds.map(userId => ({ userId, isAdmin: false })),
        ];

        return this.createWithRelations({
            members,
        });
    }

    /**
     * Create test scenarios
     */
    async createPrivateTeam(): Promise<team> {
        return this.createMinimal({
            isPrivate: true,
            isOpenToNewMembers: false,
            handle: `private_team_${nanoid(8)}`,
        });
    }

    async createOpenTeam(): Promise<team> {
        return this.createComplete({
            isPrivate: false,
            isOpenToNewMembers: true,
            handle: `open_team_${nanoid(8)}`,
        });
    }

    async createOrganizationTeam(): Promise<team> {
        return this.createWithRelations({
            overrides: {
                handle: `org_team_${nanoid(8)}`,
                isPrivate: true,
                isOpenToNewMembers: false,
            },
            translations: [
                {
                    language: "en",
                    name: "Organization Team",
                    bio: "Professional organization team structure",
                },
            ],
        });
    }

    protected async checkModelConstraints(record: team): Promise<string[]> {
        const violations: string[] = [];
        
        // Check handle uniqueness
        if (record.handle) {
            const duplicate = await this.prisma.team.findFirst({
                where: { 
                    handle: record.handle,
                    id: { not: record.id },
                },
            });
            if (duplicate) {
                violations.push("Handle must be unique");
            }
        }

        // Check handle format
        if (record.handle && !/^[a-zA-Z0-9_]+$/.test(record.handle)) {
            violations.push("Handle contains invalid characters");
        }

        // Check that team has at least one admin
        const admins = await this.prisma.member.count({
            where: {
                teamId: record.id,
                isAdmin: true,
            },
        });
        
        if (admins === 0) {
            violations.push("Team must have at least one admin");
        }

        // Check private team constraints
        if (record.isPrivate && record.isOpenToNewMembers) {
            violations.push("Private teams should not be open to new members");
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, publicId, name, handle
                isPrivate: false,
                isOpenToNewMembers: true,
            },
            invalidTypes: {
                id: "not-a-snowflake",
                publicId: 123, // Should be string
                name: null, // Should be string
                handle: true, // Should be string
                isPrivate: "yes", // Should be boolean
                isOpenToNewMembers: 1, // Should be boolean
            },
            duplicateHandle: {
                id: generatePK(),
                publicId: generatePublicId(),
                handle: "existing_team_handle", // Assumes this exists
                isPrivate: false,
                isOpenToNewMembers: true,
            },
            conflictingPrivacy: {
                id: generatePK(),
                publicId: generatePublicId(),
                handle: `conflict_${nanoid(8)}`,
                isPrivate: true,
                isOpenToNewMembers: true, // Conflict with isPrivate
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.teamCreateInput> {
        return {
            maxLengthHandle: {
                ...this.getMinimalData(),
                handle: "team_" + "a".repeat(45), // Max length handle
            },
            unicodeNameTeam: {
                ...this.getMinimalData(),
                handle: `unicode_team_${nanoid(8)}`,
            },
            largeTeam: {
                ...this.getCompleteData(),
                handle: `large_team_${nanoid(8)}`,
                // Would need to add many members after creation
            },
            multiLanguageTeam: {
                ...this.getCompleteData(),
                handle: `multilang_team_${nanoid(8)}`,
                translations: {
                    create: Array.from({ length: 5 }, (_, i) => ({
                        id: generatePK(),
                        language: ["en", "es", "fr", "de", "ja"][i],
                        name: `Team Name ${i}`,
                        bio: `Team bio in language ${i}`,
                    })),
                },
            },
        };
    }

    /**
     * Seed multiple teams with members
     */
    async seedTeamsWithMembers(
        teamCount: number,
        membersPerTeam: number,
        userIds: (string | bigint)[],
    ): Promise<team[]> {
        const teams: team[] = [];
        
        for (let i = 0; i < teamCount; i++) {
            const startIdx = (i * membersPerTeam) % userIds.length;
            const memberIds = [];
            
            for (let j = 0; j < membersPerTeam; j++) {
                const userIdx = (startIdx + j) % userIds.length;
                memberIds.push(userIds[userIdx]);
            }
            
            const team = await this.createWithOwnerAndMembers(
                memberIds[0], // First member is owner
                memberIds.slice(1), // Rest are members
            );
            
            teams.push(team);
        }
        
        return teams;
    }

    protected getCascadeInclude(): any {
        return {
            members: true,
            resources: true,
            chats: true,
            translations: true,
            // Add other relationships as needed
        };
    }

    protected async deleteRelatedRecords(
        record: team & {
            chats?: any[];
            resources?: any[];
            members?: any[];
            translations?: any[];
        },
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete chats
        if (record.chats?.length) {
            await tx.chat.deleteMany({
                where: { teamId: record.id },
            });
        }


        // Delete resources
        if (record.resources?.length) {
            await tx.resource.deleteMany({
                where: { ownedByTeamId: record.id },
            });
        }

        // Delete members
        if (record.members?.length) {
            await tx.member.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete translations
        if (record.translations?.length) {
            await tx.teamTranslation.deleteMany({
                where: { teamId: record.id },
            });
        }
    }

    /**
     * Create a team hierarchy (teams with sub-teams)
     * Note: This would require a parent-child relationship in the schema
     */
    async createTeamHierarchy(levels = 3): Promise<team[]> {
        const teams: team[] = [];
        
        // Create root team
        const rootTeam = await this.createComplete({
            handle: `root_org_${nanoid(8)}`,
        });
        teams.push(rootTeam);
        
        // Create sub-teams
        // Note: This is a placeholder - actual implementation would depend on schema
        for (let i = 0; i < levels - 1; i++) {
            const subTeam = await this.createMinimal({
                handle: `sub_team_${i}_${nanoid(8)}`,
            });
            teams.push(subTeam);
        }
        
        return teams;
    }
}

// Export factory creator function
export const createTeamDbFactory = (prisma: PrismaClient) => new TeamDbFactory(prisma);
