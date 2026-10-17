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
 * @param host - Interface to bind. Defaults to loopback: metrics describe this
 *   driver process to whatever supervises it locally, so there is no reason to
 *   accept connections from off-host.
 * @returns Promise resolving to the server instance
 */
export function createMetricsServer(port: number, host = '127.0.0.1'): Promise<Server> {
  return new Promise((resolve, reject) => {
    const server = createServer((req, res) => {
      if (req.url === '/metrics') {
        res.setHeader('Content-Type', metrics.getRegistry().contentType);
        void metrics.getMetrics()
          .then((metricsOutput) => {
            res.end(metricsOutput);
          })
          .catch((error) => {
            logger.error('Metrics server failed to collect metrics', {
              error: error instanceof Error ? error.message : String(error),
            });
            res.statusCode = 500;
            res.end('Failed to collect metrics');
          });
        return;
      }

      res.statusCode = 404;
      res.end('Not found');
    });

    server.on('error', (error: NodeJS.ErrnoException) => {
      if (error.code === 'EADDRINUSE') {
        reject(new Error(`Metrics port ${port} is already in use`));
      } else {
        reject(error);
      }
    });

    server.listen(port, host, () => {
      logger.info('Metrics server listening', {
        port,
        host,
        url: `http://${host}:${port}/metrics`,
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
