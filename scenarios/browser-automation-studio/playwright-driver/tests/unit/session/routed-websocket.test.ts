import { configureRoutedWebSockets, installRoutedWebSocketMarker } from '../../../src/session/routed-websocket';
import type { BrowserContext } from 'rebrowser-playwright';

describe('routed browser WebSockets', () => {
  it('installs only for an explicit exact test routing header', async () => {
    const addInitScript = jest.fn();
    const context = { addInitScript } as unknown as BrowserContext;
    for (const headers of [undefined, {}, { 'X-Vrooli-Test-Mode': 'true' }]) {
      await configureRoutedWebSockets(context, headers);
    }
    expect(addInitScript).not.toHaveBeenCalled();
    await configureRoutedWebSockets(context, { 'x-vrooli-test-mode': '1' });
    expect(addInitScript).toHaveBeenCalledWith(installRoutedWebSocketMarker);
  });

  it('preserves constructor protocols, constants, instance identity and existing query', () => {
    const originalSocket = globalThis.WebSocket;
    const originalLocation = Object.getOwnPropertyDescriptor(globalThis, 'location');
    class Socket {
      static OPEN = 1;
      constructor(readonly url: string, readonly protocols?: string[]) {}
    }
    try {
      globalThis.WebSocket = Socket as unknown as typeof WebSocket;
      Object.defineProperty(globalThis, 'location', { configurable: true, value: { href: 'https://fixture.test/app/' } });
      installRoutedWebSocketMarker();
      const socket = new WebSocket('wss://fixture.test/ws?other=retained', ['application']);
      expect(socket.url).toBe('wss://fixture.test/ws?other=retained&__vrooli_test_mode=1');
      expect((socket as unknown as Socket).protocols).toEqual(['application']);
      expect(socket).toBeInstanceOf(Socket);
      expect(socket).toBeInstanceOf(WebSocket);
      expect(WebSocket.OPEN).toBe(1);
      expect(new WebSocket('../ws').url).toBe('https://fixture.test/ws?__vrooli_test_mode=1');
      expect(new WebSocket('https://fixture.test/ws').url).toBe('https://fixture.test/ws?__vrooli_test_mode=1');
    } finally {
      globalThis.WebSocket = originalSocket;
      if (originalLocation) Object.defineProperty(globalThis, 'location', originalLocation);
      else Reflect.deleteProperty(globalThis, 'location');
    }
  });
});
