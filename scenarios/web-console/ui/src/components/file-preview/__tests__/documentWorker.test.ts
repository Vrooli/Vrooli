import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MAX_DOCUMENT_BYTES } from '../documentData';
import { parseDocument } from '../parseDocument';
vi.mock('../parseDocument', () => ({ parseDocument: vi.fn().mockReturnValue({ error: 'invalid' }) }));
const scope = { postMessage: vi.fn(), onmessage: null as ((event: MessageEvent<{ href: string }>) => Promise<void>) | null };
beforeEach(async () => { vi.clearAllMocks(); vi.resetModules(); vi.stubGlobal('self', scope); await import('../document.worker'); });
afterEach(() => { vi.unstubAllGlobals(); });
async function run() { if (!scope.onmessage) throw Error('Missing handler'); await scope.onmessage(new MessageEvent('message', { data: { href: '/blob' } })); }
function response(parts: Uint8Array[], length: string | null = null) {
  let index = 0; const reader = { read: vi.fn(() => { const value = parts[index++]; return Promise.resolve(value ? { value, done: false } : { value: undefined, done: true }); }), cancel: vi.fn(), releaseLock: vi.fn() };
  const body = { getReader: () => reader, cancel: vi.fn() };
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, body, headers: { get: () => length } })); return { reader, body };
}
describe('bounded document transport', () => {
  it('joins streamed bytes exactly and releases the reader before parsing', async () => {
    const { reader } = response([new Uint8Array([1, 2]), new Uint8Array([3])]); await run();
    expect(fetch).toHaveBeenCalledWith('/blob', { credentials: 'same-origin' }); expect(reader.releaseLock).toHaveBeenCalled(); expect(parseDocument).toHaveBeenCalledWith(new Uint8Array([1, 2, 3]).buffer); expect(scope.postMessage).toHaveBeenCalledWith({ error: 'invalid' });
  });
  it('rejects an oversized declared body before reading any bytes', async () => {
    const { body, reader } = response([], String(MAX_DOCUMENT_BYTES + 1)); await run(); expect(body.cancel).toHaveBeenCalled(); expect(reader.read).not.toHaveBeenCalled(); expect(parseDocument).not.toHaveBeenCalled(); expect(scope.postMessage).toHaveBeenCalledWith({ error: 'limit' });
  });
  it('cancels the stream if actual bytes exceed the limit without a truthful length', async () => {
    const { reader } = response([new Uint8Array(MAX_DOCUMENT_BYTES), new Uint8Array([1])]); await run(); expect(reader.cancel).toHaveBeenCalled(); expect(reader.releaseLock).toHaveBeenCalled(); expect(parseDocument).not.toHaveBeenCalled(); expect(scope.postMessage).toHaveBeenCalledWith({ error: 'limit' });
  });
  it('returns a fetch recovery state for rejected requests and missing streams', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('expired'))); await run(); expect(scope.postMessage).toHaveBeenLastCalledWith({ error: 'fetch' });
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false })); await run(); expect(scope.postMessage).toHaveBeenLastCalledWith({ error: 'fetch' });
  });
});
