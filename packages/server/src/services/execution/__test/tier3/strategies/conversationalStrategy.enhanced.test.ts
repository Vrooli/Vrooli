import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import winston from "winston";
import { ConversationalStrategy } from "../../../tier3/strategies/conversationalStrategy.js";
import { StrategyType } from "@vrooli/shared";

describe("ConversationalStrategy Enhanced (Legacy Patterns)", () => {
    let strategy: ConversationalStrategy;
    let logger: winston.Logger;

    const mockExecutionContext = {
        stepId: "step-123",
        stepType: "chat_response",
        inputs: {
            message: "Hello, I need help with data analysis",
            data: { records: 100, type: "customer" },
        },
        config: {
            name: "Data Analysis Chat",
            description: "Help user analyze customer data",
            instructions: "Provide detailed analysis and recommendations",
            model: "gpt-4o-mini",
            temperature: 0.7,
            expectedOutputs: {
                analysis: { name: "Analysis Results", description: "Detailed analysis" },
                recommendations: { name: "Recommendations", description: "Action items" },
            },
        },
        resources: {
            credits: 5000,
            tools: [
                {
                    name: "data_analyzer",
                    description: "Analyzes datasets",
                    parameters: { data: "object" },
                },
            ],
        },
        history: {
            recentSteps: [
                { stepId: "step-122", strategy: StrategyType.DETERMINISTIC, result: "success", duration: 2000 },
            ],
            totalExecutions: 10,
            successRate: 0.9,
        },
        constraints: {
            maxTokens: 2000,
            maxTime: 30000,
            requiredConfidence: 0.8,
        },
        metadata: {
            userId: "user-123",
        },
    };

    beforeEach(() => {
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        
        strategy = new ConversationalStrategy(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Enhanced Interface Compliance", () => {
        it("should have enhanced version number", () => {
            expect(strategy.type).toBe(StrategyType.CONVERSATIONAL);
            expect(strategy.name).toBe("ConversationalStrategy");
            expect(strategy.version).toBe("2.0.0-enhanced");
        });

        it("should implement all strategy methods", () => {
            expect(typeof strategy.canHandle).toBe("function");
            expect(typeof strategy.execute).toBe("function");
            expect(typeof strategy.estimateResources).toBe("function");
            expect(typeof strategy.getPerformanceMetrics).toBe("function");
        });
    });

    describe("Enhanced canHandle (Legacy Patterns)", () => {
        it("should handle explicit strategy requests", () => {
            expect(strategy.canHandle("any_type", { strategy: "conversational" })).toBe(true);
            expect(strategy.canHandle("any_type", { executionStrategy: "conversational" })).toBe(true);
        });

        it("should handle web routines (legacy pattern)", () => {
            expect(strategy.canHandle("RoutineWeb")).toBe(true);
            expect(strategy.canHandle("web")).toBe(true);
        });

        it("should handle conversational keywords in combined text", () => {
            expect(strategy.canHandle("process_data", {
                name: "Customer Support Chat",
                description: "Help customers with their questions",
                instructions: "Engage in friendly conversation",
            })).toBe(true);
        });

        it("should reject pure deterministic tasks", () => {
            expect(strategy.canHandle("calculate_sum")).toBe(false);
            expect(strategy.canHandle("database_query")).toBe(false);
            expect(strategy.canHandle("api_call")).toBe(false);
        });
    });

    describe("Enhanced Execute", () => {
        it("should execute with mocked LLM", async () => {
            const result = await strategy.execute(mockExecutionContext as any);

            expect(result.success).toBe(true);
            expect(result.outputs).toHaveProperty("response");
            expect(result.outputs.response).toContain("mock LLM");
            expect(result.metadata.strategy).toBe("conversational");
            expect(result.metadata.tokensUsed).toBeGreaterThan(0);
            expect(result.metadata.performance.duration).toBeGreaterThan(0);
        });

        it("should handle missing inputs gracefully", async () => {
            const contextWithoutInputs = {
                ...mockExecutionContext,
                inputs: {},
            };

            const result = await strategy.execute(contextWithoutInputs as any);

            expect(result.success).toBe(true);
            expect(result.outputs.response).toContain("mock LLM");
        });

        it("should extract relevant information from complex inputs", async () => {
            const complexContext = {
                ...mockExecutionContext,
                inputs: {
                    message: "Analyze this",
                    data: { nested: { value: 42 } },
                    metadata: { type: "analysis" },
                },
            };

            const result = await strategy.execute(complexContext as any);

            expect(result.success).toBe(true);
            // The strategy should have processed the complex inputs
            expect(result.metadata.inputTokens).toBeGreaterThan(50);
        });

        it("should respect token constraints", async () => {
            const constrainedContext = {
                ...mockExecutionContext,
                constraints: {
                    ...mockExecutionContext.constraints,
                    maxTokens: 100,
                },
            };

            const result = await strategy.execute(constrainedContext as any);

            expect(result.success).toBe(true);
            expect(result.metadata.tokensUsed).toBeLessThanOrEqual(100);
        });

        it("should handle tool integration (stubbed)", async () => {
            const result = await strategy.execute(mockExecutionContext as any);

            expect(result.success).toBe(true);
            expect(result.metadata.toolsUsed).toBe(0); // Mocked implementation doesn't use tools
        });

        it("should handle errors gracefully", async () => {
            // Create a context that will cause an error
            const errorContext = {
                ...mockExecutionContext,
                // Remove required fields to trigger validation error
                stepId: undefined,
            };

            const result = await strategy.execute(errorContext as any);

            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();
            expect(result.metadata.confidence).toBe(0);
        });
    });

    describe("Enhanced Resource Estimation", () => {
        it("should estimate resources based on complexity", () => {
            const estimate = strategy.estimateResources(mockExecutionContext as any);

            expect(estimate.tokens).toBeGreaterThan(500);
            expect(estimate.tokens).toBeLessThan(2500);
            expect(estimate.apiCalls).toBe(1);
            expect(estimate.computeTime).toBeGreaterThan(5000);
            expect(estimate.cost).toBeGreaterThan(0);
        });

        it("should adjust estimates for web routines", () => {
            const webContext = {
                ...mockExecutionContext,
                stepType: "RoutineWeb",
            };

            const estimate = strategy.estimateResources(webContext as any);

            expect(estimate.tokens).toBeGreaterThan(1000);
            expect(estimate.computeTime).toBeGreaterThan(10000);
        });

        it("should factor in tool usage", () => {
            const toolContext = {
                ...mockExecutionContext,
                resources: {
                    ...mockExecutionContext.resources,
                    tools: [
                        { name: "tool1", description: "desc1", parameters: {} },
                        { name: "tool2", description: "desc2", parameters: {} },
                    ],
                },
            };

            const estimate = strategy.estimateResources(toolContext as any);

            expect(estimate.tokens).toBeGreaterThan(strategy.estimateResources(mockExecutionContext as any).tokens);
        });
    });

    describe("Performance Metrics", () => {
        it("should track execution metrics", () => {
            const metrics = strategy.getPerformanceMetrics();

            expect(metrics).toHaveProperty("totalExecutions");
            expect(metrics).toHaveProperty("successCount");
            expect(metrics).toHaveProperty("failureCount");
            expect(metrics).toHaveProperty("averageConfidence");
            expect(metrics).toHaveProperty("averageTokensUsed");
            expect(metrics).toHaveProperty("evolutionScore");
        });

        it("should update metrics after execution", async () => {
            const beforeMetrics = strategy.getPerformanceMetrics();
            
            await strategy.execute(mockExecutionContext as any);
            
            const afterMetrics = strategy.getPerformanceMetrics();

            expect(afterMetrics.totalExecutions).toBe(beforeMetrics.totalExecutions + 1);
            expect(afterMetrics.successCount).toBe(beforeMetrics.successCount + 1);
        });
    });

    // Learning capabilities now happen through event-driven architecture
    // Tests removed: "Learning Capabilities" describe block

    describe("Legacy Pattern Support", () => {
        it("should handle instruction field properly", async () => {
            const instructionContext = {
                ...mockExecutionContext,
                config: {
                    ...mockExecutionContext.config,
                    instructions: "Follow these specific steps...",
                },
            };

            const result = await strategy.execute(instructionContext as any);

            expect(result.success).toBe(true);
            expect(result.outputs.response).toContain("instructions");
        });

        it("should process web routine inputs", async () => {
            const webContext = {
                ...mockExecutionContext,
                stepType: "RoutineWeb",
                inputs: {
                    url: "https://example.com",
                    action: "scrape",
                },
            };

            const result = await strategy.execute(webContext as any);

            expect(result.success).toBe(true);
            expect(result.outputs.response).toBeDefined();
        });
    });
});