import { ApiKeyPermission, HttpStatus } from "@local/shared";
import { Server as McpServer } from "@modelcontextprotocol/sdk/server/index.js";
import { CallToolRequestSchema, ListToolsRequestSchema, ServerResult } from "@modelcontextprotocol/sdk/types.js";
import { Express } from "express";
import { RequestService } from "../../auth/request.js";
import { runWithMcpContext } from "./context.js";
import { McpRoutineToolName, McpToolName, ToolRegistry } from "./registry.js";
import { TransportManager } from "./transport.js";
import { Logger } from "./types.js";

/**
 * How the MCP server is configured to manage connections.
 */
export enum McpServerMode {
    /** 
     * Standard Input/Output mode, for accessing locally.
     * 
     * NOTE: This may not work if developing in WSL (Windows Subsystem for Linux).
     * If you're having issues, try using SSE mode instead and running this in a VPS.
     */
    STDIO = "stdio",
    /**
     * Server-Sent Events mode, for accessing remotely.
     */
    SSE = "sse",
}

export interface ServerConfig {
    mode: McpServerMode;
    port: number;
    messagePath: string;
    heartbeatInterval: number;
    serverInfo: {
        name: string;
        version: string;
    };
}

/**
 * Main MCP Server application class that orchestrates all components
 */
export class McpServerApp {
    private app: Express;
    private config: ServerConfig;
    private logger: Logger;
    private mcpServer: McpServer;
    private toolRegistry: ToolRegistry;
    private transportManager?: TransportManager;
    private dynamicServers: Map<string, McpServer> = new Map(); // Cache for tool-specific servers
    private dynamicTransportManagers: Map<string, TransportManager> = new Map(); // Store managers keyed by toolId

    constructor(
        config: ServerConfig,
        logger: Logger,
        app: Express,
    ) {
        this.config = config;
        this.logger = logger;
        this.app = app;
        this.toolRegistry = new ToolRegistry(logger);

        // Initialize the MCP server with core capabilities
        this.mcpServer = new McpServer(
            this.config.serverInfo,
            { capabilities: { tools: {} } },
        );

        // Set up request handlers
        this.setupRequestHandlers();
    }

    /**
     * Get the underlying McpServer instance.
     * @returns The McpServer instance.
     */
    public getServerInstance(): McpServer {
        return this.mcpServer;
    }

    /**
     * Get the ToolRegistry instance.
     * @returns The ToolRegistry instance.
     */
    public getToolRegistry(): ToolRegistry {
        return this.toolRegistry;
    }

    public getTransportManager(): TransportManager {
        if (!this.transportManager) {
            throw new Error("TransportManager not initialized");
        }
        return this.transportManager;
    }

    /**
     * Set up MCP request handlers
     */
    private setupRequestHandlers(): void {
        // List tools request handler
        this.logger.info("Setting up ListToolsRequest handler...");
        this.mcpServer.setRequestHandler(ListToolsRequestSchema, async () => {
            this.logger.info("Received ListToolsRequest");
            const tools = this.toolRegistry.getBuiltInDefinitions();
            this.logger.info(`Responding with ${tools.length} tools`);
            return { tools } as ServerResult;
        });

        // Call tool request handler
        this.logger.info("Setting up CallToolRequest handler...");
        this.mcpServer.setRequestHandler(CallToolRequestSchema, async (request) => {
            const { name, arguments: args } = request.params;
            this.logger.info(`Received CallToolRequest for tool: ${name}`);
            return this.toolRegistry.execute(name as McpToolName, args) as Promise<ServerResult>;
        });
    }

    /**
     * Gets or creates a dedicated, single-tool McpServer instance.
     * @param toolId The ID of the tool
     * @returns The Server instance or null if the tool doesn't exist.
     */
    private async getOrCreateDynamicServer(toolId: string): Promise<McpServer | null> {
        if (this.dynamicServers.has(toolId)) {
            return this.dynamicServers.get(toolId)!;
        }

        // Await the promise to get the array of definitions
        const matchingRoutines = await findRoutines({ id: toolId }, this.logger);
        if (matchingRoutines.length === 0) {
            this.logger.error(`Attempted to create dynamic server for non-existent routine: ${toolId}`);
            return null;
        }

        const routine = matchingRoutines[0];
        const tools = [
            {
                name: McpRoutineToolName.StartRoutine,
                description: routine.description || routine.name || "No description",
                inputSchema: {
                    type: "object",
                    properties: {
                        arg: { type: "string" },
                    },
                    required: ["arg"],
                },
                annotations: {
                    title: routine.name || routine.id,
                    readOnlyHint: true,
                    openWorldHint: false,
                },
            },
            {
                name: McpRoutineToolName.StopRoutine,
                description: "Stop the routine (if running)",
                inputSchema: {
                    type: "object",
                    properties: {},
                    required: [],
                },
                annotations: {
                    title: "Stop",
                    readOnlyHint: true,
                    openWorldHint: false,
                },
            },
        ];

        this.logger.info(`Creating new dynamic server instance for tool: ${toolId}`);

        // Create a new low-level server instance specific to this tool
        const dynamicServerInfo = {
            ...this.config.serverInfo,
            name: `${this.config.serverInfo.name} [${toolId}]`, // Give it a distinct name
        };
        const dynamicServer = new McpServer(dynamicServerInfo, { capabilities: { tools: {} } });

        // Register ListTools handler - returns ONLY this tool
        dynamicServer.setRequestHandler(ListToolsRequestSchema, async () => {
            this.logger.info(`Dynamic Server [${toolId}]: Received ListToolsRequest`, { tools });
            return { tools } as ServerResult;
        });

        // Register CallTool handler - executes ONLY this tool
        dynamicServer.setRequestHandler(CallToolRequestSchema, async (request) => {
            const { name, arguments: args } = request.params;
            this.logger.info(`Dynamic Server [${toolId}]: Received CallToolRequest for tool: ${name} with args: ${JSON.stringify(args)}`);

            if (name === McpRoutineToolName.StartRoutine) {
                // Execute using the specific handler fetched from the registry
                // We pass the *dynamicServer's* logger if the handler accepts it
                try {
                    const result = await Promise.resolve(runRoutine({ id: routine.id, replacement: (args as { arg: string }).arg || "" }, this.logger));
                    return result as ServerResult;
                } catch (error) {
                    this.logger.error(`Dynamic Server [${toolId}]: Error executing tool ${name}:`, error);
                    return {
                        isError: true,
                        content: [{ type: "text", text: `Error executing tool '${name}': ${(error as Error).message}` }],
                    } as ServerResult;
                }
            } else if (name === McpRoutineToolName.StopRoutine) {
                this.logger.info(`Dynamic Server [${toolId}]: Stopping routine`);
                return {
                    isError: false,
                    content: [{ type: "text", text: `Routine '${toolId}' stopped` }],
                } as ServerResult;
            } else {
                this.logger.warn(`Dynamic Server [${toolId}]: Denied execution request for tool '${name}'`);
                return {
                    isError: true,
                    content: [{ type: "text", text: `Error: Tool '${name}' not implemented by this server instance` }],
                } as ServerResult;
            }
        });

        this.dynamicServers.set(toolId, dynamicServer);
        return dynamicServer;
    }

    /**
     * Start the server based on the configured mode
     */
    async start(): Promise<void> {
        this.logger.info(`Starting server in ${this.config.mode} mode...`);

        if (this.config.mode === "sse") {
            await this.startSseMode();
        } else {
            await this.startStdioMode();
        }

        this.logger.info(`Server started successfully in ${this.config.mode} mode.`);
    }

    /**
     * Start server in SSE mode - Handles standard and dynamic routes
     */
    private async startSseMode(): Promise<void> {
        this.logger.info(`Initializing SSE mode on port ${this.config.port}...`);

        // Standard TransportManager for the main /mcp/sse endpoint
        this.transportManager = new TransportManager(
            this.mcpServer,
            this.logger,
            { heartbeatInterval: this.config.heartbeatInterval, messagePath: this.config.messagePath },
        );

        // Standard Health Check
        this.app.get("/mcp/health", (req, res) => {
            // Validate request permissions via RequestService
            const { hasApiToken } = RequestService.getRequestPermissions(req);
            if (!hasApiToken) {
                return res.status(HttpStatus.Unauthorized).json({ error: "Unauthorized" });
            }

            // Reports health of the main transport manager
            const healthInfo = this.getTransportManager().getHealthInfo();
            res.json(healthInfo);
        });

        // --- Dynamic Tool Routes ---

        // 1. SSE Connection Endpoint for a specific tool
        this.app.get("/mcp/tool/:tool_id/sse", async (req, res) => {
            const { hasApiToken: hasApiTokenTool } = RequestService.getRequestPermissions(req);
            if (!hasApiTokenTool) {
                return res.status(HttpStatus.Unauthorized).json({ error: "Unauthorized" });
            }
            const toolId = req.params.tool_id;
            this.logger.info(`Dynamic SSE connection request for tool: ${toolId}`);

            const serverInstance = await this.getOrCreateDynamicServer(toolId);
            if (!serverInstance) {
                return res.status(HttpStatus.NotFound).json({ error: `Tool '${toolId}' not found or cannot be served dynamically.` });
            }

            // Get or create the TransportManager for this toolId
            let dynamicTransportManager = this.dynamicTransportManagers.get(toolId);
            if (!dynamicTransportManager) {
                this.logger.info(`Creating dedicated TransportManager for tool ${toolId}`);
                dynamicTransportManager = new TransportManager(
                    serverInstance, // Link to the specific single-tool server instance
                    this.logger,
                    {
                        heartbeatInterval: this.config.heartbeatInterval,
                        // The message path should be specific to the tool
                        messagePath: `/mcp/tool/${toolId}/message`,
                    },
                );
                this.dynamicTransportManagers.set(toolId, dynamicTransportManager);
            } else {
                this.logger.info(`Reusing existing TransportManager for tool ${toolId}`);
            }

            // Delegate SSE handling to the dedicated TransportManager
            dynamicTransportManager.handleSseConnection(req, res);
        });

        // 2. Message Endpoint for a specific tool's connection
        this.app.post("/mcp/tool/:tool_id/message", (req, res) => {
            runWithMcpContext(req, res, async () => {
                const { hasApiToken: hasApiTokenMessage } = RequestService.getRequestPermissions(req);
                if (!hasApiTokenMessage) {
                    return res.status(HttpStatus.Unauthorized).json({ error: "Unauthorized" });
                }
                const toolId = req.params.tool_id;
                this.logger.info(`Dynamic message POST received for tool ${toolId}. Path: ${req.path}`);

                const manager = this.dynamicTransportManagers.get(toolId);
                if (manager) {
                    this.logger.info(`Found TransportManager for tool ${toolId}, delegating POST handling.`);
                    manager.handlePostMessage(req, res);
                } else {
                    this.logger.error(`No active TransportManager found for tool ${toolId} to handle POST request.`);
                    if (!res.headersSent) {
                        res.status(HttpStatus.NotFound).json({ error: `No active session found for tool '${toolId}'. Please establish an SSE connection first.` });
                    }
                }
            }).catch(error => {
                this.logger.error("Error in MCP /mcp/tool/:tool_id/message handler:", error);
                if (!res.headersSent) res.status(HttpStatus.InternalServerError).end();
            });
        });

        // --- Standard MCP Routes (handled by TransportManager) ---
        this.app.get("/mcp/sse", (req, res) => {
            // Require public read permission for built-in tools
            const { permissions } = RequestService.getRequestPermissions(req);
            if (!permissions[ApiKeyPermission.ReadPublic]) {
                return res.status(HttpStatus.Unauthorized).json({ error: "Forbidden - missing read permission" });
            }
            this.logger.info("Handling standard /mcp/sse connection. Path: " + req.path);
            this.getTransportManager().handleSseConnection(req, res);
        });
        this.app.post(this.config.messagePath, (req, res) => {
            runWithMcpContext(req, res, async () => {
                const { permissions } = RequestService.getRequestPermissions(req);
                if (!permissions[ApiKeyPermission.WritePrivate]) {
                    return res.status(HttpStatus.Unauthorized).json({ error: "Forbidden - missing write permission" });
                }
                this.logger.info("Handling standard message POST. Path: " + req.path);
                this.getTransportManager().handlePostMessage(req, res);
            }).catch(error => {
                this.logger.error("Error in MCP standard message handler:", error);
                if (!res.headersSent) res.status(HttpStatus.InternalServerError).end();
            });
        });

        this.logger.info(`${this.config.serverInfo.name} routes configured on provided Express app.`);
        this.logger.info("MCP Health endpoint available at /mcp/health");
        this.logger.info("MCP SSE endpoint available at /mcp/sse");
        this.logger.info(`MCP Message endpoint at ${this.config.messagePath}`);
        // Resolve immediately as the listening is handled externally
        return Promise.resolve();
    }

    /**
     * Start server in STDIO mode
     */
    private async startStdioMode(): Promise<void> {
        this.logger.info("Initializing STDIO mode...");
        // TODO: Implement
    }

    /**
     * Gracefully shut down the server
     */
    async shutdown(): Promise<void> {
        this.logger.info("Shutting down server...");

        // Shutdown dynamic transport managers first
        this.logger.info(`Shutting down ${this.dynamicTransportManagers.size} dynamic transport managers...`);
        for (const [toolId, manager] of this.dynamicTransportManagers) {
            this.logger.info(`Shutting down manager for tool: ${toolId}`);
            try {
                await manager.shutdown(); // Assuming TransportManager has an async shutdown method
            } catch (error) {
                this.logger.error(`Error shutting down manager for tool ${toolId}:`, error);
            }
        }
        this.dynamicTransportManagers.clear();
        this.dynamicServers.clear(); // Clear the cache of dynamic servers

        // Shutdown standard transport manager
        if (this.config.mode === "sse" && this.transportManager) {
            await this.transportManager.shutdown();
        }
        this.logger.info("Server shutdown complete.");
    }
} 
