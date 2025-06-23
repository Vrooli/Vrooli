import { getPrisma } from "../setup/test-setup.js";
import type { SessionData } from "@vrooli/server";
import { generatePK, initIdGenerator } from "@vrooli/shared";

/**
 * Creates a test user with session
 */
export async function createTestUser(overrides?: Partial<any>) {
    const prisma = getPrisma();
    
    // Initialize ID generator if not already done
    try {
        await initIdGenerator(0);
    } catch (e) {
        // Already initialized
    }
    
    const user = await prisma.user.create({
        data: {
            id: generatePK(),
            handle: `test-user-${Date.now()}`,
            isPrivate: false,
            isPrivateAccountsOnly: false,
            isPrivateApis: false,
            isPrivateAwards: false,
            isPrivateBookmarks: false,
            isPrivateCodes: false,
            isPrivateDonations: false,
            isPrivateProjects: false,
            isPrivatePrompts: false,
            isPrivateQuizzes: false,
            isPrivateReminders: false,
            isPrivateRoutines: false,
            isPrivateRunProjects: false,
            isPrivateRunRoutines: false,
            isPrivateStandards: false,
            isPrivateTeams: false,
            isPrivateVotes: false,
            emails: {
                create: {
                    emailAddress: `test-${Date.now()}@example.com`,
                    verified: true,
                },
            },
            ...overrides,
        },
        include: {
            emails: true,
        },
    });

    // Create session
    const sessionData: SessionData = {
        isLoggedIn: true,
        languages: ["en"],
        users: [{
            id: user.id,
            handle: user.handle,
            theme: "light",
        }],
    };

    return { user, sessionData };
}

/**
 * Creates a test team with owner
 */
export async function createTestTeam(ownerId: string, overrides?: Partial<any>) {
    const prisma = getPrisma();
    
    return await prisma.team.create({
        data: {
            id: generatePK(),
            handle: `test-team-${Date.now()}`,
            isOpenToNewMembers: true,
            members: {
                create: {
                    userId: ownerId,
                    isAdmin: true,
                },
            },
            ...overrides,
        },
        include: {
            members: true,
        },
    });
}

/**
 * Makes an API request with session data
 */
export async function makeAuthenticatedRequest(
    endpoint: string,
    method: string,
    sessionData: SessionData,
    body?: any,
) {
    // This would integrate with your actual API testing setup
    // For now, returning a placeholder
    return {
        endpoint,
        method,
        sessionData,
        body,
    };
}

/**
 * Waits for background jobs to complete
 */
export async function waitForJobs(_timeoutMs = 5000) {
    // In real implementation, this would check job queues
    await new Promise(resolve => setTimeout(resolve, 100));
}

// Remove deep imports that aren't exported
