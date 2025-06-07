/**
 * Core type definitions for context and memory management
 * These types are shared across all three tiers
 */

/**
 * Base context interface for all tiers
 */
export interface BaseContext {
    id: string;
    tier: 1 | 2 | 3;
    timestamp: Date;
    metadata: ContextMetadata;
}

/**
 * Context metadata
 */
export interface ContextMetadata {
    userId: string;
    sessionId: string;
    requestId: string;
    parentContextId?: string;
    tags: string[];
}

/**
 * Tier 1 Coordination Context
 */
export interface CoordinationContext extends BaseContext {
    tier: 1;
    swarmId: string;
    conversationId: string;
    teams: string[]; // Team IDs
    sharedMemory: SharedMemory;
    coordinationState: CoordinationState;
}

/**
 * Shared memory for swarm coordination
 */
export interface SharedMemory {
    blackboard: Record<string, unknown>;
    decisions: DecisionRecord[];
    consensus: ConsensusRecord[];
    conflicts: ConflictRecord[];
}

/**
 * Decision record
 */
export interface DecisionRecord {
    id: string;
    timestamp: Date;
    agentId: string;
    decision: string;
    rationale: string;
    confidence: number;
}

/**
 * Consensus record
 */
export interface ConsensusRecord {
    id: string;
    timestamp: Date;
    topic: string;
    participants: string[];
    result: string;
    agreement: number; // 0-1
}

/**
 * Conflict record
 */
export interface ConflictRecord {
    id: string;
    timestamp: Date;
    type: "goal" | "resource" | "strategy" | "priority";
    parties: string[];
    description: string;
    resolution?: string;
    resolved: boolean;
}

/**
 * Coordination state
 */
export interface CoordinationState {
    phase: "planning" | "executing" | "monitoring" | "adapting";
    activeGoals: string[];
    completedGoals: string[];
    blockedGoals: string[];
}

/**
 * Tier 2 Process Context
 */
export interface ProcessContext extends BaseContext {
    tier: 2;
    runId: string;
    routineId: string;
    navigationState: NavigationState;
    processMemory: ProcessMemory;
    orchestrationState: OrchestrationState;
}

/**
 * Navigation state for process execution
 */
export interface NavigationState {
    currentLocation: string;
    locationStack: string[];
    visitedLocations: Set<string>;
    branchStates: Record<string, BranchState>;
}

/**
 * Branch state tracking
 */
export interface BranchState {
    id: string;
    status: "pending" | "active" | "completed" | "failed";
    parallel: boolean;
    startedAt?: Date;
    completedAt?: Date;
}

/**
 * Process memory
 */
export interface ProcessMemory {
    variables: Record<string, unknown>;
    checkpoints: string[]; // Checkpoint IDs
    optimizations: string[]; // Applied optimization IDs
    performanceData: PerformanceData;
}

/**
 * Performance data
 */
export interface PerformanceData {
    stepDurations: Record<string, number>;
    resourceUsage: Record<string, number>;
    bottlenecks: string[];
    efficiencyScore: number;
}

/**
 * Orchestration state
 */
export interface OrchestrationState {
    phase: "initializing" | "running" | "optimizing" | "completing";
    activeSteps: string[];
    pendingSteps: string[];
    completedSteps: string[];
    failedSteps: string[];
}

/**
 * Tier 3 Execution Context
 */
export interface ExecutionContext extends BaseContext {
    tier: 3;
    stepId: string;
    strategyType: string;
    executionMemory: ExecutionMemory;
    adaptationState: AdaptationState;
}

/**
 * Execution memory
 */
export interface ExecutionMemory {
    inputs: Record<string, unknown>;
    outputs: Record<string, unknown>;
    toolCalls: ToolCallRecord[];
    strategyData: Record<string, unknown>;
    learningData: LearningData;
}

/**
 * Tool call record
 */
export interface ToolCallRecord {
    id: string;
    timestamp: Date;
    toolName: string;
    parameters: Record<string, unknown>;
    result?: unknown;
    error?: string;
    duration: number;
}

/**
 * Learning data
 */
export interface LearningData {
    patterns: Pattern[];
    feedback: FeedbackRecord[];
    adaptations: Adaptation[];
}

/**
 * Pattern definition
 */
export interface Pattern {
    id: string;
    type: "success" | "failure" | "optimization";
    description: string;
    frequency: number;
    impact: number;
}

/**
 * Feedback record
 */
export interface FeedbackRecord {
    id: string;
    timestamp: Date;
    type: "user" | "system" | "automated";
    rating: number; // 0-1
    comment?: string;
}

/**
 * Adaptation record
 */
export interface Adaptation {
    id: string;
    timestamp: Date;
    type: "parameter" | "strategy" | "resource";
    before: unknown;
    after: unknown;
    reason: string;
    impact: number; // -1 to 1
}

/**
 * Adaptation state
 */
export interface AdaptationState {
    mode: "learning" | "optimizing" | "stable";
    currentStrategy: string;
    alternativeStrategies: string[];
    confidenceThreshold: number;
    adaptationRate: number;
}

/**
 * Cross-tier context for communication
 */
export interface CrossTierContext {
    requestId: string;
    userId: string;
    sessionId: string;
    tier1Context?: CoordinationContext;
    tier2Context?: ProcessContext;
    tier3Context?: ExecutionContext;
    sharedData: Record<string, unknown>;
    constraints: ContextConstraints;
}

/**
 * Context constraints
 */
export interface ContextConstraints {
    maxMemorySize: number;
    maxExecutionTime: number;
    maxCost: number;
    securityLevel: "public" | "private" | "restricted";
    allowedOperations: string[];
}

/**
 * Context persistence
 */
export interface ContextSnapshot {
    id: string;
    contextType: "coordination" | "process" | "execution" | "crossTier";
    context: BaseContext | CrossTierContext;
    timestamp: Date;
    size: number;
    compressed: boolean;
}
