/**
 * Core type definitions for Tier 1: Coordination Intelligence
 * These types define the swarm coordination and metacognitive capabilities
 */

import type { ChatConfigObject } from "../../shape/configs/chat.js";

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
    RESOURCE_RESERVED = "RESOURCE_RESERVED",
    RESOURCE_UNRESERVED = "RESOURCE_UNRESERVED",

    // Parent-child swarm events
    CHILD_SWARM_SPAWNED = "CHILD_SWARM_SPAWNED",
    CHILD_SWARM_COMPLETED = "CHILD_SWARM_COMPLETED",
    CHILD_SWARM_FAILED = "CHILD_SWARM_FAILED",
    PARENT_SWARM_NOTIFIED = "PARENT_SWARM_NOTIFIED",

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
export interface SwarmResourceAllocation {
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
 * Resource allocation for child swarms
 */
export interface SwarmResourceAllocation {
    credits: number;
    tokens: number;
    time: number; // milliseconds
}

/**
 * Child swarm reservation tracking
 */
export interface ChildSwarmReservation {
    childSwarmId: string;
    reserved: SwarmResourceAllocation;
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
    
    // Parent-child relationship fields
    parentSwarmId?: string;
    childSwarmIds: string[];
    
    // Resource management for hierarchical swarms
    resources: {
        allocated: SwarmResourceAllocation;
        consumed: SwarmResourceAllocation;
        remaining: SwarmResourceAllocation;
        reservedByChildren: SwarmResourceAllocation;
        childReservations: ChildSwarmReservation[];
    };
    
    // Swarm performance metrics
    metrics: {
        tasksCompleted: number;
        tasksFailed: number;
        avgTaskDuration: number;
        resourceEfficiency: number;
    };
    
    // Error tracking
    errors?: string[];
}
