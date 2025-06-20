import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface TeamRelationConfig extends RelationConfig {
    owner?: { userId: string };
    members?: Array<{ userId: string; role: string }>;
    translations?: Array<{ language: string; name: string; description?: string }>;
    projects?: number;
    resourceLists?: number;
}

/**
 * Database fixture factory for Team model
 * Handles team creation with members, projects, and other relationships
 */
export class TeamDbFactory extends DatabaseFixtureFactory<
    Prisma.Team,
    Prisma.TeamCreateInput,
    Prisma.TeamInclude,
    Prisma.TeamUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Team', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.team;
    }

    protected getMinimalData(overrides?: Partial<Prisma.TeamCreateInput>): Prisma.TeamCreateInput {
        const uniqueHandle = `team_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            name: "Test Team",
            isOpenToNewMembers: true,
            isPrivate: false,
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.TeamCreateInput>): Prisma.TeamCreateInput {
        const uniqueHandle = `complete_team_${nanoid(8)}`;
        
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            name: "Complete Test Team",
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

    protected getDefaultInclude(): Prisma.TeamInclude {
        return {
            translations: true,
            members: {
                select: {
                    id: true,
                    role: true,
                    user: {
                        select: {
                            id: true,
                            publicId: true,
                            name: true,
                            handle: true,
                        },
                    },
                },
            },
            projects: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                },
            },
            resourceLists: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            _count: {
                select: {
                    members: true,
                    projects: true,
                    resourceLists: true,
                    chats: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.TeamCreateInput,
        config: TeamRelationConfig,
        tx: any
    ): Promise<Prisma.TeamCreateInput> {
        let data = { ...baseData };

        // Handle owner (first member with Owner role)
        if (config.owner) {
            data.members = {
                create: [{
                    id: generatePK(),
                    userId: config.owner.userId,
                    role: "Owner",
                }],
            };
        }

        // Handle additional members
        if (config.members && Array.isArray(config.members)) {
            const memberCreates = config.members.map(member => ({
                id: generatePK(),
                userId: member.userId,
                role: member.role || "Member",
            }));

            if (data.members?.create) {
                // Append to existing members (owner)
                data.members.create = [
                    ...(Array.isArray(data.members.create) ? data.members.create : [data.members.create]),
                    ...memberCreates,
                ];
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
        ownerId: string,
        memberIds: string[] = []
    ): Promise<Prisma.Team> {
        const members = [
            { userId: ownerId, role: "Owner" },
            ...memberIds.map(userId => ({ userId, role: "Member" })),
        ];

        return this.createWithRelations({
            members,
        });
    }

    /**
     * Create test scenarios
     */
    async createPrivateTeam(): Promise<Prisma.Team> {
        return this.createMinimal({
            isPrivate: true,
            isOpenToNewMembers: false,
            handle: `private_team_${nanoid(8)}`,
            name: "Private Team",
        });
    }

    async createOpenTeam(): Promise<Prisma.Team> {
        return this.createComplete({
            isPrivate: false,
            isOpenToNewMembers: true,
            handle: `open_team_${nanoid(8)}`,
            name: "Open Community Team",
        });
    }

    async createOrganizationTeam(): Promise<Prisma.Team> {
        return this.createWithRelations({
            overrides: {
                name: "Organization Team",
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

    protected async checkModelConstraints(record: Prisma.Team): Promise<string[]> {
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
                violations.push('Handle must be unique');
            }
        }

        // Check handle format
        if (record.handle && !/^[a-zA-Z0-9_]+$/.test(record.handle)) {
            violations.push('Handle contains invalid characters');
        }

        // Check that team has at least one owner
        const owners = await this.prisma.member.count({
            where: {
                teamId: record.id,
                role: "Owner",
            },
        });
        
        if (owners === 0) {
            violations.push('Team must have at least one owner');
        }

        // Check private team constraints
        if (record.isPrivate && record.isOpenToNewMembers) {
            violations.push('Private teams should not be open to new members');
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
                name: "Duplicate Handle Team",
                handle: "existing_team_handle", // Assumes this exists
                isPrivate: false,
                isOpenToNewMembers: true,
            },
            conflictingPrivacy: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Conflicting Team",
                handle: `conflict_${nanoid(8)}`,
                isPrivate: true,
                isOpenToNewMembers: true, // Conflict with isPrivate
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.TeamCreateInput> {
        return {
            maxLengthHandle: {
                ...this.getMinimalData(),
                handle: 'team_' + 'a'.repeat(45), // Max length handle
            },
            unicodeNameTeam: {
                ...this.getMinimalData(),
                name: "チーム 🎌", // Unicode characters
                handle: `unicode_team_${nanoid(8)}`,
            },
            largeTeam: {
                ...this.getCompleteData(),
                handle: `large_team_${nanoid(8)}`,
                name: "Large Test Team",
                // Would need to add many members after creation
            },
            multiLanguageTeam: {
                ...this.getCompleteData(),
                handle: `multilang_team_${nanoid(8)}`,
                translations: {
                    create: Array.from({ length: 5 }, (_, i) => ({
                        id: generatePK(),
                        language: ['en', 'es', 'fr', 'de', 'ja'][i],
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
        userIds: string[]
    ): Promise<Prisma.Team[]> {
        const teams: Prisma.Team[] = [];
        
        for (let i = 0; i < teamCount; i++) {
            const startIdx = (i * membersPerTeam) % userIds.length;
            const memberIds = [];
            
            for (let j = 0; j < membersPerTeam; j++) {
                const userIdx = (startIdx + j) % userIds.length;
                memberIds.push(userIds[userIdx]);
            }
            
            const team = await this.createWithOwnerAndMembers(
                memberIds[0], // First member is owner
                memberIds.slice(1) // Rest are members
            );
            
            teams.push(team);
        }
        
        return teams;
    }

    protected getCascadeInclude(): any {
        return {
            members: true,
            projects: true,
            resourceLists: true,
            chats: true,
            translations: true,
            // Add other relationships as needed
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Team,
        remainingDepth: number,
        tx: any
    ): Promise<void> {
        // Delete in order of dependencies
        
        // Delete chats
        if (record.chats?.length) {
            await tx.chat.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete projects (if they're team-owned)
        if (record.projects?.length) {
            // First remove team association
            await tx.project.updateMany({
                where: { teamId: record.id },
                data: { teamId: null },
            });
        }

        // Delete resource lists
        if (record.resourceLists?.length) {
            await tx.resourceList.deleteMany({
                where: { teamId: record.id },
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
    async createTeamHierarchy(levels: number = 3): Promise<Prisma.Team[]> {
        const teams: Prisma.Team[] = [];
        
        // Create root team
        const rootTeam = await this.createComplete({
            name: "Root Organization",
            handle: `root_org_${nanoid(8)}`,
        });
        teams.push(rootTeam);
        
        // Create sub-teams
        // Note: This is a placeholder - actual implementation would depend on schema
        for (let i = 0; i < levels - 1; i++) {
            const subTeam = await this.createMinimal({
                name: `Sub Team Level ${i + 1}`,
                handle: `sub_team_${i}_${nanoid(8)}`,
            });
            teams.push(subTeam);
        }
        
        return teams;
    }
}

// Export factory creator function
export const createTeamDbFactory = (prisma: PrismaClient) => new TeamDbFactory(prisma);