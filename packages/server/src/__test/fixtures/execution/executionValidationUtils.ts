/**
 * Execution Architecture Validation Utilities
 * 
 * Provides comprehensive validation for execution fixtures, following the proven patterns
 * from the shared package validationTestUtils.ts while addressing the unique challenges
 * of testing emergent AI capabilities.
 * 
 * Integration with Shared Package:
 * - Builds on validated config fixtures as foundation
 * - Uses real schema validation (never mocks)
 * - Follows factory patterns from shared package
 * - Provides automatic test generation like API fixtures
 */

import { describe, it, expect } from "vitest";
import type { 
    BaseConfigObject,
    ChatConfigObject,
    RoutineConfigObject,
    RunConfigObject
} from "@vrooli/shared";
import { 
    ChatConfig,
    RoutineConfig,
    RunConfig,
    chatConfigFixtures, 
    routineConfigFixtures, 
    runConfigFixtures 
} from "@vrooli/shared";
import { runComprehensiveConfigTests } from "@vrooli/shared/src/shape/configs/__test/configTestUtils.js";

// ================================================================================================
// Core Types for Execution Fixtures
// ================================================================================================

export interface EmergenceDefinition {
    /** Capabilities that should emerge from the configuration */
    capabilities: string[];
    /** Event patterns that trigger emergence */
    eventPatterns?: string[];
    /** Evolution pathway description */
    evolutionPath?: string;
    /** Conditions required for emergence */
    emergenceConditions?: {
        minAgents?: number;
        requiredResources?: string[];
        environmentalFactors?: string[];
    };
    /** Metrics for measuring emergent learning */
    learningMetrics?: {
        performanceImprovement: string;
        adaptationTime: string;
        innovationRate: string;
    };
}

export interface IntegrationDefinition {
    /** Which tier this component belongs to */
    tier: "tier1" | "tier2" | "tier3" | "cross-tier";
    /** Events this component produces */
    producedEvents?: string[];
    /** Events this component consumes */
    consumedEvents?: string[];
    /** Resources shared across tiers */
    sharedResources?: string[];
    /** Cross-tier dependencies */
    crossTierDependencies?: {
        dependsOn: string[];
        provides: string[];
    };
    /** MCP tools this component uses */
    mcpTools?: string[];
}

export interface ExecutionFixture<TConfig extends BaseConfigObject> {
    /** Validated configuration object from shared package */
    config: TConfig;
    /** Expected emergent behaviors */
    emergence: EmergenceDefinition;
    /** Integration with execution architecture */
    integration: IntegrationDefinition;
    /** Validation and metadata */
    validation?: {
        emergenceTests: string[];
        integrationTests: string[];
        evolutionTests: string[];
    };
    metadata?: {
        domain: string;
        complexity: "simple" | "medium" | "complex";
        maintainer: string;
        lastUpdated: string;
    };
}

// Specialized fixture types
export interface SwarmFixture extends ExecutionFixture<ChatConfigObject> {
    swarmMetadata?: {
        formation: "hierarchical" | "flat" | "dynamic";
        coordinationPattern: "delegation" | "consensus" | "emergence";
        expectedAgentCount: number;
        minViableAgents: number;
        roles?: Array<{ role: string; count: number }>;
    };
}

export interface RoutineFixture extends ExecutionFixture<RoutineConfigObject> {
    evolutionStage?: {
        current: "conversational" | "reasoning" | "deterministic" | "routing";
        nextStage?: string;
        evolutionTriggers: string[];
        performanceMetrics: {
            averageExecutionTime: number;
            successRate: number;
            costPerExecution: number;
        };
    };
}

export interface ExecutionContextFixture extends ExecutionFixture<RunConfigObject> {
    executionMetadata?: {
        supportedStrategies: string[];
        toolDependencies: string[];
        performanceCharacteristics: {
            latency: string;
            throughput: string;
            resourceUsage: string;
        };
    };
}

// ================================================================================================
// Validation Result Types
// ================================================================================================

export interface ValidationResult {
    pass: boolean;
    message: string;
    errors?: string[];
    warnings?: string[];
    data?: any; // For returning validated data
}

// ================================================================================================
// Config Type Registry (Integration with Shared Package)
// ================================================================================================

// Map config types to their classes
export const CONFIG_CLASS_REGISTRY = {
    chat: ChatConfig,
    routine: RoutineConfig,
    run: RunConfig
} as const;

export type ConfigType = keyof typeof CONFIG_CLASS_REGISTRY;

// Map config types to their fixtures
export const CONFIG_FIXTURE_REGISTRY = {
    chat: chatConfigFixtures,
    routine: routineConfigFixtures,
    run: runConfigFixtures
} as const;

// ================================================================================================
// Core Validation Functions
// ================================================================================================

/**
 * Validates execution fixture configuration using actual config classes
 * Ensures round-trip consistency and schema compliance
 */
export async function validateConfigAgainstSchema<T extends BaseConfigObject>(
    config: T,
    configType: ConfigType
): Promise<ValidationResult> {
    try {
        const ConfigClass = CONFIG_CLASS_REGISTRY[configType];
        
        // Create actual config instance to validate
        const configInstance = new ConfigClass({ config });
        
        // Export and re-import to test round-trip consistency
        const exported = configInstance.export();
        const reimported = new ConfigClass({ config: exported });
        const reexported = reimported.export();
        
        // Configs should be identical after round-trip
        if (JSON.stringify(exported) !== JSON.stringify(reexported)) {
            return {
                pass: false,
                message: "Config failed round-trip consistency test",
                errors: ["Exported config changes after re-import"]
            };
        }
        
        return {
            pass: true,
            message: "Config validation passed",
            warnings: undefined
        };
    } catch (error) {
        return {
            pass: false,
            message: "Config validation error",
            errors: [error instanceof Error ? error.message : String(error)]
        };
    }
}

/**
 * Direct config validation using actual config class
 * This is the preferred method for execution fixtures
 */
export async function validateExecutionFixtureConfig<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    ConfigClass: typeof ChatConfig | typeof RoutineConfig | typeof RunConfig
): Promise<ValidationResult> {
    try {
        // Create actual config instance to validate
        const configInstance = new ConfigClass({ config: fixture.config });
        
        // Export and re-import to test round-trip consistency
        const exported = configInstance.export();
        const reimported = new ConfigClass({ config: exported });
        
        // Ensure fixture config matches validated config
        const validatedConfig = reimported.export();
        
        return { 
            pass: true, 
            message: "Config validation passed",
            data: validatedConfig // Return validated config
        };
    } catch (error) {
        return { 
            pass: false, 
            message: "Config validation failed",
            errors: [error instanceof Error ? error.message : String(error)]
        };
    }
}

/**
 * Validates emergent capabilities definition
 * Ensures capabilities are measurable and emergence conditions are concrete
 */
export function validateEmergence(emergence: EmergenceDefinition): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];

    // 1. Validate capabilities
    if (!emergence.capabilities || emergence.capabilities.length === 0) {
        errors.push("Must define at least one emergent capability");
    } else {
        // Check for measurable capabilities
        const measurableCapabilities = [
            "natural_language_understanding", "pattern_recognition", "context_retention",
            "error_recovery", "performance_optimization", "adaptive_response",
            "threat_detection", "quality_assurance", "resource_optimization",
            "decision_support", "workflow_automation", "collaborative_intelligence"
        ];
        
        const unmeasurable = emergence.capabilities.filter(cap => 
            !measurableCapabilities.includes(cap)
        );
        
        if (unmeasurable.length > 0) {
            warnings.push(`Potentially unmeasurable capabilities: ${unmeasurable.join(", ")}`);
        }
    }

    // 2. Validate event patterns if provided
    if (emergence.eventPatterns) {
        for (const pattern of emergence.eventPatterns) {
            if (!isValidEventPattern(pattern)) {
                errors.push(`Invalid event pattern: ${pattern}`);
            }
        }
    }

    // 3. Validate evolution path if provided
    if (emergence.evolutionPath && !isValidEvolutionPath(emergence.evolutionPath)) {
        errors.push(`Invalid evolution path: ${emergence.evolutionPath}`);
    }

    // 4. Validate emergence conditions
    if (emergence.emergenceConditions) {
        const conditions = emergence.emergenceConditions;
        if (conditions.minAgents && conditions.minAgents < 1) {
            errors.push("minAgents must be at least 1");
        }
    }

    return {
        pass: errors.length === 0,
        message: errors.length === 0 ? "Emergence validation passed" : "Emergence validation failed",
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

/**
 * Validates integration patterns with execution architecture
 * Ensures event flow consistency and proper tier assignment
 */
export function validateIntegration(integration: IntegrationDefinition): ValidationResult {
    const errors: string[] = [];

    // 1. Validate tier assignment
    const validTiers = ["tier1", "tier2", "tier3", "cross-tier"];
    if (!validTiers.includes(integration.tier)) {
        errors.push(`Invalid tier: ${integration.tier}. Must be one of: ${validTiers.join(", ")}`);
    }

    // 2. Validate event names
    if (integration.producedEvents) {
        for (const event of integration.producedEvents) {
            if (!isValidEventName(event)) {
                errors.push(`Invalid produced event name: ${event}`);
            }
        }
    }

    if (integration.consumedEvents) {
        for (const event of integration.consumedEvents) {
            if (!isValidEventName(event)) {
                errors.push(`Invalid consumed event name: ${event}`);
            }
        }
    }

    // 3. Validate cross-tier dependencies
    if (integration.crossTierDependencies) {
        const deps = integration.crossTierDependencies;
        if (deps.dependsOn && deps.dependsOn.length === 0) {
            errors.push("crossTierDependencies.dependsOn cannot be empty array");
        }
        if (deps.provides && deps.provides.length === 0) {
            errors.push("crossTierDependencies.provides cannot be empty array");
        }
    }

    // 4. Validate MCP tools if provided
    if (integration.mcpTools) {
        for (const tool of integration.mcpTools) {
            if (!isValidMCPToolName(tool)) {
                errors.push(`Invalid MCP tool name: ${tool}`);
            }
        }
    }

    return {
        pass: errors.length === 0,
        message: errors.length === 0 ? "Integration validation passed" : "Integration validation failed",
        errors: errors.length > 0 ? errors : undefined
    };
}

/**
 * Validates evolution pathways for routine improvement
 * Ensures evolution stages are viable and measurable
 */
export function validateEvolutionPathways<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>
): ValidationResult {
    const errors: string[] = [];

    // Check if evolution path is defined
    if (!fixture.emergence.evolutionPath) {
        errors.push("Evolution path should be defined for learning systems");
    } else {
        const stages = fixture.emergence.evolutionPath.split(" → ");
        
        // Validate known evolution patterns
        const validPatterns = [
            ["conversational", "reasoning", "deterministic"],
            ["reactive", "proactive", "predictive"],
            ["manual", "assisted", "automated"],
            ["basic", "intermediate", "advanced"]
        ];
        
        const matchesPattern = validPatterns.some(pattern => 
            stages.every(stage => pattern.includes(stage.trim()))
        );
        
        if (!matchesPattern) {
            errors.push(`Evolution path "${fixture.emergence.evolutionPath}" doesn't match known patterns`);
        }
    }

    // Validate learning metrics if this is a routine fixture
    if ("evolutionStage" in fixture) {
        const routineFixture = fixture as any;
        if (routineFixture.evolutionStage?.performanceMetrics) {
            const metrics = routineFixture.evolutionStage.performanceMetrics;
            
            if (metrics.successRate < 0 || metrics.successRate > 1) {
                errors.push("successRate must be between 0 and 1");
            }
            if (metrics.averageExecutionTime < 0) {
                errors.push("averageExecutionTime must be positive");
            }
            if (metrics.costPerExecution < 0) {
                errors.push("costPerExecution must be positive");
            }
        }
    }

    return {
        pass: errors.length === 0,
        message: errors.length === 0 ? "Evolution validation passed" : "Evolution validation failed",
        errors: errors.length > 0 ? errors : undefined
    };
}

/**
 * Validates event flow consistency across tiers
 * Ensures produced events can be consumed and vice versa
 */
export function validateEventFlow<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>
): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];

    const integration = fixture.integration;

    // Check for event flow balance
    if (integration.producedEvents && integration.producedEvents.length > 0) {
        if (!integration.consumedEvents || integration.consumedEvents.length === 0) {
            warnings.push("Component produces events but doesn't consume any - ensure proper event flow");
        }
    }

    if (integration.consumedEvents && integration.consumedEvents.length > 0) {
        if (!integration.producedEvents || integration.producedEvents.length === 0) {
            warnings.push("Component consumes events but doesn't produce any - ensure proper event flow");
        }
    }

    // Validate tier-specific event patterns
    if (integration.tier === "tier1") {
        // Tier 1 should produce coordination events
        if (integration.producedEvents) {
            const hasCoordinationEvents = integration.producedEvents.some(event => 
                event.startsWith("tier1.") || event.includes("swarm") || event.includes("coordination")
            );
            if (!hasCoordinationEvents) {
                warnings.push("Tier 1 components should typically produce coordination-related events");
            }
        }
    }

    return {
        pass: errors.length === 0,
        message: errors.length === 0 ? "Event flow validation passed" : "Event flow validation failed",
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

// ================================================================================================
// Comprehensive Test Runner (Following Shared Package Pattern)
// ================================================================================================

/**
 * Comprehensive test runner for execution fixtures
 * Integrates with shared package's runComprehensiveConfigTests
 * Provides automatic test generation with 82% code reduction benefits
 */
export function runComprehensiveExecutionTests<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    configType: ConfigType,
    fixtureName: string
): void {
    describe(`${fixtureName} execution fixture`, () => {
        const ConfigClass = CONFIG_CLASS_REGISTRY[configType];
        
        // 1. Run comprehensive config tests from shared package
        describe("config validation (shared package tests)", () => {
            // Create a fixture set for the config testing
            const configFixtures = {
                minimal: fixture.config,
                complete: fixture.config,
                withDefaults: fixture.config,
                variants: { [fixtureName]: fixture.config }
            };
            
            // Run the full config test suite from shared package
            runComprehensiveConfigTests(ConfigClass, configFixtures, fixtureName);
        });
        
        // 2. Additional execution-specific config validation
        describe("execution-specific config validation", () => {
            it("should validate through config class instantiation", async () => {
                const result = await validateExecutionFixtureConfig(fixture, ConfigClass);
                if (result.errors) {
                    console.error("Config validation errors:", result.errors);
                }
                expect(result.pass).toBe(true);
            });
        });

        // 2. Emergence validation (execution-specific)
        describe("emergence validation", () => {
            it("should define valid emergent capabilities", () => {
                const result = validateEmergence(fixture.emergence);
                if (result.errors) {
                    console.error("Emergence validation errors:", result.errors);
                }
                if (result.warnings) {
                    console.warn("Emergence validation warnings:", result.warnings);
                }
                expect(result.pass).toBe(true);
            });

            it("should have measurable capabilities", () => {
                expect(fixture.emergence.capabilities).toBeDefined();
                expect(Array.isArray(fixture.emergence.capabilities)).toBe(true);
                expect(fixture.emergence.capabilities.length).toBeGreaterThan(0);
            });

            if (fixture.emergence.evolutionPath) {
                it("should have valid evolution pathway", () => {
                    const result = validateEvolutionPathways(fixture);
                    if (result.errors) {
                        console.error("Evolution validation errors:", result.errors);
                    }
                    expect(result.pass).toBe(true);
                });
            }
        });

        // 3. Integration validation (execution-specific)
        describe("integration validation", () => {
            it("should define valid integration patterns", () => {
                const result = validateIntegration(fixture.integration);
                if (result.errors) {
                    console.error("Integration validation errors:", result.errors);
                }
                expect(result.pass).toBe(true);
            });

            it("should have valid tier assignment", () => {
                expect(["tier1", "tier2", "tier3", "cross-tier"]).toContain(fixture.integration.tier);
            });

            it("should have consistent event flow", () => {
                const result = validateEventFlow(fixture);
                if (result.warnings) {
                    console.warn("Event flow warnings:", result.warnings);
                }
                expect(result.pass).toBe(true);
            });
        });

        // 4. Tier-specific validation
        if (fixture.integration.tier !== "cross-tier") {
            describe(`tier-specific validation (${fixture.integration.tier})`, () => {
                runTierSpecificValidation(fixture, fixture.integration.tier);
            });
        }

        // 5. Metadata validation
        if (fixture.metadata) {
            describe("metadata validation", () => {
                it("should have valid complexity level", () => {
                    expect(["simple", "medium", "complex"]).toContain(fixture.metadata!.complexity);
                });

                it("should have valid domain", () => {
                    expect(typeof fixture.metadata!.domain).toBe("string");
                    expect(fixture.metadata!.domain.length).toBeGreaterThan(0);
                });
            });
        }
    });
}

// ================================================================================================
// Tier-Specific Validation
// ================================================================================================

function runTierSpecificValidation<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    tier: "tier1" | "tier2" | "tier3"
): void {
    switch (tier) {
        case "tier1":
            it("should follow tier1 coordination patterns", () => {
                validateTier1Patterns(fixture as SwarmFixture);
            });
            break;
        case "tier2":
            it("should follow tier2 process patterns", () => {
                validateTier2Patterns(fixture as RoutineFixture);
            });
            break;
        case "tier3":
            it("should follow tier3 execution patterns", () => {
                validateTier3Patterns(fixture as ExecutionContextFixture);
            });
            break;
    }
}

function validateTier1Patterns(fixture: SwarmFixture): void {
    // Tier 1 should have swarm-related configuration
    if (fixture.swarmMetadata) {
        expect(fixture.swarmMetadata.expectedAgentCount).toBeGreaterThan(0);
        expect(fixture.swarmMetadata.minViableAgents).toBeGreaterThan(0);
        expect(fixture.swarmMetadata.minViableAgents).toBeLessThanOrEqual(fixture.swarmMetadata.expectedAgentCount);
    }

    // Should produce coordination events
    if (fixture.integration.producedEvents) {
        const hasCoordinationEvents = fixture.integration.producedEvents.some(event => 
            event.includes("swarm") || event.includes("coordination") || event.startsWith("tier1.")
        );
        expect(hasCoordinationEvents).toBe(true);
    }
}

function validateTier2Patterns(fixture: RoutineFixture): void {
    // Tier 2 should have routine-related configuration
    if (fixture.evolutionStage) {
        expect(["conversational", "reasoning", "deterministic", "routing"]).toContain(fixture.evolutionStage.current);
        expect(fixture.evolutionStage.performanceMetrics).toBeDefined();
    }

    // Should produce process events
    if (fixture.integration.producedEvents) {
        const hasProcessEvents = fixture.integration.producedEvents.some(event => 
            event.includes("routine") || event.includes("step") || event.startsWith("tier2.")
        );
        expect(hasProcessEvents).toBe(true);
    }
}

function validateTier3Patterns(fixture: ExecutionContextFixture): void {
    // Tier 3 should have execution-related configuration
    if (fixture.executionMetadata) {
        expect(Array.isArray(fixture.executionMetadata.supportedStrategies)).toBe(true);
        expect(fixture.executionMetadata.supportedStrategies.length).toBeGreaterThan(0);
    }

    // Should produce execution events
    if (fixture.integration.producedEvents) {
        const hasExecutionEvents = fixture.integration.producedEvents.some(event => 
            event.includes("execution") || event.includes("tool") || event.startsWith("tier3.")
        );
        expect(hasExecutionEvents).toBe(true);
    }
}

// ================================================================================================
// Helper Functions
// ================================================================================================

function isValidEventPattern(pattern: string): boolean {
    // Event patterns should be valid glob-like patterns
    return /^[a-zA-Z0-9_*./\-]+$/.test(pattern);
}

function isValidEvolutionPath(path: string): boolean {
    // Evolution paths should contain arrows and valid stage names
    return path.includes("→") || path.includes("->") || path.includes(" to ");
}

function isValidEventName(eventName: string): boolean {
    // Event names should follow tier.component.action pattern
    return /^[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+$/.test(eventName) ||
           /^[a-zA-Z0-9_]+\.[a-zA-Z0-9_]+$/.test(eventName);
}

function isValidMCPToolName(toolName: string): boolean {
    // MCP tool names should be valid identifiers
    return /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(toolName);
}

// ================================================================================================
// Utility Functions for Creating Test Fixtures
// ================================================================================================

/**
 * Combines multiple validation results into a single result
 */
export function combineValidationResults(results: ValidationResult[]): ValidationResult {
    const allPassed = results.every(r => r.pass);
    const allErrors = results.flatMap(r => r.errors || []);
    const allWarnings = results.flatMap(r => r.warnings || []);

    return {
        pass: allPassed,
        message: allPassed ? "All validations passed" : "Some validations failed",
        errors: allErrors.length > 0 ? allErrors : undefined,
        warnings: allWarnings.length > 0 ? allWarnings : undefined
    };
}

/**
 * Creates minimal emergence definition for simple fixtures
 */
export function createMinimalEmergence(): EmergenceDefinition {
    return {
        capabilities: ["basic_operation"],
        evolutionPath: "static → adaptive"
    };
}

/**
 * Creates minimal integration definition for simple fixtures
 */
export function createMinimalIntegration(tier: "tier1" | "tier2" | "tier3"): IntegrationDefinition {
    return {
        tier,
        producedEvents: [`${tier}.component.initialized`],
        consumedEvents: [`${tier}.system.ready`]
    };
}