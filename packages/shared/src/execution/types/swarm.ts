// AI_CHECK: TYPE_SAFETY=2 | LAST: 2025-06-26
/**
 * Core type definitions for Tier 1: Coordination Intelligence
 * These types define the swarm coordination and metacognitive capabilities
 */

import type { SessionUser } from "../../api/types.js";
import type { ChatConfigObject } from "../../shape/configs/chat.js";
import type { ExecutionStates } from "./core.js";

// ExecutionStates moved to core.ts to avoid duplication
// Import from core.ts when needed

// SwarmEventType moved to events.ts to avoid duplication
// Import from events.ts when needed

/**
 * Simplified swarm event matching implementation
 * Complex event data is passed through flexible payloads
 */
export interface SwarmEvent {
    type: string;  // Any string, not limited to SwarmEventTypeValues
    timestamp?: Date;
    metadata?: Record<string, unknown>;
    conversationId?: string;
    sessionUser?: SessionUser;
    goal?: string;
    payload?: Record<string, unknown>;
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
    norms: SwarmAgentNorm[];
}

/**
 * Agent norm definition (MOISE+ compatible)
 */
export interface SwarmAgentNorm {
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
    state: ExecutionStates;
    metadata: SwarmMetadata;
}

/**
 * Swarm metadata as actually used
 */
export interface SwarmMetadata {
    goal?: string;
    resources?: any;
    agents?: Array<{ id: string; role: string }>;
    startTime?: string;
    [key: string]: unknown;  // Allow any additional fields
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
    state: ExecutionStates;
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
