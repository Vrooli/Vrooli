/**
 * Core type definitions for Tier 1: Coordination Intelligence
 * These types define the swarm coordination and metacognitive capabilities
 */

import { type ResourceUsedFor } from "../../shape/configs/base.js";

/**
 * Swarm lifecycle states
 * Maintains compatibility with existing SwarmStateMachine
 */
export enum SwarmState {
    UNINITIALIZED = "UNINITIALIZED",
    FORMING = "FORMING",
    PLANNING = "PLANNING",
    STARTING = "STARTING",
    EXECUTING = "EXECUTING",
    RUNNING = "RUNNING",
    IDLE = "IDLE",
    PAUSED = "PAUSED",
    SUSPENDED = "SUSPENDED",
    COMPLETED = "COMPLETED",
    STOPPED = "STOPPED",
    FAILED = "FAILED",
    TERMINATED = "TERMINATED"
}

/**
 * Swarm event types for event-driven coordination
 */
export enum SwarmEventType {
    // Lifecycle events
    SWARM_CREATED = "SWARM_CREATED",
    SWARM_STARTED = "SWARM_STARTED",
    SWARM_STOPPED = "SWARM_STOPPED",
    SWARM_FAILED = "SWARM_FAILED",
    SWARM_COMPLETED = "SWARM_COMPLETED",
    SWARM_TERMINATED = "SWARM_TERMINATED",
    STATE_CHANGED = "STATE_CHANGED",
    
    // Coordination events
    TEAM_FORMED = "TEAM_FORMED",
    TEAM_UPDATED = "TEAM_UPDATED",
    TEAM_DISBANDED = "TEAM_DISBANDED",
    AGENT_JOINED = "AGENT_JOINED",
    AGENT_LEFT = "AGENT_LEFT",
    
    // Goal management events
    GOAL_ASSIGNED = "GOAL_ASSIGNED",
    GOAL_COMPLETED = "GOAL_COMPLETED",
    GOAL_FAILED = "GOAL_FAILED",
    SUBTASK_CREATED = "SUBTASK_CREATED",
    
    // Resource events
    RESOURCE_ALLOCATED = "RESOURCE_ALLOCATED",
    RESOURCE_RELEASED = "RESOURCE_RELEASED",
    RESOURCE_EXHAUSTED = "RESOURCE_EXHAUSTED",
    
    // Communication events
    MESSAGE_SENT = "MESSAGE_SENT",
    CONSENSUS_REACHED = "CONSENSUS_REACHED",
    CONFLICT_DETECTED = "CONFLICT_DETECTED",
    
    // Decision events
    DECISION_PROPOSED = "DECISION_PROPOSED",
    DECISION_EXECUTED = "DECISION_EXECUTED",
    STRATEGY_ADAPTED = "STRATEGY_ADAPTED"
}

/**
 * Base event interface for all swarm events
 */
export interface SwarmEvent {
    type: SwarmEventType;
    timestamp: Date;
    swarmId: string;
    metadata?: Record<string, unknown>;
}

/**
 * Swarm resource definition
 * Compatible with existing ChatConfigObject structure
 */
export interface SwarmResource {
    id: string;
    type: "Api" | "Team" | "Project" | "Routine" | "Code" | "Note" | "Prompt" | "Standard";
    name: string;
    description?: string;
    listId?: string;
    usedFor?: ResourceUsedFor | ResourceUsedFor[];
    runnableObject?: unknown; // Can be Routine, Code, etc.
}

/**
 * Swarm subtask definition
 * Compatible with existing ChatConfigObject structure
 */
export interface SwarmSubTask {
    description: string;
    status: "ready" | "pending" | "inProgress" | "completed" | "failed" | "cancelled";
    result?: string;
}

/**
 * Blackboard item for shared memory
 */
export interface BlackboardItem {
    id: string;
    value: string;
    createdAt: Date;
    updatedAt: Date;
    createdBy: string; // Agent ID
}

/**
 * Tool call scheduling configuration
 */
export interface SchedulingConfig {
    mode: "BATCH" | "PARALLEL" | "SEQUENTIAL";
    maxConcurrency?: number;
    timeout?: number;
}

/**
 * Pending tool call entry
 */
export interface PendingToolCallEntry {
    name: string;
    arguments?: Record<string, unknown>;
    approved?: boolean;
    id?: string;
}

/**
 * Resource usage limits
 */
export interface ResourceLimits {
    maxTokens?: number;
    maxTime?: number;
    maxCost?: number;
    maxApiCalls?: number;
}

/**
 * Chat configuration object
 * Maintains full compatibility with existing system
 */
export interface ChatConfigObject {
    // Core configuration
    goal: string;
    subtasks: SwarmSubTask[];
    resources: SwarmResource[];
    
    // Shared memory
    blackboard: BlackboardItem[];
    
    // Execution configuration
    limits: ResourceLimits;
    scheduling: SchedulingConfig;
    pendingToolCalls: PendingToolCallEntry[];
    
    // Metadata
    createdAt: Date;
    updatedAt: Date;
}

/**
 * Agent role definition (MOISE+ compatible)
 */
export interface AgentRole {
    id: string;
    name: string;
    description: string;
    capabilities: string[];
    responsibilities: string[];
    norms: AgentNorm[];
}

/**
 * Agent norm definition (MOISE+ compatible)
 */
export interface AgentNorm {
    type: "obligation" | "permission" | "prohibition";
    target: string; // Action or resource
    condition?: string;
}

/**
 * Agent definition within a swarm
 */
export interface SwarmAgent {
    id: string;
    name: string;
    role: AgentRole;
    state: "active" | "idle" | "failed";
    capabilities: string[];
    currentTask?: string;
    performance: AgentPerformance;
}

/**
 * Agent performance metrics
 */
export interface AgentPerformance {
    tasksCompleted: number;
    tasksFailled: number;
    averageCompletionTime: number;
    successRate: number;
    resourceEfficiency: number;
    collaborationScore?: number;
    averageExecutionTime?: number;
}

/**
 * Swarm team structure
 */
export interface SwarmTeam {
    id: string;
    name: string;
    goal: string;
    agents: SwarmAgent[];
    hierarchy: TeamHierarchy;
    formation: Date;
    status: "forming" | "active" | "disbanding" | "disbanded";
}

/**
 * Team hierarchy structure
 */
export interface TeamHierarchy {
    leader?: string; // Agent ID
    structure: "flat" | "hierarchical" | "matrix";
    relationships: Array<{
        from: string; // Agent ID
        to: string; // Agent ID
        type: "reports_to" | "collaborates_with" | "supervises";
    }>;
}

/**
 * Swarm configuration for Tier 1
 */
export interface SwarmConfiguration {
    id: string;
    chatConfig: ChatConfigObject;
    teams: SwarmTeam[];
    state: SwarmState;
    metadata: SwarmMetadata;
}

/**
 * Swarm metadata
 */
export interface SwarmMetadata {
    conversationId: string;
    initiatingUserId: string;
    createdAt: Date;
    updatedAt: Date;
    version: string;
}

/**
 * Metacognitive reasoning result
 */
export interface ReasoningResult {
    decision: string;
    rationale: string;
    confidence: number;
    alternatives: Array<{
        decision: string;
        rationale: string;
        confidence: number;
    }>;
    learnedPatterns?: string[];
}

/**
 * Strategy selection result
 */
export interface StrategySelectionResult {
    selectedStrategy: string;
    reasoning: ReasoningResult;
    fallbackStrategies: string[];
    contextFactors: Record<string, unknown>;
}

/**
 * Swarm configuration
 */
export interface SwarmConfig {
    maxAgents: number;
    minAgents: number;
    consensusThreshold: number;
    decisionTimeout: number;
    adaptationInterval: number;
    resourceOptimization: boolean;
    learningEnabled: boolean;
    maxBudget?: number;
    maxDuration?: number;
}

/**
 * Agent capability type
 */
export type AgentCapability = 
    | "reasoning"
    | "planning"
    | "execution"
    | "monitoring"
    | "learning"
    | "communication"
    | "negotiation"
    | "resource_management";

/**
 * Resource allocation
 */
export interface ResourceAllocation {
    id: string;
    swarmId: string;
    agentId: string;
    resourceType: string;
    amount: number;
    purpose: string;
    expiresAt?: Date;
}

/**
 * Team constraints
 */
export interface TeamConstraints {
    requiredCapabilities?: AgentCapability[];
    maxSize?: number;
    minSize?: number;
    budgetLimit?: number;
}

/**
 * Consensus request
 */
export interface ConsensusRequest {
    id: string;
    swarmId: string;
    propositions: string[];
    timeout: number;
    requiredThreshold: number;
}

/**
 * Consensus result
 */
export interface ConsensusResult {
    requestId: string;
    results: number[];
    threshold: number;
    reached: boolean;
    participantCount: number;
}

/**
 * Swarm decision
 */
export interface SwarmDecision {
    id: string;
    timestamp: Date;
    decision: string;
    rationale: string;
    outcome?: string;
}

/**
 * Swarm performance
 */
export interface SwarmPerformance {
    goalProgress: number;
    resourceEfficiency: number;
    teamCohesion: number;
    adaptationRate: number;
    learningRate: number;
    overallEffectiveness: number;
}

/**
 * Metacognitive reflection
 */
export interface MetacognitiveReflection {
    swarmId: string;
    timestamp: Date;
    performance: SwarmPerformance;
    learnings: string[];
    adaptations: string[];
    confidence: number;
}

/**
 * Team formation
 */
export interface TeamFormation {
    id: string;
    swarmId: string;
    agents: SwarmAgent[];
    purpose?: string;
    constraints?: TeamConstraints;
    createdAt: Date;
}

/**
 * Extended swarm definition
 */
export interface Swarm {
    id: string;
    name: string;
    description: string;
    state: SwarmState;
    config: SwarmConfig;
    team?: TeamFormation;
    metadata: Record<string, unknown>;
    createdAt: Date;
    updatedAt?: Date;
    completedAt?: Date;
}
