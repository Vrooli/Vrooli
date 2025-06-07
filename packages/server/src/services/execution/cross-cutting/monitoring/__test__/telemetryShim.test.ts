/**
 * Tests for TelemetryShim
 */

import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import { TelemetryShim } from "../telemetryShim.js";
import type { EventBus, MonitoringEventPrefix, EventSource } from "@vrooli/shared";

describe("TelemetryShim", () => {
    let eventBus: EventBus;
    let telemetry: TelemetryShim;
    let clock: any;
    const source: EventSource = {
        tier: 3,
        component: "test-component",
        instanceId: "test-instance",
    };

    beforeEach(() => {
        clock = vi.useFakeTimers();
        eventBus = {
            publish: vi.fn().mockResolvedValue(undefined),
            publishBatch: vi.fn().mockResolvedValue(undefined),
            subscribe: vi.fn().mockResolvedValue(undefined),
            unsubscribe: vi.fn().mockResolvedValue(undefined),
            getEvents: vi.fn().mockResolvedValue([]),
            getEventStream: vi.fn(),
            start: vi.fn().mockResolvedValue(undefined),
            stop: vi.fn().mockResolvedValue(undefined),
        };
        telemetry = new TelemetryShim(eventBus as any, source);
    });

    afterEach(async () => {
        await telemetry.stop();
        clock.restore();
        vi.restoreAllMocks();
    });

    describe("Performance Events", () => {
        it("should emit execution timing event", async () => {
            const startTime = new Date();
            const endTime = new Date(startTime.getTime() + 1000);

            await telemetry.emitExecutionTiming(
                "test-component",
                "test-operation",
                startTime,
                endTime,
                true,
                { foo: "bar" },
            );

            // Advance clock to trigger flush
            clock.tick(1100);

            expect(eventBus.publishBatch.calledOnce).toBe(true);
            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events).toHaveLength(1);
            expect(events[0].type).toBe("perf.execution_timing");
            expect(events[0].payload).toMatchObject({
                type: "execution_timing",
                component: "test-component",
                operation: "test-operation",
                duration: 1000,
                success: true,
            });
        });

        it("should calculate resource utilization percentage", async () => {
            await telemetry.emitResourceUtilization(
                "test-component",
                { credits: 50, tokens: 1000, time: 30, memory: 256 },
                { credits: 100, tokens: 2000, time: 60, memory: 512 },
            );

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].payload.utilizationPercent).toBe(50);
        });

        it("should emit latency metrics", async () => {
            await telemetry.emitLatency(
                "test-component",
                "test-operation",
                50,
                { p50: 45, p90: 80, p95: 95, p99: 150, p999: 200 },
            );

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].type).toBe("perf.latency");
            expect(events[0].payload.percentiles).toEqual({
                p50: 45,
                p90: 80,
                p95: 95,
                p99: 150,
                p999: 200,
            });
        });
    });

    describe("Health Events", () => {
        it("should emit component health event with high priority", async () => {
            await telemetry.emitComponentHealth(
                "test-component",
                "degraded",
                [
                    { name: "database", status: "pass" },
                    { name: "cache", status: "warn", message: "High latency" },
                ],
            );

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].type).toBe("health.component_health");
            expect(events[0].sampling.priority).toBe("high");
        });
    });

    describe("Business Events", () => {
        it("should emit task completion event", async () => {
            await telemetry.emitTaskCompletion(
                "task-123",
                "data-processing",
                "success",
                5000,
                100,
            );

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].type).toBe("biz.task_completion");
            expect(events[0].payload).toMatchObject({
                taskId: "task-123",
                taskType: "data-processing",
                result: "success",
                duration: 5000,
                resourceCost: 100,
            });
        });

        it("should emit strategy effectiveness metrics", async () => {
            await telemetry.emitStrategyEffectiveness(
                "conversational",
                "question-answering",
                {
                    successRate: 0.95,
                    avgDuration: 2500,
                    avgCost: 50,
                    sampleSize: 100,
                },
            );

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].type).toBe("biz.strategy_effectiveness");
        });
    });

    describe("Safety Events", () => {
        it("should emit validation errors with high priority", async () => {
            await telemetry.emitValidationError(
                "input-validator",
                [
                    { field: "email", rule: "format", message: "Invalid email", severity: "error" },
                    { rule: "required", message: "Name is required", severity: "error" },
                ],
                { userId: "user-123" },
            );

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].type).toBe("safety.validation_error");
            expect(events[0].sampling.priority).toBe("high");
        });

        it("should emit security incident with always priority", async () => {
            await telemetry.emitSecurityIncident(
                "unauthorized-access",
                "high",
                { ip: "192.168.1.1", path: "/admin" },
            );

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].sampling.priority).toBe("always");
        });

        it("should extract error data correctly", async () => {
            const error = new Error("Test error");
            error.stack = "Error: Test error\n    at test.js:10\n    at runner.js:20";

            await telemetry.emitError(error, "test-component", "medium");

            clock.tick(1100);

            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events[0].payload).toMatchObject({
                errorType: "Error",
                message: "Test error",
                component: "test-component",
                severity: "medium",
            });
            expect(events[0].payload.stack).toHaveLength(3);
        });
    });

    describe("Sampling", () => {
        it("should respect sampling rates", async () => {
            // Create telemetry with 10% sampling for performance events
            telemetry = new TelemetryShim(eventBus as any, source, {
                enabled: true,
                samplingRates: {
                    [MonitoringEventPrefix.PERFORMANCE]: 0.1,
                },
            });

            // Emit 100 events
            for (let i = 0; i < 100; i++) {
                await telemetry.emitThroughput(
                    "test-component",
                    "requests",
                    100,
                    "req/s",
                    60,
                );
            }

            // Should have approximately 10 events (±5 for randomness)
            const stats = telemetry.getStats();
            expect(stats.emitted).toBeGreaterThanOrEqual(5); expect(stats.emitted).toBeLessThanOrEqual(15);
        });

        it("should always emit events with 'always' priority", async () => {
            telemetry = new TelemetryShim(eventBus as any, source, {
                enabled: true,
                samplingRates: {
                    [MonitoringEventPrefix.SAFETY]: 0.0, // 0% sampling
                },
            });

            // This should still be emitted due to 'always' priority
            await telemetry.emitSecurityIncident(
                "critical-breach",
                "critical",
                { details: "test" },
            );

            clock.tick(1100);

            expect(eventBus.publishBatch.called).toBe(true);
        });
    });

    describe("Performance Guards", () => {
        it("should drop events when buffer is full", async () => {
            telemetry = new TelemetryShim(eventBus as any, source, {
                enabled: true,
                performanceGuards: {
                    maxOverheadMs: 5,
                    maxBufferSize: 10,
                    maxBatchSize: 5,
                    dropOnOverload: true,
                },
                flushInterval: 10000, // Don't auto-flush
            });

            // Fill buffer beyond capacity
            for (let i = 0; i < 20; i++) {
                await telemetry.emitThroughput(
                    "test-component",
                    "metric",
                    i,
                    "ops",
                    60,
                );
            }

            const stats = telemetry.getStats();
            expect(stats.dropped).toBeGreaterThan(0);
            expect(stats.bufferSize).toBe(10);
        });

        it("should batch events efficiently", async () => {
            await telemetry.emitBatch([
                {
                    category: MonitoringEventPrefix.PERFORMANCE,
                    payload: {
                        type: "throughput",
                        component: "test",
                        metric: "requests",
                        value: 100,
                        unit: "req/s",
                        window: 60,
                    },
                },
                {
                    category: MonitoringEventPrefix.BUSINESS,
                    payload: {
                        type: "task_completion",
                        taskId: "task-1",
                        taskType: "processing",
                        result: "success",
                        duration: 1000,
                    },
                },
            ]);

            clock.tick(1100);

            expect(eventBus.publishBatch.calledOnce).toBe(true);
            const events = eventBus.publishBatch.firstCall.args[0];
            expect(events).toHaveLength(2);
        });
    });

    describe("Lifecycle", () => {
        it("should flush events on stop", async () => {
            await telemetry.emitThroughput("test", "metric", 100, "ops", 60);
            
            // Don't advance clock, stop immediately
            await telemetry.stop();

            expect(eventBus.publishBatch.called).toBe(true);
        });

        it("should log stats on stop", async () => {
            const consoleStub = sinon.stub(console, "info");

            await telemetry.emitThroughput("test", "metric", 100, "ops", 60);
            await telemetry.stop();

            expect(consoleStub.calledWith(
                expect.objectContaining("[TelemetryShim] Stats:"),
                sinon.match.object,
            )).to.be.true;
        });

        it("should handle disabled telemetry", async () => {
            telemetry = new TelemetryShim(eventBus as any, source, {
                enabled: false,
            });

            await telemetry.emitThroughput("test", "metric", 100, "ops", 60);
            clock.tick(2000);

            expect(eventBus.publishBatch.called).toBe(false);
        });
    });
});