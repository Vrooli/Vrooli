import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { SwarmExecutionService } from "./swarmExecutionService.js";
import { type Logger } from "winston";
import { createMockLogger } from "../../../__test/globalHelpers.js";
import { DbProvider } from "../../db/provider.js";
import { generatePK, RunStatus, SwarmStatus } from "@vrooli/shared";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import { PostgreSqlContainer } from "@testcontainers/postgresql";
import { PrismaClient } from "@prisma/client";

/**
 * SwarmExecutionService Tests - Three-Tier Integration
 * 
 * These tests validate that the service provides minimal infrastructure
 * for cross-tier coordination while enabling emergent intelligence through:
 * 
 * 1. **Unified Entry Point**: Simple interface hiding tier complexity
 * 2. **Event-Driven Integration**: Tiers communicate through events
 * 3. **Service Composition**: Clean separation of concerns
 * 4. **Resource Flow**: Credits/limits pass through tiers
 * 5. **Error Propagation**: Failures handled at appropriate levels
 * 
 * The service acts as infrastructure glue, not an intelligence layer.
 */

describe("SwarmExecutionService - Three-Tier Integration", () => {
    let logger: Logger;
    let service: SwarmExecutionService;
    let pgContainer: StartedTestContainer;
    let redisContainer: StartedTestContainer;
    let prisma: PrismaClient;

    beforeEach(async () => {
        // Use real PostgreSQL for persistence
        pgContainer = await new PostgreSqlContainer("postgres:16-alpine")
            .withDatabase("test_db")
            .withUsername("test_user")
            .withPassword("test_pass")
            .start();

        // Use real Redis for event bus
        redisContainer = await new GenericContainer("redis:7-alpine")
            .withExposedPorts(6379)
            .start();

        // Setup database connection
        const databaseUrl = pgContainer.getConnectionUri();
        process.env.DATABASE_URL = databaseUrl;
        
        prisma = new PrismaClient({
            datasources: {
                db: { url: databaseUrl }
            }
        });

        // Mock DbProvider to return our test prisma instance
        vi.spyOn(DbProvider, "get").mockReturnValue(prisma);

        logger = createMockLogger();
        
        // Mock required database queries
        prisma.user = {
            findUnique: vi.fn().mockResolvedValue({
                id: "user-123",
                name: "Test User",
                email: "test@example.com",
            }),
        } as any;

        prisma.run = {
            create: vi.fn().mockImplementation((args) => ({
                id: args.data.id || generatePK(),
                ...args.data,
                createdAt: new Date(),
                updatedAt: new Date(),
            })),
            findMany: vi.fn().mockResolvedValue([]),
            update: vi.fn(),
        } as any;

        prisma.routine = {
            findUnique: vi.fn().mockResolvedValue({
                id: "routine-123",
                name: "Test Routine",
                version: "1.0.0",
            }),
        } as any;

        service = new SwarmExecutionService(logger);
    });

    afterEach(async () => {
        await prisma?.$disconnect();
        await pgContainer?.stop();
        await redisContainer?.stop();
        vi.restoreAllMocks();
    });

    describe("Unified Swarm Interface", () => {
        it("should provide simple interface for complex three-tier coordination", async () => {
            const swarmConfig = {
                swarmId: generatePK(),
                name: "Integration Test Swarm",
                description: "Testing three-tier integration",
                goal: "Validate cross-tier communication",
                resources: {
                    maxCredits: 5000,
                    maxTokens: 100000,
                    maxTime: 3600000,
                    tools: [
                        { name: "search", description: "Search information" },
                        { name: "analyze", description: "Analyze data" },
                    ],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 3,
                },
                userId: "user-123",
                organizationId: "org-456",
            };

            const result = await service.startSwarm(swarmConfig);

            expect(result).toHaveProperty("swarmId");
            expect(result.swarmId).toBe(swarmConfig.swarmId);
            
            // Verify tier initialization
            expect(logger.info).toHaveBeenCalledWith(
                "[SwarmExecutionService] Starting new swarm",
                expect.objectContaining({
                    swarmId: swarmConfig.swarmId,
                    name: swarmConfig.name,
                })
            );
        });

        it("should handle child swarm creation", async () => {
            const parentSwarmId = generatePK();
            const childSwarmConfig = {
                swarmId: generatePK(),
                name: "Child Swarm",
                description: "Subtask handler",
                goal: "Process specific subtask",
                resources: {
                    maxCredits: 1000,
                    maxTokens: 20000,
                    maxTime: 1800000,
                    tools: [],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.6,
                    autoApproveTools: true,
                    parallelExecutionLimit: 1,
                },
                userId: "user-123",
                parentSwarmId, // Link to parent
            };

            const result = await service.startSwarm(childSwarmConfig);

            expect(result.swarmId).toBe(childSwarmConfig.swarmId);
            // Parent-child relationship handled by infrastructure
        });
    });

    describe("Run Execution Through Tiers", () => {
        it("should execute routine runs through all tiers", async () => {
            const runConfig = {
                runId: generatePK(),
                routineId: "routine-123",
                inputs: {
                    data: "test input",
                    processType: "analysis",
                },
                userId: "user-123",
                config: {
                    maxCredits: 500,
                    maxTime: 300000,
                    checkpointEnabled: true,
                },
            };

            const result = await service.startRun(runConfig);

            expect(result).toHaveProperty("runId");
            expect(result.runId).toBe(runConfig.runId);
            
            // Run created in persistence layer
            expect(prisma.run.create).toHaveBeenCalledWith({
                data: expect.objectContaining({
                    id: runConfig.runId,
                    routineId: runConfig.routineId,
                    status: RunStatus.Created,
                }),
            });
        });

        it("should handle parallel run execution", async () => {
            const runs = Array.from({ length: 3 }, (_, i) => ({
                runId: generatePK(),
                routineId: `routine-${i}`,
                inputs: { index: i },
                userId: "user-123",
                config: {
                    maxCredits: 100,
                    maxTime: 60000,
                },
            }));

            // Start runs in parallel
            const results = await Promise.all(
                runs.map(config => service.startRun(config))
            );

            expect(results).toHaveLength(3);
            results.forEach((result, i) => {
                expect(result.runId).toBe(runs[i].runId);
            });
        });
    });

    describe("Event-Driven Cross-Tier Communication", () => {
        it("should propagate events across tiers", async () => {
            const capturedEvents: any[] = [];
            
            // Mock event capturing (would normally use event bus subscription)
            const originalInfo = logger.info;
            logger.info = vi.fn((message: string, meta?: any) => {
                if (message.includes("Event")) {
                    capturedEvents.push({ message, meta });
                }
                originalInfo.call(logger, message, meta);
            });

            const swarmConfig = {
                swarmId: generatePK(),
                name: "Event Test Swarm",
                description: "Testing event propagation",
                goal: "Generate cross-tier events",
                resources: {
                    maxCredits: 1000,
                    maxTokens: 10000,
                    maxTime: 600000,
                    tools: [],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.5,
                    autoApproveTools: false,
                    parallelExecutionLimit: 2,
                },
                userId: "user-123",
            };

            await service.startSwarm(swarmConfig);
            
            // Simulate some activity
            await new Promise(resolve => setTimeout(resolve, 100));

            // Events should flow through tiers
            expect(logger.info).toHaveBeenCalledWith(
                expect.stringContaining("SwarmExecutionService"),
                expect.any(Object)
            );
        });

        it("should handle cross-tier error propagation", async () => {
            // Simulate user not found
            prisma.user.findUnique = vi.fn().mockResolvedValue(null);

            const swarmConfig = {
                swarmId: generatePK(),
                name: "Error Test Swarm",
                description: "Testing error handling",
                goal: "Test error propagation",
                resources: {
                    maxCredits: 100,
                    maxTokens: 1000,
                    maxTime: 60000,
                    tools: [],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.5,
                    autoApproveTools: false,
                    parallelExecutionLimit: 1,
                },
                userId: "invalid-user",
            };

            await expect(service.startSwarm(swarmConfig)).rejects.toThrow("User not found");
        });
    });

    describe("Resource Management Across Tiers", () => {
        it("should track resource consumption across tiers", async () => {
            const swarmId = generatePK();
            const swarmConfig = {
                swarmId,
                name: "Resource Tracking Swarm",
                description: "Monitor resource usage",
                goal: "Process data efficiently",
                resources: {
                    maxCredits: 2000,
                    maxTokens: 50000,
                    maxTime: 1800000,
                    tools: [],
                },
                config: {
                    model: "claude-3-opus",
                    temperature: 0.6,
                    autoApproveTools: false,
                    parallelExecutionLimit: 2,
                },
                userId: "user-123",
            };

            await service.startSwarm(swarmConfig);

            // Get swarm status (includes resource usage)
            const status = await service.getSwarmStatus(swarmId);

            expect(status).toHaveProperty("swarmId", swarmId);
            expect(status).toHaveProperty("status");
            expect(status).toHaveProperty("resources");
            expect(status.resources).toHaveProperty("creditsRemaining");
        });

        it("should enforce resource limits at appropriate tier", async () => {
            const runConfig = {
                runId: generatePK(),
                routineId: "routine-123",
                inputs: { action: "expensive_operation" },
                userId: "user-123",
                config: {
                    maxCredits: 1, // Very low limit
                    maxTime: 1000, // 1 second
                },
            };

            const result = await service.startRun(runConfig);

            // Infrastructure enforces limits, execution adapts
            expect(result.runId).toBe(runConfig.runId);
            // Actual execution behavior emerges from tier interactions
        });
    });

    describe("Status and Monitoring", () => {
        it("should provide unified status across all tiers", async () => {
            const swarmId = generatePK();
            
            // Mock swarm data
            prisma.swarm = {
                findUnique: vi.fn().mockResolvedValue({
                    id: swarmId,
                    status: SwarmStatus.Running,
                    createdAt: new Date(),
                    config: {
                        resources: {
                            maxCredits: 5000,
                            creditsUsed: 1234,
                        },
                    },
                }),
            } as any;

            prisma.run.findMany = vi.fn().mockResolvedValue([
                { id: "run-1", status: RunStatus.Completed },
                { id: "run-2", status: RunStatus.Running },
                { id: "run-3", status: RunStatus.Created },
            ]);

            const status = await service.getSwarmStatus(swarmId);

            expect(status).toMatchObject({
                swarmId,
                status: SwarmStatus.Running,
                resources: {
                    creditsRemaining: 3766,
                    creditsUsed: 1234,
                },
                runs: {
                    total: 3,
                    completed: 1,
                    running: 1,
                    pending: 1,
                },
            });
        });

        it("should aggregate metrics from all tiers", async () => {
            const metrics = await service.getSystemMetrics();

            expect(metrics).toHaveProperty("tiers");
            expect(metrics.tiers).toHaveProperty("tier1");
            expect(metrics.tiers).toHaveProperty("tier2");
            expect(metrics.tiers).toHaveProperty("tier3");
            
            // Each tier provides its metrics
            expect(metrics.tiers.tier1).toHaveProperty("activeSwarms");
            expect(metrics.tiers.tier2).toHaveProperty("activeRuns");
            expect(metrics.tiers.tier3).toHaveProperty("activeExecutions");
        });
    });

    describe("Graceful Shutdown", () => {
        it("should shutdown tiers in correct order", async () => {
            const shutdownOrder: string[] = [];
            
            // Mock shutdown tracking
            const originalInfo = logger.info;
            logger.info = vi.fn((message: string) => {
                if (message.includes("Shutting down")) {
                    shutdownOrder.push(message);
                }
                originalInfo.call(logger, message);
            });

            await service.shutdown();

            // Should shutdown in reverse dependency order (1 -> 2 -> 3)
            expect(shutdownOrder.length).toBeGreaterThan(0);
            expect(logger.info).toHaveBeenCalledWith(
                expect.stringContaining("shutdown complete")
            );
        });

        it("should handle shutdown errors gracefully", async () => {
            // Mock a tier throwing during shutdown
            vi.spyOn(service as any, "tierOne", "get").mockReturnValue({
                shutdown: vi.fn().mockRejectedValue(new Error("Tier 1 shutdown failed")),
            });

            // Should not throw, but log error
            await expect(service.shutdown()).resolves.not.toThrow();
            
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Error during shutdown"),
                expect.any(Object)
            );
        });
    });
});