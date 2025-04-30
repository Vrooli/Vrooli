import { API_CREDITS_MULTIPLIER, AutoPickFirstSelectionHandler, AutoPickRandomSelectionHandler, BotSettings, BotSettingsConfig, BranchProgress, CodeLanguage, CodeVersionConfig, ConfigCallDataGenerate, DecisionOption, DeferredDecisionData, FormElementBase, FormInputBase, HOURS_2_MS, InputGenerationStrategy, LlmTask, Location, LocationData, MINUTES_1_MS, PathSelectionHandler, PathSelectionStrategy, ResolvedDecisionDataChooseMultiple, ResolvedDecisionDataChooseOne, ResourceSubType, ResourceVersion, RoutineVersionConfig, Run, RunBotConfig, RunConfig, RunCreateInput, RunIdentifier, RunLoader, RunNotifier, RunPersistence, RunStateMachine, RunStatus, RunStatusChangeReason, RunSubroutineResult, RunTaskInfo, RunUpdateInput, SECONDS_10_MS, SEEDED_IDS, SubroutineExecutionStrategy, SubroutineExecutor, SubroutineIOMapping, SubroutineInputDisplayInfo, SubroutineOutputDisplayInfo, Success, TaskStatus, getTranslation, navigatorFactory, uppercaseFirstLetter, type SessionUser } from "@local/shared";
import { Job } from "bull";
import { createOneHelper } from "../../actions/creates.js";
import { readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { Notify } from "../../notify/notify.js";
import { emitSocketEvent, roomHasOpenConnections } from "../../sockets/events.js";
import { LlmTaskData, generateTaskExec } from "../../tasks/llm/converter.js";
import { LlmServiceRegistry } from "../../tasks/llm/registry.js";
import { generateResponseWithFallback } from "../../tasks/llm/service.js";
import { runUserCode } from "../../tasks/sandbox/process.js";
import { RunUserCodeInput } from "../../tasks/sandbox/types.js";
import { type SessionData } from "../../types.js";
import { reduceUserCredits } from "../../utils/reduceCredits.js";
import { permissionsCheck } from "../../validators/permissions.js";
import { type RunRequestPayload, type RunRoutinePayload } from "./queue.js";

/**
 * How long to wait before giving up on a run and marking it as failed.
 * 
 * NOTE 1: This should be a long time, so that runs can execute for a long time in low-load conditions.
 */
export const RUN_TIMEOUT_MS = HOURS_2_MS;
/**
 * How long to wait after a run has been paused to give it a chance to finish gracefully.
 */
export const RUN_SHUTDOWN_GRACE_PERIOD_MS = SECONDS_10_MS;

/**
 * The fields to select for various run-related objects
 */
export const RunProcessSelect = {
    Run: {
        id: true,
        completedComplexity: true,
        contextSwitches: true,
        isPrivate: true,
        io: {
            select: {
                id: true,
                data: true,
                nodeInputName: true,
                nodeName: true,
            },
        },
        name: true,
        resourceVersion: {
            select: {
                id: true,
                complexity: true,
                isDeleted: true,
                isPrivate: true,
                resourceSubType: true,
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
                complexity: true,
                contextSwitches: true,
                name: true,
                nodeId: true,
                order: true,
                resourceInId: true,
                resourceVersionId: true,
                startedAt: true,
                status: true,
                timeElapsed: true,
            },
        },
        timeElapsed: true,
    },
    ResourceVersion: {
        id: true,
        codeLanguage: true,
        config: true,
        complexity: true,
        isAutomatable: true,
        isDeleted: true,
        isPrivate: true,
        relatedVersions: {
            select: {
                id: true,
                labels: true,
                toVersion: {
                    select: {
                        id: true,
                        codeLanguage: true,
                        config: true,
                    },
                },
            },
        },
        resourceSubType: true,
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
                resourceType: true,
            },
        },
        simplicity: true,
        translations: {
            select: {
                id: true,
                language: true,
                description: true,
                details: true,
                instructions: true,
                name: true,
            },
        },
    },
} as const;

/**
 * Builds the minimum Request object needed to perform CRUD operations for a RunProject or RunRoutine.
 * 
 * @param userData The user's session data
 * @returns The partial Request object
 */
function buildReq(userData: SessionUser): { session: Pick<SessionData, "languages" | "users"> } {
    return {
        session: {
            languages: userData.languages,
            users: [userData],
        },
    };
}

type FormFieldInfo = Pick<FormElementBase, "description" | "helpText"> & Pick<FormInputBase, "fieldName" | "isRequired">;

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

const runStatusToTaskStatus: Record<RunStatus, TaskStatus> = {
    [RunStatus.Cancelled]: TaskStatus.Suggested,
    [RunStatus.Completed]: TaskStatus.Completed,
    [RunStatus.Failed]: TaskStatus.Failed,
    [RunStatus.InProgress]: TaskStatus.Running,
    [RunStatus.Paused]: TaskStatus.Paused,
    [RunStatus.Scheduled]: TaskStatus.Scheduled,
};

//TODO need to persist time spent and steps to next step
export async function doRunRoutine(data: RunRoutinePayload) {
    const { resourceVersionId, startedById, runFrom, runId, taskId, userData } = data;
    // Collect info for tracking limits and costs
    // Total cost allowed for this routine. User credits should already have deducted the cost of previous steps.
    // const maxCredits = calculateMaxCredits(
    //     userData.credits,
    //     limits?.maxCredits || DEFAULT_MAX_RUN_CREDITS,
    //     metrics?.creditsSpent,
    // );
    // // Total cost this step has incurred, not including previous steps
    const totalStepCost = BigInt(0);
    // // Timing variables
    // const startTime = performance.now();
    // const maxTime = limits?.maxTime || DEFAULT_MAX_RUN_TIME;
    // const previousTimeElapsed = metrics?.timeElapsed || 0;
    // // Steps completed so far
    // const stepsRun = metrics?.stepsRun || 0;
    // const maxSteps = limits?.maxSteps || Number.MAX_SAFE_INTEGER; // By default, runs should only be limited by credits and time

    // Other info needed up save run progress
    let run: Run | undefined = undefined;
    let runnableObject: ResourceVersion | undefined = undefined;
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
            // await saveRunProgress({
            //     contextSwitches: run.contextSwitches || 0, // N/A for automated runs
            //     currentStep: null, // TODO will change for multi-step support
            //     currentStepOrder: 1,// TODO will change for multi-step support
            //     currentStepRunData: null,// TODO will change for multi-step support
            //     formData,
            //     handleRunProjectUpdate: async function updateRun(apiInput) {
            //         // Update the run with the new data
            //         const { format, mutate } = ModelMap.getLogic(["format", "mutate"], "RunProject", true, "run process update project cast");
            //         const input = mutate.yup.update && mutate.yup.update({ env: process.env.NODE_ENV }).cast(apiInput, { stripUnknown: true });
            //         const data = mutate.shape.update ? await mutate.shape.update({ data: input, idsCreateToConnect: {}, preMap: {}, userData }) : input;
            //         const updateResult = await DbProvider.get().run_project.update({
            //             where: { id: run?.id || apiInput.id },
            //             data,
            //             select: runProjectSelect,
            //         });
            //         const partialInfo = InfoConverter.get().fromApiToPartialApi(runProjectSelect, format.apiRelMap, true);
            //         const converted = InfoConverter.get().fromDbToApi(updateResult, partialInfo) as RunProject;
            //         run = converted;
            //     },
            //     handleRunRoutineUpdate: async function updateRun(apiInput) {
            //         // Update the run with the new data
            //         const { format, mutate } = ModelMap.getLogic(["format", "mutate"], "RunRoutine", true, "run process update routine cast");
            //         const input = mutate.yup.update && mutate.yup.update({ env: process.env.NODE_ENV }).cast(apiInput, { stripUnknown: true });
            //         const data = mutate.shape.update ? await mutate.shape.update({ data: input, idsCreateToConnect: {}, preMap: {}, userData }) : input;
            //         const updateResult = await DbProvider.get().run_routine.update({
            //             where: { id: run?.id || apiInput.id },
            //             data,
            //             select: runRoutineSelect,
            //         });
            //         const partialInfo = InfoConverter.get().fromApiToPartialApi(runRoutineSelect, format.apiRelMap, true);
            //         const converted = InfoConverter.get().fromDbToApi(updateResult, partialInfo) as RunRoutine;
            //         run = converted;
            //     },
            //     isStepCompleted: false,// TODO will change for multi-step support
            //     isRunCompleted: statusChangeReason === RunStatusChangeReason.Completed,
            //     logger,
            //     run,
            //     runnableObject,
            //     timeElapsed: previousTimeElapsed + (performance.now() - startTime),
            // });

            // Emit socket event to update the UI
            const status = runStatusToTaskStatus[runStatus];
            const taskSocketInfo = { run, runFrom, runId, startedById, statusChangeReason, runStatus: status as unknown as RunStatus, taskId } as const;
            // emitSocketEvent("runTask", runId, taskSocketInfo);

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
        return false;
    }

    try {
        // Let the UI know that the task is Running
        emitSocketEvent("runTask", runId, { runStatus: RunStatus.InProgress, runId, startedById, taskId } as any);
        // Get the routine and run, in a way that throws an error if they don't exist or the user doesn't have permission
        const req = buildReq(userData);
        run = await readOneHelper<Run>({
            info: RunProcessSelect.Run,
            input: { id: runId },
            objectType: "Run",
            req,
        }) as Run;
        runnableObject = await readOneHelper<ResourceVersion>({
            info: RunProcessSelect.ResourceVersion,
            input: { id: resourceVersionId },
            objectType: "ResourceVersion",
            req,
        }) as ResourceVersion;
        // Parse configs and form data
        const config = RoutineVersionConfig.deserialize(runnableObject, logger, { useFallbacks: true });
        // Collect fieldName, description, and helpText for each input and output element
        const inputFieldInfo = config.formInput?.schema.elements.map(toFormFieldInfo).filter((element) => element.fieldName) || [];
        const outputFieldInfo = config.formOutput?.schema.elements.map(toFormFieldInfo).filter((element) => element.fieldName) || [];
        // Generate input/output value object like we do in the UI. Returns object where 
        // input keys are prefixed with "input-" and output keys are prefixed with "output-"
        // formData = generateRoutineInitialValues({
        //     configFormInput: config.formInput?.schema,
        //     configFormOutput: config.formOutput?.schema,
        //     logger,
        //     runInputs: (run as RunRoutine)?.inputs,
        //     runOutputs: (run as RunRoutine)?.outputs,
        // });
        formData = {} as any;//TODO
        // Override generated inputs with provided inputs
        // if (typeof formValues === "object" && !Array.isArray(formValues)) {
        //     for (const [key, value] of Object.entries(formValues)) {
        //         if (formData[key]) {
        //             formData[key] = value;
        //         }
        //     }
        // }
        if (await checkStop()) {
            return { __typename: "Success" as const, success: false };
        }
        // Combine values and field info so that we can easily generate a task message for an LLM.
        const fullInputs = formDataToIOInfo(formData, inputFieldInfo, "input-");
        const fullOutputs = formDataToIOInfo(formData, outputFieldInfo, "output-");
        // If there is at least one missing input, create a task message for the LLM to generate the missing inputs
        const taskMessage = {} as any;//generateTaskMessage(runnableObject, fullInputs, fullOutputs, userData.languages);
        // If we have a taskMessage, this means we have missing inputs. 
        if (taskMessage) {
            // Use LLM to generate response
            let botToUse: string;
            // if (config.callData && config.callData.__type === RoutineType.Generate) {
            //     const generateSchema = (config.callData as CallDataGenerateConfig).schema;
            //     botToUse = (generateSchema.botStyle === BotStyle.Specific ? generateSchema.respondingBot : startedById) || VALYXA_ID;
            // } else {
            //     botToUse = startedById || VALYXA_ID;
            // }
            // const botInfo = await getBotInfo(botToUse);
            // if (!botInfo) {
            //     throw new CustomError("0599", "InternalError", { configCallData: config.callData });
            // }
            // const respondingBotConfig = BotSettingsConfig.deserialize(botInfo, logger).schema;
            // if (!respondingBotConfig) {
            //     throw new CustomError("0619", "InternalError");
            // }
            // const { message, cost } = await generateResponseWithFallback({
            //     force: true,
            //     maxCredits,
            //     mode: "json",
            //     participantsData,
            //     respondingBotConfig,
            //     respondingBotId: botInfo.id,
            //     stream: false,
            //     task: undefined, // Don't provide task, since we're providing custom instructions
            //     taskMessage,
            //     userData,
            // });
            // totalStepCost += BigInt(cost);
            // // Transform response to input and output values
            // const { inputs: transformedInputs, outputs: transformedOutputs } = parseRunIOFromPlaintext({ formData, text: message });
            // // Update formData
            // formData = {
            //     ...formData,
            //     // Prepend "input-" to keys for inputs and "output-" to keys for outputs
            //     ...Object.entries(transformedInputs).reduce((acc, [key, value]) => {
            //         acc[`input-${key}`] = value;
            //         return acc;
            //     }, {} as Record<string, any>),
            //     ...Object.entries(transformedOutputs).reduce((acc, [key, value]) => {
            //         acc[`output-${key}`] = value;
            //         return acc;
            //     }, {} as Record<string, any>),
            // };
        }
        // // Determine if the routine being run is the overall routine or a sub-routine
        // const isSubroutine = runType !== "RunRoutine" || (run as RunRoutine).routineVersion?.id !== routineVersion.id;
        // What we do depends on the routine type
        // const routineFunction = runnableObject.routineType ? routineTypeToFunction[runnableObject.routineType] : undefined;
        // if (!routineFunction) {
        //     throw new CustomError("0593", "InternalError", { routineType: runnableObject.routineType });
        // }
        // const { cost } = await routineFunction({
        //     configCallData: {} as any, //TODO
        //     formData,
        //     handleFormDataUpdate: (updatedForm: object) => {
        //         formData = updatedForm;
        //     },
        //     inputData: inputFieldInfo,
        //     limits: limits ?? {},
        //     outputData: outputFieldInfo,
        //     remainingCredits: maxCredits - totalStepCost,
        //     routineVersion: runnableObject,
        //     run: run as RunRoutine,
        //     userData,
        // });
        // totalStepCost += BigInt(cost);

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















const DEFAULT_MODEL_HANDLING: RunBotConfig["modelHandling"] = "OnlyWhenMissing";
const DEFAULT_PROMPT_HANDLING: RunBotConfig["promptHandling"] = "Combine";
const DEFAULT_RESPONDING_BOT_HANDLING: RunBotConfig["respondingBotHandling"] = "OnlyWhenMissing";
const DEFAULT_BOT_ID = SEEDED_IDS.User.Valyxa;

/**
 * Accepted output formats for AI-generated input and output values.
 * 
 * We accept different formats because some models are better at different formats.
 */
export enum RunOutputFormat {
    Json = "Json",
    LineByLine = "LineByLine",
    Xml = "Xml",
}

type BotInfo = {
    /** The ID of the bot */
    id: string;
    /** The maximum number of credits the bot can spend */
    maxCredits: bigint | undefined;
    /** The settings for the bot */
    settings: BotSettings
}

/**
 * Handles the logic for setting up the bot config and instructions for a subroutine
 */
class SubroutineBotHandler {
    // Example outputs for the different output formats
    private exampleJson = `\`\`\`json
  {
    "name": "John Doe",
    "age": 30,
    "isActive": true
  }
  \`\`\``;
    private exampleLineByLine = `\`\`\`
  name: John Doe
  age: 30
  isActive: true
  \`\`\``;
    private exampleXml = `\`\`\`xml
  <response>
    <name>John Doe</name>
    <age>30</age>
    <isActive>true</isActive>
  </response>
  \`\`\``;

    // The subroutine 
    private subroutine: ResourceVersion;
    // The config to use for the subroutine
    private subroutineBotConfig: ConfigCallDataGenerate;
    // The user's data
    private userData: SessionUser;

    // Data required to call the LLM
    private botInfo: BotInfo | null = null;

    constructor(runBotConfig: RunBotConfig, subroutine: ResourceVersion, userData: SessionUser) {
        // Store properties
        this.subroutine = subroutine;
        this.userData = userData;
        // Deserialize the subroutine's config
        const subroutineConfig = RoutineVersionConfig.deserialize(subroutine, logger, { useFallbacks: true });
        // Combine with the run's bot config to build the final bot config
        this.subroutineBotConfig = this.buildBotConfig(runBotConfig, subroutineConfig.callDataGenerate?.schema ?? {});
    }

    /**
     * Combines the overall run bot config with the subroutine bot config.
     * 
     * @param runBotConfig The bot config for the overall run
     * @param subroutineBotConfig The bot config for the subroutine
     * @returns The combined bot config
     */
    private buildBotConfig(runBotConfig: RunBotConfig, subroutineBotConfig: ConfigCallDataGenerate): ConfigCallDataGenerate {
        // Use documented defaults if the handling options aren't specified.
        const modelHandling = runBotConfig.modelHandling || DEFAULT_MODEL_HANDLING;
        const promptHandling = runBotConfig.promptHandling || DEFAULT_PROMPT_HANDLING;
        const respondingBotHandling = runBotConfig.respondingBotHandling || DEFAULT_RESPONDING_BOT_HANDLING;

        // Determine the model:
        // - If "Override", always use runBotConfig's model.
        // - If "OnlyWhenMissing", use subroutine's model if present, otherwise the run-level model.
        let finalModel = subroutineBotConfig.model;
        if (modelHandling === "Override" && runBotConfig.model) {
            finalModel = runBotConfig.model;
        } else if (modelHandling === "OnlyWhenMissing" && !finalModel && runBotConfig.model) {
            finalModel = runBotConfig.model;
        }

        // Determine the prompt:
        // - If "Override", use runBotConfig's prompt exclusively.
        // - If "Combine", concatenate the two (if both exist) with a newline.
        let finalPrompt: string | null | undefined = subroutineBotConfig.prompt;
        if (promptHandling === "Override" && runBotConfig.prompt) {
            finalPrompt = runBotConfig.prompt;
        } else if (promptHandling === "Combine") {
            if (runBotConfig.prompt && subroutineBotConfig.prompt) {
                finalPrompt = `${runBotConfig.prompt}\n${subroutineBotConfig.prompt}`;
            } else {
                finalPrompt = runBotConfig.prompt || subroutineBotConfig.prompt;
            }
        }

        // Determine the responding bot:
        // - If "Override", always use runBotConfig's respondingBot.
        // - If "OnlyWhenMissing", use subroutine's value if available; otherwise, use the run-level value.
        let finalRespondingBot = subroutineBotConfig.respondingBot;
        if (respondingBotHandling === "Override" && runBotConfig.respondingBot) {
            finalRespondingBot = runBotConfig.respondingBot;
        } else if (respondingBotHandling === "OnlyWhenMissing" && !finalRespondingBot && runBotConfig.respondingBot) {
            finalRespondingBot = runBotConfig.respondingBot;
        }

        // Return a combined configuration. Note that we spread subroutineBotConfig
        // so that properties like botStyle and maxTokens are preserved.
        return {
            ...subroutineBotConfig,
            model: finalModel,
            prompt: finalPrompt,
            respondingBot: finalRespondingBot,
        };
    }

    /**
     * Retrieves and formats the subroutine's information to pass as context to the LLM.
     * 
     * @returns A string containing the formatted routine version information.
     */
    private getRoutineVersionInfo(): string {
        let info = "";
        const { description, instructions, name } = getTranslation(this.subroutine, this.userData.languages, true);
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
     * Builds an example code block for the given output format.
     * 
     * @param outputFormat The format of the output
     * @returns An example code block for the given output format
     */
    private getExampleOutput(outputFormat: RunOutputFormat): string {
        switch (outputFormat) {
            case RunOutputFormat.Json:
                return this.exampleJson;
            case RunOutputFormat.Xml:
                return this.exampleXml;
            case RunOutputFormat.LineByLine:
                return this.exampleLineByLine;
            default:
                return "";
        }
    }

    /**
     * Loads all of the information required to call the LLM.
     * 
     * This includes:
     * - Finding the best available model
     * - Using that model's configuration and the subroutine's config to determine the correct limits 
     *   and configuration for the LLM call.
     * - Loading the bot config for the responding bot, if applicable.
     */
    public async loadBotInfo() {
        // Return early if we already have the model info
        if (this.botInfo) {
            return;
        }

        // Find the desired bot to use
        const botId = this.subroutineBotConfig.respondingBot ?? DEFAULT_BOT_ID;
        const botData = await DbProvider.get().user.findUnique({
            where: { id: botId },
            select: {
                id: true,
                // Used to check if you have access to the bot if it's marked as private
                invitedByUser: {
                    select: {
                        id: true,
                    },
                },
                isBot: true,
                isPrivate: true,
                botSettings: true,
                name: true,
            },
        });
        if (!botData) {
            throw new CustomError("0224", "NotFound", { botId });
        }
        // Verify that the user has access to the bot
        const canUseBot = botData.isPrivate === false || botData.invitedByUser?.id === this.userData.id;
        if (!canUseBot) {
            throw new CustomError("0226", "Unauthorized", { botId });
        }
        const botSettings = BotSettingsConfig.deserialize(botData, logger).schema;

        let maxCredits: number | undefined = Math.max(Math.min(botSettings.maxTokens ?? Number.MAX_SAFE_INTEGER, this.subroutineBotConfig.maxTokens ?? Number.MAX_SAFE_INTEGER), 0);
        if (maxCredits === Number.MAX_SAFE_INTEGER) {
            maxCredits = undefined;
        }

        // Set the bot info
        this.botInfo = {
            id: botId,
            maxCredits: maxCredits ? BigInt(maxCredits) : undefined,
            settings: botSettings,
        };
    }

    /**
     * Makes sure the bot info has been properly loaded.
     * 
     * @param botInfo The bot info to check
     * @throws If the bot info has not been loaded
     */
    private assertBotInfoLoaded(botInfo: BotInfo | null): asserts botInfo is BotInfo {
        if (!this.botInfo) {
            throw new Error("Bot info has not been loaded.");
        }
    }

    /**
     * Formats the inputs or outputs that need to be generated into a readable string for the LLM prompt.
     * 
     * @param ioMap The input or output mapping to format
     * @returns A formatted string describing the inputs to generate.
     */
    private formatIOValuesForPrompt(inputs: Record<string, SubroutineInputDisplayInfo | SubroutineOutputDisplayInfo>): string {
        // Filter inputs where value is undefined (needs to be generated)
        const entriesToGenerate = Object.entries(inputs).filter(([, entryInfo]) => entryInfo.value === undefined);

        // Handle case where no inputs need generation
        if (entriesToGenerate.length === 0) {
            return "No inputs need to be generated.";
        }

        // Build the formatted string
        let result = "";
        for (const [key, entryInfo] of entriesToGenerate) {
            result += `- ${key}:\n`;
            if (entryInfo.name) {
                result += `  Name: "${entryInfo.name}"\n`;
            }
            if (entryInfo.description) {
                result += `  Description: "${entryInfo.description}"\n`;
            }
            if (Object.prototype.hasOwnProperty.call(entryInfo, "isRequired")) {
                result += `  Required: ${(entryInfo as SubroutineInputDisplayInfo).isRequired ? "true" : "false"}\n`;
            }

            // Handle props (type, schema, and other properties)
            if (entryInfo.props) {
                for (const [propKey, propValue] of Object.entries(entryInfo.props)) {
                    if (propKey === "type") {
                        result += `  Type: "${propValue}"\n`;
                    } else if (propKey === "schema" && typeof propValue === "string") {
                        try {
                            // Try to parse and pretty-print the schema if it’s JSON
                            let prettySchema = propValue;
                            try {
                                const schemaObj = JSON.parse(propValue);
                                prettySchema = JSON.stringify(schemaObj, null, 2);
                            } catch {
                                // If it's not valid JSON, include it as is
                            }
                            result += `  Schema:\n${prettySchema.split("\n").map(line => `    ${line}`).join("\n")}\n`;
                        } catch {
                            // If not valid JSON, include it as is
                            result += `  Schema: "${propValue}"\n`;
                        }
                    } else if (typeof propValue === "string" || typeof propValue === "number" || typeof propValue === "boolean") {
                        // Simple key-value pairs for primitive types
                        result += `  ${uppercaseFirstLetter(propKey)}: ${propValue}\n`;
                    } else {
                        // Stringify complex values (e.g., objects)
                        result += `  ${uppercaseFirstLetter(propKey)}: ${JSON.stringify(propValue)}\n`;
                    }
                }
            }
        }
        return result;
    }

    /**
     * Builds the introductory text for a task.
     * 
     * This is common across all task types.
     * 
     * @param taskDescription - The description of the task
     * @param outputFormat - The format of the output
     * @returns The introductory text for the task
     */
    private getIntroText(
        taskDescription: string,
        outputFormat: RunOutputFormat,
    ): string {
        const exampleOutput = this.getExampleOutput(outputFormat);
        return `I am providing you with information about a task we're working on. ${taskDescription}
  
  Your goal is to generate the missing information and respond with a valid ${outputFormat.toUpperCase()} formatted response. Ensure that your response contains *only* the ${outputFormat.toUpperCase()} code, with no additional commentary.
  
  Here is an example:
  ${exampleOutput}`;
    }

    /**
     * Generates a message for tasks that only require input generation.
     *
     * @param ioMapping The IOMapping of the subroutine
     * @param outputFormat The desired output format (e.g., "json", "xml", or "line"). Defaults to "json".
     * @returns A string containing the task message for input generation.
     */
    private generateInputOnlyMessage(
        ioMapping: SubroutineIOMapping,
        outputFormat: RunOutputFormat = RunOutputFormat.Json,
    ): string {
        const intro = this.getIntroText(
            "It includes inputs, some of which need to be filled out for us to continue with the task.",
            outputFormat,
        );
        const taskInfo = `${this.getRoutineVersionInfo()}
  Inputs: \n${this.formatIOValuesForPrompt(ioMapping.inputs)}`;

        return `${intro}\n\nHere is some information about the task:\n\n${taskInfo}`;
    }

    /**
     * Generates a message for tasks that only require output generation.
     *
     * @param ioMapping The IOMapping of the subroutine
     * @param outputFormat The desired output format (e.g., "json", "xml", or "line"). Defaults to "json".
     * @returns A string containing the task message for output generation.
     */
    private generateOutputOnlyMessage(
        ioMapping: SubroutineIOMapping,
        outputFormat: RunOutputFormat = RunOutputFormat.Json,
    ): string {
        const intro = this.getIntroText(
            "We need to generate outputs for this task.",
            outputFormat,
        );
        const taskInfo = `${this.getRoutineVersionInfo()}
  Outputs: \n${this.formatIOValuesForPrompt(ioMapping.outputs)}`;

        return `${intro}\n\nHere is some information about the task:\n\n${taskInfo}`;
    }

    /**
     * Generates a message for tasks that require both input and output generation.
     *
     * @param ioMapping The IOMapping of the subroutine
     * @param outputFormat The desired output format (e.g., "json", "xml", or "line"). Defaults to "json".
     * @returns A string containing the task message for both input and output generation.
     */
    private generateInputAndOutputMessage(
        ioMapping: SubroutineIOMapping,
        outputFormat: RunOutputFormat = RunOutputFormat.Json,
    ): string {
        const intro = this.getIntroText(
            "It includes inputs that need to be filled out and outputs that need to be generated.",
            outputFormat,
        );
        const taskInfo = `${this.getRoutineVersionInfo()}
  Inputs: \n${this.formatIOValuesForPrompt(ioMapping.inputs)}
  
  Outputs: \n${this.formatIOValuesForPrompt(ioMapping.outputs)}`;

        return `${intro}\n\nHere is some information about the task:\n\n${taskInfo}`;
    }

    /**
     * Checks if there are missing inputs in the given ioMapping.
     * 
     * @param ioMapping The IOMapping of the subroutine
     * @returns True if there are missing inputs, false otherwise
     */
    private hasMissingInputs(ioMapping: SubroutineIOMapping): boolean {
        return Object.values(ioMapping.inputs).some((input) => input.value === undefined);
    }

    /**
     * Checks if there are missing outputs in the given ioMapping.
     * 
     * @param ioMapping The IOMapping of the subroutine
     * @returns True if there are missing outputs, false otherwise
     */
    private hasMissingOutputs(ioMapping: SubroutineIOMapping): boolean {
        return Object.values(ioMapping.outputs).some((output) => output.value === undefined);
    }

    /**
     * Generates missing inputs for a subroutine, and updates the ioMapping reference with the missing inputs.
     * 
     * @param ioMapping The IOMapping of the subroutine
     * @param outputFormat The desired output format. Defaults to "json".
     * @returns The cost of generating the missing inputs as a stringified bigint
     */
    public async generateInputs(ioMapping: SubroutineIOMapping, outputFormat: RunOutputFormat = RunOutputFormat.Json): Promise<string> {
        this.assertBotInfoLoaded(this.botInfo);

        // Return early if there are no missing inputs
        if (!this.hasMissingInputs(ioMapping)) {
            return BigInt(0).toString();
        }

        // If needed, can add additional context to provide things like the overall task description, inputs from other subroutines, etc.
        // Since that adds a lot of context length and complexity, we'll add it in later if needed.
        const taskMessage = this.generateInputOnlyMessage(ioMapping, outputFormat);
        const { cost, message } = await generateResponseWithFallback({
            chatId: null, // Not tied to a chat (at least for now🤔)
            force: false, // Only applicable for command mode (e.g. creating a note). Since we're providing a custom task, this isn't relevant
            latestMessage: null, // Not tied to a chat (at least for now🤔), so not relevant
            maxCredits: this.botInfo.maxCredits,
            mode: outputFormat === RunOutputFormat.Json ? "json" : "text", // Only supports JSON or text
            participantsData: null, // Only needed for chat mode, as it adds available participants for the bot to mention in the response
            respondingBotConfig: this.botInfo.settings,
            respondingBotId: this.botInfo.id,
            stream: false, // Not necessary to stream the response
            task: undefined, // We're providing a custom task, so not relevant
            taskMessage,
            userData: this.userData,
        });
        // Parse inputs from the response
        const inputs = {}; //TODO
        // Update the ioMapping with the missing inputs
        //TODO
        // Return the cost of generating the missing inputs
        return BigInt(cost).toString();
    }

    /**
     * Estimates the cost of generating missing inputs for a subroutine.
     * 
     * @param inputsLength The number of inputs the subroutine has. Used as an upper bound for the cost.
     * @returns The estimated cost of generating the missing inputs as a stringified bigint
     */
    public async estimateInputsCost(inputsLength: number): Promise<string> {
        this.assertBotInfoLoaded(this.botInfo);

        // Return early if there are no inputs
        if (inputsLength === 0) {
            return BigInt(0).toString();
        }

        // Get the AI service and model to use
        const serviceId = LlmServiceRegistry.get().getBestService(this.botInfo.settings.model);
        if (!serviceId) {
            return BigInt(0).toString();
        }
        const serviceInstance = LlmServiceRegistry.get().getService(serviceId);
        const model = serviceInstance.getModel(this.botInfo.settings.model);

        // Estimate the number of input and output tokens based on the number of inputs being generated
        // We can't know exactly how many tokens will be generated, so we'll use a very conservative estimate
        const INPUT_TOKENS_PER_INPUT = 100; // Change if needed
        const OUTPUT_TOKENS_PER_INPUT = 100; // Change if needed
        const inputTokens = inputsLength * INPUT_TOKENS_PER_INPUT;
        const outputTokens = inputsLength * OUTPUT_TOKENS_PER_INPUT;

        // Calculate the cost of the input and output tokens
        const cost = serviceInstance.getResponseCost({ model, usage: { input: inputTokens, output: outputTokens } });

        // Return the cost
        return BigInt(cost).toString();
    }

    /**
     * Generates missing inputs and outputs for a subroutine, and updates the ioMapping reference with the missing inputs and outputs.
     * 
     * @param ioMapping The IOMapping of the subroutine
     * @param outputFormat The desired output format. Defaults to "json".
     * @returns The cost of generating the missing inputs and outputs as a stringified bigint
     */
    public async generateInputsAndOutputs(ioMapping: SubroutineIOMapping, outputFormat: RunOutputFormat = RunOutputFormat.Json): Promise<string> {
        this.assertBotInfoLoaded(this.botInfo);

        // Find message to send to the LLM based on what's missing
        const hasMissingInputs = this.hasMissingInputs(ioMapping);
        const hasMissingOutputs = this.hasMissingOutputs(ioMapping);
        const message = hasMissingInputs
            ? hasMissingOutputs
                ? this.generateInputAndOutputMessage(ioMapping)
                : this.generateInputOnlyMessage(ioMapping)
            : hasMissingOutputs
                ? this.generateOutputOnlyMessage(ioMapping)
                : "";
        // Return early if we already have the inputs and outputs
        if (message.length === 0) {
            return BigInt(0).toString();
        }

        //TODO need to build context here or pass it in, then generate response message and parse it
        return BigInt(0).toString();
    }

    /**
     * Estimates the cost of generating missing inputs and outputs for a subroutine.
     * 
     * @param inputsLength The number of inputs the subroutine has. Used as an upper bound for the cost.
     * @param outputsLength The number of outputs the subroutine has. Used as an upper bound for the cost.
     * @returns The estimated cost of generating the missing inputs and outputs as a stringified bigint
     */
    public async estimateInputsAndOutputsCost(inputsLength: number, outputsLength: number): Promise<string> {
        this.assertBotInfoLoaded(this.botInfo);

        // Return early if there are no inputs or outputs
        if (inputsLength === 0 && outputsLength === 0) {
            return BigInt(0).toString();
        }

        // Get the AI service and model to use
        const serviceId = LlmServiceRegistry.get().getBestService(this.botInfo.settings.model);
        if (!serviceId) {
            return BigInt(0).toString();
        }
        const serviceInstance = LlmServiceRegistry.get().getService(serviceId);
        const model = serviceInstance.getModel(this.botInfo.settings.model);

        // Estimate the number of input and output tokens based on the number of inputs and outputs being generated
        // We can't know exactly how many tokens will be generated, so we'll use a very conservative estimate
        const INPUT_TOKENS_PER_INPUT = 100; // Change if needed
        const OUTPUT_TOKENS_PER_INPUT = 100; // Change if needed
        const INPUT_TOKENS_PER_OUTPUT = 100; // Change if needed
        const OUTPUT_TOKENS_PER_OUTPUT = 100; // Change if needed
        const inputTokens = inputsLength * INPUT_TOKENS_PER_INPUT + outputsLength * INPUT_TOKENS_PER_OUTPUT;
        const outputTokens = inputsLength * OUTPUT_TOKENS_PER_INPUT + outputsLength * OUTPUT_TOKENS_PER_OUTPUT;

        // Calculate the cost of the input and output tokens
        const cost = serviceInstance.getResponseCost({ model, usage: { input: inputTokens, output: outputTokens } });

        // Return the cost
        return BigInt(cost).toString();
    }
}

/**
 * Decision strategy that pauses the run until the user makes a decision.
 */
export class ManualPickSelectionHandler extends PathSelectionHandler {
    __type = PathSelectionStrategy.ManualPick;

    async pickOne(options: DecisionOption[], decisionKey: string): Promise<DeferredDecisionData> {
        return {
            __type: "Waiting" as const,
            decisionType: "chooseOne" as const,
            key: decisionKey,
            options,
        };
    }

    async pickMultiple(options: DecisionOption[], decisionKey: string): Promise<DeferredDecisionData> {
        return {
            __type: "Waiting" as const,
            decisionType: "chooseMultiple" as const,
            key: decisionKey,
            options,
        };
    }
}

/**
 * Decision strategy that uses an LLM to pick the best option.
 */
export class AutoPickLLMSelectionHandler extends PathSelectionHandler {
    __type = PathSelectionStrategy.AutoPickLLM;

    async pickOne(options: DecisionOption[], decisionKey: string): Promise<ResolvedDecisionDataChooseOne> {
        //TODO
        return {} as any;
    }

    async pickMultiple(options: DecisionOption[], decisionKey: string): Promise<ResolvedDecisionDataChooseMultiple> {
        //TODO
        return {} as any;
    }
}

export class ServerRunLoader extends RunLoader {
    // The data of the user who is running the routine
    private userData: SessionUser;

    constructor(userData: SessionUser) {
        super();
        this.userData = userData;
    }

    public async fetchLocation(location: Location): Promise<LocationData | null> {
        let object: ResourceVersion | null = null;
        let subroutine: ResourceVersion | null = null;

        // Fetch the main object by its __typename and objectId
        const req = buildReq(this.userData);
        object = await readOneHelper({
            info: RunProcessSelect.ResourceVersion,
            input: { id: location.objectId },
            objectType: "ResourceVersion" as const,
            req,
        }) as ResourceVersion;

        // If there's a subroutineId, fetch that as well
        if (location.subroutineId) {
            subroutine = await readOneHelper({
                info: RunProcessSelect.ResourceVersion,
                input: { id: location.subroutineId },
                objectType: "ResourceVersion" as const,
                req,
            }) as ResourceVersion;
        }

        // If nothing was found for the main object, return null
        if (!object) {
            return null;
        }

        return {
            object,
            subroutine,
        };
    }

    protected onCacheChange(): void {
        // No-op unless we add caching
    }
}

export class ServerRunNotifier extends RunNotifier {
    // The data of the user who is running the routine
    private userData: SessionUser;

    constructor(userData: SessionUser) {
        super();
        this.userData = userData;
    }

    /**
     * Determines the type of notification to send.
     * 
     * @param runId The ID of the run
     * @returns "push" if the run has an active websocket connection, "websocket" otherwise
     */
    private getNotificationType(runId: string): "push" | "websocket" {
        return roomHasOpenConnections(runId) ? "websocket" : "push";
    }

    public emitProgressUpdate(runId: string, payload: RunTaskInfo): void {
        const notificationType = this.getNotificationType(runId);
        // Only needed if run has active websocket connection
        if (notificationType === "websocket") {
            emitSocketEvent("runTask", runId, payload);
        }
    }

    public sendDecisionRequest(runId: string, decision: DeferredDecisionData): void {
        const notificationType = this.getNotificationType(runId);
        // Send through websocket if run has active connection
        if (notificationType === "websocket") {
            emitSocketEvent("runTaskDecisionRequest", runId, decision);
        }
        // Otherwise, send through push notification
        else {
            Notify(this.userData.languages).pushNewDecisionRequest(decision, runId).toUser(this.userData.id);
        }
    }

    public sendMissingInputsRequest(runId: string, branch: BranchProgress): void {
        const notificationType = this.getNotificationType(runId);
        //TODO
    }

    public sendManualExecutionConfirmationRequest(runId: string, branch: BranchProgress): void {
        const notificationType = this.getNotificationType(runId);
        //TODO
    }
}

class ServerRunPersistence extends RunPersistence {
    // The data of the user who is running the routine
    private userData: SessionUser;

    constructor(userData: SessionUser) {
        super();
        this.userData = userData;
    }

    async postRun(input: RunCreateInput): Promise<Run | null> {
        const req = buildReq(this.userData);
        const result = await createOneHelper({
            info: RunProcessSelect.Run,
            input,
            objectType: "Run",
            req,
        }) as Run | null;
        return result;
    }

    async putRun(input: RunUpdateInput): Promise<Run | null> {
        const req = buildReq(this.userData);
        const result = await updateOneHelper({
            info: RunProcessSelect.Run,
            input,
            objectType: "Run",
            req,
        }) as Run | null;
        return result;
    }

    async fetchRun(run: RunIdentifier): Promise<Run | null> {
        const req = buildReq(this.userData);
        const storedData = await readOneHelper({
            info: RunProcessSelect.Run,
            input: { id: run.runId },
            objectType: "Run",
            req,
        }) as Run | null;
        return storedData ?? null;
    }
}

type CurrentSubroutineData = {
    /** The ID of the subroutine instance */
    subroutineInstanceId: string;
    /** The bot handler for the subroutine */
    botHandler: SubroutineBotHandler;
}

type ActionResult = {
    /** The cost of the action */
    cost: string;
}

class ServerSubroutineExecutor extends SubroutineExecutor {
    // The data of the user who is running the routine
    private userData: SessionUser;

    // Data for the last or current subroutine
    private currentSubroutineData: CurrentSubroutineData | null = null;

    constructor(userData: SessionUser) {
        super();
        this.userData = userData;
    }

    /**
     * Get or initialize the current subroutine data.
     * 
     * @param subroutineInstanceId The ID of the subroutine instance
     * @param routine The suboutine to run
     * @param runConfig The run config
     * @returns The current subroutine data
     */
    private async getCurrentSubroutineData(subroutineInstanceId: string, routine: ResourceVersion, runConfig: RunConfig): Promise<CurrentSubroutineData> {
        // If we already have the data, return it
        if (this.currentSubroutineData && this.currentSubroutineData.subroutineInstanceId === subroutineInstanceId) {
            return this.currentSubroutineData;
        }
        // Otherwise, create a new bot handler and return it
        const botHandler = new SubroutineBotHandler(runConfig.botConfig, routine, this.userData);
        await botHandler.loadBotInfo();
        // Update the current subroutine data
        this.currentSubroutineData = {
            subroutineInstanceId,
            botHandler,
        };
        // Return the current subroutine data
        return this.currentSubroutineData;
    }

    private async runAction(routine: ResourceVersion, ioMapping: SubroutineIOMapping): Promise<ActionResult> {
        const cost = BigInt(0).toString();

        // Deserialize the routine config
        const routineConfig = RoutineVersionConfig.deserialize(routine, logger);
        if (!routineConfig.callDataAction) {
            logger.error("No call data action found", { trace: "0647", resourceVersionId: routine.id });
            return { cost };
        }

        // Get the input
        const taskInput = routineConfig.callDataAction.buildTaskInput(ioMapping, this.userData.languages);

        // Get the action handler
        const taskExec = await generateTaskExec(routineConfig.callDataAction.schema.task as Exclude<LlmTask, "Start">, this.userData.languages[0], this.userData);

        // Execute the action
        const result = await taskExec(taskInput as LlmTaskData);

        // Parse the result
        routineConfig.callDataAction.parseActionResult(ioMapping, result);

        // Return the cost (which shouldn't have changed with the current implementation)
        return { cost };
    }

    private async runApi(routine: ResourceVersion, ioMapping: SubroutineIOMapping): Promise<ActionResult> {
        const cost = BigInt(0).toString();

        //TODO
        return {
            cost,
        };
    }

    private async runCode(routine: ResourceVersion, ioMapping: SubroutineIOMapping): Promise<ActionResult> {
        const cost = BigInt(0).toString();

        const code = routine.relatedVersions?.find(v => v.toVersion?.resourceSubType.startsWith("Code"))?.toVersion;
        if (!code) {
            logger.error("No code version found", { trace: "0626", resourceVersionId: routine.id });
            return { cost };
        }

        // Validate permissions to run code
        const hasPermissions = await permissionsCheck(
            { [code.id]: { ...code, __typename: "ResourceVersion" } },
            { ["Read"]: [code.id] },
            {},
            this.userData,
            true,
        );
        if (!hasPermissions) {
            logger.error("User does not have permissions to run code", { trace: "0628", resourceVersionId: routine.id, userId: this.userData.id });
            return { cost };
        }

        // Deserialize the routine and code configs
        const routineConfig = RoutineVersionConfig.deserialize(routine, logger);
        const codeConfig = CodeVersionConfig.deserialize(code, logger);
        if (!routineConfig.callDataCode) {
            logger.error("No call data code found", { trace: "0633", resourceVersionId: routine.id });
            return { cost };
        }

        // Create the payload for running the code
        const { input, shouldSpreadInput } = routineConfig.callDataCode.buildSandboxInput(ioMapping, codeConfig.inputConfig) ?? { input: undefined, shouldSpreadInput: false };
        const payload: RunUserCodeInput = {
            code: String.raw`${codeConfig.content}`,
            codeLanguage: code.codeLanguage as CodeLanguage,
            input,
            shouldSpreadInput,
        };


        // Run the code (this will reject unsupported languages)
        const sandboxOutput = await runUserCode(payload);

        // Parse the output (updates ioMapping)
        routineConfig.callDataCode.parseSandboxOutput(sandboxOutput, ioMapping, codeConfig.outputConfig);

        // Return the cost (which shouldn't have changed with the current implementation)
        return { cost };
    }

    private async runData(routine: ResourceVersion, ioMapping: SubroutineIOMapping): Promise<ActionResult> {
        const cost = BigInt(0).toString();

        //TODO
        return {
            cost,
        };
    }

    private async runInformational(routine: ResourceVersion, ioMapping: SubroutineIOMapping): Promise<ActionResult> {
        const cost = BigInt(0).toString();

        //TODO
        return {
            cost,
        };
    }

    private async runSmartContract(routine: ResourceVersion, ioMapping: SubroutineIOMapping): Promise<ActionResult> {
        const cost = BigInt(0).toString();

        //TODO
        return {
            cost,
        };
    }

    //TODO some inputs will have to be found through search instead of being generated
    async generateMissingInputs(subroutineInstanceId: string, routine: ResourceVersion, ioMapping: SubroutineIOMapping, runConfig: RunConfig): Promise<Omit<RunSubroutineResult, "updatedBranchStatus">> {
        const { botHandler } = await this.getCurrentSubroutineData(subroutineInstanceId, routine, runConfig);
        // Generate missing inputs
        const cost = await botHandler.generateInputs(ioMapping);
        // Return result
        return {
            cost,
            inputs: ioMapping.inputs,
            outputs: ioMapping.outputs,
        };
    }

    async runSubroutine(subroutineInstanceId: string, routine: ResourceVersion, ioMapping: SubroutineIOMapping, runConfig: RunConfig): Promise<RunSubroutineResult> {
        const result: RunSubroutineResult = {
            cost: BigInt(0).toString(),
            inputs: ioMapping.inputs,
            outputs: ioMapping.outputs,
        };

        // Set up the subroutine bot handler
        const { botHandler } = await this.getCurrentSubroutineData(subroutineInstanceId, routine, runConfig);

        let ioCost = BigInt(0).toString();
        // For Generate subroutines, we can generate the missing inputs and outputs at the same time
        if (routine.resourceSubType === ResourceSubType.RoutineGenerate) {
            ioCost = await botHandler.generateInputsAndOutputs(ioMapping);
        }
        // For other subroutines types, generate missing inputs
        else {
            ioCost = await botHandler.generateInputs(ioMapping);
        }
        // Add the inputs/outputs cost to the total cost
        result.cost = (BigInt(ioCost) + BigInt(result.cost)).toString();

        let actionResult: ActionResult;
        // Run the subroutine based on its type
        // NOTE: We don't support multi-step subroutines here
        switch (routine.resourceSubType) {
            case ResourceSubType.RoutineAction: {
                actionResult = await this.runAction(routine, ioMapping);
                break;
            }
            case ResourceSubType.RoutineApi: {
                actionResult = await this.runApi(routine, ioMapping);
                break;
            }
            case ResourceSubType.RoutineCode: {
                actionResult = await this.runCode(routine, ioMapping);
                break;
            }
            case ResourceSubType.RoutineData: {
                actionResult = await this.runData(routine, ioMapping);
                break;
            }
            case ResourceSubType.RoutineGenerate:
                // Do nothing. This is handled above
                actionResult = {
                    cost: BigInt(0).toString(),
                };
                break;
            case ResourceSubType.RoutineInformational: {
                actionResult = await this.runInformational(routine, ioMapping);
                break;
            }
            case ResourceSubType.RoutineSmartContract: {
                actionResult = await this.runSmartContract(routine, ioMapping);
                break;
            }
            default:
                throw new Error(`Unknown or unsupported routine type: ${routine.routineType}`);
        }
        // Add the action cost to the total cost
        result.cost = (BigInt(actionResult.cost) + BigInt(result.cost)).toString();

        // Reduce the user's credits by the cost of the subroutine
        await reduceUserCredits(this.userData.id, BigInt(result.cost));

        return result;
    }

    async estimateCost(subroutineInstanceId: string, routine: ResourceVersion, runConfig: RunConfig): Promise<string> {
        const { botHandler } = await this.getCurrentSubroutineData(subroutineInstanceId, routine, runConfig);

        let ioCost = BigInt(0).toString();
        // Estimate the cost to generate the inputs and outputs
        if (routine.resourceSubType === ResourceSubType.RoutineGenerate) {
            ioCost = await botHandler.estimateInputsAndOutputsCost(routine.inputsCount ?? routine.inputs.length, routine.outputsCount ?? routine.outputs.length);
        } else {
            ioCost = await botHandler.estimateInputsCost(routine.inputsCount ?? routine.inputs.length);
        }

        const actionCost = BigInt(0).toString();
        // Estimate the cost to perform the subroutine action
        // Only some subroutine types have a cost associated with them
        switch (routine.resourceSubType) {
            case ResourceSubType.RoutineApi:
                //TODO
                break;
            case ResourceSubType.RoutineSmartContract:
                //TODO
                break;
            // NOTE: "Generate" cost is already estimated above
        }

        const totalCost = (BigInt(ioCost) + BigInt(actionCost)).toString();
        return totalCost;
    }
}

export interface ActiveRunRecord {
    /** Whether the user has premium status */
    hasPremium: boolean;
    /** The time the run was added to the registry */
    startTime: number;
    /** The unique ID of the run */
    runId: string;
}

/**
 * Registry for active runs.
 * 
 * Allows for cancelling/pausing a run or changing its config.
 */
export class ActiveRunsRegistry {
    // Internal map for fast lookup by runId.
    private runsMap = new Map<string, RunStateMachine>();

    // Ordered list (oldest first) of runs.
    private runsList: ActiveRunRecord[] = [];

    /**
     * Adds a new run to the registry.
     *
     * @param runId - The unique ID of the run.
     * @param stateMachine - The associated RunStateMachine instance.
     * @param hasPremium - Whether the user has premium status.
     * @param startTime - (Optional) The start time in milliseconds. Defaults to now.
     * @throws An error if the runId already exists.
     */
    public add(runId: string, stateMachine: RunStateMachine, hasPremium: boolean, startTime: number = Date.now()): void {
        if (this.runsMap.has(runId)) {
            throw new Error(`Run with id ${runId} already exists in the registry.`);
        }
        this.runsMap.set(runId, stateMachine);
        this.runsList.push({ hasPremium, runId, startTime });
    }

    /**
     * Removes a run from the registry.
     *
     * @param runId - The unique ID of the run.
     * @returns true if the run was removed, false if it wasn't found.
     */
    public remove(runId: string): boolean {
        if (!this.runsMap.has(runId)) {
            return false;
        }
        this.runsMap.delete(runId);
        // Remove the record from the list.
        const index = this.runsList.findIndex(record => record.runId === runId);
        if (index !== -1) {
            this.runsList.splice(index, 1);
        }
        return true;
    }

    /**
     * Retrieves the RunStateMachine for a given runId.
     *
     * @param runId - The unique ID of the run.
     * @returns The RunStateMachine instance, or undefined if not found.
     */
    public get(runId: string): RunStateMachine | undefined {
        return this.runsMap.get(runId);
    }

    /**
     * Returns the list of active run records in order (oldest first).
     */
    public getOrderedRuns(): ActiveRunRecord[] {
        // Because we push new records to the end, runsList is already in insertion order.
        return this.runsList;
    }

    /**
     * Returns the number of active runs.
     */
    public count(): number {
        return this.runsMap.size;
    }

    /**
     * Clears all runs from the registry.
     */
    public clear(): void {
        this.runsMap.clear();
        this.runsList = [];
    }
}
export const activeRunsRegistry = new ActiveRunsRegistry();

/**
 * Process run requests for both new and existing runs.
 * 
 * Supports resuming runs, cancelling runs, and changing run configurations.
 */
export async function runProcess({ data }: Job<RunRequestPayload>): Promise<Success> {
    // Handle test
    if (data.__process === "Test") {
        logger.info("runProcess test triggered");
        return { __typename: "Success" as const, success: true };
    }

    const { config, isNewRun, runId, userData } = data;

    // Set up the decision strategy based on the provided config (or a default)
    const pathSelectionStrategy = config?.decisionConfig?.pathSelection ?? PathSelectionStrategy.AutoPickFirst;
    let pathSelectionHandler: PathSelectionHandler;
    switch (pathSelectionStrategy) {
        case "AutoPickFirst":
            pathSelectionHandler = new AutoPickFirstSelectionHandler();
            break;
        case "AutoPickLLM":
            pathSelectionHandler = new AutoPickLLMSelectionHandler();
            break;
        case "AutoPickRandom":
            pathSelectionHandler = new AutoPickRandomSelectionHandler();
            break;
        case "ManualPick":
            pathSelectionHandler = new ManualPickSelectionHandler();
            break;
        default:
            throw new Error(`Unsupported path selecetion strategy type: ${pathSelectionStrategy}`);
    }

    // Either reuse an existing state machine or create a new one.
    let stateMachine: RunStateMachine;
    const existingStateMachine = activeRunsRegistry.get(runId);
    if (existingStateMachine) {
        // For an existing run, retrieve the state machine.
        stateMachine = existingStateMachine;
        // If a new config was provided, update the state machine.
        if (config) {
            stateMachine.updateRunConfig(config);
            if (
                config.decisionConfig.pathSelection &&
                stateMachine.getPathSelectionHandler().__type !== config.decisionConfig.pathSelection
            ) {
                stateMachine.setPathSelectionHandler(pathSelectionHandler);
            }
        }
    } else {
        // Create a new state machine instance.
        const loader = new ServerRunLoader(userData);
        const notifier = new ServerRunNotifier(userData);
        const persistence = new ServerRunPersistence(userData);
        const subroutineExecutor = new ServerSubroutineExecutor(userData);
        stateMachine = new RunStateMachine({
            loader,
            logger,
            navigatorFactory,
            notifier,
            pathSelectionHandler,
            persistence,
            subroutineExecutor,
        });
        // Add the state machine to the active runs registry.
        activeRunsRegistry.add(runId, stateMachine, userData.hasPremium);
    }

    // Initialize the run
    if (isNewRun) {
        // Ensure config is provided. 
        // For now, the default config is very conservative, so that we don't run away with the API credits.
        const runConfig: RunConfig = config ?? {
            botConfig: {},
            decisionConfig: {
                inputGeneration: InputGenerationStrategy.Auto,
                pathSelection: pathSelectionStrategy,
                subroutineExecution: SubroutineExecutionStrategy.Auto,
            },
            isPrivate: true,
            isTimeSensitive: false,
            limits: {
                maxCredits: (BigInt(100) * API_CREDITS_MULTIPLIER).toString(), // $1
                maxSteps: 100,
                maxTime: MINUTES_1_MS,
                onMaxCredits: "Stop",
                onMaxSteps: "Stop",
                onMaxTime: "Stop",
            },
            loopConfig: {},
            testMode: false,
        };
        const startLocations = []; //TODO
        await stateMachine.initNewRun(startLocations, runConfig, userData);
    } else {
        const runInfo: RunIdentifier = { runId };
        await stateMachine.initExistingRun(runInfo, userData);
    }

    let result: Success = { __typename: "Success" as const, success: false };
    try {
        // Wrap the state machine run in a global timeout.
        await Promise.race([
            stateMachine.runUntilDone(),
            new Promise<never>((_resolve, reject) =>
                setTimeout(() => reject(new Error(`Global run timeout exceeded for run ${runId}`)), RUN_TIMEOUT_MS),
            ),
        ]);
        // Normal termination
        result = { __typename: "Success" as const, success: true };
    } catch (error) {
        logger.error(`Run ${runId} terminated due to global timeout or error. Initiating graceful shutdown.`, error);
        // Initiate a graceful shutdown (or pause) and wait for the grace period.
        await stateMachine.stopRun(RunStatus.Paused);
        await new Promise((resolve) => setTimeout(resolve, RUN_SHUTDOWN_GRACE_PERIOD_MS));
    } finally {
        activeRunsRegistry.remove(runId);
    }
    return result;
}
