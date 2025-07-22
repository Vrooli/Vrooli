/**
 * EventPublisher tests
 * 
 * Tests for the central event publishing interface.
 * Critical for all event-driven communication in the system.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { EventPublisher, aggregateProgression, aggregateReasons } from "./publisher.js";
import type { BotEventResponse } from "./types.js";
import { EventMode } from "./types.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        debug: vi.fn(),
        error: vi.fn(),
        info: vi.fn(),
        warning: vi.fn(),
        warn: vi.fn(),
    },
}));

vi.mock("./eventBus.js", () => ({
    getEventBus: vi.fn(),
}));

vi.mock("./registry.js", () => ({
    getEventBehavior: vi.fn(),
    EventUsageValidator: {
        trackEmission: vi.fn(),
        markProgressionChecked: vi.fn(),
    },
}));

describe("EventPublisher", () => {
    let mockEventBus: any;
    let mockGetEventBehavior: any;
    let mockEventUsageValidator: any;

    beforeEach(() => {
        const { getEventBus } = require("./eventBus.js");
        const { getEventBehavior, EventUsageValidator } = require("./registry.js");

        mockEventBus = {
            publish: vi.fn(),
        };
        getEventBus.mockReturnValue(mockEventBus);

        mockGetEventBehavior = getEventBehavior;
        mockEventUsageValidator = EventUsageValidator;

        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("emit", () => {
        it("should emit simple event successfully", async () => {
            // Setup mocks
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
                defaultPriority: "medium",
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-123",
                progression: "continue",
                wasBlocking: false,
                responses: [],
            });

            const result = await EventPublisher.emit(
                "test/event",
                { testData: "value" },
            );

            expect(result.proceed).toBe(true);
            expect(result.eventId).toBe("event-123");
            expect(result.wasBlocking).toBe(false);
            expect(result.reason).toBeUndefined();
        });

        it("should handle blocked event", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.APPROVAL,
                interceptable: true,
                defaultPriority: "high",
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-456",
                progression: "block",
                wasBlocking: true,
                responses: [{
                    responderId: "bot-123",
                    response: {
                        progression: "block",
                        reason: "Security concern detected",
                    },
                    timestamp: new Date(),
                }],
            });

            const result = await EventPublisher.emit(
                "security/alert",
                { alertType: "suspicious_activity" },
            );

            expect(result.proceed).toBe(false);
            expect(result.eventId).toBe("event-456");
            expect(result.wasBlocking).toBe(true);
            expect(result.reason).toBe("Security concern detected");
        });

        it("should use custom metadata", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.INTERCEPTABLE,
                interceptable: true,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-789",
                progression: "continue",
                wasBlocking: false,
            });

            await EventPublisher.emit(
                "user/action",
                { action: "login" },
                { userId: "user-123", priority: "high" },
            );

            expect(mockEventBus.publish).toHaveBeenCalledWith({
                type: "user/action",
                data: { action: "login" },
                metadata: {
                    priority: "high", // Custom priority
                    userId: "user-123",
                },
                progression: {
                    state: "continue",
                    processedBy: [],
                },
            });
        });

        it("should use default priority when not specified", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
                defaultPriority: "low",
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-default",
                progression: "continue",
            });

            await EventPublisher.emit("test/event", { data: "test" });

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    metadata: expect.objectContaining({
                        priority: "low", // From behavior config
                    }),
                }),
            );
        });

        it("should fall back to medium priority when no default", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
                // No defaultPriority
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-medium",
                progression: "continue",
            });

            await EventPublisher.emit("test/event", { data: "test" });

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    metadata: expect.objectContaining({
                        priority: "medium", // Fallback
                    }),
                }),
            );
        });

        it("should aggregate multiple blocking reasons", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.CONSENSUS,
                interceptable: true,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-multi",
                progression: "block",
                wasBlocking: true,
                responses: [
                    {
                        responderId: "bot-1",
                        response: {
                            progression: "block",
                            reason: "Security violation",
                        },
                        timestamp: new Date(),
                    },
                    {
                        responderId: "bot-2",
                        response: {
                            progression: "block",
                            reason: "Policy violation",
                        },
                        timestamp: new Date(),
                    },
                ],
            });

            const result = await EventPublisher.emit("restricted/action", { action: "delete" });

            expect(result.proceed).toBe(false);
            expect(result.reason).toBe("Security violation; Policy violation");
        });

        it("should handle empty responses gracefully", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.APPROVAL,
                interceptable: true,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-empty",
                progression: "block",
                wasBlocking: true,
                responses: [],
            });

            const result = await EventPublisher.emit("blocked/event", { data: "test" });

            expect(result.proceed).toBe(false);
            expect(result.reason).toBe("No specific reason provided");
        });

        it("should handle responses without reasons", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.APPROVAL,
                interceptable: true,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-no-reason",
                progression: "block",
                wasBlocking: true,
                responses: [{
                    responderId: "bot-silent",
                    response: {
                        progression: "block",
                        // No reason provided
                    },
                    timestamp: new Date(),
                }],
            });

            const result = await EventPublisher.emit("silent/block", { data: "test" });

            expect(result.proceed).toBe(false);
            expect(result.reason).toBe("Event was blocked but no reasons were provided");
        });

        it("should handle missing event ID", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                // No eventId
                progression: "continue",
            });

            const result = await EventPublisher.emit("no/id/event", { data: "test" });

            expect(result.eventId).toBe("");
        });
    });

    describe("development mode tracking", () => {
        beforeEach(() => {
            process.env.NODE_ENV = "development";
        });

        afterEach(() => {
            process.env.NODE_ENV = "test";
        });

        it("should track event emission in development", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.INTERCEPTABLE,
                interceptable: true,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-tracked",
                progression: "continue",
                wasBlocking: false,
            });

            await EventPublisher.emit("tracked/event", { data: "test" });

            expect(mockEventUsageValidator.trackEmission).toHaveBeenCalledWith(
                "tracked/event",
                "event-tracked",
                false,
            );
        });

        it("should mark progression as checked when proceed is accessed", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-check",
                progression: "continue",
            });

            const result = await EventPublisher.emit("check/event", { data: "test" });

            // Access the proceed property
            const shouldProceed = result.proceed;

            expect(shouldProceed).toBe(true);
            expect(mockEventUsageValidator.markProgressionChecked).toHaveBeenCalledWith("event-check");
        });

        it("should not track if no event ID", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                // No eventId
                progression: "continue",
            });

            await EventPublisher.emit("no/track/event", { data: "test" });

            expect(mockEventUsageValidator.trackEmission).not.toHaveBeenCalled();
        });
    });

    describe("aggregateProgression", () => {
        it("should return continue for empty responses", () => {
            const result = aggregateProgression([]);
            expect(result).toBe("continue");
        });

        it("should block immediately when blockOnFirst is true", () => {
            const responses: BotEventResponse[] = [
                { progression: "continue", reason: "OK" },
                { progression: "block", reason: "Security issue" },
                { progression: "continue", reason: "Also OK" },
            ];

            const result = aggregateProgression(responses, { blockOnFirst: true });
            expect(result).toBe("block");
        });

        it("should respect continue threshold", () => {
            const responses: BotEventResponse[] = [
                { progression: "continue", reason: "OK" },
                { progression: "continue", reason: "Also OK" },
                { progression: "block", reason: "Issue" },
            ];

            const result = aggregateProgression(responses, { continueThreshold: 2 });
            expect(result).toBe("continue");
        });

        it("should block when continue threshold not met", () => {
            const responses: BotEventResponse[] = [
                { progression: "continue", reason: "OK" },
                { progression: "block", reason: "Issue 1" },
                { progression: "block", reason: "Issue 2" },
            ];

            const result = aggregateProgression(responses, { continueThreshold: 2 });
            expect(result).toBe("block");
        });

        it("should use majority rule by default", () => {
            const responses: BotEventResponse[] = [
                { progression: "continue", reason: "OK" },
                { progression: "continue", reason: "Also OK" },
                { progression: "block", reason: "Issue" },
            ];

            const result = aggregateProgression(responses);
            expect(result).toBe("continue"); // 2 vs 1
        });

        it("should block on ties for safety", () => {
            const responses: BotEventResponse[] = [
                { progression: "continue", reason: "OK" },
                { progression: "block", reason: "Issue" },
            ];

            const result = aggregateProgression(responses);
            expect(result).toBe("block"); // 50-50 tie
        });

        it("should handle defer majority", () => {
            const responses: BotEventResponse[] = [
                { progression: "defer", reason: "Wait" },
                { progression: "defer", reason: "Also wait" },
                { progression: "continue", reason: "OK" },
            ];

            const result = aggregateProgression(responses);
            expect(result).toBe("defer");
        });

        it("should handle retry majority", () => {
            const responses: BotEventResponse[] = [
                { progression: "retry", reason: "Try again" },
                { progression: "retry", reason: "Also retry" },
                { progression: "continue", reason: "OK" },
            ];

            const result = aggregateProgression(responses);
            expect(result).toBe("retry");
        });

        it("should fall back to block for safety", () => {
            const responses: BotEventResponse[] = [
                { progression: "defer", reason: "Wait" },
                { progression: "retry", reason: "Try again" },
                { progression: "continue", reason: "OK" },
                { progression: "block", reason: "Stop" },
            ];

            const result = aggregateProgression(responses);
            expect(result).toBe("block"); // No clear majority
        });
    });

    describe("aggregateReasons", () => {
        it("should return undefined for empty responses", () => {
            const result = aggregateReasons([]);
            expect(result).toBeUndefined();
        });

        it("should return undefined for responses without reasons", () => {
            const responses: BotEventResponse[] = [
                { progression: "continue" },
                { progression: "block" },
            ];

            const result = aggregateReasons(responses);
            expect(result).toBeUndefined();
        });

        it("should format single reason with progression", () => {
            const responses: BotEventResponse[] = [
                { progression: "block", reason: "Security violation" },
            ];

            const result = aggregateReasons(responses);
            expect(result).toBe("block: Security violation");
        });

        it("should format multiple reasons with progressions", () => {
            const responses: BotEventResponse[] = [
                { progression: "block", reason: "Security violation" },
                { progression: "defer", reason: "Waiting for approval" },
                { progression: "continue" }, // No reason
            ];

            const result = aggregateReasons(responses);
            expect(result).toBe("block: Security violation; defer: Waiting for approval");
        });

        it("should filter out empty reasons", () => {
            const responses: BotEventResponse[] = [
                { progression: "block", reason: "" },
                { progression: "defer", reason: "Valid reason" },
                { progression: "continue", reason: undefined },
            ];

            const result = aggregateReasons(responses);
            expect(result).toBe("defer: Valid reason");
        });
    });

    describe("error handling and edge cases", () => {
        it("should handle event bus publish failure", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
            });

            mockEventBus.publish.mockRejectedValue(new Error("Event bus error"));

            await expect(EventPublisher.emit("failing/event", { data: "test" }))
                .rejects.toThrow("Event bus error");
        });

        it("should handle invalid progression in publish result", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-invalid",
                progression: "invalid-progression" as any,
            });

            const result = await EventPublisher.emit("invalid/event", { data: "test" });

            expect(result.proceed).toBe(false); // Should be safe and not proceed
        });

        it("should handle null/undefined event data", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.PASSIVE,
                interceptable: false,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-null",
                progression: "continue",
            });

            const result1 = await EventPublisher.emit("null/event", null as any);
            const result2 = await EventPublisher.emit("undefined/event", undefined as any);

            expect(result1.proceed).toBe(true);
            expect(result2.proceed).toBe(true);
        });

        it("should handle malformed responses in aggregation", async () => {
            mockGetEventBehavior.mockReturnValue({
                mode: EventMode.APPROVAL,
                interceptable: true,
            });

            mockEventBus.publish.mockResolvedValue({
                success: true,
                eventId: "event-malformed",
                progression: "block",
                responses: [
                    null, // Invalid response
                    {
                        responderId: "bot-1",
                        response: {
                            progression: "block",
                            reason: "Valid reason",
                        },
                        timestamp: new Date(),
                    },
                    undefined, // Invalid response
                ],
            });

            const result = await EventPublisher.emit("malformed/event", { data: "test" });

            expect(result.proceed).toBe(false);
            expect(result.reason).toBe("Valid reason"); // Should handle only valid responses
        });
    });
});
