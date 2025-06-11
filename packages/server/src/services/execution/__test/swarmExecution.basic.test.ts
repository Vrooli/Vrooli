import { describe, it, expect } from "vitest";
import { logger } from "../../../events/logger.js";

describe("SwarmExecution Basic Tests", () => {
    describe("Service Imports", () => {
        it("should import SwarmExecutionService without errors", async () => {
            // This tests if our imports and type fixes work
            const { SwarmExecutionService } = await import("../swarmExecutionService.js");
            expect(SwarmExecutionService).toBeDefined();
            expect(typeof SwarmExecutionService).toBe("function");
        });
        
        it("should import all tier components", async () => {
            const { TierOneCoordinator } = await import("../tier1/index.js");
            const { TierTwoOrchestrator } = await import("../tier2/index.js");
            const { TierThreeExecutor } = await import("../tier3/index.js");
            const { EventBus } = await import("../cross-cutting/events/eventBus.js");
            
            expect(TierOneCoordinator).toBeDefined();
            expect(TierTwoOrchestrator).toBeDefined();
            expect(TierThreeExecutor).toBeDefined();
            expect(EventBus).toBeDefined();
        });
        
        it("should import integration services", async () => {
            const { RunPersistenceService } = await import("../integration/runPersistenceService.js");
            const { RoutineStorageService } = await import("../integration/routineStorageService.js");
            const { AuthIntegrationService } = await import("../integration/authIntegrationService.js");
            
            expect(RunPersistenceService).toBeDefined();
            expect(RoutineStorageService).toBeDefined();
            expect(AuthIntegrationService).toBeDefined();
        });
    });
    
    describe("Type Compatibility", () => {
        it("should have compatible types for task processing", async () => {
            const { QueueTaskType } = await import("../../../tasks/taskTypes.js");
            
            expect(QueueTaskType.SWARM_EXECUTION).toBe("swarm-execution");
            expect(typeof QueueTaskType.SWARM_EXECUTION).toBe("string");
        });
    });
});