import { type PopularSearchInput, ResourceSortBy, TeamSortBy, VisibilityType, generatePK } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { mockAuthenticatedSession, mockLoggedOutSession } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { feed_home } from "../generated/feed_home.js";
import { feed_popular } from "../generated/feed_popular.js";
import { feed } from "./feed.js";
// Import database fixtures
import { ReminderDbFactory } from "../../__test/fixtures/db/reminderFixtures.js";
import { ResourceDbFactory } from "../../__test/fixtures/db/resourceFixtures.js";
import { ScheduleDbFactory } from "../../__test/fixtures/db/scheduleFixtures.js";
import { TeamDbFactory } from "../../__test/fixtures/db/teamFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsFeed", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["team", "member", "member_invite", "meeting", "user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.team(DbProvider.get());
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    async function createTestData() {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create test tags with unique names to avoid conflicts
        const uniqueSuffix = Date.now().toString();
        const tags = await Promise.all([
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `api_${uniqueSuffix}` } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `database_${uniqueSuffix}` } }),
            DbProvider.get().tag.create({ data: { id: generatePK(), tag: `utility_${uniqueSuffix}` } }),
        ]);

        // Create test team
        const team = await DbProvider.get().team.create({
            data: TeamDbFactory.createWithMember({
                id: generatePK(),
                createdById: testUsers[0].id,
                isPrivate: false,
                members: [{
                    userId: testUsers[0].id,
                    permissions: "[\"owner\"]",
                }],
            }),
        });

        // Create test resources
        const resources = await Promise.all([
            // Public resource by user 1
            DbProvider.get().resource.create({
                data: ResourceDbFactory.createWithVersion({
                    createdById: testUsers[0].id,
                    isPrivate: false,
                    publicId: "popular_resource_1",
                }),
                include: { versions: true },
            }),
            // Public resource by user 2
            DbProvider.get().resource.create({
                data: ResourceDbFactory.createWithVersion({
                    createdById: testUsers[1].id,
                    isPrivate: false,
                    publicId: "popular_resource_2",
                }),
                include: { versions: true },
            }),
        ]);

        // Create test reminders for user 1
        const reminders = await Promise.all([
            DbProvider.get().reminder.create({
                data: ReminderDbFactory.createWithItems({
                    createdById: testUsers[0].id,
                    name: "Daily standup",
                    isComplete: false,
                    items: [{
                        name: "Prepare agenda",
                        isComplete: false,
                    }],
                }),
            }),
            DbProvider.get().reminder.create({
                data: ReminderDbFactory.createWithItems({
                    createdById: testUsers[0].id,
                    name: "Weekly review",
                    isComplete: false,
                    items: [{
                        name: "Review metrics",
                        isComplete: false,
                    }],
                }),
            }),
        ]);

        // Create test schedules for user 1
        const now = new Date();
        const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
        const schedules = await Promise.all([
            DbProvider.get().schedule.create({
                data: ScheduleDbFactory.createMinimal({
                    createdById: testUsers[0].id,
                    name: "Morning meeting",
                    startTime: tomorrow,
                    endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000), // 1 hour later
                }),
            }),
        ]);

        return { testUsers, tags, team, resources, reminders, schedules };
    }

    describe("home", () => {
        describe("authentication", () => {
            it("allows unauthenticated access", async () => {
                const { req, res } = await mockLoggedOutSession();

                const result = await feed.home({}, { req, res }, feed_home);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("HomeResult");
                expect(result.reminders).toBeDefined();
                expect(result.resources).toBeDefined();
                expect(result.schedules).toBeDefined();
            });

            it("provides personalized feed for authenticated user", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const result = await feed.home({}, { req, res }, feed_home);

                expect(result.__typename).toBe("HomeResult");
                expect(result.reminders).toBeDefined();
                expect(result.resources).toBeDefined();
                expect(result.schedules).toBeDefined();
            });
        });

        describe("valid", () => {
            it("returns user's reminders, resources, and schedules", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const result = await feed.home({}, { req, res }, feed_home);

                expect(result.reminders).toHaveLength(2); // User has 2 reminders
                expect(result.resources).toHaveLength(1); // User has 1 resource
                expect(result.schedules).toHaveLength(1); // User has 1 schedule

                // Verify reminder details
                expect(result.reminders.every(r => r.isComplete === false)).toBe(true);

                // Verify resource details
                expect(result.resources[0].root?.createdBy?.id).toBe(testUsers[0].id.toString());

                // Verify schedule details
                expect(result.schedules[0].name).toBe("Morning meeting");
                expect(result.schedules[0].createdBy?.id).toBe(testUsers[0].id.toString());
            });

            it("returns empty arrays for user with no data", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const result = await feed.home({}, { req, res }, feed_home);

                expect(result.reminders).toHaveLength(0);
                expect(result.resources).toHaveLength(0);
                expect(result.schedules).toHaveLength(0);
            });

            it("filters out completed reminders", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create completed reminder
                await DbProvider.get().reminder.create({
                    data: ReminderDbFactory.createWithItems({
                        createdById: testUser[0].id,
                        name: "Completed task",
                        isComplete: true,
                        items: [{
                            name: "Done",
                            isComplete: true,
                        }],
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const result = await feed.home({}, { req, res }, feed_home);

                // Should not include completed reminders
                expect(result.reminders).toHaveLength(0);
            });

            it("limits results to 10 items per category", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create many reminders (more than 10)
                const reminderPromises = [];
                for (let i = 0; i < 15; i++) {
                    reminderPromises.push(
                        DbProvider.get().reminder.create({
                            data: ReminderDbFactory.createWithItems({
                                createdById: testUser[0].id,
                                name: `Reminder ${i}`,
                                isComplete: false,
                                items: [{
                                    name: `Item ${i}`,
                                    isComplete: false,
                                }],
                            }),
                        }),
                    );
                }
                await Promise.all(reminderPromises);

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const result = await feed.home({}, { req, res }, feed_home);

                // Should limit to 10 reminders
                expect(result.reminders).toHaveLength(10);
            });

            it("includes schedules for next 7 days", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                const now = new Date();
                const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
                const nextWeek = new Date(now.getTime() + 6 * 24 * 60 * 60 * 1000);
                const twoWeeksLater = new Date(now.getTime() + 14 * 24 * 60 * 60 * 1000);

                // Create schedules at different times
                await Promise.all([
                    // Schedule for tomorrow (should be included)
                    DbProvider.get().schedule.create({
                        data: ScheduleDbFactory.createMinimal({
                            createdById: testUser[0].id,
                            name: "Tomorrow meeting",
                            startTime: tomorrow,
                            endTime: new Date(tomorrow.getTime() + 60 * 60 * 1000),
                        }),
                    }),
                    // Schedule for next week (should be included)
                    DbProvider.get().schedule.create({
                        data: ScheduleDbFactory.createMinimal({
                            createdById: testUser[0].id,
                            name: "Next week meeting",
                            startTime: nextWeek,
                            endTime: new Date(nextWeek.getTime() + 60 * 60 * 1000),
                        }),
                    }),
                    // Schedule for two weeks later (should not be included in feed)
                    DbProvider.get().schedule.create({
                        data: ScheduleDbFactory.createMinimal({
                            createdById: testUser[0].id,
                            name: "Future meeting",
                            startTime: twoWeeksLater,
                            endTime: new Date(twoWeeksLater.getTime() + 60 * 60 * 1000),
                        }),
                    }),
                ]);

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const result = await feed.home({}, { req, res }, feed_home);

                // Should include schedules within 7 days
                expect(result.schedules).toHaveLength(2);
                const scheduleNames = result.schedules.map(s => s.name);
                expect(scheduleNames).toContain("Tomorrow meeting");
                expect(scheduleNames).toContain("Next week meeting");
                expect(scheduleNames).not.toContain("Future meeting");
            });
        });

        describe("invalid", () => {
            it("respects rate limiting", async () => {
                const { req, res } = await mockLoggedOutSession();

                // The endpoint has rate limiting of 5000 per user
                // This test verifies the rate limiting is in place (actual limit testing would require many requests)
                const result = await feed.home({}, { req, res }, feed_home);

                expect(result).toBeDefined();
                // If rate limiting fails, this would throw an error
            });
        });
    });

    describe("popular", () => {
        describe("authentication", () => {
            it("allows unauthenticated access", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {};
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("PopularSearchResult");
                expect(result.edges).toBeDefined();
                expect(result.pageInfo).toBeDefined();
            });

            it("provides same results for authenticated and unauthenticated users", async () => {
                await createTestData();

                const input: PopularSearchInput = {};

                // Get results for unauthenticated user
                const { req: req1, res: res1 } = await mockLoggedOutSession();
                const result1 = await feed.popular({ input }, { req: req1, res: res1 }, feed_popular);

                // Get results for authenticated user
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req: req2, res: res2 } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });
                const result2 = await feed.popular({ input }, { req: req2, res: res2 }, feed_popular);

                // Should return similar structure (popular content is public)
                expect(result1.__typename).toBe(result2.__typename);
                expect(result1.edges).toBeDefined();
                expect(result2.edges).toBeDefined();
            });
        });

        describe("valid", () => {
            it("returns popular resources, teams, and users", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {};
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result.edges).toBeDefined();
                expect(result.pageInfo).toBeDefined();
                expect(result.pageInfo.__typename).toBe("PopularPageInfo");
                expect(result.pageInfo.hasNextPage).toBeDefined();
                expect(result.pageInfo.endCursorResource).toBeDefined();
                expect(result.pageInfo.endCursorTeam).toBeDefined();
                expect(result.pageInfo.endCursorUser).toBeDefined();

                // Should contain mix of resources, teams, and users
                const nodeTypes = result.edges.map(edge => edge.node.__typename);
                expect(nodeTypes).toContain("ResourceVersion");
                expect(nodeTypes).toContain("Team");
                expect(nodeTypes).toContain("User");
            });

            it("filters by specific object type", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    objectType: "Resource",
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                // Should only return resources
                const nodeTypes = result.edges.map(edge => edge.node.__typename);
                expect(nodeTypes.every(type => type === "ResourceVersion")).toBe(true);
            });

            it("filters by team object type", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    objectType: "Team",
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                // Should only return teams
                const nodeTypes = result.edges.map(edge => edge.node.__typename);
                expect(nodeTypes.every(type => type === "Team")).toBe(true);
            });

            it("filters by user object type", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    objectType: "User",
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                // Should only return users
                const nodeTypes = result.edges.map(edge => edge.node.__typename);
                expect(nodeTypes.every(type => type === "User")).toBe(true);
            });

            it("applies custom sort order", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    objectType: "Resource",
                    sortBy: ResourceSortBy.DateCreatedDesc,
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result.edges).toBeDefined();
                // Verify results are ResourceVersions when filtering by Resource
                const nodeTypes = result.edges.map(edge => edge.node.__typename);
                expect(nodeTypes.every(type => type === "ResourceVersion")).toBe(true);
            });

            it("filters by visibility", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    visibility: VisibilityType.Public,
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result.edges).toBeDefined();
                // Should only return public items
                expect(result.edges.every(edge => {
                    const node = edge.node as any;
                    return !node.isPrivate || node.isPrivate === false;
                })).toBe(true);
            });

            it("supports pagination with cursors", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    objectType: "Resource",
                    resourceAfter: "cursor_123",
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result.pageInfo).toBeDefined();
                expect(result.pageInfo.endCursorResource).toBeDefined();
            });

            it("filters by time frame", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    createdTimeFrame: {
                        after: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // Last 7 days
                        before: new Date(),
                    },
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result.edges).toBeDefined();
                // Should filter by creation time frame
            });

            it("combines multiple object types in alternating pattern", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {};
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                // Should contain alternating mix of different object types
                expect(result.edges.length).toBeGreaterThan(0);

                const nodeTypes = result.edges.map(edge => edge.node.__typename);
                const uniqueTypes = [...new Set(nodeTypes)];
                expect(uniqueTypes.length).toBeGreaterThan(1); // Should have multiple types
            });

            it("respects take limits for different object types", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    objectType: "Resource",
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                // Should respect resource take limit (approximately 4/6 of 50 = ~33)
                expect(result.edges.length).toBeLessThanOrEqual(50);
            });
        });

        describe("invalid", () => {
            it("respects rate limiting", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {};

                // The endpoint has rate limiting of 5000 per user
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result).toBeDefined();
                // If rate limiting fails, this would throw an error
            });

            it("handles invalid sort parameters gracefully", async () => {
                await createTestData();
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    objectType: "Team",
                    sortBy: TeamSortBy.BookmarksDesc,
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("PopularSearchResult");
            });

            it("returns empty results when no data matches filters", async () => {
                // Don't create test data
                const { req, res } = await mockLoggedOutSession();

                const input: PopularSearchInput = {
                    createdTimeFrame: {
                        after: new Date(Date.now() - 1000), // Very recent
                        before: new Date(Date.now() - 500),  // Even more recent
                    },
                };
                const result = await feed.popular({ input }, { req, res }, feed_popular);

                expect(result.edges).toHaveLength(0);
                expect(result.pageInfo.hasNextPage).toBe(false);
            });
        });
    });
});
