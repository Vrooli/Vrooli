import type { IncomingMessage } from 'http';
import type { Config } from '../config';
import { logger } from '../utils';

/**
 * Parse JSON request body with size limit
 */
export async function parseJsonBody(
  req: IncomingMessage,
  config: Config
): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    let body = '';
    let size = 0;

    req.on('data', (chunk: Buffer) => {
      size += chunk.length;

      if (size > config.server.maxRequestSize) {
        reject(new Error(`Request body too large: ${size} bytes (max: ${config.server.maxRequestSize})`));
        return;
      }

      body += chunk.toString();
    });

    req.on('end', () => {
      if (!body) {
        resolve({});
        return;
      }

      try {
        const parsed = JSON.parse(body) as Record<string, unknown>;
        resolve(parsed);
      } catch (error) {
        logger.error('Failed to parse JSON body', {
          error: error instanceof Error ? error.message : String(error),
          bodyPreview: body.substring(0, 100),
        });
        reject(new Error('Invalid JSON in request body'));
      }
    });

    req.on('error', (error) => {
      reject(error);
    });
  });
}
