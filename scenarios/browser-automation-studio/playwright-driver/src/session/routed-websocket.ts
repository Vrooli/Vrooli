import type { BrowserContext } from 'rebrowser-playwright';

/** Chromium does not copy extraHTTPHeaders into browser WebSocket handshakes.
 * Carry the same non-secret development routing marker through their URL.
 * The API still requires development mode and an exact live paired lease.
 */
export async function configureRoutedWebSockets(
  context: BrowserContext,
  headers?: Record<string, string>
): Promise<void> {
  if (!Object.entries(headers ?? {}).some(([name, value]) =>
    name.toLowerCase() === 'x-vrooli-test-mode' && value === '1')) return;
  await context.addInitScript(installRoutedWebSocketMarker);
}

export function installRoutedWebSocketMarker(): void {
  const NativeWebSocket = globalThis.WebSocket;
  globalThis.WebSocket = new Proxy(NativeWebSocket, {
    construct(target, args, newTarget) {
      if (args.length === 0) return Reflect.construct(target, args, newTarget);
      const url = new URL(args[0] as string, globalThis.document?.baseURI ?? globalThis.location.href);
      // The native constructor also accepts HTTP(S) and relative URLs and
      // normalizes their schemes. Use the document base, including <base>.
      if (['ws:', 'wss:', 'http:', 'https:'].includes(url.protocol)) {
        url.searchParams.set('__vrooli_test_mode', '1');
      }
      args[0] = url.href;
      return Reflect.construct(target, args, newTarget);
    },
  });
}
