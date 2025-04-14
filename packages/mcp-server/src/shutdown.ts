import { Logger } from './types.js';

/**
 * Setup shutdown handlers for graceful process termination
 * @param shutdownFn Function to call during shutdown
 * @param logger Logger instance for shutdown messaging
 * @param timeoutMs Force exit after this many milliseconds
 */
export function setupShutdownHandlers(
    shutdownFn: () => Promise<void>,
    logger: Logger,
    timeoutMs = 5000
): void {
    let isShuttingDown = false;

    const shutdown = async (signal: string): Promise<void> => {
        // Prevent multiple shutdown attempts
        if (isShuttingDown) {
            return;
        }

        isShuttingDown = true;
        logger.info(`Received ${signal} signal. Initiating graceful shutdown...`);

        // Setup force shutdown timeout
        const forceShutdownTimer = setTimeout(() => {
            logger.error(`Shutdown timed out after ${timeoutMs}ms. Forcing exit.`);
            process.exit(1);
        }, timeoutMs);

        try {
            await shutdownFn();
            clearTimeout(forceShutdownTimer);
            logger.info('Graceful shutdown completed.');
            process.exit(0);
        } catch (error) {
            logger.error('Error during shutdown:', error);
            clearTimeout(forceShutdownTimer);
            process.exit(1);
        }
    };

    // Register signal handlers
    process.on('SIGINT', () => shutdown('SIGINT'));
    process.on('SIGTERM', () => shutdown('SIGTERM'));

    // Handle uncaught exceptions and unhandled rejections
    process.on('uncaughtException', (error) => {
        logger.error('Uncaught exception:', error);
        shutdown('uncaughtException');
    });

    process.on('unhandledRejection', (reason) => {
        logger.error('Unhandled rejection:', reason);
        shutdown('unhandledRejection');
    });
} 