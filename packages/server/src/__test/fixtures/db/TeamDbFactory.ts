/* eslint-disable no-magic-numbers */
import { type Prisma, type PrismaClient, type team } from "@prisma/client";
import { generatePublicId, nanoid } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface TeamRelationConfig extends RelationConfig {
    owner?: { userId: bigint };
    members?: Array<{ userId: bigint; role: string }>;
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
    team,
    Prisma.teamCreateInput,
    Prisma.teamInclude,
    Prisma.teamUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("Team", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.team;
    }

    /**
     * Get complete test fixtures for Team model
     */
    protected getFixtures(): DbTestFixtures<Prisma.teamCreateInput, Prisma.teamUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                publicId: generatePublicId(),
                handle: `team_${nanoid()}`,
                isOpenToNewMembers: true,
                isPrivate: false,
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            name: "Test Team",
                            bio: "A test team for testing purposes",
                        },
                    ],
                },
            },
            complete: {
                id: this.generateId(),
                publicId: generatePublicId(),
                handle: `complete_team_${nanoid()}`,
                isOpenToNewMembers: true,
                isPrivate: false,
                bannerImage: "https://example.com/team-banner.jpg",
                profileImage: "https://example.com/team-logo.jpg",
                translations: {
                    create: [
                        {
                            id: this.generateId(),
                            language: "en",
                            name: "Complete Test Team",
                            bio: "A comprehensive test team with all features",
                        },
                        {
                            id: this.generateId(),
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
                    id: BigInt(-1), // Invalid bigint (negative)
                    publicId: 123 as any, // Should be string - intentionally wrong for testing
                    handle: true as any, // Should be string - intentionally wrong for testing
                    isPrivate: "yes" as any, // Should be boolean - intentionally wrong for testing
                    isOpenToNewMembers: 1 as any, // Should be boolean - intentionally wrong for testing
                },
                duplicateHandle: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: "existing_team_handle", // Assumes this exists
                    isPrivate: false,
                    isOpenToNewMembers: true,
                },
                conflictingPrivacy: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `conflict_${nanoid()}`,
                    isPrivate: true,
                    isOpenToNewMembers: true, // Conflict with isPrivate
                },
                teamWithoutOwner: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `no_owner_${nanoid()}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    // No members with Owner role
                },
            },
            edgeCases: {
                maxLengthHandle: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: "team_" + "a".repeat(45), // Max length handle
                    isPrivate: false,
                    isOpenToNewMembers: true,
                },
                unicodeNameTeam: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `unicode_team_${nanoid()}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                },
                largeTeam: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `large_team_${nanoid()}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    // Would need to add many members after creation
                },
                privateClosedTeam: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `private_closed_${nanoid()}`,
                    isPrivate: true,
                    isOpenToNewMembers: false,
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            name: "Private Closed Team",
                        }],
                    },
                },
                multiLanguageTeam: {
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    handle: `multilang_team_${nanoid()}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    translations: {
                        create: Array.from({ length: 5 }, (_, i) => ({
                            id: this.generateId(),
                            language: ["en", "es", "fr", "de", "ja"][i],
                            name: `Team Name ${i}`,
                            bio: `Team bio in language ${i}`,
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    isOpenToNewMembers: false,
                },
                complete: {
                    isOpenToNewMembers: false,
                    bannerImage: "https://example.com/new-banner.jpg",
                    profileImage: "https://example.com/new-logo.jpg",
                    // Note: For fixture updates, we'll create new translations instead
                    // since we don't have existing translation IDs to reference
                    translations: {
                        create: [{
                            id: this.generateId(),
                            language: "en",
                            name: "Updated Team",
                            bio: "Updated team bio",
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.teamCreateInput>): Prisma.teamCreateInput {
        const uniqueHandle = `team_${nanoid()}`;

        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isOpenToNewMembers: true,
            isPrivate: false,
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    name: "Test Team",
                }],
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.teamCreateInput>): Prisma.teamCreateInput {
        const uniqueHandle = `complete_team_${nanoid()}`;

        return {
            id: this.generateId(),
            publicId: generatePublicId(),
            handle: uniqueHandle,
            isOpenToNewMembers: true,
            isPrivate: false,
            bannerImage: "https://example.com/team-banner.jpg",
            profileImage: "https://example.com/team-logo.jpg",
            translations: {
                create: [
                    {
                        id: this.generateId(),
                        language: "en",
                        name: "Complete Test Team",
                        bio: "A comprehensive test team with all features",
                    },
                    {
                        id: this.generateId(),
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
                    // Note: owner ID will be generated when scenario is used
                },
            },
            standardTeam: {
                name: "standardTeam",
                description: "Standard team with owner and members",
                config: {
                    overrides: {
                        name: "Standard Team",
                    },
                    // Note: owner ID will be generated when scenario is used
                    // Note: member IDs will be generated when scenario is used
                    memberCount: 3,
                    memberRoles: ["Admin", "Member", "Member"],
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
                    // Note: owner ID will be generated when scenario is used
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
                    // Note: owner ID will be generated when scenario is used
                    // Note: member IDs will be generated when scenario is used
                    memberCount: 5,
                    memberRoles: ["Admin", "Admin", "Member", "Member", "Member"],
                    projects: 5,
                    resourceLists: 3,
                },
            },
        };
    }

    /**
     * Seed a specific test scenario with fresh IDs
     */
    async seedScenario(scenarioName: string): Promise<team> {
        const scenario = this.scenarios[scenarioName];
        if (!scenario) {
            throw new Error(`Scenario ${scenarioName} not found`);
        }

        // Generate fresh IDs for the scenario
        const config = { ...scenario.config };

        // Generate owner ID if needed
        if (scenario.config.owner) {
            config.owner = { userId: this.generateId() };
        }

        // Generate member IDs if needed  
        if (scenario.config.memberCount && scenario.config.memberRoles) {
            config.members = scenario.config.memberRoles.map(role => ({
                userId: this.generateId(),
                role,
            }));
        }

        const data = this.generateCompleteData(scenario.config.overrides);
        const finalData = await this.applyRelationships(data, config, this.prisma);

        return await this.prisma.team.create({
            data: finalData,
            include: this.getDefaultInclude(),
        });
    }

    protected getDefaultInclude(): Prisma.teamInclude {
        return {
            translations: true,
            members: {
                select: {
                    id: true,
                    isAdmin: true,
                    permissions: true,
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
            issues: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            meetings: {
                select: {
                    id: true,
                    publicId: true,
                },
            },
            _count: {
                select: {
                    members: true,
                    meetings: true,
                    chats: true,
                    issues: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.teamCreateInput,
        config: TeamRelationConfig,
        tx: PrismaClient,
    ): Promise<Prisma.teamCreateInput> {
        const data = { ...baseData };

        // Handle owner (first member with Owner role)
        if (config.owner) {
            data.members = {
                create: [{
                    id: this.generateId(),
                    publicId: generatePublicId(),
                    user: { connect: { id: config.owner.userId } },
                    isAdmin: true,
                    permissions: { manageTeam: true, manageMembers: true },
                }],
            };
        }

        // Handle additional members
        if (config.members && Array.isArray(config.members)) {
            const memberCreates = config.members.map(member => ({
                id: this.generateId(),
                publicId: generatePublicId(),
                user: { connect: { id: member.userId } },
                isAdmin: member.role === "Admin" || member.role === "Owner",
                permissions: member.role === "Owner" ? { manageTeam: true, manageMembers: true } :
                    member.role === "Admin" ? { manageMembers: true } : {},
            }));

            if (data.members?.create) {
                // Append to existing members (owner)
                const existingMembers = Array.isArray(data.members.create) ? data.members.create : [data.members.create];
                data.members = {
                    create: [...existingMembers, ...memberCreates] as Prisma.memberCreateWithoutTeamInput[],
                };
            } else {
                data.members = { create: memberCreates as Prisma.memberCreateWithoutTeamInput[] };
            }
        }

        // Handle translations
        if (config.translations && Array.isArray(config.translations)) {
            data.translations = {
                create: config.translations.map(trans => ({
                    id: this.generateId(),
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
        ownerId: bigint,
        memberIds: bigint[] = [],
    ): Promise<team> {
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
    async createPrivateTeam(): Promise<team> {
        return await this.createMinimal({
            isPrivate: true,
            isOpenToNewMembers: false,
            handle: `private_team_${nanoid()}`,
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    name: "Private Team",
                }],
            },
        });
    }

    async createOpenCommunity(): Promise<team> {
        return await this.seedScenario("openCommunity");
    }

    async createOrganizationTeam(): Promise<team> {
        return await this.seedScenario("organizationTeam");
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

        // Check that team has at least one owner
        const owners = await this.prisma.member.count({
            where: {
                teamId: record.id,
                isAdmin: true,
                permissions: {
                    path: ["manageTeam"],
                    equals: true,
                },
            },
        });

        if (owners === 0) {
            violations.push("Team must have at least one owner");
        }

        // Check private team constraints
        if (record.isPrivate && record.isOpenToNewMembers) {
            violations.push("Private teams should not be open to new members");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            members: true,
            chats: true,
            translations: true,
            meetings: true,
            issues: true,
            emails: true,
            memberInvites: true,
            reports: true,
        };
    }

    protected async deleteRelatedRecords(
        record: team,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Helper to check if a relation should be deleted
        const shouldDelete = (relation: string) =>
            !includeOnly || includeOnly.includes(relation);

        // Delete in order of dependencies

        // Delete chats
        if (shouldDelete("chats")) {
            await tx.chat.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete meetings
        if (shouldDelete("meetings")) {
            await tx.meeting.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete issues
        if (shouldDelete("issues")) {
            await tx.issue.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete emails
        if (shouldDelete("emails")) {
            await tx.email.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete reports
        if (shouldDelete("reports")) {
            await tx.report.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete member invites
        if (shouldDelete("memberInvites")) {
            await tx.member_invite.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete members
        if (shouldDelete("members")) {
            await tx.member.deleteMany({
                where: { teamId: record.id },
            });
        }

        // Delete translations
        if (shouldDelete("translations")) {
            await tx.team_translation.deleteMany({
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
        userIds: bigint[],
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

    /**
     * Create a team hierarchy (teams with sub-teams)
     * Note: This would require a parent-child relationship in the schema
     */
    async createTeamHierarchy(levels = 3): Promise<team[]> {
        const teams: team[] = [];

        // Create root team
        const rootTeam = await this.createComplete({
            handle: `root_org_${nanoid()}`,
            translations: {
                create: [{
                    id: this.generateId(),
                    language: "en",
                    name: "Root Organization",
                }],
            },
        });
        teams.push(rootTeam);

        // Create sub-teams
        // Note: This is a placeholder - actual implementation would depend on schema
        for (let i = 0; i < levels - 1; i++) {
            const subTeam = await this.createMinimal({
                handle: `sub_team_${i}_${nanoid()}`,
                translations: {
                    create: [{
                        id: this.generateId(),
                        language: "en",
                        name: `Sub Team Level ${i + 1}`,
                    }],
                },
            });
            teams.push(subTeam);
        }

        return teams;
    }

    /**
     * Create teams for different testing scenarios
     */
    async createTestingScenarios(): Promise<{
        publicTeam: team;
        privateTeam: team;
        organizationTeam: team;
        communityTeam: team;
    }> {
        const [publicTeam, privateTeam, organizationTeam, communityTeam] = await Promise.all([
            this.createMinimal({
                isPrivate: false,
                translations: {
                    create: [{
                        id: this.generateId(),
                        language: "en",
                        name: "Public Team",
                    }],
                },
            }),
            this.createPrivateTeam(),
            this.createOrganizationTeam(),
            this.createOpenCommunity(),
        ]);

        return {
            publicTeam,
            privateTeam,
            organizationTeam: organizationTeam as unknown as team,
            communityTeam: communityTeam as unknown as team,
        };
    }
}

// Export factory creator function
export const createTeamDbFactory = (prisma: PrismaClient) =>
    new TeamDbFactory(prisma);

// Export the class for type usage
export { TeamDbFactory as TeamDbFactoryClass };
