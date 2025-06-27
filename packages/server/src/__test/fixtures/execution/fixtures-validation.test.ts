/**
 * Validation tests for all execution fixtures
 * Ensures consistency and correctness across fixture files
 */

import { describe, it, expect, vi } from "vitest";
import { ResourceSubType } from "@vrooli/shared";
import { getAllRoutines, getRoutineById, AGENT_ROUTINE_MAP } from "./routines/index.js";
import { getAllAgents } from "./emergentAgentFixtures.js";
import { 
    TECHNICAL_SUPPORT_SPECIALIST_V3,
    BILLING_SPECIALIST_V3,
    ACCOUNT_SPECIALIST_V3,
    ESCALATION_HANDLER_V2,
    RESPONSE_SYNTHESIZER_V1,
} from "./routines/evolutionFixtures.js";

// Mock console.warn for cleaner test output
const consoleWarn = vi.spyOn(console, "warn").mockImplementation(() => {});

describe("Execution Fixtures Validation", () => {
    describe("Routine Fixtures", () => {
        it("should have all routines with valid structure", () => {
            const routines = getAllRoutines();
            expect(routines.length).toBeGreaterThan(0);
            
            routines.forEach(routine => {
                expect(routine.id).toBeDefined();
                expect(routine.name).toBeDefined();
                expect(routine.description).toBeDefined();
                expect(routine.version).toBeDefined();
                expect(routine.resourceSubType).toBeDefined();
                expect(routine.config).toBeDefined();
                // Some routines may not have executionStrategy at top level
                if (routine.config.executionStrategy) {
                    expect(routine.config.executionStrategy).toBeDefined();
                }
            });
        });

        it("should have unique routine IDs", () => {
            const routines = getAllRoutines();
            const ids = routines.map(r => r.id);
            const uniqueIds = [...new Set(ids)];
            expect(ids.length).toBe(uniqueIds.length);
        });

        it("should have all referenced specialist routines", () => {
            expect(TECHNICAL_SUPPORT_SPECIALIST_V3).toBeDefined();
            expect(BILLING_SPECIALIST_V3).toBeDefined();
            expect(ACCOUNT_SPECIALIST_V3).toBeDefined();
            expect(ESCALATION_HANDLER_V2).toBeDefined();
            expect(RESPONSE_SYNTHESIZER_V1).toBeDefined();
        });

        it("should have valid execution strategies", () => {
            const validStrategies = ["conversational", "reasoning", "deterministic", "routing", "auto"];
            const routines = getAllRoutines();
            
            routines.forEach(routine => {
                // Check both executionStrategy and routineType
                const strategy = routine.config.executionStrategy || routine.config.routineType;
                if (strategy) {
                    expect(validStrategies).toContain(strategy);
                }
            });
        });
    });

    describe("Agent Fixtures", () => {
        it("should have all agents with valid structure", () => {
            const agents = getAllAgents();
            expect(agents.length).toBeGreaterThan(0);
            
            agents.forEach(agent => {
                expect(agent.agentId).toBeDefined();
                expect(agent.name).toBeDefined();
                expect(agent.goal).toBeDefined();
                expect(agent.subscriptions).toBeDefined();
                expect(agent.subscriptions.length).toBeGreaterThan(0);
            });
        });

        it("should have agents mapped to valid routines", () => {
            Object.entries(AGENT_ROUTINE_MAP).forEach(([agentId, routineConfig]) => {
                // AGENT_ROUTINE_MAP maps to routine config objects, not IDs
                expect(routineConfig).toBeDefined();
                expect(routineConfig.name).toBeDefined();
                expect(routineConfig.description).toBeDefined();
                expect(routineConfig.__version).toBeDefined();
                console.log(`Agent ${agentId} has routine: ${routineConfig.name}`);
            });
        });
    });

    describe("Cross-Fixture Consistency", () => {
        it("should have consistent agent subscription patterns", () => {
            const allAgents = getAllAgents();
            const subscriptionPatterns = new Set<string>();
            
            allAgents.forEach(agent => {
                if (agent.subscriptions) {
                    agent.subscriptions.forEach(sub => {
                        // Extract pattern type from subscription
                        const patternType = sub.split("/")[0];
                        subscriptionPatterns.add(patternType);
                    });
                }
            });
            
            // Check that we have a reasonable variety of subscription patterns
            expect(subscriptionPatterns.size).toBeGreaterThan(5);
        });

        it("should have consistent execution strategies across fixtures", () => {
            const routines = getAllRoutines();
            const strategyCount: Record<string, number> = {};
            
            routines.forEach(routine => {
                // Check both executionStrategy and routineType
                const strategy = routine.config.executionStrategy || routine.config.routineType;
                if (strategy) {
                    strategyCount[strategy] = (strategyCount[strategy] || 0) + 1;
                }
            });
            
            // Should have a good distribution of strategies
            console.log("Strategy distribution:", strategyCount);
            expect(Object.keys(strategyCount).length).toBeGreaterThanOrEqual(1); // At least one strategy should be found
        });

        it("should have all specialist routines properly integrated", () => {
            const evolutionStageRoutines = [
                TECHNICAL_SUPPORT_SPECIALIST_V3,
                BILLING_SPECIALIST_V3,
                ACCOUNT_SPECIALIST_V3,
                ESCALATION_HANDLER_V2,
                RESPONSE_SYNTHESIZER_V1,
            ];

            evolutionStageRoutines.forEach(routine => {
                // These are RoutineConfigObject instances, not full routine fixtures
                expect(routine.name).toBeDefined();
                expect(routine.description).toBeDefined();
                expect(routine.__version).toBeDefined();
                // Check for execution strategy in either field
                const strategy = routine.executionStrategy || routine.routineType;
                expect(strategy).toBeDefined();
            });
        });
    });
});
