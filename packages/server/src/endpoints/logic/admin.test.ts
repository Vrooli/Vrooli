import { type AdminSiteStatsOutput, type AdminUserListInput, type AdminUserUpdateStatusInput, AccountStatus, SEEDED_IDS } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { testEndpointRequiresAuth } from "../../__test/endpoints.js";
import { mockApiSession, mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { admin_siteStats } from "../generated/admin_siteStats.js";
import { admin_userList } from "../generated/admin_userList.js";
import { admin_userUpdateStatus } from "../generated/admin_userUpdateStatus.js";
import { admin_userResetPassword } from "../generated/admin_userResetPassword.js";
import { admin } from "./admin.js";

// Import database fixtures for seeding
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

describe("EndpointsAdmin", () => {
    let adminUser: any;
    let regularUser: any;
    let testUsers: any[];

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warn").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Reset Redis and database tables
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Create admin user
        adminUser = await DbProvider.get().user.create({
            data: UserDbFactory.createMinimal({
                id: SEEDED_IDS.User.Admin,
                name: "Admin User",
                handle: "admin-" + Math.floor(Math.random() * 1000),
                role: "Admin",
            }),
        });

        // Create regular user
        regularUser = await DbProvider.get().user.create({
            data: UserDbFactory.createMinimal({
                name: "Regular User",
                handle: "regular-" + Math.floor(Math.random() * 1000),
            }),
        });

        // Seed additional test users
        testUsers = await seedTestUsers(DbProvider.get(), 5, { withAuth: true });
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();
    });

    describe("siteStats", () => {
        it("should require authentication", async () => {
            await testEndpointRequiresAuth(admin.siteStats, admin_siteStats, {});
        });

        it("should require admin privileges", async () => {
            const response = await admin.siteStats({
                input: undefined,
                context: {
                    req: {
                        session: {
                            userId: regularUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("Admin privileges required");
        });

        it("should return site statistics for admin user", async () => {
            // Create some test data
            await DbProvider.get().routine.createMany({
                data: Array.from({ length: 3 }, () => ({
                    id: BigInt(Math.floor(Math.random() * 1e9)),
                    isInternal: false,
                    routineVersionId: null,
                })),
            });

            // Add credit entries
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    currentBalance: BigInt(1000000), // 1 credit
                    lifetimeCredits: BigInt(1000000),
                    userId: regularUser.id,
                },
            });

            const response = await admin.siteStats({
                input: undefined,
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

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
            // Mock Redis to be unavailable
            const originalGet = CacheService.get;
            vi.spyOn(CacheService, "get").mockImplementation(() => ({
                ...originalGet(),
                redis: null,
            }));

            const response = await admin.siteStats({
                input: undefined,
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            // Should still return data even if Redis is unavailable
            expect(response).toHaveProperty("creditStats");
            expect(response.creditStats.lastRolloverJobStatus).toBe("never_run");
            expect(response.creditStats.lastRolloverJobTime).toBeNull();
        });
    });

    describe("userList", () => {
        it("should require authentication", async () => {
            await testEndpointRequiresAuth(admin.userList, admin_userList, {});
        });

        it("should require admin privileges", async () => {
            const response = await admin.userList({
                input: {},
                context: {
                    req: {
                        session: {
                            userId: regularUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("Admin privileges required");
        });

        it("should return paginated user list", async () => {
            const response = await admin.userList({
                input: {
                    skip: 0,
                    take: 10,
                    sortBy: "createdAt",
                    sortOrder: "desc",
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response).toHaveProperty("users");
            expect(response).toHaveProperty("totalCount");
            expect(Array.isArray(response.users)).toBe(true);
            expect(response.users.length).toBeGreaterThan(0);
            expect(response.users[0]).toHaveProperty("id");
            expect(response.users[0]).toHaveProperty("createdAt");
        });

        it("should filter users by search term", async () => {
            const response = await admin.userList({
                input: {
                    searchTerm: adminUser.name,
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response.users.length).toBeGreaterThan(0);
            expect(response.users.some(u => u.name === adminUser.name)).toBe(true);
        });

        it("should filter users by status", async () => {
            // Lock a user
            await DbProvider.get().user.update({
                where: { id: regularUser.id },
                data: { status: AccountStatus.SoftLocked },
            });

            const response = await admin.userList({
                input: {
                    status: AccountStatus.SoftLocked,
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response.users.length).toBeGreaterThan(0);
            expect(response.users.every(u => u.status === AccountStatus.SoftLocked)).toBe(true);
        });
    });

    describe("userUpdateStatus", () => {
        it("should require authentication", async () => {
            await testEndpointRequiresAuth(admin.userUpdateStatus, admin_userUpdateStatus, {
                userId: regularUser.id.toString(),
                status: AccountStatus.SoftLocked,
            });
        });

        it("should require admin privileges", async () => {
            const response = await admin.userUpdateStatus({
                input: {
                    userId: regularUser.id.toString(),
                    status: AccountStatus.SoftLocked,
                },
                context: {
                    req: {
                        session: {
                            userId: regularUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("Admin privileges required");
        });

        it("should update user status", async () => {
            const response = await admin.userUpdateStatus({
                input: {
                    userId: regularUser.id.toString(),
                    status: AccountStatus.SoftLocked,
                    reason: "Test reason",
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response).toHaveProperty("id");
            expect(response).toHaveProperty("status");
            expect(response.status).toBe(AccountStatus.SoftLocked);

            // Verify in database
            const updatedUser = await DbProvider.get().user.findUnique({
                where: { id: regularUser.id },
            });
            expect(updatedUser?.status).toBe(AccountStatus.SoftLocked);
            expect(updatedUser?.statusReason).toBe("Test reason");
        });

        it("should prevent admin from modifying their own status", async () => {
            const response = await admin.userUpdateStatus({
                input: {
                    userId: adminUser.id.toString(),
                    status: AccountStatus.SoftLocked,
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("Cannot modify your own account status");
        });

        it("should return error for non-existent user", async () => {
            const response = await admin.userUpdateStatus({
                input: {
                    userId: "999999999",
                    status: AccountStatus.SoftLocked,
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("User not found");
        });
    });

    describe("userResetPassword", () => {
        let userWithEmail: any;

        beforeEach(async () => {
            // Create user with email
            userWithEmail = await DbProvider.get().user.create({
                data: {
                    ...UserDbFactory.createMinimal({
                        name: "User With Email",
                        handle: "email-user-" + Math.floor(Math.random() * 1000),
                    }),
                    emails: {
                        create: {
                            id: BigInt(Math.floor(Math.random() * 1e9)),
                            emailAddress: "test@example.com",
                            verifiedAt: new Date(),
                        },
                    },
                },
                include: {
                    emails: true,
                },
            });
        });

        it("should require authentication", async () => {
            await testEndpointRequiresAuth(admin.userResetPassword, admin_userResetPassword, {
                userId: userWithEmail.id.toString(),
            });
        });

        it("should require admin privileges", async () => {
            const response = await admin.userResetPassword({
                input: {
                    userId: userWithEmail.id.toString(),
                },
                context: {
                    req: {
                        session: {
                            userId: regularUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("Admin privileges required");
        });

        it("should reset user password and invalidate sessions", async () => {
            // Create a session for the user
            await DbProvider.get().session.create({
                data: {
                    id: "test-session-" + Math.floor(Math.random() * 1000),
                    userId: userWithEmail.id,
                    refreshToken: "test-refresh-token",
                    expiresAt: new Date(Date.now() + 86400000), // 1 day from now
                },
            });

            // Create an API key for the user
            await DbProvider.get().apiKey.create({
                data: {
                    id: BigInt(Math.floor(Math.random() * 1e9)),
                    name: "Test API Key",
                    hashedKey: "hashed-key",
                    createdByUserId: userWithEmail.id,
                    userId: userWithEmail.id,
                },
            });

            const response = await admin.userResetPassword({
                input: {
                    userId: userWithEmail.id.toString(),
                    reason: "Security breach",
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response).toEqual({ success: true });

            // Verify sessions are deleted
            const sessions = await DbProvider.get().session.findMany({
                where: { userId: userWithEmail.id },
            });
            expect(sessions.length).toBe(0);

            // Verify API keys are disabled
            const apiKeys = await DbProvider.get().apiKey.findMany({
                where: { userId: userWithEmail.id },
            });
            expect(apiKeys.every(key => key.disabledAt !== null)).toBe(true);
        });

        it("should return error for user without email", async () => {
            const response = await admin.userResetPassword({
                input: {
                    userId: regularUser.id.toString(),
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("User has no email address");
        });

        it("should return error for non-existent user", async () => {
            const response = await admin.userResetPassword({
                input: {
                    userId: "999999999",
                },
                context: {
                    req: {
                        session: {
                            userId: adminUser.id,
                        },
                    },
                },
            });

            expect(response).toBeInstanceOf(Error);
            expect(response.message).toContain("User not found");
        });
    });
});