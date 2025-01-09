import { API_CREDITS_MULTIPLIER, BotStyle, ConfigCallData, ConfigCallDataGenerate, DOLLARS_1_CENTS, FormElementBase, FormInputBase, MINUTES_5_MS, ProjectVersion, RoutineType, RoutineVersion, RunContext, RunProject, RunRequestLimits, RunRoutine, RunStatus, RunStatusChangeReason, TaskStatus, VALYXA_ID, generateRoutineInitialValues, getTranslation, parseBotInformation, parseConfigCallData, parseSchemaInput, parseSchemaOutput, saveRunProgress, shouldStopRun, type SessionUser } from "@local/shared";
import { Job } from "bull";
import { performance } from "perf_hooks";
import { readOneHelper } from "../../actions/reads";
import { InfoConverter } from "../../builders/infoConverter";
import { prismaInstance } from "../../db/instance";
import { parseRunIOFromPlaintext } from "../../db/seeds/codes";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { ModelMap } from "../../models/base";
import { emitSocketEvent } from "../../sockets/events";
import { getBotInfo } from "../../tasks/llm/context";
import { calculateMaxCredits, generateResponseWithFallback } from "../../tasks/llm/service";
import { runUserCode } from "../../tasks/sandbox";
import { type RecursivePartial } from "../../types";
import { reduceUserCredits } from "../../utils/reduceCredits";
import { permissionsCheck } from "../../validators/permissions";
import { type RunProjectPayload, type RunRequestPayload, type RunRoutinePayload } from "./queue";

/** Limit routines to $1 for now */
const DEFAULT_MAX_RUN_CREDITS = BigInt(DOLLARS_1_CENTS) * API_CREDITS_MULTIPLIER;
/** Limit routines to 5 minutes for now */
const DEFAULT_MAX_RUN_TIME = MINUTES_5_MS;

const runProjectSelect = {
    id: true,
    completedComplexity: true,
    contextSwitches: true,
    isPrivate: true,
    name: true,
    projectVersion: {
        select: {
            id: true,
            complexity: true,
            isDeleted: true,
            isPrivate: true,
            root: {
                select: {
                    id: true,
                    ownedByTeam: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                    ownedByUser: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                },
            },
            simplicity: true,
        },
    },
    status: true,
    steps: {
        select: {
            id: true,
            contextSwitches: true,
            name: true,
            order: true,
            step: true,
            timeElapsed: true,
        },
    },
    timeElapsed: true,
} as const;

const runRoutineSelect = {
    id: true,
    completedComplexity: true,
    contextSwitches: true,
    isPrivate: true,
    inputs: {
        select: {
            id: true,
            data: true,
            input: {
                select: {
                    id: true,
                    name: true,
                },
            },
        },
    },
    outputs: {
        select: {
            id: true,
            data: true,
            output: {
                select: {
                    id: true,
                    name: true,
                },
            },
        },
    },
    name: true,
    routineVersion: {
        select: {
            id: true,
            complexity: true,
            isDeleted: true,
            isPrivate: true,
            root: {
                select: {
                    id: true,
                    ownedByTeam: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                    ownedByUser: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                },
            },
            simplicity: true,
        },
    },
    status: true,
    steps: {
        select: {
            id: true,
            contextSwitches: true,
            name: true,
            nodeId: true,
            order: true,
            subroutineId: true,
            step: true,
            timeElapsed: true,
        },
    },
    timeElapsed: true,
} as const;

const projectVersionSelect = {
    id: true,
    complexity: true,
    isDeleted: true,
    isPrivate: true,
    root: {
        select: {
            id: true,
            ownedByTeam: {
                select: {
                    id: true,
                    isPrivate: true,
                },
            },
            ownedByUser: {
                select: {
                    id: true,
                    isPrivate: true,
                },
            },
        },
    },
    simplicity: true,
    translations: {
        select: {
            id: true,
            language: true,
            description: true,
            name: true,
        },
    },
} as const;

const routineVersionSelect = {
    id: true,
    apiVersion: {
        select: {
            callLink: true,
            schemaLanguage: true,
            schemaText: true,
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: {
                select: {
                    hasCompleteVersion: true,
                    id: true,
                    isDeleted: true,
                    isPrivate: true,
                    permissions: true,
                    ownedByTeam: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                    ownedByUser: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                },
            },
        },
    },
    codeVersion: {
        select: {
            codeLanguage: true,
            codeType: true,
            content: true,
            default: true,
            id: true,
            isDeleted: true,
            isPrivate: true,
            root: {
                select: {
                    hasCompleteVersion: true,
                    id: true,
                    isDeleted: true,
                    isPrivate: true,
                    permissions: true,
                    ownedByTeam: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                    ownedByUser: {
                        select: {
                            id: true,
                            isPrivate: true,
                        },
                    },
                },
            },
        },
    },
    configCallData: true,
    configFormInput: true,
    configFormOutput: true,
    complexity: true,
    inputs: {
        select: {
            id: true,
            index: true,
            isRequired: true,
            name: true,
            standardVersion: {
                select: {
                    id: true,
                    default: true,
                    isFile: true,
                    standardType: true,
                    props: true,
                },
            },
            translations: {
                select: {
                    id: true,
                    language: true,
                    description: true,
                    helpText: true,
                },
            },
        },
    },
    isDeleted: true,
    isPrivate: true,
    nodeLinks: {
        select: {
            id: true,
            from: {
                select: {
                    id: true,
                },
            },
            operation: true,
            to: {
                select: {
                    id: true,
                },
            },
            whens: {
                select: {
                    condition: true,
                    translations: {
                        select: {
                            id: true,
                            language: true,
                            description: true,
                            name: true,
                        },
                    },
                },
            },
        },
    },
    nodes: {
        select: {
            id: true,
            nodeType: true,
            runConditions: true,
            end: {
                select: {
                    wasSuccessful: true,
                },
            },
            loop: {
                select: {
                    loops: true,
                    maxLoops: true,
                    operation: true,
                    whiles: {
                        select: {
                            condition: true,
                            translations: {
                                select: {
                                    id: true,
                                    language: true,
                                    description: true,
                                    name: true,
                                },
                            },
                        },
                    },
                },
            },
            routineList: {
                select: {
                    isOrdered: true,
                    isOptional: true,
                    items: {
                        select: {
                            id: true,
                            index: true,
                            isOptional: true,
                            routineVersion: {
                                select: {
                                    id: true,
                                },
                            },
                        },
                    },
                },
            },
            translations: {
                select: {
                    id: true,
                    language: true,
                    description: true,
                    name: true,
                },
            },
        },
    },
    outputs: {
        select: {
            id: true,
            index: true,
            name: true,
            standardVersion: {
                select: {
                    id: true,
                    default: true,
                    isFile: true,
                    standardType: true,
                    props: true,
                },
            },
            translations: {
                select: {
                    id: true,
                    language: true,
                    description: true,
                    helpText: true,
                },
            },
        },
    },
    root: {
        select: {
            id: true,
            ownedByTeam: {
                select: {
                    id: true,
                    isPrivate: true,
                },
            },
            ownedByUser: {
                select: {
                    id: true,
                    isPrivate: true,
                },
            },
        },
    },
    routineType: true,
    simplicity: true,
    translations: {
        select: {
            id: true,
            language: true,
            description: true,
            instructions: true,
            name: true,
        },
    },
} as const;

export async function doRunProject({
    projectVersionId,
    runFrom,
    runId,
    runType,
    startedById,
    taskId,
    userData,
}: RunProjectPayload) {
    const baseTaskInfo = { runFrom, runId, runType, startedById, taskId } as const;
    try {
        // Let the UI know that the task is Running
        emitSocketEvent("runTask", runId, { status: "Running", ...baseTaskInfo });
        // Get the routine and run, in a way that throws an error if they don't exist or the user doesn't have permission
        const req = { session: { languages: userData.languages, users: [userData] } };
        const run = await readOneHelper<RunProject | RunRoutine>({
            info: runType === "RunProject" ? runProjectSelect : runRoutineSelect,
            input: { id: runId },
            objectType: runType,
            req,
        });
        const projectVersion = await readOneHelper<ProjectVersion>({
            info: projectVersionSelect,
            input: { id: projectVersionId },
            objectType: "RoutineVersion",
            req,
        });
        //TODO generate response and get cost
        const cost = 0;
        // Reduce user's credits
        await reduceUserCredits(userData.id, cost);
        //TODO perform routine
    } catch (error) {
        emitSocketEvent("runTask", runId, { status: "Failed", ...baseTaskInfo });
        logger.error("Caught error in doRun", { trace: "0587", error });
        return { __typename: "Success" as const, success: false };
    }
}

type FormFieldInfo = Pick<FormElementBase, "description" | "helpText"> & Pick<FormInputBase, "fieldName" | "isRequired">;

type DoRoutineTypeProps = {
    configCallData: ConfigCallData;
    formData: object;
    handleFormDataUpdate: (run: object) => unknown;
    inputData: FormFieldInfo[];
    limits: RunRequestLimits;
    outputData: FormFieldInfo[];
    remainingCredits: bigint;
    routineVersion: RecursivePartial<RoutineVersion>;
    run: RunRoutine;
    userData: SessionUser;
}

type DoRoutineTypeResult = {
    cost: number;
}

async function doRunRoutineAction({

}: DoRoutineTypeProps): Promise<DoRoutineTypeResult> {
    return { cost: 0 };
}

async function doRunRoutineApi({
    formData,
    handleFormDataUpdate,
    routineVersion,
    userData,
}: DoRoutineTypeProps): Promise<DoRoutineTypeResult> {
    const cost = 0;
    const codeVersion = routineVersion.codeVersion;

    // Make sure that the code is valid and runnable
    if (!codeVersion) {
        logger.error("No code version found", { trace: "0636", routineVersionId: routineVersion.id });
        return { cost };
    }
    if (!SUPPORTED_CODE_LANGUAGES.includes(codeVersion.codeLanguage.toLowerCase())) {
        logger.error("Unsupported code language", { trace: "0627", codeLanguage: codeVersion.codeLanguage, codeVersionId: codeVersion.id });
        return { cost };
    }
    const hasPermissions = await permissionsCheck(
        { [codeVersion.id]: { ...codeVersion, __typename: "CodeVersion" } },
        { ["Read"]: [codeVersion.id] },
        {},
        userData,
        true,
    );
    if (!hasPermissions) {
        logger.error("User does not have permissions to run code", { trace: "0628", codeVersionId: codeVersion.id, userId: userData.id });
        return { cost };
    }

    // Grab inputs from formData
    const inputs = Object.entries(formData).reduce((acc, [key, value]) => {
        if (key.startsWith("input-")) {
            acc[key.replace("input-", "")] = value;
        }
        return acc;
    });

    // TODO call API

    return { cost };
}

const SUPPORTED_CODE_LANGUAGES = ["javascript"];

async function doRunRoutineCode({
    formData,
    handleFormDataUpdate,
    routineVersion,
    userData,
}: DoRoutineTypeProps): Promise<DoRoutineTypeResult> {
    const cost = 0;
    const codeVersion = routineVersion.codeVersion;

    // Make sure that the code is valid and runnable
    if (!codeVersion) {
        logger.error("No code version found", { trace: "0626", routineVersionId: routineVersion.id });
        return { cost };
    }
    if (!SUPPORTED_CODE_LANGUAGES.includes(codeVersion.codeLanguage.toLowerCase())) {
        logger.error("Unsupported code language", { trace: "0627", codeLanguage: codeVersion.codeLanguage, codeVersionId: codeVersion.id });
        return { cost };
    }
    const hasPermissions = await permissionsCheck(
        { [codeVersion.id]: { ...codeVersion, __typename: "CodeVersion" } },
        { ["Read"]: [codeVersion.id] },
        {},
        userData,
        true,
    );
    if (!hasPermissions) {
        logger.error("User does not have permissions to run code", { trace: "0628", codeVersionId: codeVersion.id, userId: userData.id });
        return { cost };
    }

    // Grab inputs from formData
    const inputs = Object.entries(formData).reduce((acc, [key, value]) => {
        if (key.startsWith("input-")) {
            acc[key.replace("input-", "")] = value;
        }
        return acc;
    });

    // Run the code
    const { output: transformedResponse } = await runUserCode({
        content: codeVersion.content,
        input: inputs,
    });

    // Update form data with the transformed response


    return { cost };
}

async function doRunRoutineData({

}: DoRoutineTypeProps): Promise<DoRoutineTypeResult> {
    return { cost: 0 };
}

async function doRunRoutineGenerate(): Promise<DoRoutineTypeResult> {
    // We don't need to do anything here, since we're already generating an LLM response
    // in the main `doRunRoutine` function. It had to be put there so that we can fill in 
    // missing required inputs before running type-specific logic. Since we were already calling 
    // an LLM for inputs, it made sense to call it for the response at the same time.
    return { cost: 0 };
}

async function doRunRoutineInformational(): Promise<DoRoutineTypeResult> {
    // This is just informational, so we don't need to do anything. 
    // Updating the run context happens in the main `doRunRoutine` function.
    return { cost: 0 };
}

async function doRunRoutineMultiStep({

}: DoRoutineTypeProps): Promise<DoRoutineTypeResult> {
    return { cost: 0 };
}

async function doRunRoutineSmartContract({

}: DoRoutineTypeProps): Promise<DoRoutineTypeResult> {
    return { cost: 0 };
}

const routineTypeToFunction = {
    [RoutineType.Action]: doRunRoutineAction,
    [RoutineType.Api]: doRunRoutineApi,
    [RoutineType.Code]: doRunRoutineCode,
    [RoutineType.Data]: doRunRoutineData,
    [RoutineType.Generate]: doRunRoutineGenerate,
    [RoutineType.Informational]: doRunRoutineInformational,
    [RoutineType.MultiStep]: doRunRoutineMultiStep,
    [RoutineType.SmartContract]: doRunRoutineSmartContract,
};

/**
 * Updates overall run context, which can be added to LLM context to generate better responses. 
 * Information included in context:
 * - Overall routine/project information
 * - Ordered stack of previous routine/project information, with inputs/outputs used
 * - Previous inputs/outputs
 * @param context The current context, which has been created from previous steps, or undefined if this is the first step
 * @param runnableObject The project version or routine version for the current step
 * @param fullInputs The inputs for the current step. Both the values and field info are included
 * @param fullOutputs The outputs for the current step. Both the values and field info are included
 * @param contextSize The maximum length the context can be. If the context is longer, older context will be removed
 * @param languages The user's preferred languages
 * @returns The updated context
 */
function updateRunContext(
    context: RunContext | undefined,
    runnableObject: ProjectVersion | RoutineVersion,
    fullInputs: Record<string, { value: unknown }>,
    fullOutputs: Record<string, { value: unknown }>,
    contextSize: number, //TODO should pass in Math.min(limits.maxContextSize, llmService.getContextSize(model))
    languages: string[],
): RunContext {
    const newContext: RunContext = context ?? {
        currentTask: {
            name: "",
            description: "",
        },
        allInputs: {},
        allOutputs: {},
    };

    // Update current task
    const bestTranslation = getTranslation(runnableObject as RoutineVersion, languages, true);
    newContext.currentTask = {
        name: bestTranslation.name || "Unnamed Task",
        description: bestTranslation.description || "No description",
        instructions: bestTranslation.instructions || undefined,
    };

    // Update overall task if the context existed beforeand didn't have an overall task
    if (typeof context === "object" && !Object.prototype.hasOwnProperty.call(newContext, "overallTask")) {
        newContext.overallTask = { ...context.currentTask };
    }

    // Update inputs and outputs
    for (const [key, { value }] of Object.entries(fullInputs)) {
        newContext.allInputs[key] = value;
    }
    for (const [key, { value }] of Object.entries(fullOutputs)) {
        newContext.allOutputs[key] = value;
    }

    // Add current task to previous tasks
    if (!newContext.previousTasks) {
        newContext.previousTasks = [];
    }
    newContext.previousTasks.push(newContext.currentTask);

    // Ensure context size is within limit TODO should be using token sizes instead of stringified length, and should account for other data in context like bot info and task message
    let contextString = JSON.stringify(newContext);
    while (contextString.length > contextSize && newContext.previousTasks.length > 0) {
        newContext.previousTasks.shift();
        contextString = JSON.stringify(newContext);
    }

    // If context is still too large, start removing other elements
    if (contextString.length > contextSize) {
        delete newContext.previousTasks;
        delete newContext.overallTask;
        contextString = JSON.stringify(newContext);

        if (contextString.length > contextSize) {
            // If it's still too large, only keep the current task and minimal info
            return {
                currentTask: newContext.currentTask,
                allInputs: {},
                allOutputs: {},
            };
        }
    }

    return newContext;
}

function toFormFieldInfo(element: FormElementBase): FormFieldInfo {
    return {
        description: element.description,
        fieldName: (element as FormInputBase).fieldName,
        helpText: element.helpText,
        isRequired: (element as FormInputBase).isRequired ?? false,
    };
}

function formDataToIOInfo(
    formData: object,
    ioFieldInfo: FormFieldInfo[],
    prefix: "input-" | "output-",
): Record<string, { value: unknown }> {
    return Object.keys(formData)
        .filter(key => key.startsWith(prefix))
        .reduce((acc, key) => {
            const withoutPrefix = key.replace(prefix, "");
            const fieldInfo = ioFieldInfo.find((info) => info.fieldName === withoutPrefix);
            const isValueUnset = formData[key] === "" || formData[key] === undefined;
            acc[withoutPrefix] = {
                ...fieldInfo,
                value: isValueUnset ? undefined : formData[key],
            };
            return acc;
        }, {} as Record<string, { value: unknown }>);
}

/**
 * Retrieves and formats the routine version information for LLM tasks.
 * @param routineVersion - The version of the routine being executed.
 * @param userLanguages - An array of language codes for the user's preferred languages.
 * @returns A string containing the formatted routine version information.
 */
function getRoutineVersionInfo(routineVersion: Pick<RoutineVersion, "translations">, userLanguages: string[]): string {
    let info = "";
    const { description, instructions, name } = getTranslation(routineVersion, userLanguages, true);
    if (name && name.length) {
        info += `Name: ${name}\n`;
    }
    if (description && description.length) {
        info += `Description: ${description}\n`;
    }
    if (instructions && instructions.length) {
        info += `Instructions: ${instructions}\n\n`;
    }
    return info;
}

/**
 * Generates a task message based on the routine type, inputs, and outputs.
 * @param routineVersion - The version of the routine being executed.
 * @param fullInputs - An object containing input fields, their values, and whether they're required.
 * @param fullOutputs - An object containing output fields and their values.
 * @param userLanguages - An array of language codes for the user's preferred languages.
 * @returns A string containing the task message, or undefined if no message is needed.
 */
export function generateTaskMessage(
    routineVersion: RoutineVersion,
    fullInputs: Record<string, { value: unknown; isRequired?: boolean }>,
    fullOutputs: Record<string, { value: unknown }>,
    userLanguages: string[],
): string | undefined {
    const hasMissingInputs = Object.values(fullInputs).some(
        (input) => input.isRequired && (input.value === "" || input.value === undefined),
    );
    const hasOutputs = Object.keys(fullOutputs).length > 0;

    // Only the Generate routine type can use an LLM to create its outputs, so all 
    // other types either return a message to fill in missing inputs or undefined
    // if there are no missing inputs
    if (routineVersion.routineType !== RoutineType.Generate) {
        return hasMissingInputs ? generateInputOnlyMessage(routineVersion, fullInputs, userLanguages) : undefined;
    }

    // For Generate routine type, don't return a message if there are no missing inputs and no outputs
    if (!hasMissingInputs && !hasOutputs) {
        return undefined;
    }

    if (hasMissingInputs && hasOutputs) {
        return generateInputAndOutputMessage(routineVersion, fullInputs, fullOutputs, userLanguages);
    } else if (hasMissingInputs) {
        return generateInputOnlyMessage(routineVersion, fullInputs, userLanguages);
    } else if (hasOutputs) {
        return generateOutputOnlyMessage(routineVersion, fullOutputs, userLanguages);
    }

    return undefined;
}

/**
 * Generates a message for tasks that only require input generation.
 * @param routineVersion - The version of the routine being executed.
 * @param fullInputs - An object containing input fields, their values, and whether they're required.
 * @param userLanguages - An array of language codes for the user's preferred languages.
 * @returns A string containing the task message for input generation.
 */
export function generateInputOnlyMessage(
    routineVersion: RoutineVersion,
    fullInputs: Record<string, { value: unknown; isRequired?: boolean }>,
    userLanguages: string[],
): string {
    let message = "I am providing you with information about a task we're working on. It includes inputs, some of which need to be filled out for us to continue with the task.\n\n";
    message += "Your goal is to fill in the missing values for the list of inputs. Respond with each input value in the format `fieldName: value`. Each input should be on a new line, with nothing else in the response.\n\n";
    message += "For example:\n\n```\nfieldA: valueA\nfieldB: valueB\n```\n\n";
    message += "Here is some information about the task, along with the inputs:\n\n";

    message += getRoutineVersionInfo(routineVersion, userLanguages);
    message += `Inputs: ${JSON.stringify(fullInputs, null, 2)}`;

    return message;
}

/**
 * Generates a message for tasks that only require output generation.
 * @param routineVersion - The version of the routine being executed.
 * @param fullOutputs - An object containing output fields and their values.
 * @param userLanguages - An array of language codes for the user's preferred languages.
 * @returns A string containing the task message for output generation.
 */
export function generateOutputOnlyMessage(
    routineVersion: RoutineVersion,
    fullOutputs: Record<string, { value: unknown }>,
    userLanguages: string[],
): string {
    let message = "I am providing you with information about a task we're working on. We need to generate outputs for this task.\n\n";
    message += "Your goal is to generate values for each output. Respond with each output value in the format `fieldName: value`. Each output should be on a new line, with nothing else in the response.\n\n";
    message += "For example:\n\n```\noutputA: valueA\noutputB: valueB\n```\n\n";
    message += "Here is some information about the task, along with the expected outputs:\n\n";

    message += getRoutineVersionInfo(routineVersion, userLanguages);
    message += `Outputs: ${JSON.stringify(fullOutputs, null, 2)}`;

    return message;
}

/**
 * Generates a message for tasks that require both input and output generation.
 * @param routineVersion - The version of the routine being executed.
 * @param fullInputs - An object containing input fields, their values, and whether they're required.
 * @param fullOutputs - An object containing output fields and their values.
 * @param userLanguages - An array of language codes for the user's preferred languages.
 * @returns A string containing the task message for both input and output generation.
 */
export function generateInputAndOutputMessage(
    routineVersion: RoutineVersion,
    fullInputs: Record<string, { value: unknown; isRequired?: boolean }>,
    fullOutputs: Record<string, { value: unknown }>,
    userLanguages: string[],
): string {
    let message = "I am providing you with information about a task we're working on. It includes inputs that need to be filled out and outputs that need to be generated.\n\n";
    message += "Your goal is to fill in the missing values for the inputs and generate values for the outputs. Respond with each input and output value in the format `fieldName: value`. Each field should be on a new line, with nothing else in the response.\n\n";
    message += "For example:\n\n```\ninputA: valueA\ninputB: valueB\noutputX: valueX\noutputY: valueY\n```\n\n";
    message += "Here is some information about the task, along with the inputs and outputs:\n\n";

    message += getRoutineVersionInfo(routineVersion, userLanguages);
    message += `Inputs: ${JSON.stringify(fullInputs, null, 2)}\n\n`;
    message += `Outputs: ${JSON.stringify(fullOutputs, null, 2)}`;

    return message;
}

const runStatusToTaskStatus: Record<RunStatus, TaskStatus> = {
    [RunStatus.Cancelled]: TaskStatus.Suggested,
    [RunStatus.Completed]: TaskStatus.Completed,
    [RunStatus.Failed]: TaskStatus.Failed,
    [RunStatus.InProgress]: TaskStatus.Running,
    [RunStatus.Scheduled]: TaskStatus.Scheduled,
};

//TODO need to persist time spent and steps to next step
export async function doRunRoutine({
    context,
    formValues,
    limits,
    metrics,
    routineVersionId,
    runFrom,
    runId,
    runType,
    startedById,
    taskId,
    userData,
}: RunRoutinePayload) {
    // Collect info for tracking limits and costs
    // Total cost allowed for this routine. User credits should already have deducted the cost of previous steps.
    const maxCredits = calculateMaxCredits(
        userData.credits,
        limits?.maxCredits || DEFAULT_MAX_RUN_CREDITS,
        metrics?.creditsSpent,
    );
    // Total cost this step has incurred, not including previous steps
    let totalStepCost = BigInt(0);
    // Timing variables
    const startTime = performance.now();
    const maxTime = limits?.maxTime || DEFAULT_MAX_RUN_TIME;
    const previousTimeElapsed = metrics?.timeElapsed || 0;
    // Steps completed so far
    const stepsRun = metrics?.stepsRun || 0;
    const maxSteps = limits?.maxSteps || Number.MAX_SAFE_INTEGER; // By default, runs should only be limited by credits and time

    // Other info needed up save run progress
    let run: RunProject | RunRoutine | undefined = undefined;
    let runnableObject: RoutineVersion | undefined = undefined;
    let formData: object = {};
    let statusChangeReason: RunStatusChangeReason | undefined = undefined;
    let runStatus: RunStatus = RunStatus.InProgress;

    /**
     * Updates the run state and handles all actions performed when a run is updated
     * NOTE: Only call this function after run, runnableObject, and formData information has been set.
     * @returns True if update was successful, false if it failed
     */
    async function applyRunUpdate(): Promise<boolean> {
        if (!run || !runnableObject) {
            return false;
        }

        try {
            //TODO also need way to create run if it doesn't exist
            // TODO this currently requires step data, which makes it only work for multi-step routines. Change this
            await saveRunProgress({
                contextSwitches: run.contextSwitches || 0, // N/A for automated runs
                currentStep: null, // TODO will change for multi-step support
                currentStepOrder: 1,// TODO will change for multi-step support
                currentStepRunData: null,// TODO will change for multi-step support
                formData,
                handleRunProjectUpdate: async function updateRun(apiInput) {
                    // Update the run with the new data
                    const { format, mutate } = ModelMap.getLogic(["format", "mutate"], "RunProject", true, "run process update project cast");
                    const input = mutate.yup.update && mutate.yup.update({ env: process.env.NODE_ENV }).cast(apiInput, { stripUnknown: true });
                    const data = mutate.shape.update ? await mutate.shape.update({ data: input, idsCreateToConnect: {}, preMap: {}, userData }) : input;
                    const updateResult = await prismaInstance.run_project.update({
                        where: { id: run?.id || apiInput.id },
                        data,
                        select: runProjectSelect,
                    });
                    const partialInfo = InfoConverter.fromApiToPartialApi(runProjectSelect, format.apiRelMap, true);
                    const converted = InfoConverter.fromDbToApi(updateResult, partialInfo) as RunProject;
                    run = converted;
                },
                handleRunRoutineUpdate: async function updateRun(apiInput) {
                    // Update the run with the new data
                    const { format, mutate } = ModelMap.getLogic(["format", "mutate"], "RunRoutine", true, "run process update routine cast");
                    const input = mutate.yup.update && mutate.yup.update({ env: process.env.NODE_ENV }).cast(apiInput, { stripUnknown: true });
                    const data = mutate.shape.update ? await mutate.shape.update({ data: input, idsCreateToConnect: {}, preMap: {}, userData }) : input;
                    const updateResult = await prismaInstance.run_routine.update({
                        where: { id: run?.id || apiInput.id },
                        data,
                        select: runRoutineSelect,
                    });
                    const partialInfo = InfoConverter.fromApiToPartialApi(runRoutineSelect, format.apiRelMap, true);
                    const converted = InfoConverter.fromDbToApi(updateResult, partialInfo) as RunRoutine;
                    run = converted;
                },
                isStepCompleted: false,// TODO will change for multi-step support
                isRunCompleted: statusChangeReason === RunStatusChangeReason.Completed,
                logger,
                run,
                runnableObject,
                timeElapsed: previousTimeElapsed + (performance.now() - startTime),
            });

            // Emit socket event to update the UI
            const status = runStatusToTaskStatus[runStatus];
            const taskSocketInfo = { run, runFrom, runId, runType, startedById, statusChangeReason, status, taskId } as const;
            emitSocketEvent("runTask", runId, taskSocketInfo);

            // Reduce user's credits
            await reduceUserCredits(userData.id, totalStepCost);

            return true;
        } catch (error) {
            console.error(`Error managing run state for ${runId}:`, error);
            return false;
        }
    }
    /**
     * Checks if the run should stop, and if so, manages the run state.
     * 
     * NOTE: Only call this function after run, runnableObject, and formData information has been set.
     * @returns True if `doRunRoutine` should return, false if it should continue
     */
    async function checkStop(): Promise<boolean> {
        const runState = shouldStopRun({
            currentTimeElapsed: performance.now() - startTime,
            limits,
            maxCredits,
            maxSteps,
            maxTime,
            previousTimeElapsed,
            stepsRun,
            totalStepCost,
        });
        statusChangeReason = runState.statusChangeReason;
        runStatus = runState.runStatus;

        if (statusChangeReason) {
            const success = await applyRunUpdate();
            return !success;
        }
        return false;
    }

    try {
        // Let the UI know that the task is Running
        emitSocketEvent("runTask", runId, { status: "Running", runFrom, runId, runType, startedById, taskId });
        // Get the routine and run, in a way that throws an error if they don't exist or the user doesn't have permission
        const req = { session: { languages: userData.languages, users: [userData] } };
        run = await readOneHelper<RunProject | RunRoutine>({
            info: runType === "RunProject" ? runProjectSelect : runRoutineSelect,
            input: { id: runId },
            objectType: runType,
            req,
        }) as RunProject | RunRoutine;
        runnableObject = await readOneHelper<RoutineVersion>({
            info: routineVersionSelect,
            input: { id: routineVersionId },
            objectType: "RoutineVersion",
            req,
        }) as RoutineVersion;
        // Parse configs and form data
        const configCallData = parseConfigCallData(runnableObject.configCallData, runnableObject.routineType, logger);
        const configFormInput = parseSchemaInput(runnableObject.configFormInput, runnableObject.routineType, logger);
        const configFormOutput = parseSchemaOutput(runnableObject.configFormOutput, runnableObject.routineType, logger);
        // Collect fieldName, description, and helpText for each input and output element
        const inputFieldInfo = configFormInput.elements.map(toFormFieldInfo).filter((element) => element.fieldName);
        const outputFieldInfo = configFormOutput.elements.map(toFormFieldInfo).filter((element) => element.fieldName);
        // Generate input/output value object like we do in the UI. Returns object where 
        // input keys are prefixed with "input-" and output keys are prefixed with "output-"
        formData = generateRoutineInitialValues({
            configFormInput,
            configFormOutput,
            logger,
            runInputs: (run as RunRoutine)?.inputs,
            runOutputs: (run as RunRoutine)?.outputs,
        });
        // Override generated inputs with provided inputs
        if (typeof formValues === "object" && !Array.isArray(formValues)) {
            for (const [key, value] of Object.entries(formValues)) {
                if (formData[key]) {
                    formData[key] = value;
                }
            }
        }
        if (await checkStop()) {
            return { __typename: "Success" as const, success: false };
        }
        // Combine values and field info so that we can easily generate a task message for an LLM.
        const fullInputs = formDataToIOInfo(formData, inputFieldInfo, "input-");
        const fullOutputs = formDataToIOInfo(formData, outputFieldInfo, "output-");
        // If there is at least one missing input, create a task message for the LLM to generate the missing inputs
        const taskMessage = generateTaskMessage(runnableObject, fullInputs, fullOutputs, userData.languages);
        // If we have a taskMessage, this means we have missing inputs. 
        if (taskMessage) {
            // Use LLM to generate response
            let botToUse: string;
            if (runnableObject.routineType === RoutineType.Generate) {
                const configCallDataGenerate = configCallData as ConfigCallDataGenerate;
                botToUse = (configCallDataGenerate.botStyle === BotStyle.Specific ? configCallDataGenerate.respondingBot : startedById) || VALYXA_ID;
            } else {
                botToUse = startedById || VALYXA_ID;
            }
            const botInfo = await getBotInfo(botToUse);
            if (!botInfo) {
                throw new CustomError("0599", "InternalError", { configCallData });
            }
            const participantsData = { [botInfo.id]: botInfo };
            const respondingBotConfig = parseBotInformation(participantsData, botInfo.id, logger);
            if (!respondingBotConfig) {
                throw new CustomError("0619", "InternalError");
            }
            const { message, cost } = await generateResponseWithFallback({
                force: true,
                maxCredits,
                mode: "json",
                participantsData,
                respondingBotConfig,
                respondingBotId: botInfo.id,
                stream: false,
                task: undefined, // Don't provide task, since we're providing custom instructions
                taskMessage,
                userData,
            });
            totalStepCost += BigInt(cost);
            // Transform response to input and output values
            const { inputs: transformedInputs, outputs: transformedOutputs } = parseRunIOFromPlaintext({ formData, text: message });
            // Update formData
            formData = {
                ...formData,
                // Prepend "input-" to keys for inputs and "output-" to keys for outputs
                ...Object.entries(transformedInputs).reduce((acc, [key, value]) => {
                    acc[`input-${key}`] = value;
                    return acc;
                }, {} as Record<string, any>),
                ...Object.entries(transformedOutputs).reduce((acc, [key, value]) => {
                    acc[`output-${key}`] = value;
                    return acc;
                }, {} as Record<string, any>),
            };
        }
        // // Determine if the routine being run is the overall routine or a sub-routine
        // const isSubroutine = runType !== "RunRoutine" || (run as RunRoutine).routineVersion?.id !== routineVersion.id;
        // What we do depends on the routine type
        const routineFunction = runnableObject.routineType ? routineTypeToFunction[runnableObject.routineType] : undefined;
        if (!routineFunction) {
            throw new CustomError("0593", "InternalError", { routineType: runnableObject.routineType });
        }
        const { cost } = await routineFunction({
            configCallData,
            formData,
            handleFormDataUpdate: (updatedForm: object) => {
                formData = updatedForm;
            },
            inputData: inputFieldInfo,
            limits: limits ?? {},
            outputData: outputFieldInfo,
            remainingCredits: maxCredits - totalStepCost,
            routineVersion: runnableObject,
            run: run as RunRoutine,
            userData,
        });
        totalStepCost += BigInt(cost);

        runStatus = RunStatus.Completed;
        statusChangeReason = RunStatusChangeReason.Completed;
        await applyRunUpdate();
    } catch (error) {
        logger.error("Caught error in doRun", { trace: "0587", error });
        runStatus = RunStatus.Failed;
        statusChangeReason = RunStatusChangeReason.Error;
        await applyRunUpdate();
        return { __typename: "Success" as const, success: false };
    }
}

export async function runProcess({ data }: Job<RunRequestPayload>) {
    switch (data.__process) {
        case "Project":
            return doRunProject(data);
        case "Routine":
            return doRunRoutine(data);
        case "Test":
            logger.info("runProcess test triggered");
            return { __typename: "Success" as const, success: true };
        default:
            throw new CustomError("0568", "InternalError", { process: (data as { __process?: unknown }).__process });
    }
}
