import { vi, expect } from "vitest";
import type { MultiAgentScenario, EventAssertion, WorkflowStep } from "../fixtures/tasks/multiAgentWorkflowFactory.js";

/**
 * Multi-Agent Workflow Test Utilities
 */

export interface EventCapture {
    eventType: string;
    data: Record<string, unknown>;
    timestamp: number;
    order: number;
}

export interface BlackboardStateCapture {
    key: string;
    value: unknown;
    timestamp: number;
    step: number;
}

export class MultiAgentTestOrchestrator {
    private eventCaptures: EventCapture[] = [];
    private blackboardCaptures: BlackboardStateCapture[] = [];
    private currentStep = 0;
    private scenario: MultiAgentScenario;

    constructor(scenario: MultiAgentScenario) {
        this.scenario = scenario;
    }

    /**
     * Mock event emission to capture events
     */
    createEventCaptureMock() {
        return vi.fn((eventType: string, data: Record<string, unknown>) => {
            this.eventCaptures.push({
                eventType,
                data,
                timestamp: Date.now(),
                order: this.eventCaptures.length + 1,
            });
            return Promise.resolve({ eventId: `event_${this.eventCaptures.length}` });
        });
    }

    /**
     * Mock blackboard updates to capture state changes
     */
    createBlackboardCaptureMock() {
        return vi.fn((key: string, value: unknown) => {
            this.blackboardCaptures.push({
                key,
                value,
                timestamp: Date.now(),
                step: this.currentStep,
            });
            return Promise.resolve();
        });
    }

    /**
     * Advance to next workflow step
     */
    nextStep() {
        this.currentStep++;
    }

    /**
     * Verify event sequence matches expected pattern
     */
    verifyEventSequence() {
        const expectedEvents = this.scenario.expectedEvents;
        
        expect(this.eventCaptures).toHaveLength(expectedEvents.length);
        
        expectedEvents.forEach((expected, index) => {
            const actual = this.eventCaptures[index];
            
            expect(actual.eventType).toBe(expected.eventType);
            expect(actual.order).toBe(expected.order);
            
            // Verify expected data fields are present
            Object.keys(expected.expectedData).forEach(key => {
                expect(actual.data).toHaveProperty(key);
                if (expected.expectedData[key] !== undefined) {
                    expect(actual.data[key]).toBe(expected.expectedData[key]);
                }
            });
        });
    }

    /**
     * Verify blackboard state progression
     */
    verifyBlackboardProgression() {
        const expectedSteps = this.scenario.workflows;
        
        expectedSteps.forEach((step, stepIndex) => {
            if (step.expectedBlackboardUpdates) {
                const stepCaptures = this.blackboardCaptures.filter(c => c.step === stepIndex);
                
                Object.keys(step.expectedBlackboardUpdates).forEach(key => {
                    const capture = stepCaptures.find(c => c.key === key);
                    expect(capture).toBeDefined();
                    expect(capture?.value).toEqual(step.expectedBlackboardUpdates![key]);
                });
            }
        });
    }

    /**
     * Verify workflow step execution
     */
    verifyWorkflowStep(stepNumber: number, actualTrigger: string, actualAction: string) {
        const expectedStep = this.scenario.workflows.find(s => s.stepNumber === stepNumber);
        expect(expectedStep).toBeDefined();
        
        expect(actualTrigger).toBe(expectedStep!.expectedTrigger);
        expect(actualAction).toContain(expectedStep!.expectedAction);
    }

    /**
     * Get captured events for debugging
     */
    getCapturedEvents(): EventCapture[] {
        return [...this.eventCaptures];
    }

    /**
     * Get captured blackboard changes for debugging
     */
    getCapturedBlackboardChanges(): BlackboardStateCapture[] {
        return [...this.blackboardCaptures];
    }

    /**
     * Reset captures for fresh test run
     */
    reset() {
        this.eventCaptures = [];
        this.blackboardCaptures = [];
        this.currentStep = 0;
    }

    /**
     * Generate workflow execution summary
     */
    generateExecutionSummary() {
        return {
            totalEvents: this.eventCaptures.length,
            totalBlackboardUpdates: this.blackboardCaptures.length,
            stepsExecuted: this.currentStep,
            eventTypes: [...new Set(this.eventCaptures.map(e => e.eventType))],
            blackboardKeys: [...new Set(this.blackboardCaptures.map(b => b.key))],
            executionTime: Math.max(...this.eventCaptures.map(e => e.timestamp)) - 
                          Math.min(...this.eventCaptures.map(e => e.timestamp)),
        };
    }
}

/**
 * Helper function to create integration test with multi-agent orchestrator
 */
export function createMultiAgentIntegrationTest(scenario: MultiAgentScenario) {
    return {
        scenario,
        orchestrator: new MultiAgentTestOrchestrator(scenario),
        
        /**
         * Setup mocks for the test
         */
        setupMocks() {
            const eventEmitMock = this.orchestrator.createEventCaptureMock();
            const blackboardUpdateMock = this.orchestrator.createBlackboardCaptureMock();
            
            return {
                eventEmit: eventEmitMock,
                blackboardUpdate: blackboardUpdateMock,
            };
        },

        /**
         * Run the complete workflow test
         */
        async runWorkflowTest(
            executeStep: (stepNumber: number, trigger: string) => Promise<string>,
        ) {
            for (const step of scenario.workflows) {
                this.orchestrator.nextStep();
                
                const actualAction = await executeStep(step.stepNumber, step.expectedTrigger);
                this.orchestrator.verifyWorkflowStep(step.stepNumber, step.expectedTrigger, actualAction);
            }
            
            this.orchestrator.verifyEventSequence();
            this.orchestrator.verifyBlackboardProgression();
            
            return this.orchestrator.generateExecutionSummary();
        },
    };
}

/**
 * Redis-specific test helpers
 */
export const redisTestHelpers = {
    /**
     * Mock Redis connection failure
     */
    mockRedisConnectionFailure() {
        return vi.fn().mockResolvedValue({
            success: false,
            errors: ["Connection timeout after 3 minutes"],
            logs: ["Redis connection error: ETIMEDOUT", "Retry attempt 1 failed", "Retry attempt 2 failed"],
        });
    },

    /**
     * Mock Redis connection success
     */
    mockRedisConnectionSuccess() {
        return vi.fn().mockResolvedValue({
            success: true,
            logs: ["Redis connection established", "Health check passed"],
        });
    },

    /**
     * Mock app lifecycle operations
     */
    createAppLifecycleMocks() {
        return {
            startApp: vi.fn().mockResolvedValue({ processId: "12345" }),
            stopApp: vi.fn().mockResolvedValue({ success: true }),
            collectLogs: vi.fn().mockResolvedValue({
                logs: ["App started", "Redis connection attempt", "Redis connection failed"],
                logFile: "/tmp/test-logs.txt",
            }),
            waitPeriod: vi.fn().mockResolvedValue({ waited: 300000 }), // 5 minutes
        };
    },
};

/**
 * Assertion helpers for multi-agent tests
 */
export const multiAgentAssertions = {
    /**
     * Assert agent coordination pattern
     */
    expectAgentCoordination(
        orchestrator: MultiAgentTestOrchestrator,
        pattern: "sequential" | "parallel" | "loop",
    ) {
        const events = orchestrator.getCapturedEvents();
        
        switch (pattern) {
            case "sequential":
                // Events should follow strict order
                for (let i = 1; i < events.length; i++) {
                    expect(events[i].timestamp).toBeGreaterThan(events[i - 1].timestamp);
                }
                break;
                
            case "parallel":
                // Some events should have similar timestamps
                const timeGroups = events.reduce((groups, event) => {
                    const timeSlot = Math.floor(event.timestamp / 1000); // Group by second
                    groups[timeSlot] = (groups[timeSlot] || 0) + 1;
                    return groups;
                }, {} as Record<number, number>);
                
                expect(Object.values(timeGroups).some(count => count > 1)).toBe(true);
                break;
                
            case "loop":
                // Should see repeated event types
                const eventTypeCounts = events.reduce((counts, event) => {
                    counts[event.eventType] = (counts[event.eventType] || 0) + 1;
                    return counts;
                }, {} as Record<string, number>);
                
                expect(Object.values(eventTypeCounts).some(count => count > 1)).toBe(true);
                break;
        }
    },

    /**
     * Assert blackboard convergence
     */
    expectBlackboardConvergence(orchestrator: MultiAgentTestOrchestrator, finalState: Record<string, unknown>) {
        const blackboardChanges = orchestrator.getCapturedBlackboardChanges();
        
        Object.keys(finalState).forEach(key => {
            const lastChange = blackboardChanges
                .filter(change => change.key === key)
                .sort((a, b) => b.timestamp - a.timestamp)[0];
                
            expect(lastChange).toBeDefined();
            expect(lastChange.value).toEqual(finalState[key]);
        });
    },
};
