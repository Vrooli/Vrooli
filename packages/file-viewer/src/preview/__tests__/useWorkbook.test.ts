import { act, cleanup } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { FileViewerLoaders, SpreadsheetEngine } from '../../loaders.js';
import { renderHookWithProviders } from '../../test-utils/index.js';
import { useWorkbook, WORKBOOK_DEADLINE_MS } from '../useWorkbook.js';
import { MAX_WORKBOOK_BYTES, type WorkbookResult } from '../workbookData.js';

afterEach(() => { cleanup(); vi.useRealTimers(); });

/** An engine whose parses the test settles by hand. */
function controlledEngine() {
  const calls: { href: string; signal: AbortSignal; settle: (value: WorkbookResult) => void; fail: (reason: unknown) => void }[] = [];
  const engine: SpreadsheetEngine = {
    parse: (href, signal) => new Promise<WorkbookResult>((resolve, reject) => { calls.push({ href, signal, settle: resolve, fail: reject }); }),
  };
  const loaders: FileViewerLoaders = { spreadsheet: () => Promise.resolve(engine) };
  const latest = () => { const call = calls.at(-1); if (!call) throw Error('Missing parse'); return call; };
  return { calls, loaders, latest };
}
const flush = async () => { await act(async () => { await Promise.resolve(); await Promise.resolve(); }); };

describe('workbook loading through the host engine', () => {
  it('parses the blob with the engine and reports its result', async () => {
    const { loaders, latest } = controlledEngine();
    const { result } = renderHookWithProviders(() => useWorkbook('/blob', 12, 0), { host: { loaders } });
    await flush(); expect(result.current).toBeNull(); expect(latest().href).toBe('/blob');
    await act(async () => { latest().settle({ error: 'encrypted' }); await Promise.resolve(); });
    expect(result.current).toEqual({ error: 'encrypted' });
  });

  it('reports unsupported, without loading anything, when the host supplied no engine', () => {
    const { result } = renderHookWithProviders(() => useWorkbook('/blob', 12, 0));
    expect(result.current).toEqual({ error: 'unsupported' });
  });

  it('aborts the previous parse and ignores its late result when the source changes or closes', async () => {
    const { loaders, calls, latest } = controlledEngine();
    const { result, rerender, unmount } = renderHookWithProviders(({ href }) => useWorkbook(href, 10, 0), { host: { loaders }, initialProps: { href: '/old' } });
    await flush(); const old = latest(); rerender({ href: '/new' }); expect(old.signal.aborted).toBe(true);
    await act(async () => { old.settle({ error: 'invalid' }); await Promise.resolve(); }); expect(result.current).toBeNull();
    await flush(); expect(calls).toHaveLength(2); const current = latest(); unmount(); expect(current.signal.aborted).toBe(true);
  });

  it('rejects known oversized bytes without starting work and enforces the parsing deadline', async () => {
    vi.useFakeTimers(); const { loaders, calls, latest } = controlledEngine();
    const { result, rerender } = renderHookWithProviders(({ size }) => useWorkbook('/blob', size, 0), { host: { loaders }, initialProps: { size: MAX_WORKBOOK_BYTES + 1 } });
    expect(result.current).toEqual({ error: 'limit' }); await flush(); expect(calls).toHaveLength(0);
    rerender({ size: 10 }); await flush(); const current = latest(); act(() => { vi.advanceTimersByTime(WORKBOOK_DEADLINE_MS); });
    expect(result.current).toEqual({ error: 'limit' }); expect(current.signal.aborted).toBe(true);
  });

  it('reports unavailable when the engine fails to load or throws, and parses afresh on retry', async () => {
    const failing: FileViewerLoaders = { spreadsheet: () => Promise.reject(new Error('xlsx is not installed')) };
    const { result, rerender } = renderHookWithProviders(({ attempt }) => useWorkbook('/blob', 10, attempt), { host: { loaders: failing }, initialProps: { attempt: 0 } });
    await flush(); expect(result.current).toEqual({ error: 'unavailable' });
    const { loaders, calls, latest } = controlledEngine(); const second = renderHookWithProviders(({ attempt }) => useWorkbook('/blob', 10, attempt), { host: { loaders }, initialProps: { attempt: 0 } });
    await flush(); await act(async () => { latest().fail(new Error('boom')); await Promise.resolve(); }); expect(second.result.current).toEqual({ error: 'unavailable' });
    second.rerender({ attempt: 1 }); await flush(); expect(calls).toHaveLength(2); expect(second.result.current).toBeNull();
    rerender({ attempt: 1 });
  });
});
