/**
 * Integration tests for UnifiedRunStateMachine with routine examples
 * 
 * This test suite verifies that the unified architecture can handle
 * the complex execution patterns demonstrated in the routine examples:
 * - Multi-step BPMN workflows
 * - Decision gateways and conditional branching
 * - Parallel execution coordination
 * - Subroutine integration
 * - Context management and variable passing
 * - Multiple execution strategies
 */

import type { TierCommunicationInterface } from "@vrooli/shared";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Logger } from "winston";
import type { EventBus } from "../../../events/eventBus.js";
import { NavigatorRegistry } from "../navigation/navigatorRegistry.js";
import { getRunStateStore } from "../state/runStateStore.js";
import { MOISEGate } from "../validation/moiseGate.js";
import { UnifiedRunStateMachine } from "./unifiedRunStateMachine.js";

// Mock dependencies
const mockLogger = {
    info: vi.fn(),
    debug: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
} as unknown as Logger;

const mockEventBus = {
    on: vi.fn(),
    emit: vi.fn(),
    off: vi.fn(),
} as unknown as EventBus;

const mockTier3Executor = {
    execute: vi.fn(),
    getCapabilities: vi.fn(),
    getTierStatus: vi.fn(),
} as unknown as TierCommunicationInterface;

// Sample routine configurations representing patterns from examples
const prosConsEvaluatorRoutine = {
    __version: "3.0.0",
    graph: {
        __type: "BPMN-2.0",
        schema: {
            data: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="prosConsEvaluator">
    <bpmn:startEvent id="start" name="Decision Request"/>
    <bpmn:task id="defineDecision" name="Define Decision Context"/>
    <bpmn:task id="identifyOptions" name="Identify Available Options"/>
    <bpmn:parallelGateway id="analyzeParallel"/>
    <bpmn:task id="analyzeOption1" name="Analyze Option 1"/>
    <bpmn:task id="analyzeOption2" name="Analyze Option 2"/>
    <bpmn:parallelGateway id="convergeParallel"/>
    <bpmn:task id="compareResults" name="Compare All Options"/>
    <bpmn:exclusiveGateway id="validateResults" name="Results Conclusive?"/>
    <bpmn:task id="generateRecommendation" name="Generate Recommendation"/>
    <bpmn:task id="refineAnalysis" name="Refine Analysis"/>
    <bpmn:endEvent id="end" name="Evaluation Complete"/>

    <bpmn:sequenceFlow sourceRef="start" targetRef="defineDecision"/>
    <bpmn:sequenceFlow sourceRef="defineDecision" targetRef="identifyOptions"/>
    <bpmn:sequenceFlow sourceRef="identifyOptions" targetRef="analyzeParallel"/>
    <bpmn:sequenceFlow sourceRef="analyzeParallel" targetRef="analyzeOption1"/>
    <bpmn:sequenceFlow sourceRef="analyzeParallel" targetRef="analyzeOption2"/>
    <bpmn:sequenceFlow sourceRef="analyzeOption1" targetRef="convergeParallel"/>
    <bpmn:sequenceFlow sourceRef="analyzeOption2" targetRef="convergeParallel"/>
    <bpmn:sequenceFlow sourceRef="convergeParallel" targetRef="compareResults"/>
    <bpmn:sequenceFlow sourceRef="compareResults" targetRef="validateResults"/>
    <bpmn:sequenceFlow sourceRef="validateResults" targetRef="generateRecommendation">
      <bpmn:conditionExpression>conclusive === true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow sourceRef="validateResults" targetRef="refineAnalysis">
      <bpmn:conditionExpression>conclusive !== true</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow sourceRef="refineAnalysis" targetRef="identifyOptions"/>
    <bpmn:sequenceFlow sourceRef="generateRecommendation" targetRef="end"/>
  </process>
</definitions>`,
            activityMap: {
                defineDecision: { type: "reasoning", strategy: "conversational" },
                identifyOptions: { type: "analysis", strategy: "reasoning" },
                analyzeOption1: { type: "evaluation", strategy: "reasoning" },
                analyzeOption2: { type: "evaluation", strategy: "reasoning" },
                compareResults: { type: "comparison", strategy: "reasoning" },
                generateRecommendation: { type: "synthesis", strategy: "reasoning" },
                refineAnalysis: { type: "refinement", strategy: "conversational" },
            },
        },
    },
};

const yesManAvoidanceRoutine = {
    __version: "3.0.0",
    graph: {
        __type: "BPMN-2.0",
        schema: {
            data: `<?xml version="1.0" encoding="UTF-8"?>
<definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <process id="yesManAvoidance">
    <bpmn:startEvent id="start" name="User Request Received"/>
    <bpmn:task id="parseRequest" name="Parse Request Intent"/>
    <bpmn:exclusiveGateway id="checkComplexity" name="Is Request Complex?"/>
    <bpmn:task id="directResponse" name="Generate Direct Response"/>
    <bpmn:callActivity id="devilsAdvocate" name="Devils Advocate Analysis"/>
    <bpmn:task id="qualityCheck" name="Quality Review"/>
    <bpmn:endEvent id="end" name="Response Delivered"/>

    <bpmn:sequenceFlow sourceRef="start" targetRef="parseRequest"/>
    <bpmn:sequenceFlow sourceRef="parseRequest" targetRef="checkComplexity"/>
    <bpmn:sequenceFlow sourceRef="checkComplexity" targetRef="directResponse">
      <bpmn:conditionExpression>complexity === "simple"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow sourceRef="checkComplexity" targetRef="devilsAdvocate">
      <bpmn:conditionExpression>complexity === "complex"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow sourceRef="directResponse" targetRef="qualityCheck"/>
    <bpmn:sequenceFlow sourceRef="devilsAdvocate" targetRef="qualityCheck"/>
    <bpmn:sequenceFlow sourceRef="qualityCheck" targetRef="end"/>
  </process>
</definitions>`,
            activityMap: {
                parseRequest: { type: "analysis", strategy: "reasoning" },
                directResponse: { type: "generation", strategy: "deterministic" },
                devilsAdvocate: { type: "subroutine", strategy: "reasoning" },
                qualityCheck: { type: "validation", strategy: "reasoning" },
            },
        },
    },
};

describe("UnifiedRunStateMachine - Routine Examples Integration", () => {
    let stateMachine: UnifiedRunStateMachine;
    let navigatorRegistry: NavigatorRegistry;
    let moiseGate: MOISEGate;
    let stateStore: any;

    beforeEach(async () => {
        // Initialize dependencies
        navigatorRegistry = new NavigatorRegistry(mockLogger);
        moiseGate = new MOISEGate(mockLogger);
        stateStore = getRunStateStore();

        // Mock state store
        stateStore.initialize = vi.fn().mockResolvedValue(undefined);
        stateStore.getRun = vi.fn();
        stateStore.updateRun = vi.fn();
        stateStore.createRun = vi.fn();

        // Initialize state machine
        stateMachine = new UnifiedRunStateMachine(
            mockLogger,
            mockEventBus,
            navigatorRegistry,
            moiseGate,
            stateStore,
            mockTier3Executor,
        );
    });

    describe("Navigation System Integration", () => {
        it("should support multiple start locations from BPMN routines", async () => {
            // Test that the navigator can identify start events
            const navigator = navigatorRegistry.getNavigator("native");
            expect(navigator.canNavigate(prosConsEvaluatorRoutine)).toBe(true);

            const startLocations = navigator.getAllStartLocations(prosConsEvaluatorRoutine);
            expect(startLocations).toHaveLength(1);
            expect(startLocations[0].nodeId).toBe("start");
        });

        it("should identify parallel execution opportunities", async () => {
            // Create test context for parallel analysis
            const context = {
                runId: "test-run",
                routineId: "pros-cons-eval",
                routine: prosConsEvaluatorRoutine,
                navigator: navigatorRegistry.getNavigator("native"),
                currentLocation: { id: "test-1", routineId: "pros-cons-eval", nodeId: "analyzeParallel" },
                variables: {},
                outputs: {},
                completedSteps: [],
                parallelBranches: [],
                blackboard: {},
                scopes: [],
                resourceLimits: { maxCredits: "1000", maxDurationMs: 300000, maxMemoryMB: 512 },
                resourceUsage: { creditsUsed: "0", durationMs: 0, memoryUsedMB: 0, stepsExecuted: 0 },
                progress: { currentStepId: "analyzeParallel", completedSteps: [], percentComplete: 0 },
                retryCount: 0,
            } as any;

            // Test parallel opportunity identification
            const opportunities = await (stateMachine as any).identifyParallelOpportunities(context);
            expect(opportunities.length).toBeGreaterThan(0);

            // Should identify the parallel gateway as an opportunity
            const parallelOpp = opportunities.find(opp => opp.sourceLocationId === "analyzeParallel");
            expect(parallelOpp).toBeDefined();
            expect(parallelOpp?.parallelBranches.length).toBeGreaterThan(1);
        });

        it("should calculate total steps for complex workflows", async () => {
            const context = {
                runId: "test-run",
                routineId: "pros-cons-eval",
                routine: prosConsEvaluatorRoutine,
                navigator: navigatorRegistry.getNavigator("native"),
                currentLocation: { id: "test-1", routineId: "pros-cons-eval", nodeId: "start" },
                variables: {},
                resourceUsage: { creditsUsed: "0", durationMs: 0, memoryUsedMB: 0, stepsExecuted: 0 },
                progress: { currentStepId: "start", completedSteps: [], percentComplete: 0 },
            } as any;

            const totalSteps = await (stateMachine as any).calculateTotalSteps(context);
            expect(totalSteps).toBeGreaterThan(5); // Should identify multiple steps in the workflow
        });
    });

    describe("Event-Driven Navigation", () => {
        it("should support conditional sequence flows", async () => {
            const navigator = navigatorRegistry.getNavigator("native");
            const location = { id: "test-1", routineId: "yes-man-avoid", nodeId: "checkComplexity" };

            // Should be able to get next locations based on conditions
            const nextLocations = await navigator.getNextLocations(location, { complexity: "simple" });
            expect(nextLocations.length).toBeGreaterThan(0);
        });

        it("should handle gateway decision logic", async () => {
            const navigator = navigatorRegistry.getNavigator("native");
            const location = { id: "test-1", routineId: "pros-cons-eval", nodeId: "validateResults" };

            // Test condition-based navigation
            const nextWithConclusive = await navigator.getNextLocations(location, { conclusive: true });
            const nextWithoutConclusive = await navigator.getNextLocations(location, { conclusive: false });

            // Should route differently based on condition
            expect(nextWithConclusive).not.toEqual(nextWithoutConclusive);
        });
    });

    describe("Execution Strategy Support", () => {
        it("should support reasoning mode execution strategy", async () => {
            const capabilities = await stateMachine.getCapabilities();
            expect(capabilities.supportedStrategies).toContain("reasoning");
            expect(capabilities.supportedStrategies).toContain("conversational");
            expect(capabilities.supportedStrategies).toContain("deterministic");
        });

        it("should handle complex routine configurations", async () => {
            // Test that the state machine can handle the routine example configurations
            const navigator = navigatorRegistry.getNavigator("native");

            expect(navigator.canNavigate(prosConsEvaluatorRoutine)).toBe(true);
            expect(navigator.canNavigate(yesManAvoidanceRoutine)).toBe(true);
        });
    });

    describe("Context and State Management", () => {
        it("should maintain context across workflow steps", async () => {
            const context = {
                runId: "test-run",
                variables: { decisionContext: "test", options: ["A", "B"] },
                blackboard: { analysis: "in-progress" },
                scopes: [],
                resourceLimits: { maxCredits: "1000" },
                resourceUsage: { creditsUsed: "0", durationMs: 0, memoryUsedMB: 0, stepsExecuted: 0 },
            } as any;

            // Test context preservation through execution
            const mockResult = {
                success: true,
                outputs: { recommendation: "Option A", confidence: 0.85 },
                resourcesUsed: { creditsUsed: "10", durationMs: 1000, memoryUsedMB: 64, stepsExecuted: 1 },
            } as any;

            (stateMachine as any).updateContextWithResults(context, mockResult);

            expect(context.outputs.recommendation).toBe("Option A");
            expect(context.resourceUsage.creditsUsed).toBe("10");
            expect(context.completedSteps).toContain(context.currentLocation?.id);
        });

        it("should support swarm context inheritance", async () => {
            const swarmConfig = {
                goal: "Make informed decision",
                resources: [{ id: "res1", type: "data", content: "market research" }],
                blackboard: [{ id: "insight1", value: "Strong market demand", created_at: new Date().toISOString() }],
                executionHistory: [],
            };

            // Test swarm context integration
            const context = await (stateMachine as any).initializeContextFromSwarm(
                swarmConfig,
                prosConsEvaluatorRoutine,
                { decision: "product launch" },
                { strategy: "reasoning", maxSteps: 10, timeout: 300000 },
                "user123",
            );

            expect(context.parentContext.goal).toBe("Make informed decision");
            expect(context.parentContext.resources).toHaveLength(1);
            expect(context.sharedKnowledge).toHaveLength(1);
        });
    });

    describe("MOISE+ Integration", () => {
        it("should validate execution permissions through MOISE+ gate", async () => {
            const stepInfo = {
                id: "analyzeOption1",
                name: "Analyze Option 1",
                type: "evaluation",
                description: "Evaluate pros and cons of option 1",
            };

            const context = {
                runId: "test-run",
                variables: { userRole: "user" },
                blackboard: {},
            } as any;

            const result = await moiseGate.validateExecution({
                agent: "test-agent",
                step: stepInfo,
                context,
                teamId: "team123",
            });

            expect(result).toHaveProperty("allowed");
            expect(result).toHaveProperty("reason");
        });
    });

    describe("Error Handling and Recovery", () => {
        it("should handle navigator errors gracefully", async () => {
            const invalidRoutine = { __version: "invalid" };

            const navigator = navigatorRegistry.getNavigator("native");
            expect(navigator.canNavigate(invalidRoutine)).toBe(false);
        });

        it("should provide fallback execution plans", async () => {
            const context = {
                runId: "test-run",
                routineId: "invalid-routine",
                navigator: { type: "invalid" },
                variables: {},
                resourceLimits: { maxCredits: "1000" },
            } as any;

            // Should not crash and should provide a fallback plan
            const plan = await (stateMachine as any).createExecutionPlan(context);
            expect(plan).toHaveProperty("totalSteps");
            expect(plan).toHaveProperty("fallback");
            expect(plan.fallback).toBe(true);
        });
    });
});
