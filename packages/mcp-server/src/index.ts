import { parseConfig } from './config/index.js';
import { createLogger } from './logger.js';
import { McpServerApp } from './server.js';
import { setupShutdownHandlers } from './shutdown.js';

/**
 * Main entry point for the MCP server
 */
async function main(): Promise<void> {
    const logger = createLogger();

    try {
        logger.info('Starting MCP server...');

        // Parse config from command line args
        const config = parseConfig(process.argv.slice(2), logger);

        // Initialize and start the server
        const server = new McpServerApp(config, logger);
        await server.start();

        // Setup graceful shutdown
        setupShutdownHandlers(
            () => server.shutdown(),
            logger
        );

        logger.info(`MCP server is running in ${config.mode} mode.`);
    } catch (error) {
        logger.error('Failed to start MCP server:', error);
        process.exit(1);
    }
}

// Start the server
main().catch(error => {
    console.error('Unhandled error during server startup:', error);
    process.exit(1);
});