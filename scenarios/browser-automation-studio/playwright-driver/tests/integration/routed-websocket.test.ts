import { createServer } from 'node:http';
import { once } from 'node:events';
import { chromium } from 'rebrowser-playwright';
import { configureRoutedWebSockets } from '../../src/session/routed-websocket';

describe('Chromium routed WebSocket handshake', () => {
  it('carries the development marker when Chromium omits extra HTTP headers', async () => {
    const handshakes: Array<{ url?: string; header?: string | string[] }> = [];
    const server = createServer((_request, response) => response.end('<html><head><base href="/base/"></head></html>'));
    server.on('upgrade', (request, socket) => {
      handshakes.push({ url: request.url, header: request.headers['x-vrooli-test-mode'] });
      socket.destroy();
    });
    server.listen(0, '127.0.0.1');
    await once(server, 'listening');
    const address = server.address();
    if (!address || typeof address === 'string') throw new Error('missing fixture address');
    const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
    try {
      for (const routed of [false, true]) {
        const headers = routed ? { 'X-Vrooli-Test-Mode': '1' } : {};
        const context = await browser.newContext({ extraHTTPHeaders: headers });
        try {
          await configureRoutedWebSockets(context, headers);
          const page = await context.newPage();
          await page.goto(`http://127.0.0.1:${address.port}`);
          const constructorErrors = await page.evaluate(() => {
            return [() => Reflect.construct(WebSocket, []), () => new WebSocket(Symbol('invalid') as unknown as string)].map((construct) => {
              try { construct(); return 'accepted'; } catch (error) { return (error as Error).name; }
            });
          });
          expect(constructorErrors).toEqual(['TypeError', 'TypeError']);
          for (const url of [`ws://127.0.0.1:${address.port}/ws?other=retained`, `http://127.0.0.1:${address.port}/http`, 'relative']) {
            await page.evaluate((value) => new Promise<void>((resolve) => {
              const socket = new WebSocket(value);
              socket.onerror = () => resolve();
            }), url);
          }
        } finally {
          await context.close();
        }
      }
      expect(handshakes).toEqual([
        { url: '/ws?other=retained', header: undefined },
        { url: '/http', header: undefined },
        { url: '/base/relative', header: undefined },
        { url: '/ws?other=retained&__vrooli_test_mode=1', header: undefined },
        { url: '/http?__vrooli_test_mode=1', header: undefined },
        { url: '/base/relative?__vrooli_test_mode=1', header: undefined },
      ]);
    } finally {
      await browser.close();
      server.close();
    }
  }, 30_000);
});
