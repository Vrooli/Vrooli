/**
 * WebSocket Module
 *
 * Exports WebSocket connection management for frame streaming.
 *
 * @module frame-streaming/websocket
 */

export {
  WebSocketConnectionManager,
  createWebSocketConnectionManager,
  buildWebSocketUrl,
} from './connection';

export type {
  WebSocketConnectionState,
  WebSocketConnectionOptions,
} from './connection';
