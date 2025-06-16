import { AccountStatus, generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for User model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const userDbIds = {
    user1: generatePK(),
    user2: generatePK(),
    user3: generatePK(),
    bot1: generatePK(),
    bot2: generatePK(),
    auth1: generatePK(),
    auth2: generatePK(),
    auth3: generatePK(),
    email1: generatePK(),
    email2: generatePK(),
    email3: generatePK(),
};

/**
 * Minimal user data for database creation
 */
export const minimalUserDb: Prisma.UserCreateInput = {
    id: userDbIds.user1,
    publicId: generatePublicId(),
    name: "Test User",
    handle: `testuser_${nanoid(6)}`,
    status: AccountStatus.Unlocked,
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
};

/**
 * User with authentication
 */
export const userWithAuthDb: Prisma.UserCreateInput = {
    id: userDbIds.user2,
    publicId: generatePublicId(),
    name: "Authenticated User",
    handle: `authuser_${nanoid(6)}`,
    status: AccountStatus.Unlocked,
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
    auths: {
        create: [{
            id: userDbIds.auth1,
            provider: "Password",
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
        }],
    },
    emails: {
        create: [{
            id: userDbIds.email1,
            emailAddress: "authuser@example.com",
            verifiedAt: new Date(),
        }],
    },
};

/**
 * Complete user with all features
 */
export const completeUserDb: Prisma.UserCreateInput = {
    id: userDbIds.user3,
    publicId: generatePublicId(),
    name: "Complete User",
    handle: `complete_${nanoid(6)}`,
    status: AccountStatus.Unlocked,
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
    theme: "light",
    bannerImage: "https://example.com/banner.jpg",
    profileImage: "https://example.com/profile.jpg",
    auths: {
        create: [{
            id: userDbIds.auth2,
            provider: "Password",
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
        }],
    },
    emails: {
        create: [
            {
                id: userDbIds.email2,
                emailAddress: "complete@example.com",
                verifiedAt: new Date(),
            },
            {
                id: userDbIds.email3,
                emailAddress: "complete.secondary@example.com",
                verifiedAt: null,
            },
        ],
    },
    translations: {
        create: [
            {
                id: generatePK(),
                language: "en",
                bio: "I'm a test user with a complete profile",
            },
            {
                id: generatePK(),
                language: "es",
                bio: "Soy un usuario de prueba con un perfil completo",
            },
        ],
    },
};

/**
 * Bot user
 */
export const botUserDb: Prisma.UserCreateInput = {
    id: userDbIds.bot1,
    publicId: generatePublicId(),
    name: "Test Bot",
    handle: `testbot_${nanoid(6)}`,
    status: AccountStatus.Unlocked,
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    botSettings: {
        assistantId: "asst_test123",
        model: "gpt-4",
        temperature: 0.7,
    },
};

/**
 * Factory for creating user database fixtures with overrides
 */
export class UserDbFactory {
    static createMinimal(overrides?: Partial<Prisma.UserCreateInput>): Prisma.UserCreateInput {
        return {
            ...minimalUserDb,
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `testuser_${nanoid(6)}`,
            ...overrides,
        };
    }

    static createWithAuth(overrides?: Partial<Prisma.UserCreateInput>): Prisma.UserCreateInput {
        const email = overrides?.emails?.create?.[0]?.emailAddress || `user_${nanoid(6)}@example.com`;
        return {
            ...userWithAuthDb,
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `authuser_${nanoid(6)}`,
            auths: {
                create: [{
                    id: generatePK(),
                    provider: "Password",
                    hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                }],
            },
            emails: {
                create: [{
                    id: generatePK(),
                    emailAddress: email,
                    verifiedAt: new Date(),
                }],
            },
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.UserCreateInput>): Prisma.UserCreateInput {
        return {
            ...completeUserDb,
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `complete_${nanoid(6)}`,
            ...overrides,
        };
    }

    static createBot(overrides?: Partial<Prisma.UserCreateInput>): Prisma.UserCreateInput {
        return {
            ...botUserDb,
            id: generatePK(),
            publicId: generatePublicId(),
            handle: `bot_${nanoid(6)}`,
            ...overrides,
        };
    }

    /**
     * Create user with specific permissions/roles
     */
    static createWithRoles(
        roles: Array<{ id: string; name: string }>,
        overrides?: Partial<Prisma.UserCreateInput>
    ): Prisma.UserCreateInput {
        return {
            ...this.createWithAuth(overrides),
            roles: {
                connect: roles.map(role => ({ id: role.id })),
            },
        };
    }

    /**
     * Create user with teams
     */
    static createWithTeams(
        teams: Array<{ teamId: string; role: string }>,
        overrides?: Partial<Prisma.UserCreateInput>
    ): Prisma.UserCreateInput {
        return {
            ...this.createWithAuth(overrides),
            memberOf: {
                create: teams.map(team => ({
                    id: generatePK(),
                    team: { connect: { id: team.teamId } },
                    role: team.role,
                })),
            },
        };
    }
}

/**
 * Helper to create a user that matches the shape expected by mockAuthenticatedSession
 */
export function createSessionUser(overrides?: Partial<Prisma.UserCreateInput>) {
    const userData = UserDbFactory.createWithAuth(overrides);
    return {
        ...userData,
        lastLoginAttempt: new Date(),
        logInAttempts: 0,
        updatedAt: new Date(),
        languages: ["en"],
        premium: null,
        sessions: [],
    };
}

/**
 * Helper to seed multiple test users
 */
export async function seedTestUsers(
    prisma: any,
    count: number = 3,
    options?: {
        withAuth?: boolean;
        withBots?: boolean;
        teamId?: string;
    }
) {
    const users = [];

    for (let i = 0; i < count; i++) {
        const userData = options?.withAuth 
            ? UserDbFactory.createWithAuth({ name: `Test User ${i + 1}` })
            : UserDbFactory.createMinimal({ name: `Test User ${i + 1}` });

        if (options?.teamId) {
            userData.memberOf = {
                create: [{
                    id: generatePK(),
                    team: { connect: { id: options.teamId } },
                    role: i === 0 ? "Owner" : "Member",
                }],
            };
        }

        users.push(await prisma.user.create({ data: userData }));
    }

    // Add bots if requested
    if (options?.withBots) {
        const bot = await prisma.user.create({
            data: UserDbFactory.createBot({ name: "Test Bot" }),
        });
        users.push(bot);
    }

    return users;
}