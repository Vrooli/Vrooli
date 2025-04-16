import { HttpStatus } from "@local/shared";
import { Server as McpServer } from "@modelcontextprotocol/sdk/server/index.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import type * as http from "http";
import { Logger } from "./types.js";

/**
 * Manages all transport connections for the MCP server
 */
export class TransportManager {
    private transports = new Map<http.ServerResponse, SSEServerTransport>();
    private connections = new Set<http.ServerResponse>();
    private logger: Logger;
    private heartbeatIntervals = new Map<http.ServerResponse, NodeJS.Timeout>();
    private heartbeatIntervalMs: number;
    private messagePath: string;
    private mcpServer: McpServer;

    constructor(
        mcpServer: McpServer,
        logger: Logger,
        options: {
            heartbeatInterval: number,
            messagePath: string
        },
    ) {
        this.mcpServer = mcpServer;
        this.logger = logger;
        this.heartbeatIntervalMs = options.heartbeatInterval;
        this.messagePath = options.messagePath;
    }

    /**
     * Handles a new SSE connection
     * @param req HTTP request object
     * @param res HTTP response object
     */
    async handleSseConnection(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
        this.logger.info("SSE connection requested.");

        // Store the response object for tracking
        this.connections.add(res);
        this.logger.info(`Client connected. Total clients: ${this.connections.size}`);

        // Set up heartbeat
        const heartbeatInterval = setInterval(() => {
            if (this.connections.has(res)) {
                try {
                    res.write(": heartbeat\n\n");
                } catch (error) {
                    this.logger.error("Error sending heartbeat:", error);
                    this.cleanupConnection(res);
                }
            } else {
                clearInterval(heartbeatInterval);
                this.logger.debug("Heartbeat stopped for disconnected client.");
            }
        }, this.heartbeatIntervalMs);

        this.heartbeatIntervals.set(res, heartbeatInterval);

        // Create and connect the SSE transport for this connection
        const newTransport = new SSEServerTransport(this.messagePath, res);
        this.transports.set(res, newTransport);

        try {
            await this.mcpServer.connect(newTransport);
            this.logger.info("McpServer connected to transport for a client.", {
                endpoint: newTransport["_endpoint"],
            });
            // Handshake is sent automatically by SSEServerTransport
        } catch (error) {
            this.logger.error("Error connecting McpServer to transport for a client:", error);
            this.cleanupConnection(res);
        }

        // Set up connection cleanup on close and error
        req.on("close", () => {
            this.logger.info("Client disconnected.");
            this.cleanupConnection(res);
            this.logger.info(`Client connection closed. Total clients: ${this.connections.size}`);
        });

        res.on("error", (error) => {
            this.logger.error("Error on response stream for a client:", error);
            this.cleanupConnection(res);
            this.logger.info(`Removed client due to stream error. Total clients: ${this.connections.size}`);
        });
    }

    /**
     * Handles an incoming POST request to the message path
     * @param req HTTP request object
     * @param res HTTP response object
     */
    handlePostMessage(req: http.IncomingMessage, res: http.ServerResponse): void {
        this.logger.info(`Received POST on ${this.messagePath}`);

        // Find an active transport to handle the message
        const firstTransport = [...this.transports.values()][0];

        if (firstTransport) {
            try {
                firstTransport.handlePostMessage(req, res);
                this.logger.info("Handed POST request to an SSEServerTransport instance.");
            } catch (error) {
                this.logger.error("Error handling POST request in SSEServerTransport:", error);
                this.sendJsonRpcError(res, -32000, "Internal server error handling POST");
            }
        } else {
            this.logger.error(`Received POST on ${this.messagePath} but no active SSE transports found.`);
            this.sendJsonRpcError(res, -32000, "Server not ready or no active transports", 503);
        }
    }

    /**
     * Cleans up a connection and its associated resources
     * @param res HTTP response object to clean up
     */
    private cleanupConnection(res: http.ServerResponse): void {
        const heartbeatInterval = this.heartbeatIntervals.get(res);
        if (heartbeatInterval) {
            clearInterval(heartbeatInterval);
            this.heartbeatIntervals.delete(res);
        }

        this.connections.delete(res);
        this.transports.delete(res);

        if (!res.writableEnded) {
            res.end();
        }
    }

    /**
     * Returns health information about SSE connections
     * @returns Object with health status
     */
    getHealthInfo(): Record<string, any> {
        return {
            status: "ok",
            activeConnections: this.connections.size,
        };
    }

    /**
     * Closes all connections for clean shutdown
     */
    async shutdown(): Promise<void> {
        this.logger.info("Closing all connections...");

        // Clear all heartbeat intervals
        for (const interval of this.heartbeatIntervals.values()) {
            clearInterval(interval);
        }
        this.heartbeatIntervals.clear();

        // Close all connections
        for (const res of this.connections) {
            if (!res.writableEnded) {
                res.end();
            }
        }

        this.connections.clear();
        this.transports.clear();

        this.logger.info("All connections closed.");
    }

    /**
     * Helper to send a JSON-RPC error response
     */
    private sendJsonRpcError(
        res: http.ServerResponse,
        code: number,
        message: string,
        statusCode = HttpStatus.InternalServerError,
    ): void {
        if (!res.headersSent) {
            res.statusCode = statusCode;
            res.setHeader("Content-Type", "application/json");
            res.end(JSON.stringify({
                jsonrpc: "2.0",
                error: { code, message },
                id: null,
            }));
        } else if (!res.writableEnded) {
            res.end();
        }
    }
} 
