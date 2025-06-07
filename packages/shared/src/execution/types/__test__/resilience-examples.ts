/**
 * Test file to verify resilience types work correctly
 */

import {
    ErrorSeverity,
    ErrorCategory,
    ErrorRecoverability,
    RecoveryStrategy,
    BackoffType,
    DegradationMode,
    ResilienceEventType,
    UserImpactLevel,
    ResilienceSpecialization,
    LearningAlgorithm,
    DecisionEngineType,
    type ErrorClassification,
    type ErrorContext,
    type RecoveryStrategyConfig,
    type CircuitBreakerConfig,
    type ResilienceEvent,
    type ResilienceAgent,
    type ResilienceMetrics,
} from "../resilience.js";

// Test basic error classification
const testErrorClassification: ErrorClassification = {
    severity: ErrorSeverity.ERROR,
    category: ErrorCategory.TRANSIENT,
    recoverability: ErrorRecoverability.AUTOMATIC,
    systemFunctional: true,
    multipleComponentsAffected: false,
    dataRisk: false,
    securityRisk: false,
    confidenceScore: 0.85,
    timestamp: new Date(),
    metadata: {
        component: "test-component",
        operation: "test-operation",
    },
};

// Test error context
const testErrorContext: ErrorContext = {
    tier: 3,
    component: "test-component",
    operation: "test-operation",
    attemptCount: 1,
    previousStrategies: [],
    systemState: { healthy: true },
    resourceState: { available: true },
    performanceMetrics: { responseTime: 100 },
    userContext: {
        requestId: "test-request-123",
    },
};

// Test recovery strategy config
const testRecoveryConfig: RecoveryStrategyConfig = {
    strategyType: RecoveryStrategy.RETRY_SAME,
    maxAttempts: 3,
    backoffStrategy: {
        type: BackoffType.EXPONENTIAL,
        initialDelayMs: 1000,
        maxDelayMs: 10000,
        multiplier: 2,
        jitterPercent: 10,
        adaptiveAdjustment: false,
    },
    fallbackActions: [],
    priority: 1,
    timeoutMs: 30000,
    conditions: [],
    estimatedSuccessRate: 0.8,
    resourceRequirements: { cpu: 0.1, memory: 100 },
};

// Test circuit breaker config
const testCircuitConfig: CircuitBreakerConfig = {
    failureThreshold: 5,
    timeoutMs: 5000,
    resetTimeoutMs: 60000,
    successThreshold: 2,
    monitoringWindowMs: 300000,
    healthCheckInterval: 30000,
    degradationMode: DegradationMode.FAIL_FAST,
    errorThresholds: [],
};

// Test resilience event
const testResilienceEvent: ResilienceEvent = {
    id: "event-123",
    timestamp: new Date(),
    type: ResilienceEventType.ERROR_DETECTED,
    severity: ErrorSeverity.ERROR,
    source: {
        tier: 3,
        component: "test-component",
        operation: "test-operation",
        requestId: "request-123",
    },
    classification: testErrorClassification,
    context: testErrorContext,
    strategy: testRecoveryConfig,
    outcome: {
        success: true,
        duration: 1500,
        attemptCount: 2,
        strategiesUsed: [RecoveryStrategy.RETRY_SAME],
        finalStrategy: RecoveryStrategy.RETRY_SAME,
        qualityImpact: 0.1,
        resourceUsage: { cpu: 0.2, memory: 150 },
        userImpact: UserImpactLevel.MINIMAL,
        lessons: ["Retry strategy effective for transient errors"],
    },
    learningData: {
        similarity: 0.7,
        contextFeatures: ["transient-error", "low-load"],
        effectiveStrategies: [RecoveryStrategy.RETRY_SAME],
        ineffectiveStrategies: [],
        environmentalFactors: { load: "low", time: "peak" },
        recommendations: ["Continue using retry for similar errors"],
        confidence: 0.85,
    },
};

// Test resilience metrics
const testMetrics: ResilienceMetrics = {
    errorRate: 0.02,
    recoveryRate: 0.95,
    meanTimeToRecovery: 2500,
    circuitBreakerTrips: 3,
    fallbackUsage: 0.05,
    userImpactScore: 0.1,
    systemReliability: 0.98,
    adaptationEffectiveness: 0.87,
    timestamp: new Date(),
    breakdown: {
        tier1: 0.99,
        tier2: 0.98,
        tier3: 0.97,
    },
};

// Test resilience agent
const testAgent: ResilienceAgent = {
    id: "agent-123",
    name: "Error Classification Agent",
    tier: 3,
    specialization: [ResilienceSpecialization.ERROR_CLASSIFICATION, ResilienceSpecialization.STRATEGY_SELECTION],
    learningCapability: {
        enabled: true,
        algorithms: [LearningAlgorithm.PATTERN_MATCHING, LearningAlgorithm.STATISTICAL_ANALYSIS],
        trainingDataSize: 1000,
        adaptationRate: 0.1,
        forgettingRate: 0.01,
        confidenceThreshold: 0.8,
    },
    decisionEngine: {
        type: DecisionEngineType.HYBRID,
        rules: [],
        weights: { accuracy: 0.5, speed: 0.3, resource: 0.2 },
        thresholds: { confidence: 0.8, risk: 0.2 },
        fallbackRules: [],
    },
    knowledgeBase: {
        patterns: [],
        strategies: [],
        outcomes: [],
        circuitStates: new Map(),
        systemState: {},
        lastUpdated: new Date(),
    },
    performance: {
        successRate: 0.92,
        averageResolutionTime: 1800,
        accuracyScore: 0.88,
        learningRate: 0.15,
        adaptationTime: 500,
        resourceEfficiency: 0.85,
        userSatisfaction: 0.9,
        lastEvaluation: new Date(),
    },
};

// Export test objects for potential use in actual tests
export {
    testErrorClassification,
    testErrorContext,
    testRecoveryConfig,
    testCircuitConfig,
    testResilienceEvent,
    testMetrics,
    testAgent,
};
