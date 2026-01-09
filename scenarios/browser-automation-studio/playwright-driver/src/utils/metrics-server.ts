/**
 * Metrics Server
 *
 * Dedicated HTTP server for Prometheus metrics endpoint.
 * Separated from main server to keep infrastructure concerns isolated.
 */

import { createServer, type Server } from 'http';
import { logger } from './logger';
import { metrics } from './metrics';

/**
 * Create and start a metrics HTTP server.
 *
 * @param port - Port to listen on
 * @returns Promise resolving to the server instance
 */
export function createMetricsServer(port: number): Promise<Server> {
  return new Promise((resolve, reject) => {
    const server = createServer(async (req, res) => {
      if (req.url === '/metrics') {
        res.setHeader('Content-Type', metrics.getRegistry().contentType);
        const metricsOutput = await metrics.getMetrics();
        res.end(metricsOutput);
      } else {
        res.statusCode = 404;
        res.end('Not found');
      }
    });

    server.on('error', (error: NodeJS.ErrnoException) => {
      if (error.code === 'EADDRINUSE') {
        reject(new Error(`Metrics port ${port} is already in use`));
      } else {
        reject(error);
      }
    });

    server.listen(port, '0.0.0.0', () => {
      logger.info('Metrics server listening', {
        port,
        url: `http://0.0.0.0:${port}/metrics`,
      });
      resolve(server);
    });
  });
}

/**
 * Gracefully close metrics server.
 *
 * @param server - Server instance to close
 * @returns Promise resolving when server is closed
 */
export function closeMetricsServer(server: Server): Promise<void> {
  return new Promise((resolve) => {
    server.close(() => {
      logger.info('Metrics server closed');
      resolve();
    });
  });
}
