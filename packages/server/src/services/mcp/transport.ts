import { type Server as McpServer } from "@modelcontextprotocol/sdk/server/index.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import { HttpStatus } from "@vrooli/shared";
import { execSync } from "child_process";
import type * as http from "http";
import path from "path";
import { fileURLToPath } from "url";
import { logger } from "../../events/logger.js";

/**
 * Manages all transport connections for the MCP server
 */
export class TransportManager {
    private transports = new Map<http.ServerResponse, SSEServerTransport>();
    private connections = new Set<http.ServerResponse>();
    private heartbeatIntervals = new Map<http.ServerResponse, NodeJS.Timeout>();
    private heartbeatIntervalMs: number;
    private messagePath: string;
    private mcpServer: McpServer;

    constructor(
        mcpServer: McpServer,
        options: {
            heartbeatInterval: number,
            messagePath: string
        },
    ) {
        this.mcpServer = mcpServer;
        this.heartbeatIntervalMs = options.heartbeatInterval;
        this.messagePath = options.messagePath;
    }

    /**
     * Handles a new SSE connection
     * @param req HTTP request object
     * @param res HTTP response object
     */
    async handleSseConnection(req: http.IncomingMessage, res: http.ServerResponse): Promise<void> {
        logger.info("SSE connection requested.");

        // Store the response object for tracking
        this.connections.add(res);
        logger.info(`Client connected. Total clients: ${this.connections.size}`);

        // Set up heartbeat
        const heartbeatInterval = setInterval(() => {
            if (this.connections.has(res)) {
                try {
                    res.write(": heartbeat\n\n");
                } catch (error) {
                    logger.error("Error sending heartbeat:", error);
                    this.cleanupConnection(res);
                }
            } else {
                clearInterval(heartbeatInterval);
                logger.debug("Heartbeat stopped for disconnected client.");
            }
        }, this.heartbeatIntervalMs);

        this.heartbeatIntervals.set(res, heartbeatInterval);

        // Create and connect the SSE transport for this connection
        const newTransport = new SSEServerTransport(this.messagePath, res);
        this.transports.set(res, newTransport);

        try {
            await this.mcpServer.connect(newTransport);
            logger.info("McpServer connected to transport for a client.", {
                endpoint: newTransport["_endpoint"],
            });
            // Handshake is sent automatically by SSEServerTransport
        } catch (error) {
            logger.error("Error connecting McpServer to transport for a client:", error);
            this.cleanupConnection(res);
        }

        // Set up connection cleanup on close and error
        req.on("close", () => {
            logger.info("Client disconnected.");
            this.cleanupConnection(res);
            logger.info(`Client connection closed. Total clients: ${this.connections.size}`);
        });

        res.on("error", (error) => {
            logger.error("Error on response stream for a client:", error);
            this.cleanupConnection(res);
            logger.info(`Removed client due to stream error. Total clients: ${this.connections.size}`);
        });
    }

    /**
     * Handles an incoming POST request to the message path
     * @param req HTTP request object
     * @param res HTTP response object
     */
    handlePostMessage(req: http.IncomingMessage, res: http.ServerResponse): void {
        logger.info(`Received POST on ${this.messagePath}`);

        // Find an active transport to handle the message
        const firstTransport = [...this.transports.values()][0];

        if (firstTransport) {
            try {
                firstTransport.handlePostMessage(req, res);
                logger.info("Handed POST request to an SSEServerTransport instance.");
            } catch (error) {
                logger.error("Error handling POST request in SSEServerTransport:", error);
                this.sendJsonRpcError(res, -32000, "Internal server error handling POST");
            }
        } else {
            logger.error(`Received POST on ${this.messagePath} but no active SSE transports found.`);
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
     * Returns health information about SSE connections and Claude Code registration
     * @returns Object with health status
     */
    getHealthInfo(): Record<string, any> {
        const healthInfo: Record<string, any> = {
            status: "ok",
            activeConnections: this.connections.size,
        };

        // Add Claude Code registration status
        try {
            healthInfo.claudeCodeIntegration = this.getClaudeCodeRegistrationStatus();
        } catch (error) {
            logger.warn("Failed to get Claude Code registration status:", error);
            healthInfo.claudeCodeIntegration = {
                error: "Failed to check registration status",
            };
        }

        return healthInfo;
    }

    /**
     * Get Claude Code registration status by calling the bash script
     * @returns Object with registration information
     */
    private getClaudeCodeRegistrationStatus(): Record<string, any> {
        // Get __dirname equivalent for ESM
        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        
        try {
            // Construct path to the Claude Code management script
            const scriptPath = path.resolve(__dirname, "../../../scripts/resources/agents/claude-code/manage.sh");
            
            // Execute the mcp-status command with JSON format
            const statusOutput = execSync(
                `"${scriptPath}" --action mcp-status --format json`,
                { 
                    encoding: "utf8", 
                    timeout: 10000, // 10 second timeout
                    stdio: ["ignore", "pipe", "pipe"], // Ignore stdin, capture stdout/stderr
                },
            );
            
            // Parse the JSON response
            const status = JSON.parse(statusOutput.trim());
            
            return {
                available: true,
                ...status,
            };
        } catch (error) {
            // Handle various error cases
            if (error instanceof Error) {
                if (error.message.includes("ENOENT")) {
                    return {
                        available: false,
                        error: "Claude Code management script not found",
                    };
                } else if (error.message.includes("timeout")) {
                    return {
                        available: false,
                        error: "Registration status check timed out",
                    };
                } else {
                    return {
                        available: false,
                        error: `Registration check failed: ${error.message}`,
                    };
                }
            }
            
            return {
                available: false,
                error: "Unknown error checking registration status",
            };
        }
    }

    /**
     * Closes all connections for clean shutdown
     */
    async shutdown(): Promise<void> {
        logger.info("Closing all connections...");

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

        logger.info("All connections closed.");
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
