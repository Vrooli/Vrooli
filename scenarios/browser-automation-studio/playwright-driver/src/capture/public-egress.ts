/**
 * Public URL transport for EX9 capture.
 *
 * Every browser request passes through a private loopback proxy. The proxy
 * validates the complete DNS answer set and connects to the validated address
 * directly, so a later DNS answer cannot change the socket destination.
 */
import { lookup as dnsLookup } from 'node:dns/promises';
import { createServer, request as httpRequest, type IncomingMessage, type Server, type ServerResponse } from 'node:http';
import { isIP, connect as netConnect, type Socket } from 'node:net';
import type { Browser, BrowserContext } from 'rebrowser-playwright';
import { URL } from 'node:url';

export type ResolvedAddress = { address: string; family: number };
export type PublicEgressOptions = {
  lookup?: (hostname: string) => Promise<ResolvedAddress[]>;
  connect?: (address: string, port: number) => Socket;
};

export class PublicEgressError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'PublicEgressError';
  }
}

export function stripCredentialHeaders<T extends Record<string, string | string[] | undefined>>(headers: T): T {
  const blocked = new Set(['cookie', 'authorization', 'proxy-authorization']);
  return Object.fromEntries(Object.entries(headers).filter(([name]) => !blocked.has(name.toLowerCase()))) as T;
}

function rejectCredentialQuery(url: URL): void {
  const sensitiveQueryKey = /(?:token|secret|password|passwd|credential|authorization|auth|session|cookie|api.?key|access.?key|private.?key|signature|assertion|ticket|^sig$|^jwt$|^bearer$|^key$|^code$)/i;
  for (const key of url.searchParams.keys()) {
    const normalized = key.toLowerCase().replace(/[._-]/g, '');
    if (sensitiveQueryKey.test(normalized)) throw new PublicEgressError('Credential-like URL query parameters are not allowed');
  }
}

function ipv4Number(address: string): number {
  return address.split('.').reduce((value, octet) => ((value << 8) | Number(octet)) >>> 0, 0);
}

function inV4(address: string, base: string, bits: number): boolean {
  const mask = bits === 0 ? 0 : (0xffffffff << (32 - bits)) >>> 0;
  return (ipv4Number(address) & mask) === (ipv4Number(base) & mask);
}

const NON_PUBLIC_V4: Array<[string, number]> = [
  ['0.0.0.0', 8], ['10.0.0.0', 8], ['100.64.0.0', 10], ['127.0.0.0', 8],
  ['169.254.0.0', 16], ['172.16.0.0', 12], ['192.0.0.0', 24], ['192.0.2.0', 24],
  ['192.31.196.0', 24], ['192.52.193.0', 24], ['192.88.99.0', 24], ['192.168.0.0', 16],
  ['192.175.48.0', 24], ['198.18.0.0', 15], ['198.51.100.0', 24],
  ['203.0.113.0', 24], ['224.0.0.0', 4], ['240.0.0.0', 4],
];

function parseIpv6(address: string): bigint | undefined {
  let value = address.toLowerCase().split('%')[0] ?? '';
  if (value.includes('.')) {
    const lastColon = value.lastIndexOf(':');
    const v4 = value.slice(lastColon + 1);
    if (isIP(v4) !== 4) return undefined;
    const n = ipv4Number(v4);
    value = `${value.slice(0, lastColon)}:${((n >>> 16) & 0xffff).toString(16)}:${(n & 0xffff).toString(16)}`;
  }
  const halves = value.split('::');
  if (halves.length > 2) return undefined;
  const left = halves[0] ? halves[0].split(':') : [];
  const right = halves[1] ? halves[1].split(':') : [];
  const missing = 8 - left.length - right.length;
  if ((halves.length === 1 && missing !== 0) || (halves.length === 2 && missing < 1)) return undefined;
  const words = [...left, ...Array(missing).fill('0'), ...right];
  if (words.length !== 8 || words.some((word) => !/^[0-9a-f]{1,4}$/.test(word))) return undefined;
  return words.reduce((n, word) => (n << 16n) | BigInt(`0x${word}`), 0n);
}

function inV6(address: bigint, base: string, bits: number): boolean {
  const baseValue = parseIpv6(base);
  if (baseValue === undefined) return false;
  const mask = ((1n << BigInt(bits)) - 1n) << BigInt(128 - bits);
  return (address & mask) === (baseValue & mask);
}

/** Conservative global-unicast check, including IPv4-mapped and transition ranges. */
export function isPublicAddress(address: string): boolean {
  const family = isIP(address);
  if (family === 4) return !NON_PUBLIC_V4.some(([base, bits]) => inV4(address, base, bits));
  if (family !== 6) return false;
  const parsed = parseIpv6(address);
  if (parsed === undefined) return false;
  // IPv4-mapped IPv6 is always rejected, even when its embedded IPv4 is public.
  if (inV6(parsed, '::ffff:0:0', 96)) return false;
  // Only global-unicast space is admitted; exclude documentation, special-use,
  // 6to4 and the currently reserved 3fff documentation allocation.
  if (!inV6(parsed, '2000::', 3)) return false;
  const nonPublicV6: Array<[string, number]> = [
    ['2001::', 23], ['2001:db8::', 32], ['2002::', 16], ['3fff::', 20],
  ];
  return !nonPublicV6.some(([base, bits]) => inV6(parsed, base, bits));
}

export function validatePublicUrl(input: string): URL {
  let url: URL;
  try {
    url = new URL(input);
  } catch {
    throw new PublicEgressError('URL is invalid');
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new PublicEgressError('Only HTTP and HTTPS URLs are allowed');
  }
  if (url.username || url.password) throw new PublicEgressError('Embedded URL credentials are not allowed');
  if (!url.hostname || url.hostname.endsWith('.')) throw new PublicEgressError('Ambiguous URL hostname');
  rejectCredentialQuery(url);
  const literal = url.hostname.replace(/^\[|\]$/g, '');
  if (isIP(literal) && !isPublicAddress(literal)) throw new PublicEgressError('URL destination is not public');
  return url;
}

function validatePublicWebSocketUrl(input: string): URL {
  let url: URL;
  try { url = new URL(input); } catch { throw new PublicEgressError('WebSocket URL is invalid'); }
  if (url.protocol !== 'ws:' && url.protocol !== 'wss:') throw new PublicEgressError('Only ws and wss URLs are allowed for websocket requests');
  if (url.username || url.password) throw new PublicEgressError('Embedded websocket credentials are not allowed');
  if (!url.hostname || url.hostname.endsWith('.')) throw new PublicEgressError('Ambiguous websocket hostname');
  rejectCredentialQuery(url);
  const literal = url.hostname.replace(/^\[|\]$/g, '');
  if (isIP(literal) && !isPublicAddress(literal)) throw new PublicEgressError('WebSocket destination is not public');
  return url;
}

async function systemLookup(hostname: string): Promise<ResolvedAddress[]> {
  if (isIP(hostname)) return [{ address: hostname, family: isIP(hostname) }];
  return dnsLookup(hostname, { all: true, verbatim: true });
}

export async function resolvePublicAddress(hostname: string, lookup: NonNullable<PublicEgressOptions['lookup']> = systemLookup): Promise<string> {
  const answers = await lookup(hostname);
  if (!Array.isArray(answers) || answers.length === 0) throw new PublicEgressError('DNS returned no addresses');
  const unique = new Map<string, ResolvedAddress>();
  for (const answer of answers) {
    if (!answer || typeof answer.address !== 'string' || (answer.family !== 4 && answer.family !== 6)) {
      throw new PublicEgressError('DNS returned a malformed address');
    }
    const actualFamily = isIP(answer.address);
    if (actualFamily !== answer.family || !isPublicAddress(answer.address)) {
      throw new PublicEgressError('DNS destination is not public or has an ambiguous address family');
    }
    unique.set(`${actualFamily}:${answer.address.toLowerCase()}`, answer);
  }
  // Pick a stable address from the validated set; the socket uses this literal
  // rather than resolving the hostname again. Prefer IPv4 for broader reach.
  const candidates = [...unique.values()].sort((a, b) => a.family - b.family || a.address.localeCompare(b.address));
  const selected = candidates[0];
  if (!selected) throw new PublicEgressError('DNS returned no addresses');
  return selected.address;
}

function hostPort(authority: string, defaultPort: number): { hostname: string; port: number } {
  const parsed = new URL(`http://${authority}`);
  if (parsed.username || parsed.password || !parsed.hostname) throw new PublicEgressError('Invalid proxy destination');
  const port = parsed.port ? Number(parsed.port) : defaultPort;
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new PublicEgressError('Invalid destination port');
  return { hostname: parsed.hostname.replace(/^\[|\]$/g, ''), port };
}

function sendDenied(res: ServerResponse, error: unknown): void {
  const message = error instanceof PublicEgressError ? error.message : 'Destination resolution failed';
  res.writeHead(403, { 'content-type': 'text/plain', 'connection': 'close' });
  res.end(message);
}

export type PublicEgressProxy = {
  server: Server;
  url: string;
  close(): Promise<void>;
};

/** Start a local explicit proxy. All destination sockets use a validated IP literal. */
export async function createPublicEgressProxy(options: PublicEgressOptions = {}): Promise<PublicEgressProxy> {
  const lookup = options.lookup ?? systemLookup;
  const connect = options.connect ?? ((address, port) => netConnect({ host: address, port }));
  const sockets = new Set<Socket>();
  const track = (socket: Socket): Socket => {
    sockets.add(socket);
    socket.once('close', () => sockets.delete(socket));
    return socket;
  };
  const dial = (address: string, port: number): Socket => track(connect(address, port));
  const server = createServer((req, res) => {
    void (async () => {
      try {
        if (req.method !== 'GET' && req.method !== 'HEAD') throw new PublicEgressError('Only GET and HEAD requests are allowed');
        if ((req.headers['content-length'] && Number(req.headers['content-length']) > 0) || req.headers['transfer-encoding']) {
          throw new PublicEgressError('Request bodies are not allowed');
        }
        const url = new URL(req.url ?? '');
        const validated = validatePublicUrl(url.href);
        if (validated.protocol !== 'http:') throw new PublicEgressError('HTTPS requests must use the validated CONNECT tunnel');
        const { port } = hostPort(validated.host, 80);
        if (port !== 80) throw new PublicEgressError('Nonstandard HTTP ports are not allowed');
        const pinned = await resolvePublicAddress(validated.hostname, lookup);
        const requestHeaders = stripCredentialHeaders({ ...req.headers, host: validated.host });
        const upstream = httpRequest({
          host: pinned, port, method: req.method, path: `${validated.pathname}${validated.search}`,
          headers: requestHeaders,
          createConnection: () => dial(pinned, port),
        }, (response) => {
          const responseHeaders = { ...response.headers };
          delete responseHeaders['set-cookie'];
          res.writeHead(response.statusCode ?? 502, responseHeaders);
          response.pipe(res);
        });
        upstream.on('error', () => { if (!res.headersSent) sendDenied(res, new PublicEgressError('Public destination connection failed')); else res.destroy(); });
        req.pipe(upstream);
      } catch (error) { sendDenied(res, error); }
    })();
  });

  server.on('connect', (req, client, head) => {
    void (async () => {
      try {
        const { hostname, port } = hostPort(req.url ?? '', 443);
        if (port !== 443 || (isIP(hostname) && !isPublicAddress(hostname))) throw new PublicEgressError('Only public HTTPS destinations are allowed');
        const pinned = await resolvePublicAddress(hostname, lookup);
        const upstream = dial(pinned, port);
        upstream.once('error', () => client.destroy());
        upstream.once('connect', () => {
          client.write('HTTP/1.1 200 Connection Established\r\n\r\n');
          if (head.length) upstream.write(head);
          upstream.pipe(client);
          client.pipe(upstream);
        });
        client.once('error', () => upstream.destroy());
      } catch {
        client.end('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n');
      }
    })();
  });

  server.on('upgrade', (req: IncomingMessage, client: Socket, head: Buffer) => {
    void (async () => {
      let upstream: Socket | undefined;
      try {
        if (req.method !== 'GET') throw new PublicEgressError('Only GET websocket handshakes are allowed');
        const url = new URL(req.url ?? '');
        if (url.protocol !== 'ws:') throw new PublicEgressError('Only public ws or wss destinations are allowed');
        const { hostname, port } = hostPort(url.host, 80);
        if (port !== 80 || (isIP(hostname) && !isPublicAddress(hostname))) throw new PublicEgressError('Only public websocket destinations are allowed');
        const pinned = await resolvePublicAddress(hostname, lookup);
        const socket = dial(pinned, port);
        upstream = socket;
        socket.once('connect', () => {
          const headers = stripCredentialHeaders({ ...req.headers, host: url.host });
          const lines = Object.entries(headers).filter(([, value]) => value !== undefined).map(([key, value]) => `${key}: ${Array.isArray(value) ? value.join(', ') : value}`).join('\r\n');
          socket.write(`${req.method} ${url.pathname}${url.search} HTTP/${req.httpVersion}\r\n${lines}\r\n\r\n`);
          if (head.length) socket.write(head);
          socket.pipe(client);
          client.pipe(socket);
        });
        upstream.once('error', () => client.destroy());
        client.once('error', () => upstream?.destroy());
      } catch { client.end('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n'); upstream?.destroy(); }
    })();
  });
  server.on('connection', (socket) => { track(socket); });

  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => { server.removeListener('error', reject); resolve(); });
  });
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('Public egress proxy did not bind a TCP port');
  return {
    server,
    url: `http://127.0.0.1:${address.port}`,
    close: () => new Promise<void>((resolve, reject) => {
      server.close((error) => error ? reject(error) : resolve());
      for (const socket of sockets) socket.destroy();
    }),
  };
}

/** A capture context has no API for importing profile, cookies, or auth state. */
export async function createPublicCaptureContext(browser: Browser, rawUrl: string): Promise<{
  context: BrowserContext;
  initialUrl: string;
  close(): Promise<void>;
}> {
  const initialUrl = validatePublicUrl(rawUrl).href;
  const proxy = await createPublicEgressProxy();
  let context: BrowserContext | undefined;
  try {
    const createdContext = await browser.newContext({
      // Chromium bypasses proxies for loopback by default. Remove that implicit
      // exception so private subresources and redirects reach this policy.
      proxy: { server: proxy.url, bypass: '<-loopback>' },
      serviceWorkers: 'block',
    });
    context = createdContext;
    await createdContext.clearCookies();
    await createdContext.route('**/*', async (route) => {
      const request = route.request();
      try {
        validatePublicUrl(request.url());
      } catch {
        await route.abort('blockedbyclient');
        return;
      }
      if (request.method() !== 'GET' && request.method() !== 'HEAD') {
        await route.abort('blockedbyclient');
        return;
      }
      await route.continue({ headers: stripCredentialHeaders(await request.allHeaders()) });
    });
    await createdContext.routeWebSocket('**/*', async (route) => {
      try {
        const url = validatePublicWebSocketUrl(route.url());
        const cookieOrigin = `${url.protocol === 'wss:' ? 'https:' : 'http:'}//${url.host}`;
        if ((await createdContext.cookies(cookieOrigin)).length > 0) {
          await route.close({ code: 1008, reason: 'Unauthenticated capture blocks cookie-backed websockets' });
          return;
        }
        route.connectToServer();
      } catch {
        await route.close({ code: 1008, reason: 'WebSocket destination is not allowed for public capture' });
      }
    });
    return {
      context: createdContext,
      initialUrl,
      async close() {
        try { await createdContext.close(); } finally { await proxy.close(); }
      },
    };
  } catch (error) {
    await context?.close().catch(() => {});
    await proxy.close();
    throw error;
  }
}
