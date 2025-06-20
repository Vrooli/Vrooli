import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface TeamRelationConfig extends RelationConfig {
    owner?: { userId: string };
    members?: Array<{ userId: string; role: string }>;
    translations?: Array<{ language: string; name: string; bio?: string }>;
    projects?: number;
    resourceLists?: number;
}

/**
 * Enhanced database fixture factory for Team model
 * Provides comprehensive testing capabilities for teams with various configurations
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Owner and member management
 * - Project and resource associations
 * - Translation support
 * - Predefined test scenarios
 * - Privacy and access control testing
 */
export class TeamDbFactory extends EnhancedDatabaseFactory<
    Prisma.Team,
    Prisma.TeamCreateInput,
    Prisma.TeamInclude,
    Prisma.TeamUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Team', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.team;
    }

    /**
     * Get complete test fixtures for Team model
     */
    protected getFixtures(): DbTestFixtures<Prisma.TeamCreateInput, Prisma.TeamUpdateInput> {
        return {
            minimal: {
                id: generatePK(),
                publicId: generatePublicId(),
                handle: `team_${nanoid(8)}`,
                name: "Test Team",
                isOpenToNewMembers: true,
                isPrivate: false,
            },
            complete: {
                id: generatePK(),
                publicId: generatePublicId(),
                handle: `complete_team_${nanoid(8)}`,
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
            },
            invalid: {
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
                teamWithoutOwner: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "No Owner Team",
                    handle: `no_owner_${nanoid(8)}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    // No members with Owner role
                },
            },
            edgeCases: {
                maxLengthHandle: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Max Length Handle Team",
                    handle: 'team_' + 'a'.repeat(45), // Max length handle
                    isPrivate: false,
                    isOpenToNewMembers: true,
                },
                unicodeNameTeam: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "チーム 🎌", // Unicode characters
                    handle: `unicode_team_${nanoid(8)}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                },
                largeTeam: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Large Test Team",
                    handle: `large_team_${nanoid(8)}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    // Would need to add many members after creation
                },
                privateClosedTeam: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Private Closed Team",
                    handle: `private_closed_${nanoid(8)}`,
                    isPrivate: true,
                    isOpenToNewMembers: false,
                },
                multiLanguageTeam: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Multilingual Team",
                    handle: `multilang_team_${nanoid(8)}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    translations: {
                        create: Array.from({ length: 5 }, (_, i) => ({
                            id: generatePK(),
                            language: ['en', 'es', 'fr', 'de', 'ja'][i],
                            name: `Team Name ${i}`,
                            bio: `Team bio in language ${i}`,
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    name: "Updated Team Name",
                },
                complete: {
                    name: "Completely Updated Team",
                    isOpenToNewMembers: false,
                    bannerImage: "https://example.com/new-banner.jpg",
                    profileImage: "https://example.com/new-logo.jpg",
                    translations: {
                        update: [{
                            where: { id: "translation_id" },
                            data: { bio: "Updated team bio" },
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.TeamCreateInput>): Prisma.TeamCreateInput {
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

    protected generateCompleteData(overrides?: Partial<Prisma.TeamCreateInput>): Prisma.TeamCreateInput {
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

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            smallTeam: {
                name: "smallTeam",
                description: "Small team with just an owner",
                config: {
                    overrides: {
                        name: "Small Team",
                    },
                    owner: { userId: "owner-user-id" },
                },
            },
            standardTeam: {
                name: "standardTeam",
                description: "Standard team with owner and members",
                config: {
                    overrides: {
                        name: "Standard Team",
                    },
                    owner: { userId: "owner-user-id" },
                    members: [
                        { userId: "member-1-id", role: "Admin" },
                        { userId: "member-2-id", role: "Member" },
                        { userId: "member-3-id", role: "Member" },
                    ],
                },
            },
            organizationTeam: {
                name: "organizationTeam",
                description: "Professional organization team",
                config: {
                    overrides: {
                        name: "Organization Team",
                        isPrivate: true,
                        isOpenToNewMembers: false,
                    },
                    owner: { userId: "org-owner-id" },
                    translations: [
                        {
                            language: "en",
                            name: "Organization Team",
                            bio: "Professional organization team structure",
                        },
                    ],
                },
            },
            openCommunity: {
                name: "openCommunity",
                description: "Open community team accepting new members",
                config: {
                    overrides: {
                        name: "Open Community",
                        isPrivate: false,
                        isOpenToNewMembers: true,
                    },
                    translations: [
                        {
                            language: "en",
                            name: "Open Community",
                            bio: "An open community for everyone to join",
                        },
                    ],
                },
            },
            enterpriseTeam: {
                name: "enterpriseTeam",
                description: "Enterprise team with complex structure",
                config: {
                    overrides: {
                        name: "Enterprise Team",
                        isPrivate: true,
                        isOpenToNewMembers: false,
                    },
                    owner: { userId: "enterprise-owner-id" },
                    members: [
                        { userId: "admin-1-id", role: "Admin" },
                        { userId: "admin-2-id", role: "Admin" },
                        { userId: "member-1-id", role: "Member" },
                        { userId: "member-2-id", role: "Member" },
                        { userId: "member-3-id", role: "Member" },
                    ],
                    projects: 5,
                    resourceLists: 3,
                },
            },
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

        // Note: Projects and resource lists would typically be created separately
        // and then associated with the team

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

        return await this.createWithRelations({
            members,
        });
    }

    /**
     * Create specific team types
     */
    async createPrivateTeam(): Promise<Prisma.Team> {
        return await this.createMinimal({
            isPrivate: true,
            isOpenToNewMembers: false,
            handle: `private_team_${nanoid(8)}`,
            name: "Private Team",
        });
    }

    async createOpenCommunity(): Promise<Prisma.Team> {
        return await this.seedScenario('openCommunity');
    }

    async createOrganizationTeam(): Promise<Prisma.Team> {
        return await this.seedScenario('organizationTeam');
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

    protected getCascadeInclude(): any {
        return {
            members: true,
            projects: true,
            resourceLists: true,
            chats: true,
            translations: true,
            bookmarkLists: true,
            schedules: true,
            meetings: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.Team,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);
        
        // Delete in order of dependencies
        
        // Delete chats
        if (shouldDelete('chats') && record.chats?.length) {
            await tx.chat.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete meetings
        if (shouldDelete('meetings') && record.meetings?.length) {
            await tx.meeting.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete schedules
        if (shouldDelete('schedules') && record.schedules?.length) {
            await tx.schedule.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete bookmark lists
        if (shouldDelete('bookmarkLists') && record.bookmarkLists?.length) {
            await tx.bookmarkList.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete projects (if they're team-owned)
        if (shouldDelete('projects') && record.projects?.length) {
            // First remove team association
            await tx.project.updateMany({
                where: { teamId: record.id },
                data: { teamId: null },
            });
        }

        // Delete resource lists
        if (shouldDelete('resourceLists') && record.resourceLists?.length) {
            await tx.resourceList.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete members
        if (shouldDelete('members') && record.members?.length) {
            await tx.member.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete('translations') && record.translations?.length) {
            await tx.teamTranslation.deleteMany({
                where: { teamId: record.id },
            });
        }
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

    /**
     * Create teams for different testing scenarios
     */
    async createTestingScenarios(): Promise<{
        publicTeam: Prisma.Team;
        privateTeam: Prisma.Team;
        organizationTeam: Prisma.Team;
        communityTeam: Prisma.Team;
    }> {
        const [publicTeam, privateTeam, organizationTeam, communityTeam] = await Promise.all([
            this.createMinimal({ name: "Public Team", isPrivate: false }),
            this.createPrivateTeam(),
            this.createOrganizationTeam(),
            this.createOpenCommunity(),
        ]);

        return {
            publicTeam,
            privateTeam,
            organizationTeam: organizationTeam as unknown as Prisma.Team,
            communityTeam: communityTeam as unknown as Prisma.Team,
        };
    }
}

// Export factory creator function
export const createTeamDbFactory = (prisma: PrismaClient) => 
    TeamDbFactory.getInstance('Team', prisma);

// Export the class for type usage
export { TeamDbFactory as TeamDbFactoryClass };