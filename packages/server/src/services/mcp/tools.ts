import type { BotCreateInput, BotUpdateInput, ResourceVersionCreateInput, ResourceVersionUpdateInput, SessionUser, SwarmSubTask, TeamCreateInput, TeamUpdateInput } from "@vrooli/shared";
import { DEFAULT_LANGUAGE, DeleteType, InputGenerationStrategy, McpToolName, ModelType, PathSelectionStrategy, ResourceSubType, ResourceType, RunTriggeredFrom, SubroutineExecutionStrategy, TeamConfig, generatePK, type ChatMessage, type ChatMessageCreateInput, type MessageConfigObject, type TaskContextInfo, type TeamConfigObject } from "@vrooli/shared";
import fs from "fs";
import { fileURLToPath } from "node:url";
import * as path from "path";
import { createOneHelper } from "../../actions/creates.js";
import { deleteOneHelper } from "../../actions/deletes.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import type { RequestService } from "../../auth/request.js";
import type { PartialApiInfo } from "../../builders/types.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { activeSwarmRegistry } from "../../tasks/swarm/process.js";
import { QueueTaskType, type RunTask, type SwarmExecutionTask } from "../../tasks/taskTypes.js";
import { type CoreResourceAllocation } from "@vrooli/shared";
import { type ConversationStateStore } from "../response/chatStore.js";
import type { BotAddAttributes, BotUpdateAttributes, NoteAddAttributes, NoteUpdateAttributes, ProjectAddAttributes, ProjectUpdateAttributes, RoutineApiAddAttributes, RoutineApiUpdateAttributes, RoutineCodeAddAttributes, RoutineCodeUpdateAttributes, RoutineDataAddAttributes, RoutineDataUpdateAttributes, RoutineGenerateAddAttributes, RoutineGenerateUpdateAttributes, RoutineInformationalAddAttributes, RoutineInformationalUpdateAttributes, RoutineMultiStepAddAttributes, RoutineMultiStepUpdateAttributes, RoutineSmartContractAddAttributes, RoutineSmartContractUpdateAttributes, RoutineWebAddAttributes, RoutineWebUpdateAttributes, StandardDataStructureAddAttributes, StandardDataStructureUpdateAttributes, StandardPromptAddAttributes, StandardPromptUpdateAttributes, TeamAddAttributes, TeamUpdateAttributes } from "../types/resources.js";
import { type DefineToolParams, type EndSwarmParams, type Recipient, type ResourceManageParams, type RunRoutineParams, type SendMessageParams, type SpawnSwarmParams, type UpdateSwarmSharedStateParams } from "../types/tools.js";
import { loadSchema } from "./schemaLoader.js";
import { type ToolResponse } from "./types.js";

// Load schema using the schema loader utility
const resourceManageSchema = loadSchema("ResourceManage/schema.json");

// Type definitions for schema structures
interface SchemaProperty {
    type: string;
    const?: string;
    enum?: string[];
    description?: string;
    properties?: Record<string, SchemaProperty>;
    required?: string[];
    additionalProperties?: boolean;
}

interface SchemaDefinition {
    description: string;
    type: string;
    properties: Record<string, SchemaProperty>;
    required?: string[];
    additionalProperties?: boolean;
}

interface ToolSchema {
    name: string;
    description: string;
    inputSchema: {
        anyOf?: SchemaDefinition[];
        oneOf?: SchemaDefinition[];
        type?: string;
        properties?: Record<string, SchemaProperty>;
        required?: string[];
        additionalProperties?: boolean;
    };
    annotations?: {
        title?: string;
        readOnlyHint?: boolean;
        openWorldHint?: boolean;
    };
}

// Type guards for discriminated unions
function isRecipientChat(recipient: Recipient): recipient is { kind: "chat"; chatId: string } {
    return recipient.kind === "chat";
}

function isRecipientBot(recipient: Recipient): recipient is { kind: "bot"; botId: string } {
    return recipient.kind === "bot";
}

function isRecipientUser(recipient: Recipient): recipient is { kind: "user"; userId: string } {
    return recipient.kind === "user";
}

function isRecipientTopic(recipient: Recipient): recipient is { kind: "topic"; topic: string } {
    return recipient.kind === "topic";
}

function isResourceManageFindParams(params: ResourceManageParams): params is Extract<ResourceManageParams, { op: "find" }> {
    return params.op === "find";
}

function isResourceManageAddParams(params: ResourceManageParams): params is Extract<ResourceManageParams, { op: "add" }> {
    return params.op === "add";
}

function isResourceManageUpdateParams(params: ResourceManageParams): params is Extract<ResourceManageParams, { op: "update" }> {
    return params.op === "update";
}

function isResourceManageDeleteParams(params: ResourceManageParams): params is Extract<ResourceManageParams, { op: "delete" }> {
    return params.op === "delete";
}

function isRunRoutineStart(params: RunRoutineParams): params is Extract<RunRoutineParams, { action: "start" }> {
    return params.action === "start";
}

function isSpawnSwarmSimple(params: SpawnSwarmParams): params is Extract<SpawnSwarmParams, { kind: "simple" }> {
    return params.kind === "simple";
}

// Discriminated union for all resource add attributes
type ResourceAddAttributes =
    | NoteAddAttributes
    | ProjectAddAttributes
    | RoutineMultiStepAddAttributes
    | RoutineApiAddAttributes
    | RoutineCodeAddAttributes
    | RoutineDataAddAttributes
    | RoutineGenerateAddAttributes
    | RoutineInformationalAddAttributes
    | RoutineSmartContractAddAttributes
    | RoutineWebAddAttributes
    | StandardDataStructureAddAttributes
    | StandardPromptAddAttributes
    | TeamAddAttributes
    | BotAddAttributes;

// Discriminated union for all resource update attributes
type ResourceUpdateAttributes =
    | NoteUpdateAttributes
    | ProjectUpdateAttributes
    | RoutineMultiStepUpdateAttributes
    | RoutineApiUpdateAttributes
    | RoutineCodeUpdateAttributes
    | RoutineDataUpdateAttributes
    | RoutineGenerateUpdateAttributes
    | RoutineInformationalUpdateAttributes
    | RoutineSmartContractUpdateAttributes
    | RoutineWebUpdateAttributes
    | StandardDataStructureUpdateAttributes
    | StandardPromptUpdateAttributes
    | TeamUpdateAttributes
    | BotUpdateAttributes;

// Type guard for NoteAddAttributes
function isNoteAddAttributes(attrs: unknown): attrs is NoteAddAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return typeof obj.name === "string" &&
        typeof obj.content === "string" &&
        (obj.tagsConnect === undefined || Array.isArray(obj.tagsConnect));
}

// Type guard for ProjectAddAttributes
function isProjectAddAttributes(attrs: unknown): attrs is ProjectAddAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return typeof obj.name === "string" &&
        (obj.config === undefined || typeof obj.config === "object");
}

// Type guard for TeamAddAttributes
function isTeamAddAttributes(attrs: unknown): attrs is TeamAddAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return (obj.config === undefined || typeof obj.config === "object") &&
        (obj.memberInvitesCreate === undefined || Array.isArray(obj.memberInvitesCreate));
}

// Type guard for BotAddAttributes
function isBotAddAttributes(attrs: unknown): attrs is BotAddAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return (obj.config === undefined || typeof obj.config === "object");
}

// Helper to check if attrs has routine version properties
function hasRoutineVersionProperties(attrs: Record<string, unknown>): boolean {
    return "resourceSubType" in attrs &&
        typeof attrs.resourceSubType === "string" &&
        attrs.resourceSubType.startsWith("Routine") &&
        "rootCreate" in attrs &&
        typeof attrs.rootCreate === "object" &&
        attrs.rootCreate !== null &&
        (attrs.rootCreate as any).resourceType === ResourceType.Routine;
}

// Helper to check if attrs has standard version properties
function hasStandardVersionProperties(attrs: Record<string, unknown>): boolean {
    return "resourceSubType" in attrs &&
        typeof attrs.resourceSubType === "string" &&
        attrs.resourceSubType.startsWith("Standard") &&
        "rootCreate" in attrs &&
        typeof attrs.rootCreate === "object" &&
        attrs.rootCreate !== null &&
        (attrs.rootCreate as any).resourceType === ResourceType.Standard;
}

// Generic type guard for routine subtypes
function isRoutineAddAttributes(attrs: unknown, subType: ResourceSubType): attrs is RoutineMultiStepAddAttributes | RoutineApiAddAttributes | RoutineCodeAddAttributes | RoutineDataAddAttributes | RoutineGenerateAddAttributes | RoutineInformationalAddAttributes | RoutineSmartContractAddAttributes | RoutineWebAddAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return hasRoutineVersionProperties(obj) && obj.resourceSubType === subType;
}

// Generic type guard for standard subtypes
function isStandardAddAttributes(attrs: unknown, subType: ResourceSubType): attrs is StandardDataStructureAddAttributes | StandardPromptAddAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return hasStandardVersionProperties(obj) && obj.resourceSubType === subType;
}

// Type guards for update attributes
function isNoteUpdateAttributes(attrs: unknown): attrs is NoteUpdateAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    // All fields are optional for updates
    return (obj.name === undefined || typeof obj.name === "string") &&
        (obj.content === undefined || typeof obj.content === "string");
}

function isProjectUpdateAttributes(attrs: unknown): attrs is ProjectUpdateAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return (obj.name === undefined || typeof obj.name === "string") &&
        (obj.config === undefined || typeof obj.config === "object");
}

function isTeamUpdateAttributes(attrs: unknown): attrs is TeamUpdateAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return (obj.config === undefined || typeof obj.config === "object") &&
        (obj.memberInvitesCreate === undefined || Array.isArray(obj.memberInvitesCreate)) &&
        (obj.memberInvitesDelete === undefined || Array.isArray(obj.memberInvitesDelete)) &&
        (obj.membersDelete === undefined || Array.isArray(obj.membersDelete));
}

function isBotUpdateAttributes(attrs: unknown): attrs is BotUpdateAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return (obj.config === undefined || typeof obj.config === "object") &&
        (obj.handle === undefined || typeof obj.handle === "string") &&
        (obj.isPrivate === undefined || typeof obj.isPrivate === "boolean");
}

// Helper to check if attrs has routine version update properties
function hasRoutineVersionUpdateProperties(attrs: Record<string, unknown>): boolean {
    return "rootUpdate" in attrs &&
        typeof attrs.rootUpdate === "object" &&
        attrs.rootUpdate !== null &&
        (attrs.rootUpdate as any).resourceType === ResourceType.Routine;
}

// Helper to check if attrs has standard version update properties
function hasStandardVersionUpdateProperties(attrs: Record<string, unknown>): boolean {
    return "rootUpdate" in attrs &&
        typeof attrs.rootUpdate === "object" &&
        attrs.rootUpdate !== null &&
        (attrs.rootUpdate as any).resourceType === ResourceType.Standard;
}

// Generic type guard for routine update subtypes
function isRoutineUpdateAttributes(attrs: unknown): attrs is RoutineMultiStepUpdateAttributes | RoutineApiUpdateAttributes | RoutineCodeUpdateAttributes | RoutineDataUpdateAttributes | RoutineGenerateUpdateAttributes | RoutineInformationalUpdateAttributes | RoutineSmartContractUpdateAttributes | RoutineWebUpdateAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return hasRoutineVersionUpdateProperties(obj);
}

// Generic type guard for standard update subtypes
function isStandardUpdateAttributes(attrs: unknown): attrs is StandardDataStructureUpdateAttributes | StandardPromptUpdateAttributes {
    if (typeof attrs !== "object" || attrs === null) {
        return false;
    }
    const obj = attrs as Record<string, unknown>;
    return hasStandardVersionUpdateProperties(obj);
}

// Type-safe metadata accessors
interface McpMetadata {
    messageConfig?: MessageConfigObject;
    mcpLlmModel?: string;
    mcpLlmTaskContexts?: TaskContextInfo[];
}

function extractMcpMetadata(metadata: Record<string, unknown> | undefined): McpMetadata {
    if (!metadata) return {};

    // For messageConfig, we need to ensure it has the required __version property
    let messageConfig: MessageConfigObject | undefined = undefined;
    if (typeof metadata.messageConfig === "object" && metadata.messageConfig !== null) {
        const config = metadata.messageConfig as Record<string, unknown>;
        if (typeof config.__version === "string" && Array.isArray(config.resources)) {
            messageConfig = config as unknown as MessageConfigObject;
        }
    }

    return {
        messageConfig,
        mcpLlmModel: typeof metadata.mcpLlmModel === "string"
            ? metadata.mcpLlmModel
            : undefined,
        mcpLlmTaskContexts: Array.isArray(metadata.mcpLlmTaskContexts)
            ? metadata.mcpLlmTaskContexts as TaskContextInfo[]
            : undefined,
    };
}

// Default message config factory
function createDefaultMessageConfig(): MessageConfigObject {
    return {
        __version: "1.0",
        resources: [],
        role: "assistant",
    };
}

/**
 * Holds all built-in MCP tools.
 */
export class BuiltInTools {
    private readonly user: SessionUser;
    private readonly req: Parameters<typeof RequestService.assertRequestFrom>[0];

    constructor(user: SessionUser, req?: Parameters<typeof RequestService.assertRequestFrom>[0]) {
        this.user = user;
        this.req = req ?? {
            session: {
                fromSafeOrigin: true,
                isLoggedIn: user?.id ? true : false,
                languages: user?.languages ?? [DEFAULT_LANGUAGE],
                userId: user?.id ?? "",
                users: [user],
            },
        };
    }

    /**
     * Handler for the DefineTool MCP tool.
     * Returns the detailed input schema for ResourceManage given a resource variant and CRUD operation.
     *
     * @param args - The arguments for the tool, specifying the variant and operation.
     * @returns A ToolResponse object containing the JSON schema for the specified tool operation.
     */
    async defineTool(args: DefineToolParams): Promise<ToolResponse> {
        const { variant, op, toolName } = args;

        // Validate the tool name
        const supportedToolNames = [McpToolName.ResourceManage];
        if (!supportedToolNames.includes(toolName)) {
            const errorMsg = `Error: defineTool only supports ${supportedToolNames.join(", ")}, but received ${toolName}.`;
            logger.warn(errorMsg);
            return {
                isError: true,
                content: [{ type: "text", text: errorMsg }],
            };
        }

        const currentFilePath = fileURLToPath(import.meta.url);
        const currentDirPath = path.dirname(currentFilePath);

        // Get and validate the tool definition
        let toolDef: ToolSchema | undefined = undefined;
        switch (toolName) {
            case McpToolName.ResourceManage:
                toolDef = resourceManageSchema as ToolSchema;
                break;
        }

        if (!toolDef?.inputSchema?.anyOf && !toolDef?.inputSchema?.oneOf) {
            const errorMsg = `Error: Could not retrieve base schema for ${toolName}.`;
            logger.error(errorMsg);
            return {
                isError: true,
                content: [{ type: "text", text: errorMsg }],
            };
        }

        // Use anyOf or oneOf depending on what's available
        const schemaOptions = toolDef.inputSchema.anyOf || toolDef.inputSchema.oneOf || [];

        // Grab the correct operation schema template
        const opSchemaTemplate = schemaOptions.find(
            (s: SchemaDefinition) => s.properties?.op?.const === op,
        );

        if (!opSchemaTemplate) {
            const availableOps = schemaOptions
                .map((s: SchemaDefinition) => s.properties?.op?.const)
                .filter(Boolean);
            const errorMsg = `Error: Could not find schema for operation '${op}' in ${toolName}. Available operations: ${availableOps.join(", ")}.`;
            logger.error(errorMsg);
            return {
                isError: true,
                content: [{ type: "text", text: errorMsg }],
            };
        }

        // Deep clone the template
        const finalSchema: SchemaDefinition = JSON.parse(JSON.stringify(opSchemaTemplate));

        // General schema updates
        finalSchema.description = `Defines the parameters required to ${op} a resource of type ${variant}.`;
        if (finalSchema.properties) {
            finalSchema.properties.resource_type = { type: "string", const: variant };
        }

        // Operation-specific property refinement
        switch (op) {
            case "find": {
                this.addFilterSchemaForVariant(finalSchema, variant, currentDirPath);
                break;
            }
            case "add": {
                this.addAttributeSchemaForVariant(finalSchema, variant, currentDirPath, "add");
                break;
            }
            case "update": {
                this.addAttributeSchemaForVariant(finalSchema, variant, currentDirPath, "update");
                break;
            }
            case "delete": {
                // Intentionally blank. All delete operations are the same. No additional properties needed.
                break;
            }
            default: {
                logger.warn(`DefineTool received an unhandled op: ${op} for variant ${variant}. Returning base schema for op.`);
                break;
            }
        }

        return {
            isError: false,
            content: [
                {
                    type: "text",
                    text: JSON.stringify(finalSchema, null, 2),
                },
            ],
        };
    }

    private addFilterSchemaForVariant(finalSchema: SchemaDefinition, variant: string, currentDirPath: string): void {
        let filterProperties: Record<string, SchemaProperty> = {};
        let filterRequired: string[] = [];
        const schemaFilePath = path.join(currentDirPath, "../..", "schemas", "DefineTool", variant, "find_filters.json");

        try {
            const fileContent = fs.readFileSync(schemaFilePath, "utf-8");
            const loadedSchema = JSON.parse(fileContent);
            if (typeof loadedSchema === "object" && loadedSchema !== null) {
                filterProperties = { ...loadedSchema };
                if (Array.isArray(loadedSchema.required)) {
                    filterRequired = [...loadedSchema.required];
                    delete filterProperties.required; // Remove 'required' if it was part of the properties block itself
                }
            } else {
                logger.warn(`Loaded schema from ${schemaFilePath} is not a valid object.`);
            }
        } catch (error) {
            logger.warn(`Filter schema file not found or failed to parse for ${variant} at ${schemaFilePath}: ${(error as Error).message}`);
        }

        if (finalSchema.properties) {
            finalSchema.properties.filters = {
                type: "object",
                description: "Optional filters to filter results",
                properties: filterProperties,
                required: filterRequired,
                additionalProperties: false,
            };
        }
    }

    private addAttributeSchemaForVariant(finalSchema: SchemaDefinition, variant: string, currentDirPath: string, operation: "add" | "update"): void {
        let attributeProperties: Record<string, SchemaProperty> = {};
        let attributeRequired: string[] = [];
        const schemaFilePath = path.join(currentDirPath, "../..", "schemas", "DefineTool", variant, `${operation}_attributes.json`);

        try {
            const fileContent = fs.readFileSync(schemaFilePath, "utf-8");
            const loadedSchema = JSON.parse(fileContent);
            if (typeof loadedSchema === "object" && loadedSchema !== null) {
                attributeProperties = { ...loadedSchema };
                if (Array.isArray(loadedSchema.required)) {
                    attributeRequired = [...loadedSchema.required];
                    delete attributeProperties.required; // Remove 'required' if it was part of the properties block itself
                }
            } else {
                logger.warn(`Loaded schema from ${schemaFilePath} is not a valid object.`);
            }
        } catch (error) {
            logger.warn(`Attributes schema file not found or failed to parse for ${variant} (${operation}) at ${schemaFilePath}: ${(error as Error).message}`);
        }

        if (finalSchema.properties) {
            finalSchema.properties.attributes = {
                type: "object",
                description: `Attributes for ${operation === "add" ? "adding a new" : "updating an existing"} ${variant}.`,
                properties: attributeProperties,
                required: attributeRequired,
                additionalProperties: false,
            };
        }
    }

    /**
     * Handler for the SendMessage MCP tool.
     * 
     * @param args - The arguments for the tool, specifying recipient and content.
     * @returns A ToolResponse indicating the result of the sendMessage operation.
     */
    async sendMessage(args: SendMessageParams): Promise<ToolResponse> {
        logger.info(`sendMessage called with new params: ${JSON.stringify(args)}`);

        try {
            const agentSenderId = this.user.id;

            // Resolve recipient to chatId using type guards
            let chatId: string;
            if (isRecipientChat(args.recipient)) {
                chatId = args.recipient.chatId;
            } else if (isRecipientBot(args.recipient)) {
                // TODO: Implement logic to find/create DM chat with bot
                const errorMsg = `Error: Recipient kind 'bot' resolution to a chat ID is not yet implemented. Bot ID: ${args.recipient.botId}`;
                logger.error(errorMsg, { recipient: args.recipient });
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            } else if (isRecipientUser(args.recipient)) {
                // TODO: Implement logic to find/create DM chat with user
                const errorMsg = `Error: Recipient kind 'user' resolution to a chat ID is not yet implemented. User ID: ${args.recipient.userId}`;
                logger.error(errorMsg, { recipient: args.recipient });
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            } else if (isRecipientTopic(args.recipient)) {
                // TODO: Implement MQTT/topic-based messaging
                const errorMsg = `Error: Recipient kind 'topic' is not yet implemented. Topic: ${args.recipient.topic}`;
                logger.error(errorMsg, { recipient: args.recipient });
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            } else {
                const errorMsg = `Error: Unknown recipient kind in: ${JSON.stringify(args.recipient)}`;
                logger.error(errorMsg);
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            }

            if (!chatId) {
                const errorMsg = `Error: Invalid or missing chat ID from recipient: ${JSON.stringify(args.recipient)}.`;
                logger.error(errorMsg);
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            }

            const messageContent = typeof args.content === "string" ? args.content : JSON.stringify(args.content);
            const metadata = extractMcpMetadata(args.metadata);

            const chatMessageInput: ChatMessageCreateInput = {
                id: generatePK().toString(),
                chatConnect: chatId,
                userConnect: agentSenderId,
                text: messageContent,
                language: this.user.languages[0] ?? DEFAULT_LANGUAGE,
                versionIndex: 0,
                config: metadata.messageConfig || createDefaultMessageConfig(),
            };

            const additionalDataForHelper = {
                model: metadata.mcpLlmModel,
                taskContexts: metadata.mcpLlmTaskContexts,
            };

            if (!additionalDataForHelper.model) {
                logger.warn(`sendMessage: 'mcpLlmModel' was not found in metadata for chat ${chatId}. Bot responses might not be triggered or use a default model.`);
            }

            logger.info(`Attempting to create chat message via createOneHelper. ChatID: ${chatId}, SenderID: ${agentSenderId}, RecipientKind: ${args.recipient.kind}`);

            const result = await createOneHelper({
                input: chatMessageInput,
                additionalData: additionalDataForHelper,
                objectType: "ChatMessage",
                req: this.req,
                info: { GQLResolveInfo: {} } as PartialApiInfo,
            }) as Partial<ChatMessage> & { id: string };

            logger.info(`Chat message created: ${result.id} in chat ${chatId}`);
            return {
                isError: false,
                content: [{ type: "text", text: `Message sent to chat ${chatId} (recipient kind: ${args.recipient.kind}). Message ID: ${result.id}` }],
            };

        } catch (error) {
            logger.error("Error in sendMessage:", error);
            const errorMessage = error instanceof Error ? error.message : "An unknown error occurred.";
            return { isError: true, content: [{ type: "text", text: errorMessage }] };
        }
    }

    /**
     * Handler for the ResourceManage MCP tool.
     * 
     * @param args - Parameters for resource management (CRUD operations).
     * @returns A ToolResponse indicating the result of the operation.
     */
    async resourceManage(args: ResourceManageParams): Promise<ToolResponse> {
        logger.info(`resourceManage called with args: ${JSON.stringify(args)}`);
        let result;
        try {
            if (isResourceManageFindParams(args)) {
                const input = this._mapFindToInput(args.resource_type, args.filters || {});
                result = await readManyHelper({ info: {}, input, objectType: "ResourceVersion", req: this.req });
            } else if (isResourceManageAddParams(args)) {
                const createInput = this._mapAddToInput(args.resource_type, args.attributes as ResourceAddAttributes);
                // Determine the correct objectType based on the resource type
                let objectType: ModelType | `${ModelType}`;
                if (args.resource_type === "Team") {
                    objectType = ModelType.Team;
                } else if (args.resource_type === "Bot") {
                    objectType = ModelType.User; // Bots are stored in the User table
                } else {
                    objectType = ModelType.ResourceVersion;
                }
                result = await createOneHelper({ info: {}, input: createInput, objectType, req: this.req });
            } else if (isResourceManageUpdateParams(args)) {
                const updateInput = this._mapUpdateToInput(args.id, args.resource_type, args.attributes as ResourceUpdateAttributes);
                // Determine the correct objectType based on the resource type
                let objectType: ModelType;
                if (args.resource_type === "Team") {
                    objectType = ModelType.Team;
                } else if (args.resource_type === "Bot") {
                    objectType = ModelType.User; // Bots are stored in the User table
                } else {
                    objectType = ModelType.ResourceVersion;
                }
                result = await updateOneHelper({ info: {}, input: updateInput, objectType, req: this.req });
            } else if (isResourceManageDeleteParams(args)) {
                result = await deleteOneHelper({ input: { id: args.id, objectType: DeleteType.ResourceVersion }, req: this.req });
            } else {
                // This should never happen with proper TypeScript discriminated unions
                const errorMsg = "Unhandled resource manage operation";
                logger.error(errorMsg, args);
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            }

            return { isError: false, content: [{ type: "text", text: JSON.stringify(result) }] };
        } catch (error) {
            logger.error("Error in resourceManage:", error);
            return { isError: true, content: [{ type: "text", text: (error as Error).message }] };
        }
    }

    /**
     * Handler for the RunRoutine MCP tool.
     * 
     * @param args - Parameters for routine execution.
     * @returns A ToolResponse indicating the result of the runRoutine operation.
     */
    async runRoutine(args: RunRoutineParams): Promise<ToolResponse> {
        logger.info(`runRoutine called with args: ${JSON.stringify(args)}`);

        try {
            if (isRunRoutineStart(args)) {
                // Handle "start" action - create a new run
                return await this._handleRunRoutineStart(args);
            } else {
                // Handle manage actions: "pause", "resume", "cancel", "status"
                return await this._handleRunRoutineManage(args);
            }
        } catch (error) {
            logger.error("Error in runRoutine:", error);
            return { isError: true, content: [{ type: "text", text: (error as Error).message }] };
        }
    }

    /**
     * Handles starting a new routine run.
     */
    private async _handleRunRoutineStart(args: Extract<RunRoutineParams, { action: "start" }>): Promise<ToolResponse> {
        try {
            // Detect parent swarm context
            const parentSwarmId = await this.getParentSwarmContext();
            
            if (parentSwarmId) {
                logger.info("[runRoutine] Running in swarm-integrated mode", {
                    routineId: args.routineId,
                    parentSwarmId,
                    userId: this.user.id,
                });
                return await this._handleSwarmIntegratedRoutine(args, parentSwarmId);
            } else {
                logger.info("[runRoutine] Running in standalone mode", {
                    routineId: args.routineId,
                    userId: this.user.id,
                });
                return await this._handleStandaloneRoutine(args);
            }
        } catch (error) {
            const errorMsg = `Failed to start routine ${args.routineId}: ${(error as Error).message}`;
            logger.error(errorMsg, error);
            return { isError: true, content: [{ type: "text", text: errorMsg }] };
        }
    }

    /**
     * Detects parent swarm context for the current user
     * Uses the same pattern as spawnSwarm for consistency
     */
    private async getParentSwarmContext(): Promise<string | null> {
        const activeRecords = activeSwarmRegistry.getOrderedRecords();
        for (const record of activeRecords) {
            const swarmInstance = activeSwarmRegistry.get(record.id);
            if (swarmInstance && swarmInstance.getAssociatedUserId() === this.user.id) {
                return record.id;
            }
        }
        return null;
    }

    /**
     * Handles swarm-integrated routine execution
     * Inherits context and resources from parent swarm
     */
    private async _handleSwarmIntegratedRoutine(
        args: Extract<RunRoutineParams, { action: "start" }>,
        parentSwarmId: string,
    ): Promise<ToolResponse> {
        const { routineId, inputs = {}, mode = "sync", priority = "normal" } = args;
        const runId = generatePK().toString();
        
        // Allocate resources from parent swarm
        const allocation = await this.allocateFromParentSwarm(parentSwarmId);
        
        // Convert priority to RunTriggeredFrom
        const runFrom = RunTriggeredFrom.Api;
        
        // Lazy imports
        const { processRun } = await import("../../tasks/run/queue.js");
        const { QueueService } = await import("../../tasks/queues.js");
        
        // Create RunTask with parent context
        const runTaskForQueue: Omit<RunTask, "status"> = {
            id: generatePK().toString(),
            type: QueueTaskType.RUN_START,
            context: {
                swarmId: parentSwarmId,  // Inherit parent swarm ID
                parentSwarmId,           // Track parent relationship
                userData: this.user,
                timestamp: new Date(),
            },
            input: {
                runId,
                resourceVersionId: routineId,
                config: {
                    botConfig: {},
                    decisionConfig: {
                        inputGeneration: InputGenerationStrategy.Auto,
                        pathSelection: PathSelectionStrategy.AutoPickFirst,
                        subroutineExecution: SubroutineExecutionStrategy.Auto,
                    },
                    isPrivate: false,
                    limits: {},
                    loopConfig: {},
                    isTimeSensitive: priority === "high",
                    ...(inputs && Object.keys(inputs).length > 0 && { inputs }),
                },
                formValues: inputs,
                isNewRun: true,
                runFrom,
                startedById: this.user.id,
                status: "Scheduled",
            },
            allocation, // Resources allocated from parent
        };
        
        const result = await processRun(runTaskForQueue, QueueService.get());
        
        if (result.success) {
            const actionText = mode === "sync" ? "started" : "scheduled";
            return {
                isError: false,
                content: [{
                    type: "text",
                    text: `Routine ${runId} ${actionText} in swarm-integrated mode. Parent: ${parentSwarmId}. Allocated resources: ${allocation.maxCredits} credits, ${allocation.maxDurationMs}ms duration.`,
                }],
            };
        } else {
            throw new Error("Failed to queue swarm-integrated run task");
        }
    }

    /**
     * Allocates resources from parent swarm for child routine
     * Uses conservative percentages to prevent resource exhaustion
     */
    private async allocateFromParentSwarm(parentSwarmId: string): Promise<CoreResourceAllocation> {
        // Resource allocation percentages for child routines
        const CHILD_CREDITS_DIVISOR = 5n; // 20% of parent (1/5)
        const CHILD_DURATION_RATIO = 0.5; // 50% of parent time
        const CHILD_MEMORY_RATIO = 0.3; // 30% of parent memory
        const CHILD_MAX_CONCURRENT_STEPS = 3;
        
        // Get parent swarm from registry to verify it exists
        const parentSwarm = activeSwarmRegistry.get(parentSwarmId);
        if (!parentSwarm) {
            throw new Error("Parent swarm not found");
        }
        
        // For now, use conservative default allocations
        // In a full implementation, this would query the SwarmContextManager
        // or the parent swarm's actual resource allocation
        const defaultParentAllocation = {
            maxCredits: "10000",
            maxDurationMs: 3600000, // 1 hour
            maxMemoryMB: 1024,
            maxConcurrentSteps: 10,
        };
        
        return {
            maxCredits: (BigInt(defaultParentAllocation.maxCredits) / CHILD_CREDITS_DIVISOR).toString(),
            maxDurationMs: Math.floor(defaultParentAllocation.maxDurationMs * CHILD_DURATION_RATIO),
            maxMemoryMB: Math.floor(defaultParentAllocation.maxMemoryMB * CHILD_MEMORY_RATIO),
            maxConcurrentSteps: Math.min(CHILD_MAX_CONCURRENT_STEPS, defaultParentAllocation.maxConcurrentSteps),
        };
    }

    /**
     * Handles standalone routine execution (existing behavior)
     */
    private async _handleStandaloneRoutine(args: Extract<RunRoutineParams, { action: "start" }>): Promise<ToolResponse> {
        const { routineId, inputs = {}, mode = "sync", priority = "normal" } = args;
        
        // Generate a unique run ID
        const runId = generatePK().toString();

        // Convert priority to RunTriggeredFrom for queue prioritization
        // Use "Api" as the trigger source since this comes from MCP
        const runFrom = RunTriggeredFrom.Api;

        // Add the run to the queue
        // Lazy import to avoid circular dependency
        const { processRun } = await import("../../tasks/run/queue.js");
        const { QueueService } = await import("../../tasks/queues.js");
        
        // Create proper RunTask structure
        const runTaskForQueue: Omit<RunTask, "status"> = {
            id: generatePK().toString(), // Required for BaseTaskData
            type: QueueTaskType.RUN_START,
            context: {
                swarmId: `swarm-${runId}`,
                userData: this.user,
                timestamp: new Date(),
            },
            input: {
                runId,
                resourceVersionId: routineId,
                config: {
                    botConfig: {},
                    decisionConfig: {
                        inputGeneration: InputGenerationStrategy.Auto,
                        pathSelection: PathSelectionStrategy.AutoPickFirst,
                        subroutineExecution: SubroutineExecutionStrategy.Auto,
                    },
                    isPrivate: false,
                    limits: {},
                    loopConfig: {},
                    isTimeSensitive: priority === "high",
                    // Set other config based on args
                    ...(inputs && Object.keys(inputs).length > 0 && { inputs }),
                },
                formValues: inputs,
                isNewRun: true,
                runFrom,
                startedById: this.user.id,
                status: "Scheduled",
            },
            allocation: {
                maxCredits: "10000",
                maxDurationMs: 3600000, // 1 hour
                maxMemoryMB: 512,
                maxConcurrentSteps: 5,
            },
        };
        
        const result = await processRun(runTaskForQueue, QueueService.get());

        if (result.success) {
            if (mode === "sync") {
                return {
                    isError: false,
                    content: [{
                        type: "text",
                        text: `Run started successfully. Run ID: ${runId}. Mode: ${mode}. Check status for completion.`,
                    }],
                };
            } else {
                return {
                    isError: false,
                    content: [{
                        type: "text",
                        text: `Run scheduled successfully. Run ID: ${runId}. Mode: ${mode}.`,
                    }],
                };
            }
        } else {
            throw new Error("Failed to queue run task");
        }
    }

    /**
     * Handles managing an existing routine run (pause, resume, cancel, status).
     */
    private async _handleRunRoutineManage(args: Extract<RunRoutineParams, { action: "pause" | "resume" | "cancel" | "status" }>): Promise<ToolResponse> {
        const { runId, action } = args;

        try {
            // Lazy import to avoid circular dependency
            const { activeRunRegistry } = await import("../../tasks/run/process.js");
            const stateMachine = activeRunRegistry.get(runId);

            if (!stateMachine) {
                return {
                    isError: true,
                    content: [{ type: "text", text: `Run ${runId} not found in active runs registry (not implemented)` }],
                };
            }

            switch (action) {
                case "status": {
                    const status = stateMachine.getState();
                    return {
                        isError: false,
                        content: [{
                            type: "text",
                            text: `Run ${runId} current status: ${status}`,
                        }],
                    };
                }

                case "pause": {
                    try {
                        await stateMachine.pause();
                        return {
                            isError: false,
                            content: [{
                                type: "text",
                                text: `Run ${runId} pause request initiated successfully`,
                            }],
                        };
                    } catch (error) {
                        return {
                            isError: true,
                            content: [{
                                type: "text",
                                text: `Failed to pause run ${runId}: ${error instanceof Error ? error.message : String(error)}`,
                            }],
                        };
                    }
                }

                case "resume": {
                    // TODO: Import and use changeRunTaskStatus when implemented
                    // For resume, we need to requeue the run task
                    try {
                        // const result = await changeRunTaskStatus(runId, "Scheduled", this.user.id);
                        const result = { success: false }; // Placeholder
                        if (result.success) {
                            return {
                                isError: false,
                                content: [{
                                    type: "text",
                                    text: `Run ${runId} resume request initiated successfully`,
                                }],
                            };
                        } else {
                            throw new Error("Failed to change task status (not implemented)");
                        }
                    } catch (error) {
                        return {
                            isError: true,
                            content: [{
                                type: "text",
                                text: `Failed to resume run ${runId}: ${(error as Error).message}`,
                            }],
                        };
                    }
                }

                case "cancel": {
                    try {
                        await stateMachine.stop("Cancelled by user via MCP tool");
                        return {
                            isError: false,
                            content: [{
                                type: "text",
                                text: `Run ${runId} cancellation request initiated successfully`,
                            }],
                        };
                    } catch (error) {
                        return {
                            isError: true,
                            content: [{
                                type: "text",
                                text: `Failed to cancel run ${runId}: ${error instanceof Error ? error.message : String(error)}`,
                            }],
                        };
                    }
                }

                default:
                    return {
                        isError: true,
                        content: [{ type: "text", text: `Unsupported action: ${action}` }],
                    };
            }
        } catch (error) {
            const errorMsg = `Error managing run ${runId} with action ${action}: ${(error as Error).message}`;
            logger.error(errorMsg, error);
            return { isError: true, content: [{ type: "text", text: errorMsg }] };
        }
    }

    /**
     * Handler for the SpawnSwarm MCP tool.
     * 
     * @param args - Parameters for spawning a swarm.
     * @returns A ToolResponse indicating the result of the spawnSwarm operation.
     */
    async spawnSwarm(args: SpawnSwarmParams, parentSwarmId?: string): Promise<ToolResponse> {
        logger.info(`spawnSwarm called with args: ${JSON.stringify(args)}`);

        try {
            // Get parent swarm ID from active swarm registry if not provided
            // This is a temporary solution - ideally we'd get this from MCP context
            let actualParentSwarmId = parentSwarmId;
            if (!actualParentSwarmId) {
                // Try to find an active swarm for this user
                const activeRecords = activeSwarmRegistry.getOrderedRecords();
                for (const record of activeRecords) {
                    const swarmInstance = activeSwarmRegistry.get(record.id);
                    if (swarmInstance) {
                        const associatedUserId = swarmInstance.getAssociatedUserId();
                        if (associatedUserId === this.user.id) {
                            actualParentSwarmId = record.id;
                            logger.info(`[spawnSwarm] Found parent swarm context: ${record.id}`);
                            break;
                        }
                    }
                }
                // Placeholder - no active swarm registry available
                actualParentSwarmId = `placeholder-swarm-${this.user.id}`;
            }

            if (!actualParentSwarmId) {
                return {
                    isError: true,
                    content: [{ type: "text", text: "spawnSwarm can only be called from within an active swarm context. No active swarm found for current user." }],
                };
            }

            // Generate unique child swarm ID
            const childSwarmId = generatePK().toString();

            // Calculate resource reservation
            const defaultReservation = { creditsPct: 50, messagesPct: 50, durationPct: 50 };
            const reservation = args.reservation || defaultReservation;

            // Get parent swarm to calculate actual resource amounts
            // Note: This would ideally come from the current context, but for now we'll use default values
            const defaultParentResources = {
                maxCredits: 1000,
                maxTokens: 100000,
                maxTime: 3600000, // 1 hour in ms
            };

            const childResources = {
                maxCredits: Math.floor(defaultParentResources.maxCredits * (reservation.creditsPct / 100)),
                maxTokens: Math.floor(defaultParentResources.maxTokens * (reservation.messagesPct || reservation.creditsPct) / 100),
                maxTime: Math.floor(defaultParentResources.maxTime * (reservation.durationPct || reservation.creditsPct) / 100),
                tools: [], // Child inherits available tools
            };

            if (isSpawnSwarmSimple(args)) {
                // Handle simple child swarm
                const childSwarmTask: Omit<SwarmExecutionTask, "status"> = {
                    id: generatePK().toString(),
                    type: QueueTaskType.SWARM_EXECUTION,
                    context: {
                        swarmId: childSwarmId,
                        userData: this.user,
                        parentSwarmId: actualParentSwarmId,
                        timestamp: new Date(),
                    },
                    input: {
                        swarmId: childSwarmId,
                        goal: args.goal,
                        teamConfiguration: undefined,
                        availableTools: undefined,
                        executionConfig: {
                            model: "claude-3-haiku-20240307", // Default model for child swarms
                            temperature: 0.7,
                            parallelExecutionLimit: 2,
                        },
                        userData: this.user,
                    },
                    allocation: {
                        maxCredits: childResources.maxCredits.toString(),
                        maxDurationMs: childResources.maxTime,
                        maxMemoryMB: 256,
                        maxConcurrentSteps: 3,
                    },
                };

                // Use queue system to spawn child swarm
                const { processNewSwarmExecution } = await import("../../tasks/swarm/queue.js");
                const { QueueService } = await import("../../tasks/queues.js");
                const result = await processNewSwarmExecution(childSwarmTask, QueueService.get());

                return {
                    isError: false,
                    content: [{
                        type: "text",
                        text: `Child swarm ${childSwarmId} spawned successfully with goal: "${args.goal}". Leader: ${args.swarmLeader}. Reserved resources: ${JSON.stringify(reservation)}.`,
                    }],
                };

            } else {
                // Handle rich child swarm with team
                const childSwarmTask: Omit<SwarmExecutionTask, "status"> = {
                    id: generatePK().toString(),
                    type: QueueTaskType.SWARM_EXECUTION,
                    context: {
                        swarmId: childSwarmId,
                        userData: this.user,
                        parentSwarmId: actualParentSwarmId,
                        timestamp: new Date(),
                    },
                    input: {
                        swarmId: childSwarmId,
                        goal: args.goal,
                        teamConfiguration: undefined,
                        availableTools: undefined,
                        executionConfig: {
                            model: "claude-3-sonnet-20240229", // Rich swarms get better model
                            temperature: 0.5,
                            parallelExecutionLimit: 5,
                        },
                        userData: this.user,
                    },
                    allocation: {
                        maxCredits: childResources.maxCredits.toString(),
                        maxDurationMs: childResources.maxTime,
                        maxMemoryMB: 512,
                        maxConcurrentSteps: 5,
                    },
                };

                // Use queue system to spawn child swarm
                const { processNewSwarmExecution } = await import("../../tasks/swarm/queue.js");
                const { QueueService } = await import("../../tasks/queues.js");
                const result = await processNewSwarmExecution(childSwarmTask, QueueService.get());

                // TODO: Handle additional rich swarm features
                // - Team assignment (args.teamId)
                // - Initial subtasks (args.subtasks)
                // - Event subscriptions (args.eventSubscriptions)
                // - Policy settings (args.policy)

                return {
                    isError: false,
                    content: [{
                        type: "text",
                        text: `Rich child swarm ${childSwarmId} spawned successfully with team ${args.teamId} and goal: "${args.goal}". Reserved resources: ${JSON.stringify(reservation)}.`,
                    }],
                };
            }

        } catch (error) {
            logger.error("Error in spawnSwarm:", error);
            return {
                isError: true,
                content: [{
                    type: "text",
                    text: `Failed to spawn child swarm: ${error instanceof Error ? error.message : "Unknown error"}`,
                }],
            };
        }
    }

    // -- Mapping layer: convert MCP shapes to GraphQL action-helper inputs --
    _mapFindToInput(resourceType: string, filters: Record<string, unknown>): Record<string, unknown> {
        logger.info(`Mapping find input for resourceType: ${resourceType} with filters: ${JSON.stringify(filters)}`);

        // Spread filters into a new object (read helper function will validate/filter out any invalid filters)
        const mappedFilters = { ...filters };

        // Add any other direct mappings from 'filters' to 'mappedFilters' here.

        // Some resource types need additional filters to ensure we're getting the right resources.
        switch (resourceType) {
            // ResourceType direct mappings
            case ResourceType.Note:
                mappedFilters.rootResourceType = ResourceType.Note;
                break;
            case ResourceType.Project:
                mappedFilters.rootResourceType = ResourceType.Project;
                break;
            case ResourceType.Standard:
                mappedFilters.rootResourceType = ResourceType.Standard;
                break;
            case ResourceType.Routine:
                mappedFilters.rootResourceType = ResourceType.Routine;
                break;
            case ResourceType.Api: // Assuming ResourceType.Api is a direct root type
                mappedFilters.rootResourceType = ResourceType.Api;
                break;
            case ResourceType.Code:
                mappedFilters.rootResourceType = ResourceType.Code;
                break;
            // ResourceSubType mappings
            // Standard SubTypes
            case ResourceSubType.StandardDataStructure:
                mappedFilters.rootResourceType = ResourceType.Standard;
                mappedFilters.resourceSubType = ResourceSubType.StandardDataStructure;
                break;
            case ResourceSubType.StandardPrompt:
                mappedFilters.rootResourceType = ResourceType.Standard;
                mappedFilters.resourceSubType = ResourceSubType.StandardPrompt;
                break;
            // Routine SubTypes
            case ResourceSubType.RoutineInternalAction:
            case ResourceSubType.RoutineApi:
            case ResourceSubType.RoutineCode:
            case ResourceSubType.RoutineData:
            case ResourceSubType.RoutineGenerate:
            case ResourceSubType.RoutineInformational:
            case ResourceSubType.RoutineMultiStep:
            case ResourceSubType.RoutineSmartContract:
            case ResourceSubType.RoutineWeb:
                mappedFilters.rootResourceType = ResourceType.Routine;
                mappedFilters.resourceSubType = resourceType as ResourceSubType; // Assigns the specific routine subtype
                break;
            // Code SubTypes
            case ResourceSubType.CodeDataConverter:
            case ResourceSubType.CodeSmartContract:
                mappedFilters.rootResourceType = ResourceType.Code;
                mappedFilters.resourceSubType = resourceType as ResourceSubType; // Assigns the specific code subtype
                break;
        }
        return mappedFilters;
    }

    /**
     * Helper to create ResourceVersionCreateInput for ResourceVersion-based types
     */
    private _createResourceVersionInput(
        resourceType: ResourceType,
        resourceSubType: ResourceSubType | undefined,
        translationsCreate: Array<{ id: string; language: string; name: string; description?: string; details?: string; instructions?: string }>,
        attrs: { isPrivate?: boolean; versionLabel?: string; isComplete?: boolean; config?: any; rootCreate: any },
    ): ResourceVersionCreateInput {
        const input: ResourceVersionCreateInput = {
            id: generatePK().toString(),
            isPrivate: attrs.isPrivate ?? true,
            versionLabel: attrs.versionLabel ?? "1.0.0",
            isComplete: attrs.isComplete ?? false,
            translationsCreate,
            config: attrs.config,
            rootCreate: {
                id: generatePK().toString(),
                isPrivate: attrs.rootCreate.isPrivate ?? true,
                resourceType,
                tagsConnect: attrs.rootCreate.tagsConnect,
            },
        };

        // Only add resourceSubType if it's defined (for Routine and Standard subtypes)
        if (resourceSubType) {
            input.resourceSubType = resourceSubType;
        }

        return input;
    }

    /**
     * Maps MCP 'add' operation attributes to the appropriate GraphQL create input.
     * @param resourceType The type of resource (e.g., "Note", "Project", "Team").
     * @param attrs The attributes for the new resource.
     * @returns The GraphQL create input (ResourceVersionCreateInput, TeamCreateInput, or BotCreateInput).
     */
    _mapAddToInput(resourceType: string, attrs: ResourceAddAttributes): ResourceVersionCreateInput | TeamCreateInput | BotCreateInput {
        // Handle non-ResourceVersion types first
        switch (resourceType) {
            case "Team": {
                if (!isTeamAddAttributes(attrs)) {
                    throw new Error("Invalid attributes for Team");
                }
                const teamId = generatePK().toString();
                return {
                    id: teamId,
                    isPrivate: attrs.isPrivate ?? false,
                    handle: attrs.handle,
                    tagsConnect: attrs.tagsConnect,
                    configCreate: attrs.config ? { __typename: "TeamConfig" as const, ...attrs.config } : undefined,
                    memberInvitesCreate: attrs.memberInvitesCreate?.map(invite => ({
                        id: generatePK().toString(),
                        message: invite.message,
                        teamConnect: teamId, // Connect to the team being created
                        userConnect: invite.userConnect,
                        willBeAdmin: invite.willBeAdmin ?? false,
                    })),
                } as TeamCreateInput;
            }

            case "Bot": {
                if (!isBotAddAttributes(attrs)) {
                    throw new Error("Invalid attributes for Bot");
                }
                // Bot requires a name and uses UserTranslationCreateInput
                return {
                    id: generatePK().toString(),
                    name: attrs.handle || "Bot", // Use handle as name, or default to "Bot"
                    handle: attrs.handle,
                    isPrivate: attrs.isPrivate ?? false,
                    isBotDepictingPerson: false, // Default to false
                    botSettings: attrs.config || {}, // botSettings is required
                    translationsCreate: [], // Bots typically don't need translations at creation
                } as BotCreateInput;
            }

            // ResourceVersion-based types
            case "Note": {
                if (!isNoteAddAttributes(attrs)) {
                    throw new Error("Invalid attributes for Note: missing required properties 'name' or 'content'");
                }
                return this._createResourceVersionInput(
                    ResourceType.Note,
                    undefined, // Note doesn't have a resourceSubType
                    [{
                        id: generatePK().toString(),
                        language: DEFAULT_LANGUAGE,
                        name: attrs.name,
                        description: attrs.content,
                    }],
                    {
                        isPrivate: true,
                        versionLabel: "0.0.1",
                        rootCreate: {
                            isPrivate: true,
                            tagsConnect: attrs.tagsConnect,
                        },
                    },
                );
            }

            case "Project": {
                if (!isProjectAddAttributes(attrs)) {
                    throw new Error("Invalid attributes for Project: missing required property 'name'");
                }
                return this._createResourceVersionInput(
                    ResourceType.Project,
                    undefined, // Project doesn't have a resourceSubType
                    [{
                        id: generatePK().toString(),
                        language: DEFAULT_LANGUAGE,
                        name: attrs.name,
                        description: "", // Add empty description since it's optional but safer to include
                    }],
                    {
                        isPrivate: attrs.isPrivate ?? true,
                        versionLabel: "1.0.0",
                        config: attrs.config,
                        rootCreate: {
                            isPrivate: attrs.isPrivate ?? true,
                            tagsConnect: attrs.tagsConnect,
                            handle: attrs.handle,
                        },
                    },
                );
            }

            // Routine subtypes
            case "RoutineMultiStep":
            case "RoutineApi":
            case "RoutineCode":
            case "RoutineData":
            case "RoutineGenerate":
            case "RoutineInformational":
            case "RoutineSmartContract":
            case "RoutineWeb": {
                const subType = resourceType as ResourceSubType;
                if (!isRoutineAddAttributes(attrs, subType)) {
                    throw new Error(`Invalid attributes for ${resourceType}`);
                }
                const routineAttrs = attrs as RoutineMultiStepAddAttributes; // All routine types have similar structure
                return this._createResourceVersionInput(
                    ResourceType.Routine,
                    subType,
                    [], // Routines typically don't have translations at creation
                    {
                        isPrivate: routineAttrs.isPrivate ?? true,
                        versionLabel: routineAttrs.versionLabel,
                        isComplete: routineAttrs.isComplete,
                        config: routineAttrs.config,
                        rootCreate: routineAttrs.rootCreate,
                    },
                );
            }

            // Standard subtypes
            case "StandardDataStructure":
            case "StandardPrompt": {
                const subType = resourceType as ResourceSubType;
                if (!isStandardAddAttributes(attrs, subType)) {
                    throw new Error(`Invalid attributes for ${resourceType}`);
                }
                const standardAttrs = attrs as StandardDataStructureAddAttributes; // All standard types have similar structure
                return this._createResourceVersionInput(
                    ResourceType.Standard,
                    subType,
                    [], // Standards typically don't have translations at creation
                    {
                        isPrivate: standardAttrs.isPrivate ?? true,
                        versionLabel: standardAttrs.versionLabel,
                        isComplete: standardAttrs.isComplete,
                        config: standardAttrs.config,
                        rootCreate: standardAttrs.rootCreate,
                    },
                );
            }

            default:
                throw new Error(`Add for resource type '${resourceType}' not implemented`);
        }
    }

    /**
     * Helper to create ResourceVersionUpdateInput for ResourceVersion-based types
     */
    private _createResourceVersionUpdateInput(
        resourceId: string,
        translationsUpdate?: Array<{ id: string; language: string; name?: string; description?: string; details?: string; instructions?: string }>,
        attrs?: { isPrivate?: boolean; versionLabel?: string; isComplete?: boolean; config?: any; rootUpdate?: any },
    ): ResourceVersionUpdateInput {
        const input: ResourceVersionUpdateInput = {
            id: resourceId,
        };

        // Only add fields if they're defined
        if (attrs?.isPrivate !== undefined) input.isPrivate = attrs.isPrivate;
        if (attrs?.isComplete !== undefined) input.isComplete = attrs.isComplete;
        if (attrs?.config !== undefined) input.config = attrs.config;
        if (translationsUpdate && translationsUpdate.length > 0) {
            input.translationsUpdate = translationsUpdate;
        }
        if (attrs?.rootUpdate) {
            input.rootUpdate = {
                id: attrs.rootUpdate.id,
                isPrivate: attrs.rootUpdate.isPrivate,
                tagsConnect: attrs.rootUpdate.tagsConnect,
                tagsDisconnect: attrs.rootUpdate.tagsDisconnect,
            };
        }

        return input;
    }

    /**
     * Maps MCP 'update' operation attributes to the appropriate GraphQL update input.
     * @param resourceId The ID of the resource to update.
     * @param resourceType The type of resource (e.g., "Note", "Project", "Team").
     * @param attrs The attributes to update.
     * @returns The GraphQL update input (ResourceVersionUpdateInput, TeamUpdateInput, or BotUpdateInput).
     */
    _mapUpdateToInput(resourceId: string, resourceType: string, attrs: ResourceUpdateAttributes): ResourceVersionUpdateInput | TeamUpdateInput | BotUpdateInput {
        // Handle non-ResourceVersion types first
        switch (resourceType) {
            case "Team": {
                if (!isTeamUpdateAttributes(attrs)) {
                    throw new Error("Invalid update attributes for Team");
                }
                const input: TeamUpdateInput = {
                    id: resourceId,
                };

                // Only add fields if they're defined
                if (attrs.isPrivate !== undefined) input.isPrivate = attrs.isPrivate;
                if (attrs.handle !== undefined) input.handle = attrs.handle;
                if (attrs.tagsConnect !== undefined) input.tagsConnect = attrs.tagsConnect;
                if (attrs.tagsDisconnect !== undefined) input.tagsDisconnect = attrs.tagsDisconnect;
                if (attrs.config !== undefined) input.config = attrs.config;

                // Handle member operations
                if (attrs.memberInvitesCreate) {
                    input.memberInvitesCreate = attrs.memberInvitesCreate.map(invite => ({
                        id: generatePK().toString(),
                        message: invite.message,
                        teamConnect: resourceId, // The team being updated
                        userConnect: invite.userConnect,
                        willBeAdmin: invite.willBeAdmin ?? false,
                    }));
                }
                if (attrs.memberInvitesDelete) {
                    input.memberInvitesDelete = attrs.memberInvitesDelete;
                }
                if (attrs.membersDelete) {
                    input.membersDelete = attrs.membersDelete;
                }

                return input;
            }

            case "Bot": {
                if (!isBotUpdateAttributes(attrs)) {
                    throw new Error("Invalid update attributes for Bot");
                }
                const input: BotUpdateInput = {
                    id: resourceId,
                };

                // Only add fields if they're defined
                if (attrs.isPrivate !== undefined) input.isPrivate = attrs.isPrivate;
                if (attrs.handle !== undefined) input.handle = attrs.handle;
                if (attrs.config !== undefined) input.botSettings = attrs.config;

                return input;
            }

            // ResourceVersion-based types
            case "Note": {
                if (!isNoteUpdateAttributes(attrs)) {
                    throw new Error("Invalid update attributes for Note");
                }

                const translationsUpdate: Array<{ id: string; language: string; name?: string; description?: string }> = [];
                if (attrs.name !== undefined || attrs.content !== undefined) {
                    translationsUpdate.push({
                        id: generatePK().toString(),
                        language: DEFAULT_LANGUAGE,
                        name: attrs.name,
                        description: attrs.content,
                    });
                }

                return this._createResourceVersionUpdateInput(
                    resourceId,
                    translationsUpdate,
                    {
                        rootUpdate: attrs.tagsConnect || attrs.tagsDisconnect ? {
                            id: generatePK().toString(), // This should be the root ID, but we don't have it
                            tagsConnect: attrs.tagsConnect,
                            tagsDisconnect: attrs.tagsDisconnect,
                        } : undefined,
                    },
                );
            }

            case "Project": {
                if (!isProjectUpdateAttributes(attrs)) {
                    throw new Error("Invalid update attributes for Project");
                }

                const translationsUpdate: Array<{ id: string; language: string; name?: string }> = [];
                if (attrs.name !== undefined) {
                    translationsUpdate.push({
                        id: generatePK().toString(),
                        language: DEFAULT_LANGUAGE,
                        name: attrs.name,
                    });
                }

                return this._createResourceVersionUpdateInput(
                    resourceId,
                    translationsUpdate,
                    {
                        isPrivate: attrs.isPrivate,
                        config: attrs.config,
                        rootUpdate: (attrs.isPrivate !== undefined || attrs.handle !== undefined || attrs.tagsConnect || attrs.tagsDisconnect) ? {
                            id: generatePK().toString(), // This should be the root ID
                            isPrivate: attrs.isPrivate,
                            handle: attrs.handle,
                            tagsConnect: attrs.tagsConnect,
                            tagsDisconnect: attrs.tagsDisconnect,
                        } : undefined,
                    },
                );
            }

            // Routine subtypes
            case "RoutineMultiStep":
            case "RoutineApi":
            case "RoutineCode":
            case "RoutineData":
            case "RoutineGenerate":
            case "RoutineInformational":
            case "RoutineSmartContract":
            case "RoutineWeb": {
                if (!isRoutineUpdateAttributes(attrs)) {
                    throw new Error(`Invalid update attributes for ${resourceType}`);
                }
                const routineAttrs = attrs as RoutineMultiStepUpdateAttributes; // All routine types have similar structure

                return this._createResourceVersionUpdateInput(
                    resourceId,
                    [], // Routines typically don't have translations to update
                    {
                        isPrivate: routineAttrs.isPrivate,
                        versionLabel: routineAttrs.versionLabel,
                        isComplete: routineAttrs.isComplete,
                        config: routineAttrs.config,
                        rootUpdate: routineAttrs.rootUpdate,
                    },
                );
            }

            // Standard subtypes
            case "StandardDataStructure":
            case "StandardPrompt": {
                if (!isStandardUpdateAttributes(attrs)) {
                    throw new Error(`Invalid update attributes for ${resourceType}`);
                }
                const standardAttrs = attrs as StandardDataStructureUpdateAttributes; // All standard types have similar structure

                return this._createResourceVersionUpdateInput(
                    resourceId,
                    [], // Standards typically don't have translations to update
                    {
                        isPrivate: standardAttrs.isPrivate,
                        versionLabel: standardAttrs.versionLabel,
                        isComplete: standardAttrs.isComplete,
                        config: standardAttrs.config,
                        rootUpdate: standardAttrs.rootUpdate,
                    },
                );
            }

            default:
                throw new Error(`Update for resource type '${resourceType}' not implemented`);
        }
    }
}

/**
 * Holds all swarm-specific MCP tools.
 */
export class SwarmTools {
    private conversationStore: ConversationStateStore;

    constructor(conversationStore: ConversationStateStore) {
        this.conversationStore = conversationStore;
    }

    /**
     * Updates the shared state (sub-tasks and/or scratchpad) for a swarm.
     *
     * @param conversationId The ID of the swarm's conversation.
     * @param args The operations to perform on the shared state.
     * @param user The user session for authorization of team updates.
     * @returns An object indicating success or failure, and the updated state.
     */
    async updateSwarmSharedState(
        conversationId: string,
        args: UpdateSwarmSharedStateParams,
        user?: SessionUser,
    ): Promise<{
        success: boolean;
        message?: string;
        updatedSubTasks?: SwarmSubTask[];
        updatedSharedScratchpad?: Record<string, unknown>;
        updatedTeamConfig?: TeamConfigObject;
        error?: string; // For error codes/types
    }> {
        logger.info(
            `SwarmTools.updateSwarmSharedState called for convo ${conversationId}`,
            args,
        );
        const convoState = await this.conversationStore.get(conversationId);

        if (!convoState) {
            const errorMsg = `Conversation state not found for conversation ${conversationId}.`;
            logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_STATE_NOT_FOUND" };
        }

        if (!convoState.config) {
            const errorMsg = `Conversation config not found for conversation ${conversationId}.`;
            logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_CONFIG_NOT_FOUND" };
        }

        // Deep clone to avoid mutating the cached state directly before persistence logic in SwarmStateMachine
        let currentSubTasks: SwarmSubTask[] = structuredClone(convoState.config.subtasks || []);
        const currentScratchpad: Record<string, unknown> = {};

        // Convert BlackboardItem[] to Record<string, unknown> for backward compatibility
        if (convoState.config.blackboard) {
            for (const item of convoState.config.blackboard) {
                currentScratchpad[item.id] = item.value;
            }
        }

        let updatedTeamConfig: TeamConfigObject | undefined = undefined;

        // Handle team config updates
        if (args.teamConfig && convoState.config.teamId) {
            if (!user) {
                const errorMsg = "User session required for team config updates.";
                logger.error(errorMsg);
                return { success: false, message: errorMsg, error: "SESSION_USER_REQUIRED" };
            }

            try {
                // Create a mock request object for the updateOneHelper
                const mockReq: Parameters<typeof RequestService.assertRequestFrom>[0] = {
                    session: {
                        fromSafeOrigin: true,
                        isLoggedIn: true,
                        languages: user.languages ?? [DEFAULT_LANGUAGE],
                        userId: user.id,
                        users: [user],
                    },
                };

                // Fetch current team config to merge with updates
                const currentTeam = await readOneHelper({
                    info: { id: true, config: true } as any,
                    input: { id: convoState.config.teamId },
                    objectType: "Team",
                    req: mockReq,
                });

                if (!currentTeam || !currentTeam.config) {
                    const errorMsg = `Team ${convoState.config.teamId} not found or user lacks access.`;
                    logger.error(errorMsg);
                    return { success: false, message: errorMsg, error: "TEAM_NOT_FOUND_OR_ACCESS_DENIED" };
                }

                // Parse current team config - ensure it has __version property
                const teamConfigData = (currentTeam.config && typeof currentTeam.config === "object") 
                    ? currentTeam.config as Record<string, unknown>
                    : {};
                const configWithVersion = {
                    __version: "1.0",
                    ...teamConfigData,
                };
                const currentTeamConfig = TeamConfig.parse({ config: configWithVersion }, logger, { useFallbacks: true });

                // Apply updates to the team config
                if (args.teamConfig.structure) {
                    if (!currentTeamConfig.structure) {
                        currentTeamConfig.structure = {
                            type: args.teamConfig.structure.type || "MOISE+",
                            version: args.teamConfig.structure.version || "1.0",
                            content: args.teamConfig.structure.content || "",
                        };
                    } else {
                        if (args.teamConfig.structure.type !== undefined) {
                            currentTeamConfig.structure.type = args.teamConfig.structure.type;
                        }
                        if (args.teamConfig.structure.version !== undefined) {
                            currentTeamConfig.structure.version = args.teamConfig.structure.version;
                        }
                        if (args.teamConfig.structure.content !== undefined) {
                            currentTeamConfig.structure.content = args.teamConfig.structure.content;
                        }
                    }
                }

                // Update the team in the database
                await updateOneHelper({
                    info: {} as any,
                    input: {
                        id: convoState.config.teamId,
                        config: currentTeamConfig.export(),
                    },
                    objectType: "Team",
                    req: mockReq,
                });

                updatedTeamConfig = currentTeamConfig.export();
                logger.info(`Updated team config for team ${convoState.config.teamId}`);

                // Update the conversation state with the new team config
                try {
                    await this.conversationStore.updateTeamConfig(conversationId, updatedTeamConfig);
                    logger.debug(`Refreshed team config in conversation state for ${conversationId}`);
                } catch (updateError) {
                    logger.warn(`Failed to update team config in conversation state cache for ${conversationId}:`, updateError);
                    // Continue anyway - the team config was updated in the database and will be available on next fetch
                }

            } catch (error) {
                const errorMsg = `Failed to update team config: ${error instanceof Error ? error.message : "Unknown error"}`;
                logger.error(errorMsg, { error });
                return { success: false, message: errorMsg, error: "TEAM_CONFIG_UPDATE_FAILED" };
            }
        } else if (args.teamConfig && !convoState.config.teamId) {
            const errorMsg = "Cannot update team config: no team assigned to this swarm.";
            logger.warn(errorMsg);
            return { success: false, message: errorMsg, error: "NO_TEAM_ASSIGNED" };
        }

        // Apply subTasks operations
        if (args.subTasks) {
            const now = new Date().toISOString();
            // Add/Update
            if (args.subTasks.set) {
                for (const taskToSet of args.subTasks.set) {
                    const taskIndex = currentSubTasks.findIndex(t => t.id === taskToSet.id);
                    if (taskIndex !== -1) {
                        // Update existing task
                        currentSubTasks[taskIndex] = {
                            ...currentSubTasks[taskIndex],
                            ...taskToSet,
                            created_at: currentSubTasks[taskIndex].created_at, // Preserve original creation time
                        };
                    } else {
                        // Add new task
                        currentSubTasks.push({
                            ...taskToSet,
                            created_at: taskToSet.created_at || now, // Use provided created_at or current time
                        });
                    }
                }
            }
            // Remove
            if (args.subTasks.delete) {
                currentSubTasks = currentSubTasks.filter(
                    t => !args.subTasks?.delete?.includes(t.id),
                );
            }
        }

        // Apply blackboard operations
        if (args.blackboard) {
            // Add/Update
            if (args.blackboard.set) {
                for (const item of args.blackboard.set) {
                    currentScratchpad[item.id] = item.value;
                }
            }
            // Remove
            if (args.blackboard.delete) {
                for (const keyToRemove of args.blackboard.delete) {
                    delete currentScratchpad[keyToRemove];
                }
            }
        }

        logger.info(
            `Swarm state updated for convo ${conversationId}. SubTasks count: ${currentSubTasks.length}, Scratchpad keys: ${Object.keys(currentScratchpad).length}`,
        );

        return {
            success: true,
            updatedSubTasks: currentSubTasks,
            updatedSharedScratchpad: currentScratchpad,
            updatedTeamConfig,
        };
    }

    /**
     * Ends the swarm session. Called when the goal is complete or limits are reached.
     *
     * @param conversationId The ID of the swarm's conversation.
     * @param args The end swarm parameters (mode and reason).
     * @param user The user requesting to end the swarm (for authorization).
     * @returns An object indicating success or failure and final state information.
     */
    async endSwarm(
        conversationId: string,
        args: EndSwarmParams,
        user: SessionUser,
    ): Promise<{
        success: boolean;
        message?: string;
        finalState?: {
            endedAt: string;
            reason?: string;
            mode: "graceful" | "force";
            totalSubTasks?: number;
            completedSubTasks?: number;
            totalCreditsUsed?: string;
            totalToolCalls?: number;
        };
        error?: string;
    }> {
        logger.info(`SwarmTools.endSwarm called for convo ${conversationId} by user ${user.id}`, args);

        // Validate user authentication
        if (!user?.id) {
            const errorMsg = "User authentication required to end swarm.";
            logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "AUTHENTICATION_REQUIRED" };
        }

        // Get conversation state for authorization checks
        const convoState = await this.conversationStore.get(conversationId);
        if (!convoState) {
            const errorMsg = `Conversation state not found for conversation ${conversationId}.`;
            logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_STATE_NOT_FOUND" };
        }

        if (!convoState.config) {
            const errorMsg = `Conversation config not found for conversation ${conversationId}.`;
            logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_CONFIG_NOT_FOUND" };
        }

        // Try to get the active SwarmStateMachine instance for additional authorization
        const activeSwarmInstance = activeSwarmRegistry.get(conversationId);
        if (activeSwarmInstance) {
            // Check if user is authorized - they should be the swarm initiator or an admin
            const swarmInitiatorId = activeSwarmInstance.getAssociatedUserId?.();
            const isSwarmInitiator = swarmInitiatorId === user.id;
            let isAdmin = false;
            try {
                const adminId = await DbProvider.getAdminId();
                isAdmin = user.id === adminId;
            } catch (error) {
                logger.error("Failed to get admin ID for swarm end authorization:", error);
                // Continue without admin check if DbProvider.getAdminId() fails
            }

            if (!isSwarmInitiator && !isAdmin) {
                const errorMsg = `User ${user.id} is not authorized to end swarm ${conversationId}. Only the swarm initiator (${swarmInitiatorId}) or an admin can end a swarm.`;
                logger.warn(errorMsg);
                return {
                    success: false,
                    message: "You are not authorized to end this swarm. Only the swarm creator or an admin can end a swarm.",
                    error: "INSUFFICIENT_PERMISSIONS",
                };
            }

            logger.info(`User ${user.id} authorized to end swarm ${conversationId}. Delegating to SwarmStateMachine.`);
            try {
                const result = await activeSwarmInstance.stop(args.mode || "graceful", args.reason, user);
                return result;
            } catch (error) {
                const errorMessage = error instanceof Error ? error.message : "Unknown error in SwarmStateMachine.stop()";
                logger.error(`Error delegating to SwarmStateMachine.stop() for conversation ${conversationId}:`, error);
                return {
                    success: false,
                    message: `Failed to end swarm via SwarmStateMachine: ${errorMessage}`,
                    error: "SWARM_STATE_MACHINE_ERROR",
                };
            }
        }

        // Fallback: If no active swarm instance, we can't properly validate or end the swarm
        logger.warn(`No active SwarmStateMachine found for conversation ${conversationId}. Cannot end swarm properly without state machine.`);

        return {
            success: false,
            message: "Cannot end swarm: No active SwarmStateMachine instance found. The swarm may have already ended or not been properly started.",
            error: "NO_ACTIVE_SWARM_INSTANCE",
        };
    }
} 
