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
export * from "./emergentCapabilityHelpers.js";
export * from "./executionTestUtils.js";
export * from "./executionValidationUtils.js";
export * from "./types.js";

// Production fixtures with defensive security focus
export * from "./fixtures.js";

// Phase 2: Runtime Integration Testing
export * from "./executionRunner.js";

// Phase 3: Error Scenario Testing
export * from "./errorScenarios.js";

// Phase 4: Performance Benchmarking
export * from "./performanceBenchmarking.js";

// Tier 2: Process Intelligence
export * from "./tier2-process/index.js";

// Emergent Capabilities
export * from "./emergent-capabilities/index.js";

// Shared utilities
export * from "./testIdGenerator.js";

// Re-export key utilities for convenience (ENHANCED IMPLEMENTATION - Phase 1-4)
export {
    combineValidationResults, CONFIG_CLASS_REGISTRY,
    CONFIG_FIXTURE_REGISTRY, CONFIG_INTEGRATION_MAP, createMinimalEmergence,
    createMinimalIntegration,
    FixtureCreationUtils, runComprehensiveExecutionTests,
    runEnhancedComprehensiveExecutionTests,
    validateConfigAgainstSchema, validateConfigCompatibility, validateConfigWithSharedFixtures, validateEmergence, validateEventFlow, validateEvolutionPathways, validateIntegration,
} from "./executionValidationUtils.js";

// Phase 2: Runtime Integration Testing Exports
export {
    createRuntimeTestScenarios, ExecutionFixtureRunner, validateRuntimeExecution,
} from "./executionRunner.js";

// Phase 3: Error Scenario Testing Exports
export {
    createStandardErrorScenarios, ErrorScenarioRunner, runErrorScenarioTests,
} from "./errorScenarios.js";

// Phase 4: Performance Benchmarking Exports
export {
    PerformanceBenchmarker, runEvolutionBenchmarkTests, runPerformanceBenchmarkTests,
} from "./performanceBenchmarking.js";

// Re-export key types for convenience
export type {
    EmergenceDefinition, EnhancedEmergenceDefinition,
    EnhancedIntegrationDefinition, EventContract, ExecutionContextFixture, ExecutionFixture, ExecutionFixtureFactory, IntegrationDefinition, MeasurableCapability, RoutineFixture, SwarmFixture, ValidationResult,
} from "./types.js";

// Additional types from types.ts
export type {
    EvolutionStage, ExecutionConstraints, ExecutionContext, ExecutionTier, IntegrationScenario, MOISEOrganization, PerformanceBenchmarks, ResourceAllocation, RoutineMetrics, SafetyConfiguration, TestMetadata, ToolConfiguration, ValidationDefinition,
} from "./types.js";

/**
 * Quick Access to Key Fixtures
 */

// Production security-focused fixtures
import {
    complianceAuditSwarm,
    complianceMonitoringIntegration,
    dataPrivacyComplianceRoutine,
    executionFixtures,
    highPerformanceAnalyticsContext,
    secureCodeExecutionContext,
    securityMonitoringSwarm,
    securityResponseIntegration,
    vulnerabilityAssessmentRoutine,
} from "./fixtures.js";

export const productionFixtures = {
    // Tier 1: Coordination Intelligence
    securityMonitoringSwarm,
    complianceAuditSwarm,

    // Tier 2: Process Intelligence
    vulnerabilityAssessmentRoutine,
    dataPrivacyComplianceRoutine,

    // Tier 3: Execution Intelligence
    secureCodeExecutionContext,
    highPerformanceAnalyticsContext,

    // Integration Scenarios
    securityResponseIntegration,
    complianceMonitoringIntegration,

    // Complete collection
    all: executionFixtures,
};

// Get all swarm configurations
import { minimalSwarmConfig } from "./tier1-coordination/swarms/swarmFixtures.js";

export const swarmConfigs = {
    minimal: minimalSwarmConfig,
};

// Get all routine fixtures by evolution stage
import { getRoutinesByEvolutionStage } from "./tier2-process/index.js";

export const routinesByStage = getRoutinesByEvolutionStage();

// Get evolution metrics
import { evolutionMetrics } from "./emergent-capabilities/index.js";

export { evolutionMetrics };
