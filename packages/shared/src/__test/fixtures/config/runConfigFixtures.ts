import { InputGenerationStrategy, PathSelectionStrategy, SubroutineExecutionStrategy } from "../../../run/enums.js";
import { type RunProgress } from "../../../run/types.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Run configuration fixtures for testing run execution settings and progress
 */
type RunProgressConfigObject = {
    __version: RunProgress["__version"];
    branches: RunProgress["branches"];
    config: Omit<RunProgress["config"], "isPrivate">;
    decisions: RunProgress["decisions"];
    metrics: Pick<RunProgress["metrics"], "creditsSpent">;
    subcontexts: RunProgress["subcontexts"];
}

export const runConfigFixtures: ConfigTestFixtures<RunProgressConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
        branches: [],
        config: {
            botConfig: {},
            decisionConfig: {
                inputGeneration: InputGenerationStrategy.Auto,
                pathSelection: PathSelectionStrategy.AutoPickFirst,
                subroutineExecution: SubroutineExecutionStrategy.Auto,
            },
            limits: {},
            loopConfig: {},
            onBranchFailure: "Stop",
            onGatewayForkFailure: "Fail",
            onNormalNodeFailure: "Fail",
            onOnlyWaitingBranches: "Continue",
            testMode: false,
        },
        decisions: [],
        metrics: {
            creditsSpent: "0",
        },
        subcontexts: {},
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        branches: [
            {
                id: "branch_1",
                status: "Active",
                nodeId: "node_1",
                depth: 0,
                parentId: null,
                data: {},
                outputs: {},
                errors: [],
            },
            {
                id: "branch_2",
                status: "Waiting",
                nodeId: "node_2",
                depth: 1,
                parentId: "branch_1",
                data: { input: "test" },
                outputs: { result: "pending" },
                errors: [],
            }
        ],
        config: {
            botConfig: {
                model: "gpt-4",
                maxTokens: 2048,
            },
            decisionConfig: {
                inputGeneration: InputGenerationStrategy.ManualPrompt,
                pathSelection: PathSelectionStrategy.ManualPrompt,
                subroutineExecution: SubroutineExecutionStrategy.ManualPrompt,
            },
            limits: {
                maxRunTime: 3600000, // 1 hour
                maxDecisions: 100,
                maxComplexity: 1000,
                maxApiCalls: 50,
            },
            loopConfig: {
                maxIterations: 10,
                detectInfiniteLoops: true,
            },
            onBranchFailure: "Continue",
            onGatewayForkFailure: "Prompt",
            onNormalNodeFailure: "Retry",
            onOnlyWaitingBranches: "Wait",
            testMode: true,
        },
        decisions: [
            {
                id: "decision_1",
                branchId: "branch_1",
                nodeId: "node_1",
                type: "PathSelection",
                status: "Resolved",
                question: "Which path to take?",
                options: ["Option A", "Option B"],
                selectedOption: "Option A",
                timestamp: "2024-01-01T00:00:00Z",
            }
        ],
        metrics: {
            creditsSpent: "1000",
        },
        subcontexts: {
            "sub_1": {
                data: { key: "value" },
                outputs: { result: "success" },
            }
        },
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        branches: [],
        config: {
            botConfig: {},
            decisionConfig: {
                inputGeneration: InputGenerationStrategy.Auto,
                pathSelection: PathSelectionStrategy.AutoPickFirst,
                subroutineExecution: SubroutineExecutionStrategy.Auto,
            },
            limits: {},
            loopConfig: {},
            onBranchFailure: "Stop",
            onGatewayForkFailure: "Fail",
            onNormalNodeFailure: "Fail",
            onOnlyWaitingBranches: "Continue",
            testMode: false,
        },
        decisions: [],
        metrics: {
            creditsSpent: "0",
        },
        subcontexts: {},
    },

    invalid: {
        missingVersion: {
            branches: [],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {},
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [],
            metrics: {
                creditsSpent: "0",
            },
            subcontexts: {},
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            branches: [],
            config: {},
            decisions: [],
            metrics: {},
            subcontexts: {},
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            branches: "not an array", // Wrong type
            config: null, // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            branches: [{
                id: 123, // Should be string
                status: "Invalid", // Invalid enum value
                nodeId: true, // Should be string
            }],
            config: {
                testMode: "yes", // Should be boolean
                limits: {
                    maxRunTime: "1 hour", // Should be number
                },
            },
            metrics: {
                creditsSpent: 1000, // Should be string
            },
        }
    },

    variants: {
        autoExecutionConfig: {
            __version: LATEST_CONFIG_VERSION,
            branches: [],
            config: {
                botConfig: {
                    model: "gpt-4-turbo",
                    maxTokens: 4096,
                },
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickBest,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {
                    maxRunTime: 1800000, // 30 minutes
                    maxDecisions: 50,
                    maxComplexity: 500,
                },
                loopConfig: {
                    maxIterations: 5,
                    detectInfiniteLoops: true,
                },
                onBranchFailure: "Continue",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Retry",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [],
            metrics: {
                creditsSpent: "0",
            },
            subcontexts: {},
        },

        manualExecutionConfig: {
            __version: LATEST_CONFIG_VERSION,
            branches: [],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.ManualPrompt,
                    pathSelection: PathSelectionStrategy.ManualPrompt,
                    subroutineExecution: SubroutineExecutionStrategy.ManualPrompt,
                },
                limits: {},
                loopConfig: {},
                onBranchFailure: "Prompt",
                onGatewayForkFailure: "Prompt",
                onNormalNodeFailure: "Prompt",
                onOnlyWaitingBranches: "Prompt",
                testMode: true,
            },
            decisions: [],
            metrics: {
                creditsSpent: "0",
            },
            subcontexts: {},
        },

        withActiveBranches: {
            __version: LATEST_CONFIG_VERSION,
            branches: [
                {
                    id: "branch_main",
                    status: "Active",
                    nodeId: "node_start",
                    depth: 0,
                    parentId: null,
                    data: { initialized: true },
                    outputs: {},
                    errors: [],
                },
                {
                    id: "branch_parallel_1",
                    status: "Active",
                    nodeId: "node_task_1",
                    depth: 1,
                    parentId: "branch_main",
                    data: { taskType: "process" },
                    outputs: {},
                    errors: [],
                },
                {
                    id: "branch_parallel_2",
                    status: "Active",
                    nodeId: "node_task_2",
                    depth: 1,
                    parentId: "branch_main",
                    data: { taskType: "validate" },
                    outputs: {},
                    errors: [],
                }
            ],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {},
                loopConfig: {},
                onBranchFailure: "Continue",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Retry",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [],
            metrics: {
                creditsSpent: "500",
            },
            subcontexts: {},
        },

        withCompletedDecisions: {
            __version: LATEST_CONFIG_VERSION,
            branches: [{
                id: "branch_1",
                status: "Completed",
                nodeId: "node_1",
                depth: 0,
                parentId: null,
                data: {},
                outputs: { decision: "A" },
                errors: [],
            }],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.ManualPrompt,
                    pathSelection: PathSelectionStrategy.ManualPrompt,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {},
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [
                {
                    id: "decision_1",
                    branchId: "branch_1",
                    nodeId: "node_1",
                    type: "PathSelection",
                    status: "Resolved",
                    question: "Select path A or B?",
                    options: ["Path A", "Path B"],
                    selectedOption: "Path A",
                    timestamp: "2024-01-01T10:00:00Z",
                },
                {
                    id: "decision_2",
                    branchId: "branch_1",
                    nodeId: "node_2",
                    type: "InputGeneration",
                    status: "Resolved",
                    question: "Enter required input",
                    options: [],
                    selectedOption: '{"value": "test input"}',
                    timestamp: "2024-01-01T10:05:00Z",
                }
            ],
            metrics: {
                creditsSpent: "250",
            },
            subcontexts: {},
        },

        withErrors: {
            __version: LATEST_CONFIG_VERSION,
            branches: [
                {
                    id: "branch_failed",
                    status: "Failed",
                    nodeId: "node_error",
                    depth: 0,
                    parentId: null,
                    data: {},
                    outputs: {},
                    errors: [
                        {
                            message: "Network timeout",
                            code: "NETWORK_ERROR",
                            timestamp: "2024-01-01T12:00:00Z",
                        },
                        {
                            message: "Retry failed",
                            code: "RETRY_EXHAUSTED",
                            timestamp: "2024-01-01T12:01:00Z",
                        }
                    ],
                }
            ],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {
                    maxRetries: 3,
                },
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Retry",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [],
            metrics: {
                creditsSpent: "100",
            },
            subcontexts: {},
        },

        withSubcontexts: {
            __version: LATEST_CONFIG_VERSION,
            branches: [],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {},
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [],
            metrics: {
                creditsSpent: "2000",
            },
            subcontexts: {
                "sub_process_1": {
                    data: {
                        input: "raw data",
                        processedAt: "2024-01-01T15:00:00Z",
                    },
                    outputs: {
                        result: "processed",
                        count: 42,
                    },
                },
                "sub_validate_1": {
                    data: {
                        target: "processed data",
                        rules: ["rule1", "rule2"],
                    },
                    outputs: {
                        valid: true,
                        errors: [],
                    },
                },
                "sub_transform_1": {
                    data: {
                        source: "validated data",
                        format: "json",
                    },
                    outputs: {
                        transformed: { key: "value" },
                        metadata: { size: 1024 },
                    },
                }
            },
        }
    }
};

/**
 * Create a run config with specific decision strategies
 */
export function createRunConfigWithStrategies(
    inputGeneration: InputGenerationStrategy,
    pathSelection: PathSelectionStrategy,
    subroutineExecution: SubroutineExecutionStrategy
): RunProgressConfigObject {
    return mergeWithBaseDefaults<RunProgressConfigObject>({
        config: {
            botConfig: {},
            decisionConfig: {
                inputGeneration,
                pathSelection,
                subroutineExecution,
            },
            limits: {},
            loopConfig: {},
            onBranchFailure: "Stop",
            onGatewayForkFailure: "Fail",
            onNormalNodeFailure: "Fail",
            onOnlyWaitingBranches: "Continue",
            testMode: false,
        }
    });
}

/**
 * Create a run config with active branches
 */
export function createRunConfigWithBranches(
    branchCount: number,
    status: "Active" | "Waiting" | "Completed" | "Failed" = "Active"
): RunProgressConfigObject {
    const branches = Array.from({ length: branchCount }, (_, i) => ({
        id: `branch_${i + 1}`,
        status,
        nodeId: `node_${i + 1}`,
        depth: Math.floor(i / 2),
        parentId: i > 0 ? `branch_${Math.floor((i - 1) / 2) + 1}` : null,
        data: {},
        outputs: {},
        errors: [],
    }));

    return mergeWithBaseDefaults<RunProgressConfigObject>({
        branches,
        metrics: {
            creditsSpent: (branchCount * 100).toString(),
        }
    });
}

/**
 * Create a run config with specific limits
 */
export function createRunConfigWithLimits(
    limits: Partial<RunProgress["config"]["limits"]>
): RunProgressConfigObject {
    return mergeWithBaseDefaults<RunProgressConfigObject>({
        config: {
            botConfig: {},
            decisionConfig: {
                inputGeneration: InputGenerationStrategy.Auto,
                pathSelection: PathSelectionStrategy.AutoPickFirst,
                subroutineExecution: SubroutineExecutionStrategy.Auto,
            },
            limits,
            loopConfig: {},
            onBranchFailure: "Stop",
            onGatewayForkFailure: "Fail",
            onNormalNodeFailure: "Fail",
            onOnlyWaitingBranches: "Continue",
            testMode: false,
        }
    });
}