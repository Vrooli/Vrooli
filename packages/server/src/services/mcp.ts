import { ExecuteToolParams, ExecuteToolResponse, HttpStatus, ListToolsResponse, MCPEndpoint, MCPTool, RegisterToolParams, RegisterToolResponse, SearchToolsParams, SearchToolsResponse, ToolSpecificRequest } from "@local/shared";
import express, { Express, Request, Response } from "express";
import { logger } from "../events/logger.js";
import { MCP_SITE_WIDE_URL_SLUG, MCP_TOOL_URL_SLUG } from "../server.js";
import { ResponseService } from "../utils/response.js";

/**
 * Registry to store and manage MCP tools (routines).
 * Updated to store tools by their unique "id" property.
 */
class ToolRegistry {
    private static instance: ToolRegistry;
    private tools: Map<string, MCPTool>;

    private constructor() {
        this.tools = new Map();
    }

    public static get(): ToolRegistry {
        if (!ToolRegistry.instance) {
            ToolRegistry.instance = new ToolRegistry();
        }
        return ToolRegistry.instance;
    }

    public registerTool(tool: MCPTool): void {
        // Assume each MCPTool has a unique "id" property.
        this.tools.set(tool.id, tool);
    }

    public getTool(id: string): MCPTool | undefined {
        return this.tools.get(id);
    }

    public listTools(): MCPTool[] {
        return Array.from(this.tools.values());
    }

    public searchTools(query: string): MCPTool[] {
        const lowerQuery = query.toLowerCase();
        return this.listTools().filter(tool =>
            tool.name.toLowerCase().includes(lowerQuery) ||
            tool.description.toLowerCase().includes(lowerQuery),
        );
    }
}

/**
 * Site-wide MCP handler for executing a tool.
 * This expects a JSON-RPC style request.
 */
async function executeTool(req: Request, res: Response): Promise<void> {
    try {
        const { name, arguments: toolArgs } = req.body as ExecuteToolParams;
        const tool = ToolRegistry.get().getTool(name);
        if (!tool) {
            ResponseService.sendError(res, {
                trace: "0001",
                message: `Tool '${name}' not found`,
            }, HttpStatus.NotFound);
            return;
        }

        const result = await tool.execute(toolArgs);
        const data: ExecuteToolResponse = {
            success: true,
            result,
        };
        ResponseService.sendSuccess(res, data);
    } catch (error) {
        const trace = "0001";
        const message = "Error executing MCP tool";
        logger.error(message, { trace, error });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * Site-wide MCP handler for listing all available tools.
 */
async function listTools(_req: Request, res: Response): Promise<void> {
    try {
        const tools = ToolRegistry.get().listTools();
        const data: ListToolsResponse = {
            success: true,
            tools,
        };
        ResponseService.sendSuccess(res, data);
    } catch (error) {
        const trace = "0002";
        const message = "Error listing MCP tools";
        logger.error(message, { trace, error });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * Site-wide MCP handler for searching tools.
 */
async function searchTools(req: Request, res: Response): Promise<void> {
    try {
        const { query } = req.query as unknown as SearchToolsParams;
        const results = ToolRegistry.get().searchTools(query);
        const data: SearchToolsResponse = {
            success: true,
            results,
        };
        ResponseService.sendSuccess(res, data);
    } catch (error) {
        const trace = "0003";
        const message = "Error searching MCP tools";
        logger.error(message, { trace, error });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * Site-wide MCP handler for registering a new tool.
 */
async function registerTool(req: Request, res: Response): Promise<void> {
    try {
        const newTool = req.body as RegisterToolParams;
        ToolRegistry.get().registerTool(newTool);
        const data: RegisterToolResponse = {
            success: true,
            tool: newTool,
        };
        ResponseService.sendSuccess(res, data);
    } catch (error) {
        const trace = "0004";
        const message = "Error registering new MCP tool";
        logger.error(message, { trace, error });
        ResponseService.sendError(res, { trace, message }, HttpStatus.InternalServerError);
    }
}

/**
 * Routine-specific handler for endpoints at "/mcp/tool/:id".
 *
 * For each request:
 * - It checks if the routine is already registered.
 * - If not, it loads the routine data (placeholder for a DB call) and registers it.
 * - Then it processes the JSON-RPC request (executeTool, listTools, searchTools).
 */
async function toolSpecificHandler(req: Request, res: Response): Promise<void> {
    try {
        const routineId = req.params.id;
        let tool = ToolRegistry.get().getTool(routineId);
        if (!tool) {
            // Placeholder: load routine data from a database
            console.log(`Routine ${routineId} not found in registry. Loading from DB...`);
            // Replace this placeholder with your actual DB call:
            const loadedRoutine = {
                id: routineId,
                name: `Routine ${routineId}`,
                description: `Placeholder routine loaded from DB for id ${routineId}`,
                execute: async (args: any) => `Executed routine ${routineId} with args: ${JSON.stringify(args)}`,
            } as MCPTool;
            ToolRegistry.get().registerTool(loadedRoutine);
            tool = loadedRoutine;
        }

        // Process the JSON-RPC style request for the tool-specific endpoint.
        const { method, arguments: toolArgs } = req.body as ToolSpecificRequest;
        if (method === "executeTool") {
            const result = await tool.execute(toolArgs);
            const data: ExecuteToolResponse = {
                success: true,
                result,
            };
            ResponseService.sendSuccess(res, data);
        } else if (method === "listTools") {
            // For a routine-specific endpoint, listing tools returns only this routine.
            const data: ListToolsResponse = {
                success: true,
                tools: [tool],
            };
            ResponseService.sendSuccess(res, data);
        } else if (method === "searchTools") {
            const query = req.body.query || "";
            if (
                tool.name.toLowerCase().includes(query.toLowerCase()) ||
                tool.description.toLowerCase().includes(query.toLowerCase())
            ) {
                const data: SearchToolsResponse = {
                    success: true,
                    results: [tool],
                };
                ResponseService.sendSuccess(res, data);
            } else {
                const data: SearchToolsResponse = {
                    success: true,
                    results: [],
                };
                ResponseService.sendSuccess(res, data);
            }
        } else {
            ResponseService.sendError(res, {
                trace: "TOOL_001",
                message: `Method '${method}' not supported on routine-specific endpoint`,
            }, HttpStatus.NotFound);
        }
    } catch (error) {
        ResponseService.sendError(res, {
            trace: "TOOL_002",
            message: "Error processing routine-specific MCP request",
        }, HttpStatus.InternalServerError);
    }
}

/**
 * Sets up MCP-related routes on the provided Express application instance.
 *
 * This includes:
 * - Site-wide endpoints at "/mcp" (execute, list, search, register)
 * - Routine-specific endpoints at "/mcp/tool/:id"
 *
 * @param app - The Express application instance to attach routes to.
 */
export function setupMCP(app: Express): void {
    const mcpRouter = express.Router();

    // Site-wide MCP endpoints
    mcpRouter.post(MCPEndpoint.Execute, executeTool);
    mcpRouter.get(MCPEndpoint.ListTools, listTools);
    mcpRouter.get(MCPEndpoint.SearchTools, searchTools);
    mcpRouter.post(MCPEndpoint.RegisterTool, registerTool);

    // Attach the site-wide router to "/mcp"
    app.use(MCP_SITE_WIDE_URL_SLUG, mcpRouter);

    // Routine-specific MCP endpoints at "/mcp/tool/:id"
    const toolRouter = express.Router();
    toolRouter.use("/:id", toolSpecificHandler);
    app.use(MCP_TOOL_URL_SLUG, toolRouter);
}
