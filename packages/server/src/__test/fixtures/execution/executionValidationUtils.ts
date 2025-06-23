/**
 * Enhanced Execution Validation Utilities
 * 
 * This module extends the base executionTestUtils.ts with additional validation
 * capabilities that ensure fixtures match production configurations and schemas.
 * 
 * Key Features:
 * - Direct validation against @vrooli/shared config classes
 * - Automatic fixture integrity checking
 * - Production schema compliance
 * - Evolution path validation
 * - Cross-tier dependency verification
 */

import { describe, expect, it } from "vitest";
import {
    ChatConfig,
    RoutineConfig,
    RunConfig,
    BotConfig,
    type ChatConfigObject,
    type RoutineConfigObject,
    type RunConfigObject,
    type BotConfigObject,
    type BaseConfigObject,
} from "@vrooli/shared";
import { chatValidation, routineValidation, runValidation, botValidation } from "@vrooli/shared";
import type {
    ExecutionFixture,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    ValidationResult,
    ExecutionTier,
} from "./types.js";
import {
    testExecutionFixture,
    runEmergenceValidationTests,
    runIntegrationTests,
} from "./executionTestUtils.js";

/**
 * Map of config types to their corresponding config classes and validation schemas
 */
const CONFIG_REGISTRY = {
    chat: { 
        ConfigClass: ChatConfig, 
        validationSchema: chatValidation.create,
        configType: "ChatConfigObject" as const,
    },
    routine: { 
        ConfigClass: RoutineConfig, 
        validationSchema: routineValidation.create,
        configType: "RoutineConfigObject" as const,
    },
    run: { 
        ConfigClass: RunConfig, 
        validationSchema: runValidation.create,
        configType: "RunConfigObject" as const,
    },
    bot: { 
        ConfigClass: BotConfig, 
        validationSchema: botValidation.create,
        configType: "BotConfigObject" as const,
    },
} as const;

/**
 * Validates that a fixture's config matches production schemas
 */
export async function validateConfigAgainstSchema<TConfig extends BaseConfigObject>(
    config: TConfig,
    configType: keyof typeof CONFIG_REGISTRY,
): Promise<ValidationResult> {
    try {
        const registry = CONFIG_REGISTRY[configType];
        if (!registry) {
            throw new Error(`Unknown config type: ${configType}`);
        }

        // Validate using the Yup schema
        await registry.validationSchema.validate(config, {
            abortEarly: false,
            stripUnknown: true,
        });

        // Validate using the config class
        const configInstance = new registry.ConfigClass({ config });
        const exported = configInstance.export();
        
        // Ensure round-trip consistency
        const rebuilt = new registry.ConfigClass({ config: exported });
        const reExported = rebuilt.export();
        
        expect(reExported).toEqual(exported);

        return {
            pass: true,
            message: `Config validation passed for ${configType}`,
        };
    } catch (error: any) {
        return {
            pass: false,
            message: `Config validation failed: ${error.message}`,
            details: { 
                error: error.message, 
                errors: error.errors,
                config,
            },
        };
    }
}

/**
 * Enhanced fixture validation that includes schema compliance
 */
export async function validateExecutionFixture<TConfig extends BaseConfigObject>(
    fixture: ExecutionFixture<TConfig>,
    configType: keyof typeof CONFIG_REGISTRY,
    description?: string,
): Promise<ValidationResult> {
    // First run basic structural validation
    const structuralResult = await testExecutionFixture(
        fixture,
        CONFIG_REGISTRY[configType].ConfigClass,
        description,
    );

    if (!structuralResult.pass) {
        return structuralResult;
    }

    // Then validate against production schema
    const schemaResult = await validateConfigAgainstSchema(fixture.config, configType);
    
    if (!schemaResult.pass) {
        return {
            pass: false,
            message: `${description || "Fixture"} failed schema validation: ${schemaResult.message}`,
            details: schemaResult.details,
        };
    }

    return {
        pass: true,
        message: `${description || "Fixture"} passed all validation`,
    };
}

/**
 * Validates a collection of fixtures ensuring they all use valid configs
 */
export async function validateFixtureCollection<TConfig extends BaseConfigObject>(
    fixtures: Array<{
        fixture: ExecutionFixture<TConfig>;
        configType: keyof typeof CONFIG_REGISTRY;
        name: string;
    }>,
): Promise<Record<string, ValidationResult>> {
    const results: Record<string, ValidationResult> = {};

    for (const { fixture, configType, name } of fixtures) {
        results[name] = await validateExecutionFixture(fixture, configType, name);
    }

    return results;
}

/**
 * Evolution path validation - ensures routines properly evolve
 */
export function validateEvolutionPath(
    fixtures: RoutineFixture[],
    pathName: string,
): void {
    describe(`Evolution Path: ${pathName}`, () => {
        it("should have fixtures in evolution order", () => {
            const strategies = fixtures.map(f => f.evolutionStage.strategy);
            const expectedOrder = ["conversational", "reasoning", "deterministic"];
            
            // Check that strategies appear in the expected order
            let lastIndex = -1;
            for (const strategy of strategies) {
                const currentIndex = expectedOrder.indexOf(strategy);
                expect(currentIndex).toBeGreaterThan(lastIndex);
                lastIndex = currentIndex;
            }
        });

        it("should show performance improvements", () => {
            for (let i = 1; i < fixtures.length; i++) {
                const prev = fixtures[i - 1].evolutionStage.metrics;
                const curr = fixtures[i].evolutionStage.metrics;
                
                // Duration should decrease
                expect(curr.avgDuration).toBeLessThan(prev.avgDuration);
                
                // Success rate should increase (if defined)
                if (prev.successRate !== undefined && curr.successRate !== undefined) {
                    expect(curr.successRate).toBeGreaterThanOrEqual(prev.successRate);
                }
                
                // Error rate should decrease (if defined)
                if (prev.errorRate !== undefined && curr.errorRate !== undefined) {
                    expect(curr.errorRate).toBeLessThanOrEqual(prev.errorRate);
                }
            }
        });

        it("should maintain version chain", () => {
            for (let i = 1; i < fixtures.length; i++) {
                const prev = fixtures[i - 1].evolutionStage;
                const curr = fixtures[i].evolutionStage;
                
                // Current should reference previous
                expect(curr.previousVersion).toBe(prev.version);
                
                // Previous should reference current as next (if set)
                if (prev.nextVersion) {
                    expect(prev.nextVersion).toBe(curr.version);
                }
            }
        });
    });
}

/**
 * Cross-tier dependency validation
 */
export function validateCrossTierDependencies(
    fixtures: Array<{
        fixture: ExecutionFixture<any>;
        tier: ExecutionTier;
    }>,
): void {
    describe("Cross-Tier Dependencies", () => {
        const fixturesByTier = fixtures.reduce((acc, { fixture, tier }) => {
            if (!acc[tier]) acc[tier] = [];
            acc[tier].push(fixture);
            return acc;
        }, {} as Record<ExecutionTier, ExecutionFixture<any>[]>);

        it("should have matching event producers and consumers", () => {
            const allProducedEvents = new Set<string>();
            const allConsumedEvents = new Set<string>();

            fixtures.forEach(({ fixture }) => {
                fixture.integration.producedEvents?.forEach(e => allProducedEvents.add(e));
                fixture.integration.consumedEvents?.forEach(e => allConsumedEvents.add(e));
            });

            // Check that consumed events have producers (with some exceptions)
            const unmatchedConsumers = [...allConsumedEvents].filter(
                event => !allProducedEvents.has(event) && !event.startsWith("system.")
            );

            if (unmatchedConsumers.length > 0) {
                console.warn("Events consumed but not produced:", unmatchedConsumers);
            }
        });

        it("should have valid tier event naming", () => {
            fixtures.forEach(({ fixture, tier }) => {
                if (tier === "cross-tier") return;

                const tierPrefix = tier.replace("tier", "tier");
                fixture.integration.producedEvents?.forEach(event => {
                    // Events should start with their tier prefix
                    if (!event.startsWith(tierPrefix + ".") && !event.includes("*")) {
                        expect.fail(`Event "${event}" from ${tier} should start with "${tierPrefix}."`);
                    }
                });
            });
        });

        it("should have shared resources properly defined", () => {
            const resourcesByTier: Record<string, Set<string>> = {};

            fixtures.forEach(({ fixture, tier }) => {
                if (!resourcesByTier[tier]) resourcesByTier[tier] = new Set();
                fixture.integration.sharedResources?.forEach(r => {
                    resourcesByTier[tier].add(r);
                });
            });

            // Resources used by multiple tiers should be well-defined
            const allResources = new Set<string>();
            Object.values(resourcesByTier).forEach(tierResources => {
                tierResources.forEach(r => allResources.add(r));
            });

            // Common resources that should be shared
            const expectedSharedResources = [
                "event_bus",
                "shared_memory",
                "metrics_collector",
            ];

            expectedSharedResources.forEach(resource => {
                const tiersUsingResource = Object.entries(resourcesByTier)
                    .filter(([_, resources]) => resources.has(resource))
                    .map(([tier]) => tier);

                if (tiersUsingResource.length > 0 && tiersUsingResource.length < 2) {
                    console.warn(`Resource "${resource}" is only used by ${tiersUsingResource.join(", ")}`);
                }
            });
        });
    });
}

/**
 * Comprehensive validation suite with schema checking
 */
export function runEnhancedComprehensiveTests<TConfig extends BaseConfigObject>(
    fixture: ExecutionFixture<TConfig>,
    configType: keyof typeof CONFIG_REGISTRY,
    fixtureName: string,
): void {
    describe(`${fixtureName} - Enhanced Comprehensive Tests`, () => {
        // Schema validation
        it("should pass schema validation", async () => {
            const result = await validateExecutionFixture(fixture, configType, fixtureName);
            expect(result.pass).toBe(true);
            if (!result.pass) {
                console.error(result.details);
            }
        });

        // Config class validation
        it("should create valid config instance", () => {
            const { ConfigClass } = CONFIG_REGISTRY[configType];
            const instance = new ConfigClass({ config: fixture.config });
            expect(instance).toBeDefined();
            expect(instance.export()).toBeDefined();
        });

        // Run standard validation suites
        runEmergenceValidationTests(fixture, fixtureName);
        runIntegrationTests(fixture, fixtureName);
    });
}

/**
 * Factory for creating validated fixtures
 */
export class ValidatedExecutionFixtureFactory<TConfig extends BaseConfigObject> {
    constructor(
        private configType: keyof typeof CONFIG_REGISTRY,
        private baseConfig: Partial<TConfig>,
    ) {}

    async create(
        emergence: ExecutionFixture<TConfig>["emergence"],
        integration: ExecutionFixture<TConfig>["integration"],
        overrides?: Partial<TConfig>,
    ): Promise<ExecutionFixture<TConfig>> {
        const config = {
            ...this.baseConfig,
            ...overrides,
        } as TConfig;

        // Validate config before creating fixture
        const validationResult = await validateConfigAgainstSchema(config, this.configType);
        if (!validationResult.pass) {
            throw new Error(`Invalid config: ${validationResult.message}`);
        }

        return {
            config,
            emergence,
            integration,
        };
    }

    async createMinimal(
        emergence: ExecutionFixture<TConfig>["emergence"],
        integration: ExecutionFixture<TConfig>["integration"],
    ): Promise<ExecutionFixture<TConfig>> {
        // Create minimal valid config based on type
        const minimalOverrides = this.getMinimalConfig();
        return this.create(emergence, integration, minimalOverrides);
    }

    private getMinimalConfig(): Partial<TConfig> {
        // Return minimal required fields based on config type
        switch (this.configType) {
            case "chat":
                return {
                    __version: "1.0.0",
                    id: `test-chat-${Date.now()}`,
                } as Partial<TConfig>;
            case "routine":
                return {
                    __version: "1.0.0",
                    id: `test-routine-${Date.now()}`,
                    name: "Test Routine",
                    description: "Minimal test routine",
                    nodes: [],
                } as Partial<TConfig>;
            case "run":
                return {
                    __version: "1.0.0",
                    id: `test-run-${Date.now()}`,
                } as Partial<TConfig>;
            case "bot":
                return {
                    __version: "1.0.0",
                    id: `test-bot-${Date.now()}`,
                } as Partial<TConfig>;
            default:
                return {} as Partial<TConfig>;
        }
    }
}

/**
 * Validates all fixtures in a directory against production schemas
 */
export async function validateAllExecutionFixtures(
    fixtures: Map<string, { fixture: ExecutionFixture<any>; configType: keyof typeof CONFIG_REGISTRY }>,
): Promise<void> {
    const results = await validateFixtureCollection(
        Array.from(fixtures.entries()).map(([name, data]) => ({
            fixture: data.fixture,
            configType: data.configType,
            name,
        })),
    );

    const failures = Object.entries(results).filter(([_, result]) => !result.pass);
    
    if (failures.length > 0) {
        console.error("Fixture validation failures:");
        failures.forEach(([name, result]) => {
            console.error(`  ${name}: ${result.message}`);
            if (result.details) {
                console.error("    Details:", result.details);
            }
        });
        throw new Error(`${failures.length} fixtures failed validation`);
    }
}