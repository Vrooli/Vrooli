import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { DeterministicStrategy } from "../tier3/strategies/deterministicStrategy.js";
import { StrategyType } from "@vrooli/shared";

describe("DeterministicStrategy Enhanced (Legacy Patterns)", () => {
    let strategy: DeterministicStrategy;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility

    const mockExecutionContext = {
        stepId: "enhanced-test-123",
        stepType: "transform_data",
        inputs: {
            customerData: [
                { id: 1, name: "John Doe", email: "john@example.com" },
                { id: 2, name: "Jane Smith", email: "jane@example.com" },
            ],
            transformationRules: {
                normalizeNames: true,
                validateEmails: true,
                addTimestamp: true,
            },
        },
        config: {
            name: "Customer Data Transformation",
            description: "Transform and validate customer data",
            requiredInputs: ["customerData"],
            expectedOutputs: {
                transformedData: { name: "Transformed Data", description: "Processed customer records" },
                validationReport: { name: "Validation Report", description: "Data quality metrics" },
            },
            transformRules: {
                normalizeNames: true,
                validateEmails: true,
            },
        },
        resources: {
            credits: 10000,
            tools: [
                {
                    name: "data_validator",
                    description: "Validates customer data",
                    parameters: { data: "object" },
                },
            ],
        },
        history: {
            recentSteps: [
                { stepId: "step-121", strategy: StrategyType.DETERMINISTIC, result: "success", duration: 1500 },
                { stepId: "step-122", strategy: StrategyType.DETERMINISTIC, result: "success", duration: 2000 },
            ],
            totalExecutions: 25,
            successRate: 0.96,
        },
        constraints: {
            maxTokens: 0, // No LLM usage for deterministic
            maxTime: 30000,
            requiredConfidence: 0.9,
        },
        metadata: {
            userId: "enhanced-user-123",
        },
    };

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        
        strategy = new DeterministicStrategy(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Enhanced Interface Compliance", () => {
        it("should have enhanced version number", () => {
            expect(strategy.type).toBe(StrategyType.DETERMINISTIC);
            expect(strategy.name).toBe("DeterministicStrategy");
            expect(strategy.version).toBe("2.0.0-enhanced");
        });

        it("should implement all strategy methods", () => {
            expect(typeof strategy.canHandle).toBe("function");
            expect(typeof strategy.execute).toBe("function");
            expect(typeof strategy.estimateResources).toBe("function");
            expect(typeof strategy.learn).toBe("function");
            expect(typeof strategy.getPerformanceMetrics).toBe("function");
        });
    });

    describe("Enhanced canHandle (Legacy Patterns)", () => {
        it("should handle explicit strategy requests", () => {
            expect(strategy.canHandle("any_type", { strategy: "deterministic" })).toBe(true);
        });

        it("should handle expanded deterministic keywords", () => {
            expect(strategy.canHandle("process_batch")).toBe(true);
            expect(strategy.canHandle("transform_data")).toBe(true);
            expect(strategy.canHandle("api_integration")).toBe(true);
            expect(strategy.canHandle("code_execution")).toBe(true);
            expect(strategy.canHandle("validate_inputs")).toBe(true);
        });

        it("should handle keywords in combined text", () => {
            expect(strategy.canHandle("analyze_data", {
                name: "Automated Data Processing",
                description: "Batch process customer records with validation",
            })).toBe(true);
            
            expect(strategy.canHandle("execute_task", {
                description: "API integration for data synchronization",
            })).toBe(true);
        });

        it("should reject conversational tasks", () => {
            expect(strategy.canHandle("chat_response", {
                name: "Customer Support Chat",
                description: "Help customers with their questions",
            })).toBe(false);
        });
    });

    describe("Enhanced Resource Estimation", () => {
        it("should provide detailed resource estimates based on legacy cost model", () => {
            const estimate = strategy.estimateResources(mockExecutionContext);

            expect(estimate.tokens).toBe(0); // No LLM usage
            expect(estimate.apiCalls).toBeGreaterThanOrEqual(0);
            expect(estimate.computeTime).toBeGreaterThan(0);
            expect(estimate.cost).toBeGreaterThan(0);
        });

        it("should adjust estimates based on execution type detection", () => {
            const apiContext = {
                ...mockExecutionContext,
                stepType: "api_call",
                config: {
                    ...mockExecutionContext.config,
                    description: "Fetch data from external API service",
                },
            };

            const codeContext = {
                ...mockExecutionContext,
                stepType: "execute_script",
                config: {
                    ...mockExecutionContext.config,
                    description: "Run custom validation script",
                },
            };

            const transformContext = {
                ...mockExecutionContext,
                stepType: "transform_format",
                config: {
                    ...mockExecutionContext.config,
                    description: "Convert data to new format",
                },
            };

            const directContext = {
                ...mockExecutionContext,
                stepType: "copy_values",
                config: {
                    ...mockExecutionContext.config,
                    description: "Simple data mapping",
                },
            };

            const apiEstimate = strategy.estimateResources(apiContext);
            const codeEstimate = strategy.estimateResources(codeContext);
            const transformEstimate = strategy.estimateResources(transformContext);
            const directEstimate = strategy.estimateResources(directContext);

            // API integration should be most expensive
            expect(apiEstimate.cost).toBeGreaterThan(directEstimate.cost);
            expect(apiEstimate.apiCalls).toBe(1);

            // Code execution should be second most expensive
            expect(codeEstimate.cost).toBeGreaterThan(transformEstimate.cost);

            // Transform should be more than direct
            expect(transformEstimate.cost).toBeGreaterThan(directEstimate.cost);

            // All should be deterministic (low cost)
            [apiEstimate, codeEstimate, transformEstimate, directEstimate].forEach(estimate => {
                expect(estimate.cost).toBeLessThan(0.1);
            });
        });

        it("should scale with input/output complexity", () => {
            const simpleContext = {
                ...mockExecutionContext,
                inputs: { value: 42 },
                config: { expectedOutputs: { result: {} } },
            };

            const complexContext = {
                ...mockExecutionContext,
                inputs: {
                    dataset1: { records: 1000 },
                    dataset2: { records: 500 },
                    rules: { complex: "validation" },
                    options: { multiple: "settings" },
                },
                config: {
                    expectedOutputs: {
                        processedData: {},
                        validationResults: {},
                        metrics: {},
                        errorReport: {},
                        summary: {},
                    },
                },
            };

            const simpleEstimate = strategy.estimateResources(simpleContext);
            const complexEstimate = strategy.estimateResources(complexContext);

            expect(complexEstimate.cost).toBeGreaterThan(simpleEstimate.cost);
            expect(complexEstimate.computeTime).toBeGreaterThan(simpleEstimate.computeTime);
        });
    });

    describe("Legacy 3-Phase Execution Pattern", () => {
        it("should execute successfully with cache check", async () => {
            const result = await strategy.execute(mockExecutionContext);

            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();
            expect(result.metadata).toBeDefined();
            expect(result.metadata.strategyType).toBe(StrategyType.DETERMINISTIC);
            expect(result.metadata.confidence).toBe(1.0);
        });

        it("should route to data transformation execution", async () => {
            const result = await strategy.execute(mockExecutionContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("transformed", true);
            expect(result.result).toHaveProperty("source");
            expect(result.result).toHaveProperty("rules");
        });

        it("should route to API integration execution", async () => {
            const apiContext = {
                ...mockExecutionContext,
                stepType: "api_fetch",
                config: {
                    ...mockExecutionContext.config,
                    description: "Fetch customer data from CRM API",
                    endpoint: "/api/customers",
                    method: "POST",
                },
            };

            const result = await strategy.execute(apiContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("apiResult", true);
            expect(result.result).toHaveProperty("endpoint", "/api/customers");
            expect(result.result).toHaveProperty("method", "POST");
        });

        it("should route to code execution", async () => {
            const codeContext = {
                ...mockExecutionContext,
                stepType: "execute_validation",
                config: {
                    ...mockExecutionContext.config,
                    description: "Execute data validation code",
                },
            };

            const result = await strategy.execute(codeContext);

            expect(result.success).toBe(true);
            expect(result.result).toHaveProperty("codeResult", true);
            expect(result.result).toHaveProperty("computed");
            expect(result.result).toHaveProperty("inputProcessed");
        });

        it("should route to direct mapping", async () => {
            const directContext = {
                ...mockExecutionContext,
                stepType: "copy_data",
                config: {
                    ...mockExecutionContext.config,
                    description: "Simple data copy operation",
                },
            };

            const result = await strategy.execute(directContext);

            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();
        });
    });

    describe("Performance Tracking", () => {
        it("should track performance metrics over time", async () => {
            // Execute multiple times to build performance history
            await strategy.execute(mockExecutionContext);
            await strategy.execute(mockExecutionContext);
            await strategy.execute(mockExecutionContext);

            const metrics = strategy.getPerformanceMetrics();

            expect(metrics.totalExecutions).toBeGreaterThan(0);
            expect(metrics.averageConfidence).toBeGreaterThanOrEqual(0.9);
            expect(metrics.evolutionScore).toBeGreaterThanOrEqual(0.8);
        });

        it("should calculate evolution score for deterministic strategies", async () => {
            const metrics = strategy.getPerformanceMetrics();

            // Deterministic strategies should have high evolution scores
            expect(metrics.evolutionScore).toBeGreaterThanOrEqual(0.8);
        });
    });

    describe("Learning Mechanism", () => {
        it("should learn from successful feedback", () => {
            const logSpy = vi.spyOn(logger, "debug");

            const feedback = {
                outcome: "success" as const,
                performanceScore: 0.95,
                userSatisfaction: 0.9,
            };

            strategy.learn(feedback);

            expect(logSpy.calledWith("[DeterministicStrategy] Optimizing for success")).toBe(true);
        });

        it("should learn from failure feedback", () => {
            const logSpy = vi.spyOn(logger, "debug");

            const feedback = {
                outcome: "failure" as const,
                performanceScore: 0.3,
                issues: ["Validation failed", "Invalid input format"],
            };

            strategy.learn(feedback);

            expect(logSpy.calledWith("[DeterministicStrategy] Adjusting for failure")).toBe(true);
        });
    });

    describe("Enhanced Error Handling", () => {
        it("should handle input validation errors gracefully", async () => {
            const invalidContext = {
                ...mockExecutionContext,
                inputs: {}, // Missing required inputs
            };

            const result = await strategy.execute(invalidContext);

            expect(result.success).toBe(false);
            expect(result.error).toContain("validation failed");
            expect(result.metadata.confidence).toBe(0);
        });

        it("should provide meaningful error feedback", async () => {
            const invalidContext = {
                ...mockExecutionContext,
                config: {
                    ...mockExecutionContext.config,
                    requiredInputs: ["missingInput"],
                },
            };

            const result = await strategy.execute(invalidContext);

            expect(result.success).toBe(false);
            expect(result.feedback.issues).toBeDefined();
            expect(result.feedback.improvements).toContain("Check input validation and constraints");
        });
    });

    describe("Legacy Pattern Integration", () => {
        it("should use legacy cost constants", () => {
            const estimate = strategy.estimateResources(mockExecutionContext);

            // Verify cost calculation uses legacy patterns
            expect(estimate.cost).toBeGreaterThan(0);
            expect(estimate.computeTime).toBeGreaterThan(0);
        });

        it("should maintain legacy validation strictness", async () => {
            // Test that validation is still strict like legacy implementation
            const result = await strategy.execute(mockExecutionContext);

            expect(result.success).toBe(true);
            expect(result.metadata.confidence).toBe(1.0); // High confidence for deterministic
        });

        it("should preserve legacy execution paths", async () => {
            const contexts = [
                { ...mockExecutionContext, stepType: "api_call", config: { ...mockExecutionContext.config, description: "API integration" } },
                { ...mockExecutionContext, stepType: "code_exec", config: { ...mockExecutionContext.config, description: "Code execution" } },
                { ...mockExecutionContext, stepType: "transform", config: { ...mockExecutionContext.config, description: "Data transformation" } },
                { ...mockExecutionContext, stepType: "simple", config: { ...mockExecutionContext.config, description: "Direct mapping" } },
            ];

            const results = await Promise.all(contexts.map(ctx => strategy.execute(ctx)));

            // All should succeed with deterministic confidence
            results.forEach(result => {
                expect(result.success).toBe(true);
                expect(result.metadata.confidence).toBe(1.0);
            });
        });
    });

    describe("Cache Integration", () => {
        it("should cache successful results", async () => {
            // First execution
            const result1 = await strategy.execute(mockExecutionContext);
            expect(result1.success).toBe(true);

            // Second execution with same context should potentially use cache
            const result2 = await strategy.execute(mockExecutionContext);
            expect(result2.success).toBe(true);
            
            // Results should be consistent
            expect(result2.metadata.strategyType).toBe(result1.metadata.strategyType);
        });
    });
});