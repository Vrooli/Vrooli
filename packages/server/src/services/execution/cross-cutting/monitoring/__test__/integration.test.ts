/**
 * Integration test for monitoring infrastructure
 */

import { describe, it, vi, expect } from "vitest";
import { TelemetryShim } from "../telemetryShim.js";
import { RollingHistoryManager } from "../rollingHistory.js";
import type { EventSource, MonitoringEvent } from "@vrooli/shared";

describe("Monitoring Integration", () => {
    it("should integrate telemetry and history tracking", async () => {
        // Mock event bus that captures events
        const capturedEvents: MonitoringEvent[] = [];
        const mockEventBus = {
            publish: async (event: MonitoringEvent) => {
                capturedEvents.push(event);
            },
            publishBatch: async (events: MonitoringEvent[]) => {
                capturedEvents.push(...events);
            },
            subscribe: async () => {},
            unsubscribe: async () => {},
            getEvents: async () => [],
            getEventStream: async function* () {},
            start: async () => {},
            stop: async () => {},
        };

        const source: EventSource = {
            tier: 3,
            component: "integration-test",
            instanceId: "test-instance",
        };

        // Create monitoring components
        const telemetry = new TelemetryShim(mockEventBus as any, source, {
            enabled: true,
            flushInterval: 100, // Flush quickly for testing
        });

        const history = new RollingHistoryManager({
            maxEntries: 1000,
            maxAgeMs: 3600000,
        });

        // Emit some events
        const startTime = new Date();
        await telemetry.emitExecutionTiming(
            "test-component",
            "test-operation",
            startTime,
            new Date(startTime.getTime() + 1000),
            true,
            { testData: "value" },
        );

        await telemetry.emitResourceUtilization(
            "test-component",
            { credits: 50, tokens: 1000, time: 30, memory: 256 },
            { credits: 100, tokens: 2000, time: 60, memory: 512 },
        );

        await telemetry.emitTaskCompletion(
            "task-123",
            "integration-test",
            "success",
            1000,
            50,
        );

        // Wait for flush
        await new Promise(resolve => setTimeout(resolve, 150));

        // Record history
        for (const event of capturedEvents) {
            if (event.type?.includes("perf") || event.type?.includes("biz")) {
                history.add({
                    timestamp: event.timestamp,
                    type: event.type,
                    tier: source.tier,
                    component: source.component,
                    operation: (event.payload as any).operation || "unknown",
                    duration: (event.payload as any).duration,
                    success: (event.payload as any).success ?? true,
                    metadata: event.payload as any,
                });
            }
        }

        // Verify telemetry emitted events
        expect(capturedEvents.length).to.be.greaterThanOrEqual(3);
        expect(capturedEvents.some(e => e.type === "perf.execution_timing")).to.be.true;
        expect(capturedEvents.some(e => e.type === "perf.resource_utilization")).to.be.true;
        expect(capturedEvents.some(e => e.type === "biz.task_completion")).to.be.true;

        // Verify history tracking
        const historyEntries = history.query({});
        expect(historyEntries.length).to.be.greaterThanOrEqual(3);

        // Query specific event types
        const perfEntries = history.query({
            types: ["perf.execution_timing", "perf.resource_utilization"],
        });
        expect(perfEntries.length).to.be.greaterThanOrEqual(2);

        // Get aggregated stats
        const stats = history.aggregate({});
        expect(stats.totalEntries).to.be.greaterThanOrEqual(3);
        expect(stats.successRate).toBe(100);

        // Test memory efficiency
        const memoryUsage = history.getMemoryUsage();
        expect(memoryUsage.total).to.be.lessThan(10000); // Less than 10KB for 3 entries

        // Cleanup
        await telemetry.stop();
    });

    it("should handle high-volume scenarios efficiently", async () => {
        const capturedEvents: MonitoringEvent[] = [];
        const mockEventBus = {
            publishBatch: async (events: MonitoringEvent[]) => {
                capturedEvents.push(...events);
            },
        } as any;

        const source: EventSource = {
            tier: 2,
            component: "high-volume-test",
            instanceId: "test-instance",
        };

        const telemetry = new TelemetryShim(mockEventBus, source, {
            enabled: true,
            batchSize: 100,
            flushInterval: 500,
            performanceGuards: {
                maxOverheadMs: 5,
                maxBufferSize: 1000,
                maxBatchSize: 100,
                dropOnOverload: true,
            },
        });

        const history = new RollingHistoryManager({
            maxEntries: 500,
            maxAgeMs: 60000, // 1 minute
        });

        // Emit many events rapidly
        const startTime = Date.now();
        for (let i = 0; i < 1000; i++) {
            await telemetry.emitThroughput(
                "high-volume",
                "requests",
                1000 + i,
                "req/s",
                60,
            );
        }

        // Wait for batch flush
        await new Promise(resolve => setTimeout(resolve, 600));

        const emissionTime = Date.now() - startTime;
        
        // Verify performance
        expect(emissionTime).to.be.lessThan(1000); // Should complete in under 1 second
        
        const stats = telemetry.getStats();
        expect(stats.avgOverheadMs).to.be.lessThan(5); // Average overhead under 5ms
        
        // Some events may be dropped due to performance guards
        expect(stats.emitted + stats.dropped).toBe(1000);
        
        // Verify batching worked
        expect(capturedEvents.length).to.be.lessThanOrEqual(stats.emitted);

        await telemetry.stop();
    });
});