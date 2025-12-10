import type { IncomingMessage } from 'http';
import type { Config } from '../config';
import { logger } from '../utils';

/**
 * Parse JSON request body with size limit
 *
 * Hardened assumptions:
 * - Request stream may emit errors before, during, or after data chunks
 * - Request may be aborted by client mid-stream
 * - Body may be empty, which is valid (returns empty object)
 * - JSON parsing may fail on malformed input
 * - Size limit must be enforced and stream destroyed on violation
 */
export async function parseJsonBody(
  req: IncomingMessage,
  config: Partial<Config> | Config
): Promise<Record<string, unknown>> {
  return new Promise((resolve, reject) => {
    let body = '';
    let size = 0;
    let rejected = false;

    // Allow callers that do not have full config (e.g., record-mode routes currently pass {})
    const maxRequestSize = config?.server?.maxRequestSize ?? 5 * 1024 * 1024; // 5MB default

    const cleanup = () => {
      req.removeAllListeners('data');
      req.removeAllListeners('end');
      req.removeAllListeners('error');
      req.removeAllListeners('close');
      req.removeAllListeners('aborted');
    };

    const rejectOnce = (error: Error) => {
      if (rejected) return;
      rejected = true;
      cleanup();
      // Destroy the stream to stop receiving more data
      if (!req.destroyed) {
        req.destroy();
      }
      reject(error);
    };

    req.on('data', (chunk: Buffer) => {
      if (rejected) return;

      size += chunk.length;

      if (size > maxRequestSize) {
        logger.warn('body-parser: request body exceeds size limit', {
          receivedBytes: size,
          maxBytes: maxRequestSize,
          hint: 'Request was rejected. Increase MAX_REQUEST_SIZE if legitimate.',
        });
        rejectOnce(new Error(`Request body too large: ${size} bytes (max: ${maxRequestSize})`));
        return;
      }

      body += chunk.toString();
    });

    req.on('end', () => {
      if (rejected) return;
      cleanup();

      if (!body) {
        resolve({});
        return;
      }

      try {
        const parsed: unknown = JSON.parse(body);

        // Hardened: JSON.parse can return any valid JSON value (object, array, string, number, boolean, null)
        // We expect object bodies for our API. Handle non-object cases gracefully.
        if (parsed === null) {
          // null is valid JSON but not a valid request body for our API
          resolve({});
          return;
        }

        if (typeof parsed !== 'object') {
          // Primitives (string, number, boolean) are not valid request bodies
          logger.warn('body-parser: JSON body is not an object', {
            type: typeof parsed,
            hint: 'Request body should be a JSON object, not a primitive value',
          });
          reject(new Error(`Invalid JSON body: expected object, got ${typeof parsed}`));
          return;
        }

        if (Array.isArray(parsed)) {
          // Arrays are valid JSON but not expected for our API endpoints
          // Some endpoints might accept arrays in the future, but currently none do
          logger.warn('body-parser: JSON body is an array, expected object', {
            arrayLength: parsed.length,
            hint: 'Request body should be a JSON object, not an array',
          });
          reject(new Error('Invalid JSON body: expected object, got array'));
          return;
        }

        resolve(parsed as Record<string, unknown>);
      } catch (error) {
        logger.error('body-parser: failed to parse JSON', {
          error: error instanceof Error ? error.message : String(error),
          bodyPreview: body.substring(0, 100),
          bodyLength: body.length,
          hint: 'Request body was not valid JSON',
        });
        reject(new Error('Invalid JSON in request body'));
      }
    });

    req.on('error', (error) => {
      logger.warn('body-parser: request stream error', {
        error: error.message,
        bytesReceived: size,
      });
      rejectOnce(error);
    });

    // Handle client abort
    req.on('aborted', () => {
      logger.debug('body-parser: request aborted by client', {
        bytesReceived: size,
      });
      rejectOnce(new Error('Request aborted by client'));
    });

    // Handle premature close (client disconnect)
    req.on('close', () => {
      if (rejected) return;
      // If we haven't finished processing, this is an unexpected close
      if (!body && size === 0) {
        // Empty request that closed normally - this is fine
        return;
      }
    });
  });
}
