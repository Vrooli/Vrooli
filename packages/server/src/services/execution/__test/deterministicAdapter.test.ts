import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { DeterministicStrategyAdapter } from "../tier3/strategies/adapters/deterministicAdapter.js";
import { StrategyType } from "@vrooli/shared";

describe("DeterministicStrategy Adapter", () => {
    let adapter: DeterministicStrategyAdapter;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility

    const mockExecutionContext = {
        stepId: "adapter-test-123",
        stepType: "process_data",
        inputs: {
            data: { records: 50, type: "customer" },
            format: "json",
            validate: true,
        },
        config: {
            name: "Data Processing Task",
            description: "Process customer data with validation",
            requiredInputs: ["data", "format"],
            expectedOutputs: {
                processedData: { name: "Processed Data", description: "Cleaned customer data" },
                validation: { name: "Validation Results", description: "Data validation report" },
            },
            transformRules: {
                cleanNames: true,
                validateEmails: true,
            },
        },
        resources: {
            credits: 5000,
            tools: [],
        },
        history: {
            recentSteps: [
                { stepId: "step-122", strategy: StrategyType.CONVERSATIONAL, result: "success", duration: 2000 },
            ],
            totalExecutions: 15,
            successRate: 0.93,
        },
        constraints: {
            maxTokens: 1000,
            maxTime: 30000,
            requiredConfidence: 0.8,
        },
        metadata: {
            userId: "adapter-user-123",
        },
    };

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        
        adapter = new DeterministicStrategyAdapter(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Interface Compliance", () => {
        it("should implement ExecutionStrategy interface", () => {
            expect(adapter.type).toBe(StrategyType.DETERMINISTIC);
            expect(adapter.name).toBe("DeterministicStrategy");
            expect(adapter.version).toBe("1.0.0-adapter");
        });

        it("should have all required methods", () => {
            expect(typeof adapter.execute).toBe("function");
            expect(typeof adapter.canHandle).toBe("function");
            expect(typeof adapter.estimateResources).toBe("function");
            expect(typeof adapter.getPerformanceMetrics).toBe("function");
        });
    });

    describe("Legacy Pattern: canHandle", () => {
        it("should handle explicit strategy requests", () => {
            expect(adapter.canHandle("any_type", { strategy: "deterministic" })).toBe(true);
        });

        it("should handle deterministic keywords", () => {
            expect(adapter.canHandle("process_data")).toBe(true);
            expect(adapter.canHandle("transform_records")).toBe(true);
            expect(adapter.canHandle("calculate_metrics")).toBe(true);
            expect(adapter.canHandle("validate_inputs")).toBe(true);
        });

        it("should handle complex step descriptions", () => {
            expect(adapter.canHandle("execute_workflow", {
                name: "Data Processing Pipeline",
                description: "Automated batch processing of customer records",
            })).toBe(true);
        });

        it("should reject non-deterministic tasks", () => {
            expect(adapter.canHandle("chat_with_user", {
                name: "Customer Support",
                description: "Help customers with their questions",
            })).toBe(false);
        });
    });

    describe("Legacy Pattern: Resource Estimation", () => {
        it("should provide predictable cost estimates", () => {
            const estimate = adapter.estimateResources(mockExecutionContext);

            expect(estimate.tokens).toBe(0); // No LLM usage
            expect(estimate.apiCalls).toBeGreaterThanOrEqual(0);
            expect(estimate.computeTime).toBeGreaterThan(0);
            expect(estimate.cost).toBeGreaterThan(0);
        });

        it("should adjust estimates based on execution type", () => {
            const apiContext = {
                ...mockExecutionContext,
                stepType: "api_integration",
                config: {
                    ...mockExecutionContext.config,
                    description: "Fetch data from external API",
                },
            };

            const transformContext = {
                ...mockExecutionContext,
                stepType: "data_transformation",
                config: {
                    ...mockExecutionContext.config,
                    description: "Transform customer data format",
                },
            };

            const apiEstimate = adapter.estimateResources(apiContext);
            const transformEstimate = adapter.estimateResources(transformContext);

            expect(apiEstimate.apiCalls).toBe(1);
            expect(transformEstimate.apiCalls).toBe(0);
        });

        it("should scale costs with input/output complexity", () => {
            const simpleContext = {
                ...mockExecutionContext,
                inputs: { value: 42 },
                config: { expectedOutputs: { result: {} } },
            };

            const complexContext = {
                ...mockExecutionContext,
                inputs: {
                    data1: { large: "dataset" },
                    data2: { another: "dataset" },
                    config: { multiple: "parameters" },
                },
                config: {
                    expectedOutputs: {
                        result1: {},
                        result2: {},
                        metrics: {},
                        report: {},
                    },
                },
            };

            const simpleEstimate = adapter.estimateResources(simpleContext);
            const complexEstimate = adapter.estimateResources(complexContext);

            expect(complexEstimate.cost).toBeGreaterThan(simpleEstimate.cost);
        });
    });

    describe("Legacy Pattern: Execution", () => {
        it("should execute successfully with valid inputs", async () => {
            const result = await adapter.execute(mockExecutionContext);

            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();
            expect(result.metadata).toBeDefined();
            expect(result.metadata.strategyType).toBe(StrategyType.DETERMINISTIC);
            expect(result.metadata.confidence).toBe(1.0);
        });

        it("should detect and route to API integration execution", async () => {
            const apiContext = {
                ...mockExecutionContext,
                stepType: "api_fetch",
                config: {
                    ...mockExecutionContext.config,
                    description: "Fetch customer data from CRM API",
                    endpoint: "/api/customers",
                    method: "GET",
                },
            };

            const result = await adapter.execute(apiContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("apiResult", true);
            expect(result.result).toHaveProperty("endpoint", "/api/customers");
            expect(result.result).toHaveProperty("method", "GET");
        });

        it("should detect and route to code execution", async () => {
            const codeContext = {
                ...mockExecutionContext,
                stepType: "execute_script",
                config: {
                    ...mockExecutionContext.config,
                    description: "Run data validation script",
                },
            };

            const result = await adapter.execute(codeContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("codeResult", true);
            expect(result.result).toHaveProperty("computed");
        });

        it("should detect and route to data transformation", async () => {
            const transformContext = {
                ...mockExecutionContext,
                stepType: "transform_data",
                config: {
                    ...mockExecutionContext.config,
                    description: "Transform customer data to standard format",
                    transformRules: {
                        normalizeNames: true,
                        validateEmails: true,
                    },
                },
            };

            const result = await adapter.execute(transformContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("transformed", true);
            expect(result.result).toHaveProperty("rules");
        });

        it("should handle direct mapping for simple tasks", async () => {
            const simpleContext = {
                ...mockExecutionContext,
                stepType: "copy_data",
                config: {
                    ...mockExecutionContext.config,
                    description: "Simple data pass-through",
                },
            };

            const result = await adapter.execute(simpleContext);

            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();
        });
    });

    describe("Legacy Pattern: Validation", () => {
        it("should validate required inputs", async () => {
            const invalidContext = {
                ...mockExecutionContext,
                inputs: {
                    // Missing required 'data' and 'format' inputs
                    optional: "value",
                },
            };

            const result = await adapter.execute(invalidContext);

            expect(result.success).toBe(false);
            expect(result.error).toContain("validation failed");
        });

        it("should validate output generation", async () => {
            // This test would require mocking internal methods to force output validation failure
            // For now, we test that successful execution produces valid outputs
            const result = await adapter.execute(mockExecutionContext);

            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();
            expect(typeof result.result).toBe("object");
        });
    });

    describe("Error Handling", () => {
        it("should handle malformed context gracefully", async () => {
            const malformedContext = {
                ...mockExecutionContext,
                config: null, // Invalid config
            };

            const result = await adapter.execute(malformedContext);

            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();
            expect(result.metadata.confidence).toBe(0);
        });

        it("should provide meaningful error messages", async () => {
            const errorContext = {
                ...mockExecutionContext,
                inputs: {}, // No inputs when required
            };

            const result = await adapter.execute(errorContext);

            expect(result.success).toBe(false);
            expect(result.error).toContain("validation failed");
            expect(result.feedback.issues).toBeDefined();
        });
    });

    describe("Performance Metrics", () => {
        it("should provide performance metrics compatible with new architecture", () => {
            const metrics = adapter.getPerformanceMetrics();

            expect(metrics).toHaveProperty("totalExecutions");
            expect(metrics).toHaveProperty("successCount");
            expect(metrics).toHaveProperty("failureCount");
            expect(metrics).toHaveProperty("averageExecutionTime");
            expect(metrics).toHaveProperty("averageResourceUsage");
            expect(metrics).toHaveProperty("averageConfidence");
            expect(metrics).toHaveProperty("evolutionScore");

            // Deterministic strategies should have high confidence and evolution scores
            expect(metrics.averageConfidence).toBeGreaterThanOrEqual(0.9);
            expect(metrics.evolutionScore).toBeGreaterThanOrEqual(0.8);
        });
    });

    // Learning mechanism now happens through event-driven architecture
    // Tests removed: "Learning Mechanism" describe block

    describe("Legacy Cost Model", () => {
        it("should calculate costs using legacy constants", () => {
            const contexts = [
                { ...mockExecutionContext, stepType: "api_call", config: { ...mockExecutionContext.config, description: "API integration" } },
                { ...mockExecutionContext, stepType: "code_exec", config: { ...mockExecutionContext.config, description: "Code execution" } },
                { ...mockExecutionContext, stepType: "transform", config: { ...mockExecutionContext.config, description: "Data transformation" } },
                { ...mockExecutionContext, stepType: "simple", config: { ...mockExecutionContext.config, description: "Direct mapping" } },
            ];

            const estimates = contexts.map(ctx => adapter.estimateResources(ctx));

            // API calls should be most expensive
            expect(estimates[0].cost).toBeGreaterThan(estimates[3].cost);
            // Code execution should be second most expensive
            expect(estimates[1].cost).toBeGreaterThan(estimates[2].cost);
            // All should be affordable deterministic costs
            estimates.forEach(estimate => {
                expect(estimate.cost).toBeLessThan(0.1); // Very low cost
            });
        });
    });
});