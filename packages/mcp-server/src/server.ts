import { Server as McpServer } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { CallToolRequestSchema, ListToolsRequestSchema } from '@modelcontextprotocol/sdk/types.js';
import express from 'express';
import type * as http from 'http';
import { ServerConfig } from './config/index.js';
import { ToolRegistry } from './tools/registry.js';
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

    /**
     * Set up MCP request handlers
     */
    private setupRequestHandlers(): void {
        // List tools request handler
        this.logger.info("Setting up ListToolsRequest handler...");
        this.mcpServer.setRequestHandler(ListToolsRequestSchema, async () => {
            this.logger.info("Received ListToolsRequest");
            const tools = this.toolRegistry.getDefinitions();
            this.logger.info(`Responding with ${tools.length} tools`);
            return { tools };
        });

        // Call tool request handler
        this.logger.info("Setting up CallToolRequest handler...");
        this.mcpServer.setRequestHandler(CallToolRequestSchema, async (request) => {
            const { name, arguments: args } = request.params;
            this.logger.info(`Received CallToolRequest for tool: ${name}`);
            const result = await this.toolRegistry.execute(name, args);

            // Format response according to MCP SDK expectations
            return {
                result: result
            };
        });
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
     * Start server in SSE mode
     */
    private async startSseMode(): Promise<void> {
        this.logger.info(`Initializing SSE mode on port ${this.config.port}...`);
        const app = express();

        // Create transport manager
        this.transportManager = new TransportManager(
            this.mcpServer,
            this.logger,
            {
                heartbeatInterval: this.config.heartbeatInterval,
                messagePath: this.config.messagePath
            }
        );

        // Health check endpoint
        app.get('/health', (req, res) => {
            const healthInfo = this.transportManager!.getHealthInfo();
            this.logger.info(`Health check requested: ${JSON.stringify(healthInfo)}`);
            res.json(healthInfo);
        });

        // SSE endpoint
        app.get('/sse', (req, res) => {
            this.transportManager!.handleSseConnection(req, res);
        });

        // POST endpoint for MCP messages
        app.post(this.config.messagePath, (req, res) => {
            this.transportManager!.handlePostMessage(req, res);
        });

        // Root path
        app.get('/', (req, res) => {
            res.send(`${this.config.serverInfo.name} v${this.config.serverInfo.version} is running. Use /sse for MCP connection.`);
        });

        // Start HTTP server
        return new Promise((resolve, reject) => {
            this.httpServer = app.listen(this.config.port, () => {
                this.logger.info(`${this.config.serverInfo.name} listening on http://localhost:${this.config.port}`);
                this.logger.info(`SSE endpoint available at http://localhost:${this.config.port}/sse`);
                this.logger.info(`MCP message endpoint at http://localhost:${this.config.port}${this.config.messagePath}`);
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

        if (this.config.mode === 'sse') {
            // First shut down transport connections
            if (this.transportManager) {
                await this.transportManager.shutdown();
            }

            // Then close the HTTP server
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
        }

        this.logger.info('Server shutdown complete.');
    }
} 