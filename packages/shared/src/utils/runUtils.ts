import i18next from "i18next";
import { Node, NodeLink, NodeType, ProjectVersion, ProjectVersionDirectory, ProjectVersionDirectorySearchInput, RoutineType, RoutineVersion, RoutineVersionInput, RoutineVersionOutput, RoutineVersionSearchInput, RunProject, RunProjectStepCreateInput, RunProjectUpdateInput, RunRoutine, RunRoutineInput, RunRoutineInputCreateInput, RunRoutineInputUpdateInput, RunRoutineOutput, RunRoutineOutputUpdateInput, RunRoutineStepCreateInput, RunRoutineStepStatus, RunRoutineUpdateInput, RunStatus, TaskStatus } from "../api/generated/graphqlTypes";
import { PassableLogger, Status } from "../consts/commonTypes";
import { InputType } from "../consts/model";
import { generateInitialValues } from "../forms/defaultGenerator";
import { FormSchema } from "../forms/types";
import { uuid, uuidValidate } from "../id/uuid";
import { NodeShape } from "../shape/models/node";
import { NodeLinkShape } from "../shape/models/nodeLink";
import { getTranslation } from "../translations/translationTools";
import { arraysEqual, uniqBy } from "./arrays";
import { LlmModel } from "./bot";
import { exists } from "./exists";
import { isOfType } from "./objects";

/**
 * Where the run is being triggered from
 */
export enum RunFrom {
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
    onMaxCredits?: "Pause" | "Stop";
    /** What to do in max time reached */
    onMaxTime?: "Pause" | "Stop";
    /** What to do on max steps reached */
    onMaxSteps?: "Pause" | "Stop";
}

type RunTask = {
    name: string;
    description: string;
    instructions?: string;
};

export type RunContext = {
    /** 
     * The current subdirectory/subroutine if previous steps have been completed, 
     * or the overall project/routine if not.
     */
    currentTask: RunTask;
    /**
     * The overall project/routine being run, or undefined if we're on the first step.
     */
    overallTask?: RunTask;
    /**
     * All inputs defined/generated so far in the routine, across all steps.
     */
    allInputs: Record<string, unknown>;
    /**
     * All outputs defined/generated so far in the routine, across all steps.
     */
    allOutputs: Record<string, unknown>;
    /**
     * Previous steps completed in the routine, if any.
     */
    previousTasks?: RunTask[];
}

export enum RunStatusChangeReason {
    Completed = "Completed",
    Error = "Error",
    MaxCredits = "MaxCredits",
    MaxSteps = "MaxSteps",
    MaxTime = "MaxTime",
    UserCanceled = "UserCanceled",
    UserPaused = "UserPaused",
}

export type RunRequestPayloadBase = {
    /**
     * Accumulated data from previous steps. This is used to keep track of
     * the context of the routine, and is used to generate responses in
     * LLM models.
     */
    context?: RunContext;
    /**
     * Indicates if the task is time-sensitive
     */
    isTimeSensitive?: boolean;
    /** Limits to stop the run early */
    limits?: RunRequestLimits;
    /** Tracks progress towards the limits */
    metrics?: {
        /** Number of credits spent so far (stringified BigInt) */
        creditsSpent?: string;
        /** Number of steps run so far */
        stepsRun?: number;
        /** Time spent on the run so far */
        timeElapsed?: number;
    };
    /**
     * What triggered the run. This is a factor in determining
     * the queue priority of the task.
     */
    runFrom: RunFrom;
    runId: string;
    runType: "RunProject" | "RunRoutine";
    /**
     * ID of the user who started the run. Either a human if it matches the 
     * session user, or a bot if it doesn't.
     */
    startedById: string;
    /** The latest status of the command. */
    status: TaskStatus | `${TaskStatus}`;
    /** Why the status changed */
    statusChangeReason?: RunStatusChangeReason;
    /**
     * Unique ID to track task 
     * 
     * NOTE: This is the ID of the task job for managing the task queue, not the LlmTask task ID
     */
    taskId: string;
}

/**
 * Run task information sent through websockets
 */
export type RunTaskInfo = RunRequestPayloadBase & {
    /** Inputs to use on current step, if not already in run object */
    inputs?: Record<string, unknown>;
    /** Current subdirectory being run */
    projectVerisonId?: string;
    /** Current subroutine being run */
    routineVersionId?: string;
    /** Updated run data */
    run?: RunProject | RunRoutine;
};

export enum BotStyle {
    Default = "Default",
    Specific = "Specific",
    None = "None",
}
export type BotStyleOption = {
    description: string;
    label: string;
    value: BotStyle;
};

export type ConfigCallDataGenerate = {
    botStyle?: BotStyle;
    maxTokens?: number | null;
    model?: LlmModel | null;
    respondingBot?: string | null;
};
export type ConfigCallData = ConfigCallDataGenerate | Record<string, never>;

export enum RunStepType {
    /**
     * A list of related objects that are completed together. Like a RoutineList, 
     * but for projects.
     * 
     * When run, the user will be prompted to select one of the objects/subdirectories in the list.
     */
    Directory = "Directory",
    /**
     * A list of related steps that are completed together.
     * 
     * When run, the user will be prompted to select one of the steps in the list. If the list is ordered, 
     * the user will be suggested to complete the steps in order.
     */
    RoutineList = "RoutineList",
    /**
     * A choice for what to do next. May require user action if more than one option is available.
     * 
     * When run, the user will be prompted to select one of the options.
     */
    Decision = "Decision",
    /**
     * An end of the routine. No user action needed.
     * 
     * When run, the step is completed automatically.
     */
    End = "End",
    /**
     * Stores nodes and links of a multi-step routine.
     * 
     * When run, the user sees information about the routine.
     */
    MultiRoutine = "MultiRoutine",
    /**
     * Takes in inputs and performs an action (or used as a placeholder until more data is loaded).
     * 
     * When run, the user fills out a form and the submit button may perform an action.
     */
    SingleRoutine = "SingleRoutine",
    /**
     * The start of a routine. No user action needed.
     * 
     * When run, the step is completed automatically.
     */
    Start = "Start",
}

export type RunnableRoutineVersion = Pick<RoutineVersion, "__typename" | "created_at" | "id" | "complexity" | "configCallData" | "configFormInput" | "configFormOutput" | "inputs" | "nodeLinks" | "nodes" | "outputs" | "root" | "routineType" | "translations" | "versionLabel" | "you">
export type RunnableProjectVersion = Pick<ProjectVersion, "__typename" | "created_at" | "id" | "directories" | "root" | "translations" | "versionLabel" | "you">

/** Basic information provided to all routine steps */
export interface BaseStep {
    /** The step's name, taken from its node if relevant */
    name: string,
    /** The step's description, taken from its node if relevant */
    description: string | null,
    /** 
     * The step's location in the run, as a list of natural numbers. Examples:
     * - Root step: []
     * - First node in MultiRoutineStep root: [1]
     * - Second node in MultiRoutineStep root: [2]
     * - Third node in a RoutineListStep belonging to a MultiRoutineStep root: [3, 1]
     */
    location: number[],
}
/** Step information for all nodes in a routine */
export interface NodeStep extends BaseStep {
    /** Location of the next node to run */
    nextLocation: number[] | null,
    /** The ID of this node */
    nodeId: string,
}
/**
 * Implicit step (i.e. created based on how nodes are linked, rather 
 * than being a node itself) that represents a decision point in a routine.
 */
export interface DecisionStep extends BaseStep {
    __type: RunStepType.Decision,
    /** The options to pick */
    options: {
        link: NodeLink,
        step: DecisionStep | EndStep | RoutineListStep,
    }[];
}
/**
 * Node that marks the end of a routine. No action needed from the user.
 */
export interface EndStep extends NodeStep {
    __type: RunStepType.End,
    nextLocation: null, // End of the routine, so no next location
    /** Whether this is considered a "success" or not */
    wasSuccessful: boolean,
}
/**
 * Node that marks the start of a routine. No action needed from the user.
 */
export interface StartStep extends NodeStep {
    __type: RunStepType.Start,
}
/**
 * Either a leaf node (i.e. does not contain any subroutines/substeps of its own) 
 * or a step that needs further querying/processing to build the substeps.
 * 
 * If further processing is needed, this should be replaced with a MultiRoutineStep.
 */
export interface SingleRoutineStep extends BaseStep {
    __type: RunStepType.SingleRoutine,
    /**
     * The routine version that we'll be running in this step, or a multi-step 
     * routine that will be used to convert this step into a MultiRoutineStep
     */
    routineVersion: RunnableRoutineVersion
}
/**
 * A list of related steps in an individual routine node
 */
export interface RoutineListStep extends NodeStep {
    __type: RunStepType.RoutineList,
    /** Whether or not the steps must be run in order */
    isOrdered: boolean,
    /**
     * The ID of the routine version containing this node
     */
    parentRoutineVersionId: string,
    /**
     * The steps in this list. Leaf nodes are represented as SingleRoutineSteps,
     * while subroutines with their own steps (i.e. nodes and links) are represented
     * as MultiRoutineSteps.
     * 
     * Steps that haven't been processed yet are represented as SingleRoutineSteps
     */
    steps: (MultiRoutineStep | SingleRoutineStep)[],
}
/** Step information for a full multi-step routine */
export interface MultiRoutineStep extends BaseStep {
    __type: RunStepType.MultiRoutine,
    /** The nodes in this routine version. Unordered */
    nodes: (DecisionStep | EndStep | RoutineListStep | StartStep)[],
    /** The links in thie routine version. Unordered */
    nodeLinks: NodeLink[],
    /** The ID of this routine version */
    routineVersionId: string,
}
export interface DirectoryStep extends BaseStep {
    __type: RunStepType.Directory,
    /** ID of directory, if this is not the root directory */
    directoryId: string | null,
    hasBeenQueried: boolean,
    isOrdered: boolean,
    isRoot: boolean,
    /** ID of the project version this step is from */
    projectVersionId: string,
    steps: ProjectStep[],
}
export type ProjectStep = DirectoryStep | SingleRoutineStep | MultiRoutineStep;
/** All available step types */
export type RunStep = DecisionStep | DirectoryStep | EndStep | MultiRoutineStep | RoutineListStep | StartStep | SingleRoutineStep;
/** All step types that can be the root step */
export type RootStep = DirectoryStep | MultiRoutineStep | SingleRoutineStep;

const PERCENTS = 100;
/**
 * Calculates the percentage of the run that has been completed.
 * @param completedComplexity The number of completed steps. Ideally shouldn't exceed the 
 * totalComplexity, since steps shouldn't be counted multiple times.
 * @param totalComplexity The total number of steps.
 * @returns The percentage of the run that has been completed, 0-100.
 */
export function getRunPercentComplete(
    completedComplexity: number | null | undefined,
    totalComplexity: number | null | undefined,
) {
    if (!completedComplexity || !totalComplexity || totalComplexity === 0) return 0;
    const percentage = Math.round(completedComplexity as number / totalComplexity * PERCENTS);
    return Math.min(percentage, PERCENTS);
}

/**
 * Determines if two location arrays match, where a location array is an array of step indices
 * @param locationA The first location array 
 * @param locationB The second location array
 * @return True if the location arrays match, false otherwise
 */
export function locationArraysMatch(locationA: number[], locationB: number[]): boolean {
    if (!Array.isArray(locationA) || !Array.isArray(locationB)) {
        return false;
    }
    return arraysEqual(locationA, locationB, (a, b) => a === b);
}

/**
 * Determines if a routine version has subroutines
 * @param routine The routine version to check
 * @return True if the routine version has subroutines, false otherwise
 */
export function routineVersionHasSubroutines(routineVersion: Partial<RoutineVersion>): boolean {
    if (!routineVersion) return false;
    // Only multi-step routines have subroutines
    if (routineVersion.routineType !== RoutineType.MultiStep) return false;
    // You also need nodes and nodeLinks to have subroutines
    if (routineVersion.nodes && routineVersion.nodes.length > 0) return true;
    if (routineVersion.nodeLinks && routineVersion.nodeLinks.length > 0) return true;
    if (routineVersion.nodesCount && routineVersion.nodesCount > 0) return true;
    return false;
}

type IOKeyObject = { id?: string | null; name?: string | null };

const FIELD_NAME_DELIMITER = "-";

/**
 * Creates the formik key for a given input or output object
 * @param io The input or output object. Both RunRoutineInput/Output and RoutineVersionInput/Output should work
 * @param ioType The type of IO being sanitized ('input' or 'output')
 * @returns A sanitized key for the input or output object, or null 
 * if we couldn't generate a key
 */
export function getIOKey(
    io: IOKeyObject,
    ioType: "input" | "output",
): string | null {
    // Prefer to use name, but fall back to ID
    let identifier = io?.name || io?.id;
    if (!identifier) {
        return null;
    }
    // Sanitize identifier by replacing any non-alphanumeric characters (except underscores) with underscores
    identifier = identifier.replace(/[^a-zA-Z0-9_-]/g, "_");
    // Prepend the ioType to the identifier so we can store both inputs and outputs in one formik object
    return `${ioType}${FIELD_NAME_DELIMITER}${identifier}`; // Make sure the delimiter is not one of the sanitized characters
}

/**
 * Creates the formik field name based on a fieldName and prefix
 * @param fieldName The field name to use
 * @param prefix The prefix to use
 * @returns The formik field name
 */
export function getFormikFieldName(fieldName: string, prefix?: string): string {
    return prefix ? `${prefix}${FIELD_NAME_DELIMITER}${fieldName}` : fieldName;
}

export type ExistingInput = Pick<RunRoutineInput, "id" | "data"> & { input: Pick<RunRoutineInput["input"], "id" | "name"> };
export type ExistingOutput = Pick<RunRoutineOutput, "id" | "data"> & { output: Pick<RunRoutineOutput["output"], "id" | "name"> };

type RunIO = ExistingInput | ExistingOutput;

/**
 * Converts existing run inputs or outputs into an object for populating formik
 * @param runIOs The run input or output data
 * @param logger Logger to log errors
 * @param ioType The type of IO being parsed ('input' or 'output')
 * @returns Object to pass into formik setValues function
 */
export function parseRunIO(
    runIOs: RunIO[],
    logger: PassableLogger,
    ioType: "input" | "output",
): object {
    const result: object = {};

    if (!Array.isArray(runIOs)) return result;

    for (const io of runIOs) {
        const ioData = io[ioType as keyof RunIO] as IOKeyObject | undefined;
        if (!ioData) continue;
        const key = getIOKey(io[ioType], ioType);
        if (!key) continue;
        try {
            result[key] = JSON.parse(io.data);
        } catch (error) {
            logger.error(`Error parsing ${ioType} data for ${ioData.id}:`, typeof error === "object" && error !== null ? error : {});
            // In case of parsing error, use the raw string data
            result[key] = io.data;
        }
    }

    return result;
}

export function parseRunInputs(
    runInputs: ExistingInput[],
    logger: PassableLogger,
): object {
    return parseRunIO(runInputs, logger, "input");
}

export function parseRunOutputs(
    runOutputs: ExistingOutput[],
    logger: PassableLogger,
): object {
    return parseRunIO(runOutputs, logger, "output");
}

type GenerateRoutineInitialValuesProps = {
    configFormInput: FormSchema | null | undefined,
    configFormOutput: FormSchema | null | undefined,
    logger: PassableLogger,
    runInputs: ExistingInput[] | null | undefined,
    runOutputs: ExistingOutput[] | null | undefined,
}

/**
 * Combines form schema default values with existing run data
 * to create Formik initial values for displaying a subroutine.
 * 
 * Any run inputs/outputs which don't appear in the form inputs/outpus 
 * will be ignored, as they're likely for another step.
 * 
 * Any form inputs/outputs which don't have a corresponding run and don't have 
 * default values will be set to the InputType's default value or "" to avoid uncontrolled input warnings.
 * 
 * @param configFormInput Form schema for inputs
 * @param configFormOutput Form schema for outputs
 * @param runInputs Optional existing run inputs
 * @param runOutputs Optional existing run outputs
 * @param logger Logger to log errors
 * @returns Tnitial values for Formik to display a subroutine
 */
export function generateRoutineInitialValues({
    configFormInput,
    configFormOutput,
    logger,
    runInputs,
    runOutputs,
}: GenerateRoutineInitialValuesProps): object {
    let initialValues: object = {};

    // Generate initial values from form schema elements (defaults)
    const inputDefaults = generateInitialValues(configFormInput?.elements, "input");
    const outputDefaults = generateInitialValues(configFormOutput?.elements, "output");

    // Add all form data to result
    initialValues = {
        ...inputDefaults,
        ...outputDefaults,
    };

    // Collect run data
    let runData: object = {};
    if (runInputs) {
        runData = {
            ...runData,
            ...parseRunInputs(runInputs, logger),
        };
    }
    if (runOutputs) {
        runData = {
            ...runData,
            ...parseRunOutputs(runOutputs, logger),
        };
    }

    // Override initial values with run data
    for (const key in runData) {
        if (Object.prototype.hasOwnProperty.call(runData, key) && Object.prototype.hasOwnProperty.call(initialValues, key)) {
            initialValues[key] = runData[key];
        }
    }

    return initialValues;
}

export type RunIOUpdateParams<IOType extends "input" | "output"> = {
    /** Existing run inputs or outputs data */
    existingIO: IOType extends "input" ? ExistingInput[] : ExistingOutput[],
    /** Current input data in form */
    formData: object,
    /** If we're updating input or output data */
    ioType: IOType,
    /** Logger */
    logger: PassableLogger,
    /** Current routine's inputs or outputs */
    routineIO: IOType extends "input" ? Pick<RoutineVersionInput, "id" | "name">[] : Pick<RoutineVersionOutput, "id" | "name">[],
    /** ID of current run */
    runRoutineId: string;
}

// export type RunInputsUpdateResult = Pick<RunRoutineUpdateInput, "inputsCreate" | "inputsUpdate" | "inputsDelete">;
export type RunIOUpdateResult<IOType extends "input" | "output"> =
    IOType extends "input" ?
    Pick<RunRoutineUpdateInput, "inputsCreate" | "inputsUpdate" | "inputsDelete"> :
    Pick<RunRoutineUpdateInput, "outputsCreate" | "outputsUpdate" | "outputsDelete">;

/**
 * Converts a run inputs object to a run input update object
 * @returns The run input update input object
 */
export function runIOUpdate<IOType extends "input" | "output">({
    existingIO,
    formData,
    ioType,
    logger,
    routineIO,
    runRoutineId,
}: RunIOUpdateParams<IOType>): RunIOUpdateResult<IOType> {
    // Initialize result
    const result: Record<string, object> = {};

    // Create a map of existing inputs for quick lookup
    const existingIOMap = new Map<string, RunIO>();
    existingIO?.forEach((io: ExistingInput | ExistingOutput) => {
        const ioObject = io[ioType as keyof RunIO] as IOKeyObject;
        // In case some data is missing, store keys for both name and ID
        const keyName = getIOKey({ name: ioObject?.name }, ioType);
        const keyId = getIOKey({ id: ioObject?.id }, ioType);
        if (keyName) existingIOMap.set(keyName, io);
        if (keyId) existingIOMap.set(keyId, io);
    });

    // Process each routine input
    routineIO?.forEach(currRoutineIO => {
        // Find routine input in form data
        const keyName = getIOKey({ name: currRoutineIO?.name }, ioType);
        const keyId = getIOKey({ id: currRoutineIO?.id }, ioType);
        const inputData = (keyName ? formData[keyName] : undefined) || (keyId ? formData[keyId] : undefined);
        if (inputData === undefined) return;

        // Find existing input if it exists
        const existingInput = (keyName ? existingIOMap.get(keyName) : undefined) || (keyId ? existingIOMap.get(keyId) : undefined);
        // Update existing input if found
        if (existingInput) {
            try {
                // Data must be stored as a string
                const stringifiedData = JSON.stringify(inputData);
                // Update existing input if data has changed
                if (existingInput.data !== stringifiedData) {
                    const updateKey = `${ioType as IOType}sUpdate` as const;
                    if (!result[updateKey]) {
                        result[updateKey] = [];
                    }
                    (result[updateKey] as RunRoutineInputUpdateInput[] | RunRoutineOutputUpdateInput[]).push({
                        id: existingInput.id,
                        data: stringifiedData,
                    });
                }
            } catch (error) {
                logger.error("Error stringifying input data", typeof error === "object" && error !== null ? error : {});
            }
        }
        // Otherwise, create new input 
        else {
            const createKey = `${ioType as IOType}sCreate` as const;
            if (!result[createKey]) {
                result[createKey] = [];
            }
            // Create new input
            (result[createKey] as RunRoutineInputCreateInput[]).push({
                id: uuid(),
                // Data must be stored as a string
                data: JSON.stringify(inputData),
                [`${ioType}Connect` as "inputConnect"]: currRoutineIO.id,
                runRoutineConnect: runRoutineId,
            });
        }
    });

    // Return result
    return result;
}

export function runInputsUpdate(params: Omit<RunIOUpdateParams<"input">, "ioType">): RunIOUpdateResult<"input"> {
    return runIOUpdate({ ...params, ioType: "input" });
}

export function runOutputsUpdate(params: Omit<RunIOUpdateParams<"output">, "ioType">): RunIOUpdateResult<"output"> {
    return runIOUpdate({ ...params, ioType: "output" });
}

type RoutineVersionStatusBase = {
    status: Status;
    messages: string[];
}
type RoutineVersionStatusInformational = RoutineVersionStatusBase
type RoutineVersionStatusMultiStep = RoutineVersionStatusBase & {
    nodesById: { [id: string]: Node };
    nodesOnGraph: Node[];
    nodesOffGraph: Node[];
}
type RoutineVersionStatusGenerate = RoutineVersionStatusBase
type RoutineVersionStatusData = RoutineVersionStatusBase
type RoutineVersionStatusAction = RoutineVersionStatusBase
type RoutineVersionStatusCode = RoutineVersionStatusBase
type RoutineVersionStatusApi = RoutineVersionStatusBase
type RoutineVersionStatusSmartContract = RoutineVersionStatusBase
// type RoutineVersionStatus = |
//     RoutineVersionStatusInformational |
//     RoutineVersionStatusMultiStep |
//     RoutineVersionStatusGenerate |
//     RoutineVersionStatusData |
//     RoutineVersionStatusAction |
//     RoutineVersionStatusCode |
//     RoutineVersionStatusApi |
//     RoutineVersionStatusSmartContract
type RoutineVersionStatusMap = {
    [RoutineType.Informational]: RoutineVersionStatusInformational,
    [RoutineType.MultiStep]: RoutineVersionStatusMultiStep,
    [RoutineType.Generate]: RoutineVersionStatusGenerate,
    [RoutineType.Data]: RoutineVersionStatusData,
    [RoutineType.Action]: RoutineVersionStatusAction,
    [RoutineType.Code]: RoutineVersionStatusCode,
    [RoutineType.Api]: RoutineVersionStatusApi,
    [RoutineType.SmartContract]: RoutineVersionStatusSmartContract,
}

function routineVersionStatusInformational(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusInformational {
    return { status: Status.Invalid, messages: ["TODO not implemented"] };
}

export function routineVersionStatusMultiStep(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusMultiStep {
    if (!routineVersion || !routineVersion.nodeLinks || !routineVersion.nodes) {
        return { status: Status.Invalid, messages: ["No node or link data found"], nodesById: {}, nodesOffGraph: [], nodesOnGraph: [] };
    }
    const nodesOnGraph: Node[] = [];
    const nodesOffGraph: Node[] = [];
    const nodesById: { [id: string]: Node } = {};
    const statuses: [Status, string][] = []; // Holds all status messages, so multiple can be displayed
    // Loop through nodes and add to appropriate array (and also populate nodesById dictionary)
    for (const node of routineVersion.nodes) {
        if (exists(node.columnIndex) && exists(node.rowIndex)) {
            nodesOnGraph.push(node);
        } else {
            nodesOffGraph.push(node);
        }
        nodesById[node.id] = node;
    }
    // Now, perform a few checks to make sure that the columnIndexes and rowIndexes are valid
    // 1. Check that (columnIndex, rowIndex) pairs are all unique
    // First check
    // Remove duplicate values from positions dictionary
    const uniqueDict = uniqBy(nodesOnGraph, (n) => `${n.columnIndex}-${n.rowIndex}`);
    // Check if length of removed duplicates is equal to the length of the original positions dictionary
    if (uniqueDict.length !== Object.values(nodesOnGraph).length) {
        return { status: Status.Invalid, messages: ["Ran into error determining node positions"], nodesById, nodesOffGraph: routineVersion.nodes, nodesOnGraph: [] };
    }
    // Now perform checks to see if the routine can be run
    // 1. There is only one start node
    // 2. There is only one linked node which has no incoming edges, and it is the start node
    // 3. Every node that has no outgoing edges is an end node
    // 4. There is at least one end node that's marked as a successful end
    // 5. Validate loop TODO
    // 6. Validate redirects TODO
    // Check 1
    const startNodes = routineVersion.nodes.filter(node => node.nodeType === NodeType.Start);
    if (startNodes.length === 0) {
        statuses.push([Status.Invalid, "No start node found"]);
    }
    else if (startNodes.length > 1) {
        statuses.push([Status.Invalid, "More than one start node found"]);
    }
    // Check 2
    const nodesWithoutIncomingEdges = nodesOnGraph.filter(node => routineVersion.nodeLinks!.every(link => link.to.id !== node.id));
    if (nodesWithoutIncomingEdges.length === 0) {
        //TODO this would be fine with a redirect link
        statuses.push([Status.Invalid, "Error determining start node"]);
    }
    else if (nodesWithoutIncomingEdges.length > 1) {
        statuses.push([Status.Invalid, "Nodes are not fully connected"]);
    }
    // Check 3
    const nodesWithoutOutgoingEdges = nodesOnGraph.filter(node => routineVersion.nodeLinks!.every(link => link.from.id !== node.id));
    if (nodesWithoutOutgoingEdges.length >= 0) {
        // Check that every node without outgoing edges is an end node
        if (nodesWithoutOutgoingEdges.some(node => node.nodeType !== NodeType.End)) {
            statuses.push([Status.Invalid, "Not all paths end with an end node"]);
        }
    }
    // Check 4
    const allEndNodes = nodesOnGraph.filter(node => node.nodeType === NodeType.End);
    const unsuccessfulEndNodes = allEndNodes.filter(node => node?.end?.wasSuccessful !== true);
    if (unsuccessfulEndNodes.length >= allEndNodes.length) {
        statuses.push([Status.Invalid, "No successful end node(s) found"]);
    }
    // Performs checks which make the routine incomplete, but not invalid
    // 1. There are unpositioned nodes
    // 2. Every routine list has at least one subroutine
    // Check 1
    if (nodesOffGraph.length > 0) {
        statuses.push([Status.Incomplete, "Some nodes are not linked"]);
    }
    // Check 2
    if (nodesOnGraph.some(node => node.nodeType === NodeType.RoutineList && node.routineList?.items?.length === 0)) {
        statuses.push([Status.Incomplete, "At least one routine list is empty"]);
    }
    // Return statuses, or valid if no statuses
    if (statuses.length > 0) {
        // Status sent is the worst status
        let status = Status.Incomplete;
        if (statuses.some(status => status[0] === Status.Invalid)) status = Status.Invalid;
        return { status, messages: statuses.map(status => status[1]), nodesById, nodesOffGraph, nodesOnGraph };
    } else {
        return { status: Status.Valid, messages: ["Routine is fully connected"], nodesById, nodesOffGraph, nodesOnGraph };
    }
}

function routineVersionStatusGenerate(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusGenerate {
    // return { status: Status.Invalid, messages: ["TODO not implemented"] };
    return { status: Status.Valid, messages: [] };
}

function routineVersionStatusData(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusData {
    return { status: Status.Invalid, messages: ["TODO not implemented"] };
}

function routineVersionStatusAction(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusAction {
    return { status: Status.Invalid, messages: ["TODO not implemented"] };
}

function routineVersionStatusCode(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusCode {
    return { status: Status.Invalid, messages: ["TODO not implemented"] };
}

function routineVersionStatusApi(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusApi {
    return { status: Status.Invalid, messages: ["TODO not implemented"] };
}

function routineVersionStatusSmartContract(
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusSmartContract {
    return { status: Status.Invalid, messages: ["TODO not implemented"] };
}

const routineTypeToStatusMap = {
    [RoutineType.Informational]: routineVersionStatusInformational,
    [RoutineType.MultiStep]: routineVersionStatusMultiStep,
    [RoutineType.Generate]: routineVersionStatusGenerate,
    [RoutineType.Data]: routineVersionStatusData,
    [RoutineType.Action]: routineVersionStatusAction,
    [RoutineType.Code]: routineVersionStatusCode,
    [RoutineType.Api]: routineVersionStatusApi,
    [RoutineType.SmartContract]: routineVersionStatusSmartContract,
};

/**
 * Calculates the status of a routine (anything that's not valid cannot be run). 
 * May also return some other information which is useful for displaying routines
 * @param routine The routine to check
 */
export function routineVersionStatus<T extends RoutineType>(
    routineType: T,
    routineVersion?: Partial<RoutineVersion> | null,
): RoutineVersionStatusMap[T] {
    const defaultResult = { status: Status.Invalid, messages: ["Unknown routine type"] } as RoutineVersionStatusMap[T];
    if (!routineVersion) return defaultResult;
    const routineVersionStatusFunc = routineTypeToStatusMap[routineType];
    if (!routineVersionStatusFunc) {
        return defaultResult;
    }
    return routineVersionStatusFunc(routineVersion) as RoutineVersionStatusMap[T];
}

type GetProjectVersionStatusResult = {
    status: Status;
    messages: string[];
}

/**
 * Calculates the status of a project (anything that's not valid cannot be run). 
 * Also returns some other information which is useful for displaying projects
 * @param project The project to check
 */
export function projectVersionStatus(projectVersion?: Partial<ProjectVersion> | null): GetProjectVersionStatusResult {
    return {} as any;//TODO
}

/**
 * Multi-step routine initial data, if creating from scratch
 * @param language The language of the routine
 * @returns Initial data for a new routine
 */
export function initializeRoutineGraph(
    language: string,
    routineVersionId: string,
): { nodes: NodeShape[], nodeLinks: NodeLinkShape[] } {
    const startNode: NodeShape = {
        __typename: "Node" as const,
        id: uuid(),
        nodeType: NodeType.Start,
        columnIndex: 0,
        rowIndex: 0,
        routineVersion: { __typename: "RoutineVersion" as const, id: routineVersionId },
        translations: [],
    };
    const routineListNodeId = uuid();
    const routineListNode: NodeShape = {
        __typename: "Node",
        id: routineListNodeId,
        nodeType: NodeType.RoutineList,
        columnIndex: 1,
        rowIndex: 0,
        routineList: {
            __typename: "NodeRoutineList",
            id: uuid(),
            isOptional: false,
            isOrdered: false,
            items: [],
            node: { __typename: "Node", id: routineListNodeId },
        },
        routineVersion: { __typename: "RoutineVersion" as const, id: routineVersionId },
        translations: [{
            __typename: "NodeTranslation",
            id: uuid(),
            language,
            name: "Subroutine 1",
        }] as Node["translations"],
    };
    const endNodeId = uuid();
    const endNode: NodeShape = {
        __typename: "Node",
        id: endNodeId,
        nodeType: NodeType.End,
        columnIndex: 2,
        rowIndex: 0,
        end: {
            __typename: "NodeEnd",
            id: uuid(),
            wasSuccessful: true,
            node: { __typename: "Node", id: endNodeId },
        },
        routineVersion: { __typename: "RoutineVersion" as const, id: routineVersionId },
        translations: [{
            __typename: "NodeTranslation" as const,
            id: uuid(),
            language,
            name: "End",
            description: "",
        }],
    };
    const link1: NodeLinkShape = {
        __typename: "NodeLink",
        id: uuid(),
        from: startNode,
        to: routineListNode,
        whens: [],
        operation: null,
        routineVersion: { __typename: "RoutineVersion" as const, id: routineVersionId },
    };
    const link2: NodeLinkShape = {
        __typename: "NodeLink",
        id: uuid(),
        from: routineListNode,
        to: endNode,
        whens: [],
        operation: null,
        routineVersion: { __typename: "RoutineVersion" as const, id: routineVersionId },
    };
    return {
        nodes: [startNode, routineListNode, endNode],
        nodeLinks: [link1, link2],
    };
}

/**
 * Maximum run nesting supported
 */
export const MAX_RUN_NESTING = 20;

/**
 * Inserts step data into a given RunStep, where id matches. 
 * We recursively search through the main step object to find the location to insert the step.
 * 
 * @param stepData Step to insert into the root step object. Either a MultiRoutineStep or DirectoryStep, 
 * as these are the only step types that represent full routine versions or project versions, and the reason 
 * we need to insert steps is to inject newly-fetched routine versions or project versions into the existing step object.
 * @param rootStep The root step object to insert the step into. Either a MultiRoutineStep, DirectoryStep, or SingleRoutineStep, 
 * as we can run multi-step routines, projects, and single-step routines.
 * @param logger Logger to log errors
 * @returns Updated root step object with the step inserted.
 */
export function insertStep(
    stepData: DirectoryStep | MultiRoutineStep,
    rootStep: RootStep,
    logger: PassableLogger,
): RootStep {
    // Helper function to perform recursive insert
    function recursiveInsert(currStep: RunStep, depth: number): RunStep {
        if (depth > MAX_RUN_NESTING) {
            logger.error("Recursion depth exceeded in insertStep. No step inserted");
            return currStep;
        }
        // If the current step has a list of steps, recursively search through them
        if (currStep.__type === RunStepType.Directory || currStep.__type === RunStepType.RoutineList) {
            // If this is a directory, we may be able to insert the step here
            if (
                currStep.__type === RunStepType.Directory &&
                stepData.__type === RunStepType.Directory &&
                currStep.projectVersionId === stepData.projectVersionId
            ) {
                // Assign the new step data to the current step in a way that doesn't 
                // overwrite any fields that are missing in the new step data
                const result = currStep as DirectoryStep;
                result.description = stepData.description || currStep.description;
                result.name = stepData.name || currStep.name;
                result.directoryId = stepData.directoryId || currStep.directoryId;
                result.hasBeenQueried = stepData.hasBeenQueried ?? currStep.hasBeenQueried;
                result.isOrdered = stepData.isOrdered ?? currStep.isOrdered;
                result.isRoot = stepData.isRoot ?? currStep.isRoot;
                result.steps = stepData.steps || currStep.steps;
                return result;
            }
            // Otherwise, recursively search through the steps
            for (let i = 0; i < currStep.steps.length; i++) {
                currStep.steps[i] = recursiveInsert(currStep.steps[i]!, depth + 1) as (typeof currStep.steps)[number];
            }
        }
        // If the current step has a list of nodes
        else if (currStep.__type === RunStepType.MultiRoutine) {
            // If the currStep and stepData are the same routine, we can insert the step here
            if (
                stepData.__type === RunStepType.MultiRoutine &&
                currStep.routineVersionId === stepData.routineVersionId
            ) {
                // Assign the new step data to the current step in a way that doesn't
                // overwrite any fields that are missing in the new step data
                const result = currStep as MultiRoutineStep;
                result.description = stepData.description || currStep.description;
                result.name = stepData.name || currStep.name;
                result.nodeLinks = stepData.nodeLinks || currStep.nodeLinks;
                result.nodes = stepData.nodes || currStep.nodes;
                return result;
            }
            // Otherwise, recursively search through the nodes
            for (let i = 0; i < currStep.nodes.length; i++) {
                currStep.nodes[i] = recursiveInsert(currStep.nodes[i]!, depth + 1) as (typeof currStep.nodes)[number];
            }
        }
        // If the current step is a SingleRoutineStep (which we use as a placeholder before loading the full routine) 
        // and the stepData is a MultiRoutineStep, we may be able to insert the step here
        else if (
            currStep.__type === RunStepType.SingleRoutine &&
            stepData.__type === RunStepType.MultiRoutine &&
            currStep.routineVersion.id === stepData.routineVersionId
        ) {
            // Return stepData with location. Since these are different types, we can't 
            // do the field-by-field assignment like we did with the other step types
            const result = stepData as MultiRoutineStep;
            result.location = [...currStep.location];
            return result;
        }

        // For all other cases, return the current step unchanged
        return currStep;
    }

    // Start the recursive insertion
    return recursiveInsert(rootStep, 0) as RootStep;
}

/**
 * Uses a location array to find the step at a given location.
 * 
 * NOTE: Must have been queried already
 * @param location A list of natural numbers representing the location of the step in the root step
 * @param rootStep The root step object for the project or routine
 * @returns The requested step, or null if not found
 */
export function stepFromLocation(
    location: number[],
    rootStep: RunStep,
): RunStep | null {
    let currentStep: RunStep = rootStep;

    // If the location is empty, return null
    if (location.length === 0) return null;
    // If the first location is not 1, return null
    if (location[0] !== 1) return null;
    // If the location has a single number, return the root step
    if (location.length === 1) return rootStep;

    // Loop through the location array, skipping the first number
    for (let i = 1; i < location.length; i++) {
        const index = location[i];
        if (index === undefined) return null; // Invalid index
        if (currentStep.__type === RunStepType.Directory || currentStep.__type === RunStepType.RoutineList) {
            if (index > 0 && index <= currentStep.steps.length) {
                currentStep = currentStep.steps[index - 1]!;
            } else {
                return null; // Invalid index
            }
        } else if (currentStep.__type === RunStepType.MultiRoutine) {
            // For MultiRoutineStep, we need to find the corresponding node
            const node = currentStep.nodes.find(node => node.location[node.location.length - 1] === index);
            if (node) {
                currentStep = node;
            } else {
                return null; // Node not found
            }
        } else if (currentStep.__type === RunStepType.Decision) {
            if (index > 0 && index <= currentStep.options.length) {
                currentStep = currentStep.options[index - 1]!.step;
            } else {
                return null; // Invalid index
            }
        } else {
            return null; // Cannot navigate further
        }
    }

    return currentStep;
}

/**
 * Recursively searches for a step in the step tree that satisfies the given predicate.
 * 
 * @param step The step to start the search from
 * @param predicate A function that takes a RunStep and returns a boolean
 * @returns The first step that satisfies the predicate, or null if no such step is found
 */
export function findStep(step: RunStep, predicate: (step: RunStep) => boolean): RunStep | null {
    // Check if the current step satisfies the predicate
    if (predicate(step)) {
        return step;
    }

    // Check child steps based on the step type
    switch (step.__type) {
        case RunStepType.Directory:
        case RunStepType.RoutineList:
            for (const childStep of step.steps) {
                const result = findStep(childStep, predicate);
                if (result) return result;
            }
            break;
        case RunStepType.MultiRoutine:
            for (const node of step.nodes) {
                const result = findStep(node, predicate);
                if (result) return result;
            }
            break;
        case RunStepType.Decision:
            for (const option of step.options) {
                const result = findStep(option.step, predicate);
                if (result) return result;
            }
            break;
        // Other steps don't have children to search
    }

    // If no child steps satisfy the predicate, return null
    return null;
}

/**
 * Uses a location array to find the number of sibling steps at a given location
 * 
 * NOTE: Must be queried already
 * @param location A list of natural numbers representing the location of the step in the root step
 * @param rootStep The root step object for the project or routine
 * @param logger Logger to log errors
 * @returns The number of sibling steps. Or in other words, the number of steps which share the same base location 
 * array (e.g. [4, 2, 1, 1], [4, 2, 1, 2], [4, 2, 1, 3], ...)
 */
export function siblingsAtLocation(
    location: number[],
    rootStep: RunStep,
    logger: PassableLogger,
) {
    // If there are no numbers in the location, it's invalid
    if (location.length === 0) return 0;
    // If there is one number in the location, it's the root. 
    // There can only be one root.
    if (location.length === 1) return 1;

    // Get parent
    const parentLocation = [...location].slice(0, -1);
    const parent = stepFromLocation(parentLocation, rootStep);
    if (!parent) {
        logger.error(`Could not find parent to count siblings. Location: ${location}`);
        return 0;
    }

    // Determine siblings based on step type
    switch (parent.__type) {
        case RunStepType.Directory:
            return parent.steps.length;
        case RunStepType.MultiRoutine:
            return parent.nodes.length;
        case RunStepType.RoutineList:
            return parent.steps.length;
        // Other types can't have children, but if you're somehow here let's just return 1
        default:
            return 1;
    }
}

/**
 * Finds the previous location in a routine or project structure. Rules (in order):
 * 1. If the location is empty or the first step, return null.
 * - If the step is a type of NodeStep (i.e. part of a MultiRoutineStep), 
 * then we need to check the `nextLocation` and `DecisionStep.options` for every sibling node. 
 * If there are any matches, return the first one found.
 * - If the step has siblings and is not the first sibling, go to the first sibling
 * - Return the parent
 * 
 * NOTE: This is not a perfect inverse of `getNextLocation`, as for that we'd have to drill down 
 * the children of the parent's previous sibling. That behavior may not be preferable anyway.
 * @param location The current location
 * @param rootStep The root step object for the project or routine
 * @returns The previous location, or null if not found (e.g., at the start of the routine)
 */
export function getPreviousLocation(
    location: number[],
    rootStep: RootStep | null,
): number[] | null {
    if (!rootStep) return null;

    const currentLocation = [...location];

    // If the location is empty or 1 (i.e. the root step), return null
    if (currentLocation.length <= 1) return null;

    // Retrieve the current step based on the location provided
    const currentStep = stepFromLocation(currentLocation, rootStep);
    if (!currentStep) return null;

    // Check if any sibling or decision option points to the current location
    const parentLocation = [...currentLocation].slice(0, -1);
    const parentStep = stepFromLocation(parentLocation, rootStep);
    if (!parentStep) return null;
    // This only applies for multi-step routines
    if (parentStep.__type === RunStepType.MultiRoutine) {
        const matchingNextLocations = parentStep.nodes
            .filter(node => Array.isArray((node as NodeStep).nextLocation) && locationArraysMatch((node as NodeStep).nextLocation as number[], currentLocation)) as NodeStep[];
        const matchingDecisions = parentStep.nodes
            .filter(node => node.__type === RunStepType.Decision)
            .filter(node => (node as DecisionStep).options.some(option => locationArraysMatch(option.step.location, currentLocation))) as DecisionStep[];
        if (matchingNextLocations.length > 0) {
            return [...(matchingNextLocations[0]!.location as number[])];
        }
        if (matchingDecisions.length > 0) {
            return [...matchingDecisions[0]!.location];
        }
    }

    // If there is a sibling before the current step, return it
    const prevSiblingLocation = [...currentLocation];
    prevSiblingLocation[prevSiblingLocation.length - 1]--;
    const prevSiblingStep = stepFromLocation(prevSiblingLocation, rootStep);
    if (prevSiblingStep) {
        return [...prevSiblingStep.location];
    }

    // Return parent
    return parentLocation;
}

/**
 * Finds the next available location in a routine or project structure. Rules (in order):
 * 1. If the location is empty, return [1] (i.e. the first step)
 * 2. If the step is a DecisionStep or EndStep, it cannot have a next location, so return null
 * // At this point, the step is valid
 * 3. If the step has children, go to the first child
 * 4. If the step has a `nextLocation` field, return that if it's not null
 * 5. If the step has siblings and is not the last sibling, go to the next sibling
 * // At this point, we start backtracking up the parent chain
 * 6. If the parent has a `nextLocation` field, return that if it's not null
 * 7. If the parent has siblings and is not the last sibling, go to the next sibling
 * 8. If all else fails, return null
 * @param location The current location
 * @param rootStep The root step object for the project or routine
 * @returns The next available location, or null if not found (either because there is not next 
 * step or because the current step is a decision)
 */
export function getNextLocation(
    location: number[],
    rootStep: RootStep | null,
): number[] | null {
    if (!rootStep) return null;

    const currentLocation = [...location];

    // If the location is empty, return the first step
    if (currentLocation.length === 0) return [1];

    // Retrieve the current step based on the location provided
    const currentStep = stepFromLocation(location, rootStep);
    if (!currentStep) return null;

    // If the first location points to a DecisionStep, return null. 
    // DecisionSteps don't have a determined next step, since the user has to choose
    const firstStep = stepFromLocation(currentLocation, rootStep);
    if (firstStep?.__type === RunStepType.Decision) {
        return null;
    }

    // Attempt to go to the first child if present
    const childLocation = [...location, 1];
    const childStep = stepFromLocation(childLocation, rootStep);
    if (childStep) return childLocation;

    // If the current step has a defined next location, return that location (even if it's null)
    if (Object.prototype.hasOwnProperty.call(currentStep, "nextLocation")) {
        const nextLocation = (currentStep as NodeStep).nextLocation;
        if (nextLocation) return [...nextLocation];
    }

    // Attempt to navigate to the next sibling if available
    const nextSiblingLocation = [...location];
    nextSiblingLocation[nextSiblingLocation.length - 1]++;
    const siblingStep = stepFromLocation(nextSiblingLocation, rootStep);
    if (siblingStep) return nextSiblingLocation;

    // Start backtracking to find a valid next location from parents
    while (currentLocation.length > 0) {
        currentLocation.pop(); // Move up to the parent
        if (currentLocation.length === 0) break; // If we've reached the root, stop

        // Check the parent's next location
        const parentStep = stepFromLocation(currentLocation, rootStep);
        if (parentStep && Object.prototype.hasOwnProperty.call(parentStep, "nextLocation")) {
            const parentNextLocation = (parentStep as NodeStep).nextLocation;
            if (parentNextLocation) return parentNextLocation;
        }

        // Check the next sibling of the parent
        const parentSiblingLocation = [...currentLocation];
        parentSiblingLocation[parentSiblingLocation.length - 1]++;
        const parentSiblingStep = stepFromLocation(parentSiblingLocation, rootStep);
        if (parentSiblingStep) {
            return parentSiblingLocation;
        }
    }

    return null;
}

/**
 * Determines if a step (either subroutine or directory) needs additional queries, or if it already 
 * has enough data to render
 * @param step The step to check
 * @returns True if the step needs additional queries, false otherwise
 */
export function stepNeedsQuerying(
    step: RunStep | null | undefined,
): boolean {
    if (!step) return false;
    // If it's a subroutine, we need to query when it has its own subroutines. 
    // This works because when the data is queried, the step is replaced with a MultiRoutineStep.
    if (step.__type === RunStepType.SingleRoutine) {
        return step.routineVersion?.routineType === RoutineType.MultiStep;
    }
    // If it's a directory, we need to query when it has not been marked as being queried. 
    // This step type has a query flag because when the data is queried, we add information to the 
    // existing step object, rather than replacing it.
    if (step.__type === RunStepType.Directory) {
        const currDirectory: Partial<DirectoryStep> = step as DirectoryStep;
        return currDirectory.hasBeenQueried === false;
    }
    return false;
}

/**
 * Calculates the complexity of a step
 * @param step The step to calculate the complexity of
 * @param logger Logger to log errors
 * @returns The complexity of the step
 */
export function getStepComplexity(
    step: RunStep,
    logger: PassableLogger,
): number {
    switch (step.__type) {
        // No complexity for start and end steps, since the user doesn't interact with them
        case RunStepType.End:
        case RunStepType.Start:
            return 0;
        // One decision, so one complexity
        case RunStepType.Decision:
            return 1;
        // Complexity of subroutines stored in routine data
        case RunStepType.SingleRoutine:
            return (step as SingleRoutineStep).routineVersion.complexity;
        // Complexity of a routine is the sum of its nodes' complexities
        case RunStepType.MultiRoutine:
            return (step as MultiRoutineStep).nodes.reduce((acc, curr) => acc + getStepComplexity(curr, logger), 0);
        // Complexity of a list is the sum of its children's complexities
        case RunStepType.RoutineList:
        case RunStepType.Directory:
            return (step as RoutineListStep).steps.reduce((acc, curr) => acc + getStepComplexity(curr, logger), 0);
        // Shouldn't reach here
        default:
            logger.error("Unknown step type in getStepComplexity");
            return 0;
    }
}

/**
 * Parses the childOrder string of a project version directory into an ordered array of child IDs
 * @param childOrder Child order string (e.g. "123,456,555,222" or "l(333,222,555),r(888,123,321)")
 * @returns Ordered array of child IDs
 */
export function parseChildOrder(childOrder: string): string[] {
    // Trim the input string
    childOrder = childOrder.trim();

    // If the input is empty, return an empty array
    if (!childOrder) return [];

    // Check if it's the root format
    const rootMatch = childOrder.match(/^l\((.*?)\)\s*,?\s*r\((.*?)\)$/);
    if (rootMatch) {
        const leftOrder = rootMatch[1] ? rootMatch[1].split(",").filter(Boolean) : [];
        const rightOrder = rootMatch[2] ? rootMatch[2].split(",").filter(Boolean) : [];

        // Check for nested parentheses
        if (leftOrder.some(item => item.includes("(") || item.includes(")")) ||
            rightOrder.some(item => item.includes("(") || item.includes(")"))) {
            return [];
        }

        return [...leftOrder, ...rightOrder];
    }

    // Split by comma and/or space
    const parts = childOrder.split(/[,\s]+/).filter(Boolean);

    // Make sure each part is either a UUID or alphanumeric code
    const validParts = parts.filter(part => uuidValidate(part) || /^[a-zA-Z0-9]+$/.test(part));

    // If the number of valid parts doesn't match the original number of parts, return an empty array
    return validParts.length === parts.length ? validParts : [];
}

export type UnsortedSteps = Array<EndStep | RoutineListStep | StartStep>;
export type SortedStepsWithDecisions = Array<DecisionStep | EndStep | RoutineListStep | StartStep>;

/**
 * Traverses the routine graph using Depth-First Search (DFS) and sorts steps by visitation order. 
 * Adds DecisionSteps where multiple outgoing links are found. These are used to move the run pointer 
 * to a different part of the steps array, ensuring that we can handle multiple paths and cycles 
 * when running the routine.
 * @param steps The nodes in the routine version, represented as steps
 * @param nodeLinks The links connecting the nodes in the routine graph.
 * @param logger Logger to log errors
 * @returns An array of nodes sorted by the order they were visited.
 */
export function sortStepsAndAddDecisions(
    steps: UnsortedSteps,
    nodeLinks: NodeLink[],
    logger: PassableLogger,
): SortedStepsWithDecisions {
    const startStep = steps.find(step => step.__type === RunStepType.Start);
    if (!startStep) {
        logger.error("Routine does not have a StartStep. Cannot sort steps or generate DecisionSteps");
        return steps;
    }

    const visited: { [nodeId: string]: boolean } = {};
    let lastEndLocation = 1;
    const result: SortedStepsWithDecisions = [];

    // Get all but the last element
    const baseLocation = [...startStep.location].slice(0, -1);

    // Helper function to perform DFS
    function dfs(currentStep: StartStep | EndStep | RoutineListStep) {
        if (visited[currentStep.nodeId]) return;

        // Mark the current step as visited
        visited[currentStep.nodeId] = true;
        // Update location for the current step
        currentStep.location = [...baseLocation, lastEndLocation];
        lastEndLocation++;

        // Get all outgoing links from the current step's node
        const outgoingLinks = nodeLinks.filter(link => link.from?.id === currentStep.nodeId);
        // If there's one outgoing link, set the nextLocation to the next node's location
        if (outgoingLinks.length === 1 && outgoingLinks[0]!.to?.id) {
            const nextNode = steps.find((step) => step.nodeId === outgoingLinks[0]!.to!.id);
            if (nextNode) {
                currentStep.nextLocation = visited[nextNode.nodeId]
                    ? [...nextNode.location]
                    : [...baseLocation, lastEndLocation];
            }
        }
        // If there are multiple outgoing links, set the nextLocation to the DecisionStep we're about to create
        else if (outgoingLinks.length > 1) {
            currentStep.nextLocation = [...baseLocation, lastEndLocation];
        } else {
            currentStep.nextLocation = null;
        }

        // If the step is a RoutineListStep, update its steps' locations
        if (currentStep.__type === RunStepType.RoutineList) {
            (currentStep as RoutineListStep).steps.forEach((step, index) => {
                step.location = [...currentStep.location, index + 1];
            });
        }

        // Add current step to result
        result.push(currentStep);

        // If there is more than one outgoing link, generate a DecisionStep
        if (outgoingLinks.length > 1) {
            const decisionStep: DecisionStep = {
                __type: RunStepType.Decision,
                description: "Select a path to continue",
                location: [...baseLocation, lastEndLocation],
                name: "Decision",
                options: outgoingLinks.map(link => {
                    const nextStep = steps.find(step => step.nodeId === link.to?.id);
                    if (
                        nextStep !== null &&
                        nextStep !== undefined &&
                        nextStep.__type !== RunStepType.Start
                    ) {
                        return {
                            link,
                            step: nextStep,
                        } as DecisionStep["options"][0];
                    }
                    return null;
                }).filter(Boolean) as unknown as DecisionStep["options"],
            };
            lastEndLocation++;
            // Add DecisionStep to result
            result.push(decisionStep);
        }
        // Traverse all outgoing links
        outgoingLinks.forEach(link => {
            const nextNode = steps.find(step => step.nodeId === link.to?.id);
            if (nextNode) {
                dfs(nextNode);
            }
        });
    }

    // Start DFS from the start node
    dfs(startStep as StartStep);

    return result;
}

/**
 * Converts a single-step routine into a Step object
 * @param routineVersion The routineVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
export function singleRoutineToStep(
    routineVersion: RunnableRoutineVersion,
    location: number[],
    languages: string[],
): SingleRoutineStep | null {
    return {
        __type: RunStepType.SingleRoutine,
        description: getTranslation(routineVersion, languages, true).description || null,
        location: [...location],
        name: getTranslation(routineVersion, languages, true).name || "Untitled",
        routineVersion,
    };
}

/**
 * Converts a multi-step routine into a Step object.
 * 
 * NOTE: This is only designed for one level of nodes. All subroutines will be 
 * treated as SingleRoutineSteps, and must be converted to MultiRoutineSteps separately.
 * @param routineVersion The routineVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @param logger Logger to log errors
 * @returns RootStep for the given object, or null if invalid
 */
export function multiRoutineToStep(
    routineVersion: RunnableRoutineVersion,
    location: number[],
    languages: string[],
    logger: PassableLogger,
): MultiRoutineStep | null {
    // Convert existing nodes into steps
    const unsorted = (routineVersion.nodes || []).map(node => {
        const description = getTranslation(node, languages, true).description || null;
        const name = getTranslation(node, languages, true).name || "Untitled";
        const nodeId = node.id;

        switch (node.nodeType) {
            case NodeType.End: {
                return {
                    __type: RunStepType.End,
                    description,
                    location: [...location, 1], // Sort function will correct this
                    name,
                    nextLocation: null,
                    nodeId,
                    wasSuccessful: node.end?.wasSuccessful ?? false,
                } as EndStep;
            }
            case NodeType.RoutineList: {
                const sortedRoutineListSteps = node.routineList?.items?.sort((a, b) => {
                    return (a.index || 0) - (b.index || 0);
                }) || [];
                return {
                    __type: RunStepType.RoutineList,
                    description,
                    location: [...location, 1], // Sort function will correct this
                    name,
                    nextLocation: [...location, 1], // Sort function will correct this
                    nodeId,
                    isOrdered: node.routineList?.isOrdered ?? false,
                    parentRoutineVersionId: routineVersion.id,
                    steps: sortedRoutineListSteps.map(item => {
                        return {
                            __type: RunStepType.SingleRoutine,
                            description: getTranslation(item.routineVersion, languages, true).description || null,
                            location: [...location, 1, 1], // Sort function will correct this
                            name: getTranslation(item.routineVersion, languages, true).name || "Untitled",
                            routineVersion: item.routineVersion,
                        };
                    }) ?? [],
                } as RoutineListStep;
            }
            case NodeType.Start: {
                return {
                    __type: RunStepType.Start,
                    description,
                    location: [...location, 1], // Sort function will correct this
                    name,
                    nextLocation: [...location, 1], // Sort function will correct this
                    nodeId,
                } as StartStep;
            }
            default:
                return null;
        }
    }).filter(Boolean) as UnsortedSteps;
    // Sort steps by visitation order and add DecisionSteps where needed
    const sorted = sortStepsAndAddDecisions(unsorted, routineVersion.nodeLinks || [], logger);
    return {
        __type: RunStepType.MultiRoutine,
        description: getTranslation(routineVersion, languages, true).description || null,
        location: [...location],
        name: getTranslation(routineVersion, languages, true).name || "Untitled",
        nodeLinks: routineVersion.nodeLinks || [],
        nodes: sorted,
        routineVersionId: routineVersion.id,
    };
}

/**
 * Converts a project into a Step object
 * @param projectVersion The projectVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @returns RootStep for the given object, or null if invalid
 */
export function projectToStep(
    projectVersion: RunnableProjectVersion,
    location: number[],
    languages: string[],
): DirectoryStep | null {
    // Projects are represented as root directories
    const steps: DirectoryStep[] = [];
    for (let i = 0; i < projectVersion.directories.length; i++) {
        const directory = projectVersion.directories[i]!;
        const directoryStep = directoryToStep(directory, [...location, i + 1], languages, projectVersion.id);
        if (directoryStep) {
            steps.push(directoryStep);
        }
    }
    //TODO project version does not have order field, so can't order directories. Maybe change this in future
    return {
        __type: RunStepType.Directory,
        description: getTranslation(projectVersion, languages, true).description || null,
        directoryId: null,
        hasBeenQueried: true,
        isOrdered: false,
        isRoot: true,
        location: [...location],
        name: getTranslation(projectVersion, languages, true).name || "Untitled",
        projectVersionId: projectVersion.id,
        steps,
    };
}

/**
 * Converts a directory into a Step object
 * @param projectVersionDirectory The projectVersionDirectory being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @param projectVersionId The project version ID that the directory belongs to, if calling from projectToStep
 * @returns RootStep for the given object, or null if invalid
 */
export function directoryToStep(
    projectVersionDirectory: ProjectVersionDirectory,
    location: number[],
    languages: string[],
    projectVersionId?: string,
): DirectoryStep | null {
    // TODO we currently only generate steps for routines in the directory. 
    // We can add more types to support other children later, and update the 
    // directory type to support nested directories.
    // const childOrder = parseChildOrder(projectVersionDirectory.childOrder || "");
    return {
        __type: RunStepType.Directory,
        description: getTranslation(projectVersionDirectory, languages, true).description || null,
        directoryId: projectVersionDirectory.id,
        hasBeenQueried: true,
        isOrdered: false,
        isRoot: false,
        location: [...location],
        name: getTranslation(projectVersionDirectory, languages, true).name || "Untitled",
        projectVersionId: projectVersionId || projectVersionDirectory.projectVersion?.id || "",
        steps: [],
    };
}

/**
 * Converts a runnable object into a Step object
 * @param runnableObject The projectVersion or routineVersion being run
 * @param location The location we should give to the step we're creating
 * @param languages Preferred languages to display step data in
 * @param logger Logger to log errors
 * @returns RootStep for the given object, or null if invalid
 */
export function runnableObjectToStep(
    runnableObject: RunnableProjectVersion | RunnableRoutineVersion | null | undefined,
    location: number[],
    languages: string[],
    logger: PassableLogger,
): RootStep | null {
    if (isOfType(runnableObject, "RoutineVersion")) {
        if (runnableObject.routineType === RoutineType.MultiStep) {
            return multiRoutineToStep(runnableObject, [...location], languages, logger);
        } else {
            return singleRoutineToStep(runnableObject, [...location], languages);
        }
    } else if (isOfType(runnableObject, "ProjectVersion")) {
        return projectToStep(runnableObject, [...location], languages);
    }
    logger.error(`Invalid runnable object type in runnableObjectToStep: ${runnableObject !== null && typeof runnableObject === "object" ? (runnableObject as { __type: string }).__type : "runnableObject is not defined"}`);
    return null;
}

export type DetectSubstepLoadResult = {
    needsFurtherQuerying: boolean;
};

/**
 * Adds a list of subroutines to the root step object
 * @param subroutines List of subroutines to add
 * @param rootStep The root step object to add the subroutines to
 * @param languages Preferred languages to display step data in
 * @param logger Logger to log errors
 * @returns Updated root step object with the subroutines added
 */
export function addSubroutinesToStep(
    subroutines: RunnableRoutineVersion[],
    rootStep: RootStep,
    languages: string[],
    logger: PassableLogger,
): RootStep {
    let updatedRootStep = rootStep;
    for (const routineVersion of subroutines) {
        // We can only inject multi-step routines, as they carry all subroutines with them
        if (routineVersion.routineType !== RoutineType.MultiStep) continue;
        // Find the location we should insert the routine at
        const location = findStep(
            rootStep,
            // We use single-step routines as a placeholder before loading the full routine
            (step) => step.__type === RunStepType.SingleRoutine && (step as SingleRoutineStep).routineVersion?.id === routineVersion.id,
        )?.location;
        if (!location) {
            logger.error("Could not find location to insert routine", routineVersion);
            continue;
        }
        const subroutineStep = multiRoutineToStep(routineVersion, [...location], languages, logger);
        if (!subroutineStep) {
            logger.error("Could not convert routine to step", routineVersion);
            continue;
        }
        updatedRootStep = insertStep(subroutineStep as MultiRoutineStep, updatedRootStep, logger);
    }
    return updatedRootStep;
}

/**
 * Adds a list of subdirectories to the root step object
 * @param subdirectories List of subdirectories to add
 * @param rootStep The root step object to add the subdirectories to
 * @param languages Preferred languages to display step data in
 * @param logger Logger to log errors
 * @returns Updated root step object with the subdirectories added
 */
export function addSubdirectoriesToStep(
    subdirectories: ProjectVersionDirectory[],
    rootStep: RootStep,
    languages: string[],
    logger: PassableLogger,
): RootStep {
    let updatedRootStep = rootStep;
    for (const directory of subdirectories) {
        // Find location to insert the directory at
        const location = findStep(
            rootStep,
            (step) => step.__type === RunStepType.Directory && (step as DirectoryStep).directoryId === directory.id,
        )?.location;
        if (!location) {
            logger.error("Could not find location to insert directory", directory);
            continue;
        }
        const projectStep = directoryToStep(directory, [...location], languages);
        if (!projectStep) {
            logger.error("Could not convert directory to step", directory);
            continue;
        }
        updatedRootStep = insertStep(projectStep, updatedRootStep, logger);
    }
    return updatedRootStep;
}

/**
 * Decides if a new step location should trigger loading of new 
 * subroutine/subdirectory data
 * @param newLocation The new location to check
 * @param rootStep The root step object for the project or routine
 * @param getDirectories Callback to load directory data
 * @param getSubroutines Callback to load subroutine data
 * @param logger Logger to log errors
 * @returns An object indicating if further querying is needed
 */
export function detectSubstepLoad(
    newLocation: number[],
    rootStep: RootStep | null,
    getDirectories: (input: ProjectVersionDirectorySearchInput) => unknown,
    getSubroutines: (input: RoutineVersionSearchInput) => unknown,
    logger: PassableLogger,
): DetectSubstepLoadResult {
    const result = { needsFurtherQuerying: false };
    if (!rootStep || newLocation.length === 0) return result;

    const directoryIds: string[] = [];
    const routineVersionIds: string[] = [];

    // Find the deepest existing ancestor
    const currentLocation = [...newLocation];
    let currentStep: RunStep | null = null;
    while (currentLocation.length > 0 && !currentStep) {
        currentStep = stepFromLocation(currentLocation, rootStep);
        if (!currentStep) {
            currentLocation.pop();
        }
    }

    // If we didn't find an ancestor, the location is probably invalid
    if (!currentStep) return { needsFurtherQuerying: false };

    // Helper function to check if a step needs querying
    function checkStep(step: RunStep) {
        if (stepNeedsQuerying(step)) {
            if (step.__type === RunStepType.Directory) {
                if (step.directoryId) {
                    directoryIds.push(step.directoryId);
                } else {
                    logger.error("DirectoryStep has no directoryId. Cannot query subdirectories");
                }
            } else if (step.__type === RunStepType.SingleRoutine) {
                if (step.routineVersion?.id) {
                    routineVersionIds.push(step.routineVersion.id);
                } else {
                    logger.error("SingleRoutineStep has no routineVersionId. Cannot query subroutines");
                }
            }
        }
    }

    // Check if the step needs querying
    checkStep(currentStep);

    // Check if any children need querying
    if (currentStep.__type === RunStepType.Directory || currentStep.__type === RunStepType.RoutineList) {
        for (const step of currentStep.steps) {
            checkStep(step);
        }
    } else if (currentStep.__type === RunStepType.MultiRoutine) {
        for (const node of currentStep.nodes) {
            checkStep(node);
        }
    }

    // Query subdirectories
    if (directoryIds.length > 0) {
        getDirectories({ ids: directoryIds });
    }
    // Query subroutines
    if (routineVersionIds.length > 0) {
        getSubroutines({ ids: routineVersionIds });
    }

    // If an intermediate step is missing, we need to query further
    result.needsFurtherQuerying = currentLocation.length < newLocation.length;

    return result;
}

type SaveProgressProps = {
    /** Total context switches in the step, including previously saved context switches */
    contextSwitches: number;
    /** The current step data */
    currentStep: RunStep | null | undefined;
    /** The current order in the step array */
    currentStepOrder: number;
    /** The data stored in the run about the current step */
    currentStepRunData: { id: string } | null | undefined;
    /** Current input data in form */
    formData: object,
    /** Callback to update RunProject data */
    handleRunProjectUpdate: (inputs: RunProjectUpdateInput) => Promise<unknown>;
    /** Callback to update RunRoutine data */
    handleRunRoutineUpdate: (inputs: RunRoutineUpdateInput) => Promise<unknown>;
    /** Whether the current step should be considered completed */
    isStepCompleted: boolean;
    /** Whether the entire run should be considered completed */
    isRunCompleted: boolean;
    /** Logger to log erros */
    logger: PassableLogger;
    /** The current run */
    run: RunProject | RunRoutine;
    /** The overall object being run */
    runnableObject: RunnableRoutineVersion | RunnableProjectVersion
    /** Total time elapsed in the step, including previously saved time */
    timeElapsed: number;
}

/**
 * Stores current progress, both for overall routine and the current subroutine
 */
export async function saveRunProgress({
    contextSwitches,
    currentStep,
    currentStepOrder,
    currentStepRunData,
    formData,
    handleRunProjectUpdate,
    handleRunRoutineUpdate,
    isStepCompleted,
    isRunCompleted,
    logger,
    run,
    runnableObject,
    timeElapsed,
}: SaveProgressProps): Promise<void> {
    const runData: RunProjectUpdateInput | RunRoutineUpdateInput = {
        id: run.id,
        contextSwitches: (run.steps ?? []).reduce((acc, curr) => {
            const isCurrentStep = currentStepRunData?.id === curr.id;
            return acc + (isCurrentStep ? contextSwitches : curr.contextSwitches ?? 0);
        }, 0),
        status: isRunCompleted ? RunStatus.Completed : RunStatus.InProgress,
        timeElapsed: (run.steps ?? []).reduce((acc, curr) => {
            const isCurrentStep = currentStepRunData?.id === curr.id;
            return acc + (isCurrentStep ? timeElapsed : curr.timeElapsed ?? 0);
        }, 0),
    };

    // Handle step data
    const commonStepData = {
        timeElapsed,
        contextSwitches,
        status: isStepCompleted ? RunRoutineStepStatus.Completed : RunRoutineStepStatus.InProgress,
    } as const;
    // Update step data if found
    if (currentStepRunData) {
        runData.stepsUpdate = [{
            ...commonStepData,
            id: currentStepRunData.id,
        }];
    }
    // Create step data if not found
    else if (currentStep) {
        const stepCreate = {
            ...commonStepData,
            id: uuid(),
            name: currentStep.name,
            nodeConnect: (currentStep as NodeStep).nodeId ?? undefined,
            order: currentStepOrder,
            step: currentStep.location,
            subroutineConnect: (currentStep as SingleRoutineStep).routineVersion?.id ?? undefined,
        } as const;
        if (runnableObject.__typename === "ProjectVersion") {
            runData.stepsCreate = [{
                ...stepCreate,
                runProjectConnect: run.id,
            }] as unknown as RunProjectStepCreateInput[];
        } else {
            runData.stepsCreate = [{
                ...stepCreate,
                runRoutineConnect: run.id,
            }] as RunRoutineStepCreateInput[];
        }
        // Update overall run metrics
        runData.contextSwitches = (runData.contextSwitches ?? 0) + contextSwitches;
        runData.timeElapsed = (runData.timeElapsed ?? 0) + timeElapsed;
    }

    // Handle routine data
    if (run.__typename === "RunRoutine") {
        const { inputsCreate, inputsDelete, inputsUpdate } = runInputsUpdate({
            existingIO: (run as RunRoutine).inputs,
            formData,
            logger,
            routineIO: (runnableObject as RunnableRoutineVersion).inputs,
            runRoutineId: runData.id,
        });
        const { outputsCreate, outputsDelete, outputsUpdate } = runOutputsUpdate({
            existingIO: (run as RunRoutine).outputs,
            formData,
            logger,
            routineIO: (runnableObject as RunnableRoutineVersion).outputs,
            runRoutineId: runData.id,
        });
        const runRoutineData = runData as RunRoutineUpdateInput;
        runRoutineData.inputsCreate = inputsCreate;
        runRoutineData.inputsDelete = inputsDelete;
        runRoutineData.inputsUpdate = inputsUpdate;
        runRoutineData.outputsCreate = outputsCreate;
        runRoutineData.outputsDelete = outputsDelete;
        runRoutineData.outputsUpdate = outputsUpdate;
        await handleRunRoutineUpdate(runRoutineData);
    }
    // Handle project data
    else {
        const runProjectData = runData as RunProjectUpdateInput;
        await handleRunProjectUpdate(runProjectData);
    }
}

/**
 * @returns The default input form object for a routine. This is a function 
 * to avoid using thes same object in multiple places, which can lead to
 * unexpected behavior if the object is modified.
 */
export function defaultSchemaInput(): FormSchema {
    return { containers: [], elements: [] };
}

/**
 * @returns The default output form object for a routine. This is a function
 * to avoid using the same object in multiple places, which can lead to
 * unexpected behavior if the object is modified.
 */
export function defaultSchemaOutput(): FormSchema {
    return { containers: [], elements: [] };
}

/**
 * @return The default output form object for a Generate routine, 
 * which always returns text (for now, since we only call LLMs)
 */
export function defaultSchemaOutputGenerate(): FormSchema {
    return {
        containers: [],
        elements: [
            {
                fieldName: "response",
                id: "response",
                label: i18next.t("Response", { count: 1 }),
                props: {
                    placeholder: "Model response will be displayed here",
                },
                type: InputType.Text,
            },
        ],
    };
}

export const defaultConfigFormInputMap: { [key in RoutineType]: (() => FormSchema) } = {
    [RoutineType.Action]: () => defaultSchemaInput(),
    [RoutineType.Api]: () => defaultSchemaInput(),
    [RoutineType.Code]: () => defaultSchemaInput(),
    [RoutineType.Data]: () => defaultSchemaInput(),
    [RoutineType.Generate]: () => defaultSchemaInput(),
    [RoutineType.Informational]: () => defaultSchemaInput(),
    [RoutineType.MultiStep]: () => defaultSchemaInput(),
    [RoutineType.SmartContract]: () => defaultSchemaInput(),
};

export const defaultConfigFormOutputMap: { [key in RoutineType]: (() => FormSchema) } = {
    [RoutineType.Action]: () => defaultSchemaOutput(),
    [RoutineType.Api]: () => defaultSchemaOutput(),
    [RoutineType.Code]: () => defaultSchemaOutput(),
    [RoutineType.Data]: () => defaultSchemaOutput(),
    [RoutineType.Generate]: () => defaultSchemaOutputGenerate(),
    [RoutineType.Informational]: () => defaultSchemaOutput(),
    [RoutineType.MultiStep]: () => defaultSchemaOutput(),
    [RoutineType.SmartContract]: () => defaultSchemaOutput(),
};

export function parseSchema(
    value: unknown,
    defaultSchemaFn: () => FormSchema,
    logger: PassableLogger,
    schemaType: string,
): FormSchema {
    let parsedValue: FormSchema;

    try {
        parsedValue = JSON.parse(typeof value === "string" ? value : JSON.stringify(value));
    } catch (error) {
        logger.error(`Error parsing schema ${schemaType}: ${JSON.stringify(error)}`);
        parsedValue = defaultSchemaFn();
    }

    // Ensure the parsed value contains `containers` and `elements` arrays
    if (typeof parsedValue !== "object" || parsedValue === null) {
        parsedValue = defaultSchemaFn();
    }
    if (!Array.isArray(parsedValue.containers)) {
        parsedValue.containers = [];
    }
    if (!Array.isArray(parsedValue.elements)) {
        parsedValue.elements = [];
    }

    return parsedValue;
}

export function parseSchemaInput(
    value: unknown,
    routineType: RoutineType | null | undefined,
    logger: PassableLogger,
): FormSchema {
    return parseSchema(value, () => {
        if (!routineType) return defaultSchemaInput();
        return defaultConfigFormInputMap[routineType]();
    }, logger, "input");
}

export function parseSchemaOutput(
    value: unknown,
    routineType: RoutineType | null | undefined,
    logger: PassableLogger,
): FormSchema {
    return parseSchema(value, () => {
        if (!routineType) return defaultSchemaOutput();
        return defaultConfigFormOutputMap[routineType]();
    }, logger, "output");
}

export function defaultConfigCallData(): ConfigCallData {
    return {};
}

export function defaultConfigCallDataGenerate(): ConfigCallDataGenerate {
    return {
        botStyle: BotStyle.Default,
        maxTokens: null,
        model: null,
        respondingBot: null,
    };
}

export const defaultConfigCallDataMap: { [key in RoutineType]: (() => ConfigCallData) } = {
    [RoutineType.Action]: () => defaultConfigCallData(),
    [RoutineType.Api]: () => defaultConfigCallData(),
    [RoutineType.Code]: () => defaultConfigCallData(),
    [RoutineType.Data]: () => defaultConfigCallData(),
    [RoutineType.Generate]: () => defaultConfigCallDataGenerate(),
    [RoutineType.Informational]: () => defaultConfigCallData(),
    [RoutineType.MultiStep]: () => defaultConfigCallData(),
    [RoutineType.SmartContract]: () => defaultConfigCallData(),
};

export function parseConfigCallData(
    value: unknown,
    routineType: RoutineType | null | undefined,
    logger: PassableLogger,
): ConfigCallData {
    let parsedValue: ConfigCallData;
    const defaultConfigCallData = routineType ? (defaultConfigCallDataMap[routineType] || {}) : {};

    try {
        parsedValue = JSON.parse(typeof value === "string" ? value : JSON.stringify(value));
    } catch (error) {
        logger.error(`Error parsing configCallData: ${JSON.stringify(error)}`);
        parsedValue = defaultConfigCallData;
    }

    if (typeof parsedValue !== "object" || parsedValue === null) {
        parsedValue = defaultConfigCallData;
    }

    return parsedValue;
}

export type ShouldStopParams = {
    currentTimeElapsed: number;
    limits: RunRequestLimits | null | undefined;
    maxCredits: bigint;
    maxSteps: number;
    maxTime: number;
    previousTimeElapsed: number;
    stepsRun: number;
    totalStepCost: bigint;
}

/**
 * Checks if the run should stop due to reaching the credits or time limit.
 * @returns What state the run should be in, and the reason for the state change if applicable
 */
export function shouldStopRun(params: ShouldStopParams): { statusChangeReason: RunStatusChangeReason | undefined; runStatus: RunStatus } {
    const {
        currentTimeElapsed,
        limits,
        maxCredits,
        maxSteps,
        maxTime,
        previousTimeElapsed,
        stepsRun,
        totalStepCost,
    } = params;

    if (totalStepCost > maxCredits) {
        return {
            statusChangeReason: RunStatusChangeReason.MaxCredits,
            runStatus: limits?.onMaxCredits === "Stop" ? RunStatus.Failed : RunStatus.Cancelled,
        };
    }
    if (previousTimeElapsed + currentTimeElapsed > maxTime) {
        return {
            statusChangeReason: RunStatusChangeReason.MaxTime,
            runStatus: limits?.onMaxTime === "Stop" ? RunStatus.Failed : RunStatus.Cancelled,
        };
    }
    if (stepsRun >= maxSteps) {
        return {
            statusChangeReason: RunStatusChangeReason.MaxSteps,
            runStatus: limits?.onMaxSteps === "Stop" ? RunStatus.Failed : RunStatus.Cancelled,
        };
    }
    return { statusChangeReason: undefined, runStatus: RunStatus.InProgress };
}
