import { Server as McpServer } from "@modelcontextprotocol/sdk/server/index.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { CallToolRequestSchema, ListToolsRequestSchema } from "@modelcontextprotocol/sdk/types.js";
import express from 'express';
import type * as http from 'http'; // Import type for ServerResponse

// --- Constants ---
const serverInfo = { name: "vrooli-mcp-server", version: "0.1.0" } as const;
const protocolVersion = "2025-04-15" as const; // Keep for reference, maybe used later
const jsonRpcVersion = "2.0" as const; // Keep for reference
const SSE_PORT = 3100;
const SSE_MESSAGE_PATH = '/mcp'; // Define a path for MCP messages

// --- SSE State (Module Scope) ---
// Map to store transport instances keyed by their response object
const transports = new Map<http.ServerResponse, SSEServerTransport>();
// Keep track of active connections for graceful shutdown and metrics
const connections = new Set<http.ServerResponse>();

type Mode = 'stdio' | 'sse';

/**
 * Tool annotations providing metadata about a tool's behavior.
 */
interface ToolAnnotations {
    /** Human-readable title for the tool */
    title?: string;
    /** If true, indicates the tool does not modify its environment */
    readOnlyHint?: boolean;
    /** If true, the tool may perform destructive updates */
    destructiveHint?: boolean;
    /** If true, calling the tool repeatedly with the same arguments has no additional effect */
    idempotentHint?: boolean;
    /** If true, the tool may interact with an "open world" of external entities */
    openWorldHint?: boolean;
}

/**
 * Interface representing a Tool in the MCP protocol.
 */
interface Tool {
    /** Unique identifier for the tool */
    name: string;
    /** Human-readable description */
    description?: string;
    /** JSON Schema for the tool's parameters */
    inputSchema: {
        type: string;
        properties: Record<string, any>;
        required?: string[];
    };
    /** Optional hints about tool behavior */
    annotations?: ToolAnnotations;
}

// --- Fake Winston Logger ---
/**
 * A simple logger mimicking the Winston interface for basic logging needs.
 * Logs messages to the console.
 */
function createLogger() {
    function log(level: string, message: string, ...meta: any[]) {
        const timestamp = new Date().toISOString();
        console.log(`[${timestamp}] [${level.toUpperCase()}] ${message}`, ...meta);
    };
    return {
        info: (message: string, ...meta: any[]) => log('info', message, ...meta),
        warn: (message: string, ...meta: any[]) => log('warn', message, ...meta),
        error: (message: string, ...meta: any[]) => log('error', message, ...meta),
        debug: (message: string, ...meta: any[]) => log('debug', message, ...meta), // Added debug for completeness
    };
};

type Logger = ReturnType<typeof createLogger>;

// --- Health Check Logic ---
/**
 * Returns health information specific to the SSE mode.
 * @returns An object containing SSE health metrics.
 */
function getSseHealthInfo() {
    return {
        status: 'ok', // Basic status
        activeConnections: connections.size,
        // Add other SSE-specific metrics here in the future
    };
}

/**
 * Fetches tool information and returns mock tool data.
 * @param logger - The logger instance for logging operations.
 * @returns A Promise resolving to an array of mock tools.
 */
async function fetchToolInformation(logger: Logger): Promise<Tool[]> {
    logger.info("Fetching tool information...");

    // Simulate network delay
    await new Promise(resolve => setTimeout(resolve, 300));

    const tools: Tool[] = [
        {
            name: "web_search",
            description: "Search the web for information",
            inputSchema: {
                type: "object",
                properties: {
                    query: { type: "string" }
                },
                required: ["query"]
            },
            annotations: {
                title: "Web Search",
                readOnlyHint: true,
                openWorldHint: true
            }
        },
        {
            name: "calculate_sum",
            description: "Add two numbers together",
            inputSchema: {
                type: "object",
                properties: {
                    a: { type: "number" },
                    b: { type: "number" }
                },
                required: ["a", "b"]
            },
            annotations: {
                title: "Calculate Sum",
                readOnlyHint: true,
                openWorldHint: false
            }
        },
        {
            name: "create_file",
            description: "Create a new file with the given content",
            inputSchema: {
                type: "object",
                properties: {
                    path: { type: "string" },
                    content: { type: "string" }
                },
                required: ["path", "content"]
            },
            annotations: {
                title: "Create File",
                readOnlyHint: false,
                destructiveHint: false,
                idempotentHint: false,
                openWorldHint: false
            }
        },
        {
            name: "delete_file",
            description: "Delete a file from the filesystem",
            inputSchema: {
                type: "object",
                properties: {
                    path: { type: "string" }
                },
                required: ["path"]
            },
            annotations: {
                title: "Delete File",
                readOnlyHint: false,
                destructiveHint: true,
                idempotentHint: true,
                openWorldHint: false
            }
        }
    ];

    logger.info(`Fetched ${tools.length} tools.`);
    return tools;
}

// --- Mode Selection Function ---
/**
 * Parses command line arguments to determine the server mode.
 * @param logger - The logger instance.
 * @returns The selected mode ('stdio' or 'sse').
 */
function getModeFromArgs(logger: Logger): Mode {
    const args = process.argv.slice(2);
    let mode: Mode = 'sse'; // Default to sse

    args.forEach(arg => {
        if (arg.startsWith('--mode=')) {
            const value = arg.split('=')[1];
            if (value === 'sse' || value === 'stdio') {
                mode = value;
            } else {
                logger.warn(`Invalid mode specified: ${value}. Defaulting to '${mode}'.`);
            }
        }
    });
    logger.info(`Server mode selected: ${mode.toUpperCase()}`);
    return mode;
};

// --- Common Server Setup ---
/**
 * Creates and configures the common MCP Server instance.
 * @param logger - The logger instance.
 * @param mode - The current operating mode.
 * @returns The configured McpServer instance.
 */
function createAndConfigureMcpServer(logger: Logger, mode: Mode): McpServer {
    const commonCapabilities = {
        tools: {}, // Intentially starts empty
        // Add other capabilities shared between modes
    };

    logger.info(`Initializing McpServer: ${serverInfo.name} v${serverInfo.version}`);
    const mcpServer = new McpServer({
        name: serverInfo.name,
        version: serverInfo.version,
    }, {
        capabilities: commonCapabilities,
        // Potentially add serverInfo here if needed by the constructor signature
        // serverInfo: serverInfo,
    });

    // Set up the ListToolsRequest handler
    logger.info("Setting up ListToolsRequest handler...");
    mcpServer.setRequestHandler(ListToolsRequestSchema, async () => {
        logger.info("Received ListToolsRequest");
        const tools = await fetchToolInformation(logger);

        // Add our custom resource fetcher tool
        tools.push({
            name: "fetch_resource",
            description: "Fetch a resource by name or search string",
            inputSchema: {
                type: "object",
                properties: {
                    query: {
                        type: "string",
                        description: "Resource name or search string to find the resource"
                    }
                },
                required: ["query"]
            },
            annotations: {
                title: "Fetch Resource",
                readOnlyHint: true,
                openWorldHint: false
            }
        });

        logger.info(`Responding with ${tools.length} tools`);
        return { tools };
    });
    logger.info("ListToolsRequest handler set.");

    // Set up the CallToolRequest handler
    logger.info("Setting up CallToolRequest handler...");
    mcpServer.setRequestHandler(CallToolRequestSchema, async (request) => {
        const { name, arguments: args } = request.params;
        logger.info(`Received CallToolRequest for tool: ${name}`, args);

        // Handle tool calls based on the tool name
        switch (name) {
            case "calculate_sum":
                const { a, b } = args as { a: number, b: number };
                const sum = a + b;
                logger.info(`Calculated sum: ${sum}`);
                return {
                    content: [
                        {
                            type: "text",
                            text: String(sum)
                        }
                    ]
                };
            case "fetch_resource":
                const { query } = args as { query: string };
                logger.info(`Fetching resource with query: ${query}`);

                // Simple resource matching logic
                let resourceContent = "";
                let resourceName = "";

                // Mock resources (you could replace this with actual resource fetching logic)
                if (query.toLowerCase().includes("greeting") || query.toLowerCase().includes("hello")) {
                    resourceName = "Greeting";
                    resourceContent = "Hello, world!";
                } else if (query.toLowerCase().includes("readme") || query.toLowerCase().includes("documentation")) {
                    resourceName = "README Document";
                    resourceContent = "# Project README\n\nWelcome to the Vrooli MCP Server project.\n\nThis is a mock README document served through MCP.";
                } else if (query.toLowerCase().includes("status") || query.toLowerCase().includes("system")) {
                    resourceName = "System Status";
                    const statusData = {
                        status: "operational",
                        uptime: "3 days, 4 hours",
                        memory: {
                            used: "1.2 GB",
                            available: "4.8 GB"
                        }
                    };
                    resourceContent = JSON.stringify(statusData, null, 2);
                } else {
                    resourceName = "Resource Not Found";
                    resourceContent = `No resource found matching query: "${query}"`;
                }

                logger.info(`Returning resource: ${resourceName}`);
                return {
                    content: [
                        {
                            type: "text",
                            text: `Resource: ${resourceName}\n\n${resourceContent}`
                        }
                    ]
                };
            // Other tool implementations would go here
            default:
                logger.error(`Tool not implemented: ${name}`);
                return {
                    isError: true,
                    content: [
                        {
                            type: "text",
                            text: `Error: Tool '${name}' not implemented`
                        }
                    ]
                };
        }
    });
    logger.info("CallToolRequest handler set.");

    return mcpServer;
};


// --- SSE Mode Setup ---
/**
 * Sets up and runs the server in SSE mode using Express.
 * @param mcpServer - The McpServer instance.
 * @param logger - The logger instance.
 */
function setupSse(mcpServer: McpServer, logger: Logger): void {
    logger.info(`Initializing SSE mode on port ${SSE_PORT}...`);

    const app = express();

    // --- Health Check Endpoint (SSE specific) ---
    app.get('/health', (req, res) => {
        const healthInfo = getSseHealthInfo();
        logger.info(`Health check requested: ${JSON.stringify(healthInfo)}`);
        res.json(healthInfo);
    });

    // --- SSE Connection Handling ---
    app.get('/sse', (req, res) => {
        logger.info("SSE connection requested.");

        // Store the response object for tracking
        connections.add(res);
        logger.info(`Client connected. Total clients: ${connections.size}`);

        // Setup heartbeat
        const heartbeatInterval = setInterval(() => {
            if (connections.has(res)) {
                try {
                    res.write(': heartbeat\n\n');
                    // logger.debug('Sent heartbeat comment'); // Optional: uncomment for debugging
                } catch (error) {
                    logger.error('Error sending heartbeat:', error);
                    clearInterval(heartbeatInterval);
                    connections.delete(res);
                    const transport = transports.get(res);
                    if (transport) {
                        transports.delete(res);
                        // Consider transport.disconnect() or mcpServer.disconnect(transport);
                    }
                }
            } else {
                clearInterval(heartbeatInterval);
                logger.debug('Heartbeat stopped for disconnected client.');
            }
        }, 30000);

        // Create and connect the SSE transport for this connection
        const newTransport = new SSEServerTransport(SSE_MESSAGE_PATH, res);
        transports.set(res, newTransport);

        mcpServer.connect(newTransport)
            .then(() => {
                logger.info(`McpServer connected to transport for a client.`);
                // Handshake ($/server/ready) is sent automatically by SSEServerTransport
            })
            .catch(error => {
                logger.error(`Error connecting McpServer to transport for a client:`, error);
                if (!res.writableEnded) {
                    res.end();
                }
                connections.delete(res);
                transports.delete(res); // Clean up transport map as well
            });

        req.on('close', () => {
            logger.info(`Client disconnected.`);
            clearInterval(heartbeatInterval);
            connections.delete(res);
            const closedTransport = transports.get(res);
            if (closedTransport) {
                logger.info("Cleaning up transport for closed connection.");
                // Consider closedTransport.disconnect() or mcpServer.disconnect(closedTransport);
                transports.delete(res);
            }
            logger.info(`Client connection closed. Total clients: ${connections.size}`);
        });

        res.on('error', (error) => {
            logger.error(`Error on response stream for a client:`, error);
            clearInterval(heartbeatInterval);
            connections.delete(res);
            const errorTransport = transports.get(res);
            if (errorTransport) {
                logger.info("Cleaning up transport due to stream error.");
                // Consider errorTransport.disconnect() or mcpServer.disconnect(errorTransport);
                transports.delete(res);
            }
            logger.info(`Removed client due to stream error. Total clients: ${connections.size}`);
        });
    });

    // Route to handle incoming MCP messages POSTed from the client
    // Re-adding the POST handler, but WITHOUT express.json() middleware
    // as the SSEServerTransport likely handles parsing the raw request.
    app.post(SSE_MESSAGE_PATH, (req, res) => {
        logger.info(`Received POST on ${SSE_MESSAGE_PATH}`);
        // Find *any* active transport instance to call the method.
        // This part relies on the SDK's design of handlePostMessage.
        // If it requires instance-specific context not available here,
        // this approach might need refinement based on SDK docs.
        const [anyTransport] = transports.values(); // Assuming handlePostMessage can route or is static-like
        if (anyTransport) {
            try {
                // Pass raw request and response to the handler.
                anyTransport.handlePostMessage(req, res);
                logger.info(`Handed POST request to an SSEServerTransport instance.`);
            } catch (error) {
                logger.error(`Error handling POST request in SSEServerTransport:`, error);
                // Attempt to send a JSON-RPC error response if possible
                if (!res.headersSent) {
                    res.status(500).json({ // Assuming JSON response is acceptable for errors
                        jsonrpc: jsonRpcVersion,
                        error: { code: -32000, message: "Internal server error handling POST" },
                        id: null // Cannot safely get ID without parsing body
                    });
                } else if (!res.writableEnded) {
                    res.end(); // End the stream if headers were sent but an error occurred
                }
            }
        } else {
            logger.error(`Received POST on ${SSE_MESSAGE_PATH} but no active SSE transports found.`);
            res.status(503).json({ // Service Unavailable
                jsonrpc: jsonRpcVersion,
                error: {
                    code: -32000,
                    message: "Server not ready or no active transports"
                },
                id: null // Cannot safely get ID without parsing body
            });
        }
    });

    // Optional: Root path handler
    app.get('/', (req, res) => {
        res.send(`${serverInfo.name} v${serverInfo.version} is running. Use /sse for MCP connection.`);
    });

    const httpServer = app.listen(SSE_PORT, () => {
        logger.info(`${serverInfo.name} listening on http://localhost:${SSE_PORT}`);
        logger.info(`SSE endpoint available at http://localhost:${SSE_PORT}/sse`);
        logger.info(`MCP message endpoint at http://localhost:${SSE_PORT}${SSE_MESSAGE_PATH}`);
    });

    httpServer.on('error', (error) => {
        logger.error("HTTP server error:", error);
        process.exit(1); // Exit if the server fails to start
    });

    // Graceful shutdown for SSE mode
    const shutdown = () => {
        logger.info('Gracefully shutting down SSE server...');
        connections.forEach(res => {
            logger.info('Closing client connection.');
            if (!res.writableEnded) {
                res.end();
            }
        });
        connections.clear(); // Clear the set after closing
        transports.clear(); // Clear transports map

        httpServer.close((err) => {
            if (err) {
                logger.error('Error closing HTTP Server:', err);
            } else {
                logger.info('HTTP Server closed.');
            }
            // Optionally disconnect the McpServer if needed, though transports are cleared
            // mcpServer.disconnect(); // Consider if McpServer needs global disconnect
            logger.info('Shutdown complete.');
            process.exit(err ? 1 : 0);
        });

        // Force shutdown after a timeout
        setTimeout(() => {
            logger.error('Could not close connections gracefully, forcing shutdown.');
            process.exit(1);
        }, 5000); // 5 seconds timeout
    };

    process.on('SIGINT', shutdown);
    process.on('SIGTERM', shutdown); // Handle SIGTERM as well
};

// --- STDIO Mode Setup ---
/**
 * Sets up and runs the server in STDIO mode.
 * @param mcpServer - The McpServer instance.
 * @param logger - The logger instance.
 */
async function setupStdio(mcpServer: McpServer, logger: Logger): Promise<void> {
    logger.info(`Initializing STDIO mode...`);

    // STDIO mode capabilities might differ, adjust mcpServer capabilities if needed
    // mcpServer.setCapabilities({ ... }); // If dynamic capability setting is supported

    logger.info("Preparing StdioServerTransport...");
    const transport = new StdioServerTransport();
    logger.info("StdioServerTransport created.");

    logger.info("Attempting to connect server transport...");
    try {
        await mcpServer.connect(transport);
        logger.info("Server transport connected successfully.");
        logger.info("Server is running. Waiting for requests via stdin/stdout...");
    } catch (error) {
        logger.error("Failed to connect server transport:", error);
        process.exit(1); // Exit if connection fails
    }

    // Graceful shutdown for STDIO mode
    const shutdown = () => {
        logger.info('Received termination signal. Shutting down STDIO server...');
        // The SDK might handle cleanup, or you might need specific disconnection logic
        // mcpServer.disconnect(transport); // Disconnect specific transport if needed
        // Or global disconnect: mcpServer.disconnect();
        logger.info('Exiting.');
        process.exit(0);
    };
    process.on('SIGINT', shutdown);
    process.on('SIGTERM', shutdown); // Handle SIGTERM as well
};

// --- Main Function ---
/**
 * Main entry point for the server application.
 */
async function main(): Promise<void> {
    const logger = createLogger();
    try {
        const mode = getModeFromArgs(logger);
        const mcpServer = createAndConfigureMcpServer(logger, mode);

        if (mode === 'sse') {
            setupSse(mcpServer, logger); // This function now handles its own lifecycle and exit
        } else {
            await setupStdio(mcpServer, logger); // This function now handles its own lifecycle and exit
        }
        // Keep the process alive implicitly by the running server (SSE) or waiting transport (STDIO)
    } catch (error) {
        logger.error("An unexpected error occurred during server setup:", error);
        process.exit(1);
    }
};

// --- Execution ---
// Use a self-invoking async function or top-level await if supported
(async () => {
    await main();
})().catch(error => {
    // Catch any unhandled promise rejections from main() itself, though errors inside should be caught
    console.error("Unhandled error during server startup:", error); // Use console directly as logger might not be initialized
    process.exit(1);
});