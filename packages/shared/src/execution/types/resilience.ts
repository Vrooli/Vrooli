/**
 * Core type definitions for resilience and error handling
 * These types define the systematic error classification, recovery strategies,
 * and circuit breaker functionality across all tiers
 */

import type { EmergencyAction, EmergencyTrigger, NotificationChannel } from "./security.js";

/**
 * Error severity levels based on impact assessment
 */
export enum ErrorSeverity {
    FATAL = "FATAL",        // Complete system failure - immediate halt required
    CRITICAL = "CRITICAL",  // Severe impact - immediate attention required
    ERROR = "ERROR",        // Functionality impacted - automated recovery
    WARNING = "WARNING",    // Minor issue - monitor for patterns
    INFO = "INFO"          // Informational - standard logging
}

/**
 * Error categories for recovery strategy selection
 */
export enum ErrorCategory {
    TRANSIENT = "TRANSIENT",        // Temporary failures (network, timeouts)
    RESOURCE = "RESOURCE",          // Resource constraints (memory, credits, limits)
    LOGIC = "LOGIC",               // Business logic errors
    CONFIGURATION = "CONFIGURATION", // Configuration or setup issues
    SECURITY = "SECURITY",          // Security violations
    DATA = "DATA",                 // Data corruption or loss
    SYSTEM = "SYSTEM",             // System-level failures
    UNKNOWN = "UNKNOWN"            // Unclassified errors
}

/**
 * Error recoverability assessment
 */
export enum ErrorRecoverability {
    AUTOMATIC = "AUTOMATIC",        // Can be automatically recovered
    PARTIAL = "PARTIAL",           // Partial recovery possible
    MANUAL = "MANUAL",             // Requires manual intervention
    NONE = "NONE"                  // Cannot be recovered
}

/**
 * Recovery strategy types
 */
export enum RecoveryStrategy {
    EMERGENCY_STOP = "EMERGENCY_STOP",           // Immediate system halt
    ESCALATE_TO_HUMAN = "ESCALATE_TO_HUMAN",     // Human intervention required
    FALLBACK_STRATEGY = "FALLBACK_STRATEGY",     // Switch to alternative approach
    FALLBACK_MODEL = "FALLBACK_MODEL",           // Switch to alternative model
    REDUCE_SCOPE = "REDUCE_SCOPE",               // Reduce operation scope
    GRACEFUL_DEGRADATION = "GRACEFUL_DEGRADATION", // Accept partial success
    RETRY_SAME = "RETRY_SAME",                   // Retry exact operation
    WAIT_AND_RETRY = "WAIT_AND_RETRY",           // Wait then retry
    RETRY_MODIFIED = "RETRY_MODIFIED",           // Retry with modifications
    ESCALATE_TO_PARENT = "ESCALATE_TO_PARENT",   // Let higher tier handle
    LOG_WARNING = "LOG_WARNING",                 // Log and monitor
    LOG_INFO = "LOG_INFO",                       // Informational logging
    CIRCUIT_BREAK = "CIRCUIT_BREAK"              // Open circuit breaker
}

/**
 * Backoff strategy types for retries
 */
export enum BackoffType {
    FIXED = "FIXED",                    // Fixed delay
    LINEAR = "LINEAR",                  // Linear increase
    EXPONENTIAL = "EXPONENTIAL",        // Exponential backoff
    EXPONENTIAL_JITTER = "EXPONENTIAL_JITTER", // With jitter
    ADAPTIVE = "ADAPTIVE",              // Adaptive based on conditions
    FIBONACCI = "FIBONACCI",            // Fibonacci sequence
    CUSTOM = "CUSTOM"                   // Custom algorithm
}

/**
 * Circuit breaker states
 */
export enum CircuitState {
    CLOSED = "CLOSED",          // Normal operation
    OPEN = "OPEN",             // Circuit open - fail fast
    HALF_OPEN = "HALF_OPEN"    // Testing recovery
}

/**
 * Fallback types for recovery actions
 */
export enum FallbackType {
    RETRY = "RETRY",
    ALTERNATE_STRATEGY = "ALTERNATE_STRATEGY",
    ALTERNATE_MODEL = "ALTERNATE_MODEL",
    ALTERNATE_TOOL = "ALTERNATE_TOOL",
    REDUCE_SCOPE = "REDUCE_SCOPE",
    SKIP_STEP = "SKIP_STEP",
    USE_CACHED_RESULT = "USE_CACHED_RESULT",
    MANUAL_INTERVENTION = "MANUAL_INTERVENTION",
    DEFAULT_RESPONSE = "DEFAULT_RESPONSE"
}

/**
 * Degradation modes for circuit breakers
 */
export enum DegradationMode {
    FAIL_FAST = "FAIL_FAST",
    QUEUE_REQUESTS = "QUEUE_REQUESTS",
    USE_FALLBACK = "USE_FALLBACK",
    PARTIAL_SERVICE = "PARTIAL_SERVICE"
}

/**
 * Condition operators for fallback conditions
 */
export enum ConditionOperator {
    EQUALS = "EQUALS",
    NOT_EQUALS = "NOT_EQUALS",
    GREATER_THAN = "GREATER_THAN",
    LESS_THAN = "LESS_THAN",
    CONTAINS = "CONTAINS",
    IN = "IN",
    NOT_IN = "NOT_IN",
    REGEX_MATCH = "REGEX_MATCH"
}

/**
 * Enhanced error classification with systematic assessment
 */
export interface ErrorClassification {
    severity: ErrorSeverity;
    category: ErrorCategory;
    recoverability: ErrorRecoverability;
    systemFunctional: boolean;
    multipleComponentsAffected: boolean;
    dataRisk: boolean;
    securityRisk: boolean;
    confidenceScore: number; // 0-1 classification confidence
    timestamp: Date;
    metadata: Record<string, unknown>;
}

/**
 * Error context for classification and recovery
 */
export interface ErrorContext {
    tier: 1 | 2 | 3;
    component: string;
    operation: string;
    attemptCount: number;
    previousStrategies: RecoveryStrategy[];
    systemState: Record<string, unknown>;
    resourceState: Record<string, unknown>;
    performanceMetrics: Record<string, number>;
    userContext?: {
        userId?: string;
        sessionId?: string;
        requestId: string;
    };
}

/**
 * Recovery strategy configuration
 */
export interface RecoveryStrategyConfig {
    strategyType: RecoveryStrategy;
    maxAttempts: number;
    backoffStrategy: BackoffStrategyConfig;
    fallbackActions: FallbackAction[];
    circuitBreakerConfig?: CircuitBreakerConfig;
    priority: number;
    timeoutMs: number;
    conditions: RecoveryCondition[];
    estimatedSuccessRate: number;
    resourceRequirements: Record<string, number>;
}

/**
 * Backoff strategy configuration
 */
export interface BackoffStrategyConfig {
    type: BackoffType;
    initialDelayMs: number;
    maxDelayMs: number;
    multiplier: number;
    jitterPercent: number;
    adaptiveAdjustment: boolean;
    customParams?: Record<string, unknown>;
}

/**
 * Fallback action definition
 */
export interface FallbackAction {
    type: FallbackType;
    configuration: Record<string, unknown>;
    conditions: FallbackCondition[];
    priority: number;
    estimatedSuccessRate: number;
    qualityReduction: number; // 0-1 expected quality impact
    resourceAdjustment: Record<string, number>;
}

/**
 * Fallback condition
 */
export interface FallbackCondition {
    type: string;
    value: unknown;
    operator: ConditionOperator;
    description: string;
}

/**
 * Recovery condition
 */
export interface RecoveryCondition {
    field: string;
    operator: ConditionOperator;
    value: unknown;
    weight: number; // For weighted condition evaluation
}

/**
 * Circuit breaker configuration
 */
export interface CircuitBreakerConfig {
    failureThreshold: number;        // Failures before opening
    timeoutMs: number;               // Operation timeout
    resetTimeoutMs: number;          // Time before half-open
    successThreshold: number;        // Successes to close from half-open
    monitoringWindowMs: number;      // Window for failure counting
    fallbackStrategy?: FallbackStrategyConfig;
    healthCheckInterval: number;     // Health check frequency
    degradationMode: DegradationMode;
    errorThresholds: ErrorThresholdConfig[];
}

/**
 * Fallback strategy for circuit breakers
 */
export interface FallbackStrategyConfig {
    type: string;
    configuration: Record<string, unknown>;
    qualityReduction: number; // 0-1
}

/**
 * Error threshold configuration for circuit breakers
 */
export interface ErrorThresholdConfig {
    errorType: string;
    threshold: number;
    windowMs: number;
    action: RecoveryStrategy;
}

/**
 * Circuit breaker state information
 */
export interface CircuitBreakerState {
    state: CircuitState;
    failureCount: number;
    successCount: number;
    lastFailureTime?: Date;
    lastSuccessTime?: Date;
    stateChangeTime: Date;
    nextRetryTime?: Date;
    metadata: Record<string, unknown>;
}

/**
 * Resilience event for pattern learning
 */
export interface ResilienceEvent {
    id: string;
    timestamp: Date;
    type: ResilienceEventType;
    severity: ErrorSeverity;
    source: ResilienceEventSource;
    classification: ErrorClassification;
    context: ErrorContext;
    strategy: RecoveryStrategyConfig;
    outcome: ResilienceOutcome;
    learningData: ResilienceLearningData;
}

/**
 * Resilience event types
 */
export enum ResilienceEventType {
    ERROR_DETECTED = "ERROR_DETECTED",
    ERROR_CLASSIFIED = "ERROR_CLASSIFIED",
    RECOVERY_INITIATED = "RECOVERY_INITIATED",
    RECOVERY_COMPLETED = "RECOVERY_COMPLETED",
    RECOVERY_FAILED = "RECOVERY_FAILED",
    CIRCUIT_BREAKER_OPENED = "CIRCUIT_BREAKER_OPENED",
    CIRCUIT_BREAKER_CLOSED = "CIRCUIT_BREAKER_CLOSED",
    FALLBACK_TRIGGERED = "FALLBACK_TRIGGERED",
    ESCALATION_TRIGGERED = "ESCALATION_TRIGGERED",
    PATTERN_DETECTED = "PATTERN_DETECTED"
}

/**
 * Resilience event source
 */
export interface ResilienceEventSource {
    tier: 1 | 2 | 3;
    component: string;
    agent?: string;
    operation: string;
    requestId: string;
}

/**
 * Resilience outcome
 */
export interface ResilienceOutcome {
    success: boolean;
    duration: number;
    attemptCount: number;
    strategiesUsed: RecoveryStrategy[];
    finalStrategy?: RecoveryStrategy;
    qualityImpact: number; // 0-1
    resourceUsage: Record<string, number>;
    userImpact: UserImpactLevel;
    lessons: string[];
}

/**
 * User impact levels
 */
export enum UserImpactLevel {
    NONE = "NONE",          // No user-visible impact
    MINIMAL = "MINIMAL",    // Slight delay or degradation
    MODERATE = "MODERATE",  // Noticeable impact but functional
    SEVERE = "SEVERE",      // Significant functionality loss
    CRITICAL = "CRITICAL"   // Major disruption
}

/**
 * Resilience learning data for emergent intelligence
 */
export interface ResilienceLearningData {
    patternId?: string;
    similarity: number; // 0-1 similarity to previous events
    contextFeatures: string[];
    effectiveStrategies: RecoveryStrategy[];
    ineffectiveStrategies: RecoveryStrategy[];
    environmentalFactors: Record<string, unknown>;
    recommendations: string[];
    confidence: number; // 0-1 confidence in recommendations
}

/**
 * Error pattern for agent learning
 */
export interface ErrorPattern {
    id: string;
    name: string;
    description: string;
    frequency: number;
    severity: ErrorSeverity;
    category: ErrorCategory;
    triggerConditions: PatternCondition[];
    commonContext: Record<string, unknown>;
    effectiveStrategies: RecoveryStrategyConfig[];
    successRate: number;
    averageResolutionTime: number;
    lastSeen: Date;
    confidence: number;
}

/**
 * Pattern condition for matching
 */
export interface PatternCondition {
    field: string;
    operator: ConditionOperator;
    value: unknown;
    weight: number;
    tolerance?: number; // For numeric comparisons
}

/**
 * Resilience intelligence agent
 */
export interface ResilienceAgent {
    id: string;
    name: string;
    tier: 1 | 2 | 3;
    specialization: ResilienceSpecialization[];
    learningCapability: LearningCapability;
    decisionEngine: DecisionEngine;
    knowledgeBase: KnowledgeBase;
    performance: ResilienceAgentPerformance;
}

/**
 * Resilience specializations
 */
export enum ResilienceSpecialization {
    ERROR_CLASSIFICATION = "ERROR_CLASSIFICATION",
    STRATEGY_SELECTION = "STRATEGY_SELECTION",
    PATTERN_RECOGNITION = "PATTERN_RECOGNITION",
    CIRCUIT_BREAKER_MANAGEMENT = "CIRCUIT_BREAKER_MANAGEMENT",
    RESOURCE_OPTIMIZATION = "RESOURCE_OPTIMIZATION",
    USER_IMPACT_ASSESSMENT = "USER_IMPACT_ASSESSMENT",
    ESCALATION_MANAGEMENT = "ESCALATION_MANAGEMENT"
}

/**
 * Learning capability configuration
 */
export interface LearningCapability {
    enabled: boolean;
    algorithms: LearningAlgorithm[];
    trainingDataSize: number;
    adaptationRate: number;
    forgettingRate: number;
    confidenceThreshold: number;
}

/**
 * Learning algorithms
 */
export enum LearningAlgorithm {
    PATTERN_MATCHING = "PATTERN_MATCHING",
    STATISTICAL_ANALYSIS = "STATISTICAL_ANALYSIS",
    DECISION_TREE = "DECISION_TREE",
    NEURAL_NETWORK = "NEURAL_NETWORK",
    REINFORCEMENT_LEARNING = "REINFORCEMENT_LEARNING",
    ENSEMBLE = "ENSEMBLE"
}

/**
 * Decision engine for strategy selection
 */
export interface DecisionEngine {
    type: DecisionEngineType;
    rules: DecisionRule[];
    weights: Record<string, number>;
    thresholds: Record<string, number>;
    fallbackRules: DecisionRule[];
}

/**
 * Decision engine types
 */
export enum DecisionEngineType {
    RULE_BASED = "RULE_BASED",
    WEIGHTED_SCORING = "WEIGHTED_SCORING",
    MACHINE_LEARNING = "MACHINE_LEARNING",
    HYBRID = "HYBRID"
}

/**
 * Decision rule
 */
export interface DecisionRule {
    id: string;
    condition: string;
    action: RecoveryStrategy;
    priority: number;
    confidence: number;
    metadata: Record<string, unknown>;
}

/**
 * Knowledge base for resilience agents
 */
export interface KnowledgeBase {
    patterns: ErrorPattern[];
    strategies: RecoveryStrategyConfig[];
    outcomes: ResilienceOutcome[];
    circuitStates: Map<string, CircuitBreakerState>;
    systemState: Record<string, unknown>;
    lastUpdated: Date;
}

/**
 * Resilience agent performance metrics
 */
export interface ResilienceAgentPerformance {
    successRate: number;
    averageResolutionTime: number;
    accuracyScore: number;
    learningRate: number;
    adaptationTime: number;
    resourceEfficiency: number;
    userSatisfaction: number;
    lastEvaluation: Date;
}

/**
 * Recovery execution result
 */
export interface RecoveryResult {
    success: boolean;
    strategy: RecoveryStrategy;
    duration: number;
    attempts: number;
    error?: Error;
    outcome: ResilienceOutcome;
    nextRecommendation?: RecoveryStrategyConfig;
}

/**
 * Emergency response configuration
 */
export interface EmergencyResponse {
    enabled: boolean;
    triggers: ResilienceEmergencyTrigger[];
    actions: ResilienceEmergencyAction[];
    notificationChannels: ResilienceNotificationChannel[];
    escalationPath: EscalationLevel[];
}

/**
 * Enhanced emergency trigger for resilience
 */
export interface ResilienceEmergencyTrigger extends EmergencyTrigger {
    triggerType: EmergencyTriggerType;
    threshold: number;
    windowMs: number;
    errorSeverity: ErrorSeverity;
}

/**
 * Emergency trigger types
 */
export enum EmergencyTriggerType {
    ERROR_RATE = "ERROR_RATE",
    SYSTEM_FAILURE = "SYSTEM_FAILURE",
    SECURITY_BREACH = "SECURITY_BREACH",
    RESOURCE_EXHAUSTION = "RESOURCE_EXHAUSTION",
    MANUAL = "MANUAL"
}

/**
 * Enhanced emergency action for resilience
 */
export interface ResilienceEmergencyAction extends EmergencyAction {
    actionType: EmergencyActionType;
    priority: number;
    configuration: Record<string, unknown>;
}

/**
 * Emergency action types
 */
export enum EmergencyActionType {
    STOP_ALL = "STOP_ALL",
    STOP_TIER = "STOP_TIER",
    STOP_COMPONENT = "STOP_COMPONENT",
    PAUSE = "PAUSE",
    ISOLATE = "ISOLATE",
    QUARANTINE = "QUARANTINE",
    ROLLBACK = "ROLLBACK"
}

/**
 * Enhanced notification channel for resilience
 */
export interface ResilienceNotificationChannel extends NotificationChannel {
    severity: ErrorSeverity[];
    enabled: boolean;
}

/**
 * Notification channel types
 */
export enum NotificationChannelType {
    EMAIL = "EMAIL",
    SMS = "SMS",
    WEBHOOK = "WEBHOOK",
    CONSOLE = "CONSOLE",
    SLACK = "SLACK",
    TEAMS = "TEAMS"
}

/**
 * Escalation level
 */
export interface EscalationLevel {
    level: number;
    name: string;
    timeout: number;
    contacts: string[];
    actions: ResilienceEmergencyAction[];
    autoEscalate: boolean;
}

/**
 * Resilience metrics for monitoring
 */
export interface ResilienceMetrics {
    errorRate: number;
    recoveryRate: number;
    meanTimeToRecovery: number;
    circuitBreakerTrips: number;
    fallbackUsage: number;
    userImpactScore: number;
    systemReliability: number;
    adaptationEffectiveness: number;
    timestamp: Date;
    breakdown: Record<string, number>;
}

/**
 * Resilience health check configuration
 */
export interface ResilienceHealthCheck {
    id: string;
    name: string;
    target: string;
    interval: number;
    timeout: number;
    retries: number;
    healthIndicators: ResilienceHealthIndicator[];
    circuitBreakerIntegration: boolean;
}

/**
 * Resilience health indicator
 */
export interface ResilienceHealthIndicator {
    name: string;
    type: HealthIndicatorType;
    threshold: number;
    weight: number;
    critical: boolean;
}

/**
 * Health indicator types
 */
export enum HealthIndicatorType {
    RESPONSE_TIME = "RESPONSE_TIME",
    ERROR_RATE = "ERROR_RATE",
    THROUGHPUT = "THROUGHPUT",
    RESOURCE_USAGE = "RESOURCE_USAGE",
    AVAILABILITY = "AVAILABILITY",
    CUSTOM = "CUSTOM"
}
