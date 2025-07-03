/**
 * Tests for the simplified StepExecutor
 */

import { describe, expect, it } from "vitest";
import { StepExecutor } from "./stepExecutor.js";

describe("StepExecutor", () => {
    let executor: StepExecutor;

    beforeEach(() => {
        executor = new StepExecutor();
    });

    describe("Basic functionality", () => {
        it("should create instance", () => {
            expect(executor).toBeDefined();
        });

        it("should report capabilities", async () => {
            const capabilities = await executor.getCapabilities();
            expect(capabilities).toEqual({
                tier: "tier3",
                capabilities: [
                    "llm_execution",
                    "tool_calling",
                    "code_execution",
                    "api_calls",
                ],
                strategies: ["conversational", "reasoning", "deterministic"],
                ready: true,
            });
        });

        it("should report healthy status", async () => {
            const status = await executor.getStatus();
            expect(status).toEqual({
                healthy: true,
                tier: "tier3",
                activeExecutions: 0,
            });
        });
    });

    describe("Step execution", () => {
        it("should execute LLM call", async () => {
            const request = {
                context: {
                    swarmId: "test-swarm",
                    userData: { id: "user-123" },
                },
                input: {
                    stepId: "step-1",
                    messages: [{ role: "user", content: "Hello" }],
                },
                allocation: {},
            };

            const result = await executor.execute(request);
            expect(result.success).toBe(true);
            expect(result.outputs).toBeDefined();
            expect(result.metadata?.executionTime).toBeGreaterThan(0);
        });

        it("should execute tool call", async () => {
            const request = {
                context: {
                    swarmId: "test-swarm",
                    userData: { id: "user-123" },
                },
                input: {
                    stepId: "step-2",
                    tool: "calculator",
                    arguments: {
                        operation: "add",
                        a: 5,
                        b: 3,
                    },
                },
                allocation: {},
            };

            const result = await executor.execute(request);
            expect(result.success).toBe(true);
            expect(result.outputs).toEqual({
                value: 8,
                expression: "5 add 3 = 8",
            });
        });

        it("should handle execution errors gracefully", async () => {
            const request = {
                context: {
                    swarmId: "test-swarm",
                    userData: { id: "user-123" },
                },
                input: {
                    stepId: "step-3",
                    tool: "non_existent_tool",
                },
                allocation: {},
            };

            const result = await executor.execute(request);
            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();
            expect(result.error?.message).toContain("Tool not found");
        });
    });

    describe("Stateless behavior", () => {
        it("should always report 0 active executions", async () => {
            // Execute multiple requests
            const promises = Array(5).fill(null).map((_, i) => 
                executor.execute({
                    context: {
                        swarmId: "test-swarm",
                        userData: { id: "user-123" },
                    },
                    input: {
                        stepId: `step-${i}`,
                        messages: [{ role: "user", content: `Message ${i}` }],
                    },
                    allocation: {},
                }),
            );

            // Check status while executions are in progress
            const status = await executor.getStatus();
            expect(status.activeExecutions).toBe(0); // Always 0 - stateless!

            // Wait for all to complete
            await Promise.all(promises);

            // Check status after completion
            const finalStatus = await executor.getStatus();
            expect(finalStatus.activeExecutions).toBe(0); // Still 0 - stateless!
        });
    });
});
