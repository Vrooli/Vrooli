/**
 * Main entry point for the Vrooli MCP Server
 * This server provides tools and resources to LLMs like Claude in Cursor
 */
import cors from 'cors';
import dotenv from 'dotenv';
import express from 'express';
import http from 'http';
import os from 'os';
import path from 'path';
import { fileURLToPath } from 'url';

// Import tools
import { projectInfo } from './tools/project.js';
import { searchRoutines } from './tools/search.js';

// Import resources
import { readmeResource } from './resources/readme.js';
import { projectStructureResource } from './resources/structure.js';

// Set up __dirname equivalent for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Load environment variables
dotenv.config();

// Server configuration
const PORT = process.env.MCP_SERVER_PORT || 3100;

// Create express app
const app = express();
app.use(cors());
app.use(express.json());

// Create HTTP server
const server = http.createServer(app);

// Create a logger utility to standardize logs with timestamps
const logger = {
    info: (message, ...args) => {
        const timestamp = new Date().toISOString();
        console.log(`[${timestamp}] [INFO] ${message}`, ...args);
    },
    error: (message, ...args) => {
        const timestamp = new Date().toISOString();
        console.error(`[${timestamp}] [ERROR] ${message}`, ...args);
    },
    debug: (message, ...args) => {
        const timestamp = new Date().toISOString();
        console.log(`[${timestamp}] [DEBUG] ${message}`, ...args);
    },
    warn: (message, ...args) => {
        const timestamp = new Date().toISOString();
        console.warn(`[${timestamp}] [WARN] ${message}`, ...args);
    }
};

// Basic MCP server implementation
class McpServer {
    constructor(config) {
        this.id = config.id;
        this.name = config.name;
        this.description = config.description;
        this.version = config.version;
        this.capabilities = config.capabilities || {
            tools: true,
            resources: true,
            prompts: false
        };

        // Register tools and resources
        this.tools = config.tools || [];
        this.resources = config.resources || [];
        this.prompts = config.prompts || [];

        // Map tools and resources by ID for easy lookup
        this.toolsMap = new Map();
        this.resourcesMap = new Map();

        this.tools.forEach(tool => this.toolsMap.set(tool.id, this.formatTool(tool)));
        this.resources.forEach(resource => this.resourcesMap.set(resource.id, this.formatResource(resource)));
    }

    // Handle incoming messages
    async handleRawMessage(message) {
        try {
            try {
                // Log the raw message to help debug malformed JSON
                logger.debug(`Attempting to parse message: ${typeof message === 'string' ? message.substring(0, 200) : 'Non-string message'}`);
                // If message is already an object (from express.json()), use it directly. Otherwise, parse.
                const request = typeof message === 'string' ? JSON.parse(message) : message;
                logger.info(`Received message method: ${request.method}, id: ${request.id}`);

                if (request.params) {
                    logger.debug(`Message params: ${JSON.stringify(request.params)}`);
                }

                // Validate structure as per MCP protocol
                if (!request.method) {
                    logger.warn('Received message without method field');
                    return JSON.stringify({
                        error: {
                            code: -32600,
                            message: "Invalid Request: Missing 'method' field"
                        },
                        id: request.id || null
                    });
                }

                if (request.jsonrpc !== '2.0') {
                    logger.warn(`Unexpected or missing jsonrpc version: ${request.jsonrpc}`);
                }

                // Handle different message types
                switch (request.method) {
                    case 'initialize':
                        return this.handleInitialize(request);
                    case 'initialized':
                        logger.info('Handling initialized confirmation');
                        return JSON.stringify({ result: null });
                    case 'tools/list':
                        return this.handleToolsList(request);
                    case 'tools/call':
                        return this.handleToolsCall(request);
                    case 'resources/list':
                        // Although we send resources in initialize, handle list for completeness
                        return this.handleResourcesList(request);
                    case 'resources/get':
                        return this.handleResourcesGet(request);
                    case 'shutdown':
                        logger.info('Handling shutdown request');
                        return JSON.stringify({ result: null });
                    default:
                        logger.warn(`Unknown method requested: ${request.method}`);
                        return JSON.stringify({
                            error: {
                                code: -32601,
                                message: `Method not found: ${request.method}`
                            },
                            id: request.id
                        });
                }
            } catch (parseError) {
                logger.error('JSON parse error:', parseError);
                logger.debug('Raw message that failed to parse:', message);
                return JSON.stringify({
                    error: {
                        code: -32700,
                        message: `Parse error: ${parseError.message}`
                    },
                    id: null // Cannot determine ID for malformed JSON
                });
            }
        } catch (error) {
            logger.error(`Error handling message (method: ${request?.method}, id: ${request?.id}):`, error);
            // Ensure message is stringified for logging if it's an object
            const messageContent = typeof message === 'string' ? message : JSON.stringify(message);
            logger.debug(`Raw message content that caused error: ${messageContent.substring(0, 500)}`);
            return JSON.stringify({
                error: {
                    code: -32603,
                    message: `Internal error: ${error.message}`
                },
                id: request?.id || null
            });
        }
    }

    // Handle initialize request
    handleInitialize(request) {
        logger.info('Handling initialize request with params:', JSON.stringify(request.params));

        // Based on report, respond with detailed capabilities and server info
        // Include the list of tools/resources directly in the initialize response

        const protocolVersion = request.params?.protocolVersion || "2025-03-26"; // Use client's version or default

        // Format tools and resources for the response
        const toolsList = Array.from(this.toolsMap.values());
        const resourcesList = Array.from(this.resourcesMap.values());

        return JSON.stringify({
            jsonrpc: "2.0",
            id: request.id,
            result: {
                protocolVersion: protocolVersion, // Echo back protocol version
                capabilities: { // Detailed capabilities structure
                    tools: { listChanged: true }, // Indicate tools are available
                    resources: { listChanged: true, subscribe: false }, // Indicate resources are available
                    prompts: { listChanged: false }, // Indicate prompts are not available
                    logging: true, // Example capability
                    sampling: false // Example capability
                },
                serverInfo: {
                    // Include offerings structure if needed, or keep flat
                    offerings: [ // Summarize features
                        { type: "tools", items: toolsList.map(t => ({ id: t.id, name: t.name })) },
                        { type: "resources", items: resourcesList.map(r => ({ id: r.id, name: r.name })) }
                    ],
                    tools: toolsList, // Full tool definitions
                    resources: resourcesList, // Full resource definitions (optional, but good practice)
                    name: this.name,
                    version: this.version
                }
            }
        });
    }

    // Handle tools/list request
    handleToolsList(request) {
        console.log('Handling tools/list request');

        // Format according to standard: result should be the list directly
        const toolsList = this.tools.map(tool => ({
            id: tool.id,
            name: tool.name,
            description: tool.description,
            parameters: tool.parameters
        }));

        return JSON.stringify({
            jsonrpc: "2.0",
            result: { tools: toolsList }, // Often nested under 'tools'
            id: request.id
        });
    }

    // Handle tools/call request
    async handleToolsCall(request) {
        console.log('Handling tools/call request', request.params?.id);

        const toolId = request.params?.id;
        const params = request.params?.params || {};

        if (!toolId) {
            return JSON.stringify({
                error: {
                    code: -32602,
                    message: 'Invalid params: Missing tool id'
                },
                id: request.id
            });
        }

        const tool = this.toolsMap.get(toolId);

        if (!tool) {
            return JSON.stringify({
                error: {
                    code: -32602,
                    message: `Tool not found: ${toolId}`
                },
                id: request.id
            });
        }

        try {
            const result = await tool.handler(params);

            return JSON.stringify({
                result,
                id: request.id
            });
        } catch (error) {
            console.error(`Error calling tool ${toolId}:`, error);

            return JSON.stringify({
                error: {
                    code: -32603,
                    message: `Tool execution error: ${error.message}`
                },
                id: request.id
            });
        }
    }

    // Handle resources/list request
    handleResourcesList(request) {
        console.log('Handling resources/list request');

        // Format according to standard
        const resourcesList = this.resources.map(resource => ({
            id: resource.id,
            name: resource.name,
            description: resource.description,
            parameters: resource.parameters
        }));

        return JSON.stringify({
            jsonrpc: "2.0",
            result: { resources: resourcesList }, // Often nested under 'resources'
            id: request.id
        });
    }

    // Handle resources/get request
    async handleResourcesGet(request) {
        console.log('Handling resources/get request', request.params?.id);

        const resourceId = request.params?.id;
        const params = request.params?.params || {};

        if (!resourceId) {
            return JSON.stringify({
                error: {
                    code: -32602,
                    message: 'Invalid params: Missing resource id'
                },
                id: request.id
            });
        }

        const resource = this.resourcesMap.get(resourceId);

        if (!resource) {
            return JSON.stringify({
                error: {
                    code: -32602,
                    message: `Resource not found: ${resourceId}`
                },
                id: request.id
            });
        }

        try {
            const result = await resource.handler(params);

            return JSON.stringify({
                result,
                id: request.id
            });
        } catch (error) {
            console.error(`Error getting resource ${resourceId}:`, error);

            return JSON.stringify({
                error: {
                    code: -32603,
                    message: `Resource retrieval error: ${error.message}`
                },
                id: request.id
            });
        }
    }

    // Helper to format tool definition consistently
    formatTool(tool) {
        return {
            id: tool.id, // Assuming tool.id exists
            name: tool.name,
            description: tool.description,
            parameters: tool.parameters, // Ensure this matches expected JSON Schema format if needed
            // Add inputSchema if required by newer specs
            // inputSchema: tool.inputSchema || { type: "object", properties: tool.parameters } 
        };
    }

    // Helper to format resource definition consistently
    formatResource(resource) {
        return {
            id: resource.id,
            name: resource.name,
            description: resource.description,
            parameters: resource.parameters // Parameters for resource access
        };
    }
}

// Create MCP server
const mcpServer = new McpServer({
    id: 'vrooli-mcp-server',
    name: 'Vrooli MCP Server',
    description: 'MCP server for accessing Vrooli project data and functionality',
    version: '0.1.0',

    // Define capabilities
    capabilities: {
        tools: true,
        resources: true,
        prompts: false,
    },

    // Register tools
    tools: [
        projectInfo,
        searchRoutines
    ],

    // Register resources
    resources: [
        readmeResource,
        projectStructureResource
    ]
})

// Add HTTP request logging middleware (moved earlier for better coverage)
app.use((req, res, next) => {
    const start = Date.now();
    // Log when the request completes
    res.on('finish', () => {
        const duration = Date.now() - start;
        logger.info(`${req.method} ${req.originalUrl} ${res.statusCode} - ${duration}ms`);
    });
    next();
});

// Setup SSE endpoint for server-to-client notifications
app.get('/sse', (req, res) => {
    const clientIp = req.ip || req.socket.remoteAddress;
    logger.info(`SSE connection established from ${clientIp}`);
    logger.debug('SSE request headers:', JSON.stringify(req.headers));

    // Set SSE headers
    res.setHeader('Content-Type', 'text/event-stream');
    res.setHeader('Cache-Control', 'no-cache');
    res.setHeader('Connection', 'keep-alive');
    res.setHeader('Access-Control-Allow-Origin', '*'); // Allow CORS if needed
    res.flushHeaders(); // Send headers immediately

    // Send the initial $/server/ready notification
    // Use the correct SSE format: data: <json>\n\n
    const serverReadyMessage = {
        jsonrpc: "2.0",
        method: "$/server/ready",
        params: {
            // Include basic server info needed before initialize call
            // Capabilities might be sent fully in initialize response
            serverInfo: {
                name: mcpServer.name,
                version: mcpServer.version
            },
            capabilities: mcpServer.capabilities // Send initial caps here too
        }
    };
    res.write(`data: ${JSON.stringify(serverReadyMessage)}\n\n`);
    logger.info(`Sent MCP server/ready notification: ${JSON.stringify(serverReadyMessage)}`);

    // Keep connection alive with simple SSE comments (":") as pings
    const pingInterval = setInterval(() => {
        if (req.socket.destroyed) {
            logger.warn('SSE socket destroyed, clearing ping interval.');
            clearInterval(pingInterval);
            return;
        }
        try {
            res.write(':\n\n'); // Simple comment ping
            logger.debug('Sent SSE keep-alive ping');
        } catch (error) {
            logger.error('Error sending SSE keep-alive ping:', error);
            clearInterval(pingInterval);
        }
    }, 30000);

    // Handle client disconnection
    req.on('close', () => {
        logger.info('SSE client disconnected');
        clearInterval(pingInterval); // Stop pinging
    });

    // Handle connection errors
    req.on('error', (error) => {
        logger.error('SSE connection error:', error);
        clearInterval(pingInterval);
    });

    // DO NOT handle incoming requests (req.on('data')) here.
    // JSON-RPC requests will come via POST.
});

// Add POST endpoint for JSON-RPC requests from the client
app.post('/sse', async (req, res) => {
    const clientIp = req.ip || req.socket.remoteAddress;
    // req.body is populated by express.json() middleware
    const requestBody = req.body;

    if (!requestBody || typeof requestBody !== 'object') {
        logger.error(`Received invalid POST request body from ${clientIp}:`, requestBody);
        return res.status(400).json({
            jsonrpc: "2.0",
            error: { code: -32700, message: "Parse error: Invalid JSON received." },
            id: null
        });
    }

    logger.info(`Received POST request from ${clientIp} for method: ${requestBody.method}`);
    logger.debug(`POST request body: ${JSON.stringify(requestBody)}`);

    try {
        // Pass the parsed request object directly to the handler
        const responseJsonString = await mcpServer.handleRawMessage(requestBody);
        // Ensure response is valid JSON before sending
        try {
            const jsonResponse = JSON.parse(responseJsonString);
            res.status(200).json(jsonResponse);
            logger.debug(`Sent POST response (${responseJsonString.length} bytes)`);
        } catch (parseError) {
            logger.error('Failed to parse MCP server response:', parseError, 'Original string:', responseJsonString);
            // Send a generic internal server error if our own response is malformed
            res.status(500).json({
                jsonrpc: "2.0",
                error: { code: -32603, message: "Internal error: Failed to generate valid response." },
                id: requestBody.id || null
            });
        }
    } catch (error) {
        // Catch errors from handleRawMessage itself
        logger.error('Error processing POST request:', error);
        // Attempt to send a JSON-RPC error response
        res.status(500).json({
            jsonrpc: "2.0",
            error: {
                code: -32603,
                message: `Internal server error: ${error.message}`
            },
            id: requestBody.id || null
        });
    }
});

// Enhanced health check endpoint with more information
app.get('/health', (req, res) => {
    const uptime = process.uptime();
    const memory = process.memoryUsage();

    res.status(200).json({
        status: 'OK',
        server: 'Vrooli MCP Server',
        version: mcpServer.version,
        uptime: `${Math.floor(uptime / 60)}m ${Math.floor(uptime % 60)}s`,
        memory: {
            rss: `${Math.round(memory.rss / 1024 / 1024)} MB`,
            heapTotal: `${Math.round(memory.heapTotal / 1024 / 1024)} MB`,
            heapUsed: `${Math.round(memory.heapUsed / 1024 / 1024)} MB`
        },
        // Removed WebSocket connection count
    });

    logger.info(`Health check from ${req.ip}`);
});

// Start server
server.listen(PORT, () => {
    logger.info(`Vrooli MCP Server running on port ${PORT}`);
    logger.info(`SSE endpoint available at http://localhost:${PORT}/sse`);
    logger.info(`Health check available at http://localhost:${PORT}/health`);

    // Log environment info to help with debugging
    logEnvironmentInfo();
});

// Function to log environment information
function logEnvironmentInfo() {
    logger.info('=== Environment Information ===');
    logger.info(`Node.js version: ${process.version}`);
    logger.info(`Platform: ${process.platform}`);
    logger.info(`Architecture: ${process.arch}`);

    // Network interfaces (helpful for determining all available IPs)
    const networkInterfaces = os.networkInterfaces();

    logger.info('Network interfaces:');
    Object.keys(networkInterfaces).forEach((interfaceName) => {
        networkInterfaces[interfaceName].forEach((iface) => {
            if (iface.family === 'IPv4' || iface.family === 4) {
                logger.info(`  ${interfaceName}: ${iface.address}`);
            }
        });
    });

    // Log key environment variables that might affect connectivity (without sensitive values)
    logger.info('Key environment variables:');
    const relevantVars = [
        'NODE_ENV',
        'MCP_SERVER_PORT',
        'HOST',
        'HOSTNAME',
        'PORT'
    ];

    relevantVars.forEach((varName) => {
        if (process.env[varName]) {
            logger.info(`  ${varName}: ${process.env[varName]}`);
        }
    });

    // Log memory info
    const memoryUsage = process.memoryUsage();
    logger.info('Memory usage:');
    logger.info(`  RSS: ${Math.round(memoryUsage.rss / 1024 / 1024)} MB`);
    logger.info(`  Heap total: ${Math.round(memoryUsage.heapTotal / 1024 / 1024)} MB`);
    logger.info(`  Heap used: ${Math.round(memoryUsage.heapUsed / 1024 / 1024)} MB`);

    logger.info('==============================');
}

// Add this to your code, replacing the existing SIGINT handler
let shuttingDown = false;

process.on('SIGINT', function () {
    if (shuttingDown) {
        console.log('Forced shutdown');
        process.exit(1);
    }

    shuttingDown = true;
    console.log('Shutting down MCP server gracefully...');

    // Set a timeout to force exit if graceful shutdown takes too long
    const forceExit = setTimeout(() => {
        console.log('Could not close connections in time, forcefully shutting down');
        process.exit(1);
    }, 5000);

    // Avoid keeping the process open just for the timeout
    forceExit.unref();

    server.close(() => {
        console.log('Server closed');
        clearTimeout(forceExit);
        process.exit(0);
    });
});

// At the end of the file, after the existing SIGINT handler
// Log uncaught exceptions and unhandled rejections
process.on('uncaughtException', (error) => {
    logger.error('Uncaught exception:', error);
});

process.on('unhandledRejection', (reason, promise) => {
    logger.error('Unhandled rejection at:', promise, 'reason:', reason);
}); 