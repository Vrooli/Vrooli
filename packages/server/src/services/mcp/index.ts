import { type Express } from "express";
import fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";
import { logger } from "../../events/logger.js";
import { McpServerApp, McpServerMode, type ServerConfig } from "./server.js";

const SHUTDOWN_TIMEOUT_MS = 5_000;
const HEARTBEAT_INTERVAL_MS = 30_000;

// Helper function to read package.json version
function getPackageVersion(): string {
    try {
        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        // Assumes package.json is three levels up (from packages/server/src/services/mcp)
        const packageJsonPath = path.resolve(__dirname, "../../../package.json");
        if (!fs.existsSync(packageJsonPath)) {
            logger.error(`package.json not found at expected path: ${packageJsonPath}`);
            return "0.0.0-unknown";
        }
        const packageJsonContent = fs.readFileSync(packageJsonPath, "utf8");
        const packageJson = JSON.parse(packageJsonContent);
        return packageJson.version || "0.0.0-error"; // Fallback if version field is missing
    } catch (error) {
        logger.error("Failed to read or parse package.json version:", error);
        return "0.0.0-error"; // Fallback version on error
    }
}

// Default configuration for the MCP server
const DEFAULT_MCP_SERVER_CONFIG: ServerConfig = {
    mode: McpServerMode.SSE, // Default to SSE mode for server integration
    port: 3100, // Port is likely managed by the main app, but keep for potential standalone use
    messagePath: "/mcp/messages",
    heartbeatInterval: HEARTBEAT_INTERVAL_MS,
    serverInfo: {
        name: "vrooli-mcp-server",
        version: getPackageVersion(),
    },
};

let serverInstance: McpServerApp | null = null;


/**
 * Retrieves the singleton instance of the McpServerApp.
 * Throws an error if the server has not been initialized yet.
 *
 * @returns The McpServerApp instance.
 */
export function getMcpServer(): McpServerApp {
    if (!serverInstance) {
        throw new Error("MCP server accessed before initialization. Call initializeMcpServer first.");
    }
    return serverInstance;
}

/**
 * Registers graceful shutdown handlers for the MCP server.
 * Listens for SIGINT, SIGTERM, uncaught exceptions, and unhandled rejections.
 *
 * @param serverToShutdown - The McpServerApp instance to shut down.
 * @param timeoutMs - Maximum time allowed for shutdown before forcing exit.
 */
export function registerMcpShutdown(serverToShutdown: McpServerApp, timeoutMs = SHUTDOWN_TIMEOUT_MS): void {
    let isShuttingDown = false;

    async function performShutdown(signal: string): Promise<void> {
        if (isShuttingDown) {
            logger.info(`Shutdown already in progress. Ignoring ${signal}.`);
            return;
        }
        isShuttingDown = true;
        logger.info(`Received ${signal}. Initiating MCP server graceful shutdown...`);

        const forceShutdownTimer = setTimeout(() => {
            logger.error(`MCP server shutdown timed out after ${timeoutMs}ms. Forcing exit.`);
            process.exit(1); // Force exit if shutdown takes too long
        }, timeoutMs);

        try {
            await serverToShutdown.shutdown(); // Call the McpServerApp's shutdown method
            clearTimeout(forceShutdownTimer);
            logger.info("MCP server graceful shutdown completed.");
            // Note: We don't call process.exit(0) here anymore.
            // The main application should handle the final exit.
        } catch (error) {
            logger.error("Error during MCP server graceful shutdown:", error);
            clearTimeout(forceShutdownTimer);
            // Consider if forcing exit here is correct, or if the main app should decide
            process.exit(1);
        }
    }

    logger.info("Registering MCP server shutdown handlers...");
    process.on("SIGINT", () => performShutdown("SIGINT"));
    process.on("SIGTERM", () => performShutdown("SIGTERM"));
}

/**
 * Initializes the MCP server, configures its routes on the provided Express app,
 * and starts its internal services.
 *
 * @param app - The main Express application instance.
 * @param config - Optional server configuration, defaults to DEFAULT_MCP_SERVER_CONFIG.
 * @returns The initialized McpServerApp instance.
 */
export async function setupMCP(
    app: Express,
    config: ServerConfig = DEFAULT_MCP_SERVER_CONFIG,
): Promise<McpServerApp> {
    if (serverInstance) {
        logger.warn("MCP server already initialized.");
        return serverInstance;
    }

    logger.info("Initializing MCP server...");
    serverInstance = new McpServerApp(config, app);

    // Start the server logic (which now primarily sets up routes on the provided app)
    await serverInstance.start();

    // Note: Shutdown handlers are now managed by the central shutdown coordinator in index.ts
    // registerMcpShutdown(serverInstance);

    logger.info("MCP server initialized and routes configured.");
    return serverInstance;
}
