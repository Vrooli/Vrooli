/**
 * Tests for validation test utilities
 * 
 * Verifies that our validation functions work correctly and provide
 * meaningful feedback for both valid and invalid fixtures.
 */

import { describe, it, expect } from "vitest";
import { chatConfigFixtures } from "@vrooli/shared";
import type { SwarmFixture, EmergenceDefinition, IntegrationDefinition } from "../types.js";
import {
    validateEmergence,
    validateIntegration,
    validateEventFlow,
    validateFixtureConfig,
    combineValidationResults,
    createMinimalEmergence,
    createMinimalIntegration,
    FixtureCreationUtils,
} from "../validationTestUtils.js";

describe("Validation Test Utilities", () => {
    describe("validateEmergence", () => {
        it("should pass for valid emergence definition", () => {
            const emergence: EmergenceDefinition = {
                capabilities: ["pattern_recognition", "adaptive_response"],
                eventPatterns: ["tier1.*", "support.request.*"],
                evolutionPath: "reactive → proactive → predictive",
                emergenceConditions: {
                    minAgents: 3,
                    requiredResources: ["knowledge_base"],
                },
            };
            
            const result = validateEmergence(emergence);
            expect(result.pass).toBe(true);
            expect(result.errors).toBeUndefined();
        });
        
        it("should fail when no capabilities defined", () => {
            const emergence: EmergenceDefinition = {
                capabilities: [],
            };
            
            const result = validateEmergence(emergence);
            expect(result.pass).toBe(false);
            expect(result.errors).toContain("Must define at least one emergent capability");
        });
        
        it("should warn about invalid capability names", () => {
            const emergence: EmergenceDefinition = {
                capabilities: ["PatternRecognition", "adaptive-response", "valid_name"],
            };
            
            const result = validateEmergence(emergence);
            expect(result.pass).toBe(true);
            expect(result.warnings).toHaveLength(2);
            expect(result.warnings![0]).toContain("PatternRecognition");
            expect(result.warnings![1]).toContain("adaptive-response");
        });
        
        it("should validate event patterns", () => {
            const emergence: EmergenceDefinition = {
                capabilities: ["test"],
                eventPatterns: ["valid.pattern.*", "invalid pattern", "also.valid"],
            };
            
            const result = validateEmergence(emergence);
            expect(result.pass).toBe(false);
            expect(result.errors).toContain("Invalid event pattern: invalid pattern");
        });
    });
    
    describe("validateIntegration", () => {
        it("should pass for valid integration definition", () => {
            const integration: IntegrationDefinition = {
                tier: "tier1",
                producedEvents: ["tier1.swarm.initialized", "tier1.task.completed"],
                consumedEvents: ["support.request.received"],
                mcpTools: ["SendMessage", "ResourceManage"],
            };
            
            const result = validateIntegration(integration);
            expect(result.pass).toBe(true);
            expect(result.errors).toBeUndefined();
        });
        
        it("should fail for invalid tier", () => {
            const integration: IntegrationDefinition = {
                tier: "tier4" as any,
            };
            
            const result = validateIntegration(integration);
            expect(result.pass).toBe(false);
            expect(result.errors).toContain("Invalid tier: tier4");
        });
        
        it("should validate event names", () => {
            const integration: IntegrationDefinition = {
                tier: "tier2",
                producedEvents: ["valid.event", "Invalid Event", "123.starts.with.number"],
            };
            
            const result = validateIntegration(integration);
            expect(result.pass).toBe(false);
            expect(result.errors!.length).toBe(2);
        });
        
        it("should warn about unknown MCP tools", () => {
            const integration: IntegrationDefinition = {
                tier: "tier3",
                mcpTools: ["SendMessage", "UnknownTool"],
            };
            
            const result = validateIntegration(integration);
            expect(result.pass).toBe(true);
            expect(result.warnings).toContain("Unknown MCP tool: UnknownTool");
        });
    });
    
    describe("validateEventFlow", () => {
        it("should validate consumed events match patterns", () => {
            const fixture: SwarmFixture = {
                config: chatConfigFixtures.minimal.valid,
                emergence: {
                    capabilities: ["test"],
                    eventPatterns: ["support.*", "tier2.routine.*"],
                },
                integration: {
                    tier: "tier1",
                    consumedEvents: [
                        "support.request.received",
                        "tier2.routine.completed",
                        "unmatched.event",
                    ],
                },
            };
            
            const result = validateEventFlow(fixture);
            expect(result.pass).toBe(true);
            expect(result.warnings).toHaveLength(1);
            expect(result.warnings![0]).toContain("unmatched.event");
        });
        
        it("should warn about incorrect event tier prefixes", () => {
            const fixture: SwarmFixture = {
                config: chatConfigFixtures.minimal.valid,
                emergence: {
                    capabilities: ["test"],
                },
                integration: {
                    tier: "tier2",
                    producedEvents: [
                        "tier2.routine.started",
                        "tier1.wrong.prefix",
                        "no.prefix.event",
                    ],
                },
            };
            
            const result = validateEventFlow(fixture);
            expect(result.pass).toBe(true);
            expect(result.warnings).toHaveLength(2);
        });
    });
    
    describe("validateFixtureConfig", () => {
        it("should validate chat config fixture", async () => {
            const fixture: SwarmFixture = {
                config: chatConfigFixtures.complete.valid,
                emergence: {
                    capabilities: ["coordination"],
                },
                integration: {
                    tier: "tier1",
                },
            };
            
            const result = await validateFixtureConfig(fixture, "chat");
            expect(result.pass).toBe(true);
            expect(result.data).toBeDefined();
        });
        
        it("should fail without emergence capabilities", async () => {
            const fixture: SwarmFixture = {
                config: chatConfigFixtures.minimal.valid,
                emergence: {
                    capabilities: [],
                },
                integration: {
                    tier: "tier1",
                },
            };
            
            const result = await validateFixtureConfig(fixture, "chat");
            expect(result.pass).toBe(false);
            expect(result.errors).toContain("Fixture must define at least one emergent capability");
        });
        
        it("should fail without tier assignment", async () => {
            const fixture = {
                config: chatConfigFixtures.minimal.valid,
                emergence: {
                    capabilities: ["test"],
                },
                integration: {} as any,
            };
            
            const result = await validateFixtureConfig(fixture as SwarmFixture, "chat");
            expect(result.pass).toBe(false);
            expect(result.errors).toContain("Fixture must specify its tier assignment");
        });
    });
    
    describe("combineValidationResults", () => {
        it("should combine multiple passing results", () => {
            const results = [
                { pass: true, message: "Test 1 passed" },
                { pass: true, message: "Test 2 passed", warnings: ["Warning 1"] },
                { pass: true, message: "Test 3 passed" },
            ];
            
            const combined = combineValidationResults(results);
            expect(combined.pass).toBe(true);
            expect(combined.warnings).toHaveLength(1);
        });
        
        it("should fail if any result fails", () => {
            const results = [
                { pass: true, message: "Test 1 passed" },
                { pass: false, message: "Test 2 failed", errors: ["Error 1"] },
                { pass: true, message: "Test 3 passed", warnings: ["Warning 1"] },
            ];
            
            const combined = combineValidationResults(results);
            expect(combined.pass).toBe(false);
            expect(combined.errors).toHaveLength(1);
            expect(combined.warnings).toHaveLength(1);
        });
    });
    
    describe("FixtureCreationUtils", () => {
        it("should create complete fixture from shared config", () => {
            const fixture = FixtureCreationUtils.createCompleteFixture(
                chatConfigFixtures.minimal.valid,
                "chat",
                {
                    emergence: {
                        capabilities: ["custom_capability"],
                    },
                },
            );
            
            expect(fixture.config).toBe(chatConfigFixtures.minimal.valid);
            expect(fixture.emergence.capabilities).toContain("custom_capability");
            expect(fixture.integration.tier).toBe("tier1");
            expect(fixture.metadata).toBeDefined();
        });
        
        it("should create evolution sequence", () => {
            const stages = FixtureCreationUtils.createEvolutionSequence(
                chatConfigFixtures.minimal.valid as any, // Type assertion for test
                "routine",
                ["conversational", "reasoning", "deterministic"],
            );
            
            expect(stages).toHaveLength(3);
            expect(stages[0].evolutionStage!.current).toBe("conversational");
            expect(stages[2].evolutionStage!.current).toBe("deterministic");
            
            // Verify metrics improve
            expect(stages[1].evolutionStage!.performanceMetrics.averageExecutionTime).toBeLessThan(
                stages[0].evolutionStage!.performanceMetrics.averageExecutionTime,
            );
        });
    });
    
    describe("Minimal creation utilities", () => {
        it("should create minimal emergence", () => {
            const emergence = createMinimalEmergence("test_capability");
            expect(emergence.capabilities).toHaveLength(1);
            expect(emergence.capabilities[0]).toBe("test_capability");
        });
        
        it("should create minimal integration", () => {
            const integration = createMinimalIntegration("tier2");
            expect(integration.tier).toBe("tier2");
            expect(Object.keys(integration)).toHaveLength(1);
        });
    });
});
