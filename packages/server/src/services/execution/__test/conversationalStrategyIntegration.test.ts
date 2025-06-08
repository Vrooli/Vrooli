import { describe, it, beforeEach, afterEach, vi, expect } from "vitest";
import winston from "winston";
import { ConversationalStrategy } from "../tier3/strategies/conversationalStrategy.js";
import { ConversationalStrategyAdapter } from "../tier3/strategies/adapters/conversationalAdapter.js";
import { StrategyType } from "@vrooli/shared";

describe("ConversationalStrategy Integration", () => {
    let enhancedStrategy: ConversationalStrategy;
    let adapterStrategy: ConversationalStrategyAdapter;
    let logger: winston.Logger;
    // Sandbox type removed for Vitest compatibility

    const mockExecutionContext = {
        stepId: "integration-test-123",
        stepType: "chat_response",
        inputs: {
            message: "Help me understand machine learning concepts",
            context: "I'm a beginner in data science",
        },
        config: {
            name: "ML Education Chat",
            description: "Educational conversation about machine learning",
            model: "gpt-4o-mini",
            temperature: 0.7,
            expectedOutputs: {
                explanation: { name: "Explanation", description: "Clear explanation of concepts" },
                nextSteps: { name: "Next Steps", description: "Recommended learning path" },
            },
        },
        resources: {
            credits: 10000,
            tools: [],
        },
        history: {
            recentSteps: [],
            totalExecutions: 5,
            successRate: 1.0,
        },
        constraints: {
            maxTokens: 1500,
            maxTime: 30000,
            requiredConfidence: 0.7,
        },
        metadata: {
            userId: "integration-user-123",
        },
    };

    beforeEach(() => {
        
        logger = winston.createLogger({
            level: "error",
            transports: [new winston.transports.Console()],
        });
        
        enhancedStrategy = new ConversationalStrategy(logger);
        adapterStrategy = new ConversationalStrategyAdapter(logger);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("Enhanced Strategy vs Adapter Comparison", () => {
        it("should have consistent interfaces", () => {
            // Both should implement the same interface
            expect(enhancedStrategy.type).toBe(StrategyType.CONVERSATIONAL);
            expect(adapterStrategy.type).toBe(StrategyType.CONVERSATIONAL);
            
            expect(enhancedStrategy.name).toBe("ConversationalStrategy");
            expect(adapterStrategy.name).toBe("ConversationalStrategy");
            
            // Enhanced should have newer version
            expect(enhancedStrategy.version).toBe("2.0.0-enhanced");
            expect(adapterStrategy.version).toBe("1.0.0-adapter");
        });

        it("should handle the same step types", () => {
            const testCases = [
                "chat_response",
                "discuss_topic", 
                "brainstorm_ideas",
                "creative_writing",
            ];

            testCases.forEach(stepType => {
                expect(enhancedStrategy.canHandle(stepType)).toBe(adapterStrategy.canHandle(stepType));
            });
        });

        it("should provide resource estimates in same ballpark", () => {
            const enhancedEstimate = enhancedStrategy.estimateResources(mockExecutionContext);
            const adapterEstimate = adapterStrategy.estimateResources(mockExecutionContext);

            // Both should provide reasonable estimates
            expect(enhancedEstimate.tokens).toBeGreaterThan(500);
            expect(adapterEstimate.tokens).toBeGreaterThan(500);
            
            expect(enhancedEstimate.apiCalls).toBeGreaterThanOrEqual(1);
            expect(adapterEstimate.apiCalls).toBeGreaterThanOrEqual(1);
            
            expect(enhancedEstimate.cost).toBeGreaterThan(0);
            expect(adapterEstimate.cost).toBeGreaterThan(0);
        });
    });

    describe("Enhanced Strategy Features", () => {
        it("should provide more sophisticated resource estimation", () => {
            const simpleContext = {
                ...mockExecutionContext,
                inputs: { message: "Hi" },
            };

            const complexContext = {
                ...mockExecutionContext,
                inputs: {
                    message: "Complex analysis request",
                    data: { large: "dataset" },
                    requirements: { detailed: "specifications" },
                },
                history: {
                    recentSteps: Array(5).fill({ stepId: "step", strategy: StrategyType.CONVERSATIONAL, result: "success", duration: 1000 }),
                    totalExecutions: 50,
                    successRate: 0.85,
                },
            };

            const simpleEstimate = enhancedStrategy.estimateResources(simpleContext);
            const complexEstimate = enhancedStrategy.estimateResources(complexContext);

            // Enhanced strategy should show more nuanced estimation
            expect(complexEstimate.tokens).toBeGreaterThan(simpleEstimate.tokens);
            expect(complexEstimate.computeTime).toBeGreaterThanOrEqual(simpleEstimate.computeTime);
        });

        it("should track performance metrics over time", () => {
            // Simulate performance tracking
            const strategy = new ConversationalStrategy(logger);
            
            // Initial metrics should be empty
            const initialMetrics = strategy.getPerformanceMetrics();
            expect(initialMetrics.totalExecutions).toBe(0);
            
            // Simulate executions via private method
            (strategy as any).trackPerformance({
                timestamp: new Date(),
                executionTime: 5000,
                tokensUsed: 800,
                success: true,
                confidence: 0.85,
            });
            
            (strategy as any).trackPerformance({
                timestamp: new Date(),
                executionTime: 3000,
                tokensUsed: 600,
                success: true,
                confidence: 0.9,
            });
            
            const updatedMetrics = strategy.getPerformanceMetrics();
            expect(updatedMetrics.totalExecutions).toBe(2);
            expect(updatedMetrics.successCount).toBe(2);
            expect(updatedMetrics.averageExecutionTime).toBe(4000);
            expect(updatedMetrics.averageConfidence).toBe(0.875);
        });

        it("should provide learning capabilities", () => {
            const logSpy = vi.spyOn(logger, "info");
            
            const successFeedback = {
                outcome: "success" as const,
                performanceScore: 0.9,
                userSatisfaction: 0.95,
            };
            
            enhancedStrategy.learn(successFeedback);
            
            expect(logSpy.calledWithMatch("[ConversationalStrategy] Learning from feedback")).toBe(true);
        });
    });

    describe("Legacy Pattern Integration", () => {
        it("should build comprehensive system messages", async () => {
            const systemMessage = await (enhancedStrategy as any).buildSystemMessage(mockExecutionContext);
            
            // Should include all legacy patterns
            expect(systemMessage).toContain("ML Education Chat");
            expect(systemMessage).toContain("Educational conversation about machine learning");
            expect(systemMessage).toContain("Conversation Guidelines:");
            expect(systemMessage).toContain("Engage naturally and helpfully");
            expect(systemMessage).toContain("Expected conversation outcomes:");
            expect(systemMessage).toContain("Explanation");
            expect(systemMessage).toContain("Next Steps");
        });

        it("should build message history from context", () => {
            const contextWithHistory = {
                ...mockExecutionContext,
                history: {
                    recentSteps: [
                        { stepId: "step-1", strategy: StrategyType.DETERMINISTIC, result: "success", duration: 2000 },
                        { stepId: "step-2", strategy: StrategyType.CONVERSATIONAL, result: "success", duration: 5000 },
                    ],
                    totalExecutions: 10,
                    successRate: 0.9,
                },
            };
            
            const messages = (enhancedStrategy as any).buildMessageHistory(contextWithHistory);
            
            expect(messages).toHaveLength(3); // Initial + 2 from history
            expect(messages[0].content).toContain("Help me understand machine learning");
            expect(messages[1].content).toContain("Previous step step-1");
            expect(messages[2].content).toContain("Previous step step-2");
        });

        it("should extract outputs using legacy patterns", async () => {
            const multiOutputResponse = {
                content: "Explanation: Machine learning is a subset of AI that uses algorithms to learn from data.\\n\\nNext Steps: Start with basic statistics and Python programming.",
                reasoning: "Provided structured educational response",
                confidence: 0.9,
                tokensUsed: 250,
                toolCalls: [],
            };
            
            const outputs = await (enhancedStrategy as any).extractOutputsFromConversation(
                multiOutputResponse, 
                mockExecutionContext
            );
            
            expect(outputs.explanation).toBe("Machine learning is a subset of AI that uses algorithms to learn from data.");
            expect(outputs.nextSteps).toBe("Start with basic statistics and Python programming.");
            expect(outputs.reasoning).toBe("Provided structured educational response");
            expect(outputs.confidence).toBe(0.9);
        });

        it("should assess conversation complexity accurately", () => {
            const simpleContext = {
                ...mockExecutionContext,
                inputs: { message: "Hi" },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1 },
                constraints: {},
            };
            
            const complexContext = {
                ...mockExecutionContext,
                inputs: {
                    message: "Complex request",
                    data: { multiple: "values" },
                    config: { advanced: "settings" },
                },
                history: {
                    recentSteps: Array(10).fill({ stepId: "step", strategy: StrategyType.CONVERSATIONAL, result: "success", duration: 1000 }),
                    totalExecutions: 100,
                    successRate: 0.8,
                },
                constraints: {
                    requiredConfidence: 0.9,
                },
            };
            
            const simpleComplexity = (enhancedStrategy as any).assessConversationComplexity(simpleContext);
            const complexComplexity = (enhancedStrategy as any).assessConversationComplexity(complexContext);
            
            expect(complexComplexity).toBeGreaterThan(simpleComplexity);
            expect(simpleComplexity).toBeGreaterThanOrEqual(0.5); // Base complexity
            expect(complexComplexity).toBeLessThanOrEqual(1.0); // Max complexity
        });
    });

    describe("Error Handling Integration", () => {
        it("should handle retryable errors gracefully", () => {
            const retryableError = new Error("Request timeout - please retry");
            const mockRequest = {
                model: "gpt-4o-mini",
                messages: [{ role: "user" as const, content: "Test message" }],
            };
            
            const response = (enhancedStrategy as any).handleConversationalError(
                retryableError, 
                mockExecutionContext, 
                mockRequest
            );
            
            expect(response.content).toContain("temporary issue");
            expect(response.confidence).toBe(0.4);
            expect(response.reasoning).toContain("Retryable error");
        });

        it("should identify retryable vs non-retryable errors", () => {
            const retryableErrors = [
                new Error("Request timeout occurred"),
                new Error("Rate limit exceeded"),
                new Error("Temporary network issue"),
            ];
            
            const nonRetryableErrors = [
                new Error("Invalid API key"),
                new Error("Malformed request"),
                new Error("Authentication failed"),
            ];
            
            retryableErrors.forEach(error => {
                expect((enhancedStrategy as any).isRetryableError(error)).toBe(true);
            });
            
            nonRetryableErrors.forEach(error => {
                expect((enhancedStrategy as any).isRetryableError(error)).toBe(false);
            });
        });
    });

    describe("Performance and Quality Metrics", () => {
        it("should calculate confidence with quality adjustments", () => {
            const highQualityResponse = {
                content: "Based on your question about machine learning, here are the key concepts: \\n• Supervised learning\\n• Unsupervised learning\\n• Deep learning\\nEach has specific applications...",
                reasoning: "Comprehensive structured response",
                confidence: 0.85,
                tokensUsed: 200,
                toolCalls: [],
            };
            
            const lowQualityResponse = {
                content: "Not sure.",
                reasoning: "Unclear request",
                confidence: 0.6,
                tokensUsed: 20,
                toolCalls: [],
            };
            
            const highConfidence = (enhancedStrategy as any).calculateConfidence(highQualityResponse, mockExecutionContext);
            const lowConfidence = (enhancedStrategy as any).calculateConfidence(lowQualityResponse, mockExecutionContext);
            
            expect(highConfidence).toBeGreaterThan(lowConfidence);
            expect(highConfidence).toBeGreaterThanOrEqual(0.85); // Should maintain or increase
            expect(lowConfidence).toBeLessThan(0.6); // Should be penalized
        });

        it("should provide detailed improvement suggestions", () => {
            const poorResponse = {
                content: "I don't know much about this unclear topic.",
                reasoning: "Insufficient information",
                confidence: 0.3,
                tokensUsed: 3500,
                toolCalls: [],
            };
            
            const improvements = (enhancedStrategy as any).suggestImprovements(poorResponse, mockExecutionContext);
            
            expect(improvements.length).toBeGreaterThan(0);
            expect(improvements).toContain("Consider providing more specific context or examples to improve response quality");
            expect(improvements).toContain("Response was lengthy - consider more focused prompting or output constraints");
            expect(improvements).toContain("Response contained uncertainty - consider providing clearer instructions or context");
        });
    });
});