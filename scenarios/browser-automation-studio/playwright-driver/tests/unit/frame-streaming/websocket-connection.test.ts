import { EventEmitter } from 'events';

const mockInstances: MockWebSocket[] = [];

class MockWebSocket extends EventEmitter {
  readonly url: string;
  readyState = 0;
  send = jest.fn();
  close = jest.fn();

  constructor(url: string) {
    super();
    this.url = url;
    mockInstances.push(this);
  }
}

jest.mock('ws', () => ({
  __esModule: true,
  default: MockWebSocket,
  __mockInstances: mockInstances,
}));

import {
  WebSocketConnectionManager,
} from '../../../src/frame-streaming/websocket';

describe('WebSocket connection manager', () => {
  beforeEach(() => {
    mockInstances.length = 0;
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('marks connection ready on open', () => {
    const manager = new WebSocketConnectionManager({
      url: 'ws://localhost:1234/ws',
      sessionId: 'session-1',
      reconnectDelayMs: 10,
    });

    manager.connect();
    const ws = manager.getWebSocket() as MockWebSocket;

    ws.readyState = 1;
    ws.emit('open');

    expect(manager.isReady()).toBe(true);
  });

  it('reconnects after close while active', () => {
    const manager = new WebSocketConnectionManager({
      url: 'ws://localhost:1234/ws',
      sessionId: 'session-1',
      reconnectDelayMs: 10,
    });

    manager.connect();
    const ws = manager.getWebSocket() as MockWebSocket;

    ws.emit('close');
    expect(manager.isReady()).toBe(false);

    jest.advanceTimersByTime(10);
    expect(mockInstances.length).toBe(2);
  });

  it('ignores a stale socket close after a replacement is ready', () => {
    const manager = new WebSocketConnectionManager({
      url: 'ws://localhost:1234/ws',
      sessionId: 'session-1',
      reconnectDelayMs: 10,
    });

    manager.connect();
    const first = mockInstances[0];
    if (!first) throw new Error('first socket was not created');
    manager.connect();
    const replacement = mockInstances[1];
    if (!replacement) throw new Error('replacement socket was not created');
    replacement.readyState = 1;
    replacement.emit('open');

    first.emit('close');
    jest.advanceTimersByTime(10);

    expect(manager.getWebSocket()).toBe(replacement);
    expect(manager.isReady()).toBe(true);
    expect(mockInstances).toHaveLength(2);
  });

  it('stops reconnection when closed', () => {
    const manager = new WebSocketConnectionManager({
      url: 'ws://localhost:1234/ws',
      sessionId: 'session-1',
      reconnectDelayMs: 10,
    });

    manager.connect();
    const ws = manager.getWebSocket() as MockWebSocket;

    manager.close();
    expect(manager.isActive()).toBe(false);
    expect(ws.close).toHaveBeenCalled();

    ws.emit('close');
    jest.advanceTimersByTime(10);
    expect(mockInstances.length).toBe(1);
  });

});
