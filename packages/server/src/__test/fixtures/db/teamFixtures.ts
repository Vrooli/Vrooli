import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Team model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const teamDbIds = {
    team1: generatePK(),
    team2: generatePK(),
    team3: generatePK(),
    team4: generatePK(),
    parent1: generatePK(),
    translation1: generatePK(),
    translation2: generatePK(),
    translation3: generatePK(),
    translation4: generatePK(),
    member1: generatePK(),
    member2: generatePK(),
    member3: generatePK(),
};

/**
 * Enhanced test fixtures for Team model following standard structure
 */
export const teamDbFixtures: DbTestFixtures<Prisma.teamCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        handle: `testteam_${nanoid(6)}`,
        isPrivate: false,
        isOpenToNewMembers: true,
        translations: {
            create: [{
                id: generatePK(),
                language: "en",
                name: "Test Team",
            }],
        },
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        handle: `complete_${nanoid(6)}`,
        isPrivate: false,
        isOpenToNewMembers: true,
        bannerImage: "https://example.com/banner.jpg",
        profileImage: "https://example.com/profile.jpg",
        config: {
            theme: "light",
            allowDirectMessages: true,
            publicProfile: true,
        },
        permissions: JSON.stringify({
            canEditTeam: ["Admin", "Owner"],
            canInviteMembers: ["Admin", "Owner", "Member"],
            canCreateResources: ["Admin", "Owner", "Member"],
        }),
        translations: {
            create: [
                {
                    id: generatePK(),
                    language: "en",
                    name: "Complete Team",
                    bio: "A fully-featured test team with all capabilities",
                },
                {
                    id: generatePK(),
                    language: "es",
                    name: "Equipo Completo",
                    bio: "Un equipo de prueba completo con todas las capacidades",
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required id, handle, translations
            isPrivate: false,
            isOpenToNewMembers: true,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            handle: true, // Should be string
            isPrivate: "yes", // Should be boolean
            isOpenToNewMembers: "true", // Should be boolean
        },
        duplicateHandle: {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: "duplicate_team_handle", // Same handle as another team
            isPrivate: false,
            isOpenToNewMembers: true,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Duplicate Team",
                }],
            },
        },
        invalidTranslations: {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `invalid_trans_${nanoid(6)}`,
            isPrivate: false,
            isOpenToNewMembers: true,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "invalid-lang", // Invalid language code
                    name: "", // Empty name
                }],
            },
        },
    },
    edgeCases: {
        maxLengthHandle: {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: "a".repeat(64), // Maximum handle length
            isPrivate: false,
            isOpenToNewMembers: true,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Max Length Handle Team",
                }],
            },
        },
        privateClosedTeam: {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `private_${nanoid(6)}`,
            isPrivate: true,
            isOpenToNewMembers: false,
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Private Closed Team",
                    bio: "A private team closed to new members",
                }],
            },
        },
        multiLanguageTeam: {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `multilang_${nanoid(6)}`,
            isPrivate: false,
            isOpenToNewMembers: true,
            translations: {
                create: [
                    { id: generatePK(), language: "en", name: "Multilingual Team", bio: "English description" },
                    { id: generatePK(), language: "es", name: "Equipo Multilingüe", bio: "Descripción en español" },
                    { id: generatePK(), language: "fr", name: "Équipe Multilingue", bio: "Description française" },
                    { id: generatePK(), language: "de", name: "Mehrsprachiges Team", bio: "Deutsche Beschreibung" },
                    { id: generatePK(), language: "ja", name: "多言語チーム", bio: "日本語の説明" },
                ],
            },
        },
        teamWithComplexPermissions: {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `complex_perms_${nanoid(6)}`,
            isPrivate: false,
            isOpenToNewMembers: true,
            permissions: JSON.stringify({
                canEditTeam: ["Owner"],
                canInviteMembers: ["Admin", "Owner"],
                canCreateResources: ["Admin", "Owner", "Member"],
                canDeleteResources: ["Owner"],
                canManageMembers: ["Admin", "Owner"],
                canViewPrivateResources: ["Admin", "Owner", "Member"],
                canModerateDiscussions: ["Admin", "Owner"],
            }),
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Complex Permissions Team",
                    bio: "A team with complex permission structure",
                }],
            },
        },
        teamWithAdvancedConfig: {
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `advanced_config_${nanoid(6)}`,
            isPrivate: false,
            isOpenToNewMembers: true,
            config: {
                theme: "dark",
                allowDirectMessages: false,
                publicProfile: false,
                requireApprovalForJoining: true,
                maxMembers: 100,
                allowGuestViewing: false,
                enableNotifications: true,
                defaultLanguage: "en",
                timezone: "UTC",
            },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Advanced Config Team",
                    bio: "A team with advanced configuration options",
                }],
            },
        },
    },
};

/**
 * Legacy fixtures for backward compatibility
 */
export const minimalTeamDb: Prisma.teamCreateInput = {
    id: teamDbIds.team1,
    publicId: generatePublicId(),
    handle: `testteam_${nanoid(6)}`,
    isPrivate: false,
    isOpenToNewMembers: true,
    translations: {
        create: [{
            id: teamDbIds.translation1,
            language: "en",
            name: "Test Team",
        }],
    },
};

export const teamWithCreatorDb: Prisma.teamCreateInput = {
    id: teamDbIds.team2,
    publicId: generatePublicId(),
    handle: `creatorteam_${nanoid(6)}`,
    isPrivate: false,
    isOpenToNewMembers: true,
    translations: {
        create: [{
            id: teamDbIds.translation2,
            language: "en",
            name: "Creator Team",
            bio: "A team with a creator",
        }],
    },
};

export const completeTeamDb: Prisma.teamCreateInput = {
    id: teamDbIds.team3,
    publicId: generatePublicId(),
    handle: `complete_${nanoid(6)}`,
    isPrivate: false,
    isOpenToNewMembers: true,
    bannerImage: "https://example.com/banner.jpg",
    profileImage: "https://example.com/profile.jpg",
    config: {
        theme: "light",
        allowDirectMessages: true,
        publicProfile: true,
    },
    permissions: JSON.stringify({
        canEditTeam: ["Admin", "Owner"],
        canInviteMembers: ["Admin", "Owner", "Member"],
        canCreateResources: ["Admin", "Owner", "Member"],
    }),
    translations: {
        create: [
            {
                id: teamDbIds.translation3,
                language: "en",
                name: "Complete Team",
                bio: "A fully-featured test team with all capabilities",
            },
            {
                id: teamDbIds.translation4,
                language: "es",
                name: "Equipo Completo",
                bio: "Un equipo de prueba completo con todas las capacidades",
            },
        ],
    },
};

export const privateTeamDb: Prisma.teamCreateInput = {
    id: teamDbIds.team4,
    publicId: generatePublicId(),
    handle: `private_${nanoid(6)}`,
    isPrivate: true,
    isOpenToNewMembers: false,
    translations: {
        create: [{
            id: generatePK(),
            language: "en",
            name: "Private Team",
            bio: "A private team for testing access controls",
        }],
    },
};

/**
 * Enhanced factory for creating team database fixtures
 */
export class TeamDbFactory extends EnhancedDbFactory<Prisma.teamCreateInput> {
    
    /**
     * Get the test fixtures for Team model
     */
    protected getFixtures(): DbTestFixtures<Prisma.teamCreateInput> {
        return teamDbFixtures;
    }

    /**
     * Get Team-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: teamDbIds.team1, // Duplicate ID
                    publicId: generatePublicId(),
                    handle: "duplicate_team_handle",
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "Duplicate Team",
                        }],
                    },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    handle: `fk_team_${nanoid(6)}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    createdBy: { connect: { id: "non-existent-user-id" } },
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "FK Violation Team",
                        }],
                    },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    handle: "", // Empty handle violates check constraint
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "Check Constraint Team",
                        }],
                    },
                },
            },
            validation: {
                requiredFieldMissing: teamDbFixtures.invalid.missingRequired,
                invalidDataType: teamDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    handle: "a".repeat(100), // Handle too long
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "x".repeat(500), // Name too long
                            bio: "y".repeat(1000), // Bio too long
                        }],
                    },
                },
            },
            businessLogic: {
                privateButOpenToNewMembers: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    handle: `contradictory_${nanoid(6)}`,
                    isPrivate: true,
                    isOpenToNewMembers: true, // Contradictory settings
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "Contradictory Team Settings",
                        }],
                    },
                },
                teamWithoutTranslations: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    handle: `no_trans_${nanoid(6)}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    // Missing required translations
                },
                invalidPermissionsJson: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    handle: `invalid_perms_${nanoid(6)}`,
                    isPrivate: false,
                    isOpenToNewMembers: true,
                    permissions: "invalid-json-string", // Invalid JSON
                    translations: {
                        create: [{
                            id: generatePK(),
                            language: "en",
                            name: "Invalid Permissions Team",
                        }],
                    },
                },
            },
        };
    }

    /**
     * Add team memberships (members for the team being created)
     */
    protected addTeamMemberships(data: Prisma.teamCreateInput, members: Array<{ teamId: string; role: string }>): Prisma.teamCreateInput {
        // For Team model, we add members to the team being created
        return {
            ...data,
            members: {
                create: members.map(member => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: member.teamId } }, // Actually user ID in this context
                    role: member.role,
                    isAdmin: member.role === "Admin" || member.role === "Owner",
                })),
            },
        };
    }

    /**
     * Add other relations specific to Team model
     */
    protected addOtherRelations(data: Prisma.teamCreateInput): Prisma.teamCreateInput {
        return {
            ...data,
            tags: {
                create: [
                    { id: generatePK(), tag: { connect: { tag: "test" } } },
                    { id: generatePK(), tag: { connect: { tag: "team" } } },
                ],
            },
        };
    }

    /**
     * Team-specific validation
     */
    protected validateSpecific(data: Prisma.teamCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Team
        if (!data.handle) errors.push("Team handle is required");
        if (data.isPrivate === undefined) errors.push("Team isPrivate is required");
        if (data.isOpenToNewMembers === undefined) errors.push("Team isOpenToNewMembers is required");

        // Check business logic
        if (data.isPrivate && data.isOpenToNewMembers) {
            warnings.push("Private team should not be open to new members");
        }

        // Check translations
        if (!data.translations?.create || data.translations.create.length === 0) {
            errors.push("Team must have at least one translation");
        }

        // Check handle format
        if (data.handle && (data.handle.length < 3 || data.handle.length > 64)) {
            errors.push("Handle must be between 3 and 64 characters");
        }

        // Check permissions JSON if present
        if (data.permissions && typeof data.permissions === "string") {
            try {
                JSON.parse(data.permissions);
            } catch {
                errors.push("Permissions must be valid JSON");
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(overrides?: Partial<Prisma.teamCreateInput>): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createMinimal(overrides);
    }

    static createWithCreator(
        creatorId: string,
        overrides?: Partial<Prisma.teamCreateInput>
    ): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createMinimal({
            createdBy: { connect: { id: creatorId } },
            ...overrides,
        });
    }

    static createComplete(overrides?: Partial<Prisma.teamCreateInput>): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createComplete(overrides);
    }

    static createPrivate(overrides?: Partial<Prisma.teamCreateInput>): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createEdgeCase("privateClosedTeam");
    }

    /**
     * Create team with members
     */
    static createWithMembers(
        members: Array<{ userId: string; isAdmin?: boolean }>,
        overrides?: Partial<Prisma.teamCreateInput>
    ): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createMinimal({
            members: {
                create: members.map(member => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: member.userId } },
                    role: member.isAdmin ? "Admin" : "Member",
                    isAdmin: member.isAdmin ?? false,
                })),
            },
            ...overrides,
        });
    }

    /**
     * Create team with parent (for team forks/sub-teams)
     */
    static createWithParent(
        parentId: string,
        overrides?: Partial<Prisma.teamCreateInput>
    ): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createMinimal({
            parent: { connect: { id: parentId } },
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: "Sub Team",
                    bio: "A sub-team or fork of another team",
                }],
            },
            ...overrides,
        });
    }

    /**
     * Create team with tags
     */
    static createWithTags(
        tags: string[],
        overrides?: Partial<Prisma.teamCreateInput>
    ): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createMinimal({
            tags: {
                create: tags.map(tag => ({
                    id: generatePK(),
                    tag: { connect: { tag } },
                })),
            },
            ...overrides,
        });
    }

    /**
     * Create team with translations in multiple languages
     */
    static createWithTranslations(
        translations: Array<{ language: string; name: string; bio?: string }>,
        overrides?: Partial<Prisma.teamCreateInput>
    ): Prisma.teamCreateInput {
        const factory = new TeamDbFactory();
        return factory.createMinimal({
            translations: {
                create: translations.map(t => ({
                    id: generatePK(),
                    language: t.language,
                    name: t.name,
                    bio: t.bio,
                })),
            },
            ...overrides,
        });
    }
}

/**
 * Helper to create a team that can be used in test contexts
 */
export function createTestTeam(overrides?: Partial<Prisma.TeamCreateInput>) {
    const teamData = TeamDbFactory.createMinimal(overrides);
    return {
        ...teamData,
        createdAt: new Date(),
        updatedAt: new Date(),
        bookmarks: 0,
        views: 0,
        languages: ["en"],
    };
}

/**
 * Enhanced helper to seed multiple test teams with comprehensive options
 */
export async function seedTestTeams(
    prisma: any,
    count: number = 3,
    options?: BulkSeedOptions & {
        withCreator?: boolean;
        creatorId?: string;
        withMembers?: boolean;
        memberIds?: string[];
        isPrivate?: boolean;
    }
): Promise<BulkSeedResult<any>> {
    const factory = new TeamDbFactory();
    const teams = [];
    let authCount = 0;
    let botCount = 0;
    let teamCount = 0;

    for (let i = 0; i < count; i++) {
        const overrides = options?.overrides?.[i] || {
            translations: {
                create: [{
                    id: generatePK(),
                    language: "en",
                    name: `Test Team ${i + 1}`,
                }],
            },
        };
        
        let teamData: Prisma.teamCreateInput;
        
        if (options?.withAuth || options?.withCreator) {
            teamData = factory.createWithRelationships({ 
                withAuth: true, 
                overrides: {
                    ...overrides,
                    isPrivate: options?.isPrivate ?? false,
                    ...(options?.creatorId && { createdBy: { connect: { id: options.creatorId } } }),
                }
            }).data;
            authCount++;
        } else {
            teamData = factory.createMinimal({
                ...overrides,
                isPrivate: options?.isPrivate ?? false,
            });
        }

        // Add members if requested
        if (options?.withMembers && options.memberIds) {
            teamData.members = {
                create: options.memberIds.map((userId, index) => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: userId } },
                    role: index === 0 ? "Owner" : "Member",
                    isAdmin: index === 0, // First member is admin
                })),
            };
        }

        const team = await prisma.team.create({
            data: teamData,
            include: {
                translations: true,
                members: { include: { user: true } },
                createdBy: true,
            },
        });
        teams.push(team);
        teamCount++;
    }

    return {
        records: teams,
        summary: {
            total: teams.length,
            withAuth: authCount,
            bots: botCount,
            teams: teamCount,
        },
    };
}

/**
 * Helper to create team with full member setup
 */
export async function createTeamWithFullMembership(
    prisma: any,
    options: {
        ownerId: string;
        memberIds?: string[];
        adminIds?: string[];
        teamData?: Partial<Prisma.TeamCreateInput>;
    }
) {
    const memberIds = options.memberIds || [];
    const adminIds = options.adminIds || [];
    
    const teamData = TeamDbFactory.createWithCreator(options.ownerId, {
        ...options.teamData,
        members: {
            create: [
                // Owner is automatically an admin member
                {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: options.ownerId } },
                    isAdmin: true,
                },
                // Additional admin members
                ...adminIds.map(userId => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: userId } },
                    isAdmin: true,
                })),
                // Regular members
                ...memberIds.map(userId => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: userId } },
                    isAdmin: false,
                })),
            ],
        },
    });

    return await prisma.team.create({
        data: teamData,
        include: {
            translations: true,
            members: { 
                include: { user: { select: { id: true, name: true, handle: true } } }
            },
            createdBy: { select: { id: true, name: true, handle: true } },
        },
    });
}