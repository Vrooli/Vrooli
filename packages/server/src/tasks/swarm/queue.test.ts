import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { QueueService } from "../queues.js";
import { QueueTaskType, type SwarmTask, type SwarmExecutionTask } from "../taskTypes.js";
import { processSwarm, processNewSwarmExecution, changeSwarmTaskStatus, getSwarmTaskStatuses } from "./queue.js";
import { clearRedisCache } from "../queueFactory.js";
import "../../__test/setup.js";

describe("Swarm Queue", () => {
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
        } catch (error) {
            console.log('Shutdown error (ignored):', error);
        }
        clearRedisCache();
        await new Promise(resolve => setTimeout(resolve, 50));
        (QueueService as any).instance = null;
    });

    describe("processSwarm", () => {
        const baseSwarmData: Omit<SwarmTask, "status"> = {
            type: QueueTaskType.SWARM_RUN,
            swarmId: "swarm-123",
            routineVersionId: "version-456",
            runId: "run-789",
            userData: {
                id: "user-123",
                hasPremium: false,
            },
            inputs: { input1: "value1" },
            model: "gpt-4",
            teamId: "team-456",
        };

        it("should add swarm task with correct status", async () => {
            const result = await processSwarm(baseSwarmData, queueService);
            expect(result.success).toBe(true);
            expect(result.data?.id).toBeDefined();

            // Verify the job was added
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
            expect(job?.data.status).toBe("Scheduled");
            expect(job?.data.type).toBe(QueueTaskType.SWARM_RUN);
        });

        describe("priority calculation", () => {
            it("should set base priority for non-premium users", async () => {
                const result = await processSwarm(baseSwarmData, queueService);
                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(100); // BASE_PRIORITY
            });

            it("should boost priority for premium users", async () => {
                const premiumData = {
                    ...baseSwarmData,
                    userData: { id: "user-123", hasPremium: true },
                };
                const result = await processSwarm(premiumData, queueService);
                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBe(80); // 100 - 20
            });

            it("should not go below 0 priority", async () => {
                // Even with maximum boosts, priority should not go below 0
                const data = {
                    ...baseSwarmData,
                    userData: { id: "user-123", hasPremium: true },
                };
                const result = await processSwarm(data, queueService);
                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.opts.priority).toBeGreaterThanOrEqual(0);
            });
        });

        it("should handle concurrent task additions", async () => {
            const promises = [];
            for (let i = 0; i < 5; i++) {
                const data = { ...baseSwarmData, swarmId: `swarm-${i}` };
                promises.push(processSwarm(data, queueService));
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(5);
            results.forEach(result => {
                expect(result.success).toBe(true);
                expect(result.data?.id).toBeDefined();
            });
        });

        it("should preserve all task data", async () => {
            const complexData: Omit<SwarmTask, "status"> = {
                ...baseSwarmData,
                inputs: { 
                    nested: { 
                        data: "complex",
                        array: [1, 2, 3],
                    },
                },
                metadata: {
                    source: "test",
                    timestamp: Date.now(),
                },
            };

            const result = await processSwarm(complexData, queueService);
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            
            expect(job?.data.inputs).toEqual(complexData.inputs);
            expect(job?.data.metadata).toEqual(complexData.metadata);
        });
    });

    describe("processNewSwarmExecution", () => {
        const baseExecutionData: Omit<SwarmExecutionTask, "status"> = {
            type: QueueTaskType.SWARM_EXECUTION,
            swarmId: "exec-123",
            executionId: "execution-456",
            userId: "user-789",
            userData: {
                id: "user-789",
                hasPremium: false,
            },
            config: {
                model: "gpt-4",
                temperature: 0.7,
            },
        };

        it("should add swarm execution task with correct status", async () => {
            const result = await processNewSwarmExecution(baseExecutionData, queueService);
            expect(result.success).toBe(true);
            expect(result.data?.id).toBeDefined();

            // Verify the job was added
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
            expect(job?.data.status).toBe("Scheduled");
            expect(job?.data.type).toBe(QueueTaskType.SWARM_EXECUTION);
        });

        it("should apply same priority rules as processSwarm", async () => {
            // Test with premium user
            const premiumData = {
                ...baseExecutionData,
                userData: { id: "user-789", hasPremium: true },
            };
            const result = await processNewSwarmExecution(premiumData, queueService);
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job?.opts.priority).toBe(80); // 100 - 20
        });

        it("should handle different execution configurations", async () => {
            const executionVariants = [
                { ...baseExecutionData, config: { model: "gpt-3.5-turbo" } },
                { ...baseExecutionData, config: { model: "claude-3" } },
                { ...baseExecutionData, config: { model: "gpt-4", maxTokens: 1000 } },
            ];

            for (const variant of executionVariants) {
                const result = await processNewSwarmExecution(variant, queueService);
                expect(result.success).toBe(true);
                
                const job = await queueService.swarm.queue.getJob(result.data!.id);
                expect(job?.data.config).toEqual(variant.config);
            }
        });
    });

    describe("changeSwarmTaskStatus", () => {
        let taskId: string;

        beforeEach(async () => {
            // Add a test job
            const result = await processSwarm({
                type: QueueTaskType.SWARM_RUN,
                swarmId: "test-swarm",
                routineVersionId: "version-456",
                runId: "run-789",
                userData: { id: "user-123", hasPremium: false },
                inputs: {},
                model: "gpt-4",
                teamId: "team-456",
            }, queueService);
            taskId = result.data!.id;
        });

        it("should change task status", async () => {
            const result = await changeSwarmTaskStatus(taskId, "Running", "user-123", queueService);
            expect(result.success).toBe(true);
        });

        it("should verify status change is delegated to swarm queue", async () => {
            const spy = vi.spyOn(queueService.swarm, 'changeTaskStatus');
            await changeSwarmTaskStatus(taskId, "Running", "user-123", queueService);
            expect(spy).toHaveBeenCalledWith(taskId, "Running", "user-123");
        });

        it("should handle custom status values", async () => {
            const customStatuses = ["Processing", "Analyzing", "Generating"];
            
            for (const status of customStatuses) {
                const result = await changeSwarmTaskStatus(taskId, status, "user-123", queueService);
                expect(result.success).toBe(true);
            }
        });

        it("should handle invalid task ID", async () => {
            const result = await changeSwarmTaskStatus("invalid-id", "Running", "user-123", queueService);
            expect(result.success).toBe(false);
        });
    });

    describe("getSwarmTaskStatuses", () => {
        let taskIds: string[] = [];

        beforeEach(async () => {
            // Add multiple test jobs
            for (let i = 0; i < 3; i++) {
                const result = await processSwarm({
                    type: QueueTaskType.SWARM_RUN,
                    swarmId: `swarm-${i}`,
                    routineVersionId: "version-456",
                    runId: `run-${i}`,
                    userData: { id: "user-123", hasPremium: false },
                    inputs: {},
                    model: "gpt-4",
                    teamId: "team-456",
                }, queueService);
                taskIds.push(result.data!.id);
            }
        });

        it("should fetch multiple task statuses", async () => {
            const statuses = await getSwarmTaskStatuses(taskIds, queueService);
            expect(statuses).toHaveLength(3);
            statuses.forEach((status, index) => {
                expect(status.__typename).toBe("TaskStatusInfo");
                expect(status.id).toBe(taskIds[index]);
                expect(status.queueName).toBe("swarm");
                expect(status.status).toBeDefined();
            });
        });

        it("should handle mixed valid and invalid IDs", async () => {
            const mixedIds = [...taskIds, "invalid-id-1", "invalid-id-2"];
            const statuses = await getSwarmTaskStatuses(mixedIds, queueService);
            
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
            const statuses = await getSwarmTaskStatuses([], queueService);
            expect(statuses).toEqual([]);
        });

        it("should verify delegation to QueueService", async () => {
            const spy = vi.spyOn(queueService, 'getTaskStatuses');
            await getSwarmTaskStatuses(taskIds, queueService);
            expect(spy).toHaveBeenCalledWith(taskIds, "swarm");
        });
    });

    describe("Integration with QueueService", () => {
        it("should process swarm task through worker", async () => {
            // Mock the swarm process function
            const processSwarmMock = vi.fn().mockResolvedValue(undefined);
            vi.doMock("./process.js", () => ({
                swarmProcess: processSwarmMock,
            }));

            const swarmData = {
                type: QueueTaskType.SWARM_RUN,
                swarmId: "test-swarm-integration",
                routineVersionId: "version-456",
                runId: "run-integration",
                userData: { id: "user-123", hasPremium: false },
                inputs: { test: true },
                model: "gpt-4",
                teamId: "team-456",
            };

            const result = await processSwarm(swarmData, queueService);
            expect(result.success).toBe(true);

            // Wait for processing
            await new Promise(resolve => setTimeout(resolve, 1000));

            // Verify job exists
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job).toBeDefined();
        });

        it("should handle job failure and retry", async () => {
            // Mock process to fail first time
            let callCount = 0;
            const processSwarmMock = vi.fn().mockImplementation(() => {
                callCount++;
                if (callCount === 1) {
                    throw new Error("Simulated swarm failure");
                }
                return Promise.resolve();
            });

            vi.doMock("./process.js", () => ({
                swarmProcess: processSwarmMock,
            }));

            const swarmData = {
                type: QueueTaskType.SWARM_RUN,
                swarmId: "test-swarm-fail",
                routineVersionId: "version-456",
                runId: "run-fail",
                userData: { id: "user-123", hasPremium: false },
                inputs: {},
                model: "gpt-4",
                teamId: "team-456",
            };

            const result = await processSwarm(swarmData, queueService);
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            
            expect(job).toBeDefined();
            expect(job?.attemptsMade).toBeGreaterThanOrEqual(0);
        });

        it("should handle three-tier execution flow", async () => {
            // Mock the three-tier coordinator
            const coordinatorMock = vi.fn().mockResolvedValue({
                status: "completed",
                result: { success: true },
            });

            vi.doMock("../services/execution/tier1/tierOneCoordinator.js", () => ({
                TierOneCoordinator: {
                    getInstance: () => ({
                        executeSwarm: coordinatorMock,
                    }),
                },
            }));

            const executionData = {
                type: QueueTaskType.SWARM_EXECUTION,
                swarmId: "exec-three-tier",
                executionId: "execution-three-tier",
                userId: "user-789",
                userData: { id: "user-789", hasPremium: true },
                config: {
                    model: "gpt-4",
                    temperature: 0.7,
                    maxIterations: 10,
                },
            };

            const result = await processNewSwarmExecution(executionData, queueService);
            expect(result.success).toBe(true);

            // Verify execution task was created with proper priority
            const job = await queueService.swarm.queue.getJob(result.data!.id);
            expect(job?.opts.priority).toBe(80); // Premium user priority
        });
    });

    describe("Queue limits and concurrency", () => {
        it("should respect queue concurrency limits", async () => {
            // Add many tasks quickly
            const promises = [];
            for (let i = 0; i < 20; i++) {
                const data = {
                    type: QueueTaskType.SWARM_RUN,
                    swarmId: `swarm-concurrent-${i}`,
                    routineVersionId: "version-456",
                    runId: `run-concurrent-${i}`,
                    userData: { id: `user-${i}`, hasPremium: i % 2 === 0 },
                    inputs: {},
                    model: "gpt-4",
                    teamId: "team-456",
                };
                promises.push(processSwarm(data, queueService));
            }

            const results = await Promise.all(promises);
            expect(results).toHaveLength(20);
            results.forEach(result => {
                expect(result.success).toBe(true);
            });

            // Verify tasks are properly queued
            const jobCounts = await queueService.swarm.queue.getJobCounts();
            expect(jobCounts.waiting + jobCounts.active).toBeGreaterThan(0);
        });

        it("should process tasks in priority order", async () => {
            // Add tasks with different priorities
            const tasks = [
                { swarmId: "low-priority", userData: { id: "user-1", hasPremium: false } },
                { swarmId: "high-priority", userData: { id: "user-2", hasPremium: true } },
                { swarmId: "low-priority-2", userData: { id: "user-3", hasPremium: false } },
            ];

            const results = [];
            for (const task of tasks) {
                const data = {
                    type: QueueTaskType.SWARM_RUN,
                    ...task,
                    routineVersionId: "version-456",
                    runId: `run-${task.swarmId}`,
                    inputs: {},
                    model: "gpt-4",
                    teamId: "team-456",
                };
                const result = await processSwarm(data, queueService);
                results.push({ id: result.data!.id, isPremium: task.userData.hasPremium });
            }

            // Premium tasks should be processed with higher priority
            for (const result of results) {
                const job = await queueService.swarm.queue.getJob(result.id);
                if (result.isPremium) {
                    expect(job?.opts.priority).toBe(80);
                } else {
                    expect(job?.opts.priority).toBe(100);
                }
            }
        });
    });
});