import { API_CREDITS_MULTIPLIER, BotStyle, ConfigCallData, ConfigCallDataGenerate, DOLLARS_1_CENTS, FormElementBase, FormInputBase, ProjectVersion, RoutineType, RoutineVersion, RunContext, RunProject, RunRequestLimits, RunRoutine, VALYXA_ID, generateRoutineInitialValues, getTranslation, parseBotInformation, parseConfigCallData, parseSchemaInput, parseSchemaOutput } from "@local/shared";
import { Job } from "bull";
import { readOneHelper } from "../../actions/reads";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { emitSocketEvent } from "../../sockets/events";
import { getBotInfo } from "../../tasks/llm/context";
import { calculateMaxCredits, generateResponseWithFallback } from "../../tasks/llm/service";
import { runUserCode } from "../../tasks/sandbox";
import { type RecursivePartial, type SessionUserToken } from "../../types";
import { permissionsCheck } from "../../validators/permissions";
import { type RunProjectPayload, type RunRequestPayload, type RunRoutinePayload } from "./queue";

/** Limit routines to $1 for now */
export const DEFAULT_MAX_RUN_CREDITS = BigInt(DOLLARS_1_CENTS) * API_CREDITS_MULTIPLIER;

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
        const updatedUser = await prismaInstance.user.update({
            where: { id: userData.id },
            data: { premium: { update: { credits: { decrement: cost } } } },
            select: { premium: { select: { credits: true } } },
        });
        if (updatedUser.premium) {
            emitSocketEvent("apiCredits", userData.id, { credits: updatedUser.premium.credits + "" });
        }
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
    routineVersion: RecursivePartial<RoutineVersion>;
    run: RunRoutine;
    userData: SessionUserToken;
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

function parseRunIOFromPlaintext({ formData, text }: { formData: object; text: string }): { inputs: Record<string, string>; outputs: Record<string, string> } {
    const inputs: Record<string, string> = {};
    const outputs: Record<string, string> = {};
    const lines = text.trim().split("\n");

    for (const line of lines) {
        const [key, ...valueParts] = line.split(":");
        if (key && valueParts.length > 0) {
            const trimmedKey = key.trim();
            const value = valueParts.join(":").trim();

            if (Object.prototype.hasOwnProperty.call(formData, `input-${trimmedKey}`)) {
                inputs[trimmedKey] = value;
            } else if (Object.prototype.hasOwnProperty.call(formData, `output-${trimmedKey}`)) {
                outputs[trimmedKey] = value;
            }
            // If the key doesn't match any input or output, it's ignored
        }
    }

    return { inputs, outputs };
}

//TODO need way to track total credits/time spent, with ability to pause/cancel if limits are reached
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
    const baseTaskInfo = { runFrom, runId, runType, startedById, taskId } as const;
    // Total cost allowed for this routine. User credits should already have deducted the cost of previous steps.
    const maxCredits = calculateMaxCredits(
        userData.credits,
        limits?.maxCredits || DEFAULT_MAX_RUN_CREDITS,
        metrics?.creditsSpent,
    );
    // Total cost this step has incurred, not including previous steps
    let totalStepCost = BigInt(0);

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
        const routineVersion = await readOneHelper<RoutineVersion>({
            info: routineVersionSelect,
            input: { id: routineVersionId },
            objectType: "RoutineVersion",
            req,
        });
        // Parse configs and form data
        const configCallData = parseConfigCallData(routineVersion.configCallData, routineVersion.routineType, logger);
        const configFormInput = parseSchemaInput(routineVersion.configFormInput, routineVersion.routineType, logger);
        const configFormOutput = parseSchemaOutput(routineVersion.configFormOutput, routineVersion.routineType, logger);
        // Collect fieldName, description, and helpText for each input and output element
        const inputFieldInfo = configFormInput.elements.map(toFormFieldInfo).filter((element) => element.fieldName);
        const outputFieldInfo = configFormOutput.elements.map(toFormFieldInfo).filter((element) => element.fieldName);
        // Generate input/output value object like we do in the UI. Returns object where 
        // input keys are prefixed with "input-" and output keys are prefixed with "output-"
        let formData = generateRoutineInitialValues({
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
        // Combine values and field info so that we can easily generate a task message for an LLM.
        const fullInputs = formDataToIOInfo(formData, inputFieldInfo, "input-");
        const fullOutputs = formDataToIOInfo(formData, outputFieldInfo, "output-");
        console.log("got fullInputs", JSON.stringify(fullInputs));
        console.log("got fullOutputs", JSON.stringify(fullOutputs));
        // If there is at least one missing input, create a task message for the LLM to generate the missing inputs
        const taskMessage = generateTaskMessage(routineVersion as RoutineVersion, fullInputs, fullOutputs, userData.languages);
        console.log("got taskMessage", taskMessage);
        // If we have a taskMessage, this means we have missing inputs. 
        if (taskMessage) {
            // Use LLM to generate response
            let botToUse: string;
            if (routineVersion.routineType === RoutineType.Generate) {
                const configCallDataGenerate = configCallData as ConfigCallDataGenerate;
                botToUse = (configCallDataGenerate.botStyle === BotStyle.Specific ? configCallDataGenerate.respondingBot : startedById) || VALYXA_ID;
            } else {
                botToUse = startedById || VALYXA_ID;
            }
            const botInfo = await getBotInfo(botToUse);
            if (!botInfo) {
                throw new CustomError("0599", "InternalError", userData.languages, { configCallData });
            }
            const participantsData = { [botInfo.id]: botInfo };
            const respondingBotConfig = parseBotInformation(participantsData, botInfo.id, logger);
            if (!respondingBotConfig) {
                throw new CustomError("0619", "InternalError", userData.languages);
            }
            const { message, cost } = await generateResponseWithFallback({
                force: true,
                maxCredits,
                participantsData,
                respondingBotConfig,
                respondingBotId: botInfo.id,
                stream: false,
                task: undefined, // Don't provide task, since we're providing custom instructions
                taskMessage,
                userData,
            });
            totalStepCost += BigInt(cost);
            console.log("llm response", message, cost);
            // TODO transform response to input values
            console.time("doSandbox from run process");
            const { inputs: transformedInputs, outputs: transformedOutputs } = parseRunIOFromPlaintext({ formData, text: message });
            console.timeEnd("doSandbox from run process"); // Takes about 161ms. Will have to replace with hard-coded function, but it's good that we tested sandbox for the type-specific function
            console.log("transformed response", transformedInputs, transformedOutputs);
            //TODO Send update to UI and update formData
        }
        // // Determine if the routine being run is the overall routine or a sub-routine
        // const isSubroutine = runType !== "RunRoutine" || (run as RunRoutine).routineVersion?.id !== routineVersion.id;
        // What we do depends on the routine type
        const routineFunction = routineVersion.routineType ? routineTypeToFunction[routineVersion.routineType] : undefined;
        if (!routineFunction) {
            throw new CustomError("0593", "InternalError", userData.languages, { routineType: routineVersion.routineType });
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
            routineVersion,
            run: run as RunRoutine,
            userData,
        });
        totalStepCost += BigInt(cost);
        // TODO update run based on changed form (i.e. find new/updated inputs and outputs, add step)
        // Reduce user's credits
        const updatedUser = await prismaInstance.user.update({
            where: { id: userData.id },
            data: { premium: { update: { credits: { decrement: totalStepCost } } } },
            select: { premium: { select: { credits: true } } },
        });
        if (updatedUser.premium) {
            emitSocketEvent("apiCredits", userData.id, { credits: updatedUser.premium.credits + "" });
        }
        // Let the UI know that the task is done
        emitSocketEvent("runTask", runId, { status: "Completed", ...baseTaskInfo });
    } catch (error) {
        emitSocketEvent("runTask", runId, { status: "Failed", ...baseTaskInfo });
        logger.error("Caught error in doRun", { trace: "0587", error });
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
            throw new CustomError("0568", "InternalError", ["en"], { process: (data as { __process?: unknown }).__process });
    }
}
