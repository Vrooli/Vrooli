/**
 * Event Bus Rate Limiter Tests
 * 
 * Tests the rate limiting functionality integrated into the event bus.
 * Verifies that the token bucket algorithm works correctly and events
 * are properly rate limited based on tier, user, and event type.
 */

import { nanoid } from "nanoid";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type Logger } from "winston";
import { getEventBus } from "./eventBus.js";
import { DEFAULT_EVENT_COSTS, DEFAULT_RATE_LIMITS, EventBusRateLimiter } from "./rateLimiter.js";
import {
    type BaseEvent,
    type CoordinationEvent,
    type ExecutionEvent,
} from "./types.js";

// Mock logger
const mockLogger: Logger = {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    debug: vi.fn(),
} as any;

// Mock Redis client
const mockRedisClient = {
    script: vi.fn().mockResolvedValue("mock-script-sha"),
    evalsha: vi.fn(),
};

// Mock CacheService
vi.mock("../../redisConn.js", () => ({
    CacheService: {
        get: () => ({
            raw: () => Promise.resolve(mockRedisClient),
        }),
    },
}));

describe("EventBusRateLimiter", () => {
    let rateLimiter: EventBusRateLimiter;

    beforeEach(() => {
        vi.clearAllMocks();
        rateLimiter = new EventBusRateLimiter(mockLogger);
    });

    afterEach(async () => {
        await getEventBus().stop();
    });

    describe("Rate Limit Configuration", () => {
        it("should have default rate limits for all tiers", () => {
            expect(DEFAULT_RATE_LIMITS.coordination).toBeDefined();
            expect(DEFAULT_RATE_LIMITS.process).toBeDefined();
            expect(DEFAULT_RATE_LIMITS.execution).toBeDefined();
            expect(DEFAULT_RATE_LIMITS.safety.bypass).toBe(true);
        });

        it("should have default event costs configured", () => {
            expect(DEFAULT_EVENT_COSTS.coordination).toBe(1);
            expect(DEFAULT_EVENT_COSTS.process).toBe(5);
            expect(DEFAULT_EVENT_COSTS.execution).toBe(10);
            expect(DEFAULT_EVENT_COSTS.eventTypeMultipliers["tool/called"]).toBe(10);
        });
    });

    describe("Event Type Classification", () => {
        it("should classify coordination events correctly", async () => {
            const coordinationEvent: CoordinationEvent = {
                id: nanoid(),
                type: "swarm/goal/created",
                timestamp: new Date(),
                source: { tier: 1, component: "swarm-coordinator" },
                data: { swarmId: "test-swarm", goal: "test goal" },
                metadata: { deliveryGuarantee: "fire-and-forget", priority: "medium" },
            };

            // Mock rate limit check to return allowed
            mockRedisClient.evalsha.mockResolvedValue([1, 0]); // allowed=1, waitTime=0

            const result = await rateLimiter.checkEventRateLimit(coordinationEvent);
            expect(result.allowed).toBe(true);
        });

        it("should classify execution events correctly", async () => {
            const executionEvent: ExecutionEvent = {
                id: nanoid(),
                type: "tool/called",
                timestamp: new Date(),
                source: { tier: 3, component: "tool-orchestrator" },
                data: { toolName: "expensive_api", parameters: {} },
                metadata: { deliveryGuarantee: "reliable", priority: "high" },
            };

            // Mock rate limit check to return allowed
            mockRedisClient.evalsha.mockResolvedValue([1, 0]);

            const result = await rateLimiter.checkEventRateLimit(executionEvent);
            expect(result.allowed).toBe(true);
        });

        it("should always allow safety events", async () => {
            const safetyEvent: BaseEvent = {
                id: nanoid(),
                type: "safety/pre_action",
                timestamp: new Date(),
                source: { tier: "safety", component: "safety-validator" },
                data: { action: "dangerous_operation", riskLevel: "critical" },
                metadata: {
                    deliveryGuarantee: "barrier-sync",
                    priority: "critical",
                    barrierConfig: {
                        quorum: 1,
                        timeoutMs: 5000,
                        timeoutAction: "auto-reject",
                    },
                },
            };

            const result = await rateLimiter.checkEventRateLimit(safetyEvent);
            expect(result.allowed).toBe(true);
            // Safety events should not even call Redis
            expect(mockRedisClient.evalsha).not.toHaveBeenCalled();
        });
    });

    describe("Rate Limiting Logic", () => {
        it("should rate limit when token bucket is empty", async () => {
            const event: BaseEvent = {
                id: nanoid(),
                type: "tool/called",
                timestamp: new Date(),
                source: { tier: 3, component: "tool-orchestrator" },
                data: { toolName: "expensive_api" },
                metadata: { deliveryGuarantee: "fire-and-forget", priority: "medium" },
            };

            // Mock rate limit check to return denied with wait time
            mockRedisClient.evalsha.mockResolvedValue([0, 5000]); // allowed=0, waitTime=5000ms

            const result = await rateLimiter.checkEventRateLimit(event);
            expect(result.allowed).toBe(false);
            expect(result.retryAfterMs).toBe(5000);
            expect(result.limitType).toBeDefined();
        });

        it("should allow events when tokens are available", async () => {
            const event: BaseEvent = {
                id: nanoid(),
                type: "step/completed",
                timestamp: new Date(),
                source: { tier: 3, component: "step-executor" },
                data: { stepId: "step-123", success: true },
                metadata: { deliveryGuarantee: "fire-and-forget", priority: "low" },
            };

            // Mock rate limit check to return allowed
            mockRedisClient.evalsha.mockResolvedValue([1, 0]);

            const result = await rateLimiter.checkEventRateLimit(event);
            expect(result.allowed).toBe(true);
            expect(result.remainingQuota).toBeDefined();
        });

        it("should handle Redis failures gracefully", async () => {
            const event: BaseEvent = {
                id: nanoid(),
                type: "routine/started",
                timestamp: new Date(),
                source: { tier: 2, component: "routine-orchestrator" },
                data: { routineId: "routine-123" },
                metadata: { deliveryGuarantee: "fire-and-forget", priority: "medium" },
            };

            // Mock Redis failure
            mockRedisClient.evalsha.mockRejectedValue(new Error("Redis connection failed"));

            const result = await rateLimiter.checkEventRateLimit(event);
            // Should fail-closed on Redis errors
            expect(result.allowed).toBe(false);
            expect(result.limitType).toBe("global");
        });
    });

    describe("Event Bus Integration", () => {
        beforeEach(async () => {
            await getEventBus().start();
        });

        it("should publish events when rate limits allow", async () => {
            const event: BaseEvent = {
                id: nanoid(),
                type: "state/changed",
                timestamp: new Date(),
                source: { tier: 2, component: "state-manager" },
                data: { entityId: "entity-123", newState: "active" },
                metadata: { deliveryGuarantee: "fire-and-forget", priority: "low" },
            };

            // Mock rate limit check to return allowed
            mockRedisClient.evalsha.mockResolvedValue([1, 0]);

            const result = await getEventBus().publish(event);
            expect(result.success).toBe(true);
            expect((result as any).rateLimited).toBeUndefined();
        });

        it("should reject events when rate limits are exceeded", async () => {
            const event: BaseEvent = {
                id: nanoid(),
                type: "tool/called",
                timestamp: new Date(),
                source: { tier: 3, component: "tool-orchestrator" },
                data: { toolName: "expensive_api" },
                metadata: {
                    deliveryGuarantee: "fire-and-forget",
                    priority: "medium",
                    userId: "user-123",
                    conversationId: "conv-456",
                },
            };

            // Mock rate limit check to return denied
            mockRedisClient.evalsha.mockResolvedValue([0, 3000]);

            const result = await getEventBus().publish(event);
            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Rate limit exceeded");
        });

        it("should emit rate limited events when limits are exceeded", async () => {
            const subscriptionHandler = vi.fn();

            // Subscribe to rate limited events
            await getEventBus().subscribe("resource/rate_limited", subscriptionHandler);

            const event: BaseEvent = {
                id: nanoid(),
                type: "tool/called",
                timestamp: new Date(),
                source: { tier: 3, component: "tool-orchestrator" },
                data: { toolName: "expensive_api" },
                metadata: {
                    deliveryGuarantee: "fire-and-forget",
                    priority: "medium",
                    userId: "user-123",
                },
            };

            // Mock rate limit check to return denied
            mockRedisClient.evalsha.mockResolvedValue([0, 2000]);

            await getEventBus().publish(event);

            // Wait for async event processing
            await new Promise(resolve => setTimeout(resolve, 10));

            // Should have emitted a rate limited event
            expect(subscriptionHandler).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "resource/rate_limited",
                    data: expect.objectContaining({
                        originalEventId: event.id,
                        originalEventType: event.type,
                        userId: "user-123",
                    }),
                }),
            );
        });

        it("should provide rate limit status", async () => {
            const status = await getEventBus().getRateLimitStatus("user-123", "tool/called");

            expect(status).toHaveProperty("global");
            expect(status.global).toHaveProperty("allowed");
            expect(status.global).toHaveProperty("total");
        });

        it("should include rate limiting metrics", () => {
            const metrics = getEventBus().getMetrics();

            expect(metrics).toHaveProperty("eventsRateLimited");
            expect(metrics).toHaveProperty("rateLimitingEnabled");
            expect(metrics.rateLimitingEnabled).toBe(true);
        });
    });

    describe("User and Conversation Context", () => {
        it("should extract user ID from event metadata", async () => {
            const event: BaseEvent = {
                id: nanoid(),
                type: "swarm/goal/created",
                timestamp: new Date(),
                source: { tier: 1, component: "swarm-coordinator" },
                data: { swarmId: "swarm-123", goal: "test goal" },
                metadata: {
                    deliveryGuarantee: "fire-and-forget",
                    priority: "medium",
                    userId: "user-456",
                },
            };

            mockRedisClient.evalsha.mockResolvedValue([1, 0]);

            await rateLimiter.checkEventRateLimit(event);

            // Should have included user-specific rate limiting keys
            expect(mockRedisClient.evalsha).toHaveBeenCalledWith(
                expect.any(String),
                expect.any(Number),
                expect.arrayContaining([
                    expect.stringContaining("user:user-456"),
                ]),
                expect.any(String),
                expect.any(String),
                expect.any(String),
            );
        });

        it("should extract conversation ID from event data", async () => {
            const event: BaseEvent = {
                id: nanoid(),
                type: "routine/started",
                timestamp: new Date(),
                source: { tier: 2, component: "routine-orchestrator" },
                data: {
                    routineId: "routine-123",
                    conversationId: "conv-789",
                },
                metadata: { deliveryGuarantee: "fire-and-forget", priority: "medium" },
            };

            mockRedisClient.evalsha.mockResolvedValue([1, 0]);

            await rateLimiter.checkEventRateLimit(event);

            // Should have included conversation-specific rate limiting keys
            expect(mockRedisClient.evalsha).toHaveBeenCalledWith(
                expect.any(String),
                expect.any(Number),
                expect.arrayContaining([
                    expect.stringContaining("conversation:conv-789"),
                ]),
                expect.any(String),
                expect.any(String),
                expect.any(String),
            );
        });
    });
});

describe("Event Bus Rate Limiter Integration", () => {
    it("should demonstrate complete rate limiting workflow", async () => {
        const logger = mockLogger;
        const rateLimiter = new EventBusRateLimiter(logger);
        const eventBus = getEventBus();

        await eventBus.start();

        // Mock successful rate limit check
        mockRedisClient.evalsha.mockResolvedValue([1, 0]);

        // Create a high-cost event
        const expensiveEvent: ExecutionEvent = {
            id: nanoid(),
            type: "tool/called",
            timestamp: new Date(),
            source: { tier: 3, component: "tool-orchestrator" },
            data: {
                toolName: "openai_completion",
                parameters: { model: "gpt-4", tokens: 1000 },
            },
            metadata: {
                deliveryGuarantee: "reliable",
                priority: "high",
                userId: "user-123",
                conversationId: "conv-456",
            },
        };

        // Publish the event
        const result = await getEventBus().publish(expensiveEvent);

        // Should succeed with rate limiting info
        expect(result.success).toBe(true);
        expect((result as any).remainingQuota).toBeDefined();
        expect((result as any).resetTime).toBeDefined();

        // Check that rate limiting was invoked
        expect(mockRedisClient.evalsha).toHaveBeenCalled();

        await getEventBus().stop();
    });
});
