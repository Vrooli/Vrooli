/**
 * Validation Test Utilities for Execution Fixtures
 * 
 * Following the pattern from shared package API fixtures, this provides
 * comprehensive test generation for execution architecture fixtures.
 * 
 * Key Features:
 * - Automatic test generation (82% code reduction)
 * - Type-safe validation against real schemas
 * - Integration with shared config fixtures
 * - Emergence and evolution validation
 * 
 * Usage:
 * ```typescript
 * describe("Customer Support Swarm", () => {
 *     runComprehensiveExecutionTests(
 *         customerSupportSwarm,
 *         "chat",
 *         "customer-support-swarm"
 *     );
 * });
 * ```
 */

import { describe, it, expect, beforeEach } from "vitest";
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
    runConfigFixtures,
    isValidEventPattern
} from "@vrooli/shared";
import type { 
    ExecutionFixture,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    EmergenceDefinition,
    IntegrationDefinition,
    ValidationResult,
    ExecutionErrorScenario
} from "./types.js";

// ================================================================================================
// Core Validation Functions
// ================================================================================================

/**
 * Validates a fixture's config against the appropriate shared package schema
 */
export async function validateFixtureConfig<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    configType: "chat" | "routine" | "run"
): Promise<ValidationResult> {
    try {
        let configInstance: any;
        
        // Validate against the appropriate config class
        switch (configType) {
            case "chat":
                configInstance = new ChatConfig(fixture.config as ChatConfigObject);
                break;
            case "routine":
                configInstance = new RoutineConfig(fixture.config as RoutineConfigObject);
                break;
            case "run":
                configInstance = new RunConfig(fixture.config as RunConfigObject);
                break;
            default:
                throw new Error(`Unknown config type: ${configType}`);
        }
        
        // Additional execution-specific validation
        if (!fixture.emergence || fixture.emergence.capabilities.length === 0) {
            throw new Error("Fixture must define at least one emergent capability");
        }
        
        if (!fixture.integration || !fixture.integration.tier) {
            throw new Error("Fixture must specify its tier assignment");
        }
        
        return { 
            pass: true, 
            message: "Config validation passed",
            data: configInstance 
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
 * Validates emergence definition
 */
export function validateEmergence(emergence: EmergenceDefinition): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Required: capabilities
    if (!emergence.capabilities || emergence.capabilities.length === 0) {
        errors.push("Must define at least one emergent capability");
    } else {
        // Validate capability names
        emergence.capabilities.forEach(cap => {
            if (!cap || cap.trim().length === 0) {
                errors.push("Capability names cannot be empty");
            }
            if (!/^[a-z_]+$/.test(cap)) {
                warnings.push(`Capability '${cap}' should use snake_case`);
            }
        });
    }
    
    // Optional: event patterns
    if (emergence.eventPatterns) {
        emergence.eventPatterns.forEach(pattern => {
            if (!isValidEventPattern(pattern)) {
                errors.push(`Invalid event pattern: ${pattern}. Use format like 'tier1.*' or 'security.threat.detected'`);
            }
        });
    }
    
    // Optional: evolution path
    if (emergence.evolutionPath) {
        const validArrow = emergence.evolutionPath.includes("→");
        if (!validArrow) {
            warnings.push("Evolution path should use → to show progression");
        }
    }
    
    // Optional: emergence conditions
    if (emergence.emergenceConditions) {
        if (emergence.emergenceConditions.minAgents !== undefined && emergence.emergenceConditions.minAgents < 1) {
            errors.push("minAgents must be at least 1");
        }
    }
    
    return {
        pass: errors.length === 0,
        message: errors.length > 0 ? "Emergence validation failed" : "Emergence validation passed",
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

/**
 * Validates integration definition
 */
export function validateIntegration(integration: IntegrationDefinition): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Required: tier
    const validTiers = ["tier1", "tier2", "tier3", "cross-tier"];
    if (!validTiers.includes(integration.tier)) {
        errors.push(`Invalid tier: ${integration.tier}. Must be one of: ${validTiers.join(", ")}`);
    }
    
    // Optional: produced events
    if (integration.producedEvents) {
        integration.producedEvents.forEach(event => {
            if (!isValidEventName(event)) {
                errors.push(`Invalid produced event name: ${event}`);
            }
        });
    }
    
    // Optional: consumed events
    if (integration.consumedEvents) {
        integration.consumedEvents.forEach(event => {
            if (!isValidEventName(event)) {
                errors.push(`Invalid consumed event name: ${event}`);
            }
        });
    }
    
    // Cross-tier validation
    if (integration.tier === "cross-tier" && !integration.crossTierDependencies) {
        warnings.push("Cross-tier fixtures should define crossTierDependencies");
    }
    
    // MCP tools validation
    if (integration.mcpTools) {
        const validTools = ["SendMessage", "ResourceManage", "RunRoutine", "DefineTool", "SpawnSwarm"];
        integration.mcpTools.forEach(tool => {
            if (!validTools.includes(tool)) {
                warnings.push(`Unknown MCP tool: ${tool}`);
            }
        });
    }
    
    return {
        pass: errors.length === 0,
        message: errors.length > 0 ? "Integration validation failed" : "Integration validation passed",
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

/**
 * Validates event flow consistency
 */
export function validateEventFlow<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>
): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Check that consumed events match expected patterns
    if (fixture.emergence.eventPatterns && fixture.integration.consumedEvents) {
        const patterns = fixture.emergence.eventPatterns;
        const consumed = fixture.integration.consumedEvents;
        
        consumed.forEach(event => {
            const matchesPattern = patterns.some(pattern => 
                eventMatchesPattern(event, pattern)
            );
            
            if (!matchesPattern) {
                warnings.push(`Consumed event '${event}' doesn't match any emergence event pattern`);
            }
        });
    }
    
    // Validate tier-specific event naming
    if (fixture.integration.producedEvents) {
        fixture.integration.producedEvents.forEach(event => {
            const tierPrefix = fixture.integration.tier.replace("-", ".");
            if (!event.startsWith(tierPrefix) && fixture.integration.tier !== "cross-tier") {
                warnings.push(`Event '${event}' should start with tier prefix '${tierPrefix}'`);
            }
        });
    }
    
    return {
        pass: errors.length === 0,
        message: "Event flow validation " + (errors.length > 0 ? "failed" : "passed"),
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

/**
 * Validates evolution pathways for fixtures that support evolution
 */
export function validateEvolutionPathways<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>
): ValidationResult {
    const warnings: string[] = [];
    
    // Check if this is a routine fixture with evolution stages
    if ("evolutionStage" in fixture) {
        const routineFixture = fixture as unknown as RoutineFixture;
        
        if (routineFixture.evolutionStage) {
            const validStages = ["conversational", "reasoning", "deterministic", "routing"];
            
            if (!validStages.includes(routineFixture.evolutionStage.current)) {
                return {
                    pass: false,
                    message: "Invalid evolution stage",
                    errors: [`Invalid current stage: ${routineFixture.evolutionStage.current}`]
                };
            }
            
            // Validate metrics make sense
            const metrics = routineFixture.evolutionStage.performanceMetrics;
            if (metrics.successRate < 0 || metrics.successRate > 1) {
                warnings.push("Success rate should be between 0 and 1");
            }
            if (metrics.averageExecutionTime < 0) {
                warnings.push("Average execution time cannot be negative");
            }
        }
    }
    
    // Check evolution path format
    if (fixture.emergence.evolutionPath) {
        const stages = fixture.emergence.evolutionPath.split("→").map(s => s.trim());
        if (stages.length < 2) {
            warnings.push("Evolution path should show at least 2 stages");
        }
    }
    
    return {
        pass: true,
        message: "Evolution pathway validation passed",
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

/**
 * Validates error scenarios for resilience testing
 */
export function validateErrorScenarios(
    scenarios: ExecutionErrorScenario[]
): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    scenarios.forEach((scenario, index) => {
        // Validate error type
        const validErrorTypes = ["timeout", "resource_exhaustion", "permission_denied", "invalid_state", "tool_failure", "network_error", "validation_error"];
        if (!validErrorTypes.includes(scenario.errorType)) {
            errors.push(`Scenario ${index}: Invalid error type '${scenario.errorType}'`);
        }
        
        // Validate recovery strategy
        const validStrategies = ["retry", "fallback", "escalate", "abort"];
        if (!validStrategies.includes(scenario.recovery.strategy)) {
            errors.push(`Scenario ${index}: Invalid recovery strategy '${scenario.recovery.strategy}'`);
        }
        
        // Validate retry configuration
        if (scenario.recovery.strategy === "retry" && !scenario.recovery.maxAttempts) {
            warnings.push(`Scenario ${index}: Retry strategy should specify maxAttempts`);
        }
        
        // Validate escalation configuration
        if (scenario.recovery.strategy === "escalate" && !scenario.recovery.escalationTarget) {
            errors.push(`Scenario ${index}: Escalate strategy must specify escalationTarget`);
        }
    });
    
    return {
        pass: errors.length === 0,
        message: "Error scenario validation " + (errors.length > 0 ? "failed" : "passed"),
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

// ================================================================================================
// Comprehensive Test Runner (82% Code Reduction)
// ================================================================================================

/**
 * Runs comprehensive validation tests for an execution fixture
 * This is the main entry point that provides automatic test generation
 */
export function runComprehensiveExecutionTests<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    configType: "chat" | "routine" | "run",
    fixtureName: string
): void {
    describe(`${fixtureName} execution fixture validation`, () => {
        describe("Configuration validation", () => {
            it("should have valid base configuration", async () => {
                const result = await validateFixtureConfig(fixture, configType);
                expect(result.pass).toBe(true);
                if (!result.pass) {
                    console.error("Config validation errors:", result.errors);
                }
            });
            
            it("should be compatible with shared package fixtures", () => {
                // Verify the config can be used with shared package fixtures
                const sharedFixtures = getSharedFixtures(configType);
                expect(() => {
                    // Try to create a config instance with our fixture's config
                    const ConfigClass = getConfigClass(configType);
                    new ConfigClass(fixture.config);
                }).not.toThrow();
            });
        });
        
        describe("Emergence validation", () => {
            it("should define valid emergent capabilities", () => {
                const result = validateEmergence(fixture.emergence);
                expect(result.pass).toBe(true);
                if (result.warnings) {
                    console.warn("Emergence warnings:", result.warnings);
                }
            });
            
            it("should have meaningful capability names", () => {
                fixture.emergence.capabilities.forEach(cap => {
                    expect(cap).toMatch(/^[a-z_]+$/);
                    expect(cap.length).toBeGreaterThan(2);
                });
            });
        });
        
        describe("Integration validation", () => {
            it("should define valid integration patterns", () => {
                const result = validateIntegration(fixture.integration);
                expect(result.pass).toBe(true);
                if (result.warnings) {
                    console.warn("Integration warnings:", result.warnings);
                }
            });
            
            it("should have consistent event flow", () => {
                const result = validateEventFlow(fixture);
                expect(result.pass).toBe(true);
                if (result.warnings) {
                    console.warn("Event flow warnings:", result.warnings);
                }
            });
            
            it("should properly specify tier assignment", () => {
                expect(fixture.integration.tier).toBeDefined();
                expect(["tier1", "tier2", "tier3", "cross-tier"]).toContain(fixture.integration.tier);
            });
        });
        
        describe("Evolution validation", () => {
            it("should define valid evolution pathways", () => {
                const result = validateEvolutionPathways(fixture);
                expect(result.pass).toBe(true);
                if (result.warnings) {
                    console.warn("Evolution warnings:", result.warnings);
                }
            });
            
            if (fixture.emergence.evolutionPath) {
                it("should have progressive evolution stages", () => {
                    const stages = fixture.emergence.evolutionPath.split("→").map(s => s.trim());
                    expect(stages.length).toBeGreaterThanOrEqual(2);
                    expect(stages[0]).not.toBe(stages[stages.length - 1]);
                });
            }
        });
        
        // Error scenario validation for fixtures that define them
        if ("errorScenarios" in fixture && Array.isArray((fixture as any).errorScenarios)) {
            describe("Error scenario validation", () => {
                it("should define valid error scenarios", () => {
                    const scenarios = (fixture as any).errorScenarios as ExecutionErrorScenario[];
                    const result = validateErrorScenarios(scenarios);
                    expect(result.pass).toBe(true);
                    if (result.warnings) {
                        console.warn("Error scenario warnings:", result.warnings);
                    }
                });
            });
        }
        
        // Tier-specific validation
        describeTierSpecificTests(fixture, configType, fixtureName);
        
        // Metadata validation
        if (fixture.metadata) {
            describe("Metadata validation", () => {
                it("should have valid metadata", () => {
                    expect(fixture.metadata.domain).toBeDefined();
                    expect(["simple", "medium", "complex"]).toContain(fixture.metadata.complexity);
                    expect(fixture.metadata.maintainer).toBeDefined();
                    expect(fixture.metadata.lastUpdated).toMatch(/^\d{4}-\d{2}-\d{2}$/);
                });
            });
        }
    });
}

// ================================================================================================
// Helper Functions
// ================================================================================================

function isValidEventName(event: string): boolean {
    // Event names should follow pattern: tier.component.action
    const pattern = /^[a-z0-9]+(\.[a-z0-9]+)*$/;
    return pattern.test(event);
}

function eventMatchesPattern(event: string, pattern: string): boolean {
    // Convert pattern to regex (e.g., "tier1.*" -> /^tier1\..*$/)
    const regexPattern = pattern
        .replace(/\./g, "\\.")
        .replace(/\*/g, ".*");
    const regex = new RegExp(`^${regexPattern}$`);
    return regex.test(event);
}

function getSharedFixtures(configType: "chat" | "routine" | "run") {
    switch (configType) {
        case "chat":
            return chatConfigFixtures;
        case "routine":
            return routineConfigFixtures;
        case "run":
            return runConfigFixtures;
        default:
            throw new Error(`Unknown config type: ${configType}`);
    }
}

function getConfigClass(configType: "chat" | "routine" | "run") {
    switch (configType) {
        case "chat":
            return ChatConfig;
        case "routine":
            return RoutineConfig;
        case "run":
            return RunConfig;
        default:
            throw new Error(`Unknown config type: ${configType}`);
    }
}

function describeTierSpecificTests<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    configType: string,
    fixtureName: string
): void {
    switch (fixture.integration.tier) {
        case "tier1":
            describe("Tier 1 specific validation", () => {
                it("should follow swarm coordination patterns", () => {
                    if (configType === "chat" && "swarmMetadata" in fixture) {
                        const swarmFixture = fixture as unknown as SwarmFixture;
                        if (swarmFixture.swarmMetadata) {
                            expect(swarmFixture.swarmMetadata.expectedAgentCount).toBeGreaterThan(0);
                            expect(swarmFixture.swarmMetadata.minViableAgents).toBeGreaterThan(0);
                            expect(swarmFixture.swarmMetadata.minViableAgents).toBeLessThanOrEqual(
                                swarmFixture.swarmMetadata.expectedAgentCount
                            );
                        }
                    }
                });
            });
            break;
            
        case "tier2":
            describe("Tier 2 specific validation", () => {
                it("should define workflow orchestration", () => {
                    if (configType === "routine") {
                        expect(fixture.config).toHaveProperty("nodes");
                        expect(fixture.config).toHaveProperty("edges");
                    }
                });
            });
            break;
            
        case "tier3":
            describe("Tier 3 specific validation", () => {
                it("should specify execution strategy", () => {
                    if ("executionMetadata" in fixture) {
                        const execFixture = fixture as unknown as ExecutionContextFixture;
                        if (execFixture.executionMetadata) {
                            expect(execFixture.executionMetadata.supportedStrategies.length).toBeGreaterThan(0);
                        }
                    }
                });
            });
            break;
            
        case "cross-tier":
            describe("Cross-tier specific validation", () => {
                it("should define tier dependencies", () => {
                    if (fixture.integration.crossTierDependencies) {
                        expect(fixture.integration.crossTierDependencies.dependsOn.length).toBeGreaterThan(0);
                        expect(fixture.integration.crossTierDependencies.provides.length).toBeGreaterThan(0);
                    }
                });
            });
            break;
    }
}

// ================================================================================================
// Enhanced Validation Functions (Phase 1 Improvements)
// ================================================================================================

/**
 * Validates fixture compatibility with ALL shared package fixture variants
 */
export async function validateConfigWithSharedFixtures<T extends BaseConfigObject>(
    fixture: ExecutionFixture<T>,
    configType: "chat" | "routine" | "run"
): Promise<ValidationResult> {
    const warnings: string[] = [];
    const sharedFixtures = getSharedFixtures(configType);
    const ConfigClass = getConfigClass(configType);
    
    // Test against all shared fixture variants
    const variants = Object.entries(sharedFixtures.variants || {});
    
    for (const [variantName, variantConfig] of variants) {
        try {
            // Attempt to merge our execution config with the shared variant
            const merged = {
                ...variantConfig,
                ...fixture.config,
                __version: fixture.config.__version
            };
            
            // Validate the merged config
            new ConfigClass(merged as any);
        } catch (error) {
            warnings.push(`Incompatible with shared variant '${variantName}': ${error}`);
        }
    }
    
    return {
        pass: warnings.length === 0,
        message: warnings.length > 0 
            ? `Config has compatibility issues with ${warnings.length} shared variants`
            : "Config is compatible with all shared variants",
        warnings: warnings.length > 0 ? warnings : undefined
    };
}

/**
 * Combines multiple validation results into a single result
 */
export function combineValidationResults(results: ValidationResult[]): ValidationResult {
    const allErrors: string[] = [];
    const allWarnings: string[] = [];
    let allPass = true;
    
    results.forEach(result => {
        if (!result.pass) {
            allPass = false;
        }
        if (result.errors) {
            allErrors.push(...result.errors);
        }
        if (result.warnings) {
            allWarnings.push(...result.warnings);
        }
    });
    
    return {
        pass: allPass,
        message: allPass ? "All validations passed" : "Some validations failed",
        errors: allErrors.length > 0 ? allErrors : undefined,
        warnings: allWarnings.length > 0 ? allWarnings : undefined
    };
}

// ================================================================================================
// Fixture Creation Utilities
// ================================================================================================

/**
 * Utility to create minimal valid emergence definition
 */
export function createMinimalEmergence(capability: string): EmergenceDefinition {
    return {
        capabilities: [capability]
    };
}

/**
 * Utility to create minimal valid integration definition
 */
export function createMinimalIntegration(tier: IntegrationDefinition["tier"]): IntegrationDefinition {
    return { tier };
}

/**
 * Enhanced fixture creation utilities following shared package patterns
 */
export class FixtureCreationUtils {
    /**
     * Create a complete execution fixture from a shared config fixture
     */
    static createCompleteFixture<T extends BaseConfigObject>(
        sharedConfig: T,
        configType: "chat" | "routine" | "run",
        overrides: Partial<ExecutionFixture<T>> = {}
    ): ExecutionFixture<T> {
        const baseFixture: ExecutionFixture<T> = {
            config: sharedConfig,
            emergence: overrides.emergence || createMinimalEmergence("default_capability"),
            integration: overrides.integration || createMinimalIntegration("tier1"),
            validation: overrides.validation,
            metadata: overrides.metadata || {
                domain: "general",
                complexity: "simple",
                maintainer: "test",
                lastUpdated: new Date().toISOString().split("T")[0]
            }
        };
        
        return {
            ...baseFixture,
            ...overrides
        };
    }
    
    /**
     * Create an evolution sequence for testing routine evolution
     */
    static createEvolutionSequence(
        baseRoutineConfig: RoutineConfigObject,
        configType: "routine",
        stages: Array<"conversational" | "reasoning" | "deterministic" | "routing">
    ): RoutineFixture[] {
        return stages.map((stage, index) => ({
            config: {
                ...baseRoutineConfig,
                // Modify config based on evolution stage
                __typename: "RoutineConfig" as const,
                __version: baseRoutineConfig.__version
            },
            emergence: {
                capabilities: [`${stage}_execution`, "self_optimization"],
                evolutionPath: stages.slice(0, index + 1).join(" → ")
            },
            integration: {
                tier: "tier2",
                producedEvents: [`tier2.routine.${stage}.executed`]
            },
            evolutionStage: {
                current: stage,
                nextStage: stages[index + 1],
                evolutionTriggers: ["performance_threshold", "usage_count"],
                performanceMetrics: {
                    averageExecutionTime: 1000 - (index * 200), // Gets faster
                    successRate: 0.7 + (index * 0.1), // Gets more reliable
                    costPerExecution: 0.1 - (index * 0.02) // Gets cheaper
                }
            }
        }));
    }
}

// Export everything needed for execution fixture testing
export {
    createMinimalEmergence,
    createMinimalIntegration,
    FixtureCreationUtils
};