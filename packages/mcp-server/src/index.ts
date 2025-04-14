import { Server as McpServer } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { ListResourcesRequestSchema } from "@modelcontextprotocol/sdk/types.js"; // Assuming types.js based on history
import * as http from 'http';
import { URL } from 'url'; // Import URL for parsing

// Define server info and capabilities as per the guide
const serverInfo = { name: "vrooli-mcp-server", version: "0.1.0" } as const;
const protocolVersion = "2025-04-15" as const;
const jsonRpcVersion = "2.0" as const;
const SSE_PORT = 3100;

// --- Mode Selection ---
const args = process.argv.slice(2);
let mode: 'stdio' | 'sse' = 'sse';

const RpcMethod = {
    Ready: "$/server/ready",
    Heartbeat: "$/heartbeat",
}

type RpcReadyMessage = {
    jsonrpc: typeof jsonRpcVersion,
    method: typeof RpcMethod.Ready,
    params: {
        serverInfo: typeof serverInfo,
    }
}

args.forEach(arg => {
    if (arg.startsWith('--mode=')) {
        const value = arg.split('=')[1];
        if (value === 'sse' || value === 'stdio') {
            mode = value;
        } else {
            console.warn(`Invalid mode specified: ${value}. Defaulting to 'stdio'.`);
        }
    }
});

console.log(`Starting server in ${mode.toUpperCase()} mode...`);

// --- Server Implementation ---

if (mode === 'sse') {
    // ============================
    //      SSE Mode (HTTP)
    // ============================
    console.log(`Initializing ${serverInfo.name} v${serverInfo.version} for SSE on port ${SSE_PORT}...`);

    const capabilities = { tools: true, resources: true, prompts: false }; // Capabilities for SSE handshake

    // Store active SSE connections
    const clients = new Map<string, http.ServerResponse>();
    let clientIdCounter = 0;

    const server = http.createServer((req, res) => {
        const requestUrl = new URL(req.url ?? '', `http://${req.headers.host}`);
        const path = requestUrl.pathname;
        const method = req.method?.toUpperCase();

        console.log(`[SSE] Received request: ${method} ${path}`);

        if (method === 'GET' && path === '/sse') {
            console.log("[SSE] SSE connection requested.");

            res.writeHead(200, {
                'Content-Type': 'text/event-stream',
                'Cache-Control': 'no-cache',
                'Connection': 'keep-alive',
                'Access-Control-Allow-Origin': '*' // Adjust as needed
            });
            res.flushHeaders();

            const clientId = `client-${clientIdCounter++}`;
            clients.set(clientId, res);
            console.log(`[SSE] Client ${clientId} connected.`);

            const handshakeMessage: RpcReadyMessage = {
                jsonrpc: jsonRpcVersion,
                method: RpcMethod.Ready,
                params: {
                    serverInfo: serverInfo,
                }
            };
            const sseFormattedHandshake = `data: ${JSON.stringify(handshakeMessage)}\n\n`;
            res.write(sseFormattedHandshake);
            console.log(`[SSE] Sent $/server/ready handshake to ${clientId}:`, handshakeMessage);

            const heartbeatInterval = setInterval(() => {
                if (clients.has(clientId)) {
                    // Send a valid JSON-RPC notification as a heartbeat
                    const heartbeatMessage = {
                        jsonrpc: jsonRpcVersion,
                        method: RpcMethod.Heartbeat,
                    };
                    const heartbeatEvent = `data: ${JSON.stringify(heartbeatMessage)}\\n\\n`;
                    try {
                        // Optional: Log the heartbeat being sent for debugging
                        // console.log(`[SSE] Sending heartbeat to ${clientId}:`, heartbeatMessage);
                        res.write(heartbeatEvent);
                    } catch (error) {
                        console.error(`[SSE] Error sending heartbeat to ${clientId}:`, error);
                        clearInterval(heartbeatInterval);
                        clients.delete(clientId);
                        console.log(`[SSE] Client ${clientId} disconnected due to error.`);
                    }
                } else {
                    clearInterval(heartbeatInterval);
                    console.log(`[SSE] Heartbeat stopped for disconnected client ${clientId}.`);
                }
            }, 30000);

            req.on('close', () => {
                console.log(`[SSE] Client ${clientId} disconnected.`);
                clearInterval(heartbeatInterval);
                clients.delete(clientId);
            });

            res.on('error', (error) => {
                console.error(`[SSE] Error on response stream for ${clientId}:`, error);
                clearInterval(heartbeatInterval);
                clients.delete(clientId);
                console.log(`[SSE] Removed client ${clientId} due to stream error.`);
            });

        } else if (method === 'POST' && path === '/') {
            console.log("[SSE] Received POST request, likely for initialize.");

            let body = '';
            req.on('data', chunk => { body += chunk.toString(); });
            req.on('end', () => {
                try {
                    const requestData = JSON.parse(body);
                    console.log("[SSE] Received POST body:", requestData);

                    if (requestData.method === 'initialize') {
                        console.log("[SSE] Handling initialize request:", requestData.params);

                        const initializeResponse = {
                            jsonrpc: "2.0",
                            id: requestData.id,
                            result: {
                                // protocolVersion: protocolVersion,
                                // capabilities: { // Echo client capabilities for now
                                //     tools: { listChanged: true },
                                //     resources: { listChanged: true },
                                //     prompts: { listChanged: true },
                                //     logging: true,
                                //     streaming: true
                                // },
                                serverInfo: serverInfo,
                                tools: [],
                                // offerings: [] // TODO: Populate offerings
                            }
                        };

                        res.writeHead(200, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify(initializeResponse));
                        console.log("[SSE] Sent initialize response:", initializeResponse);

                    } else {
                        console.warn("[SSE] Received unhandled POST method:", requestData.method);
                        res.writeHead(400, { 'Content-Type': 'application/json' });
                        res.end(JSON.stringify({ jsonrpc: "2.0", error: { code: -32601, message: "Method not found" }, id: requestData.id || null }));
                    }
                } catch (error) {
                    console.error("[SSE] Error parsing POST body or handling request:", error);
                    res.writeHead(400, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify({ jsonrpc: "2.0", error: { code: -32700, message: "Parse error" }, id: null }));
                }
            });

        } else {
            console.log(`[SSE] Unhandled request: ${method} ${path}`);
            res.writeHead(404, { 'Content-Type': 'text/plain' });
            res.end('Not Found');
        }
    });

    server.on('error', (error) => {
        console.error("[SSE] Server error:", error);
    });

    server.listen(SSE_PORT, () => {
        console.log(`[SSE] ${serverInfo.name} listening on http://localhost:${SSE_PORT}`);
        console.log(`[SSE] SSE endpoint available at http://localhost:${SSE_PORT}/sse`);
    });

    process.on('SIGINT', () => {
        console.log('\n[SSE] Gracefully shutting down...');
        server.close(() => {
            console.log('[SSE] Server closed.');
            process.exit(0);
        });
        clients.forEach((clientRes, clientId) => {
            console.log(`[SSE] Closing connection for ${clientId}`);
            clientRes.end();
        });
    });

} else {
    // ============================
    //     STDIO Mode (SDK)
    // ============================
    console.log(`Initializing ${serverInfo.name} v${serverInfo.version} for STDIO...`);

    // Capabilities for SDK server initialization
    const sdkCapabilities = {
        resources: {} // Define required capabilities for SDK
        // Add other capabilities like 'tools' if needed by the SDK setup
    };

    const mcpServer = new McpServer({
        name: serverInfo.name,
        version: serverInfo.version
        // protocolVersion can often be inferred or set via options if needed
    }, {
        capabilities: sdkCapabilities
    });

    console.log("[STDIO] Server instance created with SDK.");

    // Handle requests (Example: ListResources)
    console.log("[STDIO] Setting up request handlers...");
    mcpServer.setRequestHandler(ListResourcesRequestSchema, async (request) => {
        console.log("[STDIO] Received ListResourcesRequest:", request);
        const resources = [
            {
                uri: "example://resource-stdio", // Different URI for clarity
                name: "Example STDIO Resource"
            }
        ];
        console.log("[STDIO] Responding with resources:", resources);
        return { resources };
    });
    console.log("[STDIO] ListResourcesRequest handler set.");

    // Connect transport
    async function connectStdio() {
        console.log("[STDIO] Preparing server transport...");
        const transport = new StdioServerTransport();
        console.log("[STDIO] StdioServerTransport created.");

        console.log("[STDIO] Attempting to connect server transport...");
        try {
            await mcpServer.connect(transport);
            console.log("[STDIO] Server transport connected successfully.");
            console.log("[STDIO] Server is running. Waiting for requests via stdin/stdout...");
        } catch (error) {
            console.error("[STDIO] Failed to connect server transport:", error);
            process.exit(1); // Exit if connection fails
        }
    }

    connectStdio();

    // Graceful shutdown for STDIO mode might be simpler or handled by parent process
    process.on('SIGINT', () => {
        console.log('\n[STDIO] Received SIGINT. Exiting.');
        // The SDK might handle cleanup, or you might need specific disconnection logic
        process.exit(0);
    });
}