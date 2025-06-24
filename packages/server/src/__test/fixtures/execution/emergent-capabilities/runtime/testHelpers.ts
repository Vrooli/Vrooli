/**
 * Runtime Test Helpers
 * 
 * Utility functions to simplify runtime validation testing of emergent capabilities.
 * Provides high-level API for common testing patterns.
 */

import { type EmergentCapabilityFixture } from "../emergentValidationUtils.js";
import { RuntimeExecutionValidator } from "./RuntimeExecutionValidator.js";
import { EvolutionValidator, type EvolutionValidationConfig } from "./EvolutionValidator.js";
import { IntegrationTestRunner, type IntegrationScenario } from "./IntegrationTestRunner.js";
import { EMERGENT_FACTORIES } from "../EmergentFixtureFactory.js";
import { TierMockFactories } from "./tierMockFactories.js";
import { type AIMockConfig } from "../../ai-mocks/types.js";

/**
 * Runtime test configuration
 */
export interface RuntimeTestConfig {
    debug?: boolean;
    captureMetrics?: boolean;
    iterationsPerScenario?: number;
    timeout?: number;
    includeEvolution?: boolean;
    includeIntegration?: boolean;
}

/**
 * Runtime test result summary
 */
export interface RuntimeTestSummary {
    fixture: string;
    success: boolean;
    runtimeValidation?: {
        success: boolean;
        scenariosPassed: number;
        totalScenarios: number;
        emergenceScore: number;
    };
    evolutionValidation?: {
        success: boolean;
        evolutionValidated: boolean;
        improvementDetected: boolean;
        overallScore: number;
    };
    integrationValidation?: {
        success: boolean;
        capabilitiesDetected: number;
        expectedCapabilities: number;
        eventFlowValid: boolean;
    };
    recommendations?: string[];
    duration: number;
}

/**
 * Run comprehensive runtime validation for a fixture
 */
export async function runRuntimeValidationTest(
    fixture: EmergentCapabilityFixture<any>,
    configType: "swarm" | "routine" | "execution" | "agent",
    config?: RuntimeTestConfig
): Promise<RuntimeTestSummary> {
    const startTime = Date.now();
    const summary: RuntimeTestSummary = {
        fixture: `${configType}_${fixture.config.name || "unnamed"}`,
        success: false,
        duration: 0
    };
    
    try {
        const factory = EMERGENT_FACTORIES[configType];
        if (!factory) {
            throw new Error(`Unknown factory type: ${configType}`);
        }
        
        // Create runtime configuration
        const runtimeConfig = factory.createRuntimeConfig(fixture, {
            debug: config?.debug,
            captureMetrics: config?.captureMetrics,
            iterationsPerScenario: config?.iterationsPerScenario,
            includeEvolutionPath: config?.includeEvolution
        });
        
        // Run runtime validation
        const validator = new RuntimeExecutionValidator();
        const runtimeResult = await validator.validateFixtureRuntime(runtimeConfig);
        
        summary.runtimeValidation = {
            success: runtimeResult.overallValidation.success,
            scenariosPassed: runtimeResult.scenarioResults.filter(s => s.success).length,
            totalScenarios: runtimeResult.scenarioResults.length,
            emergenceScore: runtimeResult.overallValidation.capabilityCoverage
        };
        
        // Run evolution validation if requested and available
        if (config?.includeEvolution && fixture.evolution) {
            const evolutionResult = await runEvolutionValidation(fixture, configType, config);
            summary.evolutionValidation = evolutionResult;
        }
        
        // Collect recommendations
        const recommendations: string[] = [];
        if (runtimeResult.overallValidation.recommendations) {
            recommendations.push(...runtimeResult.overallValidation.recommendations);
        }
        
        summary.recommendations = recommendations;
        summary.success = summary.runtimeValidation.success &&
                          (summary.evolutionValidation?.success !== false);
        
    } catch (error) {
        summary.recommendations = [`Runtime validation failed: ${error}`];
    } finally {
        summary.duration = Date.now() - startTime;
    }
    
    return summary;
}

/**
 * Run evolution validation specifically
 */
export async function runEvolutionValidation(
    fixture: EmergentCapabilityFixture<any>,
    configType: string,
    config?: RuntimeTestConfig
): Promise<{
    success: boolean;
    evolutionValidated: boolean;
    improvementDetected: boolean;
    overallScore: number;
}> {
    if (!fixture.evolution) {
        return {
            success: false,
            evolutionValidated: false,
            improvementDetected: false,
            overallScore: 0
        };
    }
    
    const factory = EMERGENT_FACTORIES[configType as keyof typeof EMERGENT_FACTORIES];
    const evolutionScenarios = factory.generateEvolutionScenarios(fixture.evolution);
    
    const evolutionConfig: EvolutionValidationConfig = {
        fixture,
        stages: evolutionScenarios.map(scenario => ({
            name: scenario.stage,
            description: scenario.description,
            mockBehaviors: scenario.mockBehaviors,
            expectedMetrics: scenario.expectedMetrics,
            expectedCapabilities: scenario.expectedCapabilities
        })),
        validationCriteria: {
            minimumImprovement: {
                executionTime: 0.2, // 20% improvement
                accuracy: 0.1,      // 10% improvement
                cost: 0.15,         // 15% improvement
                overall: 0.15       // 15% overall improvement
            },
            statisticalSignificance: {
                required: false,
                confidenceLevel: 0.95,
                minSampleSize: 3
            }
        },
        options: {
            debug: config?.debug,
            captureDetailedMetrics: config?.captureMetrics,
            iterationsPerStage: config?.iterationsPerScenario || 3
        }
    };
    
    const validator = new EvolutionValidator();
    const result = await validator.validateEvolutionPath(evolutionConfig);
    
    return {
        success: result.overallValidation.evolutionValidated,
        evolutionValidated: result.overallValidation.evolutionValidated,
        improvementDetected: result.overallValidation.improvementDetected,
        overallScore: result.overallValidation.confidenceScore
    };
}

/**
 * Run integration scenario testing
 */
export async function runIntegrationScenario(
    scenario: IntegrationScenario,
    config?: RuntimeTestConfig
): Promise<{
    success: boolean;
    capabilitiesDetected: number;
    expectedCapabilities: number;
    eventFlowValid: boolean;
}> {
    const runner = new IntegrationTestRunner();
    const result = await runner.runIntegrationScenario(scenario);
    
    return {
        success: result.success,
        capabilitiesDetected: result.detectedCapabilities.length,
        expectedCapabilities: scenario.expectedCapabilities.length,
        eventFlowValid: result.eventFlowValidation.valid
    };
}

/**
 * Create a quick integration scenario for testing
 */
export function createQuickIntegrationScenario(
    name: string,
    domain: string,
    fixtures: {
        tier1?: EmergentCapabilityFixture<any>;
        tier2?: EmergentCapabilityFixture<any>;
        tier3?: EmergentCapabilityFixture<any>;
    },
    expectedCapabilities: string[]
): IntegrationScenario {
    const tiers: any[] = [];
    const steps: any[] = [];
    
    // Add tier configurations
    if (fixtures.tier1) {
        tiers.push({
            tier: "tier1",
            fixture: fixtures.tier1,
            mockBehaviors: TierMockFactories.tier1.createSwarmCoordinationMocks(domain),
            expectedRole: "coordination",
            expectedOutputs: ["task_assignment", "resource_allocation"]
        });
        
        steps.push({
            name: "tier1_coordination",
            tier: "tier1",
            input: { task: "coordinate_execution", domain },
            triggeredEvents: ["tier1.coordination.started"]
        });
    }
    
    if (fixtures.tier2) {
        tiers.push({
            tier: "tier2",
            fixture: fixtures.tier2,
            mockBehaviors: TierMockFactories.tier2.createRoutineEvolutionMocks(domain),
            expectedRole: "orchestration",
            expectedOutputs: ["workflow_execution", "state_management"]
        });
        
        steps.push({
            name: "tier2_orchestration",
            tier: "tier2",
            input: { workflow: "process_request", domain },
            triggeredEvents: ["tier2.routine.started"],
            dependencies: fixtures.tier1 ? ["tier1_coordination"] : undefined
        });
    }
    
    if (fixtures.tier3) {
        tiers.push({
            tier: "tier3",
            fixture: fixtures.tier3,
            mockBehaviors: TierMockFactories.tier3.createExecutionStrategyMocks(),
            expectedRole: "execution",
            expectedOutputs: ["tool_execution", "result_generation"]
        });
        
        steps.push({
            name: "tier3_execution",
            tier: "tier3",
            input: { operation: "execute_tools", domain },
            triggeredEvents: ["tier3.execution.completed"],
            dependencies: fixtures.tier2 ? ["tier2_orchestration"] : 
                          fixtures.tier1 ? ["tier1_coordination"] : undefined
        });
    }
    
    // Add integration step
    if (tiers.length > 1) {
        steps.push({
            name: "cross_tier_integration",
            tier: "cross-tier",
            input: { integration: "validate_flow", domain },
            triggeredEvents: ["integration.completed"],
            dependencies: steps.map(s => s.name)
        });
    }
    
    return {
        name,
        description: `Integration test for ${domain} domain`,
        domain,
        tiers,
        steps,
        expectedCapabilities,
        expectedEventFlow: [
            {
                fromTier: "tier1",
                toTier: "tier2",
                eventType: "task.assigned",
                mandatory: true,
                sequence: 1
            },
            {
                fromTier: "tier2",
                toTier: "tier3",
                eventType: "workflow.step",
                mandatory: true,
                sequence: 2
            },
            {
                fromTier: "tier3",
                toTier: "tier1",
                eventType: "execution.result",
                mandatory: true,
                sequence: 3
            }
        ],
        validationCriteria: {
            minimumCapabilityDetection: 0.8,
            maximumEventLatency: 1000,
            requiredEventFlow: 0.8,
            emergenceThreshold: 0.7,
            crossTierCoordination: {
                required: true,
                minimumInteractions: tiers.length
            }
        }
    };
}

/**
 * Run a complete test suite for an emergent capability fixture
 */
export async function runCompleteTestSuite(
    fixture: EmergentCapabilityFixture<any>,
    configType: "swarm" | "routine" | "execution" | "agent",
    config?: RuntimeTestConfig & {
        includeStressTests?: boolean;
        includeErrorScenarios?: boolean;
    }
): Promise<{
    runtimeTests: RuntimeTestSummary;
    integrationTests?: any;
    stressTests?: any;
    errorTests?: any;
    overallSuccess: boolean;
    recommendations: string[];
}> {
    const results: any = {};
    const allRecommendations: string[] = [];
    
    // Run runtime validation
    results.runtimeTests = await runRuntimeValidationTest(fixture, configType, config);
    if (results.runtimeTests.recommendations) {
        allRecommendations.push(...results.runtimeTests.recommendations);
    }
    
    // Run integration tests if requested
    if (config?.includeIntegration) {
        const integrationScenario = createQuickIntegrationScenario(
            `${configType}_integration_test`,
            fixture.metadata?.domain || "general",
            { [configType.replace("swarm", "tier1").replace("routine", "tier2").replace("execution", "tier3")]: fixture },
            fixture.emergence.capabilities
        );
        
        results.integrationTests = await runIntegrationScenario(integrationScenario, config);
    }
    
    // Run stress tests if requested
    if (config?.includeStressTests) {
        results.stressTests = await runStressTests(fixture, configType, config);
    }
    
    // Run error scenario tests if requested
    if (config?.includeErrorScenarios) {
        results.errorTests = await runErrorScenarios(fixture, configType, config);
    }
    
    // Determine overall success
    const overallSuccess = results.runtimeTests.success &&
                          (results.integrationTests?.success !== false) &&
                          (results.stressTests?.success !== false) &&
                          (results.errorTests?.success !== false);
    
    return {
        ...results,
        overallSuccess,
        recommendations: allRecommendations
    };
}

/**
 * Run stress tests for a fixture
 */
async function runStressTests(
    fixture: EmergentCapabilityFixture<any>,
    configType: string,
    config?: RuntimeTestConfig
): Promise<{ success: boolean; details: any }> {
    // Create high-load scenarios
    const factory = EMERGENT_FACTORIES[configType as keyof typeof EMERGENT_FACTORIES];
    const stressConfig = factory.createRuntimeConfig(fixture, {
        debug: config?.debug,
        iterationsPerScenario: 10, // Higher iteration count
        captureMetrics: true
    });
    
    // Add stress-specific scenarios
    stressConfig.testScenarios.push({
        name: "high_load_stress_test",
        description: "Test capability under high load",
        inputEvents: Array.from({ length: 20 }, (_, i) => ({
            type: `stress.load.${i}`,
            data: { iteration: i, concurrency: "high" }
        })),
        expectedCapabilities: fixture.emergence.capabilities,
        expectedBehaviors: [
            {
                type: "response_pattern",
                pattern: /maintain|consistent|stable/i,
                occurrences: { min: 15 } // Expect stability in most responses
            }
        ],
        timeConstraints: {
            maxDuration: 30000,
            checkpoints: []
        }
    });
    
    const validator = new RuntimeExecutionValidator();
    const result = await validator.validateFixtureRuntime(stressConfig);
    
    return {
        success: result.overallValidation.success && result.overallValidation.capabilityCoverage > 0.7,
        details: {
            scenariosPassed: result.scenarioResults.filter(s => s.success).length,
            totalScenarios: result.scenarioResults.length,
            averageExecutionTime: result.scenarioResults.reduce((sum, s) => sum + s.executionTime, 0) / result.scenarioResults.length,
            emergenceScore: result.overallValidation.capabilityCoverage
        }
    };
}

/**
 * Run error scenario tests
 */
async function runErrorScenarios(
    fixture: EmergentCapabilityFixture<any>,
    configType: string,
    config?: RuntimeTestConfig
): Promise<{ success: boolean; details: any }> {
    // Create error simulation mocks
    const errorMocks = TierMockFactories.createErrorSimulationMocks();
    
    const factory = EMERGENT_FACTORIES[configType as keyof typeof EMERGENT_FACTORIES];
    const errorConfig = factory.createRuntimeConfig(fixture, {
        debug: config?.debug,
        customMockBehaviors: errorMocks
    });
    
    // Add error scenarios
    errorConfig.testScenarios.push({
        name: "error_resilience_test",
        description: "Test resilience to various error conditions",
        inputEvents: [
            { type: "error.network", data: { errorType: "timeout" } },
            { type: "error.resource", data: { errorType: "exhaustion" } },
            { type: "error.rate_limit", data: { errorType: "throttling" } }
        ],
        expectedCapabilities: fixture.emergence.capabilities.filter(cap => 
            cap.includes("resilience") || cap.includes("recovery") || cap.includes("handling")
        ),
        expectedBehaviors: [
            {
                type: "response_pattern",
                pattern: /error|recover|fallback|retry/i,
                occurrences: { min: 1 }
            }
        ]
    });
    
    const validator = new RuntimeExecutionValidator();
    const result = await validator.validateFixtureRuntime(errorConfig);
    
    // Error scenarios are successful if the system handles errors gracefully
    const errorHandlingSuccess = result.scenarioResults.some(s => 
        s.scenario === "error_resilience_test" && 
        (s.success || s.behaviorMatches.some(b => b.matched))
    );
    
    return {
        success: errorHandlingSuccess,
        details: {
            errorScenariosHandled: result.scenarioResults.filter(s => 
                s.scenario.includes("error") && s.success
            ).length,
            gracefulDegradation: errorHandlingSuccess,
            recoveryCapabilities: result.scenarioResults.flatMap(s => s.detectedCapabilities)
                .filter(cap => cap.includes("recovery") || cap.includes("resilience"))
        }
    };
}

/**
 * Helper to create mock behaviors for specific scenarios
 */
export function createScenarioMocks(
    scenario: "customer_support" | "security_monitoring" | "data_processing" | "general",
    tier: "tier1" | "tier2" | "tier3" | "cross-tier"
): Map<string, AIMockConfig> {
    const mocks = new Map<string, AIMockConfig>();
    
    // Get base tier mocks
    const baseMocks = (() => {
        switch (tier) {
            case "tier1":
                return TierMockFactories.tier1.createSwarmCoordinationMocks(scenario);
            case "tier2":
                return TierMockFactories.tier2.createRoutineEvolutionMocks(scenario);
            case "tier3":
                return TierMockFactories.tier3.createExecutionStrategyMocks();
            case "cross-tier":
                return TierMockFactories.crossTier.createIntegrationMocks(scenario);
            default:
                return new Map();
        }
    })();
    
    // Add scenario-specific enhancements
    baseMocks.forEach((config, id) => {
        const enhancedConfig = { ...config };
        
        // Add scenario-specific metadata
        enhancedConfig.metadata = {
            ...enhancedConfig.metadata,
            scenario,
            tier,
            testContext: true
        };
        
        mocks.set(id, enhancedConfig);
    });
    
    return mocks;
}

/**
 * Validate that runtime results meet minimum quality standards
 */
export function validateRuntimeQuality(
    summary: RuntimeTestSummary,
    minimumStandards?: {
        successRate?: number;
        emergenceScore?: number;
        maxDuration?: number;
    }
): {
    meetsStandards: boolean;
    issues: string[];
    recommendations: string[];
} {
    const standards = {
        successRate: minimumStandards?.successRate || 0.8,
        emergenceScore: minimumStandards?.emergenceScore || 0.7,
        maxDuration: minimumStandards?.maxDuration || 60000
    };
    
    const issues: string[] = [];
    const recommendations: string[] = [];
    
    // Check success rate
    if (summary.runtimeValidation) {
        const successRate = summary.runtimeValidation.scenariosPassed / summary.runtimeValidation.totalScenarios;
        if (successRate < standards.successRate) {
            issues.push(`Success rate ${successRate.toFixed(2)} below minimum ${standards.successRate}`);
            recommendations.push("Review scenario definitions and mock behaviors");
        }
    }
    
    // Check emergence score
    if (summary.runtimeValidation && summary.runtimeValidation.emergenceScore < standards.emergenceScore) {
        issues.push(`Emergence score ${summary.runtimeValidation.emergenceScore.toFixed(2)} below minimum ${standards.emergenceScore}`);
        recommendations.push("Enhance capability definitions and detection patterns");
    }
    
    // Check duration
    if (summary.duration > standards.maxDuration) {
        issues.push(`Test duration ${summary.duration}ms exceeds maximum ${standards.maxDuration}ms`);
        recommendations.push("Optimize test scenarios and reduce timeout values");
    }
    
    // Check evolution quality if available
    if (summary.evolutionValidation && !summary.evolutionValidation.improvementDetected) {
        issues.push("No improvement detected in evolution validation");
        recommendations.push("Review evolution stages and metrics");
    }
    
    const meetsStandards = issues.length === 0;
    
    return {
        meetsStandards,
        issues,
        recommendations
    };
}