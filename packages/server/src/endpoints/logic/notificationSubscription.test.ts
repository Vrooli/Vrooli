import { type FindByIdInput, type NotificationSubscriptionCreateInput, type NotificationSubscriptionSearchInput, type NotificationSubscriptionUpdateInput, notificationSubscriptionTestDataFactory, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { mockAuthenticatedSession, mockLoggedOutSession, mockApiSession, mockReadPrivatePermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { testEndpointRequiresAuth, testEndpointRequiresApiKeyReadPermissions, testEndpointRequiresApiKeyWritePermissions } from "../../__test/endpoints.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { notificationSubscription_findOne } from "../generated/notificationSubscription_findOne.js";
import { notificationSubscription_findMany } from "../generated/notificationSubscription_findMany.js";
import { notificationSubscription_createOne } from "../generated/notificationSubscription_createOne.js";
import { notificationSubscription_updateOne } from "../generated/notificationSubscription_updateOne.js";
import { notificationSubscription } from "./notificationSubscription.js";
// Import database fixtures
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { NotificationSubscriptionDbFactory } from "../../__test/fixtures/db/notificationSubscriptionFixtures.js";

describe("EndpointsNotificationSubscription", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.notificationSubscription.deleteMany();
        await prisma.user.deleteMany();
        // Clear Redis cache
        await CacheService.get().flushAll();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    const createTestData = async () => {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
        
        // Create notification subscriptions for users
        const subscriptions = await Promise.all([
            // User 1 subscriptions
            DbProvider.get().notificationSubscription.create({
                data: NotificationSubscriptionDbFactory.createMinimal({
                    createdById: testUsers[0].id,
                    objectType: "Project",
                    objectId: generatePK(),
                    condition: "CRUD",
                }),
            }),
            DbProvider.get().notificationSubscription.create({
                data: NotificationSubscriptionDbFactory.createMinimal({
                    createdById: testUsers[0].id,
                    objectType: "Routine",
                    objectId: generatePK(),
                    condition: "CRUD",
                }),
            }),
            // User 2 subscription
            DbProvider.get().notificationSubscription.create({
                data: NotificationSubscriptionDbFactory.createMinimal({
                    createdById: testUsers[1].id,
                    objectType: "Team",
                    objectId: generatePK(),
                    condition: "CRUD",
                }),
            }),
        ]);

        return { testUsers, subscriptions };
    };

    describe("findOne", () => {
        describe("authentication", () => {
            it("requires authentication", async () => {
                await testEndpointRequiresAuth(
                    notificationSubscription.findOne,
                    { id: generatePK() },
                    notificationSubscription_findOne,
                );
            });

            it("requires API key with read permissions", async () => {
                await testEndpointRequiresApiKeyReadPermissions(
                    notificationSubscription.findOne,
                    { id: generatePK() },
                    notificationSubscription_findOne,
                    ["NotificationSubscription"],
                );
            });
        });

        describe("valid", () => {
            it("returns own notification subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: subscriptions[0].id.toString() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeDefined();
                expect(result.id).toBe(subscriptions[0].id.toString());
                expect(result.objectType).toBe("Project");
                expect(result.condition).toBe("CRUD");
                expect(result.user?.id).toBe(testUsers[0].id.toString());
            });

            it("returns subscription with API key", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockApiSession({
                    userId: testUsers[0].id.toString(),
                    apiKeyId: generatePK(),
                    permissions: mockReadPrivatePermissions(["NotificationSubscription"]),
                });

                const input: FindByIdInput = { id: subscriptions[0].id.toString() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeDefined();
                expect(result.id).toBe(subscriptions[0].id.toString());
            });

            it("returns subscription with complete details", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: subscriptions[0].id.toString() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result.objectType).toBe("Project");
                expect(result.objectId).toBeDefined();
                expect(result.condition).toBe("CRUD");
                expect(result.createdAt).toBeDefined();
                expect(result.updatedAt).toBeDefined();
                expect(result.user).toBeDefined();
            });
        });

        describe("invalid", () => {
            it("cannot access another user's subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(), // User 2 trying to access user 1's subscription
                });

                const input: FindByIdInput = { id: subscriptions[0].id.toString() }; // User 1's subscription
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeNull();
            });

            it("returns null for non-existent subscription", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: FindByIdInput = { id: generatePK() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeNull();
            });
        });
    });

    describe("findMany", () => {
        describe("authentication", () => {
            it("requires authentication", async () => {
                await testEndpointRequiresAuth(
                    notificationSubscription.findMany,
                    {},
                    notificationSubscription_findMany,
                );
            });

            it("requires API key with read permissions", async () => {
                await testEndpointRequiresApiKeyReadPermissions(
                    notificationSubscription.findMany,
                    {},
                    notificationSubscription_findMany,
                    ["NotificationSubscription"],
                );
            });
        });

        describe("valid", () => {
            it("returns user's notification subscriptions", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionSearchInput = {};
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(2); // User 1 has 2 subscriptions
                expect(result.totalCount).toBe(2);
                expect(result.results.every(s => s.user?.id === testUsers[0].id.toString())).toBe(true);
            });

            it("filters by object type", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionSearchInput = {
                    objectType: "Project",
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].objectType).toBe("Project");
            });

            it("filters by object ID", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionSearchInput = {
                    objectId: subscriptions[0].objectId.toString(),
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].objectId).toBe(subscriptions[0].objectId.toString());
            });

            it("filters by condition", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionSearchInput = {
                    condition: "CRUD",
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results.every(s => s.condition === "CRUD")).toBe(true);
            });

            it("sorts results", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionSearchInput = {
                    sortBy: "DateCreatedDesc",
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                // Verify results are sorted by creation date (newest first)
                for (let i = 1; i < result.results.length; i++) {
                    const prevDate = new Date(result.results[i - 1].createdAt);
                    const currDate = new Date(result.results[i].createdAt);
                    expect(prevDate.getTime()).toBeGreaterThanOrEqual(currDate.getTime());
                }
            });

            it("paginates results", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionSearchInput = {
                    take: 1,
                    skip: 0,
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.totalCount).toBe(2); // Total available
            });

            it("returns empty results for user with no subscriptions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: NotificationSubscriptionSearchInput = {};
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(0);
                expect(result.totalCount).toBe(0);
            });

            it("only returns own subscriptions", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(), // User 2
                });

                const input: NotificationSubscriptionSearchInput = {};
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(1); // User 2 has 1 subscription
                expect(result.results[0].user?.id).toBe(testUsers[1].id.toString());
                expect(result.results[0].objectType).toBe("Team");
            });
        });
    });

    describe("createOne", () => {
        describe("authentication", () => {
            it("requires authentication", async () => {
                await testEndpointRequiresAuth(
                    notificationSubscription.createOne,
                    notificationSubscriptionTestDataFactory.createMinimal(),
                    notificationSubscription_createOne,
                );
            });

            it("requires API key with write permissions", async () => {
                await testEndpointRequiresApiKeyWritePermissions(
                    notificationSubscription.createOne,
                    notificationSubscriptionTestDataFactory.createMinimal(),
                    notificationSubscription_createOne,
                    ["NotificationSubscription"],
                );
            });
        });

        describe("valid", () => {
            it("creates notification subscription", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK(),
                    objectType: "Project",
                    objectId: generatePK(),
                    condition: "CRUD",
                });

                const result = await notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("NotificationSubscription");
                expect(result.id).toBe(input.id);
                expect(result.objectType).toBe("Project");
                expect(result.condition).toBe("CRUD");
                expect(result.user?.id).toBe(testUser[0].id.toString());

                // Verify in database
                const created = await DbProvider.get().notificationSubscription.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(created).toBeDefined();
                expect(created?.objectType).toBe("Project");
                expect(created?.condition).toBe("CRUD");
            });

            it("creates subscription for different object types", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const objectTypes = ["Project", "Routine", "Team", "User"];
                const results = [];

                for (const objectType of objectTypes) {
                    const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                        id: generatePK(),
                        objectType: objectType as any,
                        objectId: generatePK(),
                        condition: "CRUD",
                    });

                    const result = await notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne);
                    results.push(result);
                    expect(result.objectType).toBe(objectType);
                }

                expect(results).toHaveLength(4);
            });

            it("creates subscription with different conditions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const conditions = ["CRUD", "Updated", "Completed"];
                
                for (const condition of conditions) {
                    const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                        id: generatePK(),
                        objectType: "Project",
                        objectId: generatePK(),
                        condition: condition as any,
                    });

                    const result = await notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne);
                    expect(result.condition).toBe(condition);
                }
            });

            it("allows multiple subscriptions for same object", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const objectId = generatePK();

                // Create two subscriptions for the same object with different conditions
                const input1: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK(),
                    objectType: "Project",
                    objectId: objectId,
                    condition: "CRUD",
                });

                const input2: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK(),
                    objectType: "Project",
                    objectId: objectId,
                    condition: "Updated",
                });

                const result1 = await notificationSubscription.createOne({ input: input1 }, { req, res }, notificationSubscription_createOne);
                const result2 = await notificationSubscription.createOne({ input: input2 }, { req, res }, notificationSubscription_createOne);

                expect(result1.objectId).toBe(objectId);
                expect(result2.objectId).toBe(objectId);
                expect(result1.condition).toBe("CRUD");
                expect(result2.condition).toBe("Updated");
            });
        });

        describe("invalid", () => {
            it("prevents duplicate subscription", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                
                // Create existing subscription
                const objectId = generatePK();
                await DbProvider.get().notificationSubscription.create({
                    data: NotificationSubscriptionDbFactory.createMinimal({
                        createdById: testUser[0].id,
                        objectType: "Project",
                        objectId: BigInt(objectId),
                        condition: "CRUD",
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                // Try to create duplicate
                const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK(),
                    objectType: "Project",
                    objectId: objectId,
                    condition: "CRUD",
                });

                await expect(notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid object type", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: NotificationSubscriptionCreateInput = {
                    id: generatePK(),
                    objectType: "InvalidType" as any,
                    objectId: generatePK(),
                    condition: "CRUD",
                };

                await expect(notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid condition", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: NotificationSubscriptionCreateInput = {
                    id: generatePK(),
                    objectType: "Project",
                    objectId: generatePK(),
                    condition: "InvalidCondition" as any,
                };

                await expect(notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne))
                    .rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("authentication", () => {
            it("requires authentication", async () => {
                await testEndpointRequiresAuth(
                    notificationSubscription.updateOne,
                    { id: generatePK() },
                    notificationSubscription_updateOne,
                );
            });

            it("requires API key with write permissions", async () => {
                await testEndpointRequiresApiKeyWritePermissions(
                    notificationSubscription.updateOne,
                    { id: generatePK() },
                    notificationSubscription_updateOne,
                    ["NotificationSubscription"],
                );
            });
        });

        describe("valid", () => {
            it("updates own notification subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    condition: "Updated",
                };

                const result = await notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne);

                expect(result.condition).toBe("Updated");
                expect(result.id).toBe(subscriptions[0].id.toString());

                // Verify in database
                const updated = await DbProvider.get().notificationSubscription.findUnique({
                    where: { id: subscriptions[0].id },
                });
                expect(updated?.condition).toBe("Updated");
            });

            it("updates condition type", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    condition: "Completed",
                };

                const result = await notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne);

                expect(result.condition).toBe("Completed");
            });

            it("preserves other fields when updating", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const originalObjectType = subscriptions[0].objectType;
                const originalObjectId = subscriptions[0].objectId;

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    condition: "Updated",
                };

                const result = await notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne);

                expect(result.objectType).toBe(originalObjectType);
                expect(result.objectId).toBe(originalObjectId.toString());
                expect(result.condition).toBe("Updated");
            });
        });

        describe("invalid", () => {
            it("cannot update another user's subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(), // User 2 trying to update user 1's subscription
                });

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(), // User 1's subscription
                    condition: "Updated",
                };

                await expect(notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("cannot update non-existent subscription", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: NotificationSubscriptionUpdateInput = {
                    id: generatePK(),
                    condition: "Updated",
                };

                await expect(notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects invalid condition update", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    condition: "InvalidCondition" as any,
                };

                await expect(notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne))
                    .rejects.toThrow();
            });
        });
    });
});