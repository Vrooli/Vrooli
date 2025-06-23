import { getPrisma } from "../setup/test-setup.js";
import { generatePK } from "@vrooli/shared";

export interface TestSessionData {
    isLoggedIn: boolean;
    languages: string[];
    users: Array<{
        id: string;
        handle: string | null;
        theme: string;
    }>;
}

/**
 * Creates a test user with session
 */
export async function createTestUser(overrides?: Partial<any>) {
    const prisma = getPrisma();
    
    const userId = String(generatePK());
    const user = await prisma.user.create({
        data: {
            id: userId,
            publicId: userId, // Required field
            name: "Test User", // Required field
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
    const sessionData: TestSessionData = {
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
