/**
 * Simple Integration Test Helpers
 * 
 * Basic helper functions for integration testing without complex dependencies.
 * These helpers use the available fixtures and provide simple test data creation.
 */

import { generatePK } from "@vrooli/shared";
import { UserDbFactory, TeamDbFactory, ResourceDbFactory, ResourceVersionDbFactory } from "@vrooli/server/test-fixtures";
import { DbProvider } from "@vrooli/server";

// Session user type for testing
export interface SessionUser {
    id: string;
    name: string;
    email: string;
    accountStatus?: string;
    isBot?: boolean;
    theme?: string;
}

// Simple type definitions for testing
export interface SimpleTestUser {
    id: string;
    name: string;
    email: string;
    session?: SessionUser;
}

export interface SimpleTestTeam {
    id: string;
    name: string;
    ownerId: string;
}

export interface SimpleTestProject {
    id: string;
    name: string;
    ownerId: string;
    teamId?: string;
}

/**
 * Create a simple test user with basic properties
 */
export async function createSimpleTestUser(overrides: Partial<SimpleTestUser> = {}): Promise<{
    user: SimpleTestUser;
    sessionData: SessionUser;
}> {
    const prisma = DbProvider.get();
    const userFactory = new UserDbFactory(prisma);

    // Create user with auth and email using factory
    const dbUser = await userFactory.createWithRelations({
        overrides: {
            name: overrides.name || "Test User",
            handle: `test_${generatePK().slice(-8)}`,
        },
        withAuth: true,
        withEmails: true,
    });

    // Extract the user data we need
    const user: SimpleTestUser = {
        id: dbUser.id,
        name: dbUser.name || "Test User",
        email: dbUser.emails?.[0]?.emailAddress || `test-${dbUser.id}@example.com`,
        ...overrides,
    };

    // Create session data
    const sessionData: SessionUser = {
        id: dbUser.id,
        name: dbUser.name || "Test User",
        email: dbUser.emails?.[0]?.emailAddress || `test-${dbUser.id}@example.com`,
        accountStatus: "Unlocked",
        isBot: dbUser.isBot || false,
        theme: dbUser.theme || "light",
    };

    return { user, sessionData };
}

/**
 * Create a simple test team
 */
export async function createSimpleTestTeam(
    owner: SimpleTestUser,
    overrides: Partial<SimpleTestTeam> = {},
): Promise<SimpleTestTeam> {
    const prisma = DbProvider.get();
    const teamFactory = new TeamDbFactory(prisma);

    // Create team using factory
    const dbTeam = await teamFactory.createWithRelations({
        overrides: {
            handle: `team_${generatePK().slice(-8)}`,
        },
        owner: { userId: owner.id },
        translations: [{
            language: "en",
            name: overrides.name || "Test Team",
        }],
    });

    const team: SimpleTestTeam = {
        id: dbTeam.id,
        name: dbTeam.translations?.[0]?.name || "Test Team",
        ownerId: owner.id,
        ...overrides,
    };

    return team;
}

/**
 * Create a simple test project
 */
export async function createSimpleTestProject(
    owner: SimpleTestUser,
    team?: SimpleTestTeam,
    overrides: Partial<SimpleTestProject> = {},
): Promise<SimpleTestProject> {
    const prisma = DbProvider.get();
    const resourceFactory = new ResourceDbFactory(prisma);

    // Create project resource using factory
    const dbResource = await resourceFactory.createWithRelations({
        resourceType: "Project",
        owner: team ? { teamId: team.id } : { userId: owner.id },
        overrides: {
            isPrivate: false,
        },
        versions: [{
            versionLabel: "1.0.0",
            isLatest: true,
            isComplete: true,
            translations: [{
                language: "en",
                name: overrides.name || "Test Project",
                description: "Test project for integration testing",
            }],
        }],
        translations: [{
            language: "en",
            name: overrides.name || "Test Project",
            description: "Test project for integration testing",
        }],
    });

    const project: SimpleTestProject = {
        id: dbResource.id,
        name: dbResource.translations?.[0]?.name || "Test Project",
        ownerId: owner.id,
        teamId: team?.id,
        ...overrides,
    };

    return project;
}

/**
 * Clean up test data (simple version)
 */
export async function cleanupSimpleTestData(ids: string[]) {
    const prisma = DbProvider.get();
    if (!prisma || ids.length === 0) return;

    try {
        // Delete in order to respect foreign key constraints
        await prisma.comment.deleteMany({
            where: { id: { in: ids } },
        });

        await prisma.resource.deleteMany({
            where: { id: { in: ids } },
        });

        await prisma.team.deleteMany({
            where: { id: { in: ids } },
        });

        await prisma.user.deleteMany({
            where: { id: { in: ids } },
        });
    } catch (error) {
        console.warn("Could not clean up test data:", error);
    }
}

/**
 * Simple test context creator
 */
export function createSimpleTestContext(user: SimpleTestUser) {
    return {
        user: {
            id: user.id,
            name: user.name,
            email: user.email,
        },
        req: {},
        res: {},
    };
}

/**
 * Create test objects that can be commented on
 * Supports: Project, Routine, Standard, SmartContract, Api (via ResourceVersion)
 */
export async function createTestCommentTarget(
    type: "Project" | "Routine" | "Standard" | "SmartContract" | "Api",
    ownerId?: string,
): Promise<string> {
    const prisma = DbProvider.get();
    
    // Create a simple user if none provided
    let userId = ownerId;
    if (!userId) {
        const { user } = await createSimpleTestUser();
        userId = user.id;
    }

    switch (type) {
        case "Project": {
            const resourceFactory = new ResourceDbFactory(prisma);
            const resource = await resourceFactory.createWithRelations({
                resourceType: "Project",
                owner: { userId },
                overrides: { isPrivate: false },
                versions: [{
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    translations: [{
                        language: "en",
                        name: `Test ${type}`,
                        description: `Test ${type} for commenting`,
                    }],
                }],
                translations: [{
                    language: "en",
                    name: `Test ${type}`,
                    description: `Test ${type} for commenting`,
                }],
            });
            // Return the resource's version ID since comments attach to ResourceVersion
            return resource.versions?.[0]?.id || resource.id;
        }

        case "Routine": {
            const resourceFactory = new ResourceDbFactory(prisma);
            const resource = await resourceFactory.createWithRelations({
                resourceType: "Routine",
                owner: { userId },
                overrides: { isPrivate: false },
                versions: [{
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    translations: [{
                        language: "en",
                        name: `Test ${type}`,
                        description: `Test ${type} for commenting`,
                    }],
                }],
            });
            // Return the resource's version ID since comments attach to ResourceVersion  
            return resource.versions?.[0]?.id || resource.id;
        }

        case "Standard":
        case "SmartContract":
        case "Api": {
            // These are ResourceTypes, so create using ResourceVersionDbFactory
            const resourceVersionFactory = new ResourceVersionDbFactory(prisma);
            const resourceVersion = await resourceVersionFactory.createWithRelations({
                owner: { userId },
                overrides: {
                    versionLabel: "1.0.0",
                    isLatest: true,
                    isComplete: true,
                    isPrivate: false,
                },
                resource: {
                    isPrivate: false,
                    resourceType: type,
                },
                translations: [{
                    language: "en",
                    name: `Test ${type}`,
                    description: `Test ${type} for commenting`,
                }],
            });
            return resourceVersion.id;
        }

        default:
            throw new Error(`Unsupported comment target type: ${type}`);
    }
}
