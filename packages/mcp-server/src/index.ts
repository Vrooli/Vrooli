import { Server as McpServer } from "@modelcontextprotocol/sdk/server/index.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js"; // Added SSE transport
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { ListResourcesRequestSchema } from "@modelcontextprotocol/sdk/types.js";
import express from 'express'; // Added express
import http from 'http'; // Keep for graceful shutdown? Or remove if express handles it. Let's remove for now.

// Define server info and capabilities as per the guide
const serverInfo = { name: "vrooli-mcp-server", version: "0.1.0" } as const;
const protocolVersion = "2025-04-15" as const; // Keep for reference, maybe used later
const jsonRpcVersion = "2.0" as const; // Keep for reference
const SSE_PORT = 3100;
const SSE_MESSAGE_PATH = '/mcp'; // Define a path for MCP messages

// --- Mode Selection ---
const args = process.argv.slice(2);
let mode: 'stdio' | 'sse' = 'sse'; // Default to sse

// Removed RpcMethod and RpcReadyMessage as handshake is handled by SDK transport now

args.forEach(arg => {
    if (arg.startsWith('--mode=')) {
        const value = arg.split('=')[1];
        if (value === 'sse' || value === 'stdio') {
            mode = value;
        } else {
            // Keep warning for invalid mode, but default is now set above
            console.warn(`Invalid mode specified: ${value}. Defaulting to '${mode}'.`);
        }
    }
});

console.log(`Starting server in ${mode.toUpperCase()} mode...`);

// --- Server Implementation ---

// Instantiate McpServer common to both modes if applicable
// Capabilities might differ slightly, or be adjusted within the mode blocks
// For now, define common base capabilities
const commonCapabilities = {
    resources: {}, // Example common capability
    // Add other capabilities shared between modes
};

const mcpServer = new McpServer({
    name: serverInfo.name,
    version: serverInfo.version,
}, {
    capabilities: commonCapabilities, // Use common capabilities
    // Potentially add serverInfo here if needed by the constructor signature
    // serverInfo: serverInfo,
});

// Example request handler (can be common if logic is identical)
console.log("[MCP Server] Setting up ListResourcesRequest handler...");
mcpServer.setRequestHandler(ListResourcesRequestSchema, async (request) => {
    console.log("[MCP Server] Received ListResourcesRequest:", request);
    // Logic might differ based on mode, but let's keep it simple for now
    const resourceUriPrefix = mode === 'sse' ? 'sse-resource' : 'stdio-resource';
    const resources = [
        {
            uri: `example://${resourceUriPrefix}-${Date.now()}`, // Dynamic URI
            name: `Example ${mode.toUpperCase()} Resource`
        }
    ];
    console.log(`[MCP Server] Responding with ${mode.toUpperCase()} resources:`, resources);
    return { resources };
});
console.log("[MCP Server] ListResourcesRequest handler set.");


if (mode === 'sse') {
    // ============================
    //      SSE Mode (Express)
    // ============================
    console.log(`Initializing ${serverInfo.name} v${serverInfo.version} for SSE on port ${SSE_PORT}...`);

    const app = express();

    // Map to store transport instances keyed by their response object
    const transports = new Map<http.ServerResponse, SSEServerTransport>();

    // Keep track of active connections for graceful shutdown
    const connections = new Set<http.ServerResponse>();

    app.get('/sse', (req, res) => {
        console.log("[SSE] SSE connection requested.");

        // Let SSEServerTransport handle setting the appropriate headers.
        // We just need to keep the connection open and add it to our tracking.

        // Store the response object for tracking and potential broadcast later
        connections.add(res);
        console.log(`[SSE] Client connected. Total clients: ${connections.size}`);

        // Create and connect the SSE transport for this specific connection
        // Pass the response object directly to the transport
        const newTransport = new SSEServerTransport(SSE_MESSAGE_PATH, res);
        transports.set(res, newTransport); // Store the transport associated with this response
        mcpServer.connect(newTransport)
            .then(() => {
                console.log(`[SSE] McpServer connected to transport for a client.`);
                // Handshake ($/server/ready) is typically sent automatically by SSEServerTransport upon connection.
            })
            .catch(error => {
                console.error(`[SSE] Error connecting McpServer to transport for a client:`, error);
                res.end(); // Close the connection if transport fails
                connections.delete(res);
            });

        // Heartbeat mechanism (optional, SDK might handle keep-alive)
        // If needed, implement a clean heartbeat like before, ensuring it uses the SDK's transport if possible or sends valid JSON-RPC pings.
        // const heartbeatInterval = setInterval(() => { ... }, 30000);

        req.on('close', () => {
            console.log(`[SSE] Client disconnected.`);
            // clearInterval(heartbeatInterval); // Clear heartbeat if implemented
            connections.delete(res);
            // Consider if the transport needs explicit disconnection/cleanup here
            // sseTransport?.disconnect(); // Or similar method if available
            const closedTransport = transports.get(res);
            if (closedTransport) {
                console.log("[SSE] Cleaning up transport for closed connection.");
                // closedTransport.disconnect(); // Call disconnect if available on the transport
                // mcpServer.disconnect(closedTransport); // Or disconnect from the server side if needed
                transports.delete(res); // Remove from map
            }
            console.log(`[SSE] Client connection closed. Total clients: ${connections.size}`);
        });

        res.on('error', (error) => {
            console.error(`[SSE] Error on response stream for a client:`, error);
            // clearInterval(heartbeatInterval); // Clear heartbeat if implemented
            connections.delete(res);
            const errorTransport = transports.get(res);
            if (errorTransport) {
                console.log("[SSE] Cleaning up transport due to stream error.");
                // errorTransport.disconnect(); // Call disconnect if available
                // mcpServer.disconnect(errorTransport);
                transports.delete(res); // Remove from map
            }
            console.log(`[SSE] Removed client due to stream error. Total clients: ${connections.size}`);
        });
    });

    // Route to handle incoming MCP messages POSTed from the client
    app.post(SSE_MESSAGE_PATH, (req, res) => {
        console.log(`[SSE] Received POST on ${SSE_MESSAGE_PATH}`);
        // Determine the correct transport. This is complex with multiple clients.
        // Option 1: If the SDK's handlePostMessage intelligently routes based on req/headers.
        // We assume this for now, but it needs verification based on SDK behavior.
        // Find *any* active transport to handle the message (assuming SDK routes it). This is likely incorrect.
        const [anyTransport] = transports.values(); // simplistic, likely wrong
        if (anyTransport) {
            // Let the SDK transport handle the incoming message
            anyTransport.handlePostMessage(req, res);
            console.log(`[SSE] Handed POST request to an SSEServerTransport.`);
            // TODO: Verify how handlePostMessage identifies the correct client SSE stream.
            // The current approach of grabbing the first transport is a placeholder.
        } else {
            console.error(`[SSE] Received POST on ${SSE_MESSAGE_PATH} but no active SSE transports found or routing unclear.`);
            res.status(503).json({ jsonrpc: "2.0", error: { code: -32000, message: "Server not ready or transport not initialized" }, id: req.body?.id || null });
        }
    });

    // Optional: Handle other routes or serve static files if needed
    app.get('/', (req, res) => {
        res.send(`${serverInfo.name} v${serverInfo.version} is running. Use /sse for MCP connection.`);
    });

    const httpServer = app.listen(SSE_PORT, () => {
        console.log(`[SSE] ${serverInfo.name} listening on http://localhost:${SSE_PORT}`);
        console.log(`[SSE] SSE endpoint available at http://localhost:${SSE_PORT}/sse`);
        console.log(`[SSE] MCP message endpoint at http://localhost:${SSE_PORT}${SSE_MESSAGE_PATH}`);
    });

    httpServer.on('error', (error) => {
        console.error("[SSE] HTTP server error:", error);
    });

    // Graceful shutdown for SSE mode
    process.on('SIGINT', () => {
        console.log('[SSE] Gracefully shutting down...');
        connections.forEach(res => {
            console.log('[SSE] Closing client connection.');
            res.end(); // End active SSE connections
        });
        httpServer.close(() => {
            console.log('[SSE] HTTP Server closed.');
            // Optionally disconnect the McpServer if needed
            // mcpServer.disconnect();
            console.log('[SSE] Shutdown complete.');
            process.exit(0);
        });
        // Set a timeout for forceful shutdown if graceful shutdown takes too long
        setTimeout(() => {
            console.error('[SSE] Could not close connections gracefully, forcing shutdown.');
            process.exit(1);
        }, 5000); // 5 seconds timeout
    });

} else {
    // ============================
    //     STDIO Mode (SDK)
    // ============================
    console.log(`Initializing ${serverInfo.name} v${serverInfo.version} for STDIO...`);

    // STDIO mode capabilities might differ, adjust mcpServer capabilities if needed
    // mcpServer.setCapabilities({ ... }); // If dynamic capability setting is supported

    // Connect transport
    async function connectStdio() {
        console.log("[STDIO] Preparing server transport...");
        const transport = new StdioServerTransport();
        console.log("[STDIO] StdioServerTransport created.");

        console.log("[STDIO] Attempting to connect server transport...");
        try {
            // Use the existing mcpServer instance
            await mcpServer.connect(transport);
            console.log("[STDIO] Server transport connected successfully.");
            console.log("[STDIO] Server is running. Waiting for requests via stdin/stdout...");
        } catch (error) {
            console.error("[STDIO] Failed to connect server transport:", error);
            process.exit(1); // Exit if connection fails
        }
    }

    connectStdio();

    // Graceful shutdown for STDIO mode
    process.on('SIGINT', () => {
        console.log('[STDIO] Received SIGINT.Shutting down...');
        // The SDK might handle cleanup, or you might need specific disconnection logic
        // mcpServer.disconnect(); // Example disconnect call if available/needed
        console.log('[STDIO] Exiting.');
        process.exit(0);
    });
}