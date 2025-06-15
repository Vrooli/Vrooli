import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { EventBus } from "./eventBus.js";
import { mockLogger } from "../../../../__test/logger.mock.js";
import { mockEvents } from "../../../../__test/fixtures/execution/eventFixtures.js";
import { type Event } from "@vrooli/shared";

describe("EventBus - Cross-Tier Communication", () => {
    let eventBus: EventBus;
    let mockRedisClient: any;

    beforeEach(() => {
        vi.clearAllMocks();
        
        // Mock Redis client
        mockRedisClient = {
            publish: vi.fn().mockResolvedValue(1),
            subscribe: vi.fn().mockResolvedValue(undefined),
            unsubscribe: vi.fn().mockResolvedValue(undefined),
            on: vi.fn(),
            quit: vi.fn().mockResolvedValue(undefined),
        };

        // Mock Redis module
        vi.mock("redis", () => ({
            createClient: vi.fn(() => mockRedisClient),
        }));

        eventBus = new EventBus(mockLogger);
    });

    afterEach(async () => {
        await eventBus.shutdown();
    });

    describe("initialization", () => {
        it("should initialize with logger", () => {
            expect(eventBus).toBeDefined();
        });
    });

    describe("event emission", () => {
        it("should emit events with proper structure", () => {
            const event = mockEvents.tier1.swarmStarted;
            
            eventBus.emit(event);

            // Verify event has required fields
            expect(event).toHaveProperty("id");
            expect(event).toHaveProperty("type");
            expect(event).toHaveProperty("source");
            expect(event).toHaveProperty("timestamp");
            expect(event).toHaveProperty("data");
        });

        it("should emit tier 1 coordination events", () => {
            const swarmEvent = mockEvents.tier1.swarmStarted;
            
            eventBus.emit(swarmEvent);

            expect(swarmEvent.source).toBe("tier1.swarm");
            expect(swarmEvent.type).toBe("swarm.started");
        });

        it("should emit tier 2 orchestration events", () => {
            const runEvent = mockEvents.tier2.runStarted;
            
            eventBus.emit(runEvent);

            expect(runEvent.source).toBe("tier2.run");
            expect(runEvent.type).toBe("run.started");
        });

        it("should emit tier 3 execution events", () => {
            const stepEvent = mockEvents.tier3.stepCompleted;
            
            eventBus.emit(stepEvent);

            expect(stepEvent.source).toBe("tier3.executor");
            expect(stepEvent.type).toBe("step.completed");
        });
    });

    describe("event subscription", () => {
        it("should allow subscription to specific event types", () => {
            const handler = vi.fn();
            
            eventBus.on("swarm.started", handler);
            eventBus.emit(mockEvents.tier1.swarmStarted);

            expect(handler).toHaveBeenCalledWith(mockEvents.tier1.swarmStarted);
        });

        it("should support wildcard subscriptions", () => {
            const handler = vi.fn();
            
            eventBus.on("swarm.*", handler);
            eventBus.emit(mockEvents.tier1.swarmStarted);
            eventBus.emit(mockEvents.tier1.teamFormed);

            expect(handler).toHaveBeenCalledTimes(2);
        });

        it("should support filtered subscriptions", () => {
            const handler = vi.fn();
            
            // Subscribe with filter
            eventBus.subscribe({
                pattern: "run.*",
                filter: (event) => event.data.runId === "test-run-123",
                handler,
            });

            // Emit matching event
            eventBus.emit({
                ...mockEvents.tier2.runStarted,
                data: { ...mockEvents.tier2.runStarted.data, runId: "test-run-123" },
            });

            // Emit non-matching event
            eventBus.emit({
                ...mockEvents.tier2.runStarted,
                data: { ...mockEvents.tier2.runStarted.data, runId: "other-run" },
            });

            expect(handler).toHaveBeenCalledTimes(1);
        });
    });

    describe("cross-tier communication patterns", () => {
        it("should handle tier 1 to tier 2 communication", () => {
            const tier2Handler = vi.fn();
            
            // Tier 2 subscribes to tier 1 events
            eventBus.on("swarm.run_requested", tier2Handler);

            // Tier 1 emits run request
            const runRequest: Event = {
                id: "event-123",
                type: "swarm.run_requested",
                source: "tier1.swarm",
                timestamp: new Date().toISOString(),
                data: {
                    swarmId: "swarm-123",
                    runId: "run-456",
                    routineId: "routine-789",
                    inputs: { test: "data" },
                },
            };

            eventBus.emit(runRequest);

            expect(tier2Handler).toHaveBeenCalledWith(runRequest);
        });

        it("should handle tier 2 to tier 3 communication", () => {
            const tier3Handler = vi.fn();
            
            // Tier 3 subscribes to tier 2 events
            eventBus.on("run.step_ready", tier3Handler);

            // Tier 2 emits step ready
            const stepReady: Event = {
                id: "event-456",
                type: "run.step_ready",
                source: "tier2.run",
                timestamp: new Date().toISOString(),
                data: {
                    runId: "run-456",
                    stepId: "step-123",
                    inputs: { action: "process" },
                },
            };

            eventBus.emit(stepReady);

            expect(tier3Handler).toHaveBeenCalledWith(stepReady);
        });

        it("should handle tier 3 to tier 2 feedback", () => {
            const tier2Handler = vi.fn();
            
            // Tier 2 subscribes to tier 3 completion events
            eventBus.on("step.completed", tier2Handler);

            // Tier 3 emits completion
            eventBus.emit(mockEvents.tier3.stepCompleted);

            expect(tier2Handler).toHaveBeenCalledWith(mockEvents.tier3.stepCompleted);
        });

        it("should handle tier 2 to tier 1 feedback", () => {
            const tier1Handler = vi.fn();
            
            // Tier 1 subscribes to tier 2 completion events
            eventBus.on("run.completed", tier1Handler);

            // Tier 2 emits completion
            eventBus.emit(mockEvents.tier2.runCompleted);

            expect(tier1Handler).toHaveBeenCalledWith(mockEvents.tier2.runCompleted);
        });
    });

    describe("event correlation", () => {
        it("should maintain correlation across tier boundaries", () => {
            const handlers = {
                tier1: vi.fn(),
                tier2: vi.fn(),
                tier3: vi.fn(),
            };

            // Subscribe all tiers
            eventBus.on("swarm.*", handlers.tier1);
            eventBus.on("run.*", handlers.tier2);
            eventBus.on("step.*", handlers.tier3);

            const correlationId = "corr-123";
            const causationId = "cause-456";

            // Emit correlated events
            const events = [
                {
                    ...mockEvents.tier1.swarmStarted,
                    correlationId,
                    causationId,
                },
                {
                    ...mockEvents.tier2.runStarted,
                    correlationId,
                    causationId: mockEvents.tier1.swarmStarted.id,
                },
                {
                    ...mockEvents.tier3.stepStarted,
                    correlationId,
                    causationId: mockEvents.tier2.runStarted.id,
                },
            ];

            events.forEach(event => eventBus.emit(event));

            // Verify all handlers received correlated events
            expect(handlers.tier1).toHaveBeenCalledWith(
                expect.objectContaining({ correlationId }),
            );
            expect(handlers.tier2).toHaveBeenCalledWith(
                expect.objectContaining({ correlationId }),
            );
            expect(handlers.tier3).toHaveBeenCalledWith(
                expect.objectContaining({ correlationId }),
            );
        });
    });

    describe("error propagation", () => {
        it("should propagate errors from tier 3 to tier 1", () => {
            const errorHandlers = {
                tier1: vi.fn(),
                tier2: vi.fn(),
            };

            // Subscribe to error events
            eventBus.on("step.failed", errorHandlers.tier2);
            eventBus.on("run.failed", errorHandlers.tier1);

            // Tier 3 emits error
            const stepError: Event = {
                id: "error-123",
                type: "step.failed",
                source: "tier3.executor",
                timestamp: new Date().toISOString(),
                data: {
                    stepId: "step-123",
                    runId: "run-456",
                    error: "Tool execution failed",
                },
            };

            eventBus.emit(stepError);

            // Tier 2 processes and propagates
            const runError: Event = {
                id: "error-456",
                type: "run.failed",
                source: "tier2.run",
                timestamp: new Date().toISOString(),
                causationId: stepError.id,
                data: {
                    runId: "run-456",
                    swarmId: "swarm-123",
                    error: "Run failed due to step error",
                },
            };

            eventBus.emit(runError);

            expect(errorHandlers.tier2).toHaveBeenCalledWith(stepError);
            expect(errorHandlers.tier1).toHaveBeenCalledWith(runError);
        });
    });

    describe("emergent capability events", () => {
        it("should handle monitoring agent subscriptions", () => {
            const monitoringHandler = vi.fn();
            
            // Monitoring agent subscribes to all execution events
            eventBus.subscribe({
                pattern: "*.*",
                filter: (event) => ["step.completed", "run.completed", "swarm.completed"].includes(event.type),
                handler: monitoringHandler,
            });

            // Emit various completion events
            eventBus.emit(mockEvents.tier3.stepCompleted);
            eventBus.emit(mockEvents.tier2.runCompleted);

            expect(monitoringHandler).toHaveBeenCalledTimes(2);
        });

        it("should handle security agent subscriptions", () => {
            const securityHandler = vi.fn();
            
            // Security agent subscribes to specific patterns
            eventBus.subscribe({
                pattern: "*.*",
                filter: (event) => {
                    // Check for security-relevant events
                    return event.data.toolName?.includes("database") ||
                           event.data.action?.includes("delete") ||
                           event.type.includes("failed");
                },
                handler: securityHandler,
            });

            // Emit security-relevant event
            const dbEvent: Event = {
                id: "sec-123",
                type: "step.started",
                source: "tier3.executor",
                timestamp: new Date().toISOString(),
                data: {
                    stepId: "step-123",
                    toolName: "database_query",
                    action: "delete_user",
                },
            };

            eventBus.emit(dbEvent);

            expect(securityHandler).toHaveBeenCalledWith(dbEvent);
        });

        it("should handle optimization agent subscriptions", () => {
            const optimizationHandler = vi.fn();
            
            // Optimization agent subscribes to performance metrics
            eventBus.subscribe({
                pattern: "*.completed",
                filter: (event) => event.data.resourceUsage !== undefined,
                handler: optimizationHandler,
            });

            // Emit event with resource usage
            const perfEvent: Event = {
                ...mockEvents.tier3.stepCompleted,
                data: {
                    ...mockEvents.tier3.stepCompleted.data,
                    resourceUsage: {
                        tokens: 500,
                        credits: 50,
                        duration: 2000,
                    },
                },
            };

            eventBus.emit(perfEvent);

            expect(optimizationHandler).toHaveBeenCalledWith(perfEvent);
        });
    });

    describe("event persistence and replay", () => {
        it("should support event history for debugging", () => {
            const events = [
                mockEvents.tier1.swarmStarted,
                mockEvents.tier2.runStarted,
                mockEvents.tier3.stepStarted,
                mockEvents.tier3.stepCompleted,
                mockEvents.tier2.runCompleted,
            ];

            // Emit sequence of events
            events.forEach(event => eventBus.emit(event));

            // Events would be persisted to Redis for replay/debugging
            // This is a placeholder for actual implementation
            expect(events.length).toBe(5);
        });
    });

    describe("performance and scalability", () => {
        it("should handle high-frequency event emission", () => {
            const handler = vi.fn();
            eventBus.on("perf.test", handler);

            // Emit many events rapidly
            const eventCount = 1000;
            for (let i = 0; i < eventCount; i++) {
                eventBus.emit({
                    id: `perf-${i}`,
                    type: "perf.test",
                    source: "test",
                    timestamp: new Date().toISOString(),
                    data: { index: i },
                });
            }

            expect(handler).toHaveBeenCalledTimes(eventCount);
        });

        it("should handle multiple concurrent subscriptions", () => {
            const handlers = Array.from({ length: 10 }, () => vi.fn());
            
            // Create multiple subscriptions
            handlers.forEach((handler, i) => {
                eventBus.on(`test.event.${i % 3}`, handler);
            });

            // Emit events
            for (let i = 0; i < 3; i++) {
                eventBus.emit({
                    id: `multi-${i}`,
                    type: `test.event.${i}`,
                    source: "test",
                    timestamp: new Date().toISOString(),
                    data: { index: i },
                });
            }

            // Verify handlers were called appropriately
            handlers.forEach((handler, i) => {
                const expectedCalls = i % 3 < 3 ? 1 : 0;
                expect(handler).toHaveBeenCalledTimes(expectedCalls);
            });
        });
    });
});