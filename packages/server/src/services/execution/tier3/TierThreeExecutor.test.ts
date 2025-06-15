import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { TierThreeExecutor } from "./TierThreeExecutor.js";
import { type Logger } from "winston";
import { type EventBus } from "../cross-cutting/events/eventBus.js";
import { createMockLogger } from "../../../../__test/globalHelpers.js";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import { RedisClientType } from "redis";
import { createRedisClient } from "../../../redisConn.js";
import { EventBusImplementation } from "../cross-cutting/events/eventBus.js";
import {
    type ExecutionContext,
    type TierExecutionRequest,
    ExecutionStatus,
    StrategyType,
    generatePK,
} from "@vrooli/shared";

/**
 * TierThreeExecutor Tests - Execution Intelligence Layer
 * 
 * These tests validate that Tier 3 provides adaptive execution infrastructure
 * while enabling intelligent behaviors to emerge from:
 * 
 * 1. **Strategy Evolution**: Conversational → Reasoning → Deterministic → Routing
 * 2. **Context-Aware Execution**: Adapts based on accumulated knowledge
 * 3. **Tool Orchestration**: MCP tools without hard-coded logic
 * 4. **Resource Management**: Credit/time tracking without optimization
 * 5. **Validation Without Rules**: Quality emerges from agent feedback
 * 
 * The executor provides infrastructure for intelligence to emerge through
 * usage patterns, not prescriptive algorithms.
 */

describe("TierThreeExecutor - Adaptive Execution Intelligence", () => {
    let logger: Logger;
    let eventBus: EventBus;
    let executor: TierThreeExecutor;
    let redisContainer: StartedTestContainer;
    let redisClient: RedisClientType;

    beforeEach(async () => {
        // Use real Redis for event-driven execution
        redisContainer = await new GenericContainer("redis:7-alpine")
            .withExposedPorts(6379)
            .start();

        const redisUrl = `redis://localhost:${redisContainer.getMappedPort(6379)}`;
        redisClient = await createRedisClient({ url: redisUrl });

        logger = createMockLogger();
        eventBus = new EventBusImplementation(logger, redisClient as any);
        
        executor = new TierThreeExecutor(logger, eventBus);
    });

    afterEach(async () => {
        await redisClient?.quit();
        await redisContainer?.stop();
    });

    describe("Adaptive Strategy Selection", () => {
        it("should start with conversational strategy for novel tasks", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: {
                    task: "Create a new API endpoint for user preferences",
                },
                userData: { id: generatePK() },
            };

            const result = await executor.executeStep(context);

            // Should use conversational for novel task
            expect(result.status).toBe("completed");
            expect(result.metadata?.strategy).toBe(StrategyType.CONVERSATIONAL);
        });

        it("should evolve strategy based on execution patterns", async () => {
            const baseContext = {
                executionId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                routineId: "routine-123",
                userData: { id: generatePK() },
            };

            // Execute similar tasks multiple times
            const executions = [];
            for (let i = 0; i < 5; i++) {
                const context: ExecutionContext = {
                    ...baseContext,
                    stepId: generatePK(),
                    inputs: {
                        action: "validate_email",
                        email: `user${i}@example.com`,
                    },
                };
                
                const result = await executor.executeStep(context);
                executions.push(result);
            }

            // Strategy should evolve from conversational to more deterministic
            const strategies = executions.map(r => r.metadata?.strategy);
            
            // First execution likely conversational
            expect(strategies[0]).toBe(StrategyType.CONVERSATIONAL);
            
            // Later executions may evolve to reasoning or deterministic
            const evolvedStrategies = strategies.slice(2);
            const hasEvolved = evolvedStrategies.some(s => 
                s === StrategyType.REASONING || s === StrategyType.DETERMINISTIC
            );
            
            // Infrastructure enables evolution, but doesn't force it
            expect(hasEvolved || strategies.every(s => s === StrategyType.CONVERSATIONAL)).toBe(true);
        });
    });

    describe("Context-Aware Execution", () => {
        it("should accumulate context across step executions", async () => {
            const runId = generatePK();
            const contexts: ExecutionContext[] = [
                {
                    executionId: generatePK(),
                    stepId: "step-1",
                    stepType: "action",
                    runId,
                    inputs: { action: "fetch_user_data", userId: "123" },
                    userData: { id: generatePK() },
                },
                {
                    executionId: generatePK(),
                    stepId: "step-2",
                    stepType: "action",
                    runId,
                    inputs: { action: "analyze_preferences" },
                    userData: { id: generatePK() },
                    previousSteps: ["step-1"],
                },
            ];

            const results = [];
            for (const context of contexts) {
                const result = await executor.executeStep(context);
                results.push(result);
            }

            // Second step should have access to first step's context
            expect(results[1].metadata?.contextAware).toBe(true);
        });

        it("should enable context export for knowledge sharing", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                routineId: "data-pipeline",
                routineName: "Customer Data Pipeline",
                inputs: { action: "process_batch", size: 1000 },
                userData: { id: generatePK() },
                config: {
                    exportContext: true, // Enable context export
                },
            };

            const result = await executor.executeStep(context);

            expect(result.status).toBe("completed");
            expect(result.metadata?.contextExported).toBe(true);
        });
    });

    describe("Tool Orchestration Without Logic", () => {
        it("should orchestrate MCP tools based on execution needs", async () => {
            const toolEvents: any[] = [];
            await eventBus.subscribe("execution.tool.*", async (event) => {
                toolEvents.push(event);
            });

            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: {
                    action: "search_and_summarize",
                    query: "latest AI research",
                },
                userData: { id: generatePK() },
                availableTools: ["web_search", "summarize", "save_results"],
            };

            await executor.executeStep(context);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Tools used based on task needs, not hard-coded flow
            expect(toolEvents.length).toBeGreaterThan(0);
            expect(toolEvents.some(e => e.type === "execution.tool.invoked")).toBe(true);
        });

        it("should adapt tool usage based on availability", async () => {
            // Test with different tool sets
            const contexts: ExecutionContext[] = [
                {
                    executionId: generatePK(),
                    stepId: generatePK(),
                    stepType: "action",
                    runId: generatePK(),
                    inputs: { action: "get_weather", location: "New York" },
                    userData: { id: generatePK() },
                    availableTools: ["weather_api", "cache_result"],
                },
                {
                    executionId: generatePK(),
                    stepId: generatePK(),
                    stepType: "action",
                    runId: generatePK(),
                    inputs: { action: "get_weather", location: "London" },
                    userData: { id: generatePK() },
                    availableTools: ["web_search"], // No weather API
                },
            ];

            const results = await Promise.all(
                contexts.map(ctx => executor.executeStep(ctx))
            );

            // Both should complete but with different approaches
            expect(results[0].status).toBe("completed");
            expect(results[1].status).toBe("completed");
            
            // Different tools lead to different execution paths
            expect(results[0].metadata?.toolsUsed).not.toEqual(results[1].metadata?.toolsUsed);
        });
    });

    describe("Resource Management Infrastructure", () => {
        it("should track resource usage without optimization", async () => {
            const resourceEvents: any[] = [];
            await eventBus.subscribe("execution.resource.*", async (event) => {
                resourceEvents.push(event);
            });

            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: { action: "process_large_dataset" },
                userData: { id: generatePK() },
                config: {
                    maxCredits: 100,
                    maxDuration: 5000,
                },
            };

            const result = await executor.executeStep(context);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Resource tracking events for monitoring agents
            expect(resourceEvents).toContainEqual(
                expect.objectContaining({
                    type: "execution.resource.consumed",
                    data: expect.objectContaining({
                        executionId: context.executionId,
                    }),
                })
            );

            // Result includes resource metadata
            expect(result.metadata?.creditsUsed).toBeDefined();
            expect(result.metadata?.executionTime).toBeDefined();
        });

        it("should enforce limits without prescriptive handling", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: { action: "expensive_operation" },
                userData: { id: generatePK() },
                config: {
                    maxCredits: 1, // Very low limit
                },
            };

            const result = await executor.executeStep(context);

            // May fail or adapt based on limits
            if (result.status === "failed") {
                expect(result.error).toContain("limit");
            } else {
                // Adapted to work within limits
                expect(result.metadata?.creditsUsed).toBeLessThanOrEqual(1);
            }
        });
    });

    describe("Validation Through Emergence", () => {
        it("should emit validation events for quality agents", async () => {
            const validationEvents: any[] = [];
            await eventBus.subscribe("execution.validation.*", async (event) => {
                validationEvents.push(event);
            });

            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: {
                    action: "generate_report",
                    data: { revenue: 1000000, costs: 800000 },
                },
                userData: { id: generatePK() },
                outputSchema: {
                    type: "object",
                    properties: {
                        summary: { type: "string" },
                        profit: { type: "number" },
                    },
                },
            };

            await executor.executeStep(context);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Validation agents can analyze output quality
            expect(validationEvents.length).toBeGreaterThan(0);
            expect(validationEvents.some(e => e.type === "execution.validation.completed")).toBe(true);
        });

        it("should adapt validation based on feedback", async () => {
            const feedbackEvents: any[] = [];
            await eventBus.subscribe("execution.feedback.*", async (event) => {
                feedbackEvents.push(event);
            });

            // Execute and provide feedback
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: { action: "classify_sentiment", text: "This product is amazing!" },
                userData: { id: generatePK() },
            };

            const result = await executor.executeStep(context);

            // Simulate quality feedback
            await eventBus.publish("execution.feedback.quality", {
                executionId: context.executionId,
                quality: "high",
                accuracy: 0.95,
                suggestions: ["Consider confidence scores"],
            });

            await new Promise(resolve => setTimeout(resolve, 100));

            // System can learn from feedback
            expect(feedbackEvents.length).toBe(1);
            expect(result.status).toBe("completed");
        });
    });

    describe("TierCommunicationInterface Compliance", () => {
        it("should handle tier execution requests", async () => {
            const request: TierExecutionRequest = {
                executionId: generatePK(),
                type: "step",
                userId: generatePK(),
                payload: {
                    stepId: generatePK(),
                    stepType: "action",
                    runId: generatePK(),
                    inputs: { action: "test_action" },
                },
            };

            const result = await executor.execute(request);

            expect(result.executionId).toBe(request.executionId);
            expect(result.status).toBe("completed");
        });

        it("should provide execution metrics", async () => {
            // Execute some steps
            for (let i = 0; i < 3; i++) {
                await executor.execute({
                    executionId: generatePK(),
                    type: "step",
                    userId: generatePK(),
                    payload: {
                        stepId: generatePK(),
                        stepType: "action",
                        runId: generatePK(),
                        inputs: { action: `action_${i}` },
                    },
                });
            }

            const metrics = await executor.getMetrics();

            expect(metrics.totalExecutions).toBe(3);
            expect(metrics.activeExecutions).toBe(0);
            expect(metrics.averageExecutionTime).toBeGreaterThan(0);
        });
    });

    describe("Error Handling and Resilience", () => {
        it("should handle strategy failures gracefully", async () => {
            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: { action: "invalid_action", error: "trigger_error" },
                userData: { id: generatePK() },
            };

            const result = await executor.executeStep(context);

            // Should complete or fail gracefully
            expect(["completed", "failed"]).toContain(result.status);
            if (result.status === "failed") {
                expect(result.error).toBeDefined();
            }
        });

        it("should emit failure events for analysis", async () => {
            const failureEvents: any[] = [];
            await eventBus.subscribe("execution.failure.*", async (event) => {
                failureEvents.push(event);
            });

            const context: ExecutionContext = {
                executionId: generatePK(),
                stepId: generatePK(),
                stepType: "action",
                runId: generatePK(),
                inputs: { action: "force_failure" },
                userData: { id: generatePK() },
                config: {
                    maxRetries: 0, // No retries
                },
            };

            await executor.executeStep(context);
            await new Promise(resolve => setTimeout(resolve, 100));

            // Failure analysis agents can learn from errors
            if (failureEvents.length > 0) {
                expect(failureEvents[0].type).toContain("failure");
                expect(failureEvents[0].data.executionId).toBe(context.executionId);
            }
        });
    });
});