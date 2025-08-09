import { type FindByIdInput, type RunCreateInput, type RunSearchInput, type RunUpdateInput, generatePK, RunStatus } from "@vrooli/shared";
// eslint-disable-next-line import/extensions
import { runTestDataFactory } from "@vrooli/shared/test-fixtures/api-inputs";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertRequiresApiKeyReadPermissions, assertRequiresApiKeyWritePermissions, assertRequiresAuth } from "../../__test/authTestUtils.js";
import { mockApiSession, mockAuthenticatedSession, mockReadPrivatePermissions } from "../../__test/session.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { run_createOne } from "../generated/run_createOne.js";
import { run_findMany } from "../generated/run_findMany.js";
import { run_findOne } from "../generated/run_findOne.js";
import { run_updateOne } from "../generated/run_updateOne.js";
import { run } from "./run.js";
// Import database fixtures
import { createResourceDbFactory } from "../../__test/fixtures/db/ResourceDbFactory.js";
import { RunDbFactory } from "../../__test/fixtures/db/runFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsRun", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Clean up after each test to prevent data leakage
        await cleanupGroups.execution(DbProvider.get());

        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["run", "run_step", "run_io", "resource", "resource_version", "user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.execution(DbProvider.get());
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create test data
    async function createTestData() {
        // Create test users
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create factory instance
        const resourceFactory = createResourceDbFactory(DbProvider.get());

        // Create test projects and routines (all stored as resources)
        const project1 = await DbProvider.get().resource.create({
            data: resourceFactory.createWithVersions({
                resourceType: "Project",
                createdById: testUsers[0].id,
                isPrivate: false,
            }),
            include: { versions: true },
        });

        const project2 = await DbProvider.get().resource.create({
            data: resourceFactory.createWithVersions({
                resourceType: "Project",
                createdById: testUsers[1].id,
                isPrivate: true,
            }),
            include: { versions: true },
        });

        const routine1 = await DbProvider.get().resource.create({
            data: resourceFactory.createWithVersions({
                resourceType: "Routine",
                createdById: testUsers[0].id,
                isPrivate: false,
            }),
            include: { versions: true },
        });

        // Create test runs
        const runs = await Promise.all([
            // User 1's project run
            DbProvider.get().run.create({
                data: RunDbFactory.createWithResourceVersion(
                    project1.versions[0].id.toString(),
                    {
                        user: { connect: { id: testUsers[0].id } },
                        status: "InProgress",
                        isPrivate: false,
                    },
                ),
            }),
            // User 1's completed project run
            DbProvider.get().run.create({
                data: RunDbFactory.createWithResourceVersion(
                    project1.versions[0].id.toString(),
                    {
                        user: { connect: { id: testUsers[0].id } },
                        status: "Completed",
                        isPrivate: false,
                        completedComplexity: 10,
                        timeElapsed: 3600,
                    },
                ),
            }),
            // User 2's private project run
            DbProvider.get().run.create({
                data: RunDbFactory.createWithResourceVersion(
                    project2.versions[0].id.toString(),
                    {
                        user: { connect: { id: testUsers[1].id } },
                        status: "InProgress",
                        isPrivate: true,
                    },
                ),
            }),
            // User 1's routine run
            DbProvider.get().run.create({
                data: RunDbFactory.createWithResourceVersion(
                    routine1.versions[0].id.toString(),
                    {
                        user: { connect: { id: testUsers[0].id } },
                        status: "Scheduled",
                        isPrivate: false,
                    },
                ),
            }),
        ]);

        return { testUsers, project1, project2, routine1, runs };
    }

    describe("findOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    run.findOne,
                    { id: generatePK().toString() },
                    run_findOne,
                );
            });

            it("API key - no read permissions", async () => {
                await assertRequiresApiKeyReadPermissions(
                    run.findOne,
                    { id: generatePK().toString() },
                    run_findOne,
                    ["Run"],
                );
            });
        });

        describe("valid", () => {
            it("returns own run", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: runs[0].id.toString() };
                const result = await run.findOne({ input }, { req, res }, run_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(runs[0].id.toString());
                expect(result.status).toBe("InProgress");
                expect(result.isPrivate).toBe(false);
                expect(result.user?.id).toBe(testUsers[0].id.toString());
            });

            it("returns public run details", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[2].id.toString(), // Different user viewing public run
                });

                const input: FindByIdInput = { id: runs[0].id.toString() };
                const result = await run.findOne({ input }, { req, res }, run_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(runs[0].id.toString());
                expect(result.user?.id).toBe(testUsers[0].id.toString());
                expect(result.isPrivate).toBe(false);
            });

            it("returns completed run with results", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: runs[1].id.toString() };
                const result = await run.findOne({ input }, { req, res }, run_findOne);

                expect(result).not.toBeNull();
                expect(result.status).toBe("Completed");
                expect(result.completedComplexity).toBeGreaterThanOrEqual(0);
                expect(result.timeElapsed).toBeGreaterThanOrEqual(0);
            });

            it("returns run via API key with read permissions", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockApiSession(
                    generatePK().toString(), // apiToken
                    mockReadPrivatePermissions(), // permissions
                    {
                        id: testUsers[0].id,
                        handle: "test-user",
                        lastLoginAttempt: new Date(),
                        logInAttempts: 0,
                        name: "Test User",
                        profileImage: null,
                        publicId: "test-public-id",
                        theme: "dark",
                        status: "Unlocked" as const,
                        updatedAt: new Date(),
                        auths: [{
                            id: generatePK(),
                            provider: "Password" as const,
                            hashed_password: "dummy-hash",
                        }],
                        emails: [{
                            emailAddress: "test-user@example.com",
                        }],
                        phones: [],
                        plan: null,
                        creditAccount: null,
                        languages: ["en"],
                        sessions: [],
                    }, // userData
                );

                const input: FindByIdInput = { id: runs[0].id.toString() };
                const result = await run.findOne({ input }, { req, res }, run_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(runs[0].id.toString());
            });
        });

        describe("invalid", () => {
            it("cannot view private run of another user", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(), // User 1 trying to view user 2's private run
                });

                const input: FindByIdInput = { id: runs[2].id.toString() };
                const result = await run.findOne({ input }, { req, res }, run_findOne);

                expect(result).toBeNull();
            });

            it("returns null for non-existent run", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: generatePK().toString() };
                const result = await run.findOne({ input }, { req, res }, run_findOne);

                expect(result).toBeNull();
            });
        });
    });

    describe("findMany", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    run.findMany,
                    {},
                    run_findMany,
                );
            });

            it("API key - no read permissions", async () => {
                await assertRequiresApiKeyReadPermissions(
                    run.findMany,
                    {},
                    run_findMany,
                    ["Run"],
                );
            });
        });

        describe("valid", () => {
            it("returns only own runs", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunSearchInput = {};
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                expect(result.results).toHaveLength(3); // User 1 has 3 runs
                expect(result.totalCount).toBe(3);

                // Verify all returned runs belong to user 1
                expect(result.results.every(r => r.user?.id === testUsers[0].id.toString())).toBe(true);
            });

            it("filters by status", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunSearchInput = {
                    statuses: [RunStatus.Completed],
                };
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                expect(result.results).toHaveLength(1);
                expect(result.results[0].status).toBe("Completed");
            });

            it("filters by privacy", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunSearchInput = {
                    visibility: "Public",
                };
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                expect(result.results.every(r => !r.isPrivate)).toBe(true);
            });

            it("sorts by updated date", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunSearchInput = {
                    sortBy: "DateUpdatedDesc",
                };
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                // Verify results are sorted by update date (newest first)
                for (let i = 1; i < result.edges.length; i++) {
                    const prevDate = new Date(result.edges[i - 1].node.updated_at);
                    const currDate = new Date(result.edges[i].node.updated_at);
                    expect(prevDate.getTime()).toBeGreaterThanOrEqual(currDate.getTime());
                }
            });

            it("searches by name", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunSearchInput = {
                    searchString: "Test",
                };
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                expect(result.results.length).toBeGreaterThan(0);
                expect(result.results.some(r => r.name.includes("Test"))).toBe(true);
            });

            it("paginates results", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                // First page
                const input1: RunSearchInput = {
                    take: 2,
                    skip: 0,
                };
                const result1 = await run.findMany({ input: input1 }, { req, res }, run_findMany);

                expect(result1.edges).toHaveLength(2);
                expect(result1.pageInfo.totalCount).toBe(3);

                // Second page
                const input2: RunSearchInput = {
                    take: 2,
                    skip: 2,
                };
                const result2 = await run.findMany({ input: input2 }, { req, res }, run_findMany);

                expect(result2.edges).toHaveLength(1);
                expect(result2.pageInfo.totalCount).toBe(3);

                // Ensure no overlap
                const ids1 = result1.edges.map(edge => edge.node.id);
                const ids2 = result2.edges.map(edge => edge.node.id);
                expect(ids1.some(id => ids2.includes(id))).toBe(false);
            });

            it("filters by project", async () => {
                const { testUsers, project1 } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunSearchInput = {
                    projects: [project1.id.toString()],
                };
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                expect(result.results).toHaveLength(2); // 2 project runs for project1
                expect(result.results.every(r => r.projectVersion?.root?.id === project1.id.toString())).toBe(true);
            });

            it("filters by completion time range", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunSearchInput = {
                    minTimeElapsed: 0,
                    maxTimeElapsed: 7200, // 2 hours
                };
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                expect(result.results.every(r => r.timeElapsed <= 7200)).toBe(true);
            });
        });

        describe("invalid", () => {
            it("returns empty results for user with no runs", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[2].id.toString(), // User 3 has no runs
                });

                const input: RunSearchInput = {};
                const result = await run.findMany({ input }, { req, res }, run_findMany);

                expect(result.results).toHaveLength(0);
                expect(result.totalCount).toBe(0);
            });
        });
    });

    describe("createOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    run.createOne,
                    runTestDataFactory.createMinimal(),
                    run_createOne,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    run.createOne,
                    runTestDataFactory.createMinimal(),
                    run_createOne,
                    ["Run"],
                );
            });
        });

        describe("valid", () => {
            it("creates minimal project run", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create project for the run
                const project = await DbProvider.get().project.create({
                    data: ProjectDbFactory.createWithVersion({
                        createdById: testUser[0].id,
                        isPrivate: false,
                    }),
                    include: { versions: true },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: RunCreateInput = runTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    name: "Test Project Run",
                    status: RunStatus.InProgress,
                    isPrivate: false,
                    resourceVersionConnect: project.versions[0].id.toString(),
                });

                const result = await run.createOne({ input }, { req, res }, run_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("Run");
                expect(result.id).toBe(input.id);
                expect(result.name).toBe("Test Project Run");
                expect(result.status).toBe("InProgress");
                expect(result.user?.id).toBe(testUser[0].id.toString());
                expect(result.projectVersion?.id).toBe(project.versions[0].id.toString());

                // Verify in database
                const createdRun = await DbProvider.get().runProject.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdRun).toBeDefined();
                expect(createdRun?.name).toBe("Test Project Run");
            });

            it("creates routine run", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create routine for the run
                const routine = await DbProvider.get().resource.create({
                    data: RoutineDbFactory.createWithVersion({
                        createdById: testUser[0].id,
                        isPrivate: false,
                    }),
                    include: { versions: true },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: RunCreateInput = runTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    name: "Test Routine Run",
                    status: RunStatus.Scheduled,
                    isPrivate: false,
                    resourceVersionConnect: routine.versions[0].id.toString(),
                });

                const result = await run.createOne({ input }, { req, res }, run_createOne);

                expect(result.name).toBe("Test Routine Run");
                expect(result.status).toBe("Scheduled");
                expect(result.resourceVersion?.id).toBe(routine.versions[0].id.toString());
            });

            it("creates run with steps and IO", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                const project = await DbProvider.get().project.create({
                    data: ProjectDbFactory.createWithVersion({
                        createdById: testUser[0].id,
                    }),
                    include: { versions: true },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: RunCreateInput = runTestDataFactory.createComplete({
                    id: generatePK().toString(),
                    name: "Complete Run",
                    status: RunStatus.InProgress,
                    isPrivate: false,
                    completedComplexity: 5,
                    contextSwitches: 2,
                    timeElapsed: 3600,
                    resourceVersionConnect: project.versions[0].id.toString(),
                    data: JSON.stringify({ config: { setting: true }, progress: 50 }),
                    stepsCreate: [
                        {
                            id: generatePK().toString(),
                            name: "Step 1",
                            order: 0,
                            status: "Completed",
                            timeElapsed: 1800,
                        },
                        {
                            id: generatePK().toString(),
                            name: "Step 2",
                            order: 1,
                            status: "InProgress",
                            timeElapsed: 1800,
                        },
                    ],
                    ioCreate: [
                        {
                            id: generatePK().toString(),
                            name: "input1",
                            data: JSON.stringify({ value: "test input" }),
                        },
                        {
                            id: generatePK().toString(),
                            name: "output1",
                            data: JSON.stringify({ result: "test output" }),
                        },
                    ],
                });

                const result = await run.createOne({ input }, { req, res }, run_createOne);

                expect(result.completedComplexity).toBe(5);
                expect(result.contextSwitches).toBe(2);
                expect(result.timeElapsed).toBe(3600);
                expect(result.steps).toHaveLength(2);
                expect(result.inputs).toHaveLength(1);
                expect(result.outputs).toHaveLength(1);

                const parsedData = JSON.parse(result.data || "{}");
                expect(parsedData.config.setting).toBe(true);
                expect(parsedData.progress).toBe(50);
            });
        });

        describe("invalid", () => {
            it("rejects run for non-existent resource version", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: RunCreateInput = runTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    name: "Invalid Run",
                    status: RunStatus.InProgress,
                    resourceVersionConnect: generatePK().toString(), // Non-existent
                });

                await expect(run.createOne({ input }, { req, res }, run_createOne))
                    .rejects.toThrow();
            });

            it("rejects run for inaccessible private resource", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

                // Create private project by user 1
                const privateProject = await DbProvider.get().project.create({
                    data: ProjectDbFactory.createWithVersion({
                        createdById: testUsers[0].id,
                        isPrivate: true,
                    }),
                    include: { versions: true },
                });

                // Try to create run as user 2
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(),
                });

                const input: RunCreateInput = runTestDataFactory.createMinimal({
                    id: generatePK().toString(),
                    name: "Unauthorized Run",
                    status: RunStatus.InProgress,
                    resourceVersionConnect: privateProject.versions[0].id.toString(),
                });

                await expect(run.createOne({ input }, { req, res }, run_createOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects invalid data format", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                const resourceFactory = createResourceDbFactory(DbProvider.get());
                const project = await DbProvider.get().resource.create({
                    data: resourceFactory.createWithVersions({
                        resourceType: "Project",
                        createdById: testUser[0].id,
                        isPrivate: false,
                    }),
                    include: { versions: true },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUser[0].id.toString(),
                });

                const input: RunCreateInput = {
                    id: generatePK().toString(),
                    name: "Invalid Data Run",
                    status: RunStatus.InProgress,
                    isPrivate: false,
                    resourceVersionConnect: project.versions[0].id.toString(),
                    data: "invalid json{", // Invalid JSON
                };

                await expect(run.createOne({ input }, { req, res }, run_createOne))
                    .rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    run.updateOne,
                    { id: generatePK().toString() },
                    run_updateOne,
                );
            });

            it("API key - no write permissions", async () => {
                await assertRequiresApiKeyWritePermissions(
                    run.updateOne,
                    { id: generatePK().toString() },
                    run_updateOne,
                    ["Run"],
                );
            });
        });

        describe("valid", () => {
            it("updates own run", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunUpdateInput = {
                    id: runs[0].id.toString(),
                    status: RunStatus.Completed,
                    completedComplexity: 10,
                    timeElapsed: 7200,
                    data: JSON.stringify({ result: "success", score: 95 }),
                };

                const result = await run.updateOne({ input }, { req, res }, run_updateOne);

                expect(result.status).toBe("Completed");
                expect(result.completedComplexity).toBe(10);
                expect(result.timeElapsed).toBe(7200);

                const parsedData = JSON.parse(result.data || "{}");
                expect(parsedData.result).toBe("success");
                expect(parsedData.score).toBe(95);
            });

            it("updates run steps", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunUpdateInput = {
                    id: runs[0].id.toString(),
                    stepsCreate: [
                        {
                            id: generatePK().toString(),
                            name: "New Step",
                            order: 0,
                            status: "Completed",
                            timeElapsed: 1200,
                        },
                    ],
                };

                const result = await run.updateOne({ input }, { req, res }, run_updateOne);

                expect(result.steps).toHaveLength(1);
                expect(result.steps?.[0]?.name).toBe("New Step");
                expect(result.steps?.[0]?.status).toBe("Completed");
                expect(result.steps?.[0]?.timeElapsed).toBe(1200);
            });

            it("updates run inputs and outputs", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunUpdateInput = {
                    id: runs[0].id.toString(),
                    ioCreate: [
                        {
                            id: generatePK().toString(),
                            name: "final_result",
                            data: JSON.stringify({ value: "completed successfully" }),
                        },
                    ],
                };

                const result = await run.updateOne({ input }, { req, res }, run_updateOne);

                const outputs = result.outputs || [];
                const newOutput = outputs.find(o => o.name === "final_result");
                expect(newOutput).toBeDefined();

                const outputData = JSON.parse(newOutput?.data || "{}");
                expect(outputData.value).toBe("completed successfully");
            });

            it("marks run as failed with error details", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunUpdateInput = {
                    id: runs[0].id.toString(),
                    status: RunStatus.Failed,
                    data: JSON.stringify({
                        error: "Execution failed",
                        errorCode: "TIMEOUT",
                        timestamp: new Date().toISOString(),
                    }),
                };

                const result = await run.updateOne({ input }, { req, res }, run_updateOne);

                expect(result.status).toBe("Failed");

                const parsedData = JSON.parse(result.data || "{}");
                expect(parsedData.error).toBe("Execution failed");
                expect(parsedData.errorCode).toBe("TIMEOUT");
            });
        });

        describe("invalid", () => {
            it("cannot update another user's run", async () => {
                const { testUsers, runs } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[1].id.toString(), // User 2 trying to update user 1's run
                });

                const input: RunUpdateInput = {
                    id: runs[0].id.toString(),
                    status: RunStatus.Completed,
                };

                await expect(run.updateOne({ input }, { req, res }, run_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects update to non-existent run", async () => {
                const { testUsers } = await createTestData();
                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunUpdateInput = {
                    id: generatePK().toString(),
                    status: RunStatus.Completed,
                };

                await expect(run.updateOne({ input }, { req, res }, run_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects invalid status transition", async () => {
                const { testUsers, runs } = await createTestData();

                // Update run to completed status first
                await DbProvider.get().runProject.update({
                    where: { id: runs[1].id },
                    data: { status: "Completed" },
                });

                const { req, res } = await mockAuthenticatedSession({
                    id: testUsers[0].id.toString(),
                });

                const input: RunUpdateInput = {
                    id: runs[1].id.toString(),
                    status: RunStatus.Scheduled, // Invalid transition from Completed to Scheduled
                };

                await expect(run.updateOne({ input }, { req, res }, run_updateOne))
                    .rejects.toThrow(CustomError);
            });
        });
    });
});
