/**
 * Type definitions for execution architecture test fixtures
 * 
 * These types ensure consistency and type safety across all execution fixtures,
 * aligning with the three-tier AI architecture and emergent capabilities philosophy.
 * 
 * This file is built on proven patterns from packages/shared/src/__test/fixtures/
 * and integrates with configTestUtils.ts and integrationUtils.ts for validation.
 */

import type {
    BaseConfigObject,
    ChatConfigObject,
    RunProgress,
    BotConfigObject,
    SwarmSubTask,
    SwarmResource,
    BlackboardItem,
    ChatToolCallRecord,
    SwarmPolicy,
    PendingToolCallEntry,
} from "@vrooli/shared";

/**
 * Base structure for all execution fixtures
 */
export interface ExecutionFixture<TConfig extends BaseConfigObject = BaseConfigObject> {
    /** The data-driven configuration object */
    config: TConfig;
    
    /** Expected emergent behaviors from this configuration */
    emergence: EmergenceDefinition;
    
    /** Integration points with other fixtures/tiers */
    integration: IntegrationDefinition;
    
    /** Validation metadata */
    validation?: ValidationDefinition;
    
    /** Test metadata */
    metadata?: TestMetadata;
}

/**
 * Defines emergent behaviors expected from a fixture
 */
export interface EmergenceDefinition {
    /** Capabilities that emerge from this configuration */
    capabilities: string[];
    
    /** Event patterns that trigger emergent behaviors */
    eventPatterns?: string[];
    
    /** How this fixture can evolve to more efficient forms */
    evolutionPath?: string;
    
    /** Collaboration patterns with other agents/fixtures */
    collaborationPatterns?: string[];
    
    /** Conditions under which capabilities emerge */
    emergenceConditions?: {
        minEvents?: number;
        minAgents?: number;
        requiredResources?: string[];
        timeframe?: number;
        environmentalFactors?: string[];
    };
    
    /** Learning metrics for measuring emergence */
    learningMetrics?: {
        performanceImprovement: string;
        adaptationTime: string;
        innovationRate: string;
    };
}

/**
 * Enhanced emergence definition with measurable capabilities
 */
export interface EnhancedEmergenceDefinition extends EmergenceDefinition {
    /** Measurable capabilities with concrete metrics */
    measurableCapabilities?: MeasurableCapability[];
    
    /** Tests to verify emergence */
    emergenceTests?: {
        setup: string;
        trigger: string;
        expectedOutcome: string;
        measurementMethod: string;
    }[];
}

/**
 * Measurable capability with concrete metrics
 */
export interface MeasurableCapability {
    name: string;
    metric: string;
    baseline: number;
    target: number;
    unit: string;
    description?: string;
    /** How this capability improves over time */
    evolutionFormula?: string;
    /** Minimum events needed to observe improvement */
    minEventsForMeasurement?: number;
}

/**
 * Defines how a fixture integrates with the system
 */
export interface IntegrationDefinition {
    /** Which tier this fixture belongs to */
    tier: ExecutionTier;
    
    /** Other configurations this depends on */
    requiredConfigs?: string[];
    
    /** Events produced during execution */
    producedEvents?: string[];
    
    /** Events this fixture responds to */
    consumedEvents?: string[];
    
    /** Shared resources used/produced */
    sharedResources?: string[];
    
    /** Cross-tier dependencies */
    crossTierDependencies?: {
        tier1?: string[];
        tier2?: string[];
        tier3?: string[];
        dependsOn?: string[];
        provides?: string[];
    };
    
    /** MCP tools used by this component */
    mcpTools?: string[];
}

/**
 * Enhanced integration definition with event contracts
 */
export interface EnhancedIntegrationDefinition extends IntegrationDefinition {
    /** Event contracts for validation */
    eventContracts?: EventContract[];
    
    /** Integration test scenarios */
    integrationTests?: {
        scenario: string;
        events: Array<{
            time: number;
            event: string;
            data: any;
        }>;
        assertions: Array<{
            type: "event" | "state" | "metric";
            check: string;
            expected: any;
        }>;
    }[];
}

/**
 * Event contract for cross-tier communication
 */
export interface EventContract {
    eventName: string;
    producer: string;
    consumers: string[];
    payload: Record<string, any>;
    guarantees: "at-least-once" | "exactly-once" | "best-effort";
    description?: string;
}

/**
 * Validation configuration for fixtures
 */
export interface ValidationDefinition {
    /** Config class for runtime validation */
    configClass?: string;
    
    /** Schema name for API validation */
    schemaName?: string;
    
    /** Custom validation rules */
    customRules?: ValidationRule[];
    
    /** Expected validation results */
    expectedResults?: {
        emergenceScore?: number;
        integrationScore?: number;
        evolutionScore?: number;
    };
}

/**
 * Custom validation rule
 */
export interface ValidationRule {
    name: string;
    description: string;
    validate: (fixture: any) => ValidationResult;
}

/**
 * Validation result (aligned with shared package patterns)
 */
export interface ValidationResult {
    pass: boolean;
    message?: string;
    errors?: string[];
    warnings?: string[];
    details?: Record<string, any>;
}

/**
 * Integration with shared package validation types
 */
export interface SharedValidationResult {
    isValid: boolean;
    data?: any;
    errors?: string[];
}

/**
 * Test metadata for fixtures
 */
export interface TestMetadata {
    /** Unique identifier for this fixture */
    id: string;
    
    /** Human-readable name */
    name: string;
    
    /** Detailed description */
    description?: string;
    
    /** Tags for categorization */
    tags?: string[];
    
    /** Performance benchmarks */
    benchmarks?: PerformanceBenchmarks;
    
    /** Related fixtures */
    relatedFixtures?: string[];
    
    /** Creation/update info */
    created?: string;
    updated?: string;
    author?: string;
}

/**
 * Performance benchmarks for fixtures
 */
export interface PerformanceBenchmarks {
    /** Average execution time in ms */
    avgDuration?: number;
    
    /** Average credit cost */
    avgCredits?: number;
    
    /** Success rate (0-1) */
    successRate?: number;
    
    /** Quality score (0-1) */
    qualityScore?: number;
    
    /** Resource usage */
    resourceUsage?: {
        memory?: number;
        cpu?: number;
        tokens?: number;
    };
}

/**
 * Execution tiers in the architecture
 */
export type ExecutionTier = "tier1" | "tier2" | "tier3" | "cross-tier";

/**
 * Execution strategies
 */
export type ExecutionStrategy = "conversational" | "reasoning" | "deterministic" | "routing";

/**
 * Emergence pattern types - how capabilities emerge from configuration
 */
export interface EmergencePattern {
    /** Pattern name (e.g., "collaborative_learning", "swarm_intelligence") */
    name: string;
    
    /** Minimum configuration requirements for this pattern */
    requires: {
        minAgents?: number;
        minEvents?: number;
        eventTypes?: string[];
        resources?: string[];
        timeWindow?: number;
    };
    
    /** What emerges when pattern is triggered */
    produces: {
        capabilities: string[];
        behaviors: string[];
        optimizations: string[];
    };
    
    /** How to measure emergence */
    metrics: MeasurableCapability[];
}

/**
 * Evolution trigger - what causes a fixture to evolve
 */
export interface EvolutionTrigger {
    /** Trigger type */
    type: "performance" | "error_rate" | "usage_pattern" | "external_event";
    
    /** Threshold for triggering evolution */
    threshold: {
        metric: string;
        operator: ">" | "<" | "=" | ">=" | "<=";
        value: number;
    };
    
    /** What happens when triggered */
    action: {
        targetStrategy: ExecutionStrategy;
        preserveState: boolean;
        migrationSteps: string[];
    };
}

// Tier 1: Coordination Intelligence Types

/**
 * Swarm fixture for Tier 1 coordination
 */
export interface SwarmFixture extends ExecutionFixture<ChatConfigObject> {
    config: ChatConfigObject;
    
    /** Swarm-specific metadata */
    swarmMetadata?: {
        formation: "hierarchical" | "flat" | "matrix" | "dynamic";
        coordinationPattern: "consensus" | "delegation" | "negotiation" | "emergence";
        expectedAgentCount: number;
        minViableAgents: number;
    };
}

/**
 * Agent definition within a swarm
 */
export interface SwarmAgent {
    /** Unique agent identifier */
    id: string;
    
    /** Agent role in the swarm */
    role: string;
    
    /** Agent capabilities */
    capabilities: string[];
    
    /** Responsibilities */
    responsibilities?: string[];
    
    /** Event subscriptions specific to this agent */
    eventSubscriptions?: Record<string, boolean>;
    
    /** Learning configuration */
    learning?: {
        enabled: boolean;
        adaptationRate?: number;
        memoryRetention?: number;
    };
}

/**
 * MOISE+ organization structure
 */
export interface MOISEOrganization {
    /** Organization ID */
    id: string;
    
    /** Structural specification */
    structure: {
        roles: MOISERole[];
        groups: MOISEGroup[];
        links: MOISELink[];
    };
    
    /** Functional specification */
    functional: {
        missions: MOISEMission[];
        goals: MOISEGoal[];
        plans: MOISEPlan[];
    };
    
    /** Normative specification */
    normative: {
        norms: MOISENorm[];
        permissions: MOISEPermission[];
        obligations: MOISEObligation[];
    };
}

export interface MOISERole {
    id: string;
    name: string;
    capabilities: string[];
    cardinality?: { min: number; max: number };
}

export interface MOISEGroup {
    id: string;
    name: string;
    roles: string[];
    subgroups?: string[];
}

export interface MOISELink {
    type: "authority" | "communication" | "acquaintance";
    from: string;
    to: string;
}

export interface MOISEMission {
    id: string;
    name: string;
    goals: string[];
    minAgents?: number;
    maxAgents?: number;
}

export interface MOISEGoal {
    id: string;
    name: string;
    description: string;
    type: "achievement" | "maintenance";
}

export interface MOISEPlan {
    id: string;
    goalId: string;
    steps: string[];
}

export interface MOISENorm {
    id: string;
    type: "permission" | "obligation" | "prohibition";
    role: string;
    mission: string;
    condition?: string;
}

export interface MOISEPermission extends MOISENorm {
    type: "permission";
}

export interface MOISEObligation extends MOISENorm {
    type: "obligation";
    deadline?: number;
}

// Tier 2: Process Intelligence Types

/**
 * Routine fixture for Tier 2 process orchestration
 */
export interface RoutineFixture extends ExecutionFixture<RunProgress> {
    config: RunProgress;
    
    /** Evolution stage of this routine */
    evolutionStage: EvolutionStage;
    
    /** Domain this routine belongs to */
    domain?: RoutineDomain;
    
    /** Navigator type for workflow execution */
    navigator?: NavigatorType;
}

/**
 * Evolution stage with metrics
 */
export interface EvolutionStage {
    /** Current strategy */
    strategy: ExecutionStrategy;
    
    /** Version identifier */
    version: string;
    
    /** Performance metrics */
    metrics: RoutineMetrics;
    
    /** Previous version reference */
    previousVersion?: string;
    
    /** Next version reference (if known) */
    nextVersion?: string;
    
    /** Improvement notes */
    improvements?: string[];
}

/**
 * Routine performance metrics
 */
export interface RoutineMetrics {
    /** Average duration in ms */
    avgDuration: number;
    
    /** Average credit cost */
    avgCredits: number;
    
    /** Success rate (0-1) */
    successRate?: number;
    
    /** Error rate (0-1) */
    errorRate?: number;
    
    /** User satisfaction (0-1) */
    satisfaction?: number;
    
    /** Complexity score */
    complexity?: number;
}

/**
 * Routine domains
 */
export type RoutineDomain = 
    | "security" 
    | "medical" 
    | "performance" 
    | "system" 
    | "api-bootstrap"
    | "data-bootstrap"
    | "customer-service"
    | "financial"
    | "general";

/**
 * Navigator types for different workflow formats
 */
export type NavigatorType = 
    | "native"      // Vrooli native JSON format
    | "bpmn"        // BPMN 2.0
    | "temporal"    // Temporal workflows
    | "airflow"     // Apache Airflow
    | "n8n"         // n8n workflows
    | "custom";     // Custom navigator

/**
 * Run state fixture for testing state machines
 */
export interface RunStateFixture {
    /** Initial state */
    initialState: string;
    
    /** State transitions */
    transitions: StateTransition[];
    
    /** Expected final state */
    expectedFinalState: string;
    
    /** Events emitted during transitions */
    expectedEvents: string[];
}

export interface StateTransition {
    from: string;
    to: string;
    trigger: string;
    conditions?: string[];
    actions?: string[];
}

// Tier 3: Execution Intelligence Types

/**
 * Execution context fixture for Tier 3
 */
export interface ExecutionContextFixture extends ExecutionFixture<RunProgress> {
    /** Execution strategy */
    strategy: ExecutionStrategy;
    
    /** Execution context */
    context: ExecutionContext;
    
    /** Tool configuration */
    tools?: ToolConfiguration;
}

/**
 * Execution context definition
 */
export interface ExecutionContext {
    /** Available tools */
    tools: string[];
    
    /** Execution constraints */
    constraints: ExecutionConstraints;
    
    /** Shared memory/state */
    sharedMemory?: Record<string, any>;
    
    /** Resource allocation */
    resources?: ResourceAllocation;
    
    /** Safety configuration */
    safety?: SafetyConfiguration;
}

/**
 * Execution constraints
 */
export interface ExecutionConstraints {
    /** Maximum tokens allowed */
    maxTokens?: number;
    
    /** Timeout in ms */
    timeout?: number;
    
    /** Tools requiring approval */
    requireApproval?: string[];
    
    /** Resource limits */
    resourceLimits?: {
        memory?: number;
        cpu?: number;
        apiCalls?: number;
    };
}

/**
 * Resource allocation
 */
export interface ResourceAllocation {
    /** Credit budget */
    creditBudget: number;
    
    /** Time budget in ms */
    timeBudget: number;
    
    /** Priority level */
    priority: "low" | "medium" | "high" | "critical";
    
    /** Resource pools */
    pools?: string[];
}

/**
 * Safety configuration
 */
export interface SafetyConfiguration {
    /** Synchronous checks (<10ms) */
    syncChecks: string[];
    
    /** Asynchronous monitoring agents */
    asyncAgents: string[];
    
    /** Domain-specific rules */
    domainRules?: string[];
    
    /** Emergency stop conditions */
    emergencyStop?: {
        conditions: string[];
        actions: string[];
    };
}

/**
 * Tool configuration
 */
export interface ToolConfiguration {
    /** Available tools */
    available: ToolDefinition[];
    
    /** Tool restrictions */
    restrictions?: ToolRestrictions;
    
    /** Tool preferences */
    preferences?: ToolPreferences;
}

export interface ToolDefinition {
    id: string;
    name: string;
    category: string;
    requiresApproval?: boolean;
    costPerUse?: number;
}

export interface ToolRestrictions {
    /** Blacklisted tools */
    blacklist?: string[];
    
    /** Rate limits per tool */
    rateLimits?: Record<string, number>;
    
    /** Time-based restrictions */
    timeRestrictions?: Record<string, { start: number; end: number }>;
}

export interface ToolPreferences {
    /** Preferred tools for specific tasks */
    taskPreferences?: Record<string, string[]>;
    
    /** Cost optimization preferences */
    costOptimization?: boolean;
    
    /** Performance optimization preferences */
    performanceOptimization?: boolean;
}

// Emergent Capabilities Types

/**
 * Emergent agent fixture
 */
export interface EmergentAgentFixture {
    /** Agent configuration */
    config: EmergentAgentConfig;
    
    /** Expected emergence */
    emergence: EmergenceDefinition;
    
    /** Learning configuration */
    learning?: LearningConfiguration;
}

/**
 * Emergent agent configuration
 */
export interface EmergentAgentConfig {
    /** Agent role */
    role: string;
    
    /** Event subscriptions */
    eventSubscriptions: string[];
    
    /** Initial capabilities */
    initialCapabilities?: string[];
    
    /** Learning enabled */
    learningEnabled: boolean;
    
    /** Collaboration preferences */
    collaborationPreferences?: {
        preferredPartners?: string[];
        communicationStyle?: "direct" | "blackboard" | "event-based";
        knowledgeSharing?: boolean;
    };
}

/**
 * Learning configuration
 */
export interface LearningConfiguration {
    /** Learning rate */
    learningRate: number;
    
    /** Pattern recognition threshold */
    patternThreshold: number;
    
    /** Memory retention period */
    memoryRetention: number;
    
    /** Adaptation strategies */
    adaptationStrategies: string[];
    
    /** Knowledge transfer enabled */
    knowledgeTransfer: boolean;
}

/**
 * Evolution path definition
 */
export interface EvolutionPath {
    /** Path identifier */
    id: string;
    
    /** Domain */
    domain: string;
    
    /** Evolution stages */
    stages: EvolutionStage[];
    
    /** Triggers for evolution */
    evolutionTriggers: EvolutionTrigger[];
    
    /** Expected outcomes */
    expectedOutcomes: {
        performanceGain: number;
        costReduction: number;
        qualityImprovement: number;
    };
}

/**
 * Evolution trigger
 */
export interface EvolutionTrigger {
    /** Trigger type */
    type: "usage_count" | "error_rate" | "performance_threshold" | "time_based" | "manual";
    
    /** Threshold value */
    threshold: number;
    
    /** Measurement period */
    period?: number;
    
    /** Additional conditions */
    conditions?: string[];
}

// Integration Scenario Types

/**
 * Complete integration scenario
 */
export interface IntegrationScenario {
    /** Scenario ID */
    id: string;
    
    /** Scenario name */
    name: string;
    
    /** Tier 1 configuration */
    tier1: SwarmFixture | SwarmFixture[];
    
    /** Tier 2 configuration */
    tier2: RoutineFixture | RoutineFixture[];
    
    /** Tier 3 configuration */
    tier3: ExecutionContextFixture | ExecutionContextFixture[];
    
    /** Expected event flow */
    expectedEvents: string[];
    
    /** Expected emergence */
    emergence: {
        capabilities: string[];
        metrics?: IntegrationMetrics;
    };
    
    /** Test scenarios */
    testScenarios?: TestScenario[];
}

/**
 * Integration metrics
 */
export interface IntegrationMetrics {
    /** End-to-end latency */
    latency: number;
    
    /** Overall accuracy */
    accuracy: number;
    
    /** Total cost */
    cost: number;
    
    /** Reliability score */
    reliability?: number;
    
    /** Scalability factor */
    scalability?: number;
}

/**
 * Test scenario within integration
 */
export interface TestScenario {
    /** Scenario name */
    name: string;
    
    /** Input data */
    input: any;
    
    /** Expected output */
    expectedOutput?: any;
    
    /** Success criteria */
    successCriteria: SuccessCriteria[];
    
    /** Failure conditions */
    failureConditions?: string[];
}

/**
 * Success criteria
 */
export interface SuccessCriteria {
    /** Metric name */
    metric: string;
    
    /** Comparison operator */
    operator: ">" | "<" | ">=" | "<=" | "==" | "!=";
    
    /** Target value */
    value: number | string | boolean;
    
    /** Optional tolerance */
    tolerance?: number;
}

// Factory Types

/**
 * Enhanced fixture factory interface following shared package patterns
 */
export interface ExecutionFixtureFactory<TConfig extends BaseConfigObject> {
    /** Create minimal fixture (following shared package pattern) */
    createMinimal(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    /** Create complete fixture (following shared package pattern) */
    createComplete(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    /** Create fixture with defaults */
    createWithDefaults(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    /** Create fixture variant */
    createVariant(variant: string, overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    /** Create fixture with overrides */
    create(overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    
    /** Create multiple fixtures */
    createBatch(count: number, overrides?: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>[];
    
    /** Create evolution path */
    createEvolutionPath?(stages: number): ExecutionFixture<TConfig>[];
    
    /** Validation methods (using shared package utilities) */
    validateFixture(fixture: ExecutionFixture<TConfig>): Promise<ValidationResult>;
    validateConfig(config: TConfig): Promise<ValidationResult>;
    validateEmergence(emergence: EmergenceDefinition): ValidationResult;
    validateIntegration(integration: IntegrationDefinition): ValidationResult;
    
    /** Integration with shared package validation */
    getConfigValidator(): any;
    getIntegrationAdapter(): any;
    
    /** Factory methods */
    isValid(fixture: unknown): fixture is ExecutionFixture<TConfig>;
    merge(base: ExecutionFixture<TConfig>, override: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
    applyDefaults(partialFixture: Partial<ExecutionFixture<TConfig>>): ExecutionFixture<TConfig>;
}

/**
 * Legacy fixture factory interface for backwards compatibility
 */
export interface FixtureFactory<T> {
    /** Create minimal fixture */
    createMinimal(): T;
    
    /** Create complete fixture */
    createComplete(): T;
    
    /** Create fixture with overrides */
    create(overrides?: Partial<T>): T;
    
    /** Create multiple fixtures */
    createBatch(count: number, overrides?: Partial<T>): T[];
    
    /** Create evolution path */
    createEvolutionPath?(stages: number): T[];
}

/**
 * Fixture collection
 */
export interface FixtureCollection<T> {
    /** Collection name */
    name: string;
    
    /** Collection description */
    description: string;
    
    /** Fixtures in the collection */
    fixtures: Record<string, T>;
    
    /** Get fixture by ID */
    get(id: string): T | undefined;
    
    /** Get fixtures by tag */
    getByTag(tag: string): T[];
    
    /** Get fixtures by tier */
    getByTier(tier: ExecutionTier): T[];
}

// Helper type for creating typed fixtures
export type TypedExecutionFixture<TConfig extends BaseConfigObject> = ExecutionFixture<TConfig>;

// Re-export common types from shared package for convenience
export type {
    ChatConfigObject,
    RunProgress,
    BotConfigObject,
    SwarmSubTask,
    SwarmResource,
    BlackboardItem,
    ChatToolCallRecord,
    SwarmPolicy,
    PendingToolCallEntry,
};
