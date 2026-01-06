/**
 * Direct Frame Server
 *
 * WebSocket server for direct UI-to-driver frame streaming.
 * This bypasses the Go API hub for lower latency.
 *
 * Part of the latency research spike to measure improvement from
 * eliminating the API hop.
 *
 * Frame format: [8-byte timestamp BigInt64BE][JPEG data]
 *
 * @module frame-streaming/websocket/server
 */

/* eslint-disable @typescript-eslint/no-require-imports */
import type { IncomingMessage } from 'http';
import { logger, scopedLog, LogContext } from '../../utils';

// ws module uses CommonJS exports that don't play well with ESM imports
// Use require for proper runtime access to Server and WebSocket classes
const WebSocket = require('ws');
const WebSocketServer = WebSocket.Server;

// WebSocket ready state constant
const WS_OPEN = 1;

// Type definitions for ws module
interface WebSocketClient {
  readyState: number;
  send(data: Buffer | string): void;
  close(code?: number, reason?: string): void;
  on(event: 'message', listener: (data: Buffer | string) => void): void;
  on(event: 'close', listener: () => void): void;
  on(event: 'error', listener: (error: Error) => void): void;
}

interface WebSocketServerInstance {
  on(event: 'connection', listener: (ws: WebSocketClient, req: IncomingMessage) => void): void;
  on(event: 'error', listener: (error: Error) => void): void;
  close(callback?: () => void): void;
}

interface DirectFrameClient {
  ws: WebSocketClient;
  sessionId: string | null;
  connectedAt: Date;
}

/**
 * Direct frame streaming server for latency research spike.
 *
 * Allows UI clients to connect directly to playwright-driver
 * for lower-latency frame streaming (bypasses API hub).
 */
export class DirectFrameServer {
  private wss: WebSocketServerInstance | null = null;
  private clients: Set<DirectFrameClient> = new Set();
  private port: number;
  private isRunning = false;

  constructor(port: number) {
    this.port = port;
  }

  /**
   * Start the WebSocket server.
   */
  start(): void {
    if (this.isRunning) {
      logger.warn(scopedLog(LogContext.RECORDING, 'direct frame server already running'), {
        port: this.port,
      });
      return;
    }

    this.wss = new WebSocketServer({
      port: this.port,
      path: '/frames',
    }) as WebSocketServerInstance;

    this.wss.on('connection', (ws: WebSocketClient, req: IncomingMessage) => {
      this.handleConnection(ws, req);
    });

    this.wss.on('error', (error: Error) => {
      logger.error(scopedLog(LogContext.RECORDING, 'direct frame server error'), {
        error: error.message,
        port: this.port,
      });
    });

    this.isRunning = true;
    logger.info(scopedLog(LogContext.RECORDING, 'direct frame server started'), {
      port: this.port,
      path: '/frames',
      url: `ws://localhost:${this.port}/frames`,
    });
  }

  /**
   * Handle new WebSocket connection.
   */
  private handleConnection(ws: WebSocketClient, req: IncomingMessage): void {
    // Parse session ID from URL query params: /frames?session_id=xxx
    let sessionId: string | null = null;
    try {
      const url = new URL(req.url || '', `http://localhost:${this.port}`);
      sessionId = url.searchParams.get('session_id');
    } catch {
      // Ignore parse errors
    }

    const client: DirectFrameClient = {
      ws,
      sessionId,
      connectedAt: new Date(),
    };

    this.clients.add(client);

    logger.info(scopedLog(LogContext.RECORDING, 'direct frame client connected'), {
      sessionId,
      totalClients: this.clients.size,
    });

    // Send connection confirmation
    ws.send(JSON.stringify({
      type: 'connected',
      timestamp: Date.now(),
      sessionId,
    }));

    ws.on('message', (data: Buffer | string) => {
      // Handle subscription messages
      try {
        const msg = JSON.parse(data.toString()) as { type?: string; session_id?: string };
        if (msg.type === 'subscribe' && msg.session_id) {
          client.sessionId = msg.session_id;
          logger.debug(scopedLog(LogContext.RECORDING, 'direct frame client subscribed'), {
            sessionId: msg.session_id,
          });
        }
      } catch {
        // Ignore non-JSON messages
      }
    });

    ws.on('close', () => {
      this.clients.delete(client);
      logger.debug(scopedLog(LogContext.RECORDING, 'direct frame client disconnected'), {
        sessionId: client.sessionId,
        totalClients: this.clients.size,
      });
    });

    ws.on('error', (error: Error) => {
      logger.warn(scopedLog(LogContext.RECORDING, 'direct frame client error'), {
        sessionId: client.sessionId,
        error: error.message,
      });
      this.clients.delete(client);
    });
  }

  /**
   * Broadcast binary frame to all connected clients (or filtered by sessionId).
   *
   * Prepends 8-byte timestamp for latency measurement.
   *
   * @param frameData - Raw JPEG frame data
   * @param sessionId - Optional session ID to filter clients
   */
  broadcast(frameData: Buffer, sessionId?: string): void {
    if (!this.isRunning || this.clients.size === 0) {
      return;
    }

    // Prepend 8-byte timestamp for latency measurement
    const timestampBuffer = Buffer.alloc(8);
    timestampBuffer.writeBigInt64BE(BigInt(Date.now()), 0);
    const payload = Buffer.concat([timestampBuffer, frameData]);

    let sentCount = 0;
    for (const client of this.clients) {
      // Filter by session ID if provided
      if (sessionId && client.sessionId && client.sessionId !== sessionId) {
        continue;
      }

      if (client.ws.readyState === WS_OPEN) {
        try {
          client.ws.send(payload);
          sentCount++;
        } catch {
          // Client disconnected, will be cleaned up on error/close
        }
      }
    }

    // Debug log every 100th frame
    if (sentCount > 0 && Math.random() < 0.01) {
      logger.debug(scopedLog(LogContext.RECORDING, 'direct frame broadcast'), {
        clients: sentCount,
        frameBytes: frameData.length,
        sessionId,
      });
    }
  }

  /**
   * Check if any clients are connected.
   */
  hasClients(): boolean {
    return this.clients.size > 0;
  }

  /**
   * Check if any clients are subscribed to a specific session.
   */
  hasSubscribers(sessionId: string): boolean {
    for (const client of this.clients) {
      if (client.sessionId === sessionId && client.ws.readyState === WebSocket.OPEN) {
        return true;
      }
    }
    return false;
  }

  /**
   * Get the server port.
   */
  getPort(): number {
    return this.port;
  }

  /**
   * Check if server is running.
   */
  isActive(): boolean {
    return this.isRunning;
  }

  /**
   * Stop the WebSocket server.
   */
  stop(): void {
    if (!this.isRunning) {
      return;
    }

    this.isRunning = false;

    // Close all client connections
    for (const client of this.clients) {
      try {
        client.ws.close(1000, 'Server shutting down');
      } catch {
        // Ignore close errors
      }
    }
    this.clients.clear();

    // Close server
    if (this.wss) {
      this.wss.close(() => {
        logger.info(scopedLog(LogContext.RECORDING, 'direct frame server stopped'), {
          port: this.port,
        });
      });
      this.wss = null;
    }
  }
}

/**
 * Create a direct frame server instance.
 */
export function createDirectFrameServer(port: number): DirectFrameServer {
  return new DirectFrameServer(port);
}
