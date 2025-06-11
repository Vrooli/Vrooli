import { describe, it, expect, beforeAll, afterAll, beforeEach } from "vitest";
import { logger } from "../../../events/logger.js";
import { DbProvider } from "../../../db/provider.js";
import { SwarmExecutionService } from "../swarmExecutionService.js";
import { QueueTaskType } from "../../../tasks/taskTypes.js";
import { nanoid } from "@vrooli/shared";

describe("SwarmExecution Integration Tests", () => {
    let swarmExecutionService: SwarmExecutionService;
    
    beforeAll(async () => {
        // Initialize database for testing
        await DbProvider.init();
        
        // Wait for database to be ready
        const maxWaitTime = 10000; // 10 seconds
        const startTime = Date.now();
        
        while (!DbProvider.isConnected() && (Date.now() - startTime < maxWaitTime)) {
            await new Promise(resolve => setTimeout(resolve, 100));
        }
        
        if (!DbProvider.isConnected()) {
            throw new Error("Database not connected after maximum wait time");
        }
        
        // Initialize SwarmExecutionService
        swarmExecutionService = new SwarmExecutionService(logger);
    });
    
    afterAll(async () => {
        // Shutdown services
        if (swarmExecutionService) {
            await swarmExecutionService.shutdown();
        }
        
        // Only close database in test environment
        if (process.env.NODE_ENV === "test") {
            await DbProvider.shutdown();
        }
    });
    
    beforeEach(async () => {
        // Clean up any test data if needed
        if (process.env.NODE_ENV === "test") {
            // For now, just log that we're starting a new test
            logger.info("Starting new swarm execution test");
        }
    });
    
    describe("Service Initialization", () => {
        it("should initialize SwarmExecutionService without errors", () => {
            expect(swarmExecutionService).toBeDefined();
            expect(swarmExecutionService).toBeInstanceOf(SwarmExecutionService);
        });
    });
    
    describe("Swarm Lifecycle", () => {
        it("should handle basic swarm creation", async () => {
            const swarmId = nanoid();
            const adminId = await DbProvider.getAdminId();
            
            const swarmConfig = {
                swarmId,
                name: "Test Swarm",
                description: "Integration test swarm",
                goal: "Test the three-tier execution architecture",
                resources: {
                    maxCredits: 100,
                    maxTokens: 1000,
                    maxTime: 300000, // 5 minutes
                    tools: [
                        { name: "echo", description: "Echo back input" },
                    ],
                },
                config: {
                    model: "claude-3-sonnet-20240229",
                    temperature: 0.7,
                    autoApproveTools: true,
                    parallelExecutionLimit: 3,
                },
                userId: adminId,
                organizationId: undefined,
            };
            
            // Test swarm creation
            const result = await swarmExecutionService.startSwarm(swarmConfig);
            
            expect(result).toBeDefined();
            expect(result.swarmId).toBe(swarmId);
            
            // Test swarm status retrieval
            const status = await swarmExecutionService.getSwarmStatus(swarmId);
            
            expect(status).toBeDefined();
            expect(status.status).toBeDefined();
            
            // Cleanup: Cancel the swarm
            const cancelResult = await swarmExecutionService.cancelSwarm(swarmId, adminId, "Test cleanup");
            expect(cancelResult.success).toBe(true);
        });
        
        it("should handle invalid user permissions", async () => {
            const swarmId = nanoid();
            const invalidUserId = "invalid-user-id";
            
            const swarmConfig = {
                swarmId,
                name: "Invalid Test Swarm",
                description: "Test with invalid user",
                goal: "This should fail",
                resources: {
                    maxCredits: 100,
                    maxTokens: 1000,
                    maxTime: 300000,
                    tools: [],
                },
                config: {
                    model: "claude-3-sonnet-20240229",
                    temperature: 0.7,
                    autoApproveTools: true,
                    parallelExecutionLimit: 3,
                },
                userId: invalidUserId,
                organizationId: undefined,
            };
            
            // This should throw an error
            await expect(swarmExecutionService.startSwarm(swarmConfig)).rejects.toThrow();
        });
    });
    
    describe("Run Lifecycle", () => {
        it("should handle basic run creation", async () => {
            // Note: This test requires a valid routine in the database
            // For now, we'll test the error case where routine is not found
            
            const runId = nanoid();
            const swarmId = nanoid();
            const adminId = await DbProvider.getAdminId();
            
            const runConfig = {
                runId,
                swarmId,
                routineVersionId: "non-existent-routine",
                inputs: {
                    testInput: "Hello, world!",
                },
                config: {
                    strategy: "conversational",
                    model: "claude-3-sonnet-20240229",
                    maxSteps: 10,
                    timeout: 60000,
                },
                userId: adminId,
            };
            
            // This should throw an error because the routine doesn't exist
            await expect(swarmExecutionService.startRun(runConfig)).rejects.toThrow(/Routine version not found/);
        });
        
        it("should handle run status queries", async () => {
            const runId = "non-existent-run";
            
            const status = await swarmExecutionService.getRunStatus(runId);
            
            expect(status).toBeDefined();
            expect(status.errors).toContain("Run not found");
        });
    });
    
    describe("Error Handling", () => {
        it("should handle swarm status for non-existent swarm", async () => {
            const nonExistentSwarmId = "non-existent-swarm";
            
            const status = await swarmExecutionService.getSwarmStatus(nonExistentSwarmId);
            
            expect(status).toBeDefined();
            expect(status.errors).toBeDefined();
            expect(status.errors!.length).toBeGreaterThan(0);
        });
        
        it("should handle cancellation of non-existent swarm", async () => {
            const nonExistentSwarmId = "non-existent-swarm";
            const adminId = await DbProvider.getAdminId();
            
            const result = await swarmExecutionService.cancelSwarm(nonExistentSwarmId, adminId, "Test");
            
            expect(result).toBeDefined();
            expect(result.success).toBe(false);
            expect(result.message).toBeDefined();
        });
    });
});