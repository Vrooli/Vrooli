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
  buildWebSocketUrl,
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

  it('builds correct WebSocket URLs from callback URLs', () => {
    const recordingUrl = buildWebSocketUrl(
      'http://localhost:8080/api/v1/recordings/live/session-1/frame',
      'session-1'
    );
    expect(recordingUrl).toBe('ws://localhost:8080/ws/recording/session-1/frames');

    const executionUrl = buildWebSocketUrl(
      'https://api.example.com/api/v1/executions/exec-123/frames',
      'session-1'
    );
    expect(executionUrl).toBe('wss://api.example.com/ws/execution/exec-123/frames');

    const fallbackUrl = buildWebSocketUrl('not-a-url', 'session-1');
    expect(fallbackUrl).toBe('ws://127.0.0.1:8080/ws/recording/session-1/frames');
  });
});
