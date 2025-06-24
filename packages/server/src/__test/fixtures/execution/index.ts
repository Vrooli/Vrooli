/**
 * 🧪 Execution Architecture Test Fixtures
 * 
 * Comprehensive test fixtures for Vrooli's three-tier AI execution system,
 * demonstrating emergent capabilities, agent collaboration, and self-improving intelligence.
 * 
 * Directory Structure:
 * - tier1-coordination/: Dynamic swarm coordination through AI metacognition
 * - tier2-process/: Universal workflow execution supporting multiple formats
 * - tier3-execution/: Context-aware strategy execution with safety enforcement
 * - emergent-capabilities/: Cross-tier emergent behaviors and self-improvement
 * - integration-scenarios/: Complete system examples showing tier synergy
 * 
 * For detailed documentation, see:
 * - README.md in this directory
 * - docs/architecture/execution/ for architecture documentation
 */

// Core utilities and types (ENHANCED IMPLEMENTATION - Phase 1-4 Improvements)
export * from "./executionValidationUtils.js";
export * from "./executionFactories.js";
export * from "./executionTestUtils.js";
export * from "./integrationScenarios.js";
export * from "./types.js";

// Phase 2: Runtime Integration Testing
export * from "./executionRunner.js";

// Phase 3: Error Scenario Testing
export * from "./errorScenarios.js";

// Phase 4: Performance Benchmarking
export * from "./performanceBenchmarking.js";

// Tier 1: Coordination Intelligence
export * from "./tier1-coordination/index.js";

// Tier 2: Process Intelligence
export * from "./tier2-process/index.js";

// Tier 3: Execution Intelligence
export * from "./tier3-execution/index.js";

// Emergent Capabilities
export * from "./emergent-capabilities/index.js";

// Integration Scenarios
export * from "./integration-scenarios/index.js";

// Shared utilities
export * from "./testIdGenerator.js";

// Re-export key utilities for convenience (ENHANCED IMPLEMENTATION - Phase 1-4)
export { 
    runComprehensiveExecutionTests,
    runEnhancedComprehensiveExecutionTests,
    validateConfigAgainstSchema,
    validateConfigWithSharedFixtures,
    validateConfigCompatibility,
    validateEmergence,
    validateIntegration,
    validateEvolutionPathways,
    validateEventFlow,
    combineValidationResults,
    createMinimalEmergence,
    createMinimalIntegration,
    FixtureCreationUtils,
    CONFIG_INTEGRATION_MAP,
    CONFIG_CLASS_REGISTRY,
    CONFIG_FIXTURE_REGISTRY,
} from "./executionValidationUtils.js";

// Phase 2: Runtime Integration Testing Exports
export {
    ExecutionFixtureRunner,
    createRuntimeTestScenarios,
    validateRuntimeExecution,
} from "./executionRunner.js";

// Phase 3: Error Scenario Testing Exports
export {
    ErrorScenarioRunner,
    createStandardErrorScenarios,
    runErrorScenarioTests,
} from "./errorScenarios.js";

// Phase 4: Performance Benchmarking Exports
export {
    PerformanceBenchmarker,
    runPerformanceBenchmarkTests,
    runEvolutionBenchmarkTests,
} from "./performanceBenchmarking.js";

// Re-export factories for convenience
export {
    SwarmFixtureFactory,
    RoutineFixtureFactory,
    ExecutionContextFixtureFactory,
    swarmFactory,
    routineFactory,
    executionFactory,
    createMeasurableCapability,
    createEnhancedEmergence,
} from "./executionFactories.js";

// Re-export integration scenario utilities
export {
    IntegrationScenarioTester,
    IntegrationScenarioFactory,
    runIntegrationScenarioTests,
    runMultipleIntegrationTests,
    validateIntegrationScenario,
    ScenarioBuilder,
    createEventContract,
    commonScenarios,
    createCustomScenario,
    runIntegrationScenarioWithValidation,
} from "./integrationScenarios.js";

// Re-export key types for convenience
export type {
    ExecutionFixture,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    EmergenceDefinition,
    IntegrationDefinition,
    EnhancedEmergenceDefinition,
    EnhancedIntegrationDefinition,
    MeasurableCapability,
    EventContract,
    ValidationResult,
    ConfigType,
    ExecutionFixtureFactory,
} from "./types.js";

// Additional types from types.ts
export type {
    ValidationDefinition,
    IntegrationScenario,
    ExecutionTier,
    ExecutionStrategy,
    TestMetadata,
    PerformanceBenchmarks,
    SwarmAgent,
    MOISEOrganization,
    EvolutionStage,
    RoutineMetrics,
    ExecutionContext,
    ExecutionConstraints,
    ResourceAllocation,
    SafetyConfiguration,
    ToolConfiguration,
} from "./types.js";

// Integration scenario types
export type {
    IntegrationTestResult,
    TestScenarioResult,
    IntegrationMetrics,
    TestScenario,
    SuccessCriteria,
} from "./integrationScenarios.js";

/**
 * Quick Access to Key Fixtures
 */

// Get all swarm configurations
import { minimalSwarmConfig, completeSwarmConfig, customerSupportSwarmDb } from "./tier1-coordination/swarms/swarmFixtures.js";

export const swarmConfigs = {
    minimal: minimalSwarmConfig,
    complete: completeSwarmConfig,
    customerSupport: customerSupportSwarmDb,
};

// Get all routine fixtures by evolution stage
import { getRoutinesByEvolutionStage } from "./tier2-process/index.js";

export const routinesByStage = getRoutinesByEvolutionStage();

// Get evolution metrics
import { evolutionMetrics } from "./emergent-capabilities/index.js";

export { evolutionMetrics };

// Get integration flows
import { integrationFlows, compoundGrowthMetrics } from "./integration-scenarios/index.js";

export { integrationFlows, compoundGrowthMetrics };

/**
 * Helper function to demonstrate the three-tier architecture in action
 */
export function demonstrateThreeTierFlow(scenario: keyof typeof integrationFlows) {
    const flow = integrationFlows[scenario];
    
    return {
        name: scenario,
        flow: [
            `1. Trigger: ${flow.trigger}`,
            `2. Tier 1 (Coordination): ${flow.tier1Action}`,
            `3. Tier 2 (Process): ${flow.tier2Action}`,
            `4. Tier 3 (Execution): ${flow.tier3Action}`,
            `5. Emergent: ${flow.emergentAction}`,
            `6. Result: ${flow.outcome}`,
        ].join("\n"),
        architecture: {
            tier1: "AI metacognition for dynamic coordination",
            tier2: "Universal workflow execution",
            tier3: "Context-aware strategy execution",
            emergent: "Self-improving capabilities",
        },
    };
}
