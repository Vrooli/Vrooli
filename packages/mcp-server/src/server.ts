import { Server as McpServer } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { CallToolRequestSchema, ListToolsRequestSchema, ServerResult } from '@modelcontextprotocol/sdk/types.js';
import express from 'express';
import type * as http from 'http';
import { ServerConfig } from './config/index.js';
import { findRoutine, runRoutine } from './tools.js';
import { McpRoutineToolName, McpToolName, ToolRegistry } from './tools/registry.js';
import { TransportManager } from './transports/transport-manager.js';
import { Logger } from './types.js';

/**
 * Main MCP Server application class that orchestrates all components
 */
export class McpServerApp {
    private config: ServerConfig;
    private logger: Logger;
    private mcpServer: McpServer;
    private toolRegistry: ToolRegistry;
    private transportManager?: TransportManager;
    private httpServer?: http.Server;
    private dynamicServers: Map<string, McpServer> = new Map(); // Cache for tool-specific servers
    private dynamicTransportManagers: Map<string, TransportManager> = new Map(); // Store managers keyed by toolId

    constructor(config: ServerConfig, logger: Logger) {
        this.config = config;
        this.logger = logger;
        this.toolRegistry = new ToolRegistry(logger);

        // Initialize the MCP server with core capabilities
        this.mcpServer = new McpServer(
            this.config.serverInfo,
            { capabilities: { tools: {} } }
        );

        // Set up request handlers
        this.setupRequestHandlers();
    }

    private getTransportManager(): TransportManager {
        if (!this.transportManager) {
            throw new Error('TransportManager not initialized');
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
        const matchingRoutines = await findRoutine({ id: toolId }, this.logger);
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
                    type: 'object',
                    properties: {
                        arg: { type: 'string' }
                    },
                    required: ['arg']
                },
                annotations: {
                    title: routine.name || routine.id,
                    readOnlyHint: true,
                    openWorldHint: false
                }
            },
            {
                name: McpRoutineToolName.StopRoutine,
                description: "Stop the routine (if running)",
                inputSchema: {
                    type: 'object',
                    properties: {},
                    required: []
                },
                annotations: {
                    title: "Stop",
                    readOnlyHint: true,
                    openWorldHint: false
                }
            }
        ];
        const toolHandler = this.toolRegistry.getBuiltInTool(McpToolName.RunRoutine);

        this.logger.info(`Creating new dynamic server instance for tool: ${toolId}`);

        // Create a new low-level server instance specific to this tool
        const dynamicServerInfo = {
            ...this.config.serverInfo,
            name: `${this.config.serverInfo.name} [${toolId}]` // Give it a distinct name
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
                    const result = await Promise.resolve(runRoutine({ id: routine.id, replacement: (args as { arg: string }).arg || '' }, this.logger));
                    return result as ServerResult;
                } catch (error) {
                    this.logger.error(`Dynamic Server [${toolId}]: Error executing tool ${name}:`, error);
                    return {
                        isError: true,
                        content: [{ type: 'text', text: `Error executing tool '${name}': ${(error as Error).message}` }]
                    } as ServerResult;
                }
            } else if (name === McpRoutineToolName.StopRoutine) {
                this.logger.info(`Dynamic Server [${toolId}]: Stopping routine`);
                return {
                    isError: false,
                    content: [{ type: 'text', text: `Routine '${toolId}' stopped` }]
                } as ServerResult;
            } else {
                this.logger.warn(`Dynamic Server [${toolId}]: Denied execution request for tool '${name}'`);
                return {
                    isError: true,
                    content: [{ type: 'text', text: `Error: Tool '${name}' not implemented by this server instance` }]
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

        if (this.config.mode === 'sse') {
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
        const app = express();

        // Standard TransportManager for the main /mcp/sse endpoint
        this.transportManager = new TransportManager(
            this.mcpServer, // Connects to the main McpServer instance
            this.logger,
            { heartbeatInterval: this.config.heartbeatInterval, messagePath: this.config.messagePath }
        );

        // Standard Health Check
        app.get('/mcp/health', (req, res) => {
            // Reports health of the main transport manager
            const healthInfo = this.getTransportManager().getHealthInfo();
            res.json(healthInfo);
        });

        // --- Dynamic Tool Routes ---

        // 1. SSE Connection Endpoint for a specific tool
        app.get('/mcp/tool/:tool_id/sse', async (req, res) => {
            const toolId = req.params.tool_id;
            this.logger.info(`Dynamic SSE connection request for tool: ${toolId}`);

            const serverInstance = await this.getOrCreateDynamicServer(toolId);
            if (!serverInstance) {
                return res.status(404).json({ error: `Tool '${toolId}' not found or cannot be served dynamically.` });
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
                        messagePath: `/mcp/tool/${toolId}/message`
                    }
                );
                this.dynamicTransportManagers.set(toolId, dynamicTransportManager);
            } else {
                this.logger.info(`Reusing existing TransportManager for tool ${toolId}`);
            }

            // Delegate SSE handling to the dedicated TransportManager
            dynamicTransportManager.handleSseConnection(req, res);
        });

        // 2. Message Endpoint for a specific tool's connection
        app.post('/mcp/tool/:tool_id/message', (req, res) => { // Changed to non-async, handlePostMessage is likely sync or handles its own async
            const toolId = req.params.tool_id;
            this.logger.info(`Dynamic message POST received for tool ${toolId}. Path: ${req.path}`);

            const manager = this.dynamicTransportManagers.get(toolId);

            if (manager) {
                this.logger.info(`Found TransportManager for tool ${toolId}, delegating POST handling.`);
                // Let the specific manager handle the raw request
                manager.handlePostMessage(req, res);
            } else {
                // This might happen if a POST arrives before any SSE connection for this tool
                // or if the server restarted without persisting state.
                this.logger.error(`No active TransportManager found for tool ${toolId} to handle POST request.`);
                if (!res.headersSent) {
                    res.status(404).json({ error: `No active session found for tool '${toolId}'. Please establish an SSE connection first.` });
                }
            }
        });

        // --- Standard MCP Routes (handled by TransportManager) ---
        app.get('/mcp/sse', (req, res) => {
            this.logger.info("Handling standard /mcp/sse connection. Path: " + req.path);
            this.getTransportManager().handleSseConnection(req, res);
        });
        app.post(this.config.messagePath, (req, res) => { // Removed express.json()
            this.logger.info("Handling standard message POST. Path: " + req.path);
            // TransportManager.handlePostMessage will read the raw body
            this.getTransportManager().handlePostMessage(req, res);
        });

        // Start HTTP server
        return new Promise((resolve, reject) => {
            this.httpServer = app.listen(this.config.port, () => {
                this.logger.info(`${this.config.serverInfo.name} listening on http://localhost:${this.config.port}`);
                this.logger.info(`MCP Health endpoint available at http://localhost:${this.config.port}/mcp/health`);
                this.logger.info(`MCP SSE endpoint available at http://localhost:${this.config.port}/mcp/sse`);
                this.logger.info(`MCP Message endpoint at http://localhost:${this.config.port}${this.config.messagePath}`);
                resolve();
            });

            this.httpServer.on('error', (error) => {
                this.logger.error("HTTP server error:", error);
                reject(error);
            });
        });
    }

    /**
     * Start server in STDIO mode
     */
    private async startStdioMode(): Promise<void> {
        this.logger.info(`Initializing STDIO mode...`);

        const transport = new StdioServerTransport();

        try {
            await this.mcpServer.connect(transport);
            this.logger.info("Server transport connected successfully.");
            this.logger.info("Server is running. Waiting for requests via stdin/stdout...");
        } catch (error) {
            this.logger.error("Failed to connect server transport:", error);
            throw error;
        }
    }

    /**
     * Gracefully shut down the server
     */
    async shutdown(): Promise<void> {
        this.logger.info('Shutting down server...');

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

        // Shutdown standard transport manager and HTTP server
        if (this.config.mode === 'sse' && this.transportManager) {
            await this.transportManager.shutdown();
        }
        if (this.httpServer) {
            await new Promise<void>((resolve, reject) => {
                this.httpServer!.close((err) => {
                    if (err) {
                        this.logger.error('Error closing HTTP Server:', err);
                        reject(err);
                    } else {
                        this.logger.info('HTTP Server closed.');
                        resolve();
                    }
                });
            });
        }
        this.logger.info('Server shutdown complete.');
    }
} 