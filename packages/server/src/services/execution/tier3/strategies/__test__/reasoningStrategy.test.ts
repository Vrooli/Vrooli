import { describe, it, vi, expect } from "vitest";
import { createStubInstance } from "sinon";
import { type Logger } from "winston";
import { StrategyType } from "@vrooli/shared";
import { ReasoningStrategy } from "../reasoningStrategy.js";

describe("ReasoningStrategy", () => {
    let mockLogger: Logger;
    let strategy: ReasoningStrategy;

    beforeEach(() => {
        mockLogger = createStubInstance({
            info: () => {},
            debug: () => {},
            warn: () => {},
            error: () => {},
        } as any);

        strategy = new ReasoningStrategy(mockLogger);
    });

    describe("constructor", () => {
        it("should initialize with correct properties", () => {
            expect(strategy.type).toBe(StrategyType.REASONING);
            expect(strategy.name).toBe("ReasoningStrategy");
            expect(strategy.version).toBe("2.0.0");
        });

        it("should initialize all modular components", () => {
            // Verify that all components are properly initialized
            expect(strategy).toHaveProperty("inputHandler");
            expect(strategy).toHaveProperty("phaseExecutor");
            expect(strategy).toHaveProperty("outputGenerator");
            expect(strategy).toHaveProperty("costEstimator");
        });
    });

    describe("canHandle", () => {
        it("should return true for explicit reasoning strategy request", () => {
            const config = { strategy: "reasoning" };
            expect(strategy.canHandle("any_step", config)).to.be.true;
        });

        it("should return true for reasoning-related step types", () => {
            const reasoningSteps = [
                "analyze_data",
                "reason_about_options",
                "evaluate_results",
                "decide_next_step",
            ];

            reasoningSteps.forEach(stepType => {
                expect(strategy.canHandle(stepType)).to.be.true;
            });
        });

        it("should return false for non-reasoning step types", () => {
            const nonReasoningSteps = [
                "format_output",
                "save_file",
                "send_email",
            ];

            nonReasoningSteps.forEach(stepType => {
                expect(strategy.canHandle(stepType)).to.be.false;
            });
        });
    });

    describe("estimateResources", () => {
        it("should return ResourceUsage with all required fields", () => {
            const mockContext = {
                stepId: "test-step",
                stepType: "analyze_data",
                inputs: { data: [1, 2, 3] },
                config: { expectedOutputs: { result: { type: "Boolean" } } },
                resources: { models: [], tools: [], apis: [], credits: 5000 },
                history: { recentSteps: [], totalExecutions: 0, successRate: 1.0 },
                constraints: {},
            };

            const result = strategy.estimateResources(mockContext);

            expect(result).toHaveProperty("tokens");
            expect(result).toHaveProperty("apiCalls");
            expect(result).toHaveProperty("computeTime");
            expect(result).toHaveProperty("cost");

            expect(result.tokens).toBeGreaterThan(0);
            expect(result.apiCalls).toBeGreaterThan(0);
            expect(result.computeTime).toBeGreaterThan(0);
            expect(result.cost).toBeGreaterThan(0);
        });
    });

    describe("getPerformanceMetrics", () => {
        it("should return performance metrics structure", () => {
            const metrics = strategy.getPerformanceMetrics();

            expect(metrics).toHaveProperty("totalExecutions", 0);
            expect(metrics).toHaveProperty("successCount", 0);
            expect(metrics).toHaveProperty("failureCount", 0);
            expect(metrics).toHaveProperty("averageExecutionTime", 0);
            expect(metrics).toHaveProperty("averageResourceUsage");
            expect(metrics).toHaveProperty("averageConfidence", 0);
            expect(metrics).toHaveProperty("evolutionScore", 0.5);
        });
    });
});