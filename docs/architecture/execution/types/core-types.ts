/**
 * Core Type Definitions for Vrooli's Three-Tier Execution Architecture
 * 
 * This is the **single source of truth** for all type definitions used across
 * Vrooli's three-tier execution architecture. All other files reference types
 * from this centralized system to ensure consistency and maintainability.
 * 
 * **File Organization**:
 * - Communication patterns and interfaces
 * - Error handling and recovery types
 * - Resource management interfaces
 * - Security and context types
 * - Event system definitions
 * - Performance monitoring types
 * - State synchronization interfaces
 * 
 * **Related Documentation**:
 * - [Communication Patterns](../communication/communication-patterns.md) - Usage of communication interfaces
 * - [Error Propagation](../resilience/error-propagation.md) - Error handling type usage
 * - [Resource Management](../resource-management/resource-coordination.md) - Resource type usage
 * - [Security Boundaries](../security/security-boundaries.md) - Security type usage
 * - [Event Bus Protocol](../event-driven/event-bus-protocol.md) - Event type usage
 * - [Performance Characteristics](../monitoring/performance-characteristics.md) - Performance type usage
 * - [State Synchronization](../context-memory/state-synchronization.md) - State type usage
 * - [Integration Map](../communication/integration-map.md) - End-to-end type validation
 */

// ================================
// Basic Enums and Identifiers
// ================================

export enum Tier {
    TIER_1 = "tier_1",    // Coordination Intelligence
    TIER_2 = "tier_2",    // Process Intelligence  
    TIER_3 = "tier_3"     // Execution Intelligence
}

export enum StrategyType {
    CONVERSATIONAL = "conversational",
    REASONING = "reasoning",
    DETERMINISTIC = "deterministic"
}

export enum ExecutionPriority {
    LOW = "low",
    NORMAL = "normal",
    HIGH = "high",
    CRITICAL = "critical",
    EMERGENCY = "emergency"
}

export enum StepType {
    INPUT = "input",
    PROCESS = "process",
    OUTPUT = "output",
    DECISION = "decision",
    LOOP = "loop",
    PARALLEL = "parallel",
    SUBROUTINE = "subroutine"
}

export enum StepStatus {
    PENDING = "pending",
    RUNNING = "running",
    COMPLETED = "completed",
    FAILED = "failed",
    SKIPPED = "skipped",
    CANCELLED = "cancelled",
    BLOCKED = "blocked"
}

export enum RunStatus {
    PENDING = "pending",
    INITIALIZING = "initializing",
    RUNNING = "running",
    COMPLETED = "completed",
    FAILED = "failed",
    CANCELLED = "cancelled",
    PAUSED = "paused",
    BLOCKED = "blocked"
}

export enum NavigatorType {
    BPMN = "bpmn",
    LANGCHAIN = "langchain",
    TEMPORAL = "temporal",
    AIRFLOW = "airflow",
    N8N = "n8n",
    CUSTOM = "custom"
}

export enum DataSensitivity {
    PUBLIC = "public",
    INTERNAL = "internal",
    CONFIDENTIAL = "confidential",
    SECRET = "secret",
    PII = "pii"
}

// ================================
// Core Execution Interfaces
// ================================

export interface ExecutionStrategy {
    readonly type: StrategyType;
    readonly configuration: Record<string, unknown>;
    readonly fallbackStrategy?: ExecutionStrategy;
    readonly costMultiplier: number;        // Cost factor for this strategy
    readonly qualityTarget: number;         // Expected quality score (0.0-1.0)
}

export interface RoutineManifest {
    readonly routineId: string;
    readonly version: string;
    readonly name: string;
    readonly description?: string;
    readonly navigatorType: NavigatorType;
    readonly steps: RoutineStep[];
    readonly inputs: InputSpec[];
    readonly outputs: OutputSpec[];
    readonly resourceRequirements: ResourceRequirements;
    readonly defaultStrategy: StrategyType;
    readonly metadata: RoutineMetadata;
}

export interface RoutineStep {
    readonly stepId: string;
    readonly stepType: StepType;
    readonly name: string;
    readonly description?: string;
    readonly inputMappings: Record<string, string>;
    readonly outputMappings: Record<string, string>;
    readonly configuration: Record<string, unknown>;
    readonly strategy?: StrategyType;        // Override default strategy
    readonly required: boolean;              // Can this step be skipped?
    readonly timeout?: number;               // Step-specific timeout
}

export interface InputSpec {
    readonly name: string;
    readonly type: string;
    readonly required: boolean;
    readonly defaultValue?: unknown;
    readonly description?: string;
    readonly validation?: ValidationRule[];
}

export interface OutputSpec {
    readonly name: string;
    readonly type: string;
    readonly description?: string;
    readonly sensitivity?: DataSensitivity;
}

export interface ResourceRequirements {
    readonly minCredits: number;
    readonly estimatedCredits: number;
    readonly maxCredits: number;
    readonly estimatedDurationMs: number;
    readonly maxDurationMs: number;
    readonly memoryMB: number;
    readonly concurrencyLevel: number;
    readonly toolsRequired: string[];
}

export interface RoutineMetadata {
    readonly createdBy: string;
    readonly createdAt: Date;
    readonly updatedAt: Date;
    readonly version: string;
    readonly tags: string[];
    readonly category: string;
    readonly isPublic: boolean;
}

// ================================
// Resource Management Types (consolidated from resource-types.ts)
// ================================

export interface ResourceLimits {
    readonly maxCredits: number;           // AI model costs
    readonly maxDurationMs: number;        // Wall clock time
    readonly maxConcurrentBranches: number; // Parallel execution
    readonly maxMemoryMB: number;          // Memory usage
    readonly maxToolCalls: number;         // Tool invocation limit
    readonly priorityBonus?: number;       // Additional resources for high priority
    readonly emergencyReserve?: number;    // Emergency credit reserve
}

export interface ResourceLimitOverrides {
    readonly maxCredits?: number;
    readonly maxDurationMs?: number;
    readonly maxMemoryMB?: number;
    readonly maxToolCalls?: number;
    readonly maxConcurrentBranches?: number;
    readonly priorityBonus?: number;
    readonly emergencyReserve?: number;
}

export interface ResourceUsage {
    readonly creditsUsed: number;
    readonly timeElapsedMs: number;
    readonly memoryUsedMB: number;
    readonly toolCallsMade: number;
    readonly concurrentBranchesActive: number;
    readonly peakMemoryMB: number;
    readonly networkBytesTransferred: number;
    readonly diskBytesUsed: number;
    readonly gpuTimeMs?: number;
}

// ================================
// Security Types (consolidated from security-types.ts)
// ================================

export enum PermissionEffect {
    ALLOW = "allow",
    DENY = "deny"
}

export enum SecurityClearance {
    PUBLIC = "public",
    INTERNAL = "internal",
    CONFIDENTIAL = "confidential",
    SECRET = "secret",
    TOP_SECRET = "top_secret"
}

export interface Permission {
    readonly resource: string;              // What resource (e.g., "routine:123", "tool:web_search")
    readonly action: string;                // What action (e.g., "read", "write", "execute")
    readonly effect: PermissionEffect;      // Allow or deny
    readonly scope: string;                 // Where this permission applies
    readonly conditions?: PermissionCondition[];
    readonly priority: number;              // Higher priority overrides lower
}

export interface PermissionCondition {
    readonly type: string;
    readonly operator: string;
    readonly value: unknown;
    readonly description?: string;
}

export interface SecurityContext {
    readonly requesterTier: Tier;
    readonly targetTier: Tier;
    readonly operation: string;
    readonly clearanceLevel: SecurityClearance;
    readonly permissions: Permission[];
    readonly auditTrail: AuditEntry[];
    readonly encryptionRequired: boolean;
    readonly signatureRequired: boolean;
    readonly sessionId: string;
    readonly timestamp: Date;
}

export interface AuditEntry {
    readonly timestamp: Date;
    readonly tier: Tier;
    readonly operation: string;
    readonly resource: string;
    readonly result: string;
    readonly details?: Record<string, unknown>;
    readonly userId?: string;
    readonly agentId?: string;
}

export interface ApprovalConfig {
    readonly required: boolean;
    readonly approvers: string[];           // Agent/user IDs who can approve
    readonly threshold: number;             // Number of approvals needed
    readonly timeoutMs: number;             // How long to wait for approval
    readonly autoRejectOnTimeout: boolean;  // Auto-reject vs manual intervention
}

// ================================
// Error Types (consolidated from error-types.ts)
// ================================

export enum ErrorSeverity {
    INFO = "info",           // Informational, no action needed
    WARNING = "warning",     // Potential issue, monitor
    ERROR = "error",         // Error occurred, retry possible
    CRITICAL = "critical",   // Critical error, immediate attention
    FATAL = "fatal"          // System failure, emergency protocols
}

export enum ExecutionErrorType {
    // Common cross-tier errors
    RESOURCE_EXHAUSTED = "resource_exhausted",
    TIMEOUT = "timeout",
    RATE_LIMITED = "rate_limited",
    DEPENDENCY_UNAVAILABLE = "dependency_unavailable",
    PERMISSION_DENIED = "permission_denied",
    VALIDATION_FAILED = "validation_failed",
    NETWORK_ERROR = "network_error",
    AUTHENTICATION_FAILED = "authentication_failed",
    QUOTA_EXCEEDED = "quota_exceeded",

    // Tier 3 specific errors
    TOOL_EXECUTION_FAILED = "tool_execution_failed",
    MODEL_UNAVAILABLE = "model_unavailable",
    STRATEGY_EXECUTION_FAILED = "strategy_execution_failed",
    OUTPUT_VALIDATION_FAILED = "output_validation_failed",
    SANDBOX_BREACH = "sandbox_breach",

    // Tier 2 specific errors
    ROUTINE_PARSING_FAILED = "routine_parsing_failed",
    NAVIGATOR_UNAVAILABLE = "navigator_unavailable",
    BRANCH_SYNCHRONIZATION_FAILED = "branch_synchronization_failed",
    CONTEXT_CORRUPTION = "context_corruption",

    // Tier 1 specific errors
    TEAM_FORMATION_FAILED = "team_formation_failed",
    GOAL_DECOMPOSITION_FAILED = "goal_decomposition_failed",
    RESOURCE_ALLOCATION_FAILED = "resource_allocation_failed",
    SWARM_COORDINATION_FAILED = "swarm_coordination_failed",

    // Cross-tier system errors
    SECURITY_VIOLATION = "security_violation",
    STATE_CORRUPTION = "state_corruption",
    TRANSACTION_FAILED = "transaction_failed",
    COMMUNICATION_FAILURE = "communication_failure",
    CIRCUIT_BREAKER_OPEN = "circuit_breaker_open",
    FATAL_ERROR = "fatal_error"
}

export interface ExecutionError {
    readonly id: string;
    readonly type: ExecutionErrorType;
    readonly message: string;
    readonly details: Record<string, unknown>;
    readonly timestamp: Date;
    readonly context: ErrorContext;
    readonly stackTrace?: string;
    readonly causedBy?: ExecutionError;
    readonly relatedErrors: string[];
}

export interface ErrorContext {
    readonly tier: Tier;
    readonly componentId: string;
    readonly operationType: string;
    readonly requestId?: string;
    readonly correlationId?: string;
    readonly sessionId?: string;
    readonly userId?: string;
    readonly agentId?: string;
    readonly additionalContext: Record<string, unknown>;
}

// ================================
// Request/Response Interfaces
// ================================

export interface RoutineExecutionRequest {
    // Identity and routing
    readonly routineId: string;
    readonly requestId: string;            // For request tracking
    readonly conversationId: string;       // Swarm context
    readonly initiatingAgent: string;      // Which agent initiated

    // Execution context
    readonly inputs: Record<string, unknown>;
    readonly parentRunId?: string;         // For nested routines
    readonly strategy?: ExecutionStrategy; // Strategy override

    // Resource allocation
    readonly resourceLimits: ResourceLimits;
    readonly priority: ExecutionPriority;
    readonly timeoutMs?: number;

    // Security and permissions
    readonly permissions: Permission[];
    readonly sensitivityLevel: DataSensitivity;
    readonly approvalConfig: ApprovalConfig;
    readonly securityContext: SecurityContext;
}

export interface RoutineExecutionResult {
    // Identity
    readonly runId: string;
    readonly requestId: string;
    readonly status: RunStatus;

    // Results
    readonly outputs: Record<string, unknown>;
    readonly exports: ExportDeclaration[];  // Data to export to parent/blackboard
    readonly errors: ExecutionError[];      // Any non-fatal errors

    // Metrics
    readonly resourceUsage: ResourceUsage;
    readonly executionMetrics: ExecutionMetrics;
    readonly strategyEvolution: StrategyEvolutionReport;

    // State for resumption
    readonly finalState?: RunState;
    readonly checkpoints: Checkpoint[];
}

export interface StepExecutionRequest {
    // Step identity
    readonly stepId: string;
    readonly runId: string;
    readonly routineId: string;
    readonly stepType: StepType;

    // Execution context
    readonly context: RunContext;
    readonly strategy: ExecutionStrategy;
    readonly inputData: unknown;

    // Navigator information
    readonly navigatorType: NavigatorType;
    readonly platformSpecificConfig?: Record<string, unknown>;

    // Constraints and validation
    readonly validationRules: ValidationRule[];
    readonly outputSchema?: JsonSchema;
    readonly timeoutMs: number;

    // Resource allocation
    readonly availableCredits: number;
    readonly availableTime: number;
    readonly concurrencyBudget: number;

    // Security context
    readonly securityContext: SecurityContext;
}

export interface StepExecutionResult {
    // Identity and status
    readonly stepId: string;
    readonly runId: string;
    readonly status: StepStatus;
    readonly strategyUsed: ExecutionStrategy;

    // Results
    readonly output: unknown;
    readonly intermediateResults: IntermediateResult[];
    readonly contextUpdates: ContextUpdate[];

    // Quality metrics
    readonly qualityScore: number;         // 0.0 - 1.0
    readonly confidenceLevel: number;      // 0.0 - 1.0
    readonly validationResults: ValidationResult[];

    // Resource tracking
    readonly resourceUsage: StepResourceUsage;
    readonly performanceMetrics: StepPerformanceMetrics;

    // Error handling
    readonly warnings: ExecutionWarning[];
    readonly recoverableErrors: RecoverableError[];

    // Strategy evolution data
    readonly evolutionRecommendations: EvolutionRecommendation[];
}

// ================================
// Context Management
// ================================

export interface RunContext {
    // Static context (immutable for duration of run)
    readonly runId: string;
    readonly parentRunId?: string;
    readonly routineManifest: RoutineManifest;
    readonly permissions: Permission[];
    readonly resourceLimits: ResourceLimits;

    // Dynamic variables (mutable)
    variables: Map<string, ContextVariable>;
    intermediate: Map<string, unknown>;      // Temporary step results
    exports: ExportDeclaration[];           // Declared outputs

    // Sensitivity tracking
    sensitivityMap: Map<string, DataSensitivity>;

    // Hierarchy management
    createChild(overrides?: Partial<RunContextInit>): RunContext;
    inheritFromParent(parentContext: RunContext): void;
    resolveVariableConflicts(conflicts: VariableConflict[]): Resolution[];
    markForExport(key: string, destination: ExportDestination): void;
}

export interface ContextVariable {
    readonly key: string;
    readonly value: unknown;
    readonly source: VariableSource;       // parent, step, user, system
    readonly timestamp: Date;
    readonly sensitivity: DataSensitivity;
    readonly mutable: boolean;
}

export enum VariableSource {
    PARENT = "parent",
    STEP = "step",
    USER = "user",
    SYSTEM = "system"
}

export enum ConflictResolutionStrategy {
    PARENT_WINS = "parent_wins",           // Parent value takes precedence
    CHILD_WINS = "child_wins",             // Child value takes precedence  
    MERGE_OBJECTS = "merge_objects",       // Deep merge for objects
    ARRAY_CONCAT = "array_concat",         // Concatenate arrays
    TIMESTAMP_LATEST = "timestamp_latest", // Most recent value wins
    EXPLICIT_MAPPING = "explicit_mapping"  // Use provided mapping
}

export interface VariableConflict {
    readonly key: string;
    readonly parentValue: unknown;
    readonly childValue: unknown;
    readonly resolutionStrategy: ConflictResolutionStrategy;
}

export interface Resolution {
    readonly key: string;
    readonly resolvedValue: unknown;
    readonly strategy: ConflictResolutionStrategy;
    readonly reason: string;
}

export interface ExportDeclaration {
    readonly key: string;
    readonly destination: ExportDestination;
    readonly sensitivity: DataSensitivity;
    readonly transformation?: string;
}

export enum ExportDestination {
    PARENT_CONTEXT = "parent_context",
    BLACKBOARD = "blackboard",
    PERSISTENT_STORE = "persistent_store",
    EXTERNAL_SYSTEM = "external_system"
}

// ================================
// MCP Integration
// ================================

export interface RunRoutineMcpTool {
    name: "run_routine";
    description: "Execute a routine with specified inputs and configuration";

    // MCP tool parameters
    parameters: {
        routineId: string;                     // Which routine to execute
        inputs?: Record<string, unknown>;     // Input parameters
        isAsync?: boolean;                     // Sync vs async execution
        resourceLimits?: ResourceLimitOverrides; // Optional limit overrides
        priority?: ExecutionPriority;         // Execution priority
    };

    // MCP tool response
    response: {
        success: boolean;
        runId?: string;                        // For async execution tracking
        result?: Record<string, unknown>;     // For sync execution results
        resourceUsage?: ResourceUsage;        // Actual resource consumption
        error?: string;                        // Error message if failed
    };
}

export interface McpToolResponse {
    readonly success: boolean;
    readonly data?: Record<string, unknown>;
    readonly error?: string;
    readonly metadata?: Record<string, unknown>;
}

// ================================
// Supporting Types
// ================================

export interface ValidationRule {
    readonly type: string;
    readonly configuration: Record<string, unknown>;
    readonly errorMessage: string;
}

export interface JsonSchema {
    readonly type: string;
    readonly properties?: Record<string, JsonSchema>;
    readonly required?: string[];
    readonly additionalProperties?: boolean;
}

export interface IntermediateResult {
    readonly stepId: string;
    readonly timestamp: Date;
    readonly data: unknown;
    readonly type: string;
}

export interface ContextUpdate {
    readonly key: string;
    readonly value: unknown;
    readonly operation: UpdateOperation;
    readonly timestamp: Date;
}

export enum UpdateOperation {
    SET = "set",
    APPEND = "append",
    DELETE = "delete",
    MERGE = "merge"
}

export interface ValidationResult {
    readonly valid: boolean;
    readonly errors: string[];
    readonly warnings: string[];
    readonly rule: string;
}

export interface ExecutionMetrics {
    readonly startTime: Date;
    readonly endTime?: Date;
    readonly duration: number;
    readonly stepCount: number;
    readonly successRate: number;
    readonly averageStepDuration: number;
}

export interface StrategyEvolutionReport {
    readonly originalStrategy: StrategyType;
    readonly finalStrategy: StrategyType;
    readonly evolutionSteps: EvolutionStep[];
    readonly qualityImprovement: number;
}

export interface EvolutionStep {
    readonly fromStrategy: StrategyType;
    readonly toStrategy: StrategyType;
    readonly reason: string;
    readonly confidence: number;
}

export interface RunState {
    readonly runId: string;
    readonly currentStep: string;
    readonly completedSteps: string[];
    readonly context: Record<string, unknown>;
    readonly timestamp: Date;
}

export interface Checkpoint {
    readonly checkpointId: string;
    readonly runId: string;
    readonly stepId: string;
    readonly state: Record<string, unknown>;
    readonly timestamp: Date;
    readonly recoverable: boolean;
}

export interface StepResourceUsage {
    readonly creditsUsed: number;
    readonly timeElapsedMs: number;
    readonly memoryUsedMB: number;
    readonly toolCallsMade: number;
}

export interface StepPerformanceMetrics {
    readonly executionTime: number;
    readonly queueTime: number;
    readonly networkLatency: number;
    readonly computeEfficiency: number;
}

export interface ExecutionWarning {
    readonly type: string;
    readonly message: string;
    readonly severity: WarningSeverity;
    readonly recommendation?: string;
}

export enum WarningSeverity {
    LOW = "low",
    MEDIUM = "medium",
    HIGH = "high"
}

export interface RecoverableError {
    readonly errorId: string;
    readonly type: string;
    readonly message: string;
    readonly recoveryStrategy?: string;
    readonly retryCount: number;
}

export interface EvolutionRecommendation {
    readonly type: string;
    readonly description: string;
    readonly confidence: number;
    readonly expectedImprovement: number;
}

export interface RunContextInit {
    readonly runId: string;
    readonly parentRunId?: string;
    readonly routineManifest: RoutineManifest;
    readonly permissions: Permission[];
    readonly resourceLimits: ResourceLimits;
}

// ================================
// Utility Types
// ================================

export type JSONValue =
    | string
    | number
    | boolean
    | null
    | JSONValue[]
    | { [key: string]: JSONValue };

export type DeepReadonly<T> = {
    readonly [P in keyof T]: T[P] extends object ? DeepReadonly<T[P]> : T[P];
};

export type Optional<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;
export type Required<T, K extends keyof T> = T & { [P in K]-?: T[P] };

export interface Timestamp {
    readonly createdAt: Date;
    readonly updatedAt: Date;
}

export interface Identifiable {
    readonly id: string;
}

export interface Versioned {
    readonly version: string;
}

export interface Describable {
    readonly name: string;
    readonly description?: string;
} 
