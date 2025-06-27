/**
 * Enhanced Validation Utilities for Emergent Capabilities Fixtures
 * 
 * Integrates with shared fixture patterns to provide comprehensive validation
 * for emergent capability configurations and behaviors.
 */

import { type BaseConfigObject } from "@vrooli/shared";
import { configTestUtils } from "@vrooli/shared/__test/fixtures/config";
import { type MockSocketEmitter } from "@vrooli/shared/__test/fixtures/events";
import { type ExecutionEvent } from "@vrooli/shared";
import { type ExtendedAgentConfig, type EmergentSwarmConfig } from "./agent-types/emergentAgentFixtures.js";

/**
 * Core interface for emergent capability fixtures
 */
export interface EmergentCapabilityFixture<TConfig extends BaseConfigObject = BaseConfigObject> {
    // Base configuration (validated against shared schemas)
    config: TConfig;
    
    // Emergent capability definitions
    emergence: EmergenceDefinition;
    
    // Integration patterns
    integration: IntegrationDefinition;
    
    // Learning and evolution
    evolution?: EvolutionDefinition;
    
    // Validation metadata
    validation?: ValidationDefinition;
    
    // Additional metadata
    metadata?: FixtureMetadata;
}

/**
 * Defines what capabilities emerge from configuration
 */
export interface EmergenceDefinition {
    // Capabilities that should emerge
    capabilities: string[];
    
    // Event patterns that trigger emergence
    eventPatterns?: string[];
    
    // Path of capability evolution
    evolutionPath?: string;
    
    // Conditions required for emergence
    emergenceConditions?: {
        minAgents?: number;
        requiredResources?: string[];
        environmentalFactors?: string[];
        timeToEmergence?: string;
    };
    
    // How we measure emergent learning
    learningMetrics?: {
        performanceImprovement: string;
        adaptationTime: string;
        innovationRate: string;
        knowledgeRetention?: string;
    };
    
    // Behavioral expectations
    expectedBehaviors?: {
        patternRecognition?: string[];
        adaptiveResponses?: string[];
        collaborativeBehaviors?: string[];
    };
}

/**
 * Defines integration with execution architecture
 */
export interface IntegrationDefinition {
    // Tier assignment
    tier: "tier1" | "tier2" | "tier3" | "cross-tier";
    
    // Event-driven integration
    producedEvents?: string[];
    consumedEvents?: string[];
    
    // Shared resources
    sharedResources?: string[];
    
    // Cross-tier dependencies
    crossTierDependencies?: {
        dependsOn: string[];
        provides: string[];
    };
    
    // MCP tool usage
    mcpTools?: string[];
    
    // Socket communication patterns
    socketPatterns?: {
        rooms?: string[];
        broadcasts?: string[];
        acknowledgments?: boolean;
    };
}

/**
 * Defines evolution and learning patterns
 */
export interface EvolutionDefinition {
    // Current evolution stage
    currentStage: string;
    
    // Evolution stages with metrics
    stages: EvolutionStage[];
    
    // Triggers for evolution
    evolutionTriggers: string[];
    
    // Success criteria
    successCriteria: {
        metric: string;
        threshold: number;
        comparison: "greater" | "less" | "equal";
    }[];
    
    // Learning rate
    learningRate?: number;
    
    // Knowledge transfer
    knowledgeTransfer?: {
        enabled: boolean;
        transferMethod: "blackboard" | "event" | "direct";
        retentionPeriod?: string;
    };
}

/**
 * Evolution stage with performance metrics
 */
export interface EvolutionStage {
    name: string;
    strategy?: string;
    performanceMetrics: {
        executionTime?: number;
        successRate?: number;
        cost?: number;
        accuracy?: number;
        resourceUsage?: number;
    };
    capabilities?: string[];
    characteristics?: string[];
}

/**
 * Validation metadata for testing
 */
export interface ValidationDefinition {
    // Tests to validate emergence
    emergenceTests: string[];
    
    // Tests to validate integration
    integrationTests: string[];
    
    // Tests to validate evolution
    evolutionTests: string[];
    
    // Performance benchmarks
    benchmarks?: {
        maxLatency?: number;
        minAccuracy?: number;
        maxCost?: number;
        minAvailability?: number;
    };
}

/**
 * Fixture metadata
 */
export interface FixtureMetadata {
    domain: string;
    complexity: "simple" | "medium" | "complex";
    maintainer?: string;
    lastUpdated?: string;
    version?: string;
    tags?: string[];
}

/**
 * Validation result interface
 */
export interface ValidationResult {
    isValid: boolean;
    errors?: string[];
    warnings?: string[];
    data?: any;
}

/**
 * Comprehensive validation result
 */
export interface ComprehensiveValidationResult extends ValidationResult {
    configValidation: ValidationResult;
    emergenceValidation: ValidationResult;
    integrationValidation: ValidationResult;
    evolutionValidation?: ValidationResult;
    overallScore?: number;
}

/**
 * Validate emergent capability fixture configuration
 */
export async function validateEmergentFixture<T extends BaseConfigObject>(
    fixture: EmergentCapabilityFixture<T>,
    ConfigClass: new (config: T) => any,
): Promise<ComprehensiveValidationResult> {
    const results: ComprehensiveValidationResult = {
        isValid: true,
        configValidation: { isValid: false },
        emergenceValidation: { isValid: false },
        integrationValidation: { isValid: false },
    };
    
    // 1. Validate base configuration using shared utilities
    try {
        const configInstance = new ConfigClass(fixture.config);
        results.configValidation = { 
            isValid: true, 
            data: configInstance, 
        };
    } catch (error) {
        results.configValidation = {
            isValid: false,
            errors: [error instanceof Error ? error.message : String(error)],
        };
        results.isValid = false;
    }
    
    // 2. Validate emergence definition
    results.emergenceValidation = validateEmergenceDefinition(fixture.emergence);
    if (!results.emergenceValidation.isValid) {
        results.isValid = false;
    }
    
    // 3. Validate integration definition
    results.integrationValidation = validateIntegrationDefinition(fixture.integration);
    if (!results.integrationValidation.isValid) {
        results.isValid = false;
    }
    
    // 4. Validate evolution if present
    if (fixture.evolution) {
        results.evolutionValidation = validateEvolutionDefinition(fixture.evolution);
        if (!results.evolutionValidation.isValid) {
            results.isValid = false;
        }
    }
    
    // Calculate overall score
    const scores = [
        results.configValidation.isValid ? 1 : 0,
        results.emergenceValidation.isValid ? 1 : 0,
        results.integrationValidation.isValid ? 1 : 0,
        results.evolutionValidation?.isValid ? 1 : 0,
    ].filter(score => score !== undefined);
    
    results.overallScore = scores.reduce((a, b) => a + b, 0) / scores.length;
    
    return results;
}

/**
 * Validate emergence definition
 */
export function validateEmergenceDefinition(emergence: EmergenceDefinition): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Required fields
    if (!emergence.capabilities || emergence.capabilities.length === 0) {
        errors.push("Must define at least one emergent capability");
    }
    
    // Validate event patterns
    if (emergence.eventPatterns) {
        for (const pattern of emergence.eventPatterns) {
            if (!isValidEventPattern(pattern)) {
                errors.push(`Invalid event pattern: ${pattern}`);
            }
        }
    }
    
    // Validate evolution path
    if (emergence.evolutionPath && !isValidEvolutionPath(emergence.evolutionPath)) {
        errors.push(`Invalid evolution path: ${emergence.evolutionPath}`);
    }
    
    // Validate learning metrics
    if (emergence.learningMetrics) {
        const metrics = emergence.learningMetrics;
        if (!metrics.performanceImprovement || !metrics.adaptationTime || !metrics.innovationRate) {
            warnings.push("Learning metrics should include all core metrics");
        }
    }
    
    // Best practice warnings
    if (!emergence.emergenceConditions) {
        warnings.push("Consider defining emergence conditions for clarity");
    }
    
    if (!emergence.expectedBehaviors) {
        warnings.push("Consider defining expected behaviors for better testing");
    }
    
    return {
        isValid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
    };
}

/**
 * Validate integration definition
 */
export function validateIntegrationDefinition(integration: IntegrationDefinition): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Validate tier
    const validTiers = ["tier1", "tier2", "tier3", "cross-tier"];
    if (!validTiers.includes(integration.tier)) {
        errors.push(`Invalid tier: ${integration.tier}`);
    }
    
    // Validate event names
    if (integration.producedEvents) {
        for (const event of integration.producedEvents) {
            if (!isValidEventName(event)) {
                errors.push(`Invalid produced event: ${event}`);
            }
        }
    }
    
    if (integration.consumedEvents) {
        for (const event of integration.consumedEvents) {
            if (!isValidEventName(event)) {
                errors.push(`Invalid consumed event: ${event}`);
            }
        }
    }
    
    // Validate cross-tier dependencies
    if (integration.crossTierDependencies) {
        const deps = integration.crossTierDependencies;
        if ((!deps.dependsOn || deps.dependsOn.length === 0) && 
            (!deps.provides || deps.provides.length === 0)) {
            warnings.push("Cross-tier dependencies should specify what they depend on or provide");
        }
    }
    
    // Best practice warnings
    if (!integration.producedEvents && !integration.consumedEvents) {
        warnings.push("Integration should define event communication patterns");
    }
    
    return {
        isValid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
    };
}

/**
 * Validate evolution definition
 */
export function validateEvolutionDefinition(evolution: EvolutionDefinition): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Validate stages
    if (!evolution.stages || evolution.stages.length < 2) {
        errors.push("Evolution must define at least 2 stages");
    }
    
    // Validate current stage exists in stages
    const stageNames = evolution.stages.map(s => s.name);
    if (!stageNames.includes(evolution.currentStage)) {
        errors.push(`Current stage '${evolution.currentStage}' not found in stages`);
    }
    
    // Validate evolution triggers
    if (!evolution.evolutionTriggers || evolution.evolutionTriggers.length === 0) {
        errors.push("Must define at least one evolution trigger");
    }
    
    // Validate success criteria
    if (!evolution.successCriteria || evolution.successCriteria.length === 0) {
        errors.push("Must define at least one success criterion");
    }
    
    // Validate stage progression
    for (let i = 1; i < evolution.stages.length; i++) {
        const prev = evolution.stages[i - 1];
        const curr = evolution.stages[i];
        
        // Check for improvement indicators
        if (!hasImprovement(prev.performanceMetrics, curr.performanceMetrics)) {
            warnings.push(`No clear improvement from stage '${prev.name}' to '${curr.name}'`);
        }
    }
    
    return {
        isValid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
    };
}

/**
 * Helper to check if metrics show improvement
 */
function hasImprovement(
    prev: EvolutionStage["performanceMetrics"],
    curr: EvolutionStage["performanceMetrics"],
): boolean {
    // Lower execution time is better
    if (prev.executionTime && curr.executionTime && curr.executionTime < prev.executionTime) {
        return true;
    }
    
    // Higher success rate is better
    if (prev.successRate && curr.successRate && curr.successRate > prev.successRate) {
        return true;
    }
    
    // Lower cost is better
    if (prev.cost && curr.cost && curr.cost < prev.cost) {
        return true;
    }
    
    // Higher accuracy is better
    if (prev.accuracy && curr.accuracy && curr.accuracy > prev.accuracy) {
        return true;
    }
    
    return false;
}

/**
 * Validate event pattern (supports wildcards)
 */
export function isValidEventPattern(pattern: string): boolean {
    // Event patterns can use wildcards: "system/*", "ai/medical/*", etc.
    const validPattern = /^[a-z0-9_]+(\/(([a-z0-9_]+|\*))+)?$/;
    return validPattern.test(pattern);
}

/**
 * Validate event name
 */
export function isValidEventName(event: string): boolean {
    // Event names use dot notation: "system.error", "routine.completed"
    const validEvent = /^[a-z0-9_]+(\.[a-z0-9_]+)*$/;
    return validEvent.test(event);
}

/**
 * Validate evolution path
 */
export function isValidEvolutionPath(path: string): boolean {
    // Evolution paths use arrow notation: "reactive → proactive → predictive"
    const validPath = /^[a-z0-9_]+(\s*→\s*[a-z0-9_]+)*$/;
    return validPath.test(path);
}

/**
 * Create a mock socket emitter configured for emergent capability testing
 */
export function createEmergentSocketEmitter(
    fixture: EmergentCapabilityFixture,
    mockEmitter: MockSocketEmitter,
): MockSocketEmitter {
    // Subscribe to consumed events
    if (fixture.integration.consumedEvents) {
        for (const event of fixture.integration.consumedEvents) {
            mockEmitter.on(event, () => {
                // Track event consumption
            });
        }
    }
    
    // Configure socket patterns
    if (fixture.integration.socketPatterns) {
        const patterns = fixture.integration.socketPatterns;
        
        // Join rooms
        if (patterns.rooms) {
            for (const room of patterns.rooms) {
                mockEmitter.joinRoom(room);
            }
        }
    }
    
    return mockEmitter;
}

/**
 * Validate agent configuration
 */
export function validateAgentConfig(agent: ExtendedAgentConfig): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Required fields
    if (!agent.agentId) {
        errors.push("Agent must have an ID");
    }
    
    if (!agent.goal) {
        errors.push("Agent must have a goal");
    }
    
    if (!agent.initialRoutine) {
        errors.push("Agent must have an initial routine");
    }
    
    if (!agent.subscriptions || agent.subscriptions.length === 0) {
        errors.push("Agent must subscribe to at least one event");
    }
    
    // Validate subscriptions
    for (const subscription of agent.subscriptions || []) {
        if (!isValidEventPattern(subscription)) {
            errors.push(`Invalid subscription pattern: ${subscription}`);
        }
    }
    
    // Priority validation
    if (agent.priority !== undefined && (agent.priority < 1 || agent.priority > 10)) {
        errors.push("Priority must be between 1 and 10");
    }
    
    // Best practices
    if (!agent.name) {
        warnings.push("Agent should have a descriptive name");
    }
    
    return {
        isValid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
    };
}

/**
 * Validate swarm configuration
 */
export function validateSwarmConfig(swarm: EmergentSwarmConfig): ValidationResult {
    const errors: string[] = [];
    const warnings: string[] = [];
    
    // Required fields
    if (!swarm.swarmId) {
        errors.push("Swarm must have an ID");
    }
    
    if (!swarm.name) {
        errors.push("Swarm must have a name");
    }
    
    if (!swarm.agents || swarm.agents.length === 0) {
        errors.push("Swarm must have at least one agent");
    }
    
    // Validate each agent
    for (const agent of swarm.agents || []) {
        const agentValidation = validateAgentConfig(agent);
        if (!agentValidation.isValid) {
            errors.push(`Agent '${agent.agentId}' validation failed: ${agentValidation.errors?.join(", ")}`);
        }
    }
    
    // Coordination validation
    if (!swarm.coordination) {
        warnings.push("Swarm should define coordination patterns");
    }
    
    return {
        isValid: errors.length === 0,
        errors: errors.length > 0 ? errors : undefined,
        warnings: warnings.length > 0 ? warnings : undefined,
    };
}

/**
 * Run comprehensive validation tests for an emergent fixture
 */
export function runComprehensiveEmergentTests<T extends BaseConfigObject>(
    fixture: EmergentCapabilityFixture<T>,
    ConfigClass: new (config: T) => any,
    fixtureName: string,
): void {
    describe(`${fixtureName} emergent capability fixture`, () => {
        let validationResult: ComprehensiveValidationResult;
        
        beforeAll(async () => {
            validationResult = await validateEmergentFixture(fixture, ConfigClass);
        });
        
        // Configuration tests
        it("should have valid configuration", () => {
            expect(validationResult.configValidation.isValid).toBe(true);
            if (!validationResult.configValidation.isValid) {
                console.error("Config errors:", validationResult.configValidation.errors);
            }
        });
        
        // Emergence tests
        it("should define valid emergent capabilities", () => {
            expect(validationResult.emergenceValidation.isValid).toBe(true);
            if (!validationResult.emergenceValidation.isValid) {
                console.error("Emergence errors:", validationResult.emergenceValidation.errors);
            }
        });
        
        it("should have at least one capability", () => {
            expect(fixture.emergence.capabilities.length).toBeGreaterThan(0);
        });
        
        // Integration tests
        it("should define valid integration patterns", () => {
            expect(validationResult.integrationValidation.isValid).toBe(true);
            if (!validationResult.integrationValidation.isValid) {
                console.error("Integration errors:", validationResult.integrationValidation.errors);
            }
        });
        
        it("should specify tier assignment", () => {
            expect(["tier1", "tier2", "tier3", "cross-tier"]).toContain(fixture.integration.tier);
        });
        
        // Evolution tests (if applicable)
        if (fixture.evolution) {
            it("should define valid evolution pathways", () => {
                expect(validationResult.evolutionValidation?.isValid).toBe(true);
                if (!validationResult.evolutionValidation?.isValid) {
                    console.error("Evolution errors:", validationResult.evolutionValidation.errors);
                }
            });
            
            it("should show improvement across evolution stages", () => {
                const stages = fixture.evolution.stages;
                for (let i = 1; i < stages.length; i++) {
                    const improved = hasImprovement(
                        stages[i - 1].performanceMetrics,
                        stages[i].performanceMetrics,
                    );
                    expect(improved).toBe(true);
                }
            });
        }
        
        // Event flow tests
        it("should have consistent event patterns", () => {
            const consumed = fixture.integration.consumedEvents || [];
            const produced = fixture.integration.producedEvents || [];
            
            // Check for event pattern validity
            for (const event of [...consumed, ...produced]) {
                expect(isValidEventName(event)).toBe(true);
            }
        });
        
        // Best practices tests
        it("should follow emergent capability best practices", () => {
            // Should define learning metrics for emergent systems
            if (!fixture.emergence.learningMetrics) {
                console.warn(`${fixtureName}: Consider defining learning metrics`);
            }
            
            // Should define expected behaviors
            if (!fixture.emergence.expectedBehaviors) {
                console.warn(`${fixtureName}: Consider defining expected behaviors`);
            }
            
            // Cross-tier fixtures should define dependencies
            if (fixture.integration.tier === "cross-tier" && !fixture.integration.crossTierDependencies) {
                console.warn(`${fixtureName}: Cross-tier fixtures should define dependencies`);
            }
        });
        
        // Overall validation score
        it("should have high overall validation score", () => {
            expect(validationResult.overallScore).toBeGreaterThanOrEqual(0.8);
        });
    });
}

/**
 * Test helper to simulate emergence over time
 */
export async function simulateEmergence(
    fixture: EmergentCapabilityFixture,
    mockEmitter: MockSocketEmitter,
    events: ExecutionEvent[],
    timeSpan = 1000,
): Promise<{
    emergedCapabilities: string[];
    performanceMetrics: Record<string, number>;
    learningProgress: number;
}> {
    const startTime = Date.now();
    const emergedCapabilities: Set<string> = new Set();
    const metrics: Record<string, number[]> = {};
    
    // Emit events over time
    for (const event of events) {
        await mockEmitter.emit(event.type, event.data);
        
        // Simulate capability emergence based on event patterns
        if (fixture.emergence.eventPatterns) {
            for (const pattern of fixture.emergence.eventPatterns) {
                if (matchesEventPattern(event.type, pattern)) {
                    // Gradually emerge capabilities
                    const progress = (Date.now() - startTime) / timeSpan;
                    const capabilityIndex = Math.floor(progress * fixture.emergence.capabilities.length);
                    if (capabilityIndex < fixture.emergence.capabilities.length) {
                        emergedCapabilities.add(fixture.emergence.capabilities[capabilityIndex]);
                    }
                }
            }
        }
        
        // Track metrics
        if (event.data && typeof event.data === "object") {
            for (const [key, value] of Object.entries(event.data)) {
                if (typeof value === "number") {
                    if (!metrics[key]) metrics[key] = [];
                    metrics[key].push(value);
                }
            }
        }
        
        // Simulate time passing
        await new Promise(resolve => setTimeout(resolve, timeSpan / events.length));
    }
    
    // Calculate average metrics
    const avgMetrics: Record<string, number> = {};
    for (const [key, values] of Object.entries(metrics)) {
        avgMetrics[key] = values.reduce((a, b) => a + b, 0) / values.length;
    }
    
    // Calculate learning progress
    const learningProgress = emergedCapabilities.size / fixture.emergence.capabilities.length;
    
    return {
        emergedCapabilities: Array.from(emergedCapabilities),
        performanceMetrics: avgMetrics,
        learningProgress,
    };
}

/**
 * Helper to match event patterns with wildcards
 */
function matchesEventPattern(event: string, pattern: string): boolean {
    const regexPattern = pattern
        .replace(/\*/g, ".*")
        .replace(/\//g, "\\.");
    return new RegExp(`^${regexPattern}$`).test(event);
}

/**
 * Create test scenarios for emergence validation
 */
export function createEmergenceTestScenarios(
    fixture: EmergentCapabilityFixture,
): Array<{
    name: string;
    events: ExecutionEvent[];
    expectedCapabilities: string[];
    expectedMetrics?: Record<string, { min?: number; max?: number }>;
}> {
    const scenarios = [];
    
    // Basic emergence scenario
    scenarios.push({
        name: "Basic capability emergence",
        events: createEventsForPatterns(fixture.emergence.eventPatterns || []),
        expectedCapabilities: fixture.emergence.capabilities.slice(0, 1),
        expectedMetrics: {
            learningProgress: { min: 0.2 },
        },
    });
    
    // Full emergence scenario
    scenarios.push({
        name: "Full capability emergence",
        events: createEventsForPatterns(
            fixture.emergence.eventPatterns || [],
            fixture.emergence.capabilities.length * 10,
        ),
        expectedCapabilities: fixture.emergence.capabilities,
        expectedMetrics: {
            learningProgress: { min: 0.8 },
        },
    });
    
    // Evolution scenario (if applicable)
    if (fixture.evolution) {
        for (const stage of fixture.evolution.stages) {
            scenarios.push({
                name: `Evolution to ${stage.name}`,
                events: createEvolutionEvents(stage),
                expectedCapabilities: stage.capabilities || fixture.emergence.capabilities,
                expectedMetrics: stage.performanceMetrics as any,
            });
        }
    }
    
    return scenarios;
}

/**
 * Helper to create events for testing patterns
 */
function createEventsForPatterns(patterns: string[], count = 10): ExecutionEvent[] {
    const events: ExecutionEvent[] = [];
    
    for (let i = 0; i < count; i++) {
        const pattern = patterns[i % patterns.length] || "test/*";
        const eventType = pattern.replace("*", `event_${i}`);
        
        events.push({
            id: `test_${i}`,
            type: eventType,
            timestamp: new Date(),
            source: "test",
            tier: 1,
            category: "test",
            subcategory: "emergence",
            deliveryGuarantee: "fire-and-forget",
            priority: "medium",
            data: {
                index: i,
                pattern,
                testData: Math.random(),
            },
        });
    }
    
    return events;
}

/**
 * Helper to create evolution trigger events
 */
function createEvolutionEvents(stage: EvolutionStage): ExecutionEvent[] {
    const events: ExecutionEvent[] = [];
    
    // Create events that would trigger evolution to this stage
    events.push({
        id: `evolution_${stage.name}`,
        type: "evolution/trigger",
        timestamp: new Date(),
        source: "test",
        tier: 1,
        category: "evolution",
        subcategory: "stage",
        deliveryGuarantee: "reliable",
        priority: "high",
        data: {
            targetStage: stage.name,
            currentMetrics: stage.performanceMetrics,
            strategy: stage.strategy,
        },
    });
    
    return events;
}
