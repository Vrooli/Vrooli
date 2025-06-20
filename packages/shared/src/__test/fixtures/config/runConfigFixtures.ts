import { ResourceType } from "../../../api/types.js";
import { BranchStatus, InputGenerationStrategy, PathSelectionStrategy, SubroutineExecutionStrategy } from "../../../run/enums.js";
import { type BranchProgress, type RunProgress } from "../../../run/types.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

// Constants to avoid magic numbers
const ONE_SECOND_MS = 1000;
const TWO_SECONDS_MS = 2000;
const FIVE_SECONDS_MS = 5000;
const TEN_SECONDS_MS = 10000;
const THIRTY_MINUTES_MS = 1800000; // 30 minutes in milliseconds

// Credit amounts for testing
const CREDITS_SMALL = "50";
const CREDITS_MEDIUM = "100";
const CREDITS_LARGE = "200";
const CREDITS_XLARGE = "250";
const CREDITS_TEST_AMOUNT = "500";
const CREDITS_HIGH_LIMIT = "1000";
const CREDITS_MAX_LIMIT = "5000";

// Step and processing limits
const STEPS_MEDIUM = 500;
const STEPS_HIGH = 1000;
const LOOP_DELAY_MS = 5000;
const LOOP_DELAY_SHORT_MS = 100;

// Multiplier for credit calculation
const CREDITS_PER_BRANCH = 100;

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
                branchId: "branch_1",
                childSubroutineInstanceId: null,
                closedLocations: [],
                creditsSpent: "0",
                locationStack: [{
                    locationId: "node_1",
                    objectId: "routine_123",
                    objectType: ResourceType.Routine,
                    subroutineId: null,
                }],
                manualExecutionConfirmed: false,
                nodeStartTimeMs: Date.now(),
                processId: "process_1",
                status: BranchStatus.Active,
                stepId: null,
                subroutineInstanceId: "sub_instance_1",
                supportsParallelExecution: false,
            },
            {
                branchId: "branch_2",
                childSubroutineInstanceId: "sub_child_1",
                closedLocations: [{
                    locationId: "node_1",
                    objectId: "routine_123",
                    objectType: ResourceType.Routine,
                    subroutineId: null,
                }],
                creditsSpent: CREDITS_MEDIUM,
                locationStack: [{
                    locationId: "node_1",
                    objectId: "routine_123",
                    objectType: ResourceType.Routine,
                    subroutineId: null,
                }, {
                    locationId: "node_2",
                    objectId: "routine_123",
                    objectType: ResourceType.Routine,
                    subroutineId: "sub_456",
                }],
                manualExecutionConfirmed: true,
                nodeStartTimeMs: Date.now() - FIVE_SECONDS_MS,
                processId: "process_2",
                status: BranchStatus.Waiting,
                stepId: "step_1",
                subroutineInstanceId: "sub_instance_2",
                supportsParallelExecution: false,
            },
        ],
        config: {
            botConfig: {
                // @ts-expect-error - Intentionally invalid type for testing
                model: "gpt-4", // LlmModel type
                modelHandling: "OnlyWhenMissing",
            },
            decisionConfig: {
                inputGeneration: InputGenerationStrategy.Manual,
                pathSelection: PathSelectionStrategy.ManualPick,
                subroutineExecution: SubroutineExecutionStrategy.Manual,
            },
            limits: {
                maxTime: 3600000, // 1 hour
            },
            loopConfig: {
                loopDelayMs: LOOP_DELAY_SHORT_MS,
                loopDelayMultiplier: 1.5,
            },
            onBranchFailure: "Continue",
            onGatewayForkFailure: "Wait", // Valid options: Continue, Wait, Fail
            onNormalNodeFailure: "Continue", // Valid options: Continue, Wait, Fail
            onOnlyWaitingBranches: "Pause", // Valid options: Pause, Stop, Continue
            testMode: true,
        },
        decisions: [
            {
                __type: "Resolved" as const,
                key: "decision_branch_1_node_1",
                decisionType: "chooseOne" as const,
                result: "node_option_a",
            },
        ],
        metrics: {
            creditsSpent: CREDITS_HIGH_LIMIT,
        },
        subcontexts: {
            "sub_1": {
                currentTask: {
                    name: "Process Data",
                    description: "Process the input data",
                },
                allInputsList: [
                    { key: "input1", value: "test" },
                ],
                allInputsMap: {
                    input1: "test",
                },
                allOutputsList: [
                    { key: "output1", value: "success" },
                ],
                allOutputsMap: {
                    output1: "success",
                },
                triggeredBoundaryEventIds: [],
                nodeStartTimes: {
                    "node_1": Date.now() - TEN_SECONDS_MS,
                },
                runtimeEvents: {
                    messages: [],
                    signals: [],
                    errors: [],
                    escalations: [],
                },
                timeZone: "UTC",
            },
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
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {},
                loopConfig: {},
            },
            decisions: [],
            metrics: {
                creditsSpent: "0",
            },
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
                branchId: "branch_invalid",
                childSubroutineInstanceId: null,
                closedLocations: [],
                creditsSpent: "0",
                locationStack: [],
                manualExecutionConfirmed: false,
                nodeStartTimeMs: Date.now(),
                processId: "process_invalid",
                // @ts-expect-error - Intentionally invalid enum value for testing
                status: "Invalid", // Invalid enum value
                stepId: null,
                subroutineInstanceId: "sub_invalid",
                supportsParallelExecution: false,
            }],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {
                    // @ts-expect-error - Intentionally invalid type for testing
                    maxTime: "1 hour", // Should be number
                },
                loopConfig: {},
                // @ts-expect-error - Intentionally invalid type for testing
                testMode: "yes", // Should be boolean
            },
            metrics: {
                // @ts-expect-error - Intentionally invalid type for testing
                creditsSpent: 1000, // Should be string
            },
        },
    },

    variants: {
        autoExecutionConfig: {
            __version: LATEST_CONFIG_VERSION,
            branches: [],
            config: {
                botConfig: {
                    model: {
                        name: "model.gpt4turbo" as any, // Intentionally invalid translation key for testing
                        value: "gpt-4-turbo",
                    },
                    modelHandling: "Override",
                },
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickLLM,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {
                    maxTime: THIRTY_MINUTES_MS,
                    maxCredits: CREDITS_MAX_LIMIT,
                    maxSteps: STEPS_MEDIUM,
                },
                loopConfig: {
                    loopDelayMs: LOOP_DELAY_MS,
                    loopDelayMultiplier: 2,
                },
                onBranchFailure: "Continue",
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

        manualExecutionConfig: {
            __version: LATEST_CONFIG_VERSION,
            branches: [],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Manual,
                    pathSelection: PathSelectionStrategy.ManualPick,
                    subroutineExecution: SubroutineExecutionStrategy.Manual,
                },
                limits: {},
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
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
                    branchId: "branch_main",
                    childSubroutineInstanceId: null,
                    closedLocations: [],
                    creditsSpent: "0",
                    locationStack: [{
                        locationId: "node_start",
                        objectId: "routine_123",
                        objectType: ResourceType.Routine,
                        subroutineId: null,
                    }],
                    manualExecutionConfirmed: false,
                    nodeStartTimeMs: Date.now(),
                    processId: "process_main",
                    status: BranchStatus.Active,
                    stepId: null,
                    subroutineInstanceId: "sub_main",
                    supportsParallelExecution: true,
                },
                {
                    branchId: "branch_parallel_1",
                    childSubroutineInstanceId: null,
                    closedLocations: [],
                    creditsSpent: CREDITS_SMALL,
                    locationStack: [{
                        locationId: "node_task_1",
                        objectId: "routine_123",
                        objectType: ResourceType.Routine,
                        subroutineId: "sub_task_1",
                    }],
                    manualExecutionConfirmed: false,
                    nodeStartTimeMs: Date.now() - ONE_SECOND_MS,
                    processId: "process_main",
                    status: BranchStatus.Active,
                    stepId: "step_1",
                    subroutineInstanceId: "sub_parallel_1",
                    supportsParallelExecution: true,
                },
                {
                    branchId: "branch_parallel_2",
                    childSubroutineInstanceId: null,
                    closedLocations: [],
                    creditsSpent: "75",
                    locationStack: [{
                        locationId: "node_task_2",
                        objectId: "routine_123",
                        objectType: ResourceType.Routine,
                        subroutineId: "sub_task_2",
                    }],
                    manualExecutionConfirmed: false,
                    nodeStartTimeMs: Date.now() - TWO_SECONDS_MS,
                    processId: "process_main",
                    status: BranchStatus.Active,
                    stepId: "step_2",
                    subroutineInstanceId: "sub_parallel_2",
                    supportsParallelExecution: true,
                },
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
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [],
            metrics: {
                creditsSpent: CREDITS_TEST_AMOUNT,
            },
            subcontexts: {},
        },

        withCompletedDecisions: {
            __version: LATEST_CONFIG_VERSION,
            branches: [{
                branchId: "branch_1",
                childSubroutineInstanceId: null,
                closedLocations: [],
                creditsSpent: CREDITS_LARGE,
                locationStack: [{
                    locationId: "node_1",
                    objectId: "routine_123",
                    objectType: ResourceType.Routine,
                    subroutineId: null,
                }],
                manualExecutionConfirmed: true,
                nodeStartTimeMs: Date.now() - TEN_SECONDS_MS,
                processId: "process_1",
                status: BranchStatus.Completed,
                stepId: "step_1",
                subroutineInstanceId: "sub_instance_1",
                supportsParallelExecution: false,
            }],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Manual,
                    pathSelection: PathSelectionStrategy.ManualPick,
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
                    __type: "Resolved" as const,
                    key: "decision_branch_1_node_1",
                    decisionType: "chooseOne" as const,
                    result: "path_a",
                },
                {
                    __type: "Resolved" as const,
                    key: "decision_branch_1_node_2",
                    decisionType: "chooseOne" as const,
                    result: "input_option_1",
                },
            ],
            metrics: {
                creditsSpent: CREDITS_XLARGE,
            },
            subcontexts: {},
        },

        withErrors: {
            __version: LATEST_CONFIG_VERSION,
            branches: [
                {
                    branchId: "branch_failed",
                    childSubroutineInstanceId: null,
                    closedLocations: [],
                    creditsSpent: "150",
                    locationStack: [{
                        locationId: "node_error",
                        objectId: "routine_123",
                        objectType: ResourceType.Routine,
                        subroutineId: "sub_error",
                    }],
                    manualExecutionConfirmed: false,
                    nodeStartTimeMs: Date.now() - FIVE_SECONDS_MS,
                    processId: "process_error",
                    status: BranchStatus.Failed,
                    stepId: "step_error",
                    subroutineInstanceId: "sub_instance_error",
                    supportsParallelExecution: false,
                },
            ],
            config: {
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                limits: {
                    maxSteps: STEPS_HIGH,
                },
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            },
            decisions: [],
            metrics: {
                creditsSpent: CREDITS_MEDIUM,
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
                    currentTask: {
                        name: "Process Data",
                        description: "Process raw input data",
                        instructions: "Apply transformation rules to input",
                    },
                    allInputsList: [
                        { key: "input", value: "raw data" },
                        { key: "processedAt", value: "2024-01-01T15:00:00Z" },
                    ],
                    allInputsMap: {
                        input: "raw data",
                        processedAt: "2024-01-01T15:00:00Z",
                    },
                    allOutputsList: [
                        { key: "result", value: "processed" },
                        { key: "count", value: 42 },
                    ],
                    allOutputsMap: {
                        result: "processed",
                        count: 42,
                    },
                    triggeredBoundaryEventIds: [],
                    nodeStartTimes: {},
                    runtimeEvents: {
                        messages: [],
                        signals: [],
                        errors: [],
                        escalations: [],
                    },
                    timeZone: "UTC",
                },
                "sub_validate_1": {
                    currentTask: {
                        name: "Validate Data",
                        description: "Validate processed data against rules",
                    },
                    allInputsList: [
                        { key: "target", value: "processed data" },
                        { key: "rules", value: ["rule1", "rule2"] },
                    ],
                    allInputsMap: {
                        target: "processed data",
                        rules: ["rule1", "rule2"],
                    },
                    allOutputsList: [
                        { key: "valid", value: true },
                        { key: "errors", value: [] },
                    ],
                    allOutputsMap: {
                        valid: true,
                        errors: [],
                    },
                    triggeredBoundaryEventIds: [],
                    nodeStartTimes: {},
                    runtimeEvents: {
                        messages: [],
                        signals: [],
                        errors: [],
                        escalations: [],
                    },
                    timeZone: "UTC",
                },
                "sub_transform_1": {
                    currentTask: {
                        name: "Transform Data",
                        description: "Transform validated data to final format",
                    },
                    allInputsList: [
                        { key: "source", value: "validated data" },
                        { key: "format", value: "json" },
                    ],
                    allInputsMap: {
                        source: "validated data",
                        format: "json",
                    },
                    allOutputsList: [
                        { key: "transformed", value: { key: "value" } },
                        { key: "metadata", value: { size: 1024 } },
                    ],
                    allOutputsMap: {
                        transformed: { key: "value" },
                        metadata: { size: 1024 },
                    },
                    triggeredBoundaryEventIds: [],
                    nodeStartTimes: {},
                    runtimeEvents: {
                        messages: [],
                        signals: [],
                        errors: [],
                        escalations: [],
                    },
                    timeZone: "UTC",
                },
            },
        },
    },
};

/**
 * Create a run config with specific decision strategies
 */
export function createRunConfigWithStrategies(
    inputGeneration: InputGenerationStrategy,
    pathSelection: PathSelectionStrategy,
    subroutineExecution: SubroutineExecutionStrategy,
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
        },
    });
}

/**
 * Create a run config with active branches
 */
export function createRunConfigWithBranches(
    branchCount: number,
    status: BranchStatus = BranchStatus.Active,
): RunProgressConfigObject {
    const branches: BranchProgress[] = Array.from({ length: branchCount }, (_, i) => ({
        branchId: `branch_${i + 1}`,
        childSubroutineInstanceId: null,
        closedLocations: [],
        creditsSpent: "0",
        locationStack: [{
            locationId: `node_${i + 1}`,
            objectId: "routine_123",
            objectType: ResourceType.Routine,
            subroutineId: null,
        }],
        manualExecutionConfirmed: false,
        nodeStartTimeMs: Date.now(),
        processId: `process_${i + 1}`,
        status,
        stepId: null,
        subroutineInstanceId: `sub_instance_${i + 1}`,
        supportsParallelExecution: false,
    }));

    return mergeWithBaseDefaults<RunProgressConfigObject>({
        branches,
        metrics: {
            creditsSpent: (branchCount * CREDITS_PER_BRANCH).toString(),
        },
    });
}

/**
 * Create a run config with specific limits
 */
export function createRunConfigWithLimits(
    limits: Partial<RunProgress["config"]["limits"]>,
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
        },
    });
}
