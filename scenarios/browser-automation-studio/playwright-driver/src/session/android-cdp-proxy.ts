import { createServer, type IncomingMessage } from 'node:http';
import type { Duplex } from 'node:stream';

type RawData = Buffer | ArrayBuffer | Buffer[];
interface ProxySocket {
  readyState: number;
  send(payload: string | RawData): void;
  close(): void;
  on(event: 'open' | 'close' | 'error', listener: () => void): this;
  on(event: 'message', listener: (payload: RawData) => void): this;
}
interface ProxyServer {
  handleUpgrade(request: IncomingMessage, socket: Duplex, head: Buffer, callback: (client: ProxySocket) => void): void;
  close(): void;
}
interface WebSocketModule {
  new (url: string): ProxySocket;
  OPEN: number;
  CONNECTING: number;
  Server: new (options: { noServer: boolean }) => ProxyServer;
}
// ws intentionally has a broad module type in this workspace; keep the
// narrow protocol surface local so the proxy cannot depend on ws internals.
// eslint-disable-next-line @typescript-eslint/no-var-requires
const WS = require('ws') as unknown as WebSocketModule;

type CDPVersion = { webSocketDebuggerUrl?: string };

/**
 * Android WebView CDP exposes a page session but not Chromium browser
 * context management. Playwright sends Browser.setDownloadBehavior while
 * attaching and aborts when that optional command is rejected. Keep the
 * compatibility exception at the transport boundary: every other command
 * and event still traverses the device-owned CDP endpoint unchanged.
 */
export async function createAndroidCDPProxy(endpoint: string): Promise<{ endpoint: string; close: () => Promise<void> }> {
  const versionResponse = await fetch(`${endpoint.replace(/\/$/, '')}/json/version`);
  if (!versionResponse.ok) {
    throw new Error(`Android CDP version discovery failed with HTTP ${versionResponse.status}`);
  }
  const version = (await versionResponse.json()) as CDPVersion;
  const upstreamURL = version.webSocketDebuggerUrl?.trim();
  if (!upstreamURL) {
    throw new Error('Android CDP version discovery omitted webSocketDebuggerUrl');
  }

  const server = createServer((request, response) => {
    if (request.url === '/json/version' || request.url === '/json/version/') {
      const body = JSON.stringify({ ...version, webSocketDebuggerUrl: `ws://127.0.0.1:${(server.address() as { port: number }).port}/devtools/browser` });
      response.writeHead(200, { 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(body) });
      response.end(body);
      return;
    }
    response.writeHead(404);
    response.end();
  });
  const websocketServer = new WS.Server({ noServer: true });
  const sockets = new Set<ProxySocket>();

  server.on('upgrade', (request, socket, head) => {
    websocketServer.handleUpgrade(request, socket, head, (client: ProxySocket) => {
      sockets.add(client);
      const upstream = new WS(upstreamURL);
      const pending: string[] = [];
      const forward = (payload: string) => {
        if (upstream.readyState === WS.OPEN) upstream.send(payload);
        else pending.push(payload);
      };
      upstream.on('open', () => {
        for (const payload of pending.splice(0)) upstream.send(payload);
      });
      client.on('message', (raw: RawData) => {
        const payload = raw.toString();
        try {
          const message = JSON.parse(payload) as { id?: number; method?: string };
          if (message.method === 'Browser.setDownloadBehavior' && message.id !== undefined) {
            client.send(JSON.stringify({ id: message.id, result: {} }));
            return;
          }
        } catch {
          // Forward malformed/non-JSON CDP payloads to preserve protocol behavior.
        }
        forward(payload);
      });
      upstream.on('message', (payload: RawData) => {
        if (client.readyState === WS.OPEN) client.send(payload);
      });
      const close = () => {
        sockets.delete(client);
        if (upstream.readyState === WS.OPEN || upstream.readyState === WS.CONNECTING) upstream.close();
        if (client.readyState === WS.OPEN) client.close();
      };
      client.on('close', close);
      client.on('error', close);
      upstream.on('close', () => { if (client.readyState === WS.OPEN) client.close(); });
      upstream.on('error', () => close());
    });
  });

  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => resolve());
  });
  const address = server.address();
  if (!address || typeof address === 'string') {
    await new Promise<void>((resolve) => server.close(() => resolve()));
    throw new Error('Android CDP compatibility proxy did not expose a TCP port');
  }

  let closed = false;
  const close = async () => {
    if (closed) return;
    closed = true;
    for (const socket of sockets) socket.close();
    websocketServer.close();
    await new Promise<void>((resolve) => server.close(() => resolve()));
  };
  return { endpoint: `http://127.0.0.1:${address.port}`, close };
}
