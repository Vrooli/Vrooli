import fs from "fs";
import { fileURLToPath } from "node:url";
import path from "path";
import { type DefineToolParams, type ResourceManageParams, type RunRoutineParams, type SendMessageParams, type StartSessionParams, McpToolName, builtInToolDefinitions } from "./registry.js";
import { type Logger, type ToolResponse } from "./types.js";

/**
 * Handler for the DefineTool MCP tool.
 * Returns the detailed input schema for ResourceManage given a resource variant and CRUD operation.
 *
 * @param args - The arguments for the tool, specifying the variant and operation.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse object containing the JSON schema for the specified tool operation.
 */
export async function defineTool(args: DefineToolParams, logger: Logger): Promise<ToolResponse> {
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
    const toolDef = builtInToolDefinitions.get(toolName);
    if (!toolDef || !toolDef.inputSchema || !("oneOf" in toolDef.inputSchema)) {
        const errorMsg = `Error: Could not retrieve base schema for ${toolName}.`;
        logger.error(errorMsg);
        return {
            isError: true,
            content: [{ type: "text", text: errorMsg }],
        };
    }

    // Grab the correct operation schema template
    const opSchemaTemplate = toolDef.inputSchema.oneOf.find(
        (s: any) => s.properties && s.properties.op && s.properties.op.const === op,
    );
    if (!opSchemaTemplate) {
        const availableOps = toolDef.inputSchema.oneOf.map((s: any) => s.properties.op.const);
        const errorMsg = `Error: Could not find schema for operation '${op}' in ${toolName}. Available operations: ${availableOps.join(", ")}.`;
        logger.error(errorMsg);
        return {
            isError: true,
            content: [{ type: "text", text: errorMsg }],
        };
    }

    // Deep clone the template
    const finalSchema: Record<string, any> = JSON.parse(JSON.stringify(opSchemaTemplate));

    // General schema updates
    finalSchema.description = `Defines the parameters required to ${op} a resource of type ${variant}.`;
    finalSchema.properties.resource_type = { const: variant };

    // Operation-specific property refinement
    switch (op) {
        case "find": {
            let filterProperties: Record<string, any> = {};
            let filterRequired: string[] = [];
            const schemaFilePath = path.join(currentDirPath, "..", "schemas", variant, "find_filters.json");
            try {
                logger.info(`Attempting to load filter schema from: ${schemaFilePath}`);
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

            finalSchema.properties.filters = {
                type: "object",
                description: "Optional filters to filter results",
                properties: filterProperties,
                required: filterRequired,
                additionalProperties: false,
            };
            break;
        }
        case "add": {
            let attributeProperties: Record<string, any> = {};
            let attributeRequired: string[] = [];
            const schemaFilePath = path.join(currentDirPath, "..", "schemas", variant, "add_attributes.json");
            try {
                logger.info(`Attempting to load attributes schema from: ${schemaFilePath}`);
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
                logger.warn(`Attributes schema file not found or failed to parse for ${variant} (add) at ${schemaFilePath}: ${(error as Error).message}`);
            }

            finalSchema.properties.attributes = {
                type: "object",
                description: `Attributes for adding a new ${variant}.`,
                properties: attributeProperties,
                required: attributeRequired,
                additionalProperties: false,
            };
            break;
        }
        case "update": {
            let attributeProperties: Record<string, any> = {};
            let attributeRequired: string[] = [];
            const schemaFilePath = path.join(currentDirPath, "..", "schemas", variant, "update_attributes.json");
            try {
                logger.info(`Attempting to load attributes schema from: ${schemaFilePath}`);
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
                logger.warn(`Attributes schema file not found or failed to parse for ${variant} (update) at ${schemaFilePath}: ${(error as Error).message}`);
            }

            finalSchema.properties.attributes = {
                type: "object",
                description: `Attributes for updating an existing ${variant}.`,
                properties: attributeProperties,
                required: attributeRequired,
                additionalProperties: false,
            };
            break;
        }
        case "delete": {
            // No additional properties needed
            break;
        }
        default: {
            logger.warn(`DefineTool received an unhandled op: ${op} for variant ${variant}. Returning base schema for op.`);
            // For unhandled ops, the 'finalSchema' will be the cloned base for that op without further specialization.
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

/**
 * Handler for the SendMessage MCP tool.
 * @param args - The arguments for the tool, specifying recipient and content.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse indicating the result of the sendMessage operation.
 */
export async function sendMessage(args: SendMessageParams, logger: Logger): Promise<ToolResponse> {
    logger.info(`sendMessage called with args: ${JSON.stringify(args)}`);
    // TODO: Implement message sending logic
    return {
        isError: false,
        content: [{ type: "text", text: `Message sent to ${args.to}` }],
    };
}

/**
 * Handler for the ResourceManage MCP tool.
 * @param args - Parameters for resource management (CRUD operations).
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse indicating the result of the operation.
 */
export async function resourceManage(args: ResourceManageParams, logger: Logger): Promise<ToolResponse> {
    logger.info(`resourceManage called with args: ${JSON.stringify(args)}`);
    // TODO: Implement resource management logic
    return {
        isError: false,
        content: [{ type: "text", text: `resourceManage operation '${args.op}' for resource type '${args.resource_type}' not implemented yet.` }],
    };
}

/**
 * Handler for the RunRoutine MCP tool.
 * @param args - Parameters for routine execution.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse indicating the result of the runRoutine operation.
 */
export async function runRoutine(args: RunRoutineParams, logger: Logger): Promise<ToolResponse> {
    logger.info(`runRoutine called with args: ${JSON.stringify(args)}`);
    // TODO: Implement routine execution logic
    return {
        isError: false,
        content: [{ type: "text", text: `runRoutine operation '${args.op}'${args.routine_id ? ` for routine '${args.routine_id}'` : ""} not implemented yet.` }],
    };
}

/**
 * Handler for the StartSession MCP tool.
 * @param args - Parameters for starting a session.
 * @param logger - The logger instance for logging operations.
 * @returns A ToolResponse indicating the result of the startSession operation.
 */
export async function startSession(args: StartSessionParams, logger: Logger): Promise<ToolResponse> {
    logger.info(`startSession called with args: ${JSON.stringify(args)}`);
    // TODO: Implement session start logic
    return {
        isError: false,
        content: [{ type: "text", text: `startSession initiated${args.team_id ? ` for team '${args.team_id}'` : args.bot_ids ? ` for bots '${args.bot_ids.join(",")}'` : ""} not implemented yet.` }],
    };
}
