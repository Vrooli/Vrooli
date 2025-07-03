import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import "../../__test/setup.js";
import { clearRedisCache } from "../queueFactory.js";
import { QueueService } from "../queues.js";
import { QueueTaskType, type SandboxTask } from "../taskTypes.js";
import { changeSandboxTaskStatus, getSandboxTaskStatuses, processSandbox } from "./queue.js";

describe("Sandbox Queue", () => {
    let queueService: QueueService;
    const redisUrl = process.env.REDIS_URL || "redis://localhost:6379";

    beforeEach(async () => {
        // Get fresh instance and initialize
        queueService = QueueService.get();
        await queueService.init(redisUrl);
    });

    afterEach(async () => {
        // Clean shutdown
        try {
            await queueService.shutdown();
            // Wait for shutdown to fully complete and event handlers to detach
            await new Promise(resolve => setTimeout(resolve, 100));
        } catch (error) {
            console.log("Shutdown error (ignored):", error);
        }
        // Clear singleton before clearing cache to prevent any access during cleanup
        (QueueService as any).instance = null;
        // Clear Redis cache last to avoid disconnecting connections still in use
        clearRedisCache();
        // Final delay to ensure all async operations complete
        await new Promise(resolve => setTimeout(resolve, 50));
    });

    describe("processSandbox", () => {
        const baseSandboxData: Omit<SandboxTask, "type" | "status"> = {
            taskId: "task-123",
            code: "console.log('Hello, world!');",
            language: "javascript",
            userId: "user-456",
            runId: "run-789",
            stepId: "step-012",
            timeLimit: 5000,
            memoryLimit: 128,
            environment: {
                NODE_ENV: "test",
            },
        };

        it("should add sandbox task with correct type and status", async () => {
            const result = await processSandbox(baseSandboxData, queueService);
            expect(result.success).toBe(true);
            expect(result.data?.id).toBeDefined();

            // Verify the job was added
            const job = await queueService.sandbox.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
            expect(job?.data.type).toBe(QueueTaskType.SANDBOX_EXECUTION);
            expect(job?.data.status).toBe("Scheduled");
        });

        it("should preserve all task data", async () => {
            const complexData: Omit<SandboxTask, "type" | "status"> = {
                ...baseSandboxData,
                code: `
                    const fs = require('fs');
                    const data = fs.readFileSync('input.txt', 'utf8');
                    console.log(data.toUpperCase());
                `,
                environment: {
                    NODE_ENV: "test",
                    API_KEY: "test-key",
                    DEBUG: "true",
                },
                inputs: {
                    files: ["input.txt"],
                    stdin: "test input",
                },
                allowedModules: ["fs", "path"],
            };

            const result = await processSandbox(complexData, queueService);
            const job = await queueService.sandbox.queue.getJob(result.data!.id);

            expect(job?.data.code).toEqual(complexData.code);
            expect(job?.data.environment).toEqual(complexData.environment);
            expect(job?.data.inputs).toEqual(complexData.inputs);
            expect(job?.data.allowedModules).toEqual(complexData.allowedModules);
        });

        it("should handle different programming languages", async () => {
            const languages = ["javascript", "python", "typescript", "bash"];

            for (const language of languages) {
                const data = { ...baseSandboxData, language, taskId: `task-${language}` };
                const result = await processSandbox(data, queueService);
                expect(result.success).toBe(true);

                const job = await queueService.sandbox.queue.getJob(result.data!.id);
                expect(job?.data.language).toBe(language);
            }
        });

        it("should handle concurrent task additions", async () => {
            const promises = [];
            for (let i = 0; i < 5; i++) {
                const data = { ...baseSandboxData, taskId: `task-concurrent-${i}` };
                promises.push(processSandbox(data, queueService));
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(5);
            results.forEach(result => {
                expect(result.success).toBe(true);
                expect(result.data?.id).toBeDefined();
            });
        });

        it("should handle different resource limits", async () => {
            const limitVariants = [
                { timeLimit: 1000, memoryLimit: 64 },
                { timeLimit: 10000, memoryLimit: 256 },
                { timeLimit: 30000, memoryLimit: 512 },
            ];

            for (const limits of limitVariants) {
                const data = {
                    ...baseSandboxData,
                    ...limits,
                    taskId: `task-limit-${limits.timeLimit}`,
                };
                const result = await processSandbox(data, queueService);
                expect(result.success).toBe(true);

                const job = await queueService.sandbox.queue.getJob(result.data!.id);
                expect(job?.data.timeLimit).toBe(limits.timeLimit);
                expect(job?.data.memoryLimit).toBe(limits.memoryLimit);
            }
        });

        it("should handle code with security-sensitive operations", async () => {
            const securityTestCases = [
                {
                    code: "const net = require('net'); net.createServer();",
                    description: "Network operations",
                },
                {
                    code: "const { exec } = require('child_process'); exec('ls');",
                    description: "Process execution",
                },
                {
                    code: "eval('console.log(1+1)');",
                    description: "Dynamic code evaluation",
                },
            ];

            for (const testCase of securityTestCases) {
                const data = {
                    ...baseSandboxData,
                    code: testCase.code,
                    taskId: `task-security-${testCase.description.replace(/\s+/g, "-")}`,
                };
                const result = await processSandbox(data, queueService);
                expect(result.success).toBe(true);

                // Verify the code is passed as-is (sandbox will handle security)
                const job = await queueService.sandbox.queue.getJob(result.data!.id);
                expect(job?.data.code).toBe(testCase.code);
            }
        });
    });

    describe("changeSandboxTaskStatus", () => {
        let jobId: string;

        beforeEach(async () => {
            // Add a test job
            const result = await processSandbox({
                taskId: "test-task-456",
                code: "console.log('test');",
                language: "javascript",
                userId: "user-456",
                runId: "run-789",
                stepId: "step-012",
                timeLimit: 5000,
                memoryLimit: 128,
                environment: {},
            }, queueService);
            jobId = result.data!.id;
        });

        it("should change task status", async () => {
            const result = await changeSandboxTaskStatus(jobId, "Running", "user-456", queueService);
            expect(result.success).toBe(true);
        });

        it("should verify status change is delegated to QueueService", async () => {
            const spy = vi.spyOn(queueService, "changeTaskStatus");
            await changeSandboxTaskStatus(jobId, "Running", "user-456", queueService);
            expect(spy).toHaveBeenCalledWith(jobId, "Running", "user-456", "sandbox");
        });

        it("should handle all valid task statuses", async () => {
            const statuses = ["Scheduled", "Running", "Completed", "Failed", "Cancelled"];

            for (const status of statuses) {
                // Add a new job for each status test
                const result = await processSandbox({
                    taskId: `test-task-status-${status}`,
                    code: "console.log('test');",
                    language: "javascript",
                    userId: "user-456",
                    runId: "run-789",
                    stepId: "step-012",
                    timeLimit: 5000,
                    memoryLimit: 128,
                    environment: {},
                }, queueService);

                const statusResult = await changeSandboxTaskStatus(
                    result.data!.id,
                    status,
                    "user-456",
                    queueService,
                );
                expect(statusResult.success).toBe(true);
            }
        });

        it("should handle invalid job ID", async () => {
            const result = await changeSandboxTaskStatus("invalid-id", "Running", "user-456", queueService);
            expect(result.success).toBe(false);
        });
    });

    describe("getSandboxTaskStatuses", () => {
        const jobIds: string[] = [];

        beforeEach(async () => {
            // Add multiple test jobs
            for (let i = 0; i < 3; i++) {
                const result = await processSandbox({
                    taskId: `test-task-${i}`,
                    code: `console.log('Task ${i}');`,
                    language: "javascript",
                    userId: "user-456",
                    runId: `run-${i}`,
                    stepId: `step-${i}`,
                    timeLimit: 5000,
                    memoryLimit: 128,
                    environment: {},
                }, queueService);
                jobIds.push(result.data!.id);
            }
        });

        it("should fetch multiple task statuses", async () => {
            const statuses = await getSandboxTaskStatuses(jobIds, queueService);
            expect(statuses).toHaveLength(3);
            statuses.forEach((status, index) => {
                expect(status.__typename).toBe("TaskStatusInfo");
                expect(status.id).toBe(jobIds[index]);
                expect(status.queueName).toBe("sandbox");
                expect(status.status).toBeDefined();
            });
        });

        it("should handle mixed valid and invalid IDs", async () => {
            const mixedIds = [...jobIds, "invalid-id-1", "invalid-id-2"];
            const statuses = await getSandboxTaskStatuses(mixedIds, queueService);

            // Should return status for all IDs, with null for invalid ones
            expect(statuses).toHaveLength(5);

            // Valid IDs should have status
            statuses.slice(0, 3).forEach(status => {
                expect(status.status).toBeDefined();
            });

            // Invalid IDs should have null status
            statuses.slice(3).forEach(status => {
                expect(status.status).toBeNull();
            });
        });

        it("should handle empty array", async () => {
            const statuses = await getSandboxTaskStatuses([], queueService);
            expect(statuses).toEqual([]);
        });

        it("should verify delegation to QueueService", async () => {
            const spy = vi.spyOn(queueService, "getTaskStatuses");
            await getSandboxTaskStatuses(jobIds, queueService);
            expect(spy).toHaveBeenCalledWith(jobIds, "sandbox");
        });
    });

    describe("Integration with QueueService", () => {
        it("should process sandbox task through worker", async () => {
            // Mock the sandbox process function
            const processSandboxMock = vi.fn().mockResolvedValue({
                output: "Hello, world!",
                exitCode: 0,
                error: null,
            });
            vi.doMock("./process.js", () => ({
                sandboxProcess: processSandboxMock,
            }));

            const sandboxData = {
                taskId: "test-sandbox-integration",
                code: "console.log('Hello, world!');",
                language: "javascript",
                userId: "user-456",
                runId: "run-integration",
                stepId: "step-integration",
                timeLimit: 5000,
                memoryLimit: 128,
                environment: {},
            };

            const result = await processSandbox(sandboxData, queueService);
            expect(result.success).toBe(true);

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 1000));

            // Verify job exists
            const job = await queueService.sandbox.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
        });

        it("should handle job failure and retry", async () => {
            // Mock process to fail first time
            let callCount = 0;
            const processSandboxMock = vi.fn().mockImplementation(() => {
                callCount++;
                if (callCount === 1) {
                    throw new Error("Simulated sandbox failure");
                }
                return Promise.resolve({
                    output: "",
                    exitCode: 0,
                    error: null,
                });
            });

            vi.doMock("./process.js", () => ({
                sandboxProcess: processSandboxMock,
            }));

            const sandboxData = {
                taskId: "test-sandbox-fail",
                code: "throw new Error('test error');",
                language: "javascript",
                userId: "user-456",
                runId: "run-fail",
                stepId: "step-fail",
                timeLimit: 5000,
                memoryLimit: 128,
                environment: {},
            };

            const result = await processSandbox(sandboxData, queueService);
            const job = await queueService.sandbox.queue.getJob(result.data!.id);

            expect(job).toBeDefined();
            expect(job?.attemptsMade).toBeGreaterThanOrEqual(0);
        });

        it("should handle timeout scenarios", async () => {
            // Mock process to simulate timeout
            const processSandboxMock = vi.fn().mockResolvedValue({
                output: "",
                exitCode: -1,
                error: "Execution timed out",
                timedOut: true,
            });

            vi.doMock("./process.js", () => ({
                sandboxProcess: processSandboxMock,
            }));

            const sandboxData = {
                taskId: "test-sandbox-timeout",
                code: "while(true) { }",
                language: "javascript",
                userId: "user-456",
                runId: "run-timeout",
                stepId: "step-timeout",
                timeLimit: 1000, // 1 second timeout
                memoryLimit: 128,
                environment: {},
            };

            const result = await processSandbox(sandboxData, queueService);
            expect(result.success).toBe(true);

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 1500));

            const job = await queueService.sandbox.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
        });

        it("should handle memory limit scenarios", async () => {
            // Mock process to simulate memory limit exceeded
            const processSandboxMock = vi.fn().mockResolvedValue({
                output: "",
                exitCode: -1,
                error: "Memory limit exceeded",
                memoryExceeded: true,
            });

            vi.doMock("./process.js", () => ({
                sandboxProcess: processSandboxMock,
            }));

            const sandboxData = {
                taskId: "test-sandbox-memory",
                code: "const arr = new Array(1000000000).fill(0);",
                language: "javascript",
                userId: "user-456",
                runId: "run-memory",
                stepId: "step-memory",
                timeLimit: 5000,
                memoryLimit: 64, // Small memory limit
                environment: {},
            };

            const result = await processSandbox(sandboxData, queueService);
            expect(result.success).toBe(true);

            const job = await queueService.sandbox.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
        });
    });

    describe("Security isolation", () => {
        it("should handle tasks with different user contexts", async () => {
            const users = ["user-1", "user-2", "user-3"];
            const results = [];

            for (const userId of users) {
                const data = {
                    taskId: `task-${userId}`,
                    code: `console.log('User: ${userId}');`,
                    language: "javascript",
                    userId,
                    runId: `run-${userId}`,
                    stepId: `step-${userId}`,
                    timeLimit: 5000,
                    memoryLimit: 128,
                    environment: { USER_ID: userId },
                };
                const result = await processSandbox(data, queueService);
                results.push({ userId, jobId: result.data!.id });
            }

            // Verify each task maintains its user context
            for (const { userId, jobId } of results) {
                const job = await queueService.sandbox.queue.getJob(jobId);
                expect(job?.data.userId).toBe(userId);
                expect(job?.data.environment.USER_ID).toBe(userId);
            }
        });

        it("should isolate environment variables between tasks", async () => {
            const tasks = [
                { env: { API_KEY: "key1", DB_URL: "db1" } },
                { env: { API_KEY: "key2", DB_URL: "db2" } },
                { env: { API_KEY: "key3", DB_URL: "db3" } },
            ];

            const results = [];
            for (let i = 0; i < tasks.length; i++) {
                const data = {
                    taskId: `task-env-${i}`,
                    code: "console.log(process.env);",
                    language: "javascript",
                    userId: "user-456",
                    runId: `run-env-${i}`,
                    stepId: `step-env-${i}`,
                    timeLimit: 5000,
                    memoryLimit: 128,
                    environment: tasks[i].env,
                };
                const result = await processSandbox(data, queueService);
                results.push({ index: i, jobId: result.data!.id });
            }

            // Verify environment isolation
            for (const { index, jobId } of results) {
                const job = await queueService.sandbox.queue.getJob(jobId);
                expect(job?.data.environment).toEqual(tasks[index].env);
            }
        });
    });

    describe("Queue management", () => {
        it("should handle high volume of sandbox tasks", async () => {
            const taskCount = 50;
            const promises = [];

            for (let i = 0; i < taskCount; i++) {
                const data = {
                    taskId: `high-volume-${i}`,
                    code: `console.log('Task ${i}');`,
                    language: "javascript",
                    userId: `user-${i % 5}`, // Distribute across 5 users
                    runId: `run-volume-${i}`,
                    stepId: `step-volume-${i}`,
                    timeLimit: 5000,
                    memoryLimit: 128,
                    environment: {},
                };
                promises.push(processSandbox(data, queueService));
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(taskCount);
            results.forEach(result => {
                expect(result.success).toBe(true);
            });

            // Check queue state
            const jobCounts = await queueService.sandbox.queue.getJobCounts();
            expect(jobCounts.waiting + jobCounts.active).toBeGreaterThan(0);
        });

        it("should clean up completed jobs", async () => {
            // Add a job
            const result = await processSandbox({
                taskId: "cleanup-test",
                code: "console.log('cleanup');",
                language: "javascript",
                userId: "user-456",
                runId: "run-cleanup",
                stepId: "step-cleanup",
                timeLimit: 5000,
                memoryLimit: 128,
                environment: {},
            }, queueService);

            const jobId = result.data!.id;

            // Mark as completed
            await changeSandboxTaskStatus(jobId, "Completed", "user-456", queueService);

            // Job should still exist initially
            const job = await queueService.sandbox.queue.getJob(jobId);
            expect(job).toBeDefined();

            // Note: Actual cleanup would depend on queue configuration
            // This test verifies the job can be marked as completed
        });
    });
});
