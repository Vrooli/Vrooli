import type { IncomingMessage, ServerResponse } from 'http';
import { EventEmitter } from 'events';

/**
 * Mock IncomingMessage (HTTP Request)
 */
export function createMockRequest(options?: {
  method?: string;
  url?: string;
  headers?: Record<string, string>;
  body?: unknown;
}): jest.Mocked<IncomingMessage> {
  const req = new EventEmitter() as jest.Mocked<IncomingMessage>;

  req.method = options?.method || 'GET';
  req.url = options?.url || '/';
  req.headers = options?.headers || {};

  // Simulate body streaming
  if (options?.body) {
    process.nextTick(() => {
      const bodyStr = typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
      req.emit('data', Buffer.from(bodyStr));
      req.emit('end');
    });
  } else {
    process.nextTick(() => {
      req.emit('end');
    });
  }

  return req;
}

/**
 * Mock ServerResponse (HTTP Response)
 */
export function createMockResponse(): jest.Mocked<ServerResponse> {
  const res = new EventEmitter() as jest.Mocked<ServerResponse>;

  res.statusCode = 200;
  res.statusMessage = 'OK';
  res.setHeader = jest.fn();
  res.getHeader = jest.fn();
  res.removeHeader = jest.fn();
  res.writeHead = jest.fn();
  res.write = jest.fn();
  res.end = jest.fn((data?: unknown) => {
    if (data) {
      res.write(data);
    }
    res.emit('finish');
    return res;
  });

  // Helper to get response body
  (res as any).getBody = (): string => {
    const calls = (res.write as jest.Mock).mock.calls;
    return calls.map(call => call[0]).join('');
  };

  // Helper to get response JSON
  (res as any).getJSON = (): unknown => {
    const body = (res as any).getBody();
    return body ? JSON.parse(body) : null;
  };

  return res;
}

/**
 * Wait for response to finish
 */
export function waitForResponse(res: ServerResponse): Promise<void> {
  return new Promise((resolve) => {
    res.on('finish', () => resolve());
  });
}
