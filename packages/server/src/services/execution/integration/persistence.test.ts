import { describe, expect, test, beforeEach, afterEach, beforeAll, afterAll } from "vitest";
import { RunPersistenceService } from "./persistence.js";
import { DbProvider } from "../../../db/provider.js";
import { type PrismaClient, Prisma } from "@prisma/client";
import {
    generatePK,
    AccountStatus,
    type RunState,
    type StepStatus,
    type ExecutionResourceUsage,
} from "@vrooli/shared";
import crypto from "crypto";

// Test data
const testIds = {
    run1: generatePK().toString(),
    run2: generatePK().toString(),
    run3: generatePK().toString(),
    user1: generatePK().toString(),
    user2: generatePK().toString(),
    resourceVersion1: generatePK().toString(),
    resourceVersion2: generatePK().toString(),
    routine1: generatePK().toString(),
    routine2: generatePK().toString(),
    team1: generatePK().toString(),
    schedule1: generatePK().toString(),
    step1: generatePK().toString(),
    step2: generatePK().toString(),
    step3: generatePK().toString(),
};

describe("RunPersistenceService", () => {
    let service: RunPersistenceService;
    let prisma: PrismaClient;

    beforeAll(async () => {
        // Get Prisma client from provider
        prisma = DbProvider.get();
    });

    beforeEach(async () => {
        service = new RunPersistenceService();

        // Create test data that runs depend on
        await createTestData();
    });

    afterEach(async () => {
        // Clean up test data in reverse order of dependencies
        await cleanupTestData();
    });

    async function createTestData() {
        // Create users without email (email is in separate table)
        await prisma.user.createMany({
            data: [
                {
                    id: BigInt(testIds.user1),
                    publicId: testIds.user1,
                    name: "Test User 1",
                    handle: `test-user-1-${testIds.user1.slice(-8)}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
                {
                    id: BigInt(testIds.user2),
                    publicId: testIds.user2,
                    name: "Test User 2",
                    handle: `test-user-2-${testIds.user2.slice(-8)}`,
                    status: AccountStatus.Unlocked,
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            ],
            skipDuplicates: true,
        });

        // Create team
        await prisma.team.create({
            data: {
                id: BigInt(testIds.team1),
                publicId: testIds.team1,
                isOpenToNewMembers: false,
                resourceListUsedFor: [],
                translations: {
                    create: {
                        language: "en",
                        name: "Test Team",
                    },
                },
                createdBy: {
                    connect: { id: BigInt(testIds.user1) },
                },
            },
        }).catch(() => { }); // Ignore if exists

        // Create routines (resources)
        await prisma.resource.createMany({
            data: [
                {
                    id: BigInt(testIds.routine1),
                    publicId: testIds.routine1,
                    createdById: BigInt(testIds.user1),
                    resourceType: "Routine",
                    isPrivate: false,
                    isInternal: false,
                },
                {
                    id: BigInt(testIds.routine2),
                    publicId: testIds.routine2,
                    createdById: BigInt(testIds.user1),
                    resourceType: "Routine",
                    isPrivate: false,
                    isInternal: false,
                },
            ],
            skipDuplicates: true,
        });

        // Create resource versions
        await prisma.resource_version.createMany({
            data: [
                {
                    id: BigInt(testIds.resourceVersion1),
                    publicId: testIds.resourceVersion1,
                    versionIndex: 1,
                    versionLabel: "1.0.0",
                    resourceSubType: "Routine",
                    rootId: BigInt(testIds.routine1),
                    isPrivate: false,
                    isDeleted: false,
                },
                {
                    id: BigInt(testIds.resourceVersion2),
                    publicId: testIds.resourceVersion2,
                    versionIndex: 1,
                    versionLabel: "1.0.0",
                    resourceSubType: "Routine",
                    rootId: BigInt(testIds.routine2),
                    isPrivate: false,
                    isDeleted: false,
                },
            ],
            skipDuplicates: true,
        });

        // Add translations for resource versions
        await prisma.resource_translation.createMany({
            data: [
                {
                    id: BigInt(generatePK()),
                    language: "en",
                    name: "Test Routine 1",
                    resourceVersionId: BigInt(testIds.resourceVersion1),
                },
                {
                    id: BigInt(generatePK()),
                    language: "en",
                    name: "Test Routine 2",
                    resourceVersionId: BigInt(testIds.resourceVersion2),
                },
            ],
            skipDuplicates: true,
        });

        // Create schedule
        await prisma.schedule.create({
            data: {
                id: BigInt(testIds.schedule1),
                publicId: testIds.schedule1,
                timezone: "UTC",
                createdById: BigInt(testIds.user1),
            },
        }).catch(() => { }); // Ignore if exists
    }

    async function cleanupTestData() {
        // Delete in reverse order of dependencies
        await prisma.run_io.deleteMany({
            where: {
                runId: {
                    in: [BigInt(testIds.run1), BigInt(testIds.run2), BigInt(testIds.run3)],
                },
            },
        });

        await prisma.run_step.deleteMany({
            where: {
                runId: {
                    in: [BigInt(testIds.run1), BigInt(testIds.run2), BigInt(testIds.run3)],
                },
            },
        });

        await prisma.run.deleteMany({
            where: {
                id: {
                    in: [BigInt(testIds.run1), BigInt(testIds.run2), BigInt(testIds.run3)],
                },
            },
        });

        // Note: We don't clean up users, teams, etc. as they might be used by other tests
        // The test isolation should handle this at a higher level
    }

    describe("createRun", () => {
        test("should create a run with minimal data", async () => {
            const runData = {
                id: testIds.run1,
                routineId: testIds.resourceVersion1,
                userId: testIds.user1,
                inputs: { test: "value" },
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await service.createRun(runData);

            // Verify run was created
            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run1) },
                include: { io: true },
            });

            expect(run).toBeDefined();
            expect(run?.name).toBe(`Execution ${testIds.run1.slice(-8)}`);
            expect(run?.status).toBe("Scheduled");
            expect(run?.isPrivate).toBe(false);
            expect(run?.wasRunAutomatically).toBe(true);
            expect(run?.userId?.toString()).toBe(testIds.user1);
            expect(run?.resourceVersionId?.toString()).toBe(testIds.resourceVersion1);

            // Verify inputs were stored
            expect(run?.io).toHaveLength(1);
            expect(run?.io[0].nodeInputName).toBe("test");
            expect(JSON.parse(run?.io[0].data || "{}")).toBe("value");

            // Verify data field contains expected structure
            const parsedData = JSON.parse(run?.data || "{}");
            expect(parsedData.inputs).toEqual({ test: "value" });
            expect(parsedData.executionArchitecture).toBe("three-tier");
        });

        test("should create a run with full metadata", async () => {
            const runData = {
                id: testIds.run2,
                routineId: testIds.resourceVersion2,
                userId: testIds.user2,
                inputs: {
                    param1: "value1",
                    param2: { nested: true },
                },
                metadata: {
                    source: "api",
                    version: "1.0.0",
                    tags: ["test", "automated"],
                },
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await service.createRun(runData);

            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run2) },
                include: { io: true },
            });

            expect(run).toBeDefined();
            expect(run?.userId?.toString()).toBe(testIds.user2);

            // Verify metadata was stored
            const parsedData = JSON.parse(run?.data || "{}");
            expect(parsedData.metadata).toEqual({
                source: "api",
                version: "1.0.0",
                tags: ["test", "automated"],
            });

            // Verify multiple inputs
            expect(run?.io).toHaveLength(2);
            const ioByName = run?.io.reduce((acc, io) => {
                acc[io.nodeInputName] = JSON.parse(io.data);
                return acc;
            }, {} as Record<string, any>);
            expect(ioByName.param1).toBe("value1");
            expect(ioByName.param2).toEqual({ nested: true });
        });

        test("should handle non-existent resource version", async () => {
            const runData = {
                id: testIds.run3,
                routineId: "non-existent-id",
                userId: testIds.user1,
                inputs: {},
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await service.createRun(runData);

            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run3) },
            });

            expect(run).toBeDefined();
            expect(run?.resourceVersionId).toBeNull();
        });

        test("should handle creation errors", async () => {
            const runData = {
                id: testIds.run1, // Duplicate ID
                routineId: testIds.resourceVersion1,
                userId: testIds.user1,
                inputs: {},
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            // First creation should succeed
            await service.createRun(runData);

            // Second creation with same ID should fail
            await expect(service.createRun(runData)).rejects.toThrow();
        });
    });

    describe("updateRunState", () => {
        beforeEach(async () => {
            // Create a test run for state update tests
            await service.createRun({
                id: testIds.run1,
                routineId: testIds.resourceVersion1,
                userId: testIds.user1,
                inputs: {},
                createdAt: new Date(),
                updatedAt: new Date(),
            });
        });

        test("should update run state to RUNNING", async () => {
            await service.updateRunState(testIds.run1, "RUNNING");

            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run1) },
            });

            expect(run?.status).toBe("InProgress");
            expect(run?.startedAt).toBeDefined();
            expect(run?.completedAt).toBeNull();
        });

        test("should update run state to COMPLETED", async () => {
            await service.updateRunState(testIds.run1, "COMPLETED");

            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run1) },
            });

            expect(run?.status).toBe("Completed");
            expect(run?.completedAt).toBeDefined();
        });

        test("should update run state to FAILED", async () => {
            await service.updateRunState(testIds.run1, "FAILED");

            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run1) },
            });

            expect(run?.status).toBe("Failed");
            expect(run?.completedAt).toBeDefined();
        });

        test("should handle all state transitions", async () => {
            const states: RunState[] = [
                "UNINITIALIZED", "LOADING", "READY", "RUNNING",
                "PAUSED", "SUSPENDED", "COMPLETED", "FAILED", "CANCELLED",
            ];

            for (const state of states) {
                await service.updateRunState(testIds.run1, state);
                const run = await prisma.run.findUnique({
                    where: { id: BigInt(testIds.run1) },
                });

                // Verify mapping
                const expectedStatus = {
                    "UNINITIALIZED": "Scheduled",
                    "LOADING": "InProgress",
                    "READY": "InProgress",
                    "RUNNING": "InProgress",
                    "PAUSED": "InProgress",
                    "SUSPENDED": "InProgress",
                    "COMPLETED": "Completed",
                    "FAILED": "Failed",
                    "CANCELLED": "Cancelled",
                }[state];

                expect(run?.status).toBe(expectedStatus);
            }
        });

        test("should handle non-existent run", async () => {
            await expect(
                service.updateRunState("non-existent-id", "RUNNING"),
            ).rejects.toThrow();
        });
    });

    describe("recordStepExecution", () => {
        beforeEach(async () => {
            await service.createRun({
                id: testIds.run1,
                routineId: testIds.resourceVersion1,
                userId: testIds.user1,
                inputs: {},
                createdAt: new Date(),
                updatedAt: new Date(),
            });
        });

        test("should create new step record", async () => {
            const stepData = {
                stepId: testIds.step1,
                state: "running" as StepStatus["state"],
                startedAt: new Date(),
            };

            await service.recordStepExecution(testIds.run1, stepData);

            const step = await prisma.run_step.findFirst({
                where: {
                    runId: BigInt(testIds.run1),
                    nodeId: testIds.step1,
                },
            });

            expect(step).toBeDefined();
            expect(step?.name).toBe(`Step ${testIds.step1}`);
            expect(step?.status).toBe("InProgress");
            expect(step?.startedAt).toBeDefined();
            expect(step?.completedAt).toBeNull();
        });

        test("should update existing step record", async () => {
            // Create initial step
            await service.recordStepExecution(testIds.run1, {
                stepId: testIds.step1,
                state: "running",
                startedAt: new Date(),
            });

            // Update step to completed
            const completedAt = new Date();
            await service.recordStepExecution(testIds.run1, {
                stepId: testIds.step1,
                state: "completed",
                startedAt: new Date(completedAt.getTime() - 60000),
                completedAt,
                result: { output: "success" },
            });

            const step = await prisma.run_step.findFirst({
                where: {
                    runId: BigInt(testIds.run1),
                    nodeId: testIds.step1,
                },
            });

            expect(step?.status).toBe("Completed");
            expect(step?.completedAt).toBeDefined();
            expect(step?.timeElapsed).toBe(60000);
        });

        test("should handle all step states", async () => {
            const states: StepStatus["state"][] = [
                "pending", "ready", "running", "completed", "failed", "skipped",
            ];

            let index = 0;
            for (const state of states) {
                const stepId = `step-${state}-${index++}`;
                await service.recordStepExecution(testIds.run1, {
                    stepId,
                    state,
                    startedAt: new Date(),
                });

                const step = await prisma.run_step.findFirst({
                    where: {
                        runId: BigInt(testIds.run1),
                        nodeId: stepId,
                    },
                });

                const expectedStatus = {
                    "pending": "InProgress",
                    "ready": "InProgress",
                    "running": "InProgress",
                    "completed": "Completed",
                    "failed": "Skipped",
                    "skipped": "Skipped",
                }[state];

                expect(step?.status).toBe(expectedStatus);
            }
        });

        test("should estimate step complexity", async () => {
            const resourceUsage: ExecutionResourceUsage = {
                cpuPercent: 50,
                memoryUsedMB: 150,
                durationMs: 45000,
                toolCalls: 10,
                llmTokensUsed: 1000,
            };

            await service.recordStepExecution(testIds.run1, {
                stepId: testIds.step2,
                state: "completed",
                startedAt: new Date(),
                completedAt: new Date(),
                resourceUsage,
            });

            const step = await prisma.run_step.findFirst({
                where: {
                    runId: BigInt(testIds.run1),
                    nodeId: testIds.step2,
                },
            });

            // Complexity should be increased due to high resource usage
            expect(step?.complexity).toBeGreaterThan(1);
            expect(step?.complexity).toBeLessThanOrEqual(10);
        });
    });

    describe("loadRun", () => {
        beforeEach(async () => {
            // Create a complete run with steps and I/O
            await service.createRun({
                id: testIds.run1,
                routineId: testIds.resourceVersion1,
                userId: testIds.user1,
                inputs: { param1: "test" },
                metadata: { source: "test" },
                createdAt: new Date(),
                updatedAt: new Date(),
            });

            // Add some steps
            await service.recordStepExecution(testIds.run1, {
                stepId: testIds.step1,
                state: "completed",
                startedAt: new Date(),
                completedAt: new Date(),
            });
        });

        test("should load run with all data", async () => {
            const loadedRun = await service.loadRun(testIds.run1);

            expect(loadedRun).toBeDefined();
            expect(loadedRun?.id).toBe(testIds.run1);
            expect(loadedRun?.routineId).toBe(testIds.resourceVersion1);
            expect(loadedRun?.userId).toBe(testIds.user1);
            expect(loadedRun?.status).toBe("Scheduled");
            expect(loadedRun?.inputs).toEqual({ param1: "test" });
            expect(loadedRun?.metadata).toEqual({ source: "test" });
        });

        test("should return null for non-existent run", async () => {
            const loadedRun = await service.loadRun("non-existent-id");
            expect(loadedRun).toBeNull();
        });

        test("should handle corrupted JSON data gracefully", async () => {
            // Manually corrupt the data field
            await prisma.run.update({
                where: { id: BigInt(testIds.run1) },
                data: { data: "invalid-json{" },
            });

            const loadedRun = await service.loadRun(testIds.run1);

            expect(loadedRun).toBeDefined();
            expect(loadedRun?.inputs).toEqual({ param1: "test" }); // From IO records
            expect(loadedRun?.outputs).toEqual({});
            expect(loadedRun?.metadata).toEqual({});
        });
    });

    describe("getUserRunHistory", () => {
        beforeEach(async () => {
            // Create multiple runs for history
            const runIds = [testIds.run1, testIds.run2, testIds.run3];
            for (let i = 0; i < runIds.length; i++) {
                await service.createRun({
                    id: runIds[i],
                    routineId: i % 2 === 0 ? testIds.resourceVersion1 : testIds.resourceVersion2,
                    userId: testIds.user1,
                    inputs: {},
                    createdAt: new Date(Date.now() - i * 1000000), // Stagger creation times
                    updatedAt: new Date(),
                });

                // Complete some runs
                if (i > 0) {
                    await service.updateRunState(runIds[i], "COMPLETED");
                }
            }
        });

        test("should get user run history with default limit", async () => {
            const history = await service.getUserRunHistory(testIds.user1);

            expect(history).toHaveLength(3);
            expect(history[0].id).toBe(testIds.run1); // Most recent first
            expect(history[0].routineName).toBe("Test Routine 1");
            expect(history[0].status).toBe("Scheduled");
            expect(history[1].status).toBe("Completed");
        });

        test("should respect limit and offset", async () => {
            const history = await service.getUserRunHistory(testIds.user1, 2, 1);

            expect(history).toHaveLength(2);
            expect(history[0].id).toBe(testIds.run2);
        });

        test("should calculate duration for completed runs", async () => {
            // Update a run with start and complete times
            await prisma.run.update({
                where: { id: BigInt(testIds.run2) },
                data: {
                    startedAt: new Date(Date.now() - 300000), // 5 minutes ago
                    completedAt: new Date(Date.now() - 60000), // 1 minute ago
                },
            });

            const history = await service.getUserRunHistory(testIds.user1);
            const run2 = history.find(r => r.id === testIds.run2);

            expect(run2?.duration).toBe(240000); // 4 minutes
        });

        test("should return empty array for user with no runs", async () => {
            const history = await service.getUserRunHistory(testIds.user2);
            expect(history).toEqual([]);
        });
    });

    describe("updateRunOutputs", () => {
        beforeEach(async () => {
            await service.createRun({
                id: testIds.run1,
                routineId: testIds.resourceVersion1,
                userId: testIds.user1,
                inputs: { input: "test" },
                metadata: { version: "1.0" },
                createdAt: new Date(),
                updatedAt: new Date(),
            });
        });

        test("should update run outputs", async () => {
            const outputs = {
                result: "success",
                data: { processed: true, count: 42 },
                errors: [],
            };

            await service.updateRunOutputs(testIds.run1, outputs);

            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run1) },
            });

            const parsedData = JSON.parse(run?.data || "{}");
            expect(parsedData.outputs).toEqual(outputs);
            expect(parsedData.inputs).toEqual({ input: "test" }); // Preserved
            expect(parsedData.metadata).toEqual({ version: "1.0" }); // Preserved
            expect(run?.completedAt).toBeDefined();
        });

        test("should handle complex nested outputs", async () => {
            const complexOutputs = {
                nested: {
                    deeply: {
                        nested: {
                            value: "found",
                            array: [1, 2, { three: 3 }],
                        },
                    },
                },
                bigNumber: 9007199254740991n.toString(), // BigInt as string
                date: new Date().toISOString(),
            };

            await service.updateRunOutputs(testIds.run1, complexOutputs);

            const run = await prisma.run.findUnique({
                where: { id: BigInt(testIds.run1) },
            });

            const parsedData = JSON.parse(run?.data || "{}");
            expect(parsedData.outputs).toEqual(complexOutputs);
        });

        test("should throw error for non-existent run", async () => {
            await expect(
                service.updateRunOutputs("non-existent-id", { result: "fail" }),
            ).rejects.toThrow("Run not found");
        });
    });

    describe("error handling", () => {
        test("should handle database connection errors", async () => {
            // This would typically involve mocking DbProvider.get() to throw
            // For now, we'll test with an invalid operation
            const service = new RunPersistenceService();

            // Try to create a run with an invalid user ID that doesn't exist
            const runData = {
                id: generatePK().toString(),
                routineId: testIds.resourceVersion1,
                userId: "999999999999999999", // Non-existent user
                inputs: {},
                createdAt: new Date(),
                updatedAt: new Date(),
            };

            await expect(service.createRun(runData)).rejects.toThrow();
        });

        test("should handle BigInt conversion errors", async () => {
            // Test with invalid ID that can't be converted to BigInt
            await expect(
                service.updateRunState("not-a-number", "RUNNING"),
            ).rejects.toThrow();
        });
    });

    describe("estimateStepComplexity", () => {
        beforeEach(async () => {
            await service.createRun({
                id: testIds.run1,
                routineId: testIds.resourceVersion1,
                userId: testIds.user1,
                inputs: {},
                createdAt: new Date(),
                updatedAt: new Date(),
            });
        });

        test("should calculate complexity based on resource usage", async () => {
            const testCases = [
                {
                    usage: { cpuPercent: 10, memoryUsedMB: 50, durationMs: 1000, toolCalls: 1, llmTokensUsed: 100 },
                    minComplexity: 1,
                },
                {
                    usage: { cpuPercent: 80, memoryUsedMB: 200, durationMs: 60000, toolCalls: 10, llmTokensUsed: 5000 },
                    minComplexity: 3,
                },
            ];

            for (const testCase of testCases) {
                const stepId = generatePK();
                await service.recordStepExecution(testIds.run1, {
                    stepId,
                    state: "completed",
                    startedAt: new Date(),
                    completedAt: new Date(),
                    resourceUsage: testCase.usage,
                });

                const step = await prisma.run_step.findFirst({
                    where: { runId: BigInt(testIds.run1), nodeId: stepId)
            },
        });

        expect(step?.complexity).toBeGreaterThanOrEqual(testCase.minComplexity);
        expect(step?.complexity).toBeLessThanOrEqual(10);
    }
        });
    });
});
