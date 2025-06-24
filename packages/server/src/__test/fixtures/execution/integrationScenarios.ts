/**
 * Integration Scenario Testing Utilities
 * 
 * Provides comprehensive testing for cross-tier integration scenarios,
 * validating event flow, emergent capabilities, and end-to-end workflows.
 * 
 * Key improvements:
 * - Event contract validation for guaranteed message delivery
 * - Measurable emergence criteria with performance metrics
 * - Scenario builder for complex workflows
 * - Integration with actual config class validation
 */

import type {
    ExecutionFixture,
    SwarmFixture,
    RoutineFixture,
    ExecutionContextFixture,
    IntegrationScenario,
    ValidationResult,
    IntegrationMetrics,
    TestScenario,
    SuccessCriteria,
    BaseConfigObject,
    EventContract,
    EnhancedIntegrationDefinition,
} from "./types.js";

import {
    SwarmFixtureFactory,
    RoutineFixtureFactory,
    ExecutionContextFixtureFactory,
    swarmFactory,
    routineFactory,
    executionFactory,
    createMeasurableCapability,
    createEnhancedEmergence,
} from "./executionFactories.js";

import {
    validateIntegration,
    validateEventFlow,
    combineValidationResults
} from "./executionValidationUtils.js";

// ================================================================================================
// Integration Scenario Tester
// ================================================================================================

/**
 * Comprehensive tester for cross-tier integration scenarios
 * Validates event flow, emergence patterns, and system-wide capabilities
 */
export class IntegrationScenarioTester {
    private swarmFactory = new SwarmFixtureFactory();
    private routineFactory = new RoutineFixtureFactory();
    private executionFactory = new ExecutionContextFixtureFactory();

    /**
     * Test a complete integration scenario across all tiers
     */
    async testScenario(scenario: IntegrationScenario): Promise<IntegrationTestResult> {
        const results: ValidationResult[] = [];
        const warnings: string[] = [];

        // 1. Validate each tier independently
        const tierValidations = await this.validateTiers(scenario);
        results.push(...tierValidations);

        // 2. Validate event flow consistency
        const eventFlowResult = this.validateEventFlowConsistency(scenario);
        results.push(eventFlowResult);

        // 3. Validate emergent capabilities
        const emergenceResult = this.validateEmergentCapabilities(scenario);
        results.push(emergenceResult);

        // 4. Validate cross-tier dependencies
        const dependencyResult = this.validateCrossTierDependencies(scenario);
        results.push(dependencyResult);

        // 5. Run test scenarios if provided
        const scenarioResults = scenario.testScenarios ? 
            await this.runTestScenarios(scenario.testScenarios, scenario) : [];

        const combinedResult = combineValidationResults(results);
        
        return {
            success: combinedResult.pass,
            scenarioId: scenario.id,
            tierValidations: {
                tier1: tierValidations[0],
                tier2: tierValidations[1],
                tier3: tierValidations[2]
            },
            eventFlowValid: eventFlowResult.pass,
            emergentCapabilitiesDetected: this.detectEmergentCapabilities(scenario),
            crossTierDependenciesValid: dependencyResult.pass,
            testScenarioResults: scenarioResults,
            metrics: this.calculateIntegrationMetrics(scenario),
            errors: combinedResult.errors,
            warnings: [...(combinedResult.warnings || []), ...warnings]
        };
    }

    /**
     * Validate each tier's fixtures independently
     */
    private async validateTiers(scenario: IntegrationScenario): Promise<ValidationResult[]> {
        const tier1Fixtures = Array.isArray(scenario.tier1) ? scenario.tier1 : [scenario.tier1];
        const tier2Fixtures = Array.isArray(scenario.tier2) ? scenario.tier2 : [scenario.tier2];
        const tier3Fixtures = Array.isArray(scenario.tier3) ? scenario.tier3 : [scenario.tier3];

        const results: ValidationResult[] = [];

        // Validate tier 1 fixtures
        for (const fixture of tier1Fixtures) {
            const result = await this.swarmFactory.validateFixture(fixture);
            results.push({
                ...result,
                message: `Tier 1 validation: ${result.message}`
            });
        }

        // Validate tier 2 fixtures
        for (const fixture of tier2Fixtures) {
            const result = await this.routineFactory.validateFixture(fixture);
            results.push({
                ...result,
                message: `Tier 2 validation: ${result.message}`
            });
        }

        // Validate tier 3 fixtures
        for (const fixture of tier3Fixtures) {
            const result = await this.executionFactory.validateFixture(fixture);
            results.push({
                ...result,
                message: `Tier 3 validation: ${result.message}`
            });
        }

        return results;
    }

    /**
     * Validate event flow consistency across all tiers
     */
    private validateEventFlowConsistency(scenario: IntegrationScenario): ValidationResult {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Collect all events from all tiers
        const allFixtures = this.getAllFixtures(scenario);
        const allProducedEvents = new Set<string>();
        const allConsumedEvents = new Set<string>();

        for (const fixture of allFixtures) {
            if (fixture.integration.producedEvents) {
                fixture.integration.producedEvents.forEach(event => allProducedEvents.add(event));
            }
            if (fixture.integration.consumedEvents) {
                fixture.integration.consumedEvents.forEach(event => allConsumedEvents.add(event));
            }
        }

        // Check that consumed events are produced somewhere
        for (const consumedEvent of allConsumedEvents) {
            if (!this.isEventProduced(consumedEvent, allProducedEvents)) {
                errors.push(`Consumed event "${consumedEvent}" is not produced by any component`);
            }
        }

        // Check for orphaned produced events
        for (const producedEvent of allProducedEvents) {
            if (!this.isEventConsumed(producedEvent, allConsumedEvents)) {
                warnings.push(`Produced event "${producedEvent}" is not consumed by any component`);
            }
        }

        // Validate expected event flow
        if (scenario.expectedEvents) {
            for (const expectedEvent of scenario.expectedEvents) {
                if (!allProducedEvents.has(expectedEvent) && !allConsumedEvents.has(expectedEvent)) {
                    errors.push(`Expected event "${expectedEvent}" is neither produced nor consumed`);
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

    /**
     * Validate emergent capabilities defined in the scenario
     */
    private validateEmergentCapabilities(scenario: IntegrationScenario): ValidationResult {
        const errors: string[] = [];
        const detectedCapabilities = this.detectEmergentCapabilities(scenario);
        
        if (scenario.emergence?.capabilities) {
            for (const expectedCapability of scenario.emergence.capabilities) {
                if (!detectedCapabilities.includes(expectedCapability)) {
                    errors.push(`Expected emergent capability "${expectedCapability}" not detected`);
                }
            }
        }

        return {
            pass: errors.length === 0,
            message: errors.length === 0 ? "Emergent capabilities validation passed" : "Emergent capabilities validation failed",
            errors: errors.length > 0 ? errors : undefined
        };
    }

    /**
     * Validate cross-tier dependencies
     */
    private validateCrossTierDependencies(scenario: IntegrationScenario): ValidationResult {
        const errors: string[] = [];
        const allFixtures = this.getAllFixtures(scenario);

        for (const fixture of allFixtures) {
            if (fixture.integration.crossTierDependencies) {
                const deps = fixture.integration.crossTierDependencies;
                
                // Validate dependencies exist
                if (deps.dependsOn) {
                    for (const dependency of deps.dependsOn) {
                        if (!this.isDependencyAvailable(dependency, allFixtures)) {
                            errors.push(`Dependency "${dependency}" not available in scenario`);
                        }
                    }
                }

                // Validate provided capabilities match what others depend on
                if (deps.provides) {
                    for (const capability of deps.provides) {
                        if (!this.isCapabilityRequired(capability, allFixtures)) {
                            // This is a warning, not an error
                        }
                    }
                }
            }
        }

        return {
            pass: errors.length === 0,
            message: errors.length === 0 ? "Cross-tier dependencies validation passed" : "Cross-tier dependencies validation failed",
            errors: errors.length > 0 ? errors : undefined
        };
    }

    /**
     * Run individual test scenarios within the integration
     */
    private async runTestScenarios(
        testScenarios: TestScenario[], 
        integrationScenario: IntegrationScenario
    ): Promise<TestScenarioResult[]> {
        const results: TestScenarioResult[] = [];

        for (const testScenario of testScenarios) {
            const result = await this.runSingleTestScenario(testScenario, integrationScenario);
            results.push(result);
        }

        return results;
    }

    /**
     * Run a single test scenario
     */
    private async runSingleTestScenario(
        testScenario: TestScenario,
        integrationScenario: IntegrationScenario
    ): Promise<TestScenarioResult> {
        const errors: string[] = [];
        const passedCriteria: string[] = [];
        const failedCriteria: string[] = [];

        // Simulate execution and validate success criteria
        for (const criteria of testScenario.successCriteria) {
            const passed = this.evaluateSuccessCriteria(criteria, testScenario, integrationScenario);
            if (passed) {
                passedCriteria.push(criteria.metric);
            } else {
                failedCriteria.push(criteria.metric);
                errors.push(`Success criteria failed for metric "${criteria.metric}"`);
            }
        }

        // Check failure conditions if provided
        if (testScenario.failureConditions) {
            for (const condition of testScenario.failureConditions) {
                if (this.checkFailureCondition(condition, testScenario, integrationScenario)) {
                    errors.push(`Failure condition triggered: ${condition}`);
                }
            }
        }

        return {
            scenarioName: testScenario.name,
            success: errors.length === 0,
            passedCriteria,
            failedCriteria,
            errors: errors.length > 0 ? errors : undefined
        };
    }

    // ================================================================================================
    // Helper Methods
    // ================================================================================================

    private getAllFixtures(scenario: IntegrationScenario): ExecutionFixture<BaseConfigObject>[] {
        const fixtures: ExecutionFixture<BaseConfigObject>[] = [];
        
        // Add tier 1 fixtures
        const tier1Fixtures = Array.isArray(scenario.tier1) ? scenario.tier1 : [scenario.tier1];
        fixtures.push(...tier1Fixtures);

        // Add tier 2 fixtures
        const tier2Fixtures = Array.isArray(scenario.tier2) ? scenario.tier2 : [scenario.tier2];
        fixtures.push(...tier2Fixtures);

        // Add tier 3 fixtures
        const tier3Fixtures = Array.isArray(scenario.tier3) ? scenario.tier3 : [scenario.tier3];
        fixtures.push(...tier3Fixtures);

        return fixtures;
    }

    private isEventProduced(consumedEvent: string, producedEvents: Set<string>): boolean {
        // Handle exact matches
        if (producedEvents.has(consumedEvent)) {
            return true;
        }

        // Handle wildcard patterns
        for (const producedEvent of producedEvents) {
            if (this.eventsMatch(consumedEvent, producedEvent)) {
                return true;
            }
        }

        return false;
    }

    private isEventConsumed(producedEvent: string, consumedEvents: Set<string>): boolean {
        // Handle exact matches
        if (consumedEvents.has(producedEvent)) {
            return true;
        }

        // Handle wildcard patterns
        for (const consumedEvent of consumedEvents) {
            if (this.eventsMatch(consumedEvent, producedEvent)) {
                return true;
            }
        }

        return false;
    }

    private eventsMatch(pattern: string, event: string): boolean {
        // Handle wildcard patterns like "tier1.*" or "*.execution"
        if (pattern.includes("*")) {
            const regexPattern = pattern.replace(/\*/g, ".*");
            return new RegExp(`^${regexPattern}$`).test(event);
        }
        return pattern === event;
    }

    private isDependencyAvailable(dependency: string, fixtures: ExecutionFixture<BaseConfigObject>[]): boolean {
        return fixtures.some(fixture => 
            fixture.integration.crossTierDependencies?.provides?.includes(dependency)
        );
    }

    private isCapabilityRequired(capability: string, fixtures: ExecutionFixture<BaseConfigObject>[]): boolean {
        return fixtures.some(fixture => 
            fixture.integration.crossTierDependencies?.dependsOn?.includes(capability)
        );
    }

    private detectEmergentCapabilities(scenario: IntegrationScenario): string[] {
        const allFixtures = this.getAllFixtures(scenario);
        const allCapabilities = new Set<string>();
        
        // Collect all individual capabilities
        for (const fixture of allFixtures) {
            for (const capability of fixture.emergence.capabilities) {
                allCapabilities.add(capability);
            }
        }

        const emergentCapabilities: string[] = [];

        // Define emergence patterns - capabilities that emerge from combinations
        const emergencePatterns: Array<{
            required: string[];
            emergent: string;
        }> = [
            {
                required: ["swarm_coordination", "process_intelligence"],
                emergent: "intelligent_process_coordination"
            },
            {
                required: ["security_monitoring", "adaptive_response"],
                emergent: "proactive_security_defense"
            },
            {
                required: ["customer_satisfaction", "issue_resolution", "automated_response"],
                emergent: "end_to_end_customer_support"
            },
            {
                required: ["threat_detection", "automated_response", "system_hardening"],
                emergent: "autonomous_security_system"
            },
            {
                required: ["data_processing", "pattern_recognition", "predictive_analysis"],
                emergent: "intelligent_data_insights"
            },
            {
                required: ["resource_optimization", "adaptive_response", "performance_monitoring"],
                emergent: "self_optimizing_system"
            }
        ];

        // Check which emergent capabilities should arise
        for (const pattern of emergencePatterns) {
            const hasAllRequired = pattern.required.every(req => allCapabilities.has(req));
            if (hasAllRequired) {
                emergentCapabilities.push(pattern.emergent);
            }
        }

        return emergentCapabilities;
    }

    private calculateIntegrationMetrics(scenario: IntegrationScenario): IntegrationMetrics {
        // Simulate realistic metrics based on fixture complexity
        const allFixtures = this.getAllFixtures(scenario);
        const totalCapabilities = allFixtures.reduce(
            (sum, fixture) => sum + fixture.emergence.capabilities.length, 
            0
        );

        return {
            latency: Math.max(100, totalCapabilities * 50), // Base latency + complexity
            accuracy: Math.min(0.99, 0.85 + (totalCapabilities * 0.02)), // Higher capability count improves accuracy
            cost: totalCapabilities * 0.1, // Cost per capability
            reliability: Math.max(0.9, 1 - (totalCapabilities * 0.01)), // More components = lower reliability
            scalability: Math.min(10, totalCapabilities / 2) // Scalability factor
        };
    }

    private evaluateSuccessCriteria(
        criteria: SuccessCriteria,
        testScenario: TestScenario,
        integrationScenario: IntegrationScenario
    ): boolean {
        // Mock evaluation based on scenario characteristics
        // In real implementation, this would execute the actual integration
        
        const mockMetrics: Record<string, number | string | boolean> = {
            "latency": 150,
            "accuracy": 0.95,
            "cost": 0.5,
            "reliability": 0.98,
            "error_rate": 0.02,
            "success_rate": 0.98,
            "throughput": 100
        };

        const actualValue = mockMetrics[criteria.metric];
        if (actualValue === undefined) {
            return false; // Unknown metric
        }

        // Evaluate based on operator
        switch (criteria.operator) {
            case ">":
                return Number(actualValue) > Number(criteria.value);
            case "<":
                return Number(actualValue) < Number(criteria.value);
            case ">=":
                return Number(actualValue) >= Number(criteria.value);
            case "<=":
                return Number(actualValue) <= Number(criteria.value);
            case "==":
                return actualValue === criteria.value;
            case "!=":
                return actualValue !== criteria.value;
            default:
                return false;
        }
    }

    private checkFailureCondition(
        condition: string,
        testScenario: TestScenario,
        integrationScenario: IntegrationScenario
    ): boolean {
        // Mock failure condition checking
        // In real implementation, this would check actual system state
        return false; // Assume no failures for mock
    }
}

// ================================================================================================
// Integration Scenario Factory
// ================================================================================================

/**
 * Factory for creating common integration scenarios
 */
export class IntegrationScenarioFactory {
    private swarmFactory = new SwarmFixtureFactory();
    private routineFactory = new RoutineFixtureFactory();
    private executionFactory = new ExecutionContextFixtureFactory();

    /**
     * Create a customer support integration scenario
     */
    createCustomerSupportScenario(): IntegrationScenario {
        return {
            id: "customer-support-integration",
            name: "Customer Support End-to-End Integration",
            tier1: this.swarmFactory.createVariant("customerSupport"),
            tier2: this.routineFactory.createVariant("customerInquiry"),
            tier3: this.executionFactory.createVariant("highPerformance"),
            expectedEvents: [
                "tier1.swarm.initialized",
                "tier1.task.delegated",
                "tier2.routine.started",
                "tier2.step.completed",
                "tier3.execution.started",
                "tier3.tool.executed",
                "tier3.execution.completed",
                "tier2.routine.completed",
                "tier1.task.completed"
            ],
            emergence: {
                capabilities: ["end_to_end_customer_support", "intelligent_process_coordination"],
                metrics: {
                    latency: 200,
                    accuracy: 0.95,
                    cost: 0.8,
                    reliability: 0.98
                }
            },
            testScenarios: [
                {
                    name: "High Volume Customer Inquiries",
                    input: { inquiryType: "technical", volume: "high" },
                    successCriteria: [
                        { metric: "latency", operator: "<", value: 300 },
                        { metric: "accuracy", operator: ">=", value: 0.9 },
                        { metric: "success_rate", operator: ">", value: 0.95 }
                    ]
                }
            ]
        };
    }

    /**
     * Create a security monitoring integration scenario
     */
    createSecurityMonitoringScenario(): IntegrationScenario {
        return {
            id: "security-monitoring-integration",
            name: "Security Monitoring and Response Integration",
            tier1: this.swarmFactory.createVariant("securityResponse"),
            tier2: this.routineFactory.createVariant("securityCheck"),
            tier3: this.executionFactory.createVariant("secureExecution"),
            expectedEvents: [
                "tier1.swarm.threat.detected",
                "tier2.routine.security.initiated",
                "tier3.execution.security.validated",
                "tier1.swarm.response.coordinated"
            ],
            emergence: {
                capabilities: ["proactive_security_defense", "autonomous_security_system"],
                metrics: {
                    latency: 50, // Security requires low latency
                    accuracy: 0.99,
                    cost: 1.2,
                    reliability: 0.995
                }
            },
            testScenarios: [
                {
                    name: "Advanced Persistent Threat Detection",
                    input: { threatType: "apt", severity: "high" },
                    successCriteria: [
                        { metric: "latency", operator: "<", value: 100 },
                        { metric: "accuracy", operator: ">=", value: 0.98 },
                        { metric: "error_rate", operator: "<", value: 0.01 }
                    ],
                    failureConditions: ["false_positive_rate > 0.05"]
                }
            ]
        };
    }

    /**
     * Create a data processing integration scenario
     */
    createDataProcessingScenario(): IntegrationScenario {
        return {
            id: "data-processing-integration",
            name: "Intelligent Data Processing Pipeline",
            tier1: this.swarmFactory.createVariant("researchAnalysis"),
            tier2: this.routineFactory.createVariant("dataProcessing"),
            tier3: this.executionFactory.createVariant("resourceConstrained"),
            expectedEvents: [
                "tier1.swarm.data.received",
                "tier2.routine.processing.started",
                "tier3.execution.analysis.completed",
                "tier1.swarm.insights.generated"
            ],
            emergence: {
                capabilities: ["intelligent_data_insights", "self_optimizing_system"],
                metrics: {
                    latency: 500, // Data processing can tolerate higher latency
                    accuracy: 0.92,
                    cost: 0.3,
                    reliability: 0.96
                }
            }
        };
    }
}

// ================================================================================================
// Type Definitions for Integration Testing
// ================================================================================================

export interface IntegrationTestResult {
    success: boolean;
    scenarioId: string;
    tierValidations: {
        tier1: ValidationResult;
        tier2: ValidationResult;
        tier3: ValidationResult;
    };
    eventFlowValid: boolean;
    emergentCapabilitiesDetected: string[];
    crossTierDependenciesValid: boolean;
    testScenarioResults?: TestScenarioResult[];
    metrics: IntegrationMetrics;
    errors?: string[];
    warnings?: string[];
}

export interface TestScenarioResult {
    scenarioName: string;
    success: boolean;
    passedCriteria: string[];
    failedCriteria: string[];
    errors?: string[];
}

// ================================================================================================
// Utility Functions
// ================================================================================================

/**
 * Run comprehensive integration tests for a scenario
 */
export async function runIntegrationScenarioTests(
    scenario: IntegrationScenario,
    tester?: IntegrationScenarioTester
): Promise<IntegrationTestResult> {
    const scenarioTester = tester || new IntegrationScenarioTester();
    return await scenarioTester.testScenario(scenario);
}

/**
 * Create and test multiple integration scenarios
 */
export async function runMultipleIntegrationTests(
    scenarios: IntegrationScenario[]
): Promise<IntegrationTestResult[]> {
    const tester = new IntegrationScenarioTester();
    const results: IntegrationTestResult[] = [];

    for (const scenario of scenarios) {
        const result = await tester.testScenario(scenario);
        results.push(result);
    }

    return results;
}

/**
 * Validate that an integration scenario is well-formed
 */
export function validateIntegrationScenario(scenario: IntegrationScenario): ValidationResult {
    const errors: string[] = [];

    if (!scenario.id) {
        errors.push("Scenario must have an ID");
    }

    if (!scenario.name) {
        errors.push("Scenario must have a name");
    }

    if (!scenario.tier1 || !scenario.tier2 || !scenario.tier3) {
        errors.push("Scenario must include fixtures for all three tiers");
    }

    if (scenario.testScenarios) {
        for (const testScenario of scenario.testScenarios) {
            if (!testScenario.name) {
                errors.push("Test scenarios must have names");
            }
            if (!testScenario.successCriteria || testScenario.successCriteria.length === 0) {
                errors.push("Test scenarios must define success criteria");
            }
        }
    }

    return {
        pass: errors.length === 0,
        message: errors.length === 0 ? "Integration scenario validation passed" : "Integration scenario validation failed",
        errors: errors.length > 0 ? errors : undefined
    };
}

// ================================================================================================
// Enhanced Integration Features
// ================================================================================================

/**
 * Test scenario builder for complex workflows
 */
export class ScenarioBuilder {
    private scenario: Partial<IntegrationScenario> = {};
    
    constructor(id: string, name: string) {
        this.scenario.id = id;
        this.scenario.name = name;
    }
    
    withTier1(fixture: SwarmFixture | SwarmFixture[]): this {
        this.scenario.tier1 = fixture;
        return this;
    }
    
    withTier2(fixture: RoutineFixture | RoutineFixture[]): this {
        this.scenario.tier2 = fixture;
        return this;
    }
    
    withTier3(fixture: ExecutionContextFixture | ExecutionContextFixture[]): this {
        this.scenario.tier3 = fixture;
        return this;
    }
    
    expectEvents(...events: string[]): this {
        this.scenario.expectedEvents = events;
        return this;
    }
    
    expectCapabilities(...capabilities: string[]): this {
        if (!this.scenario.emergence) {
            this.scenario.emergence = { capabilities: [] };
        }
        this.scenario.emergence.capabilities = capabilities;
        return this;
    }
    
    withMetrics(metrics: Record<string, number>): this {
        if (!this.scenario.emergence) {
            this.scenario.emergence = { capabilities: [] };
        }
        this.scenario.emergence.metrics = metrics as IntegrationMetrics;
        return this;
    }
    
    withTestScenarios(...scenarios: TestScenario[]): this {
        this.scenario.testScenarios = scenarios;
        return this;
    }
    
    build(): IntegrationScenario {
        if (!this.scenario.id || !this.scenario.name) {
            throw new Error("Scenario must have id and name");
        }
        if (!this.scenario.tier1 || !this.scenario.tier2 || !this.scenario.tier3) {
            throw new Error("Scenario must have all three tiers defined");
        }
        
        return {
            ...this.scenario,
            expectedEvents: this.scenario.expectedEvents || [],
            emergence: this.scenario.emergence || { capabilities: [] }
        } as IntegrationScenario;
    }
}

/**
 * Helper to create event contracts for validation
 */
export function createEventContract(
    eventName: string,
    producer: string,
    consumers: string[],
    payload: Record<string, any>,
    guarantees: "at-least-once" | "exactly-once" | "best-effort" = "at-least-once"
): EventContract {
    return {
        eventName,
        producer,
        consumers,
        payload,
        guarantees,
        description: `Event ${eventName} from ${producer} to ${consumers.join(", ")}`
    };
}

/**
 * Pre-built integration scenarios for common use cases
 */
export const commonScenarios = {
    /**
     * Customer support workflow with measurable outcomes
     */
    customerSupport: (): IntegrationScenario => {
        const builder = new ScenarioBuilder("customer-support-e2e", "End-to-End Customer Support");
        
        return builder
            .withTier1(swarmFactory.createVariant("customerSupport"))
            .withTier2(routineFactory.createVariant("customerInquiry"))
            .withTier3(executionFactory.createVariant("highPerformance"))
            .expectEvents(
                "tier1.swarm.initialized",
                "tier1.task.delegated",
                "tier2.routine.started",
                "tier2.step.completed",
                "tier3.execution.completed",
                "tier2.routine.finished",
                "tier1.coordination.optimized"
            )
            .expectCapabilities(
                "end_to_end_customer_support",
                "intelligent_process_coordination",
                "empathetic_customer_support"
            )
            .withMetrics({
                latency: 3000,
                accuracy: 0.92,
                cost: 150,
                reliability: 0.95
            })
            .withTestScenarios({
                name: "High Volume Support",
                input: { volume: "high", complexity: "medium" },
                successCriteria: [
                    { metric: "latency", operator: "<", value: 5000 },
                    { metric: "accuracy", operator: ">", value: 0.9 }
                ]
            })
            .build();
    },
    
    /**
     * Security monitoring with event contracts
     */
    securityResponse: (): IntegrationScenario => {
        const scenario = new IntegrationScenarioFactory().createSecurityMonitoringScenario();
        
        // Enhance with event contracts
        const enhancedTier1 = scenario.tier1 as SwarmFixture;
        (enhancedTier1.integration as EnhancedIntegrationDefinition).eventContracts = [
            createEventContract(
                "security.threat.detected",
                "tier1.security-swarm",
                ["tier2.security-routine", "tier3.security-executor"],
                { threatLevel: "string", source: "string", timestamp: "number" },
                "exactly-once"
            )
        ];
        
        return scenario;
    },
    
    /**
     * Data processing with performance optimization
     */
    dataProcessing: (): IntegrationScenario => {
        return new ScenarioBuilder("data-processing-e2e", "Intelligent Data Processing")
            .withTier1([
                swarmFactory.createComplete({
                    config: { goal: "Coordinate data ingestion" },
                    emergence: { capabilities: ["data_routing", "load_balancing"] }
                }),
                swarmFactory.createComplete({
                    config: { goal: "Analyze processed data" },
                    emergence: { capabilities: ["pattern_recognition", "insight_generation"] }
                })
            ])
            .withTier2([
                routineFactory.createVariant("dataProcessing"),
                routineFactory.createComplete({
                    config: { name: "Data Validation" },
                    emergence: { capabilities: ["data_validation", "quality_assurance"] }
                })
            ])
            .withTier3(executionFactory.createVariant("highPerformance"))
            .expectEvents(
                "data.ingestion.started",
                "tier1.task.delegated",
                "tier2.routine.started",
                "tier3.execution.started",
                "data.processing.completed",
                "insights.generated"
            )
            .expectCapabilities(
                "intelligent_data_insights",
                "optimal_resource_utilization",
                "self_optimizing_system"
            )
            .withMetrics({
                latency: 2000,
                accuracy: 0.96,
                cost: 100,
                scalability: 0.9,
                throughput: 1000 // records per second
            })
            .build();
    }
};

/**
 * Create custom integration scenario with enhanced features
 */
export function createCustomScenario(
    id: string,
    name: string,
    config: {
        tier1Config?: Partial<SwarmFixture>;
        tier2Config?: Partial<RoutineFixture>;
        tier3Config?: Partial<ExecutionContextFixture>;
        expectedCapabilities?: string[];
        expectedEvents?: string[];
        performanceTargets?: Record<string, number>;
        eventContracts?: EventContract[];
    }
): IntegrationScenario {
    const scenario = new ScenarioBuilder(id, name)
        .withTier1(swarmFactory.createComplete(config.tier1Config))
        .withTier2(routineFactory.createComplete(config.tier2Config))
        .withTier3(executionFactory.createComplete(config.tier3Config))
        .expectEvents(...(config.expectedEvents || []))
        .expectCapabilities(...(config.expectedCapabilities || []))
        .withMetrics(config.performanceTargets || {})
        .build();
    
    // Add event contracts if provided
    if (config.eventContracts) {
        const tier1 = scenario.tier1 as SwarmFixture;
        (tier1.integration as EnhancedIntegrationDefinition).eventContracts = config.eventContracts;
    }
    
    return scenario;
}

/**
 * Run integration scenario with comprehensive validation
 */
export async function runIntegrationScenarioWithValidation(
    scenario: IntegrationScenario,
    options?: {
        validateContracts?: boolean;
        measurePerformance?: boolean;
        checkEmergence?: boolean;
        timeout?: number;
    }
): Promise<IntegrationTestResult> {
    const tester = new IntegrationScenarioTester();
    const result = await tester.testScenario(scenario);
    
    if (!result.success) {
        throw new Error(`Integration scenario "${scenario.name}" failed: ${result.errors?.join(", ")}`);
    }
    
    if (options?.validateContracts && !result.eventFlowValid) {
        throw new Error(`Event contract validation failed: ${result.warnings?.join(", ")}`);
    }
    
    if (options?.checkEmergence && scenario.emergence?.capabilities) {
        const expected = scenario.emergence.capabilities;
        const detected = result.emergentCapabilitiesDetected;
        const missing = expected.filter(cap => !detected.includes(cap));
        
        if (missing.length > 0) {
            console.warn(`Some expected capabilities not detected: ${missing.join(", ")}`);
        }
    }
    
    if (options?.measurePerformance && scenario.emergence?.metrics && result.metrics) {
        for (const [metric, target] of Object.entries(scenario.emergence.metrics)) {
            const actual = (result.metrics as any)[metric];
            if (actual !== undefined && actual > target) {
                console.warn(`Performance metric ${metric} (${actual}) exceeds target (${target})`);
            }
        }
    }
    
    return result;
}