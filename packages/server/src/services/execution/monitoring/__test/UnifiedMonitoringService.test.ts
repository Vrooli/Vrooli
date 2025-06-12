/**
 * Tests for UnifiedMonitoringService to ensure basic functionality
 */

import { UnifiedMonitoringService } from "../UnifiedMonitoringService";
import { UnifiedMetric } from "../types";

describe("UnifiedMonitoringService", () => {
    let service: UnifiedMonitoringService;
    
    beforeEach(async () => {
        service = UnifiedMonitoringService.getInstance({
            maxOverheadMs: 5,
            eventBusEnabled: false, // Disable for testing
            mcpToolsEnabled: false,
        });
        await service.initialize();
    });
    
    afterEach(async () => {
        await service.shutdown();
    });
    
    describe("Basic Functionality", () => {
        it("should record a simple metric", async () => {
            const metric: Omit<UnifiedMetric, "id" | "timestamp"> = {
                tier: "cross-cutting",
                component: "test-component",
                type: "performance",
                name: "test.metric",
                value: 100,
                unit: "ms",
            };
            
            await service.record(metric);
            
            // Query the metric back
            const result = await service.query({
                name: "test.metric",
                limit: 1,
            });
            
            expect(result.metrics).toHaveLength(1);
            expect(result.metrics[0].name).toBe("test.metric");
            expect(result.metrics[0].value).toBe(100);
            expect(result.metrics[0].component).toBe("test-component");
        });
        
        it("should record multiple metrics in batch", async () => {
            const metrics: Array<Omit<UnifiedMetric, "id" | "timestamp">> = [
                {
                    tier: 1,
                    component: "swarm",
                    type: "performance",
                    name: "swarm.task.duration",
                    value: 200,
                    unit: "ms",
                },
                {
                    tier: 2,
                    component: "run",
                    type: "performance",
                    name: "run.step.duration",
                    value: 50,
                    unit: "ms",
                },
                {
                    tier: 3,
                    component: "executor",
                    type: "performance",
                    name: "execution.duration",
                    value: 25,
                    unit: "ms",
                },
            ];
            
            await service.recordBatch(metrics);
            
            // Query all metrics
            const result = await service.query({
                type: ["performance"],
                limit: 10,
            });
            
            expect(result.metrics.length).toBeGreaterThanOrEqual(3);
            
            const taskMetric = result.metrics.find(m => m.name === "swarm.task.duration");
            expect(taskMetric).toBeDefined();
            expect(taskMetric?.tier).toBe(1);
            expect(taskMetric?.value).toBe(200);
        });
        
        it("should query metrics by time range", async () => {
            const startTime = new Date();
            
            await service.record({
                tier: "cross-cutting",
                component: "test",
                type: "business",
                name: "test.event",
                value: 1,
            });
            
            await new Promise(resolve => setTimeout(resolve, 10)); // Small delay
            
            const endTime = new Date();
            
            const result = await service.query({
                startTime,
                endTime,
                name: "test.event",
            });
            
            expect(result.metrics).toHaveLength(1);
            expect(result.metrics[0].timestamp.getTime()).toBeGreaterThanOrEqual(startTime.getTime());
            expect(result.metrics[0].timestamp.getTime()).toBeLessThanOrEqual(endTime.getTime());
        });
        
        it("should filter metrics by tier", async () => {
            await service.recordBatch([
                {
                    tier: 1,
                    component: "tier1",
                    type: "performance",
                    name: "test.tier1",
                    value: 1,
                },
                {
                    tier: 2,
                    component: "tier2",
                    type: "performance",
                    name: "test.tier2",
                    value: 2,
                },
                {
                    tier: 3,
                    component: "tier3",
                    type: "performance",
                    name: "test.tier3",
                    value: 3,
                },
            ]);
            
            const tier1Result = await service.query({
                tier: [1],
                name: ["test.tier1", "test.tier2", "test.tier3"],
            });
            
            expect(tier1Result.metrics).toHaveLength(1);
            expect(tier1Result.metrics[0].tier).toBe(1);
            expect(tier1Result.metrics[0].value).toBe(1);
        });
        
        it("should generate metric summaries", async () => {
            // Record multiple values for the same metric
            const metricName = "test.summary.metric";
            const values = [10, 20, 30, 40, 50];
            
            for (const value of values) {
                await service.record({
                    tier: "cross-cutting",
                    component: "test",
                    type: "performance",
                    name: metricName,
                    value,
                });
            }
            
            const startTime = new Date(Date.now() - 60000); // 1 minute ago
            const endTime = new Date();
            
            const summaries = await service.getSummary(metricName, startTime, endTime);
            
            expect(summaries).toHaveLength(1);
            const summary = summaries[0];
            
            expect(summary.name).toBe(metricName);
            expect(summary.count).toBe(5);
            expect(summary.avg).toBe(30); // (10+20+30+40+50)/5
            expect(summary.min).toBe(10);
            expect(summary.max).toBe(50);
            expect(summary.p50).toBe(30);
        });
    });
    
    describe("Health Monitoring", () => {
        it("should report service health", async () => {
            const health = await service.getHealth();
            
            expect(health.status).toBe("healthy");
            expect(health.details.initialized).toBe(true);
            expect(health.details.collector).toBeDefined();
            expect(health.details.store).toBeDefined();
        });
    });
    
    describe("Performance", () => {
        it("should handle high-volume metrics", async () => {
            const startTime = process.hrtime.bigint();
            const batchSize = 1000;
            const metrics: Array<Omit<UnifiedMetric, "id" | "timestamp">> = [];
            
            for (let i = 0; i < batchSize; i++) {
                metrics.push({
                    tier: "cross-cutting",
                    component: "perf-test",
                    type: "performance",
                    name: "perf.test.metric",
                    value: i,
                });
            }
            
            await service.recordBatch(metrics);
            
            const endTime = process.hrtime.bigint();
            const durationMs = Number(endTime - startTime) / 1_000_000;
            
            // Should process 1000 metrics in reasonable time
            expect(durationMs).toBeLessThan(100); // Less than 100ms
            
            // Verify metrics were stored
            const result = await service.query({
                name: "perf.test.metric",
                limit: batchSize,
            });
            
            expect(result.metrics.length).toBeGreaterThan(0);
        });
    });
});