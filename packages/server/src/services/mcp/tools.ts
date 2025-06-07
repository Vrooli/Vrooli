import type { ResourceVersionCreateInput, SessionUser, SwarmSubTask } from "@vrooli/shared";
import { DEFAULT_LANGUAGE, DeleteType, McpToolName, ResourceSubType, ResourceType, TeamConfig, generatePK, type ChatMessage, type ChatMessageCreateInput, type MessageConfigObject, type TaskContextInfo, type TeamConfigObject } from "@vrooli/shared";
import fs from "fs";
import { fileURLToPath } from "node:url";
import path from "path";
import type { Logger } from "winston";
import { createOneHelper } from "../../actions/creates.js";
import { deleteOneHelper } from "../../actions/deletes.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import type { RequestService } from "../../auth/request.js";
import type { PartialApiInfo } from "../../builders/types.js";
import { DbProvider } from "../../db/provider.js";
import { activeSwarmRegistry } from "../../tasks/swarm/process.js";
import { type ConversationStateStore } from "../conversation/chatStore.js";
import defineToolSchema from "../schemas/DefineTool/schema.json" with { type: "json" };
import type { NoteAddAttributes } from "../types/resources.js";
import { type DefineToolParams, type EndSwarmParams, type Recipient, type ResourceManageParams, type RunRoutineParams, type SendMessageParams, type SpawnSwarmParams, type UpdateSwarmSharedStateParams } from "../types/tools.js";
import { type ToolResponse } from "./types.js";

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
    private readonly logger: Logger;
    private readonly req: Parameters<typeof RequestService.assertRequestFrom>[0];

    constructor(user: SessionUser, logger: Logger, req?: Parameters<typeof RequestService.assertRequestFrom>[0]) {
        this.user = user;
        this.logger = logger;
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
            this.logger.warn(errorMsg);
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
                toolDef = defineToolSchema as ToolSchema;
                break;
        }

        if (!toolDef?.inputSchema?.anyOf && !toolDef?.inputSchema?.oneOf) {
            const errorMsg = `Error: Could not retrieve base schema for ${toolName}.`;
            this.logger.error(errorMsg);
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
            this.logger.error(errorMsg);
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
                this.logger.warn(`DefineTool received an unhandled op: ${op} for variant ${variant}. Returning base schema for op.`);
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
        const schemaFilePath = path.join(currentDirPath, "..", "schemas", "DefineTool", variant, "find_filters.json");

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
                this.logger.warn(`Loaded schema from ${schemaFilePath} is not a valid object.`);
            }
        } catch (error) {
            this.logger.warn(`Filter schema file not found or failed to parse for ${variant} at ${schemaFilePath}: ${(error as Error).message}`);
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
        const schemaFilePath = path.join(currentDirPath, "schemas", variant, `${operation}_attributes.json`);

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
                this.logger.warn(`Loaded schema from ${schemaFilePath} is not a valid object.`);
            }
        } catch (error) {
            this.logger.warn(`Attributes schema file not found or failed to parse for ${variant} (${operation}) at ${schemaFilePath}: ${(error as Error).message}`);
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
        this.logger.info(`sendMessage called with new params: ${JSON.stringify(args)}`);

        try {
            const agentSenderId = this.user.id;

            // Resolve recipient to chatId using type guards
            let chatId: string;
            if (isRecipientChat(args.recipient)) {
                chatId = args.recipient.chatId;
            } else if (isRecipientBot(args.recipient)) {
                // TODO: Implement logic to find/create DM chat with bot
                const errorMsg = `Error: Recipient kind 'bot' resolution to a chat ID is not yet implemented. Bot ID: ${args.recipient.botId}`;
                this.logger.error(errorMsg, { recipient: args.recipient });
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            } else if (isRecipientUser(args.recipient)) {
                // TODO: Implement logic to find/create DM chat with user
                const errorMsg = `Error: Recipient kind 'user' resolution to a chat ID is not yet implemented. User ID: ${args.recipient.userId}`;
                this.logger.error(errorMsg, { recipient: args.recipient });
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            } else if (isRecipientTopic(args.recipient)) {
                // TODO: Implement MQTT/topic-based messaging
                const errorMsg = `Error: Recipient kind 'topic' is not yet implemented. Topic: ${args.recipient.topic}`;
                this.logger.error(errorMsg, { recipient: args.recipient });
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            } else {
                const errorMsg = `Error: Unknown recipient kind in: ${JSON.stringify(args.recipient)}`;
                this.logger.error(errorMsg);
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            }

            if (!chatId) {
                const errorMsg = `Error: Invalid or missing chat ID from recipient: ${JSON.stringify(args.recipient)}.`;
                this.logger.error(errorMsg);
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
                this.logger.warn(`sendMessage: 'mcpLlmModel' was not found in metadata for chat ${chatId}. Bot responses might not be triggered or use a default model.`);
            }

            this.logger.info(`Attempting to create chat message via createOneHelper. ChatID: ${chatId}, SenderID: ${agentSenderId}, RecipientKind: ${args.recipient.kind}`);

            const result = await createOneHelper({
                input: chatMessageInput,
                additionalData: additionalDataForHelper,
                objectType: "ChatMessage",
                req: this.req,
                info: { GQLResolveInfo: {} } as PartialApiInfo,
            }) as Partial<ChatMessage> & { id: string };

            this.logger.info(`Chat message created: ${result.id} in chat ${chatId}`);
            return {
                isError: false,
                content: [{ type: "text", text: `Message sent to chat ${chatId} (recipient kind: ${args.recipient.kind}). Message ID: ${result.id}` }],
            };

        } catch (error) {
            this.logger.error("Error in sendMessage:", error);
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
        this.logger.info(`resourceManage called with args: ${JSON.stringify(args)}`);
        let result;
        try {
            if (isResourceManageFindParams(args)) {
                const input = this._mapFindToInput(args.resource_type, args.filters || {});
                result = await readManyHelper({ info: {}, input, objectType: "ResourceVersion", req: this.req });
            } else if (isResourceManageAddParams(args)) {
                const createInput = this._mapAddToInput(args.resource_type, args.attributes);
                result = await createOneHelper({ info: {}, input: createInput, objectType: "ResourceVersion", req: this.req });
            } else if (isResourceManageUpdateParams(args)) {
                const updateInput = this._mapUpdateToInput(args.id, args.resource_type, args.attributes);
                result = await updateOneHelper({ info: {}, input: updateInput, objectType: "ResourceVersion", req: this.req });
            } else if (isResourceManageDeleteParams(args)) {
                result = await deleteOneHelper({ input: { id: args.id, objectType: DeleteType.ResourceVersion }, req: this.req });
            } else {
                // This should never happen with proper TypeScript discriminated unions
                const errorMsg = "Unhandled resource manage operation";
                this.logger.error(errorMsg, args);
                return { isError: true, content: [{ type: "text", text: errorMsg }] };
            }

            return { isError: false, content: [{ type: "text", text: JSON.stringify(result) }] };
        } catch (error) {
            this.logger.error("Error in resourceManage:", error);
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
        this.logger.info(`runRoutine called with args: ${JSON.stringify(args)}`);
        // TODO: Implement routine execution logic

        if (isRunRoutineStart(args)) {
            return {
                isError: false,
                content: [{ type: "text", text: `runRoutine operation '${args.action}' for routine '${args.routineId}' not implemented yet.` }],
            };
        } else {
            return {
                isError: false,
                content: [{ type: "text", text: `runRoutine operation '${args.action}' for run '${args.runId}' not implemented yet.` }],
            };
        }
    }

    /**
     * Handler for the SpawnSwarm MCP tool.
     * 
     * @param args - Parameters for spawning a swarm.
     * @returns A ToolResponse indicating the result of the spawnSwarm operation.
     */
    async spawnSwarm(args: SpawnSwarmParams): Promise<ToolResponse> {
        this.logger.info(`spawnSwarm called with args: ${JSON.stringify(args)}`);
        // TODO: Implement session start logic

        if (isSpawnSwarmSimple(args)) {
            return {
                isError: false,
                content: [{ type: "text", text: `spawnSwarm initiated with leader '${args.swarmLeader}' and goal '${args.goal}' - not implemented yet.` }],
            };
        } else {
            return {
                isError: false,
                content: [{ type: "text", text: `spawnSwarm initiated for team '${args.teamId}' with goal '${args.goal}' - not implemented yet.` }],
            };
        }
    }

    // -- Mapping layer: convert MCP shapes to GraphQL action-helper inputs --
    _mapFindToInput(resourceType: string, filters: Record<string, unknown>): Record<string, unknown> {
        this.logger.info(`Mapping find input for resourceType: ${resourceType} with filters: ${JSON.stringify(filters)}`);

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
     * Maps MCP 'add' operation attributes to the GraphQL create input for a ResourceVersion.
     * @param resourceType The type of resource (e.g., "Note").
     * @param attrs The attributes for the new resource.
     * @returns The GraphQL ResourceVersionCreateInput.
     */
    _mapAddToInput(resourceType: string, attrs: Record<string, unknown>): ResourceVersionCreateInput {
        switch (resourceType) {
            case "Note": {
                if (!isNoteAddAttributes(attrs)) {
                    throw new Error("Invalid attributes for Note: missing required properties 'name' or 'content'");
                }
                return {
                    id: generatePK().toString(),
                    isPrivate: true,
                    versionLabel: "0.0.1",
                    resourceSubType: resourceType as ResourceSubType,
                    translationsCreate: [
                        { id: generatePK().toString(), language: "en", name: attrs.name, description: attrs.content },
                    ],
                    rootCreate: {
                        id: generatePK().toString(),
                        isPrivate: true,
                        resourceType: ResourceType.Note,
                        tagsConnect: attrs.tagsConnect,
                    },
                };
            }
            default:
                //TODO: Implement other resource types
                throw new Error(`Add for variant '${resourceType}' not implemented`);
        }
    }

    /**
     * Maps MCP 'update' operation attributes to the GraphQL update input for a ResourceVersion.
     * @param resourceId The ID of the ResourceVersion to update.
     * @param resourceType The type of resource (e.g., "Note").
     * @param attrs The attributes to update.
     * @returns The GraphQL input for updating a ResourceVersion.
     */
    _mapUpdateToInput(resourceId: string, resourceType: string, _attrs: Record<string, unknown>): Record<string, unknown> {
        switch (resourceType) {
            case "Note":
                //TODO: Implement Note update
                throw new Error(`Update for variant '${resourceType}' not implemented`);
            default:
                throw new Error(`Update for variant '${resourceType}' not implemented`);
        }
    }
}

/**
 * Holds all swarm-specific MCP tools.
 */
export class SwarmTools {
    private logger: Logger;
    private conversationStore: ConversationStateStore;

    constructor(logger: Logger, conversationStore: ConversationStateStore) {
        this.logger = logger;
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
        this.logger.info(
            `SwarmTools.updateSwarmSharedState called for convo ${conversationId}`,
            args,
        );
        const convoState = await this.conversationStore.get(conversationId);

        if (!convoState) {
            const errorMsg = `Conversation state not found for conversation ${conversationId}.`;
            this.logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_STATE_NOT_FOUND" };
        }

        if (!convoState.config) {
            const errorMsg = `Conversation config not found for conversation ${conversationId}.`;
            this.logger.error(errorMsg);
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
                this.logger.error(errorMsg);
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
                    this.logger.error(errorMsg);
                    return { success: false, message: errorMsg, error: "TEAM_NOT_FOUND_OR_ACCESS_DENIED" };
                }

                // Parse current team config
                const currentTeamConfig = TeamConfig.parse({ config: currentTeam.config }, this.logger, { useFallbacks: true });

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
                this.logger.info(`Updated team config for team ${convoState.config.teamId}`);

                // Update the conversation state with the new team config
                try {
                    await this.conversationStore.updateTeamConfig(conversationId, updatedTeamConfig);
                    this.logger.debug(`Refreshed team config in conversation state for ${conversationId}`);
                } catch (updateError) {
                    this.logger.warn(`Failed to update team config in conversation state cache for ${conversationId}:`, updateError);
                    // Continue anyway - the team config was updated in the database and will be available on next fetch
                }

            } catch (error) {
                const errorMsg = `Failed to update team config: ${error instanceof Error ? error.message : "Unknown error"}`;
                this.logger.error(errorMsg, { error });
                return { success: false, message: errorMsg, error: "TEAM_CONFIG_UPDATE_FAILED" };
            }
        } else if (args.teamConfig && !convoState.config.teamId) {
            const errorMsg = "Cannot update team config: no team assigned to this swarm.";
            this.logger.warn(errorMsg);
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

        this.logger.info(
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
        this.logger.info(`SwarmTools.endSwarm called for convo ${conversationId} by user ${user.id}`, args);

        // Validate user authentication
        if (!user?.id) {
            const errorMsg = "User authentication required to end swarm.";
            this.logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "AUTHENTICATION_REQUIRED" };
        }

        // Get conversation state for authorization checks
        const convoState = await this.conversationStore.get(conversationId);
        if (!convoState) {
            const errorMsg = `Conversation state not found for conversation ${conversationId}.`;
            this.logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_STATE_NOT_FOUND" };
        }

        if (!convoState.config) {
            const errorMsg = `Conversation config not found for conversation ${conversationId}.`;
            this.logger.error(errorMsg);
            return { success: false, message: errorMsg, error: "CONVERSATION_CONFIG_NOT_FOUND" };
        }

        // Try to get the active SwarmStateMachine instance for additional authorization
        const activeSwarmInstance = activeSwarmRegistry.get(conversationId);
        if (activeSwarmInstance) {
            // Check if user is authorized - they should be the swarm initiator or an admin
            const swarmInitiatorId = activeSwarmInstance.getAssociatedUserId();
            const isSwarmInitiator = swarmInitiatorId === user.id;
            let isAdmin = false;
            try {
                const adminId = await DbProvider.getAdminId();
                isAdmin = user.id === adminId;
            } catch (error) {
                this.logger.error("Failed to get admin ID for swarm end authorization:", error);
                // Continue without admin check if DbProvider.getAdminId() fails
            }

            if (!isSwarmInitiator && !isAdmin) {
                const errorMsg = `User ${user.id} is not authorized to end swarm ${conversationId}. Only the swarm initiator (${swarmInitiatorId}) or an admin can end a swarm.`;
                this.logger.warn(errorMsg);
                return {
                    success: false,
                    message: "You are not authorized to end this swarm. Only the swarm creator or an admin can end a swarm.",
                    error: "INSUFFICIENT_PERMISSIONS",
                };
            }

            this.logger.info(`User ${user.id} authorized to end swarm ${conversationId}. Delegating to SwarmStateMachine.`);
            try {
                const result = await activeSwarmInstance.stop(args.mode || "graceful", args.reason, user);
                return result;
            } catch (error) {
                const errorMessage = error instanceof Error ? error.message : "Unknown error in SwarmStateMachine.stop()";
                this.logger.error(`Error delegating to SwarmStateMachine.stop() for conversation ${conversationId}:`, error);
                return {
                    success: false,
                    message: `Failed to end swarm via SwarmStateMachine: ${errorMessage}`,
                    error: "SWARM_STATE_MACHINE_ERROR",
                };
            }
        }

        // Fallback: If no active swarm instance, we can't properly validate or end the swarm
        this.logger.warn(`No active SwarmStateMachine found for conversation ${conversationId}. Cannot end swarm properly without state machine.`);

        return {
            success: false,
            message: "Cannot end swarm: No active SwarmStateMachine instance found. The swarm may have already ended or not been properly started.",
            error: "NO_ACTIVE_SWARM_INSTANCE",
        };
    }
} 
