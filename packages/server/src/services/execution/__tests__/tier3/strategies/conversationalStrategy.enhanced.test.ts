import { describe, it, expect, beforeEach, afterEach } from "@jest/globals";
import winston from "winston";
import { ConversationalStrategy } from "../conversationalStrategy.js";
import { StrategyType } from "@vrooli/shared";

describe("ConversationalStrategy Enhanced (Legacy Patterns)", () => {
    let strategy: ConversationalStrategy;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility

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
            expect(typeof strategy.learn).toBe("function");
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
            })).toBe(true);
            
            expect(strategy.canHandle("analyze", {
                description: "Collaborative brainstorming session",
            })).toBe(true);
        });

        it("should reject non-conversational tasks", () => {
            expect(strategy.canHandle("calculate_sum", {
                name: "Math Calculation",
                description: "Add two numbers together",
            })).toBe(false);
        });
    });

    describe("Enhanced Resource Estimation", () => {
        it("should provide detailed resource estimates based on complexity", () => {
            const estimate = strategy.estimateResources(mockExecutionContext);

            expect(estimate.tokens).toBeGreaterThan(800);
            expect(estimate.apiCalls).toBeGreaterThanOrEqual(1);
            expect(estimate.computeTime).toBeGreaterThan(0);
            expect(estimate.cost).toBeGreaterThan(0);
        });

        it("should adjust estimates based on context complexity", () => {
            const simpleContext = {
                ...mockExecutionContext,
                inputs: { message: "Hi" },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
            };

            const complexContext = {
                ...mockExecutionContext,
                inputs: {
                    message: "Complex analysis request",
                    data1: { large: "dataset" },
                    data2: { another: "dataset" },
                    config: { multiple: "parameters" },
                },
                history: {
                    recentSteps: Array(10).fill({ stepId: "step", strategy: StrategyType.CONVERSATIONAL, result: "success", duration: 1000 }),
                    totalExecutions: 100,
                    successRate: 0.8,
                },
            };

            const simpleEstimate = strategy.estimateResources(simpleContext);
            const complexEstimate = strategy.estimateResources(complexContext);

            expect(complexEstimate.tokens).toBeGreaterThan(simpleEstimate.tokens);
            expect(complexEstimate.computeTime).toBeGreaterThanOrEqual(simpleEstimate.computeTime);
        });

        it("should estimate additional API calls for tool usage", () => {
            const contextWithTools = {
                ...mockExecutionContext,
                stepType: "analyze_data",
                config: {
                    ...mockExecutionContext.config,
                    description: "Analyze customer data with search tools",
                },
            };

            const estimate = strategy.estimateResources(contextWithTools);
            expect(estimate.apiCalls).toBe(2); // 1 for conversation + 1 for tools
        });
    });

    describe("Legacy System Message Building", () => {
        it("should build comprehensive system messages", async () => {
            const systemMessage = await (strategy as any).buildSystemMessage(mockExecutionContext);

            expect(systemMessage).toContain("Data Analysis Chat");
            expect(systemMessage).toContain("Help user analyze customer data");
            expect(systemMessage).toContain("Provide detailed analysis and recommendations");
            expect(systemMessage).toContain("Conversation Guidelines:");
            expect(systemMessage).toContain("Expected conversation outcomes:");
            expect(systemMessage).toContain("Analysis Results");
            expect(systemMessage).toContain("Recommendations");
        });

        it("should handle missing configuration gracefully", async () => {
            const minimalContext = {
                ...mockExecutionContext,
                config: {},
            };

            const systemMessage = await (strategy as any).buildSystemMessage(minimalContext);

            expect(systemMessage).toContain("Conversational Task");
            expect(systemMessage).toContain("Conversation Guidelines:");
        });
    });

    describe("Legacy Message History Building", () => {
        it("should build message history from inputs and context", () => {
            const messages = (strategy as any).buildMessageHistory(mockExecutionContext);

            expect(messages).toHaveLength(2); // Initial message + 1 from history
            expect(messages[0].role).toBe("user");
            expect(messages[0].content).toContain("Hello, I need help with data analysis");
            expect(messages[1].content).toContain("Previous step step-122");
        });

        it("should handle empty history gracefully", () => {
            const contextWithoutHistory = {
                ...mockExecutionContext,
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
            };

            const messages = (strategy as any).buildMessageHistory(contextWithoutHistory);
            expect(messages).toHaveLength(1); // Only initial message
        });

        it("should create message from multiple inputs", () => {
            const contextWithoutDirectMessage = {
                ...mockExecutionContext,
                inputs: {
                    data: { value: 42 },
                    type: "analysis",
                    priority: "high",
                },
            };

            const messages = (strategy as any).buildMessageHistory(contextWithoutDirectMessage);
            expect(messages[0].content).toContain("data");
            expect(messages[0].content).toContain("type");
            expect(messages[0].content).toContain("priority");
        });
    });

    describe("Performance Tracking", () => {
        it("should track performance metrics over time", () => {
            // Simulate some executions
            (strategy as any).trackPerformance({
                timestamp: new Date(),
                executionTime: 5000,
                tokensUsed: 800,
                success: true,
                confidence: 0.9,
            });

            (strategy as any).trackPerformance({
                timestamp: new Date(),
                executionTime: 3000,
                tokensUsed: 600,
                success: true,
                confidence: 0.8,
            });

            const metrics = strategy.getPerformanceMetrics();

            expect(metrics.totalExecutions).toBe(2);
            expect(metrics.successCount).toBe(2);
            expect(metrics.failureCount).toBe(0);
            expect(metrics.averageExecutionTime).toBe(4000);
            expect(metrics.averageConfidence).toBe(0.85);
        });

        it("should calculate evolution score based on performance trends", () => {
            const strategy = new ConversationalStrategy(logger);

            // Simulate improving performance over time
            for (let i = 0; i < 60; i++) {
                (strategy as any).trackPerformance({
                    timestamp: new Date(Date.now() - (60 - i) * 1000),
                    executionTime: 5000 - i * 10,
                    tokensUsed: 800,
                    success: true,
                    confidence: 0.5 + (i * 0.008), // Gradually improving confidence
                });
            }

            const metrics = strategy.getPerformanceMetrics();
            expect(metrics.evolutionScore).toBeGreaterThan(0.5); // Should show improvement
        });
    });

    describe("Learning Mechanism", () => {
        it("should learn from successful feedback", () => {
            const logSpy = vi.spyOn(logger, "debug");

            const feedback = {
                outcome: "success" as const,
                performanceScore: 0.9,
                userSatisfaction: 0.95,
            };

            strategy.learn(feedback);

            expect(logSpy.calledWith("[ConversationalStrategy] Optimizing for success")).toBe(true);
        });

        it("should learn from failure feedback", () => {
            const logSpy = vi.spyOn(logger, "debug");

            const feedback = {
                outcome: "failure" as const,
                performanceScore: 0.2,
                issues: ["Response was unclear", "Did not address the question"],
            };

            strategy.learn(feedback);

            expect(logSpy.calledWith("[ConversationalStrategy] Adjusting for failure")).toBe(true);
        });
    });

    describe("Enhanced Confidence Calculation", () => {
        it("should adjust confidence based on response quality", () => {
            const highQualityResponse = {
                content: "Based on your customer data analysis, I can provide comprehensive insights. The dataset shows clear patterns in customer behavior.",
                reasoning: "Analyzed the provided data systematically",
                confidence: 0.9,
                tokensUsed: 150,
                toolCalls: [],
            };

            const confidence = (strategy as any).calculateConfidence(highQualityResponse, mockExecutionContext);
            expect(confidence).toBe(0.9); // Should maintain high confidence

            const lowQualityResponse = {
                content: "I don't know much about this data.",
                reasoning: "Unclear request",
                confidence: 0.7,
                tokensUsed: 50,
                toolCalls: [],
            };

            const lowConfidence = (strategy as any).calculateConfidence(lowQualityResponse, mockExecutionContext);
            expect(lowConfidence).toBeLessThan(0.7); // Should be penalized
        });

        it("should give bonus for tool usage", () => {
            const responseWithTools = {
                content: "I analyzed the data using the data_analyzer tool.",
                reasoning: "Used tools for analysis",
                confidence: 0.8,
                tokensUsed: 200,
                toolCalls: [{ id: "1", name: "data_analyzer", parameters: {} }],
            };

            const confidence = (strategy as any).calculateConfidence(responseWithTools, mockExecutionContext);
            expect(confidence).toBe(0.88); // Should get 10% bonus, capped at 1.0
        });
    });

    describe("Enhanced Performance Scoring", () => {
        it("should score based on multiple quality factors", () => {
            const excellentResponse = {
                content: "Here's a comprehensive analysis:\n• Key findings from your data\n• Specific recommendations\n• Next steps to consider",
                reasoning: "Structured analytical approach",
                confidence: 0.9,
                tokensUsed: 800,
                toolCalls: [{ id: "1", name: "analyzer", parameters: {} }],
            };

            const score = (strategy as any).calculatePerformanceScore(excellentResponse, mockExecutionContext);
            expect(score).toBeGreaterThan(0.9); // Should score highly

            const poorResponse = {
                content: "Not sure what you want.",
                reasoning: "Unclear request",
                confidence: 0.3,
                tokensUsed: 50,
                toolCalls: [],
            };

            const poorScore = (strategy as any).calculatePerformanceScore(poorResponse, mockExecutionContext);
            expect(poorScore).toBeLessThan(0.5); // Should score poorly
        });
    });

    describe("Enhanced Improvement Suggestions", () => {
        it("should provide specific improvement suggestions", () => {
            const poorResponse = {
                content: "I'm not sure about that.",
                reasoning: "Unclear",
                confidence: 0.4,
                tokensUsed: 3000,
                toolCalls: [],
            };

            const improvements = (strategy as any).suggestImprovements(poorResponse, mockExecutionContext);
            
            expect(improvements).toContain("Consider providing more specific context or examples to improve response quality");
            expect(improvements).toContain("Response was lengthy - consider more focused prompting or output constraints");
            expect(improvements).toContain("Response contained uncertainty - consider providing clearer instructions or context");
        });

        it("should suggest tool usage when appropriate", () => {
            const contextNeedingTools = {
                ...mockExecutionContext,
                stepType: "search_data",
                config: {
                    ...mockExecutionContext.config,
                    description: "Search through customer database",
                },
            };

            const responseWithoutTools = {
                content: "Here's some general information.",
                reasoning: "Manual response",
                confidence: 0.7,
                tokensUsed: 200,
                toolCalls: [],
            };

            const improvements = (strategy as any).suggestImprovements(responseWithoutTools, contextNeedingTools);
            expect(improvements).toContain("Task might benefit from tool usage - consider enabling relevant tools");
        });
    });

    describe("Error Handling (Legacy Patterns)", () => {
        it("should identify retryable errors", () => {
            const retryableError = new Error("Request timeout occurred");
            const isRetryable = (strategy as any).isRetryableError(retryableError);
            expect(isRetryable).toBe(true);

            const nonRetryableError = new Error("Invalid API key");
            const isNonRetryable = (strategy as any).isRetryableError(nonRetryableError);
            expect(isNonRetryable).toBe(false);
        });

        it("should handle conversational errors gracefully", () => {
            const error = new Error("Rate limit exceeded");
            const mockRequest = {
                model: "gpt-4o-mini",
                messages: [{ role: "user" as const, content: "Hello" }],
            };

            const response = (strategy as any).handleConversationalError(error, mockExecutionContext, mockRequest);
            
            expect(response.content).toContain("temporary issue");
            expect(response.confidence).toBe(0.4);
            expect(response.reasoning).toContain("Retryable error");
        });
    });

    describe("Output Extraction (Legacy Patterns)", () => {
        it("should extract single output correctly", async () => {
            const contextWithSingleOutput = {
                ...mockExecutionContext,
                config: {
                    ...mockExecutionContext.config,
                    expectedOutputs: {
                        result: { name: "Result", description: "The main result" },
                    },
                },
            };

            const response = {
                content: "This is the main analysis result with detailed findings.",
                reasoning: "Comprehensive analysis performed",
                confidence: 0.8,
                tokensUsed: 200,
                toolCalls: [],
            };

            const outputs = await (strategy as any).extractOutputsFromConversation(response, contextWithSingleOutput);
            
            expect(outputs.result).toBe("This is the main analysis result with detailed findings.");
            expect(outputs.reasoning).toBe("Comprehensive analysis performed");
            expect(outputs.confidence).toBe(0.8);
        });

        it("should extract multiple structured outputs", async () => {
            const response = {
                content: "Analysis: The data shows positive trends.\\n\\nRecommendations: Increase marketing spend in Q2.",
                reasoning: "Structured analysis",
                confidence: 0.9,
                tokensUsed: 250,
                toolCalls: [],
            };

            const outputs = await (strategy as any).extractOutputsFromConversation(response, mockExecutionContext);
            
            expect(outputs.analysis).toBe("The data shows positive trends.");
            expect(outputs.recommendations).toBe("Increase marketing spend in Q2.");
        });

        it("should fallback to full response when no structure found", async () => {
            const response = {
                content: "This is an unstructured response without clear sections.",
                reasoning: "General response",
                confidence: 0.7,
                tokensUsed: 180,
                toolCalls: [],
            };

            const outputs = await (strategy as any).extractOutputsFromConversation(response, mockExecutionContext);
            
            expect(outputs.result).toBe("This is an unstructured response without clear sections.");
        });
    });
});