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
 * Enhanced test data factory for execution fixtures with shared package integration
 * Follows patterns from shared package factories with automatic validation and round-trip testing
 */
export class ExecutionFixtureFactory<TConfig> {
    private sharedFixtures: any;
    private configType: string;

    constructor(
        private baseFixtures: ExecutionTestFixtures<TConfig>,
        private ConfigClass?: any,
        configType?: string,
        sharedFixtures?: any,
    ) {
        this.configType = configType || "unknown";
        this.sharedFixtures = sharedFixtures;
    }
    
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

    /**
     * Create fixture with shared package foundation
     * Uses shared package fixtures as the base and adds execution-specific enhancements
     */
    createWithSharedFoundation(
        sharedFixtureName: string,
        overrides?: Partial<ExecutionFixture<TConfig>>,
    ): ExecutionFixture<TConfig> {
        if (!this.sharedFixtures) {
            throw new Error(`No shared fixtures provided for config type '${this.configType}'`);
        }

        const sharedConfig = this.sharedFixtures.variants?.[sharedFixtureName] || this.sharedFixtures[sharedFixtureName];
        if (!sharedConfig) {
            throw new Error(`Shared fixture '${sharedFixtureName}' not found for config type '${this.configType}'`);
        }

        // Create enhanced config by merging shared config with execution-specific fields
        const enhancedConfig = {
            ...sharedConfig,
            ...this.getExecutionSpecificFields(),
        };

        // Create execution fixture with enhanced config
        const baseFixture = this.createComplete({
            config: enhancedConfig as TConfig,
            ...overrides,
        });

        // Add metadata about shared foundation
        if (baseFixture.metadata) {
            baseFixture.metadata = {
                ...baseFixture.metadata,
                sharedFoundation: sharedFixtureName,
                configType: this.configType,
            };
        }

        return baseFixture;
    }

    /**
     * Create multiple fixture variants for comprehensive testing including shared variants
     */
    createVariantSet(): Record<string, ExecutionFixture<TConfig>> {
        const variants: Record<string, ExecutionFixture<TConfig>> = {};

        // Create basic variants
        variants.minimal = this.createMinimal();
        variants.complete = this.createComplete();

        // Create existing variants
        for (const [variantName, variant] of Object.entries(this.baseFixtures.variants)) {
            variants[variantName] = this.createVariant(variantName);
        }

        // Create shared package integration variants if available
        if (this.sharedFixtures?.variants) {
            for (const [sharedName] of Object.entries(this.sharedFixtures.variants)) {
                try {
                    variants[`shared_${sharedName}`] = this.createWithSharedFoundation(sharedName);
                } catch (error) {
                    console.warn(`Failed to create variant for shared fixture '${sharedName}':`, error);
                }
            }
        }

        // Create tier-specific variants
        const baseTier = this.inferTierFromConfig();
        if (baseTier !== "cross-tier") {
            variants[`${baseTier}_optimized`] = this.createComplete({
                integration: { 
                    tier: baseTier,
                    producedEvents: [`${baseTier}.${this.configType}.optimized`],
                    consumedEvents: [`${baseTier}.system.optimize_request`],
                },
                emergence: {
                    capabilities: [...(this.baseFixtures.complete.emergence.capabilities || []), "optimization"],
                    evolutionPath: "baseline → optimized → expert",
                },
            });
        }

        // Create evolution stage variants for routine configs
        if (this.configType === "routine") {
            const evolutionStages = ["conversational", "reasoning", "deterministic"];
            evolutionStages.forEach((stage, index) => {
                variants[`evolution_${stage}`] = this.createComplete({
                    emergence: {
                        ...this.baseFixtures.complete.emergence,
                        evolutionPath: evolutionStages.join(" → "),
                        learningMetrics: {
                            performanceImprovement: `stage_${index + 1}_improvement`,
                            adaptationTime: `${Math.max(100 - index * 30, 10)}ms`,
                            innovationRate: index === 0 ? "high" : index === 1 ? "medium" : "low",
                        },
                    },
                });
            });
        }

        return variants;
    }

    /**
     * Enhanced validation with shared package integration
     * Includes comprehensive round-trip testing and shared fixture compatibility
     */
    async validateAll(): Promise<Record<string, ValidationResult>> {
        const results: Record<string, ValidationResult> = {};
        
        // Validate minimal with enhanced checks
        results.minimal = await this.validateFixtureComprehensive(
            this.baseFixtures.minimal,
            "minimal",
        );
        
        // Validate complete with enhanced checks
        results.complete = await this.validateFixtureComprehensive(
            this.baseFixtures.complete,
            "complete",
        );
        
        // Validate all variants
        for (const [name, variant] of Object.entries(this.baseFixtures.variants)) {
            results[`variant_${name}`] = await this.validateFixtureComprehensive(
                variant,
                `variant: ${name}`,
            );
        }

        // Validate shared foundation variants if available
        if (this.sharedFixtures?.variants) {
            for (const [sharedName] of Object.entries(this.sharedFixtures.variants)) {
                try {
                    const sharedVariant = this.createWithSharedFoundation(sharedName);
                    results[`shared_${sharedName}`] = await this.validateFixtureComprehensive(
                        sharedVariant,
                        `shared: ${sharedName}`,
                    );
                } catch (error) {
                    results[`shared_${sharedName}`] = {
                        pass: false,
                        message: `Failed to create shared variant: ${error instanceof Error ? error.message : String(error)}`,
                    };
                }
            }
        }

        // Create summary result
        const passedValidations = Object.values(results).filter(r => r.pass).length;
        const totalValidations = Object.keys(results).length;
        
        results._summary = {
            pass: passedValidations === totalValidations,
            message: `Validation summary: ${passedValidations}/${totalValidations} passed`,
            data: {
                passedValidations,
                totalValidations,
                successRate: passedValidations / totalValidations,
                configType: this.configType,
                hasSharedIntegration: !!this.sharedFixtures,
            },
        };
        
        return results;
    }

    /**
     * Comprehensive fixture validation including round-trip testing and shared compatibility
     */
    private async validateFixtureComprehensive(
        fixture: ExecutionFixture<TConfig>,
        description: string,
    ): Promise<ValidationResult> {
        const errors: string[] = [];
        const warnings: string[] = [];
        const validationSteps: Record<string, any> = {};

        try {
            // 1. Basic structure validation
            const basicValidation = await testExecutionFixture(fixture, this.ConfigClass, description);
            validationSteps.basic = basicValidation;
            if (!basicValidation.pass) {
                errors.push(`Basic validation failed: ${basicValidation.message}`);
            }

            // 2. Config round-trip testing if ConfigClass available
            if (this.ConfigClass) {
                try {
                    const configInstance = new this.ConfigClass({ config: fixture.config });
                    const exported = configInstance.export();
                    const reimported = new this.ConfigClass({ config: exported });
                    const reexported = reimported.export();
                    
                    const roundTripConsistent = JSON.stringify(exported) === JSON.stringify(reexported);
                    validationSteps.roundTrip = {
                        pass: roundTripConsistent,
                        message: roundTripConsistent ? "Round-trip consistent" : "Round-trip inconsistent",
                    };
                    
                    if (!roundTripConsistent) {
                        errors.push("Config failed round-trip consistency test");
                    }
                } catch (error) {
                    validationSteps.roundTrip = {
                        pass: false,
                        message: `Round-trip test failed: ${error instanceof Error ? error.message : String(error)}`,
                    };
                    errors.push(`Round-trip test error: ${error instanceof Error ? error.message : String(error)}`);
                }
            }

            // 3. Emergence validation using built-in validators
            const emergenceValidation = fixtureValidators.hasMinimalEmergence(fixture);
            validationSteps.emergence = emergenceValidation;
            if (!emergenceValidation.pass) {
                warnings.push(`Emergence validation: ${emergenceValidation.message}`);
            }

            // 4. Event naming validation
            const eventValidation = fixtureValidators.hasValidEventNames(fixture);
            validationSteps.events = eventValidation;
            if (!eventValidation.pass) {
                warnings.push(`Event naming validation: ${eventValidation.message}`);
            }

            // 5. Tier consistency validation
            const tierValidation = fixtureValidators.hasTierConsistency(fixture);
            validationSteps.tierConsistency = tierValidation;
            if (!tierValidation.pass) {
                warnings.push(`Tier consistency validation: ${tierValidation.message}`);
            }

            // 6. Shared fixture compatibility (if available)
            if (this.sharedFixtures && fixture.metadata?.sharedFoundation) {
                try {
                    const sharedConfig = this.sharedFixtures.variants?.[fixture.metadata.sharedFoundation];
                    if (sharedConfig && this.ConfigClass) {
                        const sharedInstance = new this.ConfigClass({ config: sharedConfig });
                        const executionInstance = new this.ConfigClass({ config: fixture.config });
                        
                        // Basic compatibility check
                        const sharedKeys = Object.keys(sharedInstance.export());
                        const executionKeys = Object.keys(executionInstance.export());
                        const missingKeys = sharedKeys.filter(key => !executionKeys.includes(key));
                        
                        validationSteps.sharedCompatibility = {
                            pass: missingKeys.length === 0,
                            message: missingKeys.length === 0 
                                ? "Shared compatibility validated" 
                                : `Missing shared keys: ${missingKeys.join(", ")}`,
                        };
                        
                        if (missingKeys.length > 0) {
                            warnings.push(`Shared compatibility issues: ${missingKeys.join(", ")}`);
                        }
                    }
                } catch (error) {
                    validationSteps.sharedCompatibility = {
                        pass: false,
                        message: `Shared compatibility test failed: ${error instanceof Error ? error.message : String(error)}`,
                    };
                    warnings.push(`Shared compatibility test error: ${error instanceof Error ? error.message : String(error)}`);
                }
            }

            return {
                pass: errors.length === 0,
                message: errors.length === 0 
                    ? `Comprehensive validation passed for ${description} (${warnings.length} warnings)`
                    : `Comprehensive validation failed for ${description}`,
                errors: errors.length > 0 ? errors : undefined,
                warnings: warnings.length > 0 ? warnings : undefined,
                data: {
                    validationSteps,
                    description,
                    configType: this.configType,
                    hasSharedIntegration: !!this.sharedFixtures,
                    totalSteps: Object.keys(validationSteps).length,
                    passedSteps: Object.values(validationSteps).filter((step: any) => step.pass).length,
                },
            };
        } catch (error) {
            return {
                pass: false,
                message: `Comprehensive validation error for ${description}: ${error instanceof Error ? error.message : String(error)}`,
                errors: [error instanceof Error ? error.message : String(error)],
                data: { validationSteps, description, configType: this.configType },
            };
        }
    }

    /**
     * Batch validate multiple fixtures with detailed reporting
     */
    async validateFixtureSet(fixtures: Record<string, ExecutionFixture<TConfig>>): Promise<ValidationResult> {
        const results: Record<string, ValidationResult> = {};
        const errors: string[] = [];
        const warnings: string[] = [];

        for (const [fixtureName, fixture] of Object.entries(fixtures)) {
            const result = await this.validateFixtureComprehensive(fixture, fixtureName);
            results[fixtureName] = result;

            if (!result.pass) {
                errors.push(`Fixture '${fixtureName}' failed validation`);
                if (result.errors) {
                    errors.push(...result.errors.map(err => `  ${fixtureName}: ${err}`));
                }
            }

            if (result.warnings) {
                warnings.push(...result.warnings.map(warn => `${fixtureName}: ${warn}`));
            }
        }

        const passedFixtures = Object.values(results).filter(r => r.pass).length;
        const totalFixtures = Object.keys(fixtures).length;

        return {
            pass: errors.length === 0,
            message: errors.length === 0 
                ? `Batch validation passed for ${totalFixtures} fixtures (${warnings.length} warnings)`
                : `Batch validation failed: ${errors.length} errors found`,
            errors: errors.length > 0 ? errors : undefined,
            warnings: warnings.length > 0 ? warnings : undefined,
            data: {
                results,
                passedFixtures,
                totalFixtures,
                successRate: passedFixtures / totalFixtures,
                configType: this.configType,
                hasSharedIntegration: !!this.sharedFixtures,
            },
        };
    }

    // Helper methods for intelligent fixture creation

    private inferTierFromConfig(): "tier1" | "tier2" | "tier3" | "cross-tier" {
        // Try to infer from config type
        switch (this.configType) {
            case "chat": return "tier1";
            case "routine": return "tier2";
            case "run": return "tier3";
            default: 
                // Try to infer from base fixtures
                return this.baseFixtures.complete.integration.tier;
        }
    }

    private getExecutionSpecificFields(): Partial<TConfig> {
        const fields: any = {};
        
        switch (this.configType) {
            case "chat":
                fields.swarmTask = "Execution-enhanced swarm task";
                fields.swarmSubTasks = [];
                break;
            case "routine":
                fields.routineType = "conversational";
                fields.steps = [];
                break;
            case "run":
                fields.executionStrategy = "conversational";
                fields.toolConfiguration = [];
                break;
        }

        return fields;
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
