/**
 * Example test file showing how to use the enhanced validation utilities
 * for execution architecture fixtures
 */

import { describe, it, expect } from "vitest";
import {
    ValidatedExecutionFixtureFactory,
    runEnhancedComprehensiveTests,
    validateEvolutionPath,
    validateCrossTierDependencies,
} from "./executionValidationUtils.js";
import type { SwarmFixture, RoutineFixture, ExecutionContextFixture } from "./types.js";
import { testIdGenerator } from "./testIdGenerator.js";
import { commonEmergencePatterns, commonIntegrationPatterns } from "./executionTestUtils.js";

describe("Execution Fixture Validation Examples", () => {
    /**
     * Example 1: Creating and validating a swarm fixture
     */
    describe("Swarm Fixture Validation", () => {
        it("should create a valid security swarm fixture", async () => {
            const factory = new ValidatedExecutionFixtureFactory("chat", {
                swarmTask: "Monitor and respond to security threats",
                swarmSubTasks: [
                    { id: "1", task: "Detect anomalies", status: "pending" },
                    { id: "2", task: "Analyze threats", status: "pending" },
                ],
                botSettings: {
                    occupation: "security analyst",
                    persona: { tone: "professional", verbosity: "concise" },
                },
            });

            const securitySwarm = await factory.create(
                {
                    ...commonEmergencePatterns.domainExpertise,
                    capabilities: [
                        "threat_detection",
                        "pattern_recognition",
                        "automated_response",
                        "incident_correlation",
                    ],
                    evolutionPath: "reactive → proactive → predictive → autonomous",
                },
                {
                    ...commonIntegrationPatterns.tier1Coordination,
                    producedEvents: [
                        "tier1.security.threat_detected",
                        "tier1.security.response_initiated",
                        "tier1.security.incident_resolved",
                    ],
                },
                {
                    id: testIdGenerator.next("CHAT"),
                },
            );

            // Validate the fixture
            runEnhancedComprehensiveTests(securitySwarm, "chat", "security-swarm");
        });
    });

    /**
     * Example 2: Creating and validating an evolution path
     */
    describe("Routine Evolution Path Validation", () => {
        it("should validate customer inquiry evolution", async () => {
            const factory = new ValidatedExecutionFixtureFactory("routine", {
                name: "Customer Inquiry Handler",
                description: "Process customer support inquiries",
            });

            // V1: Conversational
            const v1 = await factory.create(
                {
                    capabilities: ["natural_language_understanding", "context_awareness"],
                    evolutionPath: "conversational → reasoning",
                },
                commonIntegrationPatterns.tier2Orchestration,
                {
                    id: testIdGenerator.next("ROUTINE"),
                    executionStrategy: "conversational",
                },
            ) as RoutineFixture;

            v1.evolutionStage = {
                strategy: "conversational",
                version: "1.0.0",
                metrics: {
                    avgDuration: 5000,
                    avgCredits: 10,
                    successRate: 0.75,
                    errorRate: 0.10,
                },
                improvements: ["Added context retention"],
            };

            // V2: Reasoning
            const v2 = await factory.create(
                {
                    capabilities: [
                        "natural_language_understanding",
                        "context_awareness",
                        "structured_reasoning",
                        "solution_generation",
                    ],
                    evolutionPath: "reasoning → deterministic",
                },
                commonIntegrationPatterns.tier2Orchestration,
                {
                    id: testIdGenerator.next("ROUTINE"),
                    executionStrategy: "reasoning",
                },
            ) as RoutineFixture;

            v2.evolutionStage = {
                strategy: "reasoning",
                version: "2.0.0",
                previousVersion: "1.0.0",
                metrics: {
                    avgDuration: 3000,
                    avgCredits: 8,
                    successRate: 0.85,
                    errorRate: 0.05,
                },
                improvements: ["Added chain-of-thought reasoning", "Improved accuracy"],
            };

            // V3: Deterministic
            const v3 = await factory.create(
                {
                    capabilities: [
                        "pattern_matching",
                        "template_response",
                        "fast_execution",
                        "predictable_outcomes",
                    ],
                    evolutionPath: "deterministic → routing",
                },
                commonIntegrationPatterns.tier2Orchestration,
                {
                    id: testIdGenerator.next("ROUTINE"),
                    executionStrategy: "deterministic",
                },
            ) as RoutineFixture;

            v3.evolutionStage = {
                strategy: "deterministic",
                version: "3.0.0",
                previousVersion: "2.0.0",
                metrics: {
                    avgDuration: 500,
                    avgCredits: 2,
                    successRate: 0.95,
                    errorRate: 0.02,
                },
                improvements: ["Optimized common patterns", "Reduced latency by 90%"],
            };

            // Validate the evolution path
            validateEvolutionPath([v1, v2, v3], "customer-inquiry");
        });
    });

    /**
     * Example 3: Cross-tier integration validation
     */
    describe("Cross-Tier Integration Validation", () => {
        it("should validate healthcare compliance scenario", async () => {
            // Tier 1: Compliance Monitoring Swarm
            const swarmFactory = new ValidatedExecutionFixtureFactory("chat", {
                swarmTask: "Monitor HIPAA compliance across healthcare workflows",
            });

            const complianceSwarm = await swarmFactory.create(
                {
                    capabilities: ["compliance_monitoring", "audit_trail", "alert_generation"],
                    eventPatterns: ["healthcare/*", "data/access/*"],
                },
                {
                    tier: "tier1",
                    producedEvents: ["tier1.compliance.check_requested", "tier1.compliance.alert_raised"],
                    consumedEvents: ["tier2.workflow.data_accessed", "tier3.execution.sensitive_operation"],
                    sharedResources: ["compliance_rules", "audit_log", "event_bus"],
                },
            );

            // Tier 2: Patient Data Access Routine
            const routineFactory = new ValidatedExecutionFixtureFactory("routine", {
                name: "Patient Data Access",
                description: "Secure patient data retrieval with compliance checks",
            });

            const dataRoutine = await routineFactory.create(
                {
                    capabilities: ["data_retrieval", "access_control", "encryption"],
                },
                {
                    tier: "tier2",
                    producedEvents: ["tier2.workflow.data_accessed", "tier2.workflow.access_logged"],
                    consumedEvents: ["tier1.compliance.check_requested", "tier3.security.encryption_completed"],
                    sharedResources: ["patient_database", "access_log", "event_bus"],
                },
            );

            // Tier 3: Secure Execution Context
            const contextFactory = new ValidatedExecutionFixtureFactory("run", {
                routineId: testIdGenerator.next("ROUTINE"),
            });

            const secureContext = await contextFactory.create(
                {
                    capabilities: ["encryption", "data_masking", "secure_transmission"],
                },
                {
                    tier: "tier3",
                    producedEvents: ["tier3.execution.sensitive_operation", "tier3.security.encryption_completed"],
                    consumedEvents: ["tier2.workflow.execute_step"],
                    sharedResources: ["encryption_keys", "security_monitors", "event_bus"],
                },
            );

            // Validate cross-tier dependencies
            validateCrossTierDependencies([
                { fixture: complianceSwarm, tier: "tier1" },
                { fixture: dataRoutine, tier: "tier2" },
                { fixture: secureContext, tier: "tier3" },
            ]);
        });
    });

    /**
     * Example 4: Testing invalid configurations
     */
    describe("Invalid Configuration Detection", () => {
        it("should reject invalid swarm configuration", async () => {
            const factory = new ValidatedExecutionFixtureFactory("chat", {});

            // This should throw because required fields are missing
            await expect(
                factory.create(
                    { capabilities: [] },
                    { tier: "tier1" },
                    { 
                        // Missing required swarmTask
                        id: "invalid",
                    },
                ),
            ).rejects.toThrow(/Invalid config/);
        });

        it("should reject mismatched tier events", () => {
            const fixture = {
                config: { __version: "1.0.0", id: "test" },
                emergence: { capabilities: ["test"] },
                integration: {
                    tier: "tier1" as const,
                    // tier1 producing tier2 events (wrong!)
                    producedEvents: ["tier2.something.happened"],
                },
            };

            // This would fail in validateCrossTierDependencies
            expect(() => {
                validateCrossTierDependencies([{ fixture, tier: "tier1" }]);
            }).toThrow();
        });
    });

    /**
     * Example 5: Batch validation of fixtures
     */
    describe("Batch Fixture Validation", () => {
        it("should validate multiple fixtures efficiently", async () => {
            const fixtures = new Map();

            // Add various fixtures
            fixtures.set("security-swarm", {
                fixture: await createSecuritySwarmFixture(),
                configType: "chat",
            });
            
            fixtures.set("data-routine", {
                fixture: await createDataRoutineFixture(),
                configType: "routine",
            });

            fixtures.set("execution-context", {
                fixture: await createExecutionContextFixture(),
                configType: "run",
            });

            // Validate all at once
            await validateAllExecutionFixtures(fixtures);
        });
    });
});

// Helper functions for creating test fixtures
async function createSecuritySwarmFixture(): Promise<SwarmFixture> {
    const factory = new ValidatedExecutionFixtureFactory("chat", {
        swarmTask: "Security monitoring",
    });
    return factory.createMinimal(
        commonEmergencePatterns.domainExpertise,
        commonIntegrationPatterns.tier1Coordination,
    ) as Promise<SwarmFixture>;
}

async function createDataRoutineFixture(): Promise<RoutineFixture> {
    const factory = new ValidatedExecutionFixtureFactory("routine", {
        name: "Data Processing",
    });
    const fixture = await factory.createMinimal(
        commonEmergencePatterns.selfImprovement,
        commonIntegrationPatterns.tier2Orchestration,
    ) as RoutineFixture;
    
    fixture.evolutionStage = {
        strategy: "reasoning",
        version: "1.0.0",
        metrics: { avgDuration: 1000, avgCredits: 5 },
    };
    
    return fixture;
}

async function createExecutionContextFixture(): Promise<ExecutionContextFixture> {
    const factory = new ValidatedExecutionFixtureFactory("run", {});
    const fixture = await factory.createMinimal(
        commonEmergencePatterns.resilience,
        commonIntegrationPatterns.tier3Execution,
    ) as ExecutionContextFixture;
    
    fixture.strategy = "deterministic";
    fixture.context = {
        tools: ["data_fetch", "data_transform"],
        constraints: { maxTokens: 1000, timeout: 5000 },
    };
    
    return fixture;
}

// Import helper to ensure we have all required imports
import { validateAllExecutionFixtures } from "./executionValidationUtils.js";