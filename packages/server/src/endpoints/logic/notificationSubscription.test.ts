import { type FindByIdInput, type NotificationSubscriptionCreateInput, type NotificationSubscriptionSearchInput, type NotificationSubscriptionUpdateInput, SubscribableObject, generatePK, nanoid } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertRequiresApiKeyReadPermissions, assertRequiresApiKeyWritePermissions, assertRequiresAuth } from "../../__test/authTestUtils.js";
import { mockApiSession, mockAuthenticatedSession, mockReadPrivatePermissions } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { notificationSubscription_createOne } from "../generated/notificationSubscription_createOne.js";
import { notificationSubscription_findMany } from "../generated/notificationSubscription_findMany.js";
import { notificationSubscription_findOne } from "../generated/notificationSubscription_findOne.js";
import { notificationSubscription_updateOne } from "../generated/notificationSubscription_updateOne.js";
import { notificationSubscription } from "./notificationSubscription.js";

// TODO: Import from @vrooli/shared when factory is properly exported
const notificationSubscriptionTestDataFactory = {
    createMinimal: (overrides?: any) => ({
        id: generatePK().toString(),
        objectType: SubscribableObject.Resource,
        objectConnect: generatePK().toString(),
        ...overrides,
    }),
};
// Import database fixtures
import { NotificationSubscriptionDbFactory } from "../../__test/fixtures/db/notificationFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsNotificationSubscription", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["notification_subscription", "resource", "team", "user_auth", "email", "session", "user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());

        // Create admin user for tests that need it
        const adminId = "00000000000000000001";
        try {
            await DbProvider.get().user.create({
                data: {
                    id: BigInt(adminId),
                    publicId: "admin-public-id",
                    handle: "admin",
                    name: "Admin User",
                    status: "Unlocked",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            });
        } catch (error) {
            // Admin user might already exist
        }
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create a user with auth for testing
    async function createUserWithAuth() {
        const { records: users } = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
        const userWithAuth = await DbProvider.get().user.findUnique({
            where: { id: users[0].id },
            include: { auths: true, sessions: true },
        });
        if (!userWithAuth) throw new Error("Failed to create user with auth");
        return userWithAuth;
    }

    // Helper function to create a resource for testing
    async function createTestResource(userId: bigint) {
        return await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: nanoid(),
                resourceType: "Code",
                isPrivate: false,
                hasCompleteVersion: false,
                isDeleted: false,
                ownedByUser: { connect: { id: userId } },
            },
        });
    }

    // Helper function to create a team for testing
    async function createTestTeam() {
        return await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: nanoid(),
                handle: `team-${nanoid(6)}`,
                isPrivate: false,
            },
        });
    }

    // Helper function to create test data
    async function createTestData() {
        // Create test users
        const { records: testUsers } = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Fetch users with auth data included for session mocking
        const usersWithAuth = await Promise.all(
            testUsers.map(user =>
                DbProvider.get().user.findUnique({
                    where: { id: user.id },
                    include: { auths: true, sessions: true },
                }),
            ),
        );

        // Create resources for subscriptions to point to
        const resource1 = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: nanoid(),
                resourceType: "Code",
                isPrivate: false,
                hasCompleteVersion: false,
                isDeleted: false,
                ownedByUser: { connect: { id: testUsers[0].id } },
            },
        });

        const resource2 = await DbProvider.get().resource.create({
            data: {
                id: generatePK(),
                publicId: nanoid(),
                resourceType: "Note",
                isPrivate: false,
                hasCompleteVersion: false,
                isDeleted: false,
                ownedByUser: { connect: { id: testUsers[0].id } },
            },
        });

        // Create a team for the third subscription
        const team = await DbProvider.get().team.create({
            data: {
                id: generatePK(),
                publicId: nanoid(),
                handle: `team-${nanoid(6)}`,
                isPrivate: false,
            },
        });

        // Create notification subscriptions for users
        const subscriptions = await Promise.all([
            // User 1 subscriptions
            DbProvider.get().notification_subscription.create({
                data: {
                    ...NotificationSubscriptionDbFactory.createMinimal(testUsers[0].id.toString()),
                    context: "Project updates subscription",
                    silent: false,
                    resource: { connect: { id: resource1.id } },
                },
                include: {
                    resource: true,
                    subscriber: true,
                },
            }),
            DbProvider.get().notification_subscription.create({
                data: {
                    ...NotificationSubscriptionDbFactory.createMinimal(testUsers[0].id.toString()),
                    context: "Routine updates subscription",
                    silent: false,
                    resource: { connect: { id: resource2.id } },
                },
                include: {
                    resource: true,
                    subscriber: true,
                },
            }),
            // User 2 subscription
            DbProvider.get().notification_subscription.create({
                data: {
                    ...NotificationSubscriptionDbFactory.createMinimal(testUsers[1].id.toString()),
                    context: "Team updates subscription",
                    silent: false,
                    team: { connect: { id: team.id } },
                },
                include: {
                    team: true,
                    subscriber: true,
                },
            }),
        ]);

        return { testUsers: usersWithAuth.filter(u => u !== null) as any[], subscriptions, resources: [resource1, resource2], team };
    }

    describe("findOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    notificationSubscription.findOne,
                    { id: generatePK().toString() },
                    notificationSubscription_findOne,
                );
            });

            it("API key - no read permissions", async () => {
                await assertRequiresApiKeyReadPermissions(
                    notificationSubscription.findOne,
                    { id: generatePK().toString() },
                    notificationSubscription_findOne,
                );
            });
        });

        describe("valid", () => {
            it("returns own notification subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: FindByIdInput = { id: subscriptions[0].id.toString() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeDefined();
                expect(result.id).toBe(subscriptions[0].id.toString());
                expect(result.context).toBe("Project updates subscription");
                expect(result.silent).toBe(false);
                expect(result.object.__typename).toBe("Resource");
            });

            it("returns subscription with API key", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const apiToken = nanoid();
                const { req, res } = await mockApiSession(
                    apiToken,
                    mockReadPrivatePermissions(["NotificationSubscription"]),
                    testUsers[0],
                );

                const input: FindByIdInput = { id: subscriptions[0].id.toString() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeDefined();
                expect(result.id).toBe(subscriptions[0].id.toString());
            });

            it("returns subscription with complete details", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: FindByIdInput = { id: subscriptions[0].id.toString() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result.context).toBe("Project updates subscription");
                expect(result.silent).toBe(false);
                expect(result.createdAt).toBeDefined();
                expect(result.object).toBeDefined();
                expect(result.object.__typename).toBe("Resource");
            });
        });

        describe("invalid", () => {
            it("cannot access another user's subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[1]); // User 2 trying to access user 1's subscription

                const input: FindByIdInput = { id: subscriptions[0].id.toString() }; // User 1's subscription
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeNull();
            });

            it("returns null for non-existent subscription", async () => {
                const testUser = await createUserWithAuth();
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: FindByIdInput = { id: generatePK().toString() };
                const result = await notificationSubscription.findOne({ input }, { req, res }, notificationSubscription_findOne);

                expect(result).toBeNull();
            });
        });
    });

    describe("findMany", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    notificationSubscription.findMany,
                    {},
                    notificationSubscription_findMany,
                );
            });

            it("API key - no read permissions", async () => {
                await assertRequiresApiKeyReadPermissions(
                    notificationSubscription.findMany,
                    {},
                    notificationSubscription_findMany,
                );
            });
        });

        describe("valid", () => {
            it("returns user's notification subscriptions", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: NotificationSubscriptionSearchInput = {};
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(2); // User 1 has 2 subscriptions
                expect(result.totalCount).toBe(2);
                expect(result.results.every(s => s.object !== null)).toBe(true);
            });

            it("filters by object type", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: NotificationSubscriptionSearchInput = {
                    objectType: SubscribableObject.Resource,
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(2); // Both user 1 subscriptions are for Resource
                expect(result.results.every(s => s.object.__typename === "Resource")).toBe(true);
            });

            it("filters by object ID", async () => {
                const { testUsers, resources } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                // Get the objectId from the first resource
                const objectId = resources[0].id.toString();
                const input: NotificationSubscriptionSearchInput = {
                    objectId,
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].object.id).toBe(objectId);
            });

            it("filters by silent flag", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: NotificationSubscriptionSearchInput = {
                    silent: false,
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results.every(s => s.silent === false)).toBe(true);
            });

            it("sorts results", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

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
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: NotificationSubscriptionSearchInput = {
                    take: 1,
                    skip: 0,
                };
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.totalCount).toBe(2); // Total available
            });

            it("returns empty results for user with no subscriptions", async () => {
                const testUser = await createUserWithAuth();
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: NotificationSubscriptionSearchInput = {};
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(0);
                expect(result.totalCount).toBe(0);
            });

            it("only returns own subscriptions", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[1]); // User 2

                const input: NotificationSubscriptionSearchInput = {};
                const result = await notificationSubscription.findMany({ input }, { req, res }, notificationSubscription_findMany);

                expect(result.results).toHaveLength(1); // User 2 has 1 subscription
                expect(result.results[0].subscriber?.id).toBe(testUsers[1].id.toString());
                expect(result.results[0].object.__typename).toBe("Team");
            });
        });
    });

    describe("createOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    notificationSubscription.createOne,
                    notificationSubscriptionTestDataFactory.createMinimal(),
                    notificationSubscription_createOne,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    notificationSubscription.createOne,
                    notificationSubscriptionTestDataFactory.createMinimal(),
                    notificationSubscription_createOne,
                );
            });
        });

        describe("valid", () => {
            it("creates notification subscription", async () => {
                const testUser = await createUserWithAuth();

                // Create a resource to subscribe to
                const resource = await createTestResource(testUser.id.toString());

                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    objectConnect: resource.id.toString(), // ID of object to subscribe to
                    objectType: SubscribableObject.Resource, // Type of object
                    context: "Project notification subscription",
                    silent: false,
                });

                const result = await notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("NotificationSubscription");
                expect(result.id).toBe(input.id);
                expect(result.context).toBe("Project notification subscription");
                expect(result.silent).toBe(false);
                expect(result.subscriber?.id).toBe(testUsers[0].id.toString());

                // Verify in database
                const created = await DbProvider.get().notification_subscription.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(created).toBeDefined();
                expect(created?.context).toBe("Project notification subscription");
                expect(created?.silent).toBe(false);
            });

            it("creates subscription with different contexts", async () => {
                const testUser = await createUserWithAuth();

                // Create resources to subscribe to
                const resources = await Promise.all(
                    ["Project updates", "Routine notifications", "Team announcements", "User messages"].map(() =>
                        createTestResource(testUser.id.toString()),
                    ),
                );

                const { req, res } = await mockAuthenticatedSession(testUser);

                const contexts = ["Project updates", "Routine notifications", "Team announcements", "User messages"];
                const results = [];

                for (let i = 0; i < contexts.length; i++) {
                    const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                        id: generatePK().toString(),
                        objectConnect: resources[i].id.toString(),
                        objectType: SubscribableObject.Resource,
                        context: contexts[i],
                        silent: false,
                    });

                    const result = await notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne);
                    results.push(result);
                    expect(result.context).toBe(contexts[i]);
                }

                expect(results).toHaveLength(4);
            });

            it("creates subscription with different silent settings", async () => {
                const testUser = await createUserWithAuth();
                const { req, res } = await mockAuthenticatedSession(testUser);

                const silentSettings = [true, false];

                for (const silent of silentSettings) {
                    const resource = await createTestResource(testUser.id.toString());
                    const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                        id: generatePK().toString(),
                        objectConnect: resource.id.toString(),
                        objectType: SubscribableObject.Resource,
                        context: "Project notifications",
                        silent,
                    });

                    const result = await notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne);
                    expect(result.silent).toBe(silent);
                }
            });

            it("allows multiple subscriptions with different contexts", async () => {
                const testUser = await createUserWithAuth();
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Create objects to subscribe to
                const resource = await createTestResource(testUser.id.toString());
                const team = await createTestTeam();

                // Create two subscriptions with different contexts
                const input1: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    objectConnect: resource.id.toString(),
                    objectType: SubscribableObject.Resource,
                    context: "Project updates",
                    silent: false,
                });

                const input2: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    objectConnect: team.id.toString(),
                    objectType: SubscribableObject.Team,
                    context: "Project comments",
                    silent: true,
                });

                const result1 = await notificationSubscription.createOne({ input: input1 }, { req, res }, notificationSubscription_createOne);
                const result2 = await notificationSubscription.createOne({ input: input2 }, { req, res }, notificationSubscription_createOne);

                expect(result1.context).toBe("Project updates");
                expect(result2.context).toBe("Project comments");
                expect(result1.silent).toBe(false);
                expect(result2.silent).toBe(true);
                expect(result1.object.__typename).toBe("Resource");
                expect(result2.object.__typename).toBe("Team");
            });
        });

        describe("invalid", () => {
            it("validates subscription input", async () => {
                const testUser = await createUserWithAuth();

                // Create existing subscription
                await DbProvider.get().notification_subscription.create({
                    data: NotificationSubscriptionDbFactory.createMinimal(
                        testUser.id.toString(),
                        {
                            context: "Existing subscription",
                            silent: false,
                        },
                    ),
                });

                const { req, res } = await mockAuthenticatedSession(testUser);

                // Create a resource to subscribe to
                const resource = await createTestResource(testUser.id.toString());

                // Try to create subscription with invalid data
                const input: NotificationSubscriptionCreateInput = notificationSubscriptionTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    objectConnect: resource.id.toString(),
                    objectType: SubscribableObject.Resource,
                    context: "", // Empty context should be invalid
                    silent: false,
                });

                await expect(notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid object type", async () => {
                const testUser = await createUserWithAuth();
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: NotificationSubscriptionCreateInput = {
                    id: generatePK().toString(),
                    objectType: "InvalidType" as any,
                    objectId: generatePK().toString(),
                    condition: "CRUD",
                };

                await expect(notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid input data", async () => {
                const testUser = await createUserWithAuth();
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: NotificationSubscriptionCreateInput = {
                    id: generatePK().toString(),
                    objectConnect: generatePK().toString(),
                    objectType: SubscribableObject.Resource,
                    context: null as any, // Invalid context
                    silent: "invalid" as any, // Invalid boolean
                };

                await expect(notificationSubscription.createOne({ input }, { req, res }, notificationSubscription_createOne))
                    .rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    notificationSubscription.updateOne,
                    { id: generatePK().toString() },
                    notificationSubscription_updateOne,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    notificationSubscription.updateOne,
                    { id: generatePK().toString() },
                    notificationSubscription_updateOne,
                );
            });
        });

        describe("valid", () => {
            it("updates own notification subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    context: "Updated subscription context",
                };

                const result = await notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne);

                expect(result.context).toBe("Updated subscription context");
                expect(result.id).toBe(subscriptions[0].id.toString());

                // Verify in database
                const updated = await DbProvider.get().notification_subscription.findUnique({
                    where: { id: subscriptions[0].id },
                });
                expect(updated?.context).toBe("Updated subscription context");
            });

            it("updates silent flag", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    silent: true,
                };

                const result = await notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne);

                expect(result.silent).toBe(true);
            });

            it("preserves other fields when updating", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const originalContext = subscriptions[0].context;
                const originalCreatedAt = subscriptions[0].createdAt;

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    silent: true,
                };

                const result = await notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne);

                expect(result.context).toBe(originalContext);
                expect(result.createdAt).toBe(originalCreatedAt);
                expect(result.silent).toBe(true);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's subscription", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[1]); // User 2 trying to update user 1's subscription

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(), // User 1's subscription
                    condition: "Updated",
                };

                await expect(notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("cannot update non-existent subscription", async () => {
                const testUser = await createUserWithAuth();
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: NotificationSubscriptionUpdateInput = {
                    id: generatePK().toString(),
                    context: "Updated context",
                };

                await expect(notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects invalid data update", async () => {
                const { testUsers, subscriptions } = await createTestData();
                const { req, res } = await mockAuthenticatedSession(testUsers[0]);

                const input: NotificationSubscriptionUpdateInput = {
                    id: subscriptions[0].id.toString(),
                    silent: "invalid" as any, // Invalid boolean
                };

                await expect(notificationSubscription.updateOne({ input }, { req, res }, notificationSubscription_updateOne))
                    .rejects.toThrow();
            });
        });
    });
});
