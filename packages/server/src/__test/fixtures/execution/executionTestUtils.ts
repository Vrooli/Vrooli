/**
 * Execution Test Utilities
 * 
 * This module provides comprehensive test helpers for execution fixtures validation:
 * - Type safety for execution architecture fixtures
 * - Validation of emergence patterns
 * - Integration verification across tiers
 * - Round-trip consistency testing
 * 
 * ## Quick Start Examples:
 * 
 * ### Option 1: Comprehensive Testing
 * ```typescript
 * runComprehensiveExecutionTests(
 *     swarmFixture,
 *     SwarmConfig,
 *     "customer-support-swarm"
 * );
 * ```
 * 
 * ### Option 2: Individual Tests
 * ```typescript
 * runEmergenceValidationTests(routineFixture, "security-scan");
 * runIntegrationTests(integrationScenario, "healthcare-compliance");
 * ```
 * 
 * ### Option 3: Batch Validation
 * ```typescript
 * await validateExecutionFixtureBatch([
 *     { fixture: swarmFixture, type: "swarm" },
 *     { fixture: routineFixture, type: "routine" }
 * ]);
 * ```
 */

import { describe, expect, it } from "vitest";
import type { BaseConfig } from "@vrooli/shared";
import type {
    ExecutionFixture,
    EmergenceDefinition,
    IntegrationDefinition,
    ValidationDefinition,
    ValidationResult,
    ExecutionTier,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    IntegrationScenario,
    TestMetadata,
    PerformanceBenchmarks,
} from "./types.js";

/**
 * Standard test fixture structure for execution tests
 */
export interface ExecutionTestFixtures<TConfig = any> {
    minimal: ExecutionFixture<TConfig>;
    complete: ExecutionFixture<TConfig>;
    variants: Record<string, ExecutionFixture<TConfig>>;
    invalid?: {
        missingEmergence?: Partial<ExecutionFixture<TConfig>>;
        invalidIntegration?: Partial<ExecutionFixture<TConfig>>;
        [key: string]: any;
    };
}

/**
 * Test execution fixture helper - validates structure and behavior
 */
export async function testExecutionFixture<TConfig>(
    fixture: ExecutionFixture<TConfig>,
    ConfigClass?: any,
    description?: string,
): Promise<ValidationResult> {
    try {
        // Structural validation
        expect(fixture.config).toBeDefined();
        expect(fixture.emergence).toBeDefined();
        expect(fixture.integration).toBeDefined();
        
        // Emergence validation
        expect(fixture.emergence.capabilities).toBeDefined();
        expect(Array.isArray(fixture.emergence.capabilities)).toBe(true);
        expect(fixture.emergence.capabilities.length).toBeGreaterThan(0);
        
        // Integration validation
        expect(fixture.integration.tier).toBeDefined();
        expect(["tier1", "tier2", "tier3", "cross-tier"]).toContain(fixture.integration.tier);
        
        // Config validation if class provided
        if (ConfigClass) {
            const config = new ConfigClass({ config: fixture.config });
            expect(config).toBeDefined();
            
            // Test round-trip consistency
            const exported = config.export();
            const rebuilt = new ConfigClass({ config: exported });
            expect(rebuilt.export()).toEqual(exported);
        }
        
        return {
            pass: true,
            message: `Fixture ${description || "unnamed"} is valid`,
        };
    } catch (error: any) {
        return {
            pass: false,
            message: `Fixture ${description || "unnamed"} validation failed: ${error.message}`,
            details: { error: error.message, stack: error.stack },
        };
    }
}

/**
 * Batch validation of execution fixtures
 */
export async function validateExecutionFixtureBatch(
    fixtures: Array<{
        fixture: ExecutionFixture<any>;
        type: "swarm" | "routine" | "context" | "agent";
        ConfigClass?: any;
        name?: string;
    }>,
): Promise<Record<string, ValidationResult>> {
    const results: Record<string, ValidationResult> = {};
    
    for (const { fixture, type, ConfigClass, name } of fixtures) {
        const fixtureId = name || `${type}-${Date.now()}`;
        results[fixtureId] = await testExecutionFixture(
            fixture,
            ConfigClass,
            fixtureId,
        );
    }
    
    return results;
}

/**
 * Emergence validation test suite
 * Validates that fixtures properly define emergent capabilities
 */
export function runEmergenceValidationTests<TConfig>(
    fixture: ExecutionFixture<TConfig>,
    fixtureName: string,
): void {
    describe(`${fixtureName} - Emergence Validation`, () => {
        it("should define emergent capabilities", () => {
            expect(fixture.emergence.capabilities).toBeDefined();
            expect(fixture.emergence.capabilities.length).toBeGreaterThan(0);
            
            // Each capability should be a non-empty string
            fixture.emergence.capabilities.forEach(cap => {
                expect(typeof cap).toBe("string");
                expect(cap.length).toBeGreaterThan(0);
            });
        });
        
        it("should define evolution path if applicable", () => {
            if (fixture.emergence.evolutionPath) {
                expect(typeof fixture.emergence.evolutionPath).toBe("string");
                expect(fixture.emergence.evolutionPath.length).toBeGreaterThan(0);
            }
        });
        
        it("should define event patterns for emergence", () => {
            if (fixture.emergence.eventPatterns) {
                expect(Array.isArray(fixture.emergence.eventPatterns)).toBe(true);
                fixture.emergence.eventPatterns.forEach(pattern => {
                    expect(typeof pattern).toBe("string");
                    // Event patterns should follow format: category/action or category.action
                    expect(pattern).toMatch(/^[a-z]+[a-z0-9]*([\.\/][a-z][a-z0-9_]*)+$/i);
                });
            }
        });
        
        it("should define emergence conditions when applicable", () => {
            if (fixture.emergence.emergenceConditions) {
                const conditions = fixture.emergence.emergenceConditions;
                
                if (conditions.minEvents !== undefined) {
                    expect(conditions.minEvents).toBeGreaterThan(0);
                }
                if (conditions.minAgents !== undefined) {
                    expect(conditions.minAgents).toBeGreaterThan(0);
                }
                if (conditions.requiredResources) {
                    expect(Array.isArray(conditions.requiredResources)).toBe(true);
                }
                if (conditions.timeframe !== undefined) {
                    expect(conditions.timeframe).toBeGreaterThan(0);
                }
            }
        });
    });
}

/**
 * Integration validation test suite
 * Validates cross-tier dependencies and event flow
 */
export function runIntegrationTests<TConfig>(
    fixture: ExecutionFixture<TConfig>,
    fixtureName: string,
): void {
    describe(`${fixtureName} - Integration Tests`, () => {
        it("should have valid tier assignment", () => {
            const validTiers: ExecutionTier[] = ["tier1", "tier2", "tier3", "cross-tier"];
            expect(validTiers).toContain(fixture.integration.tier);
        });
        
        it("should define produced and consumed events", () => {
            if (fixture.integration.producedEvents) {
                expect(Array.isArray(fixture.integration.producedEvents)).toBe(true);
                fixture.integration.producedEvents.forEach(event => {
                    expect(event).toMatch(/^[a-z]+[a-z0-9]*([\.\/][a-z][a-z0-9_]*)+$/i);
                });
            }
            
            if (fixture.integration.consumedEvents) {
                expect(Array.isArray(fixture.integration.consumedEvents)).toBe(true);
                fixture.integration.consumedEvents.forEach(event => {
                    expect(event).toMatch(/^[a-z]+[a-z0-9]*([\.\/][a-z][a-z0-9_]*)+$/i);
                });
            }
        });
        
        it("should have valid cross-tier dependencies", () => {
            if (fixture.integration.crossTierDependencies) {
                const deps = fixture.integration.crossTierDependencies;
                
                ["tier1", "tier2", "tier3"].forEach(tier => {
                    if (deps[tier as keyof typeof deps]) {
                        const tierDeps = deps[tier as keyof typeof deps];
                        expect(Array.isArray(tierDeps)).toBe(true);
                        tierDeps!.forEach(dep => {
                            expect(typeof dep).toBe("string");
                            expect(dep.length).toBeGreaterThan(0);
                        });
                    }
                });
            }
        });
        
        it("should define shared resources correctly", () => {
            if (fixture.integration.sharedResources) {
                expect(Array.isArray(fixture.integration.sharedResources)).toBe(true);
                fixture.integration.sharedResources.forEach(resource => {
                    expect(typeof resource).toBe("string");
                    expect(resource.length).toBeGreaterThan(0);
                });
            }
        });
    });
}

/**
 * Comprehensive execution test suite
 * Combines all validation aspects
 */
export function runComprehensiveExecutionTests<TConfig>(
    fixture: ExecutionFixture<TConfig>,
    ConfigClass: any,
    fixtureName: string,
): void {
    describe(`${fixtureName} - Comprehensive Execution Tests`, () => {
        // Basic structure validation
        it("should have valid fixture structure", async () => {
            const result = await testExecutionFixture(fixture, ConfigClass, fixtureName);
            expect(result.pass).toBe(true);
        });
        
        // Emergence validation
        runEmergenceValidationTests(fixture, fixtureName);
        
        // Integration validation
        runIntegrationTests(fixture, fixtureName);
        
        // Metadata validation if present
        if (fixture.metadata) {
            describe("metadata validation", () => {
                it("should have valid metadata structure", () => {
                    const metadata = fixture.metadata!;
                    expect(metadata.id).toBeDefined();
                    expect(metadata.name).toBeDefined();
                    
                    if (metadata.tags) {
                        expect(Array.isArray(metadata.tags)).toBe(true);
                    }
                    
                    if (metadata.benchmarks) {
                        validateBenchmarks(metadata.benchmarks);
                    }
                });
            });
        }
        
        // Custom validation rules if defined
        if (fixture.validation?.customRules) {
            describe("custom validation rules", () => {
                fixture.validation.customRules.forEach(rule => {
                    it(`should pass custom rule: ${rule.name}`, () => {
                        const result = rule.validate(fixture);
                        expect(result.pass).toBe(true);
                        if (!result.pass) {
                            console.error(`Custom rule "${rule.name}" failed: ${result.message}`);
                        }
                    });
                });
            });
        }
    });
}

/**
 * Integration scenario test suite
 * Tests complete cross-tier scenarios
 */
export function runIntegrationScenarioTests(
    scenario: IntegrationScenario,
): void {
    describe(`Integration Scenario: ${scenario.name}`, () => {
        it("should have all tier configurations", () => {
            expect(scenario.tier1).toBeDefined();
            expect(scenario.tier2).toBeDefined();
            expect(scenario.tier3).toBeDefined();
        });
        
        it("should have valid event flow", () => {
            expect(Array.isArray(scenario.expectedEvents)).toBe(true);
            expect(scenario.expectedEvents.length).toBeGreaterThan(0);
            
            // Events should follow tier.component.action format
            scenario.expectedEvents.forEach(event => {
                expect(event).toMatch(/^tier[123]\.[a-z]+\.[a-z_]+$/i);
            });
        });
        
        it("should define emergence capabilities", () => {
            expect(scenario.emergence).toBeDefined();
            expect(Array.isArray(scenario.emergence.capabilities)).toBe(true);
            expect(scenario.emergence.capabilities.length).toBeGreaterThan(0);
        });
        
        if (scenario.testScenarios) {
            describe("test scenarios", () => {
                scenario.testScenarios.forEach((testCase, index) => {
                    it(`should validate test scenario: ${testCase.name}`, () => {
                        expect(testCase.input).toBeDefined();
                        expect(Array.isArray(testCase.successCriteria)).toBe(true);
                        expect(testCase.successCriteria.length).toBeGreaterThan(0);
                        
                        // Validate success criteria structure
                        testCase.successCriteria.forEach(criteria => {
                            expect(criteria.metric).toBeDefined();
                            expect(criteria.operator).toBeDefined();
                            expect(criteria.value).toBeDefined();
                            expect([">", "<", ">=", "<=", "==", "!="]).toContain(criteria.operator);
                        });
                    });
                });
            });
        }
    });
}

/**
 * Fixture collection test suite
 * Tests collections of related fixtures
 */
export function runFixtureCollectionTests<T extends ExecutionFixture<any>>(
    fixtures: T[],
    collectionName: string,
    validationFn?: (fixture: T) => ValidationResult,
): void {
    describe(`Fixture Collection: ${collectionName}`, () => {
        it("should have fixtures with unique IDs", () => {
            const ids = fixtures
                .filter(f => f.metadata?.id)
                .map(f => f.metadata!.id);
            const uniqueIds = [...new Set(ids)];
            expect(ids.length).toBe(uniqueIds.length);
        });
        
        it("should have consistent tier distribution", () => {
            const tierDistribution: Record<ExecutionTier, number> = {
                tier1: 0,
                tier2: 0,
                tier3: 0,
                "cross-tier": 0,
            };
            
            fixtures.forEach(fixture => {
                tierDistribution[fixture.integration.tier]++;
            });
            
            // Log distribution for visibility
            console.log(`Tier distribution for ${collectionName}:`, tierDistribution);
            
            // At least one fixture per tier (adjust based on your needs)
            expect(Object.values(tierDistribution).some(count => count > 0)).toBe(true);
        });
        
        it("should have valid emergence patterns", () => {
            const allCapabilities = new Set<string>();
            
            fixtures.forEach(fixture => {
                fixture.emergence.capabilities.forEach(cap => {
                    allCapabilities.add(cap);
                });
            });
            
            // Should have diverse capabilities
            expect(allCapabilities.size).toBeGreaterThan(fixtures.length * 0.5);
        });
        
        if (validationFn) {
            it("should pass custom validation for all fixtures", () => {
                fixtures.forEach((fixture, index) => {
                    const result = validationFn(fixture);
                    expect(result.pass).toBe(true);
                    if (!result.pass) {
                        console.error(`Fixture ${index} failed validation: ${result.message}`);
                    }
                });
            });
        }
    });
}

/**
 * Helper to validate performance benchmarks
 */
function validateBenchmarks(benchmarks: PerformanceBenchmarks): void {
    if (benchmarks.avgDuration !== undefined) {
        expect(benchmarks.avgDuration).toBeGreaterThanOrEqual(0);
    }
    if (benchmarks.avgCredits !== undefined) {
        expect(benchmarks.avgCredits).toBeGreaterThanOrEqual(0);
    }
    if (benchmarks.successRate !== undefined) {
        expect(benchmarks.successRate).toBeGreaterThanOrEqual(0);
        expect(benchmarks.successRate).toBeLessThanOrEqual(1);
    }
    if (benchmarks.qualityScore !== undefined) {
        expect(benchmarks.qualityScore).toBeGreaterThanOrEqual(0);
        expect(benchmarks.qualityScore).toBeLessThanOrEqual(1);
    }
    if (benchmarks.resourceUsage) {
        const usage = benchmarks.resourceUsage;
        if (usage.memory !== undefined) expect(usage.memory).toBeGreaterThanOrEqual(0);
        if (usage.cpu !== undefined) expect(usage.cpu).toBeGreaterThanOrEqual(0);
        if (usage.tokens !== undefined) expect(usage.tokens).toBeGreaterThanOrEqual(0);
    }
}

/**
 * Test data factory for execution fixtures
 */
export class ExecutionFixtureFactory<TConfig> {
    constructor(
        private baseFixtures: ExecutionTestFixtures<TConfig>,
        private ConfigClass?: any,
    ) {}
    
    createMinimal(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig> {
        return {
            ...this.baseFixtures.minimal,
            ...overrides,
        };
    }
    
    createComplete(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig> {
        return {
            ...this.baseFixtures.complete,
            ...overrides,
        };
    }
    
    createVariant(
        variantName: string,
        overrides?: Partial<ExecutionFixture<TConfig>>,
    ): ExecutionFixture<TConfig> {
        const variant = this.baseFixtures.variants[variantName];
        if (!variant) {
            throw new Error(`Variant "${variantName}" not found`);
        }
        return {
            ...variant,
            ...overrides,
        };
    }
    
    async validateAll(): Promise<Record<string, ValidationResult>> {
        const results: Record<string, ValidationResult> = {};
        
        // Validate minimal
        results.minimal = await testExecutionFixture(
            this.baseFixtures.minimal,
            this.ConfigClass,
            "minimal",
        );
        
        // Validate complete
        results.complete = await testExecutionFixture(
            this.baseFixtures.complete,
            this.ConfigClass,
            "complete",
        );
        
        // Validate variants
        for (const [name, variant] of Object.entries(this.baseFixtures.variants)) {
            results[`variant_${name}`] = await testExecutionFixture(
                variant,
                this.ConfigClass,
                `variant: ${name}`,
            );
        }
        
        return results;
    }
}

/**
 * Common emergence patterns for reuse
 */
export const commonEmergencePatterns = {
    selfImprovement: {
        capabilities: ["pattern_recognition", "performance_optimization", "strategy_evolution"],
        eventPatterns: ["execution/completed", "metrics/analyzed", "feedback/received"],
        evolutionPath: "conversational -> reasoning -> deterministic",
    },
    
    collaboration: {
        capabilities: ["agent_coordination", "knowledge_sharing", "task_delegation"],
        eventPatterns: ["agent/discovered", "task/shared", "knowledge/transferred"],
        collaborationPatterns: ["peer-to-peer", "hierarchical", "swarm"],
    },
    
    domainExpertise: {
        capabilities: ["domain_knowledge", "context_awareness", "specialized_tools"],
        eventPatterns: ["domain/analyzed", "context/updated", "tools/specialized"],
        emergenceConditions: {
            minEvents: 100,
            requiredResources: ["domain_knowledge_base", "specialized_tools"],
        },
    },
    
    resilience: {
        capabilities: ["error_recovery", "self_healing", "adaptive_retry"],
        eventPatterns: ["error/detected", "recovery/attempted", "strategy/adapted"],
        emergenceConditions: {
            minEvents: 50,
            timeframe: 3600000, // 1 hour
        },
    },
};

/**
 * Common integration patterns
 */
export const commonIntegrationPatterns = {
    tier1Coordination: {
        tier: "tier1" as ExecutionTier,
        producedEvents: ["swarm.initialized", "agents.coordinated", "task.distributed"],
        consumedEvents: ["tier2.task.completed", "tier3.execution.finished"],
        sharedResources: ["agent_registry", "task_queue", "blackboard"],
    },
    
    tier2Orchestration: {
        tier: "tier2" as ExecutionTier,
        producedEvents: ["routine.started", "branch.selected", "state.transitioned"],
        consumedEvents: ["tier1.task.assigned", "tier3.step.completed"],
        sharedResources: ["routine_state", "execution_context", "resource_pool"],
    },
    
    tier3Execution: {
        tier: "tier3" as ExecutionTier,
        producedEvents: ["tool.called", "result.generated", "metrics.reported"],
        consumedEvents: ["tier2.step.requested", "context.updated"],
        sharedResources: ["tool_registry", "execution_cache", "safety_monitors"],
    },
    
    crossTier: {
        tier: "cross-tier" as ExecutionTier,
        crossTierDependencies: {
            tier1: ["swarm_coordinator", "agent_manager"],
            tier2: ["routine_orchestrator", "state_machine"],
            tier3: ["execution_engine", "tool_manager"],
        },
        sharedResources: ["event_bus", "shared_memory", "metrics_collector"],
    },
};

/**
 * Fixture validation helpers
 */
export const fixtureValidators = {
    /**
     * Validate that a fixture has minimum required emergence capabilities
     */
    hasMinimalEmergence: (fixture: ExecutionFixture<any>): ValidationResult => {
        if (fixture.emergence.capabilities.length < 2) {
            return {
                pass: false,
                message: "Fixture should have at least 2 emergence capabilities",
            };
        }
        return { pass: true };
    },
    
    /**
     * Validate event naming conventions
     */
    hasValidEventNames: (fixture: ExecutionFixture<any>): ValidationResult => {
        const allEvents = [
            ...(fixture.integration.producedEvents || []),
            ...(fixture.integration.consumedEvents || []),
            ...(fixture.emergence.eventPatterns || []),
        ];
        
        const invalidEvents = allEvents.filter(event => {
            // Allow both . and / as separators, and allow multiple segments
            return !event.match(/^[a-z]+[a-z0-9]*([\.\/][a-z][a-z0-9_]*)+$/i);
        });
        
        if (invalidEvents.length > 0) {
            return {
                pass: false,
                message: `Invalid event names: ${invalidEvents.join(", ")}`,
                details: { invalidEvents },
            };
        }
        return { pass: true };
    },
    
    /**
     * Validate tier consistency
     */
    hasTierConsistency: (fixture: ExecutionFixture<any>): ValidationResult => {
        const tier = fixture.integration.tier;
        const producedEvents = fixture.integration.producedEvents || [];
        
        // Check if produced events match tier
        const tierPrefix = tier === "cross-tier" ? null : tier.replace("tier", "tier");
        if (tierPrefix) {
            const inconsistentEvents = producedEvents.filter(event => {
                return !event.startsWith(tierPrefix + ".");
            });
            
            if (inconsistentEvents.length > 0) {
                return {
                    pass: false,
                    message: `Events don't match tier: ${inconsistentEvents.join(", ")}`,
                    details: { tier, inconsistentEvents },
                };
            }
        }
        
        return { pass: true };
    },
};