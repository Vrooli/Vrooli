#!/usr/bin/env node

import 'reflect-metadata';
import { createApp } from './app.js';
import { config } from './config/index.js';
import { closeDatabaseConnections } from './config/database.js';
import { logger } from './utils/logger.js';

/**
 * Start the API server
 */
async function startServer(): Promise<void> {
  try {
    // Create and configure app
    const app = await createApp();

    // Start listening
    const server = app.listen(config.port, config.host, () => {
      logger.info(`🚀 Agent Metareasoning Manager API running at http://${config.host}:${config.port}`);
      logger.info(`📚 Environment: ${config.env}`);
      logger.info(`🔧 n8n: ${config.n8n.baseUrl}`);
      logger.info(`🔧 Windmill: ${config.windmill.baseUrl}`);
    });

    // Graceful shutdown handler
    const gracefulShutdown = async (signal: string): Promise<void> => {
      logger.info(`${signal} received, starting graceful shutdown...`);

      server.close(async () => {
        logger.info('HTTP server closed');

        try {
          await closeDatabaseConnections();
          logger.info('Database connections closed');
          process.exit(0);
        } catch (error) {
          logger.error('Error during shutdown:', error);
          process.exit(1);
        }
      });

      // Force shutdown after 10 seconds
      setTimeout(() => {
        logger.error('Could not close connections in time, forcefully shutting down');
        process.exit(1);
      }, 10000);
    };

    // Handle shutdown signals
    process.on('SIGTERM', () => gracefulShutdown('SIGTERM'));
    process.on('SIGINT', () => gracefulShutdown('SIGINT'));

    // Handle uncaught errors
    process.on('uncaughtException', (error: Error) => {
      logger.error('Uncaught Exception:', error);
      process.exit(1);
    });

    process.on('unhandledRejection', (reason: any, promise: Promise<any>) => {
      logger.error('Unhandled Rejection at:', promise, 'reason:', reason);
      process.exit(1);
    });

  } catch (error) {
    logger.error('Failed to start server:', error);
    process.exit(1);
  }
}

// Start the server
startServer();