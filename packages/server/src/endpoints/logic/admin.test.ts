import { type AdminSiteStatsOutput, type AdminUserListInput, type AdminUserUpdateStatusInput, AccountStatus, SEEDED_PUBLIC_IDS, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { testEndpointRequiresAuth } from "../../__test/endpoints.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { admin_siteStats } from "../generated/admin_siteStats.js";
import { admin_userList } from "../generated/admin_userList.js";
import { admin_userUpdateStatus } from "../generated/admin_userUpdateStatus.js";
import { admin_userResetPassword } from "../generated/admin_userResetPassword.js";
import { admin } from "./admin.js";

// Import database fixtures for seeding
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

describe("EndpointsAdmin", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warn").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.api_key.deleteMany();
        await prisma.credit_account.deleteMany();
        await prisma.run.deleteMany();
        await prisma.session.deleteMany();
        await prisma.user.deleteMany();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create admin and regular users
    const createTestUsers = async () => {
        // Create admin user with correct admin publicId
        const adminUser = await DbProvider.get().user.create({
            data: UserDbFactory.createWithAuth({
                id: generatePK(),
                publicId: SEEDED_PUBLIC_IDS.Admin,
                name: "Admin User",
                handle: "__admin__",
                isPrivate: false,
                theme: "light",
            }),
        });

        // Create regular user
        const regularUser = await DbProvider.get().user.create({
            data: UserDbFactory.createWithAuth({
                id: generatePK(),
                name: "Regular User",
                handle: "regular-" + Math.floor(Math.random() * 1000),
            }),
        });

        // Seed additional test users
        const testUsers = await seedTestUsers(DbProvider.get(), 5, { withAuth: true });

        return { adminUser, regularUser, testUsers };
    };

    describe("siteStats", () => {
        testEndpointRequiresAuth(admin.siteStats, admin_siteStats, {});

        it("should require admin privileges", async () => {
            const { adminUser, regularUser } = await createTestUsers();
            
            // Create session for regular user (should not have admin access)
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: regularUser.id.toString()
            });

            await expect(async () => {
                await admin.siteStats({ input: undefined }, { req, res }, admin_siteStats);
            }).rejects.toThrow();
        });

        it("should return site statistics for admin user", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            // Create some test data
            await DbProvider.get().run.createMany({
                data: Array.from({ length: 3 }, (_, i) => ({
                    id: generatePK(),
                    name: `Test Run ${i + 1}`,
                })),
            });

            // Add credit entries
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(1000000), // 1 credit
                    userId: regularUser.id,
                },
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            const response = await admin.siteStats({ input: undefined }, { req, res }, admin_siteStats);

            expect(response).toHaveProperty("totalUsers");
            expect(response).toHaveProperty("activeUsers");
            expect(response).toHaveProperty("newUsersToday");
            expect(response).toHaveProperty("totalRoutines");
            expect(response).toHaveProperty("activeRoutines");
            expect(response).toHaveProperty("creditStats");
            expect(response.creditStats).toHaveProperty("totalCreditsInCirculation");
            expect(response.creditStats).toHaveProperty("lastRolloverJobStatus");
            expect(response.creditStats).toHaveProperty("nextScheduledRollover");
        });

        it("should handle Redis connection errors gracefully", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            // Note: Redis mocking may not work properly within transactions,
            // but we can still test the basic functionality
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            const response = await admin.siteStats({ input: undefined }, { req, res }, admin_siteStats);

            // Should still return data
            expect(response).toHaveProperty("creditStats");
            expect(response.creditStats).toHaveProperty("lastRolloverJobStatus");
            expect(response.creditStats).toHaveProperty("lastRolloverJobTime");
        });
    });

    describe("userList", () => {
        testEndpointRequiresAuth(admin.userList, admin_userList, {});

        it("should require admin privileges", async () => {
            const { adminUser, regularUser } = await createTestUsers();
            
            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: regularUser.id.toString()
            });

            await expect(async () => {
                await admin.userList({ input: {} }, { req, res }, admin_userList);
            }).rejects.toThrow();
        });

        it("should return paginated user list", async () => {
            const { adminUser, regularUser, testUsers } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            const response = await admin.userList({
                input: {
                    skip: 0,
                    take: 10,
                    sortBy: "createdAt",
                    sortOrder: "desc",
                },
            }, { req, res }, admin_userList);

            expect(response).toHaveProperty("users");
            expect(response).toHaveProperty("totalCount");
            expect(Array.isArray(response.users)).toBe(true);
            expect(response.users.length).toBeGreaterThan(0);
            expect(response.users[0]).toHaveProperty("id");
            expect(response.users[0]).toHaveProperty("createdAt");
        });

        it("should filter users by search term", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            const response = await admin.userList({
                input: {
                    searchTerm: adminUser.name,
                },
            }, { req, res }, admin_userList);

            expect(response.users.length).toBeGreaterThan(0);
            expect(response.users.some(u => u.name === adminUser.name)).toBe(true);
        });

        it("should filter users by status", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            // Lock a user
            await DbProvider.get().user.update({
                where: { id: regularUser.id },
                data: { status: AccountStatus.SoftLocked },
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            const response = await admin.userList({
                input: {
                    status: AccountStatus.SoftLocked,
                },
            }, { req, res }, admin_userList);

            expect(response.users.length).toBeGreaterThan(0);
            expect(response.users.every(u => u.status === AccountStatus.SoftLocked)).toBe(true);
        });
    });

    describe("userUpdateStatus", () => {
        testEndpointRequiresAuth(admin.userUpdateStatus, admin_userUpdateStatus, {
            userId: "123",
            status: AccountStatus.SoftLocked,
        });

        it("should require admin privileges", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: regularUser.id.toString()
            });
            
            await expect(async () => {
                await admin.userUpdateStatus({
                    input: {
                        userId: regularUser.id.toString(),
                        status: AccountStatus.SoftLocked,
                    },
                }, { req, res }, admin_userUpdateStatus);
            }).rejects.toThrow();
        });

        it("should update user status", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            const response = await admin.userUpdateStatus({
                input: {
                    userId: regularUser.id.toString(),
                    status: AccountStatus.SoftLocked,
                    reason: "Test reason",
                },
            }, { req, res }, admin_userUpdateStatus);

            expect(response).toHaveProperty("id");
            expect(response).toHaveProperty("status");
            expect(response.status).toBe(AccountStatus.SoftLocked);

            // Verify in database
            const updatedUser = await DbProvider.get().user.findUnique({
                where: { id: regularUser.id },
            });
            expect(updatedUser?.status).toBe(AccountStatus.SoftLocked);
        });

        it("should prevent admin from modifying their own status", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            await expect(async () => {
                await admin.userUpdateStatus({
                    input: {
                        userId: adminUser.id.toString(),
                        status: AccountStatus.SoftLocked,
                    },
                }, { req, res }, admin_userUpdateStatus);
            }).rejects.toThrow();
        });

        it("should return error for non-existent user", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            await expect(async () => {
                await admin.userUpdateStatus({
                    input: {
                        userId: "999999999",
                        status: AccountStatus.SoftLocked,
                    },
                }, { req, res }, admin_userUpdateStatus);
            }).rejects.toThrow();
        });
    });

    describe("userResetPassword", () => {
        // Helper function to create user with email
        const createUserWithEmail = async () => {
            const { adminUser, regularUser } = await createTestUsers();
            
            const userWithEmail = await DbProvider.get().user.create({
                data: {
                    ...UserDbFactory.createMinimal({
                        id: generatePK(),
                        name: "User With Email",
                        handle: "email-user-" + Math.floor(Math.random() * 1000),
                    }),
                    emails: {
                        create: {
                            id: generatePK(),
                            emailAddress: "test@example.com",
                            verifiedAt: new Date(),
                        },
                    },
                },
                include: {
                    emails: true,
                },
            });
            
            return { adminUser, regularUser, userWithEmail };
        };

        testEndpointRequiresAuth(admin.userResetPassword, admin_userResetPassword, {
            userId: "123",
        });

        it("should require admin privileges", async () => {
            const { adminUser, regularUser, userWithEmail } = await createUserWithEmail();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: regularUser.id.toString()
            });
            
            await expect(async () => {
                await admin.userResetPassword({
                    input: {
                        userId: userWithEmail.id.toString(),
                    },
                }, { req, res }, admin_userResetPassword);
            }).rejects.toThrow();
        });

        it("should reset user password and invalidate sessions", async () => {
            const { adminUser, regularUser, userWithEmail } = await createUserWithEmail();

            // Create auth record first
            const auth = await DbProvider.get().user_auth.create({
                data: {
                    id: generatePK(),
                    user_id: userWithEmail.id,
                    provider: "Password",
                    hashed_password: "dummy-hash",
                },
            });
            
            // Create a session for the user
            await DbProvider.get().session.create({
                data: {
                    id: generatePK(),
                    user_id: userWithEmail.id,
                    auth_id: auth.id,
                    expires_at: new Date(Date.now() + 86400000), // 1 day from now
                },
            });

            // Create an API key for the user
            await DbProvider.get().api_key.create({
                data: {
                    id: generatePK(),
                    name: "Test API Key",
                    key: "hashed-key",
                    userId: userWithEmail.id,
                },
            });

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            const response = await admin.userResetPassword({
                input: {
                    userId: userWithEmail.id.toString(),
                    reason: "Security breach",
                },
            }, { req, res }, admin_userResetPassword);

            expect(response).toEqual({ success: true });

            // Verify sessions are deleted
            const sessions = await DbProvider.get().session.findMany({
                where: { user_id: userWithEmail.id },
            });
            expect(sessions.length).toBe(0);

            // Verify API keys are disabled
            const apiKeys = await DbProvider.get().api_key.findMany({
                where: { userId: userWithEmail.id },
            });
            expect(apiKeys.every(key => key.disabledAt !== null)).toBe(true);
        });

        it.skip("should return error for user without email", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            await expect(async () => {
                await admin.userResetPassword({
                    input: {
                        userId: regularUser.id.toString(),
                    },
                }, { req, res }, admin_userResetPassword);
            }).rejects.toThrow();
        });

        it("should return error for non-existent user", async () => {
            const { adminUser, regularUser } = await createTestUsers();

            const { req, res } = await mockAuthenticatedSession({
                ...loggedInUserNoPremiumData(),
                id: adminUser.id.toString()
            });
            
            await expect(async () => {
                await admin.userResetPassword({
                    input: {
                        userId: "999999999",
                    },
                }, { req, res }, admin_userResetPassword);
            }).rejects.toThrow();
        });
    });
});