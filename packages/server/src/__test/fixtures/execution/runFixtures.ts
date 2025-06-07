import { generatePK, generatePublicId } from "@vrooli/shared";
import {
    type Run,
    type RunState,
    type RunConfig,
    type RunProgress,
    type RunContext,
    type Routine,
    type Location,
    type LocationStack,
    type StepStatus,
    type BranchExecution,
    type ContextScope,
    type StepInfo,
    RunState as RunStateEnum,
} from "@vrooli/shared";

/**
 * Database fixtures for Run/Routine execution - used for seeding test data
 * These support the Tier 2 Process Intelligence orchestration
 */

// Consistent IDs for testing
export const runDbIds = {
    run1: generatePK(),
    run2: generatePK(),
    run3: generatePK(),
    routine1: generatePK(),
    routine2: generatePK(),
    routine3: generatePK(),
    step1: generatePK(),
    step2: generatePK(),
    step3: generatePK(),
    branch1: generatePK(),
    branch2: generatePK(),
    scope1: generatePK(),
    scope2: generatePK(),
};

/**
 * Default run configuration
 */
export const defaultRunConfig: RunConfig = {
    maxSteps: 100,
    maxDepth: 10,
    maxTime: 300000, // 5 minutes
    maxCost: 10,
    parallelization: false,
    checkpointInterval: 60000, // 1 minute
    recoveryStrategy: "retry",
};

/**
 * Minimal routine definition
 */
export const minimalRoutine: Routine = {
    id: runDbIds.routine1,
    type: "native",
    version: "1.0.0",
    name: "Test Routine",
    description: "A minimal test routine",
    definition: {
        steps: [
            {
                id: runDbIds.step1,
                type: "action",
                name: "Step 1",
                action: "log",
                parameters: { message: "Hello World" },
            },
        ],
    },
    metadata: {
        author: "test-user",
        createdAt: new Date(),
        updatedAt: new Date(),
        tags: ["test"],
        complexity: "simple",
    },
};

/**
 * Complex routine with branches
 */
export const complexRoutine: Routine = {
    id: runDbIds.routine2,
    type: "native",
    version: "1.0.0",
    name: "Complex Workflow",
    description: "A complex workflow with conditional branches and parallel execution",
    definition: {
        steps: [
            {
                id: runDbIds.step1,
                type: "action",
                name: "Initialize",
                action: "initialize",
                parameters: {},
            },
            {
                id: runDbIds.step2,
                type: "condition",
                name: "Check condition",
                condition: "value > 10",
                trueBranch: [
                    {
                        id: generatePK(),
                        type: "action",
                        name: "Process high value",
                        action: "processHigh",
                    },
                ],
                falseBranch: [
                    {
                        id: generatePK(),
                        type: "action",
                        name: "Process low value",
                        action: "processLow",
                    },
                ],
            },
            {
                id: runDbIds.step3,
                type: "parallel",
                name: "Parallel processing",
                branches: [
                    [
                        {
                            id: generatePK(),
                            type: "action",
                            name: "Task A",
                            action: "taskA",
                        },
                    ],
                    [
                        {
                            id: generatePK(),
                            type: "action",
                            name: "Task B",
                            action: "taskB",
                        },
                    ],
                ],
            },
        ],
    },
    metadata: {
        author: "test-user",
        createdAt: new Date(),
        updatedAt: new Date(),
        tags: ["complex", "conditional", "parallel"],
        complexity: "complex",
        estimatedDuration: 180000, // 3 minutes
    },
};

/**
 * Minimal run data
 */
export const minimalRun: Run = {
    id: runDbIds.run1,
    routineId: runDbIds.routine1,
    state: RunStateEnum.UNINITIALIZED,
    config: defaultRunConfig,
    progress: {
        totalSteps: 1,
        completedSteps: 0,
        failedSteps: 0,
        skippedSteps: 0,
        currentLocation: {
            id: generatePK(),
            routineId: runDbIds.routine1,
            nodeId: runDbIds.step1,
        },
        locationStack: {
            locations: [],
            depth: 0,
        },
        branches: [],
    },
    context: {
        variables: {},
        blackboard: {},
        scopes: [],
    },
};

/**
 * Run in progress
 */
export const runInProgress: Run = {
    id: runDbIds.run2,
    routineId: runDbIds.routine2,
    state: RunStateEnum.RUNNING,
    config: {
        ...defaultRunConfig,
        parallelization: true,
    },
    progress: {
        totalSteps: 6,
        completedSteps: 2,
        failedSteps: 0,
        skippedSteps: 0,
        currentLocation: {
            id: generatePK(),
            routineId: runDbIds.routine2,
            nodeId: runDbIds.step2,
            index: 1,
        },
        locationStack: {
            locations: [
                {
                    id: generatePK(),
                    routineId: runDbIds.routine2,
                    nodeId: runDbIds.step1,
                    index: 0,
                },
            ],
            depth: 1,
        },
        branches: [
            {
                id: runDbIds.branch1,
                parentStepId: runDbIds.step2,
                steps: [
                    {
                        id: generatePK(),
                        state: "completed",
                        startedAt: new Date(Date.now() - 60000),
                        completedAt: new Date(Date.now() - 30000),
                        result: { processed: true },
                    },
                ],
                state: "completed",
                parallel: false,
            },
        ],
    },
    context: {
        variables: {
            value: 15,
            initialized: true,
        },
        blackboard: {
            sharedData: "test-data",
        },
        scopes: [
            {
                id: runDbIds.scope1,
                name: "global",
                variables: {
                    value: 15,
                    initialized: true,
                },
            },
        ],
    },
    startedAt: new Date(Date.now() - 120000),
};

/**
 * Completed run with results
 */
export const completedRun: Run = {
    id: runDbIds.run3,
    routineId: runDbIds.routine1,
    state: RunStateEnum.COMPLETED,
    config: defaultRunConfig,
    progress: {
        totalSteps: 1,
        completedSteps: 1,
        failedSteps: 0,
        skippedSteps: 0,
        currentLocation: {
            id: generatePK(),
            routineId: runDbIds.routine1,
            nodeId: runDbIds.step1,
        },
        locationStack: {
            locations: [],
            depth: 0,
        },
        branches: [],
    },
    context: {
        variables: {
            result: "Success",
            message: "Hello World",
        },
        blackboard: {},
        scopes: [],
    },
    startedAt: new Date(Date.now() - 300000),
    completedAt: new Date(Date.now() - 240000),
};

/**
 * Factory for creating run fixtures with overrides
 */
export class RunFactory {
    static createMinimal(overrides?: Partial<Run>): Run {
        return {
            ...minimalRun,
            id: generatePK(),
            ...overrides,
        };
    }

    static createInProgress(overrides?: Partial<Run>): Run {
        const runId = generatePK();
        const now = Date.now();
        
        return {
            ...runInProgress,
            id: runId,
            startedAt: new Date(now - 60000),
            ...overrides,
        };
    }

    static createCompleted(overrides?: Partial<Run>): Run {
        const runId = generatePK();
        const now = Date.now();
        
        return {
            ...completedRun,
            id: runId,
            startedAt: new Date(now - 300000),
            completedAt: new Date(now - 240000),
            ...overrides,
        };
    }

    /**
     * Create run in specific state
     */
    static createInState(state: RunState, overrides?: Partial<Run>): Run {
        const baseRun = state === RunStateEnum.COMPLETED 
            ? this.createCompleted()
            : state === RunStateEnum.RUNNING 
            ? this.createInProgress()
            : this.createMinimal();

        return {
            ...baseRun,
            state,
            ...overrides,
        };
    }

    /**
     * Create run with specific progress
     */
    static createWithProgress(
        totalSteps: number,
        completedSteps: number,
        overrides?: Partial<Run>
    ): Run {
        const progress: RunProgress = {
            totalSteps,
            completedSteps,
            failedSteps: 0,
            skippedSteps: 0,
            currentLocation: {
                id: generatePK(),
                routineId: runDbIds.routine1,
                nodeId: generatePK(),
                index: completedSteps,
            },
            locationStack: {
                locations: [],
                depth: 0,
            },
            branches: [],
        };

        return this.createInProgress({
            progress,
            ...overrides,
        });
    }

    /**
     * Create run with parallel branches
     */
    static createWithBranches(
        branchCount: number = 2,
        overrides?: Partial<Run>
    ): Run {
        const branches: BranchExecution[] = [];
        
        for (let i = 0; i < branchCount; i++) {
            const steps: StepStatus[] = [
                {
                    id: generatePK(),
                    state: i === 0 ? "completed" : "running",
                    startedAt: new Date(Date.now() - 30000),
                    completedAt: i === 0 ? new Date() : undefined,
                },
                {
                    id: generatePK(),
                    state: "pending",
                },
            ];

            branches.push({
                id: generatePK(),
                parentStepId: generatePK(),
                steps,
                state: i === 0 ? "completed" : "running",
                parallel: true,
            });
        }

        return this.createInProgress({
            progress: {
                ...runInProgress.progress,
                branches,
            },
            ...overrides,
        });
    }

    /**
     * Create run with error
     */
    static createFailed(
        error: string = "Test error",
        overrides?: Partial<Run>
    ): Run {
        return this.createInState(RunStateEnum.FAILED, {
            error,
            progress: {
                ...minimalRun.progress,
                failedSteps: 1,
            },
            ...overrides,
        });
    }

    /**
     * Create run with complex context
     */
    static createWithContext(
        variables: Record<string, unknown>,
        overrides?: Partial<Run>
    ): Run {
        const scopes: ContextScope[] = [
            {
                id: generatePK(),
                name: "global",
                variables,
            },
            {
                id: generatePK(),
                name: "local",
                parentId: runDbIds.scope1,
                variables: {
                    localVar: "local value",
                },
            },
        ];

        return this.createInProgress({
            context: {
                variables,
                blackboard: {
                    sharedState: "shared",
                    timestamp: new Date().toISOString(),
                },
                scopes,
            },
            ...overrides,
        });
    }
}

/**
 * Helper to create step status
 */
export function createStepStatus(
    state: StepStatus["state"] = "pending",
    overrides?: Partial<StepStatus>
): StepStatus {
    const base: StepStatus = {
        id: generatePK(),
        state,
    };

    if (state === "running") {
        base.startedAt = new Date();
    } else if (state === "completed") {
        base.startedAt = new Date(Date.now() - 30000);
        base.completedAt = new Date();
        base.result = { success: true };
    } else if (state === "failed") {
        base.startedAt = new Date(Date.now() - 30000);
        base.completedAt = new Date();
        base.error = "Step failed";
    }

    return {
        ...base,
        ...overrides,
    };
}

/**
 * Helper to create location
 */
export function createLocation(
    routineId: string,
    nodeId: string,
    overrides?: Partial<Location>
): Location {
    return {
        id: generatePK(),
        routineId,
        nodeId,
        ...overrides,
    };
}

/**
 * Helper to create routine with specific navigator type
 */
export function createRoutineForNavigator(
    navigatorType: string,
    overrides?: Partial<Routine>
): Routine {
    const definitions: Record<string, unknown> = {
        native: {
            steps: [
                { id: generatePK(), type: "action", name: "Step 1" },
                { id: generatePK(), type: "action", name: "Step 2" },
            ],
        },
        bpmn: {
            process: {
                id: generatePK(),
                name: "BPMN Process",
                tasks: [
                    { id: generatePK(), name: "Task 1" },
                    { id: generatePK(), name: "Task 2" },
                ],
            },
        },
        langchain: {
            chain: {
                type: "sequential",
                steps: [
                    { type: "llm", model: "gpt-4" },
                    { type: "tool", name: "search" },
                ],
            },
        },
        temporal: {
            workflow: {
                name: "TemporalWorkflow",
                activities: ["Activity1", "Activity2"],
            },
        },
    };

    return {
        id: generatePK(),
        type: navigatorType,
        version: "1.0.0",
        name: `${navigatorType} Routine`,
        description: `Test routine for ${navigatorType} navigator`,
        definition: definitions[navigatorType] || {},
        metadata: {
            author: "test-user",
            createdAt: new Date(),
            updatedAt: new Date(),
            tags: [navigatorType, "test"],
            complexity: "moderate",
        },
        ...overrides,
    };
}

/**
 * Helper to seed test runs
 */
export async function seedTestRuns(
    prisma: any,
    count: number = 3,
    options?: {
        routineId?: string;
        userId?: string;
        states?: RunState[];
    }
) {
    const runs = [];
    const states = options?.states || [
        RunStateEnum.UNINITIALIZED,
        RunStateEnum.RUNNING,
        RunStateEnum.COMPLETED,
    ];

    for (let i = 0; i < count; i++) {
        const state = states[i % states.length];
        const runData = RunFactory.createInState(state, {
            routineId: options?.routineId || runDbIds.routine1,
            context: {
                variables: {},
                blackboard: {},
                scopes: [],
            },
        });

        // Note: You'll need to adapt this to your actual database schema
        runs.push(runData);
    }

    return runs;
}