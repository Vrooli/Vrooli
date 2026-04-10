import { EventEmitter } from 'events';
import type { IncomingMessage } from 'http';

interface MockServerInstance {
  on(event: 'connection' | 'error', listener: (...args: unknown[]) => void): void;
  close(callback?: () => void): void;
  emitConnection(ws: MockClient, req: IncomingMessage): void;
  emitError(err: Error): void;
}

class MockServer implements MockServerInstance {
  handlers: Record<string, (...args: unknown[]) => void> = {};
  closed = false;

  constructor(public options: { port: number; path?: string }) {}

  on(event: 'connection' | 'error', listener: (...args: unknown[]) => void): void {
    this.handlers[event] = listener;
  }

  close(callback?: () => void): void {
    this.closed = true;
    callback?.();
  }

  emitConnection(ws: MockClient, req: IncomingMessage): void {
    const handler = this.handlers.connection;
    if (handler) {
      handler(ws, req);
    }
  }

  emitError(err: Error): void {
    const handler = this.handlers.error;
    if (handler) {
      handler(err);
    }
  }
}

const serverInstances: MockServer[] = [];

class MockClient extends EventEmitter {
  readyState = 1;
  send = jest.fn<void, [Buffer | string]>();
  close = jest.fn<void, [number?, string?]>();
}

jest.mock('ws', () => ({
  Server: class extends MockServer {
    constructor(options: { port: number; path?: string }) {
      super(options);
      serverInstances.push(this);
    }
  },
  OPEN: 1,
  __mockServerInstances: serverInstances,
}));

import { DirectFrameServer } from '../../../src/frame-streaming/websocket';

describe('DirectFrameServer', () => {
  beforeEach(() => {
    serverInstances.length = 0;
  });

  it('tracks connected clients and broadcasts frames', () => {
    const server = new DirectFrameServer(4567);
    server.start();

    const wsServer = serverInstances[0];
    expect(wsServer).toBeDefined();

    const client = new MockClient();
    const req = { url: '/frames?session_id=session-a' } as IncomingMessage;
    wsServer.emitConnection(client, req);

    expect(server.hasClients()).toBe(true);
    expect(client.send).toHaveBeenCalledWith(expect.stringContaining('connected'));

    server.broadcast(Buffer.from('frame-data'), 'session-a');
    expect(client.send).toHaveBeenCalledTimes(2);

    const sendCalls = client.send.mock.calls;
    const payload = sendCalls[1]?.[0];
    if (!Buffer.isBuffer(payload)) {
      throw new Error('Expected broadcast payload to be a Buffer');
    }
    expect(payload.length).toBeGreaterThan(8);
  });

  it('filters broadcasts by session ID and subscription', () => {
    const server = new DirectFrameServer(4568);
    server.start();

    const wsServer = serverInstances[0];

    const clientA = new MockClient();
    wsServer.emitConnection(clientA, { url: '/frames?session_id=session-a' } as IncomingMessage);

    const clientB = new MockClient();
    wsServer.emitConnection(clientB, { url: '/frames?session_id=session-b' } as IncomingMessage);

    server.broadcast(Buffer.from('frame-data'), 'session-a');

    const clientASends = clientA.send.mock.calls.length;
    const clientBSends = clientB.send.mock.calls.length;

    expect(clientASends).toBeGreaterThan(clientBSends);
    expect(server.hasSubscribers('session-a')).toBe(true);
    expect(server.hasSubscribers('session-missing')).toBe(false);
  });

  it('updates subscription on message', () => {
    const server = new DirectFrameServer(4569);
    server.start();

    const wsServer = serverInstances[0];
    const client = new MockClient();
    wsServer.emitConnection(client, { url: '/frames' } as IncomingMessage);

    client.emit('message', Buffer.from(JSON.stringify({ type: 'subscribe', session_id: 'session-x' })));

    expect(server.hasSubscribers('session-x')).toBe(true);
  });

  it('stops server and closes clients', () => {
    const server = new DirectFrameServer(4570);
    server.start();

    const wsServer = serverInstances[0];
    const client = new MockClient();
    wsServer.emitConnection(client, { url: '/frames?session_id=session-a' } as IncomingMessage);

    server.stop();

    expect(client.close).toHaveBeenCalledWith(1000, 'Server shutting down');
    expect(wsServer.closed).toBe(true);
    expect(server.isActive()).toBe(false);
  });
});
