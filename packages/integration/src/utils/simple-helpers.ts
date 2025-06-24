/**
 * Simple Integration Test Helpers
 * 
 * Basic helper functions for integration testing without complex dependencies.
 * These helpers use the available fixtures and provide simple test data creation.
 */

import { generatePK } from "@vrooli/shared";
import { getPrisma } from "../setup/test-setup.js";
import { 
    userFixtures, 
    teamFixtures, 
    projectFixtures,
    sessionHelpers,
    type SessionUser,
} from "../fixtures/index.js";

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
    const prisma = getPrisma();
    
    // Use fixtures if available, otherwise create basic data
    const baseUser = userFixtures?.minimal?.create || {
        name: "Test User",
        email: "test@example.com",
    };
    
    const userId = overrides.id || generatePK();
    const userName = overrides.name || baseUser.name || "Test User";
    const userEmail = overrides.email || baseUser.email || `test-${userId}@example.com`;
    
    // Create basic user data
    const user: SimpleTestUser = {
        id: userId,
        name: userName,
        email: userEmail,
        ...overrides,
    };
    
    // Create session data
    const sessionData = await sessionHelpers.quickSession('standard');
    sessionData.id = userId;
    sessionData.name = userName;
    sessionData.email = userEmail;
    
    // If we have a database connection, create the actual user record
    if (prisma) {
        try {
            await prisma.user.create({
                data: {
                    id: userId,
                    name: userName,
                    emails: {
                        create: {
                            id: generatePK(),
                            emailAddress: userEmail,
                            isVerified: true,
                        },
                    },
                },
            });
        } catch (error) {
            console.warn("Could not create user in database:", error);
            // Continue without database record - tests can still use the mock data
        }
    }
    
    return { user, sessionData };
}

/**
 * Create a simple test team
 */
export async function createSimpleTestTeam(
    owner: SimpleTestUser,
    overrides: Partial<SimpleTestTeam> = {}
): Promise<SimpleTestTeam> {
    const prisma = getPrisma();
    
    const baseTeam = teamFixtures?.minimal?.create || {
        name: "Test Team",
    };
    
    const teamId = overrides.id || generatePK();
    const teamName = overrides.name || baseTeam.name || "Test Team";
    
    const team: SimpleTestTeam = {
        id: teamId,
        name: teamName,
        ownerId: owner.id,
        ...overrides,
    };
    
    // If we have a database connection, create the actual team record
    if (prisma) {
        try {
            await prisma.team.create({
                data: {
                    id: teamId,
                    creator: { connect: { id: owner.id } },
                    members: {
                        create: {
                            id: generatePK(),
                            user: { connect: { id: owner.id } },
                            role: "Owner",
                        },
                    },
                    translations: {
                        create: {
                            id: generatePK(),
                            language: "en",
                            name: teamName,
                        },
                    },
                },
            });
        } catch (error) {
            console.warn("Could not create team in database:", error);
        }
    }
    
    return team;
}

/**
 * Create a simple test project
 */
export async function createSimpleTestProject(
    owner: SimpleTestUser,
    team?: SimpleTestTeam,
    overrides: Partial<SimpleTestProject> = {}
): Promise<SimpleTestProject> {
    const prisma = getPrisma();
    
    const baseProject = projectFixtures?.minimal?.create || {
        name: "Test Project",
    };
    
    const projectId = overrides.id || generatePK();
    const projectName = overrides.name || baseProject.name || "Test Project";
    
    const project: SimpleTestProject = {
        id: projectId,
        name: projectName,
        ownerId: owner.id,
        teamId: team?.id,
        ...overrides,
    };
    
    // If we have a database connection, create the actual project record
    if (prisma) {
        try {
            await prisma.project.create({
                data: {
                    id: projectId,
                    isPrivate: false,
                    creator: { connect: { id: owner.id } },
                    ...(team ? { team: { connect: { id: team.id } } } : { 
                        owner: { connect: { id: owner.id } } 
                    }),
                    versions: {
                        create: {
                            id: generatePK(),
                            versionLabel: "1.0.0",
                            isComplete: true,
                            isPrivate: false,
                            creator: { connect: { id: owner.id } },
                            ...(team ? {} : { root: { connect: { id: owner.id } } }),
                            translations: {
                                create: {
                                    id: generatePK(),
                                    language: "en",
                                    name: projectName,
                                    description: "Test project for integration testing",
                                },
                            },
                        },
                    },
                },
            });
        } catch (error) {
            console.warn("Could not create project in database:", error);
        }
    }
    
    return project;
}

/**
 * Clean up test data (simple version)
 */
export async function cleanupSimpleTestData(ids: string[]) {
    const prisma = getPrisma();
    if (!prisma || ids.length === 0) return;
    
    try {
        // Delete in order to respect foreign key constraints
        await prisma.comment.deleteMany({
            where: { id: { in: ids } },
        });
        
        await prisma.project.deleteMany({
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