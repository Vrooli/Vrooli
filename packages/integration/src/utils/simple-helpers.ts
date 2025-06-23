import { getPrisma } from "../setup/test-setup.js";
import { generatePK, initIdGenerator } from "@vrooli/shared";

/**
 * Creates a minimal test user
 */
export async function createSimpleTestUser() {
    const prisma = getPrisma();
    
    // Initialize ID generator if not already done
    try {
        await initIdGenerator(0);
    } catch (e) {
        // Already initialized
    }
    
    const userId = generatePK();
    const timestamp = Date.now();
    
    const user = await prisma.user.create({
        data: {
            id: userId,
            publicId: String(userId),
            name: "Test User",
            handle: `test-user-${timestamp}`,
            isPrivate: false,
            emails: {
                create: {
                    id: generatePK(),
                    emailAddress: `test-${timestamp}@example.com`,
                    verifiedAt: new Date(),
                },
            },
        },
        include: {
            emails: true,
        },
    });

    return {
        user,
        sessionData: {
            isLoggedIn: true,
            languages: ["en"],
            users: [{
                id: String(user.id),
                handle: user.handle,
                theme: "light",
            }],
        },
    };
}
