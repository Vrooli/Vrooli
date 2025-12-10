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
  /** Delay before emitting body (ms). Used to test streaming behavior. */
  bodyDelay?: number;
}): jest.Mocked<IncomingMessage> {
  const req = new EventEmitter() as jest.Mocked<IncomingMessage>;

  req.method = options?.method || 'GET';
  req.url = options?.url || '/';
  req.headers = options?.headers || {};
  // Add destroyed property to match IncomingMessage interface
  (req as any).destroyed = false;
  // Add destroy method to match IncomingMessage interface (required by hardened body-parser)
  (req as any).destroy = jest.fn(() => {
    (req as any).destroyed = true;
    req.emit('close');
  });

  const delay = options?.bodyDelay ?? 0;

  // Simulate body streaming
  if (options?.body) {
    setTimeout(() => {
      if ((req as any).destroyed) return;
      const bodyStr = typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
      req.emit('data', Buffer.from(bodyStr));
      req.emit('end');
    }, delay);
  } else {
    setTimeout(() => {
      if ((req as any).destroyed) return;
      req.emit('end');
    }, delay);
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

  // ServerResponse.end has complex overloads - use Object.defineProperty to bypass TypeScript
  const endMock = jest.fn((data?: unknown) => {
    if (data) {
      res.write(data);
    }
    res.emit('finish');
    return res;
  });
  Object.defineProperty(res, 'end', {
    value: endMock,
    writable: true,
    configurable: true,
  });

  // Helper to get response body
  (res as any).getBody = (): string => {
    const calls = (res.write as jest.Mock).mock.calls;
    return calls.map((call) => call[0]).join('');
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
