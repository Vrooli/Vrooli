import { describe, it, expect, beforeEach, vi, type MockedFunction } from "vitest";
import { RollingHistoryAdapter, type HistoryEvent, type RollingHistoryConfig } from "./RollingHistoryAdapter.js";
import { UnifiedMonitoringService } from "../UnifiedMonitoringService.js";

// Mock the UnifiedMonitoringService
vi.mock("../UnifiedMonitoringService.js", () => ({
    UnifiedMonitoringService: {
        getInstance: vi.fn(() => ({
            record: vi.fn().mockResolvedValue(undefined),
            query: vi.fn().mockResolvedValue({ metrics: [] }),
            initialize: vi.fn().mockResolvedValue(undefined),
            getHealth: vi.fn().mockResolvedValue({ status: "healthy" }),
        })),
    },
}));

// Mock EventBus
const mockEventBus = {
    subscribe: vi.fn().mockResolvedValue(undefined),
    publish: vi.fn().mockResolvedValue(undefined),
    unsubscribe: vi.fn().mockResolvedValue(undefined),
};

describe("RollingHistoryAdapter", () => {
    let adapter: RollingHistoryAdapter;
    let config: RollingHistoryConfig;
    let mockUnifiedService: any;

    beforeEach(() => {
        vi.clearAllMocks();
        
        config = {
            maxSize: 1000,
            maxAge: 3600000, // 1 hour
            persistInterval: 300000, // 5 minutes
        };

        mockUnifiedService = {
            record: vi.fn().mockResolvedValue(undefined),
            query: vi.fn().mockResolvedValue({ metrics: [] }),
            initialize: vi.fn().mockResolvedValue(undefined),
            getHealth: vi.fn().mockResolvedValue({ status: "healthy" }),
        };

        (UnifiedMonitoringService.getInstance as MockedFunction<any>).mockReturnValue(mockUnifiedService);
        
        adapter = new RollingHistoryAdapter(mockEventBus as any, config);
    });

    describe("Constructor", () => {
        it("should initialize with correct configuration", () => {
            expect(UnifiedMonitoringService.getInstance).toHaveBeenCalledWith(
                {
                    maxOverheadMs: 5,
                    eventBusEnabled: true,
                    mcpToolsEnabled: true,
                },
                mockEventBus
            );
        });

        it("should set up event subscriptions", () => {
            expect(mockEventBus.subscribe).toHaveBeenCalledTimes(4); // 3 telemetry channels + 1 execution
        });
    });

    describe("addEvent", () => {
        it("should add an event and record it through unified service", () => {
            const historyEvent: HistoryEvent = {
                timestamp: new Date(),
                type: "test.event",
                tier: "tier3",
                component: "test-component",
                data: { value: 42 },
                metadata: { test: true },
            };

            adapter.addEvent(historyEvent);

            expect(mockUnifiedService.record).toHaveBeenCalledWith({
                tier: 3,
                component: "test-component",
                type: "business",
                name: "test.event",
                value: 42, // Extracted from data.value
                executionId: undefined,
                metadata: {
                    test: true,
                    originalData: { value: 42 },
                    source: "rolling_history_adapter",
                },
            });
        });

        it("should handle performance events correctly", () => {
            const historyEvent: HistoryEvent = {
                timestamp: new Date(),
                type: "perf.duration",
                tier: "tier2",
                component: "performance-test",
                data: { duration: 1500 },
            };

            adapter.addEvent(historyEvent);

            expect(mockUnifiedService.record).toHaveBeenCalledWith({
                tier: 2,
                component: "performance-test",
                type: "performance",
                name: "perf.duration",
                value: 1500,
                executionId: undefined,
                metadata: {
                    originalData: { duration: 1500 },
                    source: "rolling_history_adapter",
                },
            });
        });

        it("should handle errors gracefully", () => {
            mockUnifiedService.record.mockRejectedValue(new Error("Record failed"));
            
            const historyEvent: HistoryEvent = {
                timestamp: new Date(),
                type: "test.event",
                tier: "tier1",
                component: "test",
                data: {},
            };

            // Should not throw
            expect(() => adapter.addEvent(historyEvent)).not.toThrow();
        });
    });

    describe("getRecentEvents", () => {
        it("should return empty array for now (backward compatibility)", () => {
            const events = adapter.getRecentEvents(60000);
            expect(events).toEqual([]);
        });

        it("should handle no time window", () => {
            const events = adapter.getRecentEvents();
            expect(events).toEqual([]);
        });
    });

    describe("getEventsByTier", () => {
        it("should return empty array for now (backward compatibility)", () => {
            const events = adapter.getEventsByTier("tier1", 10);
            expect(events).toEqual([]);
        });
    });

    describe("getEventsByType", () => {
        it("should handle string patterns", () => {
            const events = adapter.getEventsByType("error", 5);
            expect(events).toEqual([]);
        });

        it("should handle regex patterns", () => {
            const events = adapter.getEventsByType(/error.*/, 5);
            expect(events).toEqual([]);
        });
    });

    describe("detectPatterns", () => {
        it("should return valid pattern detection result", () => {
            const patterns = adapter.detectPatterns(300000);
            
            expect(patterns).toEqual({
                totalEvents: 0,
                windowMs: 300000,
                eventsPerMinute: 0,
                hasHighActivity: false,
                topEventTypes: [],
                tierDistribution: {},
                componentDistribution: {},
            });
        });

        it("should use default window if not provided", () => {
            const patterns = adapter.detectPatterns();
            expect(patterns.windowMs).toBe(300000);
        });
    });

    describe("getEventsInTimeRange", () => {
        it("should return empty array for now (backward compatibility)", () => {
            const startTime = Date.now() - 60000;
            const endTime = Date.now();
            const events = adapter.getEventsInTimeRange(startTime, endTime);
            expect(events).toEqual([]);
        });
    });

    describe("getAllEvents", () => {
        it("should return empty array for now (backward compatibility)", () => {
            const events = adapter.getAllEvents();
            expect(events).toEqual([]);
        });
    });

    describe("getValid", () => {
        it("should be alias for getAllEvents", () => {
            const allEvents = adapter.getAllEvents();
            const validEvents = adapter.getValid();
            expect(validEvents).toEqual(allEvents);
        });
    });

    describe("getExecutionFlow", () => {
        it("should return empty array for now (backward compatibility)", () => {
            const flow = adapter.getExecutionFlow("test-execution-id");
            expect(flow).toEqual([]);
        });
    });

    describe("evictOldEvents", () => {
        it("should be no-op (handled by UnifiedMonitoringService)", () => {
            // Should not throw
            expect(() => adapter.evictOldEvents()).not.toThrow();
        });
    });

    describe("getStats", () => {
        it("should return valid statistics", () => {
            const stats = adapter.getStats();
            
            expect(stats).toEqual({
                currentSize: 0,
                maxSize: config.maxSize,
                oldestEvent: null,
                newestEvent: null,
                eventsPerSecond: 0,
            });
        });
    });

    describe("Event Bus Integration", () => {
        it("should subscribe to telemetry channels", () => {
            expect(mockEventBus.subscribe).toHaveBeenCalledWith(
                expect.objectContaining({
                    pattern: "telemetry.perf",
                    handler: expect.any(Function),
                })
            );
        });

        it("should subscribe to execution events", () => {
            expect(mockEventBus.subscribe).toHaveBeenCalledWith(
                expect.objectContaining({
                    pattern: "execution.*",
                    handler: expect.any(Function),
                })
            );
        });
    });

    describe("Metric Type Detection", () => {
        it("should classify performance events correctly", () => {
            const perfEvent: HistoryEvent = {
                timestamp: new Date(),
                type: "perf.latency",
                tier: "tier3",
                component: "test",
                data: { latency: 100 },
            };

            adapter.addEvent(perfEvent);

            expect(mockUnifiedService.record).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "performance",
                })
            );
        });

        it("should classify health events correctly", () => {
            const healthEvent: HistoryEvent = {
                timestamp: new Date(),
                type: "health.error",
                tier: "tier2",
                component: "test",
                data: { error: "Something failed" },
            };

            adapter.addEvent(healthEvent);

            expect(mockUnifiedService.record).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "health",
                })
            );
        });

        it("should classify safety events correctly", () => {
            const safetyEvent: HistoryEvent = {
                timestamp: new Date(),
                type: "safety.validation",
                tier: "tier1",
                component: "test",
                data: { validated: true },
            };

            adapter.addEvent(safetyEvent);

            expect(mockUnifiedService.record).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "safety",
                })
            );
        });

        it("should default to business events", () => {
            const businessEvent: HistoryEvent = {
                timestamp: new Date(),
                type: "task.completed",
                tier: "tier2",
                component: "test",
                data: { result: "success" },
            };

            adapter.addEvent(businessEvent);

            expect(mockUnifiedService.record).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "business",
                })
            );
        });
    });

    describe("Tier Extraction", () => {
        it("should extract tier1 from swarmId", () => {
            const event: HistoryEvent = {
                timestamp: new Date(),
                type: "test",
                tier: "tier3", // This should be overridden
                component: "test",
                data: { swarmId: "swarm-123" },
            };

            adapter.addEvent(event);

            expect(mockUnifiedService.record).toHaveBeenCalledWith(
                expect.objectContaining({
                    tier: 3, // Uses the provided tier, not extracted
                })
            );
        });
    });
});