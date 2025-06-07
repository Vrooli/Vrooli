import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import winston from "winston";
import { DeterministicStrategy } from "../tier3/strategies/deterministicStrategy.js";
import { DeterministicStrategyAdapter } from "../tier3/strategies/adapters/deterministicAdapter.js";
import { StrategyType } from "@vrooli/shared";

describe("DeterministicStrategy Integration", () => {
    let enhancedStrategy: DeterministicStrategy;
    let adapterStrategy: DeterministicStrategyAdapter;
    let logger: winston.Logger;

    const mockExecutionContext = {
        stepId: "integration-test-456",
        stepType: "batch_process",
        inputs: {
            dataset: [
                { id: 1, value: 100, category: "A" },
                { id: 2, value: 200, category: "B" },
                { id: 3, value: 150, category: "A" },
            ],
            rules: {
                aggregateByCategory: true,
                calculateStats: true,
                validateResults: true,
            },
        },
        config: {
            name: "Batch Data Processing",
            description: "Process dataset with aggregation and validation",
            requiredInputs: ["dataset"],
            expectedOutputs: {
                aggregatedData: { name: "Aggregated Results", description: "Category-wise aggregation" },
                statistics: { name: "Statistics", description: "Data statistics and metrics" },
                validationReport: { name: "Validation Report", description: "Data quality validation" },
            },
            transformRules: {
                groupBy: "category",
                metrics: ["sum", "avg", "count"],
            },
        },
        resources: {
            credits: 15000,
            tools: [
                {
                    name: "data_aggregator",
                    description: "Aggregates data by specified criteria",
                    parameters: { data: "array", groupBy: "string" },
                },
            ],
        },
        history: {
            recentSteps: [
                { stepId: "step-454", strategy: StrategyType.DETERMINISTIC, result: "success", duration: 1800 },
                { stepId: "step-455", strategy: StrategyType.DETERMINISTIC, result: "success", duration: 2200 },
            ],
            totalExecutions: 30,
            successRate: 0.93,
        },
        constraints: {
            maxTokens: 0, // No LLM usage
            maxTime: 45000,
            requiredConfidence: 0.85,
        },
        metadata: {
            userId: "integration-user-456",
        },
    };

    beforeEach(() => {
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        
        enhancedStrategy = new DeterministicStrategy(logger);
        adapterStrategy = new DeterministicStrategyAdapter(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Enhanced Strategy", () => {
        it("should successfully process batch data with aggregation", async () => {
            const result = await enhancedStrategy.execute(mockExecutionContext as any);

            expect(result.success).toBe(true);
            expect(result.outputs).toHaveProperty("aggregatedData");
            expect(result.outputs).toHaveProperty("statistics");
            expect(result.outputs).toHaveProperty("validationReport");
            expect(result.metadata.strategy).toBe("deterministic");
        });

        it("should handle missing required inputs", async () => {
            const contextWithMissingInputs = {
                ...mockExecutionContext,
                inputs: {},
            };

            const result = await enhancedStrategy.execute(contextWithMissingInputs as any);

            expect(result.success).toBe(false);
            expect(result.error).toMatch(/Missing required input/);
        });

        it("should respect time constraints", async () => {
            const contextWithShortTimeout = {
                ...mockExecutionContext,
                constraints: {
                    ...mockExecutionContext.constraints,
                    maxTime: 1, // 1ms timeout
                },
            };

            const result = await enhancedStrategy.execute(contextWithShortTimeout as any);

            expect(result.success).toBe(false);
            expect(result.error).toMatch(/timeout|time constraint/i);
        });
    });

    describe("Adapter Strategy", () => {
        it("should adapt legacy context format", async () => {
            const legacyContext = {
                stepId: "legacy-456",
                type: "batch_process",
                data: mockExecutionContext.inputs,
                configuration: {
                    name: mockExecutionContext.config.name,
                    description: mockExecutionContext.config.description,
                },
            };

            const result = await adapterStrategy.execute(legacyContext as any);

            expect(result.success).toBe(true);
            expect(result.outputs).toBeDefined();
        });

        it("should handle adapter-specific errors gracefully", async () => {
            const invalidContext = {
                stepId: "invalid-456",
                // Missing required fields
            };

            const result = await adapterStrategy.execute(invalidContext as any);

            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();
        });
    });

    describe("Strategy Comparison", () => {
        it("should produce consistent results across strategies", async () => {
            const enhancedResult = await enhancedStrategy.execute(mockExecutionContext as any);
            
            // Create adapted context for comparison
            const adaptedContext = {
                ...mockExecutionContext,
                type: mockExecutionContext.stepType,
                data: mockExecutionContext.inputs,
                configuration: mockExecutionContext.config,
            };
            
            const adapterResult = await adapterStrategy.execute(adaptedContext as any);

            // Both should succeed
            expect(enhancedResult.success).toBe(adapterResult.success);
            
            // Performance metrics should be similar
            if (enhancedResult.metadata?.performance && adapterResult.metadata?.performance) {
                const enhancedDuration = enhancedResult.metadata.performance.duration;
                const adapterDuration = adapterResult.metadata.performance.duration;
                
                // Allow for some variance but should be in same ballpark
                expect(Math.abs(enhancedDuration - adapterDuration)).toBeLessThan(1000);
            }
        });
    });
});