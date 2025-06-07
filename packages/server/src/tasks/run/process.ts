import { API_CREDITS_MULTIPLIER, AutoPickFirstSelectionHandler, AutoPickRandomSelectionHandler, BotConfig, CodeVersionConfig, InputGenerationStrategy, MINUTES_1_MS, PathSelectionHandler, PathSelectionStrategy, ResourceSubType, RoutineVersionConfig, RunLoader, RunNotifier, RunPersistence, RunStateMachine, RunStatus, SEEDED_PUBLIC_IDS, SubroutineExecutionStrategy, SubroutineExecutor, getTranslation, navigatorFactory, uppercaseFirstLetter, type BotConfigObject, type BranchProgress, type CodeLanguage, type ConfigCallDataGenerate, type DecisionOption, type DeferredDecisionData, type Location, type LocationData, type ResolvedDecisionDataChooseMultiple, type ResolvedDecisionDataChooseOne, type ResourceVersion, type Run, type RunBotConfig, type RunConfig, type RunCreateInput, type RunIdentifier, type RunSubroutineResult, type RunTaskInfo, type RunUpdateInput, type SessionUser, type SubroutineIOMapping, type SubroutineInputDisplayInfo, type SubroutineOutputDisplayInfo, type Success } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { type Job } from "bullmq";
import { createOneHelper } from "../../actions/creates.js";
import { readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { Notify } from "../../notify/notify.js";
import { AIServiceRegistry } from "../../services/conversation/registry.js";
import { SocketService } from "../../sockets/io.js";
import { runUserCode } from "../../tasks/sandbox/process.js";
import { type SessionData } from "../../types.js";
import { reduceUserCredits } from "../../utils/reduceCredits.js";
import { permissionsCheck } from "../../validators/permissions.js";
import { BaseActiveTaskRegistry, type BaseActiveTaskRecord } from "../activeTaskRegistry.js";
import type { RunUserCodeInput } from "../sandbox/types.js";
import { RUN_QUEUE_LIMITS } from "./queue.js";

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
            id: true,
            data: true,
            nodeInputName: true,
            nodeName: true,
        },
        name: true,
        resourceVersion: {
            id: true,
            complexity: true,
            isDeleted: true,
            isPrivate: true,
            resourceSubType: true,
            root: {
                id: true,
                ownedByTeam: {
                    id: true,
                    isPrivate: true,
                },
                ownedByUser: {
                    id: true,
                    isPrivate: true,
                },
            },
        },
        status: true,
        steps: {
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
            id: true,
            labels: true,
            toVersion: {
                id: true,
                codeLanguage: true,
                config: true,
            },
        },
        resourceSubType: true,
        root: {
            id: true,
            ownedByTeam: {
                id: true,
                isPrivate: true,
            },
            ownedByUser: {
                id: true,
                isPrivate: true,
            },
            resourceType: true,
        },
        translations: {
            id: true,
            language: true,
            description: true,
            details: true,
            instructions: true,
            name: true,
        },
    },
} as const;

/**
 * Builds the minimum Request object needed to perform CRUD operations for a Run.
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

const DEFAULT_MODEL_HANDLING: RunBotConfig["modelHandling"] = "OnlyWhenMissing";
const DEFAULT_PROMPT_HANDLING: RunBotConfig["promptHandling"] = "Combine";
const DEFAULT_RESPONDING_BOT_HANDLING: RunBotConfig["respondingBotHandling"] = "OnlyWhenMissing";

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
    /** The configuration for the bot */
    config: BotConfigObject
    /** The maximum number of credits the bot can spend */
    maxCredits: bigint | undefined;
}

/**
 * Handles the logic for setting up the bot config and instructions for a subroutine
 */
class SubroutineBotHandler {
    //     // Example outputs for the different output formats
    //     private exampleJson = `\`\`\`json
    //   {
    //     "name": "John Doe",
    //     "age": 30,
    //     "isActive": true
    //   }
    //   \`\`\``;
    //     private exampleLineByLine = `\`\`\`
    //   name: John Doe
    //   age: 30
    //   isActive: true
    //   \`\`\``;
    //     private exampleXml = `\`\`\`xml
    //   <response>
    //     <name>John Doe</name>
    //     <age>30</age>
    //     <isActive>true</isActive>
    //   </response>
    //   \`\`\``;

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
        const subroutineConfig = RoutineVersionConfig.parse(subroutine, logger, { useFallbacks: true });
        // Combine with the run's bot config to build the final bot config
        this.subroutineBotConfig = this.buildBotConfig(runBotConfig, subroutineConfig.callDataGenerate?.schema ?? {});
    }

    /**
     * Safely gets the `where` clause for the bot to use.
     * 
     * @param respondingBotInfo The bot info to use
     * @returns The `where` clause for the bot to use
     */
    private getBotWhereClause(respondingBotInfo: ConfigCallDataGenerate["respondingBot"] | null | undefined): Prisma.userWhereUniqueInput {
        let where: Prisma.userWhereUniqueInput;
        if (!respondingBotInfo) {
            return { publicId: SEEDED_PUBLIC_IDS.Valyxa };
        }
        if (respondingBotInfo.id) {
            where = { id: BigInt(respondingBotInfo.id) };
        } else if (respondingBotInfo.publicId) {
            where = { publicId: respondingBotInfo.publicId };
        } else if (respondingBotInfo.handle) {
            where = { handle: respondingBotInfo.handle };
        } else {
            where = { publicId: SEEDED_PUBLIC_IDS.Valyxa };
        }
        return where;
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
        const where = this.getBotWhereClause(this.subroutineBotConfig.respondingBot);
        const botData = await DbProvider.get().user.findUnique({
            where,
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
            throw new CustomError("0224", "NotFound", { where });
        }
        // Verify that the user has access to the bot
        const canUseBot = botData.isPrivate === false || botData.invitedByUser?.id.toString() === this.userData.id;
        if (!canUseBot) {
            throw new CustomError("0226", "Unauthorized", { where });
        }
        const botSettings = BotConfig.parse(botData, logger).schema;

        let maxCredits: number | undefined = Math.max(Math.min(botSettings.maxTokens ?? Number.MAX_SAFE_INTEGER, this.subroutineBotConfig.maxTokens ?? Number.MAX_SAFE_INTEGER), 0);
        if (maxCredits === Number.MAX_SAFE_INTEGER) {
            maxCredits = undefined;
        }

        // Set the bot info
        this.botInfo = {
            id: botData.id.toString(),
            config: botSettings,
            maxCredits: maxCredits ? BigInt(maxCredits) : undefined,
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
                            // Try to parse and pretty-print the schema if it's JSON
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
            respondingBotConfig: this.botInfo.config,
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
        const serviceId = AIServiceRegistry.get().getBestService(this.botInfo.settings.model);
        if (!serviceId) {
            return BigInt(0).toString();
        }
        const serviceInstance = AIServiceRegistry.get().getService(serviceId);
        const model = serviceInstance.getModel(this.botInfo.config.model);

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
        const serviceId = AIServiceRegistry.get().getBestService(this.botInfo.config.model);
        if (!serviceId) {
            return BigInt(0).toString();
        }
        const serviceInstance = AIServiceRegistry.get().getService(serviceId);
        const model = serviceInstance.getModel(this.botInfo.config.model);

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
        return SocketService.get().roomHasOpenConnections(runId) ? "websocket" : "push";
    }

    public emitProgressUpdate(runId: string, payload: RunTaskInfo): void {
        const notificationType = this.getNotificationType(runId);
        // Only needed if run has active websocket connection
        if (notificationType === "websocket") {
            SocketService.get().emitSocketEvent("runTask", runId, payload);
        }
    }

    public sendDecisionRequest(runId: string, decision: DeferredDecisionData): void {
        const notificationType = this.getNotificationType(runId);
        // Send through websocket if run has active connection
        if (notificationType === "websocket") {
            SocketService.get().emitSocketEvent("runTaskDecisionRequest", runId, decision);
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
        const routineConfig = RoutineVersionConfig.parse(routine, logger);
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

        const code = routine.relatedVersions?.find(v => v.toVersion?.resourceSubType?.startsWith("Code"))?.toVersion;
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
        const routineConfig = RoutineVersionConfig.parse(routine, logger);
        const codeConfig = CodeVersionConfig.parse(code, logger);
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
            case ResourceSubType.RoutineInternalAction: {
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
                throw new Error(`Unknown or unsupported routine type: ${routine.resourceSubType}`);
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

export type ActiveRunRecord = BaseActiveTaskRecord;
export class ActiveRunsRegistry extends BaseActiveTaskRegistry<ActiveRunRecord, RunStateMachine> {
    // Add run-specific registry setup here
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
        activeRunsRegistry.add(data, stateMachine);
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
                setTimeout(() => reject(new Error(`Global run timeout exceeded for run ${runId}`)), RUN_QUEUE_LIMITS.timeoutMs),
            ),
        ]);
        // Normal termination
        result = { __typename: "Success" as const, success: true };
    } catch (error) {
        logger.error(`Run ${runId} terminated due to global timeout or error. Initiating graceful shutdown.`, error);
        // Initiate a graceful shutdown (or pause) and wait for the grace period.
        await stateMachine.stopRun(RunStatus.Paused);
        await new Promise((resolve) => setTimeout(resolve, RUN_QUEUE_LIMITS.shutdownGracePeriodMs));
    } finally {
        activeRunsRegistry.remove(runId);
    }
    return result;
}
