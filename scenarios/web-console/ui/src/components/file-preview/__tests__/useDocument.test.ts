import { act, cleanup, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useDocument } from '../useDocument';
import { MAX_DOCUMENT_BYTES, type DocumentResult } from '../documentData';
class TestWorker {
  static instances: TestWorker[] = [];
  onmessage: ((event: MessageEvent<DocumentResult>) => void) | null = null;
  onerror: ((event: ErrorEvent) => void) | null = null;
  postMessage = vi.fn(); terminate = vi.fn();
  constructor() { TestWorker.instances.push(this); }
}
beforeEach(() => { TestWorker.instances = []; vi.stubGlobal('Worker', TestWorker); });
afterEach(() => { cleanup(); vi.unstubAllGlobals(); vi.useRealTimers(); });
function worker() { const value = TestWorker.instances.at(-1); if (!value) throw Error('Missing worker'); return value; }
describe('document worker lifecycle', () => {
  it('loads only through the local worker and destroys it on result', () => {
    const { result } = renderHook(() => useDocument('/blob', 12, 0)); const parser = worker(); expect(result.current).toBeNull(); expect(parser.postMessage).toHaveBeenCalledWith({ href: '/blob' });
    act(() => { parser.onmessage?.(new MessageEvent('message', { data: { error: 'encrypted' } })); }); expect(result.current).toEqual({ error: 'encrypted' }); expect(parser.terminate).toHaveBeenCalled();
  });
  it('ignores late results after source changes and terminates on close', () => {
    const { result, rerender, unmount } = renderHook(({ href }) => useDocument(href, 10, 0), { initialProps: { href: '/old' } }); const old = worker(); rerender({ href: '/new' }); expect(old.terminate).toHaveBeenCalled();
    act(() => { old.onmessage?.(new MessageEvent('message', { data: { error: 'invalid' } })); }); expect(result.current).toBeNull(); const current = worker(); unmount(); expect(current.terminate).toHaveBeenCalled();
  });
  it('rejects known oversized bytes without starting work and enforces parsing deadline', () => {
    vi.useFakeTimers(); const { result, rerender } = renderHook(({ size }) => useDocument('/blob', size, 0), { initialProps: { size: MAX_DOCUMENT_BYTES + 1 } }); expect(result.current).toEqual({ error: 'limit' }); expect(TestWorker.instances).toHaveLength(0);
    rerender({ size: 10 }); const current = worker(); act(() => { vi.advanceTimersByTime(10000); }); expect(result.current).toEqual({ error: 'limit' }); expect(current.terminate).toHaveBeenCalled();
  });
  it('reports unavailable/crashed workers and starts a fresh parser on retry', () => {
    const { result, rerender } = renderHook(({ attempt }) => useDocument('/blob', 10, attempt), { initialProps: { attempt: 0 } });
    const error = new ErrorEvent('error', { cancelable: true }); act(() => { worker().onerror?.(error); }); expect(error.defaultPrevented).toBe(true); expect(result.current).toEqual({ error: 'invalid' });
    rerender({ attempt: 1 }); expect(TestWorker.instances).toHaveLength(2); expect(result.current).toBeNull();
    vi.stubGlobal('Worker', undefined); const unavailable = renderHook(() => useDocument('/blob', 10, 0)); expect(unavailable.result.current).toEqual({ error: 'unavailable' });
  });
});
