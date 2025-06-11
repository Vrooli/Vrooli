import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { SwarmExecutionService } from "../swarmExecutionService.js";
import { EventBus } from "../cross-cutting/events/eventBus.js";
import { TierOneCoordinator } from "../tier1/index.js";
import { TierTwoOrchestrator } from "../tier2/index.js";
import { TierThreeExecutor } from "../tier3/index.js";
import { RunPersistenceService } from "../integration/runPersistenceService.js";
import { RoutineStorageService } from "../integration/routineStorageService.js";
import { AuthIntegrationService } from "../integration/authIntegrationService.js";
import { SwarmStatus, RunStatus } from "@vrooli/shared";

describe("SwarmExecutionService", () => {
    let service: SwarmExecutionService;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility
    let tierOneStub: TierOneCoordinator;
    let tierTwoStub: TierTwoOrchestrator;
    let tierThreeStub: TierThreeExecutor;
    let persistenceStub: RunPersistenceService;
    let routineStub: RoutineStorageService;
    let authStub: AuthIntegrationService;

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });

        // Stub all dependencies
        tierOneStub = vi.mocked(new TierOneCoordinator() as any);
        tierTwoStub = vi.mocked(new TierTwoOrchestrator() as any);
        tierThreeStub = vi.mocked(new TierThreeExecutor() as any);
        persistenceStub = vi.mocked(new RunPersistenceService() as any);
        routineStub = vi.mocked(new RoutineStorageService() as any);
        authStub = vi.mocked(new AuthIntegrationService() as any);

        // Replace constructors with stubs
        sandbox.stub(TierOneCoordinator.prototype, "constructor").returns(tierOneStub as any);
        sandbox.stub(TierTwoOrchestrator.prototype, "constructor").returns(tierTwoStub as any);
        sandbox.stub(TierThreeExecutor.prototype, "constructor").returns(tierThreeStub as any);
        sandbox.stub(RunPersistenceService.prototype, "constructor").returns(persistenceStub as any);
        sandbox.stub(RoutineStorageService.prototype, "constructor").returns(routineStub as any);
        sandbox.stub(AuthIntegrationService.prototype, "constructor").returns(authStub as any);

        service = new SwarmExecutionService(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("startSwarm", () => {
        it("should successfully start a swarm with valid permissions", async () => {
            const swarmConfig = {
                swarmId: "swarm-123",
                name: "Test Swarm",
                description: "Test description",
                goal: "Test goal",
                resources: {
                    maxCredits: 10000,
                    maxTokens: 100000,
                    maxTime: 3600000,
                    tools: [{ name: "test_tool", description: "Test tool" }],
                },
                config: {
                    model: "gpt-4o-mini",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 5,
                },
                userId: "user-123",
                organizationId: "org-123",
            };

            authStub.checkPermission.mockResolvedValue(true);
            tierOneStub.startSwarm.mockResolvedValue();

            const result = await service.startSwarm(swarmConfig);

            expect(result).toEqual({ swarmId: "swarm-123" });
            expect(authStub.checkPermission).toHaveBeenCalledWith(
                "user-123",
                "swarm:create",
                "org-123"
            );
            expect(tierOneStub.startSwarm).toHaveBeenCalledWith(swarmConfig);
        });

        it("should throw error when user lacks permissions", async () => {
            const swarmConfig = {
                swarmId: "swarm-123",
                name: "Test Swarm",
                description: "Test description",
                goal: "Test goal",
                resources: {
                    maxCredits: 10000,
                    maxTokens: 100000,
                    maxTime: 3600000,
                    tools: [],
                },
                config: {
                    model: "gpt-4o-mini",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 5,
                },
                userId: "user-123",
            };

            authStub.checkPermission.mockResolvedValue(false);

            try {
                await service.startSwarm(swarmConfig);
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("User does not have permission to create swarms");
            }
            expect(tierOneStub.startSwarm).to.not.have.been.called;
        });
    });

    describe("startRun", () => {
        it("should successfully start a run with valid routine and permissions", async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routineVersionId: "routine-v1",
                inputs: { test: "input" },
                config: {
                    strategy: "reasoning",
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            const mockRoutine = {
                id: "routine-v1",
                name: "Test Routine",
                root: { name: "Test", ownerId: "owner-123" },
            };

            const mockRun = {
                id: "run-123",
                routineVersionId: "routine-v1",
                userId: "user-123",
                status: RunStatus.InProgress,
            };

            routineStub.loadRoutine.mockResolvedValue(mockRoutine);
            authStub.checkPermission.mockResolvedValue(true);
            persistenceStub.createRun.mockResolvedValue(mockRun as any);
            tierOneStub.requestRunExecution.mockResolvedValue();

            const result = await service.startRun(runConfig);

            expect(result).toEqual({ runId: "run-123" });
            expect(routineStub.loadRoutine).toHaveBeenCalledWith("routine-v1");
            expect(authStub.checkPermission).toHaveBeenCalledWith(
                "user-123",
                "routine:run",
                "owner-123"
            );
            expect(persistenceStub.createRun).toHaveBeenCalled();
            expect(tierOneStub.requestRunExecution).toHaveBeenCalledWith({
                swarmId: "swarm-123",
                runId: "run-123",
                routineVersionId: "routine-v1",
                inputs: { test: "input" },
                config: runConfig.config,
            });
        });

        it("should throw error when routine not found", async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routineVersionId: "routine-v1",
                inputs: {},
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            routineStub.loadRoutine.mockResolvedValue(null);

            try {
                await service.startRun(runConfig);
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("Routine version not found: routine-v1");
            }
            expect(persistenceStub.createRun).to.not.have.been.called;
        });

        it("should update run status to failed on error", async () => {
            const runConfig = {
                runId: "run-123",
                swarmId: "swarm-123",
                routineVersionId: "routine-v1",
                inputs: {},
                config: {
                    model: "gpt-4o-mini",
                    maxSteps: 100,
                    timeout: 300000,
                },
                userId: "user-123",
            };

            const mockRoutine = {
                id: "routine-v1",
                name: "Test Routine",
                root: { name: "Test", ownerId: "owner-123" },
            };

            routineStub.loadRoutine.mockResolvedValue(mockRoutine);
            authStub.checkPermission.mockResolvedValue(true);
            persistenceStub.createRun.mockRejectedValue(new Error("Database error"));

            try {
                await service.startRun(runConfig);
                throw new Error("Should have thrown an error");
            } catch (error) {
                expect(error.message).toBe("Database error");
            }
            expect(persistenceStub.updateRunStatus).toHaveBeenCalledWith("run-123", RunStatus.Failed);
        });
    });

    describe("getSwarmStatus", () => {
        it("should return swarm status from tier one", async () => {
            const expectedStatus = {
                status: SwarmStatus.Running,
                progress: 50,
                currentPhase: "ACTIVE",
                activeRuns: 2,
                completedRuns: 3,
                errors: [],
            };

            tierOneStub.getSwarmStatus.mockResolvedValue(expectedStatus);

            const result = await service.getSwarmStatus("swarm-123");

            expect(result).toEqual(expectedStatus);
            expect(tierOneStub.getSwarmStatus).toHaveBeenCalledWith("swarm-123");
        });

        it("should return unknown status on error", async () => {
            tierOneStub.getSwarmStatus.mockRejectedValue(new Error("Network error"));

            const result = await service.getSwarmStatus("swarm-123");

            expect(result).toEqual({
                status: SwarmStatus.Unknown,
                errors: ["Network error"],
            });
        });
    });

    describe("getRunStatus", () => {
        it("should return combined run status from persistence and tier two", async () => {
            const mockRun = {
                id: "run-123",
                status: RunStatus.InProgress,
                outputs: { result: "partial" },
            };

            const mockDetailedStatus = {
                progress: 75,
                currentStep: "step-5",
                errors: [],
            };

            persistenceStub.getRun.mockResolvedValue(mockRun as any);
            tierTwoStub.getRunStatus.mockResolvedValue(mockDetailedStatus);

            const result = await service.getRunStatus("run-123");

            expect(result).toEqual({
                status: RunStatus.InProgress,
                progress: 75,
                currentStep: "step-5",
                outputs: { result: "partial" },
                errors: [],
            });
        });

        it("should return not found status when run doesn't exist", async () => {
            persistenceStub.getRun.mockResolvedValue(null);

            const result = await service.getRunStatus("run-123");

            expect(result).toEqual({
                status: RunStatus.Unknown,
                errors: ["Run not found"],
            });
            expect(tierTwoStub.getRunStatus).to.not.have.been.called;
        });
    });

    describe("cancelSwarm", () => {
        it("should successfully cancel a swarm", async () => {
            tierOneStub.cancelSwarm.mockResolvedValue();

            const result = await service.cancelSwarm("swarm-123", "user-123", "User requested");

            expect(result).toEqual({
                success: true,
                message: "Swarm cancelled successfully",
            });
            expect(tierOneStub.cancelSwarm).toHaveBeenCalledWith(
                "swarm-123",
                "user-123",
                "User requested"
            );
        });

        it("should return failure on error", async () => {
            tierOneStub.cancelSwarm.mockRejectedValue(new Error("Swarm not found"));

            const result = await service.cancelSwarm("swarm-123", "user-123");

            expect(result).toEqual({
                success: false,
                message: "Swarm not found",
            });
        });
    });

    describe("cancelRun", () => {
        it("should successfully cancel a run", async () => {
            persistenceStub.updateRunStatus.mockResolvedValue();
            tierTwoStub.cancelRun.mockResolvedValue();

            const result = await service.cancelRun("run-123", "user-123", "User requested");

            expect(result).toEqual({
                success: true,
                message: "Run cancelled successfully",
            });
            expect(persistenceStub.updateRunStatus).toHaveBeenCalledWith("run-123", RunStatus.Cancelled);
            expect(tierTwoStub.cancelRun).toHaveBeenCalledWith("run-123", "User requested");
        });
    });

    describe("event handling", () => {
        it("should handle run completion events", async () => {
            // Get the actual event bus instance
            const eventBus = (service as any).eventBus;
            
            persistenceStub.completeRun.mockResolvedValue();

            await eventBus.publish("run.completed", {
                runId: "run-123",
                outputs: { result: "success" },
            });

            // Allow event to be processed
            await new Promise(resolve => setTimeout(resolve, 10));

            expect(persistenceStub.completeRun).toHaveBeenCalledWith("run-123", { result: "success" });
        });

        it("should handle run failure events", async () => {
            const eventBus = (service as any).eventBus;
            
            persistenceStub.updateRunStatus.mockResolvedValue();

            await eventBus.publish("run.failed", {
                runId: "run-123",
                error: "Step execution failed",
            });

            // Allow event to be processed
            await new Promise(resolve => setTimeout(resolve, 10));

            expect(persistenceStub.updateRunStatus).toHaveBeenCalledWith("run-123", RunStatus.Failed);
        });
    });

    describe("shutdown", () => {
        it("should shutdown all tiers", async () => {
            tierOneStub.shutdown.mockResolvedValue();
            tierTwoStub.shutdown.mockResolvedValue();
            tierThreeStub.shutdown.mockResolvedValue();

            await service.shutdown();

            expect(tierOneStub.shutdown).toHaveBeenCalled();
            expect(tierTwoStub.shutdown).toHaveBeenCalled();
            expect(tierThreeStub.shutdown).toHaveBeenCalled();
        });
    });
});