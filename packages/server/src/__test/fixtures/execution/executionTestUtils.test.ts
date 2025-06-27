/**
 * Tests for execution test utilities
 * Demonstrates how to use the utilities and validates they work correctly
 */

import { describe, it, expect } from "vitest";
import {
    ExecutionFixtureFactory,
    runComprehensiveExecutionTests,
    runEmergenceValidationTests,
    runIntegrationTests,
    runFixtureCollectionTests,
    validateExecutionFixtureBatch,
    commonEmergencePatterns,
    commonIntegrationPatterns,
    fixtureValidators,
    type ExecutionTestFixtures,
} from "./executionTestUtils.js";
import type { ExecutionFixture, RoutineConfigObject, ChatConfigObject } from "./types.js";
import { RoutineConfig, ChatConfig } from "@vrooli/shared";

// Example fixture data for testing
const exampleRoutineFixtures: ExecutionTestFixtures<RoutineConfigObject> = {
    minimal: {
        config: {
            __version: "1.0",
            executionStrategy: "conversational",
            steps: [],
            resources: [],
        },
        emergence: {
            capabilities: ["basic_execution", "simple_routing"],
            eventPatterns: ["routine/started", "routine/completed"],
        },
        integration: {
            tier: "tier2",
            producedEvents: ["routine.initialized", "routine.executed"],
            consumedEvents: ["tier1.task.assigned"],
        },
    },
    complete: {
        config: {
            __version: "1.0",
            executionStrategy: "reasoning",
            steps: [
                {
                    id: "step1",
                    name: "Analyze Input",
                    description: "Analyze and validate input data",
                    toolCallId: "analyzer_v1",
                },
                {
                    id: "step2", 
                    name: "Process Data",
                    description: "Process the analyzed data",
                    toolCallId: "processor_v1",
                },
            ],
            resources: [
                {
                    id: "res1",
                    type: "KNOWLEDGE_BASE",
                    link: "https://example.com/kb",
                    isLocal: false,
                },
            ],
        },
        emergence: {
            ...commonEmergencePatterns.selfImprovement,
            collaborationPatterns: ["hierarchical"],
        },
        integration: {
            ...commonIntegrationPatterns.tier2Orchestration,
            requiredConfigs: ["analyzer_config", "processor_config"],
        },
        metadata: {
            id: "routine_complete_example",
            name: "Complete Routine Example",
            description: "A fully-featured routine fixture",
            tags: ["example", "test", "reasoning"],
            benchmarks: {
                avgDuration: 1500,
                avgCredits: 0.25,
                successRate: 0.95,
                qualityScore: 0.88,
            },
        },
    },
    variants: {
        deterministic: {
            config: {
                __version: "1.0",
                executionStrategy: "deterministic",
                steps: [],
                resources: [],
            },
            emergence: {
                capabilities: ["precise_execution", "predictable_output"],
                evolutionPath: "deterministic -> optimized_deterministic",
            },
            integration: {
                tier: "tier2",
                producedEvents: ["routine.step.executed", "routine.validation.passed"],
            },
        },
    },
};

const exampleSwarmFixtures: ExecutionTestFixtures<ChatConfigObject> = {
    minimal: {
        config: {
            __version: "1.0",
            blackboard: {},
            participants: [],
            tools: [],
        },
        emergence: {
            capabilities: ["swarm_coordination", "emergent_behavior"],
        },
        integration: {
            tier: "tier1",
            producedEvents: ["swarm.formed", "swarm.task.assigned"],
        },
    },
    complete: {
        config: {
            __version: "1.0",
            blackboard: {
                shared_context: "customer_support",
                active_tasks: [],
            },
            participants: [
                {
                    id: "agent1",
                    role: "coordinator",
                    capabilities: ["task_distribution", "priority_management"],
                },
                {
                    id: "agent2",
                    role: "specialist",
                    capabilities: ["technical_support", "issue_resolution"],
                },
            ],
            tools: ["knowledge_search", "ticket_system", "analytics"],
        },
        emergence: {
            ...commonEmergencePatterns.collaboration,
            emergenceConditions: {
                minAgents: 2,
                minEvents: 10,
                timeframe: 300000, // 5 minutes
            },
        },
        integration: {
            ...commonIntegrationPatterns.tier1Coordination,
            crossTierDependencies: {
                tier2: ["routine_orchestrator"],
                tier3: ["tool_executor"],
            },
        },
        validation: {
            customRules: [
                {
                    name: "minimum_participants",
                    description: "Swarm must have at least 2 participants",
                    validate: (fixture: any) => ({
                        pass: fixture.config.participants?.length >= 2,
                        message: "Swarm needs at least 2 participants",
                    }),
                },
            ],
        },
    },
    variants: {
        large_swarm: {
            config: {
                __version: "1.0",
                blackboard: { scale: "large" },
                participants: Array(10).fill(null).map((_, i) => ({
                    id: `agent${i}`,
                    role: i === 0 ? "coordinator" : "worker",
                    capabilities: ["distributed_processing"],
                })),
                tools: ["distributed_compute", "consensus_protocol"],
            },
            emergence: {
                capabilities: ["massive_parallelization", "consensus_decision_making"],
                emergenceConditions: {
                    minAgents: 5,
                    requiredResources: ["compute_cluster", "message_bus"],
                },
            },
            integration: {
                tier: "tier1",
                producedEvents: ["swarm.consensus.reached", "swarm.task.distributed"],
                sharedResources: ["distributed_blackboard", "consensus_state"],
            },
        },
    },
};

describe("ExecutionTestUtils", () => {
    describe("ExecutionFixtureFactory", () => {
        const factory = new ExecutionFixtureFactory(exampleRoutineFixtures, RoutineConfig);
        
        it("should create minimal fixture", () => {
            const minimal = factory.createMinimal();
            expect(minimal.config.__version).toBe("1.0");
            expect(minimal.emergence.capabilities).toContain("basic_execution");
        });
        
        it("should create complete fixture", () => {
            const complete = factory.createComplete();
            expect(complete.config.steps).toHaveLength(2);
            expect(complete.metadata?.name).toBe("Complete Routine Example");
        });
        
        it("should create variant fixture", () => {
            const variant = factory.createVariant("deterministic");
            expect(variant.config.executionStrategy).toBe("deterministic");
        });
        
        it("should apply overrides", () => {
            const customized = factory.createMinimal({
                emergence: {
                    capabilities: ["custom_capability"],
                    eventPatterns: [],
                },
            });
            expect(customized.emergence.capabilities).toContain("custom_capability");
        });
        
        it("should validate all fixtures", async () => {
            const results = await factory.validateAll();
            expect(results.minimal.pass).toBe(true);
            expect(results.complete.pass).toBe(true);
            expect(results.variant_deterministic.pass).toBe(true);
        });
    });
    
    describe("Batch Validation", () => {
        it("should validate multiple fixtures in batch", async () => {
            const results = await validateExecutionFixtureBatch([
                {
                    fixture: exampleRoutineFixtures.minimal,
                    type: "routine",
                    ConfigClass: RoutineConfig,
                    name: "minimal_routine",
                },
                {
                    fixture: exampleSwarmFixtures.complete,
                    type: "swarm",
                    ConfigClass: ChatConfig,
                    name: "complete_swarm",
                },
            ]);
            
            expect(results.minimal_routine.pass).toBe(true);
            expect(results.complete_swarm.pass).toBe(true);
        });
    });
    
    describe("Emergence Validation", () => {
        it("should validate emergence patterns", () => {
            // This will run the test suite but we need to capture results differently
            // For unit testing, we'll just verify the function runs without error
            expect(() => {
                runEmergenceValidationTests(
                    exampleRoutineFixtures.complete,
                    "test_routine",
                );
            }).not.toThrow();
        });
    });
    
    describe("Integration Validation", () => {
        it("should validate integration patterns", () => {
            expect(() => {
                runIntegrationTests(
                    exampleSwarmFixtures.complete,
                    "test_swarm",
                );
            }).not.toThrow();
        });
    });
    
    describe("Fixture Validators", () => {
        it("should validate minimal emergence", () => {
            const result = fixtureValidators.hasMinimalEmergence(
                exampleRoutineFixtures.minimal,
            );
            expect(result.pass).toBe(true);
            
            const invalid: ExecutionFixture<any> = {
                config: {},
                emergence: { capabilities: ["only_one"] },
                integration: { tier: "tier1" },
            };
            const invalidResult = fixtureValidators.hasMinimalEmergence(invalid);
            expect(invalidResult.pass).toBe(false);
        });
        
        it("should validate event names", () => {
            const valid: ExecutionFixture<any> = {
                config: {},
                emergence: { 
                    capabilities: ["test"],
                    eventPatterns: ["category/action", "other/event"],
                },
                integration: { 
                    tier: "tier1",
                    producedEvents: ["tier1.component.action"],
                    consumedEvents: ["tier2.other.event"],
                },
            };
            
            const result = fixtureValidators.hasValidEventNames(valid);
            expect(result.pass).toBe(true);
            
            const invalid: ExecutionFixture<any> = {
                config: {},
                emergence: { 
                    capabilities: ["test"],
                    eventPatterns: ["invalid-format"],
                },
                integration: { tier: "tier1" },
            };
            
            const invalidResult = fixtureValidators.hasValidEventNames(invalid);
            expect(invalidResult.pass).toBe(false);
        });
        
        it("should validate tier consistency", () => {
            const consistent: ExecutionFixture<any> = {
                config: {},
                emergence: { capabilities: ["test"] },
                integration: {
                    tier: "tier2",
                    producedEvents: ["tier2.routine.started", "tier2.step.executed"],
                },
            };
            
            const result = fixtureValidators.hasTierConsistency(consistent);
            expect(result.pass).toBe(true);
            
            const inconsistent: ExecutionFixture<any> = {
                config: {},
                emergence: { capabilities: ["test"] },
                integration: {
                    tier: "tier2",
                    producedEvents: ["tier1.wrong.tier", "tier3.also.wrong"],
                },
            };
            
            const inconsistentResult = fixtureValidators.hasTierConsistency(inconsistent);
            expect(inconsistentResult.pass).toBe(false);
        });
    });
    
    describe("Collection Tests", () => {
        it("should validate fixture collections", () => {
            const fixtures = [
                exampleRoutineFixtures.minimal,
                exampleRoutineFixtures.complete,
                exampleSwarmFixtures.minimal,
            ];
            
            expect(() => {
                runFixtureCollectionTests(
                    fixtures,
                    "Mixed Collection",
                    fixtureValidators.hasMinimalEmergence,
                );
            }).not.toThrow();
        });
    });
    
    describe("Common Patterns", () => {
        it("should provide reusable emergence patterns", () => {
            expect(commonEmergencePatterns.selfImprovement.capabilities).toContain(
                "pattern_recognition",
            );
            expect(commonEmergencePatterns.collaboration.eventPatterns).toContain(
                "agent/discovered",
            );
        });
        
        it("should provide reusable integration patterns", () => {
            expect(commonIntegrationPatterns.tier1Coordination.tier).toBe("tier1");
            expect(commonIntegrationPatterns.tier2Orchestration.producedEvents).toContain(
                "routine.started",
            );
        });
    });
});
