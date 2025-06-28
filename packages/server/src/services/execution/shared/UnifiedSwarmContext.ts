/**
 * Unified Swarm Context Type System - Data-Driven State Model for Emergent Capabilities
 * 
 * This module defines the unified context types that enable emergent swarm intelligence.
 * Unlike traditional hard-coded state management, this system ensures that:
 * 
 * 1. **All swarm behavior is data-driven** - Defined by configuration objects, not code
 * 2. **Agent capabilities emerge from events** - No hardcoded agent logic
 * 3. **Context enables self-improvement** - Historical data drives optimization
 * 4. **Live updates propagate instantly** - Real-time configuration changes
 * 
 * ## Key Design Principles:
 * 
 * - **Single Source of Truth**: One unified context type replaces all fragmented contexts
 * - **Immutable with Versioning**: Atomic updates with rollback capability
 * - **Event-Driven Updates**: All changes propagate through Redis pub/sub
 * - **Emergent-Friendly**: Structure supports agent-driven modifications
 * 
 * ## Usage in Emergent Architecture:
 * 
 * ```typescript
 * // Agents can modify swarm behavior through data, not code
 * const optimizationAgent = {
 *   subscribedEvents: ["swarm/resource/exhausted"],
 *   routine: "analyze_and_optimize_allocation_strategy",
 *   configUpdates: {
 *     "policy.resource.allocation.strategy": "elastic_with_preemption"
 *   }
 * };
 * ```
 * 
 * @see SwarmContextManager - Uses these types for unified state management
 * @see /docs/architecture/execution/swarm-state-management-redesign.md - Complete context design
 */

import {
    type CoreResourceAllocation,
    type ExecutionResourceUsage,
    type SwarmId,
} from "@vrooli/shared";

/**
 * Resource pool definition for swarm-wide resource management
 */
export interface ResourcePool {
    /** Total credits available/allocated */
    credits: string; // BigInt as string to handle large values

    /** Maximum execution duration in milliseconds */
    durationMs: number;

    /** Maximum memory allocation in MB */
    memoryMB: number;

    /** Maximum concurrent executions */
    concurrentExecutions: number;

    /** Available AI models and their limits */
    models: {
        [modelName: string]: {
            tokensPerMinute: number;
            maxConcurrentRequests: number;
            costPerToken: string; // BigInt as string
        };
    };

    /** Available tools and their resource costs */
    tools: {
        [toolName: string]: {
            creditsPerUse: string; // BigInt as string
            avgExecutionTimeMs: number;
            memoryRequirementMB: number;
        };
    };
}

/**
 * Resource allocation tracking for hierarchical resource management
 */
export interface ResourceAllocation {
    /** Unique allocation identifier */
    id: string;

    /** What entity this allocation is for (routine, step, etc.) */
    consumerId: string;

    /** Type of consumer (routine, step, agent, etc.) */
    consumerType: "routine" | "step" | "agent" | "parallel_branch";

    /** Allocated resource limits */
    allocation: CoreResourceAllocation;

    /** Current usage against allocation */
    usage: ExecutionResourceUsage;

    /** Parent allocation (for hierarchical tracking) */
    parentAllocationId?: string;

    /** Allocation timestamp */
    allocatedAt: Date;

    /** Expected completion time (for resource planning) */
    expectedCompletionAt?: Date;

    /** Priority for resource contention resolution */
    priority: "low" | "medium" | "high" | "critical";
}

/**
 * Security policy configuration - fully data-driven for agent modification
 */
export interface SecurityPolicy {
    /** Allowed actions for different agent roles */
    permissions: {
        [roleName: string]: {
            canExecuteTools: string[]; // Tool names
            canAccessResources: string[]; // Resource types
            canModifyContext: string[]; // Context fields
            maxResourceAllocation: CoreResourceAllocation;
        };
    };

    /** Security scanning configuration */
    scanning: {
        enabledScanners: string[]; // "xss", "pii", "malware", etc.
        blockOnViolation: boolean;
        alertingThresholds: {
            [scannerName: string]: number; // Severity threshold
        };
    };

    /** Tool approval requirements */
    toolApproval: {
        requireApprovalForTools: string[]; // Tool names requiring approval
        autoApproveForRoles: string[]; // Roles that can auto-approve
        approvalTimeoutMs: number;
    };
}

/**
 * Resource allocation policy - enables agent-driven resource optimization
 */
export interface ResourcePolicy {
    /** Base allocation strategy configuration */
    allocation: {
        strategy: "strict" | "elastic" | "predictive" | "agent_optimized";

        /** Tier-to-tier allocation percentages (data-driven) */
        tierAllocationRatios: {
            tier1ToTier2: number; // Swarm → Routine
            tier2ToTier3: number; // Routine → Step
        };

        /** Buffer percentages for emergency allocation */
        bufferPercentages: {
            emergency: number;    // Reserve for critical failures
            optimization: number; // Reserve for agent improvements
            parallel: number;     // Reserve for parallel execution
        };

        /** Resource contention resolution */
        contention: {
            strategy: "priority_based" | "fair_share" | "elastic_preemption";
            preemptionEnabled: boolean;
            priorityWeights: {
                [priority: string]: number;
            };
        };
    };

    /** Performance thresholds that trigger agent optimizations */
    thresholds: {
        resourceUtilization: {
            warning: number;     // % utilization to trigger warnings
            critical: number;    // % utilization to trigger emergency actions
            optimization: number; // % utilization to trigger agent optimization
        };

        latency: {
            targetMs: number;    // Target execution latency
            warningMs: number;   // Latency threshold for warnings
            criticalMs: number;  // Latency threshold for escalation
        };

        failureRate: {
            warningPercent: number;  // % failure rate to trigger warnings
            criticalPercent: number; // % failure rate to trigger escalation
        };
    };

    /** Historical data for agent learning */
    history: {
        recentAllocations: ResourceAllocation[];
        performanceMetrics: {
            avgUtilization: number;
            peakUtilization: number;
            bottleneckFrequency: { [resource: string]: number };
        };
        optimizationHistory: {
            timestamp: Date;
            agentId: string;
            changeDescription: string;
            performanceImpact: number; // % improvement/degradation
        }[];
    };
}

/**
 * MOISE+ organizational policy for multi-agent coordination
 */
export interface MOISEPolicy {
    /** Organizational structure definition */
    structure: {
        /** Hierarchy levels and roles */
        hierarchy: {
            level: number;
            roles: string[];
            authority: string[]; // What this level can authorize
        }[];

        /** Groups and their members */
        groups: {
            id: string;
            name: string;
            members: string[]; // Agent IDs
            responsibilities: string[];
        }[];

        /** Dependencies between roles/groups */
        dependencies: {
            from: string; // Role or group ID
            to: string;   // Role or group ID
            type: "informational" | "authoritative" | "collaborative";
            conditions?: string[]; // When this dependency applies
        }[];
    };

    /** Functional specifications */
    functional: {
        /** Mission definitions */
        missions: {
            id: string;
            name: string;
            objectives: string[];
            assignedRoles: string[];
            expectedDuration: number;
            priority: "low" | "medium" | "high" | "critical";
        }[];

        /** Goals that emerge from agent coordination */
        goals: {
            id: string;
            description: string;
            type: "achievement" | "maintenance" | "avoid";
            assignedAgents: string[];
            completionCriteria: string[];
            emergentFromMission?: string; // Mission that generated this goal
        }[];
    };

    /** Normative specifications */
    normative: {
        /** Behavioral norms */
        norms: {
            id: string;
            name: string;
            type: "obligation" | "prohibition" | "permission";
            condition: string; // When this norm applies
            action: string;    // What action is required/forbidden/allowed
            priority: number;  // For conflict resolution
        }[];

        /** Sanctions for norm violations */
        sanctions: {
            normId: string;
            violationSeverity: "minor" | "moderate" | "severe";
            action: "warning" | "resource_reduction" | "role_suspension" | "agent_removal";
        }[];
    };
}

/**
 * Configuration settings for swarm behavior
 */
export interface SwarmConfiguration {
    /** Execution timeouts and limits */
    timeouts: {
        routineExecutionMs: number;
        stepExecutionMs: number;
        approvalTimeoutMs: number;
        idleTimeoutMs: number;
    };

    /** Retry and error handling policies */
    retries: {
        maxRetries: number;
        backoffStrategy: "linear" | "exponential" | "adaptive";
        baseDelayMs: number;
        maxDelayMs: number;
    };

    /** Feature flags for emergent capabilities */
    features: {
        emergentGoalGeneration: boolean;    // Allow agents to create new goals
        adaptiveResourceAllocation: boolean; // Allow resource strategy changes
        crossSwarmCommunication: boolean;   // Allow communication between swarms
        autonomousToolApproval: boolean;    // Allow auto-approval based on patterns
        contextualLearning: boolean;        // Enable context-based learning
    };

    /** Agent coordination settings */
    coordination: {
        maxParallelAgents: number;
        communicationProtocol: "event_driven" | "polling" | "hybrid";
        consensusThreshold: number; // For multi-agent decisions
        leadershipElection: "automatic" | "manual" | "rotating";
    };
}

/**
 * Blackboard item for shared data
 */
export interface BlackboardItem {
    key: string;
    value: any;
    type: "data" | "goal" | "plan" | "observation" | "hypothesis";
    createdBy: string; // Agent ID
    createdAt: Date;
    expiresAt?: Date;
    accessLevel: "public" | "group" | "private";
    tags: string[];
}

/**
 * Shared blackboard for inter-agent communication
 */
export interface BlackboardState {
    /** Key-value storage for shared data - using Record instead of Map for JSON serialization */
    items: Record<string, BlackboardItem>;

    /** Subscription patterns for agent notification */
    subscriptions: {
        agentId: string;
        patterns: string[]; // Key patterns to monitor
        eventTypes: string[]; // Types of changes to monitor
    }[];
}

/**
 * Current swarm execution state
 */
export interface SwarmExecutionState {
    /** Overall swarm status */
    status: "initializing" | "running" | "idle" | "paused" | "completed" | "failed" | "terminated";

    /** Active teams and their current assignments */
    teams: {
        id: string;
        name: string;
        leader: string; // Agent ID
        members: string[]; // Agent IDs
        currentMission?: string; // Mission ID
        status: "forming" | "active" | "idle" | "dissolved";
    }[];

    /** Active agents and their current state */
    agents: {
        id: string;
        name: string;
        role: string;
        status: "available" | "busy" | "offline" | "failed";
        currentTask?: string;
        teamId?: string;
        lastActivity: Date;
        capabilities: string[];
        performanceMetrics: {
            tasksCompleted: number;
            averageTaskDuration: number;
            successRate: number;
        };
    }[];

    /** Active routine executions */
    activeRuns: {
        runId: string;
        routineId: string;
        status: string;
        assignedAgents: string[];
        progress: number; // 0-100
        startedAt: Date;
        estimatedCompletionAt?: Date;
    }[];
}

/**
 * Complete unified swarm context - single source of truth
 * 
 * This replaces all fragmented context types and provides a unified model
 * that supports emergent capabilities through data-driven configuration.
 */
export interface UnifiedSwarmContext {
    // Identity and versioning
    readonly swarmId: SwarmId;
    readonly version: number;
    readonly createdAt: Date;
    readonly lastUpdated: Date;
    readonly updatedBy: string; // Agent or user ID that made the last update

    // Resource management
    resources: {
        /** Total resources available to this swarm */
        total: ResourcePool;

        /** Currently allocated resources */
        allocated: ResourceAllocation[];

        /** Available resources (total - allocated) */
        available: ResourcePool;

        /** Resource usage history for optimization */
        usageHistory: {
            timestamp: Date;
            totalUsage: ExecutionResourceUsage;
            utilizationPercent: number;
        }[];
    };

    // Policies (fully data-driven for agent modification)
    policy: {
        /** Security and access control */
        security: SecurityPolicy;

        /** Resource allocation and management */
        resource: ResourcePolicy;

        /** Organizational structure and coordination */
        organizational: MOISEPolicy;
    };

    // Configuration (enables emergent behavior)
    configuration: SwarmConfiguration;

    // Shared state for inter-agent coordination
    blackboard: BlackboardState;

    // Current execution state
    execution: SwarmExecutionState;

    // Context metadata for system management
    metadata: {
        /** Creation context */
        createdBy: string; // User or system ID
        createdFrom?: string; // Parent swarm or template ID

        /** Live update subscription info */
        subscribers: string[]; // Component IDs subscribed to updates

        /** Emergency contacts and escalation */
        emergencyContacts: string[]; // User IDs to notify on critical events

        /** Data retention and cleanup */
        retentionPolicy: {
            keepHistoryDays: number;
            archiveAfterDays: number;
            deleteAfterDays: number;
        };

        /** Performance and debugging info */
        diagnostics: {
            contextSize: number; // Size in bytes
            updateFrequency: number; // Updates per minute
            subscriptionCount: number;
            lastOptimization: Date;
        };
    };
}

/**
 * Context update event for live propagation
 */
export interface ContextUpdateEvent {
    /** Swarm being updated */
    swarmId: SwarmId;

    /** Previous version number */
    previousVersion: number;

    /** New version number */
    newVersion: number;

    /** What changed (for efficient updates) */
    changes: {
        path: string; // JSONPath to changed field
        oldValue: any;
        newValue: any;
        changeType: "created" | "updated" | "deleted";
    }[];

    /** Who made the change */
    updatedBy: string;

    /** When the change occurred */
    timestamp: Date;

    /** Change reason (for audit trail) */
    reason?: string;

    /** Whether this was an emergent change (made by an agent) */
    emergent: boolean;
}

/**
 * Context subscription for live updates
 */
export interface ContextSubscription {
    /** Unique subscription ID */
    id: string;

    /** Swarm being monitored */
    swarmId: SwarmId;

    /** Component that created the subscription */
    subscriberId: string;

    /** JSONPath patterns to monitor */
    watchPaths: string[];

    /** Callback for updates */
    handler: (event: ContextUpdateEvent) => Promise<void>;

    /** Subscription metadata */
    metadata: {
        createdAt: Date;
        lastNotified: Date;
        totalNotifications: number;
        subscriptionType: "state_machine" | "agent" | "ui" | "monitor";
    };
}

/**
 * Context query for retrieving specific context data
 */
export interface ContextQuery {
    /** Swarm to query */
    swarmId: SwarmId;

    /** JSONPath expressions for data to retrieve */
    select?: string[];

    /** Filters to apply */
    where?: {
        path: string;
        operator: "equals" | "contains" | "greaterThan" | "lessThan" | "exists";
        value: any;
    }[];

    /** Version constraints */
    version?: {
        exact?: number;
        minimum?: number;
        maximum?: number;
    };

    /** Include historical data */
    includeHistory?: boolean;
}

/**
 * Context validation result
 */
export interface ContextValidationResult {
    /** Whether the context is valid */
    valid: boolean;

    /** Validation errors */
    errors: {
        path: string;
        message: string;
        severity: "error" | "warning" | "info";
    }[];

    /** Validation warnings */
    warnings: {
        path: string;
        message: string;
        suggestion?: string;
    }[];

    /** Validation performance metrics */
    metrics: {
        validationTimeMs: number;
        rulesChecked: number;
        constraintsValidated: number;
    };
}

/**
 * Type guards for runtime type checking
 */
export function isResourcePool(obj: any): obj is ResourcePool {
    return obj &&
        typeof obj.credits === "string" &&
        typeof obj.durationMs === "number" &&
        typeof obj.memoryMB === "number" &&
        typeof obj.concurrentExecutions === "number" &&
        typeof obj.models === "object" &&
        typeof obj.tools === "object";
}

export function isUnifiedSwarmContext(obj: any): obj is UnifiedSwarmContext {
    return obj &&
        typeof obj.swarmId === "string" &&
        typeof obj.version === "number" &&
        obj.createdAt instanceof Date &&
        obj.lastUpdated instanceof Date &&
        typeof obj.updatedBy === "string" &&
        typeof obj.resources === "object" &&
        typeof obj.policy === "object" &&
        typeof obj.configuration === "object" &&
        typeof obj.blackboard === "object" &&
        typeof obj.execution === "object" &&
        typeof obj.metadata === "object";
}

export function isContextUpdateEvent(obj: any): obj is ContextUpdateEvent {
    return obj &&
        typeof obj.swarmId === "string" &&
        typeof obj.previousVersion === "number" &&
        typeof obj.newVersion === "number" &&
        Array.isArray(obj.changes) &&
        typeof obj.updatedBy === "string" &&
        obj.timestamp instanceof Date &&
        typeof obj.emergent === "boolean";
}
