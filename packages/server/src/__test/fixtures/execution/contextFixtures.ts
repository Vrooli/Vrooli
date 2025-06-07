import { generatePK, generatePublicId } from "@vrooli/shared";
import {
    type BaseContext,
    type CoordinationContext,
    type ProcessContext,
    type ExecutionContext,
    type CrossTierContext,
    type SharedMemory,
    type DecisionRecord,
    type ConsensusRecord,
    type ConflictRecord,
    type NavigationState,
    type ProcessMemory,
    type ExecutionMemory,
    type ToolCallRecord,
    type Pattern,
    type FeedbackRecord,
    type Adaptation,
    type ContextSnapshot,
    type ContextConstraints,
} from "@vrooli/shared";

/**
 * Database fixtures for Context management - used for seeding test data
 * These support cross-tier context sharing and memory management
 */

// Consistent IDs for testing
export const contextDbIds = {
    coord1: generatePK(),
    coord2: generatePK(),
    process1: generatePK(),
    process2: generatePK(),
    exec1: generatePK(),
    exec2: generatePK(),
    cross1: generatePK(),
    decision1: generatePK(),
    consensus1: generatePK(),
    conflict1: generatePK(),
    tool1: generatePK(),
    pattern1: generatePK(),
    feedback1: generatePK(),
    adaptation1: generatePK(),
};

/**
 * Default context metadata
 */
export const defaultMetadata = {
    userId: generatePublicId(),
    sessionId: generatePublicId(),
    requestId: generatePublicId(),
    tags: ["test"],
};

/**
 * Default context constraints
 */
export const defaultConstraints: ContextConstraints = {
    maxMemorySize: 1048576, // 1MB
    maxExecutionTime: 300000, // 5 minutes
    maxCost: 10,
    securityLevel: "private",
    allowedOperations: ["read", "write", "execute"],
};

/**
 * Minimal coordination context (Tier 1)
 */
export const minimalCoordinationContext: CoordinationContext = {
    id: contextDbIds.coord1,
    tier: 1,
    timestamp: new Date(),
    metadata: defaultMetadata,
    swarmId: generatePK(),
    conversationId: generatePublicId(),
    teams: [],
    sharedMemory: {
        blackboard: {},
        decisions: [],
        consensus: [],
        conflicts: [],
    },
    coordinationState: {
        phase: "planning",
        activeGoals: [],
        completedGoals: [],
        blockedGoals: [],
    },
};

/**
 * Complete coordination context with decisions
 */
export const completeCoordinationContext: CoordinationContext = {
    id: contextDbIds.coord2,
    tier: 1,
    timestamp: new Date(),
    metadata: {
        ...defaultMetadata,
        tags: ["production", "high-priority"],
    },
    swarmId: generatePK(),
    conversationId: generatePublicId(),
    teams: [generatePK(), generatePK()],
    sharedMemory: {
        blackboard: {
            currentPhase: "execution",
            progress: 0.45,
            activeAgents: 3,
        },
        decisions: [
            {
                id: contextDbIds.decision1,
                timestamp: new Date(Date.now() - 60000),
                agentId: generatePK(),
                decision: "Allocate resources to high-priority task",
                rationale: "Task has critical deadline",
                confidence: 0.85,
            },
        ],
        consensus: [
            {
                id: contextDbIds.consensus1,
                timestamp: new Date(Date.now() - 30000),
                topic: "resource_allocation",
                participants: [generatePK(), generatePK()],
                result: "approved",
                agreement: 0.9,
            },
        ],
        conflicts: [
            {
                id: contextDbIds.conflict1,
                timestamp: new Date(Date.now() - 120000),
                type: "resource",
                parties: [generatePK(), generatePK()],
                description: "Both agents need same GPU resource",
                resolution: "Time-sharing schedule implemented",
                resolved: true,
            },
        ],
    },
    coordinationState: {
        phase: "executing",
        activeGoals: ["goal1", "goal2"],
        completedGoals: ["goal0"],
        blockedGoals: [],
    },
};

/**
 * Minimal process context (Tier 2)
 */
export const minimalProcessContext: ProcessContext = {
    id: contextDbIds.process1,
    tier: 2,
    timestamp: new Date(),
    metadata: defaultMetadata,
    runId: generatePK(),
    routineId: generatePK(),
    navigationState: {
        currentLocation: "step1",
        locationStack: [],
        visitedLocations: new Set(),
        branchStates: {},
    },
    processMemory: {
        variables: {},
        checkpoints: [],
        optimizations: [],
        performanceData: {
            stepDurations: {},
            resourceUsage: {},
            bottlenecks: [],
            efficiencyScore: 1,
        },
    },
    orchestrationState: {
        phase: "initializing",
        activeSteps: [],
        pendingSteps: [],
        completedSteps: [],
        failedSteps: [],
    },
};

/**
 * Process context with navigation history
 */
export const processContextWithHistory: ProcessContext = {
    id: contextDbIds.process2,
    tier: 2,
    timestamp: new Date(),
    metadata: defaultMetadata,
    runId: generatePK(),
    routineId: generatePK(),
    navigationState: {
        currentLocation: "step5",
        locationStack: ["step1", "step2", "step3", "step4"],
        visitedLocations: new Set(["step1", "step2", "step3", "step4", "step5"]),
        branchStates: {
            branch1: {
                id: "branch1",
                status: "completed",
                parallel: false,
                startedAt: new Date(Date.now() - 120000),
                completedAt: new Date(Date.now() - 60000),
            },
            branch2: {
                id: "branch2",
                status: "active",
                parallel: true,
                startedAt: new Date(Date.now() - 30000),
            },
        },
    },
    processMemory: {
        variables: {
            counter: 5,
            results: ["result1", "result2"],
        },
        checkpoints: ["checkpoint1", "checkpoint2"],
        optimizations: ["parallel_branch2"],
        performanceData: {
            stepDurations: {
                step1: 5000,
                step2: 8000,
                step3: 3000,
                step4: 12000,
                step5: 2000,
            },
            resourceUsage: {
                cpu: 0.65,
                memory: 0.45,
                tokens: 1500,
            },
            bottlenecks: ["step4"],
            efficiencyScore: 0.78,
        },
    },
    orchestrationState: {
        phase: "running",
        activeSteps: ["step5"],
        pendingSteps: ["step6", "step7"],
        completedSteps: ["step1", "step2", "step3", "step4"],
        failedSteps: [],
    },
};

/**
 * Minimal execution context (Tier 3)
 */
export const minimalExecutionContext: ExecutionContext = {
    id: contextDbIds.exec1,
    tier: 3,
    timestamp: new Date(),
    metadata: defaultMetadata,
    stepId: generatePK(),
    strategyType: "deterministic",
    executionMemory: {
        inputs: {},
        outputs: {},
        toolCalls: [],
        strategyData: {},
        learningData: {
            patterns: [],
            feedback: [],
            adaptations: [],
        },
    },
    adaptationState: {
        mode: "stable",
        currentStrategy: "deterministic",
        alternativeStrategies: [],
        confidenceThreshold: 0.7,
        adaptationRate: 0.1,
    },
};

/**
 * Execution context with tool calls and learning
 */
export const executionContextWithHistory: ExecutionContext = {
    id: contextDbIds.exec2,
    tier: 3,
    timestamp: new Date(),
    metadata: defaultMetadata,
    stepId: generatePK(),
    strategyType: "reasoning",
    executionMemory: {
        inputs: {
            query: "Analyze customer data",
            data: { customers: 100, revenue: 50000 },
        },
        outputs: {
            analysis: "Positive growth trend",
            recommendations: ["Increase marketing", "Expand product line"],
        },
        toolCalls: [
            {
                id: contextDbIds.tool1,
                timestamp: new Date(Date.now() - 5000),
                toolName: "dataAnalyzer",
                parameters: { dataset: "customers", metric: "growth" },
                result: { growth: 0.15 },
                duration: 2500,
            },
        ],
        strategyData: {
            llmModel: "gpt-4",
            temperature: 0.7,
            reasoningSteps: 3,
        },
        learningData: {
            patterns: [
                {
                    id: contextDbIds.pattern1,
                    type: "success",
                    description: "Data analysis with growth metric yields actionable insights",
                    frequency: 5,
                    impact: 0.8,
                },
            ],
            feedback: [
                {
                    id: contextDbIds.feedback1,
                    timestamp: new Date(Date.now() - 300000),
                    type: "user",
                    rating: 0.9,
                    comment: "Excellent analysis",
                },
            ],
            adaptations: [
                {
                    id: contextDbIds.adaptation1,
                    timestamp: new Date(Date.now() - 600000),
                    type: "parameter",
                    before: { temperature: 0.9 },
                    after: { temperature: 0.7 },
                    reason: "Reduce randomness for analytical tasks",
                    impact: 0.2,
                },
            ],
        },
    },
    adaptationState: {
        mode: "optimizing",
        currentStrategy: "reasoning",
        alternativeStrategies: ["conversational", "deterministic"],
        confidenceThreshold: 0.8,
        adaptationRate: 0.2,
    },
};

/**
 * Cross-tier context
 */
export const crossTierContext: CrossTierContext = {
    requestId: generatePublicId(),
    userId: generatePublicId(),
    sessionId: generatePublicId(),
    tier1Context: minimalCoordinationContext,
    tier2Context: minimalProcessContext,
    tier3Context: minimalExecutionContext,
    sharedData: {
        goal: "Complete user request",
        priority: "high",
        deadline: new Date(Date.now() + 3600000),
    },
    constraints: defaultConstraints,
};

/**
 * Factory for creating context fixtures with overrides
 */
export class ContextFactory {
    /**
     * Create minimal coordination context
     */
    static createCoordinationContext(
        overrides?: Partial<CoordinationContext>
    ): CoordinationContext {
        return {
            ...minimalCoordinationContext,
            id: generatePK(),
            timestamp: new Date(),
            metadata: {
                ...defaultMetadata,
                requestId: generatePublicId(),
            },
            swarmId: generatePK(),
            conversationId: generatePublicId(),
            ...overrides,
        };
    }

    /**
     * Create coordination context with decisions
     */
    static createCoordinationWithDecisions(
        decisionCount: number = 3,
        overrides?: Partial<CoordinationContext>
    ): CoordinationContext {
        const decisions: DecisionRecord[] = [];
        
        for (let i = 0; i < decisionCount; i++) {
            decisions.push({
                id: generatePK(),
                timestamp: new Date(Date.now() - i * 60000),
                agentId: generatePK(),
                decision: `Decision ${i + 1}`,
                rationale: `Rationale for decision ${i + 1}`,
                confidence: 0.7 + Math.random() * 0.3,
            });
        }

        return this.createCoordinationContext({
            sharedMemory: {
                ...minimalCoordinationContext.sharedMemory,
                decisions,
            },
            ...overrides,
        });
    }

    /**
     * Create minimal process context
     */
    static createProcessContext(
        overrides?: Partial<ProcessContext>
    ): ProcessContext {
        return {
            ...minimalProcessContext,
            id: generatePK(),
            timestamp: new Date(),
            metadata: {
                ...defaultMetadata,
                requestId: generatePublicId(),
            },
            runId: generatePK(),
            routineId: generatePK(),
            ...overrides,
        };
    }

    /**
     * Create process context with performance data
     */
    static createProcessWithPerformance(
        stepCount: number = 5,
        overrides?: Partial<ProcessContext>
    ): ProcessContext {
        const stepDurations: Record<string, number> = {};
        const completedSteps: string[] = [];
        
        for (let i = 0; i < stepCount; i++) {
            const stepId = `step${i + 1}`;
            stepDurations[stepId] = Math.floor(Math.random() * 10000) + 1000;
            completedSteps.push(stepId);
        }

        return this.createProcessContext({
            processMemory: {
                ...minimalProcessContext.processMemory,
                performanceData: {
                    stepDurations,
                    resourceUsage: {
                        cpu: Math.random(),
                        memory: Math.random(),
                        tokens: Math.floor(Math.random() * 5000),
                    },
                    bottlenecks: Object.entries(stepDurations)
                        .filter(([_, duration]) => duration > 5000)
                        .map(([stepId]) => stepId),
                    efficiencyScore: 0.5 + Math.random() * 0.5,
                },
            },
            orchestrationState: {
                ...minimalProcessContext.orchestrationState,
                phase: "running",
                completedSteps,
            },
            ...overrides,
        });
    }

    /**
     * Create minimal execution context
     */
    static createExecutionContext(
        overrides?: Partial<ExecutionContext>
    ): ExecutionContext {
        return {
            ...minimalExecutionContext,
            id: generatePK(),
            timestamp: new Date(),
            metadata: {
                ...defaultMetadata,
                requestId: generatePublicId(),
            },
            stepId: generatePK(),
            ...overrides,
        };
    }

    /**
     * Create execution context with tool calls
     */
    static createExecutionWithTools(
        toolCalls: Array<{ name: string; duration?: number }>,
        overrides?: Partial<ExecutionContext>
    ): ExecutionContext {
        const toolCallRecords: ToolCallRecord[] = toolCalls.map((tool, index) => ({
            id: generatePK(),
            timestamp: new Date(Date.now() - (index + 1) * 5000),
            toolName: tool.name,
            parameters: { index },
            result: { success: true },
            duration: tool.duration || Math.floor(Math.random() * 5000) + 500,
        }));

        return this.createExecutionContext({
            executionMemory: {
                ...minimalExecutionContext.executionMemory,
                toolCalls: toolCallRecords,
            },
            ...overrides,
        });
    }

    /**
     * Create cross-tier context
     */
    static createCrossTierContext(
        tiers: Array<1 | 2 | 3> = [1, 2, 3],
        overrides?: Partial<CrossTierContext>
    ): CrossTierContext {
        const context: CrossTierContext = {
            requestId: generatePublicId(),
            userId: generatePublicId(),
            sessionId: generatePublicId(),
            sharedData: {},
            constraints: defaultConstraints,
            ...overrides,
        };

        if (tiers.includes(1)) {
            context.tier1Context = this.createCoordinationContext();
        }
        if (tiers.includes(2)) {
            context.tier2Context = this.createProcessContext();
        }
        if (tiers.includes(3)) {
            context.tier3Context = this.createExecutionContext();
        }

        return context;
    }

    /**
     * Create context snapshot
     */
    static createSnapshot(
        context: BaseContext | CrossTierContext,
        contextType: "coordination" | "process" | "execution" | "crossTier"
    ): ContextSnapshot {
        return {
            id: generatePK(),
            contextType,
            context,
            timestamp: new Date(),
            size: JSON.stringify(context).length,
            compressed: false,
        };
    }
}

/**
 * Helper to create shared memory
 */
export function createSharedMemory(
    options?: {
        decisionCount?: number;
        consensusCount?: number;
        conflictCount?: number;
        blackboardData?: Record<string, unknown>;
    }
): SharedMemory {
    const memory: SharedMemory = {
        blackboard: options?.blackboardData || {},
        decisions: [],
        consensus: [],
        conflicts: [],
    };

    // Add decisions
    const decisionCount = options?.decisionCount || 0;
    for (let i = 0; i < decisionCount; i++) {
        memory.decisions.push({
            id: generatePK(),
            timestamp: new Date(Date.now() - i * 60000),
            agentId: generatePK(),
            decision: `Decision ${i + 1}`,
            rationale: `Rationale ${i + 1}`,
            confidence: 0.7 + Math.random() * 0.3,
        });
    }

    // Add consensus records
    const consensusCount = options?.consensusCount || 0;
    for (let i = 0; i < consensusCount; i++) {
        memory.consensus.push({
            id: generatePK(),
            timestamp: new Date(Date.now() - i * 30000),
            topic: `topic_${i + 1}`,
            participants: [generatePK(), generatePK()],
            result: i % 2 === 0 ? "approved" : "rejected",
            agreement: 0.6 + Math.random() * 0.4,
        });
    }

    // Add conflicts
    const conflictCount = options?.conflictCount || 0;
    for (let i = 0; i < conflictCount; i++) {
        memory.conflicts.push({
            id: generatePK(),
            timestamp: new Date(Date.now() - i * 120000),
            type: ["goal", "resource", "strategy", "priority"][i % 4] as any,
            parties: [generatePK(), generatePK()],
            description: `Conflict ${i + 1}`,
            resolved: i < conflictCount / 2,
            resolution: i < conflictCount / 2 ? `Resolution ${i + 1}` : undefined,
        });
    }

    return memory;
}

/**
 * Helper to seed test contexts
 */
export async function seedTestContexts(
    prisma: any,
    count: number = 3,
    options?: {
        tier?: 1 | 2 | 3;
        userId?: string;
    }
) {
    const contexts = [];

    for (let i = 0; i < count; i++) {
        let context;
        
        if (!options?.tier || options.tier === 1) {
            context = ContextFactory.createCoordinationContext({
                metadata: {
                    ...defaultMetadata,
                    userId: options?.userId || generatePublicId(),
                },
            });
        } else if (options.tier === 2) {
            context = ContextFactory.createProcessContext({
                metadata: {
                    ...defaultMetadata,
                    userId: options?.userId || generatePublicId(),
                },
            });
        } else {
            context = ContextFactory.createExecutionContext({
                metadata: {
                    ...defaultMetadata,
                    userId: options?.userId || generatePublicId(),
                },
            });
        }

        contexts.push(context);
    }

    return contexts;
}