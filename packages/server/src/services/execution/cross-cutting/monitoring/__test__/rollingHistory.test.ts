/**
 * Tests for RollingHistoryManager
 */

import { describe, it, beforeEach, vi, expect } from "vitest";
import { RollingHistoryManager } from "../rollingHistory.js";
import type { HistoryEntry, HistoryQuery } from "@vrooli/shared";

describe("RollingHistoryManager", () => {
    let history: RollingHistoryManager;
    let clock: sinon.SinonFakeTimers;

    beforeEach(() => {
        clock = sinon.useFakeTimers();
        history = new RollingHistoryManager({
            maxEntries: 100,
            maxAgeMs: 3600000, // 1 hour
        });
    });

    afterEach(() => {
        clock.restore();
    });

    describe("Adding Entries", () => {
        it("should add entries to history", () => {
            history.add({
                timestamp: new Date(),
                type: "step_execution",
                tier: 3,
                component: "executor",
                operation: "execute",
                duration: 100,
                success: true,
                metadata: { stepId: "step-1" },
            });

            expect(history.getSize()).to.equal(1);
        });

        it("should add multiple entries in batch", () => {
            const entries: Array<Omit<HistoryEntry, "id">> = [
                {
                    timestamp: new Date(),
                    type: "step_execution",
                    tier: 3,
                    component: "executor",
                    operation: "execute",
                    success: true,
                    metadata: {},
                },
                {
                    timestamp: new Date(),
                    type: "validation",
                    tier: 3,
                    component: "validator",
                    operation: "validate",
                    success: false,
                    metadata: {},
                },
            ];

            history.addBatch(entries);
            expect(history.getSize()).to.equal(2);
        });

        it("should respect max entries limit", () => {
            history = new RollingHistoryManager({ maxEntries: 5 });

            for (let i = 0; i < 10; i++) {
                history.add({
                    timestamp: new Date(),
                    type: "test",
                    tier: 1,
                    component: "test",
                    operation: "test",
                    success: true,
                    metadata: { index: i },
                });
            }

            expect(history.getSize()).to.equal(5);
            
            // Verify oldest entries were overwritten
            const entries = history.query({ limit: 10 });
            const indices = entries.map(e => e.metadata.index);
            expect(indices).toEqual([9, 8, 7, 6, 5]);
        });
    });

    describe("Querying", () => {
        beforeEach(() => {
            // Add test data
            const baseTime = new Date("2024-01-01T00:00:00Z");
            
            history.add({
                timestamp: new Date(baseTime.getTime()),
                type: "step_execution",
                tier: 3,
                component: "executor",
                operation: "execute",
                duration: 100,
                success: true,
                metadata: { id: 1 },
            });
            
            history.add({
                timestamp: new Date(baseTime.getTime() + 1000),
                type: "validation",
                tier: 3,
                component: "validator",
                operation: "validate",
                duration: 50,
                success: false,
                metadata: { id: 2 },
            });
            
            history.add({
                timestamp: new Date(baseTime.getTime() + 2000),
                type: "step_execution",
                tier: 2,
                component: "orchestrator",
                operation: "orchestrate",
                duration: 200,
                success: true,
                metadata: { id: 3 },
            });
        });

        it("should query by type", () => {
            const results = history.query({
                types: ["step_execution"],
            });

            expect(results).toHaveLength(2);
            expect(results.every(r => r.type === "step_execution")).to.be.true;
        });

        it("should query by tier", () => {
            const results = history.query({
                tiers: [3],
            });

            expect(results).toHaveLength(2);
            expect(results.every(r => r.tier === 3)).to.be.true;
        });

        it("should query by component", () => {
            const results = history.query({
                components: ["executor"],
            });

            expect(results).toHaveLength(1);
            expect(results[0].component).toBe("executor");
        });

        it("should query by success status", () => {
            const results = history.query({
                success: false,
            });

            expect(results).toHaveLength(1);
            expect(results[0].success).toBe(false);
        });

        it("should query by time range", () => {
            const baseTime = new Date("2024-01-01T00:00:00Z");
            const results = history.query({
                timeRange: {
                    start: new Date(baseTime.getTime() + 500),
                    end: new Date(baseTime.getTime() + 1500),
                },
            });

            expect(results).toHaveLength(1);
            expect(results[0].metadata.id).toBe(2);
        });

        it("should apply ordering", () => {
            const results = history.query({
                orderBy: "duration",
                orderDirection: "desc",
            });

            expect(results.map(r => r.duration)).to.deep.equal([200, 100, 50]);
        });

        it("should apply limit", () => {
            const results = history.query({
                limit: 2,
            });

            expect(results).toHaveLength(2);
        });

        it("should combine multiple filters", () => {
            const results = history.query({
                types: ["step_execution"],
                tiers: [3],
                success: true,
            });

            expect(results).toHaveLength(1);
            expect(results[0].metadata.id).toBe(1);
        });
    });

    describe("Aggregation", () => {
        beforeEach(() => {
            // Add diverse test data
            for (let i = 0; i < 10; i++) {
                history.add({
                    timestamp: new Date(Date.now() + i * 1000),
                    type: i % 2 === 0 ? "step_execution" : "validation",
                    tier: (i % 3) + 1 as 1 | 2 | 3,
                    component: i % 2 === 0 ? "executor" : "validator",
                    operation: "test",
                    duration: 100 + i * 10,
                    success: i % 3 !== 0,
                    metadata: { index: i },
                });
            }
        });

        it("should calculate overall statistics", () => {
            const agg = history.aggregate({});

            expect(agg.totalEntries).toBe(10);
            expect(agg.successRate).to.be.closeTo(66.67, 0.01);
            expect(agg.avgDuration).toBe(145);
        });

        it("should aggregate by type", () => {
            const agg = history.aggregate({});

            expect(agg.byType["step_execution"]).toMatchObject({
                count: 5,
                successCount: 4,
                failureCount: 1,
                avgDuration: 140,
                minDuration: 100,
                maxDuration: 180,
            });
        });

        it("should aggregate by tier", () => {
            const agg = history.aggregate({});

            expect(agg.byTier["1"]).to.exist;
            expect(agg.byTier["2"]).to.exist;
            expect(agg.byTier["3"]).to.exist;
            
            const totalCount = Object.values(agg.byTier)
                .reduce((sum, stats) => sum + stats.count, 0);
            expect(totalCount).toBe(10);
        });

        it("should aggregate by component", () => {
            const agg = history.aggregate({});

            expect(agg.byComponent["executor"].count).toBe(5);
            expect(agg.byComponent["validator"].count).toBe(5);
        });

        it("should handle empty results", () => {
            history.clear();
            const agg = history.aggregate({});

            expect(agg.totalEntries).toBe(0);
            expect(agg.successRate).toBe(0);
            expect(agg.avgDuration).toBe(0);
            expect(agg.byType).toEqual({});
        });
    });

    describe("Recent Entries", () => {
        it("should get recent entries in reverse chronological order", () => {
            for (let i = 0; i < 5; i++) {
                history.add({
                    timestamp: new Date(),
                    type: "test",
                    tier: 1,
                    component: "test",
                    operation: "test",
                    success: true,
                    metadata: { index: i },
                });
            }

            const recent = history.getRecent(3);
            expect(recent).toHaveLength(3);
            expect(recent.map(e => e.metadata.index)).to.deep.equal([4, 3, 2]);
        });
    });

    describe("Time-based Pruning", () => {
        it("should prune old entries based on age", () => {
            history = new RollingHistoryManager({
                maxEntries: 100,
                maxAgeMs: 60000, // 1 minute
            });

            // Add old entry
            history.add({
                timestamp: new Date(Date.now() - 120000), // 2 minutes ago
                type: "old",
                tier: 1,
                component: "test",
                operation: "test",
                success: true,
                metadata: {},
            });

            // Add recent entry
            history.add({
                timestamp: new Date(),
                type: "recent",
                tier: 1,
                component: "test",
                operation: "test",
                success: true,
                metadata: {},
            });

            expect(history.getSize()).to.equal(2);

            // Advance time to trigger pruning
            clock.tick(61000);

            // Add another entry to trigger pruning
            history.add({
                timestamp: new Date(),
                type: "trigger",
                tier: 1,
                component: "test",
                operation: "test",
                success: true,
                metadata: {},
            });

            // Old entry should be pruned
            const entries = history.query({});
            expect(entries).toHaveLength(2);
            expect(entries.some(e => e.type === "old")).to.be.false;
        });
    });

    describe("Memory Management", () => {
        it("should provide memory usage estimates", () => {
            for (let i = 0; i < 10; i++) {
                history.add({
                    timestamp: new Date(),
                    type: "test",
                    tier: 1,
                    component: "test",
                    operation: "test",
                    success: true,
                    metadata: { data: "x".repeat(100) },
                });
            }

            const usage = history.getMemoryUsage();
            expect(usage.entries).toBeGreaterThan(0);
            expect(usage.indexes).toBeGreaterThan(0);
            expect(usage.total).toBe(usage.entries + usage.indexes);
        });
    });

    describe("Export/Import", () => {
        it("should export history data", () => {
            const entries = [
                {
                    timestamp: new Date(),
                    type: "test1",
                    tier: 1 as const,
                    component: "comp1",
                    operation: "op1",
                    success: true,
                    metadata: {},
                },
                {
                    timestamp: new Date(),
                    type: "test2",
                    tier: 2 as const,
                    component: "comp2",
                    operation: "op2",
                    success: false,
                    metadata: {},
                },
            ];

            history.addBatch(entries);

            const exported = history.export();
            expect(exported.entries).toHaveLength(2);
            expect(exported.metadata.size).toBe(2);
            expect(exported.metadata.timeRange).to.exist;
            expect(exported.metadata.config).to.exist;
        });

        it("should import history data", () => {
            const data = {
                entries: [
                    {
                        id: "test-1",
                        timestamp: new Date(),
                        type: "imported",
                        tier: 1 as const,
                        component: "test",
                        operation: "test",
                        success: true,
                        metadata: {},
                    },
                ],
                metadata: {
                    config: {
                        maxEntries: 50,
                    },
                },
            };

            history.import(data);

            expect(history.getSize()).to.equal(1);
            const entries = history.query({});
            expect(entries[0].type).toBe("imported");
        });
    });

    describe("Edge Cases", () => {
        it("should handle circular buffer wraparound", () => {
            history = new RollingHistoryManager({ maxEntries: 3 });

            for (let i = 0; i < 5; i++) {
                history.add({
                    timestamp: new Date(),
                    type: "test",
                    tier: 1,
                    component: "test",
                    operation: "test",
                    success: true,
                    metadata: { index: i },
                });
            }

            const entries = history.query({ orderBy: "timestamp", orderDirection: "asc" });
            expect(entries.map(e => e.metadata.index)).to.deep.equal([2, 3, 4]);
        });

        it("should handle queries on empty history", () => {
            const results = history.query({
                types: ["nonexistent"],
                limit: 10,
            });

            expect(results).toEqual([]);
        });

        it("should handle invalid time ranges gracefully", () => {
            history.add({
                timestamp: new Date(),
                type: "test",
                tier: 1,
                component: "test",
                operation: "test",
                success: true,
                metadata: {},
            });

            const results = history.query({
                timeRange: {
                    start: new Date("2025-01-01"),
                    end: new Date("2024-01-01"), // Invalid: end before start
                },
            });

            expect(results).toEqual([]);
        });
    });
});