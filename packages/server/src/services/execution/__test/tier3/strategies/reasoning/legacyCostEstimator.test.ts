import { describe, it, beforeEach, vi, expect } from "vitest";
import { type ExecutionContext } from "@vrooli/shared";
import { LegacyCostEstimator } from "../legacyCostEstimator.js";

describe("LegacyCostEstimator", () => {
    let estimator: LegacyCostEstimator;
    let baseContext: ExecutionContext;

    beforeEach(() => {
        estimator = new LegacyCostEstimator();

        baseContext = {
            stepId: "test-step-123",
            stepType: "analyze_data",
            inputs: {
                data: [1, 2, 3, 4, 5],
                threshold: 3,
            },
            config: {
                expectedOutputs: {
                    result: { type: "Boolean" },
                    summary: { type: "Text" },
                },
            },
            resources: {
                models: [],
                tools: [],
                apis: [],
                credits: 5000,
            },
            history: {
                recentSteps: [
                    { stepId: "step1", strategy: "REASONING" as any, result: "success", duration: 1000 },
                    { stepId: "step2", strategy: "CONVERSATIONAL" as any, result: "success", duration: 2000 },
                ],
                totalExecutions: 10,
                successRate: 0.9,
            },
            constraints: {
                maxTokens: 2000,
                maxTime: 30000,
                requiredConfidence: 0.8,
            },
        };
    });

    describe("estimateResources", () => {
        it("should return ResourceUsage with all required fields", () => {
            const result = estimator.estimateResources(baseContext);

            expect(result).toHaveProperty("tokens");
            expect(result).toHaveProperty("apiCalls");
            expect(result).toHaveProperty("computeTime");
            expect(result).toHaveProperty("cost");

            expect(typeof result.tokens).toBe("number");
            expect(typeof result.apiCalls).toBe("number");
            expect(typeof result.computeTime).toBe("number");
            expect(typeof result.cost).toBe("number");

            expect(result.tokens).toBeGreaterThan(0);
            expect(result.apiCalls).toBeGreaterThan(0);
            expect(result.computeTime).toBeGreaterThan(0);
            expect(result.cost).toBeGreaterThan(0);
        });

        it("should calculate costs based on legacy patterns", () => {
            const result = estimator.estimateResources(baseContext);

            // Should include base cost, token cost, complexity cost, and tool cost
            const expectedBaseCost = 50; // BASE_REASONING_COST
            const inputCount = 2;
            const outputCount = 2;
            const expectedComplexityCost = (inputCount + outputCount) * 3; // COMPLEXITY_FACTOR

            // Cost should include all components
            expect(result.cost).toBeGreaterThan((expectedBaseCost + expectedComplexityCost) * 0.00001);
        });

        it("should scale with input/output complexity", () => {
            const simpleContext = {
                ...baseContext,
                inputs: { simple: "value" },
                config: { expectedOutputs: { result: { type: "Boolean" } } },
            };

            const complexContext = {
                ...baseContext,
                inputs: {
                    input1: "value1",
                    input2: "value2",
                    input3: "value3",
                    input4: "value4",
                    input5: "value5",
                },
                config: {
                    expectedOutputs: {
                        output1: { type: "Text" },
                        output2: { type: "Number" },
                        output3: { type: "Boolean" },
                        output4: { type: "Array" },
                    },
                },
            };

            const simpleResult = estimator.estimateResources(simpleContext);
            const complexResult = estimator.estimateResources(complexContext);

            expect(complexResult.tokens).toBeGreaterThan(simpleResult.tokens);
            expect(complexResult.cost).toBeGreaterThan(simpleResult.cost);
            expect(complexResult.computeTime).toBeGreaterThan(simpleResult.computeTime);
        });

        it("should account for historical complexity", () => {
            const contextNoHistory = {
                ...baseContext,
                history: {
                    recentSteps: [],
                    totalExecutions: 0,
                    successRate: 1.0,
                },
            };

            const contextWithHistory = {
                ...baseContext,
                history: {
                    recentSteps: Array.from({ length: 10 }, (_, i) => ({
                        stepId: `step${i}`,
                        strategy: "REASONING" as any,
                        result: "success" as const,
                        duration: 1000,
                    })),
                    totalExecutions: 100,
                    successRate: 0.95,
                },
            };

            const resultNoHistory = estimator.estimateResources(contextNoHistory);
            const resultWithHistory = estimator.estimateResources(contextWithHistory);

            expect(resultWithHistory.tokens).toBeGreaterThan(resultNoHistory.tokens);
            expect(resultWithHistory.computeTime).toBeGreaterThan(resultNoHistory.computeTime);
        });

        it("should include tool costs when tools are available", () => {
            const contextNoTools = {
                ...baseContext,
                resources: {
                    ...baseContext.resources,
                    tools: [],
                },
            };

            const contextWithTools = {
                ...baseContext,
                resources: {
                    ...baseContext.resources,
                    tools: [
                        { name: "calculator", type: "math", description: "Math operations", parameters: {} },
                        { name: "validator", type: "validation", description: "Data validation", parameters: {} },
                        { name: "formatter", type: "utility", description: "Format output", parameters: {} },
                    ],
                },
            };

            const resultNoTools = estimator.estimateResources(contextNoTools);
            const resultWithTools = estimator.estimateResources(contextWithTools);

            expect(resultWithTools.cost).toBeGreaterThan(resultNoTools.cost);
            expect(resultWithTools.apiCalls).toBeGreaterThan(resultNoTools.apiCalls);
        });

        it("should adjust for different step types", () => {
            const stepTypes = [
                { type: "analyze_performance", expectedMultiplier: 1.5 },
                { type: "decide_option", expectedMultiplier: 1.3 },
                { type: "validate_results", expectedMultiplier: 1.2 },
                { type: "reason_about_data", expectedMultiplier: 1.4 },
                { type: "simple_task", expectedMultiplier: 1.0 },
            ];

            const baseResult = estimator.estimateResources({
                ...baseContext,
                stepType: "simple_task",
            });

            stepTypes.forEach(({ type, expectedMultiplier }) => {
                const result = estimator.estimateResources({
                    ...baseContext,
                    stepType: type,
                });

                if (expectedMultiplier > 1.0) {
                    expect(result.tokens).toBeGreaterThan(baseResult.tokens);
                } else {
                    expect(result.tokens).toBeGreaterThanOrEqual(baseResult.tokens);
                }
            });
        });

        it("should ensure minimum token estimate", () => {
            const minimalContext = {
                ...baseContext,
                inputs: {},
                config: { expectedOutputs: {} },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const result = estimator.estimateResources(minimalContext);

            // Should return at least the default token estimate (750)
            expect(result.tokens).toBeGreaterThanOrEqual(750);
        });

        it("should cap complexity at reasonable maximum", () => {
            const extremeContext = {
                ...baseContext,
                inputs: Array.from({ length: 50 }, (_, i) => [`input${i}`, `value${i}`])
                    .reduce((obj, [key, value]) => ({ ...obj, [key]: value }), {}),
                config: {
                    expectedOutputs: Array.from({ length: 20 }, (_, i) => [`output${i}`, { type: "Text" }])
                        .reduce((obj, [key, value]) => ({ ...obj, [key]: value }), {}),
                },
                history: {
                    recentSteps: Array.from({ length: 100 }, (_, i) => ({
                        stepId: `step${i}`,
                        strategy: "REASONING" as any,
                        result: "success" as const,
                        duration: 1000,
                    })),
                    totalExecutions: 1000,
                    successRate: 0.99,
                },
                constraints: {
                    maxTokens: 10000,
                    maxTime: 120000,
                    requiredConfidence: 0.95,
                },
            };

            const result = estimator.estimateResources(extremeContext);

            // Compute time should be capped at 60 seconds (60000ms)
            expect(result.computeTime).toBeLessThanOrEqual(60000);

            // Should still be reasonable despite extreme inputs
            expect(result.tokens).toBeLessThan(20000); // Some reasonable upper bound
        });
    });

    describe("createResourceUsageSnapshot", () => {
        it("should create ResourceUsage from actual usage data", () => {
            const snapshot = estimator.createResourceUsageSnapshot(
                1500, // actualTokens
                6,    // actualApiCalls
                25000, // actualComputeTime
                800   // actualCreditsUsed
            );

            expect(snapshot).toEqual({
                tokens: 1500,
                apiCalls: 6,
                computeTime: 25000,
                cost: 0.008, // 800 * 0.00001
            });
        });

        it("should handle zero values correctly", () => {
            const snapshot = estimator.createResourceUsageSnapshot(0, 0, 0, 0);

            expect(snapshot).toEqual({
                tokens: 0,
                apiCalls: 0,
                computeTime: 0,
                cost: 0,
            });
        });

        it("should convert credits to cost correctly", () => {
            const testCases = [
                { credits: 100, expectedCost: 0.001 },
                { credits: 1000, expectedCost: 0.01 },
                { credits: 5000, expectedCost: 0.05 },
                { credits: 10000, expectedCost: 0.1 },
            ];

            testCases.forEach(({ credits, expectedCost }) => {
                const snapshot = estimator.createResourceUsageSnapshot(0, 0, 0, credits);
                expect(snapshot.cost).toBeCloseTo(expectedCost, 5);
            });
        });
    });

    describe("legacy pattern preservation", () => {
        it("should maintain legacy cost constants", () => {
            // Verify that the legacy cost mapping constants are preserved
            const legacyConstants = (LegacyCostEstimator as any).LEGACY_COST_MAPPING;

            expect(legacyConstants).toEqual({
                BASE_REASONING_COST: 50,
                TOKEN_COST_MULTIPLIER: 1,
                TOOL_REASONING_COST: 20,
                COMPLEXITY_FACTOR: 3,
                DEFAULT_TOKEN_ESTIMATE: 750,
            });
        });

        it("should use legacy token estimation formula", () => {
            // Test the legacy pattern: baseTokens + ioTokens + complexityTokens
            const context = {
                ...baseContext,
                inputs: { input1: "a", input2: "b" }, // 2 inputs
                config: {
                    expectedOutputs: {
                        output1: { type: "Text" },
                        output2: { type: "Boolean" },
                        output3: { type: "Number" },
                    }, // 3 outputs
                },
            };

            const result = estimator.estimateResources(context);

            // Verify token calculation includes all legacy components
            // Base: 500, I/O: (2+3)*50 = 250, Complexity: varies, StepType multiplier: varies
            expect(result.tokens).toBeGreaterThan(750); // Should exceed base + I/O
        });

        it("should preserve legacy complexity calculation", () => {
            const testContext = {
                ...baseContext,
                inputs: { a: 1, b: 2, c: 3 }, // 3 inputs = 0.6 complexity
                config: {
                    expectedOutputs: {
                        out1: { type: "Text" },
                        out2: { type: "Boolean" },
                    }, // 2 outputs = 0.6 complexity
                },
                history: {
                    recentSteps: Array.from({ length: 5 }, (_, i) => ({
                        stepId: `step${i}`,
                        strategy: "REASONING" as any,
                        result: "success" as const,
                        duration: 1000,
                    })), // 5 steps = 0.5 complexity (capped)
                    totalExecutions: 20,
                    successRate: 0.95,
                },
                constraints: {
                    requiredConfidence: 0.9, // +0.5 complexity
                    maxExecutionTime: 60000, // +0.3 complexity
                },
            };

            const result = estimator.estimateResources(testContext);

            // Total complexity should be: 1 + 0.6 + 0.6 + 0.5 + 0.5 + 0.3 = 3.5, capped at 3
            // This should affect token count and compute time
            expect(result.tokens).toBeGreaterThan(1200); // Base + significant complexity
            expect(result.computeTime).toBeGreaterThan(20000); // Base + complexity time
        });

        it("should preserve legacy API call estimation (4-phase pattern)", () => {
            const result = estimator.estimateResources(baseContext);

            // Legacy pattern: 4 base calls (4-phase execution)
            expect(result.apiCalls).toBeGreaterThanOrEqual(4);
        });

        it("should preserve legacy tool usage probability", () => {
            const calculationContext = {
                ...baseContext,
                stepType: "calculate_statistics",
                resources: {
                    ...baseContext.resources,
                    tools: [
                        { name: "calculator", type: "math", description: "Calculator", parameters: {} },
                    ],
                },
            };

            const analysisContext = {
                ...baseContext,
                stepType: "analyze_trends",
                resources: {
                    ...baseContext.resources,
                    tools: [
                        { name: "analyzer", type: "analysis", description: "Analyzer", parameters: {} },
                    ],
                },
            };

            const calcResult = estimator.estimateResources(calculationContext);
            const analysisResult = estimator.estimateResources(analysisContext);

            // Calculation tasks should have higher tool usage probability (0.8)
            // Analysis tasks should have medium-high probability (0.6)
            // Both should be higher than no-tool scenarios
            const noToolResult = estimator.estimateResources({
                ...baseContext,
                resources: { ...baseContext.resources, tools: [] },
            });

            expect(calcResult.cost).toBeGreaterThan(noToolResult.cost);
            expect(analysisResult.cost).toBeGreaterThan(noToolResult.cost);
        });

        it("should maintain legacy time estimation patterns", () => {
            const contexts = [
                { complexity: "low", inputs: 1, outputs: 1, history: 0 },
                { complexity: "medium", inputs: 3, outputs: 2, history: 3 },
                { complexity: "high", inputs: 8, outputs: 5, history: 10 },
            ];

            const results = contexts.map(({ inputs, outputs, history }) => {
                const context = {
                    ...baseContext,
                    inputs: Array.from({ length: inputs }, (_, i) => [`input${i}`, `value${i}`])
                        .reduce((obj, [key, value]) => ({ ...obj, [key]: value }), {}),
                    config: {
                        expectedOutputs: Array.from({ length: outputs }, (_, i) => [`output${i}`, { type: "Text" }])
                            .reduce((obj, [key, value]) => ({ ...obj, [key]: value }), {}),
                    },
                    history: {
                        recentSteps: Array.from({ length: history }, (_, i) => ({
                            stepId: `step${i}`,
                            strategy: "REASONING" as any,
                            result: "success" as const,
                            duration: 1000,
                        })),
                        totalExecutions: history * 2,
                        successRate: 1.0,
                    },
                };

                return estimator.estimateResources(context);
            });

            // Time should increase with complexity
            expect(results[1].computeTime).toBeGreaterThan(results[0].computeTime);
            expect(results[2].computeTime).toBeGreaterThan(results[1].computeTime);

            // Base time should be 15 seconds (15000ms)
            expect(results[0].computeTime).toBeGreaterThanOrEqual(15000);
        });
    });
});