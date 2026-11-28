import { request as httpRequest } from 'node:http';
import {
  createPublicCaptureContext,
  createPublicEgressProxy,
  isPublicAddress,
  resolvePublicAddress,
  stripCredentialHeaders,
  PublicEgressError,
  validatePublicUrl,
} from '../../../src/capture/public-egress';

describe('EX9 public URL egress', () => {
  test.each([
    'http://127.0.0.1/', 'http://10.1.2.3/', 'http://169.254.169.254/latest/meta-data/',
    'http://[::1]/', 'http://[fe80::1]/', 'http://[::ffff:8.8.8.8]/',
  ])('rejects internal or special address literal %s', (url) => {
    expect(() => validatePublicUrl(url)).toThrow(PublicEgressError);
  });

  test.each([
    'http://user:password@example.com/', 'https://example.com/?token=abc',
    'https://example.com/?ACCESS_TOKEN=abc', 'https://example.com/?api_key=abc',
    'https://example.com/?client-secret=abc', 'https://example.com/?session_id=abc',
    'https://example.com/?code=oauth-secret',
    'file:///etc/passwd', 'ftp://example.com/',
  ])('rejects credentials and unsupported URL %s', (url) => {
    expect(() => validatePublicUrl(url)).toThrow(PublicEgressError);
  });

  test('accepts a benign public URL', () => {
    expect(validatePublicUrl('https://example.com/a').href).toBe('https://example.com/a');
    expect(isPublicAddress('8.8.8.8')).toBe(true);
  });

  test('accepts multiple public A and AAAA answers and selects one deterministically', async () => {
    const answers = [
      { address: '2606:4700:4700::1111', family: 6 },
      { address: '8.8.8.8', family: 4 },
      { address: '1.1.1.1', family: 4 },
    ];
    await expect(resolvePublicAddress('public.test', async () => answers)).resolves.toBe('1.1.1.1');
    await expect(resolvePublicAddress('public.test', async () => [...answers].reverse())).resolves.toBe('1.1.1.1');
    await expect(resolvePublicAddress('public.test', async () => [
      { address: '8.8.8.8', family: 6 },
    ])).rejects.toThrow(/family/i);
  });

  test.each([
    [['10.0.0.7', '8.8.8.8']],
    [['169.254.169.254']],
    [[]],
  ])('fails closed for private, metadata, or ambiguous DNS answer set %#', async (answers) => {
    const proxy = await createPublicEgressProxy({
      lookup: async () => answers.map((answer) => typeof answer === 'string'
        ? { address: answer, family: answer.includes(':') ? 6 : 4 }
        : answer),
      connect: () => { throw new Error('denied target must not open a socket'); },
    });
    try {
      const response = await proxyRequest(proxy.url, 'http://public.test/');
      expect(response.status).toBe(403);
      expect(response.body).toMatch(/not public|ambiguous|no addresses/i);
    } finally { await proxy.close(); }
  });

  test('blocks public-to-private redirects and internal or metadata subresources before opening a socket', async () => {
    let socketOpens = 0;
    const proxy = await createPublicEgressProxy({
      lookup: async (host) => [{ address: host === 'public.test' ? '8.8.8.8' : host === 'metadata.test' ? '169.254.169.254' : '10.2.3.4', family: 4 }],
      connect: () => { socketOpens += 1; throw new Error('socket should be denied before connect'); },
    });
    try {
      // A browser follows a redirect or fetches a subresource by making a new
      // proxy request; each destination is independently resolved and checked.
      for (const target of ['http://127.0.0.1/admin', 'http://private.test/next', 'http://metadata.test/latest/meta-data/']) {
        expect((await proxyRequest(proxy.url, target)).status).toBe(403);
      }
      expect(socketOpens).toBe(0);
    } finally { await proxy.close(); }
  });

  test('pins the connector to the selected public address and revalidates each new destination', async () => {
    const calls: string[] = [];
    const opened: Array<{ address: string; port: number }> = [];
    const proxy = await createPublicEgressProxy({
      lookup: async (host) => { calls.push(host); return [{ address: '8.8.8.8', family: 4 }, { address: '1.1.1.1', family: 4 }]; },
      connect: (address, port) => {
        opened.push({ address, port });
        throw new Error('stop before a real public socket is opened');
      },
    });
    try {
      expect((await connectRequest(proxy.url, 'public.test:443')).status).toBe(403);
      expect(calls).toEqual(['public.test']);
      expect(opened).toEqual([{ address: '1.1.1.1', port: 443 }]);
    } finally { await proxy.close(); }
    let answer = '8.8.8.8';
    const changingLookup = async () => [{ address: answer, family: 4 }];
    await expect(resolvePublicAddress('public.test', changingLookup)).resolves.toBe('8.8.8.8');
    answer = '10.0.0.7';
    await expect(resolvePublicAddress('public.test', changingLookup)).rejects.toThrow(/not public/i);
    await expect(resolvePublicAddress('public.test', async () => [
      { address: '8.8.8.8', family: 4 }, { address: '1.1.1.1', family: 4 },
    ])).resolves.toBe('1.1.1.1');
  });

  test('creates a clean browser context without profile, cookies, storage, or auth headers', async () => {
    const optionsSeen: Record<string, unknown>[] = [];
    let guardedRequest: ((route: never) => Promise<void>) | undefined;
    let guardedWebSocket: ((route: never) => Promise<void>) | undefined;
    const route = jest.fn(async (_pattern: string, handler: (route: never) => Promise<void>) => { guardedRequest = handler; });
    const routeWebSocket = jest.fn(async (_pattern: string, handler: (route: never) => Promise<void>) => { guardedWebSocket = handler; });
    const cookies = jest.fn(async () => [] as Array<{ name: string; value: string }>);
    const context = { clearCookies: jest.fn(async () => {}), close: jest.fn(async () => {}), route, routeWebSocket, cookies };
    const browser = { newContext: jest.fn(async (options: Record<string, unknown>) => { optionsSeen.push(options); return context; }) };
    const capture = await createPublicCaptureContext(browser as never, 'https://example.com');
    try {
      expect(optionsSeen).toHaveLength(1);
      expect(context.clearCookies).toHaveBeenCalledTimes(1);
      expect(optionsSeen[0]).toEqual(expect.objectContaining({
        serviceWorkers: 'block',
        proxy: expect.objectContaining({ server: expect.stringMatching(/^http:\/\/127\.0\.0\.1:/), bypass: '<-loopback>' }),
      }));
      for (const forbidden of ['storageState', 'httpCredentials', 'extraHTTPHeaders', 'cookies', 'browserProfile']) {
        expect(optionsSeen[0]).not.toHaveProperty(forbidden);
      }
      expect(route).toHaveBeenCalledWith('**/*', expect.any(Function));
      if (!guardedRequest) throw new Error('capture request guard was not installed');
      const deniedRoute = {
        request: () => ({ url: () => 'https://example.com/post', method: () => 'POST', allHeaders: async () => ({}) }),
        abort: jest.fn(async () => {}), continue: jest.fn(async () => {}),
      };
      await guardedRequest(deniedRoute as never);
      expect(deniedRoute.abort).toHaveBeenCalledWith('blockedbyclient');
      const credentialRoute = {
        request: () => ({ url: () => 'https://example.com/?token=secret', method: () => 'GET', allHeaders: async () => ({}) }),
        abort: jest.fn(async () => {}), continue: jest.fn(async () => {}),
      };
      await guardedRequest(credentialRoute as never);
      expect(credentialRoute.abort).toHaveBeenCalledWith('blockedbyclient');
      let continuedHeaders: Record<string, string> | undefined;
      const cleanRoute = {
        request: () => ({
          url: () => 'https://example.com/', method: () => 'GET',
          allHeaders: async () => ({ cookie: 'session=private', authorization: 'Bearer private', 'x-safe': 'yes' }),
        }),
        abort: jest.fn(async () => {}),
        continue: jest.fn(async (options: { headers: Record<string, string> }) => { continuedHeaders = options.headers; }),
      };
      await guardedRequest(cleanRoute as never);
      expect(continuedHeaders).toEqual({ 'x-safe': 'yes' });
      const headRoute = {
        request: () => ({ url: () => 'https://example.com/', method: () => 'HEAD', allHeaders: async () => ({}) }),
        abort: jest.fn(async () => {}), continue: jest.fn(async () => {}),
      };
      await guardedRequest(headRoute as never);
      expect(headRoute.continue).toHaveBeenCalledTimes(1);
      expect(routeWebSocket).toHaveBeenCalledWith('**/*', expect.any(Function));
      if (!guardedWebSocket) throw new Error('capture websocket guard was not installed');
      const forbiddenSocket = { url: () => 'wss://example.com/socket?access_token=secret', close: jest.fn(async () => {}), connectToServer: jest.fn() };
      await guardedWebSocket(forbiddenSocket as never);
      expect(forbiddenSocket.close).toHaveBeenCalledWith(expect.objectContaining({ code: 1008 }));
      expect(forbiddenSocket.connectToServer).not.toHaveBeenCalled();
      const allowedSocket = { url: () => 'wss://example.com/socket', close: jest.fn(async () => {}), connectToServer: jest.fn() };
      await guardedWebSocket(allowedSocket as never);
      expect(cookies).toHaveBeenCalledWith('https://example.com');
      expect(allowedSocket.connectToServer).toHaveBeenCalledTimes(1);
      cookies.mockResolvedValueOnce([{ name: 'session', value: 'site-issued' }]);
      const cookieBackedSocket = { url: () => 'wss://example.com/socket', close: jest.fn(async () => {}), connectToServer: jest.fn() };
      await guardedWebSocket(cookieBackedSocket as never);
      expect(cookieBackedSocket.close).toHaveBeenCalledWith(expect.objectContaining({ code: 1008 }));
      expect(cookieBackedSocket.connectToServer).not.toHaveBeenCalled();
      expect(capture.initialUrl).toBe('https://example.com/');
    } finally { await capture.close(); }
  });

  test('allows only GET or HEAD at the proxy, and strips cookie and auth headers', async () => {
    let socketOpens = 0;
    const proxy = await createPublicEgressProxy({
      lookup: async () => [{ address: '8.8.8.8', family: 4 }],
      connect: () => { socketOpens += 1; throw new Error('test socket not opened'); },
    });
    try {
      const response = await proxyRequest(proxy.url, 'http://example.test/', {
        method: 'POST', headers: { authorization: 'Bearer secret', cookie: 'session=private' },
      });
      expect(response.status).toBe(403);
      expect(response.body).toContain('GET and HEAD');
      expect(socketOpens).toBe(0);
      expect(stripCredentialHeaders({ Authorization: 'Bearer secret', Cookie: 'session=private', 'X-Trace': 'safe' }))
        .toEqual({ 'X-Trace': 'safe' });
    } finally { await proxy.close(); }
  });

  test('routes websocket attempts through the same destination checks', async () => {
    let socketOpens = 0;
    const proxy = await createPublicEgressProxy({
      lookup: async () => [{ address: '10.0.0.8', family: 4 }],
      connect: () => { socketOpens += 1; throw new Error('denied websocket opened a socket'); },
    });
    try {
      const response = await websocketProxyRequest(proxy.url, 'ws://private.test/socket');
      expect(response).toBe(403);
      expect(socketOpens).toBe(0);
    } finally { await proxy.close(); }
  });
});

function proxyRequest(proxyUrl: string, target: string, options: { method?: string; headers?: Record<string, string> } = {}): Promise<{ status: number; body: string }> {
  const proxy = new URL(proxyUrl);
  return new Promise((resolve, reject) => {
    const req = httpRequest({ hostname: proxy.hostname, port: proxy.port, method: options.method ?? 'GET', path: target, headers: options.headers }, (res) => {
      let body = '';
      res.setEncoding('utf8');
      res.on('data', (chunk: string) => { body += chunk; });
      res.on('end', () => resolve({ status: res.statusCode ?? 0, body }));
    });
    req.on('error', reject);
    req.end();
  });
}

function websocketProxyRequest(proxyUrl: string, target: string): Promise<number> {
  const proxy = new URL(proxyUrl);
  return new Promise((resolve, reject) => {
    const req = httpRequest({
      hostname: proxy.hostname, port: proxy.port, method: 'GET', path: target,
      headers: { connection: 'Upgrade', upgrade: 'websocket', 'sec-websocket-version': '13', 'sec-websocket-key': 'dGhlIHNhbXBsZSBub25jZQ==' },
    });
    req.on('upgrade', (res, socket) => { socket.destroy(); resolve(res.statusCode ?? 0); });
    req.on('response', (res) => { res.resume(); resolve(res.statusCode ?? 0); });
    req.on('error', reject);
    req.end();
  });
}

function connectRequest(proxyUrl: string, authority: string): Promise<{ status: number }> {
  const proxy = new URL(proxyUrl);
  return new Promise((resolve, reject) => {
    const req = httpRequest({ hostname: proxy.hostname, port: proxy.port, method: 'CONNECT', path: authority });
    req.on('connect', (res, socket) => { socket.destroy(); resolve({ status: res.statusCode ?? 0 }); });
    req.on('response', (res) => { res.resume(); resolve({ status: res.statusCode ?? 0 }); });
    req.on('error', reject);
    req.end();
  });
}

