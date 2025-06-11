/**
 * Communication types for three-tier execution architecture
 * 
 * These types define the interfaces for communication between tiers,
 * ensuring loose coupling and clear contracts.
 */

import type { 
    SwarmConfig, 
    SwarmState, 
    RunConfig, 
    RunState,
    RunStatus,
    SwarmStatus,
    ResourceUsage,
    SessionUser,
} from "@vrooli/shared";

/**
 * Generic tier communication interface
 * All tiers implement this interface to ensure consistent communication
 */
export interface TierCommunicationInterface {
    /**
     * Get the current status of this tier
     */
    getStatus(): Promise<TierStatus>;
    
    /**
     * Handle incoming requests from other tiers
     */
    handleRequest(request: TierRequest): Promise<TierResponse>;
    
    /**
     * Shutdown the tier gracefully
     */
    shutdown(): Promise<void>;
}

/**
 * Status information for a tier
 */
export interface TierStatus {
    tier: "tier1" | "tier2" | "tier3";
    status: "active" | "paused" | "stopping" | "stopped";
    activeProcesses: number;
    lastActivity: Date;
    errors?: string[];
}

/**
 * Generic request structure for inter-tier communication
 */
export interface TierRequest {
    id: string;
    type: string;
    source: "tier1" | "tier2" | "tier3" | "external";
    target: "tier1" | "tier2" | "tier3";
    payload: unknown;
    metadata?: {
        userId?: string;
        correlationId?: string;
        timestamp?: Date;
    };
}

/**
 * Generic response structure for inter-tier communication
 */
export interface TierResponse {
    requestId: string;
    success: boolean;
    data?: unknown;
    error?: {
        code: string;
        message: string;
        details?: unknown;
    };
}

/**
 * Tier 1 specific communication interface
 */
export interface Tier1Communication extends TierCommunicationInterface {
    /**
     * Start a new swarm
     */
    startSwarm(config: {
        swarmId: string;
        name: string;
        description: string;
        goal: string;
        resources: {
            maxCredits: number;
            maxTokens: number;
            maxTime: number;
            tools: Array<{ name: string; description: string }>;
        };
        config: {
            model: string;
            temperature: number;
            autoApproveTools: boolean;
            parallelExecutionLimit: number;
        };
        userId: string;
        organizationId?: string;
        parentSwarmId?: string;
        leaderBotId?: string;
    }): Promise<void>;
    
    /**
     * Request run execution from a swarm
     */
    requestRunExecution(request: {
        swarmId: string;
        runId: string;
        routineVersionId: string;
        inputs: Record<string, unknown>;
        config: RunConfig;
    }): Promise<void>;
    
    /**
     * Get swarm status
     */
    getSwarmStatus(swarmId: string): Promise<{
        status: SwarmStatus;
        progress?: number;
        currentPhase?: string;
        activeRuns?: number;
        completedRuns?: number;
        errors?: string[];
    }>;
    
    /**
     * Cancel a swarm
     */
    cancelSwarm(swarmId: string, userId: string, reason?: string): Promise<void>;
}

/**
 * Tier 2 specific communication interface
 */
export interface Tier2Communication extends TierCommunicationInterface {
    /**
     * Start a new run
     */
    startRun(config: {
        runId: string;
        routineVersionId: string;
        inputs: Record<string, unknown>;
        context: RunExecutionContext;
        sessionUser: SessionUser;
    }): Promise<void>;
    
    /**
     * Get run status
     */
    getRunStatus(runId: string): Promise<{
        status: RunStatus;
        progress?: number;
        currentStep?: string;
        outputs?: Record<string, unknown>;
        errors?: string[];
    }>;
    
    /**
     * Cancel a run
     */
    cancelRun(runId: string, reason?: string): Promise<void>;
}

/**
 * Tier 3 specific communication interface
 */
export interface Tier3Communication extends TierCommunicationInterface {
    /**
     * Execute a single step
     */
    executeStep(request: StepExecutionRequest): Promise<StepExecutionResult>;
    
    /**
     * Get available strategies
     */
    getAvailableStrategies(): Promise<string[]>;
    
    /**
     * Get resource usage for a session
     */
    getResourceUsage(sessionId: string): Promise<ResourceUsage>;
}

/**
 * Context passed during run execution
 */
export interface RunExecutionContext {
    swarmId?: string;
    parentRunId?: string;
    strategy?: string;
    model: string;
    maxSteps: number;
    timeout: number;
    resourceLimits: {
        maxCredits: number;
        maxTokens: number;
        maxTime: number;
    };
}

/**
 * Request to execute a single step
 */
export interface StepExecutionRequest {
    runId: string;
    stepId: string;
    nodeId: string;
    inputs: Record<string, unknown>;
    context: StepExecutionContext;
    sessionUser: SessionUser;
}

/**
 * Context for step execution
 */
export interface StepExecutionContext {
    strategy: string;
    model?: string;
    temperature?: number;
    maxTokens?: number;
    timeout?: number;
    retryCount?: number;
}

/**
 * Result of step execution
 */
export interface StepExecutionResult {
    stepId: string;
    status: "completed" | "failed" | "skipped";
    outputs?: Record<string, unknown>;
    error?: {
        code: string;
        message: string;
        recoverable: boolean;
    };
    resourceUsage: ResourceUsage;
    duration: number;
}

/**
 * Event data structures for cross-tier communication
 */
export interface TierEvent {
    id: string;
    type: string;
    source: "tier1" | "tier2" | "tier3";
    timestamp: Date;
    data: unknown;
}

export interface SwarmEvent extends TierEvent {
    source: "tier1";
    swarmId: string;
}

export interface RunEvent extends TierEvent {
    source: "tier2";
    runId: string;
}

export interface StepEvent extends TierEvent {
    source: "tier3";
    stepId: string;
}