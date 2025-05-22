/* c8 ignore start */
import { type ResourceType, type ResourceVersion, type RunStatus, type RunStepStatus, type SessionUser } from "../api/types.js";
import { type PassableLogger } from "../consts/commonTypes.js";
import { type LlmModel } from "../shape/configs/bot.js";
import { type ScheduleShape } from "../shape/models/models.js";
import { type SubroutineExecutor } from "./executor.js";
import { type RunLimitsManager } from "./limits.js";
import { type RunLoader } from "./loader.js";
import { type NavigatorFactory } from "./navigator.js";
import { type RunNotifier } from "./notifier.js";
import { type PathSelectionHandler } from "./pathSelection.js";
import { type RunPersistence } from "./persistence.js";

/** The minimum information required to identify a run. */
export type RunIdentifier = Pick<RunProgress, "runId">;

/**
 * Information required about the user who triggered the run.
 */
export type RunTriggeredBy = Pick<SessionUser, "hasPremium" | "id" | "languages">;

/**
 * A location in the run.
 * 
 * Contains enough information to be queriable directly, without having to 
 * load previous locations in the stack.
 */
export type Location = {
    /** 
     * The ID of the location within the object.
     * Typically this is a node ID within a routine
     */
    locationId: Id;
    /** 
     * The ID of the object being run.
     * For example, if this location is for a node in a routine, this would be the routine ID 
     * (and not the node ID or subroutine ID).
     */
    objectId: Id;
    /** 
     * The type of object being run
     */
    objectType: `${ResourceType.Project | ResourceType.Routine}`;
    /**
     * The ID of the subroutine, if this location refers to a graph node that points to a subroutine.
     * Can also be the directory ID if running a project.
     */
    subroutineId: Id | null;
};

/**
 * Data loaded from the database for a specific location.
 */
export type LocationData = {
    /** The object being run */
    object: ResourceVersion;
    /** The subroutine in the object being run, if any */
    subroutine: ResourceVersion | null;
}

/**
 * A stack representing the current location in the run.
 * 
 * Each time a nested graph is encountered, a new entry is added to the stack.
 * Each time a nested graph is exited, the last entry is removed from the stack.
 */
export type LocationStack = Location[];

export type Id = string;

/**
 * Behavior when a run reaches a limit.
 */
export type RunLimitBehavior = "Pause" | "Stop";

/**
 * The concurrency mode for the run.
 * This allows us to switch between parallel and sequential execution, 
 * to make sure that we don't run out of credits or severely limit AI response lengths
 * when nearing the max credits limit.
 */
export type ConcurrencyMode = "Parallel" | "Sequential";

/**
 * Limits for a run request, as well as the behavior when those limits are reached. 
 * 
 * NOTE 1: These should be reset when restarting a failed run, but not when resuming a paused run.
 * NOTE 2: Just because a limit is set doesn't mean it will be reached. If a user has 100 credits,
 * but the limit is set to 10_000, the run will stop at 100 credits.
 */
export type RunRequestLimits = {
    /** 
     * Maximum tokens in routine context, which accumulates relevant info 
     * over multiple steps. Having fewer tokens reduces the cost of 
     * generating LLM responses.
     */
    maxContextTokens?: number;
    /** Maximum number of credits that can be spent */
    maxCredits?: string;
    /** Maximum number of steps that can be run */
    maxSteps?: number;
    /** Maximum time that an be spent on the run, in milliseconds */
    maxTime?: number;
    /** What to do on max credits reached */
    onMaxCredits?: RunLimitBehavior;
    /** What to do in max time reached */
    onMaxTime?: RunLimitBehavior;
    /** What to do on max steps reached */
    onMaxSteps?: RunLimitBehavior;
}

/** 
 * How subroutine inputs should be generated.
 */
export enum InputGenerationStrategy {
    /** Let the AI generate the inputs */
    Auto = "Auto",
    /** Let the user provide the inputs */
    Manual = "Manual"
}

/**
 * The type of path selection strategy to use for the run, 
 * when we're in a situation where there are multiple outgoing branches 
 * and we can only pick one or a subset of them.
 */
export enum PathSelectionStrategy {
    /** Pick the first available branch */
    AutoPickFirst = "AutoPickFirst",
    /** Use an AI model to decide which branch to pick */
    AutoPickLLM = "AutoPickLLM",
    /** Pick a random branch */
    AutoPickRandom = "AutoPickRandom",
    /** Let the user decide which branch to pick */
    ManualPick = "ManualPick"
}

/**
 * How subroutines should be executed.
 */
export enum SubroutineExecutionStrategy {
    /** Run the subroutine as soon as the inputs are generated */
    Auto = "Auto",
    /** Make the user press a button to run the subroutine */
    Manual = "Manual"
}

export type RunDecisionConfig = {
    /** 
     * How subroutine inputs should be generated.
     * 
     * `Auto` - Let the AI generate the inputs
     * `Manual` - Let the user provide the inputs
     */
    inputGeneration: InputGenerationStrategy,
    /**
     * The type of path selection strategy to use for the run, 
     * when we're in a situation where there are multiple outgoing branches 
     * and we can only pick one or a subset of them.
     */
    pathSelection: PathSelectionStrategy,
    /** 
     * How subroutines should be executed.
     * 
     * `Auto` - Run the subroutine as soon as the inputs are generated
     * `Manual` - Make the user press a button to run the subroutine
     */
    subroutineExecution: SubroutineExecutionStrategy,
}

/**
 * Bot configuration for a run. 
 * 
 * Used to override/supplement the bot settings for subroutines 
 * when generating inputs and executing Generate routines.
 */
export type RunBotConfig = {
    /** The model to use for the LLM */
    model?: LlmModel | null;
    /**
     * How to handle the model for subroutines.
     * 
     * `OnlyWhenMissing` (DEFAULT) - Only use this config's model if the subroutine doesn't have one
     * `Override` - Use this config's model for every subroutine
     */
    modelHandling?: "OnlyWhenMissing" | "Override";
    /** The prompt to use for the LLM */
    prompt?: string;
    /** 
     * How to handle prompts for subroutines 
     * 
     * `Combine` (DEFAULT) - Pass this config's prompt and the subroutine's prompt to the LLM
     * `Override` - Use only this config's prompt for the subroutine
     */
    promptHandling?: "Combine" | "Override";
    /** The bot to use for the LLM */
    respondingBot?: { id?: Id, publicId?: string, handle?: string } | null;
    /**
     * How to handle the responding bot for subroutines.
     * 
     * `OnlyWhenMissing` (DEFAULT) - Only use this config's responding bot if the subroutine doesn't have one
     * `Override` - Use this config's responding bot for every subroutine
     */
    respondingBotHandling?: "OnlyWhenMissing" | "Override";
}

/**
 * Configuration object for setting up a run with the state machine.
 */
export type RunConfig = {
    /** Bot configuration for the run */
    botConfig: RunBotConfig;
    /**
     * The decision strategies to apply for the run.
     */
    decisionConfig: RunDecisionConfig;
    /**
     * Indicates if the run should be marked as private
     */
    isPrivate: boolean;
    /**
     * Indicates if the task is time-sensitive.
     * If so, the server may prioritize it over other tasks, 
     * at the cost of potentially higher credit usage.
     */
    isTimeSensitive?: boolean;
    /** Limits for the run */
    limits: RunRequestLimits,
    /** Config for the main loop that powers the run */
    loopConfig: RunUntilDoneConfig,
    /**
     * Run behavior when a branch fails for any reason 
     * (and there are more branches to run).
     * 
     * `Continue` - Continue running other branches as if nothing happened
     * `Pause` - Pause the run (meaning you'll have to manually resume it)
     * `Stop` (RECOMMENDED) - Stop the run and consider it failed
     */
    onBranchFailure?: "Continue" | "Pause" | "Stop",
    /**
     * Run behavior when a gateway fork has no active outgoing branches.
     * 
     * NOTE: This still applies for event-based gateways, since it's the subsequent nodes that handle 
     * the event - not the gateway itself.
     * 
     * `Continue` - Close the branch and continue the run
     * `Wait` - Wait for a valid branch to be available
     * `Fail` (RECOMMENDED) - Fail the run
     */
    onGatewayForkFailure?: "Continue" | "Wait" | "Fail",
    /**
     * Run behavior when a node (non-gateway and non-event) has no active outgoing branches.
     * 
     * `Continue` - Close the branch and continue the run
     * `Wait` - Wait for a valid branch to be available
     * `Fail` (RECOMMENDED) - Fail the run
     */
    onNormalNodeFailure?: "Continue" | "Wait" | "Fail",
    /**
     * Run behavior when the only remaining branches for a routine are waiting at gateways, and 
     * at least one branch has already failed or reached the end of the graph.
     * 
     * NOTE: If at least one branch is waiting for an event, this will not apply.
     * 
     * `Continue` (RECOMMENDED) - Close waiting branches and continue the run
     * `Pause` - Pause the run (meaning you'll have to manually resume it)
     * `Stop` - Stop the run and consider it failed
     */
    onOnlyWaitingBranches?: "Continue" | "Pause" | "Stop",
    /**
     * If true, all subroutines generate dummy data instead of running.
     * Should cost 0 credits.
     */
    testMode?: boolean,
}

/**
 * Represents the accumulated stats for current run.
 */
export type RunMetrics = {
    /** The total complexity completed so far */
    complexityCompleted: number;
    /** The total complexity of the run */
    complexityTotal: number;
    /** 
     * How many credits have been spent so far.
     * 
     * NOTE: This is a string because credits are stored using BigInt, which cannot 
     * be sent through the API or websockets.
     */
    creditsSpent: string;
    /** When the run was started */
    startedAt: Date | null;
    /** How many steps (excluding the current step) have been run so far. */
    stepsRun: number;
    /** How much time (in milliseconds) has elapsed in all steps combined */
    timeElapsed: number;
}

export type RunProgressStep = Location & {
    /** The complexity of the step */
    complexity: number;
    /** How many times the user switched tabs while completing the step, if run manually */
    contextSwitches: number;
    /** The ID of the step */
    id: Id;
    /** When the step was started */
    startedAt: Date;
    /** When the step was completed */
    completedAt?: Date;
    /** The name of the step */
    name: string;
    /** The order of the step in the run */
    order: number;
    /** The status of the step */
    status: RunStepStatus;
}

/** Represents the current run state. */
export type RunProgress = {
    /** Store the version number for this shape for future compatibility */
    __version: string;
    /** All live branches in the run (including nested branches) */
    branches: BranchProgress[];
    /** The run configuration, including limits and settings */
    config: RunConfig;
    /**
     * Decisions that have been deferred or resolved. 
     * Required for pausing a resuming a run due to blocking decisions, 
     * such as waiting for a user to manually choose an option.
     */
    decisions: (DeferredDecisionData | ResolvedDecisionData)[];
    /** The name of the run. Typically derived from the object being run */
    name: string;
    /** 
     * Who owns the run.
     * Typically the same as the user that requested the run, but can be different 
     * (e.g. the run is assigned to a team instead of a user).
     */
    owner: { __typename: "User" | "Team", id: Id };
    /** The run ID */
    runId: Id;
    /** The ID of the main object being run (either a routine or project) */
    runOnObjectId: Id | null;
    /** The schedule associated with the run, if any */
    schedule: ScheduleShape | null;
    /** Previous step history, in chronological order */
    steps: RunProgressStep[];
    /** Accumulated metrics (cost, time, etc.) from active and closed branches */
    metrics: RunMetrics;
    /** The current overall run status */
    status: RunStatus;
    /** 
     * The context collected for each multi-step subroutine, keyed by `subroutineInstanceId`. 
     * 
     * We collect subcontexts here so that branches can share context with each other.
     * 
     * We use `subroutineInstanceId` instead of `subroutineId` because the same subroutine  
     * may be triggered multiple times in the same run, and we want to keep the contexts separate.
     */
    subcontexts: Record<string, SubroutineContext>;
}

/**
 * Describes why a run changed status. 
 * Can be used to provide more context to the user or to the AI.
 */
export enum RunStatusChangeReason {
    AutomaticPause = "AutomaticPause",
    AutomaticStop = "AutomaticStop",
    Completed = "Completed",
    Error = "Error",
    MaxCredits = "MaxCredits",
    MaxLoops = "MaxLoops",
    MaxSteps = "MaxSteps",
    MaxTime = "MaxTime",
    UserCanceled = "UserCanceled",
    UserPaused = "UserPaused",
}

/**
 * A simple representation of a run step. 
 * Useful for providing context to an AI, as it removes any 
 * unnecessary details that could confuse the model and increase 
 * the context size.
 */
type RunTask = {
    name: string;
    description: string;
    instructions?: string;
};

export type IOKey = string;
export type IOValue = unknown;
export type IOEntry = { key: IOKey, value: IOValue };
export type IOMap = Record<IOKey, IOValue>;

/**
 * Configuration for `runUntilDone` (i.e. the main loop)
 */
type RunUntilDoneConfig = {
    /**
     * How much of a delay to add between each iteration of the main loop. 
     * Useful for slowing down the run for debugging, tutorials, or to avoid
     * hitting rate limits.
     */
    loopDelayMs?: number;
    /**
     * The current loop delay. This starts at `loopDelayMs` and can be increased
     * if all branches are waiting to avoid unnecessary processing.
     */
    currentLoopDelayMs?: number;
    /**
     * How quickly the loop delay should increase when all branches are waiting.
     */
    loopDelayMultiplier?: number;
    /**
     * The maximum loop delay. If the loop delay reaches this value, it will not increase further.
     */
    maxLoopDelayMs?: number;
}

/** The status of a run branch. */
export enum BranchStatus {
    /** The branch is currently running with no blockers. */
    Active = "Active",
    /** The branch has completed successfully. */
    Completed = "Completed",
    /** The branch has failed. */
    Failed = "Failed",
    /** The branch is waiting for some condition to be met before continuing. */
    Waiting = "Waiting",
}

/**
 * Contains the context data for a subroutine in the run. 
 * This is updated as the run progresses, and is used to make decisions 
 * and help AI models generate responses.
 */
export type SubroutineContext = {
    /** 
     * General information about the subroutine, such as its name, description, etc.
     */
    currentTask: RunTask;
    /**
     * General information about the overall project/routine being run, unless we're running 
     * a single-step routine (in that case, the currentTask is sufficient).
     */
    overallTask?: RunTask;
    /**
     * Ordered list of all input keys defined/generated so far in the subroutine, across all completed nodes.
     * 
     * NOTE: We grab the last k elements from this list to generate context for the AI, where k is the
     * amount of context we can fit based on our token limit.
     */
    allInputsList: IOEntry[];
    /**
     * All inputs defined/generated so far in the subroutine, across all completed nodes.
     */
    allInputsMap: IOMap;
    /**
     * Ordered list of all output keys defined/generated so far in the subroutine, across all completed nodes.
     * 
     * NOTE: We grab the last k elements from this list to generate context for the AI, where k is the
     * amount of context we can fit based on our token limit.
     */
    allOutputsList: IOEntry[];
    /**
     * All outputs defined/generated so far in the subroutine, across all completed nodes.
     */
    allOutputsMap: IOMap;
    /** Previous steps completed in the routine (in order), if any. */
    completedTasks?: RunTask[];
}

/**
 * The progress of a run branch. 
 * 
 * If parallel execution is supported, branches can be created any time there is a decision point.
 * If parallel execution is not supported, we keep a single branch and only ever pick one next step.
 */
export type BranchProgress = {
    /** ID to identify this branch. */
    branchId: Id;
    /** 
     * If this branch triggered a multi-step subroutine, this is the instance ID of the subroutine. 
     * This is needed to determine when the subroutine has completed.
     */
    childSubroutineInstanceId: Id | null;
    /** Locations that are not allowed to be visited again */
    closedLocations: Location[],
    /** The number of credits spent so far in this branch, as a stringified bigint. */
    creditsSpent: string;
    /** The current location in the run. */
    locationStack: LocationStack;
    /** 
     * If we're running in a manual subroutineExecution mode, this flag 
     * lets us know if we've manually pressed the "Run" button for the subroutine.
     * If not, we can't run the subroutine.
     */
    manualExecutionConfirmed: boolean;
    /** The start time for the current node, in milliseconds since epoch. */
    nodeStartTimeMs: number | null;
    /** 
     * When a branch is created at a start node, this ID is generated. 
     * All other branches created during the execution of the graph will have the same ID.
     * 
     * We call this a process ID because in BPMN there should only be one start node triggered per process ID.
     * When a process (think swim lane) has multiple start nodes, a decision should be made to pick one of them.
     */
    processId: Id;
    /** The branch's status */
    status: BranchStatus;
    /** The current run progress step ID linked to this branch */
    stepId: Id | null;
    /**
     * A composite ID of `${subroutineId}.${uniqueId}` to distinguish this instance of a subroutine from others.
     * 
     * If a subroutine is triggered multiple times in the same run, this allows us to 
     * distinguish between them.
     * 
     * NOTE: This is a composite ID so that we can grab the subroutine ID from it if needed.
     */
    subroutineInstanceId: string;
    /** Whether the branch is part of a navigator that supports parallel execution */
    supportsParallelExecution: boolean;
}

/**
 * Where the run is being triggered from
 */
export enum RunTriggeredFrom {
    /**
     * Triggered by an API call
     */
    Api = "Api",
    /**
     * A bot deciding to run the routine itself, perhaps though a meeting
     */
    Bot = "Bot",
    /**
     * Triggered implicity or explicitly by while a user is chatting with a bot
     */
    Chat = "Chat",
    /**
     * Running the routine through the RunView (i.e. manually entering inputs 
     * and generating outputs)
     */
    RunView = "RunView",
    /**
     * Running the routine through a schedule
     */
    Schedule = "Schedule",
    /**
     * A test run
     */
    Test = "Test",
    /**
     * Triggered by a webhook
     */
    Webhook = "Webhook",
}

/**
 * Determines which bot personas to use for a routine.
 */
export enum BotStyle {
    // The default bot
    Default = "Default",
    // Will use ConfigCallDataGenerate.respondingBot
    Specific = "Specific",
    // Don't use a bot
    None = "None",
}

/**
 * A diff for a map of IOKey to IOValue.
 */
export type MapDiff = {
    /** New or updated keys */
    set: Record<IOKey, IOValue>;
    /** Keys that were deleted */
    removed: IOKey[];
};

/**
 * A diff for a subcontext.
 */
export type MinimalSubroutineDiff = {
    allInputsMap: MapDiff;
    allOutputsMap: MapDiff;
};

export type MinimalSubroutineContext = Pick<SubroutineContext, "allInputsMap" | "allOutputsMap">;

export type SubcontextUpdates = Record<string, MinimalSubroutineContext>;

/**
 * Run task information sent through websockets.
 * Should be minimal to avoid sending too much data.
 */
export type RunTaskInfo = {
    /** Information about each current branch */
    activeBranches: Pick<BranchProgress, "locationStack" | "nodeStartTimeMs" | "processId" | "status" | "subroutineInstanceId">[];
    /** Total percentage of the run completed, out of 100 */
    percentComplete: number;
    /** The ID of the run */
    runId: string;
    /** The status of the overall run */
    runStatus: RunStatus;
    /** The reason the run status changed */
    runStatusChangeReason?: RunStatusChangeReason;
    /** Context values created, updated, or removed since the last update */
    subcontextUpdates: SubcontextUpdates;
};

/**
 * A single option for a decision.
 */
export type DecisionOption = {
    /** The ID of the node to move to */
    nodeId: Id;
    /** 
     * Optional data about the node, to provide as additional context.
     * 
     * NOTE: Must be serializable to JSON.
     */
    nodeData?: unknown;
    /** A human-readable label for the node */
    nodeLabel: string;
}

/**
 * A waiting state that may be returned from a decision strategy.
 * This indicates that the decision is not yet made and the run should 
 * be paused until further notice.
 */
export type DeferredDecisionData = {
    /** The type of decision resolved state */
    __type: "Waiting";
    /** 
     * An ID that uniquely identifies this decision, based on the branch ID 
     * and where in the state machine the decision is for.
     */
    key: string;
    /** A message to display to the user, along with the decision options */
    message?: string;
    /** The possible options for the decision */
    options: DecisionOption[];
    /** The type of decision to make */
    decisionType: "chooseOne" | "chooseMultiple";
}

/**
 * Base information for a resolved decision state that's created as a response to a deferred decision.
 */
type ResolvedDecisionDataBase = {
    /** The type of decision resolved state */
    __type: "Resolved";
    /** 
     * An ID that uniquely identifies this decision, based on the branch ID 
     * and where in the state machine the decision is for.
     */
    key: string;
}

/**
 * A resolved decision state that picks one option.
 */
export type ResolvedDecisionDataChooseOne = ResolvedDecisionDataBase & {
    /** The type of decision */
    decisionType: "chooseOne";
    /** The ID of the chosen option */
    result: Id;
}

/**
 * A resolved decision state that picks multiple options.
 */
export type ResolvedDecisionDataChooseMultiple = ResolvedDecisionDataBase & {
    /** The type of decision */
    decisionType: "chooseMultiple";
    /** The IDs of the chosen options */
    result: Id[];
}

/** 
 * A resolved decision state that's created as a response to a deferred decision.
 */
export type ResolvedDecisionData = ResolvedDecisionDataChooseOne | ResolvedDecisionDataChooseMultiple;

/**
 * A map of branch IDs to their current step's location data.
 */
export type BranchLocationDataMap = Record<Id, LocationData & {
    location: Location;
    subcontext: SubroutineContext;
}>;

/**
 * The result of executing a step (i.e. performing the action specified by 
 * the current node, and handling any branching logic).
 * 
 * This is required because steps may be executed in parallel, so we defer
 * updating branch and run progress until all steps have been executed.
 */
export type ExecuteStepResult = {
    /** The ID of the branch the step was executed in */
    branchId: Id,
    /** The new status of the branch */
    branchStatus: BranchStatus,
    /** The total amount of credits spent during this step, as a stringified bigint */
    creditsSpent: string,
    /**
    * Any deferred decisions that need to be made.
    * 
    * This should pause the branch until the decisions are resolved.
    */
    deferredDecisions: DeferredDecisionData[];
    /** Any new branches that should be created as a result of this step */
    newLocations: null | {
        /** The initial subcontext for the new branches */
        initialContext: SubroutineContext,
        /** The locations to move to */
        locations: Location[],
        /** Whether the new branches support parallel execution */
        supportsParallelExecution: boolean,
    }
    /** 
     * Step to update in the run progress.
     * 
     * May be completed or active, depending on if the subroutine finished running.
     */
    step: RunProgressStep | null,
    /** Information about the subroutine that was run, if any */
    subroutineRun: null | {
        /** Inputs generated for every subroutine that was run */
        inputs: IOMap,
        /** Outputs generated for every subroutine that was run */
        outputs: IOMap,
    }
}

/**
 * The status of the state machine.
 */
export enum StateMachineStatus {
    /** No run is active or initialized. */
    Idle = "Idle",
    /** A run has been initialized (`initNewRun` or `initExistingRun`) but not yet started. */
    Initialized = "Initialized",
    /** The run is actively processing (via `runUntilDone` or `runOneIteration`). */
    Running = "Running",
    /** The run has been stopped/paused, but is still initialized in the state machine and can be resumed. */
    Paused = "Paused",
    /** The run has been completed, so it can no longer be resumed. */
    Completed = "Completed",
    /** The run has failed, so it can no longer be resumed. */
    Failed = "Failed",
    /** The state machine is in-between statuses, so no other operations are allowed.     */
    Transitioning = "Transitioning",
}

/**
 * Services of the state machine. 
 * 
 * Services are typically passed in as constructor arguments, but 
 * may be created internally or updated using setter methods.
 */
export type RunStateMachineServices = {
    limitsManager: RunLimitsManager,
    loader: RunLoader,
    logger: PassableLogger,
    navigatorFactory: NavigatorFactory,
    notifier: RunNotifier | null,
    pathSelectionHandler: PathSelectionHandler,
    persistence: RunPersistence,
    subroutineExecutor: SubroutineExecutor,
}

/**
 * State of the state machine.
 */
export type RunStateMachineState = {
    /**
     * The pending configuration update for the current run.
     */
    pendingConfigUpdate: RunConfig | null,
    /**
     * The run identifier for the current run.
     */
    runIdentifier: RunIdentifier | null,
    /**
     * The current status of the state machine.
     */
    status: StateMachineStatus,
    /**
     * The user data for the current run.
     */
    userData: RunTriggeredBy | null,
}

/**
 * The state of the state machine when it is initialized.
 */
export type InitializedRunState = RunStateMachineState & {
    runIdentifier: RunIdentifier,
    userData: RunTriggeredBy,
}

export type SubroutineOutputDisplayInfoProps = ({
    /**
     * The type of the output, however you want to represent it based on the standard version 
     * linked to the input/output. 
     * 
     * Examples:
     * - "JSON"
     * - "CSV"
     * - "Markdown"
     * - "HTML"
     * - "Text"
     * - "True/False"
     * - "List of URLs"
     * - "JavaScript code"
     * - "Python code"
     * - "SQL"
     * - "GraphQL"
     * - "JSON Schema"
     * - "OpenAPI (Swagger)"
     */
    type: string;
    /**
     * The schema that the input/output should conform to, if any.
     * 
     * This is typically a JSON Schema, but could be something else depending on how the 
     * standard version is configured.
     */
    schema: string | undefined;
    // Any other props that are relevant to the input/output can be added here, such as max length, min value, etc.
} & { [key: string]: unknown })

/**
 * Display information for a subroutine output.
 */
export type SubroutineOutputDisplayInfo = {
    /** 
     * What value we'll set if the user or LLM don't provide one.
     * 
     * This should not be shown to the LLM, so as not to influence the AI's output.
     */
    defaultValue: unknown | undefined;
    /** The description of the output */
    description: string | undefined;
    /** The name of the output */
    name: string | undefined;
    /** 
     * The configuration of the input/output, if any.
     * 
     * With this information, we can tell the LLM something like:
     * "Generate the following inputs:
     * {
     *     "inputA": {
     *         "name": "Input A",
     *         "description": "This is the first input",
     *         "isRequired": true,
     *         "type": "JSON object",
     *         "schema": {
     *             "propA": {
     *                 "type": "string",
     *                 "isRequired": true,
     *                 "description": "This is the first property of the input"
     *             }
     *         }
     *     }
     *     //...
     * }
     */
    props: SubroutineOutputDisplayInfoProps | undefined;
    /** The existing value of the output, if any */
    value: unknown | undefined;
}

/**
 * Display information for a subroutine input.
 */
export type SubroutineInputDisplayInfo = SubroutineOutputDisplayInfo & {
    /** Whether the input is required */
    isRequired: boolean;
}

/**
 * A mapping of input and output names to their current value and display information.
 * 
 * It's important that we have enough information for an AI to understand what each input and output is for.
 */
export type SubroutineIOMapping = {
    inputs: Record<string, SubroutineInputDisplayInfo>;
    outputs: Record<string, SubroutineOutputDisplayInfo>;
}
