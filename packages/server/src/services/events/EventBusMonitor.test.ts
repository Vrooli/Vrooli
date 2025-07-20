/**
 * EventBusMonitor tests
 * 
 * Tests for performance monitoring and optimization of the event bus system.
 * Critical for maintaining system performance and detecting issues.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { EventBusMonitor } from "./EventBusMonitor.js";
import { EventMode } from "./types.js";
import { MONITOR_PERFORMANCE_LIMITS, MONITOR_SCORING, MONITOR_THRESHOLDS } from "./constants.js";
import { MINUTES_1_MS, HOURS_1_S, DAYS_1_MS } from "@vrooli/shared";

// Mock logger to suppress output during tests
vi.mock("../../events/logger.js", () => ({
    logger: {
        debug: vi.fn(),
        error: vi.fn(),
        info: vi.fn(),
        warning: vi.fn(),
        warn: vi.fn(),
    },
}));

describe("EventBusMonitor", () => {
    let monitor: EventBusMonitor;

    beforeEach(() => {
        monitor = new EventBusMonitor();
        vi.clearAllTimers();
        vi.useFakeTimers();
    });

    afterEach(() => {
        monitor.stop();
        vi.useRealTimers();
        vi.clearAllMocks();
    });

    describe("initialization and lifecycle", () => {
        it("should start monitoring with interval", () => {
            const setIntervalSpy = vi.spyOn(global, "setInterval");
            
            monitor.start();
            
            expect(setIntervalSpy).toHaveBeenCalledWith(
                expect.any(Function),
                MINUTES_1_MS,
            );
        });

        it("should not start multiple intervals", () => {
            const setIntervalSpy = vi.spyOn(global, "setInterval");
            
            monitor.start();
            monitor.start(); // Second call should be ignored
            
            expect(setIntervalSpy).toHaveBeenCalledTimes(1);
        });

        it("should stop monitoring", () => {
            const clearIntervalSpy = vi.spyOn(global, "clearInterval");
            
            monitor.start();
            monitor.stop();
            
            expect(clearIntervalSpy).toHaveBeenCalled();
        });

        it("should handle stop when not started", () => {
            const clearIntervalSpy = vi.spyOn(global, "clearInterval");
            
            monitor.stop(); // Should not throw
            
            expect(clearIntervalSpy).not.toHaveBeenCalled();
        });
    });

    describe("event recording", () => {
        it("should record event with basic metrics", () => {
            monitor.recordEvent("finance/transaction", EventMode.PASSIVE, 100, true);
            
            const report = monitor.getPerformanceReport();
            
            expect(report.summary.totalEvents).toBe(1);
            expect(report.summary.uniqueEventTypes).toBe(1);
            expect(report.topEventTypes).toHaveLength(1);
            expect(report.topEventTypes[0]).toEqual({
                type: "finance/transaction",
                count: 1,
                avgTime: 100,
            });
        });

        it("should accumulate metrics for same event type", () => {
            monitor.recordEvent("finance/transaction", EventMode.PASSIVE, 100, true);
            monitor.recordEvent("finance/transaction", EventMode.PASSIVE, 200, true);
            monitor.recordEvent("finance/transaction", EventMode.PASSIVE, 150, true);
            
            const report = monitor.getPerformanceReport();
            
            expect(report.summary.totalEvents).toBe(3);
            expect(report.summary.uniqueEventTypes).toBe(1);
            expect(report.topEventTypes[0]).toEqual({
                type: "finance/transaction",
                count: 3,
                avgTime: 150, // (100 + 200 + 150) / 3
            });
        });

        it("should track min and max times", () => {
            monitor.recordEvent("test/event", EventMode.PASSIVE, 100, true);
            monitor.recordEvent("test/event", EventMode.PASSIVE, 300, true);
            monitor.recordEvent("test/event", EventMode.PASSIVE, 50, true);
            
            // Access private metrics for detailed testing
            const metrics = (monitor as any).eventMetrics.get("test/event");
            
            expect(metrics.minTime).toBe(50);
            expect(metrics.maxTime).toBe(300);
            expect(metrics.avgTime).toBe(150); // (100 + 300 + 50) / 3
        });

        it("should track error count", () => {
            monitor.recordEvent("test/event", EventMode.PASSIVE, 100, true);
            monitor.recordEvent("test/event", EventMode.PASSIVE, 200, false); // Error
            monitor.recordEvent("test/event", EventMode.PASSIVE, 150, false); // Error
            
            const errorRate = monitor.getPerformanceReport().summary.errorRate;
            expect(errorRate).toBe(2 / 3); // 2 errors out of 3 events
        });

        it("should track mode distribution", () => {
            monitor.recordEvent("event1", EventMode.PASSIVE, 100, true);
            monitor.recordEvent("event2", EventMode.INTERCEPTABLE, 200, true);
            monitor.recordEvent("event3", EventMode.INTERCEPTABLE, 300, true);
            
            const report = monitor.getPerformanceReport();
            
            expect(report.modeDistribution[EventMode.PASSIVE]).toEqual({
                percentage: 100 / 3, // 1 out of 3
                avgTime: 100,
            });
            expect(report.modeDistribution[EventMode.INTERCEPTABLE]).toEqual({
                percentage: 200 / 3, // 2 out of 3
                avgTime: 250, // (200 + 300) / 2
            });
        });

        it("should update last seen timestamp", () => {
            const startTime = Date.now();
            vi.setSystemTime(startTime);
            
            monitor.recordEvent("test/event", EventMode.PASSIVE, 100, true);
            
            const metrics = (monitor as any).eventMetrics.get("test/event");
            expect(metrics.lastSeen).toBe(startTime);
            
            // Advance time and record another event
            vi.setSystemTime(startTime + 10000);
            monitor.recordEvent("test/event", EventMode.PASSIVE, 100, true);
            
            expect(metrics.lastSeen).toBe(startTime + 10000);
        });
    });

    describe("pattern recording", () => {
        it("should record pattern metrics", () => {
            monitor.recordPatternMatch("finance/*", 50, 3, true);
            
            const patternMetrics = (monitor as any).patternMetrics.get("finance/*");
            
            expect(patternMetrics).toEqual({
                pattern: "finance/*",
                matchCount: 1,
                avgMatchTime: 50,
                botCount: 3,
                cacheHitRate: 1, // 100% hit rate
            });
        });

        it("should calculate rolling averages for pattern metrics", () => {
            monitor.recordPatternMatch("finance/*", 40, 3, true);  // cache hit
            monitor.recordPatternMatch("finance/*", 60, 3, false); // cache miss
            monitor.recordPatternMatch("finance/*", 50, 3, true);  // cache hit
            
            const patternMetrics = (monitor as any).patternMetrics.get("finance/*");
            
            expect(patternMetrics.avgMatchTime).toBe(50); // (40 + 60 + 50) / 3
            expect(patternMetrics.cacheHitRate).toBe(2 / 3); // 2 hits out of 3
        });

        it("should update bot count", () => {
            monitor.recordPatternMatch("test/*", 50, 2, true);
            monitor.recordPatternMatch("test/*", 60, 5, true); // Different bot count
            
            const patternMetrics = (monitor as any).patternMetrics.get("test/*");
            
            expect(patternMetrics.botCount).toBe(5); // Should use latest bot count
        });
    });

    describe("system health assessment", () => {
        it("should report healthy status with no issues", () => {
            monitor.recordEvent("test/event", EventMode.PASSIVE, 50, true); // Low latency
            
            const health = monitor.getSystemHealth();
            
            expect(health.status).toBe("healthy");
            expect(health.issues).toHaveLength(0);
            expect(health.score).toBe(100);
        });

        it("should detect high latency issues", () => {
            // Record events with high latency
            monitor.recordEvent("slow/event", EventMode.PASSIVE, MONITOR_PERFORMANCE_LIMITS.MAX_LATENCY_MS + 100, true);
            
            const health = monitor.getSystemHealth();
            
            expect(health.status).toBe("degraded");
            expect(health.issues).toContain(expect.stringContaining("High average latency"));
            expect(health.recommendations).toContain(expect.stringContaining("scaling horizontally"));
            expect(health.score).toBe(100 - MONITOR_SCORING.DEDUCTION_HIGH_LATENCY);
        });

        it("should detect high error rate", () => {
            // Record events with high error rate
            monitor.recordEvent("error/event", EventMode.PASSIVE, 100, false); // Error
            monitor.recordEvent("error/event", EventMode.PASSIVE, 100, false); // Error
            monitor.recordEvent("error/event", EventMode.PASSIVE, 100, true);  // Success
            // Error rate: 2/3 = 0.67, which is > 0.05 threshold
            
            const health = monitor.getSystemHealth();
            
            expect(health.status).toBe("critical");
            expect(health.issues).toContain(expect.stringContaining("High error rate"));
            expect(health.recommendations).toContain(expect.stringContaining("Review error logs"));
        });

        it("should detect slow patterns", () => {
            // Record slow pattern
            monitor.recordPatternMatch("slow/pattern/*", MONITOR_THRESHOLDS.SLOW_PATTERN_MS + 5, 3, true);
            
            const health = monitor.getSystemHealth();
            
            expect(health.issues).toContain("1 slow patterns detected");
            expect(health.recommendations).toContain(expect.stringContaining("Optimize pattern matching"));
        });

        it("should combine multiple issues correctly", () => {
            // High latency
            monitor.recordEvent("slow/event", EventMode.PASSIVE, MONITOR_PERFORMANCE_LIMITS.MAX_LATENCY_MS + 100, true);
            // High error rate
            monitor.recordEvent("error/event", EventMode.PASSIVE, 100, false);
            monitor.recordEvent("error/event", EventMode.PASSIVE, 100, false);
            monitor.recordEvent("error/event", EventMode.PASSIVE, 100, true);
            
            const health = monitor.getSystemHealth();
            
            expect(health.status).toBe("critical");
            expect(health.issues.length).toBeGreaterThan(1);
            expect(health.score).toBeLessThan(100 - MONITOR_SCORING.DEDUCTION_HIGH_LATENCY);
        });

        it("should ensure score never goes below 0", () => {
            // Create extremely bad conditions
            for (let i = 0; i < 10; i++) {
                monitor.recordEvent("bad/event", EventMode.PASSIVE, MONITOR_PERFORMANCE_LIMITS.MAX_LATENCY_MS * 2, false);
            }
            
            const health = monitor.getSystemHealth();
            
            expect(health.score).toBeGreaterThanOrEqual(0);
        });
    });

    describe("performance reporting", () => {
        it("should generate comprehensive performance report", () => {
            // Record various events
            monitor.recordEvent("finance/transaction", EventMode.PASSIVE, 100, true);
            monitor.recordEvent("chat/message", EventMode.INTERCEPTABLE, 200, true);
            monitor.recordEvent("finance/transaction", EventMode.PASSIVE, 150, true);
            
            const report = monitor.getPerformanceReport();
            
            expect(report.summary).toEqual({
                totalEvents: 3,
                uniqueEventTypes: 2,
                avgLatency: 150, // (100 + 200 + 150) / 3
                errorRate: 0,
                throughput: 0, // No historical data yet
            });
            
            expect(report.topEventTypes).toHaveLength(2);
            expect(report.topEventTypes[0].type).toBe("finance/transaction"); // Higher count
            expect(report.topEventTypes[0].count).toBe(2);
        });

        it("should limit top event types to configured count", () => {
            // Create more event types than the limit
            for (let i = 0; i < MONITOR_THRESHOLDS.TOP_EVENT_TYPES_COUNT + 5; i++) {
                monitor.recordEvent(`event/type/${i}`, EventMode.PASSIVE, 100, true);
            }
            
            const report = monitor.getPerformanceReport();
            
            expect(report.topEventTypes).toHaveLength(MONITOR_THRESHOLDS.TOP_EVENT_TYPES_COUNT);
        });

        it("should include pattern performance in report", () => {
            monitor.recordPatternMatch("finance/*", 20, 3, true);
            monitor.recordPatternMatch("chat/*", 50, 2, false);
            
            const report = monitor.getPerformanceReport();
            
            expect(report.slowestPatterns).toHaveLength(2);
            expect(report.slowestPatterns[0].pattern).toBe("chat/*"); // Slowest first
            expect(report.slowestPatterns[0].avgTime).toBe(50);
            expect(report.slowestPatterns[1].cacheHitRate).toBe(1);
        });

        it("should handle empty metrics gracefully", () => {
            const report = monitor.getPerformanceReport();
            
            expect(report.summary.totalEvents).toBe(0);
            expect(report.summary.avgLatency).toBe(0);
            expect(report.summary.errorRate).toBe(0);
            expect(report.topEventTypes).toHaveLength(0);
            expect(report.slowestPatterns).toHaveLength(0);
        });
    });

    describe("optimization suggestions", () => {
        it("should suggest caching for high-frequency patterns", () => {
            // Create high-frequency pattern
            for (let i = 0; i < MONITOR_THRESHOLDS.HIGH_FREQUENCY + 10; i++) {
                monitor.recordPatternMatch("high/frequency/*", MONITOR_THRESHOLDS.LOW_PROCESSING_TIME - 1, 3, true);
            }
            
            const suggestions = monitor.getOptimizationSuggestions();
            
            const cacheSuggestion = suggestions.find(s => s.type === "cache");
            expect(cacheSuggestion).toBeDefined();
            expect(cacheSuggestion?.target).toBe("high/frequency/*");
        });

        it("should suggest indexing for low cache hit rates", () => {
            // Create pattern with low cache hit rate
            for (let i = 0; i < 10; i++) {
                monitor.recordPatternMatch("low/cache/*", 50, 3, false); // Always cache miss
            }
            
            const suggestions = monitor.getOptimizationSuggestions();
            
            const indexSuggestion = suggestions.find(s => s.type === "index");
            expect(indexSuggestion).toBeDefined();
            expect(indexSuggestion?.target).toBe("low/cache/*");
        });

        it("should suggest batching for burst patterns", () => {
            // Create burst pattern (high event count)
            for (let i = 0; i < MONITOR_THRESHOLDS.HIGH_EVENT_COUNT + 10; i++) {
                monitor.recordEvent("burst/event", EventMode.PASSIVE, 100, true);
            }
            
            const suggestions = monitor.getOptimizationSuggestions();
            
            const batchSuggestion = suggestions.find(s => s.type === "batch");
            expect(batchSuggestion).toBeDefined();
            expect(batchSuggestion?.target).toBe("burst/event");
        });

        it("should suggest pattern optimization for complex wildcards", () => {
            monitor.recordPatternMatch("complex/**/pattern/*", 50, 3, true);
            
            const suggestions = monitor.getOptimizationSuggestions();
            
            const patternSuggestion = suggestions.find(s => s.type === "pattern");
            expect(patternSuggestion).toBeDefined();
            expect(patternSuggestion?.target).toBe("complex/**/pattern/*");
        });

        it("should suggest pattern optimization for too many wildcards", () => {
            monitor.recordPatternMatch("a/*/b/*/c/*/d", 50, 3, true); // 3 wildcards
            
            const suggestions = monitor.getOptimizationSuggestions();
            
            const patternSuggestion = suggestions.find(s => s.type === "pattern");
            expect(patternSuggestion).toBeDefined();
        });

        it("should return empty suggestions when performance is good", () => {
            // Record only good performance metrics
            monitor.recordEvent("good/event", EventMode.PASSIVE, 50, true);
            monitor.recordPatternMatch("good/pattern", 5, 3, true);
            
            const suggestions = monitor.getOptimizationSuggestions();
            
            expect(suggestions).toHaveLength(0);
        });
    });

    describe("data management", () => {
        it("should collect time series metrics", () => {
            monitor.start();
            
            // Record some events
            monitor.recordEvent("test/event", EventMode.PASSIVE, 100, true);
            
            // Trigger metrics collection
            vi.advanceTimersByTime(MINUTES_1_MS);
            
            const throughputHistory = (monitor as any).throughputHistory;
            const latencyHistory = (monitor as any).latencyHistory;
            const errorHistory = (monitor as any).errorHistory;
            
            expect(throughputHistory.length).toBeGreaterThan(0);
            expect(latencyHistory.length).toBeGreaterThan(0);
            expect(errorHistory.length).toBeGreaterThan(0);
        });

        it("should limit time series data to one hour", () => {
            monitor.start();
            
            // Simulate data collection over time
            for (let i = 0; i < 70; i++) { // More than 60 minutes
                vi.advanceTimersByTime(MINUTES_1_MS);
            }
            
            const throughputHistory = (monitor as any).throughputHistory;
            
            // Should only keep last hour (approximately)
            expect(throughputHistory.length).toBeLessThanOrEqual(61); // 60 minutes + buffer
        });

        it("should clean up old event metrics", () => {
            const startTime = Date.now();
            vi.setSystemTime(startTime);
            
            // Record event
            monitor.recordEvent("old/event", EventMode.PASSIVE, 100, true);
            
            // Advance time by more than a day
            vi.setSystemTime(startTime + DAYS_1_MS + 1000);
            
            // Start monitoring to trigger cleanup
            monitor.start();
            vi.advanceTimersByTime(MINUTES_1_MS);
            
            const eventMetrics = (monitor as any).eventMetrics;
            expect(eventMetrics.has("old/event")).toBe(false);
        });

        it("should not clean up recent event metrics", () => {
            const startTime = Date.now();
            vi.setSystemTime(startTime);
            
            // Record event
            monitor.recordEvent("recent/event", EventMode.PASSIVE, 100, true);
            
            // Advance time by less than a day
            vi.setSystemTime(startTime + DAYS_1_MS - 1000);
            
            // Start monitoring to trigger cleanup
            monitor.start();
            vi.advanceTimersByTime(MINUTES_1_MS);
            
            const eventMetrics = (monitor as any).eventMetrics;
            expect(eventMetrics.has("recent/event")).toBe(true);
        });
    });

    describe("edge cases and error handling", () => {
        it("should handle division by zero in metrics calculation", () => {
            // Don't record any events
            const report = monitor.getPerformanceReport();
            
            expect(report.summary.avgLatency).toBe(0);
            expect(report.summary.errorRate).toBe(0);
            expect(report.summary.throughput).toBe(0);
        });

        it("should handle negative processing times", () => {
            monitor.recordEvent("test/event", EventMode.PASSIVE, -100, true);
            
            const metrics = (monitor as any).eventMetrics.get("test/event");
            expect(metrics.minTime).toBe(-100);
            expect(metrics.avgTime).toBe(-100);
        });

        it("should handle extremely large processing times", () => {
            const largeTime = Number.MAX_SAFE_INTEGER;
            monitor.recordEvent("test/event", EventMode.PASSIVE, largeTime, true);
            
            const metrics = (monitor as any).eventMetrics.get("test/event");
            expect(metrics.maxTime).toBe(largeTime);
            expect(metrics.avgTime).toBe(largeTime);
        });

        it("should handle pattern matching with empty strings", () => {
            monitor.recordPatternMatch("", 50, 0, true);
            
            const patternMetrics = (monitor as any).patternMetrics.get("");
            expect(patternMetrics).toBeDefined();
            expect(patternMetrics.botCount).toBe(0);
        });

        it("should handle zero bot count in patterns", () => {
            monitor.recordPatternMatch("test/pattern", 50, 0, true);
            
            const patternMetrics = (monitor as any).patternMetrics.get("test/pattern");
            expect(patternMetrics.botCount).toBe(0);
        });
    });
});
