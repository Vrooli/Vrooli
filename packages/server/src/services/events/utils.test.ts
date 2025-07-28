/**
 * Event utils tests
 * 
 * Tests for shared event system utilities.
 * These utilities are used throughout the event system.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import {
    DEFAULT_RETRY_STRATEGIES,
    withRetry,
    delay,
    EventPatternMatcher,
    createPatternMatcher,
    BatchPatternMatcher,
    validateEventStructure,
    getTierFromEventType,
    createDefaultMetadata,
    formatEventForLogging,
    createBarrierConfig,
    calculatePriorityScore,
    type RetryStrategy,
} from "./utils.js";
import type { ServiceEvent } from "./types.js";
import { EVENT_BUS_CONSTANTS, PRIORITY_LEVELS } from "./constants.js";
import { generatePK } from "@vrooli/shared";

describe("Event Utils", () => {
    describe("retry strategies", () => {
        describe("DEFAULT_RETRY_STRATEGIES", () => {
            it("should create linear retry strategy", () => {
                const strategy = DEFAULT_RETRY_STRATEGIES.linear(500);
                
                expect(strategy.maxAttempts).toBe(3);
                expect(strategy.backoffMs(1)).toBe(500);
                expect(strategy.backoffMs(2)).toBe(500);
                expect(strategy.backoffMs(3)).toBe(500);
            });

            it("should create exponential retry strategy", () => {
                const strategy = DEFAULT_RETRY_STRATEGIES.exponential(100);
                
                expect(strategy.maxAttempts).toBe(3);
                expect(strategy.backoffMs(1)).toBe(100); // 100 * 2^0
                expect(strategy.backoffMs(2)).toBe(200); // 100 * 2^1
                expect(strategy.backoffMs(3)).toBe(400); // 100 * 2^2
            });

            it("should create no-retry strategy", () => {
                const strategy = DEFAULT_RETRY_STRATEGIES.none();
                
                expect(strategy.maxAttempts).toBe(1);
                expect(strategy.backoffMs(1)).toBe(0);
            });

            it("should use default base delay for exponential", () => {
                const strategy = DEFAULT_RETRY_STRATEGIES.exponential();
                
                expect(strategy.backoffMs(1)).toBe(EVENT_BUS_CONSTANTS.RETRY_BASE_DELAY_MS);
            });
        });

        describe("withRetry", () => {
            beforeEach(() => {
                vi.useFakeTimers();
            });

            afterEach(() => {
                vi.useRealTimers();
            });

            it("should succeed on first attempt", async () => {
                const mockFn = vi.fn().mockResolvedValue("success");
                const strategy: RetryStrategy = { maxAttempts: 3, backoffMs: () => 100 };

                const result = await withRetry(mockFn, strategy);

                expect(result).toBe("success");
                expect(mockFn).toHaveBeenCalledTimes(1);
            });

            it("should retry on failure and eventually succeed", async () => {
                const mockFn = vi.fn()
                    .mockRejectedValueOnce(new Error("First fail"))
                    .mockRejectedValueOnce(new Error("Second fail"))
                    .mockResolvedValueOnce("success");

                const strategy: RetryStrategy = { maxAttempts: 3, backoffMs: () => 100 };

                const resultPromise = withRetry(mockFn, strategy);
                
                // Advance timers to resolve delays
                await vi.runAllTimersAsync();

                const result = await resultPromise;

                expect(result).toBe("success");
                expect(mockFn).toHaveBeenCalledTimes(3);
            });

            it("should fail after max attempts", async () => {
                const error = new Error("Persistent error");
                const mockFn = vi.fn().mockRejectedValue(error);
                const strategy: RetryStrategy = { maxAttempts: 2, backoffMs: () => 100 };

                const resultPromise = withRetry(mockFn, strategy);
                await vi.runAllTimersAsync();

                await expect(resultPromise).rejects.toThrow("Persistent error");
                expect(mockFn).toHaveBeenCalledTimes(2);
            });

            it("should respect shouldRetry predicate", async () => {
                const mockFn = vi.fn().mockRejectedValue(new Error("Network error"));
                const strategy: RetryStrategy = {
                    maxAttempts: 3,
                    backoffMs: () => 100,
                    shouldRetry: (error) => !error.message.includes("Network"),
                };

                const resultPromise = withRetry(mockFn, strategy);
                await vi.runAllTimersAsync();

                await expect(resultPromise).rejects.toThrow("Network error");
                expect(mockFn).toHaveBeenCalledTimes(1); // Should not retry
            });

            it("should log retry attempts", async () => {
                const mockLogger = { warn: vi.fn() };
                const mockFn = vi.fn()
                    .mockRejectedValueOnce(new Error("Retry error"))
                    .mockResolvedValueOnce("success");

                const strategy: RetryStrategy = { maxAttempts: 2, backoffMs: () => 200 };

                const resultPromise = withRetry(mockFn, strategy, mockLogger);
                await vi.runAllTimersAsync();

                await resultPromise;

                expect(mockLogger.warn).toHaveBeenCalledWith(
                    "Retry attempt 1/2 after 200ms",
                    {
                        error: "Retry error",
                        attempt: 1,
                        delayMs: 200,
                    },
                );
            });

            it("should handle unknown error", async () => {
                const mockFn = vi.fn().mockImplementation(() => {
                    throw undefined; // Weird but possible
                });
                const strategy: RetryStrategy = { maxAttempts: 1, backoffMs: () => 0 };

                await expect(withRetry(mockFn, strategy)).rejects.toThrow("Retry failed with unknown error");
            });
        });

        describe("delay", () => {
            beforeEach(() => {
                vi.useFakeTimers();
            });

            afterEach(() => {
                vi.useRealTimers();
            });

            it("should delay for specified time", async () => {
                const promise = delay(1000);
                
                expect(vi.getTimerCount()).toBe(1);
                
                vi.advanceTimersByTime(999);
                
                // Should not have resolved yet
                let resolved = false;
                promise.then(() => { resolved = true; });
                await Promise.resolve(); // Let promise callbacks run
                expect(resolved).toBe(false);

                vi.advanceTimersByTime(1);
                await promise;
                
                expect(resolved).toBe(true);
            });
        });
    });

    describe("EventPatternMatcher", () => {
        describe("exact patterns", () => {
            it("should match exact event types", () => {
                const matcher = new EventPatternMatcher("finance/transaction/completed");
                
                expect(matcher.matches("finance/transaction/completed")).toBe(true);
                expect(matcher.matches("finance/transaction/failed")).toBe(false);
                expect(matcher.matches("finance/payment/completed")).toBe(false);
            });
        });

        describe("wildcard patterns", () => {
            it("should match single-level wildcards", () => {
                const matcher = new EventPatternMatcher("finance/*/completed");
                
                expect(matcher.matches("finance/transaction/completed")).toBe(true);
                expect(matcher.matches("finance/payment/completed")).toBe(true);
                expect(matcher.matches("finance/transfer/completed")).toBe(true);
                expect(matcher.matches("finance/sub/level/completed")).toBe(false); // Multiple levels
                expect(matcher.matches("chat/message/completed")).toBe(false); // Wrong prefix
            });

            it("should match multi-level wildcards at end", () => {
                const matcher = new EventPatternMatcher("finance/#");
                
                expect(matcher.matches("finance/transaction")).toBe(true);
                expect(matcher.matches("finance/transaction/completed")).toBe(true);
                expect(matcher.matches("finance/payment/processed/audit")).toBe(true);
                expect(matcher.matches("chat/message")).toBe(false);
                expect(matcher.matches("finance")).toBe(true); // Edge case: without trailing slash
            });

            it("should handle catch-all pattern", () => {
                const matcher = new EventPatternMatcher("#");
                
                expect(matcher.matches("anything")).toBe(true);
                expect(matcher.matches("finance/transaction/completed")).toBe(true);
                expect(matcher.matches("")).toBe(true);
                expect(matcher.matches("multi/level/deep/event")).toBe(true);
            });

            it("should combine wildcards", () => {
                const matcher = new EventPatternMatcher("*/transaction/*");
                
                expect(matcher.matches("finance/transaction/completed")).toBe(true);
                expect(matcher.matches("banking/transaction/failed")).toBe(true);
                expect(matcher.matches("finance/payment/completed")).toBe(false);
                expect(matcher.matches("finance/transaction/audit/completed")).toBe(false); // Too many levels
            });
        });

        describe("edge cases", () => {
            it("should handle empty pattern", () => {
                const matcher = new EventPatternMatcher("");
                
                expect(matcher.matches("")).toBe(true);
                expect(matcher.matches("anything")).toBe(false);
            });

            it("should escape regex special characters", () => {
                const matcher = new EventPatternMatcher("test.event+with$special^chars");
                
                expect(matcher.matches("test.event+with$special^chars")).toBe(true);
                expect(matcher.matches("testXeventXwithXspecialXchars")).toBe(false);
            });

            it("should handle slashes in patterns", () => {
                const matcher = new EventPatternMatcher("a/b/c");
                
                expect(matcher.matches("a/b/c")).toBe(true);
                expect(matcher.matches("a-b-c")).toBe(false);
            });
        });

        describe("getPattern", () => {
            it("should return original pattern", () => {
                const pattern = "finance/*/completed";
                const matcher = new EventPatternMatcher(pattern);
                
                expect(matcher.getPattern()).toBe(pattern);
            });
        });
    });

    describe("createPatternMatcher", () => {
        it("should create EventPatternMatcher instance", () => {
            const matcher = createPatternMatcher("test/pattern");
            
            expect(matcher).toBeInstanceOf(EventPatternMatcher);
            expect(matcher.getPattern()).toBe("test/pattern");
        });
    });

    describe("BatchPatternMatcher", () => {
        it("should match any pattern in the batch", () => {
            const matcher = new BatchPatternMatcher([
                "finance/*",
                "banking/*/completed",
                "audit/#",
            ]);
            
            expect(matcher.matches("finance/transaction")).toBe(true);
            expect(matcher.matches("banking/transfer/completed")).toBe(true);
            expect(matcher.matches("audit/security/check")).toBe(true);
            expect(matcher.matches("chat/message")).toBe(false);
        });

        it("should handle empty pattern list", () => {
            const matcher = new BatchPatternMatcher([]);
            
            expect(matcher.matches("any/event")).toBe(false);
        });

        it("should return all patterns", () => {
            const patterns = ["finance/*", "banking/*", "audit/#"];
            const matcher = new BatchPatternMatcher(patterns);
            
            expect(matcher.getPatterns()).toEqual(patterns);
        });
    });

    describe("validateEventStructure", () => {
        const validEvent: ServiceEvent = {
            id: generatePK().toString(),
            type: "test/event",
            timestamp: new Date(),
            data: { test: "data" },
        };

        it("should validate correct event structure", () => {
            expect(validateEventStructure(validEvent)).toBe(true);
        });

        it("should reject non-objects", () => {
            expect(validateEventStructure(null)).toBe(false);
            expect(validateEventStructure(undefined)).toBe(false);
            expect(validateEventStructure("string")).toBe(false);
            expect(validateEventStructure(123)).toBe(false);
        });

        it("should reject missing required fields", () => {
            expect(validateEventStructure({})).toBe(false);
            expect(validateEventStructure({ id: "123" })).toBe(false);
            expect(validateEventStructure({ id: "123", type: "test" })).toBe(false);
            expect(validateEventStructure({ 
                id: "123", 
                type: "test", 
                timestamp: "not-a-date",
                data: {}, 
            })).toBe(false);
        });

        it("should accept events with optional fields", () => {
            const eventWithOptionals: ServiceEvent = {
                ...validEvent,
                metadata: { priority: "high" },
                progression: { state: "continue", processedBy: [] },
            };
            
            expect(validateEventStructure(eventWithOptionals)).toBe(true);
        });

        it("should reject invalid optional field types", () => {
            expect(validateEventStructure({
                ...validEvent,
                metadata: "invalid",
            })).toBe(false);

            expect(validateEventStructure({
                ...validEvent,
                progression: "invalid",
            })).toBe(false);
        });
    });

    describe("getTierFromEventType", () => {
        it("should identify tier 1 events", () => {
            expect(getTierFromEventType("swarm/created")).toBe(1);
            expect(getTierFromEventType("goal/completed")).toBe(1);
            expect(getTierFromEventType("team/member/added")).toBe(1);
            expect(getTierFromEventType("resource/allocated")).toBe(1);
        });

        it("should identify tier 2 events", () => {
            expect(getTierFromEventType("routine/started")).toBe(2);
            expect(getTierFromEventType("state/changed")).toBe(2);
            expect(getTierFromEventType("context/updated")).toBe(2);
        });

        it("should identify tier 3 events", () => {
            expect(getTierFromEventType("step/completed")).toBe(3);
            expect(getTierFromEventType("tool/executed")).toBe(3);
            expect(getTierFromEventType("strategy/selected")).toBe(3);
        });

        it("should identify safety events", () => {
            expect(getTierFromEventType("safety/violation")).toBe("safety");
            expect(getTierFromEventType("emergency/detected")).toBe("safety");
            expect(getTierFromEventType("threat/identified")).toBe("safety");
        });

        it("should identify cross-cutting events", () => {
            expect(getTierFromEventType("execution/failed")).toBe("cross-cutting");
            expect(getTierFromEventType("recovery/initiated")).toBe("cross-cutting");
            expect(getTierFromEventType("fallback/activated")).toBe("cross-cutting");
            expect(getTierFromEventType("circuit_breaker/opened")).toBe("cross-cutting");
        });

        it("should return undefined for unknown events", () => {
            expect(getTierFromEventType("unknown/event")).toBeUndefined();
            expect(getTierFromEventType("chat/message")).toBeUndefined();
            expect(getTierFromEventType("")).toBeUndefined();
        });
    });

    describe("createDefaultMetadata", () => {
        it("should set critical priority for safety events", () => {
            expect(createDefaultMetadata("safety/violation")).toEqual({
                priority: PRIORITY_LEVELS.CRITICAL,
            });

            expect(createDefaultMetadata("emergency/alert")).toEqual({
                priority: PRIORITY_LEVELS.CRITICAL,
            });
        });

        it("should set high priority for approval events", () => {
            expect(createDefaultMetadata("tool/approval_required")).toEqual({
                priority: PRIORITY_LEVELS.HIGH,
            });

            expect(createDefaultMetadata("finance/transfer/approval_required")).toEqual({
                priority: PRIORITY_LEVELS.HIGH,
            });
        });

        it("should default to medium priority", () => {
            expect(createDefaultMetadata("routine/completed")).toEqual({
                priority: PRIORITY_LEVELS.MEDIUM,
            });

            expect(createDefaultMetadata("unknown/event")).toEqual({
                priority: PRIORITY_LEVELS.MEDIUM,
            });
        });
    });

    describe("formatEventForLogging", () => {
        it("should format event with all fields", () => {
            const eventId = generatePK().toString();
            const runId = generatePK().toString();
            const swarmId = generatePK().toString();
            const toolCallId = generatePK().toString();
            
            const event: ServiceEvent = {
                id: eventId,
                type: "test/event",
                timestamp: new Date("2023-01-01T00:00:00Z"),
                data: { key1: "value1", key2: "value2" },
                metadata: { priority: "high" },
                progression: { state: "continue", processedBy: [] },
                execution: {
                    runId,
                    parentSwarmId: swarmId,
                    toolCallId,
                },
            };

            const formatted = formatEventForLogging(event);

            expect(formatted).toEqual({
                id: eventId,
                type: "test/event",
                timestamp: "2023-01-01T00:00:00.000Z",
                metadata: { priority: "high" },
                dataKeys: ["key1", "key2"],
                progression: "continue",
                execution: {
                    runId,
                    parentSwarmId: swarmId,
                    toolCallId,
                },
            });
        });

        it("should handle minimal event", () => {
            const simpleEventId = generatePK().toString();
            
            const event: ServiceEvent = {
                id: simpleEventId,
                type: "simple/event",
                timestamp: new Date("2023-01-01T00:00:00Z"),
                data: "string data",
            };

            const formatted = formatEventForLogging(event);

            expect(formatted).toEqual({
                id: "simple-event",
                type: "simple/event",
                timestamp: "2023-01-01T00:00:00.000Z",
                metadata: undefined,
                dataKeys: undefined, // Non-object data
                progression: undefined,
                execution: undefined,
            });
        });

        it("should handle null/undefined data", () => {
            const event: ServiceEvent = {
                id: "null-data",
                type: "null/event",
                timestamp: new Date("2023-01-01T00:00:00Z"),
                data: null,
            };

            const formatted = formatEventForLogging(event);

            expect(formatted.dataKeys).toBeUndefined();
        });
    });

    describe("createBarrierConfig", () => {
        it("should create config with defaults", () => {
            const config = createBarrierConfig();

            expect(config).toEqual({
                quorum: 1,
                timeoutMs: EVENT_BUS_CONSTANTS.DEFAULT_BARRIER_TIMEOUT_MS,
                timeoutAction: "block",
                requiredResponders: undefined,
            });
        });

        it("should override defaults with provided options", () => {
            const config = createBarrierConfig({
                quorum: "all",
                timeoutMs: 5000,
                timeoutAction: "continue",
                requiredResponders: ["security-bot"],
            });

            expect(config).toEqual({
                quorum: "all",
                timeoutMs: 5000,
                timeoutAction: "continue",
                requiredResponders: ["security-bot"],
            });
        });

        it("should partially override defaults", () => {
            const config = createBarrierConfig({
                quorum: 3,
                timeoutAction: "defer",
            });

            expect(config).toEqual({
                quorum: 3,
                timeoutMs: EVENT_BUS_CONSTANTS.DEFAULT_BARRIER_TIMEOUT_MS,
                timeoutAction: "defer",
                requiredResponders: undefined,
            });
        });
    });

    describe("calculatePriorityScore", () => {
        it("should calculate base priority scores", () => {
            const lowEvent: ServiceEvent = {
                id: "1", type: "test", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.LOW },
            };

            const mediumEvent: ServiceEvent = {
                id: "2", type: "test", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.MEDIUM },
            };

            const highEvent: ServiceEvent = {
                id: "3", type: "test", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.HIGH },
            };

            const criticalEvent: ServiceEvent = {
                id: "4", type: "test", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.CRITICAL },
            };

            expect(calculatePriorityScore(lowEvent)).toBe(1);
            expect(calculatePriorityScore(mediumEvent)).toBe(10);
            expect(calculatePriorityScore(highEvent)).toBe(100);
            expect(calculatePriorityScore(criticalEvent)).toBe(1000);
        });

        it("should boost safety events", () => {
            const safetyEvent: ServiceEvent = {
                id: "safety", type: "safety/violation", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.MEDIUM },
            };

            const emergencyEvent: ServiceEvent = {
                id: "emergency", type: "emergency/alert", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.MEDIUM },
            };

            expect(calculatePriorityScore(safetyEvent)).toBe(10 + 500); // medium + safety boost
            expect(calculatePriorityScore(emergencyEvent)).toBe(10 + 500); // medium + safety boost
        });

        it("should boost approval events", () => {
            const approvalEvent: ServiceEvent = {
                id: "approval", type: "tool/approval_required", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.MEDIUM },
            };

            expect(calculatePriorityScore(approvalEvent)).toBe(10 + 200); // medium + approval boost
        });

        it("should combine boosts", () => {
            const safetyApprovalEvent: ServiceEvent = {
                id: "both", type: "safety/approval_required", timestamp: new Date(), data: {},
                metadata: { priority: PRIORITY_LEVELS.HIGH },
            };

            expect(calculatePriorityScore(safetyApprovalEvent)).toBe(100 + 500 + 200); // high + safety + approval
        });

        it("should default to medium priority when metadata missing", () => {
            const noMetadataEvent: ServiceEvent = {
                id: "no-meta", type: "test/event", timestamp: new Date(), data: {},
            };

            expect(calculatePriorityScore(noMetadataEvent)).toBe(10); // medium priority default
        });

        it("should handle missing priority in metadata", () => {
            const noPriorityEvent: ServiceEvent = {
                id: "no-priority", type: "test/event", timestamp: new Date(), data: {},
                metadata: { userId: generatePK().toString() }, // metadata exists but no priority
            };

            expect(calculatePriorityScore(noPriorityEvent)).toBe(10); // medium priority default
        });
    });

    describe("edge cases and error handling", () => {
        it("should handle malformed patterns in EventPatternMatcher", () => {
            // Should not throw errors for malformed patterns
            expect(() => new EventPatternMatcher("**/invalid")).not.toThrow();
            expect(() => new EventPatternMatcher("((invalid")).not.toThrow();
            expect(() => new EventPatternMatcher("[invalid")).not.toThrow();
        });

        it("should handle very long patterns", () => {
            const longPattern = "a/".repeat(1000) + "*";
            const matcher = new EventPatternMatcher(longPattern);
            
            expect(matcher.getPattern()).toBe(longPattern);
            expect(() => matcher.matches("test")).not.toThrow();
        });

        it("should handle unicode characters in patterns", () => {
            const unicodePattern = "财务/交易/*";
            const matcher = new EventPatternMatcher(unicodePattern);
            
            expect(matcher.matches("财务/交易/完成")).toBe(true);
            expect(matcher.matches("finance/transaction/completed")).toBe(false);
        });

        it("should handle extreme retry attempts", () => {
            const extremeStrategy: RetryStrategy = {
                maxAttempts: 0, // Zero attempts
                backoffMs: () => 100,
            };

            expect(async () => {
                await withRetry(() => Promise.resolve("success"), extremeStrategy);
            }).not.toThrow();
        });

        it("should handle negative delays", () => {
            expect(() => delay(-100)).not.toThrow();
        });
    });
});
