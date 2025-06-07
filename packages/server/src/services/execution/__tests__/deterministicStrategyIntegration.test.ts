import { describe, it, expect, beforeEach, afterEach } from "@jest/globals";
import winston from "winston";
import { DeterministicStrategy } from "../tier3/strategies/deterministicStrategy.js";
import { DeterministicStrategyAdapter } from "../tier3/strategies/adapters/deterministicAdapter.js";
import { StrategyType } from "@vrooli/shared";

describe("DeterministicStrategy Integration", () => {
    let enhancedStrategy: DeterministicStrategy;
    let adapterStrategy: DeterministicStrategyAdapter;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility

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

    describe("Enhanced Strategy vs Adapter Comparison", () => {
        it("should have consistent interfaces", () => {
            // Both should implement the same interface
            expect(enhancedStrategy.type).toBe(StrategyType.DETERMINISTIC);
            expect(adapterStrategy.type).toBe(StrategyType.DETERMINISTIC);
            
            expect(enhancedStrategy.name).toBe("DeterministicStrategy");
            expect(adapterStrategy.name).toBe("DeterministicStrategy");
            
            // Enhanced should have newer version
            expect(enhancedStrategy.version).toBe("2.0.0-enhanced");
            expect(adapterStrategy.version).toBe("1.0.0-adapter");
        });

        it("should handle the same step types", () => {
            const testCases = [
                "process_data",
                "transform_records", 
                "batch_processing",
                "api_integration",
                "code_execution",
                "validate_data",
            ];

            testCases.forEach(stepType => {
                expect(enhancedStrategy.canHandle(stepType)).toBe(adapterStrategy.canHandle(stepType));
            });
        });

        it("should provide resource estimates in same ballpark", () => {
            const enhancedEstimate = enhancedStrategy.estimateResources(mockExecutionContext);
            const adapterEstimate = adapterStrategy.estimateResources(mockExecutionContext);

            // Both should provide similar estimates (within 50% of each other)
            expect(enhancedEstimate.tokens).toBe(0);
            expect(adapterEstimate.tokens).toBe(0);
            
            expect(enhancedEstimate.apiCalls).toBeGreaterThanOrEqual(0);
            expect(adapterEstimate.apiCalls).toBeGreaterThanOrEqual(0);
            
            expect(enhancedEstimate.cost).toBeGreaterThan(0);
            expect(adapterEstimate.cost).toBeGreaterThan(0);
            
            // Costs should be in similar range
            const costRatio = enhancedEstimate.cost / adapterEstimate.cost;
            expect(costRatio).toBeGreaterThan(0.5);
            expect(costRatio).toBeLessThan(2.0);
        });
    });

    describe("Enhanced Strategy Features", () => {
        it("should provide more sophisticated execution patterns", async () => {
            const result = await enhancedStrategy.execute(mockExecutionContext);

            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();
            expect(result.metadata.confidence).toBe(1.0);
            
            // Enhanced strategy should provide rich metadata
            expect(result.metadata.resourceUsage).toBeDefined();
            expect(result.feedback.performanceScore).toBeGreaterThan(0);
            expect(result.feedback.improvements).toBeDefined();
        });

        it("should demonstrate cache integration", async () => {
            // First execution
            const result1 = await enhancedStrategy.execute(mockExecutionContext);
            expect(result1.success).toBe(true);

            // Second execution with same context
            const result2 = await enhancedStrategy.execute(mockExecutionContext);
            expect(result2.success).toBe(true);
            
            // Should maintain consistency
            expect(result2.metadata.strategyType).toBe(result1.metadata.strategyType);
        });

        it("should track performance over multiple executions", async () => {
            // Execute multiple times
            const contexts = [
                mockExecutionContext,
                { ...mockExecutionContext, stepId: "test-2" },
                { ...mockExecutionContext, stepId: "test-3" },
            ];

            for (const context of contexts) {
                const result = await enhancedStrategy.execute(context);
                expect(result.success).toBe(true);
            }

            const metrics = enhancedStrategy.getPerformanceMetrics();
            expect(metrics.totalExecutions).toBeGreaterThan(0);
            expect(metrics.averageConfidence).toBeGreaterThanOrEqual(0.9);
        });
    });

    describe("Legacy Pattern Preservation", () => {
        it("should preserve 3-phase execution pattern in both implementations", async () => {
            const enhancedResult = await enhancedStrategy.execute(mockExecutionContext);
            const adapterResult = await adapterStrategy.execute(mockExecutionContext);

            // Both should successfully complete all phases
            expect(enhancedResult.success).toBe(true);
            expect(adapterResult.success).toBe(true);
            
            // Both should have high confidence (deterministic)
            expect(enhancedResult.metadata.confidence).toBe(1.0);
            expect(adapterResult.metadata.confidence).toBe(1.0);
        });

        it("should preserve execution routing logic", async () => {
            const executionTypes = [
                { stepType: "api_fetch", description: "API integration test" },
                { stepType: "execute_code", description: "Code execution test" },
                { stepType: "transform_data", description: "Data transformation test" },
                { stepType: "copy_values", description: "Direct mapping test" },
            ];

            for (const { stepType, description } of executionTypes) {
                const testContext = {
                    ...mockExecutionContext,
                    stepType,
                    config: {
                        ...mockExecutionContext.config,
                        description,
                    },
                };

                const enhancedResult = await enhancedStrategy.execute(testContext);
                const adapterResult = await adapterStrategy.execute(testContext);

                expect(enhancedResult.success).toBe(true);
                expect(adapterResult.success).toBe(true);
            }
        });

        it("should preserve legacy cost calculation model", () => {
            const testContexts = [
                { ...mockExecutionContext, stepType: "api_call", config: { ...mockExecutionContext.config, description: "API integration" } },
                { ...mockExecutionContext, stepType: "code_exec", config: { ...mockExecutionContext.config, description: "Code execution" } },
                { ...mockExecutionContext, stepType: "transform", config: { ...mockExecutionContext.config, description: "Data transformation" } },
                { ...mockExecutionContext, stepType: "simple", config: { ...mockExecutionContext.config, description: "Direct mapping" } },
            ];

            testContexts.forEach(context => {
                const enhancedEstimate = enhancedStrategy.estimateResources(context);
                const adapterEstimate = adapterStrategy.estimateResources(context);

                // Both should use similar cost models
                expect(enhancedEstimate.cost).toBeGreaterThan(0);
                expect(adapterEstimate.cost).toBeGreaterThan(0);
                
                // Costs should be in deterministic range (very low)
                expect(enhancedEstimate.cost).toBeLessThan(0.1);
                expect(adapterEstimate.cost).toBeLessThan(0.1);
            });
        });
    });

    describe("Cross-Architecture Compatibility", () => {
        it("should handle input validation consistently", async () => {
            const invalidContext = {
                ...mockExecutionContext,
                inputs: {}, // Missing required inputs
            };

            const enhancedResult = await enhancedStrategy.execute(invalidContext);
            const adapterResult = await adapterStrategy.execute(invalidContext);

            // Both should handle validation errors
            expect(enhancedResult.success).toBe(false);
            expect(adapterResult.success).toBe(false);
            
            expect(enhancedResult.error).toContain("validation");
            expect(adapterResult.error).toContain("validation");
        });

        it("should provide comparable error handling", async () => {
            const errorContext = {
                ...mockExecutionContext,
                config: null, // Invalid config to trigger error
            };

            const enhancedResult = await enhancedStrategy.execute(errorContext);
            const adapterResult = await adapterStrategy.execute(errorContext);

            // Both should handle errors gracefully
            expect(enhancedResult.success).toBe(false);
            expect(adapterResult.success).toBe(false);
            
            expect(enhancedResult.metadata.confidence).toBe(0);
            expect(adapterResult.metadata.confidence).toBe(0);
        });
    });

    describe("Performance and Quality Metrics", () => {
        it("should provide detailed performance insights", () => {
            const enhancedMetrics = enhancedStrategy.getPerformanceMetrics();
            const adapterMetrics = adapterStrategy.getPerformanceMetrics();

            // Both should provide comprehensive metrics
            expect(enhancedMetrics).toHaveProperty("totalExecutions");
            expect(enhancedMetrics).toHaveProperty("averageConfidence");
            expect(enhancedMetrics).toHaveProperty("evolutionScore");
            
            expect(adapterMetrics).toHaveProperty("totalExecutions");
            expect(adapterMetrics).toHaveProperty("averageConfidence");
            expect(adapterMetrics).toHaveProperty("evolutionScore");
            
            // Deterministic strategies should have high confidence
            expect(enhancedMetrics.averageConfidence).toBeGreaterThanOrEqual(0.9);
            expect(adapterMetrics.averageConfidence).toBeGreaterThanOrEqual(0.9);
        });

        it("should provide execution time comparisons", async () => {
            const start1 = Date.now();
            const enhancedResult = await enhancedStrategy.execute(mockExecutionContext);
            const enhanced1Time = Date.now() - start1;

            const start2 = Date.now();
            const adapterResult = await adapterStrategy.execute(mockExecutionContext);
            const adapter2Time = Date.now() - start2;

            expect(enhancedResult.success).toBe(true);
            expect(adapterResult.success).toBe(true);
            
            // Both should execute reasonably quickly for deterministic tasks
            expect(enhanced1Time).toBeLessThan(5000); // 5 seconds
            expect(adapter2Time).toBeLessThan(5000); // 5 seconds
        });
    });

    describe("Learning and Feedback Integration", () => {
        it("should accept and process feedback consistently", () => {
            const feedback = {
                outcome: "success" as const,
                performanceScore: 0.92,
                userSatisfaction: 0.88,
            };

            expect(() => enhancedStrategy.learn(feedback)).not.toThrow();
            expect(() => adapterStrategy.learn(feedback)).not.toThrow();
        });

        it("should provide improvement suggestions", async () => {
            const result = await enhancedStrategy.execute(mockExecutionContext);

            expect(result.success).toBe(true);
            expect(result.feedback.improvements).toBeDefined();
            expect(Array.isArray(result.feedback.improvements)).toBe(true);
        });
    });

    describe("Migration Validation", () => {
        it("should demonstrate successful migration from legacy to enhanced", async () => {
            // Test the same execution with both strategies
            const contexts = [
                mockExecutionContext,
                { ...mockExecutionContext, stepType: "api_integration", config: { ...mockExecutionContext.config, description: "API test" } },
                { ...mockExecutionContext, stepType: "transform_data", config: { ...mockExecutionContext.config, description: "Transform test" } },
            ];

            for (const context of contexts) {
                const enhancedResult = await enhancedStrategy.execute(context);
                const adapterResult = await adapterStrategy.execute(context);

                // Both should succeed
                expect(enhancedResult.success).toBe(true);
                expect(adapterResult.success).toBe(true);
                
                // Enhanced should provide richer feedback
                expect(enhancedResult.feedback.improvements).toBeDefined();
                expect(enhancedResult.metadata.resourceUsage).toBeDefined();
                
                // Both should maintain deterministic confidence
                expect(enhancedResult.metadata.confidence).toBe(1.0);
                expect(adapterResult.metadata.confidence).toBe(1.0);
            }
        });

        it("should validate template reusability for other strategy migrations", () => {
            // This test validates that our migration patterns can be reused
            const migrationPatterns = {
                interfaceAdapter: typeof adapterStrategy.execute === "function",
                enhancedStrategy: typeof enhancedStrategy.execute === "function",
                legacyPatternPreservation: enhancedStrategy.version.includes("enhanced"),
                performanceTracking: typeof enhancedStrategy.getPerformanceMetrics === "function",
                learningCapability: typeof enhancedStrategy.learn === "function",
            };

            Object.entries(migrationPatterns).forEach(([pattern, isImplemented]) => {
                expect(isImplemented).toBe(true);
            });
        });
    });
});